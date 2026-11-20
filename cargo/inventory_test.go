package cargo_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
)

const station = 60003760

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestItemVolumes(t *testing.T) {
	tests := []struct {
		name string
		it   cargo.Item
		want float64
	}{
		{"packaged ship uses packaged volume", cargo.Item{Quantity: 2, Volume: 28600, PackagedVolume: 2500, CategoryID: 6}, 5000},
		{"assembled ship uses assembled volume", cargo.Item{Quantity: 1, Singleton: true, Volume: 28600, PackagedVolume: 2500, CategoryID: 6}, 28600},
		{"no packaged volume falls back to volume", cargo.Item{Quantity: 1000, Volume: 0.01}, 10},
		{"zero quantity counts as one", cargo.Item{Singleton: true, Volume: 5}, 5},
	}
	for _, tt := range tests {
		if got := tt.it.StackVolume(); !near(got, tt.want) {
			t.Errorf("%s: StackVolume = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A Rifter with a fitted gun and two rigs, cargo, and a container in its
// hold; a loose container with ore; a packaged Venture; a Tengu with a rig.
func hangar() []cargo.Item {
	return []cargo.Item{
		{ItemID: 1, TypeID: 587, Name: "Rifter", CategoryID: 6, GroupID: 25, LocationID: station, Flag: "Hangar", Quantity: 1, Singleton: true, Volume: 27289, PackagedVolume: 2500, Owner: "Pilot"},
		{ItemID: 2, TypeID: 484, Name: "125mm Gatling AutoCannon I", LocationID: 1, Flag: "HiSlot0", Quantity: 1, Singleton: true, Volume: 5, Owner: "Pilot"},
		{ItemID: 3, TypeID: 31370, Name: "Small Cargohold Optimization I", LocationID: 1, Flag: "RigSlot0", Quantity: 1, Singleton: true, Volume: 5, Owner: "Pilot"},
		{ItemID: 4, TypeID: 31370, Name: "Small Cargohold Optimization I", LocationID: 1, Flag: "RigSlot1", Quantity: 1, Singleton: true, Volume: 5, Owner: "Pilot"},
		{ItemID: 5, TypeID: 34, Name: "Tritanium", LocationID: 1, Flag: "Cargo", Quantity: 1000, Volume: 0.01, Owner: "Pilot"},
		{ItemID: 6, TypeID: 3297, Name: "Small Secure Container", LocationID: 1, Flag: "Cargo", Quantity: 1, Singleton: true, Volume: 100, PackagedVolume: 10, Owner: "Pilot"},
		{ItemID: 7, TypeID: 35, Name: "Pyerite", LocationID: 6, Flag: "Unlocked", Quantity: 500, Volume: 0.01, Owner: "Pilot"},
		{ItemID: 8, TypeID: 3297, Name: "Small Secure Container", LocationID: station, Flag: "Hangar", Quantity: 1, Singleton: true, Volume: 100, PackagedVolume: 10, Owner: "Pilot"},
		{ItemID: 9, TypeID: 1230, Name: "Veldspar", LocationID: 8, Flag: "Unlocked", Quantity: 1000, Volume: 0.1, Owner: "Pilot"},
		{ItemID: 10, TypeID: 32880, Name: "Venture", CategoryID: 6, LocationID: station, Flag: "Hangar", Quantity: 1, Volume: 29500, PackagedVolume: 2500, Owner: "Pilot"},
		{ItemID: 11, TypeID: 29984, Name: "Tengu", CategoryID: 6, GroupID: cargo.StrategicCruiserGroupID, LocationID: station, Flag: "Hangar", Quantity: 1, Singleton: true, Volume: 92000, PackagedVolume: 5000, Owner: "Pilot"},
		{ItemID: 12, TypeID: 31370, Name: "Medium Rig", LocationID: 11, Flag: "RigSlot0", Quantity: 1, Singleton: true, Volume: 10, Owner: "Pilot"},
		{ItemID: 99, TypeID: 34, Name: "Elsewhere", LocationID: 60008494, Flag: "Hangar", Quantity: 1, Volume: 1, Owner: "Pilot"},
	}
}

func TestInventory_Totals(t *testing.T) {
	lines, got := cargo.Inventory(hangar(), station)
	want := cargo.InventoryTotals{
		ItemsM3:           100 + 2500, // the loose container (contents inside) and the packaged Venture
		AssembledShips:    2,
		ShipsAssembledM3:  27289 + 92000,
		ShipsPackagedM3:   2500 + 5000,
		ShipContentsM3:    5 + 10 + 100, // the gun, the Tritanium and the container; rigs are destroyed, the Tengu's rig stays
		ShipsWithRigsLost: 1,
	}
	want.AllRepackagedM3 = want.ItemsM3 + want.ShipsPackagedM3 + want.ShipContentsM3
	if !near(got.ItemsM3, want.ItemsM3) || got.AssembledShips != want.AssembledShips || !near(got.ShipsAssembledM3, want.ShipsAssembledM3) ||
		!near(got.ShipsPackagedM3, want.ShipsPackagedM3) || !near(got.ShipContentsM3, want.ShipContentsM3) || got.ShipsWithRigsLost != want.ShipsWithRigsLost ||
		!near(got.AllRepackagedM3, want.AllRepackagedM3) {
		t.Errorf("totals = %+v\nwant     %+v", got, want)
	}
	for _, l := range lines {
		if l.ItemID == 99 {
			t.Error("an item at another station was listed")
		}
	}
}

func TestInventory_TreeOrderAndShipDetails(t *testing.T) {
	lines, _ := cargo.Inventory(hangar(), station)
	byID := map[int64]cargo.InventoryLine{}
	pos := map[int64]int{}
	for i, l := range lines {
		byID[l.ItemID] = l
		pos[l.ItemID] = i
	}
	if len(lines) != 12 {
		t.Fatalf("got %d lines, want 12", len(lines))
	}
	if p := byID[7]; p.ParentItemID != 6 || p.Depth != 2 || pos[7] != pos[6]+1 {
		t.Errorf("Pyerite = %+v at %d, want depth 2 right after its container", p, pos[7])
	}
	rifter := byID[1].Ship
	if rifter == nil || len(rifter.RigsLost) != 2 || !near(rifter.ContentsM3, 115) || !near(rifter.PackagedM3, 2500) {
		t.Errorf("Rifter = %+v", rifter)
	}
	if !near(byID[1].ContentsM3, 110) {
		t.Errorf("Rifter hold contents = %v, want 110 (Tritanium and the container, not the gun)", byID[1].ContentsM3)
	}
	if tengu := byID[11].Ship; tengu == nil || len(tengu.RigsLost) != 0 || tengu.ContentsM3 != 0 {
		t.Errorf("Tengu = %+v, want its rig kept", tengu)
	}
	if byID[10].Ship != nil || byID[10].Assembled {
		t.Errorf("packaged Venture = %+v, want no assembled-ship details", byID[10])
	}
	if !near(byID[8].ContentsM3, 100) {
		t.Errorf("container contents = %v, want 100", byID[8].ContentsM3)
	}
}

func TestInventory_OfficesAreTransparent(t *testing.T) {
	items := []cargo.Item{
		{ItemID: 50, TypeID: 27, Name: "Office", LocationID: 1022167642188, Flag: "OfficeFolder", Quantity: 1, Singleton: true, Volume: 1e6, Owner: "Corp"},
		{ItemID: 51, TypeID: 34, Name: "Tritanium", LocationID: 50, Flag: "CorpSAG1", Quantity: 100, Volume: 0.01, Owner: "Corp"},
	}
	lines, totals := cargo.Inventory(items, 1022167642188)
	if len(lines) != 1 || lines[0].ItemID != 51 || lines[0].Depth != 0 || !near(totals.ItemsM3, 1) {
		t.Errorf("lines = %+v totals = %+v, want only the hangar item at depth 0", lines, totals)
	}
}

func TestInventory_ShipInMaintenanceBay(t *testing.T) {
	items := []cargo.Item{
		{ItemID: 1, Name: "Archon", CategoryID: 6, LocationID: station, Flag: "Hangar", Quantity: 1, Singleton: true, Volume: 13950000, PackagedVolume: 1300000},
		{ItemID: 2, Name: "Rifter", CategoryID: 6, LocationID: 1, Flag: "ShipHangar", Quantity: 1, Singleton: true, Volume: 27289, PackagedVolume: 2500},
		{ItemID: 3, Name: "Tritanium", LocationID: 2, Flag: "Cargo", Quantity: 100, Volume: 0.01},
	}
	_, totals := cargo.Inventory(items, station)
	if totals.AssembledShips != 1 || !near(totals.ShipContentsM3, 27289) || !near(totals.ShipsAssembledM3, 13950000) {
		t.Errorf("totals = %+v, want only the carrier counted, with the Rifter as its assembled contents", totals)
	}
}
