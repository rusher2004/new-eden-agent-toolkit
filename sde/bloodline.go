package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Bloodline is a minimal decode of a fuzzwork chrBloodlines row.
type Bloodline struct {
	ID            int64
	Charisma      int64
	CorporationID int64
	Description   string
	Intelligence  int64
	Memory        int64
	Name          string
	Perception    int64
	RaceID        int64
	ShipTypeID    *int64
	Willpower     int64
}

const bloodlineSelectColumns = `bloodlineID, bloodlineName, raceID, description, shipTypeID, corporationID, perception, willpower, charisma, memory, intelligence`

func scanBloodlineRow(scanner interface{ Scan(...any) error }) (Bloodline, error) {
	var (
		id          int64
		name        sql.NullString
		raceID      sql.NullInt64
		description sql.NullString
		shipTypeID  sql.NullInt64
		// raceID, corporationID, and the five attribute columns are all
		// nullable in the fuzzwork schema even though the produced Bloodline
		// type models them as plain int64 -- scan through sql.NullInt64 so a
		// NULL column (e.g. a partially-seeded row) degrades to the zero
		// value instead of failing the scan outright.
		corporationID, perception, willpower, charisma, memory, intelligence sql.NullInt64
	)
	if err := scanner.Scan(&id, &name, &raceID, &description, &shipTypeID, &corporationID, &perception, &willpower, &charisma, &memory, &intelligence); err != nil {
		return Bloodline{}, err
	}
	return Bloodline{
		ID: id, Name: name.String, RaceID: raceID.Int64, Description: description.String, ShipTypeID: nullInt(shipTypeID),
		CorporationID: corporationID.Int64, Perception: perception.Int64, Willpower: willpower.Int64,
		Charisma: charisma.Int64, Memory: memory.Int64, Intelligence: intelligence.Int64,
	}, nil
}

// Bloodlines returns every cached bloodline. Returns an empty slice, not an
// error, if the store has never synced successfully.
func (s *Store) Bloodlines(ctx context.Context, userAgent string) ([]Bloodline, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT `+bloodlineSelectColumns+` FROM chrBloodlines ORDER BY bloodlineID`)
	if err != nil {
		return nil, fmt.Errorf("query bloodlines: %w", err)
	}
	defer rows.Close()

	var out []Bloodline
	for rows.Next() {
		b, err := scanBloodlineRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bloodline: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Bloodline looks up a bloodline by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Bloodline(ctx context.Context, userAgent string, id int64) (Bloodline, bool, error) {
	if s == nil {
		return Bloodline{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Bloodline{}, false, nil
	}

	row := s.db.QueryRowContext(ctx, `SELECT `+bloodlineSelectColumns+` FROM chrBloodlines WHERE bloodlineID = ?`, id)
	b, err := scanBloodlineRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Bloodline{}, false, nil
	}
	if err != nil {
		return Bloodline{}, false, fmt.Errorf("query bloodline %d: %w", id, err)
	}
	return b, true, nil
}
