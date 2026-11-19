# ESI Assets + Calendar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cover the Assets ESI resource group (6 endpoints, including corporation-scoped variants) and the Calendar resource group (4 endpoints, including this project's first write endpoint).

**Architecture:** `esi/assets.go` and `esi/calendar.go` are thin, `describeError`-wrapped facades exactly like every prior batch. Assets' two "resolve IDs to locations/names" endpoints are POST purely because they need a request body — verified read-only from source, not a write concern. Calendar's `RespondToCalendarEvent` is this project's first write: it returns only `error` (ESI gives a 204 with no body), and its MCP handler validates the response value before ever calling ESI.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context: `fmt.Errorf("operation: %w", err)`. Never discard an error.
- Every `esi/` method that calls `.Execute()` must route its error and `*http.Response` through `describeError` (`esi/error.go`) before wrapping — existing hard rule.
- External test packages by default (`package foo_test`); `*_internal_test.go` only when a test genuinely needs an unexported symbol.
- Standard library only in tests — no testify.
- Raw-ID passthrough: no code in this plan resolves any ID (`type_id`, `location_id`) to a human-readable name. `AssetNames` is a genuine exception worth noting: it returns ESI's own player-assigned custom names (e.g. a renamed ship or container) — a dynamic per-item feature FC's API itself provides, not an SDE-backed static lookup, so passing it through directly does not violate the SDE-over-ESI rule.
- Every new MCP tool checks `config.Store` for character existence before calling ESI, returning `ErrCharacterNotFound` if not found — matching every prior character-scoped tool.
- Write-tool convention (new, established in the design spec): a write tool's name is verb-first and distinct from every `list_`/`get_` read tool (`respond_calendar_event`), and its description explicitly states it changes state. No confirmation parameter. `respond_calendar_event`'s handler validates its `response` input (`"accepted"`, `"declined"`, or `"tentative"`) **before** loading `config.Store` or calling ESI — fail fast on bad input without an unnecessary IO round-trip.
- This branch is the shared PR branch (`worktree-esi-access-list-activities`, currently at commit `af82075`) — no separate feature branch, per the standing project convention that every ESI-coverage batch since Wallet folds into the same open PR (#6). The current tool-count baseline is **13** (`add_character`, `list_characters`, `remove_character`, `get_wallet_balance`, `list_access_lists`, `get_access_list`, `list_mercenary_tactical_operations`, `get_mercenary_tactical_operation`, `list_raidable_skyhooks`, `list_alliances`, `get_alliance`, `list_alliance_corporations`, `get_alliance_icons`) — verified by reading `mcp/server.go`/`mcp/server_test.go` directly before writing this plan.
- Design spec: `docs/superpowers/specs/2026-07-26-esi-assets-calendar-design.md`.
- All code in this plan has been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and `github.com/modelcontextprotocol/go-sdk` v1.6.1 modules — including a live `httptest.Server` round-trip confirming `RequestBody([]int64)` serializes as a bare JSON array and that `PutCharactersCharacterIdCalendarEventId`'s 204-no-content response doesn't error, plus a full compile of every `mcp/assets.go`/`mcp/calendar.go` handler against real `esiapi`/`mcpsdk` types — not guessed from documentation.

---

### Task 1: `esi/assets.go` — Assets facade (read-only, character + corporation scoped)

**Files:**
- Create: `esi/assets.go`
- Create: `esi/assets_test.go`

**Interfaces:**
- Consumes: `esi.Client`, `describeError` (existing); `esiapi.APIClient.AssetsAPI.GetCharactersCharacterIdAssets(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdAssetsGetInner, *http.Response, error)`, `.PostCharactersCharacterIdAssetsLocations(ctx, characterId int64).RequestBody(itemIDs []int64) ...Execute() ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, *http.Response, error)`, `.PostCharactersCharacterIdAssetsNames(ctx, characterId int64).RequestBody(itemIDs []int64) ...Execute() ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, *http.Response, error)`, `.GetCorporationsCorporationIdAssets(ctx, corporationId int64) ...Execute() ([]esiapi.CorporationsCorporationIdAssetsGetInner, *http.Response, error)`, `.PostCorporationsCorporationIdAssetsLocations(ctx, corporationId int64).RequestBody(itemIDs []int64) ...Execute() ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, *http.Response, error)`, `.PostCorporationsCorporationIdAssetsNames(ctx, corporationId int64).RequestBody(itemIDs []int64) ...Execute() ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, *http.Response, error)` — all verified against the real installed module.
- Produces: `(*esi.Client).Assets(ctx) ([]esiapi.CharactersCharacterIdAssetsGetInner, error)`, `(*esi.Client).AssetLocations(ctx, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error)`, `(*esi.Client).AssetNames(ctx, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error)`, `(*esi.Client).CorporationAssets(ctx, corporationID int64) ([]esiapi.CorporationsCorporationIdAssetsGetInner, error)`, `(*esi.Client).CorporationAssetLocations(ctx, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error)`, `(*esi.Client).CorporationAssetNames(ctx, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error)` — all consumed by Task 3.

- [ ] **Step 1: Write the failing tests**

Create `esi/assets_test.go`:

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

func TestAssets_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"is_singleton":true,"item_id":1,"location_flag":"Hangar","location_id":60003760,"location_type":"station","quantity":1,"type_id":587}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	assets, err := client.Assets(context.Background())
	if err != nil {
		t.Fatalf("Assets returned error: %v", err)
	}
	if len(assets) != 1 || assets[0].ItemId != 1 || assets[0].TypeId != 587 {
		t.Errorf("assets = %+v, want one entry with ItemId=1 TypeId=587", assets)
	}
}

func TestAssetLocations_SendsItemIDsAsBareJSONArray(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":1,"position":{"x":1.5,"y":2.5,"z":3.5}}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	locs, err := client.AssetLocations(context.Background(), []int64{1, 2, 3})
	if err != nil {
		t.Fatalf("AssetLocations returned error: %v", err)
	}
	if gotBody != "[1,2,3]" {
		t.Errorf("request body = %q, want \"[1,2,3]\" (bare array, not wrapped)", gotBody)
	}
	if len(locs) != 1 || locs[0].ItemId != 1 || locs[0].Position.X != 1.5 {
		t.Errorf("locs = %+v, want one entry with ItemId=1 Position.X=1.5", locs)
	}
}

func TestAssetLocations_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.AssetLocations(context.Background(), []int64{1}); err == nil {
		t.Fatal("AssetLocations returned nil error, want an error for HTTP 403")
	}
}

func TestAssetNames_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":1,"name":"My Renamed Ship"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	names, err := client.AssetNames(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("AssetNames returned error: %v", err)
	}
	if len(names) != 1 || names[0].ItemId != 1 || names[0].Name != "My Renamed Ship" {
		t.Errorf("names = %+v, want one entry {ItemId:1 Name:\"My Renamed Ship\"}", names)
	}
}

