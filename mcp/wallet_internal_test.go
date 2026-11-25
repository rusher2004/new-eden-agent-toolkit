package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// openWalletFake and openWalletPubFake adapt a fake to the openers the handlers take.
func openWalletFake[C any](c C) func(context.Context, int32) (C, error) {
	return func(context.Context, int32) (C, error) { return c, nil }
}

func openWalletPubFake[C any](c C) func() C {
	return func() C { return c }
}

// fakeWalletBalanceESI fakes the ESI call behind getWalletBalanceHandler.
type fakeWalletBalanceESI func(ctx context.Context) (float64, error)

func (f fakeWalletBalanceESI) WalletBalance(ctx context.Context) (float64, error) {
	return f(ctx)
}

// fakeCharacterWalletJournalESI fakes the ESI call behind listCharacterWalletJournalHandler.
type fakeCharacterWalletJournalESI func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error)

func (f fakeCharacterWalletJournalESI) CharacterWalletJournal(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error) {
	return f(ctx, page)
}

// fakeCharacterWalletTransactionsESI fakes the ESI call behind listCharacterWalletTransactionsHandler.
type fakeCharacterWalletTransactionsESI func(ctx context.Context, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error)

func (f fakeCharacterWalletTransactionsESI) CharacterWalletTransactions(ctx context.Context, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error) {
	return f(ctx, fromID)
}

// fakeCorporationWalletsESI fakes the ESI call behind listCorporationWalletsHandler.
type fakeCorporationWalletsESI func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error)

func (f fakeCorporationWalletsESI) CorporationWallets(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error) {
	return f(ctx, corporationID)
}

// fakeCorporationWalletJournalESI fakes the ESI call behind listCorporationWalletJournalHandler.
type fakeCorporationWalletJournalESI func(ctx context.Context, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error)

func (f fakeCorporationWalletJournalESI) CorporationWalletJournal(ctx context.Context, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error) {
	return f(ctx, corporationID, division, page)
}

// fakeCorporationWalletTransactionsESI fakes the ESI call behind listCorporationWalletTransactionsHandler.
type fakeCorporationWalletTransactionsESI func(ctx context.Context, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error)

func (f fakeCorporationWalletTransactionsESI) CorporationWalletTransactions(ctx context.Context, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
	return f(ctx, corporationID, division, fromID)
}

