package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterIndustryJobs_ParsesResponseAndSendsIncludeCompleted(t *testing.T) {
	for _, includeCompleted := range []bool{true, false} {
		t.Run(map[bool]string{true: "true", false: "false"}[includeCompleted], func(t *testing.T) {
			var gotQuery string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`[{
					"activity_id": 1,
					"blueprint_id": 100,
					"blueprint_location_id": 200,
					"blueprint_type_id": 300,
					"completed_character_id": 998877,
					"completed_date": "2026-07-20T00:00:00Z",
					"cost": 12345.6,
					"duration": 3600,
					"end_date": "2026-07-21T00:00:00Z",
					"facility_id": 400,
					"installer_id": 998877,
					"job_id": 500,
					"licensed_runs": 10,
					"output_location_id": 600,
					"pause_date": "2026-07-19T00:00:00Z",
					"probability": 0.5,
					"product_type_id": 700,
					"runs": 5,
					"start_date": "2026-07-18T00:00:00Z",
					"station_id": 800,
					"status": "active",
					"successful_runs": 5
				}]`))
			}))
			defer srv.Close()

			client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

			jobs, err := client.CharacterIndustryJobs(context.Background(), includeCompleted)
			if err != nil {
				t.Fatalf("CharacterIndustryJobs returned error: %v", err)
			}

			wantQuery := "include_completed=" + map[bool]string{true: "true", false: "false"}[includeCompleted]
			if gotQuery != wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, wantQuery)
			}

			if len(jobs) != 1 {
				t.Fatalf("jobs = %+v, want one entry", jobs)
			}
			j := jobs[0]
			if j.ActivityId != 1 {
				t.Errorf("ActivityId = %d, want 1", j.ActivityId)
			}
			if j.BlueprintId != 100 {
				t.Errorf("BlueprintId = %d, want 100", j.BlueprintId)
			}
			if j.BlueprintLocationId != 200 {
				t.Errorf("BlueprintLocationId = %d, want 200", j.BlueprintLocationId)
			}
			if j.BlueprintTypeId != 300 {
				t.Errorf("BlueprintTypeId = %d, want 300", j.BlueprintTypeId)
			}
			if j.CompletedCharacterId == nil || *j.CompletedCharacterId != 998877 {
				t.Errorf("CompletedCharacterId = %v, want 998877", j.CompletedCharacterId)
			}
			wantCompletedDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
			if j.CompletedDate == nil || !j.CompletedDate.Equal(wantCompletedDate) {
				t.Errorf("CompletedDate = %v, want %v", j.CompletedDate, wantCompletedDate)
			}
			if j.Cost == nil || *j.Cost != 12345.6 {
				t.Errorf("Cost = %v, want 12345.6", j.Cost)
			}
			if j.Duration != 3600 {
				t.Errorf("Duration = %d, want 3600", j.Duration)
			}
			wantEndDate := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
			if !j.EndDate.Equal(wantEndDate) {
				t.Errorf("EndDate = %v, want %v", j.EndDate, wantEndDate)
			}
			if j.FacilityId != 400 {
				t.Errorf("FacilityId = %d, want 400", j.FacilityId)
			}
			if j.InstallerId != 998877 {
				t.Errorf("InstallerId = %d, want 998877", j.InstallerId)
			}
			if j.JobId != 500 {
				t.Errorf("JobId = %d, want 500", j.JobId)
			}
			if j.LicensedRuns == nil || *j.LicensedRuns != 10 {
				t.Errorf("LicensedRuns = %v, want 10", j.LicensedRuns)
			}
			if j.OutputLocationId != 600 {
				t.Errorf("OutputLocationId = %d, want 600", j.OutputLocationId)
			}
			wantPauseDate := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
			if j.PauseDate == nil || !j.PauseDate.Equal(wantPauseDate) {
				t.Errorf("PauseDate = %v, want %v", j.PauseDate, wantPauseDate)
			}
			if j.Probability == nil || *j.Probability != 0.5 {
				t.Errorf("Probability = %v, want 0.5", j.Probability)
			}
			if j.ProductTypeId == nil || *j.ProductTypeId != 700 {
				t.Errorf("ProductTypeId = %v, want 700", j.ProductTypeId)
			}
			if j.Runs != 5 {
				t.Errorf("Runs = %d, want 5", j.Runs)
			}
			wantStartDate := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
			if !j.StartDate.Equal(wantStartDate) {
				t.Errorf("StartDate = %v, want %v", j.StartDate, wantStartDate)
			}
			if j.StationId != 800 {
				t.Errorf("StationId = %d, want 800", j.StationId)
			}
			if j.Status != "active" {
				t.Errorf("Status = %q, want \"active\"", j.Status)
			}
			if j.SuccessfulRuns == nil || *j.SuccessfulRuns != 5 {
				t.Errorf("SuccessfulRuns = %v, want 5", j.SuccessfulRuns)
			}
		})
	}
}

