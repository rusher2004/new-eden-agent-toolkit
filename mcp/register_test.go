package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

type demoIn struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character to look up"`
}

type demoItem struct {
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
}

type demoOut struct {
	CharacterID int32      `json:"characterId"`
	Name        string     `json:"name"`
	Items       []demoItem `json:"items"`
}

const demoName, demoDescription = "demo_lookup", "Look up a demo character."

var errDemo = errors.New("demo failed")

// demoRun is the typed function both registrations wrap: Register through an
// Operation, and the SDK's typed AddTool directly, so their wire output can be
// compared.
func demoRun(_ context.Context, in demoIn) (demoOut, error) {
	if in.CharacterID == 13 {
		return demoOut{}, errDemo
	}
	return demoOut{CharacterID: in.CharacterID, Name: "Pilot", Items: []demoItem{{TypeID: 34, TypeName: "Tritanium"}, {TypeID: 35}}}, nil
}

// session serves s over an in-memory transport and returns a client session.
func session(t *testing.T, s *mcpsdk.Server) *mcpsdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcpsdk.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() {
		if err := cs.Close(); err != nil {
			t.Errorf("close client session: %v", err)
		}
		if err := ss.Wait(); err != nil {
			t.Errorf("wait server session: %v", err)
		}
	})
	return cs
}

// registered serves demoRun through Register; typed serves it through the
// SDK's typed AddTool.
func registered(t *testing.T) *mcpsdk.ClientSession {
	t.Helper()
	s := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: "test"}, nil)
	mcp.Register(s, []operation.Operation{operation.New(demoName, demoDescription, operation.Read, demoRun)})
	return session(t, s)
}

func typed(t *testing.T) *mcpsdk.ClientSession {
	t.Helper()
	s := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: "test"}, nil)
	mcpsdk.AddTool(s, &mcpsdk.Tool{Name: demoName, Description: demoDescription},
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in demoIn) (*mcpsdk.CallToolResult, demoOut, error) {
			out, err := demoRun(ctx, in)
			return nil, out, err
		})
	return session(t, s)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T: %v", v, err)
	}
	return string(b)
}

func call(t *testing.T, cs *mcpsdk.ClientSession, args any) (*mcpsdk.CallToolResult, error) {
	t.Helper()
	return cs.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: demoName, Arguments: args})
}

func TestRegister_ListsToolWithSchemas(t *testing.T) {
	ctx := context.Background()
	got, err := registered(t).ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	want, err := typed(t).ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools (typed): %v", err)
	}
	if len(got.Tools) != 1 {
		t.Fatalf("listed %d tools, want 1", len(got.Tools))
	}
	tool := got.Tools[0]
	if tool.Name != demoName || tool.Description != demoDescription {
		t.Errorf("tool = %q %q, want %q %q", tool.Name, tool.Description, demoName, demoDescription)
	}
	if tool.InputSchema == nil || tool.OutputSchema == nil {
		t.Fatalf("schemas missing: input %v, output %v", tool.InputSchema, tool.OutputSchema)
	}
	if !strings.Contains(mustJSON(t, tool.InputSchema), "the character to look up") {
		t.Errorf("input schema %s lacks the field description", mustJSON(t, tool.InputSchema))
	}
	if g, w := mustJSON(t, got.Tools), mustJSON(t, want.Tools); g != w {
		t.Errorf("tools/list differs from the SDK's typed AddTool:\n got %s\nwant %s", g, w)
	}
}

