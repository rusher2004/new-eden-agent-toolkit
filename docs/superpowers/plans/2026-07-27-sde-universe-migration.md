# Wiring `sde/` into the Universe Tag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate 18 of the Universe tag's 24 ESI-direct-exception tools onto `sde/`, using a read-through-cache-backed-by-ESI pattern; the remaining 6 (celestial detail) stay ESI-direct because their defining field doesn't exist in the SDE at all.

**Architecture:** `sde/` stays dependency-free (no `esi/` import) — each migrated `mcp/universe.go` handler tries `sde.Store` first, and on a cache miss falls back to a live `esi.Client` call, writing the result back into the cache via a new `Put*` method so the next lookup is fast. `sync.go`'s bulk loader becomes a two-phase compute-then-commit algorithm to support the handful of resources needing data computed from *other* files (reverse indices, a 3-hop join) before they can be written.

**Tech Stack:** Go 1.26.1, existing `go.etcd.io/bbolt` v1.5.0 dependency (no new dependency), standard library only otherwise.

## Global Constraints

- Branch: `sde-universe-migration`, checked out from `main` at `2afe554` before Task 1 begins.
- `sde/` must not import `esi/` or `sso/` — the read-through fallback is orchestrated entirely in `mcp/universe.go`, which already imports both.
- `*sde.Store` is opened once in `mcp/server.go`'s `NewServer` and threaded through migrated handler constructors as a parameter, the same DI shape already used for `*config.Store` — never a package-level global.
- Every SDE schema claim in this plan was verified directly against the live `data.everef.net/ccp/sde/` archive during design (real records sampled, cross-checked against `sde.riftforeve.online`'s schema docs) — re-verify anything code-critical the same way before finalizing a task if in doubt.
- 18 tools migrate: `list_universe_types`, `get_universe_type`, `list_universe_groups`, `get_universe_group`, `list_universe_categories`, `get_universe_category`, `list_universe_regions`, `get_universe_region`, `list_universe_constellations`, `get_universe_constellation`, `list_universe_systems`, `get_universe_system`, `list_universe_ancestries`, `list_universe_bloodlines`, `list_universe_races`, `list_universe_factions`, `list_universe_graphics`, `get_universe_graphic`. The 6 celestial-detail tools (`get_universe_asteroid_belt`/`_moon`/`_planet`/`_stargate`/`_star`/`_station`) are **not touched** — no task in this plan modifies their handlers.
- Documented field drops (populated on a cache miss, dropped on a cache hit — no other migrated
  tool's output changes): `get_universe_system` drops `Planets`; `get_universe_type` drops
  `PackagedVolume`; `list_universe_races` drops `AllianceID`; `get_universe_graphic` drops
  `CollisionFile`/`SofDNA`. The first three are in the approved design spec's field-drop table; the
  fourth (`get_universe_graphic`) was discovered during this plan's own verification pass (Task 1's
  `sde/graphic.go` needed a corresponding fix, folded into Task 1 directly) — flag this divergence
  from the spec document to the user, it isn't a silent scope change.
- `fmt.Errorf("...: %w", err)` wrapping at every layer; a failed cache write-back is logged via `slog.Default().WarnContext(ctx, "...", "error", err)` (matching `sde/sync.go`'s existing `ensureSynced` pattern) and swallowed, never returned as the request's error.
- Tool names, input schemas, and (aside from the three field drops) output schemas are unchanged — `mcp/server.go`'s tool registrations and count are unaffected by this plan.
- Every new external dependency, hard rule, or package addition gets its doc-sync pass in the same PR as the code that adds it, per `CLAUDE.md`'s "Keeping docs & Claude resources in sync" section — Task 10 is that pass.

---

### Task 1: `sde/region.go`, `sde/constellation.go`, `sde/graphic.go`

**Files:**
- Create: `sde/region.go`, `sde/region_test.go`
- Create: `sde/constellation.go`, `sde/constellation_test.go`
- Create: `sde/graphic.go`, `sde/graphic_test.go`

**Interfaces:**
- Consumes: `Store.db *bbolt.DB`, `Store.ensureSynced(ctx, userAgent) error`, `int64Key(id int64) []byte`, `seedStore(t)`/`putRecord(t, store, bucket, id, jsonLine)` test helpers — all from the existing sde-infrastructure PR, unchanged.
- Produces (for Task 6's sync.go rewrite and Task 8/9's mcp/ migrations):
  - `type Region struct { ID int64; Name map[string]string; Description map[string]string; ConstellationIDs []int64 }`
  - `func (s *Store) Region(ctx context.Context, userAgent string, id int64) (Region, bool, error)`
  - `func (s *Store) PutRegion(r Region) error`
  - `type Constellation struct { ID int64; RegionID int64; Name map[string]string; Position Position; SystemIDs []int64 }`
  - `type Position struct { X, Y, Z float64 }`
  - `func (s *Store) Constellation(ctx context.Context, userAgent string, id int64) (Constellation, bool, error)`
  - `func (s *Store) PutConstellation(c Constellation) error`
  - `type Graphic struct { ID int64; GraphicFile *string; IconFolder *string; SofFactionName *string; SofHullName *string; SofRaceName *string }`
  - `func (s *Store) Graphic(ctx context.Context, userAgent string, id int64) (Graphic, bool, error)`
  - `func (s *Store) PutGraphic(g Graphic) error`
  - Bucket names `"regions"`, `"constellations"`, `"graphics"` — the contract with Task 6's sync writer.

Verified real record shapes (sampled directly from the SDE archive during design):
```json
// mapRegions.jsonl
{"_key": 10000001, "constellationIDs": [20000001, ...], "description": {"en": "...", ...}, "factionID": 500007, "name": {"en": "Derelik", ...}, "nebulaID": 11799, "position": {"x": ..., "y": ..., "z": ...}, "wormholeClassID": 7}
// mapConstellations.jsonl
{"_key": 20000001, "factionID": 500007, "name": {"en": "San Matar", ...}, "position": {"x": ..., "y": ..., "z": ...}, "regionID": 10000001, "solarSystemIDs": [30000001, ...], "wormholeClassID": 7}
// graphics.jsonl (verified via a full 6051-record scan of the live archive, not a sample)
{"_key": 10, "graphicFile": "res:/dx9/model/worldobject/planet/moon.red"}
{"_key": 38, "iconFolder": "res:/dx9/model/ship/caldari/frigate/cf1/icons", "sofFactionName": "caldaribase", "sofHullName": "cf1_t1", "sofRaceName": "caldari"}
```
`description` is optional on regions (confirmed via schema docs — not every region has one).
`graphics.jsonl`'s full key set across all 6051 records is `_key, graphicFile, iconFolder,
sofFactionName, sofHullName, sofLayout, sofMaterialSetID, sofRaceName` — every existing
`UniverseGraphic` field has SDE coverage **except** `CollisionFile`/`SofDNA`, confirmed absent from
every one of the 6051 records (zero hits, not a sample coincidence). `Graphic` models the five
present fields (`GraphicFile`, `IconFolder`, `SofFactionName`, `SofHullName`, `SofRaceName`);
`CollisionFile`/`SofDNA` get the same "returned on a cache miss,
dropped on a cache hit" treatment as Task 7/8's other field drops (handled in Task 9, since that's
where `get_universe_graphic` migrates).

- [ ] **Step 1: Write the failing tests for `Region`**

Create `sde/region_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestRegion_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000001,
		`{"_key": 10000001, "constellationIDs": [20000001, 20000002], "description": {"en": "A dangerous frontier region.", "de": "Eine gefaehrliche Grenzregion."}, "name": {"en": "Derelik", "de": "Derelik"}}`)

	got, ok, err := store.Region(context.Background(), "test-agent", 10000001)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true")
	}
	if got.ID != 10000001 {
		t.Errorf("got.ID = %d, want 10000001", got.ID)
	}
	if got.Name["en"] != "Derelik" || got.Name["de"] != "Derelik" {
		t.Errorf("Name = %+v, want en=Derelik de=Derelik", got.Name)
	}
	if got.Description["en"] != "A dangerous frontier region." || got.Description["de"] != "Eine gefaehrliche Grenzregion." {
		t.Errorf("Description = %+v, want en/de set", got.Description)
	}
	if len(got.ConstellationIDs) != 2 || got.ConstellationIDs[0] != 20000001 || got.ConstellationIDs[1] != 20000002 {
		t.Errorf("ConstellationIDs = %v, want [20000001 20000002]", got.ConstellationIDs)
	}
}

func TestRegion_NoDescription_ReturnsNilDescription(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000002,
		`{"_key": 10000002, "constellationIDs": [], "name": {"en": "Region With No Description"}}`)

	got, ok, err := store.Region(context.Background(), "test-agent", 10000002)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true")
	}
	if got.Description != nil {
		t.Errorf("Description = %v, want nil", got.Description)
	}
}

func TestRegion_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Region(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if ok {
		t.Error("Region ok = true, want false for an unknown ID")
	}
}

func TestRegion_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000001,
		`{"_key": 999, "constellationIDs": [], "name": {"en": "Derelik"}}`)

	_, _, err := store.Region(context.Background(), "test-agent", 10000001)
	if err == nil {
		t.Fatal("Region returned nil error, want an error for a mismatched _key")
	}
}

func TestPutRegion_ThenRegion_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Region{ID: 10000003, Name: map[string]string{"en": "New Region"}, Description: map[string]string{"en": "Written via read-through."}, ConstellationIDs: []int64{20000099}}

	if err := store.PutRegion(want); err != nil {
		t.Fatalf("PutRegion returned error: %v", err)
	}

	got, ok, err := store.Region(context.Background(), "test-agent", 10000003)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false after PutRegion, want true")
	}
	if got.ID != want.ID || got.Name["en"] != want.Name["en"] || got.Description["en"] != want.Description["en"] {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.ConstellationIDs) != 1 || got.ConstellationIDs[0] != 20000099 {
		t.Errorf("ConstellationIDs = %v, want [20000099]", got.ConstellationIDs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run TestRegion -v`
Expected: FAIL to compile — `undefined: Region` / `s.Region undefined` / `s.PutRegion undefined`.

- [ ] **Step 3: Write the minimal implementation**

Create `sde/region.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Region is a minimal decode of an SDE region record -- the top level of
// EVE's geography (e.g. "Derelik").
type Region struct {
	ID               int64
	Name             map[string]string
	Description      map[string]string
	ConstellationIDs []int64
}

type regionRecord struct {
	Key              int64             `json:"_key"`
	Name             map[string]string `json:"name"`
	Description      map[string]string `json:"description,omitempty"`
	ConstellationIDs []int64           `json:"constellationIDs"`
}

// Region looks up a region by ID. ok is false if the ID isn't in the cache.
func (s *Store) Region(ctx context.Context, userAgent string, id int64) (Region, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Region
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("regions"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw regionRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached region %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached region %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Region{ID: raw.Key, Name: raw.Name, Description: raw.Description, ConstellationIDs: raw.ConstellationIDs}, true
		return nil
	})
	return out, ok, err
}

// PutRegion writes r into the cache, keyed by r.ID -- used by mcp/'s
// read-through fallback to cache a region fetched live from ESI.
func (s *Store) PutRegion(r Region) error {
	raw, err := json.Marshal(regionRecord{Key: r.ID, Name: r.Name, Description: r.Description, ConstellationIDs: r.ConstellationIDs})
	if err != nil {
		return fmt.Errorf("marshal region %d: %w", r.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("regions"))
		if err != nil {
			return fmt.Errorf("create regions bucket: %w", err)
		}
		return b.Put(int64Key(r.ID), raw)
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -run TestRegion -v`
Expected: PASS (5/5).

- [ ] **Step 5: Write the failing tests for `Constellation`**

Create `sde/constellation_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestConstellation_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "constellations", 20000001,
		`{"_key": 20000001, "name": {"en": "San Matar", "de": "San Matar"}, "position": {"x": 1.5, "y": 2.5, "z": 3.5}, "regionID": 10000001, "solarSystemIDs": [30000001, 30000002]}`)

	got, ok, err := store.Constellation(context.Background(), "test-agent", 20000001)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false, want true")
	}
	if got.ID != 20000001 || got.RegionID != 10000001 {
		t.Errorf("got = %+v, want ID=20000001 RegionID=10000001", got)
	}
	if got.Name["en"] != "San Matar" {
		t.Errorf("Name[en] = %q, want San Matar", got.Name["en"])
	}
	if got.Position.X != 1.5 || got.Position.Y != 2.5 || got.Position.Z != 3.5 {
		t.Errorf("Position = %+v, want {1.5 2.5 3.5}", got.Position)
	}
	if len(got.SystemIDs) != 2 || got.SystemIDs[0] != 30000001 || got.SystemIDs[1] != 30000002 {
		t.Errorf("SystemIDs = %v, want [30000001 30000002]", got.SystemIDs)
	}
}

func TestConstellation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Constellation(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if ok {
		t.Error("Constellation ok = true, want false for an unknown ID")
	}
}

func TestConstellation_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "constellations", 20000001,
		`{"_key": 999, "name": {"en": "San Matar"}, "position": {"x": 0, "y": 0, "z": 0}, "regionID": 10000001, "solarSystemIDs": []}`)

	_, _, err := store.Constellation(context.Background(), "test-agent", 20000001)
	if err == nil {
		t.Fatal("Constellation returned nil error, want an error for a mismatched _key")
	}
}

func TestPutConstellation_ThenConstellation_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Constellation{ID: 20000099, RegionID: 10000001, Name: map[string]string{"en": "New Constellation"}, Position: Position{X: 1, Y: 2, Z: 3}, SystemIDs: []int64{30000099}}

	if err := store.PutConstellation(want); err != nil {
		t.Fatalf("PutConstellation returned error: %v", err)
	}

	got, ok, err := store.Constellation(context.Background(), "test-agent", 20000099)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false after PutConstellation, want true")
	}
	if got.ID != want.ID || got.RegionID != want.RegionID || got.Position != want.Position {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./sde/... -run TestConstellation -v`
Expected: FAIL to compile — `undefined: Constellation` / `undefined: Position`.

- [ ] **Step 7: Write the minimal implementation**

Create `sde/constellation.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Position is a shared 3D coordinate, reused by every geography resource.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Constellation is a minimal decode of an SDE constellation record -- EVE's
// geography level between Region and System.
type Constellation struct {
	ID        int64
	RegionID  int64
	Name      map[string]string
	Position  Position
	SystemIDs []int64
}

type constellationRecord struct {
	Key           int64             `json:"_key"`
	Name          map[string]string `json:"name"`
	Position      Position          `json:"position"`
	RegionID      int64             `json:"regionID"`
	SystemIDs     []int64           `json:"solarSystemIDs"`
}

// Constellation looks up a constellation by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) Constellation(ctx context.Context, userAgent string, id int64) (Constellation, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Constellation
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("constellations"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw constellationRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached constellation %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached constellation %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Constellation{ID: raw.Key, RegionID: raw.RegionID, Name: raw.Name, Position: raw.Position, SystemIDs: raw.SystemIDs}, true
		return nil
	})
	return out, ok, err
}

// PutConstellation writes c into the cache, keyed by c.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutConstellation(c Constellation) error {
	raw, err := json.Marshal(constellationRecord{Key: c.ID, Name: c.Name, Position: c.Position, RegionID: c.RegionID, SystemIDs: c.SystemIDs})
	if err != nil {
		return fmt.Errorf("marshal constellation %d: %w", c.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("constellations"))
		if err != nil {
			return fmt.Errorf("create constellations bucket: %w", err)
		}
		return b.Put(int64Key(c.ID), raw)
	})
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./sde/... -run TestConstellation -v`
Expected: PASS (4/4).

- [ ] **Step 9: Write the failing tests for `Graphic`**

Create `sde/graphic_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestGraphic_KnownID_ReturnsFile(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 10, `{"_key": 10, "graphicFile": "res:/dx9/model/worldobject/planet/moon.red"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 10)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.ID != 10 {
		t.Errorf("got.ID = %d, want 10", got.ID)
	}
	if got.GraphicFile == nil || *got.GraphicFile != "res:/dx9/model/worldobject/planet/moon.red" {
		t.Errorf("GraphicFile = %v, want \"res:/dx9/model/worldobject/planet/moon.red\"", got.GraphicFile)
	}
}

func TestGraphic_KnownID_ReturnsSOFFields(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 38,
		`{"_key": 38, "iconFolder": "res:/dx9/model/ship/caldari/frigate/cf1/icons", "sofFactionName": "caldaribase", "sofHullName": "cf1_t1", "sofRaceName": "caldari"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 38)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.GraphicFile != nil {
		t.Errorf("GraphicFile = %v, want nil (this record has none)", got.GraphicFile)
	}
	if got.IconFolder == nil || *got.IconFolder != "res:/dx9/model/ship/caldari/frigate/cf1/icons" {
		t.Errorf("IconFolder = %v, want the caldari frigate icons path", got.IconFolder)
	}
	if got.SofFactionName == nil || *got.SofFactionName != "caldaribase" {
		t.Errorf("SofFactionName = %v, want caldaribase", got.SofFactionName)
	}
	if got.SofHullName == nil || *got.SofHullName != "cf1_t1" {
		t.Errorf("SofHullName = %v, want cf1_t1", got.SofHullName)
	}
	if got.SofRaceName == nil || *got.SofRaceName != "caldari" {
		t.Errorf("SofRaceName = %v, want caldari", got.SofRaceName)
	}
}

func TestGraphic_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Graphic(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if ok {
		t.Error("Graphic ok = true, want false for an unknown ID")
	}
}

func TestGraphic_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 10, `{"_key": 999, "graphicFile": "res:/x"}`)

	_, _, err := store.Graphic(context.Background(), "test-agent", 10)
	if err == nil {
		t.Fatal("Graphic returned nil error, want an error for a mismatched _key")
	}
}

func TestPutGraphic_ThenGraphic_RoundTrips(t *testing.T) {
	store := seedStore(t)
	file := "res:/dx9/model/worldobject/newthing.red"
	race := "caldari"
	want := Graphic{ID: 99, GraphicFile: &file, SofRaceName: &race}

	if err := store.PutGraphic(want); err != nil {
		t.Fatalf("PutGraphic returned error: %v", err)
	}

	got, ok, err := store.Graphic(context.Background(), "test-agent", 99)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false after PutGraphic, want true")
	}
	if got.ID != want.ID || *got.GraphicFile != *want.GraphicFile || *got.SofRaceName != *want.SofRaceName {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 10: Run test to verify it fails**

Run: `go test ./sde/... -run TestGraphic -v`
Expected: FAIL to compile — `undefined: Graphic`.

- [ ] **Step 11: Write the minimal implementation**

Create `sde/graphic.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Graphic is a minimal decode of an SDE graphic record. CollisionFile and
// SofDNA are deliberately not modeled here -- confirmed absent from every
// one of the archive's 6051 graphics.jsonl records, not merely a sample
// coincidence.
type Graphic struct {
	ID             int64
	GraphicFile    *string
	IconFolder     *string
	SofFactionName *string
	SofHullName    *string
	SofRaceName    *string
}

type graphicRecord struct {
	Key            int64   `json:"_key"`
	GraphicFile    *string `json:"graphicFile,omitempty"`
	IconFolder     *string `json:"iconFolder,omitempty"`
	SofFactionName *string `json:"sofFactionName,omitempty"`
	SofHullName    *string `json:"sofHullName,omitempty"`
	SofRaceName    *string `json:"sofRaceName,omitempty"`
}

// Graphic looks up a graphic by ID. ok is false if the ID isn't in the cache.
func (s *Store) Graphic(ctx context.Context, userAgent string, id int64) (Graphic, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Graphic
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("graphics"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw graphicRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached graphic %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached graphic %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Graphic{
			ID: raw.Key, GraphicFile: raw.GraphicFile, IconFolder: raw.IconFolder,
			SofFactionName: raw.SofFactionName, SofHullName: raw.SofHullName, SofRaceName: raw.SofRaceName,
		}, true
		return nil
	})
	return out, ok, err
}

// PutGraphic writes g into the cache, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGraphic(g Graphic) error {
	raw, err := json.Marshal(graphicRecord{
		Key: g.ID, GraphicFile: g.GraphicFile, IconFolder: g.IconFolder,
		SofFactionName: g.SofFactionName, SofHullName: g.SofHullName, SofRaceName: g.SofRaceName,
	})
	if err != nil {
		return fmt.Errorf("marshal graphic %d: %w", g.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("graphics"))
		if err != nil {
			return fmt.Errorf("create graphics bucket: %w", err)
		}
		return b.Put(int64Key(g.ID), raw)
	})
}
```

- [ ] **Step 12: Run test to verify it passes**

Run: `go test ./sde/... -run TestGraphic -v`
Expected: PASS (5/5).

- [ ] **Step 13: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 14: Commit**

```bash
git add sde/region.go sde/region_test.go sde/constellation.go sde/constellation_test.go sde/graphic.go sde/graphic_test.go
git commit -m "feat: add Region/Constellation/Graphic accessors and Put* write-back to sde/"
```

---

### Task 2: `Category.GroupIDs` / `Group.TypeIDs` reverse-index fields + `PutCategory`/`PutGroup`

**Files:**
- Modify: `sde/category.go`
- Modify: `sde/category_test.go`
- Modify: `sde/group.go`
- Modify: `sde/group_test.go`

**Interfaces:**
- Consumes: same as Task 1.
- Produces (for Task 6's sync.go rewrite and Task 7's mcp/ migrations):
  - `Category` gains `GroupIDs []int64`; `func (s *Store) PutCategory(c Category) error`.
  - `Group` gains `TypeIDs []int64`; `func (s *Store) PutGroup(g Group) error`.
  - Bucket names `"categories"`, `"groups"` unchanged from the sde-infrastructure PR.

- [ ] **Step 1: Write the failing tests**

Modify `sde/category_test.go` — add after `TestCategory_MismatchedKey_ReturnsError`:
```go

func TestCategory_KnownID_ReturnsGroupIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "categories", 6,
		`{"_key": 6, "name": {"en": "Ship"}, "published": true, "groupIds": [25, 26]}`)

	got, ok, err := store.Category(context.Background(), "test-agent", 6)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false, want true")
	}
	if len(got.GroupIDs) != 2 || got.GroupIDs[0] != 25 || got.GroupIDs[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26]", got.GroupIDs)
	}
}

func TestPutCategory_ThenCategory_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Category{ID: 999, Name: map[string]string{"en": "New Category"}, Published: true, GroupIDs: []int64{111, 222}}

	if err := store.PutCategory(want); err != nil {
		t.Fatalf("PutCategory returned error: %v", err)
	}

	got, ok, err := store.Category(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false after PutCategory, want true")
	}
	if got.ID != want.ID || got.Name["en"] != want.Name["en"] || got.Published != want.Published {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.GroupIDs) != 2 || got.GroupIDs[0] != 111 || got.GroupIDs[1] != 222 {
		t.Errorf("GroupIDs = %v, want [111 222]", got.GroupIDs)
	}
}
```

Modify `sde/group_test.go` — add after `TestGroup_MismatchedKey_ReturnsError`:
```go

func TestGroup_KnownID_ReturnsTypeIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "groups", 25,
		`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true, "typeIds": [587, 588]}`)

	got, ok, err := store.Group(context.Background(), "test-agent", 25)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false, want true")
	}
	if len(got.TypeIDs) != 2 || got.TypeIDs[0] != 587 || got.TypeIDs[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588]", got.TypeIDs)
	}
}

func TestPutGroup_ThenGroup_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Group{ID: 999, CategoryID: 6, Name: map[string]string{"en": "New Group"}, Published: true, TypeIDs: []int64{111, 222}}

	if err := store.PutGroup(want); err != nil {
		t.Fatalf("PutGroup returned error: %v", err)
	}

	got, ok, err := store.Group(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false after PutGroup, want true")
	}
	if got.ID != want.ID || got.CategoryID != want.CategoryID || got.Name["en"] != want.Name["en"] || got.Published != want.Published {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.TypeIDs) != 2 || got.TypeIDs[0] != 111 || got.TypeIDs[1] != 222 {
		t.Errorf("TypeIDs = %v, want [111 222]", got.TypeIDs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run 'TestCategory_KnownID_ReturnsGroupIDs|TestPutCategory_ThenCategory_RoundTrips|TestGroup_KnownID_ReturnsTypeIDs|TestPutGroup_ThenGroup_RoundTrips' -v`
