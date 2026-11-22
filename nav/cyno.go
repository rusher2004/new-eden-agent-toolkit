package nav

import "math"

// CynoFuelLoad is one time a cyno lighter adds fuel to the hold.
type CynoFuelLoad struct {
	BeforeLight int   `json:"beforeLight" jsonschema:"the light this fuel must be aboard for, counting from 1"`
	AddUnits    int64 `json:"addUnits" jsonschema:"units of fuel to add to what's aboard then"`
	FuelAfter   int64 `json:"fuelAfter" jsonschema:"units aboard once it's added"`
	Lights      int   `json:"lights" jsonschema:"lights this load covers, from beforeLight on"`
}

// CynoFuelLoads lays out every fuel load lighting lights cynos takes, at
// perLight units each, starting with aboard units in the hold and fitting at
// most perHold lights' fuel at a time. The fuel aboard is burned first, and
// each load tops it up to cover as many of the remaining lights as the hold
// fits. It returns nil when nothing needs adding, or when a light's fuel
// doesn't fit (perHold below 1).
func CynoFuelLoads(perLight float64, aboard int64, perHold, lights int) []CynoFuelLoad {
	if perLight <= 0 || perHold < 1 {
		return nil
	}
	const eps = 1e-9
	var loads []CynoFuelLoad
	fuel := float64(aboard)
	for next := 1; next <= lights; {
		if covered := int(math.Floor(fuel/perLight + eps)); covered > 0 {
			n := min(covered, lights-next+1)
			fuel -= float64(n) * perLight
			next += n
			continue
		}
		n := min(perHold, lights-next+1)
		add := int64(math.Ceil(float64(n)*perLight - fuel - eps))
		fuel += float64(add)
		loads = append(loads, CynoFuelLoad{BeforeLight: next, AddUnits: add, FuelAfter: int64(math.Round(fuel)), Lights: n})
	}
	return loads
}
