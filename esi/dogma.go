// Package esi's dogma.go covers all 5 of "Dogma"'s endpoints -- Dogma is EVE's live
// attribute/effect reference data (item stats, module effects, mutated/"abyssal" item detail),
// all public, no character auth required.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// DogmaAttributes returns every dogma attribute ID currently defined.
func (c *Client) DogmaAttributes(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.DogmaAPI.GetDogmaAttributes(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get dogma attributes: %w", describeError(err, resp))
	}
	return result, nil
}

// DogmaAttributeDetail returns detail for one dogma attribute.
func (c *Client) DogmaAttributeDetail(ctx context.Context, attributeID int64) (*esiapi.DogmaAttributesAttributeIdGet, error) {
	result, resp, err := c.api.DogmaAPI.GetDogmaAttributesAttributeId(ctx, attributeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get dogma attribute detail: %w", describeError(err, resp))
	}
	return result, nil
}

// DogmaDynamicItemDetail returns detail for one mutated ("abyssal") item, identified by its
// source type and item ID. Note: the generated function's own argument order is
// (itemId, typeId) -- reversed from its URL path -- this facade normalizes to (typeID, itemID)
// to match the URL and this method's own signature, swapping internally.
func (c *Client) DogmaDynamicItemDetail(ctx context.Context, typeID int64, itemID int64) (*esiapi.DogmaDynamicItemsTypeIdItemIdGet, error) {
	result, resp, err := c.api.DogmaAPI.GetDogmaDynamicItemsTypeIdItemId(ctx, itemID, typeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get dogma dynamic item detail: %w", describeError(err, resp))
	}
	return result, nil
}

// DogmaEffects returns every dogma effect ID currently defined.
func (c *Client) DogmaEffects(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.DogmaAPI.GetDogmaEffects(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get dogma effects: %w", describeError(err, resp))
	}
	return result, nil
}

// DogmaEffectDetail returns detail for one dogma effect.
func (c *Client) DogmaEffectDetail(ctx context.Context, effectID int64) (*esiapi.DogmaEffectsEffectIdGet, error) {
	result, resp, err := c.api.DogmaAPI.GetDogmaEffectsEffectId(ctx, effectID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get dogma effect detail: %w", describeError(err, resp))
	}
	return result, nil
}
