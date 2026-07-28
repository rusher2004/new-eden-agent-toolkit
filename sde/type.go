package sde

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

// TypeDogmaAttribute mirrors mcp/dogma.go's DynamicItemAttribute shape, as
// its own sde-owned type -- sde/ does not import mcp/.
type TypeDogmaAttribute struct {
	AttributeID int64
	Value       float64
}

// TypeDogmaEffect mirrors mcp/dogma.go's DynamicItemEffect shape.
type TypeDogmaEffect struct {
	EffectID  int64
	IsDefault bool
}

// Type is a minimal decode of an SDE type record -- the leaf level of
// EVE's item hierarchy (e.g. "Rifter", within the "Frigate" group).
// PackagedVolume has no source in the official SDE at all -- it's sourced
// entirely from EVE Ref's reference-data dataset at sync time, the same
// "stored but not raw from the primary source" pattern System.StationIDs
// already established.
type Type struct {
	ID              int64
	GroupID         int64
	Name            map[string]string
	Description     map[string]string
	Published       bool
	Capacity        *float64
	GraphicID       *int64
	IconID          *int64
	MarketGroupID   *int64
	Mass            *float64
	PackagedVolume  *float64
	PortionSize     *int64
	Radius          *float64
	Volume          *float64
	DogmaAttributes []TypeDogmaAttribute
	DogmaEffects    []TypeDogmaEffect
}

type typeDogmaAttributeRecord struct {
	AttributeID int64   `json:"attributeID"`
	Value       float64 `json:"value"`
}

type typeDogmaEffectRecord struct {
	EffectID  int64 `json:"effectID"`
	IsDefault bool  `json:"isDefault"`
}

type typeRecord struct {
	Key             int64                      `json:"_key"`
	GroupID         int64                      `json:"groupID"`
	Name            map[string]string          `json:"name"`
	Description     map[string]string          `json:"description"`
	Published       bool                       `json:"published"`
	Capacity        *float64                   `json:"capacity,omitempty"`
	GraphicID       *int64                     `json:"graphicID,omitempty"`
	IconID          *int64                     `json:"iconID,omitempty"`
	MarketGroupID   *int64                     `json:"marketGroupID,omitempty"`
	Mass            *float64                   `json:"mass,omitempty"`
	PackagedVolume  *float64                   `json:"packagedVolume,omitempty"`
	PortionSize     *int64                     `json:"portionSize,omitempty"`
	Radius          *float64                   `json:"radius,omitempty"`
	Volume          *float64                   `json:"volume,omitempty"`
	DogmaAttributes []typeDogmaAttributeRecord `json:"dogmaAttributes,omitempty"`
	DogmaEffects    []typeDogmaEffectRecord    `json:"dogmaEffects,omitempty"`
}

// Type looks up a type by ID. ok is false if the ID isn't in the cache.
func (s *Store) Type(ctx context.Context, userAgent string, id int64) (Type, bool, error) {
	if s == nil {
		return Type{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var (
		out Type
		ok  bool
	)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("types"))
		if b == nil {
			return nil
		}
		v := b.Get(int64Key(id))
		if v == nil {
			return nil
		}
		var raw typeRecord
		if err := json.Unmarshal(v, &raw); err != nil {
			return fmt.Errorf("parse cached type %d: %w", id, err)
		}
		if raw.Key != id {
			return fmt.Errorf("cached type %d has mismatched key %d", id, raw.Key)
		}
		out = Type{
			ID: raw.Key, GroupID: raw.GroupID, Name: raw.Name, Description: raw.Description, Published: raw.Published,
			Capacity: raw.Capacity, GraphicID: raw.GraphicID, IconID: raw.IconID, MarketGroupID: raw.MarketGroupID,
			Mass: raw.Mass, PackagedVolume: raw.PackagedVolume, PortionSize: raw.PortionSize, Radius: raw.Radius, Volume: raw.Volume,
		}
		if len(raw.DogmaAttributes) > 0 {
			out.DogmaAttributes = make([]TypeDogmaAttribute, len(raw.DogmaAttributes))
			for i, a := range raw.DogmaAttributes {
				out.DogmaAttributes[i] = TypeDogmaAttribute{AttributeID: a.AttributeID, Value: a.Value}
			}
		}
		if len(raw.DogmaEffects) > 0 {
			out.DogmaEffects = make([]TypeDogmaEffect, len(raw.DogmaEffects))
			for i, e := range raw.DogmaEffects {
				out.DogmaEffects[i] = TypeDogmaEffect{EffectID: e.EffectID, IsDefault: e.IsDefault}
			}
		}
		ok = true
		return nil
	})
	return out, ok, err
}

// PutType writes t into the cache, keyed by t.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutType(t Type) error {
	if s == nil {
		return nil
	}
	raw := typeRecord{
		Key: t.ID, GroupID: t.GroupID, Name: t.Name, Description: t.Description, Published: t.Published,
		Capacity: t.Capacity, GraphicID: t.GraphicID, IconID: t.IconID, MarketGroupID: t.MarketGroupID,
		Mass: t.Mass, PackagedVolume: t.PackagedVolume, PortionSize: t.PortionSize, Radius: t.Radius, Volume: t.Volume,
	}
	if len(t.DogmaAttributes) > 0 {
		raw.DogmaAttributes = make([]typeDogmaAttributeRecord, len(t.DogmaAttributes))
		for i, a := range t.DogmaAttributes {
			raw.DogmaAttributes[i] = typeDogmaAttributeRecord{AttributeID: a.AttributeID, Value: a.Value}
		}
	}
	if len(t.DogmaEffects) > 0 {
		raw.DogmaEffects = make([]typeDogmaEffectRecord, len(t.DogmaEffects))
		for i, e := range t.DogmaEffects {
			raw.DogmaEffects[i] = typeDogmaEffectRecord{EffectID: e.EffectID, IsDefault: e.IsDefault}
		}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal type %d: %w", t.ID, err)
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("types"))
		if err != nil {
			return fmt.Errorf("create types bucket: %w", err)
		}
		return b.Put(int64Key(t.ID), data)
	})
}

// TypeIDs returns every type ID currently cached. See CategoryIDs for why
// no separate sort is needed and what an empty result means.
func (s *Store) TypeIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)

	var ids []int64
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("types"))
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
