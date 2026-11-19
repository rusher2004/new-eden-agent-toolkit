# Closing Remaining Universe-Tag Gaps with EVE Ref's Supplemental Datasets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the final 6 Universe-tag celestial-detail tools off ESI-direct, and close 3 of the
4 remaining field drops (`Race.AllianceID`, `System.Planets`, `Graphic.SofDNA`), using EVE Ref's
`reference-data` and `esi-scrape` datasets as two new best-effort bulk-seed sources alongside the
existing FC SDE zip. `CollisionFile` stays dropped permanently — confirmed absent from ESI's own
live responses, not just the static export.

**Architecture:** `sde/sync()` gains two more fetch phases (`reference-data`'s `types.json` for
`PackagedVolume`; `esi-scrape`'s 9 needed `universe/*` members for celestial data + `AllianceID` +
`Planets` + `SofDNA`), both independently best-effort — a failure logs a warning and skips just
that augmentation, never fails the core SDE-sourced sync. The 6 new celestial resources get the
same `Store` accessor + `Put*` + read-through pattern as every other Task 1–9 resource in the prior
plan.

**Tech Stack:** Go 1.26.1, existing `go.etcd.io/bbolt` v1.5.0. Two new dependencies:
`go.yaml.in/yaml/v3` — the actively-maintained fork of `gopkg.in/yaml.v3`, hosted by the official
YAML org after the original `go-yaml/yaml` was archived as unmaintained in April 2025 (confirmed:
`go.yaml.in/yaml/v3` is the same dependency `sigs.k8s.io/yaml` itself uses, actively pushed as
recently as yesterday; API-compatible with `gopkg.in/yaml.v3` — same package, same `Unmarshal`,
same native YAML struct tags, so no need for a JSON-tag-wrapped alternative like `ghodss/yaml`,
which is itself stale and still depends on the archived `gopkg.in/yaml.v2`) — and
`github.com/ulikunitz/xz`
(pure-Go XZ decompression — `esi-scrape`/`reference-data` ship as `.tar.xz`, and neither
`archive/tar` nor any stdlib package decompresses XZ; `NewReader(io.Reader) (*Reader, error)` is
the whole API surface this plan needs).

## Global Constraints

- Branch: continues directly on `sde-universe-migration` (per your explicit choice — this extends
  PR #9 rather than stacking a new branch/PR).
- `sde/` still must not import `esi/` or `sso/` — read-through orchestration stays entirely in
  `mcp/universe.go`.
- `*sde.Store` DI is unchanged — no new parameters on `NewServer` or any handler constructor;
  every migrated/simplified tool in this plan already has `sdeStore *sde.Store` threaded through
  from the prior plan.
- Every URL/member-path/field-name claim in this plan was verified directly against the live
  `data.everef.net/reference-data/` and `data.everef.net/esi-scrape/` archives during design (real
  downloads, real `tar -tJf`/`grep` against extracted files) — re-verify the same way before
  finalizing a task if anything looks off.
- `reference-data` URL: `https://data.everef.net/reference-data/reference-data-latest.tar.xz`
  (members are flat, no directory prefix — e.g. `types.json` directly).
- `esi-scrape` URL: `https://data.everef.net/esi-scrape/eve-ref-esi-scrape-latest.tar.xz` (members
  are nested under `eve-ref-esi-scrape/data/tranquility/universe/` — e.g.
  `eve-ref-esi-scrape/data/tranquility/universe/stations.en-us.yaml`).
- Both new sources are fetched via the existing `fetchBytes(ctx, url, userAgent)` helper already
  in `sde/sync.go` — no new HTTP-fetching code, only new extraction/parsing on top of it.
- Failure isolation (confirmed): the core FC SDE zip fetch stays sync-fatal exactly as today.
  `reference-data`/`esi-scrape` fetch or parse failures are caught, logged via
  `slog.Default().WarnContext(ctx, "sde: failed to fetch <source>, skipping augmentation",
  "error", err)`, and degrade to an empty/`nil` result — the rest of `sync()` proceeds unaffected.
- Celestial names are **not** multilingual — confirmed identical across every scraped locale
  (`en-us` vs `ja` for the same moon ID). All 6 new resources use plain `string` `Name` fields, not
  `map[string]string`.
- `fmt.Errorf("...: %w", err)` wrapping at every layer, matching the existing codebase convention.
- Tool names, input schemas, and (aside from `CollisionFile`, already dropped) output schemas are
  unchanged — this plan is purely re-sourcing plus 6 tools moving off ESI-direct, not a contract
  change.
- Every new external dependency or hard-rule change gets its doc-sync pass in the same PR as the
  code that adds it — Task 9 is that pass.

---

### Task 1: Add dependencies + `sde/asteroidbelt.go`, `sde/moon.go`, `sde/planet.go`

**Files:**
- Modify: `go.mod`, `go.sum` (add `go.yaml.in/yaml/v3`, `github.com/ulikunitz/xz`)
- Create: `sde/asteroidbelt.go`, `sde/asteroidbelt_test.go`
- Create: `sde/moon.go`, `sde/moon_test.go`
- Create: `sde/planet.go`, `sde/planet_test.go`

**Interfaces:**
- Consumes: `Store.db *bbolt.DB`, `Store.ensureSynced`, `int64Key`, `Position` (Task 1 of the prior
  plan), `seedStore`/`putRecord` test helpers.
- Produces (for Task 6's sync.go rewrite and Task 7's mcp/ migrations):
  - `type AsteroidBelt struct { ID int64; Name string; Position Position; SystemID int64 }`
  - `func (s *Store) AsteroidBelt(ctx, userAgent, id) (AsteroidBelt, bool, error)`
  - `func (s *Store) PutAsteroidBelt(b AsteroidBelt) error`
  - `type Moon struct { ID int64; Name string; Position Position; SystemID int64 }`
  - `func (s *Store) Moon(ctx, userAgent, id) (Moon, bool, error)` / `PutMoon(m Moon) error`
  - `type Planet struct { ID int64; Name string; TypeID int64; Position Position; SystemID int64 }`
  - `func (s *Store) Planet(ctx, userAgent, id) (Planet, bool, error)` / `PutPlanet(p Planet) error`
  - Bucket names `"asteroidBelts"`, `"moons"`, `"planets"` — the contract with Task 6's sync writer.

Verified real record shapes (`data.everef.net/esi-scrape`'s `universe/*.en-us.yaml`, confirmed
identical across all 5 scraped locales):
```yaml
# asteroid_belts.en-us.yaml
"40000003":
  name: "Tanoo I - Asteroid Belt 1"
  position: {x: 161967513600, "y": 21288837120, z: -73505464320}
  system_id: 30000001
# moons.en-us.yaml
"40000004":
  moon_id: 40000004
  name: "Tanoo I - Moon 1"
  position: {x: 162088094286, "y": 21314854783, z: -73598621491}
  system_id: 30000001
# planets.en-us.yaml
"40000002":
  name: "Tanoo I"
  planet_id: 40000002
  position: {x: 161891117336, "y": 21288951986, z: -73529712226}
  system_id: 30000001
  type_id: 11
```
No FC SDE source exists for any of these at all (celestial names are computed server-side by FC,
never stored statically anywhere) — these three resources are sourced entirely from `esi-scrape`,
seeded in bulk at sync time (Task 6), with the usual live-ESI-on-miss fallback (Task 7).

- [ ] **Step 1: Add the two new dependencies**

Run:
```bash
go get go.yaml.in/yaml/v3@v3.0.3
go get github.com/ulikunitz/xz@latest
go mod tidy
```
Expected: `go.mod`'s first `require` block gains both as direct dependencies. Note: `go.sum` may
still carry the now-unused `gopkg.in/yaml.v3` entry as an indirect test-only dependency of another
module (e.g. `go-keyring`'s own test deps) — that's fine and expected, `go mod tidy` only removes
entries nothing in the build graph needs at all.

- [ ] **Step 2: Run build to verify it's still clean**

Run: `go build ./... && go vet ./...`
Expected: clean — adding an unused dependency doesn't break anything yet.

- [ ] **Step 3: Write the failing tests for `AsteroidBelt`**

Create `sde/asteroidbelt_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestAsteroidBelt_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "asteroidBelts", 40000003,
		`{"_key": 40000003, "name": "Tanoo I - Asteroid Belt 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	got, ok, err := store.AsteroidBelt(context.Background(), "test-agent", 40000003)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false, want true")
	}
	if got.ID != 40000003 || got.Name != "Tanoo I - Asteroid Belt 1" || got.SystemID != 30000001 {
		t.Errorf("got = %+v, want ID=40000003 Name=\"Tanoo I - Asteroid Belt 1\" SystemID=30000001", got)
	}
	if got.Position != (Position{X: 1, Y: 2, Z: 3}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
}

func TestAsteroidBelt_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.AsteroidBelt(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if ok {
		t.Error("AsteroidBelt ok = true, want false for an unknown ID")
	}
}

func TestAsteroidBelt_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "asteroidBelts", 40000003,
		`{"_key": 999, "name": "Tanoo I - Asteroid Belt 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	_, _, err := store.AsteroidBelt(context.Background(), "test-agent", 40000003)
	if err == nil {
		t.Fatal("AsteroidBelt returned nil error, want an error for a mismatched _key")
	}
}

func TestPutAsteroidBelt_ThenAsteroidBelt_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := AsteroidBelt{ID: 40000099, Name: "New Belt", Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099}

	if err := store.PutAsteroidBelt(want); err != nil {
		t.Fatalf("PutAsteroidBelt returned error: %v", err)
	}

	got, ok, err := store.AsteroidBelt(context.Background(), "test-agent", 40000099)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false after PutAsteroidBelt, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./sde/... -run TestAsteroidBelt -v`
Expected: FAIL to compile — `undefined: AsteroidBelt`.

- [ ] **Step 5: Write the minimal implementation**

Create `sde/asteroidbelt.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// AsteroidBelt is a minimal decode of an EVE Ref esi-scrape asteroid belt
// record. Unlike every other sde/ resource, there is no FC SDE source for
// this data at all -- asteroid belt names are computed server-side by FC,
// never stored in any static file. This is sourced entirely from EVE Ref's
// esi-scrape dataset (a mirror of ESI's own live responses), seeded in bulk
// at sync time; a read-through miss still falls back to a live ESI call.
type AsteroidBelt struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

type asteroidBeltRecord struct {
	Key      int64    `json:"_key"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

// AsteroidBelt looks up an asteroid belt by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) AsteroidBelt(ctx context.Context, userAgent string, id int64) (AsteroidBelt, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out AsteroidBelt
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("asteroidBelts"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw asteroidBeltRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached asteroid belt %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached asteroid belt %d has mismatched key %d", id, raw.Key)
		}
		out, ok = AsteroidBelt{ID: raw.Key, Name: raw.Name, Position: raw.Position, SystemID: raw.SystemID}, true
		return nil
	})
	return out, ok, err
}

// PutAsteroidBelt writes b into the cache, keyed by b.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutAsteroidBelt(b AsteroidBelt) error {
	raw, err := json.Marshal(asteroidBeltRecord{Key: b.ID, Name: b.Name, Position: b.Position, SystemID: b.SystemID})
	if err != nil {
		return fmt.Errorf("marshal asteroid belt %d: %w", b.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte("asteroidBelts"))
		if err != nil {
			return fmt.Errorf("create asteroidBelts bucket: %w", err)
		}
		return bkt.Put(int64Key(b.ID), raw)
	})
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./sde/... -run TestAsteroidBelt -v`
Expected: PASS (4/4).

- [ ] **Step 7: Write the failing tests for `Moon`**

Create `sde/moon_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestMoon_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "moons", 40000004,
		`{"_key": 40000004, "name": "Tanoo I - Moon 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	got, ok, err := store.Moon(context.Background(), "test-agent", 40000004)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false, want true")
	}
	if got.ID != 40000004 || got.Name != "Tanoo I - Moon 1" || got.SystemID != 30000001 {
		t.Errorf("got = %+v, want ID=40000004 Name=\"Tanoo I - Moon 1\" SystemID=30000001", got)
	}
}

func TestMoon_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Moon(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if ok {
		t.Error("Moon ok = true, want false for an unknown ID")
	}
}

func TestMoon_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "moons", 40000004,
		`{"_key": 999, "name": "Tanoo I - Moon 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	_, _, err := store.Moon(context.Background(), "test-agent", 40000004)
	if err == nil {
		t.Fatal("Moon returned nil error, want an error for a mismatched _key")
	}
}

func TestPutMoon_ThenMoon_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Moon{ID: 40000199, Name: "New Moon", Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099}

	if err := store.PutMoon(want); err != nil {
		t.Fatalf("PutMoon returned error: %v", err)
	}

	got, ok, err := store.Moon(context.Background(), "test-agent", 40000199)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false after PutMoon, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 8: Run test to verify it fails**

Run: `go test ./sde/... -run TestMoon -v`
Expected: FAIL to compile — `undefined: Moon`.

- [ ] **Step 9: Write the minimal implementation**

Create `sde/moon.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Moon is a minimal decode of an EVE Ref esi-scrape moon record. Sourced
// entirely from esi-scrape, same as AsteroidBelt -- see its doc comment.
type Moon struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

type moonRecord struct {
	Key      int64    `json:"_key"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

// Moon looks up a moon by ID. ok is false if the ID isn't in the cache.
func (s *Store) Moon(ctx context.Context, userAgent string, id int64) (Moon, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Moon
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("moons"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw moonRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached moon %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached moon %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Moon{ID: raw.Key, Name: raw.Name, Position: raw.Position, SystemID: raw.SystemID}, true
		return nil
	})
	return out, ok, err
}

// PutMoon writes m into the cache, keyed by m.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutMoon(m Moon) error {
	raw, err := json.Marshal(moonRecord{Key: m.ID, Name: m.Name, Position: m.Position, SystemID: m.SystemID})
	if err != nil {
		return fmt.Errorf("marshal moon %d: %w", m.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("moons"))
		if err != nil {
			return fmt.Errorf("create moons bucket: %w", err)
		}
		return b.Put(int64Key(m.ID), raw)
	})
}
```

- [ ] **Step 10: Run test to verify it passes**

Run: `go test ./sde/... -run TestMoon -v`
Expected: PASS (4/4).

- [ ] **Step 11: Write the failing tests for `Planet`**

Create `sde/planet_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestPlanet_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "planets", 40000002,
		`{"_key": 40000002, "name": "Tanoo I", "typeId": 11, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	got, ok, err := store.Planet(context.Background(), "test-agent", 40000002)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false, want true")
	}
	if got.ID != 40000002 || got.Name != "Tanoo I" || got.TypeID != 11 || got.SystemID != 30000001 {
		t.Errorf("got = %+v, want ID=40000002 Name=\"Tanoo I\" TypeID=11 SystemID=30000001", got)
	}
}

func TestPlanet_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Planet(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if ok {
		t.Error("Planet ok = true, want false for an unknown ID")
	}
}

func TestPlanet_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "planets", 40000002,
		`{"_key": 999, "name": "Tanoo I", "typeId": 11, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	_, _, err := store.Planet(context.Background(), "test-agent", 40000002)
	if err == nil {
		t.Fatal("Planet returned nil error, want an error for a mismatched _key")
	}
}

func TestPutPlanet_ThenPlanet_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Planet{ID: 40000299, Name: "New Planet", TypeID: 12, Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099}

	if err := store.PutPlanet(want); err != nil {
		t.Fatalf("PutPlanet returned error: %v", err)
	}

	got, ok, err := store.Planet(context.Background(), "test-agent", 40000299)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false after PutPlanet, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 12: Run test to verify it fails**

Run: `go test ./sde/... -run TestPlanet -v`
Expected: FAIL to compile — `undefined: Planet`.

- [ ] **Step 13: Write the minimal implementation**

Create `sde/planet.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Planet is a minimal decode of an EVE Ref esi-scrape planet record.
// Sourced entirely from esi-scrape, same as AsteroidBelt -- see its doc
// comment.
type Planet struct {
	ID       int64
	Name     string
	TypeID   int64
	Position Position
	SystemID int64
}

type planetRecord struct {
	Key      int64    `json:"_key"`
	Name     string   `json:"name"`
	TypeID   int64    `json:"typeId"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

// Planet looks up a planet by ID. ok is false if the ID isn't in the cache.
func (s *Store) Planet(ctx context.Context, userAgent string, id int64) (Planet, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Planet
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("planets"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw planetRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached planet %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached planet %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Planet{ID: raw.Key, Name: raw.Name, TypeID: raw.TypeID, Position: raw.Position, SystemID: raw.SystemID}, true
		return nil
	})
	return out, ok, err
}

