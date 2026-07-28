# Closing the Remaining Universe-Tag Gaps with EVE Ref's Supplemental Datasets — Design

## Goal

The just-shipped `sde-universe-migration` (PR #9) migrated 18 of 24 Universe-tag tools onto a
read-through cache backed by CCP's official Static Data Export, leaving 6 celestial-detail tools
permanently ESI-direct and 4 fields dropped on cache hits (`PackagedVolume`, `AllianceID`,
`Planets`, `CollisionFile`/`SofDNA`) — all because the *official* SDE genuinely doesn't carry that
data. This plan closes all of that except one field, using two more datasets EVE Ref hosts at
`data.everef.net`, verified directly against real downloaded data during design:

- **`reference-data`** (`data.everef.net/reference-data/`) — CCP's SDE, an ESI scrape, and a
  Hoboleaks mirror pre-merged into one bundle. Its `types.json` carries `packaged_volume` with
  real values (confirmed: type 587 "Rifter" → `2500`).
- **`esi-scrape`** (`data.everef.net/esi-scrape/`) — a full mirror of ESI's own live `universe/*`
  responses. Confirmed by direct inspection: real station/moon/planet/stargate/star/asteroid-belt
  records with names, positions, and full field sets matching our existing MCP types almost
  exactly; `races.en-us.yaml` carries `alliance_id`; `systems.en-us.yaml` carries nested
  `planets[].moons[]`/`asteroid_belts[]`; `graphics.yaml` carries `sof_dna`. Celestial names were
  verified identical across every locale (`en-us` vs `ja` for the same moon ID) — they aren't
  actually translated, so only the `en-us` copy of each locale-suffixed file is needed.

**Confirmed unfillable, not merely unsourced:** `get_universe_graphic`'s `CollisionFile` — zero
non-null values across the *entire* `esi-scrape` graphics file (every graphic, every record). ESI
itself never actually populates this field in practice, not just CCP's static export. This stays
dropped permanently, with that verification noted in the field-drop table rather than left
looking like an oversight.

**End state:** all 24 of 24 Universe-tag tools become `sde/`-backed via read-through caching (up
from 18/24); the field-drop table shrinks to exactly one entry (`CollisionFile`).

## Scope (confirmed)

Full closure: migrate all 6 celestial-detail tools, and close all 3 fillable field gaps
(`AllianceID`, `Planets`, `SofDNA`; `PackagedVolume` already gets a path via `reference-data`).
`CollisionFile` is explicitly out of scope — there is nothing to source.

## Architecture: three sources feeding one sync

`sde/`'s `sync()` already fetches one archive (the CCP SDE zip) lazily, once per `Store` lifetime,
inside one atomic `db.Update`. This adds two more fetches to the *same* `sync()` call — not a
separate lazy-sync mechanism — because celestial/race/system/graphic data is cheap enough
(combined ~91MB, one-time, background to the first tool call) that eager-with-the-SDE is simpler
than tracking three independent staleness clocks, and it keeps every accessor's `ensureSynced`
call sharing one `sync.Once` as it does today.

**Failure isolation (confirmed):** the core CCP SDE zip fetch stays sync-fatal, exactly as today —
categories/groups/types can't exist at all without it. `reference-data` and `esi-scrape` fetches
are each independently best-effort: a failure (network error, format change, EVE Ref outage) is
caught, logged via `slog.Default().WarnContext(ctx, "sde: failed to fetch <source>, skipping
augmentation", "error", err)`, and the affected map is left `nil`/empty — augmentation is simply
skipped for that sync cycle, not fatal to the rest of `sync()`. A `nil` augmentation map behaves
identically to "no data available yet": the 6 new celestial buckets just don't get created (any
lookup falls through to live ESI, self-healing via the existing `Put*` write-back, same as a
never-synced store today), and `Race.AllianceID`/`System.Planets`/`Graphic.SofDNA` simply stay
unset until the next successful sync retries. This matches the standing "SDE data is enrichment,
not a hard requirement" philosophy `ensureSynced`'s own doc comment already states — extended to
these two new sources rather than treated as a special case.

**`esi-scrape` is a single non-seekable `tar.xz`** — no partial/range extraction is possible, so
the full 77MB downloads regardless of how many of its ~40 member files are actually needed. Only
9 members are read out of the decompressed stream: `universe/{asteroid_belts,moons,planets,
stargates,stars,stations,races,systems}.en-us.yaml` (locale-suffixed but content-identical across
locales, confirmed) and `universe/graphics.yaml` (no locale suffix — no translatable fields).
`reference-data`'s 14MB `tar.xz` is smaller; only `types.json` is read out of its ~19 members.

