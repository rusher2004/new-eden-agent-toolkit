# Logistics Planning (accounts, structures, hubs, hauling math) — Design

**Status:** In progress, as stacked PRs, one per phase. Phase 0 (package split): PR #23. Phase 1 (foundations): account labels and notes live in `config.json` until phase 2 moves characters to `characters.yaml`; account warnings are given only when two or more characters are logged in, since one character can't share an account with anyone; `list_location_assets` returns a flat list in tree order with `parentItemId` and `depth` rather than nested objects, because MCP output schemas can't be self-referential; route legs carry `fromSecurity`/`toSecurity` objects (status, display, color, band), while `get_universe_system` gets `securityDisplay`/`securityColor` fields. Phase 2 ships as three stacked PRs: 2a (the shared YAML store, moving characters, facility and haul profiles and pinned routes out of `config.json`, and the route tools), 2b (structures) and 2c (hubs). In 2a, hub profiles stay in `config.json` until 2c; nested mappings that lose a key are re-added whole (moving them to the end of their entry) rather than patched key by key; haul profiles and facility profiles also record ship, pilot and system names for readability; and re-saving a profile or route keeps its existing name capitalization and notes unless new notes are given. In 2b, structures gain a `kind` (npc_station or upwell); clone bays are proved only from jump clones (reading an owner corporation's structure services is left out, since it needs a character with roles in that corporation, which is rarely the user's for the structures they use); docking checks run only for characters the caller names; and a clone bay status of unknown is stored by leaving `clone_bay` out. In 2c, hubs are keyed by station or structure ID (looked up by name, ID or system name); every pricing tool reads a structure hub's market through a character named in the call or the one that last read it (`read_via`), never by trying characters; `delete_hub_profile` is added; `validate_trade_hub` opens at most the 15 priciest contracts to list ship groups; seeding the built-in hubs is best effort, so loading never fails because a file can't be written.
**Date:** 2026-11-20

## Goal

Make moving cargo, ships and characters around New Eden as code-driven as building already is (`2026-11-18-industry-planning-design.md`). A personal planning repo (`eve-scratchpad`) worked better than neat for hauling because it gave the agent a `CLAUDE.md`, a `haul-plan` skill and hand-maintained YAML reference files (`jump_config.yml`, `hauling_config.yml`, `trade_hubs.yml`, `structures.yml`, `characters.yml`). Those files worked only because the agent was told to read them, and the skill had the agent do arithmetic by hand: fitted cargo holds, Liquid Ozone per cyno light, base fuel per leg, align times, security rounding. This design moves every calculation and game-rule check into Go, gives neat the user-maintained metadata the scratchpad proved useful, and leaves the agent only discovery and decisions.

Moving through New Eden isn't an industry activity. A Black Ops drop, a carrier suitcasing ships to a deployment's staging, a PvE pilot taking a jump clone to a new region and a jump freighter feeding a build all use the same rules for routes, jumps, cynos, docking and cargo. Buying isn't industry either: stocking a doctrine for a deployment walks the same order books as buying build materials. So this design also moves neat's existing movement, cargo and market code out of `industry/` into packages of its own, which `industry/` uses rather than owns.

## Principles

- **Movement and markets are not industry.** Routes, jumps, security, docking, cynos, clone jumps, cargo, order books and trade hubs live in their own packages (see "Packages"). `industry/` imports them for a build's purchases and haul legs, and nothing about moving or buying depends on `industry/`.
- **Code calculates and checks, the agent decides.** Anything that can be expressed as code, especially if it can be tested deterministically, is code: arithmetic, lookups, and yes/no game rules (docking, cyno compatibility, highsec entry, account conflicts, cooldowns). A rule the agent would otherwise have to remember comes back from a tool as a flag or an error. Skills and server instructions hold only what to ask, which tool to call and how to present the result.
- **Tools are how the agent finds things.** Reference data is computed from the SDE and returned by tools whose descriptions match the task, not shipped as files for the agent to be told to read. One overview tool brings together everything a logistics plan starts from.
- **Shared metadata files for what the game doesn't show.** Facts that aren't in the game's data (whether a clone bay is online, what a structure is used for, why a hub is worth checking) live in YAML files under `~/.neat/`. neat keeps them up to date whenever it learns something relevant, and they stay readable and editable by hand, including free-text `notes` fields, so the user can correct anything or add context neat could never find.
- **Static data from `sde/`, live data from ESI.** As in the hard rules. No new third-party data source: hub validation uses ESI, not Fuzzwork.
- **Unknown is not false.** A fact neat can't verify is reported as unknown, and every check treats unknown as "possibly a problem" and flags it, never as the convenient answer.

