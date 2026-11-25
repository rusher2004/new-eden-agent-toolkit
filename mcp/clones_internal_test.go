package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func clonesTestStore(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

func TestGetCharacterClonesHandler_ReturnsClones(t *testing.T) {
	store := clonesTestStore(t)
	locID := int64(60003760)
	fake := fakeCharacterClonesESI{characterClones: func(context.Context) (*esiapi.CharactersCharacterIdClonesGet, error) {
		return &esiapi.CharactersCharacterIdClonesGet{
			HomeLocation: &esiapi.CharactersCharacterIdClonesGetHomeLocation{LocationId: &locID},
			JumpClones:   []esiapi.CharactersCharacterIdClonesGetJumpClonesInner{{JumpCloneId: 1, LocationId: 60008494, LocationType: "station", Implants: []int64{19540}}},
		}, nil
	}}

	handler := getCharacterClonesHandler(openNarrow[characterClonesESI](fake), store)
	_, out, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.HomeLocationID == nil || *out.HomeLocationID != 60003760 || len(out.JumpClones) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7, HomeLocationID=*60003760 and one jump clone", out)
	}
	if jc := out.JumpClones[0]; jc.JumpCloneID != 1 || jc.LocationID != 60008494 || jc.LocationType != "station" || len(jc.Implants) != 1 {
		t.Errorf("JumpClones[0] = %+v, want clone 1 at station 60008494 with one implant", jc)
	}
}

func TestGetCharacterClonesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := getCharacterClonesHandler(noNarrow[characterClonesESI](t), store)
	_, _, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterClonesHandler_PropagatesError(t *testing.T) {
	store := clonesTestStore(t)
	wantErr := errors.New("upstream failure")
	fake := fakeCharacterClonesESI{characterClones: func(context.Context) (*esiapi.CharactersCharacterIdClonesGet, error) { return nil, wantErr }}

	handler := getCharacterClonesHandler(openNarrow[characterClonesESI](fake), store)
	_, _, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCharacterClonesHandler_PropagatesOpenError(t *testing.T) {
	store := clonesTestStore(t)
	wantErr := errors.New("no token")

	handler := getCharacterClonesHandler(failNarrow[characterClonesESI](wantErr), store)
	_, _, err := handler(context.Background(), nil, GetCharacterClonesInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterImplantsHandler_ReturnsImplants(t *testing.T) {
	store := clonesTestStore(t)
	fake := fakeCharacterImplantsESI{characterImplants: func(context.Context) ([]int64, error) { return []int64{19540, 19551}, nil }}

	handler := listCharacterImplantsHandler(openNarrow[characterImplantsESI](fake), store)
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

	handler := listCharacterImplantsHandler(noNarrow[characterImplantsESI](t), store)
	_, _, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterImplantsHandler_PropagatesError(t *testing.T) {
	store := clonesTestStore(t)
	wantErr := errors.New("upstream failure")
	fake := fakeCharacterImplantsESI{characterImplants: func(context.Context) ([]int64, error) { return nil, wantErr }}

	handler := listCharacterImplantsHandler(openNarrow[characterImplantsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterImplantsHandler_PropagatesOpenError(t *testing.T) {
	store := clonesTestStore(t)
	wantErr := errors.New("no token")

	handler := listCharacterImplantsHandler(failNarrow[characterImplantsESI](wantErr), store)
	_, _, err := handler(context.Background(), nil, ListCharacterImplantsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// fakeCharacterClonesESI fakes characterClonesESI; a nil characterClones fails as an unfaked call.
type fakeCharacterClonesESI struct {
	characterClones func(ctx context.Context) (*esiapi.CharactersCharacterIdClonesGet, error)
}

func (f fakeCharacterClonesESI) CharacterClones(ctx context.Context) (*esiapi.CharactersCharacterIdClonesGet, error) {
	if f.characterClones == nil {
		return nil, errNotFaked
	}
	return f.characterClones(ctx)
}

// fakeCharacterImplantsESI fakes characterImplantsESI; a nil characterImplants fails as an unfaked call.
type fakeCharacterImplantsESI struct {
	characterImplants func(ctx context.Context) ([]int64, error)
}

func (f fakeCharacterImplantsESI) CharacterImplants(ctx context.Context) ([]int64, error) {
	if f.characterImplants == nil {
		return nil, errNotFaked
	}
	return f.characterImplants(ctx)
}
