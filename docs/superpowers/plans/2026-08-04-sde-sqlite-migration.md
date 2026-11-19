# sde/ SQLite Backend Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `sde/`'s bbolt + FC-jsonl-zip + EVE-Ref-reference-data + EVE-Ref-esi-scrape pipeline with a single source — fuzzwork.co.uk's SQLite SDE dump — queried directly, with no change to `sde.Store`'s exported method signatures except `Name`/`Description` fields changing from `map[string]string` to `string`.

**Architecture:** `sde/` opens `~/.neat/sde/eve.db` (downloaded/gunzipped from `https://www.fuzzwork.co.uk/dump/latest-sqlite.db.gz`, ETag-checked like today's EVE Ref fetches) via `database/sql` + `modernc.org/sqlite` (pure Go, keeps `CGO_ENABLED=0`), and every accessor runs a parameterized `SELECT` instead of a bbolt bucket `Get`. A small in-process overlay (`sync.Map`) replaces bbolt as the target of `PutX`'s cache-miss write-back. Three fields with no source in the dump (`PackagedVolume`, `AllianceID`, `SofDNA`) keep coming from trimmed-down EVE Ref fetches, held as in-memory `Store` fields set once per sync.

**Tech Stack:** Go 1.26.1, `database/sql`, `modernc.org/sqlite` (new dependency), `compress/gzip` + `archive/tar` + `github.com/ulikunitz/xz` + `github.com/goccy/go-yaml` (unchanged, for the trimmed EVE Ref fetches). `go.etcd.io/bbolt` is removed entirely.

## A note on task-by-task test execution

Go compiles at the package level: **`sde/` will not build as a whole from partway through Task 1
until Task 8 finishes.** Every task from 1 through 7 rewrites some `sde/*.go` files to the new
`database/sql`-based `Store` shape while leaving others (not yet reached) still calling the old
`go.etcd.io/bbolt` API against that same `Store` — a straightforward compile error if you actually
run `go test ./sde/...` (even scoped with `-run`, since Go must compile the *entire* package
before running any subset of its tests) anywhere in that range.

This is expected, not a planning mistake to route around. Each task's "run the tests" step is
still useful — it's how you confirm that task's *own* new/changed test file and the production
code it exercises are internally consistent (right table/column names, right scan targets, right
zero-value semantics) — but treat a compile failure mentioning a *different* file than the one
this task touches as expected noise, not a signal something in this task is wrong. The first point
a real `go test ./sde/...` run can succeed is Task 8's Step 9, once every resource file has been
migrated. If you want genuine red-green-refactor cycles per task rather than reviewing code by
inspection until then, the practical way to get them is to execute Tasks 1–8 back-to-back in one
sitting before running the suite, rather than pausing for review after each one.

## Global Constraints

- `CGO_ENABLED=0` throughout — the sqlite driver must be pure Go (`modernc.org/sqlite`, not `mattn/go-sqlite3`).
- No environment variables for config/credentials — the dump path is `~/.neat/sde/eve.db`, fixed, same as every other `~/.neat/` path.
- Wrap errors with context: `fmt.Errorf("operation: %w", err)`. Never discard an error (`_ = err`) except where `sde/`'s existing nil-safety convention already does so deliberately (documented inline each time).
- External test packages by default (`package foo_test`) — **exception**: `sde/`'s tests are already `package sde` (internal) because they need `store.syncOnce`/`store.db`; this plan preserves that existing exception, it does not introduce a new one.
- Table-driven tests where a table naturally fits; independent tests otherwise. Standard library only — no `testify`.
- Never model a nested/tree-shaped ESI resource with a self-referential Go struct (unaffected by this plan — no new ESI-shaped types are introduced).
- Every `sde` resource's reference to a fuzzwork table/column name in this plan has been verified directly against the live dump (`sqlite3 ~/.neat/sde/eve.db ".schema <table>"` / `PRAGMA table_info(<table>)`), not assumed from documentation.

---

## Task 1: `sde/store.go` — sqlite-backed `Store` foundation

**Files:**
- Modify: `sde/store.go` (full rewrite)
- Modify: `go.mod`, `go.sum` (add `modernc.org/sqlite`)
- Test: `sde/store_test.go` (new — the old `seedStore`/`putRecord` helpers move here from `sde/category_test.go`, rewritten for sqlite)
- Modify: `sde/category_test.go` (remove the old `seedStore`/`putRecord` definitions — they move to `store_test.go`)

**Interfaces:**
- Produces: `type Store struct{...}` with unexported fields `dbPath`, `metaPath`, `dumpURL`, `referenceDataURL`, `esiScrapeURL`, `syncOnce sync.Once`, `syncErr error`, `db *sql.DB` (nil until `sync()` sets it), `overlay sync.Map`, `typeVolumes map[int64]float64`, `raceAlliances map[int64]int64`, `graphicSofDNA map[int64]string`.
- Produces: `func Open(path string) (*Store, error)`, `func DefaultPath() (string, error)`, `func (s *Store) Close() error` — same signatures as today.
- Produces: `func putOverlay[T any](s *Store, kind string, id int64, v T)`, `func getOverlay[T any](s *Store, kind string, id int64) (T, bool)` — used by every resource file's `PutX`/`GetX` in later tasks.
- Produces: `func nullFloat(v sql.NullFloat64) *float64`, `func nullInt(v sql.NullInt64) *int64` — used by every resource file's scan code in later tasks.
- Produces: `type storeMeta struct{...}`, `func (s *Store) readMeta() (storeMeta, error)`, `func (s *Store) writeMeta(m storeMeta) error` — used by Task 2's `sync.go`.
- Consumes: nothing (this is the foundation task).

- [ ] **Step 1: Add the `modernc.org/sqlite` dependency**

Run: `go get modernc.org/sqlite@latest`

Expected: `go.mod`/`go.sum` gain `modernc.org/sqlite` and its transitive deps. Do not remove `go.etcd.io/bbolt` yet — every other `sde/*.go` file still imports it until Tasks 3–8 replace them.

- [ ] **Step 2: Write the failing test for `Open`/`Close`/meta round-trip**

Create `sde/store_test.go`:

```go
package sde

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// testSchema is the subset of the fuzzwork dump's schema every sde/ accessor
// test needs -- far smaller than the real ~150-table dump, but identical
// column names/types (verified directly against the live dump) for every
// table sde/ actually queries.
const testSchema = `
CREATE TABLE invCategories (categoryID INTEGER PRIMARY KEY, categoryName VARCHAR(100), published BOOLEAN);
CREATE TABLE invGroups (groupID INTEGER PRIMARY KEY, categoryID INTEGER, groupName VARCHAR(100), published BOOLEAN);
CREATE TABLE invTypes (typeID INTEGER PRIMARY KEY, groupID INTEGER, typeName VARCHAR(100), description TEXT, published BOOLEAN, capacity FLOAT, graphicID INTEGER, iconID INTEGER, marketGroupID INTEGER, mass FLOAT, portionSize INTEGER, radius FLOAT, volume FLOAT, raceID INTEGER);
CREATE TABLE dgmTypeAttributes (typeID INTEGER, attributeID INTEGER, valueInt INTEGER, valueFloat FLOAT, PRIMARY KEY (typeID, attributeID));
CREATE TABLE dgmTypeEffects (typeID INTEGER, effectID INTEGER, isDefault BOOLEAN, PRIMARY KEY (typeID, effectID));
CREATE TABLE mapRegions (regionID INTEGER PRIMARY KEY, regionName VARCHAR(100));
CREATE TABLE mapConstellations (constellationID INTEGER PRIMARY KEY, regionID INTEGER, constellationName VARCHAR(100), x FLOAT, y FLOAT, z FLOAT);
CREATE TABLE mapSolarSystems (solarSystemID INTEGER PRIMARY KEY, constellationID INTEGER, solarSystemName VARCHAR(100), x FLOAT, y FLOAT, z FLOAT, security FLOAT, securityClass VARCHAR(2));
CREATE TABLE staStations (stationID INTEGER PRIMARY KEY, stationName VARCHAR(100), solarSystemID INTEGER, stationTypeID INTEGER, x FLOAT, y FLOAT, z FLOAT, maxShipVolumeDockable FLOAT, officeRentalCost FLOAT, corporationID INTEGER, reprocessingEfficiency FLOAT, reprocessingStationsTake FLOAT);
CREATE TABLE mapDenormalize (itemID INTEGER PRIMARY KEY, typeID INTEGER, groupID INTEGER, solarSystemID INTEGER, orbitID INTEGER, x FLOAT, y FLOAT, z FLOAT, itemName VARCHAR(100), celestialIndex INTEGER);
CREATE TABLE mapCelestialStatistics (celestialID INTEGER PRIMARY KEY, temperature FLOAT, spectralClass VARCHAR(10), luminosity FLOAT, age FLOAT, radius FLOAT);
CREATE TABLE mapJumps (stargateID INTEGER PRIMARY KEY, destinationID INTEGER);
CREATE TABLE chrFactions (factionID INTEGER PRIMARY KEY, factionName VARCHAR(100), description VARCHAR(2000), solarSystemID INTEGER, corporationID INTEGER, sizeFactor FLOAT, stationCount INTEGER, stationSystemCount INTEGER, militiaCorporationID INTEGER);
CREATE TABLE chrRaces (raceID INTEGER PRIMARY KEY, raceName VARCHAR(100), description VARCHAR(1000));
CREATE TABLE chrBloodlines (bloodlineID INTEGER PRIMARY KEY, bloodlineName VARCHAR(100), raceID INTEGER, description VARCHAR(1000), shipTypeID INTEGER, corporationID INTEGER, perception INTEGER, willpower INTEGER, charisma INTEGER, memory INTEGER, intelligence INTEGER);
CREATE TABLE chrAncestries (ancestryID INTEGER PRIMARY KEY, ancestryName VARCHAR(100), bloodlineID INTEGER, description VARCHAR(1000), iconID INTEGER, shortDescription VARCHAR(500));
CREATE TABLE eveGraphics (graphicID INTEGER PRIMARY KEY, sofFactionName VARCHAR(100), graphicFile VARCHAR(256), sofHullName VARCHAR(100), sofRaceName VARCHAR(100), iconFolder VARCHAR(256));
`

// seedStore opens a fresh temp-file sqlite Store with testSchema applied and
// marks its sync as already complete (successfully, doing nothing), so
// accessor tests can INSERT fixture rows directly without any network path
// ever running.
func seedStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "eve.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("apply test schema: %v", err)
	}
	store := &Store{dbPath: dbPath, metaPath: filepath.Join(dir, "meta.json"), db: db}
	store.syncOnce.Do(func() {})
	t.Cleanup(func() { store.Close() })
	return store
}

// seedExec runs a fixture INSERT (or other statement) against store's db,
// failing the test immediately on error.
func seedExec(t *testing.T, store *Store, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatalf("exec fixture %q: %v", query, err)
	}
}

func TestOpen_CreatesCacheDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "eve.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	if store.dbPath != path {
		t.Errorf("dbPath = %q, want %q", store.dbPath, path)
	}
	if store.metaPath != filepath.Join(dir, "nested", "meta.json") {
		t.Errorf("metaPath = %q, want sibling meta.json", store.metaPath)
	}
}

func TestStore_Close_NilStore_NoOp(t *testing.T) {
	var store *Store
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil Store returned error: %v", err)
	}
}

func TestStore_Close_NilDB_NoOp(t *testing.T) {
	store := &Store{}
	if err := store.Close(); err != nil {
		t.Errorf("Close with nil db returned error: %v", err)
	}
}

func TestMeta_RoundTrip(t *testing.T) {
	store := &Store{metaPath: filepath.Join(t.TempDir(), "meta.json")}

	want := storeMeta{SchemaVersion: "3", DumpVersion: `"abc123"`}
	if err := store.writeMeta(want); err != nil {
		t.Fatalf("writeMeta returned error: %v", err)
	}

	got, err := store.readMeta()
	if err != nil {
		t.Fatalf("readMeta returned error: %v", err)
	}
	if got != want {
		t.Errorf("readMeta = %+v, want %+v", got, want)
	}
}

func TestMeta_NeverWritten_ReturnsZeroValue(t *testing.T) {
	store := &Store{metaPath: filepath.Join(t.TempDir(), "meta.json")}

	got, err := store.readMeta()
	if err != nil {
		t.Fatalf("readMeta returned error: %v", err)
	}
	if got != (storeMeta{}) {
		t.Errorf("readMeta = %+v, want zero value", got)
	}
}

func TestOverlay_PutThenGet_ReturnsValue(t *testing.T) {
	store := &Store{}
	putOverlay(store, "type", 587, Type{ID: 587, Name: "Rifter"})

	got, ok := getOverlay[Type](store, "type", 587)
	if !ok {
		t.Fatal("getOverlay ok = false, want true")
	}
	if got.Name != "Rifter" {
		t.Errorf("got.Name = %q, want Rifter", got.Name)
	}
}

func TestOverlay_UnknownKey_ReturnsNotOK(t *testing.T) {
	store := &Store{}
	_, ok := getOverlay[Type](store, "type", 999)
	if ok {
		t.Error("getOverlay ok = true, want false for an unseeded key")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./sde/... -run 'TestOpen_|TestStore_Close|TestMeta_|TestOverlay_' -v`
Expected: compile failure — `Store`, `storeMeta`, `putOverlay`, `getOverlay`, `readMeta`, `writeMeta` don't exist yet (the current `sde/store.go` still has the old bbolt-based `Store`).

- [ ] **Step 4: Rewrite `sde/store.go`**

```go
// Package sde caches FC's Static Data Export -- sourced from fuzzwork.co.uk's
// periodically-refreshed SQLite conversion of the SDE, queried directly via
// database/sql -- plus a narrow, in-memory-only augmentation from EVE Ref's
// esi-scrape/reference-data bulk exports for the handful of fields (type
// PackagedVolume, race AllianceID, graphic SofDNA) the fuzzwork dump doesn't
// carry. See docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md
// for the full design and the field-drops this migration accepts.
package sde

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// Store is a local cache of fuzzwork's SQLite SDE dump, queried directly via
// database/sql. Opening a Store does not sync it -- the first lookup call
// does that lazily, at most once per Store's lifetime. A nil *Store is valid
// to call any exported method on: every accessor treats it exactly like a
// never-synced cache (a getter returns its zero value with ok=false, a PutX
// is a silent no-op) so a caller that couldn't open the cache at all can
// still pass sdeStore through unchanged and rely on the existing
// cache-miss/ESI-fallback path.
//
// db starts nil and is only ever assigned inside sync() (see sync.go), after
// any download+atomic-rename of the dump file is fully complete -- opening a
// sqlite connection against dbPath before that point would risk a stale file
// descriptor surviving the rename (a connection opened against the old inode
// keeps reading the old, pre-rename file contents on some platforms). Since
// sync() runs at most once per Store's lifetime (via syncOnce) and every
// accessor calls ensureSynced before touching db, db is guaranteed either
// fully set up or (on a total sync failure, e.g. a never-synced install with
// no network) permanently nil -- every accessor guards for the latter.
type Store struct {
	dbPath   string
	metaPath string

	dumpURL          string
	referenceDataURL string
	esiScrapeURL     string

	syncOnce sync.Once
	syncErr  error

	db *sql.DB

	// overlay holds PutX's cache-miss write-backs, checked by every GetX
	// before falling through to db. Process-lifetime only -- see the design
	// spec's "Write-back: in-memory overlay" section for why persisting it
	// to eve.db itself was rejected.
	overlay sync.Map

	// Set once per sync(), from EVE Ref's reference-data/esi-scrape bulk
	// exports -- the three fields with no source anywhere in the fuzzwork
	// dump. Never persisted to disk (unlike db): every process's single
	// sync() re-fetches both sources fresh, since there is no local copy to
	// otherwise fall back to. nil (not an error) if the corresponding fetch
	// failed or was never attempted (e.g. this Store has never synced).
	typeVolumes   map[int64]float64
	raceAlliances map[int64]int64
	graphicSofDNA map[int64]string
}

// Open opens (creating its parent directory if necessary) a Store rooted at
// path -- conventionally ~/.neat/sde/eve.db, from DefaultPath. It does not
// open a database connection or touch the filesystem beyond MkdirAll; see
// Store's doc comment for why that's deferred to sync().
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create sde cache directory: %w", err)
	}
	return &Store{
		dbPath:           path,
		metaPath:         filepath.Join(filepath.Dir(path), "meta.json"),
		dumpURL:          dumpURL,
		referenceDataURL: referenceDataURL,
		esiScrapeURL:     esiScrapeURL,
	}, nil
}

// DefaultPath returns ~/.neat/sde/eve.db.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".neat", "sde", "eve.db"), nil
}

// Close closes the underlying sqlite connection, if one was ever opened. A
// nil *Store or a Store whose db is still nil (never synced, or synced but
// failed before opening a connection) is a no-op -- see Store's doc comment.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// storeMeta is the small sidecar persisted to metaPath -- bbolt's old _meta
// bucket's replacement. Unlike the old design, only the dump's own version is
// tracked here: reference-data/esi-scrape's augmentation is never persisted
// (see typeVolumes et al.'s doc comment), so there is nothing to gate a skip
// on for those two sources.
type storeMeta struct {
	SchemaVersion string `json:"schemaVersion"`
	DumpVersion   string `json:"dumpVersion"`
}

// readMeta reads the cached meta sidecar, returning a zero-value storeMeta
// (not an error) if it has never been written.
func (s *Store) readMeta() (storeMeta, error) {
	data, err := os.ReadFile(s.metaPath)
	if errors.Is(err, os.ErrNotExist) {
		return storeMeta{}, nil
	}
	if err != nil {
		return storeMeta{}, fmt.Errorf("read sde meta: %w", err)
	}
	var m storeMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return storeMeta{}, fmt.Errorf("parse sde meta: %w", err)
	}
	return m, nil
}

// writeMeta persists m to the meta sidecar.
func (s *Store) writeMeta(m storeMeta) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sde meta: %w", err)
	}
	return os.WriteFile(s.metaPath, data, 0o600)
}

// overlayKey namespaces the overlay sync.Map by resource kind so different
// resources' IDs (e.g. type 587 and region 587) never collide.
func overlayKey(kind string, id int64) string {
	return fmt.Sprintf("%s:%d", kind, id)
}

// putOverlay stores v under (kind, id) in s's overlay. Used by every
// resource file's PutX.
func putOverlay[T any](s *Store, kind string, id int64, v T) {
	s.overlay.Store(overlayKey(kind, id), v)
}

// getOverlay retrieves a value of type T previously stored under (kind, id),
// if any. Used by every resource file's GetX, checked before querying db.
func getOverlay[T any](s *Store, kind string, id int64) (T, bool) {
	v, ok := s.overlay.Load(overlayKey(kind, id))
	if !ok {
		var zero T
		return zero, false
	}
	return v.(T), true
}

// nullFloat converts a scanned nullable FLOAT column to *float64, nil if the
// column was NULL. Shared by every resource file with an optional numeric
// field.
func nullFloat(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}

// nullInt converts a scanned nullable INTEGER column to *int64, nil if the
// column was NULL. Shared by every resource file with an optional ID field.
func nullInt(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	i := v.Int64
	return &i
}
```

- [ ] **Step 5: Remove the old `seedStore`/`putRecord` helpers from `sde/category_test.go`**

Delete these two functions (now superseded by `store_test.go`'s `seedStore`/`seedExec`) from the top of `sde/category_test.go`:

```go
func seedStore(t *testing.T) *Store { ... }
func putRecord(t *testing.T, store *Store, bucket string, id int64, jsonLine string) { ... }
```

Also remove the now-unused `"go.etcd.io/bbolt"` and `"path/filepath"` imports from `sde/category_test.go`'s import block (Task 4 rewrites the rest of that file's tests, which will re-add whatever imports they need).

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./sde/... -run 'TestOpen_|TestStore_Close|TestMeta_|TestOverlay_' -v`
Expected: still a compile failure, and it will stay one through Task 7 — see this plan's "A note on task-by-task test execution" above. Concretely here: `sde/sync.go`, `sde/type.go`, and every other not-yet-rewritten file still call bbolt's `*bbolt.DB` methods (`.View`/`.Update`) against `Store.db`, which this step just changed to `*sql.DB`; `store.go`'s new `Open` also references `dumpURL`, a constant Task 2 hasn't defined yet. Review `store.go`'s diff and this step's test code by inspection for correctness now — the genuine `go test` run happens at Task 8's Step 9.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum sde/store.go sde/store_test.go sde/category_test.go
git commit -m "sde: rewrite Store foundation for sqlite-backed cache

Replaces bbolt with database/sql + modernc.org/sqlite, a meta.json
version sidecar, and an in-memory overlay for cache-miss write-back.
Part of the fuzzwork SQLite dump migration -- see
docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md.

Other sde/*.go files still reference the old bbolt-based accessors and
will not compile until subsequent tasks rewrite them."
```

---

## Task 2: `sde/sync.go` — fetch, download, and trimmed EVE Ref augmentation

**Files:**
- Modify: `sde/sync.go` (full rewrite)
- Test: `sde/sync_test.go` (full rewrite)

**Interfaces:**
- Consumes: `Store` fields/methods from Task 1 (`s.dbPath`, `s.metaPath`, `s.dumpURL`, `s.referenceDataURL`, `s.esiScrapeURL`, `s.readMeta`, `s.writeMeta`, `s.typeVolumes`/`s.raceAlliances`/`s.graphicSofDNA`).
- Produces: `func (s *Store) ensureSynced(ctx context.Context, userAgent string) error` — same signature as today, called by every accessor in Tasks 3–8.
- Produces: package-level `const dumpURL`, `const referenceDataURL`, `const esiScrapeURL` (the latter two keep today's exact values), used by `Open` in Task 1.
- Produces: `func fetchBytes`, `func fetchRemoteVersion` — unchanged helpers from the old `sync.go`, still needed by Task 2 itself and by `fetchReferenceDataTypeVolumes`.

- [ ] **Step 1: Write the failing tests**

Create `sde/sync_test.go`:

```go
package sde

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz"
)

const testUserAgent = "neat-test/0.1"

// buildDumpGzip builds a minimal valid sqlite file (testSchema, no rows) and
// gzips it, standing in for fuzzwork's latest-sqlite.db.gz.
func buildDumpGzip(t *testing.T) []byte {
	t.Helper()
	srcPath := filepath.Join(t.TempDir(), "src.db")
	db, err := sql.Open("sqlite", srcPath)
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("apply schema to source db: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`); err != nil {
		t.Fatalf("seed source db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close source db: %v", err)
	}
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read source db: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		t.Fatalf("gzip source db: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// buildEsiScrapeArchive builds a minimal esi-scrape tar.xz containing only
// the two members sync() still reads (races.en-us.yaml, graphics.yaml).
func buildEsiScrapeArchive(t *testing.T) []byte {
	t.Helper()
	files := map[string]string{
		esiScrapePathPrefix + "races.en-us.yaml": "- race_id: 1\n  alliance_id: 500001\n",
		esiScrapePathPrefix + "graphics.yaml":    "\"2\":\n  sof_dna: \"caldari:caldari:cal\"\n",
	}

	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0o600}); err != nil {
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

// testServers wires up three httptest servers standing in for fuzzwork's
// dump, EVE Ref reference-data, and EVE Ref esi-scrape, and returns a Store
// pointed at them plus each response's ETag.
func testServers(t *testing.T) (store *Store, dumpEtag string) {
	t.Helper()

	dumpBody := buildDumpGzip(t)
	dumpEtag = `"dump-etag-1"`
	dumpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", dumpEtag)
		if r.Method == http.MethodHead {
			return
		}
		w.Write(dumpBody)
	}))
	t.Cleanup(dumpSrv.Close)

	refSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"ref-etag-1"`)
		if r.Method == http.MethodHead {
			return
		}
		// Minimal valid tar.xz with an empty types.json -- exercises the
		// "fetch succeeded, no volumes found" path, not the failure path.
		var tarBuf bytes.Buffer
		tw := tar.NewWriter(&tarBuf)
		content := []byte(`{"34": {"packaged_volume": 0.15}}`)
		tw.WriteHeader(&tar.Header{Name: "types.json", Size: int64(len(content)), Mode: 0o600})
		tw.Write(content)
		tw.Close()
		var xzBuf bytes.Buffer
		xw, _ := xz.NewWriter(&xzBuf)
		xw.Write(tarBuf.Bytes())
		xw.Close()
		w.Write(xzBuf.Bytes())
	}))
	t.Cleanup(refSrv.Close)

	scrapeBody := buildEsiScrapeArchive(t)
	scrapeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"scrape-etag-1"`)
		if r.Method == http.MethodHead {
			return
		}
		w.Write(scrapeBody)
	}))
	t.Cleanup(scrapeSrv.Close)

	dbPath := filepath.Join(t.TempDir(), "eve.db")
	store = &Store{
		dbPath: dbPath, metaPath: filepath.Join(filepath.Dir(dbPath), "meta.json"),
		dumpURL: dumpSrv.URL, referenceDataURL: refSrv.URL, esiScrapeURL: scrapeSrv.URL,
	}
	return store, dumpEtag
}

