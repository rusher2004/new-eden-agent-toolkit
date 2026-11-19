package mcp

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// Skill type IDs used by the planning tools (verified against the SDE
// 2026-11-18).
const (
	skillIndustry               int64 = 3380
	skillAdvancedIndustry       int64 = 3388
	skillMassProduction         int64 = 3387
	skillAdvancedMassProduction int64 = 24625
	skillReactions              int64 = 45746
	skillMassReactions          int64 = 45748
	skillAdvancedMassReactions  int64 = 45749
	skillLaboratoryOperation    int64 = 3406
	skillAdvancedLabOperation   int64 = 24624
	skillReprocessing           int64 = 3385
	skillReprocessingEfficiency int64 = 3389
	skillJumpDriveCalibration   int64 = 21611
	skillJumpFuelConservation   int64 = 21610
	skillJumpFreighters         int64 = 29029
	attrImplantRefiningBonus    int64 = 379 // refiningYieldMutator, percent
)

// ESI cost index activity names → industry activity names.
var esiActivity = map[string]string{
	"manufacturing":                   industry.ActManufacturing,
	"reaction":                        industry.ActReaction,
	"copying":                         industry.ActCopying,
	"invention":                       industry.ActInvention,
	"researching_material_efficiency": industry.ActResearchMaterial,
	"researching_time_efficiency":     industry.ActResearchTime,
}

// fetchSellOrders returns every sell order for a type in a region. A package
// variable so tests can fake it.
var fetchSellOrders = func(ctx context.Context, userAgent string, regionID, typeID int64) ([]industry.Order, error) {
	raw, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return listMarketOrders(ctx, userAgent, regionID, "sell", typeID, page)
	})
	if err != nil {
		return nil, fmt.Errorf("sell orders for type %d: %w", typeID, err)
	}
	out := make([]industry.Order, 0, len(raw))
	for _, o := range raw {
		out = append(out, industry.Order{Price: o.Price, Volume: o.VolumeRemain, LocationID: o.LocationId})
	}
	return out, nil
}

// planner gathers SDE, ESI and config inputs for the industry tools and
// caches them for the life of one tool call.
type planner struct {
	version, clientID string
	store             *config.Store
	sde               *sde.Store

	mu       sync.Mutex
	recipes  map[int64]*industry.Recipe // nil value = not buildable
	basics   map[int64]sde.TypeBasic
	filters  map[int64]industry.TargetFilter
	systems  map[int64]sde.RoutingSystem
	graph    *industry.RouteGraph
	adjusted map[int64]float64
	indices  map[int64]map[string]float64
	skillsBy map[int32]map[int64]int
}

func newPlanner(version, clientID string, store *config.Store, sdeStore *sde.Store) *planner {
	return &planner{version: version, clientID: clientID, store: store, sde: sdeStore,
		recipes: map[int64]*industry.Recipe{}, basics: map[int64]sde.TypeBasic{}, skillsBy: map[int32]map[int64]int{}}
}

func (p *planner) ua() string { return sso.UserAgent(p.version) }

// requireCharacter fails unless the character is logged in to neat.
func (p *planner) requireCharacter(id int32) error {
	found, err := characterExists(p.store, id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("character %d: %w", id, ErrCharacterNotFound)
	}
	return nil
}

// recipe returns how a type is made (manufacturing preferred, else
// reaction), cached.
func (p *planner) recipe(ctx context.Context, typeID int64) (industry.Recipe, bool, error) {
	p.mu.Lock()
	if r, ok := p.recipes[typeID]; ok {
		p.mu.Unlock()
		if r == nil {
			return industry.Recipe{}, false, nil
		}
		return *r, true, nil
	}
	p.mu.Unlock()
	srcs, err := p.sde.BlueprintsForProduct(ctx, p.ua(), typeID)
	if err != nil {
		return industry.Recipe{}, false, err
	}
	var rec *industry.Recipe
	for _, s := range srcs {
		bp, ok, err := p.sde.Blueprint(ctx, p.ua(), s.BlueprintTypeID)
		if err != nil {
			return industry.Recipe{}, false, err
		}
		if !ok {
			continue
		}
		act, ok := bp.Activity(s.ActivityID)
		if !ok {
			continue
		}
		r := industry.Recipe{BlueprintTypeID: s.BlueprintTypeID, Activity: industry.ActManufacturing, BaseSeconds: float64(act.TimeSeconds), OutputPerRun: s.Quantity}
		if s.ActivityID == sde.ActivityReactions {
			r.Activity = industry.ActReaction
		}
		for _, m := range act.Materials {
			r.Materials = append(r.Materials, industry.TypeQty{TypeID: m.TypeID, Quantity: m.Quantity})
		}
		for _, sk := range act.Skills {
			r.Skills = append(r.Skills, industry.TypeQty{TypeID: sk.SkillID, Quantity: sk.Level})
		}
		rec = &r
		if r.Activity == industry.ActManufacturing {
			break
		}
	}
	p.mu.Lock()
	p.recipes[typeID] = rec
	p.mu.Unlock()
	if rec == nil {
		return industry.Recipe{}, false, nil
	}
	return *rec, true, nil
}

