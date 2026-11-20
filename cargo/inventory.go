package cargo

import (
	"sort"
	"strings"
)

// Item is one asset as ESI lists it, with its type's details from the SDE.
type Item struct {
	ItemID     int64
	TypeID     int64
	Name       string
	GroupID    int64
	CategoryID int64
	LocationID int64
	Flag       string
	Quantity   int64
	Singleton  bool
	// Volume is the type's assembled volume, per unit.
	Volume float64
	// PackagedVolume is the type's packaged volume per unit; 0 when the SDE
	// has none, which means it's the same as Volume.
	PackagedVolume float64
	// Owner names the character or corporation the item belongs to.
	Owner string
}

// UnitVolume is the volume one unit of the item takes up as it is now:
// assembled for an assembled (singleton) item, packaged otherwise.
func (it Item) UnitVolume() float64 {
	if !it.Singleton && it.PackagedVolume > 0 {
		return it.PackagedVolume
	}
	return it.Volume
}

// StackVolume is the volume of the whole stack as it is now.
func (it Item) StackVolume() float64 { return it.UnitVolume() * float64(max(it.Quantity, 1)) }

// IsAssembledShip reports whether the item is a ship that is assembled.
func (it Item) IsAssembledShip() bool { return it.Singleton && it.CategoryID == ShipCategoryID }

// packagedUnitVolume is the item's volume once packaged.
func (it Item) packagedUnitVolume() float64 {
	if it.PackagedVolume > 0 {
		return it.PackagedVolume
	}
	return it.Volume
}

func isFitted(flag string) bool {
	for _, p := range fittingFlagPrefixes {
		if strings.HasPrefix(flag, p) {
			return true
		}
	}
	return false
}

// AssembledShip is what an assembled ship takes to move.
type AssembledShip struct {
	AssembledM3 float64 `json:"assembledM3" jsonschema:"the ship's volume assembled, which is what it takes in a ship maintenance bay"`
	PackagedM3  float64 `json:"packagedM3" jsonschema:"the ship's volume once repackaged, for a cargo hold"`
	// ContentsM3 is what comes out of the ship when it's repackaged: every
	// fitted module except destroyed rigs, and everything in its holds.
	ContentsM3 float64  `json:"contentsM3" jsonschema:"volume of everything that comes out of the ship to repackage it: fitted modules (except rigs that are destroyed) and the contents of its holds"`
	RigsLost   []string `json:"rigsLost,omitempty" jsonschema:"rigs that repackaging this ship destroys; strategic cruisers keep theirs"`
}

// InventoryLine is one item at a location, listed in tree order: each item
// is followed by everything inside it.
type InventoryLine struct {
	ItemID       int64   `json:"itemId"`
	ParentItemID int64   `json:"parentItemId,omitempty" jsonschema:"the container or ship this item is in; absent for items directly in a hangar"`
	Depth        int     `json:"depth" jsonschema:"0 for items in a hangar, 1 for items inside them, and so on"`
	TypeID       int64   `json:"typeId"`
	TypeName     string  `json:"typeName,omitempty"`
	Owner        string  `json:"owner"`
	Flag         string  `json:"flag"`
	Quantity     int64   `json:"quantity"`
	Assembled    bool    `json:"assembled" jsonschema:"assembled (true) or packaged (false), from ESI's isSingleton"`
	VolumeM3     float64 `json:"volumeM3" jsonschema:"the whole stack's volume as it is now"`
	ContentsM3   float64 `json:"contentsM3,omitempty" jsonschema:"for a container or ship, the volume of what's in its holds, not counting fitted modules"`
	// Ship is set for an assembled ship.
	Ship *AssembledShip `json:"assembledShip,omitempty"`
}

// InventoryTotals are the volumes a move of everything at the location
// involves. Ships and what's in them are counted apart from other items.
type InventoryTotals struct {
	ItemsM3           float64 `json:"itemsM3" jsonschema:"everything in hangars except assembled ships, at the volume it takes now; containers count at their own volume with their contents inside"`
	AssembledShips    int     `json:"assembledShips"`
	ShipsAssembledM3  float64 `json:"shipsAssembledM3" jsonschema:"assembled ships' volume, for carrying them assembled in a ship maintenance bay"`
	ShipsPackagedM3   float64 `json:"shipsPackagedM3" jsonschema:"assembled ships' volume once repackaged"`
	ShipContentsM3    float64 `json:"shipContentsM3" jsonschema:"what comes out of the assembled ships to repackage them: fitted modules except destroyed rigs, and their holds' contents"`
	AllRepackagedM3   float64 `json:"allRepackagedM3" jsonschema:"itemsM3 + shipsPackagedM3 + shipContentsM3: the cargo-hold volume of the whole location with every ship repackaged"`
	ShipsWithRigsLost int     `json:"shipsWithRigsLost" jsonschema:"how many assembled ships would lose rigs to repackaging"`
}

