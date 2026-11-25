package mcp

import (
	"context"
	"fmt"
	"log/slog"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// --- list_universe_ancestries (public, no input) ---

type UniverseAncestry struct {
	ID               int64   `json:"id"`
	BloodlineID      int64   `json:"bloodlineId"`
	Description      string  `json:"description"`
	Name             string  `json:"name"`
	IconID           *int64  `json:"iconId,omitempty"`
	ShortDescription *string `json:"shortDescription,omitempty"`
}

type ancestryCache interface {
	Ancestries(ctx context.Context, userAgent string) ([]sde.Ancestry, error)
}

// universeAncestriesESI is the ESI call list_universe_ancestries makes when the SDE cache is empty.
type universeAncestriesESI interface {
	UniverseAncestries(ctx context.Context) ([]esiapi.UniverseAncestriesGetInner, error)
}

func listUniverseAncestries(ctx context.Context, sdeStore ancestryCache, client universeAncestriesESI, userAgent string) ([]UniverseAncestry, error) {
	ancestries, err := sdeStore.Ancestries(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ancestries) == 0 {
		fetched, err := client.UniverseAncestries(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseAncestry, len(fetched))
		for i, a := range fetched {
			out[i] = UniverseAncestry{ID: a.Id, BloodlineID: a.BloodlineId, Description: a.Description, Name: a.Name, IconID: a.IconId, ShortDescription: a.ShortDescription}
		}
		return out, nil
	}
	out := make([]UniverseAncestry, len(ancestries))
	for i, a := range ancestries {
		out[i] = UniverseAncestry{ID: a.ID, BloodlineID: a.BloodlineID, Description: a.Description, Name: a.Name, IconID: a.IconID, ShortDescription: a.ShortDescription}
	}
	return out, nil
}

type ListUniverseAncestriesInput struct{}
type ListUniverseAncestriesOutput struct {
	Ancestries []UniverseAncestry `json:"ancestries"`
}

// AddListUniverseAncestries registers list_universe_ancestries. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseAncestries[C universeAncestriesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore ancestryCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_ancestries",
		Description: "List every character ancestry (public data, no character required).",
	}, listUniverseAncestriesHandler(userAgent, open, sdeStore))
}

func listUniverseAncestriesHandler[C universeAncestriesESI](userAgent string, open func() C, sdeStore ancestryCache) mcpsdk.ToolHandlerFor[ListUniverseAncestriesInput, ListUniverseAncestriesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseAncestriesInput) (*mcpsdk.CallToolResult, ListUniverseAncestriesOutput, error) {
		ancestries, err := listUniverseAncestries(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseAncestriesOutput{}, fmt.Errorf("list universe ancestries: %w", err)
		}
		return nil, ListUniverseAncestriesOutput{Ancestries: ancestries}, nil
	}
}

// --- list_universe_bloodlines (public, no input) ---

type UniverseBloodline struct {
	ID            int64  `json:"id"`
	Charisma      int64  `json:"charisma"`
	CorporationID int64  `json:"corporationId"`
	Description   string `json:"description"`
	Intelligence  int64  `json:"intelligence"`
	Memory        int64  `json:"memory"`
	Name          string `json:"name"`
	Perception    int64  `json:"perception"`
	RaceID        int64  `json:"raceId"`
	ShipTypeID    int64  `json:"shipTypeId"`
	Willpower     int64  `json:"willpower"`
}

type bloodlineCache interface {
	Bloodlines(ctx context.Context, userAgent string) ([]sde.Bloodline, error)
}

// universeBloodlinesESI is the ESI call list_universe_bloodlines makes when the SDE cache is empty.
type universeBloodlinesESI interface {
	UniverseBloodlines(ctx context.Context) ([]esiapi.UniverseBloodlinesGetInner, error)
}

func listUniverseBloodlines(ctx context.Context, sdeStore bloodlineCache, client universeBloodlinesESI, userAgent string) ([]UniverseBloodline, error) {
	bloodlines, err := sdeStore.Bloodlines(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(bloodlines) == 0 {
		fetched, err := client.UniverseBloodlines(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseBloodline, len(fetched))
		for i, b := range fetched {
			out[i] = UniverseBloodline{
				ID: b.BloodlineId, Charisma: b.Charisma, CorporationID: b.CorporationId, Description: b.Description,
				Intelligence: b.Intelligence, Memory: b.Memory, Name: b.Name, Perception: b.Perception,
				RaceID: b.RaceId, ShipTypeID: b.ShipTypeId, Willpower: b.Willpower,
			}
		}
		return out, nil
	}
	out := make([]UniverseBloodline, len(bloodlines))
	for i, b := range bloodlines {
		var shipTypeID int64
		if b.ShipTypeID != nil {
			shipTypeID = *b.ShipTypeID
		}
		out[i] = UniverseBloodline{
			ID: b.ID, Charisma: b.Charisma, CorporationID: b.CorporationID, Description: b.Description,
			Intelligence: b.Intelligence, Memory: b.Memory, Name: b.Name, Perception: b.Perception,
			RaceID: b.RaceID, ShipTypeID: shipTypeID, Willpower: b.Willpower,
		}
	}
	return out, nil
}

type ListUniverseBloodlinesInput struct{}
type ListUniverseBloodlinesOutput struct {
	Bloodlines []UniverseBloodline `json:"bloodlines"`
}

// AddListUniverseBloodlines registers list_universe_bloodlines. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseBloodlines[C universeBloodlinesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore bloodlineCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_bloodlines",
		Description: "List every character bloodline (public data, no character required).",
	}, listUniverseBloodlinesHandler(userAgent, open, sdeStore))
}

func listUniverseBloodlinesHandler[C universeBloodlinesESI](userAgent string, open func() C, sdeStore bloodlineCache) mcpsdk.ToolHandlerFor[ListUniverseBloodlinesInput, ListUniverseBloodlinesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseBloodlinesInput) (*mcpsdk.CallToolResult, ListUniverseBloodlinesOutput, error) {
		bloodlines, err := listUniverseBloodlines(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseBloodlinesOutput{}, fmt.Errorf("list universe bloodlines: %w", err)
		}
		return nil, ListUniverseBloodlinesOutput{Bloodlines: bloodlines}, nil
	}
}

// --- list_universe_races (public, no input) ---

