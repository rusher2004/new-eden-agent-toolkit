package industry

import (
	"fmt"
	"maps"
	"slices"
	"sort"
)

// TypeQty is a type and a quantity.
type TypeQty struct {
	TypeID   int64 `json:"typeId"`
	Quantity int64 `json:"quantity"`
}

// Recipe is how a type is made: one blueprint activity.
type Recipe struct {
	BlueprintTypeID int64
	Activity        string // ActManufacturing or ActReaction
	BaseSeconds     float64
	OutputPerRun    int64
	Materials       []TypeQty // per run, ME0
	Skills          []TypeQty // skill ID, level
}

// Copy is an owned blueprint the tree may use. Runs < 0 means an original
// (unlimited runs).
type Copy struct {
	ItemID int64
	ME     int
	TE     int
	Runs   int64
}

// Decision for a type in the tree.
const (
	DecideAuto  = ""
	DecideBuild = "build"
	DecideBuy   = "buy"
)

// TreeRequest is everything ExpandTree needs. Recipe and Multipliers may do
// I/O in the caller; ExpandTree itself does none.
type TreeRequest struct {
	ProductTypeID int64
	Quantity      int64 // units wanted, ignoring RootCopies when > 0
	// RootCopies, when set, are the exact copies to run for the product, one
	// job each, at their own runs (e.g. 15 single-run hull copies). Quantity
	// is then derived from them.
	RootCopies  []Copy
	Recipe      func(typeID int64) (Recipe, bool)
	Multipliers func(activity string, productTypeID int64) Multipliers
	Copies      map[int64][]Copy // owned blueprints by blueprint type ID
	Decisions   map[int64]string // per product type overrides
	Stock       map[int64]int64  // owned stock usable at the build site
	// BatchRuns caps runs per job when building from an original; 0 = one
	// job for everything.
	BatchRuns int64
	// CanBuild, when set, reports whether anyone available can run a recipe
	// (skills and slots). Auto decisions only choose build when it says yes.
	CanBuild func(Recipe) (bool, string)
}

// TreeJob is one job to install.
type TreeJob struct {
	ProductTypeID   int64     `json:"productTypeId"`
	BlueprintTypeID int64     `json:"blueprintTypeId"`
	Activity        string    `json:"activity"`
	Runs            int64     `json:"runs"`
	ME              int       `json:"me"`
	TE              int       `json:"te"`
	CopyItemID      int64     `json:"copyItemId,omitempty"` // 0 for an original or an unowned blueprint
	Output          int64     `json:"output"`
	Materials       []TypeQty `json:"materials"`
	// Seconds is base time × runs × TE × facility; builder skills apply later.
	Seconds float64   `json:"seconds"`
	Skills  []TypeQty `json:"skills,omitempty"`
}

// TreeNode is one type in the tree, aggregated over every parent.
type TreeNode struct {
	TypeID    int64   `json:"typeId"`
	Level     int     `json:"level"`
	Parents   []int64 `json:"parents,omitempty"`
	Demand    int64   `json:"demand"`
	FromStock int64   `json:"fromStock"`
	Built     int64   `json:"built"`   // units produced by jobs (may exceed need)
	Bought    int64   `json:"bought"`  // units to buy
	Surplus   int64   `json:"surplus"` // built beyond need
	Decision  string  `json:"decision"`
	Reason    string  `json:"reason"`
}

// Tree is the expanded plan.
type Tree struct {
	Nodes     []TreeNode `json:"nodes"`
	Jobs      []TreeJob  `json:"jobs"`
	Buy       []TypeQty  `json:"buy"`
	StockUsed []TypeQty  `json:"stockUsed"`
}

