package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterLoyaltyPoints_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/loyalty/points" {
			t.Errorf("request path = %q, want /characters/998877/loyalty/points", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"corporation_id":1000167,"loyalty_points":15000}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	points, err := client.CharacterLoyaltyPoints(context.Background())
	if err != nil {
		t.Fatalf("CharacterLoyaltyPoints returned error: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("points = %+v, want one entry", points)
	}
	if points[0].CorporationId != 1000167 || points[0].LoyaltyPoints != 15000 {
		t.Errorf("points[0] = %+v, want CorporationId=1000167 LoyaltyPoints=15000", points[0])
	}
}

func TestCharacterLoyaltyPoints_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterLoyaltyPoints(context.Background()); err == nil {
		t.Fatal("CharacterLoyaltyPoints returned nil error, want an error for HTTP 404")
	}
}

func TestLoyaltyStoreOffers_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loyalty/stores/1000167/offers" {
			t.Errorf("request path = %q, want /loyalty/stores/1000167/offers", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"ak_cost":50,"isk_cost":10000,"lp_cost":100,"offer_id":1,"quantity":1,"required_items":[{"quantity":2,"type_id":34}],"type_id":588}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	offers, err := client.LoyaltyStoreOffers(context.Background(), 1000167)
	if err != nil {
		t.Fatalf("LoyaltyStoreOffers returned error: %v", err)
	}
	if len(offers) != 1 {
		t.Fatalf("offers = %+v, want one entry", offers)
	}
	o := offers[0]
	if o.AkCost == nil || *o.AkCost != 50 || o.IskCost != 10000 || o.LpCost != 100 || o.OfferId != 1 || o.Quantity != 1 || o.TypeId != 588 {
		t.Errorf("offers[0] = %+v, want AkCost=50 IskCost=10000 LpCost=100 OfferId=1 Quantity=1 TypeId=588", o)
	}
	if len(o.RequiredItems) != 1 || o.RequiredItems[0].Quantity != 2 || o.RequiredItems[0].TypeId != 34 {
		t.Errorf("offers[0].RequiredItems = %+v, want one entry {Quantity:2 TypeId:34}", o.RequiredItems)
	}
}

func TestLoyaltyStoreOffers_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.LoyaltyStoreOffers(context.Background(), 1000167); err == nil {
		t.Fatal("LoyaltyStoreOffers returned nil error, want an error for HTTP 404")
	}
}
