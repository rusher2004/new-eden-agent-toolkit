package market_test

import (
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/market"
)

func TestWalkBook(t *testing.T) {
	orders := []market.Order{{Price: 207e6, Volume: 18, LocationID: 60003760}, {Price: 204.8e6, Volume: 2, LocationID: 60003760}, {Price: 1, Volume: 99, LocationID: 1}}
	q := market.WalkBook(57487, orders, 25, 60003760)
	if q.Filled != 20 || q.Shortfall != 5 || q.Lowest != 204.8e6 || q.Total != 2*204.8e6+18*207e6 || q.Depth != 20 {
		t.Errorf("quote = %+v", q)
	}
}
