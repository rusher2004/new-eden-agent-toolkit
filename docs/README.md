# Developing neat

This is the guide for working on neat itself. For installing and using it, see the [main README](../README.md). `CLAUDE.md` at the repo root holds the full package-by-package detail, hard rules and conventions; it's written for Claude Code but is the most complete reference for humans too.

## How it fits together

neat is one Go binary. `neat mcp` runs an MCP server over stdio that an AI app (Claude Code, the Claude desktop app, or another MCP client) starts and talks to. Each tool call goes one of three ways:

- **Character and corporation data** goes to Fenris Creations' (FC's) ESI API, authenticated as a logged-in character.
- **Static game data** (items, systems, blueprints, dogma) is read from a local SQLite copy of FC's Static Data Export, downloaded from fuzzwork.co.uk on first use and refreshed when it changes. ESI is only a fallback for IDs newer than the dump.
- **Industry, hauling and routing plans** are computed locally by a pure calculation package, from SDE data, ESI data and profiles the user has saved.

Logins use EVE's SSO with OAuth2 PKCE, so the binary carries a public client ID and no secret. While `add_character` runs, neat serves the OAuth2 callback and a small character page on `http://127.0.0.1:48211`. Refresh tokens are stored in the OS keychain; everything else lives under `~/.neat/`.

## Packages

The packages sit flat at the repo root, one per responsibility, with no `internal/`:

