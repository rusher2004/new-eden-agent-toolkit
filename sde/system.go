package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SystemPlanet is a minimal decode of one planet entry within a solar system
// record, mirroring mcp.UniverseSystemPlanet's shape as its own sde-owned
// type -- sde/ does not import mcp/. Computed via mapDenormalize's orbitID
// column (a moon/belt's orbitID is its parent planet's itemID) at read time,
// not a stored relation.
type SystemPlanet struct {
	PlanetID        int64
	AsteroidBeltIDs []int64
	MoonIDs         []int64
}

// System is a minimal decode of a fuzzwork mapSolarSystems row -- EVE's
// geography level between Constellation and celestial bodies. StarID,
// StargateIDs, StationIDs, and Planets are not stored columns -- all four
// are computed via reverse queries over mapDenormalize/staStations/mapJumps
// at read time (celestial groupIDs: 6=Sun, 7=Planet, 8=Moon, 9/4430=Asteroid
// Belt, 10=Stargate, verified directly against the live dump).
type System struct {
	ID              int64
	ConstellationID int64
	Name            string
	Position        Position
	SecurityStatus  float64
	SecurityClass   *string
	StarID          *int64
	StargateIDs     []int64
	StationIDs      []int64
	Planets         []SystemPlanet
}

// System looks up a solar system by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) System(ctx context.Context, userAgent string, id int64) (System, bool, error) {
	if s == nil {
		return System{}, false, nil
	}
	if v, ok := getOverlay[System](s, "system", id); ok {
		return v, true, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return System{}, false, nil
	}

	var (
		constellationID int64
		name            sql.NullString
		x, y, z         sql.NullFloat64
		security        sql.NullFloat64
		securityClass   sql.NullString
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT constellationID, solarSystemName, x, y, z, security, securityClass FROM mapSolarSystems WHERE solarSystemID = ?`, id)
	if err := row.Scan(&constellationID, &name, &x, &y, &z, &security, &securityClass); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return System{}, false, nil
		}
		return System{}, false, fmt.Errorf("query system %d: %w", id, err)
	}

	out := System{
		ID: id, ConstellationID: constellationID, Name: name.String,
		Position:       Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
		SecurityStatus: security.Float64,
	}
	if securityClass.Valid {
		out.SecurityClass = &securityClass.String
	}

	starIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE solarSystemID = ? AND groupID = 6 LIMIT 1`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d star id: %w", id, err)
	}
	if len(starIDs) > 0 {
		out.StarID = &starIDs[0]
	}

	stargateIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE solarSystemID = ? AND groupID = 10 ORDER BY itemID`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d stargate ids: %w", id, err)
	}
	out.StargateIDs = stargateIDs

	stationIDs, err := s.queryInt64Column(ctx, `SELECT stationID FROM staStations WHERE solarSystemID = ? ORDER BY stationID`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d station ids: %w", id, err)
	}
	out.StationIDs = stationIDs

	planetIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE solarSystemID = ? AND groupID = 7 ORDER BY celestialIndex, itemID`, id)
	if err != nil {
		return System{}, false, fmt.Errorf("query system %d planet ids: %w", id, err)
	}
	for _, planetID := range planetIDs {
		moonIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE orbitID = ? AND groupID = 8 ORDER BY itemID`, planetID)
		if err != nil {
			return System{}, false, fmt.Errorf("query system %d planet %d moon ids: %w", id, planetID, err)
		}
		beltIDs, err := s.queryInt64Column(ctx, `SELECT itemID FROM mapDenormalize WHERE orbitID = ? AND groupID IN (9, 4430) ORDER BY itemID`, planetID)
		if err != nil {
			return System{}, false, fmt.Errorf("query system %d planet %d belt ids: %w", id, planetID, err)
		}
		out.Planets = append(out.Planets, SystemPlanet{PlanetID: planetID, AsteroidBeltIDs: beltIDs, MoonIDs: moonIDs})
	}

	return out, true, nil
}

// PutSystem stores sys in the overlay, keyed by sys.ID -- used by mcp/'s
// read-through fallback. Unlike the sqlite-served path, a read-through fetch
// gets StationIDs and Planets directly from ESI's response, no further
// queries needed.
func (s *Store) PutSystem(sys System) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "system", sys.ID, sys)
	return nil
}

// SystemIDs returns every solar system ID currently cached, ascending.
func (s *Store) SystemIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT solarSystemID FROM mapSolarSystems ORDER BY solarSystemID`)
}
