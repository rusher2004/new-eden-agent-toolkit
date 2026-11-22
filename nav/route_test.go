package nav_test

import (
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

// Real systems, copied from the SDE 2026-11-18. Perimeter stands in for the
// highsec gate path back to Jita (in the real map Akora is 10 gates out).
func realSystems() []nav.RouteSystem {
	return []nav.RouteSystem{
		{ID: 30000142, Name: "Jita", ConstellationID: 20000020, RegionID: 10000002, RegionName: "The Forge", Security: 0.945913, Pos: jita, HasNPCStation: true},
		{ID: 30000144, Name: "Perimeter", ConstellationID: 20000020, RegionID: 10000002, RegionName: "The Forge", Security: 0.949, Pos: nav.Position{X: -1.432652330889429e+17, Y: 64923714928938896.0, Z: 1.0417862320674181e+17}, HasNPCStation: true},
		{ID: 30000163, Name: "Akora", ConstellationID: 20000023, RegionID: 10000002, RegionName: "The Forge", Security: 0.324625, Pos: nav.Position{X: -67075105340374504.0, Y: 49557054291704304.0, Z: 94326218506763040.0}, HasNPCStation: true},
		{ID: 30000205, Name: "Obe", ConstellationID: 20000029, RegionID: 10000002, RegionName: "The Forge", Security: 0.346931, Pos: nav.Position{X: -92842412380595552.0, Y: 70825121430840328.0, Z: 1.4777839733012102e+17}, HasNPCStation: true},
		{ID: 30001398, Name: "Aunenen", ConstellationID: 20000205, RegionID: 10000016, RegionName: "Lonetrek", Security: 0.44779, Pos: aunenen, HasNPCStation: true},
		{ID: 30001448, Name: "Hakonen", ConstellationID: 20000212, RegionID: 10000016, RegionName: "Lonetrek", Security: 0.296379, Pos: hakonen, HasNPCStation: true},
		{ID: 30001385, Name: "Jan", ConstellationID: 20000202, RegionID: 10000016, RegionName: "Lonetrek", Security: 0.416954, Pos: jan, HasNPCStation: true},
		{ID: 30000855, Name: "NL6V-7", ConstellationID: 20000125, RegionID: 10000010, RegionName: "Tribute", Security: -0.255748, Pos: nl6v7},
		{ID: 30001324, Name: "PF-QHK", ConstellationID: 20000193, RegionID: 10000015, RegionName: "Venal", Security: -0.134402, Pos: pfqhk, HasNPCStation: true},
	}
}

var (
	gatesToJita = [][2]int64{{30000163, 30000144}, {30000144, 30000142}}
	mb          = nav.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 5, JumpFreighters: 4}
	s544        = nav.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 4, JumpFreighters: 4}
)

func pfRequest(skills nav.JumpSkills) nav.RouteRequest {
	return nav.RouteRequest{From: 30000142, To: 30001324, Drive: nomad, Skills: skills, CanUseHighsec: true}
}

func TestValidateRoute_OutboundMatchesPlanAndDotlan(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), gatesToJita)
	rt, err := g.ValidateRoute(pfRequest(s544), []int64{30000142, 30001385, 30000855, 30001324})
	if err != nil {
		t.Fatal(err)
	}
	if rt.Jumps != 3 || rt.Gates != 0 || rt.Fuel != 69519 {
		t.Errorf("route = %d jumps, %d gates, %d fuel; want 3, 0, 69,519", rt.Jumps, rt.Gates, rt.Fuel)
	}
	if len(rt.Links) != 1 || rt.Links[0].URL != "https://evemaps.dotlan.net/jump/Nomad,544/Jita:Jan:NL6V-7:PF-QHK" {
		t.Errorf("links = %+v", rt.Links)
	}
	real, err := g.ValidateRoute(pfRequest(mb), []int64{30000142, 30001385, 30000855, 30001324})
	if err != nil {
		t.Fatal(err)
	}
	if real.Links[0].URL != "https://evemaps.dotlan.net/jump/Nomad,554/Jita:Jan:NL6V-7:PF-QHK" {
		t.Errorf("5/5/4 link = %s", real.Links[0].URL)
	}
}

func TestValidateRoute_ReturnEndsInGates(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), gatesToJita)
	req := nav.RouteRequest{From: 30001324, To: 30000142, Drive: nomad, Skills: mb, CanUseHighsec: true}
	rt, err := g.ValidateRoute(req, []int64{30001324, 30000855, 30001385, 30000205, 30000163, 30000142})
	if err != nil {
		t.Fatal(err)
	}
	if rt.Jumps != 4 || rt.Gates != 2 || rt.HighsecGates != 2 {
		t.Errorf("route = %d jumps, %d gates (%d highsec)", rt.Jumps, rt.Gates, rt.HighsecGates)
	}
	if len(rt.Links) != 2 ||
		rt.Links[0].Label != "Jumps: PF-QHK → Akora" || rt.Links[0].URL != "https://evemaps.dotlan.net/jump/Nomad,554/PF-QHK:NL6V-7:Jan:Obe:Akora" ||
		rt.Links[1].Label != "Gates: Akora → Jita" || rt.Links[1].URL != "https://evemaps.dotlan.net/route/Akora:Perimeter:Jita" {
		t.Errorf("links = %+v", rt.Links)
	}
}

