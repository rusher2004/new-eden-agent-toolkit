package nav

import (
	"math"
	"slices"
	"strings"
)

// Dogma attributes and effects behind a hull's movement facts. IDs checked
// against the SDE 2026-11-20.
const (
	AttrMaxVelocity         int64 = 37
	AttrInertiaModifier     int64 = 70
	AttrWarpScrambleStatus  int64 = 104 // negative is extra warp core strength
	AttrWarpSpeedMultiplier int64 = 600
	AttrCanJump             int64 = 861
	AttrJumpDriveFuelType   int64 = 866
	AttrJumpDriveRange      int64 = 867
	AttrJumpDriveFuelPerLY  int64 = 868
	AttrBaseWarpSpeed       int64 = 1281
	AttrDisallowInHighSec   int64 = 1970
	AttrJumpFatigue         int64 = 1971
	// EffectInterdictionNullification is the built-in nullification some
	// hulls carry (interceptorNullificationRoleBonus).
	EffectInterdictionNullification int64 = 8120
)

// Passenger attributes: a hull with one of these set can ride that kind of
// bridge or conduit jump. IDs checked against the SDE 2026-11-20.
var passengerAttributes = []struct {
	AttributeID int64
	Ride        string
}{
	{3320, "covert bridge"},
	{3322, "covert conduit"},
	{3325, "industrial bridge"},
	{3324, "industrial conduit"},
	{3319, "titan bridge"},
	{5682, "carrier conduit"},
}

// HullInput is what the SDE says about a hull, gathered by the caller.
type HullInput struct {
	TypeID    int64
	Name      string
	GroupID   int64
	GroupName string
	MassKg    float64
	Attrs     map[int64]float64
	Effects   map[int64]bool
	// CovertCloakFits and NullifierFits say whether the Covert Ops Cloaking
	// Device and the Interdiction Nullifier list the hull in their fitting
	// restrictions.
	CovertCloakFits bool
	NullifierFits   bool
}

// Docking is where a hull can dock.
type Docking struct {
	NPCStations bool     `json:"npcStations"`
	Upwell      []string `json:"upwell" jsonschema:"Upwell structures it can dock in, by base type (faction variants included)"`
	BuildOnly   []string `json:"buildOnly,omitempty" jsonschema:"structures that can build it but where it can't dock again"`
	Assumed     bool     `json:"assumed,omitempty" jsonschema:"no published rule covers this class; this is inferred from a similar class"`
}

// JumpFacts is a hull's jump drive, before the pilot's skills.
type JumpFacts struct {
	BaseRangeLY       float64  `json:"baseRangeLy"`
	FuelTypeID        int64    `json:"fuelTypeId"`
	BaseFuelPerLY     float64  `json:"baseFuelPerLy"`
	FatigueMultiplier float64  `json:"fatigueMultiplier" jsonschema:"scales the distance counted for jump fatigue; 1 is full fatigue"`
	CynoTypes         []string `json:"cynoTypes" jsonschema:"cyno types it can jump to"`
}

// HullFacts is a hull's movement facts, before skills, modules and rigs.
type HullFacts struct {
	TypeID           int64      `json:"typeId"`
	Name             string     `json:"name"`
	Class            string     `json:"class"`
	WarpSpeedAU      float64    `json:"warpSpeedAuPerSecond"`
	MaxVelocity      float64    `json:"maxVelocityMPerSecond"`
	MassKg           float64    `json:"massKg"`
	InertiaModifier  float64    `json:"inertiaModifier"`
	AlignSeconds     float64    `json:"alignSeconds" jsonschema:"ln(4) x inertia x mass / 1,000,000: an estimate before skills and fittings"`
	WarpCoreStrength int        `json:"warpCoreStrength,omitempty" jsonschema:"extra points of warp disruption it takes to stop it warping"`
	CovertOpsCloak   bool       `json:"covertOpsCloak" jsonschema:"can fit a Covert Ops Cloaking Device and warp cloaked"`
	Nullification    string     `json:"interdictionNullification,omitempty" jsonschema:"hull (always on) or module (can fit an Interdiction Nullifier)"`
	EntersHighsec    bool       `json:"entersHighsec" jsonschema:"can travel into highsec by gate"`
	Jump             *JumpFacts `json:"jump,omitempty"`
	Docking          Docking    `json:"docking"`
	Rides            []string   `json:"rides,omitempty" jsonschema:"bridges and conduit jumps it can be carried through"`
}

// AlignSeconds estimates the time to align to warp from mass and inertia:
// ln(4) x inertia modifier x mass / 1,000,000.
func AlignSeconds(massKg, inertia float64) float64 {
	return math.Log(4) * inertia * massKg / 1e6
}

// DescribeHull works out a hull's movement facts from its SDE data.
func DescribeHull(in HullInput) HullFacts {
	a := in.Attrs
	f := HullFacts{
		TypeID: in.TypeID, Name: in.Name, Class: in.GroupName,
		WarpSpeedAU: a[AttrBaseWarpSpeed] * a[AttrWarpSpeedMultiplier], MaxVelocity: a[AttrMaxVelocity],
		MassKg: in.MassKg, InertiaModifier: a[AttrInertiaModifier], AlignSeconds: AlignSeconds(in.MassKg, a[AttrInertiaModifier]),
		CovertOpsCloak: in.CovertCloakFits, EntersHighsec: a[AttrDisallowInHighSec] == 0, Docking: DockingFor(in.GroupID),
	}
	if wcs := -a[AttrWarpScrambleStatus]; wcs > 0 {
		f.WarpCoreStrength = int(wcs)
	}
	switch {
	case in.Effects[EffectInterdictionNullification]:
		f.Nullification = "hull"
	case in.NullifierFits:
		f.Nullification = "module"
	}
	if a[AttrCanJump] != 0 && a[AttrJumpDriveRange] > 0 {
		fatigue := a[AttrJumpFatigue]
		if fatigue <= 0 {
			fatigue = 1
		}
		f.Jump = &JumpFacts{BaseRangeLY: a[AttrJumpDriveRange], FuelTypeID: int64(a[AttrJumpDriveFuelType]), BaseFuelPerLY: a[AttrJumpDriveFuelPerLY],
			FatigueMultiplier: fatigue, CynoTypes: CynoTypesFor(in.GroupID)}
	}
	for _, p := range passengerAttributes {
		if a[p.AttributeID] != 0 {
			f.Rides = append(f.Rides, p.Ride)
		}
	}
	return f
}

// DockingFor is where hulls of a group can dock: the capital rules, or
// anywhere for every other ship.
func DockingFor(groupID int64) Docking {
	if d, ok := capitalDocking[groupID]; ok {
		return d
	}
	return Docking{NPCStations: true, Upwell: slices.Clone(allUpwell)}
}

// CanDock reports whether a hull of a group can dock at an Upwell structure,
// by its type name (faction variants such as 'Moreau' Fortizar match their
// base type).
func CanDock(groupID int64, structureTypeName string) bool {
	for _, base := range DockingFor(groupID).Upwell {
		if strings.Contains(structureTypeName, base) {
			return true
		}
	}
	return false
}

// CynoTypesFor is the cyno types a jump drive in a group can jump to.
func CynoTypesFor(groupID int64) []string {
	out := []string{"standard"}
	if slices.Contains(industrialCynoGroups, groupID) {
		out = append(out, "industrial")
	}
	if slices.Contains(covertCynoGroups, groupID) {
		out = append(out, "covert")
	}
	return out
}
