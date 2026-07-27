package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Alliances returns every alliance ID currently in the game.
func (c *Client) Alliances(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliances(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliances: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceDetail returns one alliance's name, ticker, founding info, and optional
// executor-corporation/faction-warfare affiliation.
func (c *Client) AllianceDetail(ctx context.Context, allianceID int64) (*esiapi.AllianceDetail, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceId(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance detail: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceCorporations returns the corporation IDs currently in one alliance.
func (c *Client) AllianceCorporations(ctx context.Context, allianceID int64) ([]int64, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceIdCorporations(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance corporations: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceIcons returns one alliance's icon URLs, if it has any.
func (c *Client) AllianceIcons(ctx context.Context, allianceID int64) (*esiapi.AlliancesAllianceIdIconsGet, error) {
	result, resp, err := c.api.AllianceAPI.GetAlliancesAllianceIdIcons(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance icons: %w", describeError(err, resp))
	}
	return result, nil
}
