// Package operation is the contract between neat's domain services and the
// adapters that expose them (the MCP server, the CLI). A domain service wraps
// each of its methods in New; an adapter reads the name, description, kind and
// schemas, decodes caller input with Decode and calls Run. The package knows
// nothing about any adapter or data source.
package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
)

// Kind says whether an operation changes state.
type Kind int

const (
	// Read operations change nothing.
	Read Kind = iota
	// Write operations change state in the game or in ~/.neat.
	Write
)

// ErrInvalidInput is wrapped by every error Decode returns, so adapters can
// tell a caller's bad input from a failure inside the operation.
var ErrInvalidInput = errors.New("invalid input")

// Operation is one thing neat can do, independent of how it is invoked.
type Operation struct {
	Name, Description string
	Kind              Kind
	// InputType and OutputType are the reflect.Type of New's In and Out.
	InputType, OutputType reflect.Type
	// Run calls the operation. in must be the value Decode returned (a *In);
	// anything else is an error. The result is an Out.
	Run func(ctx context.Context, in any) (any, error)

	inputSchema    *jsonschema.Schema
	inputResolved  *jsonschema.Resolved
	outputSchema   *jsonschema.Schema
	outputResolved *jsonschema.Resolved
	// outputZero is the zero value of Out's element type when Out is a
	// pointer, used in place of a typed nil; nil otherwise.
	outputZero any
	decodeInto func(raw []byte) (any, error)
}

// New wraps a typed function as an Operation, deriving its input and output
// JSON schemas from In's and Out's json and jsonschema struct tags.
//
// Schemas are derived exactly as go-sdk v1.8.0's typed mcp.AddTool derives
// them (toolForErr/setSchema in its mcp/server.go), so registering an
// Operation through the SDK's low-level AddTool lists the same schemas:
// jsonschema.ForType with empty options on the type with one pointer level
// removed, resolved with ValidateDefaults; an In of type any gets
// {"type":"object"}, and an Out of type any gets no output schema.
//
// New panics when a schema can't be derived or resolved (for example a
// self-referential type, or an unsupported field type such as a channel).
// That is what the SDK's AddTool does for the same types, and operations are
// built once at startup from fixed Go types, so the failure is a programming
// error that surfaces at registration with the operation's name, not at a
// call.
func New[In, Out any](name, description string, kind Kind, run func(context.Context, In) (Out, error)) Operation {
	inSchema, inResolved, err := schemaFor[In]()
	if err != nil {
		panic(fmt.Sprintf("operation.New: %q: input schema: %v", name, err))
	}
	var (
		outSchema   *jsonschema.Schema
		outResolved *jsonschema.Resolved
		outZero     any
	)
	if reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		outSchema, outResolved, err = schemaFor[Out]()
		if err != nil {
			panic(fmt.Sprintf("operation.New: %q: output schema: %v", name, err))
		}
		if rt := reflect.TypeFor[Out](); rt.Kind() == reflect.Pointer {
			outZero = reflect.Zero(rt.Elem()).Interface()
		}
	}

	return Operation{
		Name:        name,
		Description: description,
		Kind:        kind,
		InputType:   reflect.TypeFor[In](),
		OutputType:  reflect.TypeFor[Out](),
		Run: func(ctx context.Context, in any) (any, error) {
			p, ok := in.(*In)
			if !ok || p == nil {
				return nil, fmt.Errorf("%s: run: input is %T, want the *%v Decode returns", name, in, reflect.TypeFor[In]())
			}
			return run(ctx, *p)
		},
		inputSchema:    inSchema,
		inputResolved:  inResolved,
		outputSchema:   outSchema,
		outputResolved: outResolved,
		outputZero:     outZero,
		decodeInto: func(raw []byte) (any, error) {
			in := new(In)
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(in); err != nil {
				return nil, err
			}
			return in, nil
		},
	}
}

// schemaFor derives and resolves T's schema the way the SDK's setSchema does.
func schemaFor[T any]() (*jsonschema.Schema, *jsonschema.Resolved, error) {
	rt := reflect.TypeFor[T]()
	var s *jsonschema.Schema
	if rt == reflect.TypeFor[any]() {
		s = &jsonschema.Schema{Type: "object"}
	} else {
		if rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
		}
		var err error
		s, err = jsonschema.ForType(rt, &jsonschema.ForOptions{})
		if err != nil {
			return nil, nil, fmt.Errorf("derive schema for %v: %w", rt, err)
		}
		if s == nil {
			return nil, nil, fmt.Errorf("derive schema for %v: schema is nil", rt)
		}
	}
	resolved, err := s.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		return nil, nil, fmt.Errorf("resolve schema for %v: %w", rt, err)
	}
	return s, resolved, nil
}

