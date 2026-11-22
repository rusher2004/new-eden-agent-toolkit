package config

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxAccountLabelLength caps an account label, which only has to tell the
// user's accounts apart.
const MaxAccountLabelLength = 40

// NormalizeAccountLabel trims a label and checks its length. An empty result
// means unknown.
func NormalizeAccountLabel(label string) (string, error) {
	label = strings.Join(strings.Fields(label), " ")
	if utf8.RuneCountInString(label) > MaxAccountLabelLength {
		return "", fmt.Errorf("account label %q is longer than %d characters", label, MaxAccountLabelLength)
	}
	return label, nil
}

// Character returns the saved character with the given ID.
func (c *Config) Character(id int32) (Character, bool) {
	for _, ch := range c.Characters {
		if ch.ID == id {
			return ch, true
		}
	}
	return Character{}, false
}

// UpsertCharacter records a logged-in character. A character neat already
// knows keeps its account label and notes; only its name is refreshed.
func (c *Config) UpsertCharacter(ch Character) {
	for i, existing := range c.Characters {
		if existing.ID == ch.ID {
			c.Characters[i].Name = ch.Name
			return
		}
	}
	c.Characters = append(c.Characters, ch)
}

// SetCharacterAccount gives a saved character an account label; an empty
// label clears it (unknown). It returns the label the character had before.
func (c *Config) SetCharacterAccount(id int32, label string) (string, error) {
	label, err := NormalizeAccountLabel(label)
	if err != nil {
		return "", err
	}
	for i, ch := range c.Characters {
		if ch.ID == id {
			old := ch.Account
			c.Characters[i].Account = label
			return old, nil
		}
	}
	return "", fmt.Errorf("no saved character with ID %d", id)
}

// SetCharacterNotes replaces a saved character's notes. It returns the notes
// the character had before.
func (c *Config) SetCharacterNotes(id int32, notes string) (string, error) {
	for i, ch := range c.Characters {
		if ch.ID == id {
			old := ch.Notes
			c.Characters[i].Notes = strings.TrimSpace(notes)
			return old, nil
		}
	}
	return "", fmt.Errorf("no saved character with ID %d", id)
}

// RenameAccount relabels every character labelled from as to. Renaming onto
// a label already in use merges the two groups. It returns the IDs of the
// characters it changed.
func (c *Config) RenameAccount(from, to string) ([]int32, error) {
	from, err := NormalizeAccountLabel(from)
	if err != nil {
		return nil, err
	}
	to, err = NormalizeAccountLabel(to)
	if err != nil {
		return nil, err
	}
	if from == "" || to == "" {
		return nil, fmt.Errorf("renaming an account needs both its current and its new label")
	}
	var changed []int32
	for i, ch := range c.Characters {
		if ch.Account == from {
			c.Characters[i].Account = to
			changed = append(changed, ch.ID)
		}
	}
	if len(changed) == 0 {
		return nil, fmt.Errorf("no character has the account label %q", from)
	}
	return changed, nil
}

// DeleteAccount clears the label from every character that has it, so they
// become unassigned. It returns the IDs of the characters it changed.
func (c *Config) DeleteAccount(label string) ([]int32, error) {
	label, err := NormalizeAccountLabel(label)
	if err != nil {
		return nil, err
	}
	if label == "" {
		return nil, fmt.Errorf("deleting an account needs its label")
	}
	var changed []int32
	for i, ch := range c.Characters {
		if ch.Account == label {
			c.Characters[i].Account = ""
			changed = append(changed, ch.ID)
		}
	}
	if len(changed) == 0 {
		return nil, fmt.Errorf("no character has the account label %q", label)
	}
	return changed, nil
}

// AccountLabels lists the account labels in use, in the order their first
// character appears.
func (c *Config) AccountLabels() []string {
	seen := map[string]bool{}
	var out []string
	for _, ch := range c.Characters {
		if ch.Account != "" && !seen[ch.Account] {
			seen[ch.Account] = true
			out = append(out, ch.Account)
		}
	}
	return out
}

// UnlabelledCharacters lists the characters with no account label, but only
// when two or more characters are saved: with one, there is no other
// character it could share an account with.
func (c *Config) UnlabelledCharacters() []Character {
	if len(c.Characters) < 2 {
		return nil
	}
	var out []Character
	for _, ch := range c.Characters {
		if ch.Account == "" {
			out = append(out, ch)
		}
	}
	return out
}
