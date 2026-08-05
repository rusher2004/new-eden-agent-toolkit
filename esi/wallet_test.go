package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestWalletBalance_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("1234.56"))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	balance, err := client.WalletBalance(context.Background())
	if err != nil {
		t.Fatalf("WalletBalance returned error: %v", err)
	}
	if balance != 1234.56 {
		t.Errorf("balance = %v, want 1234.56", balance)
	}
}

func TestWalletBalance_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.WalletBalance(context.Background()); err == nil {
		t.Fatal("WalletBalance returned nil error, want an error for HTTP 403")
	}
}

// TestWalletBalance_ErrorIncludesESIErrorDetailAndRateLimitHeaders guards
// against a confirmed finding: goesi-openapi's generated error type
// degrades to a bare HTTP status string on its own (its formatErrorMessage
// helper looks for Title/Detail fields via reflection, but ESI's actual
// error model has Error/Details fields instead), and the *http.Response
// carrying CCP's error-limit headers was being discarded entirely.
func TestWalletBalance_ErrorIncludesESIErrorDetailAndRateLimitHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Esi-Error-Limit-Remain", "42")
		w.Header().Set("X-Esi-Error-Limit-Reset", "17")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"invalid_token: the token is expired"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	_, err := client.WalletBalance(context.Background())
	if err == nil {
		t.Fatal("WalletBalance returned nil error, want an error for HTTP 403")
	}
	msg := err.Error()
	if !strings.Contains(msg, "invalid_token: the token is expired") {
		t.Errorf("error message = %q, want it to contain the real ESI error detail", msg)
	}
	if !strings.Contains(msg, "error limit remaining: 42") || !strings.Contains(msg, "resets in 17s") {
		t.Errorf("error message = %q, want it to contain the error-limit rate headers", msg)
	}
}

func TestCharacterWalletJournal_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"date":"2026-08-01T00:00:00Z","description":"desc","id":1,"ref_type":"player_donation"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	entries, totalPages, err := client.CharacterWalletJournal(context.Background(), 2)
	if err != nil {
		t.Fatalf("CharacterWalletJournal returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want \"page=2\"", gotQuery)
	}
	if totalPages != 2 || len(entries) != 1 || entries[0].Id != 1 || entries[0].RefType != "player_donation" {
		t.Errorf("entries = %+v, totalPages = %d, want one entry with Id=1, RefType=\"player_donation\", totalPages=2", entries, totalPages)
	}
}

func TestCharacterWalletJournal_PageOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterWalletJournal(context.Background(), 0); err != nil {
		t.Fatalf("CharacterWalletJournal returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
}

func TestCharacterWalletJournal_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterWalletJournal(context.Background(), 0); err == nil {
		t.Fatal("CharacterWalletJournal returned nil error, want an error for HTTP 403")
	}
}

func TestCharacterWalletTransactions_ParsesResponseAndFromIdSentWhenSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"client_id":1,"date":"2026-08-01T00:00:00Z","is_buy":true,"is_personal":true,"journal_ref_id":2,"location_id":3,"quantity":4,"transaction_id":5,"type_id":6,"unit_price":7.5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	txns, err := client.CharacterWalletTransactions(context.Background(), 100)
	if err != nil {
		t.Fatalf("CharacterWalletTransactions returned error: %v", err)
	}
	if gotQuery != "from_id=100" {
		t.Errorf("query = %q, want \"from_id=100\"", gotQuery)
	}
	if len(txns) != 1 || txns[0].TransactionId != 5 || txns[0].TypeId != 6 {
		t.Errorf("txns = %+v, want one entry with TransactionId=5, TypeId=6", txns)
	}
}

func TestCharacterWalletTransactions_FromIdOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterWalletTransactions(context.Background(), 0); err != nil {
		t.Fatalf("CharacterWalletTransactions returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (fromID=0 should omit the query param)", gotQuery)
	}
}

func TestCharacterWalletTransactions_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterWalletTransactions(context.Background(), 0); err == nil {
		t.Fatal("CharacterWalletTransactions returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationWallets_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"balance":3330000000.5,"division":1}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	wallets, err := client.CorporationWallets(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationWallets returned error: %v", err)
	}
	if len(wallets) != 1 || wallets[0].Division != 1 || wallets[0].Balance != 3330000000.5 {
		t.Errorf("wallets = %+v, want one entry with Division=1, Balance=3330000000.5", wallets)
	}
}

func TestCorporationWallets_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CorporationWallets(context.Background(), 98000001); err == nil {
		t.Fatal("CorporationWallets returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationWalletJournal_ParsesResponseAndTotalPages(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"date":"2026-08-01T00:00:00Z","description":"desc","id":1,"ref_type":"corporation_account_withdrawal"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	entries, totalPages, err := client.CorporationWalletJournal(context.Background(), 98000001, 2, 0)
	if err != nil {
		t.Fatalf("CorporationWalletJournal returned error: %v", err)
	}
	wantPath := "/corporations/98000001/wallets/2/journal"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if totalPages != 1 || len(entries) != 1 || entries[0].Id != 1 {
		t.Errorf("entries = %+v, totalPages = %d, want one entry with Id=1, totalPages=1", entries, totalPages)
	}
}

func TestCorporationWalletJournal_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationWalletJournal(context.Background(), 98000001, 2, 0); err == nil {
		t.Fatal("CorporationWalletJournal returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationWalletTransactions_ParsesResponseAndFromIdSentWhenSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"client_id":1,"date":"2026-08-01T00:00:00Z","is_buy":false,"journal_ref_id":2,"location_id":3,"quantity":4,"transaction_id":5,"type_id":6,"unit_price":7.5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	txns, err := client.CorporationWalletTransactions(context.Background(), 98000001, 2, 100)
	if err != nil {
		t.Fatalf("CorporationWalletTransactions returned error: %v", err)
	}
	if gotQuery != "from_id=100" {
		t.Errorf("query = %q, want \"from_id=100\"", gotQuery)
	}
	if len(txns) != 1 || txns[0].TransactionId != 5 || txns[0].TypeId != 6 {
		t.Errorf("txns = %+v, want one entry with TransactionId=5, TypeId=6", txns)
	}
}

func TestCorporationWalletTransactions_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CorporationWalletTransactions(context.Background(), 98000001, 2, 0); err == nil {
		t.Fatal("CorporationWalletTransactions returned nil error, want an error for HTTP 403")
	}
}