- `cmd/`: the `main.go` entrypoint.
- `commands/`: the CLI (`neat version`, `neat setup`, `neat mcp`, `neat analytics on|off|status`).
- `analytics/`: anonymous usage statistics via Aptabase. `neat mcp` sends one `app_started` event per launch (version, `runtime.GOOS`/`runtime.GOARCH`, and a random ID that rotates each calendar month), and a `tool_called` event per tool call (tool name and `ok`/`error` only), recorded by SDK middleware, batched about once a minute, and sent under a separate session ID with no monthly ID. See `PRIVACY.md`.
- `mcp/`: the MCP server, every tool's definition and handler, the server's onboarding instructions (`instructions.go`), and the name enrichment that adds `TypeName`, `SystemName` and similar fields next to the bare IDs ESI returns.
- `esi/`: a thin facade over [`goesi-openapi`](https://github.com/fnt-eve/goesi-openapi), one file per ESI tag. Every request is checked against a generated table of each endpoint's required scope before it is sent.
- `sso/`: the login flow, the loopback callback session and its page (`sso/web/`, which also groups characters by account), scope groups, the ESI User-Agent, and the cached, auto-refreshing HTTP client per character.
- `sde/`: the local Static Data Export: syncing, and SQL lookups for universe data, industry data and routing.
- `nav/`: pure calculations for moving ships and characters, for any activity: what a character's skills let them use (cynos, bridges, conduits, hulls) from SDE requirements, clone jump timing, hull movement facts (warp, align, docking, highsec entry, cyno types, which bridges a hull can ride), security bands, displayed security and its in-game color, jump fuel, range and fatigue, and jump/gate route search with dotlan links.
- `cargo/`: pure calculations for what fits where: item volumes, everything at a station laid out with ship repackaging volumes and lost rigs, and packing cargo into loads.
- `market/`: pure buying calculations: walking order books, in-game Multibuy lists, scoring a trade hub on a basket of goods and its contracts, and ranking hubs for a shopping list.
- `dogma/`: a pure dogma evaluator: a ship's attributes after its fitted modules, rigs and pilot's skills, the way the game computes them (fitted holds, cyno fuel per light).
- `industry/`: pure industry calculations (facilities, jobs, research, invention, the blueprint tree, build-vs-buy, reprocessing, ore mixes, scheduling). It uses `nav/`, `cargo/` and `market/`; none of them use it.

The pure packages have no network, disk or clock access. Every number the SDE doesn't carry is a sourced constant in that package's `constants.go`.
- `config/`: neat's saved state under `~/.neat/`: YAML files the user can also edit (`characters.yaml`, `facilities.yaml`, `haul_profiles.yaml`, `routes.yaml`, `structures.yaml`, `hubs.yaml`), written by patching only the fields that changed so comments and the user's own keys survive, and `config.json` for internal settings and the analytics opt-out.
- `creds/`: the OS keychain wrapper (service name `neat`).
- `tools/`: code generators and CI checks (`genscopes`, `gentooldocs`, `genversion`, `versiongate`).

## Building

You need Go (see `go.mod` for the version), `jq`, and an ESI application of your own from [developers.eveonline.com](https://developers.eveonline.com/):

- Callback URL: `http://127.0.0.1:48211/callback`, exactly. FC matches it exactly, which is why the port is fixed.
- Scopes: enable every scope, so a login can request any scope group.

Put its client ID in `app_config.json` at the repo root (git-ignored):

```json
{ "esiClientId": "your-esi-client-id", "aptabaseAppKey": "A-EU-..." }
```

`aptabaseAppKey` is optional: without it the build sends no usage statistics. Keys are `A-EU-*` or `A-US-*`, and the region picks the Aptabase host. `make build` and `make install` mark their events as debug (`-X main.analyticsDebug=true`), as does any build whose version is `dev`, so a developer's own use stays out of Aptabase's release numbers. Only GoReleaser builds send release data.

`make build` builds `bin/neat` with the version, client ID and Aptabase key injected through `-ldflags`. `make install` also installs it to `~/.local/bin` (override with `PREFIX=...`). It deletes the old binary before copying, because on macOS copying a new build over an existing binary in place makes the kernel kill it on launch (exit 137).

To try a local build in Claude Code with the plugin installed, `make install` and restart Claude Code; the plugin runs whichever `neat` is first on your PATH.

## Testing

```bash
go test ./...
```

CI runs `go build`, `go vet` and `go test -race` on Linux, macOS and Windows for every push and pull request. Tests never touch the real keychain (`keyring.MockInit()`), and tests that need SDE data build small fixture databases with `sde/sdetest`.

## Generated files

Run `go generate ./...` and commit the result after:

- **changing `VERSION`**, which updates the plugin manifest's version;
- **adding, renaming or redescribing an MCP tool**, which regenerates [`docs/reference/tools.md`](reference/tools.md). A test fails while that file is stale, and generation fails if a new tool matches no area in `tools/gentooldocs`;
- **bumping `goesi-openapi`**, which regenerates `esi/scopes_gen.go`. Tests fail if an endpoint's scope belongs to no scope group in `sso/scopes.go`.

## Releasing

`VERSION` is the single source of truth for the binary version, the plugin version and the git tag. Any pull request that changes Go code, `go.mod`, `.goreleaser.yaml` or `claude_resources/` must bump it; the `versiongate` check enforces this. Docs, tests and CI config are exempt. Add a user-facing entry to `claude_resources/neat/CHANGELOG.md` alongside the bump.

Merging to `main` tags `v<VERSION>` if the tag doesn't exist yet and runs GoReleaser, which publishes binaries for macOS, Linux and Windows on amd64 and arm64. It reads the repository variables `ESI_CLIENT_ID` and `APTABASE_APP_KEY`; if `APTABASE_APP_KEY` is unset, the release sends no usage statistics.

## Claude resources

[`claude_resources/`](../claude_resources/README.md) holds the Claude Code plugin: it registers the MCP server and ships the skills that guide Claude through logging in characters, setting up facility profiles, building industry plans, and planning any move of cargo, ships or characters (haul profiles and routes included). The MCP server's own instructions (`mcp/instructions.go`) carry the essentials for clients that don't load skills, such as the Claude desktop app.

## Reference and design docs

- [`reference/tools.md`](reference/tools.md): every MCP tool, grouped by area (generated).
- [`reference/esi-coverage-roadmap.md`](reference/esi-coverage-roadmap.md): ESI coverage by tag.
- [`superpowers/specs/`](superpowers/specs/): design specs, one per feature, dated. They record what was decided at the time and aren't rewritten later; a status line notes when the code has moved on.
- [`superpowers/plans/`](superpowers/plans/): the implementation plans that went with them.
