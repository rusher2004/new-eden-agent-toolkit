package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharactersHandler_Empty(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharactersHandler(store)
	_, out, err := handler(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Characters == nil {
		t.Error("out.Characters is nil, want an empty (non-nil) slice")
	}
	if len(out.Characters) != 0 {
		t.Errorf("out.Characters = %+v, want empty", out.Characters)
	}
}

func TestListCharactersHandler_ReturnsKnownCharacters(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{
		{ID: 1, Name: "Alpha Pilot"},
		{ID: 2, Name: "Beta Pilot"},
	}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	handler := listCharactersHandler(store)
	_, out, err := handler(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []CharacterInfo{
		{CharacterID: 1, CharacterName: "Alpha Pilot"},
		{CharacterID: 2, CharacterName: "Beta Pilot"},
	}
	if len(out.Characters) != len(want) {
		t.Fatalf("out.Characters = %+v, want %+v", out.Characters, want)
	}
	for i := range want {
		if out.Characters[i] != want[i] {
			t.Errorf("out.Characters[%d] = %+v, want %+v", i, out.Characters[i], want[i])
		}
	}
}

func TestListCharactersHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	path := t.TempDir() + "/config.json"
	if err := os.WriteFile(path, []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	store := config.NewStoreAt(path)

	handler := listCharactersHandler(store)
	_, _, err := handler(context.Background(), nil, nil)
	if err == nil {
		t.Error("err = nil, want an error from a malformed config file")
	}
}
