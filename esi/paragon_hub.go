// Package esi's paragon_hub.go covers all 5 of "Paragon Hub"'s endpoints (new in compatibility
// date 2026-08-18) -- the in-game marketplace for SKINR licenses. Every endpoint is
// cursor-paginated (limit 10-100, ESI default 10). 1 is public; the other 4 need the new-style
// esi.cosmetic.char:read scope: the client's own character's posted listings, plus listings
// visible to a given alliance, corporation, or the client's own character. Every listing's Price
// is a 2-variant oneOf (ISK or PLEX); only a character's own posted listings carry a Target.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// ParagonHubSkinrListings returns the SKINR listings publicly available on the Paragon Hub,
// cursor-paginated. Public, no auth required.
func (c *Client) ParagonHubSkinrListings(ctx context.Context, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	result, resp, err := applyCursor(c.api.ParagonHubAPI.GetParagonHubSkinr(ctx), after, before, limit).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings: empty response")
	}
	return result.Listings, result.Cursor, nil
}

// CharacterParagonHubSkinrListings returns the SKINR listings the client's own character has
// posted on the Paragon Hub, cursor-paginated.
func (c *Client) CharacterParagonHubSkinrListings(ctx context.Context, after, before string, limit int64) ([]esiapi.CharactersParagonHubSkinrItem, *esiapi.Cursor, error) {
	result, resp, err := applyCursor(c.api.ParagonHubAPI.GetCharactersParagonHubSkinr(ctx, int64(c.characterID)), after, before, limit).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get character paragon hub skinr listings: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get character paragon hub skinr listings: empty response")
	}
	return result.Listings, result.Cursor, nil
}

// ParagonHubSkinrListingsForAlliance returns the SKINR listings visible to allianceID,
// cursor-paginated.
func (c *Client) ParagonHubSkinrListingsForAlliance(ctx context.Context, allianceID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	result, resp, err := applyCursor(c.api.ParagonHubAPI.GetParagonHubSkinrAlliances(ctx, allianceID), after, before, limit).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings for alliance: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings for alliance: empty response")
	}
	return result.Listings, result.Cursor, nil
}

// ParagonHubSkinrListingsForCorporation returns the SKINR listings visible to corporationID,
// cursor-paginated.
func (c *Client) ParagonHubSkinrListingsForCorporation(ctx context.Context, corporationID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	result, resp, err := applyCursor(c.api.ParagonHubAPI.GetParagonHubSkinrCorporations(ctx, corporationID), after, before, limit).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings for corporation: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings for corporation: empty response")
	}
	return result.Listings, result.Cursor, nil
}

// ParagonHubSkinrListingsForCharacter returns the SKINR listings visible to the client's own
// character (listings other sellers targeted at it), cursor-paginated. Distinct from
// CharacterParagonHubSkinrListings, which is the listings the character itself posted.
func (c *Client) ParagonHubSkinrListingsForCharacter(ctx context.Context, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	result, resp, err := applyCursor(c.api.ParagonHubAPI.GetParagonHubSkinrCharacters(ctx, int64(c.characterID)), after, before, limit).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings for character: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get paragon hub skinr listings for character: empty response")
	}
	return result.Listings, result.Cursor, nil
}
