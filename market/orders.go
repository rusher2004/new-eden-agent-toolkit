package market

import "sort"

// Order is one sell order.
type Order struct {
	Price      float64 `json:"price"`
	Volume     int64   `json:"volume"`
	LocationID int64   `json:"locationId"`
}

// Quote is the result of buying a quantity from an order book.
type Quote struct {
	TypeID    int64   `json:"typeId"`
	Quantity  int64   `json:"quantity"`
	Filled    int64   `json:"filled"`
	Total     float64 `json:"total"`
	Average   float64 `json:"average"`
	Lowest    float64 `json:"lowest"`
	Shortfall int64   `json:"shortfall"`
	Depth     int64   `json:"depth"` // units on the book at the station
}

// WalkBook buys qty from the cheapest sell orders at stationID (0 = any
// location). The average is over what filled; Shortfall is what couldn't.
func WalkBook(typeID int64, orders []Order, qty int64, stationID int64) Quote {
	var book []Order
	for _, o := range orders {
		if o.Volume > 0 && (stationID == 0 || o.LocationID == stationID) {
			book = append(book, o)
		}
	}
	sort.SliceStable(book, func(i, j int) bool { return book[i].Price < book[j].Price })
	q := Quote{TypeID: typeID, Quantity: qty}
	for _, o := range book {
		q.Depth += o.Volume
	}
	if len(book) > 0 {
		q.Lowest = book[0].Price
	}
	left := qty
	for _, o := range book {
		if left <= 0 {
			break
		}
		k := min(o.Volume, left)
		q.Total += float64(k) * o.Price
		q.Filled += k
		left -= k
	}
	q.Shortfall = left
	if q.Filled > 0 {
		q.Average = q.Total / float64(q.Filled)
	}
	return q
}