type UniverseRace struct {
	RaceID      int64  `json:"raceId"`
	AllianceID  *int64 `json:"allianceId,omitempty"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

type raceCache interface {
	Races(ctx context.Context, userAgent string) ([]sde.Race, error)
}

// universeRacesESI is the ESI call list_universe_races makes when the SDE cache is empty.
type universeRacesESI interface {
	UniverseRaces(ctx context.Context) ([]esiapi.UniverseRacesGetInner, error)
}

func listUniverseRaces(ctx context.Context, sdeStore raceCache, client universeRacesESI, userAgent string) ([]UniverseRace, error) {
	races, err := sdeStore.Races(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(races) == 0 {
		fetched, err := client.UniverseRaces(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseRace, len(fetched))
		for i, r := range fetched {
			// ESI's own schema documents alliance_id as always-present for a
			// race (a plain int64, never omitted), so the cache-miss path
			// always has a real value here.
			allianceID := r.AllianceId
			out[i] = UniverseRace{RaceID: r.RaceId, AllianceID: &allianceID, Description: r.Description, Name: r.Name}
		}
		return out, nil
	}
	out := make([]UniverseRace, len(races))
	for i, r := range races {
		out[i] = UniverseRace{RaceID: r.ID, AllianceID: r.AllianceID, Description: r.Description, Name: r.Name}
	}
	return out, nil
}

type ListUniverseRacesInput struct{}
type ListUniverseRacesOutput struct {
	Races []UniverseRace `json:"races"`
}

// AddListUniverseRaces registers list_universe_races. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseRaces[C universeRacesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore raceCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_races",
		Description: "List every character race (public data, no character required).",
	}, listUniverseRacesHandler(userAgent, open, sdeStore))
}

func listUniverseRacesHandler[C universeRacesESI](userAgent string, open func() C, sdeStore raceCache) mcpsdk.ToolHandlerFor[ListUniverseRacesInput, ListUniverseRacesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseRacesInput) (*mcpsdk.CallToolResult, ListUniverseRacesOutput, error) {
		races, err := listUniverseRaces(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseRacesOutput{}, fmt.Errorf("list universe races: %w", err)
		}
		return nil, ListUniverseRacesOutput{Races: races}, nil
	}
}

// --- list_universe_factions (public, no input) ---

type UniverseFaction struct {
	FactionID            int64   `json:"factionId"`
	Description          string  `json:"description"`
	Name                 string  `json:"name"`
	IsUnique             *bool   `json:"isUnique,omitempty"`
	SizeFactor           float64 `json:"sizeFactor"`
	StationCount         int64   `json:"stationCount"`
	StationSystemCount   int64   `json:"stationSystemCount"`
	CorporationID        *int64  `json:"corporationId,omitempty"`
	MilitiaCorporationID *int64  `json:"militiaCorporationId,omitempty"`
	SolarSystemID        *int64  `json:"solarSystemId,omitempty"`
}

type factionCache interface {
	Factions(ctx context.Context, userAgent string) ([]sde.Faction, error)
}

// universeFactionsESI is the ESI call list_universe_factions makes when the SDE cache is empty.
type universeFactionsESI interface {
	UniverseFactions(ctx context.Context) ([]esiapi.UniverseFactionsGetInner, error)
}

func listUniverseFactions(ctx context.Context, sdeStore factionCache, client universeFactionsESI, userAgent string) ([]UniverseFaction, error) {
	factions, err := sdeStore.Factions(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(factions) == 0 {
		fetched, err := client.UniverseFactions(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]UniverseFaction, len(fetched))
		for i, f := range fetched {
			// ESI's UniverseFactions response always carries a real
			// isUnique value (goesi-openapi's generated model has it as a
			// plain, always-present bool) -- unlike a sqlite-served
			// Faction, this branch genuinely knows it.
			out[i] = UniverseFaction{
				FactionID: f.FactionId, Description: f.Description, Name: f.Name, IsUnique: &fetched[i].IsUnique,
				SizeFactor: f.SizeFactor, StationCount: f.StationCount, StationSystemCount: f.StationSystemCount,
				CorporationID: f.CorporationId, MilitiaCorporationID: f.MilitiaCorporationId, SolarSystemID: f.SolarSystemId,
			}
		}
		return out, nil
	}
	out := make([]UniverseFaction, len(factions))
	for i, f := range factions {
		out[i] = UniverseFaction{
			FactionID: f.ID, Description: f.Description, Name: f.Name, IsUnique: f.IsUnique,
			SizeFactor: f.SizeFactor, StationCount: f.StationCount, StationSystemCount: f.StationSystemCount,
			CorporationID: f.CorporationID, MilitiaCorporationID: f.MilitiaCorporationID, SolarSystemID: f.SolarSystemID,
		}
	}
	return out, nil
}

type ListUniverseFactionsInput struct{}
type ListUniverseFactionsOutput struct {
	Factions []UniverseFaction `json:"factions"`
}

// AddListUniverseFactions registers list_universe_factions. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseFactions[C universeFactionsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore factionCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_factions",
		Description: "List every NPC faction (public data, no character required).",
	}, listUniverseFactionsHandler(userAgent, open, sdeStore))
}

func listUniverseFactionsHandler[C universeFactionsESI](userAgent string, open func() C, sdeStore factionCache) mcpsdk.ToolHandlerFor[ListUniverseFactionsInput, ListUniverseFactionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseFactionsInput) (*mcpsdk.CallToolResult, ListUniverseFactionsOutput, error) {
		factions, err := listUniverseFactions(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseFactionsOutput{}, fmt.Errorf("list universe factions: %w", err)
		}
		return nil, ListUniverseFactionsOutput{Factions: factions}, nil
	}
}

// universeListPageSize is ESI's own page size for its large universe list
// endpoints -- verified live against /universe/groups/?page=1 (x-pages: 2
// for ~1700 groups, 1000 ids on page 1). A cache hit serves every ID
// unpaginated from sde/, so this constant lets the read-through layer
// replicate ESI's own pagination instead of changing the tool's contract.
const universeListPageSize = 1000

// paginateIDs slices a full, cache-hit ID list the same way ESI's own
// paginated list endpoints would. page <= 0 means page 1, matching every
// migrated list handler's existing default.
func paginateIDs(ids []int64, page int32) ([]int64, int32) {
	if page <= 0 {
		page = 1
	}
	totalPages := int32((len(ids) + universeListPageSize - 1) / universeListPageSize)
	if totalPages == 0 {
		totalPages = 1
	}
	start := int(page-1) * universeListPageSize
	if start >= len(ids) {
		return []int64{}, totalPages
	}
	end := min(start+universeListPageSize, len(ids))
	return ids[start:end], totalPages
}

// --- list_universe_categories (public, no input) ---

// categoryCache is the subset of the SDE store's category methods the
// read-through cache needs -- narrow enough that tests can substitute a
// fake without a real (network-touching) store. The SDE store satisfies
// this structurally; no sde/ changes are needed to pass one in.
type categoryCache interface {
	Category(ctx context.Context, userAgent string, id int64) (sde.Category, bool, error)
	PutCategory(sde.Category) error
	CategoryIDs(ctx context.Context, userAgent string) ([]int64, error)
}

// universeCategoriesESI is the ESI calls list_universe_categories and get_universe_category make on an SDE cache miss.
type universeCategoriesESI interface {
	UniverseCategories(ctx context.Context) ([]int64, error)
	UniverseCategoryDetail(ctx context.Context, categoryID int64) (*esiapi.UniverseCategoriesCategoryIdGet, error)
}

func listUniverseCategories(ctx context.Context, sdeStore categoryCache, client universeCategoriesESI, userAgent string) ([]int64, error) {
	ids, err := sdeStore.CategoryIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	// Cache empty (never synced) -- fall back to a live ESI list call.
	return client.UniverseCategories(ctx)
}

type ListUniverseCategoriesInput struct{}
type ListUniverseCategoriesOutput struct {
	IDs []int64 `json:"ids"`
}

// AddListUniverseCategories registers list_universe_categories. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseCategories[C universeCategoriesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore categoryCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_categories",
		Description: "List every item category ID (public data, no character required).",
	}, listUniverseCategoriesHandler(userAgent, open, sdeStore))
}

func listUniverseCategoriesHandler[C universeCategoriesESI](userAgent string, open func() C, sdeStore categoryCache) mcpsdk.ToolHandlerFor[ListUniverseCategoriesInput, ListUniverseCategoriesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseCategoriesInput) (*mcpsdk.CallToolResult, ListUniverseCategoriesOutput, error) {
		ids, err := listUniverseCategories(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseCategoriesOutput{}, fmt.Errorf("list universe categories: %w", err)
		}
		return nil, ListUniverseCategoriesOutput{IDs: ids}, nil
	}
}

// --- get_universe_category (public) ---

type UniverseCategory struct {
	CategoryID int64   `json:"categoryId"`
	Name       string  `json:"name"`
	Published  bool    `json:"published"`
	GroupIDs   []int64 `json:"groupIds"`
}

func getUniverseCategory(ctx context.Context, sdeStore categoryCache, client universeCategoriesESI, userAgent string, categoryID int64) (UniverseCategory, error) {
	c, ok, err := sdeStore.Category(ctx, userAgent, categoryID)
	if err != nil {
		return UniverseCategory{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		// Cache miss (never synced, or a genuinely new ID not yet in the
		// snapshot) -- read through to ESI and write the result back so
		// subsequent lookups of this ID are fast.
		detail, err := client.UniverseCategoryDetail(ctx, categoryID)
		if err != nil {
			return UniverseCategory{}, err
		}
		c = sde.Category{ID: detail.CategoryId, Name: detail.Name, Published: detail.Published, GroupIDs: detail.Groups}
		if err := sdeStore.PutCategory(c); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through category into cache", "categoryId", c.ID, "error", err)
		}
	}
	return UniverseCategory{CategoryID: c.ID, Name: c.Name, Published: c.Published, GroupIDs: c.GroupIDs}, nil
}

type GetUniverseCategoryInput struct {
	CategoryID int64 `json:"categoryId" jsonschema:"the category ID to look up"`
}

// AddGetUniverseCategory registers get_universe_category. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseCategory[C universeCategoriesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore categoryCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_category",
		Description: "Get an item category's name, published state, and member group IDs (public data, no character required).",
	}, getUniverseCategoryHandler(userAgent, open, sdeStore))
}

func getUniverseCategoryHandler[C universeCategoriesESI](userAgent string, open func() C, sdeStore categoryCache) mcpsdk.ToolHandlerFor[GetUniverseCategoryInput, UniverseCategory] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseCategoryInput) (*mcpsdk.CallToolResult, UniverseCategory, error) {
		out, err := getUniverseCategory(ctx, sdeStore, open(), userAgent, in.CategoryID)
		if err != nil {
			return nil, UniverseCategory{}, fmt.Errorf("get universe category: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_groups (public, paginated) ---

type groupCache interface {
	Group(ctx context.Context, userAgent string, id int64) (sde.Group, bool, error)
	PutGroup(sde.Group) error
	GroupIDs(ctx context.Context, userAgent string) ([]int64, error)
}

// universeGroupsESI is the ESI calls list_universe_groups and get_universe_group make on an SDE cache miss.
type universeGroupsESI interface {
	UniverseGroups(ctx context.Context, page int32) ([]int64, int32, error)
	UniverseGroupDetail(ctx context.Context, groupID int64) (*esiapi.UniverseGroupsGroupIdGet, error)
}

func listUniverseGroups(ctx context.Context, sdeStore groupCache, client universeGroupsESI, userAgent string, page int32) ([]int64, int32, error) {
	ids, err := sdeStore.GroupIDs(ctx, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) == 0 {
		return client.UniverseGroups(ctx, page)
	}
	pageIDs, totalPages := paginateIDs(ids, page)
	return pageIDs, totalPages, nil
}

type ListUniverseGroupsInput struct {
	Page int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListUniverseGroupsOutput struct {
	IDs        []int64 `json:"ids"`
	Page       int32   `json:"page"`
	TotalPages int32   `json:"totalPages"`
}

// AddListUniverseGroups registers list_universe_groups. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseGroups[C universeGroupsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore groupCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_groups",
		Description: "List every item group ID, paginated (public data, no character required).",
	}, listUniverseGroupsHandler(userAgent, open, sdeStore))
}

func listUniverseGroupsHandler[C universeGroupsESI](userAgent string, open func() C, sdeStore groupCache) mcpsdk.ToolHandlerFor[ListUniverseGroupsInput, ListUniverseGroupsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseGroupsInput) (*mcpsdk.CallToolResult, ListUniverseGroupsOutput, error) {
		ids, totalPages, err := listUniverseGroups(ctx, sdeStore, open(), userAgent, in.Page)
		if err != nil {
			return nil, ListUniverseGroupsOutput{}, fmt.Errorf("list universe groups: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListUniverseGroupsOutput{IDs: ids, Page: page, TotalPages: totalPages}, nil
	}
}

// --- get_universe_group (public) ---

type UniverseGroup struct {
	GroupID    int64   `json:"groupId"`
	CategoryID int64   `json:"categoryId"`
	Name       string  `json:"name"`
	Published  bool    `json:"published"`
	TypeIDs    []int64 `json:"typeIds"`
}

func getUniverseGroup(ctx context.Context, sdeStore groupCache, client universeGroupsESI, userAgent string, groupID int64) (UniverseGroup, error) {
	g, ok, err := sdeStore.Group(ctx, userAgent, groupID)
	if err != nil {
		return UniverseGroup{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseGroupDetail(ctx, groupID)
		if err != nil {
			return UniverseGroup{}, err
		}
		g = sde.Group{ID: detail.GroupId, CategoryID: detail.CategoryId, Name: detail.Name, Published: detail.Published, TypeIDs: detail.Types}
		if err := sdeStore.PutGroup(g); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through group into cache", "groupId", g.ID, "error", err)
		}
	}
	return UniverseGroup{GroupID: g.ID, CategoryID: g.CategoryID, Name: g.Name, Published: g.Published, TypeIDs: g.TypeIDs}, nil
}

type GetUniverseGroupInput struct {
	GroupID int64 `json:"groupId" jsonschema:"the group ID to look up"`
}

// AddGetUniverseGroup registers get_universe_group. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseGroup[C universeGroupsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore groupCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_group",
		Description: "Get an item group's name, category, published state, and member type IDs (public data, no character required).",
	}, getUniverseGroupHandler(userAgent, open, sdeStore))
}

func getUniverseGroupHandler[C universeGroupsESI](userAgent string, open func() C, sdeStore groupCache) mcpsdk.ToolHandlerFor[GetUniverseGroupInput, UniverseGroup] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseGroupInput) (*mcpsdk.CallToolResult, UniverseGroup, error) {
		out, err := getUniverseGroup(ctx, sdeStore, open(), userAgent, in.GroupID)
		if err != nil {
			return nil, UniverseGroup{}, fmt.Errorf("get universe group: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_types (public, paginated) ---

type typeCache interface {
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
	PutType(sde.Type) error
	TypeIDs(ctx context.Context, userAgent string) ([]int64, error)
}

// universeTypesESI is the ESI calls list_universe_types and get_universe_type make on an SDE cache miss.
type universeTypesESI interface {
	UniverseTypes(ctx context.Context, page int32) ([]int64, int32, error)
	UniverseTypeDetail(ctx context.Context, typeID int64) (*esiapi.UniverseTypesTypeIdGet, error)
}

func listUniverseTypes(ctx context.Context, sdeStore typeCache, client universeTypesESI, userAgent string, page int32) ([]int64, int32, error) {
	ids, err := sdeStore.TypeIDs(ctx, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) == 0 {
		return client.UniverseTypes(ctx, page)
	}
	pageIDs, totalPages := paginateIDs(ids, page)
	return pageIDs, totalPages, nil
}

type ListUniverseTypesInput struct {
	Page int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListUniverseTypesOutput struct {
	IDs        []int64 `json:"ids"`
	Page       int32   `json:"page"`
	TotalPages int32   `json:"totalPages"`
}

// AddListUniverseTypes registers list_universe_types. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseTypes[C universeTypesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore typeCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_types",
		Description: "List every item type ID, paginated (public data, no character required).",
	}, listUniverseTypesHandler(userAgent, open, sdeStore))
}

func listUniverseTypesHandler[C universeTypesESI](userAgent string, open func() C, sdeStore typeCache) mcpsdk.ToolHandlerFor[ListUniverseTypesInput, ListUniverseTypesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseTypesInput) (*mcpsdk.CallToolResult, ListUniverseTypesOutput, error) {
		ids, totalPages, err := listUniverseTypes(ctx, sdeStore, open(), userAgent, in.Page)
		if err != nil {
			return nil, ListUniverseTypesOutput{}, fmt.Errorf("list universe types: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListUniverseTypesOutput{IDs: ids, Page: page, TotalPages: totalPages}, nil
	}
}

// --- get_universe_type (public) ---

type UniverseType struct {
	TypeID          int64                  `json:"typeId"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	GroupID         int64                  `json:"groupId"`
	Published       bool                   `json:"published"`
	Capacity        *float64               `json:"capacity,omitempty"`
	GraphicID       *int64                 `json:"graphicId,omitempty"`
	IconID          *int64                 `json:"iconId,omitempty"`
	MarketGroupID   *int64                 `json:"marketGroupId,omitempty"`
	Mass            *float64               `json:"mass,omitempty"`
	PackagedVolume  *float64               `json:"packagedVolume,omitempty"`
	PortionSize     *int64                 `json:"portionSize,omitempty"`
	Radius          *float64               `json:"radius,omitempty"`
	Volume          *float64               `json:"volume,omitempty"`
	DogmaAttributes []DynamicItemAttribute `json:"dogmaAttributes,omitempty"`
	DogmaEffects    []DynamicItemEffect    `json:"dogmaEffects,omitempty"`
}

