package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestIncursions_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"constellation_id":30000001,"faction_id":500001,"has_boss":true,"infested_solar_systems":[30000002,30000003],"influence":0.8,"staging_solar_system_id":30000004,"state":"mobilizing","type":"minor"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	incursions, err := client.Incursions(context.Background())
	if err != nil {
		t.Fatalf("Incursions returned error: %v", err)
	}
	if len(incursions) != 1 {
		t.Errorf("len(incursions) = %d, want 1", len(incursions))
	}
	if incursions[0].ConstellationId != 30000001 {
		t.Errorf("ConstellationId = %d, want 30000001", incursions[0].ConstellationId)
	}
	if incursions[0].FactionId != 500001 {
		t.Errorf("FactionId = %d, want 500001", incursions[0].FactionId)
	}
	if incursions[0].HasBoss != true {
		t.Errorf("HasBoss = %v, want true", incursions[0].HasBoss)
	}
	if len(incursions[0].InfestedSolarSystems) != 2 || incursions[0].InfestedSolarSystems[0] != 30000002 {
		t.Errorf("InfestedSolarSystems = %v, want [30000002 30000003]", incursions[0].InfestedSolarSystems)
	}
	if incursions[0].Influence != 0.8 {
		t.Errorf("Influence = %f, want 0.8", incursions[0].Influence)
	}
	if incursions[0].StagingSolarSystemId != 30000004 {
		t.Errorf("StagingSolarSystemId = %d, want 30000004", incursions[0].StagingSolarSystemId)
	}
	if incursions[0].State != "mobilizing" {
		t.Errorf("State = %s, want mobilizing", incursions[0].State)
	}
	if incursions[0].Type != "minor" {
		t.Errorf("Type = %s, want minor", incursions[0].Type)
	}
}

func TestIncursions_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.Incursions(context.Background()); err == nil {
		t.Fatal("Incursions returned nil error, want an error for HTTP 404")
	}
}
