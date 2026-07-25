# Plan 2b: MCP Server + Character Tools — Design

## Goal

Build the `neat mcp` command and the three character-management MCP tools
(`add_character`, `list_characters`, `remove_character`) that Plan 2a's
auth-core work (OAuth2/PKCE flow, custom URL scheme, callback handoff) was
built to support. Also close a compliance gap discovered during this design
pass: `sso/`'s outbound requests to CCP's login/JWKS endpoints currently
carry no identifying User-Agent, which CCP's ESI best-practices guidance
flags as a bannable offense.

This follows directly from
`docs/superpowers/specs/2026-07-23-foundation-design.md`, which already
specifies the package layout, tool flow, and startup-wiring intent this
plan implements. Nothing here contradicts that spec; it fills in the
concrete types, signatures, and a few decisions the foundation spec left
open.

## Scope

New code:
- `mcp/` package (new): the MCP server (official
  `github.com/modelcontextprotocol/go-sdk`, aliased `mcpsdk` inside this
  package's files to avoid shadowing the package's own name — same
  rationale as `commands` aliasing `cli`) and the three tool registrations.
- `commands/mcp.go` (new): thin CLI wiring for `neat mcp` — startup
  sequence, then hands off to `mcp/`.
- `sso/useragent.go` (new): User-Agent construction and the CCP HTTP calls
  that need it.

No changes needed to `config/` or `creds/` — both already expose exactly
what the tool handlers need. No `esi/` package yet — `list_characters` and
`remove_character` are pure local-state operations, and `add_character`
already gets the character's ID and name for free from the JWT claims
`sso.AddCharacter` returns. `esi/` stays deferred until a real domain tool
needs a live ESI data-endpoint call.

## `clientID` and `version` threading

`cmd/main.go` gains `var clientID = "dev"`, set via `-ldflags` the same way
`version` already is. `commands.Root(version, clientID string)` passes
`clientID` only into `mcpCommand(version, clientID)` — the other commands
don't need it, matching the existing pattern where `version` only reaches
`versionCommand`.

## ESI User-Agent compliance

