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
	if s == nil {
		return System{}, false, nil
	}
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
	if s == nil {
		return nil
	}
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

// SystemIDs returns every solar system ID currently cached. See
// sde/category.go's CategoryIDs for why no separate sort is needed and
// what an empty result means.
func (s *Store) SystemIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
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
