package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Race is a minimal decode of a fuzzwork chrRaces row, augmented at read
// time with AllianceID from EVE Ref's esi-scrape dataset (in-memory only --
// see sync.go) -- chrRaces has no such column, and there is no alliances
// table anywhere in the fuzzwork dump either.
type Race struct {
	ID          int64
	AllianceID  *int64
	Description string
	Name        string
}

// Races returns every cached race. Returns an empty slice, not an error, if
// the store has never synced successfully.
func (s *Store) Races(ctx context.Context, userAgent string) (_ []Race, err error) {
	if s == nil {
		return nil, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT raceID, raceName, description FROM chrRaces ORDER BY raceID`)
	if err != nil {
		return nil, fmt.Errorf("query races: %w", err)
	}
	defer closeInto(&err, rows, "rows")

	var out []Race
	for rows.Next() {
		var (
			id          int64
			name        sql.NullString
			description sql.NullString
		)
		if err := rows.Scan(&id, &name, &description); err != nil {
			return nil, fmt.Errorf("scan race: %w", err)
		}
		r := Race{ID: id, Name: name.String, Description: description.String}
		if allianceID, ok := s.raceAlliances[id]; ok {
			r.AllianceID = &allianceID
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Race looks up a race by ID. ok is false if the ID isn't in the cache.
func (s *Store) Race(ctx context.Context, userAgent string, id int64) (Race, bool, error) {
	if s == nil {
		return Race{}, false, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return Race{}, false, nil
	}

	var (
		name        sql.NullString
		description sql.NullString
	)
	row := s.db.QueryRowContext(ctx, `SELECT raceName, description FROM chrRaces WHERE raceID = ?`, id)
	if err := row.Scan(&name, &description); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Race{}, false, nil
		}
		return Race{}, false, fmt.Errorf("query race %d: %w", id, err)
	}

	out := Race{ID: id, Name: name.String, Description: description.String}
	if allianceID, ok := s.raceAlliances[id]; ok {
		out.AllianceID = &allianceID
	}
	return out, true, nil
}
