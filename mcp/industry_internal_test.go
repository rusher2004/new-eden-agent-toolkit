package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func industryTestStore(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

func TestListCharacterIndustryJobsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterIndustryJobs
	t.Cleanup(func() { listCharacterIndustryJobs = orig })
	endDate := time.Unix(1000, 0)
	startDate := time.Unix(500, 0)
	completedCharacterID := int64(101)
	completedDate := time.Unix(1500, 0)
	cost := 1234.5
	licensedRuns := int64(10)
	pauseDate := time.Unix(750, 0)
	probability := 0.75
	productTypeID := int64(950)
	successfulRuns := int64(9)
	listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return []esiapi.CharactersCharacterIdIndustryJobsGetInner{{
			ActivityId: 1, BlueprintId: 2, BlueprintLocationId: 3, BlueprintTypeId: 4,
			Duration: 3600, EndDate: endDate, FacilityId: 5, InstallerId: 6, JobId: 7,
			OutputLocationId: 8, Runs: 9, StartDate: startDate, Status: "active", StationId: 10,
			CompletedCharacterId: &completedCharacterID, CompletedDate: &completedDate, Cost: &cost,
			LicensedRuns: &licensedRuns, PauseDate: &pauseDate, Probability: &probability,
			ProductTypeId: &productTypeID, SuccessfulRuns: &successfulRuns,
		}}, nil
	}

	handler := listCharacterIndustryJobsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 7, IncludeCompleted: true})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || !out.IncludeCompleted || len(out.Jobs) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 IncludeCompleted=true with one job", out)
	}
	j := out.Jobs[0]
	if j.ActivityID != 1 || j.BlueprintID != 2 || j.BlueprintLocationID != 3 || j.BlueprintTypeID != 4 ||
		j.Duration != 3600 || !j.EndDate.Equal(endDate) || j.FacilityID != 5 || j.InstallerID != 6 ||
		j.JobID != 7 || j.OutputLocationID != 8 || j.Runs != 9 || !j.StartDate.Equal(startDate) ||
		j.Status != "active" || j.StationID != 10 || j.LocationID != 0 ||
		j.CompletedCharacterID == nil || *j.CompletedCharacterID != completedCharacterID ||
		j.CompletedDate == nil || !j.CompletedDate.Equal(completedDate) ||
		j.Cost == nil || *j.Cost != cost ||
		j.LicensedRuns == nil || *j.LicensedRuns != licensedRuns ||
		j.PauseDate == nil || !j.PauseDate.Equal(pauseDate) ||
		j.Probability == nil || *j.Probability != probability ||
		j.ProductTypeID == nil || *j.ProductTypeID != productTypeID ||
		j.SuccessfulRuns == nil || *j.SuccessfulRuns != successfulRuns {
		t.Errorf("job = %+v, want ActivityID=1 BlueprintID=2 BlueprintLocationID=3 BlueprintTypeID=4 Duration=3600 EndDate=%v FacilityID=5 InstallerID=6 JobID=7 OutputLocationID=8 Runs=9 StartDate=%v Status=active StationID=10 LocationID=0 CompletedCharacterID=%d CompletedDate=%v Cost=%v LicensedRuns=%d PauseDate=%v Probability=%v ProductTypeID=%d SuccessfulRuns=%d",
			j, endDate, startDate, completedCharacterID, completedDate, cost, licensedRuns, pauseDate, probability, productTypeID, successfulRuns)
	}
}

func TestListCharacterIndustryJobsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterIndustryJobs
	t.Cleanup(func() { listCharacterIndustryJobs = orig })
	listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		t.Fatal("listCharacterIndustryJobs should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterIndustryJobsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterMiningHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMining
	t.Cleanup(func() { listCharacterMining = orig })
	listCharacterMining = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdMiningGetInner{{
			Date: "2023-01-01", Quantity: 100, SolarSystemId: 30000142, TypeId: 1230,
		}}, 5, nil
	}

	handler := listCharacterMiningHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMiningInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.Page != 1 || out.TotalPages != 5 || len(out.Ledger) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 Page=1 TotalPages=5 with one ledger entry", out)
	}
	e := out.Ledger[0]
	if e.Date != "2023-01-01" || e.Quantity != 100 || e.SolarSystemID != 30000142 || e.TypeID != 1230 {
		t.Errorf("ledger entry = %+v, want Date=2023-01-01 Quantity=100 SolarSystemID=30000142 TypeID=1230", e)
	}
}

func TestListCharacterMiningHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterMining
	t.Cleanup(func() { listCharacterMining = orig })
	listCharacterMining = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		t.Fatal("listCharacterMining should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCharacterMiningHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMiningInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMiningExtractionsHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMiningExtractions
	t.Cleanup(func() { listCorporationMiningExtractions = orig })
	chunkArrivalTime := time.Unix(1000, 0)
	extractionStartTime := time.Unix(500, 0)
	naturalDecayTime := time.Unix(2000, 0)
	listCorporationMiningExtractions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningExtractionsGetInner{{
			MoonId: 40000001, StructureId: 1000000000001, ChunkArrivalTime: chunkArrivalTime,
			ExtractionStartTime: extractionStartTime, NaturalDecayTime: naturalDecayTime,
		}}, 3, nil
	}

	handler := listCorporationMiningExtractionsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMiningExtractionsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 3 || len(out.Extractions) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=3 with one extraction", out)
	}
	ex := out.Extractions[0]
	if ex.MoonID != 40000001 || ex.StructureID != 1000000000001 || !ex.ChunkArrivalTime.Equal(chunkArrivalTime) ||
		!ex.ExtractionStartTime.Equal(extractionStartTime) || !ex.NaturalDecayTime.Equal(naturalDecayTime) {
		t.Errorf("extraction = %+v, want MoonID=40000001 StructureID=1000000000001 ChunkArrivalTime=%v ExtractionStartTime=%v NaturalDecayTime=%v",
			ex, chunkArrivalTime, extractionStartTime, naturalDecayTime)
	}
}

func TestListCorporationMiningExtractionsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMiningExtractions
	t.Cleanup(func() { listCorporationMiningExtractions = orig })
	listCorporationMiningExtractions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
		t.Fatal("listCorporationMiningExtractions should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMiningExtractionsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMiningExtractionsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMiningObserversHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMiningObservers
	t.Cleanup(func() { listCorporationMiningObservers = orig })
	listCorporationMiningObservers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningObserversGetInner{{
			LastUpdated: "2023-01-01", ObserverId: 900, ObserverType: "structure",
		}}, 2, nil
	}

	handler := listCorporationMiningObserversHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMiningObserversInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Observers) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one observer", out)
	}
	o := out.Observers[0]
	if o.ObserverID != 900 || o.LastUpdated != "2023-01-01" || o.ObserverType != "structure" {
		t.Errorf("observer = %+v, want ObserverID=900 LastUpdated=2023-01-01 ObserverType=structure", o)
	}
}

func TestListCorporationMiningObserversHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMiningObservers
	t.Cleanup(func() { listCorporationMiningObservers = orig })
	listCorporationMiningObservers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
		t.Fatal("listCorporationMiningObservers should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMiningObserversHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMiningObserversInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationMiningObserverDetailHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationMiningObserverDetail
	t.Cleanup(func() { getCorporationMiningObserverDetail = orig })
	getCorporationMiningObserverDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner{{
			CharacterId: 100, LastUpdated: "2023-01-01", Quantity: 500, RecordedCorporationId: 98000001, TypeId: 1230,
		}}, 4, nil
	}

	handler := getCorporationMiningObserverDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationMiningObserverDetailInput{CharacterID: 7, CorporationID: 98000001, ObserverID: 900})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.ObserverID != 900 || out.Page != 1 || out.TotalPages != 4 || len(out.Entries) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 ObserverID=900 Page=1 TotalPages=4 with one entry", out)
	}
	e := out.Entries[0]
	if e.CharacterID != 100 || e.LastUpdated != "2023-01-01" || e.Quantity != 500 || e.RecordedCorporationID != 98000001 || e.TypeID != 1230 {
		t.Errorf("entry = %+v, want CharacterID=100 LastUpdated=2023-01-01 Quantity=500 RecordedCorporationID=98000001 TypeID=1230", e)
	}
}

func TestGetCorporationMiningObserverDetailHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationMiningObserverDetail
	t.Cleanup(func() { getCorporationMiningObserverDetail = orig })
	getCorporationMiningObserverDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
		t.Fatal("getCorporationMiningObserverDetail should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := getCorporationMiningObserverDetailHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationMiningObserverDetailInput{CharacterID: 999, CorporationID: 98000001, ObserverID: 900})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationIndustryJobsHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationIndustryJobs
	t.Cleanup(func() { listCorporationIndustryJobs = orig })
	endDate := time.Unix(1000, 0)
	startDate := time.Unix(500, 0)
	completedCharacterID := int64(101)
	completedDate := time.Unix(1500, 0)
	cost := 1234.5
	licensedRuns := int64(10)
	pauseDate := time.Unix(750, 0)
	probability := 0.75
	productTypeID := int64(950)
	successfulRuns := int64(9)
	listCorporationIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdIndustryJobsGetInner{{
			ActivityId: 1, BlueprintId: 2, BlueprintLocationId: 3, BlueprintTypeId: 4,
			Duration: 3600, EndDate: endDate, FacilityId: 5, InstallerId: 6, JobId: 7,
			LocationId: 11, OutputLocationId: 8, Runs: 9, StartDate: startDate, Status: "active",
			CompletedCharacterId: &completedCharacterID, CompletedDate: &completedDate, Cost: &cost,
			LicensedRuns: &licensedRuns, PauseDate: &pauseDate, Probability: &probability,
			ProductTypeId: &productTypeID, SuccessfulRuns: &successfulRuns,
		}}, 6, nil
	}

	handler := listCorporationIndustryJobsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 7, CorporationID: 98000001, IncludeCompleted: true})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || !out.IncludeCompleted || out.Page != 1 || out.TotalPages != 6 || len(out.Jobs) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 IncludeCompleted=true Page=1 TotalPages=6 with one job", out)
	}
	j := out.Jobs[0]
	if j.ActivityID != 1 || j.BlueprintID != 2 || j.BlueprintLocationID != 3 || j.BlueprintTypeID != 4 ||
		j.Duration != 3600 || !j.EndDate.Equal(endDate) || j.FacilityID != 5 || j.InstallerID != 6 ||
		j.JobID != 7 || j.LocationID != 11 || j.StationID != 0 || j.OutputLocationID != 8 || j.Runs != 9 ||
		!j.StartDate.Equal(startDate) || j.Status != "active" ||
		j.CompletedCharacterID == nil || *j.CompletedCharacterID != completedCharacterID ||
		j.CompletedDate == nil || !j.CompletedDate.Equal(completedDate) ||
		j.Cost == nil || *j.Cost != cost ||
		j.LicensedRuns == nil || *j.LicensedRuns != licensedRuns ||
		j.PauseDate == nil || !j.PauseDate.Equal(pauseDate) ||
		j.Probability == nil || *j.Probability != probability ||
		j.ProductTypeID == nil || *j.ProductTypeID != productTypeID ||
		j.SuccessfulRuns == nil || *j.SuccessfulRuns != successfulRuns {
		t.Errorf("job = %+v, want ActivityID=1 BlueprintID=2 BlueprintLocationID=3 BlueprintTypeID=4 Duration=3600 EndDate=%v FacilityID=5 InstallerID=6 JobID=7 LocationID=11 StationID=0 OutputLocationID=8 Runs=9 StartDate=%v Status=active CompletedCharacterID=%d CompletedDate=%v Cost=%v LicensedRuns=%d PauseDate=%v Probability=%v ProductTypeID=%d SuccessfulRuns=%d",
			j, endDate, startDate, completedCharacterID, completedDate, cost, licensedRuns, pauseDate, probability, productTypeID, successfulRuns)
	}
}

func TestListCorporationIndustryJobsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationIndustryJobs
	t.Cleanup(func() { listCorporationIndustryJobs = orig })
	listCorporationIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		t.Fatal("listCorporationIndustryJobs should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationIndustryJobsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListIndustryFacilitiesHandler_MapsFields(t *testing.T) {
	orig := listIndustryFacilities
	t.Cleanup(func() { listIndustryFacilities = orig })
	tax := 0.1
	listIndustryFacilities = func(ctx context.Context, userAgent string) ([]esiapi.IndustryFacilitiesGetInner, error) {
		return []esiapi.IndustryFacilitiesGetInner{{
			FacilityId: 500, OwnerId: 98000001, RegionId: 10000002, SolarSystemId: 30000142, TypeId: 2502, Tax: &tax,
		}}, nil
	}

	handler := listIndustryFacilitiesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIndustryFacilitiesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Facilities) != 1 {
		t.Fatalf("out = %+v, want one facility", out)
	}
	f := out.Facilities[0]
	if f.FacilityID != 500 || f.OwnerID != 98000001 || f.RegionID != 10000002 || f.SolarSystemID != 30000142 ||
		f.TypeID != 2502 || f.Tax == nil || *f.Tax != tax {
		t.Errorf("facility = %+v, want FacilityID=500 OwnerID=98000001 RegionID=10000002 SolarSystemID=30000142 TypeID=2502 Tax=%v", f, tax)
	}
}

func TestListIndustrySystemsHandler_MapsFields(t *testing.T) {
	orig := listIndustrySystems
	t.Cleanup(func() { listIndustrySystems = orig })
	listIndustrySystems = func(ctx context.Context, userAgent string) ([]esiapi.IndustrySystemsGetInner, error) {
		return []esiapi.IndustrySystemsGetInner{{
			SolarSystemId: 30000142,
			CostIndices: []esiapi.IndustrySystemsGetInnerCostIndicesInner{
				{Activity: "manufacturing", CostIndex: 0.05},
				{Activity: "invention", CostIndex: 0.02},
			},
		}}, nil
	}

	handler := listIndustrySystemsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListIndustrySystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Systems) != 1 {
		t.Fatalf("out = %+v, want one system", out)
	}
	s := out.Systems[0]
	if s.SolarSystemID != 30000142 || len(s.CostIndices) != 2 {
		t.Fatalf("system = %+v, want SolarSystemID=30000142 with two cost indices", s)
	}
	if s.CostIndices[0].Activity != "manufacturing" || s.CostIndices[0].CostIndex != 0.05 ||
		s.CostIndices[1].Activity != "invention" || s.CostIndices[1].CostIndex != 0.02 {
		t.Errorf("cost indices = %+v, want [manufacturing 0.05, invention 0.02]", s.CostIndices)
	}
}

func TestListCharacterIndustryJobsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterIndustryJobs
	t.Cleanup(func() { listCharacterIndustryJobs = orig })
	wantErr := errors.New("upstream failure")
	listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return nil, wantErr
	}

	handler := listCharacterIndustryJobsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterMiningHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMining
	t.Cleanup(func() { listCharacterMining = orig })
	wantErr := errors.New("upstream failure")
	listCharacterMining = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCharacterMiningHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMiningInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMiningExtractionsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMiningExtractions
	t.Cleanup(func() { listCorporationMiningExtractions = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMiningExtractions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationMiningExtractionsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMiningExtractionsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMiningObserversHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMiningObservers
	t.Cleanup(func() { listCorporationMiningObservers = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMiningObservers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationMiningObserversHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMiningObserversInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationMiningObserverDetailHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationMiningObserverDetail
	t.Cleanup(func() { getCorporationMiningObserverDetail = orig })
	wantErr := errors.New("upstream failure")
	getCorporationMiningObserverDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := getCorporationMiningObserverDetailHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationMiningObserverDetailInput{CharacterID: 7, CorporationID: 98000001, ObserverID: 900})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationIndustryJobsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationIndustryJobs
	t.Cleanup(func() { listCorporationIndustryJobs = orig })
	wantErr := errors.New("upstream failure")
	listCorporationIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationIndustryJobsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListIndustryFacilitiesHandler_PropagatesError(t *testing.T) {
	orig := listIndustryFacilities
	t.Cleanup(func() { listIndustryFacilities = orig })
	wantErr := errors.New("upstream failure")
	listIndustryFacilities = func(ctx context.Context, userAgent string) ([]esiapi.IndustryFacilitiesGetInner, error) {
		return nil, wantErr
	}

	handler := listIndustryFacilitiesHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListIndustryFacilitiesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListIndustrySystemsHandler_PropagatesError(t *testing.T) {
	orig := listIndustrySystems
	t.Cleanup(func() { listIndustrySystems = orig })
	wantErr := errors.New("upstream failure")
	listIndustrySystems = func(ctx context.Context, userAgent string) ([]esiapi.IndustrySystemsGetInner, error) {
		return nil, wantErr
	}

	handler := listIndustrySystemsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListIndustrySystemsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterIndustryJobsHandler_ResolvesTypeNames(t *testing.T) {
	store := industryTestStore(t)

	orig := listCharacterIndustryJobs
	t.Cleanup(func() { listCharacterIndustryJobs = orig })
	productTypeID := int64(588)
	listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return []esiapi.CharactersCharacterIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587, ProductTypeId: &productTypeID}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{587: "Rifter Blueprint", 588: "Rifter"}
	}

	handler := listCharacterIndustryJobsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 2 || gotIDs[0] != 587 || gotIDs[1] != 588 {
		t.Errorf("resolveTypeNames got ids = %v, want [587 588]", gotIDs)
	}
	if len(out.Jobs) != 1 {
		t.Fatalf("out.Jobs = %+v, want one job", out.Jobs)
	}
	j := out.Jobs[0]
	if j.BlueprintTypeName != "Rifter Blueprint" {
		t.Errorf("BlueprintTypeName = %q, want %q", j.BlueprintTypeName, "Rifter Blueprint")
	}
	if j.ProductTypeName != "Rifter" {
		t.Errorf("ProductTypeName = %q, want %q", j.ProductTypeName, "Rifter")
	}
}

func TestListCharacterIndustryJobsHandler_NilProductTypeIDNotEnriched(t *testing.T) {
	store := industryTestStore(t)

	orig := listCharacterIndustryJobs
	t.Cleanup(func() { listCharacterIndustryJobs = orig })
	listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return []esiapi.CharactersCharacterIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587, ProductTypeId: nil}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{587: "Rifter Blueprint"}
	}

	handler := listCharacterIndustryJobsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 587 {
		t.Errorf("resolveTypeNames got ids = %v, want [587] (no nil ProductTypeID dereferenced)", gotIDs)
	}
	if len(out.Jobs) != 1 {
		t.Fatalf("out.Jobs = %+v, want one job", out.Jobs)
	}
	j := out.Jobs[0]
	if j.BlueprintTypeName != "Rifter Blueprint" {
		t.Errorf("BlueprintTypeName = %q, want %q", j.BlueprintTypeName, "Rifter Blueprint")
	}
	if j.ProductTypeID != nil {
		t.Errorf("ProductTypeID = %v, want nil", j.ProductTypeID)
	}
	if j.ProductTypeName != "" {
		t.Errorf("ProductTypeName = %q, want empty (no product type to resolve)", j.ProductTypeName)
	}
}

func TestListCorporationIndustryJobsHandler_ResolvesTypeNames(t *testing.T) {
	store := industryTestStore(t)

	orig := listCorporationIndustryJobs
	t.Cleanup(func() { listCorporationIndustryJobs = orig })
	productTypeID := int64(588)
	listCorporationIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587, ProductTypeId: &productTypeID}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter Blueprint", 588: "Rifter"}
	}

	handler := listCorporationIndustryJobsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Jobs) != 1 {
		t.Fatalf("out.Jobs = %+v, want one job", out.Jobs)
	}
	j := out.Jobs[0]
	if j.BlueprintTypeName != "Rifter Blueprint" {
		t.Errorf("BlueprintTypeName = %q, want %q", j.BlueprintTypeName, "Rifter Blueprint")
	}
	if j.ProductTypeName != "Rifter" {
		t.Errorf("ProductTypeName = %q, want %q", j.ProductTypeName, "Rifter")
	}
}

