# ESI Loyalty Implementation Plan (twenty-second batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Loyalty" tag (2 endpoints: a character's own loyalty point
balances, and a corporation's public loyalty store offers) as 2 new MCP tools, folding into the
same shared PR #6 branch as every prior batch.

**Architecture:** Same two-layer split as every prior batch. `esi/loyalty.go` wraps the 2
generated `LoyaltyAPIService` methods. `CharacterLoyaltyPoints` is private/"my own character"
(`int64(c.characterID)`, per `esi/api/openapi.yaml`'s security requirement
`esi-characters.read_loyalty.v1`); `LoyaltyStoreOffers` is fully public (no security requirement in
the spec at all — same shape as `esi/insurance.go`), taking a caller-supplied `corporationID`.
`mcp/loyalty.go` exposes 2 tools on top of that: one private (reusing
`characterExists`/`ErrCharacterNotFound`), one public (no character gate at all, mirroring
`get_killmail`'s shape in `mcp/killmails.go`).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: 135 MCP tools, branch `worktree-esi-access-list-activities`,
  PR #6 (open). This plan brings the count to 137.
- Verified against the installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module AND its
  `esi/api/openapi.yaml` security requirements: `GetCharactersCharacterIdLoyaltyPoints` requires
  OAuth2 scope `esi-characters.read_loyalty.v1` (private); `GetLoyaltyStoresCorporationIdOffers`
  has no security requirement at all (public). No pagination, no write, no oneOf types on either
  endpoint.
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- Reuse `characterExists(store *config.Store, characterID int32) (bool, error)` and
  `ErrCharacterNotFound` from `mcp/character.go` — do not redeclare. Only the private tool
  (`list_character_loyalty_points`) uses this gate; the public tool
  (`list_loyalty_store_offers`) does not, matching `get_killmail`'s precedent in `mcp/killmails.go`.
- **Standing convention (added after the Killmails batch):** never model a nested/tree-shaped ESI
  resource with a self-referential Go struct. `LoyaltyStoreOffer`/`LoyaltyRequiredItem` in this
  batch are both flat, non-recursive — already sandbox-verified with a live `mcpsdk.AddTool`
  registration test (zero schema-generator errors); the plan's Task 2 must still run the real
  `go test ./... -race` before commit, not just build/vet.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-loyalty-design.md` for the verification detail.

---

### Task 1: `esi/loyalty.go` — ESI facade layer

**Files:**
- Create: `esi/loyalty.go`
- Test: `esi/loyalty_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`);
  `c.api.LoyaltyAPI` and `c.characterID` (`esi/client.go`'s `Client`, unchanged).
- Produces (for Task 2):
  `func (c *Client) CharacterLoyaltyPoints(ctx context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error)`;
  `func (c *Client) LoyaltyStoreOffers(ctx context.Context, corporationID int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error)`.

- [ ] **Step 1: Write the failing tests**

Create `esi/loyalty_test.go`:

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

func TestCharacterLoyaltyPoints_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/loyalty/points" {
			t.Errorf("request path = %q, want /characters/998877/loyalty/points", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"corporation_id":1000167,"loyalty_points":15000}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	points, err := client.CharacterLoyaltyPoints(context.Background())
	if err != nil {
		t.Fatalf("CharacterLoyaltyPoints returned error: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("points = %+v, want one entry", points)
	}
	if points[0].CorporationId != 1000167 || points[0].LoyaltyPoints != 15000 {
		t.Errorf("points[0] = %+v, want CorporationId=1000167 LoyaltyPoints=15000", points[0])
	}
}

func TestCharacterLoyaltyPoints_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterLoyaltyPoints(context.Background()); err == nil {
		t.Fatal("CharacterLoyaltyPoints returned nil error, want an error for HTTP 404")
	}
}

func TestLoyaltyStoreOffers_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loyalty/stores/1000167/offers" {
			t.Errorf("request path = %q, want /loyalty/stores/1000167/offers", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"ak_cost":50,"isk_cost":10000,"lp_cost":100,"offer_id":1,"quantity":1,"required_items":[{"quantity":2,"type_id":34}],"type_id":588}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	offers, err := client.LoyaltyStoreOffers(context.Background(), 1000167)
	if err != nil {
		t.Fatalf("LoyaltyStoreOffers returned error: %v", err)
	}
	if len(offers) != 1 {
		t.Fatalf("offers = %+v, want one entry", offers)
	}
	o := offers[0]
	if o.AkCost == nil || *o.AkCost != 50 || o.IskCost != 10000 || o.LpCost != 100 || o.OfferId != 1 || o.Quantity != 1 || o.TypeId != 588 {
		t.Errorf("offers[0] = %+v, want AkCost=50 IskCost=10000 LpCost=100 OfferId=1 Quantity=1 TypeId=588", o)
	}
	if len(o.RequiredItems) != 1 || o.RequiredItems[0].Quantity != 2 || o.RequiredItems[0].TypeId != 34 {
		t.Errorf("offers[0].RequiredItems = %+v, want one entry {Quantity:2 TypeId:34}", o.RequiredItems)
	}
}

func TestLoyaltyStoreOffers_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.LoyaltyStoreOffers(context.Background(), 1000167); err == nil {
		t.Fatal("LoyaltyStoreOffers returned nil error, want an error for HTTP 404")
	}
}
```

Note: verify the exact resolved request paths empirically by running the tests and reading the
actual failure output if a path assertion doesn't match — the generated client's trailing-slash
behavior has varied slightly across resource groups in this project's history (Location's batch
had none; earlier batches sometimes did). Do not weaken or remove the path assertions — fix the
expected string only if it doesn't match.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterLoyaltyPoints|TestLoyaltyStoreOffers' -v`
Expected: FAIL — `esi.Client` has no method `CharacterLoyaltyPoints`/`LoyaltyStoreOffers`.

- [ ] **Step 3: Write the implementation**

Create `esi/loyalty.go`:

```go
// Package esi's loyalty.go covers both of "Loyalty"'s endpoints -- a character's own loyalty
// point balances (private, "my own character") and a corporation's loyalty store offers (public,
// no character/auth needed, caller-supplied corporation ID).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterLoyaltyPoints returns the client's own character's loyalty point balance with every
// corporation they've worked for.
func (c *Client) CharacterLoyaltyPoints(ctx context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error) {
	result, resp, err := c.api.LoyaltyAPI.GetCharactersCharacterIdLoyaltyPoints(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character loyalty points: %w", describeError(err, resp))
	}
	return result, nil
}

