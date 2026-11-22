package mcp

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// fakeAddCharacters stands in for sso.AddCharacters: it "logs in" each of
// chars in turn through opts.OnAdded, as the real session does, and records
// what opts.SavedCharacters reported after the last one.
func fakeAddCharacters(t *testing.T, chars []sso.Character, savedAfter *[]sso.Character) {
	t.Helper()
	orig := addCharacters
	t.Cleanup(func() { addCharacters = orig })
	addCharacters = func(ctx context.Context, clientID, userAgent string, opts sso.AddOptions) ([]sso.Character, error) {
		for _, c := range chars {
			if err := opts.OnAdded(c); err != nil {
				return nil, err
			}
		}
		if savedAfter != nil {
			saved, err := opts.SavedCharacters()
			if err != nil {
				return nil, err
			}
			*savedAfter = saved
		}
		return chars, nil
	}
}

func TestAddCharacterHandler_SavesEachCharacterAndReturnsAll(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	chars := []sso.Character{{ID: 42, Name: "Test Pilot"}, {ID: 43, Name: "Alt Pilot"}}
	var saved []sso.Character
	fakeAddCharacters(t, chars, &saved)

	handler := addCharacterHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, AddCharacterInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []AddedCharacter{{CharacterID: 42, CharacterName: "Test Pilot"}, {CharacterID: 43, CharacterName: "Alt Pilot"}}
	for i := range want {
		want[i].ScopeGroups = sso.GrantedGroups(nil)
	}
	if !reflect.DeepEqual(out.Characters, want) {
		t.Errorf("out.Characters = %+v, want %+v", out.Characters, want)
	}
	if !reflect.DeepEqual(saved, chars) {
		t.Errorf("SavedCharacters during the session = %+v, want %+v", saved, chars)
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
	handler := addCharacterHandler("v1", "client-id", store)

	fakeAddCharacters(t, []sso.Character{{ID: 42, Name: "Test Pilot"}}, nil)
	if _, _, err := handler(context.Background(), nil, AddCharacterInput{}); err != nil {
		t.Fatalf("first handler call: %v", err)
	}

	fakeAddCharacters(t, []sso.Character{{ID: 42, Name: "Renamed Pilot"}}, nil)
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

	orig := addCharacters
	t.Cleanup(func() { addCharacters = orig })
	wantErr := errors.New("upstream failure")
	addCharacters = func(ctx context.Context, clientID, userAgent string, opts sso.AddOptions) ([]sso.Character, error) {
		return nil, wantErr
	}

	handler := addCharacterHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, AddCharacterInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// captureScopes replaces addCharacters with a fake that records the scopes
// requested and logs in one character that granted exactly those.
func captureScopes(t *testing.T, got *[]string) {
	t.Helper()
	orig := addCharacters
	t.Cleanup(func() { addCharacters = orig })
	addCharacters = func(ctx context.Context, clientID, userAgent string, opts sso.AddOptions) ([]sso.Character, error) {
		*got = opts.Scopes
		return []sso.Character{{ID: 42, Name: "Test Pilot", Scopes: opts.Scopes}}, nil
	}
}

func TestAddCharacterHandler_RequestsDefaultPlusChosenGroups(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	var got []string
	captureScopes(t, &got)

	_, out, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, AddCharacterInput{ScopeGroups: []string{"wallet", "mail"}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	want, err := sso.ScopesFor([]string{"wallet", "mail"})
	if err != nil {
		t.Fatalf("ScopesFor: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("requested scopes = %v, want %v", got, want)
	}
	if groups := out.Characters[0].ScopeGroups; !reflect.DeepEqual(groups, []string{"basic", "wallet", "mail"}) {
		t.Errorf("reported scope groups = %v", groups)
	}
}

func TestAddCharacterHandler_UnknownGroup_FailsBeforeLogin(t *testing.T) {
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	orig := addCharacters
	t.Cleanup(func() { addCharacters = orig })
	addCharacters = func(context.Context, string, string, sso.AddOptions) ([]sso.Character, error) {
		t.Fatal("login started despite an unknown scope group")
		return nil, nil
	}

	if _, _, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, AddCharacterInput{ScopeGroups: []string{"nope"}}); err == nil {
		t.Error("unknown scope group accepted")
	}
}

func TestAddCharacterHandler_ReAuthorize_KeepsExistingScopes(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	var got []string
	captureScopes(t, &got)

	origGranted := grantedScopes
	t.Cleanup(func() { grantedScopes = origGranted })
	grantedScopes = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]string, error) {
		if characterID != 42 {
			t.Errorf("read scopes of character %d, want 42", characterID)
		}
		return []string{"publicData", "esi-mail.read_mail.v1"}, nil
	}

	if _, _, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, AddCharacterInput{CharacterID: 42, ScopeGroups: []string{"wallet"}}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, want := range []string{"esi-mail.read_mail.v1", "esi-wallet.read_character_wallet.v1", "esi-skills.read_skills.v1"} {
		if !slices.Contains(got, want) {
			t.Errorf("requested scopes are missing %s: %v", want, got)
		}
	}
}

func TestListScopeGroups_MarksDefault(t *testing.T) {
	_, out, err := listScopeGroupsHandler()(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Groups) != len(sso.ScopeGroups) {
		t.Fatalf("%d groups, want %d", len(out.Groups), len(sso.ScopeGroups))
	}
	for _, g := range out.Groups {
		if g.Default != (g.Name == sso.DefaultScopeGroup) {
			t.Errorf("group %q default = %v", g.Name, g.Default)
		}
	}
}

func TestAddCharacterHandler_ReAuthorize_WarnsWhenOldScopesUnreadable(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	var got []string
	captureScopes(t, &got)

	origGranted := grantedScopes
	t.Cleanup(func() { grantedScopes = origGranted })
	grantedScopes = func(context.Context, string, int32, string) ([]string, error) {
		return nil, errors.New("refresh token revoked")
	}

	_, out, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, AddCharacterInput{CharacterID: 42, ScopeGroups: []string{"wallet"}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	want, err := sso.ScopesFor([]string{"wallet"})
	if err != nil {
		t.Fatalf("ScopesFor: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("requested scopes = %v, want only the chosen groups %v", got, want)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "refresh token revoked") || !strings.Contains(out.Warnings[0], "characterId 42") {
		t.Errorf("warnings = %q, want one naming the read error and how to re-grant", out.Warnings)
	}
}

func TestAddCharacterHandler_ReAuthorize_WarnsWhenOtherCharacterLoggedIn(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	fakeAddCharacters(t, []sso.Character{{ID: 43, Name: "Alt Pilot"}}, nil)

	origGranted := grantedScopes
	t.Cleanup(func() { grantedScopes = origGranted })
	grantedScopes = func(context.Context, string, int32, string) ([]string, error) { return nil, nil }

	_, out, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, AddCharacterInput{CharacterID: 42, ScopeGroups: []string{"wallet"}})
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
	var got []string
	captureScopes(t, &got)

	origGranted := grantedScopes
	t.Cleanup(func() { grantedScopes = origGranted })
	grantedScopes = func(context.Context, string, int32, string) ([]string, error) { return []string{"publicData"}, nil }

	for _, in := range []AddCharacterInput{{}, {CharacterID: 42, ScopeGroups: []string{"wallet"}}} {
		_, out, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, in)
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
	fakeAddCharacters(t, []sso.Character{{ID: 43, Name: "Alt Pilot"}}, nil)

	origGranted := grantedScopes
	t.Cleanup(func() { grantedScopes = origGranted })
	grantedScopes = func(context.Context, string, int32, string) ([]string, error) { return nil, nil }

	_, out, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, AddCharacterInput{CharacterID: 42})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "keeps the access it already had") || strings.Contains(out.Warnings[0], "lacks") {
		t.Errorf("warnings = %q, want one saying character 42 keeps its access", out.Warnings)
	}
}
