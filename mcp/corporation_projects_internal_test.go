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

func TestGetCorporationProjectContributionHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectContribution
	t.Cleanup(func() { getCorporationProjectContribution = orig })
	lastModified := time.Unix(1000, 0)
	getCorporationProjectContribution = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, targetCharacterID int64) (*esiapi.CorporationsProjectsContribution, error) {
		return &esiapi.CorporationsProjectsContribution{Contributed: 500, LastModified: &lastModified}, nil
	}

	handler := getCorporationProjectContributionHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectContributionInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123", TargetCharacterID: 998878,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.ProjectID != "abc-123" ||
		out.TargetCharacterID != 998878 || out.Contributed != 500 ||
		out.LastModified == nil || !out.LastModified.Equal(lastModified) {
		t.Errorf("out = %+v, want CharacterID=7 CorporationID=98000001 ProjectID=abc-123 TargetCharacterID=998878 Contributed=500 LastModified=%v", out, lastModified)
	}
}

func TestGetCorporationProjectContributionHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationProjectContribution
	t.Cleanup(func() { getCorporationProjectContribution = orig })
	getCorporationProjectContribution = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, targetCharacterID int64) (*esiapi.CorporationsProjectsContribution, error) {
		t.Fatal("getCorporationProjectContribution should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationProjectContributionHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationProjectContributionInput{
		CharacterID: 999, CorporationID: 98000001, ProjectID: "abc-123", TargetCharacterID: 998878,
	})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationProjectContributorsHandler_MapsFieldsAndCursor(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationProjectContributors
	t.Cleanup(func() { listCorporationProjectContributors = orig })
	after := "a"
	before := "b"
	listCorporationProjectContributors = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, afterIn, beforeIn string, limit int64) ([]esiapi.CorporationsProjectsContributorsContributor, *esiapi.Cursor, error) {
		return []esiapi.CorporationsProjectsContributorsContributor{
			{Contributed: 500, Id: 998879, Name: "Some Pilot"},
		}, &esiapi.Cursor{After: &after, Before: &before}, nil
	}

	handler := listCorporationProjectContributorsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationProjectContributorsInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.ProjectID != "abc-123" || len(out.Contributors) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 ProjectID=abc-123 with one contributor", out)
	}
	c := out.Contributors[0]
	if c.ContributorID != 998879 || c.Name != "Some Pilot" || c.Contributed != 500 {
		t.Errorf("contributor = %+v, want ContributorID=998879 Name=%q Contributed=500", c, "Some Pilot")
	}
	if out.NextAfter == nil || *out.NextAfter != after || out.NextBefore == nil || *out.NextBefore != before {
		t.Errorf("out = %+v, want NextAfter=%q NextBefore=%q", out, after, before)
	}
}

func TestListCorporationProjectContributorsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationProjectContributors
	t.Cleanup(func() { listCorporationProjectContributors = orig })
	listCorporationProjectContributors = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, after, before string, limit int64) ([]esiapi.CorporationsProjectsContributorsContributor, *esiapi.Cursor, error) {
		t.Fatal("listCorporationProjectContributors should not be called for an unknown character")
		return nil, nil, nil
	}

	handler := listCorporationProjectContributorsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationProjectContributorsInput{
		CharacterID: 999, CorporationID: 98000001, ProjectID: "abc-123",
	})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationProjectDetailHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })

	archetypeID := int64(1)
	factionID := int64(500001)
	constellationID := int64(20000001)
	participationLimit := int64(10)
	rewardPerContribution := 1.5
	submissionLimit := int64(100)
	submissionMultiplier := 2.0
	created := time.Unix(2000, 0)
	lastModified := time.Unix(3000, 0)

	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return &esiapi.CorporationsProjectsDetail{
			Configuration: esiapi.CorporationsProjectsDetailConfiguration{
				CaptureFwComplex: &esiapi.CaptureFwComplex{
					CaptureFwComplex: &esiapi.CorporationsProjectsDetailConfigurationcapturefwcomplex{
						Archetypes: []esiapi.CorporationsProjectsDetailConfigurationmatcherarchetype{{ArchetypeId: &archetypeID}},
						Factions:   []esiapi.CorporationsProjectsDetailConfigurationmatcherfaction{{FactionId: &factionID}},
						Locations: []esiapi.CorporationsProjectsDetailConfigurationcapturefwcomplexLocationsInner{
							{ConstellationId: &esiapi.ConstellationId{ConstellationId: &constellationID}},
						},
					},
				},
			},
			Contribution: &esiapi.CorporationsProjectsDetailContribution{
				ParticipationLimit:    &participationLimit,
				RewardPerContribution: &rewardPerContribution,
				SubmissionLimit:       &submissionLimit,
				SubmissionMultiplier:  &submissionMultiplier,
			},
			Creator: esiapi.CorporationsProjectsDetailCreator{Id: 998877, Name: "Creator Pilot"},
			Details: esiapi.CorporationsProjectsDetailDetails{
				Career: "Exploration", Created: created, Description: "A test project",
			},
			Id:           "abc-123",
			LastModified: lastModified,
			Name:         "Test Project",
			Progress:     esiapi.CorporationsProjectsDetailProgress{Current: 10, Desired: 100},
			Reward:       &esiapi.CorporationsProjectsDetailReward{Initial: 1000000.0, Remaining: 500000.0},
			State:        "Active",
		}, nil
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.ProjectID != "abc-123" {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 ProjectID=abc-123", out)
	}

	cfg := out.Configuration
	if cfg.Kind != "capture_fw_complex" {
		t.Fatalf("Configuration.Kind = %q, want capture_fw_complex", cfg.Kind)
	}
	if len(cfg.Archetypes) != 1 || cfg.Archetypes[0].ArchetypeID == nil || *cfg.Archetypes[0].ArchetypeID != archetypeID {
		t.Errorf("Configuration.Archetypes = %+v, want one entry ArchetypeID=%d", cfg.Archetypes, archetypeID)
	}
	if len(cfg.Factions) != 1 || cfg.Factions[0].FactionID == nil || *cfg.Factions[0].FactionID != factionID {
		t.Errorf("Configuration.Factions = %+v, want one entry FactionID=%d", cfg.Factions, factionID)
	}
	if len(cfg.Locations) != 1 || cfg.Locations[0].ConstellationID == nil || *cfg.Locations[0].ConstellationID != constellationID {
		t.Errorf("Configuration.Locations = %+v, want one entry ConstellationID=%d", cfg.Locations, constellationID)
	}

	if out.Contribution == nil {
		t.Fatal("Contribution = nil, want non-nil")
	}
	if out.Contribution.ParticipationLimit == nil || *out.Contribution.ParticipationLimit != participationLimit ||
		out.Contribution.RewardPerContribution == nil || *out.Contribution.RewardPerContribution != rewardPerContribution ||
		out.Contribution.SubmissionLimit == nil || *out.Contribution.SubmissionLimit != submissionLimit ||
		out.Contribution.SubmissionMultiplier == nil || *out.Contribution.SubmissionMultiplier != submissionMultiplier {
		t.Errorf("Contribution = %+v, want ParticipationLimit=%d RewardPerContribution=%v SubmissionLimit=%d SubmissionMultiplier=%v",
			out.Contribution, participationLimit, rewardPerContribution, submissionLimit, submissionMultiplier)
	}

	if out.Creator.CreatorID != 998877 || out.Creator.Name != "Creator Pilot" {
		t.Errorf("Creator = %+v, want {CreatorID:998877 Name:\"Creator Pilot\"}", out.Creator)
	}

	if out.Details.Career != "Exploration" || !out.Details.Created.Equal(created) || out.Details.Description != "A test project" ||
		out.Details.Expires != nil || out.Details.Finished != nil {
		t.Errorf("Details = %+v, want Career=Exploration Created=%v Description=\"A test project\" Expires=nil Finished=nil", out.Details, created)
	}

	if !out.LastModified.Equal(lastModified) {
		t.Errorf("LastModified = %v, want %v", out.LastModified, lastModified)
	}
	if out.Name != "Test Project" {
		t.Errorf("Name = %q, want \"Test Project\"", out.Name)
	}
	if out.Progress.Current != 10 || out.Progress.Desired != 100 {
		t.Errorf("Progress = %+v, want {Current:10 Desired:100}", out.Progress)
	}
	if out.Reward == nil || out.Reward.Initial != 1000000.0 || out.Reward.Remaining != 500000.0 {
		t.Errorf("Reward = %+v, want {Initial:1000000 Remaining:500000}", out.Reward)
	}
	if out.State != "Active" {
		t.Errorf("State = %q, want \"Active\"", out.State)
	}
}

