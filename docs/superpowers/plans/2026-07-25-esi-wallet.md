# `esi/` Package + `get_wallet_balance` Tool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `esi/` package and one real, end-to-end ESI-backed MCP tool, `get_wallet_balance`, proving the full pipeline — stored refresh token → authenticated ESI call → MCP tool result.

**Architecture:** `sso/client.go` caches a per-character auto-refreshing `*http.Client` (via `golang.org/x/oauth2`'s own refresh machinery — no manual expiry tracking needed, verified during design). `esi/` is a thin, stateless facade that asks `sso/` for that client and wraps `goesi-openapi`'s generated ESI API client around it. `mcp/wallet.go` is the fourth MCP tool, following the same conventions as the first three.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `golang.org/x/oauth2`, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context: `fmt.Errorf("operation: %w", err)`. Never discard an error.
- External test packages by default (`package foo_test`); `*_internal_test.go` only when a test genuinely needs an unexported symbol.
- `keyring.MockInit()` before any test touches keychain-backed code (`creds`).
- `mcp/` package files alias the go-sdk import as `mcpsdk`. `esi/` package files alias `github.com/fnt-eve/goesi-openapi` as `goesi` (matching `sso/`'s existing convention) and the nested `github.com/fnt-eve/goesi-openapi/esi` as `esiapi` (our own package is also named `esi`, so this avoids shadowing).
- Standard library only in tests — no testify.
- Design spec: `docs/superpowers/specs/2026-07-25-esi-wallet-design.md`.

---

### Task 1: `sso/client.go` — cached authenticated HTTP client per character

**Files:**
- Create: `sso/client.go`
- Create: `sso/client_test.go`

**Interfaces:**
- Consumes: `creds.CharacterRefreshToken(characterID int32) (string, error)`, `creds.ErrNotFound` (existing); `goesi.NewConfig(clientID, redirectURL string, keyFunc *jwt.Keyfunc) (*goesi.Config, error)`, `(*goesi.Config).Client(ctx, token) *http.Client` (verified against the real `goesi-openapi` v0.0.37 API during plan-writing); `sso.RedirectURI` (existing constant); `userAgentClient(userAgent string) *http.Client` (existing, unexported, from `sso/useragent.go`).
- Produces: `AuthenticatedClient(ctx context.Context, clientID string, characterID int32, userAgent string) (*http.Client, error)`, `InvalidateClientCache(characterID int32)` — both consumed by Task 2.

- [ ] **Step 1: Write the failing tests**

Create `sso/client_test.go`:

```go
package sso_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func TestAuthenticatedClient_CachesPerCharacter(t *testing.T) {
	keyring.MockInit()
	if err := creds.SaveCharacterRefreshToken(1, "refresh-1"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	c1, err := sso.AuthenticatedClient(context.Background(), "client-id", 1, "test-agent")
	if err != nil {
		t.Fatalf("first AuthenticatedClient: %v", err)
	}
	c2, err := sso.AuthenticatedClient(context.Background(), "client-id", 1, "test-agent")
	if err != nil {
		t.Fatalf("second AuthenticatedClient: %v", err)
	}
	if c1 != c2 {
		t.Errorf("c1 = %p, c2 = %p, want same pointer (cache hit)", c1, c2)
	}
}

func TestAuthenticatedClient_DifferentCharactersGetDifferentClients(t *testing.T) {
	keyring.MockInit()
	if err := creds.SaveCharacterRefreshToken(2, "refresh-2"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}
	if err := creds.SaveCharacterRefreshToken(3, "refresh-3"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	c1, err := sso.AuthenticatedClient(context.Background(), "client-id", 2, "test-agent")
	if err != nil {
		t.Fatalf("AuthenticatedClient(2): %v", err)
	}
	c2, err := sso.AuthenticatedClient(context.Background(), "client-id", 3, "test-agent")
	if err != nil {
		t.Fatalf("AuthenticatedClient(3): %v", err)
	}
	if c1 == c2 {
		t.Error("c1 == c2, want distinct pointers for different characters")
	}
}

func TestAuthenticatedClient_UnknownCharacter_ReturnsError(t *testing.T) {
	keyring.MockInit()
	_, err := sso.AuthenticatedClient(context.Background(), "client-id", 999, "test-agent")
	if !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound", err)
	}
}

func TestInvalidateClientCache_RemovesCachedEntry(t *testing.T) {
	keyring.MockInit()
	if err := creds.SaveCharacterRefreshToken(4, "refresh-4"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	c1, err := sso.AuthenticatedClient(context.Background(), "client-id", 4, "test-agent")
	if err != nil {
		t.Fatalf("first AuthenticatedClient: %v", err)
	}
	sso.InvalidateClientCache(4)
	c2, err := sso.AuthenticatedClient(context.Background(), "client-id", 4, "test-agent")
	if err != nil {
		t.Fatalf("second AuthenticatedClient: %v", err)
	}
	if c1 == c2 {
		t.Error("c1 == c2, want a different pointer after InvalidateClientCache")
	}
}
```

These tests use distinct character IDs (1-4) per test since the cache is a shared package-level
map — reusing an ID across tests could make one test's cached entry leak into another's
assertions.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sso/... -run 'TestAuthenticatedClient|TestInvalidateClientCache' -v`

Expected: FAIL to compile — `undefined: sso.AuthenticatedClient`, `undefined: sso.InvalidateClientCache`.

- [ ] **Step 3: Implement `sso/client.go`**

```go
package sso

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	goesi "github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

var (
	clientCacheMu sync.Mutex
	clientCache   = map[int32]*http.Client{}
)

// AuthenticatedClient returns an authenticated, auto-refreshing HTTP client
// for characterID, suitable for making real ESI API calls. The first call
// for a given character builds and caches the client; later calls for the
// same character reuse it — golang.org/x/oauth2's own token source
// transparently refreshes the access token only when it actually expires,
// so callers never need to check expiry themselves. Verified directly: a
// client built this way hits the token endpoint exactly once across
// multiple resource calls, as long as the refreshed access token's expiry
// hasn't elapsed.
func AuthenticatedClient(ctx context.Context, clientID string, characterID int32, userAgent string) (*http.Client, error) {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()

	if c, ok := clientCache[characterID]; ok {
		return c, nil
	}

	refreshToken, err := creds.CharacterRefreshToken(characterID)
	if err != nil {
		return nil, fmt.Errorf("load refresh token for character %d: %w", characterID, err)
	}

	// No JWKS keyfunc needed: this path never verifies/parses JWT claims,
	// only golang.org/x/oauth2's own refresh-token grant flow.
	cfg, err := goesi.NewConfig(clientID, RedirectURI, nil)
	if err != nil {
		return nil, fmt.Errorf("build ESI OAuth2 config: %w", err)
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, userAgentClient(userAgent))
	token := &oauth2.Token{RefreshToken: refreshToken}
	client := cfg.Client(ctx, token)

	clientCache[characterID] = client
	return client, nil
}

// InvalidateClientCache evicts characterID's cached authenticated client,
// if any. Called by remove_character so a stale in-memory client can't
// keep working against a refresh token that's just been deleted from the
// keychain.
func InvalidateClientCache(characterID int32) {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()
	delete(clientCache, characterID)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./sso/... -race -v`

Expected: all four new tests PASS, plus every pre-existing `sso` test.

- [ ] **Step 5: Commit**

```bash
git add sso/client.go sso/client_test.go
git commit -m "feat: add cached per-character authenticated HTTP client"
```

---

### Task 2: `esi/` package — `Client` + `WalletBalance`

**Files:**
- Create: `esi/client.go`
- Create: `esi/wallet.go`
- Create: `esi/wallet_test.go`

**Interfaces:**
- Consumes: `sso.AuthenticatedClient(ctx, clientID string, characterID int32, userAgent string) (*http.Client, error)` (Task 1); `goesi.NewESIClientWithOptions(httpClient *http.Client, options goesi.ClientOptions) *esiapi.APIClient`, `goesi.ClientOptions{UserAgent, CompatibilityDate, BaseURL string}`, `esiapi.APIClient.WalletAPI.GetCharactersCharacterIdWallet(ctx, characterId int64) ApiGetCharactersCharacterIdWalletRequest`, `.Execute() (float64, *http.Response, error)` — all verified against the real `goesi-openapi` v0.0.37 package during plan-writing.
- Produces: `esi.Client`, `esi.NewClient(ctx, clientID string, characterID int32, userAgent string) (*Client, error)`, `(*Client).WalletBalance(ctx) (float64, error)`, `esi.ExportedNewClientWithOptions(httpClient *http.Client, characterID int32, options goesi.ClientOptions) *Client` — all consumed by Task 3.

- [ ] **Step 1: Write the failing test**

Create `esi/wallet_test.go`:

```go
package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestWalletBalance_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("1234.56"))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	balance, err := client.WalletBalance(context.Background())
	if err != nil {
		t.Fatalf("WalletBalance returned error: %v", err)
	}
	if balance != 1234.56 {
		t.Errorf("balance = %v, want 1234.56", balance)
	}
}

func TestWalletBalance_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.WalletBalance(context.Background()); err == nil {
		t.Fatal("WalletBalance returned nil error, want an error for HTTP 403")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./esi/... -run TestWalletBalance -v`

Expected: FAIL to compile — package `esi` doesn't exist yet (`no Go files in .../esi`).

- [ ] **Step 3: Implement `esi/client.go`**

```go
package esi

import (
	"context"
	"fmt"
	"net/http"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// Client is a thin, per-character wrapper over the generated ESI API
// client. esi/ itself holds no state across calls — sso.AuthenticatedClient
// owns the one cache that matters (the auto-refreshing HTTP client).
type Client struct {
	api         *esiapi.APIClient
	characterID int32
}

// NewClient builds an authenticated ESI client for characterID.
func NewClient(ctx context.Context, clientID string, characterID int32, userAgent string) (*Client, error) {
	httpClient, err := sso.AuthenticatedClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build authenticated client: %w", err)
	}
	return newClientFromHTTPClient(httpClient, characterID, userAgent), nil
}

func newClientFromHTTPClient(httpClient *http.Client, characterID int32, userAgent string) *Client {
	api := goesi.NewESIClientWithOptions(httpClient, goesi.ClientOptions{UserAgent: userAgent})
	return &Client{api: api, characterID: characterID}
}

// ExportedNewClientWithOptions exposes newClientFromHTTPClient for external
// tests, additionally allowing a BaseURL override so tests can point the
// generated client at an httptest.Server instead of real ESI.
func ExportedNewClientWithOptions(httpClient *http.Client, characterID int32, options goesi.ClientOptions) *Client {
	api := goesi.NewESIClientWithOptions(httpClient, options)
	return &Client{api: api, characterID: characterID}
}
```

- [ ] **Step 4: Implement `esi/wallet.go`**

```go
package esi

import (
	"context"
	"fmt"
)

// WalletBalance returns the client's character's current ISK wallet balance.
func (c *Client) WalletBalance(ctx context.Context) (float64, error) {
	balance, _, err := c.api.WalletAPI.GetCharactersCharacterIdWallet(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return 0, fmt.Errorf("get wallet balance: %w", err)
	}
	return balance, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: `TestWalletBalance_ParsesResponse` and `TestWalletBalance_PropagatesHTTPError` PASS.

- [ ] **Step 6: Commit**

```bash
git add esi/client.go esi/wallet.go esi/wallet_test.go
git commit -m "feat: add esi package with Client and WalletBalance"
```

---

### Task 3: `mcp/wallet.go` tool + registration + `remove_character` cache invalidation

**Files:**
- Create: `mcp/wallet.go`
- Create: `mcp/wallet_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/remove_character.go`
- Modify: `mcp/remove_character_internal_test.go`

**Interfaces:**
- Consumes: `esi.NewClient(ctx, clientID string, characterID int32, userAgent string) (*esi.Client, error)`, `(*esi.Client).WalletBalance(ctx) (float64, error)` (Task 2); `sso.UserAgent(version string) string`, `sso.InvalidateClientCache(characterID int32)` (Task 1 and earlier).
- Produces: `mcp.GetWalletBalanceInput{CharacterID int32}`, `mcp.GetWalletBalanceOutput{CharacterID int32, BalanceISK float64}`. Nothing later in this plan depends on these.

- [ ] **Step 1: Write the failing test**

Create `mcp/wallet_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestGetWalletBalanceHandler_ReturnsBalance(t *testing.T) {
	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		return 42.5, nil
	}

	handler := getWalletBalanceHandler("v1", "client-id")
	_, out, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.BalanceISK != 42.5 {
		t.Errorf("out = %+v, want {7 42.5}", out)
	}
}

func TestGetWalletBalanceHandler_PropagatesError(t *testing.T) {
	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		return 0, errors.New("simulated failure")
	}

	handler := getWalletBalanceHandler("v1", "client-id")
	if _, _, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 7}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetWalletBalanceHandler -v`

Expected: FAIL to compile — `undefined: getWalletBalance`, `undefined: getWalletBalanceHandler`, `undefined: GetWalletBalanceInput`.

- [ ] **Step 3: Implement `mcp/wallet.go`**

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// getWalletBalance is overridable in tests so no real network/auth runs.
var getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.WalletBalance(ctx)
}

// GetWalletBalanceInput is get_wallet_balance's tool input.
type GetWalletBalanceInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// GetWalletBalanceOutput is get_wallet_balance's tool result.
type GetWalletBalanceOutput struct {
	CharacterID int32   `json:"characterId"`
	BalanceISK  float64 `json:"balanceIsk"`
}

func getWalletBalanceHandler(version, clientID string) mcpsdk.ToolHandlerFor[GetWalletBalanceInput, GetWalletBalanceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetWalletBalanceInput) (*mcpsdk.CallToolResult, GetWalletBalanceOutput, error) {
		balance, err := getWalletBalance(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("get wallet balance: %w", err)
		}
		return nil, GetWalletBalanceOutput{CharacterID: in.CharacterID, BalanceISK: balance}, nil
	}
}
```

- [ ] **Step 4: Register the tool in `mcp/server.go`**

In `mcp/server.go`, change:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// and remove_character registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: version}, nil)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "add_character",
		Description: "Log in a new EVE Online character via ESI OAuth2, opening the user's browser to complete authentication.",
	}, addCharacterHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_characters",
		Description: "List EVE Online characters currently logged in to neat.",
	}, listCharactersHandler(store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "remove_character",
		Description: "Remove a logged-in EVE Online character, deleting its stored credentials.",
	}, removeCharacterHandler(store))

	return server
}
```

to:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, and get_wallet_balance registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: version}, nil)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "add_character",
		Description: "Log in a new EVE Online character via ESI OAuth2, opening the user's browser to complete authentication.",
	}, addCharacterHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_characters",
		Description: "List EVE Online characters currently logged in to neat.",
	}, listCharactersHandler(store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "remove_character",
		Description: "Remove a logged-in EVE Online character, deleting its stored credentials.",
	}, removeCharacterHandler(store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_wallet_balance",
		Description: "Get a logged-in EVE Online character's current ISK wallet balance.",
	}, getWalletBalanceHandler(version, clientID))

	return server
}
```

- [ ] **Step 5: Add cache invalidation to `mcp/remove_character.go`**

In `mcp/remove_character.go`, change the import block and the `ErrCharacterNotFound` declaration from:

```go
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

