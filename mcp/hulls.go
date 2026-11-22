package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// Modules whose fitting restrictions say which hulls can cloak and warp, and
// which can fit an Interdiction Nullifier. Type IDs checked against the SDE
// 2026-11-20.
const (
	covertOpsCloakTypeID        int64 = 11578 // Covert Ops Cloaking Device II
	interdictionNullifierTypeID int64 = 58919 // Interdiction Nullifier I
)

// fitList reads a module's fitting restrictions: the ship groups and types
// it can be fitted to.
func fitList(attrs map[int64]float64) (groups, types []int64) {
	for _, a := range sde.AttrCanFitShipGroups {
		if g := int64(attrs[a]); g != 0 {
			groups = append(groups, g)
		}
	}
	for _, a := range sde.AttrCanFitShipTypes {
		if t := int64(attrs[a]); t != 0 {
			types = append(types, t)
		}
	}
	return groups, types
}

// fits reports whether a module's fitting restrictions allow a hull.
type fitRule struct{ groups, types []int64 }

func (f fitRule) allows(typeID, groupID int64) bool {
	return slices.Contains(f.groups, groupID) || slices.Contains(f.types, typeID)
}

func (p *planner) fitRule(ctx context.Context, moduleID int64) (fitRule, error) {
	attrs, err := p.sde.TypeAttributes(ctx, p.ua(), moduleID)
	if err != nil {
		return fitRule{}, fmt.Errorf("attributes of module %d: %w", moduleID, err)
	}
	g, t := fitList(attrs)
	return fitRule{g, t}, nil
}

// --- compare_haul_hulls ---

// CompareHaulHullsInput is compare_haul_hulls' input.
type CompareHaulHullsInput struct {
	Classes []string `json:"classes,omitempty" jsonschema:"ship classes by name, such as Jump Freighter, Deep Space Transport, Blockade Runner or Carrier; omit classes and hulls for every hauling, jump-capable and capital class"`
	Hulls   []string `json:"hulls,omitempty" jsonschema:"particular hulls by name, such as Nomad or Occator"`
	PilotID int32    `json:"pilotId,omitempty" jsonschema:"a logged-in character: says which hulls they can fly, and their jump range and fuel per light year with their skills"`
}

// HaulHull is one hull's facts for choosing what to move cargo or ships in.
type HaulHull struct {
	nav.HullFacts
	HoldsM3        map[string]float64 `json:"holdsM3" jsonschema:"every hold by base capacity, before skills, modules and rigs; special holds take only their own kind of item, and assembled ships go only in a ship maintenance bay"`
	FuelName       string             `json:"fuelName,omitempty"`
	PilotCanFly    *bool              `json:"pilotCanFly,omitempty"`
	PilotRangeLY   float64            `json:"pilotRangeLy,omitempty" jsonschema:"jump range with the pilot's Jump Drive Calibration"`
	PilotFuelPerLY float64            `json:"pilotFuelPerLy,omitempty" jsonschema:"fuel per light year with the pilot's jump skills"`
}

// CompareHaulHullsOutput is compare_haul_hulls' result.
type CompareHaulHullsOutput struct {
	Hulls []HaulHull `json:"hulls"`
}

func compareHaulHullsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[CompareHaulHullsInput, CompareHaulHullsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CompareHaulHullsInput) (*mcpsdk.CallToolResult, CompareHaulHullsOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		classTypes, err := sdeStore.PublishedTypesInGroups(ctx, p.ua(), hullClassGroups)
		if err != nil {
			return nil, CompareHaulHullsOutput{}, fmt.Errorf("hull classes: %w", err)
		}
		var types []sde.TypeBasic
		if len(in.Classes) == 0 && len(in.Hulls) == 0 {
			types = classTypes
		}
		for _, c := range in.Classes {
			n := len(types)
			for _, t := range classTypes {
				if strings.EqualFold(t.GroupName, strings.TrimSpace(c)) {
					types = append(types, t)
				}
			}
			if len(types) == n {
				var known []string
				for _, t := range classTypes {
					if !slices.Contains(known, t.GroupName) {
						known = append(known, t.GroupName)
					}
				}
				return nil, CompareHaulHullsOutput{}, fmt.Errorf("unknown ship class %q; known: %s", c, strings.Join(known, ", "))
			}
		}
		if len(in.Hulls) > 0 {
			names := make([]string, len(in.Hulls))
			for i, h := range in.Hulls {
				names[i] = strings.TrimSpace(h)
			}
			ids, err := sdeStore.TypesByName(ctx, p.ua(), names)
			if err != nil {
				return nil, CompareHaulHullsOutput{}, fmt.Errorf("hulls by name: %w", err)
			}
			var want []int64
			for _, h := range names {
				id, ok := ids[h]
				if !ok {
					return nil, CompareHaulHullsOutput{}, fmt.Errorf("unknown hull %q", h)
				}
				want = append(want, id)
			}
			basics, err := p.typeBasics(ctx, want)
			if err != nil {
				return nil, CompareHaulHullsOutput{}, fmt.Errorf("hull types: %w", err)
			}
			for _, id := range want {
				if b := basics[id]; b.CategoryID != shipCategoryID {
					return nil, CompareHaulHullsOutput{}, fmt.Errorf("%s isn't a ship", b.Name)
				} else {
					types = append(types, b)
				}
			}
		}

		// A hull named twice, or both by name and in a named class, is
		// compared once.
		seen := map[int64]bool{}
		types = slices.DeleteFunc(types, func(t sde.TypeBasic) bool {
			dup := seen[t.TypeID]
			seen[t.TypeID] = true
			return dup
		})

		var skills map[int64]int
		if in.PilotID != 0 {
			if skills, err = p.skills(ctx, in.PilotID); err != nil {
				return nil, CompareHaulHullsOutput{}, err
			}
		}
		cloak, err := p.fitRule(ctx, covertOpsCloakTypeID)
		if err != nil {
			return nil, CompareHaulHullsOutput{}, err
		}
		nullifier, err := p.fitRule(ctx, interdictionNullifierTypeID)
		if err != nil {
			return nil, CompareHaulHullsOutput{}, err
		}

		out := CompareHaulHullsOutput{Hulls: []HaulHull{}}
		for _, t := range types {
			h, err := p.haulHull(ctx, t, cloak, nullifier, skills)
			if err != nil {
				return nil, CompareHaulHullsOutput{}, err
			}
			out.Hulls = append(out.Hulls, h)
		}
		return nil, out, nil
	}
}

// haulHull gathers one hull's SDE data and describes it, with the pilot's
// view when skills are given.
func (p *planner) haulHull(ctx context.Context, t sde.TypeBasic, cloak, nullifier fitRule, skills map[int64]int) (HaulHull, error) {
	typ, _, err := p.sde.Type(ctx, p.ua(), t.TypeID)
	if err != nil {
		return HaulHull{}, fmt.Errorf("type %s: %w", t.Name, err)
	}
	attrs, err := p.sde.TypeAttributes(ctx, p.ua(), t.TypeID)
	if err != nil {
		return HaulHull{}, fmt.Errorf("attributes of %s: %w", t.Name, err)
	}
	effects := map[int64]bool{}
	for _, e := range typ.DogmaEffects {
		effects[e.EffectID] = true
	}
	var mass, capacity float64
	if typ.Mass != nil {
		mass = *typ.Mass
	}
	if typ.Capacity != nil {
		capacity = *typ.Capacity
	}
	facts := nav.DescribeHull(nav.HullInput{TypeID: t.TypeID, Name: t.Name, GroupID: t.GroupID, GroupName: t.GroupName, MassKg: mass, Attrs: attrs, Effects: effects,
		CovertCloakFits: cloak.allows(t.TypeID, t.GroupID), NullifierFits: nullifier.allows(t.TypeID, t.GroupID)})
	h := HaulHull{HullFacts: facts, HoldsM3: cargo.Holds(capacity, attrs)}
	if facts.Jump != nil {
		h.FuelName = p.typeName(ctx, facts.Jump.FuelTypeID)
	}
	if skills != nil {
		can := nav.Trained(skills, requiredSkills(attrs))
		h.PilotCanFly = &can
		if facts.Jump != nil {
			drive, err := p.jumpDrive(ctx, t.TypeID)
			if err != nil {
				return HaulHull{}, err
			}
			js := nav.JumpSkills{JumpDriveCalibration: skills[skillJumpDriveCalibration], JumpFuelConservation: skills[skillJumpFuelConservation], JumpFreighters: skills[skillJumpFreighters]}
			h.PilotRangeLY, h.PilotFuelPerLY = nav.JumpRange(drive, js), nav.FuelPerLY(drive, js)
		}
	}
	return h, nil
}