func TestSync_FirstRun_DownloadsDumpAndAugmentation(t *testing.T) {
	store, dumpEtag := testServers(t)

	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}

	if store.db == nil {
		t.Fatal("sync did not open a db connection")
	}
	var name string
	if err := store.db.QueryRow(`SELECT typeName FROM invTypes WHERE typeID = 587`).Scan(&name); err != nil {
		t.Fatalf("query downloaded db: %v", err)
	}
	if name != "Rifter" {
		t.Errorf("typeName = %q, want Rifter", name)
	}

	if v, ok := store.typeVolumes[34]; !ok || v != 0.15 {
		t.Errorf("typeVolumes[34] = %v, %v, want 0.15, true", v, ok)
	}
	if a, ok := store.raceAlliances[1]; !ok || a != 500001 {
		t.Errorf("raceAlliances[1] = %v, %v, want 500001, true", a, ok)
	}
	if dna, ok := store.graphicSofDNA[2]; !ok || dna != "caldari:caldari:cal" {
		t.Errorf("graphicSofDNA[2] = %v, %v, want caldari:caldari:cal, true", dna, ok)
	}

	meta, err := store.readMeta()
	if err != nil {
		t.Fatalf("readMeta returned error: %v", err)
	}
	if meta.DumpVersion != dumpEtag || meta.SchemaVersion != schemaVersion {
		t.Errorf("meta = %+v, want DumpVersion=%s SchemaVersion=%s", meta, dumpEtag, schemaVersion)
	}
}

func TestSync_UnchangedDump_SkipsRedownload(t *testing.T) {
	store, dumpEtag := testServers(t)
	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	firstInfo, err := os.Stat(store.dbPath)
	if err != nil {
		t.Fatalf("stat db after first sync: %v", err)
	}

	// A second Store pointed at the same dbPath/metaPath, simulating a new
	// process starting up with an already-synced cache on disk.
	second := &Store{
		dbPath: store.dbPath, metaPath: store.metaPath,
		dumpURL: store.dumpURL, referenceDataURL: store.referenceDataURL, esiScrapeURL: store.esiScrapeURL,
	}
	if err := second.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}
	secondInfo, err := os.Stat(second.dbPath)
	if err != nil {
		t.Fatalf("stat db after second sync: %v", err)
	}
	if !firstInfo.ModTime().Equal(secondInfo.ModTime()) {
		t.Error("db file was rewritten on an unchanged-ETag sync, want it left untouched")
	}
	if second.db == nil {
		t.Error("second sync did not open a db connection to the existing file")
	}

	_ = dumpEtag
}

func TestSync_DumpFetchFails_LeavesDBNil(t *testing.T) {
	store := &Store{
		dbPath: filepath.Join(t.TempDir(), "eve.db"), metaPath: filepath.Join(t.TempDir(), "meta.json"),
		dumpURL: "http://127.0.0.1:0/does-not-exist", referenceDataURL: "http://127.0.0.1:0/does-not-exist", esiScrapeURL: "http://127.0.0.1:0/does-not-exist",
	}

	err := store.sync(context.Background(), testUserAgent)
	if err == nil {
		t.Fatal("sync returned nil error, want an error when the dump can't be fetched")
	}
	if store.db != nil {
		t.Error("db was opened despite a failed dump fetch")
	}
}

func TestSync_ReferenceDataFetchFails_DumpStillSucceeds(t *testing.T) {
	store, _ := testServers(t)
	store.referenceDataURL = "http://127.0.0.1:0/does-not-exist"

	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	if store.db == nil {
		t.Error("dump sync did not proceed despite reference-data fetch failure")
	}
	if store.typeVolumes != nil {
		t.Errorf("typeVolumes = %v, want nil after a failed reference-data fetch", store.typeVolumes)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./sde/... -run TestSync_ -v`
Expected: compile failure — `sync`, `schemaVersion`, `esiScrapePathPrefix` don't exist on the new `Store` yet (the current `sde/sync.go` still targets the old bbolt-based design from Task 1's predecessor).

- [ ] **Step 3: Rewrite `sde/sync.go`**

```go
package sde

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/ulikunitz/xz"
)

const dumpURL = "https://www.fuzzwork.co.uk/dump/latest-sqlite.db.gz"
const referenceDataURL = "https://data.everef.net/reference-data/reference-data-latest.tar.xz"
const esiScrapeURL = "https://data.everef.net/esi-scrape/eve-ref-esi-scrape-latest.tar.xz"
const esiScrapePathPrefix = "eve-ref-esi-scrape/data/tranquility/universe/"

// schemaVersion identifies the shape of what sync() writes to dbPath/metaPath
// -- independent of the dump's own ETag. Bumped here from the old bbolt
// design's "2" to "3" to mark the move to a directly-queried sqlite file, so
// an install that synced under the old bbolt schema (a different dbPath,
// eve.db vs the old sde.db, in fact -- but bumping this too is cheap
// insurance and documents the shape change explicitly) forces a fresh
// download rather than trying to open a file that was never written by this
// code.
const schemaVersion = "3"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ensureSynced checks the store's cached dump version against fuzzwork's
// live ETag exactly once per Store lifetime (via s.syncOnce), downloading and
// reopening the database if it's stale or missing. A failed sync is cached
// too -- lookups fall back to whatever's already on disk (nothing, on a
// never-synced install) rather than retrying every call. SDE data is
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
	meta, err := s.readMeta()
	if err != nil {
		return fmt.Errorf("read cached sde meta: %w", err)
	}

	remoteDumpVersion, err := fetchRemoteVersion(ctx, s.dumpURL, userAgent)
	if err != nil {
		return fmt.Errorf("check sde dump version: %w", err)
	}

	if meta.SchemaVersion != schemaVersion || meta.DumpVersion != remoteDumpVersion {
		if err := s.downloadDump(ctx, userAgent); err != nil {
			return fmt.Errorf("download sde dump: %w", err)
		}
		meta.SchemaVersion = schemaVersion
		meta.DumpVersion = remoteDumpVersion
		if err := s.writeMeta(meta); err != nil {
			return fmt.Errorf("write sde meta: %w", err)
		}
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("open sde cache: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping sde cache: %w", err)
	}
	s.db = db

	// reference-data/esi-scrape augmentation is never persisted to disk (see
	// Store's typeVolumes/raceAlliances/graphicSofDNA doc comment) -- every
	// process's single sync() fetches both fresh, best-effort, regardless of
	// whether either has changed remotely, since there is no local copy to
	// otherwise fall back to skipping.
	s.typeVolumes = fetchReferenceDataTypeVolumes(ctx, s.referenceDataURL, userAgent)
	if data, ok := fetchEsiScrapeSupplements(ctx, s.esiScrapeURL, userAgent); ok {
		s.raceAlliances = data.raceAlliances
		s.graphicSofDNA = data.graphicSofDNA
	}
	return nil
}

// downloadDump fetches and gunzips the fuzzwork dump, writing it to a temp
// path and atomically renaming it over s.dbPath -- never leaves a
// partially-written db in place if the process dies mid-download. Always
// called before s.db is ever assigned (sync() only assigns it afterward,
// exactly once, gated by syncOnce), so there is never a live connection
// pointed at the file this rename replaces.
func (s *Store) downloadDump(ctx context.Context, userAgent string) error {
	body, err := fetchBytes(ctx, s.dumpURL, userAgent)
	if err != nil {
		return fmt.Errorf("download dump: %w", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	tmpPath := s.dbPath + ".tmp"
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create temp dump file: %w", err)
	}
	if _, err := io.Copy(tmp, gz); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("decompress dump: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp dump file: %w", err)
	}
	if err := os.Rename(tmpPath, s.dbPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace dump file: %w", err)
	}
	return nil
}

// fetchRemoteVersion does a lightweight HEAD request to check a URL's
// current version (ETag, falling back to Last-Modified) without downloading
// its body.
func fetchRemoteVersion(ctx context.Context, url, userAgent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", fmt.Errorf("build HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute HEAD request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from HEAD %s", resp.StatusCode, url)
	}
	if etag := resp.Header.Get("ETag"); etag != "" {
		return etag, nil
	}
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		return lastModified, nil
	}
	return "", fmt.Errorf("HEAD %s returned neither ETag nor Last-Modified", url)
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

// fetchReferenceDataTypeVolumes fetches EVE Ref's reference-data dataset and
// returns typeID -> packagedVolume for every type that has one. Returns nil
// (not an error) on any failure -- reference-data is a best-effort
// augmentation, never fatal to the rest of sync().
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

// esiScrapeData bundles the two values fetchEsiScrapeSupplements extracts
// from EVE Ref's esi-scrape dataset -- races.en-us.yaml's AllianceID and
// graphics.yaml's SofDNA. Every other resource this dataset used to supply
// (celestial bodies, systemPlanets) is now sourced directly from the fuzzwork
// dump's mapDenormalize/mapJumps/mapCelestialStatistics tables instead (see
// Tasks 6-7).
type esiScrapeData struct {
	raceAlliances map[int64]int64
	graphicSofDNA map[int64]string
}

type esiScrapeRace struct {
	AllianceID int64 `yaml:"alliance_id"`
	RaceID     int64 `yaml:"race_id"`
}

type esiScrapeGraphic struct {
	SofDNA *string `yaml:"sof_dna"`
}

// fetchEsiScrapeSupplements fetches EVE Ref's esi-scrape dataset. ok is true
// if the fetch/extract/parse succeeded overall (an individual missing member
// still leaves its own field nil within data) -- callers use ok to decide
// whether to keep the previous cycle's values or overwrite with (possibly
// nil) new ones.
func fetchEsiScrapeSupplements(ctx context.Context, url, userAgent string) (data esiScrapeData, ok bool) {
	body, err := fetchBytes(ctx, url, userAgent)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to fetch esi-scrape, skipping AllianceID/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}

	want := map[string]bool{
		esiScrapePathPrefix + "races.en-us.yaml": true,
		esiScrapePathPrefix + "graphics.yaml":    true,
	}
	members, err := extractTarXZMembers(body, want)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to extract esi-scrape, skipping AllianceID/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}

	data, err = parseEsiScrapeMembers(ctx, members)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to parse esi-scrape, skipping AllianceID/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}
	return data, true
}

// esiScrapeMemberPresent reports whether key was actually extracted from the
// archive, warning if not.
func esiScrapeMemberPresent(ctx context.Context, members map[string][]byte, key string) bool {
	if _, ok := members[key]; ok {
		return true
	}
	slog.Default().WarnContext(ctx, "sde: esi-scrape archive missing expected member, skipping its augmentation", "member", key)
	return false
}

func parseEsiScrapeMembers(ctx context.Context, members map[string][]byte) (esiScrapeData, error) {
	var data esiScrapeData

	if key := esiScrapePathPrefix + "races.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		var races []esiScrapeRace
		if err := yaml.Unmarshal(members[key], &races); err != nil {
			return esiScrapeData{}, fmt.Errorf("parse races.en-us.yaml: %w", err)
		}
		data.raceAlliances = make(map[int64]int64, len(races))
		for _, r := range races {
			data.raceAlliances[r.RaceID] = r.AllianceID
		}
	}

	if key := esiScrapePathPrefix + "graphics.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		var graphics map[string]esiScrapeGraphic
		if err := yaml.Unmarshal(members[key], &graphics); err != nil {
			return esiScrapeData{}, fmt.Errorf("parse graphics.yaml: %w", err)
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
	}

	return data, nil
}

// extractTarXZMembers decompresses an XZ-compressed tar archive and returns
// the raw bytes of every member whose path is in want, keyed by that same
// path. Members not in want are skipped without being read into memory.
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
```

The blank `_ "modernc.org/sqlite"` driver import only needs to appear once per package (it's already in `store.go` from Task 1) -- `sync.go` only needs the plain `"database/sql"` import shown above.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./sde/... -run TestSync_ -v`
Expected: per this plan's "A note on task-by-task test execution", this will still be a compile failure at this point — `sde/type.go` and every other not-yet-migrated resource file still calls the old bbolt API against the new `Store`. Review `sync.go`'s diff by inspection for now; this task's tests genuinely pass once Task 8 completes and the whole package compiles.

- [ ] **Step 5: Commit**

```bash
git add sde/sync.go sde/sync_test.go
git commit -m "sde: rewrite sync for fuzzwork dump download + trimmed EVE Ref fetch

Replaces the FC-jsonl-zip parse-and-write-to-bbolt pipeline with a
single ETag-gated download+gunzip+atomic-rename of fuzzwork's SQLite
dump. reference-data/esi-scrape shrink to just the three fields with no
home in the dump (type PackagedVolume, race AllianceID, graphic SofDNA),
held in memory rather than persisted."
```

---

## Task 3: `sde/type.go` — `Type`/`PutType`/`TypeIDs` against `invTypes`

**Files:**
- Modify: `sde/type.go` (full rewrite)
- Modify: `sde/type_test.go` (full rewrite)

