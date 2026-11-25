package cli

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"

	urfave "github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

// Flags every leaf has, which no input field may take.
const (
	jsonFlag   = "json"
	prettyFlag = "pretty"
)

var reservedFlags = []string{jsonFlag, prettyFlag, "help", "h"}

// inputFlag is one flag derived from a field of an operation's input.
type inputFlag struct {
	name     string // flag name, without dashes
	jsonName string // the field's key in the input object
	flag     urfave.Flag
}

// inputFlags derives a flag for each scalar or scalar-slice field of op's
// input struct. Other fields get no flag; they are reachable through --json.
func inputFlags(op operation.Operation) ([]inputFlag, error) {
	rt := op.InputType
	if rt != nil && rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil, nil
	}
	schema := op.InputSchema()
	var flags []inputFlag
	owner := map[string]string{} // flag name -> JSON name that produced it
	var walk func(rt reflect.Type) error
	walk = func(rt reflect.Type) error {
		for field := range rt.Fields() {
			jsonName, ok := jsonNameOf(field)
			if !ok {
				continue
			}
			ft := field.Type
			if field.Anonymous && jsonName == "" {
				if ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					// encoding/json promotes an untagged embedded struct's
					// fields into the outer object.
					if err := walk(ft); err != nil {
						return err
					}
					continue
				}
			}
			if jsonName == "" {
				jsonName = field.Name
			}
			name, usage := flagName(jsonName, ft), ""
			if schema != nil {
				if prop := schema.Properties[jsonName]; prop != nil {
					usage = prop.Description
				}
				if slices.Contains(schema.Required, jsonName) {
					usage = strings.TrimSpace(usage + " (required)")
				}
			}
			f := flagFor(name, usage, ft)
			if f == nil {
				continue
			}
			if slices.Contains(reservedFlags, name) {
				return fmt.Errorf("operation %q: input field %q would take the reserved flag --%s", op.Name, jsonName, name)
			}
			if other, dup := owner[name]; dup {
				return fmt.Errorf("operation %q: input fields %q and %q both become flag --%s", op.Name, other, jsonName, name)
			}
			owner[name] = jsonName
			flags = append(flags, inputFlag{name: name, jsonName: jsonName, flag: f})
		}
		return nil
	}
	if err := walk(rt); err != nil {
		return nil, err
	}
	return flags, nil
}

// jsonNameOf reads the field's json tag name. ok is false for a field
// encoding/json skips (unexported, or tagged "-"); name is "" when the tag
// gives none.
func jsonNameOf(field reflect.StructField) (name string, ok bool) {
	if !field.IsExported() && !field.Anonymous {
		return "", false
	}
	tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if tag == "-" {
		return "", false
	}
	return tag, true
}

// flagName kebab-cases jsonName; a scalar's trailing Id or ID is dropped
// (characterId -> character, accessListId -> access-list), a slice keeps its
// name (types -> types).
func flagName(jsonName string, ft reflect.Type) string {
	if ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	if ft.Kind() != reflect.Slice {
		for _, suffix := range []string{"Id", "ID"} {
			if trimmed, ok := strings.CutSuffix(jsonName, suffix); ok && trimmed != "" {
				jsonName = trimmed
				break
			}
		}
	}
	return kebab(jsonName)
}

// kebab turns camelCase (and acronyms, as in accessListID) into kebab-case.
func kebab(s string) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		if unicode.IsUpper(r) && i > 0 {
			prev := rs[i-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				b.WriteByte('-')
			}
		}
		if r == '_' {
			b.WriteByte('-')
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// flagFor returns the urfave flag for a field of type ft, or nil when ft is
// not a supported scalar (int32, int64, string, bool, float64, or a pointer to
// one) or a slice of int32, int64, string or float64.
func flagFor(name, usage string, ft reflect.Type) urfave.Flag {
	if ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	switch ft.Kind() {
	case reflect.Int32:
		return &urfave.Int32Flag{Name: name, Usage: usage, HideDefault: true}
	case reflect.Int64:
		return &urfave.Int64Flag{Name: name, Usage: usage, HideDefault: true}
	case reflect.String:
		return &urfave.StringFlag{Name: name, Usage: usage, HideDefault: true}
	case reflect.Bool:
		return &urfave.BoolFlag{Name: name, Usage: usage, HideDefault: true}
	case reflect.Float64:
		return &urfave.Float64Flag{Name: name, Usage: usage, HideDefault: true}
	case reflect.Slice:
		switch ft.Elem().Kind() {
		case reflect.Int32:
			return &urfave.Int32SliceFlag{Name: name, Usage: usage, HideDefault: true}
		case reflect.Int64:
			return &urfave.Int64SliceFlag{Name: name, Usage: usage, HideDefault: true}
		case reflect.String:
			return &urfave.StringSliceFlag{Name: name, Usage: usage, HideDefault: true}
		case reflect.Float64:
			return &urfave.Float64SliceFlag{Name: name, Usage: usage, HideDefault: true}
		}
	}
	return nil
}
