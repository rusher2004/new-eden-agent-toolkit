package mcp

import (
	"context"
	"math"
	"slices"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde/sdetest"
)

func shipToolsEnv(t *testing.T) (*config.Store, *planner) {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: reconPilot, Name: "Hauler"}}}); err != nil {
		t.Fatal(err)
	}
	s := sdetest.NewStore(t, "testdata/industry_fixture.sql", "testdata/readiness_fixture.sql", "testdata/dogma_fixture.sql")
	return store, newPlanner("test", "client", store, s)
}

func stubOwnedArk(t *testing.T) {
	t.Helper()
	origA, origN := listAssets, getAssetNames
	t.Cleanup(func() { listAssets, getAssetNames = origA, origN })
	listAssets = func(context.Context, string, int32, string, int32) ([]AssetItem, int32, error) {
		return []AssetItem{
			{ItemID: 500, TypeID: 28850, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},
			{ItemID: 501, TypeID: 1319, LocationID: 500, LocationFlag: "LoSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 502, TypeID: 1319, LocationID: 500, LocationFlag: "LoSlot1", Quantity: 1, IsSingleton: true},
			{ItemID: 503, TypeID: 16273, LocationID: 500, LocationFlag: "Cargo", Quantity: 100},
		}, 1, nil
	}
	getAssetNames = func(context.Context, string, int32, string, []int64) ([]AssetName, error) {
		return []AssetName{{ItemID: 500, Name: "Big Box"}}, nil
	}
}

func TestEstimateShipHold_OwnedArkWithPilotSkills(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(t)
	stubSkills(t, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	_, out, err := estimateShipHoldHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipItemID: 500}, PilotID: reconPilot})
	if err != nil {
		t.Fatal(err)
	}
	if out.Ship != "Ark" || out.ShipName != "Big Box" || len(out.Modules) != 2 || out.Holds[0].Hold != "cargo" || math.Abs(out.Holds[0].FittedM3-263351.25) > 1e-6 {
		t.Errorf("out = %+v holds %+v", out, out.Holds)
	}
	if math.Abs(out.CargoUsedM3-40) > 1e-9 || math.Abs(out.CargoFreeM3-(263351.25-40)) > 1e-6 {
		t.Errorf("used %v free %v, want 100 Liquid Ozone (40 m³) used", out.CargoUsedM3, out.CargoFreeM3)
	}
}

func TestEstimateShipHold_NamedFitUntrained(t *testing.T) {
	store, p := shipToolsEnv(t)
	_, out, err := estimateShipHoldHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipType: "Venture",
		Modules: []string{"Expanded Cargohold II", "Small Cargohold Optimization I", "Small Cargohold Optimization I", "Small Cargohold Optimization I"}}})
	if err != nil {
		t.Fatal(err)
	}
	if want := 50 * 1.275 * math.Pow(1.15, 3); math.Abs(out.Holds[0].FittedM3-want) > 1e-6 || out.Holds[0].BaseM3 != 50 {
		t.Errorf("Venture holds = %+v, want %v", out.Holds, want)
	}
	if _, _, err := estimateShipHoldHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipType: "Venture", Modules: []string{"Nope"}}}); err == nil {
		t.Error("an unknown module: want an error")
	}
}

func TestEstimateCyno_VentureIndustrial(t *testing.T) {
	store, p := shipToolsEnv(t)
	fit := ShipSpec{ShipType: "Venture", Modules: []string{"Expanded Cargohold II", "Small Cargohold Optimization I", "Small Cargohold Optimization I", "Small Cargohold Optimization I"}}
	for _, tt := range []struct {
		cft                  int64
		perLight             float64
		perHold, refillsFor3 int
	}{
		{4, 240, 1, 2},
		{5, 200, 1, 2},
	} {
		stubSkills(t, map[int32][]CharacterSkill{reconPilot: {{SkillID: 21603, ActiveSkillLevel: tt.cft}}})
		_, out, err := estimateCynoHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateCynoInput{ShipSpec: fit, LighterID: reconPilot, Cyno: "industrial", Lights: 3})
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(out.FuelPerLight-tt.perLight) > 1e-9 || out.CycleSeconds != 600 || out.Fuel != "Liquid Ozone" || math.Abs(out.FuelM3PerLight-tt.perLight*0.4) > 1e-9 {
			t.Errorf("CFT %d: %+v", tt.cft, out)
		}
		if out.LightsPerHold != tt.perHold || out.Refills != tt.refillsFor3 || out.FuelForLights != 3*tt.perLight || out.GeneratorNotFits || out.MissingSkills != nil {
			t.Errorf("CFT %d: lights per hold %d refills %d fuel %v fits %v missing %v", tt.cft, out.LightsPerHold, out.Refills, out.FuelForLights, !out.GeneratorNotFits, out.MissingSkills)
		}
	}
	stubSkills(t, map[int32][]CharacterSkill{reconPilot: nil})
	_, out, err := estimateCynoHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateCynoInput{ShipSpec: ShipSpec{ShipType: "Ark"}, LighterID: reconPilot, Cyno: "industrial"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.GeneratorNotFits || !slices.Equal(out.MissingSkills, []string{"Cynosural Field Theory I"}) {
		t.Errorf("Ark, untrained = %+v", out)
	}
}

func TestSaveHaulProfile_HoldFromShipItem(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(t)
	stubSkills(t, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	_, prof, err := saveHaulProfileHandler("test", "client", store, p.sde)(context.Background(), nil, SaveHaulProfileInput{Name: "Ark run", PilotID: int64(reconPilot), ShipItemID: 500})
	if err != nil {
		t.Fatal(err)
	}
	if prof.ShipTypeID != 28850 || math.Abs(prof.HoldM3-263351.25) > 1e-6 {
		t.Errorf("profile = %+v", prof)
	}
}
