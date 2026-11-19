package industry_test

import (
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
)

func TestBandOf(t *testing.T) {
	tests := []struct {
		name   string
		sec    float64
		region int64
		want   industry.Band
	}{
		{"Jita", 0.9459, 10000002, industry.Highsec},
		{"Aunenen rounds down to 0.4", 0.447790, 10000016, industry.Lowsec},
		{"exactly 0.45 rounds up", 0.45, 10000016, industry.Highsec},
		{"tiny positive shows as 0.1", 0.04, 10000016, industry.Lowsec},
		{"tiny negative is nullsec", -0.0099, 10000015, industry.Nullsec},
		{"zero is nullsec", 0, 10000015, industry.Nullsec},
		{"wormhole region", -1, 11000001, industry.Wormhole},
		{"Pochven", -1, industry.PochvenRegionID, industry.Nullsec},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := industry.BandOf(tt.sec, tt.region); got != tt.want {
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
		{"Pochven", -1, industry.PochvenRegionID, false},
		{"Zarzakh", -1, industry.ZarzakhRegionID, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := industry.CynoLegal(tt.sec, tt.region); got != tt.want {
				t.Errorf("CynoLegal = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRigMultiplier(t *testing.T) {
	tests := []struct {
		band industry.Band
		want float64
	}{
		{industry.Highsec, 1.0},
		{industry.Lowsec, 1.9},
		{industry.Nullsec, 2.1},
		{industry.Wormhole, 2.1},
	}
	for _, tt := range tests {
		if got := industry.RigMultiplier(tt.band, 1.0, 1.9, 2.1); got != tt.want {
			t.Errorf("RigMultiplier(%v) = %v, want %v", tt.band, got, tt.want)
		}
	}
}
