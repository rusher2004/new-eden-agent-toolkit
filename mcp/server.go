//go:generate go run ../tools/gentooldocs -out ../docs/reference/tools.md

// Package mcp builds neat's MCP server and registers its character tools.
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// Deps is everything NewServer needs from outside the package. The composition
// root (commands/mcp.go) builds the concrete config store, SDE cache and ESI
// clients and hands them over as these interfaces, so no tool in this package
// names a concrete producer type.
//
// ConfigStore, SDEStore and ESIClient are the unions of what every tool file
// needs. Each tool file declares its own small interface for the calls it
// actually makes, and the unions only have to cover them.
type Deps struct {
	// ClientID is the ESI OAuth2 client ID, used by the login flow.
	ClientID string
	Config   ConfigStore
	SDE      SDEStore
	// OpenESI returns an authenticated ESI client for a logged-in character.
	OpenESI func(ctx context.Context, characterID int32) (ESIClient, error)
	// OpenPublicESI returns an ESI client for public endpoints.
	OpenPublicESI func() ESIClient
}

// ESIClient is the union of the ESI calls the tool files make: it embeds each
// file's own small interface, so the composition root's *esi.Client must
// satisfy all of them (commands has a test that it does).
type ESIClient interface {
	loyaltyESI
	loyaltyStoreESI
}

// narrowPublicESI adapts Deps.OpenPublicESI the same way, for the tools that
// need no character. ESIClient embeds every C, so the assertion holds for any
// client the composition root can build. The client is opened per call, not
// when the server is built.
func narrowPublicESI[C any](open func() ESIClient) func() C {
	return func() C { return open().(C) }
}

// narrowESI adapts Deps.OpenESI to the small interface C one tool file needs.
// ESIClient embeds every such C, so the assertion cannot fail for a client that
// satisfies ESIClient; the error covers a client that was built wrongly.
func narrowESI[C any](open func(ctx context.Context, characterID int32) (ESIClient, error)) func(ctx context.Context, characterID int32) (C, error) {
	return func(ctx context.Context, characterID int32) (C, error) {
		var zero C
		c, err := open(ctx, characterID)
		if err != nil {
			return zero, err
		}
		narrowed, ok := c.(C)
		if !ok {
			return zero, fmt.Errorf("ESI client %T does not provide %T", c, &zero)
		}
		return narrowed, nil
	}
}

type ConfigStore interface {
	Load() (config.Config, error)
	Path() string
	Save(cfg config.Config) error
	SaveChanges(cfg config.Config) ([]config.Change, error)
	Update(change func(*config.Config) error) ([]config.Change, error)
}

type SDEStore interface {
	Ancestries(ctx context.Context, userAgent string) ([]sde.Ancestry, error)
	AsteroidBelt(ctx context.Context, userAgent string, id int64) (sde.AsteroidBelt, bool, error)
	AttributeInfos(ctx context.Context, userAgent string) (map[int64]sde.AttributeInfo, error)
	Bloodline(ctx context.Context, userAgent string, id int64) (sde.Bloodline, bool, error)
	Bloodlines(ctx context.Context, userAgent string) ([]sde.Bloodline, error)
	Blueprint(ctx context.Context, userAgent string, blueprintTypeID int64) (sde.Blueprint, bool, error)
	BlueprintSourcesForActivity(ctx context.Context, userAgent string, productTypeID, activityID int64) ([]sde.BlueprintSource, error)
	BlueprintsForProduct(ctx context.Context, userAgent string, productTypeID int64) ([]sde.BlueprintSource, error)
	Category(ctx context.Context, userAgent string, id int64) (sde.Category, bool, error)
	CategoryIDs(ctx context.Context, userAgent string) ([]int64, error)
	CompressedOres(ctx context.Context, userAgent string) ([]sde.Ore, error)
	Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error)
	ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error)
	DogmaTypes(ctx context.Context, userAgent string, ids []int64) (map[int64]sde.DogmaType, error)
	Faction(ctx context.Context, userAgent string, id int64) (sde.Faction, bool, error)
	Factions(ctx context.Context, userAgent string) ([]sde.Faction, error)
	Graphic(ctx context.Context, userAgent string, id int64) (sde.Graphic, bool, error)
	GraphicIDs(ctx context.Context, userAgent string) ([]int64, error)
	Group(ctx context.Context, userAgent string, id int64) (sde.Group, bool, error)
	GroupIDs(ctx context.Context, userAgent string) ([]int64, error)
	IndustryModifiers(ctx context.Context, userAgent string, typeID int64) ([]sde.IndustryModifierSource, error)
	Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error)
	NPCStationIndustryData(ctx context.Context, userAgent string, stationID int64) (sde.NPCStationIndustry, bool, error)
	Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error)
	PublishedTypesInGroups(ctx context.Context, userAgent string, groupIDs []int64) ([]sde.TypeBasic, error)
	PublishedTypesWithEffect(ctx context.Context, userAgent, effectName string) ([]sde.TypeBasic, error)
	PutAsteroidBelt(b sde.AsteroidBelt) error
	PutCategory(c sde.Category) error
	PutConstellation(c sde.Constellation) error
	PutGraphic(g sde.Graphic) error
	PutGroup(g sde.Group) error
	PutMoon(m sde.Moon) error
	PutPlanet(p sde.Planet) error
	PutRegion(r sde.Region) error
	PutStar(st sde.Star) error
	PutStargate(g sde.Stargate) error
	PutStation(st sde.Station) error
	PutSystem(sys sde.System) error
	PutType(t sde.Type) error
	Race(ctx context.Context, userAgent string, id int64) (sde.Race, bool, error)
	Races(ctx context.Context, userAgent string) ([]sde.Race, error)
	Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error)
	RegionIDs(ctx context.Context, userAgent string) ([]int64, error)
	ReprocessingMaterials(ctx context.Context, userAgent string, typeID int64) ([]sde.TypeQuantity, error)
	RigInfo(ctx context.Context, userAgent string, typeID int64) (sde.Rig, bool, error)
	RoutingSystems(ctx context.Context, userAgent string) ([]sde.RoutingSystem, error)
	SkinrComponent(ctx context.Context, userAgent string, id int64) (sde.SkinrComponent, bool, error)
	SkinrSlot(ctx context.Context, userAgent string, id int64) (sde.SkinrSlot, bool, error)
	Star(ctx context.Context, userAgent string, id int64) (sde.Star, bool, error)
	Stargate(ctx context.Context, userAgent string, id int64) (sde.Stargate, bool, error)
	StargateConnections(ctx context.Context, userAgent string) ([][2]int64, error)
	Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error)
	System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error)
	SystemIDs(ctx context.Context, userAgent string) ([]int64, error)
	TargetFilters(ctx context.Context, userAgent string) (map[int64]sde.TargetFilter, error)
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
	TypeAttributes(ctx context.Context, userAgent string, typeID int64) (map[int64]float64, error)
	TypeBasics(ctx context.Context, userAgent string, ids []int64) (map[int64]sde.TypeBasic, error)
	TypeIDs(ctx context.Context, userAgent string) ([]int64, error)
	TypesByName(ctx context.Context, userAgent string, names []string) (map[string]int64, error)
}

