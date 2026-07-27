# ESI Market Implementation Plan (twenty-fourth batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Market" tag (11 endpoints: character/corp market orders +
history, market groups, prices, region history/orders/types, structure market orders) as 11 new
MCP tools, folding into the same shared PR #6 branch as every prior batch.

**Architecture:** Same two-layer split as every prior batch. `esi/market.go` wraps the 11 generated
`MarketAPIService` methods: 2 private "my own character", 2 corp-scoped (character authenticates,
corporation ID is the resource, ESI's own 403 is the sole gate), 6 public, and 1
(`GetMarketsStructuresStructureId`) that looks public but requires character auth (verified via its
OAuth2 security scope). `mcp/market.go` exposes 11 tools on top of that, with two genuine shared
output types (`MarketOrder` across the 4 order-listing tools, `MarketOrderBookEntry` across the 2
order-book tools) following the `IndustryJob` shared-type precedent from `esi/industry.go`.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: 146 MCP tools, branch `worktree-esi-access-list-activities`,
  PR #6 (open). This plan brings the count to 157.
- Pagination is verified **per-endpoint**, not assumed from tag similarity: paginated —
  `CharacterMarketOrderHistory`, `CorporationMarketOrders`, `CorporationMarketOrderHistory`,
  `MarketOrders` (region), `MarketTypes` (region), `StructureMarketOrders`; **not** paginated —
  `CharacterMarketOrders`, `MarketGroups`, `MarketGroup`, `MarketPrices`, `MarketHistory`.
- `GetMarketsRegionIdHistory`'s `TypeId` and `GetMarketsRegionIdOrders`'s `OrderType` are both
  **required** at the ESI-request level (the generated `Execute` bodies error if unset), even
  though `openapi.yaml` documents a spec-level default for `OrderType` that the generated code does
  not apply. Both facade methods must always set them.
- `GetMarketsStructuresStructureId` requires OAuth2 scope `esi-markets.structure_markets.v1`
  (verified in `esi/api/openapi.yaml`) — despite its shape resembling the public
  `GetMarketsRegionIdOrders`, it is modeled as "my own character" auth (`esi.NewClient`, a
  `characterId` input, the standard `characterExists` gate), never `esi.NewPublicClient`.
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- Reuse `characterExists(store *config.Store, characterID int32) (bool, error)`,
  `ErrCharacterNotFound` (`mcp/character.go`), and `totalPagesFromHeader(resp *http.Response) int32`
  (`esi/contracts.go`) — do not redeclare.
- **Standing convention (added after the Killmails batch):** never model a nested/tree-shaped ESI
  resource with a self-referential Go struct. Not applicable to this batch's models (all flat) —
  already sandbox-verified with a live `mcpsdk.AddTool`+`Connect` registration test with zero
  schema-generator errors, but Task 2 must still run the real `go test ./... -race` before commit.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool`+`Connect` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-market-design.md` for the verification detail.

---

### Task 1: `esi/market.go` — ESI facade layer

**Files:**
- Create: `esi/market.go`
- Test: `esi/market_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`);
  `totalPagesFromHeader(resp *http.Response) int32` (`esi/contracts.go`); `c.api.MarketAPI` and
  `c.characterID` (`esi/client.go`'s `Client`, unchanged).
- Produces (for Task 2):
  `func (c *Client) CharacterMarketOrders(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error)`;
  `func (c *Client) CharacterMarketOrderHistory(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error)`;
  `func (c *Client) CorporationMarketOrders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error)`;
  `func (c *Client) CorporationMarketOrderHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error)`;
  `func (c *Client) MarketGroups(ctx context.Context) ([]int64, error)`;
  `func (c *Client) MarketGroup(ctx context.Context, marketGroupID int64) (*esiapi.MarketsGroupsMarketGroupIdGet, error)`;
  `func (c *Client) MarketPrices(ctx context.Context) ([]esiapi.MarketsPricesGetInner, error)`;
  `func (c *Client) MarketHistory(ctx context.Context, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error)`;
  `func (c *Client) MarketOrders(ctx context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error)`;
  `func (c *Client) MarketTypes(ctx context.Context, regionID int64, page int32) ([]int64, int32, error)`;
  `func (c *Client) StructureMarketOrders(ctx context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error)`.

- [ ] **Step 1: Write the failing tests**

Create `esi/market_test.go`:

```go
package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterMarketOrders_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{
			"duration": 90,
			"escrow": 100.5,
			"is_buy_order": true,
			"is_corporation": false,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 123456,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"type_id": 34,
			"volume_remain": 500,
			"volume_total": 1000
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	orders, err := client.CharacterMarketOrders(context.Background())
	if err != nil {
		t.Fatalf("CharacterMarketOrders returned error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %+v, want one entry", orders)
	}
	o := orders[0]
	if o.Duration != 90 {
		t.Errorf("Duration = %d, want 90", o.Duration)
	}
	if o.Escrow == nil || *o.Escrow != 100.5 {
		t.Errorf("Escrow = %v, want 100.5", o.Escrow)
	}
	if o.IsBuyOrder == nil || !*o.IsBuyOrder {
		t.Errorf("IsBuyOrder = %v, want true", o.IsBuyOrder)
	}
	if o.IsCorporation {
		t.Errorf("IsCorporation = %v, want false", o.IsCorporation)
	}
	wantIssued := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if !o.Issued.Equal(wantIssued) {
		t.Errorf("Issued = %v, want %v", o.Issued, wantIssued)
	}
	if o.LocationId != 60003760 {
		t.Errorf("LocationId = %d, want 60003760", o.LocationId)
	}
	if o.MinVolume == nil || *o.MinVolume != 1 {
		t.Errorf("MinVolume = %v, want 1", o.MinVolume)
	}
	if o.OrderId != 123456 {
		t.Errorf("OrderId = %d, want 123456", o.OrderId)
	}
	if o.Price != 5.55 {
		t.Errorf("Price = %v, want 5.55", o.Price)
	}
	if o.Range != "region" {
		t.Errorf("Range = %q, want \"region\"", o.Range)
	}
	if o.RegionId != 10000002 {
		t.Errorf("RegionId = %d, want 10000002", o.RegionId)
	}
	if o.TypeId != 34 {
		t.Errorf("TypeId = %d, want 34", o.TypeId)
	}
	if o.VolumeRemain != 500 {
		t.Errorf("VolumeRemain = %d, want 500", o.VolumeRemain)
	}
	if o.VolumeTotal != 1000 {
		t.Errorf("VolumeTotal = %d, want 1000", o.VolumeTotal)
	}
}

func TestCharacterMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterMarketOrders(context.Background()); err == nil {
		t.Fatal("CharacterMarketOrders returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterMarketOrderHistory_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{
			"duration": 90,
			"is_corporation": true,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 654321,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"state": "expired",
			"type_id": 34,
			"volume_remain": 0,
			"volume_total": 1000
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	history, totalPages, err := client.CharacterMarketOrderHistory(context.Background(), 2)
	if err != nil {
		t.Fatalf("CharacterMarketOrderHistory returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want \"page=2\"", gotQuery)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(history) != 1 {
		t.Fatalf("history = %+v, want one entry", history)
	}
	h := history[0]
	if h.OrderId != 654321 || h.State != "expired" || h.VolumeRemain != 0 || !h.IsCorporation {
		t.Errorf("history[0] = %+v, want OrderId=654321 State=expired VolumeRemain=0 IsCorporation=true", h)
	}
}

func TestCharacterMarketOrderHistory_PageOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterMarketOrderHistory(context.Background(), 0); err != nil {
		t.Fatalf("CharacterMarketOrderHistory returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
}

func TestCharacterMarketOrderHistory_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterMarketOrderHistory(context.Background(), 0); err == nil {
		t.Fatal("CharacterMarketOrderHistory returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationMarketOrders_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{
			"duration": 90,
			"issued": "2026-07-20T00:00:00Z",
			"issued_by": 998877,
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 111222,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"type_id": 34,
			"volume_remain": 500,
			"volume_total": 1000,
			"wallet_division": 1
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	orders, totalPages, err := client.CorporationMarketOrders(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationMarketOrders returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want \"page=1\"", gotQuery)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %+v, want one entry", orders)
	}
	o := orders[0]
	if o.IssuedBy != 998877 || o.WalletDivision != 1 || o.OrderId != 111222 {
		t.Errorf("orders[0] = %+v, want IssuedBy=998877 WalletDivision=1 OrderId=111222", o)
	}
}

func TestCorporationMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationMarketOrders(context.Background(), 98000001, 0); err == nil {
		t.Fatal("CorporationMarketOrders returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationMarketOrderHistory_ParsesResponseAndPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{
			"duration": 90,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 333444,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"state": "cancelled",
			"type_id": 34,
			"volume_remain": 0,
			"volume_total": 1000,
			"wallet_division": 2
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	history, totalPages, err := client.CorporationMarketOrderHistory(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationMarketOrderHistory returned error: %v", err)
	}
	if totalPages != 1 {
		t.Errorf("totalPages = %d, want 1", totalPages)
	}
	if len(history) != 1 {
		t.Fatalf("history = %+v, want one entry", history)
	}
	h := history[0]
	// IssuedBy is optional (*int64) for corp order history, unlike the required int64 on
	// CorporationMarketOrders -- this fixture omits it entirely to confirm it decodes as nil.
	if h.IssuedBy != nil {
		t.Errorf("IssuedBy = %v, want nil (omitted from fixture)", h.IssuedBy)
	}
	if h.State != "cancelled" || h.WalletDivision != 2 || h.OrderId != 333444 {
		t.Errorf("history[0] = %+v, want State=cancelled WalletDivision=2 OrderId=333444", h)
	}
}

func TestCorporationMarketOrderHistory_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationMarketOrderHistory(context.Background(), 98000001, 0); err == nil {
		t.Fatal("CorporationMarketOrderHistory returned nil error, want an error for HTTP 403")
	}
}

func TestMarketGroups_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[1, 2, 3]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	groups, err := client.MarketGroups(context.Background())
	if err != nil {
		t.Fatalf("MarketGroups returned error: %v", err)
	}
	if len(groups) != 3 || groups[0] != 1 || groups[1] != 2 || groups[2] != 3 {
		t.Errorf("groups = %+v, want [1 2 3]", groups)
	}
}

func TestMarketGroups_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketGroups(context.Background()); err == nil {
		t.Fatal("MarketGroups returned nil error, want an error for HTTP 500")
	}
}

func TestMarketGroup_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"description":"Ammo & Charges","market_group_id":9,"name":"Ammo & Charges","parent_group_id":1,"types":[10,11]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	group, err := client.MarketGroup(context.Background(), 9)
	if err != nil {
		t.Fatalf("MarketGroup returned error: %v", err)
	}
	if group.Description != "Ammo & Charges" || group.MarketGroupId != 9 || group.Name != "Ammo & Charges" {
		t.Errorf("group = %+v, want Description/Name=\"Ammo & Charges\" MarketGroupId=9", group)
	}
	if group.ParentGroupId == nil || *group.ParentGroupId != 1 {
		t.Errorf("ParentGroupId = %v, want 1", group.ParentGroupId)
	}
	if len(group.Types) != 2 || group.Types[0] != 10 || group.Types[1] != 11 {
		t.Errorf("Types = %+v, want [10 11]", group.Types)
	}
}

func TestMarketGroup_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketGroup(context.Background(), 9); err == nil {
		t.Fatal("MarketGroup returned nil error, want an error for HTTP 404")
	}
}

func TestMarketPrices_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"adjusted_price":306.11,"average_price":306.29,"type_id":32772}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	prices, err := client.MarketPrices(context.Background())
	if err != nil {
		t.Fatalf("MarketPrices returned error: %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("prices = %+v, want one entry", prices)
	}
	p := prices[0]
	if p.AdjustedPrice == nil || *p.AdjustedPrice != 306.11 {
		t.Errorf("AdjustedPrice = %v, want 306.11", p.AdjustedPrice)
	}
	if p.AveragePrice == nil || *p.AveragePrice != 306.29 {
		t.Errorf("AveragePrice = %v, want 306.29", p.AveragePrice)
	}
	if p.TypeId != 32772 {
		t.Errorf("TypeId = %d, want 32772", p.TypeId)
	}
}

func TestMarketPrices_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketPrices(context.Background()); err == nil {
		t.Fatal("MarketPrices returned nil error, want an error for HTTP 500")
	}
}

func TestMarketHistory_SendsRequiredTypeIdAndParsesResponse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"average":5.5,"date":"2026-07-20","highest":6.0,"lowest":5.0,"order_count":10,"volume":1000}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	history, err := client.MarketHistory(context.Background(), 10000002, 34)
	if err != nil {
		t.Fatalf("MarketHistory returned error: %v", err)
	}
	if gotQuery != "type_id=34" {
		t.Errorf("query = %q, want \"type_id=34\" (typeId is a required query param)", gotQuery)
	}
	if len(history) != 1 {
		t.Fatalf("history = %+v, want one entry", history)
	}
	h := history[0]
	if h.Average != 5.5 || h.Date != "2026-07-20" || h.Highest != 6.0 || h.Lowest != 5.0 || h.OrderCount != 10 || h.Volume != 1000 {
		t.Errorf("history[0] = %+v, want Average=5.5 Date=2026-07-20 Highest=6.0 Lowest=5.0 OrderCount=10 Volume=1000", h)
	}
}

func TestMarketHistory_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketHistory(context.Background(), 10000002, 34); err == nil {
		t.Fatal("MarketHistory returned nil error, want an error for HTTP 404")
	}
}

func TestMarketOrders_SendsRequiredOrderTypeAndOptionalFilters(t *testing.T) {
	tests := []struct {
		name      string
		orderType string
		typeID    int64
		page      int32
		wantQuery string
	}{
		{name: "all, no type filter, no page", orderType: "all", typeID: 0, page: 0, wantQuery: "order_type=all"},
		{name: "buy with type filter and page", orderType: "buy", typeID: 34, page: 2, wantQuery: "order_type=buy&page=2&type_id=34"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Pages", "5")
				w.Write([]byte(`[{
					"duration": 90,
					"is_buy_order": true,
					"issued": "2026-07-20T00:00:00Z",
					"location_id": 60003760,
					"min_volume": 1,
					"order_id": 555,
					"price": 5.55,
					"range": "region",
					"system_id": 30000142,
					"type_id": 34,
					"volume_remain": 500,
					"volume_total": 1000
				}]`))
			}))
			defer srv.Close()

			client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

			orders, totalPages, err := client.MarketOrders(context.Background(), 10000002, tt.orderType, tt.typeID, tt.page)
			if err != nil {
				t.Fatalf("MarketOrders returned error: %v", err)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tt.wantQuery)
			}
			if totalPages != 5 {
				t.Errorf("totalPages = %d, want 5", totalPages)
			}
			if len(orders) != 1 {
				t.Fatalf("orders = %+v, want one entry", orders)
			}
			o := orders[0]
			if o.SystemId != 30000142 || o.OrderId != 555 || !o.IsBuyOrder {
				t.Errorf("orders[0] = %+v, want SystemId=30000142 OrderId=555 IsBuyOrder=true", o)
			}
		})
	}
}

func TestMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.MarketOrders(context.Background(), 10000002, "all", 0, 0); err == nil {
		t.Fatal("MarketOrders returned nil error, want an error for HTTP 404")
	}
}

func TestMarketTypes_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "7")
		w.Write([]byte(`[34, 35, 36]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	types, totalPages, err := client.MarketTypes(context.Background(), 10000002, 3)
	if err != nil {
		t.Fatalf("MarketTypes returned error: %v", err)
	}
	if gotQuery != "page=3" {
		t.Errorf("query = %q, want \"page=3\"", gotQuery)
	}
	if totalPages != 7 {
		t.Errorf("totalPages = %d, want 7", totalPages)
	}
	if len(types) != 3 || types[0] != 34 || types[1] != 35 || types[2] != 36 {
		t.Errorf("types = %+v, want [34 35 36]", types)
	}
}

