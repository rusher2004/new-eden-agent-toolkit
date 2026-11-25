package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// contractsStoreWithPilot returns a config store holding character 7.
func contractsStoreWithPilot(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

// contractsStoreEmpty returns a config store with no characters.
func contractsStoreEmpty(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	return config.NewStoreAt(t.TempDir() + "/config.json")
}

var errContractsUpstream = errors.New("upstream failure")
var errContractsOpen = errors.New("keychain locked")

func TestListCharacterContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCharacterContractsESI{characterContracts: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdContractsGetInner{{ContractId: 1, Status: "outstanding"}}, 3, nil
	}}

	handler := listCharacterContractsHandler(openNarrow[characterContractsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TotalPages != 3 || out.Page != 1 || len(out.Contracts) != 1 || out.Contracts[0].ContractID != 1 || out.Contracts[0].Status != "outstanding" {
		t.Errorf("out = %+v, want TotalPages=3 Page=1 with one outstanding contract 1", out)
	}
}

func TestListCharacterContractsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	handler := listCharacterContractsHandler(noNarrow[characterContractsESI](t), contractsStoreEmpty(t))
	_, _, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContractsHandler_PropagatesError(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCharacterContractsESI{characterContracts: func(context.Context, int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error) {
		return nil, 0, errContractsUpstream
	}}

	handler := listCharacterContractsHandler(openNarrow[characterContractsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 7})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
	}
}

func TestListCharacterContractsHandler_PropagatesOpenError(t *testing.T) {
	handler := listCharacterContractsHandler(failNarrow[characterContractsESI](errContractsOpen), contractsStoreWithPilot(t))
	_, _, err := handler(context.Background(), nil, ListCharacterContractsInput{CharacterID: 7})
	if !errors.Is(err, errContractsOpen) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsOpen)
	}
}

func TestListCharacterContractBidsHandler_ReturnsBids(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCharacterContractBidsESI{characterContractBids: func(context.Context, int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error) {
		return []esiapi.CharactersCharacterIdContractsContractIdBidsGetInner{{BidId: 1, Amount: 1000000}}, nil
	}}

	handler := listCharacterContractBidsHandler(openNarrow[characterContractBidsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].Amount != 1000000 {
		t.Errorf("out = %+v, want one bid with Amount=1000000", out)
	}
}

func TestListCharacterContractBidsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	handler := listCharacterContractBidsHandler(noNarrow[characterContractBidsESI](t), contractsStoreEmpty(t))
	_, _, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 999, ContractID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContractBidsHandler_PropagatesError(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCharacterContractBidsESI{characterContractBids: func(context.Context, int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error) {
		return nil, errContractsUpstream
	}}

	handler := listCharacterContractBidsHandler(openNarrow[characterContractBidsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 7, ContractID: 1})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
	}
}

func TestListCharacterContractBidsHandler_PropagatesOpenError(t *testing.T) {
	handler := listCharacterContractBidsHandler(failNarrow[characterContractBidsESI](errContractsOpen), contractsStoreWithPilot(t))
	_, _, err := handler(context.Background(), nil, ListCharacterContractBidsInput{CharacterID: 7, ContractID: 1})
	if !errors.Is(err, errContractsOpen) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsOpen)
	}
}

