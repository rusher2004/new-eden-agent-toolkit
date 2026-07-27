package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestAlliances_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[99000001,99000002]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ids, err := client.Alliances(context.Background())
	if err != nil {
		t.Fatalf("Alliances returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 99000001 || ids[1] != 99000002 {
		t.Errorf("ids = %v, want [99000001 99000002]", ids)
	}
}

func TestAllianceDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"creator_corporation_id":1,"creator_id":2,"date_founded":"2020-01-01T00:00:00Z","name":"Test Alliance","ticker":"TEST"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.AllianceDetail(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceDetail returned error: %v", err)
	}
	if detail.Name != "Test Alliance" || detail.Ticker != "TEST" {
		t.Errorf("detail = %+v, want Name=\"Test Alliance\" Ticker=\"TEST\"", detail)
	}
	if detail.ExecutorCorporationId != nil {
		t.Errorf("detail.ExecutorCorporationId = %v, want nil (field absent from response)", detail.ExecutorCorporationId)
	}
	if detail.FactionId != nil {
		t.Errorf("detail.FactionId = %v, want nil (field absent from response)", detail.FactionId)
	}
}

func TestAllianceDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.AllianceDetail(context.Background(), 99000001); err == nil {
		t.Fatal("AllianceDetail returned nil error, want an error for HTTP 404")
	}
}

func TestAllianceCorporations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[98000001]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	corps, err := client.AllianceCorporations(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceCorporations returned error: %v", err)
	}
	if len(corps) != 1 || corps[0] != 98000001 {
		t.Errorf("corps = %v, want [98000001]", corps)
	}
}

func TestAllianceIcons_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"px128x128":"https://images.evetech.net/a.png"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	icons, err := client.AllianceIcons(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceIcons returned error: %v", err)
	}
	if icons.Px128x128 == nil || *icons.Px128x128 != "https://images.evetech.net/a.png" {
		t.Errorf("icons.Px128x128 = %v, want a pointer to the URL", icons.Px128x128)
	}
	if icons.Px64x64 != nil {
		t.Errorf("icons.Px64x64 = %v, want nil (field absent from response)", icons.Px64x64)
	}
}
