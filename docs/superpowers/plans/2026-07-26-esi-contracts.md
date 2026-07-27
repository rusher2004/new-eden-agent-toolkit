# ESI Contracts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cover the Contracts ESI resource group (9 endpoints: 3 public + 3 character-scoped + 3 corp-scoped; the sweep's first pagination pattern).

**Architecture:** `esi/contracts.go` is a thin, `describeError`-wrapped facade. 6 of its 9 methods are paginated: each takes a `page int32` param (0 = let ESI default to page 1, no `.Page()` call at all) and returns `(items, totalPages int32, error)` where `totalPages` is parsed from the `X-Pages` response header. `mcp/contracts.go` adds 9 tools reusing the existing `characterExists` helper for the 6 private ones; the 3 public ones use `esi.NewPublicClient` with no existence check.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context. Never discard an error. Every `esi/` method routes errors through `describeError`.
- External test packages by default; standard library only in tests.
- **Pagination convention (new)**: a paginated `esi/` method's `page int32` parameter of `0` means "omit `.Page()` entirely" (verified live: this sends no query param, letting ESI default to page 1). Every paginated method returns `(items []T, totalPages int32, error)`, with `totalPages` defaulting to `1` if the `X-Pages` header is missing or unparseable.
- **Verified parameter-order quirk**: the generated `GetCorporationsCorporationIdContractsContractIdBids`/`...Items` functions take `(ctx, contractId int64, corporationId int64)` — contract ID first — the reverse of the URL path and of every other corp-scoped method in this sweep. `esi/contracts.go`'s own `CorporationContractBids`/`CorporationContractItems` methods keep the conventional `(corporationID, contractID)` parameter order in their own signature and internally swap the arguments when calling the generated function — do not "fix" this to match the generated function's order, and do not accidentally swap neat's own convention.
- `mcp/contracts.go` reuses the existing `characterExists(store, characterID) (bool, error)` helper from `mcp/character.go` — do not redeclare it.
- This branch is the shared PR branch (`worktree-esi-access-list-activities`), current baseline 48 tools.
- Design spec: `docs/superpowers/specs/2026-07-26-esi-contracts-design.md`.
- All code verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and `github.com/modelcontextprotocol/go-sdk` v1.6.1 modules, including a live `httptest.Server` round-trip confirming the `page`/`X-Pages` pagination shape and the reversed corp-scoped parameter order.

---

### Task 1: `esi/contracts.go` — Contracts facade (9 methods, pagination)

**Files:**
- Create: `esi/contracts.go`
- Create: `esi/contracts_test.go`

**Interfaces:**
- Consumes: `esi.Client`, `esi.NewPublicClient`, `describeError` (existing); `esiapi.APIClient.ContractsAPI.GetCharactersCharacterIdContracts(ctx, characterId int64).Page(p) ...Execute() ([]esiapi.CharactersCharacterIdContractsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdContractsContractIdBids(ctx, characterId, contractId int64) ...Execute() ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdContractsContractIdItems(ctx, characterId, contractId int64) ...Execute() ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, *http.Response, error)`, `.GetCorporationsCorporationIdContracts(ctx, corporationId int64).Page(p) ...Execute() ([]esiapi.CorporationsCorporationIdContractsGetInner, *http.Response, error)`, `.GetCorporationsCorporationIdContractsContractIdBids(ctx, contractId, corporationId int64).Page(p) ...Execute() ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, *http.Response, error)` (note argument order: contractId then corporationId), `.GetCorporationsCorporationIdContractsContractIdItems(ctx, contractId, corporationId int64) ...Execute() ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, *http.Response, error)` (same reversed order), `.GetContractsPublicRegionId(ctx, regionId int64).Page(p) ...Execute() ([]esiapi.ContractsPublicRegionIdGetInner, *http.Response, error)`, `.GetContractsPublicBidsContractId(ctx, contractId int64).Page(p) ...Execute() ([]esiapi.ContractsPublicBidsContractIdGetInner, *http.Response, error)`, `.GetContractsPublicItemsContractId(ctx, contractId int64).Page(p) ...Execute() ([]esiapi.ContractsPublicItemsContractIdGetInner, *http.Response, error)` — all verified against the real installed module.
- Produces: `(*esi.Client).CharacterContracts`, `.CharacterContractBids`, `.CharacterContractItems`, `.CorporationContracts`, `.CorporationContractBids`, `.CorporationContractItems`, `.PublicContracts`, `.PublicContractBids`, `.PublicContractItems` — all consumed by Task 2.

- [ ] **Step 1: Write the failing tests**

Create `esi/contracts_test.go`:

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

