package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetCharacterClonesHandler_ReturnsClones(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterClones
	t.Cleanup(func() { getCharacterClones = orig })
	locID := int64(60003760)
	getCharacterClones = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterClonesOutput, error) {
		return GetCharacterClonesOutput{CharacterID: characterID, HomeLocationID: &locID, JumpClones: []JumpClone{{JumpCloneID: 1}}}, nil
	}

	handler := getCharacterClonesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.HomeLocationID == nil || *out.HomeLocationID != 60003760 || len(out.JumpClones) != 1 {
		t.Errorf("out = %+v, want HomeLocationID=*60003760 and one jump clone", out)
	}
}

func TestGetCharacterClonesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterClones
	t.Cleanup(func() { getCharacterClones = orig })
	getCharacterClones = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterClonesOutput, error) {
		t.Fatal("getCharacterClones should not be called for an unknown character")
		return GetCharacterClonesOutput{}, nil
	}

	handler := getCharacterClonesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterImplantsHandler_ReturnsImplants(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterImplants
	t.Cleanup(func() { listCharacterImplants = orig })
	listCharacterImplants = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		return []int64{19540, 19551}, nil
	}

	handler := listCharacterImplantsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.ImplantTypeIDs) != 2 || out.ImplantTypeIDs[0] != 19540 {
		t.Errorf("out = %+v, want ImplantTypeIDs=[19540 19551]", out)
	}
}

func TestListCharacterImplantsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterImplants
	t.Cleanup(func() { listCharacterImplants = orig })
	listCharacterImplants = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		t.Fatal("listCharacterImplants should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterImplantsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterClonesHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterClones
	t.Cleanup(func() { getCharacterClones = orig })
	wantErr := errors.New("upstream failure")
	getCharacterClones = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterClonesOutput, error) {
		return GetCharacterClonesOutput{}, wantErr
	}

	handler := getCharacterClonesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterImplantsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterImplants
	t.Cleanup(func() { listCharacterImplants = orig })
	wantErr := errors.New("upstream failure")
	listCharacterImplants = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listCharacterImplantsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
