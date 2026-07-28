# Wiring `sde/` into the Universe Tag — Design

## Goal

Migrate the Universe tag's tools off their documented temporary ESI-direct exception
(`CLAUDE.md`'s "SDE over ESI" hard rule) onto `sde/`, following up on the sde-infrastructure PR
which built the package but wired in zero consumers. **18 of the 24 exception-listed tools
migrate** — everything SDE can meaningfully cover. Only the **6 celestial-detail tools**
(`get_universe_asteroid_belt`/`_moon`/`_planet`/`_stargate`/`_star`/`_station`) stay ESI-direct,
and only because their defining field, `Name`, doesn't exist anywhere in the SDE — CCP generates
these names server-side from system name + orbital position, not as stored data. Migrating
"everything except Name" would still require a live ESI call per lookup, for no real latency win.
This is a hard, unavoidable gap, not a judgment call.

`get_universe_type` and `list_universe_races` **do** migrate, using the same read-through pattern
as everything else, even though each has exactly one field with no SDE source
(`PackagedVolume`, `AllianceID` respectively — both confirmed genuinely absent, not an engineering
gap). Those two fields are simply dropped on the fast (cache-hit) path, the same treatment as
`get_universe_system`'s dropped `Planets` field — maximizing SDE coverage was chosen over always
paying a live ESI round-trip to backfill one rarely-needed field on every call.

## Verified against real data, not assumed

Every schema claim below was checked directly against the live `data.everef.net/ccp/sde/` archive
and the schema docs at `sde.riftforeve.online` during design — including downloading the real
~94MB archive multiple times to sample actual records, not just reading field-list documentation.
Key findings that shaped this design:

- `mapRegions.jsonl`, `mapConstellations.jsonl`, `mapSolarSystems.jsonl` already embed their
  child-ID arrays directly (`constellationIDs`, `solarSystemIDs`, `planetIDs`, `stargateIDs`) — no
  reverse-index construction needed for the geography sub-group at all.
  `mapSolarSystems.jsonl` has **no** `stationIDs` field, though — confirmed via the schema docs,
  not assumed from a single sample potentially missing it by chance.
- `categories.jsonl`/`groups.jsonl` only point *up* (a group's `categoryID`), never down — ESI's
  `GroupIDs`/`TypeIDs` fields require building reverse indices during sync.
- `types.jsonl` genuinely never contains a `repackagedVolume`/`packagedVolume` field — confirmed
  with a zero-hit `grep -c` across the entire 151MB file, including checking a real ship's (Rifter,
  type 587) record directly, which unambiguously has a non-null `packaged_volume` on ESI. Every
  other `UniverseType` field (`Capacity`, `GraphicID`, `IconID`, `MarketGroupID`, `Mass`,
  `PortionSize`, `Radius`, `Volume`) is directly present in `types.jsonl`; `DogmaAttributes`/
  `DogmaEffects` come from `typeDogma.jsonl`, confirmed present with the exact same
  `{attributeID, value}`/`{effectID, isDefault}` shape the existing ESI-sourced code already uses.
- `races.jsonl` has no `allianceID` field, and there is no `alliances.jsonl` in the archive at
  all — NPC alliance associations aren't published as SDE data, confirmed via the schema docs.
  Every other `UniverseRace` field (`Description`, `Name`) is directly present in `races.jsonl`.
- `factions.jsonl` has no `stationCount`/`stationSystemCount` fields; producing them needs a
  3-hop join (`npcStations.jsonl` `ownerID` → `npcCorporations.jsonl`'s optional `factionID` →
  aggregate per faction), confirmed `factionID` exists on `npcCorporations.jsonl` via the schema
  docs (range 500001–500029, matching EVE's real faction ID range).
- `mapMoons.jsonl` is ~223MB uncompressed — bigger than `types.jsonl` — but is **not** touched by
  this design at all, since celestial detail is out of scope.

## Architecture: read-through cache, `sde/` stays dependency-free

`sde/` does **not** import `esi/` — the read-through fallback (fetch from ESI on a cache miss,
write the result back into the cache) is orchestrated by the caller in `mcp/universe.go`, which
already constructs `esi.Client` for other purposes. This preserves the sde-infrastructure PR's
explicit "no dependency on any other package" boundary for `sde/`.

`*sde.Store` is opened once in `mcp/server.go`'s `NewServer` (alongside the existing
`*config.Store`) and threaded through each migrated handler's constructor as a parameter — the
same dependency-injection shape already used for `*config.Store` throughout `mcp/`, not a new
package-level global. The sketch below elides that threading for brevity; `sdeStore` is that
injected parameter, not a global.

Pattern for every migrated per-ID lookup tool (`get_universe_group`, `get_universe_category`,
`get_universe_region`, `get_universe_constellation`, `get_universe_system`, `get_universe_type`,
`get_universe_graphic`):

```go
func newGetUniverseGroup(sdeStore *sde.Store) func(ctx context.Context, userAgent string, groupID int64) (UniverseGroup, error) {
	return func(ctx context.Context, userAgent string, groupID int64) (UniverseGroup, error) {
		g, ok, err := sdeStore.Group(ctx, userAgent, groupID)
		if err != nil {
			return UniverseGroup{}, fmt.Errorf("read sde cache: %w", err)
		}
		if !ok {
			// Cache miss (never synced, or a genuinely new ID not yet in the
			// snapshot) -- read through to ESI and write the result back so
			// subsequent lookups of this ID are fast.
			client := esi.NewPublicClient(userAgent)
			detail, err := client.UniverseGroupDetail(ctx, groupID)
			if err != nil {
				return UniverseGroup{}, err
			}
			g = sde.Group{ID: detail.GroupId, CategoryID: detail.CategoryId, Name: map[string]string{"en": detail.Name}, Published: detail.Published, TypeIDs: detail.Types}
			if err := sdeStore.PutGroup(g); err != nil {
				slog.Default().WarnContext(ctx, "sde: failed to write read-through group into cache", "error", err)
			}
		}
		return UniverseGroup{GroupID: g.ID, CategoryID: g.CategoryID, Name: g.Name["en"], Published: g.Published, TypeIDs: g.TypeIDs}, nil
	}
}
```

A read-through-fetched record needs no reverse-index computation of its own — ESI's per-ID detail
endpoints already return the child-ID list directly (`detail.Types` above), the same join SDE's
bulk sync has to build by scanning. This is why `sde.Store` needs one new exported write method per
per-ID resource (`PutCategory`, `PutGroup`, `PutRegion`, `PutConstellation`, `PutSystem`,
`PutType`, `PutGraphic`) — a real, non-test production capability, not a test seam — alongside the
existing `Category`/`Group`/etc. read accessors. A failed write-back is logged and swallowed
(matching `ensureSynced`'s existing "enrichment, not required" philosophy) — it only means the
next lookup of that same ID pays the same read-through cost again, not that the current request
fails.

List-only tools with no per-ID concept at all — there is no `get_universe_ancestry`/`_bloodline`/
`_race`/`_faction` tool, only `list_universe_ancestries`/`_bloodlines`/`_races`/`_factions` —
instead read-through at the whole-collection level and get **no** `Put*` method: if the bulk sync
has never populated the bucket at all, fall back to a single live ESI list call and serve that
directly (write-back here is lower-value given these are small, complete, rarely-changing
datasets, and is not required for correctness).

## `sde/` package changes

**New resource files** (one per file, matching the existing `category.go`/`group.go`/`type.go`
pattern): `region.go`, `constellation.go`, `system.go`, `ancestry.go`, `bloodline.go`, `race.go`,
`faction.go`, `graphic.go`. Each defines a minimal struct decoded from the resource's raw stored
JSON, plus a `Store` accessor; `region.go`/`constellation.go`/`graphic.go` also get a `Put*`
write-back method (per-ID tools). `Region`/`Constellation`/`Graphic` need no augmentation (child
IDs, where applicable, already embedded); `System` and `Faction` need sync-time augmentation
(below); `Ancestry`/`Bloodline`/`Race` need none (list-only, stored as-is).

**`type.go`/`group.go`/`category.go` changes**: `Group` gains `TypeIDs []int64`, `Category` gains
`GroupIDs []int64`. `Type` gains `Capacity *float64`, `GraphicID *int64`, `IconID *int64`,
`MarketGroupID *int64`, `Mass *float64`, `PortionSize *int64`, `Radius *float64`, `Volume *float64`,
`DogmaAttributes []TypeDogmaAttribute`, `DogmaEffects []TypeDogmaEffect` (two new small structs
mirroring `mcp/killmails.go`'s existing `DynamicItemAttribute`/`DynamicItemEffect` shape:
`{AttributeID int64; Value float64}` / `{EffectID int64; IsDefault bool}`) — sourced from
`types.jsonl` and a new `typeDogma.jsonl` resource respectively. `PackagedVolume` is **not** added
to `Type` — confirmed absent from the archive, dropped per the missing-field policy.

**Sync algorithm changes** (`sync.go`): the original design's "independent per-file passthrough"
loop no longer holds — categories/groups/systems/factions need data computed from *other* files
before they can be written, so the sync process becomes two phases inside the same atomic
`db.Update`:

1. **Compute phase** (pure in-memory, no bbolt writes yet): stream each source file once, in
   dependency order, accumulating small in-memory maps:
   - `typeDogma.jsonl` → `typeID → {dogmaAttributes, dogmaEffects}`
   - `types.jsonl` (augmented with the dogma map above) → `groupID → []typeID` (also the source for
     the `types` bucket itself)
   - `groups.jsonl` (augmented with the map above) → `categoryID → []groupID`, plus the augmented
     group records themselves
   - `npcCorporations.jsonl` (slim: `_key` + `factionID` only) → `corpID → factionID`
   - `npcStations.jsonl` (slim: `_key` + `solarSystemID` + `ownerID` only, in one pass using the
     map above) → `solarSystemID → []stationID` **and** `factionID → {stationCount, stationSystemCount}`
   - `mapSolarSystems.jsonl` (augmented with the station-ID map above) → augmented system records
   - `factions.jsonl` (augmented with the count map above) → augmented faction records
   - `mapRegions.jsonl`, `mapConstellations.jsonl`, `ancestries.jsonl`, `bloodlines.jsonl`,
     `races.jsonl`, `graphics.jsonl` → stored as-is, no augmentation
2. **Commit phase**: one `db.Update` transaction writes every resulting bucket (clear + rewrite,
   same as today) plus the `_meta` version, all together — unchanged atomicity guarantee from the
   sde-infrastructure PR: a failure anywhere rolls back everything computed, leaving prior data and
   version completely untouched.

`mapMoons.jsonl`/`mapAsteroidBelts.jsonl`/`mapPlanets.jsonl`/`mapStargates.jsonl`/
`mapStars.jsonl` are **not** read at all — celestial detail is out of scope, and (per the
`get_universe_system` design above) the only thing `mapPlanets.jsonl` might have contributed
(`Planets[].AsteroidBeltIDs`/`MoonIDs`) is dropped from `UniverseSystem`'s migrated output per the
missing-field policy, since building a full planet pass just for two nested ID arrays isn't worth
it — `UniverseSystem` keeps `StargateIDs`, `StationIDs`, `StarID`, drops `Planets`.

## `esi/universe.go` / `mcp/universe.go` changes

`esi/universe.go`'s methods for all 18 migrating tools **stay in the code** — they're now the
read-through fallback, not removed. Only the 6 celestial-detail methods keep their handlers calling
`esi/` directly and unchanged, exactly as today.

Each migrated `mcp/universe.go` handler's swappable var is rewritten following the pattern above;
`mcp/server.go` registration and tool signatures (names, inputs, outputs) are unchanged — this is
purely an internal data-sourcing change, not a tool-contract change, except for the three field
drops already agreed:

| Tool | Field dropped | Why |
|---|---|---|
| `get_universe_system` | `Planets` | Would need a full `mapPlanets.jsonl` pass for two nested ID arrays |
| `get_universe_type` | `PackagedVolume` | Confirmed absent from the SDE archive entirely |
| `list_universe_races` | `AllianceID` | Confirmed absent — no `alliances.jsonl` exists in the SDE at all |
| `get_universe_graphic` | `CollisionFile`, `SofDNA` | Confirmed absent from all 6051 `graphics.jsonl` records (found during implementation planning, after this spec's initial approval — every other `graphics.jsonl` field this tool exposes, including `IconFolder`/`SofFactionName`/`SofHullName`/`SofRaceName`, does have full SDE coverage) |

No other migrated tool loses a field — regions/constellations/groups/categories/ancestries/
bloodlines all have full SDE coverage for every field they currently expose.
`list_universe_factions` keeps `StationCount`/`StationSystemCount`, built via the 3-hop join, per
your explicit call to build it despite the complexity.

## Testing

Mirrors the sde-infrastructure PR's established patterns:
- `sde/` internal-package tests pre-seed buckets directly (including pre-seeded reverse-index
  data) to test the new accessors and `Put*` methods without exercising the network path.
- `sde/sync_test.go` gains fixture-zip scenarios covering the two-phase compute-then-commit order
  (e.g., a group's `TypeIDs` correctly reflects `types.jsonl`, a faction's `StationCount`
  correctly reflects the 3-hop join) — external package, `httptest.Server`-backed, same as today.
- `mcp/universe_internal_test.go`'s existing tests for the 18 migrating tools are rewritten to
  stub `sde.Store`'s accessor instead of `esi.Client`'s method; each gains a new
  `_ReadThroughOnCacheMiss` test stubbing a cache miss and asserting the ESI fallback fires and
  its result is written back via the appropriate `Put*` call.

## Docs sync

`CLAUDE.md`'s SDE-over-ESI hard-rule exception shrinks to exactly the 6 celestial-detail tools
staying ESI-direct, with the reasoning (no stored `Name` field exists for these in the SDE) stated
inline — the migration this PR does is essentially complete for the Universe tag, with the
remaining 6 being a permanent, structural decision rather than a "not done yet" gap.

## Explicitly out of scope

- The 6 celestial-detail tools and all `mapMoons`/`mapAsteroidBelts`/`mapPlanets`/`mapStargates`/
  `mapStars` data — no stored `Name` field exists for these in the SDE at all.
- Any SDE resource category beyond what's listed above — added only once a specific tool needs it.
- Any change to `neat setup`, tool names, or tool input/output contracts beyond the three documented
  field drops above (`get_universe_system`'s `Planets`, `get_universe_type`'s `PackagedVolume`,
  `list_universe_races`'s `AllianceID`).
