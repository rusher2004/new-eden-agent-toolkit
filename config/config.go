// Package config manages neat's non-secret user state, stored at
// ~/.neat/config.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Character identifies a known EVE character by ID and display name, with
// what the user has told neat about it.
type Character struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
	// Account is a label the user chose for the EVE account the character
	// is on: characters with the same label share an account, so they can't
	// be logged in at the same time. Empty means unknown, not "on an
	// account of its own". ESI has no way to tell, so only the user can set it.
	Account string `json:"account,omitempty"`
	// Notes is free text about the character: what it's for, what it's
	// trained in, anything a plan should know.
	Notes string `json:"notes,omitempty"`
}

// Config is neat's persisted, non-secret state.
type Config struct {
	Characters       []Character       `json:"characters"`
	FacilityProfiles []FacilityProfile `json:"facilityProfiles,omitempty"`
	HaulProfiles     []HaulProfile     `json:"haulProfiles,omitempty"`
	HubProfiles      []HubProfile      `json:"hubProfiles,omitempty"`
	// AnalyticsDisabled turns off neat's anonymous usage statistics (see
	// analytics/). The zero value leaves it on; `neat analytics off` sets it.
	AnalyticsDisabled bool `json:"analyticsDisabled,omitempty"`
}

// Store reads and writes a Config to a fixed path on disk.
type Store struct {
	path string
	// mu serializes writes, and Update's whole read-modify-write, so two
	// writers in one process (a login page request and a tool call, say)
	// neither lose each other's change nor share the temp file at once.
	mu sync.Mutex
}

// NewStore resolves the default config path, ~/.neat/config.json.
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	return NewStoreAt(filepath.Join(home, ".neat", "config.json")), nil
}

// NewStoreAt returns a Store backed by an explicit path, for tests and
// callers that need to override the location.
func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

// Path returns the file path this Store reads from and writes to.
func (s *Store) Path() string {
	return s.path
}

// Load reads the Config from disk. A missing file is not an error — it
// returns a zero-value Config, so first-run has nothing special to handle.
func (s *Store) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", s.path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", s.path, err)
	}
	return cfg, nil
}

// Save writes cfg to disk atomically (temp file + rename), creating parent
// directories as needed.
func (s *Store) Save(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(cfg)
}

// Update loads the Config, applies change, and saves the result, holding
// the Store's lock throughout so no other Update or Save interleaves. When
// change returns an error, nothing is saved.
func (s *Store) Update(change func(*Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	if err := change(&cfg); err != nil {
		return err
	}
	return s.save(cfg)
}

func (s *Store) save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename temp config into place: %w", err)
	}
	return nil
}
