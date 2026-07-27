package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCorporationDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"alliance_id": 99000001,
			"ceo_id": 2112625428,
			"creator_id": 1000124,
			"date_founded": "2003-12-28T22:41:00Z",
			"description": "Test Corporation",
			"enlisted_faction_id": 500001,
			"home_station_id": 60003760,
			"member_count": 100,
			"name": "Test Corp",
			"shares": 1000000,
			"ticker": "TEST",
			"url": "https://example.com",
			"friendly_fire": "true",
			"war_eligible": true,
			"state": "normal",
			"type": "regular",
			"palette": {
				"main_color": "#FF0000",
				"secondary_color": "#00FF00",
				"tertiary_color": "#0000FF"
			},
			"tax_rates": {
				"isk": 0.05,
				"loyalty_point": 0.1
			}
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CorporationDetail(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationDetail returned error: %v", err)
	}
	if detail.Name != "Test Corp" || detail.Ticker != "TEST" {
		t.Errorf("detail = %+v, want Name=\"Test Corp\" Ticker=\"TEST\"", detail)
	}
	if detail.Palette == nil || detail.Palette.MainColor != "#FF0000" {
		t.Errorf("Palette.MainColor = %v, want \"#FF0000\"", detail.Palette.MainColor)
	}
	if detail.Palette.SecondaryColor == nil || *detail.Palette.SecondaryColor != "#00FF00" {
		t.Errorf("Palette.SecondaryColor = %v, want \"#00FF00\"", detail.Palette.SecondaryColor)
	}
	if detail.Palette.TertiaryColor == nil || *detail.Palette.TertiaryColor != "#0000FF" {
		t.Errorf("Palette.TertiaryColor = %v, want \"#0000FF\"", detail.Palette.TertiaryColor)
	}
	if detail.TaxRates.Isk != 0.05 {
		t.Errorf("TaxRates.Isk = %v, want 0.05", detail.TaxRates.Isk)
	}
	if detail.TaxRates.LoyaltyPoint != 0.1 {
		t.Errorf("TaxRates.LoyaltyPoint = %v, want 0.1", detail.TaxRates.LoyaltyPoint)
	}
}

func TestCorporationDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CorporationDetail(context.Background(), 98000001); err == nil {
		t.Fatal("CorporationDetail returned nil error, want an error for HTTP 404")
	}
}

func TestCorporationAllianceHistory_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"alliance_id": 99000001, "record_id": 1, "start_date": "2003-12-28T22:41:00Z", "is_deleted": false},
			{"alliance_id": 99000002, "record_id": 2, "start_date": "2010-01-01T00:00:00Z", "is_deleted": true}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	hist, err := client.CorporationAllianceHistory(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationAllianceHistory returned error: %v", err)
	}
	if len(hist) != 2 {
		t.Errorf("hist length = %d, want 2", len(hist))
	}
	if hist[0].RecordId != 1 {
		t.Errorf("hist[0].RecordId = %d, want 1", hist[0].RecordId)
	}
	if *hist[0].AllianceId != 99000001 {
		t.Errorf("hist[0].AllianceId = %v, want 99000001", hist[0].AllianceId)
	}
}

func TestCorporationIcons_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"px64x64": "https://example.com/64.png",
			"px128x128": "https://example.com/128.png",
			"px256x256": "https://example.com/256.png"
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	icons, err := client.CorporationIcons(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationIcons returned error: %v", err)
	}
	if icons.Px64x64 == nil || *icons.Px64x64 != "https://example.com/64.png" {
		t.Errorf("Px64x64 = %v, want https://example.com/64.png", icons.Px64x64)
	}
	if icons.Px128x128 == nil || *icons.Px128x128 != "https://example.com/128.png" {
		t.Errorf("Px128x128 = %v, want https://example.com/128.png", icons.Px128x128)
	}
	if icons.Px256x256 == nil || *icons.Px256x256 != "https://example.com/256.png" {
		t.Errorf("Px256x256 = %v, want https://example.com/256.png", icons.Px256x256)
	}
}

func TestNPCCorporations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[1000001, 1000002, 1000003]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ids, err := client.NPCCorporations(context.Background())
	if err != nil {
		t.Fatalf("NPCCorporations returned error: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1000001 || ids[1] != 1000002 || ids[2] != 1000003 {
		t.Errorf("ids = %v, want [1000001 1000002 1000003]", ids)
	}
}

