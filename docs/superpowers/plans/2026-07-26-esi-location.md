# ESI Location Implementation Plan (twenty-first batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Location" tag (3 endpoints: current solar system/station/
structure, online status, current ship) as 3 new MCP tools, folding into the same shared PR #6
branch as every prior batch.

**Architecture:** Same two-layer split as every prior batch — `esi/location.go` wraps the 3
generated `LocationAPIService` methods behind the project's `Client`, using the established
"my own character" convention (`int64(c.characterID)`, never a caller-supplied character ID,
since none of these endpoints accept one). `mcp/location.go` exposes 3 tools on top of that,
reusing `characterExists`/`ErrCharacterNotFound` from `mcp/character.go`.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: 132 MCP tools, branch `worktree-esi-access-list-activities`,
  PR #6 (open). This plan brings the count to 135.
- All 3 Location endpoints are private ("my own character") and read-only — no pagination, no
  write, no corp-scoped variants, no oneOf/polymorphic types. Verified directly against the
  installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module (`esi/api_location.go`).
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- Reuse `characterExists(store *config.Store, characterID int32) (bool, error)` and
  `ErrCharacterNotFound` from `mcp/character.go` — do not redeclare.
- **Standing convention (added after the Killmails batch):** never model a nested/tree-shaped ESI
  resource with a self-referential Go struct — it compiles/vets/gofmts clean but panics at server
  startup ("cycle detected") inside the MCP SDK's output-schema generator. Not applicable to this
  batch (no nested types), but keep in mind for the ~17 tags still remaining in the sweep.
- `CharacterOnline`'s 4 return fields (`bool` + 3 nullable pointers) are grouped into an exported
  `esi.CharacterOnlineStatus` struct rather than passed as bare multi-return — this is the only
  batch-specific type decision; everything else follows `esi/fittings.go`'s exact shape.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND to pass a live
  `mcpsdk.AddTool` registration test (the same mechanism that caught the Killmails self-referential
  panic) with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-location-design.md` for the verification detail.

---

### Task 1: `esi/location.go` — ESI facade layer

**Files:**
- Create: `esi/location.go`
- Test: `esi/location_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`);
  `c.api.LocationAPI` and `c.characterID` (`esi/client.go`'s `Client`, unchanged).
- Produces (for Task 2): `func (c *Client) CharacterLocation(ctx context.Context) (*esiapi.CharactersCharacterIdLocationGet, error)`;
  `type CharacterOnlineStatus struct { Online bool; LastLogin, LastLogout *time.Time; Logins *int64 }`
  and `func (c *Client) CharacterOnline(ctx context.Context) (*CharacterOnlineStatus, error)`;
  `func (c *Client) CharacterShip(ctx context.Context) (*esiapi.CharactersCharacterIdShipGet, error)`.

- [ ] **Step 1: Write the failing tests**

Create `esi/location_test.go`:

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

func TestCharacterLocation_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/location/" {
			t.Errorf("request path = %q, want /characters/998877/location/", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"solar_system_id":30000142,"station_id":60003760}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	loc, err := client.CharacterLocation(context.Background())
	if err != nil {
		t.Fatalf("CharacterLocation returned error: %v", err)
	}
	if loc.SolarSystemId != 30000142 {
		t.Errorf("SolarSystemId = %d, want 30000142", loc.SolarSystemId)
	}
	if loc.StationId == nil || *loc.StationId != 60003760 {
		t.Errorf("StationId = %v, want pointer to 60003760", loc.StationId)
	}
	if loc.StructureId != nil {
		t.Errorf("StructureId = %v, want nil", loc.StructureId)
	}
}

func TestCharacterLocation_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterLocation(context.Background()); err == nil {
		t.Fatal("CharacterLocation returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterOnline_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/online/" {
			t.Errorf("request path = %q, want /characters/998877/online/", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"online":true,"last_login":"2026-07-01T12:00:00Z","last_logout":"2026-06-30T23:00:00Z","logins":42}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	status, err := client.CharacterOnline(context.Background())
	if err != nil {
		t.Fatalf("CharacterOnline returned error: %v", err)
	}
	if !status.Online {
		t.Error("Online = false, want true")
	}
	if status.LastLogin == nil || status.LastLogin.Year() != 2026 {
		t.Errorf("LastLogin = %v, want a 2026 timestamp", status.LastLogin)
	}
	if status.LastLogout == nil {
		t.Error("LastLogout = nil, want a timestamp")
	}
	if status.Logins == nil || *status.Logins != 42 {
		t.Errorf("Logins = %v, want pointer to 42", status.Logins)
	}
}

func TestCharacterOnline_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterOnline(context.Background()); err == nil {
		t.Fatal("CharacterOnline returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterShip_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/998877/ship/" {
			t.Errorf("request path = %q, want /characters/998877/ship/", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ship_item_id":1000000016991,"ship_name":"Percival","ship_type_id":608}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ship, err := client.CharacterShip(context.Background())
	if err != nil {
		t.Fatalf("CharacterShip returned error: %v", err)
	}
	if ship.ShipItemId != 1000000016991 || ship.ShipName != "Percival" || ship.ShipTypeId != 608 {
		t.Errorf("ship = %+v, want ShipItemId=1000000016991 ShipName=Percival ShipTypeId=608", ship)
	}
}

func TestCharacterShip_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterShip(context.Background()); err == nil {
		t.Fatal("CharacterShip returned nil error, want an error for HTTP 404")
	}
}
```

