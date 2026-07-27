# ESI Structures Implementation Plan (32nd batch)

> **For agentic workers:** This plan is executed directly by the orchestrator (no subagent
> dispatch), per the user's explicit instruction after this session's subagent spawn cap was hit
> during the Sovereignty batch. Steps use checkbox (`- [ ]`) syntax for tracking only.

**Goal:** Add ESI coverage for the "Structures" tag: a character's own mercenary dens, and a
corporation's skyhooks and sovereignty hubs (6 read-only endpoints, 6 new MCP tools).

**Architecture:** One `esi/structures.go` facade file (6 methods, no pagination, mixing
"my own character" and corp-scoped auth conventions) + one `mcp/structures.go` tool file (6 tools,
flattening 2 nested oneOf transport unions and several small nested structs into nesting-free MCP
output types).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi@v0.0.37`, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Current tool count baseline: **173** (verify with `grep -c "mcpsdk.AddTool" mcp/server.go`
  before writing any doc-count edit — if it has drifted from 173, use the real count instead of
  this plan's assumption, per this sweep's established self-correction discipline).
- Branch: `worktree-esi-access-list-activities` (shared PR #6 — every batch folds into this same
  PR, never a new one). Base commit before Task 1: `bcf5c59`.
- Every `esi/` method returns exactly what ESI returns, zero name-resolution enrichment.
- Both `esi/structures.go` methods needed OAuth scopes are ALREADY present in
  `mcp/add_character.go`'s `allScopes` (`esi-structures.read_corporation.v1`,
  `esi-structures.read_character.v1`) — no scope changes needed in this batch.
- None of the 6 endpoints is paginated — verified individually per endpoint from
  `api_structures.go` (no `.Page()` setter on any of the 6 request builders). Do not add
  pagination handling.
- Nested/tree-shaped models must never be modeled with a self-referential Go struct (verified
  panic risk in the MCP SDK's schema generator, only caught by an actual `go test` exercising
  `mcpsdk.AddTool`, not `go build`/`go vet`/`gofmt`). Every nested type in this batch's spec was
  already verified non-recursive — implementers must preserve that flattening exactly as
  specified, not "simplify" it back toward the original nested oneOf shape.
- The full design spec, including every exact field name and every generated type's real shape,
  is at `docs/superpowers/specs/2026-07-26-esi-structures-design.md`. Read it before starting
  either task — it is the source of truth for exact field names, not this plan's prose summaries.

---

## Task 1: `esi/structures.go`

**Files:**
- Create: `esi/structures.go`
- Create: `esi/structures_test.go`

**Interfaces:**
- Consumes: `Client.api.StructuresAPI` (generated field, confirmed via
  `grep -n "StructuresAPI " esi/../../go/pkg/mod/.../client.go` — it exists as
  `StructuresAPI StructuresAPI` on the generated client struct), `Client.characterID int32`,
  `describeError(err error, resp *http.Response) error` (`esi/error.go`).
- Produces: `Client.CharacterMercenaryDens(ctx) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error)`,
  `Client.CharacterMercenaryDenDetail(ctx, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error)`,
  `Client.CorporationSkyhooks(ctx, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error)`,
  `Client.CorporationSkyhookDetail(ctx, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error)`,
  `Client.CorporationSovereigntyHubs(ctx, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error)`,
  `Client.CorporationSovereigntyHubDetail(ctx, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error)`
  — these 6 exact signatures are what Task 2 will call.

- [ ] **Step 1: Write `esi/structures.go`**

```go
// Package esi's structures.go covers all 6 of "Structures"'s endpoints -- a character's own
// mercenary dens, and a corporation's skyhooks and sovereignty hubs. Mercenary dens are
// private/"my own character" (esi-structures.read_character.v1); skyhooks and sovereignty hubs
// are corp-scoped (esi-structures.read_corporation.v1), same convention as
// esi/planetary_interaction.go's CorporationCustomsOffices -- ESI's own 403 is the sole
// authorization gate, no additional role check in this facade. None of the 6 endpoints is
// paginated (verified individually: no .Page() setter on any of the 6 request builders).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterMercenaryDens returns the client's own character's mercenary dens (bare ID + planet ID
// per den; call CharacterMercenaryDenDetail for full detail on one).
func (c *Client) CharacterMercenaryDens(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
	result, resp, err := c.api.StructuresAPI.GetCharactersStructuresMercenaryDensListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mercenary dens: %w", describeError(err, resp))
	}
	return result.MercenaryDens, nil
}

