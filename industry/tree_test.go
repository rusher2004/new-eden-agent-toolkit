package industry_test

import (
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
)

// A trimmed Minokawa: the hull needs siege arrays, capacitor batteries and
// conduits; siege arrays and batteries are built from owned ME10 copies;
// conduits have no owned blueprint. Base quantities are the real ones.
const (
	hull, hullBP       = 37605, 41583
	siege, siegeBP     = 21039, 21040
	battery, batteryBP = 21019, 21020
	conduit            = 57471
	pyerite            = 35
)

func minokawaRecipes(id int64) (industry.Recipe, bool) {
	switch id {
	case hull:
		return industry.Recipe{BlueprintTypeID: hullBP, Activity: industry.ActManufacturing, BaseSeconds: 2000000, OutputPerRun: 1,
			Materials: []industry.TypeQty{{siege, 25}, {battery, 15}, {conduit, 32}}}, true
	case siege:
		return industry.Recipe{BlueprintTypeID: siegeBP, Activity: industry.ActManufacturing, BaseSeconds: 16000, OutputPerRun: 1,
			Materials: []industry.TypeQty{{pyerite, 101250}}}, true
	case battery:
		return industry.Recipe{BlueprintTypeID: batteryBP, Activity: industry.ActManufacturing, BaseSeconds: 16000, OutputPerRun: 1,
			Materials: []industry.TypeQty{{pyerite, 142500}}}, true
	case conduit:
		return industry.Recipe{BlueprintTypeID: 57508, Activity: industry.ActManufacturing, BaseSeconds: 10800, OutputPerRun: 1}, true
	}
	return industry.Recipe{}, false
}

func sotiyoMult(string, int64) industry.Multipliers {
	return industry.Multipliers{Material: 0.99 * 0.958, Time: 0.7 * 0.58, Cost: 0.95}
}

func hullCopies() []industry.Copy {
	var cs []industry.Copy
	for i := range 14 {
		cs = append(cs, industry.Copy{ItemID: int64(100 + i), ME: 9, TE: 14, Runs: 1})
	}
	return append(cs, industry.Copy{ItemID: 200, ME: 6, TE: 10, Runs: 1})
}

