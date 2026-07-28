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
	GroupIDs  []int64
}

type categoryRecord struct {
	Key       int64             `json:"_key"`
	Name      map[string]string `json:"name"`
	Published bool              `json:"published"`
	GroupIDs  []int64           `json:"groupIds"`
}

// Category looks up a category by ID. ok is false if the ID isn't in the
// cache -- either because the store has never synced successfully, or
// because no such category exists.
func (s *Store) Category(ctx context.Context, userAgent string, id int64) (Category, bool, error) {
	if s == nil {
		return Category{}, false, nil
	}
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
		var raw categoryRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached category %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached category %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Category{ID: raw.Key, Name: raw.Name, Published: raw.Published, GroupIDs: raw.GroupIDs}, true
		return nil
	})
	return out, ok, err
}

// PutCategory writes c into the cache, keyed by c.ID -- used by mcp/'s
// read-through fallback to cache a category fetched live from ESI.
func (s *Store) PutCategory(c Category) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(categoryRecord{Key: c.ID, Name: c.Name, Published: c.Published, GroupIDs: c.GroupIDs})
	if err != nil {
		return fmt.Errorf("marshal category %d: %w", c.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("categories"))
		if err != nil {
			return fmt.Errorf("create categories bucket: %w", err)
		}
		return b.Put(int64Key(c.ID), raw)
	})
}

// CategoryIDs returns every category ID currently cached. bbolt's ForEach
// visits keys in byte-sorted order, which is already numeric ascending
// order for int64Key's big-endian, non-negative encoding -- no separate
// sort needed. Returns an empty slice, not an error, if the store has never
// synced successfully; callers read that as "cache empty" and fall back to
// a live ESI list call, not "no categories exist".
func (s *Store) CategoryIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("categories"))
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
