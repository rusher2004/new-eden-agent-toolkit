# SDE-backed race/bloodline/faction/group-ID enrichment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add resolved `*Name` sibling fields for the four remaining flagged ID kinds —
`race_id`, `bloodline_id`, `faction_id`, `group_id` — closing out batch 1's originally-flagged
follow-up list. 21 occurrences across 8 non-Universe `mcp/*.go` files, plus three new single-ID
`sde.Store` accessors that don't exist yet.

**Architecture:** Add `Store.Race`/`Store.Bloodline`/`Store.Faction` single-ID accessors to
`sde/`, mirroring `sde/group.go`'s existing `Group()` method exactly (`Store.Group` already
exists, no `sde/` work needed for it). Add four new resolvers to `mcp/enrich.go`
(`resolveRaceNames`/`resolveBloodlineNames`/`resolveFactionNames`/`resolveGroupNames`), each a
package-level `var` mirroring `resolveTypeNames`'s contract. Wire all four into the 8 affected
files, following the exact additive `*Name`-sibling pattern batches 1-2 established.

**Tech Stack:** Go, `github.com/modelcontextprotocol/go-sdk/mcp`, `go.etcd.io/bbolt` (via `sde/`),
standard library `testing` only.

## Global Constraints

- Every new field is additive (`FooID`/`FooIDs` gains a sibling `FooName string`,
  `json:"...,omitempty"`) — no existing field is removed or changes type.
- Handlers take the concrete `sdeStore *sde.Store` parameter (not an interface) — the established
  convention. Every resolver is a package-level `var`, not a plain `func`, so tests can override it
  directly; tests pass `nil` for `*sde.Store` (safe — every `sde.Store` accessor has an explicit
  nil-receiver guard as its first statement) and override the relevant resolver var when they need
  an actual resolved name.
- A lookup miss or error is silently omitted (empty `*Name`), never surfaced as a tool error — the
  established `resolveTypeNames` contract, unchanged.
- No live ESI fallback in any of the four new resolvers, or the three new `sde.Store` accessors —
  same rate-limit-safety reasoning as prior batches.
- All four entities (`Race`, `Bloodline`, `Faction`, `Group`) have `Name map[string]string`
  (localized) — read via `.Name["en"]`. No plain-string asymmetry in this batch.
- Mixed `int64`/`*int64` typing per field must be preserved exactly as documented per-task below —
  do not assume uniformity across occurrences of the same ID kind.
- `sde.Group` (the item/ship group tier, e.g. "Frigate") is unrelated to ESI's *market group*
  concept (`MarketGroupID`/`ParentGroupID` in `market.go`/`sde/type.go`) — this batch does not touch
  market-group fields.
- `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -race` must stay clean.
- Wrap errors with context per this repo's convention (`fmt.Errorf("operation: %w", err)`); never
  discard an error.

---

### Task 1: `sde/` — three new single-ID accessors

**Files:**
- Modify: `sde/race.go`, `sde/bloodline.go`, `sde/faction.go` (add `Race`/`Bloodline`/`Faction`
  single-ID accessors — `Store.Group` in `sde/group.go` already exists, untouched)
- Test: `sde/race_test.go`, `sde/bloodline_test.go`, `sde/faction_test.go` (add tests to the
  existing files, alongside their current `TestRaces_*`/`TestBloodlines_*`/`TestFactions_*` tests)

**Interfaces:**
- Produces: `func (s *Store) Race(ctx context.Context, userAgent string, id int64) (Race, bool, error)`,
  `func (s *Store) Bloodline(ctx context.Context, userAgent string, id int64) (Bloodline, bool, error)`,
  `func (s *Store) Faction(ctx context.Context, userAgent string, id int64) (Faction, bool, error)` —
  every later task's `mcp/enrich.go` resolver interface calls these by name.

- [ ] **Step 1: Write the failing tests**

Add to `sde/race_test.go` (mirroring `sde/group_test.go`'s `TestGroup_KnownID_ReturnsAllLocales`/
`_UnknownID_ReturnsNotOK`/`_MalformedCachedRecord_ReturnsError`/`_MismatchedKey_ReturnsError`):

```go
func TestRace_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 1, "description": {"en": "..."}, "name": {"en": "Caldari", "de": "Kaldari"}}`)

	got, ok, err := store.Race(context.Background(), "test-agent", 1)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if !ok {
		t.Fatal("Race ok = false, want true")
	}
	if got.ID != 1 {
		t.Errorf("got.ID = %d, want 1", got.ID)
	}
	if got.Name["en"] != "Caldari" || got.Name["de"] != "Kaldari" {
		t.Errorf("Name = %+v, want en=Caldari de=Kaldari", got.Name)
	}
}

func TestRace_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Race(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if ok {
		t.Error("Race ok = true, want false for an unknown ID")
	}
}

