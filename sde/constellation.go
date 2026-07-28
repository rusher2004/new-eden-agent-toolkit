package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Position is a shared 3D coordinate, reused by every geography resource.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Constellation is a minimal decode of an SDE constellation record -- EVE's
// geography level between Region and System.
type Constellation struct {
	ID        int64
	RegionID  int64
	Name      map[string]string
	Position  Position
	SystemIDs []int64
}

type constellationRecord struct {
	Key       int64             `json:"_key"`
	Name      map[string]string `json:"name"`
	Position  Position          `json:"position"`
	RegionID  int64             `json:"regionID"`
	SystemIDs []int64           `json:"solarSystemIDs"`
}

// Constellation looks up a constellation by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) Constellation(ctx context.Context, userAgent string, id int64) (Constellation, bool, error) {
	if s == nil {
		return Constellation{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Constellation
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("constellations"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw constellationRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached constellation %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached constellation %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Constellation{ID: raw.Key, RegionID: raw.RegionID, Name: raw.Name, Position: raw.Position, SystemIDs: raw.SystemIDs}, true
		return nil
	})
	return out, ok, err
}

// PutConstellation writes c into the cache, keyed by c.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutConstellation(c Constellation) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(constellationRecord{Key: c.ID, Name: c.Name, Position: c.Position, RegionID: c.RegionID, SystemIDs: c.SystemIDs})
	if err != nil {
		return fmt.Errorf("marshal constellation %d: %w", c.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("constellations"))
		if err != nil {
			return fmt.Errorf("create constellations bucket: %w", err)
		}
		return b.Put(int64Key(c.ID), raw)
	})
}

// ConstellationIDs returns every constellation ID currently cached. See
// sde/category.go's CategoryIDs for why no separate sort is needed and
// what an empty result means.
func (s *Store) ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("constellations"))
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
