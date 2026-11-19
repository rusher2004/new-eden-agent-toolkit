# ESI Coverage Roadmap + Access List/Activities Design

## Status

Approved. Establishes the durable roadmap and conventions for covering the full ESI API surface,
plus the concrete design for the first batch (Access List + Activities).

## Goal

`esi/` currently covers one resource group (Wallet, from
`docs/superpowers/specs/2026-07-25-esi-wallet-design.md`). This spec extends that pattern to cover
the entire ESI API — 36 resource groups ("tags" in FC's own OpenAPI spec) — as a sequence of
batched plans, and designs the first batch in full.

## Source of truth for the tag order

FC's live OpenAPI 3.0 spec is fetched the same way `goesi-openapi` itself generates its client
(verified by reading `scripts/get_latest_esi_spec.sh` in the vendored
`github.com/fnt-eve/goesi-openapi@v0.0.37` module):

```
GET https://esi.evetech.net/meta/compatibility-dates   -> latest date, e.g. "2026-07-21"
GET https://esi.evetech.net/meta/openapi-3.0.json?compatibility_date=<date>
```

The downloaded spec's `info.version` (`2026-07-21`) matches the installed library's `ESI_VERSION`
file exactly, confirming this is the exact spec our client was generated from — not a community
mirror or a guess. Its top-level `tags` array (36 entries, alphabetical) is the authoritative,
complete order:

| # | Tag | Endpoints | Status |
|---|-----|-----------|--------|
| 1 | Access List | 2 | This plan |
| 2 | Activities | 3 | This plan |
| 3 | Alliance | 4 | Planned |
| 4 | Assets | 6 | Planned |
| 5 | Calendar | 4 | Planned |
| 6 | Character | 14 | Planned |
| 7 | Clones | 2 | Planned |
| 8 | Contacts | 9 | Planned |
| 9 | Contracts | 9 | Planned |
| 10 | Corporation | 22 | Planned |
| 11 | Corporation Projects | 4 | Planned |
| 12 | Dogma | 5 | Planned |
| 13 | Faction Warfare | 8 | Planned |
| 14 | Fittings | 3 | Planned |
| 15 | Fleets | 14 | Planned |
| 16 | Freelance Jobs | 6 | Planned |
| 17 | Incursions | 1 | Planned |
| 18 | Industry | 8 | Planned |
| 19 | Insurance | 1 | Planned |
| 20 | Killmails | 3 | Planned |
| 21 | Location | 3 | Planned |
| 22 | Loyalty | 2 | Planned |
| 23 | Mail | 9 | Planned |
| 24 | Market | 11 | Planned |
| 25 | Meta | 4 | Planned |
| 26 | Planetary Interaction | 4 | Planned |
| 27 | Routes | 1 | Planned |
| 28 | Search | 1 | Planned |
| 29 | Skills | 3 | Planned |
| 30 | Sovereignty | 2 | Planned |
| 31 | Status | 1 | Planned |
| 32 | Structures | 6 | Planned |
| 33 | Universe | 30 | Planned |
| 34 | User Interface | 5 | Planned |
| 35 | Wallet | 6 | **Done** (merged, PR #5) |
| 36 | Wars | 3 | Planned |

219 total endpoints across all 36 tags.

## Conventions established by this plan (apply to every future batch)

- **Tracking doc**: `docs/reference/esi-coverage-roadmap.md` holds the table above verbatim, plus a
  PR-link column. After each batch merges, its rows flip from "Planned"/"This plan" to "Done (PR
  #N)". This is the single place to check "what's left" — no re-deriving the order from the spec
  each time.
- **File layout**: one `esi/<tag_snake_case>.go` + one `mcp/<tag_snake_case>.go` per tag, always —
  even when a plan batches several tags together. Only the plan/PR boundary batches tags; file
  boundaries stay strictly per-resource, matching `esi/wallet.go`'s precedent.
- **Batching**: a plan covers a cluster of adjacent tags (in the table's order) sized so the batch
  isn't trivially small — this plan batches Access List (2 endpoints) and Activities (3 endpoints)
  since neither alone justifies a full spec/plan/PR cycle. Larger tags (Character: 14, Corporation:
  22, Universe: 30) will likely get their own plan, possibly split further in a future spec once
  their internals are known.
- **Raw-ID passthrough**: every `esi/` method returns exactly what ESI returns, including bare IDs
  (`type_id`, `corporation_id`, `dungeon_type_id`, etc.) with zero name-resolution enrichment.
  Calling ESI's own `/universe/names` or similar as a substitute for not-yet-built `sde/` would
  violate the existing "SDE over ESI" hard rule in `CLAUDE.md` — so the rule for now is simply:
  don't resolve, pass the ID through as-is. Enrichment becomes a `sde/`-backed follow-up once that
  package exists.
- **Scopes**: `allScopes` in `mcp/add_character.go` is already deliberately broad ("so any ESI
  endpoint can be exercised while domain tools are being built") and already includes every scope
  this batch needs (`esi-access.read_lists.v1`, `esi-activities.read_character.v1`, both verified
  present in the current list). No new scope requests, no re-auth needed for existing characters.
  Future batches must verify their required scopes are already in this list the same way, and only
  add new ones (with a note that existing characters need `remove_character` + `add_character` to
  pick them up) if a gap is found.

## This batch: Access List + Activities

### `esi/access_list.go`

Wraps `AccessListAPIService`, mirroring `esi/wallet.go`'s shape and error handling exactly
(`describeError(err, resp)` on every `.Execute()` call, per the existing hard rule):

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// AccessLists returns the client's character's structure/fleet access lists.
func (c *Client) AccessLists(ctx context.Context) ([]esiapi.CharactersAccessListsListingAccesslist, error) {
	result, resp, err := c.api.AccessListAPI.GetCharactersAccessListsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get access lists: %w", describeError(err, resp))
	}
	return result.AccessLists, nil
}

// AccessListDetail returns the full detail (description, membership) for one access list.
func (c *Client) AccessListDetail(ctx context.Context, accessListID int64) (*esiapi.CharactersAccessListsDetail, error) {
	result, resp, err := c.api.AccessListAPI.GetCharactersAccessListsDetail(ctx, accessListID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get access list detail: %w", describeError(err, resp))
	}
	return result, nil
}
```

Verified generated signatures (from `github.com/fnt-eve/goesi-openapi@v0.0.37/esi/api_access_list.go`):
- `GetCharactersAccessListsListing(ctx context.Context, characterId int64) ApiGetCharactersAccessListsListingRequest`, `.Execute() (*CharactersAccessListsListing, *http.Response, error)`
- `GetCharactersAccessListsDetail(ctx context.Context, accessListId int64, characterId int64) ApiGetCharactersAccessListsDetailRequest`, `.Execute() (*CharactersAccessListsDetail, *http.Response, error)`

Response models (verified from `model_characters_access_lists_listing.go` /
`model_characters_access_lists_detail.go`):
```go
type CharactersAccessListsListing struct {
	AccessLists []CharactersAccessListsListingAccesslist `json:"access_lists"`
}
type CharactersAccessListsDetail struct {
	Description string                                `json:"description"`
	Id          int64                                 `json:"id"`
	Membership  CharactersAccessListsDetailMembership `json:"membership"`
	Name        string                                `json:"name"`
}
```

### `esi/activities.go`

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// MercenaryTacticalOperations returns the client's character's mercenary tactical operations.
func (c *Client) MercenaryTacticalOperations(ctx context.Context) ([]esiapi.CharactersMercenaryTacticalOperationsListingOperation, error) {
	result, resp, err := c.api.ActivitiesAPI.GetCharactersMercenaryTacticalOperationsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get mercenary tactical operations: %w", describeError(err, resp))
	}
	return result.Operations, nil
}

// MercenaryTacticalOperationDetail returns detail for one mercenary tactical operation.
// operationID is a string in ESI's own model, not neat's usual int32/int64 ID convention.
func (c *Client) MercenaryTacticalOperationDetail(ctx context.Context, operationID string) (*esiapi.CharactersMercenaryTacticalOperationsDetail, error) {
	result, resp, err := c.api.ActivitiesAPI.GetCharactersMercenaryTacticalOperationsDetail(ctx, operationID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get mercenary tactical operation detail: %w", describeError(err, resp))
	}
	return result, nil
}

// RaidableSkyhooks returns upcoming raidable skyhooks. Unauthenticated on ESI's side (no scope
// required), but still called through the per-character authenticated client for interface
// consistency with every other esi.Client method.
func (c *Client) RaidableSkyhooks(ctx context.Context) ([]esiapi.SkyhooksRaidableVulnerableskyhook, error) {
	result, resp, err := c.api.ActivitiesAPI.GetSkyhooksRaidable(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get raidable skyhooks: %w", describeError(err, resp))
	}
	return result.Skyhooks, nil
}
```

Verified generated signatures (from `.../esi/api_activities.go`):
- `GetCharactersMercenaryTacticalOperationsListing(ctx context.Context, characterId int64) ...`, `.Execute() (*CharactersMercenaryTacticalOperationsListing, *http.Response, error)`
- `GetCharactersMercenaryTacticalOperationsDetail(ctx context.Context, operationId string, characterId int64) ...`, `.Execute() (*CharactersMercenaryTacticalOperationsDetail, *http.Response, error)`
- `GetSkyhooksRaidable(ctx context.Context) ...`, `.Execute() (*SkyhooksRaidable, *http.Response, error)` — no character ID parameter at all.

Response models (verified):
```go
type CharactersMercenaryTacticalOperationsDetail struct {
	DungeonTypeId  int64     `json:"dungeon_type_id"`
	Expires        time.Time `json:"expires"`
	Id             string    `json:"id"`
	MercenaryDenId int64     `json:"mercenary_den_id"`
	State          string    `json:"state"`
}
```

### MCP tools

Following the `get_wallet_balance` precedent exactly: every handler loads `config.Store`, checks
the character exists, returns `ErrCharacterNotFound` if not, then calls the `esi/` method via an
overridable func-var.

`mcp/access_list.go`:
- `list_access_lists` — input `{characterId int32}`, output `{characterId int32, accessListIds []int64}`. Verified: the listing model (`CharactersAccessListsListingAccesslist`) has only one field, `Id int64` — no name or description at this level, so the output is bare IDs; a name requires calling `get_access_list` per ID.
- `get_access_list` — input `{characterId int32, accessListId int64}`, output:
  ```go
  type GetAccessListOutput struct {
  	CharacterID   int32                `json:"characterId"`
  	AccessListID  int64                `json:"accessListId"`
  	Name          string               `json:"name"`
  	Description   string               `json:"description"`
  	AllowEveryone bool                 `json:"allowEveryone"`
  	Characters    []AccessListEntry    `json:"characters"`    // {EntityID int64 `json:"entityId"`, Access string `json:"access"`}, from CharacterId
  	Corporations  []AccessListEntry    `json:"corporations"`  // from CorporationId
  	Alliances     []AccessListEntry    `json:"alliances"`     // from AllianceId
  }
  type AccessListEntry struct {
  	EntityID int64  `json:"entityId"`
  	Access   string `json:"access"`
  }
  ```
  (`CharactersAccessListsDetailMembership` verified to have `Alliances []...Allianceentry{Access, AllianceId}`, `Characters []...Characterentry{Access, CharacterId}`, `Corporations []...Corporationentry{Access, CorporationId}`, `AllowEveryone bool` — all three entry sub-types share the same `{Access string, <X>Id int64}` shape, so `AccessListEntry` is one shared struct fed from each.)

`mcp/activities.go`:
- `list_mercenary_tactical_operations` — input `{characterId int32}`, output `{characterId int32, operations []OperationSummary}` where `OperationSummary{Id string, MercenaryDenId int64}` (verified: `CharactersMercenaryTacticalOperationsListingOperation` has exactly these two fields, both JSON tags `id`/`mercenary_den_id`)
- `get_mercenary_tactical_operation` — input `{characterId int32, operationId string}`, output `{characterId int32, operationId string, dungeonTypeId int64, mercenaryDenId int64, state string, expires time.Time}`
- `list_raidable_skyhooks` — input `{characterId int32}` (kept for the existence check per convention, even though the underlying ESI call needs no character scope), output `{skyhooks []SkyhookSummary}` where `SkyhookSummary{PlanetID int64, SolarSystemID int64, TheftVulnerabilityStart time.Time, TheftVulnerabilityEnd time.Time}` (verified: `SkyhooksRaidableVulnerableskyhook{PlanetId, SolarSystemId, TheftVulnerability{Start, End}}`)

`mcp/server.go`'s `TestServer_AllToolsRegistered` expected-tools list grows to include all 5 new
names.

## Testing

Same three-layer isolation as the wallet plan:
- `esi/access_list_test.go`, `esi/activities_test.go`: real `httptest.Server` + real `goesi` client
  construction, happy path + an error path asserting `describeError` wrapping (matching
  `esi/wallet_test.go`'s pattern).
- `mcp/access_list_internal_test.go`, `mcp/activities_internal_test.go`: func-var override,
  seeded `config.Store`, unknown-character case asserting `ErrCharacterNotFound` from day one.
- `mcp/server_test.go`: expected-tools list updated.

No changes to `sso/` — `AuthenticatedClient`/`RemoveCharacter` are already resource-agnostic.

## Out of scope

- The remaining 33 tags — each gets its own future batch/plan following this same roadmap doc and
  conventions, not designed here.
- Any SDE-backed name enrichment (e.g. resolving `dungeon_type_id` to a name) — deferred until
  `sde/` exists, per the raw-ID passthrough convention above.