// Inventory lays out everything at one station or structure: the items
// whose chain of containers leads to locationID, with corporation offices
// treated as part of the location rather than as a container.
func Inventory(items []Item, locationID int64) ([]InventoryLine, InventoryTotals) {
	children := map[int64][]Item{}
	for _, it := range items {
		children[it.LocationID] = append(children[it.LocationID], it)
	}
	for k := range children {
		c := children[k]
		sort.SliceStable(c, func(i, j int) bool {
			if c[i].Flag != c[j].Flag {
				return c[i].Flag < c[j].Flag
			}
			if c[i].Name != c[j].Name {
				return c[i].Name < c[j].Name
			}
			return c[i].ItemID < c[j].ItemID
		})
	}

	// Offices are transparent: their contents are top-level hangar items.
	var top []Item
	for _, it := range children[locationID] {
		if it.Flag == officeFlag {
			top = append(top, children[it.ItemID]...)
			continue
		}
		top = append(top, it)
	}

	var lines []InventoryLine
	var totals InventoryTotals
	// walk lists an item and everything in it. counted says whether this
	// item's own volume is part of what has to be hauled: true for hangar
	// items and for what comes out of a repackaged ship, false inside a
	// container, whose own volume already covers what it holds.
	var walk func(it Item, parent int64, depth int, inShip bool, counted bool) float64
	walk = func(it Item, parent int64, depth int, inShip bool, counted bool) float64 {
		line := InventoryLine{
			ItemID: it.ItemID, ParentItemID: parent, Depth: depth, TypeID: it.TypeID, TypeName: it.Name,
			Owner: it.Owner, Flag: it.Flag, Quantity: max(it.Quantity, 1), Assembled: it.Singleton, VolumeM3: it.StackVolume(),
		}
		idx := len(lines)
		lines = append(lines, line)

		if it.IsAssembledShip() {
			ship := &AssembledShip{AssembledM3: it.Volume, PackagedM3: it.packagedUnitVolume()}
			keepsRigs := it.GroupID == StrategicCruiserGroupID
			holds := 0.0
			for _, c := range children[it.ItemID] {
				rig := strings.HasPrefix(c.Flag, "RigSlot")
				if rig && !keepsRigs {
					ship.RigsLost = append(ship.RigsLost, c.Name)
					walk(c, it.ItemID, depth+1, true, false)
					continue
				}
				if rig {
					walk(c, it.ItemID, depth+1, true, false) // stays fitted
					continue
				}
				v := walk(c, it.ItemID, depth+1, true, true)
				ship.ContentsM3 += v
				if !isFitted(c.Flag) {
					holds += c.StackVolume()
				}
			}
			lines[idx].ContentsM3 = holds
			lines[idx].Ship = ship
			// A ship inside another ship (a ship maintenance bay) is part of
			// that ship's contents: it comes out assembled, with its own
			// contents riding inside it, so only hangar ships count here.
			if !inShip {
				totals.AssembledShips++
				totals.ShipsAssembledM3 += ship.AssembledM3
				totals.ShipsPackagedM3 += ship.PackagedM3
				totals.ShipContentsM3 += ship.ContentsM3
				if len(ship.RigsLost) > 0 {
					totals.ShipsWithRigsLost++
				}
			}
			return ship.AssembledM3
		}

		contents := 0.0
		for _, c := range children[it.ItemID] {
			walk(c, it.ItemID, depth+1, inShip, false)
			contents += c.StackVolume()
		}
		lines[idx].ContentsM3 = contents
		if counted && !inShip {
			totals.ItemsM3 += line.VolumeM3
		}
		return line.VolumeM3
	}
	for _, it := range top {
		walk(it, 0, 0, false, true)
	}
	totals.AllRepackagedM3 = totals.ItemsM3 + totals.ShipsPackagedM3 + totals.ShipContentsM3
	return lines, totals
}
