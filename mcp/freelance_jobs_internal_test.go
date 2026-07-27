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

func TestListCharacterFreelanceJobsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterFreelanceJobs
	t.Cleanup(func() { listCharacterFreelanceJobs = orig })
	lastModified := time.Unix(1000, 0)
	listCharacterFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.FreelanceJobsDetailFreelancejob, error) {
		return []esiapi.FreelanceJobsDetailFreelancejob{
			{
				Id: "job-1", Name: "Test Job", State: "Active", LastModified: lastModified,
				Progress: esiapi.FreelanceJobsDetailProgress{Current: 10, Desired: 100},
				Reward:   &esiapi.FreelanceJobsDetailReward{Initial: 1000000.0, Remaining: 500000.0},
			},
		}, nil
	}

	handler := listCharacterFreelanceJobsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterFreelanceJobsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Jobs) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one job", out)
	}
	j := out.Jobs[0]
	if j.JobID != "job-1" || j.Name != "Test Job" || j.State != "Active" || !j.LastModified.Equal(lastModified) ||
		j.Progress.Current != 10 || j.Progress.Desired != 100 ||
		j.Reward == nil || j.Reward.Initial != 1000000.0 || j.Reward.Remaining != 500000.0 {
		t.Errorf("job = %+v, want JobID=job-1 Name=\"Test Job\" State=Active LastModified=%v Progress={10 100} Reward={1000000 500000}", j, lastModified)
	}
}

func TestListCharacterFreelanceJobsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterFreelanceJobs
	t.Cleanup(func() { listCharacterFreelanceJobs = orig })
	listCharacterFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.FreelanceJobsDetailFreelancejob, error) {
		t.Fatal("listCharacterFreelanceJobs should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterFreelanceJobsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterFreelanceJobsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterFreelanceJobParticipationHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterFreelanceJobParticipation
	t.Cleanup(func() { getCharacterFreelanceJobParticipation = orig })
	lastModified := time.Unix(2000, 0)
	getCharacterFreelanceJobParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent string, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error) {
		return &esiapi.CharactersFreelanceJobsParticipation{Contributed: 500, LastModified: lastModified, State: "committed"}, nil
	}

	handler := getCharacterFreelanceJobParticipationHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterFreelanceJobParticipationInput{
		CharacterID: 7, JobID: "job-1",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.JobID != "job-1" || out.State != "committed" || out.Contributed != 500 || !out.LastModified.Equal(lastModified) {
		t.Errorf("out = %+v, want CharacterID=7 JobID=job-1 State=committed Contributed=500 LastModified=%v", out, lastModified)
	}
}

func TestGetCharacterFreelanceJobParticipationHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterFreelanceJobParticipation
	t.Cleanup(func() { getCharacterFreelanceJobParticipation = orig })
	getCharacterFreelanceJobParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent string, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error) {
		t.Fatal("getCharacterFreelanceJobParticipation should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterFreelanceJobParticipationHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterFreelanceJobParticipationInput{
		CharacterID: 999, JobID: "job-1",
	})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationFreelanceJobsHandler_MapsFieldsAndCursor(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFreelanceJobs
	t.Cleanup(func() { listCorporationFreelanceJobs = orig })
	after := "cursor-after"
	before := "cursor-before"
	lastModified := time.Unix(3000, 0)
	listCorporationFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, afterIn, beforeIn string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
		return []esiapi.FreelanceJobsDetailFreelancejob{
			{
				Id: "job-2", Name: "Corp Job", State: "Active", LastModified: lastModified,
				Progress: esiapi.FreelanceJobsDetailProgress{Current: 5, Desired: 50},
				Reward:   &esiapi.FreelanceJobsDetailReward{Initial: 200000.0, Remaining: 150000.0},
			},
		}, &esiapi.Cursor{After: &after, Before: &before}, nil
	}

	handler := listCorporationFreelanceJobsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationFreelanceJobsInput{
		CharacterID: 7, CorporationID: 98000001,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || len(out.Jobs) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 with one job", out)
	}
	j := out.Jobs[0]
	if j.JobID != "job-2" || j.Name != "Corp Job" || j.State != "Active" || !j.LastModified.Equal(lastModified) ||
		j.Progress.Current != 5 || j.Progress.Desired != 50 ||
		j.Reward == nil || j.Reward.Initial != 200000.0 || j.Reward.Remaining != 150000.0 {
		t.Errorf("job = %+v, want JobID=job-2 Name=\"Corp Job\" State=Active LastModified=%v Progress={5 50} Reward={200000 150000}", j, lastModified)
	}
	if out.NextAfter == nil || *out.NextAfter != after || out.NextBefore == nil || *out.NextBefore != before {
		t.Errorf("out = %+v, want NextAfter=%q NextBefore=%q", out, after, before)
	}
}

func TestListCorporationFreelanceJobsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationFreelanceJobs
	t.Cleanup(func() { listCorporationFreelanceJobs = orig })
	listCorporationFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
		t.Fatal("listCorporationFreelanceJobs should not be called for an unknown character")
		return nil, nil, nil
	}

	handler := listCorporationFreelanceJobsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationFreelanceJobsInput{
		CharacterID: 999, CorporationID: 98000001,
	})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationFreelanceJobParticipantsHandler_MapsFieldsAndCursor(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFreelanceJobParticipants
	t.Cleanup(func() { listCorporationFreelanceJobParticipants = orig })
	after := "participants-after"
	before := "participants-before"
	listCorporationFreelanceJobParticipants = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, jobID string, afterIn, beforeIn string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error) {
		return []esiapi.CorporationsFreelanceJobsParticipantsParticipant{
			{Contributed: 250, Id: 998879, Name: "Some Pilot", State: "committed"},
		}, &esiapi.Cursor{After: &after, Before: &before}, nil
	}

	handler := listCorporationFreelanceJobParticipantsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationFreelanceJobParticipantsInput{
		CharacterID: 7, CorporationID: 98000001, JobID: "job-1",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.JobID != "job-1" || len(out.Participants) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 JobID=job-1 with one participant", out)
	}
	p := out.Participants[0]
	if p.ParticipantID != 998879 || p.Name != "Some Pilot" || p.State != "committed" || p.Contributed != 250 {
		t.Errorf("participant = %+v, want ParticipantID=998879 Name=\"Some Pilot\" State=committed Contributed=250", p)
	}
	if out.NextAfter == nil || *out.NextAfter != after || out.NextBefore == nil || *out.NextBefore != before {
		t.Errorf("out = %+v, want NextAfter=%q NextBefore=%q", out, after, before)
	}
}

func TestListCorporationFreelanceJobParticipantsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationFreelanceJobParticipants
	t.Cleanup(func() { listCorporationFreelanceJobParticipants = orig })
	listCorporationFreelanceJobParticipants = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error) {
		t.Fatal("listCorporationFreelanceJobParticipants should not be called for an unknown character")
		return nil, nil, nil
	}

	handler := listCorporationFreelanceJobParticipantsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationFreelanceJobParticipantsInput{
		CharacterID: 999, CorporationID: 98000001, JobID: "job-1",
	})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// baseFreelanceJobDetail returns a *esiapi.FreelanceJobsDetail with every field other than
// Configuration.Parameters populated, so the Configuration-variant tests below can focus solely
// on the parameter being exercised while still asserting the rest of the mapping (in particular
// Creator, which must come from detail.Details.Creator, not any top-level field).
func baseFreelanceJobDetail(param esiapi.FreelanceJobsDetailConfigurationParametersValue) *esiapi.FreelanceJobsDetail {
	created := time.Unix(4000, 0)
	lastModified := time.Unix(5000, 0)
	return &esiapi.FreelanceJobsDetail{
		AccessAndVisibility: esiapi.FreelanceJobsDetailAccessandvisibility{
			AclProtected: true,
		},
		Configuration: esiapi.FreelanceJobsDetailConfiguration{
			Method: "test_method",
			Parameters: map[string]esiapi.FreelanceJobsDetailConfigurationParametersValue{
				"test_param": param,
			},
			Version: 3,
		},
		Details: esiapi.FreelanceJobsDetailDetails{
			Career:  "Exploration",
			Created: created,
			Creator: esiapi.FreelanceJobsDetailCreator{
				Character:   esiapi.FreelanceJobsDetailCreatorcharacter{Id: 998877, Name: "Creator Pilot"},
				Corporation: esiapi.FreelanceJobsDetailCreatorcorporation{Id: 98000002, Name: "Creator Corp"},
			},
			Description: "A test job",
		},
		Id:           "job-detail-1",
		LastModified: lastModified,
		Name:         "Test Job Detail",
		Progress:     esiapi.FreelanceJobsDetailProgress{Current: 1, Desired: 2},
		State:        "Active",
	}
}

