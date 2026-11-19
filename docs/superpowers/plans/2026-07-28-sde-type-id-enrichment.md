# SDE-backed type_id Enrichment (Batch 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a resolved `*Name` sibling field next to every `type_id`-like field across the ~20 non-Universe `mcp/*.go` tools that carry one, backed by `sde/`'s already-synced local cache.

**Architecture:** One shared helper (`mcp/enrich.go`'s `resolveTypeNames`) that dedups a set of type IDs and resolves each via the existing `sde.Store.Type()` read-through accessor; each affected tool's **handler** function (not its data-fetch var) calls it once after fetching ESI data and fills in the new field(s) from the returned map. This means no existing data-fetch var's signature or tests change — only handler constructors gain a `sdeStore *sde.Store` parameter, threaded from `mcp/server.go`'s existing `sdeStore` parameter.

**Tech Stack:** Go, `github.com/modelcontextprotocol/go-sdk/mcp`, `go.etcd.io/bbolt` (via `sde/`), standard library `testing` only.

## Global Constraints

- Every new field is additive (`FooTypeID int64` gains `FooTypeName string` with `json:"...,omitempty"`) — no existing field is removed or changes type. Two naming exceptions where the source is a bare ID list: `SearchResults.InventoryType []int64` → `InventoryTypeNames []string`; `ListMarketTypesOutput.TypeIDs []int64` → `TypeNames []string`.
- Input-echo fields are excluded: `create_character_fitting`'s request-body types, `open_market_details_window`'s input `TypeID`, and `get_market_history`'s `GetMarketHistoryOutput.TypeID` (which is `in.TypeID` echoed back, not per-record ESI data) get **no** enrichment.
- A lookup miss or ESI-fallback error is silently omitted (empty `TypeName`), never surfaced as a tool error — enrichment is additive, not required.
- Every affected handler constructor gains one new trailing parameter, `sdeStore *sde.Store`, and its call site in `mcp/server.go` passes the `sdeStore` parameter `NewServer` already receives.
- `go build ./...`, `go vet ./...`, `gofmt -l .`, and `go test ./... -race` must stay clean after every task.
- Tests for unexported symbols go in the existing `*_internal_test.go` file for that package (per this repo's convention); do not create `_test.go` (external package) files for these.

---

### Task 1: `mcp/enrich.go` — shared `resolveTypeNames` helper

**Files:**
- Create: `mcp/enrich.go`
- Create: `mcp/enrich_internal_test.go`

**Interfaces:**
- Produces: `typeNameCache` interface (`Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)`) and `resolveTypeNames(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string` — every later task calls this exact signature.
- Note: `mcp/universe.go` already defines a package-level `typeCache` interface, but it has two extra methods (`PutType`, `TypeIDs`) that Universe's write-back/bulk-listing needs and this batch doesn't. Reusing it verbatim would force every later task's test fake to stub two unused methods. `typeNameCache` is a new, narrower interface — `*sde.Store` (and `universe.go`'s existing `fakeTypeCache`) already satisfy it structurally, so nothing about `typeCache` changes.

- [ ] **Step 1: Write the failing tests**

```go
// mcp/enrich_internal_test.go
package mcp

import (
	"context"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

type fakeTypeNameCache struct {
	types map[int64]sde.Type
	errs  map[int64]error
	calls map[int64]int
}

func (f *fakeTypeNameCache) Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Type{}, false, err
	}
	t, ok := f.types[id]
	return t, ok, nil
}

func TestResolveTypeNames_Dedup(t *testing.T) {
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter"}},
		588: {ID: 588, Name: map[string]string{"en": "Slasher"}},
	}}

	got := resolveTypeNames(context.Background(), cache, "test-agent", []int64{587, 587, 588})

	if got[587] != "Rifter" {
		t.Errorf("got[587] = %q, want %q", got[587], "Rifter")
	}
	if got[588] != "Slasher" {
		t.Errorf("got[588] = %q, want %q", got[588], "Slasher")
	}
	if cache.calls[587] != 1 {
		t.Errorf("calls[587] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[587])
	}
}

func TestResolveTypeNames_MissOmitted(t *testing.T) {
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{}}

	got := resolveTypeNames(context.Background(), cache, "test-agent", []int64{999})

	if _, ok := got[999]; ok {
		t.Errorf("got[999] present = %v, want omitted on a cache miss", got[999])
	}
}

func TestResolveTypeNames_ErrorOmitted(t *testing.T) {
	cache := &fakeTypeNameCache{errs: map[int64]error{123: context.DeadlineExceeded}}

	got := resolveTypeNames(context.Background(), cache, "test-agent", []int64{123})

	if _, ok := got[123]; ok {
		t.Errorf("got[123] present = %v, want omitted when the lookup errors", got[123])
	}
}

func TestResolveTypeNames_EmptyIDs(t *testing.T) {
	cache := &fakeTypeNameCache{}

	got := resolveTypeNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run TestResolveTypeNames -v`
Expected: FAIL — `resolveTypeNames`/`typeNameCache`/`fakeTypeNameCache` undefined (nothing implemented yet).

- [ ] **Step 3: Implement `resolveTypeNames`**

```go
// mcp/enrich.go
package mcp

import (
	"context"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// typeNameCache is the minimal read-only view of sde.Store.Type() that
// resolveTypeNames needs -- narrower than universe.go's typeCache, which
// also exposes PutType/TypeIDs for the Universe tag's own write-back and
// bulk-listing needs that this batch's tools don't use.
type typeNameCache interface {
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
}

// resolveTypeNames looks up names for a set of type IDs, deduplicating so a
// value repeated across many records (e.g. the same item type on hundreds of
// asset rows) costs one lookup, not one per occurrence. A lookup failure --
// whether a genuine cache miss or a transient ESI error on the read-through
// fallback path -- is dropped from the result rather than surfacing an
// error: enrichment is additive, not required, so one bad ID must not fail
// an entire tool call over data ESI already returned successfully.
func resolveTypeNames(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		t, ok, err := sdeStore.Type(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = t.Name["en"]
	}
	return names
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run TestResolveTypeNames -v`
Expected: PASS (all four subtests).

- [ ] **Step 5: Commit**

```bash
git add mcp/enrich.go mcp/enrich_internal_test.go
git commit -m "feat: add resolveTypeNames sde-backed type-name resolver"
```

---

### Task 2: `mcp/assets.go` — `AssetItem.TypeID`

**Files:**
- Modify: `mcp/assets.go` (struct at lines 15-24; handlers `listAssetsHandler` lines 61-87, `listCorporationAssetsHandler` lines 246-272)
- Modify: `mcp/server.go` (call sites for `list_assets`, `list_corporation_assets`)
- Test: `mcp/assets_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames(ctx, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string` (Task 1).

- [ ] **Step 1: Add the field**

In `mcp/assets.go`, change `AssetItem`:

```go
type AssetItem struct {
	ItemID          int64  `json:"itemId"`
	TypeID          int64  `json:"typeId"`
	TypeName        string `json:"typeName,omitempty"`
	LocationID      int64  `json:"locationId"`
	LocationType    string `json:"locationType"`
	LocationFlag    string `json:"locationFlag"`
	Quantity        int64  `json:"quantity"`
	IsSingleton     bool   `json:"isSingleton"`
	IsBlueprintCopy *bool  `json:"isBlueprintCopy,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/assets_internal_test.go
func TestListAssetsHandler_ResolvesTypeNames(t *testing.T) {
	origList, origCfg := listAssets, mockCharacterStore(t, 1001)
	t.Cleanup(func() { listAssets = origList })
	listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]AssetItem, int32, error) {
		return []AssetItem{{ItemID: 1, TypeID: 587}, {ItemID: 2, TypeID: 587}}, 1, nil
	}
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter"}}}}

	handler := listAssetsHandler("v1", "client", origCfg, cache)
	_, out, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	for _, a := range out.Assets {
		if a.TypeName != "Rifter" {
			t.Errorf("Assets[itemId=%d].TypeName = %q, want %q", a.ItemID, a.TypeName, "Rifter")
		}
	}
}
```

Note: if this repo's `assets_internal_test.go` doesn't already have a `mockCharacterStore(t, characterID) *config.Store` helper, use whatever existing helper that file already uses to build a `*config.Store` containing character 1001 (check the file's other tests for the actual helper name/pattern before writing this — every existing `TestListAssetsHandler_*` test needs one).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListAssetsHandler_ResolvesTypeNames -v`
Expected: FAIL — `listAssetsHandler` doesn't accept a 4th argument yet (compile error).

- [ ] **Step 4: Implement enrichment in both handlers**

In `mcp/assets.go`, change `listAssetsHandler`'s signature and body:

```go
func listAssetsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListAssetsInput, ListAssetsOutput] {
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
		assets, totalPages, err := listAssets(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListAssetsOutput{}, fmt.Errorf("list assets: %w", err)
		}
		ids := make([]int64, len(assets))
		for i, a := range assets {
			ids[i] = a.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range assets {
			assets[i].TypeName = names[assets[i].TypeID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListAssetsOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Assets: assets}, nil
	}
}
```

Apply the identical shape (fetch → build `ids` → `resolveTypeNames` → loop-assign `TypeName`, inserted right after the existing `listCorporationAssets(...)` call and before the existing `page := in.Page` block) to `listCorporationAssetsHandler`, adding `sdeStore *sde.Store` as its 4th parameter too.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/assets.go`'s imports.

In `mcp/server.go`, update both call sites:
```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_assets",
		Description: "List a logged-in EVE Online character's assets.",
	}, listAssetsHandler(version, clientID, store, sdeStore))
```
```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_assets",
		Description: "List a corporation's assets, using a logged-in character's roles to authorize.",
	}, listCorporationAssetsHandler(version, clientID, store, sdeStore))
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListAssetsHandler_ResolvesTypeNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: all clean — confirms the existing `TestListAssetsHandler_MapsFields`/`_PropagatesError`/etc. tests still compile against the new 4-arg handler signature (they'll need a `cache` argument too — a `&fakeTypeNameCache{}` zero-value is fine for tests that don't care about names).

- [ ] **Step 7: Commit**

```bash
git add mcp/assets.go mcp/assets_internal_test.go mcp/server.go
git commit -m "feat: resolve AssetItem.TypeName via sde/"
```

---

### Task 3: `mcp/character.go` — `Blueprint.TypeID`, `AgentResearch.SkillTypeID`

**Files:**
- Modify: `mcp/character.go` (`Blueprint` struct lines 248-257, `listCharacterBlueprintsHandler` lines 286-305; `AgentResearch` struct lines 197-203, `listCharacterAgentsResearchHandler` lines 229-244)
- Modify: `mcp/server.go` (call sites for `list_character_blueprints`, `list_character_agents_research`)
- Test: `mcp/character_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type AgentResearch struct {
	AgentID         int64     `json:"agentId"`
	SkillTypeID     int64     `json:"skillTypeId"`
	SkillTypeName   string    `json:"skillTypeName,omitempty"`
	PointsPerDay    float64   `json:"pointsPerDay"`
	RemainderPoints float64   `json:"remainderPoints"`
	StartedAt       time.Time `json:"startedAt"`
}
```
```go
type Blueprint struct {
	ItemID             int64  `json:"itemId"`
	TypeID             int64  `json:"typeId"`
	TypeName           string `json:"typeName,omitempty"`
	LocationID         int64  `json:"locationId"`
	LocationFlag       string `json:"locationFlag"`
	Quantity           int64  `json:"quantity"`
	Runs               int64  `json:"runs"`
	MaterialEfficiency int64  `json:"materialEfficiency"`
	TimeEfficiency     int64  `json:"timeEfficiency"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/character_internal_test.go
