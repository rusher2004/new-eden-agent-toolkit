package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// storeWithCharacter returns a config store holding one known character, ID 7.
func storeWithCharacter(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

func TestListMilitaryCampaignsHandler_MapsFields(t *testing.T) {
	orig := listMilitaryCampaigns
	t.Cleanup(func() { listMilitaryCampaigns = orig })
	started := time.Date(2026, 6, 9, 11, 0, 0, 0, time.UTC)
	listMilitaryCampaigns = func(ctx context.Context, userAgent string) ([]esiapi.MilitaryCampaignsDetailCampaign, error) {
		return []esiapi.MilitaryCampaignsDetailCampaign{{Id: "camp-1", State: "Active", Progress: 12, Started: &started}}, nil
	}

	_, out, err := listMilitaryCampaignsHandler("v1")(context.Background(), nil, ListMilitaryCampaignsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Campaigns) != 1 {
		t.Fatalf("out = %+v, want one campaign", out)
	}
	c := out.Campaigns[0]
	if c.CampaignID != "camp-1" || c.State != "Active" || c.Progress != 12 || c.Started == nil || !c.Started.Equal(started) || c.Finished != nil {
		t.Errorf("campaign = %+v, want camp-1 Active progress 12 started %v, no finish", c, started)
	}
}

func TestListMilitaryCampaignsHandler_PropagatesError(t *testing.T) {
	orig := listMilitaryCampaigns
	t.Cleanup(func() { listMilitaryCampaigns = orig })
	listMilitaryCampaigns = func(ctx context.Context, userAgent string) ([]esiapi.MilitaryCampaignsDetailCampaign, error) {
		return nil, errors.New("boom")
	}
	if _, _, err := listMilitaryCampaignsHandler("v1")(context.Background(), nil, ListMilitaryCampaignsInput{}); err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}

func TestGetMilitaryCampaignHandler_MapsFields(t *testing.T) {
	orig := getMilitaryCampaign
	t.Cleanup(func() { getMilitaryCampaign = orig })
	var gotID string
	getMilitaryCampaign = func(ctx context.Context, userAgent, campaignID string) (*esiapi.MilitaryCampaignsDetail, error) {
		gotID = campaignID
		return &esiapi.MilitaryCampaignsDetail{Id: campaignID, State: "Completed", Progress: 30}, nil
	}

	_, out, err := getMilitaryCampaignHandler("v1")(context.Background(), nil, GetMilitaryCampaignInput{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotID != "camp-1" || out.CampaignID != "camp-1" || out.State != "Completed" || out.Progress != 30 {
		t.Errorf("gotID = %q, out = %+v, want camp-1 Completed progress 30", gotID, out)
	}
}

func TestListMilitaryCampaignObjectivesHandler_MapsFieldsAndCursor(t *testing.T) {
	orig := listMilitaryCampaignObjectives
	t.Cleanup(func() { listMilitaryCampaignObjectives = orig })
	var gotAfter, gotBefore string
	var gotLimit int64
	lastModified := time.Unix(1000, 0)
	next := "next-cursor"
	listMilitaryCampaignObjectives = func(ctx context.Context, userAgent, campaignID, after, before string, limit int64) ([]esiapi.MilitaryCampaignsObjectivesDetailObjective, *esiapi.Cursor, error) {
		gotAfter, gotBefore, gotLimit = after, before, limit
		return []esiapi.MilitaryCampaignsObjectivesDetailObjective{{
			Id: "obj-1", State: "Active", Progress: 48700, LastModified: lastModified,
			Participants: esiapi.MilitaryCampaignsObjectivesDetailParticipants{Total: 667, Committed: 585, Contributors: 59},
		}}, &esiapi.Cursor{After: &next}, nil
	}

	_, out, err := listMilitaryCampaignObjectivesHandler("v1")(context.Background(), nil, ListMilitaryCampaignObjectivesInput{CampaignID: "camp-1", After: "a", Limit: 20})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotAfter != "a" || gotBefore != "" || gotLimit != 20 {
		t.Errorf("forwarded after=%q before=%q limit=%d, want a/\"\"/20", gotAfter, gotBefore, gotLimit)
	}
	if out.CampaignID != "camp-1" || len(out.Objectives) != 1 {
		t.Fatalf("out = %+v, want camp-1 with one objective", out)
	}
	o := out.Objectives[0]
	if o.ObjectiveID != "obj-1" || o.Progress != 48700 || o.Participants != (CampaignObjectiveParticipants{Total: 667, Committed: 585, Contributors: 59}) || !o.LastModified.Equal(lastModified) {
		t.Errorf("objective = %+v, want obj-1 progress 48700 participants {667 585 59}", o)
	}
	if out.NextAfter == nil || *out.NextAfter != "next-cursor" || out.NextBefore != nil {
		t.Errorf("NextAfter/NextBefore = %v/%v, want next-cursor/nil", out.NextAfter, out.NextBefore)
	}
}

func TestGetMilitaryCampaignObjectiveHandler_MapsFields(t *testing.T) {
	orig := getMilitaryCampaignObjective
	t.Cleanup(func() { getMilitaryCampaignObjective = orig })
	getMilitaryCampaignObjective = func(ctx context.Context, userAgent, campaignID, objectiveID string) (*esiapi.MilitaryCampaignsObjectivesDetail, error) {
		return &esiapi.MilitaryCampaignsObjectivesDetail{
			Id: objectiveID, State: "Completed", Progress: 100,
			Participants: esiapi.MilitaryCampaignsObjectivesDetailParticipants{Total: 10, Contributors: 8},
		}, nil
	}

	_, out, err := getMilitaryCampaignObjectiveHandler("v1")(context.Background(), nil, GetMilitaryCampaignObjectiveInput{CampaignID: "camp-1", ObjectiveID: "obj-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CampaignID != "camp-1" || out.ObjectiveID != "obj-1" || out.State != "Completed" || out.Participants.Contributors != 8 {
		t.Errorf("out = %+v, want camp-1/obj-1 Completed with 8 contributors", out)
	}
}

func TestListCharacterMilitaryCampaignObjectivesHandler_MapsFields(t *testing.T) {
	store := storeWithCharacter(t)
	orig := listCharacterMilitaryCampaignObjectives
	t.Cleanup(func() { listCharacterMilitaryCampaignObjectives = orig })
	listCharacterMilitaryCampaignObjectives = func(ctx context.Context, clientID string, characterID int32, userAgent, after, before string, limit int64) ([]esiapi.CharactersMilitaryCampaignsObjectivesParticipationCharacterobjective, *esiapi.Cursor, error) {
		return []esiapi.CharactersMilitaryCampaignsObjectivesParticipationCharacterobjective{{Id: "obj-1", CampaignId: "camp-1", Contributed: 42, IsCommitted: true}}, nil, nil
	}

	_, out, err := listCharacterMilitaryCampaignObjectivesHandler("v1", "client-id", store)(context.Background(), nil, ListCharacterMilitaryCampaignObjectivesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Objectives) != 1 {
		t.Fatalf("out = %+v, want character 7 with one objective", out)
	}
	if o := out.Objectives[0]; o.ObjectiveID != "obj-1" || o.CampaignID != "camp-1" || o.Contributed != 42 || !o.IsCommitted {
		t.Errorf("objective = %+v, want obj-1/camp-1 committed with 42 contributed", o)
	}
	if out.NextAfter != nil || out.NextBefore != nil {
		t.Errorf("cursor = %v/%v, want both nil when ESI omits the cursor", out.NextAfter, out.NextBefore)
	}
}

func TestListCharacterMilitaryCampaignObjectivesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	orig := listCharacterMilitaryCampaignObjectives
	t.Cleanup(func() { listCharacterMilitaryCampaignObjectives = orig })
	listCharacterMilitaryCampaignObjectives = func(ctx context.Context, clientID string, characterID int32, userAgent, after, before string, limit int64) ([]esiapi.CharactersMilitaryCampaignsObjectivesParticipationCharacterobjective, *esiapi.Cursor, error) {
		t.Fatal("listCharacterMilitaryCampaignObjectives should not be called for an unknown character")
		return nil, nil, nil
	}

	_, _, err := listCharacterMilitaryCampaignObjectivesHandler("v1", "client-id", store)(context.Background(), nil, ListCharacterMilitaryCampaignObjectivesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterMilitaryCampaignObjectiveParticipationHandler_MapsFields(t *testing.T) {
	store := storeWithCharacter(t)
	orig := getCharacterMilitaryCampaignObjectiveParticipation
	t.Cleanup(func() { getCharacterMilitaryCampaignObjectiveParticipation = orig })
	getCharacterMilitaryCampaignObjectiveParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent, objectiveID string) (*esiapi.CharactersMilitaryCampaignsObjectivesParticipation, error) {
		return &esiapi.CharactersMilitaryCampaignsObjectivesParticipation{Id: objectiveID, CampaignId: "camp-1", Contributed: 5}, nil
	}

	_, out, err := getCharacterMilitaryCampaignObjectiveParticipationHandler("v1", "client-id", store)(context.Background(), nil, GetCharacterMilitaryCampaignObjectiveParticipationInput{CharacterID: 7, ObjectiveID: "obj-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.ObjectiveID != "obj-1" || out.CampaignID != "camp-1" || out.Contributed != 5 || out.IsCommitted {
		t.Errorf("out = %+v, want character 7, obj-1/camp-1, 5 contributed, not committed", out)
	}
}

func TestGetCharacterMilitaryCampaignObjectiveParticipationHandler_PropagatesError(t *testing.T) {
	store := storeWithCharacter(t)
	orig := getCharacterMilitaryCampaignObjectiveParticipation
	t.Cleanup(func() { getCharacterMilitaryCampaignObjectiveParticipation = orig })
	getCharacterMilitaryCampaignObjectiveParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent, objectiveID string) (*esiapi.CharactersMilitaryCampaignsObjectivesParticipation, error) {
		return nil, errors.New("boom")
	}
	if _, _, err := getCharacterMilitaryCampaignObjectiveParticipationHandler("v1", "client-id", store)(context.Background(), nil, GetCharacterMilitaryCampaignObjectiveParticipationInput{CharacterID: 7, ObjectiveID: "obj-1"}); err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
