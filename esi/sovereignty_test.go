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

func TestSovereigntyCampaigns_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{
			"attackers_score": 0.5,
			"campaign_id": 12345,
			"constellation_id": 20000001,
			"defender_id": 99000001,
			"defender_score": 0.75,
			"event_type": "tcu_defense",
			"participants": [{"alliance_id": 99000002, "score": 0.25}],
			"solar_system_id": 30000001,
			"start_time": "2026-07-25T20:00:00Z",
			"structure_id": 1000000000001
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	campaigns, err := client.SovereigntyCampaigns(context.Background())
	if err != nil {
		t.Fatalf("SovereigntyCampaigns returned error: %v", err)
	}
	if len(campaigns) != 1 {
		t.Fatalf("campaigns = %+v, want one entry", campaigns)
	}
	c := campaigns[0]
	if c.CampaignId != 12345 || c.ConstellationId != 20000001 || c.SolarSystemId != 30000001 || c.StructureId != 1000000000001 || c.EventType != "tcu_defense" {
		t.Errorf("campaigns[0] = %+v, want CampaignId=12345 ConstellationId=20000001 SolarSystemId=30000001 StructureId=1000000000001 EventType=tcu_defense", c)
	}
	wantStart := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	if !c.StartTime.Equal(wantStart) {
		t.Errorf("StartTime = %v, want %v", c.StartTime, wantStart)
	}
	if c.AttackersScore == nil || *c.AttackersScore != 0.5 {
		t.Errorf("AttackersScore = %v, want 0.5", c.AttackersScore)
	}
	if c.DefenderId == nil || *c.DefenderId != 99000001 {
		t.Errorf("DefenderId = %v, want 99000001", c.DefenderId)
	}
	if c.DefenderScore == nil || *c.DefenderScore != 0.75 {
		t.Errorf("DefenderScore = %v, want 0.75", c.DefenderScore)
	}
	if len(c.Participants) != 1 || c.Participants[0].AllianceId != 99000002 || c.Participants[0].Score != 0.25 {
		t.Errorf("Participants = %+v, want one entry {AllianceId:99000002 Score:0.25}", c.Participants)
	}
}

func TestSovereigntyCampaigns_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.SovereigntyCampaigns(context.Background()); err == nil {
		t.Fatal("SovereigntyCampaigns returned nil error, want an error for HTTP 500")
	}
}

func TestSovereigntySystems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"solar_systems":[
			{"solar_system_id":30000001,"claim":{"alliance":{"alliance_id":99000001,"claimed_since":"2026-01-01T00:00:00Z","corporation_id":98000001,"development":{"activity_defense_multiplier":1.0,"industrial_level":2,"military_level":3,"strategic_level":1},"is_capital_system":true,"sovereignty_hub":{"id":1234567890,"vulnerability_window":{"end":"2026-07-25T22:00:00Z","start":"2026-07-25T20:00:00Z"}}}}},
			{"solar_system_id":30000002,"claim":{"faction":{"faction_id":500001}}},
			{"solar_system_id":30000003,"claim":{"unclaimed":true}}
		]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	systems, err := client.SovereigntySystems(context.Background())
	if err != nil {
		t.Fatalf("SovereigntySystems returned error: %v", err)
	}
	if len(systems) != 3 {
		t.Fatalf("systems = %+v, want three entries", systems)
	}

	s0 := systems[0]
	if s0.SolarSystemId != 30000001 {
		t.Errorf("systems[0].SolarSystemId = %d, want 30000001", s0.SolarSystemId)
	}
	if s0.Claim.Alliance == nil || s0.Claim.Alliance.Alliance == nil {
		t.Fatal("systems[0].Claim.Alliance = nil, want non-nil")
	}
	a := s0.Claim.Alliance.Alliance
	if a.AllianceId != 99000001 || a.CorporationId != 98000001 || !a.IsCapitalSystem {
		t.Errorf("alliance claim = %+v, want AllianceId=99000001 CorporationId=98000001 IsCapitalSystem=true", a)
	}
	wantClaimedSince := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !a.ClaimedSince.Equal(wantClaimedSince) {
		t.Errorf("ClaimedSince = %v, want %v", a.ClaimedSince, wantClaimedSince)
	}
	if a.Development.ActivityDefenseMultiplier != 1.0 || a.Development.IndustrialLevel != 2 || a.Development.MilitaryLevel != 3 || a.Development.StrategicLevel != 1 {
		t.Errorf("Development = %+v, want ActivityDefenseMultiplier=1.0 IndustrialLevel=2 MilitaryLevel=3 StrategicLevel=1", a.Development)
	}
	if a.SovereigntyHub.Id != 1234567890 {
		t.Errorf("SovereigntyHub.Id = %d, want 1234567890", a.SovereigntyHub.Id)
	}
	if a.SovereigntyHub.VulnerabilityWindow == nil {
		t.Fatal("SovereigntyHub.VulnerabilityWindow = nil, want non-nil")
	}
	wantVwStart := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	wantVwEnd := time.Date(2026, 7, 25, 22, 0, 0, 0, time.UTC)
	if !a.SovereigntyHub.VulnerabilityWindow.Start.Equal(wantVwStart) || !a.SovereigntyHub.VulnerabilityWindow.End.Equal(wantVwEnd) {
		t.Errorf("VulnerabilityWindow = %+v, want Start=%v End=%v", a.SovereigntyHub.VulnerabilityWindow, wantVwStart, wantVwEnd)
	}

	s1 := systems[1]
	if s1.SolarSystemId != 30000002 || s1.Claim.Faction == nil || s1.Claim.Faction.Faction == nil || s1.Claim.Faction.Faction.FactionId != 500001 {
		t.Errorf("systems[1] = %+v, want SolarSystemId=30000002 Claim.Faction.Faction.FactionId=500001", s1)
	}

	s2 := systems[2]
	if s2.SolarSystemId != 30000003 || s2.Claim.Unclaimed == nil || s2.Claim.Unclaimed.Unclaimed == nil || !*s2.Claim.Unclaimed.Unclaimed {
		t.Errorf("systems[2] = %+v, want SolarSystemId=30000003 Claim.Unclaimed.Unclaimed=true", s2)
	}
}

func TestSovereigntySystems_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.SovereigntySystems(context.Background()); err == nil {
		t.Fatal("SovereigntySystems returned nil error, want an error for HTTP 500")
	}
}