func TestCorporationAssets_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"is_singleton":false,"item_id":2,"location_flag":"CorpDeliveries","location_id":60003760,"location_type":"station","quantity":10,"type_id":34}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	assets, err := client.CorporationAssets(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationAssets returned error: %v", err)
	}
	if len(assets) != 1 || assets[0].ItemId != 2 || assets[0].Quantity != 10 {
		t.Errorf("assets = %+v, want one entry with ItemId=2 Quantity=10", assets)
	}
}

func TestCorporationAssetLocations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":2,"position":{"x":4,"y":5,"z":6}}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	locs, err := client.CorporationAssetLocations(context.Background(), 98000001, []int64{2})
	if err != nil {
		t.Fatalf("CorporationAssetLocations returned error: %v", err)
	}
	if len(locs) != 1 || locs[0].ItemId != 2 {
		t.Errorf("locs = %+v, want one entry with ItemId=2", locs)
	}
}

func TestCorporationAssetNames_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":2,"name":"Corp Hauler"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	names, err := client.CorporationAssetNames(context.Background(), 98000001, []int64{2})
	if err != nil {
		t.Fatalf("CorporationAssetNames returned error: %v", err)
	}
	if len(names) != 1 || names[0].Name != "Corp Hauler" {
		t.Errorf("names = %+v, want one entry with Name=\"Corp Hauler\"", names)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestAssets|TestAssetLocations|TestAssetNames|TestCorporationAssets|TestCorporationAssetLocations|TestCorporationAssetNames' -v`

Expected: FAIL to compile — `undefined: (*esi.Client).Assets`, `undefined: (*esi.Client).AssetLocations`, etc.

- [ ] **Step 3: Implement `esi/assets.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Assets returns the client's character's assets (bare IDs — call AssetLocations/AssetNames
// to resolve a subset of item IDs into positions or player-assigned names).
func (c *Client) Assets(ctx context.Context) ([]esiapi.CharactersCharacterIdAssetsGetInner, error) {
	result, resp, err := c.api.AssetsAPI.GetCharactersCharacterIdAssets(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get assets: %w", describeError(err, resp))
	}
	return result, nil
}

// AssetLocations resolves a set of item IDs (from Assets) into their positions.
func (c *Client) AssetLocations(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCharactersCharacterIdAssetsLocations(ctx, int64(c.characterID)).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get asset locations: %w", describeError(err, resp))
	}
	return result, nil
}

// AssetNames resolves a set of item IDs (from Assets) into their player-assigned custom names
// (e.g. a renamed ship or container) — not an SDE-backed item-type name lookup.
func (c *Client) AssetNames(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCharactersCharacterIdAssetsNames(ctx, int64(c.characterID)).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get asset names: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssets returns corporationID's assets. The authenticated character must hold
// sufficient corp roles — ESI itself returns 403 otherwise; no client-side check needed.
func (c *Client) CorporationAssets(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdAssetsGetInner, error) {
	result, resp, err := c.api.AssetsAPI.GetCorporationsCorporationIdAssets(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation assets: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssetLocations resolves a set of corporation item IDs into their positions.
func (c *Client) CorporationAssetLocations(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCorporationsCorporationIdAssetsLocations(ctx, corporationID).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation asset locations: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssetNames resolves a set of corporation item IDs into their player-assigned names.
func (c *Client) CorporationAssetNames(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCorporationsCorporationIdAssetsNames(ctx, corporationID).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation asset names: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all seven new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/assets.go esi/assets_test.go
git commit -m "feat: add esi/assets.go facade"
```

---

### Task 2: `esi/calendar.go` — Calendar facade (including the first write method)

**Files:**
- Create: `esi/calendar.go`
- Create: `esi/calendar_test.go`

**Interfaces:**
- Consumes: `esi.Client`, `describeError` (existing); `esiapi.APIClient.CalendarAPI.GetCharactersCharacterIdCalendar(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdCalendarGetInner, *http.Response, error)`, `.GetCharactersCharacterIdCalendarEventId(ctx, characterId, eventId int64) ...Execute() (*esiapi.CharactersCharacterIdCalendarEventIdGet, *http.Response, error)`, `.GetCharactersCharacterIdCalendarEventIdAttendees(ctx, characterId, eventId int64) ...Execute() ([]esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner, *http.Response, error)`, `.PutCharactersCharacterIdCalendarEventId(ctx, characterId, eventId int64).PutCharactersCharacterIdCalendarEventIdRequest(esiapi.PutCharactersCharacterIdCalendarEventIdRequest{Response string}) ...Execute() (*http.Response, error)` — all verified against the real installed module, including a live `httptest.Server` round-trip confirming the request body shape and the 204-no-content response.
- Produces: `(*esi.Client).CalendarEvents(ctx) ([]esiapi.CharactersCharacterIdCalendarGetInner, error)`, `(*esi.Client).CalendarEventDetail(ctx, eventID int64) (*esiapi.CharactersCharacterIdCalendarEventIdGet, error)`, `(*esi.Client).CalendarEventAttendees(ctx, eventID int64) ([]esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner, error)`, `(*esi.Client).RespondToCalendarEvent(ctx, eventID int64, response string) error` — all consumed by Task 4.

- [ ] **Step 1: Write the failing tests**

Create `esi/calendar_test.go`:

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

func TestCalendarEvents_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"event_date":"2026-08-01T00:00:00Z","event_id":456,"event_response":"needs_response","importance":1,"title":"Fleet Op"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	events, err := client.CalendarEvents(context.Background())
	if err != nil {
		t.Fatalf("CalendarEvents returned error: %v", err)
	}
	if len(events) != 1 || events[0].EventId == nil || *events[0].EventId != 456 {
		t.Errorf("events = %+v, want one entry with EventId=*456", events)
	}
	if events[0].Title == nil || *events[0].Title != "Fleet Op" {
		t.Errorf("events[0].Title = %v, want *\"Fleet Op\"", events[0].Title)
	}
}

func TestCalendarEventDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"date":"2026-08-01T00:00:00Z","duration":60,"event_id":456,"importance":1,"owner_id":98000001,"owner_name":"Test Corp","owner_type":"corporation","response":"needs_response","text":"Bring your ship","title":"Fleet Op"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CalendarEventDetail(context.Background(), 456)
	if err != nil {
		t.Fatalf("CalendarEventDetail returned error: %v", err)
	}
	if detail.Title != "Fleet Op" || detail.Duration != 60 || detail.OwnerName != "Test Corp" {
		t.Errorf("detail = %+v, want Title=\"Fleet Op\" Duration=60 OwnerName=\"Test Corp\"", detail)
	}
}

func TestCalendarEventDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CalendarEventDetail(context.Background(), 456); err == nil {
		t.Fatal("CalendarEventDetail returned nil error, want an error for HTTP 404")
	}
}

