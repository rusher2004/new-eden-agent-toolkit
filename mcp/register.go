package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

// Register adds each operation to s as a tool of the same name and
// description, with the operation's input and output schemas and no
// annotations.
//
// A tool registered here behaves on the wire as one registered with the
// SDK's typed mcpsdk.AddTool over the same function (go-sdk v1.8.0's
// toolForErr): arguments that fail Decode are an IsError result and Run is
// not called; an error from Run is an IsError result holding its message,
// except a *jsonrpc.Error, which is returned as the JSON-RPC error itself;
// success sets StructuredContent to the output's JSON and one TextContent
// holding the same JSON; output the schema rejects is a JSON-RPC error.
//
// Progress an operation reports (operation.Progress) is forwarded to the
// client: as a notifications/progress carrying the request's progress token
// when the call has one, otherwise as an info-level notifications/message
// (which the SDK sends only when the client asked for a logging level, on the
// call or for the session). A notification that fails to send does not stop
// the operation; later ones are skipped, and the first failure is joined into
// the tool error when the operation fails. When the operation succeeds its result is returned as is:
// a lost progress message is not worth failing a call whose work is done.
func Register(s *mcpsdk.Server, ops []operation.Operation) {
	for _, op := range ops {
		tool := &mcpsdk.Tool{
			Name:        op.Name,
			Description: op.Description,
			// Cloned: the operation's schemas are shared by every copy of it,
			// and the server owns what it is given.
			InputSchema: op.InputSchema().CloneSchemas(),
		}
		// A nil *jsonschema.Schema inside the interface would make AddTool
		// panic; leave the field nil when the operation has no output schema.
		if out := op.OutputSchema(); out != nil {
			tool.OutputSchema = out.CloneSchemas()
		}
		s.AddTool(tool, handler(op))
	}
}

func handler(op operation.Operation) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var raw json.RawMessage
		if req.Params.Arguments != nil {
			raw = req.Params.Arguments
		}
		in, err := op.Decode(raw)
		if err != nil {
			var res mcpsdk.CallToolResult
			res.SetError(err)
			return &res, nil
		}
		progress := &clientProgress{ctx: ctx, op: op.Name, session: req.Session, token: req.Params.GetProgressToken()}
		out, err := op.Run(operation.WithProgress(ctx, progress.send), in)
		if err != nil {
			// As the SDK's typed handler does: a structured JSON-RPC error
			// passes through as one, anything else is a tool error. Joining a
			// progress failure would hide the *jsonrpc.Error's type, so only
			// a tool error carries it.
			if wireErr, ok := err.(*jsonrpc.Error); ok {
				return nil, wireErr
			}
			var res mcpsdk.CallToolResult
			res.SetError(errors.Join(err, progress.Err()))
			return &res, nil
		}
		data, err := op.EncodeOutput(out)
		if err != nil {
			return nil, fmt.Errorf("validating tool output: %w", err)
		}
		res := &mcpsdk.CallToolResult{}
		if data != nil {
			res.StructuredContent = json.RawMessage(data)
			res.Content = []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}}
		}
		return res, nil
	}
}

// clientProgress forwards one call's progress messages to the client. An
// operation may report progress from several goroutines, so sends are
// serialized; the sink can't return an error, so the first failed send is
// kept and later messages are skipped.
type clientProgress struct {
	ctx     context.Context
	op      string
	session *mcpsdk.ServerSession
	token   any

	mu   sync.Mutex
	sent float64
	err  error
}

func (p *clientProgress) send(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return
	}
	if p.session == nil {
		p.err = fmt.Errorf("%s: send progress %q: the call has no session", p.op, msg)
		return
	}
	var err error
	if p.token != nil {
		p.sent++
		err = p.session.NotifyProgress(p.ctx, &mcpsdk.ProgressNotificationParams{ProgressToken: p.token, Message: msg, Progress: p.sent})
	} else {
		// The logging notification is deprecated in the 2026-07-28 protocol
		// but still served; it is the only way to reach a client that sent no
		// progress token.
		err = p.session.Log(p.ctx, &mcpsdk.LoggingMessageParams{Level: "info", Logger: p.op, Data: msg})
	}
	if err != nil {
		p.err = fmt.Errorf("%s: send progress %q: %w", p.op, msg, err)
	}
}

// Err is the first failed send, or nil.
func (p *clientProgress) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}
