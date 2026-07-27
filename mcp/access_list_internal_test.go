package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListAccessListsHandler_ReturnsIDs(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listAccessLists
	t.Cleanup(func() { listAccessLists = orig })
	listAccessLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		return []int64{1, 2}, nil
	}

	handler := listAccessListsHandler("v1", "client-id", store)
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

	orig := listAccessLists
	t.Cleanup(func() { listAccessLists = orig })
	listAccessLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		t.Fatal("listAccessLists should not be called for an unknown character")
		return nil, nil
	}

	handler := listAccessListsHandler("v1", "client-id", store)
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

	orig := getAccessListDetail
	t.Cleanup(func() { getAccessListDetail = orig })
	getAccessListDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, accessListID int64) (AccessListDetailResult, error) {
		return AccessListDetailResult{
			Name:          "My List",
			Description:   "desc",
			AllowEveryone: true,
			Characters:    []AccessListEntry{{EntityID: 123, Access: "read"}},
		}, nil
	}

	handler := getAccessListHandler("v1", "client-id", store)
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

	orig := getAccessListDetail
	t.Cleanup(func() { getAccessListDetail = orig })
	getAccessListDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, accessListID int64) (AccessListDetailResult, error) {
		t.Fatal("getAccessListDetail should not be called for an unknown character")
		return AccessListDetailResult{}, nil
	}

	handler := getAccessListHandler("v1", "client-id", store)
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

	orig := listAccessLists
	t.Cleanup(func() { listAccessLists = orig })
	wantErr := errors.New("upstream failure")
	listAccessLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listAccessListsHandler("v1", "client-id", store)
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

	orig := getAccessListDetail
	t.Cleanup(func() { getAccessListDetail = orig })
	wantErr := errors.New("upstream failure")
	getAccessListDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, accessListID int64) (AccessListDetailResult, error) {
		return AccessListDetailResult{}, wantErr
	}

	handler := getAccessListHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetAccessListInput{CharacterID: 7, AccessListID: 1})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
