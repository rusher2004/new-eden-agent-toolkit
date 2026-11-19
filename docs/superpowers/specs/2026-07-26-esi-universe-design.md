# ESI Universe Design (33rd batch)

**Status:** approved (autonomous sweep). Written directly by the orchestrator, no subagent
dispatch, per the user's standing instruction after the session's subagent spawn cap was hit
during the Sovereignty batch.

## Scope

FC's "Universe" tag -- the largest remaining tag, 30 endpoints, all verified directly against
`github.com/fnt-eve/goesi-openapi@v0.0.37`'s `esi/api_universe.go` and `esi/api/openapi.yaml`.

| # | Endpoint | operationId | Security | Paginated |
|---|---|---|---|---|
| 1 | `GET /universe/ancestries` | `GetUniverseAncestries` | public | No |
| 2 | `GET /universe/asteroid_belts/{id}` | `GetUniverseAsteroidBeltsAsteroidBeltId` | public | No |
| 3 | `GET /universe/bloodlines` | `GetUniverseBloodlines` | public | No |
| 4 | `GET /universe/categories` | `GetUniverseCategories` | public | No |
| 5 | `GET /universe/categories/{id}` | `GetUniverseCategoriesCategoryId` | public | No |
| 6 | `GET /universe/constellations` | `GetUniverseConstellations` | public | No |
| 7 | `GET /universe/constellations/{id}` | `GetUniverseConstellationsConstellationId` | public | No |
| 8 | `GET /universe/factions` | `GetUniverseFactions` | public | No |
| 9 | `GET /universe/graphics` | `GetUniverseGraphics` | public | No |
| 10 | `GET /universe/graphics/{id}` | `GetUniverseGraphicsGraphicId` | public | No |
| 11 | `GET /universe/groups` | `GetUniverseGroups` | public | **Yes** |
| 12 | `GET /universe/groups/{id}` | `GetUniverseGroupsGroupId` | public | No |
| 13 | `POST /universe/ids` | `PostUniverseIds` | public | No |
| 14 | `GET /universe/moons/{id}` | `GetUniverseMoonsMoonId` | public | No |
| 15 | `POST /universe/names` | `PostUniverseNames` | public | No |
| 16 | `GET /universe/planets/{id}` | `GetUniversePlanetsPlanetId` | public | No |
| 17 | `GET /universe/races` | `GetUniverseRaces` | public | No |
| 18 | `GET /universe/regions` | `GetUniverseRegions` | public | No |
| 19 | `GET /universe/regions/{id}` | `GetUniverseRegionsRegionId` | public | No |
| 20 | `GET /universe/stargates/{id}` | `GetUniverseStargatesStargateId` | public | No |
| 21 | `GET /universe/stars/{id}` | `GetUniverseStarsStarId` | public | No |
| 22 | `GET /universe/stations/{id}` | `GetUniverseStationsStationId` | public | No |
| 23 | `GET /universe/structures` | `GetUniverseStructures` | public | No |
| 24 | `GET /universe/structures/{id}` | `GetUniverseStructuresStructureId` | **private** `esi-universe.read_structures.v1` | No |
| 25 | `GET /universe/system_jumps` | `GetUniverseSystemJumps` | public | No |
| 26 | `GET /universe/system_kills` | `GetUniverseSystemKills` | public | No |
| 27 | `GET /universe/systems` | `GetUniverseSystems` | public | No |
| 28 | `GET /universe/systems/{id}` | `GetUniverseSystemsSystemId` | public | No |
| 29 | `GET /universe/types` | `GetUniverseTypes` | public | **Yes** |
| 30 | `GET /universe/types/{id}` | `GetUniverseTypesTypeId` | public | No |

Only endpoint 24 (`get_universe_structure`) requires auth, and only endpoints 11 and 29 (Groups,
Types lists) are paginated -- both verified individually per endpoint, not assumed from tag
similarity.

**Structure detail auth pattern**: mirrors `mcp/market.go`'s `list_structure_market_orders`
exactly -- the querying character authenticates via OAuth2 scope
`esi-universe.read_structures.v1`, but the structure itself (not the character) is the resource;
ESI's own 403 is the sole authorization gate (the character needs docking access), no additional
role check in this facade. This is the ONLY private tool in this entire batch; every other tool
uses `esi.NewPublicClient`.

## Nested model verification (non-recursion check)

Every nested/multi-field model in this batch was read directly from generated source and confirmed
non-recursive:

- **Position** (`CharactersCharacterIdAssetsLocationsPostInnerPosition{X,Y,Z float64}`, already
  used by the Assets batch) is reused by AsteroidBelt/Moon/Planet/Constellation/System detail
  responses. `UniverseStructuresStructureIdGetPosition{X,Y,Z float64}` is a separate-but-identical
  type used only by the structure detail response (optional, `omitempty`).
