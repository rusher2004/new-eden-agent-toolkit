package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListMercenaryTacticalOperationsHandler_ReturnsOperations(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listMercenaryTacticalOperations
	t.Cleanup(func() { listMercenaryTacticalOperations = orig })
	listMercenaryTacticalOperations = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]OperationSummary, error) {
		return []OperationSummary{{ID: "op-1", MercenaryDenID: 1000}}, nil
	}

	handler := listMercenaryTacticalOperationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListMercenaryTacticalOperationsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Operations) != 1 || out.Operations[0].ID != "op-1" {
		t.Errorf("out = %+v, want CharacterID=7 with one operation ID \"op-1\"", out)
	}
}

func TestListMercenaryTacticalOperationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listMercenaryTacticalOperations
	t.Cleanup(func() { listMercenaryTacticalOperations = orig })
	listMercenaryTacticalOperations = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]OperationSummary, error) {
		t.Fatal("listMercenaryTacticalOperations should not be called for an unknown character")
		return nil, nil
	}

	handler := listMercenaryTacticalOperationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListMercenaryTacticalOperationsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetMercenaryTacticalOperationHandler_ReturnsDetail(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantExpires := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	orig := getMercenaryTacticalOperationDetail
	t.Cleanup(func() { getMercenaryTacticalOperationDetail = orig })
	getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
		return OperationDetailResult{
			DungeonTypeID:  30000,
			MercenaryDenID: 1000,
			State:          "active",
			Expires:        wantExpires,
		}, nil
	}

	handler := getMercenaryTacticalOperationHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 7, OperationID: "op-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.OperationID != "op-1" || out.DungeonTypeID != 30000 || out.State != "active" || !out.Expires.Equal(wantExpires) {
		t.Errorf("out = %+v, want OperationID=\"op-1\" DungeonTypeID=30000 State=\"active\" Expires=%v", out, wantExpires)
	}
}

func TestGetMercenaryTacticalOperationHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getMercenaryTacticalOperationDetail
	t.Cleanup(func() { getMercenaryTacticalOperationDetail = orig })
	getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
		t.Fatal("getMercenaryTacticalOperationDetail should not be called for an unknown character")
		return OperationDetailResult{}, nil
	}

	handler := getMercenaryTacticalOperationHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 999, OperationID: "op-1"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListRaidableSkyhooksHandler_ReturnsSkyhooks(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listRaidableSkyhooks
	t.Cleanup(func() { listRaidableSkyhooks = orig })
	listRaidableSkyhooks = func(ctx context.Context, userAgent string) ([]SkyhookSummary, error) {
		return []SkyhookSummary{{PlanetID: 40000001, SolarSystemID: 30000142}}, nil
	}

	handler := listRaidableSkyhooksHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListRaidableSkyhooksInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Skyhooks) != 1 || out.Skyhooks[0].PlanetID != 40000001 {
		t.Errorf("out = %+v, want one skyhook with PlanetID 40000001", out)
	}
}

func TestListRaidableSkyhooksHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listRaidableSkyhooks
	t.Cleanup(func() { listRaidableSkyhooks = orig })
	listRaidableSkyhooks = func(ctx context.Context, userAgent string) ([]SkyhookSummary, error) {
		t.Fatal("listRaidableSkyhooks should not be called for an unknown character")
		return nil, nil
	}

	handler := listRaidableSkyhooksHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListRaidableSkyhooksInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListMercenaryTacticalOperationsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listMercenaryTacticalOperations
	t.Cleanup(func() { listMercenaryTacticalOperations = orig })
	wantErr := errors.New("upstream failure")
	listMercenaryTacticalOperations = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]OperationSummary, error) {
		return nil, wantErr
	}

	handler := listMercenaryTacticalOperationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListMercenaryTacticalOperationsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetMercenaryTacticalOperationHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getMercenaryTacticalOperationDetail
	t.Cleanup(func() { getMercenaryTacticalOperationDetail = orig })
	wantErr := errors.New("upstream failure")
	getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
		return OperationDetailResult{}, wantErr
	}

	handler := getMercenaryTacticalOperationHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 7, OperationID: "op-1"})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListRaidableSkyhooksHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listRaidableSkyhooks
	t.Cleanup(func() { listRaidableSkyhooks = orig })
	wantErr := errors.New("upstream failure")
	listRaidableSkyhooks = func(ctx context.Context, userAgent string) ([]SkyhookSummary, error) {
		return nil, wantErr
	}

	handler := listRaidableSkyhooksHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListRaidableSkyhooksInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetMercenaryTacticalOperationHandler_ResolvesDungeonTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getMercenaryTacticalOperationDetail
	t.Cleanup(func() { getMercenaryTacticalOperationDetail = orig })
	getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
		return OperationDetailResult{DungeonTypeID: 34143, MercenaryDenID: 1}, nil
	}

	orig2 := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = orig2 })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{34143: "Some Dungeon"}
	}

	handler := getMercenaryTacticalOperationHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 7, OperationID: "op-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.DungeonTypeName != "Some Dungeon" {
		t.Errorf("DungeonTypeName = %q, want %q", out.DungeonTypeName, "Some Dungeon")
	}
}

func TestListRaidableSkyhooksHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listRaidableSkyhooks
	t.Cleanup(func() { listRaidableSkyhooks = orig })
	listRaidableSkyhooks = func(ctx context.Context, userAgent string) ([]SkyhookSummary, error) {
		return []SkyhookSummary{{PlanetID: 40009086, SolarSystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := listRaidableSkyhooksHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListRaidableSkyhooksInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Skyhooks[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Skyhooks[0].SolarSystemName, "Jita")
	}
	if out.Skyhooks[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Skyhooks[0].PlanetName, "Jita IV")
	}
}
