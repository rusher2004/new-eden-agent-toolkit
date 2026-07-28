package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Planet is a minimal decode of an EVE Ref esi-scrape planet record.
// Sourced entirely from esi-scrape, same as AsteroidBelt -- see its doc
// comment.
type Planet struct {
	ID       int64
	Name     string
	TypeID   int64
	Position Position
	SystemID int64
}

type planetRecord struct {
	Key      int64    `json:"_key"`
	Name     string   `json:"name"`
	TypeID   int64    `json:"typeId"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

// Planet looks up a planet by ID. ok is false if the ID isn't in the cache.
func (s *Store) Planet(ctx context.Context, userAgent string, id int64) (Planet, bool, error) {
	if s == nil {
		return Planet{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Planet
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("planets"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw planetRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached planet %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached planet %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Planet{ID: raw.Key, Name: raw.Name, TypeID: raw.TypeID, Position: raw.Position, SystemID: raw.SystemID}, true
		return nil
	})
	return out, ok, err
}

// PutPlanet writes p into the cache, keyed by p.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutPlanet(p Planet) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(planetRecord{Key: p.ID, Name: p.Name, TypeID: p.TypeID, Position: p.Position, SystemID: p.SystemID})
	if err != nil {
		return fmt.Errorf("marshal planet %d: %w", p.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("planets"))
		if err != nil {
			return fmt.Errorf("create planets bucket: %w", err)
		}
		return b.Put(int64Key(p.ID), raw)
	})
}
