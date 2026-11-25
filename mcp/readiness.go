package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// hullClassGroups are the ship groups list_character_capabilities and
// get_logistics_overview report flyable hulls for, and compare_haul_hulls
// compares by default: the ones that matter when moving cargo, capitals and
// fleets. This picks what to show, not a game rule.
var hullClassGroups = []int64{
	28,   // Hauler
	902,  // Jump Freighter
	513,  // Freighter
	380,  // Deep Space Transport
	1202, // Blockade Runner
	941,  // Industrial Command Ship
	4902, // Expedition Command Ship
	883,  // Capital Industrial Ship
	898,  // Black Ops
	547,  // Carrier
	5120, // Command Carrier
	485,  // Dreadnought
	4594, // Lancer Dreadnought
	1538, // Force Auxiliary
	659,  // Supercarrier
	30,   // Titan
}

// requiredSkills reads a type's required skills from its dogma attributes.
func requiredSkills(attrs map[int64]float64) []nav.SkillReq {
	var out []nav.SkillReq
	for _, pair := range sde.AttrRequiredSkills {
		if id := int64(attrs[pair[0]]); id != 0 {
			out = append(out, nav.SkillReq{SkillID: id, Level: int(attrs[pair[1]])})
		}
	}
	return out
}

// hulls turns SDE types into hulls with their required skills.
func (p *planner) hulls(ctx context.Context, types []sde.TypeBasic) ([]nav.Hull, error) {
	out := make([]nav.Hull, 0, len(types))
	for _, t := range types {
		attrs, err := p.sde.TypeAttributes(ctx, p.ua(), t.TypeID)
		if err != nil {
			return nil, fmt.Errorf("attributes of %s: %w", t.Name, err)
		}
		out = append(out, nav.Hull{TypeID: t.TypeID, Name: t.Name, GroupID: t.GroupID, Requires: requiredSkills(attrs)})
	}
	return out, nil
}

// moduleAbility builds an ability from a module's own skill requirements
// and the ship groups and types it can be fitted to, all from the SDE.
func (p *planner) moduleAbility(ctx context.Context, kind, variant string, moduleID int64) (nav.Ability, error) {
	attrs, err := p.sde.TypeAttributes(ctx, p.ua(), moduleID)
	if err != nil {
		return nav.Ability{}, fmt.Errorf("attributes of module %d: %w", moduleID, err)
	}
	groups, types := fitList(attrs)
	inGroups, err := p.sde.PublishedTypesInGroups(ctx, p.ua(), groups)
	if err != nil {
		return nav.Ability{}, fmt.Errorf("hulls that module %d fits by group: %w", moduleID, err)
	}
	extra, err := p.typeBasics(ctx, types)
	if err != nil {
		return nav.Ability{}, fmt.Errorf("hulls that module %d fits by type: %w", moduleID, err)
	}
	seen := make(map[int64]bool, len(inGroups))
	for _, b := range inGroups {
		seen[b.TypeID] = true
	}
	for _, id := range types {
		if b, ok := extra[id]; ok && b.Published && !seen[id] {
			seen[id] = true
			inGroups = append(inGroups, b)
		}
	}
	hulls, err := p.hulls(ctx, inGroups)
	if err != nil {
		return nav.Ability{}, fmt.Errorf("hulls for module %d: %w", moduleID, err)
	}
	return nav.Ability{Kind: kind, Variant: variant, Module: p.typeName(ctx, moduleID), Requires: requiredSkills(attrs), Hulls: hulls}, nil
}