Expected: FAIL — `got.GroupIDs undefined` / `got.TypeIDs undefined` / `s.PutCategory undefined` / `s.PutGroup undefined`.

- [ ] **Step 3: Write the minimal implementation**

Modify `sde/category.go` — replace the whole file:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Category is a minimal decode of an SDE category record -- the top level
// of EVE's item hierarchy (e.g. "Ship", "Module").
type Category struct {
	ID        int64
	Name      map[string]string
	Published bool
	GroupIDs  []int64
}

type categoryRecord struct {
	Key       int64             `json:"_key"`
	Name      map[string]string `json:"name"`
	Published bool              `json:"published"`
	GroupIDs  []int64           `json:"groupIds"`
}

// Category looks up a category by ID. ok is false if the ID isn't in the
// cache -- either because the store has never synced successfully, or
// because no such category exists.
func (s *Store) Category(ctx context.Context, userAgent string, id int64) (Category, bool, error) {
	_ = s.ensureSynced(ctx, userAgent) // best-effort; see ensureSynced's doc comment

	var (
		out Category
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("categories"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw categoryRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached category %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached category %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Category{ID: raw.Key, Name: raw.Name, Published: raw.Published, GroupIDs: raw.GroupIDs}, true
		return nil
	})
	return out, ok, err
}

// PutCategory writes c into the cache, keyed by c.ID -- used by mcp/'s
// read-through fallback to cache a category fetched live from ESI.
func (s *Store) PutCategory(c Category) error {
	raw, err := json.Marshal(categoryRecord{Key: c.ID, Name: c.Name, Published: c.Published, GroupIDs: c.GroupIDs})
	if err != nil {
		return fmt.Errorf("marshal category %d: %w", c.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("categories"))
		if err != nil {
			return fmt.Errorf("create categories bucket: %w", err)
		}
		return b.Put(int64Key(c.ID), raw)
	})
}
```

Modify `sde/group.go` — replace the whole file:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Group is a minimal decode of an SDE group record -- EVE's item hierarchy
// level between Category and Type (e.g. "Frigate", within the "Ship"
// category).
type Group struct {
	ID         int64
	CategoryID int64
	Name       map[string]string
	Published  bool
	TypeIDs    []int64
}

type groupRecord struct {
	Key        int64             `json:"_key"`
	CategoryID int64             `json:"categoryID"`
	Name       map[string]string `json:"name"`
	Published  bool              `json:"published"`
	TypeIDs    []int64           `json:"typeIds"`
}

// Group looks up a group by ID. ok is false if the ID isn't in the cache.
func (s *Store) Group(ctx context.Context, userAgent string, id int64) (Group, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Group
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("groups"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw groupRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached group %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached group %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Group{ID: raw.Key, CategoryID: raw.CategoryID, Name: raw.Name, Published: raw.Published, TypeIDs: raw.TypeIDs}, true
		return nil
	})
	return out, ok, err
}

// PutGroup writes g into the cache, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGroup(g Group) error {
	raw, err := json.Marshal(groupRecord{Key: g.ID, CategoryID: g.CategoryID, Name: g.Name, Published: g.Published, TypeIDs: g.TypeIDs})
	if err != nil {
		return fmt.Errorf("marshal group %d: %w", g.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("groups"))
		if err != nil {
			return fmt.Errorf("create groups bucket: %w", err)
		}
		return b.Put(int64Key(g.ID), raw)
	})
}
```

Note: `categories`/`groups` records already in bbolt from a prior sde-infrastructure sync predate the `groupIds`/`typeIds` fields and won't have them — `json.Unmarshal` leaves `GroupIDs`/`TypeIDs` as `nil` for those until the next full sync (Task 6) rewrites every record. This is expected and harmless (a `nil` slice reads identically to an absent one), not a bug to work around.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -run TestCategory -v && go test ./sde/... -run TestGroup -v`
Expected: PASS (all `TestCategory_*` and `TestGroup_*` tests, including the pre-existing ones from the sde-infrastructure PR).

- [ ] **Step 5: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 6: Commit**

```bash
git add sde/category.go sde/category_test.go sde/group.go sde/group_test.go
git commit -m "feat: add GroupIDs/TypeIDs reverse-index fields and Put* to Category/Group"
```

---

### Task 3: `Type` expansion (dogma + full field set) + `PutType`

**Files:**
- Modify: `sde/type.go`
- Modify: `sde/type_test.go`

**Interfaces:**
- Consumes: same as Task 1/2.
- Produces (for Task 6 and Task 7):
  - `Type` gains `Description map[string]string`, `Capacity *float64`, `GraphicID *int64`,
    `IconID *int64`, `MarketGroupID *int64`, `Mass *float64`, `PortionSize *int64`,
    `Radius *float64`, `Volume *float64`, `DogmaAttributes []TypeDogmaAttribute`,
    `DogmaEffects []TypeDogmaEffect`.
  - `type TypeDogmaAttribute struct { AttributeID int64; Value float64 }` (mirrors
    `mcp/dogma.go`'s existing `DynamicItemAttribute` shape, as its own `sde`-owned type — `sde/`
    does not import `mcp/`).
  - `type TypeDogmaEffect struct { EffectID int64; IsDefault bool }` (mirrors `DynamicItemEffect`).
  - `func (s *Store) PutType(t Type) error`.

Verified real record shapes:
```json
// types.jsonl (Rifter, type 587)
{"_key": 587, "basePrice": 400000.0, "capacity": 140.0, "description": {"de": "...", "en": "The Rifter is fast..."}, "factionID": 500002, "graphicID": 46, "groupID": 25, "marketGroupID": 64, "mass": 1067000.0, "metaGroupID": 1, "metaLevel": 0, "name": {...}, "portionSize": 1, "published": true, "raceID": 2, "radius": 31.0, "shipTreeGroupID": 8, "soundID": 20078, "techLevel": 1, "volume": 27289.0}
// typeDogma.jsonl
{"_key": 18, "dogmaAttributes": [{"attributeID": 182, "value": 3386.0}, ...]}
// dogmaAttributes always present (>= 1 item); dogmaEffects optional, e.g.:
{"_key": 587, "dogmaAttributes": [...], "dogmaEffects": [{"effectID": 4, "isDefault": true}]}
```
Confirmed `repackagedVolume`/`packagedVolume` is absent from the archive entirely (zero hits across
the full 151MB file, including on the Rifter's own record) — not modeled on `Type` at all.
`description` (multilingual, same shape as `name`) is present on real records (confirmed on type
1282, "Basic Kinetic Coating", and on the Rifter above) — `Type` models it as `Description
map[string]string`, the same treatment as `Name`, so `get_universe_type`'s existing
ESI-sourced `Description string` field has full SDE coverage on the migrated path (Task 7).

- [ ] **Step 1: Write the failing tests**

Modify `sde/type_test.go` — add after `TestType_MismatchedKey_ReturnsError`:
```go

func TestType_KnownID_ReturnsFullFieldSetAndDogma(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 587, "capacity": 140.0, "description": {"en": "The Rifter is fast."}, "graphicID": 46, "groupID": 25, "iconID": 100, "marketGroupID": 64, "mass": 1067000.0, "name": {"en": "Rifter"}, "portionSize": 1, "published": true, "radius": 31.0, "volume": 27289.0, "dogmaAttributes": [{"attributeID": 182, "value": 3386.0}], "dogmaEffects": [{"effectID": 4, "isDefault": true}]}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.Description["en"] != "The Rifter is fast." {
		t.Errorf("Description[en] = %q, want %q", got.Description["en"], "The Rifter is fast.")
	}
	if got.Capacity == nil || *got.Capacity != 140.0 {
		t.Errorf("Capacity = %v, want 140.0", got.Capacity)
	}
	if got.GraphicID == nil || *got.GraphicID != 46 {
		t.Errorf("GraphicID = %v, want 46", got.GraphicID)
	}
	if got.IconID == nil || *got.IconID != 100 {
		t.Errorf("IconID = %v, want 100", got.IconID)
	}
	if got.MarketGroupID == nil || *got.MarketGroupID != 64 {
		t.Errorf("MarketGroupID = %v, want 64", got.MarketGroupID)
	}
	if got.Mass == nil || *got.Mass != 1067000.0 {
		t.Errorf("Mass = %v, want 1067000.0", got.Mass)
	}
	if got.PortionSize == nil || *got.PortionSize != 1 {
		t.Errorf("PortionSize = %v, want 1", got.PortionSize)
	}
	if got.Radius == nil || *got.Radius != 31.0 {
		t.Errorf("Radius = %v, want 31.0", got.Radius)
	}
	if got.Volume == nil || *got.Volume != 27289.0 {
		t.Errorf("Volume = %v, want 27289.0", got.Volume)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].AttributeID != 182 || got.DogmaAttributes[0].Value != 3386.0 {
		t.Errorf("DogmaAttributes = %+v, want one entry {AttributeID:182 Value:3386.0}", got.DogmaAttributes)
	}
	if len(got.DogmaEffects) != 1 || got.DogmaEffects[0].EffectID != 4 || !got.DogmaEffects[0].IsDefault {
		t.Errorf("DogmaEffects = %+v, want one entry {EffectID:4 IsDefault:true}", got.DogmaEffects)
	}
}

func TestType_NoDogmaEffects_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 588,
		`{"_key": 588, "groupID": 25, "name": {"en": "No-effects Type"}, "published": true, "dogmaAttributes": [{"attributeID": 1, "value": 1.0}]}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 588)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if len(got.DogmaEffects) != 0 {
		t.Errorf("DogmaEffects = %+v, want empty", got.DogmaEffects)
	}
}

