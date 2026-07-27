// Package esi's freelance_jobs.go covers all 6 of "Freelance Jobs"'s endpoints. 2 are private
// character-scoped ("my own character" -- CharacterFreelanceJobs/CharacterFreelanceJobParticipation
// take no explicit characterID, using int64(c.characterID)); 2 are private corp-scoped,
// cursor-paginated; 2 are public, cursor-paginated. jobId is a string throughout, like Corporation
// Projects' projectId.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterFreelanceJobs returns the client's own character's freelance jobs. Not paginated.
func (c *Client) CharacterFreelanceJobs(ctx context.Context) ([]esiapi.FreelanceJobsDetailFreelancejob, error) {
	result, resp, err := c.api.FreelanceJobsAPI.GetCharactersFreelanceJobsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character freelance jobs: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get character freelance jobs: empty response")
	}
	return result.FreelanceJobs, nil
}

// CharacterFreelanceJobParticipation returns the client's own character's participation in
// jobID. Not paginated.
func (c *Client) CharacterFreelanceJobParticipation(ctx context.Context, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error) {
	result, resp, err := c.api.FreelanceJobsAPI.GetCharactersFreelanceJobsParticipation(ctx, int64(c.characterID), jobID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character freelance job participation: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationFreelanceJobs returns corporationID's freelance jobs, cursor-paginated. after=""
// and before="" each omit their own query param (mutually exclusive); limit=0 omits the limit
// param.
func (c *Client) CorporationFreelanceJobs(ctx context.Context, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	req := c.api.FreelanceJobsAPI.GetCorporationsFreelanceJobsListing(ctx, corporationID)
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
		return nil, nil, fmt.Errorf("get corporation freelance jobs: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get corporation freelance jobs: empty response")
	}
	return result.FreelanceJobs, result.Cursor, nil
}

// CorporationFreelanceJobParticipants returns the participants in corporationID's job jobID,
// cursor-paginated.
func (c *Client) CorporationFreelanceJobParticipants(ctx context.Context, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error) {
	req := c.api.FreelanceJobsAPI.GetCorporationsFreelanceJobsParticipants(ctx, corporationID, jobID)
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
		return nil, nil, fmt.Errorf("get corporation freelance job participants: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get corporation freelance job participants: empty response")
	}
	return result.Participants, result.Cursor, nil
}

// FreelanceJobDetail returns the full details of jobID, including its nested Configuration
// (whose Parameters map values are a 4-variant oneOf -- see mcp/freelance_jobs.go's
// ConfigParameter and toConfigParameter for the flattened shape). Public, no auth required.
func (c *Client) FreelanceJobDetail(ctx context.Context, jobID string) (*esiapi.FreelanceJobsDetail, error) {
	result, resp, err := c.api.FreelanceJobsAPI.GetFreelanceJobsDetail(ctx, jobID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get freelance job detail: %w", describeError(err, resp))
	}
	return result, nil
}

// FreelanceJobsListing returns freelance jobs game-wide, cursor-paginated and optionally
// filtered to one corporation's jobs (corporationID=0 omits the filter). Public, no auth
// required.
func (c *Client) FreelanceJobsListing(ctx context.Context, after, before string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	req := c.api.FreelanceJobsAPI.GetFreelanceJobsListing(ctx)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	if corporationID > 0 {
		req = req.CorporationId(corporationID)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get freelance jobs listing: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get freelance jobs listing: empty response")
	}
	return result.FreelanceJobs, result.Cursor, nil
}
