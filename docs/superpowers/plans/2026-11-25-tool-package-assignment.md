# Tool-to-package assignment

Status: proposal for owner review, 2026-11-25. Input to the next plan (moving areas out of `mcp/`).

Every MCP tool, its target domain package, its CLI command path, the owner flag that selects it where a path serves several operations, and whether it reads or writes. Inputs: the design spec `docs/superpowers/specs/2026-11-25-cli-interface-design.md` ("Domain packages", "The command tree") and the tool list in `commands/testdata/tools.golden.json`. Tool names, schemas and behavior don't change; only where the code lives and how the CLI reaches it.

## Count summary

Verified by script: the table below lists 277 distinct tools, the same 277 names as `commands/testdata/tools.golden.json` (`jq -r '.[].name'`), with none missing, none extra and none repeated. The script also checks that no two operations share a command path and selector, that every shared path has a selector on each variant, and that no path is both a leaf and a group (which `cli.Build` rejects).

| Package | CLI group | Tools | Write |
|---|---|---:|---:|
| `character` | `character` | 16 | 3 |
| `skill` | `skill` | 3 | 0 |
| `clone` | `clone` | 3 | 0 |
| `location` | `location` | 4 | 0 |
| `corporation` | `corporation` | 17 | 0 |
| `alliance` | `alliance` | 4 | 0 |
| `wallet` | `wallet` | 6 | 0 |
| `asset` | `asset` | 7 | 0 |
| `contract` | `contract` | 9 | 0 |
| `trade` | `market` | 17 | 3 |
| `loyalty` | `loyalty` | 2 | 0 |
| `build` | `industry` | 24 | 4 |
| `planet` | `planet` | 4 | 0 |
| `structure` | `structure` | 15 | 3 |
| `universe` | `universe` | 29 | 0 |
| `route` | `route` | 4 | 2 |
| `haul` | `haul` | 6 | 2 |
| `ship` | `ship` | 2 | 0 |
| `logistics` | `logistics` | 3 | 0 |
| `fleet` | `fleet` | 14 | 10 |
| `mail` | `mail` | 10 | 5 |
| `calendar` | `calendar` | 4 | 1 |
| `contact` | `contact` | 9 | 3 |
| `fitting` | `fitting` | 3 | 2 |
| `killmail` | `killmail` | 3 | 0 |
| `war` | `war` | 3 | 0 |
| `factionwarfare` | `faction-warfare` | 8 | 0 |
| `sovereignty` | `sovereignty` | 2 | 0 |
| `militarycampaign` | `military-campaign` | 6 | 0 |
| `freelancejob` | `freelance-job` | 6 | 0 |
| `project` | `project` | 4 | 0 |
| `skinr` | `skinr` | 8 | 0 |
| `accesslist` | `access-list` | 2 | 0 |
| `itemstats` | `dogma` | 5 | 0 |
| `mercenaryden` | `mercenary-den` | 2 | 0 |
| `incursion` | `incursion` | 1 | 0 |
| `insurance` | `insurance` | 1 | 0 |
| `search` | `search` | 1 | 0 |
| `game` | `game` | 5 | 5 |
| `server` | `server` | 5 | 0 |
| **Total** | | **277** | **43** |

Status counts: 151 on a path the approved tree names, 76 on a proposed leaf under a group the tree names without leaves, 50 marked NEEDS DECISION. Kinds: 234 read, 43 write.

Columns: **Command** is the full path. **Owner flag selects** is filled only where several operations share one path, and names the flag that picks this one; `(none)` is the variant run when no owner flag is given. **Other flags** are indicative: the CLI derives them from each input's `json` names (kebab-case, a scalar's trailing `Id` dropped, slices repeatable), and nested inputs go through `--json`. **Kind** follows `operation.Kind`: write means it changes state in the game or under `~/.neat`. **Status**: `tree` is a path the approved tree names; `proposed` is a leaf under a group the tree names without listing leaves (`fleet`, `mail`, `calendar`, ... on the tree's last line); `NEEDS DECISION` doesn't fit the tree, or its package isn't obvious, and carries the nearest proposal.

## Package names that collide with the pure packages

- **`trade`** (working name) for market orders, prices, history, trade hubs and quotes: `market/` is the pure buying package. The CLI group stays `market`; only the Go package is renamed.
- **`build`** (working name) for industry jobs, blueprints, mining, facility profiles and build planning: `industry/` is the pure industry package. The CLI group stays `industry`.
- **`itemstats`** (working name) for dogma attributes, effects and mutated items: `dogma/` is the pure dogma evaluator. The spec lists `dogma` as a domain candidate but names only `market` and `industry` as collisions; this is a third. The CLI group stays `dogma`.

No other proposed package collides with an existing directory. `character` and `universe` already exist as domain packages and grow; `loyalty` is already migrated.

## Commands that merge several operations behind owner flags

Each is one command path whose operations stay separate, with their own schemas and outputs (spec: "Owner flags select between separate operations"). Corporation and alliance variants also take `--character` as the acting character, so selection can't be "exactly one owner flag present": the proposal is that `--corporation`, `--alliance`, `--fleet`, `--region` or `--campaign` selects its variant when present, `--character` alone selects the character variant, and no owner flag selects the `(none)` variant where one exists (otherwise it's a usage error, exit 2).

