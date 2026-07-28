package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Graphic is a minimal decode of an SDE graphic record, augmented at sync
// time with SofDNA from EVE Ref's esi-scrape dataset. CollisionFile is
// deliberately not modeled -- confirmed absent from every one of the
// archive's 6051 graphics.jsonl records AND from every record in the
// esi-scrape mirror of ESI's own live responses, not merely a sample
// coincidence in either source.
type Graphic struct {
	ID             int64
	GraphicFile    *string
	IconFolder     *string
	SofDNA         *string
	SofFactionName *string
	SofHullName    *string
	SofRaceName    *string
}

type graphicRecord struct {
	Key            int64   `json:"_key"`
	GraphicFile    *string `json:"graphicFile,omitempty"`
	IconFolder     *string `json:"iconFolder,omitempty"`
	SofDNA         *string `json:"sofDna,omitempty"`
	SofFactionName *string `json:"sofFactionName,omitempty"`
	SofHullName    *string `json:"sofHullName,omitempty"`
	SofRaceName    *string `json:"sofRaceName,omitempty"`
}

// Graphic looks up a graphic by ID. ok is false if the ID isn't in the cache.
func (s *Store) Graphic(ctx context.Context, userAgent string, id int64) (Graphic, bool, error) {
	if s == nil {
		return Graphic{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Graphic
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("graphics"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw graphicRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached graphic %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached graphic %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Graphic{
			ID: raw.Key, GraphicFile: raw.GraphicFile, IconFolder: raw.IconFolder, SofDNA: raw.SofDNA,
			SofFactionName: raw.SofFactionName, SofHullName: raw.SofHullName, SofRaceName: raw.SofRaceName,
		}, true
		return nil
	})
	return out, ok, err
}

// PutGraphic writes g into the cache, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGraphic(g Graphic) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(graphicRecord{
		Key: g.ID, GraphicFile: g.GraphicFile, IconFolder: g.IconFolder, SofDNA: g.SofDNA,
		SofFactionName: g.SofFactionName, SofHullName: g.SofHullName, SofRaceName: g.SofRaceName,
	})
	if err != nil {
		return fmt.Errorf("marshal graphic %d: %w", g.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("graphics"))
		if err != nil {
			return fmt.Errorf("create graphics bucket: %w", err)
		}
		return b.Put(int64Key(g.ID), raw)
	})
}

// GraphicIDs returns every graphic ID currently cached. See
// sde/category.go's CategoryIDs for why no separate sort is needed and
// what an empty result means.
func (s *Store) GraphicIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("graphics"))
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
