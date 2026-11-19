# ESI Alliance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cover the Alliance ESI resource group (4 fully public/unauthenticated endpoints), introducing a new "public client" convention for tags where no endpoint needs a logged-in character.

**Architecture:** `esi/client.go` gains a second constructor, `NewPublicClient(userAgent string) *Client`, alongside the existing per-character `NewClient`. `esi/alliance.go` is a thin, `describeError`-wrapped facade exactly like `esi/wallet.go`, but built via the public constructor. `mcp/alliance.go` adds four MCP tools that take **no `characterId` input and no `config.Store` parameter at all** — the first tools in this project with that shape, since there's no character to check.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context: `fmt.Errorf("operation: %w", err)`. Never discard an error.
- Every `esi/` method that calls `.Execute()` must route its error and `*http.Response` through `describeError` (`esi/error.go`) before wrapping — existing hard rule.
- External test packages by default (`package foo_test`); `*_internal_test.go` only when a test genuinely needs an unexported symbol.
- Standard library only in tests — no testify.
- Raw-ID passthrough: no code in this plan resolves any ID to a human-readable name.
- This batch needs **no OAuth scopes at all** — every Alliance endpoint is public (`security: []`, verified against FC's live spec). No character needs to exist for these tools to work.
- **This branch is based on current `main`, which does NOT yet include PR #6** (the Access List/Activities batch, still open). `mcp/server.go` and `mcp/server_test.go` on this branch's base therefore still show the original 4 tools (`add_character`, `list_characters`, `remove_character`, `get_wallet_balance`) — verified by reading both files directly before writing this plan. This plan's tasks are written against that 4-tool baseline. **Whichever of PR #6 or this plan's PR merges second will hit a normal merge conflict in `mcp/server.go`/`mcp/server_test.go`** (both PRs append `AddTool` calls and expected-tool-list entries) — resolve it by keeping both PRs' additions, not by dropping either. This is expected and not a defect in either plan.
- `docs/reference/esi-coverage-roadmap.md` (created by PR #6's Task 1) does **not exist on this branch's base** for the same reason. This plan's docs-sync task therefore does not touch that file — only `CLAUDE.md`/`README.md`, both of which already exist on `main`. Once both PRs are merged, a follow-up (outside either plan) reconciles the roadmap doc's rows and the final tool count across all three merged batches.
- Design spec: `docs/superpowers/specs/2026-07-25-esi-alliance-design.md`.
- All code in this plan has been verified to compile and its exact JSON shapes verified to unmarshal correctly against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module — including a live `httptest.Server` round-trip through the new `ExportedNewPublicClientWithOptions` → `Client` → all four Alliance methods, and a verified end-to-end test that an MCP tool with a genuinely empty input struct (`struct{}`) registers and calls cleanly against the real `github.com/modelcontextprotocol/go-sdk` v1.6.1 — not guessed from documentation.

---

### Task 1: `esi/client.go` public-client addition + `esi/alliance.go` facade

**Files:**
- Modify: `esi/client.go`
- Create: `esi/alliance.go`
- Create: `esi/alliance_test.go`

**Interfaces:**
- Consumes: `goesi.NewPublicESIClient(userAgent string) *esiapi.APIClient` (verified to exist in `github.com/fnt-eve/goesi-openapi@v0.0.37/APIClient.go:26`, wraps `http.DefaultClient`); `goesi.NewESIClientWithOptions(httpClient *http.Client, options goesi.ClientOptions) *esiapi.APIClient` (existing); `esiapi.APIClient.AllianceAPI.GetAlliances(ctx) ...Execute() ([]int64, *http.Response, error)`, `.GetAlliancesAllianceId(ctx, allianceId int64) ...Execute() (*esiapi.AllianceDetail, *http.Response, error)`, `.GetAlliancesAllianceIdCorporations(ctx, allianceId int64) ...Execute() ([]int64, *http.Response, error)`, `.GetAlliancesAllianceIdIcons(ctx, allianceId int64) ...Execute() (*esiapi.AlliancesAllianceIdIconsGet, *http.Response, error)` — all four verified against the real installed module.
- Produces: `esi.NewPublicClient(userAgent string) *Client`, `esi.ExportedNewPublicClientWithOptions(httpClient *http.Client, options goesi.ClientOptions) *Client`, `(*esi.Client).Alliances(ctx) ([]int64, error)`, `(*esi.Client).AllianceDetail(ctx, allianceID int64) (*esiapi.AllianceDetail, error)`, `(*esi.Client).AllianceCorporations(ctx, allianceID int64) ([]int64, error)`, `(*esi.Client).AllianceIcons(ctx, allianceID int64) (*esiapi.AlliancesAllianceIdIconsGet, error)` — all consumed by Task 2.

- [ ] **Step 1: Write the failing tests**

Create `esi/alliance_test.go`:

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

func TestAlliances_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[99000001,99000002]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ids, err := client.Alliances(context.Background())
	if err != nil {
		t.Fatalf("Alliances returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 99000001 || ids[1] != 99000002 {
		t.Errorf("ids = %v, want [99000001 99000002]", ids)
	}
}

func TestAllianceDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"creator_corporation_id":1,"creator_id":2,"date_founded":"2020-01-01T00:00:00Z","name":"Test Alliance","ticker":"TEST"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.AllianceDetail(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceDetail returned error: %v", err)
	}
	if detail.Name != "Test Alliance" || detail.Ticker != "TEST" {
		t.Errorf("detail = %+v, want Name=\"Test Alliance\" Ticker=\"TEST\"", detail)
	}
	if detail.ExecutorCorporationId != nil {
		t.Errorf("detail.ExecutorCorporationId = %v, want nil (field absent from response)", detail.ExecutorCorporationId)
	}
	if detail.FactionId != nil {
		t.Errorf("detail.FactionId = %v, want nil (field absent from response)", detail.FactionId)
	}
}

func TestAllianceDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.AllianceDetail(context.Background(), 99000001); err == nil {
		t.Fatal("AllianceDetail returned nil error, want an error for HTTP 404")
	}
}