// ExpandTree expands a product through every buildable level, aggregating
// demand per type across parents before cutting jobs, netting stock at each
// level, and consuming owned copies largest-runs-first.
func ExpandTree(req TreeRequest) (Tree, error) {
	if req.Recipe == nil || req.Multipliers == nil {
		return Tree{}, fmt.Errorf("expand tree: Recipe and Multipliers are required")
	}
	stock := maps.Clone(req.Stock)
	if stock == nil {
		stock = map[int64]int64{}
	}
	copies := map[int64][]Copy{}
	for k, v := range req.Copies {
		c := slices.Clone(v)
		sort.SliceStable(c, func(i, j int) bool { return runsKey(c[i]) > runsKey(c[j]) })
		copies[k] = c
	}

	// Discover the graph and each type's deepest level, following only types
	// that will be built.
	level := map[int64]int{}
	parents := map[int64]map[int64]bool{}
	decision := map[int64]string{}
	reason := map[int64]string{}
	var visit func(t int64, lvl int, path map[int64]bool) error
	visit = func(t int64, lvl int, path map[int64]bool) error {
		if path[t] {
			return fmt.Errorf("expand tree: cycle at type %d", t)
		}
		if l, seen := level[t]; seen && l >= lvl {
			return nil
		}
		level[t] = lvl
		d, why := decide(req, t, lvl == 0, copies)
		decision[t], reason[t] = d, why
		if d != DecideBuild {
			return nil
		}
		r, _ := req.Recipe(t)
		path[t] = true
		defer delete(path, t)
		for _, m := range r.Materials {
			if parents[m.TypeID] == nil {
				parents[m.TypeID] = map[int64]bool{}
			}
			parents[m.TypeID][t] = true
			if err := visit(m.TypeID, lvl+1, path); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(req.ProductTypeID, 0, map[int64]bool{}); err != nil {
		return Tree{}, err
	}

	order := make([]int64, 0, len(level))
	for t := range level {
		order = append(order, t)
	}
	sort.Slice(order, func(i, j int) bool {
		if level[order[i]] != level[order[j]] {
			return level[order[i]] < level[order[j]]
		}
		return order[i] < order[j]
	})

	demand := map[int64]int64{}
	var tree Tree
	for _, t := range order {
		n := TreeNode{TypeID: t, Level: level[t], Decision: decision[t], Reason: reason[t]}
		for p := range parents[t] {
			n.Parents = append(n.Parents, p)
		}
		slices.Sort(n.Parents)

		if level[t] == 0 && len(req.RootCopies) > 0 {
			r, _ := req.Recipe(t)
			for _, c := range req.RootCopies {
				runs := c.Runs
				if runs <= 0 {
					runs = 1
				}
				job := makeJob(req, r, t, runs, c)
				tree.Jobs = append(tree.Jobs, job)
				n.Built += job.Output
				addMaterials(demand, job.Materials)
			}
			n.Demand = n.Built
			tree.Nodes = append(tree.Nodes, n)
			continue
		}

		need := demand[t]
		if level[t] == 0 {
			need = req.Quantity
		}
		n.Demand = need
		if level[t] > 0 {
			use := min(stock[t], need)
			if use > 0 {
				stock[t] -= use
				n.FromStock = use
				need -= use
				tree.StockUsed = append(tree.StockUsed, TypeQty{t, use})
			}
		}
		if need > 0 && n.Decision == DecideBuild {
			r, _ := req.Recipe(t)
			runsNeeded := ceilDiv(need, max(r.OutputPerRun, 1))
			cs := copies[r.BlueprintTypeID]
			for runsNeeded > 0 && len(cs) > 0 {
				c := cs[0]
				var runs int64
				if c.Runs < 0 {
					runs = runsNeeded
					if req.BatchRuns > 0 {
						runs = min(runs, req.BatchRuns)
					}
				} else {
					runs = min(c.Runs, runsNeeded)
					cs[0].Runs -= runs
					if cs[0].Runs == 0 {
						cs = cs[1:]
					}
				}
				job := makeJob(req, r, t, runs, c)
				tree.Jobs = append(tree.Jobs, job)
				n.Built += job.Output
				addMaterials(demand, job.Materials)
				runsNeeded -= runs
			}
			copies[r.BlueprintTypeID] = cs
			covered := min(n.Built, need)
			if covered < need {
				n.Reason = fmt.Sprintf("%s; owned blueprint runs cover %d of %d, buying the rest", n.Reason, covered, need)
			}
			n.Surplus = max(n.Built-need, 0)
			need -= covered
		}
		if need > 0 {
			n.Bought = need
			tree.Buy = append(tree.Buy, TypeQty{t, need})
		}
		tree.Nodes = append(tree.Nodes, n)
	}
	sort.Slice(tree.Buy, func(i, j int) bool { return tree.Buy[i].TypeID < tree.Buy[j].TypeID })
	return tree, nil
}

func runsKey(c Copy) int64 {
	if c.Runs < 0 {
		return 1 << 62
	}
	return c.Runs
}

func decide(req TreeRequest, t int64, root bool, copies map[int64][]Copy) (string, string) {
	r, hasRecipe := req.Recipe(t)
	if d, ok := req.Decisions[t]; ok && d != DecideAuto {
		if d == DecideBuild && !hasRecipe {
			return DecideBuy, "asked to build, but nothing in the SDE makes it"
		}
		return d, "your choice"
	}
	if !hasRecipe {
		return DecideBuy, "raw material or not buildable"
	}
	if root {
		return DecideBuild, "the product being planned"
	}
	if len(copies[r.BlueprintTypeID]) > 0 {
		if req.CanBuild != nil {
			if ok, why := req.CanBuild(r); !ok {
				return DecideBuy, "you own blueprints for it, but " + why
			}
		}
		return DecideBuild, "you own blueprints for it"
	}
	return DecideBuy, "no owned blueprint"
}

func makeJob(req TreeRequest, r Recipe, product int64, runs int64, c Copy) TreeJob {
	m := req.Multipliers(r.Activity, product)
	job := TreeJob{
		ProductTypeID: product, BlueprintTypeID: r.BlueprintTypeID, Activity: r.Activity, Runs: runs,
		ME: c.ME, TE: c.TE, CopyItemID: c.ItemID, Output: runs * max(r.OutputPerRun, 1),
		Seconds: r.BaseSeconds * float64(runs) * (1 - float64(c.TE)/100) * m.Time, Skills: r.Skills,
	}
	for _, mat := range r.Materials {
		job.Materials = append(job.Materials, TypeQty{mat.TypeID, MaterialQuantity(mat.Quantity, runs, c.ME, m.Material)})
	}
	return job
}

func addMaterials(demand map[int64]int64, mats []TypeQty) {
	for _, m := range mats {
		demand[m.TypeID] += m.Quantity
	}
}

func ceilDiv(a, b int64) int64 { return (a + b - 1) / b }

// NetStock subtracts owned stock from needs. remaining is what's still
// needed; used is what stock covered.
func NetStock(needs []TypeQty, stock map[int64]int64) (remaining, used []TypeQty) {
	for _, n := range needs {
		u := min(stock[n.TypeID], n.Quantity)
		if u > 0 {
			used = append(used, TypeQty{n.TypeID, u})
		}
		if n.Quantity-u > 0 {
			remaining = append(remaining, TypeQty{n.TypeID, n.Quantity - u})
		}
	}
	return remaining, used
}
