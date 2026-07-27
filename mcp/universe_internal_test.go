package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// --- list_universe_ancestries ---

func TestListUniverseAncestriesHandler_MapsFields(t *testing.T) {
	orig := listUniverseAncestries
	t.Cleanup(func() { listUniverseAncestries = orig })
	iconID := int64(100)
	shortDesc := "short"
	listUniverseAncestries = func(ctx context.Context, userAgent string) ([]UniverseAncestry, error) {
		return []UniverseAncestry{{ID: 11, BloodlineID: 1, Description: "desc", Name: "Tribal", IconID: &iconID, ShortDescription: &shortDesc}}, nil
	}

	handler := listUniverseAncestriesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseAncestriesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Ancestries) != 1 || out.Ancestries[0].ID != 11 || out.Ancestries[0].Name != "Tribal" {
		t.Errorf("out = %+v, want one entry ID=11 Name=Tribal", out)
	}
}

func TestListUniverseAncestriesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseAncestries
	t.Cleanup(func() { listUniverseAncestries = orig })
	wantErr := errors.New("upstream failure")
	listUniverseAncestries = func(ctx context.Context, userAgent string) ([]UniverseAncestry, error) {
		return nil, wantErr
	}

	handler := listUniverseAncestriesHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseAncestriesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_bloodlines ---

func TestListUniverseBloodlinesHandler_MapsFields(t *testing.T) {
	orig := listUniverseBloodlines
	t.Cleanup(func() { listUniverseBloodlines = orig })
	listUniverseBloodlines = func(ctx context.Context, userAgent string) ([]UniverseBloodline, error) {
		return []UniverseBloodline{{ID: 1, Name: "Deteis", RaceID: 1, CorporationID: 1000166}}, nil
	}

	handler := listUniverseBloodlinesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseBloodlinesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bloodlines) != 1 || out.Bloodlines[0].ID != 1 || out.Bloodlines[0].Name != "Deteis" {
		t.Errorf("out = %+v, want one entry ID=1 Name=Deteis", out)
	}
}

func TestListUniverseBloodlinesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseBloodlines
	t.Cleanup(func() { listUniverseBloodlines = orig })
	wantErr := errors.New("upstream failure")
	listUniverseBloodlines = func(ctx context.Context, userAgent string) ([]UniverseBloodline, error) {
		return nil, wantErr
	}

	handler := listUniverseBloodlinesHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseBloodlinesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_races ---

func TestListUniverseRacesHandler_MapsFields(t *testing.T) {
	orig := listUniverseRaces
	t.Cleanup(func() { listUniverseRaces = orig })
	listUniverseRaces = func(ctx context.Context, userAgent string) ([]UniverseRace, error) {
		return []UniverseRace{{RaceID: 1, AllianceID: 500001, Name: "Caldari"}}, nil
	}

	handler := listUniverseRacesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseRacesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Races) != 1 || out.Races[0].RaceID != 1 || out.Races[0].Name != "Caldari" {
		t.Errorf("out = %+v, want one entry RaceID=1 Name=Caldari", out)
	}
}

func TestListUniverseRacesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseRaces
	t.Cleanup(func() { listUniverseRaces = orig })
	wantErr := errors.New("upstream failure")
	listUniverseRaces = func(ctx context.Context, userAgent string) ([]UniverseRace, error) {
		return nil, wantErr
	}

	handler := listUniverseRacesHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseRacesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_factions ---

func TestListUniverseFactionsHandler_MapsFields(t *testing.T) {
	orig := listUniverseFactions
	t.Cleanup(func() { listUniverseFactions = orig })
	corpID := int64(1000035)
	listUniverseFactions = func(ctx context.Context, userAgent string) ([]UniverseFaction, error) {
		return []UniverseFaction{{FactionID: 500001, Name: "Caldari State", IsUnique: true, CorporationID: &corpID}}, nil
	}

	handler := listUniverseFactionsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseFactionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Factions) != 1 || out.Factions[0].FactionID != 500001 || !out.Factions[0].IsUnique {
		t.Errorf("out = %+v, want one entry FactionID=500001 IsUnique=true", out)
	}
	if out.Factions[0].CorporationID == nil || *out.Factions[0].CorporationID != 1000035 {
		t.Errorf("CorporationID = %v, want 1000035", out.Factions[0].CorporationID)
	}
}

func TestListUniverseFactionsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseFactions
	t.Cleanup(func() { listUniverseFactions = orig })
	wantErr := errors.New("upstream failure")
	listUniverseFactions = func(ctx context.Context, userAgent string) ([]UniverseFaction, error) {
		return nil, wantErr
	}

	handler := listUniverseFactionsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseFactionsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_categories ---

func TestListUniverseCategoriesHandler_MapsFields(t *testing.T) {
	orig := listUniverseCategories
	t.Cleanup(func() { listUniverseCategories = orig })
	listUniverseCategories = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{6, 7}, nil
	}

	handler := listUniverseCategoriesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseCategoriesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.IDs[0] != 6 {
		t.Errorf("out = %+v, want IDs=[6 7]", out)
	}
}

func TestListUniverseCategoriesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseCategories
	t.Cleanup(func() { listUniverseCategories = orig })
	wantErr := errors.New("upstream failure")
	listUniverseCategories = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseCategoriesHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseCategoriesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_category ---

func TestGetUniverseCategoryHandler_MapsFields(t *testing.T) {
	orig := getUniverseCategory
	t.Cleanup(func() { getUniverseCategory = orig })
	getUniverseCategory = func(ctx context.Context, userAgent string, categoryID int64) (UniverseCategory, error) {
		return UniverseCategory{CategoryID: 6, Name: "Ship", Published: true, GroupIDs: []int64{25, 26}}, nil
	}

	handler := getUniverseCategoryHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseCategoryInput{CategoryID: 6})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CategoryID != 6 || out.Name != "Ship" || !out.Published || len(out.GroupIDs) != 2 {
		t.Errorf("out = %+v, want CategoryID=6 Name=Ship Published=true GroupIDs=[25 26]", out)
	}
}

func TestGetUniverseCategoryHandler_PropagatesError(t *testing.T) {
	orig := getUniverseCategory
	t.Cleanup(func() { getUniverseCategory = orig })
	wantErr := errors.New("upstream failure")
	getUniverseCategory = func(ctx context.Context, userAgent string, categoryID int64) (UniverseCategory, error) {
		return UniverseCategory{}, wantErr
	}

	handler := getUniverseCategoryHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseCategoryInput{CategoryID: 6})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_groups ---

func TestListUniverseGroupsHandler_MapsFields(t *testing.T) {
	orig := listUniverseGroups
	t.Cleanup(func() { listUniverseGroups = orig })
	listUniverseGroups = func(ctx context.Context, userAgent string, page int32) ([]int64, int32, error) {
		return []int64{25, 26}, 3, nil
	}

	handler := listUniverseGroupsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseGroupsInput{Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.Page != 2 || out.TotalPages != 3 {
		t.Errorf("out = %+v, want IDs=[25 26] Page=2 TotalPages=3", out)
	}
}

func TestListUniverseGroupsHandler_DefaultsPageToOne(t *testing.T) {
	orig := listUniverseGroups
	t.Cleanup(func() { listUniverseGroups = orig })
	listUniverseGroups = func(ctx context.Context, userAgent string, page int32) ([]int64, int32, error) {
		return []int64{25}, 3, nil
	}

	handler := listUniverseGroupsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseGroupsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("Page = %d, want 1", out.Page)
	}
}

func TestListUniverseGroupsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseGroups
	t.Cleanup(func() { listUniverseGroups = orig })
	wantErr := errors.New("upstream failure")
	listUniverseGroups = func(ctx context.Context, userAgent string, page int32) ([]int64, int32, error) {
		return nil, 0, wantErr
	}

	handler := listUniverseGroupsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseGroupsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_group ---

func TestGetUniverseGroupHandler_MapsFields(t *testing.T) {
	orig := getUniverseGroup
	t.Cleanup(func() { getUniverseGroup = orig })
	getUniverseGroup = func(ctx context.Context, userAgent string, groupID int64) (UniverseGroup, error) {
		return UniverseGroup{GroupID: 25, CategoryID: 6, Name: "Frigate", Published: true, TypeIDs: []int64{587, 588}}, nil
	}

	handler := getUniverseGroupHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseGroupInput{GroupID: 25})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.GroupID != 25 || out.CategoryID != 6 || out.Name != "Frigate" || len(out.TypeIDs) != 2 {
		t.Errorf("out = %+v, want GroupID=25 CategoryID=6 Name=Frigate TypeIDs=[587 588]", out)
	}
}

func TestGetUniverseGroupHandler_PropagatesError(t *testing.T) {
	orig := getUniverseGroup
	t.Cleanup(func() { getUniverseGroup = orig })
	wantErr := errors.New("upstream failure")
	getUniverseGroup = func(ctx context.Context, userAgent string, groupID int64) (UniverseGroup, error) {
		return UniverseGroup{}, wantErr
	}

	handler := getUniverseGroupHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseGroupInput{GroupID: 25})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_types ---

