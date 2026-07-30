package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_character_location (private) ---

var getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func getCharacterLocationHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCharacterLocationInput, GetCharacterLocationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterLocationInput) (*mcpsdk.CallToolResult, GetCharacterLocationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterLocationOutput{}, err
		}
		if !found {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		solarSystemID, stationID, structureID, err := getCharacterLocation(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location: %w", err)
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), []int64{solarSystemID})
		var stationName string
		if stationID != nil {
			stationNames := resolveStationNames(ctx, sdeStore, sso.UserAgent(version), []int64{*stationID})
			stationName = stationNames[*stationID]
		}
		return nil, GetCharacterLocationOutput{CharacterID: in.CharacterID, SolarSystemID: solarSystemID, SolarSystemName: systemNames[solarSystemID], StationID: stationID, StationName: stationName, StructureID: structureID}, nil
	}
}

// --- get_character_online (private) ---

var getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func getCharacterOnlineHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterOnlineInput, GetCharacterOnlineOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterOnlineInput) (*mcpsdk.CallToolResult, GetCharacterOnlineOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, err
		}
		if !found {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		status, err := getCharacterOnline(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status: %w", err)
		}
		return nil, GetCharacterOnlineOutput{CharacterID: in.CharacterID, Online: status.Online, LastLogin: status.LastLogin, LastLogout: status.LastLogout, Logins: status.Logins}, nil
	}
}

// --- get_character_ship (private) ---

var getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, "", 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func getCharacterShipHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCharacterShipInput, GetCharacterShipOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterShipInput) (*mcpsdk.CallToolResult, GetCharacterShipOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterShipOutput{}, err
		}
		if !found {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		shipItemID, shipName, shipTypeID, err := getCharacterShip(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship: %w", err)
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{shipTypeID})
		return nil, GetCharacterShipOutput{CharacterID: in.CharacterID, ShipItemID: shipItemID, ShipName: shipName, ShipTypeID: shipTypeID, ShipTypeName: names[shipTypeID]}, nil
	}
}
