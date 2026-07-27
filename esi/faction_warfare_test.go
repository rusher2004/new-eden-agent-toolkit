package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterFwStats_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"current_rank":5,"enlisted_on":"2020-01-01T00:00:00Z","faction_id":500001,"highest_rank":8,"kills":{"last_week":10,"total":100,"yesterday":2},"victory_points":{"last_week":50,"total":5000,"yesterday":5}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	stats, err := client.CharacterFwStats(context.Background())
	if err != nil {
		t.Fatalf("CharacterFwStats returned error: %v", err)
	}
	if gotPath != "/characters/998877/fw/stats" {
		t.Errorf("path = %q, want \"/characters/998877/fw/stats\" (client's own characterID)", gotPath)
	}
	if stats.CurrentRank == nil || *stats.CurrentRank != 5 {
		t.Errorf("stats.CurrentRank = %v, want 5", stats.CurrentRank)
	}
	if stats.EnlistedOn == nil || stats.EnlistedOn.Year() != 2020 {
		t.Errorf("stats.EnlistedOn = %v, want 2020-01-01", stats.EnlistedOn)
	}
	if stats.FactionId == nil || *stats.FactionId != 500001 {
		t.Errorf("stats.FactionId = %v, want 500001", stats.FactionId)
	}
	if stats.HighestRank == nil || *stats.HighestRank != 8 {
		t.Errorf("stats.HighestRank = %v, want 8", stats.HighestRank)
	}
	if stats.Kills.LastWeek != 10 || stats.Kills.Total != 100 || stats.Kills.Yesterday != 2 {
		t.Errorf("stats.Kills = %+v, want {LastWeek:10 Total:100 Yesterday:2}", stats.Kills)
	}
	if stats.VictoryPoints.LastWeek != 50 || stats.VictoryPoints.Total != 5000 || stats.VictoryPoints.Yesterday != 5 {
		t.Errorf("stats.VictoryPoints = %+v, want {LastWeek:50 Total:5000 Yesterday:5}", stats.VictoryPoints)
	}
}

func TestCharacterFwStats_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterFwStats(context.Background()); err == nil {
		t.Fatal("CharacterFwStats returned nil error, want an error for HTTP 404")
	}
}

func TestCorporationFwStats_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"enlisted_on":"2021-06-01T00:00:00Z","faction_id":500002,"kills":{"last_week":20,"total":200,"yesterday":4},"pilots":42,"victory_points":{"last_week":60,"total":6000,"yesterday":6}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	stats, err := client.CorporationFwStats(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationFwStats returned error: %v", err)
	}
	if stats.EnlistedOn == nil || stats.EnlistedOn.Year() != 2021 {
		t.Errorf("stats.EnlistedOn = %v, want 2021-06-01", stats.EnlistedOn)
	}
	if stats.FactionId == nil || *stats.FactionId != 500002 {
		t.Errorf("stats.FactionId = %v, want 500002", stats.FactionId)
	}
	if stats.Kills.LastWeek != 20 || stats.Kills.Total != 200 || stats.Kills.Yesterday != 4 {
		t.Errorf("stats.Kills = %+v, want {LastWeek:20 Total:200 Yesterday:4}", stats.Kills)
	}
	if stats.Pilots == nil || *stats.Pilots != 42 {
		t.Errorf("stats.Pilots = %v, want 42", stats.Pilots)
	}
	if stats.VictoryPoints.LastWeek != 60 || stats.VictoryPoints.Total != 6000 || stats.VictoryPoints.Yesterday != 6 {
		t.Errorf("stats.VictoryPoints = %+v, want {LastWeek:60 Total:6000 Yesterday:6}", stats.VictoryPoints)
	}
}

