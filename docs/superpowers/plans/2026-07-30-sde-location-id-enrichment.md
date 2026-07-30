# SDE-backed location-ID enrichment (Batch 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add resolved `*Name` sibling fields for six location-entity ID kinds (`system_id`,
`region_id`, `constellation_id`, `station_id`, `planet_id`, `moon_id`) across the 14 non-Universe
`mcp/*.go` files that carry them, mirroring batch 1's `type_id` enrichment exactly.

**Architecture:** Six new resolver functions in `mcp/enrich.go` (`resolveSystemNames`,
`resolveRegionNames`, `resolveConstellationNames`, `resolveStationNames`, `resolvePlanetNames`,
`resolveMoonNames`), each a package-level `var` with its own narrow one-method interface over the
matching `sde.Store` accessor — identical shape to batch 1's `resolveTypeNames`/`typeNameCache`.
Three (`System`/`Region`/`Constellation`) read a localized `Name map[string]string` (`["en"]`);
three (`Station`/`Planet`/`Moon`) read a plain `Name string` — this is a real difference, not
uniform copy-paste.

**Tech Stack:** Go, `github.com/modelcontextprotocol/go-sdk/mcp`, `go.etcd.io/bbolt` (via `sde/`),
standard library `testing` only.

## Global Constraints

- Every new field is additive (`FooID`/`FooIDs` gains a sibling `FooName string` /
  `FooNames []string`, `json:"...,omitempty"`) — no existing field is removed or changes type.
- Handlers take the concrete `sdeStore *sde.Store` parameter (not an interface) — batch 1's settled
  convention. Every resolver is a package-level `var`, not a plain `func`, so tests can override it
  directly; tests pass `nil` for `*sde.Store` (safe — every `sde.Store` accessor has an explicit
  nil-receiver guard as its first line) and override the relevant resolver var when they need an
  actual resolved name.
- A lookup miss or error is silently omitted (empty `*Name`), never surfaced as a tool error — the
  existing `resolveTypeNames` contract, unchanged, applies identically to all six new resolvers.
- No live ESI fallback in any of the six new resolvers — same rate-limit-safety reasoning as batch
  1 (`list_market_orders`, `list_industry_systems`, `list_industry_facilities`,
  `list_sovereignty_systems` are all high-cardinality, unbounded-by-any-single-actor's-own-state
  public lists).
- A `0`-valued `int64` ID field (e.g. `MarketOrderBookEntry.SystemID` on a structure order,
  `IndustryJob.StationID` on a corp-scoped job) needs **no special-case guard** — `0` is never a
  real EVE entity ID, so looking it up is just another cache miss under the existing silent-omit
  contract, naturally producing an empty `*Name`.
- `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race` must stay clean after every
  task.
- Tests for unexported symbols go in the existing `*_internal_test.go` file for that package.
- **`mcp/freelance_jobs.go` needs NO changes in this batch.** `JobBroadcastLocation.SystemID` is
  paired with a `Name string` field that ESI already populates directly (`FreelanceJobsDetail
  Broadcastlocations.Name`, documented "Solar system name") — enriching it via `sde/` would just
  duplicate data ESI already provides for free. Confirmed directly against the vendored generated
  model; do not add a task for this file.

---

### Task 1: `mcp/enrich.go` — six new resolvers

**Files:**
- Modify: `mcp/enrich.go` (add six resolvers alongside the existing `resolveTypeNames`)
- Modify: `mcp/enrich_internal_test.go` (add six sets of tests alongside the existing four)

**Interfaces:**
- Produces: `resolveSystemNames`, `resolveRegionNames`, `resolveConstellationNames`,
  `resolveStationNames`, `resolvePlanetNames`, `resolveMoonNames` — every later task calls one of
  these six functions by its exact name, with signature
  `func(ctx context.Context, sdeStore <kind>NameCache, userAgent string, ids []int64) map[int64]string`.

- [ ] **Step 1: Write the failing tests**

Add to `mcp/enrich_internal_test.go` (below the existing `fakeTypeNameCache`/`TestResolveTypeNames_*`
block):

```go
type fakeSystemNameCache struct {
	systems map[int64]sde.System
	errs    map[int64]error
	calls   map[int64]int
}

func (f *fakeSystemNameCache) System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.System{}, false, err
	}
	s, ok := f.systems[id]
	return s, ok, nil
}

func TestResolveSystemNames_Dedup(t *testing.T) {
	cache := &fakeSystemNameCache{systems: map[int64]sde.System{
		30000142: {ID: 30000142, Name: map[string]string{"en": "Jita"}},
		30002187: {ID: 30002187, Name: map[string]string{"en": "Amarr"}},
	}}

	got := resolveSystemNames(context.Background(), cache, "test-agent", []int64{30000142, 30000142, 30002187})

	if got[30000142] != "Jita" {
		t.Errorf("got[30000142] = %q, want %q", got[30000142], "Jita")
	}
	if got[30002187] != "Amarr" {
		t.Errorf("got[30002187] = %q, want %q", got[30002187], "Amarr")
	}
	if cache.calls[30000142] != 1 {
		t.Errorf("calls[30000142] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[30000142])
	}
}

func TestResolveSystemNames_MissOmitted(t *testing.T) {
	cache := &fakeSystemNameCache{systems: map[int64]sde.System{}}

	got := resolveSystemNames(context.Background(), cache, "test-agent", []int64{99999999})

	if _, ok := got[99999999]; ok {
		t.Errorf("got[99999999] present = %v, want omitted on a cache miss", got[99999999])
	}
}

func TestResolveSystemNames_ErrorOmitted(t *testing.T) {
	cache := &fakeSystemNameCache{errs: map[int64]error{30000142: context.DeadlineExceeded}}

	got := resolveSystemNames(context.Background(), cache, "test-agent", []int64{30000142})

	if _, ok := got[30000142]; ok {
		t.Errorf("got[30000142] present = %v, want omitted when the lookup errors", got[30000142])
	}
}

func TestResolveSystemNames_EmptyIDs(t *testing.T) {
	cache := &fakeSystemNameCache{}

	got := resolveSystemNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}
```

Repeat the identical four-test shape for `resolveRegionNames` (fake type `fakeRegionNameCache`,
method `Region(ctx, userAgent, id) (sde.Region, bool, error)`, sample data e.g.
`10000002: {ID: 10000002, Name: map[string]string{"en": "The Forge"}}`), `resolveConstellationNames`
(`fakeConstellationNameCache`, `Constellation(...)`, e.g.
`20000020: {ID: 20000020, Name: map[string]string{"en": "Kimotoro"}}`), `resolveStationNames`
(`fakeStationNameCache`, `Station(...)`, e.g. `60003760: {ID: 60003760, Name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}` — **plain string, not a map**), `resolvePlanetNames`
(`fakePlanetNameCache`, `Planet(...)`, e.g. `40009086: {ID: 40009086, Name: "Jita IV"}` — plain
string), and `resolveMoonNames` (`fakeMoonNameCache`, `Moon(...)`, e.g.
`40009087: {ID: 40009087, Name: "Jita IV - Moon 4"}` — plain string). For the three plain-string
fakes, the `Name` field in the sample data and the assertion both use the bare string directly (no
`map[string]string{"en": ...}` wrapper) — this is the one place these tests genuinely differ from
each other, not a copy-paste-and-rename exercise.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestResolveSystemNames|TestResolveRegionNames|TestResolveConstellationNames|TestResolveStationNames|TestResolvePlanetNames|TestResolveMoonNames" -v`
Expected: FAIL — none of the six resolvers/interfaces/fakes exist yet.

- [ ] **Step 3: Implement the six resolvers**

Add to `mcp/enrich.go`, below the existing `resolveTypeNames`:

```go
// systemNameCache is the minimal read-only view of sde.Store.System() that
// resolveSystemNames needs.
type systemNameCache interface {
	System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error)
}

// resolveSystemNames looks up solar system names for a set of IDs, following
// resolveTypeNames's exact contract: dedup, drop a miss or error rather than
// erroring, never fall back to a live ESI call.
var resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
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
		s, ok, err := sdeStore.System(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = s.Name["en"]
	}
	return names
}

// regionNameCache is the minimal read-only view of sde.Store.Region() that
// resolveRegionNames needs.
type regionNameCache interface {
	Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error)
}

var resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
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
		r, ok, err := sdeStore.Region(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = r.Name["en"]
	}
	return names
}

// constellationNameCache is the minimal read-only view of
// sde.Store.Constellation() that resolveConstellationNames needs.
type constellationNameCache interface {
	Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error)
}

var resolveConstellationNames = func(ctx context.Context, sdeStore constellationNameCache, userAgent string, ids []int64) map[int64]string {
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
		c, ok, err := sdeStore.Constellation(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = c.Name["en"]
	}
	return names
}

// stationNameCache is the minimal read-only view of sde.Store.Station() that
// resolveStationNames needs.
type stationNameCache interface {
	Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error)
}

// resolveStationNames follows the same contract as resolveSystemNames, but
// sde.Station.Name is a plain string (station names are generated
// server-side, never localized) -- not a map[string]string like System's.
var resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
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
		s, ok, err := sdeStore.Station(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = s.Name
	}
	return names
}

// planetNameCache is the minimal read-only view of sde.Store.Planet() that
// resolvePlanetNames needs.
type planetNameCache interface {
	Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error)
}

// resolvePlanetNames follows resolveStationNames's plain-string shape --
// sde.Planet.Name is also a plain string, not a localized map.
var resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
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
		p, ok, err := sdeStore.Planet(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = p.Name
	}
	return names
}

// moonNameCache is the minimal read-only view of sde.Store.Moon() that
// resolveMoonNames needs.
type moonNameCache interface {
	Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error)
}

// resolveMoonNames follows resolveStationNames's plain-string shape --
// sde.Moon.Name is also a plain string, not a localized map.
var resolveMoonNames = func(ctx context.Context, sdeStore moonNameCache, userAgent string, ids []int64) map[int64]string {
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
		m, ok, err := sdeStore.Moon(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = m.Name
	}
	return names
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestResolveSystemNames|TestResolveRegionNames|TestResolveConstellationNames|TestResolveStationNames|TestResolvePlanetNames|TestResolveMoonNames" -v`
Expected: PASS (24 subtests total, four per resolver).

