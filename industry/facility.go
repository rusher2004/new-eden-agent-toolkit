package industry

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

// Activity names, as the SDE's indModifierSources spells them.
const (
	ActManufacturing    = "manufacturing"
	ActReaction         = "reaction"
	ActCopying          = "copying"
	ActInvention        = "invention"
	ActResearchMaterial = "researchMaterial"
	ActResearchTime     = "researchTime"
)

// Modifier kinds, as the SDE spells them.
const (
	KindMaterial = "material"
	KindTime     = "time"
	KindCost     = "cost"
)

// Facility kinds.
const (
	KindNPCStation = "npc_station"
	KindUpwell     = "upwell"
	KindStarbase   = "starbase"
)

// Service module names a facility profile records (ESI doesn't expose them).
const (
	SvcManufacturing      = "manufacturing"
	SvcCapitalShipyard    = "capital_shipyard"
	SvcSupercapShipyard   = "supercapital_shipyard"
	SvcResearch           = "research"
	SvcInvention          = "invention"
	SvcReprocessing       = "reprocessing"
	SvcBiochemicalReactor = "biochemical_reactor"
	SvcCompositeReactor   = "composite_reactor"
	SvcHybridReactor      = "hybrid_reactor"
)

// KnownServices lists every service name a profile may use.
var KnownServices = []string{SvcManufacturing, SvcCapitalShipyard, SvcSupercapShipyard, SvcResearch, SvcInvention, SvcReprocessing, SvcBiochemicalReactor, SvcCompositeReactor, SvcHybridReactor}

// Dogma attribute IDs read from rigs and structures. Mirrors sde/dogma.go so
// this package stays independent of sde/.
const (
	attrHiSecModifier     int64 = 2355
	attrLowSecModifier    int64 = 2356
	attrNullSecModifier   int64 = 2357
	attrDisallowInHighSec int64 = 1970
	attrRigSize           int64 = 1547
)

// Structure sizes from the rigSize attribute.
const (
	SizeMedium int64 = 2
	SizeLarge  int64 = 3
	SizeXL     int64 = 4
)

// ModifierSource is one industry modifier a structure or rig declares (the
// SDE's indModifierSources row).
type ModifierSource struct {
	Activity    string
	Kind        string
	AttributeID int64
	FilterID    int64 // 0 = all products
}

// StructureInput is an Upwell structure type's role-bonus data.
type StructureInput struct {
	TypeID     int64
	Name       string
	Attributes map[int64]float64
	Sources    []ModifierSource
}

// RigInput is a rig's data. TargetToSource maps the structure attribute a
// modifier names to the rig attribute that holds the percent amount (from the
// rig's dogma effect modifierInfo).
type RigInput struct {
	TypeID         int64
	Name           string
	Attributes     map[int64]float64
	Sources        []ModifierSource
	TargetToSource map[int64]int64
}

// TargetFilter limits a modifier to products in some categories or groups.
type TargetFilter struct {
	ID         int64
	Name       string
	Categories []int64
	Groups     []int64
}

func (f TargetFilter) matches(categoryID, groupID int64) bool {
	return slices.Contains(f.Categories, categoryID) || slices.Contains(f.Groups, groupID)
}

// Modifier is a resolved multiplier on one kind of one activity.
type Modifier struct {
	Activity string
	Kind     string
	Value    float64 // multiplier, e.g. 0.958 for a T1 nullsec rig's -4.2%
	FilterID int64
	Source   string // human-readable origin, e.g. "Sotiyo role bonus"
}

// Facility is a place industry happens, with everything needed to price and
// time jobs there and to decide what may run there.
type Facility struct {
	Name            string
	Kind            string
	StructureTypeID int64
	StructureName   string
	StructureSize   int64
	SystemID        int64
	RegionID        int64
	Security        float64
	Band            nav.Band
	Modifiers       []Modifier
	Filters         map[int64]TargetFilter
	Services        []string
	SupercapUpgrade bool
	FacilityTax     float64
	Warnings        []string
	// StarbaseSecurityMax is the anchoring limit of the starbase array in use
	// (attribute 1032); 0 when not a starbase or unknown.
	StarbaseSecurityMax float64
}

// ResolveFacilityInput is everything ResolveFacility needs.
type ResolveFacilityInput struct {
	Name                string
	Kind                string
	Structure           *StructureInput // Upwell only
	Rigs                []RigInput      // Upwell only
	Filters             map[int64]TargetFilter
	SystemID            int64
	RegionID            int64
	Security            float64
	Services            []string
	SupercapUpgrade     bool
	FacilityTax         float64 // ignored for NPC stations (NPCFacilityTax applies)
	StarbaseSecurityMax float64
}

