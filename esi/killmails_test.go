package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterKillmailsRecent_ParsesResponseAndPageOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"killmail_hash":"abc123","killmail_id":1}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	killmails, _, err := client.CharacterKillmailsRecent(context.Background(), 0)
	if err != nil {
		t.Fatalf("CharacterKillmailsRecent returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
	if len(killmails) != 1 || killmails[0].KillmailHash != "abc123" || killmails[0].KillmailId != 1 {
		t.Errorf("killmails = %+v, want one entry with KillmailHash=\"abc123\" and KillmailId=1", killmails)
	}
}

func TestCharacterKillmailsRecent_PageSentWhenSetAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{"killmail_hash":"abc123","killmail_id":1}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	_, totalPages, err := client.CharacterKillmailsRecent(context.Background(), 2)
	if err != nil {
		t.Fatalf("CharacterKillmailsRecent(page=2) returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want \"page=2\"", gotQuery)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
}

func TestCharacterKillmailsRecent_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterKillmailsRecent(context.Background(), 0); err == nil {
		t.Fatal("CharacterKillmailsRecent returned nil error, want an error for HTTP 404")
	}
}

func TestCorporationKillmailsRecent_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{"killmail_hash":"def456","killmail_id":2}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	killmails, totalPages, err := client.CorporationKillmailsRecent(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationKillmailsRecent returned error: %v", err)
	}
	if totalPages != 1 || len(killmails) != 1 || killmails[0].KillmailHash != "def456" || killmails[0].KillmailId != 2 {
		t.Errorf("killmails = %+v, totalPages = %d, want one entry with KillmailHash=\"def456\", KillmailId=2, totalPages=1", killmails, totalPages)
	}
}

