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

func TestWars_OmitsMaxWarIdWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[5000,5001]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.Wars(context.Background(), 0)
	if err != nil {
		t.Fatalf("Wars returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (max_war_id omitted)", gotQuery)
	}
	if len(got) != 2 || got[0] != 5000 {
		t.Errorf("got = %v, want [5000 5001]", got)
	}
}

func TestWars_IncludesMaxWarIdWhenNonzero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[4999]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.Wars(context.Background(), 5000)
	if err != nil {
		t.Fatalf("Wars returned error: %v", err)
	}
	if gotQuery != "max_war_id=5000" {
		t.Errorf("query = %q, want max_war_id=5000", gotQuery)
	}
	if len(got) != 1 || got[0] != 4999 {
		t.Errorf("got = %v, want [4999]", got)
	}
}

func TestWars_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.Wars(context.Background(), 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestWarDetail_ParsesResponse_AllianceAndCorporationParties(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"aggressor": {"alliance_id": 99000001, "isk_destroyed": 1000000, "ships_killed": 5},
			"defender": {"corporation_id": 98000001, "isk_destroyed": 500000, "ships_killed": 2},
			"allies": [{"alliance_id": 99000002}, {"corporation_id": 98000002}],
			"declared": "2026-01-01T00:00:00Z", "id": 5000, "mutual": false, "open_for_allies": true
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, err := client.WarDetail(context.Background(), 5000)
	if err != nil {
		t.Fatalf("WarDetail returned error: %v", err)
	}
	if gotPath != "/wars/5000" {
		t.Errorf("path = %q", gotPath)
	}
	if got.Id != 5000 || got.Mutual || !got.OpenForAllies {
		t.Errorf("got = %+v, want Id=5000 Mutual=false OpenForAllies=true", got)
	}
	if got.Aggressor.AllianceId == nil || *got.Aggressor.AllianceId != 99000001 || got.Aggressor.CorporationId != nil {
		t.Errorf("Aggressor = %+v, want AllianceId=99000001 CorporationId=nil", got.Aggressor)
	}
	if got.Defender.CorporationId == nil || *got.Defender.CorporationId != 98000001 || got.Defender.AllianceId != nil {
		t.Errorf("Defender = %+v, want CorporationId=98000001 AllianceId=nil", got.Defender)
	}
	if len(got.Allies) != 2 {
		t.Fatalf("Allies = %+v, want 2 entries", got.Allies)
	}
	if got.Allies[0].AllianceId == nil || *got.Allies[0].AllianceId != 99000002 || got.Allies[0].CorporationId != nil {
		t.Errorf("Allies[0] = %+v, want AllianceId=99000002 CorporationId=nil", got.Allies[0])
	}
	if got.Allies[1].CorporationId == nil || *got.Allies[1].CorporationId != 98000002 || got.Allies[1].AllianceId != nil {
		t.Errorf("Allies[1] = %+v, want CorporationId=98000002 AllianceId=nil", got.Allies[1])
	}
	wantDeclared := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Declared.Equal(wantDeclared) {
		t.Errorf("Declared = %v, want %v", got.Declared, wantDeclared)
	}
}

func TestWarDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.WarDetail(context.Background(), 5000); err == nil {
		t.Fatal("expected error")
	}
}

func TestWarKillmails_ParsesResponseAndTotalPages(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"killmail_hash":"abc123","killmail_id":90000001}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	got, totalPages, err := client.WarKillmails(context.Background(), 5000, 1)
	if err != nil {
		t.Fatalf("WarKillmails returned error: %v", err)
	}
	if gotPath != "/wars/5000/killmails" {
		t.Errorf("path = %q", gotPath)
	}
	if len(got) != 1 || got[0].KillmailId != 90000001 || got[0].KillmailHash != "abc123" || totalPages != 2 {
		t.Errorf("got = %+v, totalPages = %d, want one entry KillmailId=90000001 KillmailHash=abc123 totalPages=2", got, totalPages)
	}
}

func TestWarKillmails_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, _, err := client.WarKillmails(context.Background(), 5000, 1); err == nil {
		t.Fatal("expected error")
	}
}