func TestCharacterContracts_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{"acceptor_id":1,"assignee_id":2,"availability":"public","contract_id":1,"date_expired":"2026-08-01T00:00:00Z","date_issued":"2026-07-01T00:00:00Z","for_corporation":false,"issuer_corporation_id":98000001,"issuer_id":998877,"status":"outstanding","type":"item_exchange"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contracts, totalPages, err := client.CharacterContracts(context.Background(), 0)
	if err != nil {
		t.Fatalf("CharacterContracts returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(contracts) != 1 || contracts[0].Status != "outstanding" {
		t.Errorf("contracts = %+v, want one entry with Status=\"outstanding\"", contracts)
	}

	_, _, err = client.CharacterContracts(context.Background(), 2)
	if err != nil {
		t.Fatalf("CharacterContracts(page=2) returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want \"page=2\"", gotQuery)
	}
}

func TestCharacterContracts_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterContracts(context.Background(), 0); err == nil {
		t.Fatal("CharacterContracts returned nil error, want an error for HTTP 403")
	}
}

func TestCharacterContractBids_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"amount":1000000,"bid_id":1,"bidder_id":998878,"date_bid":"2026-07-15T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bids, err := client.CharacterContractBids(context.Background(), 1)
	if err != nil {
		t.Fatalf("CharacterContractBids returned error: %v", err)
	}
	if len(bids) != 1 || bids[0].BidderId != 998878 {
		t.Errorf("bids = %+v, want one entry with BidderId=998878", bids)
	}
}

func TestCharacterContractItems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"is_included":true,"is_singleton":false,"quantity":1,"record_id":1,"type_id":34}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	items, err := client.CharacterContractItems(context.Background(), 1)
	if err != nil {
		t.Fatalf("CharacterContractItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].TypeId != 34 {
		t.Errorf("items = %+v, want one entry with TypeId=34", items)
	}
}

func TestCorporationContracts_ParsesResponseAndPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"acceptor_id":1,"assignee_id":2,"availability":"public","contract_id":2,"date_expired":"2026-08-01T00:00:00Z","date_issued":"2026-07-01T00:00:00Z","for_corporation":true,"issuer_corporation_id":98000001,"issuer_id":998877,"status":"outstanding","type":"courier"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contracts, totalPages, err := client.CorporationContracts(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationContracts returned error: %v", err)
	}
	if totalPages != 2 || len(contracts) != 1 || contracts[0].Type != "courier" {
		t.Errorf("contracts = %+v, totalPages = %d, want totalPages=2 with one courier contract", contracts, totalPages)
	}
}