func TestCorporationMembers_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[2112625428, 2112625429, 2112625430]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	members, err := client.CorporationMembers(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationMembers returned error: %v", err)
	}
	if len(members) != 3 || members[0] != 2112625428 {
		t.Errorf("members = %v, want [2112625428 2112625429 2112625430]", members)
	}
}

func TestCorporationMembersLimit_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`500`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	limit, err := client.CorporationMembersLimit(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationMembersLimit returned error: %v", err)
	}
	if limit != 500 {
		t.Errorf("limit = %d, want 500", limit)
	}
}

func TestCorporationMemberTitles_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"character_id": 2112625428, "titles": [1, 2, 3]},
			{"character_id": 2112625429, "titles": [2, 4]}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	titles, err := client.CorporationMemberTitles(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationMemberTitles returned error: %v", err)
	}
	if len(titles) != 2 {
		t.Errorf("titles length = %d, want 2", len(titles))
	}
	if titles[0].CharacterId != 2112625428 {
		t.Errorf("titles[0].CharacterId = %d, want 2112625428", titles[0].CharacterId)
	}
	if len(titles[0].Titles) != 3 {
		t.Errorf("titles[0].Titles length = %d, want 3", len(titles[0].Titles))
	}
}

func TestCorporationMemberTracking_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"character_id": 2112625428,
				"base_id": 60003760,
				"location_id": 30002187,
				"ship_type_id": 587,
				"logon_date": "2026-07-01T10:00:00Z",
				"logoff_date": "2026-07-01T12:00:00Z",
				"start_date": "2026-01-01T00:00:00Z"
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	tracking, err := client.CorporationMemberTracking(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationMemberTracking returned error: %v", err)
	}
	if len(tracking) != 1 {
		t.Errorf("tracking length = %d, want 1", len(tracking))
	}
	if tracking[0].BaseId == nil || *tracking[0].BaseId != 60003760 {
		t.Errorf("BaseId = %v, want 60003760", tracking[0].BaseId)
	}
	if tracking[0].LocationId == nil || *tracking[0].LocationId != 30002187 {
		t.Errorf("LocationId = %v, want 30002187", tracking[0].LocationId)
	}
	if tracking[0].ShipTypeId == nil || *tracking[0].ShipTypeId != 587 {
		t.Errorf("ShipTypeId = %v, want 587", tracking[0].ShipTypeId)
	}
	if tracking[0].LogonDate == nil {
		t.Errorf("LogonDate = nil, want non-nil")
	}
	if tracking[0].LogoffDate == nil {
		t.Errorf("LogoffDate = nil, want non-nil")
	}
	if tracking[0].StartDate == nil {
		t.Errorf("StartDate = nil, want non-nil")
	}
}

func TestCorporationRoles_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"character_id": 2112625428,
				"roles": ["CEO", "Station_Manager"],
				"roles_at_base": ["Director"],
				"roles_at_hq": ["Accountant"],
				"roles_at_other": ["Factory_Manager"],
				"grantable_roles": ["Diplomat"],
				"grantable_roles_at_base": ["Tech_Officer"],
				"grantable_roles_at_hq": ["Communications_Officer"],
				"grantable_roles_at_other": ["Hangar_Manager"]
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	roles, err := client.CorporationRoles(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationRoles returned error: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("roles length = %d, want 1", len(roles))
	}
	r := roles[0]
	if len(r.Roles) != 2 || r.Roles[0] != "CEO" {
		t.Errorf("Roles = %v, want [CEO Station_Manager]", r.Roles)
	}
	if len(r.RolesAtBase) != 1 || r.RolesAtBase[0] != "Director" {
		t.Errorf("RolesAtBase = %v, want [Director]", r.RolesAtBase)
	}
	if len(r.RolesAtHq) != 1 || r.RolesAtHq[0] != "Accountant" {
		t.Errorf("RolesAtHq = %v, want [Accountant]", r.RolesAtHq)
	}
	if len(r.RolesAtOther) != 1 || r.RolesAtOther[0] != "Factory_Manager" {
		t.Errorf("RolesAtOther = %v, want [Factory_Manager]", r.RolesAtOther)
	}
	if len(r.GrantableRoles) != 1 || r.GrantableRoles[0] != "Diplomat" {
		t.Errorf("GrantableRoles = %v, want [Diplomat]", r.GrantableRoles)
	}
	if len(r.GrantableRolesAtBase) != 1 || r.GrantableRolesAtBase[0] != "Tech_Officer" {
		t.Errorf("GrantableRolesAtBase = %v, want [Tech_Officer]", r.GrantableRolesAtBase)
	}
	if len(r.GrantableRolesAtHq) != 1 || r.GrantableRolesAtHq[0] != "Communications_Officer" {
		t.Errorf("GrantableRolesAtHq = %v, want [Communications_Officer]", r.GrantableRolesAtHq)
	}
	if len(r.GrantableRolesAtOther) != 1 || r.GrantableRolesAtOther[0] != "Hangar_Manager" {
		t.Errorf("GrantableRolesAtOther = %v, want [Hangar_Manager]", r.GrantableRolesAtOther)
	}
}

