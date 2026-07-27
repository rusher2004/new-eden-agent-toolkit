package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestMercenaryTacticalOperations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"operations":[{"id":"op-1","mercenary_den_id":1000}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ops, err := client.MercenaryTacticalOperations(context.Background())
	if err != nil {
		t.Fatalf("MercenaryTacticalOperations returned error: %v", err)
	}
	if len(ops) != 1 || ops[0].Id != "op-1" || ops[0].MercenaryDenId != 1000 {
		t.Errorf("ops = %+v, want one entry {Id:\"op-1\" MercenaryDenId:1000}", ops)
	}
}

func TestMercenaryTacticalOperationDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"dungeon_type_id":30000,"expires":"2026-08-01T00:00:00Z","id":"op-1","mercenary_den_id":1000,"state":"active"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.MercenaryTacticalOperationDetail(context.Background(), "op-1")
	if err != nil {
		t.Fatalf("MercenaryTacticalOperationDetail returned error: %v", err)
	}
	if detail.DungeonTypeId != 30000 || detail.State != "active" || detail.MercenaryDenId != 1000 {
		t.Errorf("detail = %+v, want DungeonTypeId=30000 State=\"active\" MercenaryDenId=1000", detail)
	}
}

func TestMercenaryTacticalOperationDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.MercenaryTacticalOperationDetail(context.Background(), "op-1"); err == nil {
		t.Fatal("MercenaryTacticalOperationDetail returned nil error, want an error for HTTP 403")
	}
}

func TestRaidableSkyhooks_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"skyhooks":[{"planet_id":40000001,"solar_system_id":30000142,"theft_vulnerability":{"start":"2026-07-26T00:00:00Z","end":"2026-07-27T00:00:00Z"}}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	skyhooks, err := client.RaidableSkyhooks(context.Background())
	if err != nil {
		t.Fatalf("RaidableSkyhooks returned error: %v", err)
	}
	if len(skyhooks) != 1 || skyhooks[0].PlanetId != 40000001 || skyhooks[0].SolarSystemId != 30000142 {
		t.Errorf("skyhooks = %+v, want one entry {PlanetId:40000001 SolarSystemId:30000142}", skyhooks)
	}
}