func TestRegister_CallReturnsStructuredAndTextContent(t *testing.T) {
	args := map[string]any{"characterId": 7}
	got, err := call(t, registered(t), args)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if got.IsError {
		t.Fatalf("IsError set: %s", mustJSON(t, got.Content))
	}
	wantOut, err := demoRun(context.Background(), demoIn{CharacterID: 7})
	if err != nil {
		t.Fatalf("demoRun: %v", err)
	}
	var structured demoOut
	if err := json.Unmarshal([]byte(mustJSON(t, got.StructuredContent)), &structured); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	if mustJSON(t, structured) != mustJSON(t, wantOut) {
		t.Errorf("structured content = %s, want %s", mustJSON(t, structured), mustJSON(t, wantOut))
	}
	if len(got.Content) != 1 {
		t.Fatalf("content has %d blocks, want 1", len(got.Content))
	}
	text, ok := got.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("content[0] is %T, want *TextContent", got.Content[0])
	}
	if text.Text != mustJSON(t, got.StructuredContent) {
		t.Errorf("text content = %s, want the structured content's JSON %s", text.Text, mustJSON(t, got.StructuredContent))
	}

	want, err := call(t, typed(t), args)
	if err != nil {
		t.Fatalf("CallTool (typed): %v", err)
	}
	if g, w := mustJSON(t, got), mustJSON(t, want); g != w {
		t.Errorf("result differs from the SDK's typed AddTool:\n got %s\nwant %s", g, w)
	}
}

func TestRegister_RunErrorIsToolError(t *testing.T) {
	got, err := call(t, registered(t), map[string]any{"characterId": 13})
	if err != nil {
		t.Fatalf("CallTool returned a protocol error, want a tool error: %v", err)
	}
	if !got.IsError {
		t.Fatalf("IsError not set: %s", mustJSON(t, got))
	}
	if len(got.Content) != 1 {
		t.Fatalf("content has %d blocks, want 1", len(got.Content))
	}
	text, ok := got.Content[0].(*mcpsdk.TextContent)
	if !ok || text.Text != errDemo.Error() {
		t.Errorf("content = %s, want one text block %q", mustJSON(t, got.Content), errDemo.Error())
	}
	want, err := call(t, typed(t), map[string]any{"characterId": 13})
	if err != nil {
		t.Fatalf("CallTool (typed): %v", err)
	}
	if g, w := mustJSON(t, got), mustJSON(t, want); g != w {
		t.Errorf("result differs from the SDK's typed AddTool:\n got %s\nwant %s", g, w)
	}
}

// TestRegister_InvalidArgumentsFail checks that bad arguments fail the call
// the way the SDK's typed handler fails it. go-sdk v1.8.0's typed handler
// reports invalid arguments as an IsError tool result (toolForErr's
// "validating \"arguments\"" branch), not a JSON-RPC error, so Register does
// the same and never runs the operation.
func TestRegister_InvalidArgumentsFail(t *testing.T) {
	cases := map[string]any{
		"missing required": map[string]any{},
		"wrong type":       map[string]any{"characterId": "seven"},
		"unknown field":    map[string]any{"characterId": 7, "extra": true},
	}
	reg, typ := registered(t), typed(t)
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := call(t, reg, args)
			want, werr := call(t, typ, args)
			if (err != nil) != (werr != nil) {
				t.Fatalf("protocol error = %v, typed AddTool's = %v; want the same kind of failure", err, werr)
			}
			if err != nil {
				return
			}
			if !got.IsError || !want.IsError {
				t.Fatalf("IsError = %v (typed %v), want both true: %s", got.IsError, want.IsError, mustJSON(t, got))
			}
			if got.StructuredContent != nil {
				t.Errorf("structured content %s on a failed call", mustJSON(t, got.StructuredContent))
			}
			text, ok := got.Content[0].(*mcpsdk.TextContent)
			if !ok || !strings.Contains(text.Text, "invalid input") {
				t.Errorf("content = %s, want text naming the invalid input", mustJSON(t, got.Content))
			}
		})
	}
}

func TestRegister_OutputFailingSchemaIsProtocolError(t *testing.T) {
	// Run returning something other than Out can only happen through a
	// hand-built Operation; the SDK's typed handler returns a Go error
	// ("validating tool output") for output its schema rejects, which reaches
	// the client as a JSON-RPC error.
	op := operation.New(demoName, demoDescription, operation.Read, demoRun)
	op.Run = func(context.Context, any) (any, error) { return map[string]any{"name": 1}, nil }
	s := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: "test"}, nil)
	mcp.Register(s, []operation.Operation{op})
	if _, err := call(t, session(t, s), map[string]any{"characterId": 7}); err == nil {
		t.Fatal("CallTool with output failing the schema returned no error")
	}
}
