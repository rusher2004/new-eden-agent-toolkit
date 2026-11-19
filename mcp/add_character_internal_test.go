package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// fakeAddCharacters stands in for sso.AddCharacters: it "logs in" each of
// chars in turn through opts.OnAdded, as the real session does, and records
// what opts.SavedCharacters reported after the last one.
func fakeAddCharacters(t *testing.T, chars []sso.Character, savedAfter *[]sso.Character) {
	t.Helper()
	orig := addCharacters
	t.Cleanup(func() { addCharacters = orig })
	addCharacters = func(ctx context.Context, clientID, userAgent string, opts sso.AddOptions) ([]sso.Character, error) {
		for _, c := range chars {
			if err := opts.OnAdded(c); err != nil {
				return nil, err
			}
		}
		if savedAfter != nil {
			saved, err := opts.SavedCharacters()
			if err != nil {
				return nil, err
			}
			*savedAfter = saved
		}
		return chars, nil
	}
}

func TestAddCharacterHandler_SavesEachCharacterAndReturnsAll(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	chars := []sso.Character{{ID: 42, Name: "Test Pilot"}, {ID: 43, Name: "Alt Pilot"}}
	var saved []sso.Character
	fakeAddCharacters(t, chars, &saved)

	handler := addCharacterHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []AddedCharacter{{CharacterID: 42, CharacterName: "Test Pilot"}, {CharacterID: 43, CharacterName: "Alt Pilot"}}
	if !reflect.DeepEqual(out.Characters, want) {
		t.Errorf("out.Characters = %+v, want %+v", out.Characters, want)
	}
	if !reflect.DeepEqual(saved, chars) {
		t.Errorf("SavedCharacters during the session = %+v, want %+v", saved, chars)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	wantCfg := []config.Character{{ID: 42, Name: "Test Pilot"}, {ID: 43, Name: "Alt Pilot"}}
	if !reflect.DeepEqual(cfg.Characters, wantCfg) {
		t.Errorf("cfg.Characters = %+v, want %+v", cfg.Characters, wantCfg)
	}
}

func TestAddCharacterHandler_ReAddSameCharacter_ReplacesNotDuplicates(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	handler := addCharacterHandler("v1", "client-id", store)

	fakeAddCharacters(t, []sso.Character{{ID: 42, Name: "Test Pilot"}}, nil)
	if _, _, err := handler(context.Background(), nil, nil); err != nil {
		t.Fatalf("first handler call: %v", err)
	}

	fakeAddCharacters(t, []sso.Character{{ID: 42, Name: "Renamed Pilot"}}, nil)
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

	orig := addCharacters
	t.Cleanup(func() { addCharacters = orig })
	wantErr := errors.New("upstream failure")
	addCharacters = func(ctx context.Context, clientID, userAgent string, opts sso.AddOptions) ([]sso.Character, error) {
		return nil, wantErr
	}

	handler := addCharacterHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
