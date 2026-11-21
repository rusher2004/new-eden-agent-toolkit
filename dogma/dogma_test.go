package dogma_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/dogma"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// Real dogma from the SDE, 2026-11-20.
var (
	stackable = map[int64]dogma.AttrInfo{38: {Stackable: true}, 714: {Stackable: true}, 73: {Stackable: true}}

	expandedCargoholdII = dogma.Item{Kind: dogma.Module, TypeID: 1319, Attrs: map[int64]float64{149: 1.275},
		Effects: []dogma.Effect{{Name: "cargoCapacityMultiply", Category: dogma.CategoryOnline, Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "ItemModifier", ModifiedAttr: 38, ModifyingAttr: 149, Operation: dogma.OpPostMul}}}}}
	smallCargoholdOptimization = dogma.Item{Kind: dogma.Module, TypeID: 31117, Attrs: map[int64]float64{614: 15},
		Effects: []dogma.Effect{{Name: "skillFreightBonus", Category: dogma.CategoryPassive, Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "ItemModifier", ModifiedAttr: 38, ModifyingAttr: 614, Operation: dogma.OpPostPercent}}}}}
)

func TestShipAttr_ArkFittedHold(t *testing.T) {
	// Ark: 135,000 m³ base, +5% per level of Amarr Freighter (IV), two
	// Expanded Cargohold IIs: 135,000 x 1.20 x 1.275² = 263,351.25 m³.
	ark := dogma.Item{Kind: dogma.Ship, TypeID: 28850, Attrs: map[int64]float64{38: 135000, 887: 5},
		Effects: []dogma.Effect{{Name: "freighterCargoBonusA2", Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "ItemModifier", ModifiedAttr: 38, ModifyingAttr: 887, Operation: dogma.OpPostPercent}}}}}
	amarrFreighter := dogma.NewSkill(20524, 4, nil, []dogma.Effect{
		{Name: "skillEffect", Modifiers: []dogma.Modifier{{Domain: "itemID", Func: "ItemModifier", ModifiedAttr: 280, ModifyingAttr: 276, Operation: 9}}},
		{Name: "freighterA2SkillLevel", Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "ItemModifier", ModifiedAttr: 887, ModifyingAttr: 280, Operation: dogma.OpPreMul}}},
	})
	f := dogma.Fit{Ship: ark, Modules: []dogma.Item{expandedCargoholdII, expandedCargoholdII}, Skills: []dogma.Item{amarrFreighter}, Attrs: stackable}
	if got := f.ShipAttr(38); !near(got, 263351.25) {
		t.Errorf("Ark hold = %v, want 263351.25", got)
	}
}

func TestShipAttr_CynoVentureHold(t *testing.T) {
	// Venture: 50 m³, one Expanded Cargohold II and three Small Cargohold
	// Optimization I: 50 x 1.275 x 1.15³ ≈ 96.96 m³.
	venture := dogma.Item{Kind: dogma.Ship, TypeID: 32880, Attrs: map[int64]float64{38: 50}}
	f := dogma.Fit{Ship: venture, Modules: []dogma.Item{expandedCargoholdII, smallCargoholdOptimization, smallCargoholdOptimization, smallCargoholdOptimization}, Attrs: stackable}
	if got, want := f.ShipAttr(38), 50*1.275*math.Pow(1.15, 3); !near(got, want) {
		t.Errorf("Venture hold = %v, want %v", got, want)
	}
}

func cynoFit(t *testing.T, hullBonus, durationBonus float64, cftLevel int) dogma.Fit {
	t.Helper()
	const cft = 21603
	shipEffects := []dogma.Effect{{Name: "cynosuralTheoryConsumptionBonus", Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "LocationRequiredSkillModifier", ModifiedAttr: 714, ModifyingAttr: 1296, Operation: dogma.OpPostPercent, SkillTypeID: cft}}}}
	if durationBonus != 0 {
		shipEffects = append(shipEffects, dogma.Effect{Name: "cynosuralDurationBonus", Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "LocationRequiredSkillModifier", ModifiedAttr: 73, ModifyingAttr: 66, Operation: dogma.OpPostPercent, SkillTypeID: cft}}})
	}
	ship := dogma.Item{Kind: dogma.Ship, Attrs: map[int64]float64{1296: hullBonus, 66: durationBonus}, Effects: shipEffects}
	skill := dogma.NewSkill(cft, cftLevel, map[int64]float64{1296: -10}, []dogma.Effect{
		{Name: "cynosuralTheoryConsumptionBonus", Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "LocationRequiredSkillModifier", ModifiedAttr: 714, ModifyingAttr: 1296, Operation: dogma.OpPostPercent, SkillTypeID: cft}}},
		{Name: "skillConsumptionQuantityBonusPercentageSkillLevel", Modifiers: []dogma.Modifier{{Domain: "itemID", Func: "ItemModifier", ModifiedAttr: 1296, ModifyingAttr: 280, Operation: dogma.OpPreMul}}},
	})
	return dogma.Fit{Ship: ship, Skills: []dogma.Item{skill}, Attrs: stackable}
}

