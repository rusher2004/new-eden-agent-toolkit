package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"achievement_score":100,"birthday":"2020-01-01T00:00:00Z","bloodline_id":1,"corporation_id":98000001,"gender":"male","name":"Test Character","race_id":1}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	detail, err := client.CharacterDetail(context.Background(), 998877)
	if err != nil {
		t.Fatalf("CharacterDetail returned error: %v", err)
	}
	if detail.Name != "Test Character" || detail.CorporationId != 98000001 {
		t.Errorf("detail = %+v, want Name=\"Test Character\" CorporationId=98000001", detail)
	}
	if detail.AllianceId != nil {
		t.Errorf("detail.AllianceId = %v, want nil", detail.AllianceId)
	}
}

func TestCharacterDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterDetail(context.Background(), 998877); err == nil {
		t.Fatal("CharacterDetail returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterAffiliations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"character_id":998877,"corporation_id":98000001}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	affs, err := client.CharacterAffiliations(context.Background(), []int64{998877})
	if err != nil {
		t.Fatalf("CharacterAffiliations returned error: %v", err)
	}
	if len(affs) != 1 || affs[0].CharacterId != 998877 || affs[0].CorporationId != 98000001 {
		t.Errorf("affs = %+v, want one entry {CharacterId:998877 CorporationId:98000001}", affs)
	}
}

func TestCharacterCorporationHistory_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"corporation_id":98000001,"record_id":1,"start_date":"2020-01-01T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	hist, err := client.CharacterCorporationHistory(context.Background(), 998877)
	if err != nil {
		t.Fatalf("CharacterCorporationHistory returned error: %v", err)
	}
	if len(hist) != 1 || hist[0].CorporationId != 98000001 {
		t.Errorf("hist = %+v, want one entry with CorporationId=98000001", hist)
	}
}

func TestCharacterPortrait_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"px64x64":"https://images.evetech.net/a.png"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	portrait, err := client.CharacterPortrait(context.Background(), 998877)
	if err != nil {
		t.Fatalf("CharacterPortrait returned error: %v", err)
	}
	if portrait.Px64x64 == nil || *portrait.Px64x64 != "https://images.evetech.net/a.png" {
		t.Errorf("portrait.Px64x64 = %v, want a pointer to the URL", portrait.Px64x64)
	}
}

func TestCharacterAgentsResearch_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"agent_id":1,"points_per_day":10.5,"remainder_points":2.5,"skill_type_id":11450,"started_at":"2020-01-01T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	agents, err := client.CharacterAgentsResearch(context.Background())
	if err != nil {
		t.Fatalf("CharacterAgentsResearch returned error: %v", err)
	}
	if len(agents) != 1 || agents[0].AgentId != 1 {
		t.Errorf("agents = %+v, want one entry with AgentId=1", agents)
	}
}

func TestCharacterBlueprints_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "4")
		w.Write([]byte(`[{"item_id":1,"location_flag":"Hangar","location_id":60003760,"material_efficiency":10,"quantity":1,"runs":-1,"time_efficiency":20,"type_id":950}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bps, totalPages, err := client.CharacterBlueprints(context.Background(), 3)
	if err != nil {
		t.Fatalf("CharacterBlueprints returned error: %v", err)
	}
	if gotQuery != "page=3" {
		t.Errorf("query = %q, want page=3", gotQuery)
	}
	if len(bps) != 1 || bps[0].TypeId != 950 || totalPages != 4 {
		t.Errorf("bps = %+v, totalPages = %d, want one entry with TypeId=950 totalPages=4", bps, totalPages)
	}
}

func TestCharacterBlueprints_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterBlueprints(context.Background(), 0); err != nil {
		t.Fatalf("CharacterBlueprints returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page omitted)", gotQuery)
	}
}

func TestCharacterFatigue_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	fatigue, err := client.CharacterFatigue(context.Background())
	if err != nil {
		t.Fatalf("CharacterFatigue returned error: %v", err)
	}
	if fatigue.LastJumpDate != nil {
		t.Errorf("fatigue.LastJumpDate = %v, want nil", fatigue.LastJumpDate)
	}
}

func TestCharacterMedals_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"corporation_id":98000001,"date":"2020-01-01T00:00:00Z","description":"desc","graphics":[{"graphic":"g1","layer":0,"part":1}],"issuer_id":998877,"medal_id":1,"reason":"reason","status":"private","title":"Title"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	medals, err := client.CharacterMedals(context.Background())
	if err != nil {
		t.Fatalf("CharacterMedals returned error: %v", err)
	}
	if len(medals) != 1 || medals[0].Title != "Title" || len(medals[0].Graphics) != 1 {
		t.Errorf("medals = %+v, want one entry with Title=\"Title\" and one graphic", medals)
	}
}

func TestCharacterNotifications_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"notification_id":1,"sender_id":998877,"sender_type":"character","timestamp":"2020-01-01T00:00:00Z","type":"CorpAllBillMsg"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	notifs, err := client.CharacterNotifications(context.Background())
	if err != nil {
		t.Fatalf("CharacterNotifications returned error: %v", err)
	}
	if len(notifs) != 1 || notifs[0].NotificationId != 1 {
		t.Errorf("notifs = %+v, want one entry with NotificationId=1", notifs)
	}
}

func TestCharacterNotificationContacts_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"message":"msg","notification_id":1,"send_date":"2020-01-01T00:00:00Z","sender_character_id":998877,"standing_level":5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	notifs, err := client.CharacterNotificationContacts(context.Background())
	if err != nil {
		t.Fatalf("CharacterNotificationContacts returned error: %v", err)
	}
	if len(notifs) != 1 || notifs[0].StandingLevel != 5 {
		t.Errorf("notifs = %+v, want one entry with StandingLevel=5", notifs)
	}
}

func TestCharacterRoles_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"roles":["Director"]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	roles, err := client.CharacterRoles(context.Background())
	if err != nil {
		t.Fatalf("CharacterRoles returned error: %v", err)
	}
	if len(roles.Roles) != 1 || roles.Roles[0] != "Director" {
		t.Errorf("roles.Roles = %v, want [\"Director\"]", roles.Roles)
	}
}

func TestCharacterStandings_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"from_id":500001,"from_type":"npc_corp","standing":5.5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	standings, err := client.CharacterStandings(context.Background())
	if err != nil {
		t.Fatalf("CharacterStandings returned error: %v", err)
	}
	if len(standings) != 1 || standings[0].Standing != 5.5 {
		t.Errorf("standings = %+v, want one entry with Standing=5.5", standings)
	}
}

func TestCharacterTitles_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"CEO","title_id":1}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	titles, err := client.CharacterTitles(context.Background())
	if err != nil {
		t.Fatalf("CharacterTitles returned error: %v", err)
	}
	if len(titles) != 1 || titles[0].Name == nil || *titles[0].Name != "CEO" {
		t.Errorf("titles = %+v, want one entry with Name=*\"CEO\"", titles)
	}
}

func TestCharacterCSPACost_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`1000000.0`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	cost, err := client.CharacterCSPACost(context.Background(), []int64{998878})
	if err != nil {
		t.Fatalf("CharacterCSPACost returned error: %v", err)
	}
	if cost != 1000000.0 {
		t.Errorf("cost = %v, want 1000000.0", cost)
	}
}

func TestCharacterCSPACost_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterCSPACost(context.Background(), []int64{998878}); err == nil {
		t.Fatal("CharacterCSPACost returned nil error, want an error for HTTP 403")
	}
}
