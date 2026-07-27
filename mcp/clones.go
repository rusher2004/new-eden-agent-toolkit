package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

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

// getCharacterClones is overridable in tests.
var getCharacterClones = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterClonesOutput, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return GetCharacterClonesOutput{}, fmt.Errorf("build ESI client: %w", err)
	}
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

func getCharacterClonesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterClonesInput, GetCharacterClonesOutput] {
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
		out, err := getCharacterClones(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterClonesOutput{}, fmt.Errorf("get character clones: %w", err)
		}
		return nil, out, nil
	}
}

// listCharacterImplants is overridable in tests.
var listCharacterImplants = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCharacterImplantsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterImplantsInput, ListCharacterImplantsOutput] {
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
		ids, err := listCharacterImplants(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterImplantsOutput{}, fmt.Errorf("list character implants: %w", err)
		}
		return nil, ListCharacterImplantsOutput{CharacterID: in.CharacterID, ImplantTypeIDs: ids}, nil
	}
}
