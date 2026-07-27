# ESI Search Implementation Plan (twenty-eighth batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Search" tag (1 endpoint: search for entities matching a
sub-string, scoped to the client's own character) as 1 new MCP tool, folding into the same shared
PR #6 branch as every prior batch.

**Architecture:** This is a small, single-endpoint, private batch — combined into a single
implementation task, following the precedent set by Incursions/Insurance/Routes. `esi/search.go`
wraps the one generated `SearchAPIService` method, "my own character" style. `mcp/search.go`
exposes one private tool, reusing `characterExists`/`ErrCharacterNotFound`.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: 166 MCP tools (once Routes lands — this plan assumes Routes'
  27th-batch commit is already merged), branch `worktree-esi-access-list-activities`, PR #6
  (open). This plan brings the count to 167. **If Routes has not yet landed when this plan is
  executed, adjust the baseline count and the registration anchor point accordingly** (register
  after whatever the actual last tool in `mcp/server.go` is at execution time).
- `GetCharactersCharacterIdSearch` requires OAuth2 scope `esi-search.search_structures.v1`
  (private, "my own character", no caller-supplied character ID) — verified in
  `esi/api/openapi.yaml`. Not paginated.
- `Categories` and `Search` are both **required** at the ESI-request level (the generated `Execute`
  body errors if either is unset) — the facade must always set both. `Categories` accepts 1-11 of
  an 11-value enum (`agent`, `alliance`, `character`, `constellation`, `corporation`, `faction`,
  `inventory_type`, `region`, `solar_system`, `station`, `structure`) — verified against
  `openapi.yaml`'s schema. `Strict` is optional (omitted when false).
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- Reuse `characterExists(store *config.Store, characterID int32) (bool, error)` and
  `ErrCharacterNotFound` from `mcp/character.go` — do not redeclare.
- `SearchResults` is a flat, non-recursive struct of 11 optional `[]int64` fields, mirroring the
  ESI response 1:1 — no shared/reusable sub-types needed since nothing nests.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-search-design.md` for the verification detail.

---

### Task 1: `esi/search.go` + `mcp/search.go` — full batch in one task

**Files:**
- Create: `esi/search.go`
- Test: `esi/search_test.go`
- Create: `mcp/search.go`
- Test: `mcp/search_internal_test.go`
- Modify: `mcp/server.go` (register 1 tool after the last-registered tool at execution time —
  expected to be `get_route` from the Routes batch; update package doc comment)
- Modify: `mcp/server_test.go` (expected-tools list: append `"search_character"`)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Search row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (tool count incremented by 1 from whatever it currently reads, append 1 tool name)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Produces: 1 tool name — `search_character` — registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `esi/search_test.go`:

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

func TestCharacterSearch_SendsQueryParamsAndParsesResponse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"character":[90000001,90000002],"corporation":[98000001]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	results, err := client.CharacterSearch(context.Background(), []string{"character", "corporation"}, "wanted", true)
	if err != nil {
		t.Fatalf("CharacterSearch returned error: %v", err)
	}
	if gotQuery == "" {
		t.Fatal("request query string was empty, want categories/search/strict params")
	}
	if len(results.Character) != 2 || results.Character[0] != 90000001 || results.Character[1] != 90000002 {
		t.Errorf("results.Character = %+v, want [90000001 90000002]", results.Character)
	}
	if len(results.Corporation) != 1 || results.Corporation[0] != 98000001 {
		t.Errorf("results.Corporation = %+v, want [98000001]", results.Corporation)
	}
}

func TestCharacterSearch_OmitsStrictWhenFalse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterSearch(context.Background(), []string{"character"}, "wanted", false); err != nil {
		t.Fatalf("CharacterSearch returned error: %v", err)
	}
	if strictPresent := queryHasKey(gotQuery, "strict"); strictPresent {
		t.Error("strict query param present, want it omitted when strict=false")
	}
}

func queryHasKey(rawQuery, key string) bool {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false
	}
	_, ok := values[key]
	return ok
}

func TestCharacterSearch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterSearch(context.Background(), []string{"character"}, "wanted", false); err == nil {
		t.Fatal("CharacterSearch returned nil error, want an error for HTTP 404")
	}
}
```

Note: `queryHasKey` needs `net/url` imported — add `"net/url"` to the import block alongside the
existing imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterSearch' -v`
Expected: FAIL — `esi.Client` has no method `CharacterSearch`.

- [ ] **Step 3: Write the esi/ implementation**

Create `esi/search.go`:

