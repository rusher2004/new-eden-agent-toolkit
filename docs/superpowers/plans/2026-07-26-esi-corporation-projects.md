# ESI Corporation Projects (Eleventh Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 4 endpoints of the "Corporation Projects" ESI tag as a new `esi/corporation_projects.go` facade and a new `mcp/corporation_projects.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `CorporationProjectsAPI`, reusing the existing `describeError` helper (`esi/error.go`). One new `mcp/` tool file with 4 private tools (all corp-scoped, existence-checked via the existing `characterExists` helper from `mcp/character.go`). This tag introduces two new patterns not seen elsewhere in this project: cursor-based pagination (not the page/`X-Pages` style used everywhere else) and a 17-variant polymorphic `Configuration` type, flattened into 8 reusable matcher types plus a discriminated `ProjectConfiguration` output type.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `e08450d`) — no new branch, no new PR.
- Current tool count baseline: **79**. After this plan: **83**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`) — do not redeclare it.
- **`projectId` is a `string`**, not `int64` like every other ID in this project — do not "fix" this to int64, it is correct as specified.
- **`targetCharacterId`** (on `get_corporation_project_contribution` only) is a distinct field from the tool's own `characterId` (auth) — never conflate the two, never rename `targetCharacterId` to `characterId`.
- **Cursor pagination is NOT the page/`X-Pages` pattern.** Do not use or extend `totalPagesFromHeader` (`esi/contracts.go`) for this tag. Cursor-paginated `esi/` methods take `after, before string, limit int64` (empty string / `0` omits the query param) and return `(items []T, cursor *esiapi.Cursor, error)`. The `mcp/` layer surfaces `after`/`before`/`limit` as optional inputs and `nextAfter *string`/`nextBefore *string` outputs from the response cursor.
- Every private tool uses `esi.NewClient` + the existing `characterExists(store *config.Store, characterID int32) (bool, error)` helper and `ErrCharacterNotFound` sentinel (both `mcp/character.go`) — reused, not redeclared. On not-found: `fmt.Errorf("<action> for character %d: %w", characterID, ErrCharacterNotFound)`.
- The `Configuration` field's 8 matcher types (`LocationMatcher`, `IdentityMatcher`, `ShipTypeMatcher`, `DockingLocationMatcher`, `ArchetypeMatcher`, `FactionMatcher`, `CorporationMatcher`, `SignatureMatcher`) and the `ProjectConfiguration` discriminated type are each defined exactly once in `mcp/corporation_projects.go` and reused across every relevant variant in `toProjectConfiguration`'s switch — do not redeclare per-variant.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean), including a live `httptest.Server` test proving the cursor query-param omission/round-trip behavior. Implementers should still run these themselves as part of each task.

---

### Task 1: `esi/corporation_projects.go` facade (4 methods)

**Files:**
- Create: `esi/corporation_projects.go`
- Test: `esi/corporation_projects_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `Client.api` (`esi/client.go`, `*esiapi.APIClient`).
- Produces: 4 methods on `*Client` for Task 2's `mcp/corporation_projects.go` to call.

- [ ] **Step 1: Write `esi/corporation_projects.go`**

```go
// Package esi's corporation_projects.go covers all 4 of "Corporation Projects"'s endpoints:
// contribution, contributors (cursor-paginated), detail, and listing (cursor-paginated). All four
// are private/corp-scoped -- the authenticated character must hold sufficient corp roles, ESI
// itself returns 403 otherwise.
//
// Two things make this tag different from every other esi/ file in this project so far:
//
//  1. projectId is a STRING, not an int64 like every other ID handled elsewhere in this project.
//     Verified from the generated request builders (ApiGetCorporationsProjectsDetailRequest etc.)
//     and every CorporationsProjectsDetail* model's Id field.
//
//  2. GetCorporationsProjectsContribution takes a characterId path/query parameter that is NOT
//     the calling/authenticated character -- Client.characterID is baked into the SSO-backed
//     http.Client and never appears in this call at all. ESI's characterId here names whose
//     contribution to look up, which need not be the caller. This method names that parameter
//     targetCharacterID to keep the two unambiguous.
//
// Contributors and Listing use ESI's cursor-based pagination (after/before/limit query params, a
// Cursor{After, Before *string} in the response body) instead of the page/X-Pages style used
// everywhere else in this project (see esi/contracts.go's totalPagesFromHeader) -- this is a
// genuinely different pagination mechanism, not shoehorned into the old helper. For both methods:
// after="" and before="" each omit their own query param entirely (the two are mutually exclusive
// per ESI's own docs -- callers pick one direction); limit=0 omits the limit query param, letting
// ESI apply its own default. Listing additionally takes state="" to omit the state filter, letting
// ESI apply its own default ("Active").
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CorporationProjectContribution returns targetCharacterID's contribution to corporationID's
// project projectID. targetCharacterID is deliberately distinct from the Client's own
// characterID (used only to authenticate the call) -- it's the character whose contribution ESI
// should report, which need not be the calling character.
func (c *Client) CorporationProjectContribution(ctx context.Context, corporationID int64, projectID string, targetCharacterID int64) (*esiapi.CorporationsProjectsContribution, error) {
	result, resp, err := c.api.CorporationProjectsAPI.GetCorporationsProjectsContribution(ctx, corporationID, projectID, targetCharacterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation project contribution: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationProjectContributors returns the contributors to corporationID's project projectID,
// cursor-paginated. after="" and before="" each omit their own query param (mutually exclusive --
// ESI errors if both are set); limit=0 omits the limit param and lets ESI apply its default. The
// returned *esiapi.Cursor's After/Before pointers (when non-nil) are the cursor values to pass
// back in as this method's own after/before on the next call, to walk forward/backward.
func (c *Client) CorporationProjectContributors(ctx context.Context, corporationID int64, projectID string, after, before string, limit int64) ([]esiapi.CorporationsProjectsContributorsContributor, *esiapi.Cursor, error) {
	req := c.api.CorporationProjectsAPI.GetCorporationsProjectsContributors(ctx, corporationID, projectID)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get corporation project contributors: %w", describeError(err, resp))
	}
	return result.Contributors, result.Cursor, nil
}

// CorporationProjectDetail returns the full details of corporationID's project projectID,
// including its polymorphic Configuration (17 variants -- see mcp/corporation_projects.go's
// ProjectConfiguration and its toProjectConfiguration mapper for the flattened shape and the
// full variant-by-variant field mapping).
func (c *Client) CorporationProjectDetail(ctx context.Context, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
	result, resp, err := c.api.CorporationProjectsAPI.GetCorporationsProjectsDetail(ctx, corporationID, projectID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation project detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationProjectListing returns corporationID's projects, cursor-paginated and optionally
// filtered by state. after="" and before="" each omit their own query param (mutually exclusive);
// limit=0 omits the limit param; state="" omits the state param and lets ESI apply its own
// default ("Active").
func (c *Client) CorporationProjectListing(ctx context.Context, corporationID int64, after, before string, limit int64, state string) ([]esiapi.CorporationsProjectsDetailProject, *esiapi.Cursor, error) {
	req := c.api.CorporationProjectsAPI.GetCorporationsProjectsListing(ctx, corporationID)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	if state != "" {
		req = req.State(state)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get corporation project listing: %w", describeError(err, resp))
	}
	return result.Projects, result.Cursor, nil
}
```

