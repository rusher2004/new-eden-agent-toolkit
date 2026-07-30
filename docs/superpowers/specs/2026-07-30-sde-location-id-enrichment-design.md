# SDE-backed location-ID enrichment — Design

## Goal

Batch 1 (`docs/superpowers/specs/2026-07-28-sde-type-id-enrichment-design.md`, merged in PR #11)
resolved every `type_id`-like field across the non-Universe `mcp/*.go` tools to a sibling `*Name`
field, backed by the local SDE cache. That design's own "Explicitly out of scope" section named
location IDs as batch 2. This is that batch, scoped to its SDE-coverable half: `system_id`,
`region_id`, `constellation_id`, `station_id`, `planet_id`, and `moon_id`. The generic, ambiguous
`location_id` fields (which may reference either an NPC station — SDE-coverable — or a
player-owned structure — not SDE data, access-gated, needs a live ESI call) are explicitly **not**
part of this batch; they need their own resolution strategy and are a separate future round.

All six `sde.Store` accessors already exist and are already consumed by `mcp/universe.go`'s
Universe-tag tools (`System`, `Region`, `Constellation`, `Station`, `Planet`, `Moon`, each
`func (s *Store) X(ctx, userAgent, id) (X, bool, error)`, structurally identical to the `Type`
accessor batch 1 already wraps). This batch only wires the *consuming* non-Universe files — no new
`sde/` capability is needed.

## Inventory: what actually needs enriching

A full read-only audit of `mcp/*.go` (excluding `universe.go`, which already handles these six
entities for its own tools, and `enrich.go`, the shared helper itself) found:

- **15 files** carry at least one genuine (non-echoed) occurrence of one of the six ID kinds:
  `activities.go`, `corporation.go`, `corporation_projects.go`, `freelance_jobs.go`, `industry.go`,
  `faction_warfare.go`, `killmails.go`, `fleets.go`, `location.go`, `incursions.go`, `market.go`,
  `planetary_interaction.go`, `routes.go`, `sovereignty.go`, `structures.go`.
- **47 genuine struct-field occurrences**: System 27, Planet 6, Station 5, Region 3, Constellation
  3, Moon 3.
- A meaningful nesting spread: most are flat on a list element or a single-object output, but
  `corporation_projects.go`'s `LocationMatcher` (`SolarSystemID`/`RegionID`/`ConstellationID`, each
  `*int64`) is a **oneOf discriminator** — only one of the three is ever set per element, nested two
  levels inside `Output.Configuration.Locations[]` — the exact same shape batch 1's
  `ShipTypeMatcher` already established a pattern for (nil-guard on both collect and assign, across
  every field in the oneOf). `structures.go`'s sovereignty-hub workforce-transport fields go three
  and four levels deep (`Detail.WorkforceTransport.Configuration.ExportSolarSystemID`,
  `.State.ImportSources[].SolarSystemID`).
- Two fields are legitimately zero for some records depending on scope
  (`MarketOrderBookEntry.SystemID` is zero for structure orders; `IndustryJob.StationID` is zero
  for corporation-scoped jobs) — both already carry `omitempty` JSON tags, confirming CCP's own
  "0 means absent" convention. This needs **no special-case code**: looking up ID `0` against the
  SDE cache is just another cache miss under the existing silent-omit contract, so the resulting
  `*Name` field naturally stays empty for those records without extra guarding.
- Three fields are bare `[]int64` lists rather than one ID per struct — the same shape batch 1
  handled for `SearchResults.InventoryType` and `ListMarketTypesOutput.TypeIDs`: `Incursion.
  InfestedSolarSystemIDs`, `StructureSovereigntyHubTransportConfig.ImportSolarSystemIDs`, and
  `GetRouteOutput.Route` (the computed route path — real response data, not an echoed input,
  despite the field name not ending in `...ID`). Each gets an index-aligned parallel `...Names
  []string` field, consistent with the established pattern.
