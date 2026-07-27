# ESI Coverage Roadmap + Access List/Activities Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the durable ESI-coverage tracking doc, and ship the first batch of the systematic ESI sweep: the Access List and Activities resource groups (5 endpoints total), following the exact pattern already proven by `esi/wallet.go` / `mcp/wallet.go`.

**Architecture:** `esi/access_list.go` and `esi/activities.go` are thin facades over `github.com/fnt-eve/goesi-openapi`'s `AccessListAPI`/`ActivitiesAPI`, identical in shape to `esi/wallet.go` (per-character `Client`, `describeError`-wrapped errors, zero state of their own). `mcp/access_list.go` and `mcp/activities.go` add five new MCP tools, each checking `config.Store` for the character before calling ESI (matching `get_wallet_balance`'s existence-check convention) and mapping the raw `esiapi` response structs into small, JSON-tagged output DTOs — never resolving any ID to a name (raw-ID passthrough, since `sde/` doesn't exist yet).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context: `fmt.Errorf("operation: %w", err)`. Never discard an error.
- Every `esi/` method that calls `.Execute()` must route its error and `*http.Response` through `describeError` (`esi/error.go`) before wrapping — existing hard rule, verified already in place.
- External test packages by default (`package foo_test`); `*_internal_test.go` only when a test genuinely needs an unexported symbol (the `mcp/` handler tests need this, matching `mcp/wallet_internal_test.go`).
- `keyring.MockInit()` before any test touches keychain-backed code.
- Standard library only in tests — no testify.
- Raw-ID passthrough: no `esi/` or `mcp/` code in this plan resolves any ID (e.g. `mercenary_den_id`, `planet_id`, `solar_system_id`) to a human-readable name. That is out of scope until `sde/` exists.
- No new OAuth scopes needed. `mcp/add_character.go`'s `allScopes` already includes `esi-access.read_lists.v1` and `esi-activities.read_character.v1` — verified present in the current list, no re-auth needed for existing characters.
- Design spec: `docs/superpowers/specs/2026-07-25-esi-coverage-roadmap-design.md`.
- All code in this plan has been verified to compile and its exact JSON shapes verified to unmarshal correctly against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module (field names, method signatures, and full request/response round-trips against a real `httptest.Server`) during plan-writing — not guessed from documentation.

---

### Task 1: ESI coverage roadmap tracking doc

**Files:**
- Create: `docs/reference/esi-coverage-roadmap.md`

**Interfaces:** none — documentation only, no code dependencies.

- [ ] **Step 1: Create the roadmap doc**

Create `docs/reference/esi-coverage-roadmap.md`:

```markdown
# ESI Coverage Roadmap

Tracks progress covering CCP's full ESI API surface, one batch of resource groups at a time. The
tag order and endpoint counts below come directly from CCP's live OpenAPI 3.0 spec
(`https://esi.evetech.net/meta/openapi-3.0.json`), fetched the same way
`github.com/fnt-eve/goesi-openapi` itself generates its client (see
`docs/superpowers/specs/2026-07-25-esi-coverage-roadmap-design.md` for how this was verified).
Each tag's own row is updated to "Done (PR #N)" once its batch merges — this table is the one
place to check what's left, instead of re-deriving the order from the spec each time.

| # | Tag | Endpoints | Status |
|---|-----|-----------|--------|
| 1 | Access List | 2 | This plan |
| 2 | Activities | 3 | This plan |
| 3 | Alliance | 4 | Planned |
| 4 | Assets | 6 | Planned |
| 5 | Calendar | 4 | Planned |
| 6 | Character | 14 | Planned |
| 7 | Clones | 2 | Planned |
| 8 | Contacts | 9 | Planned |
| 9 | Contracts | 9 | Planned |
| 10 | Corporation | 22 | Planned |
| 11 | Corporation Projects | 4 | Planned |
| 12 | Dogma | 5 | Planned |
| 13 | Faction Warfare | 8 | Planned |
| 14 | Fittings | 3 | Planned |
| 15 | Fleets | 14 | Planned |
| 16 | Freelance Jobs | 6 | Planned |
| 17 | Incursions | 1 | Planned |
| 18 | Industry | 8 | Planned |
| 19 | Insurance | 1 | Planned |
| 20 | Killmails | 3 | Planned |
| 21 | Location | 3 | Planned |
| 22 | Loyalty | 2 | Planned |
| 23 | Mail | 9 | Planned |
| 24 | Market | 11 | Planned |
| 25 | Meta | 4 | Planned |
| 26 | Planetary Interaction | 4 | Planned |
| 27 | Routes | 1 | Planned |
| 28 | Search | 1 | Planned |
| 29 | Skills | 3 | Planned |
| 30 | Sovereignty | 2 | Planned |
| 31 | Status | 1 | Planned |
| 32 | Structures | 6 | Planned |
| 33 | Universe | 30 | Planned |
| 34 | User Interface | 5 | Planned |
| 35 | Wallet | 6 | Done (PR #5) |
| 36 | Wars | 3 | Planned |

219 total endpoints across all 36 tags.

## Conventions (apply to every batch)

- One `esi/<tag_snake_case>.go` + one `mcp/<tag_snake_case>.go` per tag, even when a plan batches
  several tags together — only the plan/PR boundary batches tags, file boundaries stay
  per-resource.
- A plan covers a cluster of adjacent tags (in the table's order above) sized so the batch isn't
  trivially small.
- Every `esi/` method returns exactly what ESI returns, including bare IDs, with zero
  name-resolution enrichment — that's a `sde/`-backed follow-up once that package exists.
- Every new batch must verify its required OAuth scopes are already present in
  `mcp/add_character.go`'s `allScopes`, the same way this plan did, before assuming no re-auth is
  needed.
```

- [ ] **Step 2: Commit**

```bash
git add docs/reference/esi-coverage-roadmap.md
git commit -m "docs: add ESI coverage roadmap tracking doc"
```

---

### Task 2: `esi/access_list.go` — Access List facade

**Files:**
- Create: `esi/access_list.go`
- Create: `esi/access_list_test.go`

**Interfaces:**
- Consumes: `esi.Client` (existing, `esi/client.go`), `describeError(err error, resp *http.Response) error` (existing, `esi/error.go`); `esiapi.APIClient.AccessListAPI.GetCharactersAccessListsListing(ctx, characterId int64) ...Execute() (*esiapi.CharactersAccessListsListing, *http.Response, error)` and `.GetCharactersAccessListsDetail(ctx, accessListId int64, characterId int64) ...Execute() (*esiapi.CharactersAccessListsDetail, *http.Response, error)` — both verified against the real installed `goesi-openapi` v0.0.37 module (exact method names, parameter order, and return types confirmed by a real compile + a live `httptest.Server` round-trip during plan-writing).
- Produces: `(*esi.Client).AccessLists(ctx) ([]esiapi.CharactersAccessListsListingAccesslist, error)`, `(*esi.Client).AccessListDetail(ctx, accessListID int64) (*esiapi.CharactersAccessListsDetail, error)` — both consumed by Task 4.

- [ ] **Step 1: Write the failing tests**

Create `esi/access_list_test.go`:

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

func TestAccessLists_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_lists":[{"id":1},{"id":2}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	lists, err := client.AccessLists(context.Background())
	if err != nil {
		t.Fatalf("AccessLists returned error: %v", err)
	}
	if len(lists) != 2 || lists[0].Id != 1 || lists[1].Id != 2 {
		t.Errorf("lists = %+v, want [{Id:1} {Id:2}]", lists)
	}
}

func TestAccessListDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"description":"desc","id":1,"membership":{"alliances":[{"access":"read","alliance_id":99000001}],"allow_everyone":false,"characters":[{"access":"read","character_id":123}],"corporations":[{"access":"read","corporation_id":456}]},"name":"My List"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.AccessListDetail(context.Background(), 1)
	if err != nil {
		t.Fatalf("AccessListDetail returned error: %v", err)
	}
	if detail.Name != "My List" || detail.Description != "desc" {
		t.Errorf("detail = %+v, want Name=\"My List\" Description=\"desc\"", detail)
	}
	if len(detail.Membership.Alliances) != 1 || detail.Membership.Alliances[0].AllianceId != 99000001 {
		t.Errorf("detail.Membership.Alliances = %+v, want one entry with AllianceId 99000001", detail.Membership.Alliances)
	}
	if len(detail.Membership.Characters) != 1 || detail.Membership.Characters[0].CharacterId != 123 {
		t.Errorf("detail.Membership.Characters = %+v, want one entry with CharacterId 123", detail.Membership.Characters)
	}
	if len(detail.Membership.Corporations) != 1 || detail.Membership.Corporations[0].CorporationId != 456 {
		t.Errorf("detail.Membership.Corporations = %+v, want one entry with CorporationId 456", detail.Membership.Corporations)
	}
}

func TestAccessListDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.AccessListDetail(context.Background(), 1); err == nil {
		t.Fatal("AccessListDetail returned nil error, want an error for HTTP 403")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestAccessLists|TestAccessListDetail' -v`

Expected: FAIL to compile — `undefined: (*esi.Client).AccessLists`, `undefined: (*esi.Client).AccessListDetail`.

- [ ] **Step 3: Implement `esi/access_list.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// AccessLists returns the client's character's structure/fleet access lists
// (bare IDs only — call AccessListDetail for name/description/membership).
func (c *Client) AccessLists(ctx context.Context) ([]esiapi.CharactersAccessListsListingAccesslist, error) {
	result, resp, err := c.api.AccessListAPI.GetCharactersAccessListsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get access lists: %w", describeError(err, resp))
	}
	return result.AccessLists, nil
}

// AccessListDetail returns the full detail (description, membership) for one access list.
func (c *Client) AccessListDetail(ctx context.Context, accessListID int64) (*esiapi.CharactersAccessListsDetail, error) {
	result, resp, err := c.api.AccessListAPI.GetCharactersAccessListsDetail(ctx, accessListID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get access list detail: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: `TestAccessLists_ParsesResponse`, `TestAccessListDetail_ParsesResponse`, and
`TestAccessListDetail_PropagatesHTTPError` PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/access_list.go esi/access_list_test.go
git commit -m "feat: add esi/access_list.go facade"
```

---

### Task 3: `esi/activities.go` — Activities facade

**Files:**
- Create: `esi/activities.go`
- Create: `esi/activities_test.go`

**Interfaces:**
- Consumes: `esi.Client`, `describeError` (as Task 2); `esiapi.APIClient.ActivitiesAPI.GetCharactersMercenaryTacticalOperationsListing(ctx, characterId int64) ...Execute() (*esiapi.CharactersMercenaryTacticalOperationsListing, *http.Response, error)`, `.GetCharactersMercenaryTacticalOperationsDetail(ctx, operationId string, characterId int64) ...Execute() (*esiapi.CharactersMercenaryTacticalOperationsDetail, *http.Response, error)`, `.GetSkyhooksRaidable(ctx) ...Execute() (*esiapi.SkyhooksRaidable, *http.Response, error)` — all three verified against the real installed module (note `operationId` is a **string**, confirmed from source, and `GetSkyhooksRaidable` takes no character ID parameter at all).
- Produces: `(*esi.Client).MercenaryTacticalOperations(ctx) ([]esiapi.CharactersMercenaryTacticalOperationsListingOperation, error)`, `(*esi.Client).MercenaryTacticalOperationDetail(ctx, operationID string) (*esiapi.CharactersMercenaryTacticalOperationsDetail, error)`, `(*esi.Client).RaidableSkyhooks(ctx) ([]esiapi.SkyhooksRaidableVulnerableskyhook, error)` — all three consumed by Task 5.

- [ ] **Step 1: Write the failing tests**

Create `esi/activities_test.go`:

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

func TestMercenaryTacticalOperations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"operations":[{"id":"op-1","mercenary_den_id":1000}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ops, err := client.MercenaryTacticalOperations(context.Background())
	if err != nil {
		t.Fatalf("MercenaryTacticalOperations returned error: %v", err)
	}
	if len(ops) != 1 || ops[0].Id != "op-1" || ops[0].MercenaryDenId != 1000 {
		t.Errorf("ops = %+v, want one entry {Id:\"op-1\" MercenaryDenId:1000}", ops)
	}
}

func TestMercenaryTacticalOperationDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"dungeon_type_id":30000,"expires":"2026-08-01T00:00:00Z","id":"op-1","mercenary_den_id":1000,"state":"active"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.MercenaryTacticalOperationDetail(context.Background(), "op-1")
	if err != nil {
		t.Fatalf("MercenaryTacticalOperationDetail returned error: %v", err)
	}
	if detail.DungeonTypeId != 30000 || detail.State != "active" || detail.MercenaryDenId != 1000 {
		t.Errorf("detail = %+v, want DungeonTypeId=30000 State=\"active\" MercenaryDenId=1000", detail)
	}
}

func TestMercenaryTacticalOperationDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.MercenaryTacticalOperationDetail(context.Background(), "op-1"); err == nil {
		t.Fatal("MercenaryTacticalOperationDetail returned nil error, want an error for HTTP 403")
	}
}

func TestRaidableSkyhooks_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"skyhooks":[{"planet_id":40000001,"solar_system_id":30000142,"theft_vulnerability":{"start":"2026-07-26T00:00:00Z","end":"2026-07-27T00:00:00Z"}}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	skyhooks, err := client.RaidableSkyhooks(context.Background())
	if err != nil {
		t.Fatalf("RaidableSkyhooks returned error: %v", err)
	}
	if len(skyhooks) != 1 || skyhooks[0].PlanetId != 40000001 || skyhooks[0].SolarSystemId != 30000142 {
		t.Errorf("skyhooks = %+v, want one entry {PlanetId:40000001 SolarSystemId:30000142}", skyhooks)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestMercenaryTacticalOperations|TestRaidableSkyhooks' -v`

Expected: FAIL to compile — `undefined: (*esi.Client).MercenaryTacticalOperations`, etc.

- [ ] **Step 3: Implement `esi/activities.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// MercenaryTacticalOperations returns the client's character's mercenary tactical operations
// (bare IDs only — call MercenaryTacticalOperationDetail for full detail).
func (c *Client) MercenaryTacticalOperations(ctx context.Context) ([]esiapi.CharactersMercenaryTacticalOperationsListingOperation, error) {
	result, resp, err := c.api.ActivitiesAPI.GetCharactersMercenaryTacticalOperationsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get mercenary tactical operations: %w", describeError(err, resp))
	}
	return result.Operations, nil
}

// MercenaryTacticalOperationDetail returns detail for one mercenary tactical operation.
// operationID is a string in ESI's own model, not neat's usual int32/int64 ID convention.
func (c *Client) MercenaryTacticalOperationDetail(ctx context.Context, operationID string) (*esiapi.CharactersMercenaryTacticalOperationsDetail, error) {
	result, resp, err := c.api.ActivitiesAPI.GetCharactersMercenaryTacticalOperationsDetail(ctx, operationID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get mercenary tactical operation detail: %w", describeError(err, resp))
	}
	return result, nil
}

// RaidableSkyhooks returns upcoming raidable skyhooks. Unauthenticated on ESI's side (no scope
// required, no character ID parameter), but still called through the per-character authenticated
// client for interface consistency with every other esi.Client method.
func (c *Client) RaidableSkyhooks(ctx context.Context) ([]esiapi.SkyhooksRaidableVulnerableskyhook, error) {
	result, resp, err := c.api.ActivitiesAPI.GetSkyhooksRaidable(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get raidable skyhooks: %w", describeError(err, resp))
	}
	return result.Skyhooks, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all four new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/activities.go esi/activities_test.go
git commit -m "feat: add esi/activities.go facade"
```

---

### Task 4: `mcp/access_list.go` tools + registration

**Files:**
- Create: `mcp/access_list.go`
- Create: `mcp/access_list_internal_test.go`
- Modify: `mcp/server.go`

**Interfaces:**
- Consumes: `esi.NewClient(ctx, clientID string, characterID int32, userAgent string) (*esi.Client, error)` (existing), `(*esi.Client).AccessLists`, `(*esi.Client).AccessListDetail` (Task 2); `sso.UserAgent(version string) string` (existing); `config.Store.Load() (config.Config, error)`, `config.Config.Characters []config.Character{ID int32, Name string}` (existing, same shape `mcp/wallet.go` already uses); `ErrCharacterNotFound` (existing, `mcp/remove_character.go`).
- Produces: `mcp.ListAccessListsInput{CharacterID int32}`, `mcp.ListAccessListsOutput{CharacterID int32, AccessListIDs []int64}`, `mcp.GetAccessListInput{CharacterID int32, AccessListID int64}`, `mcp.GetAccessListOutput{...}`, `mcp.AccessListEntry{EntityID int64, Access string}`. Nothing later in this plan depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/access_list_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListAccessListsHandler_ReturnsIDs(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listAccessLists
	t.Cleanup(func() { listAccessLists = orig })
	listAccessLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		return []int64{1, 2}, nil
	}

	handler := listAccessListsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListAccessListsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.AccessListIDs) != 2 || out.AccessListIDs[0] != 1 || out.AccessListIDs[1] != 2 {
		t.Errorf("out = %+v, want {CharacterID:7 AccessListIDs:[1 2]}", out)
	}
}

func TestListAccessListsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listAccessLists
	t.Cleanup(func() { listAccessLists = orig })
	listAccessLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		t.Fatal("listAccessLists should not be called for an unknown character")
		return nil, nil
	}

	handler := listAccessListsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListAccessListsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetAccessListHandler_ReturnsDetail(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getAccessListDetail
	t.Cleanup(func() { getAccessListDetail = orig })
	getAccessListDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, accessListID int64) (AccessListDetailResult, error) {
		return AccessListDetailResult{
			Name:          "My List",
			Description:   "desc",
			AllowEveryone: true,
			Characters:    []AccessListEntry{{EntityID: 123, Access: "read"}},
		}, nil
	}

	handler := getAccessListHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetAccessListInput{CharacterID: 7, AccessListID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Name != "My List" || out.AccessListID != 1 || !out.AllowEveryone {
		t.Errorf("out = %+v, want Name=\"My List\" AccessListID=1 AllowEveryone=true", out)
	}
	if len(out.Characters) != 1 || out.Characters[0].EntityID != 123 {
		t.Errorf("out.Characters = %+v, want one entry with EntityID 123", out.Characters)
	}
}

func TestGetAccessListHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getAccessListDetail
	t.Cleanup(func() { getAccessListDetail = orig })
	getAccessListDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, accessListID int64) (AccessListDetailResult, error) {
		t.Fatal("getAccessListDetail should not be called for an unknown character")
		return AccessListDetailResult{}, nil
	}

	handler := getAccessListHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetAccessListInput{CharacterID: 999, AccessListID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListAccessListsHandler|TestGetAccessListHandler' -v`

Expected: FAIL to compile — `undefined: listAccessLists`, `undefined: listAccessListsHandler`, `undefined: ListAccessListsInput`, `undefined: getAccessListDetail`, `undefined: getAccessListHandler`, `undefined: AccessListDetailResult`, `undefined: AccessListEntry`, `undefined: GetAccessListInput`.

- [ ] **Step 3: Implement `mcp/access_list.go`**

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

// listAccessLists is overridable in tests so no real network/auth runs.
var listAccessLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	lists, err := client.AccessLists(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(lists))
	for i, l := range lists {
		ids[i] = l.Id
	}
	return ids, nil
}

// ListAccessListsInput is list_access_lists's tool input.
type ListAccessListsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListAccessListsOutput is list_access_lists's tool result.
type ListAccessListsOutput struct {
	CharacterID   int32   `json:"characterId"`
	AccessListIDs []int64 `json:"accessListIds"`
}

func listAccessListsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListAccessListsInput, ListAccessListsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAccessListsInput) (*mcpsdk.CallToolResult, ListAccessListsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListAccessListsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListAccessListsOutput{}, fmt.Errorf("list access lists for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		ids, err := listAccessLists(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListAccessListsOutput{}, fmt.Errorf("list access lists: %w", err)
		}
		return nil, ListAccessListsOutput{CharacterID: in.CharacterID, AccessListIDs: ids}, nil
	}
}

// AccessListEntry is one access-grant entry (a character, corporation, or alliance) within an
// access list's membership, mapped from ESI's three separate but identically-shaped entry types.
type AccessListEntry struct {
	EntityID int64  `json:"entityId"`
	Access   string `json:"access"`
}

// AccessListDetailResult is the raw mapped detail returned by getAccessListDetail, before the
// handler adds the characterID/accessListID it already knows from the request.
type AccessListDetailResult struct {
	Name          string
	Description   string
	AllowEveryone bool
	Characters    []AccessListEntry
	Corporations  []AccessListEntry
	Alliances     []AccessListEntry
}

// getAccessListDetail is overridable in tests so no real network/auth runs.
var getAccessListDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, accessListID int64) (AccessListDetailResult, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return AccessListDetailResult{}, fmt.Errorf("build ESI client: %w", err)
	}
	detail, err := client.AccessListDetail(ctx, accessListID)
	if err != nil {
		return AccessListDetailResult{}, err
	}

	characters := make([]AccessListEntry, len(detail.Membership.Characters))
	for i, e := range detail.Membership.Characters {
		characters[i] = AccessListEntry{EntityID: e.CharacterId, Access: e.Access}
	}
	corporations := make([]AccessListEntry, len(detail.Membership.Corporations))
	for i, e := range detail.Membership.Corporations {
		corporations[i] = AccessListEntry{EntityID: e.CorporationId, Access: e.Access}
	}
	alliances := make([]AccessListEntry, len(detail.Membership.Alliances))
	for i, e := range detail.Membership.Alliances {
		alliances[i] = AccessListEntry{EntityID: e.AllianceId, Access: e.Access}
	}

	return AccessListDetailResult{
		Name:          detail.Name,
		Description:   detail.Description,
		AllowEveryone: detail.Membership.AllowEveryone,
		Characters:    characters,
		Corporations:  corporations,
		Alliances:     alliances,
	}, nil
}

