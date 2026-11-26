package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"

	urfave "github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/cli"
	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

type demoNested struct {
	Note string `json:"note"`
}

type demoIn struct {
	CharacterID   int32       `json:"characterId" jsonschema:"the character to look up"`
	CorporationID int64       `json:"corporationId,omitempty"`
	Types         []int64     `json:"types,omitempty"`
	Nested        *demoNested `json:"nested,omitempty"`
}

type demoOut struct {
	CharacterID   int32   `json:"characterId"`
	CorporationID int64   `json:"corporationId"`
	Types         []int64 `json:"types"`
	Note          string  `json:"note,omitempty"`
}

const (
	demoName        = "demo_lookup"
	demoDescription = "Look up a demo character. Returns its corporation and types."
	demoFailingID   = 13
)

func demoOp() operation.Operation {
	return operation.New(demoName, demoDescription, operation.Read,
		func(ctx context.Context, in demoIn) (demoOut, error) {
			operation.Progress(ctx, "looking up the demo character")
			if in.CharacterID == demoFailingID {
				return demoOut{}, errors.New("demo failed")
			}
			out := demoOut{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Types: in.Types}
			if out.Types == nil {
				out.Types = []int64{}
			}
			if in.Nested != nil {
				out.Note = in.Nested.Note
			}
			return out, nil
		})
}

// withPaths adds entries to cli.Paths and cli.Groups for the test's length.
func withPaths(t *testing.T, paths map[string][]string, groups map[string]string) {
	t.Helper()
	maps.Copy(cli.Paths, paths)
	maps.Copy(cli.Groups, groups)
	t.Cleanup(func() {
		for name := range paths {
			delete(cli.Paths, name)
		}
		for name := range groups {
			delete(cli.Groups, name)
		}
	})
}

func withDemoPath(t *testing.T) {
	t.Helper()
	withPaths(t, map[string][]string{demoName: {"demo", "lookup"}}, map[string]string{"demo": "demo commands"})
}

// staticSource returns ops as the run-time operations, counting calls.
func staticSource(calls *int, ops ...operation.Operation) cli.Source {
	return func(context.Context, func(string)) ([]operation.Operation, func() error, error) {
		*calls++
		return ops, func() error { return nil }, nil
	}
}

type result struct {
	stdout, stderr string
	code           int
}

// run builds the commands under a root whose exit handler does nothing, so
// the returned error carries the exit code instead of exiting the test.
func run(t *testing.T, cmds []*urfave.Command, stdin string, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := &urfave.Command{
		Name:           "neat",
		Commands:       cmds,
		Writer:         &stdout,
		ErrWriter:      &stderr,
		Reader:         strings.NewReader(stdin),
		ExitErrHandler: func(context.Context, *urfave.Command, error) {},
	}
	err := root.Run(context.Background(), append([]string{"neat"}, args...))
	code := 0
	if err != nil {
		var ec urfave.ExitCoder
		if !errors.As(err, &ec) {
			t.Fatalf("Run returned %v, which carries no exit code", err)
		}
		if err.Error() != "" {
			t.Errorf("exit error has message %q; the leaf must print its error itself", err.Error())
		}
		code = ec.ExitCode()
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

func build(t *testing.T, ops ...operation.Operation) []*urfave.Command {
	t.Helper()
	var calls int
	cmds, err := cli.Build(ops, staticSource(&calls, ops...))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return cmds
}

// errorLine decodes stderr's last line as {"error": "..."}.
func errorLine(t *testing.T, stderr string) string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
	last := lines[len(lines)-1]
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(last), &e); err != nil || e.Error == "" {
		t.Fatalf("stderr's last line %q is not {\"error\": ...} (%v)", last, err)
	}
	if !strings.HasSuffix(stderr, "\n") {
		t.Errorf("stderr %q is not newline-terminated", stderr)
	}
	return e.Error
}

func TestBuild_RunsOperationWithFlags(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", "7", "--corporation", "1000167")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	want := `{"characterId":7,"corporationId":1000167,"types":[]}` + "\n"
	if res.stdout != want {
		t.Errorf("stdout = %q, want %q", res.stdout, want)
	}
}