func TestRace_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `not valid json`)

	_, _, err := store.Race(context.Background(), "test-agent", 1)
	if err == nil {
		t.Fatal("Race returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestRace_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 999, "description": {"en": "..."}, "name": {"en": "Caldari"}}`)

	_, _, err := store.Race(context.Background(), "test-agent", 1)
	if err == nil {
		t.Fatal("Race returned nil error, want an error for a mismatched _key")
	}
}
```

Add the identical four-test shape to `sde/bloodline_test.go` (bucket name `"bloodlines"`, sample
record `{"_key": 1, "charisma": 6, "corporationID": 1000006, "description": {"en": "..."}, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "shipTypeID": 601, "willpower": 5}`,
assertions on `got.ID`, `got.Name["en"]`, `got.RaceID`) and to `sde/faction_test.go` (bucket name
`"factions"`, sample record `{"_key": 500001, "corporationID": 1000035, "description": {"en": "..."}, "militiaCorporationID": 1000180, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "solarSystemID": 30000145, "uniqueName": true, "stationCount": 120, "stationSystemCount": 45}`,
assertions on `got.ID`, `got.Name["en"]`, `got.IsUnique`), replacing `Group`/`groups`/`TestGroup_*`
with `Bloodline`/`bloodlines`/`TestBloodline_*` and `Faction`/`factions`/`TestFaction_*`
respectively.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sde/... -run "TestRace_KnownID|TestRace_UnknownID|TestRace_MalformedCachedRecord|TestRace_MismatchedKey|TestBloodline_KnownID|TestBloodline_UnknownID|TestBloodline_MalformedCachedRecord|TestBloodline_MismatchedKey|TestFaction_KnownID|TestFaction_UnknownID|TestFaction_MalformedCachedRecord|TestFaction_MismatchedKey" -v`
Expected: FAIL — none of the three new accessors exist yet.

- [ ] **Step 3: Implement the three accessors**

Add to `sde/race.go`, below the existing `Races` method:

```go
// Race looks up a race by ID. ok is false if the ID isn't in the cache.
func (s *Store) Race(ctx context.Context, userAgent string, id int64) (Race, bool, error) {
	if s == nil {
		return Race{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Race
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("races"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw raceRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached race %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached race %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Race{ID: raw.Key, AllianceID: raw.AllianceID, Description: raw.Description, Name: raw.Name}, true
		return nil
	})
	return out, ok, err
}
```

Add to `sde/bloodline.go`, below the existing `Bloodlines` method:

```go
// Bloodline looks up a bloodline by ID. ok is false if the ID isn't in the cache.
func (s *Store) Bloodline(ctx context.Context, userAgent string, id int64) (Bloodline, bool, error) {
	if s == nil {
		return Bloodline{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Bloodline
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("bloodlines"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw bloodlineRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached bloodline %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached bloodline %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Bloodline{
			ID: raw.Key, Charisma: raw.Charisma, CorporationID: raw.CorporationID, Description: raw.Description,
			Intelligence: raw.Intelligence, Memory: raw.Memory, Name: raw.Name, Perception: raw.Perception,
			RaceID: raw.RaceID, ShipTypeID: raw.ShipTypeID, Willpower: raw.Willpower,
		}, true
		return nil
	})
	return out, ok, err
}
```

Add to `sde/faction.go`, below the existing `Factions` method:

```go
// Faction looks up a faction by ID. ok is false if the ID isn't in the cache.
func (s *Store) Faction(ctx context.Context, userAgent string, id int64) (Faction, bool, error) {
	if s == nil {
		return Faction{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Faction
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("factions"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw factionRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached faction %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached faction %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Faction{
			ID: raw.Key, Name: raw.Name, Description: raw.Description, IsUnique: raw.IsUnique,
			SizeFactor: raw.SizeFactor, CorporationID: raw.CorporationID, MilitiaCorporationID: raw.MilitiaCorporationID,
			SolarSystemID: raw.SolarSystemID, StationCount: raw.StationCount, StationSystemCount: raw.StationSystemCount,
		}, true
		return nil
	})
	return out, ok, err
}
```

`int64Key` is already defined in `sde/sync.go` and used by `sde/group.go`'s `Group()` — no new
helper needed. No `sync.go` changes: `loadRaces`/`loadBloodlines`/`loadFactions` already write
these buckets keyed by `int64Key(id)`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./sde/... -run "TestRace_KnownID|TestRace_UnknownID|TestRace_MalformedCachedRecord|TestRace_MismatchedKey|TestBloodline_KnownID|TestBloodline_UnknownID|TestBloodline_MalformedCachedRecord|TestBloodline_MismatchedKey|TestFaction_KnownID|TestFaction_UnknownID|TestFaction_MalformedCachedRecord|TestFaction_MismatchedKey" -v`
Expected: PASS (12 tests).

- [ ] **Step 5: Commit**

```bash
git add sde/race.go sde/race_test.go sde/bloodline.go sde/bloodline_test.go sde/faction.go sde/faction_test.go
git commit -m "feat: add single-ID Race/Bloodline/Faction accessors to sde.Store"
```

---

### Task 2: `mcp/enrich.go` — four new resolvers

**Files:**
- Modify: `mcp/enrich.go` (add four resolvers alongside the existing seven)
- Modify: `mcp/enrich_internal_test.go` (add four sets of tests alongside the existing ones)

**Interfaces:**
- Consumes: `Store.Race`/`Store.Bloodline`/`Store.Faction` (Task 1), `Store.Group` (already
  exists).
- Produces: `resolveRaceNames`, `resolveBloodlineNames`, `resolveFactionNames`, `resolveGroupNames`
  — every later task calls one of these four functions by name, with signature
  `func(ctx context.Context, sdeStore <kind>NameCache, userAgent string, ids []int64) map[int64]string`.

- [ ] **Step 1: Write the failing tests**

Add to `mcp/enrich_internal_test.go`, mirroring the existing `fakeSystemNameCache`/
`TestResolveSystemNames_*` block exactly (all four entities read `.Name["en"]`, so all four fakes
and test sets share one shape):

```go
type fakeRaceNameCache struct {
	races map[int64]sde.Race
	errs  map[int64]error
	calls map[int64]int
}

func (f *fakeRaceNameCache) Race(ctx context.Context, userAgent string, id int64) (sde.Race, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Race{}, false, err
	}
	r, ok := f.races[id]
	return r, ok, nil
}

