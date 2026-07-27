package esi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestUniverseAncestries_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"bloodline_id":1,"description":"desc","icon_id":100,"id":11,"name":"Tribal","short_description":"short"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	got, err := client.UniverseAncestries(context.Background())
	if err != nil {
		t.Fatalf("UniverseAncestries returned error: %v", err)
	}
	if len(got) != 1 || got[0].Id != 11 || got[0].BloodlineId != 1 || got[0].Name != "Tribal" {
		t.Errorf("got = %+v, want one entry Id=11 BloodlineId=1 Name=Tribal", got)
	}
}

func TestUniverseAncestries_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseAncestries(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseBloodlines_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"bloodline_id":1,"charisma":6,"corporation_id":1000166,"description":"desc","intelligence":7,"memory":5,"name":"Deteis","perception":5,"race_id":1,"ship_type_id":601,"willpower":5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseBloodlines(context.Background())
	if err != nil {
		t.Fatalf("UniverseBloodlines returned error: %v", err)
	}
	if len(got) != 1 || got[0].BloodlineId != 1 || got[0].Name != "Deteis" || got[0].RaceId != 1 {
		t.Errorf("got = %+v, want one entry BloodlineId=1 Name=Deteis RaceId=1", got)
	}
}

func TestUniverseBloodlines_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseBloodlines(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseRaces_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"alliance_id":500001,"description":"desc","name":"Caldari","race_id":1}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseRaces(context.Background())
	if err != nil {
		t.Fatalf("UniverseRaces returned error: %v", err)
	}
	if len(got) != 1 || got[0].RaceId != 1 || got[0].AllianceId != 500001 || got[0].Name != "Caldari" {
		t.Errorf("got = %+v, want one entry RaceId=1 AllianceId=500001 Name=Caldari", got)
	}
}

func TestUniverseRaces_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseRaces(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseFactions_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"corporation_id":1000035,"description":"desc","faction_id":500001,"is_unique":true,"militia_corporation_id":1000180,"name":"Caldari State","size_factor":5,"solar_system_id":30000144,"station_count":100,"station_system_count":50}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseFactions(context.Background())
	if err != nil {
		t.Fatalf("UniverseFactions returned error: %v", err)
	}
	if len(got) != 1 || got[0].FactionId != 500001 || !got[0].IsUnique || got[0].Name != "Caldari State" {
		t.Errorf("got = %+v, want one entry FactionId=500001 IsUnique=true Name=Caldari State", got)
	}
	if got[0].CorporationId == nil || *got[0].CorporationId != 1000035 {
		t.Errorf("CorporationId = %v, want 1000035", got[0].CorporationId)
	}
}

func TestUniverseFactions_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseFactions(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseCategories_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[6,7,8]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseCategories(context.Background())
	if err != nil {
		t.Fatalf("UniverseCategories returned error: %v", err)
	}
	if gotPath != "/universe/categories" {
		t.Errorf("path = %q", gotPath)
	}
	if len(got) != 3 || got[0] != 6 {
		t.Errorf("got = %v, want [6 7 8]", got)
	}
}

func TestUniverseCategories_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseCategories(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseCategoryDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"category_id":6,"groups":[25,26],"name":"Ship","published":true}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseCategoryDetail(context.Background(), 6)
	if err != nil {
		t.Fatalf("UniverseCategoryDetail returned error: %v", err)
	}
	if gotPath != "/universe/categories/6" {
		t.Errorf("path = %q", gotPath)
	}
	if got.CategoryId != 6 || got.Name != "Ship" || !got.Published || len(got.Groups) != 2 {
		t.Errorf("got = %+v, want CategoryId=6 Name=Ship Published=true Groups=[25 26]", got)
	}
}

func TestUniverseCategoryDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseCategoryDetail(context.Background(), 6); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseGroups_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[25,26,27]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, totalPages, err := client.UniverseGroups(context.Background(), 2)
	if err != nil {
		t.Fatalf("UniverseGroups returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want page=2", gotQuery)
	}
	if len(got) != 3 || totalPages != 3 {
		t.Errorf("got = %v, totalPages = %d, want [25 26 27] totalPages=3", got, totalPages)
	}
}

