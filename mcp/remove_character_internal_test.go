package mcp

import (
	"context"
	"errors"
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