// LoyaltyStoreOffers returns a corporation's loyalty store offers. Public -- no authenticated
// character needed, so this is safe to call from a public Client (see esi.NewPublicClient).
func (c *Client) LoyaltyStoreOffers(ctx context.Context, corporationID int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error) {
	result, resp, err := c.api.LoyaltyAPI.GetLoyaltyStoresCorporationIdOffers(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get loyalty store offers: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestCharacterLoyaltyPoints|TestLoyaltyStoreOffers' -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./...`
Expected: all clean, no output from `gofmt -l`.

```bash
git add esi/loyalty.go esi/loyalty_test.go
git commit -m "feat: add esi/loyalty.go facade (character loyalty points, loyalty store offers)"
```

---

### Task 2: `mcp/loyalty.go` — MCP tool layer + registration + docs

**Files:**
- Create: `mcp/loyalty.go`
- Test: `mcp/loyalty_internal_test.go`
- Modify: `mcp/server.go` (register 2 tools after the `get_character_ship` block; update package doc comment)
- Modify: `mcp/server_test.go` (expected-tools list: 135 → 137)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Loyalty row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (tool count "one hundred thirty-five" → "one hundred thirty-seven", append 2 tool names)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: `esi.NewClient(ctx, clientID, characterID, userAgent) (*esi.Client, error)`;
  `esi.NewPublicClient(userAgent) *esi.Client`;
  `(*esi.Client).CharacterLoyaltyPoints/.LoyaltyStoreOffers` from Task 1;
  `characterExists`/`ErrCharacterNotFound` (`mcp/character.go`);
  `sso.UserAgent(version string) string` (`mcp/sso`); `config.Store` (`mcp/config`).
- Produces: 2 tool names — `list_character_loyalty_points`, `list_loyalty_store_offers` —
  registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `mcp/loyalty_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterLoyaltyPointsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterLoyaltyPoints
	t.Cleanup(func() { listCharacterLoyaltyPoints = orig })
	listCharacterLoyaltyPoints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]LoyaltyPointsEntry, error) {
		return []LoyaltyPointsEntry{{CorporationID: 1000167, LoyaltyPoints: 15000}}, nil
	}

	handler := listCharacterLoyaltyPointsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterLoyaltyPointsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Points) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one point entry", out)
	}
	if out.Points[0].CorporationID != 1000167 || out.Points[0].LoyaltyPoints != 15000 {
		t.Errorf("out.Points[0] = %+v, want CorporationID=1000167 LoyaltyPoints=15000", out.Points[0])
	}
}

func TestListCharacterLoyaltyPointsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterLoyaltyPoints
	t.Cleanup(func() { listCharacterLoyaltyPoints = orig })
	listCharacterLoyaltyPoints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]LoyaltyPointsEntry, error) {
		t.Fatal("listCharacterLoyaltyPoints should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterLoyaltyPointsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterLoyaltyPointsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListLoyaltyStoreOffersHandler_MapsFields(t *testing.T) {
	orig := listLoyaltyStoreOffers
	t.Cleanup(func() { listLoyaltyStoreOffers = orig })
	akCost := int64(50)
	listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
		return []LoyaltyStoreOffer{{
			OfferID: 1, TypeID: 588, Quantity: 1, IskCost: 10000, LpCost: 100, AkCost: &akCost,
			RequiredItems: []LoyaltyRequiredItem{{TypeID: 34, Quantity: 2}},
		}}, nil
	}

	handler := listLoyaltyStoreOffersHandler("v1")
	_, out, err := handler(context.Background(), nil, ListLoyaltyStoreOffersInput{CorporationID: 1000167})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 1000167 || len(out.Offers) != 1 {
		t.Fatalf("out = %+v, want CorporationID=1000167 with one offer", out)
	}
	o := out.Offers[0]
	if o.OfferID != 1 || o.TypeID != 588 || o.Quantity != 1 || o.IskCost != 10000 || o.LpCost != 100 {
		t.Errorf("offer = %+v, want OfferID=1 TypeID=588 Quantity=1 IskCost=10000 LpCost=100", o)
	}
	if o.AkCost == nil || *o.AkCost != 50 {
		t.Errorf("AkCost = %v, want pointer to 50", o.AkCost)
	}
	if len(o.RequiredItems) != 1 || o.RequiredItems[0].TypeID != 34 || o.RequiredItems[0].Quantity != 2 {
		t.Errorf("RequiredItems = %+v, want one entry {TypeID:34 Quantity:2}", o.RequiredItems)
	}
}

func TestListLoyaltyStoreOffersHandler_PropagatesError(t *testing.T) {
	orig := listLoyaltyStoreOffers
	t.Cleanup(func() { listLoyaltyStoreOffers = orig })
	listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
		return nil, errBoom
	}

	handler := listLoyaltyStoreOffersHandler("v1")
	_, _, err := handler(context.Background(), nil, ListLoyaltyStoreOffersInput{CorporationID: 1000167})
	if err == nil {
		t.Fatal("handler returned nil error, want the propagated error")
	}
}

var errBoom = errors.New("boom")
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListCharacterLoyaltyPointsHandler|TestListLoyaltyStoreOffersHandler' -v`
Expected: FAIL — undefined symbols (`listCharacterLoyaltyPoints`, `LoyaltyStoreOffer`, etc. don't exist yet).

- [ ] **Step 3: Write the implementation**

Create `mcp/loyalty.go`:

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_character_loyalty_points (private) ---

type LoyaltyPointsEntry struct {
	CorporationID int64 `json:"corporationId"`
	LoyaltyPoints int64 `json:"loyaltyPoints"`
}

var listCharacterLoyaltyPoints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]LoyaltyPointsEntry, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	entries, err := client.CharacterLoyaltyPoints(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LoyaltyPointsEntry, len(entries))
	for i, e := range entries {
		out[i] = LoyaltyPointsEntry{CorporationID: e.CorporationId, LoyaltyPoints: e.LoyaltyPoints}
	}
	return out, nil
}

type ListCharacterLoyaltyPointsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterLoyaltyPointsOutput struct {
	CharacterID int32                `json:"characterId"`
	Points      []LoyaltyPointsEntry `json:"points"`
}

func listCharacterLoyaltyPointsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterLoyaltyPointsInput, ListCharacterLoyaltyPointsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterLoyaltyPointsInput) (*mcpsdk.CallToolResult, ListCharacterLoyaltyPointsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterLoyaltyPointsOutput{}, err
		}
		if !found {
			return nil, ListCharacterLoyaltyPointsOutput{}, fmt.Errorf("list character loyalty points for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		points, err := listCharacterLoyaltyPoints(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterLoyaltyPointsOutput{}, fmt.Errorf("list character loyalty points: %w", err)
		}
		return nil, ListCharacterLoyaltyPointsOutput{CharacterID: in.CharacterID, Points: points}, nil
	}
}

