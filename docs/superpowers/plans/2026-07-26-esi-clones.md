# ESI Clones Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cover the Clones ESI resource group (2 endpoints, both standard character-scoped reads).

**Architecture:** `esi/clones.go` is a thin, `describeError`-wrapped facade, standard private pattern (no public/private split needed here, unlike Character). `mcp/clones.go` adds 2 tools with the standard inline existence-check (not the shared helper — this file has only 2 handlers, well under the threshold that justified `characterExists` in `mcp/character.go`).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context. Never discard an error.
- Every `esi/` method routes errors through `describeError`.
- External test packages by default; standard library only in tests.
- Raw-ID passthrough: implant/location type IDs are not resolved to names.
- Both tools use the standard inline existence-check loop (not `characterExists` — that helper is scoped to `mcp/character.go` only, per the prior batch's plan).
- This branch is the shared PR branch (`worktree-esi-access-list-activities`), current baseline 37 tools.
- Design spec: `docs/superpowers/specs/2026-07-26-esi-clones-design.md`.
- All code verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and `github.com/modelcontextprotocol/go-sdk` v1.6.1 modules.

---

### Task 1: `esi/clones.go` facade

**Files:**
- Create: `esi/clones.go`
- Create: `esi/clones_test.go`

**Interfaces:**
- Consumes: `esi.Client`, `describeError`; `esiapi.APIClient.ClonesAPI.GetCharactersCharacterIdClones(ctx, characterId int64) ...Execute() (*esiapi.CharactersCharacterIdClonesGet, *http.Response, error)`, `.GetCharactersCharacterIdImplants(ctx, characterId int64) ...Execute() ([]int64, *http.Response, error)` — verified against the real installed module.
- Produces: `(*esi.Client).CharacterClones(ctx) (*esiapi.CharactersCharacterIdClonesGet, error)`, `(*esi.Client).CharacterImplants(ctx) ([]int64, error)` — consumed by Task 2.

- [ ] **Step 1: Write the failing tests**

Create `esi/clones_test.go`:

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

func TestCharacterClones_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"home_location":{"location_id":60003760,"location_type":"station"},"jump_clones":[{"implants":[19540],"jump_clone_id":1,"location_id":60003760,"location_type":"station"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	clones, err := client.CharacterClones(context.Background())
	if err != nil {
		t.Fatalf("CharacterClones returned error: %v", err)
	}
	if clones.HomeLocation == nil || clones.HomeLocation.LocationId == nil || *clones.HomeLocation.LocationId != 60003760 {
		t.Errorf("clones.HomeLocation = %+v, want LocationId=*60003760", clones.HomeLocation)
	}
	if len(clones.JumpClones) != 1 || clones.JumpClones[0].JumpCloneId != 1 {
		t.Errorf("clones.JumpClones = %+v, want one entry with JumpCloneId=1", clones.JumpClones)
	}
}

func TestCharacterClones_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterClones(context.Background()); err == nil {
		t.Fatal("CharacterClones returned nil error, want an error for HTTP 403")
	}
}

func TestCharacterImplants_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[19540,19551]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	implants, err := client.CharacterImplants(context.Background())
	if err != nil {
		t.Fatalf("CharacterImplants returned error: %v", err)
	}
	if len(implants) != 2 || implants[0] != 19540 {
		t.Errorf("implants = %v, want [19540 19551]", implants)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterClones|TestCharacterImplants' -v`

Expected: FAIL to compile — undefined methods.

- [ ] **Step 3: Implement `esi/clones.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterClones returns the client's character's jump clones and home location.
func (c *Client) CharacterClones(ctx context.Context) (*esiapi.CharactersCharacterIdClonesGet, error) {
	result, resp, err := c.api.ClonesAPI.GetCharactersCharacterIdClones(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character clones: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterImplants returns the client's character's currently active implant type IDs.
func (c *Client) CharacterImplants(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.ClonesAPI.GetCharactersCharacterIdImplants(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character implants: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all 3 new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/clones.go esi/clones_test.go
git commit -m "feat: add esi/clones.go facade"
```

---

### Task 2: `mcp/clones.go` tools + registration + docs sync

**Files:**
- Create: `mcp/clones.go`
- Create: `mcp/clones_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/server_test.go`
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Modify: `docs/reference/esi-coverage-roadmap.md`

**Interfaces:**
- Consumes: `esi.NewClient`, `(*esi.Client).CharacterClones`, `.CharacterImplants` (Task 1); `sso.UserAgent`, `config.Store.Load`, `ErrCharacterNotFound` (existing).
- Produces: `mcp.GetCharacterClonesInput/Output`, `mcp.JumpClone`, `mcp.ListCharacterImplantsInput/Output`. Nothing later depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/clones_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetCharacterClonesHandler_ReturnsClones(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterClones
	t.Cleanup(func() { getCharacterClones = orig })
	locID := int64(60003760)
	getCharacterClones = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterClonesOutput, error) {
		return GetCharacterClonesOutput{CharacterID: characterID, HomeLocationID: &locID, JumpClones: []JumpClone{{JumpCloneID: 1}}}, nil
	}

	handler := getCharacterClonesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.HomeLocationID == nil || *out.HomeLocationID != 60003760 || len(out.JumpClones) != 1 {
		t.Errorf("out = %+v, want HomeLocationID=*60003760 and one jump clone", out)
	}
}

func TestGetCharacterClonesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterClones
	t.Cleanup(func() { getCharacterClones = orig })
	getCharacterClones = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterClonesOutput, error) {
		t.Fatal("getCharacterClones should not be called for an unknown character")
		return GetCharacterClonesOutput{}, nil
	}

	handler := getCharacterClonesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterImplantsHandler_ReturnsImplants(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterImplants
	t.Cleanup(func() { listCharacterImplants = orig })
	listCharacterImplants = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		return []int64{19540, 19551}, nil
	}

	handler := listCharacterImplantsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.ImplantTypeIDs) != 2 || out.ImplantTypeIDs[0] != 19540 {
		t.Errorf("out = %+v, want ImplantTypeIDs=[19540 19551]", out)
	}
}

func TestListCharacterImplantsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterImplants
	t.Cleanup(func() { listCharacterImplants = orig })
	listCharacterImplants = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		t.Fatal("listCharacterImplants should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterImplantsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestGetCharacterClonesHandler|TestListCharacterImplantsHandler' -v`

Expected: FAIL to compile.

- [ ] **Step 3: Implement `mcp/clones.go`**

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

// JumpClone is one of a character's jump clones.
type JumpClone struct {
	JumpCloneID  int64   `json:"jumpCloneId"`
	LocationID   int64   `json:"locationId"`
	LocationType string  `json:"locationType"`
	Name         *string `json:"name,omitempty"`
	Implants     []int64 `json:"implants"`
}

// GetCharacterClonesOutput is get_character_clones's tool result.
type GetCharacterClonesOutput struct {
	CharacterID           int32       `json:"characterId"`
	HomeLocationID        *int64      `json:"homeLocationId,omitempty"`
	HomeLocationType      *string     `json:"homeLocationType,omitempty"`
	LastCloneJumpDate     *time.Time  `json:"lastCloneJumpDate,omitempty"`
	LastStationChangeDate *time.Time  `json:"lastStationChangeDate,omitempty"`
	JumpClones            []JumpClone `json:"jumpClones"`
}

// getCharacterClones is overridable in tests.
var getCharacterClones = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterClonesOutput, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return GetCharacterClonesOutput{}, fmt.Errorf("build ESI client: %w", err)
	}
	clones, err := client.CharacterClones(ctx)
	if err != nil {
		return GetCharacterClonesOutput{}, err
	}
	out := GetCharacterClonesOutput{
		CharacterID:           characterID,
		LastCloneJumpDate:     clones.LastCloneJumpDate,
		LastStationChangeDate: clones.LastStationChangeDate,
	}
	if clones.HomeLocation != nil {
		out.HomeLocationID = clones.HomeLocation.LocationId
		out.HomeLocationType = clones.HomeLocation.LocationType
	}
	out.JumpClones = make([]JumpClone, len(clones.JumpClones))
	for i, jc := range clones.JumpClones {
		out.JumpClones[i] = JumpClone{JumpCloneID: jc.JumpCloneId, LocationID: jc.LocationId, LocationType: jc.LocationType, Name: jc.Name, Implants: jc.Implants}
	}
	return out, nil
}

// GetCharacterClonesInput is get_character_clones's tool input.
type GetCharacterClonesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

func getCharacterClonesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterClonesInput, GetCharacterClonesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterClonesInput) (*mcpsdk.CallToolResult, GetCharacterClonesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("get character clones for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		out, err := getCharacterClones(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("get character clones: %w", err)
		}
		return nil, out, nil
	}
}

// listCharacterImplants is overridable in tests.
var listCharacterImplants = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterImplants(ctx)
}

// ListCharacterImplantsInput is list_character_implants's tool input.
type ListCharacterImplantsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListCharacterImplantsOutput is list_character_implants's tool result.
type ListCharacterImplantsOutput struct {
	CharacterID    int32   `json:"characterId"`
	ImplantTypeIDs []int64 `json:"implantTypeIds"`
}

func listCharacterImplantsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterImplantsInput, ListCharacterImplantsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterImplantsInput) (*mcpsdk.CallToolResult, ListCharacterImplantsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("list character implants for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		ids, err := listCharacterImplants(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("list character implants: %w", err)
		}
		return nil, ListCharacterImplantsOutput{CharacterID: in.CharacterID, ImplantTypeIDs: ids}, nil
	}
}
```

- [ ] **Step 4: Register the two tools in `mcp/server.go`**

Add these two `mcpsdk.AddTool` calls right after `get_character_cspa_cost`'s (the last tool from
the prior batch), and extend the `NewServer` doc comment to mention them:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_clones",
		Description: "Get a logged-in EVE Online character's jump clones and home location.",
	}, getCharacterClonesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_implants",
		Description: "List a logged-in EVE Online character's currently active implant type IDs.",
	}, listCharacterImplantsHandler(version, clientID, store))
```

- [ ] **Step 5: Update `mcp/server_test.go`'s expected-tools list**

Append `"get_character_clones", "list_character_implants",` to the expected-tools slice (37 → 39).

- [ ] **Step 6: Update docs**

- `CLAUDE.md`/`README.md`: bump tool count from "thirty-seven" to "thirty-nine", append the two
  new tool names to the enumerated list, add "and Clones" to README's covered-tags sentence.
- `docs/reference/esi-coverage-roadmap.md`: change the Clones row from "Planned" to "This plan".

- [ ] **Step 7: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race -v`

Expected: all 4 new tests PASS, `TestServer_AllToolsRegistered` PASSes with all 39 names, `gofmt -l .`
prints nothing, every pre-existing test passes.

- [ ] **Step 8: Commit**

```bash
git add mcp/clones.go mcp/clones_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add character clones/implants MCP tools, update docs and roadmap"
```