func TestGetWalletBalanceHandler_ReturnsBalance(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	getWalletBalanceFake := fakeWalletBalanceESI(func(ctx context.Context) (float64, error) {
		return 42.5, nil
	})

	handler := getWalletBalanceHandler(openWalletFake[walletBalanceESI](getWalletBalanceFake), store)
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

	getWalletBalanceFake := fakeWalletBalanceESI(func(ctx context.Context) (float64, error) {
		return 0, errors.New("simulated failure")
	})

	handler := getWalletBalanceHandler(openWalletFake[walletBalanceESI](getWalletBalanceFake), store)
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

	getWalletBalanceFake := fakeWalletBalanceESI(func(ctx context.Context) (float64, error) {
		t.Fatal("getWalletBalance should not be called for an unknown character")
		return 0, nil
	})

	handler := getWalletBalanceHandler(openWalletFake[walletBalanceESI](getWalletBalanceFake), store)
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

	listCharacterWalletJournalFake := fakeCharacterWalletJournalESI(func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdWalletJournalGetInner{{Id: 1, RefType: "player_donation"}}, 3, nil
	})

	handler := listCharacterWalletJournalHandler(openWalletFake[characterWalletJournalESI](listCharacterWalletJournalFake), store)
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

	listCharacterWalletJournalFake := fakeCharacterWalletJournalESI(func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdWalletJournalGetInner, int32, error) {
		return nil, 0, errors.New("simulated failure")
	})

	handler := listCharacterWalletJournalHandler(openWalletFake[characterWalletJournalESI](listCharacterWalletJournalFake), store)
	if _, _, err := handler(context.Background(), nil, ListCharacterWalletJournalInput{CharacterID: 7}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCharacterWalletJournalHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterWalletJournalHandler[characterWalletJournalESI](nil, store)
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

	listCharacterWalletTransactionsFake := fakeCharacterWalletTransactionsESI(func(ctx context.Context, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error) {
		return []esiapi.CharactersCharacterIdWalletTransactionsGetInner{{TransactionId: 5, TypeId: 587}}, nil
	})

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterWalletTransactionsHandler("v1", openWalletFake[characterWalletTransactionsESI](listCharacterWalletTransactionsFake), store, nil)
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

	listCharacterWalletTransactionsFake := fakeCharacterWalletTransactionsESI(func(ctx context.Context, fromID int64) ([]esiapi.CharactersCharacterIdWalletTransactionsGetInner, error) {
		return nil, errors.New("simulated failure")
	})

	handler := listCharacterWalletTransactionsHandler("v1", openWalletFake[characterWalletTransactionsESI](listCharacterWalletTransactionsFake), store, nil)
	if _, _, err := handler(context.Background(), nil, ListCharacterWalletTransactionsInput{CharacterID: 7}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCharacterWalletTransactionsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterWalletTransactionsHandler[characterWalletTransactionsESI]("v1", nil, store, nil)
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

	listCorporationWalletsFake := fakeCorporationWalletsESI(func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error) {
		return []esiapi.CorporationsCorporationIdWalletsGetInner{{Division: 1, Balance: 3330000000.5}}, nil
	})

	handler := listCorporationWalletsHandler(openWalletFake[corporationWalletsESI](listCorporationWalletsFake), store)
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

	listCorporationWalletsFake := fakeCorporationWalletsESI(func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdWalletsGetInner, error) {
		return nil, errors.New("simulated failure")
	})

	handler := listCorporationWalletsHandler(openWalletFake[corporationWalletsESI](listCorporationWalletsFake), store)
	if _, _, err := handler(context.Background(), nil, ListCorporationWalletsInput{CharacterID: 7, CorporationID: 98000001}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCorporationWalletsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationWalletsHandler[corporationWalletsESI](nil, store)
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

	listCorporationWalletJournalFake := fakeCorporationWalletJournalESI(func(ctx context.Context, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner{{Id: 1, RefType: "corporation_account_withdrawal"}}, 1, nil
	})

	handler := listCorporationWalletJournalHandler(openWalletFake[corporationWalletJournalESI](listCorporationWalletJournalFake), store)
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

	listCorporationWalletJournalFake := fakeCorporationWalletJournalESI(func(ctx context.Context, corporationID, division int64, page int32) ([]esiapi.CorporationsCorporationIdWalletsDivisionJournalGetInner, int32, error) {
		return nil, 0, errors.New("simulated failure")
	})

	handler := listCorporationWalletJournalHandler(openWalletFake[corporationWalletJournalESI](listCorporationWalletJournalFake), store)
	if _, _, err := handler(context.Background(), nil, ListCorporationWalletJournalInput{CharacterID: 7, CorporationID: 98000001, Division: 1}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCorporationWalletJournalHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationWalletJournalHandler[corporationWalletJournalESI](nil, store)
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

	listCorporationWalletTransactionsFake := fakeCorporationWalletTransactionsESI(func(ctx context.Context, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
		return []esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner{{TransactionId: 5, TypeId: 587}}, nil
	})

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationWalletTransactionsHandler("v1", openWalletFake[corporationWalletTransactionsESI](listCorporationWalletTransactionsFake), store, nil)
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

	listCorporationWalletTransactionsFake := fakeCorporationWalletTransactionsESI(func(ctx context.Context, corporationID, division, fromID int64) ([]esiapi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
		return nil, errors.New("simulated failure")
	})

	handler := listCorporationWalletTransactionsHandler("v1", openWalletFake[corporationWalletTransactionsESI](listCorporationWalletTransactionsFake), store, nil)
	if _, _, err := handler(context.Background(), nil, ListCorporationWalletTransactionsInput{CharacterID: 7, CorporationID: 98000001, Division: 1}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestListCorporationWalletTransactionsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationWalletTransactionsHandler[corporationWalletTransactionsESI]("v1", nil, store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationWalletTransactionsInput{CharacterID: 999, CorporationID: 98000001, Division: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// TestWalletHandlers_PropagateOpenError checks the authenticated handlers report a failed ESI client build,
// wrapped so errors.Is still finds the cause.
func TestWalletHandlers_PropagateOpenError(t *testing.T) {
	store := storeWithCharacter(t)
	ctx := context.Background()
	errOpen := errors.New("open failed")

	for name, call := range map[string]func() error{
		"get_wallet_balance": func() error {
			open := func(context.Context, int32) (walletBalanceESI, error) { return nil, errOpen }
			_, _, err := getWalletBalanceHandler(open, store)(ctx, nil, GetWalletBalanceInput{CharacterID: 7})
			return err
		},
		"list_character_wallet_journal": func() error {
			open := func(context.Context, int32) (characterWalletJournalESI, error) { return nil, errOpen }
			_, _, err := listCharacterWalletJournalHandler(open, store)(ctx, nil, ListCharacterWalletJournalInput{CharacterID: 7})
			return err
		},
		"list_character_wallet_transactions": func() error {
			open := func(context.Context, int32) (characterWalletTransactionsESI, error) { return nil, errOpen }
			_, _, err := listCharacterWalletTransactionsHandler("v1", open, store, nil)(ctx, nil, ListCharacterWalletTransactionsInput{CharacterID: 7})
			return err
		},
		"list_corporation_wallets": func() error {
			open := func(context.Context, int32) (corporationWalletsESI, error) { return nil, errOpen }
			_, _, err := listCorporationWalletsHandler(open, store)(ctx, nil, ListCorporationWalletsInput{CharacterID: 7, CorporationID: 1})
			return err
		},
		"list_corporation_wallet_journal": func() error {
			open := func(context.Context, int32) (corporationWalletJournalESI, error) { return nil, errOpen }
			_, _, err := listCorporationWalletJournalHandler(open, store)(ctx, nil, ListCorporationWalletJournalInput{CharacterID: 7, CorporationID: 1, Division: 1})
			return err
		},
		"list_corporation_wallet_transactions": func() error {
			open := func(context.Context, int32) (corporationWalletTransactionsESI, error) { return nil, errOpen }
			_, _, err := listCorporationWalletTransactionsHandler("v1", open, store, nil)(ctx, nil, ListCorporationWalletTransactionsInput{CharacterID: 7, CorporationID: 1, Division: 1})
			return err
		},
	} {
		if err := call(); !errors.Is(err, errOpen) {
			t.Errorf("%s: err = %v, want wrapping errOpen", name, err)
		}
	}
}
