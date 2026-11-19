package industry

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

// BuildBuyInput compares making an intermediate with buying it.
type BuildBuyInput struct {
	TypeID       int64
	Units        int64
	InputCost    float64 // walked price of every input for Units, excluding stock
	InputM3      float64 // volume of those inputs to haul
	JobFees      float64
	Buy          Quote
	UnitM3       float64 // volume of one finished unit
	FreightPerM3 float64
	CanBuild     bool   // owned copies, skills and facility allow it
	CantBuildWhy string // when CanBuild is false
}

// BuildBuy is the comparison for one intermediate.
type BuildBuy struct {
	TypeID          int64    `json:"typeId"`
	Units           int64    `json:"units"`
	BuildPerUnit    float64  `json:"buildPerUnit"`
	BuyPerUnit      float64  `json:"buyPerUnit"`
	BuildM3         float64  `json:"buildM3"`
	BuyM3           float64  `json:"buyM3"`
	SavedByBuilding float64  `json:"savedByBuilding"` // negative when buying is cheaper
	Recommend       string   `json:"recommend"`       // build or buy
	Flags           []string `json:"flags,omitempty"`
}

// CompareBuildVsBuy prices both ways and recommends the cheaper one that is
// possible.
func CompareBuildVsBuy(in BuildBuyInput) BuildBuy {
	out := BuildBuy{TypeID: in.TypeID, Units: in.Units, BuildM3: in.InputM3, BuyM3: in.UnitM3 * float64(in.Units)}
	if in.Units <= 0 {
		return out
	}
	build := in.InputCost + in.InputM3*in.FreightPerM3 + in.JobFees
	out.BuildPerUnit = build / float64(in.Units)
	if in.Buy.Filled > 0 {
		out.BuyPerUnit = in.Buy.Average + in.UnitM3*in.FreightPerM3
	}
	buyPossible := in.Buy.Shortfall == 0 && in.Buy.Filled > 0
	if !buyPossible {
		out.Flags = append(out.Flags, "market can't supply")
	}
	if !in.CanBuild {
		out.Flags = append(out.Flags, in.CantBuildWhy)
	}
	switch {
	case in.CanBuild && (!buyPossible || out.BuildPerUnit <= out.BuyPerUnit):
		out.Recommend = DecideBuild
	case buyPossible:
		out.Recommend = DecideBuy
	default:
		out.Recommend = DecideBuy
		out.Flags = append(out.Flags, "neither building nor buying fully covers it")
	}
	if out.BuyPerUnit > 0 {
		out.SavedByBuilding = (out.BuyPerUnit - out.BuildPerUnit) * float64(in.Units)
	}
	return out
}
