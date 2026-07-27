// Package esi's search.go covers the single "Search" endpoint -- search for entities matching a
// sub-string, scoped to the client's own character (private, "my own character").
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterSearch searches for entities matching search, restricted to categories (1-11 of:
// agent, alliance, character, constellation, corporation, faction, inventory_type, region,
// solar_system, station, structure -- verified against esi/api/openapi.yaml's enum). Both
// categories and search are required at the ESI-request level -- verified from source: the
// generated Execute body errors if either is unset. strict=false requests a substring match;
// strict=true requires an exact match.
func (c *Client) CharacterSearch(ctx context.Context, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
	req := c.api.SearchAPI.GetCharactersCharacterIdSearch(ctx, int64(c.characterID)).Categories(categories).Search(search)
	if strict {
		req = req.Strict(strict)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get character search: %w", describeError(err, resp))
	}
	return result, nil
}
