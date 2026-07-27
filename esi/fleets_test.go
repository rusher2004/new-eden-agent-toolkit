package esi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterFleet_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"fleet_boss_id":998877,"fleet_id":555,"role":"squad_member","squad_id":2,"wing_id":1}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	fleet, err := client.CharacterFleet(context.Background())
	if err != nil {
		t.Fatalf("CharacterFleet returned error: %v", err)
	}
	if fleet.FleetBossId != 998877 || fleet.FleetId != 555 || fleet.Role != "squad_member" || fleet.SquadId != 2 || fleet.WingId != 1 {
		t.Errorf("fleet = %+v, want FleetBossId=998877 FleetId=555 Role=\"squad_member\" SquadId=2 WingId=1", fleet)
	}
	if gotPath != "/characters/998877/fleet/" && gotPath != "/characters/998877/fleet" {
		t.Errorf("request path = %q, want it to reference character 998877", gotPath)
	}
}

func TestCharacterFleet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterFleet(context.Background()); err == nil {
		t.Fatal("CharacterFleet returned nil error, want an error for HTTP 404")
	}
}

func TestFleetInfo_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"is_free_move":true,"is_registered":false,"is_voice_enabled":true,"motd":"Welcome"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	info, err := client.FleetInfo(context.Background(), 555)
	if err != nil {
		t.Fatalf("FleetInfo returned error: %v", err)
	}
	if !info.IsFreeMove || info.IsRegistered || !info.IsVoiceEnabled || info.Motd != "Welcome" {
		t.Errorf("info = %+v, want IsFreeMove=true IsRegistered=false IsVoiceEnabled=true Motd=\"Welcome\"", info)
	}
}

func TestUpdateFleet_SendsCorrectRequestBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	isFreeMove := true
	motd := "New MOTD"
	if err := client.UpdateFleet(context.Background(), 555, &isFreeMove, &motd); err != nil {
		t.Fatalf("UpdateFleet returned error: %v", err)
	}

	var got struct {
		IsFreeMove *bool   `json:"is_free_move"`
		Motd       *string `json:"motd"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if got.IsFreeMove == nil || *got.IsFreeMove != true || got.Motd == nil || *got.Motd != "New MOTD" {
		t.Errorf("request body = %+v, want IsFreeMove=*true Motd=*\"New MOTD\"", got)
	}
}

func TestFleetMembers_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"character_id":123456,"join_time":"2026-08-01T00:00:00Z","role":"squad_member","role_name":"Squad Member","ship_type_id":602,"solar_system_id":30000142,"squad_id":2,"station_id":60003760,"takes_fleet_warp":true,"wing_id":1}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	members, err := client.FleetMembers(context.Background(), 555)
	if err != nil {
		t.Fatalf("FleetMembers returned error: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("members = %+v, want one entry", members)
	}
	m := members[0]
	if m.CharacterId != 123456 || m.Role != "squad_member" || m.RoleName != "Squad Member" || m.ShipTypeId != 602 {
		t.Errorf("members[0] = %+v, want CharacterId=123456 Role=\"squad_member\" RoleName=\"Squad Member\" ShipTypeId=602", m)
	}
	if m.SolarSystemId != 30000142 || m.SquadId != 2 || m.WingId != 1 || !m.TakesFleetWarp {
		t.Errorf("members[0] = %+v, want SolarSystemId=30000142 SquadId=2 WingId=1 TakesFleetWarp=true", m)
	}
	if m.StationId == nil || *m.StationId != 60003760 {
		t.Errorf("members[0].StationId = %v, want *60003760", m.StationId)
	}
}

func TestInviteFleetMember_SendsCorrectRequestBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	wingID := int64(1)
	squadID := int64(2)
	if err := client.InviteFleetMember(context.Background(), 555, 123456, "squad_member", &wingID, &squadID); err != nil {
		t.Fatalf("InviteFleetMember returned error: %v", err)
	}

	var got struct {
		CharacterId int64  `json:"character_id"`
		Role        string `json:"role"`
		SquadId     *int64 `json:"squad_id"`
		WingId      *int64 `json:"wing_id"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if got.CharacterId != 123456 || got.Role != "squad_member" {
		t.Errorf("request body = %+v, want CharacterId=123456 Role=\"squad_member\"", got)
	}
	if got.SquadId == nil || *got.SquadId != 2 || got.WingId == nil || *got.WingId != 1 {
		t.Errorf("request body = %+v, want SquadId=*2 WingId=*1", got)
	}
}

