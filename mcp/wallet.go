// Package mcp's wallet.go covers all 6 of "Wallet"'s endpoints. Only get_wallet_balance (1 of 6)
// originally shipped -- the remaining 5 below (character wallet journal/transactions, and all
// three corporation wallet endpoints) close that gap; see esi/wallet.go's package doc for how it
// was found.
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// getWalletBalance is overridable in tests so no real network/auth runs.
var getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.WalletBalance(ctx)
}

// GetWalletBalanceInput is get_wallet_balance's tool input.
type GetWalletBalanceInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// GetWalletBalanceOutput is get_wallet_balance's tool result.
type GetWalletBalanceOutput struct {
	CharacterID int32   `json:"characterId"`
	BalanceISK  float64 `json:"balanceIsk"`
}

func getWalletBalanceHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[GetWalletBalanceInput, GetWalletBalanceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetWalletBalanceInput) (*mcpsdk.CallToolResult, GetWalletBalanceOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("get wallet balance for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		balance, err := getWalletBalance(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("get wallet balance: %w", err)
		}
		return nil, GetWalletBalanceOutput{CharacterID: in.CharacterID, BalanceISK: balance}, nil
	}
}

// WalletJournalEntry is shared by list_character_wallet_journal and
// list_corporation_wallet_journal -- the two underlying ESI models share every field exactly.
// FirstPartyID/SecondPartyID have no consistent entity type across ref_types (character,
// corporation, faction, or system depending on RefType), so they're left as raw IDs -- resolving
// them would require per-ref_type logic this tool doesn't attempt.
type WalletJournalEntry struct {
	Amount        *float64  `json:"amount,omitempty"`
	Balance       *float64  `json:"balance,omitempty"`
	ContextID     *int64    `json:"contextId,omitempty"`
	ContextIDType *string   `json:"contextIdType,omitempty"`
	Date          time.Time `json:"date"`
	Description   string    `json:"description"`
	FirstPartyID  *int64    `json:"firstPartyId,omitempty"`
	ID            int64     `json:"id"`
	Reason        *string   `json:"reason,omitempty"`
	RefType       string    `json:"refType"`
	SecondPartyID *int64    `json:"secondPartyId,omitempty"`
	Tax           *float64  `json:"tax,omitempty"`
	TaxReceiverID *int64    `json:"taxReceiverId,omitempty"`
}

func toWalletJournalEntryFromCharacter(e esiapi.CharactersCharacterIdWalletJournalGetInner) WalletJournalEntry {
	return WalletJournalEntry{
		Amount: e.Amount, Balance: e.Balance, ContextID: e.ContextId, ContextIDType: e.ContextIdType,
		Date: e.Date, Description: e.Description, FirstPartyID: e.FirstPartyId, ID: e.Id,
		Reason: e.Reason, RefType: e.RefType, SecondPartyID: e.SecondPartyId, Tax: e.Tax, TaxReceiverID: e.TaxReceiverId,
	}
}

func toWalletJournalEntryFromCorporation(e esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner) WalletJournalEntry {
	return WalletJournalEntry{
		Amount: e.Amount, Balance: e.Balance, ContextID: e.ContextId, ContextIDType: e.ContextIdType,
		Date: e.Date, Description: e.Description, FirstPartyID: e.FirstPartyId, ID: e.Id,
		Reason: e.Reason, RefType: e.RefType, SecondPartyID: e.SecondPartyId, Tax: e.Tax, TaxReceiverID: e.TaxReceiverId,
	}
}

// WalletTransaction is shared by list_character_wallet_transactions and
// list_corporation_wallet_transactions -- identical fields except IsPersonal, which the
// corporation model doesn't carry (verified from source, not assumed).
type WalletTransaction struct {
	ClientID      int64     `json:"clientId"`
	Date          time.Time `json:"date"`
	IsBuy         bool      `json:"isBuy"`
	IsPersonal    *bool     `json:"isPersonal,omitempty"`
	JournalRefID  int64     `json:"journalRefId"`
	LocationID    int64     `json:"locationId"`
	Quantity      int64     `json:"quantity"`
	TransactionID int64     `json:"transactionId"`
	TypeID        int64     `json:"typeId"`
	TypeName      string    `json:"typeName,omitempty"`
	UnitPrice     float64   `json:"unitPrice"`
}