// typeBasics loads and caches basic type data.
func (p *planner) typeBasics(ctx context.Context, ids []int64) (map[int64]sde.TypeBasic, error) {
	p.mu.Lock()
	var missing []int64
	for _, id := range ids {
		if _, ok := p.basics[id]; !ok && !slices.Contains(missing, id) {
			missing = append(missing, id)
		}
	}
	p.mu.Unlock()
	if len(missing) > 0 {
		got, err := p.sde.TypeBasics(ctx, p.ua(), missing)
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		for k, v := range got {
			p.basics[k] = v
		}
		p.mu.Unlock()
	}
	out := make(map[int64]sde.TypeBasic, len(ids))
	p.mu.Lock()
	for _, id := range ids {
		if b, ok := p.basics[id]; ok {
			out[id] = b
		}
	}
	p.mu.Unlock()
	return out, nil
}

func (p *planner) typeName(ctx context.Context, id int64) string {
	b, err := p.typeBasics(ctx, []int64{id})
	if err != nil || b[id].Name == "" {
		return fmt.Sprintf("type %d", id)
	}
	return b[id].Name
}

func (p *planner) targetFilters(ctx context.Context) (map[int64]industry.TargetFilter, error) {
	if p.filters != nil {
		return p.filters, nil
	}
	raw, err := p.sde.TargetFilters(ctx, p.ua())
	if err != nil {
		return nil, err
	}
	p.filters = map[int64]industry.TargetFilter{}
	for id, f := range raw {
		p.filters[id] = industry.TargetFilter{ID: f.ID, Name: f.Name, Categories: f.Categories, Groups: f.Groups}
	}
	return p.filters, nil
}

func (p *planner) system(ctx context.Context, id int64) (sde.RoutingSystem, error) {
	if p.systems == nil {
		all, err := p.sde.RoutingSystems(ctx, p.ua())
		if err != nil {
			return sde.RoutingSystem{}, err
		}
		p.systems = make(map[int64]sde.RoutingSystem, len(all))
		for _, s := range all {
			p.systems[s.ID] = s
		}
	}
	s, ok := p.systems[id]
	if !ok {
		return sde.RoutingSystem{}, fmt.Errorf("unknown solar system %d", id)
	}
	return s, nil
}

func modifierSources(raw []sde.IndustryModifierSource) []industry.ModifierSource {
	out := make([]industry.ModifierSource, len(raw))
	for i, m := range raw {
		out[i] = industry.ModifierSource{Activity: m.Activity, Kind: m.Kind, AttributeID: m.AttributeID, FilterID: m.FilterID}
	}
	return out
}

// facility resolves a saved profile into an industry.Facility using SDE
// structure and rig data.
func (p *planner) facility(ctx context.Context, prof config.FacilityProfile) (industry.Facility, error) {
	sys, err := p.system(ctx, prof.SystemID)
	if err != nil {
		return industry.Facility{}, fmt.Errorf("facility %q: %w", prof.Name, err)
	}
	filters, err := p.targetFilters(ctx)
	if err != nil {
		return industry.Facility{}, err
	}
	in := industry.ResolveFacilityInput{Name: prof.Name, Kind: prof.Kind, Filters: filters, SystemID: sys.ID, RegionID: sys.RegionID,
		Security: sys.Security, Services: prof.Services, SupercapUpgrade: prof.SupercapUpgrade, FacilityTax: prof.FacilityTax}
	switch prof.Kind {
	case industry.KindUpwell:
		attrs, err := p.sde.TypeAttributes(ctx, p.ua(), prof.StructureType)
		if err != nil {
			return industry.Facility{}, err
		}
		mods, err := p.sde.IndustryModifiers(ctx, p.ua(), prof.StructureType)
		if err != nil {
			return industry.Facility{}, err
		}
		in.Structure = &industry.StructureInput{TypeID: prof.StructureType, Name: p.typeName(ctx, prof.StructureType), Attributes: attrs, Sources: modifierSources(mods)}
		for _, rid := range prof.RigTypeIDs {
			rig, ok, err := p.sde.RigInfo(ctx, p.ua(), rid)
			if err != nil {
				return industry.Facility{}, err
			}
			mods, err := p.sde.IndustryModifiers(ctx, p.ua(), rid)
			if err != nil {
				return industry.Facility{}, err
			}
			if !ok && len(mods) == 0 {
				continue
			}
			t2s := map[int64]int64{}
			for _, m := range rig.Modifiers {
				t2s[m.TargetAttributeID] = m.SourceAttributeID
			}
			in.Rigs = append(in.Rigs, industry.RigInput{TypeID: rid, Name: rig.Name, Attributes: rig.Attributes, Sources: modifierSources(mods), TargetToSource: t2s})
		}
	case industry.KindStarbase:
		if prof.StructureType != 0 {
			attrs, err := p.sde.TypeAttributes(ctx, p.ua(), prof.StructureType)
			if err != nil {
				return industry.Facility{}, err
			}
			in.StarbaseSecurityMax = attrs[sde.AttrAnchoringSecurityMax]
		}
	}
	f := industry.ResolveFacility(in)
	if f.Name == "" {
		f.Name = prof.Name
	}
	return f, nil
}