// NewServer builds neat's MCP server with add_character, list_scope_groups, list_characters,
// remove_character, get_wallet_balance, list_character_wallet_journal,
// list_character_wallet_transactions, list_corporation_wallets,
// list_corporation_wallet_journal, list_corporation_wallet_transactions,
// list_access_lists, get_access_list,
// list_mercenary_tactical_operations, get_mercenary_tactical_operation,
// list_raidable_skyhooks, list_alliances, get_alliance,
// list_alliance_corporations, get_alliance_icons, list_assets,
// get_asset_locations, get_asset_names, list_corporation_assets,
// get_corporation_asset_locations, get_corporation_asset_names,
// list_calendar_events, get_calendar_event, list_calendar_event_attendees,
// respond_calendar_event, get_character, get_character_affiliations,
// list_character_corporation_history, get_character_portrait,
// list_character_agents_research, list_character_blueprints,
// get_character_fatigue, list_character_medals, list_character_notifications,
// list_character_notification_contacts, get_character_roles,
// list_character_standings, list_character_titles, get_character_cspa_cost,
// get_character_clones, list_character_implants, list_character_contacts,
// list_character_contact_labels, list_corporation_contacts,
// list_corporation_contact_labels, list_alliance_contacts,
// list_alliance_contact_labels, add_character_contacts,
// update_character_contacts, remove_character_contacts, list_character_contracts,
// list_character_contract_bids, list_character_contract_items, list_corporation_contracts,
// list_corporation_contract_bids, list_corporation_contract_items, list_public_contracts,
// list_public_contract_bids, list_public_contract_items, get_corporation,
// list_corporation_alliance_history, get_corporation_icons, list_npc_corporations,
// list_corporation_members, get_corporation_members_limit, list_corporation_member_titles,
// list_corporation_member_tracking, list_corporation_roles, list_corporation_roles_history,
// list_corporation_titles, get_corporation_divisions, list_corporation_shareholders,
// list_corporation_standings, list_corporation_blueprints, list_corporation_container_logs,
// list_corporation_facilities, list_corporation_medals, list_corporation_medals_issued,
// list_corporation_starbases, get_corporation_starbase, list_corporation_structures,
// get_corporation_project_contribution, list_corporation_project_contributors,
// get_corporation_project_detail, list_corporation_projects, list_dogma_attributes,
// get_dogma_attribute, get_dogma_dynamic_item, list_dogma_effects, get_dogma_effect,
// get_character_fw_stats, get_corporation_fw_stats, get_fw_leaderboards,
// get_fw_character_leaderboards, get_fw_corporation_leaderboards,
// list_fw_stats, list_fw_systems, list_fw_wars, list_character_fittings, create_character_fitting,
// delete_character_fitting, get_character_fleet, get_fleet, update_fleet,
// list_fleet_members, invite_fleet_member, move_fleet_member, kick_fleet_member, list_fleet_wings,
// create_fleet_wing, rename_fleet_wing, delete_fleet_wing, create_fleet_squad, rename_fleet_squad,
// delete_fleet_squad, list_character_freelance_jobs,
// get_character_freelance_job_participation, list_corporation_freelance_jobs,
// list_corporation_freelance_job_participants, get_freelance_job, list_freelance_jobs,
// list_incursions, list_character_industry_jobs, list_character_mining,
// list_corporation_mining_extractions, list_corporation_mining_observers,
// get_corporation_mining_observer_detail, list_corporation_industry_jobs, list_industry_facilities,
// list_industry_systems, list_insurance_prices, list_character_killmails_recent,
// list_corporation_killmails_recent, get_killmail, get_character_location,
// get_character_online, get_character_ship, list_character_loyalty_points,
// list_loyalty_store_offers, list_character_mail, get_character_mail_labels,
// list_character_mail_lists, get_character_mail, send_character_mail,
// create_character_mail_label, update_character_mail, delete_character_mail,
// delete_character_mail_label, list_character_market_orders, list_character_market_order_history,
// list_corporation_market_orders, list_corporation_market_order_history, list_market_groups,
// get_market_group, list_market_prices, get_market_history, list_market_orders, list_market_types,
// list_structure_market_orders, get_esi_changelog, list_esi_compatibility_dates, get_esi_name,
// get_esi_status, list_character_planets, get_character_planet, list_corporation_customs_offices,
// get_planetary_schematic, get_route, search_character, get_character_attributes,
// list_character_skill_queue, list_character_skills, list_sovereignty_campaigns,
// list_sovereignty_systems, get_status, list_character_mercenary_dens, get_character_mercenary_den,
// list_corporation_skyhooks, get_corporation_skyhook, list_corporation_sovereignty_hubs,
// get_corporation_sovereignty_hub, list_universe_ancestries, list_universe_bloodlines,
// list_universe_races, list_universe_factions, list_universe_categories, get_universe_category,
// list_universe_groups, get_universe_group, list_universe_types, get_universe_type,
// list_universe_regions, get_universe_region, list_universe_constellations,
// get_universe_constellation, list_universe_systems, get_universe_system,
// get_universe_asteroid_belt, get_universe_moon, get_universe_planet, get_universe_stargate,
// get_universe_star, get_universe_station, list_universe_graphics, get_universe_graphic,
// list_universe_structures, get_universe_structure, list_universe_system_jumps,
// list_universe_system_kills, resolve_universe_ids, resolve_universe_names,
// set_autopilot_waypoint, open_contract_window, open_information_window,
// open_market_details_window, open_new_mail_window, list_wars, get_war, list_war_killmails,
// list_military_campaigns, get_military_campaign, list_military_campaign_objectives,
// get_military_campaign_objective, list_character_military_campaign_objectives,
// get_character_military_campaign_objective_participation, list_character_skinr_licenses,
// list_character_skinr_component_licenses, get_skinr, list_paragon_hub_skinr_listings,
// list_character_paragon_hub_skinr_listings, list_paragon_hub_skinr_listings_for_alliance,
// list_paragon_hub_skinr_listings_for_corporation, and
// list_paragon_hub_skinr_listings_for_character registered.
func NewServer(version string, deps Deps) *mcpsdk.Server {
	clientID, store, sdeStore := deps.ClientID, deps.Config, deps.SDE
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: version}, &mcpsdk.ServerOptions{Instructions: Instructions})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "add_character",
		Description: "Log in EVE Online characters via ESI OAuth2. Opens the user's browser at EVE's login page; afterwards the browser shows a neat page listing their characters, where they can log in more characters before clicking Done or closing the tab. Blocks until then and returns every character added, with the scope groups each granted. " +
			"Every login grants the basic scope group plus scopeGroups: before calling, ask the user what they'll use neat for, including whether they want their corporation's data, and pick groups from list_scope_groups (or use only basic if they don't know). A login replaces each character's access with exactly what it requested, so to add a group to characters already logged in, either pass characterId or list every group they should end up with; list_characters doesn't show a character's current groups, so if you don't know them, pass characterId. " +
			"If no browser tab opens, the user can open http://127.0.0.1:48211/ themselves while the tool waits. " +
			"To grant more groups to a character later (e.g. after a tool fails because a scope is missing), pass its characterId: its existing scopes are kept. Check the result's warnings: they say when the user logged in as a different character, or when the character's earlier scope groups couldn't be read and may need granting again. Re-adding a known character refreshes its login.",
	}, addCharacterHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_scope_groups",
		Description: "List the scope groups add_character can grant a character: each group's name, what it lets neat do, and its ESI scopes. The basic group is always granted.",
	}, listScopeGroupsHandler())

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_characters",
		Description: "List EVE Online characters currently logged in to neat, with the account label and notes the user gave each. Characters with the same account label are on the same EVE account and can't be logged in at the same time. Its warnings name characters whose account is unknown: ask the user and save the answer with update_character_details.",
	}, listCharactersHandler(store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "update_character_details",
		Description: "Record what the user tells you about their characters: which EVE account each is on (an account label shared by the characters on that account), and notes on what a character is for. Also renames an account label for every character that has it. Ask about accounts whenever list_characters or add_character warns that one is unknown, and save notes the user gives you. Returns every value changed, old and new, so you can tell the user what was saved. Writes to ~/.neat/characters.yaml only.",
	}, updateCharacterDetailsHandler(store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "remove_character",
		Description: "Remove a logged-in EVE Online character, deleting its stored credentials.",
	}, removeCharacterHandler(store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_wallet_balance",
		Description: "Get a logged-in EVE Online character's current ISK wallet balance.",
	}, getWalletBalanceHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_wallet_journal",
		Description: "List a logged-in EVE Online character's wallet journal entries (ISK gains/losses with reasons), paginated.",
	}, listCharacterWalletJournalHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_wallet_transactions",
		Description: "List a logged-in EVE Online character's market wallet transactions (buys/sells), newest first.",
	}, listCharacterWalletTransactionsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_wallets",
		Description: "List a corporation's wallet division balances, using a logged-in character's roles to authorize.",
	}, listCorporationWalletsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_wallet_journal",
		Description: "List a corporation wallet division's journal entries, using a logged-in character's roles to authorize.",
	}, listCorporationWalletJournalHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_wallet_transactions",
		Description: "List a corporation wallet division's market transactions, using a logged-in character's roles to authorize.",
	}, listCorporationWalletTransactionsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_access_lists",
		Description: "List a logged-in EVE Online character's structure/fleet access list IDs.",
	}, listAccessListsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_access_list",
		Description: "Get the name, description, and membership detail of one of a character's access lists.",
	}, getAccessListHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_mercenary_tactical_operations",
		Description: "List a logged-in EVE Online character's mercenary tactical operations.",
	}, listMercenaryTacticalOperationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_mercenary_tactical_operation",
		Description: "Get detail (dungeon type, mercenary den, state, expiry) for one mercenary tactical operation.",
	}, getMercenaryTacticalOperationHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_raidable_skyhooks",
		Description: "List upcoming raidable skyhooks (public data, no character scope required).",
	}, listRaidableSkyhooksHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliances",
		Description: "List every alliance ID currently in EVE Online (public data, no character required).",
	}, listAlliancesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_alliance",
		Description: "Get an alliance's name, ticker, founding info, and executor corporation (public data, no character required).",
	}, getAllianceHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliance_corporations",
		Description: "List the corporation IDs currently in an alliance (public data, no character required).",
	}, listAllianceCorporationsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_alliance_icons",
		Description: "Get an alliance's icon URLs (public data, no character required).",
	}, getAllianceIconsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_assets",
		Description: "List a logged-in EVE Online character's assets.",
	}, listAssetsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_asset_locations",
		Description: "Resolve a set of a character's asset item IDs into their positions.",
	}, getAssetLocationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_asset_names",
		Description: "Resolve a set of a character's asset item IDs into their player-assigned custom names.",
	}, getAssetNamesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_assets",
		Description: "List a corporation's assets, using a logged-in character's roles to authorize.",
	}, listCorporationAssetsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_location_assets",
		Description: "List everything the user owns at one NPC station or Upwell structure, across every logged-in character (and a corporation's hangars, through a character with the roles), in one call instead of paging list_assets. Items are listed in tree order with what's inside each container and ship, and every volume comes worked out: each stack's volume as it is now, assembled ships' assembled and repackaged volumes, what comes out of each ship to repackage it, and the rigs that repackaging destroys. Totals give the volume to move with ships repackaged or carried assembled, overall and per owner in sources: items in one character's personal hangar can only be hauled by another pilot once its owner gives them over with an item exchange contract. Use it whenever a plan moves things out of a station or structure.",
	}, listLocationAssetsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_asset_locations",
		Description: "Resolve a set of a corporation's asset item IDs into their positions.",
	}, getCorporationAssetLocationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_asset_names",
		Description: "Resolve a set of a corporation's asset item IDs into their player-assigned custom names.",
	}, getCorporationAssetNamesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_calendar_events",
		Description: "List a logged-in EVE Online character's calendar events.",
	}, listCalendarEventsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_calendar_event",
		Description: "Get detail (date, duration, owner, description, RSVP status) for one calendar event.",
	}, getCalendarEventHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_calendar_event_attendees",
		Description: "List the attendees and their RSVP status for one calendar event.",
	}, listCalendarEventAttendeesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "respond_calendar_event",
		Description: "Respond to a calendar event invite (accepted/declined/tentative) — this changes your RSVP status on EVE's server.",
	}, respondCalendarEventHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character",
		Description: "Get public info (name, corporation, bloodline, security status) about an arbitrary character (public data, no character required).",
	}, getCharacterHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_affiliations",
		Description: "Resolve a set of character IDs to their current corporation/alliance/faction (public data, no character required).",
	}, getCharacterAffiliationsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_corporation_history",
		Description: "List an arbitrary character's corporation history (public data, no character required).",
	}, listCharacterCorporationHistoryHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_portrait",
		Description: "Get an arbitrary character's portrait URLs (public data, no character required).",
	}, getCharacterPortraitHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_agents_research",
		Description: "List a logged-in EVE Online character's agent research progress.",
	}, listCharacterAgentsResearchHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_blueprints",
		Description: "List a logged-in EVE Online character's blueprints.",
	}, listCharacterBlueprintsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_fatigue",
		Description: "Get a logged-in EVE Online character's jump fatigue state.",
	}, getCharacterFatigueHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_medals",
		Description: "List a logged-in EVE Online character's medals.",
	}, listCharacterMedalsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_notifications",
		Description: "List a logged-in EVE Online character's notifications.",
	}, listCharacterNotificationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_notification_contacts",
		Description: "List a logged-in EVE Online character's contact-standing notifications.",
	}, listCharacterNotificationContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_roles",
		Description: "Get a logged-in EVE Online character's corporation roles.",
	}, getCharacterRolesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_standings",
		Description: "List a logged-in EVE Online character's standings.",
	}, listCharacterStandingsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_titles",
		Description: "List a logged-in EVE Online character's corporation titles.",
	}, listCharacterTitlesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_cspa_cost",
		Description: "Calculate the CSPA charge cost for a logged-in character to contact a set of target characters.",
	}, getCharacterCSPACostHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_clones",
		Description: "Get a logged-in EVE Online character's jump clones and home location.",
	}, getCharacterClonesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_implants",
		Description: "List a logged-in EVE Online character's currently active implant type IDs.",
	}, listCharacterImplantsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contacts",
		Description: "List a logged-in EVE Online character's contacts.",
	}, listCharacterContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contact_labels",
		Description: "List a logged-in EVE Online character's contact labels.",
	}, listCharacterContactLabelsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contacts",
		Description: "List a corporation's contacts, using a logged-in character's roles to authorize.",
	}, listCorporationContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contact_labels",
		Description: "List a corporation's contact labels, using a logged-in character's roles to authorize.",
	}, listCorporationContactLabelsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliance_contacts",
		Description: "List an alliance's contacts, using a logged-in character's roles to authorize.",
	}, listAllianceContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliance_contact_labels",
		Description: "List an alliance's contact labels, using a logged-in character's roles to authorize.",
	}, listAllianceContactLabelsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "add_character_contacts",
		Description: "Add contacts to a character's contact list with a given standing — this creates new contacts on EVE's server.",
	}, addCharacterContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "update_character_contacts",
		Description: "Update the standing/label/watched status of existing contacts — this changes state on EVE's server.",
	}, updateCharacterContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "remove_character_contacts",
		Description: "Remove contacts from a character's contact list — this deletes contacts on EVE's server.",
	}, removeCharacterContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contracts",
		Description: "List a logged-in EVE Online character's contracts.",
	}, listCharacterContractsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contract_bids",
		Description: "List the bids on one of a logged-in character's contracts.",
	}, listCharacterContractBidsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contract_items",
		Description: "List the items on one of a logged-in character's contracts.",
	}, listCharacterContractItemsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contracts",
		Description: "List a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contract_bids",
		Description: "List the bids on one of a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractBidsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contract_items",
		Description: "List the items on one of a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractItemsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_public_contracts",
		Description: "List public contracts in a region (public data, no character required).",
	}, listPublicContractsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_public_contract_bids",
		Description: "List the bids on a public contract (public data, no character required).",
	}, listPublicContractBidsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_public_contract_items",
		Description: "List the items on a public contract (public data, no character required).",
	}, listPublicContractItemsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation",
		Description: "Get a corporation's public info (public data, no character required).",
	}, getCorporationHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_alliance_history",
		Description: "List a corporation's alliance membership history (public data, no character required).",
	}, listCorporationAllianceHistoryHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_icons",
		Description: "Get a corporation's icon URLs (public data, no character required).",
	}, getCorporationIconsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_npc_corporations",
		Description: "List every NPC corporation ID in the game (public data, no character required).",
	}, listNPCCorporationsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_members",
		Description: "List a corporation's member character IDs, using a logged-in character's roles to authorize.",
	}, listCorporationMembersHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_members_limit",
		Description: "Get a corporation's member limit, using a logged-in character's roles to authorize.",
	}, getCorporationMembersLimitHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_member_titles",
		Description: "List a corporation's members' assigned title IDs, using a logged-in character's roles to authorize.",
	}, listCorporationMemberTitlesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_member_tracking",
		Description: "List a corporation's member login/location/ship tracking data, using a logged-in character's roles to authorize.",
	}, listCorporationMemberTrackingHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_roles",
		Description: "List a corporation's members' corp roles, using a logged-in character's roles to authorize.",
	}, listCorporationRolesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_roles_history",
		Description: "List a corporation's role-change history, using a logged-in character's roles to authorize.",
	}, listCorporationRolesHistoryHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_titles",
		Description: "List a corporation's defined titles, using a logged-in character's roles to authorize.",
	}, listCorporationTitlesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_divisions",
		Description: "Get a corporation's hangar and wallet division names, using a logged-in character's roles to authorize.",
	}, getCorporationDivisionsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_shareholders",
		Description: "List a corporation's shareholders, using a logged-in character's roles to authorize.",
	}, listCorporationShareholdersHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_standings",
		Description: "List a corporation's standings toward NPC entities, using a logged-in character's roles to authorize.",
	}, listCorporationStandingsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_blueprints",
		Description: "List a corporation's blueprints, using a logged-in character's roles to authorize.",
	}, listCorporationBlueprintsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_container_logs",
		Description: "List a corporation's container access logs, using a logged-in character's roles to authorize.",
	}, listCorporationContainerLogsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_facilities",
		Description: "List a corporation's facilities, using a logged-in character's roles to authorize.",
	}, listCorporationFacilitiesHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_medals",
		Description: "List a corporation's defined medals, using a logged-in character's roles to authorize.",
	}, listCorporationMedalsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_medals_issued",
		Description: "List medals a corporation has issued to its members, using a logged-in character's roles to authorize.",
	}, listCorporationMedalsIssuedHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_starbases",
		Description: "List a corporation's starbases (POSes), using a logged-in character's roles to authorize.",
	}, listCorporationStarbasesHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_starbase",
		Description: "Get detail for one of a corporation's starbases (POSes), using a logged-in character's roles to authorize.",
	}, getCorporationStarbaseHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_structures",
		Description: "List a corporation's structures, using a logged-in character's roles to authorize.",
	}, listCorporationStructuresHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_project_contribution",
		Description: "Get a character's contribution to a corporation project, using a logged-in character's roles to authorize.",
	}, getCorporationProjectContributionHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_project_contributors",
		Description: "List the contributors to a corporation project, using a logged-in character's roles to authorize.",
	}, listCorporationProjectContributorsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_project_detail",
		Description: "Get the full details (configuration, progress, reward) of a corporation project, using a logged-in character's roles to authorize.",
	}, getCorporationProjectDetailHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_projects",
		Description: "List a corporation's projects, using a logged-in character's roles to authorize.",
	}, listCorporationProjectsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_dogma_attributes",
		Description: "List every dogma attribute ID currently defined (public data, no character required).",
	}, listDogmaAttributesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_dogma_attribute",
		Description: "Get detail for one dogma attribute (public data, no character required).",
	}, getDogmaAttributeHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_dogma_dynamic_item",
		Description: "Get detail for one mutated (\"abyssal\") item, identified by its source type and item ID (public data, no character required).",
	}, getDogmaDynamicItemHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_dogma_effects",
		Description: "List every dogma effect ID currently defined (public data, no character required).",
	}, listDogmaEffectsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_dogma_effect",
		Description: "Get detail for one dogma effect (public data, no character required).",
	}, getDogmaEffectHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_fw_stats",
		Description: "Get a logged-in EVE Online character's faction warfare stats.",
	}, getCharacterFwStatsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_fw_stats",
		Description: "Get a corporation's faction warfare stats, using a logged-in character's roles to authorize.",
	}, getCorporationFwStatsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fw_leaderboards",
		Description: "Get the top faction rankings by kills and victory points (public data, no character required).",
	}, getFwLeaderboardsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fw_character_leaderboards",
		Description: "Get the top pilot rankings by kills and victory points (public data, no character required).",
	}, getFwCharacterLeaderboardsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fw_corporation_leaderboards",
		Description: "Get the top corporation rankings by kills and victory points (public data, no character required).",
	}, getFwCorporationLeaderboardsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fw_stats",
		Description: "List every faction's faction warfare stats (public data, no character required).",
	}, listFwStatsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fw_systems",
		Description: "List every solar system's faction warfare occupation/contest state (public data, no character required).",
	}, listFwSystemsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fw_wars",
		Description: "List every faction warfare declaration (public data, no character required).",
	}, listFwWarsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_fittings",
		Description: "List a logged-in EVE Online character's saved ship fittings.",
	}, listCharacterFittingsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "create_character_fitting",
		Description: "Save a new ship fitting for a logged-in character — this creates a new fitting on EVE's server.",
	}, createCharacterFittingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_character_fitting",
		Description: "Delete one of a logged-in character's saved ship fittings — this deletes the fitting on EVE's server.",
	}, deleteCharacterFittingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_fleet",
		Description: "Get a logged-in EVE Online character's current fleet membership.",
	}, getCharacterFleetHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fleet",
		Description: "Get a fleet's settings, using a logged-in character's fleet membership to authorize.",
	}, getFleetHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "update_fleet",
		Description: "Update a fleet's settings — this changes state on EVE's server.",
	}, updateFleetHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fleet_members",
		Description: "List a fleet's members, using a logged-in character's fleet membership to authorize.",
	}, listFleetMembersHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "invite_fleet_member",
		Description: "Invite a character into a fleet — this changes state on EVE's server.",
	}, inviteFleetMemberHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "move_fleet_member",
		Description: "Move a fleet member to a new role, wing, or squad — this changes state on EVE's server.",
	}, moveFleetMemberHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "kick_fleet_member",
		Description: "Remove a member from a fleet — this changes state on EVE's server.",
	}, kickFleetMemberHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fleet_wings",
		Description: "List a fleet's wings and their nested squads, using a logged-in character's fleet membership to authorize.",
	}, listFleetWingsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "create_fleet_wing",
		Description: "Create a new wing in a fleet — this changes state on EVE's server.",
	}, createFleetWingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rename_fleet_wing",
		Description: "Rename a fleet wing — this changes state on EVE's server.",
	}, renameFleetWingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_fleet_wing",
		Description: "Delete a fleet wing — this changes state on EVE's server.",
	}, deleteFleetWingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "create_fleet_squad",
		Description: "Create a new squad within a fleet wing — this changes state on EVE's server.",
	}, createFleetSquadHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rename_fleet_squad",
		Description: "Rename a fleet squad — this changes state on EVE's server.",
	}, renameFleetSquadHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_fleet_squad",
		Description: "Delete a fleet squad — this changes state on EVE's server.",
	}, deleteFleetSquadHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_freelance_jobs",
		Description: "List a logged-in EVE Online character's freelance jobs.",
	}, listCharacterFreelanceJobsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_freelance_job_participation",
		Description: "Get a logged-in character's contribution to a freelance job.",
	}, getCharacterFreelanceJobParticipationHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_freelance_jobs",
		Description: "List a corporation's freelance jobs, using a logged-in character's roles to authorize.",
	}, listCorporationFreelanceJobsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_freelance_job_participants",
		Description: "List the participants in a corporation's freelance job, using a logged-in character's roles to authorize.",
	}, listCorporationFreelanceJobParticipantsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_freelance_job",
		Description: "Get the full details of a freelance job (public data, no character required).",
	}, getFreelanceJobHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_freelance_jobs",
		Description: "List freelance jobs game-wide, optionally filtered by corporation (public data, no character required).",
	}, listFreelanceJobsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_incursions",
		Description: "List every currently active incursion (public data, no character required).",
	}, listIncursionsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_industry_jobs",
		Description: "List a logged-in EVE Online character's industry jobs.",
	}, listCharacterIndustryJobsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_mining",
		Description: "List a logged-in EVE Online character's mining ledger.",
	}, listCharacterMiningHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_mining_extractions",
		Description: "List a corporation's moon mining extractions, using a logged-in character's roles to authorize.",
	}, listCorporationMiningExtractionsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_mining_observers",
		Description: "List a corporation's mining observers, using a logged-in character's roles to authorize.",
	}, listCorporationMiningObserversHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_mining_observer_detail",
		Description: "Get the mining ledger recorded by one of a corporation's mining observers, using a logged-in character's roles to authorize.",
	}, getCorporationMiningObserverDetailHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_industry_jobs",
		Description: "List a corporation's industry jobs, using a logged-in character's roles to authorize.",
	}, listCorporationIndustryJobsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_industry_facilities",
		Description: "List every industry facility in the game (public data, no character required).",
	}, listIndustryFacilitiesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_industry_systems",
		Description: "List industry cost indices for every solar system (public data, no character required).",
	}, listIndustrySystemsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_insurance_prices",
		Description: "List insurance price levels for every insurable ship type (public data, no character required).",
	}, listInsurancePricesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_killmails_recent",
		Description: "List a logged-in EVE Online character's recent killmail references.",
	}, listCharacterKillmailsRecentHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_killmails_recent",
		Description: "List a corporation's recent killmail references, using a logged-in character's roles to authorize.",
	}, listCorporationKillmailsRecentHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_killmail",
		Description: "Get the full details of a killmail, given its ID and hash (public data, no character required).",
	}, getKillmailHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_location",
		Description: "Get a logged-in EVE Online character's current solar system, and station or structure ID if docked/boarded.",
	}, getCharacterLocationHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_online",
		Description: "Get a logged-in EVE Online character's online status and login history.",
	}, getCharacterOnlineHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_ship",
		Description: "Get a logged-in EVE Online character's current ship.",
	}, getCharacterShipHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_loyalty_points",
		Description: "List a logged-in EVE Online character's loyalty point balances with every corporation they've worked for.",
	}, listCharacterLoyaltyPointsHandler(narrowESI[loyaltyESI](deps.OpenESI), store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_loyalty_store_offers",
		Description: "List a corporation's loyalty store offers (public data, no character required).",
	}, listLoyaltyStoreOffersHandler(version, narrowPublicESI[loyaltyStoreESI](deps.OpenPublicESI), sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_mail",
		Description: "List a logged-in EVE Online character's recent mail headers, optionally filtered by label and paged backwards via a mail ID cursor.",
	}, listCharacterMailHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_mail_labels",
		Description: "Get a logged-in EVE Online character's mail labels and unread counts.",
	}, getCharacterMailLabelsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_mail_lists",
		Description: "List the mailing lists a logged-in EVE Online character is subscribed to.",
	}, listCharacterMailListsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_mail",
		Description: "Get the full body and recipients of one of a logged-in EVE Online character's mails.",
	}, getCharacterMailHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "send_character_mail",
		Description: "Send a new mail as a logged-in EVE Online character. This changes state on EVE's server.",
	}, sendCharacterMailHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "create_character_mail_label",
		Description: "Create a new mail label for a logged-in EVE Online character. This changes state on EVE's server.",
	}, createCharacterMailLabelHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "update_character_mail",
		Description: "Update the read status and/or labels of one of a logged-in EVE Online character's mails. This changes state on EVE's server.",
	}, updateCharacterMailHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_character_mail",
		Description: "Delete one of a logged-in EVE Online character's mails. This changes state on EVE's server.",
	}, deleteCharacterMailHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_character_mail_label",
		Description: "Delete one of a logged-in EVE Online character's mail labels. This changes state on EVE's server.",
	}, deleteCharacterMailLabelHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_market_orders",
		Description: "List a logged-in EVE Online character's open market orders.",
	}, listCharacterMarketOrdersHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_market_order_history",
		Description: "List a logged-in EVE Online character's cancelled/expired market orders from the last 90 days.",
	}, listCharacterMarketOrderHistoryHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_market_orders",
		Description: "List a corporation's open market orders, using a logged-in character's roles to authorize.",
	}, listCorporationMarketOrdersHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_market_order_history",
		Description: "List a corporation's cancelled/expired market orders from the last 90 days, using a logged-in character's roles to authorize.",
	}, listCorporationMarketOrderHistoryHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_groups",
		Description: "List every market group ID in the game (public data, no character required).",
	}, listMarketGroupsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_market_group",
		Description: "Get detail for one market group, including its item types (public data, no character required).",
	}, getMarketGroupHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_prices",
		Description: "List ESI's adjusted/average price for every priced item type in the game (public data, no character required).",
	}, listMarketPricesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_market_history",
		Description: "Get daily historical market statistics for one item type in one region (public data, no character required).",
	}, getMarketHistoryHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_orders",
		Description: "List live market orders in a region, optionally filtered to one item type (public data, no character required).",
	}, listMarketOrdersHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_types",
		Description: "List the item type IDs with at least one active market order in a region (public data, no character required).",
	}, listMarketTypesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_structure_market_orders",
		Description: "List live market orders in a player structure, using a logged-in character's docking access to authorize.",
	}, listStructureMarketOrdersHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_esi_changelog",
		Description: "Get ESI's own changelog, keyed by date (public data, no character required).",
	}, getEsiChangelogHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_esi_compatibility_dates",
		Description: "List every compatibility date ESI currently accepts (public data, no character required).",
	}, listEsiCompatibilityDatesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_esi_name",
		Description: "Get ESI's current and historical names (public data, no character required).",
	}, getEsiNameHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_esi_status",
		Description: "Get the health status of every ESI route (public data, no character required).",
	}, getEsiStatusHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_planets",
		Description: "List a logged-in EVE Online character's colonized planets (Planetary Interaction).",
	}, listCharacterPlanetsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_planet",
		Description: "Get the full colony layout (links, pins, routes) of one of a character's planets.",
	}, getCharacterPlanetHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_customs_offices",
		Description: "List a corporation's customs offices (POCOs), including tax rates and reinforcement schedule.",
	}, listCorporationCustomsOfficesHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_planetary_schematic",
		Description: "Get a Planetary Interaction schematic's cycle time and name.",
	}, getPlanetarySchematicHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_route",
		Description: "Compute a route between two solar systems, optionally avoiding systems or adding extra connections (public data, no character required).",
	}, getRouteHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "search_character",
		Description: "Search for entities (characters, corporations, systems, etc.) matching a string, scoped to a logged-in character's search permissions.",
	}, searchCharacterHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_attributes",
		Description: "Get a logged-in EVE Online character's neural attributes (Charisma, Intelligence, Memory, Perception, Willpower) and remap history.",
	}, getCharacterAttributesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_skill_queue",
		Description: "List a logged-in EVE Online character's current skill training queue.",
	}, listCharacterSkillQueueHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_skills",
		Description: "List a logged-in EVE Online character's trained skills, total skill points, and unallocated skill points.",
	}, listCharacterSkillsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_sovereignty_campaigns",
		Description: "List every active sovereignty campaign in the game (public data, no character required).",
	}, listSovereigntyCampaignsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_sovereignty_systems",
		Description: "List every solar system and its current sovereignty claim, if any (public data, no character required).",
	}, listSovereigntySystemsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_status",
		Description: "Get the current EVE Online server status: player count, version, start time, and VIP mode (public data, no character required).",
	}, getStatusHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_mercenary_dens",
		Description: "List the given character's mercenary dens (includes resolved planet name).",
	}, listCharacterMercenaryDensHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_mercenary_den",
		Description: "Get full detail on one of the given character's mercenary dens: evolution stats, infomorph count, reinforcement timer, and attached skyhook.",
	}, getCharacterMercenaryDenHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_skyhooks",
		Description: "List the given corporation's skyhooks (includes resolved planet name).",
	}, listCorporationSkyhooksHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_skyhook",
		Description: "Get full detail on one of the given corporation's skyhooks: workforce, reagent stock, active/reinforcement state, and theft vulnerability window.",
	}, getCorporationSkyhookHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_sovereignty_hubs",
		Description: "List the given corporation's sovereignty hubs (includes resolved solar system name).",
	}, listCorporationSovereigntyHubsHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_sovereignty_hub",
		Description: "Get full detail on one of the given corporation's sovereignty hubs: reagent bay, power/workforce resources, upgrades, vulnerability window, and workforce transport configuration/state.",
	}, getCorporationSovereigntyHubHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_ancestries",
		Description: "List every character ancestry (public data, no character required).",
	}, listUniverseAncestriesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_bloodlines",
		Description: "List every character bloodline (public data, no character required).",
	}, listUniverseBloodlinesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_races",
		Description: "List every character race (public data, no character required).",
	}, listUniverseRacesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_factions",
		Description: "List every NPC faction (public data, no character required).",
	}, listUniverseFactionsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_categories",
		Description: "List every item category ID (public data, no character required).",
	}, listUniverseCategoriesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_category",
		Description: "Get an item category's name, published state, and member group IDs (public data, no character required).",
	}, getUniverseCategoryHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_groups",
		Description: "List every item group ID, paginated (public data, no character required).",
	}, listUniverseGroupsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_group",
		Description: "Get an item group's name, category, published state, and member type IDs (public data, no character required).",
	}, getUniverseGroupHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_types",
		Description: "List every item type ID, paginated (public data, no character required).",
	}, listUniverseTypesHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_type",
		Description: "Get an item type's full detail: name, group, dogma attributes/effects, and physical stats (public data, no character required).",
	}, getUniverseTypeHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_regions",
		Description: "List every region ID (public data, no character required).",
	}, listUniverseRegionsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_region",
		Description: "Get a region's name, description, and member constellation IDs (public data, no character required).",
	}, getUniverseRegionHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_constellations",
		Description: "List every constellation ID (public data, no character required).",
	}, listUniverseConstellationsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_constellation",
		Description: "Get a constellation's name, region, position, and member system IDs (public data, no character required).",
	}, getUniverseConstellationHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_systems",
		Description: "List every solar system ID (public data, no character required).",
	}, listUniverseSystemsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_system",
		Description: "Get a solar system's name, constellation, position, security status/class, star, stargates, stations, and planets (public data, no character required).",
	}, getUniverseSystemHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_asteroid_belt",
		Description: "Get an asteroid belt's name, position, and solar system (public data, no character required).",
	}, getUniverseAsteroidBeltHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_moon",
		Description: "Get a moon's name, position, and solar system (public data, no character required).",
	}, getUniverseMoonHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_planet",
		Description: "Get a planet's name, type, position, and solar system (public data, no character required).",
	}, getUniversePlanetHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_stargate",
		Description: "Get a stargate's name, type, position, solar system, and destination (public data, no character required).",
	}, getUniverseStargateHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_star",
		Description: "Get a star's age, luminosity, name, radius, solar system, spectral class, temperature, and type (public data, no character required).",
	}, getUniverseStarHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_station",
		Description: "Get a station's name, type, position, solar system, owner, services, and docking/reprocessing stats (public data, no character required).",
	}, getUniverseStationHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_graphics",
		Description: "List every graphic ID (public data, no character required).",
	}, listUniverseGraphicsHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_graphic",
		Description: "Get a graphic's collision/graphic files, icon folder, and Space Object Factory (SOF) DNA/hull/race/faction names (public data, no character required).",
	}, getUniverseGraphicHandler(version, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_structures",
		Description: "List every publicly-listed player structure ID (public data, no character required).",
	}, listUniverseStructuresHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_universe_structure",
		Description: "Get a player structure's name, owner, solar system, type, and position. The given character must have docking access to the structure.",
	}, getUniverseStructureHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_system_jumps",
		Description: "List ship jump activity per solar system over the last hour (public data, no character required).",
	}, listUniverseSystemJumpsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_universe_system_kills",
		Description: "List NPC/pod/ship kill activity per solar system over the last hour (public data, no character required).",
	}, listUniverseSystemKillsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "resolve_universe_ids",
		Description: "Resolve a list of names to IDs, grouped by category: agents, alliances, characters, constellations, corporations, factions, inventory types, regions, stations, and systems. Names ESI can't match are simply omitted from the response, not reported as errors. Accepts 1-500 names per call (public data, no character required).",
	}, resolveUniverseIDsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "resolve_universe_names",
		Description: "Resolve a list of IDs to their names and categories. Accepts 1-1000 IDs per call (public data, no character required).",
	}, resolveUniverseNamesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "set_autopilot_waypoint",
		Description: "Set an autopilot waypoint on the given character's live EVE client — this changes the character's active autopilot route.",
	}, setAutopilotWaypointHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "open_contract_window",
		Description: "Open a contract window on the given character's live EVE client — this changes what's displayed on the character's screen.",
	}, openContractWindowHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "open_information_window",
		Description: "Open an information window on the given character's live EVE client — this changes what's displayed on the character's screen.",
	}, openInformationWindowHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "open_market_details_window",
		Description: "Open a market details window on the given character's live EVE client — this changes what's displayed on the character's screen.",
	}, openMarketDetailsWindowHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "open_new_mail_window",
		Description: "Open a pre-filled new mail window on the given character's live EVE client — this changes what's displayed on the character's screen.",
	}, openNewMailWindowHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_wars",
		Description: "List war IDs, optionally only those older than a given war ID (public data, no character required).",
	}, listWarsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_war",
		Description: "Get a war's aggressor/defender/allies, declared/started/retracted/finished times, and mutual/open-for-allies flags (public data, no character required).",
	}, getWarHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_war_killmails",
		Description: "List killmail references for a war, paginated (public data, no character required).",
	}, listWarKillmailsHandler(version))

	// Military Campaigns (37th ESI tag; unblocked by goesi-openapi v0.0.42).

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_military_campaigns",
		Description: "List EVE Online military campaigns with their state, progress, and start/finish times (public data, no character required).",
	}, listMilitaryCampaignsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_military_campaign",
		Description: "Get one military campaign's state, progress, and start/finish times (public data, no character required).",
	}, getMilitaryCampaignHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_military_campaign_objectives",
		Description: "List a military campaign's objectives with progress and participant counts, cursor-paginated (public data, no character required).",
	}, listMilitaryCampaignObjectivesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_military_campaign_objective",
		Description: "Get one military campaign objective's state, progress, and participant counts (public data, no character required).",
	}, getMilitaryCampaignObjectiveHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_military_campaign_objectives",
		Description: "List the military campaign objectives a logged-in character has participated in, with contribution and commitment, cursor-paginated.",
	}, listCharacterMilitaryCampaignObjectivesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_military_campaign_objective_participation",
		Description: "Get a logged-in character's contribution to and commitment status in one military campaign objective.",
	}, getCharacterMilitaryCampaignObjectiveParticipationHandler(version, clientID, store))

	// Cosmetics.

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_skinr_licenses",
		Description: "List the SKINR (ship skin) licenses a logged-in character owns, with activated/unactivated counts.",
	}, listCharacterSkinrLicensesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_skinr_component_licenses",
		Description: "List the SKINR component licenses (nanocoatings, patterns) a logged-in character owns, with names and remaining runs.",
	}, listCharacterSkinrComponentLicensesHandler(version, clientID, store, sdeStore))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_skinr",
		Description: "Get a SKINR's full design: ship type, tier, and each cosmetic slot's nanocoating or pattern, with names (public data, no character required).",
	}, getSkinrHandler(version, sdeStore))

	// Paragon Hub.

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings",
		Description: "Browse the SKINR listings publicly available on the Paragon Hub, with ISK or PLEX prices, cursor-paginated (public data, no character required).",
	}, listParagonHubSkinrListingsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_paragon_hub_skinr_listings",
		Description: "List the SKINR listings a logged-in character has posted on the Paragon Hub, including who each targets, cursor-paginated.",
	}, listCharacterParagonHubSkinrListingsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings_for_alliance",
		Description: "Browse the Paragon Hub SKINR listings visible to an alliance, using a logged-in character to authorize, cursor-paginated.",
	}, listParagonHubSkinrListingsForAllianceHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings_for_corporation",
		Description: "Browse the Paragon Hub SKINR listings visible to a corporation, using a logged-in character to authorize, cursor-paginated.",
	}, listParagonHubSkinrListingsForCorporationHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings_for_character",
		Description: "Browse the Paragon Hub SKINR listings visible to a logged-in character (listings other sellers targeted at it), cursor-paginated.",
	}, listParagonHubSkinrListingsForCharacterHandler(version, clientID, store))

	registerNavigationTools(server, version, clientID, store, sdeStore)
	registerStructureTools(server, version, clientID, store, sdeStore)
	registerMarketPlanningTools(server, version, clientID, store, sdeStore)
	registerIndustryPlanningTools(server, version, clientID, store, sdeStore)

	return server
}