func TestAllianceCorporations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[98000001]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	corps, err := client.AllianceCorporations(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceCorporations returned error: %v", err)
	}
	if len(corps) != 1 || corps[0] != 98000001 {
		t.Errorf("corps = %v, want [98000001]", corps)
	}
}

func TestAllianceIcons_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"px128x128":"https://images.evetech.net/a.png"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	icons, err := client.AllianceIcons(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceIcons returned error: %v", err)
	}
	if icons.Px128x128 == nil || *icons.Px128x128 != "https://images.evetech.net/a.png" {
		t.Errorf("icons.Px128x128 = %v, want a pointer to the URL", icons.Px128x128)
	}
	if icons.Px64x64 != nil {
		t.Errorf("icons.Px64x64 = %v, want nil (field absent from response)", icons.Px64x64)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestAlliances|TestAllianceDetail|TestAllianceCorporations|TestAllianceIcons' -v`

Expected: FAIL to compile — `undefined: esi.ExportedNewPublicClientWithOptions`, `undefined: (*esi.Client).Alliances`, etc.

- [ ] **Step 3: Add the public-client constructors to `esi/client.go`**

In `esi/client.go`, append these two functions after the existing `ExportedNewClientWithOptions` (no new imports needed — `net/http` and `goesi "github.com/fnt-eve/goesi-openapi"` are already imported by this file):

```go
// NewPublicClient builds an unauthenticated ESI client for calling public endpoints — no
// character, no OAuth2 token. Used by resource groups where every endpoint is public (e.g.
// Alliance); see docs/superpowers/specs/2026-07-25-esi-alliance-design.md.
func NewPublicClient(userAgent string) *Client {
	api := goesi.NewPublicESIClient(userAgent)
	return &Client{api: api}
}

// ExportedNewPublicClientWithOptions exposes a Client built from an explicit httpClient and
// public ClientOptions for external tests, so tests can point the generated client at an
// httptest.Server instead of real ESI.
func ExportedNewPublicClientWithOptions(httpClient *http.Client, options goesi.ClientOptions) *Client {
	api := goesi.NewESIClientWithOptions(httpClient, options)
	return &Client{api: api}
}
```

`characterID` is left as its zero value on a `Client` built either way — safe, since no Alliance method reads `c.characterID` at all (every method's only ID parameter is `allianceID`, passed explicitly by the caller).

- [ ] **Step 4: Implement `esi/alliance.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Alliances returns every alliance ID currently in the game.
func (c *Client) Alliances(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliances(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliances: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceDetail returns one alliance's name, ticker, founding info, and optional
// executor-corporation/faction-warfare affiliation.
func (c *Client) AllianceDetail(ctx context.Context, allianceID int64) (*esiapi.AllianceDetail, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceId(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance detail: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceCorporations returns the corporation IDs currently in one alliance.
func (c *Client) AllianceCorporations(ctx context.Context, allianceID int64) ([]int64, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceIdCorporations(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance corporations: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceIcons returns one alliance's icon URLs, if it has any.
func (c *Client) AllianceIcons(ctx context.Context, allianceID int64) (*esiapi.AlliancesAllianceIdIconsGet, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceIdIcons(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance icons: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all five new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 6: Commit**

```bash
git add esi/client.go esi/alliance.go esi/alliance_test.go
git commit -m "feat: add esi.NewPublicClient and esi/alliance.go facade"
```

---

### Task 2: `mcp/alliance.go` tools + registration

**Files:**
- Create: `mcp/alliance.go`
- Create: `mcp/alliance_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/server_test.go`

**Interfaces:**
- Consumes: `esi.NewPublicClient(userAgent string) *esi.Client`, `(*esi.Client).Alliances`, `(*esi.Client).AllianceDetail`, `(*esi.Client).AllianceCorporations`, `(*esi.Client).AllianceIcons` (Task 1); `sso.UserAgent(version string) string` (existing).
- Produces: `mcp.ListAlliancesInput{}` (empty), `mcp.ListAlliancesOutput{AllianceIDs []int64}`, `mcp.GetAllianceInput{AllianceID int64}`, `mcp.GetAllianceOutput{...}`, `mcp.ListAllianceCorporationsInput{AllianceID int64}`, `mcp.ListAllianceCorporationsOutput{AllianceID int64, CorporationIDs []int64}`, `mcp.GetAllianceIconsInput{AllianceID int64}`, `mcp.GetAllianceIconsOutput{...}`. Nothing later in this plan depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/alliance_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestListAlliancesHandler_ReturnsIDs(t *testing.T) {
	orig := listAlliances
	t.Cleanup(func() { listAlliances = orig })
	listAlliances = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{99000001, 99000002}, nil
	}

	handler := listAlliancesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListAlliancesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.AllianceIDs) != 2 || out.AllianceIDs[0] != 99000001 {
		t.Errorf("out = %+v, want AllianceIDs=[99000001 99000002]", out)
	}
}

func TestListAlliancesHandler_PropagatesError(t *testing.T) {
	orig := listAlliances
	t.Cleanup(func() { listAlliances = orig })
	listAlliances = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, errors.New("simulated failure")
	}

	handler := listAlliancesHandler("v1")
	if _, _, err := handler(context.Background(), nil, ListAlliancesInput{}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestGetAllianceHandler_ReturnsDetail(t *testing.T) {
	orig := getAllianceDetail
	t.Cleanup(func() { getAllianceDetail = orig })
	getAllianceDetail = func(ctx context.Context, userAgent string, allianceID int64) (AllianceDetailResult, error) {
		execCorp := int64(3)
		return AllianceDetailResult{
			Name:                 "Test Alliance",
			Ticker:               "TEST",
			CreatorID:            2,
			CreatorCorporationID: 1,
			ExecutorCorporationID: &execCorp,
		}, nil
	}

	handler := getAllianceHandler("v1")
	_, out, err := handler(context.Background(), nil, GetAllianceInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || out.Name != "Test Alliance" || out.ExecutorCorporationID == nil || *out.ExecutorCorporationID != 3 {
		t.Errorf("out = %+v, want AllianceID=99000001 Name=\"Test Alliance\" ExecutorCorporationID=*3", out)
	}
	if out.FactionID != nil {
		t.Errorf("out.FactionID = %v, want nil", out.FactionID)
	}
}

func TestListAllianceCorporationsHandler_ReturnsIDs(t *testing.T) {
	orig := listAllianceCorporations
	t.Cleanup(func() { listAllianceCorporations = orig })
	listAllianceCorporations = func(ctx context.Context, userAgent string, allianceID int64) ([]int64, error) {
		return []int64{98000001}, nil
	}

	handler := listAllianceCorporationsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListAllianceCorporationsInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || len(out.CorporationIDs) != 1 || out.CorporationIDs[0] != 98000001 {
		t.Errorf("out = %+v, want AllianceID=99000001 CorporationIDs=[98000001]", out)
	}
}

func TestGetAllianceIconsHandler_ReturnsIcons(t *testing.T) {
	orig := getAllianceIcons
	t.Cleanup(func() { getAllianceIcons = orig })
	getAllianceIcons = func(ctx context.Context, userAgent string, allianceID int64) (AllianceIconsResult, error) {
		url := "https://images.evetech.net/a.png"
		return AllianceIconsResult{Px128x128: &url}, nil
	}

	handler := getAllianceIconsHandler("v1")
	_, out, err := handler(context.Background(), nil, GetAllianceIconsInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || out.Px128x128 == nil || *out.Px128x128 != "https://images.evetech.net/a.png" {
		t.Errorf("out = %+v, want AllianceID=99000001 Px128x128=*\"https://images.evetech.net/a.png\"", out)
	}
	if out.Px64x64 != nil {
		t.Errorf("out.Px64x64 = %v, want nil", out.Px64x64)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListAlliancesHandler|TestGetAllianceHandler|TestListAllianceCorporationsHandler|TestGetAllianceIconsHandler' -v`

Expected: FAIL to compile — all the new `mcp` symbols referenced above are undefined.

- [ ] **Step 3: Implement `mcp/alliance.go`**

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// listAlliances is overridable in tests so no real network call runs.
var listAlliances = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.Alliances(ctx)
}

// ListAlliancesInput is list_alliances's tool input. It takes no fields at all — this
// endpoint is fully public and returns every alliance ID in the game.
type ListAlliancesInput struct{}

// ListAlliancesOutput is list_alliances's tool result.
type ListAlliancesOutput struct {
	AllianceIDs []int64 `json:"allianceIds"`
}

func listAlliancesHandler(version string) mcpsdk.ToolHandlerFor[ListAlliancesInput, ListAlliancesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAlliancesInput) (*mcpsdk.CallToolResult, ListAlliancesOutput, error) {
		ids, err := listAlliances(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListAlliancesOutput{}, fmt.Errorf("list alliances: %w", err)
		}
		return nil, ListAlliancesOutput{AllianceIDs: ids}, nil
	}
}