## Decisions taken (2026-11-20)

| Question | Decision |
|---|---|
| Where does arithmetic live? | In pure, table-tested Go packages split by what they're about, never in skills or agent reasoning. Movement, cargo and markets get their own packages, separate from `industry/`. |
| Account grouping | A label per character in `~/.neat/characters.yaml`. Skills, instructions and tool warnings keep it up to date unprompted. The login page groups characters by account and lets the user create, rename, delete and reassign groups. |
| Structure metadata | `~/.neat/structures.yaml`, modelled on the scratchpad's `structures.yml` and its `notes` fields. neat maintains it, and the user can edit it too. |
| Clone bay status neat can't verify | Stays unknown. The agent asks the user, who verifies it in game; the answer is saved as user-verified with a date. |
| Which saved data is YAML | Any saved data a user benefits from editing by hand and adding context to in `notes`: characters, facility profiles, haul profiles, saved routes, structures and hubs. `config.json` keeps only internal state. |
| Trade hubs | Ship every hub in the scratchpad's `trade_hubs.yml` (five NPC hubs, the two Perimeter markets and the Amamake Keepstar) in a user-editable `~/.neat/hubs.yaml`, plus a validation tool that measures any hub on ESI data and records dated evidence. |

## What moves where

| Scratchpad source | neat home |
|---|---|
| `jump_config.yml`, `hauling_config.yml` | `compare_haul_hulls`, computed from the SDE at call time; docking and cyno rules as sourced constants. |
| `references/cargo-and-volume.md` | `estimate_ship_hold` and `save_haul_profile`'s `shipItemId`; asset volumes on `list_assets`; `list_location_assets`. |
| `references/cynos.md` | `estimate_cyno`. |
| `*_capabilities` in `characters.yml` | `list_character_capabilities`, derived from skills. |
| `account` and `notes` in `characters.yml` | `~/.neat/characters.yaml`. |
| "Jump clones" in `CLAUDE.md` | `list_jump_clones`. |
| "System security" in `CLAUDE.md` | `securityDisplay` and `securityColor` on every system neat returns. |
| `structures.yml` | `~/.neat/structures.yaml` and its tools. Bonuses and job fees stay in facility profiles, which a structure entry links to by name. |
| `trade_hubs.yml` | `~/.neat/hubs.yaml`, `validate_trade_hub`, `compare_trade_hubs`. |
| `jump_routes.yml` | `~/.neat/routes.yaml`: named routes with their notes, which haul profiles and `plan_jump_route` refer to by name. |
| Hull rules in the `haul-plan` skill | Checks inside `plan_jump_route` and `plan_haul`. |
| The `haul-plan` skill | A new plugin skill, `haul-plan`, that absorbs `haul-route-setup` as its preferences step. |

Stays in the user's own repo: corporations, alliances, factions, deployments and anything else about the user's own play, plus the user's own `CLAUDE.md` context. That repo slims down to that and points at neat's tools.

## Character accounts

