package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Faction is a minimal decode of a fuzzwork chrFactions row. IsUnique is
// *bool, not bool: chrFactions has no uniqueName-equivalent column (checked
// directly via PRAGMA table_info), a documented field-drop in the migration
// design spec, so a sqlite-served Faction always has a nil IsUnique -- a
// plain false would misrepresent the field as a known, confirmed value
// (every real EVE faction name is in fact unique, so the true value is
// almost certainly true, not false). Unlike Region.Description/
// Station.Services, there's no Put-and-overlay path here to mitigate it on
// a cache miss: mcp/universe.go's Factions() has no read-through fallback at
// all (see its full-list-only comment), so a genuinely fresh ESI fetch
// builds the mcp output type directly from ESI's response, never through
// this struct.
type Faction struct {
	ID                   int64
	Name                 string
	Description          string
	IsUnique             *bool
	SizeFactor           float64
	CorporationID        *int64
	MilitiaCorporationID *int64
	SolarSystemID        *int64
	StationCount         int64
	StationSystemCount   int64
}

func scanFactionRow(scanner interface{ Scan(...any) error }) (Faction, error) {
	var (
		id                                          int64
		name                                        sql.NullString
		description                                 sql.NullString
		solarSystemID, corporationID, militiaCorpID sql.NullInt64
		sizeFactor                                  sql.NullFloat64
		stationCount, stationSystemCount            sql.NullInt64
	)
	if err := scanner.Scan(&id, &name, &description, &solarSystemID, &corporationID, &sizeFactor, &stationCount, &stationSystemCount, &militiaCorpID); err != nil {
		return Faction{}, err
	}
	return Faction{
		ID: id, Name: name.String, Description: description.String, SizeFactor: sizeFactor.Float64,
		CorporationID: nullInt(corporationID), MilitiaCorporationID: nullInt(militiaCorpID), SolarSystemID: nullInt(solarSystemID),
		StationCount: stationCount.Int64, StationSystemCount: stationSystemCount.Int64,
	}, nil
}

const factionSelectColumns = `factionID, factionName, description, solarSystemID, corporationID, sizeFactor, stationCount, stationSystemCount, militiaCorporationID`

// Factions returns every cached faction. Returns an empty slice, not an
// error, if the store has never synced successfully -- mcp/'s caller treats
// an empty result as a signal to read-through to a live ESI list call.
func (s *Store) Factions(ctx context.Context, userAgent string) ([]Faction, error) {
	if s == nil {
		return nil, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT `+factionSelectColumns+` FROM chrFactions ORDER BY factionID`)
	if err != nil {
		return nil, fmt.Errorf("query factions: %w", err)
	}
	defer rows.Close()

	var out []Faction
	for rows.Next() {
		f, err := scanFactionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan faction: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Faction looks up a faction by ID. ok is false if the ID isn't in the cache.
func (s *Store) Faction(ctx context.Context, userAgent string, id int64) (Faction, bool, error) {
	if s == nil {
		return Faction{}, false, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Faction{}, false, nil
	}

	row := s.db.QueryRowContext(ctx, `SELECT `+factionSelectColumns+` FROM chrFactions WHERE factionID = ?`, id)
	f, err := scanFactionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Faction{}, false, nil
	}
	if err != nil {
		return Faction{}, false, fmt.Errorf("query faction %d: %w", id, err)
	}
	return f, true, nil
}
