package mcp

import (
	"context"
	"errors"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// ErrCharacterNotFound is returned (wrapped) when remove_character is
// called with a character ID that isn't in the local config.
var ErrCharacterNotFound = errors.New("character not found")

// RemoveCharacterInput is remove_character's tool input.
type RemoveCharacterInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to remove"`
}

// AddRemoveCharacter registers remove_character. removeCreds deletes a
// character's refresh token from the keychain and evicts its cached
// authenticated client atomically; the composition root passes the login
// flow's own function for it.
func AddRemoveCharacter(s *mcpsdk.Server, store configUpdater, removeCreds func(characterID int32) error) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "remove_character",
		Description: "Remove a logged-in EVE Online character, deleting its stored credentials.",
	}, removeCharacterHandler(store, removeCreds))
}

func removeCharacterHandler(store configUpdater, removeCreds func(characterID int32) error) mcpsdk.ToolHandlerFor[RemoveCharacterInput, CharacterInfo] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RemoveCharacterInput) (*mcpsdk.CallToolResult, CharacterInfo, error) {
		var removed config.Character
		credsRemoved := false
		if _, err := store.Update(func(cfg *config.Config) error {
			idx := -1
			for i, c := range cfg.Characters {
				if c.ID == in.CharacterID {
					idx = i
					break
				}
			}
			if idx == -1 {
				return fmt.Errorf("remove character %d: %w", in.CharacterID, ErrCharacterNotFound)
			}
			removed = cfg.Characters[idx]
			if err := removeCreds(removed.ID); err != nil {
				return fmt.Errorf("remove character credentials: %w", err)
			}
			credsRemoved = true
			cfg.Characters = append(cfg.Characters[:idx], cfg.Characters[idx+1:]...)
			return nil
		}); err != nil {
			if credsRemoved {
				return nil, CharacterInfo{}, fmt.Errorf("save config (character's credentials were already removed): %w", err)
			}
			return nil, CharacterInfo{}, err
		}

		return nil, characterInfo(removed), nil
	}
}
