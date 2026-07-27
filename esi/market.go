// Package esi's market.go covers all 11 of "Market"'s endpoints: character/corp own market
// orders and order history (character/corp endpoints are "my own character"/corp-scoped exactly
// like esi/industry.go and esi/contracts.go), plus public market-group listing/detail,
// adjusted/average prices, per-region historical stats, per-region live orders and active type
// IDs, and one endpoint that looks public but isn't -- GetMarketsStructuresStructureId requires
// OAuth2 scope esi-markets.structure_markets.v1 (verified in esi/api/openapi.yaml), so it's
// modeled as "my own character" auth exactly like CharacterFittings, not NewPublicClient.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterMarketOrders returns the client's own character's open market orders. Not paginated --
// verified from source: ApiGetCharactersCharacterIdOrdersRequest has no Page() setter, unlike its
// History sibling below.
func (c *Client) CharacterMarketOrders(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
	result, resp, err := c.api.MarketAPI.GetCharactersCharacterIdOrders(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character market orders: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMarketOrderHistory returns the client's own character's cancelled/expired market
// orders from the last 90 days. page=0 omits the page query parameter.
func (c *Client) CharacterMarketOrderHistory(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
	req := c.api.MarketAPI.GetCharactersCharacterIdOrdersHistory(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character market order history: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMarketOrders returns corporationID's open market orders. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise. page=0 omits the page query
// parameter.
func (c *Client) CorporationMarketOrders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
	req := c.api.MarketAPI.GetCorporationsCorporationIdOrders(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation market orders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMarketOrderHistory returns corporationID's cancelled/expired market orders from the
// last 90 days. Same corp-role enforcement as CorporationMarketOrders. page=0 omits the page query
// parameter.
func (c *Client) CorporationMarketOrderHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
	req := c.api.MarketAPI.GetCorporationsCorporationIdOrdersHistory(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation market order history: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// MarketGroups returns every market group ID in the game. Public, not paginated.
func (c *Client) MarketGroups(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsGroups(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market groups: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketGroup returns detail for one market group. Public, not paginated.
func (c *Client) MarketGroup(ctx context.Context, marketGroupID int64) (*esiapi.MarketsGroupsMarketGroupIdGet, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsGroupsMarketGroupId(ctx, marketGroupID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market group: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketPrices returns ESI's adjusted/average price for every priced type in the game. Public,
// not paginated.
func (c *Client) MarketPrices(ctx context.Context) ([]esiapi.MarketsPricesGetInner, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsPrices(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market prices: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketHistory returns daily historical market statistics for one type in one region. Public,
// not paginated. typeID is a required query parameter -- verified from source:
// GetMarketsRegionIdHistoryExecute explicitly errors ("typeId is required and must be specified")
// if it's left unset, unlike the optional TypeId filter on MarketOrders below.
func (c *Client) MarketHistory(ctx context.Context, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
	result, resp, err := c.api.MarketAPI.GetMarketsRegionIdHistory(ctx, regionID).TypeId(typeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get market history: %w", describeError(err, resp))
	}
	return result, nil
}

// MarketOrders returns live orders in a region, optionally filtered to one type. Public, IS
// paginated (page=0 omits the page query parameter). orderType is required -- verified from
// source: GetMarketsRegionIdOrdersExecute explicitly errors ("orderType is required and must be
// specified") if left unset, even though ESI's own spec lists a default ("all"); callers must
// always pass one of "buy", "sell", or "all" explicitly. typeID=0 omits the optional type_id
// filter, returning orders for every type in the region.
func (c *Client) MarketOrders(ctx context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
	req := c.api.MarketAPI.GetMarketsRegionIdOrders(ctx, regionID).OrderType(orderType)
	if typeID > 0 {
		req = req.TypeId(typeID)
	}
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get market orders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// MarketTypes returns the type IDs with at least one active order in a region, for efficient
// market indexing. Public, IS paginated. page=0 omits the page query parameter.
func (c *Client) MarketTypes(ctx context.Context, regionID int64, page int32) ([]int64, int32, error) {
	req := c.api.MarketAPI.GetMarketsRegionIdTypes(ctx, regionID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get market types: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// StructureMarketOrders returns live orders in a player structure's market. Despite resembling
// MarketOrders (region orders), this is NOT public -- verified in esi/api/openapi.yaml: it
// requires OAuth2 scope esi-markets.structure_markets.v1, i.e. the querying character must have
// docking access to the structure. Modeled as "my own character" auth, same pattern as
// CharacterFittings -- the caller-supplied structureID is the resource, c.characterID
// authenticates. IS paginated; page=0 omits the page query parameter.
func (c *Client) StructureMarketOrders(ctx context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
	req := c.api.MarketAPI.GetMarketsStructuresStructureId(ctx, structureID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get structure market orders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