func TestBuild_FlagNames(t *testing.T) {
	withDemoPath(t)
	cmds := build(t, demoOp())
	leaf := cmds[0].Command("lookup")
	if leaf == nil {
		t.Fatal("no lookup command under demo")
	}
	var names []string
	for _, f := range leaf.Flags {
		names = append(names, f.Names()...)
	}
	for _, want := range []string{"character", "corporation", "types", "json", "pretty"} {
		if !slices.Contains(names, want) {
			t.Errorf("flags %v lack --%s", names, want)
		}
	}
	for _, bad := range []string{"nested", "note", "character-id", "characterId"} {
		if slices.Contains(names, bad) {
			t.Errorf("flags %v include --%s", names, bad)
		}
	}
}

func TestBuild_FlagNamingRule(t *testing.T) {
	type namingIn struct {
		AccessListID  int64    `json:"accessListId"`
		MarketGroupID int32    `json:"marketGroupID,omitempty"`
		TypeIDs       []int64  `json:"typeIds,omitempty"`
		IsBuyOrder    bool     `json:"isBuyOrder,omitempty"`
		Price         *float64 `json:"price,omitempty"`
		Names         []string `json:"names,omitempty"`
		Count         int      `json:"count,omitempty"` // int: no flag, --json only
		Skipped       string   `json:"-"`
	}
	op := operation.New(demoName, demoDescription, operation.Read,
		func(context.Context, namingIn) (demoOut, error) { return demoOut{}, nil })
	withDemoPath(t)
	leaf := build(t, op)[0].Command("lookup")
	var got []string
	for _, f := range leaf.Flags {
		got = append(got, f.Names()...)
	}
	want := []string{"access-list", "market-group", "type-ids", "is-buy-order", "price", "names", "json", "pretty"}
	if !slices.Equal(got, want) {
		t.Errorf("flags = %v, want %v", got, want)
	}
}

func TestBuild_RepeatableSliceFlag(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", "7", "--types", "34", "--types", "35")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	want := `{"characterId":7,"corporationId":0,"types":[34,35]}` + "\n"
	if res.stdout != want {
		t.Errorf("stdout = %q, want %q", res.stdout, want)
	}
}

func TestBuild_JSONFromStdin(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), `{"characterId":7,"nested":{"note":"hi"}}`, "demo", "lookup", "--json", "-")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	// Keys are sorted: EncodeOutput re-marshals the output after applying the
	// schema's defaults, as the MCP adapter does.
	want := `{"characterId":7,"corporationId":0,"note":"hi","types":[]}` + "\n"
	if res.stdout != want {
		t.Errorf("stdout = %q, want %q", res.stdout, want)
	}
}

func TestBuild_JSONInline(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--json", `{"characterId":8}`)
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	if want := `{"characterId":8,"corporationId":0,"types":[]}` + "\n"; res.stdout != want {
		t.Errorf("stdout = %q, want %q", res.stdout, want)
	}
}

func TestBuild_EmptyOrMalformedStdinIsUsageError(t *testing.T) {
	withDemoPath(t)
	for name, stdin := range map[string]string{"empty": "", "blank": " \n", "malformed": `{"characterId":`, "not an object": `[7]`} {
		t.Run(name, func(t *testing.T) {
			var calls int
			ops := []operation.Operation{demoOp()}
			cmds, err := cli.Build(ops, staticSource(&calls, ops...))
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			res := run(t, cmds, stdin, "demo", "lookup", "--json", "-")
			if res.code != 2 {
				t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
			}
			errorLine(t, res.stderr)
			if res.stdout != "" {
				t.Errorf("stdout = %q, want empty", res.stdout)
			}
			if calls != 0 {
				t.Errorf("source called %d times for bad input, want 0", calls)
			}
		})
	}
}

func TestBuild_NonNumericAndOutOfRangeIDIsUsageError(t *testing.T) {
	withDemoPath(t)
	for _, v := range []string{"abc", "99999999999"} {
		t.Run(v, func(t *testing.T) {
			res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", v)
			if res.code != 2 {
				t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
			}
			if msg := errorLine(t, res.stderr); !strings.Contains(msg, v) {
				t.Errorf("error %q does not name the bad value %q", msg, v)
			}
			if res.stdout != "" {
				t.Errorf("stdout = %q, want empty", res.stdout)
			}
		})
	}
}

func TestBuild_UnknownFlagIsUsageError(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", "7", "--bogus", "1")
	if res.code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
	}
	errorLine(t, res.stderr)
}

func TestBuild_PositionalArgumentIsUsageError(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", "7", "extra")
	if res.code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
	}
	if msg := errorLine(t, res.stderr); !strings.Contains(msg, "extra") {
		t.Errorf("error %q does not name the argument", msg)
	}
}

