package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type testPlace struct {
	ID    int64      `yaml:"id"`
	Name  string     `yaml:"name"`
	Type  *testType  `yaml:"type,omitempty"`
	Uses  []string   `yaml:"uses,omitempty"`
	Bay   *testBay   `yaml:"clone_bay,omitempty"`
	Notes string     `yaml:"notes,omitempty"`
	Docks []testDock `yaml:"docks,omitempty"`
}

type testType struct {
	ID   int64  `yaml:"id"`
	Name string `yaml:"name"`
}

type testBay struct {
	Status  string `yaml:"status"`
	Source  string `yaml:"source,omitempty"`
	Checked string `yaml:"checked,omitempty"`
}

type testDock struct {
	ID   int64  `yaml:"id"`
	Name string `yaml:"name"`
}

const handEdited = `# Places I use. The user wrote this comment.
- id: 1
  name: CX7-70 Sotiyo # refreshed from ESI
  type:
    id: 35827
    name: Sotiyo
  uses:
    - manufacturing
  my_own_key: the user added this
  notes: |
    Hand-written notes
    on two lines.

# Between entries, also the user's.
- id: 2
  name: Second
`

func placeList(t *testing.T, content string) yamlList[testPlace] {
	t.Helper()
	path := filepath.Join(t.TempDir(), "places.yaml")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return yamlList[testPlace]{path: path, header: "Places neat knows.", key: func(p testPlace) string { return strconv.FormatInt(p.ID, 10) }}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestYAMLList_PatchKeepsCommentsOrderAndUnknownKeys(t *testing.T) {
	l := placeList(t, handEdited)
	entries, err := l.load()
	if err != nil {
		t.Fatal(err)
	}
	entries[0].Name = "CX7-70 Renamed"
	entries[0].Type.ID, entries[0].Type.Name = 35834, "Keepstar"
	entries[0].Bay = &testBay{Status: "yes", Source: "jump_clone", Checked: "2026-11-20"}

	changes, err := l.sync(entries)
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, l.path)
	for _, want := range []string{
		"# Places I use. The user wrote this comment.\n",
		"  name: CX7-70 Renamed # refreshed from ESI\n",
		"  type:\n    id: 35834\n    name: Keepstar\n",
		"  my_own_key: the user added this\n",
		"  notes: |\n    Hand-written notes\n    on two lines.\n",
		"# Between entries, also the user's.\n",
		"  clone_bay:\n    status: \"yes\"\n    source: jump_clone\n    checked: \"2026-11-20\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("file is missing %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "my_own_key") > strings.Index(got, "notes:") {
		t.Errorf("key order changed:\n%s", got)
	}
	want := map[string]bool{"name": true, "type.id": true, "type.name": true, "clone_bay": true}
	for _, c := range changes {
		if c.Entry != "1" || !want[c.Field] {
			t.Errorf("unexpected change %+v", c)
		}
		delete(want, c.Field)
	}
	if len(want) != 0 {
		t.Errorf("changes missing %v: %+v", want, changes)
	}
	for _, c := range changes {
		if c.Field == "name" && (c.Old != "CX7-70 Sotiyo" || c.New != "CX7-70 Renamed") {
			t.Errorf("name change = %+v", c)
		}
	}
}

func TestYAMLList_NoChangeDoesntRewrite(t *testing.T) {
	l := placeList(t, handEdited)
	entries, _ := l.load()
	changes, err := l.sync(entries)
	if err != nil || len(changes) != 0 {
		t.Fatalf("changes = %+v, %v", changes, err)
	}
	if got := readFile(t, l.path); got != handEdited {
		t.Errorf("an unchanged file was rewritten:\n%s", got)
	}
}

func TestYAMLList_RemovesOnlyFieldsNeatClears(t *testing.T) {
	l := placeList(t, handEdited)
	entries, _ := l.load()
	entries[0].Notes = ""
	entries[0].Uses = nil
	changes, err := l.sync(entries)
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, l.path)
	if strings.Contains(got, "notes:") || strings.Contains(got, "uses:") || !strings.Contains(got, "my_own_key") {
		t.Errorf("file:\n%s", got)
	}
	if len(changes) != 2 || changes[0].Field != "notes" || changes[1].Field != "uses" || changes[0].New != "" {
		t.Errorf("changes = %+v", changes)
	}
}

