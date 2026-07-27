package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterContracts_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{"acceptor_id":1,"assignee_id":2,"availability":"public","contract_id":1,"date_expired":"2026-08-01T00:00:00Z","date_issued":"2026-07-01T00:00:00Z","for_corporation":false,"issuer_corporation_id":98000001,"issuer_id":998877,"status":"outstanding","type":"item_exchange"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contracts, totalPages, err := client.CharacterContracts(context.Background(), 0)
	if err != nil {
		t.Fatalf("CharacterContracts returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(contracts) != 1 || contracts[0].Status != "outstanding" {
		t.Errorf("contracts = %+v, want one entry with Status=\"outstanding\"", contracts)
	}

	_, _, err = client.CharacterContracts(context.Background(), 2)
	if err != nil {
		t.Fatalf("CharacterContracts(page=2) returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want \"page=2\"", gotQuery)
	}
}

func TestCharacterContracts_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterContracts(context.Background(), 0); err == nil {
		t.Fatal("CharacterContracts returned nil error, want an error for HTTP 403")
	}
}

func TestCharacterContractBids_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"amount":1000000,"bid_id":1,"bidder_id":998878,"date_bid":"2026-07-15T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bids, err := client.CharacterContractBids(context.Background(), 1)
	if err != nil {
		t.Fatalf("CharacterContractBids returned error: %v", err)
	}
	if len(bids) != 1 || bids[0].BidderId != 998878 {
		t.Errorf("bids = %+v, want one entry with BidderId=998878", bids)
	}
}

func TestCharacterContractItems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"is_included":true,"is_singleton":false,"quantity":1,"record_id":1,"type_id":34}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	items, err := client.CharacterContractItems(context.Background(), 1)
	if err != nil {
		t.Fatalf("CharacterContractItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].TypeId != 34 {
		t.Errorf("items = %+v, want one entry with TypeId=34", items)
	}
}

func TestCorporationContracts_ParsesResponseAndPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"acceptor_id":1,"assignee_id":2,"availability":"public","contract_id":2,"date_expired":"2026-08-01T00:00:00Z","date_issued":"2026-07-01T00:00:00Z","for_corporation":true,"issuer_corporation_id":98000001,"issuer_id":998877,"status":"outstanding","type":"courier"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contracts, totalPages, err := client.CorporationContracts(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationContracts returned error: %v", err)
	}
	if totalPages != 2 || len(contracts) != 1 || contracts[0].Type != "courier" {
		t.Errorf("contracts = %+v, totalPages = %d, want totalPages=2 with one courier contract", contracts, totalPages)
	}
}

func TestCorporationContractBids_UsesReversedParameterOrder(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"amount":500000,"bid_id":5,"bidder_id":998879,"date_bid":"2026-07-15T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bids, totalPages, err := client.CorporationContractBids(context.Background(), 98000001, 3, 0)
	if err != nil {
		t.Fatalf("CorporationContractBids returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/contracts/3/bids" {
		t.Errorf("path = %q, want \"/corporations/98000001/contracts/3/bids\" (corporationID=98000001, contractID=3, not swapped)", gotPath)
	}
	if totalPages != 1 || len(bids) != 1 || bids[0].BidId != 5 {
		t.Errorf("bids = %+v, totalPages = %d, want one bid with BidId=5 and totalPages=1", bids, totalPages)
	}
}

func TestCorporationContractItems_UsesReversedParameterOrder(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"is_included":true,"is_singleton":true,"quantity":1,"record_id":1,"type_id":648}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	items, err := client.CorporationContractItems(context.Background(), 98000001, 3)
	if err != nil {
		t.Fatalf("CorporationContractItems returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/contracts/3/items" {
		t.Errorf("path = %q, want \"/corporations/98000001/contracts/3/items\"", gotPath)
	}
	if len(items) != 1 || items[0].TypeId != 648 {
		t.Errorf("items = %+v, want one entry with TypeId=648", items)
	}
}

func TestPublicContracts_ParsesResponseWithNoStatusField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "5")
		w.Write([]byte(`[{"contract_id":10,"date_expired":"2026-08-01T00:00:00Z","date_issued":"2026-07-01T00:00:00Z","issuer_corporation_id":98000001,"issuer_id":998877,"type":"item_exchange"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contracts, totalPages, err := client.PublicContracts(context.Background(), 10000002, 0)
	if err != nil {
		t.Fatalf("PublicContracts returned error: %v", err)
	}
	if totalPages != 5 || len(contracts) != 1 || contracts[0].ContractId != 10 {
		t.Errorf("contracts = %+v, totalPages = %d, want totalPages=5 with one entry ContractId=10", contracts, totalPages)
	}
}

func TestPublicContractBids_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"amount":250000,"bid_id":7,"date_bid":"2026-07-15T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bids, totalPages, err := client.PublicContractBids(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("PublicContractBids returned error: %v", err)
	}
	if totalPages != 1 || len(bids) != 1 || bids[0].BidId != 7 {
		t.Errorf("bids = %+v, totalPages = %d, want one bid with BidId=7 and totalPages=1", bids, totalPages)
	}
}

func TestPublicContractItems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"is_included":true,"item_id":123,"quantity":1,"record_id":1,"type_id":34}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	items, totalPages, err := client.PublicContractItems(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("PublicContractItems returned error: %v", err)
	}
	if totalPages != 1 || len(items) != 1 || items[0].TypeId != 34 {
		t.Errorf("items = %+v, totalPages = %d, want one entry with TypeId=34 and totalPages=1", items, totalPages)
	}
}