func TestListUniverseTypesHandler_MapsFields(t *testing.T) {
	orig := listUniverseTypes
	t.Cleanup(func() { listUniverseTypes = orig })
	listUniverseTypes = func(ctx context.Context, userAgent string, page int32) ([]int64, int32, error) {
		return []int64{587, 588}, 5, nil
	}

	handler := listUniverseTypesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseTypesInput{Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.Page != 1 || out.TotalPages != 5 {
		t.Errorf("out = %+v, want IDs=[587 588] Page=1 TotalPages=5", out)
	}
}

func TestListUniverseTypesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseTypes
	t.Cleanup(func() { listUniverseTypes = orig })
	wantErr := errors.New("upstream failure")
	listUniverseTypes = func(ctx context.Context, userAgent string, page int32) ([]int64, int32, error) {
		return nil, 0, wantErr
	}

	handler := listUniverseTypesHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseTypesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_type ---

func TestGetUniverseTypeHandler_MapsFields(t *testing.T) {
	orig := getUniverseType
	t.Cleanup(func() { getUniverseType = orig })
	mass := 1200000.0
	getUniverseType = func(ctx context.Context, userAgent string, typeID int64) (UniverseType, error) {
		return UniverseType{
			TypeID: 587, Name: "Rifter", GroupID: 25, Published: true, Mass: &mass,
			DogmaAttributes: []DynamicItemAttribute{{AttributeID: 9, Value: 2500}},
			DogmaEffects:    []DynamicItemEffect{{EffectID: 1, IsDefault: true}},
		}, nil
	}

	handler := getUniverseTypeHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseTypeInput{TypeID: 587})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TypeID != 587 || out.Name != "Rifter" || out.GroupID != 25 || !out.Published {
		t.Errorf("out = %+v, want TypeID=587 Name=Rifter GroupID=25 Published=true", out)
	}
	if out.Mass == nil || *out.Mass != 1200000 {
		t.Errorf("Mass = %v, want 1200000", out.Mass)
	}
	if len(out.DogmaAttributes) != 1 || out.DogmaAttributes[0].AttributeID != 9 || out.DogmaAttributes[0].Value != 2500 {
		t.Errorf("DogmaAttributes = %+v, want one entry AttributeID=9 Value=2500", out.DogmaAttributes)
	}
	if len(out.DogmaEffects) != 1 || out.DogmaEffects[0].EffectID != 1 || !out.DogmaEffects[0].IsDefault {
		t.Errorf("DogmaEffects = %+v, want one entry EffectID=1 IsDefault=true", out.DogmaEffects)
	}
}

func TestGetUniverseTypeHandler_PropagatesError(t *testing.T) {
	orig := getUniverseType
	t.Cleanup(func() { getUniverseType = orig })
	wantErr := errors.New("upstream failure")
	getUniverseType = func(ctx context.Context, userAgent string, typeID int64) (UniverseType, error) {
		return UniverseType{}, wantErr
	}

	handler := getUniverseTypeHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseTypeInput{TypeID: 587})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_regions ---

func TestListUniverseRegionsHandler_MapsFields(t *testing.T) {
	orig := listUniverseRegions
	t.Cleanup(func() { listUniverseRegions = orig })
	listUniverseRegions = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{10000002, 10000043}, nil
	}

	handler := listUniverseRegionsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseRegionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.IDs[0] != 10000002 {
		t.Errorf("out = %+v, want IDs=[10000002 10000043]", out)
	}
}

func TestListUniverseRegionsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseRegions
	t.Cleanup(func() { listUniverseRegions = orig })
	wantErr := errors.New("upstream failure")
	listUniverseRegions = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseRegionsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseRegionsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_region ---

func TestGetUniverseRegionHandler_MapsFields(t *testing.T) {
	orig := getUniverseRegion
	t.Cleanup(func() { getUniverseRegion = orig })
	desc := "desc"
	getUniverseRegion = func(ctx context.Context, userAgent string, regionID int64) (UniverseRegion, error) {
		return UniverseRegion{RegionID: 10000002, Name: "The Forge", Description: &desc, ConstellationIDs: []int64{20000020}}, nil
	}

	handler := getUniverseRegionHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseRegionInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.Name != "The Forge" || len(out.ConstellationIDs) != 1 {
		t.Errorf("out = %+v, want RegionID=10000002 Name=The Forge ConstellationIDs=[20000020]", out)
	}
}

func TestGetUniverseRegionHandler_PropagatesError(t *testing.T) {
	orig := getUniverseRegion
	t.Cleanup(func() { getUniverseRegion = orig })
	wantErr := errors.New("upstream failure")
	getUniverseRegion = func(ctx context.Context, userAgent string, regionID int64) (UniverseRegion, error) {
		return UniverseRegion{}, wantErr
	}

	handler := getUniverseRegionHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseRegionInput{RegionID: 10000002})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_constellations ---

func TestListUniverseConstellationsHandler_MapsFields(t *testing.T) {
	orig := listUniverseConstellations
	t.Cleanup(func() { listUniverseConstellations = orig })
	listUniverseConstellations = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{20000020}, nil
	}

	handler := listUniverseConstellationsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseConstellationsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 1 || out.IDs[0] != 20000020 {
		t.Errorf("out = %+v, want IDs=[20000020]", out)
	}
}

func TestListUniverseConstellationsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseConstellations
	t.Cleanup(func() { listUniverseConstellations = orig })
	wantErr := errors.New("upstream failure")
	listUniverseConstellations = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseConstellationsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseConstellationsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_constellation ---