func getUniverseType(ctx context.Context, sdeStore typeCache, client universeTypesESI, userAgent string, typeID int64) (UniverseType, error) {
	t, ok, err := sdeStore.Type(ctx, userAgent, typeID)
	if err != nil {
		return UniverseType{}, fmt.Errorf("read sde cache: %w", err)
	}

	if !ok {
		detail, err := client.UniverseTypeDetail(ctx, typeID)
		if err != nil {
			return UniverseType{}, err
		}
		t = sde.Type{
			ID: detail.TypeId, GroupID: detail.GroupId, Name: detail.Name,
			Description: detail.Description, Published: detail.Published,
			Capacity: detail.Capacity, GraphicID: detail.GraphicId, IconID: detail.IconId, MarketGroupID: detail.MarketGroupId,
			Mass: detail.Mass, PackagedVolume: detail.PackagedVolume, PortionSize: detail.PortionSize, Radius: detail.Radius, Volume: detail.Volume,
		}
		if len(detail.DogmaAttributes) > 0 {
			t.DogmaAttributes = make([]sde.TypeDogmaAttribute, len(detail.DogmaAttributes))
			for i, a := range detail.DogmaAttributes {
				t.DogmaAttributes[i] = sde.TypeDogmaAttribute{AttributeID: a.AttributeId, Value: a.Value}
			}
		}
		if len(detail.DogmaEffects) > 0 {
			t.DogmaEffects = make([]sde.TypeDogmaEffect, len(detail.DogmaEffects))
			for i, e := range detail.DogmaEffects {
				t.DogmaEffects[i] = sde.TypeDogmaEffect{EffectID: e.EffectId, IsDefault: e.IsDefault}
			}
		}
		if err := sdeStore.PutType(t); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through type into cache", "typeId", t.ID, "error", err)
		}
	}

	out := UniverseType{
		TypeID: t.ID, Name: t.Name, Description: t.Description, GroupID: t.GroupID, Published: t.Published,
		Capacity: t.Capacity, GraphicID: t.GraphicID, IconID: t.IconID, MarketGroupID: t.MarketGroupID,
		Mass: t.Mass, PackagedVolume: t.PackagedVolume, PortionSize: t.PortionSize, Radius: t.Radius, Volume: t.Volume,
	}
	if len(t.DogmaAttributes) > 0 {
		out.DogmaAttributes = make([]DynamicItemAttribute, len(t.DogmaAttributes))
		for i, a := range t.DogmaAttributes {
			out.DogmaAttributes[i] = DynamicItemAttribute{AttributeID: a.AttributeID, Value: a.Value}
		}
	}
	if len(t.DogmaEffects) > 0 {
		out.DogmaEffects = make([]DynamicItemEffect, len(t.DogmaEffects))
		for i, e := range t.DogmaEffects {
			out.DogmaEffects[i] = DynamicItemEffect{EffectID: e.EffectID, IsDefault: e.IsDefault}
		}
	}
	return out, nil
}

type GetUniverseTypeInput struct {
	TypeID int64 `json:"typeId" jsonschema:"the type ID to look up"`
}

// AddGetUniverseType registers get_universe_type. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseType[C universeTypesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore typeCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_type",
		Description: "Get an item type's full detail: name, group, dogma attributes/effects, and physical stats (public data, no character required).",
	}, getUniverseTypeHandler(userAgent, open, sdeStore))
}

func getUniverseTypeHandler[C universeTypesESI](userAgent string, open func() C, sdeStore typeCache) mcpsdk.ToolHandlerFor[GetUniverseTypeInput, UniverseType] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseTypeInput) (*mcpsdk.CallToolResult, UniverseType, error) {
		out, err := getUniverseType(ctx, sdeStore, open(), userAgent, in.TypeID)
		if err != nil {
			return nil, UniverseType{}, fmt.Errorf("get universe type: %w", err)
		}
		return nil, out, nil
	}
}

// Position is a shared 3D coordinate, reused by every celestial/geography detail tool below.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// --- list_universe_regions (public, no input) ---

type regionCache interface {
	Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error)
	PutRegion(sde.Region) error
	RegionIDs(ctx context.Context, userAgent string) ([]int64, error)
}

// universeRegionsESI is the ESI calls list_universe_regions and get_universe_region make on an SDE cache miss.
type universeRegionsESI interface {
	UniverseRegions(ctx context.Context) ([]int64, error)
	UniverseRegionDetail(ctx context.Context, regionID int64) (*esiapi.UniverseRegionsRegionIdGet, error)
}

func listUniverseRegions(ctx context.Context, sdeStore regionCache, client universeRegionsESI, userAgent string) ([]int64, error) {
	ids, err := sdeStore.RegionIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	return client.UniverseRegions(ctx)
}

type ListUniverseRegionsInput struct{}
type ListUniverseRegionsOutput struct {
	IDs []int64 `json:"ids"`
}

// AddListUniverseRegions registers list_universe_regions. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseRegions[C universeRegionsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore regionCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_regions",
		Description: "List every region ID (public data, no character required).",
	}, listUniverseRegionsHandler(userAgent, open, sdeStore))
}

func listUniverseRegionsHandler[C universeRegionsESI](userAgent string, open func() C, sdeStore regionCache) mcpsdk.ToolHandlerFor[ListUniverseRegionsInput, ListUniverseRegionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseRegionsInput) (*mcpsdk.CallToolResult, ListUniverseRegionsOutput, error) {
		ids, err := listUniverseRegions(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseRegionsOutput{}, fmt.Errorf("list universe regions: %w", err)
		}
		return nil, ListUniverseRegionsOutput{IDs: ids}, nil
	}
}

