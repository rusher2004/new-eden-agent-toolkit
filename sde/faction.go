package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Faction is a minimal decode of an SDE faction record. StationCount and
// StationSystemCount are not present on the raw factions.jsonl record at
// all -- sync.go's writer computes them via a 3-hop join (npcStations ->
// npcCorporations -> aggregate per faction) and injects them before
// writing.
type Faction struct {
	ID                   int64
	Name                 map[string]string
	Description          map[string]string
	IsUnique             bool
	SizeFactor           float64
	CorporationID        *int64
	MilitiaCorporationID *int64
	SolarSystemID        *int64
	StationCount         int64
	StationSystemCount   int64
}

type factionRecord struct {
	Key                  int64             `json:"_key"`
	Name                 map[string]string `json:"name"`
	Description          map[string]string `json:"description"`
	IsUnique             bool              `json:"uniqueName"`
	SizeFactor           float64           `json:"sizeFactor"`
	CorporationID        *int64            `json:"corporationID,omitempty"`
	MilitiaCorporationID *int64            `json:"militiaCorporationID,omitempty"`
	SolarSystemID        *int64            `json:"solarSystemID,omitempty"`
	StationCount         int64             `json:"stationCount"`
	StationSystemCount   int64             `json:"stationSystemCount"`
}

// Factions returns every cached faction. Returns an empty slice, not an
// error, if the store has never synced successfully -- mcp/'s caller
// treats an empty result as a signal to read-through to a live ESI list
// call.
func (s *Store) Factions(ctx context.Context, userAgent string) ([]Faction, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var out []Faction
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("factions"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw factionRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached faction: %w", err)
			}
			out = append(out, Faction{
				ID: raw.Key, Name: raw.Name, Description: raw.Description, IsUnique: raw.IsUnique,
				SizeFactor: raw.SizeFactor, CorporationID: raw.CorporationID, MilitiaCorporationID: raw.MilitiaCorporationID,
				SolarSystemID: raw.SolarSystemID, StationCount: raw.StationCount, StationSystemCount: raw.StationSystemCount,
			})
			return nil
		})
	})
	return out, err
}

// Faction looks up a faction by ID. ok is false if the ID isn't in the cache.
func (s *Store) Faction(ctx context.Context, userAgent string, id int64) (Faction, bool, error) {
	if s == nil {
		return Faction{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Faction
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("factions"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw factionRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached faction %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached faction %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Faction{
			ID: raw.Key, Name: raw.Name, Description: raw.Description, IsUnique: raw.IsUnique,
			SizeFactor: raw.SizeFactor, CorporationID: raw.CorporationID, MilitiaCorporationID: raw.MilitiaCorporationID,
			SolarSystemID: raw.SolarSystemID, StationCount: raw.StationCount, StationSystemCount: raw.StationSystemCount,
		}, true
		return nil
	})
	return out, ok, err
}