func TestGetCorporationProjectDetailHandler_ConfigurationManualVariant(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })

	manualData := map[string]interface{}{"foo": "bar", "count": float64(3)}
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return &esiapi.CorporationsProjectsDetail{
			Configuration: esiapi.CorporationsProjectsDetailConfiguration{
				Manual: &esiapi.Manual{Manual: manualData},
			},
			Creator: esiapi.CorporationsProjectsDetailCreator{Id: 998877, Name: "Creator Pilot"},
			Details: esiapi.CorporationsProjectsDetailDetails{Career: "Exploration", Description: "A manual project"},
			Id:      "abc-124",
			Name:    "Manual Project",
			State:   "Active",
		}, nil
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-124",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Configuration.Kind != "manual" {
		t.Fatalf("Configuration.Kind = %q, want manual", out.Configuration.Kind)
	}
	if len(out.Configuration.Manual) != len(manualData) || out.Configuration.Manual["foo"] != "bar" || out.Configuration.Manual["count"] != float64(3) {
		t.Errorf("Configuration.Manual = %+v, want %+v", out.Configuration.Manual, manualData)
	}
}

func TestGetCorporationProjectDetailHandler_ConfigurationUnknownVariant(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })

	unknownData := map[string]interface{}{"raw": "payload"}
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return &esiapi.CorporationsProjectsDetail{
			Configuration: esiapi.CorporationsProjectsDetailConfiguration{
				Unknown: &esiapi.Unknown{
					Unknown: &esiapi.CorporationsProjectsDetailConfigurationunknown{
						Type: "some_future_type",
						Data: unknownData,
					},
				},
			},
			Creator: esiapi.CorporationsProjectsDetailCreator{Id: 998877, Name: "Creator Pilot"},
			Details: esiapi.CorporationsProjectsDetailDetails{Career: "Exploration", Description: "An unknown project"},
			Id:      "abc-125",
			Name:    "Unknown Project",
			State:   "Active",
		}, nil
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-125",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Configuration.Kind != "unknown" {
		t.Fatalf("Configuration.Kind = %q, want unknown", out.Configuration.Kind)
	}
	if out.Configuration.UnknownType != "some_future_type" {
		t.Errorf("Configuration.UnknownType = %q, want \"some_future_type\"", out.Configuration.UnknownType)
	}
	unknownDataOut, ok := out.Configuration.UnknownData.(map[string]interface{})
	if !ok || unknownDataOut["raw"] != "payload" {
		t.Errorf("Configuration.UnknownData = %+v, want %+v", out.Configuration.UnknownData, unknownData)
	}
}

func TestGetCorporationProjectDetailHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		t.Fatal("getCorporationProjectDetail should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{
		CharacterID: 999, CorporationID: 98000001, ProjectID: "abc-123",
	})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationProjectDetailHandler_ResolvesShipTypeMatcherTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })
	typeID := int64(587)
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return &esiapi.CorporationsProjectsDetail{
			Configuration: esiapi.CorporationsProjectsDetailConfiguration{
				DamageShip: &esiapi.DamageShip{DamageShip: &esiapi.CorporationsProjectsDetailConfigurationdamageship{
					Ships: []esiapi.CorporationsProjectsDetailConfigurationdamageshipShipsInner{{TypeId: &esiapi.TypeId{TypeId: &typeID}}},
				}},
			},
			Creator:  esiapi.CorporationsProjectsDetailCreator{Id: 998877, Name: "Creator Pilot"},
			Details:  esiapi.CorporationsProjectsDetailDetails{Career: "Combat"},
			Id:       "abc-123",
			Name:     "Test Project",
			Progress: esiapi.CorporationsProjectsDetailProgress{Current: 0, Desired: 1},
			State:    "Active",
		}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Configuration.Ships) != 1 || out.Configuration.Ships[0].TypeName != "Rifter" {
		t.Errorf("Configuration.Ships = %+v, want one entry TypeName=%q", out.Configuration.Ships, "Rifter")
	}
}

