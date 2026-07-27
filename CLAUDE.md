# neat — New Eden Agent Toolkit

Go MCP server + Claude Code plugin + Claude Desktop extension for interacting with EVE Online via
CCP's ESI API and the EVE Ref Static Data Export (SDE). Design spec:
`docs/superpowers/specs/2026-07-23-foundation-design.md`.

## Architecture

Flat package structure at repo root. No `internal/`. One package per responsibility:

| Package | Responsibility |
|---|---|
| `cmd/` | `main.go` entrypoint only |
| `commands/` | `urfave/cli/v3` command tree (`version`, `setup`, `mcp`, `callback-handler` (hidden — invoked only by the OS on URL-scheme activation)) |
| `config/` | Non-secret state in `~/.neat/config.json` |
| `creds/` | OS keychain wrapper (`zalando/go-keyring`), service name `"neat"` |
| `urlscheme/` | Per-OS registration of `eveauth-neat://` as the ESI OAuth redirect handler. macOS embeds a real compiled Swift/Cocoa wrapper (`urlscheme/macos/`, built via `make urlscheme-macos-bundle`) via `go:embed` — a bare executable can't receive macOS's `kAEGetURL` Apple Event. Windows (registry) and Linux (`.desktop` + `xdg-mime`) exist but have no CI execution yet — see "Known gaps" below |
| `sso/` | ESI OAuth2/PKCE login flow: PKCE + state generation, a two-socket Unix IPC handoff for the callback (a lock socket that's bound but never `Accept()`-ed, plus a separate delivery socket), orchestration via `github.com/fnt-eve/goesi-openapi`, refresh-token persistence via `creds/`. All outbound requests to CCP (JWKS fetch, OAuth2 token exchange) carry `sso.UserAgent(version)`, per ESI's User-Agent best practices. Also caches a per-character auto-refreshing HTTP client (`AuthenticatedClient`/`RemoveCharacter`) for `esi/`'s use — refreshing relies entirely on `golang.org/x/oauth2`'s own token source (with rotated refresh tokens persisted back to `creds/` if CCP ever returns one), not hand-rolled expiry tracking. `AuthenticatedClient` and `RemoveCharacter` share one mutex across their entire keychain operation, not just the cache map, so the two can never interleave |
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its two hundred seventeen tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance`, `list_access_lists`, `get_access_list`, `list_mercenary_tactical_operations`, `get_mercenary_tactical_operation`, `list_raidable_skyhooks`, `list_alliances`, `get_alliance`, `list_alliance_corporations`, `get_alliance_icons`, `list_assets`, `get_asset_locations`, `get_asset_names`, `list_corporation_assets`, `get_corporation_asset_locations`, `get_corporation_asset_names`, `list_calendar_events`, `get_calendar_event`, `list_calendar_event_attendees`, `respond_calendar_event` (this project's first write tool -- verb-first name, description states it changes state, no confirmation parameter), `get_character`, `get_character_affiliations`, `list_character_corporation_history`, `get_character_portrait`, `list_character_agents_research`, `list_character_blueprints`, `get_character_fatigue`, `list_character_medals`, `list_character_notifications`, `list_character_notification_contacts`, `get_character_roles`, `list_character_standings`, `list_character_titles`, `get_character_cspa_cost`, `get_character_clones`, `list_character_implants`, `list_character_contacts`, `list_character_contact_labels`, `list_corporation_contacts`, `list_corporation_contact_labels`, `list_alliance_contacts`, `list_alliance_contact_labels`, `add_character_contacts`, `update_character_contacts`, `remove_character_contacts` (the project's 2nd/3rd/4th write tools — verb-first names, descriptions state they change state on EVE's server), `list_character_contracts`, `list_character_contract_bids`, `list_character_contract_items`, `list_corporation_contracts`, `list_corporation_contract_bids`, `list_corporation_contract_items`, `list_public_contracts`, `list_public_contract_bids`, `list_public_contract_items`, `get_corporation`, `list_corporation_alliance_history`, `get_corporation_icons`, `list_npc_corporations`, `list_corporation_members`, `get_corporation_members_limit`, `list_corporation_member_titles`, `list_corporation_member_tracking`, `list_corporation_roles`, `list_corporation_roles_history`, `list_corporation_titles`, `get_corporation_divisions`, `list_corporation_shareholders`, `list_corporation_standings`, `list_corporation_blueprints`, `list_corporation_container_logs`, `list_corporation_facilities`, `list_corporation_medals`, `list_corporation_medals_issued`, `list_corporation_starbases`, `get_corporation_starbase`, `list_corporation_structures`, `get_corporation_project_contribution`, `list_corporation_project_contributors`, `get_corporation_project_detail`, `list_corporation_projects`, `list_dogma_attributes`, `get_dogma_attribute`, `get_dogma_dynamic_item`, `list_dogma_effects`, `get_dogma_effect`, `get_character_fw_stats`, `get_corporation_fw_stats`, `get_fw_leaderboards`, `get_fw_character_leaderboards`, `get_fw_corporation_leaderboards`, `list_fw_stats`, `list_fw_systems`, `list_fw_wars`, `list_character_fittings`, `create_character_fitting`, `delete_character_fitting`, `get_character_fleet`, `get_fleet`, `update_fleet`, `list_fleet_members`, `invite_fleet_member`, `move_fleet_member`, `kick_fleet_member`, `list_fleet_wings`, `create_fleet_wing`, `rename_fleet_wing`, `delete_fleet_wing`, `create_fleet_squad`, `rename_fleet_squad`, `delete_fleet_squad`, `list_character_freelance_jobs`, `get_character_freelance_job_participation`, `list_corporation_freelance_jobs`, `list_corporation_freelance_job_participants`, `get_freelance_job`, `list_freelance_jobs`, `list_incursions`, `list_character_industry_jobs`, `list_character_mining`, `list_corporation_mining_extractions`, `list_corporation_mining_observers`, `get_corporation_mining_observer_detail`, `list_corporation_industry_jobs`, `list_industry_facilities`, `list_industry_systems`, and `list_insurance_prices`, `list_character_killmails_recent`, `list_corporation_killmails_recent`, `get_killmail`, `get_character_location`, `get_character_online`, `get_character_ship`, `list_character_loyalty_points`, `list_loyalty_store_offers`, `list_character_mail`, `get_character_mail_labels`, `list_character_mail_lists`, `get_character_mail`, `send_character_mail`, `create_character_mail_label`, `update_character_mail`, `delete_character_mail`, `delete_character_mail_label`, `list_character_market_orders`, `list_character_market_order_history`, `list_corporation_market_orders`, `list_corporation_market_order_history`, `list_market_groups`, `get_market_group`, `list_market_prices`, `get_market_history`, `list_market_orders`, `list_market_types`, `list_structure_market_orders`, `get_esi_changelog`, `list_esi_compatibility_dates`, `get_esi_name`, `get_esi_status`, `list_character_planets`, `get_character_planet`, `list_corporation_customs_offices`, `get_planetary_schematic`, `get_route`, `search_character`, `get_character_attributes`, `list_character_skill_queue`, `list_character_skills`, `list_sovereignty_campaigns`, `list_sovereignty_systems`, `get_status`, `list_character_mercenary_dens`, `get_character_mercenary_den`, `list_corporation_skyhooks`, `get_corporation_skyhook`, `list_corporation_sovereignty_hubs`, `get_corporation_sovereignty_hub`, `list_universe_ancestries`, `list_universe_bloodlines`, `list_universe_races`, `list_universe_factions`, `list_universe_categories`, `get_universe_category`, `list_universe_groups`, `get_universe_group`, `list_universe_types`, `get_universe_type`, `list_universe_regions`, `get_universe_region`, `list_universe_constellations`, `get_universe_constellation`, `list_universe_systems`, `get_universe_system`, `get_universe_asteroid_belt`, `get_universe_moon`, `get_universe_planet`, `get_universe_stargate`, `get_universe_star`, `get_universe_station`, `list_universe_graphics`, `get_universe_graphic`, `list_universe_structures`, `get_universe_structure`, `list_universe_system_jumps`, `list_universe_system_kills`, `resolve_universe_ids`, `resolve_universe_names`, `set_autopilot_waypoint`, `open_contract_window`, `open_information_window`, `open_market_details_window`, `open_new_mail_window`, `list_wars`, `get_war`, and `list_war_killmails` |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi`. `NewClient` builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client; `NewPublicClient` builds an unauthenticated client for resource groups where every endpoint is public (e.g. Alliance — no character needed at all). Holds no state of its own. Covers one ESI resource group ("tag") per file (`wallet.go`, `access_list.go`, `activities.go`, `alliance.go`, ...) — see `docs/reference/esi-coverage-roadmap.md` for what's covered and what's left across all 36 tags |
| `sde/` | Local cache of CCP's Static Data Export, backed by `go.etcd.io/bbolt` (`~/.neat/sde/sde.db`). Lazily syncs against EVE Ref's `data.everef.net/ccp/sde/` index on first lookup call per `Store` lifetime (at most once per `neat mcp` process run) — a failed sync doesn't error the lookup, it just falls back to whatever's already cached, since SDE data is enrichment, not a hard requirement. Covers `categories`/`groups`/`types` today; the other ~30 SDE resource categories are added only once a specific tool needs them. Nothing in `esi/`/`mcp/` calls into this package yet — that's a separate follow-up plan, starting with the Universe tag's tools currently on the documented ESI-direct exception below |

## Known gaps (tracked, not silently accepted)

- ~~No CI configuration exists in this repo at all.~~ Resolved: `.github/workflows/test.yml` runs
  `go build`/`go vet`/`go test -race` on real `ubuntu-latest`, `macos-latest`, and `windows-latest`
  runners on every push/PR — the Windows and Linux `urlscheme` tests now actually execute for the
  first time, not just cross-compile.
- **Cross-process TOCTOU in `sso.bindExclusive`'s stale-socket recovery path**: if several separate
  `neat` processes race to recover the *same* crashed process's stale lock file at the exact same
  instant, they can still collide (the common two-live-callers case is fixed and tested). Would need
  a PID-tracked lockfile or equivalent to close fully; judged disproportionate to the actual risk
  given `addCharacterMu` already serializes same-process callers.
- **The browser tab never closes/navigates after a successful `add_character` login**, even though
  the character add itself completes correctly. `eveauth-neat://callback` is a custom scheme, not
  HTTP(S), so the tab has no response body to render for it — the OS hands the URL to our registered
  handler and the tab is just left showing CCP's authorize page mid-click. This is standard behavior
  for RFC 8252-style native-app custom-scheme OAuth redirects (same as Slack/Spotify/Zoom's desktop
  login flows) and isn't fixable from `neat`'s side — a "you can close this tab" message would have
  to come from CCP's own `login.eveonline.com` page, which we don't control. Accepted trade-off of
  the custom-scheme-over-loopback-HTTP decision in the foundation design spec; a loopback server
  could serve its own success page, but was rejected for other reasons.

## Hard rules

- **SDE over ESI**: any tool needing a resource that exists in the SDE must read it from `sde/`,
  never call ESI for it. Performance and API-citizenship requirement, not a suggestion.
  Known temporary exception: the Universe tag's reference-data tools (types/groups/categories,
  regions/constellations/systems, celestial detail, ancestries/bloodlines/races/factions/graphics)
  still call ESI directly — `sde/` now exists (categories/groups/types, per the sde-infrastructure
  plan) but no `mcp/`/`esi/` tool has been migrated to consume it yet. This remains a known,
  acknowledged conflict with this rule, not an oversight; migrate the types/groups/categories
  tools first (the only ones `sde/` can currently serve), then expand `sde/`'s coverage and
  migrate the rest incrementally.