func TestFwLeaderboards_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kills":{"active_total":[{"amount":1,"faction_id":500001}],"last_week":[{"amount":2,"faction_id":500002}],"yesterday":[{"amount":3,"faction_id":500003}]},"victory_points":{"active_total":[{"amount":4,"faction_id":500004}],"last_week":[{"amount":5,"faction_id":500005}],"yesterday":[{"amount":6,"faction_id":500006}]}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	board, err := client.FwLeaderboards(context.Background())
	if err != nil {
		t.Fatalf("FwLeaderboards returned error: %v", err)
	}
	if len(board.Kills.ActiveTotal) != 1 || board.Kills.ActiveTotal[0].Amount == nil || *board.Kills.ActiveTotal[0].Amount != 1 || board.Kills.ActiveTotal[0].FactionId == nil || *board.Kills.ActiveTotal[0].FactionId != 500001 {
		t.Errorf("board.Kills.ActiveTotal = %+v, want one entry Amount=1 FactionId=500001", board.Kills.ActiveTotal)
	}
	if len(board.Kills.LastWeek) != 1 || board.Kills.LastWeek[0].Amount == nil || *board.Kills.LastWeek[0].Amount != 2 || board.Kills.LastWeek[0].FactionId == nil || *board.Kills.LastWeek[0].FactionId != 500002 {
		t.Errorf("board.Kills.LastWeek = %+v, want one entry Amount=2 FactionId=500002", board.Kills.LastWeek)
	}
	if len(board.Kills.Yesterday) != 1 || board.Kills.Yesterday[0].Amount == nil || *board.Kills.Yesterday[0].Amount != 3 || board.Kills.Yesterday[0].FactionId == nil || *board.Kills.Yesterday[0].FactionId != 500003 {
		t.Errorf("board.Kills.Yesterday = %+v, want one entry Amount=3 FactionId=500003", board.Kills.Yesterday)
	}
	if len(board.VictoryPoints.ActiveTotal) != 1 || board.VictoryPoints.ActiveTotal[0].Amount == nil || *board.VictoryPoints.ActiveTotal[0].Amount != 4 || board.VictoryPoints.ActiveTotal[0].FactionId == nil || *board.VictoryPoints.ActiveTotal[0].FactionId != 500004 {
		t.Errorf("board.VictoryPoints.ActiveTotal = %+v, want one entry Amount=4 FactionId=500004", board.VictoryPoints.ActiveTotal)
	}
	if len(board.VictoryPoints.LastWeek) != 1 || board.VictoryPoints.LastWeek[0].Amount == nil || *board.VictoryPoints.LastWeek[0].Amount != 5 || board.VictoryPoints.LastWeek[0].FactionId == nil || *board.VictoryPoints.LastWeek[0].FactionId != 500005 {
		t.Errorf("board.VictoryPoints.LastWeek = %+v, want one entry Amount=5 FactionId=500005", board.VictoryPoints.LastWeek)
	}
	if len(board.VictoryPoints.Yesterday) != 1 || board.VictoryPoints.Yesterday[0].Amount == nil || *board.VictoryPoints.Yesterday[0].Amount != 6 || board.VictoryPoints.Yesterday[0].FactionId == nil || *board.VictoryPoints.Yesterday[0].FactionId != 500006 {
		t.Errorf("board.VictoryPoints.Yesterday = %+v, want one entry Amount=6 FactionId=500006", board.VictoryPoints.Yesterday)
	}
}

