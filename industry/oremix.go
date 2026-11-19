package industry

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

// OreOption is a compressed ore the solver may buy.
type OreOption struct {
	TypeID     int64
	Name       string
	Portion    int64
	M3         float64   // per unit
	PerPortion []TypeQty // reprocessing output per portion at 100%
	Yield      float64   // this character's yield for this ore (0–1)
	Book       []Order   // sell orders at the hub
}

// MineralOption is a refined mineral the solver may buy directly.
type MineralOption struct {
	TypeID int64
	Name   string
	M3     float64
	Book   []Order
}

// OreMixRequest is the input to SolveOreMix.
type OreMixRequest struct {
	Needs        map[int64]int64 // mineral type → units still needed
	Ores         []OreOption
	Minerals     []MineralOption
	FreightPerM3 float64
	StationID    int64 // hub station to filter books by; 0 = any
	Tiers        int   // price tiers per book; 0 = 6
}

// OrePurchase is one ore in the solution.
type OrePurchase struct {
	TypeID   int64     `json:"typeId"`
	Name     string    `json:"name"`
	Quantity int64     `json:"quantity"`
	Cost     float64   `json:"cost"`
	M3       float64   `json:"m3"`
	Yields   []TypeQty `json:"yields"`
}

// MineralPurchase is a mineral bought refined.
type MineralPurchase struct {
	TypeID   int64   `json:"typeId"`
	Name     string  `json:"name"`
	Quantity int64   `json:"quantity"`
	Cost     float64 `json:"cost"`
	M3       float64 `json:"m3"`
}

// OreMix is the solver's answer, priced against the real books.
type OreMix struct {
	Ores        []OrePurchase     `json:"ores"`
	Minerals    []MineralPurchase `json:"minerals"`
	FromOre     []TypeQty         `json:"fromOre"`     // needed minerals the ore covers
	Surplus     []TypeQty         `json:"surplus"`     // needed minerals produced beyond need
	Byproducts  []TypeQty         `json:"byproducts"`  // outputs not needed at all
	TotalCost   float64           `json:"totalCost"`   // ore + minerals
	TotalM3     float64           `json:"totalM3"`     // ore + minerals hauled
	FreightCost float64           `json:"freightCost"` // TotalM3 × FreightPerM3
	RefinedM3   float64           `json:"refinedM3"`   // volume of needed minerals that come out of the ore
}

type segment struct {
	opt   int // index into options
	price float64
	cap   float64
}

