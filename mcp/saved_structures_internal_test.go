package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func stubStructureESI(t *testing.T, docking map[int32]bool, clones map[int32][]int64) {
	t.Helper()
	origS, origC, origN, origT := getUniverseStructure, getCharacterClones, resolveUniverseNames, today
	t.Cleanup(func() {
		getUniverseStructure, getCharacterClones, resolveUniverseNames, today = origS, origC, origN, origT
	})
	today = func() string { return "2026-11-20" }
	getUniverseStructure = func(_ context.Context, _ string, characterID int32, _ string, id int64) (UniverseStructure, error) {
		if !docking[characterID] {
			return UniverseStructure{}, errors.New("403 Forbidden")
		}
		typ := int64(35827)
		return UniverseStructure{Name: "CX7-70 - Sugar Daddy's Stronghold", OwnerID: 98435559, SolarSystemID: 30001324, TypeID: &typ}, nil
	}
	getCharacterClones = func(_ context.Context, _ string, characterID int32, _ string) (GetCharacterClonesOutput, error) {
		locs, ok := clones[characterID]
		if !ok {
			return GetCharacterClonesOutput{}, errors.New("missing scope")
		}
		out := GetCharacterClonesOutput{CharacterID: characterID}
		for _, l := range locs {
			out.JumpClones = append(out.JumpClones, JumpClone{LocationID: l})
		}
		return out, nil
	}
	resolveUniverseNames = func(_ context.Context, _ string, ids []int64) ([]ResolvedName, error) {
		return []ResolvedName{{Category: "corporation", ID: ids[0], Name: "The Blue Donut"}}, nil
	}
}

const sotiyo = int64(1049345291823)

