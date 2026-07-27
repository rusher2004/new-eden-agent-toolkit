package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestMetaChangelog_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"changelog":{"2026-07-01":[{"compatibility_date":"2026-07-01","description":"Added a thing","method":"GET","path":"/foo","type":"added"}]}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	changelog, err := client.MetaChangelog(context.Background())
	if err != nil {
		t.Fatalf("MetaChangelog returned error: %v", err)
	}
	entries, ok := changelog.Changelog["2026-07-01"]
	if !ok || len(entries) != 1 {
		t.Fatalf("Changelog[2026-07-01] = %+v, want one entry", entries)
	}
	e := entries[0]
	if e.CompatibilityDate != "2026-07-01" || e.Description != "Added a thing" || e.Method != "GET" || e.Path != "/foo" || e.Type != "added" {
		t.Errorf("entry = %+v, want CompatibilityDate=2026-07-01 Description=\"Added a thing\" Method=GET Path=/foo Type=added", e)
	}
}

func TestMetaChangelog_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.MetaChangelog(context.Background()); err == nil {
		t.Fatal("MetaChangelog returned nil error, want an error for HTTP 500")
	}
}

func TestMetaCompatibilityDates_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"compatibility_dates":["2026-01-01","2026-07-01"]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	dates, err := client.MetaCompatibilityDates(context.Background())
	if err != nil {
		t.Fatalf("MetaCompatibilityDates returned error: %v", err)
	}
	if len(dates.CompatibilityDates) != 2 || dates.CompatibilityDates[0] != "2026-01-01" || dates.CompatibilityDates[1] != "2026-07-01" {
		t.Errorf("CompatibilityDates = %+v, want [2026-01-01 2026-07-01]", dates.CompatibilityDates)
	}
}

func TestMetaCompatibilityDates_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.MetaCompatibilityDates(context.Background()); err == nil {
		t.Fatal("MetaCompatibilityDates returned nil error, want an error for HTTP 500")
	}
}

func TestMetaName_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"current":"ESI","history":[{"date":"2016-01-01","name":"CREST"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	name, err := client.MetaName(context.Background())
	if err != nil {
		t.Fatalf("MetaName returned error: %v", err)
	}
	if name.Current != "ESI" {
		t.Errorf("Current = %q, want \"ESI\"", name.Current)
	}
	if len(name.History) != 1 || name.History[0].Date != "2016-01-01" || name.History[0].Name != "CREST" {
		t.Errorf("History = %+v, want one entry {Date:2016-01-01 Name:CREST}", name.History)
	}
}

func TestMetaName_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.MetaName(context.Background()); err == nil {
		t.Fatal("MetaName returned nil error, want an error for HTTP 500")
	}
}

func TestMetaStatus_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"routes":[{"method":"GET","path":"/characters/{character_id}/","status":"green"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	status, err := client.MetaStatus(context.Background())
	if err != nil {
		t.Fatalf("MetaStatus returned error: %v", err)
	}
	if len(status.Routes) != 1 {
		t.Fatalf("Routes = %+v, want one entry", status.Routes)
	}
	r := status.Routes[0]
	if r.Method != "GET" || r.Path != "/characters/{character_id}/" || r.Status != "green" {
		t.Errorf("Routes[0] = %+v, want Method=GET Path=/characters/{character_id}/ Status=green", r)
	}
}

func TestMetaStatus_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.MetaStatus(context.Background()); err == nil {
		t.Fatal("MetaStatus returned nil error, want an error for HTTP 500")
	}
}