func TestResolveRaceNames_Dedup(t *testing.T) {
	cache := &fakeRaceNameCache{races: map[int64]sde.Race{
		1: {ID: 1, Name: map[string]string{"en": "Caldari"}},
		2: {ID: 2, Name: map[string]string{"en": "Minmatar"}},
	}}

	got := resolveRaceNames(context.Background(), cache, "test-agent", []int64{1, 1, 2})

	if got[1] != "Caldari" {
		t.Errorf("got[1] = %q, want %q", got[1], "Caldari")
	}
	if got[2] != "Minmatar" {
		t.Errorf("got[2] = %q, want %q", got[2], "Minmatar")
	}
	if cache.calls[1] != 1 {
		t.Errorf("calls[1] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[1])
	}
}

func TestResolveRaceNames_MissOmitted(t *testing.T) {
	cache := &fakeRaceNameCache{races: map[int64]sde.Race{}}

	got := resolveRaceNames(context.Background(), cache, "test-agent", []int64{999})

	if _, ok := got[999]; ok {
		t.Errorf("got[999] present = %v, want omitted on a cache miss", got[999])
	}
}

func TestResolveRaceNames_ErrorOmitted(t *testing.T) {
	cache := &fakeRaceNameCache{errs: map[int64]error{1: context.DeadlineExceeded}}

	got := resolveRaceNames(context.Background(), cache, "test-agent", []int64{1})

	if _, ok := got[1]; ok {
		t.Errorf("got[1] present = %v, want omitted when the lookup errors", got[1])
	}
}

func TestResolveRaceNames_EmptyIDs(t *testing.T) {
	cache := &fakeRaceNameCache{}

	got := resolveRaceNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}
```

Repeat the identical four-test shape for `resolveBloodlineNames` (`fakeBloodlineNameCache`,
`Bloodline(ctx, userAgent, id) (sde.Bloodline, bool, error)`, sample data e.g.
`1: {ID: 1, Name: map[string]string{"en": "Deteis"}}`), `resolveFactionNames`
(`fakeFactionNameCache`, `Faction(...)`, e.g.
`500001: {ID: 500001, Name: map[string]string{"en": "Caldari State"}}`), and `resolveGroupNames`
(`fakeGroupNameCache`, `Group(...)`, e.g. `25: {ID: 25, Name: map[string]string{"en": "Frigate"}}`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestResolveRaceNames|TestResolveBloodlineNames|TestResolveFactionNames|TestResolveGroupNames" -v`
Expected: FAIL — none of the four resolvers/interfaces/fakes exist yet.

- [ ] **Step 3: Implement the four resolvers**

Add to `mcp/enrich.go`, below the existing `resolveMoonNames`:

```go
// raceNameCache is the minimal read-only view of sde.Store.Race() that
// resolveRaceNames needs.
type raceNameCache interface {
	Race(ctx context.Context, userAgent string, id int64) (sde.Race, bool, error)
}

var resolveRaceNames = func(ctx context.Context, sdeStore raceNameCache, userAgent string, ids []int64) map[int64]string {
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
		r, ok, err := sdeStore.Race(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = r.Name["en"]
	}
	return names
}

// bloodlineNameCache is the minimal read-only view of sde.Store.Bloodline()
// that resolveBloodlineNames needs.
type bloodlineNameCache interface {
	Bloodline(ctx context.Context, userAgent string, id int64) (sde.Bloodline, bool, error)
}

var resolveBloodlineNames = func(ctx context.Context, sdeStore bloodlineNameCache, userAgent string, ids []int64) map[int64]string {
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
		b, ok, err := sdeStore.Bloodline(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = b.Name["en"]
	}
	return names
}

// factionNameCache is the minimal read-only view of sde.Store.Faction() that
// resolveFactionNames needs.
type factionNameCache interface {
	Faction(ctx context.Context, userAgent string, id int64) (sde.Faction, bool, error)
}

var resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
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
		f, ok, err := sdeStore.Faction(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = f.Name["en"]
	}
	return names
}

// groupNameCache is the minimal read-only view of sde.Store.Group() that
// resolveGroupNames needs -- narrower than universe.go's groupCache, which
// also exposes PutGroup/GroupIDs for the Universe tag's own write-back and
// bulk-listing needs that this batch's tools don't use.
type groupNameCache interface {
	Group(ctx context.Context, userAgent string, id int64) (sde.Group, bool, error)
}

var resolveGroupNames = func(ctx context.Context, sdeStore groupNameCache, userAgent string, ids []int64) map[int64]string {
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
		g, ok, err := sdeStore.Group(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = g.Name["en"]
	}
	return names
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestResolveRaceNames|TestResolveBloodlineNames|TestResolveFactionNames|TestResolveGroupNames" -v`
Expected: PASS (16 subtests total, four per resolver).

- [ ] **Step 5: Commit**

```bash
git add mcp/enrich.go mcp/enrich_internal_test.go
git commit -m "feat: add race/bloodline/faction/group name resolvers"
```

---

### Task 3: `mcp/alliance.go` — `GetAllianceOutput.FactionName`

**Files:**
- Modify: `mcp/alliance.go` (`AllianceDetailResult.FactionID`/`GetAllianceOutput.FactionID`,
  `getAllianceHandler`)
- Modify: `mcp/server.go` (`get_alliance` call site)
- Test: `mcp/alliance_internal_test.go`

**Interfaces:**
- Consumes: `resolveFactionNames` (Task 2).

- [ ] **Step 1: Add the fields**

```go
type AllianceDetailResult struct {
	Name                  string
	Ticker                string
	CreatorID             int64
	CreatorCorporationID  int64
	DateFounded           time.Time
	ExecutorCorporationID *int64
	FactionID             *int64
	FactionName           string
}
```
```go
type GetAllianceOutput struct {
	AllianceID            int64     `json:"allianceId"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
	CreatorID             int64     `json:"creatorId"`
	CreatorCorporationID  int64     `json:"creatorCorporationId"`
	DateFounded           time.Time `json:"dateFounded"`
	ExecutorCorporationID *int64    `json:"executorCorporationId,omitempty"`
	FactionID             *int64    `json:"factionId,omitempty"`
	FactionName           string    `json:"factionName,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/alliance_internal_test.go
func TestGetAllianceHandler_ResolvesFactionName(t *testing.T) {
	orig := getAllianceDetail
	t.Cleanup(func() { getAllianceDetail = orig })
	factionID := int64(500001)
	getAllianceDetail = func(ctx context.Context, userAgent string, allianceID int64) (AllianceDetailResult, error) {
		return AllianceDetailResult{Name: "Test Alliance", FactionID: &factionID}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getAllianceHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetAllianceInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetAllianceHandler_FactionNameEmptyWhenFactionIDNil(t *testing.T) {
	orig := getAllianceDetail
	t.Cleanup(func() { getAllianceDetail = orig })
	getAllianceDetail = func(ctx context.Context, userAgent string, allianceID int64) (AllianceDetailResult, error) {
		return AllianceDetailResult{Name: "Test Alliance"}, nil
	}

	handler := getAllianceHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetAllianceInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "" {
		t.Errorf("FactionName = %q, want empty when FactionID is nil", out.FactionName)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestGetAllianceHandler_ResolvesFactionName|TestGetAllianceHandler_FactionNameEmptyWhenFactionIDNil" -v`
Expected: FAIL — `getAllianceHandler` doesn't accept a 2nd argument yet, `AllianceDetailResult`/
`GetAllianceOutput` don't have the new field.

- [ ] **Step 4: Implement enrichment**

Add `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to `mcp/alliance.go`'s imports. Change
`getAllianceDetail`'s composite literal to also set `FactionName: ""` implicitly (no change needed
there — it's populated by the handler, not the detail-fetch func). Change `getAllianceHandler`'s
signature to `getAllianceHandler(version string, sdeStore *sde.Store)`, and after
`result, err := getAllianceDetail(...)` succeeds, insert:
```go
		if result.FactionID != nil {
			names := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), []int64{*result.FactionID})
			result.FactionName = names[*result.FactionID]
		}
