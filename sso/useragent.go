package sso

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	contactEmail = "robertinthecloud@icloud.com"
	repoURL      = "https://github.com/rusher2004/new-eden-agent-toolkit"

	// jwksURL is duplicated from goesi-openapi's unexported esiJWKSURL
	// constant. goesi.ESIDefaultKeyfunc gives no way to inject a custom
	// HTTP client, so this file builds the keyfunc itself instead of using
	// that helper. If goesi-openapi ever adds User-Agent support directly,
	// this duplication and newKeyfunc can go away in favor of calling
	// ESIDefaultKeyfunc again.
	jwksURL = "https://login.eveonline.com/oauth/jwks"
)

// UserAgent builds neat's User-Agent string for outbound FC requests, per
// https://developers.eveonline.com/docs/services/esi/best-practices/#user-agents.
func UserAgent(version string) string {
	return fmt.Sprintf("neat/%s (%s; +%s)", version, contactEmail, repoURL)
}

// userAgentTransport sets User-Agent on every outgoing request.
type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", t.userAgent)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func userAgentClient(userAgent string) *http.Client {
	return &http.Client{Transport: &userAgentTransport{userAgent: userAgent}}
}

// newKeyfunc builds the ESI JWT-verification keyfunc using a client that
// sends userAgent, replacing goesi.ESIDefaultKeyfunc (see jwksURL above).
// RefreshInterval/RefreshErrorHandler match ESIDefaultKeyfunc's own
// refresh behavior (an hourly background refresh, logged on failure) —
// parity with the code being replaced, not a new requirement.
func newKeyfunc(ctx context.Context, userAgent string) (jwt.Keyfunc, error) {
	storage, err := jwkset.NewStorageFromHTTP(jwksURL, jwkset.HTTPClientStorageOptions{
		Client:          userAgentClient(userAgent),
		Ctx:             ctx,
		RefreshInterval: time.Hour,
		RefreshErrorHandler: func(ctx context.Context, err error) {
			slog.Default().ErrorContext(ctx, "failed to refresh ESI JWKS", "error", err)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ESI JWT signing keys: %w", err)
	}
	kf, err := keyfunc.New(keyfunc.Options{Storage: storage})
	if err != nil {
		return nil, fmt.Errorf("build JWKS keyfunc: %w", err)
	}
	return kf.Keyfunc, nil
}
