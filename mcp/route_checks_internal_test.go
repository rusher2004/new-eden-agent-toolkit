package mcp

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde/sdetest"
)

const (
	jitaSys  = int64(30000142)
	hakonen  = int64(30001448)
	nl6v7    = int64(30000855)
	pfqhk    = int64(30001324)
	astrahus = int64(1049000000001)
)

func routeChecksEnv(t *testing.T) (*config.Store, *planner) {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{
		Characters: []config.Character{{ID: testPilot, Name: "Test Pilot"}},
		Structures: []config.Structure{{ID: astrahus, Name: "NL6V-7 Astrahus", Kind: config.StructureUpwell, System: &config.NamedID{ID: nl6v7, Name: "NL6V-7"}, Type: &config.NamedID{ID: 35832, Name: "Astrahus"}}},
	}); err != nil {
		t.Fatal(err)
	}
	s := sdetest.NewStore(t, "testdata/industry_fixture.sql", "testdata/readiness_fixture.sql")
	return store, newPlanner("test", openFakeESI(fakeIndustryESI()), openFakePublicESI(fakeIndustryESI()), store, s)
}

var jfPath = []int64{jitaSys, hakonen, nl6v7, pfqhk}

func TestPlanRoute_LandingsAndFuel(t *testing.T) {
	_, p := routeChecksEnv(t)
	nomad := config.HaulProfile{Name: "JF", ShipTypeID: 28846, PilotID: int64(testPilot), HoldM3: 158400}
	out, _, _, err := p.planRoute(context.Background(), nomad, jitaSys, pfqhk, jfPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(out.CynoTypes, []string{"standard", "industrial"}) || len(out.Checks) != 1 {
		t.Fatalf("cyno %v checks %+v", out.CynoTypes, out.Checks)
	}
	landings := out.Checks[0].Landings
	if len(landings) != 3 || landings[1].System != "NL6V-7" || !slices.Equal(landings[1].DockAt, []string{"NL6V-7 Astrahus"}) || landings[1].NoDock {
		t.Errorf("landings = %+v", landings)
	}
	if fuel := out.Checks[0].Fuel; fuel.RouteFuel != out.Routes[0].Fuel || fuel.FuelBayM3 != 12000 || !fuel.FitsInFuelBay {
		t.Errorf("fuel = %+v", fuel)
	}
}

func TestHaulPlanWith_TwoJumpFreighters(t *testing.T) {
	store, p := routeChecksEnv(t)
	cfg, _ := store.Load()
	cfg.UpsertRoute(config.Route{Name: "JF out", Waypoints: []config.RouteWaypoint{{ID: jitaSys}, {ID: hakonen}, {ID: nl6v7}, {ID: pfqhk}}})
	a := config.HaulProfile{Name: "JF A", ShipTypeID: 28846, PilotID: int64(testPilot), HoldM3: 158400, OutboundRoute: "JF out"}
	b := config.HaulProfile{Name: "JF B", ShipTypeID: 28846, PilotID: int64(testPilot), HoldM3: 100000}
	cfg.UpsertHaulProfile(a)
	cfg.UpsertHaulProfile(b)
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	items := []cargo.Cargo{{TypeID: 34, Name: "Tritanium", Quantity: 40000000, UnitM3: 0.01, Dest: "PF-QHK"}}
	hub, _ := p.hub("")
	out, err := p.haulPlanWith(context.Background(), []config.HaulProfile{a, b}, hub, jitaSys, pfqhk, items, map[int64]float64{34: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Hulls) != 2 || !slices.Equal(out.SharedCynoTypes, []string{"standard", "industrial"}) || len(out.Loads) != 2 {
		t.Fatalf("hulls %+v cyno %v loads %d", out.Hulls, out.SharedCynoTypes, len(out.Loads))
	}
	for _, l := range out.Loads {
		if len(l.ByHull) != 2 || l.ByHull[0].Load.M3 > 158400+1e-6 || l.ByHull[1].Load.M3 > 100000+1e-6 {
			t.Errorf("load %d split = %+v", l.Number, l.ByHull)
		}
		if strings.Count(l.Multibuy, "Hydrogen Isotopes") != 1 {
			t.Errorf("load %d multibuy should merge both hulls' fuel: %q", l.Number, l.Multibuy)
		}
	}
}

func TestHaulPlanWith_CarrierCantFlyAHighsecRoute(t *testing.T) {
	_, p := routeChecksEnv(t)
	nomad := config.HaulProfile{Name: "JF", ShipTypeID: 28846, PilotID: int64(testPilot), HoldM3: 158400}
	archon := config.HaulProfile{Name: "Carrier", ShipTypeID: 23757, PilotID: int64(testPilot), HoldM3: 1000}
	hub, _ := p.hub("")
	items := []cargo.Cargo{{TypeID: 34, Name: "Tritanium", Quantity: 1000, UnitM3: 0.01, Dest: "PF-QHK"}}
	if _, err := p.haulPlanWith(context.Background(), []config.HaulProfile{nomad, archon}, hub, jitaSys, pfqhk, items, nil); err == nil || !strings.Contains(err.Error(), "Carrier can't fly the same outbound route") {
		t.Errorf("err = %v, want the carrier refused (it can't jump out of highsec Jita)", err)
	}
}