func TestCalendarEventAttendees_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"character_id":998877,"event_response":"accepted"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	attendees, err := client.CalendarEventAttendees(context.Background(), 456)
	if err != nil {
		t.Fatalf("CalendarEventAttendees returned error: %v", err)
	}
	if len(attendees) != 1 || attendees[0].CharacterId == nil || *attendees[0].CharacterId != 998877 {
		t.Errorf("attendees = %+v, want one entry with CharacterId=*998877", attendees)
	}
}

func TestRespondToCalendarEvent_SendsCorrectBodyAndHandlesNoContent(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.RespondToCalendarEvent(context.Background(), 456, "accepted"); err != nil {
		t.Fatalf("RespondToCalendarEvent returned error: %v", err)
	}
	if gotBody != `{"response":"accepted"}` {
		t.Errorf("request body = %q, want %q", gotBody, `{"response":"accepted"}`)
	}
}

func TestRespondToCalendarEvent_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.RespondToCalendarEvent(context.Background(), 456, "accepted"); err == nil {
		t.Fatal("RespondToCalendarEvent returned nil error, want an error for HTTP 403")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCalendarEvents|TestCalendarEventDetail|TestCalendarEventAttendees|TestRespondToCalendarEvent' -v`

Expected: FAIL to compile — all four new `esi.Client` methods are undefined.

- [ ] **Step 3: Implement `esi/calendar.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CalendarEvents returns the client's character's calendar events. Several fields are pointers
// (a still-pending invite can have some fields unset) — verified from the response model.
func (c *Client) CalendarEvents(ctx context.Context) ([]esiapi.CharactersCharacterIdCalendarGetInner, error) {
	result, resp, err := c.api.CalendarAPI.GetCharactersCharacterIdCalendar(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get calendar events: %w", describeError(err, resp))
	}
	return result, nil
}

// CalendarEventDetail returns full detail for one calendar event.
func (c *Client) CalendarEventDetail(ctx context.Context, eventID int64) (*esiapi.CharactersCharacterIdCalendarEventIdGet, error) {
	result, resp, err := c.api.CalendarAPI.GetCharactersCharacterIdCalendarEventId(ctx, int64(c.characterID), eventID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get calendar event detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CalendarEventAttendees returns the attendee list for one calendar event.
func (c *Client) CalendarEventAttendees(ctx context.Context, eventID int64) ([]esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner, error) {
	result, resp, err := c.api.CalendarAPI.GetCharactersCharacterIdCalendarEventIdAttendees(ctx, int64(c.characterID), eventID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get calendar event attendees: %w", describeError(err, resp))
	}
	return result, nil
}

// RespondToCalendarEvent sets the client's character's RSVP status on a calendar event.
// response must be one of "accepted", "declined", or "tentative" — ESI's own accepted values;
// the MCP layer validates this before calling, so a bad value fails fast with a clear message.
// This is a write: it changes state on FC's server. ESI returns no response body (204).
func (c *Client) RespondToCalendarEvent(ctx context.Context, eventID int64, response string) error {
	resp, err := c.api.CalendarAPI.PutCharactersCharacterIdCalendarEventId(ctx, int64(c.characterID), eventID).
		PutCharactersCharacterIdCalendarEventIdRequest(esiapi.PutCharactersCharacterIdCalendarEventIdRequest{Response: response}).
		Execute()
	if err != nil {
		return fmt.Errorf("respond to calendar event: %w", describeError(err, resp))
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all six new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/calendar.go esi/calendar_test.go
git commit -m "feat: add esi/calendar.go facade (first write method)"
```

---

### Task 3: `mcp/assets.go` tools + registration

**Files:**
- Create: `mcp/assets.go`
- Create: `mcp/assets_internal_test.go`
- Modify: `mcp/server.go`

**Interfaces:**
- Consumes: `esi.NewClient(ctx, clientID string, characterID int32, userAgent string) (*esi.Client, error)` (existing), `(*esi.Client).Assets`, `.AssetLocations`, `.AssetNames`, `.CorporationAssets`, `.CorporationAssetLocations`, `.CorporationAssetNames` (Task 1); `sso.UserAgent(version string) string`, `config.Store.Load() (config.Config, error)`, `config.Config.Characters []config.Character{ID int32, Name string}`, `ErrCharacterNotFound` (all existing).
- Produces: `mcp.ListAssetsInput/Output`, `mcp.GetAssetLocationsInput/Output`, `mcp.GetAssetNamesInput/Output`, `mcp.ListCorporationAssetsInput/Output`, `mcp.GetCorporationAssetLocationsInput/Output`, `mcp.GetCorporationAssetNamesInput/Output`, `mcp.AssetItem{ItemID, TypeID, LocationID int64, LocationType, LocationFlag string, Quantity int64, IsSingleton bool, IsBlueprintCopy *bool}`, `mcp.AssetLocation{ItemID int64, X, Y, Z float64}`, `mcp.AssetName{ItemID int64, Name string}`. Nothing later in this plan depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/assets_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListAssetsHandler_ReturnsAssets(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listAssets
	t.Cleanup(func() { listAssets = orig })
	listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AssetItem, error) {
		return []AssetItem{{ItemID: 1, TypeID: 587, LocationID: 60003760, LocationType: "station", LocationFlag: "Hangar", Quantity: 1, IsSingleton: true}}, nil
	}

	handler := listAssetsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Assets) != 1 || out.Assets[0].ItemID != 1 {
		t.Errorf("out = %+v, want CharacterID=7 with one asset ItemID=1", out)
	}
}

func TestListAssetsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listAssets
	t.Cleanup(func() { listAssets = orig })
	listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AssetItem, error) {
		t.Fatal("listAssets should not be called for an unknown character")
		return nil, nil
	}

	handler := listAssetsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetAssetLocationsHandler_ReturnsLocations(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getAssetLocations
	t.Cleanup(func() { getAssetLocations = orig })
	getAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetLocation, error) {
		if len(itemIDs) != 2 || itemIDs[0] != 1 || itemIDs[1] != 2 {
			t.Errorf("itemIDs = %v, want [1 2]", itemIDs)
		}
		return []AssetLocation{{ItemID: 1, X: 1.5, Y: 2.5, Z: 3.5}}, nil
	}

	handler := getAssetLocationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetAssetLocationsInput{CharacterID: 7, ItemIDs: []int64{1, 2}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Locations) != 1 || out.Locations[0].X != 1.5 {
		t.Errorf("out = %+v, want one location with X=1.5", out)
	}
}

func TestGetAssetLocationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getAssetLocations
	t.Cleanup(func() { getAssetLocations = orig })
	getAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetLocation, error) {
		t.Fatal("getAssetLocations should not be called for an unknown character")
		return nil, nil
	}

	handler := getAssetLocationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetAssetLocationsInput{CharacterID: 999, ItemIDs: []int64{1}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetAssetNamesHandler_ReturnsNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getAssetNames
	t.Cleanup(func() { getAssetNames = orig })
	getAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetName, error) {
		return []AssetName{{ItemID: 1, Name: "My Renamed Ship"}}, nil
	}

	handler := getAssetNamesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetAssetNamesInput{CharacterID: 7, ItemIDs: []int64{1}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Names) != 1 || out.Names[0].Name != "My Renamed Ship" {
		t.Errorf("out = %+v, want one name \"My Renamed Ship\"", out)
	}
}

func TestGetAssetNamesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getAssetNames
	t.Cleanup(func() { getAssetNames = orig })
	getAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetName, error) {
		t.Fatal("getAssetNames should not be called for an unknown character")
		return nil, nil
	}

	handler := getAssetNamesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetAssetNamesInput{CharacterID: 999, ItemIDs: []int64{1}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationAssetsHandler_ReturnsAssets(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationAssets
	t.Cleanup(func() { listCorporationAssets = orig })
	listCorporationAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]AssetItem, error) {
		if corporationID != 98000001 {
			t.Errorf("corporationID = %d, want 98000001", corporationID)
		}
		return []AssetItem{{ItemID: 2, TypeID: 34, Quantity: 10}}, nil
	}

	handler := listCorporationAssetsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Assets) != 1 {
		t.Errorf("out = %+v, want CorporationID=98000001 with one asset", out)
	}
}

func TestListCorporationAssetsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationAssets
	t.Cleanup(func() { listCorporationAssets = orig })
	listCorporationAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]AssetItem, error) {
		t.Fatal("listCorporationAssets should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationAssetsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationAssetLocationsHandler_ReturnsLocations(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationAssetLocations
	t.Cleanup(func() { getCorporationAssetLocations = orig })
	getCorporationAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetLocation, error) {
		return []AssetLocation{{ItemID: 2, X: 4, Y: 5, Z: 6}}, nil
	}

	handler := getCorporationAssetLocationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCorporationAssetLocationsInput{CharacterID: 7, CorporationID: 98000001, ItemIDs: []int64{2}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Locations) != 1 || out.Locations[0].ItemID != 2 {
		t.Errorf("out = %+v, want one location with ItemID=2", out)
	}
}

func TestGetCorporationAssetLocationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationAssetLocations
	t.Cleanup(func() { getCorporationAssetLocations = orig })
	getCorporationAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetLocation, error) {
		t.Fatal("getCorporationAssetLocations should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationAssetLocationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationAssetLocationsInput{CharacterID: 999, CorporationID: 98000001, ItemIDs: []int64{2}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationAssetNamesHandler_ReturnsNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationAssetNames
	t.Cleanup(func() { getCorporationAssetNames = orig })
	getCorporationAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetName, error) {
		return []AssetName{{ItemID: 2, Name: "Corp Hauler"}}, nil
	}

	handler := getCorporationAssetNamesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCorporationAssetNamesInput{CharacterID: 7, CorporationID: 98000001, ItemIDs: []int64{2}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Names) != 1 || out.Names[0].Name != "Corp Hauler" {
		t.Errorf("out = %+v, want one name \"Corp Hauler\"", out)
	}
}

func TestGetCorporationAssetNamesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationAssetNames
	t.Cleanup(func() { getCorporationAssetNames = orig })
	getCorporationAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetName, error) {
		t.Fatal("getCorporationAssetNames should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationAssetNamesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationAssetNamesInput{CharacterID: 999, CorporationID: 98000001, ItemIDs: []int64{2}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListAssetsHandler|TestGetAssetLocationsHandler|TestGetAssetNamesHandler|TestListCorporationAssetsHandler|TestGetCorporationAssetLocationsHandler|TestGetCorporationAssetNamesHandler' -v`

Expected: FAIL to compile — all the new `mcp` symbols referenced above are undefined.

- [ ] **Step 3: Implement `mcp/assets.go`**

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

// AssetItem is one asset entry, shared by character- and corporation-scoped listings.
type AssetItem struct {
	ItemID          int64  `json:"itemId"`
	TypeID          int64  `json:"typeId"`
	LocationID      int64  `json:"locationId"`
	LocationType    string `json:"locationType"`
	LocationFlag    string `json:"locationFlag"`
	Quantity        int64  `json:"quantity"`
	IsSingleton     bool   `json:"isSingleton"`
	IsBlueprintCopy *bool  `json:"isBlueprintCopy,omitempty"`
}

// listAssets is overridable in tests so no real network/auth runs.
var listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AssetItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	assets, err := client.Assets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AssetItem, len(assets))
	for i, a := range assets {
		out[i] = AssetItem{
			ItemID: a.ItemId, TypeID: a.TypeId, LocationID: a.LocationId,
			LocationType: a.LocationType, LocationFlag: a.LocationFlag,
			Quantity: a.Quantity, IsSingleton: a.IsSingleton, IsBlueprintCopy: a.IsBlueprintCopy,
		}
	}
	return out, nil
}

// ListAssetsInput is list_assets's tool input.
type ListAssetsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListAssetsOutput is list_assets's tool result.
type ListAssetsOutput struct {
	CharacterID int32       `json:"characterId"`
	Assets      []AssetItem `json:"assets"`
}

func listAssetsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListAssetsInput, ListAssetsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAssetsInput) (*mcpsdk.CallToolResult, ListAssetsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListAssetsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListAssetsOutput{}, fmt.Errorf("list assets for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		assets, err := listAssets(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListAssetsOutput{}, fmt.Errorf("list assets: %w", err)
		}
		return nil, ListAssetsOutput{CharacterID: in.CharacterID, Assets: assets}, nil
	}
}

// AssetLocation is one resolved item's position.
type AssetLocation struct {
	ItemID int64   `json:"itemId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
}

// getAssetLocations is overridable in tests.
var getAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetLocation, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	locs, err := client.AssetLocations(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetLocation, len(locs))
	for i, l := range locs {
		out[i] = AssetLocation{ItemID: l.ItemId, X: l.Position.X, Y: l.Position.Y, Z: l.Position.Z}
	}
	return out, nil
}

// GetAssetLocationsInput is get_asset_locations's tool input.
type GetAssetLocationsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ItemIDs     []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_assets"`
}

// GetAssetLocationsOutput is get_asset_locations's tool result.
type GetAssetLocationsOutput struct {
	CharacterID int32           `json:"characterId"`
	Locations   []AssetLocation `json:"locations"`
}

func getAssetLocationsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetAssetLocationsInput, GetAssetLocationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAssetLocationsInput) (*mcpsdk.CallToolResult, GetAssetLocationsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetAssetLocationsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetAssetLocationsOutput{}, fmt.Errorf("get asset locations for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		locs, err := getAssetLocations(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ItemIDs)
		if err != nil {
			return nil, GetAssetLocationsOutput{}, fmt.Errorf("get asset locations: %w", err)
		}
		return nil, GetAssetLocationsOutput{CharacterID: in.CharacterID, Locations: locs}, nil
	}
}

// AssetName is one item's player-assigned custom name.
type AssetName struct {
	ItemID int64  `json:"itemId"`
	Name   string `json:"name"`
}

