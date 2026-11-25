package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// characterClonesESI is the ESI call get_character_clones makes; the
// planner family reads clones through it too.
type characterClonesESI interface {
	CharacterClones(ctx context.Context) (*esiapi.CharactersCharacterIdClonesGet, error)
}

// characterImplantsESI is the ESI call list_character_implants makes; the
// planner family reads implants through it too.
type characterImplantsESI interface {
	CharacterImplants(ctx context.Context) ([]int64, error)
}

// JumpClone is one of a character's jump clones.
type JumpClone struct {
	JumpCloneID  int64   `json:"jumpCloneId"`
	LocationID   int64   `json:"locationId"`
	LocationType string  `json:"locationType"`
	Name         *string `json:"name,omitempty"`
	Implants     []int64 `json:"implants"`
}

// GetCharacterClonesOutput is get_character_clones's tool result.
type GetCharacterClonesOutput struct {
	CharacterID           int32       `json:"characterId"`
	HomeLocationID        *int64      `json:"homeLocationId,omitempty"`
	HomeLocationType      *string     `json:"homeLocationType,omitempty"`
	LastCloneJumpDate     *time.Time  `json:"lastCloneJumpDate,omitempty"`
	LastStationChangeDate *time.Time  `json:"lastStationChangeDate,omitempty"`
	JumpClones            []JumpClone `json:"jumpClones"`
}

// getCharacterClones reads characterID's clones through their client.
func getCharacterClones(ctx context.Context, client characterClonesESI, characterID int32) (GetCharacterClonesOutput, error) {
	clones, err := client.CharacterClones(ctx)
	if err != nil {
		return GetCharacterClonesOutput{}, err
	}
	out := GetCharacterClonesOutput{
		CharacterID:           characterID,
		LastCloneJumpDate:     clones.LastCloneJumpDate,
		LastStationChangeDate: clones.LastStationChangeDate,
	}
	if clones.HomeLocation != nil {
		out.HomeLocationID = clones.HomeLocation.LocationId
		out.HomeLocationType = clones.HomeLocation.LocationType
	}
	out.JumpClones = make([]JumpClone, len(clones.JumpClones))
	for i, jc := range clones.JumpClones {
		out.JumpClones[i] = JumpClone{JumpCloneID: jc.JumpCloneId, LocationID: jc.LocationId, LocationType: jc.LocationType, Name: jc.Name, Implants: jc.Implants}
	}
	return out, nil
}

// GetCharacterClonesInput is get_character_clones's tool input.
type GetCharacterClonesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// AddGetCharacterClones registers get_character_clones. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterClones[C characterClonesESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_clones",
		Description: "Get a logged-in EVE Online character's jump clones and home location.",
	}, getCharacterClonesHandler(open, store))
}

func getCharacterClonesHandler[C characterClonesESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterClonesInput, GetCharacterClonesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterClonesInput) (*mcpsdk.CallToolResult, GetCharacterClonesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("get character clones for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("get character clones: %w", err)
		}
		out, err := getCharacterClones(ctx, client, in.CharacterID)
		if err != nil {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("get character clones: %w", err)
		}
		return nil, out, nil
	}
}

// listCharacterImplants reads the implant type IDs in a character's current
// clone.
func listCharacterImplants(ctx context.Context, client characterImplantsESI) ([]int64, error) {
	return client.CharacterImplants(ctx)
}

// ListCharacterImplantsInput is list_character_implants's tool input.
type ListCharacterImplantsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListCharacterImplantsOutput is list_character_implants's tool result.
type ListCharacterImplantsOutput struct {
	CharacterID    int32   `json:"characterId"`
	ImplantTypeIDs []int64 `json:"implantTypeIds"`
}

// AddListCharacterImplants registers list_character_implants. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterImplants[C characterImplantsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_implants",
		Description: "List a logged-in EVE Online character's currently active implant type IDs.",
	}, listCharacterImplantsHandler(open, store))
}

func listCharacterImplantsHandler[C characterImplantsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterImplantsInput, ListCharacterImplantsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterImplantsInput) (*mcpsdk.CallToolResult, ListCharacterImplantsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("list character implants for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("list character implants: %w", err)
		}
		ids, err := listCharacterImplants(ctx, client)
		if err != nil {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("list character implants: %w", err)
		}
		return nil, ListCharacterImplantsOutput{CharacterID: in.CharacterID, ImplantTypeIDs: ids}, nil
	}
}
