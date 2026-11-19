package industry_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
)

// Real SDE positions, copied 2026-11-18.
var (
	jita    = industry.Position{X: -1.29064861735e+17, Y: 60755306910000000.0, Z: 1.1746922706e+17}
	aunenen = industry.Position{X: -1.9065262629888752e+17, Y: 88927690357314880.0, Z: 1.7836417700017386e+17}
	hakonen = industry.Position{X: -1.46073864274e+17, Y: 96163862684300000.0, Z: 1.71951994581e+17}
	jan     = industry.Position{X: -1.4303798065207331e+17, Y: 79758297777864400.0, Z: 1.613755528761681e+17}
	nl6v7   = industry.Position{X: -1.4903294260526365e+17, Y: 99542048543805280.0, Z: 2.3926934557504768e+17}
	pfqhk   = industry.Position{X: -1.305672291398673e+17, Y: 98759478627101390.0, Z: 3.297507456526137e+17}
	nomad   = industry.JumpDrive{ShipTypeID: 28846, ShipName: "Nomad", BaseRangeLY: 5, FuelPerLY: 8200, FuelTypeID: 17889, SkillBonusPerLevel: -10, FatigueMultiplier: 0.1}
)

func TestJumpFuel_MatchesDotlan(t *testing.T) {
	s544 := industry.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 4, JumpFreighters: 4}
	legs := []struct {
		a, b industry.Position
		want int64
	}{{jita, jan, 15553}, {jan, nl6v7, 25148}, {nl6v7, pfqhk, 28818}}
	var total int64
	for _, l := range legs {
		got := industry.JumpFuel(nomad, s544, industry.LightYears(l.a, l.b))
		if got != l.want {
			t.Errorf("leg fuel = %d, want dotlan's %d", got, l.want)
		}
		total += got
	}
	if total != 69519 {
		t.Errorf("total = %d, want dotlan's 69,519", total)
	}
}

func TestJumpRangeAndFuelPerLY(t *testing.T) {
	s := industry.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 5, JumpFreighters: 4}
	if got := industry.JumpRange(nomad, s); got != 10 {
		t.Errorf("range = %v, want 10", got)
	}
	if got := industry.FuelPerLY(nomad, s); math.Abs(got-2460) > 1e-9 {
		t.Errorf("fuel/LY = %v, want 2460", got)
	}
}

func TestFatigue_WikiExample(t *testing.T) {
	// EVE University: three 5 LY jumps, no reduction.
	f, cd := industry.Fatigue(0, 5, 1)
	if f != 60 || cd != 6 {
		t.Errorf("jump 1 = %v, %v; want 60, 6", f, cd)
	}
	f, cd = industry.Fatigue(f, 5, 1)
	if f != 300 || cd != 6 {
		t.Errorf("jump 2 = %v, %v; want 300 (cap), 6", f, cd)
	}
	f, cd = industry.Fatigue(f, 5, 1)
	if f != 300 || cd != 30 {
		t.Errorf("jump 3 = %v, %v; want 300, 30 (cap)", f, cd)
	}
	// Jump freighter: 9.63 LY at 90% reduction counts as 0.963.
	f, cd = industry.Fatigue(0, 9.63, 0.1)
	if math.Abs(f-19.63) > 1e-9 || math.Abs(cd-1.963) > 1e-9 {
		t.Errorf("JF jump = %v, %v", f, cd)
	}
}

func TestPackLoads_EvenSlices(t *testing.T) {
	items := []industry.Cargo{
		{TypeID: 57487, Name: "Capital Core Temperature Regulator", Quantity: 15, UnitM3: 28000, Dest: "Sotiyo"},
		{TypeID: 35, Name: "Pyerite", Quantity: 159146178, UnitM3: 0.01, Dest: "Sotiyo"},
		{TypeID: 62555, Name: "Compressed Gneiss IV-Grade", Quantity: 2413600, UnitM3: 0.05, Portion: 100, Dest: "Tatara"},
	}
	loads, err := industry.PackLoads(items, 158400)
	if err != nil {
		t.Fatal(err)
	}
	sums := map[int64]int64{}
	for _, l := range loads {
		if l.M3 > 158400 {
			t.Errorf("load %d is %v m³", l.Number, l.M3)
		}
		for _, it := range l.Items {
			sums[it.TypeID] += it.Quantity
			if it.TypeID == 62555 && it.Quantity%100 != 0 {
				t.Errorf("ore split mid-portion: %d", it.Quantity)
			}
		}
		if l.ByDest["Tatara"] == 0 {
			t.Errorf("load %d has no ore", l.Number)
		}
	}
	for _, it := range items {
		if sums[it.TypeID] != it.Quantity {
			t.Errorf("%s: packed %d of %d", it.Name, sums[it.TypeID], it.Quantity)
		}
	}
	if len(loads) != 14 {
		t.Errorf("loads = %d, want 14 (2,132,142 m³ / 158,400 = 13.5)", len(loads))
	}
}

func TestPackLoads_TooBig(t *testing.T) {
	_, err := industry.PackLoads([]industry.Cargo{{Name: "Minokawa", Quantity: 1, UnitM3: 1300000}}, 158400)
	if err == nil {
		t.Error("a packaged capital fit a jump freighter")
	}
}

func TestMultibuy(t *testing.T) {
	got := industry.Multibuy([]industry.MultibuyLine{{"Tritanium", 1000}, {"Pyerite", 5}, {"Tritanium", 1}, {"Nothing", 0}})
	if got != "Tritanium\t1001\nPyerite\t5" {
		t.Errorf("multibuy = %q", got)
	}
}
