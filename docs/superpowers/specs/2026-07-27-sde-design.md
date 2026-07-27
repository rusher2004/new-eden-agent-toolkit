# `sde/` Package Infrastructure — Design

## Goal

Build the `sde/` package the foundation design spec called for and deliberately deferred: a local
cache of CCP's Static Data Export (SDE), backed by `go.etcd.io/bbolt`, that turns the bare IDs
every ESI tool returns into actual names. This plan builds the cache **infrastructure only** —
downloader, version tracking, typed lookups for three resource categories (types, groups,
categories). Wiring any `mcp/` tool to actually call into `sde/` (starting with the Universe tag's
types/groups/categories tools, which call ESI directly today as a documented temporary exception
in `CLAUDE.md`) is a separate follow-up plan, mirroring how the ESI coverage sweep built `esi/`
first and wired `mcp/` tag-by-tag afterward.

## Key finding: the 2026-07-23 design's assumptions still hold, with specifics now verified

The foundation spec assumed EVE Ref's `data.everef.net/ccp/sde/` serves a versioned JSONL zip with
an `index.json` for change detection, loaded into `bbolt` keyed by ID. Verified directly against
the live source before finalizing this plan (not re-derived from the two-year-old doc):

- `index.json` lists each file with `name`, `url`, `size`, `last_modified`, and `etag` — the
  `eve-online-static-data-latest-jsonl.zip` entry's `etag` is exactly the version signal the
  original design wanted.
- The zip (~94 MB compressed) contains 79 files at its **root** — no subdirectory nesting.
  `types.jsonl`, `groups.jsonl`, and `categories.jsonl` are all present.
- Every record is one JSON object per line, keyed by an integer `_key` field, e.g.:
  ```json
  {"_key": 0, "groupID": 0, "mass": 1.0, "name": {"de": "#System", "en": "#System", ...}, "portionSize": 1, "published": false}
  ```
  `name` (and, where present, `description`) is a multilingual object — `en` required, 7 other
  locales optional.
- Uncompressed sizes vary hugely: `categories.jsonl` ~11 KB, `groups.jsonl` ~807 KB,
  `types.jsonl` ~151 MB. The loader must stream `types.jsonl` line-by-line rather than buffer the
  whole decompressed file.

## Download strategy: in-memory, not temp-file

The whole compressed zip is downloaded into memory (`bytes.Reader`) rather than to a temp file on
disk. `archive/zip.NewReader` just needs an `io.ReaderAt` + size, which a `bytes.Reader` satisfies
directly — no temp-file lifecycle or cleanup-on-error path to write, and trivially mockable in
tests via `httptest.Server`. The transient ~94 MB allocation is a non-issue for a lazy, once-per-
process sync.

## `sde/store.go` (new)

```go
package sde

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.etcd.io/bbolt"
)

// Store is a local cache of CCP's Static Data Export, backed by bbolt.
// Opening a Store does not sync it — the first lookup call does that
// lazily, at most once per Store's lifetime.
type Store struct {
	db *bbolt.DB

	syncOnce sync.Once
	syncErr  error
}

// Open opens (creating if necessary) the bbolt database at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create sde cache directory: %w", err)
	}
	db, err := bbolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("open sde cache: %w", err)
	}
	return &Store{db: db}, nil
}

// DefaultPath returns ~/.neat/sde/sde.db.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".neat", "sde", "sde.db"), nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
```

Bucket names: `categories`, `groups`, `types` (record buckets, one JSONL line's raw bytes per
key), plus `_meta` (a single `version` key holding the last-synced `etag`). All four buckets are
written in one `db.Update` transaction during sync, so the version and the data it describes
always move together — a failed or interrupted sync can never leave a persisted version that
doesn't match what's actually in the record buckets.

## `sde/sync.go` (new)

