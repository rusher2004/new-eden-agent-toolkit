package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	urfave "github.com/urfave/cli/v3"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/cli"
	"github.com/rusher2004/new-eden-agent-toolkit/commands"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

func TestCLIPathsCoverOperations(t *testing.T) {
	seen := map[string]string{}
	for _, op := range commands.Catalog() {
		if strings.TrimSpace(op.Description) == "" {
			t.Errorf("%s has no description", op.Name)
		}
		path, ok := cli.Paths[op.Name]
		if !ok {
			t.Errorf("%s has no entry in cli.Paths", op.Name)
			continue
		}
		if len(path) < 2 {
			t.Errorf("%s: path %q needs a group and a leaf", op.Name, path)
			continue
		}
		key := strings.Join(path, " ")
		if other, dup := seen[key]; dup {
			t.Errorf("%s and %s share the path %q", other, op.Name, key)
		}
		seen[key] = op.Name
		for i := 1; i < len(path); i++ {
			group := strings.Join(path[:i], " ")
			if strings.TrimSpace(cli.Groups[group]) == "" {
				t.Errorf("%s: group %q has no help in cli.Groups", op.Name, group)
			}
		}
	}
}

type parityIn struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character"`
}

type parityItem struct {
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
}

type parityOut struct {
	CharacterID int32        `json:"characterId"`
	Name        string       `json:"name"`
	Items       []parityItem `json:"items"`
	Missing     *int64       `json:"missing,omitempty"`
}

func parityOp() operation.Operation {
	return operation.New("parity_demo", "A demo for adapter parity.", operation.Read,
		func(_ context.Context, in parityIn) (parityOut, error) {
			return parityOut{CharacterID: in.CharacterID, Name: "Pilot <&> \"quoted\"",
				Items: []parityItem{{TypeID: 34, TypeName: "Tritanium"}, {TypeID: 35}}}, nil
		})
}

func TestCLIAndMCPReturnTheSameJSON(t *testing.T) {
	cli.Paths["parity_demo"] = []string{"parity", "demo"}
	cli.Groups["parity"] = "parity demo"
	t.Cleanup(func() {
		delete(cli.Paths, "parity_demo")
		delete(cli.Groups, "parity")
	})
	ops := []operation.Operation{parityOp()}

	// Through the CLI.
	cmds, err := cli.Build(ops, func(context.Context) ([]operation.Operation, func() error, error) {
		return ops, func() error { return nil }, nil
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var stdout, stderr bytes.Buffer
	root := &urfave.Command{Name: "neat", Commands: cmds, Writer: &stdout, ErrWriter: &stderr,
		ExitErrHandler: func(context.Context, *urfave.Command, error) {}}
	if err := root.Run(context.Background(), []string{"neat", "parity", "demo", "--character", "7"}); err != nil {
		t.Fatalf("cli run: %v (stderr %q)", err, stderr.String())
	}
	cliJSON := strings.TrimSuffix(stdout.String(), "\n")

	// Through MCP.
	ctx := context.Background()
	s := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: "test"}, nil)
	mcp.Register(s, ops)
	st, ct := mcpsdk.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() {
		if err := cs.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
		if err := ss.Wait(); err != nil {
			t.Errorf("wait server: %v", err)
		}
	})
	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "parity_demo", Arguments: map[string]any{"characterId": 7}})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError || len(res.Content) != 1 {
		t.Fatalf("CallTool result = %+v", res)
	}
	text, ok := res.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("content is %T, want *TextContent", res.Content[0])
	}
	if cliJSON != text.Text {
		t.Errorf("CLI JSON %s differs from MCP JSON %s", cliJSON, text.Text)
	}
	// The structured content is the same value.
	structured, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if string(structured) != cliJSON {
		t.Errorf("CLI JSON %s differs from MCP structured content %s", cliJSON, structured)
	}
}

// runNeat runs the real root command with captured writers and a no-op exit
// handler, returning stdout, stderr and the exit code the error carries.
func runNeat(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	root := commands.Root("dev", "test-client-id", "", false)
	root.Writer, root.ErrWriter, root.Reader = &out, &errOut, strings.NewReader("")
	err := root.Run(context.Background(), append([]string{"neat"}, args...))
	if err != nil {
		var ec urfave.ExitCoder
		if !errors.As(err, &ec) {
			t.Fatalf("Run(%v) = %v, which carries no exit code", args, err)
		}
		code = ec.ExitCode()
	}
	return out.String(), errOut.String(), code
}

func TestCLI_UnknownCharacterNoESI(t *testing.T) {
	keyring.MockInit()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	stdout, stderr, code := runNeat(t, "loyalty", "points", "--character", "999")
	if code != 1 {
		t.Errorf("exit %d, want 1 (stderr %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &e); err != nil {
		t.Fatalf("stderr %q is not one {\"error\"} line: %v", stderr, err)
	}
	// This exact message comes from the logged-in check, which returns before
	// the ESI client is opened; opening one would fail with a message naming
	// the keychain token instead ("list character loyalty points: character
	// 999: ...").
	if want := "list character loyalty points for character 999: character not found"; e.Error != want {
		t.Errorf("error = %q, want %q", e.Error, want)
	}
}

func TestCLI_UsageErrorIsExit2(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	_, stderr, code := runNeat(t, "loyalty", "store-offers", "--corporation", "abc")
	if code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", code, stderr)
	}
	if !strings.HasPrefix(stderr, `{"error":`) {
		t.Errorf("stderr = %q, want a JSON error", stderr)
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatalf("read HOME: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("a usage error created %d entries under HOME, want none", len(entries))
	}
}

func TestCLI_HelpAndVersionTouchNothing(t *testing.T) {
	// HOME is a regular file, so anything that tries to create ~/.neat fails.
	home := filepath.Join(t.TempDir(), "home")
	if err := os.WriteFile(home, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write HOME file: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	for _, args := range [][]string{{"--help"}, {"loyalty", "--help"}, {"loyalty", "points", "--help"}, {"version"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, code := runNeat(t, args...)
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, stderr)
			}
			if stdout == "" {
				t.Error("printed nothing")
			}
		})
	}
	stdout, _, _ := runNeat(t, "--help")
	if !strings.Contains(stdout, "loyalty") {
		t.Errorf("neat --help does not list the loyalty group:\n%s", stdout)
	}
	if strings.Contains(stdout, "store-offers") {
		t.Errorf("neat --help lists leaf commands; it should list groups only:\n%s", stdout)
	}
	stdout, _, _ = runNeat(t, "loyalty", "--help")
	for _, want := range []string{"points", "store-offers"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("neat loyalty --help lacks %q:\n%s", want, stdout)
		}
	}
}
