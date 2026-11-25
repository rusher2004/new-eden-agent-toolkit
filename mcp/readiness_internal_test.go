package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
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
	openESI, openPublicESI := noESI()
	return store, newPlanner("test", openESI, openPublicESI, store, sdeStore)
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
	maps.Copy(levels, extra)
	var out []CharacterSkill
	for id, l := range levels {
		out = append(out, CharacterSkill{SkillID: id, ActiveSkillLevel: l, TrainedSkillLevel: l})
	}
	return out
}

// stubSkills gives each character in by its skills; any other character's
// skills can't be read.
func stubSkills(p *planner, by map[int32][]CharacterSkill) {
	addESILayer(p, func(id int32, f *fakeESI) {
		f.characterSkills = func(context.Context) (*esiapi.CharactersSkills, error) {
			s, ok := by[id]
			if !ok {
				return nil, errors.New("missing scope")
			}
			out := &esiapi.CharactersSkills{Skills: []esiapi.CharactersSkillsSkill{}}
			for _, sk := range s {
				out.Skills = append(out.Skills, esiapi.CharactersSkillsSkill{SkillId: sk.SkillID, ActiveSkillLevel: sk.ActiveSkillLevel, TrainedSkillLevel: sk.TrainedSkillLevel, SkillpointsInSkill: sk.SkillpointsInSkill})
			}
			return out, nil
		}
	})
}

func TestListCharacterCapabilities_FromSDERequirements(t *testing.T) {
	store, p := readinessEnv(t)
	stubSkills(p, map[int32][]CharacterSkill{
		reconPilot:   requirementsOf(t, p, map[int64]int64{21603: 4}, 11957, 21096), // Falcon and the standard cyno, CFT IV
		carrierPilot: requirementsOf(t, p, map[int64]int64{83094: 1}, 23757),        // Archon, Capital Jump Portal Generation I
	})
	_, out, err := listCharacterCapabilitiesHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, ListCharacterCapabilitiesInput{IncludeMissing: true})
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
	if len(out.Characters) != 2 {
		t.Fatalf("characters = %+v, want 2", out.Characters)
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

	_, only, err := listCharacterCapabilitiesHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, ListCharacterCapabilitiesInput{CharacterIDs: []int32{reconPilot}})
	if err != nil {
		t.Fatal(err)
	}
	if len(only.Characters) != 1 || len(only.Characters[0].Abilities) != 1 {
		t.Errorf("without includeMissing = %+v, want only the standard cyno", only.Characters)
	}
}

func TestListCharacterCapabilities_UnreadableSkillsAreAWarning(t *testing.T) {
	store, p := readinessEnv(t)
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: nil})
	_, out, err := listCharacterCapabilitiesHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, ListCharacterCapabilitiesInput{})
	if err != nil || len(out.Characters) != 1 || len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Carrier Pilot") {
		t.Errorf("out = %+v, %v", out, err)
	}
}

