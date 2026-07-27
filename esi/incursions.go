// Package esi's incursions.go covers the single "Incursions" endpoint -- a public, game-wide
// listing of active incursions, no auth required, not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Incursions returns every currently active incursion.
func (c *Client) Incursions(ctx context.Context) ([]esiapi.IncursionsGetInner, error) {
	result, resp, err := c.api.IncursionsAPI.GetIncursions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get incursions: %w", describeError(err, resp))
	}
	return result, nil
}
