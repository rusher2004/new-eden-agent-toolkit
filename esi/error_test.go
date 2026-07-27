package esi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"
)

// TestDescribeError_NilErrReturnsNil directly unit-tests describeError's nil-passthrough branch.
func TestDescribeError_NilErrReturnsNil(t *testing.T) {
	if err := describeError(nil, nil); err != nil {
		t.Errorf("describeError(nil, nil) = %v, want nil", err)
	}
	if err := describeError(nil, &http.Response{}); err != nil {
		t.Errorf("describeError(nil, resp) = %v, want nil", err)
	}
}

// TestDescribeError_NonESIErrorWithNilResp directly unit-tests describeError with a plain Go
// error and no response -- neither the ESI-detail branch nor the rate-limit branch applies, so
// the original error must come back unwrapped and unchanged.
func TestDescribeError_NonESIErrorWithNilResp(t *testing.T) {
	orig := &http.ProtocolError{ErrorString: "boom"}
	got := describeError(orig, nil)
	if got != orig {
		t.Errorf("describeError = %v, want the exact original error returned unwrapped", got)
	}
}

// TestDescribeError_NonESIErrorWithNoRateLimitHeaders directly unit-tests that a response with no
// X-Esi-Error-Limit-Remain header doesn't get wrapped.
func TestDescribeError_NonESIErrorWithNoRateLimitHeaders(t *testing.T) {
	orig := &http.ProtocolError{ErrorString: "boom"}
	resp := &http.Response{Header: http.Header{}}
	got := describeError(orig, resp)
	if got != orig {
		t.Errorf("describeError = %v, want the exact original error returned unwrapped", got)
	}
}

// TestDescribeError_AddsRateLimitHeadersWithoutESIDetail directly unit-tests the rate-limit-only
// branch: a non-ESI error (so the Model()-based detail extraction can't apply) with a response
// that DOES carry the error-limit headers must still get those headers folded in.
func TestDescribeError_AddsRateLimitHeadersWithoutESIDetail(t *testing.T) {
	orig := &http.ProtocolError{ErrorString: "boom"}
	resp := &http.Response{Header: http.Header{
		"X-Esi-Error-Limit-Remain": []string{"10"},
		"X-Esi-Error-Limit-Reset":  []string{"5"},
	}}
	got := describeError(orig, resp)
	if got == orig {
		t.Fatal("describeError did not wrap the error despite rate-limit headers being present")
	}
	msg := got.Error()
	if !strings.Contains(msg, "error limit remaining: 10") || !strings.Contains(msg, "resets in 5s") {
		t.Errorf("error message = %q, want it to contain the error-limit rate headers", msg)
	}
}

// TestDescribeError_SurfacesRealESIErrorDetailAndRateLimitHeaders directly unit-tests describeError
// against a real *esiapi.GenericOpenAPIError -- which can only be constructed via an actual failed
// request through the generated client, since its fields are unexported. Uses the public Status
// endpoint (arbitrary choice -- describeError has no per-endpoint logic) purely as a vehicle to
// obtain a real (err, resp) pair; the assertions are entirely about describeError's own output,
// not about esi.Status's behavior.
func TestDescribeError_SurfacesRealESIErrorDetailAndRateLimitHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Esi-Error-Limit-Remain", "42")
		w.Header().Set("X-Esi-Error-Limit-Reset", "17")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"invalid_token: the token is expired"}`))
	}))
	defer srv.Close()

	api := goesi.NewESIClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	client := &Client{api: api}

	_, resp, err := client.api.StatusAPI.GetStatus(context.Background()).Execute()
	if err == nil {
		t.Fatal("expected the generated client to return an error for HTTP 403")
	}

	got := describeError(err, resp)
	msg := got.Error()
	if !strings.Contains(msg, "invalid_token: the token is expired") {
		t.Errorf("error message = %q, want it to contain the real ESI error detail", msg)
	}
	if !strings.Contains(msg, "error limit remaining: 42") || !strings.Contains(msg, "resets in 17s") {
		t.Errorf("error message = %q, want it to contain the error-limit rate headers", msg)
	}
}
