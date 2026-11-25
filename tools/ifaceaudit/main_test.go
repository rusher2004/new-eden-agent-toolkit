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
		"CONCRETE_PARAM consumer/c.go:9 holder.H sde.Client",
		"CONCRETE_PARAM consumer/c.go:12 hook sde.Client",
		"CONCRETE_PARAM consumer/c.go:15 Trigger sde.Client",
		"CONCRETE_PARAM consumer/c.go:22 factory sde.Client",
		"CONCRETE_CTOR consumer/c.go:22 consumer sde.New",
		"PRODUCER_IFACE sde/l.go:7 sde.Lister used-by=consumer",
		"INTERFACE consumer/c.go:18 consumer.local used-by=-",
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
		"TYPE_ONLY_FILE consumer/types_only.go:6 consumer",
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

func TestAuditFlagsInterfaceDeclaredAwayFromItsUsers(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]Finding{}
	for _, f := range fs {
		if strings.HasPrefix(f.File, "away/") {
			kinds[f.Context] = f
		}
	}
	// stranger is declared in decl.go but only use.go refers to it.
	if f := kinds["away.stranger"]; f.Kind != "IFACE_AWAY" || len(f.UsedBy) != 1 || f.UsedBy[0] != "away/use.go" {
		t.Errorf("away.stranger = %+v, want IFACE_AWAY used-by away/use.go", f)
	}
	// Doer is used in its own file and reused in another: a legitimate shared interface.
	if f := kinds["away.Doer"]; f.Kind != "INTERFACE" {
		t.Errorf("away.Doer kind = %s, want INTERFACE (declared where it is used)", f.Kind)
	}
}

func TestAuditFlagsOpenConstructorCalls(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("sde/s.go", "package sde\n\ntype Store struct{}\n\nfunc Open(p string) (*Store, error) { return nil, nil }\n")
	write("sdetest/f.go", "package sdetest\n\nimport \"example.com/m/sde\"\n\nfunc Fixture() { _, _ = sde.Open(\"x\") }\n")
	write("mcp/m.go", "package mcp\n\nimport \"example.com/m/sde\"\n\nfunc build() { _, _ = sde.Open(\"x\") }\n")
	fs, err := Audit(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		if f.Kind == "CONCRETE_CTOR" {
			got = append(got, f.File)
		}
	}
	if len(got) != 1 || got[0] != "mcp/m.go" {
		t.Errorf("CONCRETE_CTOR files = %v, want only mcp/m.go (sde.Open outside a root; a *test package is exempt)", got)
	}
}

// enforcedKinds are the audit kinds CI fails on. INTERFACE (the list of every
// interface and who uses it) stays informational. The audit skips _test.go
// files, so fakes declared in tests are never reported.
var enforcedKinds = map[string]bool{
	"CONCRETE_PARAM": true, // a producer type taken, held or returned concretely outside a composition root
	"CONCRETE_CTOR":  true, // a producer New*/Open* call outside a composition root or *test fixture package
	"PRODUCER_IFACE": true, // an exported interface in a producer package
	"TYPE_ONLY_FILE": true, // a file that only declares types, or any *_iface.go
	"IFACE_AWAY":     true, // an interface declared in a file that never uses it
}

func enforcedFindings(fs []Finding) []Finding {
	var out []Finding
	for _, f := range fs {
		if enforcedKinds[f.Kind] {
			out = append(out, f)
		}
	}
	return out
}

// TestRepoConformsToConsumerInterfaceRules is the repo-conformance gate: it
// fails on any CONCRETE_PARAM, CONCRETE_CTOR, PRODUCER_IFACE, TYPE_ONLY_FILE or
// IFACE_AWAY finding in the repository, so a change must add none.
func TestRepoConformsToConsumerInterfaceRules(t *testing.T) {
	fs, err := Audit("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range enforcedFindings(fs) {
		t.Errorf("%s (declare a small interface in the consumer, inject it from commands/, and keep types beside the code that uses them): %s", f.Kind, f)
	}
}

// TestEnforcedKindsFailOnViolations proves the gate can fail: the fixture tree
// holds a violation of every enforced kind, and the same filter the repo test
// uses must report all of them (and not the informational INTERFACE kind).
func TestEnforcedKindsFailOnViolations(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range enforcedFindings(fs) {
		seen[f.Kind] = true
	}
	// CONCRETE_CTOR needs a pointer-returning constructor called from outside
	// the producer and the roots, which the fixture tree has none of.
	root := t.TempDir()
	for rel, src := range map[string]string{
		"sde/s.go": "package sde\n\ntype Store struct{}\n\nfunc Open(p string) (*Store, error) { return nil, nil }\n",
		"mcp/m.go": "package mcp\n\nimport \"example.com/m/sde\"\n\nfunc build() { _, _ = sde.Open(\"x\") }\n",
	} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctor, err := Audit(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range enforcedFindings(ctor) {
		seen[f.Kind] = true
	}
	for kind := range enforcedKinds {
		if !seen[kind] {
			t.Errorf("no %s violation reported for a tree that has one: the repo gate could not fail on it", kind)
		}
	}
	for _, f := range enforcedFindings(fs) {
		if f.Kind == "INTERFACE" {
			t.Errorf("informational INTERFACE finding treated as enforced: %s", f)
		}
	}
}

func TestAuditTreatsUnexportedProducerInterfaceAsInternal(t *testing.T) {
	fs, err := Audit("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.Context == "sde.internalSeam" && f.Kind != "INTERFACE" {
			t.Errorf("unexported producer interface kind = %s, want INTERFACE", f.Kind)
		}
	}
}
