# Plan 2b: MCP Server + Character Tools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `neat mcp` command and its three character-management MCP tools (`add_character`, `list_characters`, `remove_character`), and close a CCP ESI User-Agent compliance gap in `sso/` discovered while designing this plan.

**Architecture:** A new `mcp/` package builds the MCP server (official `modelcontextprotocol/go-sdk`) and registers the three tools against it; a new `commands/mcp.go` provides the thin CLI entrypoint and startup wiring (launcher-binary population + URL-scheme registration). `sso/` gains a `UserAgent` helper and two outbound-request changes so CCP's login/JWKS endpoints see a compliant identifying header.

**Tech Stack:** Go 1.26.1, `github.com/modelcontextprotocol/go-sdk` v1.6.1 (`mcp` subpackage, aliased `mcpsdk`), `github.com/fnt-eve/goesi-openapi`, `github.com/MicahParks/jwkset`, `github.com/MicahParks/keyfunc/v3`, `github.com/urfave/cli/v3`.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- `CGO_ENABLED=0` for the shipped binary; `go test -race` needs CGO enabled, so never set `CGO_ENABLED=0` globally in CI or local scripts.
- No environment variables for config/credentials. Config lives under `~/.neat/`; credentials live in the OS keychain only.
- No client secret — the ESI app is PKCE-only.
- Wrap every error with context: `fmt.Errorf("operation: %w", err)`. Never discard an error (`_ = err`).
- External test packages by default (`package foo_test`); `*_internal_test.go` only when a test genuinely needs an unexported symbol.
- Table-driven tests only where they naturally fit; independent, clearly-named tests otherwise. Standard library only — no `testify`.
- `keyring.MockInit()` before any test touches keychain-backed code.
- User-Agent for all outbound CCP requests: `"neat/<version> (robertinthecloud@icloud.com; +https://github.com/rusher2004/new-eden-agent-toolkit)"`.
- `mcp/` package files alias the SDK import as `mcpsdk` (`mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"`) to avoid shadowing the package's own name — same rationale as `commands` aliasing `cli`.
- Docs (`CLAUDE.md`, `README.md`) updated in the same PR as the behavior they describe, per this repo's doc-sync rule.
- Design spec: `docs/superpowers/specs/2026-07-24-mcp-server-design.md`. Foundation spec: `docs/superpowers/specs/2026-07-23-foundation-design.md`.

---

### Task 1: `sso/` — ESI User-Agent compliance

**Files:**
- Create: `sso/useragent.go`
- Create: `sso/useragent_internal_test.go`
- Modify: `sso/oauth.go`

**Interfaces:**
- Produces: `UserAgent(version string) string`; changes `AddCharacter(ctx context.Context, clientID, userAgent string, scopes []string) (Character, error)` (was `AddCharacter(ctx, clientID string, scopes []string)`).
- Consumes: nothing from other tasks in this plan — fully self-contained, and independently testable/mergeable on its own.

- [ ] **Step 1: Write the failing tests**

Create `sso/useragent_internal_test.go`:

```go
package sso

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserAgent_Format(t *testing.T) {
	got := UserAgent("1.2.3")
	want := "neat/1.2.3 (robertinthecloud@icloud.com; +https://github.com/rusher2004/new-eden-agent-toolkit)"
	if got != want {
		t.Errorf("UserAgent(%q) = %q, want %q", "1.2.3", got, want)
	}
}

func TestUserAgentTransport_SetsHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := userAgentClient("neat/test-agent")
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	resp.Body.Close()

	if gotHeader != "neat/test-agent" {
		t.Errorf("received User-Agent = %q, want %q", gotHeader, "neat/test-agent")
	}
}
```

This file uses `package sso` (not `sso_test`) because it exercises `userAgentClient`, an unexported symbol — consistent with this repo's `*_internal_test.go` convention.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sso/... -run 'TestUserAgent' -v`

Expected: FAIL to compile — `undefined: UserAgent` and `undefined: userAgentClient` (neither exists yet).

- [ ] **Step 3: Implement `sso/useragent.go`**

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
	// HTTP client, so this file builds the keyfunc itself instead of using
	// that helper. If goesi-openapi ever adds User-Agent support directly,
	// this duplication and newKeyfunc can go away in favor of calling
	// ESIDefaultKeyfunc again.
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

`github.com/MicahParks/jwkset` and `github.com/MicahParks/keyfunc/v3` are already in `go.sum` as indirect dependencies of `goesi-openapi`. After adding this file, run `go mod tidy` to promote them to direct dependencies in `go.mod`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go mod tidy && go test ./sso/... -run 'TestUserAgent' -v`

Expected: both `TestUserAgent_Format` and `TestUserAgentTransport_SetsHeader` PASS.

- [ ] **Step 5: Modify `sso/oauth.go` to use the new User-Agent plumbing**

In `sso/oauth.go`, replace:

```go
func AddCharacter(ctx context.Context, clientID string, scopes []string) (Character, error) {
	cfg, err := newOAuthConfig(ctx, clientID)
	if err != nil {
		return Character{}, fmt.Errorf("build OAuth config: %w", err)
	}
	return addCharacter(ctx, cfg, scopes)
}