func TestGetUniverseConstellationHandler_MapsFields(t *testing.T) {
	orig := getUniverseConstellation
	t.Cleanup(func() { getUniverseConstellation = orig })
	getUniverseConstellation = func(ctx context.Context, userAgent string, constellationID int64) (UniverseConstellation, error) {
		return UniverseConstellation{
			ConstellationID: 20000020, RegionID: 10000002, Name: "Kimotoro",
			Position: Position{X: 1, Y: 2, Z: 3}, SystemIDs: []int64{30000142, 30000144},
		}, nil
	}

	handler := getUniverseConstellationHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseConstellationInput{ConstellationID: 20000020})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ConstellationID != 20000020 || out.RegionID != 10000002 || out.Name != "Kimotoro" || len(out.SystemIDs) != 2 {
		t.Errorf("out = %+v, want ConstellationID=20000020 RegionID=10000002 Name=Kimotoro SystemIDs=[30000142 30000144]", out)
	}
	if out.Position.X != 1 || out.Position.Y != 2 || out.Position.Z != 3 {
		t.Errorf("Position = %+v, want X=1 Y=2 Z=3", out.Position)
	}
}

func TestGetUniverseConstellationHandler_PropagatesError(t *testing.T) {
	orig := getUniverseConstellation
	t.Cleanup(func() { getUniverseConstellation = orig })
	wantErr := errors.New("upstream failure")
	getUniverseConstellation = func(ctx context.Context, userAgent string, constellationID int64) (UniverseConstellation, error) {
		return UniverseConstellation{}, wantErr
	}

	handler := getUniverseConstellationHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseConstellationInput{ConstellationID: 20000020})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_systems ---

func TestListUniverseSystemsHandler_MapsFields(t *testing.T) {
	orig := listUniverseSystems
	t.Cleanup(func() { listUniverseSystems = orig })
	listUniverseSystems = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{30000142}, nil
	}

	handler := listUniverseSystemsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseSystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 1 || out.IDs[0] != 30000142 {
		t.Errorf("out = %+v, want IDs=[30000142]", out)
	}
}

func TestListUniverseSystemsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseSystems
	t.Cleanup(func() { listUniverseSystems = orig })
	wantErr := errors.New("upstream failure")
	listUniverseSystems = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseSystemsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseSystemsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_system ---

func TestGetUniverseSystemHandler_MapsFields(t *testing.T) {
	orig := getUniverseSystem
	t.Cleanup(func() { getUniverseSystem = orig })
	starID := int64(40000001)
	secClass := "B"
	getUniverseSystem = func(ctx context.Context, userAgent string, systemID int64) (UniverseSystem, error) {
		return UniverseSystem{
			SystemID: 30000142, ConstellationID: 20000020, Name: "Jita",
			Position: Position{X: 1, Y: 2, Z: 3}, SecurityStatus: 0.9459, SecurityClass: &secClass,
			StarID: &starID, StargateIDs: []int64{50000342}, StationIDs: []int64{60003760},
			Planets: []UniverseSystemPlanet{{PlanetID: 40000004, AsteroidBeltIDs: []int64{40000002}, MoonIDs: []int64{40000003}}},
		}, nil
	}

	handler := getUniverseSystemHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseSystemInput{SystemID: 30000142})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SystemID != 30000142 || out.Name != "Jita" || out.ConstellationID != 20000020 {
		t.Errorf("out = %+v, want SystemID=30000142 Name=Jita ConstellationID=20000020", out)
	}
	if out.SecurityStatus != 0.9459 || out.SecurityClass == nil || *out.SecurityClass != "B" {
		t.Errorf("SecurityStatus/Class = %v/%v, want 0.9459/B", out.SecurityStatus, out.SecurityClass)
	}
	if len(out.Planets) != 1 || out.Planets[0].PlanetID != 40000004 || len(out.Planets[0].AsteroidBeltIDs) != 1 || len(out.Planets[0].MoonIDs) != 1 {
		t.Errorf("Planets = %+v, want one entry PlanetID=40000004 with one asteroid belt and one moon", out.Planets)
	}
}

func TestGetUniverseSystemHandler_PropagatesError(t *testing.T) {
	orig := getUniverseSystem
	t.Cleanup(func() { getUniverseSystem = orig })
	wantErr := errors.New("upstream failure")
	getUniverseSystem = func(ctx context.Context, userAgent string, systemID int64) (UniverseSystem, error) {
		return UniverseSystem{}, wantErr
	}

	handler := getUniverseSystemHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseSystemInput{SystemID: 30000142})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_asteroid_belt ---