- [ ] **Step 5: Commit**

```bash
git add mcp/enrich.go mcp/enrich_internal_test.go
git commit -m "feat: add six location-entity name resolvers (system/region/constellation/station/planet/moon)"
```

---

### Task 2: `mcp/corporation.go` — four handlers

**Files:**
- Modify: `mcp/corporation.go` (`Starbase` struct + `listCorporationStarbasesHandler`; `Facility` +
  `listCorporationFacilitiesHandler`; `Structure` + `listCorporationStructuresHandler`;
  `GetCorporationOutput` + `getCorporationHandler`)
- Modify: `mcp/server.go` (one call site: `get_corporation` — the other three already pass
  `sdeStore`)
- Test: `mcp/corporation_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolveMoonNames`, `resolveStationNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type Starbase struct {
	MoonID          *int64     `json:"moonId,omitempty"`
	MoonName        string     `json:"moonName,omitempty"`
	OnlinedSince    *time.Time `json:"onlinedSince,omitempty"`
	ReinforcedUntil *time.Time `json:"reinforcedUntil,omitempty"`
	StarbaseID      int64      `json:"starbaseId"`
	State           *string    `json:"state,omitempty"`
	SystemID        int64      `json:"systemId"`
	SystemName      string     `json:"systemName,omitempty"`
	TypeID          int64      `json:"typeId"`
	TypeName        string     `json:"typeName,omitempty"`
	UnanchorAt      *time.Time `json:"unanchorAt,omitempty"`
}
```
```go
type Facility struct {
	FacilityID int64  `json:"facilityId"`
	SystemID   int64  `json:"systemId"`
	SystemName string `json:"systemName,omitempty"`
	TypeID     int64  `json:"typeId"`
	TypeName   string `json:"typeName,omitempty"`
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
	SystemName         string             `json:"systemName,omitempty"`
	TypeID             int64              `json:"typeId"`
	TypeName           string             `json:"typeName,omitempty"`
	UnanchorsAt        *time.Time         `json:"unanchorsAt,omitempty"`
}
```

In `GetCorporationOutput` (currently no `sdeStore`/enrichment at all), add `HomeStationName` right
after `HomeStationID`:
```go
	HomeStationID       int64      `json:"homeStationId"`
	HomeStationName     string     `json:"homeStationName,omitempty"`
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/corporation_internal_test.go
func TestListCorporationStarbasesHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStarbases
	t.Cleanup(func() { listCorporationStarbases = orig })
	moonID := int64(40009087)
	listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
		return []Starbase{{StarbaseID: 1, SystemID: 30000142, MoonID: &moonID}}, 1, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origMoon := resolveMoonNames
	t.Cleanup(func() { resolveMoonNames = origMoon })
	resolveMoonNames = func(ctx context.Context, sdeStore moonNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009087: "Jita IV - Moon 4"}
	}

	handler := listCorporationStarbasesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Starbases[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Starbases[0].SystemName, "Jita")
	}
	if out.Starbases[0].MoonName != "Jita IV - Moon 4" {
		t.Errorf("MoonName = %q, want %q", out.Starbases[0].MoonName, "Jita IV - Moon 4")
	}
}

func TestListCorporationFacilitiesHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFacilities
	t.Cleanup(func() { listCorporationFacilities = orig })
	listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
		return []Facility{{FacilityID: 1, SystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationFacilitiesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Facilities[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Facilities[0].SystemName, "Jita")
	}
}

func TestListCorporationStructuresHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStructures
	t.Cleanup(func() { listCorporationStructures = orig })
	listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
		return []Structure{{StructureID: 1, SystemID: 30000142}}, 1, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationStructuresHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Structures[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Structures[0].SystemName, "Jita")
	}
}

func TestGetCorporationHandler_ResolvesHomeStationName(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{CorporationID: 98000001, HomeStationID: 60003760}, nil
	}

	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := getCorporationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.HomeStationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("HomeStationName = %q, want %q", out.HomeStationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestListCorporationStarbasesHandler_ResolvesLocationNames|TestListCorporationFacilitiesHandler_ResolvesSystemName|TestListCorporationStructuresHandler_ResolvesSystemName|TestGetCorporationHandler_ResolvesHomeStationName" -v`
Expected: FAIL — `getCorporationHandler` doesn't accept a 2nd argument yet, and the other three
handlers' output structs don't have the new fields.

- [ ] **Step 4: Implement enrichment**

`listCorporationStarbasesHandler` already has `sdeStore *sde.Store` (batch 1). After the existing
`TypeName` loop, add a second pass for `SystemID`/`MoonID`:
```go
		systemIDs := make([]int64, len(starbases))
		var moonIDs []int64
		for i, s := range starbases {
			systemIDs[i] = s.SystemID
			if s.MoonID != nil {
				moonIDs = append(moonIDs, *s.MoonID)
			}
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		moonNames := resolveMoonNames(ctx, sdeStore, sso.UserAgent(version), moonIDs)
		for i := range starbases {
			starbases[i].SystemName = systemNames[starbases[i].SystemID]
			if starbases[i].MoonID != nil {
				starbases[i].MoonName = moonNames[*starbases[i].MoonID]
			}
		}
```

`listCorporationFacilitiesHandler`/`listCorporationStructuresHandler` already have `sdeStore`.
After each existing `TypeName` loop, add the standard flat-slice `resolveSystemNames` shape on
`SystemID`/`SystemName`.

`getCorporationHandler` currently has no `sdeStore` at all. Change its signature to
`getCorporationHandler(version string, sdeStore *sde.Store)`, and after `out, err :=
getCorporationDetail(...)` succeeds, insert:
```go
		names := resolveStationNames(ctx, sdeStore, sso.UserAgent(version), []int64{out.HomeStationID})
		out.HomeStationName = names[out.HomeStationID]
```
before the existing `return nil, out, nil`.

In `mcp/server.go`, update the `get_corporation` call site: `getCorporationHandler(version,
sdeStore)`. The other three call sites already pass `sdeStore` from batch 1 — no change needed
there.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestListCorporationStarbasesHandler_ResolvesLocationNames|TestListCorporationFacilitiesHandler_ResolvesSystemName|TestListCorporationStructuresHandler_ResolvesSystemName|TestGetCorporationHandler_ResolvesHomeStationName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean (update the pre-existing `TestGetCorporationHandler_*` tests to pass a `nil` 2nd
argument).

- [ ] **Step 7: Commit**

```bash
git add mcp/corporation.go mcp/corporation_internal_test.go mcp/server.go
git commit -m "feat: resolve corporation-tag location IDs via sde/"
```

---

### Task 3: `mcp/corporation_projects.go` — `LocationMatcher`/`DockingLocationMatcher`

**Files:**
- Modify: `mcp/corporation_projects.go`
- Modify: `mcp/server.go` (no change expected — `get_corporation_project_detail` already passes
  `sdeStore`; verify)
