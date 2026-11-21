package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// AccountRename renames an account label for every character that has it.
type AccountRename struct {
	From string `json:"from" jsonschema:"the account label in use now"`
	To   string `json:"to" jsonschema:"the new label; renaming onto a label already in use merges the two accounts"`
}

// UpdateCharacterDetailsInput is update_character_details' input.
type UpdateCharacterDetailsInput struct {
	CharacterIDs  []int32        `json:"characterIds,omitempty" jsonschema:"the characters to update, from list_characters"`
	Account       *string        `json:"account,omitempty" jsonschema:"account label for every character in characterIds: characters with the same label share an EVE account. Any short name the user recognizes, such as 'Main' or 'Account 2'. An empty string clears it, which means unknown"`
	Notes         *string        `json:"notes,omitempty" jsonschema:"replaces the notes of the one character in characterIds: what the user says the character is for or can do. An empty string clears them"`
	RenameAccount *AccountRename `json:"renameAccount,omitempty" jsonschema:"rename an account label for every character that has it"`
}

// FieldChange is one value an update changed.
type FieldChange struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
	Field         string `json:"field"`
	Old           string `json:"old"`
	New           string `json:"new"`
}

// UpdateCharacterDetailsOutput reports every value changed and the
// characters as saved afterwards.
type UpdateCharacterDetailsOutput struct {
	Changes    []FieldChange   `json:"changes"`
	Characters []CharacterInfo `json:"characters"`
	Warnings   []string        `json:"warnings,omitempty"`
}

func updateCharacterDetailsHandler(store *config.Store) mcpsdk.ToolHandlerFor[UpdateCharacterDetailsInput, UpdateCharacterDetailsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateCharacterDetailsInput) (*mcpsdk.CallToolResult, UpdateCharacterDetailsOutput, error) {
		if in.Account == nil && in.Notes == nil && in.RenameAccount == nil {
			return nil, UpdateCharacterDetailsOutput{}, fmt.Errorf("update character details: give account, notes or renameAccount")
		}
		if (in.Account != nil || in.Notes != nil) && len(in.CharacterIDs) == 0 {
			return nil, UpdateCharacterDetailsOutput{}, fmt.Errorf("update character details: account and notes need characterIds")
		}
		if in.Notes != nil && len(in.CharacterIDs) != 1 {
			return nil, UpdateCharacterDetailsOutput{}, fmt.Errorf("update character details: notes belong to one character, but %d were given", len(in.CharacterIDs))
		}
		var cfg config.Config
		before := map[int32]config.Character{}
		if _, err := store.Update(func(c *config.Config) error {
			for _, ch := range c.Characters {
				before[ch.ID] = ch
			}
			for _, id := range in.CharacterIDs {
				if _, ok := before[id]; !ok {
					return fmt.Errorf("update character %d: %w", id, ErrCharacterNotFound)
				}
			}
			if r := in.RenameAccount; r != nil {
				if _, err := c.RenameAccount(r.From, r.To); err != nil {
					return fmt.Errorf("rename account: %w", err)
				}
			}
			for _, id := range in.CharacterIDs {
				if in.Account != nil {
					if _, err := c.SetCharacterAccount(id, *in.Account); err != nil {
						return fmt.Errorf("set account: %w", err)
					}
				}
				if in.Notes != nil {
					if _, err := c.SetCharacterNotes(id, *in.Notes); err != nil {
						return fmt.Errorf("set notes: %w", err)
					}
				}
			}
			cfg = *c
			return nil
		}); err != nil {
			return nil, UpdateCharacterDetailsOutput{}, err
		}

		out := UpdateCharacterDetailsOutput{Changes: []FieldChange{}, Characters: make([]CharacterInfo, 0, len(cfg.Characters)), Warnings: accountWarnings(cfg)}
		for _, c := range cfg.Characters {
			out.Characters = append(out.Characters, characterInfo(c))
			old := before[c.ID]
			if old.Account != c.Account {
				out.Changes = append(out.Changes, FieldChange{CharacterID: c.ID, CharacterName: c.Name, Field: "account", Old: old.Account, New: c.Account})
			}
			if old.Notes != c.Notes {
				out.Changes = append(out.Changes, FieldChange{CharacterID: c.ID, CharacterName: c.Name, Field: "notes", Old: old.Notes, New: c.Notes})
			}
		}
		return nil, out, nil
	}
}
