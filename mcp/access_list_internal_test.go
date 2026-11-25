package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// fakeAccessListESI fakes the access list tools' ESI calls; err is returned by every call.
type fakeAccessListESI struct {
	lists  []esiapi.CharactersAccessListsListingAccesslist
	detail *esiapi.CharactersAccessListsDetail
	err    error
}

func (f fakeAccessListESI) AccessLists(context.Context) ([]esiapi.CharactersAccessListsListingAccesslist, error) {
	return f.lists, f.err
}

func (f fakeAccessListESI) AccessListDetail(context.Context, int64) (*esiapi.CharactersAccessListsDetail, error) {
	return f.detail, f.err
}

func openAccessListFake(c accessListESI) func(context.Context, int32) (accessListESI, error) {
	return func(context.Context, int32) (accessListESI, error) { return c, nil }
}

func TestListAccessListsHandler_ReturnsIDs(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAccessListESI{lists: []esiapi.CharactersAccessListsListingAccesslist{{Id: 1}, {Id: 2}}}

	handler := listAccessListsHandler(openAccessListFake(fake), store)
	_, out, err := handler(context.Background(), nil, ListAccessListsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.AccessListIDs) != 2 || out.AccessListIDs[0] != 1 || out.AccessListIDs[1] != 2 {
		t.Errorf("out = %+v, want {CharacterID:7 AccessListIDs:[1 2]}", out)
	}
}

func TestListAccessListsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	open := func(context.Context, int32) (accessListESI, error) {
		t.Fatal("no ESI client should be opened for an unknown character")
		return nil, nil
	}

	handler := listAccessListsHandler(open, store)
	_, _, err := handler(context.Background(), nil, ListAccessListsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetAccessListHandler_ReturnsDetail(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAccessListESI{detail: &esiapi.CharactersAccessListsDetail{
		Name:        "My List",
		Description: "desc",
		Membership: esiapi.CharactersAccessListsDetailMembership{
			AllowEveryone: true,
			Characters:    []esiapi.CharactersAccessListsDetailCharacterentry{{CharacterId: 123, Access: "read"}},
		},
	}}

	handler := getAccessListHandler(openAccessListFake(fake), store)
	_, out, err := handler(context.Background(), nil, GetAccessListInput{CharacterID: 7, AccessListID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Name != "My List" || out.AccessListID != 1 || !out.AllowEveryone {
		t.Errorf("out = %+v, want Name=\"My List\" AccessListID=1 AllowEveryone=true", out)
	}
	if len(out.Characters) != 1 || out.Characters[0].EntityID != 123 {
		t.Errorf("out.Characters = %+v, want one entry with EntityID 123", out.Characters)
	}
}

func TestGetAccessListHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	open := func(context.Context, int32) (accessListESI, error) {
		t.Fatal("no ESI client should be opened for an unknown character")
		return nil, nil
	}

	handler := getAccessListHandler(open, store)
	_, _, err := handler(context.Background(), nil, GetAccessListInput{CharacterID: 999, AccessListID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListAccessListsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")

	handler := listAccessListsHandler(openAccessListFake(fakeAccessListESI{err: wantErr}), store)
	_, _, err := handler(context.Background(), nil, ListAccessListsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetAccessListHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")

	handler := getAccessListHandler(openAccessListFake(fakeAccessListESI{err: wantErr}), store)
	_, _, err := handler(context.Background(), nil, GetAccessListInput{CharacterID: 7, AccessListID: 1})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListAccessListsHandler_PropagatesOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	open := func(context.Context, int32) (accessListESI, error) { return nil, errBoom }
	handler := listAccessListsHandler(open, store)
	_, _, err := handler(context.Background(), nil, ListAccessListsInput{CharacterID: 7})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}

func TestGetAccessListHandler_PropagatesOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	open := func(context.Context, int32) (accessListESI, error) { return nil, errBoom }
	handler := getAccessListHandler(open, store)
	_, _, err := handler(context.Background(), nil, GetAccessListInput{CharacterID: 7, AccessListID: 1})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}
