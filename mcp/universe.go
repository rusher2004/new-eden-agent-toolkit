package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
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

var listUniverseAncestries = func(ctx context.Context, userAgent string) ([]UniverseAncestry, error) {
	client := esi.NewPublicClient(userAgent)
	ancestries, err := client.UniverseAncestries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UniverseAncestry, len(ancestries))
	for i, a := range ancestries {
		out[i] = UniverseAncestry{
			ID: a.Id, BloodlineID: a.BloodlineId, Description: a.Description, Name: a.Name,
			IconID: a.IconId, ShortDescription: a.ShortDescription,
		}
	}
	return out, nil
}

type ListUniverseAncestriesInput struct{}
type ListUniverseAncestriesOutput struct {
	Ancestries []UniverseAncestry `json:"ancestries"`
}

func listUniverseAncestriesHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseAncestriesInput, ListUniverseAncestriesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseAncestriesInput) (*mcpsdk.CallToolResult, ListUniverseAncestriesOutput, error) {
		ancestries, err := listUniverseAncestries(ctx, sso.UserAgent(version))
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

var listUniverseBloodlines = func(ctx context.Context, userAgent string) ([]UniverseBloodline, error) {
	client := esi.NewPublicClient(userAgent)
	bloodlines, err := client.UniverseBloodlines(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UniverseBloodline, len(bloodlines))
	for i, b := range bloodlines {
		out[i] = UniverseBloodline{
			ID: b.BloodlineId, Charisma: b.Charisma, CorporationID: b.CorporationId, Description: b.Description,
			Intelligence: b.Intelligence, Memory: b.Memory, Name: b.Name, Perception: b.Perception,
			RaceID: b.RaceId, ShipTypeID: b.ShipTypeId, Willpower: b.Willpower,
		}
	}
	return out, nil
}

type ListUniverseBloodlinesInput struct{}
type ListUniverseBloodlinesOutput struct {
	Bloodlines []UniverseBloodline `json:"bloodlines"`
}

func listUniverseBloodlinesHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseBloodlinesInput, ListUniverseBloodlinesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseBloodlinesInput) (*mcpsdk.CallToolResult, ListUniverseBloodlinesOutput, error) {
		bloodlines, err := listUniverseBloodlines(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseBloodlinesOutput{}, fmt.Errorf("list universe bloodlines: %w", err)
		}
		return nil, ListUniverseBloodlinesOutput{Bloodlines: bloodlines}, nil
	}
}

// --- list_universe_races (public, no input) ---

