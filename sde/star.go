package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Star is a minimal decode of a fuzzwork mapDenormalize row (groupID 6, Sun)
// joined with mapCelestialStatistics for the physical attributes
// mapDenormalize itself doesn't carry (age/luminosity/spectralClass/
// temperature) -- both tables are official-SDE-derived, replacing the old
// esi-scrape-only source. Age/Radius/Temperature are truncated to int64 from
// mapCelestialStatistics's FLOAT columns, matching this type's existing
// (pre-migration) field types.
type Star struct {
	ID            int64
	Age           int64
	Luminosity    float64
	Name          string
	Radius        int64
	SolarSystemID int64
	SpectralClass string
	Temperature   int64
	TypeID        int64
}

// Star looks up a star by ID. ok is false if the ID isn't in the cache. A
// row missing from mapCelestialStatistics (shouldn't happen for a real sun,
// but not asserted) leaves the physical-attribute fields at their zero
// values rather than failing the whole lookup.
func (s *Store) Star(ctx context.Context, userAgent string, id int64) (Star, bool, error) {
	if s == nil {
		return Star{}, false, nil
	}
	if v, ok := getOverlay[Star](s, "star", id); ok {
		return v, true, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return Star{}, false, nil
	}

	var (
		name          sql.NullString
		typeID        int64
		solarSystemID int64
	)
	row := s.db.QueryRowContext(ctx, `SELECT itemName, typeID, solarSystemID FROM mapDenormalize WHERE itemID = ? AND groupID = 6`, id)
	if err := row.Scan(&name, &typeID, &solarSystemID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Star{}, false, nil
		}
		return Star{}, false, fmt.Errorf("query star %d: %w", id, err)
	}

	out := Star{ID: id, Name: name.String, TypeID: typeID, SolarSystemID: solarSystemID}

	var (
		temperature, luminosity, age, radius sql.NullFloat64
		spectralClass                        sql.NullString
	)
	statsRow := s.db.QueryRowContext(ctx,
		`SELECT temperature, spectralClass, luminosity, age, radius FROM mapCelestialStatistics WHERE celestialID = ?`, id)
	err := statsRow.Scan(&temperature, &spectralClass, &luminosity, &age, &radius)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Star{}, false, fmt.Errorf("query star %d celestial statistics: %w", id, err)
	}
	if err == nil {
		out.Temperature = int64(temperature.Float64)
		out.SpectralClass = spectralClass.String
		out.Luminosity = luminosity.Float64
		out.Age = int64(age.Float64)
		out.Radius = int64(radius.Float64)
	}

	return out, true, nil
}

// PutStar stores st in the overlay, keyed by st.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStar(st Star) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "star", st.ID, st)
	return nil
}