// PutPlanet writes p into the cache, keyed by p.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutPlanet(p Planet) error {
	raw, err := json.Marshal(planetRecord{Key: p.ID, Name: p.Name, TypeID: p.TypeID, Position: p.Position, SystemID: p.SystemID})
	if err != nil {
		return fmt.Errorf("marshal planet %d: %w", p.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("planets"))
		if err != nil {
			return fmt.Errorf("create planets bucket: %w", err)
		}
		return b.Put(int64Key(p.ID), raw)
	})
}
```

- [ ] **Step 14: Run test to verify it passes**

Run: `go test ./sde/... -run TestPlanet -v`
Expected: PASS (4/4).

- [ ] **Step 15: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 16: Commit**

```bash
git add go.mod go.sum sde/asteroidbelt.go sde/asteroidbelt_test.go sde/moon.go sde/moon_test.go sde/planet.go sde/planet_test.go
git commit -m "feat: add go.yaml.in/yaml and xz dependencies and AsteroidBelt/Moon/Planet accessors to sde/"
```

---

### Task 2: `sde/stargate.go`, `sde/star.go`, `sde/station.go`

**Files:**
- Create: `sde/stargate.go`, `sde/stargate_test.go`
- Create: `sde/star.go`, `sde/star_test.go`
- Create: `sde/station.go`, `sde/station_test.go`

**Interfaces:**
- Consumes: same as Task 1.
- Produces (for Task 6 and Task 7):
  - `type Stargate struct { ID int64; Name string; TypeID int64; Position Position; SystemID int64; DestinationStargateID int64; DestinationSystemID int64 }`
  - `func (s *Store) Stargate(ctx, userAgent, id) (Stargate, bool, error)` / `PutStargate(g Stargate) error`
  - `type Star struct { ID int64; Age int64; Luminosity float64; Name string; Radius int64; SolarSystemID int64; SpectralClass string; Temperature int64; TypeID int64 }`
  - `func (s *Store) Star(ctx, userAgent, id) (Star, bool, error)` / `PutStar(s Star) error`
  - `type Station struct { ID int64; Name string; SystemID int64; TypeID int64; Position Position; MaxDockableShipVolume float64; OfficeRentalCost float64; Owner *int64; RaceID *int64; ReprocessingEfficiency float64; ReprocessingStationsTake float64; Services []string }`
  - `func (s *Store) Station(ctx, userAgent, id) (Station, bool, error)` / `PutStation(st Station) error`
  - Bucket names `"stargates"`, `"stars"`, `"stations"`.

Verified real record shapes (`esi-scrape`'s `universe/*.en-us.yaml`):
```yaml
# stargates.en-us.yaml
"50000001":
  destination: {stargate_id: 50000482, system_id: 30000778}
  name: "Stargate (G-EURJ)"
  position: {x: ..., "y": ..., z: ...}
  stargate_id: 50000001
  system_id: 30000777
  type_id: 29633
# stars.en-us.yaml
"40000001":
  age: 14262808228
  luminosity: 0.01575000025331974
  name: "Tanoo - Star"
  radius: 63350000
  solar_system_id: 30000001
  spectral_class: "K2 V"
  temperature: 4567
  type_id: 45041
# stations.en-us.yaml
"60000004":
  max_dockable_ship_volume: 50000000
  name: "Muvolailen X - Moon 3 - CBD Corporation Storage"
  office_rental_cost: 3627229
  owner: 1000002
  position: {x: ..., "y": ..., z: ...}
  race_id: 1
  reprocessing_efficiency: 0.5
  reprocessing_stations_take: 0.05
  services: ["courier-missions", "reprocessing-plant", ...]
  station_id: 60000004
  system_id: 30002780
  type_id: 1531
```
All three match the existing ESI-sourced `mcp.UniverseStargate`/`UniverseStar`/`UniverseStation`
field sets exactly, confirmed field-for-field against `mcp/universe.go`'s pre-existing types.

- [ ] **Step 1: Write the failing tests for `Stargate`**

Create `sde/stargate_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestStargate_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stargates", 50000001,
		`{"_key": 50000001, "name": "Stargate (G-EURJ)", "typeId": 29633, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000777, "destinationStargateId": 50000482, "destinationSystemId": 30000778}`)

	got, ok, err := store.Stargate(context.Background(), "test-agent", 50000001)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false, want true")
	}
	if got.ID != 50000001 || got.Name != "Stargate (G-EURJ)" || got.TypeID != 29633 || got.SystemID != 30000777 {
		t.Errorf("got = %+v, want ID=50000001 Name=\"Stargate (G-EURJ)\" TypeID=29633 SystemID=30000777", got)
	}
	if got.DestinationStargateID != 50000482 || got.DestinationSystemID != 30000778 {
		t.Errorf("Destination = %d/%d, want 50000482/30000778", got.DestinationStargateID, got.DestinationSystemID)
	}
}

func TestStargate_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Stargate(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if ok {
		t.Error("Stargate ok = true, want false for an unknown ID")
	}
}

func TestStargate_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stargates", 50000001,
		`{"_key": 999, "name": "Stargate (G-EURJ)", "typeId": 29633, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000777, "destinationStargateId": 50000482, "destinationSystemId": 30000778}`)

	_, _, err := store.Stargate(context.Background(), "test-agent", 50000001)
	if err == nil {
		t.Fatal("Stargate returned nil error, want an error for a mismatched _key")
	}
}

func TestPutStargate_ThenStargate_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Stargate{ID: 50000099, Name: "New Stargate", TypeID: 29633, Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099, DestinationStargateID: 50000098, DestinationSystemID: 30000098}

	if err := store.PutStargate(want); err != nil {
		t.Fatalf("PutStargate returned error: %v", err)
	}

	got, ok, err := store.Stargate(context.Background(), "test-agent", 50000099)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false after PutStargate, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sde/... -run TestStargate -v`
Expected: FAIL to compile — `undefined: Stargate`.

- [ ] **Step 3: Write the minimal implementation**

Create `sde/stargate.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Stargate is a minimal decode of an EVE Ref esi-scrape stargate record.
// Sourced entirely from esi-scrape, same as AsteroidBelt -- see its doc
// comment.
type Stargate struct {
	ID                    int64
	Name                  string
	TypeID                int64
	Position              Position
	SystemID              int64
	DestinationStargateID int64
	DestinationSystemID   int64
}

type stargateRecord struct {
	Key                   int64    `json:"_key"`
	Name                  string   `json:"name"`
	TypeID                int64    `json:"typeId"`
	Position              Position `json:"position"`
	SystemID              int64    `json:"systemId"`
	DestinationStargateID int64    `json:"destinationStargateId"`
	DestinationSystemID   int64    `json:"destinationSystemId"`
}

// Stargate looks up a stargate by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Stargate(ctx context.Context, userAgent string, id int64) (Stargate, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Stargate
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("stargates"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw stargateRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached stargate %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached stargate %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Stargate{
			ID: raw.Key, Name: raw.Name, TypeID: raw.TypeID, Position: raw.Position, SystemID: raw.SystemID,
			DestinationStargateID: raw.DestinationStargateID, DestinationSystemID: raw.DestinationSystemID,
		}, true
		return nil
	})
	return out, ok, err
}

// PutStargate writes g into the cache, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStargate(g Stargate) error {
	raw, err := json.Marshal(stargateRecord{
		Key: g.ID, Name: g.Name, TypeID: g.TypeID, Position: g.Position, SystemID: g.SystemID,
		DestinationStargateID: g.DestinationStargateID, DestinationSystemID: g.DestinationSystemID,
	})
	if err != nil {
		return fmt.Errorf("marshal stargate %d: %w", g.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("stargates"))
		if err != nil {
			return fmt.Errorf("create stargates bucket: %w", err)
		}
		return b.Put(int64Key(g.ID), raw)
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./sde/... -run TestStargate -v`
Expected: PASS (4/4).

- [ ] **Step 5: Write the failing tests for `Star`**

Create `sde/star_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestStar_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stars", 40000001,
		`{"_key": 40000001, "age": 14262808228, "luminosity": 0.01575, "name": "Tanoo - Star", "radius": 63350000, "solarSystemId": 30000001, "spectralClass": "K2 V", "temperature": 4567, "typeId": 45041}`)

	got, ok, err := store.Star(context.Background(), "test-agent", 40000001)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true")
	}
	if got.ID != 40000001 || got.Name != "Tanoo - Star" || got.SolarSystemID != 30000001 || got.TypeID != 45041 {
		t.Errorf("got = %+v, want ID=40000001 Name=\"Tanoo - Star\" SolarSystemID=30000001 TypeID=45041", got)
	}
	if got.Age != 14262808228 || got.Luminosity != 0.01575 || got.Radius != 63350000 || got.SpectralClass != "K2 V" || got.Temperature != 4567 {
		t.Errorf("got = %+v, want Age=14262808228 Luminosity=0.01575 Radius=63350000 SpectralClass=\"K2 V\" Temperature=4567", got)
	}
}

func TestStar_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Star(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if ok {
		t.Error("Star ok = true, want false for an unknown ID")
	}
}

func TestStar_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stars", 40000001,
		`{"_key": 999, "age": 1, "luminosity": 0.1, "name": "X", "radius": 1, "solarSystemId": 1, "spectralClass": "K2 V", "temperature": 1, "typeId": 1}`)

	_, _, err := store.Star(context.Background(), "test-agent", 40000001)
	if err == nil {
		t.Fatal("Star returned nil error, want an error for a mismatched _key")
	}
}

func TestPutStar_ThenStar_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Star{ID: 40000199, Age: 1, Luminosity: 0.5, Name: "New Star", Radius: 100, SolarSystemID: 30000099, SpectralClass: "G2 V", Temperature: 5000, TypeID: 45041}

	if err := store.PutStar(want); err != nil {
		t.Fatalf("PutStar returned error: %v", err)
	}

	got, ok, err := store.Star(context.Background(), "test-agent", 40000199)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false after PutStar, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./sde/... -run TestStar -v`
Expected: FAIL to compile — `undefined: Star`.

- [ ] **Step 7: Write the minimal implementation**

Create `sde/star.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Star is a minimal decode of an EVE Ref esi-scrape star record. Sourced
// entirely from esi-scrape, same as AsteroidBelt -- see its doc comment.
type Star struct {
	ID            int64
	Age           int64
	Luminosity    float64
	Name          string
	Radius        int64
	SolarSystemID int64
	SpectralClass string
	Temperature   int64
	TypeID        int64
}

type starRecord struct {
	Key           int64   `json:"_key"`
	Age           int64   `json:"age"`
	Luminosity    float64 `json:"luminosity"`
	Name          string  `json:"name"`
	Radius        int64   `json:"radius"`
	SolarSystemID int64   `json:"solarSystemId"`
	SpectralClass string  `json:"spectralClass"`
	Temperature   int64   `json:"temperature"`
	TypeID        int64   `json:"typeId"`
}

// Star looks up a star by ID. ok is false if the ID isn't in the cache.
func (s *Store) Star(ctx context.Context, userAgent string, id int64) (Star, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Star
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("stars"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw starRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached star %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached star %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Star{
			ID: raw.Key, Age: raw.Age, Luminosity: raw.Luminosity, Name: raw.Name, Radius: raw.Radius,
			SolarSystemID: raw.SolarSystemID, SpectralClass: raw.SpectralClass, Temperature: raw.Temperature, TypeID: raw.TypeID,
		}, true
		return nil
	})
	return out, ok, err
}

// PutStar writes st into the cache, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStar(st Star) error {
	raw, err := json.Marshal(starRecord{
		Key: st.ID, Age: st.Age, Luminosity: st.Luminosity, Name: st.Name, Radius: st.Radius,
		SolarSystemID: st.SolarSystemID, SpectralClass: st.SpectralClass, Temperature: st.Temperature, TypeID: st.TypeID,
	})
	if err != nil {
		return fmt.Errorf("marshal star %d: %w", st.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("stars"))
		if err != nil {
			return fmt.Errorf("create stars bucket: %w", err)
		}
		return b.Put(int64Key(st.ID), raw)
	})
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./sde/... -run TestStar -v`
Expected: PASS (4/4).

- [ ] **Step 9: Write the failing tests for `Station`**

Create `sde/station_test.go`:
```go
package sde

import (
	"context"
	"testing"
)

func TestStation_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	owner := int64(1000002)
	raceID := int64(1)
	putRecord(t, store, "stations", 60000004,
		`{"_key": 60000004, "name": "Muvolailen X - Moon 3 - CBD Corporation Storage", "systemId": 30002780, "typeId": 1531, "position": {"x": 1, "y": 2, "z": 3}, "maxDockableShipVolume": 50000000, "officeRentalCost": 3627229, "owner": 1000002, "raceId": 1, "reprocessingEfficiency": 0.5, "reprocessingStationsTake": 0.05, "services": ["docking", "market"]}`)

	got, ok, err := store.Station(context.Background(), "test-agent", 60000004)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false, want true")
	}
	if got.ID != 60000004 || got.Name != "Muvolailen X - Moon 3 - CBD Corporation Storage" || got.SystemID != 30002780 || got.TypeID != 1531 {
		t.Errorf("got = %+v, want ID=60000004 SystemID=30002780 TypeID=1531", got)
	}
	if got.Owner == nil || *got.Owner != owner {
		t.Errorf("Owner = %v, want %d", got.Owner, owner)
	}
	if got.RaceID == nil || *got.RaceID != raceID {
		t.Errorf("RaceID = %v, want %d", got.RaceID, raceID)
	}
	if got.MaxDockableShipVolume != 50000000 || got.OfficeRentalCost != 3627229 || got.ReprocessingEfficiency != 0.5 || got.ReprocessingStationsTake != 0.05 {
		t.Errorf("got = %+v, want MaxDockableShipVolume=50000000 OfficeRentalCost=3627229 ReprocessingEfficiency=0.5 ReprocessingStationsTake=0.05", got)
	}
	if len(got.Services) != 2 || got.Services[0] != "docking" || got.Services[1] != "market" {
		t.Errorf("Services = %v, want [docking market]", got.Services)
	}
}

func TestStation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Station(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if ok {
		t.Error("Station ok = true, want false for an unknown ID")
	}
}

func TestStation_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stations", 60000004,
		`{"_key": 999, "name": "X", "systemId": 1, "typeId": 1, "position": {"x": 1, "y": 2, "z": 3}}`)

	_, _, err := store.Station(context.Background(), "test-agent", 60000004)
	if err == nil {
		t.Fatal("Station returned nil error, want an error for a mismatched _key")
	}
}

func TestPutStation_ThenStation_RoundTrips(t *testing.T) {
	store := seedStore(t)
	owner := int64(1000002)
	want := Station{
		ID: 60000099, Name: "New Station", SystemID: 30000099, TypeID: 1531, Position: Position{X: 9, Y: 8, Z: 7},
		MaxDockableShipVolume: 1000, OfficeRentalCost: 10000, Owner: &owner,
		ReprocessingEfficiency: 0.5, ReprocessingStationsTake: 0.05, Services: []string{"docking"},
	}

	if err := store.PutStation(want); err != nil {
		t.Fatalf("PutStation returned error: %v", err)
	}

	got, ok, err := store.Station(context.Background(), "test-agent", 60000099)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false after PutStation, want true")
	}
	if got.ID != want.ID || got.Name != want.Name || *got.Owner != *want.Owner || len(got.Services) != 1 {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 10: Run test to verify it fails**

Run: `go test ./sde/... -run TestStation -v`
Expected: FAIL to compile — `undefined: Station`.

- [ ] **Step 11: Write the minimal implementation**

Create `sde/station.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Station is a minimal decode of an EVE Ref esi-scrape NPC station record.
// Sourced entirely from esi-scrape, same as AsteroidBelt -- see its doc
// comment. This is the same NPC-station concept sde/sync.go already reads a
// slim (_key/solarSystemID/ownerID only) pass of from the official SDE's
// npcStations.jsonl for System.StationIDs/Faction.StationCount -- this
// resource is the full per-station detail those reverse indices point at.
type Station struct {
	ID                       int64
	Name                     string
	SystemID                 int64
	TypeID                   int64
	Position                 Position
	MaxDockableShipVolume    float64
	OfficeRentalCost         float64
	Owner                    *int64
	RaceID                   *int64
	ReprocessingEfficiency   float64
	ReprocessingStationsTake float64
	Services                 []string
}

type stationRecord struct {
	Key                      int64    `json:"_key"`
	Name                     string   `json:"name"`
	SystemID                 int64    `json:"systemId"`
	TypeID                   int64    `json:"typeId"`
	Position                 Position `json:"position"`
	MaxDockableShipVolume    float64  `json:"maxDockableShipVolume"`
	OfficeRentalCost         float64  `json:"officeRentalCost"`
	Owner                    *int64   `json:"owner,omitempty"`
	RaceID                   *int64   `json:"raceId,omitempty"`
	ReprocessingEfficiency   float64  `json:"reprocessingEfficiency"`
	ReprocessingStationsTake float64  `json:"reprocessingStationsTake"`
	Services                 []string `json:"services,omitempty"`
}

// Station looks up a station by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Station(ctx context.Context, userAgent string, id int64) (Station, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Station
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("stations"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw stationRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached station %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached station %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Station{
			ID: raw.Key, Name: raw.Name, SystemID: raw.SystemID, TypeID: raw.TypeID, Position: raw.Position,
			MaxDockableShipVolume: raw.MaxDockableShipVolume, OfficeRentalCost: raw.OfficeRentalCost,
			Owner: raw.Owner, RaceID: raw.RaceID, ReprocessingEfficiency: raw.ReprocessingEfficiency,
			ReprocessingStationsTake: raw.ReprocessingStationsTake, Services: raw.Services,
		}, true
		return nil
	})
	return out, ok, err
}

// PutStation writes st into the cache, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStation(st Station) error {
	raw, err := json.Marshal(stationRecord{
		Key: st.ID, Name: st.Name, SystemID: st.SystemID, TypeID: st.TypeID, Position: st.Position,
		MaxDockableShipVolume: st.MaxDockableShipVolume, OfficeRentalCost: st.OfficeRentalCost,
		Owner: st.Owner, RaceID: st.RaceID, ReprocessingEfficiency: st.ReprocessingEfficiency,
		ReprocessingStationsTake: st.ReprocessingStationsTake, Services: st.Services,
	})
	if err != nil {
		return fmt.Errorf("marshal station %d: %w", st.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("stations"))
		if err != nil {
			return fmt.Errorf("create stations bucket: %w", err)
		}
		return b.Put(int64Key(st.ID), raw)
	})
}
```

- [ ] **Step 12: Run test to verify it passes**

Run: `go test ./sde/... -run TestStation -v`
Expected: PASS (4/4).

- [ ] **Step 13: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 14: Commit**

```bash
git add sde/stargate.go sde/stargate_test.go sde/star.go sde/star_test.go sde/station.go sde/station_test.go
git commit -m "feat: add Stargate/Star/Station accessors to sde/"
```

---

### Task 3: `Race.AllianceID` + `Type.PackagedVolume`

**Files:**
- Modify: `sde/race.go` (currently: `type Race struct { ID int64; Description map[string]string; Name map[string]string }`, no `Put*` — list-only, matches the file as it exists on disk today)
- Modify: `sde/type.go` (currently has the full field set from the prior migration, `PackagedVolume` deliberately absent — see its doc comment)

**Interfaces:**
- Consumes: same as Task 1.
- Produces (for Task 6 and Task 8): `Race` gains `AllianceID *int64`; `Type` gains `PackagedVolume
  *float64`.

Verified: `esi-scrape`'s `universe/races.en-us.yaml` is a plain YAML **list** (not a map keyed by
ID, unlike every other `esi-scrape` file used in this plan):
```yaml
- alliance_id: 500001
  description: "Founded on the tenets of patriotism..."
  name: "Caldari"
  race_id: 1
```
`reference-data`'s `types.json` is a map keyed by type ID:
```json
"587": {"...": "...", "packaged_volume": 2500, "volume": 27289, "...": "..."}
```
Both values confirmed non-null for real records during design.

- [ ] **Step 1: Write the failing tests**

Modify `sde/race_test.go` — add after `TestRaces_ReturnsAll`:
```go

func TestRaces_ReturnsAllianceID(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 1, "allianceId": 500001, "description": {"en": "..."}, "name": {"en": "Caldari"}}`)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Races = %+v, want one entry", got)
	}
	if got[0].AllianceID == nil || *got[0].AllianceID != 500001 {
		t.Errorf("AllianceID = %v, want 500001", got[0].AllianceID)
	}
}

func TestRaces_NoAllianceID_ReturnsNil(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 2, `{"_key": 2, "description": {"en": "..."}, "name": {"en": "No Alliance"}}`)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 || got[0].AllianceID != nil {
		t.Errorf("AllianceID = %v, want nil", got[0].AllianceID)
	}
}
```

Modify `sde/type_test.go` — add after `TestType_KnownID_ReturnsFullFieldSetAndDogma`:
```go

func TestType_KnownID_ReturnsPackagedVolume(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 587, "groupID": 25, "name": {"en": "Rifter"}, "published": true, "packagedVolume": 2500}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.PackagedVolume == nil || *got.PackagedVolume != 2500 {
		t.Errorf("PackagedVolume = %v, want 2500", got.PackagedVolume)
	}
}

func TestType_NoPackagedVolume_ReturnsNil(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 588,
		`{"_key": 588, "groupID": 25, "name": {"en": "No Volume"}, "published": true}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 588)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.PackagedVolume != nil {
		t.Errorf("PackagedVolume = %v, want nil", got.PackagedVolume)
	}
}
```

Also modify `TestPutType_ThenType_RoundTrips` (already in `sde/type_test.go`) to also assert
`PackagedVolume` round-trips:
```go
func TestPutType_ThenType_RoundTrips(t *testing.T) {
	store := seedStore(t)
	mass := 1000.0
	packagedVolume := 500.0
	want := Type{ID: 999, GroupID: 25, Name: map[string]string{"en": "New Type"}, Description: map[string]string{"en": "A new type."}, Published: true, Mass: &mass, PackagedVolume: &packagedVolume, DogmaAttributes: []TypeDogmaAttribute{{AttributeID: 1, Value: 2.0}}}

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
	if got.PackagedVolume == nil || *got.PackagedVolume != packagedVolume {
		t.Errorf("PackagedVolume = %v, want %v", got.PackagedVolume, packagedVolume)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0] != want.DogmaAttributes[0] {
		t.Errorf("DogmaAttributes = %+v, want %+v", got.DogmaAttributes, want.DogmaAttributes)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sde/... -run 'TestRaces_ReturnsAllianceID|TestRaces_NoAllianceID|TestType_KnownID_ReturnsPackagedVolume|TestType_NoPackagedVolume|TestPutType_ThenType_RoundTrips' -v`
Expected: FAIL to compile — `got.AllianceID undefined`, `got.PackagedVolume undefined`.

- [ ] **Step 3: Write the minimal implementation**

Modify `sde/race.go`:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Race is a minimal decode of an SDE race record, augmented at sync time
// with AllianceID from EVE Ref's esi-scrape dataset -- the official SDE's
// races.jsonl has no such field, and there is no alliances.jsonl anywhere
// in the SDE archive at all.
type Race struct {
	ID          int64
	AllianceID  *int64
	Description map[string]string
	Name        map[string]string
}

type raceRecord struct {
	Key         int64             `json:"_key"`
	AllianceID  *int64            `json:"allianceId,omitempty"`
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
			out = append(out, Race{ID: raw.Key, AllianceID: raw.AllianceID, Description: raw.Description, Name: raw.Name})
			return nil
		})
	})
	return out, err
}
```

Modify `sde/type.go` — update the `Type` struct, `typeRecord`, `Type()`, and `PutType()`:
```go
// Type is a minimal decode of an SDE type record -- the leaf level of
// EVE's item hierarchy (e.g. "Rifter", within the "Frigate" group).
// PackagedVolume has no source in the official SDE at all -- it's sourced
// entirely from EVE Ref's reference-data dataset at sync time, the same
// "stored but not raw from the primary source" pattern System.StationIDs
// already established.
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
	PackagedVolume  *float64
	PortionSize     *int64
	Radius          *float64
	Volume          *float64
	DogmaAttributes []TypeDogmaAttribute
	DogmaEffects    []TypeDogmaEffect
}
```
```go
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
	PackagedVolume  *float64                   `json:"packagedVolume,omitempty"`
	PortionSize     *int64                     `json:"portionSize,omitempty"`
	Radius          *float64                   `json:"radius,omitempty"`
	Volume          *float64                   `json:"volume,omitempty"`
	DogmaAttributes []typeDogmaAttributeRecord `json:"dogmaAttributes,omitempty"`
	DogmaEffects    []typeDogmaEffectRecord    `json:"dogmaEffects,omitempty"`
}
```
In `Type()`, change the `out = Type{...}` construction to:
```go
		out = Type{
			ID: raw.Key, GroupID: raw.GroupID, Name: raw.Name, Description: raw.Description, Published: raw.Published,
			Capacity: raw.Capacity, GraphicID: raw.GraphicID, IconID: raw.IconID, MarketGroupID: raw.MarketGroupID,
			Mass: raw.Mass, PackagedVolume: raw.PackagedVolume, PortionSize: raw.PortionSize, Radius: raw.Radius, Volume: raw.Volume,
		}
```
In `PutType()`, change the `raw := typeRecord{...}` construction to:
```go
	raw := typeRecord{
		Key: t.ID, GroupID: t.GroupID, Name: t.Name, Description: t.Description, Published: t.Published,
		Capacity: t.Capacity, GraphicID: t.GraphicID, IconID: t.IconID, MarketGroupID: t.MarketGroupID,
		Mass: t.Mass, PackagedVolume: t.PackagedVolume, PortionSize: t.PortionSize, Radius: t.Radius, Volume: t.Volume,
	}
```
Every other line in both functions (the `DogmaAttributes`/`DogmaEffects` loops, the bucket
read/write plumbing) is unchanged.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./sde/... -run 'TestRaces|TestType|TestPutType' -v`
Expected: PASS — every `TestRaces_*`/`TestType_*`/`TestPutType_*` test, including the pre-existing
ones.

- [ ] **Step 5: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 6: Commit**

```bash
git add sde/race.go sde/race_test.go sde/type.go sde/type_test.go
git commit -m "feat: add Race.AllianceID and Type.PackagedVolume fields"
```

---

### Task 4: `System.Planets`

**Files:**
- Modify: `sde/system.go` (current state shown in full below — this is the file exactly as it
  exists on disk after the prior migration; `System`/`systemRecord`/`System()`/`PutSystem()` are
  reproduced here so this task's diff is unambiguous)

**Interfaces:**
- Consumes: same as Task 1.
- Produces (for Task 6 and Task 8): `System` gains `Planets []SystemPlanet`; new
  `type SystemPlanet struct { PlanetID int64; AsteroidBeltIDs []int64; MoonIDs []int64 }` (mirrors
  `mcp.UniverseSystemPlanet`'s shape as its own `sde`-owned type — `sde/` still doesn't import
  `mcp/`).

Verified real record shape (`esi-scrape`'s `universe/systems.en-us.yaml`, confirmed for Jita,
system 30000142):
```json
{
  "constellation_id": 20000020,
  "name": "Jita",
  "planets": [
    {"planet_id": 40009077},
    {"planet_id": 40009078},
    {"moons": [40009081], "planet_id": 40009080},
    {"moons": [40009083, 40009084, "..."], "planet_id": 40009082}
  ]
}
```
No `asteroid_belts` key appeared on any of Jita's planets in the verified sample (Jita's belts are
attached to different planets than the ones sampled) — the field is optional per-planet, same as
`moons`, and both are modeled as `[]int64` that's simply empty/nil when absent, exactly like
`Group.TypeIDs`/`Category.GroupIDs` already do for zero-child cases elsewhere in `sde/`.

- [ ] **Step 1: Write the failing tests**

Modify `sde/system_test.go` — add after `TestSystem_KnownID_ReturnsAugmentedStationIDs`:
```go

func TestSystem_KnownID_ReturnsPlanets(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000142,
		`{"_key": 30000142, "constellationID": 20000020, "name": {"en": "Jita"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.9459, "planets": [{"planetId": 40009077}, {"planetId": 40009080, "moonIds": [40009081]}, {"planetId": 40009082, "moonIds": [40009083, 40009084], "asteroidBeltIds": [40009085]}]}`)

	got, ok, err := store.System(context.Background(), "test-agent", 30000142)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if len(got.Planets) != 3 {
		t.Fatalf("Planets = %+v, want 3 entries", got.Planets)
	}
	if got.Planets[0].PlanetID != 40009077 || len(got.Planets[0].MoonIDs) != 0 || len(got.Planets[0].AsteroidBeltIDs) != 0 {
		t.Errorf("Planets[0] = %+v, want PlanetID=40009077 with no moons/belts", got.Planets[0])
	}
	if got.Planets[1].PlanetID != 40009080 || len(got.Planets[1].MoonIDs) != 1 || got.Planets[1].MoonIDs[0] != 40009081 {
		t.Errorf("Planets[1] = %+v, want PlanetID=40009080 MoonIDs=[40009081]", got.Planets[1])
	}
	if got.Planets[2].PlanetID != 40009082 || len(got.Planets[2].MoonIDs) != 2 || len(got.Planets[2].AsteroidBeltIDs) != 1 {
		t.Errorf("Planets[2] = %+v, want PlanetID=40009082 with 2 moons and 1 belt", got.Planets[2])
	}
}

func TestSystem_NoPlanets_ReturnsEmpty(t *testing.T) {
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
	if len(got.Planets) != 0 {
		t.Errorf("Planets = %+v, want empty", got.Planets)
	}
}
```

Also modify `TestPutSystem_ThenSystem_RoundTrips` (already in `sde/system_test.go`) to also assert
`Planets` round-trips:
```go
func TestPutSystem_ThenSystem_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := System{
		ID: 30000099, ConstellationID: 20000001, Name: map[string]string{"en": "New System"}, Position: Position{X: 9, Y: 8, Z: 7}, SecurityStatus: 0.9, StationIDs: []int64{60000099},
		Planets: []SystemPlanet{{PlanetID: 40000099, MoonIDs: []int64{40000199}}},
	}

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
	if len(got.Planets) != 1 || got.Planets[0].PlanetID != 40000099 || len(got.Planets[0].MoonIDs) != 1 || got.Planets[0].MoonIDs[0] != 40000199 {
		t.Errorf("Planets = %+v, want one entry PlanetID=40000099 MoonIDs=[40000199]", got.Planets)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sde/... -run 'TestSystem_KnownID_ReturnsPlanets|TestSystem_NoPlanets|TestPutSystem_ThenSystem_RoundTrips' -v`
Expected: FAIL to compile — `undefined: SystemPlanet`, `got.Planets undefined`.

- [ ] **Step 3: Write the minimal implementation**

Replace the whole `sde/system.go` file:
```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// SystemPlanet is a minimal decode of one planet entry within a solar
// system record, mirroring mcp.UniverseSystemPlanet's shape as its own
// sde-owned type -- sde/ does not import mcp/.
type SystemPlanet struct {
	PlanetID        int64
	AsteroidBeltIDs []int64
	MoonIDs         []int64
}

// System is a minimal decode of an SDE solar system record -- EVE's
// geography level between Constellation and celestial bodies. StationIDs is
// not present on the raw mapSolarSystems.jsonl record at all -- it's
// augmented at sync time from a reverse index over npcStations.jsonl (see
// sync.go), or supplied directly by a read-through ESI fetch (ESI's system
// detail response already includes it). Planets has no source in the
// official SDE at all (would need a full mapPlanets.jsonl pass for two
// nested ID arrays) -- it's sourced entirely from EVE Ref's esi-scrape
// dataset at sync time, or directly from a read-through ESI fetch (ESI's
// response already includes it, same as StationIDs).
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
	Planets         []SystemPlanet
}

type systemPlanetRecord struct {
	PlanetID        int64   `json:"planetId"`
	AsteroidBeltIDs []int64 `json:"asteroidBeltIds,omitempty"`
	MoonIDs         []int64 `json:"moonIds,omitempty"`
}

type systemRecord struct {
	Key             int64                `json:"_key"`
	ConstellationID int64                `json:"constellationID"`
	Name            map[string]string    `json:"name"`
	Position        Position             `json:"position"`
	SecurityStatus  float64              `json:"securityStatus"`
	SecurityClass   *string              `json:"securityClass,omitempty"`
	StarID          *int64               `json:"starID,omitempty"`
	StargateIDs     []int64              `json:"stargateIds,omitempty"`
	StationIDs      []int64              `json:"stationIds,omitempty"`
	Planets         []systemPlanetRecord `json:"planets,omitempty"`
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
		out = System{
			ID: raw.Key, ConstellationID: raw.ConstellationID, Name: raw.Name, Position: raw.Position,
			SecurityStatus: raw.SecurityStatus, SecurityClass: raw.SecurityClass, StarID: raw.StarID,
			StargateIDs: raw.StargateIDs, StationIDs: raw.StationIDs,
		}
		if len(raw.Planets) > 0 {
			out.Planets = make([]SystemPlanet, len(raw.Planets))
			for i, p := range raw.Planets {
				out.Planets[i] = SystemPlanet{PlanetID: p.PlanetID, AsteroidBeltIDs: p.AsteroidBeltIDs, MoonIDs: p.MoonIDs}
			}
		}
		ok = true
		return nil
	})
	return out, ok, err
}

// PutSystem writes sys into the cache, keyed by sys.ID -- used by mcp/'s
// read-through fallback. Unlike the bulk sync path, a read-through fetch
// gets StationIDs and Planets directly from ESI's response, no augmentation
// needed.
func (s *Store) PutSystem(sys System) error {
	raw := systemRecord{
		Key: sys.ID, ConstellationID: sys.ConstellationID, Name: sys.Name, Position: sys.Position,
		SecurityStatus: sys.SecurityStatus, SecurityClass: sys.SecurityClass, StarID: sys.StarID,
		StargateIDs: sys.StargateIDs, StationIDs: sys.StationIDs,
	}
	if len(sys.Planets) > 0 {
		raw.Planets = make([]systemPlanetRecord, len(sys.Planets))
		for i, p := range sys.Planets {
			raw.Planets[i] = systemPlanetRecord{PlanetID: p.PlanetID, AsteroidBeltIDs: p.AsteroidBeltIDs, MoonIDs: p.MoonIDs}
		}
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

Run: `go test ./sde/... -run TestSystem -v && go test ./sde/... -run TestPutSystem -v`
Expected: PASS — every `TestSystem_*`/`TestPutSystem_*` test, including the pre-existing ones.

- [ ] **Step 5: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 6: Commit**

```bash
git add sde/system.go sde/system_test.go
git commit -m "feat: add System.Planets field"
```

---

### Task 5: `Graphic.SofDNA`

**Files:**
- Modify: `sde/graphic.go` (current state shown above — reproduced in the implementation step
  below so this task's diff is unambiguous)

**Interfaces:**
- Consumes: same as Task 1.
- Produces (for Task 6 and Task 8): `Graphic` gains `SofDNA *string`.

Verified real record shape (`esi-scrape`'s `universe/graphics.yaml`, no locale suffix — no
translatable fields):
```yaml
"38":
  graphic_id: 38
  sof_dna: "cf1_t1:caldaribase:caldari"
  sof_fation_name: "caldaribase"
  sof_hull_name: "cf1_t1"
  sof_race_name: "caldari"
```
`sof_fation_name` (not `sof_faction_name`) is a pre-existing typo preserved verbatim in the
vendored `goesi-openapi` client's own `SofFationName` field (already used unchanged by the prior
migration's `mcp/universe.go`) — the scrape mirrors ESI's real response byte-for-byte, so it
carries the same typo. `CollisionFile` remains absent — confirmed zero non-null values across
every graphic record in the entire scrape file, not merely this sample.

- [ ] **Step 1: Write the failing tests**

Modify `sde/graphic_test.go` — add after `TestGraphic_KnownID_ReturnsSOFFields`:
```go

func TestGraphic_KnownID_ReturnsSofDNA(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 38,
		`{"_key": 38, "sofDna": "cf1_t1:caldaribase:caldari", "sofHullName": "cf1_t1", "sofRaceName": "caldari"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 38)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.SofDNA == nil || *got.SofDNA != "cf1_t1:caldaribase:caldari" {
		t.Errorf("SofDNA = %v, want \"cf1_t1:caldaribase:caldari\"", got.SofDNA)
	}
}

func TestGraphic_NoSofDNA_ReturnsNil(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 10, `{"_key": 10, "graphicFile": "res:/x"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 10)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.SofDNA != nil {
		t.Errorf("SofDNA = %v, want nil", got.SofDNA)
	}
}
```

Also modify `TestPutGraphic_ThenGraphic_RoundTrips` (already in `sde/graphic_test.go`) to also
assert `SofDNA` round-trips:
```go
func TestPutGraphic_ThenGraphic_RoundTrips(t *testing.T) {
	store := seedStore(t)
	file := "res:/dx9/model/worldobject/newthing.red"
	race := "caldari"
	sofDNA := "cf1_t1:caldaribase:caldari"
	want := Graphic{ID: 99, GraphicFile: &file, SofRaceName: &race, SofDNA: &sofDNA}

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
	if got.ID != want.ID || *got.GraphicFile != *want.GraphicFile || *got.SofRaceName != *want.SofRaceName || *got.SofDNA != *want.SofDNA {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sde/... -run 'TestGraphic_KnownID_ReturnsSofDNA|TestGraphic_NoSofDNA|TestPutGraphic_ThenGraphic_RoundTrips' -v`
Expected: FAIL to compile — `got.SofDNA undefined`.

- [ ] **Step 3: Write the minimal implementation**

Modify `sde/graphic.go`:
```go
// Graphic is a minimal decode of an SDE graphic record, augmented at sync
// time with SofDNA from EVE Ref's esi-scrape dataset. CollisionFile is
// deliberately not modeled -- confirmed absent from every one of the
// archive's 6051 graphics.jsonl records AND from every record in the
// esi-scrape mirror of ESI's own live responses, not merely a sample
// coincidence in either source.
type Graphic struct {
	ID             int64
	GraphicFile    *string
	IconFolder     *string
	SofDNA         *string
	SofFactionName *string
	SofHullName    *string
	SofRaceName    *string
}

type graphicRecord struct {
	Key            int64   `json:"_key"`
	GraphicFile    *string `json:"graphicFile,omitempty"`
	IconFolder     *string `json:"iconFolder,omitempty"`
	SofDNA         *string `json:"sofDna,omitempty"`
	SofFactionName *string `json:"sofFactionName,omitempty"`
	SofHullName    *string `json:"sofHullName,omitempty"`
	SofRaceName    *string `json:"sofRaceName,omitempty"`
}
```
In `Graphic()`, change the `out, ok = Graphic{...}, true` construction to:
```go
		out, ok = Graphic{
			ID: raw.Key, GraphicFile: raw.GraphicFile, IconFolder: raw.IconFolder, SofDNA: raw.SofDNA,
			SofFactionName: raw.SofFactionName, SofHullName: raw.SofHullName, SofRaceName: raw.SofRaceName,
		}, true
```
In `PutGraphic()`, change the `raw, err := json.Marshal(graphicRecord{...})` construction to:
```go
	raw, err := json.Marshal(graphicRecord{
		Key: g.ID, GraphicFile: g.GraphicFile, IconFolder: g.IconFolder, SofDNA: g.SofDNA,
		SofFactionName: g.SofFactionName, SofHullName: g.SofHullName, SofRaceName: g.SofRaceName,
	})
```
`GraphicIDs()` is unchanged.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./sde/... -run TestGraphic -v && go test ./sde/... -run TestPutGraphic -v`
Expected: PASS — every `TestGraphic_*`/`TestPutGraphic_*` test, including the pre-existing ones.

- [ ] **Step 5: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race`
Expected: clean build/vet/gofmt, all `sde/` tests pass under `-race`.

- [ ] **Step 6: Commit**

```bash
git add sde/graphic.go sde/graphic_test.go
git commit -m "feat: add Graphic.SofDNA field"
```

---

### Task 6: `sync.go` rewrite — fetch and wire in both new sources (riskiest task — give this its own focused review)

**Files:**
- Modify: `sde/store.go` (add two overridable URL fields, same shape as the existing `indexURL`)
- Modify: `sde/sync.go` (replace whole file)
- Modify: `sde/sync_test.go` (replace whole file)

**Interfaces:**
- Consumes: `asteroidBeltRecord`/`moonRecord`/`planetRecord`/`stargateRecord`/`starRecord`/
  `stationRecord`/`systemPlanetRecord` (Tasks 1/2/4, unexported, same package); `Position` (prior
  plan); `typeRecord`/`raceRecord`/`graphicRecord` with their new fields (Tasks 3/5).
- Produces: an updated `(s *Store) sync(ctx, userAgent) error` that also fetches `reference-data`
  and `esi-scrape`, writes 6 new buckets (`asteroidBelts`, `moons`, `planets`, `stargates`,
  `stars`, `stations`), and augments `types`/`races`/`systems`/`graphics` — same external signature
  and atomicity contract as before (Tasks 1–5's accessors need no further changes).

**Why `store.go` needs two new fields:** `reference-data`/`esi-scrape` are fetched from hardcoded
URLs, unlike the FC SDE zip (discovered via `s.indexURL`'s `index.json`). Without an overridable
field on `Store` the same shape as `indexURL` already has, `sync_test.go`'s fixture server
couldn't redirect these two fetches to a local `httptest.Server`, and every sync test would make
real network calls. `referenceDataURL`/`esiScrapeURL` follow `indexURL`'s exact precedent:
unexported fields, defaulted in `Open()`, overridden directly by internal-package tests exactly
the way `openWithIndexURL` already overrides `indexURL`.

**Confirmed URLs and member paths** (verified live during design):
- `reference-data`: `https://data.everef.net/reference-data/reference-data-latest.tar.xz`,
  member `types.json` (flat, no directory prefix).
- `esi-scrape`: `https://data.everef.net/esi-scrape/eve-ref-esi-scrape-latest.tar.xz`, members
  under `eve-ref-esi-scrape/data/tranquility/universe/` — `asteroid_belts.en-us.yaml`,
  `moons.en-us.yaml`, `planets.en-us.yaml`, `stargates.en-us.yaml`, `stars.en-us.yaml`,
  `stations.en-us.yaml`, `races.en-us.yaml`, `systems.en-us.yaml` (all locale-suffixed but
  content-identical across locales), and `graphics.yaml` (no locale suffix).

**Failure isolation (confirmed):** the SDE zip fetch stays sync-fatal, unchanged. Both new fetches
are wrapped so any failure (network, extraction, or parse) logs one `slog.Warn` and returns a
zero-value result (`nil` map / empty `esiScrapeData`) instead of an error — every downstream
augmentation and the 6 new bucket-writers treat a zero-value input as "nothing to write this
cycle," not a crash. A single parse failure within `esi-scrape`'s 9 members fails that whole
fetch (not just the one member) — a format change severe enough to break one member likely
affects the others too, and partial augmentation would be harder to reason about than none.

- [ ] **Step 1: Add the overridable URL fields to `Store`**

Modify `sde/store.go`:
```go
package sde

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

const defaultIndexURL = "https://data.everef.net/ccp/sde/index.json"

// Store is a local cache of FC's Static Data Export, backed by bbolt.
// Opening a Store does not sync it -- the first lookup call does that
// lazily, at most once per Store's lifetime.
type Store struct {
	db               *bbolt.DB
	indexURL         string
	referenceDataURL string
	esiScrapeURL     string

	syncOnce sync.Once
	syncErr  error
}

// Open opens (creating if necessary) the bbolt database at path. A 5-second
// timeout on the file lock means a losing opener (e.g. two neat processes
// racing to open the same ~/.neat/sde/sde.db) fails fast with an error
// instead of blocking forever -- bbolt's default Options.Timeout of 0 waits
// indefinitely.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create sde cache directory: %w", err)
	}
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open sde cache: %w", err)
	}
	return &Store{db: db, indexURL: defaultIndexURL, referenceDataURL: referenceDataURL, esiScrapeURL: esiScrapeURL}, nil
}

// DefaultPath returns ~/.neat/sde/sde.db.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".neat", "sde", "sde.db"), nil
}

// Close closes the underlying bbolt database.
func (s *Store) Close() error {
	return s.db.Close()
}
```
(`referenceDataURL`/`esiScrapeURL` package-level constants are defined in `sync.go`, Step 3 below
— matching `defaultIndexURL`'s pattern of living alongside the fetch code that uses it.)

- [ ] **Step 2: Run build to verify it's still clean (constants not yet defined)**

Run: `go build ./... 2>&1 | head -5`
Expected: FAIL — `undefined: referenceDataURL`, `undefined: esiScrapeURL` (Step 3 defines them;
this is expected, not a real regression to fix here).

- [ ] **Step 3: Replace `sde/sync_test.go` with the new fixture builders and tests**

Modify `sde/sync_test.go` — replace the whole file:
```go
package sde

import (
	"archive/tar"
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

	"github.com/ulikunitz/xz"
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

// buildTarXZ builds an in-memory .tar.xz archive from name -> content,
// used to fixture both reference-data and esi-scrape in tests without any
// network dependency.
func buildTarXZ(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0o644}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content %s: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	var xzBuf bytes.Buffer
	xw, err := xz.NewWriter(&xzBuf)
	if err != nil {
		t.Fatalf("create xz writer: %v", err)
	}
	if _, err := xw.Write(tarBuf.Bytes()); err != nil {
		t.Fatalf("write xz content: %v", err)
	}
	if err := xw.Close(); err != nil {
		t.Fatalf("close xz writer: %v", err)
	}
	return xzBuf.Bytes()
}

// fixtureReferenceDataTarXZ builds a minimal reference-data archive with
// just enough of types.json to exercise the PackagedVolume join for type
// 587 (Rifter, matching fullFixtureFiles's types.jsonl entry).
func fixtureReferenceDataTarXZ(t *testing.T) []byte {
	t.Helper()
	return buildTarXZ(t, map[string]string{
		"types.json": `{"587": {"packaged_volume": 2500}}`,
	})
}

// fixtureEsiScrapeTarXZ builds a minimal esi-scrape archive with just
// enough of each of the 9 needed members to exercise every augmentation
// this task adds, cross-linked with fullFixtureFiles's IDs where it
// matters (system 30000001, race 1).
func fixtureEsiScrapeTarXZ(t *testing.T) []byte {
	t.Helper()
	const prefix = "eve-ref-esi-scrape/data/tranquility/universe/"
	return buildTarXZ(t, map[string]string{
		prefix + "asteroid_belts.en-us.yaml": `"40000003":
  name: "Tanoo I - Asteroid Belt 1"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
`,
		prefix + "moons.en-us.yaml": `"40000004":
  name: "Tanoo I - Moon 1"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
`,
		prefix + "planets.en-us.yaml": `"40000002":
  name: "Tanoo I"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
  type_id: 11
`,
		prefix + "stargates.en-us.yaml": `"50000001":
  destination: {stargate_id: 50000482, system_id: 30000778}
  name: "Stargate (G-EURJ)"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
  type_id: 29633
`,
		prefix + "stars.en-us.yaml": `"40000001":
  age: 14262808228
  luminosity: 0.01575
  name: "Tanoo - Star"
  radius: 63350000
  solar_system_id: 30000001
  spectral_class: "K2 V"
  temperature: 4567
  type_id: 45041
`,
		prefix + "stations.en-us.yaml": `"60000001":
  max_dockable_ship_volume: 50000000
  name: "Test Station"
  office_rental_cost: 3627229
  owner: 1000035
  position: {x: 1, "y": 2, z: 3}
  race_id: 1
  reprocessing_efficiency: 0.5
  reprocessing_stations_take: 0.05
  services: ["docking"]
  system_id: 30000001
  type_id: 1531
`,
		prefix + "races.en-us.yaml": `- alliance_id: 500001
  race_id: 1
`,
		prefix + "systems.en-us.yaml": `"30000001":
  planets:
    - planet_id: 40000002
      moons: [40000004]
      asteroid_belts: [40000003]
`,
		prefix + "graphics.yaml": `"10":
  sof_dna: "cf1_t1:caldaribase:caldari"
`,
	})
}

// fixtureServer serves an index.json (pointing back at its own /sde.zip)
// and the zip body itself, counting hits to each so tests can assert
// exactly how many times each endpoint was actually called. It also serves
// fixture reference-data/esi-scrape archives at fixed paths, since those
// two sources are fetched from hardcoded URLs rather than discovered via
// an index.json.
type fixtureServer struct {
	*httptest.Server
	indexHits         atomic.Int32
	zipHits           atomic.Int32
	referenceDataHits atomic.Int32
	esiScrapeHits     atomic.Int32
	etag              string
	zipBody           []byte
	referenceDataBody []byte
	esiScrapeBody     []byte
}

func newFixtureServer(t *testing.T, etag string, zipBody []byte) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{
		etag: etag, zipBody: zipBody,
		referenceDataBody: fixtureReferenceDataTarXZ(t),
		esiScrapeBody:     fixtureEsiScrapeTarXZ(t),
	}
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
	mux.HandleFunc("/reference-data.tar.xz", func(w http.ResponseWriter, r *http.Request) {
		fs.referenceDataHits.Add(1)
		_, _ = w.Write(fs.referenceDataBody)
	})
	mux.HandleFunc("/esi-scrape.tar.xz", func(w http.ResponseWriter, r *http.Request) {
		fs.esiScrapeHits.Add(1)
		_, _ = w.Write(fs.esiScrapeBody)
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

// openWithFixtureServer is like openWithIndexURL, but also redirects
// referenceDataURL/esiScrapeURL to the same fixture server -- every sync
// test that needs the full three-source pipeline uses this instead.
func openWithFixtureServer(t *testing.T, srv *fixtureServer) *Store {
	t.Helper()
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")
	store.referenceDataURL = srv.Server.URL + "/reference-data.tar.xz"
	store.esiScrapeURL = srv.Server.URL + "/esi-scrape.tar.xz"
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

// openAtPathWithFixtureServer is the openAtPathWithIndexURL analogue of
// openWithFixtureServer, for tests that reopen the same db path as a
// second Store.
func openAtPathWithFixtureServer(t *testing.T, path string, srv *fixtureServer) *Store {
	t.Helper()
	store := openAtPathWithIndexURL(t, path, srv.Server.URL+"/index.json")
	store.referenceDataURL = srv.Server.URL + "/reference-data.tar.xz"
	store.esiScrapeURL = srv.Server.URL + "/esi-scrape.tar.xz"
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
		Published bool               `json:"published"`
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
	store := openWithFixtureServer(t, srv)

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
	if typ.PackagedVolume == nil || *typ.PackagedVolume != 2500 {
		t.Errorf("Type(587).PackagedVolume = %v, want 2500 (from reference-data)", typ.PackagedVolume)
	}

	sys, ok, err := store.System(context.Background(), testUserAgent, 30000001)
	if err != nil || !ok {
		t.Fatalf("System(30000001) = %+v, %v, %v", sys, ok, err)
	}
	if len(sys.StationIDs) != 1 || sys.StationIDs[0] != 60000001 {
		t.Errorf("System(30000001).StationIDs = %v, want [60000001] (reverse index from npcStations.jsonl)", sys.StationIDs)
	}
	if len(sys.Planets) != 1 || sys.Planets[0].PlanetID != 40000002 || len(sys.Planets[0].MoonIDs) != 1 || len(sys.Planets[0].AsteroidBeltIDs) != 1 {
		t.Errorf("System(30000001).Planets = %+v, want one entry PlanetID=40000002 with 1 moon and 1 belt (from esi-scrape)", sys.Planets)
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

	races, err := store.Races(context.Background(), testUserAgent)
	if err != nil || len(races) != 1 {
		t.Errorf("Races = %+v, %v, want one entry", races, err)
	}
	if races[0].AllianceID == nil || *races[0].AllianceID != 500001 {
		t.Errorf("Races[0].AllianceID = %v, want 500001 (from esi-scrape)", races[0].AllianceID)
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
	graphic, ok, err := store.Graphic(context.Background(), testUserAgent, 10)
	if err != nil || !ok || graphic.SofDNA == nil || *graphic.SofDNA != "cf1_t1:caldaribase:caldari" {
		t.Errorf("Graphic(10) = %+v, %v, %v, want SofDNA=cf1_t1:caldaribase:caldari (from esi-scrape)", graphic, ok, err)
	}

	belt, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40000003)
	if err != nil || !ok || belt.Name != "Tanoo I - Asteroid Belt 1" {
		t.Errorf("AsteroidBelt(40000003) = %+v, %v, %v, want Name=\"Tanoo I - Asteroid Belt 1\"", belt, ok, err)
	}
	moon, ok, err := store.Moon(context.Background(), testUserAgent, 40000004)
	if err != nil || !ok || moon.Name != "Tanoo I - Moon 1" {
		t.Errorf("Moon(40000004) = %+v, %v, %v, want Name=\"Tanoo I - Moon 1\"", moon, ok, err)
	}
	planet, ok, err := store.Planet(context.Background(), testUserAgent, 40000002)
	if err != nil || !ok || planet.Name != "Tanoo I" {
		t.Errorf("Planet(40000002) = %+v, %v, %v, want Name=\"Tanoo I\"", planet, ok, err)
	}
	stargate, ok, err := store.Stargate(context.Background(), testUserAgent, 50000001)
	if err != nil || !ok || stargate.DestinationStargateID != 50000482 {
		t.Errorf("Stargate(50000001) = %+v, %v, %v, want DestinationStargateID=50000482", stargate, ok, err)
	}
	star, ok, err := store.Star(context.Background(), testUserAgent, 40000001)
	if err != nil || !ok || star.SpectralClass != "K2 V" {
		t.Errorf("Star(40000001) = %+v, %v, %v, want SpectralClass=\"K2 V\"", star, ok, err)
	}
	station, ok, err := store.Station(context.Background(), testUserAgent, 60000001)
	if err != nil || !ok || station.Name != "Test Station" {
		t.Errorf("Station(60000001) = %+v, %v, %v, want Name=\"Test Station\"", station, ok, err)
	}

	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times, want exactly 1", srv.zipHits.Load())
	}
	if srv.referenceDataHits.Load() != 1 {
		t.Errorf("reference-data requested %d times, want exactly 1", srv.referenceDataHits.Load())
	}
	if srv.esiScrapeHits.Load() != 1 {
		t.Errorf("esi-scrape requested %d times, want exactly 1", srv.esiScrapeHits.Load())
	}
}

func TestSync_ReferenceDataFetchFails_RestOfSyncStillSucceeds(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)
	store.referenceDataURL = "http://127.0.0.1:1/does-not-exist" // guaranteed connection failure

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v, want nil (reference-data failure must not be fatal)", err)
	}

	assertRecord(t, store, "categories", 6, "Ship", true)

	typ, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil || !ok {
		t.Fatalf("Type(587) = %+v, %v, %v", typ, ok, err)
	}
	if typ.PackagedVolume != nil {
		t.Errorf("Type(587).PackagedVolume = %v, want nil (reference-data fetch failed, no augmentation this cycle)", typ.PackagedVolume)
	}
}

func TestSync_EsiScrapeFetchFails_RestOfSyncStillSucceeds(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)
	store.esiScrapeURL = "http://127.0.0.1:1/does-not-exist" // guaranteed connection failure

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v, want nil (esi-scrape failure must not be fatal)", err)
	}

	assertRecord(t, store, "categories", 6, "Ship", true)

	races, err := store.Races(context.Background(), testUserAgent)
	if err != nil || len(races) != 1 {
		t.Fatalf("Races = %+v, %v, want one entry (races.jsonl itself still loads from the SDE)", races, err)
	}
	if races[0].AllianceID != nil {
		t.Errorf("Races[0].AllianceID = %v, want nil (esi-scrape fetch failed, no augmentation this cycle)", races[0].AllianceID)
	}
	if bucketExists(t, store, "asteroidBelts") {
		t.Error("asteroidBelts bucket exists after an esi-scrape fetch failure, want no bucket (nothing to write)")
	}

	belt, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40000003)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if ok {
		t.Errorf("AsteroidBelt ok = true, want false: %+v", belt)
	}
}

func TestSync_OnceOnlyPerStoreLifetime(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)

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
	store := openWithFixtureServer(t, srv)

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

	first := openAtPathWithFixtureServer(t, dbPath, srv)
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}
	if srv.zipHits.Load() != 1 {
		t.Fatalf("zip requested %d times on first sync, want exactly 1", srv.zipHits.Load())
	}

	second := openAtPathWithFixtureServer(t, dbPath, srv)
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
	first := openAtPathWithFixtureServer(t, dbPath, srv1)
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}

	srv2 := newFixtureServer(t, "etag-v2", buildFixtureZipWithRenamedShipCategory(t))
	second := openAtPathWithFixtureServer(t, dbPath, srv2)
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
	first := openAtPathWithFixtureServer(t, dbPath, srv)
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
	store := openWithFixtureServer(t, srv)

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
	store := openWithFixtureServer(t, srv)

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
	for _, bucket := range []string{
		"categories", "groups", "types", "regions", "constellations", "systems", "factions",
		"ancestries", "bloodlines", "races", "graphics",
		"asteroidBelts", "moons", "planets", "stargates", "stars", "stations",
	} {
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
	store := openWithFixtureServer(t, srv)

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the archive is missing an expected JSONL file")
	}
}

func TestCategory_TriggersRealSyncOnNeverSyncedStore(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)

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

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./sde/... -run 'TestSync_FirstSyncLoadsAllResourcesWithAugmentation|TestSync_ReferenceDataFetchFails|TestSync_EsiScrapeFetchFails' -v 2>&1 | head -20`
Expected: FAIL to compile — `store.referenceDataURL undefined`, `undefined: buildTarXZ`, and the
new assertions (`typ.PackagedVolume`, `sys.Planets`, `store.AsteroidBelt`, etc.) don't exist yet
against the current `sync()`.

- [ ] **Step 5: Write the two-phase sync pipeline with both new sources wired in**

Replace the whole `sde/sync.go` file:
```go
package sde

import (
	"archive/tar"
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
	"strconv"
	"time"

	"github.com/ulikunitz/xz"
	"go.etcd.io/bbolt"
	"go.yaml.in/yaml/v3"
)

const sdeZipName = "eve-online-static-data-latest-jsonl.zip"
const referenceDataURL = "https://data.everef.net/reference-data/reference-data-latest.tar.xz"
const esiScrapeURL = "https://data.everef.net/esi-scrape/eve-ref-esi-scrape-latest.tar.xz"
const esiScrapePathPrefix = "eve-ref-esi-scrape/data/tranquility/universe/"

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

	// reference-data and esi-scrape are best-effort: a failure here logs a
	// warning and degrades to an empty result, never fails the sync the
	// core SDE zip is about to perform below.
	packagedVolumes := fetchReferenceDataTypeVolumes(ctx, s.referenceDataURL, userAgent)
	scrapeData := fetchEsiScrapeSupplements(ctx, s.esiScrapeURL, userAgent)

	return s.db.Update(func(tx *bbolt.Tx) error {
		// Types/groups/categories: each level needs the level below it's
		// reverse index before it can be written.
		dogma, err := loadTypeDogma(zr)
		if err != nil {
			return fmt.Errorf("load typeDogma.jsonl: %w", err)
		}
		typeIDsByGroup, err := loadTypes(tx, zr, dogma, packagedVolumes)
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
		if err := loadSystems(tx, zr, stationsBySystem, scrapeData.systemPlanets); err != nil {
			return fmt.Errorf("load mapSolarSystems.jsonl: %w", err)
		}
		if err := loadFactions(tx, zr, factionAgg); err != nil {
			return fmt.Errorf("load factions.jsonl: %w", err)
		}

		// races/graphics need one esi-scrape-sourced merge each; every
		// other passthrough resource needs no augmentation at all.
		if err := loadRaces(tx, zr, scrapeData.raceAlliances); err != nil {
			return fmt.Errorf("load races.jsonl: %w", err)
		}
		if err := loadGraphics(tx, zr, scrapeData.graphicSofDNA); err != nil {
			return fmt.Errorf("load graphics.jsonl: %w", err)
		}
		passthrough := []struct{ file, bucket string }{
			{"mapRegions.jsonl", "regions"},
			{"mapConstellations.jsonl", "constellations"},
			{"ancestries.jsonl", "ancestries"},
			{"bloodlines.jsonl", "bloodlines"},
		}
		for _, p := range passthrough {
			if err := loadResource(tx, zr, p.file, p.bucket); err != nil {
				return fmt.Errorf("load %s: %w", p.file, err)
			}
		}

		// The 6 celestial-detail resources have no official SDE source at
		// all -- entirely from esi-scrape (or empty, if that fetch failed
		// this cycle; every Store accessor already falls through to a live
		// ESI call on a cache miss regardless of why the cache is empty).
		if err := loadAsteroidBelts(tx, scrapeData.asteroidBelts); err != nil {
			return fmt.Errorf("load esi-scrape asteroid belts: %w", err)
		}
		if err := loadMoons(tx, scrapeData.moons); err != nil {
			return fmt.Errorf("load esi-scrape moons: %w", err)
		}
		if err := loadPlanets(tx, scrapeData.planets); err != nil {
			return fmt.Errorf("load esi-scrape planets: %w", err)
		}
		if err := loadStargates(tx, scrapeData.stargates); err != nil {
			return fmt.Errorf("load esi-scrape stargates: %w", err)
		}
		if err := loadStars(tx, scrapeData.stars); err != nil {
			return fmt.Errorf("load esi-scrape stars: %w", err)
		}
		if err := loadStations(tx, scrapeData.stations); err != nil {
			return fmt.Errorf("load esi-scrape stations: %w", err)
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

// loadTypes writes the types bucket (augmented with dogma and, if
// packagedVolumes is non-nil, PackagedVolume from reference-data) and
// returns the groupID -> []typeID reverse index groups.jsonl needs next.
func loadTypes(tx *bbolt.Tx, zr *zip.Reader, dogma map[int64]dogmaData, packagedVolumes map[int64]float64) (map[int64][]int64, error) {
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
		if v, ok := packagedVolumes[key]; ok {
			rec.PackagedVolume = &v
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
// field of its own) and, if systemPlanets is non-nil, Planets from
// esi-scrape (mapSolarSystems.jsonl has no planets field either).
func loadSystems(tx *bbolt.Tx, zr *zip.Reader, stationsBySystem map[int64][]int64, systemPlanets map[int64][]systemPlanetRecord) error {
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
		rec.Planets = systemPlanets[key]
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

// loadRaces writes the races bucket, augmented with AllianceID from
// esi-scrape (races.jsonl has no such field, and there is no
// alliances.jsonl anywhere in the SDE archive at all).
func loadRaces(tx *bbolt.Tx, zr *zip.Reader, raceAlliances map[int64]int64) error {
	if err := tx.DeleteBucket([]byte("races")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear races bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("races"))
	if err != nil {
		return fmt.Errorf("create races bucket: %w", err)
	}
	return scanRecords(zr, "races.jsonl", func(key int64, raw []byte) error {
		var rec raceRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse race record %d: %w", key, err)
		}
		if allianceID, ok := raceAlliances[key]; ok {
			rec.AllianceID = &allianceID
		}
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented race %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// loadGraphics writes the graphics bucket, augmented with SofDNA from
// esi-scrape (graphics.jsonl has no such field).
func loadGraphics(tx *bbolt.Tx, zr *zip.Reader, graphicSofDNA map[int64]string) error {
	if err := tx.DeleteBucket([]byte("graphics")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear graphics bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("graphics"))
	if err != nil {
		return fmt.Errorf("create graphics bucket: %w", err)
	}
	return scanRecords(zr, "graphics.jsonl", func(key int64, raw []byte) error {
		var rec graphicRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse graphic record %d: %w", key, err)
		}
		if dna, ok := graphicSofDNA[key]; ok {
			rec.SofDNA = &dna
		}
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented graphic %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// loadAsteroidBelts writes the asteroidBelts bucket directly from
// esi-scrape records -- no augmentation, no official SDE source at all.
// records is nil (a no-op) if esi-scrape couldn't be fetched this cycle.
func loadAsteroidBelts(tx *bbolt.Tx, records map[int64]asteroidBeltRecord) error {
	return writeEsiScrapeBucket(tx, "asteroidBelts", records)
}

// loadMoons is loadAsteroidBelts' analogue for moons.
func loadMoons(tx *bbolt.Tx, records map[int64]moonRecord) error {
	return writeEsiScrapeBucket(tx, "moons", records)
}

// loadPlanets is loadAsteroidBelts' analogue for planets.
func loadPlanets(tx *bbolt.Tx, records map[int64]planetRecord) error {
	return writeEsiScrapeBucket(tx, "planets", records)
}

// loadStargates is loadAsteroidBelts' analogue for stargates.
func loadStargates(tx *bbolt.Tx, records map[int64]stargateRecord) error {
	return writeEsiScrapeBucket(tx, "stargates", records)
}

// loadStars is loadAsteroidBelts' analogue for stars.
func loadStars(tx *bbolt.Tx, records map[int64]starRecord) error {
	return writeEsiScrapeBucket(tx, "stars", records)
}

// loadStations is loadAsteroidBelts' analogue for stations.
func loadStations(tx *bbolt.Tx, records map[int64]stationRecord) error {
	return writeEsiScrapeBucket(tx, "stations", records)
}

// writeEsiScrapeBucket clears and rewrites bucket from an already-decoded
// esi-scrape record map, keyed by map key. Shared by all 6 celestial
// loaders above -- unlike scanRecords-based loaders, there's no zip/JSONL
// file to stream, just an in-memory map to marshal and write.
func writeEsiScrapeBucket[T any](tx *bbolt.Tx, bucket string, records map[int64]T) error {
	if err := tx.DeleteBucket([]byte(bucket)); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear %s bucket: %w", bucket, err)
	}
	b, err := tx.CreateBucket([]byte(bucket))
	if err != nil {
		return fmt.Errorf("create %s bucket: %w", bucket, err)
	}
	for id, rec := range records {
		data, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal %s record %d: %w", bucket, id, err)
		}
		if err := b.Put(int64Key(id), data); err != nil {
			return fmt.Errorf("write %s record %d: %w", bucket, id, err)
		}
	}
	return nil
}

// esiScrapePosition mirrors sde.Position's shape for YAML decoding --
// Position itself has json tags, not yaml tags, so a separate decode type
// is needed; Position(esiScrapePositionValue) is a valid Go conversion
// since both have identical field names/types/order (struct tags don't
// affect conversion compatibility).
type esiScrapePosition struct {
	X float64 `yaml:"x"`
	Y float64 `yaml:"y"`
	Z float64 `yaml:"z"`
}

type esiScrapeAsteroidBelt struct {
	Name     string            `yaml:"name"`
	Position esiScrapePosition `yaml:"position"`
	SystemID int64             `yaml:"system_id"`
}

type esiScrapeMoon struct {
	Name     string            `yaml:"name"`
	Position esiScrapePosition `yaml:"position"`
	SystemID int64             `yaml:"system_id"`
}

type esiScrapePlanet struct {
	Name     string            `yaml:"name"`
	Position esiScrapePosition `yaml:"position"`
	SystemID int64             `yaml:"system_id"`
	TypeID   int64             `yaml:"type_id"`
}

type esiScrapeStargateDestination struct {
	StargateID int64 `yaml:"stargate_id"`
	SystemID   int64 `yaml:"system_id"`
}

type esiScrapeStargate struct {
	Destination esiScrapeStargateDestination `yaml:"destination"`
	Name        string                       `yaml:"name"`
	Position    esiScrapePosition            `yaml:"position"`
	SystemID    int64                        `yaml:"system_id"`
	TypeID      int64                        `yaml:"type_id"`
}

type esiScrapeStar struct {
	Age           int64   `yaml:"age"`
	Luminosity    float64 `yaml:"luminosity"`
	Name          string  `yaml:"name"`
	Radius        int64   `yaml:"radius"`
	SolarSystemID int64   `yaml:"solar_system_id"`
	SpectralClass string  `yaml:"spectral_class"`
	Temperature   int64   `yaml:"temperature"`
	TypeID        int64   `yaml:"type_id"`
}

type esiScrapeStation struct {
	MaxDockableShipVolume    float64           `yaml:"max_dockable_ship_volume"`
	Name                     string            `yaml:"name"`
	OfficeRentalCost         float64           `yaml:"office_rental_cost"`
	Owner                    *int64            `yaml:"owner"`
	Position                 esiScrapePosition `yaml:"position"`
	RaceID                   *int64            `yaml:"race_id"`
	ReprocessingEfficiency   float64           `yaml:"reprocessing_efficiency"`
	ReprocessingStationsTake float64           `yaml:"reprocessing_stations_take"`
	Services                 []string          `yaml:"services"`
	SystemID                 int64             `yaml:"system_id"`
	TypeID                   int64             `yaml:"type_id"`
}

type esiScrapeRace struct {
	AllianceID int64 `yaml:"alliance_id"`
	RaceID     int64 `yaml:"race_id"`
}

type esiScrapeSystemPlanet struct {
	PlanetID      int64   `yaml:"planet_id"`
	AsteroidBelts []int64 `yaml:"asteroid_belts,omitempty"`
	Moons         []int64 `yaml:"moons,omitempty"`
}

type esiScrapeSystem struct {
	Planets []esiScrapeSystemPlanet `yaml:"planets"`
}

type esiScrapeGraphic struct {
	SofDNA *string `yaml:"sof_dna"`
}

// esiScrapeData bundles every value fetchEsiScrapeSupplements extracts from
// EVE Ref's esi-scrape dataset. A zero-value esiScrapeData (every field
// nil) means the fetch or parse failed -- every augmentation and
// bucket-write using it becomes a no-op, not a crash; the 6 celestial
// buckets simply don't get (re)populated this sync cycle, and every Store
// accessor already falls through to a live ESI call on a cache miss
// regardless of why the cache is empty.
type esiScrapeData struct {
	raceAlliances map[int64]int64
	systemPlanets map[int64][]systemPlanetRecord
	graphicSofDNA map[int64]string
	asteroidBelts map[int64]asteroidBeltRecord
	moons         map[int64]moonRecord
	planets       map[int64]planetRecord
	stargates     map[int64]stargateRecord
	stars         map[int64]starRecord
	stations      map[int64]stationRecord
}

// fetchReferenceDataTypeVolumes fetches EVE Ref's reference-data dataset
// and returns typeID -> packagedVolume for every type that has one.
// Returns nil (not an error) on any failure -- reference-data is a
// best-effort augmentation, never fatal to the rest of sync().
func fetchReferenceDataTypeVolumes(ctx context.Context, url, userAgent string) map[int64]float64 {
	body, err := fetchBytes(ctx, url, userAgent)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to fetch reference-data, skipping PackagedVolume augmentation", "error", err)
		return nil
	}
	members, err := extractTarXZMembers(body, map[string]bool{"types.json": true})
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to extract reference-data, skipping PackagedVolume augmentation", "error", err)
		return nil
	}
	typesJSON, ok := members["types.json"]
	if !ok {
		slog.Default().WarnContext(ctx, "sde: reference-data archive missing types.json, skipping PackagedVolume augmentation")
		return nil
	}
	var raw map[string]struct {
		PackagedVolume *float64 `json:"packaged_volume"`
	}
	if err := json.Unmarshal(typesJSON, &raw); err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to parse reference-data types.json, skipping PackagedVolume augmentation", "error", err)
		return nil
	}
	out := make(map[int64]float64, len(raw))
	for idStr, rec := range raw {
		if rec.PackagedVolume == nil {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		out[id] = *rec.PackagedVolume
	}
	return out
}

// fetchEsiScrapeSupplements fetches EVE Ref's esi-scrape dataset and
// returns every value this plan sources from it. Returns a zero-value
// esiScrapeData (not an error) on any failure -- esi-scrape is a
// best-effort augmentation, never fatal to the rest of sync().
func fetchEsiScrapeSupplements(ctx context.Context, url, userAgent string) esiScrapeData {
	body, err := fetchBytes(ctx, url, userAgent)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to fetch esi-scrape, skipping celestial/AllianceID/Planets/SofDNA augmentation", "error", err)
		return esiScrapeData{}
	}

	want := map[string]bool{
		esiScrapePathPrefix + "asteroid_belts.en-us.yaml": true,
		esiScrapePathPrefix + "moons.en-us.yaml":           true,
		esiScrapePathPrefix + "planets.en-us.yaml":         true,
		esiScrapePathPrefix + "stargates.en-us.yaml":       true,
		esiScrapePathPrefix + "stars.en-us.yaml":           true,
		esiScrapePathPrefix + "stations.en-us.yaml":        true,
		esiScrapePathPrefix + "races.en-us.yaml":           true,
		esiScrapePathPrefix + "systems.en-us.yaml":         true,
		esiScrapePathPrefix + "graphics.yaml":              true,
	}
	members, err := extractTarXZMembers(body, want)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to extract esi-scrape, skipping celestial/AllianceID/Planets/SofDNA augmentation", "error", err)
		return esiScrapeData{}
	}

	data, err := parseEsiScrapeMembers(members)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to parse esi-scrape, skipping celestial/AllianceID/Planets/SofDNA augmentation", "error", err)
		return esiScrapeData{}
	}
	return data
}

// decodeEsiScrapeMap decodes an esi-scrape YAML file whose top level is a
// map keyed by string-formatted IDs, converting the keys to int64.
func decodeEsiScrapeMap[T any](raw []byte) (map[int64]T, error) {
	if raw == nil {
		return nil, nil
	}
	var strKeyed map[string]T
	if err := yaml.Unmarshal(raw, &strKeyed); err != nil {
		return nil, err
	}
	out := make(map[int64]T, len(strKeyed))
	for idStr, v := range strKeyed {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse key %q as int64: %w", idStr, err)
		}
		out[id] = v
	}
	return out, nil
}

// parseEsiScrapeMembers decodes every member fetchEsiScrapeSupplements
// extracted. Any single parse failure fails the whole batch -- a format
// change significant enough to break one file is likely to affect the
// others too, and partial augmentation would be harder to reason about
// than none.
func parseEsiScrapeMembers(members map[string][]byte) (esiScrapeData, error) {
	var data esiScrapeData

	asteroidBelts, err := decodeEsiScrapeMap[esiScrapeAsteroidBelt](members[esiScrapePathPrefix+"asteroid_belts.en-us.yaml"])
	if err != nil {
		return esiScrapeData{}, fmt.Errorf("parse asteroid_belts.en-us.yaml: %w", err)
	}
	data.asteroidBelts = make(map[int64]asteroidBeltRecord, len(asteroidBelts))
	for id, r := range asteroidBelts {
		data.asteroidBelts[id] = asteroidBeltRecord{Key: id, Name: r.Name, Position: Position(r.Position), SystemID: r.SystemID}
	}

	moons, err := decodeEsiScrapeMap[esiScrapeMoon](members[esiScrapePathPrefix+"moons.en-us.yaml"])
	if err != nil {
		return esiScrapeData{}, fmt.Errorf("parse moons.en-us.yaml: %w", err)
	}
	data.moons = make(map[int64]moonRecord, len(moons))
	for id, r := range moons {
		data.moons[id] = moonRecord{Key: id, Name: r.Name, Position: Position(r.Position), SystemID: r.SystemID}
	}

	planets, err := decodeEsiScrapeMap[esiScrapePlanet](members[esiScrapePathPrefix+"planets.en-us.yaml"])
	if err != nil {
		return esiScrapeData{}, fmt.Errorf("parse planets.en-us.yaml: %w", err)
	}
	data.planets = make(map[int64]planetRecord, len(planets))
	for id, r := range planets {
		data.planets[id] = planetRecord{Key: id, Name: r.Name, TypeID: r.TypeID, Position: Position(r.Position), SystemID: r.SystemID}
	}

	stargates, err := decodeEsiScrapeMap[esiScrapeStargate](members[esiScrapePathPrefix+"stargates.en-us.yaml"])
	if err != nil {
		return esiScrapeData{}, fmt.Errorf("parse stargates.en-us.yaml: %w", err)
	}
	data.stargates = make(map[int64]stargateRecord, len(stargates))
	for id, r := range stargates {
		data.stargates[id] = stargateRecord{
			Key: id, Name: r.Name, TypeID: r.TypeID, Position: Position(r.Position), SystemID: r.SystemID,
			DestinationStargateID: r.Destination.StargateID, DestinationSystemID: r.Destination.SystemID,
		}
	}

	stars, err := decodeEsiScrapeMap[esiScrapeStar](members[esiScrapePathPrefix+"stars.en-us.yaml"])
	if err != nil {
		return esiScrapeData{}, fmt.Errorf("parse stars.en-us.yaml: %w", err)
	}
	data.stars = make(map[int64]starRecord, len(stars))
	for id, r := range stars {
		data.stars[id] = starRecord{
			Key: id, Age: r.Age, Luminosity: r.Luminosity, Name: r.Name, Radius: r.Radius,
			SolarSystemID: r.SolarSystemID, SpectralClass: r.SpectralClass, Temperature: r.Temperature, TypeID: r.TypeID,
		}
	}

	stations, err := decodeEsiScrapeMap[esiScrapeStation](members[esiScrapePathPrefix+"stations.en-us.yaml"])
	if err != nil {
		return esiScrapeData{}, fmt.Errorf("parse stations.en-us.yaml: %w", err)
	}
	data.stations = make(map[int64]stationRecord, len(stations))
	for id, r := range stations {
		data.stations[id] = stationRecord{
			Key: id, Name: r.Name, SystemID: r.SystemID, TypeID: r.TypeID, Position: Position(r.Position),
			MaxDockableShipVolume: r.MaxDockableShipVolume, OfficeRentalCost: r.OfficeRentalCost,
			Owner: r.Owner, RaceID: r.RaceID, ReprocessingEfficiency: r.ReprocessingEfficiency,
			ReprocessingStationsTake: r.ReprocessingStationsTake, Services: r.Services,
		}
	}

	var races []esiScrapeRace
	if raw := members[esiScrapePathPrefix+"races.en-us.yaml"]; raw != nil {
		if err := yaml.Unmarshal(raw, &races); err != nil {
			return esiScrapeData{}, fmt.Errorf("parse races.en-us.yaml: %w", err)
		}
	}
	data.raceAlliances = make(map[int64]int64, len(races))
	for _, r := range races {
		data.raceAlliances[r.RaceID] = r.AllianceID
	}

	systems, err := decodeEsiScrapeMap[esiScrapeSystem](members[esiScrapePathPrefix+"systems.en-us.yaml"])
	if err != nil {
		return esiScrapeData{}, fmt.Errorf("parse systems.en-us.yaml: %w", err)
	}
	data.systemPlanets = make(map[int64][]systemPlanetRecord, len(systems))
	for id, r := range systems {
		if len(r.Planets) == 0 {
			continue
		}
		planetRecs := make([]systemPlanetRecord, len(r.Planets))
		for i, p := range r.Planets {
			planetRecs[i] = systemPlanetRecord{PlanetID: p.PlanetID, AsteroidBeltIDs: p.AsteroidBelts, MoonIDs: p.Moons}
		}
		data.systemPlanets[id] = planetRecs
	}

	var graphics map[string]esiScrapeGraphic
	if raw := members[esiScrapePathPrefix+"graphics.yaml"]; raw != nil {
		if err := yaml.Unmarshal(raw, &graphics); err != nil {
			return esiScrapeData{}, fmt.Errorf("parse graphics.yaml: %w", err)
		}
	}
	data.graphicSofDNA = make(map[int64]string, len(graphics))
	for idStr, g := range graphics {
		if g.SofDNA == nil {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		data.graphicSofDNA[id] = *g.SofDNA
	}

	return data, nil
}

// extractTarXZMembers decompresses an XZ-compressed tar archive and
// returns the raw bytes of every member whose path is in want, keyed by
// that same path. Members not in want are skipped without being read into
// memory. Used for both reference-data and esi-scrape -- neither format
// supports partial/range extraction (XZ isn't seekable), so the whole
// archive is always decompressed regardless of how many members are kept.
func extractTarXZMembers(body []byte, want map[string]bool) (map[string][]byte, error) {
	xr, err := xz.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("open xz stream: %w", err)
	}
	tr := tar.NewReader(xr)
	out := make(map[string][]byte, len(want))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar header: %w", err)
		}
		if !want[hdr.Name] {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read tar member %s: %w", hdr.Name, err)
		}
		out[hdr.Name] = data
	}
	return out, nil
}

// int64Key encodes id as a big-endian byte slice suitable for use as a
// bbolt key.
func int64Key(id int64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(id))
	return key
}

// int64FromKey decodes a key produced by int64Key back into its original
// int64 form.
func int64FromKey(key []byte) int64 {
	return int64(binary.BigEndian.Uint64(key))
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

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./sde/... -v`
Expected: PASS — every test in the `sde/` package, including all of Tasks 1–5's tests and every
test in the rewritten `sync_test.go`.

- [ ] **Step 7: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./sde/... -race -v`
Expected: clean build/vet/gofmt, every `sde/` test passes under `-race`.

- [ ] **Step 8: Commit**

```bash
git add sde/store.go sde/sync.go sde/sync_test.go
git commit -m "feat: fetch reference-data and esi-scrape in sync.go, wire in every augmentation"
```

---

### Task 7: `mcp/universe.go` — migrate the 6 celestial-detail tools to the read-through pattern

**Files:**
- Modify: `mcp/universe.go`
- Modify: `mcp/universe_test.go`

**Interfaces:**
- Consumes: `sde.AsteroidBelt`/`sde.Moon`/`sde.Planet`/`sde.Stargate`/`sde.Star`/`sde.Station` and their
  `Store` accessors + `Put*` methods (Tasks 1/2); `sde.Position` (prior plan).
- Produces: 6 new narrow cache interfaces (`asteroidBeltCache`, `moonCache`, `planetCache`,
  `stargateCache`, `starCache`, `stationCache`) matching the exact shape of every other migrated
  tool's cache interface in this file (`categoryCache`, `regionCache`, etc.); the 6 swappable
  `var getUniverseXxx` functions gain an `sdeStore` parameter as their first argument (after
  `ctx`), matching e.g. `getUniverseRegion`'s signature exactly.

**Confirmed current state** (`mcp/universe.go:893-1115`, read directly before writing this task):
all 6 handlers are currently ESI-direct with no cache parameter at all, e.g.:
```go
var getUniverseAsteroidBelt = func(ctx context.Context, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
	client := esi.NewPublicClient(userAgent)
	b, err := client.UniverseAsteroidBeltDetail(ctx, asteroidBeltID)
	...
}
func getUniverseAsteroidBeltHandler(version string) mcpsdk.ToolHandlerFor[...] { ... }
```
Every one of the 6 handler-constructor functions (`getUniverseAsteroidBeltHandler`, ...
`getUniverseStationHandler`) currently takes only `version string` — Task 8 (next) will need to
add `sdeStore *sde.Store` to each of their call sites in `mcp/server.go`, so this task's Step 5
below updates `mcp/server.go` too.

- [ ] **Step 1: Write the failing tests for all 6 migrated tools**

Modify `mcp/universe_test.go` — add:
```go
type fakeAsteroidBeltCache struct {
	belt    sde.AsteroidBelt
	ok      bool
	err     error
	putErr  error
	putCall *sde.AsteroidBelt
}

func (f *fakeAsteroidBeltCache) AsteroidBelt(ctx context.Context, userAgent string, id int64) (sde.AsteroidBelt, bool, error) {
	return f.belt, f.ok, f.err
}
func (f *fakeAsteroidBeltCache) PutAsteroidBelt(b sde.AsteroidBelt) error {
	f.putCall = &b
	return f.putErr
}

func TestGetUniverseAsteroidBelt_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeAsteroidBeltCache{belt: sde.AsteroidBelt{ID: 40000003, Name: "Tanoo I - Asteroid Belt 1", SystemID: 30000001}, ok: true}
	out, err := getUniverseAsteroidBelt(context.Background(), cache, testUserAgent, 40000003)
	if err != nil {
		t.Fatalf("getUniverseAsteroidBelt returned error: %v", err)
	}
	if out.Name != "Tanoo I - Asteroid Belt 1" || out.SystemID != 30000001 {
		t.Errorf("got %+v, want Name=\"Tanoo I - Asteroid Belt 1\" SystemID=30000001", out)
	}
}

func TestGetUniverseAsteroidBelt_CacheMiss_WritesBackToCache(t *testing.T) {
	cache := &fakeAsteroidBeltCache{ok: false}
	out, err := getUniverseAsteroidBelt(context.Background(), cache, testUserAgent, 40000003)
	if err != nil {
		t.Fatalf("getUniverseAsteroidBelt returned error: %v", err)
	}
	if out.Name == "" {
		t.Error("got empty Name on a cache miss, want a live-ESI-fetched name")
	}
	if cache.putCall == nil {
		t.Fatal("PutAsteroidBelt was not called after a cache miss")
	}
	if cache.putCall.ID != 40000003 {
		t.Errorf("PutAsteroidBelt called with ID %d, want 40000003", cache.putCall.ID)
	}
}

func TestGetUniverseAsteroidBelt_CacheReadError_ReturnsError(t *testing.T) {
	cache := &fakeAsteroidBeltCache{err: errors.New("boom")}
	_, err := getUniverseAsteroidBelt(context.Background(), cache, testUserAgent, 40000003)
	if err == nil {
		t.Fatal("getUniverseAsteroidBelt returned nil error, want the cache read error surfaced")
	}
}

type fakeMoonCache struct {
	moon    sde.Moon
	ok      bool
	err     error
	putErr  error
	putCall *sde.Moon
}

func (f *fakeMoonCache) Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error) {
	return f.moon, f.ok, f.err
}
func (f *fakeMoonCache) PutMoon(m sde.Moon) error {
	f.putCall = &m
	return f.putErr
}

func TestGetUniverseMoon_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeMoonCache{moon: sde.Moon{ID: 40000004, Name: "Tanoo I - Moon 1", SystemID: 30000001}, ok: true}
	out, err := getUniverseMoon(context.Background(), cache, testUserAgent, 40000004)
	if err != nil {
		t.Fatalf("getUniverseMoon returned error: %v", err)
	}
	if out.MoonID != 40000004 || out.Name != "Tanoo I - Moon 1" {
		t.Errorf("got %+v, want MoonID=40000004 Name=\"Tanoo I - Moon 1\"", out)
	}
}

func TestGetUniverseMoon_CacheMiss_WritesBackToCache(t *testing.T) {
	cache := &fakeMoonCache{ok: false}
	out, err := getUniverseMoon(context.Background(), cache, testUserAgent, 40000004)
	if err != nil {
		t.Fatalf("getUniverseMoon returned error: %v", err)
	}
	if out.Name == "" {
		t.Error("got empty Name on a cache miss, want a live-ESI-fetched name")
	}
	if cache.putCall == nil {
		t.Fatal("PutMoon was not called after a cache miss")
	}
}

type fakePlanetCache struct {
	planet  sde.Planet
	ok      bool
	err     error
	putErr  error
	putCall *sde.Planet
}

func (f *fakePlanetCache) Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error) {
	return f.planet, f.ok, f.err
}
func (f *fakePlanetCache) PutPlanet(p sde.Planet) error {
	f.putCall = &p
	return f.putErr
}

func TestGetUniversePlanet_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakePlanetCache{planet: sde.Planet{ID: 40000002, Name: "Tanoo I", TypeID: 11, SystemID: 30000001}, ok: true}
	out, err := getUniversePlanet(context.Background(), cache, testUserAgent, 40000002)
	if err != nil {
		t.Fatalf("getUniversePlanet returned error: %v", err)
	}
	if out.Name != "Tanoo I" || out.TypeID != 11 {
		t.Errorf("got %+v, want Name=\"Tanoo I\" TypeID=11", out)
	}
}

func TestGetUniversePlanet_CacheMiss_WritesBackToCache(t *testing.T) {
	cache := &fakePlanetCache{ok: false}
	out, err := getUniversePlanet(context.Background(), cache, testUserAgent, 40000002)
	if err != nil {
		t.Fatalf("getUniversePlanet returned error: %v", err)
	}
	if out.Name == "" {
		t.Error("got empty Name on a cache miss, want a live-ESI-fetched name")
	}
	if cache.putCall == nil {
		t.Fatal("PutPlanet was not called after a cache miss")
	}
}

type fakeStargateCache struct {
	stargate sde.Stargate
	ok       bool
	err      error
	putErr   error
	putCall  *sde.Stargate
}

func (f *fakeStargateCache) Stargate(ctx context.Context, userAgent string, id int64) (sde.Stargate, bool, error) {
	return f.stargate, f.ok, f.err
}
func (f *fakeStargateCache) PutStargate(g sde.Stargate) error {
	f.putCall = &g
	return f.putErr
}

func TestGetUniverseStargate_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeStargateCache{stargate: sde.Stargate{ID: 50000001, Name: "Stargate (G-EURJ)", DestinationStargateID: 50000482}, ok: true}
	out, err := getUniverseStargate(context.Background(), cache, testUserAgent, 50000001)
	if err != nil {
		t.Fatalf("getUniverseStargate returned error: %v", err)
	}
	if out.Name != "Stargate (G-EURJ)" || out.DestinationStargateID != 50000482 {
		t.Errorf("got %+v, want Name=\"Stargate (G-EURJ)\" DestinationStargateID=50000482", out)
	}
}

func TestGetUniverseStargate_CacheMiss_WritesBackToCache(t *testing.T) {
	cache := &fakeStargateCache{ok: false}
	out, err := getUniverseStargate(context.Background(), cache, testUserAgent, 50000001)
	if err != nil {
		t.Fatalf("getUniverseStargate returned error: %v", err)
	}
	if out.Name == "" {
		t.Error("got empty Name on a cache miss, want a live-ESI-fetched name")
	}
	if cache.putCall == nil {
		t.Fatal("PutStargate was not called after a cache miss")
	}
}

type fakeStarCache struct {
	star    sde.Star
	ok      bool
	err     error
	putErr  error
	putCall *sde.Star
}

func (f *fakeStarCache) Star(ctx context.Context, userAgent string, id int64) (sde.Star, bool, error) {
	return f.star, f.ok, f.err
}
func (f *fakeStarCache) PutStar(s sde.Star) error {
	f.putCall = &s
	return f.putErr
}

func TestGetUniverseStar_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeStarCache{star: sde.Star{ID: 40000001, Name: "Tanoo - Star", SpectralClass: "K2 V"}, ok: true}
	out, err := getUniverseStar(context.Background(), cache, testUserAgent, 40000001)
	if err != nil {
		t.Fatalf("getUniverseStar returned error: %v", err)
	}
	if out.Name != "Tanoo - Star" || out.SpectralClass != "K2 V" {
		t.Errorf("got %+v, want Name=\"Tanoo - Star\" SpectralClass=\"K2 V\"", out)
	}
}

func TestGetUniverseStar_CacheMiss_WritesBackToCache(t *testing.T) {
	cache := &fakeStarCache{ok: false}
	out, err := getUniverseStar(context.Background(), cache, testUserAgent, 40000001)
	if err != nil {
		t.Fatalf("getUniverseStar returned error: %v", err)
	}
	if out.Name == "" {
		t.Error("got empty Name on a cache miss, want a live-ESI-fetched name")
	}
	if cache.putCall == nil {
		t.Fatal("PutStar was not called after a cache miss")
	}
}

type fakeStationCache struct {
	station sde.Station
	ok      bool
	err     error
	putErr  error
	putCall *sde.Station
}

func (f *fakeStationCache) Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error) {
	return f.station, f.ok, f.err
}
func (f *fakeStationCache) PutStation(s sde.Station) error {
	f.putCall = &s
	return f.putErr
}

func TestGetUniverseStation_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeStationCache{station: sde.Station{ID: 60000001, Name: "Test Station", SystemID: 30000001}, ok: true}
	out, err := getUniverseStation(context.Background(), cache, testUserAgent, 60000001)
	if err != nil {
		t.Fatalf("getUniverseStation returned error: %v", err)
	}
	if out.Name != "Test Station" || out.SystemID != 30000001 {
		t.Errorf("got %+v, want Name=\"Test Station\" SystemID=30000001", out)
	}
}

func TestGetUniverseStation_CacheMiss_WritesBackToCache(t *testing.T) {
	cache := &fakeStationCache{ok: false}
	out, err := getUniverseStation(context.Background(), cache, testUserAgent, 60000001)
	if err != nil {
		t.Fatalf("getUniverseStation returned error: %v", err)
	}
	if out.Name == "" {
		t.Error("got empty Name on a cache miss, want a live-ESI-fetched name")
	}
	if cache.putCall == nil {
		t.Fatal("PutStation was not called after a cache miss")
	}
}
```
(`testUserAgent` and the `errors`/`sde` imports already exist in `mcp/universe_test.go` from the
prior migration's tests — no new imports needed beyond what every other cache-hit/-miss test pair
in this file already uses.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestGetUniverseAsteroidBelt|TestGetUniverseMoon|TestGetUniversePlanet|TestGetUniverseStargate|TestGetUniverseStar|TestGetUniverseStation' -v 2>&1 | head -30`
Expected: FAIL to compile — `getUniverseAsteroidBelt` (and the other 5) don't yet accept a cache
parameter, `fakeAsteroidBeltCache` etc. don't satisfy any expected interface.

- [ ] **Step 3: Rewrite all 6 handlers to the read-through pattern**

Modify `mcp/universe.go` — replace `mcp/universe.go:893-1115` (the entire block from `--- get_universe_asteroid_belt ---` through the end of `getUniverseStationHandler`) with:
```go
// --- get_universe_asteroid_belt (public) ---

type UniverseAsteroidBelt struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

type asteroidBeltCache interface {
	AsteroidBelt(ctx context.Context, userAgent string, id int64) (sde.AsteroidBelt, bool, error)
	PutAsteroidBelt(sde.AsteroidBelt) error
}

var getUniverseAsteroidBelt = func(ctx context.Context, sdeStore asteroidBeltCache, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
	b, ok, err := sdeStore.AsteroidBelt(ctx, userAgent, asteroidBeltID)
	if err != nil {
		return UniverseAsteroidBelt{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseAsteroidBeltDetail(ctx, asteroidBeltID)
		if err != nil {
			return UniverseAsteroidBelt{}, err
		}
		b = sde.AsteroidBelt{
			ID: asteroidBeltID, Name: detail.Name, SystemID: detail.SystemId,
			Position: sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
		}
		if err := sdeStore.PutAsteroidBelt(b); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through asteroid belt into cache", "error", err)
		}
	}
	return UniverseAsteroidBelt{Name: b.Name, Position: Position{X: b.Position.X, Y: b.Position.Y, Z: b.Position.Z}, SystemID: b.SystemID}, nil
}

type GetUniverseAsteroidBeltInput struct {
	AsteroidBeltID int64 `json:"asteroidBeltId" jsonschema:"the asteroid belt ID to look up"`
}

func getUniverseAsteroidBeltHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseAsteroidBeltInput, UniverseAsteroidBelt] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseAsteroidBeltInput) (*mcpsdk.CallToolResult, UniverseAsteroidBelt, error) {
		out, err := getUniverseAsteroidBelt(ctx, sdeStore, sso.UserAgent(version), in.AsteroidBeltID)
		if err != nil {
			return nil, UniverseAsteroidBelt{}, fmt.Errorf("get universe asteroid belt: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_moon (public) ---

type UniverseMoon struct {
	MoonID   int64    `json:"moonId"`
	SystemID int64    `json:"systemId"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

type moonCache interface {
	Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error)
	PutMoon(sde.Moon) error
}

var getUniverseMoon = func(ctx context.Context, sdeStore moonCache, userAgent string, moonID int64) (UniverseMoon, error) {
	m, ok, err := sdeStore.Moon(ctx, userAgent, moonID)
	if err != nil {
		return UniverseMoon{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseMoonDetail(ctx, moonID)
		if err != nil {
			return UniverseMoon{}, err
		}
		m = sde.Moon{
			ID: detail.MoonId, Name: detail.Name, SystemID: detail.SystemId,
			Position: sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
		}
		if err := sdeStore.PutMoon(m); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through moon into cache", "error", err)
		}
	}
	return UniverseMoon{MoonID: m.ID, SystemID: m.SystemID, Name: m.Name, Position: Position{X: m.Position.X, Y: m.Position.Y, Z: m.Position.Z}}, nil
}

type GetUniverseMoonInput struct {
	MoonID int64 `json:"moonId" jsonschema:"the moon ID to look up"`
}

func getUniverseMoonHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseMoonInput, UniverseMoon] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseMoonInput) (*mcpsdk.CallToolResult, UniverseMoon, error) {
		out, err := getUniverseMoon(ctx, sdeStore, sso.UserAgent(version), in.MoonID)
		if err != nil {
			return nil, UniverseMoon{}, fmt.Errorf("get universe moon: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_planet (public) ---

type UniversePlanet struct {
	PlanetID int64    `json:"planetId"`
	SystemID int64    `json:"systemId"`
	TypeID   int64    `json:"typeId"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

type planetCache interface {
	Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error)
	PutPlanet(sde.Planet) error
}

var getUniversePlanet = func(ctx context.Context, sdeStore planetCache, userAgent string, planetID int64) (UniversePlanet, error) {
	p, ok, err := sdeStore.Planet(ctx, userAgent, planetID)
	if err != nil {
		return UniversePlanet{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniversePlanetDetail(ctx, planetID)
		if err != nil {
			return UniversePlanet{}, err
		}
		p = sde.Planet{
			ID: detail.PlanetId, Name: detail.Name, TypeID: detail.TypeId, SystemID: detail.SystemId,
			Position: sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
		}
		if err := sdeStore.PutPlanet(p); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through planet into cache", "error", err)
		}
	}
	return UniversePlanet{
		PlanetID: p.ID, SystemID: p.SystemID, TypeID: p.TypeID, Name: p.Name,
		Position: Position{X: p.Position.X, Y: p.Position.Y, Z: p.Position.Z},
	}, nil
}

type GetUniversePlanetInput struct {
	PlanetID int64 `json:"planetId" jsonschema:"the planet ID to look up"`
}

func getUniversePlanetHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniversePlanetInput, UniversePlanet] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniversePlanetInput) (*mcpsdk.CallToolResult, UniversePlanet, error) {
		out, err := getUniversePlanet(ctx, sdeStore, sso.UserAgent(version), in.PlanetID)
		if err != nil {
			return nil, UniversePlanet{}, fmt.Errorf("get universe planet: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_stargate (public) ---

type UniverseStargate struct {
	StargateID            int64    `json:"stargateId"`
	SystemID              int64    `json:"systemId"`
	TypeID                int64    `json:"typeId"`
	Name                  string   `json:"name"`
	Position              Position `json:"position"`
	DestinationStargateID int64    `json:"destinationStargateId"`
	DestinationSystemID   int64    `json:"destinationSystemId"`
}

type stargateCache interface {
	Stargate(ctx context.Context, userAgent string, id int64) (sde.Stargate, bool, error)
	PutStargate(sde.Stargate) error
}

var getUniverseStargate = func(ctx context.Context, sdeStore stargateCache, userAgent string, stargateID int64) (UniverseStargate, error) {
	g, ok, err := sdeStore.Stargate(ctx, userAgent, stargateID)
	if err != nil {
		return UniverseStargate{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseStargateDetail(ctx, stargateID)
		if err != nil {
			return UniverseStargate{}, err
		}
		g = sde.Stargate{
			ID: detail.StargateId, Name: detail.Name, TypeID: detail.TypeId, SystemID: detail.SystemId,
			Position:              sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			DestinationStargateID: detail.Destination.StargateId, DestinationSystemID: detail.Destination.SystemId,
		}
		if err := sdeStore.PutStargate(g); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through stargate into cache", "error", err)
		}
	}
	return UniverseStargate{
		StargateID: g.ID, SystemID: g.SystemID, TypeID: g.TypeID, Name: g.Name,
		Position:              Position{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z},
		DestinationStargateID: g.DestinationStargateID, DestinationSystemID: g.DestinationSystemID,
	}, nil
}

type GetUniverseStargateInput struct {
	StargateID int64 `json:"stargateId" jsonschema:"the stargate ID to look up"`
}

func getUniverseStargateHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseStargateInput, UniverseStargate] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStargateInput) (*mcpsdk.CallToolResult, UniverseStargate, error) {
		out, err := getUniverseStargate(ctx, sdeStore, sso.UserAgent(version), in.StargateID)
		if err != nil {
			return nil, UniverseStargate{}, fmt.Errorf("get universe stargate: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_star (public) ---

type UniverseStar struct {
	StarID        int64   `json:"starId"`
	Age           int64   `json:"age"`
	Luminosity    float64 `json:"luminosity"`
	Name          string  `json:"name"`
	Radius        int64   `json:"radius"`
	SolarSystemID int64   `json:"solarSystemId"`
	SpectralClass string  `json:"spectralClass"`
	Temperature   int64   `json:"temperature"`
	TypeID        int64   `json:"typeId"`
}

type starCache interface {
	Star(ctx context.Context, userAgent string, id int64) (sde.Star, bool, error)
	PutStar(sde.Star) error
}

var getUniverseStar = func(ctx context.Context, sdeStore starCache, userAgent string, starID int64) (UniverseStar, error) {
	s, ok, err := sdeStore.Star(ctx, userAgent, starID)
	if err != nil {
		return UniverseStar{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseStarDetail(ctx, starID)
		if err != nil {
			return UniverseStar{}, err
		}
		s = sde.Star{
			ID: starID, Age: detail.Age, Luminosity: detail.Luminosity, Name: detail.Name, Radius: detail.Radius,
			SolarSystemID: detail.SolarSystemId, SpectralClass: detail.SpectralClass, Temperature: detail.Temperature, TypeID: detail.TypeId,
		}
		if err := sdeStore.PutStar(s); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through star into cache", "error", err)
		}
	}
	return UniverseStar{
		StarID: s.ID, Age: s.Age, Luminosity: s.Luminosity, Name: s.Name, Radius: s.Radius,
		SolarSystemID: s.SolarSystemID, SpectralClass: s.SpectralClass, Temperature: s.Temperature, TypeID: s.TypeID,
	}, nil
}

type GetUniverseStarInput struct {
	StarID int64 `json:"starId" jsonschema:"the star ID to look up"`
}

func getUniverseStarHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseStarInput, UniverseStar] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStarInput) (*mcpsdk.CallToolResult, UniverseStar, error) {
		out, err := getUniverseStar(ctx, sdeStore, sso.UserAgent(version), in.StarID)
		if err != nil {
			return nil, UniverseStar{}, fmt.Errorf("get universe star: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_station (public) ---

type UniverseStation struct {
	StationID                int64    `json:"stationId"`
	SystemID                 int64    `json:"systemId"`
	TypeID                   int64    `json:"typeId"`
	Name                     string   `json:"name"`
	Position                 Position `json:"position"`
	MaxDockableShipVolume    float64  `json:"maxDockableShipVolume"`
	OfficeRentalCost         float64  `json:"officeRentalCost"`
	Owner                    *int64   `json:"owner,omitempty"`
	RaceID                   *int64   `json:"raceId,omitempty"`
	ReprocessingEfficiency   float64  `json:"reprocessingEfficiency"`
	ReprocessingStationsTake float64  `json:"reprocessingStationsTake"`
	Services                 []string `json:"services"`
}

type stationCache interface {
	Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error)
	PutStation(sde.Station) error
}

var getUniverseStation = func(ctx context.Context, sdeStore stationCache, userAgent string, stationID int64) (UniverseStation, error) {
	s, ok, err := sdeStore.Station(ctx, userAgent, stationID)
	if err != nil {
		return UniverseStation{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseStationDetail(ctx, stationID)
		if err != nil {
			return UniverseStation{}, err
		}
		s = sde.Station{
			ID: detail.StationId, Name: detail.Name, SystemID: detail.SystemId, TypeID: detail.TypeId,
			Position:              sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			MaxDockableShipVolume: detail.MaxDockableShipVolume, OfficeRentalCost: detail.OfficeRentalCost,
			Owner: detail.Owner, RaceID: detail.RaceId, ReprocessingEfficiency: detail.ReprocessingEfficiency,
			ReprocessingStationsTake: detail.ReprocessingStationsTake, Services: detail.Services,
		}
		if err := sdeStore.PutStation(s); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through station into cache", "error", err)
		}
	}
	return UniverseStation{
		StationID: s.ID, SystemID: s.SystemID, TypeID: s.TypeID, Name: s.Name,
		Position:              Position{X: s.Position.X, Y: s.Position.Y, Z: s.Position.Z},
		MaxDockableShipVolume: s.MaxDockableShipVolume, OfficeRentalCost: s.OfficeRentalCost,
		Owner: s.Owner, RaceID: s.RaceID, ReprocessingEfficiency: s.ReprocessingEfficiency,
		ReprocessingStationsTake: s.ReprocessingStationsTake, Services: s.Services,
	}, nil
}

type GetUniverseStationInput struct {
	StationID int64 `json:"stationId" jsonschema:"the station ID to look up"`
}

func getUniverseStationHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetUniverseStationInput, UniverseStation] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStationInput) (*mcpsdk.CallToolResult, UniverseStation, error) {
		out, err := getUniverseStation(ctx, sdeStore, sso.UserAgent(version), in.StationID)
		if err != nil {
			return nil, UniverseStation{}, fmt.Errorf("get universe station: %w", err)
		}
		return nil, out, nil
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestGetUniverseAsteroidBelt|TestGetUniverseMoon|TestGetUniversePlanet|TestGetUniverseStargate|TestGetUniverseStar|TestGetUniverseStation' -v`
Expected: PASS — all 18 new tests (3 per tool × 6 tools) pass; this will also fail to build until
Step 5 updates the 6 call sites.

- [ ] **Step 5: Update `mcp/server.go`'s 6 call sites**

Modify `mcp/server.go` — find each of the 6 lines constructing these handlers (grep
`getUniverseAsteroidBeltHandler(version)` etc. — each currently passes only `version`) and add
`sdeStore` as the second argument, matching every other migrated Universe handler's call site in
the same file (e.g. `getUniverseRegionHandler(version, sdeStore)`):
```go
getUniverseAsteroidBeltHandler(version, sdeStore)
getUniverseMoonHandler(version, sdeStore)
getUniversePlanetHandler(version, sdeStore)
getUniverseStargateHandler(version, sdeStore)
getUniverseStarHandler(version, sdeStore)
getUniverseStationHandler(version, sdeStore)
```

- [ ] **Step 6: Run full build and test**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./mcp/... ./sde/... -race -v 2>&1 | tail -40`
Expected: clean build/vet/gofmt, all tests pass under `-race`.

- [ ] **Step 7: Commit**

```bash
git add mcp/universe.go mcp/universe_test.go mcp/server.go
git commit -m "feat: migrate 6 celestial-detail Universe tools to sde/ read-through cache"
```

---

### Task 8: `mcp/universe.go` — wire in the 3 remaining augmented fields, update the `CollisionFile` comment

**Files:**
- Modify: `mcp/universe.go`
- Modify: `mcp/universe_test.go`

**Interfaces:**
- Consumes: `sde.Type.PackagedVolume`, `sde.Race.AllianceID`, `sde.System.Planets`/`sde.SystemPlanet`,
  `sde.Graphic.SofDNA` (Tasks 3/4/5).
- Produces: no new exported types — `UniverseType`, `UniverseRace`, `UniverseSystem`/
  `UniverseSystemPlanet`, and `UniverseGraphic`'s existing fields (`PackagedVolume`, `AllianceID`,
  `Planets`, `SofDNA`) now populate on a cache hit too, not only a cache miss. Tool
  names/inputs/outputs are unchanged — this task only removes now-dead "cache miss only" special
  casing and adds one new mapping line each.

This task has 4 independent, small edits — one per tool. Each follows the same shape: remove the
comment explaining why the field was dropped on a cache hit, and map the now-present `sde.*` field
through on both the cache-hit and cache-miss paths.

- [ ] **Step 1: Write the failing tests**

Modify `mcp/universe_test.go` — add:
```go
func TestGetUniverseType_CacheHit_ReturnsPackagedVolume(t *testing.T) {
	vol := 2500.0
	cache := &fakeTypeCache{typ: sde.Type{ID: 587, Name: map[string]string{"en": "Rifter"}, PackagedVolume: &vol}, ok: true}
	out, err := getUniverseType(context.Background(), cache, testUserAgent, 587)
	if err != nil {
		t.Fatalf("getUniverseType returned error: %v", err)
	}
	if out.PackagedVolume == nil || *out.PackagedVolume != 2500 {
		t.Errorf("PackagedVolume = %v, want 2500 (from a cache hit, not only a cache miss)", out.PackagedVolume)
	}
}

func TestListUniverseRaces_CacheHit_ReturnsAllianceID(t *testing.T) {
	allianceID := int64(500001)
	cache := &fakeRaceCache{races: []sde.Race{{ID: 1, Name: map[string]string{"en": "Caldari"}, AllianceID: &allianceID}}}
	out, err := listUniverseRaces(context.Background(), cache, testUserAgent)
	if err != nil {
		t.Fatalf("listUniverseRaces returned error: %v", err)
	}
	if len(out) != 1 || out[0].AllianceID != 500001 {
		t.Errorf("got %+v, want one race with AllianceID=500001 (from a cache hit, not only a cache miss)", out)
	}
}

func TestGetUniverseSystem_CacheHit_ReturnsPlanets(t *testing.T) {
	cache := &fakeSystemCache{
		system: sde.System{
			ID: 30000001, Name: map[string]string{"en": "Tanoo"},
			Planets: []sde.SystemPlanet{{PlanetID: 40000002, MoonIDs: []int64{40000004}, AsteroidBeltIDs: []int64{40000003}}},
		},
		ok: true,
	}
	out, err := getUniverseSystem(context.Background(), cache, testUserAgent, 30000001)
	if err != nil {
		t.Fatalf("getUniverseSystem returned error: %v", err)
	}
	if len(out.Planets) != 1 || out.Planets[0].PlanetID != 40000002 || len(out.Planets[0].MoonIDs) != 1 || len(out.Planets[0].AsteroidBeltIDs) != 1 {
		t.Errorf("Planets = %+v, want one entry PlanetID=40000002 with 1 moon and 1 belt (from a cache hit, not only a cache miss)", out.Planets)
	}
}

func TestGetUniverseGraphic_CacheHit_ReturnsSofDNA(t *testing.T) {
	dna := "cf1_t1:caldaribase:caldari"
	cache := &fakeGraphicCache{graphic: sde.Graphic{ID: 10, SofDNA: &dna}, ok: true}
	out, err := getUniverseGraphic(context.Background(), cache, testUserAgent, 10)
	if err != nil {
		t.Fatalf("getUniverseGraphic returned error: %v", err)
	}
	if out.SofDNA == nil || *out.SofDNA != "cf1_t1:caldaribase:caldari" {
		t.Errorf("SofDNA = %v, want cf1_t1:caldaribase:caldari (from a cache hit, not only a cache miss)", out.SofDNA)
	}
}
```
(`fakeTypeCache`/`fakeRaceCache`/`fakeSystemCache`/`fakeGraphicCache` already exist in
`mcp/universe_test.go` from the prior migration — this task only adds new test functions using
them, no new fakes.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestGetUniverseType_CacheHit_ReturnsPackagedVolume|TestListUniverseRaces_CacheHit_ReturnsAllianceID|TestGetUniverseSystem_CacheHit_ReturnsPlanets|TestGetUniverseGraphic_CacheHit_ReturnsSofDNA' -v`
Expected: FAIL — each assertion currently sees a nil/zero-value field on a cache hit.

- [ ] **Step 3: `get_universe_type` — wire in `PackagedVolume` on a cache hit**

Modify `mcp/universe.go` — in `getUniverseType` (originally `mcp/universe.go:530-588`), replace:
```go
	// PackagedVolume has no SDE source at all (confirmed absent from the
	// full archive, Task 3) -- it's only available fresh off a cache-miss
	// ESI call, never persisted, so it's simply unset on a cache hit.
	var packagedVolume *float64
	if !ok {
```
with:
```go
	if !ok {
```
and replace the final `out := UniverseType{...}` construction's `PackagedVolume: packagedVolume,`
with `PackagedVolume: t.PackagedVolume,` (dropping the now-unused local `packagedVolume` variable
entirely — `t.PackagedVolume` is set on both the cache-hit path, via Task 6's augmented `types`
bucket, and the cache-miss path, since `sde.Type{...}` construction there already needs a
`PackagedVolume: detail.PackagedVolume` field added):
```go
		t = sde.Type{
			ID: detail.TypeId, GroupID: detail.GroupId, Name: map[string]string{"en": detail.Name},
			Description: map[string]string{"en": detail.Description}, Published: detail.Published,
			Capacity: detail.Capacity, GraphicID: detail.GraphicId, IconID: detail.IconId, MarketGroupID: detail.MarketGroupId,
			Mass: detail.Mass, PackagedVolume: detail.PackagedVolume, PortionSize: detail.PortionSize, Radius: detail.Radius, Volume: detail.Volume,
		}
```
(remove the now-orphaned `packagedVolume = detail.PackagedVolume` line that used to set the local
variable).

- [ ] **Step 4: `list_universe_races` — wire in `AllianceID` on a cache hit**

Modify `mcp/universe.go` — in `listUniverseRaces` (originally `mcp/universe.go:154-181`), replace:
```go
	out := make([]UniverseRace, len(races))
	for i, r := range races {
		// AllianceID has no SDE source at all (Task 5 -- confirmed absent
		// from races.jsonl, and there's no alliances.jsonl in the archive)
		// -- only available fresh off the ESI fallback above, dropped (left
		// 0) on this cache-hit path. This is the confirmed, permanent field
		// drop from the design spec's field-drop table.
		out[i] = UniverseRace{RaceID: r.ID, Description: r.Description["en"], Name: r.Name["en"]}
	}
```
with:
```go
	out := make([]UniverseRace, len(races))
	for i, r := range races {
		var allianceID int64
		if r.AllianceID != nil {
			allianceID = *r.AllianceID
		}
		out[i] = UniverseRace{RaceID: r.ID, AllianceID: allianceID, Description: r.Description["en"], Name: r.Name["en"]}
	}
```
(`sde.Race.AllianceID` is `*int64`, Task 3; `UniverseRace.AllianceID` stays `int64` unchanged —
the existing cache-miss branch above already sets it from ESI's atomic response, this only fixes
the cache-hit branch to match).

- [ ] **Step 5: `get_universe_system` — wire in `Planets` on a cache hit**

Modify `mcp/universe.go` — in `getUniverseSystem` (originally `mcp/universe.go:837-877`), replace:
```go
	// Planets has no SDE source (would need a full mapPlanets.jsonl pass for
	// two nested ID arrays -- confirmed not worth it during design) -- only
	// available fresh off a cache-miss ESI call, same treatment as
	// get_universe_type's PackagedVolume in Task 7.
	var planets []UniverseSystemPlanet
	if !ok {
```
with:
```go
	if !ok {
```
and replace the `if len(detail.Planets) > 0 { planets = ...}` block's assignment target from a
local `planets` variable to `sys.Planets` directly (so it's stored in the `sde.System` value that
gets both returned and passed to `PutSystem`, matching how every other migrated cache-miss branch
in this file builds up its `sde.*` struct before writing back):
```go
		if len(detail.Planets) > 0 {
			sys.Planets = make([]sde.SystemPlanet, len(detail.Planets))
			for i, p := range detail.Planets {
				sys.Planets[i] = sde.SystemPlanet{PlanetID: p.PlanetId, AsteroidBeltIDs: p.AsteroidBelts, MoonIDs: p.Moons}
			}
		}
```
then replace the final `return UniverseSystem{...}` construction's `Planets: planets,` with a
conversion from `sys.Planets []sde.SystemPlanet` to `[]UniverseSystemPlanet` (added just before
the `return`):
```go
	var planets []UniverseSystemPlanet
	if len(sys.Planets) > 0 {
		planets = make([]UniverseSystemPlanet, len(sys.Planets))
		for i, p := range sys.Planets {
			planets[i] = UniverseSystemPlanet{PlanetID: p.PlanetID, AsteroidBeltIDs: p.AsteroidBeltIDs, MoonIDs: p.MoonIDs}
		}
	}
	return UniverseSystem{
		SystemID: sys.ID, ConstellationID: sys.ConstellationID, Name: sys.Name["en"],
		Position:       Position{X: sys.Position.X, Y: sys.Position.Y, Z: sys.Position.Z},
		SecurityStatus: sys.SecurityStatus, SecurityClass: sys.SecurityClass, StarID: sys.StarID,
		StargateIDs: sys.StargateIDs, StationIDs: sys.StationIDs, Planets: planets,
	}, nil
```

- [ ] **Step 6: `get_universe_graphic` — wire in `SofDNA` on a cache hit, update the field-drop comment**

Modify `mcp/universe.go` — in `getUniverseGraphic` (originally `mcp/universe.go:1165-1195`),
replace:
```go
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
```
with:
```go
	// CollisionFile has no SDE source and no esi-scrape source either --
	// confirmed absent from all 6051 graphics.jsonl records AND from every
	// record in the esi-scrape mirror of ESI's own live responses (Task 5),
	// so it is not a sourcing gap this migration can close: it's only
	// available fresh off a cache-miss ESI call, and is the one permanent
	// field drop left in the design spec's field-drop table.
	var collisionFile *string
	if !ok {
		client := esi.NewPublicClient(userAgent)
		detail, err := client.UniverseGraphicDetail(ctx, graphicID)
		if err != nil {
			return UniverseGraphic{}, err
		}
		collisionFile = detail.CollisionFile
		g = sde.Graphic{
			ID: detail.GraphicId, GraphicFile: detail.GraphicFile, IconFolder: detail.IconFolder, SofDNA: detail.SofDna,
			SofFactionName: detail.SofFationName, SofHullName: detail.SofHullName, SofRaceName: detail.SofRaceName,
		}
		if err := sdeStore.PutGraphic(g); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through graphic into cache", "error", err)
		}
	}

	return UniverseGraphic{
		GraphicID: g.ID, CollisionFile: collisionFile, GraphicFile: g.GraphicFile, IconFolder: g.IconFolder,
		SofDNA: g.SofDNA, SofFactionName: g.SofFactionName, SofHullName: g.SofHullName, SofRaceName: g.SofRaceName,
	}, nil
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./mcp/... -v 2>&1 | tail -60`
Expected: PASS — every test in `mcp/`, including the 4 new ones from Step 1.

- [ ] **Step 8: Run full hygiene check**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race 2>&1 | tail -40`
Expected: clean build/vet/gofmt, every package's tests pass under `-race`.

- [ ] **Step 9: Commit**

```bash
git add mcp/universe.go mcp/universe_test.go
git commit -m "feat: wire PackagedVolume/AllianceID/Planets/SofDNA through on cache hits too"
```

---

### Task 9: Docs sync — close the SDE-over-ESI exception down to `CollisionFile`

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Interfaces:** None — prose-only, matching the exact shape of the prior plan's own docs-sync task.

- [ ] **Step 1: Update `CLAUDE.md`'s hard rule**

Modify `CLAUDE.md` — replace the "SDE over ESI" bullet (the text the prior migration's Task 10
wrote, quoting its exact previous wording so the diff is unambiguous):
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
with:
```markdown
- **Prefer a cached, statically-refreshed source over a live per-request ESI call**: any tool
  needing a resource available from FC's SDE or from EVE Ref's `reference-data`/`esi-scrape`
  bulk exports must read it from `sde/`, never call ESI for it directly. Performance and
  API-citizenship requirement, not a suggestion. `sde/`'s bbolt cache is bulk-seeded once per
  process lifetime from three sources: FC's own SDE zip (sync-fatal — a failure here fails the
  whole sync), and EVE Ref's `reference-data` and `esi-scrape` datasets (each independently
  best-effort — a fetch or parse failure there logs a warning and skips that source's
  augmentation, never fails the rest of the sync). Every Universe tag tool (all 24 of them) is
  `sde/`-backed via a read-through cache: try `sde/` first, fall back to a live ESI call on a
  cache miss, write the result back so the next lookup of that ID is fast. Exactly one field has
  no source anywhere -- not FC's SDE, not EVE Ref's `esi-scrape` mirror of ESI's own live
  responses either: `get_universe_graphic`'s `CollisionFile`, confirmed absent from every record
  in both. It is populated on a cache miss (fresh off ESI, in case a future record ever has one)
  but dropped on a cache hit — the sole surviving field drop in the design spec's field-drop
  table, documented at its point of implementation in `mcp/universe.go`.
```

- [ ] **Step 2: Update `CLAUDE.md`'s `sde/` architecture-table row**

Modify `CLAUDE.md` — replace the `sde/` row with:
```markdown
| `sde/` | Local cache of FC's Static Data Export plus EVE Ref's `reference-data`/`esi-scrape` bulk exports, backed by `go.etcd.io/bbolt` (`~/.neat/sde/sde.db`). Lazily syncs on first lookup call per `Store` lifetime (at most once per `neat mcp` process run) -- the FC SDE zip fetch is sync-fatal; the two EVE Ref sources are independently best-effort (a failure there logs a warning and skips that source's augmentation, never fails the rest of the sync), since SDE data is enrichment, not a hard requirement. Backs all 24 of the Universe tag's tools via a read-through cache (`mcp/universe.go` orchestrates the ESI fallback and write-back; `sde/` itself never imports `esi/`) -- only `get_universe_graphic`'s `CollisionFile` field remains permanently unsourced, confirmed absent from every dataset checked. |
```

- [ ] **Step 3: Update `README.md`'s matching text**

Modify `README.md` — replace the `sde/` row (`README.md:32`, the prior migration's Task 10 wrote
this exact text) with the same row text as Step 2 above, and replace the line-14 mention:
```markdown
a local SDE cache (`sde/`, backing 18 of the Universe tag's tools via a read-through cache) are
built. See
```
with:
```markdown
a local SDE cache (`sde/`, combining FC's SDE with EVE Ref's `reference-data`/`esi-scrape`
exports and backing all 24 of the Universe tag's tools via a read-through cache) are built. See
```

- [ ] **Step 4: Verify no stale reference remains**

Run: `grep -rn "6 celestial-detail tools\|18 of the Universe tag\|backing 18 of" CLAUDE.md README.md`
Expected: no output — every stale description of the old 18/6 split has been updated by Steps 1–3.

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: close SDE-over-ESI exception to just CollisionFile"
```

---

---