- Test: `mcp/corporation_projects_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolveRegionNames`, `resolveConstellationNames`,
  `resolveStationNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type LocationMatcher struct {
	ConstellationID   *int64 `json:"constellationId,omitempty"`
	ConstellationName string `json:"constellationName,omitempty"`
	RegionID          *int64 `json:"regionId,omitempty"`
	RegionName        string `json:"regionName,omitempty"`
	SolarSystemID     *int64 `json:"solarSystemId,omitempty"`
	SolarSystemName   string `json:"solarSystemName,omitempty"`
}
```
```go
type DockingLocationMatcher struct {
	StationID   *int64 `json:"stationId,omitempty"`
	StationName string `json:"stationName,omitempty"`
	StructureID *int64 `json:"structureId,omitempty"`
}
```
(`StructureID` gets no `*Name` sibling — player structures aren't SDE data, out of scope per the
design's explicit exclusion.)

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/corporation_projects_internal_test.go
func TestGetCorporationProjectDetailHandler_ResolvesLocationMatcherNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })
	solarSystemID := int64(30000142)
	stationID := int64(60003760)
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return &esiapi.CorporationsProjectsDetail{
			Configuration: esiapi.CorporationsProjectsDetailConfiguration{
				CaptureFwComplex: &esiapi.CaptureFwComplex{CaptureFwComplex: &esiapi.CorporationsProjectsDetailConfigurationcapturefwcomplex{
					Locations: []esiapi.CorporationsProjectsDetailConfigurationcapturefwcomplexLocationsInner{
						{SolarSystemId: &esiapi.SolarSystemId{SolarSystemId: &solarSystemID}},
					},
				}},
				DeliverItem: &esiapi.DeliverItem{DeliverItem: &esiapi.CorporationsProjectsDetailConfigurationdeliveritem{
					DockingLocations: []esiapi.CorporationsProjectsDetailConfigurationdeliveritemDockingLocationsInner{
						{StationId: &esiapi.StationId{StationId: &stationID}},
					},
				}},
			},
			Creator:  esiapi.CorporationsProjectsDetailCreator{Id: 998877, Name: "Creator Pilot"},
			Details:  esiapi.CorporationsProjectsDetailDetails{Career: "Exploration"},
			Id:       "abc-123",
			Name:     "Test Project",
			Progress: esiapi.CorporationsProjectsDetailProgress{Current: 0, Desired: 1},
			State:    "Active",
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Configuration.Locations[0].SolarSystemName != "Jita" {
		t.Errorf("Locations[0].SolarSystemName = %q, want %q", out.Configuration.Locations[0].SolarSystemName, "Jita")
	}
	if out.Configuration.DockingLocations[0].StationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("DockingLocations[0].StationName = %q, want %q", out.Configuration.DockingLocations[0].StationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}
```

Before writing this, confirm the exact `CorporationsProjectsDetailConfigurationcapturefwcomplex
LocationsInner`/`SolarSystemId`/`RegionId`/`ConstellationId`/`CorporationsProjectsDetail
Configurationdeliveritem`/`DockingLocationsInner`/`StationId` generated-model shapes against the
vendored `goesi-openapi` package directly (mirroring the level of care batch 1's Task 18 applied to
`ShipTypeMatcher`'s `TypeId` wrapper) — the field names above follow the same `XId{XId *int64}`
wrapper convention `TypeId`/`GroupId` used, but verify before assuming.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetCorporationProjectDetailHandler_ResolvesLocationMatcherNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment (conditional — all four ID fields are `*int64`, nil-guard on
  both collect and assign)**

After the existing batch-1 `ShipTypeMatcher` enrichment block (which stays untouched), add a second
block covering `Locations`/`DockingLocations`:
```go
		var systemIDs, regionIDs, constellationIDs, stationIDs []int64
		for _, l := range configuration.Locations {
			if l.SolarSystemID != nil {
				systemIDs = append(systemIDs, *l.SolarSystemID)
			}
			if l.RegionID != nil {
				regionIDs = append(regionIDs, *l.RegionID)
			}
			if l.ConstellationID != nil {
				constellationIDs = append(constellationIDs, *l.ConstellationID)
			}
		}
		for _, d := range configuration.DockingLocations {
			if d.StationID != nil {
				stationIDs = append(stationIDs, *d.StationID)
			}
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		regionNames := resolveRegionNames(ctx, sdeStore, sso.UserAgent(version), regionIDs)
		constellationNames := resolveConstellationNames(ctx, sdeStore, sso.UserAgent(version), constellationIDs)
		stationNames := resolveStationNames(ctx, sdeStore, sso.UserAgent(version), stationIDs)
		for i := range configuration.Locations {
			if configuration.Locations[i].SolarSystemID != nil {
				configuration.Locations[i].SolarSystemName = systemNames[*configuration.Locations[i].SolarSystemID]
			}
			if configuration.Locations[i].RegionID != nil {
				configuration.Locations[i].RegionName = regionNames[*configuration.Locations[i].RegionID]
			}
			if configuration.Locations[i].ConstellationID != nil {
				configuration.Locations[i].ConstellationName = constellationNames[*configuration.Locations[i].ConstellationID]
			}
		}
		for i := range configuration.DockingLocations {
			if configuration.DockingLocations[i].StationID != nil {
				configuration.DockingLocations[i].StationName = stationNames[*configuration.DockingLocations[i].StationID]
			}
		}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetCorporationProjectDetailHandler_ResolvesLocationMatcherNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/corporation_projects.go mcp/corporation_projects_internal_test.go
git commit -m "feat: resolve LocationMatcher/DockingLocationMatcher location names via sde/"
```

---

### Task 4: `mcp/activities.go` — `SkyhookSummary`

**Files:**
- Modify: `mcp/activities.go`
- Modify: `mcp/server.go` (`list_raidable_skyhooks` call site)
- Test: `mcp/activities_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolvePlanetNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type SkyhookSummary struct {
	PlanetID                int64     `json:"planetId"`
	PlanetName              string    `json:"planetName,omitempty"`
	SolarSystemID           int64     `json:"solarSystemId"`
	SolarSystemName         string    `json:"solarSystemName,omitempty"`
	TheftVulnerabilityStart time.Time `json:"theftVulnerabilityStart"`
	TheftVulnerabilityEnd   time.Time `json:"theftVulnerabilityEnd"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/activities_internal_test.go
func TestListRaidableSkyhooksHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listRaidableSkyhooks
	t.Cleanup(func() { listRaidableSkyhooks = orig })
	listRaidableSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkyhookSummary, error) {
		return []SkyhookSummary{{PlanetID: 40009086, SolarSystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := listRaidableSkyhooksHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListRaidableSkyhooksInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Skyhooks[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Skyhooks[0].SolarSystemName, "Jita")
	}
	if out.Skyhooks[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Skyhooks[0].PlanetName, "Jita IV")
	}
}
```

Confirm the exact field name for the skyhooks slice on `ListRaidableSkyhooksOutput` (likely
`Skyhooks`) against the file before writing the assertion — the audit didn't capture the output
wrapper type's field name, only `SkyhookSummary` itself.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListRaidableSkyhooksHandler_ResolvesLocationNames -v`
Expected: FAIL — handler doesn't accept a 4th argument yet.

- [ ] **Step 4: Implement enrichment**

Change `listRaidableSkyhooksHandler`'s signature to add `sdeStore *sde.Store`, and after the
existing `listRaidableSkyhooks(...)` call succeeds, insert:
```go
		systemIDs := make([]int64, len(skyhooks))
		planetIDs := make([]int64, len(skyhooks))
		for i, s := range skyhooks {
			systemIDs[i] = s.SolarSystemID
			planetIDs[i] = s.PlanetID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		planetNames := resolvePlanetNames(ctx, sdeStore, sso.UserAgent(version), planetIDs)
		for i := range skyhooks {
			skyhooks[i].SolarSystemName = systemNames[skyhooks[i].SolarSystemID]
			skyhooks[i].PlanetName = planetNames[skyhooks[i].PlanetID]
		}
```
before the existing return.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/activities.go`'s imports if not
already present (it should already be there from batch 1's `DungeonTypeName` work — verify, don't
duplicate).

In `mcp/server.go`, update the `list_raidable_skyhooks` call site: `listRaidableSkyhooksHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListRaidableSkyhooksHandler_ResolvesLocationNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/activities.go mcp/activities_internal_test.go mcp/server.go
git commit -m "feat: resolve SkyhookSummary location names via sde/"
```

---

### Task 5: `mcp/faction_warfare.go` — `FwSystem`

**Files:**
- Modify: `mcp/faction_warfare.go`
- Modify: `mcp/server.go` (`list_fw_systems` call site)
- Test: `mcp/faction_warfare_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames` (Task 1).

- [ ] **Step 1: Add the field**

```go
type FwSystem struct {
	Contested              string `json:"contested"`
	OccupierFactionID      int64  `json:"occupierFactionId"`
	OwnerFactionID         int64  `json:"ownerFactionId"`
	SolarSystemID          int64  `json:"solarSystemId"`
	SolarSystemName        string `json:"solarSystemName,omitempty"`
	VictoryPoints          int64  `json:"victoryPoints"`
	VictoryPointsThreshold int64  `json:"victoryPointsThreshold"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/faction_warfare_internal_test.go
func TestListFwSystemsHandler_ResolvesSystemName(t *testing.T) {
	orig := listFwSystems
	t.Cleanup(func() { listFwSystems = orig })
	listFwSystems = func(ctx context.Context, userAgent string) ([]FwSystem, error) {
		return []FwSystem{{SolarSystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listFwSystemsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListFwSystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Systems[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Systems[0].SolarSystemName, "Jita")
	}
}
```

Confirm the exact overridable var name (`listFwSystems`) and the output wrapper's field name
(likely `Systems`) against the file before writing this — the audit captured the handler's own
construction loop but not the underlying data-fetch var's declared name with full confidence.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListFwSystemsHandler_ResolvesSystemName -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `listFwSystemsHandler`'s signature to `listFwSystemsHandler(version string, sdeStore
*sde.Store)`, and after the systems slice is built, insert the standard flat-slice
`resolveSystemNames` shape on `SolarSystemID`/`SolarSystemName`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/faction_warfare.go`'s imports.

In `mcp/server.go`, update the `list_fw_systems` call site: `listFwSystemsHandler(version, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListFwSystemsHandler_ResolvesSystemName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/faction_warfare.go mcp/faction_warfare_internal_test.go mcp/server.go
git commit -m "feat: resolve FwSystem.SolarSystemName via sde/"
```

---

### Task 6: `mcp/killmails.go` — `GetKillmailOutput`

**Files:**
- Modify: `mcp/killmails.go`
- Modify: `mcp/server.go` (verify `get_killmail`'s call site needs no change — it already passes
  `sdeStore`)
- Test: `mcp/killmails_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolveMoonNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type GetKillmailOutput struct {
	KillmailID      int64              `json:"killmailId"`
	KillmailTime    time.Time          `json:"killmailTime"`
	SolarSystemID   int64              `json:"solarSystemId"`
	SolarSystemName string             `json:"solarSystemName,omitempty"`
	MoonID          *int64             `json:"moonId,omitempty"`
	MoonName        string             `json:"moonName,omitempty"`
	WarID           *int64             `json:"warId,omitempty"`
	Attackers       []KillmailAttacker `json:"attackers"`
	Victim          KillmailVictim     `json:"victim"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/killmails_internal_test.go
func TestGetKillmailHandler_ResolvesLocationNames(t *testing.T) {
	orig := getKillmail
	t.Cleanup(func() { getKillmail = orig })
	moonID := int64(40009087)
	getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
		return &esiapi.KillmailsKillmailIdKillmailHashGet{
			SolarSystemId: 30000142,
			MoonId:        &moonID,
			Victim:        esiapi.KillmailsKillmailIdKillmailHashGetVictim{ShipTypeId: 587},
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origMoon := resolveMoonNames
	t.Cleanup(func() { resolveMoonNames = origMoon })
	resolveMoonNames = func(ctx context.Context, sdeStore moonNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009087: "Jita IV - Moon 4"}
	}

	handler := getKillmailHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetKillmailInput{KillmailID: 1, KillmailHash: "abc"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.SolarSystemName, "Jita")
	}
	if out.MoonName != "Jita IV - Moon 4" {
		t.Errorf("MoonName = %q, want %q", out.MoonName, "Jita IV - Moon 4")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetKillmailHandler_ResolvesLocationNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment (separate resolver calls from the existing `resolveTypeNames`
  call — System/Moon are different entity kinds from Type, so they need their own `ids`/`names`)**

Insert a new block right before the existing `return nil, GetKillmailOutput{...}, nil`:
```go
		var locationIDs []int64
		locationIDs = append(locationIDs, detail.SolarSystemId)
		if detail.MoonId != nil {
			locationIDs = append(locationIDs, *detail.MoonId)
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), []int64{detail.SolarSystemId})
		var moonNames map[int64]string
		if detail.MoonId != nil {
			moonNames = resolveMoonNames(ctx, sdeStore, sso.UserAgent(version), []int64{*detail.MoonId})
		}
```
then add `SolarSystemName: systemNames[detail.SolarSystemId]` and, conditionally, `MoonName:
moonNames[*detail.MoonId]` (guarded by `detail.MoonId != nil`, since `moonNames` may be `nil`) to
the final `GetKillmailOutput{...}` composite literal alongside the existing `SolarSystemID:
detail.SolarSystemId, MoonID: detail.MoonId, ...` fields. (The unused `locationIDs` variable above
is a scratch note, not real code — write the two separate `resolveSystemNames`/`resolveMoonNames`
calls directly as shown, one per entity kind, each with its own minimal `ids` slice; do not
attempt to merge System and Moon IDs into one call, since they're different entity kinds requiring
different resolvers.)

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetKillmailHandler_ResolvesLocationNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/killmails.go mcp/killmails_internal_test.go
git commit -m "feat: resolve killmail solar system/moon names via sde/"
```

---

### Task 7: `mcp/fleets.go` — `FleetMember`

**Files:**
- Modify: `mcp/fleets.go`
- Modify: `mcp/server.go` (verify `list_fleet_members`'s call site needs no change — already passes
  `sdeStore`)
- Test: `mcp/fleets_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolveStationNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type FleetMember struct {
	CharacterID     int64     `json:"characterId"`
	JoinTime        time.Time `json:"joinTime"`
	Role            string    `json:"role"`
	RoleName        string    `json:"roleName"`
	ShipTypeID      int64     `json:"shipTypeId"`
	ShipTypeName    string    `json:"shipTypeName,omitempty"`
	SolarSystemID   int64     `json:"solarSystemId"`
	SolarSystemName string    `json:"solarSystemName,omitempty"`
	SquadID         int64     `json:"squadId"`
	StationID       *int64    `json:"stationId,omitempty"`
	StationName     string    `json:"stationName,omitempty"`
	TakesFleetWarp  bool      `json:"takesFleetWarp"`
	WingID          int64     `json:"wingId"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/fleets_internal_test.go
func TestListFleetMembersHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listFleetMembers
	t.Cleanup(func() { listFleetMembers = orig })
	stationID := int64(60003760)
	listFleetMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetMember, error) {
		return []FleetMember{{CharacterID: 2001, SolarSystemID: 30000142, StationID: &stationID}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := listFleetMembersHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListFleetMembersInput{CharacterID: 7, FleetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Members[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Members[0].SolarSystemName, "Jita")
	}
	if out.Members[0].StationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("StationName = %q, want %q", out.Members[0].StationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListFleetMembersHandler_ResolvesLocationNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

After the existing `ShipTypeName` enrichment loop in `listFleetMembersHandler`, add:
```go
		systemIDs := make([]int64, len(members))
		var stationIDs []int64
		for i, m := range members {
			systemIDs[i] = m.SolarSystemID
			if m.StationID != nil {
				stationIDs = append(stationIDs, *m.StationID)
			}
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		stationNames := resolveStationNames(ctx, sdeStore, sso.UserAgent(version), stationIDs)
		for i := range members {
			members[i].SolarSystemName = systemNames[members[i].SolarSystemID]
			if members[i].StationID != nil {
				members[i].StationName = stationNames[*members[i].StationID]
			}
		}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListFleetMembersHandler_ResolvesLocationNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/fleets.go mcp/fleets_internal_test.go
git commit -m "feat: resolve FleetMember location names via sde/"
```

---

### Task 8: `mcp/location.go` — `GetCharacterLocationOutput`

**Files:**
- Modify: `mcp/location.go`
- Modify: `mcp/server.go` (`get_character_location` call site)
- Test: `mcp/location_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolveStationNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type GetCharacterLocationOutput struct {
	CharacterID     int32  `json:"characterId"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
	StationID       *int64 `json:"stationId,omitempty"`
	StationName     string `json:"stationName,omitempty"`
	StructureID     *int64 `json:"structureId,omitempty"`
}
```
(`StructureID` gets no `*Name` sibling — same exclusion as `DockingLocationMatcher.StructureID`.)

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/location_internal_test.go
func TestGetCharacterLocationHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterLocation
	t.Cleanup(func() { getCharacterLocation = orig })
	stationID := int64(60003760)
	getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
		return 30000142, &stationID, nil, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := getCharacterLocationHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterLocationInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.SolarSystemName, "Jita")
	}
	if out.StationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("StationName = %q, want %q", out.StationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}
```

Before writing this, confirm the exact overridable var's real name and signature (the audit
captured the handler's final `return` line but not the data-fetch var's own declared name/params
with full confidence) against the actual file — adjust the stub's signature to match exactly.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetCharacterLocationHandler_ResolvesLocationNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `getCharacterLocationHandler`'s signature to add `sdeStore *sde.Store` as a 4th parameter.
After the existing data-fetch call succeeds, insert:
```go
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), []int64{solarSystemID})
		var stationName string
		var stationNames map[int64]string
		if stationID != nil {
			stationNames = resolveStationNames(ctx, sdeStore, sso.UserAgent(version), []int64{*stationID})
			stationName = stationNames[*stationID]
		}
```
then add `SolarSystemName: systemNames[solarSystemID]` and `StationName: stationName` to the
existing `GetCharacterLocationOutput{...}` composite literal.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/location.go`'s imports if not
already present (it should be, from batch 1's `get_character_ship` work in the same file — verify,
don't duplicate).

In `mcp/server.go`, update the `get_character_location` call site: `getCharacterLocationHandler(version, clientID, store, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetCharacterLocationHandler_ResolvesLocationNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/location.go mcp/location_internal_test.go mcp/server.go
git commit -m "feat: resolve GetCharacterLocationOutput location names via sde/"
```

---

### Task 9: `mcp/incursions.go` — `Incursion`

**Files:**
- Modify: `mcp/incursions.go`
- Modify: `mcp/server.go` (`list_incursions` call site)
- Test: `mcp/incursions_internal_test.go`

**Interfaces:**
- Consumes: `resolveConstellationNames`, `resolveSystemNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type Incursion struct {
	ConstellationID          int64    `json:"constellationId"`
	ConstellationName        string   `json:"constellationName,omitempty"`
	FactionID                int64    `json:"factionId"`
	HasBoss                  bool     `json:"hasBoss"`
	InfestedSolarSystemIDs   []int64  `json:"infestedSolarSystemIds"`
	InfestedSolarSystemNames []string `json:"infestedSolarSystemNames,omitempty"`
	Influence                float64  `json:"influence"`
	StagingSolarSystemID     int64    `json:"stagingSolarSystemId"`
	StagingSolarSystemName   string   `json:"stagingSolarSystemName,omitempty"`
	State                    string   `json:"state"`
	Type                     string   `json:"type"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/incursions_internal_test.go
func TestListIncursionsHandler_ResolvesLocationNames(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return []Incursion{{
			ConstellationID: 20000020, StagingSolarSystemID: 30000142,
			InfestedSolarSystemIDs: []int64{30000142, 30000144},
		}}, nil
	}

	origConstellation := resolveConstellationNames
	t.Cleanup(func() { resolveConstellationNames = origConstellation })
	resolveConstellationNames = func(ctx context.Context, sdeStore constellationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{20000020: "Kimotoro"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter"}
	}

	handler := listIncursionsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListIncursionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Incursions[0].ConstellationName != "Kimotoro" {
		t.Errorf("ConstellationName = %q, want %q", out.Incursions[0].ConstellationName, "Kimotoro")
	}
	if out.Incursions[0].StagingSolarSystemName != "Jita" {
		t.Errorf("StagingSolarSystemName = %q, want %q", out.Incursions[0].StagingSolarSystemName, "Jita")
	}
	want := []string{"Jita", "Perimeter"}
	if len(out.Incursions[0].InfestedSolarSystemNames) != len(want) {
		t.Fatalf("InfestedSolarSystemNames = %v, want %v", out.Incursions[0].InfestedSolarSystemNames, want)
	}
	for i, w := range want {
		if out.Incursions[0].InfestedSolarSystemNames[i] != w {
			t.Errorf("InfestedSolarSystemNames[%d] = %q, want %q", i, out.Incursions[0].InfestedSolarSystemNames[i], w)
		}
	}
}
```

Also add a companion partial-miss test (per batch 1's final review's own recommendation to build
this in proactively for index-aligned lists, not discover the gap after merge):
```go
func TestListIncursionsHandler_InfestedSolarSystemNamesSurvivesPartialMiss(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return []Incursion{{InfestedSolarSystemIDs: []int64{30000142, 999999999, 30000144}}}, nil
	}

	origConstellation := resolveConstellationNames
	t.Cleanup(func() { resolveConstellationNames = origConstellation })
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// 999999999 is deliberately omitted -- a miss must not shift neighbors.
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter"}
	}

	handler := listIncursionsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListIncursionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []string{"Jita", "", "Perimeter"}
	if len(out.Incursions[0].InfestedSolarSystemNames) != len(want) {
		t.Fatalf("InfestedSolarSystemNames = %v, want %v", out.Incursions[0].InfestedSolarSystemNames, want)
	}
	for i, w := range want {
		if out.Incursions[0].InfestedSolarSystemNames[i] != w {
			t.Errorf("InfestedSolarSystemNames[%d] = %q, want %q", i, out.Incursions[0].InfestedSolarSystemNames[i], w)
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestListIncursionsHandler_ResolvesLocationNames|TestListIncursionsHandler_InfestedSolarSystemNamesSurvivesPartialMiss" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `listIncursionsHandler`'s signature to `listIncursionsHandler(version string, sdeStore
*sde.Store)`, and after the incursions slice is built, insert:
```go
		var constellationIDs, systemIDs []int64
		for _, inc := range incursions {
			constellationIDs = append(constellationIDs, inc.ConstellationID)
			systemIDs = append(systemIDs, inc.StagingSolarSystemID)
			systemIDs = append(systemIDs, inc.InfestedSolarSystemIDs...)
		}
		constellationNames := resolveConstellationNames(ctx, sdeStore, sso.UserAgent(version), constellationIDs)
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		for i := range incursions {
			incursions[i].ConstellationName = constellationNames[incursions[i].ConstellationID]
			incursions[i].StagingSolarSystemName = systemNames[incursions[i].StagingSolarSystemID]
			infestedNames := make([]string, len(incursions[i].InfestedSolarSystemIDs))
			for j, id := range incursions[i].InfestedSolarSystemIDs {
				infestedNames[j] = systemNames[id]
			}
			incursions[i].InfestedSolarSystemNames = infestedNames
		}
```

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/incursions.go`'s imports.

In `mcp/server.go`, update the `list_incursions` call site: `listIncursionsHandler(version, sdeStore)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestListIncursionsHandler_ResolvesLocationNames|TestListIncursionsHandler_InfestedSolarSystemNamesSurvivesPartialMiss" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/incursions.go mcp/incursions_internal_test.go mcp/server.go
git commit -m "feat: resolve Incursion location names via sde/"
```

---

### Task 10: `mcp/market.go` — six handlers, two structs

**Files:**
- Modify: `mcp/market.go` (`MarketOrder.RegionID` via 4 mapper functions + handlers;
  `MarketOrderBookEntry.SystemID` via 2 mapper functions + handlers)
- Modify: `mcp/server.go` (verify all six call sites need no change — all already pass `sdeStore`
  from batch 1)
- Test: `mcp/market_internal_test.go`

**Interfaces:**
- Consumes: `resolveRegionNames`, `resolveSystemNames` (Task 1).

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
	RegionName     string    `json:"regionName,omitempty"`
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
	SystemName   string    `json:"systemName,omitempty"`
	TypeID       int64     `json:"typeId"`
	TypeName     string    `json:"typeName,omitempty"`
	VolumeRemain int64     `json:"volumeRemain"`
	VolumeTotal  int64     `json:"volumeTotal"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/market_internal_test.go
func TestListCharacterMarketOrdersHandler_ResolvesRegionName(t *testing.T) {
	orig := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = orig })
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MarketOrder, error) {
		return []MarketOrder{{OrderID: 1, RegionID: 10000002}}, nil
	}
	store := /* existing helper for character 7 -- check this file's convention before writing */

	origRegion := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origRegion })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Orders[0].RegionName != "The Forge" {
		t.Errorf("RegionName = %q, want %q", out.Orders[0].RegionName, "The Forge")
	}
}

func TestListMarketOrdersHandler_ResolvesSystemName(t *testing.T) {
	orig := listMarketOrders
	t.Cleanup(func() { listMarketOrders = orig })
	listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, typeID *int64, page int32) ([]MarketOrderBookEntry, int32, error) {
		return []MarketOrderBookEntry{{OrderID: 1, SystemID: 30000142}}, 1, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listMarketOrdersHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Orders[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Orders[0].SystemName, "Jita")
	}
}

func TestListStructureMarketOrdersHandler_SystemNameEmptyWhenSystemIDZero(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = orig })
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]MarketOrderBookEntry, int32, error) {
		return []MarketOrderBookEntry{{OrderID: 1}}, 1, nil // SystemID left zero, matching the real mapper
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	calls := 0
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		calls++
		return map[int64]string{0: "should never be assigned"}
	}

	handler := listStructureMarketOrdersHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 7, StructureID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Orders[0].SystemName != "" {
		t.Errorf("SystemName = %q, want empty (SystemID is 0 for structure orders)", out.Orders[0].SystemName)
	}
}
```

Confirm this file's exact existing config-store test helper convention (character ID, `keyring.
MockInit`/`config.NewStoreAt` pattern already used elsewhere in `market_internal_test.go`) and the
real parameter lists for `listMarketOrders`/`listStructureMarketOrders`/`listCharacterMarketOrders`
(the audit gives the shape but double-check parameter names/types against the actual file) before
finalizing these tests. Write the analogous `_ResolvesRegionName` test for
`listCharacterMarketOrderHistoryHandler`, `listCorporationMarketOrdersHandler`,
`listCorporationMarketOrderHistoryHandler` (all four share the identical `RegionID`→`RegionName`
shape).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "MarketOrdersHandler_ResolvesRegionName|MarketOrdersHandler_ResolvesSystemName|SystemNameEmptyWhenSystemIDZero" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment in all six handlers**

For the four `MarketOrder`-returning handlers, after the existing `TypeName` enrichment loop, add
the standard flat-slice `resolveRegionNames` shape on `RegionID`/`RegionName`.

For `listMarketOrdersHandler`/`listStructureMarketOrdersHandler`, after the existing `TypeName`
loop, add the standard flat-slice `resolveSystemNames` shape on `SystemID`/`SystemName` — no
special-case guard needed for the structure-orders case where `SystemID` is always `0`; the lookup
naturally misses and `SystemName` naturally stays empty.

No `mcp/server.go` changes needed — all six call sites already pass `sdeStore` from batch 1.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "MarketOrdersHandler_ResolvesRegionName|MarketOrdersHandler_ResolvesSystemName|SystemNameEmptyWhenSystemIDZero" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/market.go mcp/market_internal_test.go
git commit -m "feat: resolve MarketOrder/MarketOrderBookEntry location names via sde/"
```

---

### Task 11: `mcp/planetary_interaction.go` — `Planet`, `CustomsOffice`

**Files:**
- Modify: `mcp/planetary_interaction.go`
- Modify: `mcp/server.go` (`list_character_planets` call site — new; `list_corporation_customs_offices` already passes `sdeStore`)
- Test: `mcp/planetary_interaction_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolvePlanetNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type Planet struct {
	PlanetID        int64     `json:"planetId"`
	PlanetName      string    `json:"planetName,omitempty"`
	SolarSystemID   int64     `json:"solarSystemId"`
	SolarSystemName string    `json:"solarSystemName,omitempty"`
	PlanetType      string    `json:"planetType"`
	OwnerID         int64     `json:"ownerId"`
	UpgradeLevel    int64     `json:"upgradeLevel"`
	NumPins         int64     `json:"numPins"`
	LastUpdate      time.Time `json:"lastUpdate"`
}
```
```go
type CustomsOffice struct {
	OfficeID                 int64    `json:"officeId"`
	SystemID                 int64    `json:"systemId"`
	SystemName               string   `json:"systemName,omitempty"`
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
func TestListCharacterPlanetsHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterPlanets
	t.Cleanup(func() { listCharacterPlanets = orig })
	listCharacterPlanets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Planet, error) {
		return []Planet{{PlanetID: 40009086, SolarSystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := listCharacterPlanetsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterPlanetsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Planets[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Planets[0].SolarSystemName, "Jita")
	}
	if out.Planets[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Planets[0].PlanetName, "Jita IV")
	}
}

func TestListCorporationCustomsOfficesHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]CustomsOffice, error) {
		return []CustomsOffice{{OfficeID: 1, SystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Offices[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Offices[0].SystemName, "Jita")
	}
}
```

Confirm the exact output wrapper field names (`Planets`/`Offices`) against the file before
finalizing.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestListCharacterPlanetsHandler_ResolvesLocationNames|TestListCorporationCustomsOfficesHandler_ResolvesSystemName" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `listCharacterPlanetsHandler`'s signature to add `sdeStore *sde.Store`, and after the
planets slice is built, insert:
```go
		systemIDs := make([]int64, len(planets))
		planetIDs := make([]int64, len(planets))
		for i, p := range planets {
			systemIDs[i] = p.SolarSystemID
			planetIDs[i] = p.PlanetID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		planetNames := resolvePlanetNames(ctx, sdeStore, sso.UserAgent(version), planetIDs)
		for i := range planets {
			planets[i].SolarSystemName = systemNames[planets[i].SolarSystemID]
			planets[i].PlanetName = planetNames[planets[i].PlanetID]
		}
```

`listCorporationCustomsOfficesHandler` already has `sdeStore`. After its existing `TypeName`
enrichment loop, add the standard flat-slice `resolveSystemNames` shape on `SystemID`/`SystemName`.

In `mcp/server.go`, update the `list_character_planets` call site: `listCharacterPlanetsHandler(version, clientID, store, sdeStore)`. `list_corporation_customs_offices` needs no change.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestListCharacterPlanetsHandler_ResolvesLocationNames|TestListCorporationCustomsOfficesHandler_ResolvesSystemName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/planetary_interaction.go mcp/planetary_interaction_internal_test.go mcp/server.go
git commit -m "feat: resolve Planet/CustomsOffice location names via sde/"
```

---

### Task 12: `mcp/routes.go` — `GetRouteOutput`

**Files:**
- Modify: `mcp/routes.go`
- Modify: `mcp/server.go` (`get_route` call site)
- Test: `mcp/routes_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type GetRouteOutput struct {
	OriginSystemID        int64    `json:"originSystemId"`
	OriginSystemName      string   `json:"originSystemName,omitempty"`
	DestinationSystemID   int64    `json:"destinationSystemId"`
	DestinationSystemName string   `json:"destinationSystemName,omitempty"`
	Route                 []int64  `json:"route"`
	RouteNames            []string `json:"routeNames,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/routes_internal_test.go
func TestGetRouteHandler_ResolvesLocationNames(t *testing.T) {
	orig := getRoute
	t.Cleanup(func() { getRoute = orig })
	getRoute = func(ctx context.Context, userAgent string, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
		return []int64{30000142, 30000144, 30002187}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter", 30002187: "Amarr"}
	}

	handler := getRouteHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 30000142, DestinationSystemID: 30002187})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.OriginSystemName != "Jita" {
		t.Errorf("OriginSystemName = %q, want %q", out.OriginSystemName, "Jita")
	}
	if out.DestinationSystemName != "Amarr" {
		t.Errorf("DestinationSystemName = %q, want %q", out.DestinationSystemName, "Amarr")
	}
	want := []string{"Jita", "Perimeter", "Amarr"}
	if len(out.RouteNames) != len(want) {
		t.Fatalf("RouteNames = %v, want %v", out.RouteNames, want)
	}
	for i, w := range want {
		if out.RouteNames[i] != w {
			t.Errorf("RouteNames[%d] = %q, want %q", i, out.RouteNames[i], w)
		}
	}
}
```

Confirm `getRoute`'s exact current signature against the file before finalizing (the audit shows
the call site but the var's own declared parameter list should be double-checked).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetRouteHandler_ResolvesLocationNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `getRouteHandler`'s signature to `getRouteHandler(version string, sdeStore *sde.Store)`,
and after `route, err := getRoute(...)` succeeds, insert:
```go
		names := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), route)
		routeNames := make([]string, len(route))
		for i, id := range route {
			routeNames[i] = names[id]
		}
```
then change the final return to:
```go
		return nil, GetRouteOutput{
			OriginSystemID: in.OriginSystemID, OriginSystemName: names[in.OriginSystemID],
			DestinationSystemID: in.DestinationSystemID, DestinationSystemName: names[in.DestinationSystemID],
			Route: route, RouteNames: routeNames,
		}, nil
```
(`names` already covers `in.OriginSystemID`/`in.DestinationSystemID` as long as `route` includes
the origin and destination systems as its first/last elements, which ESI's route response always
does — no separate lookup call needed for those two fields.)

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/routes.go`'s imports.

In `mcp/server.go`, update the `get_route` call site: `getRouteHandler(version, sdeStore)`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetRouteHandler_ResolvesLocationNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/routes.go mcp/routes_internal_test.go mcp/server.go
git commit -m "feat: resolve GetRouteOutput location names via sde/"
```

---

### Task 13: `mcp/sovereignty.go` — `SovereigntyCampaign`, `SovereigntySystem`

**Files:**
- Modify: `mcp/sovereignty.go`
- Modify: `mcp/server.go` (`list_sovereignty_campaigns`, `list_sovereignty_systems` call sites)
- Test: `mcp/sovereignty_internal_test.go`

**Interfaces:**
- Consumes: `resolveSystemNames`, `resolveConstellationNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type SovereigntyCampaign struct {
	CampaignID        int64                    `json:"campaignId"`
	ConstellationID   int64                    `json:"constellationId"`
	ConstellationName string                   `json:"constellationName,omitempty"`
	SolarSystemID     int64                    `json:"solarSystemId"`
	SolarSystemName   string                   `json:"solarSystemName,omitempty"`
	StructureID       int64                    `json:"structureId"`
	EventType         string                   `json:"eventType"`
	StartTime         time.Time                `json:"startTime"`
	AttackersScore    *float64                 `json:"attackersScore,omitempty"`
	DefenderID        *int64                   `json:"defenderId,omitempty"`
	DefenderScore     *float64                 `json:"defenderScore,omitempty"`
	Participants      []SovereigntyParticipant `json:"participants,omitempty"`
}
```
```go
type SovereigntySystem struct {
	SolarSystemID   int64            `json:"solarSystemId"`
	SolarSystemName string           `json:"solarSystemName,omitempty"`
	Claim           SovereigntyClaim `json:"claim"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/sovereignty_internal_test.go
func TestListSovereigntyCampaignsHandler_ResolvesLocationNames(t *testing.T) {
	orig := listSovereigntyCampaigns
	t.Cleanup(func() { listSovereigntyCampaigns = orig })
	listSovereigntyCampaigns = func(ctx context.Context, userAgent string) ([]SovereigntyCampaign, error) {
		return []SovereigntyCampaign{{CampaignID: 1, ConstellationID: 20000020, SolarSystemID: 30000142}}, nil
	}

	origConstellation := resolveConstellationNames
	t.Cleanup(func() { resolveConstellationNames = origConstellation })
	resolveConstellationNames = func(ctx context.Context, sdeStore constellationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{20000020: "Kimotoro"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listSovereigntyCampaignsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListSovereigntyCampaignsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Campaigns[0].ConstellationName != "Kimotoro" {
		t.Errorf("ConstellationName = %q, want %q", out.Campaigns[0].ConstellationName, "Kimotoro")
	}
	if out.Campaigns[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Campaigns[0].SolarSystemName, "Jita")
	}
}

func TestListSovereigntySystemsHandler_ResolvesSystemName(t *testing.T) {
	orig := listSovereigntySystems
	t.Cleanup(func() { listSovereigntySystems = orig })
	listSovereigntySystems = func(ctx context.Context, userAgent string) ([]SovereigntySystem, error) {
		return []SovereigntySystem{{SolarSystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listSovereigntySystemsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListSovereigntySystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Systems[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Systems[0].SolarSystemName, "Jita")
	}
}
```

Confirm the exact output wrapper field names (`Campaigns`/`Systems`) against the file before
finalizing.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestListSovereigntyCampaignsHandler_ResolvesLocationNames|TestListSovereigntySystemsHandler_ResolvesSystemName" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Change `listSovereigntyCampaignsHandler`'s signature to `listSovereigntyCampaignsHandler(version
string, sdeStore *sde.Store)`, and after the campaigns slice is built, insert:
```go
		constellationIDs := make([]int64, len(campaigns))
		systemIDs := make([]int64, len(campaigns))
		for i, c := range campaigns {
			constellationIDs[i] = c.ConstellationID
			systemIDs[i] = c.SolarSystemID
		}
		constellationNames := resolveConstellationNames(ctx, sdeStore, sso.UserAgent(version), constellationIDs)
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		for i := range campaigns {
			campaigns[i].ConstellationName = constellationNames[campaigns[i].ConstellationID]
			campaigns[i].SolarSystemName = systemNames[campaigns[i].SolarSystemID]
		}
```

Change `listSovereigntySystemsHandler`'s signature to `listSovereigntySystemsHandler(version
string, sdeStore *sde.Store)`, and add the standard flat-slice `resolveSystemNames` shape on
`SolarSystemID`/`SolarSystemName`.

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/sovereignty.go`'s imports.

In `mcp/server.go`, update both call sites: `listSovereigntyCampaignsHandler(version, sdeStore)`,
`listSovereigntySystemsHandler(version, sdeStore)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestListSovereigntyCampaignsHandler_ResolvesLocationNames|TestListSovereigntySystemsHandler_ResolvesSystemName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/sovereignty.go mcp/sovereignty_internal_test.go mcp/server.go
git commit -m "feat: resolve sovereignty campaign/system location names via sde/"
```

---

### Task 14: `mcp/structures.go` — six handlers, deepest nesting in this batch

**Files:**
- Modify: `mcp/structures.go`
- Modify: `mcp/server.go` (three new call sites: `list_corporation_skyhooks`,
  `list_character_mercenary_dens`, `list_corporation_sovereignty_hubs`; three already pass
  `sdeStore`)
- Test: `mcp/structures_internal_test.go`

**Interfaces:**
- Consumes: `resolvePlanetNames`, `resolveSystemNames` (Task 1).

- [ ] **Step 1: Add the fields**

```go
type Skyhook struct {
	ID         int64  `json:"id"`
	PlanetID   int64  `json:"planetId"`
	PlanetName string `json:"planetName,omitempty"`
}
```
```go
type SkyhookDetail struct {
	EffectiveWorkforce *int64           `json:"effectiveWorkforce,omitempty"`
	ID                 int64            `json:"id"`
	PlanetID           int64            `json:"planetId"`
	PlanetName         string           `json:"planetName,omitempty"`
	IsActive           bool             `json:"isActive"`
	Reagents           []SkyhookReagent `json:"reagents,omitempty"`
	// ... (remaining pre-existing fields unchanged)
}
```
```go
type MercenaryDen struct {
	ID         int64  `json:"id"`
	PlanetID   int64  `json:"planetId"`
	PlanetName string `json:"planetName,omitempty"`
}
```
```go
type MercenaryDenSkyhookRef struct {
	CorporationID int64  `json:"corporationId"`
	ID            int64  `json:"id"`
	PlanetID      int64  `json:"planetId"`
	PlanetName    string `json:"planetName,omitempty"`
}
```
```go
type StructureSovereigntyHub struct {
	ID              int64  `json:"id"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
}
```
```go
type StructureSovereigntyHubTransportConfig struct {
	Kind                    string   `json:"kind"`
	ExportAmount            *int64   `json:"exportAmount,omitempty"`
	ExportSolarSystemID     *int64   `json:"exportSolarSystemId,omitempty"`
	ExportSolarSystemName   string   `json:"exportSolarSystemName,omitempty"`
	ImportSolarSystemIDs    []int64  `json:"importSolarSystemIds,omitempty"`
	ImportSolarSystemNames  []string `json:"importSolarSystemNames,omitempty"`
	Transit                 *bool    `json:"transit,omitempty"`
}
```
```go
type StructureSovereigntyHubTransportImportSource struct {
	Amount          int64  `json:"amount"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
}
```
```go
type StructureSovereigntyHubTransportState struct {
	Kind                  string                                         `json:"kind"`
	ExportAmount          *int64                                         `json:"exportAmount,omitempty"`
	ExportSolarSystemID   *int64                                         `json:"exportSolarSystemId,omitempty"`
	ExportSolarSystemName string                                         `json:"exportSolarSystemName,omitempty"`
	ImportSources         []StructureSovereigntyHubTransportImportSource `json:"importSources,omitempty"`
	Transit               *bool                                          `json:"transit,omitempty"`
}
```
`StructureSovereigntyHubDetail.SolarSystemID` also gains a `SolarSystemName string
json:"solarSystemName,omitempty"` sibling (added at the top level, alongside the existing field —
this struct's other fields are unchanged).

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/structures_internal_test.go
func TestListCorporationSkyhooksHandler_ResolvesPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationSkyhooks
	t.Cleanup(func() { listCorporationSkyhooks = orig })
	listCorporationSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Skyhook, error) {
		return []Skyhook{{ID: 1, PlanetID: 40009086}}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := listCorporationSkyhooksHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationSkyhooksInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Skyhooks[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Skyhooks[0].PlanetName, "Jita IV")
	}
}

func TestGetCorporationSkyhookHandler_ResolvesPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationSkyhook
	t.Cleanup(func() { getCorporationSkyhook = orig })
	getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
		return SkyhookDetail{ID: 1, PlanetID: 40009086}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := getCorporationSkyhookHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSkyhookInput{CharacterID: 7, CorporationID: 98000001, SkyhookID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.PlanetName, "Jita IV")
	}
}

func TestListCharacterMercenaryDensHandler_ResolvesPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMercenaryDens
	t.Cleanup(func() { listCharacterMercenaryDens = orig })
	listCharacterMercenaryDens = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MercenaryDen, error) {
		return []MercenaryDen{{ID: 1, PlanetID: 40009086}}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := listCharacterMercenaryDensHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMercenaryDensInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Dens[0].PlanetName != "Jita IV" {
		t.Errorf("PlanetName = %q, want %q", out.Dens[0].PlanetName, "Jita IV")
	}
}

func TestGetCharacterMercenaryDenHandler_ResolvesSkyhookPlanetName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterMercenaryDen
	t.Cleanup(func() { getCharacterMercenaryDen = orig })
	getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, denID int64) (MercenaryDenDetail, error) {
		return MercenaryDenDetail{ID: 1, Skyhook: MercenaryDenSkyhookRef{ID: 2, PlanetID: 40009086}}, nil
	}

	origPlanet := resolvePlanetNames
	t.Cleanup(func() { resolvePlanetNames = origPlanet })
	resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009086: "Jita IV"}
	}

	handler := getCharacterMercenaryDenHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterMercenaryDenInput{CharacterID: 7, DenID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Skyhook.PlanetName != "Jita IV" {
		t.Errorf("Skyhook.PlanetName = %q, want %q", out.Skyhook.PlanetName, "Jita IV")
	}
}

func TestListCorporationSovereigntyHubsHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationSovereigntyHubs
	t.Cleanup(func() { listCorporationSovereigntyHubs = orig })
	listCorporationSovereigntyHubs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]StructureSovereigntyHub, error) {
		return []StructureSovereigntyHub{{ID: 1, SolarSystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationSovereigntyHubsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationSovereigntyHubsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Hubs[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Hubs[0].SolarSystemName, "Jita")
	}
}

func TestGetCorporationSovereigntyHubHandler_ResolvesTransportSystemNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationSovereigntyHub
	t.Cleanup(func() { getCorporationSovereigntyHub = orig })
	exportID := int64(30000144)
	getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
		return StructureSovereigntyHubDetail{
			ID: 1, SolarSystemID: 30000142,
			WorkforceTransport: StructureSovereigntyHubTransport{
				Configuration: StructureSovereigntyHubTransportConfig{Kind: "export", ExportSolarSystemID: &exportID},
				State: StructureSovereigntyHubTransportState{
					Kind: "import",
					ImportSources: []StructureSovereigntyHubTransportImportSource{{Amount: 100, SolarSystemID: 30002187}},
				},
			},
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter", 30002187: "Amarr"}
	}

	handler := getCorporationSovereigntyHubHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationSovereigntyHubInput{CharacterID: 7, CorporationID: 98000001, SovereigntyHubID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.SolarSystemName, "Jita")
	}
	if out.WorkforceTransport.Configuration.ExportSolarSystemName != "Perimeter" {
		t.Errorf("WorkforceTransport.Configuration.ExportSolarSystemName = %q, want %q", out.WorkforceTransport.Configuration.ExportSolarSystemName, "Perimeter")
	}
	if out.WorkforceTransport.State.ImportSources[0].SolarSystemName != "Amarr" {
		t.Errorf("WorkforceTransport.State.ImportSources[0].SolarSystemName = %q, want %q", out.WorkforceTransport.State.ImportSources[0].SolarSystemName, "Amarr")
	}
}
```

Confirm the exact output wrapper field names (`Skyhooks`/`Dens`/`Hubs`) against the file before
finalizing, and confirm `getCorporationSovereigntyHub`'s real overridable-var name/signature.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "ResolvesPlanetName|ResolvesSkyhookPlanetName|ResolvesSystemName|ResolvesTransportSystemNames" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment in all six handlers**

- `listCorporationSkyhooksHandler`: add `sdeStore *sde.Store`; standard flat-slice
  `resolvePlanetNames` shape on `PlanetID`/`PlanetName`.
- `getCorporationSkyhookHandler`: already has `sdeStore` (batch 1). After the existing `Reagents`
  enrichment loop, add a single-scalar `resolvePlanetNames` call on `out.PlanetID`.
- `listCharacterMercenaryDensHandler`: add `sdeStore *sde.Store`; standard flat-slice shape on
  `PlanetID`/`PlanetName`.
- `getCharacterMercenaryDenHandler`: already has `sdeStore` (batch 1, den's own `TypeName`). After
  the existing single-scalar `TypeName` lookup, add a single-scalar `resolvePlanetNames` call on
  `detail.Skyhook.PlanetID`, assigning to `detail.Skyhook.PlanetName`.
- `listCorporationSovereigntyHubsHandler`: add `sdeStore *sde.Store`; standard flat-slice
  `resolveSystemNames` shape on `SolarSystemID`/`SolarSystemName`.
- `getCorporationSovereigntyHubHandler`: already has `sdeStore` (batch 1, reagent/upgrade
  `TypeName`). After the existing enrichment block, add:
```go
		ids := []int64{out.SolarSystemID}
		if out.WorkforceTransport.Configuration.ExportSolarSystemID != nil {
			ids = append(ids, *out.WorkforceTransport.Configuration.ExportSolarSystemID)
		}
		ids = append(ids, out.WorkforceTransport.Configuration.ImportSolarSystemIDs...)
		if out.WorkforceTransport.State.ExportSolarSystemID != nil {
			ids = append(ids, *out.WorkforceTransport.State.ExportSolarSystemID)
		}
		for _, src := range out.WorkforceTransport.State.ImportSources {
			ids = append(ids, src.SolarSystemID)
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), ids)
		out.SolarSystemName = systemNames[out.SolarSystemID]
		if out.WorkforceTransport.Configuration.ExportSolarSystemID != nil {
			out.WorkforceTransport.Configuration.ExportSolarSystemName = systemNames[*out.WorkforceTransport.Configuration.ExportSolarSystemID]
		}
		if len(out.WorkforceTransport.Configuration.ImportSolarSystemIDs) > 0 {
			importNames := make([]string, len(out.WorkforceTransport.Configuration.ImportSolarSystemIDs))
			for i, id := range out.WorkforceTransport.Configuration.ImportSolarSystemIDs {
				importNames[i] = systemNames[id]
			}
			out.WorkforceTransport.Configuration.ImportSolarSystemNames = importNames
		}
		if out.WorkforceTransport.State.ExportSolarSystemID != nil {
			out.WorkforceTransport.State.ExportSolarSystemName = systemNames[*out.WorkforceTransport.State.ExportSolarSystemID]
		}
		for i := range out.WorkforceTransport.State.ImportSources {
			out.WorkforceTransport.State.ImportSources[i].SolarSystemName = systemNames[out.WorkforceTransport.State.ImportSources[i].SolarSystemID]
		}
