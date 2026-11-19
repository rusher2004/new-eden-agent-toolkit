# Industry Planning (`industry/` + planning tools + skills) — Design

**Status:** Implemented (docs/superpowers/plans/2026-11-18-industry-planning.md). Rig coverage turned out to be in the SDE itself (`indModifierSources`/`indTargetFilter*`), so no coverage table is kept in code. Jump distances use EVE's 9.46e15 m light year with fuel rounded per jump, which matches dotlan to the unit. Unverified, and labelled so in tool output: the research and invention job fee bases, and the linear shape of the NPC reprocessing standings curve between its two documented endpoints. Still open: whether dotlan's gate route URL takes a safety-mode prefix.
**Date:** 2026-11-18

## Goal

Make "what would it cost to build X, and how do I do it" a repeatable, mostly code-driven workflow for any blueprint, not a one-off analysis. A hand-built capital build-cost study (2026-11-18) did every step in scratch Python: find the owned copies, expand the bill of materials, net out stock, walk Jita order books, compare building intermediates against buying them, solve the cheapest compressed-ore mix, apply refining yield, pack jump freighter loads and price fuel, and estimate job fees and times. This design moves every one of those calculations into Go so Claude spends tokens only on discovery and on decisions with the user.

## Principles

- **Code calculates, Claude decides.** Every number on a plan (quantities, ISK, m³, fuel, yield, time, probability) comes from a tool. Skills tell Claude what to ask, which tool to call and how to present the result; they contain no formulas.
- **Every facility, every security band.** Calculations work for NPC stations, every Upwell structure (citadels, engineering complexes, refineries, faction variants), and legacy starbase arrays, in highsec, lowsec, nullsec, wormhole space, Pochven and Zarzakh. Nothing is special-cased to any particular structure or system.
- **Bonuses come from data, not tables in code.** Structure role bonuses, rig bonuses and each rig's own security multipliers are read from SDE dogma attributes. A hard-coded number is allowed only where the SDE has no source, and each one is listed under "Constants not in the SDE" with its reference.
- **Pure core, thin I/O.** A new `industry/` package holds the formulas and algorithms with no network or disk access, so it is table-testable. `mcp/` gathers inputs (SDE, ESI, config) and calls it.
- **Static data from `sde/`, live data from ESI.** Per the hard rule. No new third-party data source.
- **Facts the API can't see live in saved profiles.** ESI does not expose a structure's rigs, its tax, a player's standings-based station tax inputs, or the jump route a pilot uses. These are saved once as named profiles under `~/.neat/` and reused by name.

## Decisions taken (2026-11-18)

| Question | Decision |
|---|---|
| Where do facility rigs, tax and haul routes come from? | Saved named profiles in `~/.neat/config.json`, managed through MCP tools. |
| Tool shape | Small primitive tools plus one `plan_build` orchestrator that chains them. |
| Market prices | Live ESI region order books, walked for depth and filtered to the hub station. |
| Facility coverage | All facility kinds in all security bands; all industry activities. |

## Facilities and security bands

### Security band

Derived from the system in the SDE, never from a profile:

| Band | Rule |
|---|---|
| Highsec | security status rounds to ≥ 0.5 (true value ≥ 0.45) |
| Lowsec | rounds to 0.1–0.4 |
| Nullsec | ≤ 0.0 in k-space, including Pochven and Zarzakh |
| Wormhole | J-space and Thera (region IDs 11000000+) |

The band picks which multiplier applies. Rig bonuses use the rig's own `hiSecModifier` (2355), `lowSecModifier` (2356) and `nullSecModifier` (2357); wormhole space uses the nullsec value. These differ by rig family. Manufacturing rigs are ×1.0 / 1.9 / 2.1 (high / low / null) and reactor rigs are ×1.0 / 1.1 (low / null) and can't be fitted in highsec (`disallowInHighSec`, 1970). Reprocessing uses its own security term (0, 0.06, 0.12) from the EVE University formula.

### Facility kinds

