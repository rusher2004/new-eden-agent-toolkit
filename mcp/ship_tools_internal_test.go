package mcp

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
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
	openESI, openPublicESI := noESI()
	return store, newPlanner("test", openESI, openPublicESI, store, s)
}

func stubOwnedArk(p *planner) {
	stubAssets(p, func(int32) ([]AssetItem, error) {
		return []AssetItem{
			{ItemID: 500, TypeID: 28850, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},
			{ItemID: 501, TypeID: 1319, LocationID: 500, LocationFlag: "LoSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 502, TypeID: 1319, LocationID: 500, LocationFlag: "LoSlot1", Quantity: 1, IsSingleton: true},
			{ItemID: 503, TypeID: 16273, LocationID: 500, LocationFlag: "Cargo", Quantity: 100},
		}, nil
	})
	stubAssetNames(p, func(int32, []int64) ([]AssetName, error) {
		return []AssetName{{ItemID: 500, Name: "Big Box"}}, nil
	})
}

func TestEstimateShipHold_OwnedArkWithPilotSkills(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(p)
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	_, out, err := estimateShipHoldHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipItemID: 500}, PilotID: reconPilot})
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
	_, out, err := estimateShipHoldHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipType: "Venture",
		Modules: []string{"Expanded Cargohold II", "Small Cargohold Optimization I", "Small Cargohold Optimization I", "Small Cargohold Optimization I"}}})
	if err != nil {
		t.Fatal(err)
	}
	if want := 50 * 1.275 * math.Pow(1.15, 3); math.Abs(out.Holds[0].FittedM3-want) > 1e-6 || out.Holds[0].BaseM3 != 50 {
		t.Errorf("Venture holds = %+v, want %v", out.Holds, want)
	}
	if _, _, err := estimateShipHoldHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipType: "Venture", Modules: []string{"Nope"}}}); err == nil {
		t.Error("an unknown module: want an error")
	}
}

func TestEstimateCyno_VentureIndustrial(t *testing.T) {
	store, p := shipToolsEnv(t)
	fit := ShipSpec{ShipType: "Venture", Modules: []string{"Expanded Cargohold II", "Small Cargohold Optimization I", "Small Cargohold Optimization I", "Small Cargohold Optimization I"}}
	for _, tt := range []struct {
		cft                int64
		perLight           float64
		perHold, fillsFor3 int
	}{
		{4, 240, 1, 3},
		{5, 200, 1, 3},
	} {
		stubSkills(p, map[int32][]CharacterSkill{reconPilot: {{SkillID: 21603, ActiveSkillLevel: tt.cft}}})
		_, out, err := estimateCynoHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateCynoInput{ShipSpec: fit, LighterID: reconPilot, Cyno: "industrial", Lights: 3})
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(out.FuelPerLight-tt.perLight) > 1e-9 || out.CycleSeconds != 600 || out.Fuel != "Liquid Ozone" || math.Abs(out.FuelM3PerLight-tt.perLight*0.4) > 1e-9 {
			t.Errorf("CFT %d: %+v", tt.cft, out)
		}
		if out.LightsPerHold != tt.perHold || out.Fills != tt.fillsFor3 || out.FuelForLights != 3*tt.perLight || out.GeneratorNotFits || out.MissingSkills != nil {
			t.Errorf("CFT %d: lights per hold %d fills %d fuel %v fits %v missing %v", tt.cft, out.LightsPerHold, out.Fills, out.FuelForLights, !out.GeneratorNotFits, out.MissingSkills)
		}
	}
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: nil})
	_, out, err := estimateCynoHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateCynoInput{ShipSpec: ShipSpec{ShipType: "Ark"}, LighterID: reconPilot, Cyno: "industrial"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.GeneratorNotFits || !slices.Equal(out.MissingSkills, []string{"Cynosural Field Theory I"}) {
		t.Errorf("Ark, untrained = %+v", out)
	}
}