**New dependency:** `go.yaml.in/yaml/v3` — both new sources ship YAML, and the Go standard library
has no YAML support. The original `gopkg.in/yaml.v3` (`go-yaml/yaml`) was archived as unmaintained
by its author in April 2025; `go.yaml.in/yaml/v3` is the actively-maintained continuation hosted by
the official YAML org (the same dependency `sigs.k8s.io/yaml` itself uses, pushed as recently as
yesterday), and is API-compatible — same package, same `Unmarshal`, same native YAML struct tags —
so no need for a JSON-tag-wrapped alternative like `ghodss/yaml` (itself stale, and still
dependent on the archived `gopkg.in/yaml.v2`).

## Hard-rule framing (confirmed: broaden it)

The "SDE over ESI" hard rule in `CLAUDE.md` reframes to "prefer a cached, statically-refreshed
source over a live per-request ESI call." The `sde/` package keeps its name (it's still primarily
the CCP SDE) but its sync pipeline gains `reference-data` and `esi-scrape` as additional bulk-seed
sources for specific resources. The hard rule's documented exception shrinks from 6 tools to
exactly one field (`get_universe_graphic`'s `CollisionFile`).

## `sde/` package changes

**Six new per-ID resources** (one file each, matching the existing one-file-per-resource
convention), each with a `Store` accessor (`(ctx, userAgent, id) (T, bool, error)`) and a `Put*`
write-back method — the same shape as every Task 1–9 resource, giving these 6 tools the same
read-through pattern as everything else migrated so far:

- `AsteroidBelt{ID int64; Name string; Position Position; SystemID int64}`
- `Moon{ID int64; Name string; Position Position; SystemID int64}`
- `Planet{ID int64; Name string; TypeID int64; Position Position; SystemID int64}`
- `Stargate{ID int64; Name string; TypeID int64; Position Position; SystemID int64; DestinationStargateID int64; DestinationSystemID int64}`
- `Star{ID int64; Age int64; Luminosity float64; Name string; Radius int64; SolarSystemID int64; SpectralClass string; Temperature int64; TypeID int64}`
- `Station{ID int64; Name string; SystemID int64; TypeID int64; Position Position; MaxDockableShipVolume float64; OfficeRentalCost float64; Owner *int64; RaceID *int64; ReprocessingEfficiency float64; ReprocessingStationsTake float64; Services []string}`

All six use plain `string` `Name` fields (not `map[string]string`) — confirmed celestial names
aren't multilingual (identical across every scraped locale), matching ESI's own single-string
contract for these endpoints exactly.

**Three existing resources gain a field:**
- `Race` gains `AllianceID *int64` — sourced from `esi-scrape`'s `races.en-us.yaml`, joined by
  `race_id` at sync time. The official SDE's `races.jsonl` still has no such field; this is purely
  an `esi-scrape` augmentation, the same "stored but not raw" treatment `System.StationIDs` already
  uses for its `npcStations.jsonl`-sourced reverse index.
- `System` gains `Planets []SystemPlanet` (new `SystemPlanet{PlanetID int64; AsteroidBeltIDs
  []int64; MoonIDs []int64}`, mirroring `mcp.UniverseSystemPlanet`'s shape as its own
  `sde`-owned type — `sde/` still doesn't import `mcp/`) — sourced from `esi-scrape`'s
  `systems.en-us.yaml`, joined by `system_id`.
- `Graphic` gains `SofDNA *string` — sourced from `esi-scrape`'s `graphics.yaml`, joined by
  `graphic_id`. `CollisionFile` is deliberately not added — confirmed absent from every record in
  the entire scrape, not merely unobserved in a sample.

**`Type` gains `PackagedVolume *float64`** — the one field on an already-existing resource that
needs a genuinely new field (the original migration deliberately never added it, since the
official SDE has no source for it at all). It's populated at sync time from `reference-data`'s
`types.json`, keyed by `type_id`, via the same kind of merge `loadTypes` already does for
`typeDogma.jsonl` — the "stored but not raw from the *primary* source" pattern `System.StationIDs`
already established, just augmented from a different source than usual.

## `sde/sync.go` changes

Two new best-effort fetch phases run before the existing dependency-ordered load pipeline:

1. `fetchReferenceDataTypeVolumes(ctx, userAgent) map[int64]float64` — fetches and extracts
   `reference-data`'s `types.json`, returns `type_id → packaged_volume`. Returns `nil` (not an
   error) on any failure, after logging a warning.
2. `fetchEsiScrapeSupplements(ctx, userAgent) (esiScrapeData, error)` — fetches and extracts the 9
   needed `esi-scrape` members, parses each with `go.yaml.in/yaml/v3`, returns a struct bundling all the maps
   (`raceAlliances map[int64]int64`, `systemPlanets map[int64][]sdePlanetRef`, `graphicSofDNA
   map[int64]string`) plus the 6 raw celestial record slices ready to write into their own
   buckets. Returns a zero-value `esiScrapeData` (not an error) on any failure, after logging a
   warning — every field empty/nil, so every downstream augmentation and bucket-write becomes a
   no-op, not a crash.

`loadTypes` gains the `reference-data` volume map as an extra parameter (merged the same way
`typeDogma.jsonl`'s map already is). `loadRaces` (currently a `loadResource` passthrough) becomes
a custom loader like `loadSystems`/`loadFactions`, merging in `raceAlliances`. `loadSystems` gains
the `systemPlanets` map as an extra merge. `loadGraphics` (currently a `loadResource` passthrough)
becomes a custom loader merging in `graphicSofDNA`. Six new `loadAsteroidBelts`/`loadMoons`/
`loadPlanets`/`loadStargates`/`loadStars`/`loadStations` functions write straightforward,
unaugmented bucket dumps from the `esi-scrape` slices — no reverse-index computation needed, since
these are per-ID resources with no children of their own that other resources need to look up.

## `mcp/universe.go` changes

**Six tools migrate to the read-through pattern** used by all 18 already-migrated tools:
`get_universe_asteroid_belt`, `get_universe_moon`, `get_universe_planet`, `get_universe_stargate`,
`get_universe_star`, `get_universe_station`. Each gets a narrow cache interface
(`asteroidBeltCache`, `moonCache`, etc.), a swappable var trying `sde.Store` first and falling
back to the existing (unchanged) `esi.Client` method on a miss, writing back via the new `Put*`.
Tool names/inputs/outputs are unchanged.

**Three simplifications** — these currently have "populated only on a cache miss" special-case
code from the last migration; that code is removed now that the field flows through `sde.Type`/
`sde.Race`/`sde.System` normally on every cache hit too:
- `get_universe_type`'s `PackagedVolume` — no longer a local variable captured only in the
  cache-miss branch; reads straight from `t.PackagedVolume` like every other field.
- `list_universe_races`' `AllianceID` — no longer hardcoded to `0` on a cache hit; reads from
  `r.AllianceID` (with the same nil-safe zero-value fallback if a given race predates the
  `esi-scrape` augmentation, i.e. was cached before this migration ran).
- `get_universe_system`'s `Planets` — no longer nil'd out on a cache hit; maps from
  `sys.Planets` the same way the miss-path already builds it from ESI's response.

**`CollisionFile` stays dropped** — the field-drop table note is upgraded from "no SDE source"
to "confirmed absent from ESI's own live responses across the entire archive, not merely the
static export."

## Testing

Mirrors the established pattern exactly:
- `sde/` internal-package tests for the 6 new resources (known-ID/unknown-ID/mismatched-key/
  Put-then-read-round-trip), following `sde/region_test.go`'s shape.
- `sde/sync_test.go`'s fixture server gains fixture `tar.xz` builders for `reference-data` and
  `esi-scrape` (small, hand-built archives with just the needed member files, same spirit as the
  existing 13-file SDE zip fixture), plus new assertions for the augmented fields and a test
  confirming a `reference-data`/`esi-scrape` fetch failure doesn't fail the whole sync (the core
  SDE-sourced buckets still populate).
- `mcp/universe_internal_test.go` gains fake-cache types and cache-hit/cache-miss tests for the 6
  new tools, following the `fakeRegionCache`-style shape exactly, plus updated tests for the 3
  simplified tools confirming their fields now populate on a cache hit too.

## Docs sync

`CLAUDE.md`'s hard rule and `sde/` architecture-table row update to reflect the broadened framing
and the new "24/24 Universe tools, 1 remaining field drop" state. `README.md`'s matching text
updates in parallel, same as the prior migration's Task 10.

## Explicitly out of scope

- `CollisionFile` — confirmed absent from ESI's own data, not just the static export; nothing to
  source.
- Any other ESI tag's data (Dogma, Market groups, Industry, NPC corporations, Blueprints, Skills,
  etc.) that this same table also covers — `reference-data`'s table lists many rows unrelated to
  Universe. Those are real, separate opportunities but a different initiative; this plan stays
  scoped to the Universe tag's already-in-flight migration.
- Any change to `neat setup`, tool names, or tool input/output contracts beyond what's already
  documented as field drops.
