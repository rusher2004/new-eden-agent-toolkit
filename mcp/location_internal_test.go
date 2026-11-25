package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
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

	station := int64(60003760)
	fake := fakeCharacterLocationESI{characterLocation: func(_ context.Context) (*esiapi.CharactersLocation, error) {
		return &esiapi.CharactersLocation{SolarSystemId: 30000142, StationId: &station}, nil
	}}

	handler := getCharacterLocationHandler("v1", openNarrow[characterLocationESI](fake), store, nil)
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

	handler := getCharacterLocationHandler("v1", noNarrow[characterLocationESI](t), store, nil)
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

	stationID := int64(60003760)
	fake := fakeCharacterLocationESI{characterLocation: func(_ context.Context) (*esiapi.CharactersLocation, error) {
		return &esiapi.CharactersLocation{SolarSystemId: 30000142, StationId: &stationID}, nil
	}}

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

	handler := getCharacterLocationHandler("v1", openNarrow[characterLocationESI](fake), store, nil)
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

	lastLogin := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	lastLogout := time.Date(2026, 6, 30, 23, 0, 0, 0, time.UTC)
	logins := int64(42)
	fake := fakeCharacterOnlineESI{characterOnline: func(_ context.Context) (*esi.CharacterOnlineStatus, error) {
		return &esi.CharacterOnlineStatus{Online: true, LastLogin: &lastLogin, LastLogout: &lastLogout, Logins: &logins}, nil
	}}

	handler := getCharacterOnlineHandler(openNarrow[characterOnlineESI](fake), store)
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

	handler := getCharacterOnlineHandler(noNarrow[characterOnlineESI](t), store)
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

	fake := fakeCharacterShipESI{characterShip: func(_ context.Context) (*esiapi.CharactersShip, error) {
		return &esiapi.CharactersShip{ShipItemId: 1000000016991, ShipName: "Percival", ShipTypeId: 608}, nil
	}}

	handler := getCharacterShipHandler("v1", openNarrow[characterShipESI](fake), store, nil)
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

	handler := getCharacterShipHandler("v1", noNarrow[characterShipESI](t), store, nil)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterLocationESI{characterLocation: func(_ context.Context) (*esiapi.CharactersLocation, error) {
		return nil, wantErr
	}}

	handler := getCharacterLocationHandler("v1", openNarrow[characterLocationESI](fake), store, nil)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterOnlineESI{characterOnline: func(_ context.Context) (*esi.CharacterOnlineStatus, error) {
		return nil, wantErr
	}}

	handler := getCharacterOnlineHandler(openNarrow[characterOnlineESI](fake), store)
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

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterShipESI{characterShip: func(_ context.Context) (*esiapi.CharactersShip, error) {
		return nil, wantErr
	}}

	handler := getCharacterShipHandler("v1", openNarrow[characterShipESI](fake), store, nil)
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

	fake := fakeCharacterShipESI{characterShip: func(_ context.Context) (*esiapi.CharactersShip, error) {
		return &esiapi.CharactersShip{ShipItemId: 1000000016991, ShipName: "My Rifter", ShipTypeId: 587}, nil
	}}

	origResolveTypeNames := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolveTypeNames })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := getCharacterShipHandler("v1", openNarrow[characterShipESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterShipInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.ShipTypeName, "Rifter")
	}
}

// fakeCharacterLocationESI fakes characterLocationESI; a nil characterLocation fails as an unfaked call.
type fakeCharacterLocationESI struct {
	characterLocation func(ctx context.Context) (*esiapi.CharactersLocation, error)
}

func (f fakeCharacterLocationESI) CharacterLocation(ctx context.Context) (*esiapi.CharactersLocation, error) {
	if f.characterLocation == nil {
		return nil, errNotFaked
	}
	return f.characterLocation(ctx)
}

// fakeCharacterOnlineESI fakes characterOnlineESI; a nil characterOnline fails as an unfaked call.
type fakeCharacterOnlineESI struct {
	characterOnline func(ctx context.Context) (*esi.CharacterOnlineStatus, error)
}

func (f fakeCharacterOnlineESI) CharacterOnline(ctx context.Context) (*esi.CharacterOnlineStatus, error) {
	if f.characterOnline == nil {
		return nil, errNotFaked
	}
	return f.characterOnline(ctx)
}

// fakeCharacterShipESI fakes characterShipESI; a nil characterShip fails as an unfaked call.
type fakeCharacterShipESI struct {
	characterShip func(ctx context.Context) (*esiapi.CharactersShip, error)
}

func (f fakeCharacterShipESI) CharacterShip(ctx context.Context) (*esiapi.CharactersShip, error) {
	if f.characterShip == nil {
		return nil, errNotFaked
	}
	return f.characterShip(ctx)
}

// Each handler reports a client that can't be opened, wrapping the cause.
func TestLocationHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	wantErr := errors.New("no refresh token")
	for name, call := range map[string]func() error{
		"get_character_location": func() error {
			_, _, err := getCharacterLocationHandler("v1", failNarrow[characterLocationESI](wantErr), store, nil)(context.Background(), nil, GetCharacterLocationInput{CharacterID: 7})
			return err
		},
		"get_character_online": func() error {
			_, _, err := getCharacterOnlineHandler(failNarrow[characterOnlineESI](wantErr), store)(context.Background(), nil, GetCharacterOnlineInput{CharacterID: 7})
			return err
		},
		"get_character_ship": func() error {
			_, _, err := getCharacterShipHandler("v1", failNarrow[characterShipESI](wantErr), store, nil)(context.Background(), nil, GetCharacterShipInput{CharacterID: 7})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}
