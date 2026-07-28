package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Station is a minimal decode of an EVE Ref esi-scrape NPC station record.
// Sourced entirely from esi-scrape, same as AsteroidBelt -- see its doc
// comment. This is the same NPC-station concept sde/sync.go already reads a
// slim (_key/solarSystemID/ownerID only) pass of from the official SDE's
// npcStations.jsonl for System.StationIDs/Faction.StationCount -- this
// resource is the full per-station detail those reverse indices point at.
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

type stationRecord struct {
	Key                      int64    `json:"_key"`
	Name                     string   `json:"name"`
	SystemID                 int64    `json:"systemId"`
	TypeID                   int64    `json:"typeId"`
	Position                 Position `json:"position"`
	MaxDockableShipVolume    float64  `json:"maxDockableShipVolume"`
	OfficeRentalCost         float64  `json:"officeRentalCost"`
	Owner                    *int64   `json:"owner,omitempty"`
	RaceID                   *int64   `json:"raceId,omitempty"`
	ReprocessingEfficiency   float64  `json:"reprocessingEfficiency"`
	ReprocessingStationsTake float64  `json:"reprocessingStationsTake"`
	Services                 []string `json:"services"`
}

// Station looks up a station by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Station(ctx context.Context, userAgent string, id int64) (Station, bool, error) {
	if s == nil {
		return Station{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Station
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("stations"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw stationRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached station %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached station %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Station{
			ID: raw.Key, Name: raw.Name, SystemID: raw.SystemID, TypeID: raw.TypeID, Position: raw.Position,
			MaxDockableShipVolume: raw.MaxDockableShipVolume, OfficeRentalCost: raw.OfficeRentalCost,
			Owner: raw.Owner, RaceID: raw.RaceID, ReprocessingEfficiency: raw.ReprocessingEfficiency,
			ReprocessingStationsTake: raw.ReprocessingStationsTake, Services: raw.Services,
		}, true
		return nil
	})
	return out, ok, err
}

// PutStation writes st into the cache, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStation(st Station) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(stationRecord{
		Key: st.ID, Name: st.Name, SystemID: st.SystemID, TypeID: st.TypeID, Position: st.Position,
		MaxDockableShipVolume: st.MaxDockableShipVolume, OfficeRentalCost: st.OfficeRentalCost,
		Owner: st.Owner, RaceID: st.RaceID, ReprocessingEfficiency: st.ReprocessingEfficiency,
		ReprocessingStationsTake: st.ReprocessingStationsTake, Services: st.Services,
	})
	if err != nil {
		return fmt.Errorf("marshal station %d: %w", st.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("stations"))
		if err != nil {
			return fmt.Errorf("create stations bucket: %w", err)
		}
		return b.Put(int64Key(st.ID), raw)
	})
}
