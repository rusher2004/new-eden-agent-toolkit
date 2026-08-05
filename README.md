# neat — New Eden Agent Toolkit

A Go MCP server, Claude Code plugin, and Claude Desktop extension for interacting with EVE Online
via CCP's ESI API and the EVE Ref Static Data Export (SDE).

Design spec: `docs/superpowers/specs/2026-07-23-foundation-design.md`.
Implementation plans: `docs/superpowers/plans/`.

## Status

Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), the MCP server with its character-management tools (`mcp/`, `neat mcp`),
real ESI-backed domain tools (`esi/`) covering all 36 ESI resource tags (Wallet through Wars), and
a local SDE cache (`sde/`, backed by fuzzwork.co.uk's directly-queried SQLite conversion of
CCP's SDE, narrowly supplemented from EVE Ref's `reference-data`/`esi-scrape` exports, and
backing 24 of the Universe tag's 30 tools via a read-through cache) are built. See
`docs/reference/esi-coverage-roadmap.md` for the full per-tag ESI breakdown.
The Claude Code plugin is available: `/plugin marketplace add rusher2004/new-eden-agent-toolkit`
then `/plugin install neat@neat` registers the `neat` MCP server (requires the `neat` binary on
`PATH` and `neat setup` run once — see `claude_resources/README.md`). The Claude Desktop extension
doesn't exist yet.

## Architecture

Flat package structure at repo root. No `internal/`. One package per responsibility:

| Package | Responsibility |
|---|---|
| `cmd/` | `main.go` entrypoint only |
| `commands/` | `urfave/cli/v3` command tree (`version`, `setup`, `mcp`, `callback-handler` (hidden)) |
| `config/` | Non-secret state in `~/.neat/config.json` |
| `creds/` | OS keychain wrapper (`zalando/go-keyring`), service name `"neat"` |
| `urlscheme/` | Per-OS registration of `eveauth-neat://` as the ESI OAuth redirect handler |
| `sso/` | ESI OAuth2/PKCE login flow: PKCE + state generation, Unix-socket callback handoff, orchestration via `goesi-openapi`, refresh-token persistence via `creds/`. All outbound requests to CCP (JWKS fetch, OAuth2 token exchange) carry `sso.UserAgent(version)`, per ESI's User-Agent best practices. Also caches a per-character auto-refreshing HTTP client for `esi/`'s use (`AuthenticatedClient`/`RemoveCharacter`, atomic with respect to each other) — refreshing relies entirely on `golang.org/x/oauth2`'s token source, including persisting a rotated refresh token back to `creds/` if CCP ever issues one |
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its two hundred twenty-three tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance`, `list_character_wallet_journal`, `list_character_wallet_transactions`, `list_corporation_wallets`, `list_corporation_wallet_journal`, `list_corporation_wallet_transactions`, `list_access_lists`, `get_access_list`, `list_mercenary_tactical_operations`, `get_mercenary_tactical_operation`, `list_raidable_skyhooks`, `list_alliances`, `get_alliance`, `list_alliance_corporations`, `get_alliance_icons`, `list_assets`, `get_asset_locations`, `get_asset_names`, `list_corporation_assets`, `get_corporation_asset_locations`, `get_corporation_asset_names`, `list_calendar_events`, `get_calendar_event`, `list_calendar_event_attendees`, `respond_calendar_event` (this project's first write tool -- verb-first name, description states it changes state, no confirmation parameter), `get_character`, `get_character_affiliations`, `list_character_corporation_history`, `get_character_portrait`, `list_character_agents_research`, `list_character_blueprints`, `get_character_fatigue`, `list_character_medals`, `list_character_notifications`, `list_character_notification_contacts`, `get_character_roles`, `list_character_standings`, `list_character_titles`, `get_character_cspa_cost`, `get_character_clones`, `list_character_implants`, `list_character_contacts`, `list_character_contact_labels`, `list_corporation_contacts`, `list_corporation_contact_labels`, `list_alliance_contacts`, `list_alliance_contact_labels`, `add_character_contacts`, `update_character_contacts`, `remove_character_contacts` (the project's 2nd/3rd/4th write tools — verb-first names, descriptions state they change state on EVE's server), `list_character_contracts`, `list_character_contract_bids`, `list_character_contract_items`, `list_corporation_contracts`, `list_corporation_contract_bids`, `list_corporation_contract_items`, `list_public_contracts`, `list_public_contract_bids`, `list_public_contract_items`, `get_corporation`, `list_corporation_alliance_history`, `get_corporation_icons`, `list_npc_corporations`, `list_corporation_members`, `get_corporation_members_limit`, `list_corporation_member_titles`, `list_corporation_member_tracking`, `list_corporation_roles`, `list_corporation_roles_history`, `list_corporation_titles`, `get_corporation_divisions`, `list_corporation_shareholders`, `list_corporation_standings`, `list_corporation_blueprints`, `list_corporation_container_logs`, `list_corporation_facilities`, `list_corporation_medals`, `list_corporation_medals_issued`, `list_corporation_starbases`, `get_corporation_starbase`, `list_corporation_structures`, `get_corporation_project_contribution`, `list_corporation_project_contributors`, `get_corporation_project_detail`, `list_corporation_projects`, `list_dogma_attributes`, `get_dogma_attribute`, `get_dogma_dynamic_item`, `list_dogma_effects`, `get_dogma_effect`, `get_character_fw_stats`, `get_corporation_fw_stats`, `get_fw_leaderboards`, `get_fw_character_leaderboards`, `get_fw_corporation_leaderboards`, `list_fw_stats`, `list_fw_systems`, `list_fw_wars`, `list_character_fittings`, `create_character_fitting`, `delete_character_fitting`, `get_character_fleet`, `get_fleet`, `update_fleet`, `list_fleet_members`, `invite_fleet_member`, `move_fleet_member`, `kick_fleet_member`, `list_fleet_wings`, `create_fleet_wing`, `rename_fleet_wing`, `delete_fleet_wing`, `create_fleet_squad`, `rename_fleet_squad`, `delete_fleet_squad`, `list_character_freelance_jobs`, `get_character_freelance_job_participation`, `list_corporation_freelance_jobs`, `list_corporation_freelance_job_participants`, `get_freelance_job`, `list_freelance_jobs`, `list_incursions`, `list_character_industry_jobs`, `list_character_mining`, `list_corporation_mining_extractions`, `list_corporation_mining_observers`, `get_corporation_mining_observer_detail`, `list_corporation_industry_jobs`, `list_industry_facilities`, `list_industry_systems`, and `list_insurance_prices`, `list_character_killmails_recent`, `list_corporation_killmails_recent`, `get_killmail`, `get_character_location`, `get_character_online`, `get_character_ship`, `list_character_loyalty_points`, `list_loyalty_store_offers`, `list_character_mail`, `get_character_mail_labels`, `list_character_mail_lists`, `get_character_mail`, `send_character_mail`, `create_character_mail_label`, `update_character_mail`, `delete_character_mail`, `delete_character_mail_label`, `list_character_market_orders`, `list_character_market_order_history`, `list_corporation_market_orders`, `list_corporation_market_order_history`, `list_market_groups`, `get_market_group`, `list_market_prices`, `get_market_history`, `list_market_orders`, `list_market_types`, `list_structure_market_orders`, `get_esi_changelog`, `list_esi_compatibility_dates`, `get_esi_name`, `get_esi_status`, `list_character_planets`, `get_character_planet`, `list_corporation_customs_offices`, `get_planetary_schematic`, `get_route`, `search_character`, `get_character_attributes`, `list_character_skill_queue`, `list_character_skills`, `list_sovereignty_campaigns`, `list_sovereignty_systems`, `get_status`, `list_character_mercenary_dens`, `get_character_mercenary_den`, `list_corporation_skyhooks`, `get_corporation_skyhook`, `list_corporation_sovereignty_hubs`, `get_corporation_sovereignty_hub`, `list_universe_ancestries`, `list_universe_bloodlines`, `list_universe_races`, `list_universe_factions`, `list_universe_categories`, `get_universe_category`, `list_universe_groups`, `get_universe_group`, `list_universe_types`, `get_universe_type`, `list_universe_regions`, `get_universe_region`, `list_universe_constellations`, `get_universe_constellation`, `list_universe_systems`, `get_universe_system`, `get_universe_asteroid_belt`, `get_universe_moon`, `get_universe_planet`, `get_universe_stargate`, `get_universe_star`, `get_universe_station`, `list_universe_graphics`, `get_universe_graphic`, `list_universe_structures`, `get_universe_structure`, `list_universe_system_jumps`, `list_universe_system_kills`, `resolve_universe_ids`, `resolve_universe_names`, `set_autopilot_waypoint`, `open_contract_window`, `open_information_window`, `open_market_details_window`, `open_new_mail_window`, `list_wars`, `get_war`, and `list_war_killmails`. `mcp/enrich.go` holds `resolveTypeNames`, the shared SDE-backed `type_id` → name resolver that populates the `*Name` sibling fields (`TypeName`, `ShipTypeName`, ...) alongside the bare `*ID` fields ESI returns, across 44 tools in 19 non-Universe files (`assets.go`, `contracts.go`, `killmails.go`, `market.go`, `industry.go`, `wallet.go`, ...). It deduplicates IDs per call and drops failed lookups rather than erroring — enrichment is additive, so one bad ID must never fail a tool call over data ESI already returned. Batch 2 extended `mcp/enrich.go` with six sibling resolvers (`resolveSystemNames`/`resolveRegionNames`/`resolveConstellationNames`/`resolveStationNames`/`resolvePlanetNames`/`resolveMoonNames`) covering the same additive-`*Name` pattern for system/region/constellation/station/planet/moon IDs across 14 more files. Batch 3 further extended `mcp/enrich.go` with four sibling resolvers (`resolveRaceNames`/`resolveBloodlineNames`/`resolveFactionNames`/`resolveGroupNames`) covering the same additive-`*Name` pattern for race/bloodline/faction/group IDs across 8 more files, closing out batch 1's originally-flagged follow-up list. The Wallet tag's two wallet-transactions tools (`list_character_wallet_transactions`/`list_corporation_wallet_transactions`) adopt the same `TypeName` enrichment via `resolveTypeNames`, bringing `wallet.go` into the enriched-file set for the first time. |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi`. `NewClient` builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client; `NewPublicClient` builds an unauthenticated client for resource groups where every endpoint is public (e.g. Alliance — no character needed at all). Holds no state of its own. Covers one ESI resource group ("tag") per file (`wallet.go`, `access_list.go`, `activities.go`, `alliance.go`, ...) — see `docs/reference/esi-coverage-roadmap.md` for what's covered and what's left across all 36 tags |
| `sde/` | Local cache of fuzzwork.co.uk's periodically-refreshed SQLite conversion of CCP's Static Data Export (`~/.neat/sde/eve.db`), queried directly via `database/sql` + `modernc.org/sqlite` — no intermediate key-value store. Lazily syncs on first lookup call per `Store` lifetime (at most once per `neat mcp` process run): an ETag-gated download+gunzip+atomic-rename of the dump (sync-fatal on failure) plus a narrow, in-memory-only fetch from EVE Ref's `reference-data`/`esi-scrape` for the three fields (type `PackagedVolume`, race `AllianceID`, graphic `SofDNA`) with no source in the dump — both independently best-effort. `PutX`'s cache-miss write-back lands in a process-lifetime in-memory overlay, not the downloaded file itself, which is treated as read-only. Backs 24 of the Universe tag's 30 tools via a read-through cache (`mcp/universe.go` orchestrates the ESI fallback and write-back; `sde/` itself never imports `esi/`) — the other 6 are permanently ESI-direct by design (live/non-cacheable data). Four fields are permanently unsourced across every dataset checked: `get_universe_graphic`'s `CollisionFile`, `get_universe_region`'s `Description`, `get_universe_station`'s `Services`, and `list_universe_factions`'s `IsUnique` — see `docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md` for how each was found and why none were solved with an additional data source. |

See `CLAUDE.md` for conventions and the doc-sync rule that keeps this file, that one, and the
design spec from drifting apart.

## Build

On macOS, first build the embedded URL-scheme handler (required before `go build`/`go test` will
compile — `go:embed` fails if the path doesn't exist):

```bash
make urlscheme-macos-bundle
```

`clientID` (the ESI OAuth2 client ID) is also injected via `-ldflags` from the git-ignored
`app_config.json` — `make build` wraps this for local development.

```bash
go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev) -X main.clientID=$(jq -r .clientId app_config.json)" -o neat ./cmd
```

## Test

```bash
go test ./...
```

## Setup

```bash
neat setup
```

Creates `~/.neat/config.json` if it doesn't already exist, and verifies the OS keychain is
accessible before you need it for anything credential-related.
