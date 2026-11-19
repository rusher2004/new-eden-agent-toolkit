# New Eden Agent Toolkit — Foundation Design

**Status:** Approved, mostly implemented. As of 2026-11-19 the Claude Code plugin ships and the OAuth2 redirect uses a loopback HTTP callback instead of the custom URL scheme below (`2026-11-19-loopback-oauth-callback-design.md`); the Claude Desktop `.mcpb` extension has not been built, so Desktop users add neat to `claude_desktop_config.json` by hand (see `README.md`).

**Superseded in part (2026-11-19):** the OAuth2 redirect no longer uses the `eveauth-neat://` custom scheme, its per-OS registration (`urlscheme/`), the `callback-handler` subcommand, the `~/.neat/bin/neat` launcher or the Unix-socket handoff described below. It is a loopback HTTP redirect to `http://127.0.0.1:48211/callback` served by `sso/` itself, reversing this spec's "custom scheme only, no loopback-HTTP fallback" decision so the browser tab can land on a neat page. See `2026-11-19-loopback-oauth-callback-design.md`.
**Date:** 2026-07-23

## Purpose

Lay the technical foundation for `neat` (New Eden Agent Toolkit): a Go application that lets a
Claude user interact with EVE Online via CCP's ESI API and the EVE Ref Static Data Export (SDE).
It ships three ways: a Claude Code plugin, a Claude Desktop extension (`.mcpb`), and (implicitly,
bundled with the plugin) Claude skills. This phase builds the scaffolding — packages, auth,
config, storage, distribution — not domain-specific ESI tools/skills, which are deferred to
follow-up work once specific use cases are chosen.

Non-negotiable constraints from the requester:
- The end user is not assumed to be technically knowledgeable. No terminal usage should be
  required for the golden path.
- Support adding zero or more authenticated EVE characters.
- Wherever a resource exists in both the SDE and the ESI API, the SDE must be used (performance +
  good API citizenship).
- Credentials go in the OS keychain; other config goes under `~/.neat/`. No environment variables.
- Flat package structure, no `internal/`, modeled on the sibling projects `ci-cd-assistant` and
  `claude-project-tracker` — but diverge from their specifics where a cleaner option exists.

## Repo & module

- Module: `github.com/rusher2004/new-eden-agent-toolkit`
- Go 1.26.1 (matches siblings)
- Binary name: `neat`
- `CGO_ENABLED=0` throughout — every dependency chosen below is pure Go, so static
  cross-compilation (linux/darwin/windows × amd64/arm64) works the same way it does for
  `ci-cd-assistant`.

## Package layout (flat, no `internal/`)