func TestSaveHaulProfile_HoldFromShipItem(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(p)
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	_, prof, err := saveHaulProfileHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, SaveHaulProfileInput{Name: "Ark run", PilotID: int64(reconPilot), ShipItemID: 500})
	if err != nil {
		t.Fatal(err)
	}
	if prof.ShipTypeID != 28850 || math.Abs(prof.HoldM3-263351.25) > 1e-6 {
		t.Errorf("profile = %+v", prof)
	}
}

func TestEstimateShipHold_LoadedChargeIsNotAModule(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(p)
	addESILayer(p, func(_ int32, f *fakeESI) {
		base := f.assets
		f.assets = func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
			items, pages, err := base(ctx, page)
			// A charge loaded in a module carries the module's slot flag.
			return append(items, esiapi.CharactersCharacterIdAssetsGetInner{ItemId: 504, TypeId: 222, LocationId: 500, LocationFlag: "LoSlot0", Quantity: 100}), pages, err
		}
	})
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	_, out, err := estimateShipHoldHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipItemID: 500}, PilotID: reconPilot})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(out.Modules, "Antimatter Charge S") || len(out.Modules) != 2 {
		t.Errorf("modules = %v, want only the two Expanded Cargohold IIs", out.Modules)
	}
}

func TestEstimateShipHold_UnopenableClient(t *testing.T) {
	store, p := shipToolsEnv(t)
	p.openESI = failOpenESI(errors.New("no refresh token"))
	_, _, err := estimateShipHoldHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipItemID: 500}, PilotID: reconPilot})
	if err == nil || !strings.Contains(err.Error(), "read the owner's assets: no refresh token") {
		t.Errorf("err = %v, want the open error", err)
	}
}

func TestEstimateShipHold_ShipItemNeedsOwner(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(p)
	_, _, err := estimateShipHoldHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateShipHoldInput{ShipSpec: ShipSpec{ShipItemID: 500}})
	if err == nil || !strings.Contains(err.Error(), "ownerId") {
		t.Errorf("err = %v, want one asking for ownerId or a pilot", err)
	}
}

func TestSaveHaulProfile_ShipMustMatchShipItem(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubOwnedArk(p)
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: {{SkillID: 20524, ActiveSkillLevel: 4}}})
	save := saveHaulProfileHandler("test", p.openESI, p.openPublicESI, store, p.sde)
	if _, _, err := save(context.Background(), nil, SaveHaulProfileInput{Name: "Wrong hull", Ship: "Rhea", PilotID: int64(reconPilot), ShipItemID: 500}); err == nil || !strings.Contains(err.Error(), "Ark") {
		t.Errorf("Rhea named for an Ark: err = %v, want a mismatch error", err)
	}
	if _, prof, err := save(context.Background(), nil, SaveHaulProfileInput{Name: "Right hull", Ship: "ark", PilotID: int64(reconPilot), ShipItemID: 500}); err != nil || prof.ShipTypeID != 28850 {
		t.Errorf("matching name in other case: prof %+v err %v", prof, err)
	}
}

