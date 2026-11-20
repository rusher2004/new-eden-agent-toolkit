package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func characterStore(t *testing.T, chars ...config.Character) *config.Store {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: chars}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

func TestUpdateCharacterDetails_SetsAccountForSeveralAndReportsChanges(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Main"}, config.Character{ID: 2, Name: "Alt"}, config.Character{ID: 3, Name: "Cyno"})
	_, out, err := updateCharacterDetailsHandler(store)(context.Background(), nil, UpdateCharacterDetailsInput{CharacterIDs: []int32{2, 3}, Account: ptr("Alts")})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Changes) != 2 || out.Changes[0] != (FieldChange{CharacterID: 2, CharacterName: "Alt", Field: "account", Old: "", New: "Alts"}) {
		t.Errorf("changes = %+v", out.Changes)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Main (1)") || strings.Contains(out.Warnings[0], "Alt (2)") {
		t.Errorf("warnings = %v, want only Main still unknown", out.Warnings)
	}
	cfg, _ := store.Load()
	if c, _ := cfg.Character(3); c.Account != "Alts" {
		t.Errorf("saved account = %q", c.Account)
	}
}

func TestUpdateCharacterDetails_NotesNeedOneCharacter(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Main"}, config.Character{ID: 2, Name: "Alt"})
	h := updateCharacterDetailsHandler(store)
	if _, _, err := h(context.Background(), nil, UpdateCharacterDetailsInput{CharacterIDs: []int32{1, 2}, Notes: ptr("x")}); err == nil {
		t.Error("notes for two characters: want an error")
	}
	_, out, err := h(context.Background(), nil, UpdateCharacterDetailsInput{CharacterIDs: []int32{1}, Notes: ptr("carrier pilot")})
	if err != nil || len(out.Changes) != 1 || out.Changes[0].Field != "notes" || out.Changes[0].New != "carrier pilot" {
		t.Errorf("out = %+v, %v", out, err)
	}
}

func TestUpdateCharacterDetails_RenameAccount(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Main", Account: "A"}, config.Character{ID: 2, Name: "Alt", Account: "A"})
	_, out, err := updateCharacterDetailsHandler(store)(context.Background(), nil, UpdateCharacterDetailsInput{RenameAccount: &AccountRename{From: "A", To: "Main account"}})
	if err != nil || len(out.Changes) != 2 || out.Changes[1].Old != "A" || out.Changes[1].New != "Main account" {
		t.Errorf("out = %+v, %v", out, err)
	}
}

func TestUpdateCharacterDetails_RejectsBadInput(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Main"})
	h := updateCharacterDetailsHandler(store)
	for name, in := range map[string]UpdateCharacterDetailsInput{
		"nothing to change":   {CharacterIDs: []int32{1}},
		"no characters":       {Account: ptr("A")},
		"unknown character":   {CharacterIDs: []int32{9}, Account: ptr("A")},
		"label too long":      {CharacterIDs: []int32{1}, Account: ptr(strings.Repeat("x", config.MaxAccountLabelLength+1))},
		"rename unused label": {RenameAccount: &AccountRename{From: "Nope", To: "B"}},
	} {
		if _, _, err := h(context.Background(), nil, in); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	cfg, _ := store.Load()
	if c, _ := cfg.Character(1); c.Account != "" {
		t.Errorf("a rejected update was saved: %+v", c)
	}
}

func TestListCharacters_ReturnsAccountsAndWarnsAboutUnknown(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Main", Account: "Main account", Notes: "PvP"}, config.Character{ID: 2, Name: "Alt"})
	_, out, err := listCharactersHandler(store)(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if out.Characters[0] != (CharacterInfo{CharacterID: 1, CharacterName: "Main", Account: "Main account", Notes: "PvP"}) {
		t.Errorf("character = %+v", out.Characters[0])
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Alt (2)") || !strings.Contains(out.Warnings[0], "Main account") {
		t.Errorf("warnings = %v", out.Warnings)
	}
}

func TestListCharacters_NoAccountWarningForOneCharacter(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Main"})
	_, out, _ := listCharactersHandler(store)(context.Background(), nil, nil)
	if out.Warnings != nil {
		t.Errorf("warnings = %v, want none", out.Warnings)
	}
}

func TestAddCharacter_KeepsAccountOnReAddAndWarnsAboutUnknown(t *testing.T) {
	store := characterStore(t, config.Character{ID: 42, Name: "Main", Account: "Main account", Notes: "PvP"})
	fakeAddCharacters(t, []sso.Character{{ID: 42, Name: "Main"}, {ID: 43, Name: "New Alt"}}, nil)
	_, out, err := addCharacterHandler("v1", "client-id", store)(context.Background(), nil, AddCharacterInput{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if out.Characters[0].Account != "Main account" || out.Characters[1].Account != "" {
		t.Errorf("characters = %+v", out.Characters)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "New Alt (43)") {
		t.Errorf("warnings = %v", out.Warnings)
	}
	cfg, _ := store.Load()
	if c, _ := cfg.Character(42); c.Account != "Main account" || c.Notes != "PvP" {
		t.Errorf("re-adding lost the account or notes: %+v", c)
	}
}