- `neat wallet balance`: `--character` → `get_wallet_balance`; `--corporation` → `list_corporation_wallets`
- `neat wallet journal`: `--character` → `list_character_wallet_journal`; `--corporation` → `list_corporation_wallet_journal`
- `neat wallet transactions`: `--character` → `list_character_wallet_transactions`; `--corporation` → `list_corporation_wallet_transactions`
- `neat asset list`: `--character` → `list_assets`; `--corporation` → `list_corporation_assets`
- `neat asset locations lookup`: `--character` → `get_asset_locations`; `--corporation` → `get_corporation_asset_locations`
- `neat asset names lookup`: `--character` → `get_asset_names`; `--corporation` → `get_corporation_asset_names`
- `neat contract bids`: `--character` → `list_character_contract_bids`; `--corporation` → `list_corporation_contract_bids`; `(none)` → `list_public_contract_bids`
- `neat contract items`: `--character` → `list_character_contract_items`; `--corporation` → `list_corporation_contract_items`; `(none)` → `list_public_contract_items`
- `neat contract list`: `--character` → `list_character_contracts`; `--corporation` → `list_corporation_contracts`; `--region` → `list_public_contracts`
- `neat market order-history`: `--character` → `list_character_market_order_history`; `--corporation` → `list_corporation_market_order_history`
- `neat market orders`: `--character` → `list_character_market_orders`; `--corporation` → `list_corporation_market_orders`; `--region` → `list_market_orders`
- `neat industry blueprints`: `--character` → `list_character_blueprints`; `--corporation` → `list_corporation_blueprints`
- `neat industry jobs`: `--character` → `list_character_industry_jobs`; `--corporation` → `list_corporation_industry_jobs`
- `neat structure list`: `--corporation` → `list_corporation_structures`; `(none)` → `list_universe_structures`
- `neat fleet get`: `--character` → `get_character_fleet`; `--fleet` → `get_fleet`
- `neat contact labels`: `--character` → `list_character_contact_labels`; `--corporation` → `list_corporation_contact_labels`; `--alliance` → `list_alliance_contact_labels`
- `neat contact list`: `--character` → `list_character_contacts`; `--corporation` → `list_corporation_contacts`; `--alliance` → `list_alliance_contacts`
- `neat killmail list`: `--character` → `list_character_killmails_recent`; `--corporation` → `list_corporation_killmails_recent`
- `neat faction-warfare stats`: `--character` → `get_character_fw_stats`; `--corporation` → `get_corporation_fw_stats`; `(none)` → `list_fw_stats`
- `neat military-campaign objectives`: `--campaign` → `list_military_campaign_objectives`; `--character` → `list_character_military_campaign_objectives`
- `neat freelance-job list`: `--character` → `list_character_freelance_jobs`; `--corporation` → `list_corporation_freelance_jobs`
- `neat skinr listings`: `(none)` → `list_paragon_hub_skinr_listings`; `--character` → `list_paragon_hub_skinr_listings_for_character`; `--corporation` → `list_paragon_hub_skinr_listings_for_corporation`; `--alliance` → `list_paragon_hub_skinr_listings_for_alliance`

Not merges: `find_owned_blueprints`, `check_material_stock` and `plan_build` are single operations whose own input takes a character or a corporation; their flags pass straight through.

## Verb-set note

The spec's rules say verbs come from a fixed set (`list`, `get`, `save`, `delete`, `add`, `update`, `remove`, `create`, `plan`, `estimate`, `check`, `find`, `compare`, `validate`, `lookup`, `resolve`) and that a test rejects any other, but the approved tree itself uses bare resource names as leaves (`loyalty points`, `loyalty store-offers`, `character roles`, `wallet balance`, `market prices`, `skill queue`, ...) and compound leaves (`industry plan-build`, `industry check-skills`, `ship estimate-hold`). The CLI foundation followed the approved tree and doesn't test a verb set (`cli.Paths` has `loyalty points`). This table does the same: bare nouns where the tree uses them, verbs where it does, and the same style for proposed leaves. It also proposes a few leaves that are neither: `mail send`, `calendar respond`, `industry profile describe|calibrate` (in the tree), `game open ...` and `game waypoint`. If a verb test is wanted later, it has to allow bare nouns or the tree changes.

## Open questions