| Kind | Bonuses and where they come from |
|---|---|
| NPC station | No material or time bonus. Manufacturing and research facility tax 0.25%. Reprocessing base yield from `staStations.reprocessingEfficiency` (30–50%) and station take from `reprocessingStationsTake`; the standings-based reprocessing tax (5% at 0 standing, 0% at 6.67) uses the higher of personal and corporation standing with the station owner, which the profile records. Available services from the station's operation ID. |
| Upwell engineering complex (Raitaru, Azbel, Sotiyo) | `strEngMatBonus` (2600), `strEngCostBonus` (2601, multiplies the system cost index only), `strEngTimeBonus` (2602). Engineering rigs per above. |
| Upwell refinery (Athanor, Tatara) | `strRefiningYieldBonus` (2722) for reprocessing; `strReactionTimeMultiplier` (2721) for reactions. Reactor and reprocessing rigs per above. |
| Upwell citadel (Astrahus, Fortizar, Keepstar, faction Fortizars) | No industry role bonus; can run manufacturing, research, invention and reprocessing with the right service module, at the Upwell base values plus rigs. Faction Fortizar rig bonuses read from their unique rigs' dogma. |
| Starbase (POS) array | Arrays are still types in the SDE but carry no material or time attributes today, so they model as no bonus. Their anchoring limits (`anchoringSecurityLevelMax`, 1032) gate what can be built where. Flagged as legacy in tool output. |

### Where things may run

Enforced by `industry/` and reported as a reason when a plan can't use a facility:

| Rule | Source |
|---|---|
| XL structures (Sotiyo, Keepstar) can't be anchored in highsec. | EVE University, Upwell structure |
| Capital ship manufacturing needs a Standup Capital Shipyard, Large or XL structure, not in highsec, or a lowsec/nullsec NPC station with capital facilities. | EVE University, Manufacturing; Fitting Upwell structures |
| Supercapital manufacturing needs a Sotiyo with a Supercapital Shipyard in a system with the Supercapital Construction Facilities upgrade. | same |
| Reactions need a refinery with the matching reactor, at 0.4 or lower. | EVE University, Reactions |
| Starbase ship building in 0.5 and above is limited to Large Assembly Arrays (battleship and smaller). | EVE University, Player-owned starbases |
| Moon ore reprocessing isn't possible in a Standup Reprocessing Facility outside refineries. | EVE University, Fitting Upwell structures |

The profile records which service modules are fitted, since ESI doesn't expose them.

## Activities

All formulas live in `industry/`. EVE University references are listed at the end.

