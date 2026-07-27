package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterLocation_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/location" {
			t.Errorf("request path = %q, want /characters/998877/location", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"solar_system_id":30000142,"station_id":60003760}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	loc, err := client.CharacterLocation(context.Background())
	if err != nil {
		t.Fatalf("CharacterLocation returned error: %v", err)
	}
	if loc.SolarSystemId != 30000142 {
		t.Errorf("SolarSystemId = %d, want 30000142", loc.SolarSystemId)
	}
	if loc.StationId == nil || *loc.StationId != 60003760 {
		t.Errorf("StationId = %v, want pointer to 60003760", loc.StationId)
	}
	if loc.StructureId != nil {
		t.Errorf("StructureId = %v, want nil", loc.StructureId)
	}
}

func TestCharacterLocation_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterLocation(context.Background()); err == nil {
		t.Fatal("CharacterLocation returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterOnline_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/online" {
			t.Errorf("request path = %q, want /characters/998877/online", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"online":true,"last_login":"2026-07-01T12:00:00Z","last_logout":"2026-06-30T23:00:00Z","logins":42}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	status, err := client.CharacterOnline(context.Background())
	if err != nil {
		t.Fatalf("CharacterOnline returned error: %v", err)
	}
	if !status.Online {
		t.Error("Online = false, want true")
	}
	if status.LastLogin == nil || status.LastLogin.Year() != 2026 {
		t.Errorf("LastLogin = %v, want a 2026 timestamp", status.LastLogin)
	}
	if status.LastLogout == nil {
		t.Error("LastLogout = nil, want a timestamp")
	}
	if status.Logins == nil || *status.Logins != 42 {
		t.Errorf("Logins = %v, want pointer to 42", status.Logins)
	}
}

func TestCharacterOnline_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterOnline(context.Background()); err == nil {
		t.Fatal("CharacterOnline returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterShip_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/ship" {
			t.Errorf("request path = %q, want /characters/998877/ship", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ship_item_id":1000000016991,"ship_name":"Percival","ship_type_id":608}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ship, err := client.CharacterShip(context.Background())
	if err != nil {
		t.Fatalf("CharacterShip returned error: %v", err)
	}
	if ship.ShipItemId != 1000000016991 || ship.ShipName != "Percival" || ship.ShipTypeId != 608 {
		t.Errorf("ship = %+v, want ShipItemId=1000000016991 ShipName=Percival ShipTypeId=608", ship)
	}
}

func TestCharacterShip_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterShip(context.Background()); err == nil {
		t.Fatal("CharacterShip returned nil error, want an error for HTTP 404")
	}
}