// CharacterMercenaryDenDetail returns one of the client's own character's mercenary dens in full
// detail: evolution stats, infomorph count, reinforcement timer (if any), and the skyhook it's
// attached to.
func (c *Client) CharacterMercenaryDenDetail(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCharactersStructuresMercenaryDensDetail(ctx, mercenaryDenID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mercenary den detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationSkyhooks returns corporationID's skyhooks (bare ID + planet ID per skyhook; call
// CorporationSkyhookDetail for full detail on one). Corp-scoped: the authenticated character must
// hold sufficient corp roles per ESI's own x-required-roles -- this facade adds no additional
// gate, ESI's 403 is authoritative.
func (c *Client) CorporationSkyhooks(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSkyhooksListing(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation skyhooks: %w", describeError(err, resp))
	}
	return result.Skyhooks, nil
}

// CorporationSkyhookDetail returns one of corporationID's skyhooks in full detail: workforce,
// reagent stock, active/reinforcement state, and theft vulnerability window (if any).
func (c *Client) CorporationSkyhookDetail(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSkyhooksDetail(ctx, skyhookID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation skyhook detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationSovereigntyHubs returns corporationID's sovereignty hubs (bare ID + solar system ID
// per hub; call CorporationSovereigntyHubDetail for full detail on one).
func (c *Client) CorporationSovereigntyHubs(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSovereigntyHubsListing(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation sovereignty hubs: %w", describeError(err, resp))
	}
	return result.SovereigntyHubs, nil
}

// CorporationSovereigntyHubDetail returns one of corporationID's sovereignty hubs in full detail:
// reagent bay, power/workforce resources, installed upgrades, vulnerability window (if any), and
// workforce transport configuration/state.
func (c *Client) CorporationSovereigntyHubDetail(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSovereigntyHubsDetail(ctx, sovereigntyHubID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation sovereignty hub detail: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Write `esi/structures_test.go`**

Follow the exact `httptest.Server` + `ExportedNewClientWithOptions`/`ExportedNewClientFromHTTPClient`
round-trip pattern used in `esi/skills_test.go` and `esi/sovereignty_test.go` (read one of those
files first for the exact test-helper names in this package — e.g. whatever exported test
constructor this package already uses to point the generated client at an `httptest.Server`).
Write 12 tests (6 endpoints × success + HTTP-error), plus 1 extra test confirming the
skyhook/sovereignty-hub detail param order:

```go
package esi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterMercenaryDens_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/12345/structures/mercenary-dens" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"mercenary_dens": []map[string]any{{"id": 1001, "planet_id": 40000001}},
		})
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	dens, err := client.CharacterMercenaryDens(t.Context())
	if err != nil {
		t.Fatalf("CharacterMercenaryDens: %v", err)
	}
	if len(dens) != 1 || dens[0].Id != 1001 || dens[0].PlanetId != 40000001 {
		t.Errorf("dens = %+v", dens)
	}
}

func TestCharacterMercenaryDens_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	if _, err := client.CharacterMercenaryDens(t.Context()); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterMercenaryDenDetail_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/12345/structures/mercenary-dens/1001" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": 1001, "type_id": 85230, "state": "online",
			"evolution": map[string]any{
				"anarchy":     map[string]any{"amount": 10, "level": "low"},
				"development": map[string]any{"amount": 20, "level": "medium"},
			},
			"infomorphs": map[string]any{"amount": 3},
			"reinforcement_timer": map[string]any{"end": "2026-08-01T00:00:00Z"},
			"skyhook": map[string]any{"corporation_id": 98000001, "id": 2001, "planet_id": 40000001},
		})
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	detail, err := client.CharacterMercenaryDenDetail(t.Context(), 1001)
	if err != nil {
		t.Fatalf("CharacterMercenaryDenDetail: %v", err)
	}
	if detail.Id != 1001 || detail.TypeId != 85230 || detail.State != "online" {
		t.Errorf("detail = %+v", detail)
	}
	if detail.Evolution.Anarchy.Amount != 10 || detail.Evolution.Development.Level != "medium" {
		t.Errorf("evolution = %+v", detail.Evolution)
	}
	if detail.ReinforcementTimer == nil || !detail.ReinforcementTimer.End.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("reinforcement timer = %+v", detail.ReinforcementTimer)
	}
	if detail.Skyhook.CorporationId != 98000001 {
		t.Errorf("skyhook = %+v", detail.Skyhook)
	}
}

func TestCharacterMercenaryDenDetail_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	if _, err := client.CharacterMercenaryDenDetail(t.Context(), 1001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSkyhooks_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/corporations/98000001/structures/skyhooks" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"skyhooks": []map[string]any{{"id": 2001, "planet_id": 40000001}},
		})
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	hooks, err := client.CorporationSkyhooks(t.Context(), 98000001)
	if err != nil {
		t.Fatalf("CorporationSkyhooks: %v", err)
	}
	if len(hooks) != 1 || hooks[0].Id != 2001 {
		t.Errorf("hooks = %+v", hooks)
	}
}

func TestCorporationSkyhooks_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	if _, err := client.CorporationSkyhooks(t.Context(), 98000001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSkyhookDetail_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/corporations/98000001/structures/skyhooks/2001" {
			t.Errorf("path = %q -- verifies skyhookId/corporationId argument order", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": 2001, "planet_id": 40000001, "is_active": true, "state": "gathering_workforce",
			"effective_workforce": 500,
			"reagents": []map[string]any{
				{"last_cycle": "2026-07-01T00:00:00Z", "secured_stock": 100, "type_id": 16274, "unsecured_stock": 50},
			},
			"reinforcement_timer": map[string]any{"end": "2026-08-01T00:00:00Z"},
			"theft_vulnerability": map[string]any{"start": "2026-07-15T00:00:00Z", "end": "2026-07-16T00:00:00Z"},
		})
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	detail, err := client.CorporationSkyhookDetail(t.Context(), 98000001, 2001)
	if err != nil {
		t.Fatalf("CorporationSkyhookDetail: %v", err)
	}
	if detail.Id != 2001 || !detail.IsActive || detail.State != "gathering_workforce" {
		t.Errorf("detail = %+v", detail)
	}
	if detail.EffectiveWorkforce == nil || *detail.EffectiveWorkforce != 500 {
		t.Errorf("effective workforce = %+v", detail.EffectiveWorkforce)
	}
	if len(detail.Reagents) != 1 || detail.Reagents[0].SecuredStock != 100 {
		t.Errorf("reagents = %+v", detail.Reagents)
	}
	if detail.TheftVulnerability == nil {
		t.Errorf("theft vulnerability = nil")
	}
}

func TestCorporationSkyhookDetail_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	if _, err := client.CorporationSkyhookDetail(t.Context(), 98000001, 2001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSovereigntyHubs_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/corporations/98000001/structures/sovereignty-hubs" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sovereignty_hubs": []map[string]any{{"id": 3001, "solar_system_id": 30000142}},
		})
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	hubs, err := client.CorporationSovereigntyHubs(t.Context(), 98000001)
	if err != nil {
		t.Fatalf("CorporationSovereigntyHubs: %v", err)
	}
	if len(hubs) != 1 || hubs[0].Id != 3001 {
		t.Errorf("hubs = %+v", hubs)
	}
}

func TestCorporationSovereigntyHubs_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	if _, err := client.CorporationSovereigntyHubs(t.Context(), 98000001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSovereigntyHubDetail_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/corporations/98000001/structures/sovereignty-hubs/3001" {
			t.Errorf("path = %q -- verifies sovereigntyHubId/corporationId argument order", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": 3001, "solar_system_id": 30000142, "fuel_access_list_id": 5001,
			"reagent_bay": map[string]any{
				"last_updated": "2026-07-20T00:00:00Z",
				"reagents":     []map[string]any{{"amount": 1000, "burning_per_hour": 10, "type_id": 16274}},
			},
			"resources": map[string]any{
				"power":     map[string]any{"allocated": 100, "available": 200},
				"workforce": map[string]any{"allocated": 50, "available": 75},
			},
			"upgrades": []map[string]any{{"power_state": "online", "type_id": 81000}},
			"vulnerability_window": map[string]any{
				"start": "2026-07-25T00:00:00Z", "end": "2026-07-25T02:00:00Z",
			},
			"workforce_transport": map[string]any{
				"configuration": map[string]any{
					"import": map[string]any{"sources": []map[string]any{{"solar_system_id": 30000144}}},
				},
				"state": map[string]any{
					"import": map[string]any{
						"sources": []map[string]any{{"amount": 500, "solar_system_id": 30000144}},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	detail, err := client.CorporationSovereigntyHubDetail(t.Context(), 98000001, 3001)
	if err != nil {
		t.Fatalf("CorporationSovereigntyHubDetail: %v", err)
	}
	if detail.Id != 3001 || detail.SolarSystemId != 30000142 {
		t.Errorf("detail = %+v", detail)
	}
	if detail.FuelAccessListId == nil || *detail.FuelAccessListId != 5001 {
		t.Errorf("fuel access list id = %+v", detail.FuelAccessListId)
	}
	if len(detail.ReagentBay.Reagents) != 1 || detail.ReagentBay.Reagents[0].Amount != 1000 {
		t.Errorf("reagent bay = %+v", detail.ReagentBay)
	}
	if detail.Resources.Power.Allocated != 100 || detail.Resources.Workforce.Available != 75 {
		t.Errorf("resources = %+v", detail.Resources)
	}
	if len(detail.Upgrades) != 1 || detail.Upgrades[0].TypeId != 81000 {
		t.Errorf("upgrades = %+v", detail.Upgrades)
	}
	if detail.VulnerabilityWindow == nil {
		t.Errorf("vulnerability window = nil")
	}
	if detail.WorkforceTransport.Configuration.ModelImport == nil ||
		len(detail.WorkforceTransport.Configuration.ModelImport.Import.Sources) != 1 ||
		detail.WorkforceTransport.Configuration.ModelImport.Import.Sources[0].SolarSystemId != 30000144 {
		t.Errorf("transport configuration = %+v", detail.WorkforceTransport.Configuration)
	}
	if detail.WorkforceTransport.State.Import1 == nil ||
		len(detail.WorkforceTransport.State.Import1.Import.Sources) != 1 ||
		detail.WorkforceTransport.State.Import1.Import.Sources[0].Amount != 500 {
		t.Errorf("transport state = %+v", detail.WorkforceTransport.State)
	}
}

func TestCorporationSovereigntyHubDetail_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := esi.ExportedNewClientWithOptions(server.Client(), 12345, esi.ExportedClientOptionsForTest(server.URL))
	if _, err := client.CorporationSovereigntyHubDetail(t.Context(), 98000001, 3001); err == nil {
		t.Fatal("expected error")
	}
}
```

Before writing this file for real: read `esi/sovereignty_test.go` (or `esi/skills_test.go`) in
full first to copy the *exact* exported test-helper name(s) this package uses to point the
generated client at an `httptest.Server` (the sketch above uses placeholder names
`ExportedNewClientWithOptions`/`ExportedClientOptionsForTest` — confirm the real names from an
existing test file and use those, not these placeholders).

- [ ] **Step 3: Run the new tests**

Run: `go test ./esi/... -run 'TestCharacterMercenaryDens|TestCharacterMercenaryDenDetail|TestCorporationSkyhooks|TestCorporationSkyhookDetail|TestCorporationSovereigntyHubs|TestCorporationSovereigntyHubDetail' -v`
Expected: all pass (13 tests: 12 + the param-order assertions embedded in the detail tests).

- [ ] **Step 4: `go build ./... && go vet ./... && gofmt -l .`**

Expected: clean (no output from `gofmt -l .`, no errors from build/vet).

- [ ] **Step 5: Commit**

```bash
git add esi/structures.go esi/structures_test.go
git commit -m "feat: add structures esi facade (mercenary dens, skyhooks, sovereignty hubs)"
```

---

## Task 2: `mcp/structures.go` + registration + docs

**Files:**
- Create: `mcp/structures.go`
- Create: `mcp/structures_internal_test.go`
- Modify: `mcp/server.go` (register 6 tools, update package doc comment)
- Modify: `mcp/server_test.go` (append 6 tool names to expected-tools list)
- Modify: `CLAUDE.md` (tool count + tool list)
- Modify: `README.md` (tool count + tool list, mirrors CLAUDE.md)
- Modify: `docs/reference/esi-coverage-roadmap.md` (row 32 "Structures": `Planned` → `This plan`)

**Interfaces:**
- Consumes: the 6 `esi.Client` methods from Task 1 (exact signatures above); `characterExists(store
  *config.Store, characterID int32) (bool, error)` and `ErrCharacterNotFound` (`mcp/character.go`);
  `esi.NewClient(ctx, clientID string, characterID int32, userAgent string) (*esi.Client, error)`;
  `sso.UserAgent(version string) string`.
- Produces: 6 new `mcpsdk.AddTool` registrations in `mcp/server.go`.

- [ ] **Step 1: Write `mcp/structures.go`**

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

// --- list_character_mercenary_dens (private, "my own character") ---

type MercenaryDen struct {
	ID       int64 `json:"id"`
	PlanetID int64 `json:"planetId"`
}

var listCharacterMercenaryDens = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MercenaryDen, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	dens, err := client.CharacterMercenaryDens(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MercenaryDen, len(dens))
	for i, d := range dens {
		out[i] = MercenaryDen{ID: d.Id, PlanetID: d.PlanetId}
	}
	return out, nil
}

type ListCharacterMercenaryDensInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterMercenaryDensOutput struct {
	CharacterID int32          `json:"characterId"`
	Dens        []MercenaryDen `json:"dens"`
}

func listCharacterMercenaryDensHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterMercenaryDensInput, ListCharacterMercenaryDensOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMercenaryDensInput) (*mcpsdk.CallToolResult, ListCharacterMercenaryDensOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMercenaryDensOutput{}, err
		}
		if !found {
			return nil, ListCharacterMercenaryDensOutput{}, fmt.Errorf("list character mercenary dens for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		dens, err := listCharacterMercenaryDens(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterMercenaryDensOutput{}, fmt.Errorf("list character mercenary dens: %w", err)
		}
		return nil, ListCharacterMercenaryDensOutput{CharacterID: in.CharacterID, Dens: dens}, nil
	}
}

// --- get_character_mercenary_den (private, "my own character") ---

type MercenaryDenEvolutionStat struct {
	Amount int64  `json:"amount"`
	Level  string `json:"level"`
}

type MercenaryDenEvolution struct {
	Anarchy     MercenaryDenEvolutionStat `json:"anarchy"`
	Development MercenaryDenEvolutionStat `json:"development"`
}

type MercenaryDenSkyhookRef struct {
	CorporationID int64 `json:"corporationId"`
	ID            int64 `json:"id"`
	PlanetID      int64 `json:"planetId"`
}

type MercenaryDenDetail struct {
	ID               int64                  `json:"id"`
	TypeID           int64                  `json:"typeId"`
	State            string                 `json:"state"`
	Evolution        MercenaryDenEvolution  `json:"evolution"`
	InfomorphsAmount int64                  `json:"infomorphsAmount"`
	ReinforcementEnd *time.Time             `json:"reinforcementEnd,omitempty"`
	Skyhook          MercenaryDenSkyhookRef `json:"skyhook"`
}

var getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, mercenaryDenID int64) (MercenaryDenDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return MercenaryDenDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	d, err := client.CharacterMercenaryDenDetail(ctx, mercenaryDenID)
	if err != nil {
		return MercenaryDenDetail{}, err
	}
	out := MercenaryDenDetail{
		ID:     d.Id,
		TypeID: d.TypeId,
		State:  d.State,
		Evolution: MercenaryDenEvolution{
			Anarchy:     MercenaryDenEvolutionStat{Amount: d.Evolution.Anarchy.Amount, Level: d.Evolution.Anarchy.Level},
			Development: MercenaryDenEvolutionStat{Amount: d.Evolution.Development.Amount, Level: d.Evolution.Development.Level},
		},
		InfomorphsAmount: d.Infomorphs.Amount,
		Skyhook: MercenaryDenSkyhookRef{
			CorporationID: d.Skyhook.CorporationId,
			ID:            d.Skyhook.Id,
			PlanetID:      d.Skyhook.PlanetId,
		},
	}
	if d.ReinforcementTimer != nil {
		out.ReinforcementEnd = &d.ReinforcementTimer.End
	}
	return out, nil
}

type GetCharacterMercenaryDenInput struct {
	CharacterID    int32 `json:"characterId" jsonschema:"the character ID to check"`
	MercenaryDenID int64 `json:"mercenaryDenId" jsonschema:"the mercenary den ID to look up"`
}

func getCharacterMercenaryDenHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterMercenaryDenInput, MercenaryDenDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterMercenaryDenInput) (*mcpsdk.CallToolResult, MercenaryDenDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, MercenaryDenDetail{}, err
		}
		if !found {
			return nil, MercenaryDenDetail{}, fmt.Errorf("get character mercenary den for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCharacterMercenaryDen(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.MercenaryDenID)
		if err != nil {
			return nil, MercenaryDenDetail{}, fmt.Errorf("get character mercenary den: %w", err)
		}
		return nil, detail, nil
	}
}

// --- list_corporation_skyhooks (private, corp-scoped) ---

type Skyhook struct {
	ID       int64 `json:"id"`
	PlanetID int64 `json:"planetId"`
}

var listCorporationSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Skyhook, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	hooks, err := client.CorporationSkyhooks(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]Skyhook, len(hooks))
	for i, h := range hooks {
		out[i] = Skyhook{ID: h.Id, PlanetID: h.PlanetId}
	}
	return out, nil
}

type ListCorporationSkyhooksInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationSkyhooksOutput struct {
	CharacterID   int32     `json:"characterId"`
	CorporationID int64     `json:"corporationId"`
	Skyhooks      []Skyhook `json:"skyhooks"`
}

func listCorporationSkyhooksHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationSkyhooksInput, ListCorporationSkyhooksOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationSkyhooksInput) (*mcpsdk.CallToolResult, ListCorporationSkyhooksOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSkyhooksOutput{}, err
		}
		if !found {
			return nil, ListCorporationSkyhooksOutput{}, fmt.Errorf("list corporation skyhooks for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		hooks, err := listCorporationSkyhooks(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationSkyhooksOutput{}, fmt.Errorf("list corporation skyhooks: %w", err)
		}
		return nil, ListCorporationSkyhooksOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Skyhooks: hooks}, nil
	}
}

// --- get_corporation_skyhook (private, corp-scoped) ---

type SkyhookReagent struct {
	LastCycle      time.Time `json:"lastCycle"`
	SecuredStock   int64     `json:"securedStock"`
	TypeID         int64     `json:"typeId"`
	UnsecuredStock int64     `json:"unsecuredStock"`
}

type SkyhookDetail struct {
	EffectiveWorkforce      *int64           `json:"effectiveWorkforce,omitempty"`
	ID                      int64            `json:"id"`
	PlanetID                int64            `json:"planetId"`
	IsActive                bool             `json:"isActive"`
	Reagents                []SkyhookReagent `json:"reagents,omitempty"`
	ReinforcementEnd        *time.Time       `json:"reinforcementEnd,omitempty"`
	State                   string           `json:"state"`
	TheftVulnerabilityStart *time.Time       `json:"theftVulnerabilityStart,omitempty"`
	TheftVulnerabilityEnd   *time.Time       `json:"theftVulnerabilityEnd,omitempty"`
}

var getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return SkyhookDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	d, err := client.CorporationSkyhookDetail(ctx, corporationID, skyhookID)
	if err != nil {
		return SkyhookDetail{}, err
	}
	out := SkyhookDetail{
		EffectiveWorkforce: d.EffectiveWorkforce,
		ID:                 d.Id,
		PlanetID:           d.PlanetId,
		IsActive:           d.IsActive,
		State:              d.State,
	}
	if len(d.Reagents) > 0 {
		out.Reagents = make([]SkyhookReagent, len(d.Reagents))
		for i, r := range d.Reagents {
			out.Reagents[i] = SkyhookReagent{
				LastCycle: r.LastCycle, SecuredStock: r.SecuredStock, TypeID: r.TypeId, UnsecuredStock: r.UnsecuredStock,
			}
		}
	}
	if d.ReinforcementTimer != nil {
		out.ReinforcementEnd = &d.ReinforcementTimer.End
	}
	if d.TheftVulnerability != nil {
		out.TheftVulnerabilityStart = &d.TheftVulnerability.Start
		out.TheftVulnerabilityEnd = &d.TheftVulnerability.End
	}
	return out, nil
}

type GetCorporationSkyhookInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	SkyhookID     int64 `json:"skyhookId" jsonschema:"the skyhook ID to look up"`
}

func getCorporationSkyhookHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationSkyhookInput, SkyhookDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationSkyhookInput) (*mcpsdk.CallToolResult, SkyhookDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SkyhookDetail{}, err
		}
		if !found {
			return nil, SkyhookDetail{}, fmt.Errorf("get corporation skyhook for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCorporationSkyhook(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.SkyhookID)
		if err != nil {
			return nil, SkyhookDetail{}, fmt.Errorf("get corporation skyhook: %w", err)
		}
		return nil, detail, nil
	}
}

// --- list_corporation_sovereignty_hubs (private, corp-scoped) ---

type SovereigntyHub struct {
	ID            int64 `json:"id"`
	SolarSystemID int64 `json:"solarSystemId"`
}

var listCorporationSovereigntyHubs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]SovereigntyHub, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	hubs, err := client.CorporationSovereigntyHubs(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]SovereigntyHub, len(hubs))
	for i, h := range hubs {
		out[i] = SovereigntyHub{ID: h.Id, SolarSystemID: h.SolarSystemId}
	}
	return out, nil
}

type ListCorporationSovereigntyHubsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationSovereigntyHubsOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	Hubs          []SovereigntyHub `json:"hubs"`
}

func listCorporationSovereigntyHubsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationSovereigntyHubsInput, ListCorporationSovereigntyHubsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationSovereigntyHubsInput) (*mcpsdk.CallToolResult, ListCorporationSovereigntyHubsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSovereigntyHubsOutput{}, err
		}
		if !found {
			return nil, ListCorporationSovereigntyHubsOutput{}, fmt.Errorf("list corporation sovereignty hubs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		hubs, err := listCorporationSovereigntyHubs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationSovereigntyHubsOutput{}, fmt.Errorf("list corporation sovereignty hubs: %w", err)
		}
		return nil, ListCorporationSovereigntyHubsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Hubs: hubs}, nil
	}
}

