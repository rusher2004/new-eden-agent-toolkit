package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Moon is a minimal decode of an EVE Ref esi-scrape moon record. Sourced
// entirely from esi-scrape, same as AsteroidBelt -- see its doc comment.
type Moon struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

type moonRecord struct {
	Key      int64    `json:"_key"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

// Moon looks up a moon by ID. ok is false if the ID isn't in the cache.
func (s *Store) Moon(ctx context.Context, userAgent string, id int64) (Moon, bool, error) {
	if s == nil {
		return Moon{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Moon
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("moons"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw moonRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached moon %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached moon %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Moon{ID: raw.Key, Name: raw.Name, Position: raw.Position, SystemID: raw.SystemID}, true
		return nil
	})
	return out, ok, err
}

// PutMoon writes m into the cache, keyed by m.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutMoon(m Moon) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(moonRecord{Key: m.ID, Name: m.Name, Position: m.Position, SystemID: m.SystemID})
	if err != nil {
		return fmt.Errorf("marshal moon %d: %w", m.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("moons"))
		if err != nil {
			return fmt.Errorf("create moons bucket: %w", err)
		}
		return b.Put(int64Key(m.ID), raw)
	})
}
