package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListAssetsHandler_ReturnsAssets(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAssetsESI{assets: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdAssetsGetInner{{ItemId: 1, TypeId: 587, LocationId: 60003760, LocationType: "station", LocationFlag: "Hangar", Quantity: 1, IsSingleton: true}}, 3, nil
	}}

	handler := listAssetsHandler("v1", openNarrow[assetsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 7, Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Assets) != 1 || out.Assets[0].ItemID != 1 || out.Page != 2 || out.TotalPages != 3 {
		t.Errorf("out = %+v, want CharacterID=7 Page=2 TotalPages=3 with one asset ItemID=1", out)
	}
}

func TestListAssetsHandler_ResolvesTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAssetsESI{assets: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdAssetsGetInner{{ItemId: 1, TypeId: 587}, {ItemId: 2, TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{587: "Rifter"}
	}

	handler := listAssetsHandler("v1", openNarrow[assetsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 2 || gotIDs[0] != 587 || gotIDs[1] != 587 {
		t.Errorf("resolveTypeNames got ids = %v, want [587 587]", gotIDs)
	}
	for _, a := range out.Assets {
		if a.TypeName != "Rifter" {
			t.Errorf("Assets[itemId=%d].TypeName = %q, want %q", a.ItemID, a.TypeName, "Rifter")
		}
	}
}

func TestListAssetsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeAssetsESI{assets: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listAssetsHandler("v1", openNarrow[assetsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListAssetsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listAssetsHandler("v1", noNarrow[assetsESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetAssetLocationsHandler_ReturnsLocations(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getAssetLocations
	t.Cleanup(func() { getAssetLocations = orig })
	getAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetLocation, error) {
		if len(itemIDs) != 2 || itemIDs[0] != 1 || itemIDs[1] != 2 {
			t.Errorf("itemIDs = %v, want [1 2]", itemIDs)
		}
		return []AssetLocation{{ItemID: 1, X: 1.5, Y: 2.5, Z: 3.5}}, nil
	}

	handler := getAssetLocationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetAssetLocationsInput{CharacterID: 7, ItemIDs: []int64{1, 2}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Locations) != 1 || out.Locations[0].X != 1.5 {
		t.Errorf("out = %+v, want one location with X=1.5", out)
	}
}

func TestGetAssetLocationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getAssetLocations
	t.Cleanup(func() { getAssetLocations = orig })
	getAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetLocation, error) {
		t.Fatal("getAssetLocations should not be called for an unknown character")
		return nil, nil
	}

	handler := getAssetLocationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetAssetLocationsInput{CharacterID: 999, ItemIDs: []int64{1}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetAssetNamesHandler_ReturnsNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAssetNamesESI{assetNames: func(_ context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
		return []esiapi.CharactersCharacterIdAssetsNamesPostInner{{ItemId: 1, Name: "My Renamed Ship"}}, nil
	}}

	handler := getAssetNamesHandler(openNarrow[assetNamesESI](fake), store)
	_, out, err := handler(context.Background(), nil, GetAssetNamesInput{CharacterID: 7, ItemIDs: []int64{1}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Names) != 1 || out.Names[0].Name != "My Renamed Ship" {
		t.Errorf("out = %+v, want one name \"My Renamed Ship\"", out)
	}
}

func TestGetAssetNamesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := getAssetNamesHandler(noNarrow[assetNamesESI](t), store)
	_, _, err := handler(context.Background(), nil, GetAssetNamesInput{CharacterID: 999, ItemIDs: []int64{1}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationAssetsHandler_ReturnsAssets(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationAssetsESI{corporationAssets: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error) {
		if corporationID != 98000001 {
			t.Errorf("corporationID = %d, want 98000001", corporationID)
		}
		return []esiapi.CorporationsCorporationIdAssetsGetInner{{ItemId: 2, TypeId: 34, Quantity: 10}}, 5, nil
	}}

	handler := listCorporationAssetsHandler("v1", openNarrow[corporationAssetsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 7, CorporationID: 98000001, Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Assets) != 1 || out.Page != 2 || out.TotalPages != 5 {
		t.Errorf("out = %+v, want CorporationID=98000001 Page=2 TotalPages=5 with one asset", out)
	}
}

func TestListCorporationAssetsHandler_ResolvesTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationAssetsESI{corporationAssets: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdAssetsGetInner{{ItemId: 2, TypeId: 34}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{34: "Tritanium"}
	}

	handler := listCorporationAssetsHandler("v1", openNarrow[corporationAssetsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 34 {
		t.Errorf("resolveTypeNames got ids = %v, want [34]", gotIDs)
	}
	if out.Assets[0].TypeName != "Tritanium" {
		t.Errorf("Assets[0].TypeName = %q, want %q", out.Assets[0].TypeName, "Tritanium")
	}
}

func TestListCorporationAssetsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationAssetsESI{corporationAssets: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationAssetsHandler("v1", openNarrow[corporationAssetsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationAssetsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationAssetsHandler("v1", noNarrow[corporationAssetsESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationAssetLocationsHandler_ReturnsLocations(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationAssetLocations
	t.Cleanup(func() { getCorporationAssetLocations = orig })
	getCorporationAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetLocation, error) {
		return []AssetLocation{{ItemID: 2, X: 4, Y: 5, Z: 6}}, nil
	}

	handler := getCorporationAssetLocationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCorporationAssetLocationsInput{CharacterID: 7, CorporationID: 98000001, ItemIDs: []int64{2}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Locations) != 1 || out.Locations[0].ItemID != 2 {
		t.Errorf("out = %+v, want one location with ItemID=2", out)
	}
}

func TestGetCorporationAssetLocationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationAssetLocations
	t.Cleanup(func() { getCorporationAssetLocations = orig })
	getCorporationAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetLocation, error) {
		t.Fatal("getCorporationAssetLocations should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationAssetLocationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationAssetLocationsInput{CharacterID: 999, CorporationID: 98000001, ItemIDs: []int64{2}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationAssetNamesHandler_ReturnsNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationAssetNamesESI{corporationAssetNames: func(_ context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
		return []esiapi.CharactersCharacterIdAssetsNamesPostInner{{ItemId: 2, Name: "Corp Hauler"}}, nil
	}}

	handler := getCorporationAssetNamesHandler(openNarrow[corporationAssetNamesESI](fake), store)
	_, out, err := handler(context.Background(), nil, GetCorporationAssetNamesInput{CharacterID: 7, CorporationID: 98000001, ItemIDs: []int64{2}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Names) != 1 || out.Names[0].Name != "Corp Hauler" {
		t.Errorf("out = %+v, want one name \"Corp Hauler\"", out)
	}
}

func TestGetCorporationAssetNamesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := getCorporationAssetNamesHandler(noNarrow[corporationAssetNamesESI](t), store)
	_, _, err := handler(context.Background(), nil, GetCorporationAssetNamesInput{CharacterID: 999, CorporationID: 98000001, ItemIDs: []int64{2}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// fakeAssetsESI fakes assetsESI; a nil assets fails as an unfaked call.
type fakeAssetsESI struct {
	assets func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error)
}

func (f fakeAssetsESI) Assets(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
	if f.assets == nil {
		return nil, 0, errNotFaked
	}
	return f.assets(ctx, page)
}

// fakeAssetNamesESI fakes assetNamesESI; a nil assetNames fails as an unfaked call.
type fakeAssetNamesESI struct {
	assetNames func(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error)
}

func (f fakeAssetNamesESI) AssetNames(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	if f.assetNames == nil {
		return nil, errNotFaked
	}
	return f.assetNames(ctx, itemIDs)
}

// fakeCorporationAssetsESI fakes corporationAssetsESI; a nil corporationAssets fails as an unfaked call.
type fakeCorporationAssetsESI struct {
	corporationAssets func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error)
}

func (f fakeCorporationAssetsESI) CorporationAssets(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error) {
	if f.corporationAssets == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationAssets(ctx, corporationID, page)
}

// fakeCorporationAssetNamesESI fakes corporationAssetNamesESI; a nil corporationAssetNames fails as an unfaked call.
type fakeCorporationAssetNamesESI struct {
	corporationAssetNames func(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error)
}

func (f fakeCorporationAssetNamesESI) CorporationAssetNames(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	if f.corporationAssetNames == nil {
		return nil, errNotFaked
	}
	return f.corporationAssetNames(ctx, corporationID, itemIDs)
}

// Each handler reports a client that can't be opened, wrapping the cause.
func TestAssetsHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	wantErr := errors.New("no refresh token")
	for name, call := range map[string]func() error{
		"list_assets": func() error {
			_, _, err := listAssetsHandler("v1", failNarrow[assetsESI](wantErr), store, nil)(context.Background(), nil, ListAssetsInput{CharacterID: 7})
			return err
		},
		"get_asset_names": func() error {
			_, _, err := getAssetNamesHandler(failNarrow[assetNamesESI](wantErr), store)(context.Background(), nil, GetAssetNamesInput{CharacterID: 7, ItemIDs: []int64{1}})
			return err
		},
		"list_corporation_assets": func() error {
			_, _, err := listCorporationAssetsHandler("v1", failNarrow[corporationAssetsESI](wantErr), store, nil)(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"get_corporation_asset_names": func() error {
			_, _, err := getCorporationAssetNamesHandler(failNarrow[corporationAssetNamesESI](wantErr), store)(context.Background(), nil, GetCorporationAssetNamesInput{CharacterID: 7, CorporationID: 98000001, ItemIDs: []int64{1}})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}
