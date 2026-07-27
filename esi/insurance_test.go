package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestInsurancePrices_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"type_id":587,"levels":[{"cost":900.0,"name":"Standard","payout":450.0},{"cost":1350.0,"name":"Extended","payout":675.0}]}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	prices, err := client.InsurancePrices(context.Background())
	if err != nil {
		t.Fatalf("InsurancePrices returned error: %v", err)
	}
	if len(prices) != 1 {
		t.Errorf("len(prices) = %d, want 1", len(prices))
	}
	if prices[0].TypeId != 587 {
		t.Errorf("TypeId = %d, want 587", prices[0].TypeId)
	}
	if len(prices[0].Levels) != 2 {
		t.Errorf("len(levels) = %d, want 2", len(prices[0].Levels))
	}
	if prices[0].Levels[0].Cost != 900.0 {
		t.Errorf("Levels[0].Cost = %f, want 900.0", prices[0].Levels[0].Cost)
	}
	if prices[0].Levels[0].Name != "Standard" {
		t.Errorf("Levels[0].Name = %s, want Standard", prices[0].Levels[0].Name)
	}
	if prices[0].Levels[0].Payout != 450.0 {
		t.Errorf("Levels[0].Payout = %f, want 450.0", prices[0].Levels[0].Payout)
	}
	if prices[0].Levels[1].Cost != 1350.0 {
		t.Errorf("Levels[1].Cost = %f, want 1350.0", prices[0].Levels[1].Cost)
	}
	if prices[0].Levels[1].Name != "Extended" {
		t.Errorf("Levels[1].Name = %s, want Extended", prices[0].Levels[1].Name)
	}
	if prices[0].Levels[1].Payout != 675.0 {
		t.Errorf("Levels[1].Payout = %f, want 675.0", prices[0].Levels[1].Payout)
	}
}

func TestInsurancePrices_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.InsurancePrices(context.Background()); err == nil {
		t.Fatal("InsurancePrices returned nil error, want an error for HTTP 404")
	}
}