// getAssetNames is overridable in tests.
var getAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetName, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	names, err := client.AssetNames(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetName, len(names))
	for i, n := range names {
		out[i] = AssetName{ItemID: n.ItemId, Name: n.Name}
	}
	return out, nil
}

// GetAssetNamesInput is get_asset_names's tool input.
type GetAssetNamesInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ItemIDs     []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_assets"`
}

// GetAssetNamesOutput is get_asset_names's tool result.
type GetAssetNamesOutput struct {
	CharacterID int32       `json:"characterId"`
	Names       []AssetName `json:"names"`
}

func getAssetNamesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetAssetNamesInput, GetAssetNamesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAssetNamesInput) (*mcpsdk.CallToolResult, GetAssetNamesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("get asset names for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		names, err := getAssetNames(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ItemIDs)
		if err != nil {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("get asset names: %w", err)
		}
		return nil, GetAssetNamesOutput{CharacterID: in.CharacterID, Names: names}, nil
	}
}

// listCorporationAssets is overridable in tests.
var listCorporationAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]AssetItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	assets, err := client.CorporationAssets(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]AssetItem, len(assets))
	for i, a := range assets {
		out[i] = AssetItem{
			ItemID: a.ItemId, TypeID: a.TypeId, LocationID: a.LocationId,
			LocationType: a.LocationType, LocationFlag: a.LocationFlag,
			Quantity: a.Quantity, IsSingleton: a.IsSingleton, IsBlueprintCopy: a.IsBlueprintCopy,
		}
	}
	return out, nil
}

// ListCorporationAssetsInput is list_corporation_assets's tool input.
type ListCorporationAssetsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}

// ListCorporationAssetsOutput is list_corporation_assets's tool result.
type ListCorporationAssetsOutput struct {
	CharacterID   int32       `json:"characterId"`
	CorporationID int64       `json:"corporationId"`
	Assets        []AssetItem `json:"assets"`
}

func listCorporationAssetsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationAssetsInput, ListCorporationAssetsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationAssetsInput) (*mcpsdk.CallToolResult, ListCorporationAssetsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("list corporation assets for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		assets, err := listCorporationAssets(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("list corporation assets: %w", err)
		}
		return nil, ListCorporationAssetsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Assets: assets}, nil
	}
}

// getCorporationAssetLocations is overridable in tests.
var getCorporationAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetLocation, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	locs, err := client.CorporationAssetLocations(ctx, corporationID, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetLocation, len(locs))
	for i, l := range locs {
		out[i] = AssetLocation{ItemID: l.ItemId, X: l.Position.X, Y: l.Position.Y, Z: l.Position.Z}
	}
	return out, nil
}

// GetCorporationAssetLocationsInput is get_corporation_asset_locations's tool input.
type GetCorporationAssetLocationsInput struct {
	CharacterID   int32   `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64   `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ItemIDs       []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_corporation_assets"`
}

// GetCorporationAssetLocationsOutput is get_corporation_asset_locations's tool result.
type GetCorporationAssetLocationsOutput struct {
	CharacterID   int32           `json:"characterId"`
	CorporationID int64           `json:"corporationId"`
	Locations     []AssetLocation `json:"locations"`
}

func getCorporationAssetLocationsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationAssetLocationsInput, GetCorporationAssetLocationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationAssetLocationsInput) (*mcpsdk.CallToolResult, GetCorporationAssetLocationsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("get corporation asset locations for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		locs, err := getCorporationAssetLocations(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ItemIDs)
		if err != nil {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("get corporation asset locations: %w", err)
		}
		return nil, GetCorporationAssetLocationsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Locations: locs}, nil
	}
}

// getCorporationAssetNames is overridable in tests.
var getCorporationAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetName, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	names, err := client.CorporationAssetNames(ctx, corporationID, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetName, len(names))
	for i, n := range names {
		out[i] = AssetName{ItemID: n.ItemId, Name: n.Name}
	}
	return out, nil
}

// GetCorporationAssetNamesInput is get_corporation_asset_names's tool input.
type GetCorporationAssetNamesInput struct {
	CharacterID   int32   `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64   `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ItemIDs       []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_corporation_assets"`
}

// GetCorporationAssetNamesOutput is get_corporation_asset_names's tool result.
type GetCorporationAssetNamesOutput struct {
	CharacterID   int32       `json:"characterId"`
	CorporationID int64       `json:"corporationId"`
	Names         []AssetName `json:"names"`
}

func getCorporationAssetNamesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationAssetNamesInput, GetCorporationAssetNamesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationAssetNamesInput) (*mcpsdk.CallToolResult, GetCorporationAssetNamesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("get corporation asset names for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		names, err := getCorporationAssetNames(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ItemIDs)
		if err != nil {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("get corporation asset names: %w", err)
		}
		return nil, GetCorporationAssetNamesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Names: names}, nil
	}
}
```

- [ ] **Step 4: Register the six tools in `mcp/server.go`**

In `mcp/server.go`, change the doc comment and add six new `mcpsdk.AddTool` calls right after
`get_alliance_icons`'s. Change:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, get_wallet_balance, list_access_lists, get_access_list,
// list_mercenary_tactical_operations, get_mercenary_tactical_operation,
// list_raidable_skyhooks, list_alliances, get_alliance,
// list_alliance_corporations, and get_alliance_icons registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
```

to:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, get_wallet_balance, list_access_lists, get_access_list,
// list_mercenary_tactical_operations, get_mercenary_tactical_operation,
// list_raidable_skyhooks, list_alliances, get_alliance,
// list_alliance_corporations, get_alliance_icons, list_assets,
// get_asset_locations, get_asset_names, list_corporation_assets,
// get_corporation_asset_locations, and get_corporation_asset_names registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
```

And change:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_alliance_icons",
		Description: "Get an alliance's icon URLs (public data, no character required).",
	}, getAllianceIconsHandler(version))

	return server
}
```

to:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_alliance_icons",
		Description: "Get an alliance's icon URLs (public data, no character required).",
	}, getAllianceIconsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_assets",
		Description: "List a logged-in EVE Online character's assets.",
	}, listAssetsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_asset_locations",
		Description: "Resolve a set of a character's asset item IDs into their positions.",
	}, getAssetLocationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_asset_names",
		Description: "Resolve a set of a character's asset item IDs into their player-assigned custom names.",
	}, getAssetNamesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_assets",
		Description: "List a corporation's assets, using a logged-in character's roles to authorize.",
	}, listCorporationAssetsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_asset_locations",
		Description: "Resolve a set of a corporation's asset item IDs into their positions.",
	}, getCorporationAssetLocationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_asset_names",
		Description: "Resolve a set of a corporation's asset item IDs into their player-assigned custom names.",
	}, getCorporationAssetNamesHandler(version, clientID, store))

	return server
}
```

- [ ] **Step 5: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./mcp/... -race -v`

Expected: all twelve new tests from Step 1 PASS, plus every pre-existing `mcp` test (note
`TestServer_AllToolsRegistered` isn't updated until Task 4 — it only asserts a subset of tool names
via membership check, so it still passes with six extra tools registered).