func newOAuthConfig(ctx context.Context, clientID string) (oauthConfig, error) {
	keyFunc, err := goesi.ESIDefaultKeyfunc(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch ESI JWT signing keys: %w", err)
	}
	cfg, err := goesi.NewConfig(clientID, RedirectURI, &keyFunc)
	if err != nil {
		return nil, fmt.Errorf("build ESI OAuth2 config: %w", err)
	}
	return cfg, nil
}
```

with:

```go
func AddCharacter(ctx context.Context, clientID, userAgent string, scopes []string) (Character, error) {
	cfg, err := newOAuthConfig(ctx, clientID, userAgent)
	if err != nil {
		return Character{}, fmt.Errorf("build OAuth config: %w", err)
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, userAgentClient(userAgent))
	return addCharacter(ctx, cfg, scopes)
}

func newOAuthConfig(ctx context.Context, clientID, userAgent string) (oauthConfig, error) {
	keyFunc, err := newKeyfunc(ctx, userAgent)
	if err != nil {
		return nil, err
	}
	cfg, err := goesi.NewConfig(clientID, RedirectURI, &keyFunc)
	if err != nil {
		return nil, fmt.Errorf("build ESI OAuth2 config: %w", err)
	}
	return cfg, nil
}
```

No other file in the repo calls `AddCharacter` or `newOAuthConfig` yet (nothing outside `sso/` does today — `mcp/`'s call site is added in Task 2), and `sso/oauth_test.go`'s existing tests exercise `addCharacter` (unexported) directly via `ExportedAddCharacter`, which does not change signature — so no test file needs updating for this step.

**Known gap, flagged not fixed:** `goesi.Config.RefreshToken` is exported but unused today (no token-refresh code exists yet). Whoever adds it later must also inject `oauth2.HTTPClient` into that call's context, or refreshes will silently use the default User-Agent.

- [ ] **Step 6: Run the full `sso` package test suite and vet**

Run: `go build ./... && go vet ./... && go test ./sso/... -race -v`

Expected: PASS for every existing test (`TestAddCharacter_FullFlow_Succeeds`, `TestAddCharacter_CallbackStateMismatch_Fails`, `TestAddCharacter_SecondCallWhileFirstInFlight_Fails`, plus the two new User-Agent tests) with no build or vet errors.

- [ ] **Step 7: Commit**

```bash
git add sso/useragent.go sso/useragent_internal_test.go sso/oauth.go go.mod go.sum
git commit -m "feat: add CCP-compliant User-Agent to sso outbound requests"
```

---

### Task 2: `mcp/` package foundation + `add_character` tool

**Files:**
- Create: `mcp/server.go`
- Create: `mcp/add_character.go`
- Create: `mcp/add_character_internal_test.go`

**Interfaces:**
- Consumes: `sso.AddCharacter(ctx, clientID, userAgent string, scopes []string) (sso.Character, error)` and `sso.UserAgent(version string) string` (Task 1); `config.Store.Load() (config.Config, error)`, `store.Save(config.Config) error`, `config.Character{ID int32, Name string}` (existing).
- Produces: `NewServer(version, clientID string, store *config.Store) *mcpsdk.Server` (Task 5 depends on this); `AddCharacterOutput{CharacterID int32, CharacterName string}`.

- [ ] **Step 1: Add the go-sdk dependency**

Run: `go get github.com/modelcontextprotocol/go-sdk@v1.6.1 && go mod tidy`

Expected: `go.mod` gains `github.com/modelcontextprotocol/go-sdk v1.6.1` as a direct requirement (it will show as indirect until a file actually imports it — that happens in Step 3 below, so run `go mod tidy` again after Step 3 if it's still marked indirect).

- [ ] **Step 2: Write the failing test**

Create `mcp/add_character_internal_test.go`:

```go
package mcp

