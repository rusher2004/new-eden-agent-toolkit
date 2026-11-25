// Package cli is the command line adapter: it builds neat's urfave/cli
// command tree from operation.Operation values. Each operation is a leaf
// command at its Paths entry; its flags come from its input struct, it writes
// its result as JSON on stdout, and it reports a failure as {"error": "..."}
// on stderr with exit code 1, or 2 for a usage error.
//
// It imports only the standard library, urfave/cli and operation (enforced by
// tools/archaudit), so it knows nothing about any domain or data source.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	urfave "github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

// Exit codes a leaf returns.
const (
	exitFailure = 1
	exitUsage   = 2
)

// Source builds the working operations when a leaf runs, with a function that
// releases what it opened. Build never calls it while building or for help.
type Source = func(ctx context.Context) (ops []operation.Operation, closeFn func() error, err error)

// Build returns the top-level group commands for describe, the operations
// whose names, descriptions and input types give the commands, help and
// flags. source is called only when a leaf runs; the leaf runs the operation
// of the same name from what it returns.
//
// It fails when a described operation has no Paths entry or no description,
// when two operations share a path, when a path is both a leaf and a group,
// when a group has no Groups entry, or when two input fields of one
// operation would take the same flag.
func Build(describe []operation.Operation, source Source) ([]*urfave.Command, error) {
	var (
		errs   []error
		top    []*urfave.Command
		groups = map[string]*urfave.Command{}
		leaves = map[string]string{} // joined path -> operation name
	)
	for _, op := range describe {
		path, ok := Paths[op.Name]
		if !ok {
			errs = append(errs, fmt.Errorf("operation %q has no command path in cli.Paths", op.Name))
			continue
		}
		if len(path) < 2 {
			errs = append(errs, fmt.Errorf("operation %q: command path %q needs a group and a leaf", op.Name, path))
			continue
		}
		if strings.TrimSpace(op.Description) == "" {
			errs = append(errs, fmt.Errorf("operation %q has no description", op.Name))
			continue
		}
		key := strings.Join(path, " ")
		if other, dup := leaves[key]; dup {
			errs = append(errs, fmt.Errorf("operations %q and %q share the command path %q", other, op.Name, key))
			continue
		}
		leaves[key] = op.Name
		leaf, err := leafCommand(op, path[len(path)-1], source)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		parent, err := group(path[:len(path)-1], groups, &top)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		parent.Commands = append(parent.Commands, leaf)
	}
	for key, name := range leaves {
		if _, clash := groups[key]; clash {
			errs = append(errs, fmt.Errorf("operation %q: command path %q is also a group", name, key))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("build cli commands: %w", err)
	}
	sortCommands(top)
	return top, nil
}

// group returns the group command at path, creating it and its ancestors.
func group(path []string, groups map[string]*urfave.Command, top *[]*urfave.Command) (*urfave.Command, error) {
	var parent *urfave.Command
	for i := range path {
		key := strings.Join(path[:i+1], " ")
		g, ok := groups[key]
		if !ok {
			usage, ok := Groups[key]
			if !ok {
				return nil, fmt.Errorf("command group %q has no help in cli.Groups", key)
			}
			g = &urfave.Command{Name: path[i], Usage: usage}
			groups[key] = g
			if parent == nil {
				*top = append(*top, g)
			} else {
				parent.Commands = append(parent.Commands, g)
			}
		}
		parent = g
	}
	return parent, nil
}

func sortCommands(cmds []*urfave.Command) {
	slices.SortFunc(cmds, func(a, b *urfave.Command) int { return strings.Compare(a.Name, b.Name) })
	for _, c := range cmds {
		sortCommands(c.Commands)
	}
}

// leaf is one operation's command.
type leaf struct {
	op     operation.Operation
	flags  []inputFlag
	source Source
}

func leafCommand(op operation.Operation, name string, source Source) (*urfave.Command, error) {
	flags, err := inputFlags(op)
	if err != nil {
		return nil, err
	}
	l := leaf{op: op, flags: flags, source: source}

	usage, description := firstSentence(op.Description), op.Description
	if op.Kind == operation.Write {
		usage += " (changes state)"
		description += "\n\nThis command changes state."
	}
	cmdFlags := make([]urfave.Flag, 0, len(flags)+2)
	for _, f := range flags {
		cmdFlags = append(cmdFlags, f.flag)
	}
	cmdFlags = append(cmdFlags,
		&urfave.StringFlag{Name: jsonFlag, Usage: "the whole input as one JSON object, or - to read it from stdin; not combined with other input flags"},
		&urfave.BoolFlag{Name: prettyFlag, Usage: "indent the JSON output"},
	)
	return &urfave.Command{
		Name:        name,
		Usage:       usage,
		Description: description,
		Flags:       cmdFlags,
		// Flag parse errors (a bad number, an unknown flag) come here.
		OnUsageError: func(_ context.Context, cmd *urfave.Command, err error, _ bool) error {
			return fail(cmd, err, exitUsage)
		},
		Action: l.action,
	}, nil
}

func (l leaf) action(ctx context.Context, cmd *urfave.Command) error {
	data, code, err := l.run(ctx, cmd)
	if err != nil {
		return fail(cmd, err, code)
	}
	if cmd.Bool(prettyFlag) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err != nil {
			return fail(cmd, fmt.Errorf("%s: indent output: %w", l.op.Name, err), exitFailure)
		}
		data = buf.Bytes()
	}
	if _, err := fmt.Fprintf(cmd.Root().Writer, "%s\n", data); err != nil {
		return fail(cmd, fmt.Errorf("%s: write output: %w", l.op.Name, err), exitFailure)
	}
	return nil
}