- [ ] **Step 6: Commit**

```bash
git add mcp/assets.go mcp/assets_internal_test.go mcp/server.go
git commit -m "feat: add list_assets, get_asset_locations, get_asset_names, and corporation variants MCP tools"
```

---

### Task 4: `mcp/calendar.go` tools + registration + `server_test.go` update

**Files:**
- Create: `mcp/calendar.go`
- Create: `mcp/calendar_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/server_test.go`

**Interfaces:**
- Consumes: `esi.NewClient` (existing), `(*esi.Client).CalendarEvents`, `.CalendarEventDetail`, `.CalendarEventAttendees`, `.RespondToCalendarEvent` (Task 2); `sso.UserAgent`, `config.Store.Load`, `ErrCharacterNotFound` (as Task 3).
- Produces: `mcp.ListCalendarEventsInput/Output`, `mcp.GetCalendarEventInput/Output`, `mcp.ListCalendarEventAttendeesInput/Output`, `mcp.RespondCalendarEventInput/Output`. Nothing later in this plan depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/calendar_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCalendarEventsHandler_ReturnsEvents(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCalendarEvents
	t.Cleanup(func() { listCalendarEvents = orig })
	eventID := int64(456)
	title := "Fleet Op"
	listCalendarEvents = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CalendarEventSummary, error) {
		return []CalendarEventSummary{{EventID: &eventID, Title: &title}}, nil
	}

	handler := listCalendarEventsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCalendarEventsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Events) != 1 || out.Events[0].EventID == nil || *out.Events[0].EventID != 456 {
		t.Errorf("out = %+v, want one event with EventID=*456", out)
	}
}

func TestListCalendarEventsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCalendarEvents
	t.Cleanup(func() { listCalendarEvents = orig })
	listCalendarEvents = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CalendarEventSummary, error) {
		t.Fatal("listCalendarEvents should not be called for an unknown character")
		return nil, nil
	}

	handler := listCalendarEventsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCalendarEventsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCalendarEventHandler_ReturnsDetail(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	orig := getCalendarEventDetail
	t.Cleanup(func() { getCalendarEventDetail = orig })
	getCalendarEventDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) (CalendarEventDetailResult, error) {
		return CalendarEventDetailResult{
			Date: wantDate, Duration: 60, Importance: 1, OwnerID: 98000001,
			OwnerName: "Test Corp", OwnerType: "corporation", Response: "needs_response",
			Text: "Bring your ship", Title: "Fleet Op",
		}, nil
	}

	handler := getCalendarEventHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCalendarEventInput{CharacterID: 7, EventID: 456})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.EventID != 456 || out.Title != "Fleet Op" || out.Duration != 60 || !out.Date.Equal(wantDate) {
		t.Errorf("out = %+v, want EventID=456 Title=\"Fleet Op\" Duration=60 Date=%v", out, wantDate)
	}
}

func TestGetCalendarEventHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCalendarEventDetail
	t.Cleanup(func() { getCalendarEventDetail = orig })
	getCalendarEventDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) (CalendarEventDetailResult, error) {
		t.Fatal("getCalendarEventDetail should not be called for an unknown character")
		return CalendarEventDetailResult{}, nil
	}

	handler := getCalendarEventHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCalendarEventInput{CharacterID: 999, EventID: 456})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCalendarEventAttendeesHandler_ReturnsAttendees(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCalendarEventAttendees
	t.Cleanup(func() { listCalendarEventAttendees = orig })
	charID := int64(998877)
	response := "accepted"
	listCalendarEventAttendees = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) ([]CalendarAttendee, error) {
		return []CalendarAttendee{{CharacterID: &charID, EventResponse: &response}}, nil
	}

	handler := listCalendarEventAttendeesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCalendarEventAttendeesInput{CharacterID: 7, EventID: 456})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Attendees) != 1 || out.Attendees[0].CharacterID == nil || *out.Attendees[0].CharacterID != 998877 {
		t.Errorf("out = %+v, want one attendee with CharacterID=*998877", out)
	}
}

func TestListCalendarEventAttendeesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCalendarEventAttendees
	t.Cleanup(func() { listCalendarEventAttendees = orig })
	listCalendarEventAttendees = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) ([]CalendarAttendee, error) {
		t.Fatal("listCalendarEventAttendees should not be called for an unknown character")
		return nil, nil
	}

	handler := listCalendarEventAttendeesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCalendarEventAttendeesInput{CharacterID: 999, EventID: 456})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRespondCalendarEventHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := respondToCalendarEvent
	t.Cleanup(func() { respondToCalendarEvent = orig })
	var gotResponse string
	respondToCalendarEvent = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64, response string) error {
		gotResponse = response
		return nil
	}

	handler := respondCalendarEventHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 7, EventID: 456, Response: "accepted"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotResponse != "accepted" || out.Response != "accepted" || out.EventID != 456 {
		t.Errorf("out = %+v, gotResponse = %q, want response \"accepted\" passed through and echoed", out, gotResponse)
	}
}

func TestRespondCalendarEventHandler_InvalidResponse_NeverCallsESI(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := respondToCalendarEvent
	t.Cleanup(func() { respondToCalendarEvent = orig })
	respondToCalendarEvent = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64, response string) error {
		t.Fatal("respondToCalendarEvent should not be called for an invalid response value")
		return nil
	}

	handler := respondCalendarEventHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 7, EventID: 456, Response: "maybe"})
	if err == nil {
		t.Fatal("handler returned nil error, want a validation error for an invalid response value")
	}
}

func TestRespondCalendarEventHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := respondToCalendarEvent
	t.Cleanup(func() { respondToCalendarEvent = orig })
	respondToCalendarEvent = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64, response string) error {
		t.Fatal("respondToCalendarEvent should not be called for an unknown character")
		return nil
	}

	handler := respondCalendarEventHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 999, EventID: 456, Response: "accepted"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListCalendarEventsHandler|TestGetCalendarEventHandler|TestListCalendarEventAttendeesHandler|TestRespondCalendarEventHandler' -v`

Expected: FAIL to compile — all the new `mcp` symbols referenced above are undefined.

- [ ] **Step 3: Implement `mcp/calendar.go`**

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

// CalendarEventSummary is one calendar event's bare-ID summary. All fields are pointers — a
// still-pending invite can have some fields unset (verified from ESI's response model).
type CalendarEventSummary struct {
	EventID       *int64     `json:"eventId,omitempty"`
	EventDate     *time.Time `json:"eventDate,omitempty"`
	EventResponse *string    `json:"eventResponse,omitempty"`
	Importance    *int64     `json:"importance,omitempty"`
	Title         *string    `json:"title,omitempty"`
}

// listCalendarEvents is overridable in tests.
var listCalendarEvents = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CalendarEventSummary, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	events, err := client.CalendarEvents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CalendarEventSummary, len(events))
	for i, e := range events {
		out[i] = CalendarEventSummary{
			EventID: e.EventId, EventDate: e.EventDate, EventResponse: e.EventResponse,
			Importance: e.Importance, Title: e.Title,
		}
	}
	return out, nil
}