func TestListCorporationProjectsHandler_MapsFieldsAndCursor(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationProjects
	t.Cleanup(func() { listCorporationProjects = orig })
	after := "cursor-after"
	before := "cursor-before"
	lastModified := time.Unix(4000, 0)
	listCorporationProjects = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, afterIn, beforeIn string, limit int64, state string) ([]esiapi.CorporationsProjectsDetailProject, *esiapi.Cursor, error) {
		return []esiapi.CorporationsProjectsDetailProject{
			{
				Id: "proj-1", LastModified: lastModified, Name: "Test Project",
				Progress: esiapi.CorporationsProjectsDetailProgress{Current: 5, Desired: 50},
				Reward:   &esiapi.CorporationsProjectsDetailReward{Initial: 200000.0, Remaining: 150000.0},
				State:    "Active",
			},
		}, &esiapi.Cursor{After: &after, Before: &before}, nil
	}

	handler := listCorporationProjectsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationProjectsInput{
		CharacterID: 7, CorporationID: 98000001,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || len(out.Projects) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 with one project", out)
	}
	p := out.Projects[0]
	if p.ProjectID != "proj-1" || p.Name != "Test Project" || p.State != "Active" || !p.LastModified.Equal(lastModified) ||
		p.Progress.Current != 5 || p.Progress.Desired != 50 ||
		p.Reward == nil || p.Reward.Initial != 200000.0 || p.Reward.Remaining != 150000.0 {
		t.Errorf("project = %+v, want ProjectID=proj-1 Name=\"Test Project\" State=Active LastModified=%v Progress={5 50} Reward={200000 150000}", p, lastModified)
	}
	if out.NextAfter == nil || *out.NextAfter != after || out.NextBefore == nil || *out.NextBefore != before {
		t.Errorf("out = %+v, want NextAfter=%q NextBefore=%q", out, after, before)
	}
}

func TestListCorporationProjectsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationProjects
	t.Cleanup(func() { listCorporationProjects = orig })
	listCorporationProjects = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64, state string) ([]esiapi.CorporationsProjectsDetailProject, *esiapi.Cursor, error) {
		t.Fatal("listCorporationProjects should not be called for an unknown character")
		return nil, nil, nil
	}

	handler := listCorporationProjectsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationProjectsInput{
		CharacterID: 999, CorporationID: 98000001,
	})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// TestToProjectConfiguration_AllVariants exercises every one of the 17 branches of
// toProjectConfiguration's switch, plus the fallback default -- prior coverage only reached 3 of
// them because every other test in this file stubs the esi-facade wrapper above the mapping
// layer, so toProjectConfiguration itself never actually ran under test.
func TestToProjectConfiguration_AllVariants(t *testing.T) {
	tests := []struct {
		name string
		cfg  esiapi.CorporationsProjectsDetailConfiguration
		want string
	}{
		{"capture_fw_complex", esiapi.CorporationsProjectsDetailConfiguration{
			CaptureFwComplex: &esiapi.CaptureFwComplex{CaptureFwComplex: &esiapi.CorporationsProjectsDetailConfigurationcapturefwcomplex{}},
		}, "capture_fw_complex"},
		{"damage_ship", esiapi.CorporationsProjectsDetailConfiguration{
			DamageShip: &esiapi.DamageShip{DamageShip: &esiapi.CorporationsProjectsDetailConfigurationdamageship{}},
		}, "damage_ship"},
		{"defend_fw_complex", esiapi.CorporationsProjectsDetailConfiguration{
			DefendFwComplex: &esiapi.DefendFwComplex{DefendFwComplex: &esiapi.CorporationsProjectsDetailConfigurationdefendfwcomplex{}},
		}, "defend_fw_complex"},
		{"deliver_item", esiapi.CorporationsProjectsDetailConfiguration{
			DeliverItem: &esiapi.DeliverItem{DeliverItem: &esiapi.CorporationsProjectsDetailConfigurationdeliveritem{}},
		}, "deliver_item"},
		{"destroy_npc", esiapi.CorporationsProjectsDetailConfiguration{
			DestroyNpc: &esiapi.DestroyNpc{DestroyNpc: &esiapi.CorporationsProjectsDetailConfigurationdestroynpc{}},
		}, "destroy_npc"},
		{"destroy_ship", esiapi.CorporationsProjectsDetailConfiguration{
			DestroyShip: &esiapi.DestroyShip{DestroyShip: &esiapi.CorporationsProjectsDetailConfigurationdestroyship{}},
		}, "destroy_ship"},
		{"earn_loyalty_point", esiapi.CorporationsProjectsDetailConfiguration{
			EarnLoyaltyPoint: &esiapi.EarnLoyaltyPoint{EarnLoyaltyPoint: &esiapi.CorporationsProjectsDetailConfigurationearnloyaltypoints{}},
		}, "earn_loyalty_point"},
		{"lost_ship", esiapi.CorporationsProjectsDetailConfiguration{
			LostShip: &esiapi.LostShip{LostShip: &esiapi.CorporationsProjectsDetailConfigurationlostship{}},
		}, "lost_ship"},
		{"manual", esiapi.CorporationsProjectsDetailConfiguration{
			Manual: &esiapi.Manual{Manual: map[string]interface{}{"note": "test"}},
		}, "manual"},
		{"manufacture_item", esiapi.CorporationsProjectsDetailConfiguration{
			ManufactureItem: &esiapi.ManufactureItem{ManufactureItem: &esiapi.CorporationsProjectsDetailConfigurationmanufactureitem{}},
		}, "manufacture_item"},
		{"mine_material", esiapi.CorporationsProjectsDetailConfiguration{
			MineMaterial: &esiapi.MineMaterial{MineMaterial: &esiapi.CorporationsProjectsDetailConfigurationminematerial{}},
		}, "mine_material"},
		{"remote_boost_shield", esiapi.CorporationsProjectsDetailConfiguration{
			RemoteBoostShield: &esiapi.RemoteBoostShield{RemoteBoostShield: &esiapi.CorporationsProjectsDetailConfigurationremoteboostshield{}},
		}, "remote_boost_shield"},
		{"remote_repair_armor", esiapi.CorporationsProjectsDetailConfiguration{
			RemoteRepairArmor: &esiapi.RemoteRepairArmor{RemoteRepairArmor: &esiapi.CorporationsProjectsDetailConfigurationremoterepairarmor{}},
		}, "remote_repair_armor"},
		{"salvage_wreck", esiapi.CorporationsProjectsDetailConfiguration{
			SalvageWreck: &esiapi.SalvageWreck{SalvageWreck: &esiapi.CorporationsProjectsDetailConfigurationsalvagewreck{}},
		}, "salvage_wreck"},
		{"scan_signature", esiapi.CorporationsProjectsDetailConfiguration{
			ScanSignature: &esiapi.ScanSignature{ScanSignature: &esiapi.CorporationsProjectsDetailConfigurationscansignature{}},
		}, "scan_signature"},
		{"ship_insurance", esiapi.CorporationsProjectsDetailConfiguration{
			ShipInsurance: &esiapi.ShipInsurance{ShipInsurance: &esiapi.CorporationsProjectsDetailConfigurationshipinsurance{}},
		}, "ship_insurance"},
		{"unknown", esiapi.CorporationsProjectsDetailConfiguration{
			Unknown: &esiapi.Unknown{Unknown: &esiapi.CorporationsProjectsDetailConfigurationunknown{Type: "some_future_type"}},
		}, "unknown"},
		{"unrecognized variant (default branch)", esiapi.CorporationsProjectsDetailConfiguration{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toProjectConfiguration(tc.cfg)
			if got.Kind != tc.want {
				t.Errorf("Kind = %q, want %q", got.Kind, tc.want)
			}
		})
	}
}

func TestGetCorporationProjectContributionHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectContribution
	t.Cleanup(func() { getCorporationProjectContribution = orig })
	wantErr := errors.New("upstream failure")
	getCorporationProjectContribution = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, targetCharacterID int64) (*esiapi.CorporationsProjectsContribution, error) {
		return nil, wantErr
	}

	handler := getCorporationProjectContributionHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationProjectContributionInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123", TargetCharacterID: 998878,
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationProjectContributorsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationProjectContributors
	t.Cleanup(func() { listCorporationProjectContributors = orig })
	wantErr := errors.New("upstream failure")
	listCorporationProjectContributors = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string, after, before string, limit int64) ([]esiapi.CorporationsProjectsContributorsContributor, *esiapi.Cursor, error) {
		return nil, nil, wantErr
	}

	handler := listCorporationProjectContributorsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationProjectContributorsInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123",
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationProjectDetailHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationProjectDetail
	t.Cleanup(func() { getCorporationProjectDetail = orig })
	wantErr := errors.New("upstream failure")
	getCorporationProjectDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, projectID string) (*esiapi.CorporationsProjectsDetail, error) {
		return nil, wantErr
	}

	handler := getCorporationProjectDetailHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationProjectDetailInput{
		CharacterID: 7, CorporationID: 98000001, ProjectID: "abc-123",
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationProjectsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationProjects
	t.Cleanup(func() { listCorporationProjects = orig })
	wantErr := errors.New("upstream failure")
	listCorporationProjects = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64, state string) ([]esiapi.CorporationsProjectsDetailProject, *esiapi.Cursor, error) {
		return nil, nil, wantErr
	}

	handler := listCorporationProjectsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationProjectsInput{
		CharacterID: 7, CorporationID: 98000001,
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
