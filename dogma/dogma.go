package dogma

import (
	"math"
	"slices"
	"sort"
)

// Effect categories that apply to a fit: passive effects (hulls, skills,
// rigs) and online effects (fitted modules). Active effects apply only
// while a module is cycled, which a fit doesn't model.
const (
	CategoryPassive = 0
	CategoryActive  = 1
	CategoryOnline  = 4
)

// Dogma operations, in the order the game applies them.
const (
	OpPreAssign   = -1
	OpPreMul      = 0
	OpPreDiv      = 1
	OpModAdd      = 2
	OpModSub      = 3
	OpPostMul     = 4
	OpPostDiv     = 5
	OpPostPercent = 6
	OpPostAssign  = 7
)

var opOrder = []int{OpPreAssign, OpPreMul, OpPreDiv, OpModAdd, OpModSub, OpPostMul, OpPostDiv, OpPostPercent, OpPostAssign}

// AttrSkillLevel is a skill's level attribute (skillLevel).
const AttrSkillLevel int64 = 280

// stackingFactor is the stacking penalty constant: the nth strongest
// penalized modifier applies at exp(-(n/2.67)^2) of its strength. Source:
// EVE University, Stacking penalties.
const stackingFactor = 2.67

// Modifier is one entry of an effect's modifierInfo.
type Modifier struct {
	Domain        string // itemID, shipID, charID, ...
	Func          string // ItemModifier, LocationModifier, LocationGroupModifier, LocationRequiredSkillModifier, OwnerRequiredSkillModifier
	ModifiedAttr  int64
	ModifyingAttr int64
	Operation     int
	SkillTypeID   int64 // LocationRequiredSkillModifier, OwnerRequiredSkillModifier
	GroupID       int64 // LocationGroupModifier
}

// Effect is a dogma effect with its modifiers.
type Effect struct {
	Name      string
	Category  int
	Modifiers []Modifier
}

// Kind is what part of a fit an item is.
type Kind int

// Item kinds.
const (
	Ship Kind = iota
	Module
	Skill
)

// Item is a ship, a fitted module or a trained skill.
type Item struct {
	Kind    Kind
	TypeID  int64
	GroupID int64
	Attrs   map[int64]float64
	Effects []Effect
	// RequiredSkills are the skills the type requires (from its
	// requiredSkillN attributes), for LocationRequiredSkillModifier.
	RequiredSkills []int64
}

// AttrInfo is what the SDE says about an attribute.
type AttrInfo struct {
	Default   float64
	Stackable bool
}

// Fit is a ship with its fitted modules and its pilot's trained skills.
type Fit struct {
	Ship    Item
	Modules []Item
	Skills  []Item
	Attrs   map[int64]AttrInfo

	memo     map[ref]float64
	visiting map[ref]bool
}

// Ref names an item in a fit: the ship, the nth module or the nth skill.
type ref struct {
	kind  Kind
	index int
	attr  int64
}

// skillEffect is the generic effect every skill carries to set its level
// from its skill points; a fit sets levels directly, so it's skipped.
const skillEffect = "skillEffect"

// NewSkill is a trained skill item at a level.
func NewSkill(typeID int64, level int, attrs map[int64]float64, effects []Effect) Item {
	a := make(map[int64]float64, len(attrs)+1)
	for k, v := range attrs {
		a[k] = v
	}
	a[AttrSkillLevel] = float64(level)
	return Item{Kind: Skill, TypeID: typeID, Attrs: a, Effects: effects}
}

func (f *Fit) item(kind Kind, i int) *Item {
	switch kind {
	case Module:
		return &f.Modules[i]
	case Skill:
		return &f.Skills[i]
	}
	return &f.Ship
}

// ShipAttr is the ship's attribute after every modifier.
func (f *Fit) ShipAttr(attr int64) float64 { return f.value(ref{Ship, 0, attr}) }

// ModuleAttr is the ith module's attribute after every modifier.
func (f *Fit) ModuleAttr(i int, attr int64) float64 { return f.value(ref{Module, i, attr}) }

type applied struct {
	op        int
	value     float64
	penalized bool
}

