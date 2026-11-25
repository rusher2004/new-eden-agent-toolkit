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
	fake := fakeCharacterDetailESI{characterDetail: func(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error) {
		return &esiapi.CharactersDetail{Name: "Test Character", CorporationId: 98000001}, nil
	}}

	handler := getCharacterHandler("v1", publicNarrow[characterDetailESI](fake), nil)
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
	fake := fakeCharacterCorporationHistoryESI{characterCorporationHistory: func(ctx context.Context, characterID int64) ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, error) {
		return []esiapi.CharactersCharacterIdCorporationhistoryGetInner{{CorporationId: 98000001, RecordId: 1}}, nil
	}}

	handler := listCharacterCorporationHistoryHandler(publicNarrow[characterCorporationHistoryESI](fake))
	_, out, err := handler(context.Background(), nil, ListCharacterCorporationHistoryInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.History) != 1 || out.History[0].CorporationID != 98000001 {
		t.Errorf("out = %+v, want one history entry with CorporationID=98000001", out)
	}
}

func TestGetCharacterPortraitHandler_ReturnsPortrait(t *testing.T) {
	url := "https://images.evetech.net/a.png"
	fake := fakeCharacterPortraitESI{characterPortrait: func(ctx context.Context, characterID int64) (*esiapi.CharactersCharacterIdPortraitGet, error) {
		return &esiapi.CharactersCharacterIdPortraitGet{Px64x64: &url}, nil
	}}

	handler := getCharacterPortraitHandler(publicNarrow[characterPortraitESI](fake))
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

	fake := fakeCharacterAgentsResearchESI{characterAgentsResearch: func(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error) {
		return []esiapi.CharactersCharacterIdAgentsResearchGetInner{{AgentId: 1}}, nil
	}}

	handler := listCharacterAgentsResearchHandler("v1", openNarrow[characterAgentsResearchESI](fake), store, nil)
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

	handler := listCharacterAgentsResearchHandler("v1", noNarrow[characterAgentsResearchESI](t), store, nil)
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

	fake := fakeCharacterMedalsESI{characterMedals: func(ctx context.Context) ([]esiapi.CharactersCharacterIdMedalsGetInner, error) {
		return []esiapi.CharactersCharacterIdMedalsGetInner{{MedalId: 1, Title: "Title"}}, nil
	}}

	handler := listCharacterMedalsHandler(openNarrow[characterMedalsESI](fake), store)
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

	handler := listCharacterMedalsHandler(noNarrow[characterMedalsESI](t), store)
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

	fake := fakeCharacterNotificationsESI{characterNotifications: func(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsGetInner, error) {
		return []esiapi.CharactersCharacterIdNotificationsGetInner{{NotificationId: 1}}, nil
	}}

	handler := listCharacterNotificationsHandler(openNarrow[characterNotificationsESI](fake), store)
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

	handler := listCharacterNotificationsHandler(noNarrow[characterNotificationsESI](t), store)
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

	fake := fakeCharacterNotificationContactsESI{characterNotificationContacts: func(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, error) {
		return []esiapi.CharactersCharacterIdNotificationsContactsGetInner{{NotificationId: 1, StandingLevel: 5}}, nil
	}}

	handler := listCharacterNotificationContactsHandler(openNarrow[characterNotificationContactsESI](fake), store)
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

	handler := listCharacterNotificationContactsHandler(noNarrow[characterNotificationContactsESI](t), store)
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

	fake := fakeCharacterRolesESI{characterRoles: func(ctx context.Context) (*esiapi.CharactersCharacterIdRolesGet, error) {
		return &esiapi.CharactersCharacterIdRolesGet{Roles: []string{"Director"}}, nil
	}}

	handler := getCharacterRolesHandler(openNarrow[characterRolesESI](fake), store)
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

	handler := getCharacterRolesHandler(noNarrow[characterRolesESI](t), store)
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

	fake := fakeCharacterStandingsESI{characterStandings: func(ctx context.Context) ([]esiapi.CharactersCharacterIdStandingsGetInner, error) {
		return []esiapi.CharactersCharacterIdStandingsGetInner{{FromId: 500001, Standing: 5.5}}, nil
	}}

	handler := listCharacterStandingsHandler(openNarrow[characterStandingsESI](fake), store)
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

	handler := listCharacterStandingsHandler(noNarrow[characterStandingsESI](t), store)
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

	name := "CEO"
	fake := fakeCharacterTitlesESI{characterTitles: func(ctx context.Context) ([]esiapi.CharactersCharacterIdTitlesGetInner, error) {
		return []esiapi.CharactersCharacterIdTitlesGetInner{{Name: &name}}, nil
	}}

	handler := listCharacterTitlesHandler(openNarrow[characterTitlesESI](fake), store)
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

	handler := listCharacterTitlesHandler(noNarrow[characterTitlesESI](t), store)
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

	fake := fakeCharacterCSPACostESI{characterCSPACost: func(ctx context.Context, targetCharacterIDs []int64) (float64, error) {
		return 1000000.0, nil
	}}

	handler := getCharacterCSPACostHandler(openNarrow[characterCSPACostESI](fake), store)
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

	handler := getCharacterCSPACostHandler(noNarrow[characterCSPACostESI](t), store)
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

	fake := fakeCharacterAgentsResearchESI{characterAgentsResearch: func(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error) {
		return []esiapi.CharactersCharacterIdAgentsResearchGetInner{{AgentId: 1, SkillTypeId: 3403}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	var gotIDs []int64
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		gotIDs = ids
		return map[int64]string{3403: "Mining"}
	}

	handler := listCharacterAgentsResearchHandler("v1", openNarrow[characterAgentsResearchESI](fake), store, nil)
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
	factionID := int64(500001)
	fake := fakeCharacterDetailESI{characterDetail: func(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error) {
		return &esiapi.CharactersDetail{FactionId: &factionID, BloodlineId: 1, RaceId: 1}, nil
	}}

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

	handler := getCharacterHandler("v1", publicNarrow[characterDetailESI](fake), nil)
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
	fake := fakeCharacterDetailESI{characterDetail: func(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error) {
		return &esiapi.CharactersDetail{BloodlineId: 1, RaceId: 1}, nil
	}}

	handler := getCharacterHandler("v1", publicNarrow[characterDetailESI](fake), nil)
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

// fakeCharacterDetailESI fakes characterDetailESI; a nil characterDetail fails as an unfaked call.
type fakeCharacterDetailESI struct {
	characterDetail func(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error)
}

func (f fakeCharacterDetailESI) CharacterDetail(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error) {
	if f.characterDetail == nil {
		return nil, errNotFaked
	}
	return f.characterDetail(ctx, characterID)
}

// fakeCharacterCorporationHistoryESI fakes characterCorporationHistoryESI; a nil characterCorporationHistory fails as an unfaked call.
type fakeCharacterCorporationHistoryESI struct {
	characterCorporationHistory func(ctx context.Context, characterID int64) ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, error)
}

func (f fakeCharacterCorporationHistoryESI) CharacterCorporationHistory(ctx context.Context, characterID int64) ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, error) {
	if f.characterCorporationHistory == nil {
		return nil, errNotFaked
	}
	return f.characterCorporationHistory(ctx, characterID)
}

// fakeCharacterPortraitESI fakes characterPortraitESI; a nil characterPortrait fails as an unfaked call.
type fakeCharacterPortraitESI struct {
	characterPortrait func(ctx context.Context, characterID int64) (*esiapi.CharactersCharacterIdPortraitGet, error)
}

func (f fakeCharacterPortraitESI) CharacterPortrait(ctx context.Context, characterID int64) (*esiapi.CharactersCharacterIdPortraitGet, error) {
	if f.characterPortrait == nil {
		return nil, errNotFaked
	}
	return f.characterPortrait(ctx, characterID)
}

// fakeCharacterAgentsResearchESI fakes characterAgentsResearchESI; a nil characterAgentsResearch fails as an unfaked call.
type fakeCharacterAgentsResearchESI struct {
	characterAgentsResearch func(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error)
}

func (f fakeCharacterAgentsResearchESI) CharacterAgentsResearch(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error) {
	if f.characterAgentsResearch == nil {
		return nil, errNotFaked
	}
	return f.characterAgentsResearch(ctx)
}

// fakeCharacterMedalsESI fakes characterMedalsESI; a nil characterMedals fails as an unfaked call.
type fakeCharacterMedalsESI struct {
	characterMedals func(ctx context.Context) ([]esiapi.CharactersCharacterIdMedalsGetInner, error)
}

func (f fakeCharacterMedalsESI) CharacterMedals(ctx context.Context) ([]esiapi.CharactersCharacterIdMedalsGetInner, error) {
	if f.characterMedals == nil {
		return nil, errNotFaked
	}
	return f.characterMedals(ctx)
}

// fakeCharacterNotificationsESI fakes characterNotificationsESI; a nil characterNotifications fails as an unfaked call.
type fakeCharacterNotificationsESI struct {
	characterNotifications func(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsGetInner, error)
}

func (f fakeCharacterNotificationsESI) CharacterNotifications(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsGetInner, error) {
	if f.characterNotifications == nil {
		return nil, errNotFaked
	}
	return f.characterNotifications(ctx)
}

// fakeCharacterNotificationContactsESI fakes characterNotificationContactsESI; a nil characterNotificationContacts fails as an unfaked call.
type fakeCharacterNotificationContactsESI struct {
	characterNotificationContacts func(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, error)
}

func (f fakeCharacterNotificationContactsESI) CharacterNotificationContacts(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, error) {
	if f.characterNotificationContacts == nil {
		return nil, errNotFaked
	}
	return f.characterNotificationContacts(ctx)
}

// fakeCharacterRolesESI fakes characterRolesESI; a nil characterRoles fails as an unfaked call.
type fakeCharacterRolesESI struct {
	characterRoles func(ctx context.Context) (*esiapi.CharactersCharacterIdRolesGet, error)
}

func (f fakeCharacterRolesESI) CharacterRoles(ctx context.Context) (*esiapi.CharactersCharacterIdRolesGet, error) {
	if f.characterRoles == nil {
		return nil, errNotFaked
	}
	return f.characterRoles(ctx)
}

// fakeCharacterStandingsESI fakes characterStandingsESI; a nil characterStandings fails as an unfaked call.
type fakeCharacterStandingsESI struct {
	characterStandings func(ctx context.Context) ([]esiapi.CharactersCharacterIdStandingsGetInner, error)
}

func (f fakeCharacterStandingsESI) CharacterStandings(ctx context.Context) ([]esiapi.CharactersCharacterIdStandingsGetInner, error) {
	if f.characterStandings == nil {
		return nil, errNotFaked
	}
	return f.characterStandings(ctx)
}

// fakeCharacterTitlesESI fakes characterTitlesESI; a nil characterTitles fails as an unfaked call.
type fakeCharacterTitlesESI struct {
	characterTitles func(ctx context.Context) ([]esiapi.CharactersCharacterIdTitlesGetInner, error)
}

func (f fakeCharacterTitlesESI) CharacterTitles(ctx context.Context) ([]esiapi.CharactersCharacterIdTitlesGetInner, error) {
	if f.characterTitles == nil {
		return nil, errNotFaked
	}
	return f.characterTitles(ctx)
}

// fakeCharacterCSPACostESI fakes characterCSPACostESI; a nil characterCSPACost fails as an unfaked call.
type fakeCharacterCSPACostESI struct {
	characterCSPACost func(ctx context.Context, targetCharacterIDs []int64) (float64, error)
}

func (f fakeCharacterCSPACostESI) CharacterCSPACost(ctx context.Context, targetCharacterIDs []int64) (float64, error) {
	if f.characterCSPACost == nil {
		return 0, errNotFaked
	}
	return f.characterCSPACost(ctx, targetCharacterIDs)
}

// openOrESIFailure calls a handler once with an opener that fails and once with
// a client whose ESI call is unfaked, returning both errors.
type openOrESIFailure func(failOpen bool) error

// Each private character handler reports a client that can't be opened, and an
// ESI call that fails, wrapping the cause.
func TestCharacterHandlers_PropagateOpenAndESIErrors(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	ctx := context.Background()
	openErr := errors.New("no refresh token")
	for name, call := range map[string]openOrESIFailure{
		"list_character_agents_research": func(failOpen bool) error {
			open := openNarrow[characterAgentsResearchESI](fakeCharacterAgentsResearchESI{})
			if failOpen {
				open = failNarrow[characterAgentsResearchESI](openErr)
			}
			_, _, err := listCharacterAgentsResearchHandler("v1", open, store, nil)(ctx, nil, ListCharacterAgentsResearchInput{CharacterID: 7})
			return err
		},
		"list_character_medals": func(failOpen bool) error {
			open := openNarrow[characterMedalsESI](fakeCharacterMedalsESI{})
			if failOpen {
				open = failNarrow[characterMedalsESI](openErr)
			}
			_, _, err := listCharacterMedalsHandler(open, store)(ctx, nil, ListCharacterMedalsInput{CharacterID: 7})
			return err
		},
		"list_character_notifications": func(failOpen bool) error {
			open := openNarrow[characterNotificationsESI](fakeCharacterNotificationsESI{})
			if failOpen {
				open = failNarrow[characterNotificationsESI](openErr)
			}
			_, _, err := listCharacterNotificationsHandler(open, store)(ctx, nil, ListCharacterNotificationsInput{CharacterID: 7})
			return err
		},
		"list_character_notification_contacts": func(failOpen bool) error {
			open := openNarrow[characterNotificationContactsESI](fakeCharacterNotificationContactsESI{})
			if failOpen {
				open = failNarrow[characterNotificationContactsESI](openErr)
			}
			_, _, err := listCharacterNotificationContactsHandler(open, store)(ctx, nil, ListCharacterNotificationContactsInput{CharacterID: 7})
			return err
		},
		"get_character_roles": func(failOpen bool) error {
			open := openNarrow[characterRolesESI](fakeCharacterRolesESI{})
			if failOpen {
				open = failNarrow[characterRolesESI](openErr)
			}
			_, _, err := getCharacterRolesHandler(open, store)(ctx, nil, GetCharacterRolesInput{CharacterID: 7})
			return err
		},
		"list_character_standings": func(failOpen bool) error {
			open := openNarrow[characterStandingsESI](fakeCharacterStandingsESI{})
			if failOpen {
				open = failNarrow[characterStandingsESI](openErr)
			}
			_, _, err := listCharacterStandingsHandler(open, store)(ctx, nil, ListCharacterStandingsInput{CharacterID: 7})
			return err
		},
		"list_character_titles": func(failOpen bool) error {
			open := openNarrow[characterTitlesESI](fakeCharacterTitlesESI{})
			if failOpen {
				open = failNarrow[characterTitlesESI](openErr)
			}
			_, _, err := listCharacterTitlesHandler(open, store)(ctx, nil, ListCharacterTitlesInput{CharacterID: 7})
			return err
		},
		"get_character_cspa_cost": func(failOpen bool) error {
			open := openNarrow[characterCSPACostESI](fakeCharacterCSPACostESI{})
			if failOpen {
				open = failNarrow[characterCSPACostESI](openErr)
			}
			_, _, err := getCharacterCSPACostHandler(open, store)(ctx, nil, GetCharacterCSPACostInput{CharacterID: 7})
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

// Each public character handler reports an ESI call that fails, wrapping the cause.
func TestCharacterPublicHandlers_PropagateESIErrors(t *testing.T) {
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"get_character": func() error {
			_, _, err := getCharacterHandler("v1", publicNarrow[characterDetailESI](fakeCharacterDetailESI{}), nil)(ctx, nil, GetCharacterInput{CharacterID: 998877})
			return err
		},
		"list_character_corporation_history": func() error {
			_, _, err := listCharacterCorporationHistoryHandler(publicNarrow[characterCorporationHistoryESI](fakeCharacterCorporationHistoryESI{}))(ctx, nil, ListCharacterCorporationHistoryInput{CharacterID: 998877})
			return err
		},
		"get_character_portrait": func() error {
			_, _, err := getCharacterPortraitHandler(publicNarrow[characterPortraitESI](fakeCharacterPortraitESI{}))(ctx, nil, GetCharacterPortraitInput{CharacterID: 998877})
			return err
		},
	} {
		if err := call(); !errors.Is(err, errNotFaked) {
			t.Errorf("%s: err = %v, want wrapping errNotFaked", name, err)
		}
	}
}

// get_character_cspa_cost passes the target characters to ESI.
func TestGetCharacterCSPACostHandler_PassesTargets(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	var got []int64
	fake := fakeCharacterCSPACostESI{characterCSPACost: func(_ context.Context, targets []int64) (float64, error) {
		got = targets
		return 1, nil
	}}
	_, _, err := getCharacterCSPACostHandler(openNarrow[characterCSPACostESI](fake), store)(context.Background(), nil, GetCharacterCSPACostInput{CharacterID: 7, TargetCharacterIDs: []int64{1, 2}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("ESI got targets = %v, want [1 2]", got)
	}
}
