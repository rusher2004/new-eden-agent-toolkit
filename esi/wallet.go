package esi

import (
	"context"
	"fmt"
)

// WalletBalance returns the client's character's current ISK wallet balance.
func (c *Client) WalletBalance(ctx context.Context) (float64, error) {
	balance, resp, err := c.api.WalletAPI.GetCharactersCharacterIdWallet(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return 0, fmt.Errorf("get wallet balance: %w", describeError(err, resp))
	}
	return balance, nil
}
