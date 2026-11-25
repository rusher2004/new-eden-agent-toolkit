package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterFittingsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fittingsFake{}
	fake.characterFittings = func(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error) {
		return []esiapi.CharactersCharacterIdFittingsGetInner{{
			FittingId: 1, ShipTypeId: 600, Description: "PvP fit", Name: "My Fit",
			Items: []esiapi.CharactersCharacterIdFittingsGetInnerItemsInner{{Flag: "HiSlot0", Quantity: 1, TypeId: 2456}},
		}}, nil
	}

	handler := listCharacterFittingsHandler("v1", fake.open, store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterFittingsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Fittings) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one fitting", out)
	}
	f := out.Fittings[0]
	if f.FittingID != 1 || f.ShipTypeID != 600 || f.Description != "PvP fit" || f.Name != "My Fit" || len(f.Items) != 1 {
		t.Fatalf("fitting = %+v, want FittingID=1 ShipTypeID=600 Description=%q Name=%q with one item", f, "PvP fit", "My Fit")
	}
	item := f.Items[0]
	if item.Flag != "HiSlot0" || item.Quantity != 1 || item.TypeID != 2456 {
		t.Errorf("item = %+v, want Flag=HiSlot0 Quantity=1 TypeID=2456", item)
	}
}

func TestListCharacterFittingsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fittingsFake{}
	fake.characterFittings = func(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error) {
		t.Fatal("listCharacterFittings should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterFittingsHandler("v1", fake.open, store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterFittingsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterFittingsHandler_ResolvesTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fittingsFake{}
	fake.characterFittings = func(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error) {
		return []esiapi.CharactersCharacterIdFittingsGetInner{{
			FittingId: 1, ShipTypeId: 587,
			Items: []esiapi.CharactersCharacterIdFittingsGetInnerItemsInner{{Flag: "HiSlot0", Quantity: 1, TypeId: 588}},
		}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "150mm Railgun II"}
	}

	handler := listCharacterFittingsHandler("v1", fake.open, store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterFittingsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Fittings[0].ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.Fittings[0].ShipTypeName, "Rifter")
	}
	if out.Fittings[0].Items[0].TypeName != "150mm Railgun II" {
		t.Errorf("Items[0].TypeName = %q, want %q", out.Fittings[0].Items[0].TypeName, "150mm Railgun II")
	}
}

func TestCreateCharacterFittingHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fittingsFake{}

	var gotDescription, gotName string
	var gotShipTypeID int64
	var gotItems []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner
	fake.createCharacterFitting = func(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error) {
		gotDescription = description
		gotName = name
		gotShipTypeID = shipTypeID
		gotItems = items
		return 12345, nil
	}

	input := CreateCharacterFittingInput{
		CharacterID: 7,
		Description: "PvP fit",
		Name:        "My Fit",
		ShipTypeID:  600,
		Items: []FittingItemInput{
			{Flag: "HiSlot0", Quantity: 1, TypeID: 2456},
		},
	}

	handler := createCharacterFittingHandler(fake.open, store)
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotDescription != input.Description || gotName != input.Name || gotShipTypeID != input.ShipTypeID {
		t.Errorf("captured description/name/shipTypeID = %q/%q/%d, want %q/%q/%d", gotDescription, gotName, gotShipTypeID, input.Description, input.Name, input.ShipTypeID)
	}
	if len(gotItems) != len(input.Items) {
		t.Fatalf("captured items = %+v, want %+v", gotItems, input.Items)
	}
	for i, it := range gotItems {
		want := input.Items[i]
		if it.Flag != want.Flag || it.Quantity != want.Quantity || it.TypeId != want.TypeID {
			t.Errorf("captured items[%d] = %+v, want %+v", i, it, want)
		}
	}

	if out.CharacterID != input.CharacterID || out.FittingID != 12345 {
		t.Errorf("out = %+v, want CharacterID=%d FittingID=12345", out, input.CharacterID)
	}
}

func TestCreateCharacterFittingHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fittingsFake{}
	fake.createCharacterFitting = func(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error) {
		t.Fatal("createCharacterFitting should not be called for an unknown character")
		return 0, nil
	}

	handler := createCharacterFittingHandler(fake.open, store)
	_, _, err := handler(context.Background(), nil, CreateCharacterFittingInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestDeleteCharacterFittingHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fittingsFake{}
	fake.deleteCharacterFitting = func(ctx context.Context, fittingID int64) error {
		return nil
	}

	handler := deleteCharacterFittingHandler(fake.open, store)
	_, out, err := handler(context.Background(), nil, DeleteCharacterFittingInput{CharacterID: 7, FittingID: 12345})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FittingID != 12345 {
		t.Errorf("out = %+v, want CharacterID=7 FittingID=12345", out)
	}
}

func TestDeleteCharacterFittingHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fittingsFake{}
	fake.deleteCharacterFitting = func(ctx context.Context, fittingID int64) error {
		t.Fatal("deleteCharacterFitting should not be called for an unknown character")
		return nil
	}

	handler := deleteCharacterFittingHandler(fake.open, store)
	_, _, err := handler(context.Background(), nil, DeleteCharacterFittingInput{CharacterID: 999, FittingID: 12345})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// TestFittingsHandlers_OpenFailure: every handler that opens an ESI client wraps and returns
// the opener's error, and never reaches an ESI call.
func TestFittingsHandlers_OpenFailure(t *testing.T) {
	store := newTestStoreWithCharacter(t)
	openErr := errors.New("keychain locked")
	fake := &fittingsFake{openErr: openErr}
	tests := []struct {
		name string
		call func() error
	}{
		{"list_character_fittings", func() error {
			return callHandlerForOpenFailure(listCharacterFittingsHandler("v1", fake.open, store, nil), ListCharacterFittingsInput{CharacterID: 7})
		}},
		{"create_character_fitting", func() error {
			return callHandlerForOpenFailure(createCharacterFittingHandler(fake.open, store), CreateCharacterFittingInput{CharacterID: 7})
		}},
		{"delete_character_fitting", func() error {
			return callHandlerForOpenFailure(deleteCharacterFittingHandler(fake.open, store), DeleteCharacterFittingInput{CharacterID: 7})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if !errors.Is(err, openErr) {
				t.Fatalf("err = %v, want wrapping %v", err, openErr)
			}
			if !strings.Contains(err.Error(), strings.ReplaceAll(tt.name, "_", " ")) {
				t.Errorf("err = %q, want it to start with the tool name %q", err, tt.name)
			}
		})
	}
}

// fittingsFake fakes fittingsESI: set a field to answer that call, and open records the character IDs it was asked for.
type fittingsFake struct {
	characterFittings      func(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error)
	createCharacterFitting func(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error)
	deleteCharacterFitting func(ctx context.Context, fittingID int64) error
	openErr                error
	openedFor              []int32
}

func (f *fittingsFake) open(ctx context.Context, characterID int32) (fittingsESI, error) {
	f.openedFor = append(f.openedFor, characterID)
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f, nil
}

func (f *fittingsFake) CharacterFittings(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error) {
	if f.characterFittings == nil {
		return nil, errors.New("fittingsFake: unexpected CharacterFittings call")
	}
	return f.characterFittings(ctx)
}

func (f *fittingsFake) CreateCharacterFitting(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error) {
	if f.createCharacterFitting == nil {
		return 0, errors.New("fittingsFake: unexpected CreateCharacterFitting call")
	}
	return f.createCharacterFitting(ctx, description, items, name, shipTypeID)
}

func (f *fittingsFake) DeleteCharacterFitting(ctx context.Context, fittingID int64) error {
	if f.deleteCharacterFitting == nil {
		return errors.New("fittingsFake: unexpected DeleteCharacterFitting call")
	}
	return f.deleteCharacterFitting(ctx, fittingID)
}
