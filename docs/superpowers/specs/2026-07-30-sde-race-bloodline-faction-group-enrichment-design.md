# SDE-backed race/bloodline/faction/group-ID enrichment — Design

## Goal

Batch 1 (`docs/superpowers/specs/2026-07-28-sde-type-id-enrichment-design.md`, PR #11) resolved
`type_id`-like fields. Batch 2 (`docs/superpowers/specs/2026-07-30-sde-location-id-enrichment-design.md`,
PR #12) resolved six location-entity ID kinds. Both specs' own follow-up lists named
`race_id`/`bloodline_id`/`faction_id`/`group_id` as the next batch. This is that batch: add
SDE-resolved `*Name` sibling fields for these four ID kinds across the non-Universe `mcp/*.go`
tools, following the exact additive pattern batches 1-2 established.

## Inventory: what actually needs enriching

A full read-only audit of `mcp/*.go` (excluding `universe.go`, which already resolves these four
entities for its own `list_universe_races`/`list_universe_bloodlines`/`list_universe_factions`/
`list_universe_groups` tools via direct `sde.Store` bulk/ID access, and `enrich.go`, the shared
helper itself) found:

- **8 files** carry at least one genuine (non-echoed, ESI-response-derived) occurrence of one of
  the four ID kinds: `alliance.go`, `character.go`, `corporation.go`, `corporation_projects.go`,
  `faction_warfare.go`, `incursions.go`, `killmails.go`, `sovereignty.go`.
- **21 genuine struct-field occurrences**: FactionID 18 (across 8 files), BloodlineID 1
  (`character.go`), RaceID 1 (`character.go`), GroupID 1 (`corporation_projects.go`).
- Mixed typing that must be preserved per-site, not assumed uniform: most `FactionID` occurrences
  are `*int64` (nil-guard both collect and assign, same convention as batch 2's oneOf fields), but
  `FwFactionStats.FactionID`, `FwSystem.OccupierFactionID`/`.OwnerFactionID`, `FwWar.FactionID`, and
  `Incursion.FactionID` are plain `int64` (resolve unconditionally, no guard needed).
  `GetCharacterOutput.BloodlineID`/`.RaceID` are also plain `int64` — ESI's character-detail
  response always includes both.
- No parallel-array (`[]int64`) occurrences of any of these four ID kinds anywhere in the audited
  files — unlike batch 2, this batch has no index-aligned-list exception to handle.
- `corporation_projects.go`'s `getCorporationProjectDetailHandler` already has batch-1/2
  enrichment blocks (`ShipTypeMatcher.TypeName`, `LocationMatcher`'s four location-name siblings,
  `DockingLocationMatcher.StationName`) — this batch adds `ShipTypeMatcher.GroupName`,
  `IdentityMatcher.FactionName`, and `FactionMatcher.FactionName` as additional, separate passes
  in that same handler, not merged into the existing loops.
- No occurrence in this batch already carries an ESI-supplied name alongside its ID (unlike batch
  2's `freelance_jobs.go` exclusion) — every one of the 21 occurrences is a bare ID with no sibling
  name field today.

## Explicitly out of scope / excluded

- **ESI market groups are a different concept entirely.** `market.go`'s `GetMarketGroupOutput`/
  `ListMarketGroupsOutput` (`MarketGroupID`/`ParentGroupID`) and `sde/type.go`'s `Type.MarketGroupID`
  refer to ESI's market-group hierarchy (the pricing/browsing tree), which is unrelated to
  `sde.Group` (the item/ship group tier between Category and Type, e.g. "Frigate"). This batch's
  `group_id` enrichment applies only to `sde.Group`-shaped fields — confirmed the only such field
  in non-Universe files is `corporation_projects.go`'s `ShipTypeMatcher.GroupID`. Market-group
  fields are untouched.
- Confirmed no request/input-echo fields named race/bloodline/faction/group anywhere in this
  batch's affected files' `*Input` structs — nothing to exclude on that basis, unlike batch 1's
  `RegionID` input-echo exclusions.

## Architecture

### `sde/` additions

`sde.Store` already has a single-ID `Group(ctx, userAgent, id int64) (Group, bool, error)`
accessor (used by `mcp/universe.go`'s `get_universe_group`), structurally identical to batch 1/2's
`Type`/`System`/etc. accessors. `Race`, `Bloodline`, and `Faction` currently only have bulk
`Races(ctx, userAgent) ([]Race, error)`/`Bloodlines(...)`/`Factions(...)` accessors (used by
`list_universe_races`/etc.'s read-through). Add single-ID `Store.Race`/`Store.Bloodline`/
`Store.Faction` accessors to `sde/race.go`/`sde/bloodline.go`/`sde/faction.go`, each mirroring
`sde/group.go`'s `Group()` method exactly: `func (s *Store) X(ctx context.Context, userAgent
string, id int64) (X, bool, error)`, reading from the same `int64Key(id)`-keyed bbolt bucket the
existing bulk loaders (`loadRaces`/`loadBloodlines`/`loadFactions` in `sync.go`) already populate.
**No `sync.go` changes are needed** — the buckets already exist and are already keyed by ID; this
is a pure read-accessor addition, not a new sync source.

All four entities (`Race`, `Bloodline`, `Faction`, `Group`) have `Name map[string]string`
(localized, read via `.Name["en"]`) — confirmed by reading each struct definition. No plain-string
asymmetry in this batch, unlike batch 2's Station/Planet/Moon split.

### `mcp/enrich.go` additions

Four new resolvers — `resolveRaceNames`, `resolveBloodlineNames`, `resolveFactionNames`,
`resolveGroupNames` — each a package-level `var` with its own narrow one-method interface
(`raceNameCache`, `bloodlineNameCache`, `factionNameCache`, `groupNameCache`), following
`resolveTypeNames`'s exact contract: dedup input IDs, drop a miss or error from the result map
rather than erroring, no live-ESI fallback. All four read `.Name["en"]`, so they share one shape
(unlike batch 2's two-shape split) — still four separate small functions, not one generic
abstraction, for the same reason batch 1/2 rejected a generic resolver: Go can't express "generic
over these four types' `Name` field" without reflection, and four ~15-line functions cost less
than the abstraction would.

### Wiring: 21 occurrences, additive `*Name` siblings

Every `FooID int64`/`*int64` field gains a sibling `FooName string` (`json:"...,omitempty"`),
suffix-swap naming: `FactionID`→`FactionName`, `RaceID`→`RaceName`, `BloodlineID`→`BloodlineName`,
`GroupID`→`GroupName`. Nil-guarded on both collect and assign for every `*int64` site (most
`FactionID` occurrences, `ShipTypeMatcher.GroupID`, `IdentityMatcher.FactionID`,
`FactionMatcher.FactionID`); resolved unconditionally for the five plain-`int64` sites
(`FwFactionStats.FactionID`, `FwSystem.OccupierFactionID`/`.OwnerFactionID`, `FwWar.FactionID`,
`Incursion.FactionID`, `GetCharacterOutput.BloodlineID`/`.RaceID`).

`corporation_projects.go`'s three affected fields (`ShipTypeMatcher.GroupID`,
`IdentityMatcher.FactionID`, `FactionMatcher.FactionID`) all live in
`getCorporationProjectDetailHandler`, which already has `sdeStore` and existing enrichment blocks
from batches 1-2. Add a new, separate enrichment pass for `GroupName`/`FactionName` after the
existing blocks, not merged into them — same "additive pass, don't touch prior loops" convention
established in batch 2's `corporation.go`/`market.go`/etc. tasks.

## Performance

Same reasoning as batches 1-2: all four entity tables are part of the same bulk sync as
Type/System/etc., so lookups are fast in-process bbolt reads, not network calls. No live ESI
fallback on a miss, for the same rate-limit-safety reason established previously. Cardinality here
is much lower than batch 2's `list_market_orders`/`list_sovereignty_systems` concerns — none of
the 21 occurrences appear in an unbounded-cardinality list; `killmails.go`'s attacker/victim lists
are the largest, and killmails have a natural small upper bound per kill.

## Testing

- Four new resolver functions in `mcp/enrich_internal_test.go`, each with the same four cases as
  `resolveTypeNames`'s own tests (dedup, miss, error, empty input).
- Each of the 8 affected files gets its handler(s) updated with a `*Name`-asserting test using the
  established resolver-var-override pattern (pass `nil` for `*sde.Store`, override the relevant
  resolver var). Nil-guarded fields get an explicit nil-value test case, not just a happy-path one.
- `corporation_projects.go`'s `getCorporationProjectDetailHandler` test gets extended (not
  replaced) to also assert `ShipTypeMatcher.GroupName`, `IdentityMatcher.FactionName`, and
  `FactionMatcher.FactionName` resolve correctly, following the same oneOf-fixture pattern already
  used there for `TypeName`/location names.
- New `sde/race_test.go`/`sde/bloodline_test.go`/`sde/faction_test.go` (or additions to existing
  test files if already present) cover the new single-ID accessors, mirroring `sde/group_test.go`'s
  existing `Group()` test coverage (hit, miss, error cases).

## Docs sync

- `docs/reference/esi-coverage-roadmap.md`'s Conventions section note gets a third sentence noting
  race/bloodline/faction/group IDs are now also resolved, closing out the last of batch 1's
  originally-flagged follow-up list.
- `CLAUDE.md`'s and `README.md`'s `mcp/` package-table `enrich.go` clause both get extended to
  mention the four new resolvers, keeping both in sync per this repo's own convention.
- `VERSION` bump (minor — additive feature) once the batch lands, mirroring 0.2.0→0.3.0.

## Explicitly out of scope (future work)

- The generic ambiguous `location_id` fields (NPC station vs. player-owned structure), deferred by
  batch 2, remain deferred — unaffected by this batch.
- With this batch, all four ID kinds named in batch 1's original follow-up list
  (`race_id`/`bloodline_id`/`faction_id`/`group_id`) will be resolved. No further batches are
  currently flagged beyond the `location_id` disambiguation above.
- `mcp/search.go`'s `SearchResults.Faction []int64` is a bare faction-ID list that was never
  enriched in batch 1, 2, or this batch, even though the same struct already has
  `InventoryTypeNames []string` from batch 1's parallel-array pattern. Identified during this
  batch's final whole-branch review as a pre-existing gap predating this batch; intentionally left
  alone here. Worth picking up in a future batch alongside the deferred `location_id`
  disambiguation above.
