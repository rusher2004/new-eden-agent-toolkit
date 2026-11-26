// Package cli is the command line adapter: it builds neat's urfave/cli
// command tree from operation.Operation values. Each operation is a leaf
// command at its Paths entry; its flags come from its input struct, it writes
// its result as JSON on stdout, and it reports a failure as {"error": "..."}
// on stderr with exit code 1, or 2 for a usage error. Stdout holds nothing
// but the result; stderr holds progress lines, warnings as
// {"warning": "..."}, and the error. A failure to release the sources after
// a successful run is a warning, not a failure. An unknown command under a
// group (or under the root, through HelpOrUnknown) is a usage error too.
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
	"strconv"
	"strings"
	"sync"

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
// warn reports something the caller should know that does not stop the run
// (a cache that could not open); the leaf writes it on stderr as
// {"warning": "..."}.
type Source = func(ctx context.Context, warn func(msg string)) (ops []operation.Operation, closeFn func() error, err error)

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
			g = &urfave.Command{Name: path[i], Usage: usage, Action: HelpOrUnknown}
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

// HelpOrUnknown is the action of a command that only groups others: with no
// argument it shows the command's help, and with one (a command name that
// matched none of its subcommands) it fails as a usage error, writing
// {"error": "..."} on stderr and exiting 2. Build sets it on every group;
// the root command sets it itself.
func HelpOrUnknown(_ context.Context, cmd *urfave.Command) error {
	if !cmd.Args().Present() {
		show := urfave.ShowSubcommandHelp
		if cmd.Root() == cmd {
			show = urfave.ShowRootCommandHelp
		}
		if err := show(cmd); err != nil {
			return fmt.Errorf("show help for %q: %w", cmd.FullName(), err)
		}
		return nil
	}
	return fail(cmd, fmt.Errorf("unknown command %q for %q; run %q for the list", cmd.Args().First(), cmd.FullName(), cmd.FullName()+" --help"), exitUsage)
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
	stderr := &stderrLines{w: cmd.Root().ErrWriter}
	data, closeFn, code, err := l.run(ctx, cmd, stderr)
	if err != nil {
		// The operation failed: a close failure joins its error, and the exit
		// code stays the one the failure chose.
		err = errors.Join(err, l.close(closeFn))
		return fail(cmd, l.usageMessage(cmd, err, code), code)
	}
	if err := l.write(cmd, data); err != nil {
		return fail(cmd, errors.Join(err, l.close(closeFn)), exitFailure)
	}
	// The result is out, so a close failure no longer makes the run a
	// failure: it is reported as a warning and the exit code stays 0.
	if cerr := l.close(closeFn); cerr != nil {
		stderr.warning(cerr.Error())
	}
	if err := stderr.Err(); err != nil {
		return fail(cmd, fmt.Errorf("%s: %w", l.op.Name, err), exitFailure)
	}
	return nil
}

// write prints data, indented with --pretty, as one line on stdout.
func (l leaf) write(cmd *urfave.Command, data json.RawMessage) error {
	if cmd.Bool(prettyFlag) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err != nil {
			return fmt.Errorf("%s: indent output: %w", l.op.Name, err)
		}
		data = buf.Bytes()
	}
	if _, err := fmt.Fprintf(cmd.Root().Writer, "%s\n", data); err != nil {
		return fmt.Errorf("%s: write output: %w", l.op.Name, err)
	}
	return nil
}

// close calls closeFn, when there is one, and wraps its error.
func (l leaf) close(closeFn func() error) error {
	if closeFn == nil {
		return nil
	}
	if err := closeFn(); err != nil {
		return fmt.Errorf("%s: close sources: %w", l.op.Name, err)
	}
	return nil
}