func TestListCharacterContractItemsHandler_ReturnsItems(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCharacterContractItemsESI{characterContractItems: func(context.Context, int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
		return []esiapi.CharactersCharacterIdContractsContractIdItemsGetInner{{RecordId: 1, TypeId: 34}}, nil
	}}

	handler := listCharacterContractItemsHandler("v1", openNarrow[characterContractItemsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 34 {
		t.Errorf("out = %+v, want one item with TypeID=34", out)
	}
}

func TestListCharacterContractItemsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	handler := listCharacterContractItemsHandler("v1", noNarrow[characterContractItemsESI](t), contractsStoreEmpty(t), nil)
	_, _, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 999, ContractID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContractItemsHandler_PropagatesError(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCharacterContractItemsESI{characterContractItems: func(context.Context, int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
		return nil, errContractsUpstream
	}}

	handler := listCharacterContractItemsHandler("v1", openNarrow[characterContractItemsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
	}
}

func TestListCharacterContractItemsHandler_PropagatesOpenError(t *testing.T) {
	handler := listCharacterContractItemsHandler("v1", failNarrow[characterContractItemsESI](errContractsOpen), contractsStoreWithPilot(t), nil)
	_, _, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if !errors.Is(err, errContractsOpen) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsOpen)
	}
}

func TestListCharacterContractItemsHandler_ResolvesTypeName(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCharacterContractItemsESI{characterContractItems: func(context.Context, int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
		return []esiapi.CharactersCharacterIdContractsContractIdItemsGetInner{{RecordId: 1, TypeId: 587}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterContractItemsHandler("v1", openNarrow[characterContractItemsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterContractItemsInput{CharacterID: 7, ContractID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Items[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Items[0].TypeName, "Rifter")
	}
}

func TestListCorporationContractsHandler_ReturnsContractsAndPagination(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCorporationContractsESI{corporationContracts: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdContractsGetInner{{ContractId: 2}}, 2, nil
	}}

	handler := listCorporationContractsHandler(openNarrow[corporationContractsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.TotalPages != 2 || len(out.Contracts) != 1 {
		t.Errorf("out = %+v, want CorporationID=98000001 TotalPages=2 with one contract", out)
	}
}

func TestListCorporationContractsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	handler := listCorporationContractsHandler(noNarrow[corporationContractsESI](t), contractsStoreEmpty(t))
	_, _, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractsHandler_PropagatesError(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCorporationContractsESI{corporationContracts: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error) {
		return nil, 0, errContractsUpstream
	}}

	handler := listCorporationContractsHandler(openNarrow[corporationContractsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
	}
}

func TestListCorporationContractsHandler_PropagatesOpenError(t *testing.T) {
	handler := listCorporationContractsHandler(failNarrow[corporationContractsESI](errContractsOpen), contractsStoreWithPilot(t))
	_, _, err := handler(context.Background(), nil, ListCorporationContractsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, errContractsOpen) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsOpen)
	}
}

func TestListCorporationContractBidsHandler_ReturnsBidsAndPagination(t *testing.T) {
	store := contractsStoreWithPilot(t)
	bidderID := int64(42)
	fake := fakeCorporationContractBidsESI{corporationContractBids: func(context.Context, int64, int64, int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdContractsContractIdBidsGetInner{{BidId: 5, BidderId: bidderID}}, 1, nil
	}}

	handler := listCorporationContractBidsHandler(openNarrow[corporationContractBidsESI](fake), store)
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
	handler := listCorporationContractBidsHandler(noNarrow[corporationContractBidsESI](t), contractsStoreEmpty(t))
	_, _, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 999, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractBidsHandler_PropagatesError(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCorporationContractBidsESI{corporationContractBids: func(context.Context, int64, int64, int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error) {
		return nil, 0, errContractsUpstream
	}}

	handler := listCorporationContractBidsHandler(openNarrow[corporationContractBidsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
	}
}

func TestListCorporationContractBidsHandler_PropagatesOpenError(t *testing.T) {
	handler := listCorporationContractBidsHandler(failNarrow[corporationContractBidsESI](errContractsOpen), contractsStoreWithPilot(t))
	_, _, err := handler(context.Background(), nil, ListCorporationContractBidsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, errContractsOpen) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsOpen)
	}
}

func TestListCorporationContractItemsHandler_ReturnsItems(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCorporationContractItemsESI{corporationContractItems: func(context.Context, int64, int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
		return []esiapi.CharactersCharacterIdContractsContractIdItemsGetInner{{RecordId: 1, TypeId: 648}}, nil
	}}

	handler := listCorporationContractItemsHandler("v1", openNarrow[corporationContractItemsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].TypeID != 648 {
		t.Errorf("out = %+v, want one item with TypeID=648", out)
	}
}

func TestListCorporationContractItemsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	handler := listCorporationContractItemsHandler("v1", noNarrow[corporationContractItemsESI](t), contractsStoreEmpty(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 999, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContractItemsHandler_PropagatesError(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCorporationContractItemsESI{corporationContractItems: func(context.Context, int64, int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
		return nil, errContractsUpstream
	}}

	handler := listCorporationContractItemsHandler("v1", openNarrow[corporationContractItemsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
	}
}

func TestListCorporationContractItemsHandler_PropagatesOpenError(t *testing.T) {
	handler := listCorporationContractItemsHandler("v1", failNarrow[corporationContractItemsESI](errContractsOpen), contractsStoreWithPilot(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if !errors.Is(err, errContractsOpen) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsOpen)
	}
}

func TestListCorporationContractItemsHandler_ResolvesTypeName(t *testing.T) {
	store := contractsStoreWithPilot(t)
	fake := fakeCorporationContractItemsESI{corporationContractItems: func(context.Context, int64, int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
		return []esiapi.CharactersCharacterIdContractsContractIdItemsGetInner{{RecordId: 1, TypeId: 648}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{648: "Atron"}
	}

	handler := listCorporationContractItemsHandler("v1", openNarrow[corporationContractItemsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationContractItemsInput{CharacterID: 7, CorporationID: 98000001, ContractID: 3})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Items[0].TypeName != "Atron" {
		t.Errorf("TypeName = %q, want %q", out.Items[0].TypeName, "Atron")
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

func TestListPublicContractsHandler_PropagatesError(t *testing.T) {
	fake := fakePublicContractsESI{publicContracts: func(_ context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error) {
		return nil, 0, errContractsUpstream
	}}

	handler := listPublicContractsHandler(publicNarrow[publicContractsESI](fake))
	_, _, err := handler(context.Background(), nil, ListPublicContractsInput{RegionID: 10000002})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
	}
}

func TestListPublicContractBidsHandler_ReturnsBids(t *testing.T) {
	fake := fakePublicContractBidsESI{publicContractBids: func(context.Context, int64, int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error) {
		return []esiapi.ContractsPublicBidsContractIdGetInner{{BidId: 7}}, 1, nil
	}}

	handler := listPublicContractBidsHandler(publicNarrow[publicContractBidsESI](fake))
	_, out, err := handler(context.Background(), nil, ListPublicContractBidsInput{ContractID: 10})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bids) != 1 || out.Bids[0].BidID != 7 {
		t.Errorf("out = %+v, want one bid with BidID=7", out)
	}
}

func TestListPublicContractBidsHandler_PropagatesError(t *testing.T) {
	fake := fakePublicContractBidsESI{publicContractBids: func(context.Context, int64, int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error) {
		return nil, 0, errContractsUpstream
	}}

	handler := listPublicContractBidsHandler(publicNarrow[publicContractBidsESI](fake))
	_, _, err := handler(context.Background(), nil, ListPublicContractBidsInput{ContractID: 10})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
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

func TestListPublicContractItemsHandler_PropagatesError(t *testing.T) {
	fake := fakePublicContractItemsESI{publicContractItems: func(_ context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error) {
		return nil, 0, errContractsUpstream
	}}

	handler := listPublicContractItemsHandler("v1", publicNarrow[publicContractItemsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListPublicContractItemsInput{ContractID: 10})
	if !errors.Is(err, errContractsUpstream) {
		t.Errorf("err = %v, want wrapping %v", err, errContractsUpstream)
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

// fakeCharacterContractsESI fakes characterContractsESI; a nil characterContracts fails as an unfaked call.
type fakeCharacterContractsESI struct {
	characterContracts func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error)
}

func (f fakeCharacterContractsESI) CharacterContracts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error) {
	if f.characterContracts == nil {
		return nil, 0, errNotFaked
	}
	return f.characterContracts(ctx, page)
}

// fakeCharacterContractBidsESI fakes characterContractBidsESI; a nil characterContractBids fails as an unfaked call.
type fakeCharacterContractBidsESI struct {
	characterContractBids func(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error)
}

func (f fakeCharacterContractBidsESI) CharacterContractBids(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error) {
	if f.characterContractBids == nil {
		return nil, errNotFaked
	}
	return f.characterContractBids(ctx, contractID)
}

// fakeCharacterContractItemsESI fakes characterContractItemsESI; a nil characterContractItems fails as an unfaked call.
type fakeCharacterContractItemsESI struct {
	characterContractItems func(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error)
}

func (f fakeCharacterContractItemsESI) CharacterContractItems(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
	if f.characterContractItems == nil {
		return nil, errNotFaked
	}
	return f.characterContractItems(ctx, contractID)
}

// fakeCorporationContractsESI fakes corporationContractsESI; a nil corporationContracts fails as an unfaked call.
type fakeCorporationContractsESI struct {
	corporationContracts func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error)
}

func (f fakeCorporationContractsESI) CorporationContracts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error) {
	if f.corporationContracts == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationContracts(ctx, corporationID, page)
}

// fakeCorporationContractBidsESI fakes corporationContractBidsESI; a nil corporationContractBids fails as an unfaked call.
type fakeCorporationContractBidsESI struct {
	corporationContractBids func(ctx context.Context, corporationID, contractID int64, page int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error)
}

func (f fakeCorporationContractBidsESI) CorporationContractBids(ctx context.Context, corporationID int64, contractID int64, page int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error) {
	if f.corporationContractBids == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationContractBids(ctx, corporationID, contractID, page)
}

// fakeCorporationContractItemsESI fakes corporationContractItemsESI; a nil corporationContractItems fails as an unfaked call.
type fakeCorporationContractItemsESI struct {
	corporationContractItems func(ctx context.Context, corporationID, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error)
}

func (f fakeCorporationContractItemsESI) CorporationContractItems(ctx context.Context, corporationID int64, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error) {
	if f.corporationContractItems == nil {
		return nil, errNotFaked
	}
	return f.corporationContractItems(ctx, corporationID, contractID)
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

// fakePublicContractBidsESI fakes publicContractBidsESI; a nil publicContractBids fails as an unfaked call.
type fakePublicContractBidsESI struct {
	publicContractBids func(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error)
}

func (f fakePublicContractBidsESI) PublicContractBids(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error) {
	if f.publicContractBids == nil {
		return nil, 0, errNotFaked
	}
	return f.publicContractBids(ctx, contractID, page)
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
