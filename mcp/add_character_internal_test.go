package mcp

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// fakeLogin stands in for the ESI login flow, so no browser, network or
// OAuth2 exchange runs. Scope groups "basic", "wallet" and "mail" exist; a
// group's scope is "scope:<group>". AddCharacters "logs in" each of chars in
// turn through opts.OnAdded, as the real session does, and records what
// opts.SavedCharacters reported after the last one.
type fakeLogin struct {
	t *testing.T
	// chars are logged in by AddCharacters. When grantRequested is set, one
	// character 42 is logged in instead, granted exactly the scopes asked for.
	chars          []LoginCharacter
	grantRequested bool
	addErr         error
	// granted answers GrantedScopes; nil means the character has none.
	granted func(characterID int32) ([]string, error)

	requested  []string
	savedAfter []LoginCharacter
	logins     int
}

func (f *fakeLogin) ScopesFor(groups []string) ([]string, error) {
	set := map[string]bool{"scope:basic": true}
	for _, g := range groups {
		if g != "wallet" && g != "mail" {
			return nil, fmt.Errorf("unknown scope group %q", g)
		}
		set["scope:"+g] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	slices.Sort(out)
	return out, nil
}

func (f *fakeLogin) GrantedGroups(scopes []string) []string {
	var out []string
	for _, g := range []string{"basic", "wallet", "mail"} {
		if slices.Contains(scopes, "scope:"+g) {
			out = append(out, g)
		}
	}
	return out
}

func (f *fakeLogin) GrantedScopes(_ context.Context, characterID int32) ([]string, error) {
	if f.granted == nil {
		return nil, nil
	}
	return f.granted(characterID)
}

func (f *fakeLogin) AddCharacters(_ context.Context, opts LoginOptions) ([]LoginCharacter, error) {
	f.logins++
	f.requested = opts.Scopes
	if f.addErr != nil {
		return nil, f.addErr
	}
	chars := f.chars
	if f.grantRequested {
		chars = []LoginCharacter{{ID: 42, Name: "Test Pilot", Scopes: opts.Scopes}}
	}
	for _, c := range chars {
		if err := opts.OnAdded(c); err != nil {
			return nil, err
		}
	}
	saved, err := opts.SavedCharacters()
	if err != nil {
		return nil, err
	}
	f.savedAfter = saved
	return chars, nil
}

func TestAddCharacterHandler_SavesEachCharacterAndReturnsAll(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	chars := []LoginCharacter{{ID: 42, Name: "Test Pilot"}, {ID: 43, Name: "Alt Pilot"}}
	login := &fakeLogin{t: t, chars: chars}

	handler := addCharacterHandler(login, store)
	_, out, err := handler(context.Background(), nil, AddCharacterInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []AddedCharacter{{CharacterID: 42, CharacterName: "Test Pilot"}, {CharacterID: 43, CharacterName: "Alt Pilot"}}
	if !reflect.DeepEqual(out.Characters, want) {
		t.Errorf("out.Characters = %+v, want %+v", out.Characters, want)
	}
	if !reflect.DeepEqual(login.savedAfter, chars) {
		t.Errorf("SavedCharacters during the session = %+v, want %+v", login.savedAfter, chars)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	wantCfg := []config.Character{{ID: 42, Name: "Test Pilot"}, {ID: 43, Name: "Alt Pilot"}}
	if !reflect.DeepEqual(cfg.Characters, wantCfg) {
		t.Errorf("cfg.Characters = %+v, want %+v", cfg.Characters, wantCfg)
	}
}

func TestAddCharacterHandler_ReAddSameCharacter_ReplacesNotDuplicates(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	login := &fakeLogin{t: t, chars: []LoginCharacter{{ID: 42, Name: "Test Pilot"}}}
	handler := addCharacterHandler(login, store)
	if _, _, err := handler(context.Background(), nil, AddCharacterInput{}); err != nil {
		t.Fatalf("first handler call: %v", err)
	}

	login.chars = []LoginCharacter{{ID: 42, Name: "Renamed Pilot"}}
	if _, _, err := handler(context.Background(), nil, AddCharacterInput{}); err != nil {
		t.Fatalf("second handler call: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if len(cfg.Characters) != 1 || cfg.Characters[0].Name != "Renamed Pilot" {
		t.Errorf("cfg.Characters = %+v, want exactly one updated entry", cfg.Characters)
	}
}

func TestAddCharacterHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	wantErr := errors.New("upstream failure")
	login := &fakeLogin{t: t, addErr: wantErr}

	handler := addCharacterHandler(login, store)
	_, _, err := handler(context.Background(), nil, AddCharacterInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestAddCharacterHandler_RequestsDefaultPlusChosenGroups(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	login := &fakeLogin{t: t, grantRequested: true}

	_, out, err := addCharacterHandler(login, store)(context.Background(), nil, AddCharacterInput{ScopeGroups: []string{"wallet", "mail"}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	want := []string{"scope:basic", "scope:mail", "scope:wallet"}
	if !reflect.DeepEqual(login.requested, want) {
		t.Errorf("requested scopes = %v, want %v", login.requested, want)
	}
	if groups := out.Characters[0].ScopeGroups; !reflect.DeepEqual(groups, []string{"basic", "wallet", "mail"}) {
		t.Errorf("reported scope groups = %v", groups)
	}
}

func TestAddCharacterHandler_UnknownGroup_FailsBeforeLogin(t *testing.T) {
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	login := &fakeLogin{t: t}

	if _, _, err := addCharacterHandler(login, store)(context.Background(), nil, AddCharacterInput{ScopeGroups: []string{"nope"}}); err == nil {
		t.Error("unknown scope group accepted")
	}
	if login.logins != 0 {
		t.Error("login started despite an unknown scope group")
	}
}

func TestAddCharacterHandler_ReAuthorize_KeepsExistingScopes(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	login := &fakeLogin{t: t, grantRequested: true, granted: func(characterID int32) ([]string, error) {
		if characterID != 42 {
			t.Errorf("read scopes of character %d, want 42", characterID)
		}
		return []string{"scope:mail"}, nil
	}}

	if _, _, err := addCharacterHandler(login, store)(context.Background(), nil, AddCharacterInput{CharacterID: 42, ScopeGroups: []string{"wallet"}}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, want := range []string{"scope:basic", "scope:mail", "scope:wallet"} {
		if !slices.Contains(login.requested, want) {
			t.Errorf("requested scopes are missing %s: %v", want, login.requested)
		}
	}
}

// fakeScopeCatalog lists fixed scope groups.
type fakeScopeCatalog []ScopeGroupInfo

func (f fakeScopeCatalog) ScopeGroups() []ScopeGroupInfo { return f }

func TestListScopeGroups_ReturnsTheCatalog(t *testing.T) {
	catalog := fakeScopeCatalog{
		{Name: "basic", Description: "Basic", Default: true, Scopes: []string{"scope:basic"}},
		{Name: "wallet", Description: "Wallet", Scopes: []string{"scope:wallet"}},
	}
	_, out, err := listScopeGroupsHandler(catalog)(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !reflect.DeepEqual(out.Groups, []ScopeGroupInfo(catalog)) {
		t.Errorf("groups = %+v, want %+v", out.Groups, catalog)
	}
}

func TestAddCharacterHandler_ReAuthorize_WarnsWhenOldScopesUnreadable(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	login := &fakeLogin{t: t, grantRequested: true, granted: func(int32) ([]string, error) {
		return nil, errors.New("refresh token revoked")
	}}

	_, out, err := addCharacterHandler(login, store)(context.Background(), nil, AddCharacterInput{CharacterID: 42, ScopeGroups: []string{"wallet"}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	want := []string{"scope:basic", "scope:wallet"}
	if !reflect.DeepEqual(login.requested, want) {
		t.Errorf("requested scopes = %v, want only the chosen groups %v", login.requested, want)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "refresh token revoked") || !strings.Contains(out.Warnings[0], "characterId 42") {
		t.Errorf("warnings = %q, want one naming the read error and how to re-grant", out.Warnings)
	}
}

func TestAddCharacterHandler_ReAuthorize_WarnsWhenOtherCharacterLoggedIn(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	login := &fakeLogin{t: t, chars: []LoginCharacter{{ID: 43, Name: "Alt Pilot"}}}

	_, out, err := addCharacterHandler(login, store)(context.Background(), nil, AddCharacterInput{CharacterID: 42, ScopeGroups: []string{"wallet"}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "character 42 was not re-authorized") || !strings.Contains(out.Warnings[0], "still lacks [wallet]") {
		t.Errorf("warnings = %q, want one saying character 42 wasn't re-authorized", out.Warnings)
	}
}

func TestAddCharacterHandler_NoWarningsOnPlainAddOrMatchingReAuthorize(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	login := &fakeLogin{t: t, grantRequested: true, granted: func(int32) ([]string, error) { return []string{"scope:basic"}, nil }}

	for _, in := range []AddCharacterInput{{}, {CharacterID: 42, ScopeGroups: []string{"wallet"}}} {
		_, out, err := addCharacterHandler(login, store)(context.Background(), nil, in)
		if err != nil {
			t.Fatalf("handler(%+v): %v", in, err)
		}
		if len(out.Warnings) != 0 {
			t.Errorf("handler(%+v) warnings = %q, want none", in, out.Warnings)
		}
	}
}

func TestAddCharacterHandler_ReAuthorize_NoNewGroups_SaysAccessIsKept(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	login := &fakeLogin{t: t, chars: []LoginCharacter{{ID: 43, Name: "Alt Pilot"}}}

	_, out, err := addCharacterHandler(login, store)(context.Background(), nil, AddCharacterInput{CharacterID: 42})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "keeps the access it already had") || strings.Contains(out.Warnings[0], "lacks") {
		t.Errorf("warnings = %q, want one saying character 42 keeps its access", out.Warnings)
	}
}
