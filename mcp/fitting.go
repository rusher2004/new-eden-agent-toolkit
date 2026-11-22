package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/rusher2004/new-eden-agent-toolkit/dogma"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// isFittedFlag reports whether an asset's location flag puts it in one of
// a ship's fitting slots rather than a hold.
func isFittedFlag(flag string) bool {
	for _, p := range []string{"HiSlot", "MedSlot", "LoSlot", "RigSlot", "SubSystemSlot"} {
		if strings.HasPrefix(flag, p) {
			return true
		}
	}
	return false
}

// Inventory categories a fitted slot can hold besides modules and rigs.
const (
	chargeCategoryID    = 8
	subsystemCategoryID = 32
)

func dogmaItem(kind dogma.Kind, t sde.DogmaType) dogma.Item {
	it := dogma.Item{Kind: kind, TypeID: t.TypeID, GroupID: t.GroupID, Attrs: t.Attrs}
	for _, r := range requiredSkills(t.Attrs) {
		it.RequiredSkills = append(it.RequiredSkills, r.SkillID)
	}
	for _, e := range t.Effects {
		de := dogma.Effect{Name: e.Name, Category: e.Category}
		for _, m := range e.Modifiers {
			de.Modifiers = append(de.Modifiers, dogma.Modifier{Domain: m.Domain, Func: m.Func, ModifiedAttr: m.ModifiedAttr, ModifyingAttr: m.ModifyingAttr,
				Operation: m.Operation, SkillTypeID: m.SkillTypeID, GroupID: m.GroupID})
		}
		it.Effects = append(it.Effects, de)
	}
	return it
}

// fit builds a dogma fit for a hull, its modules and a pilot's skills
// (skill ID to level; nil for an untrained pilot), from the SDE.
func (p *planner) fit(ctx context.Context, hull int64, modules []int64, skills map[int64]int) (*dogma.Fit, error) {
	ids := append([]int64{hull}, modules...)
	for id, lvl := range skills {
		if lvl > 0 {
			ids = append(ids, id)
		}
	}
	types, err := p.sde.DogmaTypes(ctx, p.ua(), ids)
	if err != nil {
		return nil, err
	}
	h, ok := types[hull]
	if !ok {
		return nil, fmt.Errorf("unknown hull type %d", hull)
	}
	info, err := p.sde.AttributeInfos(ctx, p.ua())
	if err != nil {
		return nil, err
	}
	f := &dogma.Fit{Ship: dogmaItem(dogma.Ship, h), Attrs: map[int64]dogma.AttrInfo{}}
	for id, a := range info {
		f.Attrs[id] = dogma.AttrInfo{Default: a.Default, Stackable: a.Stackable}
	}
	basics, err := p.typeBasics(ctx, modules)
	if err != nil {
		return nil, err
	}
	for _, m := range modules {
		t, ok := types[m]
		if !ok {
			return nil, fmt.Errorf("unknown module type %d", m)
		}
		it := dogmaItem(dogma.Module, t)
		// The game exempts subsystems from stacking penalties.
		it.Unpenalized = basics[m].CategoryID == subsystemCategoryID
		f.Modules = append(f.Modules, it)
	}
	for id, lvl := range skills {
		if t, ok := types[id]; ok && lvl > 0 {
			s := dogmaItem(dogma.Skill, t)
			f.Skills = append(f.Skills, dogma.NewSkill(s.TypeID, lvl, s.Attrs, s.Effects))
		}
	}
	return f, nil
}
