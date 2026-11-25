package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// planetsESI is the ESI calls the authenticated planetary interaction tools make.
type planetsESI interface {
	CharacterPlanets(ctx context.Context) ([]esiapi.CharactersCharacterIdPlanetsGetInner, error)
	CharacterPlanetDetail(ctx context.Context, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error)
	CorporationCustomsOffices(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error)
}

// colonyNameCache is the SDE lookups list_character_planets makes: system and planet names.
type colonyNameCache interface {
	systemNameCache
	planetNameCache
}

// customsOfficeNameCache is the SDE lookups list_corporation_customs_offices makes: type and system names.
type customsOfficeNameCache interface {
	typeNameCache
	systemNameCache
}

// planetarySchematicESI is the ESI call get_planetary_schematic makes; it is public.
type planetarySchematicESI interface {
	PlanetarySchematic(ctx context.Context, schematicID int64) (*esiapi.UniverseSchematicsSchematicIdGet, error)
}

// --- list_character_planets (private) ---

type Planet struct {
	PlanetID        int64     `json:"planetId"`
	PlanetName      string    `json:"planetName,omitempty"`
	SolarSystemID   int64     `json:"solarSystemId"`
	SolarSystemName string    `json:"solarSystemName,omitempty"`
	PlanetType      string    `json:"planetType"`
	OwnerID         int64     `json:"ownerId"`
	UpgradeLevel    int64     `json:"upgradeLevel"`
	NumPins         int64     `json:"numPins"`
	LastUpdate      time.Time `json:"lastUpdate"`
}

func listCharacterPlanets(ctx context.Context, client planetsESI) ([]Planet, error) {
	planets, err := client.CharacterPlanets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Planet, len(planets))
	for i, p := range planets {
		out[i] = Planet{
			PlanetID: p.PlanetId, SolarSystemID: p.SolarSystemId, PlanetType: p.PlanetType,
			OwnerID: p.OwnerId, UpgradeLevel: p.UpgradeLevel, NumPins: p.NumPins, LastUpdate: p.LastUpdate,
		}
	}
	return out, nil
}

type ListCharacterPlanetsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterPlanetsOutput struct {
	CharacterID int32    `json:"characterId"`
	Planets     []Planet `json:"planets"`
}

// AddListCharacterPlanets registers list_character_planets. open builds the ESI client; any client with the
// planetsESI methods will do.
func AddListCharacterPlanets[C planetsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore colonyNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_planets",
		Description: "List a logged-in EVE Online character's colonized planets (Planetary Interaction).",
	}, listCharacterPlanetsHandler(userAgent, open, store, sdeStore))
}

