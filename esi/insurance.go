// Package esi's insurance.go covers the single "Insurance" endpoint -- a public, game-wide
// listing of insurance prices per ship type, no auth required, not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// InsurancePrices returns insurance price levels for every insurable ship type.
func (c *Client) InsurancePrices(ctx context.Context) ([]esiapi.InsurancePricesGetInner, error) {
	result, resp, err := c.api.InsuranceAPI.GetInsurancePrices(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get insurance prices: %w", describeError(err, resp))
	}
	return result, nil
}