- [ ] **Step 2: Write `esi/corporation_projects_test.go`**

Read `esi/contracts_test.go` first to copy its exact `httptest.Server` + `ExportedNewPublicClientWithOptions` setup style. Cover:

- `TestCorporationProjectContribution_ParsesResponse` — serve `{contributed, last_modified}`, assert both fields map (including the `*time.Time` pointer being non-nil and correct).
- `TestCorporationProjectContribution_HTTPError` — serve a 403, assert the error is non-nil and wraps a description.
- `TestCorporationProjectContributors_ParsesResponse` — serve `{contributors: [{contributed, id, name}], cursor: {after, before}}`, assert every contributor field maps and both `Cursor.After`/`.Before` are non-nil and correct.
- `TestCorporationProjectContributors_CursorOmittedWhenEmpty` — call with `after="", before="", limit=0`, assert the request the test server received has none of `after`/`before`/`limit` in its query string.
- `TestCorporationProjectContributors_CursorSentWhenSet` — call with `after="abc123"`, assert the query string contains exactly `after=abc123` and no `before`/`limit`.
- `TestCorporationProjectDetail_ParsesResponse` — serve a full `CorporationsProjectsDetail` with a `configuration` set to ONE variant (e.g. `capture_fw_complex` with a populated `archetypes`/`factions`/`locations`), assert `Configuration.CaptureFwComplex.CaptureFwComplex` (the double-wrapped shape) is non-nil and its nested fields map, plus assert every top-level field (`Contribution`, `Creator`, `Details`, `Id`, `LastModified`, `Name`, `Progress`, `Reward`, `State`) maps correctly.
- `TestCorporationProjectListing_ParsesResponse` — serve `{projects: [{id, last_modified, name, progress: {current, desired}, reward: {initial, remaining}, state}], cursor: {after, before}}`, assert every field maps.
- `TestCorporationProjectListing_CursorAndStateOmittedWhenEmpty` — call with all filter params empty/zero, assert no `after`/`before`/`limit`/`state` in the query string.
- `TestCorporationProjectListing_CursorAndStateSentWhenSet` — call with `before="xyz"`, `limit=5`, `state="Completed"`, assert exactly those 3 query params are sent and `after` is absent.

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run CorporationProject -v`
Expected: all tests PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/corporation_projects.go esi/corporation_projects_test.go
git commit -m "feat: add esi/corporation_projects.go facade (cursor pagination)"
```

---

### Task 2: `mcp/corporation_projects.go` tools (4 tools), server registration, and docs sync

**Files:**
- Create: `mcp/corporation_projects.go`
- Test: `mcp/corporation_projects_internal_test.go`
- Modify: `mcp/server.go` (register 4 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 79 to 83)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Corporation Projects row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "seventy-nine" → "eighty-three")
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: all 4 `esi/corporation_projects.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent`.
- Produces: 4 `mcpsdk.ToolHandlerFor[...]`-returning functions for `mcp/server.go`: `getCorporationProjectContributionHandler`, `listCorporationProjectContributorsHandler`, `getCorporationProjectDetailHandler`, `listCorporationProjectsHandler` — all `(version, clientID string, store *config.Store)`.

- [ ] **Step 1: Write `mcp/corporation_projects.go`**

