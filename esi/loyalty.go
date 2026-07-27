// Package esi's loyalty.go covers both of "Loyalty"'s endpoints -- a character's own loyalty
// point balances (private, "my own character") and a corporation's loyalty store offers (public,
// no character/auth needed, caller-supplied corporation ID).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterLoyaltyPoints returns the client's own character's loyalty point balance with every
// corporation they've worked for.
func (c *Client) CharacterLoyaltyPoints(ctx context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error) {
	result, resp, err := c.api.LoyaltyAPI.GetCharactersCharacterIdLoyaltyPoints(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character loyalty points: %w", describeError(err, resp))
	}
	return result, nil
}

// LoyaltyStoreOffers returns a corporation's loyalty store offers. Public -- no authenticated
// character needed, so this is safe to call from a public Client (see esi.NewPublicClient).
func (c *Client) LoyaltyStoreOffers(ctx context.Context, corporationID int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error) {
	result, resp, err := c.api.LoyaltyAPI.GetLoyaltyStoresCorporationIdOffers(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get loyalty store offers: %w", describeError(err, resp))
	}
	return result, nil
}