**Interfaces:**
- Consumes: `putOverlay`/`getOverlay`/`nullFloat`/`nullInt` (Task 1), `ensureSynced` (Task 2), `s.typeVolumes` (Task 2), `seedStore`/`seedExec` (Task 1's `store_test.go`).
- Produces: `type Type struct{ ID, GroupID int64; Name, Description string; Published bool; Capacity, Mass, Radius, Volume, PackagedVolume *float64; GraphicID, IconID, MarketGroupID, PortionSize *int64; DogmaAttributes []TypeDogmaAttribute; DogmaEffects []TypeDogmaEffect }` — `Name`/`Description` are now plain `string`, not `map[string]string`. `TypeDogmaAttribute{AttributeID int64; Value float64}` and `TypeDogmaEffect{EffectID int64; IsDefault bool}` are unchanged from today. Consumed by `mcp/enrich.go` and `mcp/universe.go` in Task 9.
- Produces: `func (s *Store) Type(ctx, userAgent, id) (Type, bool, error)`, `func (s *Store) PutType(Type) error`, `func (s *Store) TypeIDs(ctx, userAgent) ([]int64, error)` — same signatures as today.

- [ ] **Step 1: Write the failing tests**

Replace `sde/type_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestType_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, description, published) VALUES (587, 25, 'Rifter', 'A fast frigate.', 1)`)
	seedExec(t, store, `INSERT INTO dgmTypeAttributes (typeID, attributeID, valueFloat) VALUES (587, 182, 3386.0)`)
	seedExec(t, store, `INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt) VALUES (587, 4, 1)`)
	seedExec(t, store, `INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (587, 4, 1)`)
	store.typeVolumes = map[int64]float64{587: 2.5}

	got, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.ID != 587 || got.GroupID != 25 || !got.Published {
		t.Errorf("got = %+v, want ID=587 GroupID=25 Published=true", got)
	}
	if got.Name != "Rifter" {
		t.Errorf("Name = %q, want Rifter", got.Name)
	}
	if got.Description != "A fast frigate." {
		t.Errorf("Description = %q, want %q", got.Description, "A fast frigate.")
	}
	if got.PackagedVolume == nil || *got.PackagedVolume != 2.5 {
		t.Errorf("PackagedVolume = %v, want 2.5", got.PackagedVolume)
	}
	if len(got.DogmaAttributes) != 2 {
		t.Fatalf("len(DogmaAttributes) = %d, want 2", len(got.DogmaAttributes))
	}
	if len(got.DogmaEffects) != 1 || got.DogmaEffects[0].EffectID != 4 || !got.DogmaEffects[0].IsDefault {
		t.Errorf("DogmaEffects = %+v, want one {EffectID:4 IsDefault:true}", got.DogmaEffects)
	}
}

func TestType_DogmaAttribute_PrefersValueFloatOverValueInt(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`)
	seedExec(t, store, `INSERT INTO dgmTypeAttributes (typeID, attributeID, valueFloat) VALUES (587, 182, 3386.5)`)

	got, _, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].Value != 3386.5 {
		t.Errorf("DogmaAttributes = %+v, want one {AttributeID:182 Value:3386.5}", got.DogmaAttributes)
	}
}

func TestType_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Type(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if ok {
		t.Error("Type ok = true, want false for an unknown ID")
	}
}

func TestType_NilStore_ReturnsNotOK(t *testing.T) {
	var store *Store
	_, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type on nil Store returned error: %v", err)
	}
	if ok {
		t.Error("Type on nil Store ok = true, want false")
	}
}

func TestType_NeverSynced_ReturnsNotOK(t *testing.T) {
	store := &Store{}
	_, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type on never-synced Store returned error: %v", err)
	}
	if ok {
		t.Error("Type on never-synced Store ok = true, want false")
	}
}

func TestPutType_ThenType_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Type{ID: 999, GroupID: 25, Name: "Read-Through Rifter", Published: true}

	if err := store.PutType(want); err != nil {
		t.Fatalf("PutType returned error: %v", err)
	}
	got, ok, err := store.Type(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true after PutType")
	}
	// Type is not comparable with != (it has slice fields DogmaAttributes/
	// DogmaEffects), so this compares the scalar fields the test actually sets.
	if got.ID != want.ID || got.GroupID != want.GroupID || got.Name != want.Name || got.Published != want.Published {
		t.Errorf("Type = %+v, want %+v", got, want)
	}
}

func TestTypeIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (588, 25, 'Slasher', 1)`)

	got, err := store.TypeIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("TypeIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 587 || got[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588]", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./sde/... -run 'TestType|TestPutType|TestTypeIDs' -v`
Expected: compile failure — the current `sde/type.go` still targets bbolt.

- [ ] **Step 3: Rewrite `sde/type.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// Type is a minimal decode of a fuzzwork invTypes row -- the leaf level of
// EVE's item hierarchy (e.g. "Rifter", within the "Frigate" group).
// PackagedVolume has no source in the fuzzwork dump at all -- it's sourced
// entirely from EVE Ref's reference-data dataset at sync time (see sync.go),
// held in-memory rather than a stored column.
type Type struct {
	ID              int64
	GroupID         int64
	Name            string
	Description     string
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

// Type looks up a type by ID. ok is false if the ID isn't in the cache.
func (s *Store) Type(ctx context.Context, userAgent string, id int64) (Type, bool, error) {
	if s == nil {
		return Type{}, false, nil
	}
	if v, ok := getOverlay[Type](s, "type", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Type{}, false, nil
	}

	var (
		groupID                                       int64
		name                                           string
		description                                    sql.NullString
		published                                      bool
		capacity, mass, radius, volume                 sql.NullFloat64
		graphicID, iconID, marketGroupID, portionSize sql.NullInt64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT groupID, typeName, description, published, capacity, graphicID, iconID, marketGroupID, mass, portionSize, radius, volume
		 FROM invTypes WHERE typeID = ?`, id)
	err := row.Scan(&groupID, &name, &description, &published, &capacity, &graphicID, &iconID, &marketGroupID, &mass, &portionSize, &radius, &volume)
	if errors.Is(err, sql.ErrNoRows) {
		return Type{}, false, nil
	}
	if err != nil {
		return Type{}, false, fmt.Errorf("query type %d: %w", id, err)
	}

	out := Type{
		ID: id, GroupID: groupID, Name: name, Description: description.String, Published: published,
		Capacity: nullFloat(capacity), GraphicID: nullInt(graphicID), IconID: nullInt(iconID),
		MarketGroupID: nullInt(marketGroupID), Mass: nullFloat(mass), PortionSize: nullInt(portionSize),
		Radius: nullFloat(radius), Volume: nullFloat(volume),
	}
	if v, ok := s.typeVolumes[id]; ok {
		out.PackagedVolume = &v
	}

	attrRows, err := s.db.QueryContext(ctx, `SELECT attributeID, valueInt, valueFloat FROM dgmTypeAttributes WHERE typeID = ?`, id)
	if err != nil {
		return Type{}, false, fmt.Errorf("query type %d dogma attributes: %w", id, err)
	}
	defer attrRows.Close()
	for attrRows.Next() {
		var attributeID int64
		var valueInt sql.NullInt64
		var valueFloat sql.NullFloat64
		if err := attrRows.Scan(&attributeID, &valueInt, &valueFloat); err != nil {
			return Type{}, false, fmt.Errorf("scan type %d dogma attribute: %w", id, err)
		}
		var v float64
		switch {
		case valueFloat.Valid:
			v = valueFloat.Float64
		case valueInt.Valid:
			v = float64(valueInt.Int64)
		}
		out.DogmaAttributes = append(out.DogmaAttributes, TypeDogmaAttribute{AttributeID: attributeID, Value: v})
	}
	if err := attrRows.Err(); err != nil {
		return Type{}, false, fmt.Errorf("iterate type %d dogma attributes: %w", id, err)
	}

	effectRows, err := s.db.QueryContext(ctx, `SELECT effectID, isDefault FROM dgmTypeEffects WHERE typeID = ?`, id)
	if err != nil {
		return Type{}, false, fmt.Errorf("query type %d dogma effects: %w", id, err)
	}
	defer effectRows.Close()
	for effectRows.Next() {
		var effectID int64
		var isDefault bool
		if err := effectRows.Scan(&effectID, &isDefault); err != nil {
			return Type{}, false, fmt.Errorf("scan type %d dogma effect: %w", id, err)
		}
		out.DogmaEffects = append(out.DogmaEffects, TypeDogmaEffect{EffectID: effectID, IsDefault: isDefault})
	}
	if err := effectRows.Err(); err != nil {
		return Type{}, false, fmt.Errorf("iterate type %d dogma effects: %w", id, err)
	}

	return out, true, nil
}

// PutType stores t in the overlay, keyed by t.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutType(t Type) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "type", t.ID, t)
	return nil
}

// TypeIDs returns every type ID currently cached, ascending.
func (s *Store) TypeIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT typeID FROM invTypes ORDER BY typeID`)
	if err != nil {
		return nil, fmt.Errorf("query type ids: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan type id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./sde/... -run 'TestType|TestPutType|TestTypeIDs' -v`
Expected: still a compile failure per this plan's "A note on task-by-task test execution" — `sde/group.go` and later files haven't migrated yet. Review `type.go`'s diff by inspection; it genuinely passes once Task 8 completes.

- [ ] **Step 5: Commit**

```bash
git add sde/type.go sde/type_test.go
git commit -m "sde: rewrite Type against invTypes/dgmTypeAttributes/dgmTypeEffects

Name/Description become plain strings (every consumer already only read
the en locale). PackagedVolume merges in from the in-memory augmentation
map Task 2's sync() populates. PutType writes to the overlay instead of
bbolt."
```

---

## Task 4: `sde/group.go` + `sde/category.go` — reverse-relation queries against `invGroups`/`invCategories`

**Files:**
- Modify: `sde/group.go`, `sde/category.go` (full rewrites)
- Modify: `sde/category_test.go` (full rewrite of the test bodies left after Task 1 removed `seedStore`/`putRecord`)
- Modify: `sde/store.go` (add the shared `queryInt64Column` helper, Step 4)
- Modify: `sde/type.go` (simplify `TypeIDs` to use `queryInt64Column`, Step 4)
- Test: `sde/group_test.go` (new)

**Interfaces:**
- Consumes: `putOverlay`/`getOverlay` (Task 1), `ensureSynced` (Task 2), `seedStore`/`seedExec` (Task 1).
- Produces: `type Group struct{ ID, CategoryID int64; Name string; Published bool; TypeIDs []int64 }`, `func (s *Store) Group(...) (Group, bool, error)`, `func (s *Store) PutGroup(Group) error`, `func (s *Store) GroupIDs(...) ([]int64, error)`.
- Produces: `type Category struct{ ID int64; Name string; Published bool; GroupIDs []int64 }`, `func (s *Store) Category(...) (Category, bool, error)`, `func (s *Store) PutCategory(Category) error`, `func (s *Store) CategoryIDs(...) ([]int64, error)`.

- [ ] **Step 1: Write the failing tests**

Create `sde/group_test.go`:

```go
package sde

import (
	"context"
	"testing"
)

func TestGroup_KnownID_ReturnsRowWithTypeIDs(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (25, 6, 'Frigate', 1)`)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (588, 25, 'Slasher', 1)`)

	got, ok, err := store.Group(context.Background(), testUserAgent, 25)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false, want true")
	}
	if got.ID != 25 || got.CategoryID != 6 || got.Name != "Frigate" || !got.Published {
		t.Errorf("got = %+v, want ID=25 CategoryID=6 Name=Frigate Published=true", got)
	}
	if len(got.TypeIDs) != 2 || got.TypeIDs[0] != 587 || got.TypeIDs[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588]", got.TypeIDs)
	}
}

func TestGroup_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Group(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if ok {
		t.Error("Group ok = true, want false for an unknown ID")
	}
}

func TestPutGroup_ThenGroup_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Group{ID: 999, CategoryID: 6, Name: "Read-Through Group", Published: true, TypeIDs: []int64{1, 2}}

	if err := store.PutGroup(want); err != nil {
		t.Fatalf("PutGroup returned error: %v", err)
	}
	got, ok, err := store.Group(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false, want true after PutGroup")
	}
	if got.Name != want.Name || len(got.TypeIDs) != 2 {
		t.Errorf("Group = %+v, want %+v", got, want)
	}
}

func TestGroupIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (25, 6, 'Frigate', 1)`)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (26, 6, 'Cruiser', 1)`)

	got, err := store.GroupIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("GroupIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 25 || got[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26]", got)
	}
}
```

Replace `sde/category_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestCategory_KnownID_ReturnsRowWithGroupIDs(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invCategories (categoryID, categoryName, published) VALUES (6, 'Ship', 1)`)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (25, 6, 'Frigate', 1)`)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (26, 6, 'Cruiser', 1)`)

	got, ok, err := store.Category(context.Background(), testUserAgent, 6)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false, want true")
	}
	if got.ID != 6 || got.Name != "Ship" || !got.Published {
		t.Errorf("got = %+v, want ID=6 Name=Ship Published=true", got)
	}
	if len(got.GroupIDs) != 2 || got.GroupIDs[0] != 25 || got.GroupIDs[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26]", got.GroupIDs)
	}
}

func TestCategory_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Category(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if ok {
		t.Error("Category ok = true, want false for an unknown ID")
	}
}

func TestPutCategory_ThenCategory_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Category{ID: 999, Name: "Read-Through Category", Published: true, GroupIDs: []int64{1, 2}}

	if err := store.PutCategory(want); err != nil {
		t.Fatalf("PutCategory returned error: %v", err)
	}
	got, ok, err := store.Category(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false, want true after PutCategory")
	}
	if got.Name != want.Name || len(got.GroupIDs) != 2 {
		t.Errorf("Category = %+v, want %+v", got, want)
	}
}

func TestCategoryIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invCategories (categoryID, categoryName, published) VALUES (6, 'Ship', 1)`)
	seedExec(t, store, `INSERT INTO invCategories (categoryID, categoryName, published) VALUES (7, 'Module', 1)`)

	got, err := store.CategoryIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("CategoryIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 6 || got[1] != 7 {
		t.Errorf("CategoryIDs = %v, want [6 7]", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./sde/... -run 'TestGroup|TestPutGroup|TestGroupIDs|TestCategory|TestPutCategory|TestCategoryIDs' -v`
Expected: compile failure — `sde/group.go`/`sde/category.go` still target bbolt.

- [ ] **Step 3: Rewrite `sde/group.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Group is a minimal decode of a fuzzwork invGroups row -- EVE's item
// hierarchy level between Category and Type (e.g. "Frigate", within the
// "Ship" category). TypeIDs is not a stored column -- it's computed via a
// reverse query over invTypes at read time.
type Group struct {
	ID         int64
	CategoryID int64
	Name       string
	Published  bool
	TypeIDs    []int64
}

// Group looks up a group by ID. ok is false if the ID isn't in the cache.
func (s *Store) Group(ctx context.Context, userAgent string, id int64) (Group, bool, error) {
	if s == nil {
		return Group{}, false, nil
	}
	if v, ok := getOverlay[Group](s, "group", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Group{}, false, nil
	}

	var (
		categoryID int64
		name       string
		published  bool
	)
	row := s.db.QueryRowContext(ctx, `SELECT categoryID, groupName, published FROM invGroups WHERE groupID = ?`, id)
	if err := row.Scan(&categoryID, &name, &published); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Group{}, false, nil
		}
		return Group{}, false, fmt.Errorf("query group %d: %w", id, err)
	}

	typeIDs, err := s.queryInt64Column(ctx, `SELECT typeID FROM invTypes WHERE groupID = ? ORDER BY typeID`, id)
	if err != nil {
		return Group{}, false, fmt.Errorf("query group %d type ids: %w", id, err)
	}

	return Group{ID: id, CategoryID: categoryID, Name: name, Published: published, TypeIDs: typeIDs}, true, nil
}

// PutGroup stores g in the overlay, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGroup(g Group) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "group", g.ID, g)
	return nil
}

// GroupIDs returns every group ID currently cached, ascending.
func (s *Store) GroupIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	ids, err := s.queryInt64Column(ctx, `SELECT groupID FROM invGroups ORDER BY groupID`)
	if err != nil {
		return nil, fmt.Errorf("query group ids: %w", err)
	}
	return ids, nil
}
```

- [ ] **Step 4: Add the shared `queryInt64Column` helper to `sde/store.go`**

Every reverse-relation query in this and every subsequent task (`Group.TypeIDs`, `Category.GroupIDs`, `Region.ConstellationIDs`, `Constellation.SystemIDs`, `System.StationIDs`/`StargateIDs`, ...) has the identical shape: run a query that returns one `int64` column, collect it into a slice. Add this once to `sde/store.go` (append after `nullInt`):

```go
// queryInt64Column runs query (which must select exactly one int64 column)
// and returns every row's value as a slice, in the order the query returned
// them. Shared by every resource file's reverse-relation lookups (a group's
// TypeIDs, a system's StationIDs, ...).
func (s *Store) queryInt64Column(ctx context.Context, query string, args ...any) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

Add `"context"` to `sde/store.go`'s import block if not already present from a prior task (it is not, as of Task 1 -- add it now).

Also simplify Task 3's `sde/type.go` `TypeIDs` to use this helper instead of its own inline loop -- replace its body with:

```go
func (s *Store) TypeIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT typeID FROM invTypes ORDER BY typeID`)
}
```

- [ ] **Step 5: Rewrite `sde/category.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Category is a minimal decode of a fuzzwork invCategories row -- the top
// level of EVE's item hierarchy (e.g. "Ship", "Module"). GroupIDs is not a
// stored column -- it's computed via a reverse query over invGroups at read
// time.
type Category struct {
	ID        int64
	Name      string
	Published bool
	GroupIDs  []int64
}

// Category looks up a category by ID. ok is false if the ID isn't in the
// cache -- either because the store has never synced successfully, or
// because no such category exists.
func (s *Store) Category(ctx context.Context, userAgent string, id int64) (Category, bool, error) {
	if s == nil {
		return Category{}, false, nil
	}
	if v, ok := getOverlay[Category](s, "category", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Category{}, false, nil
	}

	var (
		name      string
		published bool
	)
	row := s.db.QueryRowContext(ctx, `SELECT categoryName, published FROM invCategories WHERE categoryID = ?`, id)
	if err := row.Scan(&name, &published); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, false, nil
		}
		return Category{}, false, fmt.Errorf("query category %d: %w", id, err)
	}

	groupIDs, err := s.queryInt64Column(ctx, `SELECT groupID FROM invGroups WHERE categoryID = ? ORDER BY groupID`, id)
	if err != nil {
		return Category{}, false, fmt.Errorf("query category %d group ids: %w", id, err)
	}

	return Category{ID: id, Name: name, Published: published, GroupIDs: groupIDs}, true, nil
}

// PutCategory stores c in the overlay, keyed by c.ID -- used by mcp/'s
// read-through fallback to cache a category fetched live from ESI.
func (s *Store) PutCategory(c Category) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "category", c.ID, c)
	return nil
}

// CategoryIDs returns every category ID currently cached, ascending. Returns
// an empty slice, not an error, if the store has never synced successfully;
// callers read that as "cache empty" and fall back to a live ESI list call,
// not "no categories exist".
func (s *Store) CategoryIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT categoryID FROM invCategories ORDER BY categoryID`)
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./sde/... -run 'TestGroup|TestPutGroup|TestGroupIDs|TestCategory|TestPutCategory|TestCategoryIDs|TestType|TestTypeIDs' -v`
Expected: still a compile failure per this plan's "A note on task-by-task test execution" — `sde/region.go` and later files haven't migrated yet. Review `group.go`/`category.go`'s diffs by inspection; they genuinely pass (along with Task 3's re-run `Type`/`TypeIDs` tests, since Step 4 changed `TypeIDs`'s body) once Task 8 completes.

- [ ] **Step 7: Commit**

```bash
git add sde/store.go sde/group.go sde/group_test.go sde/category.go sde/category_test.go sde/type.go
git commit -m "sde: rewrite Group/Category against invGroups/invCategories