// ErrCharacterNotFound is returned (wrapped) when remove_character is
// called with a character ID that isn't in the local config.
var ErrCharacterNotFound = errors.New("character not found")
```

to:

```go
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// ErrCharacterNotFound is returned (wrapped) when remove_character is
// called with a character ID that isn't in the local config.
var ErrCharacterNotFound = errors.New("character not found")

// invalidateClientCache is overridable in tests.
var invalidateClientCache = sso.InvalidateClientCache
```

And change:

```go
		cfg.Characters = append(cfg.Characters[:idx], cfg.Characters[idx+1:]...)
		if err := store.Save(cfg); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("save config: %w", err)
		}

		return nil, CharacterInfo{CharacterID: removed.ID, CharacterName: removed.Name}, nil
```

to:

```go
		cfg.Characters = append(cfg.Characters[:idx], cfg.Characters[idx+1:]...)
		if err := store.Save(cfg); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("save config: %w", err)
		}
		invalidateClientCache(removed.ID)

		return nil, CharacterInfo{CharacterID: removed.ID, CharacterName: removed.Name}, nil
```

- [ ] **Step 6: Add a cache-invalidation test to `mcp/remove_character_internal_test.go`**

Append this test to the existing file:

```go
func TestRemoveCharacterHandler_InvalidatesClientCache(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 5, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	var invalidatedID int32 = -1
	orig := invalidateClientCache
	t.Cleanup(func() { invalidateClientCache = orig })
	invalidateClientCache = func(characterID int32) { invalidatedID = characterID }

	handler := removeCharacterHandler(store)
	if _, _, err := handler(context.Background(), nil, RemoveCharacterInput{CharacterID: 5}); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if invalidatedID != 5 {
		t.Errorf("invalidatedID = %d, want 5", invalidatedID)
	}
}
```

This file already imports `context`, `testing`, `github.com/zalando/go-keyring`, and
`github.com/rusher2004/new-eden-agent-toolkit/config` (used by its existing tests) — no new
imports needed for this addition.

- [ ] **Step 7: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./mcp/... -race -v`

