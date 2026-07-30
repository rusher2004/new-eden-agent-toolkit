package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- search_character (private) ---

type SearchResults struct {
	Agent              []int64  `json:"agent,omitempty"`
	Alliance           []int64  `json:"alliance,omitempty"`
	Character          []int64  `json:"character,omitempty"`
	Constellation      []int64  `json:"constellation,omitempty"`
	Corporation        []int64  `json:"corporation,omitempty"`
	Faction            []int64  `json:"faction,omitempty"`
	InventoryType      []int64  `json:"inventoryType,omitempty"`
	InventoryTypeNames []string `json:"inventoryTypeNames,omitempty"`
	Region             []int64  `json:"region,omitempty"`
	SolarSystem        []int64  `json:"solarSystem,omitempty"`
	Station            []int64  `json:"station,omitempty"`
	Structure          []int64  `json:"structure,omitempty"`
}

var searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterSearch(ctx, categories, search, strict)
}

type SearchCharacterInput struct {
	CharacterID int32    `json:"characterId" jsonschema:"the character ID to check"`
	Categories  []string `json:"categories" jsonschema:"1-11 of: agent, alliance, character, constellation, corporation, faction, inventory_type, region, solar_system, station, structure"`
	Search      string   `json:"search" jsonschema:"the search string, at least 3 characters"`
	Strict      bool     `json:"strict,omitempty" jsonschema:"require an exact match instead of a substring match; omit for false"`
}
type SearchCharacterOutput struct {
	CharacterID int32         `json:"characterId"`
	Results     SearchResults `json:"results"`
}

func searchCharacterHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[SearchCharacterInput, SearchCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SearchCharacterInput) (*mcpsdk.CallToolResult, SearchCharacterOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SearchCharacterOutput{}, err
		}
		if !found {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		results, err := searchCharacter(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Categories, in.Search, in.Strict)
		if err != nil {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character: %w", err)
		}
		nameMap := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), results.InventoryType)
		inventoryTypeNames := make([]string, len(results.InventoryType))
		for i, id := range results.InventoryType {
			inventoryTypeNames[i] = nameMap[id]
		}
		return nil, SearchCharacterOutput{CharacterID: in.CharacterID, Results: SearchResults{
			Agent: results.Agent, Alliance: results.Alliance, Character: results.Character,
			Constellation: results.Constellation, Corporation: results.Corporation, Faction: results.Faction,
			InventoryType: results.InventoryType, InventoryTypeNames: inventoryTypeNames, Region: results.Region, SolarSystem: results.SolarSystem,
			Station: results.Station, Structure: results.Structure,
		}}, nil
	}
}