```
before the existing `return nil, GetAllianceOutput{...}, nil`, then add `FactionName:
result.FactionName` to that composite literal.

In `mcp/server.go`, update the `get_alliance` call site: `getAllianceHandler(version, sdeStore)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestGetAllianceHandler_ResolvesFactionName|TestGetAllianceHandler_FactionNameEmptyWhenFactionIDNil" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/alliance.go mcp/alliance_internal_test.go mcp/server.go
git commit -m "feat: resolve GetAllianceOutput.FactionName via sde/"
```

---

### Task 4: `mcp/character.go` — `GetCharacterOutput`, `CharacterAffiliation`

**Files:**
- Modify: `mcp/character.go` (`GetCharacterOutput` + `getCharacterHandler`;
  `CharacterAffiliation` + `getCharacterAffiliationsHandler`)
- Modify: `mcp/server.go` (`get_character`, `get_character_affiliations` call sites)
- Test: `mcp/character_internal_test.go`

**Interfaces:**
- Consumes: `resolveFactionNames`, `resolveRaceNames`, `resolveBloodlineNames` (Task 2).

- [ ] **Step 1: Add the fields**

```go
type GetCharacterOutput struct {
	CharacterID      int64     `json:"characterId"`
	Name             string    `json:"name"`
	CorporationID    int64     `json:"corporationId"`
	AllianceID       *int64    `json:"allianceId,omitempty"`
	FactionID        *int64    `json:"factionId,omitempty"`
	FactionName      string    `json:"factionName,omitempty"`
	Birthday         time.Time `json:"birthday"`
	BloodlineID      int64     `json:"bloodlineId"`
	BloodlineName    string    `json:"bloodlineName,omitempty"`
	RaceID           int64     `json:"raceId"`
	RaceName         string    `json:"raceName,omitempty"`
	Gender           string    `json:"gender"`
	SecurityStatus   *float64  `json:"securityStatus,omitempty"`
	Description      *string   `json:"description,omitempty"`
	Title            *string   `json:"title,omitempty"`
	AchievementScore int64     `json:"achievementScore"`
}
```
```go
type CharacterAffiliation struct {
	CharacterID   int64  `json:"characterId"`
	CorporationID int64  `json:"corporationId"`
	AllianceID    *int64 `json:"allianceId,omitempty"`
	FactionID     *int64 `json:"factionId,omitempty"`
	FactionName   string `json:"factionName,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/character_internal_test.go
func TestGetCharacterHandler_ResolvesFactionRaceBloodlineNames(t *testing.T) {
	orig := getCharacterDetail
	t.Cleanup(func() { getCharacterDetail = orig })
	factionID := int64(500001)
	getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
		return GetCharacterOutput{CharacterID: characterID, FactionID: &factionID, BloodlineID: 1, RaceID: 1}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}
	origRace := resolveRaceNames
	t.Cleanup(func() { resolveRaceNames = origRace })
	resolveRaceNames = func(ctx context.Context, sdeStore raceNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1: "Caldari"}
	}
	origBloodline := resolveBloodlineNames
	t.Cleanup(func() { resolveBloodlineNames = origBloodline })
	resolveBloodlineNames = func(ctx context.Context, sdeStore bloodlineNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1: "Deteis"}
	}

	handler := getCharacterHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCharacterInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
	if out.RaceName != "Caldari" {
		t.Errorf("RaceName = %q, want %q", out.RaceName, "Caldari")
	}
	if out.BloodlineName != "Deteis" {
		t.Errorf("BloodlineName = %q, want %q", out.BloodlineName, "Deteis")
	}
}

func TestGetCharacterHandler_FactionNameEmptyWhenFactionIDNil(t *testing.T) {
	orig := getCharacterDetail
	t.Cleanup(func() { getCharacterDetail = orig })
	getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
		return GetCharacterOutput{CharacterID: characterID, BloodlineID: 1, RaceID: 1}, nil
	}

	handler := getCharacterHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCharacterInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "" {
		t.Errorf("FactionName = %q, want empty when FactionID is nil", out.FactionName)
	}
}

func TestGetCharacterAffiliationsHandler_ResolvesFactionName(t *testing.T) {
	orig := getCharacterAffiliations
	t.Cleanup(func() { getCharacterAffiliations = orig })
	factionID := int64(500001)
	getCharacterAffiliations = func(ctx context.Context, userAgent string, characterIDs []int64) ([]CharacterAffiliation, error) {
		return []CharacterAffiliation{{CharacterID: 998877, CorporationID: 98000001, FactionID: &factionID}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getCharacterAffiliationsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCharacterAffiliationsInput{CharacterIDs: []int64{998877}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Affiliations) != 1 || out.Affiliations[0].FactionName != "Caldari State" {
		t.Errorf("Affiliations = %+v, want one entry FactionName=%q", out.Affiliations, "Caldari State")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestGetCharacterHandler_ResolvesFactionRaceBloodlineNames|TestGetCharacterHandler_FactionNameEmptyWhenFactionIDNil|TestGetCharacterAffiliationsHandler_ResolvesFactionName" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

`sde` is already imported in `mcp/character.go`. Change `getCharacterHandler`'s signature to
`getCharacterHandler(version string, sdeStore *sde.Store)`, and after `out, err :=
getCharacterDetail(...)` succeeds, insert:
```go
			raceNames := resolveRaceNames(ctx, sdeStore, sso.UserAgent(version), []int64{out.RaceID})
			out.RaceName = raceNames[out.RaceID]
			bloodlineNames := resolveBloodlineNames(ctx, sdeStore, sso.UserAgent(version), []int64{out.BloodlineID})
			out.BloodlineName = bloodlineNames[out.BloodlineID]
			if out.FactionID != nil {
				factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), []int64{*out.FactionID})
				out.FactionName = factionNames[*out.FactionID]
			}
