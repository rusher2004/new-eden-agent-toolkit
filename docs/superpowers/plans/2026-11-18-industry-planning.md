# Industry Planning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement [the industry planning design](../specs/2026-11-18-industry-planning-design.md): a pure `industry/` calculation package, `sde/` and `esi/` additions, saved facility/haul/hub profiles, about 25 MCP tools including `plan_build` and `plan_jump_route`, and three plugin skills.

**Architecture:** `industry/` is pure Go with no I/O. `mcp/industry_*.go` handlers gather SDE, ESI and config inputs, call `industry/`, and shape outputs. Profiles persist in `~/.neat/config.json`. Routing and dotlan links live in `industry/route.go`.

**Tech stack:** Go 1.26.1; new dependency `gonum.org/v1/gonum` (only `optimize/convex/lp` and `mat`) for the ore-mix linear program; everything else already in `go.mod`.

## Global constraints

- `CGO_ENABLED=0` stays true (gonum is pure Go).
- No environment variables; profiles live in `~/.neat/config.json`; saved plans in `~/.neat/plans/`.
- Static data only through `sde/`; live data only through `esi/` with `sso.UserAgent(version)` and `describeError`.
- Industry bonuses are read from SDE dogma. Every number not in the SDE is a named constant in `industry/constants.go` with a source comment.
- No self-referential structs in any MCP input or output type. Trees are flat node lists with parent IDs.
- Errors wrapped with context; never discarded.
- External test packages by default; standard library only; table-driven where a table fits.

## Resolved during planning

**Rig coverage (spec open item): resolved by the SDE itself.** The fuzzwork dump carries the game's industry modifier tables (verified 2026-11-18):
- `indModifierSources` lists, per structure or rig type, each activity (`manufacturing`, `reaction`, `copying`, `invention`, `researchMaterial`, `researchTime`), the modifier kind (`material`, `time`, `cost`), the dogma attribute and an optional target filter. The Sotiyo gives 2600/2601/2602 for all activities with no filter; the Tatara gives `reaction` `time` 2721; the Athanor has no entry; Standup XL-Set Ship Manufacturing Efficiency I gives `manufacturing` `material` 2592 and `time` 2591 on filter 3 "Ships".
- `indTargetFilters`, `indTargetFilterCategories` and `indTargetFilterGroups` define the 18 filters by product category and group. For example: 13 "Capital Components" = group 873; 11 "Capital Ships" = groups 485, 547, 883, 1538, 4594, 5120; 18 "Composite Reactions" = groups 428, 429, 4932.
- For structures, the attribute value on the structure type is the multiplier (0.99, 0.7, ...). For rigs, the listed attribute is the structure attribute the rig modifies; the amount is the rig attribute that its dogma effect's `modifierInfo` maps onto it (2593, 2594, 2595, 2713 or 2714, a percent), scaled by the rig's own security multiplier.

No bucket table is needed in code. Task 3 builds facility multipliers from these tables.

---

## Task 1: Dependencies and `industry/` foundation

**Files:** `go.mod`, `go.sum`; create `industry/doc.go`, `industry/constants.go`, `industry/security.go`, `industry/security_test.go`.

- [ ] `go get gonum.org/v1/gonum@v0.16.0` (or the latest version at least two weeks old).
- [ ] `constants.go`:
  - SCC surcharge 0.04, Alpha tax 0.0025, industry tax cap 0.10, NPC facility tax 0.0025.
  - Upwell reprocessing base 50, rig terms (T1 1, T2 3), security terms (0, 0.06, 0.12).
  - NPC reprocessing standings curve (5% at 0 standing, 0% at 6.67).
  - Invention base chances by group, the decryptor table, research level base durations, copy factor 0.02 and research factor 0.02105, LY = 9.4607e15 m.
  - Each constant has a source comment.
- [ ] `security.go`:
  - `type Band int` with `Highsec`, `Lowsec`, `Nullsec` and `Wormhole`.
  - `BandOf(security float64, regionID int64) Band`: round security to one decimal the way the game does (0 < s < 0.05 shows as 0.1, else round half up); regions ≥ 11000000 are wormhole space; Pochven (10000070) and Zarzakh are nullsec.
  - `CynoLegal(band, regionID) bool`: false for highsec, wormhole space, Pochven and Zarzakh.
  - `RigMultiplier(band, hi, low, null float64) float64`: wormhole space uses the nullsec value.