func TestCorporationRolesHistory_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[
			{
				"character_id": 2112625428,
				"issuer_id": 2112625427,
				"changed_at": "2026-07-01T10:00:00Z",
				"role_type": "grantable_at_hq",
				"old_roles": ["Diplomat"],
				"new_roles": ["Diplomat", "CEO"]
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	_, _, err := client.CorporationRolesHistory(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationRolesHistory returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
}

func TestCorporationRolesHistory_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[
			{
				"character_id": 2112625428,
				"issuer_id": 2112625427,
				"changed_at": "2026-07-01T10:00:00Z",
				"role_type": "grantable_at_hq",
				"old_roles": ["Diplomat"],
				"new_roles": ["Diplomat", "CEO"]
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	history, totalPages, err := client.CorporationRolesHistory(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationRolesHistory returned error: %v", err)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(history) != 1 {
		t.Errorf("history length = %d, want 1", len(history))
	}
	if history[0].CharacterId != 2112625428 {
		t.Errorf("history[0].CharacterId = %d, want 2112625428", history[0].CharacterId)
	}
}

func TestCorporationTitles_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"title_id": 1, "name": "Chief Executive Officer"},
			{"title_id": 2, "name": "Director"}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	titles, err := client.CorporationTitles(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationTitles returned error: %v", err)
	}
	if len(titles) != 2 {
		t.Errorf("titles length = %d, want 2", len(titles))
	}
	if titles[0].TitleId == nil || *titles[0].TitleId != 1 {
		t.Errorf("titles[0].TitleId = %v, want 1", titles[0].TitleId)
	}
	if titles[0].Name == nil || *titles[0].Name != "Chief Executive Officer" {
		t.Errorf("titles[0].Name = %v, want \"Chief Executive Officer\"", titles[0].Name)
	}
}

func TestCorporationDivisions_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"hangar": [
				{"division": 1, "name": "Hangar 1"},
				{"division": 2, "name": "Hangar 2"}
			],
			"wallet": [
				{"division": 1, "name": "Wallet 1"}
			]
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	divisions, err := client.CorporationDivisions(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationDivisions returned error: %v", err)
	}
	if len(divisions.Hangar) != 2 {
		t.Errorf("hangar length = %d, want 2", len(divisions.Hangar))
	}
	if divisions.Hangar[0].Division == nil || *divisions.Hangar[0].Division != 1 {
		t.Errorf("hangar[0].Division = %v, want 1", divisions.Hangar[0].Division)
	}
	if divisions.Hangar[0].Name == nil || *divisions.Hangar[0].Name != "Hangar 1" {
		t.Errorf("hangar[0].Name = %v, want \"Hangar 1\"", divisions.Hangar[0].Name)
	}
	if len(divisions.Wallet) != 1 {
		t.Errorf("wallet length = %d, want 1", len(divisions.Wallet))
	}
}

func TestCorporationShareholders_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[
			{"shareholder_id": 2112625428, "shareholder_type": "character", "share_count": 500},
			{"shareholder_id": 98000001, "shareholder_type": "corporation", "share_count": 300}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	shareholders, totalPages, err := client.CorporationShareholders(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationShareholders returned error: %v", err)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(shareholders) != 2 {
		t.Errorf("shareholders length = %d, want 2", len(shareholders))
	}
	if shareholders[0].ShareholderId != 2112625428 {
		t.Errorf("shareholders[0].ShareholderId = %d, want 2112625428", shareholders[0].ShareholderId)
	}
	if shareholders[0].ShareholderType != "character" {
		t.Errorf("shareholders[0].ShareholderType = %s, want \"character\"", shareholders[0].ShareholderType)
	}
	if shareholders[0].ShareCount != 500 {
		t.Errorf("shareholders[0].ShareCount = %d, want 500", shareholders[0].ShareCount)
	}
}

