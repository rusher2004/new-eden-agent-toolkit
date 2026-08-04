package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Race is a minimal decode of an SDE race record, augmented at sync time
// with AllianceID from EVE Ref's esi-scrape dataset -- the official SDE's
// races.jsonl has no such field, and there is no alliances.jsonl anywhere
// in the SDE archive at all.
type Race struct {
	ID          int64
	AllianceID  *int64
	Description map[string]string
	Name        map[string]string
}

type raceRecord struct {
	Key         int64             `json:"_key"`
	AllianceID  *int64            `json:"allianceId,omitempty"`
	Description map[string]string `json:"description"`
	Name        map[string]string `json:"name"`
}

// Races returns every cached race. Returns an empty slice, not an error, if
// the store has never synced successfully.
func (s *Store) Races(ctx context.Context, userAgent string) ([]Race, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var out []Race
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("races"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw raceRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached race: %w", err)
			}
			out = append(out, Race{ID: raw.Key, AllianceID: raw.AllianceID, Description: raw.Description, Name: raw.Name})
			return nil
		})
	})
	return out, err
}

// Race looks up a race by ID. ok is false if the ID isn't in the cache.
func (s *Store) Race(ctx context.Context, userAgent string, id int64) (Race, bool, error) {
	if s == nil {
		return Race{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Race
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("races"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw raceRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached race %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached race %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Race{ID: raw.Key, AllianceID: raw.AllianceID, Description: raw.Description, Name: raw.Name}, true
		return nil
	})
	return out, ok, err
}