func TestGetUniverseAsteroidBeltHandler_MapsFields(t *testing.T) {
	orig := getUniverseAsteroidBelt
	t.Cleanup(func() { getUniverseAsteroidBelt = orig })
	getUniverseAsteroidBelt = func(ctx context.Context, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
		return UniverseAsteroidBelt{Name: "Jita IV - Asteroid Belt 1", Position: Position{X: 1, Y: 2, Z: 3}, SystemID: 30000142}, nil
	}

	handler := getUniverseAsteroidBeltHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseAsteroidBeltInput{AsteroidBeltID: 40000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Name != "Jita IV - Asteroid Belt 1" || out.SystemID != 30000142 {
		t.Errorf("out = %+v, want Name=Jita IV - Asteroid Belt 1 SystemID=30000142", out)
	}
}

func TestGetUniverseAsteroidBeltHandler_PropagatesError(t *testing.T) {
	orig := getUniverseAsteroidBelt
	t.Cleanup(func() { getUniverseAsteroidBelt = orig })
	wantErr := errors.New("upstream failure")
	getUniverseAsteroidBelt = func(ctx context.Context, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
		return UniverseAsteroidBelt{}, wantErr
	}

	handler := getUniverseAsteroidBeltHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseAsteroidBeltInput{AsteroidBeltID: 40000002})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_moon ---

func TestGetUniverseMoonHandler_MapsFields(t *testing.T) {
	orig := getUniverseMoon
	t.Cleanup(func() { getUniverseMoon = orig })
	getUniverseMoon = func(ctx context.Context, userAgent string, moonID int64) (UniverseMoon, error) {
		return UniverseMoon{MoonID: 40000003, SystemID: 30000142, Name: "Jita IV - Moon 1", Position: Position{X: 1, Y: 2, Z: 3}}, nil
	}

	handler := getUniverseMoonHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseMoonInput{MoonID: 40000003})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.MoonID != 40000003 || out.SystemID != 30000142 || out.Name != "Jita IV - Moon 1" {
		t.Errorf("out = %+v, want MoonID=40000003 SystemID=30000142 Name=Jita IV - Moon 1", out)
	}
}

func TestGetUniverseMoonHandler_PropagatesError(t *testing.T) {
	orig := getUniverseMoon
	t.Cleanup(func() { getUniverseMoon = orig })
	wantErr := errors.New("upstream failure")
	getUniverseMoon = func(ctx context.Context, userAgent string, moonID int64) (UniverseMoon, error) {
		return UniverseMoon{}, wantErr
	}

	handler := getUniverseMoonHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseMoonInput{MoonID: 40000003})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_planet ---

func TestGetUniversePlanetHandler_MapsFields(t *testing.T) {
	orig := getUniversePlanet
	t.Cleanup(func() { getUniversePlanet = orig })
	getUniversePlanet = func(ctx context.Context, userAgent string, planetID int64) (UniversePlanet, error) {
		return UniversePlanet{PlanetID: 40000004, SystemID: 30000142, TypeID: 2016, Name: "Jita IV", Position: Position{X: 1, Y: 2, Z: 3}}, nil
	}

	handler := getUniversePlanetHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniversePlanetInput{PlanetID: 40000004})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.PlanetID != 40000004 || out.SystemID != 30000142 || out.TypeID != 2016 || out.Name != "Jita IV" {
		t.Errorf("out = %+v, want PlanetID=40000004 SystemID=30000142 TypeID=2016 Name=Jita IV", out)
	}
}

func TestGetUniversePlanetHandler_PropagatesError(t *testing.T) {
	orig := getUniversePlanet
	t.Cleanup(func() { getUniversePlanet = orig })
	wantErr := errors.New("upstream failure")
	getUniversePlanet = func(ctx context.Context, userAgent string, planetID int64) (UniversePlanet, error) {
		return UniversePlanet{}, wantErr
	}

	handler := getUniversePlanetHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniversePlanetInput{PlanetID: 40000004})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_stargate ---

func TestGetUniverseStargateHandler_MapsFields(t *testing.T) {
	orig := getUniverseStargate
	t.Cleanup(func() { getUniverseStargate = orig })
	getUniverseStargate = func(ctx context.Context, userAgent string, stargateID int64) (UniverseStargate, error) {
		return UniverseStargate{
			StargateID: 50000342, SystemID: 30000142, TypeID: 16, Name: "Stargate (Perimeter)",
			Position: Position{X: 1, Y: 2, Z: 3}, DestinationStargateID: 50000343, DestinationSystemID: 30000144,
		}, nil
	}

	handler := getUniverseStargateHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseStargateInput{StargateID: 50000342})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.StargateID != 50000342 || out.SystemID != 30000142 || out.TypeID != 16 {
		t.Errorf("out = %+v, want StargateID=50000342 SystemID=30000142 TypeID=16", out)
	}
	if out.DestinationStargateID != 50000343 || out.DestinationSystemID != 30000144 {
		t.Errorf("Destination = %d/%d, want 50000343/30000144", out.DestinationStargateID, out.DestinationSystemID)
	}
}

func TestGetUniverseStargateHandler_PropagatesError(t *testing.T) {
	orig := getUniverseStargate
	t.Cleanup(func() { getUniverseStargate = orig })
	wantErr := errors.New("upstream failure")
	getUniverseStargate = func(ctx context.Context, userAgent string, stargateID int64) (UniverseStargate, error) {
		return UniverseStargate{}, wantErr
	}

	handler := getUniverseStargateHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseStargateInput{StargateID: 50000342})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_star ---

