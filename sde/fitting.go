package sde

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Dogma attributes the dump keeps as invTypes columns.
const (
	AttrMass     int64 = 4
	AttrCapacity int64 = 38
)

// DogmaModifier is one entry of an effect's modifierInfo.
type DogmaModifier struct {
	Domain        string
	Func          string
	ModifiedAttr  int64
	ModifyingAttr int64
	Operation     int
	SkillTypeID   int64
	GroupID       int64
}

// DogmaEffect is a type's dogma effect with its parsed modifiers.
type DogmaEffect struct {
	ID        int64
	Name      string
	Category  int
	Modifiers []DogmaModifier
}

// DogmaType is everything dogma needs about a type: its group, attributes
// and effects.
type DogmaType struct {
	TypeID  int64
	Name    string
	GroupID int64
	Attrs   map[int64]float64
	Effects []DogmaEffect
}

// DogmaTypes loads many types' attributes and effects at once. Unknown IDs
// are absent from the result.
func (s *Store) DogmaTypes(ctx context.Context, userAgent string, ids []int64) (map[int64]DogmaType, error) {
	out := map[int64]DogmaType{}
	if len(ids) == 0 {
		return out, nil
	}
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	for start := 0; start < len(ids); start += 500 {
		chunk := ids[start:min(start+500, len(ids))]
		ph := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		if err := s.eachRow(ctx, `SELECT typeID, typeName, groupID, capacity, mass FROM invTypes WHERE typeID IN (`+ph+`)`, args, func(r *sql.Rows) error {
			var t DogmaType
			var name sql.NullString
			var capacity, mass sql.NullFloat64
			if err := r.Scan(&t.TypeID, &name, &t.GroupID, &capacity, &mass); err != nil {
				return err
			}
			// The dump keeps capacity and mass as invTypes columns, not as
			// their dogma attributes; dogma starts from them.
			t.Name, t.Attrs = name.String, map[int64]float64{}
			if capacity.Valid {
				t.Attrs[AttrCapacity] = capacity.Float64
			}
			if mass.Valid {
				t.Attrs[AttrMass] = mass.Float64
			}
			out[t.TypeID] = t
			return nil
		}); err != nil {
			return nil, fmt.Errorf("dogma types: %w", err)
		}
		if err := s.eachRow(ctx, `SELECT typeID, attributeID, COALESCE(valueFloat, valueInt, 0) FROM dgmTypeAttributes WHERE typeID IN (`+ph+`)`, args, func(r *sql.Rows) error {
			var id, attr int64
			var v float64
			if err := r.Scan(&id, &attr, &v); err != nil {
				return err
			}
			if t, ok := out[id]; ok {
				t.Attrs[attr] = v
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("dogma attributes: %w", err)
		}
		if err := s.eachRow(ctx, `SELECT te.typeID, e.effectID, COALESCE(e.effectName, ''), COALESCE(e.effectCategory, 0), COALESCE(e.modifierInfo, '')
			FROM dgmTypeEffects te JOIN dgmEffects e ON e.effectID = te.effectID WHERE te.typeID IN (`+ph+`) ORDER BY te.typeID, e.effectID`, args, func(r *sql.Rows) error {
			var id int64
			var e DogmaEffect
			var raw string
			if err := r.Scan(&id, &e.ID, &e.Name, &e.Category, &raw); err != nil {
				return err
			}
			if raw != "" && raw[0] == '[' {
				var mods []modifierInfo
				if err := json.Unmarshal([]byte(raw), &mods); err != nil {
					return fmt.Errorf("parse modifierInfo of effect %d: %w", e.ID, err)
				}
				for _, m := range mods {
					e.Modifiers = append(e.Modifiers, DogmaModifier{Domain: m.Domain, Func: m.Func, ModifiedAttr: m.ModifiedAttributeID, ModifyingAttr: m.ModifyingAttributeID,
						Operation: m.Operation, SkillTypeID: m.SkillTypeID, GroupID: m.GroupID})
				}
			}
			if t, ok := out[id]; ok {
				t.Effects = append(t.Effects, e)
				out[id] = t
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("dogma effects: %w", err)
		}
	}
	return out, nil
}

// AttributeInfo is an attribute's default value and whether modules
// changing it are free of stacking penalties.
type AttributeInfo struct {
	Default   float64
	Stackable bool
}

// AttributeInfos loads every dogma attribute's default and stackability.
func (s *Store) AttributeInfos(ctx context.Context, userAgent string) (map[int64]AttributeInfo, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	out := map[int64]AttributeInfo{}
	err := s.eachRow(ctx, `SELECT attributeID, COALESCE(defaultValue, 0), COALESCE(stackable, 1) FROM dgmAttributeTypes`, nil, func(r *sql.Rows) error {
		var id int64
		var a AttributeInfo
		if err := r.Scan(&id, &a.Default, &a.Stackable); err != nil {
			return err
		}
		out[id] = a
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("attribute info: %w", err)
	}
	return out, nil
}