Expected: `TestGetWalletBalanceHandler_ReturnsBalance`, `TestGetWalletBalanceHandler_PropagatesError`,
and `TestRemoveCharacterHandler_InvalidatesClientCache` all PASS, plus every pre-existing `mcp` test.

- [ ] **Step 8: Commit**

```bash
git add mcp/wallet.go mcp/wallet_internal_test.go mcp/server.go mcp/remove_character.go mcp/remove_character_internal_test.go
git commit -m "feat: add get_wallet_balance MCP tool, invalidate client cache on remove_character"
```

---

### Task 4: Docs sync (`CLAUDE.md`, `README.md`)

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Interfaces:** none — documentation only.

- [ ] **Step 1: Update `CLAUDE.md`'s package table**

Replace:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its three tools: `add_character`, `list_characters`, `remove_character` |

More packages (`esi/`, `sde/`) land in follow-up plans — see `docs/superpowers/plans/`.
```

with:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its four tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance` |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi` — builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client. Holds no state of its own |

More packages (`sde/`) land in follow-up plans — see `docs/superpowers/plans/`.
```

Also update `sso/`'s row to mention the client cache. Replace:

```markdown
| `sso/` | ESI OAuth2/PKCE login flow: PKCE + state generation, a two-socket Unix IPC handoff for the callback (a lock socket that's bound but never `Accept()`-ed, plus a separate delivery socket), orchestration via `github.com/fnt-eve/goesi-openapi`, refresh-token persistence via `creds/`. All outbound requests to FC (JWKS fetch, OAuth2 token exchange) carry `sso.UserAgent(version)`, per ESI's User-Agent best practices |
```

