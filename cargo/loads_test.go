package cargo_test

import (
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
)

func TestPackLoads_EvenSlices(t *testing.T) {
	items := []cargo.Cargo{
		{TypeID: 57487, Name: "Capital Core Temperature Regulator", Quantity: 15, UnitM3: 28000, Dest: "Sotiyo"},
		{TypeID: 35, Name: "Pyerite", Quantity: 159146178, UnitM3: 0.01, Dest: "Sotiyo"},
		{TypeID: 62555, Name: "Compressed Gneiss IV-Grade", Quantity: 2413600, UnitM3: 0.05, Portion: 100, Dest: "Tatara"},
	}
	loads, err := cargo.PackLoads(items, 158400)
	if err != nil {
		t.Fatal(err)
	}
	sums := map[int64]int64{}
	for _, l := range loads {
		if l.M3 > 158400 {
			t.Errorf("load %d is %v m³", l.Number, l.M3)
		}
		for _, it := range l.Items {
			sums[it.TypeID] += it.Quantity
			if it.TypeID == 62555 && it.Quantity%100 != 0 {
				t.Errorf("ore split mid-portion: %d", it.Quantity)
			}
		}
		if l.ByDest["Tatara"] == 0 {
			t.Errorf("load %d has no ore", l.Number)
		}
	}
	for _, it := range items {
		if sums[it.TypeID] != it.Quantity {
			t.Errorf("%s: packed %d of %d", it.Name, sums[it.TypeID], it.Quantity)
		}
	}
	if len(loads) != 14 {
		t.Errorf("loads = %d, want 14 (2,132,142 m³ / 158,400 = 13.5)", len(loads))
	}
}

func TestPackLoads_TooBig(t *testing.T) {
	_, err := cargo.PackLoads([]cargo.Cargo{{Name: "Minokawa", Quantity: 1, UnitM3: 1300000}}, 158400)
	if err == nil {
		t.Error("a packaged capital fit a jump freighter")
	}
}
