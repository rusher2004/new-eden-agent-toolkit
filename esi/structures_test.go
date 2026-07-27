package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterMercenaryDens_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"mercenary_dens":[{"id":1001,"planet_id":40000001}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	dens, err := client.CharacterMercenaryDens(context.Background())
	if err != nil {
		t.Fatalf("CharacterMercenaryDens returned error: %v", err)
	}
	if gotPath != "/characters/12345/structures/mercenary-dens" {
		t.Errorf("path = %q", gotPath)
	}
	if len(dens) != 1 || dens[0].Id != 1001 || dens[0].PlanetId != 40000001 {
		t.Errorf("dens = %+v, want one entry {Id:1001 PlanetId:40000001}", dens)
	}
}

func TestCharacterMercenaryDens_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterMercenaryDens(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterMercenaryDenDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": 1001, "type_id": 85230, "state": "online",
			"evolution": {
				"anarchy": {"amount": 10, "level": "low"},
				"development": {"amount": 20, "level": "medium"}
			},
			"infomorphs": {"amount": 3},
			"reinforcement_timer": {"end": "2026-08-01T00:00:00Z"},
			"skyhook": {"corporation_id": 98000001, "id": 2001, "planet_id": 40000001}
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CharacterMercenaryDenDetail(context.Background(), 1001)
	if err != nil {
		t.Fatalf("CharacterMercenaryDenDetail returned error: %v", err)
	}
	if gotPath != "/characters/12345/structures/mercenary-dens/1001" {
		t.Errorf("path = %q", gotPath)
	}
	if detail.Id != 1001 || detail.TypeId != 85230 || detail.State != "online" {
		t.Errorf("detail = %+v, want Id=1001 TypeId=85230 State=online", detail)
	}
	if detail.Evolution.Anarchy.Amount != 10 || detail.Evolution.Anarchy.Level != "low" {
		t.Errorf("Evolution.Anarchy = %+v, want Amount=10 Level=low", detail.Evolution.Anarchy)
	}
	if detail.Evolution.Development.Amount != 20 || detail.Evolution.Development.Level != "medium" {
		t.Errorf("Evolution.Development = %+v, want Amount=20 Level=medium", detail.Evolution.Development)
	}
	if detail.Infomorphs.Amount != 3 {
		t.Errorf("Infomorphs.Amount = %v, want 3", detail.Infomorphs.Amount)
	}
	wantEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if detail.ReinforcementTimer == nil || !detail.ReinforcementTimer.End.Equal(wantEnd) {
		t.Errorf("ReinforcementTimer = %v, want End=%v", detail.ReinforcementTimer, wantEnd)
	}
	if detail.Skyhook.CorporationId != 98000001 || detail.Skyhook.Id != 2001 || detail.Skyhook.PlanetId != 40000001 {
		t.Errorf("Skyhook = %+v, want CorporationId=98000001 Id=2001 PlanetId=40000001", detail.Skyhook)
	}
}

func TestCharacterMercenaryDenDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterMercenaryDenDetail(context.Background(), 1001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSkyhooks_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"skyhooks":[{"id":2001,"planet_id":40000001}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	hooks, err := client.CorporationSkyhooks(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationSkyhooks returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/structures/skyhooks" {
		t.Errorf("path = %q", gotPath)
	}
	if len(hooks) != 1 || hooks[0].Id != 2001 || hooks[0].PlanetId != 40000001 {
		t.Errorf("hooks = %+v, want one entry {Id:2001 PlanetId:40000001}", hooks)
	}
}

func TestCorporationSkyhooks_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CorporationSkyhooks(context.Background(), 98000001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSkyhookDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": 2001, "planet_id": 40000001, "is_active": true, "state": "gathering_workforce",
			"effective_workforce": 500,
			"reagents": [
				{"last_cycle": "2026-07-01T00:00:00Z", "secured_stock": 100, "type_id": 16274, "unsecured_stock": 50}
			],
			"reinforcement_timer": {"end": "2026-08-01T00:00:00Z"},
			"theft_vulnerability": {"start": "2026-07-15T00:00:00Z", "end": "2026-07-16T00:00:00Z"}
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CorporationSkyhookDetail(context.Background(), 98000001, 2001)
	if err != nil {
		t.Fatalf("CorporationSkyhookDetail returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/structures/skyhooks/2001" {
		t.Errorf("path = %q, want skyhookId in path segment after corporationId's structures/skyhooks", gotPath)
	}
	if detail.Id != 2001 || detail.PlanetId != 40000001 || !detail.IsActive || detail.State != "gathering_workforce" {
		t.Errorf("detail = %+v, want Id=2001 PlanetId=40000001 IsActive=true State=gathering_workforce", detail)
	}
	if detail.EffectiveWorkforce == nil || *detail.EffectiveWorkforce != 500 {
		t.Errorf("EffectiveWorkforce = %v, want 500", detail.EffectiveWorkforce)
	}
	if len(detail.Reagents) != 1 {
		t.Fatalf("Reagents = %+v, want one entry", detail.Reagents)
	}
	r := detail.Reagents[0]
	if r.SecuredStock != 100 || r.TypeId != 16274 || r.UnsecuredStock != 50 {
		t.Errorf("Reagents[0] = %+v, want SecuredStock=100 TypeId=16274 UnsecuredStock=50", r)
	}
	if detail.ReinforcementTimer == nil {
		t.Error("ReinforcementTimer = nil, want set")
	}
	if detail.TheftVulnerability == nil {
		t.Error("TheftVulnerability = nil, want set")
	}
}

func TestCorporationSkyhookDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CorporationSkyhookDetail(context.Background(), 98000001, 2001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSovereigntyHubs_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"sovereignty_hubs":[{"id":3001,"solar_system_id":30000142}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	hubs, err := client.CorporationSovereigntyHubs(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationSovereigntyHubs returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/structures/sovereignty-hubs" {
		t.Errorf("path = %q", gotPath)
	}
	if len(hubs) != 1 || hubs[0].Id != 3001 || hubs[0].SolarSystemId != 30000142 {
		t.Errorf("hubs = %+v, want one entry {Id:3001 SolarSystemId:30000142}", hubs)
	}
}

func TestCorporationSovereigntyHubs_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CorporationSovereigntyHubs(context.Background(), 98000001); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorporationSovereigntyHubDetail_ParsesResponse_ExportVariants(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": 3001, "solar_system_id": 30000142, "fuel_access_list_id": 5001,
			"reagent_bay": {
				"last_updated": "2026-07-20T00:00:00Z",
				"reagents": [{"amount": 1000, "burning_per_hour": 10, "type_id": 16274}]
			},
			"resources": {
				"power": {"allocated": 100, "available": 200},
				"workforce": {"allocated": 50, "available": 75}
			},
			"upgrades": [{"power_state": "online", "type_id": 81000}],
			"vulnerability_window": {"start": "2026-07-25T00:00:00Z", "end": "2026-07-25T02:00:00Z"},
			"workforce_transport": {
				"configuration": {"export": {"amount": 300, "solar_system_id": 30000144}},
				"state": {"export": {"amount": 150, "solar_system_id": 30000144}}
			}
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CorporationSovereigntyHubDetail(context.Background(), 98000001, 3001)
	if err != nil {
		t.Fatalf("CorporationSovereigntyHubDetail returned error: %v", err)
	}
	if gotPath != "/corporations/98000001/structures/sovereignty-hubs/3001" {
		t.Errorf("path = %q, want sovereigntyHubId in path segment after corporationId's structures/sovereignty-hubs", gotPath)
	}
	if detail.Id != 3001 || detail.SolarSystemId != 30000142 {
		t.Errorf("detail = %+v, want Id=3001 SolarSystemId=30000142", detail)
	}
	if detail.FuelAccessListId == nil || *detail.FuelAccessListId != 5001 {
		t.Errorf("FuelAccessListId = %v, want 5001", detail.FuelAccessListId)
	}
	if len(detail.ReagentBay.Reagents) != 1 || detail.ReagentBay.Reagents[0].Amount != 1000 || detail.ReagentBay.Reagents[0].BurningPerHour != 10 {
		t.Errorf("ReagentBay = %+v, want one reagent Amount=1000 BurningPerHour=10", detail.ReagentBay)
	}
	if detail.Resources.Power.Allocated != 100 || detail.Resources.Power.Available != 200 {
		t.Errorf("Resources.Power = %+v, want Allocated=100 Available=200", detail.Resources.Power)
	}
	if detail.Resources.Workforce.Allocated != 50 || detail.Resources.Workforce.Available != 75 {
		t.Errorf("Resources.Workforce = %+v, want Allocated=50 Available=75", detail.Resources.Workforce)
	}
	if len(detail.Upgrades) != 1 || detail.Upgrades[0].PowerState != "online" || detail.Upgrades[0].TypeId != 81000 {
		t.Errorf("Upgrades = %+v, want one entry PowerState=online TypeId=81000", detail.Upgrades)
	}
	if detail.VulnerabilityWindow == nil {
		t.Fatal("VulnerabilityWindow = nil, want set")
	}
	wantStart := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	if !detail.VulnerabilityWindow.Start.Equal(wantStart) {
		t.Errorf("VulnerabilityWindow.Start = %v, want %v", detail.VulnerabilityWindow.Start, wantStart)
	}

	cfg := detail.WorkforceTransport.Configuration
	if cfg.Export == nil || cfg.Export.Export == nil || cfg.Export.Export.Amount != 300 {
		t.Errorf("Configuration.Export = %+v, want Amount=300", cfg.Export)
	}
	if cfg.Export.Export.SolarSystemId == nil || *cfg.Export.Export.SolarSystemId != 30000144 {
		t.Errorf("Configuration.Export.SolarSystemId = %v, want 30000144", cfg.Export.Export.SolarSystemId)
	}
	if cfg.ModelImport != nil || cfg.Transit != nil {
		t.Errorf("Configuration should only have Export set, got ModelImport=%+v Transit=%+v", cfg.ModelImport, cfg.Transit)
	}

	st := detail.WorkforceTransport.State
	if st.Export1 == nil || st.Export1.Export == nil || st.Export1.Export.Amount == nil || *st.Export1.Export.Amount != 150 {
		t.Errorf("State.Export1 = %+v, want Amount=150", st.Export1)
	}
	if st.Import1 != nil || st.Transit1 != nil {
		t.Errorf("State should only have Export1 set, got Import1=%+v Transit1=%+v", st.Import1, st.Transit1)
	}
}

func TestCorporationSovereigntyHubDetail_ParsesResponse_ImportVariants(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": 3001, "solar_system_id": 30000142,
			"reagent_bay": {"last_updated": "2026-07-20T00:00:00Z", "reagents": []},
			"resources": {
				"power": {"allocated": 0, "available": 0},
				"workforce": {"allocated": 0, "available": 0}
			},
			"upgrades": [],
			"workforce_transport": {
				"configuration": {"import": {"sources": [{"solar_system_id": 30000144}]}},
				"state": {"import": {"sources": [{"amount": 500, "solar_system_id": 30000144}]}}
			}
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CorporationSovereigntyHubDetail(context.Background(), 98000001, 3001)
	if err != nil {
		t.Fatalf("CorporationSovereigntyHubDetail returned error: %v", err)
	}

	cfg := detail.WorkforceTransport.Configuration
	if cfg.ModelImport == nil || cfg.ModelImport.Import == nil || len(cfg.ModelImport.Import.Sources) != 1 || cfg.ModelImport.Import.Sources[0].SolarSystemId != 30000144 {
		t.Errorf("Configuration.ModelImport = %+v, want one source SolarSystemId=30000144", cfg.ModelImport)
	}
	if cfg.Export != nil || cfg.Transit != nil {
		t.Errorf("Configuration should only have ModelImport set, got Export=%+v Transit=%+v", cfg.Export, cfg.Transit)
	}

	st := detail.WorkforceTransport.State
	if st.Import1 == nil || st.Import1.Import == nil || len(st.Import1.Import.Sources) != 1 {
		t.Fatalf("State.Import1 = %+v, want one source", st.Import1)
	}
	if st.Import1.Import.Sources[0].Amount != 500 || st.Import1.Import.Sources[0].SolarSystemId != 30000144 {
		t.Errorf("State.Import1.Import.Sources[0] = %+v, want Amount=500 SolarSystemId=30000144", st.Import1.Import.Sources[0])
	}
	if st.Export1 != nil || st.Transit1 != nil {
		t.Errorf("State should only have Import1 set, got Export1=%+v Transit1=%+v", st.Export1, st.Transit1)
	}
}

func TestCorporationSovereigntyHubDetail_ParsesResponse_TransitVariants(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": 3001, "solar_system_id": 30000142,
			"reagent_bay": {"last_updated": "2026-07-20T00:00:00Z", "reagents": []},
			"resources": {
				"power": {"allocated": 0, "available": 0},
				"workforce": {"allocated": 0, "available": 0}
			},
			"upgrades": [],
			"workforce_transport": {
				"configuration": {"transit": true},
				"state": {"transit": false}
			}
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CorporationSovereigntyHubDetail(context.Background(), 98000001, 3001)
	if err != nil {
		t.Fatalf("CorporationSovereigntyHubDetail returned error: %v", err)
	}

	cfg := detail.WorkforceTransport.Configuration
	if cfg.Transit == nil {
		t.Fatal("Configuration.Transit = nil, want set")
	}
	got := cfg.Transit.Transit.Get()
	if got == nil || *got != true {
		t.Errorf("Configuration.Transit.Transit = %v, want true", got)
	}
	if cfg.Export != nil || cfg.ModelImport != nil {
		t.Errorf("Configuration should only have Transit set, got Export=%+v ModelImport=%+v", cfg.Export, cfg.ModelImport)
	}

	st := detail.WorkforceTransport.State
	if st.Transit1 == nil {
		t.Fatal("State.Transit1 = nil, want set")
	}
	gotState := st.Transit1.Transit.Get()
	if gotState == nil || *gotState != false {
		t.Errorf("State.Transit1.Transit = %v, want false", gotState)
	}
	if st.Export1 != nil || st.Import1 != nil {
		t.Errorf("State should only have Transit1 set, got Export1=%+v Import1=%+v", st.Export1, st.Import1)
	}
}

func TestCorporationSovereigntyHubDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 12345, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CorporationSovereigntyHubDetail(context.Background(), 98000001, 3001); err == nil {
		t.Fatal("expected error")
	}
}
