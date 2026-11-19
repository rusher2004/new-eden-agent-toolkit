# neat plugin — changelog

## 0.8.0
- **Action needed: change the callback URL on your ESI application** at developers.eveonline.com to `http://127.0.0.1:48211/callback`. Until you do, EVE's login page rejects `add_character` logins with an invalid redirect URI error. Existing characters keep working; refresh tokens don't depend on the callback URL.
- After you log in, the browser tab now lands on a neat page instead of being left on EVE's login page. The page lists your characters, marks the ones just added, and has **Add another character** and **I'm done** buttons. You can also just close the tab.
- `add_character` can add several characters in one call and returns them all: its output is now `{"characters": [{"characterId", "characterName"}, ...]}` instead of a single character. Each character is saved as soon as its login completes.
- The `eveauth-neat://` URL scheme, its macOS helper app, and the hidden `callback-handler` command are removed. `neat mcp` no longer registers anything with the OS or copies itself to `~/.neat/bin/neat` at startup. Leftovers from older versions are harmless and can be deleted by hand: `~/.neat/NeatURLHandler.app` and `~/.neat/bin/` on macOS, the `HKCU\Software\Classes\eveauth-neat` registry key on Windows, `~/.local/share/applications/neat-url-handler.desktop` on Linux.
- Login needs port 48211 on 127.0.0.1 to be free while it runs. If something else is using it, `add_character` fails with an error naming the port.

## 0.7.1
- Prepares the repo for sharing: tests, fixtures, examples and design docs no longer reference specific players' characters, corporations, structures or home systems. The jump-route regression tests now use Jita → PF-QHK, checked against dotlan. The plugin README drops the private-repo install notes.

## 0.7.0
- Industry planning. A new pure `industry/` package does every calculation, and 22 new MCP tools expose it:
  - facility, haul and hub profiles, plus `describe_facility` and `calibrate_facility_profile`;
  - owned blueprints, build skills and stock checks;
  - order-book quotes with in-game Multibuy lists;
  - job fees, research, copying and invention estimates;
  - reprocessing yield and the cheapest compressed-ore mix;
  - jump and gate routes with dotlan links, and haul load packing;
  - `plan_build`, which plans a whole build end to end and saves it under `~/.neat/plans/`.
- Every facility kind (NPC station, Upwell structure, starbase) in every security band is supported. Structure and rig bonuses, which products each rig covers, refinery and implant bonuses, and jump drive attributes all come from the SDE.
- Three skills: `industry-build-plan`, `industry-profile-setup` and `haul-route-setup`. They cover discovery and decisions only; the tools do the math.

## 0.6.0
- **Fixes `get_universe_type` and every `TypeName`/`ShipTypeName` enrichment, both broken against the current SDE.** fuzzwork's dump dropped the `invTypes.radius` column, so every type lookup failed. `get_universe_type` errored outright, and the name fields across all type-enriched tools silently came back empty. `get_universe_type`'s `radius` now comes from EVE Ref's `reference-data` export, fetched once per process alongside the existing `esi-scrape` fetch (about 57 MB more on the first SDE lookup, with a brief memory spike while it's parsed). If that fetch fails, `radius` is absent on a cache hit and populated only when the type is fetched fresh from ESI.
- Adds 14 tools for three ESI tags new since the last sweep. Military Campaigns: `list_military_campaigns`, `get_military_campaign`, `list_military_campaign_objectives`, `get_military_campaign_objective`, `list_character_military_campaign_objectives`, `get_character_military_campaign_objective_participation`. Cosmetics: `list_character_skinr_licenses`, `list_character_skinr_component_licenses`, `get_skinr`. Paragon Hub: `list_paragon_hub_skinr_listings`, `list_character_paragon_hub_skinr_listings`, `list_paragon_hub_skinr_listings_for_alliance`, `list_paragon_hub_skinr_listings_for_corporation`, `list_paragon_hub_skinr_listings_for_character`. SKINR ship types, cosmetic slots, and components get SDE-resolved names.
- `add_character` now also requests `esi.activity.char:read` and `esi.cosmetic.char:read`, needed by the new character-scoped tools. Characters added before this version must be re-added to grant them; existing tools keep working without re-adding.
- `get_status`'s `vip` is now always present (ESI made it required); `players` widened to a 64-bit integer.
- When an ESI response can't be parsed, or an error carries no ESI error detail, the tool error now includes the start of the raw response (up to 512 bytes), so unexpected replies can be diagnosed.
- Targets ESI compatibility date 2026-08-18 (via `goesi-openapi` v0.0.42), up from 2026-07-21.

## 0.5.0
- `sde/` moves from a bbolt-backed cache seeded from CCP's own SDE jsonl export to a directly
  SQL-queried SQLite dump published by fuzzwork.co.uk (`~/.neat/sde/eve.db`, replacing
  `~/.neat/sde/sde.db`, which is left in place and orphaned by this change). No tool-facing
  behavior change other than the three field regressions below — same 24 Universe tools remain
  SDE-backed via the same read-through cache.
- Three tool outputs now always come back absent (previously populated for at least some rows):
  `get_universe_region`'s `description`, `get_universe_station`'s `services`, and
  `list_universe_factions`' `isUnique`. None have a source column in the fuzzwork dump; a
  cache-miss ESI fallback still populates them fresh, only cache hits are affected.

## 0.4.0
- Adds SDE-resolved `*Name` fields (`RaceName`, `BloodlineName`, `FactionName`, `GroupName`)
  alongside the existing bare `*ID` fields across 8 non-Universe MCP files (alliance, character,
  corporation, corporation projects, faction warfare, incursions, killmails, sovereignty). Closes
  out the last of the ID kinds flagged in batch 1's own follow-up list. Names resolve from the
  local SDE cache only, same as batches 1-2's enrichment — no ESI calls are made on this path.
  Purely additive.

## 0.3.0
- Adds SDE-resolved location `*Name` fields (`SystemName`, `RegionName`, `ConstellationName`,
  `StationName`, `PlanetName`, `MoonName`, plus the index-aligned `*Names` list variants) alongside
  the existing bare `*ID` fields across 14 non-Universe MCP files (corporation, industry,
  killmails, market, planetary interaction, structures, sovereignty, fleets, and more). Names
  resolve from the local SDE cache only, same as batch 1's `type_id` enrichment — no ESI calls are
  made on this path. Purely additive.

## 0.2.0
- Adds SDE-resolved `*Name` fields (`TypeName`, `ShipTypeName`, etc.) alongside the existing bare
  `*ID` fields across 42 non-Universe MCP tools (assets, contracts, killmails, market, industry,
  planetary interaction, structures, and more). Names resolve from the local SDE cache only — no
  ESI calls are made on this path at all. Purely additive — no existing field was renamed, removed, or
  changed, and an unresolvable ID simply omits its `*Name` field rather than failing the call.

## 0.1.1
- No plugin-facing change; VERSION bump required by `versiongate` for CI/tooling fixes to
  `tools/versiongate` and `.github/workflows/version-gate.yml` (fail-closed on git errors,
  GITHUB_OUTPUT sanitization, wired the `go generate` no-diff guard into CI).

## 0.1.0
- Initial release: registers the `neat` MCP server (217 tools covering ESI + a local SDE cache) with Claude Code.
