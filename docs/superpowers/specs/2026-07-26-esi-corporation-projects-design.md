# ESI Corporation Projects (Eleventh Batch) Design

## Status

Approved (autonomous execution). Eleventh batch: the "Corporation Projects" tag alone (4
endpoints, all private/corp-scoped). Small endpoint count, but the tag introduces two genuinely
new patterns to this sweep that make it more involved than its size suggests: **cursor-based
pagination** (the first non-page-based pagination style encountered) and a **17-variant
polymorphic `Configuration` type**. Both were verified against the real installed
`github.com/fnt-eve/goesi-openapi@v0.0.37` source and compile-checked in an isolated sandbox
(including a live `httptest.Server` round-trip proving the cursor query-param/response-body
behavior) before this spec was finalized.

## Scope

All 4 endpoints are **private** (character auth + corp roles, existence-checked via the existing
`characterExists` helper): `GetCorporationsProjectsContribution`, `GetCorporationsProjectsContributors`
(cursor-paginated), `GetCorporationsProjectsDetail`, `GetCorporationsProjectsListing`
(cursor-paginated, plus a `state` filter). New file `esi/corporation_projects.go` /
`mcp/corporation_projects.go` — distinct from `esi/corporation.go`/`mcp/corporation.go`, since
Corporation Projects is its own ESI tag, per the project's one-file-per-tag convention.

No new scopes needed beyond what's already requested — this tag uses corp-role authorization the
same way every other corp-scoped tag does (403 from ESI itself if unauthorized).

## New pattern 1: `projectId` is a string

Unlike every ID handled elsewhere in this project (`corporationId`, `characterId`, etc., all
`int64`), `projectId` is a **string** on all 4 endpoints — confirmed from the generated request
builders and every `CorporationsProjectsDetail*.Id`/`CorporationsProjectsContributorsContributor.Id`
field. Both `esi/` methods and `mcp/` tool inputs use `string` for this field.

## New pattern 2: `characterId` on Contribution is not the caller

`GetCorporationsProjectsContribution(ctx, corporationId, projectId, characterId int64)` takes a
`characterId` that names **whose contribution to look up** — it is not the character whose stored
OAuth token authenticates the call, and need not match it. To avoid confusion with every other
private tool's `characterId` (which always means "whose stored auth token to use"), this parameter
is named `targetCharacterID` in `esi.Client.CorporationProjectContribution` and surfaced as the tool
input field `targetCharacterId`, kept clearly distinct from the tool's own `characterId`.

## New pattern 3: cursor-based pagination

Verified live via `httptest.Server`: `Contributors` and `Listing` take optional `after string`,
`before string` (mutually exclusive — ESI's own contract, not something this project's code
enforces client-side), and `limit int64` query parameters; the response body includes a
`Cursor{After, Before *string}` struct. Confirmed: omitting all three sends zero query params
(ESI applies its own default limit); setting any one sends exactly that param; the response
cursor's `after`/`before` round-trip correctly.