// GetAccessListInput is get_access_list's tool input.
type GetAccessListInput struct {
	CharacterID  int32 `json:"characterId" jsonschema:"the character ID to check"`
	AccessListID int64 `json:"accessListId" jsonschema:"the access list ID, from list_access_lists"`
}

// GetAccessListOutput is get_access_list's tool result.
type GetAccessListOutput struct {
	CharacterID   int32             `json:"characterId"`
	AccessListID  int64             `json:"accessListId"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	AllowEveryone bool              `json:"allowEveryone"`
	Characters    []AccessListEntry `json:"characters"`
	Corporations  []AccessListEntry `json:"corporations"`
	Alliances     []AccessListEntry `json:"alliances"`
}

func getAccessListHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetAccessListInput, GetAccessListOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAccessListInput) (*mcpsdk.CallToolResult, GetAccessListOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetAccessListOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetAccessListOutput{}, fmt.Errorf("get access list for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		result, err := getAccessListDetail(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.AccessListID)
		if err != nil {
			return nil, GetAccessListOutput{}, fmt.Errorf("get access list detail: %w", err)
		}
		return nil, GetAccessListOutput{
			CharacterID:   in.CharacterID,
			AccessListID:  in.AccessListID,
			Name:          result.Name,
			Description:   result.Description,
			AllowEveryone: result.AllowEveryone,
			Characters:    result.Characters,
			Corporations:  result.Corporations,
			Alliances:     result.Alliances,
		}, nil
	}
}
```

- [ ] **Step 4: Register the two tools in `mcp/server.go`**

In `mcp/server.go`, change the doc comment and add the two new `mcpsdk.AddTool` calls right after
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
// remove_character, get_wallet_balance, list_access_lists, and
// get_access_list registered.
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
		Name:        "list_access_lists",
		Description: "List a logged-in EVE Online character's structure/fleet access list IDs.",
	}, listAccessListsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_access_list",
		Description: "Get the name, description, and membership detail of one of a character's access lists.",
	}, getAccessListHandler(version, clientID, store))

	return server
}
```

- [ ] **Step 5: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./mcp/... -race -v`

