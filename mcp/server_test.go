package mcp_test

import (
	"context"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

func connectedSession(t *testing.T, store *config.Store) (*mcpsdk.ClientSession, *mcpsdk.ServerSession) {
	t.Helper()
	server := mcp.NewServer("test", "client-id", store)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() {
		clientSession.Close()
		serverSession.Wait()
	})
	return clientSession, serverSession
}

func TestServer_AllToolsRegistered(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	clientSession, _ := connectedSession(t, store)

	res, err := clientSession.ListTools(context.Background(), &mcpsdk.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"add_character", "list_characters", "remove_character", "get_wallet_balance"} {
		if !names[want] {
			t.Errorf("missing tool %q, got %v", want, names)
		}
	}
}

func TestServer_RemoveCharacterNotFound_SurfacesAsToolError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	clientSession, _ := connectedSession(t, store)

	res, err := clientSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "remove_character",
		Arguments: map[string]any{"characterId": 12345},
	})
	if err != nil {
		t.Fatalf("CallTool transport-level error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true for missing character, got %+v", res)
	}

	// GetError() always returns nil client-side (the SDK does not
	// serialize the underlying Go error across the wire) — the only
	// client-visible signal is IsError plus the human-readable text
	// content, so that's what this test asserts against.
	var text string
	for _, c := range res.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			text = tc.Text
		}
	}
	if !strings.Contains(text, "character not found") {
		t.Errorf("content text = %q, want it to contain %q", text, "character not found")
	}
}
