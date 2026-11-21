# neat plugin — changelog

## 0.17.0
- Jump routes now come with checks for the ship flying them: which cyno types it can jump to, where it can dock at each landing (NPC stations it's allowed in and your saved structures it fits, flagging any landing with nowhere to dock, such as a carrier landing next to only an Astrahus), how many gates one cyno alt has to travel between landings, and whether the route's fuel fits the fuel bay.
- Whether a ship can travel into highsec now comes from the game's own data for that hull.
- `plan_haul` can plan several ships flying together with `with`: their holds add up, each load is split between them, every ship is checked against the same route, and it tells you which cyno types they can all share, so one cyno serves the group.

## 0.16.0
- Claude no longer asks you for your cargo hold. New `estimate_ship_hold` works out every hold of a ship the way the game does, from its real fitted modules and rigs and the pilot's skills (an Ark with Amarr Freighter IV and two Expanded Cargohold IIs comes out at 263,351.25 m³), along with what's already in the hold, warp speed, align time and jump range. `save_haul_profile` does the same when given the ship.
- New `estimate_cyno` works out what lighting cynos takes: Liquid Ozone per light and cycle length with the lighter's Cynosural Field Theory and the hull's bonuses, how many lights a full hold covers, and for a whole jump chain the ozone needed and how many refills. It also says whether the generator fits the ship and what skills the lighter still needs.

## 0.15.0
- New `get_logistics_overview` is where Claude starts any move of ships, cargo or characters: for each of your characters in one call, their account, whether they're online, where they are, their ship, what cynos, bridges and conduits they can use, which haulers and capitals they can fly, when they can next clone jump, and their jump fatigue. It also groups your characters by account and points out a label the game contradicts, such as two characters you said share an account being online at once.
- New `compare_haul_hulls` compares ships for hauling from the game's own data: every hold, warp speed and align time, warp core strength, whether it can cloak or nullify bubbles, whether it can enter highsec, where it can dock (a carrier can't dock in an Astrahus; a supercarrier only in a Keepstar), its jump drive and which cynos it can use, and which bridges and conduits it can ride. Name a pilot to see which they can fly and their jump range and fuel.

## 0.14.0
- New `list_character_capabilities` shows what each of your characters can do in moving ships and cargo, worked out from their skills and the game's own data: which cynos they can light (standard, industrial, covert) and on which ships they can fly, which jump bridges and conduit jumps they can open, and which haulers, jump freighters and capitals they can fly. Claude uses it to pick pilots and cyno alts instead of guessing from skill lists.
- New `list_jump_clones` shows every character's jump clones and their implants, their home station, how many jump clones their skills allow, and when each can next clone jump.

## 0.13.0
- Trade hubs now live in `~/.neat/hubs.yaml`, which you can edit, and neat starts it with the five NPC hubs (Jita, Amarr, Dodixie, Hek, Rens), the two player markets in Perimeter and the Amamake Keepstar, the lowsec hub where capitals trade. Delete any you don't use and they stay deleted. Hubs you saved before come first, so your default doesn't change.
- New `validate_trade_hub` measures a hub on live data and records the result with its date: how many of 24 everyday goods it has for sale, its sell and buy order counts, and optionally its public contracts by value with the capital ships on offer. Hubs not checked in the last 30 days are marked stale.
- New `compare_trade_hubs` ranks your hubs for a shopping list: what each can supply in full, the total cost, and how many jumps each is from a system such as your staging system.
- Player structure markets now work everywhere neat prices things, including build plans and ore mixes. neat reads them through a character with docking access and remembers which one, so you only name it once. `delete_hub_profile` removes a hub.

## 0.12.0
- neat now keeps a list of the stations and structures you work from, in `~/.neat/structures.yaml`, with what the game won't tell it: what you use each for (manufacturing, reactions, market, staging...), its facility profile, whether it has a working clone bay, and your notes. New tools `list_structures`, `save_structure` and `delete_structure`. Names, systems and owners are filled in from the game.
- Clone bays: neat records one as confirmed whenever one of your characters has a jump clone there, and checks again each time it lists your structures. Anywhere else it stays unknown until you check in game and tell Claude, since the game only shows a structure's services to its owners. Claude asks rather than guessing either way.
- `list_structures` can also check, live, which of your characters can dock at each Upwell structure.

