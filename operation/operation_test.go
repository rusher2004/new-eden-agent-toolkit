package operation_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

type doubleIn struct {
	A int `json:"a"`
}

type doubleOut struct {
	B int `json:"b"`
}

func newDoubleOp() operation.Operation {
	return operation.New("double", "doubles a", operation.Read,
		func(_ context.Context, in doubleIn) (doubleOut, error) {
			return doubleOut{B: in.A * 2}, nil
		})
}

func TestNew_RunsTypedFunction(t *testing.T) {
	op := newDoubleOp()
	in, err := op.Decode([]byte(`{"a":2}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	got, err := op.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	out, ok := got.(doubleOut)
	if !ok {
		t.Fatalf("Run returned %T, want doubleOut", got)
	}
	if out.B != 4 {
		t.Errorf("B = %d, want 4 (double of 2)", out.B)
	}
	if op.Name != "double" || op.Description != "doubles a" || op.Kind != operation.Read {
		t.Errorf("metadata = %q %q %v", op.Name, op.Description, op.Kind)
	}
}

func TestNew_EmptyInputDecodesAsEmptyObject(t *testing.T) {
	type optIn struct {
		A int `json:"a,omitempty"`
	}
	op := operation.New("opt", "", operation.Read,
		func(_ context.Context, in optIn) (doubleOut, error) { return doubleOut{B: in.A}, nil })
	for _, raw := range [][]byte{nil, {}} {
		in, err := op.Decode(raw)
		if err != nil {
			t.Fatalf("Decode(%q): %v", raw, err)
		}
		if _, err := op.Run(context.Background(), in); err != nil {
			t.Fatalf("Run: %v", err)
		}
	}
}

func TestDecode_RejectsInvalidInput(t *testing.T) {
	op := newDoubleOp()
	cases := map[string]string{
		"unknown field":  `{"a":1,"c":2}`,
		"wrong type":     `{"a":"two"}`,
		"missing field":  `{}`,
		"malformed JSON": `{"a":`,
		"not an object":  `[1]`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := op.Decode([]byte(raw))
			if !errors.Is(err, operation.ErrInvalidInput) {
				t.Fatalf("Decode(%s) error = %v, want ErrInvalidInput", raw, err)
			}
		})
	}
}

func TestSchemas_UseJSONAndJSONSchemaTags(t *testing.T) {
	type in struct {
		CharacterID int32  `json:"characterId" jsonschema:"the character"`
		Note        string `json:"note,omitempty"`
	}
	type out struct {
		Name string `json:"name"`
	}
	op := operation.New("tagged", "", operation.Write,
		func(context.Context, in) (out, error) { return out{}, nil })

	s := op.InputSchema()
	prop, ok := s.Properties["characterId"]
	if !ok {
		t.Fatalf("no characterId property in %v", s.Properties)
	}
	if prop.Description != "the character" {
		t.Errorf("description = %q, want %q", prop.Description, "the character")
	}
	if !slices.Contains(s.Required, "characterId") {
		t.Errorf("required = %v, want characterId in it", s.Required)
	}
	if slices.Contains(s.Required, "note") {
		t.Errorf("required = %v, omitempty note must not be required", s.Required)
	}
	if _, ok := op.OutputSchema().Properties["name"]; !ok {
		t.Errorf("output schema has no name property: %v", op.OutputSchema().Properties)
	}
}

func TestRun_WrongInputTypeIsError(t *testing.T) {
	op := newDoubleOp()
	for _, bad := range []any{nil, doubleIn{A: 1}, "x"} {
		if _, err := op.Run(context.Background(), bad); err == nil {
			t.Errorf("Run(%#v) returned no error", bad)
		}
	}
}

func TestProgress_ReachesSink(t *testing.T) {
	var got []string
	ctx := operation.WithProgress(context.Background(), func(msg string) { got = append(got, msg) })
	operation.Progress(ctx, "one")
	operation.Progress(ctx, "two")
	if !slices.Equal(got, []string{"one", "two"}) {
		t.Errorf("sink got %v, want [one two]", got)
	}
	// Without a sink, Progress is a no-op and must not panic.
	operation.Progress(context.Background(), "ignored")
}

func TestEncodeOutput_MarshalsOutput(t *testing.T) {
	op := newDoubleOp()
	got, err := op.EncodeOutput(doubleOut{B: 3})
	if err != nil {
		t.Fatalf("EncodeOutput: %v", err)
	}
	if string(got) != `{"b":3}` {
		t.Errorf("EncodeOutput = %s, want {\"b\":3}", got)
	}
}

func TestEncodeOutput_RejectsOutputNotMatchingSchema(t *testing.T) {
	op := newDoubleOp()
	// A value of the wrong type marshals to JSON the schema rejects (b missing).
	if _, err := op.EncodeOutput(struct {
		C int `json:"c"`
	}{C: 1}); err == nil {
		t.Fatal("EncodeOutput of a value lacking required b returned no error")
	}
}

func TestEncodeOutput_TypedNilPointerIsElementZero(t *testing.T) {
	op := operation.New("ptr", "", operation.Read,
		func(context.Context, doubleIn) (*doubleOut, error) { return nil, nil })
	got, err := op.EncodeOutput((*doubleOut)(nil))
	if err != nil {
		t.Fatalf("EncodeOutput: %v", err)
	}
	if string(got) != `{"b":0}` {
		t.Errorf("EncodeOutput(nil *doubleOut) = %s, want {\"b\":0}", got)
	}
}

func TestEncodeOutput_AnyOutHasNoSchema(t *testing.T) {
	op := operation.New("anyout", "", operation.Read,
		func(context.Context, doubleIn) (any, error) { return nil, nil })
	got, err := op.EncodeOutput([]int{1, 2})
	if err != nil {
		t.Fatalf("EncodeOutput: %v", err)
	}
	if string(got) != `[1,2]` {
		t.Errorf("EncodeOutput = %s, want [1,2]", got)
	}
	if got, err := op.EncodeOutput(nil); err != nil || got != nil {
		t.Errorf("EncodeOutput(nil) = %s, %v; want nil, nil", got, err)
	}
}
