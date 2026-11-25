package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContracts
	t.Cleanup(func() { listCharacterContracts = orig })
	listCharacterContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Contract, int32, error) {
		return []Contract{{ContractID: 1, Status: "outstanding"}}, 3, nil
	}

	handler := listCharacterContractsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TotalPages != 3 || out.Page != 1 || len(out.Contracts) != 1 {
		t.Errorf("out = %+v, want TotalPages=3 Page=1 with one contract", out)
	}
}

func TestListCharacterContractsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContracts
	t.Cleanup(func() { listCharacterContracts = orig })
	listCharacterContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Contract, int32, error) {
		t.Fatal("listCharacterContracts should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCharacterContractsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContractBidsHandler_ReturnsBids(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContractBids
	t.Cleanup(func() { listCharacterContractBids = orig })
	listCharacterContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractBid, error) {
		return []ContractBid{{BidID: 1, Amount: 1000000}}, nil
	}

	handler := listCharacterContractBidsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].Amount != 1000000 {
		t.Errorf("out = %+v, want one bid with Amount=1000000", out)
	}
}

func TestListCharacterContractBidsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContractBids
	t.Cleanup(func() { listCharacterContractBids = orig })
	listCharacterContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractBid, error) {
		t.Fatal("listCharacterContractBids should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterContractBidsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 999, ContractID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContractItemsHandler_ReturnsItems(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContractItems
	t.Cleanup(func() { listCharacterContractItems = orig })
	listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
		return []ContractItem{{RecordID: 1, TypeID: 34}}, nil
	}

	handler := listCharacterContractItemsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 34 {
		t.Errorf("out = %+v, want one item with TypeID=34", out)
	}
}

func TestListCharacterContractItemsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContractItems
	t.Cleanup(func() { listCharacterContractItems = orig })
	listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
		t.Fatal("listCharacterContractItems should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterContractItemsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 999, ContractID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContracts
	t.Cleanup(func() { listCorporationContracts = orig })
	listCorporationContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Contract, int32, error) {
		return []Contract{{ContractID: 2}}, 2, nil
	}

	handler := listCorporationContractsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.TotalPages != 2 || len(out.Contracts) != 1 {
		t.Errorf("out = %+v, want CorporationID=98000001 TotalPages=2 with one contract", out)
	}
}

func TestListCorporationContractsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContracts
	t.Cleanup(func() { listCorporationContracts = orig })
	listCorporationContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Contract, int32, error) {
		t.Fatal("listCorporationContracts should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationContractsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractBidsHandler_ReturnsBidsAndPagination(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContractBids
	t.Cleanup(func() { listCorporationContractBids = orig })
	bidderID := int64(42)
	listCorporationContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
		return []ContractBid{{BidID: 5, BidderID: &bidderID}}, 1, nil
	}

	handler := listCorporationContractBidsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].BidID != 5 || out.TotalPages != 1 {
		t.Errorf("out = %+v, want one bid with BidID=5 and TotalPages=1", out)
	}
	if out.Bids[0].BidderID == nil || *out.Bids[0].BidderID != bidderID {
		t.Errorf("out.Bids[0].BidderID = %v, want pointer to %d", out.Bids[0].BidderID, bidderID)
	}
}

func TestListCorporationContractBidsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContractBids
	t.Cleanup(func() { listCorporationContractBids = orig })
	listCorporationContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
		t.Fatal("listCorporationContractBids should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationContractBidsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 999, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractItemsHandler_ReturnsItems(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContractItems
	t.Cleanup(func() { listCorporationContractItems = orig })
	listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
		return []ContractItem{{RecordID: 1, TypeID: 648}}, nil
	}

	handler := listCorporationContractItemsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 648 {
		t.Errorf("out = %+v, want one item with TypeID=648", out)
	}
}

func TestListCorporationContractItemsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContractItems
	t.Cleanup(func() { listCorporationContractItems = orig })
	listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
		t.Fatal("listCorporationContractItems should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationContractItemsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 999, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListPublicContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	fake := fakePublicContractsESI{publicContracts: func(_ context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error) {
		return []esiapi.ContractsPublicRegionIdGetInner{{ContractId: 10}}, 5, nil
	}}

	handler := listPublicContractsHandler(publicNarrow[publicContractsESI](fake))
	_, out, err := handler(context.Background(), nil, ListPublicContractsInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.TotalPages != 5 || len(out.Contracts) != 1 {
		t.Errorf("out = %+v, want RegionID=10000002 TotalPages=5 with one contract", out)
	}
}

func TestListPublicContractBidsHandler_ReturnsBids(t *testing.T) {
	orig := listPublicContractBids
	t.Cleanup(func() { listPublicContractBids = orig })
	listPublicContractBids = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractBid, int32, error) {
		return []ContractBid{{BidID: 7}}, 1, nil
	}

	handler := listPublicContractBidsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListPublicContractBidsInput{ContractID: 10})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].BidID != 7 {
		t.Errorf("out = %+v, want one bid with BidID=7", out)
	}
}

