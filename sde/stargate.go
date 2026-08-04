package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Stargate is a minimal decode of a fuzzwork mapDenormalize row (groupID 10)
// joined with mapJumps for its destination stargate ID, plus a second query
// against the destination gate's own mapDenormalize row for
// DestinationSystemID (mapJumps only carries the destination *gate* ID, not
// its system) -- both tables official-SDE-derived, replacing the old
// esi-scrape-only source.
type Stargate struct {
	ID                    int64
	Name                  string
	TypeID                int64
	Position              Position
	SystemID              int64
	DestinationStargateID int64
	DestinationSystemID   int64
}

// Stargate looks up a stargate by ID. ok is false if the ID isn't in the
// cache.
func (s *Store) Stargate(ctx context.Context, userAgent string, id int64) (Stargate, bool, error) {
	if s == nil {
		return Stargate{}, false, nil
	}
	if v, ok := getOverlay[Stargate](s, "stargate", id); ok {
		return v, true, nil
	}
	_ = s.ensureSynced(ctx, userAgent)
	if s.db == nil {
		return Stargate{}, false, nil
	}

	var (
		name     sql.NullString
		typeID   int64
		systemID int64
		x, y, z  sql.NullFloat64
	)
	row := s.db.QueryRowContext(ctx,
		`SELECT itemName, typeID, solarSystemID, x, y, z FROM mapDenormalize WHERE itemID = ? AND groupID = 10`, id)
	if err := row.Scan(&name, &typeID, &systemID, &x, &y, &z); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Stargate{}, false, nil
		}
		return Stargate{}, false, fmt.Errorf("query stargate %d: %w", id, err)
	}

	out := Stargate{
		ID: id, Name: name.String, TypeID: typeID, SystemID: systemID,
		Position: Position{X: x.Float64, Y: y.Float64, Z: z.Float64},
	}

	var destinationStargateID sql.NullInt64
	destRow := s.db.QueryRowContext(ctx, `SELECT destinationID FROM mapJumps WHERE stargateID = ?`, id)
	err := destRow.Scan(&destinationStargateID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Stargate{}, false, fmt.Errorf("query stargate %d destination: %w", id, err)
	}
	if err == nil && destinationStargateID.Valid {
		out.DestinationStargateID = destinationStargateID.Int64

		var destinationSystemID sql.NullInt64
		destSystemRow := s.db.QueryRowContext(ctx, `SELECT solarSystemID FROM mapDenormalize WHERE itemID = ?`, destinationStargateID.Int64)
		if err := destSystemRow.Scan(&destinationSystemID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Stargate{}, false, fmt.Errorf("query stargate %d destination system: %w", id, err)
		}
		out.DestinationSystemID = destinationSystemID.Int64
	}

	return out, true, nil
}

// PutStargate stores g in the overlay, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutStargate(g Stargate) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "stargate", g.ID, g)
	return nil
}
