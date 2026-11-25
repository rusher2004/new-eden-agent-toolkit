package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetCharacterHandler_ReturnsDetail(t *testing.T) {
	orig := getCharacterDetail
	t.Cleanup(func() { getCharacterDetail = orig })
	getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
		return GetCharacterOutput{CharacterID: characterID, Name: "Test Character", CorporationID: 98000001}, nil
	}

	handler := getCharacterHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCharacterInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 998877 || out.Name != "Test Character" {
		t.Errorf("out = %+v, want CharacterID=998877 Name=\"Test Character\"", out)
	}
}

func TestGetCharacterAffiliationsHandler_ReturnsAffiliations(t *testing.T) {
	var gotIDs []int64
	fake := fakeCharacterAffiliationsESI{characterAffiliations: func(_ context.Context, ids []int64) ([]esiapi.CharactersAffiliationPostInner, error) {
		gotIDs = ids
		return []esiapi.CharactersAffiliationPostInner{{CharacterId: 998877, CorporationId: 98000001}}, nil
	}}

	handler := getCharacterAffiliationsHandler("v1", publicNarrow[characterAffiliationsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetCharacterAffiliationsInput{CharacterIDs: []int64{998877}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 998877 {
		t.Errorf("ESI got ids = %v, want [998877]", gotIDs)
	}
	if len(out.Affiliations) != 1 || out.Affiliations[0].CharacterID != 998877 || out.Affiliations[0].CorporationID != 98000001 {
		t.Errorf("out = %+v, want one affiliation with CharacterID=998877 CorporationID=98000001", out)
	}
}

func TestGetCharacterAffiliationsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCharacterAffiliationsESI{characterAffiliations: func(context.Context, []int64) ([]esiapi.CharactersAffiliationPostInner, error) {
		return nil, wantErr
	}}

	handler := getCharacterAffiliationsHandler("v1", publicNarrow[characterAffiliationsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, GetCharacterAffiliationsInput{CharacterIDs: []int64{998877}})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterCorporationHistoryHandler_ReturnsHistory(t *testing.T) {
	orig := listCharacterCorporationHistory
	t.Cleanup(func() { listCharacterCorporationHistory = orig })
	listCharacterCorporationHistory = func(ctx context.Context, userAgent string, characterID int64) ([]CorporationHistoryEntry, error) {
		return []CorporationHistoryEntry{{CorporationID: 98000001, RecordID: 1}}, nil
	}

	handler := listCharacterCorporationHistoryHandler("v1")
	_, out, err := handler(context.Background(), nil, ListCharacterCorporationHistoryInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.History) != 1 || out.History[0].CorporationID != 98000001 {
		t.Errorf("out = %+v, want one history entry with CorporationID=98000001", out)
	}
}

func TestGetCharacterPortraitHandler_ReturnsPortrait(t *testing.T) {
	orig := getCharacterPortrait
	t.Cleanup(func() { getCharacterPortrait = orig })
	url := "https://images.evetech.net/a.png"
	getCharacterPortrait = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterPortraitOutput, error) {
		return GetCharacterPortraitOutput{CharacterID: characterID, Px64x64: &url}, nil
	}

	handler := getCharacterPortraitHandler("v1")
	_, out, err := handler(context.Background(), nil, GetCharacterPortraitInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Px64x64 == nil || *out.Px64x64 != url {
		t.Errorf("out = %+v, want Px64x64=*%q", out, url)
	}
}

func TestListCharacterAgentsResearchHandler_ReturnsAgents(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterAgentsResearch
	t.Cleanup(func() { listCharacterAgentsResearch = orig })
	listCharacterAgentsResearch = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AgentResearch, error) {
		return []AgentResearch{{AgentID: 1}}, nil
	}

	handler := listCharacterAgentsResearchHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterAgentsResearchInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Agents) != 1 || out.Agents[0].AgentID != 1 {
		t.Errorf("out = %+v, want one agent with AgentID=1", out)
	}
}

func TestListCharacterAgentsResearchHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterAgentsResearch
	t.Cleanup(func() { listCharacterAgentsResearch = orig })
	listCharacterAgentsResearch = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AgentResearch, error) {
		t.Fatal("listCharacterAgentsResearch should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterAgentsResearchHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterAgentsResearchInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterBlueprintsHandler_ReturnsBlueprints(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	var gotPage int32
	fake := fakeCharacterBlueprintsESI{characterBlueprints: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, int32, error) {
		gotPage = page
		return []esiapi.CharactersCharacterIdBlueprintsGetInner{{ItemId: 1, TypeId: 950, LocationId: 60003760, LocationFlag: "Hangar", Quantity: -2, Runs: 10, MaterialEfficiency: 10, TimeEfficiency: 20}}, 2, nil
	}}

	handler := listCharacterBlueprintsHandler("v1", openNarrow[characterBlueprintsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 7, Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotPage != 1 {
		t.Errorf("ESI got page %d, want 1", gotPage)
	}
	if len(out.Blueprints) != 1 || out.Blueprints[0].TypeID != 950 || out.Page != 1 || out.TotalPages != 2 {
		t.Fatalf("out = %+v, want one blueprint with TypeID=950 Page=1 TotalPages=2", out)
	}
	if b := out.Blueprints[0]; b.ItemID != 1 || b.LocationID != 60003760 || b.LocationFlag != "Hangar" || b.Quantity != -2 || b.Runs != 10 || b.MaterialEfficiency != 10 || b.TimeEfficiency != 20 {
		t.Errorf("Blueprints[0] = %+v, want every field mapped", b)
	}
}

func TestListCharacterBlueprintsHandler_PropagatesOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("no token")
	handler := listCharacterBlueprintsHandler("v1", failNarrow[characterBlueprintsESI](wantErr), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterBlueprintsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterBlueprintsESI{characterBlueprints: func(context.Context, int32) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCharacterBlueprintsHandler("v1", openNarrow[characterBlueprintsESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterBlueprintsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterBlueprintsHandler("v1", noNarrow[characterBlueprintsESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterFatigueHandler_ReturnsFatigue(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	expires := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	fake := fakeCharacterFatigueESI{characterFatigue: func(context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error) {
		return &esiapi.CharactersCharacterIdFatigueGet{JumpFatigueExpireDate: &expires}, nil
	}}

	handler := getCharacterFatigueHandler(openNarrow[characterFatigueESI](fake), store)
	_, out, err := handler(context.Background(), nil, GetCharacterFatigueInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.JumpFatigueExpireDate == nil || !out.JumpFatigueExpireDate.Equal(expires) {
		t.Errorf("out = %+v, want CharacterID=7 with the fatigue expiry", out)
	}
}

func TestGetCharacterFatigueHandler_PropagatesErrors(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	wantErr := errors.New("upstream failure")
	failing := fakeCharacterFatigueESI{characterFatigue: func(context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error) { return nil, wantErr }}

	for name, open := range map[string]func(context.Context, int32) (characterFatigueESI, error){
		"ESI call": openNarrow[characterFatigueESI](failing),
		"open":     failNarrow[characterFatigueESI](wantErr),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := getCharacterFatigueHandler(open, store)(context.Background(), nil, GetCharacterFatigueInput{CharacterID: 7})
			if !errors.Is(err, wantErr) {
				t.Errorf("err = %v, want wrapping %v", err, wantErr)
			}
		})
	}
}

func TestGetCharacterFatigueHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := getCharacterFatigueHandler(noNarrow[characterFatigueESI](t), store)
	_, _, err := handler(context.Background(), nil, GetCharacterFatigueInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterMedalsHandler_ReturnsMedals(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMedals
	t.Cleanup(func() { listCharacterMedals = orig })
	listCharacterMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Medal, error) {
		return []Medal{{MedalID: 1, Title: "Title"}}, nil
	}

	handler := listCharacterMedalsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterMedalsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Medals) != 1 || out.Medals[0].Title != "Title" {
		t.Errorf("out = %+v, want one medal with Title=\"Title\"", out)
	}
}

func TestListCharacterMedalsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterMedals
	t.Cleanup(func() { listCharacterMedals = orig })
	listCharacterMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Medal, error) {
		t.Fatal("listCharacterMedals should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMedalsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterMedalsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterNotificationsHandler_ReturnsNotifications(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterNotifications
	t.Cleanup(func() { listCharacterNotifications = orig })
	listCharacterNotifications = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Notification, error) {
		return []Notification{{NotificationID: 1}}, nil
	}

	handler := listCharacterNotificationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterNotificationsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Notifications) != 1 || out.Notifications[0].NotificationID != 1 {
		t.Errorf("out = %+v, want one notification with NotificationID=1", out)
	}
}

func TestListCharacterNotificationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterNotifications
	t.Cleanup(func() { listCharacterNotifications = orig })
	listCharacterNotifications = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Notification, error) {
		t.Fatal("listCharacterNotifications should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterNotificationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterNotificationsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterNotificationContactsHandler_ReturnsNotifications(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterNotificationContacts
	t.Cleanup(func() { listCharacterNotificationContacts = orig })
	listCharacterNotificationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactNotification, error) {
		return []ContactNotification{{NotificationID: 1, StandingLevel: 5}}, nil
	}

	handler := listCharacterNotificationContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterNotificationContactsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Notifications) != 1 || out.Notifications[0].StandingLevel != 5 {
		t.Errorf("out = %+v, want one notification with StandingLevel=5", out)
	}
}

func TestListCharacterNotificationContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterNotificationContacts
	t.Cleanup(func() { listCharacterNotificationContacts = orig })
	listCharacterNotificationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactNotification, error) {
		t.Fatal("listCharacterNotificationContacts should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterNotificationContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterNotificationContactsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterRolesHandler_ReturnsRoles(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterRoles
	t.Cleanup(func() { getCharacterRoles = orig })
	getCharacterRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterRolesOutput, error) {
		return GetCharacterRolesOutput{CharacterID: characterID, Roles: []string{"Director"}}, nil
	}

	handler := getCharacterRolesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterRolesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Roles) != 1 || out.Roles[0] != "Director" {
		t.Errorf("out = %+v, want Roles=[\"Director\"]", out)
	}
}

func TestGetCharacterRolesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterRoles
	t.Cleanup(func() { getCharacterRoles = orig })
	getCharacterRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterRolesOutput, error) {
		t.Fatal("getCharacterRoles should not be called for an unknown character")
		return GetCharacterRolesOutput{}, nil
	}

	handler := getCharacterRolesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterRolesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterStandingsHandler_ReturnsStandings(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterStandings
	t.Cleanup(func() { listCharacterStandings = orig })
	listCharacterStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Standing, error) {
		return []Standing{{FromID: 500001, Standing: 5.5}}, nil
	}

	handler := listCharacterStandingsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterStandingsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Standings) != 1 || out.Standings[0].Standing != 5.5 {
		t.Errorf("out = %+v, want one standing with Standing=5.5", out)
	}
}

func TestListCharacterStandingsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterStandings
	t.Cleanup(func() { listCharacterStandings = orig })
	listCharacterStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Standing, error) {
		t.Fatal("listCharacterStandings should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterStandingsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterStandingsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterTitlesHandler_ReturnsTitles(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterTitles
	t.Cleanup(func() { listCharacterTitles = orig })
	name := "CEO"
	listCharacterTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Title, error) {
		return []Title{{Name: &name}}, nil
	}

	handler := listCharacterTitlesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterTitlesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Titles) != 1 || out.Titles[0].Name == nil || *out.Titles[0].Name != "CEO" {
		t.Errorf("out = %+v, want one title with Name=*\"CEO\"", out)
	}
}

func TestListCharacterTitlesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterTitles
	t.Cleanup(func() { listCharacterTitles = orig })
	listCharacterTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Title, error) {
		t.Fatal("listCharacterTitles should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterTitlesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterTitlesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterCSPACostHandler_ReturnsCost(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterCSPACost
	t.Cleanup(func() { getCharacterCSPACost = orig })
	getCharacterCSPACost = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetCharacterIDs []int64) (float64, error) {
		return 1000000.0, nil
	}

	handler := getCharacterCSPACostHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterCSPACostInput{CharacterID: 7, TargetCharacterIDs: []int64{998878}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Cost != 1000000.0 {
		t.Errorf("out = %+v, want Cost=1000000.0", out)
	}
}

func TestGetCharacterCSPACostHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterCSPACost
	t.Cleanup(func() { getCharacterCSPACost = orig })
	getCharacterCSPACost = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetCharacterIDs []int64) (float64, error) {
		t.Fatal("getCharacterCSPACost should not be called for an unknown character")
		return 0, nil
	}

	handler := getCharacterCSPACostHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterCSPACostInput{CharacterID: 999, TargetCharacterIDs: []int64{998878}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterAgentsResearchHandler_ResolvesSkillTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterAgentsResearch
	t.Cleanup(func() { listCharacterAgentsResearch = orig })
	listCharacterAgentsResearch = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AgentResearch, error) {
		return []AgentResearch{{AgentID: 1, SkillTypeID: 3403}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{3403: "Mining"}
	}

	handler := listCharacterAgentsResearchHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterAgentsResearchInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 3403 {
		t.Errorf("resolveTypeNames got ids = %v, want [3403]", gotIDs)
	}
	if out.Agents[0].SkillTypeName != "Mining" {
		t.Errorf("SkillTypeName = %q, want %q", out.Agents[0].SkillTypeName, "Mining")
	}
}

func TestListCharacterBlueprintsHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCharacterBlueprintsESI{characterBlueprints: func(context.Context, int32) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdBlueprintsGetInner{{ItemId: 1, TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{587: "Rifter Blueprint"}
	}

	handler := listCharacterBlueprintsHandler("v1", openNarrow[characterBlueprintsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 587 {
		t.Errorf("resolveTypeNames got ids = %v, want [587]", gotIDs)
	}
	if out.Blueprints[0].TypeName != "Rifter Blueprint" {
		t.Errorf("TypeName = %q, want %q", out.Blueprints[0].TypeName, "Rifter Blueprint")
	}
}

func TestGetCharacterHandler_ResolvesFactionRaceBloodlineNames(t *testing.T) {
	orig := getCharacterDetail
	t.Cleanup(func() { getCharacterDetail = orig })
	factionID := int64(500001)
	getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
		return GetCharacterOutput{CharacterID: characterID, FactionID: &factionID, BloodlineID: 1, RaceID: 1}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}
	origRace := resolveRaceNames
	t.Cleanup(func() { resolveRaceNames = origRace })
	resolveRaceNames = func(ctx context.Context, sdeStore raceNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1: "Caldari"}
	}
	origBloodline := resolveBloodlineNames
	t.Cleanup(func() { resolveBloodlineNames = origBloodline })
	resolveBloodlineNames = func(ctx context.Context, sdeStore bloodlineNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1: "Deteis"}
	}

	handler := getCharacterHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCharacterInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
	if out.RaceName != "Caldari" {
		t.Errorf("RaceName = %q, want %q", out.RaceName, "Caldari")
	}
	if out.BloodlineName != "Deteis" {
		t.Errorf("BloodlineName = %q, want %q", out.BloodlineName, "Deteis")
	}
}

