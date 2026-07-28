package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Star is a minimal decode of an EVE Ref esi-scrape star record. Sourced
// entirely from esi-scrape, same as AsteroidBelt -- see its doc comment.
type Star struct {
	ID            int64
	Age           int64
	Luminosity    float64
	Name          string
	Radius        int64
	SolarSystemID int64
	SpectralClass string
	Temperature   int64
	TypeID        int64
}

type starRecord struct {
	Key           int64   `json:"_key"`
	Age           int64   `json:"age"`
	Luminosity    float64 `json:"luminosity"`
	Name          string  `json:"name"`
	Radius        int64   `json:"radius"`
	SolarSystemID int64   `json:"solarSystemId"`
	SpectralClass string  `json:"spectralClass"`
	Temperature   int64   `json:"temperature"`
	TypeID        int64   `json:"typeId"`
}

// Star looks up a star by ID. ok is false if the ID isn't in the cache.
func (s *Store) Star(ctx context.Context, userAgent string, id int64) (Star, bool, error) {
	if s == nil {
		return Star{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Star
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("stars"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw starRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached star %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached star %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Star{
			ID: raw.Key, Age: raw.Age, Luminosity: raw.Luminosity, Name: raw.Name, Radius: raw.Radius,
			SolarSystemID: raw.SolarSystemID, SpectralClass: raw.SpectralClass, Temperature: raw.Temperature, TypeID: raw.TypeID,
		}, true
		return nil
	})
	return out, ok, err
}

// PutStar writes st into the cache, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStar(st Star) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(starRecord{
		Key: st.ID, Age: st.Age, Luminosity: st.Luminosity, Name: st.Name, Radius: st.Radius,
		SolarSystemID: st.SolarSystemID, SpectralClass: st.SpectralClass, Temperature: st.Temperature, TypeID: st.TypeID,
	})
	if err != nil {
		return fmt.Errorf("marshal star %d: %w", st.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("stars"))
		if err != nil {
			return fmt.Errorf("create stars bucket: %w", err)
		}
		return b.Put(int64Key(st.ID), raw)
	})
}
