package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Type is a minimal decode of an SDE type record -- the leaf level of
// EVE's item hierarchy (e.g. "Rifter", within the "Frigate" group).
type Type struct {
	ID        int64
	GroupID   int64
	Name      map[string]string
	Published bool
}

// Type looks up a type by ID. ok is false if the ID isn't in the cache.
func (s *Store) Type(ctx context.Context, userAgent string, id int64) (Type, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Type
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("types"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw struct {
			Key       int64             `json:"_key"`
			GroupID   int64             `json:"groupID"`
			Name      map[string]string `json:"name"`
			Published bool              `json:"published"`
		}
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached type %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached type %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Type{ID: raw.Key, GroupID: raw.GroupID, Name: raw.Name, Published: raw.Published}, true
		return nil
	})
	return out, ok, err
}
