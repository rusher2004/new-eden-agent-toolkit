package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterContactsHandler_ReturnsContacts(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCharacterContactsESI{characterContacts: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdContactsGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdContactsGetInner{{ContactId: 998878, Standing: 5.5}}, 2, nil
	}}

	handler := listCharacterContactsHandler(openNarrow[characterContactsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListCharacterContactsInput{CharacterID: 7, Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Contacts) != 1 || out.Contacts[0].Standing != 5.5 || out.Page != 1 || out.TotalPages != 2 {
		t.Errorf("out = %+v, want one contact with Standing=5.5 Page=1 TotalPages=2", out)
	}
}

func TestListCharacterContactsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterContactsESI{characterContacts: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdContactsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCharacterContactsHandler(openNarrow[characterContactsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCharacterContactsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterContactsHandler(noNarrow[characterContactsESI](t), store)
	_, _, err := handler(context.Background(), nil, ListCharacterContactsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContactLabelsHandler_ReturnsLabels(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCharacterContactLabelsESI{characterContactLabels: func(ctx context.Context) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
		return []esiapi.AlliancesAllianceIdContactsLabelsGetInner{{LabelId: 1, LabelName: "Friends"}}, nil
	}}

	handler := listCharacterContactLabelsHandler(openNarrow[characterContactLabelsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListCharacterContactLabelsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Labels) != 1 || out.Labels[0].LabelName != "Friends" {
		t.Errorf("out = %+v, want one label \"Friends\"", out)
	}
}

func TestListCharacterContactLabelsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterContactLabelsHandler(noNarrow[characterContactLabelsESI](t), store)
	_, _, err := handler(context.Background(), nil, ListCharacterContactLabelsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContactsHandler_ReturnsContacts(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationContactsESI{corporationContacts: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContactsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdContactsGetInner{{ContactId: 998879}}, 2, nil
	}}

	handler := listCorporationContactsHandler(openNarrow[corporationContactsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListCorporationContactsInput{CharacterID: 7, CorporationID: 98000001, Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Contacts) != 1 || out.Page != 1 || out.TotalPages != 2 {
		t.Errorf("out = %+v, want CorporationID=98000001 Page=1 TotalPages=2 with one contact", out)
	}
}

func TestListCorporationContactsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationContactsESI{corporationContacts: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContactsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationContactsHandler(openNarrow[corporationContactsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCorporationContactsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationContactsHandler(noNarrow[corporationContactsESI](t), store)
	_, _, err := handler(context.Background(), nil, ListCorporationContactsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContactLabelsHandler_ReturnsLabels(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationContactLabelsESI{corporationContactLabels: func(ctx context.Context, corporationID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
		return []esiapi.AlliancesAllianceIdContactsLabelsGetInner{{LabelId: 2, LabelName: "Corp VIP"}}, nil
	}}

	handler := listCorporationContactLabelsHandler(openNarrow[corporationContactLabelsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListCorporationContactLabelsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Labels) != 1 || out.Labels[0].LabelName != "Corp VIP" {
		t.Errorf("out = %+v, want one label \"Corp VIP\"", out)
	}
}

func TestListCorporationContactLabelsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationContactLabelsHandler(noNarrow[corporationContactLabelsESI](t), store)
	_, _, err := handler(context.Background(), nil, ListCorporationContactLabelsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListAllianceContactsHandler_ReturnsContacts(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAllianceContactsESI{allianceContacts: func(_ context.Context, allianceID int64, page int32) ([]esiapi.AlliancesAllianceIdContactsGetInner, int32, error) {
		return []esiapi.AlliancesAllianceIdContactsGetInner{{ContactId: 998880, Standing: -5.0}}, 2, nil
	}}

	handler := listAllianceContactsHandler(openNarrow[allianceContactsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListAllianceContactsInput{CharacterID: 7, AllianceID: 99000001, Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || len(out.Contacts) != 1 || out.Page != 1 || out.TotalPages != 2 {
		t.Errorf("out = %+v, want AllianceID=99000001 Page=1 TotalPages=2 with one contact", out)
	}
}

func TestListAllianceContactsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeAllianceContactsESI{allianceContacts: func(_ context.Context, allianceID int64, page int32) ([]esiapi.AlliancesAllianceIdContactsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listAllianceContactsHandler(openNarrow[allianceContactsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListAllianceContactsInput{CharacterID: 7, AllianceID: 99000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListAllianceContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listAllianceContactsHandler(noNarrow[allianceContactsESI](t), store)
	_, _, err := handler(context.Background(), nil, ListAllianceContactsInput{CharacterID: 999, AllianceID: 99000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListAllianceContactLabelsHandler_ReturnsLabels(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAllianceContactLabelsESI{allianceContactLabels: func(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
		return []esiapi.AlliancesAllianceIdContactsLabelsGetInner{{LabelId: 3, LabelName: "Alliance Blues"}}, nil
	}}

	handler := listAllianceContactLabelsHandler(openNarrow[allianceContactLabelsESI](fake), store)
	_, out, err := handler(context.Background(), nil, ListAllianceContactLabelsInput{CharacterID: 7, AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Labels) != 1 || out.Labels[0].LabelName != "Alliance Blues" {
		t.Errorf("out = %+v, want one label \"Alliance Blues\"", out)
	}
}

func TestListAllianceContactLabelsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listAllianceContactLabelsHandler(noNarrow[allianceContactLabelsESI](t), store)
	_, _, err := handler(context.Background(), nil, ListAllianceContactLabelsInput{CharacterID: 999, AllianceID: 99000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestAddCharacterContactsHandler_ReturnsAddedIDs(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeAddCharacterContactsESI{addCharacterContacts: func(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
		return []int64{998878}, nil
	}}

	handler := addCharacterContactsHandler(openNarrow[addCharacterContactsESI](fake), store)
	_, out, err := handler(context.Background(), nil, AddCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{998878}, Standing: 5.0})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.AddedContactIDs) != 1 || out.AddedContactIDs[0] != 998878 {
		t.Errorf("out = %+v, want AddedContactIDs=[998878]", out)
	}
}

func TestAddCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := addCharacterContactsHandler(noNarrow[addCharacterContactsESI](t), store)
	_, _, err := handler(context.Background(), nil, AddCharacterContactsInput{CharacterID: 999, ContactIDs: []int64{998878}, Standing: 5.0})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestUpdateCharacterContactsHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	var gotStanding float64
	fake := fakeUpdateCharacterContactsESI{updateCharacterContacts: func(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
		gotStanding = standing
		return nil
	}}

	handler := updateCharacterContactsHandler(openNarrow[updateCharacterContactsESI](fake), store)
	_, out, err := handler(context.Background(), nil, UpdateCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{998878}, Standing: 3.0})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotStanding != 3.0 || len(out.ContactIDs) != 1 {
		t.Errorf("gotStanding = %v, out = %+v, want Standing=3.0 with one contact ID echoed", gotStanding, out)
	}
}

func TestUpdateCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := updateCharacterContactsHandler(noNarrow[updateCharacterContactsESI](t), store)
	_, _, err := handler(context.Background(), nil, UpdateCharacterContactsInput{CharacterID: 999, ContactIDs: []int64{998878}, Standing: 3.0})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRemoveCharacterContactsHandler_ReturnsRemovedIDs(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeRemoveCharacterContactsESI{removeCharacterContacts: func(ctx context.Context, contactIDs []int64) error {
		return nil
	}}

	handler := removeCharacterContactsHandler(openNarrow[removeCharacterContactsESI](fake), store)
	_, out, err := handler(context.Background(), nil, RemoveCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{998878}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.RemovedContactIDs) != 1 || out.RemovedContactIDs[0] != 998878 {
		t.Errorf("out = %+v, want RemovedContactIDs=[998878]", out)
	}
}

func TestRemoveCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := removeCharacterContactsHandler(noNarrow[removeCharacterContactsESI](t), store)
	_, _, err := handler(context.Background(), nil, RemoveCharacterContactsInput{CharacterID: 999, ContactIDs: []int64{998878}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// fakeCharacterContactsESI fakes characterContactsESI; a nil characterContacts fails as an unfaked call.
type fakeCharacterContactsESI struct {
	characterContacts func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContactsGetInner, int32, error)
}

func (f fakeCharacterContactsESI) CharacterContacts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContactsGetInner, int32, error) {
	if f.characterContacts == nil {
		return nil, 0, errNotFaked
	}
	return f.characterContacts(ctx, page)
}

// fakeCorporationContactsESI fakes corporationContactsESI; a nil corporationContacts fails as an unfaked call.
type fakeCorporationContactsESI struct {
	corporationContacts func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContactsGetInner, int32, error)
}

func (f fakeCorporationContactsESI) CorporationContacts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContactsGetInner, int32, error) {
	if f.corporationContacts == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationContacts(ctx, corporationID, page)
}

// fakeAllianceContactsESI fakes allianceContactsESI; a nil allianceContacts fails as an unfaked call.
type fakeAllianceContactsESI struct {
	allianceContacts func(ctx context.Context, allianceID int64, page int32) ([]esiapi.AlliancesAllianceIdContactsGetInner, int32, error)
}

func (f fakeAllianceContactsESI) AllianceContacts(ctx context.Context, allianceID int64, page int32) ([]esiapi.AlliancesAllianceIdContactsGetInner, int32, error) {
	if f.allianceContacts == nil {
		return nil, 0, errNotFaked
	}
	return f.allianceContacts(ctx, allianceID, page)
}

// Each handler reports a client that can't be opened, wrapping the cause.
func TestContactsHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	wantErr := errors.New("no refresh token")
	for name, call := range map[string]func() error{
		"list_character_contacts": func() error {
			_, _, err := listCharacterContactsHandler(failNarrow[characterContactsESI](wantErr), store)(context.Background(), nil, ListCharacterContactsInput{CharacterID: 7})
			return err
		},
		"list_corporation_contacts": func() error {
			_, _, err := listCorporationContactsHandler(failNarrow[corporationContactsESI](wantErr), store)(context.Background(), nil, ListCorporationContactsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_alliance_contacts": func() error {
			_, _, err := listAllianceContactsHandler(failNarrow[allianceContactsESI](wantErr), store)(context.Background(), nil, ListAllianceContactsInput{CharacterID: 7, AllianceID: 99000001})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}

// fakeCharacterContactLabelsESI fakes characterContactLabelsESI; a nil characterContactLabels fails as an unfaked call.
type fakeCharacterContactLabelsESI struct {
	characterContactLabels func(ctx context.Context) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
}

func (f fakeCharacterContactLabelsESI) CharacterContactLabels(ctx context.Context) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	if f.characterContactLabels == nil {
		return nil, errNotFaked
	}
	return f.characterContactLabels(ctx)
}

// fakeCorporationContactLabelsESI fakes corporationContactLabelsESI; a nil corporationContactLabels fails as an unfaked call.
type fakeCorporationContactLabelsESI struct {
	corporationContactLabels func(ctx context.Context, corporationID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
}

func (f fakeCorporationContactLabelsESI) CorporationContactLabels(ctx context.Context, corporationID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	if f.corporationContactLabels == nil {
		return nil, errNotFaked
	}
	return f.corporationContactLabels(ctx, corporationID)
}

// fakeAllianceContactLabelsESI fakes allianceContactLabelsESI; a nil allianceContactLabels fails as an unfaked call.
type fakeAllianceContactLabelsESI struct {
	allianceContactLabels func(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
}

func (f fakeAllianceContactLabelsESI) AllianceContactLabels(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	if f.allianceContactLabels == nil {
		return nil, errNotFaked
	}
	return f.allianceContactLabels(ctx, allianceID)
}

// fakeAddCharacterContactsESI fakes addCharacterContactsESI; a nil addCharacterContacts fails as an unfaked call.
type fakeAddCharacterContactsESI struct {
	addCharacterContacts func(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error)
}

func (f fakeAddCharacterContactsESI) AddCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
	if f.addCharacterContacts == nil {
		return nil, errNotFaked
	}
	return f.addCharacterContacts(ctx, contactIDs, standing, labelIDs, watched)
}

// fakeUpdateCharacterContactsESI fakes updateCharacterContactsESI; a nil updateCharacterContacts fails as an unfaked call.
type fakeUpdateCharacterContactsESI struct {
	updateCharacterContacts func(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error
}

func (f fakeUpdateCharacterContactsESI) UpdateCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
	if f.updateCharacterContacts == nil {
		return errNotFaked
	}
	return f.updateCharacterContacts(ctx, contactIDs, standing, labelIDs, watched)
}

// fakeRemoveCharacterContactsESI fakes removeCharacterContactsESI; a nil removeCharacterContacts fails as an unfaked call.
type fakeRemoveCharacterContactsESI struct {
	removeCharacterContacts func(ctx context.Context, contactIDs []int64) error
}

func (f fakeRemoveCharacterContactsESI) RemoveCharacterContacts(ctx context.Context, contactIDs []int64) error {
	if f.removeCharacterContacts == nil {
		return errNotFaked
	}
	return f.removeCharacterContacts(ctx, contactIDs)
}

// Each label and write handler reports a client that can't be opened, and an
// ESI call that fails, wrapping the cause.
func TestContactsLabelAndWriteHandlers_PropagateOpenAndESIErrors(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	ctx := context.Background()
	openErr := errors.New("no refresh token")
	for name, call := range map[string]openOrESIFailure{
		"list_character_contact_labels": func(failOpen bool) error {
			open := openNarrow[characterContactLabelsESI](fakeCharacterContactLabelsESI{})
			if failOpen {
				open = failNarrow[characterContactLabelsESI](openErr)
			}
			_, _, err := listCharacterContactLabelsHandler(open, store)(ctx, nil, ListCharacterContactLabelsInput{CharacterID: 7})
			return err
		},
		"list_corporation_contact_labels": func(failOpen bool) error {
			open := openNarrow[corporationContactLabelsESI](fakeCorporationContactLabelsESI{})
			if failOpen {
				open = failNarrow[corporationContactLabelsESI](openErr)
			}
			_, _, err := listCorporationContactLabelsHandler(open, store)(ctx, nil, ListCorporationContactLabelsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_alliance_contact_labels": func(failOpen bool) error {
			open := openNarrow[allianceContactLabelsESI](fakeAllianceContactLabelsESI{})
			if failOpen {
				open = failNarrow[allianceContactLabelsESI](openErr)
			}
			_, _, err := listAllianceContactLabelsHandler(open, store)(ctx, nil, ListAllianceContactLabelsInput{CharacterID: 7, AllianceID: 99000001})
			return err
		},
		"add_character_contacts": func(failOpen bool) error {
			open := openNarrow[addCharacterContactsESI](fakeAddCharacterContactsESI{})
			if failOpen {
				open = failNarrow[addCharacterContactsESI](openErr)
			}
			_, _, err := addCharacterContactsHandler(open, store)(ctx, nil, AddCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{1}})
			return err
		},
		"update_character_contacts": func(failOpen bool) error {
			open := openNarrow[updateCharacterContactsESI](fakeUpdateCharacterContactsESI{})
			if failOpen {
				open = failNarrow[updateCharacterContactsESI](openErr)
			}
			_, _, err := updateCharacterContactsHandler(open, store)(ctx, nil, UpdateCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{1}})
			return err
		},
		"remove_character_contacts": func(failOpen bool) error {
			open := openNarrow[removeCharacterContactsESI](fakeRemoveCharacterContactsESI{})
			if failOpen {
				open = failNarrow[removeCharacterContactsESI](openErr)
			}
			_, _, err := removeCharacterContactsHandler(open, store)(ctx, nil, RemoveCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{1}})
			return err
		},
	} {
		if err := call(true); !errors.Is(err, openErr) {
			t.Errorf("%s: open err = %v, want wrapping %v", name, err, openErr)
		}
		if err := call(false); !errors.Is(err, errNotFaked) {
			t.Errorf("%s: ESI err = %v, want wrapping errNotFaked", name, err)
		}
	}
}

// add_character_contacts and remove_character_contacts pass their arguments to ESI.
func TestContactsWriteHandlers_PassArgumentsToESI(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	ctx := context.Background()

	var addGot struct {
		ids      []int64
		standing float64
		labels   []int64
		watched  bool
	}
	add := fakeAddCharacterContactsESI{addCharacterContacts: func(_ context.Context, ids []int64, standing float64, labels []int64, watched bool) ([]int64, error) {
		addGot.ids, addGot.standing, addGot.labels, addGot.watched = ids, standing, labels, watched
		return ids, nil
	}}
	if _, _, err := addCharacterContactsHandler(openNarrow[addCharacterContactsESI](add), store)(ctx, nil, AddCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{5}, Standing: -5, LabelIDs: []int64{9}, Watched: true}); err != nil {
		t.Fatalf("add handler returned error: %v", err)
	}
	if len(addGot.ids) != 1 || addGot.ids[0] != 5 || addGot.standing != -5 || len(addGot.labels) != 1 || addGot.labels[0] != 9 || !addGot.watched {
		t.Errorf("add ESI got %+v, want ids=[5] standing=-5 labels=[9] watched=true", addGot)
	}

	var removeGot []int64
	remove := fakeRemoveCharacterContactsESI{removeCharacterContacts: func(_ context.Context, ids []int64) error {
		removeGot = ids
		return nil
	}}
	if _, _, err := removeCharacterContactsHandler(openNarrow[removeCharacterContactsESI](remove), store)(ctx, nil, RemoveCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{5, 6}}); err != nil {
		t.Fatalf("remove handler returned error: %v", err)
	}
	if len(removeGot) != 2 || removeGot[0] != 5 || removeGot[1] != 6 {
		t.Errorf("remove ESI got ids = %v, want [5 6]", removeGot)
	}
}