func TestCorporationContractBids_UsesReversedParameterOrder(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"amount":500000,"bid_id":5,"date_bid":"2026-07-15T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bids, totalPages, err := client.CorporationContractBids(context.Background(), 98000001, 3, 0)
	if err != nil {
		t.Fatalf("CorporationContractBids returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/contracts/3/bids" {
		t.Errorf("path = %q, want \"/corporations/98000001/contracts/3/bids\" (corporationID=98000001, contractID=3, not swapped)", gotPath)
	}
	if totalPages != 1 || len(bids) != 1 || bids[0].BidId != 5 {
		t.Errorf("bids = %+v, totalPages = %d, want one bid with BidId=5 and totalPages=1", bids, totalPages)
	}
}

func TestCorporationContractItems_UsesReversedParameterOrder(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"is_included":true,"is_singleton":true,"quantity":1,"record_id":1,"type_id":648}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	items, err := client.CorporationContractItems(context.Background(), 98000001, 3)
	if err != nil {
		t.Fatalf("CorporationContractItems returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/contracts/3/items" {
		t.Errorf("path = %q, want \"/corporations/98000001/contracts/3/items\"", gotPath)
	}
	if len(items) != 1 || items[0].TypeId != 648 {
		t.Errorf("items = %+v, want one entry with TypeId=648", items)
	}
}

func TestPublicContracts_ParsesResponseWithNoStatusField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "5")
		w.Write([]byte(`[{"contract_id":10,"date_expired":"2026-08-01T00:00:00Z","date_issued":"2026-07-01T00:00:00Z","issuer_corporation_id":98000001,"issuer_id":998877,"type":"item_exchange"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contracts, totalPages, err := client.PublicContracts(context.Background(), 10000002, 0)
	if err != nil {
		t.Fatalf("PublicContracts returned error: %v", err)
	}
	if totalPages != 5 || len(contracts) != 1 || contracts[0].ContractId != 10 {
		t.Errorf("contracts = %+v, totalPages = %d, want totalPages=5 with one entry ContractId=10", contracts, totalPages)
	}
}

func TestPublicContractBids_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"amount":250000,"bid_id":7,"date_bid":"2026-07-15T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bids, totalPages, err := client.PublicContractBids(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("PublicContractBids returned error: %v", err)
	}
	if totalPages != 1 || len(bids) != 1 || bids[0].BidId != 7 {
		t.Errorf("bids = %+v, totalPages = %d, want one bid with BidId=7 and totalPages=1", bids, totalPages)
	}
}

func TestPublicContractItems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"is_included":true,"item_id":123,"quantity":1,"record_id":1,"type_id":34}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	items, totalPages, err := client.PublicContractItems(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("PublicContractItems returned error: %v", err)
	}
	if totalPages != 1 || len(items) != 1 || items[0].TypeId != 34 {
		t.Errorf("items = %+v, totalPages = %d, want one entry with TypeId=34 and totalPages=1", items, totalPages)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterContracts|TestCharacterContractBids|TestCharacterContractItems|TestCorporationContracts|TestCorporationContractBids|TestCorporationContractItems|TestPublicContracts|TestPublicContractBids|TestPublicContractItems' -v`

Expected: FAIL to compile.

- [ ] **Step 3: Implement `esi/contracts.go`**

```go
package esi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// totalPagesFromHeader parses the X-Pages response header, defaulting to 1 if missing or
// unparseable -- ESI's own convention for "this response isn't paginated" or a malformed header.
func totalPagesFromHeader(resp *http.Response) int32 {
	if resp == nil {
		return 1
	}
	n, err := strconv.Atoi(resp.Header.Get("X-Pages"))
	if err != nil || n < 1 {
		return 1
	}
	return int32(n)
}

// CharacterContracts returns the client's character's contracts. page=0 omits the page query
// parameter entirely, letting ESI default to page 1.
func (c *Client) CharacterContracts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error) {
	req := c.api.ContractsAPI.GetCharactersCharacterIdContracts(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character contracts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CharacterContractBids returns the bids on one of the client's character's contracts. Not paginated.
func (c *Client) CharacterContractBids(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error) {
	result, resp, err := c.api.ContractsAPI.GetCharactersCharacterIdContractsContractIdBids(ctx, int64(c.characterID), contractID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character contract bids: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterContractItems returns the items on one of the client's character's contracts. Not paginated.
func (c *Client) CharacterContractItems(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
	result, resp, err := c.api.ContractsAPI.GetCharactersCharacterIdContractsContractIdItems(ctx, int64(c.characterID), contractID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character contract items: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationContracts returns corporationID's contracts. The authenticated character must hold
// sufficient corp roles — ESI itself returns 403 otherwise. page=0 omits the page query parameter.
func (c *Client) CorporationContracts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error) {
	req := c.api.ContractsAPI.GetCorporationsCorporationIdContracts(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation contracts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationContractBids returns the bids on one of corporationID's contracts. Note: the
// generated call takes (contractID, corporationID) — reversed from this method's own
// (corporationID, contractID) signature and from the URL path — verified from source.
func (c *Client) CorporationContractBids(ctx context.Context, corporationID int64, contractID int64, page int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error) {
	req := c.api.ContractsAPI.GetCorporationsCorporationIdContractsContractIdBids(ctx, contractID, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation contract bids: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationContractItems returns the items on one of corporationID's contracts. Same reversed
// generated-call parameter order as CorporationContractBids. Not paginated.
func (c *Client) CorporationContractItems(ctx context.Context, corporationID int64, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
	result, resp, err := c.api.ContractsAPI.GetCorporationsCorporationIdContractsContractIdItems(ctx, contractID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation contract items: %w", describeError(err, resp))
	}
	return result, nil
}

// PublicContracts returns public contracts in regionID. page=0 omits the page query parameter.
func (c *Client) PublicContracts(ctx context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error) {
	req := c.api.ContractsAPI.GetContractsPublicRegionId(ctx, regionID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get public contracts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// PublicContractBids returns the bids on a public contract. page=0 omits the page query parameter.
func (c *Client) PublicContractBids(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error) {
	req := c.api.ContractsAPI.GetContractsPublicBidsContractId(ctx, contractID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get public contract bids: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// PublicContractItems returns the items on a public contract. page=0 omits the page query parameter.
func (c *Client) PublicContractItems(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error) {
	req := c.api.ContractsAPI.GetContractsPublicItemsContractId(ctx, contractID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get public contract items: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all 10 new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/contracts.go esi/contracts_test.go
git commit -m "feat: add esi/contracts.go facade (first paginated endpoints)"
```

---

### Task 2: `mcp/contracts.go` tools + registration + docs sync

**Files:**
- Create: `mcp/contracts.go`
- Create: `mcp/contracts_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/server_test.go`
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Modify: `docs/reference/esi-coverage-roadmap.md`

**Interfaces:**
- Consumes: `esi.NewClient`, `esi.NewPublicClient`, all 9 `(*esi.Client).*Contract*` methods (Task 1); `sso.UserAgent`, `config.Store`, `ErrCharacterNotFound`, and the **existing** `characterExists` helper from `mcp/character.go` (do not redeclare).
- Produces: 9 new tool Input/Output types plus shared `mcp.Contract`, `mcp.ContractBid`, `mcp.ContractItem` types. Nothing later depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/contracts_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContracts
	t.Cleanup(func() { listCharacterContracts = orig })
	listCharacterContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Contract, int32, error) {
		return []Contract{{ContractID: 1, Status: "outstanding"}}, 3, nil
	}

	handler := listCharacterContractsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TotalPages != 3 || out.Page != 1 || len(out.Contracts) != 1 {
		t.Errorf("out = %+v, want TotalPages=3 Page=1 with one contract", out)
	}
}

func TestListCharacterContractsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContracts
	t.Cleanup(func() { listCharacterContracts = orig })
	listCharacterContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Contract, int32, error) {
		t.Fatal("listCharacterContracts should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCharacterContractsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContractBidsHandler_ReturnsBids(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContractBids
	t.Cleanup(func() { listCharacterContractBids = orig })
	listCharacterContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractBid, error) {
		return []ContractBid{{BidID: 1, Amount: 1000000}}, nil
	}

	handler := listCharacterContractBidsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].Amount != 1000000 {
		t.Errorf("out = %+v, want one bid with Amount=1000000", out)
	}
}

func TestListCharacterContractBidsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContractBids
	t.Cleanup(func() { listCharacterContractBids = orig })
	listCharacterContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractBid, error) {
		t.Fatal("listCharacterContractBids should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterContractBidsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 999, ContractID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContractItemsHandler_ReturnsItems(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContractItems
	t.Cleanup(func() { listCharacterContractItems = orig })
	listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
		return []ContractItem{{RecordID: 1, TypeID: 34}}, nil
	}

	handler := listCharacterContractItemsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 34 {
		t.Errorf("out = %+v, want one item with TypeID=34", out)
	}
}

func TestListCharacterContractItemsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContractItems
	t.Cleanup(func() { listCharacterContractItems = orig })
	listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
		t.Fatal("listCharacterContractItems should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterContractItemsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 999, ContractID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContracts
	t.Cleanup(func() { listCorporationContracts = orig })
	listCorporationContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Contract, int32, error) {
		return []Contract{{ContractID: 2}}, 2, nil
	}

	handler := listCorporationContractsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.TotalPages != 2 || len(out.Contracts) != 1 {
		t.Errorf("out = %+v, want CorporationID=98000001 TotalPages=2 with one contract", out)
	}
}

func TestListCorporationContractsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContracts
	t.Cleanup(func() { listCorporationContracts = orig })
	listCorporationContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Contract, int32, error) {
		t.Fatal("listCorporationContracts should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationContractsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractBidsHandler_ReturnsBidsAndPagination(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContractBids
	t.Cleanup(func() { listCorporationContractBids = orig })
	listCorporationContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
		return []ContractBid{{BidID: 5}}, 1, nil
	}

	handler := listCorporationContractBidsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].BidID != 5 || out.TotalPages != 1 {
		t.Errorf("out = %+v, want one bid with BidID=5 and TotalPages=1", out)
	}
}

func TestListCorporationContractBidsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContractBids
	t.Cleanup(func() { listCorporationContractBids = orig })
	listCorporationContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
		t.Fatal("listCorporationContractBids should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationContractBidsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 999, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractItemsHandler_ReturnsItems(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContractItems
	t.Cleanup(func() { listCorporationContractItems = orig })
	listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
		return []ContractItem{{RecordID: 1, TypeID: 648}}, nil
	}

	handler := listCorporationContractItemsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 648 {
		t.Errorf("out = %+v, want one item with TypeID=648", out)
	}
}

func TestListCorporationContractItemsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContractItems
	t.Cleanup(func() { listCorporationContractItems = orig })
	listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
		t.Fatal("listCorporationContractItems should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationContractItemsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 999, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListPublicContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	orig := listPublicContracts
	t.Cleanup(func() { listPublicContracts = orig })
	listPublicContracts = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]Contract, int32, error) {
		return []Contract{{ContractID: 10}}, 5, nil
	}

	handler := listPublicContractsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListPublicContractsInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.TotalPages != 5 || len(out.Contracts) != 1 {
		t.Errorf("out = %+v, want RegionID=10000002 TotalPages=5 with one contract", out)
	}
}

func TestListPublicContractBidsHandler_ReturnsBids(t *testing.T) {
	orig := listPublicContractBids
	t.Cleanup(func() { listPublicContractBids = orig })
	listPublicContractBids = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractBid, int32, error) {
		return []ContractBid{{BidID: 7}}, 1, nil
	}

	handler := listPublicContractBidsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListPublicContractBidsInput{ContractID: 10})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].BidID != 7 {
		t.Errorf("out = %+v, want one bid with BidID=7", out)
	}
}

