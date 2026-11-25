//go:generate go run ../tools/gentooldocs -out ../docs/reference/tools.md

// Package mcp holds neat's MCP tools. A converted tool file declares the small
// interfaces it needs, a handler constructor that takes them, and an exported
// Add function that registers the tool; the composition root (commands/)
// builds the concrete config store, SDE cache and ESI clients and passes them
// to each Add function, so nothing here constructs one. See loyalty.go.
package mcp

import (
	"context"
	"errors"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// Deps is what NewServer hands to the tools not yet converted to the pattern in
// loyalty.go. It is transitional: ConfigStore and SDEStore are unions of what
// those files need, and each file that moves to its own small interfaces and
// an exported Add function (registered from commands/mcp_tools.go) leaves them.
// They are deleted, with Deps, when the last file moves.
type Deps struct {
	// ClientID is the ESI OAuth2 client ID, used by the login flow.
	ClientID string
	Config   ConfigStore
	SDE      SDEStore
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

// NewServer builds neat's MCP server, or returns an error naming the first missing
// dependency in deps, with add_character, list_scope_groups, list_characters,
// remove_character, list_assets, get_asset_locations, get_asset_names,
// list_corporation_assets, get_corporation_asset_locations, get_corporation_asset_names,
// get_character, get_character_affiliations, list_character_corporation_history,
// get_character_portrait, list_character_agents_research, list_character_blueprints,
// get_character_fatigue, list_character_medals, list_character_notifications,
// list_character_notification_contacts, get_character_roles, list_character_standings,
// list_character_titles, get_character_cspa_cost, get_character_clones,
// list_character_implants, list_character_contacts, list_character_contact_labels,
// list_corporation_contacts, list_corporation_contact_labels, list_alliance_contacts,
// list_alliance_contact_labels, add_character_contacts, update_character_contacts,
// remove_character_contacts, list_character_contracts, list_character_contract_bids,
// list_character_contract_items, list_corporation_contracts,
// list_corporation_contract_bids, list_corporation_contract_items, list_public_contracts,
// list_public_contract_bids, list_public_contract_items, get_corporation,
// list_corporation_alliance_history, get_corporation_icons, list_npc_corporations,
// list_corporation_members, get_corporation_members_limit, list_corporation_member_titles,
// list_corporation_member_tracking, list_corporation_roles,
// list_corporation_roles_history, list_corporation_titles, get_corporation_divisions,
// list_corporation_shareholders, list_corporation_standings, list_corporation_blueprints,
// list_corporation_container_logs, list_corporation_facilities, list_corporation_medals,
// list_corporation_medals_issued, list_corporation_starbases, get_corporation_starbase,
// list_corporation_structures, get_corporation_project_contribution,
// list_corporation_project_contributors, get_corporation_project_detail,
// list_corporation_projects, get_character_fw_stats, get_corporation_fw_stats,
// get_fw_leaderboards, get_fw_character_leaderboards, get_fw_corporation_leaderboards,
// list_fw_stats, list_fw_systems, list_fw_wars, list_incursions,
// list_character_industry_jobs, list_character_mining,
// list_corporation_mining_extractions, list_corporation_mining_observers,
// get_corporation_mining_observer_detail, list_corporation_industry_jobs,
// list_industry_facilities, list_industry_systems, get_character_location,
// get_character_online, get_character_ship, list_character_market_orders,
// list_character_market_order_history, list_corporation_market_orders,
// list_corporation_market_order_history, list_market_groups, get_market_group,
// list_market_prices, get_market_history, list_market_orders, list_market_types,
// list_structure_market_orders, get_character_attributes, list_character_skill_queue,
// list_character_skills, list_sovereignty_campaigns, list_sovereignty_systems,
// list_character_mercenary_dens, get_character_mercenary_den, list_corporation_skyhooks,
// get_corporation_skyhook, list_corporation_sovereignty_hubs,
// get_corporation_sovereignty_hub, list_universe_ancestries, list_universe_bloodlines,
// list_universe_races, list_universe_factions, list_universe_categories,
// get_universe_category, list_universe_groups, get_universe_group, list_universe_types,
// get_universe_type, list_universe_regions, get_universe_region,
// list_universe_constellations, get_universe_constellation, list_universe_systems,
// get_universe_system, get_universe_asteroid_belt, get_universe_moon, get_universe_planet,
// get_universe_stargate, get_universe_star, get_universe_station, list_universe_graphics,
// get_universe_graphic, list_universe_structures, get_universe_structure,
// list_universe_system_jumps, list_universe_system_kills, resolve_universe_ids, and
// resolve_universe_names registered.
func NewServer(version string, deps Deps) (*mcpsdk.Server, error) {
	switch {
	case deps.Config == nil:
		return nil, errors.New("new server: Deps.Config is required")
	case deps.SDE == nil:
		return nil, errors.New("new server: Deps.SDE is required")
	}
	clientID, store := deps.ClientID, deps.Config
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

	return server, nil
}