func TestGetUniverseStarHandler_MapsFields(t *testing.T) {
	orig := getUniverseStar
	t.Cleanup(func() { getUniverseStar = orig })
	getUniverseStar = func(ctx context.Context, userAgent string, starID int64) (UniverseStar, error) {
		return UniverseStar{
			StarID: 40000001, Age: 15000000000, Luminosity: 0.02, Name: "Jita - Star", Radius: 100000,
			SolarSystemID: 30000142, SpectralClass: "K2 V", Temperature: 4465, TypeID: 3796,
		}, nil
	}

	handler := getUniverseStarHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseStarInput{StarID: 40000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.StarID != 40000001 || out.SolarSystemID != 30000142 || out.Name != "Jita - Star" || out.SpectralClass != "K2 V" {
		t.Errorf("out = %+v, want StarID=40000001 SolarSystemID=30000142 Name=Jita - Star SpectralClass=K2 V", out)
	}
}

func TestGetUniverseStarHandler_PropagatesError(t *testing.T) {
	orig := getUniverseStar
	t.Cleanup(func() { getUniverseStar = orig })
	wantErr := errors.New("upstream failure")
	getUniverseStar = func(ctx context.Context, userAgent string, starID int64) (UniverseStar, error) {
		return UniverseStar{}, wantErr
	}

	handler := getUniverseStarHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseStarInput{StarID: 40000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_station ---

func TestGetUniverseStationHandler_MapsFields(t *testing.T) {
	orig := getUniverseStation
	t.Cleanup(func() { getUniverseStation = orig })
	owner := int64(1000035)
	getUniverseStation = func(ctx context.Context, userAgent string, stationID int64) (UniverseStation, error) {
		return UniverseStation{
			StationID: 60003760, SystemID: 30000142, TypeID: 1529,
			Name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant", Position: Position{X: 1, Y: 2, Z: 3},
			MaxDockableShipVolume: 50000000, OfficeRentalCost: 10000, Owner: &owner,
			ReprocessingEfficiency: 0.5, ReprocessingStationsTake: 0.05, Services: []string{"market", "repair-facilities"},
		}, nil
	}

	handler := getUniverseStationHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseStationInput{StationID: 60003760})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.StationID != 60003760 || out.SystemID != 30000142 || out.TypeID != 1529 {
		t.Errorf("out = %+v, want StationID=60003760 SystemID=30000142 TypeID=1529", out)
	}
	if out.Owner == nil || *out.Owner != 1000035 {
		t.Errorf("Owner = %v, want 1000035", out.Owner)
	}
	if len(out.Services) != 2 {
		t.Errorf("Services = %v, want 2 entries", out.Services)
	}
}

func TestGetUniverseStationHandler_PropagatesError(t *testing.T) {
	orig := getUniverseStation
	t.Cleanup(func() { getUniverseStation = orig })
	wantErr := errors.New("upstream failure")
	getUniverseStation = func(ctx context.Context, userAgent string, stationID int64) (UniverseStation, error) {
		return UniverseStation{}, wantErr
	}

	handler := getUniverseStationHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseStationInput{StationID: 60003760})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_graphics ---

func TestListUniverseGraphicsHandler_MapsFields(t *testing.T) {
	orig := listUniverseGraphics
	t.Cleanup(func() { listUniverseGraphics = orig })
	listUniverseGraphics = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{1, 2, 3}, nil
	}

	handler := listUniverseGraphicsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseGraphicsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 3 || out.IDs[0] != 1 {
		t.Errorf("out = %+v, want IDs=[1 2 3]", out)
	}
}

func TestListUniverseGraphicsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseGraphics
	t.Cleanup(func() { listUniverseGraphics = orig })
	wantErr := errors.New("upstream failure")
	listUniverseGraphics = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseGraphicsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseGraphicsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_graphic ---

func TestGetUniverseGraphicHandler_MapsFields(t *testing.T) {
	orig := getUniverseGraphic
	t.Cleanup(func() { getUniverseGraphic = orig })
	graphicFile := "file.gr2"
	getUniverseGraphic = func(ctx context.Context, userAgent string, graphicID int64) (UniverseGraphic, error) {
		return UniverseGraphic{GraphicID: 1234, GraphicFile: &graphicFile}, nil
	}

	handler := getUniverseGraphicHandler("v1")
	_, out, err := handler(context.Background(), nil, GetUniverseGraphicInput{GraphicID: 1234})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.GraphicID != 1234 || out.GraphicFile == nil || *out.GraphicFile != "file.gr2" {
		t.Errorf("out = %+v, want GraphicID=1234 GraphicFile=file.gr2", out)
	}
}

func TestGetUniverseGraphicHandler_PropagatesError(t *testing.T) {
	orig := getUniverseGraphic
	t.Cleanup(func() { getUniverseGraphic = orig })
	wantErr := errors.New("upstream failure")
	getUniverseGraphic = func(ctx context.Context, userAgent string, graphicID int64) (UniverseGraphic, error) {
		return UniverseGraphic{}, wantErr
	}

	handler := getUniverseGraphicHandler("v1")
	_, _, err := handler(context.Background(), nil, GetUniverseGraphicInput{GraphicID: 1234})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_structures ---

func TestListUniverseStructuresHandler_MapsFields(t *testing.T) {
	orig := listUniverseStructures
	t.Cleanup(func() { listUniverseStructures = orig })
	var gotFilter string
	listUniverseStructures = func(ctx context.Context, userAgent string, filter string) ([]int64, error) {
		gotFilter = filter
		return []int64{1000000000001}, nil
	}

	handler := listUniverseStructuresHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseStructuresInput{Filter: "market"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotFilter != "market" {
		t.Errorf("captured filter = %q, want market", gotFilter)
	}
	if len(out.IDs) != 1 || out.IDs[0] != 1000000000001 {
		t.Errorf("out = %+v, want IDs=[1000000000001]", out)
	}
}

func TestListUniverseStructuresHandler_RejectsInvalidFilter(t *testing.T) {
	orig := listUniverseStructures
	t.Cleanup(func() { listUniverseStructures = orig })
	listUniverseStructures = func(ctx context.Context, userAgent string, filter string) ([]int64, error) {
		t.Fatal("listUniverseStructures should not be called with an invalid filter")
		return nil, nil
	}

	handler := listUniverseStructuresHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseStructuresInput{Filter: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid filter")
	}
}

func TestListUniverseStructuresHandler_PropagatesError(t *testing.T) {
	orig := listUniverseStructures
	t.Cleanup(func() { listUniverseStructures = orig })
	wantErr := errors.New("upstream failure")
	listUniverseStructures = func(ctx context.Context, userAgent string, filter string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseStructuresHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseStructuresInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_structure (private -- the only private tool in this batch) ---

func TestGetUniverseStructureHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getUniverseStructure
	t.Cleanup(func() { getUniverseStructure = orig })
	getUniverseStructure = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64) (UniverseStructure, error) {
		return UniverseStructure{Name: "Test Citadel", OwnerID: 98000001, SolarSystemID: 30000142, Position: &Position{X: 1, Y: 2, Z: 3}}, nil
	}

	handler := getUniverseStructureHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetUniverseStructureInput{CharacterID: 7, StructureID: 1000000000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Name != "Test Citadel" || out.OwnerID != 98000001 || out.SolarSystemID != 30000142 {
		t.Errorf("out = %+v, want Name=Test Citadel OwnerID=98000001 SolarSystemID=30000142", out)
	}
	if out.Position == nil || out.Position.X != 1 {
		t.Errorf("Position = %v, want X=1", out.Position)
	}
}

func TestGetUniverseStructureHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getUniverseStructure
	t.Cleanup(func() { getUniverseStructure = orig })
	getUniverseStructure = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64) (UniverseStructure, error) {
		t.Fatal("getUniverseStructure should not be called for an unknown character")
		return UniverseStructure{}, nil
	}

	handler := getUniverseStructureHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetUniverseStructureInput{CharacterID: 999, StructureID: 1000000000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetUniverseStructureHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getUniverseStructure
	t.Cleanup(func() { getUniverseStructure = orig })
	wantErr := errors.New("upstream failure")
	getUniverseStructure = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64) (UniverseStructure, error) {
		return UniverseStructure{}, wantErr
	}

	handler := getUniverseStructureHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetUniverseStructureInput{CharacterID: 7, StructureID: 1000000000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_system_jumps ---

func TestListUniverseSystemJumpsHandler_MapsFields(t *testing.T) {
	orig := listUniverseSystemJumps
	t.Cleanup(func() { listUniverseSystemJumps = orig })
	listUniverseSystemJumps = func(ctx context.Context, userAgent string) ([]UniverseSystemJumps, error) {
		return []UniverseSystemJumps{{SystemID: 30000142, ShipJumps: 10}}, nil
	}

	handler := listUniverseSystemJumpsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseSystemJumpsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.SystemJumps) != 1 || out.SystemJumps[0].SystemID != 30000142 || out.SystemJumps[0].ShipJumps != 10 {
		t.Errorf("out = %+v, want one entry SystemID=30000142 ShipJumps=10", out)
	}
}

func TestListUniverseSystemJumpsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseSystemJumps
	t.Cleanup(func() { listUniverseSystemJumps = orig })
	wantErr := errors.New("upstream failure")
	listUniverseSystemJumps = func(ctx context.Context, userAgent string) ([]UniverseSystemJumps, error) {
		return nil, wantErr
	}

	handler := listUniverseSystemJumpsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseSystemJumpsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_system_kills ---

func TestListUniverseSystemKillsHandler_MapsFields(t *testing.T) {
	orig := listUniverseSystemKills
	t.Cleanup(func() { listUniverseSystemKills = orig })
	listUniverseSystemKills = func(ctx context.Context, userAgent string) ([]UniverseSystemKills, error) {
		return []UniverseSystemKills{{SystemID: 30000142, NPCKills: 5, PodKills: 1, ShipKills: 2}}, nil
	}

	handler := listUniverseSystemKillsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseSystemKillsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.SystemKills) != 1 || out.SystemKills[0].SystemID != 30000142 || out.SystemKills[0].NPCKills != 5 {
		t.Errorf("out = %+v, want one entry SystemID=30000142 NPCKills=5", out)
	}
}

func TestListUniverseSystemKillsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseSystemKills
	t.Cleanup(func() { listUniverseSystemKills = orig })
	wantErr := errors.New("upstream failure")
	listUniverseSystemKills = func(ctx context.Context, userAgent string) ([]UniverseSystemKills, error) {
		return nil, wantErr
	}

	handler := listUniverseSystemKillsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseSystemKillsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- resolve_universe_ids ---

func TestResolveUniverseIDsHandler_MapsFields_FullFixture(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	charID := int64(12345)
	charName := "Test Pilot"
	sysID := int64(30000142)
	sysName := "Jita"
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		return ResolveUniverseIDsOutput{
			Characters: []IDNamePair{{ID: &charID, Name: &charName}},
			Systems:    []IDNamePair{{ID: &sysID, Name: &sysName}},
		}, nil
	}

	handler := resolveUniverseIDsHandler("v1")
	_, out, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: []string{"Test Pilot", "Jita"}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Characters) != 1 || out.Characters[0].ID == nil || *out.Characters[0].ID != 12345 {
		t.Errorf("Characters = %+v, want one entry ID=12345", out.Characters)
	}
	if len(out.Systems) != 1 || out.Systems[0].Name == nil || *out.Systems[0].Name != "Jita" {
		t.Errorf("Systems = %+v, want one entry Name=Jita", out.Systems)
	}
	if len(out.Alliances) != 0 || len(out.Corporations) != 0 || len(out.Factions) != 0 || len(out.Regions) != 0 || len(out.Stations) != 0 || len(out.Agents) != 0 || len(out.Constellations) != 0 || len(out.InventoryTypes) != 0 {
		t.Errorf("out = %+v, want only Characters and Systems populated", out)
	}
}