```
  (one combined `resolveSystemNames` call covering all five possible sites: the hub's own system,
  the two oneOf-style transport-config/state export IDs, the config's import list, and the state's
  import-sources list — matches the design's stated single-combined-call approach.)

In `mcp/server.go`, update the three new call sites (`list_corporation_skyhooks`,
`list_character_mercenary_dens`, `list_corporation_sovereignty_hubs`) to pass `sdeStore`. The other
three already do.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "ResolvesPlanetName|ResolvesSkyhookPlanetName|ResolvesSystemName|ResolvesTransportSystemNames" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/structures.go mcp/structures_internal_test.go mcp/server.go
git commit -m "feat: resolve structures-tag location names via sde/"
```

---

### Task 15: `mcp/industry.go` — five handlers, shared-helper split

**Files:**
- Modify: `mcp/industry.go`
- Modify: `mcp/server.go` (`list_industry_systems`, `list_corporation_mining_extractions` call
  sites — new; three already pass `sdeStore`)
- Test: `mcp/industry_internal_test.go`

**Interfaces:**
- Consumes: `resolveRegionNames`, `resolveSystemNames`, `resolveMoonNames`, `resolveStationNames`
  (Task 1).

- [ ] **Step 1: Add the fields**

```go
type IndustryFacility struct {
	FacilityID      int64    `json:"facilityId"`
	OwnerID         int64    `json:"ownerId"`
	RegionID        int64    `json:"regionId"`
	RegionName      string   `json:"regionName,omitempty"`
	SolarSystemID   int64    `json:"solarSystemId"`
	SolarSystemName string   `json:"solarSystemName,omitempty"`
	TypeID          int64    `json:"typeId"`
	TypeName        string   `json:"typeName,omitempty"`
	Tax             *float64 `json:"tax,omitempty"`
}
```
```go
type IndustrySystem struct {
	SolarSystemID   int64       `json:"solarSystemId"`
	SolarSystemName string      `json:"solarSystemName,omitempty"`
	CostIndices     []CostIndex `json:"costIndices"`
}
```
```go
type MiningLedgerEntry struct {
	Date            string `json:"date"`
	Quantity        int64  `json:"quantity"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
	TypeID          int64  `json:"typeId"`
	TypeName        string `json:"typeName,omitempty"`
}
```
```go
type MiningExtraction struct {
	MoonID              int64     `json:"moonId"`
	MoonName            string    `json:"moonName,omitempty"`
	StructureID         int64     `json:"structureId"`
	ChunkArrivalTime    time.Time `json:"chunkArrivalTime"`
	ExtractionStartTime time.Time `json:"extractionStartTime"`
	NaturalDecayTime    time.Time `json:"naturalDecayTime"`
}
```
`IndustryJob` gains `StationName string json:"stationName,omitempty"` immediately after the
existing `StationID int64 json:"stationId,omitempty"` field (no other change to that struct's field
list).

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/industry_internal_test.go
func TestListIndustryFacilitiesHandler_ResolvesLocationNames(t *testing.T) {
	orig := listIndustryFacilities
	t.Cleanup(func() { listIndustryFacilities = orig })
	listIndustryFacilities = func(ctx context.Context, userAgent string) ([]IndustryFacility, error) {
		return []IndustryFacility{{FacilityID: 1, RegionID: 10000002, SolarSystemID: 30000142}}, nil
	}

	origRegion := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origRegion })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listIndustryFacilitiesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIndustryFacilitiesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Facilities[0].RegionName != "The Forge" {
		t.Errorf("RegionName = %q, want %q", out.Facilities[0].RegionName, "The Forge")
	}
	if out.Facilities[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Facilities[0].SolarSystemName, "Jita")
	}
}

func TestListIndustrySystemsHandler_ResolvesSystemName(t *testing.T) {
	orig := listIndustrySystems
	t.Cleanup(func() { listIndustrySystems = orig })
	listIndustrySystems = func(ctx context.Context, userAgent string) ([]IndustrySystem, error) {
		return []IndustrySystem{{SolarSystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listIndustrySystemsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIndustrySystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Systems[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Systems[0].SolarSystemName, "Jita")
	}
}

func TestListCharacterMiningHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMining
	t.Cleanup(func() { listCharacterMining = orig })
	listCharacterMining = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MiningLedgerEntry, error) {
		return []MiningLedgerEntry{{Date: "2026-07-30", SolarSystemID: 30000142, TypeID: 34}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCharacterMiningHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMiningInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Ledger[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Ledger[0].SolarSystemName, "Jita")
	}
}

func TestListCorporationMiningExtractionsHandler_ResolvesMoonName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMiningExtractions
	t.Cleanup(func() { listCorporationMiningExtractions = orig })
	listCorporationMiningExtractions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MiningExtraction, error) {
		return []MiningExtraction{{MoonID: 40009087, StructureID: 1}}, nil
	}

	origMoon := resolveMoonNames
	t.Cleanup(func() { resolveMoonNames = origMoon })
	resolveMoonNames = func(ctx context.Context, sdeStore moonNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009087: "Jita IV - Moon 4"}
	}

	handler := listCorporationMiningExtractionsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMiningExtractionsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Extractions[0].MoonName != "Jita IV - Moon 4" {
		t.Errorf("MoonName = %q, want %q", out.Extractions[0].MoonName, "Jita IV - Moon 4")
	}
}

func TestListCharacterIndustryJobsHandler_ResolvesStationName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterIndustryJobs
	t.Cleanup(func() { listCharacterIndustryJobs = orig })
	listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]IndustryJob, error) {
		return []IndustryJob{{JobID: 1, BlueprintTypeID: 587, StationID: 60003760}}, nil
	}

	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := listCharacterIndustryJobsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterIndustryJobsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Jobs[0].StationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("StationName = %q, want %q", out.Jobs[0].StationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}

func TestListCorporationIndustryJobsHandler_StationNameEmptyWhenStationIDZero(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationIndustryJobs
	t.Cleanup(func() { listCorporationIndustryJobs = orig })
	listCorporationIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]IndustryJob, error) {
		return []IndustryJob{{JobID: 1, BlueprintTypeID: 587}}, nil // StationID left zero, matching the real corp mapper
	}

	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{0: "should never be assigned"}
	}

	handler := listCorporationIndustryJobsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationIndustryJobsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Jobs[0].StationName != "" {
		t.Errorf("StationName = %q, want empty (StationID is 0 for corp-scoped jobs)", out.Jobs[0].StationName)
	}
}
```