func listCharacterPlanetsHandler[C planetsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore colonyNameCache) mcpsdk.ToolHandlerFor[ListCharacterPlanetsInput, ListCharacterPlanetsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterPlanetsInput) (*mcpsdk.CallToolResult, ListCharacterPlanetsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterPlanetsOutput{}, err
		}
		if !found {
			return nil, ListCharacterPlanetsOutput{}, fmt.Errorf("list character planets for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterPlanetsOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		planets, err := listCharacterPlanets(ctx, client)
		if err != nil {
			return nil, ListCharacterPlanetsOutput{}, fmt.Errorf("list character planets: %w", err)
		}

		systemIDs := make([]int64, len(planets))
		planetIDs := make([]int64, len(planets))
		for i, p := range planets {
			systemIDs[i] = p.SolarSystemID
			planetIDs[i] = p.PlanetID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		planetNames := resolvePlanetNames(ctx, sdeStore, userAgent, planetIDs)
		for i := range planets {
			planets[i].SolarSystemName = systemNames[planets[i].SolarSystemID]
			planets[i].PlanetName = planetNames[planets[i].PlanetID]
		}

		return nil, ListCharacterPlanetsOutput{CharacterID: in.CharacterID, Planets: planets}, nil
	}
}

// --- get_character_planet (private) ---
//
// The colony-layout response is this batch's most structurally complex model, but it is not
// recursive: ESI's own CharactersCharacterIdPlanetsPlanetIdGet has exactly 3 flat sibling
// slices -- Links, Pins, Routes -- and PlanetLink/PlanetPin/PlanetRoute below mirror that shape
// exactly, one distinct type per real nesting level (PlanetPin's ExtractorDetails/FactoryDetails/
// Contents go one level deeper; PlanetExtractorDetails' Heads go one level deeper than that).
// Links and Routes reference pins purely by int64 ID (SourcePinID/DestinationPinID/Waypoints),
// never by embedding another PlanetLink/PlanetPin/PlanetRoute value, so none of these types is
// self-referential -- no Killmails-style split into "outer"/"inner" variants of the same
// conceptual type is needed here, because goesi-openapi never generated a recursive shape for PI
// in the first place.

type PlanetLink struct {
	SourcePinID      int64 `json:"sourcePinId"`
	DestinationPinID int64 `json:"destinationPinId"`
	LinkLevel        int64 `json:"linkLevel"`
}

type PlanetPinContent struct {
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
	Amount   int64  `json:"amount"`
}

type PlanetExtractorHead struct {
	HeadID    int64   `json:"headId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PlanetExtractorDetails struct {
	CycleTime       *int64                `json:"cycleTime,omitempty"`
	HeadRadius      *float64              `json:"headRadius,omitempty"`
	ProductTypeID   *int64                `json:"productTypeId,omitempty"`
	ProductTypeName string                `json:"productTypeName,omitempty"`
	QtyPerCycle     *int64                `json:"qtyPerCycle,omitempty"`
	Heads           []PlanetExtractorHead `json:"heads"`
}

type PlanetFactoryDetails struct {
	SchematicID int64 `json:"schematicId"`
}

type PlanetPin struct {
	PinID            int64                   `json:"pinId"`
	TypeID           int64                   `json:"typeId"`
	TypeName         string                  `json:"typeName,omitempty"`
	Latitude         float64                 `json:"latitude"`
	Longitude        float64                 `json:"longitude"`
	SchematicID      *int64                  `json:"schematicId,omitempty"`
	ExpiryTime       *time.Time              `json:"expiryTime,omitempty"`
	InstallTime      *time.Time              `json:"installTime,omitempty"`
	LastCycleStart   *time.Time              `json:"lastCycleStart,omitempty"`
	Contents         []PlanetPinContent      `json:"contents,omitempty"`
	ExtractorDetails *PlanetExtractorDetails `json:"extractorDetails,omitempty"`
	FactoryDetails   *PlanetFactoryDetails   `json:"factoryDetails,omitempty"`
}

type PlanetRoute struct {
	RouteID          int64   `json:"routeId"`
	SourcePinID      int64   `json:"sourcePinId"`
	DestinationPinID int64   `json:"destinationPinId"`
	ContentTypeID    int64   `json:"contentTypeId"`
	ContentTypeName  string  `json:"contentTypeName,omitempty"`
	Quantity         float64 `json:"quantity"`
	Waypoints        []int64 `json:"waypoints,omitempty"`
}

func toPlanetPins(pins []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInner) []PlanetPin {
	out := make([]PlanetPin, len(pins))
	for i, p := range pins {
		pin := PlanetPin{
			PinID: p.PinId, TypeID: p.TypeId, Latitude: p.Latitude, Longitude: p.Longitude,
			SchematicID: p.SchematicId, ExpiryTime: p.ExpiryTime, InstallTime: p.InstallTime,
			LastCycleStart: p.LastCycleStart,
		}
		if len(p.Contents) > 0 {
			contents := make([]PlanetPinContent, len(p.Contents))
			for j, c := range p.Contents {
				contents[j] = PlanetPinContent{TypeID: c.TypeId, Amount: c.Amount}
			}
			pin.Contents = contents
		}
		if p.ExtractorDetails != nil {
			heads := make([]PlanetExtractorHead, len(p.ExtractorDetails.Heads))
			for j, h := range p.ExtractorDetails.Heads {
				heads[j] = PlanetExtractorHead{HeadID: h.HeadId, Latitude: h.Latitude, Longitude: h.Longitude}
			}
			pin.ExtractorDetails = &PlanetExtractorDetails{
				CycleTime: p.ExtractorDetails.CycleTime, HeadRadius: p.ExtractorDetails.HeadRadius,
				ProductTypeID: p.ExtractorDetails.ProductTypeId, QtyPerCycle: p.ExtractorDetails.QtyPerCycle,
				Heads: heads,
			}
		}
		if p.FactoryDetails != nil {
			pin.FactoryDetails = &PlanetFactoryDetails{SchematicID: p.FactoryDetails.SchematicId}
		}
		out[i] = pin
	}
	return out
}

func toPlanetLinks(links []esiapi.CharactersCharacterIdPlanetsPlanetIdGetLinksInner) []PlanetLink {
	out := make([]PlanetLink, len(links))
	for i, l := range links {
		out[i] = PlanetLink{SourcePinID: l.SourcePinId, DestinationPinID: l.DestinationPinId, LinkLevel: l.LinkLevel}
	}
	return out
}

func toPlanetRoutes(routes []esiapi.CharactersCharacterIdPlanetsPlanetIdGetRoutesInner) []PlanetRoute {
	out := make([]PlanetRoute, len(routes))
	for i, r := range routes {
		out[i] = PlanetRoute{
			RouteID: r.RouteId, SourcePinID: r.SourcePinId, DestinationPinID: r.DestinationPinId,
			ContentTypeID: r.ContentTypeId, Quantity: r.Quantity, Waypoints: r.Waypoints,
		}
	}
	return out
}

func getCharacterPlanet(ctx context.Context, client planetsESI, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
	return client.CharacterPlanetDetail(ctx, planetID)
}

type GetCharacterPlanetInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	PlanetID    int64 `json:"planetId" jsonschema:"the planet ID, from list_character_planets"`
}
type GetCharacterPlanetOutput struct {
	CharacterID int32         `json:"characterId"`
	PlanetID    int64         `json:"planetId"`
	Links       []PlanetLink  `json:"links"`
	Pins        []PlanetPin   `json:"pins"`
	Routes      []PlanetRoute `json:"routes"`
}

// AddGetCharacterPlanet registers get_character_planet. open builds the ESI client; any client with the
// planetsESI methods will do.
func AddGetCharacterPlanet[C planetsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_planet",
		Description: "Get the full colony layout (links, pins, routes) of one of a character's planets.",
	}, getCharacterPlanetHandler(userAgent, open, store, sdeStore))
}

func getCharacterPlanetHandler[C planetsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[GetCharacterPlanetInput, GetCharacterPlanetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterPlanetInput) (*mcpsdk.CallToolResult, GetCharacterPlanetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterPlanetOutput{}, err
		}
		if !found {
			return nil, GetCharacterPlanetOutput{}, fmt.Errorf("get character planet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterPlanetOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		detail, err := getCharacterPlanet(ctx, client, in.PlanetID)
		if err != nil {
			return nil, GetCharacterPlanetOutput{}, fmt.Errorf("get character planet: %w", err)
		}
		pins := toPlanetPins(detail.Pins)
		routes := toPlanetRoutes(detail.Routes)

		ids := make([]int64, 0, len(pins)+len(routes))
		for _, p := range pins {
			ids = append(ids, p.TypeID)
			for _, c := range p.Contents {
				ids = append(ids, c.TypeID)
			}
			if p.ExtractorDetails != nil && p.ExtractorDetails.ProductTypeID != nil {
				ids = append(ids, *p.ExtractorDetails.ProductTypeID)
			}
		}
		for _, r := range routes {
			ids = append(ids, r.ContentTypeID)
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range pins {
			pins[i].TypeName = names[pins[i].TypeID]
			for j := range pins[i].Contents {
				pins[i].Contents[j].TypeName = names[pins[i].Contents[j].TypeID]
			}
			if pins[i].ExtractorDetails != nil && pins[i].ExtractorDetails.ProductTypeID != nil {
				pins[i].ExtractorDetails.ProductTypeName = names[*pins[i].ExtractorDetails.ProductTypeID]
			}
		}
		for i := range routes {
			routes[i].ContentTypeName = names[routes[i].ContentTypeID]
		}

		return nil, GetCharacterPlanetOutput{
			CharacterID: in.CharacterID, PlanetID: in.PlanetID,
			Links: toPlanetLinks(detail.Links), Pins: pins, Routes: routes,
		}, nil
	}
}

// --- list_corporation_customs_offices (private, corp-scoped, paginated) ---

type CustomsOffice struct {
	OfficeID                 int64    `json:"officeId"`
	SystemID                 int64    `json:"systemId"`
	SystemName               string   `json:"systemName,omitempty"`
	TypeID                   *int64   `json:"typeId,omitempty"`
	TypeName                 string   `json:"typeName,omitempty"`
	ReinforceExitStart       int64    `json:"reinforceExitStart"`
	ReinforceExitEnd         int64    `json:"reinforceExitEnd"`
	AllowAllianceAccess      bool     `json:"allowAllianceAccess"`
	AllowAccessWithStandings bool     `json:"allowAccessWithStandings"`
	StandingLevel            *string  `json:"standingLevel,omitempty"`
	AllianceTaxRate          *float64 `json:"allianceTaxRate,omitempty"`
	CorporationTaxRate       *float64 `json:"corporationTaxRate,omitempty"`
	ExcellentStandingTaxRate *float64 `json:"excellentStandingTaxRate,omitempty"`
	GoodStandingTaxRate      *float64 `json:"goodStandingTaxRate,omitempty"`
	NeutralStandingTaxRate   *float64 `json:"neutralStandingTaxRate,omitempty"`
	BadStandingTaxRate       *float64 `json:"badStandingTaxRate,omitempty"`
	TerribleStandingTaxRate  *float64 `json:"terribleStandingTaxRate,omitempty"`
}

func listCorporationCustomsOffices(ctx context.Context, client planetsESI, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
	return client.CorporationCustomsOffices(ctx, corporationID, page)
}

type ListCorporationCustomsOfficesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationCustomsOfficesOutput struct {
	CharacterID   int32           `json:"characterId"`
	CorporationID int64           `json:"corporationId"`
	Page          int32           `json:"page"`
	TotalPages    int32           `json:"totalPages"`
	Offices       []CustomsOffice `json:"offices"`
}

// AddListCorporationCustomsOffices registers list_corporation_customs_offices. open builds the ESI client; any client with the
// planetsESI methods will do.
func AddListCorporationCustomsOffices[C planetsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore customsOfficeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_customs_offices",
		Description: "List a corporation's customs offices (POCOs), including tax rates and reinforcement schedule.",
	}, listCorporationCustomsOfficesHandler(userAgent, open, store, sdeStore))
}

func listCorporationCustomsOfficesHandler[C planetsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore customsOfficeNameCache) mcpsdk.ToolHandlerFor[ListCorporationCustomsOfficesInput, ListCorporationCustomsOfficesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationCustomsOfficesInput) (*mcpsdk.CallToolResult, ListCorporationCustomsOfficesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationCustomsOfficesOutput{}, err
		}
		if !found {
			return nil, ListCorporationCustomsOfficesOutput{}, fmt.Errorf("list corporation customs offices for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationCustomsOfficesOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		offices, totalPages, err := listCorporationCustomsOffices(ctx, client, in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationCustomsOfficesOutput{}, fmt.Errorf("list corporation customs offices: %w", err)
		}
		out := make([]CustomsOffice, len(offices))
		for i, o := range offices {
			out[i] = CustomsOffice{
				OfficeID: o.OfficeId, SystemID: o.SystemId, TypeID: o.TypeId,
				ReinforceExitStart: o.ReinforceExitStart, ReinforceExitEnd: o.ReinforceExitEnd,
				AllowAllianceAccess: o.AllowAllianceAccess, AllowAccessWithStandings: o.AllowAccessWithStandings,
				StandingLevel: o.StandingLevel, AllianceTaxRate: o.AllianceTaxRate,
				CorporationTaxRate: o.CorporationTaxRate, ExcellentStandingTaxRate: o.ExcellentStandingTaxRate,
				GoodStandingTaxRate: o.GoodStandingTaxRate, NeutralStandingTaxRate: o.NeutralStandingTaxRate,
				BadStandingTaxRate: o.BadStandingTaxRate, TerribleStandingTaxRate: o.TerribleStandingTaxRate,
			}
		}

		ids := make([]int64, 0, len(out))
		for _, o := range out {
			if o.TypeID != nil {
				ids = append(ids, *o.TypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range out {
			if out[i].TypeID != nil {
				out[i].TypeName = names[*out[i].TypeID]
			}
		}

		systemIDs := make([]int64, len(out))
		for i, o := range out {
			systemIDs[i] = o.SystemID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		for i := range out {
			out[i].SystemName = systemNames[out[i].SystemID]
		}

		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationCustomsOfficesOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Offices: out,
		}, nil
	}
}

// --- get_planetary_schematic (public) ---

type GetPlanetarySchematicOutput struct {
	SchematicID   int64  `json:"schematicId"`
	SchematicName string `json:"schematicName"`
	CycleTime     int64  `json:"cycleTime"`
}

func getPlanetarySchematic(ctx context.Context, client planetarySchematicESI, schematicID int64) (GetPlanetarySchematicOutput, error) {
	schematic, err := client.PlanetarySchematic(ctx, schematicID)
	if err != nil {
		return GetPlanetarySchematicOutput{}, err
	}
	return GetPlanetarySchematicOutput{SchematicID: schematicID, SchematicName: schematic.SchematicName, CycleTime: schematic.CycleTime}, nil
}

type GetPlanetarySchematicInput struct {
	SchematicID int64 `json:"schematicId" jsonschema:"the PI schematic ID"`
}

// AddGetPlanetarySchematic registers get_planetary_schematic. open builds the ESI client; any client with the
// planetarySchematicESI methods will do.
func AddGetPlanetarySchematic[C planetarySchematicESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_planetary_schematic",
		Description: "Get a Planetary Interaction schematic's cycle time and name.",
	}, getPlanetarySchematicHandler(open))
}

func getPlanetarySchematicHandler[C planetarySchematicESI](open func() C) mcpsdk.ToolHandlerFor[GetPlanetarySchematicInput, GetPlanetarySchematicOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetPlanetarySchematicInput) (*mcpsdk.CallToolResult, GetPlanetarySchematicOutput, error) {
		out, err := getPlanetarySchematic(ctx, open(), in.SchematicID)
		if err != nil {
			return nil, GetPlanetarySchematicOutput{}, fmt.Errorf("get planetary schematic: %w", err)
		}
		return nil, out, nil
	}
}