- [ ] Tests:
  - Aunenen 0.447790 → lowsec, Jita 0.9459 → highsec, −0.0099 → nullsec, 0.04 → lowsec;
  - wormhole region → wormhole band; Pochven → nullsec but not cyno-legal;
  - rig multiplier picks the right column.
- [ ] `go test ./industry/...`.

## Task 2: `sde/` industry accessors

**Files:** create `sde/industry.go` (`sde/industry_test.go`), `sde/dogma.go` (`sde/dogma_test.go`), `sde/routing.go` (`sde/routing_test.go`); modify `sde/store_test.go` (`testSchema` gains `industryActivity`, `industryActivityMaterials`, `industryActivityProducts`, `industryActivitySkills`, `industryActivityProbabilities`, `invTypeMaterials`, `dgmEffects`, `dgmAttributeTypes`, `mapSolarSystemJumps`, and the `staStations` columns `operationID`, `security`, `reprocessingHangarFlag`).

- [ ] `Blueprint(ctx, ua, blueprintTypeID) (Blueprint, bool, error)`: per activity, the materials, products (with probability), skills and base time.
- [ ] `BlueprintsForProduct(ctx, ua, productTypeID) ([]BlueprintActivityRef, error)`: blueprint and activity pairs that output the product (manufacturing 1 or reactions 11).
- [ ] `ReprocessingMaterials(ctx, ua, typeID) ([]TypeQuantity, error)`.
- [ ] `CompressedOres(ctx, ua) ([]Ore, error)`: published types named `Compressed %` in category 25, excluding ice and moon groups unless asked; portion size, volume and reprocessing skill (attribute 790).
- [ ] `TypesByName(ctx, ua, names []string) (map[string]int64, error)`, exact and case-insensitive.
- [ ] `NPCStationIndustry(ctx, ua, stationID)`: reprocessing efficiency, station take, owner corporation, operation ID and system.
- [ ] `TypeAttributes(ctx, ua, typeID) (map[int64]float64, error)` plus helpers `StructureRoleBonuses`, `RigProfile` (bucket attribute IDs from `dgmEffects.modifierInfo`, bonus values, security multipliers, `disallowInHighSec`) and `JumpDrive(shipTypeID)` (attributes 866, 867, 868, 1971, and the ship's jump-freighter fuel bonus attribute 1312).
- [ ] `SystemsForRouting(ctx, ua) ([]RoutingSystem, error)`: ID, name, constellation, region, region name, security and position for all k-space systems; `Stargates(ctx, ua) ([][2]int64, error)` from `mapSolarSystemJumps`; `StationSystems(ctx, ua) (map[int64]bool, error)`.
- [ ] Fixture tests per accessor, seeded with real rows copied from the live dump (the Minokawa blueprint, the Capital Siege Array blueprint, Compressed Gneiss IV-Grade, Jita 4-4, the Sotiyo, Tatara and the T1 XL ship rig, the Nomad, Jita / Aunenen / NL6V-7 / PF-QHK positions).

## Task 3: Facility resolution

**Files:** modify `sde/dogma.go` (add `IndustryModifiers(typeID)` and `TargetFilters()`), create `industry/facility.go`, `industry/facility_test.go`.

- [ ] `sde`: `IndustryModifiers` reads `indModifierSources` for a type; `TargetFilters` reads the three filter tables.
- [ ] `facility.go`:
  - `type Facility struct { Kind; StructureType; Band; RoleBonuses; Rigs []Rig; Services; SupercapUpgrade; FacilityTax; ReprocessingTax; NPCStation *NPCStation; Calibration *Calibration }`.
  - `func (f Facility) Multipliers(activity, categoryID, groupID) (Multipliers, []Source)`: material, time and cost multipliers. Rigs hitting any bucket the product is in combine multiplicatively per distinct rig; structure role bonuses; NPC stations 1.0.
  - `func (f Facility) CanRun(activity, product ProductInfo) error`, with reasons for:
    - capital work without a capital shipyard, or in highsec;
    - supercapital work without a Sotiyo, a supercapital shipyard or the sovereignty upgrade;
    - reactions in highsec or without the matching reactor;
    - starbase battleship limits at 0.5 and above;
    - XL structures in highsec;
    - moon ore outside a refinery.
- [ ] Tests (matrix):
  - Sotiyo with a T1 XL ship rig in nullsec gives 0.99 × 0.958 materials and 0.7 × 0.58 time on a Minokawa and on a Capital Siege Array;
  - the same rig gives ×1.9 in lowsec and ×1.0 in highsec;
  - a T2 rig (−2.4 / −24);
  - Tatara with an L-Set Reactor Efficiency I in nullsec gives a 0.978 material multiplier on composite reactions (the rig's −2.2% in nullsec) and 0.75 × 0.78 time;
  - a Raitaru in highsec, an Azbel in lowsec, a citadel with no role bonus;
  - an NPC station gives 1.0;
  - a highsec capital build is rejected, a highsec reactor is rejected, a wormhole-space rig uses the nullsec value.

## Task 4: Job formulas

**Files:** create `industry/materials.go`, `industry/time.go`, `industry/jobcost.go`, `industry/research.go`, `industry/invention.go` and their `_test.go` files.

- [ ] `MaterialQuantity(base, runs int64, me int, facilityMult float64) int64` with the game's rounding: `max(runs, ceil(round(x, 2)))`.
- [ ] `JobTime(baseSeconds float64, runs int64, te int, skills SkillMultipliers, facilityMult float64) float64`.
- [ ] `EstimatedItemValue(materials, adjustedPrices, runs)`, `JobFee(eiv, costIndex, structureCostBonus, rigCostMult, facilityTax, alpha bool)` with the 10% tax cap, and `CalibrateFeeRate(eiv, actual)`.
- [ ] `ResearchTime/Cost`, `CopyTime/Cost`, `InventionChance`, `InventionOutput`, `ExpectedCostPerSuccess`.
- [ ] Regression tests:
  - ME9 Minokawa conduits 32 → 28, shield emitters 20 → 18;
  - ME6 copy: capacitor batteries 15 → 14, conduits 32 → 29;
  - component per-job rounding: 5 runs of a Capital Siege Array at ME10 needs ceil(round(101,250 × 5 × 0.9 × 0.99 × 0.958, 2)) Pyerite;
  - one capital component run takes 3,533.9 s;
  - a Minokawa hull job at TE14 takes 474,858 s, matching an in-game job;
  - fee rate recovers 6.68% from EIV 2,692,208,286 and cost 179,799,038;
  - an invention chance case from EVE University's formula.

## Task 5: Tree, stock, market and build-vs-buy

**Files:** create `industry/tree.go`, `industry/stock.go`, `industry/market.go`, `industry/buildbuy.go` and tests.

- [ ] `tree.go`:
  - `ExpandTree(req TreeRequest) (Tree, error)`. Inputs: product, quantity or runs, per-node `Decision` (Build / Buy / Auto), available copies per blueprint (ME, TE, runs; originals have unlimited runs), facility resolver per activity.
  - Copies are consumed largest-runs-first, one job per copy, so per-job rounding matches how the user will install jobs. Original blueprints use one job per requested batch size (default: whole quantity).
  - Output is a flat `[]Node{ID, ParentID, TypeID, Quantity, Activity, Jobs[]}` plus aggregated leaf `[]TypeQuantity` and `[]Job`.
- [ ] `stock.go`: `NetStock(needs, stock) (remaining, used []TypeQuantity)`.
- [ ] `market.go`:
  - `WalkBook(orders []Order, qty int64) Quote{Avg, Total, Filled, Shortfall}`, sorted by price, sell side only, filtered by station.
  - `QuoteList` for many types.
- [ ] `buildbuy.go`: `CompareBuildVsBuy(...) []BuildBuy` with per-unit build cost (walked input prices + freight + fee) and buy cost (walked price + freight), plus flags.
- [ ] Tests:
  - the Minokawa tree for 14 ME9 + 1 ME6 hulls reproduces 135 propulsion engines, 196 capacitor batteries, 331 siege arrays, 5,201 seals and 2,601 support units;
  - netting Isogen 1,720,083 out of stock;
  - walking a three-order book with a shortfall;
  - a component with 4 units on the market is flagged "market can't supply".

## Task 6: Reprocessing and ore mix

**Files:** create `industry/reprocess.go`, `industry/oremix.go` and tests.

- [ ] `ReprocessYield(facility, skills, oreSkillLevel, implant)` for Upwell and NPC stations; `Reprocess(ore, qty, yield) []TypeQuantity` with whole portions; scrapmetal variant.
- [ ] `SolveOreMix(needs, ores []OreOption, minerals []MineralOption, freightPerM3)` using `optimize/convex/lp.Simplex`:
  - variables are order segments;
  - minimise price plus m³ × freight subject to yield ≥ need;
  - round ores up to whole portions, then cover any remainder with direct mineral buys;
  - report byproducts and surplus.
- [ ] Tests:
  - Reprocessing V / Reprocessing Efficiency V yield is 82.3% without an implant and 85.6% with an RX-804 (Simple, Coherent and Variegated ore at IV);
  - a two-ore, two-mineral hand case with a known optimum;
  - an infeasible case returns an error naming the short mineral;
  - Compressed Gneiss IV-Grade × 2,413,600 yields 19,012,595 Isogen at 85.6226%.

## Task 7: Hauling, fuel, fatigue, Multibuy

**Files:** create `industry/haul.go`, `industry/multibuy.go` and tests.

- [ ] `LightYears(a, b Position)`, `JumpFuel(drive JumpDrive, skills, ly) int64` (rounded up per jump), `JumpRange(drive, jdc)`, `Fatigue(prev, ly, multiplier)` per EVE University jump fatigue.
- [ ] `PackLoads(items []Cargo, holdM3 float64) []Load`: even slices; indivisible or portioned items by cumulative rounding; the rest water-filled; each load tagged with drop-off volumes.
- [ ] `Multibuy(items) string`.
- [ ] Tests:
  - the Jita → Jan → NL6V-7 → PF-QHK Nomad route at 5/4/4 burns 69,519, and Jita → Hakonen → NL6V-7 → PF-QHK at 5/5/4 burns 59,025 (dotlan, matched per leg);
  - Nomad range at Jump Drive Calibration V is 10 LY;
  - packing 2,678,010 m³ into 158,400 m³ gives 17 loads, all within the hold, with quantities that sum exactly;
  - the Multibuy format.

## Task 8: Routing and dotlan links

**Files:** create `industry/route.go`, `industry/dotlan.go` and tests.

- [ ] Graph:
  - jump edges between k-space systems within range whose destination is cyno-legal and not avoided;
  - gate edges from stargates;
  - weights per `Optimize` (fuel = LY, jumps = 1, fatigue = fatigue increment), plus gate penalties (security, recent kills, gate cap) and a preferred-midpoint discount;
  - required midpoints restrict jump destinations.
- [ ] `PlanRoute(req) ([]Route, error)`: best route plus alternatives (no-preference, fewest jumps, least fatigue), deduplicated.
- [ ] `ValidateRoute(pinned)`: range, landing legality, avoid lists.
- [ ] `dotlan.go`:
  - `ExactJumpURL(ship, skills, waypoints)`;
  - `SuggestionJumpURL(ship, skills, endpoints, prefs, avoidNames)`;
  - `GateURL(a, b)`;
  - names with spaces → underscores; flags `S` and `I` only when on; constellations expanded to systems.
- [ ] Tests:
  - the exact URLs from the spec: `Nomad,544/Jita:Jan:NL6V-7:PF-QHK`, `Nomad,554/...`, `Thanatos,544,S,I,-Ahbazon:-Tama/Jita:NL6V-7:PF-QHK`, `Thanatos,544,-Derelik:-Scalding_Pass/Tash-Murkon_Prime:6RCQ-V`;
  - the return route yields a jump link then a gate link;
  - a route never lands in highsec, wormhole space, Pochven or Zarzakh;
  - an avoided system is never used; required midpoints are honoured;
  - an out-of-range pinned leg is rejected with its distance;
  - the gate cap is enforced.
  - Use a small synthetic graph for the algorithm tests, plus the real systems for the PF-QHK regression.
- [ ] Manual check: whether `/route/` accepts a safety prefix. If it does, add it to `GateURL` and the spec; if not, note it in the spec.

## Task 9: Scheduling

**Files:** create `industry/schedule.go` and test.

- [ ] `Schedule(jobs []Job, builders []Builder) Assignment`: longest job first onto the earliest-free slot of an eligible builder; returns makespan, per-builder job lists and slot-days.
- [ ] Test: 293 component jobs over 40 slots gives about 2.45 days.

## Task 10: Profiles in `config/`

**Files:** modify `config/config.go`; create `config/profiles.go` and `config/profiles_test.go`.

- [ ] Add `FacilityProfiles`, `HaulProfiles`, `HubProfiles` to `Config` (with `omitempty`), the structs from the spec, `Calibration`, `RoutePrefs`.
- [ ] Helpers: `UpsertFacilityProfile`, `DeleteFacilityProfile`, `FacilityProfile(name)`, and the same for haul and hub profiles; names unique and case-insensitive.
- [ ] Tests: the round trip, upsert replaces, delete reports missing names, and existing `Characters` data survives.

## Task 11: `esi/` helpers

**Files:** create `esi/pages.go` (and tests); modify `esi/contacts.go` and `esi/sovereignty.go` only if the needed calls are missing.

- [ ] Generic `AllPages[T]` walker used by: region market orders by type, corporation assets, corporation blueprints, character assets and blueprints, corporation industry jobs.
- [ ] Character, corporation and alliance standings/contacts (existing calls), the sovereignty map and incursions (existing), and system kills (existing).
- [ ] Every new `.Execute()` path goes through `describeError`.

## Task 12: Profile tools

**Files:** create `mcp/industry_profiles.go` and its internal test; modify `mcp/server.go`.

- [ ] Tools:
  - `save_facility_profile`, `list_facility_profiles`, `delete_facility_profile`;
  - `save_haul_profile`, `list_haul_profiles`, `delete_haul_profile`;
  - `save_hub_profile`, `list_hub_profiles`;
  - `calibrate_facility_profile`, `describe_facility`.
- [ ] `save_facility_profile` validates against the SDE: the structure type is a structure, the rigs are rigs whose size fits the structure, the services are known names, and the system exists. `describe_facility` returns the band and every multiplier with its source.
- [ ] Tests with a temp config and a seeded SDE store.

## Task 13: Primitive tools

**Files:** create `mcp/industry_tools.go` (split by area if it grows past about 800 lines) and its internal tests; modify `mcp/server.go`.

- [ ] Tools:
  - `find_owned_blueprints`, `get_build_tree`, `check_build_skills`, `check_material_stock`;
  - `quote_market_purchase`, `compare_build_vs_buy`, `estimate_job_cost`;
  - `estimate_research`, `estimate_invention`;
  - `estimate_reprocessing_yield`, `solve_ore_mix`;
  - `plan_jump_route`, `plan_haul`.
- [ ] Each handler only gathers inputs and calls `industry/`. ESI calls go through package-level function variables so tests fake them, as the existing tools do.
- [ ] Order-book fetches are de-duplicated per call and run with bounded concurrency (4).
- [ ] Tests per tool with fakes.

## Task 14: `plan_build`

**Files:** create `mcp/industry_plan.go` and its internal test; modify `mcp/server.go`.

- [ ] Chain the steps:
  1. Copies, tree, stock, quotes and build-vs-buy.
  2. Optional ore mix and reprocessing.
  3. Route, packing and fuel.
  4. Fees and the schedule.
  5. Assemble the output described in the spec.
- [ ] `detail: summary | full`.
- [ ] Save the full plan JSON to `~/.neat/plans/<timestamp>-<product>.json` and return the path.
- [ ] Test: an end-to-end run with fakes on a two-level synthetic blueprint, including ore mix and two loads.

## Task 15: Plugin skills

**Files:** create `claude_resources/neat/skills/industry-build-plan/SKILL.md`, `claude_resources/neat/skills/industry-profile-setup/SKILL.md`, `claude_resources/neat/skills/haul-route-setup/SKILL.md`.

- [ ] Each skill:
  - has frontmatter `name` and a trigger-oriented `description`;
  - holds discovery steps and decision questions only, with no formulas or arithmetic;
  - names the exact tools to call and the in-game values to ask for;
  - for build plans, requires an artifact with m³ on every material move and Multibuy copy buttons at every purchase point.

## Task 16: Docs, version, verification

**Files:** `CLAUDE.md`, `README.md`, `claude_resources/README.md`, `claude_resources/neat/.claude-plugin/plugin.json` (via `go generate`), `VERSION`, `claude_resources/neat/CHANGELOG.md`, the spec's status line.

- [ ] `CLAUDE.md`:
  - a package table row for `industry/`;
  - the `mcp/` tool list and count;
  - a hard rule that industry bonuses come from SDE dogma, with the constants file as the only exception.
- [ ] `VERSION` → 0.6.0; `go generate ./...`.
- [ ] `go build ./... && go vet ./... && go test -race ./...`.
- [ ] Live smoke test against the real SDE and ESI:
  - reproduce the hand-built capital study's headline numbers within rounding;
  - reproduce the PF-QHK route URLs.
- [ ] Update the spec's status line to Implemented.
