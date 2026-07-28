package mcp_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

func connectedSession(t *testing.T, store *config.Store) (*mcpsdk.ClientSession, *mcpsdk.ServerSession) {
	t.Helper()
	sdeStore, err := sde.Open(filepath.Join(t.TempDir(), "sde.db"))
	if err != nil {
		t.Fatalf("sde.Open: %v", err)
	}
	t.Cleanup(func() { sdeStore.Close() })
	server := mcp.NewServer("test", "client-id", store, sdeStore)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() {
		clientSession.Close()
		serverSession.Wait()
	})
	return clientSession, serverSession
}

func TestServer_AllToolsRegistered(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	clientSession, _ := connectedSession(t, store)

	res, err := clientSession.ListTools(context.Background(), &mcpsdk.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"add_character", "list_characters", "remove_character", "get_wallet_balance",
		"list_access_lists", "get_access_list",
		"list_mercenary_tactical_operations", "get_mercenary_tactical_operation", "list_raidable_skyhooks",
		"list_alliances", "get_alliance", "list_alliance_corporations", "get_alliance_icons",
		"list_assets", "get_asset_locations", "get_asset_names",
		"list_corporation_assets", "get_corporation_asset_locations", "get_corporation_asset_names",
		"list_calendar_events", "get_calendar_event", "list_calendar_event_attendees", "respond_calendar_event",
		"get_character", "get_character_affiliations", "list_character_corporation_history", "get_character_portrait",
		"list_character_agents_research", "list_character_blueprints", "get_character_fatigue",
		"list_character_medals", "list_character_notifications", "list_character_notification_contacts",
		"get_character_roles", "list_character_standings", "list_character_titles", "get_character_cspa_cost",
		"get_character_clones", "list_character_implants",
		"list_character_contacts", "list_character_contact_labels",
		"list_corporation_contacts", "list_corporation_contact_labels",
		"list_alliance_contacts", "list_alliance_contact_labels",
		"add_character_contacts", "update_character_contacts", "remove_character_contacts",
		"list_character_contracts", "list_character_contract_bids", "list_character_contract_items",
		"list_corporation_contracts", "list_corporation_contract_bids", "list_corporation_contract_items",
		"list_public_contracts", "list_public_contract_bids", "list_public_contract_items",
		"get_corporation", "list_corporation_alliance_history", "get_corporation_icons", "list_npc_corporations",
		"list_corporation_members", "get_corporation_members_limit", "list_corporation_member_titles",
		"list_corporation_member_tracking", "list_corporation_roles", "list_corporation_roles_history",
		"list_corporation_titles", "get_corporation_divisions", "list_corporation_shareholders",
		"list_corporation_standings",
		"list_corporation_blueprints", "list_corporation_container_logs", "list_corporation_facilities",
		"list_corporation_medals", "list_corporation_medals_issued", "list_corporation_starbases",
		"get_corporation_starbase", "list_corporation_structures",
		"get_corporation_project_contribution", "list_corporation_project_contributors",
		"get_corporation_project_detail", "list_corporation_projects",
		"list_dogma_attributes", "get_dogma_attribute", "get_dogma_dynamic_item",
		"list_dogma_effects", "get_dogma_effect",
		"get_character_fw_stats", "get_corporation_fw_stats", "get_fw_leaderboards",
		"get_fw_character_leaderboards", "get_fw_corporation_leaderboards",
		"list_fw_stats", "list_fw_systems", "list_fw_wars",
		"list_character_fittings", "create_character_fitting", "delete_character_fitting",
		"get_character_fleet", "get_fleet", "update_fleet", "list_fleet_members",
		"invite_fleet_member", "move_fleet_member", "kick_fleet_member", "list_fleet_wings",
		"create_fleet_wing", "rename_fleet_wing", "delete_fleet_wing", "create_fleet_squad",
		"rename_fleet_squad", "delete_fleet_squad",
		"list_character_freelance_jobs", "get_character_freelance_job_participation",
		"list_corporation_freelance_jobs", "list_corporation_freelance_job_participants",
		"get_freelance_job", "list_freelance_jobs",
		"list_incursions",
		"list_character_industry_jobs", "list_character_mining", "list_corporation_mining_extractions",
		"list_corporation_mining_observers", "get_corporation_mining_observer_detail",
		"list_corporation_industry_jobs", "list_industry_facilities", "list_industry_systems",
		"list_insurance_prices",
		"list_character_killmails_recent", "list_corporation_killmails_recent", "get_killmail",
		"get_character_location", "get_character_online", "get_character_ship",
		"list_character_loyalty_points", "list_loyalty_store_offers",
		"list_character_mail", "get_character_mail_labels", "list_character_mail_lists",
		"get_character_mail", "send_character_mail", "create_character_mail_label",
		"update_character_mail", "delete_character_mail", "delete_character_mail_label",
		"list_character_market_orders", "list_character_market_order_history",
		"list_corporation_market_orders", "list_corporation_market_order_history",
		"list_market_groups", "get_market_group", "list_market_prices", "get_market_history",
		"list_market_orders", "list_market_types", "list_structure_market_orders",
		"get_esi_changelog", "list_esi_compatibility_dates", "get_esi_name", "get_esi_status",
		"list_character_planets", "get_character_planet", "list_corporation_customs_offices",
		"get_planetary_schematic", "get_route", "search_character",
		"get_character_attributes", "list_character_skill_queue", "list_character_skills",
		"list_sovereignty_campaigns", "list_sovereignty_systems", "get_status",
		"list_character_mercenary_dens", "get_character_mercenary_den",
		"list_corporation_skyhooks", "get_corporation_skyhook",
		"list_corporation_sovereignty_hubs", "get_corporation_sovereignty_hub",
		"list_universe_ancestries", "list_universe_bloodlines", "list_universe_races", "list_universe_factions",
		"list_universe_categories", "get_universe_category", "list_universe_groups", "get_universe_group",
		"list_universe_types", "get_universe_type",
		"list_universe_regions", "get_universe_region", "list_universe_constellations", "get_universe_constellation",
		"list_universe_systems", "get_universe_system",
		"get_universe_asteroid_belt", "get_universe_moon", "get_universe_planet", "get_universe_stargate",
		"get_universe_star", "get_universe_station",
		"list_universe_graphics", "get_universe_graphic", "list_universe_structures", "get_universe_structure",
		"list_universe_system_jumps", "list_universe_system_kills", "resolve_universe_ids", "resolve_universe_names",
		"set_autopilot_waypoint", "open_contract_window", "open_information_window",
		"open_market_details_window", "open_new_mail_window", "list_wars", "get_war", "list_war_killmails",
	} {
		if !names[want] {
			t.Errorf("missing tool %q, got %v", want, names)
		}
	}
}

func TestServer_RemoveCharacterNotFound_SurfacesAsToolError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	clientSession, _ := connectedSession(t, store)

	res, err := clientSession.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "remove_character",
		Arguments: map[string]any{"characterId": 12345},
	})
	if err != nil {
		t.Fatalf("CallTool transport-level error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true for missing character, got %+v", res)
	}

	// GetError() always returns nil client-side (the SDK does not
	// serialize the underlying Go error across the wire) — the only
	// client-visible signal is IsError plus the human-readable text
	// content, so that's what this test asserts against.
	var text string
	for _, c := range res.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			text = tc.Text
		}
	}
	if !strings.Contains(text, "character not found") {
		t.Errorf("content text = %q, want it to contain %q", text, "character not found")
	}
}