// usageMessage names flags instead of JSON fields in an invalid-input error
// given with flags: a schema error such as missing properties: ["characterId"]
// becomes missing properties: [--character], so the caller is pointed at the
// flag to pass. The rest of the message is left as it is. Input given with
// --json keeps the field names, which are what the caller wrote.
func (l leaf) usageMessage(cmd *urfave.Command, err error, code int) error {
	if code != exitUsage || !errors.Is(err, operation.ErrInvalidInput) || cmd.IsSet(jsonFlag) {
		return err
	}
	msg := err.Error()
	for _, f := range l.flags {
		msg = strings.ReplaceAll(msg, strconv.Quote(f.jsonName), "--"+f.name)
	}
	if msg == err.Error() {
		return err
	}
	return errors.New(msg)
}

// run decodes the input, builds the sources, runs the operation and encodes
// its output, returning the exit code that goes with an error and the
// function that closes the sources, which the caller calls once the result is
// written (or the failure known). closeFn is returned with an error too.
func (l leaf) run(ctx context.Context, cmd *urfave.Command, stderr *stderrLines) (data json.RawMessage, closeFn func() error, code int, err error) {
	if cmd.Args().Present() {
		return nil, nil, exitUsage, fmt.Errorf("unexpected argument %q: input is given with flags or --json", cmd.Args().First())
	}
	raw, err := l.input(cmd)
	if err != nil {
		return nil, nil, exitUsage, err
	}
	// Validate before building any source, so bad input touches nothing.
	if _, err := l.op.Decode(raw); err != nil {
		return nil, nil, exitUsage, err
	}

	ops, closeFn, err := l.source(ctx, stderr.warning)
	if err != nil {
		return nil, closeFn, exitFailure, fmt.Errorf("%s: open sources: %w", l.op.Name, err)
	}
	i := slices.IndexFunc(ops, func(o operation.Operation) bool { return o.Name == l.op.Name })
	if i < 0 {
		return nil, closeFn, exitFailure, fmt.Errorf("%s: the source provides no operation of that name", l.op.Name)
	}
	op := ops[i]
	in, err := op.Decode(raw)
	if err != nil {
		return nil, closeFn, exitUsage, err
	}

	out, err := op.Run(operation.WithProgress(ctx, stderr.progress), in)
	if err != nil {
		code := exitFailure
		if errors.Is(err, operation.ErrInvalidInput) {
			code = exitUsage
		}
		if serr := stderr.Err(); serr != nil {
			err = errors.Join(err, fmt.Errorf("%s: %w", op.Name, serr))
		}
		return nil, closeFn, code, err
	}
	if serr := stderr.Err(); serr != nil {
		return nil, closeFn, exitFailure, fmt.Errorf("%s: %w", op.Name, serr)
	}
	// The same encoding mcp.Register uses, so both adapters return the same
	// bytes for the same result.
	data, err = op.EncodeOutput(out)
	if err != nil {
		return nil, closeFn, exitFailure, err
	}
	if data == nil {
		data = json.RawMessage("null")
	}
	return data, closeFn, 0, nil
}

// stderrLines writes a leaf's progress and warning lines on stderr. An
// operation may report progress from several goroutines, so writes are
// serialized. The sinks can't return an error, so the first failed write is
// kept, later lines are dropped, and the leaf reads it with Err: a failure if
// the operation had not already failed, joined into its error if it had.
type stderrLines struct {
	mu  sync.Mutex
	w   io.Writer
	err error
}

// progress writes msg as a plain line.
func (s *stderrLines) progress(msg string) {
	s.writeLine([]byte(msg), "progress")
}

// warning writes msg as {"warning": "..."}.
func (s *stderrLines) warning(msg string) {
	line, err := json.Marshal(struct {
		Warning string `json:"warning"`
	}{Warning: msg})
	if err != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.err == nil {
			s.err = fmt.Errorf("encode warning %q: %w", msg, err)
		}
		return
	}
	s.writeLine(line, "warning")
}

func (s *stderrLines) writeLine(line []byte, what string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return
	}
	if _, err := fmt.Fprintf(s.w, "%s\n", line); err != nil {
		s.err = fmt.Errorf("write %s: %w", what, err)
	}
}

// Err is the first failed write, or nil.
func (s *stderrLines) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
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
