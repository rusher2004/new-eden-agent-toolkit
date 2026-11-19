package sde

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
)

// Dogma attribute IDs used by industry planning. Verified against the
// fuzzwork dump's dgmAttributeTypes on 2026-11-18.
const (
	AttrReprocessingSkill         int64 = 790  // reprocessingSkillType on ores
	AttrJumpDriveConsumptionType  int64 = 866  // fuel type
	AttrJumpDriveRange            int64 = 867  // base range, light years
	AttrJumpDriveConsumption      int64 = 868  // fuel per light year
	AttrAnchoringSecurityMax      int64 = 1032 // anchoringSecurityLevelMax (starbase arrays)
	AttrJumpFreighterFuelBonus    int64 = 1312 // eliteBonusJumpFreighter2, percent per Jump Freighters level
	AttrDisallowInHighSec         int64 = 1970
	AttrJumpFatigueMultiplier     int64 = 1971
	AttrRigSize                   int64 = 1547
	AttrHiSecModifier             int64 = 2355
	AttrLowSecModifier            int64 = 2356
	AttrNullSecModifier           int64 = 2357
	AttrEngRigTimeBonus           int64 = 2593 // percent
	AttrEngRigMatBonus            int64 = 2594 // percent
	AttrEngRigCostBonus           int64 = 2595 // percent
	AttrStrEngMatBonus            int64 = 2600 // multiplier
	AttrStrEngCostBonus           int64 = 2601 // multiplier on the system cost index
	AttrStrEngTimeBonus           int64 = 2602 // multiplier
	AttrRefRigTimeBonus           int64 = 2713 // percent (reactor rigs)
	AttrRefRigMatBonus            int64 = 2714 // percent (reactor rigs)
	AttrStrReactionTimeMultiplier int64 = 2721
	AttrStrRefiningYieldBonus     int64 = 2722 // percent
)

