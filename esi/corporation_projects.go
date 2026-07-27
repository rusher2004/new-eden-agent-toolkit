// Package esi's corporation_projects.go covers all 4 of "Corporation Projects"'s endpoints:
// contribution, contributors (cursor-paginated), detail, and listing (cursor-paginated). All four
// are private/corp-scoped -- the authenticated character must hold sufficient corp roles, ESI
// itself returns 403 otherwise.
//
// Two things make this tag different from every other esi/ file in this project so far:
//
//  1. projectId is a STRING, not an int64 like every other ID handled elsewhere in this project.
//     Verified from the generated request builders (ApiGetCorporationsProjectsDetailRequest etc.)
//     and every CorporationsProjectsDetail* model's Id field.
//
//  2. GetCorporationsProjectsContribution takes a characterId path/query parameter that is NOT
//     the calling/authenticated character -- Client.characterID is baked into the SSO-backed
//     http.Client and never appears in this call at all. ESI's characterId here names whose
//     contribution to look up, which need not be the caller. This method names that parameter
//     targetCharacterID to keep the two unambiguous.
//
// Contributors and Listing use ESI's cursor-based pagination (after/before/limit query params, a
// Cursor{After, Before *string} in the response body) instead of the page/X-Pages style used
// everywhere else in this project (see esi/contracts.go's totalPagesFromHeader) -- this is a
// genuinely different pagination mechanism, not shoehorned into the old helper. For both methods:
// after="" and before="" each omit their own query param entirely (the two are mutually exclusive
// per ESI's own docs -- callers pick one direction); limit=0 omits the limit query param, letting
// ESI apply its own default. Listing additionally takes state="" to omit the state filter, letting
// ESI apply its own default ("Active").
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CorporationProjectContribution returns targetCharacterID's contribution to corporationID's
// project projectID. targetCharacterID is deliberately distinct from the Client's own
// characterID (used only to authenticate the call) -- it's the character whose contribution ESI
// should report, which need not be the calling character.
func (c *Client) CorporationProjectContribution(ctx context.Context, corporationID int64, projectID string, targetCharacterID int64) (*esiapi.CorporationsProjectsContribution, error) {
	result, resp, err := c.api.CorporationProjectsAPI.GetCorporationsProjectsContribution(ctx, corporationID, projectID, targetCharacterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation project contribution: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationProjectContributors returns the contributors to corporationID's project projectID,
// cursor-paginated. after="" and before="" each omit their own query param (mutually exclusive --
// ESI errors if both are set); limit=0 omits the limit param and lets ESI apply its default. The
// returned *esiapi.Cursor's After/Before pointers (when non-nil) are the cursor values to pass
// back in as this method's own after/before on the next call, to walk forward/backward.
func (c *Client) CorporationProjectContributors(ctx context.Context, corporationID int64, projectID string, after, before string, limit int64) ([]esiapi.CorporationsProjectsContributorsContributor, *esiapi.Cursor, error) {
	req := c.api.CorporationProjectsAPI.GetCorporationsProjectsContributors(ctx, corporationID, projectID)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get corporation project contributors: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get corporation project contributors: empty response")
	}
	return result.Contributors, result.Cursor, nil
}

// CorporationProjectDetail returns the full details of corporationID's project projectID,
// including its polymorphic Configuration (17 variants -- see mcp/corporation_projects.go's
// ProjectConfiguration and its toProjectConfiguration mapper for the flattened shape and the
// full variant-by-variant field mapping).
func (c *Client) CorporationProjectDetail(ctx context.Context, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
	result, resp, err := c.api.CorporationProjectsAPI.GetCorporationsProjectsDetail(ctx, corporationID, projectID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation project detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationProjectListing returns corporationID's projects, cursor-paginated and optionally
// filtered by state. after="" and before="" each omit their own query param (mutually exclusive);
// limit=0 omits the limit param; state="" omits the state param and lets ESI apply its own
// default ("Active").
func (c *Client) CorporationProjectListing(ctx context.Context, corporationID int64, after, before string, limit int64, state string) ([]esiapi.CorporationsProjectsDetailProject, *esiapi.Cursor, error) {
	req := c.api.CorporationProjectsAPI.GetCorporationsProjectsListing(ctx, corporationID)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	if state != "" {
		req = req.State(state)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get corporation project listing: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get corporation project listing: empty response")
	}
	return result.Projects, result.Cursor, nil
}
