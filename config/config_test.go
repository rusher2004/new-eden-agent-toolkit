package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestStore_Update_ConcurrentChangesAreAllKept(t *testing.T) {
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))
	const n = 20
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.Update(func(c *config.Config) error {
				c.UpsertCharacter(config.Character{ID: int32(i + 1), Name: "c"})
				return nil
			}); err != nil {
				t.Errorf("Update: %v", err)
			}
		}()
	}
	wg.Wait()
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Characters) != n {
		t.Errorf("got %d characters, want %d: a concurrent Update lost another's change", len(cfg.Characters), n)
	}
}

func TestStore_Update_ErrorSavesNothing(t *testing.T) {
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))
	if _, err := store.Update(func(c *config.Config) error {
		c.UpsertCharacter(config.Character{ID: 1, Name: "c"})
		return errors.New("refused")
	}); err == nil {
		t.Fatal("Update returned nil, want the change's error")
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Characters) != 0 {
		t.Errorf("Characters = %v, want nothing saved", cfg.Characters)
	}
}

func TestStore_Load_MissingFileReturnsZeroValue(t *testing.T) {
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.Characters) != 0 {
		t.Errorf("Characters = %v, want empty", cfg.Characters)
	}
}

func TestStore_SaveThenLoad_RoundTrips(t *testing.T) {
	// Nested, not-yet-created directory: Save must create it.
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "nested", "config.json"))
	want := config.Config{
		Characters: []config.Character{{ID: 12345, Name: "Test Pilot"}},
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
