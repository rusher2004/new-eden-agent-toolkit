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
	TypeIDs    []int64
}

type groupRecord struct {
	Key        int64             `json:"_key"`
	CategoryID int64             `json:"categoryID"`
	Name       map[string]string `json:"name"`
	Published  bool              `json:"published"`
	TypeIDs    []int64           `json:"typeIds"`
}

// Group looks up a group by ID. ok is false if the ID isn't in the cache.
func (s *Store) Group(ctx context.Context, userAgent string, id int64) (Group, bool, error) {
	if s == nil {
		return Group{}, false, nil
	}
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
		var raw groupRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached group %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached group %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Group{ID: raw.Key, CategoryID: raw.CategoryID, Name: raw.Name, Published: raw.Published, TypeIDs: raw.TypeIDs}, true
		return nil
	})
	return out, ok, err
}

// PutGroup writes g into the cache, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGroup(g Group) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(groupRecord{Key: g.ID, CategoryID: g.CategoryID, Name: g.Name, Published: g.Published, TypeIDs: g.TypeIDs})
	if err != nil {
		return fmt.Errorf("marshal group %d: %w", g.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("groups"))
		if err != nil {
			return fmt.Errorf("create groups bucket: %w", err)
		}
		return b.Put(int64Key(g.ID), raw)
	})
}

// GroupIDs returns every group ID currently cached. See CategoryIDs for why
// no separate sort is needed and what an empty result means.
func (s *Store) GroupIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("groups"))
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