func TestListCharacterMiningHandler_ResolvesTypeName(t *testing.T) {
	store := industryTestStore(t)

	orig := listCharacterMining
	t.Cleanup(func() { listCharacterMining = orig })
	listCharacterMining = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdMiningGetInner{{Date: "2023-01-01", Quantity: 100, SolarSystemId: 30000142, TypeId: 1230}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1230: "Veldspar"}
	}

	handler := listCharacterMiningHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMiningInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Ledger) != 1 {
		t.Fatalf("out.Ledger = %+v, want one entry", out.Ledger)
	}
	if out.Ledger[0].TypeName != "Veldspar" {
		t.Errorf("TypeName = %q, want %q", out.Ledger[0].TypeName, "Veldspar")
	}
}

func TestGetCorporationMiningObserverDetailHandler_ResolvesTypeName(t *testing.T) {
	store := industryTestStore(t)

	orig := getCorporationMiningObserverDetail
	t.Cleanup(func() { getCorporationMiningObserverDetail = orig })
	getCorporationMiningObserverDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner{{
			CharacterId: 100, LastUpdated: "2023-01-01", Quantity: 500, RecordedCorporationId: 98000001, TypeId: 1230,
		}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1230: "Veldspar"}
	}

	handler := getCorporationMiningObserverDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationMiningObserverDetailInput{CharacterID: 7, CorporationID: 98000001, ObserverID: 900})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Entries) != 1 {
		t.Fatalf("out.Entries = %+v, want one entry", out.Entries)
	}
	if out.Entries[0].TypeName != "Veldspar" {
		t.Errorf("TypeName = %q, want %q", out.Entries[0].TypeName, "Veldspar")
	}
}

func TestListIndustryFacilitiesHandler_ResolvesTypeName(t *testing.T) {
	orig := listIndustryFacilities
	t.Cleanup(func() { listIndustryFacilities = orig })
	listIndustryFacilities = func(ctx context.Context, userAgent string) ([]esiapi.IndustryFacilitiesGetInner, error) {
		return []esiapi.IndustryFacilitiesGetInner{{
			FacilityId: 500, OwnerId: 98000001, RegionId: 10000002, SolarSystemId: 30000142, TypeId: 2502,
		}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{2502: "Standup Manufacturing Plant I"}
	}

	handler := listIndustryFacilitiesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIndustryFacilitiesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Facilities) != 1 {
		t.Fatalf("out.Facilities = %+v, want one facility", out.Facilities)
	}
	if out.Facilities[0].TypeName != "Standup Manufacturing Plant I" {
		t.Errorf("TypeName = %q, want %q", out.Facilities[0].TypeName, "Standup Manufacturing Plant I")
	}
}
