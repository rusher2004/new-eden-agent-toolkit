// Package sde caches CCP's Static Data Export -- plus EVE Ref's
// reference-data and esi-scrape bulk exports, which augment a handful of
// fields the SDE itself doesn't carry -- locally in a bbolt database, so
// tools can resolve ESI's bare IDs into names without hitting ESI's own
// (much more limited, rate-limited) name-resolution endpoints.
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

// Store is a local cache of CCP's Static Data Export plus EVE Ref's
// reference-data/esi-scrape bulk exports, backed by bbolt. Opening a Store
// does not sync it -- the first lookup call does that lazily, at most once
// per Store's lifetime. A nil *Store is valid to call any exported method
// on: every accessor treats it exactly like a never-synced cache (a getter
// returns its zero value with ok=false, a Put* is a silent no-op) so a
// caller that couldn't open the cache at all can still pass sdeStore
// through unchanged and rely on the existing cache-miss/ESI-fallback path.
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

// Close closes the underlying bbolt database. A nil *Store is a no-op --
// see Store's doc comment.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	return s.db.Close()
}