func TestMarketTypes_PageOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.MarketTypes(context.Background(), 10000002, 0); err != nil {
		t.Fatalf("MarketTypes returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
}

func TestMarketTypes_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.MarketTypes(context.Background(), 10000002, 0); err == nil {
		t.Fatal("MarketTypes returned nil error, want an error for HTTP 404")
	}
}

func TestStructureMarketOrders_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{
			"duration": 90,
			"is_buy_order": false,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 1000000000001,
			"min_volume": 1,
			"order_id": 777,
			"price": 12.5,
			"range": "station",
			"type_id": 34,
			"volume_remain": 250,
			"volume_total": 1000
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	orders, totalPages, err := client.StructureMarketOrders(context.Background(), 1000000000001, 1)
	if err != nil {
		t.Fatalf("StructureMarketOrders returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want \"page=1\"", gotQuery)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %+v, want one entry", orders)
	}
	o := orders[0]
	if o.OrderId != 777 || o.LocationId != 1000000000001 || o.IsBuyOrder {
		t.Errorf("orders[0] = %+v, want OrderId=777 LocationId=1000000000001 IsBuyOrder=false", o)
	}
}

func TestStructureMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.StructureMarketOrders(context.Background(), 1000000000001, 0); err == nil {
		t.Fatal("StructureMarketOrders returned nil error, want an error for HTTP 403 (no docking access)")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterMarketOrders|TestCorporationMarketOrders|TestMarketGroups|TestMarketGroup|TestMarketPrices|TestMarketHistory|TestMarketOrders|TestMarketTypes|TestStructureMarketOrders' -v`
Expected: FAIL — `esi.Client` has none of these methods yet.

- [ ] **Step 3: Write the implementation**

Create `esi/market.go`:

```go
// Package esi's market.go covers all 11 of "Market"'s endpoints: character/corp own market
// orders and order history (character/corp endpoints are "my own character"/corp-scoped exactly
// like esi/industry.go and esi/contracts.go), plus public market-group listing/detail,
// adjusted/average prices, per-region historical stats, per-region live orders and active type
// IDs, and one endpoint that looks public but isn't -- GetMarketsStructuresStructureId requires
// OAuth2 scope esi-markets.structure_markets.v1 (verified in esi/api/openapi.yaml), so it's
// modeled as "my own character" auth exactly like CharacterFittings, not NewPublicClient.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterMarketOrders returns the client's own character's open market orders. Not paginated --
// verified from source: ApiGetCharactersCharacterIdOrdersRequest has no Page() setter, unlike its
// History sibling below.
func (c *Client) CharacterMarketOrders(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
	result, resp, err := c.api.MarketAPI.GetCharactersCharacterIdOrders(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character market orders: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMarketOrderHistory returns the client's own character's cancelled/expired market
// orders from the last 90 days. page=0 omits the page query parameter.
func (c *Client) CharacterMarketOrderHistory(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
	req := c.api.MarketAPI.GetCharactersCharacterIdOrdersHistory(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character market order history: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMarketOrders returns corporationID's open market orders. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise. page=0 omits the page query
// parameter.
func (c *Client) CorporationMarketOrders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
	req := c.api.MarketAPI.GetCorporationsCorporationIdOrders(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation market orders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMarketOrderHistory returns corporationID's cancelled/expired market orders from the
// last 90 days. Same corp-role enforcement as CorporationMarketOrders. page=0 omits the page query
// parameter.
func (c *Client) CorporationMarketOrderHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
	req := c.api.MarketAPI.GetCorporationsCorporationIdOrdersHistory(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation market order history: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// MarketGroups returns every market group ID in the game. Public, not paginated.
func (c *Client) MarketGroups(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsGroups(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market groups: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketGroup returns detail for one market group. Public, not paginated.
func (c *Client) MarketGroup(ctx context.Context, marketGroupID int64) (*esiapi.MarketsGroupsMarketGroupIdGet, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsGroupsMarketGroupId(ctx, marketGroupID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market group: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketPrices returns ESI's adjusted/average price for every priced type in the game. Public,
// not paginated.
func (c *Client) MarketPrices(ctx context.Context) ([]esiapi.MarketsPricesGetInner, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsPrices(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market prices: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketHistory returns daily historical market statistics for one type in one region. Public,
// not paginated. typeID is a required query parameter -- verified from source:
// GetMarketsRegionIdHistoryExecute explicitly errors ("typeId is required and must be specified")
// if it's left unset, unlike the optional TypeId filter on MarketOrders below.
func (c *Client) MarketHistory(ctx context.Context, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsRegionIdHistory(ctx, regionID).TypeId(typeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market history: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketOrders returns live orders in a region, optionally filtered to one type. Public, IS
// paginated (page=0 omits the page query parameter). orderType is required -- verified from
// source: GetMarketsRegionIdOrdersExecute explicitly errors ("orderType is required and must be
// specified") if left unset, even though ESI's own spec lists a default ("all"); callers must
// always pass one of "buy", "sell", or "all" explicitly. typeID=0 omits the optional type_id
// filter, returning orders for every type in the region.
func (c *Client) MarketOrders(ctx context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
	req := c.api.MarketAPI.GetMarketsRegionIdOrders(ctx, regionID).OrderType(orderType)
	if typeID > 0 {
		req = req.TypeId(typeID)
	}
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get market orders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// MarketTypes returns the type IDs with at least one active order in a region, for efficient
// market indexing. Public, IS paginated. page=0 omits the page query parameter.
func (c *Client) MarketTypes(ctx context.Context, regionID int64, page int32) ([]int64, int32, error) {
	req := c.api.MarketAPI.GetMarketsRegionIdTypes(ctx, regionID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get market types: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// StructureMarketOrders returns live orders in a player structure's market. Despite resembling
// MarketOrders (region orders), this is NOT public -- verified in esi/api/openapi.yaml: it
// requires OAuth2 scope esi-markets.structure_markets.v1, i.e. the querying character must have
// docking access to the structure. Modeled as "my own character" auth, same pattern as
// CharacterFittings -- the caller-supplied structureID is the resource, c.characterID
// authenticates. IS paginated; page=0 omits the page query parameter.
func (c *Client) StructureMarketOrders(ctx context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
	req := c.api.MarketAPI.GetMarketsStructuresStructureId(ctx, structureID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get structure market orders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestCharacterMarketOrders|TestCorporationMarketOrders|TestMarketGroups|TestMarketGroup|TestMarketPrices|TestMarketHistory|TestMarketOrders|TestMarketTypes|TestStructureMarketOrders' -v`
Expected: PASS (24 tests).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./...`
Expected: all clean, no output from `gofmt -l`.

```bash
git add esi/market.go esi/market_test.go
git commit -m "feat: add esi/market.go facade (character/corp orders, groups, prices, region/structure orders)"
```

---

### Task 2: `mcp/market.go` — MCP tool layer + registration + docs

**Files:**
- Create: `mcp/market.go`
- Test: `mcp/market_internal_test.go`
- Modify: `mcp/server.go` (register 11 tools after the `delete_character_mail_label` block; update package doc comment)
- Modify: `mcp/server_test.go` (expected-tools list: 146 → 157)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Market row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (tool count "one hundred forty-six" → "one hundred fifty-seven", append 11 tool names)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: `esi.NewClient(ctx, clientID, characterID, userAgent) (*esi.Client, error)`;
  `esi.NewPublicClient(userAgent) *esi.Client`; all 11 `(*esi.Client)` methods from Task 1;
  `characterExists`/`ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent(version string) string`
  (`mcp/sso`); `config.Store` (`mcp/config`).
- Produces: 11 tool names — `list_character_market_orders`, `list_character_market_order_history`,
  `list_corporation_market_orders`, `list_corporation_market_order_history`, `list_market_groups`,
  `get_market_group`, `list_market_prices`, `get_market_history`, `list_market_orders`,
  `list_market_types`, `list_structure_market_orders` — registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `mcp/market_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// --- list_character_market_orders ---

func TestListCharacterMarketOrdersHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = orig })
	issued := time.Unix(1000, 0)
	escrow := 100.5
	isBuyOrder := true
	minVolume := int64(1)
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return []esiapi.CharactersCharacterIdOrdersGetInner{{
			Duration: 90, Escrow: &escrow, IsBuyOrder: &isBuyOrder, IsCorporation: false,
			Issued: issued, LocationId: 60003760, MinVolume: &minVolume, OrderId: 123456,
			Price: 5.55, Range: "region", RegionId: 10000002, TypeId: 34,
			VolumeRemain: 500, VolumeTotal: 1000,
		}}, nil
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || o.Escrow == nil || *o.Escrow != escrow || o.IsBuyOrder == nil || !*o.IsBuyOrder ||
		o.IsCorporation == nil || *o.IsCorporation || !o.Issued.Equal(issued) || o.LocationID != 60003760 ||
		o.MinVolume == nil || *o.MinVolume != minVolume || o.OrderID != 123456 || o.Price != 5.55 ||
		o.Range != "region" || o.RegionID != 10000002 || o.TypeID != 34 || o.VolumeRemain != 500 ||
		o.VolumeTotal != 1000 || o.State != "" || o.IssuedBy != nil || o.WalletDivision != nil {
		t.Errorf("order = %+v, want Duration=90 Escrow=%v IsBuyOrder=true IsCorporation=false Issued=%v LocationID=60003760 MinVolume=%v OrderID=123456 Price=5.55 Range=region RegionID=10000002 TypeID=34 VolumeRemain=500 VolumeTotal=1000 State=\"\" IssuedBy=nil WalletDivision=nil",
			o, escrow, issued, minVolume)
	}
}

func TestListCharacterMarketOrdersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = orig })
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		t.Fatal("listCharacterMarketOrders should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_character_market_order_history ---

func TestListCharacterMarketOrderHistoryHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMarketOrderHistory
	t.Cleanup(func() { listCharacterMarketOrderHistory = orig })
	issued := time.Unix(2000, 0)
	listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdOrdersHistoryGetInner{{
			Duration: 90, IsCorporation: true, Issued: issued, LocationId: 60003760,
			OrderId: 654321, Price: 5.55, Range: "region", RegionId: 10000002,
			State: "expired", TypeId: 34, VolumeRemain: 0, VolumeTotal: 1000,
		}}, 3, nil
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrderHistoryInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.Page != 1 || out.TotalPages != 3 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 Page=1 TotalPages=3 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || o.IsCorporation == nil || !*o.IsCorporation || !o.Issued.Equal(issued) ||
		o.LocationID != 60003760 || o.OrderID != 654321 || o.Price != 5.55 || o.Range != "region" ||
		o.RegionID != 10000002 || o.State != "expired" || o.TypeID != 34 || o.VolumeRemain != 0 ||
		o.VolumeTotal != 1000 || o.Escrow != nil || o.IsBuyOrder != nil || o.MinVolume != nil ||
		o.IssuedBy != nil || o.WalletDivision != nil {
		t.Errorf("order = %+v, want Duration=90 IsCorporation=true Issued=%v LocationID=60003760 OrderID=654321 Price=5.55 Range=region RegionID=10000002 State=expired TypeID=34 VolumeRemain=0 VolumeTotal=1000",
			o, issued)
	}
}

func TestListCharacterMarketOrderHistoryHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterMarketOrderHistory
	t.Cleanup(func() { listCharacterMarketOrderHistory = orig })
	listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		t.Fatal("listCharacterMarketOrderHistory should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterMarketOrderHistoryInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_corporation_market_orders ---

func TestListCorporationMarketOrdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMarketOrders
	t.Cleanup(func() { listCorporationMarketOrders = orig })
	issued := time.Unix(3000, 0)
	listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersGetInner{{
			Duration: 90, Issued: issued, IssuedBy: 998877, LocationId: 60003760,
			OrderId: 111222, Price: 5.55, Range: "region", RegionId: 10000002,
			TypeId: 34, VolumeRemain: 500, VolumeTotal: 1000, WalletDivision: 1,
		}}, 2, nil
	}

	handler := listCorporationMarketOrdersHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrdersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || !o.Issued.Equal(issued) || o.IssuedBy == nil || *o.IssuedBy != 998877 ||
		o.LocationID != 60003760 || o.OrderID != 111222 || o.Price != 5.55 || o.Range != "region" ||
		o.RegionID != 10000002 || o.TypeID != 34 || o.VolumeRemain != 500 || o.VolumeTotal != 1000 ||
		o.WalletDivision == nil || *o.WalletDivision != 1 ||
		o.IsCorporation != nil || o.State != "" || o.Escrow != nil || o.IsBuyOrder != nil || o.MinVolume != nil {
		t.Errorf("order = %+v, want Duration=90 Issued=%v IssuedBy=998877 LocationID=60003760 OrderID=111222 Price=5.55 Range=region RegionID=10000002 TypeID=34 VolumeRemain=500 VolumeTotal=1000 WalletDivision=1",
			o, issued)
	}
}

func TestListCorporationMarketOrdersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMarketOrders
	t.Cleanup(func() { listCorporationMarketOrders = orig })
	listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		t.Fatal("listCorporationMarketOrders should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMarketOrdersHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrdersInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_corporation_market_order_history ---

func TestListCorporationMarketOrderHistoryHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMarketOrderHistory
	t.Cleanup(func() { listCorporationMarketOrderHistory = orig })
	issued := time.Unix(4000, 0)
	listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersHistoryGetInner{{
			Duration: 90, Issued: issued, IssuedBy: nil, LocationId: 60003760,
			OrderId: 333444, Price: 5.55, Range: "region", RegionId: 10000002,
			State: "cancelled", TypeId: 34, VolumeRemain: 0, VolumeTotal: 1000, WalletDivision: 2,
		}}, 1, nil
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 1 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=1 with one order", out)
	}
	o := out.Orders[0]
	// IssuedBy is optional (*int64) for corp order history, unlike the required int64 on
	// list_corporation_market_orders -- this fixture leaves it nil to confirm the handler
	// passes that nil through rather than synthesizing a zero-value pointer.
	if o.Duration != 90 || !o.Issued.Equal(issued) || o.IssuedBy != nil ||
		o.LocationID != 60003760 || o.OrderID != 333444 || o.Price != 5.55 || o.Range != "region" ||
		o.RegionID != 10000002 || o.State != "cancelled" || o.TypeID != 34 || o.VolumeRemain != 0 ||
		o.VolumeTotal != 1000 || o.WalletDivision == nil || *o.WalletDivision != 2 ||
		o.IsCorporation != nil || o.Escrow != nil || o.IsBuyOrder != nil || o.MinVolume != nil {
		t.Errorf("order = %+v, want Duration=90 Issued=%v IssuedBy=nil LocationID=60003760 OrderID=333444 Price=5.55 Range=region RegionID=10000002 State=cancelled TypeID=34 VolumeRemain=0 VolumeTotal=1000 WalletDivision=2",
			o, issued)
	}
}

func TestListCorporationMarketOrderHistoryHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMarketOrderHistory
	t.Cleanup(func() { listCorporationMarketOrderHistory = orig })
	listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		t.Fatal("listCorporationMarketOrderHistory should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_market_groups (public) ---

func TestListMarketGroupsHandler_MapsFields(t *testing.T) {
	orig := listMarketGroups
	t.Cleanup(func() { listMarketGroups = orig })
	listMarketGroups = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{1, 2, 3}, nil
	}

	handler := listMarketGroupsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListMarketGroupsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.MarketGroupIDs) != 3 || out.MarketGroupIDs[0] != 1 || out.MarketGroupIDs[1] != 2 || out.MarketGroupIDs[2] != 3 {
		t.Errorf("out.MarketGroupIDs = %+v, want [1 2 3]", out.MarketGroupIDs)
	}
}

// --- get_market_group (public) ---

func TestGetMarketGroupHandler_MapsFields(t *testing.T) {
	orig := getMarketGroup
	t.Cleanup(func() { getMarketGroup = orig })
	parentGroupID := int64(1)
	getMarketGroup = func(ctx context.Context, userAgent string, marketGroupID int64) (GetMarketGroupOutput, error) {
		return GetMarketGroupOutput{
			MarketGroupID: marketGroupID, Name: "Ammo & Charges", Description: "Ammo & Charges",
			ParentGroupID: &parentGroupID, Types: []int64{10, 11},
		}, nil
	}

	handler := getMarketGroupHandler("v1")
	_, out, err := handler(context.Background(), nil, GetMarketGroupInput{MarketGroupID: 9})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.MarketGroupID != 9 || out.Name != "Ammo & Charges" || out.Description != "Ammo & Charges" ||
		out.ParentGroupID == nil || *out.ParentGroupID != 1 || len(out.Types) != 2 || out.Types[0] != 10 || out.Types[1] != 11 {
		t.Errorf("out = %+v, want MarketGroupID=9 Name/Description=\"Ammo & Charges\" ParentGroupID=1 Types=[10 11]", out)
	}
}

// --- list_market_prices (public) ---

func TestListMarketPricesHandler_MapsFields(t *testing.T) {
	orig := listMarketPrices
	t.Cleanup(func() { listMarketPrices = orig })
	adjustedPrice := 306.11
	averagePrice := 306.29
	listMarketPrices = func(ctx context.Context, userAgent string) ([]esiapi.MarketsPricesGetInner, error) {
		return []esiapi.MarketsPricesGetInner{{
			AdjustedPrice: &adjustedPrice, AveragePrice: &averagePrice, TypeId: 32772,
		}}, nil
	}

	handler := listMarketPricesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListMarketPricesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Prices) != 1 {
		t.Fatalf("out = %+v, want one price", out)
	}
	p := out.Prices[0]
	if p.TypeID != 32772 || p.AdjustedPrice == nil || *p.AdjustedPrice != adjustedPrice ||
		p.AveragePrice == nil || *p.AveragePrice != averagePrice {
		t.Errorf("price = %+v, want TypeID=32772 AdjustedPrice=%v AveragePrice=%v", p, adjustedPrice, averagePrice)
	}
}

// --- get_market_history (public) ---

func TestGetMarketHistoryHandler_MapsFields(t *testing.T) {
	orig := getMarketHistory
	t.Cleanup(func() { getMarketHistory = orig })
	getMarketHistory = func(ctx context.Context, userAgent string, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
		return []esiapi.MarketsRegionIdHistoryGetInner{{
			Average: 5.5, Date: "2026-07-20", Highest: 6.0, Lowest: 5.0, OrderCount: 10, Volume: 1000,
		}}, nil
	}

	handler := getMarketHistoryHandler("v1")
	_, out, err := handler(context.Background(), nil, GetMarketHistoryInput{RegionID: 10000002, TypeID: 34})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.TypeID != 34 || len(out.History) != 1 {
		t.Fatalf("out = %+v, want RegionID=10000002 TypeID=34 with one history entry", out)
	}
	h := out.History[0]
	if h.Average != 5.5 || h.Date != "2026-07-20" || h.Highest != 6.0 || h.Lowest != 5.0 || h.OrderCount != 10 || h.Volume != 1000 {
		t.Errorf("history[0] = %+v, want Average=5.5 Date=2026-07-20 Highest=6.0 Lowest=5.0 OrderCount=10 Volume=1000", h)
	}
}

// --- list_market_orders (public, paginated) ---

func TestListMarketOrdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	orig := listMarketOrders
	t.Cleanup(func() { listMarketOrders = orig })
	issued := time.Unix(5000, 0)
	var gotRegionID, gotTypeID int64
	var gotOrderType string
	var gotPage int32
	listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		gotRegionID, gotOrderType, gotTypeID, gotPage = regionID, orderType, typeID, page
		return []esiapi.MarketsRegionIdOrdersGetInner{{
			Duration: 90, IsBuyOrder: true, Issued: issued, LocationId: 60003760,
			MinVolume: 1, OrderId: 555, Price: 5.55, Range: "region", SystemId: 30000142,
			TypeId: 34, VolumeRemain: 500, VolumeTotal: 1000,
		}}, 5, nil
	}

	handler := listMarketOrdersHandler("v1")
	_, out, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002, OrderType: "buy", TypeID: 34})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotRegionID != 10000002 || gotOrderType != "buy" || gotTypeID != 34 || gotPage != 0 {
		t.Errorf("captured args = regionID=%d orderType=%q typeID=%d page=%d, want 10000002/buy/34/0", gotRegionID, gotOrderType, gotTypeID, gotPage)
	}
	if out.RegionID != 10000002 || out.OrderType != "buy" || out.Page != 1 || out.TotalPages != 5 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want RegionID=10000002 OrderType=buy Page=1 TotalPages=5 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || !o.IsBuyOrder || !o.Issued.Equal(issued) || o.LocationID != 60003760 ||
		o.MinVolume != 1 || o.OrderID != 555 || o.Price != 5.55 || o.Range != "region" ||
		o.SystemID != 30000142 || o.TypeID != 34 || o.VolumeRemain != 500 || o.VolumeTotal != 1000 {
		t.Errorf("order = %+v, want Duration=90 IsBuyOrder=true Issued=%v LocationID=60003760 MinVolume=1 OrderID=555 Price=5.55 Range=region SystemID=30000142 TypeID=34 VolumeRemain=500 VolumeTotal=1000",
			o, issued)
	}
}

// --- list_market_types (public, paginated) ---

func TestListMarketTypesHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	orig := listMarketTypes
	t.Cleanup(func() { listMarketTypes = orig })
	listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{34, 35, 36}, 7, nil
	}

	handler := listMarketTypesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListMarketTypesInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.Page != 1 || out.TotalPages != 7 || len(out.TypeIDs) != 3 ||
		out.TypeIDs[0] != 34 || out.TypeIDs[1] != 35 || out.TypeIDs[2] != 36 {
		t.Errorf("out = %+v, want RegionID=10000002 Page=1 TotalPages=7 TypeIDs=[34 35 36]", out)
	}
}

// --- list_structure_market_orders (private, paginated) ---

func TestListStructureMarketOrdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = orig })
	issued := time.Unix(6000, 0)
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return []esiapi.MarketsStructuresStructureIdGetInner{{
			Duration: 90, IsBuyOrder: false, Issued: issued, LocationId: 1000000000001,
			MinVolume: 1, OrderId: 777, Price: 12.5, Range: "station",
			TypeId: 34, VolumeRemain: 250, VolumeTotal: 1000,
		}}, 2, nil
	}

	handler := listStructureMarketOrdersHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 7, StructureID: 1000000000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.StructureID != 1000000000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 StructureID=1000000000001 Page=1 TotalPages=2 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || o.IsBuyOrder || !o.Issued.Equal(issued) || o.LocationID != 1000000000001 ||
		o.MinVolume != 1 || o.OrderID != 777 || o.Price != 12.5 || o.Range != "station" ||
		o.SystemID != 0 || o.TypeID != 34 || o.VolumeRemain != 250 || o.VolumeTotal != 1000 {
		t.Errorf("order = %+v, want Duration=90 IsBuyOrder=false Issued=%v LocationID=1000000000001 MinVolume=1 OrderID=777 Price=12.5 Range=station SystemID=0 TypeID=34 VolumeRemain=250 VolumeTotal=1000",
			o, issued)
	}
}

func TestListStructureMarketOrdersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = orig })
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		t.Fatal("listStructureMarketOrders should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listStructureMarketOrdersHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 999, StructureID: 1000000000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListCharacterMarketOrdersHandler|TestListCharacterMarketOrderHistoryHandler|TestListCorporationMarketOrdersHandler|TestListCorporationMarketOrderHistoryHandler|TestListMarketGroupsHandler|TestGetMarketGroupHandler|TestListMarketPricesHandler|TestGetMarketHistoryHandler|TestListMarketOrdersHandler|TestListMarketTypesHandler|TestListStructureMarketOrdersHandler' -v`
Expected: FAIL — undefined symbols (none of these handlers/types exist yet).

- [ ] **Step 3: Write the implementation**

Create `mcp/market.go`:

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// MarketOrder is shared by list_character_market_orders, list_character_market_order_history,
// list_corporation_market_orders, and list_corporation_market_order_history -- the four
// underlying ESI models share 13 fields exactly. IsCorporation is populated only for the two
// character-scoped variants (character open orders + character order history both carry it --
// verified from source, not just the open-orders one); State is populated only for the two
// order-history variants; IssuedBy/WalletDivision are populated only for the two
// corporation-scoped variants. Only the applicable fields are populated -- same "shared model,
// zero-value/nil signals not-applicable" precedent as mcp/industry.go's IndustryJob.
type MarketOrder struct {
	Duration       int64     `json:"duration"`
	Escrow         *float64  `json:"escrow,omitempty"`
	IsBuyOrder     *bool     `json:"isBuyOrder,omitempty"`
	Issued         time.Time `json:"issued"`
	LocationID     int64     `json:"locationId"`
	MinVolume      *int64    `json:"minVolume,omitempty"`
	OrderID        int64     `json:"orderId"`
	Price          float64   `json:"price"`
	Range          string    `json:"range"`
	RegionID       int64     `json:"regionId"`
	TypeID         int64     `json:"typeId"`
	VolumeRemain   int64     `json:"volumeRemain"`
	VolumeTotal    int64     `json:"volumeTotal"`
	IsCorporation  *bool     `json:"isCorporation,omitempty"`
	State          string    `json:"state,omitempty"`
	IssuedBy       *int64    `json:"issuedBy,omitempty"`
	WalletDivision *int64    `json:"walletDivision,omitempty"`
}

func toMarketOrderFromCharacterOrder(o esiapi.CharactersCharacterIdOrdersGetInner) MarketOrder {
	isCorp := o.IsCorporation
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, IsCorporation: &isCorp,
	}
}

func toMarketOrderFromCharacterOrderHistory(o esiapi.CharactersCharacterIdOrdersHistoryGetInner) MarketOrder {
	isCorp := o.IsCorporation
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, IsCorporation: &isCorp, State: o.State,
	}
}

func toMarketOrderFromCorporationOrder(o esiapi.CorporationsCorporationIdOrdersGetInner) MarketOrder {
	issuedBy := o.IssuedBy
	walletDivision := o.WalletDivision
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, IssuedBy: &issuedBy, WalletDivision: &walletDivision,
	}
}

func toMarketOrderFromCorporationOrderHistory(o esiapi.CorporationsCorporationIdOrdersHistoryGetInner) MarketOrder {
	walletDivision := o.WalletDivision
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, State: o.State, IssuedBy: o.IssuedBy, WalletDivision: &walletDivision,
	}
}

// MarketOrderBookEntry is shared by list_market_orders (region order book) and
// list_structure_market_orders (structure order book) -- the two underlying ESI models are
// identical except SystemID, present only for region orders (a structure pins its own system, so
// ESI doesn't repeat it per-order there). Named for the data shape (an anonymous order-book
// entry with no owner/issuer identity fields), not for auth: list_structure_market_orders is
// privately scoped despite sharing this shape -- see its own doc comment below.
type MarketOrderBookEntry struct {
	Duration     int64     `json:"duration"`
	IsBuyOrder   bool      `json:"isBuyOrder"`
	Issued       time.Time `json:"issued"`
	LocationID   int64     `json:"locationId"`
	MinVolume    int64     `json:"minVolume"`
	OrderID      int64     `json:"orderId"`
	Price        float64   `json:"price"`
	Range        string    `json:"range"`
	SystemID     int64     `json:"systemId,omitempty"`
	TypeID       int64     `json:"typeId"`
	VolumeRemain int64     `json:"volumeRemain"`
	VolumeTotal  int64     `json:"volumeTotal"`
}

func toMarketOrderBookEntryFromRegion(o esiapi.MarketsRegionIdOrdersGetInner) MarketOrderBookEntry {
	return MarketOrderBookEntry{
		Duration: o.Duration, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued, LocationID: o.LocationId,
		MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price, Range: o.Range,
		SystemID: o.SystemId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain, VolumeTotal: o.VolumeTotal,
	}
}

func toMarketOrderBookEntryFromStructure(o esiapi.MarketsStructuresStructureIdGetInner) MarketOrderBookEntry {
	return MarketOrderBookEntry{
		Duration: o.Duration, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued, LocationID: o.LocationId,
		MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price, Range: o.Range,
		TypeID: o.TypeId, VolumeRemain: o.VolumeRemain, VolumeTotal: o.VolumeTotal,
	}
}

// --- list_character_market_orders (private) ---

var listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterMarketOrders(ctx)
}

type ListCharacterMarketOrdersInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterMarketOrdersOutput struct {
	CharacterID int32         `json:"characterId"`
	Orders      []MarketOrder `json:"orders"`
}

func listCharacterMarketOrdersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterMarketOrdersInput, ListCharacterMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMarketOrdersInput) (*mcpsdk.CallToolResult, ListCharacterMarketOrdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMarketOrdersOutput{}, err
		}
		if !found {
			return nil, ListCharacterMarketOrdersOutput{}, fmt.Errorf("list character market orders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		orders, err := listCharacterMarketOrders(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterMarketOrdersOutput{}, fmt.Errorf("list character market orders: %w", err)
		}
		out := make([]MarketOrder, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderFromCharacterOrder(o)
		}
		return nil, ListCharacterMarketOrdersOutput{CharacterID: in.CharacterID, Orders: out}, nil
	}
}

// --- list_character_market_order_history (private, paginated) ---

var listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterMarketOrderHistory(ctx, page)
}

type ListCharacterMarketOrderHistoryInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterMarketOrderHistoryOutput struct {
	CharacterID int32         `json:"characterId"`
	Page        int32         `json:"page"`
	TotalPages  int32         `json:"totalPages"`
	Orders      []MarketOrder `json:"orders"`
}

func listCharacterMarketOrderHistoryHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterMarketOrderHistoryInput, ListCharacterMarketOrderHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMarketOrderHistoryInput) (*mcpsdk.CallToolResult, ListCharacterMarketOrderHistoryOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMarketOrderHistoryOutput{}, err
		}
		if !found {
			return nil, ListCharacterMarketOrderHistoryOutput{}, fmt.Errorf("list character market order history for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		history, totalPages, err := listCharacterMarketOrderHistory(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListCharacterMarketOrderHistoryOutput{}, fmt.Errorf("list character market order history: %w", err)
		}
		out := make([]MarketOrder, len(history))
		for i, o := range history {
			out[i] = toMarketOrderFromCharacterOrderHistory(o)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCharacterMarketOrderHistoryOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_corporation_market_orders (private, corp-scoped, paginated) ---

var listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMarketOrders(ctx, corporationID, page)
}

type ListCorporationMarketOrdersInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMarketOrdersOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Orders        []MarketOrder `json:"orders"`
}

func listCorporationMarketOrdersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMarketOrdersInput, ListCorporationMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMarketOrdersInput) (*mcpsdk.CallToolResult, ListCorporationMarketOrdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMarketOrdersOutput{}, err
		}
		if !found {
			return nil, ListCorporationMarketOrdersOutput{}, fmt.Errorf("list corporation market orders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		orders, totalPages, err := listCorporationMarketOrders(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMarketOrdersOutput{}, fmt.Errorf("list corporation market orders: %w", err)
		}
		out := make([]MarketOrder, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderFromCorporationOrder(o)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationMarketOrdersOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_corporation_market_order_history (private, corp-scoped, paginated) ---

var listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMarketOrderHistory(ctx, corporationID, page)
}

type ListCorporationMarketOrderHistoryInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMarketOrderHistoryOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Orders        []MarketOrder `json:"orders"`
}

func listCorporationMarketOrderHistoryHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMarketOrderHistoryInput, ListCorporationMarketOrderHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMarketOrderHistoryInput) (*mcpsdk.CallToolResult, ListCorporationMarketOrderHistoryOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMarketOrderHistoryOutput{}, err
		}
		if !found {
			return nil, ListCorporationMarketOrderHistoryOutput{}, fmt.Errorf("list corporation market order history for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		history, totalPages, err := listCorporationMarketOrderHistory(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMarketOrderHistoryOutput{}, fmt.Errorf("list corporation market order history: %w", err)
		}
		out := make([]MarketOrder, len(history))
		for i, o := range history {
			out[i] = toMarketOrderFromCorporationOrderHistory(o)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationMarketOrderHistoryOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_market_groups (public, no input) ---

var listMarketGroups = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketGroups(ctx)
}

type ListMarketGroupsInput struct{}
type ListMarketGroupsOutput struct {
	MarketGroupIDs []int64 `json:"marketGroupIds"`
}

func listMarketGroupsHandler(version string) mcpsdk.ToolHandlerFor[ListMarketGroupsInput, ListMarketGroupsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketGroupsInput) (*mcpsdk.CallToolResult, ListMarketGroupsOutput, error) {
		ids, err := listMarketGroups(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListMarketGroupsOutput{}, fmt.Errorf("list market groups: %w", err)
		}
		return nil, ListMarketGroupsOutput{MarketGroupIDs: ids}, nil
	}
}

// --- get_market_group (public) ---