// AllianceDetailResult is the raw mapped detail returned by getAllianceDetail, before the
// handler adds the allianceID it already knows from the request.
type AllianceDetailResult struct {
	Name                  string
	Ticker                string
	CreatorID             int64
	CreatorCorporationID  int64
	DateFounded           time.Time
	ExecutorCorporationID *int64
	FactionID             *int64
}

// getAllianceDetail is overridable in tests.
var getAllianceDetail = func(ctx context.Context, userAgent string, allianceID int64) (AllianceDetailResult, error) {
	client := esi.NewPublicClient(userAgent)
	detail, err := client.AllianceDetail(ctx, allianceID)
	if err != nil {
		return AllianceDetailResult{}, err
	}
	return AllianceDetailResult{
		Name:                  detail.Name,
		Ticker:                detail.Ticker,
		CreatorID:             detail.CreatorId,
		CreatorCorporationID:  detail.CreatorCorporationId,
		DateFounded:           detail.DateFounded,
		ExecutorCorporationID: detail.ExecutorCorporationId,
		FactionID:             detail.FactionId,
	}, nil
}

// GetAllianceInput is get_alliance's tool input.
type GetAllianceInput struct {
	AllianceID int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}

// GetAllianceOutput is get_alliance's tool result.
type GetAllianceOutput struct {
	AllianceID            int64     `json:"allianceId"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
	CreatorID             int64     `json:"creatorId"`
	CreatorCorporationID  int64     `json:"creatorCorporationId"`
	DateFounded           time.Time `json:"dateFounded"`
	ExecutorCorporationID *int64    `json:"executorCorporationId,omitempty"`
	FactionID             *int64    `json:"factionId,omitempty"`
}

func getAllianceHandler(version string) mcpsdk.ToolHandlerFor[GetAllianceInput, GetAllianceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAllianceInput) (*mcpsdk.CallToolResult, GetAllianceOutput, error) {
		result, err := getAllianceDetail(ctx, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, GetAllianceOutput{}, fmt.Errorf("get alliance detail: %w", err)
		}
		return nil, GetAllianceOutput{
			AllianceID:            in.AllianceID,
			Name:                  result.Name,
			Ticker:                result.Ticker,
			CreatorID:             result.CreatorID,
			CreatorCorporationID:  result.CreatorCorporationID,
			DateFounded:           result.DateFounded,
			ExecutorCorporationID: result.ExecutorCorporationID,
			FactionID:             result.FactionID,
		}, nil
	}
}

// listAllianceCorporations is overridable in tests.
var listAllianceCorporations = func(ctx context.Context, userAgent string, allianceID int64) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.AllianceCorporations(ctx, allianceID)
}

// ListAllianceCorporationsInput is list_alliance_corporations's tool input.
type ListAllianceCorporationsInput struct {
	AllianceID int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}

// ListAllianceCorporationsOutput is list_alliance_corporations's tool result.
type ListAllianceCorporationsOutput struct {
	AllianceID     int64   `json:"allianceId"`
	CorporationIDs []int64 `json:"corporationIds"`
}

func listAllianceCorporationsHandler(version string) mcpsdk.ToolHandlerFor[ListAllianceCorporationsInput, ListAllianceCorporationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAllianceCorporationsInput) (*mcpsdk.CallToolResult, ListAllianceCorporationsOutput, error) {
		ids, err := listAllianceCorporations(ctx, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, ListAllianceCorporationsOutput{}, fmt.Errorf("list alliance corporations: %w", err)
		}
		return nil, ListAllianceCorporationsOutput{AllianceID: in.AllianceID, CorporationIDs: ids}, nil
	}
}

// AllianceIconsResult is the raw mapped result returned by getAllianceIcons, before the
// handler adds the allianceID it already knows from the request.
type AllianceIconsResult struct {
	Px128x128 *string
	Px64x64   *string
}

// getAllianceIcons is overridable in tests.
var getAllianceIcons = func(ctx context.Context, userAgent string, allianceID int64) (AllianceIconsResult, error) {
	client := esi.NewPublicClient(userAgent)
	icons, err := client.AllianceIcons(ctx, allianceID)
	if err != nil {
		return AllianceIconsResult{}, err
	}
	return AllianceIconsResult{Px128x128: icons.Px128x128, Px64x64: icons.Px64x64}, nil
}

// GetAllianceIconsInput is get_alliance_icons's tool input.
type GetAllianceIconsInput struct {
	AllianceID int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}

// GetAllianceIconsOutput is get_alliance_icons's tool result.
type GetAllianceIconsOutput struct {
	AllianceID int64   `json:"allianceId"`
	Px128x128  *string `json:"px128x128,omitempty"`
	Px64x64    *string `json:"px64x64,omitempty"`
}

func getAllianceIconsHandler(version string) mcpsdk.ToolHandlerFor[GetAllianceIconsInput, GetAllianceIconsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAllianceIconsInput) (*mcpsdk.CallToolResult, GetAllianceIconsOutput, error) {
		result, err := getAllianceIcons(ctx, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, GetAllianceIconsOutput{}, fmt.Errorf("get alliance icons: %w", err)
		}
		return nil, GetAllianceIconsOutput{
			AllianceID: in.AllianceID,
			Px128x128:  result.Px128x128,
			Px64x64:    result.Px64x64,
		}, nil
	}
}
```

This file needs a `"time"` import for `time.Time` — the full import block is:

```go
import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)
```

- [ ] **Step 4: Register the four tools in `mcp/server.go`**

In `mcp/server.go`, change the doc comment and add four new `mcpsdk.AddTool` calls right after
`get_wallet_balance`'s. Change:

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
	}, getWalletBalanceHandler(version, clientID, store))

	return server
}
```

to:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, get_wallet_balance, list_alliances, get_alliance,
// list_alliance_corporations, and get_alliance_icons registered.
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
	}, getWalletBalanceHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliances",
		Description: "List every alliance ID currently in EVE Online (public data, no character required).",
	}, listAlliancesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_alliance",
		Description: "Get an alliance's name, ticker, founding info, and executor corporation (public data, no character required).",
	}, getAllianceHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliance_corporations",
		Description: "List the corporation IDs currently in an alliance (public data, no character required).",
	}, listAllianceCorporationsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_alliance_icons",
		Description: "Get an alliance's icon URLs (public data, no character required).",
	}, getAllianceIconsHandler(version))

	return server
}
```