// ResolveFacility turns structure and rig data into resolved modifiers for
// the system's security band, and records problems as warnings.
func ResolveFacility(in ResolveFacilityInput) Facility {
	f := Facility{
		Name: in.Name, Kind: in.Kind, SystemID: in.SystemID, RegionID: in.RegionID, Security: in.Security,
		Band: nav.BandOf(in.Security, in.RegionID), Filters: in.Filters, Services: in.Services,
		SupercapUpgrade: in.SupercapUpgrade, FacilityTax: in.FacilityTax, StarbaseSecurityMax: in.StarbaseSecurityMax,
	}
	if in.Kind == KindNPCStation {
		f.FacilityTax = NPCFacilityTax
		f.Services = []string{SvcManufacturing, SvcResearch, SvcInvention, SvcReprocessing}
		return f
	}
	if in.FacilityTax > IndustryTaxCap {
		f.Warnings = append(f.Warnings, fmt.Sprintf("facility tax %.2f%% is above the %.0f%% cap; the cap is used", in.FacilityTax*100, IndustryTaxCap*100))
		f.FacilityTax = IndustryTaxCap
	}
	if in.Kind != KindUpwell || in.Structure == nil {
		return f
	}
	s := in.Structure
	f.StructureTypeID, f.StructureName = s.TypeID, s.Name
	f.StructureSize = int64(s.Attributes[attrRigSize])
	if s.Attributes[attrDisallowInHighSec] != 0 && f.Band == nav.Highsec {
		f.Warnings = append(f.Warnings, fmt.Sprintf("%s can't be anchored in highsec", s.Name))
	}
	for _, src := range s.Sources {
		v, ok := s.Attributes[src.AttributeID]
		if !ok || v == 0 {
			continue
		}
		f.Modifiers = append(f.Modifiers, Modifier{Activity: src.Activity, Kind: src.Kind, Value: v, FilterID: src.FilterID, Source: s.Name + " role bonus"})
	}
	for _, rig := range in.Rigs {
		if size := int64(rig.Attributes[attrRigSize]); size != 0 && f.StructureSize != 0 && size != f.StructureSize {
			f.Warnings = append(f.Warnings, fmt.Sprintf("%s is size %d but %s takes size %d rigs; ignored", rig.Name, size, s.Name, f.StructureSize))
			continue
		}
		if rig.Attributes[attrDisallowInHighSec] != 0 && f.Band == nav.Highsec {
			f.Warnings = append(f.Warnings, fmt.Sprintf("%s can't be fitted in highsec; ignored", rig.Name))
			continue
		}
		secMult := RigMultiplier(f.Band, attrOr(rig.Attributes, attrHiSecModifier, 1), attrOr(rig.Attributes, attrLowSecModifier, 1), attrOr(rig.Attributes, attrNullSecModifier, 1))
		for _, src := range rig.Sources {
			pct := rig.Attributes[rig.TargetToSource[src.AttributeID]]
			if pct == 0 {
				continue
			}
			f.Modifiers = append(f.Modifiers, Modifier{
				Activity: src.Activity, Kind: src.Kind, Value: 1 + pct/100*secMult, FilterID: src.FilterID,
				Source: fmt.Sprintf("%s (%+.1f%% × %.1f %s)", rig.Name, pct, secMult, f.Band),
			})
		}
	}
	return f
}

func attrOr(attrs map[int64]float64, id int64, def float64) float64 {
	if v, ok := attrs[id]; ok {
		return v
	}
	return def
}

// Multipliers are the combined facility multipliers for one job.
type Multipliers struct {
	Material float64
	Time     float64
	Cost     float64 // multiplies the system cost index only
	Sources  []string
}

// Multipliers combines every modifier that applies to an activity and a
// product. Modifiers multiply together.
func (f Facility) Multipliers(activity string, categoryID, groupID int64) Multipliers {
	m := Multipliers{Material: 1, Time: 1, Cost: 1}
	for _, mod := range f.Modifiers {
		if mod.Activity != activity {
			continue
		}
		if mod.FilterID != 0 {
			flt, ok := f.Filters[mod.FilterID]
			if !ok || !flt.matches(categoryID, groupID) {
				continue
			}
		}
		switch mod.Kind {
		case KindMaterial:
			m.Material *= mod.Value
		case KindTime:
			m.Time *= mod.Value
		case KindCost:
			m.Cost *= mod.Value
		}
		m.Sources = append(m.Sources, fmt.Sprintf("%s %s ×%.4f: %s", activity, mod.Kind, mod.Value, mod.Source))
	}
	return m
}