func TestPutType_ThenType_RoundTrips(t *testing.T) {
	store := seedStore(t)
	mass := 1000.0
	want := Type{ID: 999, GroupID: 25, Name: map[string]string{"en": "New Type"}, Description: map[string]string{"en": "A new type."}, Published: true, Mass: &mass, DogmaAttributes: []TypeDogmaAttribute{{AttributeID: 1, Value: 2.0}}}

	if err := store.PutType(want); err != nil {
		t.Fatalf("PutType returned error: %v", err)
	}

	got, ok, err := store.Type(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false after PutType, want true")
	}
	if got.ID != want.ID || got.GroupID != want.GroupID || *got.Mass != *want.Mass || got.Description["en"] != want.Description["en"] {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0] != want.DogmaAttributes[0] {
		t.Errorf("DogmaAttributes = %+v, want %+v", got.DogmaAttributes, want.DogmaAttributes)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run 'TestType_KnownID_ReturnsFullFieldSetAndDogma|TestType_NoDogmaEffects_ReturnsEmptySlice|TestPutType_ThenType_RoundTrips' -v`
Expected: FAIL to compile — `got.Capacity undefined` and similar for every new field, `undefined: TypeDogmaAttribute`, `s.PutType undefined`.

- [ ] **Step 3: Write the minimal implementation**

Modify `sde/type.go` — replace the whole file:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// TypeDogmaAttribute mirrors mcp/dogma.go's DynamicItemAttribute shape, as
// its own sde-owned type -- sde/ does not import mcp/.
type TypeDogmaAttribute struct {
	AttributeID int64
	Value       float64
}

// TypeDogmaEffect mirrors mcp/dogma.go's DynamicItemEffect shape.
type TypeDogmaEffect struct {
	EffectID  int64
	IsDefault bool
}

// Type is a minimal decode of an SDE type record -- the leaf level of
// EVE's item hierarchy (e.g. "Rifter", within the "Frigate" group).
// PackagedVolume is deliberately not modeled here -- confirmed absent from
// the SDE archive entirely (verified against a real ship's record, which
// unambiguously has a non-null packaged_volume on ESI).
type Type struct {
	ID              int64
	GroupID         int64
	Name            map[string]string
	Description     map[string]string
	Published       bool
	Capacity        *float64
	GraphicID       *int64
	IconID          *int64
	MarketGroupID   *int64
	Mass            *float64
	PortionSize     *int64
	Radius          *float64
	Volume          *float64
	DogmaAttributes []TypeDogmaAttribute
	DogmaEffects    []TypeDogmaEffect
}

type typeDogmaAttributeRecord struct {
	AttributeID int64   `json:"attributeID"`
	Value       float64 `json:"value"`
}

type typeDogmaEffectRecord struct {
	EffectID  int64 `json:"effectID"`
	IsDefault bool  `json:"isDefault"`
}

type typeRecord struct {
	Key             int64                      `json:"_key"`
	GroupID         int64                      `json:"groupID"`
	Name            map[string]string          `json:"name"`
	Description     map[string]string          `json:"description"`
	Published       bool                       `json:"published"`
	Capacity        *float64                   `json:"capacity,omitempty"`
	GraphicID       *int64                     `json:"graphicID,omitempty"`
	IconID          *int64                     `json:"iconID,omitempty"`
	MarketGroupID   *int64                     `json:"marketGroupID,omitempty"`
	Mass            *float64                   `json:"mass,omitempty"`
	PortionSize     *int64                     `json:"portionSize,omitempty"`
	Radius          *float64                   `json:"radius,omitempty"`
	Volume          *float64                   `json:"volume,omitempty"`
	DogmaAttributes []typeDogmaAttributeRecord `json:"dogmaAttributes,omitempty"`
	DogmaEffects    []typeDogmaEffectRecord    `json:"dogmaEffects,omitempty"`
}

// Type looks up a type by ID. ok is false if the ID isn't in the cache.
func (s *Store) Type(ctx context.Context, userAgent string, id int64) (Type, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Type
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("types"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw typeRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached type %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached type %d has mismatched key %d", id, raw.Key)
		}
		out = Type{
			ID: raw.Key, GroupID: raw.GroupID, Name: raw.Name, Description: raw.Description, Published: raw.Published,
			Capacity: raw.Capacity, GraphicID: raw.GraphicID, IconID: raw.IconID, MarketGroupID: raw.MarketGroupID,
			Mass: raw.Mass, PortionSize: raw.PortionSize, Radius: raw.Radius, Volume: raw.Volume,
		}
		if len(raw.DogmaAttributes) > 0 {
			out.DogmaAttributes = make([]TypeDogmaAttribute, len(raw.DogmaAttributes))
			for i, a := range raw.DogmaAttributes {
				out.DogmaAttributes[i] = TypeDogmaAttribute{AttributeID: a.AttributeID, Value: a.Value}
			}
		}
		if len(raw.DogmaEffects) > 0 {
			out.DogmaEffects = make([]TypeDogmaEffect, len(raw.DogmaEffects))
			for i, e := range raw.DogmaEffects {
				out.DogmaEffects[i] = TypeDogmaEffect{EffectID: e.EffectID, IsDefault: e.IsDefault}
			}
		}
		ok = true
		return nil
	})
	return out, ok, err
}

// PutType writes t into the cache, keyed by t.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutType(t Type) error {
	raw := typeRecord{
		Key: t.ID, GroupID: t.GroupID, Name: t.Name, Description: t.Description, Published: t.Published,
		Capacity: t.Capacity, GraphicID: t.GraphicID, IconID: t.IconID, MarketGroupID: t.MarketGroupID,
		Mass: t.Mass, PortionSize: t.PortionSize, Radius: t.Radius, Volume: t.Volume,
	}
	if len(t.DogmaAttributes) > 0 {
		raw.DogmaAttributes = make([]typeDogmaAttributeRecord, len(t.DogmaAttributes))
		for i, a := range t.DogmaAttributes {
			raw.DogmaAttributes[i] = typeDogmaAttributeRecord{AttributeID: a.AttributeID, Value: a.Value}
		}
	}
	if len(t.DogmaEffects) > 0 {
		raw.DogmaEffects = make([]typeDogmaEffectRecord, len(t.DogmaEffects))
		for i, e := range t.DogmaEffects {
			raw.DogmaEffects[i] = typeDogmaEffectRecord{EffectID: e.EffectID, IsDefault: e.IsDefault}
		}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal type %d: %w", t.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("types"))
		if err != nil {
			return fmt.Errorf("create types bucket: %w", err)
		}
		return b.Put(int64Key(t.ID), data)
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -run TestType -v`
Expected: PASS (all `TestType_*` tests, including the pre-existing ones).

- [ ] **Step 5: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 6: Commit**

```bash
git add sde/type.go sde/type_test.go
git commit -m "feat: expand Type with full field set, dogma attributes/effects, and Put*"
```

---

### Task 4: `sde/system.go` (with `StationIDs` as a stored-but-not-raw field)

**Files:**
- Create: `sde/system.go`, `sde/system_test.go`

**Interfaces:**
- Consumes: same as Task 1.
- Produces (for Task 6 and Task 8):
  - `type System struct { ID int64; ConstellationID int64; Name map[string]string; Position Position; SecurityStatus float64; SecurityClass *string; StarID *int64; StargateIDs []int64; StationIDs []int64 }`
  - `func (s *Store) System(ctx context.Context, userAgent string, id int64) (System, bool, error)`
  - `func (s *Store) PutSystem(sys System) error`
  - Bucket name `"systems"`.

Verified real record shape (`mapSolarSystems.jsonl`):
```json
{"_key": 30000001, "border": true, "constellationID": 20000001, "hub": true, "international": true, "luminosity": 0.01575, "name": {"en": "Tanoo", ...}, "planetIDs": [...], "position": {...}, "position2D": {...}, "radius": 1323338301440.0, "regionID": 10000001, "regional": true, "securityClass": "B", "securityStatus": 0.858324, "starID": 40000001, "stargateIDs": [50000056, 50000057, 50000058]}
```
Confirmed via `sde.riftforeve.online`'s schema docs: **no `stationIDs` field exists on this
record at all** (required fields: `_key`, `constellationID`, `name`, `position`, `radius`,
`regionID`, `securityStatus`; optional: `stargateIDs`, `planetIDs`, `starID`, plus several boolean/
classification flags not modeled here since `UniverseSystem` never exposed them). `StationIDs` is
therefore a field this struct stores but the raw JSONL line never carries — Task 6's sync writer
augments each record with a reverse index built from a slim pass over `npcStations.jsonl` before
writing it. A read-through-fetched `System` (via `PutSystem`) needs no such augmentation: ESI's
`UniverseSystemDetail` already returns `Stations` directly in the same response, so the caller
(Task 8) just passes it straight through.

- [ ] **Step 1: Write the failing tests**

Create `sde/system_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestSystem_KnownID_ReturnsAugmentedStationIDs(t *testing.T) {
	store := seedStore(t)
	starID := int64(40000001)
	secClass := "B"
	// Simulates what Task 6's sync writer produces: the raw mapSolarSystems.jsonl
	// fields plus an augmented "stationIds" key the source file never has.
	putRecord(t, store, "systems", 30000001,
		`{"_key": 30000001, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.858324, "securityClass": "B", "starID": 40000001, "stargateIDs": [50000056, 50000057], "stationIds": [60000001, 60000002]}`)

	got, ok, err := store.System(context.Background(), "test-agent", 30000001)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.ID != 30000001 || got.ConstellationID != 20000001 {
		t.Errorf("got = %+v, want ID=30000001 ConstellationID=20000001", got)
	}
	if got.Name["en"] != "Tanoo" {
		t.Errorf("Name[en] = %q, want Tanoo", got.Name["en"])
	}
	if got.Position != (Position{X: 1, Y: 2, Z: 3}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
	if got.SecurityStatus != 0.858324 {
		t.Errorf("SecurityStatus = %v, want 0.858324", got.SecurityStatus)
	}
	if got.SecurityClass == nil || *got.SecurityClass != secClass {
		t.Errorf("SecurityClass = %v, want %q", got.SecurityClass, secClass)
	}
	if got.StarID == nil || *got.StarID != starID {
		t.Errorf("StarID = %v, want %d", got.StarID, starID)
	}
	if len(got.StargateIDs) != 2 || got.StargateIDs[0] != 50000056 {
		t.Errorf("StargateIDs = %v, want [50000056 50000057]", got.StargateIDs)
	}
	if len(got.StationIDs) != 2 || got.StationIDs[0] != 60000001 || got.StationIDs[1] != 60000002 {
		t.Errorf("StationIDs = %v, want [60000001 60000002]", got.StationIDs)
	}
}

func TestSystem_NoStationsOrStargates_ReturnsEmpty(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000002,
		`{"_key": 30000002, "constellationID": 20000001, "name": {"en": "Empty System"}, "position": {"x": 0, "y": 0, "z": 0}, "securityStatus": 0.5}`)

	got, ok, err := store.System(context.Background(), "test-agent", 30000002)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.SecurityClass != nil {
		t.Errorf("SecurityClass = %v, want nil", got.SecurityClass)
	}
	if got.StarID != nil {
		t.Errorf("StarID = %v, want nil", got.StarID)
	}
	if len(got.StargateIDs) != 0 || len(got.StationIDs) != 0 {
		t.Errorf("StargateIDs/StationIDs = %v/%v, want both empty", got.StargateIDs, got.StationIDs)
	}
}

func TestSystem_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.System(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if ok {
		t.Error("System ok = true, want false for an unknown ID")
	}
}

func TestSystem_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000001,
		`{"_key": 999, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 0, "y": 0, "z": 0}, "securityStatus": 0.5}`)

	_, _, err := store.System(context.Background(), "test-agent", 30000001)
	if err == nil {
		t.Fatal("System returned nil error, want an error for a mismatched _key")
	}
}

func TestPutSystem_ThenSystem_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := System{ID: 30000099, ConstellationID: 20000001, Name: map[string]string{"en": "New System"}, Position: Position{X: 9, Y: 8, Z: 7}, SecurityStatus: 0.9, StationIDs: []int64{60000099}}

	if err := store.PutSystem(want); err != nil {
		t.Fatalf("PutSystem returned error: %v", err)
	}

	got, ok, err := store.System(context.Background(), "test-agent", 30000099)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false after PutSystem, want true")
	}
	if got.ID != want.ID || got.SecurityStatus != want.SecurityStatus {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.StationIDs) != 1 || got.StationIDs[0] != 60000099 {
		t.Errorf("StationIDs = %v, want [60000099]", got.StationIDs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run TestSystem -v`
Expected: FAIL to compile — `undefined: System`.

- [ ] **Step 3: Write the minimal implementation**

Create `sde/system.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// System is a minimal decode of an SDE solar system record -- EVE's
// geography level between Constellation and celestial bodies. StationIDs is
// not present on the raw mapSolarSystems.jsonl record at all -- it's
// augmented at sync time from a reverse index over npcStations.jsonl (see
// sync.go), or supplied directly by a read-through ESI fetch (ESI's system
// detail response already includes it).
type System struct {
	ID              int64
	ConstellationID int64
	Name            map[string]string
	Position        Position
	SecurityStatus  float64
	SecurityClass   *string
	StarID          *int64
	StargateIDs     []int64
	StationIDs      []int64
}

type systemRecord struct {
	Key             int64             `json:"_key"`
	ConstellationID int64             `json:"constellationID"`
	Name            map[string]string `json:"name"`
	Position        Position          `json:"position"`
	SecurityStatus  float64           `json:"securityStatus"`
	SecurityClass   *string           `json:"securityClass,omitempty"`
	StarID          *int64            `json:"starID,omitempty"`
	StargateIDs     []int64           `json:"stargateIds,omitempty"`
	StationIDs      []int64           `json:"stationIds,omitempty"`
}

// System looks up a solar system by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) System(ctx context.Context, userAgent string, id int64) (System, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out System
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("systems"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw systemRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached system %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached system %d has mismatched key %d", id, raw.Key)
		}
		out, ok = System{
			ID: raw.Key, ConstellationID: raw.ConstellationID, Name: raw.Name, Position: raw.Position,
			SecurityStatus: raw.SecurityStatus, SecurityClass: raw.SecurityClass, StarID: raw.StarID,
			StargateIDs: raw.StargateIDs, StationIDs: raw.StationIDs,
		}, true
		return nil
	})
	return out, ok, err
}

// PutSystem writes sys into the cache, keyed by sys.ID -- used by mcp/'s
// read-through fallback. Unlike the bulk sync path, a read-through fetch
// gets StationIDs directly from ESI's response, no augmentation needed.
func (s *Store) PutSystem(sys System) error {
	raw := systemRecord{
		Key: sys.ID, ConstellationID: sys.ConstellationID, Name: sys.Name, Position: sys.Position,
		SecurityStatus: sys.SecurityStatus, SecurityClass: sys.SecurityClass, StarID: sys.StarID,
		StargateIDs: sys.StargateIDs, StationIDs: sys.StationIDs,
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal system %d: %w", sys.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("systems"))
		if err != nil {
			return fmt.Errorf("create systems bucket: %w", err)
		}
		return b.Put(int64Key(sys.ID), data)
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -run TestSystem -v`
Expected: PASS (5/5).

- [ ] **Step 5: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 6: Commit**

```bash
git add sde/system.go sde/system_test.go
git commit -m "feat: add System accessor to sde/ with augmented StationIDs"
```

---

### Task 5: `sde/faction.go`, `sde/ancestry.go`, `sde/bloodline.go`, `sde/race.go`

**Files:**
- Create: `sde/faction.go`, `sde/faction_test.go`
- Create: `sde/ancestry.go`, `sde/ancestry_test.go`
- Create: `sde/bloodline.go`, `sde/bloodline_test.go`
- Create: `sde/race.go`, `sde/race_test.go`

**Interfaces:**
- Consumes: same as Task 1.
- Produces (for Task 6 and Task 9):
  - `type Faction struct { ID int64; Name map[string]string; Description map[string]string; IsUnique bool; SizeFactor float64; CorporationID *int64; MilitiaCorporationID *int64; SolarSystemID *int64; StationCount int64; StationSystemCount int64 }`
  - `func (s *Store) Factions(ctx context.Context, userAgent string) ([]Faction, error)`
  - `type Ancestry struct { ID int64; BloodlineID int64; Description map[string]string; Name map[string]string; IconID *int64; ShortDescription *string }`
  - `func (s *Store) Ancestries(ctx context.Context, userAgent string) ([]Ancestry, error)`
  - `type Bloodline struct { ID int64; Charisma int64; CorporationID int64; Description map[string]string; Intelligence int64; Memory int64; Name map[string]string; Perception int64; RaceID int64; ShipTypeID *int64; Willpower int64 }`
  - `func (s *Store) Bloodlines(ctx context.Context, userAgent string) ([]Bloodline, error)`
  - `type Race struct { ID int64; Description map[string]string; Name map[string]string }`
  - `func (s *Store) Races(ctx context.Context, userAgent string) ([]Race, error)`
  - Bucket names `"factions"`, `"ancestries"`, `"bloodlines"`, `"races"`.
  - None of these four get a `Put*` method — there is no `get_universe_faction`/`_ancestry`/
    `_bloodline`/`_race` tool, only whole-collection `list_` tools (per the design spec).

Verified real record shapes:
```json
// factions.jsonl
{"_key": 500001, "corporationID": 1000035, "description": {"en": "..."}, "militiaCorporationID": 1000180, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "solarSystemID": 30000145, "uniqueName": true}
// ancestries.jsonl
{"_key": 1, "bloodlineID": 5, "description": {"en": "..."}, "iconID": 1641, "name": {"en": "Liberal Holders"}, "shortDescription": "Progressive members of the upper class who have rejected their traditional ways."}
// bloodlines.jsonl
{"_key": 1, "charisma": 6, "corporationID": 1000006, "description": {"en": "..."}, "iconID": 1633, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "willpower": 5}
// races.jsonl
{"_key": 1, "description": {"en": "..."}, "iconID": 1439, "name": {"en": "Caldari"}, "shipTypeID": 601, "skills": [...]}
```
`shortDescription` on ancestries is a **plain string**, not multilingual — confirmed from the real
sample, matching the existing `UniverseAncestry.ShortDescription` shape exactly. `uniqueName`
(boolean) is the SDE field backing the existing `UniverseFaction.IsUnique` field.

**Verify before finalizing this task**: the sampled `bloodlines.jsonl` record above has no
`shipTypeID` field, but the existing ESI-sourced `UniverseBloodline.ShipTypeID` is a non-optional
`int64`. Check `sde.riftforeve.online`'s `bloodlines` schema page to confirm whether `shipTypeID`
is actually always present (sample coincidence) or genuinely optional/rare before finalizing —
this task models it as `*int64` (safe either way); if the schema confirms it's always present,
simplify to non-pointer `int64` instead and drop the corresponding "missing" test case below.

`Faction.StationCount`/`StationSystemCount` are **not** decoded from the raw `factions.jsonl`
record at all here — Task 6's sync writer computes and injects them via the 3-hop join
(`npcStations.jsonl` → `npcCorporations.jsonl` → aggregate), the same "stored but not raw" pattern
`System.StationIDs` used in Task 4.

- [ ] **Step 1: Write the failing tests for `Faction`**

Create `sde/faction_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestFactions_ReturnsAllWithAugmentedCounts(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "factions", 500001,
		`{"_key": 500001, "corporationID": 1000035, "description": {"en": "The Caldari State..."}, "militiaCorporationID": 1000180, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "solarSystemID": 30000145, "uniqueName": true, "stationCount": 120, "stationSystemCount": 45}`)

	got, err := store.Factions(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Factions = %+v, want one entry", got)
	}
	f := got[0]
	if f.ID != 500001 || !f.IsUnique || f.SizeFactor != 5.0 {
		t.Errorf("got = %+v, want ID=500001 IsUnique=true SizeFactor=5.0", f)
	}
	if f.Name["en"] != "Caldari State" {
		t.Errorf("Name[en] = %q, want Caldari State", f.Name["en"])
	}
	if f.CorporationID == nil || *f.CorporationID != 1000035 {
		t.Errorf("CorporationID = %v, want 1000035", f.CorporationID)
	}
	if f.MilitiaCorporationID == nil || *f.MilitiaCorporationID != 1000180 {
		t.Errorf("MilitiaCorporationID = %v, want 1000180", f.MilitiaCorporationID)
	}
	if f.SolarSystemID == nil || *f.SolarSystemID != 30000145 {
		t.Errorf("SolarSystemID = %v, want 30000145", f.SolarSystemID)
	}
	if f.StationCount != 120 || f.StationSystemCount != 45 {
		t.Errorf("StationCount/StationSystemCount = %d/%d, want 120/45", f.StationCount, f.StationSystemCount)
	}
}

func TestFactions_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Factions(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Factions = %+v, want empty", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run TestFactions -v`
Expected: FAIL to compile — `undefined: Faction` / `s.Factions undefined`.

- [ ] **Step 3: Write the minimal implementation**

Create `sde/faction.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Faction is a minimal decode of an SDE faction record. StationCount and
// StationSystemCount are not present on the raw factions.jsonl record at
// all -- Task 6's sync writer computes them via a 3-hop join (npcStations
// -> npcCorporations -> aggregate per faction) and injects them before
// writing.
type Faction struct {
	ID                   int64
	Name                 map[string]string
	Description          map[string]string
	IsUnique             bool
	SizeFactor           float64
	CorporationID        *int64
	MilitiaCorporationID *int64
	SolarSystemID        *int64
	StationCount         int64
	StationSystemCount   int64
}

type factionRecord struct {
	Key                  int64             `json:"_key"`
	Name                 map[string]string `json:"name"`
	Description          map[string]string `json:"description"`
	IsUnique             bool              `json:"uniqueName"`
	SizeFactor           float64           `json:"sizeFactor"`
	CorporationID        *int64            `json:"corporationID,omitempty"`
	MilitiaCorporationID *int64            `json:"militiaCorporationID,omitempty"`
	SolarSystemID        *int64            `json:"solarSystemID,omitempty"`
	StationCount         int64             `json:"stationCount"`
	StationSystemCount   int64             `json:"stationSystemCount"`
}

// Factions returns every cached faction. Returns an empty slice, not an
// error, if the store has never synced successfully -- mcp/'s caller
// treats an empty result as a signal to read-through to a live ESI list
// call.
func (s *Store) Factions(ctx context.Context, userAgent string) ([]Faction, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var out []Faction
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("factions"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw factionRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached faction: %w", err)
			}
			out = append(out, Faction{
				ID: raw.Key, Name: raw.Name, Description: raw.Description, IsUnique: raw.IsUnique,
				SizeFactor: raw.SizeFactor, CorporationID: raw.CorporationID, MilitiaCorporationID: raw.MilitiaCorporationID,
				SolarSystemID: raw.SolarSystemID, StationCount: raw.StationCount, StationSystemCount: raw.StationSystemCount,
			})
			return nil
		})
	})
	return out, err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -run TestFactions -v`
Expected: PASS (2/2).

- [ ] **Step 5: Write the failing tests for `Ancestry`**

Create `sde/ancestry_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestAncestries_ReturnsAll(t *testing.T) {
	store := seedStore(t)
	shortDesc := "Progressive members of the upper class."
	putRecord(t, store, "ancestries", 1,
		`{"_key": 1, "bloodlineID": 5, "description": {"en": "Holders are..."}, "iconID": 1641, "name": {"en": "Liberal Holders"}, "shortDescription": "Progressive members of the upper class."}`)

	got, err := store.Ancestries(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Ancestries = %+v, want one entry", got)
	}
	a := got[0]
	if a.ID != 1 || a.BloodlineID != 5 {
		t.Errorf("got = %+v, want ID=1 BloodlineID=5", a)
	}
	if a.Name["en"] != "Liberal Holders" {
		t.Errorf("Name[en] = %q, want \"Liberal Holders\"", a.Name["en"])
	}
	if a.IconID == nil || *a.IconID != 1641 {
		t.Errorf("IconID = %v, want 1641", a.IconID)
	}
	if a.ShortDescription == nil || *a.ShortDescription != shortDesc {
		t.Errorf("ShortDescription = %v, want %q", a.ShortDescription, shortDesc)
	}
}

func TestAncestries_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Ancestries(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Ancestries = %+v, want empty", got)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./sde/... -run TestAncestries -v`
Expected: FAIL to compile — `undefined: Ancestry`.

- [ ] **Step 7: Write the minimal implementation**

Create `sde/ancestry.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Ancestry is a minimal decode of an SDE ancestry record.
type Ancestry struct {
	ID               int64
	BloodlineID      int64
	Description      map[string]string
	Name             map[string]string
	IconID           *int64
	ShortDescription *string
}

type ancestryRecord struct {
	Key              int64             `json:"_key"`
	BloodlineID      int64             `json:"bloodlineID"`
	Description      map[string]string `json:"description"`
	Name             map[string]string `json:"name"`
	IconID           *int64            `json:"iconID,omitempty"`
	ShortDescription *string           `json:"shortDescription,omitempty"`
}

// Ancestries returns every cached ancestry. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Ancestries(ctx context.Context, userAgent string) ([]Ancestry, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var out []Ancestry
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("ancestries"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw ancestryRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached ancestry: %w", err)
			}
			out = append(out, Ancestry{
				ID: raw.Key, BloodlineID: raw.BloodlineID, Description: raw.Description,
				Name: raw.Name, IconID: raw.IconID, ShortDescription: raw.ShortDescription,
			})
			return nil
		})
	})
	return out, err
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./sde/... -run TestAncestries -v`
Expected: PASS (2/2).

- [ ] **Step 9: Write the failing tests for `Bloodline`**

Create `sde/bloodline_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestBloodlines_ReturnsAll(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "bloodlines", 1,
		`{"_key": 1, "charisma": 6, "corporationID": 1000006, "description": {"en": "The Deteis are..."}, "iconID": 1633, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "shipTypeID": 601, "willpower": 5}`)

	got, err := store.Bloodlines(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Bloodlines returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Bloodlines = %+v, want one entry", got)
	}
	b := got[0]
	if b.ID != 1 || b.Charisma != 6 || b.CorporationID != 1000006 {
		t.Errorf("got = %+v, want ID=1 Charisma=6 CorporationID=1000006", b)
	}
	if b.Name["en"] != "Deteis" {
		t.Errorf("Name[en] = %q, want Deteis", b.Name["en"])
	}
	if b.Intelligence != 7 || b.Memory != 7 || b.Perception != 5 || b.Willpower != 5 || b.RaceID != 1 {
		t.Errorf("got = %+v, want Intelligence=7 Memory=7 Perception=5 Willpower=5 RaceID=1", b)
	}
	if b.ShipTypeID == nil || *b.ShipTypeID != 601 {
		t.Errorf("ShipTypeID = %v, want 601", b.ShipTypeID)
	}
}

func TestBloodlines_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Bloodlines(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Bloodlines returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Bloodlines = %+v, want empty", got)
	}
}
```

- [ ] **Step 10: Run test to verify it fails**

Run: `go test ./sde/... -run TestBloodlines -v`
Expected: FAIL to compile — `undefined: Bloodline`.

- [ ] **Step 11: Write the minimal implementation**

Create `sde/bloodline.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Bloodline is a minimal decode of an SDE bloodline record.
type Bloodline struct {
	ID            int64
	Charisma      int64
	CorporationID int64
	Description   map[string]string
	Intelligence  int64
	Memory        int64
	Name          map[string]string
	Perception    int64
	RaceID        int64
	ShipTypeID    *int64
	Willpower     int64
}

type bloodlineRecord struct {
	Key           int64             `json:"_key"`
	Charisma      int64             `json:"charisma"`
	CorporationID int64             `json:"corporationID"`
	Description   map[string]string `json:"description"`
	Intelligence  int64             `json:"intelligence"`
	Memory        int64             `json:"memory"`
	Name          map[string]string `json:"name"`
	Perception    int64             `json:"perception"`
	RaceID        int64             `json:"raceID"`
	ShipTypeID    *int64            `json:"shipTypeID,omitempty"`
	Willpower     int64             `json:"willpower"`
}

// Bloodlines returns every cached bloodline. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Bloodlines(ctx context.Context, userAgent string) ([]Bloodline, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var out []Bloodline
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("bloodlines"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw bloodlineRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached bloodline: %w", err)
			}
			out = append(out, Bloodline{
				ID: raw.Key, Charisma: raw.Charisma, CorporationID: raw.CorporationID, Description: raw.Description,
				Intelligence: raw.Intelligence, Memory: raw.Memory, Name: raw.Name, Perception: raw.Perception,
				RaceID: raw.RaceID, ShipTypeID: raw.ShipTypeID, Willpower: raw.Willpower,
			})
			return nil
		})
	})
	return out, err
}
```

- [ ] **Step 12: Run test to verify it passes**

Run: `go test ./sde/... -run TestBloodlines -v`
Expected: PASS (2/2).

- [ ] **Step 13: Write the failing tests for `Race`**

Create `sde/race_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestRaces_ReturnsAll(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 1, "description": {"en": "Founded on the tenets..."}, "name": {"en": "Caldari"}}`)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Races = %+v, want one entry", got)
	}
	r := got[0]
	if r.ID != 1 {
		t.Errorf("got.ID = %d, want 1", r.ID)
	}
	if r.Name["en"] != "Caldari" {
		t.Errorf("Name[en] = %q, want Caldari", r.Name["en"])
	}
}

func TestRaces_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Races = %+v, want empty", got)
	}
}
```

- [ ] **Step 14: Run test to verify it fails**

Run: `go test ./sde/... -run TestRaces -v`
Expected: FAIL to compile — `undefined: Race`.

- [ ] **Step 15: Write the minimal implementation**

Create `sde/race.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Race is a minimal decode of an SDE race record. AllianceID is
// deliberately not modeled -- confirmed absent from races.jsonl, and there
// is no alliances.jsonl anywhere in the SDE archive.
type Race struct {
	ID          int64
	Description map[string]string
	Name        map[string]string
}

type raceRecord struct {
	Key         int64             `json:"_key"`
	Description map[string]string `json:"description"`
	Name        map[string]string `json:"name"`
}

// Races returns every cached race. Returns an empty slice, not an error, if
// the store has never synced successfully.
func (s *Store) Races(ctx context.Context, userAgent string) ([]Race, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var out []Race
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("races"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw raceRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached race: %w", err)
			}
			out = append(out, Race{ID: raw.Key, Description: raw.Description, Name: raw.Name})
			return nil
		})
	})
	return out, err
}
```

- [ ] **Step 16: Run test to verify it passes**

Run: `go test ./sde/... -run TestRaces -v`
Expected: PASS (2/2).

- [ ] **Step 17: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 18: Commit**

```bash
git add sde/faction.go sde/faction_test.go sde/ancestry.go sde/ancestry_test.go sde/bloodline.go sde/bloodline_test.go sde/race.go sde/race_test.go
git commit -m "feat: add Faction/Ancestry/Bloodline/Race list accessors to sde/"
```

---

### Task 6: `sync.go` two-phase rewrite (the riskiest task — give this its own focused review)

**Files:**
- Modify: `sde/sync.go`
- Modify: `sde/sync_test.go`

**Interfaces:**
- Consumes: `typeRecord`, `groupRecord`, `categoryRecord`, `systemRecord`, `factionRecord` (all
  unexported, defined in Tasks 2/3/4/5, same package — `sync.go` references them directly),
  `int64Key`, `Store.db`, `Store.indexURL`, `Store.syncOnce`, `Store.syncErr`.
- Produces: an updated `(s *Store) sync(ctx, userAgent) error` that loads 13 source files (up from
  3) in dependency order, writing 11 buckets (`categories`, `groups`, `types`, `regions`,
  `constellations`, `systems`, `ancestries`, `bloodlines`, `races`, `factions`, `graphics`) plus
  `_meta`, all inside one atomic `db.Update` — same external signature and atomicity contract as
  before, so Tasks 1–5's `ensureSynced`-calling accessors need no changes.

The original `syncedFiles` map (independent per-file passthrough) is replaced by an explicit,
ordered pipeline — `groups.jsonl` needs `types.jsonl`'s data before it can be written,
`categories.jsonl` needs `groups.jsonl`'s, `mapSolarSystems.jsonl` needs a slim `npcStations.jsonl`
pass, and `factions.jsonl` needs both `npcStations.jsonl` and `npcCorporations.jsonl`. `loadResource`
(the existing simple passthrough helper) stays, used for the 6 resources needing no augmentation.

- [ ] **Step 1: Write the failing tests — replace the fixture-zip builders and existing assertions**

Modify `sde/sync_test.go` — replace the whole file:
```go
package sde

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"go.etcd.io/bbolt"
)

const testUserAgent = "neat-test/0.1"

// fullFixtureFiles returns the complete 13-file source set sync() now
// requires, with enough cross-linked data (group 25 belongs to category 6
// and contains type 587; station 60000001 is in system 30000001, owned by
// corp 1000035, which belongs to faction 500001) to exercise every
// augmentation this task adds: Group.TypeIDs, Category.GroupIDs,
// System.StationIDs, Faction.StationCount/StationSystemCount.
func fullFixtureFiles(shipCategoryName string) map[string][]string {
	return map[string][]string{
		"categories.jsonl": {
			`{"_key": 6, "name": {"en": "` + shipCategoryName + `"}, "published": true}`,
		},
		"groups.jsonl": {
			`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true}`,
		},
		"types.jsonl": {
			`{"_key": 587, "groupID": 25, "name": {"en": "Rifter"}, "published": true}`,
		},
		"typeDogma.jsonl": {
			`{"_key": 587, "dogmaAttributes": [{"attributeID": 182, "value": 3386.0}], "dogmaEffects": [{"effectID": 4, "isDefault": true}]}`,
		},
		"npcCorporations.jsonl": {
			`{"_key": 1000035, "factionID": 500001}`,
		},
		"npcStations.jsonl": {
			`{"_key": 60000001, "solarSystemID": 30000001, "ownerID": 1000035}`,
		},
		"mapSolarSystems.jsonl": {
			`{"_key": 30000001, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.85}`,
		},
		"factions.jsonl": {
			`{"_key": 500001, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "uniqueName": true}`,
		},
		"mapRegions.jsonl": {
			`{"_key": 10000001, "constellationIDs": [20000001], "name": {"en": "Derelik"}}`,
		},
		"mapConstellations.jsonl": {
			`{"_key": 20000001, "name": {"en": "San Matar"}, "position": {"x": 0, "y": 0, "z": 0}, "regionID": 10000001, "solarSystemIDs": [30000001]}`,
		},
		"ancestries.jsonl": {
			`{"_key": 1, "bloodlineID": 5, "description": {"en": "..."}, "name": {"en": "Liberal Holders"}}`,
		},
		"bloodlines.jsonl": {
			`{"_key": 1, "charisma": 6, "corporationID": 1000006, "description": {"en": "..."}, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "willpower": 5}`,
		},
		"races.jsonl": {
			`{"_key": 1, "description": {"en": "..."}, "name": {"en": "Caldari"}}`,
		},
		"graphics.jsonl": {
			`{"_key": 10, "graphicFile": "res:/x"}`,
		},
	}
}

func buildFixtureZip(t *testing.T) []byte {
	t.Helper()
	return buildZip(t, fullFixtureFiles("Ship"))
}

func buildFixtureZipWithRenamedShipCategory(t *testing.T) []byte {
	t.Helper()
	return buildZip(t, fullFixtureFiles("Ship Hull"))
}

func buildZip(t *testing.T, files map[string][]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, lines := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		for _, line := range lines {
			if _, err := w.Write([]byte(line + "\n")); err != nil {
				t.Fatalf("write zip entry %s: %v", name, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

// fixtureServer serves an index.json (pointing back at its own /sde.zip)
// and the zip body itself, counting hits to each so tests can assert
// exactly how many times each endpoint was actually called.
type fixtureServer struct {
	*httptest.Server
	indexHits atomic.Int32
	zipHits   atomic.Int32
	etag      string
	zipBody   []byte
}

func newFixtureServer(t *testing.T, etag string, zipBody []byte) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{etag: etag, zipBody: zipBody}
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		fs.indexHits.Add(1)
		idx := map[string]any{
			"files": []map[string]string{
				{"name": "eve-online-static-data-latest-jsonl.zip", "url": fs.Server.URL + "/sde.zip", "etag": fs.etag},
			},
		}
		_ = json.NewEncoder(w).Encode(idx)
	})
	mux.HandleFunc("/sde.zip", func(w http.ResponseWriter, r *http.Request) {
		fs.zipHits.Add(1)
		_, _ = w.Write(fs.zipBody)
	})
	fs.Server = httptest.NewServer(mux)
	t.Cleanup(fs.Close)
	return fs
}

func openWithIndexURL(t *testing.T, indexURL string) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "sde.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	store.indexURL = indexURL
	return store
}

