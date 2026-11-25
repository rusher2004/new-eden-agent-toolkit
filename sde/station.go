package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Station is a minimal decode of a fuzzwork staStations row. RaceID is not a
// stored column on staStations at all -- it's joined in from the station's
// own type (invTypes.raceID via stationTypeID), the same value ESI itself
// almost certainly derives it from for NPC stations, but a join rather than
// a stored fact (a semantic shift from the old esi-scrape design's direct
// field, documented in the migration design spec). Services is *[]string,
// not []string: staOperationServices has zero rows in the dump (verified
// directly), so a sqlite-served lookup has no source for it at all -- nil
// means "we don't know," not "confirmed no services," which a bare empty
// slice would misrepresent as a known fact. A cache-miss overlay write can
// still carry a populated slice, same *T-means-"unknown" pattern as
// Region.Description.
type Station struct {
	ID                       int64
	Name                     string
	SystemID                 int64
	TypeID                   int64
	Position                 Position
	MaxDockableShipVolume    float64
	OfficeRentalCost         float64
	Owner                    *int64
	RaceID                   *int64
	ReprocessingEfficiency   float64
	ReprocessingStationsTake float64
	Services                 *[]string
}

// Station looks up a station by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Station(ctx context.Context, userAgent string, id int64) (Station, bool, error) {
	if s == nil {
		return Station{}, false, nil
	}
	if v, ok := getOverlay[Station](s, "station", id); ok {
		return v, true, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return Station{}, false, nil
	}

	var (
		name                                                 sql.NullString
		systemName                                           string
		systemID, typeID                                     int64
		x, y, z                                              sql.NullFloat64
		maxDockable, officeRentalCost, reprocEff, reprocTake sql.NullFloat64
		owner                                                sql.NullInt64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT stationName, `+systemNameSQL("staStations.solarSystemID")+`, solarSystemID, stationTypeID, x, y, z, maxShipVolumeDockable, officeRentalCost, corporationID, reprocessingEfficiency, reprocessingStationsTake
		 FROM staStations WHERE stationID = ?`, id)
	if err := row.Scan(&name, &systemName, &systemID, &typeID, &x, &y, &z, &maxDockable, &officeRentalCost, &owner, &reprocEff, &reprocTake); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Station{}, false, nil
		}
		return Station{}, false, fmt.Errorf("query station %d: %w", id, err)
	}

	out := Station{
		ID: id, Name: gameName(name.String, systemName), SystemID: systemID, TypeID: typeID,
		Position:                 Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
		MaxDockableShipVolume:    maxDockable.Float64,
		OfficeRentalCost:         officeRentalCost.Float64,
		Owner:                    nullInt(owner),
		ReprocessingEfficiency:   reprocEff.Float64,
		ReprocessingStationsTake: reprocTake.Float64,
		// Services stays nil -- no source in the dump, see the doc comment.
	}

	var raceID sql.NullInt64
	raceRow := s.db.QueryRowContext(ctx, `SELECT raceID FROM invTypes WHERE typeID = ?`, typeID)
	if err := raceRow.Scan(&raceID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Station{}, false, fmt.Errorf("query station %d type %d race id: %w", id, typeID, err)
	}
	out.RaceID = nullInt(raceID)

	return out, true, nil
}

// PutStation stores st in the overlay, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStation(st Station) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "station", st.ID, st)
	return nil
}