func toWalletTransactionFromCharacter(t esiapi.CharactersCharacterIdWalletTransactionsGetInner) WalletTransaction {
	return WalletTransaction{
		ClientID: t.ClientId, Date: t.Date, IsBuy: t.IsBuy, IsPersonal: &t.IsPersonal, JournalRefID: t.JournalRefId,
		LocationID: t.LocationId, Quantity: t.Quantity, TransactionID: t.TransactionId, TypeID: t.TypeId, UnitPrice: t.UnitPrice,
	}
}

func toWalletTransactionFromCorporation(t esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner) WalletTransaction {
	return WalletTransaction{
		ClientID: t.ClientId, Date: t.Date, IsBuy: t.IsBuy, JournalRefID: t.JournalRefId,
		LocationID: t.LocationId, Quantity: t.Quantity, TransactionID: t.TransactionId, TypeID: t.TypeId, UnitPrice: t.UnitPrice,
	}
}

// --- list_character_wallet_journal (private, paginated) ---

var listCharacterWalletJournal = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterWalletJournal(ctx, page)
}

type ListCharacterWalletJournalInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterWalletJournalOutput struct {
	CharacterID int32                `json:"characterId"`
	Page        int32                `json:"page"`
	TotalPages  int32                `json:"totalPages"`
	Entries     []WalletJournalEntry `json:"entries"`
}

func listCharacterWalletJournalHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCharacterWalletJournalInput, ListCharacterWalletJournalOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterWalletJournalInput) (*mcpsdk.CallToolResult, ListCharacterWalletJournalOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterWalletJournalOutput{}, err
		}
		if !found {
			return nil, ListCharacterWalletJournalOutput{}, fmt.Errorf("list character wallet journal for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		entries, totalPages, err := listCharacterWalletJournal(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListCharacterWalletJournalOutput{}, fmt.Errorf("list character wallet journal: %w", err)
		}
		out := make([]WalletJournalEntry, len(entries))
		for i, e := range entries {
			out[i] = toWalletJournalEntryFromCharacter(e)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCharacterWalletJournalOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Entries: out}, nil
	}
}

// --- list_character_wallet_transactions (private, cursor-paginated) ---

var listCharacterWalletTransactions = func(ctx context.Context, clientID string, characterID int32, userAgent string, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterWalletTransactions(ctx, fromID)
}

type ListCharacterWalletTransactionsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	FromID      int64 `json:"fromId,omitempty" jsonschema:"only return transactions with a transactionId strictly less than this value; omit to start from the most recent transaction"`
}
type ListCharacterWalletTransactionsOutput struct {
	CharacterID  int32               `json:"characterId"`
	Transactions []WalletTransaction `json:"transactions"`
}

func listCharacterWalletTransactionsHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListCharacterWalletTransactionsInput, ListCharacterWalletTransactionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterWalletTransactionsInput) (*mcpsdk.CallToolResult, ListCharacterWalletTransactionsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterWalletTransactionsOutput{}, err
		}
		if !found {
			return nil, ListCharacterWalletTransactionsOutput{}, fmt.Errorf("list character wallet transactions for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		txns, err := listCharacterWalletTransactions(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FromID)
		if err != nil {
			return nil, ListCharacterWalletTransactionsOutput{}, fmt.Errorf("list character wallet transactions: %w", err)
		}
		out := make([]WalletTransaction, len(txns))
		for i, t := range txns {
			out[i] = toWalletTransactionFromCharacter(t)
		}
		ids := make([]int64, len(out))
		for i, t := range out {
			ids[i] = t.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		return nil, ListCharacterWalletTransactionsOutput{CharacterID: in.CharacterID, Transactions: out}, nil
	}
}

// --- list_corporation_wallets (private, corp-scoped) ---

type CorporationWalletDivision struct {
	Division   int64   `json:"division"`
	BalanceISK float64 `json:"balanceIsk"`
}

var listCorporationWallets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationWallets(ctx, corporationID)
}

type ListCorporationWalletsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationWalletsOutput struct {
	CharacterID   int32                       `json:"characterId"`
	CorporationID int64                       `json:"corporationId"`
	Wallets       []CorporationWalletDivision `json:"wallets"`
}

func listCorporationWalletsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCorporationWalletsInput, ListCorporationWalletsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationWalletsInput) (*mcpsdk.CallToolResult, ListCorporationWalletsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationWalletsOutput{}, err
		}
		if !found {
			return nil, ListCorporationWalletsOutput{}, fmt.Errorf("list corporation wallets for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		wallets, err := listCorporationWallets(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationWalletsOutput{}, fmt.Errorf("list corporation wallets: %w", err)
		}
		out := make([]CorporationWalletDivision, len(wallets))
		for i, w := range wallets {
			out[i] = CorporationWalletDivision{Division: w.Division, BalanceISK: w.Balance}
		}
		return nil, ListCorporationWalletsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Wallets: out}, nil
	}
}

// --- list_corporation_wallet_journal (private, corp-scoped, paginated) ---

var listCorporationWalletJournal = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationWalletJournal(ctx, corporationID, division, page)
}

type ListCorporationWalletJournalInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Division      int64 `json:"division" jsonschema:"the wallet division number (1-7) to query -- see get_corporation_divisions for names"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationWalletJournalOutput struct {
	CharacterID   int32                `json:"characterId"`
	CorporationID int64                `json:"corporationId"`
	Division      int64                `json:"division"`
	Page          int32                `json:"page"`
	TotalPages    int32                `json:"totalPages"`
	Entries       []WalletJournalEntry `json:"entries"`
}

func listCorporationWalletJournalHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCorporationWalletJournalInput, ListCorporationWalletJournalOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationWalletJournalInput) (*mcpsdk.CallToolResult, ListCorporationWalletJournalOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationWalletJournalOutput{}, err
		}
		if !found {
			return nil, ListCorporationWalletJournalOutput{}, fmt.Errorf("list corporation wallet journal for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		entries, totalPages, err := listCorporationWalletJournal(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Division, in.Page)
		if err != nil {
			return nil, ListCorporationWalletJournalOutput{}, fmt.Errorf("list corporation wallet journal: %w", err)
		}
		out := make([]WalletJournalEntry, len(entries))
		for i, e := range entries {
			out[i] = toWalletJournalEntryFromCorporation(e)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationWalletJournalOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, Division: in.Division,
			Page: page, TotalPages: totalPages, Entries: out,
		}, nil
	}
}

// --- list_corporation_wallet_transactions (private, corp-scoped, cursor-paginated) ---

var listCorporationWalletTransactions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationWalletTransactions(ctx, corporationID, division, fromID)
}

type ListCorporationWalletTransactionsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Division      int64 `json:"division" jsonschema:"the wallet division number (1-7) to query -- see get_corporation_divisions for names"`
	FromID        int64 `json:"fromId,omitempty" jsonschema:"only return transactions with a transactionId strictly less than this value; omit to start from the most recent transaction"`
}
type ListCorporationWalletTransactionsOutput struct {
	CharacterID   int32               `json:"characterId"`
	CorporationID int64               `json:"corporationId"`
	Division      int64               `json:"division"`
	Transactions  []WalletTransaction `json:"transactions"`
}

func listCorporationWalletTransactionsHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListCorporationWalletTransactionsInput, ListCorporationWalletTransactionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationWalletTransactionsInput) (*mcpsdk.CallToolResult, ListCorporationWalletTransactionsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationWalletTransactionsOutput{}, err
		}
		if !found {
			return nil, ListCorporationWalletTransactionsOutput{}, fmt.Errorf("list corporation wallet transactions for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		txns, err := listCorporationWalletTransactions(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Division, in.FromID)
		if err != nil {
			return nil, ListCorporationWalletTransactionsOutput{}, fmt.Errorf("list corporation wallet transactions: %w", err)
		}
		out := make([]WalletTransaction, len(txns))
		for i, t := range txns {
			out[i] = toWalletTransactionFromCorporation(t)
		}
		ids := make([]int64, len(out))
		for i, t := range out {
			ids[i] = t.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		return nil, ListCorporationWalletTransactionsOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, Division: in.Division, Transactions: out,
		}, nil
	}
}