func TestCorporationStandings_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[
			{"from_id": 1000001, "from_type": "npc_corp", "standing": 5.5},
			{"from_id": 500001, "from_type": "faction", "standing": -2.3}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	standings, totalPages, err := client.CorporationStandings(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationStandings returned error: %v", err)
	}
	if totalPages != 1 {
		t.Errorf("totalPages = %d, want 1", totalPages)
	}
	if len(standings) != 2 {
		t.Errorf("standings length = %d, want 2", len(standings))
	}
	if standings[0].FromId != 1000001 {
		t.Errorf("standings[0].FromId = %d, want 1000001", standings[0].FromId)
	}
	if standings[0].FromType != "npc_corp" {
		t.Errorf("standings[0].FromType = %s, want \"npc_corp\"", standings[0].FromType)
	}
	if standings[0].Standing != 5.5 {
		t.Errorf("standings[0].Standing = %f, want 5.5", standings[0].Standing)
	}
}

func TestCorporationBlueprints_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"item_id": 1000000001,
				"location_flag": "Hangar",
				"location_id": 60003760,
				"material_efficiency": 10,
				"quantity": -1,
				"runs": 100,
				"time_efficiency": 20,
				"type_id": 1
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	blueprints, _, err := client.CorporationBlueprints(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationBlueprints returned error: %v", err)
	}
	if len(blueprints) != 1 {
		t.Errorf("blueprints length = %d, want 1", len(blueprints))
	}
	if blueprints[0].ItemId != 1000000001 {
		t.Errorf("ItemId = %d, want 1000000001", blueprints[0].ItemId)
	}
	if blueprints[0].LocationFlag != "Hangar" {
		t.Errorf("LocationFlag = %s, want \"Hangar\"", blueprints[0].LocationFlag)
	}
	if blueprints[0].LocationId != 60003760 {
		t.Errorf("LocationId = %d, want 60003760", blueprints[0].LocationId)
	}
	if blueprints[0].MaterialEfficiency != 10 {
		t.Errorf("MaterialEfficiency = %d, want 10", blueprints[0].MaterialEfficiency)
	}
	if blueprints[0].Quantity != -1 {
		t.Errorf("Quantity = %d, want -1", blueprints[0].Quantity)
	}
	if blueprints[0].Runs != 100 {
		t.Errorf("Runs = %d, want 100", blueprints[0].Runs)
	}
	if blueprints[0].TimeEfficiency != 20 {
		t.Errorf("TimeEfficiency = %d, want 20", blueprints[0].TimeEfficiency)
	}
	if blueprints[0].TypeId != 1 {
		t.Errorf("TypeId = %d, want 1", blueprints[0].TypeId)
	}
}

func TestCorporationBlueprints_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	_, _, err := client.CorporationBlueprints(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationBlueprints returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
}

func TestCorporationContainerLogs_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"action": "add",
				"character_id": 2112625428,
				"container_id": 1000000001,
				"container_type_id": 587,
				"location_flag": "Hangar",
				"location_id": 60003760,
				"logged_at": "2026-07-01T10:00:00Z",
				"new_config_bitmask": 1,
				"old_config_bitmask": 0,
				"password_type": "general",
				"quantity": 5,
				"type_id": 34
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	logs, _, err := client.CorporationContainerLogs(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationContainerLogs returned error: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("logs length = %d, want 1", len(logs))
	}
	if logs[0].NewConfigBitmask == nil || *logs[0].NewConfigBitmask != 1 {
		t.Errorf("NewConfigBitmask = %v, want 1", logs[0].NewConfigBitmask)
	}
	if logs[0].OldConfigBitmask == nil || *logs[0].OldConfigBitmask != 0 {
		t.Errorf("OldConfigBitmask = %v, want 0", logs[0].OldConfigBitmask)
	}
	if logs[0].PasswordType == nil || *logs[0].PasswordType != "general" {
		t.Errorf("PasswordType = %v, want \"general\"", logs[0].PasswordType)
	}
	if logs[0].Quantity == nil || *logs[0].Quantity != 5 {
		t.Errorf("Quantity = %v, want 5", logs[0].Quantity)
	}
	if logs[0].TypeId == nil || *logs[0].TypeId != 34 {
		t.Errorf("TypeId = %v, want 34", logs[0].TypeId)
	}
}