func TestResolveUniverseIDsHandler_RejectsEmptyNames(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		t.Fatal("resolveUniverseIDs should not be called with empty names")
		return ResolveUniverseIDsOutput{}, nil
	}

	handler := resolveUniverseIDsHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: []string{}})
	if err == nil {
		t.Fatal("expected error for empty names")
	}
}

func TestResolveUniverseIDsHandler_RejectsOverCapNames(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		t.Fatal("resolveUniverseIDs should not be called with over-cap names")
		return ResolveUniverseIDsOutput{}, nil
	}

	names := make([]string, 501)
	for i := range names {
		names[i] = "x"
	}
	handler := resolveUniverseIDsHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: names})
	if err == nil {
		t.Fatal("expected error for over-cap names (501 > ESI's 500 max)")
	}
}

func TestResolveUniverseIDsHandler_PropagatesError(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	wantErr := errors.New("upstream failure")
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		return ResolveUniverseIDsOutput{}, wantErr
	}

	handler := resolveUniverseIDsHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: []string{"Jita"}})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- resolve_universe_names ---

func TestResolveUniverseNamesHandler_MapsFields(t *testing.T) {
	orig := resolveUniverseNames
	t.Cleanup(func() { resolveUniverseNames = orig })
	resolveUniverseNames = func(ctx context.Context, userAgent string, ids []int64) ([]ResolvedName, error) {
		return []ResolvedName{{Category: "solar_system", ID: 30000142, Name: "Jita"}}, nil
	}

	handler := resolveUniverseNamesHandler("v1")
	_, out, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: []int64{30000142}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Names) != 1 || out.Names[0].Category != "solar_system" || out.Names[0].ID != 30000142 || out.Names[0].Name != "Jita" {
		t.Errorf("out = %+v, want one entry Category=solar_system ID=30000142 Name=Jita", out)
	}
}

