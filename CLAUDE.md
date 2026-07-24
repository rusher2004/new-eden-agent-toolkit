# neat — New Eden Agent Toolkit

Go MCP server + Claude Code plugin + Claude Desktop extension for interacting with EVE Online via
CCP's ESI API and the EVE Ref Static Data Export (SDE). Design spec:
`docs/superpowers/specs/2026-07-23-foundation-design.md`.

## Architecture

Flat package structure at repo root. No `internal/`. One package per responsibility:

| Package | Responsibility |
|---|---|
| `cmd/` | `main.go` entrypoint only |
| `commands/` | `urfave/cli/v3` command tree (`version`, `setup`, `callback-handler` (hidden — invoked only by the OS on URL-scheme activation), and more to come) |
| `config/` | Non-secret state in `~/.neat/config.json` |
| `creds/` | OS keychain wrapper (`zalando/go-keyring`), service name `"neat"` |
| `urlscheme/` | Per-OS registration of `eveauth-neat://` as the ESI OAuth redirect handler. macOS embeds a real compiled Swift/Cocoa wrapper (`urlscheme/macos/`, built via `make urlscheme-macos-bundle`) via `go:embed` — a bare executable can't receive macOS's `kAEGetURL` Apple Event. Windows (registry) and Linux (`.desktop` + `xdg-mime`) exist but have no CI execution yet — see "Known gaps" below |
| `sso/` | ESI OAuth2/PKCE login flow: PKCE + state generation, a two-socket Unix IPC handoff for the callback (a lock socket that's bound but never `Accept()`-ed, plus a separate delivery socket), orchestration via `github.com/fnt-eve/goesi-openapi`, refresh-token persistence via `creds/` |

More packages (`esi/`, `sde/`, `mcp/`) land in follow-up plans — see `docs/superpowers/plans/`.

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

## Hard rules

- **SDE over ESI**: any tool needing a resource that exists in the SDE must read it from `sde/`,
  never call ESI for it. Performance and API-citizenship requirement, not a suggestion.
- **No environment variables for config/credentials.** Config lives under `~/.neat/`; credentials
  live in the OS keychain only.
- **No client secret.** The ESI app is a PKCE-only public app — nothing secret ships in the binary.

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

```bash
go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev)" -o neat ./cmd
```

## Test

```bash
go test ./...
```
