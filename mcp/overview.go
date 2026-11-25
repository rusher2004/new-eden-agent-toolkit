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
	CooldownReady bool          `json:"cooldownReady" jsonschema:"their clone jump cooldown has passed"`
	ReadyNow      bool          `json:"readyNow" jsonschema:"true only when the cooldown has passed and they're docked where a clone bay is known to be (jumpFromHere yes)"`
	JumpFromHere  *JumpFromHere `json:"jumpFromHere,omitempty" jsonschema:"whether they can start a clone jump where they are now; absent when their location couldn't be read"`
	NextCloneJump *time.Time    `json:"nextCloneJump,omitempty"`
	JumpClonesAt  []string      `json:"jumpClonesAt" jsonschema:"where their jump clones are; list_jump_clones has the implants"`
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
	Conflicts  []string            `json:"conflicts,omitempty" jsonschema:"account labels the game contradicts: two characters with the same label are online at once, so they can't share an account; a recent login is checked again after ESI's cache expires before it's flagged"`
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

// overviewClient is the authenticated ESI calls get_logistics_overview makes
// for each character: the planner's (skills) and its own location, ship,
// online status, clones and fatigue.
type overviewClient interface {
	plannerClient
	characterLocationESI
	characterShipESI
	characterOnlineESI
	characterClonesESI
	characterFatigueESI
}

// readOverview fetches one character's location, ship, online status,
// clones, fatigue and skills in parallel. These calls share no planner
// state, so they're safe to run together. A client that can't be opened
// fails every read with the same error.
func readOverview[C overviewClient](ctx context.Context, open func(ctx context.Context, characterID int32) (C, error), id int32) overviewReads {
	var r overviewReads
	client, err := open(ctx, id)
	if err != nil {
		r.locErr, r.shipErr, r.onlineErr, r.clonesErr, r.fatigueErr, r.skillsErr = err, err, err, err, err, err
		return r
	}
	var wg sync.WaitGroup
	run := func(f func()) {
		wg.Go(func() { ; f() })
	}
	run(func() {
		var st, sr *int64
		r.systemID, st, sr, r.locErr = getCharacterLocation(ctx, client)
		if st != nil {
			r.stationID = *st
		}
		if sr != nil {
			r.structureID = *sr
		}
	})
	run(func() { r.shipItem, r.shipName, r.shipType, r.shipErr = getCharacterShip(ctx, client) })
	run(func() { r.online, r.onlineErr = getCharacterOnline(ctx, client) })
	run(func() { r.clones, r.clonesErr = getCharacterClones(ctx, client, id) })
	run(func() { r.fatigue, r.fatigueErr = getCharacterFatigue(ctx, client, id) })
	run(func() { r.skills, _, _, r.skillsErr = listCharacterSkills(ctx, client) })
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

// AddGetLogisticsOverview registers get_logistics_overview. openESI and openPublicESI build the ESI clients;
// any client with the methods the planner and this tool call will do.
func AddGetLogisticsOverview[C overviewClient, P plannerPublicClient](s *mcpsdk.Server, userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_logistics_overview",
		Description: "Start here for any move of ships, cargo or characters. One call gives, for each logged-in character: their account label, whether they're online, where they are (system with its displayed security, and the station or structure they're docked in), their ship, the cynos, bridges and conduits they can use with the hulls for each, the hauling and capital hulls they can fly, when they can next clone jump, whether they can clone jump from where they're docked, and where their jump clones are, and their jump fatigue. It also groups characters by account, flags labels the game contradicts (two characters labelled alike online at once; when one of them logged in within the last minute, it waits about a minute and checks again, since ESI can still show a character who just switched off as online), and warns about unknown accounts. Parts ESI won't give are named per character, not dropped silently.",
	}, getLogisticsOverviewHandler(userAgent, openESI, openPublicESI, store, sdeStore))
}

