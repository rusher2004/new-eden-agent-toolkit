package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

const legacyConfig = `{
  "characters": [{"id": 1, "name": "Main"}, {"id": 2, "name": "Alt"}],
  "facilityProfiles": [{"name": "Home Sotiyo", "kind": "upwell", "structureType": 35827, "systemId": 30001324, "notes": "measured 2026-11-17"}],
  "haulProfiles": [{"name": "JF Nomad", "shipTypeId": 28846, "pilotId": 1, "holdM3": 158400, "outbound": [30000142, 30001324], "prefs": {"optimize": "fuel"}}],
  "hubProfiles": [{"name": "Jita 4-4", "regionId": 10000002, "stationId": 60003760}],
  "analyticsDisabled": true
}`

func legacyStore(t *testing.T) (*config.Store, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(legacyConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	return config.NewStoreAt(filepath.Join(dir, "config.json")), dir
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestStore_MovesLegacySectionsToYAML(t *testing.T) {
	store, dir := legacyStore(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Characters) != 2 || len(cfg.FacilityProfiles) != 1 || len(cfg.HaulProfiles) != 1 || len(cfg.HubProfiles) != len(config.BuiltinHubs) || !cfg.AnalyticsDisabled {
		t.Fatalf("loaded = %+v", cfg)
	}
	prof := cfg.HaulProfiles[0]
	if prof.OutboundRoute != "JF Nomad outbound" || prof.Prefs.Optimize != "fuel" {
		t.Errorf("haul profile = %+v", prof)
	}
	r, ok := cfg.Route("JF Nomad outbound")
	if !ok || len(r.Waypoints) != 2 || r.Waypoints[1].ID != 30001324 {
		t.Errorf("route = %+v, %v", r, ok)
	}
	if f, _ := cfg.FacilityProfile("home sotiyo"); f.Notes != "measured 2026-11-17" {
		t.Errorf("facility notes lost: %+v", f)
	}

	var left map[string]any
	if err := json.Unmarshal([]byte(read(t, filepath.Join(dir, "config.json"))), &left); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"characters", "facilityProfiles", "haulProfiles"} {
		if _, ok := left[k]; ok {
			t.Errorf("config.json still has %s", k)
		}
	}
	if left["analyticsDisabled"] != true || left["hubProfiles"] != nil || len(left["offeredHubIds"].([]any)) != len(config.BuiltinHubs) {
		t.Errorf("config.json = %v, want analytics kept, hubs moved and built-ins recorded as offered", left)
	}
	if h, _ := cfg.HubProfile(""); h.Name != "Jita 4-4" || h.Region.ID != 10000002 {
		t.Errorf("default hub = %+v, want the user's own Jita 4-4 first", h)
	}
	chars := read(t, filepath.Join(dir, "characters.yaml"))
	if !strings.HasPrefix(chars, "# Characters logged in to neat.") || !strings.Contains(chars, "- id: 1\n  name: Main\n") {
		t.Errorf("characters.yaml:\n%s", chars)
	}
	if hp := read(t, filepath.Join(dir, "haul_profiles.yaml")); !strings.Contains(hp, "outbound_route: JF Nomad outbound") || !strings.Contains(hp, "ship_type_id: 28846") {
		t.Errorf("haul_profiles.yaml:\n%s", hp)
	}

	again, err := store.Load()
	if err != nil || len(again.Characters) != 2 || len(again.Routes) != 1 {
		t.Errorf("second load = %+v, %v", again, err)
	}
}

func TestStore_ExistingYAMLWinsOverLegacy(t *testing.T) {
	store, dir := legacyStore(t)
	if err := os.WriteFile(filepath.Join(dir, "characters.yaml"), []byte("- id: 9\n  name: Already here\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0].ID != 9 {
		t.Errorf("characters = %+v, want the YAML file's", cfg.Characters)
	}
}

func TestStore_SaveKeepsHandEditsAndReportsChanges(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStoreAt(filepath.Join(dir, "config.json"))
	hand := "# My characters\n- id: 1\n  name: Main\n  notes: |\n    My main. Written by hand.\n  fleet_role: fc # my own key\n- id: 2\n  name: Alt\n"
	if err := os.WriteFile(filepath.Join(dir, "characters.yaml"), []byte(hand), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Characters[0].Notes != "My main. Written by hand.\n" {
		t.Errorf("notes = %q", cfg.Characters[0].Notes)
	}
	if _, err := cfg.SetCharacterAccount(2, "Alts"); err != nil {
		t.Fatal(err)
	}
	changes, err := store.SaveChanges(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0] != (config.Change{File: "characters.yaml", Entry: "2", Field: "account", New: "Alts"}) {
		t.Errorf("changes = %+v", changes)
	}
	got := read(t, filepath.Join(dir, "characters.yaml"))
	for _, want := range []string{"# My characters\n", "    My main. Written by hand.\n", "  fleet_role: fc # my own key\n", "- id: 2\n  name: Alt\n  account: Alts\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("characters.yaml is missing %q:\n%s", want, got)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "facilities.yaml")); !os.IsNotExist(err) {
		t.Error("saving created an empty facilities.yaml")
	}
}

func TestStore_BrokenYAMLFailsLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStoreAt(filepath.Join(dir, "config.json"))
	broken := "- id: 1\n  name: [oops\n"
	if err := os.WriteFile(filepath.Join(dir, "characters.yaml"), []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil || !strings.Contains(err.Error(), "characters.yaml isn't valid YAML") {
		t.Errorf("load err = %v", err)
	}
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 1, Name: "X"}}}); err == nil {
		t.Error("save over a broken file: want an error")
	}
	if got := read(t, filepath.Join(dir, "characters.yaml")); got != broken {
		t.Errorf("broken file changed: %q", got)
	}
}