- `UniverseStargatesStargateIdGetDestination{StargateId, SystemId int64}` -- flat.
- `UniverseSystemsSystemIdGetPlanetsInner{AsteroidBelts []int64, Moons []int64, PlanetId int64}`
  -- flat, slices of scalars only.
- `UniverseTypesTypeIdGet`'s `DogmaAttributes`/`DogmaEffects` fields reuse
  `DogmaDynamicItemsTypeIdItemIdGetDogmaAttributesInner{AttributeId int64, Value float64}` and
  `...DogmaEffectsInner{EffectId int64, IsDefault bool}` -- both flat, and both already have MCP
  equivalents (`DynamicItemAttribute`, `DynamicItemEffect` in `mcp/dogma.go`) to reuse rather than
  redeclare.
- `UniverseIdsPost`'s 10 category fields (Agents, Alliances, Characters, Constellations,
  Corporations, Factions, InventoryTypes, Regions, Stations, Systems) are ALL the identical shape
  `{Id *int64, Name *string}` -- verified individually for every one of the 10 generated inner
  types, not assumed from the first. Flattens to one shared MCP type (`IDNamePair`), reusing the
  shared-type-for-near-identical-models pattern.
- `UniverseNamesPostInner{Category string, Id int64, Name string}` -- flat.

No recursion anywhere in this batch.

## esi/universe.go -- 30 methods