func TestResolveUniverseNamesHandler_RejectsEmptyIDs(t *testing.T) {
	orig := resolveUniverseNames
	t.Cleanup(func() { resolveUniverseNames = orig })
	resolveUniverseNames = func(ctx context.Context, userAgent string, ids []int64) ([]ResolvedName, error) {
		t.Fatal("resolveUniverseNames should not be called with empty ids")
		return nil, nil
	}

	handler := resolveUniverseNamesHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: []int64{}})
	if err == nil {
		t.Fatal("expected error for empty ids")
	}
}

func TestResolveUniverseNamesHandler_RejectsOverCapIDs(t *testing.T) {
	orig := resolveUniverseNames
	t.Cleanup(func() { resolveUniverseNames = orig })
	resolveUniverseNames = func(ctx context.Context, userAgent string, ids []int64) ([]ResolvedName, error) {
		t.Fatal("resolveUniverseNames should not be called with over-cap ids")
		return nil, nil
	}

	ids := make([]int64, 1001)
	handler := resolveUniverseNamesHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: ids})
	if err == nil {
		t.Fatal("expected error for over-cap ids (1001 > ESI's 1000 max)")
	}
}

func TestResolveUniverseNamesHandler_PropagatesError(t *testing.T) {
	orig := resolveUniverseNames
	t.Cleanup(func() { resolveUniverseNames = orig })
	wantErr := errors.New("upstream failure")
	resolveUniverseNames = func(ctx context.Context, userAgent string, ids []int64) ([]ResolvedName, error) {
		return nil, wantErr
	}

	handler := resolveUniverseNamesHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: []int64{30000142}})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
