package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestSearchCharacterHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	var gotCategories []string
	var gotSearch string
	var gotStrict bool
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		gotCategories, gotSearch, gotStrict = categories, search, strict
		return &esiapi.CharactersCharacterIdSearchGet{
			Character:   []int64{90000001, 90000002},
			Corporation: []int64{98000001},
		}, nil
	}

	handler := searchCharacterHandler("v1", "client-id", store, nil)
	input := SearchCharacterInput{CharacterID: 7, Categories: []string{"character", "corporation"}, Search: "wanted", Strict: true}
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if len(gotCategories) != 2 || gotCategories[0] != "character" || gotCategories[1] != "corporation" {
		t.Errorf("captured categories = %+v, want [character corporation]", gotCategories)
	}
	if gotSearch != "wanted" {
		t.Errorf("captured search = %q, want \"wanted\"", gotSearch)
	}
	if !gotStrict {
		t.Error("captured strict = false, want true")
	}

	if out.CharacterID != 7 {
		t.Errorf("out.CharacterID = %d, want 7", out.CharacterID)
	}
	if len(out.Results.Character) != 2 || out.Results.Character[0] != 90000001 || out.Results.Character[1] != 90000002 {
		t.Errorf("out.Results.Character = %+v, want [90000001 90000002]", out.Results.Character)
	}
	if len(out.Results.Corporation) != 1 || out.Results.Corporation[0] != 98000001 {
		t.Errorf("out.Results.Corporation = %+v, want [98000001]", out.Results.Corporation)
	}
	if len(out.Results.Agent) != 0 || len(out.Results.Alliance) != 0 || len(out.Results.Constellation) != 0 ||
		len(out.Results.Faction) != 0 || len(out.Results.InventoryType) != 0 || len(out.Results.Region) != 0 ||
		len(out.Results.SolarSystem) != 0 || len(out.Results.Station) != 0 || len(out.Results.Structure) != 0 {
		t.Errorf("out.Results = %+v, want all other category fields empty", out.Results)
	}
}

func TestSearchCharacterHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		t.Fatal("searchCharacter should not be called for an unknown character")
		return nil, nil
	}

	handler := searchCharacterHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, SearchCharacterInput{CharacterID: 999, Categories: []string{"character"}, Search: "wanted"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestSearchCharacterHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	wantErr := errors.New("upstream failure")
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		return nil, wantErr
	}

	handler := searchCharacterHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, SearchCharacterInput{CharacterID: 7, Categories: []string{"character"}, Search: "wanted"})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestSearchCharacterHandler_ResolvesInventoryTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 1001, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		return &esiapi.CharactersCharacterIdSearchGet{InventoryType: []int64{587, 588}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := searchCharacterHandler("v1", "client", store, nil)
	_, out, err := handler(context.Background(), nil, SearchCharacterInput{CharacterID: 1001, Categories: []string{"inventory_type"}, Search: "rifter"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Results.InventoryTypeNames) != 2 || out.Results.InventoryTypeNames[0] != "Rifter" || out.Results.InventoryTypeNames[1] != "Slasher" {
		t.Errorf("InventoryTypeNames = %v, want [Rifter Slasher]", out.Results.InventoryTypeNames)
	}
}

func TestSearchCharacterHandler_InventoryTypeNamesSurvivesPartialMiss(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 1001, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		return &esiapi.CharactersCharacterIdSearchGet{InventoryType: []int64{587, 999, 588}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		// 999 is deliberately omitted -- a cache miss on the middle ID must
		// not shift the surrounding names out of position.
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := searchCharacterHandler("v1", "client", store, nil)
	_, out, err := handler(context.Background(), nil, SearchCharacterInput{CharacterID: 1001, Categories: []string{"inventory_type"}, Search: "rifter"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []string{"Rifter", "", "Slasher"}
	if len(out.Results.InventoryTypeNames) != len(want) {
		t.Fatalf("InventoryTypeNames = %v, want %v", out.Results.InventoryTypeNames, want)
	}
	for i, w := range want {
		if out.Results.InventoryTypeNames[i] != w {
			t.Errorf("InventoryTypeNames[%d] = %q, want %q", i, out.Results.InventoryTypeNames[i], w)
		}
	}
}