func TestModuleAttr_CynoConsumption(t *testing.T) {
	industrial := dogma.Item{Kind: dogma.Module, TypeID: 52694, Attrs: map[int64]float64{714: 800, 73: 600000}, RequiredSkills: []int64{21603}}
	standard := dogma.Item{Kind: dogma.Module, TypeID: 21096, Attrs: map[int64]float64{714: 500, 73: 600000}, RequiredSkills: []int64{21603}}
	tests := []struct {
		name                   string
		gen                    dogma.Item
		hull, duration         float64
		cft                    int
		wantOzone, wantCycleMs float64
	}{
		{"Venture, industrial cyno, CFT IV", industrial, -50, 0, 4, 240, 600000},
		{"Venture, industrial cyno, CFT V", industrial, -50, 0, 5, 200, 600000},
		{"Falcon, standard cyno, CFT V", standard, -80, -50, 5, 50, 300000},
	}
	for _, tt := range tests {
		f := cynoFit(t, tt.hull, tt.duration, tt.cft)
		f.Modules = []dogma.Item{tt.gen}
		if got := f.ModuleAttr(0, 714); !near(got, tt.wantOzone) {
			t.Errorf("%s: ozone = %v, want %v", tt.name, got, tt.wantOzone)
		}
		if got := f.ModuleAttr(0, 73); !near(got, tt.wantCycleMs) {
			t.Errorf("%s: cycle = %v, want %v", tt.name, got, tt.wantCycleMs)
		}
	}
}

func TestStackingPenalty(t *testing.T) {
	// Three +10% modules on a non-stackable attribute: 1.10 x (1 + 0.10 x 0.8691) x (1 + 0.10 x 0.5706).
	mod := dogma.Item{Kind: dogma.Module, Attrs: map[int64]float64{9999: 10},
		Effects: []dogma.Effect{{Category: dogma.CategoryPassive, Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "ItemModifier", ModifiedAttr: 5, ModifyingAttr: 9999, Operation: dogma.OpPostPercent}}}}}
	f := dogma.Fit{Ship: dogma.Item{Kind: dogma.Ship, Attrs: map[int64]float64{5: 100}}, Modules: []dogma.Item{mod, mod, mod}, Attrs: map[int64]dogma.AttrInfo{5: {Stackable: false}}}
	want := 100 * 1.10 * (1 + 0.10*math.Exp(-math.Pow(1/2.67, 2))) * (1 + 0.10*math.Exp(-math.Pow(2/2.67, 2)))
	if got := f.ShipAttr(5); !near(got, want) {
		t.Errorf("penalized = %v, want %v", got, want)
	}
}

func TestActiveEffectsDontApply(t *testing.T) {
	active := dogma.Item{Kind: dogma.Module, Attrs: map[int64]float64{1: 2},
		Effects: []dogma.Effect{{Category: dogma.CategoryActive, Modifiers: []dogma.Modifier{{Domain: "shipID", Func: "ItemModifier", ModifiedAttr: 38, ModifyingAttr: 1, Operation: dogma.OpPostMul}}}}}
	f := dogma.Fit{Ship: dogma.Item{Kind: dogma.Ship, Attrs: map[int64]float64{38: 100}}, Modules: []dogma.Item{active}, Attrs: stackable}
	if got := f.ShipAttr(38); got != 100 {
		t.Errorf("an active effect applied: %v", got)
	}
}

func TestDefaultsAndCycles(t *testing.T) {
	// An attribute missing from the type takes the SDE default; a modifier
	// cycle falls back to base values instead of looping.
	cyc := dogma.Item{Kind: dogma.Ship, Attrs: map[int64]float64{1: 10, 2: 3},
		Effects: []dogma.Effect{{Modifiers: []dogma.Modifier{
			{Domain: "itemID", Func: "ItemModifier", ModifiedAttr: 1, ModifyingAttr: 2, Operation: dogma.OpModAdd},
			{Domain: "itemID", Func: "ItemModifier", ModifiedAttr: 2, ModifyingAttr: 1, Operation: dogma.OpModAdd},
		}}}}
	f := dogma.Fit{Ship: cyc, Attrs: map[int64]dogma.AttrInfo{7: {Default: 42}}}
	if got := f.ShipAttr(7); got != 42 {
		t.Errorf("default = %v, want 42", got)
	}
	if got := f.ShipAttr(1); got != 23 {
		t.Errorf("cycle = %v, want 10 + (3 + 10) = 23", got)
	}
}
