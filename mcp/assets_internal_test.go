package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListAssetsHandler_ReturnsAssets(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listAssets
	t.Cleanup(func() { listAssets = orig })
	listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]AssetItem, int32, error) {
		return []AssetItem{{ItemID: 1, TypeID: 587, LocationID: 60003760, LocationType: "station", LocationFlag: "Hangar", Quantity: 1, IsSingleton: true}}, 3, nil
	}

	handler := listAssetsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 7, Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Assets) != 1 || out.Assets[0].ItemID != 1 || out.Page != 2 || out.TotalPages != 3 {
		t.Errorf("out = %+v, want CharacterID=7 Page=2 TotalPages=3 with one asset ItemID=1", out)
	}
}

func TestListAssetsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listAssets
	t.Cleanup(func() { listAssets = orig })
	wantErr := errors.New("upstream failure")
	listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]AssetItem, int32, error) {
		return nil, 0, wantErr
	}

	handler := listAssetsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListAssetsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListAssetsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listAssets
	t.Cleanup(func() { listAssets = orig })
	listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]AssetItem, int32, error) {
		t.Fatal("listAssets should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listAssetsHandler("v1", "client-id", store)
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

	orig := getAssetNames
	t.Cleanup(func() { getAssetNames = orig })
	getAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetName, error) {
		return []AssetName{{ItemID: 1, Name: "My Renamed Ship"}}, nil
	}

	handler := getAssetNamesHandler("v1", "client-id", store)
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

	orig := getAssetNames
	t.Cleanup(func() { getAssetNames = orig })
	getAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetName, error) {
		t.Fatal("getAssetNames should not be called for an unknown character")
		return nil, nil
	}

	handler := getAssetNamesHandler("v1", "client-id", store)
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

	orig := listCorporationAssets
	t.Cleanup(func() { listCorporationAssets = orig })
	listCorporationAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]AssetItem, int32, error) {
		if corporationID != 98000001 {
			t.Errorf("corporationID = %d, want 98000001", corporationID)
		}
		return []AssetItem{{ItemID: 2, TypeID: 34, Quantity: 10}}, 5, nil
	}

	handler := listCorporationAssetsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 7, CorporationID: 98000001, Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Assets) != 1 || out.Page != 2 || out.TotalPages != 5 {
		t.Errorf("out = %+v, want CorporationID=98000001 Page=2 TotalPages=5 with one asset", out)
	}
}

func TestListCorporationAssetsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationAssets
	t.Cleanup(func() { listCorporationAssets = orig })
	wantErr := errors.New("upstream failure")
	listCorporationAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]AssetItem, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationAssetsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationAssetsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationAssetsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationAssets
	t.Cleanup(func() { listCorporationAssets = orig })
	listCorporationAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]AssetItem, int32, error) {
		t.Fatal("listCorporationAssets should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationAssetsHandler("v1", "client-id", store)
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

	orig := getCorporationAssetNames
	t.Cleanup(func() { getCorporationAssetNames = orig })
	getCorporationAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetName, error) {
		return []AssetName{{ItemID: 2, Name: "Corp Hauler"}}, nil
	}

	handler := getCorporationAssetNamesHandler("v1", "client-id", store)
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

	orig := getCorporationAssetNames
	t.Cleanup(func() { getCorporationAssetNames = orig })
	getCorporationAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetName, error) {
		t.Fatal("getCorporationAssetNames should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationAssetNamesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationAssetNamesInput{CharacterID: 999, CorporationID: 98000001, ItemIDs: []int64{2}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
