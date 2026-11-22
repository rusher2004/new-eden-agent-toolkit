package nav

import (
	"math"
	"strconv"
)

// Band is a security band. It picks which rig security multiplier applies and
// which activities are allowed.
type Band int

const (
	Highsec Band = iota
	Lowsec
	Nullsec
	Wormhole
)

func (b Band) String() string {
	switch b {
	case Highsec:
		return "highsec"
	case Lowsec:
		return "lowsec"
	case Nullsec:
		return "nullsec"
	case Wormhole:
		return "wormhole"
	}
	return "unknown"
}

// Region IDs with special rules.
const (
	PochvenRegionID       int64 = 10000070
	ZarzakhRegionID       int64 = 10001000
	FirstWormholeRegionID int64 = 11000000
)

// DisplaySecurity rounds a true security status the way the game displays it:
// to one decimal, except that any status strictly between 0 and 0.05 shows as
// 0.1 rather than 0.0.
func DisplaySecurity(sec float64) float64 {
	if sec > 0 && sec < 0.05 {
		return 0.1
	}
	return math.Round(sec*10) / 10
}

// BandOf returns the security band for a system's true security status and
// region.
func BandOf(sec float64, regionID int64) Band {
	if regionID >= FirstWormholeRegionID {
		return Wormhole
	}
	d := DisplaySecurity(sec)
	switch {
	case d >= 0.5:
		return Highsec
	case d > 0:
		return Lowsec
	default:
		return Nullsec
	}
}

// CynoLegal reports whether a jump can land in a system: only lowsec and
// nullsec k-space, excluding Pochven and Zarzakh.
func CynoLegal(sec float64, regionID int64) bool {
	if regionID == PochvenRegionID || regionID == ZarzakhRegionID {
		return false
	}
	b := BandOf(sec, regionID)
	return b == Lowsec || b == Nullsec
}

// Security is a system's security as players see it in game.
type Security struct {
	Status  float64 `json:"status" jsonschema:"true security status, usually to 6 decimal places"`
	Display string  `json:"display" jsonschema:"the value the game shows, rounded to one decimal; quote this one to the user, e.g. Aunenen 0.4"`
	Color   string  `json:"color" jsonschema:"the in-game color for the displayed value, as #RRGGBB, for formatted output"`
	Band    string  `json:"band" jsonschema:"highsec, lowsec, nullsec or wormhole"`
}

// SecurityOf describes a system's security for display.
func SecurityOf(sec float64, regionID int64) Security {
	return Security{Status: sec, Display: SecurityLabel(sec), Color: SecurityColor(sec), Band: BandOf(sec, regionID).String()}
}

// SecurityLabel formats the displayed security value as the game shows it,
// such as "0.4" or "-0.3".
func SecurityLabel(sec float64) string {
	d := DisplaySecurity(sec)
	if d == 0 {
		return "0.0" // never "-0.0"
	}
	return strconv.FormatFloat(d, 'f', 1, 64)
}

// SecurityColor is the in-game color of a system's displayed security
// value, as #RRGGBB.
func SecurityColor(sec float64) string {
	d := DisplaySecurity(sec)
	if d <= 0 {
		return securityColorNull
	}
	i := int(math.Round(d * 10))
	return securityColors[min(i, 10)]
}
