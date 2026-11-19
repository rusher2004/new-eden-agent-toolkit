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
	if strings.Contains(msg, "response body:") {
		t.Errorf("error message = %q, want no raw body when ESI's own error detail was extracted", msg)
	}
}

// TestDescribeError_DecodeFailureSurfacesRawBody reproduces the transient failure seen live in
// the 2026-11-18 audit: a 2xx whose JSON lacks a property the generated model requires. The
// error must carry what ESI actually sent, not only what the model expected.
func TestDescribeError_DecodeFailureSurfacesRawBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"cursor":{}}`))
	}))
	defer srv.Close()

	api := goesi.NewESIClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	client := &Client{api: api, characterID: 998877}

	_, _, err := client.ParagonHubSkinrListingsForCharacter(context.Background(), "", "", 0)
	if err == nil {
		t.Fatal("expected an error for a 200 response missing the required listings property")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no value given for required property listings") {
		t.Errorf("error message = %q, want it to keep the decode error", msg)
	}
	if !strings.Contains(msg, `response body: "{\"cursor\":{}}"`) {
		t.Errorf("error message = %q, want it to include the raw response body", msg)
	}
}

func TestDescribeError_NonESIErrorBodySurfacesRawBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<html>upstream timed out</html>`))
	}))
	defer srv.Close()

	api := goesi.NewESIClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	_, resp, err := api.StatusAPI.GetStatus(context.Background()).Execute()
	if err == nil {
		t.Fatal("expected the generated client to return an error for HTTP 502")
	}

	msg := describeError(err, resp).Error()
	if !strings.Contains(msg, "upstream timed out") {
		t.Errorf("error message = %q, want it to include the non-ESI error body", msg)
	}
}

func TestTruncateBody(t *testing.T) {
	short := []byte(`{"a":1}`)
	if got, want := truncateBody(short), `"{\"a\":1}"`; got != want {
		t.Errorf("truncateBody(short) = %s, want %s", got, want)
	}

	long := []byte(strings.Repeat("x", maxErrorBodyBytes+100))
	got := truncateBody(long)
	if !strings.HasSuffix(got, "... (100 more bytes)") {
		t.Errorf("truncateBody(long) = %q, want a truncation marker noting 100 more bytes", got)
	}
	if strings.Count(got, "x") != maxErrorBodyBytes {
		t.Errorf("truncateBody(long) kept %d bytes of body, want %d", strings.Count(got, "x"), maxErrorBodyBytes)
	}

	// A 3-byte rune straddling the cut must be dropped whole, not split
	// into \x escapes.
	straddle := []byte(strings.Repeat("x", maxErrorBodyBytes-1) + "€" + strings.Repeat("y", 10))
	got = truncateBody(straddle)
	if strings.Contains(got, `\x`) {
		t.Errorf("truncateBody(straddle) = %q, want no split-rune \\x escapes", got)
	}
	if !strings.HasSuffix(got, "... (13 more bytes)") {
		t.Errorf("truncateBody(straddle) = %q, want a marker counting the dropped rune's bytes (13 more bytes)", got)
	}
}