Note: verify the exact resolved request paths (`/characters/998877/location/` etc., including the
trailing slash) by first running the tests and reading the actual failure output if the path
assertions don't match the generated client's real routing — the generated client's exact path
formatting (trailing slash or not) has varied slightly across resource groups in this sweep, so
confirm empirically rather than assuming.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterLocation|TestCharacterOnline|TestCharacterShip' -v`
Expected: FAIL — `esi.Client` has no method `CharacterLocation`/`CharacterOnline`/`CharacterShip`.

- [ ] **Step 3: Write the implementation**

Create `esi/location.go`:

```go
// Package esi's location.go covers all 3 of "Location"'s endpoints -- current solar system/
// station/structure, online status, and current ship. All 3 are private/"my own character" --
// the generated request builders take characterId as a plain positional int64, which this facade
// passes as int64(c.characterID), never a caller-supplied parameter.
package esi

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterLocation returns the client's own character's current solar system, and station or
// structure ID if docked/boarded.
func (c *Client) CharacterLocation(ctx context.Context) (*esiapi.CharactersCharacterIdLocationGet, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdLocation(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character location: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterOnlineStatus is CharacterOnline's return shape, grouped into a struct instead of a
// long multi-value return since 4 nullable/boolean fields is unwieldy as bare return values.
type CharacterOnlineStatus struct {
	Online     bool
	LastLogin  *time.Time
	LastLogout *time.Time
	Logins     *int64
}

// CharacterOnline returns the client's own character's online status and login history.
func (c *Client) CharacterOnline(ctx context.Context) (*CharacterOnlineStatus, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdOnline(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character online status: %w", describeError(err, resp))
	}
	return &CharacterOnlineStatus{
		Online:     result.Online,
		LastLogin:  result.LastLogin,
		LastLogout: result.LastLogout,
		Logins:     result.Logins,
	}, nil
}

// CharacterShip returns the client's own character's current ship.
func (c *Client) CharacterShip(ctx context.Context) (*esiapi.CharactersCharacterIdShipGet, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdShip(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character ship: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestCharacterLocation|TestCharacterOnline|TestCharacterShip' -v`
Expected: PASS (6 tests).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./...`
Expected: all clean, no output from `gofmt -l`.

```bash
git add esi/location.go esi/location_test.go
git commit -m "feat: add esi/location.go facade (character location, online status, current ship)"
```

---

### Task 2: `mcp/location.go` — MCP tool layer + registration + docs

**Files:**
- Create: `mcp/location.go`
- Test: `mcp/location_internal_test.go`
- Modify: `mcp/server.go` (register 3 tools after the `get_killmail` block; update package doc comment)
- Modify: `mcp/server_test.go` (expected-tools list: 132 → 135)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Location row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (tool count "one hundred thirty-two" → "one hundred thirty-five", append 3 tool names)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: `esi.NewClient(ctx, clientID, characterID, userAgent) (*esi.Client, error)`;
  `(*esi.Client).CharacterLocation/.CharacterOnline/.CharacterShip` from Task 1;
  `esi.CharacterOnlineStatus` from Task 1; `characterExists`/`ErrCharacterNotFound`
  (`mcp/character.go`); `sso.UserAgent(version string) string` (`mcp/sso`);
  `config.Store` (`mcp/config`).
- Produces: 3 tool names — `get_character_location`, `get_character_online`, `get_character_ship`
  — registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `mcp/location_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestGetCharacterLocationHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterLocation
	t.Cleanup(func() { getCharacterLocation = orig })
	station := int64(60003760)
	getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
		return 30000142, &station, nil, nil
	}

	handler := getCharacterLocationHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterLocationInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.SolarSystemID != 30000142 {
		t.Fatalf("out = %+v, want CharacterID=7 SolarSystemID=30000142", out)
	}
	if out.StationID == nil || *out.StationID != 60003760 {
		t.Errorf("StationID = %v, want pointer to 60003760", out.StationID)
	}
	if out.StructureID != nil {
		t.Errorf("StructureID = %v, want nil", out.StructureID)
	}
}

func TestGetCharacterLocationHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterLocation
	t.Cleanup(func() { getCharacterLocation = orig })
	getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
		t.Fatal("getCharacterLocation should not be called for an unknown character")
		return 0, nil, nil, nil
	}

	handler := getCharacterLocationHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterLocationInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterOnlineHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterOnline
	t.Cleanup(func() { getCharacterOnline = orig })
	lastLogin := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	lastLogout := time.Date(2026, 6, 30, 23, 0, 0, 0, time.UTC)
	logins := int64(42)
	getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
		return &esi.CharacterOnlineStatus{Online: true, LastLogin: &lastLogin, LastLogout: &lastLogout, Logins: &logins}, nil
	}

	handler := getCharacterOnlineHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterOnlineInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || !out.Online {
		t.Fatalf("out = %+v, want CharacterID=7 Online=true", out)
	}
	if out.LastLogin == nil || !out.LastLogin.Equal(lastLogin) {
		t.Errorf("LastLogin = %v, want %v", out.LastLogin, lastLogin)
	}
	if out.LastLogout == nil || !out.LastLogout.Equal(lastLogout) {
		t.Errorf("LastLogout = %v, want %v", out.LastLogout, lastLogout)
	}
	if out.Logins == nil || *out.Logins != 42 {
		t.Errorf("Logins = %v, want pointer to 42", out.Logins)
	}
}

func TestGetCharacterOnlineHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterOnline
	t.Cleanup(func() { getCharacterOnline = orig })
	getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
		t.Fatal("getCharacterOnline should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterOnlineHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterOnlineInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterShipHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterShip
	t.Cleanup(func() { getCharacterShip = orig })
	getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
		return 1000000016991, "Percival", 608, nil
	}

	handler := getCharacterShipHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.ShipItemID != 1000000016991 || out.ShipName != "Percival" || out.ShipTypeID != 608 {
		t.Fatalf("out = %+v, want CharacterID=7 ShipItemID=1000000016991 ShipName=Percival ShipTypeID=608", out)
	}
}

func TestGetCharacterShipHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterShip
	t.Cleanup(func() { getCharacterShip = orig })
	getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
		t.Fatal("getCharacterShip should not be called for an unknown character")
		return 0, "", 0, nil
	}

	handler := getCharacterShipHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestGetCharacterLocationHandler|TestGetCharacterOnlineHandler|TestGetCharacterShipHandler' -v`
Expected: FAIL — undefined symbols (`getCharacterLocation`, `GetCharacterLocationInput`, etc. don't exist yet).

- [ ] **Step 3: Write the implementation**

Create `mcp/location.go`:

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

// --- get_character_location (private) ---

var getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	loc, err := client.CharacterLocation(ctx)
	if err != nil {
		return 0, nil, nil, err
	}
	return loc.SolarSystemId, loc.StationId, loc.StructureId, nil
}

type GetCharacterLocationInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterLocationOutput struct {
	CharacterID   int32  `json:"characterId"`
	SolarSystemID int64  `json:"solarSystemId"`
	StationID     *int64 `json:"stationId,omitempty"`
	StructureID   *int64 `json:"structureId,omitempty"`
}

func getCharacterLocationHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterLocationInput, GetCharacterLocationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterLocationInput) (*mcpsdk.CallToolResult, GetCharacterLocationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterLocationOutput{}, err
		}
		if !found {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		solarSystemID, stationID, structureID, err := getCharacterLocation(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location: %w", err)
		}
		return nil, GetCharacterLocationOutput{CharacterID: in.CharacterID, SolarSystemID: solarSystemID, StationID: stationID, StructureID: structureID}, nil
	}
}

// --- get_character_online (private) ---

var getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterOnline(ctx)
}

type GetCharacterOnlineInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterOnlineOutput struct {
	CharacterID int32      `json:"characterId"`
	Online      bool       `json:"online"`
	LastLogin   *time.Time `json:"lastLogin,omitempty"`
	LastLogout  *time.Time `json:"lastLogout,omitempty"`
	Logins      *int64     `json:"logins,omitempty"`
}

func getCharacterOnlineHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterOnlineInput, GetCharacterOnlineOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterOnlineInput) (*mcpsdk.CallToolResult, GetCharacterOnlineOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, err
		}
		if !found {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		status, err := getCharacterOnline(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status: %w", err)
		}
		return nil, GetCharacterOnlineOutput{CharacterID: in.CharacterID, Online: status.Online, LastLogin: status.LastLogin, LastLogout: status.LastLogout, Logins: status.Logins}, nil
	}
}

// --- get_character_ship (private) ---

var getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, "", 0, fmt.Errorf("build ESI client: %w", err)
	}
	ship, err := client.CharacterShip(ctx)
	if err != nil {
		return 0, "", 0, err
	}
	return ship.ShipItemId, ship.ShipName, ship.ShipTypeId, nil
}

type GetCharacterShipInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterShipOutput struct {
	CharacterID int32  `json:"characterId"`
	ShipItemID  int64  `json:"shipItemId"`
	ShipName    string `json:"shipName"`
	ShipTypeID  int64  `json:"shipTypeId"`
}

func getCharacterShipHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterShipInput, GetCharacterShipOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterShipInput) (*mcpsdk.CallToolResult, GetCharacterShipOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterShipOutput{}, err
		}
		if !found {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		shipItemID, shipName, shipTypeID, err := getCharacterShip(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship: %w", err)
		}
		return nil, GetCharacterShipOutput{CharacterID: in.CharacterID, ShipItemID: shipItemID, ShipName: shipName, ShipTypeID: shipTypeID}, nil
	}
}
```

Modify `mcp/server.go`: append after the existing `get_killmail` block (immediately before
`return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_location",
		Description: "Get a logged-in EVE Online character's current solar system, and station or structure ID if docked/boarded.",
	}, getCharacterLocationHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_online",
		Description: "Get a logged-in EVE Online character's online status and login history.",
	}, getCharacterOnlineHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_ship",
		Description: "Get a logged-in EVE Online character's current ship.",
	}, getCharacterShipHandler(version, clientID, store))
```

Update `mcp/server.go`'s package doc comment: change the trailing
`"... list_character_killmails_recent, list_corporation_killmails_recent, and get_killmail registered."`
to end with
`"... list_character_killmails_recent, list_corporation_killmails_recent, get_killmail, get_character_location, get_character_online, and get_character_ship registered."`

Modify `mcp/server_test.go`: append `"get_character_location"`, `"get_character_online"`,
`"get_character_ship"` to the expected-tools slice (after `"get_killmail"`), and confirm the test's
expected count is 135 (whether it's a literal count or `len(expected)` computed from the slice —
check the existing test to see which, and match that style).

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Location row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: change "one hundred thirty-two" to "one hundred thirty-five", and append
`get_character_location`, `get_character_online`, `get_character_ship` to the `mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestGetCharacterLocationHandler|TestGetCharacterOnlineHandler|TestGetCharacterShipHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (6 new handler tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, 135 tools registered without panic.

```bash
git add mcp/location.go mcp/location_internal_test.go mcp/server.go mcp/server_test.go \
  CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add location MCP tools (3 tools: current location, online status, ship)"
```