// abilityCatalog lists every cyno, bridge and conduit ability, built from
// the SDE.
func (p *planner) abilityCatalog(ctx context.Context) ([]nav.Ability, error) {
	specs := []struct {
		kind, variant string
		module        int64
	}{
		{"cyno", "standard", nav.StandardCynoGenerator},
		{"cyno", "industrial", nav.IndustrialCynoGenerator},
		{"cyno", "covert", nav.CovertCynoGenerator},
		{"bridge", "covert", nav.CovertPortalGenerator},
		{"bridge", "industrial", nav.IndustrialPortalGen},
		{"bridge", "titan", nav.TitanPortalGenerator},
		{"conduit", "covert", nav.CovertPortalGenerator},
		{"conduit", "industrial", nav.IndustrialPortalGen},
	}
	var out []nav.Ability
	for _, s := range specs {
		a, err := p.moduleAbility(ctx, s.kind, s.variant, s.module)
		if err != nil {
			return nil, fmt.Errorf("%s %s ability: %w", s.variant, s.kind, err)
		}
		out = append(out, a)
	}
	carriers, err := p.sde.PublishedTypesWithEffect(ctx, p.ua(), nav.CapitalConduitEffect)
	if err != nil {
		return nil, fmt.Errorf("carrier conduit hulls: %w", err)
	}
	hulls, err := p.hulls(ctx, carriers)
	if err != nil {
		return nil, fmt.Errorf("carrier conduit hulls: %w", err)
	}
	out = append(out, nav.Ability{Kind: "conduit", Variant: "carrier", Module: "built into the hull", Requires: []nav.SkillReq{{SkillID: nav.SkillCapitalJumpPortalGeneration, Level: 1}}, Hulls: hulls})
	return out, nil
}

// skillLabel names a skill requirement, such as "Cynosural Field Theory V".
func (p *planner) skillLabel(ctx context.Context, r nav.SkillReq) string {
	return fmt.Sprintf("%s %s", p.typeName(ctx, r.SkillID), romanLevel(r.Level))
}

func romanLevel(l int) string {
	if l >= 1 && l <= 5 {
		return []string{"I", "II", "III", "IV", "V"}[l-1]
	}
	return fmt.Sprint(l)
}

// targetCharacters is the characters a tool runs for: the ones asked for,
// or every logged-in character.
func targetCharacters(cfg config.Config, ids []int32) ([]config.Character, error) {
	if len(ids) == 0 {
		return cfg.Characters, nil
	}
	out := make([]config.Character, 0, len(ids))
	seen := make(map[int32]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		c, ok := cfg.Character(id)
		if !ok {
			return nil, fmt.Errorf("character %d: %w", id, ErrCharacterNotFound)
		}
		out = append(out, c)
	}
	return out, nil
}

// --- list_character_capabilities ---

// ListCharacterCapabilitiesInput is list_character_capabilities' input.
type ListCharacterCapabilitiesInput struct {
	CharacterIDs   []int32 `json:"characterIds,omitempty" jsonschema:"omit for every logged-in character"`
	IncludeMissing bool    `json:"includeMissing,omitempty" jsonschema:"also list abilities a character doesn't have yet, with the module skills they still need"`
}

// AbilityView is one cyno, bridge or conduit ability for a character.
type AbilityView struct {
	Kind          string   `json:"kind" jsonschema:"cyno, bridge or conduit"`
	Variant       string   `json:"variant" jsonschema:"standard, industrial, covert, titan or carrier"`
	Module        string   `json:"module"`
	Can           bool     `json:"can"`
	Hulls         []string `json:"hulls,omitempty" jsonschema:"hulls they can fly that this works on"`
	MissingSkills []string `json:"missingSkills,omitempty" jsonschema:"module skills still needed"`
}

// HullClass is the hulls of one ship group a character can fly.
type HullClass struct {
	Class string   `json:"class"`
	Hulls []string `json:"hulls"`
}

// CharacterCapabilities is what one character can do.
type CharacterCapabilities struct {
	CharacterID   int32         `json:"characterId"`
	CharacterName string        `json:"characterName"`
	Account       string        `json:"account,omitempty"`
	Abilities     []AbilityView `json:"abilities"`
	Hulls         []HullClass   `json:"hulls" jsonschema:"hauling, jump-capable and capital hulls they can fly, by class"`
}

// ListCharacterCapabilitiesOutput is list_character_capabilities' result.
type ListCharacterCapabilitiesOutput struct {
	Characters []CharacterCapabilities `json:"characters"`
	Warnings   []string                `json:"warnings,omitempty"`
}

func listCharacterCapabilitiesHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListCharacterCapabilitiesInput, ListCharacterCapabilitiesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterCapabilitiesInput) (*mcpsdk.CallToolResult, ListCharacterCapabilitiesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCharacterCapabilitiesOutput{}, err
		}
		chars, err := targetCharacters(cfg, in.CharacterIDs)
		if err != nil {
			return nil, ListCharacterCapabilitiesOutput{}, err
		}
		p := newPlanner(version, clientID, store, sdeStore)
		catalog, err := p.abilityCatalog(ctx)
		if err != nil {
			return nil, ListCharacterCapabilitiesOutput{}, fmt.Errorf("ability catalog: %w", err)
		}
		classTypes, err := sdeStore.PublishedTypesInGroups(ctx, p.ua(), hullClassGroups)
		if err != nil {
			return nil, ListCharacterCapabilitiesOutput{}, fmt.Errorf("hull classes: %w", err)
		}
		classHulls, err := p.hulls(ctx, classTypes)
		if err != nil {
			return nil, ListCharacterCapabilitiesOutput{}, fmt.Errorf("hull class requirements: %w", err)
		}
		groupName := map[int64]string{}
		for _, t := range classTypes {
			groupName[t.GroupID] = t.GroupName
		}

		out := ListCharacterCapabilitiesOutput{Characters: []CharacterCapabilities{}}
		for _, c := range chars {
			skills, err := p.skills(ctx, c.ID)
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("couldn't read %s's skills: %v", c.Name, err))
				continue
			}
			cc := CharacterCapabilities{CharacterID: c.ID, CharacterName: c.Name, Account: c.Account, Abilities: []AbilityView{}, Hulls: []HullClass{}}
			for _, r := range nav.Abilities(skills, catalog) {
				if !r.Can && !in.IncludeMissing {
					continue
				}
				v := AbilityView{Kind: r.Kind, Variant: r.Variant, Module: r.Module, Can: r.Can, Hulls: r.Hulls}
				for _, m := range r.Missing {
					v.MissingSkills = append(v.MissingSkills, p.skillLabel(ctx, m))
				}
				cc.Abilities = append(cc.Abilities, v)
			}
			byGroup := map[int64][]string{}
			for _, h := range nav.FlyableHulls(skills, classHulls) {
				byGroup[h.GroupID] = append(byGroup[h.GroupID], h.Name)
			}
			for _, g := range hullClassGroups {
				if len(byGroup[g]) > 0 {
					cc.Hulls = append(cc.Hulls, HullClass{Class: groupName[g], Hulls: byGroup[g]})
				}
			}
			out.Characters = append(out.Characters, cc)
		}
		return nil, out, nil
	}
}

// --- list_jump_clones ---

// ListJumpClonesInput is list_jump_clones' input.
type ListJumpClonesInput struct {
	CharacterIDs []int32 `json:"characterIds,omitempty" jsonschema:"omit for every logged-in character"`
}

// CloneLocation is where a jump clone or home station is.
type CloneLocation struct {
	LocationID   int64    `json:"locationId"`
	LocationName string   `json:"locationName,omitempty"`
	LocationType string   `json:"locationType,omitempty" jsonschema:"station or structure"`
	Implants     []string `json:"implants,omitempty"`
}

// CharacterClones is one character's clone jump readiness.
type CharacterClones struct {
	CharacterID    int32           `json:"characterId"`
	CharacterName  string          `json:"characterName"`
	Account        string          `json:"account,omitempty"`
	CooldownHours  int             `json:"cooldownHours" jsonschema:"hours between clone jumps for their Infomorph Synchronizing"`
	LastCloneJump  *time.Time      `json:"lastCloneJump,omitempty"`
	NextCloneJump  *time.Time      `json:"nextCloneJump,omitempty" jsonschema:"when they can clone jump again; absent if they never have"`
	CooldownReady  bool            `json:"cooldownReady" jsonschema:"their clone jump cooldown has passed; whether they can jump from where they are is jumpFromHere"`
	ReadyNow       bool            `json:"readyNow" jsonschema:"true only when the cooldown has passed and they're docked where a clone bay is known to be (jumpFromHere yes)"`
	JumpFromHere   JumpFromHere    `json:"jumpFromHere" jsonschema:"whether they can start a clone jump where they are now"`
	Capacity       int             `json:"capacity" jsonschema:"how many jump clones their Infomorph Psychology skills allow"`
	Home           *CloneLocation  `json:"home,omitempty" jsonschema:"where they wake up after being podded"`
	ActiveImplants *[]string       `json:"activeImplants,omitempty" jsonschema:"implants in the clone they're in now, which they lose if podded; an empty list means none, and absent means they couldn't be read (see warnings), not that there are none"`
	JumpClones     []CloneLocation `json:"jumpClones"`
}