func assertBaseFreelanceJobDetailMapping(t *testing.T, out GetFreelanceJobOutput) {
	t.Helper()
	if out.JobID != "job-detail-1" || out.Name != "Test Job Detail" || out.State != "Active" {
		t.Errorf("out = %+v, want JobID=job-detail-1 Name=\"Test Job Detail\" State=Active", out)
	}
	if !out.LastModified.Equal(time.Unix(5000, 0)) {
		t.Errorf("LastModified = %v, want %v", out.LastModified, time.Unix(5000, 0))
	}
	if out.Progress.Current != 1 || out.Progress.Desired != 2 {
		t.Errorf("Progress = %+v, want {Current:1 Desired:2}", out.Progress)
	}
	if out.Reward != nil {
		t.Errorf("Reward = %+v, want nil", out.Reward)
	}
	if out.Configuration.Method != "test_method" || out.Configuration.Version != 3 {
		t.Errorf("Configuration.Method/Version = %q/%d, want test_method/3", out.Configuration.Method, out.Configuration.Version)
	}
	if out.Contribution != nil {
		t.Errorf("Contribution = %+v, want nil", out.Contribution)
	}
	if out.Creator.Character.ID != 998877 || out.Creator.Character.Name != "Creator Pilot" ||
		out.Creator.Corporation.ID != 98000002 || out.Creator.Corporation.Name != "Creator Corp" {
		t.Errorf("Creator = %+v, want Character={998877 Creator Pilot} Corporation={98000002 Creator Corp} (from detail.Details.Creator)", out.Creator)
	}
	if out.Details.Career != "Exploration" || !out.Details.Created.Equal(time.Unix(4000, 0)) || out.Details.Description != "A test job" ||
		out.Details.Expires != nil || out.Details.Finished != nil {
		t.Errorf("Details = %+v, want Career=Exploration Created=%v Description=\"A test job\" Expires=nil Finished=nil", out.Details, time.Unix(4000, 0))
	}
	if !out.AccessAndVisibility.AclProtected || len(out.AccessAndVisibility.BroadcastLocations) != 0 || out.AccessAndVisibility.Restrictions != nil {
		t.Errorf("AccessAndVisibility = %+v, want AclProtected=true, no BroadcastLocations, no Restrictions", out.AccessAndVisibility)
	}
}

