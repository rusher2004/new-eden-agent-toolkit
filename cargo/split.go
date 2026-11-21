package cargo

import (
	"fmt"
	"math"
	"sort"
)

// SplitLoad divides one load among several holds flying together, so ships
// that share a trip each carry a fair part. Each hull gets a share of every
// item in proportion to its hold, keeping portions whole; anything that
// doesn't fit that way goes, largest items first, to whichever hold has the
// most room left.
func SplitLoad(load Load, holds []float64, portions map[int64]int64) ([]Load, error) {
	total := 0.0
	for _, h := range holds {
		if h <= 0 {
			return nil, fmt.Errorf("split load: every hold must be positive")
		}
		total += h
	}
	if load.M3 > total+1e-6 {
		return nil, fmt.Errorf("split load: %.0f m³ doesn't fit %.0f m³ of holds", load.M3, total)
	}
	out := make([]Load, len(holds))
	for i := range out {
		out[i] = Load{Number: load.Number, ByDest: map[string]float64{}}
	}
	add := func(i int, it LoadItem, qty int64, unit float64) {
		if qty <= 0 {
			return
		}
		v := float64(qty) * unit
		out[i].Items = append(out[i].Items, LoadItem{TypeID: it.TypeID, Name: it.Name, Quantity: qty, M3: v, Dest: it.Dest})
		out[i].M3 += v
		out[i].ByDest[it.Dest] += v
	}
	type rest struct {
		it   LoadItem
		qty  int64
		unit float64
	}
	var left []rest
	for _, it := range load.Items {
		if it.Quantity <= 0 {
			continue
		}
		unit := it.M3 / float64(it.Quantity)
		p := max(portions[it.TypeID], 1)
		units := it.Quantity / p
		given := int64(0)
		cum := 0.0
		for i, h := range holds {
			from := int64(math.Round(float64(units) * cum / total))
			cum += h
			to := int64(math.Round(float64(units) * cum / total))
			q := (to - from) * p
			if out[i].M3+float64(q)*unit > holds[i]+1e-6 {
				q = max(int64(math.Floor((holds[i]-out[i].M3)/(unit*float64(p))+1e-9)), 0) * p
			}
			add(i, it, q, unit)
			given += q
		}
		if r := it.Quantity - given; r > 0 {
			left = append(left, rest{it, r, unit})
		}
	}
	sort.SliceStable(left, func(i, j int) bool { return left[i].unit > left[j].unit })
	for _, r := range left {
		p := max(portions[r.it.TypeID], 1)
		for r.qty > 0 {
			best := 0
			for i := range holds {
				if holds[i]-out[i].M3 > holds[best]-out[best].M3 {
					best = i
				}
			}
			room := holds[best] - out[best].M3
			chunk := max(p, 1)
			fit := int64(math.Floor(room/(r.unit*float64(chunk))+1e-9)) * chunk
			if r.qty < chunk {
				fit = r.qty
				if float64(fit)*r.unit > room+1e-6 {
					fit = 0
				}
			}
			if fit <= 0 {
				return nil, fmt.Errorf("split load: %s doesn't fit the remaining space", r.it.Name)
			}
			q := min(fit, r.qty)
			add(best, r.it, q, r.unit)
			r.qty -= q
		}
	}
	for i := range out {
		SortLoadItems(&out[i])
	}
	return out, nil
}
