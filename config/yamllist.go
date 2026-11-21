package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// Change is one value a write changed in one of neat's YAML files, so a tool
// can tell the user exactly what it recorded.
type Change struct {
	File  string `json:"file" jsonschema:"the file under ~/.neat/ that changed"`
	Entry string `json:"entry" jsonschema:"the entry's ID or name"`
	Field string `json:"field,omitempty" jsonschema:"the field that changed, as a dotted path; absent when a whole entry was added or removed"`
	Old   string `json:"old,omitempty"`
	New   string `json:"new,omitempty"`
}

// yamlList is a YAML file holding a list of entries of type T, each a
// mapping that one key identifies. The user can edit the file by hand, so
// writes patch the parsed document instead of re-rendering it: only the
// fields a change concerns are touched, and comments, ordering and keys neat
// doesn't know about all survive.
type yamlList[T any] struct {
	path   string
	header string // comment for the top of a new file, without "# "
	key    func(T) string
}

// fileLocks serializes read-modify-write cycles on each file within this
// process (the login page and tool calls can both write).
var fileLocks sync.Map

func lockFile(path string) func() {
	m, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (l yamlList[T]) name() string { return filepath.Base(l.path) }

// read returns the file's contents, or nil if it doesn't exist.
func (l yamlList[T]) read() ([]byte, error) {
	data, err := os.ReadFile(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", l.path, err)
	}
	return data, nil
}

// parseError describes a YAML error with its line, for the user to fix.
func (l yamlList[T]) parseError(err error) error {
	return fmt.Errorf("%s isn't valid YAML, so neat won't read or change it until it's fixed: %s", l.path, strings.TrimSpace(yaml.FormatError(err, false, true)))
}

// load reads every entry. A missing or empty file is an empty list.
func (l yamlList[T]) load() ([]T, error) {
	data, err := l.read()
	if err != nil || isBlank(data) {
		return nil, err
	}
	var out []T
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, l.parseError(err)
	}
	return out, nil
}

// isBlank reports whether data holds nothing but comments and whitespace.
func isBlank(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") {
			return false
		}
	}
	return true
}

// leadingComments returns the comment and blank lines at the top of a file.
func leadingComments(data []byte) string {
	var b strings.Builder
	for _, line := range strings.SplitAfter(string(data), "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") {
			break
		}
		b.WriteString(line)
	}
	return b.String()
}

// sync makes the file hold exactly entries, matched to the file's entries by
// key, and returns what it changed. An unchanged file isn't rewritten.
func (l yamlList[T]) sync(entries []T) ([]Change, error) {
	unlock := lockFile(l.path)
	defer unlock()

	data, err := l.read()
	if err != nil {
		return nil, err
	}
	if isBlank(data) {
		if len(entries) == 0 {
			return nil, nil
		}
		return l.rewrite(data, entries, nil)
	}
	f, err := parser.ParseBytes(data, parser.ParseComments)
	if err != nil {
		return nil, l.parseError(err)
	}
	var seq *ast.SequenceNode
	if len(f.Docs) == 1 {
		seq, _ = f.Docs[0].Body.(*ast.SequenceNode)
	}
	if seq == nil {
		return nil, fmt.Errorf("%s should hold a list of entries (each starting with \"- \"), so neat won't change it until it's fixed", l.path)
	}
	var old []T
	if err := yaml.Unmarshal(data, &old); err != nil {
		return nil, l.parseError(err)
	}
	if seq.IsFlowStyle || len(seq.Values) == 0 {
		// An empty or [flow] list has no block layout to patch into.
		return l.rewrite(data, entries, old)
	}

	at := map[string]int{}
	for i, e := range old {
		if _, dup := at[l.key(e)]; !dup {
			at[l.key(e)] = i
		}
	}
	want := map[string]bool{}
	var changes []Change
	var added []T
	for _, e := range entries {
		k := l.key(e)
		want[k] = true
		i, ok := at[k]
		if !ok {
			added = append(added, e)
			changes = append(changes, Change{File: l.name(), Entry: k, New: "added"})
			continue
		}
		c, err := l.patchEntry(f, i, k, old[i], e)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c...)
	}
	var removed []int
	for i, e := range old {
		if !want[l.key(e)] {
			removed = append(removed, i)
			changes = append(changes, Change{File: l.name(), Entry: l.key(e), Old: "removed"})
		}
	}
	if len(changes) == 0 {
		return nil, nil
	}
	if len(removed) == len(seq.Values) && len(added) == 0 {
		return changes, l.write([]byte(leadingComments(data) + "[]\n"))
	}
	if len(removed) == len(seq.Values) {
		if _, err := l.rewrite(data, added, nil); err != nil {
			return nil, err
		}
		return changes, nil
	}
	for j := len(removed) - 1; j >= 0; j-- {
		i := removed[j]
		seq.Values = append(seq.Values[:i], seq.Values[i+1:]...)
	}
	if len(added) > 0 {
		text, err := yaml.Marshal(added)
		if err != nil {
			return nil, fmt.Errorf("encode %s entries: %w", l.name(), err)
		}
		if err := mergeAt(f, "$", text); err != nil {
			return nil, fmt.Errorf("add %s entries: %w", l.name(), err)
		}
	}
	return changes, l.write([]byte(f.String()))
}