func TestListJumpClones_CooldownCapacityAndNames(t *testing.T) {
	store, p := readinessEnv(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.UpsertStructure(config.Structure{ID: 1049345291823, Name: "CX7-70 Sotiyo", Kind: config.StructureUpwell})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	stubSkills(p, map[int32][]CharacterSkill{
		reconPilot: {{SkillID: 33399, ActiveSkillLevel: 4}, {SkillID: 24242, ActiveSkillLevel: 5}, {SkillID: 33407, ActiveSkillLevel: 2}},
	})
	origN := now
	t.Cleanup(func() { now = origN })
	stubImplants(p, func(int32) ([]int64, error) { return []int64{35}, nil })
	stubLocations(p, map[int32]int64{reconPilot: 1049345291823})
	now = func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) }
	last := time.Date(2026, 11, 19, 20, 0, 0, 0, time.UTC)
	home := int64(60003760)
	kind := "station"
	stubClones(p, func(characterID int32) (GetCharacterClonesOutput, error) {
		if characterID != reconPilot {
			return GetCharacterClonesOutput{}, errors.New("missing scope")
		}
		return GetCharacterClonesOutput{CharacterID: characterID, LastCloneJumpDate: &last, HomeLocationID: &home, HomeLocationType: &kind,
			JumpClones: []JumpClone{{LocationID: 1049345291823, LocationType: "structure", Implants: []int64{34}}, {LocationID: 1, LocationType: "structure"}}}, nil
	})
	_, out, err := listJumpClonesHandler("test", fakeOpener(p), p.openPublicESI, store, p.sde)(context.Background(), nil, ListJumpClonesInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Characters) != 1 || len(out.Warnings) != 1 {
		t.Fatalf("out = %+v", out)
	}
	c := out.Characters[0]
	if c.CooldownHours != 20 || c.ReadyNow || c.CooldownReady || c.JumpFromHere.Status != jumpFromYes || !c.NextCloneJump.Equal(time.Date(2026, 11, 20, 16, 0, 0, 0, time.UTC)) || c.Capacity != 7 {
		t.Errorf("readiness = %+v next %v", c, c.NextCloneJump)
	}
	if c.Home == nil || c.Home.LocationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("home = %+v", c.Home)
	}
	if c.JumpClones[0].LocationName != "CX7-70 Sotiyo" || !slices.Equal(c.JumpClones[0].Implants, []string{"Tritanium"}) || c.JumpClones[1].LocationName != "" {
		t.Errorf("clones = %+v", c.JumpClones)
	}
	if c.ActiveImplants == nil || !slices.Equal(*c.ActiveImplants, []string{"Pyerite"}) {
		t.Errorf("active implants = %v, want the current clone's", c.ActiveImplants)
	}
}