func TestGetCharacterHandler_FactionNameEmptyWhenFactionIDNil(t *testing.T) {
	orig := getCharacterDetail
	t.Cleanup(func() { getCharacterDetail = orig })
	getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
		return GetCharacterOutput{CharacterID: characterID, BloodlineID: 1, RaceID: 1}, nil
	}

	handler := getCharacterHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCharacterInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "" {
		t.Errorf("FactionName = %q, want empty when FactionID is nil", out.FactionName)
	}
}

func TestGetCharacterAffiliationsHandler_ResolvesFactionName(t *testing.T) {
	factionID := int64(500001)
	fake := fakeCharacterAffiliationsESI{characterAffiliations: func(context.Context, []int64) ([]esiapi.CharactersAffiliationPostInner, error) {
		return []esiapi.CharactersAffiliationPostInner{{CharacterId: 998877, CorporationId: 98000001, FactionId: &factionID}}, nil
	}}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getCharacterAffiliationsHandler("v1", publicNarrow[characterAffiliationsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetCharacterAffiliationsInput{CharacterIDs: []int64{998877}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Affiliations) != 1 || out.Affiliations[0].FactionName != "Caldari State" {
		t.Errorf("Affiliations = %+v, want one entry FactionName=%q", out.Affiliations, "Caldari State")
	}
}

// fakeCharacterAffiliationsESI fakes characterAffiliationsESI; a nil characterAffiliations fails as an unfaked call.
type fakeCharacterAffiliationsESI struct {
	characterAffiliations func(ctx context.Context, characterIDs []int64) ([]esiapi.CharactersAffiliationPostInner, error)
}

func (f fakeCharacterAffiliationsESI) CharacterAffiliations(ctx context.Context, characterIDs []int64) ([]esiapi.CharactersAffiliationPostInner, error) {
	if f.characterAffiliations == nil {
		return nil, errNotFaked
	}
	return f.characterAffiliations(ctx, characterIDs)
}

// fakeCharacterBlueprintsESI fakes characterBlueprintsESI; a nil characterBlueprints fails as an unfaked call.
type fakeCharacterBlueprintsESI struct {
	characterBlueprints func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, int32, error)
}

func (f fakeCharacterBlueprintsESI) CharacterBlueprints(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, int32, error) {
	if f.characterBlueprints == nil {
		return nil, 0, errNotFaked
	}
	return f.characterBlueprints(ctx, page)
}

// fakeCharacterFatigueESI fakes characterFatigueESI; a nil characterFatigue fails as an unfaked call.
type fakeCharacterFatigueESI struct {
	characterFatigue func(ctx context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error)
}

func (f fakeCharacterFatigueESI) CharacterFatigue(ctx context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error) {
	if f.characterFatigue == nil {
		return nil, errNotFaked
	}
	return f.characterFatigue(ctx)
}