TypeIDs/GroupIDs move from sync-time precomputation to read-time
reverse queries via the new shared queryInt64Column helper, which
Type.TypeIDs also adopts."
```

---

## Task 5: `sde/region.go` + `sde/constellation.go` — `mapRegions`/`mapConstellations`

**Files:**
- Modify: `sde/region.go`, `sde/constellation.go` (full rewrites)
- Test: `sde/region_test.go`, `sde/constellation_test.go` (full rewrites)

**Interfaces:**
- Consumes: `putOverlay`/`getOverlay`/`queryInt64Column` (Tasks 1/4), `ensureSynced` (Task 2), `seedStore`/`seedExec` (Task 1).
- Produces: `type Position struct{ X, Y, Z float64 }` (unchanged shape, defined in `constellation.go` as today).
- Produces: `type Region struct{ ID int64; Name string; Description *string; ConstellationIDs []int64 }` — **`Description` is `*string`, not plain `string`**, per the design spec's documented exception (`mapRegions` has no description column; a fresh ESI cache-miss can still populate it via the overlay). `func (s *Store) Region(...) (Region, bool, error)`, `func (s *Store) PutRegion(Region) error`, `func (s *Store) RegionIDs(...) ([]int64, error)`.
- Produces: `type Constellation struct{ ID, RegionID int64; Name string; Position Position; SystemIDs []int64 }`, `func (s *Store) Constellation(...) (Constellation, bool, error)`, `func (s *Store) PutConstellation(Constellation) error`, `func (s *Store) ConstellationIDs(...) ([]int64, error)`.

- [ ] **Step 1: Write the failing tests**

Replace `sde/region_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestRegion_KnownID_ReturnsRowWithNilDescription(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapRegions (regionID, regionName) VALUES (10000001, 'Derelik')`)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000001, 10000001, 'San Matar')`)

	got, ok, err := store.Region(context.Background(), testUserAgent, 10000001)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true")
	}
	if got.ID != 10000001 || got.Name != "Derelik" {
		t.Errorf("got = %+v, want ID=10000001 Name=Derelik", got)
	}
	if got.Description != nil {
		t.Errorf("Description = %v, want nil (mapRegions has no description column)", *got.Description)
	}
	if len(got.ConstellationIDs) != 1 || got.ConstellationIDs[0] != 20000001 {
		t.Errorf("ConstellationIDs = %v, want [20000001]", got.ConstellationIDs)
	}
}

func TestRegion_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Region(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if ok {
		t.Error("Region ok = true, want false for an unknown ID")
	}
}

func TestPutRegion_ThenRegion_ReturnsOverlaidDescription(t *testing.T) {
	store := seedStore(t)
	description := "A dangerous frontier region."
	want := Region{ID: 999, Name: "Read-Through Region", Description: &description, ConstellationIDs: []int64{1}}

	if err := store.PutRegion(want); err != nil {
		t.Fatalf("PutRegion returned error: %v", err)
	}
	got, ok, err := store.Region(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true after PutRegion")
	}
	if got.Description == nil || *got.Description != description {
		t.Errorf("Description = %v, want %q (overlay values keep whatever PutRegion was given)", got.Description, description)
	}
}

func TestRegionIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapRegions (regionID, regionName) VALUES (10000001, 'Derelik')`)
	seedExec(t, store, `INSERT INTO mapRegions (regionID, regionName) VALUES (10000002, 'The Forge')`)

	got, err := store.RegionIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("RegionIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 10000001 || got[1] != 10000002 {
		t.Errorf("RegionIDs = %v, want [10000001 10000002]", got)
	}
}
```

Replace `sde/constellation_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestConstellation_KnownID_ReturnsRowWithSystemIDs(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName, x, y, z) VALUES (20000001, 10000001, 'San Matar', 1.0, 2.0, 3.0)`)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000001, 20000001, 'Tanoo')`)

	got, ok, err := store.Constellation(context.Background(), testUserAgent, 20000001)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false, want true")
	}
	if got.ID != 20000001 || got.RegionID != 10000001 || got.Name != "San Matar" {
		t.Errorf("got = %+v, want ID=20000001 RegionID=10000001 Name=San Matar", got)
	}
	if got.Position != (Position{X: 1.0, Y: 2.0, Z: 3.0}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
	if len(got.SystemIDs) != 1 || got.SystemIDs[0] != 30000001 {
		t.Errorf("SystemIDs = %v, want [30000001]", got.SystemIDs)
	}
}

func TestConstellation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Constellation(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if ok {
		t.Error("Constellation ok = true, want false for an unknown ID")
	}
}

func TestPutConstellation_ThenConstellation_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Constellation{ID: 999, RegionID: 10000001, Name: "Read-Through Constellation", SystemIDs: []int64{1}}

	if err := store.PutConstellation(want); err != nil {
		t.Fatalf("PutConstellation returned error: %v", err)
	}
	got, ok, err := store.Constellation(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false, want true after PutConstellation")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestConstellationIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000001, 10000001, 'San Matar')`)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000002, 10000001, 'Second')`)

	got, err := store.ConstellationIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("ConstellationIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 20000001 || got[1] != 20000002 {
		t.Errorf("ConstellationIDs = %v, want [20000001 20000002]", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./sde/... -run 'TestRegion|TestPutRegion|TestRegionIDs|TestConstellation|TestPutConstellation|TestConstellationIDs' -v`
Expected: compile failure.

- [ ] **Step 3: Rewrite `sde/region.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Region is a minimal decode of a fuzzwork mapRegions row -- the top level of
// EVE's geography (e.g. "Derelik"). Description is *string, not string: the
// fuzzwork dump's mapRegions table has no description column at all (checked
// directly -- regionID/regionName/x/y/z/xMin.../factionID/nebula/radius,
// nothing else), so every sqlite-served Region has a nil Description. A
// cache-miss ESI fetch's response still includes it, though (ESI's own
// schema carries it), so the overlay can still hold a populated value for
// that process's lifetime -- see docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md's
// "Exception: Region.Description stays *string" section.
type Region struct {
	ID               int64
	Name             string
	Description      *string
	ConstellationIDs []int64
}

// Region looks up a region by ID. ok is false if the ID isn't in the cache.
func (s *Store) Region(ctx context.Context, userAgent string, id int64) (Region, bool, error) {
	if s == nil {
		return Region{}, false, nil
	}
	if v, ok := getOverlay[Region](s, "region", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Region{}, false, nil
	}

	var name string
	row := s.db.QueryRowContext(ctx, `SELECT regionName FROM mapRegions WHERE regionID = ?`, id)
	if err := row.Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Region{}, false, nil
		}
		return Region{}, false, fmt.Errorf("query region %d: %w", id, err)
	}

	constellationIDs, err := s.queryInt64Column(ctx, `SELECT constellationID FROM mapConstellations WHERE regionID = ? ORDER BY constellationID`, id)
	if err != nil {
		return Region{}, false, fmt.Errorf("query region %d constellation ids: %w", id, err)
	}

	return Region{ID: id, Name: name, ConstellationIDs: constellationIDs}, true, nil
}

// PutRegion stores r in the overlay, keyed by r.ID -- used by mcp/'s
// read-through fallback to cache a region fetched live from ESI. Unlike a
// sqlite-served Region, an overlaid one may carry a non-nil Description --
// see Region's doc comment.
func (s *Store) PutRegion(r Region) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "region", r.ID, r)
	return nil
}

// RegionIDs returns every region ID currently cached, ascending.
func (s *Store) RegionIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT regionID FROM mapRegions ORDER BY regionID`)
}
```

- [ ] **Step 4: Rewrite `sde/constellation.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Position is a shared 3D coordinate, reused by every geography resource.
type Position struct {
	X float64
	Y float64
	Z float64
}

// Constellation is a minimal decode of a fuzzwork mapConstellations row --
// EVE's geography level between Region and System. SystemIDs is not a
// stored column -- it's computed via a reverse query over mapSolarSystems at
// read time.
type Constellation struct {
	ID        int64
	RegionID  int64
	Name      string
	Position  Position
	SystemIDs []int64
}

// Constellation looks up a constellation by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) Constellation(ctx context.Context, userAgent string, id int64) (Constellation, bool, error) {
	if s == nil {
		return Constellation{}, false, nil
	}
	if v, ok := getOverlay[Constellation](s, "constellation", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Constellation{}, false, nil
	}

	var (
		regionID int64
		name     string
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx, `SELECT regionID, constellationName, x, y, z FROM mapConstellations WHERE constellationID = ?`, id)
	if err := row.Scan(&regionID, &name, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Constellation{}, false, nil
		}
		return Constellation{}, false, fmt.Errorf("query constellation %d: %w", id, err)
	}

	systemIDs, err := s.queryInt64Column(ctx, `SELECT solarSystemID FROM mapSolarSystems WHERE constellationID = ? ORDER BY solarSystemID`, id)
	if err != nil {
		return Constellation{}, false, fmt.Errorf("query constellation %d system ids: %w", id, err)
	}

	return Constellation{
		ID: id, RegionID: regionID, Name: name,
		Position:  Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
		SystemIDs: systemIDs,
	}, true, nil
}

// PutConstellation stores c in the overlay, keyed by c.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutConstellation(c Constellation) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "constellation", c.ID, c)
	return nil
}

// ConstellationIDs returns every constellation ID currently cached, ascending.
func (s *Store) ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT constellationID FROM mapConstellations ORDER BY constellationID`)
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./sde/... -run 'TestRegion|TestPutRegion|TestRegionIDs|TestConstellation|TestPutConstellation|TestConstellationIDs' -v`
Expected: still a compile failure per this plan's "A note on task-by-task test execution" — `sde/system.go` and later files haven't migrated yet. Review `region.go`/`constellation.go`'s diffs by inspection; they genuinely pass once Task 8 completes.

- [ ] **Step 6: Commit**

```bash
git add sde/region.go sde/region_test.go sde/constellation.go sde/constellation_test.go
git commit -m "sde: rewrite Region/Constellation against mapRegions/mapConstellations

Region.Description becomes *string (was map[string]string) and is
always nil from a sqlite-served lookup -- mapRegions has no description
column in the fuzzwork dump, a documented regression from the bbolt
design. A cache-miss overlay write can still carry one."
```

---

## Task 6: `sde/system.go` + `sde/station.go` — `mapSolarSystems`/`staStations`/`mapDenormalize`/`mapJumps`

**Files:**
- Modify: `sde/system.go`, `sde/station.go` (full rewrites)
- Test: `sde/system_test.go`, `sde/station_test.go` (full rewrites)

**Interfaces:**
- Consumes: `putOverlay`/`getOverlay`/`queryInt64Column`/`nullInt` (Tasks 1/4), `Position` (Task 5), `ensureSynced` (Task 2), `seedStore`/`seedExec` (Task 1).
- Produces: `type SystemPlanet struct{ PlanetID int64; AsteroidBeltIDs, MoonIDs []int64 }`, `type System struct{ ID, ConstellationID int64; Name string; Position Position; SecurityStatus float64; SecurityClass *string; StarID *int64; StargateIDs, StationIDs []int64; Planets []SystemPlanet }`, `func (s *Store) System(...) (System, bool, error)`, `func (s *Store) PutSystem(System) error`, `func (s *Store) SystemIDs(...) ([]int64, error)`.
- Produces: `type Station struct{ ID int64; Name string; SystemID, TypeID int64; Position Position; MaxDockableShipVolume, OfficeRentalCost, ReprocessingEfficiency, ReprocessingStationsTake float64; Owner, RaceID *int64; Services []string }`, `func (s *Store) Station(...) (Station, bool, error)`, `func (s *Store) PutStation(Station) error`.

Note: `Station.Services` is always an empty slice from a sqlite-served lookup (`staOperationServices` is empty in the dump, per the design spec's field-drop audit) — a cache-miss overlay write can still carry a populated one, same pattern as `Region.Description`.

- [ ] **Step 1: Write the failing tests**

Replace `sde/system_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestSystem_KnownID_ReturnsRowWithReverseRelations(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName, x, y, z, security, securityClass) VALUES (30000142, 20000020, 'Jita', 1.0, 2.0, 3.0, 0.9, 'B')`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009076, 3796, 6, 30000142, NULL, 'Jita - Sun', 0)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009077, 2016, 7, 30000142, 40009076, 'Jita 1', 1)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009078, 14, 8, 30000142, 40009077, 'Jita 1 - Moon 1', NULL)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009079, 16, 9, 30000142, 40009077, 'Jita 1 - Asteroid Belt 1', NULL)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (50008077, 16, 10, 30000142, NULL, 'Jita - Perimeter', NULL)`)
	seedExec(t, store, `INSERT INTO staStations (stationID, stationName, solarSystemID, stationTypeID) VALUES (60003760, 'Jita IV - Moon 4', 30000142, 1531)`)

	got, ok, err := store.System(context.Background(), testUserAgent, 30000142)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.ID != 30000142 || got.ConstellationID != 20000020 || got.Name != "Jita" {
		t.Errorf("got = %+v, want ID=30000142 ConstellationID=20000020 Name=Jita", got)
	}
	if got.SecurityStatus != 0.9 || got.SecurityClass == nil || *got.SecurityClass != "B" {
		t.Errorf("SecurityStatus/SecurityClass = %v/%v, want 0.9/B", got.SecurityStatus, got.SecurityClass)
	}
	if got.StarID == nil || *got.StarID != 40009076 {
		t.Errorf("StarID = %v, want 40009076", got.StarID)
	}
	if len(got.StargateIDs) != 1 || got.StargateIDs[0] != 50008077 {
		t.Errorf("StargateIDs = %v, want [50008077]", got.StargateIDs)
	}
	if len(got.StationIDs) != 1 || got.StationIDs[0] != 60003760 {
		t.Errorf("StationIDs = %v, want [60003760]", got.StationIDs)
	}
	if len(got.Planets) != 1 {
		t.Fatalf("len(Planets) = %d, want 1", len(got.Planets))
	}
	p := got.Planets[0]
	if p.PlanetID != 40009077 {
		t.Errorf("Planets[0].PlanetID = %d, want 40009077", p.PlanetID)
	}
	if len(p.MoonIDs) != 1 || p.MoonIDs[0] != 40009078 {
		t.Errorf("Planets[0].MoonIDs = %v, want [40009078]", p.MoonIDs)
	}
	if len(p.AsteroidBeltIDs) != 1 || p.AsteroidBeltIDs[0] != 40009079 {
		t.Errorf("Planets[0].AsteroidBeltIDs = %v, want [40009079]", p.AsteroidBeltIDs)
	}
}

func TestSystem_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.System(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if ok {
		t.Error("System ok = true, want false for an unknown ID")
	}
}

func TestSystem_NoStar_ReturnsNilStarID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000001, 20000001, 'Tanoo')`)

	got, ok, err := store.System(context.Background(), testUserAgent, 30000001)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.StarID != nil {
		t.Errorf("StarID = %v, want nil", *got.StarID)
	}
}

func TestPutSystem_ThenSystem_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := System{ID: 999, ConstellationID: 20000001, Name: "Read-Through System"}

	if err := store.PutSystem(want); err != nil {
		t.Fatalf("PutSystem returned error: %v", err)
	}
	got, ok, err := store.System(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true after PutSystem")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestSystemIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000001, 20000001, 'Tanoo')`)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000142, 20000020, 'Jita')`)

	got, err := store.SystemIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("SystemIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 30000001 || got[1] != 30000142 {
		t.Errorf("SystemIDs = %v, want [30000001 30000142]", got)
	}
}
```

Replace `sde/station_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestStation_KnownID_ReturnsRowWithJoinedRaceID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published, raceID) VALUES (1531, 15, 'Caldari Trading Post', 1, 1)`)
	seedExec(t, store, `INSERT INTO staStations (stationID, stationName, solarSystemID, stationTypeID, x, y, z, maxShipVolumeDockable, officeRentalCost, corporationID, reprocessingEfficiency, reprocessingStationsTake)
		VALUES (60003760, 'Jita IV - Moon 4', 30000142, 1531, 1.0, 2.0, 3.0, 500000000.0, 800000.0, 1000035, 0.5, 0.05)`)

	got, ok, err := store.Station(context.Background(), testUserAgent, 60003760)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false, want true")
	}
	if got.ID != 60003760 || got.Name != "Jita IV - Moon 4" || got.SystemID != 30000142 || got.TypeID != 1531 {
		t.Errorf("got = %+v, want ID=60003760 Name=%q SystemID=30000142 TypeID=1531", got, "Jita IV - Moon 4")
	}
	if got.Owner == nil || *got.Owner != 1000035 {
		t.Errorf("Owner = %v, want 1000035", got.Owner)
	}
	if got.RaceID == nil || *got.RaceID != 1 {
		t.Errorf("RaceID = %v, want 1 (joined from invTypes.raceID)", got.RaceID)
	}
	if len(got.Services) != 0 {
		t.Errorf("Services = %v, want empty (staOperationServices has no rows in the fuzzwork dump)", got.Services)
	}
}

func TestStation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Station(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if ok {
		t.Error("Station ok = true, want false for an unknown ID")
	}
}

func TestPutStation_ThenStation_ReturnsOverlaidServices(t *testing.T) {
	store := seedStore(t)
	want := Station{ID: 999, Name: "Read-Through Station", Services: []string{"market", "reprocessing-plant"}}

	if err := store.PutStation(want); err != nil {
		t.Fatalf("PutStation returned error: %v", err)
	}
	got, ok, err := store.Station(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false, want true after PutStation")
	}
	if len(got.Services) != 2 {
		t.Errorf("Services = %v, want [market reprocessing-plant] (overlay values keep whatever PutStation was given)", got.Services)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./sde/... -run 'TestSystem|TestPutSystem|TestSystemIDs|TestStation|TestPutStation' -v`
Expected: compile failure.