// SolveOreMix finds the cheapest mix of compressed ore and refined minerals,
// counting freight, that covers every need.
func SolveOreMix(req OreMixRequest) (OreMix, error) {
	tiers := req.Tiers
	if tiers <= 0 {
		tiers = 6
	}
	needIDs := make([]int64, 0, len(req.Needs))
	for t, q := range req.Needs {
		if q > 0 {
			needIDs = append(needIDs, t)
		}
	}
	slices.Sort(needIDs)
	if len(needIDs) == 0 {
		return OreMix{}, nil
	}
	row := map[int64]int{}
	for i, t := range needIDs {
		row[t] = i
	}

	// Each option: its per-unit yield of needed minerals and its volume.
	type option struct {
		ore     *OreOption
		mineral *MineralOption
		yield   map[int64]float64
		m3      float64
		book    []Order
	}
	var opts []option
	for i := range req.Ores {
		o := &req.Ores[i]
		y := map[int64]float64{}
		for _, m := range o.PerPortion {
			if _, ok := row[m.TypeID]; ok {
				y[m.TypeID] = float64(m.Quantity) * o.Yield / float64(max(o.Portion, 1))
			}
		}
		if len(y) > 0 {
			opts = append(opts, option{ore: o, yield: y, m3: o.M3, book: o.Book})
		}
	}
	for i := range req.Minerals {
		m := &req.Minerals[i]
		if _, ok := row[m.TypeID]; ok {
			opts = append(opts, option{mineral: m, yield: map[int64]float64{m.TypeID: 1}, m3: m.M3, book: m.Book})
		}
	}

	// Feasibility: can the books cover each need at all?
	supply := map[int64]float64{}
	var segs []segment
	for oi, o := range opts {
		book := stationBook(o.book, req.StationID)
		useful := 0.0
		for t, y := range o.yield {
			useful = math.Max(useful, float64(req.Needs[t])/y)
		}
		useful = useful*1.05 + 1
		var taken float64
		var tierOrders [][]Order
		for _, ord := range book {
			if taken >= useful {
				break
			}
			tierOrders = append(tierOrders, []Order{ord})
			taken += float64(ord.Volume)
		}
		for _, grp := range mergeTiers(tierOrders, tiers) {
			var q, cost float64
			for _, ord := range grp {
				q += float64(ord.Volume)
				cost += float64(ord.Volume) * ord.Price
			}
			if q <= 0 {
				continue
			}
			segs = append(segs, segment{opt: oi, price: cost/q + o.m3*req.FreightPerM3, cap: q})
			for t, y := range o.yield {
				supply[t] += q * y
			}
		}
	}
	var short []string
	for _, t := range needIDs {
		if supply[t] < float64(req.Needs[t]) {
			short = append(short, fmt.Sprintf("type %d (need %d, books supply %.0f)", t, req.Needs[t], supply[t]))
		}
	}
	if len(short) > 0 {
		return OreMix{}, fmt.Errorf("solve ore mix: the market can't cover %s", strings.Join(short, ", "))
	}

	// LP in standard form over y_j = x_j / cap_j ∈ [0, 1]:
	//   minimise Σ price_j cap_j y_j
	//   Σ_j (yield_jt cap_j / need_t) y_j − s_t = 1   for each needed mineral t
	//   y_j + u_j = 1                                for each segment j
	n, m := len(segs), len(needIDs)
	cols := 2*n + m
	A := mat.NewDense(m+n, cols, nil)
	b := make([]float64, m+n)
	c := make([]float64, cols)
	for j, sg := range segs {
		c[j] = sg.price * sg.cap / 1e6
		for t, y := range opts[sg.opt].yield {
			A.Set(row[t], j, y*sg.cap/float64(req.Needs[t]))
		}
		A.Set(m+j, j, 1)
		A.Set(m+j, n+m+j, 1)
		b[m+j] = 1
	}
	for i := range needIDs {
		A.Set(i, n+i, -1)
		b[i] = 1
	}
	_, x, err := lp.Simplex(c, A, b, 1e-10, nil)
	if err != nil {
		return OreMix{}, fmt.Errorf("solve ore mix: %w", err)
	}

	// Sum chosen quantity per option, round ores up to whole portions.
	chosen := make([]float64, len(opts))
	for j, sg := range segs {
		chosen[sg.opt] += x[j] * sg.cap
	}
	var mix OreMix
	got := map[int64]int64{}
	for oi, o := range opts {
		if o.ore == nil || chosen[oi] < 1e-6 {
			continue
		}
		p := max(o.ore.Portion, 1)
		qty := int64(math.Ceil(chosen[oi]/float64(p)-1e-9)) * p
		if qty <= 0 {
			continue
		}
		q := WalkBook(o.ore.TypeID, o.ore.Book, qty, req.StationID)
		out := Reprocess(o.ore.PerPortion, p, qty, o.ore.Yield)
		for _, y := range out {
			got[y.TypeID] += y.Quantity
		}
		mix.Ores = append(mix.Ores, OrePurchase{TypeID: o.ore.TypeID, Name: o.ore.Name, Quantity: qty, Cost: q.Total, M3: float64(qty) * o.ore.M3, Yields: out})
	}
	for _, mo := range req.Minerals {
		need := req.Needs[mo.TypeID] - got[mo.TypeID]
		if need <= 0 {
			continue
		}
		q := WalkBook(mo.TypeID, mo.Book, need, req.StationID)
		if q.Shortfall > 0 {
			return OreMix{}, fmt.Errorf("solve ore mix: after rounding, the market is %d short of %s", q.Shortfall, mo.Name)
		}
		mix.Minerals = append(mix.Minerals, MineralPurchase{TypeID: mo.TypeID, Name: mo.Name, Quantity: need, Cost: q.Total, M3: float64(need) * mo.M3})
	}
	for t, q := range got {
		need, ok := req.Needs[t]
		switch {
		case !ok:
			mix.Byproducts = append(mix.Byproducts, TypeQty{t, q})
		case q > need:
			mix.FromOre = append(mix.FromOre, TypeQty{t, need})
			mix.Surplus = append(mix.Surplus, TypeQty{t, q - need})
		default:
			mix.FromOre = append(mix.FromOre, TypeQty{t, q})
		}
	}
	for _, list := range [][]TypeQty{mix.FromOre, mix.Surplus, mix.Byproducts} {
		sort.Slice(list, func(i, j int) bool { return list[i].TypeID < list[j].TypeID })
	}
	sort.Slice(mix.Ores, func(i, j int) bool { return mix.Ores[i].Cost > mix.Ores[j].Cost })
	for _, o := range mix.Ores {
		mix.TotalCost += o.Cost
		mix.TotalM3 += o.M3
	}
	for _, mp := range mix.Minerals {
		mix.TotalCost += mp.Cost
		mix.TotalM3 += mp.M3
	}
	for _, f := range mix.FromOre {
		mix.RefinedM3 += float64(f.Quantity) * mineralM3(req.Minerals, f.TypeID)
	}
	mix.FreightCost = mix.TotalM3 * req.FreightPerM3
	return mix, nil
}

func mineralM3(ms []MineralOption, t int64) float64 {
	for _, m := range ms {
		if m.TypeID == t {
			return m.M3
		}
	}
	return 0.01
}

func stationBook(orders []Order, stationID int64) []Order {
	var b []Order
	for _, o := range orders {
		if o.Volume > 0 && (stationID == 0 || o.LocationID == stationID) {
			b = append(b, o)
		}
	}
	sort.SliceStable(b, func(i, j int) bool { return b[i].Price < b[j].Price })
	return b
}

// mergeTiers groups consecutive orders (already sorted by price) into at most
// k tiers of roughly equal quantity.
func mergeTiers(orders [][]Order, k int) [][]Order {
	if len(orders) <= k {
		out := make([][]Order, len(orders))
		copy(out, orders)
		return out
	}
	var total float64
	for _, g := range orders {
		total += float64(g[0].Volume)
	}
	per := total / float64(k)
	var out [][]Order
	var cur []Order
	var acc float64
	for _, g := range orders {
		cur = append(cur, g...)
		acc += float64(g[0].Volume)
		if acc >= per && len(out) < k-1 {
			out = append(out, cur)
			cur, acc = nil, 0
		}
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}
