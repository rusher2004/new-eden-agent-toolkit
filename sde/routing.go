package sde

import (
	"context"
	"database/sql"
	"fmt"
)

// RoutingSystem is a solar system with what routing needs.
type RoutingSystem struct {
	ID                int64
	Name              string
	ConstellationID   int64
	ConstellationName string
	RegionID          int64
	RegionName        string
	Security          float64
	X, Y, Z           float64
	HasNPCStation     bool
}

// RoutingSystems returns every solar system with its position, security and
// region, and whether it has an NPC station.
func (s *Store) RoutingSystems(ctx context.Context, userAgent string) ([]RoutingSystem, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	var out []RoutingSystem
	err := s.eachRow(ctx, `SELECT s.solarSystemID, COALESCE(s.solarSystemName, ''), s.constellationID, COALESCE(c.constellationName, ''),
			COALESCE(c.regionID, 0), COALESCE(r.regionName, ''), COALESCE(s.security, 0), COALESCE(s.x, 0), COALESCE(s.y, 0), COALESCE(s.z, 0),
			EXISTS (SELECT 1 FROM staStations st WHERE st.solarSystemID = s.solarSystemID)
		FROM mapSolarSystems s
		LEFT JOIN mapConstellations c ON c.constellationID = s.constellationID
		LEFT JOIN mapRegions r ON r.regionID = c.regionID
		ORDER BY s.solarSystemID`, nil, func(r *sql.Rows) error {
		var rs RoutingSystem
		if err := r.Scan(&rs.ID, &rs.Name, &rs.ConstellationID, &rs.ConstellationName, &rs.RegionID, &rs.RegionName, &rs.Security, &rs.X, &rs.Y, &rs.Z, &rs.HasNPCStation); err != nil {
			return err
		}
		out = append(out, rs)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("routing systems: %w", err)
	}
	return out, nil
}

// StargateConnections returns every system-to-system stargate link (both
// directions appear as separate rows in the SDE).
func (s *Store) StargateConnections(ctx context.Context, userAgent string) ([][2]int64, error) {
	if err := s.requireDB(ctx, userAgent); err != nil {
		return nil, err
	}
	var out [][2]int64
	err := s.eachRow(ctx, `SELECT fromSolarSystemID, toSolarSystemID FROM mapSolarSystemJumps`, nil, func(r *sql.Rows) error {
		var e [2]int64
		if err := r.Scan(&e[0], &e[1]); err != nil {
			return err
		}
		out = append(out, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stargate connections: %w", err)
	}
	return out, nil
}
