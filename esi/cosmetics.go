// Package esi's cosmetics.go covers all 3 of "Cosmetics"'s endpoints (new in compatibility date
// 2026-08-18). 2 are private character-scoped ("my own character" -- no explicit characterID,
// using int64(c.characterID)) and need the new-style esi.cosmetic.char:read scope; 1 is public.
// skinrID is a hex string, not an integer.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterSkinrLicenses returns the SKINR licenses the client's own character owns. Not
// paginated.
func (c *Client) CharacterSkinrLicenses(ctx context.Context) ([]esiapi.CharactersCosmeticsSkinrItem, error) {
	result, resp, err := c.api.CosmeticsAPI.GetCharactersCosmeticsSkinr(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character skinr licenses: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get character skinr licenses: empty response")
	}
	return result.Licenses, nil
}

// CharacterSkinrComponentLicenses returns the SKINR component licenses (nanocoatings, patterns,
// ...) the client's own character owns. Not paginated. Each license's Runs is a 2-variant oneOf
// (remaining count, or unlimited) -- see mcp/cosmetics.go for the flattened shape.
func (c *Client) CharacterSkinrComponentLicenses(ctx context.Context) ([]esiapi.CharactersCosmeticsSkinrComponentsItem, error) {
	result, resp, err := c.api.CosmeticsAPI.GetCharactersCosmeticsSkinrComponents(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character skinr component licenses: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get character skinr component licenses: empty response")
	}
	return result.Licenses, nil
}

// SkinrDetail returns every attribute of one SKINR: its ship type, tier, and per-slot layout
// (each slot's Configuration a 2-variant oneOf -- nanocoating or pattern). Public, no auth
// required.
func (c *Client) SkinrDetail(ctx context.Context, skinrID string) (*esiapi.CosmeticsSkinr, error) {
	result, resp, err := c.api.CosmeticsAPI.GetCosmeticsSkinr(ctx, skinrID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get skinr detail: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get skinr detail: empty response")
	}
	return result, nil
}