func TestValidateRoute_OutOfRange(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), nil)
	_, err := g.ValidateRoute(pfRequest(mb), []int64{30000142, 30001324})
	if err == nil || !strings.Contains(err.Error(), "LY") {
		t.Errorf("err = %v, want an out-of-range error with the distance", err)
	}
}

func TestPlanRoutes_RealSystems(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), gatesToJita)
	routes, err := g.PlanRoutes(pfRequest(mb))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range routes {
		for _, l := range r.Legs {
			if l.Kind != "jump" {
				continue
			}
			to := g.Systems[l.To]
			if !nav.CynoLegal(to.Security, to.RegionID) {
				t.Errorf("%s lands in %s", r.Label, l.ToName)
			}
			if l.LY > 10 {
				t.Errorf("%s jumps %.2f LY", r.Label, l.LY)
			}
		}
	}
	if routes[0].Systems[len(routes[0].Systems)-1] != 30001324 {
		t.Errorf("best route doesn't reach PF-QHK: %+v", routes[0].Systems)
	}
}

func TestPlanRoutes_AvoidAndRequiredMidpoints(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), gatesToJita)
	req := pfRequest(mb)
	req.Prefs.AvoidSystems = map[int64]bool{30001385: true} // Jan
	routes, err := g.PlanRoutes(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range routes {
		for _, id := range r.Systems {
			if id == 30001385 {
				t.Errorf("%s uses avoided Jan", r.Label)
			}
		}
	}
	req = pfRequest(mb)
	req.Prefs.Midpoints = map[int64]bool{30001398: true, 30000855: true} // Aunenen, NL6V-7
	req.Prefs.MidpointsRequired = true
	routes, err = g.PlanRoutes(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range routes[0].Legs {
		if l.Kind == "jump" && l.To != 30001324 && !req.Prefs.Midpoints[l.To] {
			t.Errorf("required midpoints ignored: lands in %s", l.ToName)
		}
	}
}

func TestPlanRoutes_NoHighsecLanding(t *testing.T) {
	// A carrier can't use highsec: from Akora it must not route through Jita. Range is set to 10 LY so the fixture's 9.8 LY NL6V-7 → PF-QHK hop is reachable.
	g := nav.NewRouteGraph(realSystems(), gatesToJita)
	req := nav.RouteRequest{From: 30000163, To: 30001324, Drive: nav.JumpDrive{ShipName: "Thanatos", BaseRangeLY: 5, FuelPerLY: 3000}, Skills: mb}
	routes, err := g.PlanRoutes(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range routes[0].Systems {
		if s := g.Systems[id]; nav.BandOf(s.Security, s.RegionID) == nav.Highsec {
			t.Errorf("carrier route enters highsec %s", s.Name)
		}
	}
}

func TestPlanRoutes_GateCap(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), gatesToJita)
	req := nav.RouteRequest{From: 30001324, To: 30000142, Drive: nomad, Skills: mb, CanUseHighsec: true}
	req.Prefs.MaxGates = 1
	if _, err := g.PlanRoutes(req); err == nil {
		t.Error("route to Jita found with a 1-gate cap; Jita is 2 gates from the nearest legal landing here")
	}
}

func TestDotlanURLs(t *testing.T) {
	thanatos544 := nav.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 4, JumpFreighters: 4}
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"flags and avoid list", nav.DotlanJumpURL("Thanatos", thanatos544, true, true, []string{"Ahbazon", "Tama"}, []string{"Jita", "NL6V-7", "PF-QHK"}),
			"https://evemaps.dotlan.net/jump/Thanatos,544,S,I,-Ahbazon:-Tama/Jita:NL6V-7:PF-QHK"},
		{"spaces become underscores", nav.DotlanJumpURL("Thanatos", thanatos544, false, false, []string{"Derelik", "Scalding Pass"}, []string{"Tash-Murkon Prime", "6RCQ-V"}),
			"https://evemaps.dotlan.net/jump/Thanatos,544,-Derelik:-Scalding_Pass/Tash-Murkon_Prime:6RCQ-V"},
		{"gate route", nav.DotlanGateURL([]string{"Akora", "New Caldari"}), "https://evemaps.dotlan.net/route/Akora:New_Caldari"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s:\n got  %s\n want %s", tt.name, tt.got, tt.want)
		}
	}
}

func TestSuggestion_ExpandsConstellations(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), gatesToJita)
	req := nav.RouteRequest{Drive: nav.JumpDrive{ShipName: "Thanatos"}, Skills: nav.JumpSkills{JumpDriveCalibration: 5, JumpFuelConservation: 4, JumpFreighters: 4}}
	req.Prefs.AvoidConstellations = map[int64]bool{20000020: true} // Jita and Perimeter
	req.Prefs.AvoidSovSystems = map[int64]bool{1: true}
	link := g.Suggestion(req, []int64{30000163, 30001324})
	if link.URL != "https://evemaps.dotlan.net/jump/Thanatos,544,-Jita:-Perimeter/Akora:PF-QHK" {
		t.Errorf("url = %s", link.URL)
	}
	if !strings.Contains(link.Label, "sovereignty") {
		t.Errorf("label doesn't flag sov preferences: %s", link.Label)
	}
}
