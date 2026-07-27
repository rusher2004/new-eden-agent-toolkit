package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// AccessLists returns the client's character's structure/fleet access lists
// (bare IDs only — call AccessListDetail for name/description/membership).
func (c *Client) AccessLists(ctx context.Context) ([]esiapi.CharactersAccessListsListingAccesslist, error) {
	result, resp, err := c.api.AccessListAPI.GetCharactersAccessListsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get access lists: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get access lists: empty response")
	}
	return result.AccessLists, nil
}

// AccessListDetail returns the full detail (description, membership) for one access list.
func (c *Client) AccessListDetail(ctx context.Context, accessListID int64) (*esiapi.CharactersAccessListsDetail, error) {
	result, resp, err := c.api.AccessListAPI.GetCharactersAccessListsDetail(ctx, accessListID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get access list detail: %w", describeError(err, resp))
	}
	return result, nil
}
