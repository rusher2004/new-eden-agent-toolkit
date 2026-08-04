package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Graphic is a minimal decode of a fuzzwork eveGraphics row, augmented at
// read time with SofDNA from EVE Ref's esi-scrape dataset (in-memory only --
// see sync.go). CollisionFile is deliberately not modeled -- confirmed
// absent from both the fuzzwork dump and the esi-scrape mirror of ESI's own
// live responses.
type Graphic struct {
	ID             int64
	GraphicFile    *string
	IconFolder     *string
	SofDNA         *string
	SofFactionName *string
	SofHullName    *string
	SofRaceName    *string
}

// Graphic looks up a graphic by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Graphic(ctx context.Context, userAgent string, id int64) (Graphic, bool, error) {
	if s == nil {
		return Graphic{}, false, nil
	}
	if v, ok := getOverlay[Graphic](s, "graphic", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Graphic{}, false, nil
	}

	var graphicFile, iconFolder, sofFactionName, sofHullName, sofRaceName sql.NullString
	row := s.db.QueryRowContext(ctx,
		`SELECT graphicFile, iconFolder, sofFactionName, sofHullName, sofRaceName FROM eveGraphics WHERE graphicID = ?`, id)
	if err := row.Scan(&graphicFile, &iconFolder, &sofFactionName, &sofHullName, &sofRaceName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Graphic{}, false, nil
		}
		return Graphic{}, false, fmt.Errorf("query graphic %d: %w", id, err)
	}

	out := Graphic{ID: id}
	if graphicFile.Valid {
		out.GraphicFile = &graphicFile.String
	}
	if iconFolder.Valid {
		out.IconFolder = &iconFolder.String
	}
	if sofFactionName.Valid {
		out.SofFactionName = &sofFactionName.String
	}
	if sofHullName.Valid {
		out.SofHullName = &sofHullName.String
	}
	if sofRaceName.Valid {
		out.SofRaceName = &sofRaceName.String
	}
	if dna, ok := s.graphicSofDNA[id]; ok {
		out.SofDNA = &dna
	}

	return out, true, nil
}

// PutGraphic stores g in the overlay, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGraphic(g Graphic) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "graphic", g.ID, g)
	return nil
}

// GraphicIDs returns every graphic ID currently cached, ascending.
func (s *Store) GraphicIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT graphicID FROM eveGraphics ORDER BY graphicID`)
}