Per
[CCP's ESI best-practices guidance](https://developers.eveonline.com/docs/services/esi/best-practices/#user-agents):
all ESI requests should carry a User-Agent identifying the app; email and
app name/version are the most strongly preferred identifying fields.
Non-compliant apps risk being temporarily banned from ESI. `sso/`'s two
outbound HTTP calls to CCP (JWKS fetch, OAuth2 token exchange) currently
use Go's default `User-Agent`, added in Plan 2a without full CCP
best-practice compliance.

**`sso/useragent.go` (new):**

```go
package sso

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	contactEmail = "robertinthecloud@icloud.com"
	repoURL      = "https://github.com/rusher2004/new-eden-agent-toolkit"

	// jwksURL is duplicated from goesi-openapi's unexported esiJWKSURL
	// constant. goesi.ESIDefaultKeyfunc gives no way to inject a custom
	// HTTP client, so this package builds the keyfunc itself instead of
	// using that helper. If goesi-openapi ever adds User-Agent support
	// directly, this duplication and newKeyfunc can go away in favor of
	// calling ESIDefaultKeyfunc again.
	jwksURL = "https://login.eveonline.com/oauth/jwks"
)

// UserAgent builds neat's User-Agent string for outbound CCP requests, per
// https://developers.eveonline.com/docs/services/esi/best-practices/#user-agents.
func UserAgent(version string) string {
	return fmt.Sprintf("neat/%s (%s; +%s)", version, contactEmail, repoURL)
}

// userAgentTransport sets User-Agent on every outgoing request.
type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", t.userAgent)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func userAgentClient(userAgent string) *http.Client {
	return &http.Client{Transport: &userAgentTransport{userAgent: userAgent}}
}

// newKeyfunc builds the ESI JWT-verification keyfunc using a client that
// sends userAgent, replacing goesi.ESIDefaultKeyfunc (see jwksURL above).
func newKeyfunc(ctx context.Context, userAgent string) (jwt.Keyfunc, error) {
	storage, err := jwkset.NewStorageFromHTTP(jwksURL, jwkset.HTTPClientStorageOptions{
		Client: userAgentClient(userAgent),
		Ctx:    ctx,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch ESI JWT signing keys: %w", err)
	}
	kf, err := keyfunc.New(keyfunc.Options{Storage: storage})
	if err != nil {
		return nil, fmt.Errorf("build JWKS keyfunc: %w", err)
	}
	return kf.Keyfunc, nil
}
```

**Changes to `sso/oauth.go`:**
- `AddCharacter(ctx context.Context, clientID, userAgent string, scopes []string) (Character, error)` —
  one new parameter.
- `newOAuthConfig(ctx, clientID, userAgent string)` calls `newKeyfunc`
  instead of `goesi.ESIDefaultKeyfunc(ctx)`.
- `AddCharacter` wraps `ctx = context.WithValue(ctx, oauth2.HTTPClient,
  userAgentClient(userAgent))` before calling `addCharacter(ctx, cfg,
  scopes)`, so `cfg.Exchange`'s request — made via `golang.org/x/oauth2`,
  which honors that context key — also carries the header.
- `addCharacter`'s internal logic (PKCE, socket handoff, state validation)
  is unchanged. The existing `oauthConfig` interface test seam still
  applies, so Plan 2a's existing unit tests for that logic need no changes.

**Known gap, flagged not fixed:** `goesi.Config.RefreshToken` is exported
but unused today (no `ValidAccessToken`/token-refresh code exists yet,
since nothing calls live ESI data endpoints). Whoever builds that later
must also inject `oauth2.HTTPClient` into that call's context, or refresh
requests will silently revert to the default User-Agent. Noting it here so
it isn't lost.

**New tests:** `UserAgent()` formatting, and `userAgentTransport.RoundTrip`
(pure — verifies the header is set and the request is otherwise
unmodified). Both run with no network access.

## MCP tool schemas (`mcp/` package)

One file per tool, plus a server file:

**`mcp/server.go`:**
```go
func NewServer(version, clientID string, store *config.Store) (*mcpsdk.Server, error)
```
Constructs the `mcpsdk.Server` (name `"neat"`, version `version`) and
registers all three tools against it.

**`mcp/add_character.go`:**
- Input: empty struct — `clientID`, User-Agent, and scopes are all
  app-level, never model-supplied.
- Output: `struct { CharacterID int32 \`json:"characterId"\`; CharacterName
  string \`json:"characterName"\` }`
- `allScopes`: a committed `[]string` literal — the ~70 scopes currently
  listed in (git-ignored) `app_config.json`. Scopes are public ESI
  permission identifiers, not secrets, so unlike `clientID` they're
  committed directly rather than injected at build time. Per your call:
  request everything now, refine to a minimal per-tool set once real
  domain tools exist and re-auth can be scoped down.
- Handler: calls `sso.AddCharacter(ctx, clientID, sso.UserAgent(version),
  allScopes)` through an overridable `var addCharacter =
  sso.AddCharacter` func-var (same test-substitution pattern already used
  for `openBrowser`/`commandRunner` elsewhere in this codebase) → loads
  config → upserts the returned `Character{ID,Name}` into `Characters`
  (replaces an existing entry with the same ID, else appends) → saves →
  returns output.

**`mcp/list_characters.go`:**
- Input: empty struct.
- Output: `struct { Characters []CharacterInfo \`json:"characters"\` }`,
  `CharacterInfo{CharacterID int32 \`json:"characterId"\`, CharacterName
  string \`json:"characterName"\`}` — deliberately distinct from
  `config.Character` so the MCP-facing contract doesn't leak the internal
  storage schema.
- Handler: loads config, maps `Characters` to `[]CharacterInfo`. Returns an
  empty slice (never `null`) when there are no characters.

**`mcp/remove_character.go`:**
- Input: `struct { CharacterID int32 \`json:"characterId"\` }` (required).
- Output: `CharacterInfo` — the removed character's cached id/name.
- `var ErrCharacterNotFound = errors.New("character not found")` — mirrors
  `creds.ErrNotFound`'s existing pattern.
- Handler: loads config, finds the character by ID (returns
  `ErrCharacterNotFound`, wrapped with the ID, if absent) → deletes the
  keychain entry via `creds.DeleteCharacterRefreshToken` (idempotent
  either way) → removes it from `Characters` and saves → returns the
  removed `CharacterInfo`.
  Order matters: creds deletion happens before the config save, so a
  partial failure leans toward "token gone, config stale" rather than
  "token still present, config says removed" — the safer failure mode.

## Startup wiring (`commands/mcp.go`)

```go
func mcpCommand(version, clientID string) *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "run the neat MCP server (stdio transport)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store, err := config.NewStore()
			if err != nil {
				return fmt.Errorf("resolve config store: %w", err)
			}

			launcherPath, err := ensureLauncher()
			if err != nil {
				return fmt.Errorf("ensure launcher binary: %w", err)
			}

			if err := urlscheme.Register(launcherPath); err != nil {
				return fmt.Errorf("register URL scheme: %w", err)
			}

			server, err := mcp.NewServer(version, clientID, store)
			if err != nil {
				return fmt.Errorf("build MCP server: %w", err)
			}

			return server.Run(ctx, &mcpsdk.StdioTransport{})
		},
	}
}
```

`ensureLauncher()` (unexported helper in `commands/mcp.go`):
1. Resolves `~/.neat/bin/neat` (creating `~/.neat/bin` with `0700` if
   needed).
2. Resolves the currently running binary via `os.Executable()`, symlink-
   resolved via `filepath.EvalSymlinks`, and compares it against the
   symlink-resolved target path.
3. If they already match (e.g. dev workflow running the installed binary
   directly), returns the target path unchanged — no copy.
4. Otherwise copies the running binary's bytes into a temp file in the
   target directory, marks it executable, and atomically renames it over
   the target path. Always copies when the paths differ — no
   mtime/hash-based shortcut. This runs once per `neat mcp` startup, not a
   hot path, so correctness (always current) wins over the marginal cost
   of a redundant copy.
5. Returns the resolved target path, which is what gets passed to
   `urlscheme.Register`.

`urlscheme.Register` is already idempotent by design (documented on the
function itself), so calling it unconditionally on every `neat mcp`
startup — per the foundation spec's stated intent — is safe.

## Error handling

- `fmt.Errorf("...: %w", err)` wrapping throughout, matching the rest of
  the codebase.
- `mcp/`'s `ErrCharacterNotFound` sentinel follows `creds.ErrNotFound`'s
  existing convention.
- Tool handlers return `(Output, error)` directly; a non-nil error becomes
  a client-visible tool error per the official SDK's `ToolHandlerFor`
  convention. Exact mechanics (e.g. whether the SDK wraps it as an
  `isError` result automatically or requires an explicit `CallToolResult`)
  get confirmed via `go doc` once the dependency is added, during
  plan-writing/implementation — not assumed here.
- `sso.ErrAddCharacterInProgress` (existing, from Plan 2a) propagates
  through `add_character`'s handler wrapped with context
  (`fmt.Errorf("add character: %w", err)`), so `errors.Is` still works for
  callers that care.

## Testing

- `mcp/` package tests reuse established patterns: `keyring.MockInit()`
  (already used in `sso`/`commands` tests) for creds isolation,
  `config.NewStoreAt(tempFile)` for config isolation, and func-var
  substitution for `sso.AddCharacter` (via the `addCharacter` package
  var in `mcp/add_character.go`) — no real network, browser, or OS
  keychain access in unit tests.
- One integration-style test uses the SDK's `NewInMemoryTransports` to
  spin up a real in-process client/server pair and confirm all three tools
  are registered with the expected names and schemas — catches wiring
  mistakes (typo'd tool names, schema-generation panics on the empty-input
  structs) that isolated handler-function tests would miss.
- `sso/useragent_test.go` (new): `UserAgent()` formatting and
  `userAgentTransport.RoundTrip` header injection, both network-free.

## Local dev build convenience

New `Makefile` target, for local testing only — not release automation,
which stays a separate future concern (the foundation spec's `Build &
release` section already anticipates a `.goreleaser.yaml` doing the real
`-ldflags` injection at release time):

```makefile
build:
	go build -ldflags "-X main.version=dev -X main.clientID=$$(jq -r .clientId app_config.json)" -o bin/neat ./cmd
```

## Explicitly out of scope for this plan

- `esi/` package and any live ESI data-endpoint calls — deferred until a
  real domain tool needs one.
- Token refresh (`ValidAccessToken` or equivalent) — nothing calls ESI
  data endpoints yet, so nothing needs a refreshed access token. The
  User-Agent gap this would reopen is flagged above.
- Narrowing `add_character`'s scope list down from "everything" — explicit
  choice for development flexibility; revisit once real domain tools
  exist and their actual scope needs are known.
- Full release-time build/CI wiring for `clientID`/`version` injection —
  the `Makefile` target above is a local-dev convenience only.
- Claude Code plugin / Claude Desktop `.mcpb` packaging — a distribution
  phase, not this plan's scope.
