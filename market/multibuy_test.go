package market_test

import (
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/market"
)

func TestMultibuy(t *testing.T) {
	got := market.Multibuy([]market.MultibuyLine{{"Tritanium", 1000}, {"Pyerite", 5}, {"Tritanium", 1}, {"Nothing", 0}})
	if got != "Tritanium\t1001\nPyerite\t5" {
		t.Errorf("multibuy = %q", got)
	}
}