func openAtPathWithIndexURL(t *testing.T, path, indexURL string) *Store {
	t.Helper()
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	store.indexURL = indexURL
	return store
}

func assertRecord(t *testing.T, store *Store, bucket string, id int64, wantEnglishName string, wantPublished bool) {
	t.Helper()
	raw := getRawRecord(t, store, bucket, id)
	if raw == nil {
		t.Fatalf("no cached record for %s[%d]", bucket, id)
	}
	var decoded struct {
		Name      map[string]string `json:"name"`
		Published bool              `json:"published"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode cached %s record %d: %v", bucket, id, err)
	}
	if decoded.Name["en"] != wantEnglishName {
		t.Errorf("%s[%d].Name[en] = %q, want %q", bucket, id, decoded.Name["en"], wantEnglishName)
	}
	if decoded.Published != wantPublished {
		t.Errorf("%s[%d].Published = %v, want %v", bucket, id, decoded.Published, wantPublished)
	}
}

func getRawRecord(t *testing.T, store *Store, bucket string, id int64) []byte {
	t.Helper()
	var raw []byte
	err := store.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		if v := b.Get(int64Key(id)); v != nil {
			raw = append([]byte(nil), v...)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("read %s[%d]: %v", bucket, id, err)
	}
	return raw
}

func bucketExists(t *testing.T, store *Store, bucket string) bool {
	t.Helper()
	var exists bool
	err := store.db.View(func(tx *bbolt.Tx) error {
		exists = tx.Bucket([]byte(bucket)) != nil
		return nil
	})
	if err != nil {
		t.Fatalf("check bucket %s: %v", bucket, err)
	}
	return exists
}

func TestSync_FirstSyncLoadsAllResourcesWithAugmentation(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v", err)
	}

	assertRecord(t, store, "categories", 6, "Ship", true)
	assertRecord(t, store, "groups", 25, "Frigate", true)
	assertRecord(t, store, "types", 587, "Rifter", true)

	group, ok, err := store.Group(context.Background(), testUserAgent, 25)
	if err != nil || !ok {
		t.Fatalf("Group(25) = %+v, %v, %v", group, ok, err)
	}
	if len(group.TypeIDs) != 1 || group.TypeIDs[0] != 587 {
		t.Errorf("Group(25).TypeIDs = %v, want [587] (reverse index from types.jsonl)", group.TypeIDs)
	}

	category, ok, err := store.Category(context.Background(), testUserAgent, 6)
	if err != nil || !ok {
		t.Fatalf("Category(6) = %+v, %v, %v", category, ok, err)
	}
	if len(category.GroupIDs) != 1 || category.GroupIDs[0] != 25 {
		t.Errorf("Category(6).GroupIDs = %v, want [25] (reverse index from groups.jsonl)", category.GroupIDs)
	}

	typ, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil || !ok {
		t.Fatalf("Type(587) = %+v, %v, %v", typ, ok, err)
	}
	if len(typ.DogmaAttributes) != 1 || typ.DogmaAttributes[0].AttributeID != 182 {
		t.Errorf("Type(587).DogmaAttributes = %+v, want one entry with AttributeID=182 (from typeDogma.jsonl)", typ.DogmaAttributes)
	}
	if len(typ.DogmaEffects) != 1 || typ.DogmaEffects[0].EffectID != 4 {
		t.Errorf("Type(587).DogmaEffects = %+v, want one entry with EffectID=4", typ.DogmaEffects)
	}

	sys, ok, err := store.System(context.Background(), testUserAgent, 30000001)
	if err != nil || !ok {
		t.Fatalf("System(30000001) = %+v, %v, %v", sys, ok, err)
	}
	if len(sys.StationIDs) != 1 || sys.StationIDs[0] != 60000001 {
		t.Errorf("System(30000001).StationIDs = %v, want [60000001] (reverse index from npcStations.jsonl)", sys.StationIDs)
	}

	factions, err := store.Factions(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(factions) != 1 {
		t.Fatalf("Factions = %+v, want one entry", factions)
	}
	if factions[0].StationCount != 1 || factions[0].StationSystemCount != 1 {
		t.Errorf("Faction(500001) StationCount/StationSystemCount = %d/%d, want 1/1 (3-hop join: station 60000001 -> corp 1000035 -> faction 500001)",
			factions[0].StationCount, factions[0].StationSystemCount)
	}

	region, ok, err := store.Region(context.Background(), testUserAgent, 10000001)
	if err != nil || !ok || region.Name["en"] != "Derelik" {
		t.Errorf("Region(10000001) = %+v, %v, %v, want Name[en]=Derelik", region, ok, err)
	}
	constellation, ok, err := store.Constellation(context.Background(), testUserAgent, 20000001)
	if err != nil || !ok || constellation.Name["en"] != "San Matar" {
		t.Errorf("Constellation(20000001) = %+v, %v, %v, want Name[en]=San Matar", constellation, ok, err)
	}
	ancestries, err := store.Ancestries(context.Background(), testUserAgent)
	if err != nil || len(ancestries) != 1 {
		t.Errorf("Ancestries = %+v, %v, want one entry", ancestries, err)
	}
	bloodlines, err := store.Bloodlines(context.Background(), testUserAgent)
	if err != nil || len(bloodlines) != 1 {
		t.Errorf("Bloodlines = %+v, %v, want one entry", bloodlines, err)
	}
	races, err := store.Races(context.Background(), testUserAgent)
	if err != nil || len(races) != 1 {
		t.Errorf("Races = %+v, %v, want one entry", races, err)
	}
	graphic, ok, err := store.Graphic(context.Background(), testUserAgent, 10)
	if err != nil || !ok || graphic.GraphicFile == nil || *graphic.GraphicFile != "res:/x" {
		t.Errorf("Graphic(10) = %+v, %v, %v, want GraphicFile=res:/x", graphic, ok, err)
	}

	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times, want exactly 1", srv.zipHits.Load())
	}
}

func TestSync_OnceOnlyPerStoreLifetime(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first ensureSynced returned error: %v", err)
	}
	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second ensureSynced returned error: %v", err)
	}

	if srv.indexHits.Load() != 1 {
		t.Errorf("index requested %d times across two ensureSynced calls on the same Store, want exactly 1", srv.indexHits.Load())
	}
}

func TestSync_ConcurrentEnsureSyncedCallsOnlySyncOnce(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.ensureSynced(context.Background(), testUserAgent)
		}()
	}
	wg.Wait()

	if srv.indexHits.Load() != 1 {
		t.Errorf("index requested %d times across 20 concurrent ensureSynced calls, want exactly 1", srv.indexHits.Load())
	}
}

func TestSync_UnchangedEtagSkipsDownload(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))

	first := openAtPathWithIndexURL(t, dbPath, srv.Server.URL+"/index.json")
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}
	if srv.zipHits.Load() != 1 {
		t.Fatalf("zip requested %d times on first sync, want exactly 1", srv.zipHits.Load())
	}

	second := openAtPathWithIndexURL(t, dbPath, srv.Server.URL+"/index.json")
	t.Cleanup(func() { second.Close() })
	if err := second.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times after a second Store synced against an unchanged etag, want still exactly 1", srv.zipHits.Load())
	}
	if srv.indexHits.Load() != 2 {
		t.Errorf("index requested %d times total, want exactly 2 (one per Store)", srv.indexHits.Load())
	}
}

func TestSync_ChangedEtagReloads(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")

	srv1 := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	first := openAtPathWithIndexURL(t, dbPath, srv1.Server.URL+"/index.json")
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}

	srv2 := newFixtureServer(t, "etag-v2", buildFixtureZipWithRenamedShipCategory(t))
	second := openAtPathWithIndexURL(t, dbPath, srv2.Server.URL+"/index.json")
	t.Cleanup(func() { second.Close() })
	if err := second.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	assertRecord(t, second, "categories", 6, "Ship Hull", true)
	if srv2.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times on the changed-etag sync, want exactly 1", srv2.zipHits.Load())
	}
}

func TestSync_FailedSyncFallsBackToCachedData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")

	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	first := openAtPathWithIndexURL(t, dbPath, srv.Server.URL+"/index.json")
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failingServer.Close)

	second := openAtPathWithIndexURL(t, dbPath, failingServer.URL+"/index.json")
	t.Cleanup(func() { second.Close() })

	if err := second.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error from the failing index server")
	}

	assertRecord(t, second, "categories", 6, "Ship", true)
}

func TestSync_EmptyRemoteEtagReturnsError(t *testing.T) {
	srv := newFixtureServer(t, "", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error for an empty remote etag")
	}
	if srv.zipHits.Load() != 0 {
		t.Errorf("zip requested %d times, want 0 (should fail before attempting a download)", srv.zipHits.Load())
	}
}

func TestSync_FailedResourceLoadRollsBackWholeTransaction(t *testing.T) {
	files := fullFixtureFiles("Ship")
	files["types.jsonl"] = []string{"not valid json"}
	badZip := buildZip(t, files)
	srv := newFixtureServer(t, "etag-v1", badZip)
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want a parse error from the malformed types.jsonl line")
	}

	v, err := store.version()
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if v != "" {
		t.Errorf("version = %q, want empty (a failed sync must not persist a version)", v)
	}
	for _, bucket := range []string{"categories", "groups", "types", "regions", "constellations", "systems", "factions", "ancestries", "bloodlines", "races", "graphics"} {
		if bucketExists(t, store, bucket) {
			t.Errorf("bucket %q exists after a failed sync, want no bucket (the whole transaction should roll back)", bucket)
		}
	}
}

func TestSync_MalformedIndexJSON_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not valid json"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	store := openWithIndexURL(t, srv.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want a parse error for malformed index.json")
	}
}

func TestSync_IndexMissingSDEZipEntry_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		idx := map[string]any{
			"files": []map[string]string{
				{"name": "some-other-file.zip", "url": "http://example.invalid/other.zip", "etag": "etag-x"},
			},
		}
		_ = json.NewEncoder(w).Encode(idx)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	store := openWithIndexURL(t, srv.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the index has no eve-online-static-data-latest-jsonl.zip entry")
	}
}

func TestSync_ZipFetchFails_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		idx := map[string]any{
			"files": []map[string]string{
				{"name": "eve-online-static-data-latest-jsonl.zip", "url": srv.URL + "/sde.zip", "etag": "etag-v1"},
			},
		}
		_ = json.NewEncoder(w).Encode(idx)
	})
	mux.HandleFunc("/sde.zip", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	store := openWithIndexURL(t, srv.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the zip fetch itself fails")
	}
}

func TestSync_ZipMissingExpectedEntry_ReturnsError(t *testing.T) {
	files := fullFixtureFiles("Ship")
	delete(files, "types.jsonl")
	incompleteZip := buildZip(t, files)
	srv := newFixtureServer(t, "etag-v1", incompleteZip)
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the archive is missing an expected JSONL file")
	}
}

func TestCategory_TriggersRealSyncOnNeverSyncedStore(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	got, ok, err := store.Category(context.Background(), testUserAgent, 6)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok || got.Name["en"] != "Ship" {
		t.Errorf("Category(6) = %+v, ok=%v, want Name[en]=Ship (from a real triggered sync)", got, ok)
	}
	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times, want exactly 1 (Category should trigger the lazy sync itself)", srv.zipHits.Load())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run 'TestSync_FirstSyncLoadsAllResourcesWithAugmentation|TestSync_FailedResourceLoadRollsBackWholeTransaction|TestSync_ZipMissingExpectedEntry_ReturnsError' -v`
Expected: FAIL — the fixture zip now contains 13 files but `sync()` still only reads the original 3
(`syncedFiles`), so `Group(25).TypeIDs`/`Category(6).GroupIDs`/`System.StationIDs`/`Faction`
assertions fail (empty/zero), and the "missing types.jsonl" test doesn't error the way it used to
since `sync()` doesn't try to read `types.jsonl` as a hard requirement the same way yet. Some of
these may currently pass by accident — the important thing is the augmentation assertions
(`TypeIDs`, `GroupIDs`, `StationIDs`, `StationCount`) fail until Step 3 lands.

- [ ] **Step 3: Write the two-phase sync pipeline**

Modify `sde/sync.go` — replace the whole file:
```go
package sde

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.etcd.io/bbolt"
)

const sdeZipName = "eve-online-static-data-latest-jsonl.zip"

var httpClient = &http.Client{Timeout: 30 * time.Second}

type sdeIndex struct {
	Files []sdeIndexFile `json:"files"`
}

type sdeIndexFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Etag string `json:"etag"`
}

// ensureSynced checks the store's cached SDE version against EVE Ref's live
// index exactly once per Store lifetime (via s.syncOnce), downloading and
// reloading every resource if it's stale or missing. A failed sync is
// cached too -- lookups fall back to whatever's already in bbolt (empty,
// on a never-synced install) rather than retrying every call. SDE data is
// enrichment, not a hard requirement for any ESI tool to function -- callers
// are expected to discard this method's return value, but the failure is
// still logged here (once, at the point it happens) so it isn't completely
// invisible when it does occur.
func (s *Store) ensureSynced(ctx context.Context, userAgent string) error {
	s.syncOnce.Do(func() {
		s.syncErr = s.sync(ctx, userAgent)
		if s.syncErr != nil {
			slog.Default().ErrorContext(ctx, "sde: sync failed, falling back to cached data", "error", s.syncErr)
		}
	})
	return s.syncErr
}

func (s *Store) sync(ctx context.Context, userAgent string) error {
	remoteEtag, zipURL, err := latestZipInfo(ctx, s.indexURL, userAgent)
	if err != nil {
		return fmt.Errorf("fetch sde index: %w", err)
	}
	if remoteEtag == "" {
		// A never-synced Store also has currentVersion == "" -- without this
		// check, an index response missing its etag field would silently
		// look "already up to date" and permanently skip downloading.
		return fmt.Errorf("sde index returned an empty etag for %s", sdeZipName)
	}

	currentVersion, err := s.version()
	if err != nil {
		return fmt.Errorf("read cached sde version: %w", err)
	}
	if currentVersion == remoteEtag {
		return nil
	}

	body, err := fetchBytes(ctx, zipURL, userAgent)
	if err != nil {
		return fmt.Errorf("download sde archive: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("open sde archive: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		// Types/groups/categories: each level needs the level below it's
		// reverse index before it can be written.
		dogma, err := loadTypeDogma(zr)
		if err != nil {
			return fmt.Errorf("load typeDogma.jsonl: %w", err)
		}
		typeIDsByGroup, err := loadTypes(tx, zr, dogma)
		if err != nil {
			return fmt.Errorf("load types.jsonl: %w", err)
		}
		categoryToGroups, err := loadGroups(tx, zr, typeIDsByGroup)
		if err != nil {
			return fmt.Errorf("load groups.jsonl: %w", err)
		}
		if err := loadCategories(tx, zr, categoryToGroups); err != nil {
			return fmt.Errorf("load categories.jsonl: %w", err)
		}

		// Systems/factions: both need a slim pass over npcStations.jsonl;
		// factions additionally need npcCorporations.jsonl to resolve each
		// station's owning corp to a faction.
		corpToFaction, err := loadCorpFactionMap(zr)
		if err != nil {
			return fmt.Errorf("load npcCorporations.jsonl: %w", err)
		}
		stationsBySystem, factionAgg, err := loadStationIndices(zr, corpToFaction)
		if err != nil {
			return fmt.Errorf("load npcStations.jsonl: %w", err)
		}
		if err := loadSystems(tx, zr, stationsBySystem); err != nil {
			return fmt.Errorf("load mapSolarSystems.jsonl: %w", err)
		}
		if err := loadFactions(tx, zr, factionAgg); err != nil {
			return fmt.Errorf("load factions.jsonl: %w", err)
		}

		// Everything else needs no augmentation -- plain passthrough.
		passthrough := []struct{ file, bucket string }{
			{"mapRegions.jsonl", "regions"},
			{"mapConstellations.jsonl", "constellations"},
			{"ancestries.jsonl", "ancestries"},
			{"bloodlines.jsonl", "bloodlines"},
			{"races.jsonl", "races"},
			{"graphics.jsonl", "graphics"},
		}
		for _, p := range passthrough {
			if err := loadResource(tx, zr, p.file, p.bucket); err != nil {
				return fmt.Errorf("load %s: %w", p.file, err)
			}
		}

		meta, err := tx.CreateBucketIfNotExists([]byte("_meta"))
		if err != nil {
			return fmt.Errorf("create _meta bucket: %w", err)
		}
		return meta.Put([]byte("version"), []byte(remoteEtag))
	})
}

// version returns the SDE version currently cached in bbolt's _meta bucket,
// or "" if the store has never been synced.
func (s *Store) version() (string, error) {
	var version string
	err := s.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte("_meta"))
		if meta == nil {
			return nil
		}
		version = string(meta.Get([]byte("version")))
		return nil
	})
	return version, err
}

// scanRecords opens file inside zr and calls fn once per JSONL line with
// that record's _key and raw bytes, shared by every loader below.
func scanRecords(zr *zip.Reader, file string, fn func(key int64, raw []byte) error) error {
	f, err := zr.Open(file)
	if err != nil {
		return fmt.Errorf("open zip entry: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // types.jsonl has long lines
	for scanner.Scan() {
		line := scanner.Bytes()
		var keyed struct {
			Key int64 `json:"_key"`
		}
		if err := json.Unmarshal(line, &keyed); err != nil {
			return fmt.Errorf("parse record: %w", err)
		}
		if err := fn(keyed.Key, append([]byte(nil), line...)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// loadResource clears and rewrites bucket from file's JSONL lines within
// the zip, verbatim, keyed by each record's _key field. Used for resources
// needing no augmentation.
func loadResource(tx *bbolt.Tx, zr *zip.Reader, file, bucket string) error {
	if err := tx.DeleteBucket([]byte(bucket)); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte(bucket))
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	return scanRecords(zr, file, func(key int64, raw []byte) error {
		if err := b.Put(int64Key(key), raw); err != nil {
			return fmt.Errorf("write record %d: %w", key, err)
		}
		return nil
	})
}

type dogmaData struct {
	Attributes []typeDogmaAttributeRecord `json:"dogmaAttributes,omitempty"`
	Effects    []typeDogmaEffectRecord    `json:"dogmaEffects,omitempty"`
}

// loadTypeDogma reads typeDogma.jsonl into an in-memory typeID -> dogma map;
// it has no bucket of its own, only used to augment types.jsonl records.
func loadTypeDogma(zr *zip.Reader) (map[int64]dogmaData, error) {
	out := make(map[int64]dogmaData)
	err := scanRecords(zr, "typeDogma.jsonl", func(key int64, raw []byte) error {
		var d dogmaData
		if err := json.Unmarshal(raw, &d); err != nil {
			return fmt.Errorf("parse typeDogma record %d: %w", key, err)
		}
		out[key] = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// loadTypes writes the types bucket (augmented with dogma) and returns the
// groupID -> []typeID reverse index groups.jsonl needs next.
func loadTypes(tx *bbolt.Tx, zr *zip.Reader, dogma map[int64]dogmaData) (map[int64][]int64, error) {
	if err := tx.DeleteBucket([]byte("types")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return nil, fmt.Errorf("clear types bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("types"))
	if err != nil {
		return nil, fmt.Errorf("create types bucket: %w", err)
	}

	groupToTypes := make(map[int64][]int64)
	err = scanRecords(zr, "types.jsonl", func(key int64, raw []byte) error {
		var rec typeRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse type record %d: %w", key, err)
		}
		if d, ok := dogma[key]; ok {
			rec.DogmaAttributes = d.Attributes
			rec.DogmaEffects = d.Effects
		}
		groupToTypes[rec.GroupID] = append(groupToTypes[rec.GroupID], key)
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented type %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
	if err != nil {
		return nil, err
	}
	return groupToTypes, nil
}

// loadGroups writes the groups bucket (augmented with TypeIDs from the map
// loadTypes produced) and returns the categoryID -> []groupID reverse index
// categories.jsonl needs next.
func loadGroups(tx *bbolt.Tx, zr *zip.Reader, typeIDsByGroup map[int64][]int64) (map[int64][]int64, error) {
	if err := tx.DeleteBucket([]byte("groups")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return nil, fmt.Errorf("clear groups bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("groups"))
	if err != nil {
		return nil, fmt.Errorf("create groups bucket: %w", err)
	}

	categoryToGroups := make(map[int64][]int64)
	err = scanRecords(zr, "groups.jsonl", func(key int64, raw []byte) error {
		var rec groupRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse group record %d: %w", key, err)
		}
		rec.TypeIDs = typeIDsByGroup[key]
		categoryToGroups[rec.CategoryID] = append(categoryToGroups[rec.CategoryID], key)
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented group %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
	if err != nil {
		return nil, err
	}
	return categoryToGroups, nil
}

// loadCategories writes the categories bucket, augmented with GroupIDs from
// the map loadGroups produced.
func loadCategories(tx *bbolt.Tx, zr *zip.Reader, groupIDsByCategory map[int64][]int64) error {
	if err := tx.DeleteBucket([]byte("categories")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear categories bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("categories"))
	if err != nil {
		return fmt.Errorf("create categories bucket: %w", err)
	}
	return scanRecords(zr, "categories.jsonl", func(key int64, raw []byte) error {
		var rec categoryRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse category record %d: %w", key, err)
		}
		rec.GroupIDs = groupIDsByCategory[key]
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented category %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

type corpFactionRecord struct {
	FactionID *int64 `json:"factionID,omitempty"`
}

// loadCorpFactionMap reads a slim pass over npcCorporations.jsonl (just the
// optional factionID field) into an in-memory corpID -> factionID map; it
// has no bucket of its own.
func loadCorpFactionMap(zr *zip.Reader) (map[int64]int64, error) {
	out := make(map[int64]int64)
	err := scanRecords(zr, "npcCorporations.jsonl", func(key int64, raw []byte) error {
		var rec corpFactionRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse npcCorporations record %d: %w", key, err)
		}
		if rec.FactionID != nil {
			out[key] = *rec.FactionID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type stationSlimRecord struct {
	SolarSystemID int64 `json:"solarSystemID"`
	OwnerID       int64 `json:"ownerID"`
}

type factionStationAgg struct {
	StationCount int64
	Systems      map[int64]struct{}
}

// loadStationIndices does one slim pass over npcStations.jsonl, producing
// both the solarSystemID -> []stationID reverse index mapSolarSystems.jsonl
// needs, and the factionID -> station/system-count aggregate factions.jsonl
// needs (via corpToFaction, from loadCorpFactionMap) -- reading the file
// once serves both, rather than scanning it twice.
func loadStationIndices(zr *zip.Reader, corpToFaction map[int64]int64) (stationsBySystem map[int64][]int64, factionAgg map[int64]*factionStationAgg, err error) {
	stationsBySystem = make(map[int64][]int64)
	factionAgg = make(map[int64]*factionStationAgg)

	err = scanRecords(zr, "npcStations.jsonl", func(key int64, raw []byte) error {
		var rec stationSlimRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse npcStations record %d: %w", key, err)
		}
		stationsBySystem[rec.SolarSystemID] = append(stationsBySystem[rec.SolarSystemID], key)

		if factionID, ok := corpToFaction[rec.OwnerID]; ok {
			agg, exists := factionAgg[factionID]
			if !exists {
				agg = &factionStationAgg{Systems: make(map[int64]struct{})}
				factionAgg[factionID] = agg
			}
			agg.StationCount++
			agg.Systems[rec.SolarSystemID] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return stationsBySystem, factionAgg, nil
}

// loadSystems writes the systems bucket, augmented with StationIDs from the
// map loadStationIndices produced (mapSolarSystems.jsonl has no stationIDs
// field of its own).
func loadSystems(tx *bbolt.Tx, zr *zip.Reader, stationsBySystem map[int64][]int64) error {
	if err := tx.DeleteBucket([]byte("systems")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear systems bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("systems"))
	if err != nil {
		return fmt.Errorf("create systems bucket: %w", err)
	}
	return scanRecords(zr, "mapSolarSystems.jsonl", func(key int64, raw []byte) error {
		var rec systemRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse system record %d: %w", key, err)
		}
		rec.StationIDs = stationsBySystem[key]
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented system %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// loadFactions writes the factions bucket, augmented with StationCount/
// StationSystemCount from the aggregate loadStationIndices produced
// (factions.jsonl has neither field itself).
func loadFactions(tx *bbolt.Tx, zr *zip.Reader, factionAgg map[int64]*factionStationAgg) error {
	if err := tx.DeleteBucket([]byte("factions")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear factions bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("factions"))
	if err != nil {
		return fmt.Errorf("create factions bucket: %w", err)
	}
	return scanRecords(zr, "factions.jsonl", func(key int64, raw []byte) error {
		var rec factionRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse faction record %d: %w", key, err)
		}
		if agg, ok := factionAgg[key]; ok {
			rec.StationCount = agg.StationCount
			rec.StationSystemCount = int64(len(agg.Systems))
		}
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented faction %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// int64Key encodes id as a big-endian byte slice suitable for use as a
// bbolt key.
func int64Key(id int64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(id))
	return key
}

func latestZipInfo(ctx context.Context, indexURL, userAgent string) (etag, url string, err error) {
	body, err := fetchBytes(ctx, indexURL, userAgent)
	if err != nil {
		return "", "", fmt.Errorf("fetch index: %w", err)
	}
	var idx sdeIndex
	if err := json.Unmarshal(body, &idx); err != nil {
		return "", "", fmt.Errorf("parse index: %w", err)
	}
	for _, f := range idx.Files {
		if f.Name == sdeZipName {
			return f.Etag, f.URL, nil
		}
	}
	return "", "", fmt.Errorf("%s not found in sde index", sdeZipName)
}

func fetchBytes(ctx context.Context, url, userAgent string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -v`
Expected: PASS — every test in the `sde/` package, including all of Tasks 1–5's tests (their
`seedStore`-based tests are unaffected by this rewrite) and every test in the rewritten
`sync_test.go`.

- [ ] **Step 5: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race -v`
Expected: clean build/vet/gofmt, every `sde/` test passes under `-race`.

- [ ] **Step 6: Commit**

```bash
git add sde/sync.go sde/sync_test.go
git commit -m "feat: rewrite sync.go as a two-phase compute-then-commit pipeline"
```

---

### Task 7: Migrate types/groups/categories (6 tools) to the read-through pattern

**Files:**
- Modify: `sde/sync.go` (add `int64FromKey`, the inverse of `int64Key`)
- Modify: `sde/category.go`, `sde/category_test.go` (add `CategoryIDs`)
- Modify: `sde/group.go`, `sde/group_test.go` (add `GroupIDs`)
- Modify: `sde/type.go`, `sde/type_test.go` (add `TypeIDs`)
- Modify: `mcp/universe.go` (rewrite the 6 handlers/vars to read-through)
- Modify: `mcp/universe_internal_test.go` (rewrite the 6 tools' tests)
- Modify: `mcp/server.go` (`NewServer` gains `sdeStore *sde.Store`; 6 registration call sites pass it)
- Modify: `mcp/server_test.go` (`connectedSession` needs an `*sde.Store` for the new `NewServer` param)
- Modify: `commands/mcp.go` (open the `*sde.Store` once, pass it into `mcp.NewServer`)

**Interfaces:**
- Consumes: `sde.Category`/`sde.Group`/`sde.Type` and their `Store` accessors/`Put*` methods (Tasks
  2/3); `esi.Client.UniverseCategoryDetail`/`UniverseGroupDetail`/`UniverseTypeDetail`/
  `UniverseCategories`/`UniverseGroups`/`UniverseTypes` (unchanged, `esi/universe.go`); the existing
  `UniverseCategory`/`UniverseGroup`/`UniverseType`/`ListUniverse*Output` MCP types (unchanged
  shape, except `UniverseType.PackagedVolume` is now populated only on a cache miss).
- Produces (for Task 8/9, which follow the exact same pattern): `(s *Store) CategoryIDs(ctx,
  userAgent) ([]int64, error)`, `GroupIDs`, `TypeIDs` (same "empty slice, not an error, if never
  synced" contract as Task 5's whole-collection accessors); `NewServer(version, clientID string,
  store *config.Store, sdeStore *sde.Store) *mcpsdk.Server` (Tasks 8/9 add no further parameters —
  every migrated handler constructor in those tasks takes the same already-threaded `sdeStore`).

**Why a narrow interface per resource, not `*sde.Store` directly, inside the swappable vars:**
`*sde.Store`'s accessors call `ensureSynced` internally, which does live network I/O on a real
`*sde.Store` the first time any accessor is called — and `Store.syncOnce`/`indexURL` are
unexported, so only tests *inside* package `sde` can neutralize that (as Tasks 1–6 already do via
`seedStore`'s `store.syncOnce.Do(func(){})` trick). `mcp/universe_test.go` tests are package `mcp`
and can't reach those unexported fields, so a test constructing a real `*sde.Store` would trigger
an actual outbound HTTPS call on every test run — slow, network-dependent, and nondeterministic
(a "successful" real sync would silently turn a cache-miss test into a cache-hit-against-real-data
test). Rather than exporting a test-only seam from `sde/` (exactly the `Exported*ForTest` pattern
PR #7's review already flagged and removed), each migrated tool's swappable var is typed against a
minimal interface capturing only the methods it needs. `*sde.Store` satisfies these interfaces
structurally, with zero changes to `sde/`'s public surface — production code passes the real
`*sde.Store` straight through; tests substitute a small in-memory fake.

**Verified against real data:**
```
curl -sD - "https://esi.evetech.net/latest/universe/groups/?page=1" -> x-pages: 2, 1000 IDs on page 1
```
Confirmed live against ESI: `list_universe_groups`/`list_universe_types` paginate at exactly 1000
IDs per page. On a cache hit, `sde/`'s bucket holds every ID unpaginated — the read-through layer
must replicate this page size itself to keep `Page`/`TotalPages` semantics identical to today.

**Field-drop nuance (applies here and to every later field-drop in Tasks 8/9):** `PackagedVolume`
has no SDE source at all. On a cache miss, the live ESI response already has it — the read-through
handler returns it in that response even though it isn't persisted into `sde.Type` (there's no
field to hold it). Only a cache *hit* actually lacks the field. "Drop on cache hits" was your
exact call — the miss path stays full-fidelity since the data is already in hand.

- [ ] **Step 1: Write the failing tests for `sde/`'s three new ID-list accessors**

Modify `sde/category_test.go` — add after `TestCategory_MismatchedKey_ReturnsError`:
```go

func TestCategoryIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "categories", 6, `{"_key": 6, "name": {"en": "Ship"}, "published": true}`)
	putRecord(t, store, "categories", 4, `{"_key": 4, "name": {"en": "Material"}, "published": true}`)

	ids, err := store.CategoryIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("CategoryIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 4 || ids[1] != 6 {
		t.Errorf("CategoryIDs = %v, want [4 6] (ascending)", ids)
	}
}

func TestCategoryIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.CategoryIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("CategoryIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("CategoryIDs = %v, want empty", ids)
	}
}
```

Modify `sde/group_test.go` — add the analogous pair:
```go

func TestGroupIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "groups", 25, `{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true}`)
	putRecord(t, store, "groups", 26, `{"_key": 26, "categoryID": 6, "name": {"en": "Cruiser"}, "published": true}`)

	ids, err := store.GroupIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GroupIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 25 || ids[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26] (ascending)", ids)
	}
}

func TestGroupIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.GroupIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GroupIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("GroupIDs = %v, want empty", ids)
	}
}
```

Modify `sde/type_test.go` — add the analogous pair:
```go

func TestTypeIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587, `{"_key": 587, "groupID": 25, "name": {"en": "Rifter"}, "published": true}`)
	putRecord(t, store, "types", 588, `{"_key": 588, "groupID": 25, "name": {"en": "Breacher"}, "published": true}`)

	ids, err := store.TypeIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("TypeIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 587 || ids[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588] (ascending)", ids)
	}
}

func TestTypeIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.TypeIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("TypeIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("TypeIDs = %v, want empty", ids)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sde/... -run 'TestCategoryIDs|TestGroupIDs|TestTypeIDs' -v`
Expected: FAIL to compile — `store.CategoryIDs undefined`, `store.GroupIDs undefined`,
`store.TypeIDs undefined`.

- [ ] **Step 3: Implement the three accessors**

Modify `sde/sync.go` — add immediately after `int64Key`:
```go

// int64FromKey decodes a key produced by int64Key back into its original
// int64 form.
func int64FromKey(key []byte) int64 {
	return int64(binary.BigEndian.Uint64(key))
}
```

Modify `sde/category.go` — add at the end of the file:
```go

// CategoryIDs returns every category ID currently cached. bbolt's ForEach
// visits keys in byte-sorted order, which is already numeric ascending
// order for int64Key's big-endian, non-negative encoding -- no separate
// sort needed. Returns an empty slice, not an error, if the store has never
// synced successfully; callers read that as "cache empty" and fall back to
// a live ESI list call, not "no categories exist".
func (s *Store) CategoryIDs(ctx context.Context, userAgent string) ([]int64, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("categories"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, int64FromKey(k))
			return nil
		})
	})
	return ids, err
}
```

Modify `sde/group.go` — add at the end of the file:
```go

// GroupIDs returns every group ID currently cached. See CategoryIDs for why
// no separate sort is needed and what an empty result means.
func (s *Store) GroupIDs(ctx context.Context, userAgent string) ([]int64, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("groups"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, int64FromKey(k))
			return nil
		})
	})
	return ids, err
}
```

Modify `sde/type.go` — add at the end of the file:
```go

// TypeIDs returns every type ID currently cached. See CategoryIDs for why
// no separate sort is needed and what an empty result means.
func (s *Store) TypeIDs(ctx context.Context, userAgent string) ([]int64, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("types"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, int64FromKey(k))
			return nil
		})
	})
	return ids, err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./sde/... -run 'TestCategoryIDs|TestGroupIDs|TestTypeIDs' -v`
Expected: PASS.

- [ ] **Step 5: Commit the `sde/` additions**

```bash
git add sde/sync.go sde/category.go sde/category_test.go sde/group.go sde/group_test.go sde/type.go sde/type_test.go
git commit -m "feat: add CategoryIDs/GroupIDs/TypeIDs bucket-enumeration accessors"
```

- [ ] **Step 6: Write the failing tests for the 6 migrated `mcp/universe.go` tools**

Modify `mcp/universe_internal_test.go` — replace every existing test function named
`TestListUniverseCategoriesHandler_*`, `TestGetUniverseCategoryHandler_*`,
`TestListUniverseGroupsHandler_*`, `TestGetUniverseGroupHandler_*`, `TestListUniverseTypesHandler_*`,
`TestGetUniverseTypeHandler_*` (all currently swap the old no-cache package vars) with the versions
below. Also add the three fake-cache types at the top of the block (used by every test in it):
```go

// fakeCategoryCache/fakeGroupCache/fakeTypeCache satisfy mcp's narrow
// categoryCache/groupCache/typeCache interfaces without a real (network-
// touching) *sde.Store -- see Task 7's plan notes for why *sde.Store can't
// be used directly in package mcp's tests.
type fakeCategoryCache struct {
	category   sde.Category
	categoryOK bool
	categoryErr error
	ids        []int64
	idsErr     error
	putErr     error
	putCalls   []sde.Category
}

func (f *fakeCategoryCache) Category(ctx context.Context, userAgent string, id int64) (sde.Category, bool, error) {
	return f.category, f.categoryOK, f.categoryErr
}
func (f *fakeCategoryCache) PutCategory(c sde.Category) error {
	f.putCalls = append(f.putCalls, c)
	return f.putErr
}
func (f *fakeCategoryCache) CategoryIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeGroupCache struct {
	group     sde.Group
	groupOK   bool
	groupErr  error
	ids       []int64
	idsErr    error
	putErr    error
	putCalls  []sde.Group
}

func (f *fakeGroupCache) Group(ctx context.Context, userAgent string, id int64) (sde.Group, bool, error) {
	return f.group, f.groupOK, f.groupErr
}
func (f *fakeGroupCache) PutGroup(g sde.Group) error {
	f.putCalls = append(f.putCalls, g)
	return f.putErr
}
func (f *fakeGroupCache) GroupIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeTypeCache struct {
	typ      sde.Type
	typeOK   bool
	typeErr  error
	ids      []int64
	idsErr   error
	putErr   error
	putCalls []sde.Type
}

func (f *fakeTypeCache) Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error) {
	return f.typ, f.typeOK, f.typeErr
}
func (f *fakeTypeCache) PutType(t sde.Type) error {
	f.putCalls = append(f.putCalls, t)
	return f.putErr
}
func (f *fakeTypeCache) TypeIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

func TestListUniverseCategories_CacheHit_ReturnsCachedIDs(t *testing.T) {
	cache := &fakeCategoryCache{ids: []int64{4, 6}}

	ids, err := listUniverseCategories(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseCategories returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 4 || ids[1] != 6 {
		t.Errorf("ids = %v, want [4 6]", ids)
	}
}

func TestListUniverseCategories_CacheError_ReturnsError(t *testing.T) {
	cache := &fakeCategoryCache{idsErr: errors.New("boom")}

	_, err := listUniverseCategories(context.Background(), cache, "test-agent")
	if err == nil {
		t.Fatal("listUniverseCategories returned nil error, want the wrapped cache error")
	}
}

func TestGetUniverseCategory_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeCategoryCache{
		category:   sde.Category{ID: 6, Name: map[string]string{"en": "Ship"}, Published: true, GroupIDs: []int64{25}},
		categoryOK: true,
	}

	got, err := getUniverseCategory(context.Background(), cache, "test-agent", 6)
	if err != nil {
		t.Fatalf("getUniverseCategory returned error: %v", err)
	}
	if got.CategoryID != 6 || got.Name != "Ship" || !got.Published || len(got.GroupIDs) != 1 || got.GroupIDs[0] != 25 {
		t.Errorf("got = %+v, want CategoryID=6 Name=Ship Published=true GroupIDs=[25]", got)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutCategory called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}

func TestGetUniverseCategory_CacheError_ReturnsError(t *testing.T) {
	cache := &fakeCategoryCache{categoryErr: errors.New("boom")}

	_, err := getUniverseCategory(context.Background(), cache, "test-agent", 6)
	if err == nil {
		t.Fatal("getUniverseCategory returned nil error, want the wrapped cache error")
	}
}

func TestListUniverseGroups_CacheHit_PaginatesAt1000(t *testing.T) {
	ids := make([]int64, 1500)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	cache := &fakeGroupCache{ids: ids}

	page1, totalPages, err := listUniverseGroups(context.Background(), cache, "test-agent", 1)
	if err != nil {
		t.Fatalf("listUniverseGroups page 1 returned error: %v", err)
	}
	if len(page1) != 1000 || page1[0] != 1 || page1[999] != 1000 {
		t.Errorf("page 1 = %d ids starting %d ending %d, want 1000 ids from 1 to 1000", len(page1), page1[0], page1[len(page1)-1])
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}

	page2, _, err := listUniverseGroups(context.Background(), cache, "test-agent", 2)
	if err != nil {
		t.Fatalf("listUniverseGroups page 2 returned error: %v", err)
	}
	if len(page2) != 500 || page2[0] != 1001 {
		t.Errorf("page 2 = %d ids starting %d, want 500 ids starting 1001", len(page2), page2[0])
	}
}

func TestListUniverseGroups_CacheHit_PageZeroDefaultsToOne(t *testing.T) {
	cache := &fakeGroupCache{ids: []int64{25, 26}}

	ids, totalPages, err := listUniverseGroups(context.Background(), cache, "test-agent", 0)
	if err != nil {
		t.Fatalf("listUniverseGroups returned error: %v", err)
	}
	if len(ids) != 2 || totalPages != 1 {
		t.Errorf("ids = %v, totalPages = %d, want [25 26], 1", ids, totalPages)
	}
}

func TestGetUniverseGroup_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeGroupCache{
		group:   sde.Group{ID: 25, CategoryID: 6, Name: map[string]string{"en": "Frigate"}, Published: true, TypeIDs: []int64{587}},
		groupOK: true,
	}

	got, err := getUniverseGroup(context.Background(), cache, "test-agent", 25)
	if err != nil {
		t.Fatalf("getUniverseGroup returned error: %v", err)
	}
	want := UniverseGroup{GroupID: 25, CategoryID: 6, Name: "Frigate", Published: true, TypeIDs: []int64{587}}
	if got.GroupID != want.GroupID || got.CategoryID != want.CategoryID || got.Name != want.Name || got.Published != want.Published || len(got.TypeIDs) != 1 || got.TypeIDs[0] != 587 {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}

func TestListUniverseTypes_CacheHit_Paginates(t *testing.T) {
	cache := &fakeTypeCache{ids: []int64{587, 588}}

	ids, totalPages, err := listUniverseTypes(context.Background(), cache, "test-agent", 1)
	if err != nil {
		t.Fatalf("listUniverseTypes returned error: %v", err)
	}
	if len(ids) != 2 || totalPages != 1 {
		t.Errorf("ids = %v, totalPages = %d, want [587 588], 1", ids, totalPages)
	}
}

func TestGetUniverseType_CacheHit_MapsFieldsAndDropsPackagedVolume(t *testing.T) {
	mass := 1067000.0
	cache := &fakeTypeCache{
		typ: sde.Type{
			ID: 587, GroupID: 25, Name: map[string]string{"en": "Rifter"}, Description: map[string]string{"en": "Fast."},
			Published: true, Mass: &mass,
			DogmaAttributes: []sde.TypeDogmaAttribute{{AttributeID: 182, Value: 3386.0}},
			DogmaEffects:    []sde.TypeDogmaEffect{{EffectID: 4, IsDefault: true}},
		},
		typeOK: true,
	}

	got, err := getUniverseType(context.Background(), cache, "test-agent", 587)
	if err != nil {
		t.Fatalf("getUniverseType returned error: %v", err)
	}
	if got.TypeID != 587 || got.Name != "Rifter" || got.Description != "Fast." || got.GroupID != 25 || !got.Published {
		t.Errorf("got = %+v, want TypeID=587 Name=Rifter Description=Fast. GroupID=25 Published=true", got)
	}
	if got.Mass == nil || *got.Mass != 1067000.0 {
		t.Errorf("Mass = %v, want 1067000.0", got.Mass)
	}
	if got.PackagedVolume != nil {
		t.Errorf("PackagedVolume = %v, want nil on a cache hit (no SDE source)", got.PackagedVolume)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].AttributeID != 182 {
		t.Errorf("DogmaAttributes = %+v, want one entry with AttributeID=182", got.DogmaAttributes)
	}
	if len(got.DogmaEffects) != 1 || got.DogmaEffects[0].EffectID != 4 {
		t.Errorf("DogmaEffects = %+v, want one entry with EffectID=4", got.DogmaEffects)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutType called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}
```

Keep the existing `TestListUniverseCategoriesHandler_MapsFields`-style handler tests, but update
each one's stubbed var signature to accept the new `sdeStore` parameter (unused by the stub, since
the handler-level tests only verify the handler's own plumbing — ctx/input mapping and error
propagation — not the read-through logic, which the tests above cover directly). For example:
```go
func TestListUniverseCategoriesHandler_MapsFields(t *testing.T) {
	orig := listUniverseCategories
	t.Cleanup(func() { listUniverseCategories = orig })
	listUniverseCategories = func(ctx context.Context, sdeStore categoryCache, userAgent string) ([]int64, error) {
		return []int64{4, 6}, nil
	}

	handler := listUniverseCategoriesHandler("test", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseCategoriesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 {
		t.Errorf("IDs = %v, want 2 entries", out.IDs)
	}
}
```
Apply the same one-parameter addition (`sdeStore categoryCache`/`groupCache`/`typeCache`, always
`nil` when the var is stubbed wholesale) to every other pre-existing `_MapsFields`/
`_PropagatesError`/`_DefaultsPageToOne` test for these 6 tools, and to their `handler := ...Handler("test", ...)`
call, which now takes `nil` as its new second argument (a real `*sde.Store` is never needed here —
the var is fully stubbed).

Add `"errors"` and `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to this test file's
imports if not already present.

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListUniverseCategories|TestGetUniverseCategory|TestListUniverseGroups|TestGetUniverseGroup|TestListUniverseTypes|TestGetUniverseType' -v`
Expected: FAIL to compile — `categoryCache`/`groupCache`/`typeCache` undefined, wrong number of
arguments to `listUniverseCategories`/etc., `listUniverseCategoriesHandler("test", nil)` too many
arguments.

- [ ] **Step 8: Rewrite `mcp/universe.go`'s 6 handlers to the read-through pattern**