```
before the existing `return nil, out, nil`.

Change `getCharacterAffiliationsHandler`'s signature to `getCharacterAffiliationsHandler(version
string, sdeStore *sde.Store)`, and after `affs, err := getCharacterAffiliations(...)` succeeds,
insert:
```go
		var factionIDs []int64
		for _, a := range affs {
			if a.FactionID != nil {
				factionIDs = append(factionIDs, *a.FactionID)
			}
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for i := range affs {
			if affs[i].FactionID != nil {
				affs[i].FactionName = factionNames[*affs[i].FactionID]
			}
		}
```
before the existing `return nil, GetCharacterAffiliationsOutput{Affiliations: affs}, nil`.

In `mcp/server.go`, update both call sites: `getCharacterHandler(version, sdeStore)`,
`getCharacterAffiliationsHandler(version, sdeStore)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestGetCharacterHandler_ResolvesFactionRaceBloodlineNames|TestGetCharacterHandler_FactionNameEmptyWhenFactionIDNil|TestGetCharacterAffiliationsHandler_ResolvesFactionName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/character.go mcp/character_internal_test.go mcp/server.go
git commit -m "feat: resolve character faction/race/bloodline names via sde/"
```

---

### Task 5: `mcp/corporation.go` — `GetCorporationOutput.FactionName`

**Files:**
- Modify: `mcp/corporation.go`
- Test: `mcp/corporation_internal_test.go`

**Interfaces:**
- Consumes: `resolveFactionNames` (Task 2).

- [ ] **Step 1: Add the field**

In `GetCorporationOutput` (already has `sdeStore` in `getCorporationHandler` from batch 2's
`HomeStationName` work), add `FactionName` right after `FactionID`:
```go
	FactionID           *int64     `json:"factionId,omitempty"`
	FactionName         string     `json:"factionName,omitempty"`
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/corporation_internal_test.go
func TestGetCorporationHandler_ResolvesFactionName(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	factionID := int64(500001)
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{CorporationID: 98000001, FactionID: &factionID, HomeStationID: 60003760}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}
	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{}
	}

	handler := getCorporationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetCorporationHandler_FactionNameEmptyWhenFactionIDNil(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{CorporationID: 98000001, HomeStationID: 60003760}, nil
	}

	handler := getCorporationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "" {
		t.Errorf("FactionName = %q, want empty when FactionID is nil", out.FactionName)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestGetCorporationHandler_ResolvesFactionName|TestGetCorporationHandler_FactionNameEmptyWhenFactionIDNil" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

After the existing `names := resolveStationNames(...)` / `out.HomeStationName = ...` block in
`getCorporationHandler`, insert:
```go
		if out.FactionID != nil {
			factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), []int64{*out.FactionID})
			out.FactionName = factionNames[*out.FactionID]
		}
```
before the existing `return nil, out, nil`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestGetCorporationHandler_ResolvesFactionName|TestGetCorporationHandler_FactionNameEmptyWhenFactionIDNil" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/corporation.go mcp/corporation_internal_test.go
git commit -m "feat: resolve GetCorporationOutput.FactionName via sde/"
```

---

### Task 6: `mcp/corporation_projects.go` — `ShipTypeMatcher.GroupName`, `IdentityMatcher`/`FactionMatcher.FactionName`

**Files:**
- Modify: `mcp/corporation_projects.go`
- Test: `mcp/corporation_projects_internal_test.go`

**Interfaces:**
- Consumes: `resolveGroupNames`, `resolveFactionNames` (Task 2).

- [ ] **Step 1: Add the fields**

```go
type IdentityMatcher struct {
	AllianceID    *int64 `json:"allianceId,omitempty"`
	CharacterID   *int64 `json:"characterId,omitempty"`
	CorporationID *int64 `json:"corporationId,omitempty"`
	FactionID     *int64 `json:"factionId,omitempty"`
	FactionName   string `json:"factionName,omitempty"`
}
```
```go
type ShipTypeMatcher struct {
	GroupID   *int64 `json:"groupId,omitempty"`
	GroupName string `json:"groupName,omitempty"`
	TypeID    *int64 `json:"typeId,omitempty"`
	TypeName  string `json:"typeName,omitempty"`
}
```
```go
type FactionMatcher struct {
	FactionID   *int64 `json:"factionId,omitempty"`
	FactionName string `json:"factionName,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/corporation_projects_internal_test.go
func TestGetCorporationProjectDetailHandler_ResolvesGroupAndFactionNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })
	groupID := int64(25)
	factionID := int64(500001)
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return &esiapi.CorporationsProjectsDetail{
			Configuration: esiapi.CorporationsProjectsDetailConfiguration{
				DamageShip: &esiapi.DamageShip{DamageShip: &esiapi.CorporationsProjectsDetailConfigurationdamageship{
					Identities: []esiapi.CorporationsProjectsDetailConfigurationdamageshipIdentitiesInner{
						{FactionId: &esiapi.FactionId{FactionId: &factionID}},
					},
					Ships: []esiapi.CorporationsProjectsDetailConfigurationdamageshipShipsInner{
						{GroupId: &esiapi.GroupId{GroupId: &groupID}},
					},
				}},
			},
			Creator:  esiapi.CorporationsProjectsDetailCreator{Id: 998877, Name: "Creator Pilot"},
			Details:  esiapi.CorporationsProjectsDetailDetails{Career: "Exploration"},
			Id:       "abc-200",
			Name:     "Test Project",
			Progress: esiapi.CorporationsProjectsDetailProgress{Current: 0, Desired: 1},
			State:    "Active",
		}, nil
	}

	origGroup := resolveGroupNames
	t.Cleanup(func() { resolveGroupNames = origGroup })
	resolveGroupNames = func(ctx context.Context, sdeStore groupNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{25: "Frigate"}
	}
	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-200"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Configuration.Ships) != 1 || out.Configuration.Ships[0].GroupName != "Frigate" {
		t.Errorf("Configuration.Ships = %+v, want one entry GroupName=%q", out.Configuration.Ships, "Frigate")
	}
	if len(out.Configuration.Identities) != 1 || out.Configuration.Identities[0].FactionName != "Caldari State" {
		t.Errorf("Configuration.Identities = %+v, want one entry FactionName=%q", out.Configuration.Identities, "Caldari State")
	}
}
```

Before writing this, confirm the exact `esiapi.DamageShip`/`CorporationsProjectsDetailConfigurationdamageship`/
`FactionId`/`GroupId` generated-model shapes against the vendored `goesi-openapi` package (the same
level of care batch 1/2 applied — these wrapper types follow the established `XId{XId *int64}`
convention already used by `TypeId`/`RegionId`/etc. in this same file, so they should match, but
verify before assuming).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetCorporationProjectDetailHandler_ResolvesGroupAndFactionNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Add a new, separate enrichment block to `getCorporationProjectDetailHandler`, after the existing
`resolveStationNames`/`DockingLocations` block and before the final `return`:
```go
		var groupIDs, factionIDs []int64
		for _, group := range [][]ShipTypeMatcher{configuration.Ships, configuration.Items, configuration.Materials} {
			for _, m := range group {
				if m.GroupID != nil {
					groupIDs = append(groupIDs, *m.GroupID)
				}
			}
		}
		for _, ident := range configuration.Identities {
			if ident.FactionID != nil {
				factionIDs = append(factionIDs, *ident.FactionID)
			}
		}
		for _, fm := range configuration.Factions {
			if fm.FactionID != nil {
				factionIDs = append(factionIDs, *fm.FactionID)
			}
		}
		groupNames := resolveGroupNames(ctx, sdeStore, sso.UserAgent(version), groupIDs)
		projectFactionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for _, group := range [][]ShipTypeMatcher{configuration.Ships, configuration.Items, configuration.Materials} {
			for i := range group {
				if group[i].GroupID != nil {
					group[i].GroupName = groupNames[*group[i].GroupID]
				}
			}
		}
		for i := range configuration.Identities {
			if configuration.Identities[i].FactionID != nil {
				configuration.Identities[i].FactionName = projectFactionNames[*configuration.Identities[i].FactionID]
			}
		}
		for i := range configuration.Factions {
			if configuration.Factions[i].FactionID != nil {
				configuration.Factions[i].FactionName = projectFactionNames[*configuration.Factions[i].FactionID]
			}
		}
```
(This must NOT be merged into the existing `resolveTypeNames`-based `TypeID`/`TypeName` loop over
`[][]ShipTypeMatcher{...}` — it is a separate pass over the same slices, resolving a different ID
kind via a different resolver, following the established "additive, separate pass" convention.)

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetCorporationProjectDetailHandler_ResolvesGroupAndFactionNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/corporation_projects.go mcp/corporation_projects_internal_test.go
git commit -m "feat: resolve ShipTypeMatcher.GroupName and Identity/FactionMatcher.FactionName via sde/"
```

---

### Task 7: `mcp/faction_warfare.go` — six handlers, the largest task in this plan

**Files:**
- Modify: `mcp/faction_warfare.go`
- Modify: `mcp/server.go` (five call sites: `get_character_fw_stats`, `get_corporation_fw_stats`,
  `get_fw_leaderboards`, `list_fw_stats`, `list_fw_wars` — `list_fw_systems` already has `sdeStore`)
- Test: `mcp/faction_warfare_internal_test.go`

**Interfaces:**
- Consumes: `resolveFactionNames` (Task 2).

- [ ] **Step 1: Add the fields**

```go
type GetCharacterFwStatsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CurrentRank   *int64             `json:"currentRank,omitempty"`
	HighestRank   *int64             `json:"highestRank,omitempty"`
	FactionID     *int64             `json:"factionId,omitempty"`
	FactionName   string             `json:"factionName,omitempty"`
	EnlistedOn    *time.Time         `json:"enlistedOn,omitempty"`
	Kills         KillsOrPointsStats `json:"kills"`
	VictoryPoints KillsOrPointsStats `json:"victoryPoints"`
}
```
```go
type GetCorporationFwStatsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	FactionID     *int64             `json:"factionId,omitempty"`
	FactionName   string             `json:"factionName,omitempty"`
	Pilots        *int64             `json:"pilots,omitempty"`
	EnlistedOn    *time.Time         `json:"enlistedOn,omitempty"`
	Kills         KillsOrPointsStats `json:"kills"`
	VictoryPoints KillsOrPointsStats `json:"victoryPoints"`
}
```
```go
type LeaderboardFactionEntry struct {
	FactionID   *int64 `json:"factionId,omitempty"`
	FactionName string `json:"factionName,omitempty"`
	Amount      *int64 `json:"amount,omitempty"`
}
```
```go
type FwFactionStats struct {
	FactionID         int64              `json:"factionId"`
	FactionName       string             `json:"factionName,omitempty"`
	Pilots            int64              `json:"pilots"`
	SystemsControlled int64              `json:"systemsControlled"`
	Kills             KillsOrPointsStats `json:"kills"`
	VictoryPoints     KillsOrPointsStats `json:"victoryPoints"`
}
```
```go
type FwSystem struct {
	Contested              string `json:"contested"`
	OccupierFactionID      int64  `json:"occupierFactionId"`
	OccupierFactionName    string `json:"occupierFactionName,omitempty"`
	OwnerFactionID         int64  `json:"ownerFactionId"`
	OwnerFactionName       string `json:"ownerFactionName,omitempty"`
	SolarSystemID          int64  `json:"solarSystemId"`
	SolarSystemName        string `json:"solarSystemName,omitempty"`
	VictoryPoints          int64  `json:"victoryPoints"`
	VictoryPointsThreshold int64  `json:"victoryPointsThreshold"`
}
```
```go
type FwWar struct {
	AgainstID   int64  `json:"againstId"`
	FactionID   int64  `json:"factionId"`
	FactionName string `json:"factionName,omitempty"`
}
```

- [ ] **Step 2: Write the failing tests**

```go
// add to mcp/faction_warfare_internal_test.go
func TestGetCharacterFwStatsHandler_ResolvesFactionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterFwStats
	t.Cleanup(func() { getCharacterFwStats = orig })
	factionID := int64(500001)
	getCharacterFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		return &esiapi.CharactersCharacterIdFwStatsGet{FactionId: &factionID}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getCharacterFwStatsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterFwStatsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetCorporationFwStatsHandler_ResolvesFactionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationFwStats
	t.Cleanup(func() { getCorporationFwStats = orig })
	factionID := int64(500001)
	getCorporationFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		return &esiapi.CorporationsCorporationIdFwStatsGet{FactionId: &factionID}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getCorporationFwStatsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetFwLeaderboardsHandler_ResolvesFactionNames(t *testing.T) {
	orig := getFwLeaderboards
	t.Cleanup(func() { getFwLeaderboards = orig })
	factionID := int64(500001)
	amount := int64(42)
	getFwLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsGet, error) {
		return &esiapi.FwLeaderboardsGet{
			Kills: esiapi.FwLeaderboardsGetKills{
				ActiveTotal: []esiapi.FwLeaderboardsGetKillsActiveTotalInner{{FactionId: &factionID, Amount: &amount}},
			},
			VictoryPoints: esiapi.FwLeaderboardsGetVictoryPoints{},
		}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getFwLeaderboardsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetFwLeaderboardsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Kills.ActiveTotal) != 1 || out.Kills.ActiveTotal[0].FactionName != "Caldari State" {
		t.Errorf("Kills.ActiveTotal = %+v, want one entry FactionName=%q", out.Kills.ActiveTotal, "Caldari State")
	}
}