func TestKillmailDetail_ParsesResponseAndUsesCorrectArgumentOrder(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"attackers": [{
				"alliance_id": 1,
				"character_id": 2,
				"corporation_id": 3,
				"damage_done": 4000,
				"faction_id": 5,
				"final_blow": true,
				"security_status": -1.5,
				"ship_type_id": 6,
				"weapon_type_id": 7
			}],
			"killmail_id": 42,
			"killmail_time": "2026-07-01T00:00:00Z",
			"moon_id": 8,
			"solar_system_id": 30000142,
			"victim": {
				"alliance_id": 9,
				"character_id": 10,
				"corporation_id": 11,
				"damage_taken": 5000,
				"faction_id": 12,
				"items": [{
					"flag": 13,
					"item_type_id": 14,
					"items": [{
						"flag": 15,
						"item_type_id": 16,
						"quantity_destroyed": 1,
						"quantity_dropped": 2,
						"singleton": 0
					}],
					"quantity_destroyed": 3,
					"quantity_dropped": 4,
					"singleton": 1
				}],
				"position": {"x": 1.1, "y": 2.2, "z": 3.3},
				"ship_type_id": 17
			},
			"war_id": 18
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	km, err := client.KillmailDetail(context.Background(), 42, "somehash")
	if err != nil {
		t.Fatalf("KillmailDetail returned error: %v", err)
	}

	wantPath := "/killmails/42/somehash"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q (killmailID=42, killmailHash=\"somehash\", not swapped)", gotPath, wantPath)
	}

	if km.KillmailId != 42 {
		t.Errorf("KillmailId = %d, want 42", km.KillmailId)
	}
	if km.SolarSystemId != 30000142 {
		t.Errorf("SolarSystemId = %d, want 30000142", km.SolarSystemId)
	}
	if km.MoonId == nil || *km.MoonId != 8 {
		t.Errorf("MoonId = %v, want 8", km.MoonId)
	}
	if km.WarId == nil || *km.WarId != 18 {
		t.Errorf("WarId = %v, want 18", km.WarId)
	}
	if km.KillmailTime.IsZero() {
		t.Error("KillmailTime is zero, want parsed timestamp")
	}

	if len(km.Attackers) != 1 {
		t.Fatalf("len(Attackers) = %d, want 1", len(km.Attackers))
	}
	a := km.Attackers[0]
	if a.AllianceId == nil || *a.AllianceId != 1 {
		t.Errorf("Attackers[0].AllianceId = %v, want 1", a.AllianceId)
	}
	if a.CharacterId == nil || *a.CharacterId != 2 {
		t.Errorf("Attackers[0].CharacterId = %v, want 2", a.CharacterId)
	}
	if a.CorporationId == nil || *a.CorporationId != 3 {
		t.Errorf("Attackers[0].CorporationId = %v, want 3", a.CorporationId)
	}
	if a.DamageDone != 4000 {
		t.Errorf("Attackers[0].DamageDone = %d, want 4000", a.DamageDone)
	}
	if a.FactionId == nil || *a.FactionId != 5 {
		t.Errorf("Attackers[0].FactionId = %v, want 5", a.FactionId)
	}
	if !a.FinalBlow {
		t.Error("Attackers[0].FinalBlow = false, want true")
	}
	if a.SecurityStatus != -1.5 {
		t.Errorf("Attackers[0].SecurityStatus = %v, want -1.5", a.SecurityStatus)
	}
	if a.ShipTypeId == nil || *a.ShipTypeId != 6 {
		t.Errorf("Attackers[0].ShipTypeId = %v, want 6", a.ShipTypeId)
	}
	if a.WeaponTypeId == nil || *a.WeaponTypeId != 7 {
		t.Errorf("Attackers[0].WeaponTypeId = %v, want 7", a.WeaponTypeId)
	}

	v := km.Victim
	if v.AllianceId == nil || *v.AllianceId != 9 {
		t.Errorf("Victim.AllianceId = %v, want 9", v.AllianceId)
	}
	if v.CharacterId == nil || *v.CharacterId != 10 {
		t.Errorf("Victim.CharacterId = %v, want 10", v.CharacterId)
	}
	if v.CorporationId == nil || *v.CorporationId != 11 {
		t.Errorf("Victim.CorporationId = %v, want 11", v.CorporationId)
	}
	if v.DamageTaken != 5000 {
		t.Errorf("Victim.DamageTaken = %d, want 5000", v.DamageTaken)
	}
	if v.FactionId == nil || *v.FactionId != 12 {
		t.Errorf("Victim.FactionId = %v, want 12", v.FactionId)
	}
	if v.ShipTypeId != 17 {
		t.Errorf("Victim.ShipTypeId = %d, want 17", v.ShipTypeId)
	}
	if v.Position == nil || v.Position.X != 1.1 || v.Position.Y != 2.2 || v.Position.Z != 3.3 {
		t.Errorf("Victim.Position = %+v, want {X:1.1 Y:2.2 Z:3.3}", v.Position)
	}

	if len(v.Items) != 1 {
		t.Fatalf("len(Victim.Items) = %d, want 1", len(v.Items))
	}
	item := v.Items[0]
	if item.Flag != 13 {
		t.Errorf("Victim.Items[0].Flag = %d, want 13", item.Flag)
	}
	if item.ItemTypeId != 14 {
		t.Errorf("Victim.Items[0].ItemTypeId = %d, want 14", item.ItemTypeId)
	}
	if item.QuantityDestroyed == nil || *item.QuantityDestroyed != 3 {
		t.Errorf("Victim.Items[0].QuantityDestroyed = %v, want 3", item.QuantityDestroyed)
	}
	if item.QuantityDropped == nil || *item.QuantityDropped != 4 {
		t.Errorf("Victim.Items[0].QuantityDropped = %v, want 4", item.QuantityDropped)
	}
	if item.Singleton != 1 {
		t.Errorf("Victim.Items[0].Singleton = %d, want 1", item.Singleton)
	}

	if len(item.Items) != 1 {
		t.Fatalf("len(Victim.Items[0].Items) = %d, want 1 (container-in-item nesting)", len(item.Items))
	}
	nested := item.Items[0]
	if nested.Flag != 15 {
		t.Errorf("Victim.Items[0].Items[0].Flag = %d, want 15", nested.Flag)
	}
	if nested.ItemTypeId != 16 {
		t.Errorf("Victim.Items[0].Items[0].ItemTypeId = %d, want 16", nested.ItemTypeId)
	}
	if nested.QuantityDestroyed == nil || *nested.QuantityDestroyed != 1 {
		t.Errorf("Victim.Items[0].Items[0].QuantityDestroyed = %v, want 1", nested.QuantityDestroyed)
	}
	if nested.QuantityDropped == nil || *nested.QuantityDropped != 2 {
		t.Errorf("Victim.Items[0].Items[0].QuantityDropped = %v, want 2", nested.QuantityDropped)
	}
	if nested.Singleton != 0 {
		t.Errorf("Victim.Items[0].Items[0].Singleton = %d, want 0", nested.Singleton)
	}
}

func TestKillmailDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.KillmailDetail(context.Background(), 42, "somehash"); err == nil {
		t.Fatal("KillmailDetail returned nil error, want an error for HTTP 404")
	}
}