| Activity | Formula |
|---|---|
| Manufacturing materials | Per job: `max(runs, ceil(round(base × runs × (1 − ME/100) × structureMat × rigMat, 2)))`, where `rigMat = 1 + rigBonus/100 × bandMultiplier`. Rounding is per job, not per run. |
| Manufacturing time | `base × runs × (1 − TE/100) × (1 − 0.04 × Industry) × (1 − 0.03 × Advanced Industry) × Π(1 − 0.01 × science skill) × structureTime × rigTime` |
| Reaction materials and time | Same material formula with reactor rigs; time `× (1 − 0.04 × Reactions) × strReactionTimeMultiplier × rigTime`. |
| Job fee (manufacturing, reactions) | `EIV × (costIndex × structureCostBonus × rigCostMult + facilityTax + 0.04 SCC + 0.0025 if Alpha)`, EIV = Σ ME0 material quantity × ESI adjusted price × runs. Industry taxes are capped at 10%. |
| ME and TE research | Time = level base duration × blueprint rank × skill (Metallurgy or Research, −5%/level; Advanced Industry −3%/level) × facility. Cost = process time value × costIndex × structure bonus, plus facility tax. |
| Copying | Time = 80% of build time per run × copies × runs × (1 − 0.05 × Science) × facility. Cost = costIndex × copies × runs × 0.02 × product EIV, plus tax. |
| Invention | Chance = base × (1 + (science1 + science2)/30 + encryption/40) × (1 + decryptor%/100). Output ME 2 / TE 4 plus decryptor, runs 10 (1 for ships and rigs) plus decryptor. Expected cost per successful copy = attempt cost ÷ chance. |
| Reprocessing (ore, ice, moon ore) | Upwell: `(50 + Rm) × (1 + Sec) × (1 + Sm) × (1 + 0.03 R) × (1 + 0.02 Re) × (1 + 0.02 Op) × (1 + Im)`. NPC station: `stationBase × (1 + 0.03 R) × (1 + 0.02 Re) × (1 + 0.02 Op) × (1 + Im)`, less standings tax. Batch rounding by portion size. |
| Scrapmetal | `base × (1 + 0.02 × Scrapmetal Processing)`; structures use 50% with no structure, rig or implant bonus. |
| Jump fuel | `consumptionAmount (868) × LY × (1 − 0.1 × Jump Fuel Conservation) × (1 + shipSkillBonus × level / 100)`, rounded to the nearest unit per jump; LY = distance between SDE system positions ÷ 9.46 × 10¹⁵ m (EVE's rounded light year, not the astronomical 9.4607 × 10¹⁵). Verified 2026-11-18: dotlan's per-leg Nomad fuel at 5/4/4 (Jita → Aunenen 28,420; Jita → Jan → NL6V-7 → PF-QHK 15,553, 25,148, 28,818) matches exactly with these two choices and with no other combination tried. |
| Jump range | `jumpDriveRange (867) × (1 + 0.2 × Jump Drive Calibration)`; 5 LY base and 10 LY at level V for jump freighters. |
| Jump fatigue | Reactivation delay and fatigue per jump from LY and the ship's `jumpFatigueMultiplier` (1971; 0.1 for jump freighters), using EVE University's jump fatigue formula; cap and decay as documented there. Reported per leg and accumulated over a route. |

`plan_build` covers manufacturing and reaction chains, buying or refining raw inputs. It can also cost an invention step for T2 copies the user doesn't own, as expected cost per successful copy. ME/TE research and copying are separate primitives, not part of `plan_build`.

## Jump and gate routing

Pilots choose routes for reasons the map can't know: who's friendly in an area, where they have docking or a cyno alt, which pipes are camped. So the router finds candidate routes, applies the pilot's saved preferences, and returns several ranked options with the facts needed to choose. It does not pick silently.

**Why built in-house.** dotlan's jump planner, the usual tool, has no documented API. ESI's `/route/` endpoint covers stargates only. The open-source everoute library is old JavaScript, not a service. Everything the router needs is already available: system positions, security and stargate connections from the SDE, and sovereignty, recent kills and contacts from ESI.

**Rules the router enforces:**
- A jump must be within the ship's jump range (above).
- A jump can only land where a cyno can be lit: lowsec or nullsec k-space. Never highsec, wormhole space, Pochven or Zarzakh.
- Jumping out of highsec is allowed for jump freighters and blockade-runner-class jump ships, but jumping into it isn't. A route to or from a highsec hub therefore ends or starts with a gate segment through the nearest legal lowsec system.

**Pilot preferences.** Saved on the haul profile and overridable per call:
- Avoid lists: systems, constellations, regions.
- Prefer station systems, and avoid incursions (from ESI's incursion list), matching dotlan's two options.
- Avoid sovereignty held by named alliances, or by anyone the pilot, their corporation or their alliance has set to bad standing (ESI contacts). Optionally prefer space held by blue alliances.
- Preferred midpoints: systems or structures where the pilot has docking, a cyno alt or a friendly POS. These can be marked required ("only land here") or just preferred (a cost discount).
- Optimise for fuel, number of jumps, or fatigue.
- Gate segments: prefer safer gates (a penalty per low- or nullsec gate, as ESI's "secure" mode does), avoid systems with recent kills above a threshold (ESI system kills), and a cap on gate count.

**Algorithm.** Dijkstra or A\* over a graph with two edge kinds:
- jump edges: system to system within range, landing only where legal and allowed, weighted by the chosen objective;
- gate edges: from the SDE stargate table, weighted by security penalty, recent kills and the gate-count cap.

The router computes the best route under the preferences, then a few alternatives: the best route with no preferred midpoints, the route with the fewest jumps, and the route with the least fatigue. The haul profile can still pin an exact route, which the router validates (range, landing legality, avoid lists) and costs instead of searching.

**Output per route:**
- each leg: from, to, kind (jump or gate), LY, fuel, fatigue after the jump;
- each midpoint: security, sovereignty holder and its standing to you, NPC stations and known accessible structures, and kills in the last hour;
- totals: fuel and its ISK cost, jumps, gates (and highsec gates), and fatigue at arrival;
- any preference the route breaks, with the reason;
- dotlan links that display exactly this route (below).

`plan_haul` and `plan_build` use the top route unless the pilot picked one, and show the alternatives for the pilot to choose.

### dotlan links

Every route the code produces, in every tool that returns one (`plan_jump_route`, `plan_haul`, `plan_build`), carries dotlan URLs that show the same route, so the pilot can check it on the map they already use. The links are built in `industry/route.go` from the route's own data, never by Claude.

**Jump planner URL format** (confirmed with the user 2026-11-18):

```
https://evemaps.dotlan.net/jump/<Ship>,<JDC><JFC><JF>[,S][,I][,-<Avoid>:-<Avoid>…]/<Waypoint>:<Waypoint>:…
```

| Part | Meaning |
|---|---|
| `<Ship>` | Hull type name, e.g. `Nomad`, `Thanatos`. |
| `<JDC><JFC><JF>` | The pilot's Jump Drive Calibration, Jump Fuel Conservation and Jump Freighters levels. Always three digits, whatever the hull; dotlan ignores the third for hulls it doesn't apply to. |
| `,S` | "Prefer Station Systems" is on. |
| `,I` | "Avoid Incursions" is on. |
| `,-A:-B` | Avoid list: system or region names, each prefixed `-`, colon-separated. Omitted when empty. |
| `/<Waypoint>:…` | Systems in order, colon-separated. Where two consecutive waypoints are out of range, dotlan plans the midpoints itself. |

Names: spaces become underscores; hyphens and other characters stay as they are (`Tash-Murkon_Prime`, `Scalding_Pass`, `PF-QHK`). The `S` and `I` flags are omitted when off.

Examples:
- `/jump/Thanatos,544,S,I,-Ahbazon:-Tama/Jita:NL6V-7:PF-QHK`: Thanatos at 5/4/4, prefer stations, avoid incursions, avoid Ahbazon and Tama, dotlan picks the midpoints.
- `/jump/Thanatos,544,-Derelik:-Scalding_Pass/Tash-Murkon_Prime:6RCQ-V`: no flags, two avoided regions (one with a space), a system name with a space.
- `/jump/Nomad,544/Jita:Jan:NL6V-7:PF-QHK`: shows 23.55 LY and 69,519 Hydrogen Isotopes, exactly our formula at 5/4/4.

**What every route carries.** Two kinds of link, each labelled with its segment:
1. **Exact route** (always): every landing system as a waypoint, with the pilot's real skill digits, so dotlan replays our route leg for leg and its fuel matches ours. For a Nomad pilot at 5/5/4 this is `/jump/Nomad,554/Jita:Jan:NL6V-7:PF-QHK`.
2. **dotlan's own suggestion** (for comparison): only the route's endpoints and any required midpoints as waypoints, plus the pilot's preferences translated into dotlan's options:
   - `S` when the pilot prefers station systems;
   - `I` when they avoid incursions;
   - their avoided systems and regions as `-Name` entries, with avoided constellations expanded to their systems since dotlan's list takes only systems and regions.
   Sovereignty and standings preferences have no dotlan equivalent and are named as such next to the link.

**Gate segments.** `https://evemaps.dotlan.net/route/<System>:<System>`, one link per contiguous gate segment, such as Akora → Jita on the return. A mixed route returns its links in order: "Jumps: PF-QHK → Akora", then "Gates: Akora → Jita".

One detail is still unconfirmed and gets checked in the routing task: whether the gate route URL takes a safety-mode prefix. Until then, gate links are plain `/route/A:B`.

### Constants not in the SDE

Each one is documented in code with its source: SCC surcharge 4%, Alpha clone tax 0.25%, industry tax cap 10%, NPC station facility tax 0.25%, the NPC reprocessing standings curve, the Upwell reprocessing base 50% and rig terms (T1 +1, T2 +3) with security terms (0 / 0.06 / 0.12), invention base chances by product group, decryptor modifiers, research level base durations, the copy and research cost factors (0.02, 0.02105), and the security band thresholds. If a constant is later found in the SDE, the code switches to the SDE value.

## Architecture

```
mcp/industry_planning.go ──┬── sde/ (blueprints, reprocessing, dogma bonuses, stations, systems)
  (tool handlers, input    ├── esi/ (order books, adjusted prices, cost indices, assets,
   gathering, output)      │         blueprints, skills, jobs, standings — all pages)
                           ├── config/ (facility, haul and hub profiles)
                           └── industry/ (pure: formulas, tree expansion, LP, packing)
```

### New package `industry/`

| File | Responsibility |
|---|---|
| `security.go` | Security band from status and region; band multiplier lookup for a rig. |
| `facility.go` | Resolves a facility (kind, structure type, rigs, services, band) into per-activity, per-product material, time and cost multipliers; enforces the run-location rules. |
| `materials.go`, `time.go`, `jobcost.go` | Formulas above, including `CalibrateFeeRate(eiv, actualCost)`. |
| `research.go`, `invention.go` | ME/TE research, copying and invention formulas. |
| `tree.go` | Expands a product through manufacturing and reaction activities, honouring per-node build or buy decisions, per-job rounding and the ME of the copies actually used. Flat node list with parent IDs (no recursive types). |
| `stock.go` | Nets needs against owned stock at a location. |
| `market.go` | Walks an order book for a quantity: average, total, shortfall. |
| `buildbuy.go` | Build versus buy per intermediate, with flags for no copies, too few runs, missing skills, facility can't build it here, and market can't supply. |
| `reprocess.go` | Reprocessing for every facility kind, ore, ice, moon ore and scrap. |
| `oremix.go` | Linear program over compressed-ore and mineral order segments using `gonum.org/v1/gonum/optimize/convex/lp` (pure Go simplex); integer portions; byproducts reported. |
| `haul.go` | Light-year distance, jump fuel and fatigue per leg, load packing (even slices; indivisible items by cumulative rounding; each item tagged with its drop-off). |
| `route.go` | Jump and gate route search with pilot preferences, alternatives and validation of pinned routes (see Jump and gate routing). |
| `multibuy.go` | In-game Multibuy "Buy All" text: `Name<TAB>Quantity` per line. |
| `schedule.go` | Longest-job-first slot assignment across characters, respecting skill gates and slot counts. |

### `sde/` additions

- `Blueprint(typeID)` and `BlueprintForProduct(productTypeID)` for all activities (manufacturing 1, TE research 3, ME research 4, copying 5, invention 8, reactions 11): materials, products, probabilities, required skills, base time.
- `ReprocessingMaterials(typeID)`; `CompressedOres()` with portion size, volume and reprocessing skill (attribute 790).
- `NPCStation(stationID)` with reprocessing efficiency, station take, owner corporation and operation services.
- `TypeByName(name)`.
- Named dogma helpers for structure bonuses (2600–2602, 2721, 2722), rig bonuses (2593–2595, 2713, 2714, and the reprocessing rig attribute), security multipliers (2355–2358), `disallowInHighSec` (1970), `anchoringSecurityLevelMax` (1032) and jump drive fuel (866, 868, and the ship's jump-freighter fuel bonus).
- System security status, region and position.

Rig and structure coverage comes from the SDE's own industry modifier tables (`indModifierSources`, `indTargetFilters`, `indTargetFilterCategories`, `indTargetFilterGroups`): per structure or rig, each activity, modifier kind, attribute and product filter. No coverage table is kept in code. Verified 2026-11-18; see the plan's "Resolved during planning" note.

### `esi/` additions

Helpers that walk all pages of existing calls (market orders, corporation assets, corporation blueprints, industry jobs), plus character and corporation standings for NPC reprocessing tax. Every new method routes errors through `describeError`.

### `config/` additions

```go
type FacilityProfile struct {
    Name            string
    Kind            string  // "npc_station" | "upwell" | "starbase"
    StationID       int64   // NPC station or Upwell structure ID
    StructureType   int64   // Upwell or starbase array type ID; role bonuses read from SDE
    SystemID        int64   // cost index and security band (from SDE)
    RigTypeIDs      []int64 // bonuses and security multipliers read from SDE
    Services        []string // fitted service modules: manufacturing, capital_shipyard, supercapital_shipyard, research, invention, reprocessing, biochemical_reactor, composite_reactor, hybrid_reactor
    SupercapUpgrade bool    // sov upgrade present (supercapital building)
    FacilityTax     float64 // owner-set; NPC stations use the 0.25% constant
    ReprocessingTax float64 // owner-set for structures; NPC stations use the standings curve
    Calibration     *Calibration // optional: observed fee rate, time and material multipliers from real jobs
}
type HaulProfile struct {
    Name       string
    ShipTypeID int64
    PilotID    int64   // skills and contacts read from ESI
    HoldM3     float64 // fitted hold; defaults to SDE capacity with the pilot's hull skill bonus
    Outbound   []int64 // optional pinned route (waypoints incl. gate segments), hub → destination
    Return     []int64 // optional pinned return route
    Prefs      RoutePrefs
}
type RoutePrefs struct {
    AvoidSystems, AvoidConstellations, AvoidRegions []int64
    AvoidAlliances     []int64 // sov holders to avoid
    AvoidBadStanding   bool    // avoid sov held by anyone at bad standing in pilot/corp/alliance contacts
    PreferAlliances    []int64 // blue sov to prefer
    Midpoints          []int64 // systems or structure IDs with docking, cyno alts or friendly POSes
    MidpointsRequired  bool    // land only on Midpoints
    PreferStationSystems bool  // prefer midpoints with an NPC station (dotlan "S")
    AvoidIncursions    bool    // avoid systems with an active incursion, from ESI incursions (dotlan "I")
    Optimize           string  // "fuel" | "jumps" | "fatigue"
    MaxGates           int
    MaxRecentKills     int     // per gate system, last hour (ESI system kills)
}
type HubProfile struct {
    Name      string
    RegionID  int64
    StationID int64 // Jita 4-4 by default
}
```

## MCP tools

All new tools are read-only except the profile writers. Verb-first names. Large outputs accept `detail: "summary" | "full"`.

**Profiles:** `save_facility_profile`, `list_facility_profiles`, `delete_facility_profile`, `save_haul_profile`, `list_haul_profiles`, `delete_haul_profile`, `save_hub_profile`, `list_hub_profiles`, `calibrate_facility_profile` (backs out the fee rate and time and material multipliers from a completed job).

**Primitives:**

| Tool | Does |
|---|---|
| `find_owned_blueprints` | A character's or corporation's originals and copies by product, with ME, TE, runs and location resolved to structure, hangar division and named container; flags copies in active jobs. |
| `describe_facility` | Shows a profile's resolved security band and every multiplier per activity and product category, with where each number came from, and any rule that stops an activity there. |
| `get_build_tree` | Bill of materials for N runs at a facility profile, with build/buy defaults, required skills and base times. |
| `check_build_skills` | Which characters can run each job, with slot counts and time multipliers. |
| `check_material_stock` | Nets a material list against assets at a location. |
| `quote_market_purchase` | Walks hub order books for a list; averages, totals, m³, shortfalls. |
| `compare_build_vs_buy` | Per-intermediate build versus buy, with flags. |
| `estimate_job_cost` | Fee for any activity at a facility, or the calibrated rate. |
| `estimate_research` | ME/TE research and copying time and cost. |
| `estimate_invention` | Chance, output copy and expected cost per success, per decryptor. |
| `estimate_reprocessing_yield` | Yield per ore category for a character, facility and implant, any facility kind. |
| `solve_ore_mix` | Cheapest compressed-ore plus direct-mineral purchase for a mineral need. |
| `plan_jump_route` | Ranked jump and gate routes between two systems for a ship and pilot, applying saved or per-call preferences, or validates and costs a pinned route; per-leg LY, fuel and fatigue, midpoint security, sovereignty, standings, docking and recent kills, and labelled dotlan links for every segment. |
| `plan_haul` | Packs cargo into loads for a haul profile and its chosen route; per-load contents, m³ by drop-off, fuel and a Multibuy text per load. |

**Orchestrator:** `plan_build` takes owned blueprint item IDs (or product and count), the owner, builder characters, a build facility profile per activity (manufacturing can differ from reactions), an optional refining facility and refiner, a haul profile, a hub profile and options: compressed ore or not, forced build or buy for named intermediates, ignore stock, invent missing T2 copies. It returns one plan: headline cost, cost breakdown, every decision with its reason, shopping lists, ore mix, refining and in-system shuttle volumes, haul loads with Multibuy texts, job lists with times and fees, and the assumptions and constants used. It saves the full plan as JSON under `~/.neat/plans/` and returns the path.

## Claude skills

In the plugin at `claude_resources/neat/skills/`. Discovery and decisions only; no arithmetic.

- **`industry-build-plan`**:
  1. Find what's buildable (`find_owned_blueprints`), and confirm what to build and how many.
  2. Confirm the owner, builders and refiner (`check_build_skills`).
  3. Make sure facility, haul and hub profiles exist (`describe_facility` to sanity-check them); otherwise hand off to `industry-profile-setup`.
  4. Ask only the real decisions: compressed ore or refined minerals, build-or-buy overrides, whether in-progress jobs count, invention for missing T2 copies.
  5. Call `plan_build`, present the verdict and the numbers that matter, then publish the plan as an artifact with m³ on every material move and a Multibuy copy button at every purchase point.
- **`haul-route-setup`**: discovers the pilot's routing preferences and saves them on the haul profile: which areas and alliances to avoid, where they have docking, cyno alts or friendly POSes, and whether to optimise for fuel, jumps or fatigue. It then calls `plan_jump_route`, shows the alternatives side by side with their dotlan links, and saves the route the pilot picks. Re-run when politics change.
- **`industry-profile-setup`**: lists candidate facilities (NPC stations from the user's assets and jobs; Upwell structures from corp offices), asks for exactly the in-game values ESI can't provide (rigs and services fitted, owner tax, the reprocessing window's yield and fee, the fitted hold size, standings for NPC reprocessing), saves the profile, runs `describe_facility`, and offers `calibrate_facility_profile` against a recent job.

## Testing

- `industry/`: table-driven tests for every formula across the facility and band matrix. They cover:
  - an NPC station
  - Raitaru, Azbel and Sotiyo with T1 and T2 rigs in each band, including wormhole space
  - Athanor and Tatara reactions and reprocessing in lowsec and nullsec
  - a highsec reactor rig rejected
  - a capital build rejected in highsec
  - a starbase array with no bonus
- **Minokawa regression anchors:**
  - an ME9 Minokawa at a T1-XL nullsec Sotiyo needs 28 R-O Trigger Neurolink Conduits and 18 Capital Shield Emitters
  - a capital component run takes 3,534 s at TE20 with Industry V and Advanced Industry V
  - the reactions plan's −2.2% reactor material bonus in nullsec
  - Upwell refining yield at Reprocessing V and Reprocessing Efficiency V, T1 rig, nullsec Tatara, ore skill IV is 82.3% without an implant and 85.6% with an RX-804, confirmed in game
  - the Nomad's Jita → Hakonen → NL6V-7 → PF-QHK route at 5/5/4 burns 59,025 isotopes (dotlan)
  - the job fee rate is about 6.7%
- **Linear program:** a hand-checkable case and an infeasible case.
- **Router:**
  - the Jita → Jan → NL6V-7 → PF-QHK route validates and costs dotlan's 69,519 isotopes at 5/4/4
  - a route never lands in highsec, wormhole space, Pochven or Zarzakh
  - avoid lists and required midpoints are honoured
  - an out-of-range pinned leg is rejected with its distance
  - alternatives differ when fuel, jumps and fatigue are optimised
  - gate segments respect the gate cap
  - the outbound PF-QHK route at Jump Drive Calibration V, Jump Fuel Conservation IV and Jump Freighters IV yields exactly `https://evemaps.dotlan.net/jump/Nomad,544/Jita:Jan:NL6V-7:PF-QHK`, and 5/5/4 skills yield `Nomad,554`
  - a Thanatos at 5/4/4 preferring stations, avoiding incursions, avoiding Ahbazon and Tama, Jita to PF-QHK via NL6V-7 yields exactly `https://evemaps.dotlan.net/jump/Thanatos,544,S,I,-Ahbazon:-Tama/Jita:NL6V-7:PF-QHK` as the comparison link
  - a Thanatos at 5/4/4 from Tash-Murkon Prime to 6RCQ-V avoiding the Derelik and Scalding Pass regions, with no flags, yields exactly `https://evemaps.dotlan.net/jump/Thanatos,544,-Derelik:-Scalding_Pass/Tash-Murkon_Prime:6RCQ-V`
  - an avoided constellation expands to its systems in the comparison link
  - the return route yields one jump link (PF-QHK → Akora) followed by one gate link (Akora → Jita), in order and labelled
- `sde/`: accessors against fixture databases. `mcp/`: internal handler tests with the existing ESI function-variable fakes.
- `go test ./...` must pass the MCP schema check (no self-referential types).

## Docs and versioning

Same PR: `CLAUDE.md` package table (new `industry/`, tool count) and a new hard rule that industry bonuses come from SDE dogma, `README.md`, `claude_resources/README.md` (skills now exist), plugin description, `VERSION` minor bump to 0.6.0 and `go generate ./...`.

## Out of scope

Selling output; contract-based freight beyond a flat ISK/m³ override; reading structure rigs, services or standings-free tax automatically; T3 relic invention chances beyond the documented base values; Ansiblex jump bridges and wormhole connections in routing (Thera and Turnur connections could come later from EVE-Scout's public API); third-party kill or gate-camp feeds such as zKillboard (the router uses ESI's hourly system kills only).

## References

- EVE University: [Manufacturing](https://wiki.eveuniversity.org/Manufacturing), [Upwell structure](https://wiki.eveuniversity.org/Upwell_structures), [Engineering Complexes](https://wiki.eveuniversity.org/Engineering_Complexes), [Fitting upwell structures](https://wiki.eveuniversity.org/Fitting_upwell_structures), [Reprocessing](https://wiki.eveuniversity.org/Reprocessing#Upwell_reprocessing_formula), [Reactions](https://wiki.eveuniversity.org/Reactions), [Research](https://wiki.eveuniversity.org/Research), [Invention](https://wiki.eveuniversity.org/Invention), [Player-owned starbases](https://wiki.eveuniversity.org/Player-owned_starbases).
- SDE (fuzzwork dump): `dgmTypeAttributes` for structure, rig and ship bonuses; `staStations` for NPC reprocessing; `industryActivity*` for blueprints; `invTypeMaterials` for reprocessing outputs; `mapSolarSystems` and `mapSolarSystemJumps` for routing.
- Routing: [dotlan jump planner](https://evemaps.dotlan.net/jump) (no documented API; used only as an outbound link), [ESI routes](https://github.com/esi/esi-routes) (gates only), [everoute](https://github.com/dertseha/everoute) (reviewed, not used).