// Decode validates raw JSON against the input schema, applies its defaults
// and returns a *In for Run. Empty or nil raw is treated as {}, since tools
// without arguments send none. Every error wraps ErrInvalidInput.
//
// It mirrors the SDK's typed tool handler (applySchema in its mcp/tool.go):
// decode into a map, apply defaults, validate, then decode into In.
func (o Operation) Decode(raw []byte) (any, error) {
	if o.decodeInto == nil {
		return nil, fmt.Errorf("%w: operation %q was not built with operation.New", ErrInvalidInput, o.Name)
	}
	var v any = map[string]any{}
	if len(raw) > 0 {
		m := map[string]any{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("%w: %s: unmarshaling arguments: %w", ErrInvalidInput, o.Name, err)
		}
		v = m
	}
	if err := o.inputResolved.ApplyDefaults(&v); err != nil {
		return nil, fmt.Errorf("%w: %s: applying schema defaults: %w", ErrInvalidInput, o.Name, err)
	}
	if err := o.inputResolved.Validate(&v); err != nil {
		return nil, fmt.Errorf("%w: %s: validating arguments: %w", ErrInvalidInput, o.Name, err)
	}
	withDefaults, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: marshaling arguments with defaults: %w", ErrInvalidInput, o.Name, err)
	}
	in, err := o.decodeInto(withDefaults)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: decoding arguments: %w", ErrInvalidInput, o.Name, err)
	}
	return in, nil
}

// EncodeOutput marshals out, a value Run returned, to JSON, validates it
// against the output schema and applies the schema's defaults. A typed nil
// pointer encodes as its element type's zero value rather than null. It
// returns nil, nil for a nil out from an operation whose Out is any, which has
// no output schema.
//
// It mirrors the output half of go-sdk v1.8.0's typed tool handler
// (toolForErr in its mcp/server.go, applySchema with forOutput in its
// mcp/tool.go): the JSON is decoded into an untyped value, defaults are
// applied when it is an object, it is validated, and it is marshaled again
// when defaults were applied, so the bytes match what the SDK sends.
func (o Operation) EncodeOutput(out any) (json.RawMessage, error) {
	if o.outputZero != nil {
		if v := reflect.ValueOf(out); out == nil || (v.Kind() == reflect.Pointer && v.IsNil()) {
			out = o.outputZero
		}
	}
	if out == nil {
		return nil, nil
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("%s: marshaling output: %w", o.Name, err)
	}
	if o.outputResolved == nil {
		return data, nil
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("%s: unmarshaling output: %w", o.Name, err)
	}
	applied := false
	if _, ok := v.(map[string]any); ok {
		if err := o.outputResolved.ApplyDefaults(&v); err != nil {
			return nil, fmt.Errorf("%s: applying output schema defaults: %w", o.Name, err)
		}
		applied = true
	} else if v == nil && o.outputResolved.Schema().Type == "object" {
		v = map[string]any{}
		if err := o.outputResolved.ApplyDefaults(&v); err != nil {
			return nil, fmt.Errorf("%s: applying output schema defaults: %w", o.Name, err)
		}
		applied = true
	}
	if err := o.outputResolved.Validate(&v); err != nil {
		return nil, fmt.Errorf("%s: validating output: %w", o.Name, err)
	}
	if !applied {
		return data, nil
	}
	data, err = json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%s: marshaling output with defaults: %w", o.Name, err)
	}
	return data, nil
}

// InputSchema is the JSON schema of the operation's input. Callers must not
// modify it; it is shared by every copy of the Operation.
func (o Operation) InputSchema() *jsonschema.Schema { return o.inputSchema }

// OutputSchema is the JSON schema of the operation's output, or nil when Out
// is any. Callers must not modify it.
func (o Operation) OutputSchema() *jsonschema.Schema { return o.outputSchema }

type progressKey struct{}

// WithProgress returns a context whose Progress calls reach sink. Adapters
// install one to forward an operation's progress (MCP progress notifications,
// stderr lines).
func WithProgress(ctx context.Context, sink func(msg string)) context.Context {
	return context.WithValue(ctx, progressKey{}, sink)
}

// Progress reports msg to the sink installed by WithProgress, and does
// nothing when there is none.
func Progress(ctx context.Context, msg string) {
	if sink, ok := ctx.Value(progressKey{}).(func(string)); ok && sink != nil {
		sink(msg)
	}
}