func TestFwCharacterLeaderboards_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kills":{"active_total":[{"amount":1,"character_id":998801}],"last_week":[{"amount":2,"character_id":998802}],"yesterday":[{"amount":3,"character_id":998803}]},"victory_points":{"active_total":[{"amount":4,"character_id":998804}],"last_week":[{"amount":5,"character_id":998805}],"yesterday":[{"amount":6,"character_id":998806}]}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	board, err := client.FwCharacterLeaderboards(context.Background())
	if err != nil {
		t.Fatalf("FwCharacterLeaderboards returned error: %v", err)
	}
	if len(board.Kills.ActiveTotal) != 1 || board.Kills.ActiveTotal[0].Amount == nil || *board.Kills.ActiveTotal[0].Amount != 1 || board.Kills.ActiveTotal[0].CharacterId == nil || *board.Kills.ActiveTotal[0].CharacterId != 998801 {
		t.Errorf("board.Kills.ActiveTotal = %+v, want one entry Amount=1 CharacterId=998801", board.Kills.ActiveTotal)
	}
	if len(board.Kills.LastWeek) != 1 || board.Kills.LastWeek[0].Amount == nil || *board.Kills.LastWeek[0].Amount != 2 || board.Kills.LastWeek[0].CharacterId == nil || *board.Kills.LastWeek[0].CharacterId != 998802 {
		t.Errorf("board.Kills.LastWeek = %+v, want one entry Amount=2 CharacterId=998802", board.Kills.LastWeek)
	}
	if len(board.Kills.Yesterday) != 1 || board.Kills.Yesterday[0].Amount == nil || *board.Kills.Yesterday[0].Amount != 3 || board.Kills.Yesterday[0].CharacterId == nil || *board.Kills.Yesterday[0].CharacterId != 998803 {
		t.Errorf("board.Kills.Yesterday = %+v, want one entry Amount=3 CharacterId=998803", board.Kills.Yesterday)
	}
	if len(board.VictoryPoints.ActiveTotal) != 1 || board.VictoryPoints.ActiveTotal[0].Amount == nil || *board.VictoryPoints.ActiveTotal[0].Amount != 4 || board.VictoryPoints.ActiveTotal[0].CharacterId == nil || *board.VictoryPoints.ActiveTotal[0].CharacterId != 998804 {
		t.Errorf("board.VictoryPoints.ActiveTotal = %+v, want one entry Amount=4 CharacterId=998804", board.VictoryPoints.ActiveTotal)
	}
	if len(board.VictoryPoints.LastWeek) != 1 || board.VictoryPoints.LastWeek[0].Amount == nil || *board.VictoryPoints.LastWeek[0].Amount != 5 || board.VictoryPoints.LastWeek[0].CharacterId == nil || *board.VictoryPoints.LastWeek[0].CharacterId != 998805 {
		t.Errorf("board.VictoryPoints.LastWeek = %+v, want one entry Amount=5 CharacterId=998805", board.VictoryPoints.LastWeek)
	}
	if len(board.VictoryPoints.Yesterday) != 1 || board.VictoryPoints.Yesterday[0].Amount == nil || *board.VictoryPoints.Yesterday[0].Amount != 6 || board.VictoryPoints.Yesterday[0].CharacterId == nil || *board.VictoryPoints.Yesterday[0].CharacterId != 998806 {
		t.Errorf("board.VictoryPoints.Yesterday = %+v, want one entry Amount=6 CharacterId=998806", board.VictoryPoints.Yesterday)
	}
}

func TestFwCorporationLeaderboards_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kills":{"active_total":[{"amount":1,"corporation_id":98000001}],"last_week":[{"amount":2,"corporation_id":98000002}],"yesterday":[{"amount":3,"corporation_id":98000003}]},"victory_points":{"active_total":[{"amount":4,"corporation_id":98000004}],"last_week":[{"amount":5,"corporation_id":98000005}],"yesterday":[{"amount":6,"corporation_id":98000006}]}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	board, err := client.FwCorporationLeaderboards(context.Background())
	if err != nil {
		t.Fatalf("FwCorporationLeaderboards returned error: %v", err)
	}
	if len(board.Kills.ActiveTotal) != 1 || board.Kills.ActiveTotal[0].Amount == nil || *board.Kills.ActiveTotal[0].Amount != 1 || board.Kills.ActiveTotal[0].CorporationId == nil || *board.Kills.ActiveTotal[0].CorporationId != 98000001 {
		t.Errorf("board.Kills.ActiveTotal = %+v, want one entry Amount=1 CorporationId=98000001", board.Kills.ActiveTotal)
	}
	if len(board.Kills.LastWeek) != 1 || board.Kills.LastWeek[0].Amount == nil || *board.Kills.LastWeek[0].Amount != 2 || board.Kills.LastWeek[0].CorporationId == nil || *board.Kills.LastWeek[0].CorporationId != 98000002 {
		t.Errorf("board.Kills.LastWeek = %+v, want one entry Amount=2 CorporationId=98000002", board.Kills.LastWeek)
	}
	if len(board.Kills.Yesterday) != 1 || board.Kills.Yesterday[0].Amount == nil || *board.Kills.Yesterday[0].Amount != 3 || board.Kills.Yesterday[0].CorporationId == nil || *board.Kills.Yesterday[0].CorporationId != 98000003 {
		t.Errorf("board.Kills.Yesterday = %+v, want one entry Amount=3 CorporationId=98000003", board.Kills.Yesterday)
	}
	if len(board.VictoryPoints.ActiveTotal) != 1 || board.VictoryPoints.ActiveTotal[0].Amount == nil || *board.VictoryPoints.ActiveTotal[0].Amount != 4 || board.VictoryPoints.ActiveTotal[0].CorporationId == nil || *board.VictoryPoints.ActiveTotal[0].CorporationId != 98000004 {
		t.Errorf("board.VictoryPoints.ActiveTotal = %+v, want one entry Amount=4 CorporationId=98000004", board.VictoryPoints.ActiveTotal)
	}
	if len(board.VictoryPoints.LastWeek) != 1 || board.VictoryPoints.LastWeek[0].Amount == nil || *board.VictoryPoints.LastWeek[0].Amount != 5 || board.VictoryPoints.LastWeek[0].CorporationId == nil || *board.VictoryPoints.LastWeek[0].CorporationId != 98000005 {
		t.Errorf("board.VictoryPoints.LastWeek = %+v, want one entry Amount=5 CorporationId=98000005", board.VictoryPoints.LastWeek)
	}
	if len(board.VictoryPoints.Yesterday) != 1 || board.VictoryPoints.Yesterday[0].Amount == nil || *board.VictoryPoints.Yesterday[0].Amount != 6 || board.VictoryPoints.Yesterday[0].CorporationId == nil || *board.VictoryPoints.Yesterday[0].CorporationId != 98000006 {
		t.Errorf("board.VictoryPoints.Yesterday = %+v, want one entry Amount=6 CorporationId=98000006", board.VictoryPoints.Yesterday)
	}
}

