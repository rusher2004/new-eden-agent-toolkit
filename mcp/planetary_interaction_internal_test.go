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

// --- list_character_planets ---

func TestListCharacterPlanetsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterPlanets
	t.Cleanup(func() { listCharacterPlanets = orig })
	lastUpdate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	listCharacterPlanets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Planet, error) {
		return []Planet{{
			PlanetID: 40000001, SolarSystemID: 30000001, PlanetType: "plasma",
			OwnerID: 90000001, UpgradeLevel: 3, NumPins: 5, LastUpdate: lastUpdate,
		}}, nil
	}

	handler := listCharacterPlanetsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterPlanetsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Planets) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one planet", out)
	}
	p := out.Planets[0]
	if p.PlanetID != 40000001 || p.SolarSystemID != 30000001 || p.PlanetType != "plasma" ||
		p.OwnerID != 90000001 || p.UpgradeLevel != 3 || p.NumPins != 5 || !p.LastUpdate.Equal(lastUpdate) {
		t.Errorf("planet = %+v, want PlanetID=40000001 SolarSystemID=30000001 PlanetType=plasma OwnerID=90000001 UpgradeLevel=3 NumPins=5 LastUpdate=%v", p, lastUpdate)
	}
}

func TestListCharacterPlanetsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterPlanets
	t.Cleanup(func() { listCharacterPlanets = orig })
	listCharacterPlanets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Planet, error) {
		t.Fatal("listCharacterPlanets should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterPlanetsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterPlanetsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// TestListCharacterPlanetsHandler_ResolvesLocationNames proves the new sdeStore-backed
// SolarSystemName/PlanetName enrichment via resolveSystemNames/resolvePlanetNames.
func TestListCharacterPlanetsHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterPlanets
	t.Cleanup(func() { listCharacterPlanets = orig })
	listCharacterPlanets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Planet, error) {
		return []Planet{{PlanetID: 40009086, SolarSystemID: 30000142}}, nil
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

	handler := listCharacterPlanetsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterPlanetsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Planets[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Planets[0].SolarSystemName, "Jita")
	}
	if out.Planets[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Planets[0].PlanetName, "Jita IV")
	}
}

// TestListCorporationCustomsOfficesHandler_ResolvesSystemName proves the new SystemName
// enrichment pass added alongside the existing TypeName enrichment.
func TestListCorporationCustomsOfficesHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdCustomsOfficesGetInner{{OfficeId: 1, SystemId: 30000142}}, 1, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Offices[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Offices[0].SystemName, "Jita")
	}
}

// --- get_character_planet ---
//
// This test exercises every nested level of the colony-layout mapping: top-level Links/Pins/
// Routes, plus a Pin carrying Contents, ExtractorDetails (with a nested Heads entry), and
// FactoryDetails, and a second Pin with none of those optional nested fields set (proving the
// pointer/slice fields are correctly left nil/empty rather than defaulted to zero-value structs).

func TestGetCharacterPlanetHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterPlanet
	t.Cleanup(func() { getCharacterPlanet = orig })

	expiryTime := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	installTime := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	lastCycleStart := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	schematicID := int64(100)
	cycleTime := int64(1800)
	headRadius := 200.5
	productTypeID := int64(2073)
	qtyPerCycle := int64(100)

	getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
		return &esiapi.CharactersCharacterIdPlanetsPlanetIdGet{
			Links: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetLinksInner{
				{SourcePinId: 1001, DestinationPinId: 1002, LinkLevel: 1},
			},
			Pins: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInner{
				{
					PinId: 1001, TypeId: 2254, Latitude: 0.1, Longitude: 0.2,
					SchematicId: &schematicID, ExpiryTime: &expiryTime, InstallTime: &installTime,
					LastCycleStart: &lastCycleStart,
					Contents: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerContentsInner{
						{TypeId: 2073, Amount: 500},
					},
					ExtractorDetails: &esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerExtractorDetails{
						CycleTime: &cycleTime, HeadRadius: &headRadius, ProductTypeId: &productTypeID, QtyPerCycle: &qtyPerCycle,
						Heads: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerExtractorDetailsHeadsInner{
							{HeadId: 0, Latitude: 0.3, Longitude: 0.4},
						},
					},
					FactoryDetails: &esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerFactoryDetails{SchematicId: 100},
				},
				{PinId: 1002, TypeId: 2255, Latitude: 0.5, Longitude: 0.6},
			},
			Routes: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetRoutesInner{
				{
					RouteId: 5001, SourcePinId: 1001, DestinationPinId: 1002, ContentTypeId: 2073,
					Quantity: 20, Waypoints: []int64{1003, 1004},
				},
			},
		}, nil
	}

	handler := getCharacterPlanetHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterPlanetInput{CharacterID: 7, PlanetID: 40000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.PlanetID != 40000001 {
		t.Fatalf("out = %+v, want CharacterID=7 PlanetID=40000001", out)
	}

	if len(out.Links) != 1 {
		t.Fatalf("Links = %+v, want one entry", out.Links)
	}
	l := out.Links[0]
	if l.SourcePinID != 1001 || l.DestinationPinID != 1002 || l.LinkLevel != 1 {
		t.Errorf("Links[0] = %+v, want SourcePinID=1001 DestinationPinID=1002 LinkLevel=1", l)
	}

	if len(out.Pins) != 2 {
		t.Fatalf("Pins = %+v, want two entries", out.Pins)
	}
	p0 := out.Pins[0]
	if p0.PinID != 1001 || p0.TypeID != 2254 || p0.Latitude != 0.1 || p0.Longitude != 0.2 {
		t.Errorf("Pins[0] = %+v, want PinID=1001 TypeID=2254 Latitude=0.1 Longitude=0.2", p0)
	}
	if p0.SchematicID == nil || *p0.SchematicID != 100 {
		t.Errorf("Pins[0].SchematicID = %v, want *100", p0.SchematicID)
	}
	if p0.ExpiryTime == nil || !p0.ExpiryTime.Equal(expiryTime) {
		t.Errorf("Pins[0].ExpiryTime = %v, want %v", p0.ExpiryTime, expiryTime)
	}
	if p0.InstallTime == nil || !p0.InstallTime.Equal(installTime) {
		t.Errorf("Pins[0].InstallTime = %v, want %v", p0.InstallTime, installTime)
	}
	if p0.LastCycleStart == nil || !p0.LastCycleStart.Equal(lastCycleStart) {
		t.Errorf("Pins[0].LastCycleStart = %v, want %v", p0.LastCycleStart, lastCycleStart)
	}
	if len(p0.Contents) != 1 || p0.Contents[0].TypeID != 2073 || p0.Contents[0].Amount != 500 {
		t.Errorf("Pins[0].Contents = %+v, want one entry {TypeID:2073 Amount:500}", p0.Contents)
	}
	if p0.ExtractorDetails == nil {
		t.Fatal("Pins[0].ExtractorDetails = nil, want non-nil")
	}
	ed := p0.ExtractorDetails
	if ed.CycleTime == nil || *ed.CycleTime != 1800 {
		t.Errorf("ExtractorDetails.CycleTime = %v, want *1800", ed.CycleTime)
	}
	if ed.HeadRadius == nil || *ed.HeadRadius != 200.5 {
		t.Errorf("ExtractorDetails.HeadRadius = %v, want *200.5", ed.HeadRadius)
	}
	if ed.ProductTypeID == nil || *ed.ProductTypeID != 2073 {
		t.Errorf("ExtractorDetails.ProductTypeID = %v, want *2073", ed.ProductTypeID)
	}
	if ed.QtyPerCycle == nil || *ed.QtyPerCycle != 100 {
		t.Errorf("ExtractorDetails.QtyPerCycle = %v, want *100", ed.QtyPerCycle)
	}
	if len(ed.Heads) != 1 || ed.Heads[0].HeadID != 0 || ed.Heads[0].Latitude != 0.3 || ed.Heads[0].Longitude != 0.4 {
		t.Errorf("ExtractorDetails.Heads = %+v, want one entry {HeadID:0 Latitude:0.3 Longitude:0.4}", ed.Heads)
	}
	if p0.FactoryDetails == nil || p0.FactoryDetails.SchematicID != 100 {
		t.Errorf("Pins[0].FactoryDetails = %+v, want non-nil with SchematicID=100", p0.FactoryDetails)
	}
	p1 := out.Pins[1]
	if p1.PinID != 1002 || p1.TypeID != 2255 {
		t.Errorf("Pins[1] = %+v, want PinID=1002 TypeID=2255", p1)
	}
	if p1.ExtractorDetails != nil || p1.FactoryDetails != nil || len(p1.Contents) != 0 || p1.SchematicID != nil {
		t.Errorf("Pins[1] = %+v, want no extractor/factory details, no contents, no schematicId", p1)
	}

	if len(out.Routes) != 1 {
		t.Fatalf("Routes = %+v, want one entry", out.Routes)
	}
	r := out.Routes[0]
	if r.RouteID != 5001 || r.SourcePinID != 1001 || r.DestinationPinID != 1002 || r.ContentTypeID != 2073 || r.Quantity != 20 {
		t.Errorf("Routes[0] = %+v, want RouteID=5001 SourcePinID=1001 DestinationPinID=1002 ContentTypeID=2073 Quantity=20", r)
	}
	if len(r.Waypoints) != 2 || r.Waypoints[0] != 1003 || r.Waypoints[1] != 1004 {
		t.Errorf("Routes[0].Waypoints = %+v, want [1003 1004]", r.Waypoints)
	}
}

func TestGetCharacterPlanetHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterPlanet
	t.Cleanup(func() { getCharacterPlanet = orig })
	getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
		t.Fatal("getCharacterPlanet should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterPlanetHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterPlanetInput{CharacterID: 999, PlanetID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_corporation_customs_offices ---

func TestListCorporationCustomsOfficesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })

	allianceTax := 0.1
	badTax := 0.5
	corpTax := 0.05
	excellentTax := 0.02
	goodTax := 0.03
	neutralTax := 0.08
	terribleTax := 0.75
	standingLevel := "good"
	typeID := int64(2233)

	var gotCorporationID int64
	var gotPage int32
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		gotCorporationID = corporationID
		gotPage = page
		return []esiapi.CorporationsCorporationIdCustomsOfficesGetInner{
			{
				OfficeId: 1234567890, SystemId: 30000001, TypeId: &typeID,
				ReinforceExitStart: 16, ReinforceExitEnd: 18,
				AllowAllianceAccess: true, AllowAccessWithStandings: true,
				StandingLevel: &standingLevel, AllianceTaxRate: &allianceTax, CorporationTaxRate: &corpTax,
				ExcellentStandingTaxRate: &excellentTax, GoodStandingTaxRate: &goodTax, NeutralStandingTaxRate: &neutralTax,
				BadStandingTaxRate: &badTax, TerribleStandingTaxRate: &terribleTax,
			},
		}, 3, nil
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 7, CorporationID: 98000001, Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCorporationID != 98000001 || gotPage != 2 {
		t.Errorf("captured corporationID/page = %d/%d, want 98000001/2", gotCorporationID, gotPage)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 2 || out.TotalPages != 3 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=2 TotalPages=3", out)
	}
	if len(out.Offices) != 1 {
		t.Fatalf("Offices = %+v, want one entry", out.Offices)
	}
	o := out.Offices[0]
	if o.OfficeID != 1234567890 || o.SystemID != 30000001 {
		t.Errorf("office = %+v, want OfficeID=1234567890 SystemID=30000001", o)
	}
	if o.TypeID == nil || *o.TypeID != 2233 {
		t.Errorf("office.TypeID = %v, want *2233", o.TypeID)
	}
	if o.ReinforceExitStart != 16 || o.ReinforceExitEnd != 18 {
		t.Errorf("office reinforce window = start=%d end=%d, want start=16 end=18", o.ReinforceExitStart, o.ReinforceExitEnd)
	}
	if !o.AllowAllianceAccess || !o.AllowAccessWithStandings {
		t.Errorf("office = %+v, want AllowAllianceAccess and AllowAccessWithStandings both true", o)
	}
	if o.StandingLevel == nil || *o.StandingLevel != "good" {
		t.Errorf("office.StandingLevel = %v, want *\"good\"", o.StandingLevel)
	}
	if o.AllianceTaxRate == nil || *o.AllianceTaxRate != 0.1 {
		t.Errorf("office.AllianceTaxRate = %v, want *0.1", o.AllianceTaxRate)
	}
	if o.CorporationTaxRate == nil || *o.CorporationTaxRate != 0.05 {
		t.Errorf("office.CorporationTaxRate = %v, want *0.05", o.CorporationTaxRate)
	}
	if o.ExcellentStandingTaxRate == nil || *o.ExcellentStandingTaxRate != 0.02 {
		t.Errorf("office.ExcellentStandingTaxRate = %v, want *0.02", o.ExcellentStandingTaxRate)
	}
	if o.GoodStandingTaxRate == nil || *o.GoodStandingTaxRate != 0.03 {
		t.Errorf("office.GoodStandingTaxRate = %v, want *0.03", o.GoodStandingTaxRate)
	}
	if o.NeutralStandingTaxRate == nil || *o.NeutralStandingTaxRate != 0.08 {
		t.Errorf("office.NeutralStandingTaxRate = %v, want *0.08", o.NeutralStandingTaxRate)
	}
	if o.BadStandingTaxRate == nil || *o.BadStandingTaxRate != 0.5 {
		t.Errorf("office.BadStandingTaxRate = %v, want *0.5", o.BadStandingTaxRate)
	}
	if o.TerribleStandingTaxRate == nil || *o.TerribleStandingTaxRate != 0.75 {
		t.Errorf("office.TerribleStandingTaxRate = %v, want *0.75", o.TerribleStandingTaxRate)
	}
}

