// Package esi's wallet.go covers all 6 of "Wallet"'s endpoints. Originally only WalletBalance
// (1 of 6) was implemented despite docs/reference/esi-coverage-roadmap.md marking the whole tag
// "Done (PR #5)" -- the remaining 5 (character wallet journal/transactions, and all three
// corporation wallet endpoints) were a real coverage gap, not an intentional scoping decision.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// WalletBalance returns the client's character's current ISK wallet balance.
func (c *Client) WalletBalance(ctx context.Context) (float64, error) {
	balance, resp, err := c.api.WalletAPI.GetCharactersCharacterIdWallet(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return 0, fmt.Errorf("get wallet balance: %w", describeError(err, resp))
	}
	return balance, nil
}

// CharacterWalletJournal returns the client's character's wallet journal entries (paginated).
// page=0 omits the page query parameter.
func (c *Client) CharacterWalletJournal(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error) {
	req := c.api.WalletAPI.GetCharactersCharacterIdWalletJournal(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character wallet journal: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CharacterWalletTransactions returns the client's character's wallet transactions, newest first.
// fromID=0 omits the from_id cursor (returns the most recent transactions); pass the smallest
// transactionId seen in a prior page to fetch older entries -- ESI paginates this endpoint by
// cursor, not by page number, unlike the journal endpoint above.
func (c *Client) CharacterWalletTransactions(ctx context.Context, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error) {
	req := c.api.WalletAPI.GetCharactersCharacterIdWalletTransactions(ctx, int64(c.characterID))
	if fromID > 0 {
		req = req.FromId(fromID)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get character wallet transactions: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationWallets returns corporationID's wallet division balances. The authenticated
// character must hold sufficient corp roles -- ESI itself returns 403 otherwise.
func (c *Client) CorporationWallets(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error) {
	result, resp, err := c.api.WalletAPI.GetCorporationsCorporationIdWallets(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation wallets: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationWalletJournal returns corporationID's division wallet journal entries (paginated).
// The authenticated character must hold sufficient corp roles -- ESI itself returns 403 otherwise.
// page=0 omits the page query parameter.
func (c *Client) CorporationWalletJournal(ctx context.Context, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error) {
	req := c.api.WalletAPI.GetCorporationsCorporationIdWalletsDivisionJournal(ctx, corporationID, division)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation wallet division journal: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationWalletTransactions returns corporationID's division wallet transactions, newest
// first. fromID=0 omits the from_id cursor. The authenticated character must hold sufficient corp
// roles -- ESI itself returns 403 otherwise.
func (c *Client) CorporationWalletTransactions(ctx context.Context, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
	req := c.api.WalletAPI.GetCorporationsCorporationIdWalletsDivisionTransactions(ctx, corporationID, division)
	if fromID > 0 {
		req = req.FromId(fromID)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation wallet division transactions: %w", describeError(err, resp))
	}
	return result, nil
}
