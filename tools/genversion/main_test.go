package main

import "testing"

func TestReplaceVersion(t *testing.T) {
	manifest := []byte(`{
  "name": "neat",
  "version": "0.1.0",
  "description": "x"
}`)

	out, changed, err := replaceVersion(manifest, "0.2.0")
	if err != nil {
		t.Fatalf("replaceVersion: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	want := `{
  "name": "neat",
  "version": "0.2.0",
  "description": "x"
}`
	if string(out) != want {
		t.Errorf("output mismatch:\n got: %s\nwant: %s", out, want)
	}
}

func TestReplaceVersionIdempotent(t *testing.T) {
	manifest := []byte(`{"version": "0.2.0"}`)
	out, changed, err := replaceVersion(manifest, "0.2.0")
	if err != nil {
		t.Fatalf("replaceVersion: %v", err)
	}
	if changed {
		t.Error("changed = true, want false when already in sync")
	}
	if string(out) != string(manifest) {
		t.Errorf("output altered when in sync: %s", out)
	}
}

func TestReplaceVersionRejectsZeroOrMultiple(t *testing.T) {
	if _, _, err := replaceVersion([]byte(`{"name": "x"}`), "0.2.0"); err == nil {
		t.Error("expected error when no version field present")
	}
	multi := []byte(`{"version": "0.1.0", "nested": {"version": "9.9.9"}}`)
	if _, _, err := replaceVersion(multi, "0.2.0"); err == nil {
		t.Error("expected error when more than one version field present (would clobber nested key)")
	}
}