Expected: all four new tests from Step 1 PASS, plus every pre-existing `mcp` test (note
`TestServer_AllToolsRegistered` in `mcp/server_test.go` isn't updated until Task 5 — it only
asserts a subset of tool names are present via a membership check, so it still passes with two
extra tools registered).

- [ ] **Step 6: Commit**

```bash
git add mcp/access_list.go mcp/access_list_internal_test.go mcp/server.go
git commit -m "feat: add list_access_lists and get_access_list MCP tools"
```

---

### Task 5: `mcp/activities.go` tools + registration + `server_test.go` update

**Files:**
- Create: `mcp/activities.go`
- Create: `mcp/activities_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/server_test.go`

**Interfaces:**
- Consumes: `esi.NewClient` (existing), `(*esi.Client).MercenaryTacticalOperations`, `(*esi.Client).MercenaryTacticalOperationDetail`, `(*esi.Client).RaidableSkyhooks` (Task 3); `sso.UserAgent`, `config.Store.Load`, `ErrCharacterNotFound` (as Task 4).
- Produces: `mcp.ListMercenaryTacticalOperationsInput/Output`, `mcp.GetMercenaryTacticalOperationInput/Output`, `mcp.ListRaidableSkyhooksInput/Output`. Nothing later in this plan depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/activities_internal_test.go`:

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

func TestListMercenaryTacticalOperationsHandler_ReturnsOperations(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listMercenaryTacticalOperations
	t.Cleanup(func() { listMercenaryTacticalOperations = orig })
	listMercenaryTacticalOperations = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]OperationSummary, error) {
		return []OperationSummary{{ID: "op-1", MercenaryDenID: 1000}}, nil
	}

	handler := listMercenaryTacticalOperationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListMercenaryTacticalOperationsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Operations) != 1 || out.Operations[0].ID != "op-1" {
		t.Errorf("out = %+v, want CharacterID=7 with one operation ID \"op-1\"", out)
	}
}

func TestListMercenaryTacticalOperationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listMercenaryTacticalOperations
	t.Cleanup(func() { listMercenaryTacticalOperations = orig })
	listMercenaryTacticalOperations = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]OperationSummary, error) {
		t.Fatal("listMercenaryTacticalOperations should not be called for an unknown character")
		return nil, nil
	}

	handler := listMercenaryTacticalOperationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListMercenaryTacticalOperationsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetMercenaryTacticalOperationHandler_ReturnsDetail(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantExpires := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	orig := getMercenaryTacticalOperationDetail
	t.Cleanup(func() { getMercenaryTacticalOperationDetail = orig })
	getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
		return OperationDetailResult{
			DungeonTypeID:  30000,
			MercenaryDenID: 1000,
			State:          "active",
			Expires:        wantExpires,
		}, nil
	}

	handler := getMercenaryTacticalOperationHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 7, OperationID: "op-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.OperationID != "op-1" || out.DungeonTypeID != 30000 || out.State != "active" || !out.Expires.Equal(wantExpires) {
		t.Errorf("out = %+v, want OperationID=\"op-1\" DungeonTypeID=30000 State=\"active\" Expires=%v", out, wantExpires)
	}
}

func TestGetMercenaryTacticalOperationHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getMercenaryTacticalOperationDetail
	t.Cleanup(func() { getMercenaryTacticalOperationDetail = orig })
	getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
		t.Fatal("getMercenaryTacticalOperationDetail should not be called for an unknown character")
		return OperationDetailResult{}, nil
	}

	handler := getMercenaryTacticalOperationHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 999, OperationID: "op-1"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListRaidableSkyhooksHandler_ReturnsSkyhooks(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listRaidableSkyhooks
	t.Cleanup(func() { listRaidableSkyhooks = orig })
	listRaidableSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkyhookSummary, error) {
		return []SkyhookSummary{{PlanetID: 40000001, SolarSystemID: 30000142}}, nil
	}

	handler := listRaidableSkyhooksHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListRaidableSkyhooksInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Skyhooks) != 1 || out.Skyhooks[0].PlanetID != 40000001 {
		t.Errorf("out = %+v, want one skyhook with PlanetID 40000001", out)
	}
}

func TestListRaidableSkyhooksHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listRaidableSkyhooks
	t.Cleanup(func() { listRaidableSkyhooks = orig })
	listRaidableSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkyhookSummary, error) {
		t.Fatal("listRaidableSkyhooks should not be called for an unknown character")
		return nil, nil
	}

	handler := listRaidableSkyhooksHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListRaidableSkyhooksInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListMercenaryTacticalOperationsHandler|TestGetMercenaryTacticalOperationHandler|TestListRaidableSkyhooksHandler' -v`

