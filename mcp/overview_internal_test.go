package mcp

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
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
	loggedIn := time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC) // well before the cache window, so no re-check
	getCharacterOnline = func(_ context.Context, _ string, id int32, _ string) (*esi.CharacterOnlineStatus, error) {
		return &esi.CharacterOnlineStatus{Online: true, LastLogin: &loggedIn}, nil
	}
	stubSleep(t, func(time.Duration) { t.Error("re-checked online status for logins older than the cache window") })
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

// stubSleep replaces sleepCtx so tests never wait, calling f with each wait.
func stubSleep(t *testing.T, f func(time.Duration)) {
	t.Helper()
	orig := sleepCtx
	t.Cleanup(func() { sleepCtx = orig })
	sleepCtx = func(_ context.Context, d time.Duration) error { f(d); return nil }
}

func TestConfirmOnlineConflicts(t *testing.T) {
	origO, origN := getCharacterOnline, now
	t.Cleanup(func() { getCharacterOnline, now = origO, origN })
	at := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return at }
	old, recent := at.Add(-10*time.Minute), at.Add(-20*time.Second)
	a, b := config.Character{ID: 1, Name: "Alt A"}, config.Character{ID: 2, Name: "Alt B"}
	suspect := map[string][]config.Character{"Main": {a, b}}

	tests := []struct {
		name      string
		logins    [2]*time.Time
		recheck   map[int32]bool // online on the re-check; absent fails it
		wantWait  time.Duration
		wantNames []string
	}{
		{name: "old logins stand without a re-check", logins: [2]*time.Time{&old, &old}, wantNames: []string{"Alt A", "Alt B"}},
		{name: "recent switch cleared by the re-check", logins: [2]*time.Time{&old, &recent}, recheck: map[int32]bool{1: false, 2: true}, wantWait: onlineCacheWindow - 20*time.Second},
		{name: "recent login still online together", logins: [2]*time.Time{&old, &recent}, recheck: map[int32]bool{1: true, 2: true}, wantWait: onlineCacheWindow - 20*time.Second, wantNames: []string{"Alt A", "Alt B"}},
		{name: "unknown login time waits the full window", logins: [2]*time.Time{nil, &old}, recheck: map[int32]bool{1: true, 2: true}, wantWait: onlineCacheWindow, wantNames: []string{"Alt A", "Alt B"}},
		{name: "a failed re-check isn't reported", logins: [2]*time.Time{&recent, &recent}, recheck: map[int32]bool{2: true}, wantWait: onlineCacheWindow - 20*time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			online := map[int32]*esi.CharacterOnlineStatus{1: {Online: true, LastLogin: tt.logins[0]}, 2: {Online: true, LastLogin: tt.logins[1]}}
			var waited time.Duration
			stubSleep(t, func(d time.Duration) { waited = d })
			getCharacterOnline = func(_ context.Context, _ string, id int32, _ string) (*esi.CharacterOnlineStatus, error) {
				on, ok := tt.recheck[id]
				if !ok {
					return nil, errors.New("ESI unavailable")
				}
				return &esi.CharacterOnlineStatus{Online: on}, nil
			}
			got, err := confirmOnlineConflicts(context.Background(), "client", "ua", suspect, online)
			if err != nil {
				t.Fatal(err)
			}
			if waited != tt.wantWait {
				t.Errorf("waited %v, want %v", waited, tt.wantWait)
			}
			if !slices.Equal(got["Main"], tt.wantNames) {
				t.Errorf("confirmed = %v, want %v", got["Main"], tt.wantNames)
			}
		})
	}
}

func TestConfirmOnlineConflicts_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	online := map[int32]*esi.CharacterOnlineStatus{1: {Online: true}, 2: {Online: true}}
	suspect := map[string][]config.Character{"Main": {{ID: 1}, {ID: 2}}}
	if _, err := confirmOnlineConflicts(ctx, "client", "ua", suspect, online); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
