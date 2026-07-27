# ESI Universe Implementation Plan (33rd batch)

> **For agentic workers:** This plan is executed directly by the orchestrator (no subagent
> dispatch), per the user's explicit instruction after this session's subagent spawn cap was hit
> during the Sovereignty batch. Steps use checkbox (`- [ ]`) syntax for tracking only.

**Goal:** Add ESI coverage for the "Universe" tag -- 30 endpoints covering reference data
(categories/groups/types, races/bloodlines/ancestries/factions), geography (regions/
constellations/systems), celestial detail (asteroid belts/moons/planets/stargates/stars/stations),
graphics, player structures, system activity stats, and bulk ID/name resolution.

**Architecture:** One `esi/universe.go` facade file (30 methods, almost entirely public reference
data via `esi.NewPublicClient`; exactly one private method, `UniverseStructureDetail`) + one
`mcp/universe.go` tool file (30 tools). Given the unusual size, both files are built up across 3
tasks each (esi/ then mcp/), rather than this sweep's usual 1-task-per-layer split.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi@v0.0.37`,
`github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Current tool count baseline: **179** (verify with `grep -c "mcpsdk.AddTool" mcp/server.go`
  before any doc-count edit -- if drifted, use the real count).
- Branch: `worktree-esi-access-list-activities` (shared PR #6). Base commit before Task 1: `ad01bc1`.
- Every method returns exactly what ESI returns, zero name-resolution enrichment -- notably
  applies even to the `ids`/`names` resolution endpoints themselves: they return ESI's raw
  category-keyed shape, not a further-joined result.
- Only `GetUniverseStructuresStructureId` (`get_universe_structure`) requires OAuth2
  (`esi-universe.read_structures.v1`) -- already present in `mcp/add_character.go`'s `allScopes`
  from an earlier batch, no scope changes needed. Every other endpoint in this batch is public --
  use `esi.NewPublicClient`, not `esi.NewClient`.
- Only `GetUniverseGroups` and `GetUniverseTypes` are paginated (verified individually from
  `api_universe.go` -- both have a `.Page()` setter; no other endpoint in this batch does). Use
  `totalPagesFromHeader` (`esi/contracts.go`) for both, same as
  `esi/planetary_interaction.go`'s `CorporationCustomsOffices`.
- `get_universe_structure` follows `mcp/market.go`'s `list_structure_market_orders` pattern
  exactly: `CharacterID` authenticates, `StructureID` is the resource, no separate owner/corp
  field, ESI's 403 is the sole gate.
- The full design spec, including every verified field name and model shape, is at
  `docs/superpowers/specs/2026-07-26-esi-universe-design.md` -- read it before starting any task.
  It is the source of truth for exact field names, not this plan's prose summaries.
- Nested/tree-shaped models must never use a self-referential Go struct (verified panic risk in
  the MCP SDK's schema generator). Every nested type in this batch was already verified
  non-recursive in the design spec -- none require flattening beyond what's already documented
  there (Position, IDNamePair, the Dogma attribute/effect reuse).
- `mcp/dogma.go` already defines `DynamicItemAttribute`/`DynamicItemEffect` -- reuse them for
  `get_universe_type`'s dogma fields, do not redeclare.
- Watch for naming collisions against existing `mcp/*.go` types before introducing any new type
  name (the Structures batch found one real collision this way -- `SovereigntyHub`). Grep
  `mcp/*.go` for `Position`, `IDNamePair`, `ResolvedName`, `UniverseStructure` etc. before adding
  them.

---

## Task 1: `esi/universe.go` part A -- reference data (10 methods)

**Files:**
- Create: `esi/universe.go`
- Create: `esi/universe_test.go`

**Interfaces:**
- Consumes: `esi.NewPublicClient(userAgent) *Client`, `describeError`, `totalPagesFromHeader`
  (`esi/contracts.go`).
- Produces:
  ```go
  func (c *Client) UniverseAncestries(ctx context.Context) ([]esiapi.UniverseAncestriesGetInner, error)
  func (c *Client) UniverseBloodlines(ctx context.Context) ([]esiapi.UniverseBloodlinesGetInner, error)
  func (c *Client) UniverseRaces(ctx context.Context) ([]esiapi.UniverseRacesGetInner, error)
  func (c *Client) UniverseFactions(ctx context.Context) ([]esiapi.UniverseFactionsGetInner, error)
  func (c *Client) UniverseCategories(ctx context.Context) ([]int64, error)
  func (c *Client) UniverseCategoryDetail(ctx context.Context, categoryID int64) (*esiapi.UniverseCategoriesCategoryIdGet, error)
  func (c *Client) UniverseGroups(ctx context.Context, page int32) ([]int64, int32, error)
  func (c *Client) UniverseGroupDetail(ctx context.Context, groupID int64) (*esiapi.UniverseGroupsGroupIdGet, error)
  func (c *Client) UniverseTypes(ctx context.Context, page int32) ([]int64, int32, error)
  func (c *Client) UniverseTypeDetail(ctx context.Context, typeID int64) (*esiapi.UniverseTypesTypeIdGet, error)
  ```
  These exact signatures are what Task 4 will call.

- [ ] **Step 1: Write `esi/universe.go`**

```go
// Package esi's universe.go covers all 30 of "Universe"'s endpoints -- static reference data
// (ancestries, bloodlines, races, factions, categories/groups/types), geography (regions,
// constellations, systems), celestial detail (asteroid belts, moons, planets, stargates, stars,
// stations), graphics, player structures, system activity stats, and bulk ID/name resolution.
// Every endpoint is public EXCEPT UniverseStructureDetail (esi-universe.read_structures.v1) --
// verified individually per endpoint from esi/api/openapi.yaml, not assumed from tag similarity.
// Only GetUniverseGroups and GetUniverseTypes are paginated (verified individually: no .Page()
// setter on any other of the 30 request builders).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func (c *Client) UniverseAncestries(ctx context.Context) ([]esiapi.UniverseAncestriesGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseAncestries(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe ancestries: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseBloodlines(ctx context.Context) ([]esiapi.UniverseBloodlinesGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseBloodlines(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe bloodlines: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseRaces(ctx context.Context) ([]esiapi.UniverseRacesGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseRaces(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe races: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseFactions(ctx context.Context) ([]esiapi.UniverseFactionsGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseFactions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe factions: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseCategories(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseCategories(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe categories: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseCategoryDetail(ctx context.Context, categoryID int64) (*esiapi.UniverseCategoriesCategoryIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseCategoriesCategoryId(ctx, categoryID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe category detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseGroups(ctx context.Context, page int32) ([]int64, int32, error) {
	req := c.api.UniverseAPI.GetUniverseGroups(ctx)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get universe groups: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

func (c *Client) UniverseGroupDetail(ctx context.Context, groupID int64) (*esiapi.UniverseGroupsGroupIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseGroupsGroupId(ctx, groupID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe group detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseTypes(ctx context.Context, page int32) ([]int64, int32, error) {
	req := c.api.UniverseAPI.GetUniverseTypes(ctx)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get universe types: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

func (c *Client) UniverseTypeDetail(ctx context.Context, typeID int64) (*esiapi.UniverseTypesTypeIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseTypesTypeId(ctx, typeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe type detail: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Write `esi/universe_test.go`**

Follow the exact `httptest.Server` + `esi.ExportedNewPublicClientWithOptions(srv.Client(),
goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})` pattern from
`esi/sovereignty_test.go`. Write 20 tests (10 methods x success + HTTP-error). For the 2 paginated
methods, also assert `totalPages` is parsed from an `X-Pages` response header (set
`w.Header().Set("X-Pages", "3")` in the test server and assert the returned int32 is 3), mirroring
`esi/planetary_interaction_test.go`'s `TestCorporationCustomsOffices_ParsesResponseAndTotalPages`.
Use realistic fixture values (e.g. category_id 6 = Ship, group_id 25 = Frigate, type_id 587 =
Rifter) -- exact values don't matter for correctness, just use consistent IDs across a given
test's request path and response body.

- [ ] **Step 3: Run the new tests**

Run: `go test ./esi/... -run 'TestUniverseAncestries|TestUniverseBloodlines|TestUniverseRaces|TestUniverseFactions|TestUniverseCategories|TestUniverseCategoryDetail|TestUniverseGroups|TestUniverseGroupDetail|TestUniverseTypes|TestUniverseTypeDetail' -v`
Expected: all pass.

- [ ] **Step 4: `go build ./... && go vet ./... && gofmt -l .`**

Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/universe.go esi/universe_test.go
git commit -m "feat: add universe esi facade part A (ancestries, bloodlines, races, factions, categories, groups, types)"
```

---

## Task 2: `esi/universe.go` part B -- geography + celestial detail (12 methods)

**Files:**
- Modify: `esi/universe.go` (append)
- Modify: `esi/universe_test.go` (append)

**Interfaces:**
- Produces:
  ```go
  func (c *Client) UniverseRegions(ctx context.Context) ([]int64, error)
  func (c *Client) UniverseRegionDetail(ctx context.Context, regionID int64) (*esiapi.UniverseRegionsRegionIdGet, error)
  func (c *Client) UniverseConstellations(ctx context.Context) ([]int64, error)
  func (c *Client) UniverseConstellationDetail(ctx context.Context, constellationID int64) (*esiapi.UniverseConstellationsConstellationIdGet, error)
  func (c *Client) UniverseSystems(ctx context.Context) ([]int64, error)
  func (c *Client) UniverseSystemDetail(ctx context.Context, systemID int64) (*esiapi.UniverseSystemsSystemIdGet, error)
  func (c *Client) UniverseAsteroidBeltDetail(ctx context.Context, asteroidBeltID int64) (*esiapi.UniverseAsteroidBeltsAsteroidBeltIdGet, error)
  func (c *Client) UniverseMoonDetail(ctx context.Context, moonID int64) (*esiapi.UniverseMoonsMoonIdGet, error)
  func (c *Client) UniversePlanetDetail(ctx context.Context, planetID int64) (*esiapi.UniversePlanetsPlanetIdGet, error)
  func (c *Client) UniverseStargateDetail(ctx context.Context, stargateID int64) (*esiapi.UniverseStargatesStargateIdGet, error)
  func (c *Client) UniverseStarDetail(ctx context.Context, starID int64) (*esiapi.UniverseStarsStarIdGet, error)
  func (c *Client) UniverseStationDetail(ctx context.Context, stationID int64) (*esiapi.UniverseStationsStationIdGet, error)
  ```

- [ ] **Step 1: Append the 12 methods to `esi/universe.go`**

Follow the exact same shape as Task 1's methods -- each is `req := c.api.UniverseAPI.Get...(ctx,
id).Execute()`, wrap errors with `describeError`, return the result. None of these 12 is paginated.
Every ID param name matches the ESI path parameter (`regionID`, `constellationID`, `systemID`,
`asteroidBeltID`, `moonID`, `planetID`, `stargateID`, `starID`, `stationID`) -- verify each against
`api_universe.go`'s actual generated parameter name before writing, per this project's standing
"verify every method/model/security/pagination detail directly, never assume from tag similarity"
rule.

- [ ] **Step 2: Append 24 tests to `esi/universe_test.go`** (12 methods x success + HTTP-error)

Use the `CharactersCharacterIdAssetsLocationsPostInnerPosition{X,Y,Z}` shape for
`position`/`x`/`y`/`z` fields in fixtures (AsteroidBelt, Moon, Planet, Constellation, System all
carry this field). For `UniverseSystemDetail`'s `planets` field, include at least one
`UniverseSystemsSystemIdGetPlanetsInner{AsteroidBelts, Moons, PlanetId}` entry in the fixture and
assert it round-trips.

- [ ] **Step 3: Run the new tests, then build/vet/gofmt**

Run: `go test ./esi/... -run 'TestUniverseRegions|TestUniverseRegionDetail|TestUniverseConstellations|TestUniverseConstellationDetail|TestUniverseSystems|TestUniverseSystemDetail|TestUniverseAsteroidBeltDetail|TestUniverseMoonDetail|TestUniversePlanetDetail|TestUniverseStargateDetail|TestUniverseStarDetail|TestUniverseStationDetail' -v`
then `go build ./... && go vet ./... && gofmt -l .`
Expected: all pass, clean.

- [ ] **Step 4: Commit**

```bash
git add esi/universe.go esi/universe_test.go
git commit -m "feat: add universe esi facade part B (regions, constellations, systems, celestial detail)"
```

---

## Task 3: `esi/universe.go` part C -- graphics, structures, stats, bulk resolve (8 methods)

**Files:**
- Modify: `esi/universe.go` (append)
- Modify: `esi/universe_test.go` (append)

**Interfaces:**
- Produces:
  ```go
  func (c *Client) UniverseGraphics(ctx context.Context) ([]int64, error)
  func (c *Client) UniverseGraphicDetail(ctx context.Context, graphicID int64) (*esiapi.UniverseGraphicsGraphicIdGet, error)
  func (c *Client) UniverseStructures(ctx context.Context) ([]int64, error)
  func (c *Client) UniverseStructureDetail(ctx context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error)
  func (c *Client) UniverseSystemJumps(ctx context.Context) ([]esiapi.UniverseSystemJumpsGetInner, error)
  func (c *Client) UniverseSystemKills(ctx context.Context) ([]esiapi.UniverseSystemKillsGetInner, error)
  func (c *Client) UniverseIDsByName(ctx context.Context, names []string) (*esiapi.UniverseIdsPost, error)
  func (c *Client) UniverseNamesByID(ctx context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error)
  ```

- [ ] **Step 1: Append the 8 methods to `esi/universe.go`**

`UniverseStructureDetail` is the ONLY private method in this file -- it is called on a `*Client`
built via `esi.NewClient` (per-character authenticated), same as every other private method in the
codebase; it needs no additional characterID parameter of its own since that's baked into the
`Client` at construction (mirrors `esi/market.go`'s `StructureMarketOrders`). Every other method
here, like every method in Tasks 1-2, is public.

`UniverseIDsByName` and `UniverseNamesByID` are POST-based bulk lookups -- use
`.RequestBody(names)` / `.RequestBody(ids)` on the request builder before `.Execute()`, mirroring
`esi/assets.go`'s `AssetLocations`/`AssetNames` POST precedent.

```go
func (c *Client) UniverseGraphics(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseGraphics(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe graphics: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseGraphicDetail(ctx context.Context, graphicID int64) (*esiapi.UniverseGraphicsGraphicIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseGraphicsGraphicId(ctx, graphicID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe graphic detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseStructures(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseStructures(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe structures: %w", describeError(err, resp))
	}
	return result, nil
}

// UniverseStructureDetail is the only private method in this file: it must be called on a
// *Client built via esi.NewClient (per-character authenticated), not esi.NewPublicClient.
func (c *Client) UniverseStructureDetail(ctx context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseStructuresStructureId(ctx, structureID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe structure detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseSystemJumps(ctx context.Context) ([]esiapi.UniverseSystemJumpsGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseSystemJumps(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe system jumps: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseSystemKills(ctx context.Context) ([]esiapi.UniverseSystemKillsGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseSystemKills(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe system kills: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseIDsByName(ctx context.Context, names []string) (*esiapi.UniverseIdsPost, error) {
	result, resp, err := c.api.UniverseAPI.PostUniverseIds(ctx).RequestBody(names).Execute()
	if err != nil {
		return nil, fmt.Errorf("resolve universe ids: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseNamesByID(ctx context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error) {
	result, resp, err := c.api.UniverseAPI.PostUniverseNames(ctx).RequestBody(ids).Execute()
	if err != nil {
		return nil, fmt.Errorf("resolve universe names: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Append 16 tests to `esi/universe_test.go`** (8 methods x success + HTTP-error)

`UniverseStructureDetail`'s test must build its client via `esi.ExportedNewClientWithOptions`
(private/authenticated), NOT `ExportedNewPublicClientWithOptions` -- every other test in this batch
uses the public variant. For `UniverseIDsByName`, assert at least 2 of the 10 categories in a
fixture round-trip (e.g. `characters` and `systems`), and assert the request body sent to the test
server is the exact `names` slice (decode `r.Body` as `[]string` and compare) -- mirroring the
POST body verification already done for `esi/assets.go`'s `AssetLocations`. Same body-verification
requirement for `UniverseNamesByID` with `[]int64`.

- [ ] **Step 3: Run the new tests, then build/vet/gofmt, then the full suite + race**

Run: `go test ./esi/... -run 'TestUniverseGraphics|TestUniverseGraphicDetail|TestUniverseStructures|TestUniverseStructureDetail|TestUniverseSystemJumps|TestUniverseSystemKills|TestUniverseIDsByName|TestUniverseNamesByID' -v`
then `go build ./... && go vet ./... && gofmt -l .`
then `go test ./... -race`
Expected: all pass, clean, all packages `ok`. This closes out the `esi/` layer for this batch --
30 methods total across Tasks 1-3.

- [ ] **Step 4: Commit**

```bash
git add esi/universe.go esi/universe_test.go
git commit -m "feat: add universe esi facade part C (graphics, structures, system stats, bulk id/name resolution)"
```

---

## Task 4: `mcp/universe.go` part A -- reference data tools (10 tools)

**Files:**
- Create: `mcp/universe.go`
- Create: `mcp/universe_internal_test.go`
- Modify: `mcp/server.go` (register these 10 tools)

**Interfaces:**
- Consumes: the 10 `esi.Client` methods from Task 1; `esi.NewPublicClient(userAgent) *Client`;
  `sso.UserAgent(version)`.

- [ ] **Step 1: Write `mcp/universe.go`** with these tools (all public, no `characterExists` gate,
  same shape as `mcp/dogma.go`'s public tools):

  - `list_universe_ancestries` -> `[]esiapi.UniverseAncestriesGetInner` flattened to a
    `UniverseAncestry{ID, BloodlineID int64, Description, Name string, IconID *int64,
    ShortDescription *string}` slice.
  - `list_universe_bloodlines` -> `UniverseBloodline{ID, Charisma, CorporationID, Intelligence,
    Memory, Perception, RaceID, ShipTypeID, Willpower int64, Description, Name string}` slice.
  - `list_universe_races` -> `UniverseRace{RaceID, AllianceID int64, Description, Name string}`
    slice.
  - `list_universe_factions` -> `UniverseFaction{FactionID int64, Description, Name string,
    IsUnique bool, SizeFactor float64, StationCount, StationSystemCount int64, CorporationID,
    MilitiaCorporationID, SolarSystemID *int64}` slice.
  - `list_universe_categories` -> `{IDs []int64}`.
  - `get_universe_category` -> input `{CategoryID int64}`, output
    `UniverseCategory{CategoryID int64, Name string, Published bool, GroupIDs []int64}`.
  - `list_universe_groups` -> input `{Page int32}`, output `{IDs []int64, Page, TotalPages int32}`.
  - `get_universe_group` -> input `{GroupID int64}`, output
    `UniverseGroup{GroupID, CategoryID int64, Name string, Published bool, TypeIDs []int64}`.
  - `list_universe_types` -> input `{Page int32}`, output `{IDs []int64, Page, TotalPages int32}`.
  - `get_universe_type` -> input `{TypeID int64}`, output `UniverseType` with all
    `UniverseTypesTypeIdGet` fields mapped 1:1 (camelCase), `DogmaAttributes []DynamicItemAttribute`
    and `DogmaEffects []DynamicItemEffect` reusing `mcp/dogma.go`'s existing types (map field-by-
    field: `AttributeId`->`AttributeID`, `Value`->`Value` for attributes;
    `EffectId`->`EffectID`, `IsDefault`->`IsDefault` for effects -- verify the exact field names on
    `DynamicItemAttribute`/`DynamicItemEffect` in `mcp/dogma.go` before writing this mapping, do
    not assume).

  Use the naming convention already established throughout this codebase: package-level swappable
  `var listX = func(ctx, userAgent string) (...)`, `XInput`/`XOutput` types, `xHandler(version
  string) mcpsdk.ToolHandlerFor[...]` closures (all public tools take only `version`, no
  `clientID`/`store` params, matching `mcp/dogma.go` and `mcp/sovereignty.go`).

- [ ] **Step 2: Write `mcp/universe_internal_test.go`** covering these 10 tools: one MapsFields
  test + one PropagatesError test per tool (20 tests). Follow
  `mcp/sovereignty_internal_test.go`'s public-tool pattern (no store/keyring setup needed since
  these tools take no `characterId`).

- [ ] **Step 3: Register the 10 tools in `mcp/server.go`**

  First run `grep -c "mcpsdk.AddTool" mcp/server.go` to get the real baseline (this plan assumes
  179). Append after the `get_corporation_sovereignty_hub` block. Do NOT update
  `mcp/server_test.go`, `CLAUDE.md`, `README.md`, or the roadmap doc yet -- that happens once, in
  Task 6, after all 30 tools exist (updating them 3 times across Tasks 4/5/6 would just create
  merge-noise within this same branch).

- [ ] **Step 4: Run tests, then build/vet/gofmt**

  Run: `go test ./mcp/... -run 'Ancestr|Bloodline|Race|Faction|Categor|Group|Type' -v`
  then `go build ./... && go vet ./... && gofmt -l .`
  Expected: all pass, clean. (`go vet`/`build` will not fully validate tool registration count
  mismatches -- that's caught by `mcp/server_test.go` in Task 6.)

- [ ] **Step 5: Commit**

```bash
git add mcp/universe.go mcp/universe_internal_test.go mcp/server.go
git commit -m "feat: add universe MCP tools part A (ancestries, bloodlines, races, factions, categories, groups, types)"
```

---

## Task 5: `mcp/universe.go` part B -- geography + celestial detail tools (12 tools)

**Files:**
- Modify: `mcp/universe.go` (append)
- Modify: `mcp/universe_internal_test.go` (append)
- Modify: `mcp/server.go` (register these 12 tools)

- [ ] **Step 1: Append the 12 tools**

  Define `Position{X, Y, Z float64}` once in this file (check first that no `mcp/*.go` file
  already declares a bare `Position` type -- grep `mcp/*.go` for `^type Position struct` before
  adding).

  - `list_universe_regions` -> `{IDs []int64}`.
  - `get_universe_region` -> input `{RegionID int64}`, output
    `UniverseRegion{RegionID int64, Name string, Description *string, ConstellationIDs []int64}`.
  - `list_universe_constellations` -> `{IDs []int64}`.
  - `get_universe_constellation` -> input `{ConstellationID int64}`, output
    `UniverseConstellation{ConstellationID, RegionID int64, Name string, Position Position,
    SystemIDs []int64}`.
  - `list_universe_systems` -> `{IDs []int64}`.
  - `get_universe_system` -> input `{SystemID int64}`, output `UniverseSystem` with
    `Planets []UniverseSystemPlanet{PlanetID int64, AsteroidBeltIDs, MoonIDs []int64}`,
    `Position Position`, plus the remaining flat fields (`ConstellationID`, `Name`,
    `SecurityClass *string`, `SecurityStatus float64`, `StarID *int64`, `StargateIDs`,
    `StationIDs []int64`, `SystemID int64`).
  - `get_universe_asteroid_belt` -> input `{AsteroidBeltID int64}`, output
    `UniverseAsteroidBelt{Name string, Position Position, SystemID int64}`.
  - `get_universe_moon` -> input `{MoonID int64}`, output
    `UniverseMoon{MoonID, SystemID int64, Name string, Position Position}`.
  - `get_universe_planet` -> input `{PlanetID int64}`, output
    `UniversePlanet{PlanetID, SystemID, TypeID int64, Name string, Position Position}`.
  - `get_universe_stargate` -> input `{StargateID int64}`, output
    `UniverseStargate{StargateID, SystemID, TypeID int64, Name string, Position Position,
    DestinationStargateID, DestinationSystemID int64}` (flatten the nested `Destination` struct
    into these 2 prefixed fields).
  - `get_universe_star` -> input `{StarID int64}`, output `UniverseStar` with all
    `UniverseStarsStarIdGet` fields mapped 1:1 (camelCase).
  - `get_universe_station` -> input `{StationID int64}`, output `UniverseStation` with all
    `UniverseStationsStationIdGet` fields mapped 1:1 (camelCase, `Position Position`).

- [ ] **Step 2: Append 24 tests to `mcp/universe_internal_test.go`** (12 tools x MapsFields +
  PropagatesError).

- [ ] **Step 3: Register the 12 tools in `mcp/server.go`** (append after Task 4's block).

- [ ] **Step 4: Run tests, then build/vet/gofmt**

  Run: `go test ./mcp/... -run 'Region|Constellation|UniverseSystem|AsteroidBelt|Moon|Planet|Stargate|Star|Station' -v`
  then `go build ./... && go vet ./... && gofmt -l .`
  Expected: all pass, clean.

- [ ] **Step 5: Commit**

```bash
git add mcp/universe.go mcp/universe_internal_test.go mcp/server.go
git commit -m "feat: add universe MCP tools part B (regions, constellations, systems, celestial detail)"
```

---

## Task 6: `mcp/universe.go` part C -- graphics/structures/stats/resolve tools + final docs sync (8 tools)

**Files:**
- Modify: `mcp/universe.go` (append)
- Modify: `mcp/universe_internal_test.go` (append)
- Modify: `mcp/server.go` (register these 8 tools + update package doc comment tail)
- Modify: `mcp/server_test.go` (append all 30 tool names -- the full batch)
- Modify: `CLAUDE.md` (tool count + tool list)
- Modify: `README.md` (tool count + tool list)
- Modify: `docs/reference/esi-coverage-roadmap.md` (row 33 "Universe": `Planned` -> `This plan`)

- [ ] **Step 1: Append the final 8 tools**

  Define `IDNamePair{ID *int64, Name *string}` once in this file (grep `mcp/*.go` for
  `^type IDNamePair struct` first).

  - `list_universe_graphics` -> `{IDs []int64}`.
  - `get_universe_graphic` -> input `{GraphicID int64}`, output `UniverseGraphic` with all
    `UniverseGraphicsGraphicIdGet` fields mapped 1:1 (camelCase, all `*string`).
  - `list_universe_structures` -> `{IDs []int64}` (public -- bare structure IDs only).
  - `get_universe_structure` (**private**) -> input `{CharacterID int32, StructureID int64}`,
    `characterExists` gate (mirrors `list_structure_market_orders`), output
    `UniverseStructure{Name string, OwnerID, SolarSystemID int64, TypeID *int64,
    Position *Position}`.
  - `list_universe_system_jumps` -> `[]UniverseSystemJumps{SystemID, ShipJumps int64}`.
  - `list_universe_system_kills` -> `[]UniverseSystemKills{SystemID, NPCKills, PodKills,
    ShipKills int64}`.
  - `resolve_universe_ids` -> input `{Names []string}`, output `ResolveUniverseIDsOutput` (all 10
    `[]IDNamePair` category fields, per the design spec).
  - `resolve_universe_names` -> input `{IDs []int64}`, output `{Names []ResolvedName}` where
    `ResolvedName{Category string, ID int64, Name string}`.

- [ ] **Step 2: Append 16 tests to `mcp/universe_internal_test.go`** (8 tools x MapsFields +
  PropagatesError; `get_universe_structure` additionally gets an UnknownCharacter test, the only
  private tool in this batch; `resolve_universe_ids` additionally gets a "sparse fixture" test
  confirming only-populated categories appear and others are nil/empty, no cross-category bleed).

- [ ] **Step 3: Register the 8 tools in `mcp/server.go`**, and update the package doc comment tail
  to end with `..., get_corporation_sovereignty_hub, list_universe_ancestries, ...,
  resolve_universe_names registered.` (spell out all 30 new names in the same style as prior
  updates).

- [ ] **Step 4: Update `mcp/server_test.go`**

  Append all 30 new tool names after the `get_corporation_sovereignty_hub` entry.

- [ ] **Step 5: Run the full test suite + race**

  Run: `go test ./... -race`
  Expected: all packages `ok`.

- [ ] **Step 6: Update `CLAUDE.md` and `README.md`**

  Change "one hundred seventy-nine" to "two hundred nine" (verify the real current count first --
  `grep -c "mcpsdk.AddTool" mcp/server.go` should read 209 after this task; adjust the words if the
  baseline drifted). Append all 30 new tool names to the tool list.

- [ ] **Step 7: Flip the roadmap doc's Universe row**

  Change row 33 (`| 33 | Universe | 30 | Planned |`) to `| 33 | Universe | 30 | This plan |`.
  Verify with `git diff` that this is the ONLY row changed.

- [ ] **Step 8: Commit**

```bash
git add mcp/universe.go mcp/universe_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add universe MCP tools part C (graphics, structures, system stats, bulk id/name resolution)"
```

- [ ] **Step 9: Push**

```bash
git push origin worktree-esi-access-list-activities
```