Expected: FAIL to compile — all the new `mcp` symbols referenced above are undefined.

- [ ] **Step 3: Implement `mcp/activities.go`**

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

// OperationSummary is one mercenary tactical operation's bare-ID summary.
type OperationSummary struct {
	ID             string `json:"id"`
	MercenaryDenID int64  `json:"mercenaryDenId"`
}

// listMercenaryTacticalOperations is overridable in tests.
var listMercenaryTacticalOperations = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]OperationSummary, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	ops, err := client.MercenaryTacticalOperations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OperationSummary, len(ops))
	for i, o := range ops {
		out[i] = OperationSummary{ID: o.Id, MercenaryDenID: o.MercenaryDenId}
	}
	return out, nil
}

// ListMercenaryTacticalOperationsInput is list_mercenary_tactical_operations's tool input.
type ListMercenaryTacticalOperationsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListMercenaryTacticalOperationsOutput is list_mercenary_tactical_operations's tool result.
type ListMercenaryTacticalOperationsOutput struct {
	CharacterID int32              `json:"characterId"`
	Operations  []OperationSummary `json:"operations"`
}

func listMercenaryTacticalOperationsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListMercenaryTacticalOperationsInput, ListMercenaryTacticalOperationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMercenaryTacticalOperationsInput) (*mcpsdk.CallToolResult, ListMercenaryTacticalOperationsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("list mercenary tactical operations for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		ops, err := listMercenaryTacticalOperations(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("list mercenary tactical operations: %w", err)
		}
		return nil, ListMercenaryTacticalOperationsOutput{CharacterID: in.CharacterID, Operations: ops}, nil
	}
}