func TestUniverseGroups_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, _, err := client.UniverseGroups(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseGroupDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"category_id":6,"group_id":25,"name":"Frigate","published":true,"types":[587,588]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseGroupDetail(context.Background(), 25)
	if err != nil {
		t.Fatalf("UniverseGroupDetail returned error: %v", err)
	}
	if gotPath != "/universe/groups/25" {
		t.Errorf("path = %q", gotPath)
	}
	if got.GroupId != 25 || got.CategoryId != 6 || got.Name != "Frigate" || len(got.Types) != 2 {
		t.Errorf("got = %+v, want GroupId=25 CategoryId=6 Name=Frigate Types=[587 588]", got)
	}
}

func TestUniverseGroupDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseGroupDetail(context.Background(), 25); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseTypes_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "5")
		w.Write([]byte(`[587,588]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, totalPages, err := client.UniverseTypes(context.Background(), 1)
	if err != nil {
		t.Fatalf("UniverseTypes returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want page=1", gotQuery)
	}
	if len(got) != 2 || totalPages != 5 {
		t.Errorf("got = %v, totalPages = %d, want [587 588] totalPages=5", got, totalPages)
	}
}

func TestUniverseTypes_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, _, err := client.UniverseTypes(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseTypeDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"capacity": 400, "description": "desc", "graphic_id": 1234, "group_id": 25,
			"icon_id": 100, "market_group_id": 5, "mass": 1200000, "name": "Rifter",
			"packaged_volume": 2500, "portion_size": 1, "published": true, "radius": 40,
			"type_id": 587, "volume": 27289,
			"dogma_attributes": [{"attribute_id": 9, "value": 2500}],
			"dogma_effects": [{"effect_id": 1, "is_default": true}]
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseTypeDetail(context.Background(), 587)
	if err != nil {
		t.Fatalf("UniverseTypeDetail returned error: %v", err)
	}
	if gotPath != "/universe/types/587" {
		t.Errorf("path = %q", gotPath)
	}
	if got.TypeId != 587 || got.Name != "Rifter" || got.GroupId != 25 || !got.Published {
		t.Errorf("got = %+v, want TypeId=587 Name=Rifter GroupId=25 Published=true", got)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].AttributeId != 9 || got.DogmaAttributes[0].Value != 2500 {
		t.Errorf("DogmaAttributes = %+v, want one entry AttributeId=9 Value=2500", got.DogmaAttributes)
	}
	if len(got.DogmaEffects) != 1 || got.DogmaEffects[0].EffectId != 1 || !got.DogmaEffects[0].IsDefault {
		t.Errorf("DogmaEffects = %+v, want one entry EffectId=1 IsDefault=true", got.DogmaEffects)
	}
}

func TestUniverseTypeDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseTypeDetail(context.Background(), 587); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseRegions_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[10000002,10000043]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseRegions(context.Background())
	if err != nil {
		t.Fatalf("UniverseRegions returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 10000002 {
		t.Errorf("got = %v, want [10000002 10000043]", got)
	}
}

func TestUniverseRegions_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseRegions(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseRegionDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"constellations":[20000020,20000021],"description":"desc","name":"The Forge","region_id":10000002}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseRegionDetail(context.Background(), 10000002)
	if err != nil {
		t.Fatalf("UniverseRegionDetail returned error: %v", err)
	}
	if gotPath != "/universe/regions/10000002" {
		t.Errorf("path = %q", gotPath)
	}
	if got.RegionId != 10000002 || got.Name != "The Forge" || len(got.Constellations) != 2 {
		t.Errorf("got = %+v, want RegionId=10000002 Name=The Forge Constellations=[20000020 20000021]", got)
	}
	if got.Description == nil || *got.Description != "desc" {
		t.Errorf("Description = %v, want desc", got.Description)
	}
}

func TestUniverseRegionDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseRegionDetail(context.Background(), 10000002); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseConstellations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[20000020,20000021]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseConstellations(context.Background())
	if err != nil {
		t.Fatalf("UniverseConstellations returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 20000020 {
		t.Errorf("got = %v, want [20000020 20000021]", got)
	}
}

func TestUniverseConstellations_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseConstellations(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseConstellationDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"constellation_id":20000020,"name":"Kimotoro","position":{"x":1,"y":2,"z":3},"region_id":10000002,"systems":[30000142,30000144]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseConstellationDetail(context.Background(), 20000020)
	if err != nil {
		t.Fatalf("UniverseConstellationDetail returned error: %v", err)
	}
	if gotPath != "/universe/constellations/20000020" {
		t.Errorf("path = %q", gotPath)
	}
	if got.ConstellationId != 20000020 || got.Name != "Kimotoro" || got.RegionId != 10000002 || len(got.Systems) != 2 {
		t.Errorf("got = %+v, want ConstellationId=20000020 Name=Kimotoro RegionId=10000002 Systems=[30000142 30000144]", got)
	}
	if got.Position.X != 1 || got.Position.Y != 2 || got.Position.Z != 3 {
		t.Errorf("Position = %+v, want X=1 Y=2 Z=3", got.Position)
	}
}

func TestUniverseConstellationDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseConstellationDetail(context.Background(), 20000020); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseSystems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[30000142,30000144]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseSystems(context.Background())
	if err != nil {
		t.Fatalf("UniverseSystems returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 30000142 {
		t.Errorf("got = %v, want [30000142 30000144]", got)
	}
}

func TestUniverseSystems_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseSystems(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseSystemDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"constellation_id": 20000020, "name": "Jita", "position": {"x":1,"y":2,"z":3},
			"security_class": "B", "security_status": 0.9459, "star_id": 40000001,
			"stargates": [50000342], "stations": [60003760], "system_id": 30000142,
			"planets": [{"asteroid_belts": [40000002], "moons": [40000003], "planet_id": 40000004}]
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseSystemDetail(context.Background(), 30000142)
	if err != nil {
		t.Fatalf("UniverseSystemDetail returned error: %v", err)
	}
	if gotPath != "/universe/systems/30000142" {
		t.Errorf("path = %q", gotPath)
	}
	if got.SystemId != 30000142 || got.Name != "Jita" || got.ConstellationId != 20000020 {
		t.Errorf("got = %+v, want SystemId=30000142 Name=Jita ConstellationId=20000020", got)
	}
	if got.SecurityStatus != 0.9459 || got.SecurityClass == nil || *got.SecurityClass != "B" {
		t.Errorf("SecurityStatus/Class = %v/%v, want 0.9459/B", got.SecurityStatus, got.SecurityClass)
	}
	if len(got.Planets) != 1 || got.Planets[0].PlanetId != 40000004 || len(got.Planets[0].AsteroidBelts) != 1 || len(got.Planets[0].Moons) != 1 {
		t.Errorf("Planets = %+v, want one entry PlanetId=40000004 with one asteroid belt and one moon", got.Planets)
	}
}

func TestUniverseSystemDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseSystemDetail(context.Background(), 30000142); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseAsteroidBeltDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Jita IV - Asteroid Belt 1","position":{"x":1,"y":2,"z":3},"system_id":30000142}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseAsteroidBeltDetail(context.Background(), 40000002)
	if err != nil {
		t.Fatalf("UniverseAsteroidBeltDetail returned error: %v", err)
	}
	if gotPath != "/universe/asteroid_belts/40000002" {
		t.Errorf("path = %q", gotPath)
	}
	if got.Name != "Jita IV - Asteroid Belt 1" || got.SystemId != 30000142 {
		t.Errorf("got = %+v, want Name=Jita IV - Asteroid Belt 1 SystemId=30000142", got)
	}
}

func TestUniverseAsteroidBeltDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseAsteroidBeltDetail(context.Background(), 40000002); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseMoonDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"moon_id":40000003,"name":"Jita IV - Moon 1","position":{"x":1,"y":2,"z":3},"system_id":30000142}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseMoonDetail(context.Background(), 40000003)
	if err != nil {
		t.Fatalf("UniverseMoonDetail returned error: %v", err)
	}
	if gotPath != "/universe/moons/40000003" {
		t.Errorf("path = %q", gotPath)
	}
	if got.MoonId != 40000003 || got.Name != "Jita IV - Moon 1" || got.SystemId != 30000142 {
		t.Errorf("got = %+v, want MoonId=40000003 Name=Jita IV - Moon 1 SystemId=30000142", got)
	}
}

func TestUniverseMoonDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseMoonDetail(context.Background(), 40000003); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniversePlanetDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Jita IV","planet_id":40000004,"position":{"x":1,"y":2,"z":3},"system_id":30000142,"type_id":2016}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniversePlanetDetail(context.Background(), 40000004)
	if err != nil {
		t.Fatalf("UniversePlanetDetail returned error: %v", err)
	}
	if gotPath != "/universe/planets/40000004" {
		t.Errorf("path = %q", gotPath)
	}
	if got.PlanetId != 40000004 || got.Name != "Jita IV" || got.SystemId != 30000142 || got.TypeId != 2016 {
		t.Errorf("got = %+v, want PlanetId=40000004 Name=Jita IV SystemId=30000142 TypeId=2016", got)
	}
}

func TestUniversePlanetDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniversePlanetDetail(context.Background(), 40000004); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseStargateDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"destination": {"stargate_id": 50000343, "system_id": 30000144},
			"name": "Stargate (Perimeter)", "position": {"x":1,"y":2,"z":3},
			"stargate_id": 50000342, "system_id": 30000142, "type_id": 16
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseStargateDetail(context.Background(), 50000342)
	if err != nil {
		t.Fatalf("UniverseStargateDetail returned error: %v", err)
	}
	if gotPath != "/universe/stargates/50000342" {
		t.Errorf("path = %q", gotPath)
	}
	if got.StargateId != 50000342 || got.SystemId != 30000142 || got.TypeId != 16 {
		t.Errorf("got = %+v, want StargateId=50000342 SystemId=30000142 TypeId=16", got)
	}
	if got.Destination.StargateId != 50000343 || got.Destination.SystemId != 30000144 {
		t.Errorf("Destination = %+v, want StargateId=50000343 SystemId=30000144", got.Destination)
	}
}

func TestUniverseStargateDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseStargateDetail(context.Background(), 50000342); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseStarDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"age": 15000000000, "luminosity": 0.02, "name": "Jita - Star",
			"radius": 100000, "solar_system_id": 30000142, "spectral_class": "K2 V",
			"temperature": 4465, "type_id": 3796
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseStarDetail(context.Background(), 40000001)
	if err != nil {
		t.Fatalf("UniverseStarDetail returned error: %v", err)
	}
	if gotPath != "/universe/stars/40000001" {
		t.Errorf("path = %q", gotPath)
	}
	if got.SolarSystemId != 30000142 || got.Name != "Jita - Star" || got.SpectralClass != "K2 V" || got.TypeId != 3796 {
		t.Errorf("got = %+v, want SolarSystemId=30000142 Name=Jita - Star SpectralClass=K2 V TypeId=3796", got)
	}
}

func TestUniverseStarDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseStarDetail(context.Background(), 40000001); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseStationDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"max_dockable_ship_volume": 50000000, "name": "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
			"office_rental_cost": 10000, "owner": 1000035, "position": {"x":1,"y":2,"z":3},
			"race_id": 1, "reprocessing_efficiency": 0.5, "reprocessing_stations_take": 0.05,
			"services": ["market","repair-facilities"], "station_id": 60003760, "system_id": 30000142, "type_id": 1529
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseStationDetail(context.Background(), 60003760)
	if err != nil {
		t.Fatalf("UniverseStationDetail returned error: %v", err)
	}
	if gotPath != "/universe/stations/60003760" {
		t.Errorf("path = %q", gotPath)
	}
	if got.StationId != 60003760 || got.SystemId != 30000142 || got.TypeId != 1529 {
		t.Errorf("got = %+v, want StationId=60003760 SystemId=30000142 TypeId=1529", got)
	}
	if got.Owner == nil || *got.Owner != 1000035 {
		t.Errorf("Owner = %v, want 1000035", got.Owner)
	}
	if len(got.Services) != 2 {
		t.Errorf("Services = %v, want 2 entries", got.Services)
	}
}

func TestUniverseStationDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseStationDetail(context.Background(), 60003760); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseGraphics_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[1,2,3]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseGraphics(context.Background())
	if err != nil {
		t.Fatalf("UniverseGraphics returned error: %v", err)
	}
	if len(got) != 3 || got[0] != 1 {
		t.Errorf("got = %v, want [1 2 3]", got)
	}
}

func TestUniverseGraphics_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseGraphics(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseGraphicDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"graphic_file":"file.gr2","graphic_id":1234,"sof_hull_name":"caldari_frigate"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseGraphicDetail(context.Background(), 1234)
	if err != nil {
		t.Fatalf("UniverseGraphicDetail returned error: %v", err)
	}
	if gotPath != "/universe/graphics/1234" {
		t.Errorf("path = %q", gotPath)
	}
	if got.GraphicId != 1234 || got.GraphicFile == nil || *got.GraphicFile != "file.gr2" {
		t.Errorf("got = %+v, want GraphicId=1234 GraphicFile=file.gr2", got)
	}
}

func TestUniverseGraphicDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseGraphicDetail(context.Background(), 1234); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseStructures_ParsesResponse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[1000000000001,1000000000002]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseStructures(context.Background(), "market")
	if err != nil {
		t.Fatalf("UniverseStructures returned error: %v", err)
	}
	if gotQuery != "filter=market" {
		t.Errorf("query = %q, want filter=market", gotQuery)
	}
	if len(got) != 2 || got[0] != 1000000000001 {
		t.Errorf("got = %v, want [1000000000001 1000000000002]", got)
	}
}

func TestUniverseStructures_EmptyFilterOmitsParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseStructures(context.Background(), ""); err != nil {
		t.Fatalf("UniverseStructures returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (filter omitted)", gotQuery)
	}
}

func TestUniverseStructures_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseStructures(context.Background(), ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseStructureDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Test Citadel","owner_id":98000001,"position":{"x":1,"y":2,"z":3},"solar_system_id":30000142,"type_id":35832}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseStructureDetail(context.Background(), 1000000000001)
	if err != nil {
		t.Fatalf("UniverseStructureDetail returned error: %v", err)
	}
	if gotPath != "/universe/structures/1000000000001" {
		t.Errorf("path = %q", gotPath)
	}
	if got.Name != "Test Citadel" || got.OwnerId != 98000001 || got.SolarSystemId != 30000142 {
		t.Errorf("got = %+v, want Name=Test Citadel OwnerId=98000001 SolarSystemId=30000142", got)
	}
	if got.Position == nil || got.Position.X != 1 {
		t.Errorf("Position = %v, want X=1", got.Position)
	}
}

func TestUniverseStructureDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseStructureDetail(context.Background(), 1000000000001); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseSystemJumps_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"ship_jumps":10,"system_id":30000142}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseSystemJumps(context.Background())
	if err != nil {
		t.Fatalf("UniverseSystemJumps returned error: %v", err)
	}
	if len(got) != 1 || got[0].SystemId != 30000142 || got[0].ShipJumps != 10 {
		t.Errorf("got = %+v, want one entry SystemId=30000142 ShipJumps=10", got)
	}
}

func TestUniverseSystemJumps_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseSystemJumps(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseSystemKills_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"npc_kills":5,"pod_kills":1,"ship_kills":2,"system_id":30000142}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseSystemKills(context.Background())
	if err != nil {
		t.Fatalf("UniverseSystemKills returned error: %v", err)
	}
	if len(got) != 1 || got[0].SystemId != 30000142 || got[0].NpcKills != 5 || got[0].PodKills != 1 || got[0].ShipKills != 2 {
		t.Errorf("got = %+v, want one entry SystemId=30000142 NpcKills=5 PodKills=1 ShipKills=2", got)
	}
}

func TestUniverseSystemKills_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseSystemKills(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseIDsByName_SendsNamesAndParsesResponse(t *testing.T) {
	var gotBody []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"characters":[{"id":12345,"name":"Test Pilot"}],"systems":[{"id":30000142,"name":"Jita"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseIDsByName(context.Background(), []string{"Test Pilot", "Jita"})
	if err != nil {
		t.Fatalf("UniverseIDsByName returned error: %v", err)
	}
	if len(gotBody) != 2 || gotBody[0] != "Test Pilot" || gotBody[1] != "Jita" {
		t.Errorf("request body = %v, want [Test Pilot Jita]", gotBody)
	}
	if len(got.Characters) != 1 || got.Characters[0].Id == nil || *got.Characters[0].Id != 12345 {
		t.Errorf("Characters = %+v, want one entry Id=12345", got.Characters)
	}
	if len(got.Systems) != 1 || got.Systems[0].Name == nil || *got.Systems[0].Name != "Jita" {
		t.Errorf("Systems = %+v, want one entry Name=Jita", got.Systems)
	}
	if len(got.Alliances) != 0 {
		t.Errorf("Alliances = %+v, want empty", got.Alliances)
	}
}

func TestUniverseIDsByName_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseIDsByName(context.Background(), []string{"Jita"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestUniverseNamesByID_SendsIDsAndParsesResponse(t *testing.T) {
	var gotBody []int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"category":"solar_system","id":30000142,"name":"Jita"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.UniverseNamesByID(context.Background(), []int64{30000142})
	if err != nil {
		t.Fatalf("UniverseNamesByID returned error: %v", err)
	}
	if len(gotBody) != 1 || gotBody[0] != 30000142 {
		t.Errorf("request body = %v, want [30000142]", gotBody)
	}
	if len(got) != 1 || got[0].Category != "solar_system" || got[0].Id != 30000142 || got[0].Name != "Jita" {
		t.Errorf("got = %+v, want one entry Category=solar_system Id=30000142 Name=Jita", got)
	}
}

func TestUniverseNamesByID_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.UniverseNamesByID(context.Background(), []int64{30000142}); err == nil {
		t.Fatal("expected error")
	}
}