// --- get_universe_region (public) ---

type UniverseRegion struct {
	RegionID         int64   `json:"regionId"`
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	ConstellationIDs []int64 `json:"constellationIds"`
}

func getUniverseRegion(ctx context.Context, sdeStore regionCache, client universeRegionsESI, userAgent string, regionID int64) (UniverseRegion, error) {
	r, ok, err := sdeStore.Region(ctx, userAgent, regionID)
	if err != nil {
		return UniverseRegion{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseRegionDetail(ctx, regionID)
		if err != nil {
			return UniverseRegion{}, err
		}
		r = sde.Region{ID: detail.RegionId, Name: detail.Name, Description: detail.Description, ConstellationIDs: detail.Constellations}
		if err := sdeStore.PutRegion(r); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through region into cache", "regionId", r.ID, "error", err)
		}
	}
	return UniverseRegion{RegionID: r.ID, Name: r.Name, Description: r.Description, ConstellationIDs: r.ConstellationIDs}, nil
}

type GetUniverseRegionInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
}

// AddGetUniverseRegion registers get_universe_region. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseRegion[C universeRegionsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore regionCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_region",
		Description: "Get a region's name, description, and member constellation IDs (public data, no character required).",
	}, getUniverseRegionHandler(userAgent, open, sdeStore))
}

func getUniverseRegionHandler[C universeRegionsESI](userAgent string, open func() C, sdeStore regionCache) mcpsdk.ToolHandlerFor[GetUniverseRegionInput, UniverseRegion] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseRegionInput) (*mcpsdk.CallToolResult, UniverseRegion, error) {
		out, err := getUniverseRegion(ctx, sdeStore, open(), userAgent, in.RegionID)
		if err != nil {
			return nil, UniverseRegion{}, fmt.Errorf("get universe region: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_constellations (public, no input) ---

type constellationCache interface {
	Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error)
	PutConstellation(sde.Constellation) error
	ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error)
}

// universeConstellationsESI is the ESI calls list_universe_constellations and get_universe_constellation make on an SDE cache miss.
type universeConstellationsESI interface {
	UniverseConstellations(ctx context.Context) ([]int64, error)
	UniverseConstellationDetail(ctx context.Context, constellationID int64) (*esiapi.UniverseConstellationsConstellationIdGet, error)
}

func listUniverseConstellations(ctx context.Context, sdeStore constellationCache, client universeConstellationsESI, userAgent string) ([]int64, error) {
	ids, err := sdeStore.ConstellationIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	return client.UniverseConstellations(ctx)
}

type ListUniverseConstellationsInput struct{}
type ListUniverseConstellationsOutput struct {
	IDs []int64 `json:"ids"`
}

// AddListUniverseConstellations registers list_universe_constellations. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseConstellations[C universeConstellationsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore constellationCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_constellations",
		Description: "List every constellation ID (public data, no character required).",
	}, listUniverseConstellationsHandler(userAgent, open, sdeStore))
}

func listUniverseConstellationsHandler[C universeConstellationsESI](userAgent string, open func() C, sdeStore constellationCache) mcpsdk.ToolHandlerFor[ListUniverseConstellationsInput, ListUniverseConstellationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseConstellationsInput) (*mcpsdk.CallToolResult, ListUniverseConstellationsOutput, error) {
		ids, err := listUniverseConstellations(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseConstellationsOutput{}, fmt.Errorf("list universe constellations: %w", err)
		}
		return nil, ListUniverseConstellationsOutput{IDs: ids}, nil
	}
}

// --- get_universe_constellation (public) ---

type UniverseConstellation struct {
	ConstellationID int64    `json:"constellationId"`
	RegionID        int64    `json:"regionId"`
	Name            string   `json:"name"`
	Position        Position `json:"position"`
	SystemIDs       []int64  `json:"systemIds"`
}

func getUniverseConstellation(ctx context.Context, sdeStore constellationCache, client universeConstellationsESI, userAgent string, constellationID int64) (UniverseConstellation, error) {
	c, ok, err := sdeStore.Constellation(ctx, userAgent, constellationID)
	if err != nil {
		return UniverseConstellation{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseConstellationDetail(ctx, constellationID)
		if err != nil {
			return UniverseConstellation{}, err
		}
		c = sde.Constellation{
			ID: detail.ConstellationId, RegionID: detail.RegionId, Name: detail.Name,
			Position:  sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			SystemIDs: detail.Systems,
		}
		if err := sdeStore.PutConstellation(c); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through constellation into cache", "constellationId", c.ID, "error", err)
		}
	}
	return UniverseConstellation{
		ConstellationID: c.ID, RegionID: c.RegionID, Name: c.Name,
		Position:  Position{X: c.Position.X, Y: c.Position.Y, Z: c.Position.Z},
		SystemIDs: c.SystemIDs,
	}, nil
}

type GetUniverseConstellationInput struct {
	ConstellationID int64 `json:"constellationId" jsonschema:"the constellation ID to look up"`
}

// AddGetUniverseConstellation registers get_universe_constellation. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseConstellation[C universeConstellationsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore constellationCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_constellation",
		Description: "Get a constellation's name, region, position, and member system IDs (public data, no character required).",
	}, getUniverseConstellationHandler(userAgent, open, sdeStore))
}

func getUniverseConstellationHandler[C universeConstellationsESI](userAgent string, open func() C, sdeStore constellationCache) mcpsdk.ToolHandlerFor[GetUniverseConstellationInput, UniverseConstellation] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseConstellationInput) (*mcpsdk.CallToolResult, UniverseConstellation, error) {
		out, err := getUniverseConstellation(ctx, sdeStore, open(), userAgent, in.ConstellationID)
		if err != nil {
			return nil, UniverseConstellation{}, fmt.Errorf("get universe constellation: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_systems (public, no input) ---

type systemCache interface {
	System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error)
	PutSystem(sde.System) error
	SystemIDs(ctx context.Context, userAgent string) ([]int64, error)
}

// universeSystemsESI is the ESI calls list_universe_systems and get_universe_system make on an SDE cache miss.
type universeSystemsESI interface {
	UniverseSystems(ctx context.Context) ([]int64, error)
	UniverseSystemDetail(ctx context.Context, systemID int64) (*esiapi.UniverseSystemsSystemIdGet, error)
}

func listUniverseSystems(ctx context.Context, sdeStore systemCache, client universeSystemsESI, userAgent string) ([]int64, error) {
	ids, err := sdeStore.SystemIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	return client.UniverseSystems(ctx)
}

type ListUniverseSystemsInput struct{}
type ListUniverseSystemsOutput struct {
	IDs []int64 `json:"ids"`
}

// AddListUniverseSystems registers list_universe_systems. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseSystems[C universeSystemsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore systemCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_systems",
		Description: "List every solar system ID (public data, no character required).",
	}, listUniverseSystemsHandler(userAgent, open, sdeStore))
}

func listUniverseSystemsHandler[C universeSystemsESI](userAgent string, open func() C, sdeStore systemCache) mcpsdk.ToolHandlerFor[ListUniverseSystemsInput, ListUniverseSystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseSystemsInput) (*mcpsdk.CallToolResult, ListUniverseSystemsOutput, error) {
		ids, err := listUniverseSystems(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseSystemsOutput{}, fmt.Errorf("list universe systems: %w", err)
		}
		return nil, ListUniverseSystemsOutput{IDs: ids}, nil
	}
}

// --- get_universe_system (public) ---

type UniverseSystemPlanet struct {
	PlanetID        int64   `json:"planetId"`
	AsteroidBeltIDs []int64 `json:"asteroidBeltIds,omitempty"`
	MoonIDs         []int64 `json:"moonIds,omitempty"`
}

type UniverseSystem struct {
	SystemID        int64                  `json:"systemId"`
	ConstellationID int64                  `json:"constellationId"`
	Name            string                 `json:"name"`
	Position        Position               `json:"position"`
	SecurityStatus  float64                `json:"securityStatus"`
	SecurityDisplay string                 `json:"securityDisplay" jsonschema:"the security value the game shows, rounded to one decimal; quote this one to the user, e.g. Aunenen 0.4"`
	SecurityColor   string                 `json:"securityColor" jsonschema:"the in-game color for the displayed security, as #RRGGBB, for formatted output"`
	SecurityClass   *string                `json:"securityClass,omitempty"`
	StarID          *int64                 `json:"starId,omitempty"`
	StargateIDs     []int64                `json:"stargateIds,omitempty"`
	StationIDs      []int64                `json:"stationIds,omitempty"`
	Planets         []UniverseSystemPlanet `json:"planets,omitempty"`
}

func getUniverseSystem(ctx context.Context, sdeStore systemCache, client universeSystemsESI, userAgent string, systemID int64) (UniverseSystem, error) {
	sys, ok, err := sdeStore.System(ctx, userAgent, systemID)
	if err != nil {
		return UniverseSystem{}, fmt.Errorf("read sde cache: %w", err)
	}

	if !ok {
		detail, err := client.UniverseSystemDetail(ctx, systemID)
		if err != nil {
			return UniverseSystem{}, err
		}
		sys = sde.System{
			ID: detail.SystemId, ConstellationID: detail.ConstellationId, Name: detail.Name,
			Position:       sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			SecurityStatus: detail.SecurityStatus, SecurityClass: detail.SecurityClass, StarID: detail.StarId,
			StargateIDs: detail.Stargates, StationIDs: detail.Stations,
		}
		if len(detail.Planets) > 0 {
			sys.Planets = make([]sde.SystemPlanet, len(detail.Planets))
			for i, p := range detail.Planets {
				sys.Planets[i] = sde.SystemPlanet{PlanetID: p.PlanetId, AsteroidBeltIDs: p.AsteroidBelts, MoonIDs: p.Moons}
			}
		}
		if err := sdeStore.PutSystem(sys); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through system into cache", "systemId", sys.ID, "error", err)
		}
	}

	var planets []UniverseSystemPlanet
	if len(sys.Planets) > 0 {
		planets = make([]UniverseSystemPlanet, len(sys.Planets))
		for i, p := range sys.Planets {
			planets[i] = UniverseSystemPlanet{PlanetID: p.PlanetID, AsteroidBeltIDs: p.AsteroidBeltIDs, MoonIDs: p.MoonIDs}
		}
	}
	return UniverseSystem{
		SystemID: sys.ID, ConstellationID: sys.ConstellationID, Name: sys.Name,
		Position:       Position{X: sys.Position.X, Y: sys.Position.Y, Z: sys.Position.Z},
		SecurityStatus: sys.SecurityStatus, SecurityClass: sys.SecurityClass, StarID: sys.StarID,
		SecurityDisplay: nav.SecurityLabel(sys.SecurityStatus), SecurityColor: nav.SecurityColor(sys.SecurityStatus),
		StargateIDs: sys.StargateIDs, StationIDs: sys.StationIDs, Planets: planets,
	}, nil
}