func TestListCharacterAgentsResearchHandler_ResolvesSkillTypeName(t *testing.T) {
	origList := listCharacterAgentsResearch
	t.Cleanup(func() { listCharacterAgentsResearch = origList })
	listCharacterAgentsResearch = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AgentResearch, error) {
		return []AgentResearch{{AgentID: 1, SkillTypeID: 3403}}, nil
	}
	store := /* existing helper for a *config.Store containing character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{3403: {ID: 3403, Name: map[string]string{"en": "Mining"}}}}

	handler := listCharacterAgentsResearchHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListCharacterAgentsResearchInput{CharacterID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Agents[0].SkillTypeName != "Mining" {
		t.Errorf("SkillTypeName = %q, want %q", out.Agents[0].SkillTypeName, "Mining")
	}
}

func TestListCharacterBlueprintsHandler_ResolvesTypeName(t *testing.T) {
	origList := listCharacterBlueprints
	t.Cleanup(func() { listCharacterBlueprints = origList })
	listCharacterBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Blueprint, int32, error) {
		return []Blueprint{{ItemID: 1, TypeID: 587}}, 1, nil
	}
	store := /* existing helper for a *config.Store containing character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter Blueprint"}}}}

	handler := listCharacterBlueprintsHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Blueprints[0].TypeName != "Rifter Blueprint" {
		t.Errorf("TypeName = %q, want %q", out.Blueprints[0].TypeName, "Rifter Blueprint")
	}
}
```

Replace the `/* existing helper ... */` comments with whatever this file's existing tests already use to build a populated `*config.Store` — check e.g. `TestListCharacterBlueprintsHandler_MapsFields` for the exact pattern before writing these.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestListCharacterAgentsResearchHandler_ResolvesSkillTypeName|TestListCharacterBlueprintsHandler_ResolvesTypeName" -v`
Expected: FAIL — handler signatures don't accept a 4th argument yet.

- [ ] **Step 4: Implement enrichment**

In `listCharacterAgentsResearchHandler`, change its signature to add `sdeStore *sde.Store`, and after `agents, err := listCharacterAgentsResearch(...)` succeeds, insert:
```go
		ids := make([]int64, len(agents))
		for i, a := range agents {
			ids[i] = a.SkillTypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range agents {
			agents[i].SkillTypeName = names[agents[i].SkillTypeID]
		}
```
before the existing `return nil, ListCharacterAgentsResearchOutput{...}` line.

In `listCharacterBlueprintsHandler`, change its signature to add `sdeStore *sde.Store`, and after `bps, totalPages, err := listCharacterBlueprints(...)` succeeds, insert the same shape (`ids` from `bps[i].TypeID`, `resolveTypeNames`, assign `bps[i].TypeName`) before the existing `page := in.Page` block.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/character.go`'s imports.

In `mcp/server.go`, update both call sites to pass `sdeStore` as the trailing argument (`listCharacterAgentsResearchHandler(version, clientID, store, sdeStore)`, `listCharacterBlueprintsHandler(version, clientID, store, sdeStore)`).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestListCharacterAgentsResearchHandler_ResolvesSkillTypeName|TestListCharacterBlueprintsHandler_ResolvesTypeName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean (update the existing `TestListCharacterAgentsResearchHandler_*`/`TestListCharacterBlueprintsHandler_*` call sites to pass a `&fakeTypeNameCache{}` 4th argument).

- [ ] **Step 7: Commit**

```bash
git add mcp/character.go mcp/character_internal_test.go mcp/server.go
git commit -m "feat: resolve AgentResearch.SkillTypeName and Blueprint.TypeName via sde/"
```

---

### Task 4: `mcp/corporation.go` — six handlers

**Files:**
- Modify: `mcp/corporation.go` (`CorporationBlueprint` lines 689-698 / `listCorporationBlueprintsHandler` lines 733-752; `ContainerLog` lines 756-769 / `listCorporationContainerLogsHandler` lines 805-824; `Facility` lines 828-832 / `listCorporationFacilitiesHandler` lines 860-875; `MemberTracking` lines 296-304 / `listCorporationMemberTrackingHandler` lines 332-347; `Starbase` lines 1000-1009 / `listCorporationStarbasesHandler`; `StarbaseFuel`/`StarbaseDetail` / `getCorporationStarbaseHandler`; `Structure` / `listCorporationStructuresHandler`)
- Modify: `mcp/server.go` (six call sites)
- Test: `mcp/corporation_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

This task is one file but six independent handlers; each sub-step below is its own bite-sized add-field/test/implement cycle, ending in one combined verify + commit for the whole file (all six touch the same file, so one commit is more coherent than six tiny ones).

- [ ] **Step 1: Add all six fields**

```go
type CorporationBlueprint struct {
	ItemID             int64  `json:"itemId"`
	LocationFlag       string `json:"locationFlag"`
	LocationID         int64  `json:"locationId"`
	MaterialEfficiency int64  `json:"materialEfficiency"`
	Quantity           int64  `json:"quantity"`
	Runs               int64  `json:"runs"`
	TimeEfficiency     int64  `json:"timeEfficiency"`
	TypeID             int64  `json:"typeId"`
	TypeName           string `json:"typeName,omitempty"`
}
```
```go
type ContainerLog struct {
	Action           string    `json:"action"`
	CharacterID      int64     `json:"characterId"`
	ContainerID      int64     `json:"containerId"`
	ContainerTypeID  int64     `json:"containerTypeId"`
	ContainerTypeName string   `json:"containerTypeName,omitempty"`
	LocationFlag     string    `json:"locationFlag"`
	LocationID       int64     `json:"locationId"`
	LoggedAt         time.Time `json:"loggedAt"`
	NewConfigBitmask *int64    `json:"newConfigBitmask,omitempty"`
	OldConfigBitmask *int64    `json:"oldConfigBitmask,omitempty"`
	PasswordType     *string   `json:"passwordType,omitempty"`
	Quantity         *int64    `json:"quantity,omitempty"`
	TypeID           *int64    `json:"typeId,omitempty"`
	TypeName         string    `json:"typeName,omitempty"`
}
```
```go
type Facility struct {
	FacilityID int64  `json:"facilityId"`
	SystemID   int64  `json:"systemId"`
	TypeID     int64  `json:"typeId"`
	TypeName   string `json:"typeName,omitempty"`
}
```
```go
type MemberTracking struct {
	CharacterID  int64      `json:"characterId"`
	BaseID       *int64     `json:"baseId,omitempty"`
	LocationID   *int64     `json:"locationId,omitempty"`
	ShipTypeID   *int64     `json:"shipTypeId,omitempty"`
	ShipTypeName string     `json:"shipTypeName,omitempty"`
	LogonDate    *time.Time `json:"logonDate,omitempty"`
	LogoffDate   *time.Time `json:"logoffDate,omitempty"`
	StartDate    *time.Time `json:"startDate,omitempty"`
}
```
```go
type Starbase struct {
	MoonID          *int64     `json:"moonId,omitempty"`
	OnlinedSince    *time.Time `json:"onlinedSince,omitempty"`
	ReinforcedUntil *time.Time `json:"reinforcedUntil,omitempty"`
	StarbaseID      int64      `json:"starbaseId"`
	State           *string    `json:"state,omitempty"`
	SystemID        int64      `json:"systemId"`
	TypeID          int64      `json:"typeId"`
	TypeName        string     `json:"typeName,omitempty"`
	UnanchorAt      *time.Time `json:"unanchorAt,omitempty"`
}
```
```go
type StarbaseFuel struct {
	Quantity int64  `json:"quantity"`
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
}
```
```go
type Structure struct {
	CorporationID      int64              `json:"corporationId"`
	FuelExpires        *time.Time         `json:"fuelExpires,omitempty"`
	Name               *string            `json:"name,omitempty"`
	NextReinforceApply *time.Time         `json:"nextReinforceApply,omitempty"`
	NextReinforceHour  *int64             `json:"nextReinforceHour,omitempty"`
	ProfileID          int64              `json:"profileId"`
	ReinforceHour      *int64             `json:"reinforceHour,omitempty"`
	Services           []StructureService `json:"services,omitempty"`
	State              string             `json:"state"`
	StateTimerEnd      *time.Time         `json:"stateTimerEnd,omitempty"`
	StateTimerStart    *time.Time         `json:"stateTimerStart,omitempty"`
	StructureID        int64              `json:"structureId"`
	SystemID           int64              `json:"systemId"`
	TypeID             int64              `json:"typeId"`
	TypeName           string             `json:"typeName,omitempty"`
	UnanchorsAt        *time.Time         `json:"unanchorsAt,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests** (one per handler — table below gives the fetch stub and assertion for each; follow `TestListAssetsHandler_ResolvesTypeNames` from Task 2 as the template, substituting each handler's own types/input)

| Test name | Data-fetch stub returns | Handler call | Assertion |
|---|---|---|---|
| `TestListCorporationBlueprintsHandler_ResolvesTypeName` | `[]CorporationBlueprint{{ItemID: 1, TypeID: 587}}, 1, nil` via `listCorporationBlueprints` | `listCorporationBlueprintsHandler("v1", "client", store, cache)` | `out.Blueprints[0].TypeName == "Rifter"` |
| `TestListCorporationContainerLogsHandler_ResolvesTypeNames` | `[]ContainerLog{{ContainerTypeID: 588, TypeID: ptrInt64(587)}}, 1, nil` via `listCorporationContainerLogs` | `listCorporationContainerLogsHandler("v1", "client", store, cache)` | `out.Logs[0].ContainerTypeName == "Slasher"` and `out.Logs[0].TypeName == "Rifter"` (stub cache with both 587 and 588) |
| `TestListCorporationFacilitiesHandler_ResolvesTypeName` | `[]Facility{{FacilityID: 1, TypeID: 587}}, nil` via `listCorporationFacilities` | `listCorporationFacilitiesHandler("v1", "client", store, cache)` | `out.Facilities[0].TypeName == "Rifter"` |
| `TestListCorporationMemberTrackingHandler_ResolvesShipTypeName` | `[]MemberTracking{{CharacterID: 1, ShipTypeID: ptrInt64(587)}}, nil` via `listCorporationMemberTracking` | `listCorporationMemberTrackingHandler("v1", "client", store, cache)` | `out.Members[0].ShipTypeName == "Rifter"` |
| `TestListCorporationStarbasesHandler_ResolvesTypeName` | `[]Starbase{{StarbaseID: 1, TypeID: 587}}, 1, nil` via `listCorporationStarbases` | `listCorporationStarbasesHandler("v1", "client", store, cache)` | `out.Starbases[0].TypeName == "Rifter"` |
| `TestGetCorporationStarbaseHandler_ResolvesFuelTypeNames` | `StarbaseDetail{Fuels: []StarbaseFuel{{TypeID: 587}}}, nil` via `getCorporationStarbase` | `getCorporationStarbaseHandler("v1", "client", store, cache)` | `out.Detail.Fuels[0].TypeName == "Rifter"` |
| `TestListCorporationStructuresHandler_ResolvesTypeName` | `[]Structure{{StructureID: 1, TypeID: 587}}, 1, nil` via `listCorporationStructures` | `listCorporationStructuresHandler("v1", "client", store, cache)` | `out.Structures[0].TypeName == "Rifter"` |