// OperationDetailResult is the raw mapped detail returned by
// getMercenaryTacticalOperationDetail, before the handler adds the
// characterID/operationID it already knows from the request.
type OperationDetailResult struct {
	DungeonTypeID  int64
	MercenaryDenID int64
	State          string
	Expires        time.Time
}

// getMercenaryTacticalOperationDetail is overridable in tests.
var getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return OperationDetailResult{}, fmt.Errorf("build ESI client: %w", err)
	}
	detail, err := client.MercenaryTacticalOperationDetail(ctx, operationID)
	if err != nil {
		return OperationDetailResult{}, err
	}
	return OperationDetailResult{
		DungeonTypeID:  detail.DungeonTypeId,
		MercenaryDenID: detail.MercenaryDenId,
		State:          detail.State,
		Expires:        detail.Expires,
	}, nil
}

// GetMercenaryTacticalOperationInput is get_mercenary_tactical_operation's tool input.
type GetMercenaryTacticalOperationInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check"`
	OperationID string `json:"operationId" jsonschema:"the operation ID, from list_mercenary_tactical_operations"`
}

// GetMercenaryTacticalOperationOutput is get_mercenary_tactical_operation's tool result.
type GetMercenaryTacticalOperationOutput struct {
	CharacterID    int32     `json:"characterId"`
	OperationID    string    `json:"operationId"`
	DungeonTypeID  int64     `json:"dungeonTypeId"`
	MercenaryDenID int64     `json:"mercenaryDenId"`
	State          string    `json:"state"`
	Expires        time.Time `json:"expires"`
}

func getMercenaryTacticalOperationHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetMercenaryTacticalOperationInput, GetMercenaryTacticalOperationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMercenaryTacticalOperationInput) (*mcpsdk.CallToolResult, GetMercenaryTacticalOperationOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("get mercenary tactical operation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		result, err := getMercenaryTacticalOperationDetail(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.OperationID)
		if err != nil {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("get mercenary tactical operation detail: %w", err)
		}
		return nil, GetMercenaryTacticalOperationOutput{
			CharacterID:    in.CharacterID,
			OperationID:    in.OperationID,
			DungeonTypeID:  result.DungeonTypeID,
			MercenaryDenID: result.MercenaryDenID,
			State:          result.State,
			Expires:        result.Expires,
		}, nil
	}
}

// SkyhookSummary is one upcoming raidable skyhook.
type SkyhookSummary struct {
	PlanetID                int64     `json:"planetId"`
	SolarSystemID           int64     `json:"solarSystemId"`
	TheftVulnerabilityStart time.Time `json:"theftVulnerabilityStart"`
	TheftVulnerabilityEnd   time.Time `json:"theftVulnerabilityEnd"`
}

// listRaidableSkyhooks is overridable in tests.
var listRaidableSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkyhookSummary, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	skyhooks, err := client.RaidableSkyhooks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SkyhookSummary, len(skyhooks))
	for i, s := range skyhooks {
		out[i] = SkyhookSummary{
			PlanetID:                s.PlanetId,
			SolarSystemID:           s.SolarSystemId,
			TheftVulnerabilityStart: s.TheftVulnerability.Start,
			TheftVulnerabilityEnd:   s.TheftVulnerability.End,
		}
	}
	return out, nil
}

