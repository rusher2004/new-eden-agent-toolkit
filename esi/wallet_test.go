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
