package esi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

var templateParam = regexp.MustCompile(`\{[^}]+\}`)

// TestRequiredScope_EveryEndpointResolvesToItself builds a concrete request
// path from every generated template and checks it resolves to that
// template's own scope, so no public endpoint is held to an authenticated
// one's scope and no authenticated endpoint is matched by another.
func TestRequiredScope_EveryEndpointResolvesToItself(t *testing.T) {
	for _, e := range esi.ExportedEndpointScopes() {
		path := templateParam.ReplaceAllString(e.Path, "1001")
		got, ok := esi.ExportedRequiredScope(e.Method, path)
		if got != e.Scope || ok != (e.Scope != "") {
			t.Errorf("%s %s resolved to (%q, %v), want (%q, %v)", e.Method, path, got, ok, e.Scope, e.Scope != "")
		}
	}
}

func TestRequiredScope_LiteralSegmentBeatsTemplate(t *testing.T) {
	got, _ := esi.ExportedRequiredScope(http.MethodGet, "/characters/1001/mail/labels")
	if got != "esi-mail.read_mail.v1" {
		t.Errorf("mail labels scope = %q", got)
	}
	if _, ok := esi.ExportedRequiredScope(http.MethodGet, "/markets/10000002/history"); ok {
		t.Error("public market history was given a scope")
	}
}

// TestEndpointScopes_EveryScopeHasAGroup guards against an ESI endpoint
// whose scope no login can request: after a goesi-openapi bump adds one,
// it must be put in an sso scope group.
func TestEndpointScopes_EveryScopeHasAGroup(t *testing.T) {
	for _, e := range esi.ExportedEndpointScopes() {
		if e.Scope != "" && sso.ScopeGroupFor(e.Scope) == "" {
			t.Errorf("%s %s needs %s, which is in no sso scope group", e.Method, e.Path, e.Scope)
		}
	}
}

// TestScopeGroups_EveryScopeIsUsed guards against a group requesting a
// scope no endpoint needs (a typo, or one FC retired).
func TestScopeGroups_EveryScopeIsUsed(t *testing.T) {
	used := map[string]bool{"publicData": true}
	for _, e := range esi.ExportedEndpointScopes() {
		used[e.Scope] = true
	}
	for _, g := range sso.ScopeGroups {
		for _, s := range g.Scopes {
			if !used[s] {
				t.Errorf("group %q requests %s, which no ESI endpoint needs", g.Name, s)
			}
		}
	}
}

func TestScopeCheckTransport(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(upstream.Close)

	client := &http.Client{Transport: esi.ExportedScopeCheckTransport(http.DefaultTransport, 1001, []string{"esi-skills.read_skills.v1"})}

	tests := []struct {
		name, path  string
		wantMissing string
	}{
		{"granted", "/characters/1001/skills", ""},
		{"public", "/characters/1001", ""},
		{"missing", "/characters/1001/wallet", "esi-wallet.read_character_wallet.v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(upstream.URL + tt.path)
			if tt.wantMissing == "" {
				if err != nil {
					t.Fatalf("GET %s: %v", tt.path, err)
				}
				resp.Body.Close()
				return
			}
			var missing *esi.MissingScopeError
			if !errors.As(err, &missing) {
				t.Fatalf("err = %v, want a MissingScopeError", err)
			}
			if missing.Scope != tt.wantMissing || missing.Group != "wallet" || missing.CharacterID != 1001 {
				t.Errorf("MissingScopeError = %+v", missing)
			}
			if !strings.Contains(err.Error(), `scopeGroups ["wallet"]`) {
				t.Errorf("error doesn't say how to fix it: %v", err)
			}
		})
	}
}