func getLogisticsOverviewHandler[C overviewClient, P plannerPublicClient](userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) mcpsdk.ToolHandlerFor[GetLogisticsOverviewInput, GetLogisticsOverviewOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetLogisticsOverviewInput) (*mcpsdk.CallToolResult, GetLogisticsOverviewOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, fmt.Errorf("load config: %w", err)
		}
		chars, err := targetCharacters(cfg, in.CharacterIDs)
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, err
		}
		p := newPlanner(userAgent, plannerOpener(openESI), plannerPublicOpener(openPublicESI), store, sdeStore)
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
		online := map[int32]*esi.CharacterOnlineStatus{} // characters reported online
		var allClones []GetCharacterClonesOutput
		for _, c := range chars {
			r := readOverview(ctx, openESI, c.ID)
			oc := OverviewCharacter{CharacterID: c.ID, CharacterName: c.Name, Account: c.Account, Abilities: []string{}, Hulls: []HullClass{}}
			unread := func(part string, err error) { oc.Unread = append(oc.Unread, fmt.Sprintf("%s: %v", part, err)) }

			if r.onlineErr != nil {
				unread("online status", r.onlineErr)
			} else if r.online != nil {
				o := r.online.Online
				oc.Online = &o
				if o {
					online[c.ID] = r.online
				}
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
				cl := &OverviewClones{CooldownReady: true, JumpClonesAt: []string{}}
				if r.clones.LastCloneJumpDate != nil {
					next := nav.NextCloneJump(*r.clones.LastCloneJumpDate, skills[nav.SkillInfomorphSynchronizing])
					cl.NextCloneJump, cl.CooldownReady = &next, !now().Before(next)
				}
				allClones = append(allClones, r.clones)
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

		// Clone bays are judged once every character's jump clones are read,
		// since one character's clone proves a bay for all, including
		// characters that weren't asked about.
		others, warnings := otherCharactersClones(ctx, openESI, cfg, chars)
		out.Warnings = append(out.Warnings, warnings...)
		proven := provenCloneBays(append(allClones, others...)...)
		for i := range out.Characters {
			oc := &out.Characters[i]
			if oc.Clones == nil || oc.Location == nil {
				continue
			}
			j := p.jumpFromHere(ctx, cfg, oc.Location.DockedAtID, proven)
			oc.Clones.JumpFromHere = &j
			oc.Clones.ReadyNow = oc.Clones.CooldownReady && j.Status == jumpFromYes
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
		suspect := map[string][]config.Character{}
		for _, l := range labels {
			g := AccountGroup{Account: l}
			var on []config.Character
			for _, c := range byAccount[l] {
				g.Characters = append(g.Characters, c.Name)
				if online[c.ID] != nil {
					on = append(on, c)
				}
			}
			out.Accounts = append(out.Accounts, g)
			if len(on) > 1 {
				suspect[l] = on
			}
		}
		confirmed, recheckWarnings, err := confirmOnlineConflicts(ctx, openESI, suspect, online)
		if err != nil {
			return nil, GetLogisticsOverviewOutput{}, err
		}
		out.Warnings = append(out.Warnings, recheckWarnings...)
		for _, l := range labels {
			if on := confirmed[l]; len(on) > 1 {
				sort.Strings(on)
				out.Conflicts = append(out.Conflicts, fmt.Sprintf("%s are all labelled %q but are online at the same time, so they aren't all on one account. Ask the user which account each is on and fix the labels with update_character_details.", strings.Join(on, " and "), l))
			}
		}
		return nil, out, nil
	}
}

// onlineCacheWindow is how long ESI caches a character's online status
// (the Expires header on GET /characters/{id}/online is 60 seconds), plus
// a few seconds' margin. A character who logged off within it can still
// read as online, so two characters on one account can both look online
// right after a switch.
const onlineCacheWindow = 65 * time.Second

// sleepCtx waits for d or until ctx is done. Tests replace it.
var sleepCtx = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// confirmOnlineConflicts returns, for each account label, the names of its
// characters that really are online together, leaving out labels with
// fewer than two. A suspected conflict whose
// newest login is older than onlineCacheWindow stands: every cached status
// was read after that login, so the characters were online at once. A
// newer one may be a character switch the cache hasn't caught up with, so
// it waits until the cache has expired and checks those characters again,
// keeping only the ones still online. A character whose re-check fails is
// left out, since a conflict it can't confirm isn't reported.
func confirmOnlineConflicts[C characterOnlineESI](ctx context.Context, open func(ctx context.Context, characterID int32) (C, error), suspect map[string][]config.Character, online map[int32]*esi.CharacterOnlineStatus) (map[string][]string, []string, error) {
	confirmed := map[string][]string{}
	var warnings []string
	recheck := map[string][]config.Character{}
	var wait time.Duration
	for l, chars := range suspect {
		var newest *time.Time
		for _, c := range chars {
			if ll := online[c.ID].LastLogin; ll == nil {
				newest = nil
				break
			} else if newest == nil || ll.After(*newest) {
				newest = ll
			}
		}
		if newest != nil && now().Sub(*newest) >= onlineCacheWindow {
			for _, c := range chars {
				confirmed[l] = append(confirmed[l], c.Name)
			}
			continue
		}
		recheck[l] = chars
		left := onlineCacheWindow // an unknown login time waits the full window
		if newest != nil {
			left -= max(now().Sub(*newest), 0)
		}
		wait = max(wait, left)
	}
	if len(recheck) == 0 {
		return confirmed, nil, nil
	}
	if err := sleepCtx(ctx, wait); err != nil {
		return nil, nil, fmt.Errorf("wait to re-check online status: %w", err)
	}
	for l, chars := range recheck {
		for _, c := range chars {
			client, err := open(ctx, c.ID)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("couldn't re-check %s's online status, so an account conflict involving them may be missed: open ESI client: %v", c.Name, err))
				continue
			}
			st, err := getCharacterOnline(ctx, client)
			if err != nil || st == nil {
				warnings = append(warnings, fmt.Sprintf("couldn't re-check %s's online status, so an account conflict involving them may be missed: %v", c.Name, err))
				continue
			}
			if st.Online {
				confirmed[l] = append(confirmed[l], c.Name)
			}
		}
		if len(confirmed[l]) < 2 {
			delete(confirmed, l)
		}
	}
	return confirmed, warnings, nil
}