// rewrite renders the whole file from entries, keeping its top comment (or
// adding the default header to a new file). It's used only when there's no
// existing block list to patch.
func (l yamlList[T]) rewrite(data []byte, entries, old []T) ([]Change, error) {
	head := leadingComments(data)
	if data == nil && l.header != "" {
		for _, line := range strings.Split(l.header, "\n") {
			head += strings.TrimRight("# "+line, " ") + "\n"
		}
	}
	text, err := yaml.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", l.name(), err)
	}
	had := map[string]bool{}
	for _, e := range old {
		had[l.key(e)] = true
	}
	var changes []Change
	for _, e := range entries {
		if !had[l.key(e)] {
			changes = append(changes, Change{File: l.name(), Entry: l.key(e), New: "added"})
		}
	}
	return changes, l.write([]byte(head + string(text)))
}

func (l yamlList[T]) write(data []byte) error {
	if !bytes.HasSuffix(data, []byte("\n")) {
		data = append(data, '\n')
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return fmt.Errorf("create %s's directory: %w", l.name(), err)
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", l.name(), err)
	}
	if err := os.Rename(tmp, l.path); err != nil {
		return fmt.Errorf("move %s into place: %w", l.name(), err)
	}
	return nil
}

// patchEntry changes the fields of entry i that differ between what neat
// understood of it (old) and what it should now hold (now).
func (l yamlList[T]) patchEntry(f *ast.File, i int, key string, old, now T) ([]Change, error) {
	oldM, err := generic(old)
	if err != nil {
		return nil, err
	}
	newM, err := ordered(now)
	if err != nil {
		return nil, err
	}
	var changes []Change
	err = patchMapping(f, fmt.Sprintf("$[%d]", i), "", oldM, newM, func(field string, o, n any) {
		changes = append(changes, Change{File: l.name(), Entry: key, Field: field, Old: show(o), New: show(n)})
	})
	if err != nil {
		return nil, fmt.Errorf("update %s entry %s: %w", l.name(), key, err)
	}
	return changes, nil
}

// patchMapping patches the mapping at path from old to now. Keys only in
// the file (unknown to neat) are never in old, so they're never removed.
func patchMapping(f *ast.File, path, field string, old map[string]any, now yaml.MapSlice, report func(string, any, any)) error {
	node, err := filterPath(f, path)
	if err != nil {
		return err
	}
	inFile := map[string]bool{}
	for _, mv := range mappingValues(node) {
		inFile[mv.Key.String()] = true
	}
	seen := map[string]bool{}
	for _, item := range now {
		k := fmt.Sprint(item.Key)
		seen[k] = true
		sub, fieldName := path+"."+k, joinField(field, k)
		o, had := old[k]
		n := plain(item.Value)
		switch {
		case had && reflect.DeepEqual(o, n):
			continue
		case !had && !inFile[k]:
			text, err := yaml.Marshal(yaml.MapSlice{item})
			if err != nil {
				return err
			}
			if err := mergeAt(f, path, text); err != nil {
				return err
			}
			report(fieldName, nil, n)
		case had:
			om, oIsMap := o.(map[string]any)
			nm, nIsMap := item.Value.(yaml.MapSlice)
			// A flow mapping ({...}) can't take block-style keys merged
			// into it, so it's re-added whole too.
			if oIsMap && nIsMap && !removesKeys(om, nm) && !isFlowMapping(f, sub) {
				if err := patchMapping(f, sub, fieldName, om, nm, report); err != nil {
					return err
				}
				continue
			}
			if oIsMap || nIsMap {
				// A nested mapping losing keys, or changing kind, is
				// re-added whole rather than patched key by key.
				if err := deleteKey(f, path, k); err != nil {
					return err
				}
				text, err := yaml.Marshal(yaml.MapSlice{item})
				if err != nil {
					return err
				}
				if err := mergeAt(f, path, text); err != nil {
					return err
				}
				report(fieldName, o, n)
				continue
			}
			fallthrough
		default:
			if err := replaceAt(f, sub, item.Value); err != nil {
				return err
			}
			report(fieldName, o, n)
		}
	}
	var gone []string
	for k := range old {
		if !seen[k] {
			gone = append(gone, k)
		}
	}
	sort.Strings(gone)
	for _, k := range gone {
		if err := deleteKey(f, path, k); err != nil {
			return err
		}
		report(joinField(field, k), old[k], nil)
	}
	return nil
}

