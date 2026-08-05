package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetWalletBalanceHandler_ReturnsBalance(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		return 42.5, nil
	}

	handler := getWalletBalanceHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.BalanceISK != 42.5 {
		t.Errorf("out = %+v, want {7 42.5}", out)
	}
}

func TestGetWalletBalanceHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		return 0, errors.New("simulated failure")
	}

	handler := getWalletBalanceHandler("v1", "client-id", store)
	if _, _, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 7}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

// TestGetWalletBalanceHandler_UnknownCharacter_ReturnsErrCharacterNotFound
// guards against a confirmed consistency gap: unlike add_character/
// list_characters/remove_character, this handler used to skip checking
// config.Store entirely, so an unknown character ID surfaced as a deeply
// wrapped creds.ErrNotFound string instead of the same ErrCharacterNotFound
// sentinel the sibling tools use.
func TestGetWalletBalanceHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		t.Fatal("getWalletBalance should not be called for an unknown character")
		return 0, nil
	}

	handler := getWalletBalanceHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterWalletJournalHandler_ReturnsEntries(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterWalletJournal
	t.Cleanup(func() { listCharacterWalletJournal = orig })
	listCharacterWalletJournal = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdWalletJournalGetInner{{Id: 1, RefType: "player_donation"}}, 3, nil
	}

	handler := listCharacterWalletJournalHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterWalletJournalInput{CharacterID: 7, Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 2 || out.TotalPages != 3 || len(out.Entries) != 1 || out.Entries[0].ID != 1 {
		t.Errorf("out = %+v, want Page=2, TotalPages=3, one entry with ID=1", out)
	}
}

func TestListCharacterWalletJournalHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterWalletJournal
	t.Cleanup(func() { listCharacterWalletJournal = orig })
	listCharacterWalletJournal = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error) {
		return nil, 0, errors.New("simulated failure")
	}

	handler := listCharacterWalletJournalHandler("v1", "client-id", store)
	if _, _, err := handler(context.Background(), nil, ListCharacterWalletJournalInput{CharacterID: 7}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCharacterWalletJournalHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterWalletJournalHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterWalletJournalInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterWalletTransactionsHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	origFetch := listCharacterWalletTransactions
	t.Cleanup(func() { listCharacterWalletTransactions = origFetch })
	listCharacterWalletTransactions = func(ctx context.Context, clientID string, characterID int32, userAgent string, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error) {
		return []esiapi.CharactersCharacterIdWalletTransactionsGetInner{{TransactionId: 5, TypeId: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterWalletTransactionsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterWalletTransactionsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Transactions) != 1 || out.Transactions[0].TypeName != "Rifter" {
		t.Errorf("out.Transactions = %+v, want one transaction with TypeName=Rifter", out.Transactions)
	}
}

func TestListCharacterWalletTransactionsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterWalletTransactions
	t.Cleanup(func() { listCharacterWalletTransactions = orig })
	listCharacterWalletTransactions = func(ctx context.Context, clientID string, characterID int32, userAgent string, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error) {
		return nil, errors.New("simulated failure")
	}

	handler := listCharacterWalletTransactionsHandler("v1", "client-id", store, nil)
	if _, _, err := handler(context.Background(), nil, ListCharacterWalletTransactionsInput{CharacterID: 7}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCharacterWalletTransactionsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterWalletTransactionsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterWalletTransactionsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationWalletsHandler_ReturnsWallets(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationWallets
	t.Cleanup(func() { listCorporationWallets = orig })
	listCorporationWallets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error) {
		return []esiapi.CorporationsCorporationIdWalletsGetInner{{Division: 1, Balance: 3330000000.5}}, nil
	}

	handler := listCorporationWalletsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationWalletsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Wallets) != 1 || out.Wallets[0].Division != 1 || out.Wallets[0].BalanceISK != 3330000000.5 {
		t.Errorf("out = %+v, want CorporationID=98000001, one wallet Division=1 BalanceISK=3330000000.5", out)
	}
}

func TestListCorporationWalletsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationWallets
	t.Cleanup(func() { listCorporationWallets = orig })
	listCorporationWallets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error) {
		return nil, errors.New("simulated failure")
	}

	handler := listCorporationWalletsHandler("v1", "client-id", store)
	if _, _, err := handler(context.Background(), nil, ListCorporationWalletsInput{CharacterID: 7, CorporationID: 98000001}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCorporationWalletsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationWalletsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationWalletsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationWalletJournalHandler_ReturnsEntries(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationWalletJournal
	t.Cleanup(func() { listCorporationWalletJournal = orig })
	listCorporationWalletJournal = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner{{Id: 1, RefType: "corporation_account_withdrawal"}}, 1, nil
	}

	handler := listCorporationWalletJournalHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationWalletJournalInput{CharacterID: 7, CorporationID: 98000001, Division: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Division != 1 || out.TotalPages != 1 || len(out.Entries) != 1 || out.Entries[0].ID != 1 {
		t.Errorf("out = %+v, want Division=1, TotalPages=1, one entry with ID=1", out)
	}
}

func TestListCorporationWalletJournalHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationWalletJournal
	t.Cleanup(func() { listCorporationWalletJournal = orig })
	listCorporationWalletJournal = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error) {
		return nil, 0, errors.New("simulated failure")
	}

	handler := listCorporationWalletJournalHandler("v1", "client-id", store)
	if _, _, err := handler(context.Background(), nil, ListCorporationWalletJournalInput{CharacterID: 7, CorporationID: 98000001, Division: 1}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCorporationWalletJournalHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationWalletJournalHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationWalletJournalInput{CharacterID: 999, CorporationID: 98000001, Division: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationWalletTransactionsHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	origFetch := listCorporationWalletTransactions
	t.Cleanup(func() { listCorporationWalletTransactions = origFetch })
	listCorporationWalletTransactions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
		return []esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner{{TransactionId: 5, TypeId: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationWalletTransactionsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationWalletTransactionsInput{CharacterID: 7, CorporationID: 98000001, Division: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Transactions) != 1 || out.Transactions[0].TypeName != "Rifter" {
		t.Errorf("out.Transactions = %+v, want one transaction with TypeName=Rifter", out.Transactions)
	}
}

func TestListCorporationWalletTransactionsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationWalletTransactions
	t.Cleanup(func() { listCorporationWalletTransactions = orig })
	listCorporationWalletTransactions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
		return nil, errors.New("simulated failure")
	}

	handler := listCorporationWalletTransactionsHandler("v1", "client-id", store, nil)
	if _, _, err := handler(context.Background(), nil, ListCorporationWalletTransactionsInput{CharacterID: 7, CorporationID: 98000001, Division: 1}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCorporationWalletTransactionsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationWalletTransactionsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationWalletTransactionsInput{CharacterID: 999, CorporationID: 98000001, Division: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
