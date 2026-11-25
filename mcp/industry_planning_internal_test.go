package mcp

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/sde/sdetest"
)

const (
	testPilot int32 = 90000001
	testCorp  int64 = 98000001
)

func industryTestEnv(t *testing.T) (*config.Store, *planner) {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: testPilot, Name: "Test Pilot"}}}); err != nil {
		t.Fatal(err)
	}
	sdeStore := sdetest.NewStore(t, "testdata/industry_fixture.sql")
	openESI, openPublicESI := noESI()
	return store, newPlanner("test", openESI, openPublicESI, store, sdeStore)
}

func TestSaveAndDescribeFacility_NullsecSotiyo(t *testing.T) {
	store, p := industryTestEnv(t)
	save := saveFacilityProfileHandler("test", p.openESI, p.openPublicESI, store, p.sde)
	_, view, err := save(context.Background(), nil, SaveFacilityProfileInput{
		Name: "Home Sotiyo", Kind: industry.KindUpwell, StructureType: "Sotiyo", System: "PF-QHK",
		Rigs:     []string{"Standup XL-Set Ship Manufacturing Efficiency I", "Standup XL-Set Structure and Component Manufacturing Efficiency I"},
		Services: []string{industry.SvcManufacturing, industry.SvcCapitalShipyard},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Band != "nullsec" || len(view.Modifiers) == 0 {
		t.Errorf("view = %+v", view)
	}
	describe := describeFacilityHandler("test", p.openESI, p.openPublicESI, store, p.sde)
	_, d, err := describe(context.Background(), nil, DescribeFacilityInput{Name: "home sotiyo", ProductName: "Capital Siege Array"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Multipliers == nil || math.Abs(d.Multipliers.Material-0.99*0.958) > 1e-9 || math.Abs(d.Multipliers.Time-0.7*0.58) > 1e-9 {
		t.Errorf("siege array multipliers = %+v", d.Multipliers)
	}
	if d.CanRun[industry.ActManufacturing] != "yes" {
		t.Errorf("can't manufacture: %v", d.CanRun)
	}
	_, _, err = save(context.Background(), nil, SaveFacilityProfileInput{Name: "x", Kind: industry.KindUpwell, StructureType: "Sotiyo", System: "PF-QHK", Services: []string{"teleporter"}})
	if err == nil || !strings.Contains(err.Error(), "unknown service") {
		t.Errorf("bad service accepted: %v", err)
	}
}

// fakeIndustryESI is the ESI the industry planning tests read: one owned
// 10-run copy, adjusted prices, a cost index, a builder's skills, and sell
// orders at Jita 4-4.
func fakeIndustryESI() fakeESI {
	price := func(id int64, p float64) esiapi.MarketsPricesGetInner {
		return esiapi.MarketsPricesGetInner{TypeId: id, AdjustedPrice: &p}
	}
	book := map[int64]float64{34: 4, 35: 16, 36: 55, 37: 206, 38: 780, 39: 1079, 40: 2094, 2876: 2.14e6, 57457: 3466, 62555: 2000, 17889: 590, 21039: 11.88e6}
	return fakeESI{
		fakeCorporationBlueprintsESI: fakeCorporationBlueprintsESI{corporationBlueprints: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
			return []esiapi.CorporationsCorporationIdBlueprintsGetInner{{ItemId: 1, TypeId: 21040, MaterialEfficiency: 10, TimeEfficiency: 20, Runs: 10, Quantity: -2, LocationId: 99, LocationFlag: "CorpSAG6"}}, 1, nil
		}},
		fakeCorporationIndustryJobsESI: fakeCorporationIndustryJobsESI{corporationIndustryJobs: func(context.Context, int64, bool, int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
			return nil, 1, nil
		}},
		fakeMarketPricesESI: fakeMarketPricesESI{marketPrices: func(context.Context) ([]esiapi.MarketsPricesGetInner, error) {
			return []esiapi.MarketsPricesGetInner{price(34, 4), price(35, 16), price(36, 55), price(37, 206), price(38, 780), price(39, 1079), price(40, 2094), price(2876, 2.1e6), price(57457, 3466)}, nil
		}},
		fakeIndustrySystemsESI: fakeIndustrySystemsESI{industrySystems: func(context.Context) ([]esiapi.IndustrySystemsGetInner, error) {
			return []esiapi.IndustrySystemsGetInner{{SolarSystemId: 30001324, CostIndices: []esiapi.IndustrySystemsGetInnerCostIndicesInner{{Activity: "manufacturing", CostIndex: 0.03}}}}, nil
		}},
		fakeCharacterSkillsESI: fakeCharacterSkillsESI{characterSkills: func(context.Context) (*esiapi.CharactersSkills, error) {
			lv := map[int64]int64{3380: 5, 3388: 5, 3387: 5, 24625: 4, 22242: 4, 3385: 5, 3389: 5, 60379: 4, 21611: 5, 21610: 5, 29029: 4}
			out := &esiapi.CharactersSkills{}
			for id, l := range lv {
				out.Skills = append(out.Skills, esiapi.CharactersSkillsSkill{SkillId: id, ActiveSkillLevel: l, TrainedSkillLevel: l})
			}
			return out, nil
		}},
		fakeCharacterImplantsESI: fakeCharacterImplantsESI{characterImplants: func(context.Context) ([]int64, error) { return nil, nil }},
		fakeUniverseSystemKillsESI: fakeUniverseSystemKillsESI{universeSystemKills: func(context.Context) ([]esiapi.UniverseSystemKillsGetInner, error) {
			return nil, nil
		}},
		fakeMarketOrdersESI: fakeMarketOrdersESI{marketOrders: sellBook(book)},
	}
}

// sellBook fakes a region's order book: one deep sell order at Jita 4-4
// for each type priced in book.
func sellBook(book map[int64]float64) func(context.Context, int64, string, int64, int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
	return func(_ context.Context, _ int64, orderType string, typeID int64, _ int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		p, ok := book[typeID]
		if !ok || orderType == "buy" {
			return nil, 1, nil
		}
		return []esiapi.MarketsRegionIdOrdersGetInner{{TypeId: typeID, Price: p, VolumeRemain: 1e9, LocationId: 60003760}}, 1, nil
	}
}

// useFakeESI points the planner's openers at f.
func useFakeESI(p *planner, f fakeESI) {
	p.openESI, p.openPublicESI = openFakeESI(f), openFakePublicESI(f)
}

func TestPlanBuild_EndToEnd(t *testing.T) {
	store, p := industryTestEnv(t)
	useFakeESI(p, fakeIndustryESI())
	cfg, _ := store.Load()
	cfg.UpsertFacilityProfile(config.FacilityProfile{Name: "Home Sotiyo", Kind: industry.KindUpwell, StructureType: 35827, SystemID: 30001324, RigTypeIDs: []int64{37180, 43704}, Services: []string{"manufacturing", "capital_shipyard"}})
	cfg.UpsertFacilityProfile(config.FacilityProfile{Name: "Home Tatara", Kind: industry.KindUpwell, StructureType: 35836, SystemID: 30001324, Services: []string{"reprocessing"}, ReprocessingRigTier: 1})
	cfg.UpsertRoute(config.Route{Name: "JF Nomad outbound", Waypoints: []config.RouteWaypoint{{ID: 30000142}, {ID: 30001448}, {ID: 30000855}, {ID: 30001324}}})
	cfg.UpsertHaulProfile(config.HaulProfile{Name: "JF Nomad", ShipTypeID: 28846, PilotID: int64(testPilot), HoldM3: 158400, OutboundRoute: "JF Nomad outbound"})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	out, err := p.planBuild(context.Background(), PlanBuildInput{
		CharacterID: testPilot, CorporationID: testCorp, Product: "Capital Siege Array", Facility: "Home Sotiyo",
		RefineFacility: "Home Tatara", HaulProfile: "JF Nomad", UseCompressedOre: true, IgnoreStock: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Units != 10 {
		t.Errorf("units = %d, want 10 (one 10-run copy)", out.Units)
	}
	if out.OreMix == nil || len(out.OreMix.Ores) == 0 || out.OreMix.Ores[0].Name != "Compressed Gneiss IV-Grade" {
		t.Fatalf("ore mix = %+v", out.OreMix)
	}
	if out.ShuttleM3 <= 0 {
		t.Error("no refined-mineral shuttle volume reported")
	}
	if out.Haul == nil || len(out.Haul.Loads) == 0 || !strings.Contains(out.Haul.Loads[0].Multibuy, "Hydrogen Isotopes\t") {
		t.Fatalf("haul = %+v", out.Haul)
	}
	if got := out.Haul.Route.Links[0].URL; got != "https://evemaps.dotlan.net/jump/Nomad,554/Jita:Hakonen:NL6V-7:PF-QHK" {
		t.Errorf("route link = %s", got)
	}
	if out.Haul.Route.Fuel != 59025 {
		t.Errorf("outbound fuel = %d, want dotlan's 59,025", out.Haul.Route.Fuel)
	}
	byDest := out.Haul.Loads[0].ByDest
	if byDest["Home Tatara"] == 0 || byDest["Home Sotiyo"] == 0 {
		t.Errorf("load drop-offs = %v", byDest)
	}
	if len(out.BuildVsBuy) != 0 {
		t.Errorf("the final product shouldn't appear in build vs buy: %+v", out.BuildVsBuy)
	}
	if out.TotalCost <= 0 || out.PerUnit <= 0 || !strings.Contains(out.Multibuy, "Wetware Mainframe\t10") {
		t.Errorf("cost %v per unit %v multibuy %q", out.TotalCost, out.PerUnit, out.Multibuy)
	}
	if _, err := os.Stat(out.PlanPath); err != nil {
		t.Errorf("plan not saved: %v", err)
	}
}

func TestPlanBuild_QuantityStillUsesOwnedCopies(t *testing.T) {
	store, p := industryTestEnv(t)
	useFakeESI(p, fakeIndustryESI())
	cfg, _ := store.Load()
	cfg.UpsertFacilityProfile(config.FacilityProfile{Name: "Home Sotiyo", Kind: industry.KindUpwell, StructureType: 35827, SystemID: 30001324, Services: []string{"manufacturing", "capital_shipyard"}})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	out, err := p.planBuild(context.Background(), PlanBuildInput{CharacterID: testPilot, CorporationID: testCorp, Product: "Capital Siege Array", Facility: "Home Sotiyo", Quantity: 4, IgnoreStock: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.Units != 4 {
		t.Errorf("units = %d, want 4 built from the owned 10-run copy", out.Units)
	}
	for _, d := range out.Decisions {
		if d.Name == "Capital Siege Array" && d.Bought != 0 {
			t.Errorf("bought %d of the product instead of building it", d.Bought)
		}
	}
}

func TestPlanBuild_NoCopiesNeedsQuantity(t *testing.T) {
	store, p := industryTestEnv(t)
	fake := fakeIndustryESI()
	fake.fakeCorporationBlueprintsESI.corporationBlueprints = func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
		return nil, 1, nil
	}
	useFakeESI(p, fake)
	cfg, _ := store.Load()
	cfg.UpsertFacilityProfile(config.FacilityProfile{Name: "Home Sotiyo", Kind: industry.KindUpwell, StructureType: 35827, SystemID: 30001324, Services: []string{"manufacturing"}})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	_, err := p.planBuild(context.Background(), PlanBuildInput{CharacterID: testPilot, CorporationID: testCorp, Product: "Capital Siege Array", Facility: "Home Sotiyo", IgnoreStock: true})
	if err == nil || !strings.Contains(err.Error(), "quantity") {
		t.Errorf("err = %v", err)
	}
}
