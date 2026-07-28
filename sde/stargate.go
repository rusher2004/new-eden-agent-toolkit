package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// Stargate is a minimal decode of an EVE Ref esi-scrape stargate record.
// Sourced entirely from esi-scrape, same as AsteroidBelt -- see its doc
// comment.
type Stargate struct {
	ID                    int64
	Name                  string
	TypeID                int64
	Position              Position
	SystemID              int64
	DestinationStargateID int64
	DestinationSystemID   int64
}

type stargateRecord struct {
	Key                   int64    `json:"_key"`
	Name                  string   `json:"name"`
	TypeID                int64    `json:"typeId"`
	Position              Position `json:"position"`
	SystemID              int64    `json:"systemId"`
	DestinationStargateID int64    `json:"destinationStargateId"`
	DestinationSystemID   int64    `json:"destinationSystemId"`
}

// Stargate looks up a stargate by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Stargate(ctx context.Context, userAgent string, id int64) (Stargate, bool, error) {
	if s == nil {
		return Stargate{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Stargate
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("stargates"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw stargateRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached stargate %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached stargate %d has mismatched key %d", id, raw.Key)
		}
		out, ok = Stargate{
			ID: raw.Key, Name: raw.Name, TypeID: raw.TypeID, Position: raw.Position, SystemID: raw.SystemID,
			DestinationStargateID: raw.DestinationStargateID, DestinationSystemID: raw.DestinationSystemID,
		}, true
		return nil
	})
	return out, ok, err
}

// PutStargate writes g into the cache, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStargate(g Stargate) error {
	if s == nil {
		return nil
	}
	raw, err := json.Marshal(stargateRecord{
		Key: g.ID, Name: g.Name, TypeID: g.TypeID, Position: g.Position, SystemID: g.SystemID,
		DestinationStargateID: g.DestinationStargateID, DestinationSystemID: g.DestinationSystemID,
	})
	if err != nil {
		return fmt.Errorf("marshal stargate %d: %w", g.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("stargates"))
		if err != nil {
			return fmt.Errorf("create stargates bucket: %w", err)
		}
		return b.Put(int64Key(g.ID), raw)
	})
}