// ListRaidableSkyhooksInput is list_raidable_skyhooks's tool input. It still takes a
// characterId, purely for the existence check — the underlying ESI call needs no character
// scope at all (RaidableSkyhooks is public data).
type ListRaidableSkyhooksInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListRaidableSkyhooksOutput is list_raidable_skyhooks's tool result.
type ListRaidableSkyhooksOutput struct {
	Skyhooks []SkyhookSummary `json:"skyhooks"`
}

func listRaidableSkyhooksHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListRaidableSkyhooksInput, ListRaidableSkyhooksOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListRaidableSkyhooksInput) (*mcpsdk.CallToolResult, ListRaidableSkyhooksOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("list raidable skyhooks for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		skyhooks, err := listRaidableSkyhooks(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("list raidable skyhooks: %w", err)
		}
		return nil, ListRaidableSkyhooksOutput{Skyhooks: skyhooks}, nil
	}
}
```

- [ ] **Step 4: Register the three tools in `mcp/server.go`**

In `mcp/server.go`, change the doc comment and add the three new `mcpsdk.AddTool` calls right
after `get_access_list`'s. Change:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, get_wallet_balance, list_access_lists, and
// get_access_list registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
```

to:

```go
// NewServer builds neat's MCP server with add_character, list_characters,
// remove_character, get_wallet_balance, list_access_lists, get_access_list,
// list_mercenary_tactical_operations, get_mercenary_tactical_operation, and
// list_raidable_skyhooks registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
```

And change:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_access_list",
		Description: "Get the name, description, and membership detail of one of a character's access lists.",
	}, getAccessListHandler(version, clientID, store))

	return server
}
```

to:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_access_list",
		Description: "Get the name, description, and membership detail of one of a character's access lists.",
	}, getAccessListHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_mercenary_tactical_operations",
		Description: "List a logged-in EVE Online character's mercenary tactical operations.",
	}, listMercenaryTacticalOperationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_mercenary_tactical_operation",
		Description: "Get detail (dungeon type, mercenary den, state, expiry) for one mercenary tactical operation.",
	}, getMercenaryTacticalOperationHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_raidable_skyhooks",
		Description: "List upcoming raidable skyhooks (public data, no character scope required).",
	}, listRaidableSkyhooksHandler(version, clientID, store))

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
		"list_access_lists", "get_access_list",
		"list_mercenary_tactical_operations", "get_mercenary_tactical_operation", "list_raidable_skyhooks",
	} {
```

- [ ] **Step 6: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./... -race -v`

Expected: every new test from this task and Task 4 PASSes, `TestServer_AllToolsRegistered` PASSes
with all 9 tool names present, and every pre-existing test across the whole module still passes.

- [ ] **Step 7: Commit**

```bash
git add mcp/activities.go mcp/activities_internal_test.go mcp/server.go mcp/server_test.go
git commit -m "feat: add mercenary tactical operations and raidable skyhooks MCP tools"
```

---

### Task 6: Docs sync (`CLAUDE.md`, `README.md`)

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Interfaces:** none — documentation only.

- [ ] **Step 1: Update `CLAUDE.md`'s package table**

Replace:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its four tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance` |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi` — builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client. Holds no state of its own |

More packages (`sde/`) land in follow-up plans — see `docs/superpowers/plans/`.
```

with:

```markdown
| `mcp/` | MCP server (official `github.com/modelcontextprotocol/go-sdk`) and its nine tools: `add_character`, `list_characters`, `remove_character`, `get_wallet_balance`, `list_access_lists`, `get_access_list`, `list_mercenary_tactical_operations`, `get_mercenary_tactical_operation`, `list_raidable_skyhooks` |
| `esi/` | Thin facade over `github.com/fnt-eve/goesi-openapi` — builds a per-character authenticated ESI API client from `sso.AuthenticatedClient`'s cached, auto-refreshing HTTP client. Holds no state of its own. Covers one ESI resource group ("tag") per file (`wallet.go`, `access_list.go`, `activities.go`, ...) — see `docs/reference/esi-coverage-roadmap.md` for what's covered and what's left across all 36 tags |

More packages (`sde/`) land in follow-up plans — see `docs/superpowers/plans/`.
```

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
and real ESI-backed domain tools (`esi/`) covering Wallet, Access List, and Activities are built.
See `docs/reference/esi-coverage-roadmap.md` for full ESI-coverage progress. The SDE cache and
plugin/extension packaging don't exist yet. See the plans directory for what's next.
```

Update the package table the same way as `CLAUDE.md`'s Step 1 (tool count, `esi/` row).

- [ ] **Step 3: Sanity-check the docs against the code**

Run: `grep -n "esi-coverage-roadmap\|list_access_lists\|list_raidable_skyhooks" CLAUDE.md README.md`

Expected: both files mention the roadmap doc and the new tool names.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: document access_list/activities tools and the ESI coverage roadmap"
```
