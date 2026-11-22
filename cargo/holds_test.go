package cargo_test

import (
	"maps"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
)

func TestHolds(t *testing.T) {
	// An Archon: cargo, ship maintenance bay, fleet hangar, fuel bay, fighter hangar.
	got := cargo.Holds(1000, map[int64]float64{908: 2000000, 912: 50000, 1549: 140000, 2055: 60000, 37: 80})
	want := map[string]float64{"cargo": 1000, "ship_maintenance_bay": 2000000, "fleet_hangar": 50000, "fuel_bay": 140000, "fighter_hangar": 60000}
	if !maps.Equal(got, want) {
		t.Errorf("holds = %v, want %v", got, want)
	}
	if got := cargo.Holds(0, nil); len(got) != 0 {
		t.Errorf("no holds = %v", got)
	}
}