// Statuses for JumpFromHere.
const (
	jumpFromYes       = "yes"
	jumpFromNo        = "no"
	jumpFromUnknown   = "unknown"
	jumpFromNotDocked = "not_docked"
)

// JumpFromHere says whether a character can start a clone jump where they
// are now: a clone jump starts docked at a station or structure with a
// clone bay.
type JumpFromHere struct {
	Status       string `json:"status" jsonschema:"yes: a clone bay is known to be here; no: the user found none here; unknown: neat can't tell, so ask the user to check in game; not_docked: they're in space. A location that couldn't be read is unknown, with a warning"`
	LocationID   int64  `json:"locationId,omitempty" jsonschema:"the station or structure they're docked in"`
	LocationName string `json:"locationName,omitempty"`
	Reason       string `json:"reason"`
}

// provenCloneBays is every place a logged-in character has a jump clone,
// which proves a clone bay there.
func provenCloneBays(clones ...GetCharacterClonesOutput) map[int64]bool {
	proven := map[int64]bool{}
	for _, c := range clones {
		for _, jc := range c.JumpClones {
			proven[jc.LocationID] = true
		}
	}
	return proven
}

// otherCharactersClones reads the jump clones of every logged-in character
// not in chars, since any character's jump clone proves a clone bay even
// when only some characters were asked about. A character whose clones
// can't be read is named in the returned warnings.
func otherCharactersClones(ctx context.Context, clientID, userAgent string, cfg config.Config, chars []config.Character) ([]GetCharacterClonesOutput, []string) {
	asked := make(map[int32]bool, len(chars))
	for _, c := range chars {
		asked[c.ID] = true
	}
	var clones []GetCharacterClonesOutput
	var warnings []string
	for _, c := range cfg.Characters {
		if asked[c.ID] {
			continue
		}
		cl, err := getCharacterClones(ctx, clientID, c.ID, userAgent)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("couldn't read %s's jump clones, so clone bays they'd prove aren't counted: %v", c.Name, err))
			continue
		}
		clones = append(clones, cl)
	}
	return clones, warnings
}

// jumpFromHere says whether a clone jump can start at dockedAt (0 when in
// space), from a jump clone there or the clone bay saved in structures.yaml.
func (p *planner) jumpFromHere(ctx context.Context, cfg config.Config, dockedAt int64, proven map[int64]bool) JumpFromHere {
	if dockedAt == 0 {
		return JumpFromHere{Status: jumpFromNotDocked, Reason: "not docked: a clone jump has to start docked where there's a clone bay"}
	}
	j := JumpFromHere{LocationID: dockedAt, LocationName: p.placeName(ctx, cfg, dockedAt)}
	saved, ok := cfg.Structure(dockedAt)
	switch {
	case proven[dockedAt]:
		j.Status, j.Reason = jumpFromYes, "a logged-in character has a jump clone here, so it has a clone bay"
	case ok && saved.CloneBayStatus() == config.CloneBayYes:
		j.Status, j.Reason = jumpFromYes, "saved in structures.yaml as having a clone bay"
		if saved.CloneBay.Checked != "" {
			j.Reason += ", checked " + saved.CloneBay.Checked
		}
	case ok && saved.CloneBayStatus() == config.CloneBayNo:
		j.Status, j.Reason = jumpFromNo, "saved in structures.yaml as having no clone bay, so they can't clone jump from here: they have to move to a station or structure with one first"
		if saved.CloneBay.Checked != "" {
			j.Reason += " (checked " + saved.CloneBay.Checked + ")"
		}
	default:
		j.Status, j.Reason = jumpFromUnknown, "neat doesn't know whether this has a clone bay: ask the user to check in game, and save the answer with save_structure"
	}
	return j
}

// ListJumpClonesOutput is list_jump_clones' result.
type ListJumpClonesOutput struct {
	Characters []CharacterClones `json:"characters"`
	Warnings   []string          `json:"warnings,omitempty"`
}

// cloneLocation names a station from the SDE or a structure from
// structures.yaml; an unsaved structure keeps only its ID.
func (p *planner) cloneLocation(ctx context.Context, cfg config.Config, id int64, kind string, implants []int64) CloneLocation {
	l := CloneLocation{LocationID: id, LocationType: kind}
	if s, ok := cfg.Structure(id); ok {
		l.LocationName = s.Name
	} else if names := resolveStationNames(ctx, p.sde, p.ua(), []int64{id}); names != nil {
		l.LocationName = names[id]
	}
	for _, imp := range implants {
		l.Implants = append(l.Implants, p.typeName(ctx, imp))
	}
	return l
}

func listJumpClonesHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListJumpClonesInput, ListJumpClonesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListJumpClonesInput) (*mcpsdk.CallToolResult, ListJumpClonesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListJumpClonesOutput{}, err
		}
		chars, err := targetCharacters(cfg, in.CharacterIDs)
		if err != nil {
			return nil, ListJumpClonesOutput{}, err
		}
		p := newPlanner(version, clientID, store, sdeStore)
		out := ListJumpClonesOutput{Characters: []CharacterClones{}}
		var allClones []GetCharacterClonesOutput
		docked := map[int32]int64{} // character → where they're docked
		located := map[int32]bool{}
		for _, c := range chars {
			clones, err := getCharacterClones(ctx, clientID, c.ID, p.ua())
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("couldn't read %s's clones: %v", c.Name, err))
				continue
			}
			allClones = append(allClones, clones)
			if _, st, sr, err := getCharacterLocation(ctx, clientID, c.ID, p.ua()); err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("couldn't read %s's location, so whether they can clone jump from there is unknown: %v", c.Name, err))
			} else {
				located[c.ID] = true
				if st != nil {
					docked[c.ID] = *st
				} else if sr != nil {
					docked[c.ID] = *sr
				}
			}
			skills, err := p.skills(ctx, c.ID)
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("couldn't read %s's skills, so their cooldown and clone capacity assume untrained skills: %v", c.Name, err))
				skills = map[int64]int{}
			}
			sync := skills[nav.SkillInfomorphSynchronizing]
			cc := CharacterClones{CharacterID: c.ID, CharacterName: c.Name, Account: c.Account, JumpClones: []CloneLocation{},
				CooldownHours: int(nav.CloneJumpCooldown(sync).Hours()),
				Capacity:      nav.JumpCloneCapacity(skills[nav.SkillInfomorphPsychology], skills[nav.SkillAdvancedInfomorphPsychology], skills[nav.SkillEliteInfomorphPsychology]),
				CooldownReady: true}
			if clones.LastCloneJumpDate != nil {
				last := *clones.LastCloneJumpDate
				next := nav.NextCloneJump(last, sync)
				cc.LastCloneJump, cc.NextCloneJump = &last, &next
				cc.CooldownReady = !now().Before(next)
			}
			if clones.HomeLocationID != nil {
				kind := ""
				if clones.HomeLocationType != nil {
					kind = *clones.HomeLocationType
				}
				home := p.cloneLocation(ctx, cfg, *clones.HomeLocationID, kind, nil)
				cc.Home = &home
			}
			for _, jc := range clones.JumpClones {
				cc.JumpClones = append(cc.JumpClones, p.cloneLocation(ctx, cfg, jc.LocationID, jc.LocationType, jc.Implants))
			}
			if implants, err := listCharacterImplants(ctx, clientID, c.ID, p.ua()); err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("couldn't read the implants in %s's current clone, so they're unknown, not none: %v", c.Name, err))
			} else {
				names := []string{}
				for _, imp := range implants {
					names = append(names, p.typeName(ctx, imp))
				}
				cc.ActiveImplants = &names
			}
			out.Characters = append(out.Characters, cc)
		}
		// Every character's jump clones are read before any clone bay is
		// judged, since one character's clone proves a bay for all, including
		// characters that weren't asked about.
		others, warnings := otherCharactersClones(ctx, clientID, p.ua(), cfg, chars)
		out.Warnings = append(out.Warnings, warnings...)
		proven := provenCloneBays(append(allClones, others...)...)
		for i := range out.Characters {
			cc := &out.Characters[i]
			if !located[cc.CharacterID] {
				cc.JumpFromHere = JumpFromHere{Status: jumpFromUnknown, Reason: "couldn't read their location (see warnings)"}
			} else {
				cc.JumpFromHere = p.jumpFromHere(ctx, cfg, docked[cc.CharacterID], proven)
			}
			cc.ReadyNow = cc.CooldownReady && cc.JumpFromHere.Status == jumpFromYes
		}
		return nil, out, nil
	}
}