func TestListPublicContractItemsHandler_ReturnsItems(t *testing.T) {
	orig := listPublicContractItems
	t.Cleanup(func() { listPublicContractItems = orig })
	listPublicContractItems = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractItem, int32, error) {
		return []ContractItem{{RecordID: 1, TypeID: 34}}, 1, nil
	}

	handler := listPublicContractItemsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListPublicContractItemsInput{ContractID: 10})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 34 {
		t.Errorf("out = %+v, want one item with TypeID=34", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListCharacterContractsHandler|TestListCharacterContractBidsHandler|TestListCharacterContractItemsHandler|TestListCorporationContractsHandler|TestListCorporationContractBidsHandler|TestListCorporationContractItemsHandler|TestListPublicContractsHandler|TestListPublicContractBidsHandler|TestListPublicContractItemsHandler' -v`

Expected: FAIL to compile.

- [ ] **Step 3: Implement `mcp/contracts.go`**

**Do not declare `characterExists` or `ErrCharacterNotFound` in this file — both already exist package-wide.**

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// Contract is one contract entry, shared by character/corporation/public listings. Status is
// empty ("") when populated from a public-region listing, which has no status field at all
// (public listings are implicitly outstanding).
type Contract struct {
	ContractID          int64      `json:"contractId"`
	AcceptorID          int64      `json:"acceptorId,omitempty"`
	AssigneeID          int64      `json:"assigneeId,omitempty"`
	IssuerID            int64      `json:"issuerId"`
	IssuerCorporationID int64      `json:"issuerCorporationId"`
	Availability        string     `json:"availability,omitempty"`
	Status              string     `json:"status,omitempty"`
	Type                string     `json:"type"`
	ForCorporation      bool       `json:"forCorporation"`
	DateIssued          time.Time  `json:"dateIssued"`
	DateExpired         time.Time  `json:"dateExpired"`
	DateAccepted        *time.Time `json:"dateAccepted,omitempty"`
	DateCompleted       *time.Time `json:"dateCompleted,omitempty"`
	DaysToComplete      *int64     `json:"daysToComplete,omitempty"`
	EndLocationID       *int64     `json:"endLocationId,omitempty"`
	StartLocationID     *int64     `json:"startLocationId,omitempty"`
	Buyout              *float64   `json:"buyout,omitempty"`
	Collateral          *float64   `json:"collateral,omitempty"`
	Price               *float64   `json:"price,omitempty"`
	Reward              *float64   `json:"reward,omitempty"`
	Volume              *float64   `json:"volume,omitempty"`
	Title               *string    `json:"title,omitempty"`
}

// ContractBid is one bid, shared by character/corporation/public listings. BidderID is only
// populated for character-scoped bids (the only variant whose model exposes it).
type ContractBid struct {
	BidID    int64      `json:"bidId"`
	BidderID *int64     `json:"bidderId,omitempty"`
	Amount   float64    `json:"amount"`
	DateBid  time.Time  `json:"dateBid"`
}

// ContractItem is one item, shared by character/corporation/public listings. Field availability
// differs slightly by scope (see esi/contracts.go's model documentation).
type ContractItem struct {
	RecordID           int64  `json:"recordId"`
	TypeID             int64  `json:"typeId"`
	Quantity           int64  `json:"quantity"`
	IsIncluded         bool   `json:"isIncluded"`
	IsSingleton        bool   `json:"isSingleton,omitempty"`
	RawQuantity        *int64 `json:"rawQuantity,omitempty"`
	ItemID             *int64 `json:"itemId,omitempty"`
	MaterialEfficiency *int64 `json:"materialEfficiency,omitempty"`
	Runs               *int64 `json:"runs,omitempty"`
	TimeEfficiency     *int64 `json:"timeEfficiency,omitempty"`
	IsBlueprintCopy    *bool  `json:"isBlueprintCopy,omitempty"`
}

// --- list_character_contracts ---

var listCharacterContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Contract, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	contracts, totalPages, err := client.CharacterContracts(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contract, len(contracts))
	for i, c := range contracts {
		out[i] = Contract{
			ContractID: c.ContractId, AcceptorID: c.AcceptorId, AssigneeID: c.AssigneeId,
			IssuerID: c.IssuerId, IssuerCorporationID: c.IssuerCorporationId, Availability: c.Availability,
			Status: c.Status, Type: c.Type, ForCorporation: c.ForCorporation,
			DateIssued: c.DateIssued, DateExpired: c.DateExpired, DateAccepted: c.DateAccepted,
			DateCompleted: c.DateCompleted, DaysToComplete: c.DaysToComplete, EndLocationID: c.EndLocationId,
			StartLocationID: c.StartLocationId, Buyout: c.Buyout, Collateral: c.Collateral,
			Price: c.Price, Reward: c.Reward, Volume: c.Volume, Title: c.Title,
		}
	}
	return out, totalPages, nil
}

type ListCharacterContractsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterContractsOutput struct {
	CharacterID int32      `json:"characterId"`
	Page        int32      `json:"page"`
	TotalPages  int32      `json:"totalPages"`
	Contracts   []Contract `json:"contracts"`
}

func listCharacterContractsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterContractsInput, ListCharacterContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractsInput) (*mcpsdk.CallToolResult, ListCharacterContractsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractsOutput{}, fmt.Errorf("list character contracts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contracts, totalPages, err := listCharacterContracts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListCharacterContractsOutput{}, fmt.Errorf("list character contracts: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCharacterContractsOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Contracts: contracts}, nil
	}
}

