package mcp

import (
	"context"
	"errors"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

// ErrCharacterNotFound is returned (wrapped) when remove_character is
// called with a character ID that isn't in the local config.
var ErrCharacterNotFound = errors.New("character not found")

// RemoveCharacterInput is remove_character's tool input.
type RemoveCharacterInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to remove"`
}

func removeCharacterHandler(store *config.Store) mcpsdk.ToolHandlerFor[RemoveCharacterInput, CharacterInfo] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RemoveCharacterInput) (*mcpsdk.CallToolResult, CharacterInfo, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("load config: %w", err)
		}

		idx := -1
		for i, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, CharacterInfo{}, fmt.Errorf("remove character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		removed := cfg.Characters[idx]

		if err := creds.DeleteCharacterRefreshToken(removed.ID); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("delete refresh token: %w", err)
		}

		cfg.Characters = append(cfg.Characters[:idx], cfg.Characters[idx+1:]...)
		if err := store.Save(cfg); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("save config: %w", err)
		}

		return nil, CharacterInfo{CharacterID: removed.ID, CharacterName: removed.Name}, nil
	}
}
