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

	orig := listCharacterContactLabels
	t.Cleanup(func() { listCharacterContactLabels = orig })
	listCharacterContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactLabel, error) {
		return []ContactLabel{{LabelID: 1, LabelName: "Friends"}}, nil
	}

	handler := listCharacterContactLabelsHandler("v1", "client-id", store)
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

	orig := listCharacterContactLabels
	t.Cleanup(func() { listCharacterContactLabels = orig })
	listCharacterContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactLabel, error) {
		t.Fatal("listCharacterContactLabels should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterContactLabelsHandler("v1", "client-id", store)
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

	orig := listCorporationContactLabels
	t.Cleanup(func() { listCorporationContactLabels = orig })
	listCorporationContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]ContactLabel, error) {
		return []ContactLabel{{LabelID: 2, LabelName: "Corp VIP"}}, nil
	}

	handler := listCorporationContactLabelsHandler("v1", "client-id", store)
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

	orig := listCorporationContactLabels
	t.Cleanup(func() { listCorporationContactLabels = orig })
	listCorporationContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]ContactLabel, error) {
		t.Fatal("listCorporationContactLabels should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationContactLabelsHandler("v1", "client-id", store)
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

	orig := listAllianceContactLabels
	t.Cleanup(func() { listAllianceContactLabels = orig })
	listAllianceContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]ContactLabel, error) {
		return []ContactLabel{{LabelID: 3, LabelName: "Alliance Blues"}}, nil
	}

	handler := listAllianceContactLabelsHandler("v1", "client-id", store)
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

	orig := listAllianceContactLabels
	t.Cleanup(func() { listAllianceContactLabels = orig })
	listAllianceContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]ContactLabel, error) {
		t.Fatal("listAllianceContactLabels should not be called for an unknown character")
		return nil, nil
	}

	handler := listAllianceContactLabelsHandler("v1", "client-id", store)
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

	orig := addCharacterContacts
	t.Cleanup(func() { addCharacterContacts = orig })
	addCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
		return []int64{998878}, nil
	}

	handler := addCharacterContactsHandler("v1", "client-id", store)
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

	orig := addCharacterContacts
	t.Cleanup(func() { addCharacterContacts = orig })
	addCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
		t.Fatal("addCharacterContacts should not be called for an unknown character")
		return nil, nil
	}

	handler := addCharacterContactsHandler("v1", "client-id", store)
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

	orig := updateCharacterContacts
	t.Cleanup(func() { updateCharacterContacts = orig })
	var gotStanding float64
	updateCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
		gotStanding = standing
		return nil
	}

	handler := updateCharacterContactsHandler("v1", "client-id", store)
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

	orig := updateCharacterContacts
	t.Cleanup(func() { updateCharacterContacts = orig })
	updateCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
		t.Fatal("updateCharacterContacts should not be called for an unknown character")
		return nil
	}

	handler := updateCharacterContactsHandler("v1", "client-id", store)
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

	orig := removeCharacterContacts
	t.Cleanup(func() { removeCharacterContacts = orig })
	removeCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64) error {
		return nil
	}

	handler := removeCharacterContactsHandler("v1", "client-id", store)
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

	orig := removeCharacterContacts
	t.Cleanup(func() { removeCharacterContacts = orig })
	removeCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64) error {
		t.Fatal("removeCharacterContacts should not be called for an unknown character")
		return nil
	}

	handler := removeCharacterContactsHandler("v1", "client-id", store)
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
