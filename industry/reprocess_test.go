package industry_test

import (
	"math"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/market"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

var allFive = industry.ReprocessSkills{Reprocessing: 5, ReprocessingEfficiency: 5}

func TestUpwellYield_AllFiveAtTatara(t *testing.T) {
	tatara := industry.UpwellYieldInput{RigTier: 1, Band: nav.Nullsec, StructureBonusPct: 5.5}
	// Confirmed in game 2026-11-18: 82.3% without an implant.
	if got := industry.UpwellYield(tatara, allFive, 4); math.Abs(got-0.82333) > 0.0001 {
		t.Errorf("no implant = %.5f, want 0.8233", got)
	}
	withImplant := allFive
	withImplant.Implant = 0.04
	if got := industry.UpwellYield(tatara, withImplant, 4); math.Abs(got-0.856226) > 0.00001 {
		t.Errorf("RX-804 = %.6f, want 0.856226", got)
	}
	// EVE University's worked example: T2 rig, max skills, RX-804 = 90.6%.
	max := industry.ReprocessSkills{Reprocessing: 5, ReprocessingEfficiency: 5, Implant: 0.04}
	if got := industry.UpwellYield(industry.UpwellYieldInput{RigTier: 2, Band: nav.Nullsec, StructureBonusPct: 5.5}, max, 5); math.Abs(got-0.90628) > 0.0001 {
		t.Errorf("wiki example = %.5f, want 0.9063", got)
	}
	// No rig: the security term doesn't apply.
	if got := industry.UpwellYield(industry.UpwellYieldInput{Band: nav.Nullsec}, industry.ReprocessSkills{}, 0); got != 0.5 {
		t.Errorf("bare citadel = %v, want 0.5", got)
	}
}

func TestNPCYieldAndTax(t *testing.T) {
	if got := industry.NPCYield(0.5, allFive, 4); math.Abs(got-0.5*1.15*1.1*1.08) > 1e-9 {
		t.Errorf("npc yield = %v", got)
	}
	if industry.NPCReprocessingTax(0) != 0.05 || industry.NPCReprocessingTax(6.67) != 0 || industry.NPCReprocessingTax(-3) != 0.05 {
		t.Error("tax endpoints wrong")
	}
}

func TestReprocess_GneissMatchesPlan(t *testing.T) {
	gneiss := []industry.TypeQty{{35, 2300}, {36, 1725}, {37, 920}}
	out := industry.Reprocess(gneiss, 100, 2413600, 0.856226)
	got := map[int64]int64{}
	for _, o := range out {
		got[o.TypeID] = o.Quantity
	}
	if got[37] < 19012500 || got[37] > 19012700 {
		t.Errorf("isogen = %d, want about 19,012,595", got[37])
	}
	if partial := industry.Reprocess(gneiss, 100, 99, 1); len(partial) != 0 {
		t.Errorf("a partial portion reprocessed: %v", partial)
	}
}

func TestSolveOreMix_PicksCheaperPerMineral(t *testing.T) {
	// Need 1,000 A and 500 B. Ore X yields 10 A + 5 B per unit (portion 1) at
	// 100; buying minerals directly costs 15 per A and 15 per B. Ore covers
	// both at 10 per A-equivalent, so the optimum is 100 X = 10,000 ISK.
	req := industry.OreMixRequest{
		Needs: map[int64]int64{1: 1000, 2: 500},
		Ores:  []industry.OreOption{{TypeID: 10, Name: "X", Portion: 1, M3: 0.1, PerPortion: []industry.TypeQty{{1, 10}, {2, 5}}, Yield: 1, Book: []market.Order{{Price: 100, Volume: 1000}}}},
		Minerals: []industry.MineralOption{
			{TypeID: 1, Name: "A", M3: 0.01, Book: []market.Order{{Price: 15, Volume: 1e6}}},
			{TypeID: 2, Name: "B", M3: 0.01, Book: []market.Order{{Price: 15, Volume: 1e6}}},
		},
	}
	mix, err := industry.SolveOreMix(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(mix.Ores) != 1 || mix.Ores[0].Quantity != 100 || len(mix.Minerals) != 0 || mix.TotalCost != 10000 {
		t.Errorf("mix = %+v", mix)
	}
}

func TestSolveOreMix_FreightTipsToMinerals(t *testing.T) {
	// Same as above but ore is bulky: 200 m³ per unit at 1 ISK/m³ freight
	// makes X cost 300 per unit against 225 for its minerals, so direct minerals win.
	req := industry.OreMixRequest{
		Needs:        map[int64]int64{1: 1000, 2: 500},
		Ores:         []industry.OreOption{{TypeID: 10, Name: "X", Portion: 1, M3: 200, PerPortion: []industry.TypeQty{{1, 10}, {2, 5}}, Yield: 1, Book: []market.Order{{Price: 100, Volume: 1000}}}},
		Minerals:     []industry.MineralOption{{TypeID: 1, M3: 0.01, Book: []market.Order{{Price: 15, Volume: 1e6}}}, {TypeID: 2, M3: 0.01, Book: []market.Order{{Price: 15, Volume: 1e6}}}},
		FreightPerM3: 1,
	}
	mix, err := industry.SolveOreMix(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(mix.Ores) != 0 || len(mix.Minerals) != 2 {
		t.Errorf("mix = %+v", mix)
	}
}

func TestSolveOreMix_PortionsAndByproducts(t *testing.T) {
	req := industry.OreMixRequest{
		Needs:    map[int64]int64{1: 950},
		Ores:     []industry.OreOption{{TypeID: 10, Name: "X", Portion: 100, M3: 0.01, PerPortion: []industry.TypeQty{{1, 1000}, {3, 50}}, Yield: 1, Book: []market.Order{{Price: 1, Volume: 1000}}}},
		Minerals: []industry.MineralOption{{TypeID: 1, M3: 0.01, Book: []market.Order{{Price: 100, Volume: 1e6}}}},
	}
	mix, err := industry.SolveOreMix(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(mix.Ores) != 1 || mix.Ores[0].Quantity != 100 || len(mix.Surplus) != 1 || mix.Surplus[0].Quantity != 50 || len(mix.Byproducts) != 1 {
		t.Errorf("mix = %+v", mix)
	}
}

func TestSolveOreMix_Infeasible(t *testing.T) {
	_, err := industry.SolveOreMix(industry.OreMixRequest{
		Needs:    map[int64]int64{40: 1000},
		Minerals: []industry.MineralOption{{TypeID: 40, Name: "Megacyte", M3: 0.01, Book: []market.Order{{Price: 2000, Volume: 10}}}},
	})
	if err == nil || !strings.Contains(err.Error(), "type 40") {
		t.Errorf("err = %v, want a shortfall naming type 40", err)
	}
}
