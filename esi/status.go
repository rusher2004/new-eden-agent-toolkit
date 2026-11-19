// Package esi's status.go covers the single "Status" endpoint -- server player count, version,
// start time, and VIP-mode flag. Public (no security block in esi/api/openapi.yaml), not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Status returns the current server status: online player count, running version, start time,
// and whether VIP mode is active.
func (c *Client) Status(ctx context.Context) (*esiapi.Status, error) {
	result, resp, err := c.api.StatusAPI.GetStatus(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get status: %w", describeError(err, resp))
	}
	return result, nil
}