func TestCharacterIndustryJobs_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterIndustryJobs(context.Background(), true); err == nil {
		t.Fatal("CharacterIndustryJobs returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterMining_ParsesResponseAndPageOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"date":"2026-07-20","quantity":1000,"solar_system_id":30000142,"type_id":1230}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	ledger, _, err := client.CharacterMining(context.Background(), 0)
	if err != nil {
		t.Fatalf("CharacterMining returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
	if len(ledger) != 1 {
		t.Fatalf("ledger = %+v, want one entry", ledger)
	}
	e := ledger[0]
	if e.Date != "2026-07-20" {
		t.Errorf("Date = %q, want \"2026-07-20\"", e.Date)
	}
	if e.Quantity != 1000 {
		t.Errorf("Quantity = %d, want 1000", e.Quantity)
	}
	if e.SolarSystemId != 30000142 {
		t.Errorf("SolarSystemId = %d, want 30000142", e.SolarSystemId)
	}
	if e.TypeId != 1230 {
		t.Errorf("TypeId = %d, want 1230", e.TypeId)
	}
}

func TestCharacterMining_PageSentWhenSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "4")
		w.Write([]byte(`[{"date":"2026-07-20","quantity":1000,"solar_system_id":30000142,"type_id":1230}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	_, totalPages, err := client.CharacterMining(context.Background(), 3)
	if err != nil {
		t.Fatalf("CharacterMining(page=3) returned error: %v", err)
	}
	if gotQuery != "page=3" {
		t.Errorf("query = %q, want \"page=3\"", gotQuery)
	}
	if totalPages != 4 {
		t.Errorf("totalPages = %d, want 4", totalPages)
	}
}

func TestCorporationMiningExtractions_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{
			"chunk_arrival_time": "2026-07-25T00:00:00Z",
			"extraction_start_time": "2026-07-20T00:00:00Z",
			"moon_id": 40000001,
			"natural_decay_time": "2026-07-27T00:00:00Z",
			"structure_id": 1000000000001
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	extractions, totalPages, err := client.CorporationMiningExtractions(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationMiningExtractions returned error: %v", err)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(extractions) != 1 {
		t.Fatalf("extractions = %+v, want one entry", extractions)
	}
	e := extractions[0]
	wantChunkArrival := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	if !e.ChunkArrivalTime.Equal(wantChunkArrival) {
		t.Errorf("ChunkArrivalTime = %v, want %v", e.ChunkArrivalTime, wantChunkArrival)
	}
	wantStart := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if !e.ExtractionStartTime.Equal(wantStart) {
		t.Errorf("ExtractionStartTime = %v, want %v", e.ExtractionStartTime, wantStart)
	}
	if e.MoonId != 40000001 {
		t.Errorf("MoonId = %d, want 40000001", e.MoonId)
	}
	wantDecay := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	if !e.NaturalDecayTime.Equal(wantDecay) {
		t.Errorf("NaturalDecayTime = %v, want %v", e.NaturalDecayTime, wantDecay)
	}
	if e.StructureId != 1000000000001 {
		t.Errorf("StructureId = %d, want 1000000000001", e.StructureId)
	}
}

func TestCorporationMiningObservers_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{"last_updated":"2026-07-20","observer_id":1000000000002,"observer_type":"structure"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	observers, totalPages, err := client.CorporationMiningObservers(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationMiningObservers returned error: %v", err)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(observers) != 1 {
		t.Fatalf("observers = %+v, want one entry", observers)
	}
	o := observers[0]
	if o.LastUpdated != "2026-07-20" {
		t.Errorf("LastUpdated = %q, want \"2026-07-20\"", o.LastUpdated)
	}
	if o.ObserverId != 1000000000002 {
		t.Errorf("ObserverId = %d, want 1000000000002", o.ObserverId)
	}
	if o.ObserverType != "structure" {
		t.Errorf("ObserverType = %q, want \"structure\"", o.ObserverType)
	}
}

func TestCorporationMiningObserverDetail_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"character_id":998877,"last_updated":"2026-07-20","quantity":500,"recorded_corporation_id":98000001,"type_id":1230}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	entries, totalPages, err := client.CorporationMiningObserverDetail(context.Background(), 98000001, 1000000000002, 0)
	if err != nil {
		t.Fatalf("CorporationMiningObserverDetail returned error: %v", err)
	}
	if totalPages != 1 {
		t.Errorf("totalPages = %d, want 1", totalPages)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v, want one entry", entries)
	}
	e := entries[0]
	if e.CharacterId != 998877 {
		t.Errorf("CharacterId = %d, want 998877", e.CharacterId)
	}
	if e.LastUpdated != "2026-07-20" {
		t.Errorf("LastUpdated = %q, want \"2026-07-20\"", e.LastUpdated)
	}
	if e.Quantity != 500 {
		t.Errorf("Quantity = %d, want 500", e.Quantity)
	}
	if e.RecordedCorporationId != 98000001 {
		t.Errorf("RecordedCorporationId = %d, want 98000001", e.RecordedCorporationId)
	}
	if e.TypeId != 1230 {
		t.Errorf("TypeId = %d, want 1230", e.TypeId)
	}
}

func TestCorporationIndustryJobs_ParsesResponseAndSendsIncludeCompleted(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{
			"activity_id": 1,
			"blueprint_id": 100,
			"blueprint_location_id": 200,
			"blueprint_type_id": 300,
			"completed_character_id": 998877,
			"completed_date": "2026-07-20T00:00:00Z",
			"cost": 12345.6,
			"duration": 3600,
			"end_date": "2026-07-21T00:00:00Z",
			"facility_id": 400,
			"installer_id": 998877,
			"job_id": 500,
			"licensed_runs": 10,
			"location_id": 900,
			"output_location_id": 600,
			"pause_date": "2026-07-19T00:00:00Z",
			"probability": 0.5,
			"product_type_id": 700,
			"runs": 5,
			"start_date": "2026-07-18T00:00:00Z",
			"status": "active",
			"successful_runs": 5
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	jobs, totalPages, err := client.CorporationIndustryJobs(context.Background(), 98000001, true, 0)
	if err != nil {
		t.Fatalf("CorporationIndustryJobs returned error: %v", err)
	}
	if gotQuery != "include_completed=true" {
		t.Errorf("query = %q, want \"include_completed=true\"", gotQuery)
	}
	if totalPages != 1 {
		t.Errorf("totalPages = %d, want 1", totalPages)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs = %+v, want one entry", jobs)
	}
	j := jobs[0]
	if j.ActivityId != 1 {
		t.Errorf("ActivityId = %d, want 1", j.ActivityId)
	}
	if j.BlueprintId != 100 {
		t.Errorf("BlueprintId = %d, want 100", j.BlueprintId)
	}
	if j.BlueprintLocationId != 200 {
		t.Errorf("BlueprintLocationId = %d, want 200", j.BlueprintLocationId)
	}
	if j.BlueprintTypeId != 300 {
		t.Errorf("BlueprintTypeId = %d, want 300", j.BlueprintTypeId)
	}
	if j.CompletedCharacterId == nil || *j.CompletedCharacterId != 998877 {
		t.Errorf("CompletedCharacterId = %v, want 998877", j.CompletedCharacterId)
	}
	wantCompletedDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if j.CompletedDate == nil || !j.CompletedDate.Equal(wantCompletedDate) {
		t.Errorf("CompletedDate = %v, want %v", j.CompletedDate, wantCompletedDate)
	}
	if j.Cost == nil || *j.Cost != 12345.6 {
		t.Errorf("Cost = %v, want 12345.6", j.Cost)
	}
	if j.Duration != 3600 {
		t.Errorf("Duration = %d, want 3600", j.Duration)
	}
	wantEndDate := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	if !j.EndDate.Equal(wantEndDate) {
		t.Errorf("EndDate = %v, want %v", j.EndDate, wantEndDate)
	}
	if j.FacilityId != 400 {
		t.Errorf("FacilityId = %d, want 400", j.FacilityId)
	}
	if j.InstallerId != 998877 {
		t.Errorf("InstallerId = %d, want 998877", j.InstallerId)
	}
	if j.JobId != 500 {
		t.Errorf("JobId = %d, want 500", j.JobId)
	}
	if j.LicensedRuns == nil || *j.LicensedRuns != 10 {
		t.Errorf("LicensedRuns = %v, want 10", j.LicensedRuns)
	}
	if j.LocationId != 900 {
		t.Errorf("LocationId = %d, want 900", j.LocationId)
	}
	if j.OutputLocationId != 600 {
		t.Errorf("OutputLocationId = %d, want 600", j.OutputLocationId)
	}
	wantPauseDate := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	if j.PauseDate == nil || !j.PauseDate.Equal(wantPauseDate) {
		t.Errorf("PauseDate = %v, want %v", j.PauseDate, wantPauseDate)
	}
	if j.Probability == nil || *j.Probability != 0.5 {
		t.Errorf("Probability = %v, want 0.5", j.Probability)
	}
	if j.ProductTypeId == nil || *j.ProductTypeId != 700 {
		t.Errorf("ProductTypeId = %v, want 700", j.ProductTypeId)
	}
	if j.Runs != 5 {
		t.Errorf("Runs = %d, want 5", j.Runs)
	}
	wantStartDate := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	if !j.StartDate.Equal(wantStartDate) {
		t.Errorf("StartDate = %v, want %v", j.StartDate, wantStartDate)
	}
	if j.Status != "active" {
		t.Errorf("Status = %q, want \"active\"", j.Status)
	}
	if j.SuccessfulRuns == nil || *j.SuccessfulRuns != 5 {
		t.Errorf("SuccessfulRuns = %v, want 5", j.SuccessfulRuns)
	}
}

func TestIndustryFacilities_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"facility_id":60000001,"owner_id":98000001,"region_id":10000002,"solar_system_id":30000142,"tax":0.1,"type_id":2502}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	facilities, err := client.IndustryFacilities(context.Background())
	if err != nil {
		t.Fatalf("IndustryFacilities returned error: %v", err)
	}
	if len(facilities) != 1 {
		t.Fatalf("facilities = %+v, want one entry", facilities)
	}
	f := facilities[0]
	if f.FacilityId != 60000001 {
		t.Errorf("FacilityId = %d, want 60000001", f.FacilityId)
	}
	if f.OwnerId != 98000001 {
		t.Errorf("OwnerId = %d, want 98000001", f.OwnerId)
	}
	if f.RegionId != 10000002 {
		t.Errorf("RegionId = %d, want 10000002", f.RegionId)
	}
	if f.SolarSystemId != 30000142 {
		t.Errorf("SolarSystemId = %d, want 30000142", f.SolarSystemId)
	}
	if f.Tax == nil || *f.Tax != 0.1 {
		t.Errorf("Tax = %v, want 0.1", f.Tax)
	}
	if f.TypeId != 2502 {
		t.Errorf("TypeId = %d, want 2502", f.TypeId)
	}
}

func TestIndustrySystems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"cost_indices":[{"activity":"manufacturing","cost_index":0.02}],"solar_system_id":30000142}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	systems, err := client.IndustrySystems(context.Background())
	if err != nil {
		t.Fatalf("IndustrySystems returned error: %v", err)
	}
	if len(systems) != 1 {
		t.Fatalf("systems = %+v, want one entry", systems)
	}
	s := systems[0]
	if s.SolarSystemId != 30000142 {
		t.Errorf("SolarSystemId = %d, want 30000142", s.SolarSystemId)
	}
	if len(s.CostIndices) != 1 {
		t.Fatalf("CostIndices = %+v, want one entry", s.CostIndices)
	}
	if s.CostIndices[0].Activity != "manufacturing" {
		t.Errorf("CostIndices[0].Activity = %q, want \"manufacturing\"", s.CostIndices[0].Activity)
	}
	if s.CostIndices[0].CostIndex != 0.02 {
		t.Errorf("CostIndices[0].CostIndex = %v, want 0.02", s.CostIndices[0].CostIndex)
	}
}
