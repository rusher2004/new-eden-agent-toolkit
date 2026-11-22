package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Planet is a minimal decode of a fuzzwork mapDenormalize row filtered to
// groupID 7 (Planet). mapDenormalize is official-SDE-derived (unlike the old
// bbolt design's esi-scrape-only source for this data), covering every
// celestial body's name/position/system/type in one table.
type Planet struct {
	ID       int64
	Name     string
	TypeID   int64
	Position Position
	SystemID int64
}

// Planet looks up a planet by ID. ok is false if the ID isn't in the cache,
// or belongs to a different celestial groupID (e.g. a moon's itemID).
func (s *Store) Planet(ctx context.Context, userAgent string, id int64) (Planet, bool, error) {
	if s == nil {
		return Planet{}, false, nil
	}
	if v, ok := getOverlay[Planet](s, "planet", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Planet{}, false, nil
	}

	var (
		name       sql.NullString
		systemName string
		typeID     int64
		systemID   int64
		x, y, z    sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, `+systemNameSQL("mapDenormalize.solarSystemID")+`, typeID, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID = 7`, id)
	if err := row.Scan(&name, &systemName, &typeID, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Planet{}, false, nil
		}
		return Planet{}, false, fmt.Errorf("query planet %d: %w", id, err)
	}

	return Planet{ID: id, Name: gameName(name.String, systemName), TypeID: typeID, SystemID: systemID, Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64}}, true, nil
}

// PutPlanet stores p in the overlay, keyed by p.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutPlanet(p Planet) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "planet", p.ID, p)
	return nil
}
