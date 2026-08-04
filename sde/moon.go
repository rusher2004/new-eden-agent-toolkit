package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Moon is a minimal decode of a fuzzwork mapDenormalize row filtered to
// groupID 8 (Moon). See Planet's doc comment for why mapDenormalize replaces
// the old esi-scrape source.
type Moon struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

// Moon looks up a moon by ID. ok is false if the ID isn't in the cache.
func (s *Store) Moon(ctx context.Context, userAgent string, id int64) (Moon, bool, error) {
	if s == nil {
		return Moon{}, false, nil
	}
	if v, ok := getOverlay[Moon](s, "moon", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Moon{}, false, nil
	}

	var (
		name     sql.NullString
		systemID int64
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID = 8`, id)
	if err := row.Scan(&name, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Moon{}, false, nil
		}
		return Moon{}, false, fmt.Errorf("query moon %d: %w", id, err)
	}

	return Moon{ID: id, Name: name.String, SystemID: systemID, Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64}}, true, nil
}

// PutMoon stores m in the overlay, keyed by m.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutMoon(m Moon) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "moon", m.ID, m)
	return nil
}
