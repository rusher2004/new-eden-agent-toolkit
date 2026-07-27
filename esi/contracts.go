package esi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// totalPagesFromHeader parses the X-Pages response header, defaulting to 1 if missing or
// unparseable -- ESI's own convention for "this response isn't paginated" or a malformed header.
func totalPagesFromHeader(resp *http.Response) int32 {
	if resp == nil {
		return 1
	}
	n, err := strconv.Atoi(resp.Header.Get("X-Pages"))
	if err != nil || n < 1 {
		return 1
	}
	return int32(n)
}

// CharacterContracts returns the client's character's contracts. page=0 omits the page query
// parameter entirely, letting ESI default to page 1.
func (c *Client) CharacterContracts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error) {
	req := c.api.ContractsAPI.GetCharactersCharacterIdContracts(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character contracts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CharacterContractBids returns the bids on one of the client's character's contracts. Not paginated.
func (c *Client) CharacterContractBids(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error) {
	result, resp, err := c.api.ContractsAPI.GetCharactersCharacterIdContractsContractIdBids(ctx, int64(c.characterID), contractID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character contract bids: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterContractItems returns the items on one of the client's character's contracts. Not paginated.
func (c *Client) CharacterContractItems(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
	result, resp, err := c.api.ContractsAPI.GetCharactersCharacterIdContractsContractIdItems(ctx, int64(c.characterID), contractID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character contract items: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationContracts returns corporationID's contracts. The authenticated character must hold
// sufficient corp roles — ESI itself returns 403 otherwise. page=0 omits the page query parameter.
func (c *Client) CorporationContracts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error) {
	req := c.api.ContractsAPI.GetCorporationsCorporationIdContracts(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation contracts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationContractBids returns the bids on one of corporationID's contracts. Note: the
// generated call takes (contractID, corporationID) — reversed from this method's own
// (corporationID, contractID) signature and from the URL path — verified from source.
func (c *Client) CorporationContractBids(ctx context.Context, corporationID int64, contractID int64, page int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error) {
	req := c.api.ContractsAPI.GetCorporationsCorporationIdContractsContractIdBids(ctx, contractID, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation contract bids: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationContractItems returns the items on one of corporationID's contracts. Same reversed
// generated-call parameter order as CorporationContractBids. Not paginated.
func (c *Client) CorporationContractItems(ctx context.Context, corporationID int64, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
	result, resp, err := c.api.ContractsAPI.GetCorporationsCorporationIdContractsContractIdItems(ctx, contractID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation contract items: %w", describeError(err, resp))
	}
	return result, nil
}

// PublicContracts returns public contracts in regionID. page=0 omits the page query parameter.
func (c *Client) PublicContracts(ctx context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error) {
	req := c.api.ContractsAPI.GetContractsPublicRegionId(ctx, regionID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get public contracts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// PublicContractBids returns the bids on a public contract. page=0 omits the page query parameter.
func (c *Client) PublicContractBids(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error) {
	req := c.api.ContractsAPI.GetContractsPublicBidsContractId(ctx, contractID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get public contract bids: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// PublicContractItems returns the items on a public contract. page=0 omits the page query parameter.
func (c *Client) PublicContractItems(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error) {
	req := c.api.ContractsAPI.GetContractsPublicItemsContractId(ctx, contractID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get public contract items: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
