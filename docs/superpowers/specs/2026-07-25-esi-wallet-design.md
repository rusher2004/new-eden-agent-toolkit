# `esi/` Package + First Domain Tool (`get_wallet_balance`) — Design

## Goal

Build the `esi/` package (the thin facade over `github.com/fnt-eve/goesi-openapi` that the
foundation design spec already calls for) and one real, end-to-end ESI-backed MCP tool,
`get_wallet_balance`, proving the full pipeline — stored refresh token → authenticated ESI
call → MCP tool result — now that character auth (Plans 2a/2b) is live and confirmed working
against real EVE SSO.

This follows directly from `docs/superpowers/specs/2026-07-23-foundation-design.md`, which
specifies `esi/`'s responsibility and `sso/`'s ownership of the token lifecycle. It refines —
does not contradict — that spec's assumed shape for token refresh, based on API behavior
discovered while designing this plan (see below).

## Key finding: token refresh needs no manual expiry tracking

The foundation spec envisioned `sso.ValidAccessToken(ctx, characterID) (string, error)`,
checking expiry and refreshing-then-persisting transparently. In practice,
`goesi.Config.Client(ctx, token) *http.Client` — called internally by
`goesi.NewAuthenticatedESIClient` — is just `golang.org/x/oauth2`'s own `Config.Client()`.
Given an `oauth2.Token{RefreshToken: <stored>}` (zero-value `AccessToken`/`Expiry`), the
returned `*http.Client` automatically exchanges the refresh token for a new access token on
first use and transparently re-refreshes only when the cached one actually expires — entirely
via `golang.org/x/oauth2`'s own battle-tested `ReuseTokenSource` machinery. No manual expiry
checking, no manual call to `goesi.Config.RefreshToken`, and — since this path never parses JWT
claims — no JWKS fetch either (that's only needed to verify/derive a character's identity from
a *fresh login*, not to refresh a already-known character's token).

Per [EVE's own SSO documentation](https://developers.eveonline.com/docs/services/sso/),
refresh tokens are long-lived and reusable indefinitely (not rotated on each use), so nothing
needs to be re-persisted to `creds/` after a refresh — the same stored refresh token remains
valid for the life of the character's authorization.

**Consequence:** `sso/`'s job simplifies to caching a per-character auto-refreshing
`*http.Client` in memory (per your choice, rather than always-refresh), and `esi/` stays
fully stateless — it just asks `sso/` for a client "per request," and `sso/` decides whether
that's a cache hit.

## `sso/client.go` (new)

```go
package sso

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	goesi "github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

var (
	clientCacheMu sync.Mutex
	clientCache   = map[int32]*http.Client{}
)

// AuthenticatedClient returns an authenticated, auto-refreshing HTTP client
// for characterID, suitable for making real ESI API calls. The first call
// for a given character builds and caches the client; later calls for the
// same character reuse it — the returned client's own internal token
// source handles refreshing transparently, so callers never need to check
// expiry themselves.
func AuthenticatedClient(ctx context.Context, clientID string, characterID int32, userAgent string) (*http.Client, error) {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()

	if c, ok := clientCache[characterID]; ok {
		return c, nil
	}

	refreshToken, err := creds.CharacterRefreshToken(characterID)
	if err != nil {
		return nil, fmt.Errorf("load refresh token for character %d: %w", characterID, err)
	}

	// No JWKS keyfunc needed: this path never verifies/parses JWT claims,
	// only golang.org/x/oauth2's own refresh-token grant flow.
	cfg, err := goesi.NewConfig(clientID, RedirectURI, nil)
	if err != nil {
		return nil, fmt.Errorf("build ESI OAuth2 config: %w", err)
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, userAgentClient(userAgent))
	token := &oauth2.Token{RefreshToken: refreshToken}
	client := cfg.Client(ctx, token)

	clientCache[characterID] = client
	return client, nil
}

// InvalidateClientCache evicts characterID's cached authenticated client,
// if any. Called by remove_character so a stale in-memory client can't
// keep working against a refresh token that's just been deleted from the
// keychain.
func InvalidateClientCache(characterID int32) {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()
	delete(clientCache, characterID)
}
```

**Testing** (`sso/client_test.go`, network-free — `oauth2.Config.Client()` is lazy, so building
a client never makes an HTTP call on its own):
- Same character, called twice → the exact same `*http.Client` pointer both times (proves the
  cache hit).
- Two different characters (both with seeded refresh tokens via `creds.SaveCharacterRefreshToken`
  + `keyring.MockInit()`) → distinct pointers.
- Unknown character (no stored refresh token) → wrapped `creds.ErrNotFound`.
- `InvalidateClientCache` then a repeat call for the same character → a *new*, different pointer
  than before invalidation.

## `esi/` package (new)

Naming note: our package is `esi/`, and `goesi-openapi`'s generated API types live in a nested
package also literally named `esi` (`github.com/fnt-eve/goesi-openapi/esi`) — files in our `esi/`
package alias that import (`esiapi "github.com/fnt-eve/goesi-openapi/esi"`), matching the
existing `commands`/`cli` and `mcp`/`mcpsdk` aliasing convention.

**`esi/client.go`:**
```go
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
```

**`esi/wallet.go`:**
```go
package esi

import (
	"context"
	"fmt"
)

// WalletBalance returns the client's character's current ISK wallet balance.
func (c *Client) WalletBalance(ctx context.Context) (float64, error) {
	balance, _, err := c.api.WalletAPI.GetCharactersCharacterIdWallet(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return 0, fmt.Errorf("get wallet balance: %w", err)
	}
	return balance, nil
}
```

Verified against the actual `goesi-openapi` v0.0.37 API: `WalletAPI.GetCharactersCharacterIdWallet(ctx, characterId int64) ApiGetCharactersCharacterIdWalletRequest`,
`.Execute() (float64, *http.Response, error)`. The generated client's default `esi.NewConfiguration()`
already sets a current `CompatibilityDate`, so no extra configuration is needed for this call.

**Testing** (`esi/wallet_test.go`, external `esi_test` package): an `httptest.Server` standing in
for ESI, returning a JSON float64 body for `GET /characters/{id}/wallet/`. Built via
`esi.ExportedNewClientWithOptions(srv.Client(), characterID, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test"})`
— bypassing `sso` entirely, so this layer's tests never touch auth or the token cache.

## `mcp/wallet.go` (new)

Grouped by ESI resource (wallet), not by individual tool — room for future tools on the same
resource (wallet journal, transactions) without a rename, matching how `goesi-openapi` itself
groups `WalletAPI`'s several endpoints together.

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// getWalletBalance is overridable in tests so no real network/auth runs.
var getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.WalletBalance(ctx)
}

// GetWalletBalanceInput is get_wallet_balance's tool input.
type GetWalletBalanceInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// GetWalletBalanceOutput is get_wallet_balance's tool result.
type GetWalletBalanceOutput struct {
	CharacterID int32   `json:"characterId"`
	BalanceISK  float64 `json:"balanceIsk"`
}

func getWalletBalanceHandler(version, clientID string) mcpsdk.ToolHandlerFor[GetWalletBalanceInput, GetWalletBalanceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetWalletBalanceInput) (*mcpsdk.CallToolResult, GetWalletBalanceOutput, error) {
		balance, err := getWalletBalance(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("get wallet balance: %w", err)
		}
		return nil, GetWalletBalanceOutput{CharacterID: in.CharacterID, BalanceISK: balance}, nil
	}
}
```

Requires an explicit `characterId` input — no "just pick the only character" magic, consistent
with `remove_character`'s existing pattern, since zero-or-more characters is a first-class case.

**Testing** (`mcp/wallet_internal_test.go`, internal package — needs the unexported
`getWalletBalanceHandler`/`getWalletBalance` var): same func-var override pattern as
`mcp/add_character.go`.

## `mcp/server.go` and `mcp/remove_character.go` changes

`NewServer` registers a fourth tool, `get_wallet_balance`, alongside the existing three.

`remove_character`'s handler calls `sso.InvalidateClientCache(removed.ID)` after successfully
saving the updated config (same ordering rationale as the existing creds-then-config sequence:
invalidate the in-memory cache only once the character is durably gone, not before). Its test
gains a case confirming the cache-invalidation call happens (via an overridable
`var invalidateClientCache = sso.InvalidateClientCache`, matching `add_character.go`'s pattern
of wrapping any call this package needs to fake in tests).

## Error handling

Consistent with the rest of the codebase: `fmt.Errorf("...: %w", err)` wrapping at every layer.
An unknown character ID surfaces as a wrapped `creds.ErrNotFound` propagating naturally from
`creds` → `sso.AuthenticatedClient` → `esi.NewClient` → the tool handler — no special-casing
needed. ESI-side HTTP errors (bad scope, rate limiting, etc.) propagate as a wrapped error from
`Execute()`; no ESI-specific error-code parsing for this first tool (YAGNI — revisit once more
than one tool needs it).

## Scopes

No new scope needed — `esi-wallet.read_character_wallet.v1` is already in `add_character`'s
full requested scope list from Plan 2b.

## File layout summary

- Create: `sso/client.go`, `sso/client_test.go`
- Create: `esi/client.go`, `esi/wallet.go`, `esi/wallet_test.go`
- Create: `mcp/wallet.go`, `mcp/wallet_internal_test.go`
- Modify: `mcp/server.go` (register `get_wallet_balance`), `mcp/remove_character.go` +
  `mcp/remove_character_internal_test.go` (cache invalidation)
- Modify: `CLAUDE.md`/`README.md` package tables (new `esi/` row, `mcp/`'s row updated to list
  four tools)

## Explicitly out of scope for this plan

- Any ESI domain tool beyond `get_wallet_balance` (wallet journal/transactions, character
  location, skill queue, etc.) — deferred to follow-up tools on the same `esi/wallet.go`/
  `mcp/wallet.go` files or new resource-scoped files, once chosen.
- SDE cache (`sde/` package) — this tool was deliberately chosen to need no SDE lookups.
- ESI-specific error-code handling/retries (rate limiting backoff, etc.) — revisit once a
  second tool's needs clarify what's actually worth building.
- Updating `goesi.Config.RefreshToken`'s own JWT-claims-parsing path to use the User-Agent
  transport — not exercised by this plan at all (this plan's refresh path goes through plain
  `oauth2.Config.Client()`, never `goesi.Config.RefreshToken`), so it stays exactly as flagged
  in Plan 2b's known-gap note.
