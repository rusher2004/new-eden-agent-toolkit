package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestStore_Load_MissingFileReturnsZeroValue(t *testing.T) {
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.Characters) != 0 {
		t.Errorf("Characters = %v, want empty", cfg.Characters)
	}
	if cfg.SDEVersion != "" {
		t.Errorf("SDEVersion = %q, want empty", cfg.SDEVersion)
	}
}

func TestStore_SaveThenLoad_RoundTrips(t *testing.T) {
	// Nested, not-yet-created directory: Save must create it.
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "nested", "config.json"))
	want := config.Config{
		Characters: []config.Character{{ID: 12345, Name: "Test Pilot"}},
		SDEVersion: "sde-20250303",
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got.Characters) != 1 || got.Characters[0] != want.Characters[0] {
		t.Errorf("Characters = %v, want %v", got.Characters, want.Characters)
	}
	if got.SDEVersion != want.SDEVersion {
		t.Errorf("SDEVersion = %q, want %q", got.SDEVersion, want.SDEVersion)
	}
}

func TestStore_Load_MalformedJSONReturnsWrappedError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	store := config.NewStoreAt(path)

	_, err := store.Load()
	if err == nil {
		t.Fatal("Load returned nil error, want a parse error for malformed JSON")
	}
}

func TestStore_Path_ReturnsConfiguredPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := config.NewStoreAt(path)

	if got := store.Path(); got != path {
		t.Errorf("Path() = %q, want %q", got, path)
	}
}
