package commands_test

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

func TestOperations_ZeroDepsDoNotTouchSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ops := commands.Catalog()

	var names []string
	for _, op := range ops {
		names = append(names, op.Name)
	}
	slices.Sort(names)
	want := []string{"list_character_loyalty_points", "list_loyalty_store_offers"}
	if !slices.Equal(names, want) {
		t.Errorf("operation names = %v, want %v", names, want)
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatalf("read HOME: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("building the operations created %d entries under HOME (first %q), want none", len(entries), entries[0].Name())
	}
}

// TestOperationTypesHaveNoESIModels keeps ESI's generated models out of every
// operation's input and output, so the contract adapters see never depends on
// goesi-openapi.
func TestOperationTypesHaveNoESIModels(t *testing.T) {
	for _, op := range commands.Catalog() {
		for _, rt := range []reflect.Type{op.InputType, op.OutputType} {
			if bad := esiModelIn(rt, map[reflect.Type]bool{}); bad != nil {
				t.Errorf("%s: %v contains ESI model %v (%s)", op.Name, rt, bad, bad.PkgPath())
			}
		}
	}
}

// esiModelIn returns the first type reachable from rt whose package is
// goesi-openapi, or nil.
func esiModelIn(rt reflect.Type, seen map[reflect.Type]bool) reflect.Type {
	if rt == nil || seen[rt] {
		return nil
	}
	seen[rt] = true
	if strings.Contains(rt.PkgPath(), "goesi-openapi") {
		return rt
	}
	switch rt.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return esiModelIn(rt.Elem(), seen)
	case reflect.Map:
		if bad := esiModelIn(rt.Key(), seen); bad != nil {
			return bad
		}
		return esiModelIn(rt.Elem(), seen)
	case reflect.Struct:
		for field := range rt.Fields() {
			if bad := esiModelIn(field.Type, seen); bad != nil {
				return bad
			}
		}
	}
	return nil
}
