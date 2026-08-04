package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Region is a minimal decode of a fuzzwork mapRegions row -- the top level of
// EVE's geography (e.g. "Derelik"). Description is *string, not string: the
// fuzzwork dump's mapRegions table has no description column at all (checked
// directly -- regionID/regionName/x/y/z/xMin.../factionID/nebula/radius,
// nothing else), so every sqlite-served Region has a nil Description. A
// cache-miss ESI fetch's response still includes it, though (ESI's own
// schema carries it), so the overlay can still hold a populated value for
// that process's lifetime -- see docs/superpowers/specs/2026-08-04-sde-sqlite-migration-design.md's
// "Exception: Region.Description stays *string" section.
type Region struct {
	ID               int64
	Name             string
	Description      *string
	ConstellationIDs []int64
}

// Region looks up a region by ID. ok is false if the ID isn't in the cache.
func (s *Store) Region(ctx context.Context, userAgent string, id int64) (Region, bool, error) {
	if s == nil {
		return Region{}, false, nil
	}
	if v, ok := getOverlay[Region](s, "region", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Region{}, false, nil
	}

	var name sql.NullString
	row := s.db.QueryRowContext(ctx, `SELECT regionName FROM mapRegions WHERE regionID = ?`, id)
	if err := row.Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Region{}, false, nil
		}
		return Region{}, false, fmt.Errorf("query region %d: %w", id, err)
	}

	constellationIDs, err := s.queryInt64Column(ctx, `SELECT constellationID FROM mapConstellations WHERE regionID = ? ORDER BY constellationID`, id)
	if err != nil {
		return Region{}, false, fmt.Errorf("query region %d constellation ids: %w", id, err)
	}

	return Region{ID: id, Name: name.String, ConstellationIDs: constellationIDs}, true, nil
}

// PutRegion stores r in the overlay, keyed by r.ID -- used by mcp/'s
// read-through fallback to cache a region fetched live from ESI. Unlike a
// sqlite-served Region, an overlaid one may carry a non-nil Description --
// see Region's doc comment.
func (s *Store) PutRegion(r Region) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "region", r.ID, r)
	return nil
}

// RegionIDs returns every region ID currently cached, ascending.
func (s *Store) RegionIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT regionID FROM mapRegions ORDER BY regionID`)
}