type GetMarketGroupOutput struct {
	MarketGroupID int64   `json:"marketGroupId"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	ParentGroupID *int64  `json:"parentGroupId,omitempty"`
	Types         []int64 `json:"types"`
}

var getMarketGroup = func(ctx context.Context, userAgent string, marketGroupID int64) (GetMarketGroupOutput, error) {
	client := esi.NewPublicClient(userAgent)
	group, err := client.MarketGroup(ctx, marketGroupID)
	if err != nil {
		return GetMarketGroupOutput{}, err
	}
	return GetMarketGroupOutput{
		MarketGroupID: group.MarketGroupId, Name: group.Name, Description: group.Description,
		ParentGroupID: group.ParentGroupId, Types: group.Types,
	}, nil
}

type GetMarketGroupInput struct {
	MarketGroupID int64 `json:"marketGroupId" jsonschema:"the market group ID to look up"`
}

func getMarketGroupHandler(version string) mcpsdk.ToolHandlerFor[GetMarketGroupInput, GetMarketGroupOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMarketGroupInput) (*mcpsdk.CallToolResult, GetMarketGroupOutput, error) {
		out, err := getMarketGroup(ctx, sso.UserAgent(version), in.MarketGroupID)
		if err != nil {
			return nil, GetMarketGroupOutput{}, fmt.Errorf("get market group: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_market_prices (public, no input) ---

type MarketPrice struct {
	TypeID        int64    `json:"typeId"`
	AdjustedPrice *float64 `json:"adjustedPrice,omitempty"`
	AveragePrice  *float64 `json:"averagePrice,omitempty"`
}

var listMarketPrices = func(ctx context.Context, userAgent string) ([]esiapi.MarketsPricesGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketPrices(ctx)
}

type ListMarketPricesInput struct{}
type ListMarketPricesOutput struct {
	Prices []MarketPrice `json:"prices"`
}

func listMarketPricesHandler(version string) mcpsdk.ToolHandlerFor[ListMarketPricesInput, ListMarketPricesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketPricesInput) (*mcpsdk.CallToolResult, ListMarketPricesOutput, error) {
		prices, err := listMarketPrices(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListMarketPricesOutput{}, fmt.Errorf("list market prices: %w", err)
		}
		out := make([]MarketPrice, len(prices))
		for i, p := range prices {
			out[i] = MarketPrice{TypeID: p.TypeId, AdjustedPrice: p.AdjustedPrice, AveragePrice: p.AveragePrice}
		}
		return nil, ListMarketPricesOutput{Prices: out}, nil
	}
}

// --- get_market_history (public) ---

type MarketHistoryEntry struct {
	Average    float64 `json:"average"`
	Date       string  `json:"date"`
	Highest    float64 `json:"highest"`
	Lowest     float64 `json:"lowest"`
	OrderCount int64   `json:"orderCount"`
	Volume     int64   `json:"volume"`
}

var getMarketHistory = func(ctx context.Context, userAgent string, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketHistory(ctx, regionID, typeID)
}

type GetMarketHistoryInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
	TypeID   int64 `json:"typeId" jsonschema:"the item type ID to get historical daily stats for"`
}
type GetMarketHistoryOutput struct {
	RegionID int64                `json:"regionId"`
	TypeID   int64                `json:"typeId"`
	History  []MarketHistoryEntry `json:"history"`
}

func getMarketHistoryHandler(version string) mcpsdk.ToolHandlerFor[GetMarketHistoryInput, GetMarketHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMarketHistoryInput) (*mcpsdk.CallToolResult, GetMarketHistoryOutput, error) {
		history, err := getMarketHistory(ctx, sso.UserAgent(version), in.RegionID, in.TypeID)
		if err != nil {
			return nil, GetMarketHistoryOutput{}, fmt.Errorf("get market history: %w", err)
		}
		out := make([]MarketHistoryEntry, len(history))
		for i, h := range history {
			out[i] = MarketHistoryEntry{Average: h.Average, Date: h.Date, Highest: h.Highest, Lowest: h.Lowest, OrderCount: h.OrderCount, Volume: h.Volume}
		}
		return nil, GetMarketHistoryOutput{RegionID: in.RegionID, TypeID: in.TypeID, History: out}, nil
	}
}

// --- list_market_orders (public, paginated) ---

var listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketOrders(ctx, regionID, orderType, typeID, page)
}

type ListMarketOrdersInput struct {
	RegionID  int64  `json:"regionId" jsonschema:"the region ID to look up"`
	OrderType string `json:"orderType" jsonschema:"which order types to return: buy, sell, or all"`
	TypeID    int64  `json:"typeId,omitempty" jsonschema:"optionally filter to just this item type ID"`
	Page      int32  `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListMarketOrdersOutput struct {
	RegionID   int64                  `json:"regionId"`
	OrderType  string                 `json:"orderType"`
	Page       int32                  `json:"page"`
	TotalPages int32                  `json:"totalPages"`
	Orders     []MarketOrderBookEntry `json:"orders"`
}

func listMarketOrdersHandler(version string) mcpsdk.ToolHandlerFor[ListMarketOrdersInput, ListMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketOrdersInput) (*mcpsdk.CallToolResult, ListMarketOrdersOutput, error) {
		orders, totalPages, err := listMarketOrders(ctx, sso.UserAgent(version), in.RegionID, in.OrderType, in.TypeID, in.Page)
		if err != nil {
			return nil, ListMarketOrdersOutput{}, fmt.Errorf("list market orders: %w", err)
		}
		out := make([]MarketOrderBookEntry, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderBookEntryFromRegion(o)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListMarketOrdersOutput{RegionID: in.RegionID, OrderType: in.OrderType, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_market_types (public, paginated) ---

var listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketTypes(ctx, regionID, page)
}

type ListMarketTypesInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
	Page     int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListMarketTypesOutput struct {
	RegionID   int64   `json:"regionId"`
	Page       int32   `json:"page"`
	TotalPages int32   `json:"totalPages"`
	TypeIDs    []int64 `json:"typeIds"`
}

func listMarketTypesHandler(version string) mcpsdk.ToolHandlerFor[ListMarketTypesInput, ListMarketTypesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketTypesInput) (*mcpsdk.CallToolResult, ListMarketTypesOutput, error) {
		types, totalPages, err := listMarketTypes(ctx, sso.UserAgent(version), in.RegionID, in.Page)
		if err != nil {
			return nil, ListMarketTypesOutput{}, fmt.Errorf("list market types: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListMarketTypesOutput{RegionID: in.RegionID, Page: page, TotalPages: totalPages, TypeIDs: types}, nil
	}
}

// --- list_structure_market_orders (private, paginated) ---
//
// Despite resembling list_market_orders (a public region order book), this tool requires a
// character auth gate identical to every other private tool in this file: ESI requires OAuth2
// scope esi-markets.structure_markets.v1 for GetMarketsStructuresStructureId (verified in
// esi/api/openapi.yaml), meaning the querying character must have docking access to the
// structure. There's no separate ownerId/corporationId input -- the structure itself is the
// resource, and ESI's own 403 is the sole authorization gate, same as every corp-scoped tool.

var listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.StructureMarketOrders(ctx, structureID, page)
}

type ListStructureMarketOrdersInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check -- must have docking access to the structure"`
	StructureID int64 `json:"structureId" jsonschema:"the structure ID to look up"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListStructureMarketOrdersOutput struct {
	CharacterID int32                  `json:"characterId"`
	StructureID int64                  `json:"structureId"`
	Page        int32                  `json:"page"`
	TotalPages  int32                  `json:"totalPages"`
	Orders      []MarketOrderBookEntry `json:"orders"`
}

func listStructureMarketOrdersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListStructureMarketOrdersInput, ListStructureMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListStructureMarketOrdersInput) (*mcpsdk.CallToolResult, ListStructureMarketOrdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListStructureMarketOrdersOutput{}, err
		}
		if !found {
			return nil, ListStructureMarketOrdersOutput{}, fmt.Errorf("list structure market orders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		orders, totalPages, err := listStructureMarketOrders(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.StructureID, in.Page)
		if err != nil {
			return nil, ListStructureMarketOrdersOutput{}, fmt.Errorf("list structure market orders: %w", err)
		}
		out := make([]MarketOrderBookEntry, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderBookEntryFromStructure(o)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListStructureMarketOrdersOutput{CharacterID: in.CharacterID, StructureID: in.StructureID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}
```

Modify `mcp/server.go`: append after the existing `delete_character_mail_label` block (immediately
before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_market_orders",
		Description: "List a logged-in EVE Online character's open market orders.",
	}, listCharacterMarketOrdersHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_market_order_history",
		Description: "List a logged-in EVE Online character's cancelled/expired market orders from the last 90 days.",
	}, listCharacterMarketOrderHistoryHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_market_orders",
		Description: "List a corporation's open market orders, using a logged-in character's roles to authorize.",
	}, listCorporationMarketOrdersHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_market_order_history",
		Description: "List a corporation's cancelled/expired market orders from the last 90 days, using a logged-in character's roles to authorize.",
	}, listCorporationMarketOrderHistoryHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_groups",
		Description: "List every market group ID in the game (public data, no character required).",
	}, listMarketGroupsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_market_group",
		Description: "Get detail for one market group, including its item types (public data, no character required).",
	}, getMarketGroupHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_prices",
		Description: "List ESI's adjusted/average price for every priced item type in the game (public data, no character required).",
	}, listMarketPricesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_market_history",
		Description: "Get daily historical market statistics for one item type in one region (public data, no character required).",
	}, getMarketHistoryHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_orders",
		Description: "List live market orders in a region, optionally filtered to one item type (public data, no character required).",
	}, listMarketOrdersHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_types",
		Description: "List the item type IDs with at least one active market order in a region (public data, no character required).",
	}, listMarketTypesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_structure_market_orders",
		Description: "List live market orders in a player structure, using a logged-in character's docking access to authorize.",
	}, listStructureMarketOrdersHandler(version, clientID, store))
```

Update `mcp/server.go`'s package doc comment: change the trailing
`"... and delete_character_mail_label registered."`
to end with
`"... delete_character_mail_label, list_character_market_orders, list_character_market_order_history, list_corporation_market_orders, list_corporation_market_order_history, list_market_groups, get_market_group, list_market_prices, get_market_history, list_market_orders, list_market_types, and list_structure_market_orders registered."`

Modify `mcp/server_test.go`: append `"list_character_market_orders"`,
`"list_character_market_order_history"`, `"list_corporation_market_orders"`,
`"list_corporation_market_order_history"`, `"list_market_groups"`, `"get_market_group"`,
`"list_market_prices"`, `"get_market_history"`, `"list_market_orders"`, `"list_market_types"`,
`"list_structure_market_orders"` to the expected-tools slice (after
`"delete_character_mail_label"`), and confirm the test's expected count is 157 (whether it's a
literal count or `len(expected)` computed from the slice — check the existing test to see which,
and match that style).

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Market row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: change "one hundred forty-six" to "one hundred fifty-seven", and append the
11 new tool names to the `mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestListCharacterMarketOrdersHandler|TestListCharacterMarketOrderHistoryHandler|TestListCorporationMarketOrdersHandler|TestListCorporationMarketOrderHistoryHandler|TestListMarketGroupsHandler|TestGetMarketGroupHandler|TestListMarketPricesHandler|TestGetMarketHistoryHandler|TestListMarketOrdersHandler|TestListMarketTypesHandler|TestListStructureMarketOrdersHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (16 new handler tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, 157 tools registered without panic.

```bash
git add mcp/market.go mcp/market_internal_test.go mcp/server.go mcp/server_test.go \
  CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add market MCP tools (11 tools: character/corp orders, groups, prices, region/structure orders)"
```