// --- get_corporation_sovereignty_hub (private, corp-scoped) ---

type SovereigntyHubReagent struct {
	Amount         int64 `json:"amount"`
	BurningPerHour int64 `json:"burningPerHour"`
	TypeID         int64 `json:"typeId"`
}

type SovereigntyHubReagentBay struct {
	LastUpdated time.Time               `json:"lastUpdated"`
	Reagents    []SovereigntyHubReagent `json:"reagents"`
}

type SovereigntyHubResourceLevel struct {
	Allocated int64 `json:"allocated"`
	Available int64 `json:"available"`
}

type SovereigntyHubResources struct {
	Power     SovereigntyHubResourceLevel `json:"power"`
	Workforce SovereigntyHubResourceLevel `json:"workforce"`
}

type SovereigntyHubUpgrade struct {
	PowerState string `json:"powerState"`
	TypeID     int64  `json:"typeId"`
}

// SovereigntyHubTransportConfig flattens ESI's 3-way oneOf (export/import/transit) for the
// requested workforce transport configuration. Kind is one of "export", "import", "transit".
type SovereigntyHubTransportConfig struct {
	Kind                 string  `json:"kind"`
	ExportAmount         *int64  `json:"exportAmount,omitempty"`
	ExportSolarSystemID  *int64  `json:"exportSolarSystemId,omitempty"`
	ImportSolarSystemIDs []int64 `json:"importSolarSystemIds,omitempty"`
	Transit              *bool  `json:"transit,omitempty"`
}

type SovereigntyHubTransportImportSource struct {
	Amount        int64 `json:"amount"`
	SolarSystemID int64 `json:"solarSystemId"`
}

// SovereigntyHubTransportState flattens ESI's 3-way oneOf (export/import/transit) for the actual
// in-progress workforce transport state. Kind is one of "export", "import", "transit". Note this
// is NOT the same shape as SovereigntyHubTransportConfig's import case: state's import sources
// carry an amount, configuration's do not (verified from the generated model source).
type SovereigntyHubTransportState struct {
	Kind                string                                `json:"kind"`
	ExportAmount        *int64                                `json:"exportAmount,omitempty"`
	ExportSolarSystemID *int64                                `json:"exportSolarSystemId,omitempty"`
	ImportSources       []SovereigntyHubTransportImportSource  `json:"importSources,omitempty"`
	Transit             *bool                                  `json:"transit,omitempty"`
}

type SovereigntyHubTransport struct {
	Configuration SovereigntyHubTransportConfig `json:"configuration"`
	State         SovereigntyHubTransportState  `json:"state"`
}

type SovereigntyHubDetail struct {
	FuelAccessListID         *int64                   `json:"fuelAccessListId,omitempty"`
	ID                       int64                    `json:"id"`
	SolarSystemID            int64                    `json:"solarSystemId"`
	ReagentBay               SovereigntyHubReagentBay `json:"reagentBay"`
	Resources                SovereigntyHubResources  `json:"resources"`
	Upgrades                 []SovereigntyHubUpgrade  `json:"upgrades"`
	VulnerabilityWindowStart *time.Time               `json:"vulnerabilityWindowStart,omitempty"`
	VulnerabilityWindowEnd   *time.Time               `json:"vulnerabilityWindowEnd,omitempty"`
	WorkforceTransport       SovereigntyHubTransport  `json:"workforceTransport"`
}

type GetCorporationSovereigntyHubInput struct {
	CharacterID      int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID    int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	SovereigntyHubID int64 `json:"sovereigntyHubId" jsonschema:"the sovereignty hub ID to look up"`
}

var getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (SovereigntyHubDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return SovereigntyHubDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	d, err := client.CorporationSovereigntyHubDetail(ctx, corporationID, sovereigntyHubID)
	if err != nil {
		return SovereigntyHubDetail{}, err
	}
	out := SovereigntyHubDetail{
		FuelAccessListID: d.FuelAccessListId,
		ID:               d.Id,
		SolarSystemID:    d.SolarSystemId,
		ReagentBay: SovereigntyHubReagentBay{
			LastUpdated: d.ReagentBay.LastUpdated,
		},
		Resources: SovereigntyHubResources{
			Power:     SovereigntyHubResourceLevel{Allocated: d.Resources.Power.Allocated, Available: d.Resources.Power.Available},
			Workforce: SovereigntyHubResourceLevel{Allocated: d.Resources.Workforce.Allocated, Available: d.Resources.Workforce.Available},
		},
	}
	out.ReagentBay.Reagents = make([]SovereigntyHubReagent, len(d.ReagentBay.Reagents))
	for i, r := range d.ReagentBay.Reagents {
		out.ReagentBay.Reagents[i] = SovereigntyHubReagent{Amount: r.Amount, BurningPerHour: r.BurningPerHour, TypeID: r.TypeId}
	}
	out.Upgrades = make([]SovereigntyHubUpgrade, len(d.Upgrades))
	for i, u := range d.Upgrades {
		out.Upgrades[i] = SovereigntyHubUpgrade{PowerState: u.PowerState, TypeID: u.TypeId}
	}
	if d.VulnerabilityWindow != nil {
		out.VulnerabilityWindowStart = &d.VulnerabilityWindow.Start
		out.VulnerabilityWindowEnd = &d.VulnerabilityWindow.End
	}

	cfg := d.WorkforceTransport.Configuration
	switch {
	case cfg.Export != nil && cfg.Export.Export != nil:
		out.WorkforceTransport.Configuration = SovereigntyHubTransportConfig{
			Kind: "export", ExportAmount: &cfg.Export.Export.Amount, ExportSolarSystemID: cfg.Export.Export.SolarSystemId,
		}
	case cfg.ModelImport != nil && cfg.ModelImport.Import != nil:
		ids := make([]int64, len(cfg.ModelImport.Import.Sources))
		for i, s := range cfg.ModelImport.Import.Sources {
			ids[i] = s.SolarSystemId
		}
		out.WorkforceTransport.Configuration = SovereigntyHubTransportConfig{Kind: "import", ImportSolarSystemIDs: ids}
	case cfg.Transit != nil:
		transit := cfg.Transit.Transit.Get()
		out.WorkforceTransport.Configuration = SovereigntyHubTransportConfig{Kind: "transit", Transit: transit}
	}

	st := d.WorkforceTransport.State
	switch {
	case st.Export1 != nil && st.Export1.Export != nil:
		out.WorkforceTransport.State = SovereigntyHubTransportState{
			Kind: "export", ExportAmount: st.Export1.Export.Amount, ExportSolarSystemID: st.Export1.Export.SolarSystemId,
		}
	case st.Import1 != nil && st.Import1.Import != nil:
		sources := make([]SovereigntyHubTransportImportSource, len(st.Import1.Import.Sources))
		for i, s := range st.Import1.Import.Sources {
			sources[i] = SovereigntyHubTransportImportSource{Amount: s.Amount, SolarSystemID: s.SolarSystemId}
		}
		out.WorkforceTransport.State = SovereigntyHubTransportState{Kind: "import", ImportSources: sources}
	case st.Transit1 != nil:
		transit := st.Transit1.Transit.Get()
		out.WorkforceTransport.State = SovereigntyHubTransportState{Kind: "transit", Transit: transit}
	}

	return out, nil
}

func getCorporationSovereigntyHubHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationSovereigntyHubInput, SovereigntyHubDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationSovereigntyHubInput) (*mcpsdk.CallToolResult, SovereigntyHubDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SovereigntyHubDetail{}, err
		}
		if !found {
			return nil, SovereigntyHubDetail{}, fmt.Errorf("get corporation sovereignty hub for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCorporationSovereigntyHub(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.SovereigntyHubID)
		if err != nil {
			return nil, SovereigntyHubDetail{}, fmt.Errorf("get corporation sovereignty hub: %w", err)
		}
		return nil, detail, nil
	}
}
```

Verify directly from `model_corporations_structures_sovereignty_hubs_detail_transportstateexport.go`
whether `Amount`/`SolarSystemId` are `*int64` (both optional) — the plan's Design Verification
section above states both are optional pointers on the *state* export side, vs. the
*configuration* export side where `Amount` is required (`int64`, no pointer) and only
`SolarSystemId` is optional. Get this exactly right per-field; do not assume symmetry between the
configuration and state export shapes.

- [ ] **Step 2: Write `mcp/structures_internal_test.go`**

Follow the swappable-var-override pattern from `mcp/sovereignty_internal_test.go` (read it first
for the exact helper names this package uses, e.g. `newTestStore`/`withCharacter` or equivalent).
Cover:
- `TestListCharacterMercenaryDensHandler_MapsFields` + `_PropagatesError` + character-not-found case
- `TestGetCharacterMercenaryDenHandler_MapsFields` (with reinforcement timer set) +
  `_MapsFields_NoReinforcementTimer` (nil case) + `_PropagatesError`
- `TestListCorporationSkyhooksHandler_MapsFields` + `_PropagatesError`
- `TestGetCorporationSkyhookHandler_MapsFields_FullFixture` (all optional fields populated) +
  `_MapsFields_SparseFixture` (all optional fields nil) + `_PropagatesError`
- `TestListCorporationSovereigntyHubsHandler_MapsFields` + `_PropagatesError`
- `TestGetCorporationSovereigntyHubHandler_MapsFields_ExportVariant`,
  `_MapsFields_ImportVariant`, `_MapsFields_TransitVariant` (asserting each Kind's fields are
  populated correctly AND the other two Kinds' fields are empty/zero -- no cross-variant bleed,
  mirroring `mcp/sovereignty_internal_test.go`'s claim-variant tests) + `_PropagatesError`

For each MapsFields test, swap the package-level `listCharacterMercenaryDens` /
`getCharacterMercenaryDen` / `listCorporationSkyhooks` / `getCorporationSkyhook` /
`listCorporationSovereigntyHubs` / `getCorporationSovereigntyHub` vars with a fake matching the
exact signature declared in Step 1, call the corresponding `*Handler` function, and assert the
returned struct's fields. For PropagatesError tests, make the fake return an error and assert the
handler wraps and returns it. For character-not-found, don't swap the var at all -- call the
handler with a character ID that doesn't exist in a fresh `config.NewStoreAt(t.TempDir() +
"/config.json")` store and assert the error wraps `ErrCharacterNotFound`.

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run 'MercenaryDen|Skyhook|SovereigntyHub' -v`
Expected: all pass.