// Product groups with special location rules. Source: EVE University,
// Manufacturing (build locations table) and the SDE's "Capital Ships" target
// filter 11.
var (
	// CapitalShipyardGroups need a capital shipyard (or a capable NPC
	// station) outside highsec: dreadnoughts, carriers, capital industrial
	// ships, force auxiliaries, lancer dreadnoughts and group 5120.
	CapitalShipyardGroups = []int64{485, 547, 883, 1538, 4594, 5120}
	// SupercapitalGroups need a Sotiyo with a supercapital shipyard and the
	// sovereignty upgrade: titans and supercarriers.
	SupercapitalGroups = []int64{30, 659}
	// LargeAssemblyArrayMaxGroups are ship groups a highsec starbase can't
	// build (anything above battleship): freighters, jump freighters,
	// industrial command ships, plus every capital and supercapital group.
	HighsecStarbaseBlockedGroups = []int64{513, 902, 941, 485, 547, 883, 1538, 4594, 5120, 30, 659}
)

// Reaction families by the SDE target filter their products fall in.
var reactionFilterService = map[string]string{
	"Hybrid Reactions":      SvcHybridReactor,
	"Biochemical Reactions": SvcBiochemicalReactor,
	"Composite Reactions":   SvcCompositeReactor,
}

// Product describes what a job makes, for location checks.
type Product struct {
	Name       string
	CategoryID int64
	GroupID    int64
}

func (f Facility) hasService(s string) bool { return slices.Contains(f.Services, s) }

// CanRun reports why an activity can't run here for a product, or nil.
func (f Facility) CanRun(activity string, p Product) error {
	var reasons []string
	add := func(format string, a ...any) { reasons = append(reasons, fmt.Sprintf(format, a...)) }
	isCap := slices.Contains(CapitalShipyardGroups, p.GroupID)
	isSuper := slices.Contains(SupercapitalGroups, p.GroupID)

	switch activity {
	case ActManufacturing:
		if f.Kind == KindUpwell && !f.hasService(SvcManufacturing) {
			add("%s has no manufacturing service", f.Name)
		}
		if isSuper {
			if f.Kind != KindUpwell || f.StructureName != "Sotiyo" || !f.hasService(SvcSupercapShipyard) || !f.SupercapUpgrade {
				add("%s needs a Sotiyo with a Standup Supercapital Shipyard in a system with the Supercapital Construction Facilities upgrade", p.Name)
			}
		} else if isCap {
			switch {
			case f.Band == nav.Highsec:
				add("%s can't be built in highsec", p.Name)
			case f.Kind == KindUpwell && (f.StructureSize < SizeLarge || !f.hasService(SvcCapitalShipyard)):
				add("%s needs a Large or XL structure with a Standup Capital Shipyard", p.Name)
			case f.Kind == KindStarbase && f.StarbaseSecurityMax != 0 && nav.DisplaySecurity(f.Security) > f.StarbaseSecurityMax:
				add("this starbase array can't operate above %.1f security", f.StarbaseSecurityMax)
			}
		}
		if f.Kind == KindStarbase && f.Band == nav.Highsec && slices.Contains(HighsecStarbaseBlockedGroups, p.GroupID) {
			add("starbases at 0.5 and above are limited to Large Assembly Arrays (battleship and smaller)")
		}
	case ActReaction:
		if f.Band == nav.Highsec {
			add("reactions can't run in highsec")
		}
		if f.Kind != KindUpwell || (f.StructureName != "Athanor" && f.StructureName != "Tatara") {
			add("reactions need an Athanor or Tatara")
		}
		svc := ""
		for _, flt := range f.Filters {
			if s, ok := reactionFilterService[flt.Name]; ok && flt.matches(p.CategoryID, p.GroupID) {
				svc = s
			}
		}
		if svc != "" && !f.hasService(svc) {
			add("%s needs a %s", p.Name, strings.ReplaceAll(svc, "_", " "))
		}
	case ActCopying, ActResearchMaterial, ActResearchTime:
		if f.Kind == KindUpwell && !f.hasService(SvcResearch) {
			add("%s has no research service", f.Name)
		}
	case ActInvention:
		if f.Kind == KindUpwell && !f.hasService(SvcInvention) {
			add("%s has no invention service", f.Name)
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	return fmt.Errorf("can't run %s for %s at %s: %s", activity, p.Name, f.Name, strings.Join(reasons, "; "))
}
