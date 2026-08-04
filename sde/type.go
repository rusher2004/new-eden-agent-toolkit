package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// Type is a minimal decode of a fuzzwork invTypes row -- the leaf level of
// EVE's item hierarchy (e.g. "Rifter", within the "Frigate" group).
// PackagedVolume has no source in the fuzzwork dump at all -- it's sourced
// entirely from EVE Ref's reference-data dataset at sync time (see sync.go),
// held in-memory rather than a stored column.
type Type struct {
	ID              int64
	GroupID         int64
	Name            string
	Description     string
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

// Type looks up a type by ID. ok is false if the ID isn't in the cache.
func (s *Store) Type(ctx context.Context, userAgent string, id int64) (Type, bool, error) {
	if s == nil {
		return Type{}, false, nil
	}
	if v, ok := getOverlay[Type](s, "type", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Type{}, false, nil
	}

	var (
		groupID                                       int64
		name, description                             sql.NullString
		published                                     sql.NullBool
		capacity, mass, radius, volume                sql.NullFloat64
		graphicID, iconID, marketGroupID, portionSize sql.NullInt64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT groupID, typeName, description, published, capacity, graphicID, iconID, marketGroupID, mass, portionSize, radius, volume
		 FROM invTypes WHERE typeID = ?`, id)
	err := row.Scan(&groupID, &name, &description, &published, &capacity, &graphicID, &iconID, &marketGroupID, &mass, &portionSize, &radius, &volume)
	if errors.Is(err, sql.ErrNoRows) {
		return Type{}, false, nil
	}
	if err != nil {
		return Type{}, false, fmt.Errorf("query type %d: %w", id, err)
	}

	out := Type{
		ID: id, GroupID: groupID, Name: name.String, Description: description.String, Published: published.Bool,
		Capacity: nullFloat(capacity), GraphicID: nullInt(graphicID), IconID: nullInt(iconID),
		MarketGroupID: nullInt(marketGroupID), Mass: nullFloat(mass), PortionSize: nullInt(portionSize),
		Radius: nullFloat(radius), Volume: nullFloat(volume),
	}
	if v, ok := s.typeVolumes[id]; ok {
		out.PackagedVolume = &v
	}

	attrRows, err := s.db.QueryContext(ctx, `SELECT attributeID, valueInt, valueFloat FROM dgmTypeAttributes WHERE typeID = ?`, id)
	if err != nil {
		return Type{}, false, fmt.Errorf("query type %d dogma attributes: %w", id, err)
	}
	defer attrRows.Close()
	for attrRows.Next() {
		var attributeID int64
		var valueInt sql.NullInt64
		var valueFloat sql.NullFloat64
		if err := attrRows.Scan(&attributeID, &valueInt, &valueFloat); err != nil {
			return Type{}, false, fmt.Errorf("scan type %d dogma attribute: %w", id, err)
		}
		var v float64
		switch {
		case valueFloat.Valid:
			v = valueFloat.Float64
		case valueInt.Valid:
			v = float64(valueInt.Int64)
		}
		out.DogmaAttributes = append(out.DogmaAttributes, TypeDogmaAttribute{AttributeID: attributeID, Value: v})
	}
	if err := attrRows.Err(); err != nil {
		return Type{}, false, fmt.Errorf("iterate type %d dogma attributes: %w", id, err)
	}

	effectRows, err := s.db.QueryContext(ctx, `SELECT effectID, isDefault FROM dgmTypeEffects WHERE typeID = ?`, id)
	if err != nil {
		return Type{}, false, fmt.Errorf("query type %d dogma effects: %w", id, err)
	}
	defer effectRows.Close()
	for effectRows.Next() {
		var effectID int64
		var isDefault bool
		if err := effectRows.Scan(&effectID, &isDefault); err != nil {
			return Type{}, false, fmt.Errorf("scan type %d dogma effect: %w", id, err)
		}
		out.DogmaEffects = append(out.DogmaEffects, TypeDogmaEffect{EffectID: effectID, IsDefault: isDefault})
	}
	if err := effectRows.Err(); err != nil {
		return Type{}, false, fmt.Errorf("iterate type %d dogma effects: %w", id, err)
	}

	return out, true, nil
}

// PutType stores t in the overlay, keyed by t.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutType(t Type) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "type", t.ID, t)
	return nil
}

// TypeIDs returns every type ID currently cached, ascending.
func (s *Store) TypeIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT typeID FROM invTypes ORDER BY typeID`)
}
