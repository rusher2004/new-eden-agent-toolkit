package mcp

import (
	"context"
	"errors"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// ErrCharacterNotFound is returned (wrapped) when remove_character is
// called with a character ID that isn't in the local config.
var ErrCharacterNotFound = errors.New("character not found")

// removeCharacterCreds is overridable in tests. It deletes the character's
// refresh token from the keychain and evicts its cached authenticated
// client atomically -- see sso.RemoveCharacter's doc comment.
var removeCharacterCreds = sso.RemoveCharacter

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

		if err := removeCharacterCreds(removed.ID); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("remove character credentials: %w", err)
		}

		cfg.Characters = append(cfg.Characters[:idx], cfg.Characters[idx+1:]...)
		if err := store.Save(cfg); err != nil {
			return nil, CharacterInfo{}, fmt.Errorf("save config (character's credentials were already removed): %w", err)
		}

		return nil, characterInfo(removed), nil
	}
}
