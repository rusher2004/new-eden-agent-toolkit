package nav

import "math"

// Position is a solar system's SDE coordinates in meters.
type Position struct{ X, Y, Z float64 }

// LightYears between two systems, using EVE's 9.46e15 m light year.
func LightYears(a, b Position) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx+dy*dy+dz*dz) / MetersPerLightYear
}

// JumpDrive is a jump-capable hull's drive, from SDE dogma.
type JumpDrive struct {
	ShipTypeID         int64
	ShipName           string
	BaseRangeLY        float64 // 867
	FuelPerLY          float64 // 868
	FuelTypeID         int64   // 866
	SkillBonusPerLevel float64 // 1312 for jump freighters, percent per Jump Freighters level (e.g. -10); 0 otherwise
	FatigueMultiplier  float64 // 1971; 0.1 for jump freighters; 0 or 1 = no reduction
}

// JumpSkills are a pilot's jump skill levels.
type JumpSkills struct {
	JumpDriveCalibration int
	JumpFuelConservation int
	JumpFreighters       int
}

// JumpRange is base range × (1 + 0.2 × Jump Drive Calibration).
func JumpRange(d JumpDrive, s JumpSkills) float64 {
	return d.BaseRangeLY * (1 + 0.2*float64(s.JumpDriveCalibration))
}

// FuelPerLY after Jump Fuel Conservation (−10%/level) and the hull's own
// skill bonus (Jump Freighters for jump freighters).
func FuelPerLY(d JumpDrive, s JumpSkills) float64 {
	return d.FuelPerLY * (1 - 0.1*float64(s.JumpFuelConservation)) * (1 + d.SkillBonusPerLevel/100*float64(s.JumpFreighters))
}

// JumpFuel is fuel for one jump, rounded to the nearest unit (matches
// dotlan to the unit; see MetersPerLightYear).
func JumpFuel(d JumpDrive, s JumpSkills, ly float64) int64 {
	return int64(math.Round(FuelPerLY(d, s) * ly))
}

// Fatigue applies one jump of ly light years to existing fatigue (minutes).
// It returns the new fatigue and the jump activation cooldown, in minutes.
// The hull's fatigue multiplier scales the distance.
func Fatigue(prevFatigue, ly, multiplier float64) (fatigue, cooldown float64) {
	if multiplier <= 0 {
		multiplier = 1
	}
	d := ly * multiplier
	fatigue = math.Min(math.Max(prevFatigue*(1+d), fatigueMinBase*(1+d)), fatigueCapMinutes)
	cooldown = math.Min(math.Max(1+d, prevFatigue/10), cooldownCapMinutes)
	return fatigue, cooldown
}