func TestMoveFleetMember_SendsBodyAndHandlesNoContent(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	wingID := int64(3)
	squadID := int64(4)
	if err := client.MoveFleetMember(context.Background(), 555, 123456, "squad_commander", &wingID, &squadID); err != nil {
		t.Fatalf("MoveFleetMember returned error: %v", err)
	}

	var got struct {
		Role    string `json:"role"`
		SquadId *int64 `json:"squad_id"`
		WingId  *int64 `json:"wing_id"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if got.Role != "squad_commander" || got.SquadId == nil || *got.SquadId != 4 || got.WingId == nil || *got.WingId != 3 {
		t.Errorf("request body = %+v, want Role=\"squad_commander\" SquadId=*4 WingId=*3", got)
	}
}

func TestKickFleetMember_HandlesNoContentAndHTTPError(t *testing.T) {
	t.Run("no content", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
			BaseURL:   srv.URL,
			UserAgent: "test-agent",
		})

		if err := client.KickFleetMember(context.Background(), 555, 123456); err != nil {
			t.Fatalf("KickFleetMember returned error: %v", err)
		}
	})

	t.Run("http error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
			BaseURL:   srv.URL,
			UserAgent: "test-agent",
		})

		if err := client.KickFleetMember(context.Background(), 555, 123456); err == nil {
			t.Fatal("KickFleetMember returned nil error, want an error for HTTP 404")
		}
	})
}

func TestFleetWings_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":1,"name":"Wing 1","squads":[{"id":2,"name":"Squad 1"}]}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	wings, err := client.FleetWings(context.Background(), 555)
	if err != nil {
		t.Fatalf("FleetWings returned error: %v", err)
	}
	if len(wings) != 1 || wings[0].Id != 1 || wings[0].Name != "Wing 1" {
		t.Fatalf("wings = %+v, want one entry {Id:1 Name:\"Wing 1\"}", wings)
	}
	squads := wings[0].Squads
	if len(squads) != 1 || squads[0].Id != 2 || squads[0].Name != "Squad 1" {
		t.Errorf("wings[0].Squads = %+v, want one entry {Id:2 Name:\"Squad 1\"}", squads)
	}
}

func TestCreateFleetWing_ReturnsWingIDAndSendsNoBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"wing_id":42}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	wingID, err := client.CreateFleetWing(context.Background(), 555)
	if err != nil {
		t.Fatalf("CreateFleetWing returned error: %v", err)
	}
	if len(gotBody) != 0 {
		t.Errorf("request body = %q, want empty", gotBody)
	}
	if wingID != 42 {
		t.Errorf("wingID = %d, want 42", wingID)
	}
}

func TestRenameFleetWing_UsesSquadRenameBodyShape(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.RenameFleetWing(context.Background(), 555, 1, "Alpha Wing"); err != nil {
		t.Fatalf("RenameFleetWing returned error: %v", err)
	}

	// The generated setter for this endpoint is (mis)named
	// PutFleetsFleetIdSquadsSquadIdRequest and reuses the squad-rename model, but the
	// wire shape it produces is still just {"name": "..."}. This test proves that the
	// real generated PutFleetsFleetIdWingsWingId call, invoked exactly as RenameFleetWing
	// invokes it, round-trips the name through that misnamed method correctly -- not a
	// mock standing in for it.
	if gotBody == nil {
		t.Fatal("request body was not captured")
	}
	var got struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if got.Name != "Alpha Wing" {
		t.Errorf("request body Name = %q, want \"Alpha Wing\"", got.Name)
	}

	var raw map[string]any
	if err := json.Unmarshal(gotBody, &raw); err != nil {
		t.Fatalf("failed to unmarshal request body %q as raw map: %v", gotBody, err)
	}
	if len(raw) != 1 {
		t.Errorf("request body = %q, want exactly one field (\"name\"), got %d fields", gotBody, len(raw))
	}
}

func TestDeleteFleetWing_HandlesNoContentAndHTTPError(t *testing.T) {
	t.Run("no content", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
			BaseURL:   srv.URL,
			UserAgent: "test-agent",
		})

		if err := client.DeleteFleetWing(context.Background(), 555, 1); err != nil {
			t.Fatalf("DeleteFleetWing returned error: %v", err)
		}
	})

	t.Run("http error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
			BaseURL:   srv.URL,
			UserAgent: "test-agent",
		})

		if err := client.DeleteFleetWing(context.Background(), 555, 1); err == nil {
			t.Fatal("DeleteFleetWing returned nil error, want an error for HTTP 404")
		}
	})
}

func TestCreateFleetSquad_ReturnsSquadIDAndSendsNoBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"squad_id":99}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	squadID, err := client.CreateFleetSquad(context.Background(), 555, 1)
	if err != nil {
		t.Fatalf("CreateFleetSquad returned error: %v", err)
	}
	if len(gotBody) != 0 {
		t.Errorf("request body = %q, want empty", gotBody)
	}
	if squadID != 99 {
		t.Errorf("squadID = %d, want 99", squadID)
	}
}

func TestRenameFleetSquad_SendsCorrectRequestBody(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.RenameFleetSquad(context.Background(), 555, 2, "Bravo Squad"); err != nil {
		t.Fatalf("RenameFleetSquad returned error: %v", err)
	}

	// The body may have a trailing newline from the JSON encoder
	if s := string(gotBody); s != `{"name":"Bravo Squad"}` && s != `{"name":"Bravo Squad"}`+"\n" {
		t.Errorf("request body = %q, want %q", gotBody, `{"name":"Bravo Squad"}`)
	}
}

func TestDeleteFleetSquad_HandlesNoContentAndHTTPError(t *testing.T) {
	t.Run("no content", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
			BaseURL:   srv.URL,
			UserAgent: "test-agent",
		})

		if err := client.DeleteFleetSquad(context.Background(), 555, 2); err != nil {
			t.Fatalf("DeleteFleetSquad returned error: %v", err)
		}
	})

	t.Run("http error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
			BaseURL:   srv.URL,
			UserAgent: "test-agent",
		})

		if err := client.DeleteFleetSquad(context.Background(), 555, 2); err == nil {
			t.Fatal("DeleteFleetSquad returned nil error, want an error for HTTP 404")
		}
	})
}
