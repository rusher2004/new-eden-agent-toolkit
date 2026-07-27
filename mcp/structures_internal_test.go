package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// --- list_character_mercenary_dens ---

func TestListCharacterMercenaryDensHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMercenaryDens
	t.Cleanup(func() { listCharacterMercenaryDens = orig })
	listCharacterMercenaryDens = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MercenaryDen, error) {
		return []MercenaryDen{{ID: 1001, PlanetID: 40000001}}, nil
	}

	handler := listCharacterMercenaryDensHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterMercenaryDensInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Dens) != 1 || out.Dens[0].ID != 1001 || out.Dens[0].PlanetID != 40000001 {
		t.Errorf("out = %+v, want CharacterID=7 Dens=[{ID:1001 PlanetID:40000001}]", out)
	}
}

func TestListCharacterMercenaryDensHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterMercenaryDens
	t.Cleanup(func() { listCharacterMercenaryDens = orig })
	listCharacterMercenaryDens = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MercenaryDen, error) {
		t.Fatal("listCharacterMercenaryDens should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMercenaryDensHandler("v1", "client-id", store)
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

	orig := listCharacterMercenaryDens
	t.Cleanup(func() { listCharacterMercenaryDens = orig })
	wantErr := errors.New("upstream failure")
	listCharacterMercenaryDens = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MercenaryDen, error) {
		return nil, wantErr
	}

	handler := listCharacterMercenaryDensHandler("v1", "client-id", store)
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

	orig := getCharacterMercenaryDen
	t.Cleanup(func() { getCharacterMercenaryDen = orig })
	reinforcementEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, mercenaryDenID int64) (MercenaryDenDetail, error) {
		return MercenaryDenDetail{
			ID: 1001, TypeID: 85230, State: "online",
			Evolution: MercenaryDenEvolution{
				Anarchy:     MercenaryDenEvolutionStat{Amount: 10, Level: "low"},
				Development: MercenaryDenEvolutionStat{Amount: 20, Level: "medium"},
			},
			InfomorphsAmount: 3,
			ReinforcementEnd: &reinforcementEnd,
			Skyhook:          MercenaryDenSkyhookRef{CorporationID: 98000001, ID: 2001, PlanetID: 40000001},
		}, nil
	}

	handler := getCharacterMercenaryDenHandler("v1", "client-id", store)
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

	orig := getCharacterMercenaryDen
	t.Cleanup(func() { getCharacterMercenaryDen = orig })
	getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, mercenaryDenID int64) (MercenaryDenDetail, error) {
		return MercenaryDenDetail{ID: 1001, TypeID: 85230, State: "offline"}, nil
	}

	handler := getCharacterMercenaryDenHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 7, MercenaryDenID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ReinforcementEnd != nil {
		t.Errorf("ReinforcementEnd = %v, want nil", out.ReinforcementEnd)
	}
}

func TestGetCharacterMercenaryDenHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterMercenaryDen
	t.Cleanup(func() { getCharacterMercenaryDen = orig })
	getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, mercenaryDenID int64) (MercenaryDenDetail, error) {
		t.Fatal("getCharacterMercenaryDen should not be called for an unknown character")
		return MercenaryDenDetail{}, nil
	}

	handler := getCharacterMercenaryDenHandler("v1", "client-id", store)
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

	orig := getCharacterMercenaryDen
	t.Cleanup(func() { getCharacterMercenaryDen = orig })
	wantErr := errors.New("upstream failure")
	getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, mercenaryDenID int64) (MercenaryDenDetail, error) {
		return MercenaryDenDetail{}, wantErr
	}

	handler := getCharacterMercenaryDenHandler("v1", "client-id", store)
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

	orig := listCorporationSkyhooks
	t.Cleanup(func() { listCorporationSkyhooks = orig })
	listCorporationSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Skyhook, error) {
		return []Skyhook{{ID: 2001, PlanetID: 40000001}}, nil
	}

	handler := listCorporationSkyhooksHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationSkyhooksInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || len(out.Skyhooks) != 1 || out.Skyhooks[0].ID != 2001 {
		t.Errorf("out = %+v, want CharacterID=7 CorporationID=98000001 Skyhooks=[{ID:2001}]", out)
	}
}

func TestListCorporationSkyhooksHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationSkyhooks
	t.Cleanup(func() { listCorporationSkyhooks = orig })
	listCorporationSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Skyhook, error) {
		t.Fatal("listCorporationSkyhooks should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationSkyhooksHandler("v1", "client-id", store)
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

	orig := listCorporationSkyhooks
	t.Cleanup(func() { listCorporationSkyhooks = orig })
	wantErr := errors.New("upstream failure")
	listCorporationSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Skyhook, error) {
		return nil, wantErr
	}

	handler := listCorporationSkyhooksHandler("v1", "client-id", store)
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

	orig := getCorporationSkyhook
	t.Cleanup(func() { getCorporationSkyhook = orig })
	effectiveWorkforce := int64(500)
	reinforcementEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	theftStart := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	theftEnd := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
		return SkyhookDetail{
			EffectiveWorkforce: &effectiveWorkforce,
			ID:                 2001, PlanetID: 40000001, IsActive: true, State: "gathering_workforce",
			Reagents: []SkyhookReagent{
				{LastCycle: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), SecuredStock: 100, TypeID: 16274, UnsecuredStock: 50},
			},
			ReinforcementEnd:        &reinforcementEnd,
			TheftVulnerabilityStart: &theftStart,
			TheftVulnerabilityEnd:   &theftEnd,
		}, nil
	}

	handler := getCorporationSkyhookHandler("v1", "client-id", store)
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

	orig := getCorporationSkyhook
	t.Cleanup(func() { getCorporationSkyhook = orig })
	getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
		return SkyhookDetail{ID: 2001, PlanetID: 40000001, IsActive: false, State: "unpowered"}, nil
	}

	handler := getCorporationSkyhookHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 7, CorporationID: 98000001, SkyhookID: 2001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.EffectiveWorkforce != nil || out.Reagents != nil || out.ReinforcementEnd != nil || out.TheftVulnerabilityStart != nil || out.TheftVulnerabilityEnd != nil {
		t.Errorf("out = %+v, want all optional fields nil/empty", out)
	}
}

func TestGetCorporationSkyhookHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationSkyhook
	t.Cleanup(func() { getCorporationSkyhook = orig })
	getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
		t.Fatal("getCorporationSkyhook should not be called for an unknown character")
		return SkyhookDetail{}, nil
	}

	handler := getCorporationSkyhookHandler("v1", "client-id", store)
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

	orig := getCorporationSkyhook
	t.Cleanup(func() { getCorporationSkyhook = orig })
	wantErr := errors.New("upstream failure")
	getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
		return SkyhookDetail{}, wantErr
	}

	handler := getCorporationSkyhookHandler("v1", "client-id", store)
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

	orig := listCorporationSovereigntyHubs
	t.Cleanup(func() { listCorporationSovereigntyHubs = orig })
	listCorporationSovereigntyHubs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]StructureSovereigntyHub, error) {
		return []StructureSovereigntyHub{{ID: 3001, SolarSystemID: 30000142}}, nil
	}

	handler := listCorporationSovereigntyHubsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationSovereigntyHubsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || len(out.Hubs) != 1 || out.Hubs[0].ID != 3001 || out.Hubs[0].SolarSystemID != 30000142 {
		t.Errorf("out = %+v, want CharacterID=7 CorporationID=98000001 Hubs=[{ID:3001 SolarSystemID:30000142}]", out)
	}
}

func TestListCorporationSovereigntyHubsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationSovereigntyHubs
	t.Cleanup(func() { listCorporationSovereigntyHubs = orig })
	listCorporationSovereigntyHubs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]StructureSovereigntyHub, error) {
		t.Fatal("listCorporationSovereigntyHubs should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationSovereigntyHubsHandler("v1", "client-id", store)
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

	orig := listCorporationSovereigntyHubs
	t.Cleanup(func() { listCorporationSovereigntyHubs = orig })
	wantErr := errors.New("upstream failure")
	listCorporationSovereigntyHubs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]StructureSovereigntyHub, error) {
		return nil, wantErr
	}

	handler := listCorporationSovereigntyHubsHandler("v1", "client-id", store)
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

	orig := getCorporationSovereigntyHub
	t.Cleanup(func() { getCorporationSovereigntyHub = orig })
	fuelAccessListID := int64(5001)
	vwStart := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	vwEnd := time.Date(2026, 7, 25, 2, 0, 0, 0, time.UTC)
	exportAmount := int64(300)
	exportSolarSystemID := int64(30000144)
	stateExportAmount := int64(150)
	getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
		return StructureSovereigntyHubDetail{
			FuelAccessListID: &fuelAccessListID,
			ID:               3001, SolarSystemID: 30000142,
			ReagentBay: StructureSovereigntyHubReagentBay{
				LastUpdated: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
				Reagents:    []StructureSovereigntyHubReagent{{Amount: 1000, BurningPerHour: 10, TypeID: 16274}},
			},
			Resources: StructureSovereigntyHubResources{
				Power:     StructureSovereigntyHubResourceLevel{Allocated: 100, Available: 200},
				Workforce: StructureSovereigntyHubResourceLevel{Allocated: 50, Available: 75},
			},
			Upgrades:                 []StructureSovereigntyHubUpgrade{{PowerState: "online", TypeID: 81000}},
			VulnerabilityWindowStart: &vwStart,
			VulnerabilityWindowEnd:   &vwEnd,
			WorkforceTransport: StructureSovereigntyHubTransport{
				Configuration: StructureSovereigntyHubTransportConfig{Kind: "export", ExportAmount: &exportAmount, ExportSolarSystemID: &exportSolarSystemID},
				State:         StructureSovereigntyHubTransportState{Kind: "export", ExportAmount: &stateExportAmount, ExportSolarSystemID: &exportSolarSystemID},
			},
		}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", "client-id", store)
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

	orig := getCorporationSovereigntyHub
	t.Cleanup(func() { getCorporationSovereigntyHub = orig })
	getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
		return StructureSovereigntyHubDetail{
			ID: 3001, SolarSystemID: 30000142,
			WorkforceTransport: StructureSovereigntyHubTransport{
				Configuration: StructureSovereigntyHubTransportConfig{Kind: "import", ImportSolarSystemIDs: []int64{30000144}},
				State: StructureSovereigntyHubTransportState{
					Kind:          "import",
					ImportSources: []StructureSovereigntyHubTransportImportSource{{Amount: 500, SolarSystemID: 30000144}},
				},
			},
		}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", "client-id", store)
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

	orig := getCorporationSovereigntyHub
	t.Cleanup(func() { getCorporationSovereigntyHub = orig })
	cfgTransit := true
	stateTransit := false
	getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
		return StructureSovereigntyHubDetail{
			ID: 3001, SolarSystemID: 30000142,
			WorkforceTransport: StructureSovereigntyHubTransport{
				Configuration: StructureSovereigntyHubTransportConfig{Kind: "transit", Transit: &cfgTransit},
				State:         StructureSovereigntyHubTransportState{Kind: "transit", Transit: &stateTransit},
			},
		}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", "client-id", store)
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

func TestGetCorporationSovereigntyHubHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationSovereigntyHub
	t.Cleanup(func() { getCorporationSovereigntyHub = orig })
	getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
		t.Fatal("getCorporationSovereigntyHub should not be called for an unknown character")
		return StructureSovereigntyHubDetail{}, nil
	}

	handler := getCorporationSovereigntyHubHandler("v1", "client-id", store)
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

	orig := getCorporationSovereigntyHub
	t.Cleanup(func() { getCorporationSovereigntyHub = orig })
	wantErr := errors.New("upstream failure")
	getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
		return StructureSovereigntyHubDetail{}, wantErr
	}

	handler := getCorporationSovereigntyHubHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 3001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
