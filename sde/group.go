package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Group is a minimal decode of an SDE group record -- EVE's item hierarchy
// level between Category and Type (e.g. "Frigate", within the "Ship"
// category).
type Group struct {
	ID         int64
	CategoryID int64
	Name       map[string]string
	Published  bool
}

// Group looks up a group by ID. ok is false if the ID isn't in the cache.
func (s *Store) Group(ctx context.Context, userAgent string, id int64) (Group, bool, error) {
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Group
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("groups"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw struct {
			Key        int64             `json:"_key"`
			CategoryID int64             `json:"categoryID"`
			Name       map[string]string `json:"name"`
			Published  bool              `json:"published"`
		}
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached group %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached group %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Group{ID: raw.Key, CategoryID: raw.CategoryID, Name: raw.Name, Published: raw.Published}, true
		return nil
	})
	return out, ok, err
}