- High-cardinality lists worth calling out for the no-live-ESI-fallback design constraint (mirrors
  batch 1's `list_market_prices` callout): `list_industry_systems`, `list_market_orders`,
  `list_industry_facilities`, `list_sovereignty_systems` — all public, unbounded by any single
  character's or corporation's own state, potentially thousands of elements per call.
- Explicitly **excluded** as request/input or pure input-echo fields (same principle as batch 1's
  `get_market_history` exclusion — resolving a value the caller already supplied adds nothing):
  `list_public_contracts`' `RegionID` passthrough; `get_market_history`/`list_market_orders`/
  `list_market_types`'s `RegionID` inputs, already echoed back; `get_corporation_starbase`'s input
  `SystemID` (request-only, never appears in the response); `get_route`'s `OriginSystemID`/
  `DestinationSystemID` inputs, echoed back (the computed `Route` path itself, above, is not an
  echo and **is** in scope); `get_character_planet`'s input `PlanetID`, echoed back since ESI's
  planet-detail body doesn't carry its own ID.
- Explicitly **out of scope**, noted only so nothing is silently lost track of: the generic
  `LocationID`-family fields (`AssetItem.LocationID`, `Blueprint.LocationID`, `MemberTracking.
  LocationID`, `CorporationBlueprint.LocationID`, `ContainerLog.LocationID`, `IndustryJob.
  LocationID`/`BlueprintLocationID`/`OutputLocationID`, `MarketOrder.LocationID`/
  `MarketOrderBookEntry.LocationID`, `Contract.EndLocationID`/`StartLocationID`, `JumpClone.
  LocationID`, `GetCharacterClonesOutput.HomeLocationID`) — these are the ambiguous
  station-vs-structure fields deferred to a future round, per the scoping decision above.

## Output shape: additive sibling field (unchanged from batch 1)

Every `FooID int64`/`*int64` field gains a sibling `FooName string` (`json:"...,omitempty"`),
named by the same suffix-swap convention batch 1 established: `SystemID`→`SystemName`,
`SolarSystemID`→`SolarSystemName`, `HomeStationID`→`HomeStationName`, `ExportSolarSystemID`→
`ExportSolarSystemName`, and so on. Purely additive — no existing field changes shape or is
removed.

Three naming exceptions, where the source is a list of IDs rather than one ID per record (same
treatment as batch 1's `InventoryType`/`TypeIDs`): `Incursion.InfestedSolarSystemIDs []int64` gains
`InfestedSolarSystemNames []string`; `StructureSovereigntyHubTransportConfig.
ImportSolarSystemIDs []int64` gains `ImportSolarSystemNames []string`; `GetRouteOutput.
Route []int64` gains `RouteNames []string` — all index-aligned with their source slice.

`corporation_projects.go`'s `LocationMatcher` (`SolarSystemID`/`RegionID`/`ConstellationID`, each
`*int64`, oneOf-style — only one is ever set) and `DockingLocationMatcher.StationID` (`*int64`)
each gain a `string` `*Name` sibling, nil-guarded on both the collect and assign steps, exactly
mirroring batch 1's `ShipTypeMatcher.TypeName` treatment.

## Architecture: six resolvers, same shape as `resolveTypeNames`

`mcp/enrich.go` gains six new package-level `var` resolver functions — `resolveSystemNames`,
`resolveRegionNames`, `resolveConstellationNames`, `resolveStationNames`, `resolvePlanetNames`,
`resolveMoonNames` — each following `resolveTypeNames`'s exact shape (dedup input IDs, look up each
via the corresponding narrow one-method interface, drop a miss or error from the result map rather
than surfacing an error). Each gets its own narrow interface (`systemNameCache`, `regionNameCache`,
`constellationNameCache`, `stationNameCache`, `planetNameCache`, `moonNameCache`), matching
`typeNameCache`'s existing convention of exposing only the single `X(ctx, userAgent, id) (X, bool,
error)` method each resolver needs — not the fuller `sde.Store` surface, and not shared with
`universe.go`'s own per-entity interfaces (`systemCache` etc., which also expose `Put*`/bulk-listing
methods this batch doesn't use).

**One real asymmetry to implement correctly**: `Region`/`Constellation`/`System` have localized
`Name map[string]string` (read via `.Name["en"]`, matching `Type.Name`'s existing treatment);
`Station`/`Planet`/`Moon` have plain `Name string` (server-generated names are never translated —
confirmed directly against the `sde/` struct definitions, not assumed). The three
map-backed resolvers and the three plain-string resolvers are two distinct (but each internally
uniform) shapes — not one generic function parameterized over both, and not six independent
one-off implementations either.

Handlers take the concrete `sdeStore *sde.Store` parameter (not an interface) — the same
convention batch 1 settled on after discovering `resolveTypeNames` had to be a `var`, not a `func`,
specifically so tests can override it (a real `*sde.Store` call triggers a live network sync on
first use, which only `sde/`'s own tests can suppress). Tests pass `nil` for `*sde.Store` (safe —
every `sde.Store` accessor has an explicit nil-receiver guard) and override the relevant resolver
var when they need an actual resolved name.

**Rejected: one generic resolver parameterized by entity type.** Go has no way to express "a
function generic over `System`/`Region`/.../`Moon`'s differently-shaped `Name` field" without
reflection or a wrapper closure per call site — which would cost more than six small, obviously-
correct functions costs. This mirrors batch 1's own rejection of a generic enrichment abstraction
beyond `resolveTypeNames` itself.

## Performance

Same reasoning as batch 1: `sde/`'s System/Region/Constellation/Station/Planet/Moon tables are
part of the same full bulk sync as the Type table, so virtually every lookup here is a fast
in-process bbolt read, not a network call, even for the flagged high-cardinality lists
(`list_industry_systems`, `list_market_orders`, `list_industry_facilities`,
`list_sovereignty_systems`). No live ESI fallback, for the same rate-limit-safety reason
batch 1 established for `resolveTypeNames` — a single miss cascading into thousands of live ESI
calls on a busy trade-hub region's order book would be a stampede.

## Testing

- Six new resolver functions in `mcp/enrich_internal_test.go`, each with the same four cases as
  `resolveTypeNames`'s own tests (dedup, miss, error, empty input) — cheap given the shape is
  identical to an already-proven pattern.
- Each of the 15 affected files gets its handler(s) updated with a `*Name`-asserting test using the
  resolver-var-override pattern established in batch 1 (pass `nil` for `*sde.Store`, override the
  relevant resolver var). Nil-guarded fields (`LocationMatcher`'s three `*int64`s,
  `DockingLocationMatcher.StationID`, `FleetMember.StationID`, `Starbase.MoonID`, etc.) get an
  explicit nil-value test case, not just a happy-path one.
- The three index-aligned-list fields each get a partial-cache-miss test proving positional
  alignment survives a miss — batch 1's own final review flagged this exact gap as worth doing
  proactively for a new field, not discovering it after merge (as happened with
  `InventoryTypeNames`/`TypeNames` in batch 1's post-merge review round).
- `freelance_jobs.go`'s `JobBroadcastLocation.SystemID` needs a quick verification pass against the
  real generated ESI model during implementation (it's currently assigned from a raw field named
  `.Id`, not `.SystemId` — very likely correct per the endpoint's documented schema, but confirm
  directly against the vendored model before wiring in enrichment, the same verification discipline
  batch 1 applied to every generated-model field it touched).

## Docs sync

- `docs/reference/esi-coverage-roadmap.md`'s Conventions section note (already updated once by
  batch 1 to describe `type_id` enrichment) gets a second sentence noting the six location-ID kinds
  are now also resolved, and that the generic `location_id` fields remain the one open follow-up.
- `CLAUDE.md`'s `mcp/` package-table row and `README.md`'s matching row both get their `enrich.go`
  clause extended to mention the six location resolvers, keeping both in sync as this repo's own
  convention requires.
- `VERSION` bump (minor — another additive feature, not a bugfix) once the batch lands, mirroring
  batch 1's 0.1.1→0.2.0 bump.

## Explicitly out of scope (future work)

- The generic ambiguous `location_id` fields (NPC station vs. player-owned structure) — needs a
  live-ESI-fallback strategy for structures (which aren't SDE data and are access-gated), a real
  design question this batch deliberately doesn't answer.
- The rarer `race_id`/`bloodline_id`/`faction_id`/`group_id` fields named in batch 1's own
  follow-up list remain their own future batch, unaffected by this one.

## Implementation status

Shipped as PR #12. The "15 files" inventory count above includes `freelance_jobs.go`, which turned
out during implementation to need **no changes**: `JobBroadcastLocation.SystemID` is already paired
with a `Name` field ESI populates directly, so enriching it via `sde/` would just duplicate data
ESI already provides for free. The actual shipped count is **14 modified files**, consistent
everywhere else (CLAUDE.md, README.md, the CHANGELOG, and the roadmap doc all say 14).
