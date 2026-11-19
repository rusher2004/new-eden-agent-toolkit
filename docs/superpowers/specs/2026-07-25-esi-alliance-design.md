# ESI Alliance (Third Batch) Design

## Status

Approved. Third batch of the ESI coverage sweep (`docs/reference/esi-coverage-roadmap.md`),
covering the Alliance tag alone (4 endpoints). Batches 1 (Wallet) and 2 (Access List + Activities,
PR #6) precede this one.

## Goal

Cover the Alliance resource group. Unlike every prior tag, all four Alliance endpoints are public
— verified against FC's live OpenAPI spec, every operation has `security: []`. This introduces a
new convention: public/unauthenticated tags don't need a per-character existence check or even a
logged-in character at all.

## Why Alliance is scoped alone, not bundled

The next three tags in the roadmap's order are Alliance (4), Assets (6), Calendar (4). Bundling
them was considered and rejected for this batch:
- Assets includes POST-based sub-resources (`PostCharactersCharacterIdAssetsLocations`,
  `PostCharactersCharacterIdAssetsNames`) and separate corporation-scoped variants — meaningfully
  more design surface than a simple GET facade.
- Calendar includes `PutCharactersCharacterIdCalendarEventId` — the first **write** endpoint
  encountered in this sweep (responding to a calendar invite). Mutating actions deserve their own
  focused design (naming convention, safety considerations) rather than being introduced
  incidentally inside a bundled batch.
- Alliance alone already introduces one new pattern (public/unauthenticated tools). Keeping it
  isolated lets that pattern be designed and reviewed cleanly.

Assets and Calendar remain "Planned" in the roadmap doc and will each get their own
brainstorm/spec/plan when picked up.

## New convention: public/unauthenticated tags

Verified via FC's live OpenAPI spec (`https://esi.evetech.net/meta/openapi-3.0.json`): all four
Alliance operations have `security: []`.

```
GET /alliances                          -> PUBLIC (no auth)
GET /alliances/{alliance_id}            -> PUBLIC (no auth)
GET /alliances/{alliance_id}/corporations -> PUBLIC (no auth)
GET /alliances/{alliance_id}/icons      -> PUBLIC (no auth)
```

This differs from `list_raidable_skyhooks` (Activities batch): that was one public endpoint inside
an otherwise character-scoped tag, where keeping a `characterId` input made sense for consistency
with its siblings (`list_mercenary_tactical_operations`, `get_mercenary_tactical_operation`). Here,
**every** endpoint in the tag is public — there's no "self" to scope to, so:

- `esi/client.go` gains a second constructor, `NewPublicClient(userAgent string) *Client`, wrapping
  `goesi.NewPublicESIClient(userAgent)` (verified to exist in `github.com/fnt-eve/goesi-openapi@v0.0.37`,
  uses `http.DefaultClient`, sets the User-Agent header correctly — confirmed via a live
  `httptest.Server` round-trip during design).
- MCP tools built on a fully-public tag take **no `characterId` input at all** and skip the
  `config.Store` existence check entirely.
- This applies only when *every* endpoint in a tag is public. A mixed tag still follows the
  existing per-character pattern for consistency with its authenticated siblings — this is not a
  blanket relaxation of the existence-check convention.

## `esi/alliance.go`

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Alliances returns every alliance ID currently in the game.
func (c *Client) Alliances(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliances(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliances: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceDetail returns one alliance's name, ticker, founding info, and optional
// executor-corporation/faction-warfare affiliation.
func (c *Client) AllianceDetail(ctx context.Context, allianceID int64) (*esiapi.AllianceDetail, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceId(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance detail: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceCorporations returns the corporation IDs currently in one alliance.
func (c *Client) AllianceCorporations(ctx context.Context, allianceID int64) ([]int64, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceIdCorporations(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance corporations: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceIcons returns one alliance's icon URLs, if it has any.
func (c *Client) AllianceIcons(ctx context.Context, allianceID int64) (*esiapi.AlliancesAllianceIdIconsGet, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceIdIcons(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance icons: %w", describeError(err, resp))
	}
	return result, nil
}
```

Verified generated signatures (from `github.com/fnt-eve/goesi-openapi@v0.0.37/esi/api_alliance.go`):
- `GetAlliances(ctx context.Context) ...`, `.Execute() ([]int64, *http.Response, error)`
- `GetAlliancesAllianceId(ctx context.Context, allianceId int64) ...`, `.Execute() (*AllianceDetail, *http.Response, error)`
- `GetAlliancesAllianceIdCorporations(ctx context.Context, allianceId int64) ...`, `.Execute() ([]int64, *http.Response, error)`
- `GetAlliancesAllianceIdIcons(ctx context.Context, allianceId int64) ...`, `.Execute() (*AlliancesAllianceIdIconsGet, *http.Response, error)`

Response models (verified from `model_alliance_detail.go` / `model_alliances_alliance_id_icons_get.go`,
including a real JSON round-trip through both a full and a minimal payload to confirm optional
fields correctly unmarshal to `nil` rather than erroring):
```go
type AllianceDetail struct {
	CreatorCorporationId   int64      `json:"creator_corporation_id"`
	CreatorId              int64      `json:"creator_id"`
	DateFounded            time.Time  `json:"date_founded"`
	ExecutorCorporationId  *int64     `json:"executor_corporation_id,omitempty"`
	FactionId              *int64     `json:"faction_id,omitempty"`
	Name                   string     `json:"name"`
	Ticker                 string     `json:"ticker"`
}
type AlliancesAllianceIdIconsGet struct {
	Px128x128 *string `json:"px128x128,omitempty"`
	Px64x64   *string `json:"px64x64,omitempty"`
}
```
`ExecutorCorporationId`, `FactionId`, `Px128x128`, `Px64x64` are all pointer fields — nil when the
alliance has no executor corp, isn't faction-warfare-aligned, or has no icon set. These pass
through as pointers in the MCP output DTOs, never dereferenced unchecked.

## `esi/client.go` addition

```go
// NewPublicClient builds an unauthenticated ESI client for calling public endpoints — no
// character, no OAuth2 token. Used by resource groups where every endpoint is public (e.g.
// Alliance); see docs/superpowers/specs/2026-07-25-esi-alliance-design.md.
func NewPublicClient(userAgent string) *Client {
	api := goesi.NewPublicESIClient(userAgent)
	return &Client{api: api}
}
```

`characterID` is left as its zero value on a client built this way — safe, since no Alliance
method reads `c.characterID` at all (every method's only ID parameter is `allianceID`, passed
explicitly).

For tests, `esi/client.go` also needs a public-client analogue of the existing
`ExportedNewClientWithOptions` (which requires a `characterID` that Alliance tests don't need, but
still needs an explicit `httpClient` param so tests can point at an `httptest.Server` via
`options.BaseURL` — mirroring `ExportedNewClientWithOptions`'s existing shape exactly, just without
the `characterID` parameter):
```go
// ExportedNewPublicClientWithOptions exposes a Client built from an explicit httpClient and
// public ClientOptions for external tests, so tests can point the generated client at an
// httptest.Server instead of real ESI.
func ExportedNewPublicClientWithOptions(httpClient *http.Client, options goesi.ClientOptions) *Client {
	api := goesi.NewESIClientWithOptions(httpClient, options)
	return &Client{api: api}
}
```

## `mcp/alliance.go`

Four tools, none taking a `characterId` input — the first tools in the project with no character
existence check at all:

- `list_alliances` — input: none, output `{allianceIds []int64}` (every alliance ID in the game;
  returned as-is, no client-side pagination added since ESI itself doesn't paginate this endpoint)
- `get_alliance` — input `{allianceId int64}`, output:
  ```go
  type GetAllianceOutput struct {
  	AllianceID            int64      `json:"allianceId"`
  	Name                  string     `json:"name"`
  	Ticker                string     `json:"ticker"`
  	CreatorID             int64      `json:"creatorId"`
  	CreatorCorporationID  int64      `json:"creatorCorporationId"`
  	DateFounded           time.Time  `json:"dateFounded"`
  	ExecutorCorporationID *int64     `json:"executorCorporationId,omitempty"`
  	FactionID             *int64     `json:"factionId,omitempty"`
  }
  ```
- `list_alliance_corporations` — input `{allianceId int64}`, output `{allianceId int64, corporationIds []int64}`
- `get_alliance_icons` — input `{allianceId int64}`, output:
  ```go
  type GetAllianceIconsOutput struct {
  	AllianceID int64   `json:"allianceId"`
  	Px128x128  *string `json:"px128x128,omitempty"`
  	Px64x64    *string `json:"px64x64,omitempty"`
  }
  ```

Every handler follows the existing `getWalletBalanceHandler`-style shape (`func(version, clientID string) mcpsdk.ToolHandlerFor[...]` — note: **no `store *config.Store` parameter**, since there's nothing to check it against), calling an overridable func-var that builds `esi.NewPublicClient(userAgent)` directly (no `sso.AuthenticatedClient` call, no stored refresh token needed, no character required to exist at all).

## Testing

- `esi/alliance_test.go`: real `httptest.Server` + `esi.ExportedNewPublicClientWithOptions`,
  happy path + HTTP-error path per method, matching `esi/wallet_test.go`'s depth.
- `mcp/alliance_internal_test.go`: func-var override pattern, but **no** `keyring.MockInit()` or
  `config.Store` seeding — these tests need neither, since there's no character existence check.
  Simpler setup than every prior tool's tests.
- `mcp/server_test.go`: expected-tools list grows by 4 names (`list_alliances`, `get_alliance`,
  `list_alliance_corporations`, `get_alliance_icons`).

No changes to `sso/` — this batch never touches per-character auth at all.

## Out of scope

- Assets and Calendar remain "Planned" — deferred to their own future batches per the reasoning
  above.
- No SDE-backed enrichment (e.g. resolving `CreatorCorporationId`/`ExecutorCorporationId` to a
  corporation name) — same raw-ID-passthrough convention as every prior batch.