// multipliersFor returns a function giving facility multipliers per activity
// and product type, choosing the reaction facility for reactions.
func (p *planner) multipliersFor(ctx context.Context, mfg, rx *industry.Facility) func(activity string, product int64) industry.Multipliers {
	return func(activity string, product int64) industry.Multipliers {
		f := mfg
		if activity == industry.ActReaction && rx != nil {
			f = rx
		}
		if f == nil {
			return industry.Multipliers{Material: 1, Time: 1, Cost: 1}
		}
		b, _ := p.typeBasics(ctx, []int64{product})
		return f.Multipliers(activity, b[product].CategoryID, b[product].GroupID)
	}
}

// sellBooks fetches sell order books for many types with bounded concurrency.
func (p *planner) sellBooks(ctx context.Context, regionID int64, typeIDs []int64) (map[int64][]industry.Order, error) {
	ids := slices.Clone(typeIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	out := make(map[int64][]industry.Order, len(ids))
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(id int64) {
			defer wg.Done()
			defer func() { <-sem }()
			orders, err := fetchSellOrders(ctx, p.ua(), regionID, id)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			out[id] = orders
		}(id)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func (p *planner) adjustedPrices(ctx context.Context) (map[int64]float64, error) {
	if p.adjusted != nil {
		return p.adjusted, nil
	}
	raw, err := listMarketPrices(ctx, p.ua())
	if err != nil {
		return nil, fmt.Errorf("adjusted prices: %w", err)
	}
	p.adjusted = make(map[int64]float64, len(raw))
	for _, r := range raw {
		if r.AdjustedPrice != nil {
			p.adjusted[r.TypeId] = *r.AdjustedPrice
		}
	}
	return p.adjusted, nil
}

// costIndex returns a system's cost index for an activity.
func (p *planner) costIndex(ctx context.Context, systemID int64, activity string) (float64, error) {
	if p.indices == nil {
		raw, err := listIndustrySystems(ctx, p.ua())
		if err != nil {
			return 0, fmt.Errorf("cost indices: %w", err)
		}
		p.indices = make(map[int64]map[string]float64, len(raw))
		for _, s := range raw {
			m := map[string]float64{}
			for _, ci := range s.CostIndices {
				if a, ok := esiActivity[ci.Activity]; ok {
					m[a] = ci.CostIndex
				}
			}
			p.indices[s.SolarSystemId] = m
		}
	}
	return p.indices[systemID][activity], nil
}

// skills returns a character's active skill levels.
func (p *planner) skills(ctx context.Context, characterID int32) (map[int64]int, error) {
	if s, ok := p.skillsBy[characterID]; ok {
		return s, nil
	}
	if err := p.requireCharacter(characterID); err != nil {
		return nil, err
	}
	raw, _, _, err := listCharacterSkills(ctx, p.clientID, characterID, p.ua())
	if err != nil {
		return nil, fmt.Errorf("skills of %d: %w", characterID, err)
	}
	m := make(map[int64]int, len(raw))
	for _, s := range raw {
		m[s.SkillID] = int(s.ActiveSkillLevel)
	}
	p.skillsBy[characterID] = m
	return m, nil
}

// implantRefiningBonus is the character's reprocessing implant bonus (0–1).
func (p *planner) implantRefiningBonus(ctx context.Context, characterID int32) (float64, string, error) {
	ids, err := listCharacterImplants(ctx, p.clientID, characterID, p.ua())
	if err != nil {
		return 0, "", fmt.Errorf("implants of %d: %w", characterID, err)
	}
	best, name := 0.0, ""
	for _, id := range ids {
		attrs, err := p.sde.TypeAttributes(ctx, p.ua(), id)
		if err != nil {
			return 0, "", err
		}
		if v := attrs[attrImplantRefiningBonus] / 100; v > best {
			best, name = v, p.typeName(ctx, id)
		}
	}
	return best, name, nil
}

// builder turns a character into an industry.Builder.
func (p *planner) builder(ctx context.Context, characterID int32, name string) (industry.Builder, error) {
	s, err := p.skills(ctx, characterID)
	if err != nil {
		return industry.Builder{}, err
	}
	return industry.Builder{
		CharacterID: int64(characterID), Name: name,
		Slots: map[string]int{
			industry.ActManufacturing: 1 + s[skillMassProduction] + s[skillAdvancedMassProduction],
			industry.ActReaction:      boolInt(s[skillReactions] > 0) * (1 + s[skillMassReactions] + s[skillAdvancedMassReactions]),
			"science":                 1 + s[skillLaboratoryOperation] + s[skillAdvancedLabOperation],
		},
		TimeMult: map[string]float64{
			industry.ActManufacturing: industry.ManufacturingSkillMultiplier(s[skillIndustry], s[skillAdvancedIndustry]),
			industry.ActReaction:      industry.ReactionSkillMultiplier(s[skillReactions]),
		},
		Skills: s,
	}, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (p *planner) characterName(id int32) string {
	cfg, err := p.store.Load()
	if err == nil {
		for _, c := range cfg.Characters {
			if c.ID == id {
				return c.Name
			}
		}
	}
	return fmt.Sprintf("character %d", id)
}

// jumpDrive reads a hull's jump drive from SDE dogma.
func (p *planner) jumpDrive(ctx context.Context, shipTypeID int64) (industry.JumpDrive, error) {
	attrs, err := p.sde.TypeAttributes(ctx, p.ua(), shipTypeID)
	if err != nil {
		return industry.JumpDrive{}, err
	}
	return industry.JumpDrive{
		ShipTypeID: shipTypeID, ShipName: p.typeName(ctx, shipTypeID),
		BaseRangeLY: attrs[sde.AttrJumpDriveRange], FuelPerLY: attrs[sde.AttrJumpDriveConsumption],
		FuelTypeID: int64(attrs[sde.AttrJumpDriveConsumptionType]), SkillBonusPerLevel: attrs[sde.AttrJumpFreighterFuelBonus],
		FatigueMultiplier: attrs[sde.AttrJumpFatigueMultiplier],
	}, nil
}

// canUseHighsec reports whether a hull may enter highsec: everything except
// the capital groups that are barred from it.
func canUseHighsec(groupID int64) bool {
	return !slices.Contains(industry.CapitalShipyardGroups, groupID) && !slices.Contains(industry.SupercapitalGroups, groupID)
}

func (p *planner) routeGraph(ctx context.Context) (*industry.RouteGraph, error) {
	if p.graph != nil {
		return p.graph, nil
	}
	sys, err := p.sde.RoutingSystems(ctx, p.ua())
	if err != nil {
		return nil, err
	}
	gates, err := p.sde.StargateConnections(ctx, p.ua())
	if err != nil {
		return nil, err
	}
	rs := make([]industry.RouteSystem, len(sys))
	for i, s := range sys {
		rs[i] = industry.RouteSystem{ID: s.ID, Name: s.Name, ConstellationID: s.ConstellationID, RegionID: s.RegionID, RegionName: s.RegionName,
			Security: s.Security, Pos: industry.Position{X: s.X, Y: s.Y, Z: s.Z}, HasNPCStation: s.HasNPCStation}
	}
	p.graph = industry.NewRouteGraph(rs, gates)
	return p.graph, nil
}

// routePrefs resolves saved preferences into system sets, fetching
// sovereignty, contacts, incursions and recent kills only when needed.
func (p *planner) routePrefs(ctx context.Context, pilotID int32, prefs config.RoutePrefs) (industry.RoutePrefs, []string, error) {
	set := func(ids []int64) map[int64]bool {
		m := map[int64]bool{}
		for _, id := range ids {
			m[id] = true
		}
		return m
	}
	out := industry.RoutePrefs{
		AvoidSystems: set(prefs.AvoidSystems), AvoidConstellations: set(prefs.AvoidConstellations), AvoidRegions: set(prefs.AvoidRegions),
		Midpoints: set(prefs.Midpoints), MidpointsRequired: prefs.MidpointsRequired, PreferStationSystems: prefs.PreferStationSystems,
		AvoidIncursions: prefs.AvoidIncursions, Optimize: prefs.Optimize, MaxGates: prefs.MaxGates, MaxRecentKills: prefs.MaxRecentKills,
		AvoidSovSystems: map[int64]bool{}, PreferSovSystems: map[int64]bool{},
	}
	var notes []string
	avoidAlliances := set(prefs.AvoidAlliances)
	preferAlliances := set(prefs.PreferAlliances)
	if prefs.AvoidBadStanding {
		hostile, err := p.badStandingAlliances(ctx, pilotID)
		if err != nil {
			notes = append(notes, "couldn't read contacts, so bad-standing sovereignty isn't avoided: "+err.Error())
		}
		for id := range hostile {
			avoidAlliances[id] = true
		}
	}
	if len(avoidAlliances) > 0 || len(preferAlliances) > 0 {
		sov, err := listSovereigntySystems(ctx, p.ua())
		if err != nil {
			return out, notes, fmt.Errorf("sovereignty map: %w", err)
		}
		for _, s := range sov {
			if s.Claim.Alliance == nil || s.Claim.Alliance.Alliance == nil {
				continue
			}
			a := s.Claim.Alliance.Alliance.AllianceId
			if avoidAlliances[a] {
				out.AvoidSovSystems[s.SolarSystemId] = true
			}
			if preferAlliances[a] {
				out.PreferSovSystems[s.SolarSystemId] = true
			}
		}
	}
	if prefs.AvoidIncursions {
		inc, err := listIncursions(ctx, p.ua())
		if err != nil {
			return out, notes, fmt.Errorf("incursions: %w", err)
		}
		out.IncursionSystems = map[int64]bool{}
		for _, i := range inc {
			for _, s := range i.InfestedSolarSystemIDs {
				out.IncursionSystems[s] = true
			}
		}
	}
	kills, err := listUniverseSystemKills(ctx, p.ua())
	if err != nil {
		notes = append(notes, "recent kills unavailable: "+err.Error())
	} else {
		out.RecentKills = map[int64]int{}
		for _, k := range kills {
			out.RecentKills[k.SystemID] = int(k.ShipKills + k.PodKills)
		}
	}
	return out, notes, nil
}

// badStandingAlliances returns alliances the pilot, their corporation or
// their alliance have set below zero standing.
func (p *planner) badStandingAlliances(ctx context.Context, pilotID int32) (map[int64]bool, error) {
	out := map[int64]bool{}
	add := func(cs []Contact) {
		for _, c := range cs {
			if c.ContactType == "alliance" && c.Standing < 0 {
				out[c.ContactID] = true
			}
		}
	}
	mine, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Contact, int32, error) {
		return listCharacterContacts(ctx, p.clientID, pilotID, p.ua(), page)
	})
	if err != nil {
		return out, fmt.Errorf("character contacts: %w", err)
	}
	add(mine)
	aff, err := getCharacterAffiliations(ctx, p.ua(), []int64{int64(pilotID)})
	if err != nil || len(aff) == 0 {
		return out, nil
	}
	if corp, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Contact, int32, error) {
		return listCorporationContacts(ctx, p.clientID, pilotID, p.ua(), aff[0].CorporationID, page)
	}); err == nil {
		add(corp)
	}
	if aff[0].AllianceID != nil {
		if al, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Contact, int32, error) {
			return listAllianceContacts(ctx, p.clientID, pilotID, p.ua(), *aff[0].AllianceID, page)
		}); err == nil {
			add(al)
		}
	}
	return out, nil
}

// systemIDs resolves system names to IDs via the routing system list.
func (p *planner) systemIDs(ctx context.Context, names []string) ([]int64, error) {
	if _, err := p.system(ctx, 30000142); err != nil && p.systems == nil {
		return nil, err
	}
	byName := map[string]int64{}
	for id, s := range p.systems {
		byName[strings.ToLower(s.Name)] = id
	}
	out := make([]int64, 0, len(names))
	for _, n := range names {
		id, ok := byName[strings.ToLower(strings.TrimSpace(n))]
		if !ok {
			return nil, fmt.Errorf("unknown system %q", n)
		}
		out = append(out, id)
	}
	return out, nil
}

func sortedKeys[V any](m map[int64]V) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
