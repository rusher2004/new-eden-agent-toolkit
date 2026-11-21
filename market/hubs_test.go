package market_test

import (
	"slices"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/market"
)

func TestDefaultBasket_Has24DistinctItems(t *testing.T) {
	b := slices.Clone(market.DefaultBasket)
	slices.Sort(b)
	if len(slices.Compact(b)) != 24 {
		t.Errorf("basket has %d distinct items, want 24", len(slices.Compact(b)))
	}
}

func TestScoreBasket(t *testing.T) {
	const jita = 60003760
	orders := []market.BookOrder{
		{TypeID: 34, Volume: 100, LocationID: jita},
		{TypeID: 34, Volume: 5, LocationID: jita},
		{TypeID: 34, Volume: 9, LocationID: jita, IsBuy: true},
		{TypeID: 35, Volume: 7, LocationID: 1},       // elsewhere in the region
		{TypeID: 36, Volume: 0, LocationID: jita},    // empty order
		{TypeID: 99999, Volume: 1, LocationID: jita}, // not in the basket
		{TypeID: 37, Volume: 1, LocationID: jita, IsBuy: true},
	}
	got := market.ScoreBasket(orders, jita, []int64{34, 35, 36, 37})
	if got.Stocked != 1 || got.Of != 4 || got.SellOrders != 2 || got.BuyOrders != 2 || !slices.Equal(got.Missing, []int64{35, 36, 37}) {
		t.Errorf("score = %+v", got)
	}
}

func TestBandContracts(t *testing.T) {
	const amamake = 1022167642188
	cs := []market.PublicContract{
		{ID: 1, Type: "item_exchange", Price: 45e9, StartLocationID: amamake},
		{ID: 2, Type: "auction", Price: 2e9, StartLocationID: amamake},
		{ID: 3, Type: "item_exchange", Price: 5e6, StartLocationID: amamake},
		{ID: 4, Type: "courier", Price: 90e9, StartLocationID: amamake},
		{ID: 5, Type: "item_exchange", Price: 80e9, StartLocationID: 60004588},
		{ID: 6, Type: "item_exchange", Price: 12e9, StartLocationID: amamake},
	}
	got := market.BandContracts(cs, amamake, 2)
	if got.Public != 4 || got.Over1B != 3 || got.Over10B != 2 || !slices.Equal(got.Priciest, []int64{1, 6}) {
		t.Errorf("bands = %+v", got)
	}
}

func TestRankHubs(t *testing.T) {
	full := func(total float64) market.Quote { return market.Quote{Filled: 10, Total: total} }
	short := market.Quote{TypeID: 2, Filled: 3, Shortfall: 7, Total: 1}
	got := market.RankHubs([]market.HubQuote{
		{Hub: "Partial but close", Quotes: []market.Quote{full(100), short}, Jumps: 1},
		{Hub: "Full but pricey", Quotes: []market.Quote{full(100), full(300)}, Jumps: 9},
		{Hub: "Full and cheap", Quotes: []market.Quote{full(100), full(140)}, Jumps: -1},
		{Hub: "Full, same price, nearer", Quotes: []market.Quote{full(150), full(150)}, Jumps: 2},
	})
	var order []string
	for _, r := range got {
		order = append(order, r.Hub)
	}
	want := []string{"Full and cheap", "Full, same price, nearer", "Full but pricey", "Partial but close"}
	if !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
	if last := got[3]; last.Covered != 1 || last.Of != 2 || !slices.Equal(last.Shortfall, []int64{2}) {
		t.Errorf("partial hub = %+v", last)
	}
}
