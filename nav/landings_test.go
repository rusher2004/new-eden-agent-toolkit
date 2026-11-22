package nav_test

import (
	"slices"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

func TestCheckLandings(t *testing.T) {
	g := nav.NewRouteGraph(realSystems(), append(gatesToJita, [2]int64{30000855, 30001324}))
	r, err := g.ValidateRoute(pfRequest(mb), []int64{30000142, 30001448, 30000855, 30001324})
	if err != nil {
		t.Fatal(err)
	}
	places := []nav.Place{
		{SystemID: 30000855, Name: "NL6V-7 Astrahus", TypeName: "Astrahus"},
		{SystemID: 30001324, Name: "PF-QHK Keepstar", TypeName: "Upwell Palatine Keepstar"},
	}
	// A carrier: NPC stations and Fortizars up, but not an Astrahus.
	got := nav.CheckLandings(g, r, 547, places)
	if len(got) != 3 {
		t.Fatalf("landings = %+v", got)
	}
	if got[0].System != "Hakonen" || !slices.Equal(got[0].DockAt, []string{"NPC station"}) || got[0].LighterGates != nil || got[0].Security.Display != "0.3" {
		t.Errorf("Hakonen = %+v", got[0])
	}
	if !got[1].NoDock || got[1].LighterGates == nil || *got[1].LighterGates != -1 {
		t.Errorf("NL6V-7 = %+v, want no dock (Astrahus) and no gate path from Hakonen", got[1])
	}
	if !slices.Equal(got[2].DockAt, []string{"NPC station", "PF-QHK Keepstar"}) || *got[2].LighterGates != 1 || !slices.Equal(got[2].CynoTypes, []string{"standard"}) {
		t.Errorf("PF-QHK = %+v", got[2])
	}
	// A jump freighter docks in the Astrahus.
	if jf := nav.CheckLandings(g, r, 902, places); jf[1].NoDock || !slices.Contains(jf[1].DockAt, "NL6V-7 Astrahus") {
		t.Errorf("JF at NL6V-7 = %+v", jf[1])
	}
}

func TestSharedCynoTypes(t *testing.T) {
	if got := nav.SharedCynoTypes([]int64{902, 902}); !slices.Equal(got, []string{"standard", "industrial"}) {
		t.Errorf("two JFs = %v", got)
	}
	if got := nav.SharedCynoTypes([]int64{902, 547}); !slices.Equal(got, []string{"standard"}) {
		t.Errorf("JF and carrier = %v, want standard only", got)
	}
}

func TestFuelFit(t *testing.T) {
	// A Nomad's 12,000 m³ fuel bay holds 400,000 isotopes at 0.03 m³.
	if got := nav.FuelFit(69500, 12000, 0.03); !got.FitsInFuelBay || got.FuelBayUnits != 400000 {
		t.Errorf("fits = %+v", got)
	}
	got := nav.FuelFit(450000, 12000, 0.03)
	if got.FitsInFuelBay || got.ExtraUnits != 50000 || got.ExtraM3 != 1500 {
		t.Errorf("overflow = %+v", got)
	}
}
