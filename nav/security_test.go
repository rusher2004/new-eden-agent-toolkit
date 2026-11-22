package nav_test

import (
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

func TestBandOf(t *testing.T) {
	tests := []struct {
		name   string
		sec    float64
		region int64
		want   nav.Band
	}{
		{"Jita", 0.9459, 10000002, nav.Highsec},
		{"Aunenen rounds down to 0.4", 0.447790, 10000016, nav.Lowsec},
		{"exactly 0.45 rounds up", 0.45, 10000016, nav.Highsec},
		{"tiny positive shows as 0.1", 0.04, 10000016, nav.Lowsec},
		{"tiny negative is nullsec", -0.0099, 10000015, nav.Nullsec},
		{"zero is nullsec", 0, 10000015, nav.Nullsec},
		{"wormhole region", -1, 11000001, nav.Wormhole},
		{"Pochven", -1, nav.PochvenRegionID, nav.Nullsec},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nav.BandOf(tt.sec, tt.region); got != tt.want {
				t.Errorf("BandOf(%v, %d) = %v, want %v", tt.sec, tt.region, got, tt.want)
			}
		})
	}
}

func TestCynoLegal(t *testing.T) {
	tests := []struct {
		name   string
		sec    float64
		region int64
		want   bool
	}{
		{"highsec", 0.9, 10000002, false},
		{"lowsec", 0.36, 10000016, true},
		{"nullsec", -0.01, 10000051, true},
		{"wormhole", -1, 11000001, false},
		{"Pochven", -1, nav.PochvenRegionID, false},
		{"Zarzakh", -1, nav.ZarzakhRegionID, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nav.CynoLegal(tt.sec, tt.region); got != tt.want {
				t.Errorf("CynoLegal = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecurityOf(t *testing.T) {
	tests := []struct {
		name   string
		sec    float64
		region int64
		want   nav.Security
	}{
		{"Aunenen shows 0.4", 0.447790, 10000016, nav.Security{Status: 0.447790, Display: "0.4", Color: "#DC6C06", Band: "lowsec"}},
		{"Jita", 0.945913, 10000002, nav.Security{Status: 0.945913, Display: "0.9", Color: "#399AEB", Band: "highsec"}},
		{"exactly 0.45 shows 0.5", 0.45, 10000016, nav.Security{Status: 0.45, Display: "0.5", Color: "#F5FF83", Band: "highsec"}},
		{"tiny positive shows 0.1", 0.04, 10000016, nav.Security{Status: 0.04, Display: "0.1", Color: "#731F1F", Band: "lowsec"}},
		{"1.0", 1.0, 10000002, nav.Security{Status: 1.0, Display: "1.0", Color: "#2C75E1", Band: "highsec"}},
		{"zero", 0, 10000015, nav.Security{Status: 0, Display: "0.0", Color: "#8D3163", Band: "nullsec"}},
		{"tiny negative is 0.0, not -0.0", -0.0099, 10000015, nav.Security{Status: -0.0099, Display: "0.0", Color: "#8D3163", Band: "nullsec"}},
		{"nullsec", -0.255748, 10000010, nav.Security{Status: -0.255748, Display: "-0.3", Color: "#8D3163", Band: "nullsec"}},
		{"wormhole", -0.99, 11000001, nav.Security{Status: -0.99, Display: "-1.0", Color: "#8D3163", Band: "wormhole"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nav.SecurityOf(tt.sec, tt.region); got != tt.want {
				t.Errorf("SecurityOf(%v) = %+v, want %+v", tt.sec, got, tt.want)
			}
		})
	}
}