// A clone with no implants reads as an empty list; implants that couldn't be
// read are left out, so neither is mistaken for the other.
func TestListJumpClones_NoImplantsIsNotUnknownImplants(t *testing.T) {
	store, p := readinessEnv(t)
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: nil, carrierPilot: nil})
	stubLocations(p, map[int32]int64{})
	stubClones(p, func(characterID int32) (GetCharacterClonesOutput, error) {
		return GetCharacterClonesOutput{CharacterID: characterID}, nil
	})
	stubImplants(p, func(characterID int32) ([]int64, error) {
		if characterID == carrierPilot {
			return nil, errors.New("missing scope")
		}
		return nil, nil
	})
	_, out, err := listJumpClonesHandler("test", fakeOpener(p), p.openPublicESI, store, p.sde)(context.Background(), nil, ListJumpClonesInput{})
	if err != nil {
		t.Fatal(err)
	}
	none, unknown := out.Characters[0], out.Characters[1]
	if none.ActiveImplants == nil || len(*none.ActiveImplants) != 0 {
		t.Errorf("no implants: %v, want an empty list", none.ActiveImplants)
	}
	if unknown.ActiveImplants != nil || len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "unknown, not none") {
		t.Errorf("unreadable implants: %v, warnings %v; want them left out with a warning", unknown.ActiveImplants, out.Warnings)
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"activeImplants":[]`) || strings.Count(string(b), "activeImplants") != 1 {
		t.Errorf("json = %s, want activeImplants:[] for the first character only", b)
	}
}

// stubLocations docks each character at the given station or structure
// (stations below 1e9), leaving the rest in space.
func stubLocations(p *planner, docked map[int32]int64) {
	addESILayer(p, func(id int32, f *fakeESI) {
		f.characterLocation = func(context.Context) (*esiapi.CharactersLocation, error) {
			at, ok := docked[id]
			switch {
			case !ok:
				return &esiapi.CharactersLocation{SolarSystemId: 30000142}, nil
			case at < 1e9:
				return &esiapi.CharactersLocation{SolarSystemId: 30000142, StationId: &at}, nil
			default:
				return &esiapi.CharactersLocation{SolarSystemId: 30000142, StructureId: &at}, nil
			}
		}
	})
}

// readyNow needs a clone bay where the character is docked, not just a
// finished cooldown.
func TestListJumpClones_ReadyNowNeedsACloneBayHere(t *testing.T) {
	const noBay, unknownBay, provenBay = int64(1049345291823), int64(1049345291824), int64(60003760)
	tests := []struct {
		name       string
		docked     map[int32]int64
		wantStatus string
		wantReady  bool
	}{
		{"saved with no clone bay", map[int32]int64{reconPilot: noBay}, jumpFromNo, false},
		{"clone bay unknown", map[int32]int64{reconPilot: unknownBay}, jumpFromUnknown, false},
		{"another character's jump clone proves it", map[int32]int64{reconPilot: provenBay}, jumpFromYes, true},
		{"in space", map[int32]int64{}, jumpFromNotDocked, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, p := readinessEnv(t)
			cfg, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			cfg.UpsertStructure(config.Structure{ID: noBay, Name: "No Bay Fortizar", Kind: config.StructureUpwell,
				CloneBay: &config.CloneBay{Status: config.CloneBayNo, Source: config.CloneBayFromUser, Checked: "2026-11-01"}})
			if err := store.Save(cfg); err != nil {
				t.Fatal(err)
			}
			stubSkills(p, map[int32][]CharacterSkill{reconPilot: nil, carrierPilot: nil})
			stubLocations(p, tt.docked)
			stubImplants(p, func(int32) ([]int64, error) { return nil, nil })
			stubClones(p, func(id int32) (GetCharacterClonesOutput, error) {
				out := GetCharacterClonesOutput{CharacterID: id}
				if id == carrierPilot {
					out.JumpClones = []JumpClone{{LocationID: provenBay, LocationType: "station"}}
				}
				return out, nil
			})
			_, out, err := listJumpClonesHandler("test", fakeOpener(p), p.openPublicESI, store, p.sde)(context.Background(), nil, ListJumpClonesInput{CharacterIDs: []int32{reconPilot, carrierPilot}})
			if err != nil {
				t.Fatal(err)
			}
			c := out.Characters[0]
			if !c.CooldownReady || c.JumpFromHere.Status != tt.wantStatus || c.ReadyNow != tt.wantReady || c.JumpFromHere.Reason == "" {
				t.Errorf("cooldownReady %v readyNow %v jumpFromHere %+v; want status %s, readyNow %v", c.CooldownReady, c.ReadyNow, c.JumpFromHere, tt.wantStatus, tt.wantReady)
			}
		})
	}
}

// A jump clone of a character the call didn't ask about still proves a
// clone bay where the asked-about character is docked.
func TestListJumpClones_UnaskedCharacterProvesCloneBay(t *testing.T) {
	const provenBay = int64(60003760)
	store, p := readinessEnv(t)
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: nil, carrierPilot: nil})
	stubLocations(p, map[int32]int64{reconPilot: provenBay})
	stubImplants(p, func(int32) ([]int64, error) { return nil, nil })
	stubClones(p, func(id int32) (GetCharacterClonesOutput, error) {
		out := GetCharacterClonesOutput{CharacterID: id}
		if id == carrierPilot {
			out.JumpClones = []JumpClone{{LocationID: provenBay, LocationType: "station"}}
		}
		return out, nil
	})

	_, out, err := listJumpClonesHandler("test", fakeOpener(p), p.openPublicESI, store, p.sde)(context.Background(), nil, ListJumpClonesInput{CharacterIDs: []int32{reconPilot}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Characters) != 1 || out.Characters[0].JumpFromHere.Status != jumpFromYes || !out.Characters[0].ReadyNow {
		t.Errorf("characters = %+v, want only the recon pilot, ready to jump from a bay the carrier pilot's clone proves", out.Characters)
	}
}