Confirm the exact output wrapper field names (`Facilities`/`Systems`/`Ledger`/`Extractions`/`Jobs`)
against the file before finalizing.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "ResolvesLocationNames|ResolvesSystemName|ResolvesMoonName|ResolvesStationName|StationNameEmptyWhenStationIDZero" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

`listIndustryFacilitiesHandler`: already has `sdeStore`. After the existing `TypeName` loop, add
region + system resolution:
```go
		regionIDs := make([]int64, len(facilities))
		systemIDs := make([]int64, len(facilities))
		for i, f := range facilities {
			regionIDs[i] = f.RegionID
			systemIDs[i] = f.SolarSystemID
		}
		regionNames := resolveRegionNames(ctx, sdeStore, sso.UserAgent(version), regionIDs)
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		for i := range facilities {
			facilities[i].RegionName = regionNames[facilities[i].RegionID]
			facilities[i].SolarSystemName = systemNames[facilities[i].SolarSystemID]
		}
```

`listIndustrySystemsHandler`: add `sdeStore *sde.Store`; standard flat-slice `resolveSystemNames`
shape on `SolarSystemID`/`SolarSystemName`.

`listCharacterMiningHandler`: already has `sdeStore`. After the existing `TypeName` loop, add the
standard flat-slice `resolveSystemNames` shape on `SolarSystemID`/`SolarSystemName`.

