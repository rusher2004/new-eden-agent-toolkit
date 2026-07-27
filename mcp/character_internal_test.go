package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetCharacterHandler_ReturnsDetail(t *testing.T) {
	orig := getCharacterDetail
	t.Cleanup(func() { getCharacterDetail = orig })
	getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
		return GetCharacterOutput{CharacterID: characterID, Name: "Test Character", CorporationID: 98000001}, nil
	}

	handler := getCharacterHandler("v1")
	_, out, err := handler(context.Background(), nil, GetCharacterInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 998877 || out.Name != "Test Character" {
		t.Errorf("out = %+v, want CharacterID=998877 Name=\"Test Character\"", out)
	}
}

func TestGetCharacterAffiliationsHandler_ReturnsAffiliations(t *testing.T) {
	orig := getCharacterAffiliations
	t.Cleanup(func() { getCharacterAffiliations = orig })
	getCharacterAffiliations = func(ctx context.Context, userAgent string, characterIDs []int64) ([]CharacterAffiliation, error) {
		return []CharacterAffiliation{{CharacterID: 998877, CorporationID: 98000001}}, nil
	}

	handler := getCharacterAffiliationsHandler("v1")
	_, out, err := handler(context.Background(), nil, GetCharacterAffiliationsInput{CharacterIDs: []int64{998877}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Affiliations) != 1 || out.Affiliations[0].CharacterID != 998877 {
		t.Errorf("out = %+v, want one affiliation with CharacterID=998877", out)
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

	handler := listCharacterAgentsResearchHandler("v1", "client-id", store)
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

	handler := listCharacterAgentsResearchHandler("v1", "client-id", store)
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

	orig := listCharacterBlueprints
	t.Cleanup(func() { listCharacterBlueprints = orig })
	listCharacterBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Blueprint, int32, error) {
		return []Blueprint{{ItemID: 1, TypeID: 950}}, 2, nil
	}

	handler := listCharacterBlueprintsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 7, Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Blueprints) != 1 || out.Blueprints[0].TypeID != 950 || out.Page != 1 || out.TotalPages != 2 {
		t.Errorf("out = %+v, want one blueprint with TypeID=950 Page=1 TotalPages=2", out)
	}
}

func TestListCharacterBlueprintsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterBlueprints
	t.Cleanup(func() { listCharacterBlueprints = orig })
	wantErr := errors.New("upstream failure")
	listCharacterBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Blueprint, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCharacterBlueprintsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterBlueprintsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterBlueprints
	t.Cleanup(func() { listCharacterBlueprints = orig })
	listCharacterBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Blueprint, int32, error) {
		t.Fatal("listCharacterBlueprints should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCharacterBlueprintsHandler("v1", "client-id", store)
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

	orig := getCharacterFatigue
	t.Cleanup(func() { getCharacterFatigue = orig })
	getCharacterFatigue = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFatigueOutput, error) {
		return GetCharacterFatigueOutput{CharacterID: characterID}, nil
	}

	handler := getCharacterFatigueHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterFatigueInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 {
		t.Errorf("out = %+v, want CharacterID=7", out)
	}
}

func TestGetCharacterFatigueHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterFatigue
	t.Cleanup(func() { getCharacterFatigue = orig })
	getCharacterFatigue = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFatigueOutput, error) {
		t.Fatal("getCharacterFatigue should not be called for an unknown character")
		return GetCharacterFatigueOutput{}, nil
	}

	handler := getCharacterFatigueHandler("v1", "client-id", store)
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