- [ ] **Step 5: Update `mcp/server_test.go`'s expected-tools list**

In `mcp/server_test.go`, change:

```go
	for _, want := range []string{"add_character", "list_characters", "remove_character", "get_wallet_balance"} {
```

to:

```go
	for _, want := range []string{
		"add_character", "list_characters", "remove_character", "get_wallet_balance",
		"list_alliances", "get_alliance", "list_alliance_corporations", "get_alliance_icons",
	} {
```

- [ ] **Step 6: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./... -race -v`

Expected: every new test from Step 1 PASSes, `TestServer_AllToolsRegistered` PASSes with all 8
tool names, and every pre-existing test across the whole module still passes.

- [ ] **Step 7: Commit**

```bash
git add mcp/alliance.go mcp/alliance_internal_test.go mcp/server.go mcp/server_test.go
git commit -m "feat: add list_alliances, get_alliance, list_alliance_corporations, get_alliance_icons MCP tools"
```

---

### Task 3: Docs sync (`CLAUDE.md`, `README.md`)

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Interfaces:** none — documentation only.

- [ ] **Step 1: Update `CLAUDE.md`'s package table**

Replace:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its four tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance` |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi` — builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client. Holds no state of its own |
```

with:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its eight tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance`, `list_alliances`, `get_alliance`, `list_alliance_corporations`, `get_alliance_icons` |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi`. `NewClient` builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client; `NewPublicClient` builds an unauthenticated client for resource groups where every endpoint is public (e.g. Alliance — no character needed at all). Holds no state of its own |
```

(Note: this branch does not yet include PR #6's Access List/Activities tools or its
`docs/reference/esi-coverage-roadmap.md` doc, since that PR hasn't merged — this update is written
against the current 4-tool baseline on `main`. Whichever PR merges second will need to reconcile
this table's tool count and wording; that reconciliation is not part of this task.)

- [ ] **Step 2: Update `README.md`**

Replace the `## Status` section:

```markdown
Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), the MCP server with its character-management tools (`mcp/`, `neat mcp`),
and a first real ESI-backed domain tool (`esi/`, `get_wallet_balance`) are built. The SDE cache
and plugin/extension packaging don't exist yet. See the plans directory for what's next.
```

with:

```markdown
Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), the MCP server with its character-management tools (`mcp/`, `neat mcp`),
and real ESI-backed domain tools (`esi/`) covering Wallet and Alliance are built. The SDE cache
and plugin/extension packaging don't exist yet. See the plans directory for what's next.
```

Update the package table the same way as `CLAUDE.md`'s Step 1 (tool count, `esi/` row).

- [ ] **Step 3: Sanity-check the docs against the code**

Run: `grep -n "list_alliances\|get_alliance_icons\|NewPublicClient" CLAUDE.md README.md`

Expected: both files mention the new tool names and the `NewPublicClient` addition.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: document Alliance tools and esi.NewPublicClient"
```