| Package | Responsibility |
|---|---|
| `cmd/` | `main.go` entrypoint only, dispatches into `commands/` |
| `commands/` | urfave/cli/v3 command tree: `mcp`, `setup`, `version`, and a hidden `callback-handler <url>` (invoked only by the OS on URL-scheme activation, never by a user directly) — named `commands`, not `cli`, to avoid shadowing the urfave/cli import, same rationale as `ci-cd-assistant` |
| `config/` | Non-secret state in `~/.neat/config.json` (known characters' display names/IDs, SDE version) |
| `creds/` | Keychain wrapper (`zalando/go-keyring`), service name `"neat"`. One entry per character: key `character_refresh_token:<characterID>` — the multi-character requirement (zero or more) means there's no fixed key set the way the siblings have, so keys are generated per character rather than named constants |
| `sso/` | ESI OAuth2/PKCE login flow orchestration (browser + local IPC callback listener) and full token lifecycle ownership (exchange, refresh, persistence via `creds/`), built on `goesi-openapi`'s OAuth2 primitives |
| `urlscheme/` | Per-OS registration of the `eveauth-neat://` custom URI scheme as the ESI OAuth redirect handler (build-tag-separated `_darwin.go`/`_windows.go`/`_linux.go` implementations) |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi` — adds our user-agent string; calls `sso/` for a valid access token per request rather than owning any refresh logic itself |
| `sde/` | EVE Ref SDE downloader + local `bbolt` cache |
| `mcp/` | MCP server (official `modelcontextprotocol/go-sdk`) + tool registrations |

Rationale for the facade packages (`esi/`, `sso/`): `mcp/` tool handlers never import
`goesi-openapi` directly. `goesi-openapi` is young (pre-1.0, small community) — wrapping it keeps
a future swap contained to two packages instead of a rewrite.

## Third-party dependencies

| Concern | Library | Why |
|---|---|---|
| MCP server | `github.com/modelcontextprotocol/go-sdk` | Official, Google-collab, spec-current. Verified directly via `gh api repos/modelcontextprotocol/go-sdk` on 2026-07-23: not archived, pushed 2026-07-17, v1.6.1 latest stable release (v1.7.0-pre.x exists as prerelease only). Preferred over the community `mark3labs/mcp-go` used by `claude-project-tracker` — that project predates the official SDK's maturity. |
| ESI client | `github.com/fnt-eve/goesi-openapi` | Generated from CCP's official OpenAPI 3.x spec (covers all endpoints without hand-wrapping); ships OAuth2/PKCE + JWT claim verification (JWKS-based, via `MicahParks/keyfunc`) that overlaps almost exactly with what `sso/` needs. Verified via `gh api repos/fnt-eve/goesi-openapi` on 2026-07-23: MIT licensed, not archived, latest release v0.0.37 pushed 2026-07-22. Accepted risk: pre-1.0, 9 stars — mitigated by facading it (see above). |
| CLI framework | `github.com/urfave/cli/v3` | Matches `ci-cd-assistant`'s structured command tree + exit-code discipline; more scalable than `claude-project-tracker`'s hand-rolled switch for a project that also needs a real `setup` flow. |
| Keychain | `github.com/zalando/go-keyring` | Same as both siblings. |
| SDE cache | `go.etcd.io/bbolt` | Pure-Go embedded KV store, per your evaluation request — fits the "stays up to date with the app, no re-parsing JSONL on every startup" goal. Bucket-per-resource, key = ID, value = JSON-encoded record. |

## Auth: adding a character

New territory — neither sibling project has a real OAuth2 flow to model.

- **App registration**: reconfigure the EVE dev-portal app as a **PKCE-only public app** — no
  client secret. A secret shipped inside a binary distributed to many users isn't actually secret,
  so it's dropped from the design entirely. Revoke/regenerate the current confidential-app secret.
- **Seeding**: the only surviving app value, `clientId`, is baked into the binary at build time via
  `-ldflags` (same mechanism `ci-cd-assistant` uses for `version`). `app_config.json` never gets
  committed (add to `.gitignore`); CI supplies the real value at release time. Nothing for the end
  user to configure. **Naming note**: `app_config.json` (repo root, git-ignored, build-time-only —
  never read by the shipped binary at runtime) and `~/.neat/config.json` (runtime, per-user, owned
  by `config/`) are unrelated files that happen to share a substring — not two names for the same
  thing.
- **Redirect URI**: a custom scheme, `eveauth-neat://callback` — ESI's app settings explicitly
  allow schemes starting with `eveauth` (lower-case letters/digits/`+`/`.`/`-`, ending in a letter
  or digit). **Security note, corrected from an earlier draft**: RFC 8252 §8.1 does not endorse
  custom schemes over loopback — it flags an interception risk for *each*: a custom scheme can be
  registered by more than one app (indeterminate which app receives the code), while a loopback
  port can be bound by another local process first. PKCE (§6, already required here via
  `goesi-openapi`) is what actually neutralizes code interception under either mechanism, not the
  choice of redirect URI. The real reason for choosing the custom scheme here is the practical one
  you raised earlier: the callback port/URL isn't fixed long-term, and a registered scheme doesn't
  depend on a specific port being free. **Decision: custom scheme only, no loopback-HTTP
  fallback** — if OS registration didn't succeed, `add_character` fails with a clear, specific
  error rather than silently degrading to a different mechanism.
- **OS registration** (`urlscheme/`): registers `eveauth-neat://` as a handler that invokes the
  stable launcher path (`~/.neat/bin/neat`, see Distribution) with the activation URL, so
  registration survives binary updates without needing to re-run. Runs idempotently on every
  `neat mcp` startup (whichever channel launched it — plugin bootstrap or `.mcpb`-embedded binary),
  not only via `neat setup`, so registration happens automatically the first time Claude
  Desktop/Code starts the server — no terminal step required on the golden path. `neat setup`
  remains available as an explicit manual entrypoint (troubleshooting, re-registration, scripting),
  it's just not required to get a working install:
  - **macOS**: no bare CLI binary can register a URL scheme directly — Launch Services requires an
    actual `.app` bundle. `neat setup` builds a minimal wrapper bundle
    (`~/.neat/NeatURLHandler.app`, `Info.plist` declaring `CFBundleURLTypes`/
    `CFBundleURLSchemes: ["eveauth-neat"]`, `CFBundleExecutable` pointing to a tiny shim that
    execs `~/.neat/bin/neat callback-handler <url>`), then registers it via `lsregister -f`.
  - **Windows**: writes `HKEY_CURRENT_USER\Software\Classes\eveauth-neat` (`URL Protocol` +
    `shell\open\command` → `"<launcher path>" callback-handler "%1"`) — no admin elevation needed
    since it's HKCU, not HKLM.
  - **Linux**: writes a `.desktop` file to `~/.local/share/applications/` with
    `Exec=neat callback-handler %u` and `MimeType=x-scheme-handler/eveauth-neat;`, then runs
    `xdg-mime default neat-url-handler.desktop x-scheme-handler/eveauth-neat`. Requires
    `xdg-utils`; a missing/minimal desktop environment is treated as a pre-existing constraint of
    "open a browser" in general, not something specific to this design.
- **Cross-process callback handoff**: the OS launches a *new* `neat callback-handler <url>`
  process to deliver the redirect — it isn't the same process that's blocked waiting inside the
  `add_character` tool call. Handoff between them is a local Unix domain socket at
  `~/.neat/run/callback.sock`, created with `0600` permissions (owner-only). Go's `net` package
  supports the `"unix"` network on Windows via `afunix.sys`, but that requires **Windows 10
  version 1803+ / Windows Server 2019+** — needs verification during implementation (e.g. exact
  Go-version behavior, any unsupported socket options), not settled fact; no platform-specific IPC
  code is needed on macOS/Linux. If `add_character` is called while a listener already holds that
  socket path, the new call fails immediately with a clear "a character-add flow is already in
  progress" error rather than colliding on the fixed path.
- **Flow**: an `add_character` MCP tool is the *only* way to add a character — no CLI equivalent,
  per your explicit call that this must work entirely from a Claude conversation. The tool handler
  (`sso/`):
  1. Generates PKCE verifier/challenge + `state` (via `goesi-openapi`'s OAuth2 config helpers).
  2. Opens a Unix-socket listener at `~/.neat/run/callback.sock`.
  3. Opens the user's default browser to the ESI authorize URL (`redirect_uri=eveauth-neat://callback`).
  4. Blocks reading from the socket for **up to 5 minutes** for the `callback-handler` process to
     deliver the redirect URL, then validates `state` and tears down the socket. (Decision, not a
     placeholder: 5 minutes is enough time for a real login without holding the MCP call open
     indefinitely.)
  5. Exchanges the code for tokens (`goesi-openapi`'s `Exchange`), which also returns verified JWT
     claims containing the character's ID and name directly — no extra ESI call needed just to
     know who logged in.
  6. Persists the refresh token via `creds/`.
- **Token lifecycle ownership**: `sso/` owns the full lifecycle — initial exchange, refresh, and
  persistence via `creds/`. It exposes something like `sso.ValidAccessToken(ctx, characterID)
  (string, error)` that checks expiry and refreshes-then-persists transparently if needed. `esi/`
  never refreshes anything itself; it calls `sso.ValidAccessToken` before each authenticated
  request and only holds the resulting access token in memory for that call. (Corrected from an
  earlier draft, which inconsistently attributed refresh to `esi/` in one place and to no package
  in another.)
- `list_characters` / `remove_character` are also plain MCP tools (no browser involved), keeping
  all character management inside the conversation.

## SDE storage & the SDE-vs-ESI rule

- Downloader checks EVE Ref's `index.json` at `data.everef.net/ccp/sde/` using `ETag`/
  `Last-Modified` (per their documented client-etiquette guidance) against the version recorded in
  `config/`, and only pulls the dated zip archive when it's changed.
  Extracted JSON Lines get loaded into a local `bbolt` database at `~/.neat/sde/sde.db`, one bucket
  per resource type, keyed by ID.
- This phase wires the pipeline end-to-end for a couple of representative resources (types,
  groups) as proof; loading the remaining ~30 SDE resource categories is deferred until a specific
  skill needs them.
- **Hard rule, documented in `CLAUDE.md`**: any tool needing a resource that exists in the SDE
  *must* read it from `sde/`, never call ESI for it. Enforced by package boundaries and code
  review for now, not a runtime routing abstraction — revisit only if that stops scaling once real
  tools exist.

## Distribution

Diverging from `ci-cd-assistant`'s `claude_resources/` wrapper — current Claude Code docs make it
unnecessary. Per `code.claude.com/docs/en/plugins-reference` and `.../plugin-marketplaces`
(researched 2026-07-23): `.claude-plugin/plugin.json` can live at the repo root for a single-plugin
repo, no `marketplace.json` required, and `plugin.json` can declare `mcpServers` directly (so
installing the plugin auto-wires the server) alongside `skills/`. The same research confirmed two
gaps that shape the sections below: `mcpServers.command` has no documented per-OS/arch variant
(unlike `.mcpb`'s `platform_overrides`), and there's no install/update lifecycle hook — neither is
officially addressed, both are inferred gaps rather than documented non-features.

**Claude Code plugin** (repo root):
```
.claude-plugin/
  plugin.json          # declares mcpServers + skills/
skills/
  (placeholder for now — specific skills are a follow-up)
bin/
  neat-launcher.js     # bootstrap, see below
```

**The binary-distribution problem**: `plugin.json`'s `mcpServers.command` is a single fixed
string with no per-OS/arch variant (confirmed gap vs. `.mcpb`), and there's no install/update
lifecycle hook to fetch a platform binary once at install time. Rather than committing compiled
binaries into the git tree (bloats history every release) or maintaining OS-specific script
variants (no way to select between them from one fixed `command`), `mcpServers.command` is
`"node"` running `bin/neat-launcher.js` — plain text, no npm dependencies, works identically on
every OS since it only runs inside Claude Code's own (Node-based) process. On each launch it:
1. Resolves `~/.neat/bin/` and checks for a cached `neat` binary at the latest known version.
2. If missing/stale, downloads the matching release tarball from GitHub Releases for
   `process.platform`/`process.arch`, extracts, and marks it executable.
3. Execs it with `mcp` (stdio passthrough).

Net effect: **zero compiled binaries ever committed to git**, and the plugin's MCP server
self-updates the next time it launches after a new release — no manual reinstall for this channel.
Marketplace-level updates (skills content, `plugin.json` itself) are pulled via
`/plugin marketplace update` (manual) or auto-update (opt-in for third-party marketplaces, off by
default — the README/setup guidance should tell users to enable it via `/plugin` → Marketplaces).

**Claude Desktop extension** (`.mcpb`): separate `extension/manifest.json`, built via
`npx @anthropic-ai/mcpb pack`, embedding the real per-platform binary directly (`.mcpb` *does*
support `platform_overrides`, unlike `plugin.json`) — no bootstrap script needed on this channel.
Accepted limitation: `.mcpb` updates are manual — per Claude's Desktop Extensions support docs
(researched 2026-07-23), auto-update applies only to official Anthropic-built extensions; there's
no auto-update path for third-party/custom `.mcpb`s. Not worth building a custom updater for in
this phase.

**Non-technical install story**: Desktop users double-click the `.mcpb`; Code-plugin users type
one `/plugin marketplace add rusher2004/new-eden-agent-toolkit` line in a conversation. Neither
requires a terminal.

**Keeping the stable launcher path populated regardless of channel**: `urlscheme/` registration
(above) points the OS handler at `~/.neat/bin/neat`, but only the plugin's `neat-launcher.js`
writes that path today — a Desktop-only install (`.mcpb`, which embeds its binary inside the
extension's own install location) would never populate it, leaving the registered callback handler
pointing at a file that doesn't exist. Fix: `commands/mcp` startup (shared by both channels) always
ensures `~/.neat/bin/neat` exists and matches the currently-running binary — copying `os.Executable()`
into place if missing or stale — before performing/verifying `urlscheme` registration. So whichever
channel the user installed, the first time its `neat mcp` runs, the launcher path and the OS
registration both become correct.

## Build & release

- `.goreleaser.yaml` modeled on `ci-cd-assistant`: linux/darwin/windows × amd64/arm64,
  `CGO_ENABLED=0`, `-ldflags` injecting `version` and the ESI `clientId`, checksummed archives,
  conventional-commit changelog grouping.
- `VERSION` file drives releases the same way it does for `ci-cd-assistant`.
- Release CI is also responsible for building the `.mcpb` (via `mcpb pack`) as a separate
  artifact/asset — the plugin channel needs no binary artifact at all (the launcher fetches
  directly from GitHub Releases).

## Conventions (`CLAUDE.md`)

Adopt both siblings' converged conventions directly:
- `fmt.Errorf("operation: %w", err)` wrapping everywhere; never discard an error.
- External test packages by default (`package foo_test`); `*_internal_test.go` only for
  unexported-internals tests; table-driven, stdlib-only (no testify) for our own code.
- Docs updated in the same commit/PR as the behavior they describe.
- The SDE-vs-ESI rule (above) stated explicitly as a hard rule, not a suggestion.

## Explicitly out of scope for this phase

- Any ESI domain-specific tool beyond `add_character` / `list_characters` / `remove_character`
  (e.g. assets, market, killmails, industry) — deferred until specific skills/use-cases are chosen.
- Loading the full SDE resource catalog into `bbolt` — only a couple of representative resources
  are wired up now.
- A loopback-HTTP fallback for the OAuth callback if `eveauth-neat://` registration fails —
  explicitly declined; failures surface as clear errors instead.
- A runtime-enforced SDE-vs-ESI routing abstraction — the rule is enforced by package boundaries
  and code review for now (see SDE storage section); revisit only if that stops scaling.
- Any Claude Desktop `.mcpb` auto-update mechanism.
- Self-hosting the SDE (currently sourced from EVE Ref; revisit later per the original request).
