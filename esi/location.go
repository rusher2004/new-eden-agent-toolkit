// Package esi's location.go covers all 3 of "Location"'s endpoints -- current solar system/
// station/structure, online status, and current ship. All 3 are private/"my own character" --
// the generated request builders take characterId as a plain positional int64, which this facade
// passes as int64(c.characterID), never a caller-supplied parameter.
package esi

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterLocation returns the client's own character's current solar system, and station or
// structure ID if docked/boarded.
func (c *Client) CharacterLocation(ctx context.Context) (*esiapi.CharactersLocation, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdLocation(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character location: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterOnlineStatus is CharacterOnline's return shape, grouped into a struct instead of a
// long multi-value return since 4 nullable/boolean fields is unwieldy as bare return values.
type CharacterOnlineStatus struct {
	Online     bool
	LastLogin  *time.Time
	LastLogout *time.Time
	Logins     *int64
}

// CharacterOnline returns the client's own character's online status and login history.
func (c *Client) CharacterOnline(ctx context.Context) (*CharacterOnlineStatus, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdOnline(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character online status: %w", describeError(err, resp))
	}
	return &CharacterOnlineStatus{
		Online:     result.Online,
		LastLogin:  result.LastLogin,
		LastLogout: result.LastLogout,
		Logins:     result.Logins,
	}, nil
}

// CharacterShip returns the client's own character's current ship.
func (c *Client) CharacterShip(ctx context.Context) (*esiapi.CharactersShip, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdShip(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character ship: %w", describeError(err, resp))
	}
	return result, nil
}
