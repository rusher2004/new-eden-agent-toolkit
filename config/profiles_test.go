package config_test

import (
	"path/filepath"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestProfiles_RoundTripKeepsCharacters(t *testing.T) {
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))
	cfg := config.Config{Characters: []config.Character{{ID: 1, Name: "Test Pilot"}}}
	cfg.UpsertFacilityProfile(config.FacilityProfile{Name: "Home Sotiyo", Kind: "upwell", StructureType: 35827, SystemID: 30001324, RigTypeIDs: []int64{37180, 43704}, Services: []string{"manufacturing", "capital_shipyard"}})
	cfg.UpsertHaulProfile(config.HaulProfile{Name: "JF Nomad", ShipTypeID: 28846, PilotID: 90000001, HoldM3: 158400, Prefs: config.RoutePrefs{Optimize: "fuel", AvoidSystems: []int64{30005196}}})
	cfg.UpsertHubProfile(config.HubProfile{Name: "My Jita", Region: config.NamedID{ID: 10000002}, StationID: 60003760})
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Characters) != 1 || len(got.FacilityProfiles) != 1 || len(got.HaulProfiles) != 1 || len(got.HubProfiles) != len(config.BuiltinHubs) || got.HubProfiles[0].Name != "My Jita" {
		t.Fatalf("loaded = %+v", got)
	}
	p, ok := got.FacilityProfile("home sotiyo")
	if !ok || len(p.RigTypeIDs) != 2 {
		t.Errorf("case-insensitive lookup = %+v, %v", p, ok)
	}
	if h, _ := got.HaulProfile("JF Nomad"); h.Prefs.AvoidSystems[0] != 30005196 {
		t.Errorf("haul prefs = %+v", h.Prefs)
	}
}

func TestProfiles_UpsertReplacesAndDelete(t *testing.T) {
	var cfg config.Config
	cfg.UpsertFacilityProfile(config.FacilityProfile{Name: "A", FacilityTax: 0.01})
	cfg.UpsertFacilityProfile(config.FacilityProfile{Name: "a", FacilityTax: 0.02})
	if len(cfg.FacilityProfiles) != 1 || cfg.FacilityProfiles[0].FacilityTax != 0.02 {
		t.Errorf("upsert = %+v", cfg.FacilityProfiles)
	}
	if err := cfg.DeleteFacilityProfile("missing"); err == nil {
		t.Error("deleting a missing profile succeeded")
	}
	if err := cfg.DeleteFacilityProfile("A"); err != nil || len(cfg.FacilityProfiles) != 0 {
		t.Errorf("delete = %v, %+v", err, cfg.FacilityProfiles)
	}
}

func TestHubProfile_DefaultsToJita(t *testing.T) {
	var cfg config.Config
	h, ok := cfg.HubProfile("")
	if !ok || h.StationID != 60003760 {
		t.Errorf("default hub = %+v", h)
	}
	if _, ok := cfg.HubProfile("Amarr"); ok {
		t.Error("unknown hub found")
	}
}