// --- list_character_contract_bids ---

var listCharacterContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractBid, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	bids, err := client.CharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, err
	}
	out := make([]ContractBid, len(bids))
	for i, b := range bids {
		bidderID := b.BidderId
		out[i] = ContractBid{BidID: b.BidId, BidderID: &bidderID, Amount: b.Amount, DateBid: b.DateBid}
	}
	return out, nil
}

type ListCharacterContractBidsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	ContractID  int64 `json:"contractId" jsonschema:"the contract ID, from list_character_contracts"`
}
type ListCharacterContractBidsOutput struct {
	CharacterID int32         `json:"characterId"`
	ContractID  int64         `json:"contractId"`
	Bids        []ContractBid `json:"bids"`
}

func listCharacterContractBidsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterContractBidsInput, ListCharacterContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractBidsInput) (*mcpsdk.CallToolResult, ListCharacterContractBidsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractBidsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractBidsOutput{}, fmt.Errorf("list character contract bids for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		bids, err := listCharacterContractBids(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContractID)
		if err != nil {
			return nil, ListCharacterContractBidsOutput{}, fmt.Errorf("list character contract bids: %w", err)
		}
		return nil, ListCharacterContractBidsOutput{CharacterID: in.CharacterID, ContractID: in.ContractID, Bids: bids}, nil
	}
}

// --- list_character_contract_items ---

var listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	items, err := client.CharacterContractItems(ctx, contractID)
	if err != nil {
		return nil, err
	}
	out := make([]ContractItem, len(items))
	for i, it := range items {
		out[i] = ContractItem{RecordID: it.RecordId, TypeID: it.TypeId, Quantity: it.Quantity, IsIncluded: it.IsIncluded, IsSingleton: it.IsSingleton, RawQuantity: it.RawQuantity}
	}
	return out, nil
}

type ListCharacterContractItemsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	ContractID  int64 `json:"contractId" jsonschema:"the contract ID, from list_character_contracts"`
}
type ListCharacterContractItemsOutput struct {
	CharacterID int32          `json:"characterId"`
	ContractID  int64          `json:"contractId"`
	Items       []ContractItem `json:"items"`
}

func listCharacterContractItemsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterContractItemsInput, ListCharacterContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractItemsInput) (*mcpsdk.CallToolResult, ListCharacterContractItemsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractItemsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractItemsOutput{}, fmt.Errorf("list character contract items for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, err := listCharacterContractItems(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContractID)
		if err != nil {
			return nil, ListCharacterContractItemsOutput{}, fmt.Errorf("list character contract items: %w", err)
		}
		return nil, ListCharacterContractItemsOutput{CharacterID: in.CharacterID, ContractID: in.ContractID, Items: items}, nil
	}
}

// --- list_corporation_contracts ---

var listCorporationContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Contract, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	contracts, totalPages, err := client.CorporationContracts(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contract, len(contracts))
	for i, c := range contracts {
		out[i] = Contract{
			ContractID: c.ContractId, AcceptorID: c.AcceptorId, AssigneeID: c.AssigneeId,
			IssuerID: c.IssuerId, IssuerCorporationID: c.IssuerCorporationId, Availability: c.Availability,
			Status: c.Status, Type: c.Type, ForCorporation: c.ForCorporation,
			DateIssued: c.DateIssued, DateExpired: c.DateExpired, DateAccepted: c.DateAccepted,
			DateCompleted: c.DateCompleted, DaysToComplete: c.DaysToComplete, EndLocationID: c.EndLocationId,
			StartLocationID: c.StartLocationId, Buyout: c.Buyout, Collateral: c.Collateral,
			Price: c.Price, Reward: c.Reward, Volume: c.Volume, Title: c.Title,
		}
	}
	return out, totalPages, nil
}

type ListCorporationContractsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationContractsOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Page          int32      `json:"page"`
	TotalPages    int32      `json:"totalPages"`
	Contracts     []Contract `json:"contracts"`
}

func listCorporationContractsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationContractsInput, ListCorporationContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractsInput) (*mcpsdk.CallToolResult, ListCorporationContractsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractsOutput{}, fmt.Errorf("list corporation contracts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contracts, totalPages, err := listCorporationContracts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationContractsOutput{}, fmt.Errorf("list corporation contracts: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationContractsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Contracts: contracts}, nil
	}
}

// --- list_corporation_contract_bids ---

var listCorporationContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	bids, totalPages, err := client.CorporationContractBids(ctx, corporationID, contractID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContractBid, len(bids))
	for i, b := range bids {
		out[i] = ContractBid{BidID: b.BidId, Amount: b.Amount, DateBid: b.DateBid}
	}
	return out, totalPages, nil
}

type ListCorporationContractBidsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ContractID    int64 `json:"contractId" jsonschema:"the contract ID, from list_corporation_contracts"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationContractBidsOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	ContractID    int64         `json:"contractId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Bids          []ContractBid `json:"bids"`
}

func listCorporationContractBidsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationContractBidsInput, ListCorporationContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractBidsInput) (*mcpsdk.CallToolResult, ListCorporationContractBidsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractBidsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractBidsOutput{}, fmt.Errorf("list corporation contract bids for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		bids, totalPages, err := listCorporationContractBids(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ContractID, in.Page)
		if err != nil {
			return nil, ListCorporationContractBidsOutput{}, fmt.Errorf("list corporation contract bids: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationContractBidsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, ContractID: in.ContractID, Page: page, TotalPages: totalPages, Bids: bids}, nil
	}
}

// --- list_corporation_contract_items ---

var listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	items, err := client.CorporationContractItems(ctx, corporationID, contractID)
	if err != nil {
		return nil, err
	}
	out := make([]ContractItem, len(items))
	for i, it := range items {
		out[i] = ContractItem{RecordID: it.RecordId, TypeID: it.TypeId, Quantity: it.Quantity, IsIncluded: it.IsIncluded, IsSingleton: it.IsSingleton, RawQuantity: it.RawQuantity}
	}
	return out, nil
}

type ListCorporationContractItemsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ContractID    int64 `json:"contractId" jsonschema:"the contract ID, from list_corporation_contracts"`
}
type ListCorporationContractItemsOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	ContractID    int64          `json:"contractId"`
	Items         []ContractItem `json:"items"`
}

func listCorporationContractItemsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationContractItemsInput, ListCorporationContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractItemsInput) (*mcpsdk.CallToolResult, ListCorporationContractItemsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractItemsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractItemsOutput{}, fmt.Errorf("list corporation contract items for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, err := listCorporationContractItems(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ContractID)
		if err != nil {
			return nil, ListCorporationContractItemsOutput{}, fmt.Errorf("list corporation contract items: %w", err)
		}
		return nil, ListCorporationContractItemsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, ContractID: in.ContractID, Items: items}, nil
	}
}

// --- list_public_contracts ---

var listPublicContracts = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]Contract, int32, error) {
	client := esi.NewPublicClient(userAgent)
	contracts, totalPages, err := client.PublicContracts(ctx, regionID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contract, len(contracts))
	for i, c := range contracts {
		var forCorp bool
		if c.ForCorporation != nil {
			forCorp = *c.ForCorporation
		}
		out[i] = Contract{
			ContractID: c.ContractId, IssuerID: c.IssuerId, IssuerCorporationID: c.IssuerCorporationId,
			Type: c.Type, ForCorporation: forCorp, DateIssued: c.DateIssued, DateExpired: c.DateExpired,
			DaysToComplete: c.DaysToComplete, EndLocationID: c.EndLocationId, StartLocationID: c.StartLocationId,
			Buyout: c.Buyout, Collateral: c.Collateral, Price: c.Price, Reward: c.Reward, Volume: c.Volume, Title: c.Title,
		}
	}
	return out, totalPages, nil
}

type ListPublicContractsInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
	Page     int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListPublicContractsOutput struct {
	RegionID   int64      `json:"regionId"`
	Page       int32      `json:"page"`
	TotalPages int32      `json:"totalPages"`
	Contracts  []Contract `json:"contracts"`
}

func listPublicContractsHandler(version string) mcpsdk.ToolHandlerFor[ListPublicContractsInput, ListPublicContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractsInput) (*mcpsdk.CallToolResult, ListPublicContractsOutput, error) {
		contracts, totalPages, err := listPublicContracts(ctx, sso.UserAgent(version), in.RegionID, in.Page)
		if err != nil {
			return nil, ListPublicContractsOutput{}, fmt.Errorf("list public contracts: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListPublicContractsOutput{RegionID: in.RegionID, Page: page, TotalPages: totalPages, Contracts: contracts}, nil
	}
}

// --- list_public_contract_bids ---

var listPublicContractBids = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractBid, int32, error) {
	client := esi.NewPublicClient(userAgent)
	bids, totalPages, err := client.PublicContractBids(ctx, contractID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContractBid, len(bids))
	for i, b := range bids {
		out[i] = ContractBid{BidID: b.BidId, Amount: b.Amount, DateBid: b.DateBid}
	}
	return out, totalPages, nil
}

