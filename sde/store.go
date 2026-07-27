// Package sde caches CCP's Static Data Export locally in a bbolt database,
// so tools can resolve ESI's bare IDs into names without hitting ESI's own
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

// Store is a local cache of CCP's Static Data Export, backed by bbolt.
// Opening a Store does not sync it -- the first lookup call does that
// lazily, at most once per Store's lifetime.
type Store struct {
	db       *bbolt.DB
	indexURL string

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
	return &Store{db: db, indexURL: defaultIndexURL}, nil
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
