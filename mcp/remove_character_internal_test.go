package mcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

func TestRemoveCharacterHandler_NotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := removeCharacterHandler(store)
	_, _, err := handler(context.Background(), nil, RemoveCharacterInput{CharacterID: 42})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRemoveCharacterHandler_RemovesConfigAndCreds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{
		{ID: 1, Name: "Alpha Pilot"},
		{ID: 2, Name: "Beta Pilot"},
	}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	if err := creds.SaveCharacterRefreshToken(1, "refresh-token-1"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	handler := removeCharacterHandler(store)
	_, out, err := handler(context.Background(), nil, RemoveCharacterInput{CharacterID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out != (CharacterInfo{CharacterID: 1, CharacterName: "Alpha Pilot"}) {
		t.Errorf("out = %+v, want {1 Alpha Pilot}", out)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0].ID != 2 {
		t.Errorf("cfg.Characters = %+v, want only character 2 remaining", cfg.Characters)
	}

	if _, err := creds.CharacterRefreshToken(1); !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("CharacterRefreshToken(1) err = %v, want wrapping creds.ErrNotFound", err)
	}
}

func TestRemoveCharacterHandler_RemovesCharacterCreds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 5, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	var removedID int32 = -1
	orig := removeCharacterCreds
	t.Cleanup(func() { removeCharacterCreds = orig })
	removeCharacterCreds = func(characterID int32) error { removedID = characterID; return nil }

	handler := removeCharacterHandler(store)
	if _, _, err := handler(context.Background(), nil, RemoveCharacterInput{CharacterID: 5}); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if removedID != 5 {
		t.Errorf("removedID = %d, want 5", removedID)
	}
}

// TestRemoveCharacterHandler_RemovesCredsEvenIfConfigSaveFails guards
// against a confirmed ordering bug: cache invalidation used to run only
// after store.Save succeeded, so a save failure (after the keychain
// credential was already deleted) left a stale cached client working
// indefinitely against a now-deleted credential. removeCharacterCreds now
// runs unconditionally before store.Save.
//
// Forces the Save half to fail by making the config's directory read-only
// after seeding a valid, readable config file in it: Load (a plain read)
// still succeeds, but Save's temp-file-then-rename can't create the new
// temp file. Skipped on Windows, where os.Chmod doesn't block directory
// writes the same way -- this is a test-portability limitation, not a
// platform difference in the code being tested.
func TestRemoveCharacterHandler_RemovesCredsEvenIfConfigSaveFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory-permission-based Save failure isn't portable to Windows")
	}
	keyring.MockInit()

	dir := t.TempDir()
	store := config.NewStoreAt(filepath.Join(dir, "config.json"))
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 5, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("seed store.Save: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	var removedID int32 = -1
	orig := removeCharacterCreds
	t.Cleanup(func() { removeCharacterCreds = orig })
	removeCharacterCreds = func(characterID int32) error { removedID = characterID; return nil }

	handler := removeCharacterHandler(store)
	_, _, err := handler(context.Background(), nil, RemoveCharacterInput{CharacterID: 5})
	if err == nil {
		t.Fatal("handler returned nil error, want a save failure")
	}
	if removedID != 5 {
		t.Errorf("removedID = %d, want 5 (credentials must be removed even though save failed)", removedID)
	}
}