(`ptrInt64` is whatever helper this file already uses for `*int64` literals in tests — check existing tests like `TestListCorporationMemberTrackingHandler_MapsFields` for the pattern; if none exists, define a small local `func ptrInt64(v int64) *int64 { return &v }` in the test file.)

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "ResolvesTypeName|ResolvesTypeNames|ResolvesShipTypeName|ResolvesFuelTypeNames" -v`
Expected: FAIL — none of the seven handlers accept a 4th argument yet.

- [ ] **Step 4: Implement enrichment in all seven handlers**

Same shape as Task 2/3 for each: add `sdeStore *sde.Store` as the trailing parameter, and after the existing data-fetch call succeeds, insert an `ids`-build + `resolveTypeNames` + assign-back block using each handler's own field name(s):

- `listCorporationBlueprintsHandler`: `ids[i] = bps[i].TypeID` → `bps[i].TypeName = names[bps[i].TypeID]`.
- `listCorporationContainerLogsHandler`: two ID sources per entry (`ContainerTypeID` always set, `TypeID` a `*int64` sometimes nil) — build `ids` from both (skip nil `TypeID`s), then assign `logs[i].ContainerTypeName = names[logs[i].ContainerTypeID]` and, only `if logs[i].TypeID != nil`, `logs[i].TypeName = names[*logs[i].TypeID]`.
- `listCorporationFacilitiesHandler`: `ids[i] = facilities[i].TypeID` → `facilities[i].TypeName = names[facilities[i].TypeID]`.
- `listCorporationMemberTrackingHandler`: `ShipTypeID` is `*int64` — only append to `ids` and assign `ShipTypeName` `if members[i].ShipTypeID != nil`.
- `listCorporationStarbasesHandler`: `ids[i] = starbases[i].TypeID` → `starbases[i].TypeName = names[starbases[i].TypeID]`.
- `getCorporationStarbaseHandler`: single object, not a list — after `detail, err := getCorporationStarbase(...)` succeeds, build `ids` from `detail.Fuels[i].TypeID` for each fuel, call `resolveTypeNames`, then loop `detail.Fuels[i].TypeName = names[detail.Fuels[i].TypeID]`.
- `listCorporationStructuresHandler`: `ids[i] = structures[i].TypeID` → `structures[i].TypeName = names[structures[i].TypeID]`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/corporation.go`'s imports (if not already present via another symbol).

In `mcp/server.go`, update all seven call sites (`list_corporation_blueprints`, `list_corporation_container_logs`, `list_corporation_facilities`, `list_corporation_member_tracking`, `list_corporation_starbases`, `get_corporation_starbase`, `list_corporation_structures`) to pass `sdeStore` as the trailing argument.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "ResolvesTypeName|ResolvesTypeNames|ResolvesShipTypeName|ResolvesFuelTypeNames" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean (update every pre-existing test for these seven handlers to pass a `&fakeTypeNameCache{}` 4th argument).

- [ ] **Step 7: Commit**

```bash
git add mcp/corporation.go mcp/corporation_internal_test.go mcp/server.go
git commit -m "feat: resolve corporation-tag type_id fields via sde/"
```

---

### Task 5: `mcp/contracts.go` — `ContractItem.TypeID` (three scopes)