Modify `mcp/universe.go` — add `"log/slog"` and
`"github.com/rusher2004/new-eden-agent-toolkit/sde"` to the import block. Replace the
`list_universe_categories` through `get_universe_type` sections (everything from the
`// --- list_universe_categories (public, no input) ---` comment through the end of
`getUniverseTypeHandler`, i.e. the code at `mcp/universe.go:188-394` before this task) with:
```go
// universeListPageSize is ESI's own page size for its large universe list
// endpoints -- verified live against /universe/groups/?page=1 (x-pages: 2
// for ~1700 groups, 1000 ids on page 1). A cache hit serves every ID
// unpaginated from sde/, so this constant lets the read-through layer
// replicate ESI's own pagination instead of changing the tool's contract.
const universeListPageSize = 1000

// paginateIDs slices a full, cache-hit ID list the same way ESI's own
// paginated list endpoints would. page <= 0 means page 1, matching every
// migrated list handler's existing default.
func paginateIDs(ids []int64, page int32) ([]int64, int32) {
	if page <= 0 {
		page = 1
	}
	totalPages := int32((len(ids) + universeListPageSize - 1) / universeListPageSize)
	if totalPages == 0 {
		totalPages = 1
	}
	start := int(page-1) * universeListPageSize
	if start >= len(ids) {
		return []int64{}, totalPages
	}
	end := start + universeListPageSize
	if end > len(ids) {
		end = len(ids)
	}
	return ids[start:end], totalPages
}

// --- list_universe_categories (public, no input) ---

// categoryCache is the subset of *sde.Store's category methods the
// read-through cache needs -- narrow enough that tests can substitute a
// fake without a real (network-touching) *sde.Store. *sde.Store satisfies
// this structurally; no sde/ changes are needed to pass one in.
type categoryCache interface {
	Category(ctx context.Context, userAgent string, id int64) (sde.Category, bool, error)
	PutCategory(sde.Category) error
	CategoryIDs(ctx context.Context, userAgent string) ([]int64, error)
}

var listUniverseCategories = func(ctx context.Context, sdeStore categoryCache, userAgent string) ([]int64, error) {
	ids, err := sdeStore.CategoryIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	// Cache empty (never synced) -- fall back to a live ESI list call.
	client := esi.NewPublicClient(userAgent)
	return client.UniverseCategories(ctx)
}

type ListUniverseCategoriesInput struct{}
type ListUniverseCategoriesOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseCategoriesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseCategoriesInput, ListUniverseCategoriesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseCategoriesInput) (*mcpsdk.CallToolResult, ListUniverseCategoriesOutput, error) {
		ids, err := listUniverseCategories(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseCategoriesOutput{}, fmt.Errorf("list universe categories: %w", err)
		}
		return nil, ListUniverseCategoriesOutput{IDs: ids}, nil
	}
}

// --- get_universe_category (public) ---

type UniverseCategory struct {
	CategoryID int64   `json:"categoryId"`
	Name       string  `json:"name"`
	Published  bool    `json:"published"`
	GroupIDs   []int64 `json:"groupIds"`
}

var getUniverseCategory = func(ctx context.Context, sdeStore categoryCache, userAgent string, categoryID int64) (UniverseCategory, error) {
	c, ok, err := sdeStore.Category(ctx, userAgent, categoryID)
	if err != nil {
		return UniverseCategory{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		// Cache miss (never synced, or a genuinely new ID not yet in the
		// snapshot) -- read through to ESI and write the result back so
		// subsequent lookups of this ID are fast.
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseCategoryDetail(ctx, categoryID)
		if err != nil {
			return UniverseCategory{}, err
		}
		c = sde.Category{ID: detail.CategoryId, Name: map[string]string{"en": detail.Name}, Published: detail.Published, GroupIDs: detail.Groups}
		if err := sdeStore.PutCategory(c); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through category into cache", "error", err)
		}
	}
	return UniverseCategory{CategoryID: c.ID, Name: c.Name["en"], Published: c.Published, GroupIDs: c.GroupIDs}, nil
}

type GetUniverseCategoryInput struct {
	CategoryID int64 `json:"categoryId" jsonschema:"the category ID to look up"`
}

func getUniverseCategoryHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseCategoryInput, UniverseCategory] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseCategoryInput) (*mcpsdk.CallToolResult, UniverseCategory, error) {
		out, err := getUniverseCategory(ctx, sdeStore, sso.UserAgent(version), in.CategoryID)
		if err != nil {
			return nil, UniverseCategory{}, fmt.Errorf("get universe category: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_groups (public, paginated) ---

type groupCache interface {
	Group(ctx context.Context, userAgent string, id int64) (sde.Group, bool, error)
	PutGroup(sde.Group) error
	GroupIDs(ctx context.Context, userAgent string) ([]int64, error)
}

var listUniverseGroups = func(ctx context.Context, sdeStore groupCache, userAgent string, page int32) ([]int64, int32, error) {
	ids, err := sdeStore.GroupIDs(ctx, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) == 0 {
		client := esi.NewPublicClient(userAgent)
		return client.UniverseGroups(ctx, page)
	}
	pageIDs, totalPages := paginateIDs(ids, page)
	return pageIDs, totalPages, nil
}

type ListUniverseGroupsInput struct {
	Page int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListUniverseGroupsOutput struct {
	IDs        []int64 `json:"ids"`
	Page       int32   `json:"page"`
	TotalPages int32   `json:"totalPages"`
}

func listUniverseGroupsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseGroupsInput, ListUniverseGroupsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseGroupsInput) (*mcpsdk.CallToolResult, ListUniverseGroupsOutput, error) {
		ids, totalPages, err := listUniverseGroups(ctx, sdeStore, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListUniverseGroupsOutput{}, fmt.Errorf("list universe groups: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListUniverseGroupsOutput{IDs: ids, Page: page, TotalPages: totalPages}, nil
	}
}

// --- get_universe_group (public) ---

type UniverseGroup struct {
	GroupID    int64   `json:"groupId"`
	CategoryID int64   `json:"categoryId"`
	Name       string  `json:"name"`
	Published  bool    `json:"published"`
	TypeIDs    []int64 `json:"typeIds"`
}

var getUniverseGroup = func(ctx context.Context, sdeStore groupCache, userAgent string, groupID int64) (UniverseGroup, error) {
	g, ok, err := sdeStore.Group(ctx, userAgent, groupID)
	if err != nil {
		return UniverseGroup{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseGroupDetail(ctx, groupID)
		if err != nil {
			return UniverseGroup{}, err
		}
		g = sde.Group{ID: detail.GroupId, CategoryID: detail.CategoryId, Name: map[string]string{"en": detail.Name}, Published: detail.Published, TypeIDs: detail.Types}
		if err := sdeStore.PutGroup(g); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through group into cache", "error", err)
		}
	}
	return UniverseGroup{GroupID: g.ID, CategoryID: g.CategoryID, Name: g.Name["en"], Published: g.Published, TypeIDs: g.TypeIDs}, nil
}

type GetUniverseGroupInput struct {
	GroupID int64 `json:"groupId" jsonschema:"the group ID to look up"`
}

func getUniverseGroupHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseGroupInput, UniverseGroup] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseGroupInput) (*mcpsdk.CallToolResult, UniverseGroup, error) {
		out, err := getUniverseGroup(ctx, sdeStore, sso.UserAgent(version), in.GroupID)
		if err != nil {
			return nil, UniverseGroup{}, fmt.Errorf("get universe group: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_types (public, paginated) ---

type typeCache interface {
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
	PutType(sde.Type) error
	TypeIDs(ctx context.Context, userAgent string) ([]int64, error)
}

var listUniverseTypes = func(ctx context.Context, sdeStore typeCache, userAgent string, page int32) ([]int64, int32, error) {
	ids, err := sdeStore.TypeIDs(ctx, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) == 0 {
		client := esi.NewPublicClient(userAgent)
		return client.UniverseTypes(ctx, page)
	}
	pageIDs, totalPages := paginateIDs(ids, page)
	return pageIDs, totalPages, nil
}

type ListUniverseTypesInput struct {
	Page int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListUniverseTypesOutput struct {
	IDs        []int64 `json:"ids"`
	Page       int32   `json:"page"`
	TotalPages int32   `json:"totalPages"`
}

func listUniverseTypesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseTypesInput, ListUniverseTypesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseTypesInput) (*mcpsdk.CallToolResult, ListUniverseTypesOutput, error) {
		ids, totalPages, err := listUniverseTypes(ctx, sdeStore, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListUniverseTypesOutput{}, fmt.Errorf("list universe types: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListUniverseTypesOutput{IDs: ids, Page: page, TotalPages: totalPages}, nil
	}
}

// --- get_universe_type (public) ---

type UniverseType struct {
	TypeID          int64                  `json:"typeId"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	GroupID         int64                  `json:"groupId"`
	Published       bool                   `json:"published"`
	Capacity        *float64               `json:"capacity,omitempty"`
	GraphicID       *int64                 `json:"graphicId,omitempty"`
	IconID          *int64                 `json:"iconId,omitempty"`
	MarketGroupID   *int64                 `json:"marketGroupId,omitempty"`
	Mass            *float64               `json:"mass,omitempty"`
	PackagedVolume  *float64               `json:"packagedVolume,omitempty"`
	PortionSize     *int64                 `json:"portionSize,omitempty"`
	Radius          *float64               `json:"radius,omitempty"`
	Volume          *float64               `json:"volume,omitempty"`
	DogmaAttributes []DynamicItemAttribute `json:"dogmaAttributes,omitempty"`
	DogmaEffects    []DynamicItemEffect    `json:"dogmaEffects,omitempty"`
}

var getUniverseType = func(ctx context.Context, sdeStore typeCache, userAgent string, typeID int64) (UniverseType, error) {
	t, ok, err := sdeStore.Type(ctx, userAgent, typeID)
	if err != nil {
		return UniverseType{}, fmt.Errorf("read sde cache: %w", err)
	}

	// PackagedVolume has no SDE source at all (confirmed absent from the
	// full archive, Task 3) -- it's only available fresh off a cache-miss
	// ESI call, never persisted, so it's simply unset on a cache hit.
	var packagedVolume *float64
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseTypeDetail(ctx, typeID)
		if err != nil {
			return UniverseType{}, err
		}
		packagedVolume = detail.PackagedVolume
		t = sde.Type{
			ID: detail.TypeId, GroupID: detail.GroupId, Name: map[string]string{"en": detail.Name},
			Description: map[string]string{"en": detail.Description}, Published: detail.Published,
			Capacity: detail.Capacity, GraphicID: detail.GraphicId, IconID: detail.IconId, MarketGroupID: detail.MarketGroupId,
			Mass: detail.Mass, PortionSize: detail.PortionSize, Radius: detail.Radius, Volume: detail.Volume,
		}
		if len(detail.DogmaAttributes) > 0 {
			t.DogmaAttributes = make([]sde.TypeDogmaAttribute, len(detail.DogmaAttributes))
			for i, a := range detail.DogmaAttributes {
				t.DogmaAttributes[i] = sde.TypeDogmaAttribute{AttributeID: a.AttributeId, Value: a.Value}
			}
		}
		if len(detail.DogmaEffects) > 0 {
			t.DogmaEffects = make([]sde.TypeDogmaEffect, len(detail.DogmaEffects))
			for i, e := range detail.DogmaEffects {
				t.DogmaEffects[i] = sde.TypeDogmaEffect{EffectID: e.EffectId, IsDefault: e.IsDefault}
			}
		}
		if err := sdeStore.PutType(t); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through type into cache", "error", err)
		}
	}

	out := UniverseType{
		TypeID: t.ID, Name: t.Name["en"], Description: t.Description["en"], GroupID: t.GroupID, Published: t.Published,
		Capacity: t.Capacity, GraphicID: t.GraphicID, IconID: t.IconID, MarketGroupID: t.MarketGroupID,
		Mass: t.Mass, PackagedVolume: packagedVolume, PortionSize: t.PortionSize, Radius: t.Radius, Volume: t.Volume,
	}
	if len(t.DogmaAttributes) > 0 {
		out.DogmaAttributes = make([]DynamicItemAttribute, len(t.DogmaAttributes))
		for i, a := range t.DogmaAttributes {
			out.DogmaAttributes[i] = DynamicItemAttribute{AttributeID: a.AttributeID, Value: a.Value}
		}
	}
	if len(t.DogmaEffects) > 0 {
		out.DogmaEffects = make([]DynamicItemEffect, len(t.DogmaEffects))
		for i, e := range t.DogmaEffects {
			out.DogmaEffects[i] = DynamicItemEffect{EffectID: e.EffectID, IsDefault: e.IsDefault}
		}
	}
	return out, nil
}

type GetUniverseTypeInput struct {
	TypeID int64 `json:"typeId" jsonschema:"the type ID to look up"`
}

func getUniverseTypeHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseTypeInput, UniverseType] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseTypeInput) (*mcpsdk.CallToolResult, UniverseType, error) {
		out, err := getUniverseType(ctx, sdeStore, sso.UserAgent(version), in.TypeID)
		if err != nil {
			return nil, UniverseType{}, fmt.Errorf("get universe type: %w", err)
		}
		return nil, out, nil
	}
}
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestListUniverseCategories|TestGetUniverseCategory|TestListUniverseGroups|TestGetUniverseGroup|TestListUniverseTypes|TestGetUniverseType' -v`
Expected: PASS — all new cache-hit/cache-error tests, and all pre-existing handler-level
`_MapsFields`/`_PropagatesError`/`_DefaultsPageToOne` tests updated in Step 6.

- [ ] **Step 10: Thread `*sde.Store` through `NewServer` and this task's 6 registration call sites**

Modify `mcp/server.go` — change the signature and add the import:
```go
import (
	// ... existing imports ...
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

func NewServer(version, clientID string, store *config.Store, sdeStore *sde.Store) *mcpsdk.Server {
```
Modify the 6 registration call sites for this task's tools (`mcp/server.go:1000-1027` before this
task) to pass `sdeStore`:
```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_categories",
		Description: "List every item category ID (public data, no character required).",
	}, listUniverseCategoriesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_category",
		Description: "Get an item category's name, published state, and member group IDs (public data, no character required).",
	}, getUniverseCategoryHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_groups",
		Description: "List every item group ID, paginated (public data, no character required).",
	}, listUniverseGroupsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_group",
		Description: "Get an item group's name, category, published state, and member type IDs (public data, no character required).",
	}, getUniverseGroupHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_types",
		Description: "List every item type ID, paginated (public data, no character required).",
	}, listUniverseTypesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_type",
		Description: "Get an item type's full detail: name, group, dogma attributes/effects, and physical stats (public data, no character required).",
	}, getUniverseTypeHandler(version, sdeStore))
```
Every other registration call site in this file (all not-yet-migrated tools, Tasks 8/9, plus the
6 permanently-ESI-direct celestial tools) is unchanged — those handler constructors don't take
`sdeStore` yet.

Modify `mcp/server_test.go`'s `connectedSession` to open a real (empty, never-synced) `*sde.Store`
-- safe here because `NewServer` only stores the pointer for later handler use; it never calls an
accessor (so never syncs) during construction/registration:
```go
func connectedSession(t *testing.T, store *config.Store) (*mcpsdk.ClientSession, *mcpsdk.ServerSession) {
	t.Helper()
	sdeStore, err := sde.Open(filepath.Join(t.TempDir(), "sde.db"))
	if err != nil {
		t.Fatalf("sde.Open: %v", err)
	}
	t.Cleanup(func() { sdeStore.Close() })
	server := mcp.NewServer("test", "client-id", store, sdeStore)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() {
		clientSession.Close()
		serverSession.Wait()
	})
	return clientSession, serverSession
}
```
Add `"path/filepath"` and `"github.com/rusher2004/new-eden-agent-toolkit/sde"` to this file's
imports.

- [ ] **Step 11: Run tests to verify they pass**

Run: `go test ./mcp/... -v`
Expected: PASS — including `TestServer_AllToolsRegistered` (tool set unchanged, only re-sourced)
and `TestServer_RemoveCharacterNotFound_SurfacesAsToolError` (unaffected by this task).

- [ ] **Step 12: Wire `commands/mcp.go`'s real call site**

Modify `commands/mcp.go` — add the `sde` import and open the store before constructing the server:
```go
import (
	// ... existing imports ...
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

func mcpCommand(version, clientID string) *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "run the neat MCP server (stdio transport)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store, err := config.NewStore()
			if err != nil {
				return fmt.Errorf("resolve config store: %w", err)
			}

			sdePath, err := sde.DefaultPath()
			if err != nil {
				return fmt.Errorf("resolve sde cache path: %w", err)
			}
			sdeStore, err := sde.Open(sdePath)
			if err != nil {
				return fmt.Errorf("open sde cache: %w", err)
			}
			defer sdeStore.Close()

			launcherPath, err := ensureLauncher()
			if err != nil {
				return fmt.Errorf("ensure launcher binary: %w", err)
			}

			if err := urlscheme.Register(launcherPath); err != nil {
				return fmt.Errorf("register URL scheme: %w", err)
			}

			server := mcp.NewServer(version, clientID, store, sdeStore)
			return server.Run(ctx, &mcpsdk.StdioTransport{})
		},
	}
}
```

- [ ] **Step 13: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: clean build/vet/gofmt, every test in the module passes under `-race`.

- [ ] **Step 14: Commit**

```bash
git add sde/sync.go sde/category.go sde/category_test.go sde/group.go sde/group_test.go sde/type.go sde/type_test.go mcp/universe.go mcp/universe_internal_test.go mcp/server.go mcp/server_test.go commands/mcp.go
git commit -m "feat: migrate types/groups/categories tools to sde/-backed read-through cache"
```

---

### Task 8: Migrate geography (regions/constellations/systems, 6 tools) to the read-through pattern

