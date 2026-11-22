package sde

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Dogma attributes for what a character can use: the skills a type
// requires, and the ship groups and types a module can be fitted to.
// Verified against the fuzzwork dump's dgmAttributeTypes on 2026-11-20.
var (
	// AttrRequiredSkills pairs each requiredSkillN attribute with its
	// requiredSkillNLevel attribute.
	AttrRequiredSkills = [][2]int64{{182, 277}, {183, 278}, {184, 279}, {1285, 1286}, {1289, 1287}, {1290, 1288}}
	// AttrCanFitShipGroups are canFitShipGroup01 to canFitShipGroup20.
	AttrCanFitShipGroups = []int64{1298, 1299, 1300, 1301, 1872, 1879, 1880, 1881, 2065, 2396, 2476, 2477, 2478, 2479, 2480, 2481, 2482, 2483, 2484, 2485}
	// AttrCanFitShipTypes are canFitShipType1 to canFitShipType12.
	AttrCanFitShipTypes = []int64{1302, 1303, 1304, 1305, 1944, 2103, 2463, 2486, 2487, 2488, 2758, 5948}
)

// PublishedTypesInGroups lists the published types in any of the groups.
func (s *Store) PublishedTypesInGroups(ctx context.Context, userAgent string, groupIDs []int64) ([]TypeBasic, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(groupIDs)), ",")
	args := make([]any, len(groupIDs))
	for i, g := range groupIDs {
		args[i] = g
	}
	return s.typeBasicsWhere(ctx, `t.groupID IN (`+ph+`) AND t.published = 1`, args)
}

// PublishedTypesWithEffect lists the published types that have a dogma
// effect, by the effect's name.
func (s *Store) PublishedTypesWithEffect(ctx context.Context, userAgent, effectName string) ([]TypeBasic, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	return s.typeBasicsWhere(ctx, `t.published = 1 AND t.typeID IN (SELECT te.typeID FROM dgmTypeEffects te JOIN dgmEffects e ON e.effectID = te.effectID WHERE e.effectName = ?)`, []any{effectName})
}

func (s *Store) typeBasicsWhere(ctx context.Context, where string, args []any) ([]TypeBasic, error) {
	var out []TypeBasic
	err := s.eachRow(ctx, `SELECT t.typeID, t.typeName, t.groupID, COALESCE(g.groupName, ''), COALESCE(g.categoryID, 0), t.marketGroupID,
			COALESCE(t.packagedVolume, t.volume, 0), COALESCE(t.volume, 0), COALESCE(t.portionSize, 1), COALESCE(t.published, 0)
		FROM invTypes t LEFT JOIN invGroups g ON g.groupID = t.groupID WHERE `+where+` ORDER BY t.typeName`, args, func(r *sql.Rows) error {
		var b TypeBasic
		var mg sql.NullInt64
		if err := r.Scan(&b.TypeID, &b.Name, &b.GroupID, &b.GroupName, &b.CategoryID, &mg, &b.Volume, &b.AssembledVolume, &b.PortionSize, &b.Published); err != nil {
			return err
		}
		b.MarketGroupID = nullInt(mg)
		out = append(out, b)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("types: %w", err)
	}
	return out, nil
}
