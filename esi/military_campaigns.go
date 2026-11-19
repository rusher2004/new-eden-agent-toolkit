// Package esi's military_campaigns.go covers all 6 of "Military Campaigns"'s endpoints -- the 37th
// ESI tag, which CCP added after the original 36-tag sweep and which first became implementable
// with goesi-openapi v0.0.42. 4 are public (campaigns + their objectives); 2 are private
// character-scoped ("my own character" -- CharacterMilitaryCampaignObjectives/
// CharacterMilitaryCampaignObjectiveParticipation take no explicit characterID, using
// int64(c.characterID)) and need the new-style esi.activity.char:read scope. campaignID and
// objectiveID are UUID strings throughout, like Freelance Jobs' jobId.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// MilitaryCampaigns returns every military campaign. Not paginated. Public, no auth required.
func (c *Client) MilitaryCampaigns(ctx context.Context) ([]esiapi.MilitaryCampaignsDetailCampaign, error) {
	result, resp, err := c.api.MilitaryCampaignsAPI.GetMilitaryCampaignsListing(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get military campaigns: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get military campaigns: empty response")
	}
	return result.Campaigns, nil
}

// MilitaryCampaignDetail returns one military campaign. Public, no auth required.
func (c *Client) MilitaryCampaignDetail(ctx context.Context, campaignID string) (*esiapi.MilitaryCampaignsDetail, error) {
	result, resp, err := c.api.MilitaryCampaignsAPI.GetMilitaryCampaignsDetail(ctx, campaignID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get military campaign detail: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get military campaign detail: empty response")
	}
	return result, nil
}

// MilitaryCampaignObjectives returns campaignID's objectives, cursor-paginated. after="" and
// before="" each omit their own query param (mutually exclusive); limit=0 omits the limit param.
// Public, no auth required.
func (c *Client) MilitaryCampaignObjectives(ctx context.Context, campaignID, after, before string, limit int64) ([]esiapi.MilitaryCampaignsObjectivesDetailObjective, *esiapi.Cursor, error) {
	req := c.api.MilitaryCampaignsAPI.GetMilitaryCampaignsObjectivesListing(ctx, campaignID)
	result, resp, err := applyCursor(req, after, before, limit).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get military campaign objectives: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get military campaign objectives: empty response")
	}
	return result.Objectives, result.Cursor, nil
}

// MilitaryCampaignObjectiveDetail returns one objective of campaignID. Public, no auth required.
func (c *Client) MilitaryCampaignObjectiveDetail(ctx context.Context, campaignID, objectiveID string) (*esiapi.MilitaryCampaignsObjectivesDetail, error) {
	result, resp, err := c.api.MilitaryCampaignsAPI.GetMilitaryCampaignsObjectivesDetail(ctx, campaignID, objectiveID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get military campaign objective detail: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get military campaign objective detail: empty response")
	}
	return result, nil
}

// CharacterMilitaryCampaignObjectives returns the military campaign objectives the client's own
// character has participated in, cursor-paginated.
func (c *Client) CharacterMilitaryCampaignObjectives(ctx context.Context, after, before string, limit int64) ([]esiapi.CharactersMilitaryCampaignsObjectivesParticipationCharacterobjective, *esiapi.Cursor, error) {
	req := c.api.MilitaryCampaignsAPI.GetCharactersMilitaryCampaignsObjectivesListing(ctx, int64(c.characterID))
	result, resp, err := applyCursor(req, after, before, limit).Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get character military campaign objectives: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, nil, fmt.Errorf("get character military campaign objectives: empty response")
	}
	return result.Objectives, result.Cursor, nil
}

// CharacterMilitaryCampaignObjectiveParticipation returns the client's own character's
// participation in one objective. Not paginated.
func (c *Client) CharacterMilitaryCampaignObjectiveParticipation(ctx context.Context, objectiveID string) (*esiapi.CharactersMilitaryCampaignsObjectivesParticipation, error) {
	result, resp, err := c.api.MilitaryCampaignsAPI.GetCharactersMilitaryCampaignsObjectivesParticipation(ctx, int64(c.characterID), objectiveID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character military campaign objective participation: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get character military campaign objective participation: empty response")
	}
	return result, nil
}
