// Package config manages neat's non-secret user state under ~/.neat/:
// config.json for internal state, and YAML files the user can edit too for
// characters, facility profiles, haul profiles and routes.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Character identifies a known EVE character by ID and display name, with
// what the user has told neat about it.
type Character struct {
	ID   int32  `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
	// Account is a label the user chose for the EVE account the character
	// is on: characters with the same label share an account, so they can't
	// be logged in at the same time. Empty means unknown, not "on an
	// account of its own". ESI has no way to tell, so only the user can set it.
	Account string `json:"account,omitempty" yaml:"account,omitempty"`
	// Notes is free text about the character: what it's for, what it's
	// trained in, anything a plan should know.
	Notes string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// Config is neat's persisted, non-secret state. Most of it lives in YAML
// files under ~/.neat/ that the user can edit too (see Store); config.json
// keeps only internal state.
type Config struct {
	Characters       []Character
	FacilityProfiles []FacilityProfile
	HaulProfiles     []HaulProfile
	Routes           []Route
	HubProfiles      []HubProfile
	// AnalyticsDisabled turns off neat's anonymous usage statistics (see
	// analytics/). The zero value leaves it on; `neat analytics off` sets it.
	AnalyticsDisabled bool
}

// fileConfig is config.json. Characters, facility profiles and haul profiles
// were kept here before they moved to YAML; they're read only to move them.
type fileConfig struct {
	Characters        []Character         `json:"characters,omitempty"`
	FacilityProfiles  []FacilityProfile   `json:"facilityProfiles,omitempty"`
	HaulProfiles      []legacyHaulProfile `json:"haulProfiles,omitempty"`
	HubProfiles       []HubProfile        `json:"hubProfiles,omitempty"`
	AnalyticsDisabled bool                `json:"analyticsDisabled,omitempty"`
}

// legacyHaulProfile is a haul profile as config.json held it, with its
// pinned routes inline as system IDs.
type legacyHaulProfile struct {
	HaulProfile
	Outbound []int64 `json:"outbound,omitempty"`
	Return   []int64 `json:"return,omitempty"`
}

// Store reads and writes neat's state: config.json, plus one YAML file per
// kind of saved data next to it, which the user can edit by hand. Writes
// change only the fields that differ, so the user's comments, ordering,
// notes and their own keys survive.
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
// callers that need to override the location. The YAML files sit in the
// same directory.
func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

// Path returns config.json's path.
func (s *Store) Path() string {
	return s.path
}

func (s *Store) file(name string) string { return filepath.Join(filepath.Dir(s.path), name) }

func (s *Store) characters() yamlList[Character] {
	return yamlList[Character]{path: s.file("characters.yaml"), key: func(c Character) string { return strconv.Itoa(int(c.ID)) }, header: `Characters logged in to neat. neat keeps this file up to date, and you can edit it too.
account: your label for the EVE account the character is on. Characters with the same label share an account,
  so they can't be logged in at the same time. Leave it out if you don't know.
notes: anything about the character worth knowing when planning.`}
}

func (s *Store) facilities() yamlList[FacilityProfile] {
	return yamlList[FacilityProfile]{path: s.file("facilities.yaml"), key: func(p FacilityProfile) string { return strings.ToLower(p.Name) }, header: `Industry and reprocessing facilities neat plans with. neat keeps this file up to date, and you can edit it too.
Bonuses come from EVE's static data; this file records what EVE can't tell neat: rigs, services and taxes.`}
}

func (s *Store) haulProfiles() yamlList[HaulProfile] {
	return yamlList[HaulProfile]{path: s.file("haul_profiles.yaml"), key: func(p HaulProfile) string { return strings.ToLower(p.Name) }, header: `Hauling ships, their pilots and routing preferences. neat keeps this file up to date, and you can edit it too.
outbound_route and return_route name routes in routes.yaml.`}
}

func (s *Store) routes() yamlList[Route] {
	return yamlList[Route]{path: s.file("routes.yaml"), key: func(r Route) string { return strings.ToLower(r.Name) }, header: `Saved jump routes, by name: the systems in order. neat keeps this file up to date, and you can edit it too.
notes: why the route goes the way it does, for whoever plans with it next.`}
}

// Load reads neat's state. Missing files are empty, not errors, so first run
// has nothing special to handle. The first load after an upgrade moves
// characters, facility profiles and haul profiles out of config.json into
// their YAML files.
func (s *Store) Load() (Config, error) {
	fc, err := s.loadFile()
	if err != nil {
		return Config{}, err
	}
	if len(fc.Characters)+len(fc.FacilityProfiles)+len(fc.HaulProfiles) > 0 {
		if fc, err = s.migrate(fc); err != nil {
			return Config{}, err
		}
	}
	cfg := Config{HubProfiles: fc.HubProfiles, AnalyticsDisabled: fc.AnalyticsDisabled}
	if cfg.Characters, err = s.characters().load(); err != nil {
		return Config{}, err
	}
	if cfg.FacilityProfiles, err = s.facilities().load(); err != nil {
		return Config{}, err
	}
	if cfg.HaulProfiles, err = s.haulProfiles().load(); err != nil {
		return Config{}, err
	}
	if cfg.Routes, err = s.routes().load(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *Store) loadFile() (fileConfig, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return fileConfig{}, nil
	}
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config %s: %w", s.path, err)
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return fileConfig{}, fmt.Errorf("parse config %s: %w", s.path, err)
	}
	return fc, nil
}

// migrate moves config.json's characters, facility profiles and haul
// profiles into their YAML files. A YAML file that already exists wins (a
// previous move was interrupted after writing it). config.json loses each
// section only after its YAML file is written.
func (s *Store) migrate(fc fileConfig) (fileConfig, error) {
	unlock := lockFile(s.path)
	defer unlock()
	if len(fc.Characters) > 0 {
		if err := moveTo(s.characters(), fc.Characters); err != nil {
			return fc, err
		}
	}
	if len(fc.FacilityProfiles) > 0 {
		if err := moveTo(s.facilities(), fc.FacilityProfiles); err != nil {
			return fc, err
		}
	}
	if len(fc.HaulProfiles) > 0 {
		var profiles []HaulProfile
		var routes []Route
		for _, lp := range fc.HaulProfiles {
			p := lp.HaulProfile
			if len(lp.Outbound) > 0 {
				p.OutboundRoute = p.Name + " outbound"
				routes = append(routes, legacyRoute(p.OutboundRoute, lp.Outbound))
			}
			if len(lp.Return) > 0 {
				p.ReturnRoute = p.Name + " return"
				routes = append(routes, legacyRoute(p.ReturnRoute, lp.Return))
			}
			profiles = append(profiles, p)
		}
		if len(routes) > 0 {
			if err := moveTo(s.routes(), routes); err != nil {
				return fc, err
			}
		}
		if err := moveTo(s.haulProfiles(), profiles); err != nil {
			return fc, err
		}
	}
	fc.Characters, fc.FacilityProfiles, fc.HaulProfiles = nil, nil, nil
	return fc, s.writeFile(fc)
}

func legacyRoute(name string, ids []int64) Route {
	r := Route{Name: name, Notes: "Moved from config.json, where it was pinned on the haul profile."}
	for _, id := range ids {
		r.Waypoints = append(r.Waypoints, RouteWaypoint{ID: id})
	}
	return r
}

// moveTo writes entries to a YAML file that doesn't exist yet.
func moveTo[T any](l yamlList[T], entries []T) error {
	if _, err := os.Stat(l.path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check %s: %w", l.path, err)
	}
	if _, err := l.sync(entries); err != nil {
		return fmt.Errorf("move %s out of config.json: %w", l.name(), err)
	}
	return nil
}

// updateLocks holds one lock per config.json path, so an Update's load and
// save can't interleave with another Update or Save in this process, through
// any Store for the same path.
var updateLocks sync.Map

func (s *Store) lockUpdates() func() {
	m, _ := updateLocks.LoadOrStore(s.path, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// Update loads neat's state, applies change to it and saves the result,
// holding one lock from the load to the save. Every writer goes through
// Update: a Save of a Config loaded earlier writes back that whole Config,
// undoing whatever another writer saved in between (the login page and
// parallel tool calls can both write). If change returns an error, nothing
// is saved. Update returns every value it changed in the YAML files.
func (s *Store) Update(change func(*Config) error) ([]Change, error) {
	unlock := s.lockUpdates()
	defer unlock()
	cfg, err := s.Load()
	if err != nil {
		return nil, err
	}
	if err := change(&cfg); err != nil {
		return nil, err
	}
	return s.saveChanges(cfg)
}

// Save writes cfg, changing only what differs from what's on disk. It
// replaces everything with cfg, so use Update to change state that's
// already saved.
func (s *Store) Save(cfg Config) error {
	_, err := s.SaveChanges(cfg)
	return err
}

// SaveChanges writes cfg like Save and returns every value it changed in the
// YAML files.
func (s *Store) SaveChanges(cfg Config) ([]Change, error) {
	unlock := s.lockUpdates()
	defer unlock()
	return s.saveChanges(cfg)
}

func (s *Store) saveChanges(cfg Config) ([]Change, error) {
	if err := s.saveFile(cfg); err != nil {
		return nil, err
	}
	var all []Change
	for _, sync := range []func() ([]Change, error){
		func() ([]Change, error) { return s.characters().sync(cfg.Characters) },
		func() ([]Change, error) { return s.facilities().sync(cfg.FacilityProfiles) },
		func() ([]Change, error) { return s.haulProfiles().sync(cfg.HaulProfiles) },
		func() ([]Change, error) { return s.routes().sync(cfg.Routes) },
	} {
		c, err := sync()
		if err != nil {
			return all, err
		}
		all = append(all, c...)
	}
	return all, nil
}

func (s *Store) saveFile(cfg Config) error {
	unlock := lockFile(s.path)
	defer unlock()
	fc, err := s.loadFile()
	if err != nil {
		return err
	}
	if fc.AnalyticsDisabled == cfg.AnalyticsDisabled && slices.Equal(fc.HubProfiles, cfg.HubProfiles) && len(fc.Characters)+len(fc.FacilityProfiles)+len(fc.HaulProfiles) == 0 {
		if _, err := os.Stat(s.path); err == nil {
			return nil
		}
	}
	fc = fileConfig{HubProfiles: cfg.HubProfiles, AnalyticsDisabled: cfg.AnalyticsDisabled}
	return s.writeFile(fc)
}

// writeFile writes config.json atomically (temp file + rename), creating
// parent directories as needed.
func (s *Store) writeFile(fc fileConfig) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(fc, "", "  ")
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
