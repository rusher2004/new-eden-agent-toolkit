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

func TestStatus_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"players":23456,"server_version":"1234567","start_time":"2026-07-25T11:00:00Z","vip":false}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Players != 23456 || status.ServerVersion != "1234567" {
		t.Errorf("status = %+v, want Players=23456 ServerVersion=\"1234567\"", status)
	}
	wantStart := time.Date(2026, 7, 25, 11, 0, 0, 0, time.UTC)
	if !status.StartTime.Equal(wantStart) {
		t.Errorf("StartTime = %v, want %v", status.StartTime, wantStart)
	}
	if status.Vip == nil || *status.Vip {
		t.Errorf("Vip = %v, want *false", status.Vip)
	}
}

func TestStatus_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.Status(context.Background()); err == nil {
		t.Fatal("Status returned nil error, want an error for HTTP 500")
	}
}
