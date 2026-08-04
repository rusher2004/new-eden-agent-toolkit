package sde

import (
	"context"
	"database/sql"
	"fmt"
)

// Ancestry is a minimal decode of a fuzzwork chrAncestries row.
type Ancestry struct {
	ID               int64
	BloodlineID      int64
	Description      string
	Name             string
	IconID           *int64
	ShortDescription *string
}

// Ancestries returns every cached ancestry. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Ancestries(ctx context.Context, userAgent string) ([]Ancestry, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT ancestryID, ancestryName, bloodlineID, description, iconID, shortDescription FROM chrAncestries ORDER BY ancestryID`)
	if err != nil {
		return nil, fmt.Errorf("query ancestries: %w", err)
	}
	defer rows.Close()

	var out []Ancestry
	for rows.Next() {
		var (
			id, bloodlineID  int64
			name             sql.NullString
			description      sql.NullString
			iconID           sql.NullInt64
			shortDescription sql.NullString
		)
		if err := rows.Scan(&id, &name, &bloodlineID, &description, &iconID, &shortDescription); err != nil {
			return nil, fmt.Errorf("scan ancestry: %w", err)
		}
		a := Ancestry{ID: id, Name: name.String, BloodlineID: bloodlineID, Description: description.String, IconID: nullInt(iconID)}
		if shortDescription.Valid {
			a.ShortDescription = &shortDescription.String
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
