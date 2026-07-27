// Package esi's routes.go covers the single "Routes" endpoint -- a public, stateless route
// computation between two solar systems. POST-based (like esi/assets.go's AssetLocations/
// AssetNames) despite being read-only: ESI uses POST here because the request carries a body
// (avoid-list, extra connections, preference), not because this mutates any state.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Route computes a route between originSystemID and destinationSystemID, returning the ordered
// list of solar system IDs to traverse. avoidSystems and connections may be nil (no restriction/
// no extra connections); preference and securityPenalty may be nil to accept ESI's own defaults.
// Public -- no security block on this operation in esi/api/openapi.yaml.
func (c *Client) Route(ctx context.Context, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []esiapi.RouteConnection, preference *string, securityPenalty *int64) ([]int64, error) {
	body := esiapi.RouteRequestBody{
		AvoidSystems: avoidSystems, Connections: connections, Preference: preference, SecurityPenalty: securityPenalty,
	}
	result, resp, err := c.api.RoutesAPI.PostRoute(ctx, originSystemID, destinationSystemID).RouteRequestBody(body).Execute()
	if err != nil {
		return nil, fmt.Errorf("get route: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get route: empty response")
	}
	return result.Route, nil
}