func TestYAMLList_NestedKeyRemovedRewritesThatMapping(t *testing.T) {
	l := placeList(t, "- id: 1\n  name: A\n  clone_bay:\n    status: yes\n    source: user\n    checked: \"2026-11-01\"\n  notes: keep\n")
	entries, _ := l.load()
	entries[0].Bay = &testBay{Status: "unknown"}
	if _, err := l.sync(entries); err != nil {
		t.Fatal(err)
	}
	back, err := l.load()
	if err != nil {
		t.Fatal(err)
	}
	if back[0].Bay == nil || *back[0].Bay != (testBay{Status: "unknown"}) || back[0].Notes != "keep" {
		t.Errorf("after sync: %+v %+v\n%s", back[0], back[0].Bay, readFile(t, l.path))
	}
}

func TestYAMLList_AddAndRemoveEntries(t *testing.T) {
	l := placeList(t, handEdited)
	entries, _ := l.load()
	entries = append(entries[:1], testPlace{ID: 3, Name: "Third", Docks: []testDock{{1, "a"}, {2, "b"}}})
	changes, err := l.sync(entries)
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, l.path)
	if strings.Contains(got, "Second") || !strings.Contains(got, "- id: 3\n  name: Third\n  docks:\n") || !strings.Contains(got, "my_own_key") {
		t.Errorf("file:\n%s", got)
	}
	if len(changes) != 2 || changes[0] != (Change{File: "places.yaml", Entry: "3", New: "added"}) || changes[1] != (Change{File: "places.yaml", Entry: "2", Old: "removed"}) {
		t.Errorf("changes = %+v", changes)
	}
	back, err := l.load()
	if err != nil || len(back) != 2 || len(back[1].Docks) != 2 {
		t.Errorf("reload = %+v, %v", back, err)
	}
}

func TestYAMLList_NewFileGetsHeader(t *testing.T) {
	l := placeList(t, "")
	if changes, err := l.sync(nil); err != nil || changes != nil {
		t.Fatalf("syncing nothing = %+v, %v", changes, err)
	}
	if _, err := os.Stat(l.path); !os.IsNotExist(err) {
		t.Error("syncing nothing created the file")
	}
	if _, err := l.sync([]testPlace{{ID: 1, Name: "A"}}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, l.path); got != "# Places neat knows.\n- id: 1\n  name: A\n" {
		t.Errorf("new file = %q", got)
	}
}

func TestYAMLList_EmptyFlowListAndRemovingEverything(t *testing.T) {
	l := placeList(t, "# My places\n[]\n")
	if _, err := l.sync([]testPlace{{ID: 1, Name: "A"}}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, l.path); got != "# My places\n- id: 1\n  name: A\n" {
		t.Errorf("after adding to []: %q", got)
	}
	if _, err := l.sync(nil); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, l.path); got != "# My places\n[]\n" {
		t.Errorf("after removing everything: %q", got)
	}
}

func TestYAMLList_BrokenFileIsNeverWritten(t *testing.T) {
	broken := "- id: 1\n  name: [unclosed\n"
	l := placeList(t, broken)
	if _, err := l.load(); err == nil || !strings.Contains(err.Error(), "isn't valid YAML") {
		t.Errorf("load error = %v", err)
	}
	if _, err := l.sync([]testPlace{{ID: 1, Name: "A"}}); err == nil {
		t.Error("sync of a broken file: want an error")
	}
	if got := readFile(t, l.path); got != broken {
		t.Errorf("a broken file was changed: %q", got)
	}
}

func TestYAMLList_NotAListIsRefused(t *testing.T) {
	l := placeList(t, "id: 1\nname: A\n")
	if _, err := l.sync([]testPlace{{ID: 1, Name: "B"}}); err == nil || !strings.Contains(err.Error(), "should hold a list") {
		t.Errorf("err = %v", err)
	}
}

func TestYAMLList_HandEditBetweenReadAndWriteSurvives(t *testing.T) {
	l := placeList(t, handEdited)
	entries, _ := l.load()
	// The user edits the file after neat read it.
	edited := strings.Replace(handEdited, "my_own_key: the user added this", "my_own_key: edited meanwhile", 1)
	if err := os.WriteFile(l.path, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	entries[1].Name = "Second renamed"
	if _, err := l.sync(entries); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, l.path)
	if !strings.Contains(got, "edited meanwhile") || !strings.Contains(got, "Second renamed") {
		t.Errorf("file:\n%s", got)
	}
}