```go
// Package esi's search.go covers the single "Search" endpoint -- search for entities matching a
// sub-string, scoped to the client's own character (private, "my own character").
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterSearch searches for entities matching search, restricted to categories (1-11 of:
// agent, alliance, character, constellation, corporation, faction, inventory_type, region,
// solar_system, station, structure -- verified against esi/api/openapi.yaml's enum). Both
// categories and search are required at the ESI-request level -- verified from source: the
// generated Execute body errors if either is unset. strict=false requests a substring match;
// strict=true requires an exact match.
func (c *Client) CharacterSearch(ctx context.Context, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
	req := c.api.SearchAPI.GetCharactersCharacterIdSearch(ctx, int64(c.characterID)).Categories(categories).Search(search)
	if strict {
		req = req.Strict(strict)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get character search: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestCharacterSearch' -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Write the failing mcp/ tests**

Create `mcp/search_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestSearchCharacterHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	var gotCategories []string
	var gotSearch string
	var gotStrict bool
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		gotCategories, gotSearch, gotStrict = categories, search, strict
		return &esiapi.CharactersCharacterIdSearchGet{
			Character:   []int64{90000001, 90000002},
			Corporation: []int64{98000001},
		}, nil
	}

	handler := searchCharacterHandler("v1", "client-id", store)
	input := SearchCharacterInput{CharacterID: 7, Categories: []string{"character", "corporation"}, Search: "wanted", Strict: true}
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if len(gotCategories) != 2 || gotCategories[0] != "character" || gotCategories[1] != "corporation" {
		t.Errorf("captured categories = %+v, want [character corporation]", gotCategories)
	}
	if gotSearch != "wanted" {
		t.Errorf("captured search = %q, want \"wanted\"", gotSearch)
	}
	if !gotStrict {
		t.Error("captured strict = false, want true")
	}

	if out.CharacterID != 7 {
		t.Errorf("out.CharacterID = %d, want 7", out.CharacterID)
	}
	if len(out.Results.Character) != 2 || out.Results.Character[0] != 90000001 || out.Results.Character[1] != 90000002 {
		t.Errorf("out.Results.Character = %+v, want [90000001 90000002]", out.Results.Character)
	}
	if len(out.Results.Corporation) != 1 || out.Results.Corporation[0] != 98000001 {
		t.Errorf("out.Results.Corporation = %+v, want [98000001]", out.Results.Corporation)
	}
	if len(out.Results.Agent) != 0 || len(out.Results.Alliance) != 0 || len(out.Results.Constellation) != 0 ||
		len(out.Results.Faction) != 0 || len(out.Results.InventoryType) != 0 || len(out.Results.Region) != 0 ||
		len(out.Results.SolarSystem) != 0 || len(out.Results.Station) != 0 || len(out.Results.Structure) != 0 {
		t.Errorf("out.Results = %+v, want all other category fields empty", out.Results)
	}
}

func TestSearchCharacterHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		t.Fatal("searchCharacter should not be called for an unknown character")
		return nil, nil
	}

	handler := searchCharacterHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, SearchCharacterInput{CharacterID: 999, Categories: []string{"character"}, Search: "wanted"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestSearchCharacterHandler' -v`
Expected: FAIL — undefined symbols (`searchCharacter`, `SearchCharacterInput`, etc. don't exist yet).

- [ ] **Step 7: Write the mcp/ implementation**

Create `mcp/search.go`:

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- search_character (private) ---

type SearchResults struct {
	Agent         []int64 `json:"agent,omitempty"`
	Alliance      []int64 `json:"alliance,omitempty"`
	Character     []int64 `json:"character,omitempty"`
	Constellation []int64 `json:"constellation,omitempty"`
	Corporation   []int64 `json:"corporation,omitempty"`
	Faction       []int64 `json:"faction,omitempty"`
	InventoryType []int64 `json:"inventoryType,omitempty"`
	Region        []int64 `json:"region,omitempty"`
	SolarSystem   []int64 `json:"solarSystem,omitempty"`
	Station       []int64 `json:"station,omitempty"`
	Structure     []int64 `json:"structure,omitempty"`
}

var searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterSearch(ctx, categories, search, strict)
}

type SearchCharacterInput struct {
	CharacterID int32    `json:"characterId" jsonschema:"the character ID to check"`
	Categories  []string `json:"categories" jsonschema:"1-11 of: agent, alliance, character, constellation, corporation, faction, inventory_type, region, solar_system, station, structure"`
	Search      string   `json:"search" jsonschema:"the search string, at least 3 characters"`
	Strict      bool     `json:"strict,omitempty" jsonschema:"require an exact match instead of a substring match; omit for false"`
}
type SearchCharacterOutput struct {
	CharacterID int32         `json:"characterId"`
	Results     SearchResults `json:"results"`
}

func searchCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[SearchCharacterInput, SearchCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SearchCharacterInput) (*mcpsdk.CallToolResult, SearchCharacterOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SearchCharacterOutput{}, err
		}
		if !found {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		results, err := searchCharacter(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Categories, in.Search, in.Strict)
		if err != nil {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character: %w", err)
		}
		return nil, SearchCharacterOutput{CharacterID: in.CharacterID, Results: SearchResults{
			Agent: results.Agent, Alliance: results.Alliance, Character: results.Character,
			Constellation: results.Constellation, Corporation: results.Corporation, Faction: results.Faction,
			InventoryType: results.InventoryType, Region: results.Region, SolarSystem: results.SolarSystem,
			Station: results.Station, Structure: results.Structure,
		}}, nil
	}
}
```

Modify `mcp/server.go`: append after whatever the currently-last-registered tool block is at
execution time (immediately before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "search_character",
		Description: "Search for entities (characters, corporations, systems, etc.) matching a string, scoped to a logged-in character's search permissions.",
	}, searchCharacterHandler(version, clientID, store))
```

Update `mcp/server.go`'s package doc comment: append `search_character` to the trailing tool list
(preserving the "and X registered" grammar — insert before the final "and", make the previously-last
name no longer the final one).

Modify `mcp/server_test.go`: append `"search_character"` to the expected-tools slice, matching the
file's existing membership-check style.

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Search row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: increment the tool count by 1 from whatever it currently reads (spell out the
new number in words, matching the file's existing style), and append `search_character` to the
`mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestSearchCharacterHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (2 new handler tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 9: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, no panics on tool registration.

```bash
git add esi/search.go esi/search_test.go mcp/search.go mcp/search_internal_test.go \
  mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add search MCP tool (search_character: search entities matching a string)"
```