- [ ] **Step 4: Register the 6 tools in `mcp/server.go`**

First run `grep -c "mcpsdk.AddTool" mcp/server.go` to confirm the real current count (this plan
assumes 173 — if different, that's the real baseline; use it instead for every doc-count edit
below). Read the end of `mcp/server.go` to see the last `AddTool` block (should be `get_status`
from the previous batch) and append after it, following the exact `mcpsdk.AddTool` call shape used
for every other tool in this file:

```go
mcpsdk.AddTool(server, &mcpsdk.Tool{
	Name:        "list_character_mercenary_dens",
	Description: "List the given character's mercenary dens (bare ID + planet ID per den).",
}, listCharacterMercenaryDensHandler(version, clientID, store))

mcpsdk.AddTool(server, &mcpsdk.Tool{
	Name:        "get_character_mercenary_den",
	Description: "Get full detail on one of the given character's mercenary dens: evolution stats, infomorph count, reinforcement timer, and attached skyhook.",
}, getCharacterMercenaryDenHandler(version, clientID, store))

mcpsdk.AddTool(server, &mcpsdk.Tool{
	Name:        "list_corporation_skyhooks",
	Description: "List the given corporation's skyhooks (bare ID + planet ID per skyhook).",
}, listCorporationSkyhooksHandler(version, clientID, store))

mcpsdk.AddTool(server, &mcpsdk.Tool{
	Name:        "get_corporation_skyhook",
	Description: "Get full detail on one of the given corporation's skyhooks: workforce, reagent stock, active/reinforcement state, and theft vulnerability window.",
}, getCorporationSkyhookHandler(version, clientID, store))

mcpsdk.AddTool(server, &mcpsdk.Tool{
	Name:        "list_corporation_sovereignty_hubs",
	Description: "List the given corporation's sovereignty hubs (bare ID + solar system ID per hub).",
}, listCorporationSovereigntyHubsHandler(version, clientID, store))

mcpsdk.AddTool(server, &mcpsdk.Tool{
	Name:        "get_corporation_sovereignty_hub",
	Description: "Get full detail on one of the given corporation's sovereignty hubs: reagent bay, power/workforce resources, upgrades, vulnerability window, and workforce transport configuration/state.",
}, getCorporationSovereigntyHubHandler(version, clientID, store))
```

Also update the package doc comment at the top of `mcp/server.go` -- it currently ends "...
list_sovereignty_systems, and get_status registered." Change the tail so `get_status` is followed
by `, list_character_mercenary_dens, get_character_mercenary_den, list_corporation_skyhooks,
get_corporation_skyhook, list_corporation_sovereignty_hubs, and get_corporation_sovereignty_hub
registered.`

- [ ] **Step 5: Update `mcp/server_test.go`**

Append after `"list_sovereignty_campaigns", "list_sovereignty_systems", "get_status",`:
```go
"list_character_mercenary_dens", "get_character_mercenary_den",
"list_corporation_skyhooks", "get_corporation_skyhook",
"list_corporation_sovereignty_hubs", "get_corporation_sovereignty_hub",
```

- [ ] **Step 6: Run the full test suite**

Run: `go test ./... -race`
Expected: all packages `ok`.

- [ ] **Step 7: Update `CLAUDE.md` and `README.md`**

In both files: change "one hundred seventy-three" to "one hundred seventy-nine" (verify the real
current count first per Step 4's grep -- adjust the word if it's not 173→179), and append the 6
new tool names after `get_status` in the tool list, comma-separated, ending with "... and
`get_corporation_sovereignty_hub`".

- [ ] **Step 8: Flip the roadmap doc's Structures row**

In `docs/reference/esi-coverage-roadmap.md`, change row 32 (`| 32 | Structures | 6 | Planned |`)
to `| 32 | Structures | 6 | This plan |`. Verify with `git diff` that this is the ONLY row changed
in that file.

- [ ] **Step 9: Commit**

```bash
git add mcp/structures.go mcp/structures_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add structures MCP tools (mercenary dens, skyhooks, sovereignty hubs)"
```

- [ ] **Step 10: Push**

```bash
git push origin worktree-esi-access-list-activities
```