func TestStore_BuiltinHubsSeededOnceAndDeletionsStick(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStoreAt(filepath.Join(dir, "config.json"))
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.HubProfiles) != len(config.BuiltinHubs) || cfg.HubProfiles[0].StationID != 60003760 {
		t.Fatalf("hubs = %+v", cfg.HubProfiles)
	}
	if !strings.HasPrefix(read(t, filepath.Join(dir, "hubs.yaml")), "# Trade hubs to buy and sell at.") {
		t.Error("hubs.yaml has no header")
	}
	if err := cfg.DeleteHubProfile("Amamake"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	again, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(again.HubProfiles) != len(config.BuiltinHubs)-1 {
		t.Errorf("a deleted built-in hub came back: %d hubs", len(again.HubProfiles))
	}
	if _, ok := again.HubProfile("Amamake"); ok {
		t.Error("Amamake found after deleting it")
	}
	if h, ok := again.HubProfile("Perimeter - Tranquility Trading Tower"); !ok || !h.IsStructure() {
		t.Errorf("structure hub = %+v, %v", h, ok)
	}
	if h, ok := again.HubProfile("amarr"); !ok || h.StationID != 60008494 {
		t.Errorf("lookup by system name = %+v, %v", h, ok)
	}
}

func TestStore_LegacyBuiltinHubKeepsItsSystem(t *testing.T) {
	store, _ := legacyStore(t)
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	h, ok := cfg.HubProfile("Jita")
	if !ok || h.Name != "Jita 4-4" || h.System == nil || h.System.ID != 30000142 || h.Region.Name != "The Forge" || h.Tier != "major" {
		t.Errorf("migrated Jita 4-4 = %+v, %v; want the user's name with the built-in hub's system, region and tier", h, ok)
	}
}

func TestConfig_HubBySharedSystemNameIsAmbiguous(t *testing.T) {
	cfg := config.Config{HubProfiles: slices.Clone(config.BuiltinHubs)}
	if h, ok := cfg.HubProfile("Perimeter"); ok {
		t.Errorf("Perimeter found %s, though two hubs are there", h.Name)
	}
	if err := cfg.DeleteHubProfile("Perimeter"); err == nil || !strings.Contains(err.Error(), "name one") {
		t.Errorf("delete Perimeter = %v, want an ambiguity error", err)
	}
	if len(cfg.HubProfiles) != len(config.BuiltinHubs) {
		t.Errorf("an ambiguous delete removed a hub")
	}
}