// ListCalendarEventsInput is list_calendar_events's tool input.
type ListCalendarEventsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListCalendarEventsOutput is list_calendar_events's tool result.
type ListCalendarEventsOutput struct {
	CharacterID int32                  `json:"characterId"`
	Events      []CalendarEventSummary `json:"events"`
}

func listCalendarEventsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCalendarEventsInput, ListCalendarEventsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCalendarEventsInput) (*mcpsdk.CallToolResult, ListCalendarEventsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCalendarEventsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListCalendarEventsOutput{}, fmt.Errorf("list calendar events for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		events, err := listCalendarEvents(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCalendarEventsOutput{}, fmt.Errorf("list calendar events: %w", err)
		}
		return nil, ListCalendarEventsOutput{CharacterID: in.CharacterID, Events: events}, nil
	}
}

// CalendarEventDetailResult is the raw mapped detail returned by getCalendarEventDetail, before
// the handler adds the characterID/eventID it already knows from the request.
type CalendarEventDetailResult struct {
	Date       time.Time
	Duration   int64
	Importance int64
	OwnerID    int64
	OwnerName  string
	OwnerType  string
	Response   string
	Text       string
	Title      string
}

// getCalendarEventDetail is overridable in tests.
var getCalendarEventDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) (CalendarEventDetailResult, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return CalendarEventDetailResult{}, fmt.Errorf("build ESI client: %w", err)
	}
	detail, err := client.CalendarEventDetail(ctx, eventID)
	if err != nil {
		return CalendarEventDetailResult{}, err
	}
	return CalendarEventDetailResult{
		Date: detail.Date, Duration: detail.Duration, Importance: detail.Importance,
		OwnerID: detail.OwnerId, OwnerName: detail.OwnerName, OwnerType: detail.OwnerType,
		Response: detail.Response, Text: detail.Text, Title: detail.Title,
	}, nil
}

// GetCalendarEventInput is get_calendar_event's tool input.
type GetCalendarEventInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	EventID     int64 `json:"eventId" jsonschema:"the calendar event ID, from list_calendar_events"`
}

// GetCalendarEventOutput is get_calendar_event's tool result.
type GetCalendarEventOutput struct {
	CharacterID int32     `json:"characterId"`
	EventID     int64     `json:"eventId"`
	Date        time.Time `json:"date"`
	Duration    int64     `json:"duration"`
	Importance  int64     `json:"importance"`
	OwnerID     int64     `json:"ownerId"`
	OwnerName   string    `json:"ownerName"`
	OwnerType   string    `json:"ownerType"`
	Response    string    `json:"response"`
	Text        string    `json:"text"`
	Title       string    `json:"title"`
}

func getCalendarEventHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCalendarEventInput, GetCalendarEventOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCalendarEventInput) (*mcpsdk.CallToolResult, GetCalendarEventOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetCalendarEventOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetCalendarEventOutput{}, fmt.Errorf("get calendar event for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		result, err := getCalendarEventDetail(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.EventID)
		if err != nil {
			return nil, GetCalendarEventOutput{}, fmt.Errorf("get calendar event: %w", err)
		}
		return nil, GetCalendarEventOutput{
			CharacterID: in.CharacterID, EventID: in.EventID, Date: result.Date, Duration: result.Duration,
			Importance: result.Importance, OwnerID: result.OwnerID, OwnerName: result.OwnerName,
			OwnerType: result.OwnerType, Response: result.Response, Text: result.Text, Title: result.Title,
		}, nil
	}
}

// CalendarAttendee is one calendar event attendee.
type CalendarAttendee struct {
	CharacterID   *int64  `json:"characterId,omitempty"`
	EventResponse *string `json:"eventResponse,omitempty"`
}

// listCalendarEventAttendees is overridable in tests.
var listCalendarEventAttendees = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) ([]CalendarAttendee, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	attendees, err := client.CalendarEventAttendees(ctx, eventID)
	if err != nil {
		return nil, err
	}
	out := make([]CalendarAttendee, len(attendees))
	for i, a := range attendees {
		out[i] = CalendarAttendee{CharacterID: a.CharacterId, EventResponse: a.EventResponse}
	}
	return out, nil
}

// ListCalendarEventAttendeesInput is list_calendar_event_attendees's tool input.
type ListCalendarEventAttendeesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	EventID     int64 `json:"eventId" jsonschema:"the calendar event ID, from list_calendar_events"`
}

// ListCalendarEventAttendeesOutput is list_calendar_event_attendees's tool result.
type ListCalendarEventAttendeesOutput struct {
	CharacterID int32              `json:"characterId"`
	EventID     int64              `json:"eventId"`
	Attendees   []CalendarAttendee `json:"attendees"`
}

func listCalendarEventAttendeesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCalendarEventAttendeesInput, ListCalendarEventAttendeesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCalendarEventAttendeesInput) (*mcpsdk.CallToolResult, ListCalendarEventAttendeesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCalendarEventAttendeesOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListCalendarEventAttendeesOutput{}, fmt.Errorf("list calendar event attendees for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		attendees, err := listCalendarEventAttendees(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.EventID)
		if err != nil {
			return nil, ListCalendarEventAttendeesOutput{}, fmt.Errorf("list calendar event attendees: %w", err)
		}
		return nil, ListCalendarEventAttendeesOutput{CharacterID: in.CharacterID, EventID: in.EventID, Attendees: attendees}, nil
	}
}

// validCalendarResponses are ESI's own accepted values for RespondToCalendarEvent.
var validCalendarResponses = map[string]bool{"accepted": true, "declined": true, "tentative": true}

// respondToCalendarEvent is overridable in tests.
var respondToCalendarEvent = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64, response string) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.RespondToCalendarEvent(ctx, eventID, response)
}

// RespondCalendarEventInput is respond_calendar_event's tool input.
type RespondCalendarEventInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to respond as"`
	EventID     int64  `json:"eventId" jsonschema:"the calendar event ID to respond to"`
	Response    string `json:"response" jsonschema:"one of accepted, declined, or tentative"`
}

// RespondCalendarEventOutput is respond_calendar_event's tool result.
type RespondCalendarEventOutput struct {
	CharacterID int32  `json:"characterId"`
	EventID     int64  `json:"eventId"`
	Response    string `json:"response"`
}