func TestListCorporationCustomsOfficesHandler_DefaultsPageToOne(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		return nil, 1, nil
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("Page = %d, want 1 when input page is omitted", out.Page)
	}
}

func TestListCorporationCustomsOfficesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		t.Fatal("listCorporationCustomsOffices should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// TestGetCharacterPlanetHandler_ResolvesTypeNames exercises all four enrichment sites in one
// combined resolveTypeNames call: a pin's flat TypeID, a pin content's nested TypeID, an
// extractor-details' nested *int64 ProductTypeID, and a sibling route's flat ContentTypeID.
func TestGetCharacterPlanetHandler_ResolvesTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterPlanet
	t.Cleanup(func() { getCharacterPlanet = orig })
	productTypeID := int64(587)
	getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
		return &esiapi.CharactersCharacterIdPlanetsPlanetIdGet{
			Pins: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInner{{
				TypeId:           587,
				Contents:         []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerContentsInner{{TypeId: 588, Amount: 100}},
				ExtractorDetails: &esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerExtractorDetails{ProductTypeId: &productTypeID},
			}},
			Routes: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetRoutesInner{{ContentTypeId: 588}},
		}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Command Center", 588: "Water"}
	}

	handler := getCharacterPlanetHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterPlanetInput{CharacterID: 7, PlanetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Pins) != 1 {
		t.Fatalf("Pins = %+v, want one entry", out.Pins)
	}
	if out.Pins[0].TypeName != "Command Center" {
		t.Errorf("Pins[0].TypeName = %q, want %q", out.Pins[0].TypeName, "Command Center")
	}
	if len(out.Pins[0].Contents) != 1 || out.Pins[0].Contents[0].TypeName != "Water" {
		t.Errorf("Pins[0].Contents = %+v, want one entry with TypeName=Water", out.Pins[0].Contents)
	}
	if out.Pins[0].ExtractorDetails == nil || out.Pins[0].ExtractorDetails.ProductTypeName != "Command Center" {
		t.Errorf("Pins[0].ExtractorDetails = %+v, want non-nil with ProductTypeName=Command Center", out.Pins[0].ExtractorDetails)
	}
	if len(out.Routes) != 1 || out.Routes[0].ContentTypeName != "Water" {
		t.Errorf("Routes = %+v, want one entry with ContentTypeName=Water", out.Routes)
	}
}

