package nav_test

import (
	"slices"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

func TestCynoFuelLoads(t *testing.T) {
	tests := []struct {
		name     string
		perLight float64
		aboard   int64
		perHold  int
		lights   int
		want     []nav.CynoFuelLoad
	}{
		// Issue #39: 242 aboard covers the first light; the second needs 158 more.
		{"fuel aboard covers the first light", 200, 242, 1, 2, []nav.CynoFuelLoad{{BeforeLight: 2, AddUnits: 158, FuelAfter: 200, Lights: 1}}},
		{"empty hold, one light per load", 200, 0, 1, 2, []nav.CynoFuelLoad{
			{BeforeLight: 1, AddUnits: 200, FuelAfter: 200, Lights: 1},
			{BeforeLight: 2, AddUnits: 200, FuelAfter: 200, Lights: 1},
		}},
		{"partial load tops up to a full hold", 50, 30, 3, 5, []nav.CynoFuelLoad{
			{BeforeLight: 1, AddUnits: 120, FuelAfter: 150, Lights: 3},
			{BeforeLight: 4, AddUnits: 100, FuelAfter: 100, Lights: 2},
		}},
		{"enough aboard", 200, 500, 2, 2, nil},
		{"fuel doesn't fit", 200, 0, 0, 2, nil},
		{"fractional fuel rounds up", 96.5, 0, 2, 2, []nav.CynoFuelLoad{{BeforeLight: 1, AddUnits: 193, FuelAfter: 193, Lights: 2}}},
	}
	for _, tt := range tests {
		if got := nav.CynoFuelLoads(tt.perLight, tt.aboard, tt.perHold, tt.lights); !slices.Equal(got, tt.want) {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
