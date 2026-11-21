package mcp

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde/sdetest"
)

const (
	reconPilot   = int32(101)
	carrierPilot = int32(102)
)

func readinessEnv(t *testing.T) (*config.Store, *planner) {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{
		{ID: reconPilot, Name: "Recon Pilot", Account: "Main"},
		{ID: carrierPilot, Name: "Carrier Pilot", Account: "Alt"},
	}}); err != nil {
		t.Fatal(err)
	}
	sdeStore := sdetest.NewStore(t, "testdata/industry_fixture.sql", "testdata/readiness_fixture.sql")
	return store, newPlanner("test", "client", store, sdeStore)
}

// requirementsOf gives a character exactly the skills the listed types
// require, read from the fixture, plus extra.
func requirementsOf(t *testing.T, p *planner, extra map[int64]int64, typeIDs ...int64) []CharacterSkill {
	t.Helper()
	levels := map[int64]int64{}
	for _, id := range typeIDs {
		attrs, err := p.sde.TypeAttributes(context.Background(), "test", id)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range requiredSkills(attrs) {
			levels[r.SkillID] = max(levels[r.SkillID], int64(r.Level))
		}
	}
	for id, l := range extra {
		levels[id] = l
	}
	var out []CharacterSkill
	for id, l := range levels {
		out = append(out, CharacterSkill{SkillID: id, ActiveSkillLevel: l, TrainedSkillLevel: l})
	}
	return out
}

func stubSkills(t *testing.T, by map[int32][]CharacterSkill) {
	t.Helper()
	orig := listCharacterSkills
	t.Cleanup(func() { listCharacterSkills = orig })
	listCharacterSkills = func(_ context.Context, _ string, characterID int32, _ string) ([]CharacterSkill, int64, *int64, error) {
		s, ok := by[characterID]
		if !ok {
			return nil, 0, nil, errors.New("missing scope")
		}
		return s, 0, nil, nil
	}
}

func TestListCharacterCapabilities_FromSDERequirements(t *testing.T) {
	store, p := readinessEnv(t)
	stubSkills(t, map[int32][]CharacterSkill{
		reconPilot:   requirementsOf(t, p, map[int64]int64{21603: 4}, 11957, 21096), // Falcon and the standard cyno, CFT IV
		carrierPilot: requirementsOf(t, p, map[int64]int64{83094: 1}, 23757),        // Archon, Capital Jump Portal Generation I
	})
	_, out, err := listCharacterCapabilitiesHandler("test", "client", store, p.sde)(context.Background(), nil, ListCharacterCapabilitiesInput{IncludeMissing: true})
	if err != nil {
		t.Fatal(err)
	}
	find := func(cc CharacterCapabilities, kind, variant string) AbilityView {
		for _, a := range cc.Abilities {
			if a.Kind == kind && a.Variant == variant {
				return a
			}
		}
		t.Fatalf("%s has no %s %s entry", cc.CharacterName, variant, kind)
		return AbilityView{}
	}
	recon, carrier := out.Characters[0], out.Characters[1]
	if a := find(recon, "cyno", "standard"); !a.Can || !slices.Equal(a.Hulls, []string{"Falcon"}) {
		t.Errorf("standard cyno = %+v", a)
	}
	if a := find(recon, "cyno", "covert"); a.Can || !slices.Equal(a.MissingSkills, []string{"Cynosural Field Theory V"}) {
		t.Errorf("covert cyno = %+v, want missing CFT V", a)
	}
	if a := find(recon, "cyno", "industrial"); a.Can {
		t.Errorf("industrial cyno = %+v, want no hull they can fly", a)
	}
	if a := find(carrier, "conduit", "carrier"); !a.Can || !slices.Equal(a.Hulls, []string{"Archon"}) {
		t.Errorf("carrier conduit = %+v", a)
	}
	if a := find(carrier, "cyno", "standard"); a.Can {
		t.Errorf("carrier pilot can light a cyno? %+v", a)
	}
	if len(carrier.Hulls) != 1 || carrier.Hulls[0].Class != "Carrier" || !slices.Equal(carrier.Hulls[0].Hulls, []string{"Archon"}) {
		t.Errorf("carrier hulls = %+v", carrier.Hulls)
	}

	_, only, _ := listCharacterCapabilitiesHandler("test", "client", store, p.sde)(context.Background(), nil, ListCharacterCapabilitiesInput{CharacterIDs: []int32{reconPilot}})
	if len(only.Characters) != 1 || len(only.Characters[0].Abilities) != 1 {
		t.Errorf("without includeMissing = %+v, want only the standard cyno", only.Characters)
	}
}

func TestListCharacterCapabilities_UnreadableSkillsAreAWarning(t *testing.T) {
	store, p := readinessEnv(t)
	stubSkills(t, map[int32][]CharacterSkill{reconPilot: nil})
	_, out, err := listCharacterCapabilitiesHandler("test", "client", store, p.sde)(context.Background(), nil, ListCharacterCapabilitiesInput{})
	if err != nil || len(out.Characters) != 1 || len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Carrier Pilot") {
		t.Errorf("out = %+v, %v", out, err)
	}
}

func TestListJumpClones_CooldownCapacityAndNames(t *testing.T) {
	store, p := readinessEnv(t)
	cfg, _ := store.Load()
	cfg.UpsertStructure(config.Structure{ID: 1049345291823, Name: "CX7-70 Sotiyo", Kind: config.StructureUpwell})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	stubSkills(t, map[int32][]CharacterSkill{
		reconPilot: {{SkillID: 33399, ActiveSkillLevel: 4}, {SkillID: 24242, ActiveSkillLevel: 5}, {SkillID: 33407, ActiveSkillLevel: 2}},
	})
	origC, origN := getCharacterClones, now
	t.Cleanup(func() { getCharacterClones, now = origC, origN })
	now = func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) }
	last := time.Date(2026, 11, 19, 20, 0, 0, 0, time.UTC)
	home := int64(60003760)
	kind := "station"
	getCharacterClones = func(_ context.Context, _ string, characterID int32, _ string) (GetCharacterClonesOutput, error) {
		if characterID != reconPilot {
			return GetCharacterClonesOutput{}, errors.New("missing scope")
		}
		return GetCharacterClonesOutput{CharacterID: characterID, LastCloneJumpDate: &last, HomeLocationID: &home, HomeLocationType: &kind,
			JumpClones: []JumpClone{{LocationID: 1049345291823, LocationType: "structure", Implants: []int64{34}}, {LocationID: 1, LocationType: "structure"}}}, nil
	}
	_, out, err := listJumpClonesHandler("test", "client", store, p.sde)(context.Background(), nil, ListJumpClonesInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Characters) != 1 || len(out.Warnings) != 1 {
		t.Fatalf("out = %+v", out)
	}
	c := out.Characters[0]
	if c.CooldownHours != 20 || c.ReadyNow || !c.NextCloneJump.Equal(time.Date(2026, 11, 20, 16, 0, 0, 0, time.UTC)) || c.Capacity != 7 {
		t.Errorf("readiness = %+v next %v", c, c.NextCloneJump)
	}
	if c.Home == nil || !strings.HasPrefix(c.Home.LocationName, "Jita 4") {
		t.Errorf("home = %+v", c.Home)
	}
	if c.JumpClones[0].LocationName != "CX7-70 Sotiyo" || !slices.Equal(c.JumpClones[0].Implants, []string{"Tritanium"}) || c.JumpClones[1].LocationName != "" {
		t.Errorf("clones = %+v", c.JumpClones)
	}
}