This is a **different mechanism** from the page/`X-Pages` pattern used by every other paginated
`esi/` method in this project (`esi/contracts.go`'s `totalPagesFromHeader`) — it is not shoehorned
into that helper. Convention for this tag: each cursor-paginated `esi/` method takes
`after, before string, limit int64` (empty string / `0` omits the query param) and returns
`(items []T, cursor *esiapi.Cursor, error)`. The `mcp/` tool layer surfaces `after`/`before`/`limit`
as optional inputs and `nextAfter *string`/`nextBefore *string` outputs (from the response cursor's
pointer fields) so a caller can walk forward or backward explicitly — no automatic multi-page
aggregation, consistent with this project's existing pagination philosophy.

`Listing` additionally takes an optional `state string` filter (ESI defaults to `"Active"` when
omitted) — handled the same "empty string omits" way, verified from the generated request builder.

## New pattern 4: the `Configuration` polymorphic type (17 variants)

`CorporationsProjectsDetail.Configuration` is `CorporationsProjectsDetailConfiguration`, a 17-field
oneOf struct (exactly one field non-nil per ESI's own contract): `CaptureFwComplex`, `DamageShip`,
`DefendFwComplex`, `DeliverItem`, `DestroyNpc`, `DestroyShip`, `EarnLoyaltyPoint`, `LostShip`,
`Manual`, `ManufactureItem`, `MineMaterial`, `RemoteBoostShield`, `RemoteRepairArmor`,
`SalvageWreck`, `ScanSignature`, `ShipInsurance`, `Unknown`. Two of the 17 are genuinely free-form
per ESI's own spec — `Manual` (`map[string]interface{}`) and `Unknown` (`{Type string, Data
interface{}}`) — passed through as-is in the MCP layer's output since there is no further typed
structure to map; this is the honest representation, not a shortcut.

The other 15 variants reduce, field-by-field verified against the generated source, to 8 reusable
primitive matcher shapes (each defined once in `mcp/corporation_projects.go` and reused across
every variant that needs it, mirroring — and flattening — the generated package's own type reuse):

- **`LocationMatcher`** (constellation/region/solar-system oneOf) — the *generated package itself*
  reuses one Go type for this across 12 of the 15 typed variants (CaptureFwComplex, DamageShip,
  DefendFwComplex, DestroyNpc, DestroyShip, LostShip, MineMaterial, RemoteBoostShield,
  RemoteRepairArmor, SalvageWreck, ScanSignature, ShipInsurance).
- **`IdentityMatcher`** (alliance/character/corporation/faction oneOf) — DamageShip, DestroyShip,
  LostShip, RemoteBoostShield, RemoteRepairArmor, ShipInsurance.
- **`ShipTypeMatcher`** (group/type oneOf) — appears as **two distinct generated Go types with an
  identical shape**, a pure codegen artifact of the same oneOf appearing under two different parent
  field names: `GroupId`/`TypeId` on "ships" fields (DamageShip, DestroyShip, LostShip,
  RemoteBoostShield, RemoteRepairArmor, ShipInsurance) vs. `GroupId1`/`TypeId` on "items"/"materials"
  fields (DeliverItem, ManufactureItem, MineMaterial) — verified from source, not assumed. Flattened
  to one MCP-layer type since the distinction carries no semantic meaning, but kept as two separate
  `esi/`-facing mapper functions (`toShipTypeMatchersShips`/`toShipTypeMatchersItems`) since the
  underlying Go types genuinely differ.
- **`DockingLocationMatcher`** (station/structure oneOf) — DeliverItem, ManufactureItem.
- **`ArchetypeMatcher`, `FactionMatcher`, `CorporationMatcher`, `SignatureMatcher`** — four
  single-field (no oneOf) matchers: Archetype/Faction used by CaptureFwComplex+DefendFwComplex;
  Corporation by EarnLoyaltyPoint; Signature by ScanSignature.

Variant-specific extra scalar fields: `DeliverItem.OfficeID *int64` (optional), `ManufactureItem.Owner
string` (required), `ShipInsurance.ConflictType string` + `ReimburseImplants bool` (both required).

The flattened MCP output type is `ProjectConfiguration{Kind string, ...}` — `Kind` names the active
variant using ESI's own snake_case discriminator (`"capture_fw_complex"`, `"damage_ship"`, ...,
`"manual"`, `"unknown"`); only the fields relevant to that `Kind` are populated, every other field
left zero. A single `toProjectConfiguration` mapper function does the full 17-way switch.

## `esi/corporation_projects.go` (4 methods)

```go
func (c *Client) CorporationProjectContribution(ctx context.Context, corporationID int64, projectID string, targetCharacterID int64) (*esiapi.CorporationsProjectsContribution, error)
func (c *Client) CorporationProjectContributors(ctx context.Context, corporationID int64, projectID string, after, before string, limit int64) ([]esiapi.CorporationsProjectsContributorsContributor, *esiapi.Cursor, error)
func (c *Client) CorporationProjectDetail(ctx context.Context, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error)
func (c *Client) CorporationProjectListing(ctx context.Context, corporationID int64, after, before string, limit int64, state string) ([]esiapi.CorporationsProjectsDetailProject, *esiapi.Cursor, error)
```

All 4 verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37`,
routing errors through the existing `describeError`. `Contributors` and `Listing` return the
response's inner slice directly (`result.Contributors`/`result.Projects`) plus its `Cursor` — no
new pagination helper redeclared; `totalPagesFromHeader` is deliberately NOT used here since this
tag's pagination mechanism is different.

Verified model shapes: `CorporationsProjectsContribution{Contributed int64, LastModified
*time.Time}`; `CorporationsProjectsContributorsContributor{Contributed, Id int64, Name string}`;
`CorporationsProjectsDetailProject{Id, Name, State string, LastModified time.Time, Progress
CorporationsProjectsDetailProgress, Reward *CorporationsProjectsDetailReward}`;
`CorporationsProjectsDetailContribution{ParticipationLimit, SubmissionLimit *int64,
RewardPerContribution, SubmissionMultiplier *float64}`; `CorporationsProjectsDetailCreator{Id int64,
Name string}`; `CorporationsProjectsDetailDetails{Career, Description string, Created time.Time,
Expires, Finished *time.Time}`; `CorporationsProjectsDetailProgress{Current, Desired int64}`;
`CorporationsProjectsDetailReward{Initial, Remaining float64}`.

## `mcp/corporation_projects.go` (4 tools)

All private (existence-checked via existing `characterExists`):

- `get_corporation_project_contribution` — input `{characterId int32, corporationId int64,
  projectId string, targetCharacterId int64}`, output `{characterId, corporationId, projectId,
  targetCharacterId, contributed int64, lastModified *time.Time}`
- `list_corporation_project_contributors` — input `{characterId int32, corporationId int64,
  projectId string, after, before string, limit int64}`, output `{characterId, corporationId,
  projectId, contributors []ProjectContributor, nextAfter, nextBefore *string}` where
  `ProjectContributor{ContributorID, Contributed int64, Name string}`
- `get_corporation_project_detail` — input `{characterId int32, corporationId int64, projectId
  string}`, output `{characterId, corporationId, projectId, configuration ProjectConfiguration,
  contribution *ProjectContributionLimits, creator ProjectCreator, details ProjectDetails,
  lastModified time.Time, name, state string, progress ProjectProgress, reward *ProjectReward}`
- `list_corporation_projects` — input `{characterId int32, corporationId int64, after, before
  string, limit int64, state string}`, output `{characterId, corporationId, projects
  []ProjectSummary, nextAfter, nextBefore *string}` where `ProjectSummary{ProjectID, Name, State
  string, LastModified time.Time, Progress ProjectProgress, Reward *ProjectReward}`

Shared/reused types: `ProjectProgress{Current, Desired int64}`, `ProjectReward{Initial, Remaining
float64}`, `ProjectContributionLimits{ParticipationLimit, SubmissionLimit *int64,
RewardPerContribution, SubmissionMultiplier *float64}`, `ProjectCreator{CreatorID int64, Name
string}`, `ProjectDetails{Career, Description string, Created time.Time, Expires, Finished
*time.Time}`, plus the 8 matcher types and `ProjectConfiguration` documented above.

## Testing

Same three-layer isolation as every prior batch. `esi/corporation_projects_test.go` covers all 4
methods (parsing + at least one error path), plus dedicated cursor-pagination tests for
`Contributors`/`Listing` proving the omit-when-empty/set-when-present query-param behavior and the
response `Cursor` round-trip (already proven live in the design-verification sandbox; the real
tests re-derive this against the real repo's `esi.ExportedNewPublicClientWithOptions` harness).
`mcp/corporation_projects_internal_test.go` covers happy-path + unknown-character for all 4 tools,
plus a dedicated test asserting `get_corporation_project_detail`'s `toProjectConfiguration` mapper
correctly discriminates at least 3 of the 17 `Configuration` variants (including one of the two
free-form ones, `Manual` or `Unknown`) and correctly reuses each shared matcher type. `mcp/server_test.go`
grows from 79 to 83 tools.

## Out of scope

- No client-side enforcement of ESI's own "after and before are mutually exclusive" contract —
  ESI itself is the authority; this project doesn't duplicate API-level validation it doesn't own.
- No automatic multi-page/cursor aggregation — `nextAfter`/`nextBefore` are surfaced, walking is
  the caller's explicit choice, consistent with every other paginated tool in this project.