// run decodes the input, builds the sources, runs the operation and encodes
// its output, returning the exit code that goes with an error.
func (l leaf) run(ctx context.Context, cmd *urfave.Command) (data json.RawMessage, code int, err error) {
	if cmd.Args().Present() {
		return nil, exitUsage, fmt.Errorf("unexpected argument %q: input is given with flags or --json", cmd.Args().First())
	}
	raw, err := l.input(cmd)
	if err != nil {
		return nil, exitUsage, err
	}
	// Validate before building any source, so bad input touches nothing.
	if _, err := l.op.Decode(raw); err != nil {
		return nil, exitUsage, err
	}

	ops, closeFn, err := l.source(ctx)
	if err != nil {
		return nil, exitFailure, fmt.Errorf("%s: open sources: %w", l.op.Name, err)
	}
	if closeFn != nil {
		defer func() {
			if cerr := closeFn(); cerr != nil {
				// A result the caller can't trust its cleanup for is still
				// reported as a failure: the error is never dropped.
				data, code, err = nil, exitFailure, errors.Join(err, fmt.Errorf("%s: close sources: %w", l.op.Name, cerr))
			}
		}()
	}
	i := slices.IndexFunc(ops, func(o operation.Operation) bool { return o.Name == l.op.Name })
	if i < 0 {
		return nil, exitFailure, fmt.Errorf("%s: the source provides no operation of that name", l.op.Name)
	}
	op := ops[i]
	in, err := op.Decode(raw)
	if err != nil {
		return nil, exitUsage, err
	}

	var progressErr error
	errW := cmd.Root().ErrWriter
	ctx = operation.WithProgress(ctx, func(msg string) {
		if progressErr != nil {
			return
		}
		if _, err := fmt.Fprintln(errW, msg); err != nil {
			progressErr = fmt.Errorf("%s: write progress: %w", op.Name, err)
		}
	})
	out, err := op.Run(ctx, in)
	if err != nil {
		code := exitFailure
		if errors.Is(err, operation.ErrInvalidInput) {
			code = exitUsage
		}
		return nil, code, errors.Join(err, progressErr)
	}
	if progressErr != nil {
		return nil, exitFailure, progressErr
	}
	// The same encoding mcp.Register uses, so both adapters return the same
	// bytes for the same result.
	data, err = op.EncodeOutput(out)
	if err != nil {
		return nil, exitFailure, err
	}
	if data == nil {
		data = json.RawMessage("null")
	}
	return data, 0, nil
}

// input builds the operation's input JSON from --json or the input flags.
func (l leaf) input(cmd *urfave.Command) ([]byte, error) {
	obj := map[string]any{}
	var set []string
	for _, f := range l.flags {
		if cmd.IsSet(f.name) {
			set = append(set, "--"+f.name)
			obj[f.jsonName] = cmd.Value(f.name)
		}
	}
	if !cmd.IsSet(jsonFlag) {
		raw, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("%s: encode flags as input: %w", l.op.Name, err)
		}
		return raw, nil
	}
	if len(set) > 0 {
		return nil, fmt.Errorf("--json cannot be combined with %s", strings.Join(set, ", "))
	}
	raw := []byte(cmd.String(jsonFlag))
	source := "--json"
	if string(raw) == "-" {
		source = "stdin"
		r := cmd.Root().Reader
		if r == nil {
			return nil, errors.New("--json -: no stdin to read")
		}
		var err error
		if raw, err = io.ReadAll(r); err != nil {
			return nil, fmt.Errorf("--json -: read stdin: %w", err)
		}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("--json: empty input from %s, want a JSON object", source)
	}
	return raw, nil
}

// fail writes err as {"error": "..."} on stderr and returns an exit error with
// no message, so nothing prints it again.
func fail(cmd *urfave.Command, err error, code int) error {
	line, merr := json.Marshal(struct {
		Error string `json:"error"`
	}{Error: err.Error()})
	if merr != nil {
		return urfave.Exit(fmt.Sprintf("%v (encoding the error as JSON failed: %v)", err, merr), code)
	}
	if _, werr := fmt.Fprintf(cmd.Root().ErrWriter, "%s\n", line); werr != nil {
		return urfave.Exit(fmt.Sprintf("%v (writing the error to stderr failed: %v)", err, werr), code)
	}
	return urfave.Exit("", code)
}

// firstSentence is description up to its first sentence end, without the
// period, for a command's one-line usage.
func firstSentence(description string) string {
	s := strings.TrimSpace(description)
	if i := strings.Index(s, ". "); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSuffix(s, ".")
}
