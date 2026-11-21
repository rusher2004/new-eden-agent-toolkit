package nav

import (
	"math"
	"slices"
)

// Place is a station or structure a ship might dock at.
type Place struct {
	SystemID int64
	Name     string
	// TypeName is the structure's type, such as Fortizar; empty for an NPC
	// station.
	TypeName string
}

// LandingCheck is what a jump leg's landing means for the ship and the
// cyno lighter.
type LandingCheck struct {
	Leg       int      `json:"leg" jsonschema:"the jump leg, counting from 1"`
	System    string   `json:"system"`
	Security  Security `json:"security"`
	CynoTypes []string `json:"cynoTypes" jsonschema:"cyno types this hull can jump to; the lighter needs one of these"`
	DockAt    []string `json:"dockAt,omitempty" jsonschema:"places in the system this hull can dock: NPC stations if it can use them, and saved structures it fits"`
	NoDock    bool     `json:"noDock,omitempty" jsonschema:"no NPC station or saved structure here can take this hull: it waits in space (or a structure the user hasn't saved) between jumps"`
	// LighterGates is how far a single lighter has to travel by gate from
	// the previous landing to light this one; -1 when no gate path joins
	// them, absent for the first jump.
	LighterGates *int `json:"lighterGatesFromPreviousLanding,omitempty"`
}

// CheckLandings looks at each jump leg's landing for a hull of a ship
// group: the cyno types it needs, where it can dock (NPC stations in the
// system, if the hull may use them, and the given places it fits), and how
// far one lighter would travel by gate between consecutive landings.
func CheckLandings(g *RouteGraph, r Route, groupID int64, places []Place) []LandingCheck {
	docking := DockingFor(groupID)
	cyno := CynoTypesFor(groupID)
	var out []LandingCheck
	prev := int64(0)
	for _, l := range r.Legs {
		if l.Kind != "jump" {
			continue
		}
		sys := g.Systems[l.To]
		c := LandingCheck{Leg: len(out) + 1, System: l.ToName, Security: SecurityOf(sys.Security, sys.RegionID), CynoTypes: cyno}
		if docking.NPCStations && sys.HasNPCStation {
			c.DockAt = append(c.DockAt, "NPC station")
		}
		for _, pl := range places {
			if pl.SystemID == l.To && pl.TypeName != "" && CanDock(groupID, pl.TypeName) {
				c.DockAt = append(c.DockAt, pl.Name)
			}
		}
		c.NoDock = len(c.DockAt) == 0
		if prev != 0 {
			gates := g.GateJumps(prev, l.To)
			c.LighterGates = &gates
		}
		prev = l.To
		out = append(out, c)
	}
	return out
}

// SharedCynoTypes is the cyno types every hull in a group of ship groups
// can use, so they can jump to the same cyno.
func SharedCynoTypes(groupIDs []int64) []string {
	if len(groupIDs) == 0 {
		return nil
	}
	shared := CynoTypesFor(groupIDs[0])
	for _, g := range groupIDs[1:] {
		mine := CynoTypesFor(g)
		shared = slices.DeleteFunc(shared, func(t string) bool { return !slices.Contains(mine, t) })
	}
	return shared
}

// FuelCheck is whether a route's fuel fits the fuel bay.
type FuelCheck struct {
	RouteFuel     int64   `json:"routeFuel"`
	FuelBayM3     float64 `json:"fuelBayM3"`
	FuelBayUnits  int64   `json:"fuelBayUnits" jsonschema:"fuel units a full fuel bay holds"`
	FitsInFuelBay bool    `json:"fitsInFuelBay"`
	ExtraUnits    int64   `json:"extraUnits,omitempty" jsonschema:"fuel beyond the fuel bay, which has to ride in the cargo hold"`
	ExtraM3       float64 `json:"extraM3,omitempty" jsonschema:"cargo space that extra fuel takes"`
}

// FuelFit checks a route's fuel against a fuel bay.
func FuelFit(routeFuel int64, fuelBayM3, unitM3 float64) FuelCheck {
	c := FuelCheck{RouteFuel: routeFuel, FuelBayM3: fuelBayM3}
	if unitM3 > 0 {
		c.FuelBayUnits = int64(math.Floor(fuelBayM3/unitM3 + 1e-9))
	}
	c.FitsInFuelBay = routeFuel <= c.FuelBayUnits
	if !c.FitsInFuelBay {
		c.ExtraUnits = routeFuel - c.FuelBayUnits
		c.ExtraM3 = float64(c.ExtraUnits) * unitM3
	}
	return c
}
