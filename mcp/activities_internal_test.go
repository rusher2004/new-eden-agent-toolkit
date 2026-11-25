package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// fakeOperationsESI fakes the mercenary tactical operation tools' ESI calls; err is returned by every call.
type fakeOperationsESI struct {
	ops    []esiapi.CharactersMercenaryTacticalOperationsListingOperation
	detail *esiapi.CharactersMercenaryTacticalOperationsDetail
	err    error
}

func (f fakeOperationsESI) MercenaryTacticalOperations(context.Context) ([]esiapi.CharactersMercenaryTacticalOperationsListingOperation, error) {
	return f.ops, f.err
}

func (f fakeOperationsESI) MercenaryTacticalOperationDetail(context.Context, string) (*esiapi.CharactersMercenaryTacticalOperationsDetail, error) {
	return f.detail, f.err
}

func openOperationsFake(c operationsESI) func(context.Context, int32) (operationsESI, error) {
	return func(context.Context, int32) (operationsESI, error) { return c, nil }
}

func openOperationsMustNotRun(t *testing.T) func(context.Context, int32) (operationsESI, error) {
	t.Helper()
	return func(context.Context, int32) (operationsESI, error) {
		t.Fatal("no ESI client should be opened for an unknown character")
		return nil, nil
	}
}

// fakeSkyhooksESI fakes list_raidable_skyhooks's one ESI call.
type fakeSkyhooksESI struct {
	skyhooks []esiapi.SkyhooksRaidableVulnerableskyhook
	err      error
}

func (f fakeSkyhooksESI) RaidableSkyhooks(context.Context) ([]esiapi.SkyhooksRaidableVulnerableskyhook, error) {
	return f.skyhooks, f.err
}

func openSkyhooksFake(c skyhooksESI) func() skyhooksESI { return func() skyhooksESI { return c } }

func TestListMercenaryTacticalOperationsHandler_ReturnsOperations(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeOperationsESI{ops: []esiapi.CharactersMercenaryTacticalOperationsListingOperation{{Id: "op-1", MercenaryDenId: 1000}}}

	handler := listMercenaryTacticalOperationsHandler(openOperationsFake(fake), store)
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

	handler := listMercenaryTacticalOperationsHandler(openOperationsMustNotRun(t), store)
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
	fake := fakeOperationsESI{detail: &esiapi.CharactersMercenaryTacticalOperationsDetail{
		DungeonTypeId:  30000,
		MercenaryDenId: 1000,
		State:          "active",
		Expires:        wantExpires,
	}}

	handler := getMercenaryTacticalOperationHandler("v1", openOperationsFake(fake), store, nil)
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

	handler := getMercenaryTacticalOperationHandler("v1", openOperationsMustNotRun(t), store, nil)
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

	fake := fakeSkyhooksESI{skyhooks: []esiapi.SkyhooksRaidableVulnerableskyhook{{PlanetId: 40000001, SolarSystemId: 30000142}}}

	handler := listRaidableSkyhooksHandler("v1", openSkyhooksFake(fake), store, nil)
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

	open := func() skyhooksESI {
		t.Fatal("no ESI client should be opened for an unknown character")
		return nil
	}

	handler := listRaidableSkyhooksHandler("v1", open, store, nil)
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

	wantErr := errors.New("upstream failure")

	handler := listMercenaryTacticalOperationsHandler(openOperationsFake(fakeOperationsESI{err: wantErr}), store)
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

	wantErr := errors.New("upstream failure")

	handler := getMercenaryTacticalOperationHandler("v1", openOperationsFake(fakeOperationsESI{err: wantErr}), store, nil)
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

	wantErr := errors.New("upstream failure")

	handler := listRaidableSkyhooksHandler("v1", openSkyhooksFake(fakeSkyhooksESI{err: wantErr}), store, nil)
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

	fake := fakeOperationsESI{detail: &esiapi.CharactersMercenaryTacticalOperationsDetail{DungeonTypeId: 34143, MercenaryDenId: 1}}

	orig2 := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = orig2 })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{34143: "Some Dungeon"}
	}

	handler := getMercenaryTacticalOperationHandler("v1", openOperationsFake(fake), store, nil)
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

	fake := fakeSkyhooksESI{skyhooks: []esiapi.SkyhooksRaidableVulnerableskyhook{{PlanetId: 40009086, SolarSystemId: 30000142}}}

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

	handler := listRaidableSkyhooksHandler("v1", openSkyhooksFake(fake), store, nil)
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

func TestMercenaryTacticalOperationHandlers_PropagateOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	open := func(context.Context, int32) (operationsESI, error) { return nil, errBoom }

	_, _, err := listMercenaryTacticalOperationsHandler(open, store)(context.Background(), nil, ListMercenaryTacticalOperationsInput{CharacterID: 7})
	if !errors.Is(err, errBoom) {
		t.Errorf("list: err = %v, want wrapping errBoom", err)
	}
	_, _, err = getMercenaryTacticalOperationHandler("v1", open, store, nil)(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 7, OperationID: "op-1"})
	if !errors.Is(err, errBoom) {
		t.Errorf("get: err = %v, want wrapping errBoom", err)
	}
}
