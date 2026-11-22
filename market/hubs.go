package market

import (
	"slices"
	"sort"
)

// DefaultBasket is the everyday goods a trade hub is scored on: the seven
// minerals, the four isotopes and fuel blocks, Liquid Ozone, Strontium
// Clathrates, Nanite Repair Paste, PLEX, a Large Skill Injector, a Venture,
// and three common T2 modules. Type IDs checked against the SDE 2026-11-20.
var DefaultBasket = []int64{
	34, 35, 36, 37, 38, 39, 40, // Tritanium … Megacyte
	16274, 17889, 17888, 17887, // Helium, Hydrogen, Nitrogen, Oxygen Isotopes
	4247, 4246, 4051, 4312, // Helium, Hydrogen, Nitrogen, Oxygen Fuel Blocks
	16273, 16275, 28668, // Liquid Ozone, Strontium Clathrates, Nanite Repair Paste
	44992, 40520, 32880, // PLEX, Large Skill Injector, Venture
	2048, 3244, 12058, // Damage Control II, Warp Disruptor II, 10MN Afterburner II
}

// BookOrder is one market order, buy or sell.
type BookOrder struct {
	TypeID     int64
	Price      float64
	Volume     int64
	LocationID int64
	IsBuy      bool
}

// BasketScore is how well a hub stocks a basket.
type BasketScore struct {
	Stocked    int
	Of         int
	SellOrders int
	BuyOrders  int
	Missing    []int64 // basket items with no sell order at the hub
}

// ScoreBasket counts, among orders at locationID, the basket items with at
// least one sell order, and the sell and buy orders for basket items.
func ScoreBasket(orders []BookOrder, locationID int64, basket []int64) BasketScore {
	in := map[int64]bool{}
	for _, t := range basket {
		in[t] = true
	}
	stocked := map[int64]bool{}
	s := BasketScore{Of: len(in)}
	for _, o := range orders {
		if o.LocationID != locationID || !in[o.TypeID] || o.Volume <= 0 {
			continue
		}
		if o.IsBuy {
			s.BuyOrders++
			continue
		}
		s.SellOrders++
		stocked[o.TypeID] = true
	}
	s.Stocked = len(stocked)
	seen := map[int64]bool{}
	for _, t := range basket {
		if !stocked[t] && !seen[t] {
			s.Missing = append(s.Missing, t)
		}
		seen[t] = true
	}
	return s
}

// PublicContract is the part of a public contract hub scoring needs.
type PublicContract struct {
	ID              int64
	Type            string // item_exchange, auction or courier
	Price           float64
	StartLocationID int64
}

// Contract value bands, in ISK.
const (
	contractBand1B  = 1e9
	contractBand10B = 1e10
)

// ContractBands counts the public item exchange and auction contracts that
// start at a location, by value.
type ContractBands struct {
	Public   int
	Over1B   int
	Over10B  int
	Priciest []int64 // the most expensive contracts there, priciest first
}

// BandContracts counts contracts starting at locationID (couriers aside) and
// lists the top priciest.
func BandContracts(contracts []PublicContract, locationID int64, top int) ContractBands {
	var here []PublicContract
	for _, c := range contracts {
		if c.StartLocationID == locationID && c.Type != "courier" {
			here = append(here, c)
		}
	}
	sort.SliceStable(here, func(i, j int) bool { return here[i].Price > here[j].Price })
	b := ContractBands{Public: len(here)}
	for i, c := range here {
		if c.Price >= contractBand1B {
			b.Over1B++
		}
		if c.Price >= contractBand10B {
			b.Over10B++
		}
		if i < top {
			b.Priciest = append(b.Priciest, c.ID)
		}
	}
	return b
}

// HubQuote is one hub's walked order book for a shopping list.
type HubQuote struct {
	Hub    string
	ID     int64 // the hub's station or structure ID, carried to its HubRank
	Quotes []Quote
	// Jumps is the gate jumps from the reference system, or -1 when unknown.
	Jumps int
}

// HubRank is a hub's place for a shopping list.
type HubRank struct {
	Hub       string  `json:"hub"`
	ID        int64   `json:"-"`
	Covered   int     `json:"covered" jsonschema:"items the hub can supply in full"`
	Of        int     `json:"of"`
	Total     float64 `json:"total" jsonschema:"ISK for everything the hub can supply, walking its order book"`
	Shortfall []int64 `json:"shortfallTypeIds,omitempty" jsonschema:"items it can't supply in full"`
	Jumps     int     `json:"jumps" jsonschema:"gate jumps from the reference system; -1 when unknown"`
}

// RankHubs orders hubs for a shopping list: the most items supplied in
// full first, then the lowest total among hubs covering the same items,
// then the fewest jumps.
func RankHubs(quotes []HubQuote) []HubRank {
	ranks := make([]HubRank, 0, len(quotes))
	for _, hq := range quotes {
		r := HubRank{Hub: hq.Hub, ID: hq.ID, Of: len(hq.Quotes), Jumps: hq.Jumps}
		for _, q := range hq.Quotes {
			r.Total += q.Total
			if q.Shortfall == 0 && q.Filled > 0 {
				r.Covered++
			} else {
				r.Shortfall = append(r.Shortfall, q.TypeID)
			}
		}
		ranks = append(ranks, r)
	}
	sort.SliceStable(ranks, func(i, j int) bool {
		a, b := ranks[i], ranks[j]
		if a.Covered != b.Covered {
			return a.Covered > b.Covered
		}
		if a.Covered == a.Of && b.Covered == b.Of && a.Total != b.Total {
			return a.Total < b.Total
		}
		if a.Jumps != b.Jumps {
			return jumpsKey(a.Jumps) < jumpsKey(b.Jumps)
		}
		return slices.Compare([]byte(a.Hub), []byte(b.Hub)) < 0
	})
	return ranks
}

// jumpsKey sorts unknown distances last.
func jumpsKey(j int) int {
	if j < 0 {
		return int(^uint(0) >> 1)
	}
	return j
}