```go
// Package mcp's corporation_projects.go covers all 4 endpoints of the "Corporation Projects" ESI
// tag: get_corporation_project_contribution, list_corporation_project_contributors,
// get_corporation_project_detail, and list_corporation_projects. All four are private/corp-scoped
// (need a stored character's auth token) and live in their own file per this project's
// one-file-per-tag convention -- distinct from mcp/corporation.go.
//
// Naming note: get_corporation_project_contribution's input has both CharacterID (int32 -- whose
// stored auth token to use, same as every other private tool in this project) and
// TargetCharacterID (int64 -- whose contribution to look up, an ESI query parameter that need not
// match CharacterID). The two are deliberately named differently to avoid confusion; see
// esi/corporation_projects.go's package doc for the full rationale.
//
// Pagination note: list_corporation_project_contributors and list_corporation_projects use ESI's
// cursor-based pagination (after/before/limit, mutually-exclusive after/before) instead of the
// page-number style used by every other paginated tool in this project (see mcp/contracts.go).
// Input fields After/Before/Limit are optional (zero value omits the query param); output fields
// NextAfter/NextBefore surface the response Cursor's After/Before pointers so a caller can walk
// forward or backward.
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- shared Configuration matcher types ---
//
// ESI's CorporationsProjectsDetailConfiguration is a 17-variant oneOf. Two variants (Manual,
// Unknown) are themselves free-form per ESI's own spec -- represented below as generic JSON-ish
// values since there is no further typed structure to map. The other 15 variants are built from a
// small set of reusable primitive matcher shapes; each is defined once here and reused by every
// variant that needs it, mirroring (and flattening) the generated package's own reuse of e.g.
// CorporationsProjectsDetailConfigurationcapturefwcomplexLocationsInner across CaptureFwComplex,
// DamageShip, DefendFwComplex, DestroyNpc, DestroyShip, LostShip, MineMaterial, RemoteBoostShield,
// RemoteRepairArmor, SalvageWreck, ScanSignature, and ShipInsurance.

// LocationMatcher is a oneOf{constellation, region, solar system} location filter -- at most one
// field is set, mirroring ESI's own oneOf. Shared by 12 of the 17 Configuration variants.
type LocationMatcher struct {
	ConstellationID *int64 `json:"constellationId,omitempty"`
	RegionID        *int64 `json:"regionId,omitempty"`
	SolarSystemID   *int64 `json:"solarSystemId,omitempty"`
}

// IdentityMatcher is a oneOf{alliance, character, corporation, faction} identity filter. Shared by
// every Configuration variant with an "identities" field (DamageShip, DestroyShip, LostShip,
// RemoteBoostShield, RemoteRepairArmor, ShipInsurance).
type IdentityMatcher struct {
	AllianceID    *int64 `json:"allianceId,omitempty"`
	CharacterID   *int64 `json:"characterId,omitempty"`
	CorporationID *int64 `json:"corporationId,omitempty"`
	FactionID     *int64 `json:"factionId,omitempty"`
}

// ShipTypeMatcher is a oneOf{ship-tree group, type} filter. ESI's generated package produces two
// distinct Go types for this identical shape purely as a codegen artifact of the same oneOf
// appearing in two different parent contexts: GroupId on "ships" fields (DamageShip, DestroyShip,
// LostShip, RemoteBoostShield, RemoteRepairArmor, ShipInsurance) vs. GroupId1 on "items"/
// "materials" fields (DeliverItem, ManufactureItem, MineMaterial) -- verified from source, not
// assumed. This MCP layer flattens both to the same ShipTypeMatcher shape since the distinction
// carries no semantic meaning.
type ShipTypeMatcher struct {
	GroupID *int64 `json:"groupId,omitempty"`
	TypeID  *int64 `json:"typeId,omitempty"`
}

// DockingLocationMatcher is a oneOf{station, structure} docking-location filter. Shared by
// DeliverItem and ManufactureItem.
type DockingLocationMatcher struct {
	StationID   *int64 `json:"stationId,omitempty"`
	StructureID *int64 `json:"structureId,omitempty"`
}

// ArchetypeMatcher, FactionMatcher, CorporationMatcher, and SignatureMatcher are single-field
// matchers -- no oneOf, just one optional ID each -- reused across the variants below.
type ArchetypeMatcher struct {
	ArchetypeID *int64 `json:"archetypeId,omitempty"`
}
type FactionMatcher struct {
	FactionID *int64 `json:"factionId,omitempty"`
}
type CorporationMatcher struct {
	CorporationID *int64 `json:"corporationId,omitempty"`
}
type SignatureMatcher struct {
	SignatureTypeID *int64 `json:"signatureTypeId,omitempty"`
}

// ProjectConfiguration is the flattened form of CorporationsProjectsDetailConfiguration, ESI's
// 17-variant oneOf. Kind names the active variant using ESI's own snake_case discriminator (e.g.
// "capture_fw_complex", "damage_ship", ..., "manual", "unknown"); only the fields relevant to that
// Kind are populated below, every other field left zero. See toProjectConfiguration for the full
// variant-by-variant mapping.
type ProjectConfiguration struct {
	Kind string `json:"kind"`

	// capture_fw_complex, defend_fw_complex
	Archetypes []ArchetypeMatcher `json:"archetypes,omitempty"`
	Factions   []FactionMatcher   `json:"factions,omitempty"`

	// capture_fw_complex, damage_ship, defend_fw_complex, destroy_npc, destroy_ship, lost_ship,
	// mine_material, remote_boost_shield, remote_repair_armor, salvage_wreck, scan_signature,
	// ship_insurance
	Locations []LocationMatcher `json:"locations,omitempty"`

	// damage_ship, destroy_ship, lost_ship, remote_boost_shield, remote_repair_armor,
	// ship_insurance
	Identities []IdentityMatcher `json:"identities,omitempty"`
	Ships      []ShipTypeMatcher `json:"ships,omitempty"`

	// deliver_item, manufacture_item
	DockingLocations []DockingLocationMatcher `json:"dockingLocations,omitempty"`
	Items            []ShipTypeMatcher        `json:"items,omitempty"`
	OfficeID         *int64                   `json:"officeId,omitempty"`
	Owner            string                   `json:"owner,omitempty"`

	// mine_material
	Materials []ShipTypeMatcher `json:"materials,omitempty"`

	// earn_loyalty_point
	Corporations []CorporationMatcher `json:"corporations,omitempty"`

	// scan_signature
	Signatures []SignatureMatcher `json:"signatures,omitempty"`

	// ship_insurance
	ConflictType      string `json:"conflictType,omitempty"`
	ReimburseImplants bool   `json:"reimburseImplants,omitempty"`

	// manual -- free-form per ESI's own spec, no further typed structure to map
	Manual map[string]interface{} `json:"manual,omitempty"`

	// unknown -- free-form per ESI's own spec
	UnknownType string      `json:"unknownType,omitempty"`
	UnknownData interface{} `json:"unknownData,omitempty"`
}

func toLocationMatcher(v esiapi.CorporationsProjectsDetailConfigurationcapturefwcomplexLocationsInner) LocationMatcher {
	m := LocationMatcher{}
	if v.ConstellationId != nil {
		m.ConstellationID = v.ConstellationId.ConstellationId
	}
	if v.RegionId != nil {
		m.RegionID = v.RegionId.RegionId
	}
	if v.SolarSystemId != nil {
		m.SolarSystemID = v.SolarSystemId.SolarSystemId
	}
	return m
}

func toLocationMatchers(vs []esiapi.CorporationsProjectsDetailConfigurationcapturefwcomplexLocationsInner) []LocationMatcher {
	out := make([]LocationMatcher, len(vs))
	for i, v := range vs {
		out[i] = toLocationMatcher(v)
	}
	return out
}

func toIdentityMatcher(v esiapi.CorporationsProjectsDetailConfigurationdamageshipIdentitiesInner) IdentityMatcher {
	m := IdentityMatcher{}
	if v.AllianceId != nil {
		m.AllianceID = v.AllianceId.AllianceId
	}
	if v.CharacterId != nil {
		m.CharacterID = v.CharacterId.CharacterId
	}
	if v.CorporationId != nil {
		m.CorporationID = v.CorporationId.CorporationId
	}
	if v.FactionId != nil {
		m.FactionID = v.FactionId.FactionId
	}
	return m
}

func toIdentityMatchers(vs []esiapi.CorporationsProjectsDetailConfigurationdamageshipIdentitiesInner) []IdentityMatcher {
	out := make([]IdentityMatcher, len(vs))
	for i, v := range vs {
		out[i] = toIdentityMatcher(v)
	}
	return out
}

// toShipTypeMatchersShips maps the "ships"-field oneOf (GroupId/TypeId) shared by DamageShip,
// DestroyShip, LostShip, RemoteBoostShield, RemoteRepairArmor, and ShipInsurance.
func toShipTypeMatchersShips(vs []esiapi.CorporationsProjectsDetailConfigurationdamageshipShipsInner) []ShipTypeMatcher {
	out := make([]ShipTypeMatcher, len(vs))
	for i, v := range vs {
		m := ShipTypeMatcher{}
		if v.GroupId != nil {
			m.GroupID = v.GroupId.GroupId
		}
		if v.TypeId != nil {
			m.TypeID = v.TypeId.TypeId
		}
		out[i] = m
	}
	return out
}

// toShipTypeMatchersItems maps the "items"/"materials"-field oneOf (GroupId1/TypeId) shared by
// DeliverItem, ManufactureItem, and MineMaterial. See ShipTypeMatcher's doc comment for why this
// is a distinct generated Go type from the "ships"-field oneOf despite the identical shape.
func toShipTypeMatchersItems(vs []esiapi.CorporationsProjectsDetailConfigurationdeliveritemItemsInner) []ShipTypeMatcher {
	out := make([]ShipTypeMatcher, len(vs))
	for i, v := range vs {
		m := ShipTypeMatcher{}
		if v.GroupId1 != nil {
			m.GroupID = v.GroupId1.GroupId
		}
		if v.TypeId != nil {
			m.TypeID = v.TypeId.TypeId
		}
		out[i] = m
	}
	return out
}

func toDockingLocationMatchers(vs []esiapi.CorporationsProjectsDetailConfigurationdeliveritemDockingLocationsInner) []DockingLocationMatcher {
	out := make([]DockingLocationMatcher, len(vs))
	for i, v := range vs {
		m := DockingLocationMatcher{}
		if v.StationId != nil {
			m.StationID = v.StationId.StationId
		}
		if v.StructureId != nil {
			m.StructureID = v.StructureId.StructureId
		}
		out[i] = m
	}
	return out
}

func toArchetypeMatchers(vs []esiapi.CorporationsProjectsDetailConfigurationmatcherarchetype) []ArchetypeMatcher {
	out := make([]ArchetypeMatcher, len(vs))
	for i, v := range vs {
		out[i] = ArchetypeMatcher{ArchetypeID: v.ArchetypeId}
	}
	return out
}

func toFactionMatchers(vs []esiapi.CorporationsProjectsDetailConfigurationmatcherfaction) []FactionMatcher {
	out := make([]FactionMatcher, len(vs))
	for i, v := range vs {
		out[i] = FactionMatcher{FactionID: v.FactionId}
	}
	return out
}

func toCorporationMatchers(vs []esiapi.CorporationsProjectsDetailConfigurationmatchercorporation) []CorporationMatcher {
	out := make([]CorporationMatcher, len(vs))
	for i, v := range vs {
		out[i] = CorporationMatcher{CorporationID: v.CorporationId}
	}
	return out
}

func toSignatureMatchers(vs []esiapi.CorporationsProjectsDetailConfigurationmatchersignature) []SignatureMatcher {
	out := make([]SignatureMatcher, len(vs))
	for i, v := range vs {
		out[i] = SignatureMatcher{SignatureTypeID: v.SignatureTypeId}
	}
	return out
}

// toProjectConfiguration flattens ESI's 17-variant Configuration oneOf into ProjectConfiguration.
// Exactly one of cfg's pointer fields is expected non-nil per ESI's own oneOf contract; if somehow
// none are set, Kind is left "" and every other field zero.
func toProjectConfiguration(cfg esiapi.CorporationsProjectsDetailConfiguration) ProjectConfiguration {
	switch {
	case cfg.CaptureFwComplex != nil && cfg.CaptureFwComplex.CaptureFwComplex != nil:
		v := cfg.CaptureFwComplex.CaptureFwComplex
		return ProjectConfiguration{
			Kind:       "capture_fw_complex",
			Archetypes: toArchetypeMatchers(v.Archetypes),
			Factions:   toFactionMatchers(v.Factions),
			Locations:  toLocationMatchers(v.Locations),
		}
	case cfg.DamageShip != nil && cfg.DamageShip.DamageShip != nil:
		v := cfg.DamageShip.DamageShip
		return ProjectConfiguration{
			Kind:       "damage_ship",
			Identities: toIdentityMatchers(v.Identities),
			Locations:  toLocationMatchers(v.Locations),
			Ships:      toShipTypeMatchersShips(v.Ships),
		}
	case cfg.DefendFwComplex != nil && cfg.DefendFwComplex.DefendFwComplex != nil:
		v := cfg.DefendFwComplex.DefendFwComplex
		return ProjectConfiguration{
			Kind:       "defend_fw_complex",
			Archetypes: toArchetypeMatchers(v.Archetypes),
			Factions:   toFactionMatchers(v.Factions),
			Locations:  toLocationMatchers(v.Locations),
		}
	case cfg.DeliverItem != nil && cfg.DeliverItem.DeliverItem != nil:
		v := cfg.DeliverItem.DeliverItem
		return ProjectConfiguration{
			Kind:             "deliver_item",
			DockingLocations: toDockingLocationMatchers(v.DockingLocations),
			Items:            toShipTypeMatchersItems(v.Items),
			OfficeID:         v.OfficeId,
		}
	case cfg.DestroyNpc != nil && cfg.DestroyNpc.DestroyNpc != nil:
		v := cfg.DestroyNpc.DestroyNpc
		return ProjectConfiguration{
			Kind:      "destroy_npc",
			Locations: toLocationMatchers(v.Locations),
		}
	case cfg.DestroyShip != nil && cfg.DestroyShip.DestroyShip != nil:
		v := cfg.DestroyShip.DestroyShip
		return ProjectConfiguration{
			Kind:       "destroy_ship",
			Identities: toIdentityMatchers(v.Identities),
			Locations:  toLocationMatchers(v.Locations),
			Ships:      toShipTypeMatchersShips(v.Ships),
		}
	case cfg.EarnLoyaltyPoint != nil && cfg.EarnLoyaltyPoint.EarnLoyaltyPoint != nil:
		v := cfg.EarnLoyaltyPoint.EarnLoyaltyPoint
		return ProjectConfiguration{
			Kind:         "earn_loyalty_point",
			Corporations: toCorporationMatchers(v.Corporations),
		}
	case cfg.LostShip != nil && cfg.LostShip.LostShip != nil:
		v := cfg.LostShip.LostShip
		return ProjectConfiguration{
			Kind:       "lost_ship",
			Identities: toIdentityMatchers(v.Identities),
			Locations:  toLocationMatchers(v.Locations),
			Ships:      toShipTypeMatchersShips(v.Ships),
		}
	case cfg.Manual != nil:
		// Manual is itself free-form per ESI's own spec (map[string]interface{}) -- there is no
		// further typed structure to map, so it's passed through as-is.
		return ProjectConfiguration{
			Kind:   "manual",
			Manual: cfg.Manual.Manual,
		}
	case cfg.ManufactureItem != nil && cfg.ManufactureItem.ManufactureItem != nil:
		v := cfg.ManufactureItem.ManufactureItem
		return ProjectConfiguration{
			Kind:             "manufacture_item",
			DockingLocations: toDockingLocationMatchers(v.DockingLocations),
			Items:            toShipTypeMatchersItems(v.Items),
			Owner:            v.Owner,
		}
	case cfg.MineMaterial != nil && cfg.MineMaterial.MineMaterial != nil:
		v := cfg.MineMaterial.MineMaterial
		return ProjectConfiguration{
			Kind:      "mine_material",
			Locations: toLocationMatchers(v.Locations),
			Materials: toShipTypeMatchersItems(v.Materials),
		}
	case cfg.RemoteBoostShield != nil && cfg.RemoteBoostShield.RemoteBoostShield != nil:
		v := cfg.RemoteBoostShield.RemoteBoostShield
		return ProjectConfiguration{
			Kind:       "remote_boost_shield",
			Identities: toIdentityMatchers(v.Identities),
			Locations:  toLocationMatchers(v.Locations),
			Ships:      toShipTypeMatchersShips(v.Ships),
		}
	case cfg.RemoteRepairArmor != nil && cfg.RemoteRepairArmor.RemoteRepairArmor != nil:
		v := cfg.RemoteRepairArmor.RemoteRepairArmor
		return ProjectConfiguration{
			Kind:       "remote_repair_armor",
			Identities: toIdentityMatchers(v.Identities),
			Locations:  toLocationMatchers(v.Locations),
			Ships:      toShipTypeMatchersShips(v.Ships),
		}
	case cfg.SalvageWreck != nil && cfg.SalvageWreck.SalvageWreck != nil:
		v := cfg.SalvageWreck.SalvageWreck
		return ProjectConfiguration{
			Kind:      "salvage_wreck",
			Locations: toLocationMatchers(v.Locations),
		}
	case cfg.ScanSignature != nil && cfg.ScanSignature.ScanSignature != nil:
		v := cfg.ScanSignature.ScanSignature
		return ProjectConfiguration{
			Kind:       "scan_signature",
			Locations:  toLocationMatchers(v.Locations),
			Signatures: toSignatureMatchers(v.Signatures),
		}
	case cfg.ShipInsurance != nil && cfg.ShipInsurance.ShipInsurance != nil:
		v := cfg.ShipInsurance.ShipInsurance
		return ProjectConfiguration{
			Kind:              "ship_insurance",
			ConflictType:      v.ConflictType,
			Identities:        toIdentityMatchers(v.Identities),
			Locations:         toLocationMatchers(v.Locations),
			ReimburseImplants: v.ReimburseImplants,
			Ships:             toShipTypeMatchersShips(v.Ships),
		}
	case cfg.Unknown != nil && cfg.Unknown.Unknown != nil:
		v := cfg.Unknown.Unknown
		// Unknown is itself free-form per ESI's own spec (Data is interface{}) -- there is no
		// further typed structure to map, so it's passed through as-is alongside ESI's own Type
		// discriminator string.
		return ProjectConfiguration{
			Kind:        "unknown",
			UnknownType: v.Type,
			UnknownData: v.Data,
		}
	default:
		return ProjectConfiguration{}
	}
}

// --- shared project summary/detail types ---

type ProjectProgress struct {
	Current int64 `json:"current"`
	Desired int64 `json:"desired"`
}

func toProjectProgress(p esiapi.CorporationsProjectsDetailProgress) ProjectProgress {
	return ProjectProgress{Current: p.Current, Desired: p.Desired}
}

type ProjectReward struct {
	Initial   float64 `json:"initial"`
	Remaining float64 `json:"remaining"`
}

func toProjectReward(r *esiapi.CorporationsProjectsDetailReward) *ProjectReward {
	if r == nil {
		return nil
	}
	return &ProjectReward{Initial: r.Initial, Remaining: r.Remaining}
}

// ProjectSummary is one entry from list_corporation_projects (CorporationsProjectsDetailProject).
type ProjectSummary struct {
	ProjectID    string          `json:"projectId"`
	Name         string          `json:"name"`
	State        string          `json:"state"`
	LastModified time.Time       `json:"lastModified"`
	Progress     ProjectProgress `json:"progress"`
	Reward       *ProjectReward  `json:"reward,omitempty"`
}

func toProjectSummary(p esiapi.CorporationsProjectsDetailProject) ProjectSummary {
	return ProjectSummary{
		ProjectID:    p.Id,
		Name:         p.Name,
		State:        p.State,
		LastModified: p.LastModified,
		Progress:     toProjectProgress(p.Progress),
		Reward:       toProjectReward(p.Reward),
	}
}

// ProjectContributionLimits mirrors CorporationsProjectsDetailContribution -- the project's
// configured contribution limits/rates, not any one contributor's progress.
type ProjectContributionLimits struct {
	ParticipationLimit    *int64   `json:"participationLimit,omitempty"`
	RewardPerContribution *float64 `json:"rewardPerContribution,omitempty"`
	SubmissionLimit       *int64   `json:"submissionLimit,omitempty"`
	SubmissionMultiplier  *float64 `json:"submissionMultiplier,omitempty"`
}

func toProjectContributionLimits(c *esiapi.CorporationsProjectsDetailContribution) *ProjectContributionLimits {
	if c == nil {
		return nil
	}
	return &ProjectContributionLimits{
		ParticipationLimit:    c.ParticipationLimit,
		RewardPerContribution: c.RewardPerContribution,
		SubmissionLimit:       c.SubmissionLimit,
		SubmissionMultiplier:  c.SubmissionMultiplier,
	}
}

type ProjectCreator struct {
	CreatorID int64  `json:"creatorId"`
	Name      string `json:"name"`
}

type ProjectDetails struct {
	Career      string     `json:"career"`
	Created     time.Time  `json:"created"`
	Description string     `json:"description"`
	Expires     *time.Time `json:"expires,omitempty"`
	Finished    *time.Time `json:"finished,omitempty"`
}

// --- get_corporation_project_contribution (private) ---

var getCorporationProjectContribution = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, targetCharacterID int64) (*esiapi.CorporationsProjectsContribution, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationProjectContribution(ctx, corporationID, projectID, targetCharacterID)
}

type GetCorporationProjectContributionInput struct {
	CharacterID       int32  `json:"characterId" jsonschema:"the character ID to check (whose stored auth token authenticates this call)"`
	CorporationID     int64  `json:"corporationId" jsonschema:"the corporation ID the project belongs to"`
	ProjectID         string `json:"projectId" jsonschema:"the project ID, from list_corporation_projects"`
	TargetCharacterID int64  `json:"targetCharacterId" jsonschema:"the character ID whose contribution to look up -- may differ from characterId"`
}
type GetCorporationProjectContributionOutput struct {
	CharacterID       int32      `json:"characterId"`
	CorporationID     int64      `json:"corporationId"`
	ProjectID         string     `json:"projectId"`
	TargetCharacterID int64      `json:"targetCharacterId"`
	Contributed       int64      `json:"contributed"`
	LastModified      *time.Time `json:"lastModified,omitempty"`
}

func getCorporationProjectContributionHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationProjectContributionInput, GetCorporationProjectContributionOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationProjectContributionInput) (*mcpsdk.CallToolResult, GetCorporationProjectContributionOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationProjectContributionOutput{}, err
		}
		if !found {
			return nil, GetCorporationProjectContributionOutput{}, fmt.Errorf("get corporation project contribution for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contribution, err := getCorporationProjectContribution(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ProjectID, in.TargetCharacterID)
		if err != nil {
			return nil, GetCorporationProjectContributionOutput{}, fmt.Errorf("get corporation project contribution: %w", err)
		}
		return nil, GetCorporationProjectContributionOutput{
			CharacterID:       in.CharacterID,
			CorporationID:     in.CorporationID,
			ProjectID:         in.ProjectID,
			TargetCharacterID: in.TargetCharacterID,
			Contributed:       contribution.Contributed,
			LastModified:      contribution.LastModified,
		}, nil
	}
}

// --- list_corporation_project_contributors (private, cursor-paginated) ---

type ProjectContributor struct {
	ContributorID int64  `json:"contributorId"`
	Name          string `json:"name"`
	Contributed   int64  `json:"contributed"`
}

var listCorporationProjectContributors = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, after, before string, limit int64) ([]esiapi.CorporationsProjectsContributorsContributor, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationProjectContributors(ctx, corporationID, projectID, after, before, limit)
}

type ListCorporationProjectContributorsInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation ID the project belongs to"`
	ProjectID     string `json:"projectId" jsonschema:"the project ID, from list_corporation_projects"`
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
}
type ListCorporationProjectContributorsOutput struct {
	CharacterID   int32                `json:"characterId"`
	CorporationID int64                `json:"corporationId"`
	ProjectID     string               `json:"projectId"`
	Contributors  []ProjectContributor `json:"contributors"`
	NextAfter     *string              `json:"nextAfter,omitempty"`
	NextBefore    *string              `json:"nextBefore,omitempty"`
}

func listCorporationProjectContributorsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationProjectContributorsInput, ListCorporationProjectContributorsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationProjectContributorsInput) (*mcpsdk.CallToolResult, ListCorporationProjectContributorsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationProjectContributorsOutput{}, err
		}
		if !found {
			return nil, ListCorporationProjectContributorsOutput{}, fmt.Errorf("list corporation project contributors for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contributors, cursor, err := listCorporationProjectContributors(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ProjectID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListCorporationProjectContributorsOutput{}, fmt.Errorf("list corporation project contributors: %w", err)
		}
		out := make([]ProjectContributor, len(contributors))
		for i, c := range contributors {
			out[i] = ProjectContributor{ContributorID: c.Id, Name: c.Name, Contributed: c.Contributed}
		}
		result := ListCorporationProjectContributorsOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, ProjectID: in.ProjectID, Contributors: out,
		}
		if cursor != nil {
			result.NextAfter = cursor.After
			result.NextBefore = cursor.Before
		}
		return nil, result, nil
	}
}

// --- get_corporation_project_detail (private) ---

var getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationProjectDetail(ctx, corporationID, projectID)
}

type GetCorporationProjectDetailInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation ID the project belongs to"`
	ProjectID     string `json:"projectId" jsonschema:"the project ID, from list_corporation_projects"`
}
type GetCorporationProjectDetailOutput struct {
	CharacterID   int32                      `json:"characterId"`
	CorporationID int64                      `json:"corporationId"`
	ProjectID     string                     `json:"projectId"`
	Configuration ProjectConfiguration       `json:"configuration"`
	Contribution  *ProjectContributionLimits `json:"contribution,omitempty"`
	Creator       ProjectCreator             `json:"creator"`
	Details       ProjectDetails             `json:"details"`
	LastModified  time.Time                  `json:"lastModified"`
	Name          string                     `json:"name"`
	Progress      ProjectProgress            `json:"progress"`
	Reward        *ProjectReward             `json:"reward,omitempty"`
	State         string                     `json:"state"`
}

func getCorporationProjectDetailHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationProjectDetailInput, GetCorporationProjectDetailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationProjectDetailInput) (*mcpsdk.CallToolResult, GetCorporationProjectDetailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationProjectDetailOutput{}, err
		}
		if !found {
			return nil, GetCorporationProjectDetailOutput{}, fmt.Errorf("get corporation project detail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCorporationProjectDetail(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ProjectID)
		if err != nil {
			return nil, GetCorporationProjectDetailOutput{}, fmt.Errorf("get corporation project detail: %w", err)
		}
		return nil, GetCorporationProjectDetailOutput{
			CharacterID:   in.CharacterID,
			CorporationID: in.CorporationID,
			ProjectID:     in.ProjectID,
			Configuration: toProjectConfiguration(detail.Configuration),
			Contribution:  toProjectContributionLimits(detail.Contribution),
			Creator:       ProjectCreator{CreatorID: detail.Creator.Id, Name: detail.Creator.Name},
			Details: ProjectDetails{
				Career: detail.Details.Career, Created: detail.Details.Created, Description: detail.Details.Description,
				Expires: detail.Details.Expires, Finished: detail.Details.Finished,
			},
			LastModified: detail.LastModified,
			Name:         detail.Name,
			Progress:     toProjectProgress(detail.Progress),
			Reward:       toProjectReward(detail.Reward),
			State:        detail.State,
		}, nil
	}
}

