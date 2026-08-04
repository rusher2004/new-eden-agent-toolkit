package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Position is a shared 3D coordinate, reused by every geography resource.
type Position struct {
	X float64
	Y float64
	Z float64
}

// Constellation is a minimal decode of a fuzzwork mapConstellations row --
// EVE's geography level between Region and System. SystemIDs is not a
// stored column -- it's computed via a reverse query over mapSolarSystems at
// read time.
type Constellation struct {
	ID        int64
	RegionID  int64
	Name      string
	Position  Position
	SystemIDs []int64
}

// Constellation looks up a constellation by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) Constellation(ctx context.Context, userAgent string, id int64) (Constellation, bool, error) {
	if s == nil {
		return Constellation{}, false, nil
	}
	if v, ok := getOverlay[Constellation](s, "constellation", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Constellation{}, false, nil
	}

	var (
		regionID int64
		name     sql.NullString
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx, `SELECT regionID, constellationName, x, y, z FROM mapConstellations WHERE constellationID = ?`, id)
	if err := row.Scan(&regionID, &name, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Constellation{}, false, nil
		}
		return Constellation{}, false, fmt.Errorf("query constellation %d: %w", id, err)
	}

	systemIDs, err := s.queryInt64Column(ctx, `SELECT solarSystemID FROM mapSolarSystems WHERE constellationID = ? ORDER BY solarSystemID`, id)
	if err != nil {
		return Constellation{}, false, fmt.Errorf("query constellation %d system ids: %w", id, err)
	}

	return Constellation{
		ID: id, RegionID: regionID, Name: name.String,
		Position:  Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
		SystemIDs: systemIDs,
	}, true, nil
}

// PutConstellation stores c in the overlay, keyed by c.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutConstellation(c Constellation) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "constellation", c.ID, c)
	return nil
}

// ConstellationIDs returns every constellation ID currently cached, ascending.
func (s *Store) ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT constellationID FROM mapConstellations ORDER BY constellationID`)
}