## 0.11.0
- Your characters, facility profiles and haul profiles now live in YAML files in `~/.neat/` (`characters.yaml`, `facilities.yaml`, `haul_profiles.yaml`) that you can open and edit yourself: fix a value, add notes, or add your own comments and keys. neat keeps them up to date too, but only changes the fields it has a reason to, so your comments, notes and anything else you added stay put. If a file's YAML gets broken, neat tells you which line and won't touch it until it's fixed. Everything moves out of `config.json` automatically the first time this version runs.
- Jump routes can be saved by name with your notes on why they go the way they do, in `~/.neat/routes.yaml`. New tools `save_route`, `list_routes` and `delete_route`; `plan_jump_route` can cost a saved route by name, and haul profiles pin saved routes for their regular runs. Routes pinned on your existing haul profiles become saved routes named after the profile.
- Saving a facility or haul profile again no longer wipes the notes on it.

## 0.10.0
- neat now knows which of your characters share an EVE account. Characters on one account can't be logged in at the same time, which matters when a plan needs two of them online together, such as a jump freighter pilot and their cyno alt. neat's login page groups your characters by account: pick one for each character or start a new one, and rename or delete groups. When you log in several characters in a row, neat suggests putting each in the same account as the one before, and saves it only when you confirm. EVE doesn't tell neat this, so Claude also asks you when an account is unknown.
- Claude can save notes about each character, such as what it's for or what it flies, with the new `update_character_details` tool, which also sets and renames account labels. `list_characters` now shows each character's account and notes.
- New `list_location_assets` lists everything you and your other characters (and your corporation, through a character with the roles) own at one station or structure, in one call, with everything inside each container and ship. It works out every volume for you: each stack as it is now, each assembled ship's assembled and repackaged size, what comes out of a ship to repackage it, and which rigs repackaging would destroy (strategic cruisers keep theirs), with totals for moving the lot.
- `list_assets` and `list_corporation_assets` now give each item's volume.
- Systems now come with their security as the game shows it (Aunenen is "0.4", not 0.448) and its in-game color, in `get_universe_system`, on every leg of a jump route, and in `describe_facility`. Claude quotes the displayed value and never guesses security from a system's name.

## 0.9.2
- Jump routes, hauling, trade hubs and purchase pricing are no longer filed under industry planning. They now have their own sections in neat's tool reference, because moving ships and buying supplies matter for PvP and PvE as much as for building. No tool was renamed and nothing about how they work has changed; this prepares for upcoming logistics features.

## 0.9.1
- Claude now always asks whether you want your corporation's data when you add characters, and suggests the `corporation`, `corporation_finance` and `corporation_industry` groups up front instead of waiting for a corporation tool to fail. It explains that these only show data for characters with the matching in-game roles, and that granting them to your other characters does no harm.
- `structures` is now suggested whenever you want to see your assets or plan industry, so Claude can look up the names of the player structures your items, ships and jobs are in instead of showing only a number.
- When a corporation tool is refused because a character lacks an in-game role, Claude tries one other character in the same corporation or asks you which one has it, rather than trying every character and using up ESI's error limit.
- The `add-character` skill explains how to give a new group to several characters in one login without losing the groups they already have.
- If no browser tab opens when you add a character, you can go to http://127.0.0.1:48211/ yourself. Claude now tells you this before the login starts, and the login now keeps waiting for you there even when neat can't start a browser at all.
- `neat setup` no longer looks stuck while your keychain waits to be unlocked: after 5 seconds it says it's waiting and what to look for. A failed keychain check now ends with a hint for your OS, including what to do on Linux when there's no usable default keyring.
- The README's Troubleshooting section covers a locked or missing Linux keyring, unlock prompts that open on another screen, `neat` missing from PATH in some terminals, and no browser tab opening at login.

## 0.9.0
- neat now sends anonymous usage statistics to Aptabase, so the project can show FC's partner program how many people use it each month and see which features get used. Each launch sends neat's version, which download you're running (for example macOS on `arm64`) and a random ID that's replaced every month. Each tool call sends the tool's name and whether it worked, with no ID that links it to you or your install. Neither includes your characters, your EVE account, what you ask, what a tool was given or returned, or anything about your computer. [PRIVACY.md](../../PRIVACY.md) has the details.
- New command `neat analytics off` turns it all off and deletes the monthly ID. `neat analytics on` and `neat analytics status` turn it back on and show whether it's on.
- Building from source: `app_config.json`'s `clientId` key is now `esiClientId`, and an optional `aptabaseAppKey` sits beside it. Builds without the key send nothing.