func TestSaveStructure_UpwellFromESIWithUserFacts(t *testing.T) {
	store, p := industryTestEnv(t)
	stubStructureESI(t, map[int32]bool{testPilot: true}, nil)
	save := saveStructureHandler("test", "client", store, p.sde)

	if _, _, err := save(context.Background(), nil, SaveStructureInput{ID: sotiyo}); err == nil || !strings.Contains(err.Error(), "needs characterId") {
		t.Errorf("an Upwell structure without a character: err = %v", err)
	}
	_, out, err := save(context.Background(), nil, SaveStructureInput{ID: sotiyo, CharacterID: testPilot, Uses: []string{"manufacturing"}, CloneBay: "yes", Notes: ptr("XL rig, measured fee 6.7%")})
	if err != nil {
		t.Fatal(err)
	}
	s := out.Structure
	if s.Kind != config.StructureUpwell || s.Name != "CX7-70 - Sugar Daddy's Stronghold" || s.Owner.Name != "The Blue Donut" || s.System.Name != "PF-QHK" || s.Type.Name != "Sotiyo" {
		t.Errorf("structure = %+v type %+v system %+v owner %+v", s, s.Type, s.System, s.Owner)
	}
	if *s.CloneBay != (config.CloneBay{Status: "yes", Source: "user", Checked: "2026-11-20"}) || s.Notes != "XL rig, measured fee 6.7%" {
		t.Errorf("user facts = %+v %q", s.CloneBay, s.Notes)
	}
	if len(out.Changes) != 1 || out.Changes[0].New != "added" {
		t.Errorf("changes = %+v", out.Changes)
	}

	// Updating later without a character keeps the game facts and the notes.
	_, out, err = save(context.Background(), nil, SaveStructureInput{ID: sotiyo, Uses: []string{"manufacturing", "staging"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Structure.Name == "" || out.Structure.Notes != "XL rig, measured fee 6.7%" || len(out.Warnings) != 1 {
		t.Errorf("update = %+v warnings %v", out.Structure, out.Warnings)
	}
	if len(out.Changes) != 1 || out.Changes[0].Field != "uses" {
		t.Errorf("changes = %+v", out.Changes)
	}
}

func TestSaveStructure_NPCStationFromSDE(t *testing.T) {
	store, p := industryTestEnv(t)
	stubStructureESI(t, nil, nil)
	_, out, err := saveStructureHandler("test", "client", store, p.sde)(context.Background(), nil, SaveStructureInput{ID: 60003760, Uses: []string{"market"}})
	if err != nil {
		t.Fatal(err)
	}
	if s := out.Structure; s.Kind != config.StructureNPCStation || s.System.Name != "Jita" || !strings.HasPrefix(s.Name, "Jita 4") {
		t.Errorf("station = %+v system %+v", s, s.System)
	}
}

func TestSaveStructure_RejectsBadInput(t *testing.T) {
	store, p := industryTestEnv(t)
	stubStructureESI(t, nil, nil)
	save := saveStructureHandler("test", "client", store, p.sde)
	for name, in := range map[string]SaveStructureInput{
		"no id":             {},
		"bad clone bay":     {ID: 60003760, CloneBay: "maybe"},
		"unknown character": {ID: sotiyo, CharacterID: 42},
		"unknown facility":  {ID: 60003760, FacilityProfile: ptr("Nope")},
		"no docking access": {ID: sotiyo, CharacterID: testPilot},
	} {
		if _, _, err := save(context.Background(), nil, in); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestListStructures_ProvesCloneBaysAndChecksDocking(t *testing.T) {
	store, p := industryTestEnv(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Characters = append(cfg.Characters, config.Character{ID: 7, Name: "No Scope"})
	cfg.UpsertStructure(config.Structure{ID: sotiyo, Name: "Sotiyo", Kind: config.StructureUpwell, CloneBay: &config.CloneBay{Status: "no", Source: "user", Checked: "2026-10-01"}})
	cfg.UpsertStructure(config.Structure{ID: 60003760, Name: "Jita 4-4", Kind: config.StructureNPCStation})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	stubStructureESI(t, map[int32]bool{testPilot: true}, map[int32][]int64{testPilot: {sotiyo}})

	_, out, err := listStructuresHandler("test", "client", store, p.sde)(context.Background(), nil, ListStructuresInput{CharacterIDs: []int32{testPilot, 7}})
	if err != nil {
		t.Fatal(err)
	}
	if bay := out.Structures[0].CloneBay; bay == nil || *bay != (config.CloneBay{Status: "yes", Source: "jump_clone", Checked: "2026-11-20"}) {
		t.Errorf("clone bay = %+v, want proved by the jump clone", bay)
	}
	if len(out.Changes) == 0 || out.Changes[0].Field != "clone_bay.status" || out.Changes[0].Old != "no" || out.Changes[0].New != "yes" {
		t.Errorf("changes = %+v, want the user's 'no' overridden and reported", out.Changes)
	}
	if out.Structures[1].CloneBay != nil {
		t.Errorf("Jita clone bay = %+v, want still unknown", out.Structures[1].CloneBay)
	}
	if d := out.Structures[0].Docking; len(d) != 2 || d[0].Access != "yes" || d[1].Access != "unknown" || !strings.Contains(d[1].Detail, "403") {
		t.Errorf("docking = %+v", d)
	}
	if out.Structures[1].Docking != nil {
		t.Errorf("an NPC station got docking checks: %+v", out.Structures[1].Docking)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "No Scope") {
		t.Errorf("warnings = %v", out.Warnings)
	}

	_, again, err := listStructuresHandler("test", "client", store, p.sde)(context.Background(), nil, ListStructuresInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Changes) != 0 {
		t.Errorf("a second call changed things again: %+v", again.Changes)
	}
}

func TestDeleteStructure(t *testing.T) {
	store, _ := industryTestEnv(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.UpsertStructure(config.Structure{ID: sotiyo, Name: "Sotiyo", Kind: config.StructureUpwell})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	_, out, err := deleteStructureHandler(store)(context.Background(), nil, DeleteStructureInput{ID: sotiyo})
	if err != nil || out.Deleted != "Sotiyo" {
		t.Errorf("out = %+v, %v", out, err)
	}
	if _, _, err := deleteStructureHandler(store)(context.Background(), nil, DeleteStructureInput{ID: sotiyo}); err == nil {
		t.Error("deleting twice: want an error")
	}
}

func TestListStructures_KeepsWritesMadeDuringLookups(t *testing.T) {
	store, p := industryTestEnv(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.UpsertStructure(config.Structure{ID: sotiyo, Name: "Sotiyo", Kind: config.StructureUpwell})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	stubStructureESI(t, nil, map[int32][]int64{testPilot: {sotiyo}})
	clones := getCharacterClones
	getCharacterClones = func(ctx context.Context, clientID string, characterID int32, ua string) (GetCharacterClonesOutput, error) {
		// Another tool saves a structure while this call waits on ESI.
		c, err := store.Load()
		if err != nil {
			t.Fatal(err)
		}
		c.UpsertStructure(config.Structure{ID: 60003760, Name: "Jita 4-4", Kind: config.StructureNPCStation})
		if err := store.Save(c); err != nil {
			t.Fatal(err)
		}
		return clones(ctx, clientID, characterID, ua)
	}
	if _, _, err := listStructuresHandler("test", "client", store, p.sde)(context.Background(), nil, ListStructuresInput{}); err != nil {
		t.Fatal(err)
	}
	after, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := after.Structure(60003760); !ok {
		t.Error("list_structures saved over a structure added while it read jump clones")
	}
	if s, _ := after.Structure(sotiyo); s.CloneBayStatus() != config.CloneBayYes {
		t.Errorf("clone bay = %q, want yes", s.CloneBayStatus())
	}
}

func TestListStructures_StopsDockingChecksAfterRefusals(t *testing.T) {
	store, p := industryTestEnv(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	for i := range maxDockingRefusals + 5 {
		cfg.UpsertStructure(config.Structure{ID: sotiyo + int64(i), Name: "Upwell", Kind: config.StructureUpwell})
	}
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	stubStructureESI(t, nil, map[int32][]int64{testPilot: nil})
	calls := 0
	getUniverseStructure = func(context.Context, string, int32, string, int64) (UniverseStructure, error) {
		calls++
		return UniverseStructure{}, errors.New("403 Forbidden")
	}
	_, out, err := listStructuresHandler("test", "client", store, p.sde)(context.Background(), nil, ListStructuresInput{CharacterIDs: []int32{testPilot}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != maxDockingRefusals {
		t.Errorf("ESI calls = %d, want %d", calls, maxDockingRefusals)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "error limit") {
		t.Errorf("warnings = %v", out.Warnings)
	}
	var unknown, notChecked int
	for _, v := range out.Structures {
		for _, d := range v.Docking {
			switch d.Access {
			case "unknown":
				unknown++
			case "not_checked":
				notChecked++
			}
		}
	}
	if unknown != maxDockingRefusals || notChecked != 5 {
		t.Errorf("unknown = %d, not_checked = %d, want %d and 5", unknown, notChecked, maxDockingRefusals)
	}
}

func TestSaveStructure_ConcurrentSavesKeepEveryStructure(t *testing.T) {
	store, p := industryTestEnv(t)
	stubStructureESI(t, map[int32]bool{testPilot: true}, nil)
	save := saveStructureHandler("test", "client", store, p.sde)
	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := save(context.Background(), nil, SaveStructureInput{ID: sotiyo + int64(i), CharacterID: testPilot, Notes: ptr(fmt.Sprintf("structure %d", i))})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Structures) != n {
		t.Errorf("saved %d structures after %d concurrent saves, want %d", len(cfg.Structures), n, n)
	}
}
