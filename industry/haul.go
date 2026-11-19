package industry

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

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

// Cargo is something to haul.
type Cargo struct {
	TypeID   int64   `json:"typeId"`
	Name     string  `json:"name"`
	Quantity int64   `json:"quantity"`
	UnitM3   float64 `json:"unitM3"`
	Portion  int64   `json:"portion,omitempty"` // keep quantities in multiples of this (e.g. 100 for ore); 0 or 1 = any
	Dest     string  `json:"dest"`              // drop-off label, e.g. "Sotiyo" or "Tatara"
}

// LoadItem is one cargo line in a load.
type LoadItem struct {
	TypeID   int64   `json:"typeId"`
	Name     string  `json:"name"`
	Quantity int64   `json:"quantity"`
	M3       float64 `json:"m3"`
	Dest     string  `json:"dest"`
}

// Load is one trip's cargo.
type Load struct {
	Number int                `json:"number"`
	Items  []LoadItem         `json:"items"`
	M3     float64            `json:"m3"`
	ByDest map[string]float64 `json:"byDest"`
}

// PackLoads splits cargo into the fewest loads that fit the hold, each an
// even slice of everything, so that work can start when the first load lands.
// Items with a unit (or portion) of 1 m³ or more are spread by cumulative
// rounding; smaller items fill the remaining space in proportion to it.
func PackLoads(items []Cargo, holdM3 float64) ([]Load, error) {
	if holdM3 <= 0 {
		return nil, fmt.Errorf("pack loads: hold must be positive")
	}
	total := 0.0
	for _, it := range items {
		if it.UnitM3*float64(portionOf(it)) > holdM3 {
			return nil, fmt.Errorf("pack loads: one %s (%.0f m³) doesn't fit a %.0f m³ hold", it.Name, it.UnitM3*float64(portionOf(it)), holdM3)
		}
		total += it.UnitM3 * float64(it.Quantity)
	}
	if total == 0 {
		return nil, nil
	}
	for n := int(math.Ceil(total / holdM3)); n <= int(math.Ceil(total/holdM3))*2+len(items)+1; n++ {
		if loads, ok := tryPack(items, holdM3, n); ok {
			return loads, nil
		}
	}
	return nil, fmt.Errorf("pack loads: couldn't fit %.0f m³ into %.0f m³ loads", total, holdM3)
}

func portionOf(c Cargo) int64 { return max(c.Portion, 1) }

func tryPack(items []Cargo, hold float64, n int) ([]Load, bool) {
	qty := make([]map[int]int64, n)
	m3 := make([]float64, n)
	for k := range qty {
		qty[k] = map[int]int64{}
	}
	spread := func(i int, units int64, cum []float64) {
		p := portionOf(items[i])
		for k := range n {
			u := int64(math.Round(float64(units)*cum[k+1])) - int64(math.Round(float64(units)*cum[k]))
			if u > 0 {
				qty[k][i] += u * p
				m3[k] += float64(u*p) * items[i].UnitM3
			}
		}
	}
	even := make([]float64, n+1)
	for k := range even {
		even[k] = float64(k) / float64(n)
	}
	var small []int
	for i, it := range items {
		if it.UnitM3*float64(portionOf(it)) >= 1 {
			spread(i, it.Quantity/portionOf(it), even)
		} else {
			small = append(small, i)
		}
	}
	free := 0.0
	rem := make([]float64, n)
	for k := range n {
		rem[k] = hold - m3[k]
		if rem[k] < 0 {
			return nil, false
		}
		free += rem[k]
	}
	smallM3 := 0.0
	for _, i := range small {
		smallM3 += items[i].UnitM3 * float64(items[i].Quantity)
	}
	if smallM3 > free {
		return nil, false
	}
	cum := make([]float64, n+1)
	for k := range n {
		cum[k+1] = cum[k] + rem[k]/free
	}
	for _, i := range small {
		spread(i, items[i].Quantity/portionOf(items[i]), cum)
	}
	// Remainders that aren't a whole portion go on the last load.
	for i, it := range items {
		if r := it.Quantity % portionOf(it); r > 0 {
			qty[n-1][i] += r
			m3[n-1] += float64(r) * it.UnitM3
		}
	}
	loads := make([]Load, n)
	for k := range n {
		if m3[k] > hold+1e-6 {
			return nil, false
		}
		l := Load{Number: k + 1, ByDest: map[string]float64{}}
		for i, it := range items {
			if q := qty[k][i]; q > 0 {
				v := float64(q) * it.UnitM3
				l.Items = append(l.Items, LoadItem{TypeID: it.TypeID, Name: it.Name, Quantity: q, M3: v, Dest: it.Dest})
				l.M3 += v
				l.ByDest[it.Dest] += v
			}
		}
		loads[k] = l
	}
	return loads, true
}

// MultibuyLine is one line of an in-game Multibuy list.
type MultibuyLine struct {
	Name     string
	Quantity int64
}

// Multibuy formats lines for the in-game Multibuy "Buy All" import: one
// "Name<TAB>Quantity" line per item, no thousands separators. Duplicate names
// are merged; order follows first appearance.
func Multibuy(lines []MultibuyLine) string {
	idx := map[string]int{}
	var merged []MultibuyLine
	for _, l := range lines {
		if l.Quantity <= 0 || l.Name == "" {
			continue
		}
		if i, ok := idx[l.Name]; ok {
			merged[i].Quantity += l.Quantity
			continue
		}
		idx[l.Name] = len(merged)
		merged = append(merged, l)
	}
	var b strings.Builder
	for i, l := range merged {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s\t%d", l.Name, l.Quantity)
	}
	return b.String()
}

// SortLoadItems orders a load's items by drop-off then name, for display.
func SortLoadItems(l *Load) {
	sort.SliceStable(l.Items, func(i, j int) bool {
		if l.Items[i].Dest != l.Items[j].Dest {
			return l.Items[i].Dest < l.Items[j].Dest
		}
		return l.Items[i].Name < l.Items[j].Name
	})
}
