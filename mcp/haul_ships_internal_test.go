package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func stubHaulShipAssets(p *planner) {
	stubAssets(p, func(characterID int32) ([]AssetItem, error) {
		if characterID != reconPilot {
			return nil, errors.New("missing scope")
		}
		return []AssetItem{
			{ItemID: 1, TypeID: 23757, LocationID: 60003760, LocationType: "station", LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},       // Archon
			{ItemID: 2, TypeID: 648, LocationID: 1, LocationType: "item", LocationFlag: "ShipHangar", Quantity: 1, IsSingleton: true},               // Badger in its bay
			{ItemID: 3, TypeID: 32880, LocationID: 60003760, LocationType: "station", LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},       // cyno Venture
			{ItemID: 4, TypeID: 52694, LocationID: 3, LocationType: "item", LocationFlag: "HiSlot0", Quantity: 1, IsSingleton: true},                // its generator
			{ItemID: 5, TypeID: 32880, LocationID: 60003760, LocationType: "station", LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},       // plain Venture
			{ItemID: 6, TypeID: 28352, LocationID: 30000142, LocationType: "solar_system", LocationFlag: "AutoFit", Quantity: 1, IsSingleton: true}, // Rorqual in space
			{ItemID: 7, TypeID: 22430, LocationID: 1049345291823, LocationType: "item", LocationFlag: "Hangar", Quantity: 1, IsSingleton: true},     // Sin in a structure
			{ItemID: 8, TypeID: 648, LocationID: 60003760, LocationType: "station", LocationFlag: "Hangar", Quantity: 1},                            // packaged Badger
		}, nil
	})
	stubAssetNames(p, func(int32, []int64) ([]AssetName, error) {
		return []AssetName{{ItemID: 1, Name: "Big Brick"}}, nil
	})
}

func TestListHaulShips_FindsHaulersCapitalsAndCynoShips(t *testing.T) {
	store, p := readinessEnv(t)
	if _, err := store.Update(func(c *config.Config) error {
		c.UpsertStructure(config.Structure{ID: 1049345291823, Name: "Jita Keepstar", Kind: config.StructureUpwell, System: &config.NamedID{ID: 30000142, Name: "Jita"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	stubHaulShipAssets(p)

	_, out, err := listHaulShipsHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, ListHaulShipsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Carrier Pilot") {
		t.Errorf("warnings = %v, want one for the character whose assets failed", out.Warnings)
	}
	got := map[int64]HaulShip{}
	for _, s := range out.Ships {
		got[s.ItemID] = s
	}
	if len(out.Ships) != 5 {
		t.Fatalf("ships = %+v, want the Archon, the Badger in its bay, the cyno Venture, the Rorqual and the Sin", out.Ships)
	}
	if _, ok := got[5]; ok {
		t.Error("a Venture with no cyno generator isn't a haul ship")
	}
	if a := got[1]; a.Class != "Carrier" || a.Name != "Big Brick" || a.OwnerID != reconPilot || a.Location != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" || a.System != "Jita" || a.SecurityDisplay != "0.9" {
		t.Errorf("Archon = %+v", a)
	}
	if b := got[2]; b.InsideItemID != 1 || b.Inside != "Archon" || b.LocationID != 60003760 || b.Class != "Hauler" {
		t.Errorf("Badger in the Archon = %+v", b)
	}
	if v := got[3]; v.Class != cynoClass || v.CynoGenerator != "Industrial Cynosural Field Generator" {
		t.Errorf("cyno Venture = %+v", v)
	}
	if r := got[6]; r.Location != "Jita (in space)" || r.System != "Jita" {
		t.Errorf("Rorqual in space = %+v", r)
	}
	if s := got[7]; s.Location != "Jita Keepstar" || s.System != "Jita" || s.Class != "Black Ops" {
		t.Errorf("Sin in a saved structure = %+v", s)
	}
}

func TestListHaulShips_FiltersByClass(t *testing.T) {
	store, p := readinessEnv(t)
	stubHaulShipAssets(p)
	h := listHaulShipsHandler("test", p.openESI, p.openPublicESI, store, p.sde)
	_, out, err := h(context.Background(), nil, ListHaulShipsInput{CharacterIDs: []int32{reconPilot}, Classes: []string{"carrier", "Cyno ship"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Ships) != 2 || out.Ships[0].Type != "Archon" || out.Ships[1].Class != cynoClass {
		t.Errorf("ships = %+v, want the Archon then the cyno Venture", out.Ships)
	}
	if _, _, err := h(context.Background(), nil, ListHaulShipsInput{Classes: []string{"Battleship"}}); err == nil || !strings.Contains(err.Error(), "Carrier") {
		t.Errorf("unknown class: err = %v, want one listing the classes", err)
	}
}

// A character whose client can't be opened fails like any other asset
// read: here the only one, so the call fails naming the open error.
func TestListHaulShips_UnopenableClient(t *testing.T) {
	store, p := readinessEnv(t)
	p.openESI = failOpenESI(errors.New("no refresh token"))
	_, _, err := listHaulShipsHandler("test", p.openESI, p.openPublicESI, store, p.sde)(context.Background(), nil, ListHaulShipsInput{CharacterIDs: []int32{reconPilot}})
	if err == nil || !strings.Contains(err.Error(), "couldn't read Recon Pilot's assets: no refresh token") {
		t.Errorf("err = %v, want the open error", err)
	}
}

func TestRootOf_CorporationOfficeIsTransparent(t *testing.T) {
	office := AssetItem{ItemID: 10, TypeID: 27, LocationID: 60003760, LocationType: "station", LocationFlag: "OfficeFolder", IsSingleton: true}
	ship := AssetItem{ItemID: 11, TypeID: 28352, LocationID: 10, LocationType: "item", LocationFlag: "CorpSAG1", IsSingleton: true}
	byID := map[int64]AssetItem{office.ItemID: office, ship.ItemID: ship}
	loc, kind, holder := rootOf(byID, ship)
	if loc != 60003760 || kind != "station" || holder != 0 {
		t.Errorf("rootOf = %d, %q, %d; want the station with no holder", loc, kind, holder)
	}
}