func TestCorporationContainerLogs_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[
			{
				"action": "add",
				"character_id": 2112625428,
				"container_id": 1000000001,
				"container_type_id": 587,
				"location_flag": "Hangar",
				"location_id": 60003760,
				"logged_at": "2026-07-01T10:00:00Z"
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	logs, totalPages, err := client.CorporationContainerLogs(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationContainerLogs returned error: %v", err)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(logs) != 1 {
		t.Errorf("logs length = %d, want 1", len(logs))
	}
}

func TestCorporationFacilities_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"facility_id": 1000000001, "system_id": 30000142, "type_id": 2502}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	facilities, err := client.CorporationFacilities(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationFacilities returned error: %v", err)
	}
	if len(facilities) != 1 {
		t.Errorf("facilities length = %d, want 1", len(facilities))
	}
	if facilities[0].FacilityId != 1000000001 {
		t.Errorf("FacilityId = %d, want 1000000001", facilities[0].FacilityId)
	}
	if facilities[0].SystemId != 30000142 {
		t.Errorf("SystemId = %d, want 30000142", facilities[0].SystemId)
	}
	if facilities[0].TypeId != 2502 {
		t.Errorf("TypeId = %d, want 2502", facilities[0].TypeId)
	}
}

func TestCorporationMedals_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[
			{
				"created_at": "2026-01-01T00:00:00Z",
				"creator_id": 2112625428,
				"description": "Test Medal",
				"medal_id": 1,
				"title": "Elite"
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	medals, totalPages, err := client.CorporationMedals(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationMedals returned error: %v", err)
	}
	if totalPages != 1 {
		t.Errorf("totalPages = %d, want 1", totalPages)
	}
	if len(medals) != 1 {
		t.Errorf("medals length = %d, want 1", len(medals))
	}
	wantCreatedAt, err := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse returned error: %v", err)
	}
	if !medals[0].CreatedAt.Equal(wantCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", medals[0].CreatedAt, wantCreatedAt)
	}
	if medals[0].CreatorId != 2112625428 {
		t.Errorf("CreatorId = %d, want 2112625428", medals[0].CreatorId)
	}
	if medals[0].Description != "Test Medal" {
		t.Errorf("Description = %s, want \"Test Medal\"", medals[0].Description)
	}
	if medals[0].MedalId != 1 {
		t.Errorf("MedalId = %d, want 1", medals[0].MedalId)
	}
	if medals[0].Title != "Elite" {
		t.Errorf("Title = %s, want \"Elite\"", medals[0].Title)
	}
}

func TestCorporationMedalsIssued_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[
			{
				"character_id": 2112625428,
				"issued_at": "2026-07-01T10:00:00Z",
				"issuer_id": 2112625427,
				"medal_id": 1,
				"reason": "Outstanding service",
				"status": "public"
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	medalsIssued, totalPages, err := client.CorporationMedalsIssued(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationMedalsIssued returned error: %v", err)
	}
	if totalPages != 1 {
		t.Errorf("totalPages = %d, want 1", totalPages)
	}
	if len(medalsIssued) != 1 {
		t.Errorf("medalsIssued length = %d, want 1", len(medalsIssued))
	}
	if medalsIssued[0].CharacterId != 2112625428 {
		t.Errorf("CharacterId = %d, want 2112625428", medalsIssued[0].CharacterId)
	}
	wantIssuedAt, err := time.Parse(time.RFC3339, "2026-07-01T10:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse returned error: %v", err)
	}
	if !medalsIssued[0].IssuedAt.Equal(wantIssuedAt) {
		t.Errorf("IssuedAt = %v, want %v", medalsIssued[0].IssuedAt, wantIssuedAt)
	}
	if medalsIssued[0].IssuerId != 2112625427 {
		t.Errorf("IssuerId = %d, want 2112625427", medalsIssued[0].IssuerId)
	}
	if medalsIssued[0].MedalId != 1 {
		t.Errorf("MedalId = %d, want 1", medalsIssued[0].MedalId)
	}
	if medalsIssued[0].Reason != "Outstanding service" {
		t.Errorf("Reason = %s, want \"Outstanding service\"", medalsIssued[0].Reason)
	}
	if medalsIssued[0].Status != "public" {
		t.Errorf("Status = %s, want \"public\"", medalsIssued[0].Status)
	}
}

