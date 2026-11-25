package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// today is overridable in tests.
var today = func() string { return time.Now().UTC().Format("2006-01-02") }

// registerStructureTools adds the tools for the stations and structures the
// user works from (~/.neat/structures.yaml). See
// docs/superpowers/specs/2026-11-20-logistics-planning-design.md.
func registerStructureTools(server *mcpsdk.Server, version, clientID string, store *config.Store, sdeStore *sde.Store) {
	add := func(name, desc string) *mcpsdk.Tool { return &mcpsdk.Tool{Name: name, Description: desc} }
	mcpsdk.AddTool(server, add("list_structures", fmt.Sprintf("List the NPC stations and Upwell structures the user works from (~/.neat/structures.yaml), with what the game won't tell neat: what they use each for, whether it has a working clone bay and how that's known, their corporation offices, and their notes. Each call checks every logged-in character's jump clones and records any clone bay they prove; a clone bay neat can't prove stays unknown, and only the user can check it in game. Pass characterIds to check, live, which characters have docking access to each Upwell structure: one ESI lookup per Upwell structure and character, and ESI counts every refused lookup (usually no docking access) against its error limit, so after %d refusals in one call the rest are returned as access not_checked with a warning. Pass only the characters you need, and don't call again to fill in not_checked entries for characters that were already refused. Use it before planning a clone jump, a staging move or anything that needs to dock somewhere.", maxDockingRefusals)), listStructuresHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("save_structure", "Save an NPC station or Upwell structure the user works from, or update one. Its name, type, system and owner are filled in from the game (an Upwell structure needs a character with docking access the first time). Record what the user tells you: what they use it for, its facility profile, a clone bay they checked in game (cloneBay yes or no), and notes. Offer to save a location whenever a plan uses one that isn't saved yet. Returns every value it changed, old and new, so you can tell the user what was recorded. Writes to ~/.neat/structures.yaml only."), saveStructureHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("delete_structure", "Delete a saved station or structure from ~/.neat/structures.yaml."), deleteStructureHandler(store))
}

// SaveStructureInput is save_structure's input.
type SaveStructureInput struct {
	ID              int64    `json:"id" jsonschema:"the NPC station or Upwell structure ID"`
	CharacterID     int32    `json:"characterId,omitempty" jsonschema:"a logged-in character with docking access, to read an Upwell structure's name, type, system and owner; needed the first time an Upwell structure is saved"`
	Uses            []string `json:"uses,omitempty" jsonschema:"what the user uses it for, such as manufacturing, reprocessing, reactions, market, staging or headquarters; replaces the saved list, and omitting it keeps the list"`
	FacilityProfile *string  `json:"facilityProfile,omitempty" jsonschema:"the facility profile for industry here; an empty string clears it"`
	CloneBay        string   `json:"cloneBay,omitempty" jsonschema:"what the user found in game: yes, no or unknown. Only set this from the user's own check; neat records clone bays it can prove itself"`
	Notes           *string  `json:"notes,omitempty" jsonschema:"what the user says about this place; omit to keep the notes it has"`
}

// SaveStructureOutput is the saved entry and every value the save changed.
type SaveStructureOutput struct {
	Structure config.Structure `json:"structure"`
	Changes   []config.Change  `json:"changes"`
	Warnings  []string         `json:"warnings,omitempty"`
}

// structureFacts reads what the game says about a station or structure.
func structureFacts(ctx context.Context, clientID, ua string, sdeStore *sde.Store, id int64, characterID int32) (config.Structure, error) {
	if st, ok, err := sdeStore.Station(ctx, ua, id); err != nil {
		return config.Structure{}, fmt.Errorf("look up station %d: %w", id, err)
	} else if ok {
		s := config.Structure{ID: id, Name: st.Name, Kind: config.StructureNPCStation, System: &config.NamedID{ID: st.SystemID}, Type: &config.NamedID{ID: st.TypeID}}
		if st.Owner != nil {
			s.Owner = &config.NamedID{ID: *st.Owner}
		}
		return s, nil
	}
	if characterID == 0 {
		return config.Structure{}, fmt.Errorf("%d isn't an NPC station, so it's read as an Upwell structure, which needs characterId: a logged-in character with docking access", id)
	}
	us, err := getUniverseStructure(ctx, clientID, characterID, ua, id)
	if err != nil {
		return config.Structure{}, fmt.Errorf("read structure %d through character %d (it needs docking access and the structures scope group): %w", id, characterID, err)
	}
	s := config.Structure{ID: id, Name: us.Name, Kind: config.StructureUpwell, System: &config.NamedID{ID: us.SolarSystemID}, Owner: &config.NamedID{ID: us.OwnerID}}
	if us.TypeID != nil {
		s.Type = &config.NamedID{ID: *us.TypeID}
	}
	return s, nil
}

