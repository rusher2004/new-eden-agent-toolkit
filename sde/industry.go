package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ErrUnavailable is returned by accessors that have no ESI fallback (the
// industry, dogma and routing data planning tools need) when the store has
// never synced and nothing is cached on disk.
var ErrUnavailable = errors.New("sde: static data is not available (sync failed and no cached copy)")

// requireDB syncs if needed and reports ErrUnavailable when there is no
// database to query. Used by accessors whose callers cannot fall back to ESI.
func (s *Store) requireDB(ctx context.Context, userAgent string) error {
	if s == nil {
		return ErrUnavailable
	}
	// A sync failure with a usable cached dump is not fatal (ensureSynced
	// logs it); it only matters when there is nothing to query at all.
	if err := s.ensureSynced(ctx, userAgent); err != nil && s.db == nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if s.db == nil {
		return ErrUnavailable
	}
	return nil
}

// Industry activity IDs, from ramActivities.
const (
	ActivityManufacturing int64 = 1
	ActivityResearchTE    int64 = 3
	ActivityResearchME    int64 = 4
	ActivityCopying       int64 = 5
	ActivityInvention     int64 = 8
	ActivityReactions     int64 = 11
)

// TypeQuantity is a type and a count.
type TypeQuantity struct {
	TypeID   int64
	Quantity int64
}

// BlueprintProduct is one output of a blueprint activity.
type BlueprintProduct struct {
	TypeID      int64
	Quantity    int64
	Probability float64 // invention only; 0 otherwise
}

// BlueprintSkill is a skill an activity requires.
type BlueprintSkill struct {
	SkillID int64
	Level   int64
}

// BlueprintActivity is one activity (manufacturing, reactions, ...) of a
// blueprint, with its base time, inputs, outputs and required skills.
type BlueprintActivity struct {
	ActivityID  int64
	TimeSeconds int64
	Materials   []TypeQuantity
	Products    []BlueprintProduct
	Skills      []BlueprintSkill
}

// Blueprint is a blueprint (or reaction formula) type and its activities.
type Blueprint struct {
	TypeID     int64
	Activities []BlueprintActivity
}

// Activity returns the given activity, or false if the blueprint has none.
func (b Blueprint) Activity(activityID int64) (BlueprintActivity, bool) {
	for _, a := range b.Activities {
		if a.ActivityID == activityID {
			return a, true
		}
	}
	return BlueprintActivity{}, false
}

// Blueprint loads a blueprint's activities. ok is false if the type has no
// industry activities.
func (s *Store) Blueprint(ctx context.Context, userAgent string, blueprintTypeID int64) (Blueprint, bool, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return Blueprint{}, false, err
	}
	bp := Blueprint{TypeID: blueprintTypeID}
	idx := map[int64]int{}

	rows, err := s.db.QueryContext(ctx, `SELECT activityID, time FROM industryActivity WHERE typeID = ? ORDER BY activityID`, blueprintTypeID)
	if err != nil {
		return Blueprint{}, false, fmt.Errorf("query blueprint %d activities: %w", blueprintTypeID, err)
	}
	for rows.Next() {
		var a BlueprintActivity
		if err := rows.Scan(&a.ActivityID, &a.TimeSeconds); err != nil {
			rows.Close()
			return Blueprint{}, false, fmt.Errorf("scan blueprint activity: %w", err)
		}
		idx[a.ActivityID] = len(bp.Activities)
		bp.Activities = append(bp.Activities, a)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Blueprint{}, false, fmt.Errorf("iterate blueprint activities: %w", err)
	}
	rows.Close()
	if len(bp.Activities) == 0 {
		return Blueprint{}, false, nil
	}

	if err := s.eachRow(ctx, `SELECT activityID, materialTypeID, quantity FROM industryActivityMaterials WHERE typeID = ? ORDER BY activityID, materialTypeID`, []any{blueprintTypeID}, func(r *sql.Rows) error {
		var act int64
		var m TypeQuantity
		if err := r.Scan(&act, &m.TypeID, &m.Quantity); err != nil {
			return err
		}
		if i, ok := idx[act]; ok {
			bp.Activities[i].Materials = append(bp.Activities[i].Materials, m)
		}
		return nil
	}); err != nil {
		return Blueprint{}, false, fmt.Errorf("blueprint %d materials: %w", blueprintTypeID, err)
	}

	if err := s.eachRow(ctx, `SELECT p.activityID, p.productTypeID, p.quantity, COALESCE(pr.probability, 0)
		FROM industryActivityProducts p
		LEFT JOIN industryActivityProbabilities pr ON pr.typeID = p.typeID AND pr.activityID = p.activityID AND pr.productTypeID = p.productTypeID
		WHERE p.typeID = ? ORDER BY p.activityID, p.productTypeID`, []any{blueprintTypeID}, func(r *sql.Rows) error {
		var act int64
		var p BlueprintProduct
		if err := r.Scan(&act, &p.TypeID, &p.Quantity, &p.Probability); err != nil {
			return err
		}
		if i, ok := idx[act]; ok {
			bp.Activities[i].Products = append(bp.Activities[i].Products, p)
		}
		return nil
	}); err != nil {
		return Blueprint{}, false, fmt.Errorf("blueprint %d products: %w", blueprintTypeID, err)
	}

	if err := s.eachRow(ctx, `SELECT activityID, skillID, level FROM industryActivitySkills WHERE typeID = ? ORDER BY activityID, skillID`, []any{blueprintTypeID}, func(r *sql.Rows) error {
		var act int64
		var sk BlueprintSkill
		if err := r.Scan(&act, &sk.SkillID, &sk.Level); err != nil {
			return err
		}
		if i, ok := idx[act]; ok {
			bp.Activities[i].Skills = append(bp.Activities[i].Skills, sk)
		}
		return nil
	}); err != nil {
		return Blueprint{}, false, fmt.Errorf("blueprint %d skills: %w", blueprintTypeID, err)
	}
	return bp, true, nil
}

