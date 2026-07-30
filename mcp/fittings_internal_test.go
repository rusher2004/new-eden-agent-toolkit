package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterFittingsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterFittings
	t.Cleanup(func() { listCharacterFittings = orig })
	listCharacterFittings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Fitting, error) {
		return []Fitting{{
			FittingID: 1, ShipTypeID: 600, Description: "PvP fit", Name: "My Fit",
			Items: []FittingItem{{Flag: "HiSlot0", Quantity: 1, TypeID: 2456}},
		}}, nil
	}

	handler := listCharacterFittingsHandler("v1", "client-id", store, nil)
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

	orig := listCharacterFittings
	t.Cleanup(func() { listCharacterFittings = orig })
	listCharacterFittings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Fitting, error) {
		t.Fatal("listCharacterFittings should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterFittingsHandler("v1", "client-id", store, nil)
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

	orig := listCharacterFittings
	t.Cleanup(func() { listCharacterFittings = orig })
	listCharacterFittings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Fitting, error) {
		return []Fitting{{
			FittingID: 1, ShipTypeID: 587,
			Items: []FittingItem{{Flag: "HiSlot0", Quantity: 1, TypeID: 588}},
		}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "150mm Railgun II"}
	}

	handler := listCharacterFittingsHandler("v1", "client-id", store, nil)
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

	orig := createCharacterFitting
	t.Cleanup(func() { createCharacterFitting = orig })

	var gotDescription, gotName string
	var gotShipTypeID int64
	var gotItems []FittingItemInput
	createCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, description, name string, shipTypeID int64, items []FittingItemInput) (int64, error) {
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

	handler := createCharacterFittingHandler("v1", "client-id", store)
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
		if it != input.Items[i] {
			t.Errorf("captured items[%d] = %+v, want %+v", i, it, input.Items[i])
		}
	}

	if out.CharacterID != input.CharacterID || out.FittingID != 12345 {
		t.Errorf("out = %+v, want CharacterID=%d FittingID=12345", out, input.CharacterID)
	}
}

func TestCreateCharacterFittingHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := createCharacterFitting
	t.Cleanup(func() { createCharacterFitting = orig })
	createCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, description, name string, shipTypeID int64, items []FittingItemInput) (int64, error) {
		t.Fatal("createCharacterFitting should not be called for an unknown character")
		return 0, nil
	}

	handler := createCharacterFittingHandler("v1", "client-id", store)
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

	orig := deleteCharacterFitting
	t.Cleanup(func() { deleteCharacterFitting = orig })
	deleteCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, fittingID int64) error {
		return nil
	}

	handler := deleteCharacterFittingHandler("v1", "client-id", store)
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

	orig := deleteCharacterFitting
	t.Cleanup(func() { deleteCharacterFitting = orig })
	deleteCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, fittingID int64) error {
		t.Fatal("deleteCharacterFitting should not be called for an unknown character")
		return nil
	}

	handler := deleteCharacterFittingHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, DeleteCharacterFittingInput{CharacterID: 999, FittingID: 12345})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