**Files:**
- Modify: `sde/region.go`, `sde/region_test.go` (add `RegionIDs`)
- Modify: `sde/constellation.go`, `sde/constellation_test.go` (add `ConstellationIDs`)
- Modify: `sde/system.go`, `sde/system_test.go` (add `SystemIDs`)
- Modify: `mcp/universe.go` (rewrite the 6 handlers/vars to read-through)
- Modify: `mcp/universe_internal_test.go` (rewrite the 6 tools' tests)
- Modify: `mcp/server.go` (6 registration call sites pass the already-threaded `sdeStore`)

**Interfaces:**
- Consumes: `sde.Region`/`sde.Constellation`/`sde.System` and their `Store` accessors/`Put*` methods
  (Tasks 1/4); `sde.Position` (Task 1); `esi.Client.UniverseRegionDetail`/
  `UniverseConstellationDetail`/`UniverseSystemDetail`/`UniverseRegions`/`UniverseConstellations`/
  `UniverseSystems` (unchanged, `esi/universe.go`); `NewServer(version, clientID string, store
  *config.Store, sdeStore *sde.Store)` (Task 7 — unchanged signature, no new parameter needed).
- Produces (for consistency with Task 9, same pattern): `(s *Store) RegionIDs(ctx, userAgent)
  ([]int64, error)`, `ConstellationIDs`, `SystemIDs` (same "empty slice, not an error, if never
  synced" contract as Task 7's `CategoryIDs`/`GroupIDs`/`TypeIDs`).

None of these 3 list tools are paginated (verified: `list_universe_regions`/`_constellations`/
`_systems` have no `page` input, unlike groups/types) — `RegionIDs`/`ConstellationIDs`/`SystemIDs`
are returned whole on a cache hit, exactly like `list_universe_categories` in Task 7.

**Field-drop nuance, consistent with Task 7's policy:** `get_universe_system`'s `Planets` field has
no SDE source (would need a full `mapPlanets.jsonl` pass for two nested ID arrays — confirmed not
worth it during design). Same treatment as `get_universe_type`'s `PackagedVolume`: nil on a cache
hit, but returned in full on a cache miss, since the live ESI response already has it for free.

**`sde.Position` vs `mcp.Position`:** these are two distinct types in two different packages (both
named `Position`, both `{X, Y, Z float64}`) — every read-through mapping below converts explicitly
between them field-by-field; there is no shared/aliased type.

- [ ] **Step 1: Write the failing tests for `sde/`'s three new ID-list accessors**

Modify `sde/region_test.go` — add after `TestPutRegion_ThenRegion_RoundTrips`:
```go

func TestRegionIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000002, `{"_key": 10000002, "constellationIDs": [], "name": {"en": "B"}}`)
	putRecord(t, store, "regions", 10000001, `{"_key": 10000001, "constellationIDs": [], "name": {"en": "A"}}`)

	ids, err := store.RegionIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("RegionIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10000001 || ids[1] != 10000002 {
		t.Errorf("RegionIDs = %v, want [10000001 10000002] (ascending)", ids)
	}
}

func TestRegionIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.RegionIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("RegionIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("RegionIDs = %v, want empty", ids)
	}
}
```

Modify `sde/constellation_test.go` — add the analogous pair:
```go

func TestConstellationIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "constellations", 20000001, `{"_key": 20000001, "name": {"en": "San Matar"}, "position": {"x": 0, "y": 0, "z": 0}, "regionID": 10000001, "solarSystemIDs": []}`)

	ids, err := store.ConstellationIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("ConstellationIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 20000001 {
		t.Errorf("ConstellationIDs = %v, want [20000001]", ids)
	}
}

func TestConstellationIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.ConstellationIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("ConstellationIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("ConstellationIDs = %v, want empty", ids)
	}
}
```

Modify `sde/system_test.go` — add the analogous pair:
```go

func TestSystemIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000001, `{"_key": 30000001, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.85}`)

	ids, err := store.SystemIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("SystemIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 30000001 {
		t.Errorf("SystemIDs = %v, want [30000001]", ids)
	}
}

func TestSystemIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.SystemIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("SystemIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("SystemIDs = %v, want empty", ids)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sde/... -run 'TestRegionIDs|TestConstellationIDs|TestSystemIDs' -v`
Expected: FAIL to compile — `store.RegionIDs undefined`, `store.ConstellationIDs undefined`,
`store.SystemIDs undefined`.

- [ ] **Step 3: Implement the three accessors**

Modify `sde/region.go` — add at the end of the file:
```go

// RegionIDs returns every region ID currently cached. See sde/category.go's
// CategoryIDs for why no separate sort is needed and what an empty result
// means.
func (s *Store) RegionIDs(ctx context.Context, userAgent string) ([]int64, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("regions"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, int64FromKey(k))
			return nil
		})
	})
	return ids, err
}
```

Modify `sde/constellation.go` — add at the end of the file:
```go

// ConstellationIDs returns every constellation ID currently cached.
func (s *Store) ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("constellations"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, int64FromKey(k))
			return nil
		})
	})
	return ids, err
}
```

Modify `sde/system.go` — add at the end of the file:
```go

// SystemIDs returns every solar system ID currently cached.
func (s *Store) SystemIDs(ctx context.Context, userAgent string) ([]int64, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("systems"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, int64FromKey(k))
			return nil
		})
	})
	return ids, err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./sde/... -run 'TestRegionIDs|TestConstellationIDs|TestSystemIDs' -v`
Expected: PASS.

- [ ] **Step 5: Commit the `sde/` additions**

```bash
git add sde/region.go sde/region_test.go sde/constellation.go sde/constellation_test.go sde/system.go sde/system_test.go
git commit -m "feat: add RegionIDs/ConstellationIDs/SystemIDs bucket-enumeration accessors"
```

- [ ] **Step 6: Write the failing tests for the 6 migrated `mcp/universe.go` tools**

Modify `mcp/universe_internal_test.go` — replace every existing test function named
`TestListUniverseRegionsHandler_*`, `TestGetUniverseRegionHandler_*`,
`TestListUniverseConstellationsHandler_*`, `TestGetUniverseConstellationHandler_*`,
`TestListUniverseSystemsHandler_*`, `TestGetUniverseSystemHandler_*` with the versions below,
following exactly the same shape Task 7 established (fake cache types substituting for `*sde.Store`
to avoid live network I/O in package `mcp`'s tests, handler-level tests keep swapping the whole
var wholesale with an added unused `sdeStore` parameter). Add the three fake-cache types:
```go

type fakeRegionCache struct {
	region    sde.Region
	regionOK  bool
	regionErr error
	ids       []int64
	idsErr    error
	putErr    error
	putCalls  []sde.Region
}

func (f *fakeRegionCache) Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error) {
	return f.region, f.regionOK, f.regionErr
}
func (f *fakeRegionCache) PutRegion(r sde.Region) error {
	f.putCalls = append(f.putCalls, r)
	return f.putErr
}
func (f *fakeRegionCache) RegionIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeConstellationCache struct {
	constellation   sde.Constellation
	constellationOK bool
	err             error
	ids             []int64
	idsErr          error
	putErr          error
	putCalls        []sde.Constellation
}

func (f *fakeConstellationCache) Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error) {
	return f.constellation, f.constellationOK, f.err
}
func (f *fakeConstellationCache) PutConstellation(c sde.Constellation) error {
	f.putCalls = append(f.putCalls, c)
	return f.putErr
}
func (f *fakeConstellationCache) ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeSystemCache struct {
	system   sde.System
	systemOK bool
	err      error
	ids      []int64
	idsErr   error
	putErr   error
	putCalls []sde.System
}

func (f *fakeSystemCache) System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error) {
	return f.system, f.systemOK, f.err
}
func (f *fakeSystemCache) PutSystem(s sde.System) error {
	f.putCalls = append(f.putCalls, s)
	return f.putErr
}
func (f *fakeSystemCache) SystemIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

func TestListUniverseRegions_CacheHit_ReturnsCachedIDs(t *testing.T) {
	cache := &fakeRegionCache{ids: []int64{10000001, 10000002}}

	ids, err := listUniverseRegions(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseRegions returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10000001 {
		t.Errorf("ids = %v, want [10000001 10000002]", ids)
	}
}

func TestGetUniverseRegion_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeRegionCache{
		region: sde.Region{
			ID: 10000001, Name: map[string]string{"en": "Derelik"},
			Description: map[string]string{"en": "A dangerous frontier region."}, ConstellationIDs: []int64{20000001},
		},
		regionOK: true,
	}

	got, err := getUniverseRegion(context.Background(), cache, "test-agent", 10000001)
	if err != nil {
		t.Fatalf("getUniverseRegion returned error: %v", err)
	}
	if got.RegionID != 10000001 || got.Name != "Derelik" || got.Description == nil || *got.Description != "A dangerous frontier region." {
		t.Errorf("got = %+v, want RegionID=10000001 Name=Derelik Description=\"A dangerous frontier region.\"", got)
	}
	if len(got.ConstellationIDs) != 1 || got.ConstellationIDs[0] != 20000001 {
		t.Errorf("ConstellationIDs = %v, want [20000001]", got.ConstellationIDs)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutRegion called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}

func TestGetUniverseRegion_CacheHit_NoDescription_ReturnsNilDescription(t *testing.T) {
	cache := &fakeRegionCache{
		region:   sde.Region{ID: 10000002, Name: map[string]string{"en": "No Description"}},
		regionOK: true,
	}

	got, err := getUniverseRegion(context.Background(), cache, "test-agent", 10000002)
	if err != nil {
		t.Fatalf("getUniverseRegion returned error: %v", err)
	}
	if got.Description != nil {
		t.Errorf("Description = %v, want nil", got.Description)
	}
}

func TestGetUniverseConstellation_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeConstellationCache{
		constellation: sde.Constellation{
			ID: 20000001, RegionID: 10000001, Name: map[string]string{"en": "San Matar"},
			Position: sde.Position{X: 1, Y: 2, Z: 3}, SystemIDs: []int64{30000001},
		},
		constellationOK: true,
	}

	got, err := getUniverseConstellation(context.Background(), cache, "test-agent", 20000001)
	if err != nil {
		t.Fatalf("getUniverseConstellation returned error: %v", err)
	}
	if got.ConstellationID != 20000001 || got.RegionID != 10000001 || got.Name != "San Matar" {
		t.Errorf("got = %+v, want ConstellationID=20000001 RegionID=10000001 Name=San Matar", got)
	}
	if got.Position != (Position{X: 1, Y: 2, Z: 3}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
	if len(got.SystemIDs) != 1 || got.SystemIDs[0] != 30000001 {
		t.Errorf("SystemIDs = %v, want [30000001]", got.SystemIDs)
	}
}

func TestGetUniverseSystem_CacheHit_MapsFieldsAndDropsPlanets(t *testing.T) {
	secClass := "B"
	starID := int64(40000001)
	cache := &fakeSystemCache{
		system: sde.System{
			ID: 30000001, ConstellationID: 20000001, Name: map[string]string{"en": "Tanoo"},
			Position: sde.Position{X: 1, Y: 2, Z: 3}, SecurityStatus: 0.85, SecurityClass: &secClass,
			StarID: &starID, StargateIDs: []int64{50000056}, StationIDs: []int64{60000001},
		},
		systemOK: true,
	}

	got, err := getUniverseSystem(context.Background(), cache, "test-agent", 30000001)
	if err != nil {
		t.Fatalf("getUniverseSystem returned error: %v", err)
	}
	if got.SystemID != 30000001 || got.ConstellationID != 20000001 || got.Name != "Tanoo" || got.SecurityStatus != 0.85 {
		t.Errorf("got = %+v, want SystemID=30000001 ConstellationID=20000001 Name=Tanoo SecurityStatus=0.85", got)
	}
	if len(got.StargateIDs) != 1 || got.StargateIDs[0] != 50000056 {
		t.Errorf("StargateIDs = %v, want [50000056]", got.StargateIDs)
	}
	if len(got.StationIDs) != 1 || got.StationIDs[0] != 60000001 {
		t.Errorf("StationIDs = %v, want [60000001]", got.StationIDs)
	}
	if got.Planets != nil {
		t.Errorf("Planets = %v, want nil on a cache hit (no SDE source)", got.Planets)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutSystem called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}
```

Update every pre-existing `TestListUniverseRegionsHandler_*`/`TestGetUniverseRegionHandler_*`/
`TestListUniverseConstellationsHandler_*`/`TestGetUniverseConstellationHandler_*`/
`TestListUniverseSystemsHandler_*`/`TestGetUniverseSystemHandler_*` test the same mechanical way
Task 7 updated the categories/groups/types handler tests: add the unused `sdeStore` parameter
(typed `regionCache`/`constellationCache`/`systemCache`) to the stubbed var's signature, and pass
`nil` as the new second argument to the `...Handler("test", ...)` call under test.

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListUniverseRegions|TestGetUniverseRegion|TestListUniverseConstellations|TestGetUniverseConstellation|TestListUniverseSystems|TestGetUniverseSystem' -v`
Expected: FAIL to compile — `regionCache`/`constellationCache`/`systemCache` undefined, wrong
number of arguments to `listUniverseRegions`/etc.

- [ ] **Step 8: Rewrite `mcp/universe.go`'s 6 handlers to the read-through pattern**

Modify `mcp/universe.go` — replace the code from the `// --- list_universe_regions (public, no
input) ---` comment through the end of `getUniverseSystemHandler` (everything at
`mcp/universe.go:403-592` before this task, i.e. everything between the shared `Position` struct
and `// --- get_universe_asteroid_belt (public) ---`) with:
```go
// --- list_universe_regions (public, no input) ---

type regionCache interface {
	Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error)
	PutRegion(sde.Region) error
	RegionIDs(ctx context.Context, userAgent string) ([]int64, error)
}

var listUniverseRegions = func(ctx context.Context, sdeStore regionCache, userAgent string) ([]int64, error) {
	ids, err := sdeStore.RegionIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	client := esi.NewPublicClient(userAgent)
	return client.UniverseRegions(ctx)
}

type ListUniverseRegionsInput struct{}
type ListUniverseRegionsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseRegionsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseRegionsInput, ListUniverseRegionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseRegionsInput) (*mcpsdk.CallToolResult, ListUniverseRegionsOutput, error) {
		ids, err := listUniverseRegions(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseRegionsOutput{}, fmt.Errorf("list universe regions: %w", err)
		}
		return nil, ListUniverseRegionsOutput{IDs: ids}, nil
	}
}

// --- get_universe_region (public) ---

type UniverseRegion struct {
	RegionID         int64   `json:"regionId"`
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	ConstellationIDs []int64 `json:"constellationIds"`
}

var getUniverseRegion = func(ctx context.Context, sdeStore regionCache, userAgent string, regionID int64) (UniverseRegion, error) {
	r, ok, err := sdeStore.Region(ctx, userAgent, regionID)
	if err != nil {
		return UniverseRegion{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseRegionDetail(ctx, regionID)
		if err != nil {
			return UniverseRegion{}, err
		}
		r = sde.Region{ID: detail.RegionId, Name: map[string]string{"en": detail.Name}, ConstellationIDs: detail.Constellations}
		if detail.Description != nil {
			r.Description = map[string]string{"en": *detail.Description}
		}
		if err := sdeStore.PutRegion(r); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through region into cache", "error", err)
		}
	}
	var description *string
	if v, ok := r.Description["en"]; ok {
		description = &v
	}
	return UniverseRegion{RegionID: r.ID, Name: r.Name["en"], Description: description, ConstellationIDs: r.ConstellationIDs}, nil
}

type GetUniverseRegionInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
}

func getUniverseRegionHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseRegionInput, UniverseRegion] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseRegionInput) (*mcpsdk.CallToolResult, UniverseRegion, error) {
		out, err := getUniverseRegion(ctx, sdeStore, sso.UserAgent(version), in.RegionID)
		if err != nil {
			return nil, UniverseRegion{}, fmt.Errorf("get universe region: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_constellations (public, no input) ---

type constellationCache interface {
	Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error)
	PutConstellation(sde.Constellation) error
	ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error)
}

var listUniverseConstellations = func(ctx context.Context, sdeStore constellationCache, userAgent string) ([]int64, error) {
	ids, err := sdeStore.ConstellationIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	client := esi.NewPublicClient(userAgent)
	return client.UniverseConstellations(ctx)
}

type ListUniverseConstellationsInput struct{}
type ListUniverseConstellationsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseConstellationsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseConstellationsInput, ListUniverseConstellationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseConstellationsInput) (*mcpsdk.CallToolResult, ListUniverseConstellationsOutput, error) {
		ids, err := listUniverseConstellations(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseConstellationsOutput{}, fmt.Errorf("list universe constellations: %w", err)
		}
		return nil, ListUniverseConstellationsOutput{IDs: ids}, nil
	}
}

// --- get_universe_constellation (public) ---

type UniverseConstellation struct {
	ConstellationID int64    `json:"constellationId"`
	RegionID        int64    `json:"regionId"`
	Name            string   `json:"name"`
	Position        Position `json:"position"`
	SystemIDs       []int64  `json:"systemIds"`
}

var getUniverseConstellation = func(ctx context.Context, sdeStore constellationCache, userAgent string, constellationID int64) (UniverseConstellation, error) {
	c, ok, err := sdeStore.Constellation(ctx, userAgent, constellationID)
	if err != nil {
		return UniverseConstellation{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseConstellationDetail(ctx, constellationID)
		if err != nil {
			return UniverseConstellation{}, err
		}
		c = sde.Constellation{
			ID: detail.ConstellationId, RegionID: detail.RegionId, Name: map[string]string{"en": detail.Name},
			Position:  sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			SystemIDs: detail.Systems,
		}
		if err := sdeStore.PutConstellation(c); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through constellation into cache", "error", err)
		}
	}
	return UniverseConstellation{
		ConstellationID: c.ID, RegionID: c.RegionID, Name: c.Name["en"],
		Position:  Position{X: c.Position.X, Y: c.Position.Y, Z: c.Position.Z},
		SystemIDs: c.SystemIDs,
	}, nil
}

type GetUniverseConstellationInput struct {
	ConstellationID int64 `json:"constellationId" jsonschema:"the constellation ID to look up"`
}

func getUniverseConstellationHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseConstellationInput, UniverseConstellation] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseConstellationInput) (*mcpsdk.CallToolResult, UniverseConstellation, error) {
		out, err := getUniverseConstellation(ctx, sdeStore, sso.UserAgent(version), in.ConstellationID)
		if err != nil {
			return nil, UniverseConstellation{}, fmt.Errorf("get universe constellation: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_systems (public, no input) ---

type systemCache interface {
	System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error)
	PutSystem(sde.System) error
	SystemIDs(ctx context.Context, userAgent string) ([]int64, error)
}

var listUniverseSystems = func(ctx context.Context, sdeStore systemCache, userAgent string) ([]int64, error) {
	ids, err := sdeStore.SystemIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	client := esi.NewPublicClient(userAgent)
	return client.UniverseSystems(ctx)
}

type ListUniverseSystemsInput struct{}
type ListUniverseSystemsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseSystemsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseSystemsInput, ListUniverseSystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseSystemsInput) (*mcpsdk.CallToolResult, ListUniverseSystemsOutput, error) {
		ids, err := listUniverseSystems(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseSystemsOutput{}, fmt.Errorf("list universe systems: %w", err)
		}
		return nil, ListUniverseSystemsOutput{IDs: ids}, nil
	}
}

// --- get_universe_system (public) ---

type UniverseSystemPlanet struct {
	PlanetID        int64   `json:"planetId"`
	AsteroidBeltIDs []int64 `json:"asteroidBeltIds,omitempty"`
	MoonIDs         []int64 `json:"moonIds,omitempty"`
}

type UniverseSystem struct {
	SystemID        int64                  `json:"systemId"`
	ConstellationID int64                  `json:"constellationId"`
	Name            string                 `json:"name"`
	Position        Position               `json:"position"`
	SecurityStatus  float64                `json:"securityStatus"`
	SecurityClass   *string                `json:"securityClass,omitempty"`
	StarID          *int64                 `json:"starId,omitempty"`
	StargateIDs     []int64                `json:"stargateIds,omitempty"`
	StationIDs      []int64                `json:"stationIds,omitempty"`
	Planets         []UniverseSystemPlanet `json:"planets,omitempty"`
}

var getUniverseSystem = func(ctx context.Context, sdeStore systemCache, userAgent string, systemID int64) (UniverseSystem, error) {
	sys, ok, err := sdeStore.System(ctx, userAgent, systemID)
	if err != nil {
		return UniverseSystem{}, fmt.Errorf("read sde cache: %w", err)
	}

	// Planets has no SDE source (would need a full mapPlanets.jsonl pass for
	// two nested ID arrays -- confirmed not worth it during design) -- only
	// available fresh off a cache-miss ESI call, same treatment as
	// get_universe_type's PackagedVolume in Task 7.
	var planets []UniverseSystemPlanet
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseSystemDetail(ctx, systemID)
		if err != nil {
			return UniverseSystem{}, err
		}
		sys = sde.System{
			ID: detail.SystemId, ConstellationID: detail.ConstellationId, Name: map[string]string{"en": detail.Name},
			Position:       sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			SecurityStatus: detail.SecurityStatus, SecurityClass: detail.SecurityClass, StarID: detail.StarId,
			StargateIDs: detail.Stargates, StationIDs: detail.Stations,
		}
		if len(detail.Planets) > 0 {
			planets = make([]UniverseSystemPlanet, len(detail.Planets))
			for i, p := range detail.Planets {
				planets[i] = UniverseSystemPlanet{PlanetID: p.PlanetId, AsteroidBeltIDs: p.AsteroidBelts, MoonIDs: p.Moons}
			}
		}
		if err := sdeStore.PutSystem(sys); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through system into cache", "error", err)
		}
	}

	return UniverseSystem{
		SystemID: sys.ID, ConstellationID: sys.ConstellationID, Name: sys.Name["en"],
		Position:       Position{X: sys.Position.X, Y: sys.Position.Y, Z: sys.Position.Z},
		SecurityStatus: sys.SecurityStatus, SecurityClass: sys.SecurityClass, StarID: sys.StarID,
		StargateIDs: sys.StargateIDs, StationIDs: sys.StationIDs, Planets: planets,
	}, nil
}

type GetUniverseSystemInput struct {
	SystemID int64 `json:"systemId" jsonschema:"the solar system ID to look up"`
}

func getUniverseSystemHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseSystemInput, UniverseSystem] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseSystemInput) (*mcpsdk.CallToolResult, UniverseSystem, error) {
		out, err := getUniverseSystem(ctx, sdeStore, sso.UserAgent(version), in.SystemID)
		if err != nil {
			return nil, UniverseSystem{}, fmt.Errorf("get universe system: %w", err)
		}
		return nil, out, nil
	}
}
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestListUniverseRegions|TestGetUniverseRegion|TestListUniverseConstellations|TestGetUniverseConstellation|TestListUniverseSystems|TestGetUniverseSystem' -v`
Expected: PASS.

- [ ] **Step 10: Wire this task's 6 registration call sites**

Modify `mcp/server.go` — the 6 registration call sites for this task's tools (unchanged
`Name`/`Description`, only the handler constructor call changes):
```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_regions",
		Description: "List every region ID (public data, no character required).",
	}, listUniverseRegionsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_region",
		Description: "Get a region's name, description, and member constellation IDs (public data, no character required).",
	}, getUniverseRegionHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_constellations",
		Description: "List every constellation ID (public data, no character required).",
	}, listUniverseConstellationsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_constellation",
		Description: "Get a constellation's name, region, position, and member system IDs (public data, no character required).",
	}, getUniverseConstellationHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_systems",
		Description: "List every solar system ID (public data, no character required).",
	}, listUniverseSystemsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_system",
		Description: "Get a solar system's name, constellation, position, security status/class, star, stargates, stations, and planets (public data, no character required).",
	}, getUniverseSystemHandler(version, sdeStore))
```

- [ ] **Step 11: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: clean build/vet/gofmt, every test in the module passes under `-race`.

- [ ] **Step 12: Commit**

```bash
git add sde/region.go sde/region_test.go sde/constellation.go sde/constellation_test.go sde/system.go sde/system_test.go mcp/universe.go mcp/universe_internal_test.go mcp/server.go
git commit -m "feat: migrate geography tools to sde/-backed read-through cache"
```

---

### Task 9: Migrate reference lists (ancestries/bloodlines/races/factions/graphics, 6 tools)

**Files:**
- Modify: `sde/graphic.go`, `sde/graphic_test.go` (add `GraphicIDs`)
- Modify: `mcp/universe.go` (rewrite the 6 handlers/vars to read-through)
- Modify: `mcp/universe_internal_test.go` (rewrite the 6 tools' tests)
- Modify: `mcp/server.go` (6 registration call sites pass the already-threaded `sdeStore`)

**Interfaces:**
- Consumes: `sde.Ancestry`/`sde.Bloodline`/`sde.Race`/`sde.Faction` and their whole-collection
  `Store` accessors (Task 5); `sde.Graphic` and its `Store` accessor/`Put*` (Task 1, corrected this
  task's design note below); `esi.Client.UniverseAncestries`/`UniverseBloodlines`/`UniverseRaces`/
  `UniverseFactions`/`UniverseGraphics`/`UniverseGraphicDetail` (unchanged, `esi/universe.go`).
- Produces: `(s *Store) GraphicIDs(ctx, userAgent) ([]int64, error)` (same contract as Tasks 7/8's
  ID-list accessors).

**Corrected finding — `sde/graphic.go` needed a fix before this task could give
`get_universe_graphic` full-fidelity coverage:** a full 6051-record scan of the live
`graphics.jsonl` (not a sample) shows its complete key set is `_key, graphicFile, iconFolder,
sofFactionName, sofHullName, sofLayout, sofMaterialSetID, sofRaceName`. Task 1, as originally
written, only modeled `GraphicFile` on `Graphic` — three real fields (`IconFolder`,
`SofFactionName`, `SofHullName`, `SofRaceName` is a fourth already covered) were missing SDE
coverage they should have had. That gap is fixed directly in Task 1's plan text (struct,
`graphicRecord`, `Graphic()`/`PutGraphic()`, and tests all updated) rather than repeated or worked
around here. Genuinely absent from the archive (zero hits across all 6051 records, confirmed, not
a sample coincidence): `CollisionFile`, `SofDNA` — these two get the same "returned on a cache
miss, dropped on a cache hit" treatment as every other field drop in this migration.

**No `Put*` for ancestries/bloodlines/races/factions**, per the design: these are list-only (no
`get_universe_ancestry`/`_bloodline`/`_race`/`_faction` tool exists), so a cache miss (bucket never
synced) falls back to one live ESI list call and returns it directly — no write-back, matching
Task 5's decision that these are "small, complete, rarely-changing datasets" where write-back isn't
worth the complexity.

**`Bloodline.ShipTypeID`/`Race.AllianceID` on a cache hit:** `sde.Bloodline.ShipTypeID` is `*int64`
(Task 5 modeled it defensively); the existing `UniverseBloodline.ShipTypeID` is a non-optional
`int64` — a nil pointer maps to the zero value `0` on a cache hit (there is no live ESI call to
backfill it in the list-tools path, unlike the per-ID tools' miss path). `UniverseRace.AllianceID`
is the confirmed, permanent field drop from the design spec's field-drop table — `0` on every cache
hit, populated for real only when the never-synced fallback calls ESI directly.

- [ ] **Step 1: Write the failing test for `GraphicIDs`**

Modify `sde/graphic_test.go` — add after `TestPutGraphic_ThenGraphic_RoundTrips`:
```go

func TestGraphicIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 38, `{"_key": 38, "sofRaceName": "caldari"}`)
	putRecord(t, store, "graphics", 10, `{"_key": 10, "graphicFile": "res:/x"}`)

	ids, err := store.GraphicIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GraphicIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 38 {
		t.Errorf("GraphicIDs = %v, want [10 38] (ascending)", ids)
	}
}

func TestGraphicIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.GraphicIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GraphicIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("GraphicIDs = %v, want empty", ids)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run TestGraphicIDs -v`
Expected: FAIL to compile — `store.GraphicIDs undefined`.

- [ ] **Step 3: Implement the accessor**

Modify `sde/graphic.go` — add at the end of the file:
```go

// GraphicIDs returns every graphic ID currently cached. See
// sde/category.go's CategoryIDs for why no separate sort is needed and
// what an empty result means.
func (s *Store) GraphicIDs(ctx context.Context, userAgent string) ([]int64, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("graphics"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			ids = append(ids, int64FromKey(k))
			return nil
		})
	})
	return ids, err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -run TestGraphicIDs -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add sde/graphic.go sde/graphic_test.go
git commit -m "feat: add GraphicIDs bucket-enumeration accessor"
```

- [ ] **Step 6: Write the failing tests for the 6 migrated `mcp/universe.go` tools**

Modify `mcp/universe_internal_test.go` — replace every existing test function named
`TestListUniverseAncestriesHandler_*`, `TestListUniverseBloodlinesHandler_*`,
`TestListUniverseRacesHandler_*`, `TestListUniverseFactionsHandler_*`,
`TestListUniverseGraphicsHandler_*`, `TestGetUniverseGraphicHandler_*` with the versions below,
following exactly Task 7/8's shape. Add the five fake-cache types:
```go

type fakeAncestryCache struct {
	ancestries []sde.Ancestry
	err        error
}

func (f *fakeAncestryCache) Ancestries(ctx context.Context, userAgent string) ([]sde.Ancestry, error) {
	return f.ancestries, f.err
}

type fakeBloodlineCache struct {
	bloodlines []sde.Bloodline
	err        error
}

func (f *fakeBloodlineCache) Bloodlines(ctx context.Context, userAgent string) ([]sde.Bloodline, error) {
	return f.bloodlines, f.err
}

type fakeRaceCache struct {
	races []sde.Race
	err   error
}

func (f *fakeRaceCache) Races(ctx context.Context, userAgent string) ([]sde.Race, error) {
	return f.races, f.err
}

type fakeFactionCache struct {
	factions []sde.Faction
	err      error
}

func (f *fakeFactionCache) Factions(ctx context.Context, userAgent string) ([]sde.Faction, error) {
	return f.factions, f.err
}

type fakeGraphicCache struct {
	graphic   sde.Graphic
	graphicOK bool
	err       error
	ids       []int64
	idsErr    error
	putErr    error
	putCalls  []sde.Graphic
}

func (f *fakeGraphicCache) Graphic(ctx context.Context, userAgent string, id int64) (sde.Graphic, bool, error) {
	return f.graphic, f.graphicOK, f.err
}
func (f *fakeGraphicCache) PutGraphic(g sde.Graphic) error {
	f.putCalls = append(f.putCalls, g)
	return f.putErr
}
func (f *fakeGraphicCache) GraphicIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

func TestListUniverseAncestries_CacheHit_MapsFields(t *testing.T) {
	iconID := int64(1641)
	shortDesc := "Progressive members of the upper class."
	cache := &fakeAncestryCache{ancestries: []sde.Ancestry{
		{ID: 1, BloodlineID: 5, Description: map[string]string{"en": "Holders are..."}, Name: map[string]string{"en": "Liberal Holders"}, IconID: &iconID, ShortDescription: &shortDesc},
	}}

	got, err := listUniverseAncestries(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseAncestries returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].Name != "Liberal Holders" || got[0].Description != "Holders are..." {
		t.Errorf("got = %+v, want one entry ID=1 Name=\"Liberal Holders\" Description=\"Holders are...\"", got)
	}
	if got[0].ShortDescription == nil || *got[0].ShortDescription != shortDesc {
		t.Errorf("ShortDescription = %v, want %q", got[0].ShortDescription, shortDesc)
	}
}

func TestListUniverseAncestries_CacheError_ReturnsError(t *testing.T) {
	cache := &fakeAncestryCache{err: errors.New("boom")}

	_, err := listUniverseAncestries(context.Background(), cache, "test-agent")
	if err == nil {
		t.Fatal("listUniverseAncestries returned nil error, want the wrapped cache error")
	}
}

func TestListUniverseBloodlines_CacheHit_MapsFieldsAndDefaultsMissingShipTypeID(t *testing.T) {
	cache := &fakeBloodlineCache{bloodlines: []sde.Bloodline{
		{ID: 1, Charisma: 6, CorporationID: 1000006, Description: map[string]string{"en": "The Deteis are..."}, Intelligence: 7, Memory: 7, Name: map[string]string{"en": "Deteis"}, Perception: 5, RaceID: 1, Willpower: 5},
	}}

	got, err := listUniverseBloodlines(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseBloodlines returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].Name != "Deteis" || got[0].ShipTypeID != 0 {
		t.Errorf("got = %+v, want one entry ID=1 Name=Deteis ShipTypeID=0 (no SDE source for this record)", got)
	}
}

func TestListUniverseRaces_CacheHit_DropsAllianceID(t *testing.T) {
	cache := &fakeRaceCache{races: []sde.Race{
		{ID: 1, Description: map[string]string{"en": "Founded on the tenets..."}, Name: map[string]string{"en": "Caldari"}},
	}}

	got, err := listUniverseRaces(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseRaces returned error: %v", err)
	}
	if len(got) != 1 || got[0].RaceID != 1 || got[0].Name != "Caldari" || got[0].AllianceID != 0 {
		t.Errorf("got = %+v, want one entry RaceID=1 Name=Caldari AllianceID=0 (no SDE source)", got)
	}
}

func TestListUniverseFactions_CacheHit_MapsFieldsIncludingStationCounts(t *testing.T) {
	corpID := int64(1000035)
	cache := &fakeFactionCache{factions: []sde.Faction{
		{ID: 500001, Name: map[string]string{"en": "Caldari State"}, Description: map[string]string{"en": "The Caldari State..."}, IsUnique: true, SizeFactor: 5.0, CorporationID: &corpID, StationCount: 120, StationSystemCount: 45},
	}}

	got, err := listUniverseFactions(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseFactions returned error: %v", err)
	}
	if len(got) != 1 || got[0].FactionID != 500001 || got[0].StationCount != 120 || got[0].StationSystemCount != 45 {
		t.Errorf("got = %+v, want one entry FactionID=500001 StationCount=120 StationSystemCount=45", got)
	}
}

func TestListUniverseGraphics_CacheHit_ReturnsCachedIDs(t *testing.T) {
	cache := &fakeGraphicCache{ids: []int64{10, 38}}

	ids, err := listUniverseGraphics(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseGraphics returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 {
		t.Errorf("ids = %v, want [10 38]", ids)
	}
}

func TestGetUniverseGraphic_CacheHit_MapsFieldsAndDropsCollisionFileAndSofDNA(t *testing.T) {
	iconFolder := "res:/dx9/model/ship/caldari/frigate/cf1/icons"
	factionName := "caldaribase"
	cache := &fakeGraphicCache{
		graphic:   sde.Graphic{ID: 38, IconFolder: &iconFolder, SofFactionName: &factionName},
		graphicOK: true,
	}

	got, err := getUniverseGraphic(context.Background(), cache, "test-agent", 38)
	if err != nil {
		t.Fatalf("getUniverseGraphic returned error: %v", err)
	}
	if got.GraphicID != 38 || got.IconFolder == nil || *got.IconFolder != iconFolder {
		t.Errorf("got = %+v, want GraphicID=38 IconFolder=%q", got, iconFolder)
	}
	if got.CollisionFile != nil || got.SofDNA != nil {
		t.Errorf("CollisionFile/SofDNA = %v/%v, want nil/nil on a cache hit (no SDE source)", got.CollisionFile, got.SofDNA)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutGraphic called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}
```

Update every pre-existing handler-level test for these 6 tools the same mechanical way Tasks 7/8
did: add the unused `sdeStore` parameter (typed `ancestryCache`/`bloodlineCache`/`raceCache`/
`factionCache`/`graphicCache`) to the stubbed var's signature, and pass `nil` as the new second
argument to the `...Handler("test", ...)` call under test.

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListUniverseAncestries|TestListUniverseBloodlines|TestListUniverseRaces|TestListUniverseFactions|TestListUniverseGraphics|TestGetUniverseGraphic' -v`
Expected: FAIL to compile.

- [ ] **Step 8: Rewrite `mcp/universe.go`'s 6 handlers to the read-through pattern**

Modify `mcp/universe.go` — replace the code from the `// --- list_universe_ancestries (public, no
input) ---` comment through the end of `TestListUniverseFactionsHandler`'s handler (i.e. the whole
`mcp/universe.go:14-186` block before this task — ancestries, bloodlines, races, factions), and
separately the `// --- list_universe_graphics (public, no input) ---` through the end of
`getUniverseGraphicHandler` block (`mcp/universe.go:818-877` before this task), with:
```go
// --- list_universe_ancestries (public, no input) ---

type UniverseAncestry struct {
	ID               int64   `json:"id"`
	BloodlineID      int64   `json:"bloodlineId"`
	Description      string  `json:"description"`
	Name             string  `json:"name"`
	IconID           *int64  `json:"iconId,omitempty"`
	ShortDescription *string `json:"shortDescription,omitempty"`
}

type ancestryCache interface {
	Ancestries(ctx context.Context, userAgent string) ([]sde.Ancestry, error)
}

var listUniverseAncestries = func(ctx context.Context, sdeStore ancestryCache, userAgent string) ([]UniverseAncestry, error) {
	ancestries, err := sdeStore.Ancestries(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ancestries) == 0 {
		client := esi.NewPublicClient(userAgent)
		fetched, err := client.UniverseAncestries(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseAncestry, len(fetched))
		for i, a := range fetched {
			out[i] = UniverseAncestry{ID: a.Id, BloodlineID: a.BloodlineId, Description: a.Description, Name: a.Name, IconID: a.IconId, ShortDescription: a.ShortDescription}
		}
		return out, nil
	}
	out := make([]UniverseAncestry, len(ancestries))
	for i, a := range ancestries {
		out[i] = UniverseAncestry{ID: a.ID, BloodlineID: a.BloodlineID, Description: a.Description["en"], Name: a.Name["en"], IconID: a.IconID, ShortDescription: a.ShortDescription}
	}
	return out, nil
}

type ListUniverseAncestriesInput struct{}
type ListUniverseAncestriesOutput struct {
	Ancestries []UniverseAncestry `json:"ancestries"`
}

func listUniverseAncestriesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseAncestriesInput, ListUniverseAncestriesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseAncestriesInput) (*mcpsdk.CallToolResult, ListUniverseAncestriesOutput, error) {
		ancestries, err := listUniverseAncestries(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseAncestriesOutput{}, fmt.Errorf("list universe ancestries: %w", err)
		}
		return nil, ListUniverseAncestriesOutput{Ancestries: ancestries}, nil
	}
}

// --- list_universe_bloodlines (public, no input) ---

type UniverseBloodline struct {
	ID            int64  `json:"id"`
	Charisma      int64  `json:"charisma"`
	CorporationID int64  `json:"corporationId"`
	Description   string `json:"description"`
	Intelligence  int64  `json:"intelligence"`
	Memory        int64  `json:"memory"`
	Name          string `json:"name"`
	Perception    int64  `json:"perception"`
	RaceID        int64  `json:"raceId"`
	ShipTypeID    int64  `json:"shipTypeId"`
	Willpower     int64  `json:"willpower"`
}

type bloodlineCache interface {
	Bloodlines(ctx context.Context, userAgent string) ([]sde.Bloodline, error)
}

var listUniverseBloodlines = func(ctx context.Context, sdeStore bloodlineCache, userAgent string) ([]UniverseBloodline, error) {
	bloodlines, err := sdeStore.Bloodlines(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(bloodlines) == 0 {
		client := esi.NewPublicClient(userAgent)
		fetched, err := client.UniverseBloodlines(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseBloodline, len(fetched))
		for i, b := range fetched {
			out[i] = UniverseBloodline{
				ID: b.BloodlineId, Charisma: b.Charisma, CorporationID: b.CorporationId, Description: b.Description,
				Intelligence: b.Intelligence, Memory: b.Memory, Name: b.Name, Perception: b.Perception,
				RaceID: b.RaceId, ShipTypeID: b.ShipTypeId, Willpower: b.Willpower,
			}
		}
		return out, nil
	}
	out := make([]UniverseBloodline, len(bloodlines))
	for i, b := range bloodlines {
		var shipTypeID int64
		if b.ShipTypeID != nil {
			shipTypeID = *b.ShipTypeID
		}
		out[i] = UniverseBloodline{
			ID: b.ID, Charisma: b.Charisma, CorporationID: b.CorporationID, Description: b.Description["en"],
			Intelligence: b.Intelligence, Memory: b.Memory, Name: b.Name["en"], Perception: b.Perception,
			RaceID: b.RaceID, ShipTypeID: shipTypeID, Willpower: b.Willpower,
		}
	}
	return out, nil
}

type ListUniverseBloodlinesInput struct{}
type ListUniverseBloodlinesOutput struct {
	Bloodlines []UniverseBloodline `json:"bloodlines"`
}

func listUniverseBloodlinesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseBloodlinesInput, ListUniverseBloodlinesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseBloodlinesInput) (*mcpsdk.CallToolResult, ListUniverseBloodlinesOutput, error) {
		bloodlines, err := listUniverseBloodlines(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseBloodlinesOutput{}, fmt.Errorf("list universe bloodlines: %w", err)
		}
		return nil, ListUniverseBloodlinesOutput{Bloodlines: bloodlines}, nil
	}
}

// --- list_universe_races (public, no input) ---

type UniverseRace struct {
	RaceID      int64  `json:"raceId"`
	AllianceID  int64  `json:"allianceId"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

type raceCache interface {
	Races(ctx context.Context, userAgent string) ([]sde.Race, error)
}

var listUniverseRaces = func(ctx context.Context, sdeStore raceCache, userAgent string) ([]UniverseRace, error) {
	races, err := sdeStore.Races(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(races) == 0 {
		client := esi.NewPublicClient(userAgent)
		fetched, err := client.UniverseRaces(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseRace, len(fetched))
		for i, r := range fetched {
			out[i] = UniverseRace{RaceID: r.RaceId, AllianceID: r.AllianceId, Description: r.Description, Name: r.Name}
		}
		return out, nil
	}
	out := make([]UniverseRace, len(races))
	for i, r := range races {
		// AllianceID has no SDE source at all (Task 5 -- confirmed absent
		// from races.jsonl, and there's no alliances.jsonl in the archive)
		// -- only available fresh off the ESI fallback above, dropped (left
		// 0) on this cache-hit path. This is the confirmed, permanent field
		// drop from the design spec's field-drop table.
		out[i] = UniverseRace{RaceID: r.ID, Description: r.Description["en"], Name: r.Name["en"]}
	}
	return out, nil
}

type ListUniverseRacesInput struct{}
type ListUniverseRacesOutput struct {
	Races []UniverseRace `json:"races"`
}

func listUniverseRacesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseRacesInput, ListUniverseRacesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseRacesInput) (*mcpsdk.CallToolResult, ListUniverseRacesOutput, error) {
		races, err := listUniverseRaces(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseRacesOutput{}, fmt.Errorf("list universe races: %w", err)
		}
		return nil, ListUniverseRacesOutput{Races: races}, nil
	}
}

// --- list_universe_factions (public, no input) ---

type UniverseFaction struct {
	FactionID            int64   `json:"factionId"`
	Description          string  `json:"description"`
	Name                 string  `json:"name"`
	IsUnique             bool    `json:"isUnique"`
	SizeFactor           float64 `json:"sizeFactor"`
	StationCount         int64   `json:"stationCount"`
	StationSystemCount   int64   `json:"stationSystemCount"`
	CorporationID        *int64  `json:"corporationId,omitempty"`
	MilitiaCorporationID *int64  `json:"militiaCorporationId,omitempty"`
	SolarSystemID        *int64  `json:"solarSystemId,omitempty"`
}

type factionCache interface {
	Factions(ctx context.Context, userAgent string) ([]sde.Faction, error)
}

var listUniverseFactions = func(ctx context.Context, sdeStore factionCache, userAgent string) ([]UniverseFaction, error) {
	factions, err := sdeStore.Factions(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(factions) == 0 {
		client := esi.NewPublicClient(userAgent)
		fetched, err := client.UniverseFactions(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseFaction, len(fetched))
		for i, f := range fetched {
			out[i] = UniverseFaction{
				FactionID: f.FactionId, Description: f.Description, Name: f.Name, IsUnique: f.IsUnique,
				SizeFactor: f.SizeFactor, StationCount: f.StationCount, StationSystemCount: f.StationSystemCount,
				CorporationID: f.CorporationId, MilitiaCorporationID: f.MilitiaCorporationId, SolarSystemID: f.SolarSystemId,
			}
		}
		return out, nil
	}
	out := make([]UniverseFaction, len(factions))
	for i, f := range factions {
		out[i] = UniverseFaction{
			FactionID: f.ID, Description: f.Description["en"], Name: f.Name["en"], IsUnique: f.IsUnique,
			SizeFactor: f.SizeFactor, StationCount: f.StationCount, StationSystemCount: f.StationSystemCount,
			CorporationID: f.CorporationID, MilitiaCorporationID: f.MilitiaCorporationID, SolarSystemID: f.SolarSystemID,
		}
	}
	return out, nil
}

type ListUniverseFactionsInput struct{}
type ListUniverseFactionsOutput struct {
	Factions []UniverseFaction `json:"factions"`
}

func listUniverseFactionsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseFactionsInput, ListUniverseFactionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseFactionsInput) (*mcpsdk.CallToolResult, ListUniverseFactionsOutput, error) {
		factions, err := listUniverseFactions(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseFactionsOutput{}, fmt.Errorf("list universe factions: %w", err)
		}
		return nil, ListUniverseFactionsOutput{Factions: factions}, nil
	}
}
```

And, separately in the same file, replace the `list_universe_graphics`/`get_universe_graphic`
section with:
```go
// --- list_universe_graphics (public, no input) ---

