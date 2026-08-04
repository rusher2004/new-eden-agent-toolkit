package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Bloodline is a minimal decode of an SDE bloodline record.
type Bloodline struct {
	ID            int64
	Charisma      int64
	CorporationID int64
	Description   map[string]string
	Intelligence  int64
	Memory        int64
	Name          map[string]string
	Perception    int64
	RaceID        int64
	ShipTypeID    *int64
	Willpower     int64
}

type bloodlineRecord struct {
	Key           int64             `json:"_key"`
	Charisma      int64             `json:"charisma"`
	CorporationID int64             `json:"corporationID"`
	Description   map[string]string `json:"description"`
	Intelligence  int64             `json:"intelligence"`
	Memory        int64             `json:"memory"`
	Name          map[string]string `json:"name"`
	Perception    int64             `json:"perception"`
	RaceID        int64             `json:"raceID"`
	ShipTypeID    *int64            `json:"shipTypeID,omitempty"`
	Willpower     int64             `json:"willpower"`
}

// Bloodlines returns every cached bloodline. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Bloodlines(ctx context.Context, userAgent string) ([]Bloodline, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var out []Bloodline
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("bloodlines"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw bloodlineRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached bloodline: %w", err)
			}
			out = append(out, Bloodline{
				ID: raw.Key, Charisma: raw.Charisma, CorporationID: raw.CorporationID, Description: raw.Description,
				Intelligence: raw.Intelligence, Memory: raw.Memory, Name: raw.Name, Perception: raw.Perception,
				RaceID: raw.RaceID, ShipTypeID: raw.ShipTypeID, Willpower: raw.Willpower,
			})
			return nil
		})
	})
	return out, err
}

// Bloodline looks up a bloodline by ID. ok is false if the ID isn't in the cache.
func (s *Store) Bloodline(ctx context.Context, userAgent string, id int64) (Bloodline, bool, error) {
	if s == nil {
		return Bloodline{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Bloodline
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("bloodlines"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw bloodlineRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached bloodline %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached bloodline %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Bloodline{
			ID: raw.Key, Charisma: raw.Charisma, CorporationID: raw.CorporationID, Description: raw.Description,
			Intelligence: raw.Intelligence, Memory: raw.Memory, Name: raw.Name, Perception: raw.Perception,
			RaceID: raw.RaceID, ShipTypeID: raw.ShipTypeID, Willpower: raw.Willpower,
		}, true
		return nil
	})
	return out, ok, err
}
