package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// CharacterInfo is the MCP-facing representation of a known character,
// deliberately distinct from config.Character so the tool contract doesn't
// leak the internal storage schema.
type CharacterInfo struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
}

// ListCharactersOutput is list_characters' tool result.
type ListCharactersOutput struct {
	Characters []CharacterInfo `json:"characters"`
}

func listCharactersHandler(store *config.Store) mcpsdk.ToolHandlerFor[any, ListCharactersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, ListCharactersOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCharactersOutput{}, fmt.Errorf("load config: %w", err)
		}

		out := ListCharactersOutput{Characters: make([]CharacterInfo, 0, len(cfg.Characters))}
		for _, c := range cfg.Characters {
			out.Characters = append(out.Characters, CharacterInfo{CharacterID: c.ID, CharacterName: c.Name})
		}
		return nil, out, nil
	}
}