type GetUniverseSystemInput struct {
	SystemID int64 `json:"systemId" jsonschema:"the solar system ID to look up"`
}

// AddGetUniverseSystem registers get_universe_system. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseSystem[C universeSystemsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore systemCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_system",
		Description: "Get a solar system's name, constellation, position, security status/class, star, stargates, stations, and planets (public data, no character required).",
	}, getUniverseSystemHandler(userAgent, open, sdeStore))
}

func getUniverseSystemHandler[C universeSystemsESI](userAgent string, open func() C, sdeStore systemCache) mcpsdk.ToolHandlerFor[GetUniverseSystemInput, UniverseSystem] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseSystemInput) (*mcpsdk.CallToolResult, UniverseSystem, error) {
		out, err := getUniverseSystem(ctx, sdeStore, open(), userAgent, in.SystemID)
		if err != nil {
			return nil, UniverseSystem{}, fmt.Errorf("get universe system: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_asteroid_belt (public) ---

type UniverseAsteroidBelt struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
	SystemID int64    `json:"systemId"`
}

type asteroidBeltCache interface {
	AsteroidBelt(ctx context.Context, userAgent string, id int64) (sde.AsteroidBelt, bool, error)
	PutAsteroidBelt(sde.AsteroidBelt) error
}

// universeAsteroidBeltESI is the ESI call get_universe_asteroid_belt makes on an SDE cache miss.
type universeAsteroidBeltESI interface {
	UniverseAsteroidBeltDetail(ctx context.Context, asteroidBeltID int64) (*esiapi.UniverseAsteroidBeltsAsteroidBeltIdGet, error)
}

func getUniverseAsteroidBelt(ctx context.Context, sdeStore asteroidBeltCache, client universeAsteroidBeltESI, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
	b, ok, err := sdeStore.AsteroidBelt(ctx, userAgent, asteroidBeltID)
	if err != nil {
		return UniverseAsteroidBelt{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseAsteroidBeltDetail(ctx, asteroidBeltID)
		if err != nil {
			return UniverseAsteroidBelt{}, err
		}
		b = sde.AsteroidBelt{
			ID: asteroidBeltID, Name: detail.Name, SystemID: detail.SystemId,
			Position: sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
		}
		if err := sdeStore.PutAsteroidBelt(b); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through asteroid belt into cache", "asteroidBeltId", b.ID, "error", err)
		}
	}
	return UniverseAsteroidBelt{Name: b.Name, Position: Position{X: b.Position.X, Y: b.Position.Y, Z: b.Position.Z}, SystemID: b.SystemID}, nil
}

type GetUniverseAsteroidBeltInput struct {
	AsteroidBeltID int64 `json:"asteroidBeltId" jsonschema:"the asteroid belt ID to look up"`
}

// AddGetUniverseAsteroidBelt registers get_universe_asteroid_belt. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseAsteroidBelt[C universeAsteroidBeltESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore asteroidBeltCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_asteroid_belt",
		Description: "Get an asteroid belt's name, position, and solar system (public data, no character required).",
	}, getUniverseAsteroidBeltHandler(userAgent, open, sdeStore))
}

func getUniverseAsteroidBeltHandler[C universeAsteroidBeltESI](userAgent string, open func() C, sdeStore asteroidBeltCache) mcpsdk.ToolHandlerFor[GetUniverseAsteroidBeltInput, UniverseAsteroidBelt] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseAsteroidBeltInput) (*mcpsdk.CallToolResult, UniverseAsteroidBelt, error) {
		out, err := getUniverseAsteroidBelt(ctx, sdeStore, open(), userAgent, in.AsteroidBeltID)
		if err != nil {
			return nil, UniverseAsteroidBelt{}, fmt.Errorf("get universe asteroid belt: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_moon (public) ---

type UniverseMoon struct {
	MoonID   int64    `json:"moonId"`
	SystemID int64    `json:"systemId"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

type moonCache interface {
	Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error)
	PutMoon(sde.Moon) error
}

// universeMoonESI is the ESI call get_universe_moon makes on an SDE cache miss.
type universeMoonESI interface {
	UniverseMoonDetail(ctx context.Context, moonID int64) (*esiapi.UniverseMoonsMoonIdGet, error)
}

func getUniverseMoon(ctx context.Context, sdeStore moonCache, client universeMoonESI, userAgent string, moonID int64) (UniverseMoon, error) {
	m, ok, err := sdeStore.Moon(ctx, userAgent, moonID)
	if err != nil {
		return UniverseMoon{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseMoonDetail(ctx, moonID)
		if err != nil {
			return UniverseMoon{}, err
		}
		m = sde.Moon{
			ID: detail.MoonId, Name: detail.Name, SystemID: detail.SystemId,
			Position: sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
		}
		if err := sdeStore.PutMoon(m); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through moon into cache", "moonId", m.ID, "error", err)
		}
	}
	return UniverseMoon{MoonID: m.ID, SystemID: m.SystemID, Name: m.Name, Position: Position{X: m.Position.X, Y: m.Position.Y, Z: m.Position.Z}}, nil
}

type GetUniverseMoonInput struct {
	MoonID int64 `json:"moonId" jsonschema:"the moon ID to look up"`
}

// AddGetUniverseMoon registers get_universe_moon. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseMoon[C universeMoonESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore moonCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_moon",
		Description: "Get a moon's name, position, and solar system (public data, no character required).",
	}, getUniverseMoonHandler(userAgent, open, sdeStore))
}

func getUniverseMoonHandler[C universeMoonESI](userAgent string, open func() C, sdeStore moonCache) mcpsdk.ToolHandlerFor[GetUniverseMoonInput, UniverseMoon] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseMoonInput) (*mcpsdk.CallToolResult, UniverseMoon, error) {
		out, err := getUniverseMoon(ctx, sdeStore, open(), userAgent, in.MoonID)
		if err != nil {
			return nil, UniverseMoon{}, fmt.Errorf("get universe moon: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_planet (public) ---

type UniversePlanet struct {
	PlanetID int64    `json:"planetId"`
	SystemID int64    `json:"systemId"`
	TypeID   int64    `json:"typeId"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

type planetCache interface {
	Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error)
	PutPlanet(sde.Planet) error
}

// universePlanetESI is the ESI call get_universe_planet makes on an SDE cache miss.
type universePlanetESI interface {
	UniversePlanetDetail(ctx context.Context, planetID int64) (*esiapi.UniversePlanetsPlanetIdGet, error)
}

func getUniversePlanet(ctx context.Context, sdeStore planetCache, client universePlanetESI, userAgent string, planetID int64) (UniversePlanet, error) {
	p, ok, err := sdeStore.Planet(ctx, userAgent, planetID)
	if err != nil {
		return UniversePlanet{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniversePlanetDetail(ctx, planetID)
		if err != nil {
			return UniversePlanet{}, err
		}
		p = sde.Planet{
			ID: detail.PlanetId, Name: detail.Name, TypeID: detail.TypeId, SystemID: detail.SystemId,
			Position: sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
		}
		if err := sdeStore.PutPlanet(p); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through planet into cache", "planetId", p.ID, "error", err)
		}
	}
	return UniversePlanet{
		PlanetID: p.ID, SystemID: p.SystemID, TypeID: p.TypeID, Name: p.Name,
		Position: Position{X: p.Position.X, Y: p.Position.Y, Z: p.Position.Z},
	}, nil
}

type GetUniversePlanetInput struct {
	PlanetID int64 `json:"planetId" jsonschema:"the planet ID to look up"`
}

// AddGetUniversePlanet registers get_universe_planet. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniversePlanet[C universePlanetESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore planetCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_planet",
		Description: "Get a planet's name, type, position, and solar system (public data, no character required).",
	}, getUniversePlanetHandler(userAgent, open, sdeStore))
}