func TestGetFreelanceJobHandler_ConfigurationBooleanVariant(t *testing.T) {
	orig := getFreelanceJob
	t.Cleanup(func() { getFreelanceJob = orig })

	param := esiapi.FreelanceJobsDetailConfigurationParametersValue{
		Boolean: &esiapi.Boolean{
			Boolean: &esiapi.FreelanceJobsDetailParameterboolean{Value: true},
		},
	}
	getFreelanceJob = func(ctx context.Context, userAgent string, jobID string) (*esiapi.FreelanceJobsDetail, error) {
		return baseFreelanceJobDetail(param), nil
	}

	handler := getFreelanceJobHandler("v1")
	_, out, err := handler(context.Background(), nil, GetFreelanceJobInput{JobID: "job-detail-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertBaseFreelanceJobDetailMapping(t, out)

	cp, ok := out.Configuration.Parameters["test_param"]
	if !ok {
		t.Fatalf("Configuration.Parameters missing key \"test_param\": %+v", out.Configuration.Parameters)
	}
	if cp.Kind != "boolean" {
		t.Fatalf("Kind = %q, want boolean", cp.Kind)
	}
	if cp.BooleanValue == nil || *cp.BooleanValue != true {
		t.Errorf("BooleanValue = %v, want true", cp.BooleanValue)
	}
	if cp.CorporationItemDelivery != nil || cp.Matcher != nil || cp.Options != nil {
		t.Errorf("cp = %+v, want only BooleanValue populated", cp)
	}
}

func TestGetFreelanceJobHandler_ConfigurationCorporationItemDeliveryVariant(t *testing.T) {
	orig := getFreelanceJob
	t.Cleanup(func() { getFreelanceJob = orig })

	param := esiapi.FreelanceJobsDetailConfigurationParametersValue{
		CorporationItemDelivery: &esiapi.CorporationItemDelivery{
			CorporationItemDelivery: &esiapi.FreelanceJobsDetailParametercorporationitemdelivery{
				CorporationOfficeLocation: esiapi.FreelanceJobsDetailParametermatcher{
					Values: []esiapi.FreelanceJobsDetailParametermatchervalue{
						{ValueType: "station_id", Values: []string{"60003760"}},
					},
				},
				ItemType: esiapi.FreelanceJobsDetailParametermatcher{
					Values: []esiapi.FreelanceJobsDetailParametermatchervalue{
						{ValueType: "type_id", Values: []string{"34", "35"}},
					},
				},
			},
		},
	}
	getFreelanceJob = func(ctx context.Context, userAgent string, jobID string) (*esiapi.FreelanceJobsDetail, error) {
		return baseFreelanceJobDetail(param), nil
	}

	handler := getFreelanceJobHandler("v1")
	_, out, err := handler(context.Background(), nil, GetFreelanceJobInput{JobID: "job-detail-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertBaseFreelanceJobDetailMapping(t, out)

	cp, ok := out.Configuration.Parameters["test_param"]
	if !ok {
		t.Fatalf("Configuration.Parameters missing key \"test_param\": %+v", out.Configuration.Parameters)
	}
	if cp.Kind != "corporation_item_delivery" {
		t.Fatalf("Kind = %q, want corporation_item_delivery", cp.Kind)
	}
	if cp.BooleanValue != nil || cp.Matcher != nil || cp.Options != nil {
		t.Errorf("cp = %+v, want only CorporationItemDelivery populated", cp)
	}
	if cp.CorporationItemDelivery == nil {
		t.Fatalf("CorporationItemDelivery = nil, want non-nil")
	}
	loc := cp.CorporationItemDelivery.CorporationOfficeLocation
	if len(loc.Values) != 1 || loc.Values[0].ValueType != "station_id" || len(loc.Values[0].Values) != 1 || loc.Values[0].Values[0] != "60003760" {
		t.Errorf("CorporationOfficeLocation = %+v, want one value ValueType=station_id Values=[60003760]", loc)
	}
	item := cp.CorporationItemDelivery.ItemType
	if len(item.Values) != 1 || item.Values[0].ValueType != "type_id" || len(item.Values[0].Values) != 2 ||
		item.Values[0].Values[0] != "34" || item.Values[0].Values[1] != "35" {
		t.Errorf("ItemType = %+v, want one value ValueType=type_id Values=[34 35]", item)
	}
}

func TestGetFreelanceJobHandler_ConfigurationMatcherVariant(t *testing.T) {
	orig := getFreelanceJob
	t.Cleanup(func() { getFreelanceJob = orig })

	param := esiapi.FreelanceJobsDetailConfigurationParametersValue{
		Matcher: &esiapi.Matcher{
			Matcher: &esiapi.FreelanceJobsDetailParametermatcher{
				Values: []esiapi.FreelanceJobsDetailParametermatchervalue{
					{ValueType: "region_id", Values: []string{"10000002"}},
				},
			},
		},
	}
	getFreelanceJob = func(ctx context.Context, userAgent string, jobID string) (*esiapi.FreelanceJobsDetail, error) {
		return baseFreelanceJobDetail(param), nil
	}

	handler := getFreelanceJobHandler("v1")
	_, out, err := handler(context.Background(), nil, GetFreelanceJobInput{JobID: "job-detail-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertBaseFreelanceJobDetailMapping(t, out)

	cp, ok := out.Configuration.Parameters["test_param"]
	if !ok {
		t.Fatalf("Configuration.Parameters missing key \"test_param\": %+v", out.Configuration.Parameters)
	}
	if cp.Kind != "matcher" {
		t.Fatalf("Kind = %q, want matcher", cp.Kind)
	}
	if cp.BooleanValue != nil || cp.CorporationItemDelivery != nil || cp.Options != nil {
		t.Errorf("cp = %+v, want only Matcher populated", cp)
	}
	if cp.Matcher == nil {
		t.Fatalf("Matcher = nil, want non-nil")
	}
	if len(cp.Matcher.Values) != 1 || cp.Matcher.Values[0].ValueType != "region_id" || len(cp.Matcher.Values[0].Values) != 1 ||
		cp.Matcher.Values[0].Values[0] != "10000002" {
		t.Errorf("Matcher.Values = %+v, want one value ValueType=region_id Values=[10000002]", cp.Matcher.Values)
	}
}

func TestGetFreelanceJobHandler_ConfigurationOptionsVariant(t *testing.T) {
	orig := getFreelanceJob
	t.Cleanup(func() { getFreelanceJob = orig })

	param := esiapi.FreelanceJobsDetailConfigurationParametersValue{
		Options: &esiapi.Options{
			Options: &esiapi.FreelanceJobsDetailParameteroptions{Selected: []string{"opt-a", "opt-b"}},
		},
	}
	getFreelanceJob = func(ctx context.Context, userAgent string, jobID string) (*esiapi.FreelanceJobsDetail, error) {
		return baseFreelanceJobDetail(param), nil
	}

	handler := getFreelanceJobHandler("v1")
	_, out, err := handler(context.Background(), nil, GetFreelanceJobInput{JobID: "job-detail-1"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertBaseFreelanceJobDetailMapping(t, out)

	cp, ok := out.Configuration.Parameters["test_param"]
	if !ok {
		t.Fatalf("Configuration.Parameters missing key \"test_param\": %+v", out.Configuration.Parameters)
	}
	if cp.Kind != "options" {
		t.Fatalf("Kind = %q, want options", cp.Kind)
	}
	if cp.BooleanValue != nil || cp.CorporationItemDelivery != nil || cp.Matcher != nil {
		t.Errorf("cp = %+v, want only Options populated", cp)
	}
	if cp.Options == nil || len(cp.Options.Selected) != 2 || cp.Options.Selected[0] != "opt-a" || cp.Options.Selected[1] != "opt-b" {
		t.Errorf("Options = %+v, want Selected=[opt-a opt-b]", cp.Options)
	}
}

func TestListFreelanceJobsHandler_MapsFieldsAndCursor(t *testing.T) {
	orig := listFreelanceJobs
	t.Cleanup(func() { listFreelanceJobs = orig })
	after := "list-after"
	before := "list-before"
	lastModified := time.Unix(6000, 0)
	listFreelanceJobs = func(ctx context.Context, userAgent string, afterIn, beforeIn string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
		return []esiapi.FreelanceJobsDetailFreelancejob{
			{
				Id: "job-3", Name: "Public Job", State: "Active", LastModified: lastModified,
				Progress: esiapi.FreelanceJobsDetailProgress{Current: 7, Desired: 70},
				Reward:   &esiapi.FreelanceJobsDetailReward{Initial: 300000.0, Remaining: 250000.0},
			},
		}, &esiapi.Cursor{After: &after, Before: &before}, nil
	}

	handler := listFreelanceJobsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListFreelanceJobsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Jobs) != 1 {
		t.Fatalf("out = %+v, want one job", out)
	}
	j := out.Jobs[0]
	if j.JobID != "job-3" || j.Name != "Public Job" || j.State != "Active" || !j.LastModified.Equal(lastModified) ||
		j.Progress.Current != 7 || j.Progress.Desired != 70 ||
		j.Reward == nil || j.Reward.Initial != 300000.0 || j.Reward.Remaining != 250000.0 {
		t.Errorf("job = %+v, want JobID=job-3 Name=\"Public Job\" State=Active LastModified=%v Progress={7 70} Reward={300000 250000}", j, lastModified)
	}
	if out.NextAfter == nil || *out.NextAfter != after || out.NextBefore == nil || *out.NextBefore != before {
		t.Errorf("out = %+v, want NextAfter=%q NextBefore=%q", out, after, before)
	}
}

// TestToConfigParameter_AllVariants exercises every branch of toConfigParameter's switch, plus
// the fallback default -- prior coverage never called this mapper directly.
func TestToConfigParameter_AllVariants(t *testing.T) {
	tests := []struct {
		name string
		v    esiapi.FreelanceJobsDetailConfigurationParametersValue
		want string
	}{
		{"boolean", esiapi.FreelanceJobsDetailConfigurationParametersValue{
			Boolean: &esiapi.Boolean{Boolean: &esiapi.FreelanceJobsDetailParameterboolean{Value: true}},
		}, "boolean"},
		{"corporation_item_delivery", esiapi.FreelanceJobsDetailConfigurationParametersValue{
			CorporationItemDelivery: &esiapi.CorporationItemDelivery{CorporationItemDelivery: &esiapi.FreelanceJobsDetailParametercorporationitemdelivery{}},
		}, "corporation_item_delivery"},
		{"matcher", esiapi.FreelanceJobsDetailConfigurationParametersValue{
			Matcher: &esiapi.Matcher{Matcher: &esiapi.FreelanceJobsDetailParametermatcher{
				Values: []esiapi.FreelanceJobsDetailParametermatchervalue{{ValueType: "system", Values: []string{"Jita"}}},
			}},
		}, "matcher"},
		{"options", esiapi.FreelanceJobsDetailConfigurationParametersValue{
			Options: &esiapi.Options{Options: &esiapi.FreelanceJobsDetailParameteroptions{Selected: []string{"a", "b"}}},
		}, "options"},
		{"unrecognized variant (default branch)", esiapi.FreelanceJobsDetailConfigurationParametersValue{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toConfigParameter(tc.v)
			if got.Kind != tc.want {
				t.Errorf("Kind = %q, want %q", got.Kind, tc.want)
			}
		})
	}
}

func TestListCharacterFreelanceJobsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterFreelanceJobs
	t.Cleanup(func() { listCharacterFreelanceJobs = orig })
	wantErr := errors.New("upstream failure")
	listCharacterFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.FreelanceJobsDetailFreelancejob, error) {
		return nil, wantErr
	}

	handler := listCharacterFreelanceJobsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterFreelanceJobsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCharacterFreelanceJobParticipationHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterFreelanceJobParticipation
	t.Cleanup(func() { getCharacterFreelanceJobParticipation = orig })
	wantErr := errors.New("upstream failure")
	getCharacterFreelanceJobParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent string, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error) {
		return nil, wantErr
	}

	handler := getCharacterFreelanceJobParticipationHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterFreelanceJobParticipationInput{
		CharacterID: 7, JobID: "job-1",
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationFreelanceJobsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFreelanceJobs
	t.Cleanup(func() { listCorporationFreelanceJobs = orig })
	wantErr := errors.New("upstream failure")
	listCorporationFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
		return nil, nil, wantErr
	}

	handler := listCorporationFreelanceJobsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationFreelanceJobsInput{
		CharacterID: 7, CorporationID: 98000001,
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationFreelanceJobParticipantsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFreelanceJobParticipants
	t.Cleanup(func() { listCorporationFreelanceJobParticipants = orig })
	wantErr := errors.New("upstream failure")
	listCorporationFreelanceJobParticipants = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error) {
		return nil, nil, wantErr
	}

	handler := listCorporationFreelanceJobParticipantsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationFreelanceJobParticipantsInput{
		CharacterID: 7, CorporationID: 98000001, JobID: "job-1",
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetFreelanceJobHandler_PropagatesError(t *testing.T) {
	orig := getFreelanceJob
	t.Cleanup(func() { getFreelanceJob = orig })
	wantErr := errors.New("upstream failure")
	getFreelanceJob = func(ctx context.Context, userAgent string, jobID string) (*esiapi.FreelanceJobsDetail, error) {
		return nil, wantErr
	}

	handler := getFreelanceJobHandler("v1")
	_, _, err := handler(context.Background(), nil, GetFreelanceJobInput{JobID: "job-detail-1"})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFreelanceJobsHandler_PropagatesError(t *testing.T) {
	orig := listFreelanceJobs
	t.Cleanup(func() { listFreelanceJobs = orig })
	wantErr := errors.New("upstream failure")
	listFreelanceJobs = func(ctx context.Context, userAgent string, after, before string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
		return nil, nil, wantErr
	}

	handler := listFreelanceJobsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListFreelanceJobsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