import (
	"context"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func TestAddCharacterHandler_UpsertsIntoConfig(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := addCharacter
	t.Cleanup(func() { addCharacter = orig })
	addCharacter = func(ctx context.Context, clientID, userAgent string, scopes []string) (sso.Character, error) {
		return sso.Character{ID: 42, Name: "Test Pilot"}, nil
	}

	handler := addCharacterHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 42 || out.CharacterName != "Test Pilot" {
		t.Errorf("out = %+v, want {42 Test Pilot}", out)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0] != (config.Character{ID: 42, Name: "Test Pilot"}) {
		t.Errorf("cfg.Characters = %+v, want one entry {42 Test Pilot}", cfg.Characters)
	}
}

func TestAddCharacterHandler_ReAddSameCharacter_ReplacesNotDuplicates(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := addCharacter
	t.Cleanup(func() { addCharacter = orig })
	addCharacter = func(ctx context.Context, clientID, userAgent string, scopes []string) (sso.Character, error) {
		return sso.Character{ID: 42, Name: "Test Pilot"}, nil
	}

	handler := addCharacterHandler("v1", "client-id", store)
	if _, _, err := handler(context.Background(), nil, nil); err != nil {
		t.Fatalf("first handler call: %v", err)
	}

	addCharacter = func(ctx context.Context, clientID, userAgent string, scopes []string) (sso.Character, error) {
		return sso.Character{ID: 42, Name: "Renamed Pilot"}, nil
	}
	if _, _, err := handler(context.Background(), nil, nil); err != nil {
		t.Fatalf("second handler call: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0].Name != "Renamed Pilot" {
		t.Errorf("cfg.Characters = %+v, want exactly one updated entry", cfg.Characters)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestAddCharacterHandler -v`

Expected: FAIL to compile — package `mcp` doesn't exist yet (`no Go files in .../mcp`), and `addCharacter`/`addCharacterHandler` are undefined.

- [ ] **Step 4: Implement `mcp/add_character.go`**

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// addCharacter is overridable in tests so no real browser/network/OAuth2
// flow runs during unit tests.
var addCharacter = sso.AddCharacter

// allScopes is the full ESI scope list neat currently requests at login.
// Deliberately broad during development so any ESI endpoint can be
// exercised while domain tools are being built; narrowing this to a
// minimal per-tool set is a follow-up once real tools exist and their
// actual scope needs are known.
var allScopes = []string{
	"publicData",
	"esi-calendar.respond_calendar_events.v1",
	"esi-calendar.read_calendar_events.v1",
	"esi-location.read_location.v1",
	"esi-location.read_ship_type.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-skills.read_skills.v1",
	"esi-skills.read_skillqueue.v1",
	"esi-wallet.read_corporation_wallet.v1",
	"esi-search.search_structures.v1",
	"esi-clones.read_clones.v1",
	"esi-characters.read_contacts.v1",
	"esi-killmails.read_killmails.v1",
	"esi-corporations.read_corporation_membership.v1",
	"esi-assets.read_assets.v1",
	"esi-planets.manage_planets.v1",
	"esi-fleets.read_fleet.v1",
	"esi-fleets.write_fleet.v1",
	"esi-ui.open_window.v1",
	"esi-characters.write_contacts.v1",
	"esi-fittings.read_fittings.v1",
	"esi-fittings.write_fittings.v1",
	"esi-markets.structure_markets.v1",
	"esi-corporations.read_structures.v1",
	"esi-characters.read_loyalty.v1",
	"esi-characters.read_chat_channels.v1",
	"esi-characters.read_medals.v1",
	"esi-characters.read_standings.v1",
	"esi-characters.read_agents_research.v1",
	"esi-industry.read_character_jobs.v1",
	"esi-markets.read_character_orders.v1",
	"esi-characters.read_blueprints.v1",
	"esi-characters.read_corporation_roles.v1",
	"esi-location.read_online.v1",
	"esi-contracts.read_character_contracts.v1",
	"esi-clones.read_implants.v1",
	"esi-characters.read_fatigue.v1",
	"esi-killmails.read_corporation_killmails.v1",
	"esi-corporations.track_members.v1",
	"esi-wallet.read_corporation_wallets.v1",
	"esi-characters.read_notifications.v1",
	"esi-corporations.read_divisions.v1",
	"esi-corporations.read_contacts.v1",
	"esi-assets.read_corporation_assets.v1",
	"esi-corporations.read_titles.v1",
	"esi-corporations.read_blueprints.v1",
	"esi-corporations.read_standings.v1",
	"esi-industry.read_corporation_jobs.v1",
	"esi-markets.read_corporation_orders.v1",
	"esi-corporations.read_container_logs.v1",
	"esi-industry.read_character_mining.v1",
	"esi-industry.read_corporation_mining.v1",
	"esi-planets.read_customs_offices.v1",
	"esi-corporations.read_facilities.v1",
	"esi-corporations.read_medals.v1",
	"esi-characters.read_titles.v1",
	"esi-alliances.read_contacts.v1",
	"esi-corporations.read_fw_stats.v1",
	"esi-corporations.read_projects.v1",
	"esi-corporations.read_freelance_jobs.v1",
	"esi-characters.read_freelance_jobs.v1",
	"esi-structures.read_corporation.v1",
	"esi-structures.read_character.v1",
	"esi-activities.read_character.v1",
	"esi-access.read_lists.v1",
	"esi-wallet.read_character_wallet.v1",
	"esi-universe.read_structures.v1",
	"esi-ui.write_waypoint.v1",
	"esi-contracts.read_corporation_contracts.v1",
	"esi-corporations.read_starbases.v1",
	"esi-characters.read_fw_stats.v1",
}

// AddCharacterOutput is add_character's tool result.
type AddCharacterOutput struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
}

func addCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[any, AddCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, AddCharacterOutput, error) {
		char, err := addCharacter(ctx, clientID, sso.UserAgent(version), allScopes)
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("add character: %w", err)
		}

		cfg, err := store.Load()
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("load config: %w", err)
		}
		upsertCharacter(&cfg, config.Character{ID: char.ID, Name: char.Name})
		if err := store.Save(cfg); err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("save config: %w", err)
		}

		return nil, AddCharacterOutput{CharacterID: char.ID, CharacterName: char.Name}, nil
	}
}

// upsertCharacter replaces the entry in cfg.Characters matching c.ID, or
// appends c if no existing entry matches.
func upsertCharacter(cfg *config.Config, c config.Character) {
	for i, existing := range cfg.Characters {
		if existing.ID == c.ID {
			cfg.Characters[i] = c
			return
		}
	}
	cfg.Characters = append(cfg.Characters, c)
}
```

The `allScopes` list above is the current full contents of (git-ignored) `app_config.json`'s `"scopes"` array, copied verbatim.

- [ ] **Step 5: Implement `mcp/server.go`**

```go
// Package mcp builds neat's MCP server and registers its character tools.
package mcp

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// NewServer builds neat's MCP server with add_character registered. (list_characters
// and remove_character are registered here too, once Tasks 3 and 4 add them —
// for now this task registers only add_character.)
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: version}, nil)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "add_character",
		Description: "Log in a new EVE Online character via ESI OAuth2, opening the user's browser to complete authentication.",
	}, addCharacterHandler(version, clientID, store))

	return server
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go mod tidy && go build ./... && go vet ./... && go test ./mcp/... -race -v`

Expected: both `TestAddCharacterHandler_UpsertsIntoConfig` and
`TestAddCharacterHandler_ReAddSameCharacter_ReplacesNotDuplicates` PASS; build and vet clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/server.go mcp/add_character.go mcp/add_character_internal_test.go go.mod go.sum
git commit -m "feat: add mcp package with add_character tool"
```

---

### Task 3: `mcp/list_characters` tool

**Files:**
- Create: `mcp/list_characters.go`
- Create: `mcp/list_characters_internal_test.go`
- Modify: `mcp/server.go`

**Interfaces:**
- Consumes: `config.Store.Load() (config.Config, error)`, `config.Character{ID, Name}` (existing).
- Produces: `CharacterInfo{CharacterID int32, CharacterName string}` (Task 4 reuses this as `remove_character`'s output type), `ListCharactersOutput{Characters []CharacterInfo}`.

- [ ] **Step 1: Write the failing test**

Create `mcp/list_characters_internal_test.go`:

```go
package mcp

import (
	"context"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharactersHandler_Empty(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharactersHandler(store)
	_, out, err := handler(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Characters == nil {
		t.Error("out.Characters is nil, want an empty (non-nil) slice")
	}
	if len(out.Characters) != 0 {
		t.Errorf("out.Characters = %+v, want empty", out.Characters)
	}
}

func TestListCharactersHandler_ReturnsKnownCharacters(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{
		{ID: 1, Name: "Alpha Pilot"},
		{ID: 2, Name: "Beta Pilot"},
	}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	handler := listCharactersHandler(store)
	_, out, err := handler(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []CharacterInfo{
		{CharacterID: 1, CharacterName: "Alpha Pilot"},
		{CharacterID: 2, CharacterName: "Beta Pilot"},
	}
	if len(out.Characters) != len(want) {
		t.Fatalf("out.Characters = %+v, want %+v", out.Characters, want)
	}
	for i := range want {
		if out.Characters[i] != want[i] {
			t.Errorf("out.Characters[%d] = %+v, want %+v", i, out.Characters[i], want[i])
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListCharactersHandler -v`

Expected: FAIL to compile — `undefined: listCharactersHandler`, `undefined: CharacterInfo`, `undefined: config.Config` fields mismatch is not expected (that type already exists) but the handler/type in `mcp` do not.

- [ ] **Step 3: Implement `mcp/list_characters.go`**

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// CharacterInfo is the MCP-facing representation of a known character,
// deliberately distinct from config.Character so the tool contract doesn't
// leak the internal storage schema.
type CharacterInfo struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
}

// ListCharactersOutput is list_characters' tool result.
type ListCharactersOutput struct {
	Characters []CharacterInfo `json:"characters"`
}

func listCharactersHandler(store *config.Store) mcpsdk.ToolHandlerFor[any, ListCharactersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, ListCharactersOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCharactersOutput{}, fmt.Errorf("load config: %w", err)
		}

		out := ListCharactersOutput{Characters: make([]CharacterInfo, 0, len(cfg.Characters))}
		for _, c := range cfg.Characters {
			out.Characters = append(out.Characters, CharacterInfo{CharacterID: c.ID, CharacterName: c.Name})
		}
		return nil, out, nil
	}
}
```

- [ ] **Step 4: Register the tool in `mcp/server.go`**

In `mcp/server.go`, add before `return server`:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_characters",
		Description: "List EVE Online characters currently logged in to neat.",
	}, listCharactersHandler(store))
```

Also update the function's doc comment to drop the "for now this task registers only add_character" note from Task 2, since two tools are now registered.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./mcp/... -race -v`

Expected: `TestListCharactersHandler_Empty` and `TestListCharactersHandler_ReturnsKnownCharacters` PASS, plus Task 2's tests still PASS.

- [ ] **Step 6: Commit**

```bash
git add mcp/list_characters.go mcp/list_characters_internal_test.go mcp/server.go
git commit -m "feat: add list_characters MCP tool"
```

---

### Task 4: `mcp/remove_character` tool + cross-tool wiring test

**Files:**
- Create: `mcp/remove_character.go`
- Create: `mcp/remove_character_internal_test.go`
- Create: `mcp/server_test.go`
- Modify: `mcp/server.go`

**Interfaces:**
- Consumes: `creds.DeleteCharacterRefreshToken(characterID int32) error` (existing); `CharacterInfo` (Task 3).
- Produces: `ErrCharacterNotFound`, `RemoveCharacterInput{CharacterID int32}`, `removeCharacterHandler`. Nothing later in this plan depends on these — Task 5 only needs `NewServer`, already produced by Task 2.

- [ ] **Step 1: Write the failing test**

Create `mcp/remove_character_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

func TestRemoveCharacterHandler_NotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := removeCharacterHandler(store)
	_, _, err := handler(context.Background(), nil, RemoveCharacterInput{CharacterID: 42})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRemoveCharacterHandler_RemovesConfigAndCreds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{
		{ID: 1, Name: "Alpha Pilot"},
		{ID: 2, Name: "Beta Pilot"},
	}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	if err := creds.SaveCharacterRefreshToken(1, "refresh-token-1"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	handler := removeCharacterHandler(store)
	_, out, err := handler(context.Background(), nil, RemoveCharacterInput{CharacterID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out != (CharacterInfo{CharacterID: 1, CharacterName: "Alpha Pilot"}) {
		t.Errorf("out = %+v, want {1 Alpha Pilot}", out)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0].ID != 2 {
		t.Errorf("cfg.Characters = %+v, want only character 2 remaining", cfg.Characters)
	}

	if _, err := creds.CharacterRefreshToken(1); !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("CharacterRefreshToken(1) err = %v, want wrapping creds.ErrNotFound", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mcp/... -run TestRemoveCharacterHandler -v`

Expected: FAIL to compile — `undefined: removeCharacterHandler`, `undefined: RemoveCharacterInput`, `undefined: ErrCharacterNotFound`.

- [ ] **Step 3: Implement `mcp/remove_character.go`**

```go
package mcp

import (
	"context"
	"errors"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

// ErrCharacterNotFound is returned (wrapped) when remove_character is
// called with a character ID that isn't in the local config.
var ErrCharacterNotFound = errors.New("character not found")

// RemoveCharacterInput is remove_character's tool input.
type RemoveCharacterInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to remove"`
}

func removeCharacterHandler(store *config.Store) mcpsdk.ToolHandlerFor[RemoveCharacterInput, CharacterInfo] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RemoveCharacterInput) (*mcpsdk.CallToolResult, CharacterInfo, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("load config: %w", err)
		}

		idx := -1
		for i, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, CharacterInfo{}, fmt.Errorf("remove character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		removed := cfg.Characters[idx]

		if err := creds.DeleteCharacterRefreshToken(removed.ID); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("delete refresh token: %w", err)
		}

		cfg.Characters = append(cfg.Characters[:idx], cfg.Characters[idx+1:]...)
		if err := store.Save(cfg); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("save config: %w", err)
		}

		return nil, CharacterInfo{CharacterID: removed.ID, CharacterName: removed.Name}, nil
	}
}
```

Deletion order matters: `creds.DeleteCharacterRefreshToken` runs *before* `store.Save`, so a failure partway through leans toward "token gone, config stale" rather than "token still present, config says removed" — the safer failure mode.

- [ ] **Step 4: Register the tool in `mcp/server.go`**

In `mcp/server.go`, add before `return server`:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "remove_character",
		Description: "Remove a logged-in EVE Online character, deleting its stored credentials.",
	}, removeCharacterHandler(store))
```

- [ ] **Step 5: Run unit tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./mcp/... -race -v`

Expected: `TestRemoveCharacterHandler_NotFound` and `TestRemoveCharacterHandler_RemovesConfigAndCreds` PASS, plus every test from Tasks 2 and 3.

- [ ] **Step 6: Write the cross-tool wiring test**

Create `mcp/server_test.go` (external test package — this test only calls exported symbols):

```go
package mcp_test

import (
	"context"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

func connectedSession(t *testing.T, store *config.Store) (*mcpsdk.ClientSession, *mcpsdk.ServerSession) {
	t.Helper()
	server := mcp.NewServer("test", "client-id", store)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() {
		clientSession.Close()
		serverSession.Wait()
	})
	return clientSession, serverSession
}

func TestServer_AllToolsRegistered(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	clientSession, _ := connectedSession(t, store)

	res, err := clientSession.ListTools(context.Background(), &mcpsdk.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"add_character", "list_characters", "remove_character"} {
		if !names[want] {
			t.Errorf("missing tool %q, got %v", want, names)
		}
	}
}

func TestServer_RemoveCharacterNotFound_SurfacesAsToolError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	clientSession, _ := connectedSession(t, store)

	res, err := clientSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "remove_character",
		Arguments: map[string]any{"characterId": 12345},
	})
	if err != nil {
		t.Fatalf("CallTool transport-level error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true for missing character, got %+v", res)
	}

	// GetError() always returns nil client-side (the SDK does not
	// serialize the underlying Go error across the wire) — the only
	// client-visible signal is IsError plus the human-readable text
	// content, so that's what this test asserts against.
	var text string
	for _, c := range res.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			text = tc.Text
		}
	}
	if !strings.Contains(text, "character not found") {
		t.Errorf("content text = %q, want it to contain %q", text, "character not found")
	}
}
```

- [ ] **Step 7: Run the wiring test to verify it passes**

Run: `go test ./mcp/... -run TestServer -v`

Expected: both `TestServer_AllToolsRegistered` and `TestServer_RemoveCharacterNotFound_SurfacesAsToolError` PASS.

- [ ] **Step 8: Run the full `mcp` package suite**

Run: `go build ./... && go vet ./... && go test ./mcp/... -race -v`

Expected: every test in the `mcp` package (Tasks 2, 3, and 4) PASSes.

- [ ] **Step 9: Commit**

```bash
git add mcp/remove_character.go mcp/remove_character_internal_test.go mcp/server_test.go mcp/server.go
git commit -m "feat: add remove_character MCP tool, verify full server wiring"
```

---

### Task 5: `commands/mcp.go` startup wiring + `cmd/main.go` + `Makefile`

**Files:**
- Create: `commands/mcp.go`
- Create: `commands/mcp_internal_test.go`
- Modify: `commands/commands.go`
- Modify: `commands/callback_handler_test.go`, `commands/setup_test.go`, `commands/version_test.go` (existing `commands.Root(...)` call sites need a second argument)
- Modify: `cmd/main.go`
- Modify: `Makefile`

**Interfaces:**
- Consumes: `mcp.NewServer(version, clientID string, store *config.Store) *mcpsdk.Server` (Task 2); `config.NewStore() (*config.Store, error)`, `urlscheme.Register(launcherPath string) error` (existing).
- Produces: `mcpCommand(version, clientID string) *cli.Command`, wired into `commands.Root`.

- [ ] **Step 1: Write the failing test**

Create `commands/mcp_internal_test.go` (`package commands` — needs the unexported `ensureLauncher` and `currentExecutable` test seam):

```go
package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLauncher_CopiesWhenMissing(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	fakeBinary := filepath.Join(t.TempDir(), "neat-fake")
	if err := os.WriteFile(fakeBinary, []byte("fake binary contents v1"), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	orig := currentExecutable
	t.Cleanup(func() { currentExecutable = orig })
	currentExecutable = func() (string, error) { return fakeBinary, nil }

	target, err := ensureLauncher()
	if err != nil {
		t.Fatalf("ensureLauncher: %v", err)
	}
	wantTarget := filepath.Join(tmpHome, ".neat", "bin", "neat")
	if target != wantTarget {
		t.Errorf("target = %q, want %q", target, wantTarget)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "fake binary contents v1" {
		t.Errorf("target contents = %q, want copied fake binary contents", got)
	}
}

func TestEnsureLauncher_NoOpWhenAlreadyCurrent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	fakeBinary := filepath.Join(t.TempDir(), "neat-fake")
	if err := os.WriteFile(fakeBinary, []byte("v1"), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	orig := currentExecutable
	t.Cleanup(func() { currentExecutable = orig })
	currentExecutable = func() (string, error) { return fakeBinary, nil }
	target, err := ensureLauncher()
	if err != nil {
		t.Fatalf("first ensureLauncher: %v", err)
	}

	// Simulate running from the installed location: current == target.
	currentExecutable = func() (string, error) { return target, nil }
	target2, err := ensureLauncher()
	if err != nil {
		t.Fatalf("second ensureLauncher: %v", err)
	}
	if target2 != target {
		t.Errorf("second target = %q, want %q", target2, target)
	}
}

func TestEnsureLauncher_OverwritesWhenStale(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	fakeBinary := filepath.Join(t.TempDir(), "neat-fake")
	if err := os.WriteFile(fakeBinary, []byte("v1"), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	orig := currentExecutable
	t.Cleanup(func() { currentExecutable = orig })
	currentExecutable = func() (string, error) { return fakeBinary, nil }
	target, err := ensureLauncher()
	if err != nil {
		t.Fatalf("first ensureLauncher: %v", err)
	}

	if err := os.WriteFile(fakeBinary, []byte("v2"), 0o700); err != nil {
		t.Fatalf("rewrite fake binary: %v", err)
	}
	if _, err := ensureLauncher(); err != nil {
		t.Fatalf("second ensureLauncher: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("target contents = %q, want %q", got, "v2")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./commands/... -run TestEnsureLauncher -v`

Expected: FAIL to compile — `undefined: currentExecutable`, `undefined: ensureLauncher`.

- [ ] **Step 3: Implement `commands/mcp.go`**

```go
package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/urlscheme"
)

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

			server := mcp.NewServer(version, clientID, store)
			return server.Run(ctx, &mcpsdk.StdioTransport{})
		},
	}
}

// currentExecutable resolves the path to the currently running binary.
// Overridable in tests so they don't depend on the actual go test binary.
var currentExecutable = os.Executable

// ensureLauncher makes sure ~/.neat/bin/neat exists and matches the
// currently running binary, copying it into place if missing or the
// running binary lives elsewhere. Returns the resolved launcher path.
func ensureLauncher() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	target := filepath.Join(home, ".neat", "bin", "neat")

	current, err := currentExecutable()
	if err != nil {
		return "", fmt.Errorf("resolve running executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(current); err == nil {
		current = resolved
	}

	if resolvedTarget, err := filepath.EvalSymlinks(target); err == nil && resolvedTarget == current {
		return target, nil
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", fmt.Errorf("create launcher directory: %w", err)
	}

	src, err := os.Open(current)
	if err != nil {
		return "", fmt.Errorf("open running executable: %w", err)
	}
	defer src.Close()

	tmp := target + ".tmp"
	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return "", fmt.Errorf("create launcher temp file: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return "", fmt.Errorf("copy launcher binary: %w", err)
	}
	if err := dst.Close(); err != nil {
		return "", fmt.Errorf("close launcher temp file: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		return "", fmt.Errorf("rename launcher into place: %w", err)
	}

	return target, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./commands/... -race -v`

Expected: `TestEnsureLauncher_CopiesWhenMissing`, `TestEnsureLauncher_NoOpWhenAlreadyCurrent`, and `TestEnsureLauncher_OverwritesWhenStale` all PASS, plus every existing `commands` test.

- [ ] **Step 5: Wire `mcpCommand` into the command tree**

In `commands/commands.go`, change:

```go
// Root builds the top-level neat command, wiring in every subcommand.
func Root(version string) *cli.Command {
	return &cli.Command{
		Name:  "neat",
		Usage: "New Eden Agent Toolkit",
		Commands: []*cli.Command{
			versionCommand(version),
			setupCommand(),
			callbackHandlerCommand(),
		},
	}
}
```

to:

```go
// Root builds the top-level neat command, wiring in every subcommand.
func Root(version, clientID string) *cli.Command {
	return &cli.Command{
		Name:  "neat",
		Usage: "New Eden Agent Toolkit",
		Commands: []*cli.Command{
			versionCommand(version),
			setupCommand(),
			callbackHandlerCommand(),
			mcpCommand(version, clientID),
		},
	}
}
```

- [ ] **Step 6: Update existing test call sites for `Root`'s new signature**

`commands.Root` is called with a single argument in three existing test files. Update each:

Run:
```bash
sed -i '' 's/commands\.Root("dev")/commands.Root("dev", "test-client-id")/g' commands/callback_handler_test.go commands/setup_test.go
sed -i '' 's/commands\.Root("1.2.3")/commands.Root("1.2.3", "test-client-id")/g' commands/version_test.go
```

(On Linux, drop the empty `''` after `-i`.) This updates 5 call sites in `commands/callback_handler_test.go`, 4 in `commands/setup_test.go`, and 1 in `commands/version_test.go` — all currently read `commands.Root("dev")` or `commands.Root("1.2.3")` verbatim, so the substitution is exact and complete (verified by grepping for `commands.Root` across the repo before writing this step).

- [ ] **Step 7: Update `cmd/main.go`**

Change:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	root := commands.Root(version)
	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "neat: %v\n", err)
		os.Exit(1)
	}
}
```

to:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

// clientID is the ESI OAuth2 client ID, set at build time via
// -ldflags "-X main.clientID=<id>". Never committed to source — see
// app_config.json (git-ignored) and the Makefile's build target.
var clientID = "dev"

func main() {
	root := commands.Root(version, clientID)
	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "neat: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 8: Add the local-dev `build` target to the `Makefile`**

In `Makefile`, change the `.PHONY` line and append a new target:

```makefile
.PHONY: urlscheme-macos-bundle build

BUNDLE := urlscheme/macos/build/NeatURLHandler.app

urlscheme-macos-bundle:
	mkdir -p $(BUNDLE)/Contents/MacOS
	swiftc urlscheme/macos/main.swift -o $(BUNDLE)/Contents/MacOS/NeatURLHandler
	cp urlscheme/macos/Info.plist $(BUNDLE)/Contents/Info.plist

build:
	go build -ldflags "-X main.version=dev -X main.clientID=$$(jq -r .clientId app_config.json)" -o bin/neat ./cmd
```

This target is local-dev convenience only (requires `jq` and a local `app_config.json`), not release automation — release-time `-ldflags` injection is a separate, not-yet-built concern (see the foundation spec's `Build & release` section).

- [ ] **Step 9: Run the full build and test suite**

Run: `go build ./... && go vet ./... && go test ./... -race -v`

Expected: every package's tests PASS (`commands`, `config`, `creds`, `mcp`, `sso`, `urlscheme`), full build and vet clean. On macOS, run `make urlscheme-macos-bundle` first if it hasn't been run yet this session (see `CLAUDE.md`'s Build section).

- [ ] **Step 10: Commit**

```bash
git add commands/mcp.go commands/mcp_internal_test.go commands/commands.go commands/callback_handler_test.go commands/setup_test.go commands/version_test.go cmd/main.go Makefile
git commit -m "feat: add neat mcp command with launcher/urlscheme startup wiring"
```

---

### Task 6: Docs sync (`CLAUDE.md`, `README.md`)

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Interfaces:** none — documentation only, no code interfaces produced or consumed.

- [ ] **Step 1: Update `CLAUDE.md`'s package table**

Replace:

```markdown
| `sso/` | ESI OAuth2/PKCE login flow: PKCE + state generation, a two-socket Unix IPC handoff for the callback (a lock socket that's bound but never `Accept()`-ed, plus a separate delivery socket), orchestration via `github.com/fnt-eve/goesi-openapi`, refresh-token persistence via `creds/` |

More packages (`esi/`, `sde/`, `mcp/`) land in follow-up plans — see `docs/superpowers/plans/`.
```

with:

```markdown
| `sso/` | ESI OAuth2/PKCE login flow: PKCE + state generation, a two-socket Unix IPC handoff for the callback (a lock socket that's bound but never `Accept()`-ed, plus a separate delivery socket), orchestration via `github.com/fnt-eve/goesi-openapi`, refresh-token persistence via `creds/`. All outbound requests to CCP (JWKS fetch, OAuth2 token exchange) carry `sso.UserAgent(version)`, per ESI's User-Agent best practices |
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its three tools: `add_character`, `list_characters`, `remove_character` |

More packages (`esi/`, `sde/`) land in follow-up plans — see `docs/superpowers/plans/`.
```

Also update the `commands/` row to mention `mcp`:

```markdown
| `commands/` | `urfave/cli/v3` command tree (`version`, `setup`, `mcp`, `callback-handler` (hidden — invoked only by the OS on URL-scheme activation)) |
```

- [ ] **Step 2: Add a User-Agent hard rule to `CLAUDE.md`**

In the `## Hard rules` section, add:

```markdown
- **ESI User-Agent compliance**: every outbound request to CCP's login/ESI endpoints must carry `sso.UserAgent(version)`. CCP's best-practices guidance treats non-compliant User-Agents as grounds for a temporary ban — see `sso/useragent.go`.
```

- [ ] **Step 3: Update `CLAUDE.md`'s Build section**

Replace:

```bash
go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev)" -o neat ./cmd
```

with:

```bash
go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev) -X main.clientID=$(jq -r .clientId app_config.json)" -o neat ./cmd
```

and add a line above it:

```markdown
`clientID` (the ESI OAuth2 client ID) is also injected via `-ldflags` from the git-ignored
`app_config.json` — `make build` wraps this for local development.
```

- [ ] **Step 4: Update `README.md`**

Replace the `## Status` section:

```markdown
Foundation phase. CLI skeleton (`config/`, `creds/`) and the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`) are built. No MCP tool exposure yet — `sso.AddCharacter` has no CLI/MCP
entrypoint, and the SDE cache and plugin/extension packaging don't exist yet either. See the plans
directory for what's next.
```

with:

```markdown
Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), and the MCP server with its character-management tools (`mcp/`,
`neat mcp`) are built. The SDE cache and plugin/extension packaging don't exist yet. See the plans
directory for what's next.
```

Replace the package table row and follow-up line the same way as Step 1 above (`sso/` gains the User-Agent note, add the `mcp/` row, `commands/` row mentions `mcp`, drop `mcp/` from the "land in follow-up plans" list).

Update the Build section's `go build` example the same way as CLAUDE.md's Step 3.

- [ ] **Step 5: Sanity-check the docs against the code**

Run: `grep -n "mcp\b" CLAUDE.md README.md`

Expected: both files mention the `mcp/` package, the `neat mcp` command, and the User-Agent rule; no leftover references to "no MCP tool exposure yet."

- [ ] **Step 6: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: document the mcp package, neat mcp command, and User-Agent rule"
```