func removesKeys(old map[string]any, now yaml.MapSlice) bool {
	keep := map[string]bool{}
	for _, item := range now {
		keep[fmt.Sprint(item.Key)] = true
	}
	for k := range old {
		if !keep[k] {
			return true
		}
	}
	return false
}

func joinField(prefix, k string) string {
	if prefix == "" {
		return k
	}
	return prefix + "." + k
}

func filterPath(f *ast.File, path string) (ast.Node, error) {
	p, err := yaml.PathString(path)
	if err != nil {
		return nil, err
	}
	return p.FilterFile(f)
}

// isFlowMapping reports whether the node at path is a flow-style mapping,
// such as the {} an empty struct is written as.
func isFlowMapping(f *ast.File, path string) bool {
	node, err := filterPath(f, path)
	if err != nil {
		return false
	}
	m, ok := node.(*ast.MappingNode)
	return ok && m.IsFlowStyle
}

func mappingValues(n ast.Node) []*ast.MappingValueNode {
	switch m := n.(type) {
	case *ast.MappingNode:
		return m.Values
	case *ast.MappingValueNode:
		return []*ast.MappingValueNode{m}
	}
	return nil
}

func mergeAt(f *ast.File, path string, text []byte) error {
	p, err := yaml.PathString(path)
	if err != nil {
		return err
	}
	return p.MergeFromReader(f, bytes.NewReader(text))
}

// replaceAt swaps the value at path, keeping any comment on the old value.
func replaceAt(f *ast.File, path string, v any) error {
	p, err := yaml.PathString(path)
	if err != nil {
		return err
	}
	n, err := yaml.ValueToNode(v)
	if err != nil {
		return err
	}
	if old, err := p.FilterFile(f); err == nil && old.GetComment() != nil {
		if err := n.SetComment(old.GetComment()); err != nil {
			return err
		}
	}
	return p.ReplaceWithNode(f, n)
}

func deleteKey(f *ast.File, path, key string) error {
	node, err := filterPath(f, path)
	if err != nil {
		return err
	}
	m, ok := node.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("can't remove %s from %s: it isn't a mapping with several keys", key, path)
	}
	for i, mv := range m.Values {
		if mv.Key.String() == key {
			m.Values = append(m.Values[:i], m.Values[i+1:]...)
			return nil
		}
	}
	return nil
}

// generic is v as neat writes it, decoded back into plain maps and values:
// what neat understands of an entry, with unknown keys left out.
func generic(v any) (map[string]any, error) {
	text, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(text, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ordered is v as neat writes it, keeping field order for new keys.
func ordered(v any) (yaml.MapSlice, error) {
	text, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m yaml.MapSlice
	if err := yaml.UnmarshalWithOptions(text, &m, yaml.UseOrderedMap()); err != nil {
		return nil, err
	}
	return m, nil
}

// plain converts ordered maps back to plain ones, for comparing with generic.
func plain(v any) any {
	switch t := v.(type) {
	case yaml.MapSlice:
		m := make(map[string]any, len(t))
		for _, item := range t {
			m[fmt.Sprint(item.Key)] = plain(item.Value)
		}
		return m
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = plain(x)
		}
		return out
	}
	return v
}

// show renders a value for a Change: scalars as text, the rest as JSON.
func show(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case map[string]any, []any:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		return string(b)
	}
	return fmt.Sprint(v)
}
