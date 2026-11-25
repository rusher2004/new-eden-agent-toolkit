package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func audit(t *testing.T, fixture string) []string {
	t.Helper()
	fs, err := Audit(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, f.String())
	}
	return got
}

func TestAuditFlagsDomainImportingMCP(t *testing.T) {
	got := audit(t, "domain_mcp")
	want := "DOMAIN_IMPORTS_ADAPTER character/c.go example.com/fx/mcp"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want [%q]", got, want)
	}
}

func TestAuditFlagsDomainImportingSDK(t *testing.T) {
	got := audit(t, "domain_sdk")
	want := "DOMAIN_IMPORTS_ADAPTER universe/u.go github.com/modelcontextprotocol/go-sdk/mcp"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want [%q]", got, want)
	}
}

func TestAuditFlagsDomainImportingHTTP(t *testing.T) {
	got := audit(t, "domain_http")
	want := "DOMAIN_IMPORTS_ADAPTER loyalty/l.go net/http"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want [%q]", got, want)
	}
}

func TestAuditFlagsCLIImportingDomain(t *testing.T) {
	got := audit(t, "cli_domain")
	want := "CLI_IMPORTS_DOMAIN cli/c.go example.com/fx/character"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want [%q] (operation, stdlib and third-party are allowed)", got, want)
	}
}

func TestAuditAllowsSourceUsingHTTP(t *testing.T) {
	got := audit(t, "source_http")
	// esi uses net/http and the MCP SDK freely; sde importing commands is not.
	want := "DOMAIN_IMPORTS_ADAPTER sde/s.go example.com/fx/commands"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want only [%q]", got, want)
	}
}

func TestAuditSkipsTestFilesAndHiddenDirs(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/fx\n")
	write("character/c_test.go", "package character\nimport _ \"net/http\"\n")
	write(".claude/worktrees/x/character/c.go", "package character\nimport _ \"net/http\"\n")
	write("tools/t/t.go", "package main\nimport _ \"net/http\"\n")
	fs, err := Audit(root)
	if err != nil || len(fs) != 0 {
		t.Fatalf("got %v, %v; want no findings", fs, err)
	}
}

func TestAuditUnparsableFileIsError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/fx\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "character"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "character", "bad.go"), []byte("package x\nfunc {"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Audit(root)
	if err == nil || !strings.Contains(err.Error(), "bad.go") {
		t.Fatalf("err = %v, want a parse error naming bad.go", err)
	}
}

func TestAuditMissingGoModIsError(t *testing.T) {
	if _, err := Audit(t.TempDir()); err == nil {
		t.Fatal("want an error without go.mod")
	}
}

var errWrite = errors.New("synthetic write failure")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestPrintFindingsWriteFailureIsError(t *testing.T) {
	err := printFindings(failingWriter{}, []Finding{{Kind: "CLI_IMPORTS_DOMAIN", File: "x.go", Import: "y"}})
	if !errors.Is(err, errWrite) {
		t.Fatalf("err = %v, want the write error", err)
	}
}

// TestRepoRespectsBoundaries is the repo-conformance gate: any finding in the
// repository fails it.
func TestRepoRespectsBoundaries(t *testing.T) {
	fs, err := Audit("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		t.Errorf("%s (domain, pure and operation packages must not import adapters, the MCP SDK, urfave/cli or net/http; cli may import only operation)", f)
	}
}
