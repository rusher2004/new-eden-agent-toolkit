package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Assets returns the client's character's assets (bare IDs — call AssetLocations/AssetNames
// to resolve a subset of item IDs into positions or player-assigned names). page=0 omits the
// page query parameter.
func (c *Client) Assets(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
	req := c.api.AssetsAPI.GetCharactersCharacterIdAssets(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get assets: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// AssetLocations resolves a set of item IDs (from Assets) into their positions.
func (c *Client) AssetLocations(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCharactersCharacterIdAssetsLocations(ctx, int64(c.characterID)).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get asset locations: %w", describeError(err, resp))
	}
	return result, nil
}

// AssetNames resolves a set of item IDs (from Assets) into their player-assigned custom names
// (e.g. a renamed ship or container) — not an SDE-backed item-type name lookup.
func (c *Client) AssetNames(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCharactersCharacterIdAssetsNames(ctx, int64(c.characterID)).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get asset names: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssets returns corporationID's assets. The authenticated character must hold
// sufficient corp roles — ESI itself returns 403 otherwise; no client-side check needed.
// page=0 omits the page query parameter.
func (c *Client) CorporationAssets(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error) {
	req := c.api.AssetsAPI.GetCorporationsCorporationIdAssets(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation assets: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationAssetLocations resolves a set of corporation item IDs into their positions.
func (c *Client) CorporationAssetLocations(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCorporationsCorporationIdAssetsLocations(ctx, corporationID).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation asset locations: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssetNames resolves a set of corporation item IDs into their player-assigned names.
func (c *Client) CorporationAssetNames(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCorporationsCorporationIdAssetsNames(ctx, corporationID).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation asset names: %w", describeError(err, resp))
	}
	return result, nil
}
