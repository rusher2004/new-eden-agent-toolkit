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

// NewServer builds neat's MCP server, or returns an error naming the first
// missing dependency in deps. It registers only the tools not yet moved to
// their own Add functions: add_character, list_scope_groups, list_characters,
// update_character_details, remove_character, the asset-location tools
// (get_asset_locations, get_corporation_asset_locations), the contract tools,
// the market order and market group tools, get_market_history and
// list_market_types. Every other tool is registered by commands/mcp_tools*.go.
func NewServer(version string, deps Deps) (*mcpsdk.Server, error) {
	switch {
	case deps.Config == nil:
		return nil, errors.New("new server: Deps.Config is required")
	// Catches only a literal nil interface. A nil *sde.Store from the
	// composition root is a non-nil interface and passes: its methods are
	// nil-safe and return no data.
	case deps.SDE == nil:
		return nil, errors.New("new server: Deps.SDE is required")
	}
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
		Name:        "get_asset_locations",
		Description: "Resolve a set of a character's asset item IDs into their positions.",
	}, getAssetLocationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_asset_locations",
		Description: "Resolve a set of a corporation's asset item IDs into their positions.",
	}, getCorporationAssetLocationsHandler(version, clientID, store))

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
		Name:        "list_public_contract_bids",
		Description: "List the bids on a public contract (public data, no character required).",
	}, listPublicContractBidsHandler(version))

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
		Name:        "get_market_history",
		Description: "Get daily historical market statistics for one item type in one region (public data, no character required).",
	}, getMarketHistoryHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_market_types",
		Description: "List the item type IDs with at least one active market order in a region (public data, no character required).",
	}, listMarketTypesHandler(version, sdeStore))

	return server, nil
}