func TestBuild_JSONWithFlagsIsUsageError(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--json", `{"characterId":7}`, "--corporation", "1")
	if res.code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
	}
	if msg := errorLine(t, res.stderr); !strings.Contains(msg, "--corporation") {
		t.Errorf("error %q does not name the conflicting flag", msg)
	}
}

func TestBuild_MissingRequiredIsUsageError(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--corporation", "1")
	if res.code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
	}
	// Given with flags, the message names the flag to pass, not the field.
	msg := errorLine(t, res.stderr)
	if !strings.Contains(msg, "--character") || strings.Contains(msg, "characterId") {
		t.Errorf("error %q should name --character and not characterId", msg)
	}
}

func TestBuild_MissingRequiredWithJSONNamesTheField(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--json", `{"corporationId":1}`)
	if res.code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
	}
	if msg := errorLine(t, res.stderr); !strings.Contains(msg, `"characterId"`) {
		t.Errorf("error %q does not name the missing field as written in --json", msg)
	}
}

func TestBuild_OperationErrorIsExit1(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", "13")
	if res.code != 1 {
		t.Errorf("exit %d, want 1 (stderr %q)", res.code, res.stderr)
	}
	if msg := errorLine(t, res.stderr); msg != "demo failed" {
		t.Errorf("error = %q, want %q", msg, "demo failed")
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty", res.stdout)
	}
}

func TestBuild_InvalidInputFromRunIsExit2(t *testing.T) {
	withDemoPath(t)
	op := operation.New(demoName, demoDescription, operation.Read,
		func(context.Context, demoIn) (demoOut, error) {
			return demoOut{}, errors.Join(operation.ErrInvalidInput, errors.New("no such corporation"))
		})
	res := run(t, build(t, op), "", "demo", "lookup", "--character", "7")
	if res.code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
	}
}

