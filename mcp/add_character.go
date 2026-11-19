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
}

// AddCharacterOutput is add_character's tool result: every character the
// user logged in during the browser session, in the order they were added.
type AddCharacterOutput struct {
	Characters []AddedCharacter `json:"characters"`
}

func addCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[AddCharacterInput, AddCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in AddCharacterInput) (*mcpsdk.CallToolResult, AddCharacterOutput, error) {
		userAgent := sso.UserAgent(version)
		scopes, err := sso.ScopesFor(in.ScopeGroups)
		if err != nil {
			return nil, AddCharacterOutput{}, err
		}
		if in.CharacterID != 0 {
			// CCP replaces a token's scopes on every login rather than
			// adding to them, so a re-authorization must ask again for
			// everything the character already granted.
			kept, err := grantedScopes(ctx, clientID, in.CharacterID, userAgent)
			if err != nil {
				slog.Default().WarnContext(ctx, "add_character: can't read the character's current scopes; requesting only the chosen groups", "characterId", in.CharacterID, "error", err)
			}
			scopes = union(scopes, kept)
		}

		chars, err := addCharacters(ctx, clientID, userAgent, sso.AddOptions{
			Scopes: scopes,
			// Save each character as its login completes, not once the
			// session ends, so the session page lists it and an abandoned
			// session doesn't lose it.
			OnAdded: func(c sso.Character) error {
				cfg, err := store.Load()
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				upsertCharacter(&cfg, config.Character{ID: c.ID, Name: c.Name})
				if err := store.Save(cfg); err != nil {
					return fmt.Errorf("save config: %w", err)
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
					out[i] = sso.Character{ID: c.ID, Name: c.Name}
				}
				return out, nil
			},
		})
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("add character: %w", err)
		}

		out := AddCharacterOutput{Characters: make([]AddedCharacter, len(chars))}
		for i, c := range chars {
			out.Characters[i] = AddedCharacter{CharacterID: c.ID, CharacterName: c.Name, ScopeGroups: sso.GrantedGroups(c.Scopes)}
		}
		return nil, out, nil
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

// upsertCharacter replaces the entry in cfg.Characters matching c.ID, or
// appends c if no existing entry matches.
func upsertCharacter(cfg *config.Config, c config.Character) {
	for i, existing := range cfg.Characters {
		if existing.ID == c.ID {
			cfg.Characters[i] = c
			return
		}
	}
	cfg.Characters = append(cfg.Characters, c)
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