ESI has nothing that ties characters to an account (the login token's `owner` hash is per character), so the user supplies it.

### Data

Characters move from `config.json` to `~/.neat/characters.yaml` (see "Shared YAML files"), and each gains `account`, a label the user chooses, and `notes`. Characters with the same label share an account. An empty label is **unknown**, which is not the same as "on its own account".

### Tools

- `list_characters` returns each character's `account` and `notes`, with the same unknown-account warning as `add_character`.
- `update_character_details` (new) sets, renames or clears the account label for one or more characters, and sets a character's `notes`. Renaming a label renames it for every character that has it. Writes `characters.yaml` only.
- `remove_character` drops the character with its label, as today.
- `add_character` returns a `warnings` entry naming every logged-in character with no account label, not only the ones just added, whenever two or more characters are logged in. A character neat already knows keeps its label and notes when it's logged in again.

### Login page

The `sso/web/` page lists characters grouped by account, with an "Unassigned" group. The user can move a character between groups, create a group, rename one, or delete one (its characters return to Unassigned). Changes POST to new endpoints protected by the existing page token and Host check, and write `characters.yaml`. A character added in the same session as the one before it is placed in that character's group as a suggestion, highlighted until the user confirms or moves it, because FC's login site usually remembers the signed-in account between back-to-back logins. A suggestion the user never confirms is saved as unassigned, not as the guess.

### Contradiction check

Two characters online at the same moment can't share an account. Where both characters granted the online-status scope, the overview tool compares `get_character_online` results and flags any pair labelled alike that are online together. This catches a wrong label; it can't prove a right one.

### Keeping it current

- The server instructions and the `add-character` skill: after any login, if `add_character` warns about unlabelled characters, ask the user which characters share an account and save the answer, or point them to the login page's grouping.
- The `haul-plan` skill and the overview tool: any plan that needs two characters online together checks their labels, and an unknown label is flagged before pilots or lighters are chosen.

## Shared YAML files

Any saved data a user benefits from editing by hand, and adding context to in `notes`, lives in a YAML file under `~/.neat/`, one file per kind:

| File | Holds | Comes from |
|---|---|---|
| `characters.yaml` | logged-in characters: ID, name, account label, notes | `config.json`'s `characters` |
| `facilities.yaml` | facility profiles, each with notes (for example how a job fee was measured) | `config.json`'s `facilityProfiles` |
| `haul_profiles.yaml` | haul profiles: hull, pilot, hold, preferences, the names of their pinned routes, notes | `config.json`'s `haulProfiles` |
| `routes.yaml` | named jump routes: ship class, waypoints, notes (for example why a route avoids a system, or how much ISK a JF should gate into Jita with) | pinned `outbound`/`return` routes on haul profiles |
| `structures.yaml` | structures the user works from (below) | new |
| `hubs.yaml` | trade hubs (below) | `config.json`'s `hubProfiles` |

Each existing section of `config.json` moves to its file the first time neat loads it, and is removed from `config.json` only after the new file is written. A pinned route becomes a named route in `routes.yaml`, named after its haul profile and direction, and the profile refers to it by name. `plan_jump_route` accepts a `route` name in place of `waypoints`. `config.json` keeps only internal state that nobody edits by hand: `analyticsDisabled` and the built-in hub IDs already offered. `README.md`'s "where data lives" section lists the new files.

`config/` owns every file, through one shared YAML store that the rules below apply to. Tool names don't change.

These files are shared between neat and the user. neat creates and updates them through its tools whenever it has something relevant to record, and the user can open and edit them at any time. The rules every file follows:

- **neat keeps them current.** Any field can be written by neat's tools when it has reason to: refreshing a name or owner that changed in ESI, recording a clone bay it proved, saving validation evidence, or recording what the user said in conversation, such as what a structure is used for, a trait of a hub, or context for `notes`. The skills and server instructions tell the agent to record durable facts as they come up, not to wait to be asked.
- **Targeted updates.** A tool changes only the fields its update concerns, patching the parsed document (using `goccy/go-yaml`'s AST, already in `go.mod`), never rewriting the file from a model. Comments, ordering, and keys neat doesn't know about are preserved, so a user's edits elsewhere in the file survive.
- **Every write is reported.** A write tool returns each field it changed, with the old and new value, so the agent can tell the user what was recorded. When a change overrides a value the user wrote, the old value is in that report. Example: a jump clone proves a clone bay the file says is missing.
- **Hand edits take effect straight away.** neat re-reads the file on every tool call and writes atomically (temp file plus rename, as `config/` does today), re-reading just before it writes so a hand edit made mid-conversation isn't lost.
- **A broken file is never overwritten.** A file that fails to parse makes read tools return an error naming the file and line, and write tools refuse. The user fixes it by hand.
- **Same IDs as the game.** Every `id` is the real EVE ID, so entries can be passed straight to other tools. Names are there for readability.
- **Missing means "doesn't apply" only where the schema says so.** For verifiable facts, the schema uses explicit `unknown` instead.

### `structures.yaml`

One entry per Upwell structure, NPC station or starbase the user works from. Entries are added by `save_structure` whenever a plan uses a location the file doesn't have yet, or by hand.

```yaml
- id: 1049345291823
  name: CX7-70 - Sugar Daddy's Stronghold   # refreshed from ESI
  type: {id: 35827, name: Sotiyo}
  system: {id: 30004072, name: CX7-70}
  owner: {id: 98435559, name: The Blue Donut}
  uses: [manufacturing]
  facility_profile: CX7-70 Sotiyo             # optional link to a facility profile
  clone_bay:
    status: yes          # yes | no | unknown
    source: user         # jump_clone | owner_services | user
    checked: 2026-11-20
  corp_hangars:          # division names come from get_corporation_divisions
    - corporation: {id: 98450125}
      office_id: 1054910829836
  notes: |
    free text, from the user or recorded by neat from what the user said
```

**Clone bay.** neat sets `status: yes` itself when it can prove one: a jump clone of any logged-in character is installed there (`source: jump_clone`), or the owner is a user corp whose structure services neat can read through a character with roles (`source: owner_services`). Otherwise the status stays `unknown`. A clone-jump check against an unknown bay is flagged, the agent asks the user to check in game, and `save_structure` saves the answer with `source: user` and the date. neat never infers a missing bay from a missing clone.

**Docking access.** Not stored. ESI refuses a structure lookup by a character without docking access, so `list_structures` and the planners check it per character, live, when they need it.

**Tools.** `list_structures` returns entries with the live checks (docking access per requested character, clone bay proof), and records any clone bay it proves. `save_structure` adds an entry by ID or refreshes one: it fills IDs, names, type, system and owner from ESI, and takes any field the agent has learned (`uses`, `facility_profile`, `clone_bay` with its source, `notes`). `delete_structure` removes an entry.

### `hubs.yaml`

Replaces the hub profiles in `config.json`. On first load, existing `hubProfiles` move into `hubs.yaml` and are removed from `config.json`. `save_hub_profile` and `list_hub_profiles` keep their names and now read and write this file; `DefaultHub` (Jita 4-4) stays the fallback.

```yaml
- id: 1022167642188
  name: Amamake - 5 times nearly AT winners
  kind: structure          # npc_station | structure
  tier: lowsec             # major | secondary | lowsec
  system: {id: 30002537, name: Amamake}
  region: {id: 10000030, name: Heimatar}
  traits:
    - the place to buy and sell capitals and supercapitals, by contract
  validation:              # written by validate_trade_hub
    date: 2026-11-20
    basket: {stocked: 21, of: 24, sell_orders: 140, buy_orders: 20}
    contracts: {over_1b: 30, over_10b: 7}
  notes: |
    free text, from the user or recorded by neat from what the user said
```

**Built-in hubs.** neat ships the eight hubs from the scratchpad's `trade_hubs.yml` (Jita 4-4, Amarr VIII, Dodixie IX, Hek VIII, Rens VI, the two Perimeter markets and the Amamake Keepstar), with their kind, tier and traits but no validation evidence, since that evidence came from Fuzzwork and neat measures on ESI. They are written into `hubs.yaml` the first time it's created. A later release that adds a built-in hub appends it only if its ID was never offered before (offered IDs are recorded in `config.json`), so a hub the user deleted stays deleted.

**Structure hubs** need a character with docking access and the structure markets scope to read their orders. Tools that use one say which character they read it through, and fail with a clear error if none can.

**Tools.**

- `validate_trade_hub` (new): measures one hub on ESI data and writes its `validation`. It reports how many items of a basket have sell orders at the hub, with sell and buy order counts. The default basket is a named list of type IDs in `market/` covering the scratchpad's categories (minerals, isotopes, fuel blocks, Liquid Ozone, strontium, nanite paste, PLEX, skill injectors, a Venture and a few T2 modules); the caller can pass its own. With `contracts: true` it also counts public contracts at the hub's location by value band and lists the ship groups on offer (capitals trade by contract, so this is the check for a capital market). Evidence carries its date, and other tools mark evidence older than 30 days as stale.
- `compare_trade_hubs` (new): for a list of items and a reference system, ranks every hub in `hubs.yaml` by how much of the list it stocks, the total price on its live order book, and its jump and gate distance from the reference system. This answers "the nearest hub that actually stocks it".

## Packages

Three new flat packages at the repo root, following `industry/`'s rules: pure calculations with no network, disk or clock access, table-tested, with `mcp/` gathering SDE, ESI and config inputs and calling them. Times such as "now" are passed in.

| Package | Responsibility |
|---|---|
| `nav/` | Moving ships and characters through New Eden: security bands, the displayed value and its color, and where a cyno can be lit; light years, jump range, fuel and fatigue; jump and gate route search with pilot preferences and dotlan links; docking rules and highsec entry per hull; cyno types, consumption, cycles, lights per hold and lighter pacing; clone jump availability and clone capacity; movement capabilities from skills (cynos, bridges, conduits, jump-capable hulls); account conflicts between characters a plan needs online together. |
| `cargo/` | What fits where: item volumes (packaged or assembled, with nested contents), fitted holds and special holds from a hull, its fit and the pilot's skills, which assembled ships fit in which bays, rigs lost to repackaging, and packing a cargo list into loads across one or more hulls. |
| `market/` | Buying and selling: walking an order book for a quantity at a station, in-game Multibuy lists, scoring a hub against a basket of items and its public contracts by value band, and ranking hubs for a shopping list by coverage, price and distance (distance passed in from `nav/`). |
| `industry/` | Unchanged in purpose: facilities, jobs, research, invention, reprocessing, the blueprint tree, build-vs-buy, ore mixes and scheduling. It imports `nav/` for security bands, `cargo/` for haul loads and `market/` for order books, and keeps only its own constants. Build-vs-buy and the ore-mix solver stay here because both answer industry questions; both take order books as `market.Order`. |

The dependency runs one way: `industry/` → `nav/`, `cargo/` and `market/`. None of the three imports another, or `industry/`: a route takes the hold and fuel it needs as numbers, and a hub ranking takes distances as numbers.

### Moving the existing code

| From `industry/` | To |
|---|---|
| `security.go`: `Band`, `BandOf`, `DisplaySecurity`, `CynoLegal` | `nav/security.go`. `RigMultiplier` and `ReprocessingSecurityTerm` stay in `industry/`, taking `nav.Band`. |
| `route.go`: route graph, search, validation, dotlan links | `nav/route.go` |
| `haul.go`: `Position`, `LightYears`, `JumpDrive`, `JumpSkills`, `JumpRange`, `FuelPerLY`, `JumpFuel`, `Fatigue` | `nav/jump.go` |
| `haul.go`: `Cargo`, `Load`, `PackLoads` | `cargo/` |
| `haul.go`: `Multibuy`, `MultibuyLine`; `market.go`: `Order`, `Quote`, `WalkBook` | `market/` |
| `constants.go`: `MetersPerLightYear` | `nav/constants.go` |

In `mcp/`, haul profiles, hub profiles, `quote_market_purchase`, `plan_jump_route` and `plan_haul` move out of `mcp/industry_*.go` into their own files, and `industry_register.go` registers only industry tools. The shared SDE and ESI gathering in `mcp/industry_data.go` (jump drives, skills, the route graph, order books) moves with them into a file every planner uses. Tool names don't change, so no client breaks. `tools/gentooldocs` gets navigation and hauling, and trade hubs, as areas of their own, listed before industry, so these tools stop appearing under industry in `docs/reference/tools.md`. Haul profiles stay in `config/`, which already holds every kind of saved state.

## New calculations

Each one is pure and table-tested, with cases taken from the worked figures the scratchpad checked by hand.

| Package | Calculation | Inputs | Test cases |
|---|---|---|---|
| `nav/` | Displayed security and color | true `securityStatus` (rounding already exists as `DisplaySecurity`; color is new) | Aunenen 0.448 → "0.4"; 0.04 → "0.1"; 0.0 → "0.0"; −0.3 → "−0.3"; each displayed value → its in-game color |
| `cargo/` | Fitted hold | hull base capacity (attr 38), hull cargo bonuses and skill levels, Expanded Cargohold multipliers (attr 149), Cargohold Optimization rig bonuses (attr 614), cargo already aboard | Ark, Amarr Freighter IV, 2× Expanded Cargohold II: 135,000 × 1.20 × 1.275² = 263,351.25 m³; cyno Venture, 1× Expanded Cargohold II, 3× Small Cargohold Optimization I: 50 × 1.275 × 1.15³ ≈ 96.96 m³ |
| `cargo/` | Special holds | the same, for each hold's own attributes | ship maintenance bay, fleet hangar, fuel bay per hull |
| `cargo/` | Cargo volume | type `volume`/`packagedVolume`, `isSingleton`, nested contents | a packaged ship uses `packagedVolume`; an assembled ship uses `volume` plus contents for bays |
| `cargo/` | Repackage loss | rigs fitted to an assembled ship, hull group | a T1 hull lists its rigs as lost; a strategic cruiser keeps them |
| `cargo/` | Bay fit | assembled ship volume, bay capacity | which assembled ships fit in which capital's ship maintenance bay |
| `nav/` | Cyno consumption and cycle | generator attrs 714 and 73, Cynosural Field Theory level, hull attr 1296 and 66 | Venture, industrial cyno, CFT IV: 800 × 0.6 × 0.5 = 240 per light; CFT V: 200; Falcon, standard cyno, CFT V: 50 per light, 5-minute cycle |
| `nav/` | Lights per hold and refills | consumption per light, fitted hold, route legs | 242 ozone in a CFT IV Venture is one light; a chain of N legs needs N lights and the refill points between them |
| `nav/` | Cyno type required | hulls in a jump group | JF alone: industrial or standard; JF and carrier together: standard only; Black Ops: any type |
| `nav/` | Highsec entry | hull, from the SDE's `disallowInHighSec` attribute (already read by `industry/facility.go` for structures and rigs) | jump freighters and freighters may gate into highsec; carriers, dreadnoughts, force auxiliaries, supercarriers, titans and the Rorqual may not |
| `nav/` | Docking | hull class, landing structure type or NPC station | a carrier can't dock in an Astrahus or Raitaru; a supercarrier only in a Keepstar |
| `nav/` | Fuel sufficiency | fuel in the fuel bay and cargo, route fuel | shortfall and the leg where a refuel is needed |
| `nav/` | Clone jump availability | `lastCloneJumpDate`, Infomorph Synchronizing level, plan times | IS V gives 19 hours; whether a jump now blocks the planned return |
| `nav/` | Jump clone capacity | Infomorph Psychology, Advanced and Elite levels | one clone per level of each |
| `nav/` | Capabilities | trained skills, hull and module requirements from the SDE | cyno (standard, industrial, covert), bridges (covert, industrial), conduits (covert, industrial, carrier), JF hulls, capital hulls |
| `nav/` | Account conflicts | account labels, plan roles | a pilot and lighter on one label conflict; an unknown label is flagged |
| `nav/` | Lighter pacing | cyno cycle length, the pilot's jump activation cooldown per leg, the lighter's gate count between consecutive cyno points | per-leg figures with one lighter, so the agent weighs numbers when deciding whether to add a second |

## Constants not in the SDE

Each one is a named constant in its package's `constants.go` with its source, per the hard rules. The hard rule in `CLAUDE.md` that names `industry/constants.go` is widened to every calculation package. Before adding any constant, the implementation plan checks the SDE first: highsec entry, for example, comes from `disallowInHighSec` if every relevant hull carries it.

| Package | Constant | Source |
|---|---|---|
| `nav/` | Docking rules per hull class | EVE University, Capital ships#Docking. Black Ops (dock where a battleship can) and command carriers (as carriers) are assumed, since the wiki doesn't cover them; tool output labels them as assumed. |
| `nav/` | Which hull classes can jump to each cyno type | EVE University, Cynosural field |
| `cargo/` | Strategic cruisers keep rigs when repackaged | EVE University, Strategic cruisers |
| `nav/` | Clone jump cooldown: 24 hours minus 1 hour per level of Infomorph Synchronizing | EVE University, Jump clones |
| `nav/` | Jump clone count: 1 per level of Infomorph Psychology, Advanced and Elite Infomorph Psychology | EVE University, Jump clones |
| `nav/` | Security display rounding and the in-game security colors | FC developer guide, System security |
| `market/` | Default hub validation basket | This design |

## New and changed tools

New: `update_character_details`, `save_route`, `list_routes`, `delete_route`, `list_location_assets`, `compare_haul_hulls`, `estimate_ship_hold`, `estimate_cyno`, `list_character_capabilities`, `list_jump_clones`, `get_logistics_overview`, `list_structures`, `save_structure`, `delete_structure`, `validate_trade_hub`, `compare_trade_hubs`.

- `get_logistics_overview` is the entry point for any move. One call returns, per logged-in character, the account label, capabilities, current location and ship, jump clones and next clone jump time, and jump fatigue. It also returns the account conflicts and unknown labels. Characters missing a scope are reported per field, not dropped.
- `list_location_assets` lists every logged-in character's assets at one station or structure, plus corp assets where a character has the roles. Contents are nested under their containers and ships, with packaged and assembled volumes, repackage loss per assembled ship, and totals. It replaces paging `list_assets` into files and querying them with `jq`.

Changed:

- `list_characters` and `add_character` (accounts, as above).
- `list_assets` and `list_corporation_assets` gain additive `volumeM3`, packaged or assembled per `isSingleton`, from the SDE.
- `get_universe_system`, and every system in route and haul output, gain `securityDisplay` and `securityColor`.
- `save_haul_profile` accepts `shipItemId` instead of `holdM3`, and works out the fitted hold from the ship's fit and the pilot's skills. `holdM3` stays as an override.
- `plan_jump_route` takes a ship and pilot directly as well as a haul profile, so a one-off plan needs no throwaway profile. It also checks docking at each landing, highsec entry on gate legs, fuel sufficiency, the cyno type each leg needs and lighter pacing.
- `plan_haul` packs loads across several hulls at once, summing holds and bays, and says which hulls can share a cyno.
- `save_hub_profile` and `list_hub_profiles` move to `hubs.yaml`. `save_hub_profile` also takes `kind`, `tier`, `traits` and `notes`.

New tools get areas in `tools/gentooldocs`, and `docs/reference/tools.md` is regenerated.

## Skills and instructions

- **`haul-plan`** (new plugin skill, replacing `haul-route-setup`). It covers any move, not only industry: PvP deployments and staging moves, Black Ops drops, suitcasing ships, PvE relocations and getting characters into position by jump clone, as well as a build's haul legs. `industry-build-plan` hands off to it for hauling. Steps:
  1. Call `get_logistics_overview` and resolve unknown account labels.
  2. Confirm what's moving, from where and to where.
  3. Call `list_location_assets`.
  4. Ask what to do with each assembled ship (repackage, suitcase or fly), showing the rig losses the tool reports.
  5. Ask about route preferences and politics (the current `haul-route-setup` steps).
  6. Choose pilots and lighters among the valid candidates the tools return, weighing the implants at risk.
  7. Call the planners.
  8. Ask about unknown clone bays, and record those answers and any other durable context the user shares (a structure's use, a hub's character, a note) with `save_structure` or `save_hub_profile`.
  9. Present the plan with the security colors and dotlan links.

  The skill contains no formulas and no game rules a tool already checks.
- **`add-character`** gains the account step.
- **`mcp/instructions.go`** gains a short logistics paragraph for clients without plugin skills: start from `get_logistics_overview`, keep account labels current, never assume security from a name, record durable facts in `structures.yaml` and `hubs.yaml` as they come up, and tell the user what was recorded.

## Phases

Each phase is one PR with its own `VERSION` bump and CHANGELOG entry, and updates `README.md`, `docs/README.md` and `CLAUDE.md` where it changes them.

0. **Split the packages.** Move the existing movement, cargo and market code out of `industry/` into `nav/`, `cargo/` and `market/`, split the `mcp/` files and add the new doc areas, with no change to any tool's behavior. Update `CLAUDE.md`'s package table and hard rules, `docs/README.md`'s package list, and the status line of `2026-11-18-industry-planning-design.md`.
1. **Foundations.** Account labels (stored in `config.json` until phase 2 moves characters to `characters.yaml`; `update_character_details`, `list_characters`, `add_character` warnings), login page grouping, `securityDisplay`/`securityColor`, asset volumes and `list_location_assets`.
2. **Shared YAML files.** The shared YAML store and its rules, moving characters, facility profiles, haul profiles and pinned routes out of `config.json`, `routes.yaml` and `plan_jump_route`'s `route` input, `structures.yaml` and its tools, `hubs.yaml` with migration, the built-in hubs, `validate_trade_hub` and `compare_trade_hubs`.
3. **Readiness.** `list_character_capabilities`, `list_jump_clones`, `compare_haul_hulls` with the docking and cyno constants, and `get_logistics_overview`.
4. **Planning math.** `estimate_ship_hold` and `shipItemId`, `estimate_cyno`, and the `plan_jump_route` and `plan_haul` changes.
5. **Skills.** `haul-plan`, the `add-character` update and the instructions. Then the user's own repo drops what neat now covers.

## Testing

- `nav/`, `cargo/` and `market/` calculations: table tests with the worked figures above, plus edge cases (no skills trained, unknown hull class, empty route).
- The move out of `industry/`: the existing `route_test.go`, `haul_test.go`, `security_test.go` and market tests move with their code and pass unchanged apart from package names, and `plan_build`'s tests pass unchanged, which shows the move changed no behavior.
- YAML files: round-trip tests that a neat write changes only the fields it targets and preserves comments, ordering, unknown keys and every other field; the change report lists each field with its old and new value; a hand edit made between a read and a write survives; a parse error is reported with its line and the file is left untouched; each `config.json` section moves to its YAML file exactly once, and is removed from `config.json` only after the new file is written; pinned routes become named routes the profile still finds; a deleted built-in hub isn't re-added.
- Login page: handler tests for the grouping endpoints with and without the page token, and for the suggestion being saved as unassigned unless confirmed.
- Tools: `mcp/` tests against `sde/sdetest` fixtures and stubbed ESI, as the industry tools do today.

## Resolved questions (2026-11-20)

- **Which saved data is YAML?** Anything a user benefits from editing by hand and annotating in `notes`. See "Shared YAML files".
- **Gate travel time.** Lighter pacing reports gate counts, not minutes, for now. Estimating travel time needs in-system warp distances (stargate positions are in the SDE) and the ship's warp profile. That's deterministic, but it's follow-up work, tracked in https://github.com/rusher2004/new-eden-agent-toolkit/issues/22.
- **Structure services.** neat reads them only where a character with roles in the owning corp has granted the scope. Everywhere else, the user provides and keeps up the facts (clone bay and the rest) in `structures.yaml`, and neat records them when the user gives them.