- [ ] **Step 3: Rewrite `sde/system.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SystemPlanet is a minimal decode of one planet entry within a solar system
// record, mirroring mcp.UniverseSystemPlanet's shape as its own sde-owned
// type -- sde/ does not import mcp/. Computed via mapDenormalize's orbitID
// column (a moon/belt's orbitID is its parent planet's itemID) at read time,
// not a stored relation.
type SystemPlanet struct {
	PlanetID        int64
	AsteroidBeltIDs []int64
	MoonIDs         []int64
}

// System is a minimal decode of a fuzzwork mapSolarSystems row -- EVE's
// geography level between Constellation and celestial bodies. StarID,
// StargateIDs, StationIDs, and Planets are not stored columns -- all four
// are computed via reverse queries over mapDenormalize/staStations/mapJumps
// at read time (celestial groupIDs: 6=Sun, 7=Planet, 8=Moon, 9/4430=Asteroid
// Belt, 10=Stargate, verified directly against the live dump).
type System struct {
	ID              int64
	ConstellationID int64
	Name            string
	Position        Position
	SecurityStatus  float64
	SecurityClass   *string
	StarID          *int64
	StargateIDs     []int64
	StationIDs      []int64
	Planets         []SystemPlanet
}

// System looks up a solar system by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) System(ctx context.Context, userAgent string, id int64) (System, bool, error) {
	if s == nil {
		return System{}, false, nil
	}
	if v, ok := getOverlay[System](s, "system", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return System{}, false, nil
	}

	var (
		constellationID int64
		name            string
		x, y, z         sql.NullFloat64
		security        sql.NullFloat64
		securityClass   sql.NullString
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT constellationID, solarSystemName, x, y, z, security, securityClass FROM mapSolarSystems WHERE solarSystemID = ?`, id)
	if err := row.Scan(&constellationID, &name, &x, &y, &z, &security, &securityClass); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return System{}, false, nil
		}
		return System{}, false, fmt.Errorf("query system %d: %w", id, err)
	}

	out := System{
		ID: id, ConstellationID: constellationID, Name: name,
		Position:       Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
		SecurityStatus: security.Float64,
	}
	if securityClass.Valid {
		out.SecurityClass = &securityClass.String
	}

	starIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE solarSystemID = ? AND groupID = 6 LIMIT 1`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d star id: %w", id, err)
	}
	if len(starIDs) > 0 {
		out.StarID = &starIDs[0]
	}

	stargateIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE solarSystemID = ? AND groupID = 10 ORDER BY itemID`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d stargate ids: %w", id, err)
	}
	out.StargateIDs = stargateIDs

	stationIDs, err := s.queryInt64Column(ctx, `SELECT stationID FROM staStations WHERE solarSystemID = ? ORDER BY stationID`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d station ids: %w", id, err)
	}
	out.StationIDs = stationIDs

	planetIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE solarSystemID = ? AND groupID = 7 ORDER BY celestialIndex`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d planet ids: %w", id, err)
	}
	for _, planetID := range planetIDs {
		moonIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE orbitID = ? AND groupID = 8 ORDER BY itemID`, planetID)
		if err != nil {
			return System{}, false, fmt.Errorf("query system %d planet %d moon ids: %w", id, planetID, err)
		}
		beltIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE orbitID = ? AND groupID IN (9, 4430) ORDER BY itemID`, planetID)
		if err != nil {
			return System{}, false, fmt.Errorf("query system %d planet %d belt ids: %w", id, planetID, err)
		}
		out.Planets = append(out.Planets, SystemPlanet{PlanetID: planetID, AsteroidBeltIDs: beltIDs, MoonIDs: moonIDs})
	}

	return out, true, nil
}

// PutSystem stores sys in the overlay, keyed by sys.ID -- used by mcp/'s
// read-through fallback. Unlike the sqlite-served path, a read-through fetch
// gets StationIDs and Planets directly from ESI's response, no further
// queries needed.
func (s *Store) PutSystem(sys System) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "system", sys.ID, sys)
	return nil
}

// SystemIDs returns every solar system ID currently cached, ascending.
func (s *Store) SystemIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT solarSystemID FROM mapSolarSystems ORDER BY solarSystemID`)
}
```

- [ ] **Step 4: Rewrite `sde/station.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Station is a minimal decode of a fuzzwork staStations row. RaceID is not a
// stored column on staStations at all -- it's joined in from the station's
// own type (invTypes.raceID via stationTypeID), the same value ESI itself
// almost certainly derives it from for NPC stations, but a join rather than
// a stored fact (a semantic shift from the old esi-scrape design's direct
// field, documented in the migration design spec). Services is always empty
// from a sqlite-served lookup -- staOperationServices has zero rows in the
// dump (verified directly), a documented field-drop; a cache-miss overlay
// write can still carry a populated one.
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

// Station looks up a station by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Station(ctx context.Context, userAgent string, id int64) (Station, bool, error) {
	if s == nil {
		return Station{}, false, nil
	}
	if v, ok := getOverlay[Station](s, "station", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Station{}, false, nil
	}

	var (
		name                                                   string
		systemID, typeID                                       int64
		x, y, z                                                sql.NullFloat64
		maxDockable, officeRentalCost, reprocEff, reprocTake    sql.NullFloat64
		owner                                                   sql.NullInt64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT stationName, solarSystemID, stationTypeID, x, y, z, maxShipVolumeDockable, officeRentalCost, corporationID, reprocessingEfficiency, reprocessingStationsTake
		 FROM staStations WHERE stationID = ?`, id)
	if err := row.Scan(&name, &systemID, &typeID, &x, &y, &z, &maxDockable, &officeRentalCost, &owner, &reprocEff, &reprocTake); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Station{}, false, nil
		}
		return Station{}, false, fmt.Errorf("query station %d: %w", id, err)
	}

	out := Station{
		ID: id, Name: name, SystemID: systemID, TypeID: typeID,
		Position:                 Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
		MaxDockableShipVolume:    maxDockable.Float64,
		OfficeRentalCost:         officeRentalCost.Float64,
		Owner:                    nullInt(owner),
		ReprocessingEfficiency:   reprocEff.Float64,
		ReprocessingStationsTake: reprocTake.Float64,
		Services:                []string{},
	}

	var raceID sql.NullInt64
	raceRow := s.db.QueryRowContext(ctx, `SELECT raceID FROM invTypes WHERE typeID = ?`, typeID)
	if err := raceRow.Scan(&raceID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Station{}, false, fmt.Errorf("query station %d type %d race id: %w", id, typeID, err)
	}
	out.RaceID = nullInt(raceID)

	return out, true, nil
}

// PutStation stores st in the overlay, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStation(st Station) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "station", st.ID, st)
	return nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./sde/... -run 'TestSystem|TestPutSystem|TestSystemIDs|TestStation|TestPutStation' -v`
Expected: still a compile failure per this plan's "A note on task-by-task test execution" — `sde/planet.go` and later files haven't migrated yet. Review `system.go`/`station.go`'s diffs by inspection; they genuinely pass once Task 8 completes.

- [ ] **Step 6: Commit**

```bash
git add sde/system.go sde/system_test.go sde/station.go sde/station_test.go
git commit -m "sde: rewrite System/Station against mapSolarSystems/staStations/mapDenormalize

System's StarID/StargateIDs/StationIDs/Planets all move to read-time
reverse queries over mapDenormalize (celestial groupIDs 6/7/8/9/10) and
staStations -- more complete than the old best-effort esi-scrape path,
since mapDenormalize is official-SDE-derived. Station.RaceID becomes a
join to invTypes; Services regresses to always-empty (staOperationServices
is empty in the dump), documented in the migration design spec."
```

---

## Task 7: `sde/planet.go` + `sde/moon.go` + `sde/asteroidbelt.go` + `sde/star.go` + `sde/stargate.go` — `mapDenormalize`/`mapCelestialStatistics`/`mapJumps`

**Files:**
- Modify: `sde/planet.go`, `sde/moon.go`, `sde/asteroidbelt.go`, `sde/star.go`, `sde/stargate.go` (full rewrites)
- Test: `sde/planet_test.go`, `sde/moon_test.go`, `sde/asteroidbelt_test.go`, `sde/star_test.go`, `sde/stargate_test.go` (full rewrites)

**Interfaces:**
- Consumes: `putOverlay`/`getOverlay`/`nullInt` (Task 1), `Position` (Task 5), `ensureSynced` (Task 2), `seedStore`/`seedExec` (Task 1).
- Produces: `type Planet struct{ ID, TypeID, SystemID int64; Name string; Position Position }`, `func (s *Store) Planet(...) (Planet, bool, error)`, `func (s *Store) PutPlanet(Planet) error`.
- Produces: `type Moon struct{ ID, SystemID int64; Name string; Position Position }`, `func (s *Store) Moon(...) (Moon, bool, error)`, `func (s *Store) PutMoon(Moon) error`.
- Produces: `type AsteroidBelt struct{ ID, SystemID int64; Name string; Position Position }`, `func (s *Store) AsteroidBelt(...) (AsteroidBelt, bool, error)`, `func (s *Store) PutAsteroidBelt(AsteroidBelt) error`.
- Produces: `type Star struct{ ID, SolarSystemID, TypeID, Age, Radius, Temperature int64; Luminosity float64; Name, SpectralClass string }`, `func (s *Store) Star(...) (Star, bool, error)`, `func (s *Store) PutStar(Star) error`.
- Produces: `type Stargate struct{ ID, TypeID, SystemID, DestinationStargateID, DestinationSystemID int64; Name string; Position Position }`, `func (s *Store) Stargate(...) (Stargate, bool, error)`, `func (s *Store) PutStargate(Stargate) error`.

All five share the same shape: a single-row query against `mapDenormalize` filtered by `itemID` and the resource's `groupID` (7=Planet, 8=Moon, 9/4430=Asteroid Belt, 6=Sun, 10=Stargate), keeping the existing `AsteroidBelt`/`Moon`/`Planet`/`Star`/`Stargate` doc comments' explanation of *why* each one exists (no separate FC jsonl source, all celestial detail lives only in `mapDenormalize`/`mapCelestialStatistics` now, not esi-scrape).

- [ ] **Step 1: Write the failing tests**

Replace `sde/planet_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestPlanet_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (40009077, 2016, 7, 30000142, 1.0, 2.0, 3.0, 'Jita 1')`)

	got, ok, err := store.Planet(context.Background(), testUserAgent, 40009077)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false, want true")
	}
	if got.ID != 40009077 || got.TypeID != 2016 || got.SystemID != 30000142 || got.Name != "Jita 1" {
		t.Errorf("got = %+v, want ID=40009077 TypeID=2016 SystemID=30000142 Name=%q", got, "Jita 1")
	}
	if got.Position != (Position{X: 1.0, Y: 2.0, Z: 3.0}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
}

func TestPlanet_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Planet(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if ok {
		t.Error("Planet ok = true, want false for an unknown ID")
	}
}

func TestPlanet_WrongGroupID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009078, 14, 8, 30000142, 'Jita 1 - Moon 1')`)

	_, ok, err := store.Planet(context.Background(), testUserAgent, 40009078)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if ok {
		t.Error("Planet ok = true, want false for a moon's itemID (groupID 8, not 7)")
	}
}

func TestPutPlanet_ThenPlanet_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Planet{ID: 999, Name: "Read-Through Planet", TypeID: 2016, SystemID: 30000142}

	if err := store.PutPlanet(want); err != nil {
		t.Fatalf("PutPlanet returned error: %v", err)
	}
	got, ok, err := store.Planet(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false, want true after PutPlanet")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
```

Replace `sde/moon_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestMoon_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (40009078, 14, 8, 30000142, 1.0, 2.0, 3.0, 'Jita 1 - Moon 1')`)

	got, ok, err := store.Moon(context.Background(), testUserAgent, 40009078)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false, want true")
	}
	if got.ID != 40009078 || got.SystemID != 30000142 || got.Name != "Jita 1 - Moon 1" {
		t.Errorf("got = %+v, want ID=40009078 SystemID=30000142 Name=%q", got, "Jita 1 - Moon 1")
	}
}

func TestMoon_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Moon(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if ok {
		t.Error("Moon ok = true, want false for an unknown ID")
	}
}

func TestPutMoon_ThenMoon_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Moon{ID: 999, Name: "Read-Through Moon", SystemID: 30000142}

	if err := store.PutMoon(want); err != nil {
		t.Fatalf("PutMoon returned error: %v", err)
	}
	got, ok, err := store.Moon(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false, want true after PutMoon")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
```

Replace `sde/asteroidbelt_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestAsteroidBelt_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (40009079, 16, 9, 30000142, 1.0, 2.0, 3.0, 'Jita 1 - Asteroid Belt 1')`)

	got, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40009079)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false, want true")
	}
	if got.ID != 40009079 || got.SystemID != 30000142 || got.Name != "Jita 1 - Asteroid Belt 1" {
		t.Errorf("got = %+v, want ID=40009079 SystemID=30000142 Name=%q", got, "Jita 1 - Asteroid Belt 1")
	}
}

func TestAsteroidBelt_GroupID4430_AlsoMatches(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009999, 16, 4430, 30000142, 'Wormhole Belt')`)

	_, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40009999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Error("AsteroidBelt ok = false, want true for groupID 4430 (the wormhole-system asteroid belt group)")
	}
}

func TestAsteroidBelt_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if ok {
		t.Error("AsteroidBelt ok = true, want false for an unknown ID")
	}
}

func TestPutAsteroidBelt_ThenAsteroidBelt_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := AsteroidBelt{ID: 999, Name: "Read-Through Belt", SystemID: 30000142}

	if err := store.PutAsteroidBelt(want); err != nil {
		t.Fatalf("PutAsteroidBelt returned error: %v", err)
	}
	got, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false, want true after PutAsteroidBelt")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
```

Replace `sde/star_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestStar_KnownID_ReturnsRowWithCelestialStatistics(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009076, 3796, 6, 30000142, 'Jita - Sun')`)
	seedExec(t, store, `INSERT INTO mapCelestialStatistics (celestialID, temperature, spectralClass, luminosity, age, radius) VALUES (40009076, 7305.0, 'F1 V', 1.692, 21750000000000000.0, 696000000.0)`)

	got, ok, err := store.Star(context.Background(), testUserAgent, 40009076)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true")
	}
	if got.ID != 40009076 || got.TypeID != 3796 || got.SolarSystemID != 30000142 || got.Name != "Jita - Sun" {
		t.Errorf("got = %+v, want ID=40009076 TypeID=3796 SolarSystemID=30000142 Name=%q", got, "Jita - Sun")
	}
	if got.SpectralClass != "F1 V" || got.Luminosity != 1.692 || got.Temperature != 7305 {
		t.Errorf("SpectralClass/Luminosity/Temperature = %q/%v/%v, want F1 V/1.692/7305", got.SpectralClass, got.Luminosity, got.Temperature)
	}
}

func TestStar_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Star(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if ok {
		t.Error("Star ok = true, want false for an unknown ID")
	}
}

func TestStar_NoCelestialStatisticsRow_ReturnsZeroValuedPhysicalFields(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009076, 3796, 6, 30000142, 'Jita - Sun')`)

	got, ok, err := store.Star(context.Background(), testUserAgent, 40009076)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true even without a mapCelestialStatistics row")
	}
	if got.SpectralClass != "" || got.Luminosity != 0 {
		t.Errorf("SpectralClass/Luminosity = %q/%v, want zero values", got.SpectralClass, got.Luminosity)
	}
}

func TestPutStar_ThenStar_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Star{ID: 999, Name: "Read-Through Star", SolarSystemID: 30000142, TypeID: 3796}

	if err := store.PutStar(want); err != nil {
		t.Fatalf("PutStar returned error: %v", err)
	}
	got, ok, err := store.Star(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true after PutStar")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
```

Replace `sde/stargate_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestStargate_KnownID_ReturnsRowWithDestination(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (50008077, 16, 10, 30000142, 1.0, 2.0, 3.0, 'Stargate (Perimeter)')`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (50008078, 16, 10, 30000144, 'Stargate (Jita)')`)
	seedExec(t, store, `INSERT INTO mapJumps (stargateID, destinationID) VALUES (50008077, 50008078)`)

	got, ok, err := store.Stargate(context.Background(), testUserAgent, 50008077)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false, want true")
	}
	if got.ID != 50008077 || got.SystemID != 30000142 {
		t.Errorf("got = %+v, want ID=50008077 SystemID=30000142", got)
	}
	if got.DestinationStargateID != 50008078 {
		t.Errorf("DestinationStargateID = %d, want 50008078", got.DestinationStargateID)
	}
	if got.DestinationSystemID != 30000144 {
		t.Errorf("DestinationSystemID = %d, want 30000144 (joined from the destination gate's own solarSystemID)", got.DestinationSystemID)
	}
}

func TestStargate_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Stargate(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if ok {
		t.Error("Stargate ok = true, want false for an unknown ID")
	}
}

func TestPutStargate_ThenStargate_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Stargate{ID: 999, Name: "Read-Through Stargate", SystemID: 30000142, DestinationSystemID: 30000144}

	if err := store.PutStargate(want); err != nil {
		t.Fatalf("PutStargate returned error: %v", err)
	}
	got, ok, err := store.Stargate(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false, want true after PutStargate")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./sde/... -run 'TestPlanet|TestPutPlanet|TestMoon|TestPutMoon|TestAsteroidBelt|TestPutAsteroidBelt|TestStar|TestPutStar|TestStargate|TestPutStargate' -v`
Expected: compile failure.

- [ ] **Step 3: Rewrite `sde/planet.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Planet is a minimal decode of a fuzzwork mapDenormalize row filtered to
// groupID 7 (Planet). mapDenormalize is official-SDE-derived (unlike the old
// bbolt design's esi-scrape-only source for this data), covering every
// celestial body's name/position/system/type in one table.
type Planet struct {
	ID       int64
	Name     string
	TypeID   int64
	Position Position
	SystemID int64
}

// Planet looks up a planet by ID. ok is false if the ID isn't in the cache,
// or belongs to a different celestial groupID (e.g. a moon's itemID).
func (s *Store) Planet(ctx context.Context, userAgent string, id int64) (Planet, bool, error) {
	if s == nil {
		return Planet{}, false, nil
	}
	if v, ok := getOverlay[Planet](s, "planet", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Planet{}, false, nil
	}

	var (
		name     string
		typeID   int64
		systemID int64
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, typeID, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID = 7`, id)
	if err := row.Scan(&name, &typeID, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Planet{}, false, nil
		}
		return Planet{}, false, fmt.Errorf("query planet %d: %w", id, err)
	}

	return Planet{ID: id, Name: name, TypeID: typeID, SystemID: systemID, Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64}}, true, nil
}

// PutPlanet stores p in the overlay, keyed by p.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutPlanet(p Planet) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "planet", p.ID, p)
	return nil
}
```

- [ ] **Step 4: Rewrite `sde/moon.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Moon is a minimal decode of a fuzzwork mapDenormalize row filtered to
// groupID 8 (Moon). See Planet's doc comment for why mapDenormalize replaces
// the old esi-scrape source.
type Moon struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

