package esi

import (
	"context"
	"fmt"
	"net/http"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// Client is a thin, per-character wrapper over the generated ESI API
// client. esi/ itself holds no state across calls — sso.AuthenticatedClient
// owns the one cache that matters (the auto-refreshing HTTP client).
type Client struct {
	api         *esiapi.APIClient
	characterID int32
}

// NewClient builds an authenticated ESI client for characterID.
func NewClient(ctx context.Context, clientID string, characterID int32, userAgent string) (*Client, error) {
	httpClient, err := sso.AuthenticatedClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build authenticated client: %w", err)
	}
	return newClientFromHTTPClient(httpClient, characterID, userAgent), nil
}

func newClientFromHTTPClient(httpClient *http.Client, characterID int32, userAgent string) *Client {
	api := goesi.NewESIClientWithOptions(httpClient, goesi.ClientOptions{UserAgent: userAgent})
	return &Client{api: api, characterID: characterID}
}

// ExportedNewClientWithOptions exposes newClientFromHTTPClient for external
// tests, additionally allowing a BaseURL override so tests can point the
// generated client at an httptest.Server instead of real ESI.
func ExportedNewClientWithOptions(httpClient *http.Client, characterID int32, options goesi.ClientOptions) *Client {
	api := goesi.NewESIClientWithOptions(httpClient, options)
	return &Client{api: api, characterID: characterID}
}

// NewPublicClient builds an unauthenticated ESI client for calling public endpoints — no
// character, no OAuth2 token. Used by resource groups where every endpoint is public (e.g.
// Alliance); see docs/superpowers/specs/2026-07-25-esi-alliance-design.md.
func NewPublicClient(userAgent string) *Client {
	api := goesi.NewPublicESIClient(userAgent)
	return &Client{api: api}
}

// ExportedNewPublicClientWithOptions exposes a Client built from an explicit httpClient and
// public ClientOptions for external tests, so tests can point the generated client at an
// httptest.Server instead of real ESI.
func ExportedNewPublicClientWithOptions(httpClient *http.Client, options goesi.ClientOptions) *Client {
	api := goesi.NewESIClientWithOptions(httpClient, options)
	return &Client{api: api}
}
