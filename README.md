# neat — New Eden Agent Toolkit

A Go MCP server, Claude Code plugin, and Claude Desktop extension for interacting with EVE Online
via CCP's ESI API and the EVE Ref Static Data Export (SDE).

Design spec: `docs/superpowers/specs/2026-07-23-foundation-design.md`.
Implementation plans: `docs/superpowers/plans/`.

## Status

Foundation phase (plan 1 of 4): CLI skeleton, `config/`, and `creds/` packages. No ESI/SSO
integration, SDE cache, or plugin/extension packaging yet — see the plans directory for what's
next.

## Architecture

Flat package structure at repo root. No `internal/`. One package per responsibility:

| Package | Responsibility |
|---|---|
| `cmd/` | `main.go` entrypoint only |
| `commands/` | `urfave/cli/v3` command tree (`version`, `setup`, and more to come) |
| `config/` | Non-secret state in `~/.neat/config.json` |
| `creds/` | OS keychain wrapper (`zalando/go-keyring`), service name `"neat"` |

More packages (`sso/`, `urlscheme/`, `esi/`, `sde/`, `mcp/`) land in follow-up plans.

See `CLAUDE.md` for conventions and the doc-sync rule that keeps this file, that one, and the
design spec from drifting apart.

## Build

```bash
go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev)" -o neat ./cmd
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