// Moon looks up a moon by ID. ok is false if the ID isn't in the cache.
func (s *Store) Moon(ctx context.Context, userAgent string, id int64) (Moon, bool, error) {
	if s == nil {
		return Moon{}, false, nil
	}
	if v, ok := getOverlay[Moon](s, "moon", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Moon{}, false, nil
	}

	var (
		name     string
		systemID int64
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID = 8`, id)
	if err := row.Scan(&name, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Moon{}, false, nil
		}
		return Moon{}, false, fmt.Errorf("query moon %d: %w", id, err)
	}

	return Moon{ID: id, Name: name, SystemID: systemID, Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64}}, true, nil
}

// PutMoon stores m in the overlay, keyed by m.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutMoon(m Moon) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "moon", m.ID, m)
	return nil
}
```

- [ ] **Step 5: Rewrite `sde/asteroidbelt.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// AsteroidBelt is a minimal decode of a fuzzwork mapDenormalize row filtered
// to groupID 9 or 4430 (both observed as "Asteroid Belt" in invGroups --
// 4430 covers wormhole-system belts). Unlike every other sde/ resource,
// there is no FC SDE jsonl source for this data at all even historically --
// asteroid belt names are computed server-side by FC. mapDenormalize
// carries it directly, replacing the old esi-scrape-only source.
type AsteroidBelt struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

// AsteroidBelt looks up an asteroid belt by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) AsteroidBelt(ctx context.Context, userAgent string, id int64) (AsteroidBelt, bool, error) {
	if s == nil {
		return AsteroidBelt{}, false, nil
	}
	if v, ok := getOverlay[AsteroidBelt](s, "asteroidBelt", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return AsteroidBelt{}, false, nil
	}

	var (
		name     string
		systemID int64
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID IN (9, 4430)`, id)
	if err := row.Scan(&name, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AsteroidBelt{}, false, nil
		}
		return AsteroidBelt{}, false, fmt.Errorf("query asteroid belt %d: %w", id, err)
	}

	return AsteroidBelt{ID: id, Name: name, SystemID: systemID, Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64}}, true, nil
}

// PutAsteroidBelt stores b in the overlay, keyed by b.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutAsteroidBelt(b AsteroidBelt) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "asteroidBelt", b.ID, b)
	return nil
}
```

- [ ] **Step 6: Rewrite `sde/star.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Star is a minimal decode of a fuzzwork mapDenormalize row (groupID 6, Sun)
// joined with mapCelestialStatistics for the physical attributes
// mapDenormalize itself doesn't carry (age/luminosity/spectralClass/
// temperature) -- both tables are official-SDE-derived, replacing the old
// esi-scrape-only source. Age/Radius/Temperature are truncated to int64 from
// mapCelestialStatistics's FLOAT columns, matching this type's existing
// (pre-migration) field types.
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

// Star looks up a star by ID. ok is false if the ID isn't in the cache. A
// row missing from mapCelestialStatistics (shouldn't happen for a real sun,
// but not asserted) leaves the physical-attribute fields at their zero
// values rather than failing the whole lookup.
func (s *Store) Star(ctx context.Context, userAgent string, id int64) (Star, bool, error) {
	if s == nil {
		return Star{}, false, nil
	}
	if v, ok := getOverlay[Star](s, "star", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Star{}, false, nil
	}

	var (
		name          string
		typeID        int64
		solarSystemID int64
	)
	row := s.db.QueryRowContext(ctx, `SELECT itemName, typeID, solarSystemID FROM mapDenormalize WHERE itemID = ? AND groupID = 6`, id)
	if err := row.Scan(&name, &typeID, &solarSystemID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Star{}, false, nil
		}
		return Star{}, false, fmt.Errorf("query star %d: %w", id, err)
	}

	out := Star{ID: id, Name: name, TypeID: typeID, SolarSystemID: solarSystemID}

	var (
		temperature, luminosity, age, radius sql.NullFloat64
		spectralClass                        sql.NullString
	)
	statsRow := s.db.QueryRowContext(ctx,
		`SELECT temperature, spectralClass, luminosity, age, radius FROM mapCelestialStatistics WHERE celestialID = ?`, id)
	err := statsRow.Scan(&temperature, &spectralClass, &luminosity, &age, &radius)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Star{}, false, fmt.Errorf("query star %d celestial statistics: %w", id, err)
	}
	if err == nil {
		out.Temperature = int64(temperature.Float64)
		out.SpectralClass = spectralClass.String
		out.Luminosity = luminosity.Float64
		out.Age = int64(age.Float64)
		out.Radius = int64(radius.Float64)
	}

	return out, true, nil
}

// PutStar stores st in the overlay, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStar(st Star) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "star", st.ID, st)
	return nil
}
```

- [ ] **Step 7: Rewrite `sde/stargate.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Stargate is a minimal decode of a fuzzwork mapDenormalize row (groupID 10)
// joined with mapJumps for its destination stargate ID, plus a second query
// against the destination gate's own mapDenormalize row for
// DestinationSystemID (mapJumps only carries the destination *gate* ID, not
// its system) -- both tables official-SDE-derived, replacing the old
// esi-scrape-only source.
type Stargate struct {
	ID                    int64
	Name                  string
	TypeID                int64
	Position              Position
	SystemID              int64
	DestinationStargateID int64
	DestinationSystemID   int64
}

// Stargate looks up a stargate by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Stargate(ctx context.Context, userAgent string, id int64) (Stargate, bool, error) {
	if s == nil {
		return Stargate{}, false, nil
	}
	if v, ok := getOverlay[Stargate](s, "stargate", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Stargate{}, false, nil
	}

	var (
		name     string
		typeID   int64
		systemID int64
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, typeID, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID = 10`, id)
	if err := row.Scan(&name, &typeID, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Stargate{}, false, nil
		}
		return Stargate{}, false, fmt.Errorf("query stargate %d: %w", id, err)
	}

	out := Stargate{
		ID: id, Name: name, TypeID: typeID, SystemID: systemID,
		Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
	}

	var destinationStargateID sql.NullInt64
	destRow := s.db.QueryRowContext(ctx, `SELECT destinationID FROM mapJumps WHERE stargateID = ?`, id)
	err := destRow.Scan(&destinationStargateID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Stargate{}, false, fmt.Errorf("query stargate %d destination: %w", id, err)
	}
	if err == nil && destinationStargateID.Valid {
		out.DestinationStargateID = destinationStargateID.Int64

		var destinationSystemID sql.NullInt64
		destSystemRow := s.db.QueryRowContext(ctx, `SELECT solarSystemID FROM mapDenormalize WHERE itemID = ?`, destinationStargateID.Int64)
		if err := destSystemRow.Scan(&destinationSystemID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Stargate{}, false, fmt.Errorf("query stargate %d destination system: %w", id, err)
		}
		out.DestinationSystemID = destinationSystemID.Int64
	}

	return out, true, nil
}

// PutStargate stores g in the overlay, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStargate(g Stargate) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "stargate", g.ID, g)
	return nil
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./sde/... -run 'TestPlanet|TestPutPlanet|TestMoon|TestPutMoon|TestAsteroidBelt|TestPutAsteroidBelt|TestStar|TestPutStar|TestStargate|TestPutStargate' -v`
Expected: still a compile failure per this plan's "A note on task-by-task test execution" — `sde/faction.go` and the other Task 8 files haven't migrated yet. Review this task's five diffs by inspection; they genuinely pass once Task 8 completes.

- [ ] **Step 9: Commit**

```bash
git add sde/planet.go sde/planet_test.go sde/moon.go sde/moon_test.go sde/asteroidbelt.go sde/asteroidbelt_test.go sde/star.go sde/star_test.go sde/stargate.go sde/stargate_test.go
git commit -m "sde: rewrite celestial-body resources against mapDenormalize

Planet/Moon/AsteroidBelt/Star/Stargate all move from the old
esi-scrape-only source (best-effort, could silently skip a sync cycle)
to mapDenormalize + mapCelestialStatistics + mapJumps -- all three
official-SDE-derived, a strict reliability improvement over the design
this replaces."
```

---

## Task 8: `sde/faction.go` + `sde/race.go` + `sde/bloodline.go` + `sde/ancestry.go` + `sde/graphic.go` — the remaining `chr*`/`eveGraphics` tables

**Files:**
- Modify: `sde/faction.go`, `sde/race.go`, `sde/bloodline.go`, `sde/ancestry.go`, `sde/graphic.go` (full rewrites)
- Test: `sde/faction_test.go`, `sde/race_test.go`, `sde/bloodline_test.go`, `sde/ancestry_test.go`, `sde/graphic_test.go` (full rewrites)

**Interfaces:**
- Consumes: `putOverlay`/`getOverlay`/`queryInt64Column`/`nullInt` (Tasks 1/4), `ensureSynced` (Task 2), `s.raceAlliances`/`s.graphicSofDNA` (Task 2), `seedStore`/`seedExec` (Task 1).
- Produces: `type Faction struct{ ID int64; Name, Description string; IsUnique bool; SizeFactor float64; CorporationID, MilitiaCorporationID, SolarSystemID *int64; StationCount, StationSystemCount int64 }`, `func (s *Store) Factions(...) ([]Faction, error)`, `func (s *Store) Faction(...) (Faction, bool, error)`. **`IsUnique` is always `false` from a sqlite-served row** — `chrFactions` has no `uniqueName`-equivalent column (design spec's field-drop audit).
- Produces: `type Race struct{ ID int64; AllianceID *int64; Description, Name string }`, `func (s *Store) Races(...) ([]Race, error)`, `func (s *Store) Race(...) (Race, bool, error)`.
- Produces: `type Bloodline struct{ ID, CorporationID, RaceID, Charisma, Intelligence, Memory, Perception, Willpower int64; Description, Name string; ShipTypeID *int64 }`, `func (s *Store) Bloodlines(...) ([]Bloodline, error)`, `func (s *Store) Bloodline(...) (Bloodline, bool, error)`.
- Produces: `type Ancestry struct{ ID, BloodlineID int64; Description, Name string; IconID *int64; ShortDescription *string }`, `func (s *Store) Ancestries(...) ([]Ancestry, error)`.
- Produces: `type Graphic struct{ ID int64; GraphicFile, IconFolder, SofDNA, SofFactionName, SofHullName, SofRaceName *string }`, `func (s *Store) Graphic(...) (Graphic, bool, error)`, `func (s *Store) PutGraphic(Graphic) error`, `func (s *Store) GraphicIDs(...) ([]int64, error)`.

- [ ] **Step 1: Write the failing tests**

Replace `sde/faction_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestFactions_ReturnsEveryCachedFactionWithIsUniqueFalse(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrFactions (factionID, factionName, description, solarSystemID, corporationID, sizeFactor, stationCount, stationSystemCount, militiaCorporationID)
		VALUES (500001, 'Caldari State', 'The Caldari State.', 30000142, 1000035, 1.0, 10, 5, 1000180)`)

	got, err := store.Factions(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Factions) = %d, want 1", len(got))
	}
	f := got[0]
	if f.ID != 500001 || f.Name != "Caldari State" || f.Description != "The Caldari State." {
		t.Errorf("got = %+v, want ID=500001 Name=%q Description=%q", f, "Caldari State", "The Caldari State.")
	}
	if f.IsUnique {
		t.Error("IsUnique = true, want false (chrFactions has no uniqueName-equivalent column)")
	}
	if f.StationCount != 10 || f.StationSystemCount != 5 {
		t.Errorf("StationCount/StationSystemCount = %d/%d, want 10/5", f.StationCount, f.StationSystemCount)
	}
	if f.CorporationID == nil || *f.CorporationID != 1000035 {
		t.Errorf("CorporationID = %v, want 1000035", f.CorporationID)
	}
}

func TestFactions_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := &Store{}
	got, err := store.Factions(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Factions = %v, want empty", got)
	}
}

func TestFaction_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrFactions (factionID, factionName, sizeFactor, stationCount, stationSystemCount) VALUES (500001, 'Caldari State', 1.0, 0, 0)`)

	got, ok, err := store.Faction(context.Background(), testUserAgent, 500001)
	if err != nil {
		t.Fatalf("Faction returned error: %v", err)
	}
	if !ok {
		t.Fatal("Faction ok = false, want true")
	}
	if got.Name != "Caldari State" {
		t.Errorf("Name = %q, want Caldari State", got.Name)
	}
}

func TestFaction_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Faction(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Faction returned error: %v", err)
	}
	if ok {
		t.Error("Faction ok = true, want false for an unknown ID")
	}
}
```

Replace `sde/race_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestRaces_MergesAllianceIDFromInMemoryAugmentation(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrRaces (raceID, raceName, description) VALUES (1, 'Caldari', 'The Caldari.')`)
	store.raceAlliances = map[int64]int64{1: 500001}

	got, err := store.Races(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Races) = %d, want 1", len(got))
	}
	r := got[0]
	if r.ID != 1 || r.Name != "Caldari" || r.Description != "The Caldari." {
		t.Errorf("got = %+v, want ID=1 Name=Caldari Description=%q", r, "The Caldari.")
	}
	if r.AllianceID == nil || *r.AllianceID != 500001 {
		t.Errorf("AllianceID = %v, want 500001", r.AllianceID)
	}
}

func TestRaces_NoAugmentation_ReturnsNilAllianceID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrRaces (raceID, raceName) VALUES (1, 'Caldari')`)

	got, err := store.Races(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 || got[0].AllianceID != nil {
		t.Errorf("Races = %+v, want one Race with nil AllianceID", got)
	}
}

func TestRace_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrRaces (raceID, raceName) VALUES (1, 'Caldari')`)

	got, ok, err := store.Race(context.Background(), testUserAgent, 1)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if !ok {
		t.Fatal("Race ok = false, want true")
	}
	if got.Name != "Caldari" {
		t.Errorf("Name = %q, want Caldari", got.Name)
	}
}

func TestRace_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Race(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if ok {
		t.Error("Race ok = true, want false for an unknown ID")
	}
}
```

Replace `sde/bloodline_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestBloodlines_ReturnsEveryCachedBloodline(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrBloodlines (bloodlineID, bloodlineName, raceID, description, shipTypeID, corporationID, perception, willpower, charisma, memory, intelligence)
		VALUES (1, 'Deteis', 1, 'The Deteis.', 601, 1000035, 5, 6, 7, 8, 9)`)

	got, err := store.Bloodlines(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Bloodlines returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Bloodlines) = %d, want 1", len(got))
	}
	b := got[0]
	if b.ID != 1 || b.Name != "Deteis" || b.RaceID != 1 || b.CorporationID != 1000035 {
		t.Errorf("got = %+v, want ID=1 Name=Deteis RaceID=1 CorporationID=1000035", b)
	}
	if b.ShipTypeID == nil || *b.ShipTypeID != 601 {
		t.Errorf("ShipTypeID = %v, want 601", b.ShipTypeID)
	}
	if b.Perception != 5 || b.Willpower != 6 || b.Charisma != 7 || b.Memory != 8 || b.Intelligence != 9 {
		t.Errorf("attribute fields = %+v, want Perception=5 Willpower=6 Charisma=7 Memory=8 Intelligence=9", b)
	}
}

func TestBloodline_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrBloodlines (bloodlineID, bloodlineName, raceID) VALUES (1, 'Deteis', 1)`)

	got, ok, err := store.Bloodline(context.Background(), testUserAgent, 1)
	if err != nil {
		t.Fatalf("Bloodline returned error: %v", err)
	}
	if !ok {
		t.Fatal("Bloodline ok = false, want true")
	}
	if got.Name != "Deteis" {
		t.Errorf("Name = %q, want Deteis", got.Name)
	}
}

func TestBloodline_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Bloodline(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Bloodline returned error: %v", err)
	}
	if ok {
		t.Error("Bloodline ok = true, want false for an unknown ID")
	}
}
```

Replace `sde/ancestry_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestAncestries_ReturnsEveryCachedAncestry(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrAncestries (ancestryID, ancestryName, bloodlineID, description, iconID, shortDescription)
		VALUES (1, 'Immigrant', 1, 'An immigrant.', 100, 'Short.')`)

	got, err := store.Ancestries(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Ancestries) = %d, want 1", len(got))
	}
	a := got[0]
	if a.ID != 1 || a.Name != "Immigrant" || a.BloodlineID != 1 || a.Description != "An immigrant." {
		t.Errorf("got = %+v, want ID=1 Name=Immigrant BloodlineID=1 Description=%q", a, "An immigrant.")
	}
	if a.IconID == nil || *a.IconID != 100 {
		t.Errorf("IconID = %v, want 100", a.IconID)
	}
	if a.ShortDescription == nil || *a.ShortDescription != "Short." {
		t.Errorf("ShortDescription = %v, want Short.", a.ShortDescription)
	}
}

func TestAncestries_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := &Store{}
	got, err := store.Ancestries(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Ancestries = %v, want empty", got)
	}
}
```

Replace `sde/graphic_test.go` entirely:

```go
package sde

import (
	"context"
	"testing"
)

func TestGraphic_KnownID_MergesSofDNAFromInMemoryAugmentation(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID, sofFactionName, graphicFile, sofHullName, sofRaceName, iconFolder)
		VALUES (2, 'caldari', 'res:/dx9/model/graphic.red', 'caldari_hull', 'caldari', 'icon/Folder/')`)
	store.graphicSofDNA = map[int64]string{2: "caldari:caldari:cal"}

	got, ok, err := store.Graphic(context.Background(), testUserAgent, 2)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.ID != 2 || got.SofFactionName == nil || *got.SofFactionName != "caldari" {
		t.Errorf("got = %+v, want ID=2 SofFactionName=caldari", got)
	}
	if got.SofDNA == nil || *got.SofDNA != "caldari:caldari:cal" {
		t.Errorf("SofDNA = %v, want caldari:caldari:cal", got.SofDNA)
	}
}

func TestGraphic_NoAugmentation_ReturnsNilSofDNA(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID, sofFactionName) VALUES (2, 'caldari')`)

	got, ok, err := store.Graphic(context.Background(), testUserAgent, 2)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok || got.SofDNA != nil {
		t.Errorf("Graphic = %+v, ok=%v, want SofDNA=nil ok=true", got, ok)
	}
}

func TestGraphic_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Graphic(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if ok {
		t.Error("Graphic ok = true, want false for an unknown ID")
	}
}

func TestPutGraphic_ThenGraphic_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	sofDNA := "read-through:dna"
	want := Graphic{ID: 999, SofDNA: &sofDNA}

	if err := store.PutGraphic(want); err != nil {
		t.Fatalf("PutGraphic returned error: %v", err)
	}
	got, ok, err := store.Graphic(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true after PutGraphic")
	}
	if got.SofDNA == nil || *got.SofDNA != sofDNA {
		t.Errorf("SofDNA = %v, want %q", got.SofDNA, sofDNA)
	}
}

func TestGraphicIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID) VALUES (2)`)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID) VALUES (3)`)

	got, err := store.GraphicIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("GraphicIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("GraphicIDs = %v, want [2 3]", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./sde/... -run 'TestFaction|TestRace|TestBloodline|TestAncestries|TestGraphic|TestPutGraphic|TestGraphicIDs' -v`
Expected: compile failure.

