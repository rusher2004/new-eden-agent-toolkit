// Package esi's meta.go covers all 4 of "Meta"'s endpoints -- ESI's own changelog, compatibility
// dates, name history, and route health status. All 4 are public (no security block in
// esi/api/openapi.yaml) and not paginated (no .Page() setter on any of the 4 request builders).
// This is meta-information about the ESI service itself, not EVE game data.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// MetaChangelog returns ESI's own changelog, keyed by date.
func (c *Client) MetaChangelog(ctx context.Context) (*esiapi.MetaChangelog, error) {
	result, resp, err := c.api.MetaAPI.GetMetaChangelog(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta changelog: %w", describeError(err, resp))
	}
	return result, nil
}

// MetaCompatibilityDates returns every compatibility date ESI currently accepts.
func (c *Client) MetaCompatibilityDates(ctx context.Context) (*esiapi.MetaCompatibilityDates, error) {
	result, resp, err := c.api.MetaAPI.GetMetaCompatibilityDates(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta compatibility dates: %w", describeError(err, resp))
	}
	return result, nil
}

// MetaName returns ESI's current and historical names.
func (c *Client) MetaName(ctx context.Context) (*esiapi.MetaName, error) {
	result, resp, err := c.api.MetaAPI.GetMetaName(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta name: %w", describeError(err, resp))
	}
	return result, nil
}

// MetaStatus returns the health status of every ESI route.
func (c *Client) MetaStatus(ctx context.Context) (*esiapi.MetaStatus, error) {
	result, resp, err := c.api.MetaAPI.GetMetaStatus(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta status: %w", describeError(err, resp))
	}
	return result, nil
}