// TestGetCharacterPlanetHandler_ResolvesTypeNames_NoExtractorDetails proves the ExtractorDetails
// nil-guard: a pin with no ExtractorDetails must not panic, and pins with an ExtractorDetails
// whose ProductTypeID is nil must not panic either.
func TestGetCharacterPlanetHandler_ResolvesTypeNames_NoExtractorDetails(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterPlanet
	t.Cleanup(func() { getCharacterPlanet = orig })
	getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
		return &esiapi.CharactersCharacterIdPlanetsPlanetIdGet{
			Pins: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInner{
				{TypeId: 587},
				{TypeId: 588, ExtractorDetails: &esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerExtractorDetails{}},
			},
		}, nil
	}

	handler := getCharacterPlanetHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterPlanetInput{CharacterID: 7, PlanetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Pins) != 2 {
		t.Fatalf("Pins = %+v, want two entries", out.Pins)
	}
	if out.Pins[1].ExtractorDetails == nil || out.Pins[1].ExtractorDetails.ProductTypeName != "" {
		t.Errorf("Pins[1].ExtractorDetails = %+v, want non-nil with empty ProductTypeName", out.Pins[1].ExtractorDetails)
	}
}

// TestListCorporationCustomsOfficesHandler_ResolvesTypeName proves the *int64 TypeID nil-guard:
// one office with a TypeID gets a resolved name, one without a TypeID is left untouched.
func TestListCorporationCustomsOfficesHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	typeID := int64(2233)
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdCustomsOfficesGetInner{
			{OfficeId: 1, TypeId: &typeID},
			{OfficeId: 2},
		}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{2233: "Customs Office"}
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 7, CorporationID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Offices) != 2 {
		t.Fatalf("Offices = %+v, want two entries", out.Offices)
	}
	if out.Offices[0].TypeName != "Customs Office" {
		t.Errorf("Offices[0].TypeName = %q, want %q", out.Offices[0].TypeName, "Customs Office")
	}
	if out.Offices[1].TypeID != nil || out.Offices[1].TypeName != "" {
		t.Errorf("Offices[1] = %+v, want nil TypeID and empty TypeName", out.Offices[1])
	}
}

// --- get_planetary_schematic (public, no character check) ---

func TestGetPlanetarySchematicHandler_MapsFields(t *testing.T) {
	orig := getPlanetarySchematic
	t.Cleanup(func() { getPlanetarySchematic = orig })
	getPlanetarySchematic = func(ctx context.Context, userAgent string, schematicID int64) (GetPlanetarySchematicOutput, error) {
		return GetPlanetarySchematicOutput{SchematicID: schematicID, SchematicName: "Water", CycleTime: 1800}, nil
	}

	handler := getPlanetarySchematicHandler("v1")
	_, out, err := handler(context.Background(), nil, GetPlanetarySchematicInput{SchematicID: 90})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SchematicID != 90 || out.SchematicName != "Water" || out.CycleTime != 1800 {
		t.Errorf("out = %+v, want SchematicID=90 SchematicName=\"Water\" CycleTime=1800", out)
	}
}

func TestGetPlanetarySchematicHandler_PropagatesError(t *testing.T) {
	orig := getPlanetarySchematic
	t.Cleanup(func() { getPlanetarySchematic = orig })
	getPlanetarySchematic = func(ctx context.Context, userAgent string, schematicID int64) (GetPlanetarySchematicOutput, error) {
		return GetPlanetarySchematicOutput{}, errors.New("boom")
	}

	handler := getPlanetarySchematicHandler("v1")
	_, _, err := handler(context.Background(), nil, GetPlanetarySchematicInput{SchematicID: 90})
	if err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
