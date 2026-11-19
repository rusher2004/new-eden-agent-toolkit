package esi

//go:generate go run ../tools/genscopes -out scopes_gen.go

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// endpointScope is one row of the generated endpointScopes table: an ESI
// path template (e.g. "/characters/{character_id}/wallet") and the scope a
// request to it with method needs, "" for a public endpoint.
type endpointScope struct {
	method, path, scope string
}

// requiredScope returns the scope an ESI request needs, if the endpoint is
// an authenticated one. path is the request's URL path, which matches a
// template segment by segment with any value standing in for "{...}".
// Public endpoints are in the table too, so a public request matches its
// own template rather than an authenticated one of the same shape.
func requiredScope(method, path string) (string, bool) {
	// The most specific match wins, so a literal segment
	// (".../mail/labels") beats a template one (".../mail/{mail_id}").
	segs := strings.Split(strings.Trim(path, "/"), "/")
	best, bestLiterals := "", -1
	for _, e := range endpointScopes {
		if e.method != method {
			continue
		}
		if literals, ok := matchPath(strings.Split(strings.Trim(e.path, "/"), "/"), segs); ok && literals > bestLiterals {
			best, bestLiterals = e.scope, literals
		}
	}
	return best, best != ""
}

// matchPath reports whether segs fits template, and how many of the
// template's segments were literal.
func matchPath(template, segs []string) (int, bool) {
	if len(template) != len(segs) {
		return 0, false
	}
	literals := 0
	for i, t := range template {
		if strings.HasPrefix(t, "{") {
			continue
		}
		if t != segs[i] {
			return 0, false
		}
		literals++
	}
	return literals, true
}

// MissingScopeError reports that a character hasn't granted neat the ESI
// scope an endpoint needs. Its message tells the agent how to fix it, since
// it reaches the agent verbatim as the tool error.
type MissingScopeError struct {
	CharacterID int32
	Scope       string
	// Group is the sso scope group holding Scope.
	Group string
}

func (e *MissingScopeError) Error() string {
	return fmt.Sprintf("character %d hasn't granted neat the ESI scope %s (scope group %q). "+
		"Ask the user whether to grant it; if they agree, call add_character with characterId %d and scopeGroups [%q] "+
		"and have them log in as that character. Their existing scopes are kept.",
		e.CharacterID, e.Scope, e.Group, e.CharacterID, e.Group)
}

// scopeCheckTransport fails a request before it is sent when the character
// hasn't granted the scope its endpoint needs, rather than letting ESI
// answer with a 403 that doesn't say which scope was missing.
type scopeCheckTransport struct {
	base        http.RoundTripper
	characterID int32
	granted     []string
}

func (t *scopeCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if scope, ok := requiredScope(req.Method, req.URL.Path); ok && !slices.Contains(t.granted, scope) {
		missing := &MissingScopeError{CharacterID: t.characterID, Scope: scope, Group: sso.ScopeGroupFor(scope)}
		// http.RoundTripper must close the request body, even on error.
		if req.Body != nil {
			if err := req.Body.Close(); err != nil {
				return nil, errors.Join(missing, fmt.Errorf("close request body: %w", err))
			}
		}
		return nil, missing
	}
	return t.base.RoundTrip(req)
}

// ExportedRequiredScope exposes requiredScope for external tests.
func ExportedRequiredScope(method, path string) (string, bool) {
	return requiredScope(method, path)
}

// ExportedEndpoint is one row of the generated table, for external tests.
type ExportedEndpoint struct {
	Method, Path, Scope string
}

// ExportedEndpointScopes returns the generated table, for external tests.
func ExportedEndpointScopes() []ExportedEndpoint {
	out := make([]ExportedEndpoint, len(endpointScopes))
	for i, e := range endpointScopes {
		out[i] = ExportedEndpoint{e.method, e.path, e.scope}
	}
	return out
}

// ExportedScopeCheckTransport exposes scopeCheckTransport for external tests.
func ExportedScopeCheckTransport(base http.RoundTripper, characterID int32, granted []string) http.RoundTripper {
	return &scopeCheckTransport{base: base, characterID: characterID, granted: granted}
}