`listCorporationMiningExtractionsHandler`: add `sdeStore *sde.Store` as a 4th parameter; standard
flat-slice `resolveMoonNames` shape on `MoonID`/`MoonName`.

`IndustryJob.StationID`/`StationName`: **do not modify the shared `enrichIndustryJobs` helper** —
`StationID` is only ever set by the character-scoped mapper, so add a SEPARATE, small enrichment
step directly inside `listCharacterIndustryJobsHandler` (not inside `enrichIndustryJobs`, and not
duplicated into `listCorporationIndustryJobsHandler`), right after the existing
`enrichIndustryJobs(ctx, jobs, sdeStore, sso.UserAgent(version))` call:
```go
		stationIDs := make([]int64, 0, len(jobs))
		for _, j := range jobs {
			if j.StationID != 0 {
				stationIDs = append(stationIDs, j.StationID)
			}
		}
		stationNames := resolveStationNames(ctx, sdeStore, sso.UserAgent(version), stationIDs)
		for i := range jobs {
			if jobs[i].StationID != 0 {
				jobs[i].StationName = stationNames[jobs[i].StationID]
			}
		}
```
`listCorporationIndustryJobsHandler` needs no change at all — its jobs always have `StationID ==
0`, so `StationName` naturally never gets set there, which is correct (verified this exact
"skip when 0" behavior would also work with no guard at all, per the Global Constraints note — the
explicit `!= 0` check here is just an optimization to skip a pointless `resolveStationNames` call
on an all-zero slice for the corp-scoped case, not a correctness requirement).

