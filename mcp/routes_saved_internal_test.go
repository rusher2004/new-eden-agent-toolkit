package mcp

import (
	"context"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
)

func TestSaveRoute_NamesSystemsAndKeepsNotes(t *testing.T) {
	store, p := industryTestEnv(t)
	save := saveRouteHandler("test", p.openESI, p.openPublicESI, store, p.sde)
	_, out, err := save(context.Background(), nil, SaveRouteInput{Name: "Jita to PF", Systems: []string{"Jita", "Hakonen", "NL6V-7", "PF-QHK"}, ShipClass: "jump_freighter", Notes: new("Hakonen is quiet.")})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Route.Waypoints) != 4 || out.Route.Waypoints[1] != (config.RouteWaypoint{ID: 30001448, Name: "Hakonen"}) || out.Route.Notes != "Hakonen is quiet." {
		t.Errorf("route = %+v", out.Route)
	}
	if len(out.Changes) != 1 || out.Changes[0].New != "added" || out.Changes[0].File != "routes.yaml" {
		t.Errorf("changes = %+v", out.Changes)
	}

	_, again, err := save(context.Background(), nil, SaveRouteInput{Name: "jita to pf", Systems: []string{"Jita", "NL6V-7", "PF-QHK"}})
	if err != nil {
		t.Fatal(err)
	}
	if again.Route.Notes != "Hakonen is quiet." || again.Route.ShipClass != "jump_freighter" {
		t.Errorf("re-saving without notes or class lost them: %+v", again.Route)
	}
	if len(again.Changes) != 1 || again.Changes[0].Field != "waypoints" {
		t.Errorf("changes = %+v", again.Changes)
	}

	if _, _, err := save(context.Background(), nil, SaveRouteInput{Name: "Bad", Systems: []string{"Jita"}}); err == nil {
		t.Error("a one-system route: want an error")
	}
	_, list, _ := listRoutesHandler(store)(context.Background(), nil, struct{}{})
	if len(list.Routes) != 1 {
		t.Errorf("routes = %+v", list.Routes)
	}
}

func TestSaveHaulProfile_PinsRoutesAndKeepsNotes(t *testing.T) {
	store, p := industryTestEnv(t)
	save := saveHaulProfileHandler("test", p.openESI, p.openPublicESI, store, p.sde)
	_, prof, err := save(context.Background(), nil, SaveHaulProfileInput{Name: "JF Nomad", Ship: "Nomad", PilotID: int64(testPilot), HoldM3: 158400,
		Outbound: []string{"Jita", "Hakonen", "NL6V-7", "PF-QHK"}, Notes: new("Only under 300M ISK through Jita.")})
	if err != nil {
		t.Fatal(err)
	}
	if prof.OutboundRoute != "JF Nomad outbound" || prof.ShipName != "Nomad" || prof.PilotName != "Test Pilot" {
		t.Errorf("profile = %+v", prof)
	}
	_, prof, err = save(context.Background(), nil, SaveHaulProfileInput{Name: "JF Nomad", Ship: "Nomad", PilotID: int64(testPilot), HoldM3: 160000, OutboundRoute: "JF Nomad outbound"})
	if err != nil {
		t.Fatal(err)
	}
	if prof.Notes != "Only under 300M ISK through Jita." || prof.OutboundRoute != "JF Nomad outbound" {
		t.Errorf("re-saved profile = %+v", prof)
	}
	if _, _, err := save(context.Background(), nil, SaveHaulProfileInput{Name: "X", Ship: "Nomad", PilotID: int64(testPilot), HoldM3: 1, OutboundRoute: "Nope"}); err == nil {
		t.Error("pinning a route that isn't saved: want an error")
	}
}

func TestSaveFacilityProfile_KeepsNotesWhenOmitted(t *testing.T) {
	store, p := industryTestEnv(t)
	save := saveFacilityProfileHandler("test", p.openESI, p.openPublicESI, store, p.sde)
	in := SaveFacilityProfileInput{Name: "Home Sotiyo", Kind: industry.KindUpwell, StructureType: "Sotiyo", System: "PF-QHK", Notes: new("fee measured 2026-11-17")}
	if _, _, err := save(context.Background(), nil, in); err != nil {
		t.Fatal(err)
	}
	in.Notes = nil
	if _, _, err := save(context.Background(), nil, in); err != nil {
		t.Fatal(err)
	}
	cfg, _ := store.Load()
	f, _ := cfg.FacilityProfile("Home Sotiyo")
	if f.Notes != "fee measured 2026-11-17" || f.SystemName != "PF-QHK" {
		t.Errorf("facility = %+v", f)
	}
}
