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
	fake := fakeCharacterIndustryJobsESI{characterIndustryJobs: func(_ context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return []esiapi.CharactersCharacterIdIndustryJobsGetInner{{
			ActivityId: 1, BlueprintId: 2, BlueprintLocationId: 3, BlueprintTypeId: 4,
			Duration: 3600, EndDate: endDate, FacilityId: 5, InstallerId: 6, JobId: 7,
			OutputLocationId: 8, Runs: 9, StartDate: startDate, Status: "active", StationId: 10,
			CompletedCharacterId: &completedCharacterID, CompletedDate: &completedDate, Cost: &cost,
			LicensedRuns: &licensedRuns, PauseDate: &pauseDate, Probability: &probability,
			ProductTypeId: &productTypeID, SuccessfulRuns: &successfulRuns,
		}}, nil
	}}

	handler := listCharacterIndustryJobsHandler("v1", openNarrow[characterIndustryJobsESI](fake), store, nil)
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

	handler := listCharacterIndustryJobsHandler("v1", noNarrow[characterIndustryJobsESI](t), store, nil)
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

	fake := fakeCharacterMiningESI{characterMining: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdMiningGetInner{{
			Date: "2023-01-01", Quantity: 100, SolarSystemId: 30000142, TypeId: 1230,
		}}, 5, nil
	}}

	handler := listCharacterMiningHandler("v1", openNarrow[characterMiningESI](fake), store, nil)
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

	handler := listCharacterMiningHandler("v1", noNarrow[characterMiningESI](t), store, nil)
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

	chunkArrivalTime := time.Unix(1000, 0)
	extractionStartTime := time.Unix(500, 0)
	naturalDecayTime := time.Unix(2000, 0)
	fake := fakeCorporationMiningExtractionsESI{corporationMiningExtractions: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningExtractionsGetInner{{
			MoonId: 40000001, StructureId: 1000000000001, ChunkArrivalTime: chunkArrivalTime,
			ExtractionStartTime: extractionStartTime, NaturalDecayTime: naturalDecayTime,
		}}, 3, nil
	}}

	handler := listCorporationMiningExtractionsHandler("v1", openNarrow[corporationMiningExtractionsESI](fake), store, nil)
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

	handler := listCorporationMiningExtractionsHandler("v1", noNarrow[corporationMiningExtractionsESI](t), store, nil)
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

	fake := fakeCorporationMiningObserversESI{corporationMiningObservers: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningObserversGetInner{{
			LastUpdated: "2023-01-01", ObserverId: 900, ObserverType: "structure",
		}}, 2, nil
	}}

	handler := listCorporationMiningObserversHandler(openNarrow[corporationMiningObserversESI](fake), store)
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

	handler := listCorporationMiningObserversHandler(noNarrow[corporationMiningObserversESI](t), store)
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

	fake := fakeCorporationMiningObserverDetailESI{corporationMiningObserverDetail: func(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner{{
			CharacterId: 100, LastUpdated: "2023-01-01", Quantity: 500, RecordedCorporationId: 98000001, TypeId: 1230,
		}}, 4, nil
	}}

	handler := getCorporationMiningObserverDetailHandler("v1", openNarrow[corporationMiningObserverDetailESI](fake), store, nil)
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

	handler := getCorporationMiningObserverDetailHandler("v1", noNarrow[corporationMiningObserverDetailESI](t), store, nil)
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
	fake := fakeCorporationIndustryJobsESI{corporationIndustryJobs: func(_ context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdIndustryJobsGetInner{{
			ActivityId: 1, BlueprintId: 2, BlueprintLocationId: 3, BlueprintTypeId: 4,
			Duration: 3600, EndDate: endDate, FacilityId: 5, InstallerId: 6, JobId: 7,
			LocationId: 11, OutputLocationId: 8, Runs: 9, StartDate: startDate, Status: "active",
			CompletedCharacterId: &completedCharacterID, CompletedDate: &completedDate, Cost: &cost,
			LicensedRuns: &licensedRuns, PauseDate: &pauseDate, Probability: &probability,
			ProductTypeId: &productTypeID, SuccessfulRuns: &successfulRuns,
		}}, 6, nil
	}}

	handler := listCorporationIndustryJobsHandler("v1", openNarrow[corporationIndustryJobsESI](fake), store, nil)
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

	handler := listCorporationIndustryJobsHandler("v1", noNarrow[corporationIndustryJobsESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListIndustryFacilitiesHandler_MapsFields(t *testing.T) {
	tax := 0.1
	fake := fakeIndustryFacilitiesESI{industryFacilities: func(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error) {
		return []esiapi.IndustryFacilitiesGetInner{{
			FacilityId: 500, OwnerId: 98000001, RegionId: 10000002, SolarSystemId: 30000142, TypeId: 2502, Tax: &tax,
		}}, nil
	}}

	handler := listIndustryFacilitiesHandler("v1", publicNarrow[industryFacilitiesESI](fake), nil)
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
	fake := fakeIndustrySystemsESI{industrySystems: func(_ context.Context) ([]esiapi.IndustrySystemsGetInner, error) {
		return []esiapi.IndustrySystemsGetInner{{
			SolarSystemId: 30000142,
			CostIndices: []esiapi.IndustrySystemsGetInnerCostIndicesInner{
				{Activity: "manufacturing", CostIndex: 0.05},
				{Activity: "invention", CostIndex: 0.02},
			},
		}}, nil
	}}

	handler := listIndustrySystemsHandler("v1", publicNarrow[industrySystemsESI](fake), nil)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterIndustryJobsESI{characterIndustryJobs: func(_ context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return nil, wantErr
	}}

	handler := listCharacterIndustryJobsHandler("v1", openNarrow[characterIndustryJobsESI](fake), store, nil)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterMiningESI{characterMining: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCharacterMiningHandler("v1", openNarrow[characterMiningESI](fake), store, nil)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationMiningExtractionsESI{corporationMiningExtractions: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationMiningExtractionsHandler("v1", openNarrow[corporationMiningExtractionsESI](fake), store, nil)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationMiningObserversESI{corporationMiningObservers: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationMiningObserversHandler(openNarrow[corporationMiningObserversESI](fake), store)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationMiningObserverDetailESI{corporationMiningObserverDetail: func(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := getCorporationMiningObserverDetailHandler("v1", openNarrow[corporationMiningObserverDetailESI](fake), store, nil)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationIndustryJobsESI{corporationIndustryJobs: func(_ context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationIndustryJobsHandler("v1", openNarrow[corporationIndustryJobsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListIndustryFacilitiesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeIndustryFacilitiesESI{industryFacilities: func(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error) {
		return nil, wantErr
	}}

	handler := listIndustryFacilitiesHandler("v1", publicNarrow[industryFacilitiesESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListIndustryFacilitiesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListIndustrySystemsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeIndustrySystemsESI{industrySystems: func(_ context.Context) ([]esiapi.IndustrySystemsGetInner, error) {
		return nil, wantErr
	}}

	handler := listIndustrySystemsHandler("v1", publicNarrow[industrySystemsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListIndustrySystemsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterIndustryJobsHandler_ResolvesTypeNames(t *testing.T) {
	store := industryTestStore(t)

	productTypeID := int64(588)
	fake := fakeCharacterIndustryJobsESI{characterIndustryJobs: func(_ context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return []esiapi.CharactersCharacterIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587, ProductTypeId: &productTypeID}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{587: "Rifter Blueprint", 588: "Rifter"}
	}

	handler := listCharacterIndustryJobsHandler("v1", openNarrow[characterIndustryJobsESI](fake), store, nil)
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

	fake := fakeCharacterIndustryJobsESI{characterIndustryJobs: func(_ context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return []esiapi.CharactersCharacterIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587, ProductTypeId: nil}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{587: "Rifter Blueprint"}
	}

	handler := listCharacterIndustryJobsHandler("v1", openNarrow[characterIndustryJobsESI](fake), store, nil)
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

	productTypeID := int64(588)
	fake := fakeCorporationIndustryJobsESI{corporationIndustryJobs: func(_ context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587, ProductTypeId: &productTypeID}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter Blueprint", 588: "Rifter"}
	}

	handler := listCorporationIndustryJobsHandler("v1", openNarrow[corporationIndustryJobsESI](fake), store, nil)
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

	fake := fakeCharacterMiningESI{characterMining: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdMiningGetInner{{Date: "2023-01-01", Quantity: 100, SolarSystemId: 30000142, TypeId: 1230}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1230: "Veldspar"}
	}

	handler := listCharacterMiningHandler("v1", openNarrow[characterMiningESI](fake), store, nil)
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

	fake := fakeCorporationMiningObserverDetailESI{corporationMiningObserverDetail: func(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner{{
			CharacterId: 100, LastUpdated: "2023-01-01", Quantity: 500, RecordedCorporationId: 98000001, TypeId: 1230,
		}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1230: "Veldspar"}
	}

	handler := getCorporationMiningObserverDetailHandler("v1", openNarrow[corporationMiningObserverDetailESI](fake), store, nil)
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
	fake := fakeIndustryFacilitiesESI{industryFacilities: func(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error) {
		return []esiapi.IndustryFacilitiesGetInner{{
			FacilityId: 500, OwnerId: 98000001, RegionId: 10000002, SolarSystemId: 30000142, TypeId: 2502,
		}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{2502: "Standup Manufacturing Plant I"}
	}

	handler := listIndustryFacilitiesHandler("v1", publicNarrow[industryFacilitiesESI](fake), nil)
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

func TestListIndustryFacilitiesHandler_ResolvesLocationNames(t *testing.T) {
	fake := fakeIndustryFacilitiesESI{industryFacilities: func(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error) {
		return []esiapi.IndustryFacilitiesGetInner{{
			FacilityId: 500, OwnerId: 98000001, RegionId: 10000002, SolarSystemId: 30000142, TypeId: 2502,
		}}, nil
	}}

	origRegion := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origRegion })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listIndustryFacilitiesHandler("v1", publicNarrow[industryFacilitiesESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListIndustryFacilitiesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Facilities) != 1 {
		t.Fatalf("out.Facilities = %+v, want one facility", out.Facilities)
	}
	if out.Facilities[0].RegionName != "The Forge" {
		t.Errorf("RegionName = %q, want %q", out.Facilities[0].RegionName, "The Forge")
	}
	if out.Facilities[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Facilities[0].SolarSystemName, "Jita")
	}
}

func TestListIndustrySystemsHandler_ResolvesSystemName(t *testing.T) {
	fake := fakeIndustrySystemsESI{industrySystems: func(_ context.Context) ([]esiapi.IndustrySystemsGetInner, error) {
		return []esiapi.IndustrySystemsGetInner{{SolarSystemId: 30000142}}, nil
	}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listIndustrySystemsHandler("v1", publicNarrow[industrySystemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListIndustrySystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Systems) != 1 {
		t.Fatalf("out.Systems = %+v, want one system", out.Systems)
	}
	if out.Systems[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Systems[0].SolarSystemName, "Jita")
	}
}

func TestListCharacterMiningHandler_ResolvesSystemName(t *testing.T) {
	store := industryTestStore(t)

	fake := fakeCharacterMiningESI{characterMining: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdMiningGetInner{{Date: "2026-07-30", Quantity: 100, SolarSystemId: 30000142, TypeId: 34}}, 1, nil
	}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCharacterMiningHandler("v1", openNarrow[characterMiningESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMiningInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Ledger) != 1 {
		t.Fatalf("out.Ledger = %+v, want one entry", out.Ledger)
	}
	if out.Ledger[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Ledger[0].SolarSystemName, "Jita")
	}
}

func TestListCorporationMiningExtractionsHandler_ResolvesMoonName(t *testing.T) {
	store := industryTestStore(t)

	fake := fakeCorporationMiningExtractionsESI{corporationMiningExtractions: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
		return []esiapi.CorporationCorporationIdMiningExtractionsGetInner{{MoonId: 40009087, StructureId: 1}}, 1, nil
	}}

	origMoon := resolveMoonNames
	t.Cleanup(func() { resolveMoonNames = origMoon })
	resolveMoonNames = func(ctx context.Context, sdeStore moonNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009087: "Jita IV - Moon 4"}
	}

	handler := listCorporationMiningExtractionsHandler("v1", openNarrow[corporationMiningExtractionsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMiningExtractionsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Extractions) != 1 {
		t.Fatalf("out.Extractions = %+v, want one extraction", out.Extractions)
	}
	if out.Extractions[0].MoonName != "Jita IV - Moon 4" {
		t.Errorf("MoonName = %q, want %q", out.Extractions[0].MoonName, "Jita IV - Moon 4")
	}
}

func TestListCharacterIndustryJobsHandler_ResolvesStationName(t *testing.T) {
	store := industryTestStore(t)

	fake := fakeCharacterIndustryJobsESI{characterIndustryJobs: func(_ context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
		return []esiapi.CharactersCharacterIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587, StationId: 60003760}}, nil
	}}

	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := listCharacterIndustryJobsHandler("v1", openNarrow[characterIndustryJobsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Jobs) != 1 {
		t.Fatalf("out.Jobs = %+v, want one job", out.Jobs)
	}
	if out.Jobs[0].StationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("StationName = %q, want %q", out.Jobs[0].StationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}

func TestListCorporationIndustryJobsHandler_StationNameEmptyWhenStationIDZero(t *testing.T) {
	store := industryTestStore(t)

	fake := fakeCorporationIndustryJobsESI{corporationIndustryJobs: func(_ context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
		// StationID is never populated on the corp mapper (LocationID is used instead), so it's left zero here to match reality.
		return []esiapi.CorporationsCorporationIdIndustryJobsGetInner{{JobId: 1, BlueprintTypeId: 587}}, 1, nil
	}}

	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{0: "should never be assigned"}
	}

	handler := listCorporationIndustryJobsHandler("v1", openNarrow[corporationIndustryJobsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Jobs) != 1 {
		t.Fatalf("out.Jobs = %+v, want one job", out.Jobs)
	}
	if out.Jobs[0].StationName != "" {
		t.Errorf("StationName = %q, want empty (StationID is 0 for corp-scoped jobs)", out.Jobs[0].StationName)
	}
}

// fakeCharacterIndustryJobsESI fakes characterIndustryJobsESI; a nil characterIndustryJobs fails as an unfaked call.
type fakeCharacterIndustryJobsESI struct {
	characterIndustryJobs func(ctx context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error)
}

func (f fakeCharacterIndustryJobsESI) CharacterIndustryJobs(ctx context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
	if f.characterIndustryJobs == nil {
		return nil, errNotFaked
	}
	return f.characterIndustryJobs(ctx, includeCompleted)
}

// fakeCorporationIndustryJobsESI fakes corporationIndustryJobsESI; a nil corporationIndustryJobs fails as an unfaked call.
type fakeCorporationIndustryJobsESI struct {
	corporationIndustryJobs func(ctx context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error)
}

func (f fakeCorporationIndustryJobsESI) CorporationIndustryJobs(ctx context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
	if f.corporationIndustryJobs == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationIndustryJobs(ctx, corporationID, includeCompleted, page)
}

// fakeIndustrySystemsESI fakes industrySystemsESI; a nil industrySystems fails as an unfaked call.
type fakeIndustrySystemsESI struct {
	industrySystems func(ctx context.Context) ([]esiapi.IndustrySystemsGetInner, error)
}

func (f fakeIndustrySystemsESI) IndustrySystems(ctx context.Context) ([]esiapi.IndustrySystemsGetInner, error) {
	if f.industrySystems == nil {
		return nil, errNotFaked
	}
	return f.industrySystems(ctx)
}

// Each handler reports a client that can't be opened, wrapping the cause.
func TestIndustryJobsHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	wantErr := errors.New("no refresh token")
	for name, call := range map[string]func() error{
		"list_character_industry_jobs": func() error {
			_, _, err := listCharacterIndustryJobsHandler("v1", failNarrow[characterIndustryJobsESI](wantErr), store, nil)(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 7})
			return err
		},
		"list_corporation_industry_jobs": func() error {
			_, _, err := listCorporationIndustryJobsHandler("v1", failNarrow[corporationIndustryJobsESI](wantErr), store, nil)(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}

// fakeCharacterMiningESI fakes characterMiningESI; a nil characterMining fails as an unfaked call.
type fakeCharacterMiningESI struct {
	characterMining func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error)
}

func (f fakeCharacterMiningESI) CharacterMining(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
	if f.characterMining == nil {
		return nil, 0, errNotFaked
	}
	return f.characterMining(ctx, page)
}

// fakeCorporationMiningExtractionsESI fakes corporationMiningExtractionsESI; a nil corporationMiningExtractions fails as an unfaked call.
type fakeCorporationMiningExtractionsESI struct {
	corporationMiningExtractions func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error)
}

func (f fakeCorporationMiningExtractionsESI) CorporationMiningExtractions(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
	if f.corporationMiningExtractions == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationMiningExtractions(ctx, corporationID, page)
}

// fakeCorporationMiningObserversESI fakes corporationMiningObserversESI; a nil corporationMiningObservers fails as an unfaked call.
type fakeCorporationMiningObserversESI struct {
	corporationMiningObservers func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error)
}

func (f fakeCorporationMiningObserversESI) CorporationMiningObservers(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
	if f.corporationMiningObservers == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationMiningObservers(ctx, corporationID, page)
}

// fakeCorporationMiningObserverDetailESI fakes corporationMiningObserverDetailESI; a nil corporationMiningObserverDetail fails as an unfaked call.
type fakeCorporationMiningObserverDetailESI struct {
	corporationMiningObserverDetail func(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error)
}

func (f fakeCorporationMiningObserverDetailESI) CorporationMiningObserverDetail(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
	if f.corporationMiningObserverDetail == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationMiningObserverDetail(ctx, corporationID, observerID, page)
}

// fakeIndustryFacilitiesESI fakes industryFacilitiesESI; a nil industryFacilities fails as an unfaked call.
type fakeIndustryFacilitiesESI struct {
	industryFacilities func(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error)
}

func (f fakeIndustryFacilitiesESI) IndustryFacilities(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error) {
	if f.industryFacilities == nil {
		return nil, errNotFaked
	}
	return f.industryFacilities(ctx)
}

// Each mining handler reports a client that can't be opened, wrapping the cause.
func TestMiningHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	ctx := context.Background()
	openErr := errors.New("no refresh token")
	for name, call := range map[string]func() error{
		"list_character_mining": func() error {
			_, _, err := listCharacterMiningHandler("v1", failNarrow[characterMiningESI](openErr), store, nil)(ctx, nil, ListCharacterMiningInput{CharacterID: 7})
			return err
		},
		"list_corporation_mining_extractions": func() error {
			_, _, err := listCorporationMiningExtractionsHandler("v1", failNarrow[corporationMiningExtractionsESI](openErr), store, nil)(ctx, nil, ListCorporationMiningExtractionsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_mining_observers": func() error {
			_, _, err := listCorporationMiningObserversHandler(failNarrow[corporationMiningObserversESI](openErr), store)(ctx, nil, ListCorporationMiningObserversInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"get_corporation_mining_observer_detail": func() error {
			_, _, err := getCorporationMiningObserverDetailHandler("v1", failNarrow[corporationMiningObserverDetailESI](openErr), store, nil)(ctx, nil, GetCorporationMiningObserverDetailInput{CharacterID: 7, CorporationID: 98000001, ObserverID: 1})
			return err
		},
	} {
		if err := call(); !errors.Is(err, openErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, openErr)
		}
	}
}
