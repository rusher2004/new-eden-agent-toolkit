# neat — New Eden Agent Toolkit

A Go MCP server, Claude Code plugin, and Claude Desktop extension for interacting with EVE Online
via CCP's ESI API and the EVE Ref Static Data Export (SDE).

Design spec: `docs/superpowers/specs/2026-07-23-foundation-design.md`.
Implementation plans: `docs/superpowers/plans/`.

## Status

Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), the MCP server with its character-management tools (`mcp/`, `neat mcp`),
and a first real ESI-backed domain tool (`esi/`, `get_wallet_balance`) are built. The SDE cache
and plugin/extension packaging don't exist yet. See the plans directory for what's next.

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
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its four tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance` |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi` — builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client. Holds no state of its own |

More packages (`sde/`) land in follow-up plans.

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
