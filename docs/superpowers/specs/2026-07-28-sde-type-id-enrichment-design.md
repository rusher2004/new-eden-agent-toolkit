# SDE-backed `type_id` enrichment — Design

## Goal

Every `esi/` method returns exactly what ESI returns, including bare IDs, with zero
name-resolution enrichment — an explicit, intentional convention stated in
`docs/reference/esi-coverage-roadmap.md`, deferred there as "a `sde/`-backed follow-up once that
package exists." `sde/` now exists and already backs 24 of the Universe tag's 30 tools
(`docs/superpowers/specs/2026-07-27-sde-universe-migration-design.md`). This is that follow-up,
scoped to its first and highest-value slice: resolving `type_id`-like fields (item, ship, and
blueprint types) across the other 35 ESI tags.

This is batch 1 of a larger sweep. Location IDs (`system_id`/`region_id`/`constellation_id`/
`station_id`/`planet_id`/`moon_id`) and the rarer `race_id`/`bloodline_id`/`faction_id`/`group_id`
fields are explicit follow-up batches — not part of this design.

## Inventory: what actually needs enriching

A full read-only audit of `esi/*.go`/`mcp/*.go` (excluding `universe.go`, which already handles
its own `type_id`s, and `wallet.go`, which has none) found:

- **24 of the 35 non-Universe tags** carry at least one SDE-coverable bare ID; 11 have none
  (`access_list`, `calendar`, `clones`, `contacts`, `meta`, `skills`, `status`, `wars`, `mail`,
  `freelance_jobs`, `incursions`'s non-type fields).
- **~45 distinct struct fields** are `type_id`-like, spanning `assets.go`, `contracts.go`,
  `character.go` (blueprints, agent research), `corporation.go` (blueprints, container logs,
  facilities, member tracking, starbases, structures), `dogma.go`, `fittings.go`, `fleets.go`,
  `industry.go` (jobs, mining, facilities), `insurance.go`, `killmails.go`, `location.go`
  (`get_character_ship`), `loyalty.go`, `market.go`, `planetary_interaction.go`, `search.go`,
  `structures.go` (mercenary dens, skyhooks, sovereignty hubs), `corporation_projects.go`.
- A handful of endpoints return **lists where every item carries its own `type_id`**:
  `list_market_prices` (effectively every tradeable type in the game), `list_market_orders`,
  `list_character_contract_items`/`list_public_contract_items`, `get_killmail` (items nested inside
  items, for destroyed containers), `list_character_blueprints`/`list_assets`,
  `get_character_planet` (pins, nested pin contents, and routes each carry their own `type_id`).
  These matter for the design's dedup requirement below, not because they're individually
  expensive — see "Performance."
- `mercenary tactical operations`' `DungeonTypeID` is not confirmed to live in the SDE `Type`
  table — verify during implementation before wiring it up; if absent, drop it from batch 1 rather
  than blocking on it.
- Two fields are bare ID **lists**, not a single ID per record: `SearchResults.InventoryType
  []int64` (`search_character`) and `ListMarketTypesOutput.TypeIDs []int64` (`list_market_types`).
- Explicitly **excluded** from this batch: `character_id`/`corporation_id`/`alliance_id` fields
  (not SDE data at all); generic `location_id` fields that could reference either an NPC station
  (SDE-coverable) or a player-owned structure (not SDE-coverable — needs a live, uncacheable ESI
  call) — these need their own resolution strategy and belong in the location-ID batch, not here;
  pure object identifiers (`contract_id`, `job_id`, `order_id`, ...) that don't reference a named
  entity at all; and input-echo fields (e.g. `create_character_fitting`'s request body,
  `open_market_details_window`'s input `TypeID`) — enriching a caller-supplied value with a name
  the caller could already look up adds no information and isn't response data.

## Output shape: additive sibling field

Every `FooTypeID int64` field gains a sibling `FooTypeName string` (`json:"fooTypeName,omitempty"`),
named by mechanically replacing the `...ID` suffix with `...Name` — e.g. `ShipTypeID` →
`ShipTypeName`, `BlueprintTypeID` → `BlueprintTypeName`, `MutatorTypeID` → `MutatorTypeName`. This
is purely additive: no existing field changes shape or is removed, so no tool's output schema
breaks for a caller already depending on it.

Two naming exceptions, where the source is a list of IDs rather than one ID per record:
`SearchResults.InventoryType []int64` gains `InventoryTypeNames []string`, index-aligned with the
original; `ListMarketTypesOutput.TypeIDs []int64` gains `TypeNames []string`, same treatment.

A rejected alternative was a nested `{id, name}` object (e.g. replacing `TypeID int64` with `Type
struct { ID int64; Name string }`), which groups the two together more cleanly but changes every
affected struct's shape — a breaking change to tool output schemas already shipped in v0.1.x, for
no benefit proportional to that cost.

## Architecture: one shared resolver, called per tool

A new `mcp/enrich.go` defines:

```go
// resolveTypeNames looks up names for a set of type IDs, deduplicating so a
// value repeated across many records (e.g. the same item type on hundreds of
// asset rows) costs one lookup, not one per occurrence. A lookup failure —
// whether a genuine miss or a transient ESI error on the read-through
// fallback path — is dropped from the result rather than surfacing an error:
// enrichment is additive, not required, so one bad ID must not fail an
// entire tool call over data ESI already returned successfully.
func resolveTypeNames(ctx context.Context, sdeStore typeCache, userAgent string, ids []int64) map[int64]string {
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		t, ok, err := sdeStore.Type(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = t.Name["en"]
	}
	return names
}
```

`typeCache` is the interface `mcp/universe.go` already defines over `sde.Store.Type()` (cache-hit,
or ESI-fallback-and-write-back on miss) — reused as-is, not redefined. Each affected tool's handler
calls `resolveTypeNames` once per response with every type ID it found in that response, then fills
in the `*Name` siblings from the returned map. No new interface, no changes to `sde/` or `esi/`.

**Rejected: inline per-tool duplication.** `universe.go`'s existing per-tool pattern (each tool
does its own single-ID lookup inline) doesn't scale to ~20 tools each populating several fields —
that's ~20 near-identical copies of dedup-and-miss-handling logic, and a maintenance liability at
this scale (e.g. changing miss behavior later means editing 20 files instead of 1).

**Rejected: pushing batching into `sde/`.** A `sde.Store.TypeNames(ctx, ua, ids []int64)` method
would need to orchestrate the ESI-fallback-and-write-back path itself, which requires constructing
an `esi.Client` — something `sde/` deliberately never does (`sde/` never imports `esi/`; the
sde-universe-migration design established that boundary and this shouldn't erode it for
convenience).

## Performance

`sde/`'s type table is bulk-synced in full at process start (FC's SDE zip fetch is sync-fatal, so
if the process is running at all, the full type table is present). Because of that, the
high-cardinality endpoints flagged in the inventory (`list_market_prices`, `get_killmail`, contract
items) are still fast: virtually every lookup is a cache hit — a single in-process bbolt read, not
a network call. `resolveTypeNames`'s dedup already removes the main source of redundant lookups
within one response. No bulk/batch bbolt-read API is being added for `sde/` in this batch; revisit
only if real profiling shows a problem, not preemptively.