func TestExpandTree_MinokawaAggregatesAcrossHulls(t *testing.T) {
	tree, err := industry.ExpandTree(industry.TreeRequest{
		ProductTypeID: hull, RootCopies: hullCopies(), Recipe: minokawaRecipes, Multipliers: sotiyoMult,
		Copies: map[int64][]industry.Copy{
			siegeBP:   {{ItemID: 1, ME: 10, TE: 20, Runs: 300}, {ItemID: 2, ME: 10, TE: 20, Runs: 143}},
			batteryBP: {{ItemID: 3, ME: 10, TE: 20, Runs: 256}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[int64]industry.TreeNode{}
	for _, n := range tree.Nodes {
		nodes[n.TypeID] = n
	}
	// Real plan totals: 331 siege arrays, 196 capacitor batteries, 421 conduits.
	if n := nodes[siege]; n.Demand != 331 || n.Built != 331 || n.Decision != industry.DecideBuild {
		t.Errorf("siege = %+v", n)
	}
	if n := nodes[battery]; n.Demand != 196 || n.Built != 196 {
		t.Errorf("battery = %+v", n)
	}
	if n := nodes[conduit]; n.Demand != 421 || n.Bought != 421 || n.Decision != industry.DecideBuy {
		t.Errorf("conduit = %+v", n)
	}
	var hullJobs, siegeJobs int
	for _, j := range tree.Jobs {
		switch j.ProductTypeID {
		case hull:
			hullJobs++
		case siege:
			siegeJobs++
			if j.Runs != 300 && j.Runs != 31 {
				t.Errorf("siege job runs = %d, want copies consumed largest first (300 then 31)", j.Runs)
			}
		}
	}
	if hullJobs != 15 || siegeJobs != 2 {
		t.Errorf("jobs: %d hull, %d siege", hullJobs, siegeJobs)
	}
}

func TestExpandTree_StockAndShortRuns(t *testing.T) {
	tree, err := industry.ExpandTree(industry.TreeRequest{
		ProductTypeID: hull, RootCopies: hullCopies()[:1], Recipe: minokawaRecipes, Multipliers: sotiyoMult,
		Copies: map[int64][]industry.Copy{siegeBP: {{ItemID: 1, ME: 10, TE: 20, Runs: 10}}},
		Stock:  map[int64]int64{siege: 5, pyerite: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range tree.Nodes {
		if n.TypeID == siege {
			// need 22, 5 from stock, 10 runs owned, 7 bought
			if n.Demand != 22 || n.FromStock != 5 || n.Built != 10 || n.Bought != 7 {
				t.Errorf("siege = %+v", n)
			}
		}
		if n.TypeID == battery && n.Decision != industry.DecideBuy {
			t.Errorf("battery with no copies should be bought: %+v", n)
		}
	}
}

func TestExpandTree_ForcedDecisions(t *testing.T) {
	tree, err := industry.ExpandTree(industry.TreeRequest{
		ProductTypeID: hull, RootCopies: hullCopies()[:1], Recipe: minokawaRecipes, Multipliers: sotiyoMult,
		Copies:    map[int64][]industry.Copy{siegeBP: {{ItemID: 1, ME: 10, TE: 20, Runs: 100}}},
		Decisions: map[int64]string{siege: industry.DecideBuy, pyerite: industry.DecideBuild},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range tree.Nodes {
		if n.TypeID == siege && (n.Decision != industry.DecideBuy || n.Bought != 22) {
			t.Errorf("forced buy = %+v", n)
		}
	}
}

func TestNetStock(t *testing.T) {
	rem, used := industry.NetStock([]industry.TypeQty{{37, 19011945 + 1720083}, {34, 5}}, map[int64]int64{37: 1720083})
	if len(rem) != 2 || rem[0].Quantity != 19011945 || len(used) != 1 || used[0].Quantity != 1720083 {
		t.Errorf("rem %v used %v", rem, used)
	}
}

func TestWalkBook(t *testing.T) {
	orders := []industry.Order{{Price: 207e6, Volume: 18, LocationID: 60003760}, {Price: 204.8e6, Volume: 2, LocationID: 60003760}, {Price: 1, Volume: 99, LocationID: 1}}
	q := industry.WalkBook(57487, orders, 25, 60003760)
	if q.Filled != 20 || q.Shortfall != 5 || q.Lowest != 204.8e6 || q.Total != 2*204.8e6+18*207e6 || q.Depth != 20 {
		t.Errorf("quote = %+v", q)
	}
}

func TestCompareBuildVsBuy(t *testing.T) {
	// Capacitor batteries: Jita has 4, the plan needs 196.
	got := industry.CompareBuildVsBuy(industry.BuildBuyInput{TypeID: battery, Units: 196, InputCost: 196 * 10.5e6, InputM3: 196 * 2050, JobFees: 196 * 0.4e6,
		Buy: industry.Quote{Filled: 4, Shortfall: 192, Average: 31e6}, UnitM3: 2000, FreightPerM3: 500, CanBuild: true})
	if got.Recommend != industry.DecideBuild || len(got.Flags) != 1 {
		t.Errorf("battery = %+v", got)
	}
	cheapBuy := industry.CompareBuildVsBuy(industry.BuildBuyInput{Units: 10, InputCost: 100, Buy: industry.Quote{Filled: 10, Average: 5}, CanBuild: true})
	if cheapBuy.Recommend != industry.DecideBuy || cheapBuy.SavedByBuilding >= 0 {
		t.Errorf("cheap buy = %+v", cheapBuy)
	}
}

func TestExpandTree_SkipsBuildWithoutSkills(t *testing.T) {
	tree, err := industry.ExpandTree(industry.TreeRequest{
		ProductTypeID: hull, RootCopies: hullCopies()[:1], Recipe: minokawaRecipes, Multipliers: sotiyoMult,
		Copies: map[int64][]industry.Copy{siegeBP: {{ItemID: 1, ME: 10, TE: 20, Runs: 100}}},
		CanBuild: func(r industry.Recipe) (bool, string) {
			if r.BlueprintTypeID == siegeBP {
				return false, "no builder has the skills"
			}
			return true, ""
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range tree.Nodes {
		if n.TypeID == siege && (n.Decision != industry.DecideBuy || n.Bought != 22) {
			t.Errorf("siege without skills = %+v", n)
		}
	}
}
