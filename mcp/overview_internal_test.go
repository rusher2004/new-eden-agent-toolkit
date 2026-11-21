package mcp

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCompareHaulHulls_FromSDE(t *testing.T) {
	store, p := readinessEnv(t)
	stubSkills(t, map[int32][]CharacterSkill{carrierPilot: requirementsOf(t, p, map[int64]int64{21611: 5, 21610: 4}, 23757)})
	_, out, err := compareHaulHullsHandler("test", "client", store, p.sde)(context.Background(), nil, CompareHaulHullsInput{Hulls: []string{"Nomad", "Archon", "Badger"}, PilotID: carrierPilot})
	if err != nil {
		t.Fatal(err)
	}
	nomad, archon, badger := out.Hulls[0], out.Hulls[1], out.Hulls[2]
	if nomad.WarpSpeedAU != 1.5 || !nomad.EntersHighsec || nomad.Jump == nil || nomad.Jump.FatigueMultiplier != 0.1 || nomad.HoldsM3["fuel_bay"] != 12000 || nomad.FuelName == "" {
		t.Errorf("Nomad = %+v jump %+v holds %v", nomad.HullFacts, nomad.Jump, nomad.HoldsM3)
	}
	if !slices.Equal(nomad.Jump.CynoTypes, []string{"standard", "industrial"}) || *nomad.PilotCanFly {
		t.Errorf("Nomad cynos %v, pilot can fly %v", nomad.Jump.CynoTypes, *nomad.PilotCanFly)
	}
	if archon.EntersHighsec || archon.Jump == nil || archon.Jump.FatigueMultiplier != 1 || archon.HoldsM3["fleet_hangar"] == 0 || !*archon.PilotCanFly {
		t.Errorf("Archon = %+v holds %v", archon.HullFacts, archon.HoldsM3)
	}
	if archon.PilotRangeLY != archon.Jump.BaseRangeLY*2 {
		t.Errorf("Archon pilot range = %v, want double the base at Jump Drive Calibration V", archon.PilotRangeLY)
	}
	if slices.Contains(archon.Docking.Upwell, "Astrahus") || !archon.Docking.NPCStations {
		t.Errorf("Archon docking = %+v", archon.Docking)
	}
	if badger.Nullification != "module" || badger.CovertOpsCloak || badger.Jump != nil || badger.HoldsM3["cargo"] == 0 || badger.AlignSeconds <= 0 {
		t.Errorf("Badger = %+v holds %v", badger.HullFacts, badger.HoldsM3)
	}
}

func TestCompareHaulHulls_ClassesAndErrors(t *testing.T) {
	store, p := readinessEnv(t)
	h := compareHaulHullsHandler("test", "client", store, p.sde)
	_, out, err := h(context.Background(), nil, CompareHaulHullsInput{Classes: []string{"jump freighter"}})
	if err != nil || len(out.Hulls) != 1 || out.Hulls[0].Name != "Nomad" || out.Hulls[0].PilotCanFly != nil {
		t.Errorf("class = %+v, %v", out.Hulls, err)
	}
	if _, _, err := h(context.Background(), nil, CompareHaulHullsInput{Classes: []string{"Gunboat"}}); err == nil || !strings.Contains(err.Error(), "known:") {
		t.Errorf("unknown class err = %v", err)
	}
	if _, _, err := h(context.Background(), nil, CompareHaulHullsInput{Hulls: []string{"Tritanium"}}); err == nil || !strings.Contains(err.Error(), "isn't a ship") {
		t.Errorf("not a ship err = %v", err)
	}
}

func TestGetLogisticsOverview(t *testing.T) {
	store, p := readinessEnv(t)
	cfg, _ := store.Load()
	cfg.Characters[1].Account = "Main" // both labelled Main, but online together below
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	stubSkills(t, map[int32][]CharacterSkill{
		reconPilot: requirementsOf(t, p, map[int64]int64{21603: 4}, 11957, 21096),
	})
	origL, origS, origO, origC, origF, origN := getCharacterLocation, getCharacterShip, getCharacterOnline, getCharacterClones, getCharacterFatigue, now
	t.Cleanup(func() {
		getCharacterLocation, getCharacterShip, getCharacterOnline, getCharacterClones, getCharacterFatigue, now = origL, origS, origO, origC, origF, origN
	})
	now = func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) }
	station := int64(60003760)
	getCharacterLocation = func(_ context.Context, _ string, id int32, _ string) (int64, *int64, *int64, error) {
		if id == carrierPilot {
			return 0, nil, nil, errors.New("missing scope esi-location.read_location.v1")
		}
		return 30000142, &station, nil, nil
	}
	getCharacterShip = func(_ context.Context, _ string, id int32, _ string) (int64, string, int64, error) {
		return 1000 + int64(id), "Cyno Falcon", 11957, nil
	}
	getCharacterOnline = func(_ context.Context, _ string, id int32, _ string) (*esi.CharacterOnlineStatus, error) {
		return &esi.CharacterOnlineStatus{Online: true}, nil
	}
	expires := time.Date(2026, 11, 20, 13, 0, 0, 0, time.UTC)
	getCharacterFatigue = func(_ context.Context, _ string, id int32, _ string) (GetCharacterFatigueOutput, error) {
		return GetCharacterFatigueOutput{CharacterID: id, JumpFatigueExpireDate: &expires}, nil
	}
	getCharacterClones = func(_ context.Context, _ string, id int32, _ string) (GetCharacterClonesOutput, error) {
		return GetCharacterClonesOutput{CharacterID: id, JumpClones: []JumpClone{{LocationID: station}}}, nil
	}

	_, out, err := getLogisticsOverviewHandler("test", "client", store, p.sde)(context.Background(), nil, GetLogisticsOverviewInput{})
	if err != nil {
		t.Fatal(err)
	}
	recon, carrier := out.Characters[0], out.Characters[1]
	if recon.Location == nil || recon.Location.System != "Jita" || recon.Location.SecurityDisplay != "0.9" || !strings.HasPrefix(recon.Location.DockedAt, "Jita 4") {
		t.Errorf("location = %+v", recon.Location)
	}
	if recon.Ship == nil || recon.Ship.Type != "Falcon" || len(recon.Abilities) != 1 || recon.Abilities[0] != "standard cyno: Falcon" {
		t.Errorf("ship %+v abilities %v", recon.Ship, recon.Abilities)
	}
	if recon.Fatigue == nil || !recon.Fatigue.Fatigued || recon.Clones == nil || !recon.Clones.ReadyNow || len(recon.Clones.JumpClonesAt) != 1 {
		t.Errorf("fatigue %+v clones %+v", recon.Fatigue, recon.Clones)
	}
	if carrier.Location != nil || len(carrier.Unread) != 2 || !strings.Contains(strings.Join(carrier.Unread, ";"), "location: missing scope") || carrier.Clones == nil {
		t.Errorf("carrier unread = %v location %+v clones %+v", carrier.Unread, carrier.Location, carrier.Clones)
	}
	if len(out.Accounts) != 1 || len(out.Accounts[0].Characters) != 2 || len(out.Conflicts) != 1 || !strings.Contains(out.Conflicts[0], "online at the same time") {
		t.Errorf("accounts %+v conflicts %v", out.Accounts, out.Conflicts)
	}
}