func TestListFwStatsHandler_ResolvesFactionName(t *testing.T) {
	orig := listFwStats
	t.Cleanup(func() { listFwStats = orig })
	listFwStats = func(ctx context.Context, userAgent string) ([]esiapi.FwStatsGetInner, error) {
		return []esiapi.FwStatsGetInner{{FactionId: 500001}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := listFwStatsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListFwStatsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Stats) != 1 || out.Stats[0].FactionName != "Caldari State" {
		t.Errorf("Stats = %+v, want one entry FactionName=%q", out.Stats, "Caldari State")
	}
}

func TestListFwSystemsHandler_ResolvesOccupierAndOwnerFactionNames(t *testing.T) {
	orig := listFwSystems
	t.Cleanup(func() { listFwSystems = orig })
	listFwSystems = func(ctx context.Context, userAgent string) ([]esiapi.FwSystemsGetInner, error) {
		return []esiapi.FwSystemsGetInner{{OccupierFactionId: 500001, OwnerFactionId: 500002, SolarSystemId: 30000142}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State", 500002: "Gallente Federation"}
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
	if len(out.Systems) != 1 || out.Systems[0].OccupierFactionName != "Caldari State" || out.Systems[0].OwnerFactionName != "Gallente Federation" {
		t.Errorf("Systems = %+v, want one entry OccupierFactionName=%q OwnerFactionName=%q", out.Systems, "Caldari State", "Gallente Federation")
	}
}

func TestListFwWarsHandler_ResolvesFactionName(t *testing.T) {
	orig := listFwWars
	t.Cleanup(func() { listFwWars = orig })
	listFwWars = func(ctx context.Context, userAgent string) ([]esiapi.FwWarsGetInner, error) {
		return []esiapi.FwWarsGetInner{{AgainstId: 500002, FactionId: 500001}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := listFwWarsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListFwWarsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Wars) != 1 || out.Wars[0].FactionName != "Caldari State" {
		t.Errorf("Wars = %+v, want one entry FactionName=%q", out.Wars, "Caldari State")
	}
}
```

Before finalizing, confirm the exact `esiapi.FwLeaderboardsGetKillsActiveTotalInner`/etc. generated
type names against the real file (this test file's own existing tests already construct
`esiapi.CharactersCharacterIdFwStatsGet`, `esiapi.FwStatsGetInner`, etc. — mirror those exactly).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./mcp/... -run "TestGetCharacterFwStatsHandler_ResolvesFactionName|TestGetCorporationFwStatsHandler_ResolvesFactionName|TestGetFwLeaderboardsHandler_ResolvesFactionNames|TestListFwStatsHandler_ResolvesFactionName|TestListFwSystemsHandler_ResolvesOccupierAndOwnerFactionNames|TestListFwWarsHandler_ResolvesFactionName" -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment in all six handlers**

`getCharacterFwStatsHandler`: change signature to `getCharacterFwStatsHandler(version, clientID
string, store *config.Store, sdeStore *sde.Store)`. After the existing `stats, err :=
getCharacterFwStats(...)` succeeds, insert a single-scalar factionID nil-guard before the return:
```go
			var factionName string
			if stats.FactionId != nil {
				names := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), []int64{*stats.FactionId})
				factionName = names[*stats.FactionId]
			}
```
and add `FactionName: factionName` to the `GetCharacterFwStatsOutput{...}` composite literal.

`getCorporationFwStatsHandler`: identical shape, change signature to add `sdeStore *sde.Store`,
same `factionName` nil-guard block, add `FactionName: factionName` to the output literal.

`getFwLeaderboardsHandler`: change signature to `getFwLeaderboardsHandler(version string, sdeStore
*sde.Store)`. After all six slices (`killsActiveTotal`, `killsLastWeek`, `killsYesterday`,
`vpActiveTotal`, `vpLastWeek`, `vpYesterday`) are built, insert one combined collect-resolve-assign
pass before the final `return`:
```go
		var factionIDs []int64
		for _, group := range [][]LeaderboardFactionEntry{killsActiveTotal, killsLastWeek, killsYesterday, vpActiveTotal, vpLastWeek, vpYesterday} {
			for _, e := range group {
				if e.FactionID != nil {
					factionIDs = append(factionIDs, *e.FactionID)
				}
			}
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for _, group := range [][]LeaderboardFactionEntry{killsActiveTotal, killsLastWeek, killsYesterday, vpActiveTotal, vpLastWeek, vpYesterday} {
			for i := range group {
				if group[i].FactionID != nil {
					group[i].FactionName = factionNames[*group[i].FactionID]
				}
			}
		}
```
(one combined `resolveFactionNames` call across all six slices, mirroring the sovereignty-hub
combined-call pattern from batch 2).

`listFwStatsHandler`: change signature to `listFwStatsHandler(version string, sdeStore
*sde.Store)`. After the `out` slice is built, add the standard flat-slice `resolveFactionNames`
shape on `FactionID`/`FactionName` (plain `int64`, no nil-guard needed).

`listFwSystemsHandler`: already has `sdeStore`. After the existing `resolveSystemNames`/
`SolarSystemName` block, add:
```go
		var factionIDs []int64
		for _, s := range out {
			factionIDs = append(factionIDs, s.OccupierFactionID, s.OwnerFactionID)
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for i := range out {
			out[i].OccupierFactionName = factionNames[out[i].OccupierFactionID]
			out[i].OwnerFactionName = factionNames[out[i].OwnerFactionID]
		}
```

`listFwWarsHandler`: change signature to `listFwWarsHandler(version string, sdeStore *sde.Store)`.
After the `out` slice is built, add the standard flat-slice `resolveFactionNames` shape on
`FactionID`/`FactionName` (plain `int64`, no nil-guard needed).

In `mcp/server.go`, update the five call sites needing a new `sdeStore` argument:
`getCharacterFwStatsHandler(version, clientID, store, sdeStore)`,
`getCorporationFwStatsHandler(version, clientID, store, sdeStore)`,
`getFwLeaderboardsHandler(version, sdeStore)`, `listFwStatsHandler(version, sdeStore)`,
`listFwWarsHandler(version, sdeStore)`. `list_fw_systems` needs no change (already passes
`sdeStore`).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./mcp/... -run "TestGetCharacterFwStatsHandler_ResolvesFactionName|TestGetCorporationFwStatsHandler_ResolvesFactionName|TestGetFwLeaderboardsHandler_ResolvesFactionNames|TestListFwStatsHandler_ResolvesFactionName|TestListFwSystemsHandler_ResolvesOccupierAndOwnerFactionNames|TestListFwWarsHandler_ResolvesFactionName" -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/faction_warfare.go mcp/faction_warfare_internal_test.go mcp/server.go
git commit -m "feat: resolve faction warfare tag's faction names via sde/"
```

---

### Task 8: `mcp/incursions.go` — `Incursion.FactionName`

**Files:**
- Modify: `mcp/incursions.go`
- Test: `mcp/incursions_internal_test.go`

**Interfaces:**
- Consumes: `resolveFactionNames` (Task 2).

- [ ] **Step 1: Add the field**

`Incursion.FactionID` is plain `int64` (already has `sdeStore` from batch 2). Add `FactionName`
right after `FactionID`:
```go
	FactionID                int64    `json:"factionId"`
	FactionName              string   `json:"factionName,omitempty"`
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/incursions_internal_test.go
func TestListIncursionsHandler_ResolvesFactionName(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return []Incursion{{FactionID: 500001}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Sansha's Nation"}
	}

	handler := listIncursionsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIncursionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Incursions) != 1 || out.Incursions[0].FactionName != "Sansha's Nation" {
		t.Errorf("Incursions = %+v, want one entry FactionName=%q", out.Incursions, "Sansha's Nation")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListIncursionsHandler_ResolvesFactionName -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

In `listIncursionsHandler`, add `factionIDs = append(factionIDs, inc.FactionID)` to the existing
`for _, inc := range incursions { ... }` collect loop (alongside the existing
`constellationIDs`/`systemIDs` appends), then add:
```go
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
```
alongside the existing `resolveConstellationNames`/`resolveSystemNames` calls, and add
`incursions[i].FactionName = factionNames[incursions[i].FactionID]` to the existing assign loop.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListIncursionsHandler_ResolvesFactionName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/incursions.go mcp/incursions_internal_test.go
git commit -m "feat: resolve Incursion.FactionName via sde/"
```

---

### Task 9: `mcp/killmails.go` — `KillmailAttacker`/`KillmailVictim.FactionName`

**Files:**
- Modify: `mcp/killmails.go`
- Test: `mcp/killmails_internal_test.go`

**Interfaces:**
- Consumes: `resolveFactionNames` (Task 2).

- [ ] **Step 1: Add the fields**

```go
type KillmailAttacker struct {
	AllianceID     *int64  `json:"allianceId,omitempty"`
	CharacterID    *int64  `json:"characterId,omitempty"`
	CorporationID  *int64  `json:"corporationId,omitempty"`
	DamageDone     int64   `json:"damageDone"`
	FactionID      *int64  `json:"factionId,omitempty"`
	FactionName    string  `json:"factionName,omitempty"`
	FinalBlow      bool    `json:"finalBlow"`
	SecurityStatus float64 `json:"securityStatus"`
	ShipTypeID     *int64  `json:"shipTypeId,omitempty"`
	ShipTypeName   string  `json:"shipTypeName,omitempty"`
	WeaponTypeID   *int64  `json:"weaponTypeId,omitempty"`
	WeaponTypeName string  `json:"weaponTypeName,omitempty"`
}
```
```go
type KillmailVictim struct {
	AllianceID    *int64            `json:"allianceId,omitempty"`
	CharacterID   *int64            `json:"characterId,omitempty"`
	CorporationID *int64            `json:"corporationId,omitempty"`
	DamageTaken   int64             `json:"damageTaken"`
	FactionID     *int64            `json:"factionId,omitempty"`
	FactionName   string            `json:"factionName,omitempty"`
	ShipTypeID    int64             `json:"shipTypeId"`
	ShipTypeName  string            `json:"shipTypeName,omitempty"`
	Items         []KillmailItem    `json:"items,omitempty"`
	Position      *KillmailPosition `json:"position,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/killmails_internal_test.go
func TestGetKillmailHandler_ResolvesFactionNames(t *testing.T) {
	orig := getKillmail
	t.Cleanup(func() { getKillmail = orig })
	attackerFactionID := int64(500001)
	victimFactionID := int64(500002)
	getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
		return &esiapi.KillmailsKillmailIdKillmailHashGet{
			Attackers: []esiapi.KillmailsKillmailIdKillmailHashGetAttackersInner{
				{FactionId: &attackerFactionID, FinalBlow: true},
			},
			Victim: esiapi.KillmailsKillmailIdKillmailHashGetVictim{FactionId: &victimFactionID, ShipTypeId: 587},
		}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State", 500002: "Gallente Federation"}
	}

	handler := getKillmailHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetKillmailInput{KillmailID: 1, KillmailHash: "abc"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Attackers) != 1 || out.Attackers[0].FactionName != "Caldari State" {
		t.Errorf("Attackers = %+v, want one entry FactionName=%q", out.Attackers, "Caldari State")
	}
	if out.Victim.FactionName != "Gallente Federation" {
		t.Errorf("Victim.FactionName = %q, want %q", out.Victim.FactionName, "Gallente Federation")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestGetKillmailHandler_ResolvesFactionNames -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

Add a new, separate block to `getKillmailHandler`, after the existing
`resolveSystemNames`/`resolveMoonNames` block and before the final `return`:
```go
		var factionIDs []int64
		for _, a := range attackers {
			if a.FactionID != nil {
				factionIDs = append(factionIDs, *a.FactionID)
			}
		}
		if victim.FactionID != nil {
			factionIDs = append(factionIDs, *victim.FactionID)
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for i := range attackers {
			if attackers[i].FactionID != nil {
				attackers[i].FactionName = factionNames[*attackers[i].FactionID]
			}
		}
		if victim.FactionID != nil {
			victim.FactionName = factionNames[*victim.FactionID]
		}
```
This must be a SEPARATE block from the existing `resolveTypeNames`/`resolveSystemNames`/
`resolveMoonNames` calls, not merged into any of them — different entity kind, different
resolver.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestGetKillmailHandler_ResolvesFactionNames -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/killmails.go mcp/killmails_internal_test.go
git commit -m "feat: resolve killmail attacker/victim faction names via sde/"
```

---

### Task 10: `mcp/sovereignty.go` — `SovereigntyClaim.FactionName`

**Files:**
- Modify: `mcp/sovereignty.go`
- Test: `mcp/sovereignty_internal_test.go`

**Interfaces:**
- Consumes: `resolveFactionNames` (Task 2).

- [ ] **Step 1: Add the field**

`SovereigntyClaim.FactionID` (`*int64`) is populated by `toSovereigntyClaim`'s `"faction"` branch,
inside `SovereigntySystem.Claim`, used by `listSovereigntySystemsHandler` (already has
`sdeStore`). Add `FactionName` right after `FactionID`:
```go
type SovereigntyClaim struct {
	Kind            string                  `json:"kind"` // alliance, faction, or unclaimed
	AllianceID      *int64                  `json:"allianceId,omitempty"`
	CorporationID   *int64                  `json:"corporationId,omitempty"`
	ClaimedSince    *time.Time              `json:"claimedSince,omitempty"`
	IsCapitalSystem *bool                   `json:"isCapitalSystem,omitempty"`
	Development     *SovereigntyDevelopment `json:"development,omitempty"`
	SovereigntyHub  *SovereigntyHub         `json:"sovereigntyHub,omitempty"`
	FactionID       *int64                  `json:"factionId,omitempty"`
	FactionName     string                  `json:"factionName,omitempty"`
}
```

- [ ] **Step 2: Write the failing test**

```go
// add to mcp/sovereignty_internal_test.go
func TestListSovereigntySystemsHandler_ResolvesClaimFactionName(t *testing.T) {
	orig := listSovereigntySystems
	t.Cleanup(func() { listSovereigntySystems = orig })
	factionID := int64(500001)
	listSovereigntySystems = func(ctx context.Context, userAgent string) ([]esiapi.SovereigntySystemsSolarsystem, error) {
		return []esiapi.SovereigntySystemsSolarsystem{
			{SolarSystemId: 30000142, Claim: esiapi.SovereigntySystemsSolarsystemClaim{
				Faction: &esiapi.Faction{Faction: &esiapi.SovereigntySystemsSolarsystemClaimfaction{FactionId: factionID}},
			}},
		}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Sansha's Nation"}
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
	if len(out.Systems) != 1 || out.Systems[0].Claim.FactionName != "Sansha's Nation" {
		t.Errorf("Systems = %+v, want one entry Claim.FactionName=%q", out.Systems, "Sansha's Nation")
	}
}
```

Before finalizing, confirm the exact `esiapi.SovereigntySystemsSolarsystemClaimfaction`/`Faction`
wrapper-type field names against the real vendored model (the existing `toSovereigntyClaim`
function in this file already constructs against these exact types — mirror its usage).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./mcp/... -run TestListSovereigntySystemsHandler_ResolvesClaimFactionName -v`
Expected: FAIL.

- [ ] **Step 4: Implement enrichment**

In `listSovereigntySystemsHandler`, after the existing `systemNames := resolveSystemNames(...)` /
assign loop, add a new block:
```go
		var factionIDs []int64
		for _, s := range out {
			if s.Claim.FactionID != nil {
				factionIDs = append(factionIDs, *s.Claim.FactionID)
			}
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for i := range out {
			if out[i].Claim.FactionID != nil {
				out[i].Claim.FactionName = factionNames[*out[i].Claim.FactionID]
			}
		}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mcp/... -run TestListSovereigntySystemsHandler_ResolvesClaimFactionName -v`
Expected: PASS.

- [ ] **Step 6: Run the full package suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... -race`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add mcp/sovereignty.go mcp/sovereignty_internal_test.go
git commit -m "feat: resolve SovereigntyClaim.FactionName via sde/"
```

---

### Task 11: Docs sync + VERSION bump

**Files:**
- Modify: `docs/reference/esi-coverage-roadmap.md`
- Modify: `CLAUDE.md` (package table's `mcp/` row)
- Modify: `README.md` (matching `mcp/` row)
- Modify: `VERSION`, `claude_resources/neat/.claude-plugin/plugin.json`,
  `claude_resources/neat/CHANGELOG.md`

**Interfaces:** none — pure docs/version-bump task, run last once every prior task has landed.

- [ ] **Step 1: Update the roadmap doc**

In `docs/reference/esi-coverage-roadmap.md`'s Conventions section, extend the sentence covering
`type_id`/location-ID enrichment with a third clause noting race/bloodline/faction/group IDs are
now also resolved:
```
- Every `esi/` method returns exactly what ESI returns, including bare IDs, with zero
  name-resolution enrichment. `type_id`-like fields, the six location-entity ID kinds
  (system/region/constellation/station/planet/moon), and now race/bloodline/faction/group IDs
  across the non-Universe tags are all enriched at the `mcp/` layer via `mcp/enrich.go`'s resolver
  functions (see `docs/superpowers/specs/2026-07-28-sde-type-id-enrichment-design.md`,
  `docs/superpowers/specs/2026-07-30-sde-location-id-enrichment-design.md`, and
  `docs/superpowers/specs/2026-07-30-sde-race-bloodline-faction-group-enrichment-design.md`) — the
  generic ambiguous `location_id` fields remain the one open follow-up.
```

- [ ] **Step 2: Update `CLAUDE.md`'s and `README.md`'s `mcp/` package-table rows**

Both files have an identical `mcp/enrich.go` clause (extended by batch 2's Task 16 with "Batch 2
extended..."). Extend it in both files with a third sentence, e.g.: "Batch 3 further extended
`mcp/enrich.go` with four sibling resolvers (`resolveRaceNames`/`resolveBloodlineNames`/
`resolveFactionNames`/`resolveGroupNames`) covering the same additive-`*Name` pattern for
race/bloodline/faction/group IDs across 8 more files, closing out batch 1's originally-flagged
follow-up list."

- [ ] **Step 3: Bump VERSION and regenerate the plugin manifest**

```bash
echo "0.4.0" > VERSION  # minor bump -- another additive feature, not a bugfix
go generate ./...
```

Add a changelog entry to `claude_resources/neat/CHANGELOG.md`:
```markdown
## 0.4.0
- Adds SDE-resolved `*Name` fields (`RaceName`, `BloodlineName`, `FactionName`, `GroupName`)
  alongside the existing bare `*ID` fields across 8 non-Universe MCP files (alliance, character,
  corporation, faction warfare, incursions, killmails, sovereignty). Closes out the last of the
  ID kinds flagged in batch 1's own follow-up list. Names resolve from the local SDE cache only,
  same as batches 1-2's enrichment — no ESI calls are made on this path. Purely additive.
```

- [ ] **Step 4: Verify everything is clean**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race && actionlint`
Expected: all clean.

- [ ] **Step 5: Commit**

```bash
git add docs/reference/esi-coverage-roadmap.md CLAUDE.md README.md VERSION claude_resources/neat/.claude-plugin/plugin.json claude_resources/neat/CHANGELOG.md
git commit -m "docs: document race/bloodline/faction/group-ID enrichment in the roadmap and package table, bump VERSION"
```