func getUniversePlanetHandler[C universePlanetESI](userAgent string, open func() C, sdeStore planetCache) mcpsdk.ToolHandlerFor[GetUniversePlanetInput, UniversePlanet] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniversePlanetInput) (*mcpsdk.CallToolResult, UniversePlanet, error) {
		out, err := getUniversePlanet(ctx, sdeStore, open(), userAgent, in.PlanetID)
		if err != nil {
			return nil, UniversePlanet{}, fmt.Errorf("get universe planet: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_stargate (public) ---

type UniverseStargate struct {
	StargateID            int64    `json:"stargateId"`
	SystemID              int64    `json:"systemId"`
	TypeID                int64    `json:"typeId"`
	Name                  string   `json:"name"`
	Position              Position `json:"position"`
	DestinationStargateID int64    `json:"destinationStargateId"`
	DestinationSystemID   int64    `json:"destinationSystemId"`
}

type stargateCache interface {
	Stargate(ctx context.Context, userAgent string, id int64) (sde.Stargate, bool, error)
	PutStargate(sde.Stargate) error
}

// universeStargateESI is the ESI call get_universe_stargate makes on an SDE cache miss.
type universeStargateESI interface {
	UniverseStargateDetail(ctx context.Context, stargateID int64) (*esiapi.UniverseStargatesStargateIdGet, error)
}

func getUniverseStargate(ctx context.Context, sdeStore stargateCache, client universeStargateESI, userAgent string, stargateID int64) (UniverseStargate, error) {
	g, ok, err := sdeStore.Stargate(ctx, userAgent, stargateID)
	if err != nil {
		return UniverseStargate{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseStargateDetail(ctx, stargateID)
		if err != nil {
			return UniverseStargate{}, err
		}
		g = sde.Stargate{
			ID: detail.StargateId, Name: detail.Name, TypeID: detail.TypeId, SystemID: detail.SystemId,
			Position:              sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			DestinationStargateID: detail.Destination.StargateId, DestinationSystemID: detail.Destination.SystemId,
		}
		if err := sdeStore.PutStargate(g); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through stargate into cache", "stargateId", g.ID, "error", err)
		}
	}
	return UniverseStargate{
		StargateID: g.ID, SystemID: g.SystemID, TypeID: g.TypeID, Name: g.Name,
		Position:              Position{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z},
		DestinationStargateID: g.DestinationStargateID, DestinationSystemID: g.DestinationSystemID,
	}, nil
}

type GetUniverseStargateInput struct {
	StargateID int64 `json:"stargateId" jsonschema:"the stargate ID to look up"`
}

// AddGetUniverseStargate registers get_universe_stargate. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseStargate[C universeStargateESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore stargateCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_stargate",
		Description: "Get a stargate's name, type, position, solar system, and destination (public data, no character required).",
	}, getUniverseStargateHandler(userAgent, open, sdeStore))
}

func getUniverseStargateHandler[C universeStargateESI](userAgent string, open func() C, sdeStore stargateCache) mcpsdk.ToolHandlerFor[GetUniverseStargateInput, UniverseStargate] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStargateInput) (*mcpsdk.CallToolResult, UniverseStargate, error) {
		out, err := getUniverseStargate(ctx, sdeStore, open(), userAgent, in.StargateID)
		if err != nil {
			return nil, UniverseStargate{}, fmt.Errorf("get universe stargate: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_star (public) ---

type UniverseStar struct {
	StarID        int64   `json:"starId"`
	Age           int64   `json:"age"`
	Luminosity    float64 `json:"luminosity"`
	Name          string  `json:"name"`
	Radius        int64   `json:"radius"`
	SolarSystemID int64   `json:"solarSystemId"`
	SpectralClass string  `json:"spectralClass"`
	Temperature   int64   `json:"temperature"`
	TypeID        int64   `json:"typeId"`
}

type starCache interface {
	Star(ctx context.Context, userAgent string, id int64) (sde.Star, bool, error)
	PutStar(sde.Star) error
}

// universeStarESI is the ESI call get_universe_star makes on an SDE cache miss.
type universeStarESI interface {
	UniverseStarDetail(ctx context.Context, starID int64) (*esiapi.UniverseStarsStarIdGet, error)
}

func getUniverseStar(ctx context.Context, sdeStore starCache, client universeStarESI, userAgent string, starID int64) (UniverseStar, error) {
	s, ok, err := sdeStore.Star(ctx, userAgent, starID)
	if err != nil {
		return UniverseStar{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseStarDetail(ctx, starID)
		if err != nil {
			return UniverseStar{}, err
		}
		s = sde.Star{
			ID: starID, Age: detail.Age, Luminosity: detail.Luminosity, Name: detail.Name, Radius: detail.Radius,
			SolarSystemID: detail.SolarSystemId, SpectralClass: detail.SpectralClass, Temperature: detail.Temperature, TypeID: detail.TypeId,
		}
		if err := sdeStore.PutStar(s); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through star into cache", "starId", s.ID, "error", err)
		}
	}
	return UniverseStar{
		StarID: s.ID, Age: s.Age, Luminosity: s.Luminosity, Name: s.Name, Radius: s.Radius,
		SolarSystemID: s.SolarSystemID, SpectralClass: s.SpectralClass, Temperature: s.Temperature, TypeID: s.TypeID,
	}, nil
}

type GetUniverseStarInput struct {
	StarID int64 `json:"starId" jsonschema:"the star ID to look up"`
}

// AddGetUniverseStar registers get_universe_star. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseStar[C universeStarESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore starCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_star",
		Description: "Get a star's age, luminosity, name, radius, solar system, spectral class, temperature, and type (public data, no character required).",
	}, getUniverseStarHandler(userAgent, open, sdeStore))
}

func getUniverseStarHandler[C universeStarESI](userAgent string, open func() C, sdeStore starCache) mcpsdk.ToolHandlerFor[GetUniverseStarInput, UniverseStar] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStarInput) (*mcpsdk.CallToolResult, UniverseStar, error) {
		out, err := getUniverseStar(ctx, sdeStore, open(), userAgent, in.StarID)
		if err != nil {
			return nil, UniverseStar{}, fmt.Errorf("get universe star: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_station (public) ---

type UniverseStation struct {
	StationID                int64     `json:"stationId"`
	SystemID                 int64     `json:"systemId"`
	TypeID                   int64     `json:"typeId"`
	Name                     string    `json:"name"`
	Position                 Position  `json:"position"`
	MaxDockableShipVolume    float64   `json:"maxDockableShipVolume"`
	OfficeRentalCost         float64   `json:"officeRentalCost"`
	Owner                    *int64    `json:"owner,omitempty"`
	RaceID                   *int64    `json:"raceId,omitempty"`
	ReprocessingEfficiency   float64   `json:"reprocessingEfficiency"`
	ReprocessingStationsTake float64   `json:"reprocessingStationsTake"`
	Services                 *[]string `json:"services,omitempty"`
}

type stationCache interface {
	Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error)
	PutStation(sde.Station) error
}

// universeStationESI is the ESI call get_universe_station makes on an SDE cache miss.
type universeStationESI interface {
	UniverseStationDetail(ctx context.Context, stationID int64) (*esiapi.UniverseStationsStationIdGet, error)
}

func getUniverseStation(ctx context.Context, sdeStore stationCache, client universeStationESI, userAgent string, stationID int64) (UniverseStation, error) {
	s, ok, err := sdeStore.Station(ctx, userAgent, stationID)
	if err != nil {
		return UniverseStation{}, fmt.Errorf("read sde cache: %w", err)
	}
	if !ok {
		detail, err := client.UniverseStationDetail(ctx, stationID)
		if err != nil {
			return UniverseStation{}, err
		}
		s = sde.Station{
			ID: detail.StationId, Name: detail.Name, SystemID: detail.SystemId, TypeID: detail.TypeId,
			Position:              sde.Position{X: detail.Position.X, Y: detail.Position.Y, Z: detail.Position.Z},
			MaxDockableShipVolume: detail.MaxDockableShipVolume, OfficeRentalCost: detail.OfficeRentalCost,
			Owner: detail.Owner, RaceID: detail.RaceId, ReprocessingEfficiency: detail.ReprocessingEfficiency,
			ReprocessingStationsTake: detail.ReprocessingStationsTake, Services: &detail.Services,
		}
		if err := sdeStore.PutStation(s); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through station into cache", "stationId", s.ID, "error", err)
		}
	}
	return UniverseStation{
		StationID: s.ID, SystemID: s.SystemID, TypeID: s.TypeID, Name: s.Name,
		Position:              Position{X: s.Position.X, Y: s.Position.Y, Z: s.Position.Z},
		MaxDockableShipVolume: s.MaxDockableShipVolume, OfficeRentalCost: s.OfficeRentalCost,
		Owner: s.Owner, RaceID: s.RaceID, ReprocessingEfficiency: s.ReprocessingEfficiency,
		ReprocessingStationsTake: s.ReprocessingStationsTake, Services: s.Services,
	}, nil
}

type GetUniverseStationInput struct {
	StationID int64 `json:"stationId" jsonschema:"the station ID to look up"`
}

// AddGetUniverseStation registers get_universe_station. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseStation[C universeStationESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore stationCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_station",
		Description: "Get a station's name, type, position, solar system, owner, services, and docking/reprocessing stats (public data, no character required).",
	}, getUniverseStationHandler(userAgent, open, sdeStore))
}

func getUniverseStationHandler[C universeStationESI](userAgent string, open func() C, sdeStore stationCache) mcpsdk.ToolHandlerFor[GetUniverseStationInput, UniverseStation] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStationInput) (*mcpsdk.CallToolResult, UniverseStation, error) {
		out, err := getUniverseStation(ctx, sdeStore, open(), userAgent, in.StationID)
		if err != nil {
			return nil, UniverseStation{}, fmt.Errorf("get universe station: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_graphics (public, no input) ---

type graphicCache interface {
	Graphic(ctx context.Context, userAgent string, id int64) (sde.Graphic, bool, error)
	PutGraphic(sde.Graphic) error
	GraphicIDs(ctx context.Context, userAgent string) ([]int64, error)
}

// universeGraphicsESI is the ESI calls list_universe_graphics and get_universe_graphic make on an SDE cache miss.
type universeGraphicsESI interface {
	UniverseGraphics(ctx context.Context) ([]int64, error)
	UniverseGraphicDetail(ctx context.Context, graphicID int64) (*esiapi.UniverseGraphicsGraphicIdGet, error)
}

func listUniverseGraphics(ctx context.Context, sdeStore graphicCache, client universeGraphicsESI, userAgent string) ([]int64, error) {
	ids, err := sdeStore.GraphicIDs(ctx, userAgent)
	if err != nil {
		return nil, fmt.Errorf("read sde cache: %w", err)
	}
	if len(ids) > 0 {
		return ids, nil
	}
	return client.UniverseGraphics(ctx)
}

type ListUniverseGraphicsInput struct{}
type ListUniverseGraphicsOutput struct {
	IDs []int64 `json:"ids"`
}

// AddListUniverseGraphics registers list_universe_graphics. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseGraphics[C universeGraphicsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore graphicCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_graphics",
		Description: "List every graphic ID (public data, no character required).",
	}, listUniverseGraphicsHandler(userAgent, open, sdeStore))
}