func TestFwStats_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"faction_id":500001,"kills":{"last_week":10,"total":100,"yesterday":2},"pilots":1000,"systems_controlled":15,"victory_points":{"last_week":50,"total":5000,"yesterday":5}}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	stats, err := client.FwStats(context.Background())
	if err != nil {
		t.Fatalf("FwStats returned error: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("stats = %+v, want one entry", stats)
	}
	s := stats[0]
	if s.FactionId != 500001 {
		t.Errorf("s.FactionId = %d, want 500001", s.FactionId)
	}
	if s.Kills.LastWeek != 10 || s.Kills.Total != 100 || s.Kills.Yesterday != 2 {
		t.Errorf("s.Kills = %+v, want {LastWeek:10 Total:100 Yesterday:2}", s.Kills)
	}
	if s.Pilots != 1000 {
		t.Errorf("s.Pilots = %d, want 1000", s.Pilots)
	}
	if s.SystemsControlled != 15 {
		t.Errorf("s.SystemsControlled = %d, want 15", s.SystemsControlled)
	}
	if s.VictoryPoints.LastWeek != 50 || s.VictoryPoints.Total != 5000 || s.VictoryPoints.Yesterday != 5 {
		t.Errorf("s.VictoryPoints = %+v, want {LastWeek:50 Total:5000 Yesterday:5}", s.VictoryPoints)
	}
}

func TestFwSystems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"contested":"contested","occupier_faction_id":500001,"owner_faction_id":500002,"solar_system_id":30002537,"victory_points":100,"victory_points_threshold":1000}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	systems, err := client.FwSystems(context.Background())
	if err != nil {
		t.Fatalf("FwSystems returned error: %v", err)
	}
	if len(systems) != 1 {
		t.Fatalf("systems = %+v, want one entry", systems)
	}
	sys := systems[0]
	if sys.Contested != "contested" {
		t.Errorf("sys.Contested = %q, want \"contested\"", sys.Contested)
	}
	if sys.OccupierFactionId != 500001 {
		t.Errorf("sys.OccupierFactionId = %d, want 500001", sys.OccupierFactionId)
	}
	if sys.OwnerFactionId != 500002 {
		t.Errorf("sys.OwnerFactionId = %d, want 500002", sys.OwnerFactionId)
	}
	if sys.SolarSystemId != 30002537 {
		t.Errorf("sys.SolarSystemId = %d, want 30002537", sys.SolarSystemId)
	}
	if sys.VictoryPoints != 100 {
		t.Errorf("sys.VictoryPoints = %d, want 100", sys.VictoryPoints)
	}
	if sys.VictoryPointsThreshold != 1000 {
		t.Errorf("sys.VictoryPointsThreshold = %d, want 1000", sys.VictoryPointsThreshold)
	}
}

func TestFwWars_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"against_id":500002,"faction_id":500001}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	wars, err := client.FwWars(context.Background())
	if err != nil {
		t.Fatalf("FwWars returned error: %v", err)
	}
	if len(wars) != 1 {
		t.Fatalf("wars = %+v, want one entry", wars)
	}
	if wars[0].AgainstId != 500002 {
		t.Errorf("wars[0].AgainstId = %d, want 500002", wars[0].AgainstId)
	}
	if wars[0].FactionId != 500001 {
		t.Errorf("wars[0].FactionId = %d, want 500001", wars[0].FactionId)
	}
}
