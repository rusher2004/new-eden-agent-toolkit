package mcp

import (
	"context"
	"encoding/json"
	"fmt"

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
		out, err := op.Run(ctx, in)
		if err != nil {
			// As the SDK's typed handler does: a structured JSON-RPC error
			// passes through as one, anything else is a tool error.
			if wireErr, ok := err.(*jsonrpc.Error); ok {
				return nil, wireErr
			}
			var res mcpsdk.CallToolResult
			res.SetError(err)
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