- [ ] **Step 3: Rewrite `sde/faction.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Faction is a minimal decode of a fuzzwork chrFactions row. IsUnique is
// always false -- chrFactions has no uniqueName-equivalent column (checked
// directly via PRAGMA table_info), a documented field-drop in the migration
// design spec. Every real EVE faction name is in fact unique, so this
// field's practical value was already close to constant true.
type Faction struct {
	ID                   int64
	Name                 string
	Description          string
	IsUnique             bool
	SizeFactor           float64
	CorporationID        *int64
	MilitiaCorporationID *int64
	SolarSystemID        *int64
	StationCount         int64
	StationSystemCount   int64
}

func scanFactionRow(scanner interface{ Scan(...any) error }) (Faction, error) {
	var (
		id                                             int64
		name                                           string
		description                                    sql.NullString
		solarSystemID, corporationID, militiaCorpID     sql.NullInt64
		sizeFactor                                      sql.NullFloat64
		stationCount, stationSystemCount                sql.NullInt64
	)
	if err := scanner.Scan(&id, &name, &description, &solarSystemID, &corporationID, &sizeFactor, &stationCount, &stationSystemCount, &militiaCorpID); err != nil {
		return Faction{}, err
	}
	return Faction{
		ID: id, Name: name, Description: description.String, SizeFactor: sizeFactor.Float64,
		CorporationID: nullInt(corporationID), MilitiaCorporationID: nullInt(militiaCorpID), SolarSystemID: nullInt(solarSystemID),
		StationCount: stationCount.Int64, StationSystemCount: stationSystemCount.Int64,
	}, nil
}

const factionSelectColumns = `factionID, factionName, description, solarSystemID, corporationID, sizeFactor, stationCount, stationSystemCount, militiaCorporationID`

// Factions returns every cached faction. Returns an empty slice, not an
// error, if the store has never synced successfully -- mcp/'s caller treats
// an empty result as a signal to read-through to a live ESI list call.
func (s *Store) Factions(ctx context.Context, userAgent string) ([]Faction, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT `+factionSelectColumns+` FROM chrFactions ORDER BY factionID`)
	if err != nil {
		return nil, fmt.Errorf("query factions: %w", err)
	}
	defer rows.Close()

	var out []Faction
	for rows.Next() {
		f, err := scanFactionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan faction: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Faction looks up a faction by ID. ok is false if the ID isn't in the cache.
func (s *Store) Faction(ctx context.Context, userAgent string, id int64) (Faction, bool, error) {
	if s == nil {
		return Faction{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Faction{}, false, nil
	}

	row := s.db.QueryRowContext(ctx, `SELECT `+factionSelectColumns+` FROM chrFactions WHERE factionID = ?`, id)
	f, err := scanFactionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Faction{}, false, nil
	}
	if err != nil {
		return Faction{}, false, fmt.Errorf("query faction %d: %w", id, err)
	}
	return f, true, nil
}
```

- [ ] **Step 4: Rewrite `sde/race.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Race is a minimal decode of a fuzzwork chrRaces row, augmented at read
// time with AllianceID from EVE Ref's esi-scrape dataset (in-memory only --
// see sync.go) -- chrRaces has no such column, and there is no alliances
// table anywhere in the fuzzwork dump either.
type Race struct {
	ID          int64
	AllianceID  *int64
	Description string
	Name        string
}

// Races returns every cached race. Returns an empty slice, not an error, if
// the store has never synced successfully.
func (s *Store) Races(ctx context.Context, userAgent string) ([]Race, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT raceID, raceName, description FROM chrRaces ORDER BY raceID`)
	if err != nil {
		return nil, fmt.Errorf("query races: %w", err)
	}
	defer rows.Close()

	var out []Race
	for rows.Next() {
		var (
			id          int64
			name        string
			description sql.NullString
		)
		if err := rows.Scan(&id, &name, &description); err != nil {
			return nil, fmt.Errorf("scan race: %w", err)
		}
		r := Race{ID: id, Name: name, Description: description.String}
		if allianceID, ok := s.raceAlliances[id]; ok {
			r.AllianceID = &allianceID
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Race looks up a race by ID. ok is false if the ID isn't in the cache.
func (s *Store) Race(ctx context.Context, userAgent string, id int64) (Race, bool, error) {
	if s == nil {
		return Race{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Race{}, false, nil
	}

	var (
		name        string
		description sql.NullString
	)
	row := s.db.QueryRowContext(ctx, `SELECT raceName, description FROM chrRaces WHERE raceID = ?`, id)
	if err := row.Scan(&name, &description); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Race{}, false, nil
		}
		return Race{}, false, fmt.Errorf("query race %d: %w", id, err)
	}

	out := Race{ID: id, Name: name, Description: description.String}
	if allianceID, ok := s.raceAlliances[id]; ok {
		out.AllianceID = &allianceID
	}
	return out, true, nil
}
```

- [ ] **Step 5: Rewrite `sde/bloodline.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Bloodline is a minimal decode of a fuzzwork chrBloodlines row.
type Bloodline struct {
	ID            int64
	Charisma      int64
	CorporationID int64
	Description   string
	Intelligence  int64
	Memory        int64
	Name          string
	Perception    int64
	RaceID        int64
	ShipTypeID    *int64
	Willpower     int64
}

const bloodlineSelectColumns = `bloodlineID, bloodlineName, raceID, description, shipTypeID, corporationID, perception, willpower, charisma, memory, intelligence`

func scanBloodlineRow(scanner interface{ Scan(...any) error }) (Bloodline, error) {
	var (
		id, raceID  int64
		name        string
		description sql.NullString
		shipTypeID  sql.NullInt64
		// corporationID and the five attribute columns are nullable in the
		// fuzzwork schema even though the produced Bloodline type models
		// them as plain int64 -- scan through sql.NullInt64 so a NULL
		// column (e.g. a partially-seeded row) degrades to the zero value
		// instead of failing the scan outright.
		corporationID, perception, willpower, charisma, memory, intelligence sql.NullInt64
	)
	if err := scanner.Scan(&id, &name, &raceID, &description, &shipTypeID, &corporationID, &perception, &willpower, &charisma, &memory, &intelligence); err != nil {
		return Bloodline{}, err
	}
	return Bloodline{
		ID: id, Name: name, RaceID: raceID, Description: description.String, ShipTypeID: nullInt(shipTypeID),
		CorporationID: corporationID.Int64, Perception: perception.Int64, Willpower: willpower.Int64,
		Charisma: charisma.Int64, Memory: memory.Int64, Intelligence: intelligence.Int64,
	}, nil
}

// Bloodlines returns every cached bloodline. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Bloodlines(ctx context.Context, userAgent string) ([]Bloodline, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT `+bloodlineSelectColumns+` FROM chrBloodlines ORDER BY bloodlineID`)
	if err != nil {
		return nil, fmt.Errorf("query bloodlines: %w", err)
	}
	defer rows.Close()

	var out []Bloodline
	for rows.Next() {
		b, err := scanBloodlineRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bloodline: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Bloodline looks up a bloodline by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Bloodline(ctx context.Context, userAgent string, id int64) (Bloodline, bool, error) {
	if s == nil {
		return Bloodline{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Bloodline{}, false, nil
	}

	row := s.db.QueryRowContext(ctx, `SELECT `+bloodlineSelectColumns+` FROM chrBloodlines WHERE bloodlineID = ?`, id)
	b, err := scanBloodlineRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Bloodline{}, false, nil
	}
	if err != nil {
		return Bloodline{}, false, fmt.Errorf("query bloodline %d: %w", id, err)
	}
	return b, true, nil
}
```

- [ ] **Step 6: Rewrite `sde/ancestry.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"fmt"
)

// Ancestry is a minimal decode of a fuzzwork chrAncestries row.
type Ancestry struct {
	ID               int64
	BloodlineID      int64
	Description      string
	Name             string
	IconID           *int64
	ShortDescription *string
}

// Ancestries returns every cached ancestry. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Ancestries(ctx context.Context, userAgent string) ([]Ancestry, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT ancestryID, ancestryName, bloodlineID, description, iconID, shortDescription FROM chrAncestries ORDER BY ancestryID`)
	if err != nil {
		return nil, fmt.Errorf("query ancestries: %w", err)
	}
	defer rows.Close()

	var out []Ancestry
	for rows.Next() {
		var (
			id, bloodlineID  int64
			name             string
			description      sql.NullString
			iconID           sql.NullInt64
			shortDescription sql.NullString
		)
		if err := rows.Scan(&id, &name, &bloodlineID, &description, &iconID, &shortDescription); err != nil {
			return nil, fmt.Errorf("scan ancestry: %w", err)
		}
		a := Ancestry{ID: id, Name: name, BloodlineID: bloodlineID, Description: description.String, IconID: nullInt(iconID)}
		if shortDescription.Valid {
			a.ShortDescription = &shortDescription.String
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
```

- [ ] **Step 7: Rewrite `sde/graphic.go`**

```go
package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Graphic is a minimal decode of a fuzzwork eveGraphics row, augmented at
// read time with SofDNA from EVE Ref's esi-scrape dataset (in-memory only --
// see sync.go). CollisionFile is deliberately not modeled -- confirmed
// absent from both the fuzzwork dump and the esi-scrape mirror of ESI's own
// live responses.
type Graphic struct {
	ID             int64
	GraphicFile    *string
	IconFolder     *string
	SofDNA         *string
	SofFactionName *string
	SofHullName    *string
	SofRaceName    *string
}

// Graphic looks up a graphic by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Graphic(ctx context.Context, userAgent string, id int64) (Graphic, bool, error) {
	if s == nil {
		return Graphic{}, false, nil
	}
	if v, ok := getOverlay[Graphic](s, "graphic", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Graphic{}, false, nil
	}

	var graphicFile, iconFolder, sofFactionName, sofHullName, sofRaceName sql.NullString
	row := s.db.QueryRowContext(ctx,
		`SELECT graphicFile, iconFolder, sofFactionName, sofHullName, sofRaceName FROM eveGraphics WHERE graphicID = ?`, id)
	if err := row.Scan(&graphicFile, &iconFolder, &sofFactionName, &sofHullName, &sofRaceName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Graphic{}, false, nil
		}
		return Graphic{}, false, fmt.Errorf("query graphic %d: %w", id, err)
	}

	out := Graphic{ID: id}
	if graphicFile.Valid {
		out.GraphicFile = &graphicFile.String
	}
	if iconFolder.Valid {
		out.IconFolder = &iconFolder.String
	}
	if sofFactionName.Valid {
		out.SofFactionName = &sofFactionName.String
	}
	if sofHullName.Valid {
		out.SofHullName = &sofHullName.String
	}
	if sofRaceName.Valid {
		out.SofRaceName = &sofRaceName.String
	}
	if dna, ok := s.graphicSofDNA[id]; ok {
		out.SofDNA = &dna
	}

	return out, true, nil
}

// PutGraphic stores g in the overlay, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGraphic(g Graphic) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "graphic", g.ID, g)
	return nil
}

// GraphicIDs returns every graphic ID currently cached, ascending.
func (s *Store) GraphicIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT graphicID FROM eveGraphics ORDER BY graphicID`)
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./sde/... -run 'TestFaction|TestRace|TestBloodline|TestAncestries|TestGraphic|TestPutGraphic|TestGraphicIDs' -v`
Expected: PASS.

- [ ] **Step 9: Run the entire `sde/` package's test suite**

Run: `go test ./sde/... -v`
Expected: PASS across every test in the package — this is the first point since Task 1 where the whole package compiles and runs together (Tasks 3-8 each only ran their own subset while other files were still mid-migration).

- [ ] **Step 10: Commit**

```bash
git add sde/faction.go sde/faction_test.go sde/race.go sde/race_test.go sde/bloodline.go sde/bloodline_test.go sde/ancestry.go sde/ancestry_test.go sde/graphic.go sde/graphic_test.go
git commit -m "sde: rewrite Faction/Race/Bloodline/Ancestry/Graphic against chr*/eveGraphics

Faction.IsUnique regresses to always-false (chrFactions has no
uniqueName-equivalent column), documented in the migration design spec.
Race.AllianceID and Graphic.SofDNA merge in from the in-memory
augmentation maps Task 2's sync() populates. This completes sde/'s
migration off bbolt -- the whole package compiles and its full test
suite passes for the first time since Task 1."
```

---

## Task 9: `mcp/enrich.go` + `mcp/universe.go` — adapt to `Name`/`Description` becoming plain strings

**Files:**
- Modify: `mcp/enrich.go` (8 one-line edits)
- Modify: `mcp/universe.go` (10 edit sites: 4 list-fallback simplifications, 6 construct-then-extract pairs)

**Interfaces:**
- Consumes: `sde.Type`/`sde.System`/`sde.Region`/`sde.Constellation`/`sde.Race`/`sde.Bloodline`/`sde.Faction`/`sde.Group`/`sde.Category` (Tasks 3–8) — every field this task touches now has `Name`/`Description` as `string` (or, for `sde.Region.Description`, `*string`).
- Produces: no new symbols — every `UniverseX`/`resolveXNames` output type and signature is unchanged; only the internal expressions reading/constructing `sde` struct fields change.

This task is purely mechanical — no behavior changes to any MCP tool's output schema — but every edit must be applied for the package to compile, since Tasks 3–8 already changed the field types these files reference.

- [ ] **Step 1: Edit `mcp/enrich.go`'s 8 `.Name["en"]` sites**

Each of these is a one-line change from `x.Name["en"]` to `x.Name`, inside the resolver named in the comment:

```go
// resolveTypeNames
-		names[id] = t.Name["en"]
+		names[id] = t.Name

// resolveSystemNames
-		names[id] = s.Name["en"]
+		names[id] = s.Name

// resolveRegionNames
-		names[id] = r.Name["en"]
+		names[id] = r.Name

// resolveConstellationNames
-		names[id] = c.Name["en"]
+		names[id] = c.Name

// resolveRaceNames
-		names[id] = r.Name["en"]
+		names[id] = r.Name

// resolveBloodlineNames
-		names[id] = b.Name["en"]
+		names[id] = b.Name

// resolveFactionNames
-		names[id] = f.Name["en"]
+		names[id] = f.Name

// resolveGroupNames
-		names[id] = g.Name["en"]
+		names[id] = g.Name
```

`resolveStationNames`/`resolvePlanetNames`/`resolveMoonNames` are untouched — `sde.Station.Name`/`sde.Planet.Name`/`sde.Moon.Name` were already plain strings before this migration.

- [ ] **Step 2: Edit `mcp/universe.go`'s 4 list-fallback sites**

These four resources have no `PutX`/read-through path — only a full-list ESI fallback when the cache is empty — so only the cache-branch extraction needs updating, not a construction site:

```go
// listUniverseAncestries's cache branch
-		out[i] = UniverseAncestry{ID: a.ID, BloodlineID: a.BloodlineID, Description: a.Description["en"], Name: a.Name["en"], IconID: a.IconID, ShortDescription: a.ShortDescription}
+		out[i] = UniverseAncestry{ID: a.ID, BloodlineID: a.BloodlineID, Description: a.Description, Name: a.Name, IconID: a.IconID, ShortDescription: a.ShortDescription}

// listUniverseBloodlines's cache branch
-			ID: b.ID, Charisma: b.Charisma, CorporationID: b.CorporationID, Description: b.Description["en"],
-			Intelligence: b.Intelligence, Memory: b.Memory, Name: b.Name["en"], Perception: b.Perception,
+			ID: b.ID, Charisma: b.Charisma, CorporationID: b.CorporationID, Description: b.Description,
+			Intelligence: b.Intelligence, Memory: b.Memory, Name: b.Name, Perception: b.Perception,

// listUniverseRaces's cache branch
-		out[i] = UniverseRace{RaceID: r.ID, AllianceID: r.AllianceID, Description: r.Description["en"], Name: r.Name["en"]}
+		out[i] = UniverseRace{RaceID: r.ID, AllianceID: r.AllianceID, Description: r.Description, Name: r.Name}

// listUniverseFactions's cache branch
-			FactionID: f.ID, Description: f.Description["en"], Name: f.Name["en"], IsUnique: f.IsUnique,
+			FactionID: f.ID, Description: f.Description, Name: f.Name, IsUnique: f.IsUnique,
```

- [ ] **Step 3: Edit `mcp/universe.go`'s Category construct/extract pair**

```go
// getUniverseCategory's ESI-fallback construction
-			c = sde.Category{ID: detail.CategoryId, Name: map[string]string{"en": detail.Name}, Published: detail.Published, GroupIDs: detail.Groups}
+			c = sde.Category{ID: detail.CategoryId, Name: detail.Name, Published: detail.Published, GroupIDs: detail.Groups}

// getUniverseCategory's return
-	return UniverseCategory{CategoryID: c.ID, Name: c.Name["en"], Published: c.Published, GroupIDs: c.GroupIDs}, nil
+	return UniverseCategory{CategoryID: c.ID, Name: c.Name, Published: c.Published, GroupIDs: c.GroupIDs}, nil
```

- [ ] **Step 4: Edit `mcp/universe.go`'s Group construct/extract pair**

```go
// getUniverseGroup's ESI-fallback construction
-			g = sde.Group{ID: detail.GroupId, CategoryID: detail.CategoryId, Name: map[string]string{"en": detail.Name}, Published: detail.Published, TypeIDs: detail.Types}
+			g = sde.Group{ID: detail.GroupId, CategoryID: detail.CategoryId, Name: detail.Name, Published: detail.Published, TypeIDs: detail.Types}

// getUniverseGroup's return
-	return UniverseGroup{GroupID: g.ID, CategoryID: g.CategoryID, Name: g.Name["en"], Published: g.Published, TypeIDs: g.TypeIDs}, nil
+	return UniverseGroup{GroupID: g.ID, CategoryID: g.CategoryID, Name: g.Name, Published: g.Published, TypeIDs: g.TypeIDs}, nil
```

- [ ] **Step 5: Edit `mcp/universe.go`'s Type construct/extract pair**

```go
// getUniverseType's ESI-fallback construction
-			t = sde.Type{
-				ID: detail.TypeId, GroupID: detail.GroupId, Name: map[string]string{"en": detail.Name},
-				Description: map[string]string{"en": detail.Description}, Published: detail.Published,
+			t = sde.Type{
+				ID: detail.TypeId, GroupID: detail.GroupId, Name: detail.Name,
+				Description: detail.Description, Published: detail.Published,

// getUniverseType's return
-		TypeID: t.ID, Name: t.Name["en"], Description: t.Description["en"], GroupID: t.GroupID, Published: t.Published,
+		TypeID: t.ID, Name: t.Name, Description: t.Description, GroupID: t.GroupID, Published: t.Published,
```

- [ ] **Step 6: Edit `mcp/universe.go`'s Region construct/extract (the `*string` exception)**

Unlike every other resource, `sde.Region.Description` is `*string`, not `string` (Task 5), so this site simplifies further than a plain field-name change -- the intermediate `map[string]string{"en": ...}` wrapping and unwrapping disappears entirely, not just the `["en"]` lookup:

