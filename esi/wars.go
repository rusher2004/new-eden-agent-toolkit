// Package esi's wars.go covers all 3 of "Wars"'s endpoints. All 3 are public. GetWars uses
// FC's cursor-based pagination (max_war_id, "wars with ID less than this"), NOT this project's
// usual page/X-Pages pattern -- verified individually: no .Page() setter on GetWars's request
// builder. GetWarsWarIdKillmails, unlike the other 2, IS page/X-Pages paginated -- verified
// individually.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func (c *Client) Wars(ctx context.Context, maxWarID int64) ([]int64, error) {
	req := c.api.WarsAPI.GetWars(ctx)
	if maxWarID > 0 {
		req = req.MaxWarId(maxWarID)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get wars: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) WarDetail(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error) {
	result, resp, err := c.api.WarsAPI.GetWarsWarId(ctx, warID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get war detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) WarKillmails(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	req := c.api.WarsAPI.GetWarsWarIdKillmails(ctx, warID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get war killmails: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