// A cyno Venture loaded with its own Liquid Ozone (issue #37): the fuel
// aboard isn't cargo in the way, so a full load still covers a light, and
// the fills start from what's aboard.
func TestEstimateCyno_OwnedVentureCountsFuelAboard(t *testing.T) {
	store, p := shipToolsEnv(t)
	ozone := int64(242)
	stubAssets(p, func(int32) ([]AssetItem, error) {
		return []AssetItem{
			{ItemID: 600, TypeID: 32880, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},
			{ItemID: 601, TypeID: 1319, LocationID: 600, LocationFlag: "LoSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 602, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 603, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot1", Quantity: 1, IsSingleton: true},
			{ItemID: 604, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot2", Quantity: 1, IsSingleton: true},
			{ItemID: 605, TypeID: 52694, LocationID: 600, LocationFlag: "HiSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 606, TypeID: 16273, LocationID: 600, LocationFlag: "Cargo", Quantity: ozone},
		}, nil
	})
	stubAssetNames(p, func(int32, []int64) ([]AssetName, error) { return nil, nil })
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: {{SkillID: 21603, ActiveSkillLevel: 5}}})

	_, out, err := estimateCynoHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, EstimateCynoInput{ShipSpec: ShipSpec{ShipItemID: 600}, LighterID: reconPilot, Cyno: "industrial", Lights: 2})
	if err != nil {
		t.Fatal(err)
	}
	hold := 50 * 1.275 * math.Pow(1.15, 3)
	if math.Abs(out.CargoM3-(hold-96.8)) > 1e-6 || math.Abs(out.FuelSpaceM3-hold) > 1e-6 {
		t.Errorf("cargo %v fuel space %v, want %v free and the whole %v hold for fuel", out.CargoM3, out.FuelSpaceM3, hold-96.8, hold)
	}
	if out.FuelAboard != ozone || out.LightsAboard != 1 || out.LightsPerHold != 1 || out.Fills != 1 || out.FuelDoesntFit {
		t.Errorf("aboard %d covers %d, per hold %d, fills %d, doesn't fit %v; want 242 covering 1, 1 per hold, 1 fill",
			out.FuelAboard, out.LightsAboard, out.LightsPerHold, out.Fills, out.FuelDoesntFit)
	}
	if len(out.Loads) != 1 || out.Loads[0] != (nav.CynoFuelLoad{BeforeLight: 2, AddUnits: 200*2 - ozone, FuelAfter: 200, Lights: 1}) {
		t.Errorf("loads = %+v, want 158 more before the second light", out.Loads)
	}
}

// An owned cyno Venture with no fuel aboard: two lights at one light per
// load take two fills, the first load included, same as the hull by name.
func TestEstimateCyno_OwnedEmptyHoldCountsFirstFill(t *testing.T) {
	store, p := shipToolsEnv(t)
	stubAssets(p, func(int32) ([]AssetItem, error) {
		return []AssetItem{
			{ItemID: 600, TypeID: 32880, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},
			{ItemID: 601, TypeID: 1319, LocationID: 600, LocationFlag: "LoSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 602, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot0", Quantity: 1, IsSingleton: true},
			{ItemID: 603, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot1", Quantity: 1, IsSingleton: true},
			{ItemID: 604, TypeID: 31117, LocationID: 600, LocationFlag: "RigSlot2", Quantity: 1, IsSingleton: true},
			{ItemID: 605, TypeID: 52694, LocationID: 600, LocationFlag: "HiSlot0", Quantity: 1, IsSingleton: true},
		}, nil
	})
	stubAssetNames(p, func(int32, []int64) ([]AssetName, error) { return nil, nil })
	stubSkills(p, map[int32][]CharacterSkill{reconPilot: {{SkillID: 21603, ActiveSkillLevel: 5}}})

	in := EstimateCynoInput{ShipSpec: ShipSpec{ShipItemID: 600}, LighterID: reconPilot, Cyno: "industrial", Lights: 2}
	_, owned, err := estimateCynoHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, in)
	if err != nil {
		t.Fatal(err)
	}
	in.ShipSpec = ShipSpec{ShipType: "Venture", Modules: []string{"Expanded Cargohold II", "Small Cargohold Optimization I", "Small Cargohold Optimization I", "Small Cargohold Optimization I"}}
	_, named, err := estimateCynoHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, in)
	if err != nil {
		t.Fatal(err)
	}
	if owned.FuelAboard != 0 || owned.LightsAboard != 0 || owned.LightsPerHold != 1 || owned.Fills != 2 {
		t.Errorf("owned: aboard %d covers %d, per hold %d, fills %d; want nothing aboard, 1 per hold, 2 fills", owned.FuelAboard, owned.LightsAboard, owned.LightsPerHold, owned.Fills)
	}
	if named.Fills != owned.Fills {
		t.Errorf("by name: %d fills, want the owned ship's %d", named.Fills, owned.Fills)
	}
}
