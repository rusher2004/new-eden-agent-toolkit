package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func TestAddCharacterHandler_UpsertsIntoConfig(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := addCharacter
	t.Cleanup(func() { addCharacter = orig })
	addCharacter = func(ctx context.Context, clientID, userAgent string, scopes []string) (sso.Character, error) {
		return sso.Character{ID: 42, Name: "Test Pilot"}, nil
	}

	handler := addCharacterHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 42 || out.CharacterName != "Test Pilot" {
		t.Errorf("out = %+v, want {42 Test Pilot}", out)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0] != (config.Character{ID: 42, Name: "Test Pilot"}) {
		t.Errorf("cfg.Characters = %+v, want one entry {42 Test Pilot}", cfg.Characters)
	}
}

func TestAddCharacterHandler_ReAddSameCharacter_ReplacesNotDuplicates(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := addCharacter
	t.Cleanup(func() { addCharacter = orig })
	addCharacter = func(ctx context.Context, clientID, userAgent string, scopes []string) (sso.Character, error) {
		return sso.Character{ID: 42, Name: "Test Pilot"}, nil
	}

	handler := addCharacterHandler("v1", "client-id", store)
	if _, _, err := handler(context.Background(), nil, nil); err != nil {
		t.Fatalf("first handler call: %v", err)
	}

	addCharacter = func(ctx context.Context, clientID, userAgent string, scopes []string) (sso.Character, error) {
		return sso.Character{ID: 42, Name: "Renamed Pilot"}, nil
	}
	if _, _, err := handler(context.Background(), nil, nil); err != nil {
		t.Fatalf("second handler call: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0].Name != "Renamed Pilot" {
		t.Errorf("cfg.Characters = %+v, want exactly one updated entry", cfg.Characters)
	}
}

func TestAddCharacterHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := addCharacter
	t.Cleanup(func() { addCharacter = orig })
	wantErr := errors.New("upstream failure")
	addCharacter = func(ctx context.Context, clientID, userAgent string, scopes []string) (sso.Character, error) {
		return sso.Character{}, wantErr
	}

	handler := addCharacterHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
