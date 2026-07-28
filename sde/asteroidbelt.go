package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// AsteroidBelt is a minimal decode of an EVE Ref esi-scrape asteroid belt
// record. Unlike every other sde/ resource, there is no CCP SDE source for
// this data at all -- asteroid belt names are computed server-side by CCP,
// never stored in any static file. This is sourced entirely from EVE Ref's
// esi-scrape dataset (a mirror of ESI's own live responses), seeded in bulk
// at sync time; a read-through miss still falls back to a live ESI call.
type AsteroidBelt struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

type asteroidBeltRecord struct {
	Key      int64    `json:"_key"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

// AsteroidBelt looks up an asteroid belt by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) AsteroidBelt(ctx context.Context, userAgent string, id int64) (AsteroidBelt, bool, error) {
	if s == nil {
		return AsteroidBelt{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out AsteroidBelt
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("asteroidBelts"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw asteroidBeltRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached asteroid belt %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached asteroid belt %d has mismatched key %d", id, raw.Key)
		}
		out, ok = AsteroidBelt{ID: raw.Key, Name: raw.Name, Position: raw.Position, SystemID: raw.SystemID}, true
		return nil
	})
	return out, ok, err
}

// PutAsteroidBelt writes b into the cache, keyed by b.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutAsteroidBelt(b AsteroidBelt) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(asteroidBeltRecord{Key: b.ID, Name: b.Name, Position: b.Position, SystemID: b.SystemID})
	if err != nil {
		return fmt.Errorf("marshal asteroid belt %d: %w", b.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte("asteroidBelts"))
		if err != nil {
			return fmt.Errorf("create asteroidBelts bucket: %w", err)
		}
		return bkt.Put(int64Key(b.ID), raw)
	})
}