type graphicCache interface {
	Graphic(ctx context.Context, userAgent string, id int64) (sde.Graphic, bool, error)
	PutGraphic(sde.Graphic) error
	GraphicIDs(ctx context.Context, userAgent string) ([]int64, error)
}

var listUniverseGraphics = func(ctx context.Context, sdeStore graphicCache, userAgent string) ([]int64, error) {
	ids, err := sdeStore.GraphicIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	client := esi.NewPublicClient(userAgent)
	return client.UniverseGraphics(ctx)
}

type ListUniverseGraphicsInput struct{}
type ListUniverseGraphicsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseGraphicsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListUniverseGraphicsInput, ListUniverseGraphicsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseGraphicsInput) (*mcpsdk.CallToolResult, ListUniverseGraphicsOutput, error) {
		ids, err := listUniverseGraphics(ctx, sdeStore, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseGraphicsOutput{}, fmt.Errorf("list universe graphics: %w", err)
		}
		return nil, ListUniverseGraphicsOutput{IDs: ids}, nil
	}
}

// --- get_universe_graphic (public) ---

type UniverseGraphic struct {
	GraphicID      int64   `json:"graphicId"`
	CollisionFile  *string `json:"collisionFile,omitempty"`
	GraphicFile    *string `json:"graphicFile,omitempty"`
	IconFolder     *string `json:"iconFolder,omitempty"`
	SofDNA         *string `json:"sofDna,omitempty"`
	SofFactionName *string `json:"sofFactionName,omitempty"`
	SofHullName    *string `json:"sofHullName,omitempty"`
	SofRaceName    *string `json:"sofRaceName,omitempty"`
}

var getUniverseGraphic = func(ctx context.Context, sdeStore graphicCache, userAgent string, graphicID int64) (UniverseGraphic, error) {
	g, ok, err := sdeStore.Graphic(ctx, userAgent, graphicID)
	if err != nil {
		return UniverseGraphic{}, fmt.Errorf("read sde cache: %w", err)
	}

	// CollisionFile/SofDNA have no SDE source (confirmed absent from all
	// 6051 graphics.jsonl records) -- only available fresh off a cache-miss
	// ESI call, same treatment as every other field drop in this migration.
	var collisionFile, sofDNA *string
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseGraphicDetail(ctx, graphicID)
		if err != nil {
			return UniverseGraphic{}, err
		}
		collisionFile, sofDNA = detail.CollisionFile, detail.SofDna
		g = sde.Graphic{
			ID: detail.GraphicId, GraphicFile: detail.GraphicFile, IconFolder: detail.IconFolder,
			SofFactionName: detail.SofFationName, SofHullName: detail.SofHullName, SofRaceName: detail.SofRaceName,
		}
		if err := sdeStore.PutGraphic(g); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through graphic into cache", "error", err)
		}
	}

	return UniverseGraphic{
		GraphicID: g.ID, CollisionFile: collisionFile, GraphicFile: g.GraphicFile, IconFolder: g.IconFolder,
		SofDNA: sofDNA, SofFactionName: g.SofFactionName, SofHullName: g.SofHullName, SofRaceName: g.SofRaceName,
	}, nil
}

type GetUniverseGraphicInput struct {
	GraphicID int64 `json:"graphicId" jsonschema:"the graphic ID to look up"`
}

func getUniverseGraphicHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseGraphicInput, UniverseGraphic] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseGraphicInput) (*mcpsdk.CallToolResult, UniverseGraphic, error) {
		out, err := getUniverseGraphic(ctx, sdeStore, sso.UserAgent(version), in.GraphicID)
		if err != nil {
			return nil, UniverseGraphic{}, fmt.Errorf("get universe graphic: %w", err)
		}
		return nil, out, nil
	}
}
```
Note: `g.SofFationName` (not `SofFactionName`) matches the vendored `goesi-openapi` client's actual
generated field name — a pre-existing typo in that library, not introduced here; keep it exactly
as-is, same as the original pre-migration code already did.

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestListUniverseAncestries|TestListUniverseBloodlines|TestListUniverseRaces|TestListUniverseFactions|TestListUniverseGraphics|TestGetUniverseGraphic' -v`
Expected: PASS.

- [ ] **Step 10: Wire this task's 6 registration call sites**

Modify `mcp/server.go`:
```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_ancestries",
		Description: "List every character ancestry (public data, no character required).",
	}, listUniverseAncestriesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_bloodlines",
		Description: "List every character bloodline (public data, no character required).",
	}, listUniverseBloodlinesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_races",
		Description: "List every character race (public data, no character required).",
	}, listUniverseRacesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_factions",
		Description: "List every NPC faction (public data, no character required).",
	}, listUniverseFactionsHandler(version, sdeStore))
```
and, further down where `list_universe_graphics`/`get_universe_graphic` are registered:
```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_graphics",
		Description: "List every graphic ID (public data, no character required).",
	}, listUniverseGraphicsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_graphic",
		Description: "Get a graphic's collision/graphic files, icon folder, and Space Object Factory (SOF) DNA/hull/race/faction names (public data, no character required).",
	}, getUniverseGraphicHandler(version, sdeStore))
```

- [ ] **Step 11: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: clean build/vet/gofmt, every test in the module passes under `-race`. All 18 migrating
tools are now read-through-cache-backed; only the 6 celestial-detail tools remain ESI-direct.

- [ ] **Step 12: Commit**

```bash
git add sde/graphic.go sde/graphic_test.go mcp/universe.go mcp/universe_internal_test.go mcp/server.go
git commit -m "feat: migrate reference-list tools to sde/-backed read-through cache"
```

---

### Task 10: Docs sync — shrink the SDE-over-ESI hard-rule exception to the 6 celestial-detail tools

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Interfaces:** None — this task only updates prose documentation to match the code shipped in
Tasks 1–9. No behavior change.

- [ ] **Step 1: Update `CLAUDE.md`'s hard rule**

Modify `CLAUDE.md` — replace the "SDE over ESI" bullet (currently `CLAUDE.md:46-54`):
```markdown
- **SDE over ESI**: any tool needing a resource that exists in the SDE must read it from `sde/`,
  never call ESI for it. Performance and API-citizenship requirement, not a suggestion.
  Permanent exception: the Universe tag's 6 celestial-detail tools
  (`get_universe_asteroid_belt`/`_moon`/`_planet`/`_stargate`/`_star`/`_station`) call ESI
  directly and always will — their defining `Name` field doesn't exist anywhere in the SDE (FC
  generates these names server-side from system name + orbital position, not as stored data), so
  "migrating everything except Name" would still cost a live ESI call per lookup for no latency
  win. Every other Universe tag tool (types/groups/categories, regions/constellations/systems,
  ancestries/bloodlines/races/factions/graphics — 18 tools) is `sde/`-backed via a read-through
  cache: try `sde/` first, fall back to ESI on a cache miss, write the result back so the next
  lookup of that ID is fast. A handful of fields with no SDE source at all
  (`get_universe_type`'s `PackagedVolume`, `list_universe_races`' `AllianceID`,
  `get_universe_system`'s `Planets`, `get_universe_graphic`'s `CollisionFile`/`SofDNA`) are
  populated on a cache miss (fresh off ESI) but dropped on a cache hit — documented per-tool where
  each is implemented.
```

- [ ] **Step 2: Update `CLAUDE.md`'s `sde/` architecture-table row**

Modify `CLAUDE.md` — replace the `sde/` row (`CLAUDE.md:21`):
```markdown
| `sde/` | Local cache of FC's Static Data Export, backed by `go.etcd.io/bbolt` (`~/.neat/sde/sde.db`). Lazily syncs against EVE Ref's `data.everef.net/ccp/sde/` index on first lookup call per `Store` lifetime (at most once per `neat mcp` process run) — a failed sync doesn't error the lookup, it just falls back to whatever's already cached, since SDE data is enrichment, not a hard requirement. Backs 18 of the Universe tag's tools via a read-through cache (`mcp/universe.go` orchestrates the ESI fallback and write-back; `sde/` itself never imports `esi/`) — only the 6 celestial-detail tools remain ESI-direct, permanently, since their defining `Name` field has no SDE source. |
```

- [ ] **Step 3: Update `README.md`'s matching text**

Modify `README.md` — replace the `sde/` row (`README.md:33`) with the same text as Step 2's
`CLAUDE.md` row, and replace the line-14 mention (`a local SDE cache (`sde/`, covering
categories/groups/types) are built. See ...`) so it no longer describes `sde/` as
categories/groups/types-only:
```markdown
a local SDE cache (`sde/`, backing 18 of the Universe tag's tools via a read-through cache) are
built. See
```

- [ ] **Step 4: Verify no other stale reference remains**

Run: `grep -rn "categories/groups/types\|ESI-direct exception\|no mcp/.*tool has been migrated" CLAUDE.md README.md`
Expected: no output — every stale description of `sde/`'s old scope or the old 24-tool exception
list has been updated by Steps 1–3.

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: shrink SDE-over-ESI exception to the 6 celestial-detail tools"
```

---
