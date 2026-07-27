package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Category is a minimal decode of an SDE category record -- the top level
// of EVE's item hierarchy (e.g. "Ship", "Module").
type Category struct {
	ID        int64
	Name      map[string]string
	Published bool
}

// Category looks up a category by ID. ok is false if the ID isn't in the
// cache -- either because the store has never synced successfully, or
// because no such category exists.
func (s *Store) Category(ctx context.Context, userAgent string, id int64) (Category, bool, error) {
	_ = s.ensureSynced(ctx, userAgent) // best-effort; see ensureSynced's doc comment

	var (
		out Category
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("categories"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw struct {
			Key       int64             `json:"_key"`
			Name      map[string]string `json:"name"`
			Published bool              `json:"published"`
		}
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached category %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached category %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Category{ID: raw.Key, Name: raw.Name, Published: raw.Published}, true
		return nil
	})
	return out, ok, err
}