// nameStructure fills in type, system and owner names. Names are for
// reading the file, so a lookup that fails leaves the name out.
func nameStructure(ctx context.Context, p *planner, s *config.Structure) {
	ua := p.ua()
	if s.Type != nil {
		if b, err := p.typeBasics(ctx, []int64{s.Type.ID}); err == nil {
			s.Type.Name = b[s.Type.ID].Name
		}
	}
	if s.System != nil {
		if sys, err := p.system(ctx, s.System.ID); err == nil {
			s.System.Name = sys.Name
		}
	}
	if s.Owner != nil {
		if names, err := resolveUniverseNames(ctx, ua, []int64{s.Owner.ID}); err == nil && len(names) == 1 {
			s.Owner.Name = names[0].Name
		}
	}
}

// keepNames keeps a saved name wherever a fresh lookup of the same ID
// failed, so a transient failure doesn't blank names in the file.
func keepNames(facts *config.Structure, saved config.Structure) {
	keep := func(n, old *config.NamedID) {
		if n != nil && n.Name == "" && old != nil && old.ID == n.ID {
			n.Name = old.Name
		}
	}
	keep(facts.Type, saved.Type)
	keep(facts.System, saved.System)
	keep(facts.Owner, saved.Owner)
}

func saveStructureHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[SaveStructureInput, SaveStructureOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveStructureInput) (*mcpsdk.CallToolResult, SaveStructureOutput, error) {
		if in.ID == 0 {
			return nil, SaveStructureOutput{}, fmt.Errorf("save structure: id is required")
		}
		if in.CloneBay != "" && !config.ValidCloneBayStatus(in.CloneBay) {
			return nil, SaveStructureOutput{}, fmt.Errorf("save structure: cloneBay must be yes, no or unknown, not %q", in.CloneBay)
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, SaveStructureOutput{}, err
		}
		if in.CharacterID != 0 {
			if _, ok := cfg.Character(in.CharacterID); !ok {
				return nil, SaveStructureOutput{}, fmt.Errorf("save structure through character %d: %w", in.CharacterID, ErrCharacterNotFound)
			}
		}
		if in.FacilityProfile != nil && strings.TrimSpace(*in.FacilityProfile) != "" {
			if _, ok := cfg.FacilityProfile(*in.FacilityProfile); !ok {
				return nil, SaveStructureOutput{}, fmt.Errorf("save structure: no facility profile named %q", *in.FacilityProfile)
			}
		}
		ua := sso.UserAgent(version)
		var warnings []string
		facts, factsErr := structureFacts(ctx, clientID, ua, sdeStore, in.ID, in.CharacterID)
		if factsErr == nil {
			nameStructure(ctx, newPlanner(version, clientID, store, sdeStore), &facts)
		}
		// The lookups above go over the network, so the entry is merged into
		// whatever is saved now, under Update's lock, rather than saved over
		// anything written meanwhile.
		var s config.Structure
		changes, err := store.Update(func(cfg *config.Config) error {
			var saved bool
			s, saved = cfg.Structure(in.ID)
			switch {
			case factsErr == nil:
				keepNames(&facts, s)
				s.ID, s.Name, s.Kind, s.Type, s.System, s.Owner = facts.ID, facts.Name, facts.Kind, facts.Type, facts.System, facts.Owner
			case saved:
				warnings = append(warnings, "kept the saved name, type, system and owner: "+factsErr.Error())
			default:
				return fmt.Errorf("save structure: %w", factsErr)
			}
			if in.Uses != nil {
				s.Uses = nil
				for _, u := range in.Uses {
					if u = strings.TrimSpace(u); u != "" {
						s.Uses = append(s.Uses, u)
					}
				}
			}
			if in.FacilityProfile != nil {
				s.FacilityProfile = strings.TrimSpace(*in.FacilityProfile)
				if p, ok := cfg.FacilityProfile(s.FacilityProfile); ok {
					s.FacilityProfile = p.Name
				}
			}
			if in.CloneBay != "" {
				s.CloneBay = &config.CloneBay{Status: in.CloneBay, Source: config.CloneBayFromUser, Checked: today()}
				if in.CloneBay == config.CloneBayUnknown {
					s.CloneBay = nil
				}
			}
			if in.Notes != nil {
				s.Notes = strings.TrimSpace(*in.Notes)
			}
			cfg.UpsertStructure(s)
			return nil
		})
		if err != nil {
			return nil, SaveStructureOutput{}, err
		}
		if changes == nil {
			changes = []config.Change{}
		}
		return nil, SaveStructureOutput{Structure: s, Changes: changes, Warnings: warnings}, nil
	}
}

// ListStructuresInput is list_structures' input.
type ListStructuresInput struct {
	CharacterIDs []int32 `json:"characterIds,omitempty" jsonschema:"check, live, whether these characters can dock at each Upwell structure; omit to skip the check. Checking stops after 10 refused lookups in one call, so pass only the characters you need"`
}

