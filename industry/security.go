package industry

import "math"

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

// RigMultiplier picks a rig's own security multiplier for a band. Wormhole
// space uses the nullsec value.
func RigMultiplier(b Band, hi, low, null float64) float64 {
	switch b {
	case Highsec:
		return hi
	case Lowsec:
		return low
	default:
		return null
	}
}
