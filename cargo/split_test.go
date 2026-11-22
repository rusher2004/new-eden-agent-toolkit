package cargo_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
)

func TestSplitLoad_ProportionalAndWithinHolds(t *testing.T) {
	load := cargo.Load{Number: 1, M3: 300000, Items: []cargo.LoadItem{
		{TypeID: 34, Name: "Tritanium", Quantity: 10000000, M3: 100000, Dest: "A"},
		{TypeID: 21039, Name: "Capital Siege Array", Quantity: 20, M3: 200000, Dest: "A"},
	}}
	// Two JFs: 263,351 m³ and 158,400 m³.
	holds := []float64{263351, 158400}
	got, err := cargo.SplitLoad(load, holds, nil)
	if err != nil {
		t.Fatal(err)
	}
	total := map[int64]int64{}
	for i, l := range got {
		if l.M3 > holds[i]+1e-6 {
			t.Errorf("hull %d carries %v m³, over its %v", i, l.M3, holds[i])
		}
		for _, it := range l.Items {
			total[it.TypeID] += it.Quantity
		}
	}
	if total[34] != 10000000 || total[21039] != 20 {
		t.Errorf("quantities = %v, want everything carried exactly once", total)
	}
	if share := got[0].M3 / 300000; math.Abs(share-263351.0/421751) > 0.05 {
		t.Errorf("first hull carries %.2f of the load, want about its hold's share", share)
	}
}

func TestSplitLoad_PortionsAndOverflow(t *testing.T) {
	load := cargo.Load{Number: 1, M3: 1000, Items: []cargo.LoadItem{{TypeID: 1, Name: "Ore", Quantity: 1000, M3: 1000, Dest: "A"}}}
	got, err := cargo.SplitLoad(load, []float64{500, 500}, map[int64]int64{1: 100})
	if err != nil || got[0].Items[0].Quantity%100 != 0 || got[1].Items[0].Quantity%100 != 0 {
		t.Errorf("portions = %+v, %v", got, err)
	}
	if _, err := cargo.SplitLoad(load, []float64{400, 400}, nil); err == nil {
		t.Error("a load bigger than every hold together: want an error")
	}
	big := cargo.Load{Number: 1, M3: 600, Items: []cargo.LoadItem{{TypeID: 2, Name: "Container", Quantity: 1, M3: 600, Dest: "A"}}}
	if _, err := cargo.SplitLoad(big, []float64{500, 500}, nil); err == nil {
		t.Error("an item bigger than any one hold: want an error")
	}
}
