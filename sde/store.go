// Package sde caches CCP's Static Data Export -- sourced from fuzzwork.co.uk's
// periodically-refreshed SQLite conversion of the SDE, queried directly via
// database/sql -- plus a narrow, in-memory-only augmentation from EVE Ref's
// bulk exports for the handful of fields the fuzzwork dump doesn't carry:
// type Radius from reference-data (fuzzwork dropped invTypes.radius; found in
// the 2026-11-18 audit), and race AllianceID/graphic SofDNA from esi-scrape.
// Type PackagedVolume used to come from reference-data too, until CCP added
// it to the SDE itself (see https://github.com/esi/esi-issues/issues/1499)
// and fuzzwork's dump picked it up as an invTypes column. See
// docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md for the
// full design and the field-drops this migration accepts.
package sde

import (
	"context"
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
// db starts nil and is only ever assigned inside sync() (see sync.go).
// sync() opens whatever's already on disk first (verified usable via a
// sentinel query, not just a bare Ping -- SQLite happily creates a valid,
// empty database for a missing or corrupt path, which Ping alone can't
// distinguish from the real thing), before ever calling out to the network,
// so a later failure (a HEAD timeout, a broken download) degrades to
// "keep serving what's on disk" rather than leaving db nil for the rest of
// the process's life. Any handle sync() replaces is closed before the
// dump's atomic-rename runs (an open handle on the destination can block
// the rename on Windows), and reopened after, never left pointing at a
// stale pre-rename inode. Since sync() runs at most once per Store's
// lifetime (via syncOnce) and every accessor calls ensureSynced before
// touching db, db is guaranteed either fully set up or (on a first-ever
// install with no network and nothing cached yet) permanently nil -- every
// accessor guards for the latter.
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
	//
	// Known gap (PR #14 review finding #4): every *IDs/list accessor
	// (RegionIDs, TypeIDs, GraphicIDs, Factions, ...) queries db only --
	// none of them consult overlay. A GetX(id) can succeed for an ID a
	// PutX just wrote back, while the matching *IDs()/list call never
	// includes it, since the two paths look at different stores. This
	// surfaces only for a genuinely new ID CCP added after the fuzzwork
	// dump's last refresh -- rare, and self-healing at the next refresh.
	// Deliberately not fixed by merging overlay entries into every *IDs/
	// list result: that would add real complexity (each accessor filtering
	// and merging a sync.Map scan) for an edge case that resolves itself.
	overlay sync.Map

	// Set once per sync(), from EVE Ref's reference-data (typeRadii) and
	// esi-scrape (raceAlliances, graphicSofDNA) bulk exports -- the three
	// fields with no source anywhere in the fuzzwork dump. Never persisted to
	// disk (unlike db): every process's single sync() re-fetches both sources
	// fresh, since there is no local copy to otherwise fall back to. nil (not
	// an error) if the corresponding fetch failed or was never attempted (e.g.
	// this Store has never synced).
	typeRadii     map[int64]float64
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
// (see typeRadii et al.'s doc comment), so there is nothing to gate a skip on
// for those two sources.
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
//
// The type assertion is checked, not bare -- every (kind, T) pairing across
// the package is correct today (verified: each resource file uses exactly
// one kind string with exactly one type), but a future kind-string typo or
// copy-paste mismatch should degrade to a cache miss, not panic the whole
// MCP server on the next lookup of that kind.
func getOverlay[T any](s *Store, kind string, id int64) (T, bool) {
	v, ok := s.overlay.Load(overlayKey(kind, id))
	if !ok {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return t, true
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

// OpenStatic opens an existing fuzzwork-format SQLite dump at path for
// reading, without ever syncing it from the network. Use it when the caller
// manages the dump file itself (and in tests). PutX write-backs still go to
// the in-memory overlay.
func OpenStatic(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sde dump %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open sde dump %s: %w", path, err)
	}
	s := &Store{dbPath: path, metaPath: filepath.Join(filepath.Dir(path), "meta.json"), db: db}
	s.syncOnce.Do(func() {})
	return s, nil
}
