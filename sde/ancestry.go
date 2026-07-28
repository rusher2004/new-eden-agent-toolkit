package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Ancestry is a minimal decode of an SDE ancestry record.
type Ancestry struct {
	ID               int64
	BloodlineID      int64
	Description      map[string]string
	Name             map[string]string
	IconID           *int64
	ShortDescription *string
}

type ancestryRecord struct {
	Key              int64             `json:"_key"`
	BloodlineID      int64             `json:"bloodlineID"`
	Description      map[string]string `json:"description"`
	Name             map[string]string `json:"name"`
	IconID           *int64            `json:"iconID,omitempty"`
	ShortDescription *string           `json:"shortDescription,omitempty"`
}

// Ancestries returns every cached ancestry. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Ancestries(ctx context.Context, userAgent string) ([]Ancestry, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var out []Ancestry
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("ancestries"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var raw ancestryRecord
			if err := json.Unmarshal(v, &raw); err != nil {
				return fmt.Errorf("parse cached ancestry: %w", err)
			}
			out = append(out, Ancestry{
				ID: raw.Key, BloodlineID: raw.BloodlineID, Description: raw.Description,
				Name: raw.Name, IconID: raw.IconID, ShortDescription: raw.ShortDescription,
			})
			return nil
		})
	})
	return out, err
}
