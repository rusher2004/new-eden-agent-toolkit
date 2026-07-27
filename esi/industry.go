// Package esi's industry.go covers all 8 of "Industry"'s endpoints: character/corp industry
// jobs, character/corp mining ledgers/extractions/observers, and public facility/system data.
// Back to the classic page/X-Pages pagination style (esi/contracts.go's totalPagesFromHeader) --
// not cursor-based, unlike the two most recent batches (Corporation Projects, Freelance Jobs).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterIndustryJobs returns the client's own character's industry jobs. Not paginated.
// includeCompleted is always sent explicitly -- a genuine boolean filter, unlike page which is
// conditionally omitted.
func (c *Client) CharacterIndustryJobs(ctx context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
	result, resp, err := c.api.IndustryAPI.GetCharactersCharacterIdIndustryJobs(ctx, int64(c.characterID)).IncludeCompleted(includeCompleted).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character industry jobs: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMining returns the client's own character's mining ledger. page=0 omits the page
// query parameter.
func (c *Client) CharacterMining(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCharactersCharacterIdMining(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character mining: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMiningExtractions returns corporationID's moon mining extractions. page=0 omits
// the page query parameter.
func (c *Client) CorporationMiningExtractions(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationCorporationIdMiningExtractions(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation mining extractions: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMiningObservers returns corporationID's mining observers (structures/entities that
// recorded mining activity). page=0 omits the page query parameter.
func (c *Client) CorporationMiningObservers(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationCorporationIdMiningObservers(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation mining observers: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMiningObserverDetail returns the mining ledger recorded by one observer. page=0
// omits the page query parameter.
func (c *Client) CorporationMiningObserverDetail(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationCorporationIdMiningObserversObserverId(ctx, corporationID, observerID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation mining observer detail: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationIndustryJobs returns corporationID's industry jobs. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise. page=0 omits the page
// query parameter; includeCompleted is always sent explicitly.
func (c *Client) CorporationIndustryJobs(ctx context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationsCorporationIdIndustryJobs(ctx, corporationID).IncludeCompleted(includeCompleted)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation industry jobs: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// IndustryFacilities returns every industry facility in the game. Public, not paginated.
func (c *Client) IndustryFacilities(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error) {
	result, resp, err := c.api.IndustryAPI.GetIndustryFacilities(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get industry facilities: %w", describeError(err, resp))
	}
	return result, nil
}

// IndustrySystems returns industry cost indices for every solar system. Public, not paginated.
func (c *Client) IndustrySystems(ctx context.Context) ([]esiapi.IndustrySystemsGetInner, error) {
	result, resp, err := c.api.IndustryAPI.GetIndustrySystems(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get industry systems: %w", describeError(err, resp))
	}
	return result, nil
}