**Files:**
- Modify: `mcp/contracts.go` (`ContractItem` struct lines 54-68; `listCharacterContractItemsHandler`, `listCorporationContractItemsHandler`, `listPublicContractItemsHandler` — the handlers wrapping `listCharacterContractItems`/`listCorporationContractItems`/`listPublicContractItems` at lines 176-190/336-350/472-487 respectively)
- Modify: `mcp/server.go` (three call sites: `list_character_contract_items`, `list_corporation_contract_items`, `list_public_contract_items`)
- Test: `mcp/contracts_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the field**

```go
type ContractItem struct {
	RecordID           int64  `json:"recordId"`
	TypeID             int64  `json:"typeId"`
	TypeName           string `json:"typeName,omitempty"`
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
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/contracts_internal_test.go
func TestListCharacterContractItemsHandler_ResolvesTypeName(t *testing.T) {
	orig := listCharacterContractItems
	t.Cleanup(func() { listCharacterContractItems = orig })
	listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
		return []ContractItem{{RecordID: 1, TypeID: 587}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter"}}}}

	handler := listCharacterContractItemsHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 1001, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Items[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Items[0].TypeName, "Rifter")
	}
}
```

Write the analogous `TestListCorporationContractItemsHandler_ResolvesTypeName` and `TestListPublicContractItemsHandler_ResolvesTypeName`, each stubbing the file's own `listCorporationContractItems`/`listPublicContractItems` var and calling `listCorporationContractItemsHandler("v1", "client", store, cache)` / `listPublicContractItemsHandler("v1", cache)` (public — no `clientID`/`store`, matching its existing signature) with the same shape.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "ContractItemsHandler_ResolvesTypeName" -v`
Expected: FAIL — signatures don't accept the cache argument yet.

- [ ] **Step 4: Implement enrichment in all three handlers**

Same shape each time: add `sdeStore *sde.Store` as the trailing parameter (for `listPublicContractItemsHandler`, which today only takes `version string`, it becomes the 2nd parameter), and after the existing items-fetch call succeeds, insert:
```go
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range items {
			items[i].TypeName = names[items[i].TypeID]
		}
```
(using each handler's own local variable name for the fetched slice — check the exact name in each function before inserting).

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/contracts.go`'s imports.

In `mcp/server.go`, update all three call sites to pass `sdeStore` (`listCharacterContractItemsHandler(version, clientID, store, sdeStore)`, `listCorporationContractItemsHandler(version, clientID, store, sdeStore)`, `listPublicContractItemsHandler(version, sdeStore)`).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "ContractItemsHandler_ResolvesTypeName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/contracts.go mcp/contracts_internal_test.go mcp/server.go
git commit -m "feat: resolve ContractItem.TypeName via sde/ across all three contract scopes"
```

---

### Task 6: `mcp/dogma.go` — `GetDogmaDynamicItemOutput` (single object, 3 fields)

**Files:**
- Modify: `mcp/dogma.go` (struct lines 94-102; `getDogmaDynamicItem` var lines 107-125; `getDogmaDynamicItemHandler`)
- Modify: `mcp/server.go` (`get_dogma_dynamic_item` call site)
- Test: `mcp/dogma_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type GetDogmaDynamicItemOutput struct {
	TypeID          int64                  `json:"typeId"`
	TypeName        string                 `json:"typeName,omitempty"`
	ItemID          int64                  `json:"itemId"`
	CreatedBy       int64                  `json:"createdBy"`
	MutatorTypeID   int64                  `json:"mutatorTypeId"`
	MutatorTypeName string                 `json:"mutatorTypeName,omitempty"`
	SourceTypeID    int64                  `json:"sourceTypeId"`
	SourceTypeName  string                 `json:"sourceTypeName,omitempty"`
	DogmaAttributes []DynamicItemAttribute `json:"dogmaAttributes"`
	DogmaEffects    []DynamicItemEffect    `json:"dogmaEffects"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/dogma_internal_test.go
func TestGetDogmaDynamicItemHandler_ResolvesTypeNames(t *testing.T) {
	orig := getDogmaDynamicItem
	t.Cleanup(func() { getDogmaDynamicItem = orig })
	getDogmaDynamicItem = func(ctx context.Context, userAgent string, typeID, itemID int64) (GetDogmaDynamicItemOutput, error) {
		return GetDogmaDynamicItemOutput{TypeID: 587, MutatorTypeID: 588, SourceTypeID: 587}, nil
	}
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter"}},
		588: {ID: 588, Name: map[string]string{"en": "Abyssal Mutaplasmid"}},
	}}

	handler := getDogmaDynamicItemHandler("v1", cache)
	_, out, err := handler(context.Background(), nil, GetDogmaDynamicItemInput{TypeID: 587, ItemID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.TypeName, "Rifter")
	}
	if out.MutatorTypeName != "Abyssal Mutaplasmid" {
		t.Errorf("MutatorTypeName = %q, want %q", out.MutatorTypeName, "Abyssal Mutaplasmid")
	}
	if out.SourceTypeName != "Rifter" {
		t.Errorf("SourceTypeName = %q, want %q", out.SourceTypeName, "Rifter")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetDogmaDynamicItemHandler_ResolvesTypeNames -v`
Expected: FAIL — `getDogmaDynamicItemHandler` doesn't accept a cache argument yet.

- [ ] **Step 4: Implement enrichment**

Change `getDogmaDynamicItemHandler`'s signature to `getDogmaDynamicItemHandler(version string, sdeStore *sde.Store)`, and after `out, err := getDogmaDynamicItem(...)` succeeds, insert (three flat scalar lookups, not a loop):
```go
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{out.TypeID, out.MutatorTypeID, out.SourceTypeID})
		out.TypeName = names[out.TypeID]
		out.MutatorTypeName = names[out.MutatorTypeID]
		out.SourceTypeName = names[out.SourceTypeID]
```
before the existing `return nil, out, nil`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/dogma.go`'s imports.

In `mcp/server.go`, update the `get_dogma_dynamic_item` call site: `getDogmaDynamicItemHandler(version, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetDogmaDynamicItemHandler_ResolvesTypeNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/dogma.go mcp/dogma_internal_test.go mcp/server.go
git commit -m "feat: resolve GetDogmaDynamicItemOutput type names via sde/"
```

---

### Task 7: `mcp/fittings.go` — `Fitting.ShipTypeID` + nested `FittingItem.TypeID`

**Files:**
- Modify: `mcp/fittings.go` (structs lines 18-30; `listCharacterFittingsHandler` wrapping `listCharacterFittings` lines 32-50) — **not** `FittingItemInput`/`CreateCharacterFittingInput` (write-side echoes, out of scope)
- Modify: `mcp/server.go` (`list_character_fittings` call site)
- Test: `mcp/fittings_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type FittingItem struct {
	Flag     string `json:"flag"`
	Quantity int64  `json:"quantity"`
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
}

type Fitting struct {
	FittingID    int64         `json:"fittingId"`
	ShipTypeID   int64         `json:"shipTypeId"`
	ShipTypeName string        `json:"shipTypeName,omitempty"`
	Description  string        `json:"description"`
	Name         string        `json:"name"`
	Items        []FittingItem `json:"items"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/fittings_internal_test.go
func TestListCharacterFittingsHandler_ResolvesTypeNames(t *testing.T) {
	orig := listCharacterFittings
	t.Cleanup(func() { listCharacterFittings = orig })
	listCharacterFittings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Fitting, error) {
		return []Fitting{{
			FittingID: 1, ShipTypeID: 587,
			Items: []FittingItem{{TypeID: 588, Quantity: 1}},
		}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter"}},
		588: {ID: 588, Name: map[string]string{"en": "150mm Railgun II"}},
	}}

	handler := listCharacterFittingsHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListCharacterFittingsInput{CharacterID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Fittings[0].ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.Fittings[0].ShipTypeName, "Rifter")
	}
	if out.Fittings[0].Items[0].TypeName != "150mm Railgun II" {
		t.Errorf("Items[0].TypeName = %q, want %q", out.Fittings[0].Items[0].TypeName, "150mm Railgun II")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListCharacterFittingsHandler_ResolvesTypeNames -v`
Expected: FAIL — signature doesn't accept the cache argument yet.

- [ ] **Step 4: Implement enrichment (two-level: outer ship type + nested item types)**

Change `listCharacterFittingsHandler`'s signature to add `sdeStore *sde.Store`, and after `fittings, err := listCharacterFittings(...)` succeeds, insert:
```go
		ids := make([]int64, 0, len(fittings))
		for _, f := range fittings {
			ids = append(ids, f.ShipTypeID)
			for _, it := range f.Items {
				ids = append(ids, it.TypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range fittings {
			fittings[i].ShipTypeName = names[fittings[i].ShipTypeID]
			for j := range fittings[i].Items {
				fittings[i].Items[j].TypeName = names[fittings[i].Items[j].TypeID]
			}
		}
```
before the existing return.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/fittings.go`'s imports.

In `mcp/server.go`, update the `list_character_fittings` call site: `listCharacterFittingsHandler(version, clientID, store, sdeStore)`. Do **not** touch `create_character_fitting`'s or `delete_character_fitting`'s call sites — those handlers are unaffected (write-only input, no response data to enrich).

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListCharacterFittingsHandler_ResolvesTypeNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/fittings.go mcp/fittings_internal_test.go mcp/server.go
git commit -m "feat: resolve Fitting.ShipTypeName and FittingItem.TypeName via sde/"
```

---

### Task 8: `mcp/fleets.go` — `FleetMember.ShipTypeID`

**Files:**
- Modify: `mcp/fleets.go` (struct lines 182-194; `listFleetMembersHandler` wrapping `listFleetMembers` lines 185-203)
- Modify: `mcp/server.go` (`list_fleet_members` call site)
- Test: `mcp/fleets_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the field**

```go
type FleetMember struct {
	CharacterID    int64     `json:"characterId"`
	JoinTime       time.Time `json:"joinTime"`
	Role           string    `json:"role"`
	RoleName       string    `json:"roleName"`
	ShipTypeID     int64     `json:"shipTypeId"`
	ShipTypeName   string    `json:"shipTypeName,omitempty"`
	SolarSystemID  int64     `json:"solarSystemId"`
	SquadID        int64     `json:"squadId"`
	StationID      *int64    `json:"stationId,omitempty"`
	TakesFleetWarp bool      `json:"takesFleetWarp"`
	WingID         int64     `json:"wingId"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/fleets_internal_test.go
func TestListFleetMembersHandler_ResolvesShipTypeName(t *testing.T) {
	orig := listFleetMembers
	t.Cleanup(func() { listFleetMembers = orig })
	listFleetMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetMember, error) {
		return []FleetMember{{CharacterID: 2001, ShipTypeID: 587}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter"}}}}

	handler := listFleetMembersHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListFleetMembersInput{CharacterID: 1001, FleetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Members[0].ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.Members[0].ShipTypeName, "Rifter")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListFleetMembersHandler_ResolvesShipTypeName -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `listFleetMembersHandler`'s signature to add `sdeStore *sde.Store`, and after `members, err := listFleetMembers(...)` succeeds, insert:
```go
		ids := make([]int64, len(members))
		for i, m := range members {
			ids[i] = m.ShipTypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range members {
			members[i].ShipTypeName = names[members[i].ShipTypeID]
		}
```
before the existing return.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/fleets.go`'s imports. In `mcp/server.go`, update the `list_fleet_members` call site: `listFleetMembersHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListFleetMembersHandler_ResolvesShipTypeName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/fleets.go mcp/fleets_internal_test.go mcp/server.go
git commit -m "feat: resolve FleetMember.ShipTypeName via sde/"
```

---

### Task 9: `mcp/industry.go` — four structs

**Files:**
- Modify: `mcp/industry.go` (`IndustryJob` lines 21-45 via `toIndustryJobFromCharacter`/`toIndustryJobFromCorporation` + `listCharacterIndustryJobsHandler`/`listCorporationIndustryJobsHandler`; `MiningLedgerEntry` lines 114-119 via `listCharacterMiningHandler`; `MiningObserverEntry` lines 280-286 via `getCorporationMiningObserverDetailHandler`; `IndustryFacility` lines 397-404 via `listIndustryFacilitiesHandler`)
- Modify: `mcp/server.go` (five call sites: `list_character_industry_jobs`, `list_corporation_industry_jobs`, `list_character_mining`, `get_corporation_mining_observer_detail`, `list_industry_facilities`)
- Test: `mcp/industry_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type IndustryJob struct {
	ActivityID           int64      `json:"activityId"`
	BlueprintID          int64      `json:"blueprintId"`
	BlueprintLocationID  int64      `json:"blueprintLocationId"`
	BlueprintTypeID      int64      `json:"blueprintTypeId"`
	BlueprintTypeName    string     `json:"blueprintTypeName,omitempty"`
	Duration             int64      `json:"duration"`
	EndDate              time.Time  `json:"endDate"`
	FacilityID           int64      `json:"facilityId"`
	InstallerID          int64      `json:"installerId"`
	JobID                int64      `json:"jobId"`
	OutputLocationID     int64      `json:"outputLocationId"`
	Runs                 int64      `json:"runs"`
	StartDate            time.Time  `json:"startDate"`
	Status               string     `json:"status"`
	StationID            int64      `json:"stationId,omitempty"`
	LocationID           int64      `json:"locationId,omitempty"`
	CompletedCharacterID *int64     `json:"completedCharacterId,omitempty"`
	CompletedDate        *time.Time `json:"completedDate,omitempty"`
	Cost                 *float64   `json:"cost,omitempty"`
	LicensedRuns         *int64     `json:"licensedRuns,omitempty"`
	PauseDate            *time.Time `json:"pauseDate,omitempty"`
	Probability          *float64   `json:"probability,omitempty"`
	ProductTypeID        *int64     `json:"productTypeId,omitempty"`
	ProductTypeName      string     `json:"productTypeName,omitempty"`
	SuccessfulRuns       *int64     `json:"successfulRuns,omitempty"`
}
```
```go
type MiningLedgerEntry struct {
	Date          string `json:"date"`
	Quantity      int64  `json:"quantity"`
	SolarSystemID int64  `json:"solarSystemId"`
	TypeID        int64  `json:"typeId"`
	TypeName      string `json:"typeName,omitempty"`
}
```
```go
type MiningObserverEntry struct {
	CharacterID           int64  `json:"characterId"`
	LastUpdated           string `json:"lastUpdated"`
	Quantity              int64  `json:"quantity"`
	RecordedCorporationID int64  `json:"recordedCorporationId"`
	TypeID                int64  `json:"typeId"`
	TypeName              string `json:"typeName,omitempty"`
}
```
```go
type IndustryFacility struct {
	FacilityID    int64    `json:"facilityId"`
	OwnerID       int64    `json:"ownerId"`
	RegionID      int64    `json:"regionId"`
	SolarSystemID int64    `json:"solarSystemId"`
	TypeID        int64    `json:"typeId"`
	TypeName      string   `json:"typeName,omitempty"`
	Tax           *float64 `json:"tax,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/industry_internal_test.go
func TestListCharacterIndustryJobsHandler_ResolvesTypeNames(t *testing.T) {
	orig := listCharacterIndustryJobs
	t.Cleanup(func() { listCharacterIndustryJobs = orig })
	listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]IndustryJob, error) {
		productTypeID := int64(588)
		return []IndustryJob{{JobID: 1, BlueprintTypeID: 587, ProductTypeID: &productTypeID}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter Blueprint"}},
		588: {ID: 588, Name: map[string]string{"en": "Rifter"}},
	}}

	handler := listCharacterIndustryJobsHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Jobs[0].BlueprintTypeName != "Rifter Blueprint" {
		t.Errorf("BlueprintTypeName = %q, want %q", out.Jobs[0].BlueprintTypeName, "Rifter Blueprint")
	}
	if out.Jobs[0].ProductTypeName != "Rifter" {
		t.Errorf("ProductTypeName = %q, want %q", out.Jobs[0].ProductTypeName, "Rifter")
	}
}
```

Write the analogous `TestListCorporationIndustryJobsHandler_ResolvesTypeNames` (stub `listCorporationIndustryJobs`), `TestListCharacterMiningHandler_ResolvesTypeName` (stub `listCharacterMining`, assert `out.Ledger[0].TypeName`), `TestGetCorporationMiningObserverDetailHandler_ResolvesTypeName` (stub the observer-detail fetch var, assert on the returned entries), and `TestListIndustryFacilitiesHandler_ResolvesTypeName` (stub `listIndustryFacilities`, public — no character/store, matching its existing signature).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "Industry.*ResolvesTypeName|Mining.*ResolvesTypeName" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment in all five handlers**

- `listCharacterIndustryJobsHandler`/`listCorporationIndustryJobsHandler`: add `sdeStore *sde.Store`; after the jobs slice is fetched, build `ids` from both `j.BlueprintTypeID` (always) and `*j.ProductTypeID` (only when non-nil) for each job, call `resolveTypeNames`, then assign `jobs[i].BlueprintTypeName = names[jobs[i].BlueprintTypeID]` and, `if jobs[i].ProductTypeID != nil`, `jobs[i].ProductTypeName = names[*jobs[i].ProductTypeID]`.
- `listCharacterMiningHandler`: add `sdeStore *sde.Store`; standard flat-slice shape on `TypeID`/`TypeName`.
- `getCorporationMiningObserverDetailHandler`: add `sdeStore *sde.Store`; standard flat-slice shape on `TypeID`/`TypeName`.
- `listIndustryFacilitiesHandler`: add `sdeStore *sde.Store` (as its only new/2nd parameter, since it's public); standard flat-slice shape on `TypeID`/`TypeName`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/industry.go`'s imports.

In `mcp/server.go`, update all five call sites to pass `sdeStore`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "Industry.*ResolvesTypeName|Mining.*ResolvesTypeName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/industry.go mcp/industry_internal_test.go mcp/server.go
git commit -m "feat: resolve industry-tag type_id fields via sde/"
```

---

### Task 10: `mcp/insurance.go` — `InsurancePrice.TypeID`

**Files:**
- Modify: `mcp/insurance.go` (struct lines 21-24; `listInsurancePricesHandler` wrapping `listInsurancePrices` lines 26-41)
- Modify: `mcp/server.go` (`list_insurance_prices` call site)
- Test: `mcp/insurance_internal_test.go` (existing file — extend it, per the read-back above)

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the field**

```go
type InsurancePrice struct {
	TypeID   int64            `json:"typeId"`
	TypeName string           `json:"typeName,omitempty"`
	Levels   []InsuranceLevel `json:"levels"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/insurance_internal_test.go
func TestListInsurancePricesHandler_ResolvesTypeName(t *testing.T) {
	orig := listInsurancePrices
	t.Cleanup(func() { listInsurancePrices = orig })
	listInsurancePrices = func(ctx context.Context, userAgent string) ([]InsurancePrice, error) {
		return []InsurancePrice{{TypeID: 587, Levels: []InsuranceLevel{{Cost: 900.0, Payout: 450.0, Name: "Standard"}}}}, nil
	}
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter"}}}}

	handler := listInsurancePricesHandler("v1", cache)
	_, out, err := handler(context.Background(), nil, ListInsurancePricesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Prices[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Prices[0].TypeName, "Rifter")
	}
}
```

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to this test file's imports (for `sde.Type`) alongside the existing `context`/`errors`/`testing`.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListInsurancePricesHandler_ResolvesTypeName -v`
Expected: FAIL — `listInsurancePricesHandler("v1")` currently takes one argument only.

- [ ] **Step 4: Implement enrichment**

Change `listInsurancePricesHandler`'s signature to `listInsurancePricesHandler(version string, sdeStore *sde.Store)`, and after `prices, err := listInsurancePrices(...)` succeeds, insert:
```go
		ids := make([]int64, len(prices))
		for i, p := range prices {
			ids[i] = p.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range prices {
			prices[i].TypeName = names[prices[i].TypeID]
		}
```
before the existing return.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` and `"github.com/rusher2004/new-eden-agent-toolkit/sso"` to `mcp/insurance.go`'s imports.

In `mcp/server.go`, update the `list_insurance_prices` call site: `listInsurancePricesHandler(version, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListInsurancePricesHandler_ResolvesTypeName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean (update `TestListInsurancePricesHandler_MapsFields`/`_PropagatesError` to pass a `&fakeTypeNameCache{}` 2nd argument).

- [ ] **Step 7: Commit**

```bash
git add mcp/insurance.go mcp/insurance_internal_test.go mcp/server.go
git commit -m "feat: resolve InsurancePrice.TypeName via sde/"
```

---

### Task 11: `mcp/killmails.go` — deepest nesting in the batch

**Files:**
- Modify: `mcp/killmails.go` (`KillmailAttacker` lines 120-130, `KillmailInnerItem` lines 142-148, `KillmailItem` lines 152-159, `KillmailVictim` lines 167-176; `getKillmailHandler` lines 220-249 and its helpers `toKillmailInnerItems`/`toKillmailItems`)
- Modify: `mcp/server.go` (`get_killmail` call site)
- Test: `mcp/killmails_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type KillmailAttacker struct {
	AllianceID     *int64  `json:"allianceId,omitempty"`
	CharacterID    *int64  `json:"characterId,omitempty"`
	CorporationID  *int64  `json:"corporationId,omitempty"`
	DamageDone     int64   `json:"damageDone"`
	FactionID      *int64  `json:"factionId,omitempty"`
	FinalBlow      bool    `json:"finalBlow"`
	SecurityStatus float64 `json:"securityStatus"`
	ShipTypeID     *int64  `json:"shipTypeId,omitempty"`
	ShipTypeName   string  `json:"shipTypeName,omitempty"`
	WeaponTypeID   *int64  `json:"weaponTypeId,omitempty"`
	WeaponTypeName string  `json:"weaponTypeName,omitempty"`
}
```
```go
type KillmailInnerItem struct {
	Flag              int64  `json:"flag"`
	ItemTypeID        int64  `json:"itemTypeId"`
	ItemTypeName      string `json:"itemTypeName,omitempty"`
	Singleton         int64  `json:"singleton"`
	QuantityDestroyed *int64 `json:"quantityDestroyed,omitempty"`
	QuantityDropped   *int64 `json:"quantityDropped,omitempty"`
}

type KillmailItem struct {
	Flag              int64               `json:"flag"`
	ItemTypeID        int64               `json:"itemTypeId"`
	ItemTypeName      string              `json:"itemTypeName,omitempty"`
	Singleton         int64               `json:"singleton"`
	QuantityDestroyed *int64              `json:"quantityDestroyed,omitempty"`
	QuantityDropped   *int64              `json:"quantityDropped,omitempty"`
	Items             []KillmailInnerItem `json:"items,omitempty"`
}
```
```go
type KillmailVictim struct {
	AllianceID    *int64            `json:"allianceId,omitempty"`
	CharacterID   *int64            `json:"characterId,omitempty"`
	CorporationID *int64            `json:"corporationId,omitempty"`
	DamageTaken   int64             `json:"damageTaken"`
	FactionID     *int64            `json:"factionId,omitempty"`
	ShipTypeID    int64             `json:"shipTypeId"`
	ShipTypeName  string            `json:"shipTypeName,omitempty"`
	Items         []KillmailItem    `json:"items,omitempty"`
	Position      *KillmailPosition `json:"position,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/killmails_internal_test.go
func TestGetKillmailHandler_ResolvesTypeNames(t *testing.T) {
	orig := getKillmail
	t.Cleanup(func() { getKillmail = orig })
	getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
		shipTypeID, weaponTypeID := int64(587), int64(2456)
		return &esiapi.KillmailsKillmailIdKillmailHashGet{
			Attackers: []esiapi.KillmailsKillmailIdKillmailHashGetAttackersInner{{ShipTypeId: &shipTypeID, WeaponTypeId: &weaponTypeID, FinalBlow: true}},
			Victim: esiapi.KillmailsKillmailIdKillmailHashGetVictim{
				ShipTypeId: 587,
				Items: []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInner{
					{ItemTypeId: 34, Items: []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInnerItemsInner{{ItemTypeId: 34}}},
				},
			},
		}, nil
	}
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587:  {ID: 587, Name: map[string]string{"en": "Rifter"}},
		2456: {ID: 2456, Name: map[string]string{"en": "125mm Gatling AutoCannon II"}},
	}}

	handler := getKillmailHandler("v1", cache)
	_, out, err := handler(context.Background(), nil, GetKillmailInput{KillmailID: 1, Hash: "abc"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Victim.ShipTypeName != "Rifter" {
		t.Errorf("Victim.ShipTypeName = %q, want %q", out.Victim.ShipTypeName, "Rifter")
	}
	if out.Attackers[0].ShipTypeName != "Rifter" {
		t.Errorf("Attackers[0].ShipTypeName = %q, want %q", out.Attackers[0].ShipTypeName, "Rifter")
	}
	if out.Attackers[0].WeaponTypeName != "125mm Gatling AutoCannon II" {
		t.Errorf("Attackers[0].WeaponTypeName = %q, want %q", out.Attackers[0].WeaponTypeName, "125mm Gatling AutoCannon II")
	}
}
```

The overridable var is `getKillmail` (declared at `mcp/killmails.go:201` as `var getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error)`), confirmed directly against the generated model (`KillmailsKillmailIdKillmailHashGetAttackersInner.ShipTypeId`/`.WeaponTypeId *int64`, `.FinalBlow bool`; `KillmailsKillmailIdKillmailHashGetVictim.ShipTypeId int64`/`.Items []KillmailsKillmailIdKillmailHashGetVictimItemsInner`).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetKillmailHandler_ResolvesTypeNames -v`
Expected: FAIL — `getKillmailHandler` doesn't accept a cache argument yet.

- [ ] **Step 4: Implement enrichment (four sites: attackers, victim ship, victim items, victim inner items)**

Change `getKillmailHandler`'s signature to add `sdeStore *sde.Store`. After `attackers` and the `Victim` value are both built (i.e. right before the final `return nil, GetKillmailOutput{...}, nil`), collect every ID across all four levels in one pass, resolve them all with a single `resolveTypeNames` call, then assign back:

```go
		ids := make([]int64, 0)
		for _, a := range attackers {
			if a.ShipTypeID != nil {
				ids = append(ids, *a.ShipTypeID)
			}
			if a.WeaponTypeID != nil {
				ids = append(ids, *a.WeaponTypeID)
			}
		}
		ids = append(ids, victim.ShipTypeID)
		for _, it := range victim.Items {
			ids = append(ids, it.ItemTypeID)
			for _, inner := range it.Items {
				ids = append(ids, inner.ItemTypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range attackers {
			if attackers[i].ShipTypeID != nil {
				attackers[i].ShipTypeName = names[*attackers[i].ShipTypeID]
			}
			if attackers[i].WeaponTypeID != nil {
				attackers[i].WeaponTypeName = names[*attackers[i].WeaponTypeID]
			}
		}
		victim.ShipTypeName = names[victim.ShipTypeID]
		for i := range victim.Items {
			victim.Items[i].ItemTypeName = names[victim.Items[i].ItemTypeID]
			for j := range victim.Items[i].Items {
				victim.Items[i].Items[j].ItemTypeName = names[victim.Items[i].Items[j].ItemTypeID]
			}
		}
```

Since the current code constructs `Victim: KillmailVictim{...}` inline inside the final `return` statement (see the audit's lines 505-514), this requires first assigning that literal to a local `victim := KillmailVictim{...}` variable (instead of an inline composite literal in the return), so the enrichment block above can mutate it before the `return nil, GetKillmailOutput{..., Victim: victim}, nil` at the end. Restructure accordingly — this is a mechanical extraction, not a behavior change to the existing field-mapping logic.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/killmails.go`'s imports.

In `mcp/server.go`, update the `get_killmail` call site: `getKillmailHandler(version, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetKillmailHandler_ResolvesTypeNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean (update every existing `TestGetKillmailHandler_*` call site to pass a `&fakeTypeNameCache{}` 2nd argument).

- [ ] **Step 7: Commit**

```bash
git add mcp/killmails.go mcp/killmails_internal_test.go mcp/server.go
git commit -m "feat: resolve killmail attacker/victim/item type names via sde/"
```

---

### Task 12: `mcp/location.go` — `GetCharacterShipOutput.ShipTypeID`

**Files:**
- Modify: `mcp/location.go` (struct lines 111-116; `getCharacterShipHandler` lines 118-133)
- Modify: `mcp/server.go` (`get_character_ship` call site)
- Test: `mcp/location_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the field**

```go
type GetCharacterShipOutput struct {
	CharacterID  int32  `json:"characterId"`
	ShipItemID   int64  `json:"shipItemId"`
	ShipName     string `json:"shipName"`
	ShipTypeID   int64  `json:"shipTypeId"`
	ShipTypeName string `json:"shipTypeName,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/location_internal_test.go
func TestGetCharacterShipHandler_ResolvesShipTypeName(t *testing.T) {
	orig := getCharacterShip
	t.Cleanup(func() { getCharacterShip = orig })
	getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
		return 1, "My Rifter", 587, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter"}}}}

	handler := getCharacterShipHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.ShipTypeName, "Rifter")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetCharacterShipHandler_ResolvesShipTypeName -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment (single object, no loop)**

Change `getCharacterShipHandler`'s signature to add `sdeStore *sde.Store`, and after `shipItemID, shipName, shipTypeID, err := getCharacterShip(...)` succeeds, insert:
```go
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{shipTypeID})
```
then change the existing return line to:
```go
		return nil, GetCharacterShipOutput{CharacterID: in.CharacterID, ShipItemID: shipItemID, ShipName: shipName, ShipTypeID: shipTypeID, ShipTypeName: names[shipTypeID]}, nil
```

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/location.go`'s imports. In `mcp/server.go`, update the `get_character_ship` call site: `getCharacterShipHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetCharacterShipHandler_ResolvesShipTypeName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/location.go mcp/location_internal_test.go mcp/server.go
git commit -m "feat: resolve GetCharacterShipOutput.ShipTypeName via sde/"
```

---

### Task 13: `mcp/loyalty.go` — `LoyaltyStoreOffer.TypeID` + nested `LoyaltyRequiredItem.TypeID`

**Files:**
- Modify: `mcp/loyalty.go` (structs lines 64-77; `listLoyaltyStoreOffersHandler` wrapping `listLoyaltyStoreOffers` lines 79-97)
- Modify: `mcp/server.go` (`list_loyalty_store_offers` call site)
- Test: `mcp/loyalty_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type LoyaltyRequiredItem struct {
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
	Quantity int64  `json:"quantity"`
}

type LoyaltyStoreOffer struct {
	OfferID       int64                 `json:"offerId"`
	TypeID        int64                 `json:"typeId"`
	TypeName      string                `json:"typeName,omitempty"`
	Quantity      int64                 `json:"quantity"`
	IskCost       int64                 `json:"iskCost"`
	LpCost        int64                 `json:"lpCost"`
	AkCost        *int64                `json:"akCost,omitempty"`
	RequiredItems []LoyaltyRequiredItem `json:"requiredItems"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/loyalty_internal_test.go
func TestListLoyaltyStoreOffersHandler_ResolvesTypeNames(t *testing.T) {
	orig := listLoyaltyStoreOffers
	t.Cleanup(func() { listLoyaltyStoreOffers = orig })
	listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
		return []LoyaltyStoreOffer{{
			OfferID: 1, TypeID: 587,
			RequiredItems: []LoyaltyRequiredItem{{TypeID: 588, Quantity: 1}},
		}}, nil
	}
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter"}},
		588: {ID: 588, Name: map[string]string{"en": "Tritanium"}},
	}}

	handler := listLoyaltyStoreOffersHandler("v1", cache)
	_, out, err := handler(context.Background(), nil, ListLoyaltyStoreOffersInput{CorporationID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Offers[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Offers[0].TypeName, "Rifter")
	}
	if out.Offers[0].RequiredItems[0].TypeName != "Tritanium" {
		t.Errorf("RequiredItems[0].TypeName = %q, want %q", out.Offers[0].RequiredItems[0].TypeName, "Tritanium")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListLoyaltyStoreOffersHandler_ResolvesTypeNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment (two-level, mirrors Task 7's fittings shape)**

Change `listLoyaltyStoreOffersHandler`'s signature to add `sdeStore *sde.Store`, and after `offers, err := listLoyaltyStoreOffers(...)` succeeds, insert:
```go
		ids := make([]int64, 0, len(offers))
		for _, o := range offers {
			ids = append(ids, o.TypeID)
			for _, it := range o.RequiredItems {
				ids = append(ids, it.TypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range offers {
			offers[i].TypeName = names[offers[i].TypeID]
			for j := range offers[i].RequiredItems {
				offers[i].RequiredItems[j].TypeName = names[offers[i].RequiredItems[j].TypeID]
			}
		}
```
before the existing return.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/loyalty.go`'s imports. In `mcp/server.go`, update the `list_loyalty_store_offers` call site: `listLoyaltyStoreOffersHandler(version, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListLoyaltyStoreOffersHandler_ResolvesTypeNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/loyalty.go mcp/loyalty_internal_test.go mcp/server.go
git commit -m "feat: resolve LoyaltyStoreOffer/LoyaltyRequiredItem.TypeName via sde/"
```

---

### Task 14: `mcp/market.go` — `MarketOrder`, `MarketOrderBookEntry`, `MarketPrice`, `ListMarketTypesOutput`

**Files:**
- Modify: `mcp/market.go` (`MarketOrder` lines 25-43 via four mapper functions + four handlers; `MarketOrderBookEntry` lines 92-105 via two mappers + two handlers; `MarketPrice` lines 364-368 via `listMarketPricesHandler`; `ListMarketTypesOutput` lines 484-489 via `listMarketTypesHandler`)
- Modify: `mcp/server.go` (nine call sites: `list_character_market_orders`, `list_character_market_order_history`, `list_corporation_market_orders`, `list_corporation_market_order_history`, `list_market_prices`, `list_market_orders`, `list_market_types`, `list_structure_market_orders` — 8 handlers; `get_market_history` is explicitly **not** touched, see Global Constraints)
- Test: `mcp/market_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
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
	TypeName       string    `json:"typeName,omitempty"`
	VolumeRemain   int64     `json:"volumeRemain"`
	VolumeTotal    int64     `json:"volumeTotal"`
	IsCorporation  *bool     `json:"isCorporation,omitempty"`
	State          string    `json:"state,omitempty"`
	IssuedBy       *int64    `json:"issuedBy,omitempty"`
	WalletDivision *int64    `json:"walletDivision,omitempty"`
}
```
```go
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
	TypeName     string    `json:"typeName,omitempty"`
	VolumeRemain int64     `json:"volumeRemain"`
	VolumeTotal  int64     `json:"volumeTotal"`
}
```
```go
type MarketPrice struct {
	TypeID        int64    `json:"typeId"`
	TypeName      string   `json:"typeName,omitempty"`
	AdjustedPrice *float64 `json:"adjustedPrice,omitempty"`
	AveragePrice  *float64 `json:"averagePrice,omitempty"`
}
```
```go
type ListMarketTypesOutput struct {
	RegionID   int64    `json:"regionId"`
	Page       int32    `json:"page"`
	TotalPages int32    `json:"totalPages"`
	TypeIDs    []int64  `json:"typeIds"`
	TypeNames  []string `json:"typeNames,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/market_internal_test.go
func TestListCharacterMarketOrdersHandler_ResolvesTypeName(t *testing.T) {
	orig := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = orig })
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MarketOrder, error) {
		return []MarketOrder{{OrderID: 1, TypeID: 587}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter"}}}}

	handler := listCharacterMarketOrdersHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 1001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Orders[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Orders[0].TypeName, "Rifter")
	}
}

func TestListMarketTypesHandler_ResolvesTypeNames(t *testing.T) {
	orig := listMarketTypes
	t.Cleanup(func() { listMarketTypes = orig })
	listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{587, 588}, 1, nil
	}
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter"}},
		588: {ID: 588, Name: map[string]string{"en": "Slasher"}},
	}}

	handler := listMarketTypesHandler("v1", cache)
	_, out, err := handler(context.Background(), nil, ListMarketTypesInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.TypeNames) != 2 || out.TypeNames[0] != "Rifter" || out.TypeNames[1] != "Slasher" {
		t.Errorf("TypeNames = %v, want [Rifter Slasher]", out.TypeNames)
	}
}
```

Write the same shape of test (stub the file's own data-fetch var, assert the mapped output's `TypeName`) for: `listCharacterMarketOrderHistoryHandler`, `listCorporationMarketOrdersHandler`, `listCorporationMarketOrderHistoryHandler`, `listMarketPricesHandler` (assert on `MarketPrice.TypeName`), `listMarketOrdersHandler` (assert on `MarketOrderBookEntry.TypeName`), `listStructureMarketOrdersHandler` (same). Do **not** write a test expecting `GetMarketHistoryOutput.TypeName` — that field is intentionally excluded (input echo).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "MarketOrdersHandler_ResolvesTypeName|MarketPricesHandler_ResolvesTypeName|MarketTypesHandler_ResolvesTypeNames" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment in all eight handlers**

For the six `MarketOrder`-returning handlers and two `MarketOrderBookEntry`-returning handlers: add `sdeStore *sde.Store` as the trailing parameter, and after the existing fetch-and-map loop produces `out []MarketOrder` (or `[]MarketOrderBookEntry`), insert the standard flat-slice shape:
```go
		ids := make([]int64, len(out))
		for i, o := range out {
			ids[i] = o.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
```
(using each handler's own local variable name for the slice — several are named `out`, `orders`, or similar; check before inserting).

For `listMarketPricesHandler`: same flat-slice shape on the `[]MarketPrice` result.

For `listMarketTypesHandler`: after the existing `types, totalPages, err := listMarketTypes(...)` call, insert a batch (not per-struct) resolution:
```go
		nameMap := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), types)
		typeNames := make([]string, len(types))
		for i, id := range types {
			typeNames[i] = nameMap[id]
		}
```
then change the existing return to `ListMarketTypesOutput{RegionID: in.RegionID, Page: page, TotalPages: totalPages, TypeIDs: types, TypeNames: typeNames}`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/market.go`'s imports.

In `mcp/server.go`, update the eight call sites listed in this task's Files section to pass `sdeStore`. Leave `get_market_history`'s call site (`getMarketHistoryHandler(version)`) unchanged.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "MarketOrdersHandler_ResolvesTypeName|MarketPricesHandler_ResolvesTypeName|MarketTypesHandler_ResolvesTypeNames" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/market.go mcp/market_internal_test.go mcp/server.go
git commit -m "feat: resolve market-tag type_id fields via sde/"
```

---

### Task 15: `mcp/planetary_interaction.go` — most structurally complex file in the batch

**Files:**
- Modify: `mcp/planetary_interaction.go` (`PlanetPinContent`/`PlanetExtractorDetails`/`PlanetPin`/`PlanetRoute` lines 92-136 via `toPlanetPins`/`toPlanetRoutes` + `getCharacterPlanetHandler`; `CustomsOffice` lines 233-249 via `listCorporationCustomsOfficesHandler`)
- Modify: `mcp/server.go` (`get_character_planet`, `list_corporation_customs_offices` call sites)
- Test: `mcp/planetary_interaction_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type PlanetPinContent struct {
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
	Amount   int64  `json:"amount"`
}
```
```go
type PlanetExtractorDetails struct {
	CycleTime       *int64                `json:"cycleTime,omitempty"`
	HeadRadius      *float64              `json:"headRadius,omitempty"`
	ProductTypeID   *int64                `json:"productTypeId,omitempty"`
	ProductTypeName string                `json:"productTypeName,omitempty"`
	QtyPerCycle     *int64                `json:"qtyPerCycle,omitempty"`
	Heads           []PlanetExtractorHead `json:"heads"`
}
```
```go
type PlanetPin struct {
	PinID            int64                   `json:"pinId"`
	TypeID           int64                   `json:"typeId"`
	TypeName         string                  `json:"typeName,omitempty"`
	Latitude         float64                 `json:"latitude"`
	Longitude        float64                 `json:"longitude"`
	SchematicID      *int64                  `json:"schematicId,omitempty"`
	ExpiryTime       *time.Time              `json:"expiryTime,omitempty"`
	InstallTime      *time.Time              `json:"installTime,omitempty"`
	LastCycleStart   *time.Time              `json:"lastCycleStart,omitempty"`
	Contents         []PlanetPinContent      `json:"contents,omitempty"`
	ExtractorDetails *PlanetExtractorDetails `json:"extractorDetails,omitempty"`
	FactoryDetails   *PlanetFactoryDetails   `json:"factoryDetails,omitempty"`
}
```
```go
type PlanetRoute struct {
	RouteID          int64   `json:"routeId"`
	SourcePinID      int64   `json:"sourcePinId"`
	DestinationPinID int64   `json:"destinationPinId"`
	ContentTypeID    int64   `json:"contentTypeId"`
	ContentTypeName  string  `json:"contentTypeName,omitempty"`
	Quantity         float64 `json:"quantity"`
	Waypoints        []int64 `json:"waypoints,omitempty"`
}
```
```go
type CustomsOffice struct {
	OfficeID                 int64    `json:"officeId"`
	SystemID                 int64    `json:"systemId"`
	TypeID                   *int64   `json:"typeId,omitempty"`
	TypeName                 string   `json:"typeName,omitempty"`
	ReinforceExitStart       int64    `json:"reinforceExitStart"`
	ReinforceExitEnd         int64    `json:"reinforceExitEnd"`
	AllowAllianceAccess      bool     `json:"allowAllianceAccess"`
	AllowAccessWithStandings bool     `json:"allowAccessWithStandings"`
	StandingLevel            *string  `json:"standingLevel,omitempty"`
	AllianceTaxRate          *float64 `json:"allianceTaxRate,omitempty"`
	CorporationTaxRate       *float64 `json:"corporationTaxRate,omitempty"`
	ExcellentStandingTaxRate *float64 `json:"excellentStandingTaxRate,omitempty"`
	GoodStandingTaxRate      *float64 `json:"goodStandingTaxRate,omitempty"`
	NeutralStandingTaxRate   *float64 `json:"neutralStandingTaxRate,omitempty"`
	BadStandingTaxRate       *float64 `json:"badStandingTaxRate,omitempty"`
	TerribleStandingTaxRate  *float64 `json:"terribleStandingTaxRate,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/planetary_interaction_internal_test.go
func TestGetCharacterPlanetHandler_ResolvesTypeNames(t *testing.T) {
	orig := getCharacterPlanet
	t.Cleanup(func() { getCharacterPlanet = orig })
	productTypeID := int64(587)
	getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
		return &esiapi.CharactersCharacterIdPlanetsPlanetIdGet{
			Pins: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInner{{
				TypeId:   587,
				Contents: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerContentsInner{{TypeId: 588, Amount: 100}},
				ExtractorDetails: &esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerExtractorDetails{ProductTypeId: &productTypeID},
			}},
			Routes: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetRoutesInner{{ContentTypeId: 588}},
		}, nil
	}
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Command Center"}},
		588: {ID: 588, Name: map[string]string{"en": "Water"}},
	}}

	handler := getCharacterPlanetHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, GetCharacterPlanetInput{CharacterID: 1001, PlanetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Pins[0].TypeName != "Command Center" {
		t.Errorf("Pins[0].TypeName = %q, want %q", out.Pins[0].TypeName, "Command Center")
	}
	if out.Pins[0].Contents[0].TypeName != "Water" {
		t.Errorf("Pins[0].Contents[0].TypeName = %q, want %q", out.Pins[0].Contents[0].TypeName, "Water")
	}
	if out.Pins[0].ExtractorDetails.ProductTypeName != "Command Center" {
		t.Errorf("Pins[0].ExtractorDetails.ProductTypeName = %q, want %q", out.Pins[0].ExtractorDetails.ProductTypeName, "Command Center")
	}
	if out.Routes[0].ContentTypeName != "Water" {
		t.Errorf("Routes[0].ContentTypeName = %q, want %q", out.Routes[0].ContentTypeName, "Water")
	}
}

func TestListCorporationCustomsOfficesHandler_ResolvesTypeName(t *testing.T) {
	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	typeID := int64(2233)
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]CustomsOffice, error) {
		return []CustomsOffice{{OfficeID: 1, TypeID: &typeID}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{2233: {ID: 2233, Name: map[string]string{"en": "Customs Office"}}}}

	handler := listCorporationCustomsOfficesHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 1001, CorporationID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Offices[0].TypeName != "Customs Office" {
		t.Errorf("TypeName = %q, want %q", out.Offices[0].TypeName, "Customs Office")
	}
}
```

The overridable var is `getCharacterPlanet` (declared at `mcp/planetary_interaction.go:191`), confirmed directly against the generated model (`CharactersCharacterIdPlanetsPlanetIdGet{Links, Pins, Routes}`; `PinsInner{Contents, ExtractorDetails *..., TypeId int64, ...}`; `PinsInnerContentsInner{Amount, TypeId}`; `PinsInnerExtractorDetails{ProductTypeId *int64, ...}`; `RoutesInner{ContentTypeId, ...}`).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "GetCharacterPlanetHandler_ResolvesTypeNames|ListCorporationCustomsOfficesHandler_ResolvesTypeName" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

For `getCharacterPlanetHandler`: add `sdeStore *sde.Store`. After `toPlanetPins`/`toPlanetRoutes` produce `pins []PlanetPin` and `routes []PlanetRoute` (before the final return), insert:
```go
		ids := make([]int64, 0, len(pins)+len(routes))
		for _, p := range pins {
			ids = append(ids, p.TypeID)
			for _, c := range p.Contents {
				ids = append(ids, c.TypeID)
			}
			if p.ExtractorDetails != nil && p.ExtractorDetails.ProductTypeID != nil {
				ids = append(ids, *p.ExtractorDetails.ProductTypeID)
			}
		}
		for _, r := range routes {
			ids = append(ids, r.ContentTypeID)
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range pins {
			pins[i].TypeName = names[pins[i].TypeID]
			for j := range pins[i].Contents {
				pins[i].Contents[j].TypeName = names[pins[i].Contents[j].TypeID]
			}
			if pins[i].ExtractorDetails != nil && pins[i].ExtractorDetails.ProductTypeID != nil {
				pins[i].ExtractorDetails.ProductTypeName = names[*pins[i].ExtractorDetails.ProductTypeID]
			}
		}
		for i := range routes {
			routes[i].ContentTypeName = names[routes[i].ContentTypeID]
		}
```
(inserted after the `toPlanetPins(detail.Pins)`/`toPlanetRoutes(detail.Routes)` calls currently inlined in the return statement — restructure to assign `pins := toPlanetPins(detail.Pins)` and `routes := toPlanetRoutes(detail.Routes)` to local variables first, mirroring Task 11's `victim` extraction, so the enrichment block above can mutate them before the final `return nil, GetCharacterPlanetOutput{..., Pins: pins, Routes: routes}, nil`).

For `listCorporationCustomsOfficesHandler`: add `sdeStore *sde.Store`, and after the offices slice is fetched, insert:
```go
		ids := make([]int64, 0, len(offices))
		for _, o := range offices {
			if o.TypeID != nil {
				ids = append(ids, *o.TypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range offices {
			if offices[i].TypeID != nil {
				offices[i].TypeName = names[*offices[i].TypeID]
			}
		}
```

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/planetary_interaction.go`'s imports.

In `mcp/server.go`, update both call sites: `getCharacterPlanetHandler(version, clientID, store, sdeStore)`, `listCorporationCustomsOfficesHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "GetCharacterPlanetHandler_ResolvesTypeNames|ListCorporationCustomsOfficesHandler_ResolvesTypeName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/planetary_interaction.go mcp/planetary_interaction_internal_test.go mcp/server.go
git commit -m "feat: resolve planetary-interaction type_id fields via sde/"
```

---

### Task 16: `mcp/search.go` — `SearchResults.InventoryType` (batch list, not per-struct)

**Files:**
- Modify: `mcp/search.go` (struct lines 18-30; `searchCharacterHandler` lines 51-71)
- Modify: `mcp/server.go` (`search_character` call site)
- Test: `mcp/search_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the field**

```go
type SearchResults struct {
	Agent              []int64  `json:"agent,omitempty"`
	Alliance           []int64  `json:"alliance,omitempty"`
	Character          []int64  `json:"character,omitempty"`
	Constellation      []int64  `json:"constellation,omitempty"`
	Corporation        []int64  `json:"corporation,omitempty"`
	Faction            []int64  `json:"faction,omitempty"`
	InventoryType      []int64  `json:"inventoryType,omitempty"`
	InventoryTypeNames []string `json:"inventoryTypeNames,omitempty"`
	Region             []int64  `json:"region,omitempty"`
	SolarSystem        []int64  `json:"solarSystem,omitempty"`
	Station            []int64  `json:"station,omitempty"`
	Structure          []int64  `json:"structure,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/search_internal_test.go
func TestSearchCharacterHandler_ResolvesInventoryTypeNames(t *testing.T) {
	orig := searchCharacter
	t.Cleanup(func() { searchCharacter = orig })
	searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
		return &esiapi.CharactersCharacterIdSearchGet{InventoryType: []int64{587, 588}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter"}},
		588: {ID: 588, Name: map[string]string{"en": "Slasher"}},
	}}

	handler := searchCharacterHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, SearchCharacterInput{CharacterID: 1001, Categories: []string{"inventory_type"}, Search: "rifter"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Results.InventoryTypeNames) != 2 || out.Results.InventoryTypeNames[0] != "Rifter" || out.Results.InventoryTypeNames[1] != "Slasher" {
		t.Errorf("InventoryTypeNames = %v, want [Rifter Slasher]", out.Results.InventoryTypeNames)
	}
}
```

`searchCharacter` is declared at `mcp/search.go:32` with exactly this signature, confirmed against the generated model (`CharactersCharacterIdSearchGet{Agent, Alliance, Character, Constellation, Corporation, Faction, InventoryType, Region, SolarSystem, Station, Structure []int64}`).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestSearchCharacterHandler_ResolvesInventoryTypeNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment (batch list, index-aligned — no per-struct loop)**

Change `searchCharacterHandler`'s signature to add `sdeStore *sde.Store`, and after `results, err := searchCharacter(...)` succeeds, insert:
```go
		nameMap := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), results.InventoryType)
		inventoryTypeNames := make([]string, len(results.InventoryType))
		for i, id := range results.InventoryType {
			inventoryTypeNames[i] = nameMap[id]
		}
```
then add `InventoryTypeNames: inventoryTypeNames` to the existing `SearchResults{...}` composite literal.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/search.go`'s imports. In `mcp/server.go`, update the `search_character` call site: `searchCharacterHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestSearchCharacterHandler_ResolvesInventoryTypeNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/search.go mcp/search_internal_test.go mcp/server.go
git commit -m "feat: resolve SearchResults.InventoryTypeNames via sde/"
```

---

### Task 17: `mcp/structures.go` — four single-object handlers

**Files:**
- Modify: `mcp/structures.go` (`MercenaryDenDetail` lines 81-89 via `getCharacterMercenaryDenHandler`; `SkyhookReagent` lines 195-200 via `getCorporationSkyhookHandler`; `StructureSovereigntyHubReagent`/`StructureSovereigntyHubUpgrade` lines 323-347 via `getCorporationSovereigntyHubHandler`)
- Modify: `mcp/server.go` (`get_character_mercenary_den`, `get_corporation_skyhook`, `get_corporation_sovereignty_hub` call sites)
- Test: `mcp/structures_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type MercenaryDenDetail struct {
	ID               int64                  `json:"id"`
	TypeID           int64                  `json:"typeId"`
	TypeName         string                 `json:"typeName,omitempty"`
	State            string                 `json:"state"`
	Evolution        MercenaryDenEvolution  `json:"evolution"`
	InfomorphsAmount int64                  `json:"infomorphsAmount"`
	ReinforcementEnd *time.Time             `json:"reinforcementEnd,omitempty"`
	Skyhook          MercenaryDenSkyhookRef `json:"skyhook"`
}
```
```go
type SkyhookReagent struct {
	LastCycle      time.Time `json:"lastCycle"`
	SecuredStock   int64     `json:"securedStock"`
	TypeID         int64     `json:"typeId"`
	TypeName       string    `json:"typeName,omitempty"`
	UnsecuredStock int64     `json:"unsecuredStock"`
}
```
```go
type StructureSovereigntyHubReagent struct {
	Amount         int64  `json:"amount"`
	BurningPerHour int64  `json:"burningPerHour"`
	TypeID         int64  `json:"typeId"`
	TypeName       string `json:"typeName,omitempty"`
}

type StructureSovereigntyHubUpgrade struct {
	PowerState string `json:"powerState"`
	TypeID     int64  `json:"typeId"`
	TypeName   string `json:"typeName,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/structures_internal_test.go
func TestGetCharacterMercenaryDenHandler_ResolvesTypeName(t *testing.T) {
	orig := getCharacterMercenaryDen
	t.Cleanup(func() { getCharacterMercenaryDen = orig })
	getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, denID int64) (MercenaryDenDetail, error) {
		return MercenaryDenDetail{ID: 1, TypeID: 56674}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{56674: {ID: 56674, Name: map[string]string{"en": "Mercenary Den"}}}}

	handler := getCharacterMercenaryDenHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 1001, DenID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TypeName != "Mercenary Den" {
		t.Errorf("TypeName = %q, want %q", out.TypeName, "Mercenary Den")
	}
}
```

Write the analogous tests using these two vars directly (both already return the mapped MCP type, not a raw ESI model, so no oneOf/generated-model reconstruction is needed):

```go
func TestGetCorporationSkyhookHandler_ResolvesReagentTypeName(t *testing.T) {
	orig := getCorporationSkyhook
	t.Cleanup(func() { getCorporationSkyhook = orig })
	getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
		return SkyhookDetail{Reagents: []SkyhookReagent{{TypeID: 16273}}}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{16273: {ID: 16273, Name: map[string]string{"en": "Oxygen"}}}}

	handler := getCorporationSkyhookHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 1001, CorporationID: 1, SkyhookID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Reagents[0].TypeName != "Oxygen" {
		t.Errorf("Reagents[0].TypeName = %q, want %q", out.Reagents[0].TypeName, "Oxygen")
	}
}

func TestGetCorporationSovereigntyHubHandler_ResolvesTypeNames(t *testing.T) {
	orig := getCorporationSovereigntyHub
	t.Cleanup(func() { getCorporationSovereigntyHub = orig })
	getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
		return StructureSovereigntyHubDetail{
			ReagentBay: StructureSovereigntyHubReagentBay{Reagents: []StructureSovereigntyHubReagent{{TypeID: 16273}}},
			Upgrades:   []StructureSovereigntyHubUpgrade{{TypeID: 587}},
		}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		16273: {ID: 16273, Name: map[string]string{"en": "Oxygen"}},
		587:   {ID: 587, Name: map[string]string{"en": "Rifter"}},
	}}

	handler := getCorporationSovereigntyHubHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 1001, CorporationID: 1, SovereigntyHubID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ReagentBay.Reagents[0].TypeName != "Oxygen" {
		t.Errorf("ReagentBay.Reagents[0].TypeName = %q, want %q", out.ReagentBay.Reagents[0].TypeName, "Oxygen")
	}
	if out.Upgrades[0].TypeName != "Rifter" {
		t.Errorf("Upgrades[0].TypeName = %q, want %q", out.Upgrades[0].TypeName, "Rifter")
	}
}
```

(`getCorporationSkyhook` declared at `mcp/structures.go:214`, `getCorporationSovereigntyHub` at `mcp/structures.go:399` — both confirmed via `grep`.)

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "MercenaryDenHandler_ResolvesTypeName|SkyhookHandler_ResolvesReagentTypeName|SovereigntyHubHandler_ResolvesTypeNames" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment (three single-object handlers)**

`getCharacterMercenaryDenHandler`: add `sdeStore *sde.Store`; after the detail is fetched, single lookup:
```go
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{detail.TypeID})
		detail.TypeName = names[detail.TypeID]
```

`getCorporationSkyhookHandler`: add `sdeStore *sde.Store`; after the `Reagents` slice is populated on `out`, insert the standard flat-slice shape keyed on `out.Reagents[i].TypeID`/`.TypeName`.

`getCorporationSovereigntyHubHandler`: add `sdeStore *sde.Store`; after both `out.ReagentBay.Reagents` and `out.Upgrades` are populated, collect IDs from both in one pass, one `resolveTypeNames` call, then assign back to both:
```go
		ids := make([]int64, 0, len(out.ReagentBay.Reagents)+len(out.Upgrades))
		for _, r := range out.ReagentBay.Reagents {
			ids = append(ids, r.TypeID)
		}
		for _, u := range out.Upgrades {
			ids = append(ids, u.TypeID)
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out.ReagentBay.Reagents {
			out.ReagentBay.Reagents[i].TypeName = names[out.ReagentBay.Reagents[i].TypeID]
		}
		for i := range out.Upgrades {
			out.Upgrades[i].TypeName = names[out.Upgrades[i].TypeID]
		}
```

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/structures.go`'s imports.

In `mcp/server.go`, update all three call sites to pass `sdeStore`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "MercenaryDenHandler_ResolvesTypeName|SkyhookHandler_ResolvesReagentTypeName|SovereigntyHubHandler_ResolvesTypeNames" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/structures.go mcp/structures_internal_test.go mcp/server.go
git commit -m "feat: resolve mercenary den/skyhook/sovereignty hub type names via sde/"
```

---

### Task 18: `mcp/corporation_projects.go` — `ShipTypeMatcher.TypeID` (conditional oneOf)

**Files:**
- Modify: `mcp/corporation_projects.go` (`ShipTypeMatcher` lines 64-74; `getCorporationProjectDetailHandler`, which builds `ProjectConfiguration` via `toShipTypeMatchersShips`/`toShipTypeMatchersItems`)
- Modify: `mcp/server.go` (`get_corporation_project_detail` call site)
- Test: `mcp/corporation_projects_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

- [ ] **Step 1: Add the field**

```go
type ShipTypeMatcher struct {
	GroupID  *int64 `json:"groupId,omitempty"`
	TypeID   *int64 `json:"typeId,omitempty"`
	TypeName string `json:"typeName,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/corporation_projects_internal_test.go
func TestGetCorporationProjectDetailHandler_ResolvesShipTypeMatcherTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })
	typeID := int64(587)
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return &esiapi.CorporationsProjectsDetail{
			Configuration: esiapi.CorporationsProjectsDetailConfiguration{
				DamageShip: &esiapi.DamageShip{DamageShip: &esiapi.CorporationsProjectsDetailConfigurationdamageship{
					Ships: []esiapi.CorporationsProjectsDetailConfigurationdamageshipShipsInner{{TypeId: &esiapi.TypeId{TypeId: &typeID}}},
				}},
			},
			Creator: esiapi.CorporationsProjectsDetailCreator{Id: 998877, Name: "Creator Pilot"},
			Details: esiapi.CorporationsProjectsDetailDetails{Career: "Combat"},
			Id:      "abc-123",
			Name:    "Test Project",
			Progress: esiapi.CorporationsProjectsDetailProgress{Current: 0, Desired: 1},
			State:   "Active",
		}, nil
	}

	cache := &fakeTypeNameCache{types: map[int64]sde.Type{587: {ID: 587, Name: map[string]string{"en": "Rifter"}}}}
	handler := getCorporationProjectDetailHandler("v1", "client-id", store, cache)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Configuration.Ships[0].TypeName != "Rifter" {
		t.Errorf("Configuration.Ships[0].TypeName = %q, want %q", out.Configuration.Ships[0].TypeName, "Rifter")
	}
}
```

`getCorporationProjectDetail` is declared at `mcp/corporation_projects.go:607` — note `projectID` is `string`, not `int64` (confirmed via `grep`; the design's own doc comment mistakenly implied a numeric ID). The oneOf discriminator shape (`DamageShip *esiapi.DamageShip` → `.DamageShip *esiapi.CorporationsProjectsDetailConfigurationdamageship` → `.Ships []...ShipsInner` → `.TypeId *esiapi.TypeId` → `.TypeId *int64`) and the `keyring.MockInit()`/`config.NewStoreAt`/`store.Save` setup are copied verbatim from this file's existing `TestGetCorporationProjectDetailHandler_MapsFields` (line 120) and `TestToProjectConfiguration_AllVariants` (line 409) — confirmed directly against both the generated model and the existing test file, not guessed. `Configuration.Ships` on the *mapped* `ProjectConfiguration` output type is what `toProjectConfiguration`'s `DamageShip` branch (`Ships: toShipTypeMatchersShips(v.Ships)`) populates from this input.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetCorporationProjectDetailHandler_ResolvesShipTypeMatcherTypeName -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment (conditional — TypeID is `*int64`, only set on one arm of the oneOf)**

Change `getCorporationProjectDetailHandler`'s signature to add `sdeStore *sde.Store`. After `configuration` (containing `.Ships`/`.Items`/`.Materials []ShipTypeMatcher`) is built, insert:
```go
		var ids []int64
		for _, group := range [][]ShipTypeMatcher{configuration.Ships, configuration.Items, configuration.Materials} {
			for _, m := range group {
				if m.TypeID != nil {
					ids = append(ids, *m.TypeID)
				}
			}
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for _, group := range [][]ShipTypeMatcher{configuration.Ships, configuration.Items, configuration.Materials} {
			for i := range group {
				if group[i].TypeID != nil {
					group[i].TypeName = names[*group[i].TypeID]
				}
			}
		}
```
Confirm the exact field names on the struct that holds `Ships`/`Items`/`Materials` (referred to as `ProjectConfiguration` in the audit) before inserting — adjust `configuration.` prefixes to match the real local variable name in `getCorporationProjectDetailHandler`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/corporation_projects.go`'s imports. In `mcp/server.go`, update the `get_corporation_project_detail` call site: `getCorporationProjectDetailHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetCorporationProjectDetailHandler_ResolvesShipTypeMatcherTypeName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/corporation_projects.go mcp/corporation_projects_internal_test.go mcp/server.go
git commit -m "feat: resolve ShipTypeMatcher.TypeName via sde/"
```

---

### Task 19: `mcp/activities.go` — `GetMercenaryTacticalOperationOutput.DungeonTypeID`

**Files:**
- Modify: `mcp/activities.go` (`GetMercenaryTacticalOperationOutput` lines 109-116; `getMercenaryTacticalOperationHandler` lines 118-148)
- Modify: `mcp/server.go` (`get_mercenary_tactical_operation` call site)
- Test: `mcp/activities_internal_test.go`

**Interfaces:**
- Consumes: `resolveTypeNames` (Task 1).

**Verification caveat carried from the design spec**: confirm `DungeonTypeID` actually resolves against a real SDE `Type` record (dungeon type IDs are a different ID space from ship/item types in some FC data; this needs a real check, not an assumption) before or during this task. If manual verification (e.g. running `neat mcp` locally and calling `get_universe_type` with a known dungeon type ID from a live `get_mercenary_tactical_operation` response) shows it isn't a valid SDE `Type` entry, keep the code below anyway — `resolveTypeNames` silently omits unresolvable IDs, so `DungeonTypeName` simply stays empty in that case, which is the correct, already-designed-for outcome, not a bug requiring a different approach.

- [ ] **Step 1: Add the field**

```go
type GetMercenaryTacticalOperationOutput struct {
	CharacterID     int32     `json:"characterId"`
	OperationID     string    `json:"operationId"`
	DungeonTypeID   int64     `json:"dungeonTypeId"`
	DungeonTypeName string    `json:"dungeonTypeName,omitempty"`
	MercenaryDenID  int64     `json:"mercenaryDenId"`
	State           string    `json:"state"`
	Expires         time.Time `json:"expires"`
}
```

(`OperationDetailResult`, the unexported intermediate type, does **not** need the field — enrichment happens after `result` is unpacked into the JSON-facing output, in the handler.)

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/activities_internal_test.go
func TestGetMercenaryTacticalOperationHandler_ResolvesDungeonTypeName(t *testing.T) {
	orig := getMercenaryTacticalOperationDetail
	t.Cleanup(func() { getMercenaryTacticalOperationDetail = orig })
	getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
		return OperationDetailResult{DungeonTypeID: 34143, MercenaryDenID: 1}, nil
	}
	store := /* existing helper for character 1001 */
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{34143: {ID: 34143, Name: map[string]string{"en": "Some Dungeon"}}}}

	handler := getMercenaryTacticalOperationHandler("v1", "client", store, cache)
	_, out, err := handler(context.Background(), nil, GetMercenaryTacticalOperationInput{CharacterID: 1001, OperationID: "op-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.DungeonTypeName != "Some Dungeon" {
		t.Errorf("DungeonTypeName = %q, want %q", out.DungeonTypeName, "Some Dungeon")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetMercenaryTacticalOperationHandler_ResolvesDungeonTypeName -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `getMercenaryTacticalOperationHandler`'s signature to add `sdeStore *sde.Store`, and after `result, err := getMercenaryTacticalOperationDetail(...)` succeeds, insert:
```go
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{result.DungeonTypeID})
```
then change the existing return to include `DungeonTypeName: names[result.DungeonTypeID]` alongside the other fields already being assigned from `result`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/activities.go`'s imports. In `mcp/server.go`, update the `get_mercenary_tactical_operation` call site: `getMercenaryTacticalOperationHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetMercenaryTacticalOperationHandler_ResolvesDungeonTypeName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/activities.go mcp/activities_internal_test.go mcp/server.go
git commit -m "feat: resolve GetMercenaryTacticalOperationOutput.DungeonTypeName via sde/"
```

---

### Task 20: Docs sync

**Files:**
- Modify: `docs/reference/esi-coverage-roadmap.md`
- Modify: `CLAUDE.md` (package table's `mcp/` row)
- Modify: `VERSION`, `claude_resources/neat/.claude-plugin/plugin.json`, `claude_resources/neat/CHANGELOG.md` (this repo's `versiongate` requires a bump for any PR touching `mcp/*.go` — see Global Constraints in `CLAUDE.md`)

**Interfaces:** none — this is a pure docs/version-bump task, run last once every prior task has landed.

- [ ] **Step 1: Update the roadmap doc**

In `docs/reference/esi-coverage-roadmap.md`'s "Conventions" section, change:
```
- Every `esi/` method returns exactly what ESI returns, including bare IDs, with zero
  name-resolution enrichment — that's a `sde/`-backed follow-up once that package exists.
```
to:
```
- Every `esi/` method returns exactly what ESI returns, including bare IDs, with zero
  name-resolution enrichment. `type_id`-like fields across the non-Universe tags are enriched
  at the `mcp/` layer via `mcp/enrich.go`'s `resolveTypeNames` (see
  `docs/superpowers/specs/2026-07-28-sde-type-id-enrichment-design.md`) — location IDs and the
  rarer race/bloodline/faction/group IDs are still open follow-up batches.
```

- [ ] **Step 2: Update `CLAUDE.md`'s `mcp/` package table row**

Add a clause to the end of the `mcp/` row (after the existing Universe-tag sentence) noting `enrich.go`'s role, e.g.: "`mcp/enrich.go`'s `resolveTypeNames` additionally backs the `type_id`-like fields on ~20 non-Universe tools with sde-resolved names (see the type-id-enrichment design spec) — a shared, dedup'd lookup over the same `sde.Store.Type()` accessor Universe tools use, not a second cache."

- [ ] **Step 3: Bump VERSION and regenerate the plugin manifest**

```bash
current=$(cat VERSION)
# bump the patch component, e.g. 0.1.1 -> 0.1.2
echo "<incremented version>" > VERSION
go generate ./...
```

Add a changelog entry to `claude_resources/neat/CHANGELOG.md`:
```markdown
## <new version>
- No plugin-facing change; VERSION bump required by `versiongate` for the type_id-enrichment
  sweep (`mcp/*.go`, `mcp/enrich.go`).
```

- [ ] **Step 4: Verify everything is clean**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race && actionlint`
Expected: all clean, matching the state confirmed at the end of every prior task.

- [ ] **Step 5: Commit**

```bash
git add docs/reference/esi-coverage-roadmap.md CLAUDE.md VERSION claude_resources/neat/.claude-plugin/plugin.json claude_resources/neat/CHANGELOG.md
git commit -m "docs: document type_id enrichment in the roadmap and package table, bump VERSION"
```