type UniverseRace struct {
	RaceID      int64  `json:"raceId"`
	AllianceID  int64  `json:"allianceId"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

var listUniverseRaces = func(ctx context.Context, userAgent string) ([]UniverseRace, error) {
	client := esi.NewPublicClient(userAgent)
	races, err := client.UniverseRaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UniverseRace, len(races))
	for i, r := range races {
		out[i] = UniverseRace{RaceID: r.RaceId, AllianceID: r.AllianceId, Description: r.Description, Name: r.Name}
	}
	return out, nil
}

type ListUniverseRacesInput struct{}
type ListUniverseRacesOutput struct {
	Races []UniverseRace `json:"races"`
}

func listUniverseRacesHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseRacesInput, ListUniverseRacesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseRacesInput) (*mcpsdk.CallToolResult, ListUniverseRacesOutput, error) {
		races, err := listUniverseRaces(ctx, sso.UserAgent(version))
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
	IsUnique             bool    `json:"isUnique"`
	SizeFactor           float64 `json:"sizeFactor"`
	StationCount         int64   `json:"stationCount"`
	StationSystemCount   int64   `json:"stationSystemCount"`
	CorporationID        *int64  `json:"corporationId,omitempty"`
	MilitiaCorporationID *int64  `json:"militiaCorporationId,omitempty"`
	SolarSystemID        *int64  `json:"solarSystemId,omitempty"`
}

var listUniverseFactions = func(ctx context.Context, userAgent string) ([]UniverseFaction, error) {
	client := esi.NewPublicClient(userAgent)
	factions, err := client.UniverseFactions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UniverseFaction, len(factions))
	for i, f := range factions {
		out[i] = UniverseFaction{
			FactionID: f.FactionId, Description: f.Description, Name: f.Name, IsUnique: f.IsUnique,
			SizeFactor: f.SizeFactor, StationCount: f.StationCount, StationSystemCount: f.StationSystemCount,
			CorporationID: f.CorporationId, MilitiaCorporationID: f.MilitiaCorporationId, SolarSystemID: f.SolarSystemId,
		}
	}
	return out, nil
}

type ListUniverseFactionsInput struct{}
type ListUniverseFactionsOutput struct {
	Factions []UniverseFaction `json:"factions"`
}

func listUniverseFactionsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseFactionsInput, ListUniverseFactionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseFactionsInput) (*mcpsdk.CallToolResult, ListUniverseFactionsOutput, error) {
		factions, err := listUniverseFactions(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListUniverseFactionsOutput{}, fmt.Errorf("list universe factions: %w", err)
		}
		return nil, ListUniverseFactionsOutput{Factions: factions}, nil
	}
}

// --- list_universe_categories (public, no input) ---

var listUniverseCategories = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseCategories(ctx)
}

type ListUniverseCategoriesInput struct{}
type ListUniverseCategoriesOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseCategoriesHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseCategoriesInput, ListUniverseCategoriesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseCategoriesInput) (*mcpsdk.CallToolResult, ListUniverseCategoriesOutput, error) {
		ids, err := listUniverseCategories(ctx, sso.UserAgent(version))
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

var getUniverseCategory = func(ctx context.Context, userAgent string, categoryID int64) (UniverseCategory, error) {
	client := esi.NewPublicClient(userAgent)
	c, err := client.UniverseCategoryDetail(ctx, categoryID)
	if err != nil {
		return UniverseCategory{}, err
	}
	return UniverseCategory{CategoryID: c.CategoryId, Name: c.Name, Published: c.Published, GroupIDs: c.Groups}, nil
}

type GetUniverseCategoryInput struct {
	CategoryID int64 `json:"categoryId" jsonschema:"the category ID to look up"`
}

func getUniverseCategoryHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseCategoryInput, UniverseCategory] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseCategoryInput) (*mcpsdk.CallToolResult, UniverseCategory, error) {
		out, err := getUniverseCategory(ctx, sso.UserAgent(version), in.CategoryID)
		if err != nil {
			return nil, UniverseCategory{}, fmt.Errorf("get universe category: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_groups (public, paginated) ---

var listUniverseGroups = func(ctx context.Context, userAgent string, page int32) ([]int64, int32, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseGroups(ctx, page)
}

type ListUniverseGroupsInput struct {
	Page int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListUniverseGroupsOutput struct {
	IDs        []int64 `json:"ids"`
	Page       int32   `json:"page"`
	TotalPages int32   `json:"totalPages"`
}

func listUniverseGroupsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseGroupsInput, ListUniverseGroupsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseGroupsInput) (*mcpsdk.CallToolResult, ListUniverseGroupsOutput, error) {
		ids, totalPages, err := listUniverseGroups(ctx, sso.UserAgent(version), in.Page)
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

var getUniverseGroup = func(ctx context.Context, userAgent string, groupID int64) (UniverseGroup, error) {
	client := esi.NewPublicClient(userAgent)
	g, err := client.UniverseGroupDetail(ctx, groupID)
	if err != nil {
		return UniverseGroup{}, err
	}
	return UniverseGroup{GroupID: g.GroupId, CategoryID: g.CategoryId, Name: g.Name, Published: g.Published, TypeIDs: g.Types}, nil
}

type GetUniverseGroupInput struct {
	GroupID int64 `json:"groupId" jsonschema:"the group ID to look up"`
}

func getUniverseGroupHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseGroupInput, UniverseGroup] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseGroupInput) (*mcpsdk.CallToolResult, UniverseGroup, error) {
		out, err := getUniverseGroup(ctx, sso.UserAgent(version), in.GroupID)
		if err != nil {
			return nil, UniverseGroup{}, fmt.Errorf("get universe group: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_types (public, paginated) ---

var listUniverseTypes = func(ctx context.Context, userAgent string, page int32) ([]int64, int32, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseTypes(ctx, page)
}

type ListUniverseTypesInput struct {
	Page int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListUniverseTypesOutput struct {
	IDs        []int64 `json:"ids"`
	Page       int32   `json:"page"`
	TotalPages int32   `json:"totalPages"`
}

func listUniverseTypesHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseTypesInput, ListUniverseTypesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseTypesInput) (*mcpsdk.CallToolResult, ListUniverseTypesOutput, error) {
		ids, totalPages, err := listUniverseTypes(ctx, sso.UserAgent(version), in.Page)
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

var getUniverseType = func(ctx context.Context, userAgent string, typeID int64) (UniverseType, error) {
	client := esi.NewPublicClient(userAgent)
	t, err := client.UniverseTypeDetail(ctx, typeID)
	if err != nil {
		return UniverseType{}, err
	}
	out := UniverseType{
		TypeID: t.TypeId, Name: t.Name, Description: t.Description, GroupID: t.GroupId, Published: t.Published,
		Capacity: t.Capacity, GraphicID: t.GraphicId, IconID: t.IconId, MarketGroupID: t.MarketGroupId,
		Mass: t.Mass, PackagedVolume: t.PackagedVolume, PortionSize: t.PortionSize, Radius: t.Radius, Volume: t.Volume,
	}
	if len(t.DogmaAttributes) > 0 {
		out.DogmaAttributes = make([]DynamicItemAttribute, len(t.DogmaAttributes))
		for i, a := range t.DogmaAttributes {
			out.DogmaAttributes[i] = DynamicItemAttribute{AttributeID: a.AttributeId, Value: a.Value}
		}
	}
	if len(t.DogmaEffects) > 0 {
		out.DogmaEffects = make([]DynamicItemEffect, len(t.DogmaEffects))
		for i, e := range t.DogmaEffects {
			out.DogmaEffects[i] = DynamicItemEffect{EffectID: e.EffectId, IsDefault: e.IsDefault}
		}
	}
	return out, nil
}

type GetUniverseTypeInput struct {
	TypeID int64 `json:"typeId" jsonschema:"the type ID to look up"`
}

func getUniverseTypeHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseTypeInput, UniverseType] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseTypeInput) (*mcpsdk.CallToolResult, UniverseType, error) {
		out, err := getUniverseType(ctx, sso.UserAgent(version), in.TypeID)
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

var listUniverseRegions = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseRegions(ctx)
}

type ListUniverseRegionsInput struct{}
type ListUniverseRegionsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseRegionsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseRegionsInput, ListUniverseRegionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseRegionsInput) (*mcpsdk.CallToolResult, ListUniverseRegionsOutput, error) {
		ids, err := listUniverseRegions(ctx, sso.UserAgent(version))
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

var getUniverseRegion = func(ctx context.Context, userAgent string, regionID int64) (UniverseRegion, error) {
	client := esi.NewPublicClient(userAgent)
	r, err := client.UniverseRegionDetail(ctx, regionID)
	if err != nil {
		return UniverseRegion{}, err
	}
	return UniverseRegion{RegionID: r.RegionId, Name: r.Name, Description: r.Description, ConstellationIDs: r.Constellations}, nil
}

type GetUniverseRegionInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
}

func getUniverseRegionHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseRegionInput, UniverseRegion] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseRegionInput) (*mcpsdk.CallToolResult, UniverseRegion, error) {
		out, err := getUniverseRegion(ctx, sso.UserAgent(version), in.RegionID)
		if err != nil {
			return nil, UniverseRegion{}, fmt.Errorf("get universe region: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_constellations (public, no input) ---

var listUniverseConstellations = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseConstellations(ctx)
}

type ListUniverseConstellationsInput struct{}
type ListUniverseConstellationsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseConstellationsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseConstellationsInput, ListUniverseConstellationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseConstellationsInput) (*mcpsdk.CallToolResult, ListUniverseConstellationsOutput, error) {
		ids, err := listUniverseConstellations(ctx, sso.UserAgent(version))
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

var getUniverseConstellation = func(ctx context.Context, userAgent string, constellationID int64) (UniverseConstellation, error) {
	client := esi.NewPublicClient(userAgent)
	c, err := client.UniverseConstellationDetail(ctx, constellationID)
	if err != nil {
		return UniverseConstellation{}, err
	}
	return UniverseConstellation{
		ConstellationID: c.ConstellationId, RegionID: c.RegionId, Name: c.Name,
		Position:  Position{X: c.Position.X, Y: c.Position.Y, Z: c.Position.Z},
		SystemIDs: c.Systems,
	}, nil
}

type GetUniverseConstellationInput struct {
	ConstellationID int64 `json:"constellationId" jsonschema:"the constellation ID to look up"`
}

func getUniverseConstellationHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseConstellationInput, UniverseConstellation] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseConstellationInput) (*mcpsdk.CallToolResult, UniverseConstellation, error) {
		out, err := getUniverseConstellation(ctx, sso.UserAgent(version), in.ConstellationID)
		if err != nil {
			return nil, UniverseConstellation{}, fmt.Errorf("get universe constellation: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_systems (public, no input) ---

var listUniverseSystems = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseSystems(ctx)
}

type ListUniverseSystemsInput struct{}
type ListUniverseSystemsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseSystemsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseSystemsInput, ListUniverseSystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseSystemsInput) (*mcpsdk.CallToolResult, ListUniverseSystemsOutput, error) {
		ids, err := listUniverseSystems(ctx, sso.UserAgent(version))
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
	SecurityClass   *string                `json:"securityClass,omitempty"`
	StarID          *int64                 `json:"starId,omitempty"`
	StargateIDs     []int64                `json:"stargateIds,omitempty"`
	StationIDs      []int64                `json:"stationIds,omitempty"`
	Planets         []UniverseSystemPlanet `json:"planets,omitempty"`
}

var getUniverseSystem = func(ctx context.Context, userAgent string, systemID int64) (UniverseSystem, error) {
	client := esi.NewPublicClient(userAgent)
	s, err := client.UniverseSystemDetail(ctx, systemID)
	if err != nil {
		return UniverseSystem{}, err
	}
	out := UniverseSystem{
		SystemID: s.SystemId, ConstellationID: s.ConstellationId, Name: s.Name,
		Position:       Position{X: s.Position.X, Y: s.Position.Y, Z: s.Position.Z},
		SecurityStatus: s.SecurityStatus, SecurityClass: s.SecurityClass, StarID: s.StarId,
		StargateIDs: s.Stargates, StationIDs: s.Stations,
	}
	if len(s.Planets) > 0 {
		out.Planets = make([]UniverseSystemPlanet, len(s.Planets))
		for i, p := range s.Planets {
			out.Planets[i] = UniverseSystemPlanet{PlanetID: p.PlanetId, AsteroidBeltIDs: p.AsteroidBelts, MoonIDs: p.Moons}
		}
	}
	return out, nil
}

type GetUniverseSystemInput struct {
	SystemID int64 `json:"systemId" jsonschema:"the solar system ID to look up"`
}

func getUniverseSystemHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseSystemInput, UniverseSystem] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseSystemInput) (*mcpsdk.CallToolResult, UniverseSystem, error) {
		out, err := getUniverseSystem(ctx, sso.UserAgent(version), in.SystemID)
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

var getUniverseAsteroidBelt = func(ctx context.Context, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
	client := esi.NewPublicClient(userAgent)
	b, err := client.UniverseAsteroidBeltDetail(ctx, asteroidBeltID)
	if err != nil {
		return UniverseAsteroidBelt{}, err
	}
	return UniverseAsteroidBelt{Name: b.Name, Position: Position{X: b.Position.X, Y: b.Position.Y, Z: b.Position.Z}, SystemID: b.SystemId}, nil
}

type GetUniverseAsteroidBeltInput struct {
	AsteroidBeltID int64 `json:"asteroidBeltId" jsonschema:"the asteroid belt ID to look up"`
}

func getUniverseAsteroidBeltHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseAsteroidBeltInput, UniverseAsteroidBelt] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseAsteroidBeltInput) (*mcpsdk.CallToolResult, UniverseAsteroidBelt, error) {
		out, err := getUniverseAsteroidBelt(ctx, sso.UserAgent(version), in.AsteroidBeltID)
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

var getUniverseMoon = func(ctx context.Context, userAgent string, moonID int64) (UniverseMoon, error) {
	client := esi.NewPublicClient(userAgent)
	m, err := client.UniverseMoonDetail(ctx, moonID)
	if err != nil {
		return UniverseMoon{}, err
	}
	return UniverseMoon{MoonID: m.MoonId, SystemID: m.SystemId, Name: m.Name, Position: Position{X: m.Position.X, Y: m.Position.Y, Z: m.Position.Z}}, nil
}

type GetUniverseMoonInput struct {
	MoonID int64 `json:"moonId" jsonschema:"the moon ID to look up"`
}

func getUniverseMoonHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseMoonInput, UniverseMoon] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseMoonInput) (*mcpsdk.CallToolResult, UniverseMoon, error) {
		out, err := getUniverseMoon(ctx, sso.UserAgent(version), in.MoonID)
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

var getUniversePlanet = func(ctx context.Context, userAgent string, planetID int64) (UniversePlanet, error) {
	client := esi.NewPublicClient(userAgent)
	p, err := client.UniversePlanetDetail(ctx, planetID)
	if err != nil {
		return UniversePlanet{}, err
	}
	return UniversePlanet{
		PlanetID: p.PlanetId, SystemID: p.SystemId, TypeID: p.TypeId, Name: p.Name,
		Position: Position{X: p.Position.X, Y: p.Position.Y, Z: p.Position.Z},
	}, nil
}

type GetUniversePlanetInput struct {
	PlanetID int64 `json:"planetId" jsonschema:"the planet ID to look up"`
}

func getUniversePlanetHandler(version string) mcpsdk.ToolHandlerFor[GetUniversePlanetInput, UniversePlanet] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniversePlanetInput) (*mcpsdk.CallToolResult, UniversePlanet, error) {
		out, err := getUniversePlanet(ctx, sso.UserAgent(version), in.PlanetID)
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

var getUniverseStargate = func(ctx context.Context, userAgent string, stargateID int64) (UniverseStargate, error) {
	client := esi.NewPublicClient(userAgent)
	g, err := client.UniverseStargateDetail(ctx, stargateID)
	if err != nil {
		return UniverseStargate{}, err
	}
	return UniverseStargate{
		StargateID: g.StargateId, SystemID: g.SystemId, TypeID: g.TypeId, Name: g.Name,
		Position:              Position{X: g.Position.X, Y: g.Position.Y, Z: g.Position.Z},
		DestinationStargateID: g.Destination.StargateId, DestinationSystemID: g.Destination.SystemId,
	}, nil
}

type GetUniverseStargateInput struct {
	StargateID int64 `json:"stargateId" jsonschema:"the stargate ID to look up"`
}

func getUniverseStargateHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseStargateInput, UniverseStargate] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStargateInput) (*mcpsdk.CallToolResult, UniverseStargate, error) {
		out, err := getUniverseStargate(ctx, sso.UserAgent(version), in.StargateID)
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

var getUniverseStar = func(ctx context.Context, userAgent string, starID int64) (UniverseStar, error) {
	client := esi.NewPublicClient(userAgent)
	s, err := client.UniverseStarDetail(ctx, starID)
	if err != nil {
		return UniverseStar{}, err
	}
	return UniverseStar{
		StarID: starID, Age: s.Age, Luminosity: s.Luminosity, Name: s.Name, Radius: s.Radius,
		SolarSystemID: s.SolarSystemId, SpectralClass: s.SpectralClass, Temperature: s.Temperature, TypeID: s.TypeId,
	}, nil
}

type GetUniverseStarInput struct {
	StarID int64 `json:"starId" jsonschema:"the star ID to look up"`
}

func getUniverseStarHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseStarInput, UniverseStar] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStarInput) (*mcpsdk.CallToolResult, UniverseStar, error) {
		out, err := getUniverseStar(ctx, sso.UserAgent(version), in.StarID)
		if err != nil {
			return nil, UniverseStar{}, fmt.Errorf("get universe star: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_universe_station (public) ---

type UniverseStation struct {
	StationID                int64    `json:"stationId"`
	SystemID                 int64    `json:"systemId"`
	TypeID                   int64    `json:"typeId"`
	Name                     string   `json:"name"`
	Position                 Position `json:"position"`
	MaxDockableShipVolume    float64  `json:"maxDockableShipVolume"`
	OfficeRentalCost         float64  `json:"officeRentalCost"`
	Owner                    *int64   `json:"owner,omitempty"`
	RaceID                   *int64   `json:"raceId,omitempty"`
	ReprocessingEfficiency   float64  `json:"reprocessingEfficiency"`
	ReprocessingStationsTake float64  `json:"reprocessingStationsTake"`
	Services                 []string `json:"services"`
}

var getUniverseStation = func(ctx context.Context, userAgent string, stationID int64) (UniverseStation, error) {
	client := esi.NewPublicClient(userAgent)
	s, err := client.UniverseStationDetail(ctx, stationID)
	if err != nil {
		return UniverseStation{}, err
	}
	return UniverseStation{
		StationID: s.StationId, SystemID: s.SystemId, TypeID: s.TypeId, Name: s.Name,
		Position:              Position{X: s.Position.X, Y: s.Position.Y, Z: s.Position.Z},
		MaxDockableShipVolume: s.MaxDockableShipVolume, OfficeRentalCost: s.OfficeRentalCost,
		Owner: s.Owner, RaceID: s.RaceId, ReprocessingEfficiency: s.ReprocessingEfficiency,
		ReprocessingStationsTake: s.ReprocessingStationsTake, Services: s.Services,
	}, nil
}

type GetUniverseStationInput struct {
	StationID int64 `json:"stationId" jsonschema:"the station ID to look up"`
}

func getUniverseStationHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseStationInput, UniverseStation] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStationInput) (*mcpsdk.CallToolResult, UniverseStation, error) {
		out, err := getUniverseStation(ctx, sso.UserAgent(version), in.StationID)
		if err != nil {
			return nil, UniverseStation{}, fmt.Errorf("get universe station: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_graphics (public, no input) ---

var listUniverseGraphics = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseGraphics(ctx)
}

type ListUniverseGraphicsInput struct{}
type ListUniverseGraphicsOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseGraphicsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseGraphicsInput, ListUniverseGraphicsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseGraphicsInput) (*mcpsdk.CallToolResult, ListUniverseGraphicsOutput, error) {
		ids, err := listUniverseGraphics(ctx, sso.UserAgent(version))
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

var getUniverseGraphic = func(ctx context.Context, userAgent string, graphicID int64) (UniverseGraphic, error) {
	client := esi.NewPublicClient(userAgent)
	g, err := client.UniverseGraphicDetail(ctx, graphicID)
	if err != nil {
		return UniverseGraphic{}, err
	}
	return UniverseGraphic{
		GraphicID: g.GraphicId, CollisionFile: g.CollisionFile, GraphicFile: g.GraphicFile, IconFolder: g.IconFolder,
		SofDNA: g.SofDna, SofFactionName: g.SofFationName, SofHullName: g.SofHullName, SofRaceName: g.SofRaceName,
	}, nil
}

type GetUniverseGraphicInput struct {
	GraphicID int64 `json:"graphicId" jsonschema:"the graphic ID to look up"`
}

func getUniverseGraphicHandler(version string) mcpsdk.ToolHandlerFor[GetUniverseGraphicInput, UniverseGraphic] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseGraphicInput) (*mcpsdk.CallToolResult, UniverseGraphic, error) {
		out, err := getUniverseGraphic(ctx, sso.UserAgent(version), in.GraphicID)
		if err != nil {
			return nil, UniverseGraphic{}, fmt.Errorf("get universe graphic: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_universe_structures (public, bare structure IDs, optionally filtered by service) ---

// validUniverseStructureFilters are ESI's own accepted values for the filter param.
var validUniverseStructureFilters = map[string]bool{"": true, "market": true, "manufacturing_basic": true}

var listUniverseStructures = func(ctx context.Context, userAgent string, filter string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.UniverseStructures(ctx, filter)
}

type ListUniverseStructuresInput struct {
	Filter string `json:"filter,omitempty" jsonschema:"only list public structures with this service online: market or manufacturing_basic; omit for no filter"`
}
type ListUniverseStructuresOutput struct {
	IDs []int64 `json:"ids"`
}

func listUniverseStructuresHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseStructuresInput, ListUniverseStructuresOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseStructuresInput) (*mcpsdk.CallToolResult, ListUniverseStructuresOutput, error) {
		if !validUniverseStructureFilters[in.Filter] {
			return nil, ListUniverseStructuresOutput{}, fmt.Errorf("list universe structures: invalid filter %q, must be empty, market, or manufacturing_basic", in.Filter)
		}
		ids, err := listUniverseStructures(ctx, sso.UserAgent(version), in.Filter)
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

var getUniverseStructure = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64) (UniverseStructure, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return UniverseStructure{}, fmt.Errorf("build ESI client: %w", err)
	}
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

func getUniverseStructureHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetUniverseStructureInput, UniverseStructure] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetUniverseStructureInput) (*mcpsdk.CallToolResult, UniverseStructure, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, UniverseStructure{}, err
		}
		if !found {
			return nil, UniverseStructure{}, fmt.Errorf("get universe structure for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		out, err := getUniverseStructure(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.StructureID)
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

var listUniverseSystemJumps = func(ctx context.Context, userAgent string) ([]UniverseSystemJumps, error) {
	client := esi.NewPublicClient(userAgent)
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

func listUniverseSystemJumpsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseSystemJumpsInput, ListUniverseSystemJumpsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseSystemJumpsInput) (*mcpsdk.CallToolResult, ListUniverseSystemJumpsOutput, error) {
		jumps, err := listUniverseSystemJumps(ctx, sso.UserAgent(version))
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

var listUniverseSystemKills = func(ctx context.Context, userAgent string) ([]UniverseSystemKills, error) {
	client := esi.NewPublicClient(userAgent)
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

func listUniverseSystemKillsHandler(version string) mcpsdk.ToolHandlerFor[ListUniverseSystemKillsInput, ListUniverseSystemKillsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListUniverseSystemKillsInput) (*mcpsdk.CallToolResult, ListUniverseSystemKillsOutput, error) {
		kills, err := listUniverseSystemKills(ctx, sso.UserAgent(version))
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

var resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
	client := esi.NewPublicClient(userAgent)
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

func resolveUniverseIDsHandler(version string) mcpsdk.ToolHandlerFor[ResolveUniverseIDsInput, ResolveUniverseIDsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ResolveUniverseIDsInput) (*mcpsdk.CallToolResult, ResolveUniverseIDsOutput, error) {
		// ESI's own documented cap for this endpoint (esi/api/openapi.yaml): 1-500 names.
		// Validate client-side rather than let an out-of-range request reach ESI as an opaque 400.
		if len(in.Names) == 0 {
			return nil, ResolveUniverseIDsOutput{}, fmt.Errorf("resolve universe ids: names must not be empty")
		}
		if len(in.Names) > 500 {
			return nil, ResolveUniverseIDsOutput{}, fmt.Errorf("resolve universe ids: got %d names, ESI accepts at most 500 per request", len(in.Names))
		}
		out, err := resolveUniverseIDs(ctx, sso.UserAgent(version), in.Names)
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

var resolveUniverseNames = func(ctx context.Context, userAgent string, ids []int64) ([]ResolvedName, error) {
	client := esi.NewPublicClient(userAgent)
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

func resolveUniverseNamesHandler(version string) mcpsdk.ToolHandlerFor[ResolveUniverseNamesInput, ResolveUniverseNamesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ResolveUniverseNamesInput) (*mcpsdk.CallToolResult, ResolveUniverseNamesOutput, error) {
		// ESI's own documented cap for this endpoint (esi/api/openapi.yaml): 1-1000 IDs.
		// Validate client-side rather than let an out-of-range request reach ESI as an opaque 400.
		if len(in.IDs) == 0 {
			return nil, ResolveUniverseNamesOutput{}, fmt.Errorf("resolve universe names: ids must not be empty")
		}
		if len(in.IDs) > 1000 {
			return nil, ResolveUniverseNamesOutput{}, fmt.Errorf("resolve universe names: got %d ids, ESI accepts at most 1000 per request", len(in.IDs))
		}
		names, err := resolveUniverseNames(ctx, sso.UserAgent(version), in.IDs)
		if err != nil {
			return nil, ResolveUniverseNamesOutput{}, fmt.Errorf("resolve universe names: %w", err)
		}
		return nil, ResolveUniverseNamesOutput{Names: names}, nil
	}
}
