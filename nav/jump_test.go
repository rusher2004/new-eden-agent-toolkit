package nav_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

// Real SDE positions, copied 2026-11-18.
var (
	jita    = nav.Position{X: -1.29064861735e+17, Y: 60755306910000000.0, Z: 1.1746922706e+17}
	aunenen = nav.Position{X: -1.9065262629888752e+17, Y: 88927690357314880.0, Z: 1.7836417700017386e+17}
	hakonen = nav.Position{X: -1.46073864274e+17, Y: 96163862684300000.0, Z: 1.71951994581e+17}
	jan     = nav.Position{X: -1.4303798065207331e+17, Y: 79758297777864400.0, Z: 1.613755528761681e+17}
	nl6v7   = nav.Position{X: -1.4903294260526365e+17, Y: 99542048543805280.0, Z: 2.3926934557504768e+17}
	pfqhk   = nav.Position{X: -1.305672291398673e+17, Y: 98759478627101390.0, Z: 3.297507456526137e+17}
	nomad   = nav.JumpDrive{ShipTypeID: 28846, ShipName: "Nomad", BaseRangeLY: 5, FuelPerLY: 8200, FuelTypeID: 17889, SkillBonusPerLevel: -10, FatigueMultiplier: 0.1}
)

func TestJumpFuel_MatchesDotlan(t *testing.T) {
	s544 := nav.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 4, JumpFreighters: 4}
	legs := []struct {
		a, b nav.Position
		want int64
	}{{jita, jan, 15553}, {jan, nl6v7, 25148}, {nl6v7, pfqhk, 28818}}
	var total int64
	for _, l := range legs {
		got := nav.JumpFuel(nomad, s544, nav.LightYears(l.a, l.b))
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
	s := nav.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 5, JumpFreighters: 4}
	if got := nav.JumpRange(nomad, s); got != 10 {
		t.Errorf("range = %v, want 10", got)
	}
	if got := nav.FuelPerLY(nomad, s); math.Abs(got-2460) > 1e-9 {
		t.Errorf("fuel/LY = %v, want 2460", got)
	}
}

func TestFatigue_WikiExample(t *testing.T) {
	// EVE University: three 5 LY jumps, no reduction.
	f, cd := nav.Fatigue(0, 5, 1)
	if f != 60 || cd != 6 {
		t.Errorf("jump 1 = %v, %v; want 60, 6", f, cd)
	}
	f, cd = nav.Fatigue(f, 5, 1)
	if f != 300 || cd != 6 {
		t.Errorf("jump 2 = %v, %v; want 300 (cap), 6", f, cd)
	}
	f, cd = nav.Fatigue(f, 5, 1)
	if f != 300 || cd != 30 {
		t.Errorf("jump 3 = %v, %v; want 300, 30 (cap)", f, cd)
	}
	// Jump freighter: 9.63 LY at 90% reduction counts as 0.963.
	f, cd = nav.Fatigue(0, 9.63, 0.1)
	if math.Abs(f-19.63) > 1e-9 || math.Abs(cd-1.963) > 1e-9 {
		t.Errorf("JF jump = %v, %v", f, cd)
	}
}