## Testing

- `resolveTypeNames`: dedup (two records sharing an ID cost one `Type()` call, verified via a call
  counter on the stub `typeCache`), a miss (returns a map missing that ID, no error), and an
  ESI-fallback error path (a stub `typeCache` returning `err != nil`) — confirms the tool call still
  succeeds with that one field simply omitted.
- Each of the ~20 affected tools' existing tests gets a case asserting the new `*Name` field
  populates correctly against a stubbed `typeCache`, plus one asserting it's omitted (not zero'd
  awkwardly, not erroring) on a stubbed miss.
- `DungeonTypeID`: verify against real SDE data before wiring it up; if it's not present, exclude
  it from this batch and note why in the roadmap doc, the same way `get_universe_type`'s
  `PackagedVolume`/`get_universe_graphic`'s `CollisionFile` gaps are documented rather than
  silently dropped.

## Docs sync

- `docs/reference/esi-coverage-roadmap.md`'s per-tag rows don't change (this doesn't add or remove
  endpoints), but its "Conventions" section's `esi/`-methods-return-bare-IDs note should gain a
  pointer to this design once implemented, so a future reader doesn't have to rediscover that the
  `sde/`-backed follow-up it mentions has actually happened.
- `CLAUDE.md`'s package table entry for `mcp/` should mention `enrich.go`'s role once implemented,
  per this repo's own "keeping docs & Claude resources in sync" rule.

## Explicitly out of scope (future batches)

- **Batch 2**: location IDs (`system_id`/`region_id`/`constellation_id`/`station_id`/`planet_id`/
  `moon_id`), including designing a resolution strategy for the ambiguous generic `location_id`
  fields (NPC station vs. player-owned structure).
- **Batch 3**: the rarer `race_id`/`bloodline_id`/`faction_id`/`group_id` fields, mostly on
  character/corporation/alliance/killmail records.
- Enriching `character_id`/`corporation_id`/`alliance_id` fields at all — these aren't SDE data;
  resolving them would mean live ESI calls (`resolve_universe_ids`/`_names` already exist as
  standalone tools for that) and is a different, non-`sde/`-backed problem.
