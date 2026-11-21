package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// addCharacters and grantedScopes are overridable in tests so no real
// browser/network/OAuth2 flow runs during unit tests.
var (
	addCharacters = sso.AddCharacters
	grantedScopes = sso.GrantedScopes
)

// AddCharacterInput is add_character's input.
type AddCharacterInput struct {
	ScopeGroups []string `json:"scopeGroups,omitempty" jsonschema:"scope groups to grant beyond the always-included basic group, by name from list_scope_groups; pick them from what the user says they'll use neat for, and leave empty if they don't know"`
	CharacterID int32    `json:"characterId,omitempty" jsonschema:"set when re-authorizing an existing character to grant more scope groups: its currently granted scopes are kept and scopeGroups added to them"`
}

// AddedCharacter is one character added by add_character.
type AddedCharacter struct {
	CharacterID   int32    `json:"characterId"`
	CharacterName string   `json:"characterName"`
	ScopeGroups   []string `json:"scopeGroups"`
	Account       string   `json:"account,omitempty" jsonschema:"the account label the user gave this character, if any"`
}

// AddCharacterOutput is add_character's tool result: every character the
// user logged in during the browser session, in the order they were added,
// and warnings about a re-authorization that didn't go as asked.
type AddCharacterOutput struct {
	Characters []AddedCharacter `json:"characters"`
	Warnings   []string         `json:"warnings,omitempty"`
}

func addCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[AddCharacterInput, AddCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in AddCharacterInput) (*mcpsdk.CallToolResult, AddCharacterOutput, error) {
		userAgent := sso.UserAgent(version)
		scopes, err := sso.ScopesFor(in.ScopeGroups)
		if err != nil {
			return nil, AddCharacterOutput{}, err
		}
		var warnings []string
		scopesKept := true
		if in.CharacterID != 0 {
			// FC replaces a token's scopes on every login rather than
			// adding to them, so a re-authorization must ask again for
			// everything the character already granted. If they can't be
			// read (most often a revoked refresh token, which this login
			// is the fix for), go ahead, but say what may have been lost.
			kept, err := grantedScopes(ctx, clientID, in.CharacterID, userAgent)
			if err != nil {
				slog.Default().WarnContext(ctx, "add_character: can't read the character's current scopes; requesting only the chosen groups", "characterId", in.CharacterID, "error", err)
				scopesKept = false
				warnings = append(warnings, fmt.Sprintf("couldn't read the scope groups character %d had already granted (%v), so this login requested only %v. Any other group it had before is no longer granted: ask the user whether it needs any back, and if so call add_character again with characterId %d and those groups.", in.CharacterID, err, sso.GrantedGroups(scopes), in.CharacterID))
			}
			scopes = union(scopes, kept)
		}

		chars, err := addCharacters(ctx, clientID, userAgent, sso.AddOptions{
			Scopes: scopes,
			// Save each character as its login completes, not once the
			// session ends, so the session page lists it and an abandoned
			// session doesn't lose it.
			OnAdded: func(c sso.Character) error {
				if _, err := store.Update(func(cfg *config.Config) error {
					cfg.UpsertCharacter(config.Character{ID: c.ID, Name: c.Name})
					return nil
				}); err != nil {
					return fmt.Errorf("save character: %w", err)
				}
				return nil
			},
			SavedCharacters: func() ([]sso.Character, error) {
				cfg, err := store.Load()
				if err != nil {
					return nil, fmt.Errorf("load config: %w", err)
				}
				out := make([]sso.Character, len(cfg.Characters))
				for i, c := range cfg.Characters {
					out[i] = sso.Character{ID: c.ID, Name: c.Name, Account: c.Account}
				}
				return out, nil
			},
			Accounts: accountEditor(store),
		})
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("add character: %w", err)
		}

		out := AddCharacterOutput{Characters: make([]AddedCharacter, len(chars))}
		reauthorized := false
		for i, c := range chars {
			out.Characters[i] = AddedCharacter{CharacterID: c.ID, CharacterName: c.Name, ScopeGroups: sso.GrantedGroups(c.Scopes)}
			if c.ID == in.CharacterID {
				reauthorized = true
			}
		}
		if in.CharacterID != 0 && !reauthorized {
			// Every character logged in this session was granted the
			// target's scopes, but the target itself is unchanged.
			if len(in.ScopeGroups) == 0 {
				// Only say the other logins got the target's groups when
				// those groups were read and requested.
				given := ""
				if scopesKept {
					given = fmt.Sprintf(" Any character the user logged in instead was given character %d's scope groups.", in.CharacterID)
				}
				warnings = append(warnings, fmt.Sprintf("character %d was not re-authorized: the user didn't log in as it on EVE's page. It keeps the access it already had; only its login wasn't refreshed.%s If they meant to refresh character %d, call add_character with characterId %d again and ask them to log in as that character.", in.CharacterID, given, in.CharacterID, in.CharacterID))
			} else {
				warnings = append(warnings, fmt.Sprintf("character %d was not re-authorized: the user didn't log in as it on EVE's page, so it still lacks %v. Tell the user, and if they want to try again, call add_character with characterId %d and ask them to log in as that character.", in.CharacterID, in.ScopeGroups, in.CharacterID))
			}
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("load config: %w", err)
		}
		for i := range out.Characters {
			if c, ok := cfg.Character(out.Characters[i].CharacterID); ok {
				out.Characters[i].Account = c.Account
			}
		}
		out.Warnings = append(warnings, accountWarnings(cfg)...)
		return nil, out, nil
	}
}

// accountEditor saves the account grouping the user sets on the login page.
func accountEditor(store *config.Store) *sso.AccountEditor {
	edit := func(change func(*config.Config) error) error {
		_, err := store.Update(change)
		return err
	}
	return &sso.AccountEditor{
		Set: func(id int32, label string) error {
			return edit(func(c *config.Config) error { _, err := c.SetCharacterAccount(id, label); return err })
		},
		Rename: func(from, to string) error {
			return edit(func(c *config.Config) error { _, err := c.RenameAccount(from, to); return err })
		},
		Delete: func(label string) error {
			return edit(func(c *config.Config) error { _, err := c.DeleteAccount(label); return err })
		},
	}
}

// union returns the sorted, de-duplicated union of a and b.
func union(a, b []string) []string {
	set := make(map[string]bool, len(a)+len(b))
	for _, s := range a {
		set[s] = true
	}
	for _, s := range b {
		set[s] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// ScopeGroupInfo describes one scope group for list_scope_groups.
type ScopeGroupInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Default     bool     `json:"default"`
	Scopes      []string `json:"scopes"`
}

// ListScopeGroupsOutput is list_scope_groups' tool result.
type ListScopeGroupsOutput struct {
	Groups []ScopeGroupInfo `json:"groups"`
}

func listScopeGroupsHandler() mcpsdk.ToolHandlerFor[any, ListScopeGroupsOutput] {
	return func(context.Context, *mcpsdk.CallToolRequest, any) (*mcpsdk.CallToolResult, ListScopeGroupsOutput, error) {
		out := ListScopeGroupsOutput{Groups: make([]ScopeGroupInfo, len(sso.ScopeGroups))}
		for i, g := range sso.ScopeGroups {
			out.Groups[i] = ScopeGroupInfo{Name: g.Name, Description: g.Description, Default: g.Name == sso.DefaultScopeGroup, Scopes: g.Scopes}
		}
		return nil, out, nil
	}
}