// --- list_corporation_projects (private, cursor-paginated) ---

var listCorporationProjects = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64, state string) ([]esiapi.CorporationsProjectsDetailProject, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationProjectListing(ctx, corporationID, after, before, limit, state)
}

type ListCorporationProjectsInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation ID to look up"`
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
	State         string `json:"state,omitempty" jsonschema:"filter by project state; omit for ESI's default (Active)"`
}
type ListCorporationProjectsOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	Projects      []ProjectSummary `json:"projects"`
	NextAfter     *string          `json:"nextAfter,omitempty"`
	NextBefore    *string          `json:"nextBefore,omitempty"`
}

func listCorporationProjectsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationProjectsInput, ListCorporationProjectsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationProjectsInput) (*mcpsdk.CallToolResult, ListCorporationProjectsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationProjectsOutput{}, err
		}
		if !found {
			return nil, ListCorporationProjectsOutput{}, fmt.Errorf("list corporation projects for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		projects, cursor, err := listCorporationProjects(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.After, in.Before, in.Limit, in.State)
		if err != nil {
			return nil, ListCorporationProjectsOutput{}, fmt.Errorf("list corporation projects: %w", err)
		}
		out := make([]ProjectSummary, len(projects))
		for i, p := range projects {
			out[i] = toProjectSummary(p)
		}
		result := ListCorporationProjectsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Projects: out}
		if cursor != nil {
			result.NextAfter = cursor.After
			result.NextBefore = cursor.Before
		}
		return nil, result, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated
sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the
identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Write `mcp/corporation_projects_internal_test.go`**

Follow `mcp/corporation_internal_test.go`'s pattern (package-level `var` override + `t.Cleanup`;
`keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). Cover:

- `TestGetCorporationProjectContributionHandler_MapsFields` — stub `getCorporationProjectContribution` to return a populated `*esiapi.CorporationsProjectsContribution`, assert every output field maps, INCLUDING that `TargetCharacterID` in the output matches the distinct `TargetCharacterID` passed in the input (not `CharacterID`).
- `TestGetCorporationProjectContributionHandler_UnknownCharacter`
- `TestListCorporationProjectContributorsHandler_MapsFieldsAndCursor` — stub to return contributors plus a non-nil `*esiapi.Cursor{After: &"a", Before: &"b"}`, assert `NextAfter`/`NextBefore` map from the cursor.
- `TestListCorporationProjectContributorsHandler_UnknownCharacter`
- `TestGetCorporationProjectDetailHandler_MapsFields` — stub `getCorporationProjectDetail` to return a `*esiapi.CorporationsProjectsDetail` with `Configuration` set to the `CaptureFwComplex` variant (populate its `Archetypes`/`Factions`/`Locations`), assert `Configuration.Kind == "capture_fw_complex"` and its `Archetypes`/`Factions`/`Locations` map through `toArchetypeMatchers`/`toFactionMatchers`/`toLocationMatchers` correctly; also assert `Creator`, `Details`, `Progress`, `Reward`, `Contribution` all map.
- `TestGetCorporationProjectDetailHandler_ConfigurationManualVariant` — a second detail test with `Configuration.Manual` set (the free-form variant), asserting `Configuration.Kind == "manual"` and `Configuration.Manual` equals the stubbed map.
- `TestGetCorporationProjectDetailHandler_ConfigurationUnknownVariant` — a third detail test with `Configuration.Unknown` set, asserting `Configuration.Kind == "unknown"`, `UnknownType`, and `UnknownData` all map.
- `TestGetCorporationProjectDetailHandler_UnknownCharacter`
- `TestListCorporationProjectsHandler_MapsFieldsAndCursor` — stub to return projects plus a cursor, assert `ProjectSummary` fields and `NextAfter`/`NextBefore` map.
- `TestListCorporationProjectsHandler_UnknownCharacter`

That's 4 happy-path-family tests (one of which -- detail -- gets 3 variant-coverage tests instead of 1) plus 4 unknown-character tests = 10 test functions total.

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run CorporationProject -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 4 tools in `mcp/server.go`**

Add these 4 `mcpsdk.AddTool` blocks immediately after the existing `list_corporation_structures` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_project_contribution",
		Description: "Get a character's contribution to a corporation project, using a logged-in character's roles to authorize.",
	}, getCorporationProjectContributionHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_project_contributors",
		Description: "List the contributors to a corporation project, using a logged-in character's roles to authorize.",
	}, listCorporationProjectContributorsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_project_detail",
		Description: "Get the full details (configuration, progress, reward) of a corporation project, using a logged-in character's roles to authorize.",
	}, getCorporationProjectDetailHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_projects",
		Description: "List a corporation's projects, using a logged-in character's roles to authorize.",
	}, listCorporationProjectsHandler(version, clientID, store))

	return server
}
```

Update the package doc comment above `NewServer` to append the 4 new tool names to its running
list (ending in `..., list_corporation_structures, get_corporation_project_contribution,
list_corporation_project_contributors, get_corporation_project_detail, and
list_corporation_projects registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 4 names to the `[]string{...}` literal, right after `"list_corporation_structures",`:

```go
			"get_corporation_project_contribution", "list_corporation_project_contributors",
			"get_corporation_project_detail", "list_corporation_projects",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 83 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "seventy-nine" to "eighty-three".
- Append the 4 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Corporation Projects row from `Planned` to `This plan`.

- [ ] **Step 8: Commit**

```bash
git add mcp/corporation_projects.go mcp/corporation_projects_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add corporation projects MCP tools (4 tools: cursor pagination, 17-variant Configuration)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff e08450d..HEAD` — the design spec commit forward).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