// respondCalendarEventHandler is neat's first write tool. It changes state on FC's server, so
// unlike every read-only handler in this project it validates its input BEFORE loading
// config.Store or calling ESI — failing fast on bad input without an unnecessary IO round-trip.
func respondCalendarEventHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[RespondCalendarEventInput, RespondCalendarEventOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RespondCalendarEventInput) (*mcpsdk.CallToolResult, RespondCalendarEventOutput, error) {
		if !validCalendarResponses[in.Response] {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("respond to calendar event: invalid response %q, must be one of accepted, declined, tentative", in.Response)
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("respond to calendar event for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := respondToCalendarEvent(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.EventID, in.Response); err != nil {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("respond to calendar event: %w", err)
		}
		return nil, RespondCalendarEventOutput{CharacterID: in.CharacterID, EventID: in.EventID, Response: in.Response}, nil
	}
}
```

- [ ] **Step 4: Register the four tools in `mcp/server.go`**

In `mcp/server.go`, change the doc comment and add four new `mcpsdk.AddTool` calls right after
`get_corporation_asset_names`'s. Change:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, get_wallet_balance, list_access_lists, get_access_list,
// list_mercenary_tactical_operations, get_mercenary_tactical_operation,
// list_raidable_skyhooks, list_alliances, get_alliance,
// list_alliance_corporations, get_alliance_icons, list_assets,
// get_asset_locations, get_asset_names, list_corporation_assets,
// get_corporation_asset_locations, and get_corporation_asset_names registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
```

to:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, get_wallet_balance, list_access_lists, get_access_list,
// list_mercenary_tactical_operations, get_mercenary_tactical_operation,
// list_raidable_skyhooks, list_alliances, get_alliance,
// list_alliance_corporations, get_alliance_icons, list_assets,
// get_asset_locations, get_asset_names, list_corporation_assets,
// get_corporation_asset_locations, get_corporation_asset_names,
// list_calendar_events, get_calendar_event, list_calendar_event_attendees,
// and respond_calendar_event registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
```

And change:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_asset_names",
		Description: "Resolve a set of a corporation's asset item IDs into their player-assigned custom names.",
	}, getCorporationAssetNamesHandler(version, clientID, store))

	return server
}
```

to:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_asset_names",
		Description: "Resolve a set of a corporation's asset item IDs into their player-assigned custom names.",
	}, getCorporationAssetNamesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_calendar_events",
		Description: "List a logged-in EVE Online character's calendar events.",
	}, listCalendarEventsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_calendar_event",
		Description: "Get detail (date, duration, owner, description, RSVP status) for one calendar event.",
	}, getCalendarEventHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_calendar_event_attendees",
		Description: "List the attendees and their RSVP status for one calendar event.",
	}, listCalendarEventAttendeesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "respond_calendar_event",
		Description: "Respond to a calendar event invite (accepted/declined/tentative) — this changes your RSVP status on EVE's server.",
	}, respondCalendarEventHandler(version, clientID, store))

	return server
}
```

- [ ] **Step 5: Update `mcp/server_test.go`'s expected-tools list**

In `mcp/server_test.go`, change:

```go
	for _, want := range []string{
		"add_character", "list_characters", "remove_character", "get_wallet_balance",
		"list_access_lists", "get_access_list",
		"list_mercenary_tactical_operations", "get_mercenary_tactical_operation", "list_raidable_skyhooks",
		"list_alliances", "get_alliance", "list_alliance_corporations", "get_alliance_icons",
	} {
```

to:

```go
	for _, want := range []string{
		"add_character", "list_characters", "remove_character", "get_wallet_balance",
		"list_access_lists", "get_access_list",
		"list_mercenary_tactical_operations", "get_mercenary_tactical_operation", "list_raidable_skyhooks",
		"list_alliances", "get_alliance", "list_alliance_corporations", "get_alliance_icons",
		"list_assets", "get_asset_locations", "get_asset_names",
		"list_corporation_assets", "get_corporation_asset_locations", "get_corporation_asset_names",
		"list_calendar_events", "get_calendar_event", "list_calendar_event_attendees", "respond_calendar_event",
	} {
```

- [ ] **Step 6: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race -v`

Expected: every new test from this task and Task 3 PASSes, `TestServer_AllToolsRegistered` PASSes
with all 23 tool names, `gofmt -l .` prints nothing, and every pre-existing test across the whole
module still passes.

- [ ] **Step 7: Commit**

```bash
git add mcp/calendar.go mcp/calendar_internal_test.go mcp/server.go mcp/server_test.go
git commit -m "feat: add calendar MCP tools including respond_calendar_event (first write tool)"
```

---

### Task 5: Docs sync (`CLAUDE.md`, `README.md`, roadmap doc)

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Modify: `docs/reference/esi-coverage-roadmap.md`

**Interfaces:** none — documentation only.

- [ ] **Step 1: Update `CLAUDE.md`'s package table**

Replace:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its thirteen tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance`, `list_access_lists`, `get_access_list`, `list_mercenary_tactical_operations`, `get_mercenary_tactical_operation`, `list_raidable_skyhooks`, `list_alliances`, `get_alliance`, `list_alliance_corporations`, `get_alliance_icons` |
```

with:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its twenty-three tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance`, `list_access_lists`, `get_access_list`, `list_mercenary_tactical_operations`, `get_mercenary_tactical_operation`, `list_raidable_skyhooks`, `list_alliances`, `get_alliance`, `list_alliance_corporations`, `get_alliance_icons`, `list_assets`, `get_asset_locations`, `get_asset_names`, `list_corporation_assets`, `get_corporation_asset_locations`, `get_corporation_asset_names`, `list_calendar_events`, `get_calendar_event`, `list_calendar_event_attendees`, `respond_calendar_event` (the project's first write tool — verb-first name, description states it changes state, no confirmation parameter) |
```

- [ ] **Step 2: Update `README.md`**

Replace the `## Status` section's tool-coverage sentence:

```markdown
and real ESI-backed domain tools (`esi/`) covering Wallet, Access List, Activities, and Alliance
are built. See `docs/reference/esi-coverage-roadmap.md` for full ESI-coverage progress. The SDE
cache and plugin/extension packaging don't exist yet. See the plans directory for what's next.
```

with:

```markdown
and real ESI-backed domain tools (`esi/`) covering Wallet, Access List, Activities, Alliance,
Assets, and Calendar are built. See `docs/reference/esi-coverage-roadmap.md` for full
ESI-coverage progress. The SDE cache and plugin/extension packaging don't exist yet. See the
plans directory for what's next.
```

Update the package table's `mcp/` row the same way as `CLAUDE.md`'s Step 1.

- [ ] **Step 3: Update the roadmap doc**

In `docs/reference/esi-coverage-roadmap.md`, change:

```markdown
| 4 | Assets | 6 | Planned |
| 5 | Calendar | 4 | Planned |
```

to:

```markdown
| 4 | Assets | 6 | This plan |
| 5 | Calendar | 4 | This plan |
```

- [ ] **Step 4: Sanity-check the docs against the code**

Run: `grep -n "respond_calendar_event\|list_corporation_assets\|twenty-three" CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md`

Expected: `CLAUDE.md` and `README.md` mention the new tool names and tool count; the roadmap doc's
grep only matches via the other two files (it doesn't mention tool names, just tag rows — this
command still confirms all three files were touched as expected).

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "docs: document assets/calendar tools, update roadmap and tool count"
```
