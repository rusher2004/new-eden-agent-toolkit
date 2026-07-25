package sso

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserAgent_Format(t *testing.T) {
	got := UserAgent("1.2.3")
	want := "neat/1.2.3 (robertinthecloud@icloud.com; +https://github.com/rusher2004/new-eden-agent-toolkit)"
	if got != want {
		t.Errorf("UserAgent(%q) = %q, want %q", "1.2.3", got, want)
	}
}

func TestUserAgentTransport_SetsHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := userAgentClient("neat/test-agent")
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	resp.Body.Close()

	if gotHeader != "neat/test-agent" {
		t.Errorf("received User-Agent = %q, want %q", gotHeader, "neat/test-agent")
	}
}
