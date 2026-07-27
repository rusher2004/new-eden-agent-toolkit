// Package esi's fittings.go covers all 3 of "Fittings"'s endpoints -- list, create, and delete a
// character's saved ship fittings. All 3 are private/"my own character" -- the generated request
// builders take characterId as a plain positional int64, which this facade passes as
// int64(c.characterID), never a caller-supplied parameter.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterFittings returns the client's own character's saved ship fittings. Not paginated.
func (c *Client) CharacterFittings(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error) {
	result, resp, err := c.api.FittingsAPI.GetCharactersCharacterIdFittings(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character fittings: %w", describeError(err, resp))
	}
	return result, nil
}

// CreateCharacterFitting saves a new fitting for the client's own character and returns its
// assigned fitting ID -- the only field ESI's response contains.
func (c *Client) CreateCharacterFitting(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error) {
	result, resp, err := c.api.FittingsAPI.PostCharactersCharacterIdFittings(ctx, int64(c.characterID)).
		PostCharactersCharacterIdFittingsRequest(esiapi.PostCharactersCharacterIdFittingsRequest{
			Description: description, Items: items, Name: name, ShipTypeId: shipTypeID,
		}).Execute()
	if err != nil {
		return 0, fmt.Errorf("create character fitting: %w", describeError(err, resp))
	}
	if result == nil {
		return 0, fmt.Errorf("create character fitting: empty response")
	}
	return result.FittingId, nil
}

// DeleteCharacterFitting deletes one of the client's own character's saved fittings. ESI's
// response has no body -- same shape as RespondToCalendarEvent (esi/calendar.go).
func (c *Client) DeleteCharacterFitting(ctx context.Context, fittingID int64) error {
	resp, err := c.api.FittingsAPI.DeleteCharactersCharacterIdFittingsFittingId(ctx, int64(c.characterID), fittingID).Execute()
	if err != nil {
		return fmt.Errorf("delete character fitting: %w", describeError(err, resp))
	}
	return nil
}
