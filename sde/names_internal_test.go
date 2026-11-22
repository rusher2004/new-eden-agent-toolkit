package sde

import "testing"

func TestGameName(t *testing.T) {
	for _, tt := range []struct{ name, system, want string }{
		{"Vlillirier 7 - Moon 19 - FedMart Warehouse", "Vlillirier", "Vlillirier VII - Moon 19 - FedMart Warehouse"},
		{"Eugales 5 - Federal Defense Union Assembly Plant", "Eugales", "Eugales V - Federal Defense Union Assembly Plant"},
		{"Jita 4", "Jita", "Jita IV"},
		{"Jita 14 - Asteroid Belt 1", "Jita", "Jita XIV - Asteroid Belt 1"},
		{"Jita IV - Moon 4 - Caldari Navy Assembly Plant", "Jita", "Jita IV - Moon 4 - Caldari Navy Assembly Plant"},
		{"Jita", "Jita", "Jita"},
		{"Stargate (Perimeter)", "Jita", "Stargate (Perimeter)"},
		{"Sentinel MZ 3 - Moon 1", "Sentinel MZ", "Sentinel MZ III - Moon 1"},
		{"J123456 2", "J123456", "J123456 II"},
		{"Jita 04", "Jita", "Jita 04"},
		{"Jita 4", "", "Jita 4"},
	} {
		if got := gameName(tt.name, tt.system); got != tt.want {
			t.Errorf("gameName(%q, %q) = %q, want %q", tt.name, tt.system, got, tt.want)
		}
	}
}
