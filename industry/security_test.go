package industry_test

import (
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

func TestRigMultiplier(t *testing.T) {
	tests := []struct {
		band nav.Band
		want float64
	}{
		{nav.Highsec, 1.0},
		{nav.Lowsec, 1.9},
		{nav.Nullsec, 2.1},
		{nav.Wormhole, 2.1},
	}
	for _, tt := range tests {
		if got := industry.RigMultiplier(tt.band, 1.0, 1.9, 2.1); got != tt.want {
			t.Errorf("RigMultiplier(%v) = %v, want %v", tt.band, got, tt.want)
		}
	}
}
