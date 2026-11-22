package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// AsteroidBelt is a minimal decode of a fuzzwork mapDenormalize row filtered
// to groupID 9 or 4430 (both observed as "Asteroid Belt" in invGroups --
// 4430 covers wormhole-system belts). Unlike every other sde/ resource,
// there is no FC SDE jsonl source for this data at all even historically --
// asteroid belt names are computed server-side by FC. mapDenormalize
// carries it directly, replacing the old esi-scrape-only source.
type AsteroidBelt struct {
	ID       int64
	Name     string
	Position Position
	SystemID int64
}

// AsteroidBelt looks up an asteroid belt by ID. ok is false if the ID isn't
// in the cache.
func (s *Store) AsteroidBelt(ctx context.Context, userAgent string, id int64) (AsteroidBelt, bool, error) {
	if s == nil {
		return AsteroidBelt{}, false, nil
	}
	if v, ok := getOverlay[AsteroidBelt](s, "asteroidBelt", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return AsteroidBelt{}, false, nil
	}

	var (
		name       sql.NullString
		systemName string
		systemID   int64
		x, y, z    sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, `+systemNameSQL("mapDenormalize.solarSystemID")+`, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID IN (9, 4430)`, id)
	if err := row.Scan(&name, &systemName, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AsteroidBelt{}, false, nil
		}
		return AsteroidBelt{}, false, fmt.Errorf("query asteroid belt %d: %w", id, err)
	}

	return AsteroidBelt{ID: id, Name: gameName(name.String, systemName), SystemID: systemID, Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64}}, true, nil
}

// PutAsteroidBelt stores b in the overlay, keyed by b.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutAsteroidBelt(b AsteroidBelt) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "asteroidBelt", b.ID, b)
	return nil
}