func (f *Fit) value(r ref) float64 {
	if f.memo == nil {
		f.memo, f.visiting = map[ref]float64{}, map[ref]bool{}
	}
	if v, ok := f.memo[r]; ok {
		return v
	}
	it := f.item(r.kind, r.index)
	base, ok := it.Attrs[r.attr]
	if !ok {
		base = f.Attrs[r.attr].Default
	}
	if f.visiting[r] {
		return base // a cycle: use the unmodified value
	}
	f.visiting[r] = true
	defer delete(f.visiting, r)

	var mods []applied
	f.each(func(src ref, m Modifier) {
		if m.ModifiedAttr != r.attr || !f.targets(src, m, r) {
			return
		}
		v := f.value(ref{src.kind, src.index, m.ModifyingAttr})
		// An attribute missing from Attrs counts as stackable: stacking
		// penalties are the exception, marked in the SDE.
		info, known := f.Attrs[r.attr]
		penalized := src.kind == Module && known && !info.Stackable && r.kind == Ship
		mods = append(mods, applied{op: m.Operation, value: v, penalized: penalized})
	})
	v := apply(base, mods)
	f.memo[r] = v
	return v
}

// each calls fn for every modifier of every effect that applies to the fit.
func (f *Fit) each(fn func(src ref, m Modifier)) {
	visit := func(kind Kind, i int, it Item) {
		for _, e := range it.Effects {
			if e.Name == skillEffect {
				continue
			}
			if e.Category != CategoryPassive && !(kind == Module && e.Category == CategoryOnline) {
				continue
			}
			for _, m := range e.Modifiers {
				fn(ref{kind, i, 0}, m)
			}
		}
	}
	visit(Ship, 0, f.Ship)
	for i, m := range f.Modules {
		visit(Module, i, m)
	}
	for i, s := range f.Skills {
		visit(Skill, i, s)
	}
}

// targets reports whether a modifier from src applies to the target item.
func (f *Fit) targets(src ref, m Modifier, t ref) bool {
	switch m.Domain {
	case "itemID":
		return src.kind == t.kind && src.index == t.index
	case "shipID":
		switch m.Func {
		case "ItemModifier":
			return t.kind == Ship
		case "LocationModifier":
			return t.kind == Module
		case "LocationGroupModifier":
			return t.kind == Module && f.Modules[t.index].GroupID == m.GroupID
		case "LocationRequiredSkillModifier":
			return t.kind == Module && slices.Contains(f.Modules[t.index].RequiredSkills, m.SkillTypeID)
		}
	case "charID":
		return m.Func == "OwnerRequiredSkillModifier" && t.kind == Module && slices.Contains(f.Modules[t.index].RequiredSkills, m.SkillTypeID)
	}
	return false
}

// apply runs modifiers over a base value in operation order, with stacking
// penalties on the penalized multiplicative ones.
func apply(base float64, mods []applied) float64 {
	v := base
	for _, op := range opOrder {
		var free, penalized []float64
		for _, m := range mods {
			if m.op != op {
				continue
			}
			if m.penalized && isMultiplicative(op) {
				penalized = append(penalized, multiplier(op, m.value))
			} else {
				free = append(free, m.value)
			}
		}
		for _, x := range free {
			v = step(op, v, x)
		}
		v *= penalizedProduct(penalized)
	}
	return v
}

func isMultiplicative(op int) bool {
	return op == OpPreMul || op == OpPreDiv || op == OpPostMul || op == OpPostDiv || op == OpPostPercent
}

// multiplier turns a multiplicative modifier into the factor it applies.
func multiplier(op int, x float64) float64 {
	switch op {
	case OpPreDiv, OpPostDiv:
		if x == 0 {
			return 1
		}
		return 1 / x
	case OpPostPercent:
		return 1 + x/100
	}
	return x
}

func step(op int, v, x float64) float64 {
	switch op {
	case OpPreAssign, OpPostAssign:
		return x
	case OpModAdd:
		return v + x
	case OpModSub:
		return v - x
	}
	return v * multiplier(op, x)
}

// penalizedProduct applies stacking penalties: bonuses and penalties are
// each sorted strongest first, and the nth applies at exp(-(n/2.67)^2).
func penalizedProduct(factors []float64) float64 {
	var up, down []float64
	for _, f := range factors {
		if f >= 1 {
			up = append(up, f)
		} else {
			down = append(down, f)
		}
	}
	sort.Slice(up, func(i, j int) bool { return up[i] > up[j] })
	sort.Slice(down, func(i, j int) bool { return down[i] < down[j] })
	out := 1.0
	for _, list := range [][]float64{up, down} {
		for n, f := range list {
			out *= 1 + (f-1)*math.Exp(-math.Pow(float64(n)/stackingFactor, 2))
		}
	}
	return out
}
