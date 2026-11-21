package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func threeCharacters() config.Config {
	return config.Config{Characters: []config.Character{
		{ID: 1, Name: "Main", Account: "Main account", Notes: "PvP"},
		{ID: 2, Name: "Alt", Account: "Alt account"},
		{ID: 3, Name: "Cyno"},
	}}
}

func TestNormalizeAccountLabel(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantErr        bool
	}{
		{"trims and collapses spaces", "  Alt   account ", "Alt account", false},
		{"empty means unknown", "   ", "", false},
		{"too long", strings.Repeat("a", config.MaxAccountLabelLength+1), "", true},
		{"exactly the limit", strings.Repeat("é", config.MaxAccountLabelLength), strings.Repeat("é", config.MaxAccountLabelLength), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.NormalizeAccountLabel(tt.in)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("NormalizeAccountLabel(%q) = %q, %v; want %q, error %v", tt.in, got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestUpsertCharacter_KeepsAccountAndNotes(t *testing.T) {
	cfg := threeCharacters()
	cfg.UpsertCharacter(config.Character{ID: 1, Name: "Main Renamed"})
	got, _ := cfg.Character(1)
	if got.Name != "Main Renamed" || got.Account != "Main account" || got.Notes != "PvP" {
		t.Errorf("character 1 = %+v, want the new name with its account and notes kept", got)
	}
	cfg.UpsertCharacter(config.Character{ID: 4, Name: "New"})
	if len(cfg.Characters) != 4 {
		t.Errorf("a new character wasn't appended: %+v", cfg.Characters)
	}
}

func TestSetCharacterAccount(t *testing.T) {
	cfg := threeCharacters()
	old, err := cfg.SetCharacterAccount(3, " Alt account ")
	if err != nil || old != "" {
		t.Fatalf("SetCharacterAccount = %q, %v", old, err)
	}
	if got, _ := cfg.Character(3); got.Account != "Alt account" {
		t.Errorf("account = %q", got.Account)
	}
	old, err = cfg.SetCharacterAccount(3, "")
	if err != nil || old != "Alt account" {
		t.Errorf("clearing = %q, %v", old, err)
	}
	if _, err := cfg.SetCharacterAccount(99, "x"); err == nil {
		t.Error("unknown character: want an error")
	}
}

func TestSetCharacterNotes(t *testing.T) {
	cfg := threeCharacters()
	old, err := cfg.SetCharacterNotes(1, "  carrier pilot\n")
	if err != nil || old != "PvP" {
		t.Fatalf("SetCharacterNotes = %q, %v", old, err)
	}
	if got, _ := cfg.Character(1); got.Notes != "carrier pilot" {
		t.Errorf("notes = %q", got.Notes)
	}
}

func TestRenameAccount_MergesIntoExistingLabel(t *testing.T) {
	cfg := threeCharacters()
	changed, err := cfg.RenameAccount("Alt account", "Main account")
	if err != nil || !slices.Equal(changed, []int32{2}) {
		t.Fatalf("RenameAccount = %v, %v", changed, err)
	}
	if got := cfg.AccountLabels(); !slices.Equal(got, []string{"Main account"}) {
		t.Errorf("labels = %v", got)
	}
	if _, err := cfg.RenameAccount("Nobody", "x"); err == nil {
		t.Error("renaming a label nobody has: want an error")
	}
	if _, err := cfg.RenameAccount("Main account", " "); err == nil {
		t.Error("renaming to an empty label: want an error")
	}
}

func TestDeleteAccount_UnassignsItsCharacters(t *testing.T) {
	cfg := threeCharacters()
	changed, err := cfg.DeleteAccount("Main account")
	if err != nil || !slices.Equal(changed, []int32{1}) {
		t.Fatalf("DeleteAccount = %v, %v", changed, err)
	}
	if got, _ := cfg.Character(1); got.Account != "" || got.Notes != "PvP" {
		t.Errorf("character 1 = %+v, want unassigned with its notes kept", got)
	}
	if _, err := cfg.DeleteAccount("Main account"); err == nil {
		t.Error("deleting a label nobody has any more: want an error")
	}
}

func TestUnlabelledCharacters(t *testing.T) {
	cfg := threeCharacters()
	if got := cfg.UnlabelledCharacters(); len(got) != 1 || got[0].ID != 3 {
		t.Errorf("unlabelled = %+v, want only character 3", got)
	}
	one := config.Config{Characters: []config.Character{{ID: 1, Name: "Only"}}}
	if got := one.UnlabelledCharacters(); got != nil {
		t.Errorf("a single character can't share an account: got %+v", got)
	}
}