func TestBuild_SourceErrorIsExit1(t *testing.T) {
	withDemoPath(t)
	ops := []operation.Operation{demoOp()}
	cmds, err := cli.Build(ops, func(context.Context, func(string)) ([]operation.Operation, func() error, error) {
		return nil, nil, errors.New("config unreadable")
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res := run(t, cmds, "", "demo", "lookup", "--character", "7")
	if res.code != 1 || !strings.Contains(errorLine(t, res.stderr), "config unreadable") {
		t.Errorf("source error: exit %d, stderr %q; want exit 1 naming it", res.code, res.stderr)
	}
}

// failingClose is a source whose close fails, counting the closes.
func failingClose(closed *int, ops ...operation.Operation) cli.Source {
	return func(context.Context, func(string)) ([]operation.Operation, func() error, error) {
		return ops, func() error { *closed++; return errors.New("sde close failed") }, nil
	}
}

// warningLine decodes line as {"warning": "..."}.
func warningLine(t *testing.T, line string) string {
	t.Helper()
	var w struct {
		Warning string `json:"warning"`
	}
	if err := json.Unmarshal([]byte(line), &w); err != nil || w.Warning == "" {
		t.Fatalf("line %q is not {\"warning\": ...} (%v)", line, err)
	}
	return w.Warning
}

func TestBuild_CloseErrorAfterSuccessIsWarning(t *testing.T) {
	withDemoPath(t)
	closed := 0
	cmds, err := cli.Build([]operation.Operation{demoOp()}, failingClose(&closed, demoOp()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res := run(t, cmds, "", "demo", "lookup", "--character", "7")
	if closed != 1 {
		t.Errorf("close called %d times, want 1", closed)
	}
	if res.code != 0 {
		t.Errorf("exit %d, want 0 (stderr %q)", res.code, res.stderr)
	}
	if want := `{"characterId":7,"corporationId":0,"types":[]}` + "\n"; res.stdout != want {
		t.Errorf("stdout = %q, want the result %q", res.stdout, want)
	}
	lines := strings.Split(strings.TrimRight(res.stderr, "\n"), "\n")
	if len(lines) != 2 || lines[0] != "looking up the demo character" {
		t.Fatalf("stderr = %q, want the progress line then one warning line", res.stderr)
	}
	if msg := warningLine(t, lines[1]); !strings.Contains(msg, "sde close failed") {
		t.Errorf("warning %q does not name the close failure", msg)
	}
}

func TestBuild_CloseErrorAfterFailureJoinsAndKeepsExitCode(t *testing.T) {
	withDemoPath(t)
	for name, tc := range map[string]struct {
		op   operation.Operation
		code int
		want string
	}{
		"operation error": {op: demoOp(), code: 1, want: "demo failed"},
		"invalid input from Run": {op: operation.New(demoName, demoDescription, operation.Read,
			func(context.Context, demoIn) (demoOut, error) {
				return demoOut{}, errors.Join(operation.ErrInvalidInput, errors.New("no such corporation"))
			}), code: 2, want: "no such corporation"},
	} {
		t.Run(name, func(t *testing.T) {
			closed := 0
			cmds, err := cli.Build([]operation.Operation{tc.op}, failingClose(&closed, tc.op))
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			res := run(t, cmds, "", "demo", "lookup", "--character", "13")
			if closed != 1 {
				t.Errorf("close called %d times, want 1", closed)
			}
			if res.code != tc.code {
				t.Errorf("exit %d, want %d (stderr %q)", res.code, tc.code, res.stderr)
			}
			msg := errorLine(t, res.stderr)
			for _, want := range []string{tc.want, "sde close failed"} {
				if !strings.Contains(msg, want) {
					t.Errorf("error %q does not name %q", msg, want)
				}
			}
			if res.stdout != "" {
				t.Errorf("stdout = %q, want empty", res.stdout)
			}
		})
	}
}

func TestBuild_SourceWarningIsJSONLine(t *testing.T) {
	withDemoPath(t)
	ops := []operation.Operation{demoOp()}
	cmds, err := cli.Build(ops, func(_ context.Context, warn func(string)) ([]operation.Operation, func() error, error) {
		warn(`cache "x" unavailable`)
		return ops, nil, nil
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res := run(t, cmds, "", "demo", "lookup", "--character", "7")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	first, _, _ := strings.Cut(res.stderr, "\n")
	if msg := warningLine(t, first); msg != `cache "x" unavailable` {
		t.Errorf("warning = %q, want the source's message", msg)
	}
}

func TestBuild_ProgressFromManyGoroutines(t *testing.T) {
	const workers, each = 8, 50
	op := operation.New(demoName, demoDescription, operation.Read,
		func(ctx context.Context, in demoIn) (demoOut, error) {
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range each {
						operation.Progress(ctx, "step")
					}
				})
			}
			wg.Wait()
			return demoOut{CharacterID: in.CharacterID, Types: []int64{}}, nil
		})
	withDemoPath(t)
	res := run(t, build(t, op), "", "demo", "lookup", "--character", "7")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	if want := strings.Repeat("step\n", workers*each); res.stderr != want {
		t.Errorf("stderr has %d bytes, want %d whole step lines", len(res.stderr), workers*each)
	}
}

func TestBuild_ZeroFieldInput(t *testing.T) {
	type emptyIn struct{}
	op := operation.New(demoName, demoDescription, operation.Read,
		func(context.Context, emptyIn) (demoOut, error) { return demoOut{CharacterID: 1, Types: []int64{}}, nil })
	withDemoPath(t)
	cmds := build(t, op)
	leaf := cmds[0].Command("lookup")
	var names []string
	for _, f := range leaf.Flags {
		names = append(names, f.Names()...)
	}
	if want := []string{"json", "pretty"}; !slices.Equal(names, want) {
		t.Errorf("flags = %v, want only %v", names, want)
	}
	want := `{"characterId":1,"corporationId":0,"types":[]}` + "\n"
	for _, args := range [][]string{{"demo", "lookup"}, {"demo", "lookup", "--json", "{}"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := run(t, cmds, "", args...)
			if res.code != 0 {
				t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
			}
			if res.stdout != want {
				t.Errorf("stdout = %q, want %q", res.stdout, want)
			}
		})
	}
}

func TestBuild_UnknownCommandUnderGroupIsUsageError(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "bogus")
	if res.code != 2 {
		t.Errorf("exit %d, want 2 (stderr %q)", res.code, res.stderr)
	}
	if msg := errorLine(t, res.stderr); !strings.Contains(msg, "bogus") {
		t.Errorf("error %q does not name the unknown command", msg)
	}
	if strings.Count(res.stderr, "\n") != 1 {
		t.Errorf("stderr = %q, want only the error line", res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty", res.stdout)
	}
}

func TestBuild_PrettyFlag(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", "7", "--pretty")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	want := "{\n  \"characterId\": 7,\n  \"corporationId\": 0,\n  \"types\": []\n}\n"
	if res.stdout != want {
		t.Errorf("stdout = %q, want %q", res.stdout, want)
	}
}

func TestBuild_StdoutHoldsOnlyResult(t *testing.T) {
	withDemoPath(t)
	res := run(t, build(t, demoOp()), "", "demo", "lookup", "--character", "7")
	if res.code != 0 {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(res.stdout), &v); err != nil {
		t.Errorf("stdout %q is not one JSON value: %v", res.stdout, err)
	}
	if res.stderr != "looking up the demo character\n" {
		t.Errorf("stderr = %q, want the progress line", res.stderr)
	}
}

func TestBuild_HelpDoesNotCallSource(t *testing.T) {
	withDemoPath(t)
	ops := []operation.Operation{demoOp()}
	panicking := func(context.Context, func(string)) ([]operation.Operation, func() error, error) {
		panic("source called for help")
	}
	for _, args := range [][]string{{"--help"}, {"demo", "--help"}, {"demo", "lookup", "--help"}, {"demo"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmds, err := cli.Build(ops, panicking)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			res := run(t, cmds, "", args...)
			if res.code != 0 {
				t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
			}
			if res.stdout == "" {
				t.Error("help printed nothing")
			}
		})
	}
	cmds, err := cli.Build(ops, panicking)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	res := run(t, cmds, "", "demo", "lookup", "--help")
	for _, want := range []string{"Look up a demo character", demoDescription, "--character", "the character to look up"} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("leaf help lacks %q:\n%s", want, res.stdout)
		}
	}
	if strings.Contains(res.stdout, "changes state") {
		t.Errorf("help of a Read operation says it changes state:\n%s", res.stdout)
	}
}