// BlueprintSource names a blueprint activity that produces a type.
type BlueprintSource struct {
	BlueprintTypeID int64
	ActivityID      int64
	Quantity        int64 // output per run
}

// BlueprintsForProduct lists the manufacturing and reaction blueprints that
// produce a type.
func (s *Store) BlueprintsForProduct(ctx context.Context, userAgent string, productTypeID int64) ([]BlueprintSource, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	var out []BlueprintSource
	err := s.eachRow(ctx, `SELECT typeID, activityID, quantity FROM industryActivityProducts WHERE productTypeID = ? AND activityID IN (1, 11) ORDER BY activityID, typeID`, []any{productTypeID}, func(r *sql.Rows) error {
		var b BlueprintSource
		if err := r.Scan(&b.BlueprintTypeID, &b.ActivityID, &b.Quantity); err != nil {
			return err
		}
		out = append(out, b)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("blueprints for product %d: %w", productTypeID, err)
	}
	return out, nil
}

// ReprocessingMaterials returns what one portion of a type reprocesses into
// at 100% yield.
func (s *Store) ReprocessingMaterials(ctx context.Context, userAgent string, typeID int64) ([]TypeQuantity, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	var out []TypeQuantity
	err := s.eachRow(ctx, `SELECT materialTypeID, quantity FROM invTypeMaterials WHERE typeID = ? ORDER BY materialTypeID`, []any{typeID}, func(r *sql.Rows) error {
		var m TypeQuantity
		if err := r.Scan(&m.TypeID, &m.Quantity); err != nil {
			return err
		}
		out = append(out, m)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reprocessing materials for %d: %w", typeID, err)
	}
	return out, nil
}

// Ore is a compressed ore (or ice) type with what reprocessing needs.
type Ore struct {
	TypeID      int64
	Name        string
	GroupID     int64
	GroupName   string
	Volume      float64
	PortionSize int64
	SkillID     int64 // reprocessing skill (dogma attribute 790)
	Materials   []TypeQuantity
}

// CompressedOres lists every published, market-tradeable compressed asteroid
// type (ore, moon ore and ice) with its reprocessing outputs per portion.
// Callers filter by group if they want only some kinds.
func (s *Store) CompressedOres(ctx context.Context, userAgent string) ([]Ore, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	var out []Ore
	idx := map[int64]int{}
	err := s.eachRow(ctx, `SELECT t.typeID, t.typeName, t.groupID, g.groupName, COALESCE(t.volume, 0), COALESCE(t.portionSize, 1),
			COALESCE((SELECT COALESCE(a.valueInt, a.valueFloat) FROM dgmTypeAttributes a WHERE a.typeID = t.typeID AND a.attributeID = 790), 0)
		FROM invTypes t JOIN invGroups g ON g.groupID = t.groupID
		WHERE g.categoryID = 25 AND t.published = 1 AND t.marketGroupID IS NOT NULL AND t.typeName LIKE 'Compressed %'
		ORDER BY t.typeID`, nil, func(r *sql.Rows) error {
		var o Ore
		var skill float64
		if err := r.Scan(&o.TypeID, &o.Name, &o.GroupID, &o.GroupName, &o.Volume, &o.PortionSize, &skill); err != nil {
			return err
		}
		o.SkillID = int64(skill)
		idx[o.TypeID] = len(out)
		out = append(out, o)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("compressed ores: %w", err)
	}
	err = s.eachRow(ctx, `SELECT m.typeID, m.materialTypeID, m.quantity FROM invTypeMaterials m
		JOIN invTypes t ON t.typeID = m.typeID JOIN invGroups g ON g.groupID = t.groupID
		WHERE g.categoryID = 25 AND t.typeName LIKE 'Compressed %' ORDER BY m.typeID, m.materialTypeID`, nil, func(r *sql.Rows) error {
		var t int64
		var m TypeQuantity
		if err := r.Scan(&t, &m.TypeID, &m.Quantity); err != nil {
			return err
		}
		if i, ok := idx[t]; ok {
			out[i].Materials = append(out[i].Materials, m)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("compressed ore materials: %w", err)
	}
	return out, nil
}

// TypeBasic is the handful of type fields planning needs, loaded in bulk.
type TypeBasic struct {
	TypeID        int64
	Name          string
	GroupID       int64
	GroupName     string
	CategoryID    int64
	MarketGroupID *int64
	Volume        float64 // packaged volume when the SDE has one, otherwise volume
	// AssembledVolume is the type's volume when assembled (invTypes.volume),
	// which differs from Volume for ships and containers.
	AssembledVolume float64
	PortionSize     int64
	Published       bool
}

// TypeBasics loads TypeBasic for many types at once. Unknown IDs are absent
// from the result.
func (s *Store) TypeBasics(ctx context.Context, userAgent string, ids []int64) (map[int64]TypeBasic, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	out := make(map[int64]TypeBasic, len(ids))
	for start := 0; start < len(ids); start += 500 {
		end := min(start+500, len(ids))
		chunk := ids[start:end]
		ph := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		err := s.eachRow(ctx, `SELECT t.typeID, t.typeName, t.groupID, COALESCE(g.groupName, ''), COALESCE(g.categoryID, 0), t.marketGroupID,
				COALESCE(t.packagedVolume, t.volume, 0), COALESCE(t.volume, 0), COALESCE(t.portionSize, 1), COALESCE(t.published, 0)
			FROM invTypes t LEFT JOIN invGroups g ON g.groupID = t.groupID WHERE t.typeID IN (`+ph+`)`, args, func(r *sql.Rows) error {
			var b TypeBasic
			var mg sql.NullInt64
			if err := r.Scan(&b.TypeID, &b.Name, &b.GroupID, &b.GroupName, &b.CategoryID, &mg, &b.Volume, &b.AssembledVolume, &b.PortionSize, &b.Published); err != nil {
				return err
			}
			b.MarketGroupID = nullInt(mg)
			out[b.TypeID] = b
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("type basics: %w", err)
		}
	}
	return out, nil
}

// TypesByName resolves exact type names (case-insensitive) to IDs, preferring
// published types. Names that don't match are absent from the result, which is
// keyed by the name as given.
func (s *Store) TypesByName(ctx context.Context, userAgent string, names []string) (map[string]int64, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(names))
	for _, n := range names {
		var id int64
		err := s.db.QueryRowContext(ctx, `SELECT typeID FROM invTypes WHERE typeName = ? COLLATE NOCASE ORDER BY published DESC, typeID LIMIT 1`, n).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("type by name %q: %w", n, err)
		}
		out[n] = id
	}
	return out, nil
}

// NPCStationIndustry is the industry-relevant data for an NPC station.
type NPCStationIndustry struct {
	StationID                int64
	Name                     string
	SystemID                 int64
	OwnerCorporationID       int64
	OperationID              int64
	ReprocessingEfficiency   float64
	ReprocessingStationsTake float64
}

// NPCStationIndustryData loads an NPC station's reprocessing efficiency,
// station take, owner and operation. ok is false for unknown stations.
func (s *Store) NPCStationIndustryData(ctx context.Context, userAgent string, stationID int64) (NPCStationIndustry, bool, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return NPCStationIndustry{}, false, err
	}
	var st NPCStationIndustry
	var name sql.NullString
	var systemName string
	var corp, op sql.NullInt64
	var eff, take sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `SELECT stationID, stationName, `+systemNameSQL("staStations.solarSystemID")+`, solarSystemID, corporationID, operationID, reprocessingEfficiency, reprocessingStationsTake FROM staStations WHERE stationID = ?`, stationID).
		Scan(&st.StationID, &name, &systemName, &st.SystemID, &corp, &op, &eff, &take)
	if errors.Is(err, sql.ErrNoRows) {
		return NPCStationIndustry{}, false, nil
	}
	if err != nil {
		return NPCStationIndustry{}, false, fmt.Errorf("npc station %d: %w", stationID, err)
	}
	st.Name, st.OwnerCorporationID, st.OperationID = gameName(name.String, systemName), corp.Int64, op.Int64
	st.ReprocessingEfficiency, st.ReprocessingStationsTake = eff.Float64, take.Float64
	return st, true, nil
}

// eachRow runs a query and calls fn for each row, closing rows afterwards.
func (s *Store) eachRow(ctx context.Context, query string, args []any, fn func(*sql.Rows) error) error {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	return nil
}

// BlueprintSourcesForActivity lists blueprints whose given activity outputs
// a type, e.g. the T1 blueprints that invent a T2 blueprint (activity 8).
func (s *Store) BlueprintSourcesForActivity(ctx context.Context, userAgent string, productTypeID, activityID int64) ([]BlueprintSource, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	var out []BlueprintSource
	err := s.eachRow(ctx, `SELECT typeID, activityID, quantity FROM industryActivityProducts WHERE productTypeID = ? AND activityID = ? ORDER BY typeID`, []any{productTypeID, activityID}, func(r *sql.Rows) error {
		var b BlueprintSource
		if err := r.Scan(&b.BlueprintTypeID, &b.ActivityID, &b.Quantity); err != nil {
			return err
		}
		out = append(out, b)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("blueprints with activity %d for %d: %w", activityID, productTypeID, err)
	}
	return out, nil
}
