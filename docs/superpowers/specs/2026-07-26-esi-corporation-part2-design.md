# ESI Corporation, Part 2 (Tenth Batch) Design

## Status

Approved (autonomous execution). Part 2 completes the Corporation tag: the remaining 8 of 22
endpoints (assets/infrastructure — blueprints, container logs, facilities, medals, medals/issued,
starbases ×2, structures), all deferred from Part 1 (PR #6, merged into this branch) purely for
file-size manageability. Since this is the same ESI tag as Part 1, per the project's "one file per
tag" convention (`CLAUDE.md`), these 8 methods/tools are appended directly to the existing
`esi/corporation.go` and `mcp/corporation.go` — no new files. Once this batch lands, the roadmap
doc's Corporation row flips from "Planned" to "This plan" (both parts now done).

## Scope

All 8 endpoints are **private** (corp-scoped, authenticated client + the existing `characterExists`
helper) — unlike Part 1, there are no public endpoints in this half.

`GetCorporationsCorporationIdBlueprints` (paginated), `GetCorporationsCorporationIdContainersLogs`
(paginated), `GetCorporationsCorporationIdFacilities` (not paginated), `GetCorporationsCorporationIdMedals`
(paginated), `GetCorporationsCorporationIdMedalsIssued` (paginated), `GetCorporationsCorporationIdStarbases`
(paginated), `GetCorporationsCorporationIdStarbasesStarbaseId` (not paginated — single starbase
detail, requires a `systemId` query parameter that ESI needs to resolve the request — see below),
`GetCorporationsCorporationIdStructures` (paginated).

**New parameter pattern — required `systemId` on `get_corporation_starbase`**: verified directly
from the generated request builder (`ApiGetCorporationsCorporationIdStarbasesStarbaseIdRequest`),
this endpoint's generated struct has a `systemId *int64` field with a `.SystemId(int64)` builder
method. Although the generated Go signature makes it optional (a builder pattern, not a required
positional arg), ESI's real API requires `system_id` to resolve which solar system's starbase to
look up — the `esi/` facade method therefore takes `systemID int64` as a required parameter (not
`page`-style `0`-means-omit), and the MCP tool surfaces it as a required input field, matching how
every other genuinely-required ESI query parameter is already exposed in this project.

All scopes already present in `allScopes` (`mcp/add_character.go`) from Part 1 — no new scope
additions needed: `esi-corporations.read_blueprints.v1`, `read_container_logs.v1`,
`read_facilities.v1`, `read_medals.v1`, `read_starbases.v1`, `read_structures.v1`.

## `esi/corporation.go` (append 8 methods)

```go
func (c *Client) CorporationBlueprints(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error)
func (c *Client) CorporationContainerLogs(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error)
func (c *Client) CorporationFacilities(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error)
func (c *Client) CorporationMedals(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsGetInner, int32, error)
func (c *Client) CorporationMedalsIssued(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsIssuedGetInner, int32, error)
func (c *Client) CorporationStarbases(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error)
func (c *Client) CorporationStarbaseDetail(ctx context.Context, corporationID int64, starbaseID int64, systemID int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error)
func (c *Client) CorporationStructures(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error)
```

All 8 verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37`.
Same conventions as Part 1: paginated methods take `page int32` (`0` omits `.Page()`), route
errors through the existing `describeError`, use the existing `totalPagesFromHeader` for pagination
— neither helper is redeclared, both already live in `esi/error.go`/`esi/contracts.go`.
`CorporationStarbaseDetail` calls `.SystemId(systemID)` unconditionally (it's always required, no
`0`-means-omit branching like `page`).

Verified model shapes: `CorporationsCorporationIdBlueprintsGetInner` (item_id, location_flag,
location_id, material_efficiency, quantity, runs, time_efficiency, type_id — all required, no
pointers); `CorporationsCorporationIdContainersLogsGetInner` (action, character_id, container_id,
container_type_id, location_flag, location_id, logged_at required; new_config_bitmask,
old_config_bitmask, password_type, quantity, type_id all optional/pointer);
`CorporationsCorporationIdFacilitiesGetInner` (facility_id, system_id, type_id — all required);
`CorporationsCorporationIdMedalsGetInner` (created_at, creator_id, description, medal_id, title —
all required); `CorporationsCorporationIdMedalsIssuedGetInner` (character_id, issued_at, issuer_id,
medal_id, reason, status — all required); `CorporationsCorporationIdStarbasesGetInner` (starbase_id,
system_id, type_id required; moon_id, onlined_since, reinforced_until, state, unanchor_at all
optional/pointer); `CorporationsCorporationIdStarbasesStarbaseIdGet` (allow_alliance_members through
use_alliance_standings, mostly required booleans/strings; attack_security_status_threshold,
attack_standing_threshold optional floats; `fuels []CorporationsCorporationIdStarbasesStarbaseIdGetFuelsInner`
optional slice, each entry `{quantity, type_id int64}` both required);
`CorporationsCorporationIdStructuresGetInner` (corporation_id, profile_id, state, structure_id,
system_id, type_id required; fuel_expires, name, next_reinforce_apply, next_reinforce_hour,
reinforce_hour, state_timer_end, state_timer_start, unanchors_at all optional/pointer;
`services []CorporationsCorporationIdStructuresGetInnerServicesInner` optional slice, each entry
`{name, state string}` both required).

## `mcp/corporation.go` (append 8 tools)

All 8 are private (existence-checked via existing `characterExists`) — no public tools in this half.

- `list_corporation_blueprints` — input `{characterId int32, corporationId int64, page int32}`, output `{characterId, corporationId, page, totalPages int32, blueprints []CorporationBlueprint}` where `CorporationBlueprint{ItemID, LocationID int64, LocationFlag string, MaterialEfficiency, Quantity, Runs, TimeEfficiency, TypeID int64}`
- `list_corporation_container_logs` — input adds `page`, output `{..., logs []ContainerLog}` where `ContainerLog{Action string, CharacterID, ContainerID, ContainerTypeID int64, LocationFlag string, LocationID int64, LoggedAt time.Time, NewConfigBitmask, OldConfigBitmask, Quantity, TypeID *int64, PasswordType *string}`
- `list_corporation_facilities` — input `{characterId, corporationId}` (no page — unpaginated), output `{characterId, corporationId, facilities []Facility}` where `Facility{FacilityID, SystemID, TypeID int64}`
- `list_corporation_medals` — input adds `page`, output `{..., medals []CorporationMedal}` where `CorporationMedal{CreatedAt time.Time, CreatorID, MedalID int64, Description, Title string}`
- `list_corporation_medals_issued` — input adds `page`, output `{..., issued []MedalIssued}` where `MedalIssued{CharacterID, IssuerID, MedalID int64, IssuedAt time.Time, Reason, Status string}`
- `list_corporation_starbases` — input adds `page`, output `{..., starbases []Starbase}` where `Starbase{StarbaseID, SystemID, TypeID int64, MoonID *int64, OnlinedSince, ReinforcedUntil, UnanchorAt *time.Time, State *string}`
- `get_corporation_starbase` — input `{characterId, corporationId, starbaseId, systemId int64}` (systemId is required, not optional/paginated), output `{characterId, corporationId, starbaseId int64, detail StarbaseDetail}` where `StarbaseDetail{AllowAllianceMembers, AllowCorporationMembers, AttackIfAtWar, AttackIfOtherSecurityStatusDropping, UseAllianceStandings bool, Anchor, FuelBayTake, FuelBayView, Offline, Online, Unanchor string, AttackSecurityStatusThreshold, AttackStandingThreshold *float64, Fuels []StarbaseFuel}` and `StarbaseFuel{Quantity, TypeID int64}`
- `list_corporation_structures` — input adds `page`, output `{..., structures []Structure}` where `Structure{CorporationID, ProfileID, StructureID, SystemID, TypeID int64, State string, FuelExpires, NextReinforceApply, StateTimerEnd, StateTimerStart, UnanchorsAt *time.Time, Name *string, NextReinforceHour, ReinforceHour *int64, Services []StructureService}` and `StructureService{Name, State string}`

All 5 paginated tools (`list_corporation_blueprints`, `list_corporation_container_logs`,
`list_corporation_medals`, `list_corporation_medals_issued`, `list_corporation_starbases`,
`list_corporation_structures` — 6 actually, all but `list_corporation_facilities` and
`get_corporation_starbase`) default `page` to 1 in the output when input page is 0, matching the
established convention.

## Testing

Same three-layer isolation as every prior batch. `esi/corporation_test.go` grows with 8 more test
functions (parsing + at least one pagination-omission/total-pages check for the 6 paginated methods,
plus a check that `CorporationStarbaseDetail` always sends `system_id` in the query string since
it's unconditional, not `page`-style optional). `mcp/corporation_internal_test.go` grows with 8
happy-path tests + 8 unknown-character tests (16 new test functions) — all 8 tools are private.
`mcp/server_test.go` grows from 71 to 79 tools.

## Docs sync

`CLAUDE.md`/`README.md` tool count "seventy-one" → "seventy-nine", append the 8 new tool names.
`docs/reference/esi-coverage-roadmap.md`'s Corporation row flips from "Planned" to "This plan" —
both parts now land in this PR.
