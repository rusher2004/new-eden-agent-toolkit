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

// fakeStructuresESI fakes every ESI call the structure tools make; a test sets
// the function for the call it exercises.
type fakeStructuresESI struct {
	listCharacterMercenaryDens     func(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error)
	getCharacterMercenaryDen       func(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error)
	listCorporationSkyhooks        func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error)
	getCorporationSkyhook          func(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error)
	listCorporationSovereigntyHubs func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error)
	getCorporationSovereigntyHub   func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error)
}

func (f *fakeStructuresESI) CharacterMercenaryDens(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
	return f.listCharacterMercenaryDens(ctx)
}
func (f *fakeStructuresESI) CharacterMercenaryDenDetail(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
	return f.getCharacterMercenaryDen(ctx, mercenaryDenID)
}
func (f *fakeStructuresESI) CorporationSkyhooks(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
	return f.listCorporationSkyhooks(ctx, corporationID)
}
func (f *fakeStructuresESI) CorporationSkyhookDetail(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
	return f.getCorporationSkyhook(ctx, corporationID, skyhookID)
}
func (f *fakeStructuresESI) CorporationSovereigntyHubs(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
	return f.listCorporationSovereigntyHubs(ctx, corporationID)
}
func (f *fakeStructuresESI) CorporationSovereigntyHubDetail(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
	return f.getCorporationSovereigntyHub(ctx, corporationID, sovereigntyHubID)
}

// openStructuresFake opens the fake as the small interface one tool needs.
func openStructuresFake[C any](c C) func(context.Context, int32) (C, error) {
	return func(context.Context, int32) (C, error) { return c, nil }
}

func TestStructureHandlers_PropagateOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	wantErr := errors.New("keychain locked")
	ctx := context.Background()

	t.Run("list_character_mercenary_dens", func(t *testing.T) {
		open := func(context.Context, int32) (mercenaryDensESI, error) { return nil, wantErr }
		_, _, err := listCharacterMercenaryDensHandler("v1", open, store, nil)(ctx, nil, ListCharacterMercenaryDensInput{CharacterID: 7})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want wrapping %v", err, wantErr)
		}
	})
	t.Run("get_character_mercenary_den", func(t *testing.T) {
		open := func(context.Context, int32) (mercenaryDenESI, error) { return nil, wantErr }
		_, _, err := getCharacterMercenaryDenHandler("v1", open, store, nil)(ctx, nil, GetCharacterMercenaryDenInput{CharacterID: 7})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want wrapping %v", err, wantErr)
		}
	})
	t.Run("list_corporation_skyhooks", func(t *testing.T) {
		open := func(context.Context, int32) (corporationSkyhooksESI, error) { return nil, wantErr }
		_, _, err := listCorporationSkyhooksHandler("v1", open, store, nil)(ctx, nil, ListCorporationSkyhooksInput{CharacterID: 7})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want wrapping %v", err, wantErr)
		}
	})
	t.Run("get_corporation_skyhook", func(t *testing.T) {
		open := func(context.Context, int32) (skyhookESI, error) { return nil, wantErr }
		_, _, err := getCorporationSkyhookHandler("v1", open, store, nil)(ctx, nil, GetCorporationSkyhookInput{CharacterID: 7})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want wrapping %v", err, wantErr)
		}
	})
	t.Run("list_corporation_sovereignty_hubs", func(t *testing.T) {
		open := func(context.Context, int32) (sovereigntyHubsESI, error) { return nil, wantErr }
		_, _, err := listCorporationSovereigntyHubsHandler("v1", open, store, nil)(ctx, nil, ListCorporationSovereigntyHubsInput{CharacterID: 7})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want wrapping %v", err, wantErr)
		}
	})
	t.Run("get_corporation_sovereignty_hub", func(t *testing.T) {
		open := func(context.Context, int32) (sovereigntyHubESI, error) { return nil, wantErr }
		_, _, err := getCorporationSovereigntyHubHandler("v1", open, store, nil)(ctx, nil, GetCorporationSovereigntyHubInput{CharacterID: 7})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want wrapping %v", err, wantErr)
		}
	})
}

// --- list_character_mercenary_dens ---

func TestListCharacterMercenaryDensHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.listCharacterMercenaryDens = func(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
		return []esiapi.CharactersStructuresMercenaryDensListingMercenaryden{{Id: 1001, PlanetId: 40000001}}, nil
	}

	handler := listCharacterMercenaryDensHandler("v1", openStructuresFake[mercenaryDensESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMercenaryDensInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Dens) != 1 || out.Dens[0].ID != 1001 || out.Dens[0].PlanetID != 40000001 {
		t.Errorf("out = %+v, want CharacterID=7 Dens=[{ID:1001 PlanetID:40000001}]", out)
	}
}

func TestListCharacterMercenaryDensHandler_ResolvesPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.listCharacterMercenaryDens = func(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
		return []esiapi.CharactersStructuresMercenaryDensListingMercenaryden{{Id: 1, PlanetId: 40009086}}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := listCharacterMercenaryDensHandler("v1", openStructuresFake[mercenaryDensESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMercenaryDensInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Dens[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Dens[0].PlanetName, "Jita IV")
	}
}

func TestListCharacterMercenaryDensHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeStructuresESI{}
	fake.listCharacterMercenaryDens = func(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
		t.Fatal("listCharacterMercenaryDens should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMercenaryDensHandler("v1", openStructuresFake[mercenaryDensESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMercenaryDensInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterMercenaryDensHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	wantErr := errors.New("upstream failure")
	fake.listCharacterMercenaryDens = func(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
		return nil, wantErr
	}

	handler := listCharacterMercenaryDensHandler("v1", openStructuresFake[mercenaryDensESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMercenaryDensInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_character_mercenary_den ---

func TestGetCharacterMercenaryDenHandler_MapsFields_WithReinforcementTimer(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	reinforcementEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	fake.getCharacterMercenaryDen = func(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
		return &esiapi.CharactersStructuresMercenaryDensDetail{
			Id: 1001, TypeId: 85230, State: "online",
			Evolution: esiapi.CharactersStructuresMercenaryDensDetailEvolution{
				Anarchy:     esiapi.CharactersStructuresMercenaryDensDetailEvolutionanarchy{Amount: 10, Level: "low"},
				Development: esiapi.CharactersStructuresMercenaryDensDetailEvolutiondevelopment{Amount: 20, Level: "medium"},
			},
			Infomorphs:         esiapi.CharactersStructuresMercenaryDensDetailInfomorphs{Amount: 3},
			ReinforcementTimer: &esiapi.CharactersStructuresMercenaryDensDetailReinforcementtimer{End: reinforcementEnd},
			Skyhook:            esiapi.CharactersStructuresMercenaryDensDetailSkyhook{CorporationId: 98000001, Id: 2001, PlanetId: 40000001},
		}, nil
	}

	handler := getCharacterMercenaryDenHandler("v1", openStructuresFake[mercenaryDenESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 7, MercenaryDenID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ID != 1001 || out.TypeID != 85230 || out.State != "online" || out.InfomorphsAmount != 3 {
		t.Errorf("out = %+v, want ID=1001 TypeID=85230 State=online InfomorphsAmount=3", out)
	}
	if out.Evolution.Anarchy.Amount != 10 || out.Evolution.Anarchy.Level != "low" {
		t.Errorf("Evolution.Anarchy = %+v, want Amount=10 Level=low", out.Evolution.Anarchy)
	}
	if out.Evolution.Development.Amount != 20 || out.Evolution.Development.Level != "medium" {
		t.Errorf("Evolution.Development = %+v, want Amount=20 Level=medium", out.Evolution.Development)
	}
	if out.ReinforcementEnd == nil || !out.ReinforcementEnd.Equal(reinforcementEnd) {
		t.Errorf("ReinforcementEnd = %v, want %v", out.ReinforcementEnd, reinforcementEnd)
	}
	if out.Skyhook.CorporationID != 98000001 || out.Skyhook.ID != 2001 || out.Skyhook.PlanetID != 40000001 {
		t.Errorf("Skyhook = %+v, want CorporationID=98000001 ID=2001 PlanetID=40000001", out.Skyhook)
	}
}

func TestGetCharacterMercenaryDenHandler_MapsFields_NoReinforcementTimer(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCharacterMercenaryDen = func(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
		return &esiapi.CharactersStructuresMercenaryDensDetail{Id: 1001, TypeId: 85230, State: "offline"}, nil
	}

	handler := getCharacterMercenaryDenHandler("v1", openStructuresFake[mercenaryDenESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 7, MercenaryDenID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ReinforcementEnd != nil {
		t.Errorf("ReinforcementEnd = %v, want nil", out.ReinforcementEnd)
	}
}

func TestGetCharacterMercenaryDenHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCharacterMercenaryDen = func(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
		return &esiapi.CharactersStructuresMercenaryDensDetail{Id: 1001, TypeId: 56674}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{56674: "Mercenary Den"}
	}

	handler := getCharacterMercenaryDenHandler("v1", openStructuresFake[mercenaryDenESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 7, MercenaryDenID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 56674 {
		t.Errorf("resolveTypeNames got ids = %v, want [56674]", gotIDs)
	}
	if out.TypeName != "Mercenary Den" {
		t.Errorf("TypeName = %q, want %q", out.TypeName, "Mercenary Den")
	}
}

func TestGetCharacterMercenaryDenHandler_ResolvesSkyhookPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCharacterMercenaryDen = func(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
		return &esiapi.CharactersStructuresMercenaryDensDetail{Id: 1, Skyhook: esiapi.CharactersStructuresMercenaryDensDetailSkyhook{Id: 2, PlanetId: 40009086}}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := getCharacterMercenaryDenHandler("v1", openStructuresFake[mercenaryDenESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 7, MercenaryDenID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Skyhook.PlanetName != "Jita IV" {
		t.Errorf("Skyhook.PlanetName = %q, want %q", out.Skyhook.PlanetName, "Jita IV")
	}
}

func TestGetCharacterMercenaryDenHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeStructuresESI{}
	fake.getCharacterMercenaryDen = func(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
		t.Fatal("getCharacterMercenaryDen should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterMercenaryDenHandler("v1", openStructuresFake[mercenaryDenESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 999, MercenaryDenID: 1001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterMercenaryDenHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	wantErr := errors.New("upstream failure")
	fake.getCharacterMercenaryDen = func(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
		return nil, wantErr
	}

	handler := getCharacterMercenaryDenHandler("v1", openStructuresFake[mercenaryDenESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 7, MercenaryDenID: 1001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_skyhooks ---

func TestListCorporationSkyhooksHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.listCorporationSkyhooks = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
		return []esiapi.CorporationsStructuresSkyhooksListingSkyhook{{Id: 2001, PlanetId: 40000001}}, nil
	}

	handler := listCorporationSkyhooksHandler("v1", openStructuresFake[corporationSkyhooksESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationSkyhooksInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || len(out.Skyhooks) != 1 || out.Skyhooks[0].ID != 2001 {
		t.Errorf("out = %+v, want CharacterID=7 CorporationID=98000001 Skyhooks=[{ID:2001}]", out)
	}
}

func TestListCorporationSkyhooksHandler_ResolvesPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.listCorporationSkyhooks = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
		return []esiapi.CorporationsStructuresSkyhooksListingSkyhook{{Id: 1, PlanetId: 40009086}}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := listCorporationSkyhooksHandler("v1", openStructuresFake[corporationSkyhooksESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationSkyhooksInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Skyhooks[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Skyhooks[0].PlanetName, "Jita IV")
	}
}

func TestListCorporationSkyhooksHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeStructuresESI{}
	fake.listCorporationSkyhooks = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
		t.Fatal("listCorporationSkyhooks should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationSkyhooksHandler("v1", openStructuresFake[corporationSkyhooksESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationSkyhooksInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationSkyhooksHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	wantErr := errors.New("upstream failure")
	fake.listCorporationSkyhooks = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
		return nil, wantErr
	}

	handler := listCorporationSkyhooksHandler("v1", openStructuresFake[corporationSkyhooksESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationSkyhooksInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_corporation_skyhook ---

func TestGetCorporationSkyhookHandler_MapsFields_FullFixture(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	effectiveWorkforce := int64(500)
	reinforcementEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	theftStart := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	theftEnd := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	fake.getCorporationSkyhook = func(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
		return &esiapi.CorporationsStructuresSkyhooksDetail{
			EffectiveWorkforce: &effectiveWorkforce,
			Id:                 2001, PlanetId: 40000001, IsActive: true, State: "gathering_workforce",
			Reagents: []esiapi.CorporationsStructuresSkyhooksDetailReagent{
				{LastCycle: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), SecuredStock: 100, TypeId: 16274, UnsecuredStock: 50},
			},
			ReinforcementTimer: &esiapi.CorporationsStructuresSkyhooksDetailReinforcementtimer{End: reinforcementEnd},
			TheftVulnerability: &esiapi.CorporationsStructuresSkyhooksDetailTheftvulnerability{Start: theftStart, End: theftEnd},
		}, nil
	}

	handler := getCorporationSkyhookHandler("v1", openStructuresFake[skyhookESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 7, CorporationID: 98000001, SkyhookID: 2001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ID != 2001 || out.PlanetID != 40000001 || !out.IsActive || out.State != "gathering_workforce" {
		t.Errorf("out = %+v, want ID=2001 PlanetID=40000001 IsActive=true State=gathering_workforce", out)
	}
	if out.EffectiveWorkforce == nil || *out.EffectiveWorkforce != 500 {
		t.Errorf("EffectiveWorkforce = %v, want 500", out.EffectiveWorkforce)
	}
	if len(out.Reagents) != 1 || out.Reagents[0].SecuredStock != 100 || out.Reagents[0].TypeID != 16274 || out.Reagents[0].UnsecuredStock != 50 {
		t.Errorf("Reagents = %+v, want one entry SecuredStock=100 TypeID=16274 UnsecuredStock=50", out.Reagents)
	}
	if out.ReinforcementEnd == nil || !out.ReinforcementEnd.Equal(reinforcementEnd) {
		t.Errorf("ReinforcementEnd = %v, want %v", out.ReinforcementEnd, reinforcementEnd)
	}
	if out.TheftVulnerabilityStart == nil || !out.TheftVulnerabilityStart.Equal(theftStart) {
		t.Errorf("TheftVulnerabilityStart = %v, want %v", out.TheftVulnerabilityStart, theftStart)
	}
	if out.TheftVulnerabilityEnd == nil || !out.TheftVulnerabilityEnd.Equal(theftEnd) {
		t.Errorf("TheftVulnerabilityEnd = %v, want %v", out.TheftVulnerabilityEnd, theftEnd)
	}
}

func TestGetCorporationSkyhookHandler_MapsFields_SparseFixture(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSkyhook = func(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
		return &esiapi.CorporationsStructuresSkyhooksDetail{Id: 2001, PlanetId: 40000001, IsActive: false, State: "unpowered"}, nil
	}

	handler := getCorporationSkyhookHandler("v1", openStructuresFake[skyhookESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 7, CorporationID: 98000001, SkyhookID: 2001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.EffectiveWorkforce != nil || out.Reagents != nil || out.ReinforcementEnd != nil || out.TheftVulnerabilityStart != nil || out.TheftVulnerabilityEnd != nil {
		t.Errorf("out = %+v, want all optional fields nil/empty", out)
	}
}

func TestGetCorporationSkyhookHandler_ResolvesReagentTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSkyhook = func(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
		return &esiapi.CorporationsStructuresSkyhooksDetail{Reagents: []esiapi.CorporationsStructuresSkyhooksDetailReagent{{TypeId: 16273}}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{16273: "Oxygen"}
	}

	handler := getCorporationSkyhookHandler("v1", openStructuresFake[skyhookESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 7, CorporationID: 98000001, SkyhookID: 2001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 16273 {
		t.Errorf("resolveTypeNames got ids = %v, want [16273]", gotIDs)
	}
	if out.Reagents[0].TypeName != "Oxygen" {
		t.Errorf("Reagents[0].TypeName = %q, want %q", out.Reagents[0].TypeName, "Oxygen")
	}
}

func TestGetCorporationSkyhookHandler_ResolvesPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSkyhook = func(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
		return &esiapi.CorporationsStructuresSkyhooksDetail{Id: 1, PlanetId: 40009086}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := getCorporationSkyhookHandler("v1", openStructuresFake[skyhookESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 7, CorporationID: 98000001, SkyhookID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.PlanetName, "Jita IV")
	}
}

func TestGetCorporationSkyhookHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeStructuresESI{}
	fake.getCorporationSkyhook = func(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
		t.Fatal("getCorporationSkyhook should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationSkyhookHandler("v1", openStructuresFake[skyhookESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 999, CorporationID: 98000001, SkyhookID: 2001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationSkyhookHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	wantErr := errors.New("upstream failure")
	fake.getCorporationSkyhook = func(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
		return nil, wantErr
	}

	handler := getCorporationSkyhookHandler("v1", openStructuresFake[skyhookESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 7, CorporationID: 98000001, SkyhookID: 2001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_sovereignty_hubs ---

func TestListCorporationSovereigntyHubsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.listCorporationSovereigntyHubs = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
		return []esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub{{Id: 3001, SolarSystemId: 30000142}}, nil
	}

	handler := listCorporationSovereigntyHubsHandler("v1", openStructuresFake[sovereigntyHubsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationSovereigntyHubsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || len(out.Hubs) != 1 || out.Hubs[0].ID != 3001 || out.Hubs[0].SolarSystemID != 30000142 {
		t.Errorf("out = %+v, want CharacterID=7 CorporationID=98000001 Hubs=[{ID:3001 SolarSystemID:30000142}]", out)
	}
}

func TestListCorporationSovereigntyHubsHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.listCorporationSovereigntyHubs = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
		return []esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub{{Id: 1, SolarSystemId: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationSovereigntyHubsHandler("v1", openStructuresFake[sovereigntyHubsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationSovereigntyHubsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Hubs[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Hubs[0].SolarSystemName, "Jita")
	}
}

func TestListCorporationSovereigntyHubsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeStructuresESI{}
	fake.listCorporationSovereigntyHubs = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
		t.Fatal("listCorporationSovereigntyHubs should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationSovereigntyHubsHandler("v1", openStructuresFake[sovereigntyHubsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationSovereigntyHubsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationSovereigntyHubsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	wantErr := errors.New("upstream failure")
	fake.listCorporationSovereigntyHubs = func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
		return nil, wantErr
	}

	handler := listCorporationSovereigntyHubsHandler("v1", openStructuresFake[sovereigntyHubsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationSovereigntyHubsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_corporation_sovereignty_hub ---

func TestGetCorporationSovereigntyHubHandler_MapsFields_ExportVariant(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fuelAccessListID := int64(5001)
	vwStart := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	vwEnd := time.Date(2026, 7, 25, 2, 0, 0, 0, time.UTC)
	exportAmount := int64(300)
	exportSolarSystemID := int64(30000144)
	stateExportAmount := int64(150)
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			FuelAccessListId: &fuelAccessListID,
			Id:               3001, SolarSystemId: 30000142,
			ReagentBay: esiapi.CorporationsStructuresSovereigntyHubsDetailReagentbay{
				LastUpdated: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
				Reagents:    []esiapi.CorporationsStructuresSovereigntyHubsDetailReagent{{Amount: 1000, BurningPerHour: 10, TypeId: 16274}},
			},
			Resources: esiapi.CorporationsStructuresSovereigntyHubsDetailResources{
				Power:     esiapi.CorporationsStructuresSovereigntyHubsDetailResourcepower{Allocated: 100, Available: 200},
				Workforce: esiapi.CorporationsStructuresSovereigntyHubsDetailResourceworkforce{Allocated: 50, Available: 75},
			},
			Upgrades:            []esiapi.CorporationsStructuresSovereigntyHubsDetailUpgrade{{PowerState: "online", TypeId: 81000}},
			VulnerabilityWindow: &esiapi.CorporationsStructuresSovereigntyHubsDetailVulnerabilitywindow{Start: vwStart, End: vwEnd},
			WorkforceTransport: esiapi.CorporationsStructuresSovereigntyHubsDetailTransport{
				Configuration: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportConfiguration{
					Export: &esiapi.Export{Export: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationexport{Amount: exportAmount, SolarSystemId: &exportSolarSystemID}},
				},
				State: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportState{
					Export1: &esiapi.Export1{Export: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportstateexport{Amount: &stateExportAmount, SolarSystemId: &exportSolarSystemID}},
				},
			},
		}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ID != 3001 || out.SolarSystemID != 30000142 {
		t.Errorf("out = %+v, want ID=3001 SolarSystemID=30000142", out)
	}
	if out.FuelAccessListID == nil || *out.FuelAccessListID != 5001 {
		t.Errorf("FuelAccessListID = %v, want 5001", out.FuelAccessListID)
	}
	if len(out.ReagentBay.Reagents) != 1 || out.ReagentBay.Reagents[0].Amount != 1000 || out.ReagentBay.Reagents[0].BurningPerHour != 10 {
		t.Errorf("ReagentBay = %+v, want one reagent Amount=1000 BurningPerHour=10", out.ReagentBay)
	}
	if out.Resources.Power.Allocated != 100 || out.Resources.Workforce.Available != 75 {
		t.Errorf("Resources = %+v, want Power.Allocated=100 Workforce.Available=75", out.Resources)
	}
	if len(out.Upgrades) != 1 || out.Upgrades[0].PowerState != "online" || out.Upgrades[0].TypeID != 81000 {
		t.Errorf("Upgrades = %+v, want one entry PowerState=online TypeID=81000", out.Upgrades)
	}
	if out.VulnerabilityWindowStart == nil || !out.VulnerabilityWindowStart.Equal(vwStart) {
		t.Errorf("VulnerabilityWindowStart = %v, want %v", out.VulnerabilityWindowStart, vwStart)
	}

	cfg := out.WorkforceTransport.Configuration
	if cfg.Kind != "export" || cfg.ExportAmount == nil || *cfg.ExportAmount != 300 || cfg.ExportSolarSystemID == nil || *cfg.ExportSolarSystemID != 30000144 {
		t.Errorf("Configuration = %+v, want Kind=export ExportAmount=300 ExportSolarSystemID=30000144", cfg)
	}
	if len(cfg.ImportSolarSystemIDs) != 0 || cfg.Transit != nil {
		t.Errorf("Configuration = %+v, want no import/transit fields set", cfg)
	}

	st := out.WorkforceTransport.State
	if st.Kind != "export" || st.ExportAmount == nil || *st.ExportAmount != 150 {
		t.Errorf("State = %+v, want Kind=export ExportAmount=150", st)
	}
	if len(st.ImportSources) != 0 || st.Transit != nil {
		t.Errorf("State = %+v, want no import/transit fields set", st)
	}
}

func TestGetCorporationSovereigntyHubHandler_MapsFields_ImportVariant(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			Id: 3001, SolarSystemId: 30000142,
			WorkforceTransport: esiapi.CorporationsStructuresSovereigntyHubsDetailTransport{
				Configuration: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportConfiguration{
					ModelImport: &esiapi.ModelImport{Import: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationimport{
						Sources: []esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationsource{{SolarSystemId: 30000144}},
					}},
				},
				State: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportState{
					Import1: &esiapi.Import1{Import: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportstateimport{
						Sources: []esiapi.CorporationsStructuresSovereigntyHubsDetailTransportstateimportsource{{Amount: 500, SolarSystemId: 30000144}},
					}},
				},
			},
		}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	cfg := out.WorkforceTransport.Configuration
	if cfg.Kind != "import" || len(cfg.ImportSolarSystemIDs) != 1 || cfg.ImportSolarSystemIDs[0] != 30000144 {
		t.Errorf("Configuration = %+v, want Kind=import ImportSolarSystemIDs=[30000144]", cfg)
	}
	if cfg.ExportAmount != nil || cfg.ExportSolarSystemID != nil || cfg.Transit != nil {
		t.Errorf("Configuration = %+v, want no export/transit fields set", cfg)
	}

	st := out.WorkforceTransport.State
	if st.Kind != "import" || len(st.ImportSources) != 1 || st.ImportSources[0].Amount != 500 || st.ImportSources[0].SolarSystemID != 30000144 {
		t.Errorf("State = %+v, want Kind=import ImportSources=[{Amount:500 SolarSystemID:30000144}]", st)
	}
	if st.ExportAmount != nil || st.ExportSolarSystemID != nil || st.Transit != nil {
		t.Errorf("State = %+v, want no export/transit fields set", st)
	}
}

func TestGetCorporationSovereigntyHubHandler_MapsFields_TransitVariant(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	cfgTransit := true
	stateTransit := false
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			Id: 3001, SolarSystemId: 30000142,
			WorkforceTransport: esiapi.CorporationsStructuresSovereigntyHubsDetailTransport{
				Configuration: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportConfiguration{
					Transit: &esiapi.Transit{Transit: *esiapi.NewNullableBool(&cfgTransit)},
				},
				State: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportState{
					Transit1: &esiapi.Transit1{Transit: *esiapi.NewNullableBool(&stateTransit)},
				},
			},
		}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	cfg := out.WorkforceTransport.Configuration
	if cfg.Kind != "transit" || cfg.Transit == nil || *cfg.Transit != true {
		t.Errorf("Configuration = %+v, want Kind=transit Transit=true", cfg)
	}
	if cfg.ExportAmount != nil || cfg.ExportSolarSystemID != nil || len(cfg.ImportSolarSystemIDs) != 0 {
		t.Errorf("Configuration = %+v, want no export/import fields set", cfg)
	}

	st := out.WorkforceTransport.State
	if st.Kind != "transit" || st.Transit == nil || *st.Transit != false {
		t.Errorf("State = %+v, want Kind=transit Transit=false", st)
	}
	if st.ExportAmount != nil || st.ExportSolarSystemID != nil || len(st.ImportSources) != 0 {
		t.Errorf("State = %+v, want no export/import fields set", st)
	}
}

func TestGetCorporationSovereigntyHubHandler_ResolvesTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			ReagentBay: esiapi.CorporationsStructuresSovereigntyHubsDetailReagentbay{Reagents: []esiapi.CorporationsStructuresSovereigntyHubsDetailReagent{{TypeId: 16273}}},
			Upgrades:   []esiapi.CorporationsStructuresSovereigntyHubsDetailUpgrade{{TypeId: 587}},
		}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{
			16273: "Oxygen",
			587:   "Rifter",
		}
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 2 || gotIDs[0] != 16273 || gotIDs[1] != 587 {
		t.Errorf("resolveTypeNames got ids = %v, want [16273 587]", gotIDs)
	}
	if out.ReagentBay.Reagents[0].TypeName != "Oxygen" {
		t.Errorf("ReagentBay.Reagents[0].TypeName = %q, want %q", out.ReagentBay.Reagents[0].TypeName, "Oxygen")
	}
	if out.Upgrades[0].TypeName != "Rifter" {
		t.Errorf("Upgrades[0].TypeName = %q, want %q", out.Upgrades[0].TypeName, "Rifter")
	}
}

func TestGetCorporationSovereigntyHubHandler_ResolvesTransportSystemNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	exportID := int64(30000144)
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			Id: 1, SolarSystemId: 30000142,
			WorkforceTransport: esiapi.CorporationsStructuresSovereigntyHubsDetailTransport{
				Configuration: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportConfiguration{
					Export: &esiapi.Export{Export: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationexport{SolarSystemId: &exportID}},
				},
				State: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportState{
					Import1: &esiapi.Import1{Import: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportstateimport{
						Sources: []esiapi.CorporationsStructuresSovereigntyHubsDetailTransportstateimportsource{{Amount: 100, SolarSystemId: 30002187}},
					}},
				},
			},
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter", 30002187: "Amarr"}
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.SolarSystemName, "Jita")
	}
	if out.WorkforceTransport.Configuration.ExportSolarSystemName != "Perimeter" {
		t.Errorf("WorkforceTransport.Configuration.ExportSolarSystemName = %q, want %q", out.WorkforceTransport.Configuration.ExportSolarSystemName, "Perimeter")
	}
	if out.WorkforceTransport.State.ImportSources[0].SolarSystemName != "Amarr" {
		t.Errorf("WorkforceTransport.State.ImportSources[0].SolarSystemName = %q, want %q", out.WorkforceTransport.State.ImportSources[0].SolarSystemName, "Amarr")
	}
}

func TestGetCorporationSovereigntyHubHandler_ImportSolarSystemNamesSurvivesPartialMiss(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			Id: 3001, SolarSystemId: 30000142,
			WorkforceTransport: esiapi.CorporationsStructuresSovereigntyHubsDetailTransport{
				Configuration: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportConfiguration{
					ModelImport: &esiapi.ModelImport{Import: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationimport{
						Sources: []esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationsource{{SolarSystemId: 30000142}, {SolarSystemId: 999999999}, {SolarSystemId: 30000144}},
					}},
				},
			},
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// 999999999 is deliberately omitted -- a miss must not shift neighbors.
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter"}
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []string{"Jita", "", "Perimeter"}
	got := out.WorkforceTransport.Configuration.ImportSolarSystemNames
	if len(got) != len(want) {
		t.Fatalf("ImportSolarSystemNames = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("ImportSolarSystemNames[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestGetCorporationSovereigntyHubHandler_ImportSolarSystemNamesNilOnTotalMiss(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			Id: 3001, SolarSystemId: 999999996,
			WorkforceTransport: esiapi.CorporationsStructuresSovereigntyHubsDetailTransport{
				Configuration: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportConfiguration{
					ModelImport: &esiapi.ModelImport{Import: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationimport{
						Sources: []esiapi.CorporationsStructuresSovereigntyHubsDetailTransportconfigurationsource{{SolarSystemId: 999999997}, {SolarSystemId: 999999998}, {SolarSystemId: 999999999}},
					}},
				},
			},
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// Total sync failure: nothing resolves. ImportSolarSystemNames must stay
		// nil so omitempty drops it, rather than serializing an all-empty-string
		// slice that an LLM consumer could misread as "confirmed unnamed."
		return map[int64]string{}
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.WorkforceTransport.Configuration.ImportSolarSystemNames != nil {
		t.Errorf("ImportSolarSystemNames = %v, want nil on a total miss", out.WorkforceTransport.Configuration.ImportSolarSystemNames)
	}
}

func TestGetCorporationSovereigntyHubHandler_ResolvesStateExportSolarSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	exportID := int64(30002187)
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{
			Id: 1, SolarSystemId: 30000142,
			WorkforceTransport: esiapi.CorporationsStructuresSovereigntyHubsDetailTransport{
				State: esiapi.CorporationsStructuresSovereigntyHubsDetailTransportState{
					Export1: &esiapi.Export1{Export: &esiapi.CorporationsStructuresSovereigntyHubsDetailTransportstateexport{SolarSystemId: &exportID}},
				},
			},
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30002187: "Amarr"}
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// This is the last remaining untested transport leg: State's own export case,
	// distinct from Configuration.ExportSolarSystemID (already covered elsewhere) --
	// confirms the two same-named fields on different structs resolve independently.
	if out.WorkforceTransport.State.ExportSolarSystemName != "Amarr" {
		t.Errorf("WorkforceTransport.State.ExportSolarSystemName = %q, want %q", out.WorkforceTransport.State.ExportSolarSystemName, "Amarr")
	}
}

func TestGetCorporationSovereigntyHubHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeStructuresESI{}
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		t.Fatal("getCorporationSovereigntyHub should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 999, CorporationID: 98000001, SovereigntyHubID: 3001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationSovereigntyHubHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	wantErr := errors.New("upstream failure")
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return nil, wantErr
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationSovereigntyHubHandler_UnknownTransportVariant(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeStructuresESI{}
	fake.getCorporationSovereigntyHub = func(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
		return &esiapi.CorporationsStructuresSovereigntyHubsDetail{Id: 3001, SolarSystemId: 30000142}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", openStructuresFake[sovereigntyHubESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.WorkforceTransport.Configuration.Kind != "unknown" || out.WorkforceTransport.State.Kind != "unknown" {
		t.Errorf("WorkforceTransport = %+v, want Kind=unknown for both configuration and state", out.WorkforceTransport)
	}
}