func listUniverseGraphicsHandler[C universeGraphicsESI](userAgent string, open func() C, sdeStore graphicCache) mcpsdk.ToolHandlerFor[ListUniverseGraphicsInput, ListUniverseGraphicsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseGraphicsInput) (*mcpsdk.CallToolResult, ListUniverseGraphicsOutput, error) {
		ids, err := listUniverseGraphics(ctx, sdeStore, open(), userAgent)
		if err != nil {
			return nil, ListUniverseGraphicsOutput{}, fmt.Errorf("list universe graphics: %w", err)
		}
		return nil, ListUniverseGraphicsOutput{IDs: ids}, nil
	}
}

// --- get_universe_graphic (public) ---

type UniverseGraphic struct {
	GraphicID      int64   `json:"graphicId"`
	CollisionFile  *string `json:"collisionFile,omitempty"`
	GraphicFile    *string `json:"graphicFile,omitempty"`
	IconFolder     *string `json:"iconFolder,omitempty"`
	SofDNA         *string `json:"sofDna,omitempty"`
	SofFactionName *string `json:"sofFactionName,omitempty"`
	SofHullName    *string `json:"sofHullName,omitempty"`
	SofRaceName    *string `json:"sofRaceName,omitempty"`
}

func getUniverseGraphic(ctx context.Context, sdeStore graphicCache, client universeGraphicsESI, userAgent string, graphicID int64) (UniverseGraphic, error) {
	g, ok, err := sdeStore.Graphic(ctx, userAgent, graphicID)
	if err != nil {
		return UniverseGraphic{}, fmt.Errorf("read sde cache: %w", err)
	}

	// CollisionFile has no SDE source and no esi-scrape source either --
	// confirmed absent from all 6051 graphics.jsonl records AND from every
	// record in the esi-scrape mirror of ESI's own live responses, so it is
	// not a sourcing gap this migration can close: it's only available
	// fresh off a cache-miss ESI call, and is the one permanent
	// field drop left in the design spec's field-drop table.
	var collisionFile *string
	if !ok {
		detail, err := client.UniverseGraphicDetail(ctx, graphicID)
		if err != nil {
			return UniverseGraphic{}, err
		}
		collisionFile = detail.CollisionFile
		g = sde.Graphic{
			ID: detail.GraphicId, GraphicFile: detail.GraphicFile, IconFolder: detail.IconFolder, SofDNA: detail.SofDna,
			SofFactionName: detail.SofFationName, SofHullName: detail.SofHullName, SofRaceName: detail.SofRaceName,
		}
		if err := sdeStore.PutGraphic(g); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to write read-through graphic into cache", "graphicId", g.ID, "error", err)
		}
	}

	return UniverseGraphic{
		GraphicID: g.ID, CollisionFile: collisionFile, GraphicFile: g.GraphicFile, IconFolder: g.IconFolder,
		SofDNA: g.SofDNA, SofFactionName: g.SofFactionName, SofHullName: g.SofHullName, SofRaceName: g.SofRaceName,
	}, nil
}

type GetUniverseGraphicInput struct {
	GraphicID int64 `json:"graphicId" jsonschema:"the graphic ID to look up"`
}

// AddGetUniverseGraphic registers get_universe_graphic. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseGraphic[C universeGraphicsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore graphicCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_graphic",
		Description: "Get a graphic's collision/graphic files, icon folder, and Space Object Factory (SOF) DNA/hull/race/faction names (public data, no character required).",
	}, getUniverseGraphicHandler(userAgent, open, sdeStore))
}

func getUniverseGraphicHandler[C universeGraphicsESI](userAgent string, open func() C, sdeStore graphicCache) mcpsdk.ToolHandlerFor[GetUniverseGraphicInput, UniverseGraphic] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseGraphicInput) (*mcpsdk.CallToolResult, UniverseGraphic, error) {
		out, err := getUniverseGraphic(ctx, sdeStore, open(), userAgent, in.GraphicID)
		if err != nil {
			return nil, UniverseGraphic{}, fmt.Errorf("get universe graphic: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_structures (public, bare structure IDs, optionally filtered by service) ---

// validUniverseStructureFilters are ESI's own accepted values for the filter param.
var validUniverseStructureFilters = map[string]bool{"": true, "market": true, "manufacturing_basic": true}

// universeStructuresESI is the public ESI call list_universe_structures makes.
type universeStructuresESI interface {
	UniverseStructures(ctx context.Context, filter string) ([]int64, error)
}

func listUniverseStructures(ctx context.Context, client universeStructuresESI, filter string) ([]int64, error) {
	return client.UniverseStructures(ctx, filter)
}

type ListUniverseStructuresInput struct {
	Filter string `json:"filter,omitempty" jsonschema:"only list public structures with this service online: market or manufacturing_basic; omit for no filter"`
}
type ListUniverseStructuresOutput struct {
	IDs []int64 `json:"ids"`
}

// AddListUniverseStructures registers list_universe_structures. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseStructures[C universeStructuresESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_structures",
		Description: "List every publicly-listed player structure ID (public data, no character required).",
	}, listUniverseStructuresHandler(open))
}

func listUniverseStructuresHandler[C universeStructuresESI](open func() C) mcpsdk.ToolHandlerFor[ListUniverseStructuresInput, ListUniverseStructuresOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseStructuresInput) (*mcpsdk.CallToolResult, ListUniverseStructuresOutput, error) {
		if !validUniverseStructureFilters[in.Filter] {
			return nil, ListUniverseStructuresOutput{}, fmt.Errorf("list universe structures: invalid filter %q, must be empty, market, or manufacturing_basic", in.Filter)
		}
		ids, err := listUniverseStructures(ctx, open(), in.Filter)
		if err != nil {
			return nil, ListUniverseStructuresOutput{}, fmt.Errorf("list universe structures: %w", err)
		}
		return nil, ListUniverseStructuresOutput{IDs: ids}, nil
	}
}

// --- get_universe_structure (private) ---
//
// The only private tool in this file (every other Universe tool is public): ESI requires OAuth2
// scope esi-universe.read_structures.v1 for GetUniverseStructuresStructureId (verified in
// esi/api/openapi.yaml), meaning the querying character must have docking access to the
// structure. There's no separate ownerId/corporationId input -- the structure itself is the
// resource, and ESI's own 403 is the sole authorization gate, same as list_structure_market_orders
// in mcp/market.go.

type UniverseStructure struct {
	Name          string    `json:"name"`
	OwnerID       int64     `json:"ownerId"`
	SolarSystemID int64     `json:"solarSystemId"`
	TypeID        *int64    `json:"typeId,omitempty"`
	Position      *Position `json:"position,omitempty"`
}

// universeStructureESI is the ESI call get_universe_structure makes; the
// trade hub and saved structure tools read structures through it too.
type universeStructureESI interface {
	UniverseStructureDetail(ctx context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error)
}

func getUniverseStructure(ctx context.Context, client universeStructureESI, structureID int64) (UniverseStructure, error) {
	s, err := client.UniverseStructureDetail(ctx, structureID)
	if err != nil {
		return UniverseStructure{}, err
	}
	out := UniverseStructure{Name: s.Name, OwnerID: s.OwnerId, SolarSystemID: s.SolarSystemId, TypeID: s.TypeId}
	if s.Position != nil {
		out.Position = &Position{X: s.Position.X, Y: s.Position.Y, Z: s.Position.Z}
	}
	return out, nil
}

type GetUniverseStructureInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check -- must have docking access to the structure"`
	StructureID int64 `json:"structureId" jsonschema:"the structure ID to look up"`
}

// AddGetUniverseStructure registers get_universe_structure. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetUniverseStructure[C universeStructureESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_universe_structure",
		Description: "Get a player structure's name, owner, solar system, type, and position. The given character must have docking access to the structure.",
	}, getUniverseStructureHandler(open, store))
}

func getUniverseStructureHandler[C universeStructureESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetUniverseStructureInput, UniverseStructure] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStructureInput) (*mcpsdk.CallToolResult, UniverseStructure, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, UniverseStructure{}, err
		}
		if !found {
			return nil, UniverseStructure{}, fmt.Errorf("get universe structure for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, UniverseStructure{}, fmt.Errorf("get universe structure: %w", err)
		}
		out, err := getUniverseStructure(ctx, client, in.StructureID)
		if err != nil {
			return nil, UniverseStructure{}, fmt.Errorf("get universe structure: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_system_jumps (public, no input) ---

type UniverseSystemJumps struct {
	SystemID  int64 `json:"systemId"`
	ShipJumps int64 `json:"shipJumps"`
}

// universeSystemJumpsESI is the public ESI call list_universe_system_jumps makes.
type universeSystemJumpsESI interface {
	UniverseSystemJumps(ctx context.Context) ([]esiapi.UniverseSystemJumpsGetInner, error)
}

func listUniverseSystemJumps(ctx context.Context, client universeSystemJumpsESI) ([]UniverseSystemJumps, error) {
	jumps, err := client.UniverseSystemJumps(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UniverseSystemJumps, len(jumps))
	for i, j := range jumps {
		out[i] = UniverseSystemJumps{SystemID: j.SystemId, ShipJumps: j.ShipJumps}
	}
	return out, nil
}

type ListUniverseSystemJumpsInput struct{}
type ListUniverseSystemJumpsOutput struct {
	SystemJumps []UniverseSystemJumps `json:"systemJumps"`
}

// AddListUniverseSystemJumps registers list_universe_system_jumps. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseSystemJumps[C universeSystemJumpsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_system_jumps",
		Description: "List ship jump activity per solar system over the last hour (public data, no character required).",
	}, listUniverseSystemJumpsHandler(open))
}