## 0.8.4
- Updated for EVE's developer rebranding as Fenris Creations (FC): neat's descriptions, instructions to Claude and documentation now use the new name. Nothing about how neat works has changed.
- The README, the privacy notice and neat's login page now carry the proprietary notice that EVE's Developer License Agreement asks third-party apps to keep.
- The README suggests a new thing to ask: help setting up a PI network with your characters.

## 0.8.3
- neat no longer downloads about 134 MB from EVE Ref every time it starts. It saves the few details it uses from EVE Ref's two archives in `~/.neat/sde/everef.json` and downloads an archive again only when EVE Ref publishes a new version, so the first question about items or systems in each session no longer waits on those downloads. If EVE Ref can't be reached, neat uses the saved copy.

## 0.8.2
- neat now gives every app it connects to a short set of instructions: how to help a new user log in their first character, what to do when a character is missing access, to check with you before tools that change things in EVE, and to leave industry numbers to neat's tools. This matters most in the Claude desktop app and other apps that don't load the plugin's skills.
- The `add-character` skill checks who's already logged in before starting a login, explains the two common login errors, and suggests a few first questions after you add your first character.
- New documentation: the README is rewritten for players (install, connecting to Claude Code or the Claude desktop app, choosing what neat can see, troubleshooting), and `docs/reference/tools.md` lists every tool by area. The plugin and marketplace descriptions no longer quote a stale tool count.

## 0.8.1
- No changes to neat itself. Building from source gains `make install`, which builds `neat` and installs it to `~/.local/bin`, removing the old binary first so macOS doesn't kill the new one on launch.

## 0.8.0
- **Action needed: change the callback URL on your ESI application** at developers.eveonline.com to `http://127.0.0.1:48211/callback`. Until you do, EVE's login page rejects `add_character` logins with an invalid redirect URI error. Existing characters keep working; refresh tokens don't depend on the callback URL.
- After you log in, the browser tab now lands on a neat page instead of being left on EVE's login page. The page lists your characters, marks the ones just added, and has **Add another character** and **I'm done** buttons. You can also just close the tab.
- `add_character` can add several characters in one call and returns them all: its output is now `{"characters": [{"characterId", "characterName"}, ...]}` instead of a single character. Each character is saved as soon as its login completes.
- The `eveauth-neat://` URL scheme, its macOS helper app, and the hidden `callback-handler` command are removed. `neat mcp` no longer registers anything with the OS or copies itself to `~/.neat/bin/neat` at startup. Leftovers from older versions are harmless and can be deleted by hand: `~/.neat/NeatURLHandler.app` and `~/.neat/bin/` on macOS, the `HKCU\Software\Classes\eveauth-neat` registry key on Windows, `~/.local/share/applications/neat-url-handler.desktop` on Linux.
- Login needs port 48211 on 127.0.0.1 to be free while it runs. If something else is using it, `add_character` fails with an error naming the port.
- **Logins now ask for only the access you need.** Requesting all 72 ESI scopes at once made EVE's login page fail with "HTTP Error 400. The size of the request headers is too long" after a couple of logins. Scopes now come in groups: `basic` (skills, location, ship, clones) is always granted, and you choose the others, such as `wallet`, `assets`, `industry`, `market`, `mail` or `corporation`. The new `list_scope_groups` tool lists them, and the new `add-character` skill has Claude ask what you'll use neat for and pick the groups.
- `add_character` takes `scopeGroups` and, to grant more access to a character later, `characterId`; a re-authorization keeps everything the character already granted. Its output now lists the scope groups each character granted, plus `warnings` when a re-authorization didn't go as asked: the user logged in as a different character, or the character's earlier scopes couldn't be read so only the requested groups were granted.
- A tool that needs a scope its character hasn't granted now fails before calling ESI, with an error naming the scope group and how to grant it, instead of an unexplained 403. Characters added before this version keep all their scopes; nothing changes for them.
- The ESI application at developers.eveonline.com still needs every scope enabled, so that any group can be requested.

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
- `sde/` moves from a bbolt-backed cache seeded from FC's own SDE jsonl export to a directly
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