func TestBuild_WriteOperationHelpSaysChangesState(t *testing.T) {
	withDemoPath(t)
	op := operation.New(demoName, demoDescription, operation.Write,
		func(context.Context, demoIn) (demoOut, error) { return demoOut{}, nil })
	cmds := build(t, op)
	res := run(t, cmds, "", "demo", "lookup", "--help")
	if !strings.Contains(res.stdout, "changes state") {
		t.Errorf("help of a Write operation does not say it changes state:\n%s", res.stdout)
	}
}

func TestBuild_OperationWithoutPathIsError(t *testing.T) {
	op := operation.New("unmapped_demo", "Not in the table.", operation.Read,
		func(context.Context, demoIn) (demoOut, error) { return demoOut{}, nil })
	var calls int
	_, err := cli.Build([]operation.Operation{op}, staticSource(&calls, op))
	if err == nil || !strings.Contains(err.Error(), "unmapped_demo") {
		t.Fatalf("Build error = %v, want one naming unmapped_demo", err)
	}
}

func TestBuild_SharedPathIsError(t *testing.T) {
	other := operation.New("demo_other", "Another demo.", operation.Read,
		func(context.Context, demoIn) (demoOut, error) { return demoOut{}, nil })
	withPaths(t, map[string][]string{demoName: {"demo", "lookup"}, "demo_other": {"demo", "lookup"}},
		map[string]string{"demo": "demo commands"})
	var calls int
	_, err := cli.Build([]operation.Operation{demoOp(), other}, staticSource(&calls))
	if err == nil || !strings.Contains(err.Error(), "demo lookup") {
		t.Fatalf("Build error = %v, want one naming the shared path", err)
	}
}

func TestBuild_DuplicateFlagNameIsError(t *testing.T) {
	type clashIn struct {
		CharacterID int32 `json:"characterId"`
		Character   int32 `json:"character,omitempty"`
	}
	op := operation.New(demoName, demoDescription, operation.Read,
		func(context.Context, clashIn) (demoOut, error) { return demoOut{}, nil })
	withDemoPath(t)
	var calls int
	_, err := cli.Build([]operation.Operation{op}, staticSource(&calls))
	if err == nil || !strings.Contains(err.Error(), "--character") {
		t.Fatalf("Build error = %v, want one naming --character", err)
	}
}

func TestBuild_GroupWithoutUsageIsError(t *testing.T) {
	withPaths(t, map[string][]string{demoName: {"nogroup", "lookup"}}, nil)
	var calls int
	_, err := cli.Build([]operation.Operation{demoOp()}, staticSource(&calls))
	if err == nil || !strings.Contains(err.Error(), "nogroup") {
		t.Fatalf("Build error = %v, want one naming the group", err)
	}
}
