package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Region is a minimal decode of an SDE region record -- the top level of
// EVE's geography (e.g. "Derelik").
type Region struct {
	ID               int64
	Name             map[string]string
	Description      map[string]string
	ConstellationIDs []int64
}

type regionRecord struct {
	Key              int64             `json:"_key"`
	Name             map[string]string `json:"name"`
	Description      map[string]string `json:"description,omitempty"`
	ConstellationIDs []int64           `json:"constellationIDs"`
}

// Region looks up a region by ID. ok is false if the ID isn't in the cache.
func (s *Store) Region(ctx context.Context, userAgent string, id int64) (Region, bool, error) {
	if s == nil {
		return Region{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Region
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("regions"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw regionRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached region %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached region %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Region{ID: raw.Key, Name: raw.Name, Description: raw.Description, ConstellationIDs: raw.ConstellationIDs}, true
		return nil
	})
	return out, ok, err
}

// PutRegion writes r into the cache, keyed by r.ID -- used by mcp/'s
// read-through fallback to cache a region fetched live from ESI.
func (s *Store) PutRegion(r Region) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(regionRecord{Key: r.ID, Name: r.Name, Description: r.Description, ConstellationIDs: r.ConstellationIDs})
	if err != nil {
		return fmt.Errorf("marshal region %d: %w", r.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("regions"))
		if err != nil {
			return fmt.Errorf("create regions bucket: %w", err)
		}
		return b.Put(int64Key(r.ID), raw)
	})
}

// RegionIDs returns every region ID currently cached. See sde/category.go's
// CategoryIDs for why no separate sort is needed and what an empty result
// means.
func (s *Store) RegionIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("regions"))
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