```go
// getUniverseRegion's ESI-fallback construction
-			r = sde.Region{ID: detail.RegionId, Name: map[string]string{"en": detail.Name}, ConstellationIDs: detail.Constellations}
-			if detail.Description != nil {
-				r.Description = map[string]string{"en": *detail.Description}
-			}
+			r = sde.Region{ID: detail.RegionId, Name: detail.Name, Description: detail.Description, ConstellationIDs: detail.Constellations}

// getUniverseRegion's return
-	var description *string
-	if v, ok := r.Description["en"]; ok {
-		description = &v
-	}
-	return UniverseRegion{RegionID: r.ID, Name: r.Name["en"], Description: description, ConstellationIDs: r.ConstellationIDs}, nil
+	return UniverseRegion{RegionID: r.ID, Name: r.Name, Description: r.Description, ConstellationIDs: r.ConstellationIDs}, nil
```

- [ ] **Step 7: Edit `mcp/universe.go`'s Constellation construct/extract pair**

```go
// getUniverseConstellation's ESI-fallback construction
-			c = sde.Constellation{
-				ID: detail.ConstellationId, RegionID: detail.RegionId, Name: map[string]string{"en": detail.Name},
+			c = sde.Constellation{
+				ID: detail.ConstellationId, RegionID: detail.RegionId, Name: detail.Name,

// getUniverseConstellation's return
-		ConstellationID: c.ID, RegionID: c.RegionID, Name: c.Name["en"],
+		ConstellationID: c.ID, RegionID: c.RegionID, Name: c.Name,
```

- [ ] **Step 8: Edit `mcp/universe.go`'s System construct/extract pair**

```go
// getUniverseSystem's ESI-fallback construction
-			sys = sde.System{
-				ID: detail.SystemId, ConstellationID: detail.ConstellationId, Name: map[string]string{"en": detail.Name},
+			sys = sde.System{
+				ID: detail.SystemId, ConstellationID: detail.ConstellationId, Name: detail.Name,

// getUniverseSystem's return
-		SystemID: sys.ID, ConstellationID: sys.ConstellationID, Name: sys.Name["en"],
+		SystemID: sys.ID, ConstellationID: sys.ConstellationID, Name: sys.Name,
```

- [ ] **Step 9: Run the full `mcp/` package test suite**

Run: `go test ./mcp/... -v`
Expected: PASS. Every `mcp/*_test.go` file that constructs an `sde.Type`/`sde.System`/etc. fixture with `map[string]string{"en": "..."}` will fail to compile until updated the same way (grep for `map[string]string{"en":` across `mcp/*_test.go` and change each to a plain string literal — the exact files/line numbers depend on which tests exist by the time this task runs, so enumerate them via `grep -rn 'map\[string\]string{"en"' mcp/*_test.go` and fix each occurrence the same way as this task's Steps 1-8, matching whichever `sde` struct field that particular fixture is setting).

- [ ] **Step 10: Commit**

```bash
git add mcp/enrich.go mcp/universe.go mcp/*_test.go
git commit -m "mcp: adapt to sde Name/Description becoming plain strings

Mechanical follow-through on the sde/ sqlite migration -- every
.Name[\"en\"]/.Description[\"en\"] read and map[string]string{\"en\": x}
construction site updates to match Tasks 3-8's new field types. No
MCP tool's output schema changes."
```

---

## Task 10: `commands/mcp.go`, `go.mod` cleanup, and full-repo verification

**Files:**
- Modify: `commands/mcp.go` (one comment update, no logic change)
- Modify: `go.mod`, `go.sum` (drop `go.etcd.io/bbolt`)

**Interfaces:**
- Consumes: `sde.Open`/`sde.DefaultPath`/`(*sde.Store).Close` (Task 1) — signatures unchanged, so `commands/mcp.go`'s call sites need no logic changes, only its stale comment.

- [ ] **Step 1: Update the stale bbolt-specific comment in `commands/mcp.go`**

The comment above `sde.Open` currently attributes a failed `Open` to "bbolt's exclusive file lock" — no longer accurate, since `Open` (Task 1) doesn't touch the filesystem beyond `MkdirAll` at all, and any lock contention would happen later, inside `sync()`'s sqlite connection open. Update it to describe the new failure mode:

```go
// commands/mcp.go, around line 35
-			// A failed Open (most commonly bbolt's exclusive file lock
-			// already held by another neat process sharing this home
-			// directory) must not take down the other ~190 non-sde/ tools
-			// with it -- sdeStore stays nil, and every sde.Store method is
-			// nil-safe (see Store's doc comment), so every Universe tool's
-			// existing cache-miss path already falls through to ESI.
+			// A failed Open (e.g. an unwritable ~/.neat/sde directory) must
+			// not take down the other ~190 non-sde/ tools with it --
+			// sdeStore stays nil, and every sde.Store method is nil-safe
+			// (see Store's doc comment), so every Universe tool's existing
+			// cache-miss path already falls through to ESI. A sync-time
+			// failure (e.g. the sqlite file locked by another neat process,
+			// or a failed dump download) is handled separately, inside
+			// ensureSynced -- it's cached and logged once, not surfaced here.
 			sdeStore, err := sde.Open(sdePath)
```

- [ ] **Step 2: Run `go mod tidy` to drop `go.etcd.io/bbolt`**

Run: `go mod tidy`
Expected: `go.mod`/`go.sum` no longer list `go.etcd.io/bbolt` or its transitive deps — Tasks 1–8 already removed every `"go.etcd.io/bbolt"` import from `sde/*.go`, so nothing else in the module references it. Verify explicitly:

Run: `grep -r "go.etcd.io/bbolt" go.mod go.sum`
Expected: no output.

- [ ] **Step 3: Full build/vet/test verification**

Run: `go build ./...`
Expected: succeeds with no errors.

Run: `go vet ./...`
Expected: no findings.

Run: `go test ./... -race`
Expected: PASS across every package. This is the first point in the plan where the entire module — not just `sde/` or `mcp/` in isolation — is verified together.

- [ ] **Step 4: Manual smoke test against the real fuzzwork dump**

This step exercises the real download path (not a test fixture) exactly once, since Tasks 1–9's automated tests all use `httptest`/temp files. Move the user's already-downloaded `~/.neat/sde/eve.db` aside first (it was hand-placed for this investigation, not produced by `sync()`, so its schema/freshness can't be assumed compatible with what Task 1's `Open`/Task 2's `sync` now expect):

```bash
mv ~/.neat/sde/eve.db ~/.neat/sde/eve.db.manual-investigation-backup
```

**Correction found during execution:** starting `go run ./cmd mcp &` and sleeping does *not* exercise the sync path — `ensureSynced` only runs inside an actual `sde.Store` accessor call (`Type`, `RegionIDs`, ...), and an idle stdio MCP server that receives no tool call never reaches one. `neat mcp`'s sync is lazy-per-accessor-call by design (see `sde/sync.go`'s `ensureSynced` doc comment), not eager on process start. Exercise the real path directly instead, with a small throwaway program (delete it afterward, never commit it):

```go
// /tmp/sde_smoke_test.go -- delete after running, do not commit
package main

import (
	"context"
	"fmt"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

func main() {
	path, _ := sde.DefaultPath()
	store, err := sde.Open(path)
	if err != nil {
		panic(err)
	}
	defer store.Close()
	ids, err := store.RegionIDs(context.Background(), "neat-smoke-test/0.1")
	fmt.Printf("regions=%d err=%v\n", len(ids), err)
}
```

```bash
go run /tmp/sde_smoke_test.go
ls -la ~/.neat/sde/
rm /tmp/sde_smoke_test.go
```

Expected: `regions=<nonzero> err=<nil>`, and `~/.neat/sde/eve.db`/`~/.neat/sde/meta.json` both exist with a fresh mtime and `meta.json`'s `schemaVersion` reading `"3"`. If this step fails, do not proceed to Task 11 — diagnose against the real fuzzwork endpoint before declaring the migration done, since every other test in this plan runs against synthetic fixtures.

- [ ] **Step 5: Commit**

```bash
git add commands/mcp.go go.mod go.sum
git commit -m "chore: drop bbolt dependency, update stale sync-failure comment

go.etcd.io/bbolt is no longer imported anywhere in the module now that
sde/ is fully migrated to a directly-queried sqlite file (Tasks 1-9).
Verified via go build/vet/test ./... and a live smoke test against the
real fuzzwork dump."
```

---

## Task 11: Update `CLAUDE.md`, `README.md`, and the three superseded specs' status lines

**Files:**
- Modify: `CLAUDE.md` (package table row + Hard rules paragraph)
- Modify: `README.md` (architecture summary sentence + package table row)
- Modify: `docs/superpowers/specs/2026-07-27-sde-design.md`, `docs/superpowers/specs/2026-07-27-sde-everef-supplemental-sources-design.md`, `docs/superpowers/specs/2026-07-27-sde-universe-migration-design.md` (one-line superseded-by pointer each)

**Interfaces:** None — this task only touches prose documentation, no code.

- [ ] **Step 1: Update `CLAUDE.md`'s `sde/` package-table row**

```markdown
- `sde/` | Local cache of FC's Static Data Export plus EVE Ref's
  `reference-data`/`esi-scrape` bulk exports, backed by `go.etcd.io/bbolt`
  (`~/.neat/sde/sde.db`). Lazily syncs on first lookup call per `Store`
  lifetime (at most once per `neat mcp` process run) — the FC SDE zip fetch
  is sync-fatal; the two EVE Ref sources are independently best-effort (a
  failure there logs a warning and skips that source's augmentation, never
  fails the rest of the sync), since SDE data is enrichment, not a hard
  requirement. Backs 24 of the Universe tag's 30 tools via a read-through
  cache (`mcp/universe.go` orchestrates the ESI fallback and write-back;
  `sde/` itself never imports `esi/`) — the other 6 are permanently
  ESI-direct by design (live/non-cacheable data), and only
  `get_universe_graphic`'s `CollisionFile` field remains permanently
  unsourced, confirmed absent from every dataset checked.
+ `sde/` | Local cache of fuzzwork.co.uk's periodically-refreshed SQLite
+ conversion of FC's Static Data Export (`~/.neat/sde/eve.db`), queried
+ directly via `database/sql` + `modernc.org/sqlite` — no intermediate
+ key-value store. Lazily syncs on first lookup call per `Store` lifetime (at
+ most once per `neat mcp` process run): an ETag-gated download+gunzip+atomic-
+ rename of the dump (sync-fatal on failure) plus a narrow, in-memory-only
+ fetch from EVE Ref's `reference-data`/`esi-scrape` for the three fields
+ (type `PackagedVolume`, race `AllianceID`, graphic `SofDNA`) with no source
+ in the dump — both independently best-effort. `PutX`'s cache-miss write-back
+ lands in a process-lifetime in-memory overlay, not the downloaded file
+ itself, which is treated as read-only. Backs 24 of the Universe tag's 30
+ tools via a read-through cache (`mcp/universe.go` orchestrates the ESI
+ fallback and write-back; `sde/` itself never imports `esi/`) — the other 6
+ are permanently ESI-direct by design (live/non-cacheable data). Four fields
+ are permanently unsourced across every dataset checked:
+ `get_universe_graphic`'s `CollisionFile`, `get_universe_region`'s
+ `Description`, `get_universe_station`'s `Services`, and
+ `list_universe_factions`'s `IsUnique` — see
+ `docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md` for how
+ each was found and why none were solved with an additional data source.
```

- [ ] **Step 2: Update `CLAUDE.md`'s "Prefer a cached, statically-refreshed source" Hard rule**

```markdown
- **Prefer a cached, statically-refreshed source over a live per-request ESI call**: any tool
  needing a resource available from FC's SDE or from EVE Ref's `reference-data`/`esi-scrape`
  bulk exports must read it from `sde/`, never call ESI for it directly. Performance and
- API-citizenship requirement, not a suggestion. `sde/`'s bbolt cache is bulk-seeded once per
- process lifetime from three sources: FC's own SDE zip (sync-fatal — a failure here fails the
- whole sync), and EVE Ref's `reference-data` and `esi-scrape` datasets (each independently
- best-effort — a fetch or parse failure there logs a warning and skips that source's
- augmentation, never fails the rest of the sync). 24 of the Universe tag's 30 tools are
+ API-citizenship requirement, not a suggestion. `sde/` is backed by a single downloaded file —
+ fuzzwork.co.uk's SQLite conversion of FC's SDE, ETag-checked and re-downloaded once per process
+ lifetime on change (sync-fatal on failure) — queried directly via SQL, no intermediate
+ key-value store. Three fields with no source in that file (type `PackagedVolume`, race
+ `AllianceID`, graphic `SofDNA`) still come from a narrow, in-memory-only fetch against EVE Ref's
+ `reference-data`/`esi-scrape` datasets (each independently best-effort — a fetch or parse
+ failure there logs a warning and skips that field's augmentation, never fails the rest of the
+ sync). 24 of the Universe tag's 30 tools are
  `sde/`-backed via a read-through cache: try `sde/` first, fall back to a live ESI call on a
- cache miss, write the result back so the next lookup of that ID is fast. The other 6
+ cache miss, write the result back (into a process-lifetime in-memory overlay, not the downloaded
+ file, which is treated as read-only) so the next lookup of that ID is fast. The other 6
  (`list_universe_structures`/`get_universe_structure`, `list_universe_system_jumps`/`_kills`,
  `resolve_universe_ids`/`_names`) are permanently ESI-direct by design -- they resolve live,
  non-cacheable data (structures are player-owned and access-gated; jumps/kills are
  moment-in-time activity counters; the resolve tools take arbitrary caller-supplied
- names/IDs), not an SDE sourcing gap. Exactly one field has no source anywhere -- not FC's
- SDE, not EVE Ref's `esi-scrape` mirror of ESI's own live responses either:
- `get_universe_graphic`'s `CollisionFile`, confirmed absent from every record in both. It is
- populated on a cache miss (fresh off ESI, in case a future record ever has one) but dropped on
- a cache hit — the sole surviving field drop in the design spec's field-drop table, documented
- at its point of implementation in `mcp/universe.go`.
+ names/IDs), not an SDE sourcing gap. Four fields have no source anywhere in the fuzzwork
+ dump: `get_universe_graphic`'s `CollisionFile` (also absent from EVE Ref's `esi-scrape` mirror
+ of ESI's own live responses -- unaffected by the sqlite migration), `get_universe_region`'s
+ `Description` (`mapRegions` has no description column at all -- a genuine regression from the
+ prior bbolt design, which sourced it from FC's own jsonl), `get_universe_station`'s `Services`
+ (`staOperationServices` has zero rows in the dump), and `list_universe_factions`'s `IsUnique`
+ (`chrFactions` has no `uniqueName`-equivalent column). Each is populated on a cache miss (fresh
+ off ESI) but not on a cache hit -- documented in
+ `docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md`'s field-drop findings.
```

- [ ] **Step 3: Update `README.md`'s architecture summary sentence**

```markdown
Foundation phase. CLI skeleton (`config/`, `creds/`), the ESI OAuth2/PKCE character-auth flow
(`urlscheme/`, `sso/`), the MCP server with its character-management tools (`mcp/`, `neat mcp`),
real ESI-backed domain tools (`esi/`) covering all 36 ESI resource tags (Wallet through Wars), and
- a local SDE cache (`sde/`, combining FC's SDE with EVE Ref's `reference-data`/`esi-scrape`
- exports and backing 24 of the Universe tag's 30 tools via a read-through cache) are built. See
+ a local SDE cache (`sde/`, backed by fuzzwork.co.uk's directly-queried SQLite conversion of
+ FC's SDE, narrowly supplemented from EVE Ref's `reference-data`/`esi-scrape` exports, and
+ backing 24 of the Universe tag's 30 tools via a read-through cache) are built. See
`docs/reference/esi-coverage-roadmap.md` for the full per-tag ESI breakdown.
```

- [ ] **Step 4: Update `README.md`'s `sde/` package-table row**

Apply the identical diff from Step 1 (`CLAUDE.md`'s package-table row) to `README.md`'s matching row — the two are kept in sync verbatim per this repo's "Keeping docs & Claude resources in sync" convention.

- [ ] **Step 5: Add a superseded-by pointer to each of the three prior `sde/` specs**

None of these three files currently has a `**Status:**` header line (unlike newer specs) — add one directly below the title, the smallest possible touch that doesn't rewrite the historical record:

```markdown
# `sde/` Package Infrastructure — Design

+ **Superseded by:** [`2026-08-04-sde-sqlite-migration-design.md`](2026-08-04-sde-sqlite-migration-design.md) — this plan's bbolt + FC-jsonl-zip pipeline was replaced by a directly-queried SQLite dump. This document remains an accurate record of what was built and why at the time.

## Goal
```

Apply the same two-line insertion (adjusting only the `# Title` line to match each file) to `2026-07-27-sde-everef-supplemental-sources-design.md` and `2026-07-27-sde-universe-migration-design.md`.

- [ ] **Step 6: Final sanity check**

Run: `grep -rn "go.etcd.io/bbolt\|sde\.db\b" --include="*.md" .`
Expected: no output outside the three just-updated historical specs (which correctly still describe the old design as history) — confirms no other doc still references the removed dependency or the old `sde.db` filename as if it were current.

- [ ] **Step 7: Commit**

```bash
git add CLAUDE.md README.md docs/superpowers/specs/2026-07-27-sde-design.md docs/superpowers/specs/2026-07-27-sde-everef-supplemental-sources-design.md docs/superpowers/specs/2026-07-27-sde-universe-migration-design.md
git commit -m "docs: sync CLAUDE.md/README.md to the sqlite-backed sde/ design

Package-table rows and the SDE-sourcing Hard rule now describe the
fuzzwork dump + in-memory overlay design instead of bbolt + FC-jsonl +
EVE Ref, including the two new field-drops (Region.Description,
Station.Services, Faction.IsUnique) this migration introduced. The
three prior sde/ specs get a one-line superseded-by pointer, per this
repo's convention of not rewriting historical design docs."
```

---

## Plan-level self-review notes

- **Spec coverage:** every section of `docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md` maps to a task — storage engine (Tasks 1, 3–8), fetch/freshness (Task 2), write-back overlay (Tasks 1, 3–8), struct changes (Tasks 3–8), what EVE Ref still supplies (Tasks 2, 3, 8), testing (every task's Steps), and docs (Task 11). The three field-drops found while writing this plan (`Region.Description`, `Station.Services`, `Faction.IsUnique`) are each called out in their owning task (5, 6, 8) and in Task 11's doc updates — none were left as a silent behavior change.
- **No placeholders:** every task's code steps contain complete, compilable Go — no "similar to Task N" references; each resource file's full rewrite is written out even where the shape closely mirrors a sibling file, per the writing-plans skill's explicit prohibition on that shortcut.
- **Type consistency:** `Type`/`Group`/`Category`/`Region`/`Constellation`/`System`/`Station`/`Planet`/`Moon`/`AsteroidBelt`/`Star`/`Stargate`/`Faction`/`Race`/`Bloodline`/`Ancestry`/`Graphic` are each defined exactly once (in their owning task) and referenced with identical field names/types in every later task that touches them (Task 9's `mcp/` edits, in particular, were cross-checked field-by-field against Tasks 3–8's struct definitions rather than assumed).