// maxDockingRefusals is how many refused docking lookups one list_structures
// call makes before it stops asking ESI. list_structures' description uses
// it; its input schema and DockingCheck.Access spell out 10, so change them
// with it.
const maxDockingRefusals = 10

// DockingCheck is whether one character can dock at a structure.
type DockingCheck struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
	Access        string `json:"access" jsonschema:"yes; unknown when ESI refused the lookup, which usually means no docking access (the detail says why); or not_checked when this call had already had 10 lookups refused and stopped asking ESI"`
	Detail        string `json:"detail,omitempty"`
}

// StructureView is a saved structure with the live checks.
type StructureView struct {
	config.Structure
	Docking []DockingCheck `json:"docking,omitempty"`
}

// ListStructuresOutput is list_structures' result.
type ListStructuresOutput struct {
	Structures []StructureView `json:"structures"`
	Changes    []config.Change `json:"changes,omitempty" jsonschema:"clone bays this call proved and recorded"`
	Warnings   []string        `json:"warnings,omitempty"`
}

func listStructuresHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListStructuresInput, ListStructuresOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListStructuresInput) (*mcpsdk.CallToolResult, ListStructuresOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListStructuresOutput{}, err
		}
		for _, id := range in.CharacterIDs {
			if _, ok := cfg.Character(id); !ok {
				return nil, ListStructuresOutput{}, fmt.Errorf("check docking for character %d: %w", id, ErrCharacterNotFound)
			}
		}
		ua := sso.UserAgent(version)
		out := ListStructuresOutput{Structures: []StructureView{}}

		// A jump clone of any logged-in character proves a clone bay.
		proved := map[int64]string{}
		for _, c := range cfg.Characters {
			clones, err := getCharacterClones(ctx, clientID, c.ID, ua)
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("couldn't read %s's jump clones, so their clone bays weren't checked: %v", c.Name, err))
				continue
			}
			for _, jc := range clones.JumpClones {
				if _, ok := proved[jc.LocationID]; !ok {
					proved[jc.LocationID] = c.Name
				}
			}
		}
		// The clone lookups above go over the network, so proved clone bays
		// are recorded on whatever is saved now, under Update's lock, rather
		// than saved over anything written meanwhile.
		changes, err := store.Update(func(c *config.Config) error {
			for i, s := range c.Structures {
				if _, ok := proved[s.ID]; ok {
					bay := &config.CloneBay{Status: config.CloneBayYes, Source: config.CloneBayFromJumpClone, Checked: today()}
					if s.CloneBay == nil || *s.CloneBay != *bay {
						c.Structures[i].CloneBay = bay
					}
				}
			}
			cfg = *c
			return nil
		})
		if err != nil {
			return nil, ListStructuresOutput{}, err
		}
		out.Changes = changes

		// Every refused lookup counts against ESI's error limit, so stop
		// asking after a few rather than one per structure and character.
		refused := 0
		for _, s := range cfg.Structures {
			v := StructureView{Structure: s}
			if s.Kind == config.StructureUpwell {
				for _, id := range in.CharacterIDs {
					c, ok := cfg.Character(id)
					if !ok {
						continue // logged out since the check above
					}
					check := DockingCheck{CharacterID: id, CharacterName: c.Name, Access: "yes"}
					if refused >= maxDockingRefusals {
						check.Access, check.Detail = "not_checked", fmt.Sprintf("not checked: ESI had already refused %d docking lookups in this call", maxDockingRefusals)
					} else if _, err := getUniverseStructure(ctx, clientID, id, ua, s.ID); err != nil {
						check.Access, check.Detail = "unknown", err.Error()
						if _, ok := errors.AsType[*esi.MissingScopeError](err); !ok {
							refused++
						}
					}
					v.Docking = append(v.Docking, check)
				}
			}
			out.Structures = append(out.Structures, v)
		}
		if refused >= maxDockingRefusals {
			out.Warnings = append(out.Warnings, fmt.Sprintf("stopped checking docking after ESI refused %d lookups, since each refusal counts against ESI's error limit; the remaining checks are not_checked. Ask the user which character should have access to the structures that matter rather than checking every character again", refused))
		}
		return nil, out, nil
	}
}

// DeleteStructureInput is delete_structure's input.
type DeleteStructureInput struct {
	ID int64 `json:"id" jsonschema:"the saved station or structure's ID"`
}

func deleteStructureHandler(store *config.Store) mcpsdk.ToolHandlerFor[DeleteStructureInput, DeletedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteStructureInput) (*mcpsdk.CallToolResult, DeletedOutput, error) {
		var name string
		if _, err := store.Update(func(cfg *config.Config) error {
			s, _ := cfg.Structure(in.ID)
			name = s.Name
			return cfg.DeleteStructure(in.ID)
		}); err != nil {
			return nil, DeletedOutput{}, err
		}
		return nil, DeletedOutput{Deleted: name}, nil
	}
}