- **No environment variables for config/credentials.** Config lives under `~/.neat/`; credentials
  live in the OS keychain only.
- **No client secret.** The ESI app is a PKCE-only public app — nothing secret ships in the binary.
- **ESI User-Agent compliance**: every outbound request to CCP's login/ESI endpoints must carry `sso.UserAgent(version)`. CCP's best-practices guidance treats non-compliant User-Agents as grounds for a temporary ban — see `sso/useragent.go`.
- **ESI error detail must not be discarded**: every `esi/` method that calls `.Execute()` on a generated API request must route its error and `*http.Response` through `describeError` (`esi/error.go`) before wrapping and returning it. `goesi-openapi`'s generated error type degrades to a bare HTTP status string on its own — its message formatter looks for `Title`/`Detail` fields via reflection, but ESI's actual error model uses `Error`/`Details` instead, so the real detail (invalid scope, expired token, etc.) and the `X-Esi-Error-Limit-Remain`/`-Reset` rate-limit headers are silently lost unless routed through this helper. Confirmed via direct reproduction against `goesi-openapi` v0.0.37 — see `esi/wallet.go` for the pattern every new `esi/` method should follow.

## Conventions

- Wrap errors with context: `fmt.Errorf("operation: %w", err)`. Never discard an error (`_ = err`).
- External test packages by default (`package foo_test`); use `*_internal_test.go` only when a
  test genuinely needs an unexported symbol.
