package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

// characterLocationESI is the ESI call get_character_location makes; the
// planner family reads locations through it too.
type characterLocationESI interface {
	CharacterLocation(ctx context.Context) (*esiapi.CharactersLocation, error)
}

// characterOnlineESI is the ESI call get_character_online makes; the
// logistics overview reads online status through it too.
type characterOnlineESI interface {
	CharacterOnline(ctx context.Context) (*esi.CharacterOnlineStatus, error)
}

// characterShipESI is the ESI call get_character_ship makes; the logistics
// overview reads ships through it too.
type characterShipESI interface {
	CharacterShip(ctx context.Context) (*esiapi.CharactersShip, error)
}

// locationNameCache is the SDE lookups get_character_location names its
// system and station with.
type locationNameCache interface {
	systemNameCache
	stationNameCache
}

// --- get_character_location (private) ---

func getCharacterLocation(ctx context.Context, client characterLocationESI) (int64, *int64, *int64, error) {
	loc, err := client.CharacterLocation(ctx)
	if err != nil {
		return 0, nil, nil, err
	}
	return loc.SolarSystemId, loc.StationId, loc.StructureId, nil
}

type GetCharacterLocationInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterLocationOutput struct {
	CharacterID     int32  `json:"characterId"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
	StationID       *int64 `json:"stationId,omitempty"`
	StationName     string `json:"stationName,omitempty"`
	StructureID     *int64 `json:"structureId,omitempty"`
}

// AddGetCharacterLocation registers get_character_location. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterLocation[C characterLocationESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore locationNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_location",
		Description: "Get a logged-in EVE Online character's current solar system, and station or structure ID if docked/boarded.",
	}, getCharacterLocationHandler(userAgent, open, store, sdeStore))
}

func getCharacterLocationHandler[C characterLocationESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore locationNameCache) mcpsdk.ToolHandlerFor[GetCharacterLocationInput, GetCharacterLocationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterLocationInput) (*mcpsdk.CallToolResult, GetCharacterLocationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterLocationOutput{}, err
		}
		if !found {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location: %w", err)
		}
		solarSystemID, stationID, structureID, err := getCharacterLocation(ctx, client)
		if err != nil {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location: %w", err)
		}
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, []int64{solarSystemID})
		var stationName string
		if stationID != nil {
			stationNames := resolveStationNames(ctx, sdeStore, userAgent, []int64{*stationID})
			stationName = stationNames[*stationID]
		}
		return nil, GetCharacterLocationOutput{CharacterID: in.CharacterID, SolarSystemID: solarSystemID, SolarSystemName: systemNames[solarSystemID], StationID: stationID, StationName: stationName, StructureID: structureID}, nil
	}
}

// --- get_character_online (private) ---

func getCharacterOnline(ctx context.Context, client characterOnlineESI) (*esi.CharacterOnlineStatus, error) {
	return client.CharacterOnline(ctx)
}

type GetCharacterOnlineInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterOnlineOutput struct {
	CharacterID int32      `json:"characterId"`
	Online      bool       `json:"online"`
	LastLogin   *time.Time `json:"lastLogin,omitempty"`
	LastLogout  *time.Time `json:"lastLogout,omitempty"`
	Logins      *int64     `json:"logins,omitempty"`
}

// AddGetCharacterOnline registers get_character_online. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterOnline[C characterOnlineESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_online",
		Description: "Get a logged-in EVE Online character's online status and login history.",
	}, getCharacterOnlineHandler(open, store))
}

func getCharacterOnlineHandler[C characterOnlineESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterOnlineInput, GetCharacterOnlineOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterOnlineInput) (*mcpsdk.CallToolResult, GetCharacterOnlineOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, err
		}
		if !found {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status: open ESI client: %w", err)
		}
		status, err := getCharacterOnline(ctx, client)
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status: %w", err)
		}
		return nil, GetCharacterOnlineOutput{CharacterID: in.CharacterID, Online: status.Online, LastLogin: status.LastLogin, LastLogout: status.LastLogout, Logins: status.Logins}, nil
	}
}

// --- get_character_ship (private) ---

func getCharacterShip(ctx context.Context, client characterShipESI) (int64, string, int64, error) {
	ship, err := client.CharacterShip(ctx)
	if err != nil {
		return 0, "", 0, err
	}
	return ship.ShipItemId, ship.ShipName, ship.ShipTypeId, nil
}

type GetCharacterShipInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterShipOutput struct {
	CharacterID  int32  `json:"characterId"`
	ShipItemID   int64  `json:"shipItemId"`
	ShipName     string `json:"shipName"`
	ShipTypeID   int64  `json:"shipTypeId"`
	ShipTypeName string `json:"shipTypeName,omitempty"`
}

// AddGetCharacterShip registers get_character_ship. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterShip[C characterShipESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_ship",
		Description: "Get a logged-in EVE Online character's current ship.",
	}, getCharacterShipHandler(userAgent, open, store, sdeStore))
}

func getCharacterShipHandler[C characterShipESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[GetCharacterShipInput, GetCharacterShipOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterShipInput) (*mcpsdk.CallToolResult, GetCharacterShipOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterShipOutput{}, err
		}
		if !found {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship: %w", err)
		}
		shipItemID, shipName, shipTypeID, err := getCharacterShip(ctx, client)
		if err != nil {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship: %w", err)
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, []int64{shipTypeID})
		return nil, GetCharacterShipOutput{CharacterID: in.CharacterID, ShipItemID: shipItemID, ShipName: shipName, ShipTypeID: shipTypeID, ShipTypeName: names[shipTypeID]}, nil
	}
}