```go
package sde

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.etcd.io/bbolt"
)

const indexURL = "https://data.everef.net/ccp/sde/index.json"

var syncedFiles = []string{"categories.jsonl", "groups.jsonl", "types.jsonl"}

// ensureSynced checks the store's cached SDE version against EVE Ref's live
// index exactly once per Store lifetime (via s.syncOnce), downloading and
// reloading categories/groups/types if it's stale or missing. A failed sync
// is cached too -- lookups fall back to whatever's already in bbolt (empty,
// on a never-synced install) rather than retrying every call. SDE data is
// enrichment, not a hard requirement for any ESI tool to function.
func (s *Store) ensureSynced(ctx context.Context, userAgent string) error {
	s.syncOnce.Do(func() {
		s.syncErr = s.sync(ctx, userAgent)
	})
	return s.syncErr
}

func (s *Store) sync(ctx context.Context, userAgent string) error {
	remoteEtag, zipURL, err := latestZipInfo(ctx, userAgent)
	if err != nil {
		return fmt.Errorf("fetch sde index: %w", err)
	}

	currentVersion, err := s.version()
	if err != nil {
		return fmt.Errorf("read cached sde version: %w", err)
	}
	if currentVersion == remoteEtag {
		return nil
	}

	body, err := fetch(ctx, zipURL, userAgent)
	if err != nil {
		return fmt.Errorf("download sde archive: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("open sde archive: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, name := range syncedFiles {
			bucket := name[:len(name)-len(".jsonl")]
			if err := loadResource(tx, zr, name, bucket); err != nil {
				return fmt.Errorf("load %s: %w", name, err)
			}
		}
		meta, err := tx.CreateBucketIfNotExists([]byte("_meta"))
		if err != nil {
			return err
		}
		return meta.Put([]byte("version"), []byte(remoteEtag))
	})
}

// loadResource clears and rewrites bucket from name's JSONL lines within the
// zip, keyed by each record's _key field.
func loadResource(tx *bbolt.Tx, zr *zip.Reader, name, bucket string) error {
	f, err := zr.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tx.DeleteBucket([]byte(bucket)); err != nil && err != bbolt.ErrBucketNotFound {
		return err
	}
	b, err := tx.CreateBucket([]byte(bucket))
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // types.jsonl has long lines
	for scanner.Scan() {
		line := scanner.Bytes()
		var keyed struct {
			Key int64 `json:"_key"`
		}
		if err := json.Unmarshal(line, &keyed); err != nil {
			return fmt.Errorf("parse record: %w", err)
		}
		if err := b.Put(int64Key(keyed.Key), append([]byte(nil), line...)); err != nil {
			return err
		}
	}
	return scanner.Err()
}
```

`int64Key`/`fetch`/`latestZipInfo` are small helpers (big-endian encode the ID for the bbolt key;
plain `net/http` GETs carrying `sso.UserAgent(version)`, matching every other outbound request this
project makes to CCP — good citizenship toward EVE Ref too).

## `sde/category.go`, `sde/group.go`, `sde/type.go` (new)

One file per resource, each a minimal record struct plus its accessor. Full raw JSONL bytes are
what's stored in bbolt (not a pre-shrunk struct) — the accessor decodes only the fields it exposes
today, so a later plan needing an extra field (`basePrice`, `iconID`, ...) just adds a field to the
struct and its decode, no resync required. `Name` is `map[string]string` (locale code → string)
because the source data already carries every locale for free — English is simply the only one any
tool reads today (`Name["en"]`), not a technical ceiling.

```go
package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

type Category struct {
	ID        int64
	Name      map[string]string
	Published bool
}

// Category looks up a category by ID. ok is false if the ID isn't in the
// cache (unsynced install, or an ID that genuinely doesn't exist).
func (s *Store) Category(ctx context.Context, userAgent string, id int64) (Category, bool, error) {
	_ = s.ensureSynced(ctx, userAgent) // best-effort; see ensureSynced doc
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
		var raw struct {
			Key       int64             `json:"_key"`
			Name      map[string]string `json:"name"`
			Published bool              `json:"published"`
		}
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached category %d: %w", id, err)
		}
		out, ok = Category{ID: raw.Key, Name: raw.Name, Published: raw.Published}, true
		return nil
	})
	return out, ok, err
}
```

