package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditFindsConcreteClientParam(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, f.String())
	}
	out := strings.Join(got, "\n")
	for _, want := range []string{
		"CONCRETE_PARAM commands/c.go:9 holder.H sde.Client",
		"CONCRETE_PARAM commands/c.go:12 hook sde.Client",
		"CONCRETE_PARAM commands/c.go:15 Trigger sde.Client",
		"CONCRETE_PARAM commands/c.go:22 factory sde.Client",
		"CONCRETE_CTOR commands/c.go:22 commands sde.New",
		"PRODUCER_IFACE sde/l.go:7 sde.Lister used-by=commands",
		"INTERFACE commands/c.go:18 commands.local used-by=-",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "skipped") {
		t.Errorf("test files must be skipped:\n%s", out)
	}
	if strings.Count(out, "CONCRETE_PARAM") != 4 {
		t.Errorf("want exactly 4 CONCRETE_PARAM:\n%s", out)
	}
}

func TestAuditDeterministicAndSorted(t *testing.T) {
	render := func() []Finding {
		fs, err := Audit("testdata")
		if err != nil {
			t.Fatal(err)
		}
		return fs
	}
	a, b := render(), render()
	if len(a) != len(b) || len(a) == 0 {
		t.Fatalf("runs differ in length: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].String() != b[i].String() {
			t.Fatalf("run mismatch at %d: %q vs %q", i, a[i], b[i])
		}
		if i > 0 && (a[i-1].File > a[i].File || a[i-1].File == a[i].File && a[i-1].Line > a[i].Line) {
			t.Errorf("not sorted by file then line: %q before %q", a[i-1], a[i])
		}
	}
}

func TestAuditResolvesImportAlias(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		// c.go declares `var hook func(c *hz.Client)` with hz aliasing sde.
		if f.Kind == "CONCRETE_PARAM" && f.Context == "hook" && f.Type == "sde.Client" {
			return
		}
	}
	t.Error("aliased import hz.Client was not resolved to sde.Client")
}

func TestAuditFlagsTypeOnlyFiles(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		if f.Kind == "TYPE_ONLY_FILE" {
			got = append(got, f.String())
		}
	}
	want := []string{
		"TYPE_ONLY_FILE commands/types_only.go:6 commands",
		"TYPE_ONLY_FILE sde/l.go:4 sde",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("TYPE_ONLY_FILE findings:\n got %q\nwant %q", got, want)
	}
}

// TestAuditFlagsIfaceFileByName covers the name rule: a *_iface.go file is
// flagged even when it also holds code. The fixture is built in a temp dir so no
// *_iface.go file is ever tracked in the repository.
func TestAuditFlagsIfaceFileByName(t *testing.T) {
	root := t.TempDir()
	src := "package commands\n\ntype namedIface interface{ Do() }\n\nfunc useNamed(n namedIface) { n.Do() }\n"
	if err := os.WriteFile(filepath.Join(root, "named_iface.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := Audit(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.String() == "TYPE_ONLY_FILE named_iface.go:3 commands" {
			return
		}
	}
	t.Errorf("named_iface.go was not flagged: %v", fs)
}

func TestAuditUnparsableFileIsError(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "bad.go")
	if err := os.WriteFile(bad, []byte("package x\nfunc {"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Audit(root)
	if err == nil || !strings.Contains(err.Error(), "bad.go") {
		t.Fatalf("err = %v, want a parse error naming bad.go", err)
	}
}

var errWrite = errors.New("synthetic write failure")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestPrintFindingsWriteFailureIsError(t *testing.T) {
	err := printFindings(failingWriter{}, []Finding{{Kind: "INTERFACE", File: "x.go", Line: 1, Context: "x.I"}})
	if !errors.Is(err, errWrite) {
		t.Fatalf("err = %v, want the write error", err)
	}
}

func TestAuditIgnoresPlainDataStructs(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.Kind == "CONCRETE_PARAM" && f.Type == "sde.Record" {
			t.Errorf("plain data struct reported as a concrete client: %s", f)
		}
	}
}

func TestAuditSkipsHiddenDirectories(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if strings.Contains(f.File, ".hidden") {
			t.Errorf("hidden directory was scanned: %s", f)
		}
	}
}

func TestAuditKeepsSamePackageNameDirectoriesApart(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, f := range fs {
		if f.Kind == "INTERFACE" && strings.HasPrefix(f.File, "dup/") {
			got[f.File] = f.UsedBy
		}
	}
	if len(got) != 2 {
		t.Fatalf("want one INTERFACE per dup directory, got %v", got)
	}
	if u := got["dup/a/a.go"]; len(u) != 1 || u[0] != "-" {
		t.Errorf("dup/a Runner used-by = %v, want [-]: dup/b's use must not leak in", u)
	}
	if u := got["dup/b/b.go"]; len(u) != 1 || u[0] != "dup" {
		t.Errorf("dup/b Runner used-by = %v, want [dup]", u)
	}
}
