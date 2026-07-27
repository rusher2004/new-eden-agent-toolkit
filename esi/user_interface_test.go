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

func TestSetAutopilotWaypoint_SendsParams(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.SetAutopilotWaypoint(context.Background(), 30000142, true, false); err != nil {
		t.Fatalf("SetAutopilotWaypoint returned error: %v", err)
	}
	if gotPath != "/ui/autopilot/waypoint" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "add_to_beginning=true&clear_other_waypoints=false&destination_id=30000142" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestSetAutopilotWaypoint_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.SetAutopilotWaypoint(context.Background(), 30000142, true, false); err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenContractWindow_SendsParams(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.OpenContractWindow(context.Background(), 5001); err != nil {
		t.Fatalf("OpenContractWindow returned error: %v", err)
	}
	if gotPath != "/ui/openwindow/contract" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "contract_id=5001" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestOpenContractWindow_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.OpenContractWindow(context.Background(), 5001); err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenInformationWindow_SendsParams(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.OpenInformationWindow(context.Background(), 98000001); err != nil {
		t.Fatalf("OpenInformationWindow returned error: %v", err)
	}
	if gotPath != "/ui/openwindow/information" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "target_id=98000001" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestOpenInformationWindow_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.OpenInformationWindow(context.Background(), 98000001); err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenMarketDetailsWindow_SendsParams(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.OpenMarketDetailsWindow(context.Background(), 587); err != nil {
		t.Fatalf("OpenMarketDetailsWindow returned error: %v", err)
	}
	if gotPath != "/ui/openwindow/marketdetails" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "type_id=587" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestOpenMarketDetailsWindow_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.OpenMarketDetailsWindow(context.Background(), 587); err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenNewMailWindow_SendsRequestBody(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	toCorpOrAllianceID := int64(1000035)
	err := client.OpenNewMailWindow(context.Background(), []int64{12345}, "Subject", "Body text", &toCorpOrAllianceID, nil)
	if err != nil {
		t.Fatalf("OpenNewMailWindow returned error: %v", err)
	}
	if gotBody["subject"] != "Subject" || gotBody["body"] != "Body text" {
		t.Errorf("body = %+v, want subject=Subject body=Body text", gotBody)
	}
	recipients, _ := gotBody["recipients"].([]any)
	if len(recipients) != 1 || recipients[0].(float64) != 12345 {
		t.Errorf("recipients = %v, want [12345]", gotBody["recipients"])
	}
	if gotBody["to_corp_or_alliance_id"] != float64(1000035) {
		t.Errorf("to_corp_or_alliance_id = %v, want 1000035", gotBody["to_corp_or_alliance_id"])
	}
	if _, ok := gotBody["to_mailing_list_id"]; ok {
		t.Errorf("to_mailing_list_id = %v, want omitted", gotBody["to_mailing_list_id"])
	}
}

func TestOpenNewMailWindow_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if err := client.OpenNewMailWindow(context.Background(), []int64{12345}, "Subject", "Body", nil, nil); err == nil {
		t.Fatal("expected error")
	}
}
