package industry

import "github.com/rusher2004/new-eden-agent-toolkit/nav"

// RigMultiplier picks a rig's own security multiplier for a band. Wormhole
// space uses the nullsec value.
func RigMultiplier(b nav.Band, hi, low, null float64) float64 {
	switch b {
	case nav.Highsec:
		return hi
	case nav.Lowsec:
		return low
	default:
		return null
	}
}
