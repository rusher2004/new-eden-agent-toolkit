# `sde/` SQLite Backend Migration — Design

**Status:** Implemented (docs/superpowers/plans/2026-08-04-sde-sqlite-migration.md, 11/11 tasks). Amended by the 2026-11-18 SDE/ESI audit: fuzzwork's dump dropped `invTypes.radius`, so type `Radius` is now sourced from EVE Ref's `reference-data` export (which this spec's `fetchReferenceDataTypeVolumes` used for `PackagedVolume` before PR #16 removed it), via the same best-effort, in-memory pattern; `esi-scrape` was rejected for it because it has been frozen since 2025-11-26. See CLAUDE.md's Hard rules and Known gaps.
Update (2026-08-12): CCP added `PackagedVolume` to the SDE itself
(https://github.com/esi/esi-issues/issues/1499), and fuzzwork's dump picked it up as a plain
`invTypes.packagedVolume` column — the "fields with no home anywhere in the dump" list below is
now down to `AllianceID`/`SofDNA`, and the EVE Ref `reference-data` fetch this spec describes for
`PackagedVolume` (`fetchReferenceDataTypeVolumes`, `typeVolumes`) has been removed from `sde/`.
**Date:** 2026-08-04
**Supersedes:** [`2026-07-27-sde-design.md`](2026-07-27-sde-design.md),
[`2026-07-27-sde-everef-supplemental-sources-design.md`](2026-07-27-sde-everef-supplemental-sources-design.md),
and [`2026-07-27-sde-universe-migration-design.md`](2026-07-27-sde-universe-migration-design.md) —
those specs describe the bbolt + CCP-jsonl-zip + EVE-Ref-reference-data + EVE-Ref-esi-scrape
pipeline this plan replaces. They remain accurate history of what was built and why; this spec
describes what replaces it and why that's now a better trade.

## Goal

Replace `sde/`'s storage/sync pipeline — currently CCP's SDE jsonl zip parsed into `bbolt`,
augmented at sync time from two independently-versioned EVE Ref bulk exports
(`reference-data`, `esi-scrape`) — with a single source: [fuzzwork.co.uk's SQLite SDE
dump](https://www.fuzzwork.co.uk/dump/latest-sqlite.db.gz), a community-maintained,
periodically-refreshed conversion of CCP's SDE into a fully-indexed, directly queryable SQLite
database. `sde/`'s public `Store` API (one typed accessor per resource, `PutX` write-back for
`mcp/`'s ESI-fallback path, nil-safe zero-value semantics) is unchanged — this is an internal
storage/sync rewrite, not an API redesign.

## Why: what triggered this

The user independently downloaded the fuzzwork dump and asked whether it could match `sde/`'s
current name-resolution coverage. Direct inspection of the live file (`sqlite3 eve.db ".schema"`,
row counts, spot-checks against Jita's system) found it not just comparable but **more reliable
for celestial bodies specifically**: `sde/` currently sources planets/moons/asteroid
belts/stars/stargates from EVE Ref's `esi-scrape` mirror, a best-effort augmentation that silently
skips a sync cycle on failure (by design — see the superseded specs). The fuzzwork dump's
`mapDenormalize` table carries the same data (name, position, system, type, keyed by `groupID`:
6=Sun, 7=Planet, 8=Moon, 9=Asteroid Belt, 10=Stargate) sourced from CCP's SDE itself, and
`mapJumps` (`stargateID → destinationID`) covers stargate destinations the same way. Trading three
independently-versioned sources for one, while *improving* reliability on the weakest link, is a
straightforward win.

## Key findings verified against the live file

- **Schema**: 495 MB uncompressed, ~150 tables. `invTypes`/`invGroups`/`invCategories`,
  `mapSolarSystems`/`mapRegions`/`mapConstellations`/`staStations`, `chrRaces`/`chrBloodlines`/
  `chrAncestries`/`chrFactions` all map directly onto `sde/`'s existing resource types, one row per
  ID, with indexes already in place on every foreign key `sde/` needs to join on
  (`ix_invTypes_groupID`, `ix_invGroups_categoryID`, `ix_staStations_solarSystemID`,
  `ix_mapSolarSystems_regionID`, `ix_mapSolarSystems_constellationID`, etc.).
- **Celestial bodies**: confirmed via `mapDenormalize` filtered to Jita (`solarSystemID=30000142`)
  — returns the sun, 8 planets, and their moons with correct `itemName`/`typeID`/position in one
  table, no per-category fetch needed. `mapJumps` confirmed to carry stargate destination pairs.
- **`chrFactions`** already has `stationCount`/`stationSystemCount` as precomputed columns — no
  need for `sde/`'s current manual `npcStations` + `npcCorporations` join.
- **Localization**: `trnTranslations` + `trnTranslationLanguages` carries all 8 of CCP's locales,
  keyed by an *undocumented* `tcID` per source column (confirmed `tcID=8` = `invTypes.typeName` by
  cross-referencing "Tritanium"; other tables' `tcID`s are unmapped). **Moot for this migration**:
  a repo-wide grep found every current consumer (`mcp/enrich.go`, `mcp/universe.go`, 18 call
  sites) reads only `Name["en"]`/`Description["en"]` — no tool has ever exposed another locale.
  Reverse-engineering `tcID` for zero current benefit is out of scope (see Non-goals).
- **Fields with no home anywhere in the dump**: race→`AllianceID`, type→`PackagedVolume`,
  graphic→`SofDNA` (checked `eveGraphics`' columns directly — `sofFactionName`/`sofHullName`/
  `sofRaceName`/`sofMaterialSetID`/`graphicFile`, no DNA string). All three currently come from
  EVE Ref's `esi-scrape`/`reference-data` mirrors in the superseded design and stay sourced that
  way here (see "What EVE Ref still supplies" below). `get_universe_graphic`'s `CollisionFile`
  remains unsourced everywhere, unaffected by this migration.
- **New field-drop found while writing the implementation plan**: `mapRegions` has no description
  column at all (`PRAGMA table_info(mapRegions)` checked directly — `regionID`/`regionName`/`x`/
  `y`/`z`/`xMin..zMax`/`factionID`/`nebula`/`radius`, nothing else), and no other table in the dump
  carries region lore text either (`mapLandmarks` is an unrelated concept — named celestial
  landmarks, not regions). Today's `sde/region.go` sources `Region.Description` from CCP's own
  `mapRegions.jsonl`, which does carry it for at least some regions (confirmed via
  `sde/region_test.go`'s fixture: Derelik → "A dangerous frontier region."). This migration drops
  it — `get_universe_region`'s `Description *string` output field goes from "populated for some
  regions" to "always nil" (see the updated Non-goals entry below). Same treatment as
  `CollisionFile`: documented, not silently absorbed.
- **Freshness**: `HEAD https://www.fuzzwork.co.uk/dump/latest-sqlite.db.gz` returns both `ETag`
  and `Last-Modified` — the exact same staleness-check shape `sync.go`'s
  `fetchRemoteVersion` already implements for `reference-data`/`esi-scrape`. Compressed size
  ~170 MB (`Content-Length` confirmed live), decompresses to the 495 MB observed on disk.

## Design

### Storage engine: query SQLite directly, drop bbolt

`sde/` stops using `go.etcd.io/bbolt` entirely. The downloaded dump *is* the cache — it's already
indexed and query-ready, so staging it into a second store (bbolt) would just be a slower copy of
the same data. `Store` wraps a `database/sql.DB` opened against `~/.neat/sde/eve.db` via
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) — a pure-Go driver, preserving the
foundation design's `CGO_ENABLED=0` static-cross-compilation constraint (rules out `mattn/go-
sqlite3`, which requires cgo).

Every accessor becomes a parameterized `SELECT`:

```go
func (s *Store) Type(ctx context.Context, userAgent string, id int64) (Type, bool, error) {
	if s == nil {
		return Type{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var t Type
	err := s.db.QueryRowContext(ctx,
		`SELECT typeID, groupID, typeName, description, published, capacity, graphicID,
		        iconID, marketGroupID, mass, portionSize, radius, volume
		 FROM invTypes WHERE typeID = ?`, id,
	).Scan(&t.ID, &t.GroupID, &t.Name, &t.Description, &t.Published, &t.Capacity, &t.GraphicID,
		&t.IconID, &t.MarketGroupID, &t.Mass, &t.PortionSize, &t.Radius, &t.Volume)
	if errors.Is(err, sql.ErrNoRows) {
		return Type{}, false, nil
	}
	if err != nil {
		return Type{}, false, fmt.Errorf("query type %d: %w", id, err)
	}
	// PackagedVolume/DogmaAttributes/DogmaEffects: see "What EVE Ref still supplies" and
	// the dogma sidecar query below.
	return t, true, nil
}
```

Reverse-relation fields `sync.go` currently precomputes at write time (a group's `TypeIDs`, a
category's `GroupIDs`, a system's `StationIDs`) become `SELECT id FROM child_table WHERE
parent_fk = ?` at read time instead, using the indexes confirmed above — no behavior change, no
precomputation pass needed.

Dogma attributes/effects: confirmed against type 587 (Rifter) — `dgmTypeAttributes(typeID,
attributeID, valueInt, valueFloat)` and `dgmTypeEffects(typeID, effectID, isDefault)` cover it
directly (one row per `(typeID, attributeID)`/`(typeID, effectID)` pair, indexed on `typeID` via
each table's primary key). `valueInt`/`valueFloat` are mutually exclusive per row — `Type` fetches
both and uses whichever is non-null, coalescing to `TypeDogmaAttribute.Value float64`. A second
query against these two tables replaces `loadTypeDogma`'s single in-memory pass, run at read time
instead of write time.

### Write-back: in-memory overlay, not a second write path into `eve.db`

Every `PutX` method (`PutType`, `PutCategory`, `PutGroup`, `PutSystem`, `PutRegion`,
`PutConstellation`, `PutStation`, `PutPlanet`, `PutMoon`, `PutStar`, `PutStargate`, `PutGraphic`,
`PutAsteroidBelt` — 13 in total) exists so `mcp/`'s read-through path can cache an ESI-fetched
result after a cache miss, so a subsequent lookup of that ID is fast. With bbolt this was a plain
key-value write. `eve.db` is a bulk-downloaded artifact, though — writing arbitrary `INSERT`s into
CCP's exact table schema, or having a write vanish under the next resync's atomic-rename (see
"Fetch & freshness" below), is unnecessary complexity for what should be a rare path: the dump
covers nearly every real ID, so a miss mainly means "CCP added this ID after the dump's last
refresh."

`Store` gains a small in-process overlay (e.g. a `sync.Map` keyed by `resource kind + ID`) that
`PutX` writes into and every `GetX` checks *before* falling through to `eve.db`. It does not
survive a process restart — acceptable because `neat mcp` is a long-running server for the
duration of a session, not restarted per tool call; a restart just means the next miss re-fetches
from ESI once more, exactly like a fresh install's first lookup today.

### Struct changes: `Name`/`Description` become plain strings

Every `sde` resource type's `Name map[string]string` / `Description map[string]string` becomes
`Name string` / `Description string`, sourced straight from the table's English column. This
matches the "Localization" finding above exactly — no consumer loses anything, and every call
site simplifies. `mcp/enrich.go` and `mcp/universe.go`'s ~18 occurrences of `.Name["en"]` /
`.Description["en"]` become `.Name` / `.Description` — mechanical, no output-schema change on any
MCP tool.

**Exception: `Region.Description` stays `*string`, not `string`.** Unlike every other resource,
`mapRegions` carries no description column at all (see "New field-drop" above) — a sqlite-served
`Region` never has one. A cache-miss ESI fetch's response still includes it, though, so the
in-memory overlay can still populate it for that process's lifetime, same pattern as
`Graphic.SofDNA`/`CollisionFile`'s hit-vs-miss asymmetry. `nil` on every sqlite-served lookup,
possibly populated on a fresh miss.

### What EVE Ref still supplies

`sync.go` keeps two trimmed-down fetches, reduced to exactly the fields with no home in the
fuzzwork dump:

- `fetchReferenceDataTypeVolumes` (unchanged) — `PackagedVolume` per type from `reference-data`'s
  `types.json`.
- A reduced `fetchEsiScrapeSupplements` — only `races.en-us.yaml` (→ race `AllianceID`) and
  `graphics.yaml` (→ graphic `SofDNA`) survive. The six celestial-body loaders
  (`loadAsteroidBelts`/`loadMoons`/`loadPlanets`/`loadStargates`/`loadStars`/`loadStations`), their
  YAML decode types (`esiScrapeAsteroidBelt`, `esiScrapeMoon`, ...), and `systems.en-us.yaml`
  (→ `systemPlanets`, no longer needed now that `mapDenormalize` carries the system→planet
  relationship natively) are deleted outright.

Both fetches keep today's best-effort semantics: a failure logs a warning and leaves that
augmentation empty for the cycle, never fails the sqlite-dump sync. Unlike today, the three
resulting maps (`typeID → PackagedVolume`, `raceID → AllianceID`, `graphicID → SofDNA`) are never
written into a database at all — they become plain in-memory `Store` fields, set once inside
`sync()` (guarded by the same `syncOnce` every other sync-time value already relies on) and merged
into `Type()`/`Race()`/`Graphic()`'s query results at read time. `eve.db` stays untouched by
augmentation the same way it stays untouched by write-back.

### Fetch & freshness

`sync()`'s shape is unchanged in spirit, narrower in scope:

1. `HEAD` the fuzzwork URL, compare `ETag` against a cached version. bbolt's `_meta` bucket goes
   away with bbolt itself; its replacement is a flat `~/.neat/sde/meta.json` holding the three
   version strings (sqlite dump, reference-data, esi-scrape) — simplest option that avoids opening
   a second database handle just to track three strings, and (unlike a table inside `eve.db`)
   survives untouched across the atomic rename in step 2.
2. On change: download the ~170 MB gzip (`fetchBytes`, unchanged helper), decompress
   (`compress/gzip`, stdlib) to a temp path under `~/.neat/sde/`, then atomically rename over
   `eve.db` — never leaves a partially-written db in place if the process dies mid-download.
3. Fetch `reference-data`/`esi-scrape` unconditionally on every `sync()` call, with no `HEAD`-based
   gating at all -- unlike the fuzzwork dump, neither result is ever persisted to disk (both live
   only as process-lifetime in-memory `Store` fields), so there is nothing on disk for a version
   check to compare against or a gate to usefully skip. This trades a per-launch bandwidth cost
   (~14MB reference-data + ~77MB esi-scrape, ~91MB combined) for correctness -- the alternative
   (caching each fetch's own ETag in `meta.json` to skip a re-fetch) was considered and rejected
   as unnecessary complexity for two narrow, best-effort augmentations.
4. `schemaVersion` bumps (marks the cache *shape* change, same purpose as always) — an install
   that synced under the old bbolt schema detects the mismatch and does a full resync.

The old `~/.neat/sde/sde.db` (bbolt) file is simply orphaned by this migration — no migration
code deletes it. Documented as a known, low-priority cleanup gap (a user can delete it by hand;
it's inert once `neat` stops opening it).

### Testing

`sde/`'s `seedStore(t)` test helper moves from "open a temp bbolt file, `putRecord` raw JSON into
a bucket" to "open a temp-file SQLite db (`t.TempDir()` + `modernc.org/sqlite`), run the fuzzwork
schema's `CREATE TABLE` for whichever tables that test file touches, `INSERT` fixture rows."
Assertions simplify alongside the struct change (`got.Name != "Rifter"` instead of
`got.Name["en"] != "Rifter"`). `creds/`'s "never hit the real backing store in tests" principle
carries over unchanged — no test opens the real `~/.neat/sde/eve.db`.

## Non-goals (explicitly out of scope)

- **Multi-locale support.** Nothing consumes it today; reverse-engineering `trnTranslations`'
  `tcID` mapping is deferred until a real use case exists. If one shows up, `Name`/`Description`
  would need to become `map[string]string` again — a documented, deliberate future break from
  this spec's "plain string" decision, not an oversight.
- **`get_universe_graphic`'s `CollisionFile`.** Confirmed absent from `eveGraphics` too. No change
  from the status quo (unsourced everywhere).
- **`get_universe_region`'s `Description`.** A genuine regression introduced by this migration
  (unlike `CollisionFile`, which was already unsourced before this spec): `mapRegions` has no
  description column, so every sqlite-served region lookup returns `nil` where the bbolt-backed
  design could return lore text for at least some regions. Accepted rather than solved with a
  third EVE Ref fetch — region descriptions are flavor text, not used by any other tool, and
  reintroducing a fetch for one optional field on one tool isn't worth the complexity this
  migration is removing elsewhere. See "Exception: `Region.Description` stays `*string`" above for
  the one partial mitigation (a fresh cache-miss ESI fetch still populates it in-memory).
- **`get_universe_station`'s `Services`.** Found during a complete field-by-field audit of every
  `sde` struct against the live schema (done after the two gaps above, to catch the rest in one
  pass rather than one at a time): `staOperationServices` — the join table that would map a
  station's `operationID` to its list of service names via `staServices` — has **zero rows** in
  the dump (`SELECT COUNT(*) FROM staOperationServices` confirmed 0, despite `staServices` itself
  having all 27 defined service names). `Services []string` becomes permanently empty on every
  sqlite-served lookup; a fresh cache-miss ESI fetch still populates it in-memory, same mitigation
  pattern as `Region.Description`.
- **`get_universe_faction`'s `IsUnique`** (surfaced on `list_universe_factions`, there is no
  separate get tool). `chrFactions` has no `uniqueName` column or equivalent (`PRAGMA
  table_info(chrFactions)` checked directly). Lower-impact than the other two: every real EVE
  faction name is in fact unique, so this field's practical value was already close to constant
  `true` for every row that path could return. Becomes Go's zero value (`false`) on every
  sqlite-served lookup — a behavior change worth documenting even though no known caller branches
  on it today.

  Same-audit confirmation that nothing else regresses: every other field on every other `sde`
  struct (`Group`, `Category`, `Constellation`, `System`, `Planet`/`Moon`/`AsteroidBelt`/`Star`/
  `Stargate`, `Race`, `Bloodline`, `Ancestry`) has a direct column or join-derivable source in the
  dump. `Station.RaceID` is the one field worth calling out as *derived* rather than direct: the
  fuzzwork schema has no per-station race column, so it's sourced via
  `invTypes.raceID` joined on the station's own `stationTypeID` — the same value ESI itself almost
  certainly derives it from for NPC stations, but a join, not a stored fact, unlike today's direct
  esi-scrape field.
- **The 6 permanently-ESI-direct Universe tools** (`list_universe_structures`/
  `get_universe_structure`, `list_universe_system_jumps`/`_kills`, `resolve_universe_ids`/
  `_names`) — untouched. They resolve live/non-cacheable/arbitrary-input data by design, not an
  SDE sourcing gap.
- **Deleting the orphaned old bbolt file.** Left in place; see "Fetch & freshness" above.

## Follow-up doc/rule updates required (same commit/PR, per this repo's CLAUDE.md)

- `CLAUDE.md`: rewrite the `sde/` package-table row and the "Hard rules" SDE-sourcing paragraph
  (currently describes the bbolt + 3-source pipeline) to describe the sqlite-backed design.
  `go.mod`'s new `modernc.org/sqlite` dependency and dropped `go.etcd.io/bbolt` dependency are
  worth a one-line mention in Conventions if either dependency choice needs justification later.
- `README.md`: update the architecture section's `sde/` description to match.
- Status lines on the three superseded specs (linked at the top of this doc) — add a one-line
  pointer to this spec, per this repo's existing convention of not rewriting historical specs.