In `mcp/server.go`, update `list_industry_systems` and `list_corporation_mining_extractions` to
pass `sdeStore`. The other three call sites already do.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "ResolvesLocationNames|ResolvesSystemName|ResolvesMoonName|ResolvesStationName|StationNameEmptyWhenStationIDZero" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/industry.go mcp/industry_internal_test.go mcp/server.go
git commit -m "feat: resolve industry-tag location names via sde/"
```

---

### Task 16: Docs sync + VERSION bump

**Files:**
- Modify: `docs/reference/esi-coverage-roadmap.md`
- Modify: `CLAUDE.md` (package table's `mcp/` row)
- Modify: `README.md` (matching `mcp/` row)
- Modify: `VERSION`, `claude_resources/neat/.claude-plugin/plugin.json`,
  `claude_resources/neat/CHANGELOG.md`

**Interfaces:** none — pure docs/version-bump task, run last once every prior task has landed.

- [ ] **Step 1: Update the roadmap doc**

In `docs/reference/esi-coverage-roadmap.md`'s "Conventions" section, extend the sentence batch 1
added about `type_id` enrichment with a note that location IDs are now also covered:
```
- Every `esi/` method returns exactly what ESI returns, including bare IDs, with zero
  name-resolution enrichment. `type_id`-like fields and the six location-entity ID kinds
  (system/region/constellation/station/planet/moon) across the non-Universe tags are enriched
  at the `mcp/` layer via `mcp/enrich.go`'s resolver functions (see
  `docs/superpowers/specs/2026-07-28-sde-type-id-enrichment-design.md` and
  `docs/superpowers/specs/2026-07-30-sde-location-id-enrichment-design.md`) — the generic
  ambiguous `location_id` fields and the rarer race/bloodline/faction/group IDs are still open
  follow-up batches.
