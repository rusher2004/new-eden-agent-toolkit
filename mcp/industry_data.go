package mcp

import (
	"context"
	"fmt"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
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

func (p *planner) adjustedPrices(ctx context.Context) (map[int64]float64, error) {
	if p.adjusted != nil {
		return p.adjusted, nil
	}
	raw, err := listMarketPrices(ctx, p.openPublicESI())
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
		raw, err := listIndustrySystems(ctx, p.openPublicESI())
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

// implantRefiningBonus is the character's reprocessing implant bonus (0–1).
func (p *planner) implantRefiningBonus(ctx context.Context, characterID int32) (float64, string, error) {
	client, err := p.esi(ctx, characterID)
	if err != nil {
		return 0, "", fmt.Errorf("implants: %w", err)
	}
	ids, err := listCharacterImplants(ctx, client)
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
