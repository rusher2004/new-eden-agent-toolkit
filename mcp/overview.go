package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// GetLogisticsOverviewInput is get_logistics_overview's input.
type GetLogisticsOverviewInput struct {
	CharacterIDs []int32 `json:"characterIds,omitempty" jsonschema:"omit for every logged-in character"`
}

// OverviewLocation is where a character is.
type OverviewLocation struct {
	System          string `json:"system"`
	SecurityDisplay string `json:"securityDisplay"`
	SecurityColor   string `json:"securityColor"`
	DockedAt        string `json:"dockedAt,omitempty" jsonschema:"station or structure they're docked in; absent in space"`
	DockedAtID      int64  `json:"dockedAtId,omitempty"`
}

// OverviewShip is the ship a character is in.
type OverviewShip struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	ItemID int64  `json:"itemId"`
}

// OverviewClones is a character's clone jump readiness in brief.
type OverviewClones struct {
	ReadyNow      bool       `json:"readyNow"`
	NextCloneJump *time.Time `json:"nextCloneJump,omitempty"`
	JumpClonesAt  []string   `json:"jumpClonesAt" jsonschema:"where their jump clones are; list_jump_clones has the implants"`
}

// OverviewFatigue is a character's jump fatigue.
type OverviewFatigue struct {
	Fatigued  bool       `json:"fatigued"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	LastJump  *time.Time `json:"lastJump,omitempty"`
}

// OverviewCharacter is everything a logistics plan starts from, for one
// character. A part that couldn't be read is absent and named in Unread.
type OverviewCharacter struct {
	CharacterID   int32             `json:"characterId"`
	CharacterName string            `json:"characterName"`
	Account       string            `json:"account,omitempty" jsonschema:"the user's label for their EVE account; absent means unknown"`
	Online        *bool             `json:"online,omitempty"`
	Location      *OverviewLocation `json:"location,omitempty"`
	Ship          *OverviewShip     `json:"ship,omitempty"`
	Abilities     []string          `json:"abilities" jsonschema:"cynos, bridges and conduits they can use, with the hulls they can fly for each; list_character_capabilities has the detail"`
	Hulls         []HullClass       `json:"hulls" jsonschema:"hauling, jump-capable and capital hulls they can fly"`
	Clones        *OverviewClones   `json:"clones,omitempty"`
	Fatigue       *OverviewFatigue  `json:"fatigue,omitempty"`
	Unread        []string          `json:"unread,omitempty" jsonschema:"parts ESI wouldn't give, and why; usually a scope group the character hasn't granted"`
}

// AccountGroup is the characters sharing one account label.
type AccountGroup struct {
	Account    string   `json:"account"`
	Characters []string `json:"characters"`
}

// GetLogisticsOverviewOutput is get_logistics_overview's result.
type GetLogisticsOverviewOutput struct {
	Characters []OverviewCharacter `json:"characters"`
	Accounts   []AccountGroup      `json:"accounts" jsonschema:"characters by account: characters in one account can't be online together"`
	Conflicts  []string            `json:"conflicts,omitempty" jsonschema:"account labels the game contradicts: two characters with the same label are online at once, so they can't share an account"`
	Warnings   []string            `json:"warnings,omitempty"`
}

// overviewReads is one character's ESI reads, fetched together.
type overviewReads struct {
	systemID, stationID, structureID int64
	locErr                           error
	shipItem                         int64
	shipName                         string
	shipType                         int64
	shipErr                          error
	online                           *esi.CharacterOnlineStatus
	onlineErr                        error
	clones                           GetCharacterClonesOutput
	clonesErr                        error
	fatigue                          GetCharacterFatigueOutput
	fatigueErr                       error
	skills                           []CharacterSkill
	skillsErr                        error
}

// readOverview fetches one character's location, ship, online status,
// clones, fatigue and skills in parallel. These calls share no planner
// state, so they're safe to run together.
func readOverview(ctx context.Context, clientID, ua string, id int32) overviewReads {
	var r overviewReads
	var wg sync.WaitGroup
	run := func(f func()) {
		wg.Add(1)
		go func() { defer wg.Done(); f() }()
	}
	run(func() {
		var st, sr *int64
		r.systemID, st, sr, r.locErr = getCharacterLocation(ctx, clientID, id, ua)
		if st != nil {
			r.stationID = *st
		}
		if sr != nil {
			r.structureID = *sr
		}
	})
	run(func() { r.shipItem, r.shipName, r.shipType, r.shipErr = getCharacterShip(ctx, clientID, id, ua) })
	run(func() { r.online, r.onlineErr = getCharacterOnline(ctx, clientID, id, ua) })
	run(func() { r.clones, r.clonesErr = getCharacterClones(ctx, clientID, id, ua) })
	run(func() { r.fatigue, r.fatigueErr = getCharacterFatigue(ctx, clientID, id, ua) })
	run(func() { r.skills, _, _, r.skillsErr = listCharacterSkills(ctx, clientID, id, ua) })
	wg.Wait()
	return r
}

// placeName names a station from the SDE, or a structure from
// structures.yaml, falling back to its ID.
func (p *planner) placeName(ctx context.Context, cfg config.Config, id int64) string {
	if s, ok := cfg.Structure(id); ok {
		return s.Name
	}
	if names := resolveStationNames(ctx, p.sde, p.ua(), []int64{id}); names[id] != "" {
		return names[id]
	}
	return fmt.Sprintf("structure %d", id)
}

func getLogisticsOverviewHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetLogisticsOverviewInput, GetLogisticsOverviewOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetLogisticsOverviewInput) (*mcpsdk.CallToolResult, GetLogisticsOverviewOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, fmt.Errorf("load config: %w", err)
		}
		chars, err := targetCharacters(cfg, in.CharacterIDs)
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, err
		}
		p := newPlanner(version, clientID, store, sdeStore)
		catalog, err := p.abilityCatalog(ctx)
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, err
		}
		classTypes, err := sdeStore.PublishedTypesInGroups(ctx, p.ua(), hullClassGroups)
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, fmt.Errorf("hull classes: %w", err)
		}
		classHulls, err := p.hulls(ctx, classTypes)
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, err
		}
		groupName := map[int64]string{}
		for _, t := range classTypes {
			groupName[t.GroupID] = t.GroupName
		}

		out := GetLogisticsOverviewOutput{Characters: []OverviewCharacter{}, Accounts: []AccountGroup{}, Warnings: accountWarnings(cfg)}
		online := map[int32]bool{}
		for _, c := range chars {
			r := readOverview(ctx, clientID, p.ua(), c.ID)
			oc := OverviewCharacter{CharacterID: c.ID, CharacterName: c.Name, Account: c.Account, Abilities: []string{}, Hulls: []HullClass{}}
			unread := func(part string, err error) { oc.Unread = append(oc.Unread, fmt.Sprintf("%s: %v", part, err)) }

			if r.onlineErr != nil {
				unread("online status", r.onlineErr)
			} else if r.online != nil {
				o := r.online.Online
				oc.Online = &o
				online[c.ID] = o
			}
			if r.locErr != nil {
				unread("location", r.locErr)
			} else {
				loc := &OverviewLocation{}
				if sys, err := p.system(ctx, r.systemID); err != nil {
					loc.System = fmt.Sprintf("system %d", r.systemID)
					unread("location's system", err)
				} else {
					loc.System = sys.Name
					loc.SecurityDisplay, loc.SecurityColor = nav.SecurityLabel(sys.Security), nav.SecurityColor(sys.Security)
				}
				if id := max(r.stationID, r.structureID); id != 0 {
					loc.DockedAtID, loc.DockedAt = id, p.placeName(ctx, cfg, id)
				}
				oc.Location = loc
			}
			if r.shipErr != nil {
				unread("ship", r.shipErr)
			} else {
				oc.Ship = &OverviewShip{Type: p.typeName(ctx, r.shipType), Name: r.shipName, ItemID: r.shipItem}
			}
			skills := make(map[int64]int, len(r.skills))
			for _, s := range r.skills {
				skills[s.SkillID] = int(s.ActiveSkillLevel)
			}
			if r.skillsErr != nil {
				unread("skills (so abilities and hulls are left out, and the clone cooldown assumes untrained Infomorph Synchronizing)", r.skillsErr)
			} else {
				for _, a := range nav.Abilities(skills, catalog) {
					if a.Can {
						oc.Abilities = append(oc.Abilities, fmt.Sprintf("%s %s: %s", a.Variant, a.Kind, strings.Join(a.Hulls, ", ")))
					}
				}
				byGroup := map[int64][]string{}
				for _, h := range nav.FlyableHulls(skills, classHulls) {
					byGroup[h.GroupID] = append(byGroup[h.GroupID], h.Name)
				}
				for _, g := range hullClassGroups {
					if len(byGroup[g]) > 0 {
						oc.Hulls = append(oc.Hulls, HullClass{Class: groupName[g], Hulls: byGroup[g]})
					}
				}
			}
			if r.clonesErr == nil {
				cl := &OverviewClones{ReadyNow: true, JumpClonesAt: []string{}}
				if r.clones.LastCloneJumpDate != nil {
					next := nav.NextCloneJump(*r.clones.LastCloneJumpDate, skills[nav.SkillInfomorphSynchronizing])
					cl.NextCloneJump, cl.ReadyNow = &next, !now().Before(next)
				}
				for _, jc := range r.clones.JumpClones {
					cl.JumpClonesAt = append(cl.JumpClonesAt, p.placeName(ctx, cfg, jc.LocationID))
				}
				oc.Clones = cl
			}
			if r.clonesErr != nil {
				unread("clones", r.clonesErr)
			}
			if r.fatigueErr != nil {
				unread("jump fatigue", r.fatigueErr)
			} else {
				f := &OverviewFatigue{ExpiresAt: r.fatigue.JumpFatigueExpireDate, LastJump: r.fatigue.LastJumpDate}
				f.Fatigued = f.ExpiresAt != nil && now().Before(*f.ExpiresAt)
				oc.Fatigue = f
			}
			out.Characters = append(out.Characters, oc)
		}

		// Accounts, and labels the game contradicts.
		byAccount := map[string][]config.Character{}
		var labels []string
		for _, c := range chars {
			if c.Account == "" {
				continue
			}
			if _, ok := byAccount[c.Account]; !ok {
				labels = append(labels, c.Account)
			}
			byAccount[c.Account] = append(byAccount[c.Account], c)
		}
		for _, l := range labels {
			g := AccountGroup{Account: l}
			var on []string
			for _, c := range byAccount[l] {
				g.Characters = append(g.Characters, c.Name)
				if online[c.ID] {
					on = append(on, c.Name)
				}
			}
			out.Accounts = append(out.Accounts, g)
			if len(on) > 1 {
				sort.Strings(on)
				out.Conflicts = append(out.Conflicts, fmt.Sprintf("%s are all labelled %q but are online at the same time, so they aren't all on one account. Ask the user which account each is on and fix the labels with update_character_details.", strings.Join(on, " and "), l))
			}
		}
		return nil, out, nil
	}
}
