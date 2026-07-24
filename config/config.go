// Package config manages neat's non-secret user state, stored at
// ~/.neat/config.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Character identifies a known EVE character by ID and display name.
type Character struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Config is neat's persisted, non-secret state.
type Config struct {
	Characters []Character `json:"characters"`
	SDEVersion string      `json:"sdeVersion"`
}

// Store reads and writes a Config to a fixed path on disk.
type Store struct {
	path string
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