func TestCorporationStarbases_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[
			{
				"starbase_id": 1000000001,
				"system_id": 30000142,
				"type_id": 16213,
				"moon_id": 40000001,
				"onlined_since": "2026-01-01T00:00:00Z",
				"reinforced_until": "2026-07-08T00:00:00Z",
				"state": "online",
				"unanchor_at": "2026-08-01T00:00:00Z"
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	starbases, totalPages, err := client.CorporationStarbases(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationStarbases returned error: %v", err)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(starbases) != 1 {
		t.Errorf("starbases length = %d, want 1", len(starbases))
	}
	if starbases[0].MoonId == nil || *starbases[0].MoonId != 40000001 {
		t.Errorf("MoonId = %v, want 40000001", starbases[0].MoonId)
	}
	if starbases[0].OnlinedSince == nil {
		t.Errorf("OnlinedSince = nil, want non-nil")
	}
	if starbases[0].ReinforcedUntil == nil {
		t.Errorf("ReinforcedUntil = nil, want non-nil")
	}
	if starbases[0].State == nil || *starbases[0].State != "online" {
		t.Errorf("State = %v, want \"online\"", starbases[0].State)
	}
	if starbases[0].UnanchorAt == nil {
		t.Errorf("UnanchorAt = nil, want non-nil")
	}
}

func TestCorporationStarbaseDetail_ParsesResponseAndSendsSystemId(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"allow_alliance_members": false,
			"allow_corporation_members": true,
			"anchor": "alliance",
			"attack_if_at_war": false,
			"attack_if_other_security_status_dropping": false,
			"fuel_bay_take": "corporation",
			"fuel_bay_view": "corporation",
			"fuels": [
				{"quantity": 100, "type_id": 4051}
			],
			"offline": "corporation",
			"online": "corporation",
			"unanchor": "corporation",
			"use_alliance_standings": false
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CorporationStarbaseDetail(context.Background(), 98000001, 1000000001, 30000142)
	if err != nil {
		t.Fatalf("CorporationStarbaseDetail returned error: %v", err)
	}
	if detail.AllowAllianceMembers != false {
		t.Errorf("AllowAllianceMembers = %v, want false", detail.AllowAllianceMembers)
	}
	if detail.AllowCorporationMembers != true {
		t.Errorf("AllowCorporationMembers = %v, want true", detail.AllowCorporationMembers)
	}
	if detail.Anchor != "alliance" {
		t.Errorf("Anchor = %s, want \"alliance\"", detail.Anchor)
	}
	if detail.AttackIfAtWar != false {
		t.Errorf("AttackIfAtWar = %v, want false", detail.AttackIfAtWar)
	}
	if detail.AttackIfOtherSecurityStatusDropping != false {
		t.Errorf("AttackIfOtherSecurityStatusDropping = %v, want false", detail.AttackIfOtherSecurityStatusDropping)
	}
	if detail.FuelBayTake != "corporation" {
		t.Errorf("FuelBayTake = %s, want \"corporation\"", detail.FuelBayTake)
	}
	if detail.FuelBayView != "corporation" {
		t.Errorf("FuelBayView = %s, want \"corporation\"", detail.FuelBayView)
	}
	if detail.Offline != "corporation" {
		t.Errorf("Offline = %s, want \"corporation\"", detail.Offline)
	}
	if detail.Online != "corporation" {
		t.Errorf("Online = %s, want \"corporation\"", detail.Online)
	}
	if detail.Unanchor != "corporation" {
		t.Errorf("Unanchor = %s, want \"corporation\"", detail.Unanchor)
	}
	if detail.UseAllianceStandings != false {
		t.Errorf("UseAllianceStandings = %v, want false", detail.UseAllianceStandings)
	}
	if detail.Fuels == nil || len(detail.Fuels) != 1 {
		t.Errorf("Fuels length = %v, want [1 fuel]", detail.Fuels)
	}
	if detail.Fuels[0].Quantity != 100 {
		t.Errorf("Fuels[0].Quantity = %d, want 100", detail.Fuels[0].Quantity)
	}
	if detail.Fuels[0].TypeId != 4051 {
		t.Errorf("Fuels[0].TypeId = %d, want 4051", detail.Fuels[0].TypeId)
	}
	if !strings.Contains(gotQuery, "system_id=30000142") {
		t.Errorf("query = %q, want to contain system_id=30000142", gotQuery)
	}
}

func TestCorporationStructures_ParsesResponseAndTotalPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[
			{
				"corporation_id": 98000001,
				"profile_id": 1,
				"state": "online",
				"structure_id": 1000000001,
				"system_id": 30000142,
				"type_id": 35832,
				"fuel_expires": "2026-08-15T00:00:00Z",
				"name": "Test Structure",
				"next_reinforce_apply": "2026-07-26T12:00:00Z",
				"next_reinforce_hour": 12,
				"reinforce_hour": 10,
				"state_timer_end": "2026-07-27T10:00:00Z",
				"state_timer_start": "2026-07-26T10:00:00Z",
				"unanchors_at": "2026-09-01T00:00:00Z",
				"services": [
					{"name": "Cloning", "state": "online"}
				]
			}
		]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	structures, totalPages, err := client.CorporationStructures(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationStructures returned error: %v", err)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(structures) != 1 {
		t.Errorf("structures length = %d, want 1", len(structures))
	}
	if structures[0].FuelExpires == nil {
		t.Errorf("FuelExpires = nil, want non-nil")
	}
	if structures[0].Name == nil || *structures[0].Name != "Test Structure" {
		t.Errorf("Name = %v, want \"Test Structure\"", structures[0].Name)
	}
	if structures[0].NextReinforceApply == nil {
		t.Errorf("NextReinforceApply = nil, want non-nil")
	}
	if structures[0].NextReinforceHour == nil || *structures[0].NextReinforceHour != 12 {
		t.Errorf("NextReinforceHour = %v, want 12", structures[0].NextReinforceHour)
	}
	if structures[0].ReinforceHour == nil || *structures[0].ReinforceHour != 10 {
		t.Errorf("ReinforceHour = %v, want 10", structures[0].ReinforceHour)
	}
	if structures[0].StateTimerEnd == nil {
		t.Errorf("StateTimerEnd = nil, want non-nil")
	}
	if structures[0].StateTimerStart == nil {
		t.Errorf("StateTimerStart = nil, want non-nil")
	}
	if structures[0].UnanchorsAt == nil {
		t.Errorf("UnanchorsAt = nil, want non-nil")
	}
	if structures[0].Services == nil || len(structures[0].Services) != 1 {
		t.Errorf("Services length = %v, want [1 service]", structures[0].Services)
	}
	if structures[0].Services[0].Name != "Cloning" {
		t.Errorf("Services[0].Name = %s, want \"Cloning\"", structures[0].Services[0].Name)
	}
	if structures[0].Services[0].State != "online" {
		t.Errorf("Services[0].State = %s, want \"online\"", structures[0].Services[0].State)
	}
	if structures[0].CorporationId != 98000001 {
		t.Errorf("CorporationId = %d, want 98000001", structures[0].CorporationId)
	}
	if structures[0].ProfileId != 1 {
		t.Errorf("ProfileId = %d, want 1", structures[0].ProfileId)
	}
	if structures[0].State != "online" {
		t.Errorf("State = %s, want \"online\"", structures[0].State)
	}
	if structures[0].StructureId != 1000000001 {
		t.Errorf("StructureId = %d, want 1000000001", structures[0].StructureId)
	}
	if structures[0].SystemId != 30000142 {
		t.Errorf("SystemId = %d, want 30000142", structures[0].SystemId)
	}
	if structures[0].TypeId != 35832 {
		t.Errorf("TypeId = %d, want 35832", structures[0].TypeId)
	}
}

func TestCorporationBlueprints_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, _, err := client.CorporationBlueprints(context.Background(), 98000001, 1); err == nil {
		t.Fatal("CorporationBlueprints returned nil error, want an error for HTTP 403")
	}
}
