package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestGetCharacterLocationHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterLocation
	t.Cleanup(func() { getCharacterLocation = orig })
	station := int64(60003760)
	getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
		return 30000142, &station, nil, nil
	}

	handler := getCharacterLocationHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterLocationInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.SolarSystemID != 30000142 {
		t.Fatalf("out = %+v, want CharacterID=7 SolarSystemID=30000142", out)
	}
	if out.StationID == nil || *out.StationID != 60003760 {
		t.Errorf("StationID = %v, want pointer to 60003760", out.StationID)
	}
	if out.StructureID != nil {
		t.Errorf("StructureID = %v, want nil", out.StructureID)
	}
}

func TestGetCharacterLocationHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterLocation
	t.Cleanup(func() { getCharacterLocation = orig })
	getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
		t.Fatal("getCharacterLocation should not be called for an unknown character")
		return 0, nil, nil, nil
	}

	handler := getCharacterLocationHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterLocationInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterLocationHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterLocation
	t.Cleanup(func() { getCharacterLocation = orig })
	stationID := int64(60003760)
	getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
		return 30000142, &stationID, nil, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := getCharacterLocationHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterLocationInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.SolarSystemName, "Jita")
	}
	if out.StationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("StationName = %q, want %q", out.StationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}

func TestGetCharacterOnlineHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterOnline
	t.Cleanup(func() { getCharacterOnline = orig })
	lastLogin := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	lastLogout := time.Date(2026, 6, 30, 23, 0, 0, 0, time.UTC)
	logins := int64(42)
	getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
		return &esi.CharacterOnlineStatus{Online: true, LastLogin: &lastLogin, LastLogout: &lastLogout, Logins: &logins}, nil
	}

	handler := getCharacterOnlineHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterOnlineInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || !out.Online {
		t.Fatalf("out = %+v, want CharacterID=7 Online=true", out)
	}
	if out.LastLogin == nil || !out.LastLogin.Equal(lastLogin) {
		t.Errorf("LastLogin = %v, want %v", out.LastLogin, lastLogin)
	}
	if out.LastLogout == nil || !out.LastLogout.Equal(lastLogout) {
		t.Errorf("LastLogout = %v, want %v", out.LastLogout, lastLogout)
	}
	if out.Logins == nil || *out.Logins != 42 {
		t.Errorf("Logins = %v, want pointer to 42", out.Logins)
	}
}

func TestGetCharacterOnlineHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterOnline
	t.Cleanup(func() { getCharacterOnline = orig })
	getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
		t.Fatal("getCharacterOnline should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterOnlineHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterOnlineInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterShipHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterShip
	t.Cleanup(func() { getCharacterShip = orig })
	getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
		return 1000000016991, "Percival", 608, nil
	}

	handler := getCharacterShipHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.ShipItemID != 1000000016991 || out.ShipName != "Percival" || out.ShipTypeID != 608 {
		t.Fatalf("out = %+v, want CharacterID=7 ShipItemID=1000000016991 ShipName=Percival ShipTypeID=608", out)
	}
}

func TestGetCharacterShipHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterShip
	t.Cleanup(func() { getCharacterShip = orig })
	getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
		t.Fatal("getCharacterShip should not be called for an unknown character")
		return 0, "", 0, nil
	}

	handler := getCharacterShipHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterLocationHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterLocation
	t.Cleanup(func() { getCharacterLocation = orig })
	wantErr := errors.New("upstream failure")
	getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
		return 0, nil, nil, wantErr
	}

	handler := getCharacterLocationHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterLocationInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCharacterOnlineHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterOnline
	t.Cleanup(func() { getCharacterOnline = orig })
	wantErr := errors.New("upstream failure")
	getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
		return nil, wantErr
	}

	handler := getCharacterOnlineHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterOnlineInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCharacterShipHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterShip
	t.Cleanup(func() { getCharacterShip = orig })
	wantErr := errors.New("upstream failure")
	getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
		return 0, "", 0, wantErr
	}

	handler := getCharacterShipHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCharacterShipHandler_ResolvesShipTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterShip
	t.Cleanup(func() { getCharacterShip = orig })
	getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
		return 1000000016991, "My Rifter", 587, nil
	}

	origResolveTypeNames := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolveTypeNames })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := getCharacterShipHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.ShipTypeName, "Rifter")
	}
}