- Table-driven where a table naturally fits (many input variations on one code path);
  independent, clearly-named tests otherwise — don't force behaviorally distinct scenarios into
  one table just to have a table. Standard library only, either way — no `testify` or other
  assertion libraries.
- `creds/` tests must call `keyring.MockInit()` before touching keychain functions — never let
  tests hit the real OS keychain.
- Never model a nested/tree-shaped ESI resource with a self-referential Go struct (a type whose
  field is `[]ThatSameType`). It passes `go build`/`go vet`/`gofmt` but panics at server startup
  ("cycle detected for type ...") in the MCP SDK's output-schema generator — only `go test ./...`
  actually calling `mcpsdk.AddTool` catches it. Instead, define one distinct, non-recursive type
  per real nesting level, matching the ESI schema's actual depth (see `mcp/killmails.go`'s
  `KillmailItem`/`KillmailInnerItem` split).

## Keeping docs & Claude resources in sync

`neat` is consumed by both humans and agents through several surfaces that must not drift from the
code. **When a change affects behavior, update the relevant surfaces in the same commit/PR** —
treat it as part of "done," not a follow-up:

- **Packages / architecture** → the package table and conventions in **this `CLAUDE.md`**, and
  `README.md`'s architecture section. Note significant departures from
  `docs/superpowers/specs/2026-07-23-foundation-design.md` in that spec's own status line rather
  than rewriting it — it's a historical record of what was decided, not a living doc.
- **CLI surface** (new command, flag, or exit-code contract) → `README.md`'s Build/Test/usage
  section and this file's package table.
- **Credentials** (new keychain key, service name, or verification flow) → `creds/`,
  `commands/setup.go`, and the Hard rules / Conventions sections here.
- **Hard rules** (SDE-vs-ESI routing, no client secret, no env vars for storage) → keep these
  listed here exactly as they're enforced in code; if a rule is relaxed or tightened, this file
  changes in the same commit as the code that changes it.
- **Agent-facing surfaces that don't exist yet** — once built in a later plan, add them to this
  list: the Claude Code plugin (`.claude-plugin/plugin.json` + `skills/`), the Claude Desktop
  extension (`extension/manifest.json`), and any `reference/*.md` API-contract docs. Don't leave a
  plan's own doc updates (e.g. its plan file, or a spec status line) as a "follow-up" task.

Before opening a PR, sanity-check that `README.md` and this file still describe what the code
actually does.

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
