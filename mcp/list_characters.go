package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// configLoader is the one config call most tools make: read the saved config,
// usually to check that a character is logged in. The tool files share it;
// ones that write declare their own interface.
type configLoader interface {
	Load() (config.Config, error)
}

// CharacterInfo is the MCP-facing representation of a known character,
// deliberately distinct from config.Character so the tool contract doesn't
// leak the internal storage schema.
type CharacterInfo struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
	Account       string `json:"account,omitempty" jsonschema:"the user's label for the EVE account this character is on; characters with the same label share an account and can't be logged in at the same time; absent means unknown"`
	Notes         string `json:"notes,omitempty" jsonschema:"what the user has said about this character"`
}

func characterInfo(c config.Character) CharacterInfo {
	return CharacterInfo{CharacterID: c.ID, CharacterName: c.Name, Account: c.Account, Notes: c.Notes}
}

// ListCharactersOutput is list_characters' tool result.
type ListCharactersOutput struct {
	Characters []CharacterInfo `json:"characters"`
	Warnings   []string        `json:"warnings,omitempty"`
}

func listCharactersHandler(store configLoader) mcpsdk.ToolHandlerFor[any, ListCharactersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, ListCharactersOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCharactersOutput{}, fmt.Errorf("load config: %w", err)
		}

		out := ListCharactersOutput{Characters: make([]CharacterInfo, 0, len(cfg.Characters)), Warnings: accountWarnings(cfg)}
		for _, c := range cfg.Characters {
			out.Characters = append(out.Characters, characterInfo(c))
		}
		return nil, out, nil
	}
}

// accountWarnings asks the agent to find out which account each unlabelled
// character is on, since plans that need two characters online at once
// depend on it and ESI can't tell.
func accountWarnings(cfg config.Config) []string {
	missing := cfg.UnlabelledCharacters()
	if len(missing) == 0 {
		return nil
	}
	names := make([]string, len(missing))
	for i, c := range missing {
		names[i] = fmt.Sprintf("%s (%d)", c.Name, c.ID)
	}
	labels := "none yet"
	if l := cfg.AccountLabels(); len(l) > 0 {
		labels = strings.Join(l, ", ")
	}
	return []string{fmt.Sprintf("Unknown EVE account for %s. Characters on the same account can't be logged in at the same time, so plans that need two characters online together (a jump pilot and their cyno lighter, say) depend on it, and ESI can't tell. Ask the user which of their characters share an account and save it with update_character_details, or point them to the account groups on neat's login page. Account labels in use: %s.", strings.Join(names, ", "), labels)}
}