```

- [ ] **Step 2: Update `CLAUDE.md`'s and `README.md`'s `mcp/` package-table rows**

Both files have an identical `mcp/enrich.go` clause (added by batch 1's Task 20). Extend it in both
files with the same added sentence, e.g.: "Batch 2 extended `mcp/enrich.go` with six sibling
resolvers (`resolveSystemNames`/`resolveRegionNames`/`resolveConstellationNames`/
`resolveStationNames`/`resolvePlanetNames`/`resolveMoonNames`) covering the same additive-`*Name`
pattern for system/region/constellation/station/planet/moon IDs across 14 more tools."

- [ ] **Step 3: Bump VERSION and regenerate the plugin manifest**

```bash
current=$(cat VERSION)  # 0.2.0
echo "0.3.0" > VERSION  # minor bump -- another additive feature, not a bugfix
go generate ./...
```

Add a changelog entry to `claude_resources/neat/CHANGELOG.md`:
```markdown
## 0.3.0
- Adds SDE-resolved location `*Name` fields (`SystemName`, `RegionName`, `ConstellationName`,
  `StationName`, `PlanetName`, `MoonName`, plus the index-aligned `*Names` list variants) alongside
  the existing bare `*ID` fields across 14 non-Universe MCP tools (corporation, industry,
  killmails, market, planetary interaction, structures, sovereignty, fleets, and more). Names
  resolve from the local SDE cache only, same as batch 1's `type_id` enrichment — no ESI calls are
  made on this path. Purely additive.
```

- [ ] **Step 4: Verify everything is clean**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race && actionlint`
Expected: all clean.

- [ ] **Step 5: Commit**

```bash
git add docs/reference/esi-coverage-roadmap.md CLAUDE.md README.md VERSION claude_resources/neat/.claude-plugin/plugin.json claude_resources/neat/CHANGELOG.md
git commit -m "docs: document location-ID enrichment in the roadmap and package table, bump VERSION"
```