// --- list_loyalty_store_offers (public) ---

type LoyaltyRequiredItem struct {
	TypeID   int64 `json:"typeId"`
	Quantity int64 `json:"quantity"`
}

type LoyaltyStoreOffer struct {
	OfferID       int64                 `json:"offerId"`
	TypeID        int64                 `json:"typeId"`
	Quantity      int64                 `json:"quantity"`
	IskCost       int64                 `json:"iskCost"`
	LpCost        int64                 `json:"lpCost"`
	AkCost        *int64                `json:"akCost,omitempty"`
	RequiredItems []LoyaltyRequiredItem `json:"requiredItems"`
}

var listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
	client := esi.NewPublicClient(userAgent)
	offers, err := client.LoyaltyStoreOffers(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]LoyaltyStoreOffer, len(offers))
	for i, o := range offers {
		items := make([]LoyaltyRequiredItem, len(o.RequiredItems))
		for j, it := range o.RequiredItems {
			items[j] = LoyaltyRequiredItem{TypeID: it.TypeId, Quantity: it.Quantity}
		}
		out[i] = LoyaltyStoreOffer{
			OfferID: o.OfferId, TypeID: o.TypeId, Quantity: o.Quantity,
			IskCost: o.IskCost, LpCost: o.LpCost, AkCost: o.AkCost, RequiredItems: items,
		}
	}
	return out, nil
}

type ListLoyaltyStoreOffersInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID whose loyalty store to look up"`
}
type ListLoyaltyStoreOffersOutput struct {
	CorporationID int64               `json:"corporationId"`
	Offers        []LoyaltyStoreOffer `json:"offers"`
}

func listLoyaltyStoreOffersHandler(version string) mcpsdk.ToolHandlerFor[ListLoyaltyStoreOffersInput, ListLoyaltyStoreOffersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListLoyaltyStoreOffersInput) (*mcpsdk.CallToolResult, ListLoyaltyStoreOffersOutput, error) {
		offers, err := listLoyaltyStoreOffers(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListLoyaltyStoreOffersOutput{}, fmt.Errorf("list loyalty store offers: %w", err)
		}
		return nil, ListLoyaltyStoreOffersOutput{CorporationID: in.CorporationID, Offers: offers}, nil
	}
}
```

Modify `mcp/server.go`: append after the existing `get_character_ship` block (immediately before
`return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_loyalty_points",
		Description: "List a logged-in EVE Online character's loyalty point balances with every corporation they've worked for.",
	}, listCharacterLoyaltyPointsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_loyalty_store_offers",
		Description: "List a corporation's loyalty store offers (public data, no character required).",
	}, listLoyaltyStoreOffersHandler(version))
```

Update `mcp/server.go`'s package doc comment: change the trailing
`"... get_killmail, get_character_location, get_character_online, and get_character_ship registered."`
to end with
`"... get_killmail, get_character_location, get_character_online, get_character_ship, list_character_loyalty_points, and list_loyalty_store_offers registered."`

Modify `mcp/server_test.go`: append `"list_character_loyalty_points"`, `"list_loyalty_store_offers"`
to the expected-tools slice (after `"get_character_ship"`), and confirm the test's expected count
is 137 (whether it's a literal count or `len(expected)` computed from the slice — check the
existing test to see which, and match that style).

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Loyalty row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: change "one hundred thirty-five" to "one hundred thirty-seven", and append
`list_character_loyalty_points`, `list_loyalty_store_offers` to the `mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestListCharacterLoyaltyPointsHandler|TestListLoyaltyStoreOffersHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (4 new handler tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, 137 tools registered without panic.

```bash
git add mcp/loyalty.go mcp/loyalty_internal_test.go mcp/server.go mcp/server_test.go \
  CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add loyalty MCP tools (2 tools: character points, store offers)"
```
