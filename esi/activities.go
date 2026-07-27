package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// MercenaryTacticalOperations returns the client's character's mercenary tactical operations
// (bare IDs only — call MercenaryTacticalOperationDetail for full detail).
func (c *Client) MercenaryTacticalOperations(ctx context.Context) ([]esiapi.CharactersMercenaryTacticalOperationsListingOperation, error) {
	result, resp, err := c.api.ActivitiesAPI.GetCharactersMercenaryTacticalOperationsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get mercenary tactical operations: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get mercenary tactical operations: empty response")
	}
	return result.Operations, nil
}

// MercenaryTacticalOperationDetail returns detail for one mercenary tactical operation.
// operationID is a string in ESI's own model, not neat's usual int32/int64 ID convention.
func (c *Client) MercenaryTacticalOperationDetail(ctx context.Context, operationID string) (*esiapi.CharactersMercenaryTacticalOperationsDetail, error) {
	result, resp, err := c.api.ActivitiesAPI.GetCharactersMercenaryTacticalOperationsDetail(ctx, operationID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get mercenary tactical operation detail: %w", describeError(err, resp))
	}
	return result, nil
}

// RaidableSkyhooks returns upcoming raidable skyhooks. Unauthenticated on ESI's side (no scope
// required, no character ID parameter) -- call it on a Client built via NewPublicClient, not
// NewClient, since no character token is needed.
func (c *Client) RaidableSkyhooks(ctx context.Context) ([]esiapi.SkyhooksRaidableVulnerableskyhook, error) {
	result, resp, err := c.api.ActivitiesAPI.GetSkyhooksRaidable(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get raidable skyhooks: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get raidable skyhooks: empty response")
	}
	return result.Skyhooks, nil
}
