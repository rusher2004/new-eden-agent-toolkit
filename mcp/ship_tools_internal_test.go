package mcp

import (
	"context"
	"math"
	"slices"
	"strings"
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

func TestEstimateShipHold_LoadedChargeIsNotAModule(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(t)
	base := listAssets
	listAssets = func(ctx context.Context, v string, c int32, cl string, page int32) ([]AssetItem, int32, error) {
		items, pages, err := base(ctx, v, c, cl, page)
		// A charge loaded in a module carries the module's slot flag.
		return append(items, AssetItem{ItemID: 504, TypeID: 222, LocationID: 500, LocationFlag: "LoSlot0", Quantity: 100}), pages, err
	}
	stubSkills(t, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	_, out, err := estimateShipHoldHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipItemID: 500}, PilotID: reconPilot})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(out.Modules, "Antimatter Charge S") || len(out.Modules) != 2 {
		t.Errorf("modules = %v, want only the two Expanded Cargohold IIs", out.Modules)
	}
}

func TestEstimateShipHold_ShipItemNeedsOwner(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(t)
	_, _, err := estimateShipHoldHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipItemID: 500}})
	if err == nil || !strings.Contains(err.Error(), "ownerId") {
		t.Errorf("err = %v, want one asking for ownerId or a pilot", err)
	}
}

func TestSaveHaulProfile_ShipMustMatchShipItem(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(t)
	stubSkills(t, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	save := saveHaulProfileHandler("test", "client", store, p.sde)
	if _, _, err := save(context.Background(), nil, SaveHaulProfileInput{Name: "Wrong hull", Ship: "Rhea", PilotID: int64(reconPilot), ShipItemID: 500}); err == nil || !strings.Contains(err.Error(), "Ark") {
		t.Errorf("Rhea named for an Ark: err = %v, want a mismatch error", err)
	}
	if _, prof, err := save(context.Background(), nil, SaveHaulProfileInput{Name: "Right hull", Ship: "ark", PilotID: int64(reconPilot), ShipItemID: 500}); err != nil || prof.ShipTypeID != 28850 {
		t.Errorf("matching name in other case: prof %+v err %v", prof, err)
	}
}

// A cyno Venture loaded with its own Liquid Ozone (issue #37): the fuel
// aboard isn't cargo in the way, so a full load still covers a light, and
// the refills start from what's aboard.
func TestEstimateCyno_OwnedVentureCountsFuelAboard(t *testing.T) {
	store, p := shipToolsEnv(t)
	origA, origN := listAssets, getAssetNames
	t.Cleanup(func() { listAssets, getAssetNames = origA, origN })
	ozone := int64(242)
	listAssets = func(context.Context, string, int32, string, int32) ([]AssetItem, int32, error) {
		return []AssetItem{
			{ItemID: 600, TypeID: 32880, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},
			{ItemID: 601, TypeID: 1319, LocationID: 600, LocationFlag: "LoSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 602, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 603, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot1", Quantity: 1, IsSingleton: true},
			{ItemID: 604, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot2", Quantity: 1, IsSingleton: true},
			{ItemID: 605, TypeID: 52694, LocationID: 600, LocationFlag: "HiSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 606, TypeID: 16273, LocationID: 600, LocationFlag: "Cargo", Quantity: ozone},
		}, 1, nil
	}
	getAssetNames = func(context.Context, string, int32, string, []int64) ([]AssetName, error) { return nil, nil }
	stubSkills(t, map[int32][]CharacterSkill{reconPilot: {{SkillID: 21603, ActiveSkillLevel: 5}}})

	_, out, err := estimateCynoHandler("test", "client", store, p.sde)(context.Background(), nil, EstimateCynoInput{ShipSpec: ShipSpec{ShipItemID: 600}, LighterID: reconPilot, Cyno: "industrial", Lights: 2})
	if err != nil {
		t.Fatal(err)
	}
	hold := 50 * 1.275 * math.Pow(1.15, 3)
	if math.Abs(out.CargoM3-(hold-96.8)) > 1e-6 || math.Abs(out.FuelSpaceM3-hold) > 1e-6 {
		t.Errorf("cargo %v fuel space %v, want %v free and the whole %v hold for fuel", out.CargoM3, out.FuelSpaceM3, hold-96.8, hold)
	}
	if out.FuelAboard != ozone || out.LightsAboard != 1 || out.LightsPerHold != 1 || out.Refills != 1 || out.FuelDoesntFit {
		t.Errorf("aboard %d covers %d, per hold %d, refills %d, doesn't fit %v; want 242 covering 1, 1 per hold, 1 refill",
			out.FuelAboard, out.LightsAboard, out.LightsPerHold, out.Refills, out.FuelDoesntFit)
	}
}