type ListPublicContractBidsInput struct {
	ContractID int64 `json:"contractId" jsonschema:"the contract ID, from list_public_contracts"`
	Page       int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListPublicContractBidsOutput struct {
	ContractID int64         `json:"contractId"`
	Page       int32         `json:"page"`
	TotalPages int32         `json:"totalPages"`
	Bids       []ContractBid `json:"bids"`
}

func listPublicContractBidsHandler(version string) mcpsdk.ToolHandlerFor[ListPublicContractBidsInput, ListPublicContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractBidsInput) (*mcpsdk.CallToolResult, ListPublicContractBidsOutput, error) {
		bids, totalPages, err := listPublicContractBids(ctx, sso.UserAgent(version), in.ContractID, in.Page)
		if err != nil {
			return nil, ListPublicContractBidsOutput{}, fmt.Errorf("list public contract bids: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListPublicContractBidsOutput{ContractID: in.ContractID, Page: page, TotalPages: totalPages, Bids: bids}, nil
	}
}

// --- list_public_contract_items ---

var listPublicContractItems = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractItem, int32, error) {
	client := esi.NewPublicClient(userAgent)
	items, totalPages, err := client.PublicContractItems(ctx, contractID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContractItem, len(items))
	for i, it := range items {
		out[i] = ContractItem{
			RecordID: it.RecordId, TypeID: it.TypeId, Quantity: it.Quantity, IsIncluded: it.IsIncluded,
			ItemID: it.ItemId, MaterialEfficiency: it.MaterialEfficiency, Runs: it.Runs,
			TimeEfficiency: it.TimeEfficiency, IsBlueprintCopy: it.IsBlueprintCopy,
		}
	}
	return out, totalPages, nil
}

type ListPublicContractItemsInput struct {
	ContractID int64 `json:"contractId" jsonschema:"the contract ID, from list_public_contracts"`
	Page       int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListPublicContractItemsOutput struct {
	ContractID int64          `json:"contractId"`
	Page       int32          `json:"page"`
	TotalPages int32          `json:"totalPages"`
	Items      []ContractItem `json:"items"`
}

func listPublicContractItemsHandler(version string) mcpsdk.ToolHandlerFor[ListPublicContractItemsInput, ListPublicContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractItemsInput) (*mcpsdk.CallToolResult, ListPublicContractItemsOutput, error) {
		items, totalPages, err := listPublicContractItems(ctx, sso.UserAgent(version), in.ContractID, in.Page)
		if err != nil {
			return nil, ListPublicContractItemsOutput{}, fmt.Errorf("list public contract items: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListPublicContractItemsOutput{ContractID: in.ContractID, Page: page, TotalPages: totalPages, Items: items}, nil
	}
}
```

- [ ] **Step 4: Register the nine tools in `mcp/server.go`**

Add 9 new `mcpsdk.AddTool` calls right after `remove_character_contacts`'s (the last tool from
the prior batch), updating the `NewServer` doc comment. Use these exact `Name`/`Description`/
handler pairs:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contracts",
		Description: "List a logged-in EVE Online character's contracts.",
	}, listCharacterContractsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contract_bids",
		Description: "List the bids on one of a logged-in character's contracts.",
	}, listCharacterContractBidsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contract_items",
		Description: "List the items on one of a logged-in character's contracts.",
	}, listCharacterContractItemsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contracts",
		Description: "List a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contract_bids",
		Description: "List the bids on one of a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractBidsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contract_items",
		Description: "List the items on one of a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractItemsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_public_contracts",
		Description: "List public contracts in a region (public data, no character required).",
	}, listPublicContractsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_public_contract_bids",
		Description: "List the bids on a public contract (public data, no character required).",
	}, listPublicContractBidsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_public_contract_items",
		Description: "List the items on a public contract (public data, no character required).",
	}, listPublicContractItemsHandler(version))
```

- [ ] **Step 5: Update `mcp/server_test.go`'s expected-tools list**

Append these 9 names (48 → 57):

```go
		"list_character_contracts", "list_character_contract_bids", "list_character_contract_items",
		"list_corporation_contracts", "list_corporation_contract_bids", "list_corporation_contract_items",
		"list_public_contracts", "list_public_contract_bids", "list_public_contract_items",
```

- [ ] **Step 6: Update docs**

- `CLAUDE.md`/`README.md`: bump tool count from "forty-eight" to "fifty-seven", append the 9 new
  tool names, add "and Contracts" to README's covered-tags sentence.
- `docs/reference/esi-coverage-roadmap.md`: change the Contracts row from "Planned" to "This plan".

- [ ] **Step 7: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race -v`

Expected: all 17 new tests PASS, `TestServer_AllToolsRegistered` PASSes with all 57 names,
`gofmt -l .` prints nothing, every pre-existing test passes.

- [ ] **Step 8: Commit**

```bash
git add mcp/contracts.go mcp/contracts_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add contracts MCP tools (9 tools: pagination + mixed public/private)"
```