1. **Owner-flag selection rule.** The spec says a merged command takes exactly one owner flag, and also that corporation operations take `--character` too. Proposed above: the more specific owner flag wins, `--character` alone means the character variant. Confirm.
2. **No-owner variants.** Public contract items and bids, public structure IDs, every faction's FW stats and the public SKINR listings are proposed as the `(none)` variant of the owned command. Confirm, or give them their own leaves.
3. **`contract public`.** The tree has a `public` leaf and a `--region` flag. This table puts `list_public_contracts` on `contract list --region` and leaves `public` unused. Alternative: `contract public --region` for the list, with public items and bids still on `contract items|bids` with no owner flag.
4. **`route plan`.** The tree names `route plan` and `route plan-jump`, but only `plan_jump_route` fits there. `get_route` stays `universe route` (tree and spec put gate routes in `universe`). Either drop `route plan`, or move `get_route` to `route plan` in package `route`.
5. **Detail operations under a plural leaf.** The tree has `market groups`, `structure skyhooks|sovereignty-hubs|starbases` and (proposed) `industry mining-observers` and `structure mercenary-dens` with no place for the matching get. Proposed: a singular leaf for the get (`market group`, `structure skyhook`, `structure starbase`, ...). Alternative: turn each into a group with `list|get`, which changes tree leaves.
6. **Selectors outside the spec's owner set.** `--campaign` (military campaign objectives) is used as a selector; the spec lists `--character`, `--corporation`, `--alliance`, `--fleet`, `--region`. `list_war_killmails` is kept out of `killmail list` so `--war` doesn't become one. Confirm `--campaign`, or split the objectives commands.
7. **Write operations named like reads.** `list_structures` records clone bays it finds into `structures.yaml`, so it is `Write`; `validate_trade_hub`, `calibrate_facility_profile` and `plan_build` also write. Keep `list_structures` as a write, or move the recording into its own operation (a behavior change, out of scope for the migration).
8. **`dogma` collision.** `itemstats` is a working name; alternatives are folding attribute and effect definitions into `universe` (they're static data beside types and groups) and keeping mutated items elsewhere.
9. **Mining.** The character mining ledger, corporation mining observers and moon extractions go in `build` because the tree puts `mining` under `industry`. A separate `mining` package would pass the one-sentence test better.
10. **Mercenary dens and tactical operations.** Not in the spec's candidates. Proposed: dens in `structure` (they're deployables, and `mcp/structures.go` already holds them with skyhooks and sovereignty hubs), tactical operations in a new `mercenaryden` package with CLI group `mercenary-den`. Alternative: both in `structure`, or both in `mercenaryden`.
11. **ESI's own metadata.** `get_esi_status`, `get_esi_changelog`, `get_esi_name` and `list_esi_compatibility_dates` describe the data source itself, against the spec's anti-goal of exposing where data comes from. Proposed `server esi status|changelog|name|compatibility-dates`. Alternative: keep them MCP-only, which breaks "every tool is reachable as a CLI command".
12. **Input names that break the acting-character convention.** `list_haul_ships` and `list_location_assets` take `corporationCharacterId` (`--corporation-character`), and the ship and haul tools use `pilotId`, `ownerId`, `lighterId`. MCP schemas can't change, so their CLI flags differ from the corporation tools'. Allow a per-operation flag alias in the `cli/` table, or accept the difference.
13. **Output shape differs across merged variants.** `faction-warfare stats` (one entity, or every faction), `wallet balance` (one balance, or every division) and `fleet get` (membership, or the fleet's settings) return different shapes per variant. The spec allows this; the merged help must show each variant's output.
14. **Tools placed outside the tree's leaves** (all marked NEEDS DECISION below): `character contact-notifications`, `location fatigue`, `corporation member-titles|roles-history|medals-issued|members-limit`, `asset container-logs`, `industry mining-observers|mining-observer|moon-extractions`, `structure raidable-skyhooks|mercenary-dens|mercenary-den`, `logistics inventory` (`list_location_assets`), `mail cspa-cost`, `war killmails`, `freelance-job find`, `skinr posted-listings`, `faction-warfare leaderboard factions|characters|corporations`, `game open ...`, `game waypoint`.
15. **Packages not obvious from the concept** (kept where the tree puts them, flagged in Notes): `list_character_agents_research` (character or build), `list_jump_clones` (clone or logistics), `list_corporation_facilities` (corporation or build), `get_character_cspa_cost` (mail), `get_universe_structure` and `list_universe_structures` (structure, though their MCP names say universe), `list_war_killmails` (war or killmail), `list_raidable_skyhooks` (structure or sovereignty), `list_location_assets` (logistics or asset).

## Assignment by package

### `character` (16)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_characters` | `neat character list` |  |  | read | tree |  |
| `add_character` | `neat character add` |  | --character (optional: re-authorize), --scope-groups | write | tree | Blocks on the browser; progress to stderr (operation.Progress). |
| `remove_character` | `neat character remove` |  | --character | write | tree |  |
| `update_character_details` | `neat character update` |  | --character-ids, --account, --notes, --rename-account | write | tree |  |
| `list_scope_groups` | `neat character scope-groups` |  |  | read | tree | Bare noun leaf. |
| `get_character` | `neat character get` |  | --character (any character; public) | read | tree | Public: --character names a target, not a logged-in owner. |
| `get_character_roles` | `neat character roles` |  | --character | read | tree |  |
| `list_character_titles` | `neat character titles` |  | --character | read | tree |  |
| `list_character_medals` | `neat character medals` |  | --character | read | tree |  |
| `list_character_standings` | `neat character standings` |  | --character | read | tree |  |
| `get_character_portrait` | `neat character portrait` |  | --character (any character; public) | read | tree |  |
| `list_character_notifications` | `neat character notifications` |  | --character | read | tree |  |
| `list_character_notification_contacts` | `neat character contact-notifications` |  | --character | read | NEEDS DECISION | Not in the tree; `character notifications` is a leaf so it can't also be a group. Alternative: `contact notifications` in `contact`. |
| `get_character_affiliations` | `neat character affiliations lookup` |  | --character-ids (public) | read | tree |  |
| `list_character_corporation_history` | `neat character corporation-history` |  | --character (any character; public) | read | tree |  |
| `list_character_agents_research` | `neat character agents-research` |  | --character | read | tree | Package not obvious: R&D agent research feeds datacores for invention (`build`). Kept in `character` per the tree. |

### `skill` (3)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_skills` | `neat skill list` |  | --character | read | tree |  |
| `get_character_attributes` | `neat skill attributes` |  | --character | read | tree |  |
| `list_character_skill_queue` | `neat skill queue` |  | --character | read | tree |  |

### `clone` (3)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_character_clones` | `neat clone list` |  | --character | read | tree |  |
| `list_character_implants` | `neat clone implants` |  | --character | read | tree |  |
| `list_jump_clones` | `neat clone jump-clones` |  | --character-ids | read | tree | Package not obvious: a planning view (jumpFromHere reads structures.yaml and every character's clones, nav clone rules); could sit in `logistics`. Kept in `clone` per the tree. |

### `location` (4)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_character_location` | `neat location get` |  | --character | read | tree |  |
| `get_character_online` | `neat location online` |  | --character | read | tree |  |
| `get_character_ship` | `neat location ship` |  | --character | read | tree |  |
| `get_character_fatigue` | `neat location fatigue` |  | --character | read | NEEDS DECISION | Not in the tree. Jump fatigue is movement state, so `location`; alternatives `character fatigue` or `logistics fatigue`. |

### `corporation` (17)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_corporation` | `neat corporation get` |  | --corporation (public) | read | tree |  |
| `list_corporation_members` | `neat corporation members` |  | --corporation, --character (acting) | read | tree |  |
| `list_corporation_titles` | `neat corporation titles` |  | --corporation, --character (acting) | read | tree |  |
| `list_corporation_member_titles` | `neat corporation member-titles` |  | --corporation, --character (acting) | read | NEEDS DECISION | Not in the tree. |
| `list_corporation_roles` | `neat corporation roles` |  | --corporation, --character (acting) | read | tree |  |
| `list_corporation_roles_history` | `neat corporation roles-history` |  | --corporation, --character (acting), --page | read | NEEDS DECISION | Not in the tree. |
| `list_corporation_standings` | `neat corporation standings` |  | --corporation, --character (acting), --page | read | tree |  |
| `get_corporation_divisions` | `neat corporation divisions` |  | --corporation, --character (acting) | read | tree |  |
| `list_corporation_shareholders` | `neat corporation shareholders` |  | --corporation, --character (acting), --page | read | tree |  |
| `list_corporation_medals` | `neat corporation medals` |  | --corporation, --character (acting), --page | read | tree |  |
| `list_corporation_medals_issued` | `neat corporation medals-issued` |  | --corporation, --character (acting), --page | read | NEEDS DECISION | Not in the tree. |
| `get_corporation_icons` | `neat corporation icons` |  | --corporation (public) | read | tree |  |
| `list_corporation_facilities` | `neat corporation facilities` |  | --corporation, --character (acting) | read | tree | Package not obvious: these are the corporation's industry facilities (`build`), and `industry facilities` already exists for the game-wide list. Kept in `corporation` per the tree. |
| `list_corporation_member_tracking` | `neat corporation member-tracking` |  | --corporation, --character (acting) | read | tree |  |
| `list_corporation_alliance_history` | `neat corporation alliance-history` |  | --corporation (public) | read | tree |  |
| `list_npc_corporations` | `neat corporation npc list` |  |  | read | tree |  |
| `get_corporation_members_limit` | `neat corporation members-limit` |  | --corporation, --character (acting) | read | NEEDS DECISION | Not in the tree. |

### `alliance` (4)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_alliances` | `neat alliance list` |  |  | read | tree |  |
| `get_alliance` | `neat alliance get` |  | --alliance (public) | read | tree |  |
| `list_alliance_corporations` | `neat alliance corporations` |  | --alliance (public) | read | tree |  |
| `get_alliance_icons` | `neat alliance icons` |  | --alliance (public) | read | tree |  |

### `wallet` (6)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_wallet_balance` | `neat wallet balance` | `--character` |  | read | tree |  |
| `list_corporation_wallets` | `neat wallet balance` | `--corporation` | --character (acting) | read | tree | Returns every division's balance; no --division. |
| `list_character_wallet_journal` | `neat wallet journal` | `--character` | --page | read | tree |  |
| `list_corporation_wallet_journal` | `neat wallet journal` | `--corporation` | --character (acting), --division, --page | read | tree |  |
| `list_character_wallet_transactions` | `neat wallet transactions` | `--character` | --from | read | tree |  |
| `list_corporation_wallet_transactions` | `neat wallet transactions` | `--corporation` | --character (acting), --division, --from | read | tree |  |

### `asset` (7)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_assets` | `neat asset list` | `--character` | --page | read | tree |  |
| `list_corporation_assets` | `neat asset list` | `--corporation` | --character (acting), --page | read | tree |  |
| `get_asset_locations` | `neat asset locations lookup` | `--character` | --item-ids | read | tree |  |
| `get_corporation_asset_locations` | `neat asset locations lookup` | `--corporation` | --character (acting), --item-ids | read | tree |  |
| `get_asset_names` | `neat asset names lookup` | `--character` | --item-ids | read | tree |  |
| `get_corporation_asset_names` | `neat asset names lookup` | `--corporation` | --character (acting), --item-ids | read | tree |  |
| `list_corporation_container_logs` | `neat asset container-logs` |  | --corporation, --character (acting), --page | read | NEEDS DECISION | Not in the tree. Container access logs are about who touched assets; alternative `corporation container-logs`. |

### `contract` (9)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_contracts` | `neat contract list` | `--character` | --page | read | tree |  |
| `list_corporation_contracts` | `neat contract list` | `--corporation` | --character (acting), --page | read | tree |  |
| `list_public_contracts` | `neat contract list` | `--region` | --page | read | NEEDS DECISION | The tree also lists a `contract public` leaf; this table puts public contracts on `contract list --region` and leaves `public` unused. See open questions. |
| `list_character_contract_items` | `neat contract items` | `--character` | --contract | read | tree |  |
| `list_corporation_contract_items` | `neat contract items` | `--corporation` | --character (acting), --contract | read | tree |  |
| `list_public_contract_items` | `neat contract items` | `(none)` | --contract, --page | read | NEEDS DECISION | A public contract needs no owner and has no region: selected when no owner flag is given. See open questions. |
| `list_character_contract_bids` | `neat contract bids` | `--character` | --contract | read | tree |  |
| `list_corporation_contract_bids` | `neat contract bids` | `--corporation` | --character (acting), --contract, --page | read | tree |  |
| `list_public_contract_bids` | `neat contract bids` | `(none)` | --contract, --page | read | NEEDS DECISION | As list_public_contract_items. |

### `trade` (17)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_market_orders` | `neat market orders` | `--character` |  | read | tree |  |
| `list_corporation_market_orders` | `neat market orders` | `--corporation` | --character (acting), --page | read | tree |  |
| `list_market_orders` | `neat market orders` | `--region` | --type, --order-type, --page | read | tree |  |
| `list_structure_market_orders` | `neat market structure` |  | --structure, --character (acting: docking access), --page | read | tree |  |
| `list_character_market_order_history` | `neat market order-history` | `--character` | --page | read | tree |  |
| `list_corporation_market_order_history` | `neat market order-history` | `--corporation` | --character (acting), --page | read | tree |  |
| `get_market_history` | `neat market history` |  | --region, --type | read | tree |  |
| `list_market_prices` | `neat market prices` |  |  | read | tree |  |
| `list_market_groups` | `neat market groups` |  |  | read | tree |  |
| `get_market_group` | `neat market group` |  | --market-group | read | NEEDS DECISION | The tree has only `market groups`; the list and the get can't share it with no owner flag between them. Proposed singular leaf for the get (same pattern as structure skyhook/starbase). |
| `list_market_types` | `neat market types` |  | --region, --page | read | tree |  |
| `quote_market_purchase` | `neat market quote` |  | --hub, --character (optional, structure market), --json for items | read | tree |  |
| `list_hub_profiles` | `neat market hub list` |  |  | read | tree | Loading hubs.yaml seeds built-in hubs once (config's documented seeding write); the operation itself is a read. |
| `save_hub_profile` | `neat market hub save` |  | --id, --character, --name, --tier, --notes, --traits | write | tree |  |
| `delete_hub_profile` | `neat market hub delete` |  | --hub | write | tree |  |
| `validate_trade_hub` | `neat market hub validate` |  | --hub, --character, --contracts, --json for basket | write | tree | Write: records the dated validation on the hub in hubs.yaml. |
| `compare_trade_hubs` | `neat market hub compare` |  | --hubs, --from, --character, --json for items | read | tree |  |

### `loyalty` (2)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_loyalty_points` | `neat loyalty points` |  | --character | read | tree | Migrated (cli.Paths). |
| `list_loyalty_store_offers` | `neat loyalty store-offers` |  | --corporation (public) | read | tree | Migrated (cli.Paths). |

### `build` (24)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_industry_jobs` | `neat industry jobs` | `--character` | --include-completed | read | tree |  |
| `list_corporation_industry_jobs` | `neat industry jobs` | `--corporation` | --character (acting), --include-completed, --page | read | tree |  |
| `list_character_mining` | `neat industry mining` |  | --character, --page | read | tree | Package not obvious: mining is not building. Alternative: a `mining` package for the character ledger and the corporation observers and extractions. |
| `list_corporation_mining_observers` | `neat industry mining-observers` |  | --corporation, --character (acting), --page | read | NEEDS DECISION | Not in the tree; `industry mining` is a leaf so it can't be a group. Same package question as list_character_mining. |
| `get_corporation_mining_observer_detail` | `neat industry mining-observer` |  | --corporation, --character (acting), --observer, --page | read | NEEDS DECISION | Not in the tree; singular leaf for the detail. |
| `list_corporation_mining_extractions` | `neat industry moon-extractions` |  | --corporation, --character (acting), --page | read | NEEDS DECISION | Not in the tree. |
| `list_character_blueprints` | `neat industry blueprints` | `--character` | --page | read | tree |  |
| `list_corporation_blueprints` | `neat industry blueprints` | `--corporation` | --character (acting), --page | read | tree |  |
| `list_industry_facilities` | `neat industry facilities` |  |  | read | tree |  |
| `list_industry_systems` | `neat industry systems` |  |  | read | tree |  |
| `plan_build` | `neat industry plan-build` |  | --product, --quantity, --facility, --hub, --character, --corporation, --builder-ids, ...; --json for decisions | write | tree | Write: saves the full plan under ~/.neat/plans/. One operation taking both --character and --corporation (stock owner), not an owner-flag merge. |
| `find_owned_blueprints` | `neat industry find-blueprints` |  | --character \| --corporation (one operation), --product, --blueprint | read | tree | One operation whose input takes either owner; not an owner-flag merge. |
| `check_build_skills` | `neat industry check-skills` |  | --character-ids, --product | read | tree |  |
| `check_material_stock` | `neat industry check-stock` |  | --character \| --corporation (one operation), --location, --division, --json for materials | read | tree | One operation whose input takes either owner. |
| `solve_ore_mix` | `neat industry solve-ore-mix` |  | --facility, --hub, --refiner, --include-moon-ore, ...; --json for minerals | read | tree |  |
| `estimate_job_cost` | `neat industry estimate-job-cost` |  | --product, --facility, --runs, --alpha | read | tree |  |
| `estimate_research` | `neat industry estimate-research` |  | --blueprint, --facility, --character, --kind, ... | read | tree |  |
| `estimate_invention` | `neat industry estimate-invention` |  | --product, --facility, --character, --hub | read | tree |  |
| `estimate_reprocessing_yield` | `neat industry estimate-reprocessing` |  | --character, --facility, --use-implant | read | tree |  |
| `list_facility_profiles` | `neat industry profile list` |  |  | read | tree |  |
| `save_facility_profile` | `neat industry profile save` |  | --name, --kind, --system, ...; --json for rigs and services | write | tree |  |
| `delete_facility_profile` | `neat industry profile delete` |  | --name | write | tree |  |
| `describe_facility` | `neat industry profile describe` |  | --name, --product-name | read | tree | `describe` is not in the fixed verb set. |
| `calibrate_facility_profile` | `neat industry profile calibrate` |  | --name, --job, --character, --corporation | write | tree | Write: saves the backed-out fee rate on the profile. `calibrate` is not in the fixed verb set. |

### `planet` (4)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_planets` | `neat planet list` |  | --character | read | tree |  |
| `get_character_planet` | `neat planet get` |  | --character, --planet | read | tree |  |
| `get_planetary_schematic` | `neat planet schematic` |  | --schematic | read | tree |  |
| `list_corporation_customs_offices` | `neat planet customs-offices` |  | --corporation, --character (acting), --page | read | tree |  |

### `structure` (15)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_corporation_structures` | `neat structure list` | `--corporation` | --character (acting), --page | read | tree |  |
| `list_universe_structures` | `neat structure list` | `(none)` | --filter | read | NEEDS DECISION | Public structure IDs. Proposed as the no-owner variant of `structure list`; alternative `universe structure list`. Its MCP name says universe; it moves by concept. |
| `get_universe_structure` | `neat structure get` |  | --structure, --character (acting: docking access) | read | tree | MCP name says universe; the tree and the spec put Upwell structures in `structure`. |
| `list_corporation_skyhooks` | `neat structure skyhooks` |  | --corporation, --character (acting) | read | tree |  |
| `get_corporation_skyhook` | `neat structure skyhook` |  | --corporation, --character (acting), --skyhook | read | NEEDS DECISION | The tree has only the list leaf; singular leaf for the detail. |
| `list_raidable_skyhooks` | `neat structure raidable-skyhooks` |  | --character | read | NEEDS DECISION | Not in the tree. Alternative: `sovereignty raidable-skyhooks` (raids are a sovereignty mechanic). |
| `list_corporation_sovereignty_hubs` | `neat structure sovereignty-hubs` |  | --corporation, --character (acting) | read | tree |  |
| `get_corporation_sovereignty_hub` | `neat structure sovereignty-hub` |  | --corporation, --character (acting), --sovereignty-hub | read | NEEDS DECISION | Singular leaf for the detail. |
| `list_corporation_starbases` | `neat structure starbases` |  | --corporation, --character (acting), --page | read | tree |  |
| `get_corporation_starbase` | `neat structure starbase` |  | --corporation, --character (acting), --starbase, --system | read | NEEDS DECISION | Singular leaf for the detail. |
| `list_character_mercenary_dens` | `neat structure mercenary-dens` |  | --character | read | NEEDS DECISION | Not in the tree. Mercenary dens are deployable structures (today in mcp/structures.go). See mercenary tactical operations below. |
| `get_character_mercenary_den` | `neat structure mercenary-den` |  | --character, --mercenary-den | read | NEEDS DECISION | Singular leaf for the detail. |
| `list_structures` | `neat structure saved list` |  | --character-ids | write | tree | Write: records a clone bay as yes/jump_clone in structures.yaml when a jump clone is found there, and reports the change. A list that writes; see open questions. |
| `save_structure` | `neat structure saved save` |  | --id, --character, --uses, --facility-profile, --clone-bay, --notes | write | tree |  |
| `delete_structure` | `neat structure saved delete` |  | --id | write | tree |  |

### `universe` (29)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_universe_asteroid_belt` | `neat universe asteroid-belt get` |  | --asteroid-belt | read | tree |  |
| `get_universe_category` | `neat universe category get` |  | --category | read | tree |  |
| `list_universe_categories` | `neat universe category list` |  |  | read | tree |  |
| `get_universe_constellation` | `neat universe constellation get` |  | --constellation | read | tree |  |
| `list_universe_constellations` | `neat universe constellation list` |  |  | read | tree |  |
| `get_universe_graphic` | `neat universe graphic get` |  | --graphic | read | tree |  |
| `list_universe_graphics` | `neat universe graphic list` |  |  | read | tree |  |
| `get_universe_group` | `neat universe group get` |  | --group | read | tree |  |
| `list_universe_groups` | `neat universe group list` |  | --page | read | tree |  |
| `get_universe_moon` | `neat universe moon get` |  | --moon | read | tree |  |
| `get_universe_planet` | `neat universe planet get` |  | --planet | read | tree | Celestial planet; `planet get` is a character's colony. |
| `get_universe_region` | `neat universe region get` |  | --region | read | tree |  |
| `list_universe_regions` | `neat universe region list` |  |  | read | tree |  |
| `get_universe_star` | `neat universe star get` |  | --star | read | tree |  |
| `get_universe_stargate` | `neat universe stargate get` |  | --stargate | read | tree |  |
| `get_universe_station` | `neat universe station get` |  | --station | read | tree |  |
| `get_universe_system` | `neat universe system get` |  | --system | read | tree |  |
| `list_universe_systems` | `neat universe system list` |  |  | read | tree |  |
| `list_universe_system_jumps` | `neat universe system jumps` |  |  | read | proposed | Live activity counters (ESI-direct by design); leaf under the system group. |
| `list_universe_system_kills` | `neat universe system kills` |  |  | read | proposed | As list_universe_system_jumps. |
| `get_universe_type` | `neat universe type get` |  | --type | read | tree |  |
| `list_universe_types` | `neat universe type list` |  | --page | read | tree |  |
| `list_universe_ancestries` | `neat universe ancestry list` |  |  | read | tree |  |
| `list_universe_bloodlines` | `neat universe bloodline list` |  |  | read | tree |  |
| `list_universe_factions` | `neat universe faction list` |  |  | read | tree |  |
| `list_universe_races` | `neat universe race list` |  |  | read | tree |  |
| `resolve_universe_ids` | `neat universe ids resolve` |  | --names | read | tree |  |
| `resolve_universe_names` | `neat universe names resolve` |  | --ids | read | tree |  |
| `get_route` | `neat universe route` |  | --origin-system, --destination-system, --preference, --avoid-systems, --security-penalty; --json for connections | read | tree | The tree also has `route plan`, which no tool fills; see open questions. |

### `route` (4)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `plan_jump_route` | `neat route plan-jump` |  | --from, --to, --haul-profile, --route, --waypoints; --json for prefs | read | tree |  |
| `list_routes` | `neat route saved list` |  |  | read | tree |  |
| `save_route` | `neat route saved save` |  | --name, --systems, --ship-class, --notes | write | tree |  |
| `delete_route` | `neat route saved delete` |  | --name | write | tree |  |

### `haul` (6)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `plan_haul` | `neat haul plan` |  | --from, --to, --haul-profile, --hub, --with; --json for items | read | tree |  |
| `list_haul_ships` | `neat haul ships` |  | --character-ids, --classes, --corporation, --corporation-character | read | tree | Its acting corporation character is `corporationCharacterId` (--corporation-character), not --character: input names differ from the corporation tools' convention. |
| `compare_haul_hulls` | `neat haul compare-hulls` |  | --hulls, --classes, --pilot | read | tree |  |
| `list_haul_profiles` | `neat haul profile list` |  |  | read | tree |  |
| `save_haul_profile` | `neat haul profile save` |  | --name, --ship, --ship-item, --pilot, --hold-m3, ...; --json for prefs | write | tree |  |
| `delete_haul_profile` | `neat haul profile delete` |  | --name | write | tree |  |

### `ship` (2)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `estimate_ship_hold` | `neat ship estimate-hold` |  | --ship-item, --owner, --ship-type, --modules, --pilot | read | tree |  |
| `estimate_cyno` | `neat ship estimate-cyno` |  | --ship-item, --owner, --ship-type, --modules, --lighter, --cyno, --lights | read | tree |  |

### `logistics` (3)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_logistics_overview` | `neat logistics overview` |  | --character-ids | read | tree |  |
| `list_character_capabilities` | `neat logistics capabilities` |  | --character-ids, --include-missing | read | tree |  |
| `list_location_assets` | `neat logistics inventory` |  | --location, --character-ids, --corporation, --corporation-character | read | NEEDS DECISION | Not in the tree. Everything owned at one station, laid out for a move (cargo.Inventory); alternative `asset at-location` in `asset`. |

### `fleet` (14)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_character_fleet` | `neat fleet get` | `--character` |  | read | proposed | The character's own fleet membership (fleet ID, role). |
| `get_fleet` | `neat fleet get` | `--fleet` | --character (acting: fleet member) | read | proposed | Both variants take --character; --fleet selects this one. |
| `update_fleet` | `neat fleet update` |  | --fleet, --character (acting), --motd, --is-free-move | write | proposed |  |
| `list_fleet_members` | `neat fleet member list` |  | --fleet, --character (acting) | read | proposed |  |
| `invite_fleet_member` | `neat fleet member add` |  | --fleet, --character (acting), --invite-character, --role, --wing, --squad | write | proposed | Invite mapped to the fixed verb `add`. |
| `kick_fleet_member` | `neat fleet member remove` |  | --fleet, --character (acting), --member | write | proposed | Kick mapped to `remove`. |
| `move_fleet_member` | `neat fleet member update` |  | --fleet, --character (acting), --member, --role, --wing, --squad | write | proposed | Move mapped to `update`. |
| `list_fleet_wings` | `neat fleet wing list` |  | --fleet, --character (acting) | read | proposed | Returns wings with their squads nested. |
| `create_fleet_wing` | `neat fleet wing create` |  | --fleet, --character (acting) | write | proposed |  |
| `delete_fleet_wing` | `neat fleet wing delete` |  | --fleet, --character (acting), --wing | write | proposed |  |
| `rename_fleet_wing` | `neat fleet wing update` |  | --fleet, --character (acting), --wing, --name | write | proposed | Rename mapped to `update`. |
| `create_fleet_squad` | `neat fleet squad create` |  | --fleet, --character (acting), --wing | write | proposed |  |
| `delete_fleet_squad` | `neat fleet squad delete` |  | --fleet, --character (acting), --squad | write | proposed |  |
| `rename_fleet_squad` | `neat fleet squad update` |  | --fleet, --character (acting), --squad, --name | write | proposed | Rename mapped to `update`. |

### `mail` (10)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_mail` | `neat mail list` |  | --character, --label-ids, --last-mail | read | proposed |  |
| `get_character_mail` | `neat mail get` |  | --character, --mail | read | proposed |  |
| `send_character_mail` | `neat mail send` |  | --character, --subject, --body, --approved-cost; --json for recipients | write | NEEDS DECISION | `send` is not in the fixed verb set; `create` reads poorly for mail. See the verb-set note. |
| `update_character_mail` | `neat mail update` |  | --character, --mail, --read, --labels | write | proposed |  |
| `delete_character_mail` | `neat mail delete` |  | --character, --mail | write | proposed |  |
| `get_character_mail_labels` | `neat mail label list` |  | --character | read | proposed | A get returning every label with unread counts, mapped to `list`. |
| `create_character_mail_label` | `neat mail label create` |  | --character, --name, --color | write | proposed |  |
| `delete_character_mail_label` | `neat mail label delete` |  | --character, --label | write | proposed |  |
| `list_character_mail_lists` | `neat mail mailing-lists` |  | --character | read | proposed |  |
| `get_character_cspa_cost` | `neat mail cspa-cost` |  | --character, --target-character-ids | read | NEEDS DECISION | Package not obvious (ESI's Character tag). CSPA is the charge for mailing or messaging a character, and send_character_mail takes approvedCost, so `mail`. |

### `calendar` (4)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_calendar_events` | `neat calendar list` |  | --character | read | proposed |  |
| `get_calendar_event` | `neat calendar get` |  | --character, --event | read | proposed |  |
| `list_calendar_event_attendees` | `neat calendar attendees` |  | --character, --event | read | proposed |  |
| `respond_calendar_event` | `neat calendar respond` |  | --character, --event, --response | write | NEEDS DECISION | `respond` is not in the fixed verb set; `update` would hide what it does. |

### `contact` (9)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_contacts` | `neat contact list` | `--character` | --page | read | proposed |  |
| `list_corporation_contacts` | `neat contact list` | `--corporation` | --character (acting), --page | read | proposed |  |
| `list_alliance_contacts` | `neat contact list` | `--alliance` | --character (acting), --page | read | proposed |  |
| `list_character_contact_labels` | `neat contact labels` | `--character` |  | read | proposed |  |
| `list_corporation_contact_labels` | `neat contact labels` | `--corporation` | --character (acting) | read | proposed |  |
| `list_alliance_contact_labels` | `neat contact labels` | `--alliance` | --character (acting) | read | proposed |  |
| `add_character_contacts` | `neat contact add` |  | --character, --contact-ids, --standing, --label-ids, --watched | write | proposed |  |
| `update_character_contacts` | `neat contact update` |  | --character, --contact-ids, --standing, --label-ids, --watched | write | proposed |  |
| `remove_character_contacts` | `neat contact remove` |  | --character, --contact-ids | write | proposed |  |

### `fitting` (3)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_character_fittings` | `neat fitting list` |  | --character | read | proposed |  |
| `create_character_fitting` | `neat fitting create` |  | --character, --name, --description, --ship-type; --json for items | write | proposed |  |
| `delete_character_fitting` | `neat fitting delete` |  | --character, --fitting | write | proposed |  |

### `killmail` (3)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_killmail` | `neat killmail get` |  | --killmail, --killmail-hash (public) | read | proposed |  |
| `list_character_killmails_recent` | `neat killmail list` | `--character` | --page | read | proposed |  |
| `list_corporation_killmails_recent` | `neat killmail list` | `--corporation` | --character (acting), --page | read | proposed |  |

### `war` (3)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_wars` | `neat war list` |  | --max-war | read | proposed |  |
| `get_war` | `neat war get` |  | --war | read | proposed |  |
| `list_war_killmails` | `neat war killmails` |  | --war, --page | read | NEEDS DECISION | Package not obvious: killmail references, like killmail list. Alternative `killmail list --war`, which would make --war an owner flag the spec doesn't list. |

### `factionwarfare` (8)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_character_fw_stats` | `neat faction-warfare stats` | `--character` |  | read | proposed |  |
| `get_corporation_fw_stats` | `neat faction-warfare stats` | `--corporation` | --character (acting) | read | proposed |  |
| `list_fw_stats` | `neat faction-warfare stats` | `(none)` |  | read | NEEDS DECISION | Every faction's stats: the no-owner variant. Output shapes differ by variant (one entity vs all factions). |
| `get_fw_leaderboards` | `neat faction-warfare leaderboard factions` |  |  | read | NEEDS DECISION | The three leaderboards take no input (character/corporation name what is ranked, not an owner), so owner flags can't split them. Proposed subgroup. |
| `get_fw_character_leaderboards` | `neat faction-warfare leaderboard characters` |  |  | read | NEEDS DECISION | As get_fw_leaderboards. |
| `get_fw_corporation_leaderboards` | `neat faction-warfare leaderboard corporations` |  |  | read | NEEDS DECISION | As get_fw_leaderboards. |
| `list_fw_systems` | `neat faction-warfare systems` |  |  | read | proposed |  |
| `list_fw_wars` | `neat faction-warfare wars` |  |  | read | proposed |  |

### `sovereignty` (2)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_sovereignty_campaigns` | `neat sovereignty campaigns` |  |  | read | proposed |  |
| `list_sovereignty_systems` | `neat sovereignty systems` |  |  | read | proposed |  |

### `militarycampaign` (6)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_military_campaigns` | `neat military-campaign list` |  |  | read | proposed |  |
| `get_military_campaign` | `neat military-campaign get` |  | --campaign | read | proposed |  |
| `list_military_campaign_objectives` | `neat military-campaign objectives` | `--campaign` | --after, --before, --limit | read | NEEDS DECISION | Merged with the character's objectives; --campaign is not one of the spec's owner flags. |
| `list_character_military_campaign_objectives` | `neat military-campaign objectives` | `--character` | --after, --before, --limit | read | NEEDS DECISION | As list_military_campaign_objectives. |
| `get_military_campaign_objective` | `neat military-campaign objective` |  | --campaign, --objective | read | proposed |  |
| `get_character_military_campaign_objective_participation` | `neat military-campaign participation` |  | --character, --objective | read | proposed |  |

### `freelancejob` (6)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_freelance_jobs` | `neat freelance-job find` |  | --corporation (filter), --after, --before, --limit | read | NEEDS DECISION | Public browse whose --corporation is a filter, not an owner; on `freelance-job list` it would collide with the corporation variant. Proposed `find`. |
| `list_character_freelance_jobs` | `neat freelance-job list` | `--character` |  | read | proposed |  |
| `list_corporation_freelance_jobs` | `neat freelance-job list` | `--corporation` | --character (acting), --after, --before, --limit | read | proposed |  |
| `get_freelance_job` | `neat freelance-job get` |  | --job (public) | read | proposed |  |
| `get_character_freelance_job_participation` | `neat freelance-job participation` |  | --character, --job | read | proposed |  |
| `list_corporation_freelance_job_participants` | `neat freelance-job participants` |  | --corporation, --character (acting), --job, --after, --before, --limit | read | proposed |  |

### `project` (4)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_corporation_projects` | `neat project list` |  | --corporation, --character (acting), --state, --after, --before, --limit | read | proposed |  |
| `get_corporation_project_detail` | `neat project get` |  | --corporation, --character (acting), --project | read | proposed |  |
| `list_corporation_project_contributors` | `neat project contributors` |  | --corporation, --character (acting), --project, --after, --before, --limit | read | proposed |  |
| `get_corporation_project_contribution` | `neat project contribution` |  | --corporation, --character (acting), --project, --target-character | read | proposed |  |

### `skinr` (8)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_skinr` | `neat skinr get` |  | --skinr (public) | read | proposed |  |
| `list_character_skinr_licenses` | `neat skinr licenses` |  | --character | read | proposed |  |
| `list_character_skinr_component_licenses` | `neat skinr component-licenses` |  | --character | read | proposed |  |
| `list_paragon_hub_skinr_listings` | `neat skinr listings` | `(none)` | --after, --before, --limit | read | proposed | Paragon Hub (an ESI tag) folds into `skinr` by concept. |
| `list_paragon_hub_skinr_listings_for_character` | `neat skinr listings` | `--character` | --after, --before, --limit | read | proposed | Listings other sellers targeted at the character. |
| `list_paragon_hub_skinr_listings_for_corporation` | `neat skinr listings` | `--corporation` | --character (acting), --after, --before, --limit | read | proposed |  |
| `list_paragon_hub_skinr_listings_for_alliance` | `neat skinr listings` | `--alliance` | --character (acting), --after, --before, --limit | read | proposed |  |
| `list_character_paragon_hub_skinr_listings` | `neat skinr posted-listings` |  | --character, --after, --before, --limit | read | NEEDS DECISION | The listings the character posted; can't share `skinr listings --character` with the listings targeted at it. |

### `accesslist` (2)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_access_lists` | `neat access-list list` |  | --character | read | proposed |  |
| `get_access_list` | `neat access-list get` |  | --character, --access-list | read | proposed |  |

### `itemstats` (5)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_dogma_attributes` | `neat dogma attribute list` |  |  | read | NEEDS DECISION | Package name collides with pure `dogma/`; working name `itemstats`. CLI group `dogma` is free. |
| `get_dogma_attribute` | `neat dogma attribute get` |  | --attribute | read | NEEDS DECISION | As list_dogma_attributes. |
| `list_dogma_effects` | `neat dogma effect list` |  |  | read | NEEDS DECISION | As list_dogma_attributes. |
| `get_dogma_effect` | `neat dogma effect get` |  | --effect | read | NEEDS DECISION | As list_dogma_attributes. |
| `get_dogma_dynamic_item` | `neat dogma dynamic-item get` |  | --type, --item | read | NEEDS DECISION | As list_dogma_attributes. A mutated (abyssal) item's rolled stats. |

### `mercenaryden` (2)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_mercenary_tactical_operations` | `neat mercenary-den operations` |  | --character | read | NEEDS DECISION | Not a spec candidate. Tactical operations are run from mercenary dens; proposed new package `mercenaryden` (alternative: dens and operations both in `structure`). |
| `get_mercenary_tactical_operation` | `neat mercenary-den operation` |  | --character, --operation | read | NEEDS DECISION | As list_mercenary_tactical_operations. |

### `incursion` (1)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_incursions` | `neat incursion list` |  |  | read | proposed |  |

### `insurance` (1)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `list_insurance_prices` | `neat insurance prices` |  |  | read | proposed |  |

### `search` (1)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `search_character` | `neat search find` |  | --character, --search, --categories, --strict | read | proposed |  |

### `game` (5)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `open_contract_window` | `neat game open contract` |  | --character, --contract | write | NEEDS DECISION | `open` is not in the fixed verb set; proposed subgroup `game open`. |
| `open_information_window` | `neat game open information` |  | --character, --target | write | NEEDS DECISION | As open_contract_window. |
| `open_market_details_window` | `neat game open market-details` |  | --character, --type | write | NEEDS DECISION | As open_contract_window. |
| `open_new_mail_window` | `neat game open new-mail` |  | --character, --subject, --body, --to-corp-or-alliance, --to-mailing-list; --json for recipients | write | NEEDS DECISION | As open_contract_window. |
| `set_autopilot_waypoint` | `neat game waypoint` |  | --character, --destination, --add-to-beginning, --clear-other-waypoints | write | NEEDS DECISION | `set` is not in the fixed verb set. |

### `server` (5)

| Tool | Command | Owner flag selects | Other flags | Kind | Status | Notes |
|---|---|---|---|---|---|---|
| `get_status` | `neat server status` |  |  | read | proposed |  |
| `get_esi_status` | `neat server esi status` |  |  | read | NEEDS DECISION | These four describe ESI itself; the spec's anti-goal is exposing the data source, which these can't avoid. Proposed subgroup `server esi`. |
| `get_esi_changelog` | `neat server esi changelog` |  |  | read | NEEDS DECISION | As get_esi_status. |
| `get_esi_name` | `neat server esi name` |  |  | read | NEEDS DECISION | As get_esi_status. |
| `list_esi_compatibility_dates` | `neat server esi compatibility-dates` |  |  | read | NEEDS DECISION | As get_esi_status. |
