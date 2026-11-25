package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// LoginCharacter is a character the login flow knows about: one just logged
// in, or one neat already has saved.
type LoginCharacter struct {
	ID   int32
	Name string
	// Account is the user's label for the EVE account the character is on;
	// empty means unknown.
	Account string
	// Scopes are the ESI scopes the login granted, empty for a saved
	// character that wasn't logged in this session.
	Scopes []string
}

// AccountEdits saves the account grouping the user sets on the login page.
// Each call is one change; an error is shown on the page.
type AccountEdits struct {
	// Set gives a character an account label; "" makes it unassigned.
	Set func(characterID int32, label string) error
	// Rename relabels every character in an account.
	Rename func(from, to string) error
	// Delete unassigns every character in an account.
	Delete func(label string) error
}

// LoginOptions configures one browser login session.
type LoginOptions struct {
	// Scopes are the ESI scopes requested for every character.
	Scopes []string
	// OnAdded is called as each character's login completes, so it can be
	// saved while the session is still open.
	OnAdded func(LoginCharacter) error
	// SavedCharacters lists the characters neat already knows, for the
	// session page.
	SavedCharacters func() ([]LoginCharacter, error)
	// Accounts lets the user group characters by account on the session page.
	Accounts *AccountEdits
}

// characterLogin is what add_character asks of the login flow: map scope
// group names to scopes, read what a character already granted, and run a
// browser session that logs in characters. The composition root implements it
// over the ESI login flow.
type characterLogin interface {
	// ScopesFor returns the scopes for the always-included basic group plus
	// groups, or an error naming an unknown group.
	ScopesFor(groups []string) ([]string, error)
	// GrantedGroups names the scope groups scopes fully cover.
	GrantedGroups(scopes []string) []string
	// GrantedScopes reads the scopes a logged-in character currently holds.
	GrantedScopes(ctx context.Context, characterID int32) ([]string, error)
	// AddCharacters runs a login session and returns every character added.
	AddCharacters(ctx context.Context, opts LoginOptions) ([]LoginCharacter, error)
}

// scopeGroupCatalog is what list_scope_groups asks of the login flow.
type scopeGroupCatalog interface {
	// ScopeGroups lists every scope group a login can request.
	ScopeGroups() []ScopeGroupInfo
}

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

// AddAddCharacter registers add_character. login is the ESI login flow;
// store is the config the added characters are saved to.
func AddAddCharacter(s *mcpsdk.Server, login characterLogin, store addCharacterConfig) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name: "add_character",
		Description: "Log in EVE Online characters via ESI OAuth2. Opens the user's browser at EVE's login page; afterwards the browser shows a neat page listing their characters, where they can log in more characters before clicking Done or closing the tab. Blocks until then and returns every character added, with the scope groups each granted. " +
			"Every login grants the basic scope group plus scopeGroups: before calling, ask the user what they'll use neat for, including whether they want their corporation's data, and pick groups from list_scope_groups (or use only basic if they don't know). A login replaces each character's access with exactly what it requested, so to add a group to characters already logged in, either pass characterId or list every group they should end up with; list_characters doesn't show a character's current groups, so if you don't know them, pass characterId. " +
			"If no browser tab opens, the user can open http://127.0.0.1:48211/ themselves while the tool waits. " +
			"To grant more groups to a character later (e.g. after a tool fails because a scope is missing), pass its characterId: its existing scopes are kept. Check the result's warnings: they say when the user logged in as a different character, or when the character's earlier scope groups couldn't be read and may need granting again. Re-adding a known character refreshes its login.",
	}, addCharacterHandler(login, store))
}

// addCharacterConfig is the config calls add_character makes: read the saved
// characters, and change them under the store's lock.
type addCharacterConfig interface {
	configLoader
	configUpdater
}

func addCharacterHandler(login characterLogin, store addCharacterConfig) mcpsdk.ToolHandlerFor[AddCharacterInput, AddCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in AddCharacterInput) (*mcpsdk.CallToolResult, AddCharacterOutput, error) {
		scopes, err := login.ScopesFor(in.ScopeGroups)
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
			kept, err := login.GrantedScopes(ctx, in.CharacterID)
			if err != nil {
				slog.Default().WarnContext(ctx, "add_character: can't read the character's current scopes; requesting only the chosen groups", "characterId", in.CharacterID, "error", err)
				scopesKept = false
				warnings = append(warnings, fmt.Sprintf("couldn't read the scope groups character %d had already granted (%v), so this login requested only %v. Any other group it had before is no longer granted: ask the user whether it needs any back, and if so call add_character again with characterId %d and those groups.", in.CharacterID, err, login.GrantedGroups(scopes), in.CharacterID))
			}
			scopes = union(scopes, kept)
		}

		chars, err := login.AddCharacters(ctx, LoginOptions{
			Scopes: scopes,
			// Save each character as its login completes, not once the
			// session ends, so the session page lists it and an abandoned
			// session doesn't lose it.
			OnAdded: func(c LoginCharacter) error {
				if _, err := store.Update(func(cfg *config.Config) error {
					cfg.UpsertCharacter(config.Character{ID: c.ID, Name: c.Name})
					return nil
				}); err != nil {
					return fmt.Errorf("save character: %w", err)
				}
				return nil
			},
			SavedCharacters: func() ([]LoginCharacter, error) {
				cfg, err := store.Load()
				if err != nil {
					return nil, fmt.Errorf("load config: %w", err)
				}
				out := make([]LoginCharacter, len(cfg.Characters))
				for i, c := range cfg.Characters {
					out[i] = LoginCharacter{ID: c.ID, Name: c.Name, Account: c.Account}
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
			out.Characters[i] = AddedCharacter{CharacterID: c.ID, CharacterName: c.Name, ScopeGroups: login.GrantedGroups(c.Scopes)}
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
func accountEditor(store configUpdater) *AccountEdits {
	edit := func(change func(*config.Config) error) error {
		_, err := store.Update(change)
		return err
	}
	return &AccountEdits{
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

// AddListScopeGroups registers list_scope_groups. catalog lists the scope
// groups the login flow can grant.
func AddListScopeGroups(s *mcpsdk.Server, catalog scopeGroupCatalog) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_scope_groups",
		Description: "List the scope groups add_character can grant a character: each group's name, what it lets neat do, and its ESI scopes. The basic group is always granted.",
	}, listScopeGroupsHandler(catalog))
}

func listScopeGroupsHandler(catalog scopeGroupCatalog) mcpsdk.ToolHandlerFor[any, ListScopeGroupsOutput] {
	return func(context.Context, *mcpsdk.CallToolRequest, any) (*mcpsdk.CallToolResult, ListScopeGroupsOutput, error) {
		return nil, ListScopeGroupsOutput{Groups: catalog.ScopeGroups()}, nil
	}
}