All methods use `esi.NewPublicClient` except `UniverseStructureDetail`, which uses the
per-character authenticated client (mirrors `esi/market.go`'s `StructureMarketOrders`). Every
method returns exactly what ESI returns, zero name-resolution enrichment (per the project's
standing convention -- this is notable here because several of these endpoints, e.g. `types/{id}`,
`ids`, `names`, exist specifically to help a caller resolve names, but the facade still returns
ESI's raw shape rather than pre-joining anything further).

Representative signatures (grouped by the task split below):

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

func (c *Client) UniverseGraphics(ctx context.Context) ([]int64, error)
func (c *Client) UniverseGraphicDetail(ctx context.Context, graphicID int64) (*esiapi.UniverseGraphicsGraphicIdGet, error)
func (c *Client) UniverseStructures(ctx context.Context) ([]int64, error)
func (c *Client) UniverseStructureDetail(ctx context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error) // private, needs characterID baked into Client
func (c *Client) UniverseSystemJumps(ctx context.Context) ([]esiapi.UniverseSystemJumpsGetInner, error)
func (c *Client) UniverseSystemKills(ctx context.Context) ([]esiapi.UniverseSystemKillsGetInner, error)
func (c *Client) UniverseIDsByName(ctx context.Context, names []string) (*esiapi.UniverseIdsPost, error)
func (c *Client) UniverseNamesByID(ctx context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error)
```

`UniverseStructureDetail` is the only method on a `*Client` built via `esi.NewClient` (private);
every other method in this file is called on a client built via `esi.NewPublicClient`.

## mcp/universe.go -- 30 tools

Naming: `list_universe_ancestries`, `list_universe_bloodlines`, `list_universe_races`,
`list_universe_factions`, `list_universe_categories`, `get_universe_category`,
`list_universe_groups`, `get_universe_group`, `list_universe_types`, `get_universe_type`,
`list_universe_regions`, `get_universe_region`, `list_universe_constellations`,
`get_universe_constellation`, `list_universe_systems`, `get_universe_system`,
`get_universe_asteroid_belt`, `get_universe_moon`, `get_universe_planet`,
`get_universe_stargate`, `get_universe_star`, `get_universe_station`,
`list_universe_graphics`, `get_universe_graphic`, `list_universe_structures`,
`get_universe_structure` (private), `list_universe_system_jumps`, `list_universe_system_kills`,
`resolve_universe_ids`, `resolve_universe_names`.

Shared flattened types (defined once in `mcp/universe.go`, reused across tools in that file):

```go
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type IDNamePair struct {
	ID   *int64  `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}
```

`get_universe_type` reuses `mcp/dogma.go`'s existing `DynamicItemAttribute`/`DynamicItemEffect`
types for its `DogmaAttributes`/`DogmaEffects` fields -- do not redeclare them.

`resolve_universe_ids` (wraps `PostUniverseIds`, input `{Names []string}`) returns one
`IDNamePair` slice per category:
```go
type ResolveUniverseIDsOutput struct {
	Agents         []IDNamePair `json:"agents,omitempty"`
	Alliances      []IDNamePair `json:"alliances,omitempty"`
	Characters     []IDNamePair `json:"characters,omitempty"`
	Constellations []IDNamePair `json:"constellations,omitempty"`
	Corporations   []IDNamePair `json:"corporations,omitempty"`
	Factions       []IDNamePair `json:"factions,omitempty"`
	InventoryTypes []IDNamePair `json:"inventoryTypes,omitempty"`
	Regions        []IDNamePair `json:"regions,omitempty"`
	Stations       []IDNamePair `json:"stations,omitempty"`
	Systems        []IDNamePair `json:"systems,omitempty"`
}
```

`resolve_universe_names` (wraps `PostUniverseNames`, input `{IDs []int64}`) returns:
```go
type ResolvedName struct {
	Category string `json:"category"`
	ID       int64  `json:"id"`
	Name     string `json:"name"`
}
```

`get_universe_structure` is the only private tool -- follows `list_structure_market_orders`'s
exact pattern: input `{CharacterID int32, StructureID int64}`, `characterExists` gate, no separate
owner/corp field (the structure ID is the resource, ESI's 403 is the sole gate).

All list-of-bare-IDs endpoints (Categories, Groups, Types, Regions, Constellations, Systems,
Graphics, Structures) return `{IDs []int64}`-shaped output structs, following the existing
`list_dogma_attributes`/`list_dogma_effects` precedent (bare `[]int64` passthrough, no
enrichment).

## Testing approach

Same as every prior batch: `esi/universe_test.go` uses `httptest.Server` + raw JSON response
bodies, covering both the 200 success path and the HTTP-error-wrapped-through-`describeError` path
per method (60 tests: 30 endpoints x 2), following the exact
`esi.ExportedNewPublicClientWithOptions`/`esi.ExportedNewClientWithOptions` +
`goesi.ClientOptions{BaseURL, UserAgent}` pattern used throughout this project (see
`esi/sovereignty_test.go` for the public variant, `esi/skills_test.go` for the private variant).
Groups/Types pagination tests additionally assert `X-Pages` header parsing via
`totalPagesFromHeader`, mirroring `esi/planetary_interaction_test.go`'s
`CorporationCustomsOffices` precedent.

`mcp/universe_internal_test.go` swaps each package-level var for a fake per the established
internal-test pattern; for `resolve_universe_ids`/`resolve_universe_names`, include a "sparse"
fixture (only some categories populated) alongside a "full" fixture to confirm no cross-category
bleed, mirroring the Sovereignty/Structures batches' claim/transport-variant test style. All 30
handlers get a MapsFields test + a PropagatesError test; `get_universe_structure` additionally
gets an UnknownCharacter test (the only private tool in this batch).

## Task breakdown

Given this batch's unusual size (30 endpoints -- the largest in the sweep), it is split into 6
tasks instead of the usual 2 (esi/mcp), each committing incrementally to the single
`esi/universe.go`/`mcp/universe.go` files per the project's one-file-per-tag convention:

- **Task 1** (`esi/universe.go` part A): `UniverseAncestries`, `UniverseBloodlines`,
  `UniverseRaces`, `UniverseFactions`, `UniverseCategories`, `UniverseCategoryDetail`,
  `UniverseGroups` (paginated), `UniverseGroupDetail`, `UniverseTypes` (paginated),
  `UniverseTypeDetail` -- 10 methods, all public reference-data lookups. Plus
  `esi/universe_test.go` covering these 10.
- **Task 2** (`esi/universe.go` part B): `UniverseRegions`, `UniverseRegionDetail`,
  `UniverseConstellations`, `UniverseConstellationDetail`, `UniverseSystems`,
  `UniverseSystemDetail`, `UniverseAsteroidBeltDetail`, `UniverseMoonDetail`,
  `UniversePlanetDetail`, `UniverseStargateDetail`, `UniverseStarDetail`,
  `UniverseStationDetail` -- 12 methods, geography + celestial detail lookups. Appends to the same
  two files from Task 1.
- **Task 3** (`esi/universe.go` part C): `UniverseGraphics`, `UniverseGraphicDetail`,
  `UniverseStructures`, `UniverseStructureDetail` (private), `UniverseSystemJumps`,
  `UniverseSystemKills`, `UniverseIDsByName`, `UniverseNamesByID` -- 8 methods, closing out the
  esi/ layer. Appends to the same two files.
- **Task 4** (`mcp/universe.go` part A): the 10 tools for Task 1's methods + `Position`/
  `IDNamePair` shared type declarations (even though Position isn't used until Task 5, declaring it
  once here avoids a mid-batch redeclaration conflict -- IDNamePair is used starting Task 6, same
  reasoning) + partial `mcp/server.go` registration for these 10 tools.
- **Task 5** (`mcp/universe.go` part B): the 12 tools for Task 2's methods + `mcp/server.go`
  registration for these 12.
- **Task 6** (`mcp/universe.go` part C): the 8 tools for Task 3's methods + `mcp/server.go`
  registration for these 8 + `mcp/server_test.go`'s full 30-tool expected-tools-list update (179 ->
  209 tools) + docs sync (`CLAUDE.md`/`README.md` tool count and list, roadmap doc row 33 flipped
  to "This plan").

Every task done directly by the orchestrator (no subagent dispatch), verified independently with
`go build`/`go vet`/`gofmt -l`/targeted `go test -v`, with a full `go test ./... -race` at the end
of Task 3 (esi/ layer complete) and again at the end of Task 6 (mcp/ layer complete, before commit
and push).