`Group` adds `CategoryID int64` (from `categoryID`); `Type` adds `GroupID int64` (from `groupID`).
Same shape otherwise.

## Testing

`sde/sync_test.go` (external `sde_test` package, no real network):
- `httptest.Server` serving a fixture `index.json` (one entry, a fixed `etag`) and a fixture zip
  built at test time via `archive/zip.Writer`, containing a handful of representative
  `categories.jsonl`/`groups.jsonl`/`types.jsonl` lines (including a multi-locale `name` to prove
  the map round-trips, and one `published: false` record to prove that field isn't dropped).
- First sync against a fresh `Store` (temp-dir `sde.db`) loads all three buckets; a second sync
  with the same `etag` makes zero additional HTTP requests (assert via a request-counting
  `http.RoundTripper` wrapping the test server's client).
- A changed `etag` on the second sync triggers a real reload; a *third* Store pointed at the same
  already-populated `sde.db` file, given an `index.json` that 500s, still returns the previously
  cached records (`ensureSynced`'s error doesn't block reads).
- `sync.Once` semantics: two lookups against the same `Store` instance only ever call the index
  endpoint once, confirmed via the request-counting transport.

`sde/category_test.go` / `group_test.go` / `type_test.go` (internal `sde` package, since they need
the private `ensureSynced` bypass — pre-seed the bucket directly via `db.Update` in the test,
skipping `sync` entirely, so these are pure decode/accessor tests independent of the network path):
- Known ID → correct struct, all locales present in `Name`.
- Unknown ID → `ok == false`, no error.
- Malformed cached JSON (simulated corruption) → wrapped decode error.

## Error handling

Consistent with the rest of the codebase: `fmt.Errorf("...: %w", err)` wrapping at every layer.
`ensureSynced`'s error is deliberately *not* returned from the public accessors — sync failures are
swallowed at that boundary and only surface via the lookup returning `ok == false` against an empty
or stale cache, per the "enrichment, not a hard requirement" decision. Genuine data-corruption
errors (a cached record that fails to unmarshal) still propagate as real errors from the accessor,
since that's not a "haven't synced yet" case — it's a cache that's actually broken.

## `config.Config.SDEVersion`

This field (added speculatively during the foundation phase, never wired to anything) becomes
fully superseded by the `_meta` bucket inside `sde.db` — this plan removes it from
`config/config.go` and its test rather than leave known-dead code in a package this plan doesn't
otherwise touch.

## File layout summary

- Create: `sde/store.go`, `sde/sync.go`, `sde/sync_test.go`, `sde/category.go`,
  `sde/category_test.go`, `sde/group.go`, `sde/group_test.go`, `sde/type.go`, `sde/type_test.go`
- Modify: `config/config.go`, `config/config_test.go` (remove `SDEVersion`)
- Modify: `go.mod`/`go.sum` (add `go.etcd.io/bbolt`)
- Modify: `CLAUDE.md`/`README.md` (new `sde/` package row, architecture table)

## Explicitly out of scope for this plan

- Wiring any `mcp/` or `esi/` tool to actually call into `sde/` — including the Universe tag's
  types/groups/categories tools, which stay on their documented temporary direct-ESI exception
  until a dedicated follow-up plan migrates them.
- Any SDE resource category beyond types/groups/categories — the other ~30 categories are added
  only once a specific tool needs them, per the foundation design's original stance.
- `neat setup` changes of any kind — the sync trigger is fully internal to `sde/`'s lazy,
  per-`Store`-lifetime check.
- Retry/backoff on sync failure, partial-sync recovery beyond the single-transaction atomicity
  already described, and any SDE resource beyond the three named above.