func TestListPublicContractItemsHandler_ReturnsItems(t *testing.T) {
	fake := fakePublicContractItemsESI{publicContractItems: func(_ context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error) {
		return []esiapi.ContractsPublicItemsContractIdGetInner{{RecordId: 1, TypeId: 34}}, 1, nil
	}}

	handler := listPublicContractItemsHandler("v1", publicNarrow[publicContractItemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListPublicContractItemsInput{ContractID: 10})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 34 {
		t.Errorf("out = %+v, want one item with TypeID=34", out)
	}
}

func TestListCharacterContractsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContracts
	t.Cleanup(func() { listCharacterContracts = orig })
	wantErr := errors.New("upstream failure")
	listCharacterContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Contract, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCharacterContractsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterContractBidsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContractBids
	t.Cleanup(func() { listCharacterContractBids = orig })
	wantErr := errors.New("upstream failure")
	listCharacterContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractBid, error) {
		return nil, wantErr
	}

	handler := listCharacterContractBidsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 7, ContractID: 1})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterContractItemsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContractItems
	t.Cleanup(func() { listCharacterContractItems = orig })
	wantErr := errors.New("upstream failure")
	listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
		return nil, wantErr
	}

	handler := listCharacterContractItemsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationContractsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContracts
	t.Cleanup(func() { listCorporationContracts = orig })
	wantErr := errors.New("upstream failure")
	listCorporationContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Contract, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationContractsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationContractBidsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContractBids
	t.Cleanup(func() { listCorporationContractBids = orig })
	wantErr := errors.New("upstream failure")
	listCorporationContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationContractBidsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationContractItemsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContractItems
	t.Cleanup(func() { listCorporationContractItems = orig })
	wantErr := errors.New("upstream failure")
	listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
		return nil, wantErr
	}

	handler := listCorporationContractItemsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListPublicContractsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakePublicContractsESI{publicContracts: func(_ context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listPublicContractsHandler(publicNarrow[publicContractsESI](fake))
	_, _, err := handler(context.Background(), nil, ListPublicContractsInput{RegionID: 10000002})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListPublicContractBidsHandler_PropagatesError(t *testing.T) {
	orig := listPublicContractBids
	t.Cleanup(func() { listPublicContractBids = orig })
	wantErr := errors.New("upstream failure")
	listPublicContractBids = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractBid, int32, error) {
		return nil, 0, wantErr
	}

	handler := listPublicContractBidsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListPublicContractBidsInput{ContractID: 10})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListPublicContractItemsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakePublicContractItemsESI{publicContractItems: func(_ context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listPublicContractItemsHandler("v1", publicNarrow[publicContractItemsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListPublicContractItemsInput{ContractID: 10})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterContractItemsHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContractItems
	t.Cleanup(func() { listCharacterContractItems = orig })
	listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
		return []ContractItem{{RecordID: 1, TypeID: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterContractItemsHandler("v1", "client", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Items[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Items[0].TypeName, "Rifter")
	}
}

func TestListCorporationContractItemsHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContractItems
	t.Cleanup(func() { listCorporationContractItems = orig })
	listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
		return []ContractItem{{RecordID: 1, TypeID: 648}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{648: "Atron"}
	}

	handler := listCorporationContractItemsHandler("v1", "client", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Items[0].TypeName != "Atron" {
		t.Errorf("TypeName = %q, want %q", out.Items[0].TypeName, "Atron")
	}
}

func TestListPublicContractItemsHandler_ResolvesTypeName(t *testing.T) {
	fake := fakePublicContractItemsESI{publicContractItems: func(_ context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error) {
		return []esiapi.ContractsPublicItemsContractIdGetInner{{RecordId: 1, TypeId: 34}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{34: "Tritanium"}
	}

	handler := listPublicContractItemsHandler("v1", publicNarrow[publicContractItemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListPublicContractItemsInput{ContractID: 10})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Items[0].TypeName != "Tritanium" {
		t.Errorf("TypeName = %q, want %q", out.Items[0].TypeName, "Tritanium")
	}
}

// fakePublicContractsESI fakes publicContractsESI; a nil publicContracts fails as an unfaked call.
type fakePublicContractsESI struct {
	publicContracts func(ctx context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error)
}

func (f fakePublicContractsESI) PublicContracts(ctx context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error) {
	if f.publicContracts == nil {
		return nil, 0, errNotFaked
	}
	return f.publicContracts(ctx, regionID, page)
}

// fakePublicContractItemsESI fakes publicContractItemsESI; a nil publicContractItems fails as an unfaked call.
type fakePublicContractItemsESI struct {
	publicContractItems func(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error)
}

func (f fakePublicContractItemsESI) PublicContractItems(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error) {
	if f.publicContractItems == nil {
		return nil, 0, errNotFaked
	}
	return f.publicContractItems(ctx, contractID, page)
}
