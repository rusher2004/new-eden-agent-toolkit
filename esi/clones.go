package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterClones returns the client's character's jump clones and home location.
func (c *Client) CharacterClones(ctx context.Context) (*esiapi.CharactersCharacterIdClonesGet, error) {
	result, resp, err := c.api.ClonesAPI.GetCharactersCharacterIdClones(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character clones: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterImplants returns the client's character's currently active implant type IDs.
func (c *Client) CharacterImplants(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.ClonesAPI.GetCharactersCharacterIdImplants(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character implants: %w", describeError(err, resp))
	}
	return result, nil
}
