package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// searchESI is the ESI call search_character makes.
type searchESI interface {
	CharacterSearch(ctx context.Context, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error)
}

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

func searchCharacter(ctx context.Context, client searchESI, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
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

// AddSearchCharacter registers search_character. open builds the ESI client; any client with the
// searchESI methods will do.
func AddSearchCharacter[C searchESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "search_character",
		Description: "Search for entities (characters, corporations, systems, etc.) matching a string, scoped to a logged-in character's search permissions.",
	}, searchCharacterHandler(userAgent, open, store, sdeStore))
}

func searchCharacterHandler[C searchESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[SearchCharacterInput, SearchCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SearchCharacterInput) (*mcpsdk.CallToolResult, SearchCharacterOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SearchCharacterOutput{}, err
		}
		if !found {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, SearchCharacterOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		results, err := searchCharacter(ctx, client, in.Categories, in.Search, in.Strict)
		if err != nil {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character: %w", err)
		}
		nameMap := resolveTypeNames(ctx, sdeStore, userAgent, results.InventoryType)
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