func listUniverseSystemJumpsHandler[C universeSystemJumpsESI](open func() C) mcpsdk.ToolHandlerFor[ListUniverseSystemJumpsInput, ListUniverseSystemJumpsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseSystemJumpsInput) (*mcpsdk.CallToolResult, ListUniverseSystemJumpsOutput, error) {
		jumps, err := listUniverseSystemJumps(ctx, open())
		if err != nil {
			return nil, ListUniverseSystemJumpsOutput{}, fmt.Errorf("list universe system jumps: %w", err)
		}
		return nil, ListUniverseSystemJumpsOutput{SystemJumps: jumps}, nil
	}
}

// --- list_universe_system_kills (public, no input) ---

type UniverseSystemKills struct {
	SystemID  int64 `json:"systemId"`
	NPCKills  int64 `json:"npcKills"`
	PodKills  int64 `json:"podKills"`
	ShipKills int64 `json:"shipKills"`
}

// universeSystemKillsESI is the public ESI call list_universe_system_kills
// makes; route planning reads recent kills through it too.
type universeSystemKillsESI interface {
	UniverseSystemKills(ctx context.Context) ([]esiapi.UniverseSystemKillsGetInner, error)
}

func listUniverseSystemKills(ctx context.Context, client universeSystemKillsESI) ([]UniverseSystemKills, error) {
	kills, err := client.UniverseSystemKills(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UniverseSystemKills, len(kills))
	for i, k := range kills {
		out[i] = UniverseSystemKills{SystemID: k.SystemId, NPCKills: k.NpcKills, PodKills: k.PodKills, ShipKills: k.ShipKills}
	}
	return out, nil
}

type ListUniverseSystemKillsInput struct{}
type ListUniverseSystemKillsOutput struct {
	SystemKills []UniverseSystemKills `json:"systemKills"`
}

// AddListUniverseSystemKills registers list_universe_system_kills. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListUniverseSystemKills[C universeSystemKillsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_universe_system_kills",
		Description: "List NPC/pod/ship kill activity per solar system over the last hour (public data, no character required).",
	}, listUniverseSystemKillsHandler(open))
}

func listUniverseSystemKillsHandler[C universeSystemKillsESI](open func() C) mcpsdk.ToolHandlerFor[ListUniverseSystemKillsInput, ListUniverseSystemKillsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseSystemKillsInput) (*mcpsdk.CallToolResult, ListUniverseSystemKillsOutput, error) {
		kills, err := listUniverseSystemKills(ctx, open())
		if err != nil {
			return nil, ListUniverseSystemKillsOutput{}, fmt.Errorf("list universe system kills: %w", err)
		}
		return nil, ListUniverseSystemKillsOutput{SystemKills: kills}, nil
	}
}

// --- resolve_universe_ids (public) ---

// IDNamePair flattens ESI's UniverseIdsPost category-item shape -- all 10 categories
// (agents, alliances, characters, constellations, corporations, factions, inventory_types,
// regions, stations, systems) use this identical {id, name} shape, verified individually from
// generated source for every one of the 10 inner types.
type IDNamePair struct {
	ID   *int64  `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type ResolveUniverseIDsOutput struct {
	Agents         []IDNamePair `json:"agents,omitempty"`
	Alliances      []IDNamePair `json:"alliances,omitempty"`
	Characters     []IDNamePair `json:"characters,omitempty"`
	Constellations []IDNamePair `json:"constellations,omitempty"`
	Corporations   []IDNamePair `json:"corporations,omitempty"`
	Factions       []IDNamePair `json:"factions,omitempty"`
	InventoryTypes []IDNamePair `json:"inventoryTypes,omitempty"`
	Regions        []IDNamePair `json:"regions,omitempty"`
	Stations       []IDNamePair `json:"stations,omitempty"`
	Systems        []IDNamePair `json:"systems,omitempty"`
}

// universeIDsESI is the public ESI call resolve_universe_ids makes.
type universeIDsESI interface {
	UniverseIDsByName(ctx context.Context, names []string) (*esiapi.UniverseIdsPost, error)
}

func resolveUniverseIDs(ctx context.Context, client universeIDsESI, names []string) (ResolveUniverseIDsOutput, error) {
	result, err := client.UniverseIDsByName(ctx, names)
	if err != nil {
		return ResolveUniverseIDsOutput{}, err
	}
	out := ResolveUniverseIDsOutput{}
	for _, a := range result.Agents {
		out.Agents = append(out.Agents, IDNamePair{ID: a.Id, Name: a.Name})
	}
	for _, a := range result.Alliances {
		out.Alliances = append(out.Alliances, IDNamePair{ID: a.Id, Name: a.Name})
	}
	for _, c := range result.Characters {
		out.Characters = append(out.Characters, IDNamePair{ID: c.Id, Name: c.Name})
	}
	for _, c := range result.Constellations {
		out.Constellations = append(out.Constellations, IDNamePair{ID: c.Id, Name: c.Name})
	}
	for _, c := range result.Corporations {
		out.Corporations = append(out.Corporations, IDNamePair{ID: c.Id, Name: c.Name})
	}
	for _, f := range result.Factions {
		out.Factions = append(out.Factions, IDNamePair{ID: f.Id, Name: f.Name})
	}
	for _, i := range result.InventoryTypes {
		out.InventoryTypes = append(out.InventoryTypes, IDNamePair{ID: i.Id, Name: i.Name})
	}
	for _, r := range result.Regions {
		out.Regions = append(out.Regions, IDNamePair{ID: r.Id, Name: r.Name})
	}
	for _, s := range result.Stations {
		out.Stations = append(out.Stations, IDNamePair{ID: s.Id, Name: s.Name})
	}
	for _, s := range result.Systems {
		out.Systems = append(out.Systems, IDNamePair{ID: s.Id, Name: s.Name})
	}
	return out, nil
}

type ResolveUniverseIDsInput struct {
	Names []string `json:"names" jsonschema:"the names to resolve to IDs, grouped by category in the response"`
}

// AddResolveUniverseIDs registers resolve_universe_ids. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddResolveUniverseIDs[C universeIDsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "resolve_universe_ids",
		Description: "Resolve a list of names to IDs, grouped by category: agents, alliances, characters, constellations, corporations, factions, inventory types, regions, stations, and systems. Names ESI can't match are simply omitted from the response, not reported as errors. Accepts 1-500 names per call (public data, no character required).",
	}, resolveUniverseIDsHandler(open))
}

func resolveUniverseIDsHandler[C universeIDsESI](open func() C) mcpsdk.ToolHandlerFor[ResolveUniverseIDsInput, ResolveUniverseIDsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ResolveUniverseIDsInput) (*mcpsdk.CallToolResult, ResolveUniverseIDsOutput, error) {
		// ESI's own documented cap for this endpoint (esi/api/openapi.yaml): 1-500 names.
		// Validate client-side rather than let an out-of-range request reach ESI as an opaque 400.
		if len(in.Names) == 0 {
			return nil, ResolveUniverseIDsOutput{}, fmt.Errorf("resolve universe ids: names must not be empty")
		}
		if len(in.Names) > 500 {
			return nil, ResolveUniverseIDsOutput{}, fmt.Errorf("resolve universe ids: got %d names, ESI accepts at most 500 per request", len(in.Names))
		}
		out, err := resolveUniverseIDs(ctx, open(), in.Names)
		if err != nil {
			return nil, ResolveUniverseIDsOutput{}, fmt.Errorf("resolve universe ids: %w", err)
		}
		return nil, out, nil
	}
}

// --- resolve_universe_names (public) ---

type ResolvedName struct {
	Category string `json:"category"`
	ID       int64  `json:"id"`
	Name     string `json:"name"`
}

// universeNamesESI is the public ESI call resolve_universe_names makes;
// save_structure names a structure's owner through it too.
type universeNamesESI interface {
	UniverseNamesByID(ctx context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error)
}

func resolveUniverseNames(ctx context.Context, client universeNamesESI, ids []int64) ([]ResolvedName, error) {
	names, err := client.UniverseNamesByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ResolvedName, len(names))
	for i, n := range names {
		out[i] = ResolvedName{Category: n.Category, ID: n.Id, Name: n.Name}
	}
	return out, nil
}

type ResolveUniverseNamesInput struct {
	IDs []int64 `json:"ids" jsonschema:"the IDs to resolve to names and categories"`
}
type ResolveUniverseNamesOutput struct {
	Names []ResolvedName `json:"names"`
}

// AddResolveUniverseNames registers resolve_universe_names. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddResolveUniverseNames[C universeNamesESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "resolve_universe_names",
		Description: "Resolve a list of IDs to their names and categories. Accepts 1-1000 IDs per call (public data, no character required).",
	}, resolveUniverseNamesHandler(open))
}

func resolveUniverseNamesHandler[C universeNamesESI](open func() C) mcpsdk.ToolHandlerFor[ResolveUniverseNamesInput, ResolveUniverseNamesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ResolveUniverseNamesInput) (*mcpsdk.CallToolResult, ResolveUniverseNamesOutput, error) {
		// ESI's own documented cap for this endpoint (esi/api/openapi.yaml): 1-1000 IDs.
		// Validate client-side rather than let an out-of-range request reach ESI as an opaque 400.
		if len(in.IDs) == 0 {
			return nil, ResolveUniverseNamesOutput{}, fmt.Errorf("resolve universe names: ids must not be empty")
		}
		if len(in.IDs) > 1000 {
			return nil, ResolveUniverseNamesOutput{}, fmt.Errorf("resolve universe names: got %d ids, ESI accepts at most 1000 per request", len(in.IDs))
		}
		names, err := resolveUniverseNames(ctx, open(), in.IDs)
		if err != nil {
			return nil, ResolveUniverseNamesOutput{}, fmt.Errorf("resolve universe names: %w", err)
		}
		return nil, ResolveUniverseNamesOutput{Names: names}, nil
	}
}
