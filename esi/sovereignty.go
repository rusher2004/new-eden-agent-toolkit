// Package esi's sovereignty.go covers both of "Sovereignty"'s endpoints -- active sovereignty
// campaigns and the current sovereignty claim (if any) for every solar system. Both are public
// (no security block in esi/api/openapi.yaml) and not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// SovereigntyCampaigns returns every active sovereignty campaign in the game.
func (c *Client) SovereigntyCampaigns(ctx context.Context) ([]esiapi.SovereigntyCampaignsGetInner, error) {
	result, resp, err := c.api.SovereigntyAPI.GetSovereigntyCampaigns(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get sovereignty campaigns: %w", describeError(err, resp))
	}
	return result, nil
}

// SovereigntySystems returns every solar system and its current sovereignty claim (if any).
func (c *Client) SovereigntySystems(ctx context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error) {
	result, resp, err := c.api.SovereigntyAPI.GetSovereigntySystems(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get sovereignty systems: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get sovereignty systems: empty response")
	}
	return result.SolarSystems, nil
}
