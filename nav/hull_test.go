package nav_test

import (
	"math"
	"slices"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

func TestDescribeHull_JumpFreighter(t *testing.T) {
	// The Nomad's base values from the SDE.
	f := nav.DescribeHull(nav.HullInput{TypeID: 28846, Name: "Nomad", GroupID: 902, GroupName: "Jump Freighter", MassKg: 820000000,
		Attrs: map[int64]float64{37: 80, 70: 0.0275, 600: 0.5, 861: 1, 866: 17889, 867: 5, 868: 8200, 1281: 3, 1971: 0.1}})
	if f.WarpSpeedAU != 1.5 || !f.EntersHighsec || f.Nullification != "" || f.CovertOpsCloak {
		t.Errorf("facts = %+v", f)
	}
	if want := math.Log(4) * 0.0275 * 820000000 / 1e6; math.Abs(f.AlignSeconds-want) > 1e-9 {
		t.Errorf("align = %v, want %v", f.AlignSeconds, want)
	}
	if f.Jump == nil || f.Jump.BaseRangeLY != 5 || f.Jump.FatigueMultiplier != 0.1 || !slices.Equal(f.Jump.CynoTypes, []string{"standard", "industrial"}) {
		t.Errorf("jump = %+v", f.Jump)
	}
	if !f.Docking.NPCStations || !slices.Contains(f.Docking.Upwell, "Astrahus") {
		t.Errorf("docking = %+v", f.Docking)
	}
}

func TestDescribeHull_CarrierAndTransport(t *testing.T) {
	carrier := nav.DescribeHull(nav.HullInput{Name: "Archon", GroupID: 547, Attrs: map[int64]float64{861: 1, 867: 3.5, 1970: 1}})
	if carrier.EntersHighsec || carrier.Jump.FatigueMultiplier != 1 || !slices.Equal(carrier.Jump.CynoTypes, []string{"standard"}) {
		t.Errorf("carrier = %+v jump %+v", carrier, carrier.Jump)
	}
	if carrier.Docking.NPCStations != true || slices.Contains(carrier.Docking.Upwell, "Astrahus") || !slices.Equal(carrier.Docking.BuildOnly, []string{"Azbel"}) {
		t.Errorf("carrier docking = %+v", carrier.Docking)
	}
	dst := nav.DescribeHull(nav.HullInput{Name: "Occator", GroupID: 380, Attrs: map[int64]float64{104: -2, 3324: 1, 3325: 1}, NullifierFits: true})
	if dst.WarpCoreStrength != 2 || dst.Nullification != "module" || dst.Jump != nil || !slices.Equal(dst.Rides, []string{"industrial bridge", "industrial conduit"}) {
		t.Errorf("DST = %+v", dst)
	}
	ceptor := nav.DescribeHull(nav.HullInput{Name: "Stiletto", GroupID: 831, Effects: map[int64]bool{8120: true}, NullifierFits: true})
	if ceptor.Nullification != "hull" {
		t.Errorf("interceptor nullification = %q", ceptor.Nullification)
	}
}

func TestCanDock(t *testing.T) {
	tests := []struct {
		group     int64
		structure string
		want      bool
	}{
		{659, "Keepstar", true},
		{659, "Upwell Palatine Keepstar", true},
		{659, "Fortizar", false},
		{547, "'Moreau' Fortizar", true},
		{547, "Astrahus", false},
		{883, "Tatara", true},
		{902, "Raitaru", true},
		{27, "Athanor", true}, // a battleship
	}
	for _, tt := range tests {
		if got := nav.CanDock(tt.group, tt.structure); got != tt.want {
			t.Errorf("CanDock(%d, %q) = %v, want %v", tt.group, tt.structure, got, tt.want)
		}
	}
}