with:

```markdown
| `sso/` | ESI OAuth2/PKCE login flow: PKCE + state generation, a two-socket Unix IPC handoff for the callback (a lock socket that's bound but never `Accept()`-ed, plus a separate delivery socket), orchestration via `github.com/fnt-eve/goesi-openapi`, refresh-token persistence via `creds/`. All outbound requests to FC (JWKS fetch, OAuth2 token exchange) carry `sso.UserAgent(version)`, per ESI's User-Agent best practices. Also caches a per-character auto-refreshing HTTP client (`AuthenticatedClient`/`InvalidateClientCache`) for `esi/`'s use — refreshing relies entirely on `golang.org/x/oauth2`'s own token source, not hand-rolled expiry tracking |
```

- [ ] **Step 2: Update `README.md`**

Replace the `## Status` section:

```markdown
Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), and the MCP server with its character-management tools (`mcp/`,
`neat mcp`) are built. The SDE cache and plugin/extension packaging don't exist yet. See the plans
directory for what's next.
```

with:

```markdown
Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), the MCP server with its character-management tools (`mcp/`, `neat mcp`),
and a first real ESI-backed domain tool (`esi/`, `get_wallet_balance`) are built. The SDE cache
and plugin/extension packaging don't exist yet. See the plans directory for what's next.
```

Update the package table the same way as `CLAUDE.md`'s Step 1 (add the `esi/` row, update
`sso/`'s and `mcp/`'s rows, drop the "land in follow-up plans" line's now-stale `esi/` mention).

- [ ] **Step 3: Sanity-check the docs against the code**

Run: `grep -n "esi/\|get_wallet_balance" CLAUDE.md README.md`

Expected: both files mention the `esi/` package and the `get_wallet_balance` tool.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: document the esi package and get_wallet_balance tool"
```
