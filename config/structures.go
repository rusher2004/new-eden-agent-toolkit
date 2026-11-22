package config

import (
	"fmt"
	"slices"
	"strconv"
)

// Clone bay statuses. Unknown is the default: neat records yes only when it
// can prove it, and the user is the only source for anything else.
const (
	CloneBayYes     = "yes"
	CloneBayNo      = "no"
	CloneBayUnknown = "unknown"
)

// Where a clone bay status came from.
const (
	CloneBayFromJumpClone = "jump_clone" // a logged-in character has a jump clone there
	CloneBayFromUser      = "user"       // the user checked in game
)

// NamedID is an EVE ID with its name, for reading the file.
type NamedID struct {
	ID   int64  `json:"id" yaml:"id"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// CloneBay records whether a station or structure has a working clone bay,
// how neat knows, and when that was last checked.
type CloneBay struct {
	Status  string `json:"status" yaml:"status" jsonschema:"yes, no or unknown"`
	Source  string `json:"source,omitempty" yaml:"source,omitempty" jsonschema:"jump_clone (proved by a character's jump clone there) or user (the user checked in game)"`
	Checked string `json:"checked,omitempty" yaml:"checked,omitempty" jsonschema:"the date it was last checked, YYYY-MM-DD"`
}

// CorpHangar is an office one of the user's corporations has there.
type CorpHangar struct {
	Corporation NamedID `json:"corporation" yaml:"corporation"`
	OfficeID    int64   `json:"officeId,omitempty" yaml:"office_id,omitempty"`
}

// Structure is an NPC station or Upwell structure the user works from, with
// what the game won't tell neat about it.
type Structure struct {
	ID              int64        `json:"id" yaml:"id"`
	Name            string       `json:"name" yaml:"name"`
	Kind            string       `json:"kind" yaml:"kind" jsonschema:"npc_station or upwell"`
	Type            *NamedID     `json:"type,omitempty" yaml:"type,omitempty"`
	System          *NamedID     `json:"system,omitempty" yaml:"system,omitempty"`
	Owner           *NamedID     `json:"owner,omitempty" yaml:"owner,omitempty"`
	Uses            []string     `json:"uses,omitempty" yaml:"uses,omitempty" jsonschema:"what the user uses it for, such as manufacturing, reprocessing, reactions, market, staging or headquarters"`
	FacilityProfile string       `json:"facilityProfile,omitempty" yaml:"facility_profile,omitempty" jsonschema:"the facility profile (facilities.yaml) for industry here"`
	CloneBay        *CloneBay    `json:"cloneBay,omitempty" yaml:"clone_bay,omitempty"`
	CorpHangars     []CorpHangar `json:"corpHangars,omitempty" yaml:"corp_hangars,omitempty"`
	Notes           string       `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// Structure kinds.
const (
	StructureNPCStation = "npc_station"
	StructureUpwell     = "upwell"
)

// CloneBayStatus is the structure's clone bay status, unknown when unset.
func (s Structure) CloneBayStatus() string {
	if s.CloneBay == nil || s.CloneBay.Status == "" {
		return CloneBayUnknown
	}
	return s.CloneBay.Status
}

func structureKey(s Structure) string { return strconv.FormatInt(s.ID, 10) }

// Structure looks up a saved structure by ID.
func (c Config) Structure(id int64) (Structure, bool) {
	for _, s := range c.Structures {
		if s.ID == id {
			return s, true
		}
	}
	return Structure{}, false
}

// UpsertStructure adds or replaces a structure by ID.
func (c *Config) UpsertStructure(s Structure) {
	for i, e := range c.Structures {
		if e.ID == s.ID {
			c.Structures[i] = s
			return
		}
	}
	c.Structures = append(c.Structures, s)
}

// DeleteStructure removes a structure by ID.
func (c *Config) DeleteStructure(id int64) error {
	i := slices.IndexFunc(c.Structures, func(s Structure) bool { return s.ID == id })
	if i < 0 {
		return fmt.Errorf("no saved structure with ID %d", id)
	}
	c.Structures = slices.Delete(c.Structures, i, i+1)
	return nil
}

// ValidCloneBayStatus reports whether s is yes, no or unknown.
func ValidCloneBayStatus(s string) bool {
	return s == CloneBayYes || s == CloneBayNo || s == CloneBayUnknown
}
