package commands

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// addAdminTools registers the character administration tools: the ones that
// log characters in, list and describe them, and remove them.
func addAdminTools(s *mcpsdk.Server, d toolDeps) {
	login := ssoLogin{clientID: d.clientID, userAgent: d.userAgent}
	mcp.AddAddCharacter(s, login, d.store)
	mcp.AddListScopeGroups(s, login)
	mcp.AddListCharacters(s, d.store)
	mcp.AddUpdateCharacterDetails(s, d.store)
	mcp.AddRemoveCharacter(s, d.store, sso.RemoveCharacter)
}

// ssoLogin is the ESI login flow as the mcp package's login interfaces see
// it. It carries the ESI client ID and User-Agent that sso's functions take,
// and converts between mcp's login types and sso's.
type ssoLogin struct {
	clientID  string
	userAgent string
}

func (ssoLogin) ScopesFor(groups []string) ([]string, error) { return sso.ScopesFor(groups) }

func (ssoLogin) GrantedGroups(scopes []string) []string { return sso.GrantedGroups(scopes) }

func (l ssoLogin) GrantedScopes(ctx context.Context, characterID int32) ([]string, error) {
	return sso.GrantedScopes(ctx, l.clientID, characterID, l.userAgent)
}

// ScopeGroups lists every group a login can request, marking the always
// granted one.
func (ssoLogin) ScopeGroups() []mcp.ScopeGroupInfo {
	out := make([]mcp.ScopeGroupInfo, len(sso.ScopeGroups))
	for i, g := range sso.ScopeGroups {
		out[i] = mcp.ScopeGroupInfo{Name: g.Name, Description: g.Description, Default: g.Name == sso.DefaultScopeGroup, Scopes: g.Scopes}
	}
	return out
}

func (l ssoLogin) AddCharacters(ctx context.Context, opts mcp.LoginOptions) ([]mcp.LoginCharacter, error) {
	ssoOpts := sso.AddOptions{Scopes: opts.Scopes}
	if opts.OnAdded != nil {
		ssoOpts.OnAdded = func(c sso.Character) error { return opts.OnAdded(loginCharacter(c)) }
	}
	if opts.SavedCharacters != nil {
		ssoOpts.SavedCharacters = func() ([]sso.Character, error) {
			saved, err := opts.SavedCharacters()
			if err != nil {
				return nil, err
			}
			out := make([]sso.Character, len(saved))
			for i, c := range saved {
				out[i] = sso.Character(c)
			}
			return out, nil
		}
	}
	if a := opts.Accounts; a != nil {
		ssoOpts.Accounts = &sso.AccountEditor{Set: a.Set, Rename: a.Rename, Delete: a.Delete}
	}
	chars, err := sso.AddCharacters(ctx, l.clientID, l.userAgent, ssoOpts)
	if err != nil {
		return nil, fmt.Errorf("log in characters: %w", err)
	}
	out := make([]mcp.LoginCharacter, len(chars))
	for i, c := range chars {
		out[i] = loginCharacter(c)
	}
	return out, nil
}

// loginCharacter converts sso's character to mcp's; the two have the same
// fields, so the conversion fails to compile if either one changes.
func loginCharacter(c sso.Character) mcp.LoginCharacter { return mcp.LoginCharacter(c) }