// TypeAttributes returns every dogma attribute value of a type.
func (s *Store) TypeAttributes(ctx context.Context, userAgent string, typeID int64) (map[int64]float64, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	out := map[int64]float64{}
	err := s.eachRow(ctx, `SELECT attributeID, COALESCE(valueFloat, valueInt, 0) FROM dgmTypeAttributes WHERE typeID = ?`, []any{typeID}, func(r *sql.Rows) error {
		var id int64
		var v float64
		if err := r.Scan(&id, &v); err != nil {
			return err
		}
		out[id] = v
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("attributes of %d: %w", typeID, err)
	}
	return out, nil
}

// RigModifier is one structure attribute a rig changes, and the rig attribute
// that supplies the amount.
type RigModifier struct {
	TargetAttributeID int64 // structure attribute, e.g. 2592 all-ships material
	SourceAttributeID int64 // rig attribute, e.g. 2594 material bonus
}

// Rig is a structure rig's industry bonuses as the SDE describes them.
type Rig struct {
	TypeID            int64
	Name              string
	Size              int64
	Modifiers         []RigModifier
	Attributes        map[int64]float64
	DisallowInHighSec bool
}

type modifierInfo struct {
	Domain               string `json:"domain"`
	Func                 string `json:"func"`
	ModifiedAttributeID  int64  `json:"modifiedAttributeID"`
	ModifyingAttributeID int64  `json:"modifyingAttributeID"`
}

// RigInfo loads a rig's structure modifiers (from its dogma effects) and its
// attributes. ok is false if the type has no structure-domain modifiers.
func (s *Store) RigInfo(ctx context.Context, userAgent string, typeID int64) (Rig, bool, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return Rig{}, false, err
	}
	rig := Rig{TypeID: typeID}
	var name sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT typeName FROM invTypes WHERE typeID = ?`, typeID).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return Rig{}, false, nil
		}
		return Rig{}, false, fmt.Errorf("rig %d name: %w", typeID, err)
	}
	rig.Name = name.String
	seen := map[RigModifier]bool{}
	err := s.eachRow(ctx, `SELECT COALESCE(e.modifierInfo, '') FROM dgmTypeEffects te JOIN dgmEffects e ON e.effectID = te.effectID WHERE te.typeID = ?`, []any{typeID}, func(r *sql.Rows) error {
		var raw string
		if err := r.Scan(&raw); err != nil {
			return err
		}
		if raw == "" || raw[0] != '[' {
			return nil
		}
		var mods []modifierInfo
		if err := json.Unmarshal([]byte(raw), &mods); err != nil {
			return fmt.Errorf("parse modifierInfo: %w", err)
		}
		for _, m := range mods {
			if m.Domain != "structureID" {
				continue
			}
			rm := RigModifier{TargetAttributeID: m.ModifiedAttributeID, SourceAttributeID: m.ModifyingAttributeID}
			if !seen[rm] {
				seen[rm] = true
				rig.Modifiers = append(rig.Modifiers, rm)
			}
		}
		return nil
	})
	if err != nil {
		return Rig{}, false, fmt.Errorf("rig %d effects: %w", typeID, err)
	}
	sort.Slice(rig.Modifiers, func(i, j int) bool {
		if rig.Modifiers[i].TargetAttributeID != rig.Modifiers[j].TargetAttributeID {
			return rig.Modifiers[i].TargetAttributeID < rig.Modifiers[j].TargetAttributeID
		}
		return rig.Modifiers[i].SourceAttributeID < rig.Modifiers[j].SourceAttributeID
	})
	attrs, err := s.TypeAttributes(ctx, userAgent, typeID)
	if err != nil {
		return Rig{}, false, err
	}
	rig.Attributes = attrs
	rig.Size = int64(attrs[AttrRigSize])
	rig.DisallowInHighSec = attrs[AttrDisallowInHighSec] != 0
	return rig, len(rig.Modifiers) > 0, nil
}

// IndustryModifierSource is one row of the SDE's indModifierSources: a
// structure or rig changes one kind of one activity's cost, through a dogma
// attribute, optionally only for products matching a target filter.
type IndustryModifierSource struct {
	Activity    string // manufacturing, reaction, copying, invention, researchMaterial, researchTime
	Kind        string // material, time, cost
	AttributeID int64
	FilterID    int64 // 0 = applies to every product
}

// IndustryModifiers lists a structure's or rig's industry modifier sources.
func (s *Store) IndustryModifiers(ctx context.Context, userAgent string, typeID int64) ([]IndustryModifierSource, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	var out []IndustryModifierSource
	err := s.eachRow(ctx, `SELECT activityName, modifierType, dogmaAttributeID, COALESCE(filterID, 0) FROM indModifierSources WHERE typeID = ? ORDER BY activityName, modifierType, dogmaAttributeID`, []any{typeID}, func(r *sql.Rows) error {
		var m IndustryModifierSource
		if err := r.Scan(&m.Activity, &m.Kind, &m.AttributeID, &m.FilterID); err != nil {
			return err
		}
		out = append(out, m)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("industry modifiers of %d: %w", typeID, err)
	}
	return out, nil
}

// TargetFilter is an industry target filter: the product categories and
// groups a modifier applies to.
type TargetFilter struct {
	ID         int64
	Name       string
	Categories []int64
	Groups     []int64
}

// TargetFilters returns every industry target filter, keyed by ID.
func (s *Store) TargetFilters(ctx context.Context, userAgent string) (map[int64]TargetFilter, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	out := map[int64]TargetFilter{}
	if err := s.eachRow(ctx, `SELECT targetFilterID, name FROM indTargetFilters`, nil, func(r *sql.Rows) error {
		var f TargetFilter
		if err := r.Scan(&f.ID, &f.Name); err != nil {
			return err
		}
		out[f.ID] = f
		return nil
	}); err != nil {
		return nil, fmt.Errorf("target filters: %w", err)
	}
	add := func(query string, set func(f *TargetFilter, id int64)) error {
		return s.eachRow(ctx, query, nil, func(r *sql.Rows) error {
			var fid, id int64
			if err := r.Scan(&fid, &id); err != nil {
				return err
			}
			if f, ok := out[fid]; ok {
				set(&f, id)
				out[fid] = f
			}
			return nil
		})
	}
	if err := add(`SELECT targetFilterID, categoryID FROM indTargetFilterCategories ORDER BY 1, 2`, func(f *TargetFilter, id int64) { f.Categories = append(f.Categories, id) }); err != nil {
		return nil, fmt.Errorf("target filter categories: %w", err)
	}
	if err := add(`SELECT targetFilterID, groupID FROM indTargetFilterGroups ORDER BY 1, 2`, func(f *TargetFilter, id int64) { f.Groups = append(f.Groups, id) }); err != nil {
		return nil, fmt.Errorf("target filter groups: %w", err)
	}
	return out, nil
}
