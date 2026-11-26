package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

var update = flag.Bool("update", false, "rewrite testdata/tools.golden.json")

const goldenPath = "testdata/tools.golden.json"

type goldenTool struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Annotations  any    `json:"annotations"`
	InputSchema  any    `json:"inputSchema"`
	OutputSchema any    `json:"outputSchema"`
}

// listTools lists every tool of the server over an in-memory session, sorted
// by name, as the JSON an MCP client sees in tools/list.
func listTools(ctx context.Context, t *testing.T) (out []byte, err error) {
	server := commands.NewMCPServer("test", "client-id", config.NewStoreAt(filepath.Join(t.TempDir(), "config.json")), nil)
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect server: %w", err)
	}
	defer func() {
		if cerr := serverSession.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close server session: %w", cerr)
		}
	}()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "golden", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect client: %w", err)
	}
	defer func() {
		if cerr := session.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close client session: %w", cerr)
		}
	}()

	var tools []goldenTool
	for tool, terr := range session.Tools(ctx, nil) {
		if terr != nil {
			return nil, fmt.Errorf("list tools: %w", terr)
		}
		tools = append(tools, goldenTool{
			Name:         tool.Name,
			Description:  tool.Description,
			Annotations:  tool.Annotations,
			InputSchema:  tool.InputSchema,
			OutputSchema: tool.OutputSchema,
		})
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(tools); err != nil {
		return nil, fmt.Errorf("marshal tools: %w", err)
	}
	return buf.Bytes(), nil
}

func TestMCPToolListMatchesGolden(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := listTools(context.Background(), t)
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("create testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (regenerate with -update): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("tool list differs from %s; review the change and regenerate with: go test ./commands -run TestMCPToolListMatchesGolden -update", goldenPath)
	}
}
