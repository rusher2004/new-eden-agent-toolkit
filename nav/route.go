package nav

import (
	"container/heap"
	"fmt"
	"math"
	"slices"
	"strings"
)

// RouteSystem is a solar system for routing.
type RouteSystem struct {
	ID              int64
	Name            string
	ConstellationID int64
	RegionID        int64
	RegionName      string
	Security        float64
	Pos             Position
	HasNPCStation   bool
}

// RouteGraph holds systems, stargates and a spatial index for jump search.
type RouteGraph struct {
	Systems map[int64]RouteSystem
	gates   map[int64][]int64
	cells   map[[3]int64][]int64
	cellLY  float64
}

// NewRouteGraph builds a graph from systems and stargate connections. Gate
// links are treated as two-way.
func NewRouteGraph(systems []RouteSystem, gates [][2]int64) *RouteGraph {
	g := &RouteGraph{Systems: map[int64]RouteSystem{}, gates: map[int64][]int64{}, cells: map[[3]int64][]int64{}, cellLY: 5}
	for _, s := range systems {
		g.Systems[s.ID] = s
		if s.RegionID < FirstWormholeRegionID {
			c := g.cellOf(s.Pos)
			g.cells[c] = append(g.cells[c], s.ID)
		}
	}
	add := func(a, b int64) {
		if !slices.Contains(g.gates[a], b) {
			g.gates[a] = append(g.gates[a], b)
		}
	}
	for _, e := range gates {
		if _, ok := g.Systems[e[0]]; !ok {
			continue
		}
		if _, ok := g.Systems[e[1]]; !ok {
			continue
		}
		add(e[0], e[1])
		add(e[1], e[0])
	}
	return g
}

func (g *RouteGraph) cellOf(p Position) [3]int64 {
	s := g.cellLY * MetersPerLightYear
	return [3]int64{int64(math.Floor(p.X / s)), int64(math.Floor(p.Y / s)), int64(math.Floor(p.Z / s))}
}

// withinRange lists k-space systems within rangeLY of a system.
func (g *RouteGraph) withinRange(from int64, rangeLY float64) []int64 {
	src := g.Systems[from]
	c := g.cellOf(src.Pos)
	r := int64(math.Ceil(rangeLY / g.cellLY))
	var out []int64
	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			for dz := -r; dz <= r; dz++ {
				for _, id := range g.cells[[3]int64{c[0] + dx, c[1] + dy, c[2] + dz}] {
					if id != from && LightYears(src.Pos, g.Systems[id].Pos) <= rangeLY {
						out = append(out, id)
					}
				}
			}
		}
	}
	return out
}

// Optimize modes.
const (
	OptimizeFuel    = "fuel"
	OptimizeJumps   = "jumps"
	OptimizeFatigue = "fatigue"
)

// RoutePrefs are the pilot's routing preferences. Set-valued fields are
// system ID sets the caller resolves (e.g. sovereignty held by hostile
// alliances, active incursions).
type RoutePrefs struct {
	AvoidSystems         map[int64]bool
	AvoidConstellations  map[int64]bool
	AvoidRegions         map[int64]bool
	AvoidSovSystems      map[int64]bool // held by avoided or bad-standing alliances
	PreferSovSystems     map[int64]bool // held by blue alliances
	Midpoints            map[int64]bool // systems with docking, cyno alts or friendly POSes
	MidpointsRequired    bool
	PreferStationSystems bool
	AvoidIncursions      bool
	IncursionSystems     map[int64]bool
	Optimize             string
	MaxGates             *int          // nil = no limit; 0 = no gates at all
	MaxRecentKills       int           // 0 = no limit; per gate system
	RecentKills          map[int64]int // ship + pod kills in the last hour by system
}

// RouteRequest asks for routes between two systems.
type RouteRequest struct {
	From, To int64
	Drive    JumpDrive
	Skills   JumpSkills
	Prefs    RoutePrefs
	// CanUseHighsec is true for hulls allowed in highsec (freighters, jump
	// freighters, blockade runners): they may start jumps from highsec and use
	// highsec gates. Capitals that can't enter highsec set it false.
	CanUseHighsec bool
}

// Leg is one jump or gate step.
type Leg struct {
	From, To         int64
	FromName, ToName string
	FromSecurity     Security `json:"fromSecurity"`
	ToSecurity       Security `json:"toSecurity"`
	Kind             string   // "jump" or "gate"
	LY               float64  `json:"ly,omitempty"`
	Fuel             int64    `json:"fuel,omitempty"`
	FatigueAfter     float64  `json:"fatigueAfterMinutes,omitempty"`
	CooldownMinutes  float64  `json:"cooldownMinutes,omitempty"`
}

// DotlanLink is a labelled dotlan URL for one segment of a route.
type DotlanLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// Route is one route option.
type Route struct {
	Label          string       `json:"label"`
	Systems        []int64      `json:"systems"`
	Legs           []Leg        `json:"legs"`
	Jumps          int          `json:"jumps"`
	Gates          int          `json:"gates"`
	HighsecGates   int          `json:"highsecGates"`
	LY             float64      `json:"ly"`
	Fuel           int64        `json:"fuel"`
	FatigueMinutes float64      `json:"fatigueMinutes"`
	Broken         []string     `json:"brokenPreferences,omitempty"`
	Links          []DotlanLink `json:"dotlanLinks"`
	Suggestion     *DotlanLink  `json:"dotlanSuggestion,omitempty"`
}

// Gate leg costs, in the same units as the objective (heuristics, not game
// rules): a highsec gate is cheap, a low/nullsec gate costs more, and each
// recent kill in the system adds to it.
const (
	gateCostHighsec = 0.3
	gateCostUnsafe  = 1.5
	gateCostPerKill = 0.5
	preferDiscount  = 0.7
	noStationFactor = 1.3
)

func (g *RouteGraph) avoided(id int64, p RoutePrefs, req RouteRequest) bool {
	if id == req.From || id == req.To {
		return false
	}
	s := g.Systems[id]
	return p.AvoidSystems[id] || p.AvoidConstellations[s.ConstellationID] || p.AvoidRegions[s.RegionID] ||
		p.AvoidSovSystems[id] || (p.AvoidIncursions && p.IncursionSystems[id])
}

type pqItem struct {
	node  int64
	gates int
	cost  float64
}
type pq []pqItem

func (q pq) Len() int           { return len(q) }
func (q pq) Less(i, j int) bool { return q[i].cost < q[j].cost }
func (q pq) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *pq) Push(x any)        { *q = append(*q, x.(pqItem)) }
func (q *pq) Pop() any          { o := *q; it := o[len(o)-1]; *q = o[:len(o)-1]; return it }

type stateKey struct {
	node  int64
	gates int
}

// shortest runs Dijkstra and returns the system sequence and the kind of
// each step ("jump" or "gate"), or nil.
func (g *RouteGraph) shortest(req RouteRequest, soft bool) ([]int64, []string) {
	p := req.Prefs
	rng := JumpRange(req.Drive, req.Skills)
	maxGates := -1 // no limit
	if p.MaxGates != nil {
		maxGates = max(*p.MaxGates, 0)
	}
	gateKey := func(n int) int {
		if maxGates < 0 {
			return 0
		}
		return n
	}
	dist := map[stateKey]float64{}
	prev := map[stateKey]stateKey{}
	kind := map[stateKey]string{}
	start := stateKey{req.From, 0}
	dist[start] = 0
	q := &pq{{req.From, 0, 0}}
	var end *stateKey
	for q.Len() > 0 {
		it := heap.Pop(q).(pqItem)
		k := stateKey{it.node, gateKey(it.gates)}
		if it.cost > dist[k] {
			continue
		}
		if it.node == req.To {
			end = &k
			break
		}
		cur := g.Systems[it.node]
		relax := func(to int64, gates int, w float64, how string) {
			nk := stateKey{to, gateKey(gates)}
			if d, ok := dist[nk]; !ok || it.cost+w < d {
				dist[nk] = it.cost + w
				prev[nk] = k
				kind[nk] = how
				heap.Push(q, pqItem{to, gates, it.cost + w})
			}
		}
		// Jumps.
		if rng > 0 && (req.CanUseHighsec || BandOf(cur.Security, cur.RegionID) != Highsec) {
			for _, to := range g.withinRange(it.node, rng) {
				dst := g.Systems[to]
				if !CynoLegal(dst.Security, dst.RegionID) || g.avoided(to, p, req) {
					continue
				}
				if p.MidpointsRequired && to != req.To && !p.Midpoints[to] {
					continue
				}
				ly := LightYears(cur.Pos, dst.Pos)
				var w float64
				switch p.Optimize {
				case OptimizeJumps:
					w = 1 + ly*1e-3
				case OptimizeFatigue:
					w = math.Log1p(ly*fatigueMult(req.Drive)) + 1e-4*ly
				default:
					w = ly
				}
				if soft && to != req.To {
					if p.Midpoints[to] || p.PreferSovSystems[to] {
						w *= preferDiscount
					}
					if p.PreferStationSystems && !dst.HasNPCStation {
						w *= noStationFactor
					}
				}
				relax(to, it.gates, w, "jump")
			}
		}
		// Gates.
		if maxGates >= 0 && it.gates >= maxGates {
			continue
		}
		for _, to := range g.gates[it.node] {
			dst := g.Systems[to]
			if g.avoided(to, p, req) {
				continue
			}
			hs := BandOf(dst.Security, dst.RegionID) == Highsec
			if hs && !req.CanUseHighsec {
				continue
			}
			kills := p.RecentKills[to]
			if p.MaxRecentKills > 0 && kills > p.MaxRecentKills && to != req.To {
				continue
			}
			w := gateCostUnsafe
			if hs {
				w = gateCostHighsec
			}
			w += gateCostPerKill * float64(kills)
			if p.Optimize == OptimizeJumps {
				w *= 0.5
			}
			relax(to, it.gates+1, w, "gate")
		}
	}
	if end == nil {
		return nil, nil
	}
	var path []int64
	var kinds []string
	for k := *end; ; {
		path = append(path, k.node)
		if k == start {
			break
		}
		kinds = append(kinds, kind[k])
		k = prev[k]
	}
	slices.Reverse(path)
	slices.Reverse(kinds)
	return path, kinds
}

func fatigueMult(d JumpDrive) float64 {
	if d.FatigueMultiplier <= 0 {
		return 1
	}
	return d.FatigueMultiplier
}

// PlanRoutes returns the best route under the pilot's preferences plus
// alternatives (fewest jumps, least fatigue, ignoring soft preferences),
// without duplicates.
func (g *RouteGraph) PlanRoutes(req RouteRequest) ([]Route, error) {
	if _, ok := g.Systems[req.From]; !ok {
		return nil, fmt.Errorf("plan route: unknown origin system %d", req.From)
	}
	if _, ok := g.Systems[req.To]; !ok {
		return nil, fmt.Errorf("plan route: unknown destination system %d", req.To)
	}
	if req.Prefs.Optimize == "" {
		req.Prefs.Optimize = OptimizeFuel
	}
	type variant struct {
		label    string
		optimize string
		soft     bool
		hard     bool // keep MidpointsRequired
	}
	variants := []variant{
		{"Best for your preferences (" + req.Prefs.Optimize + ")", req.Prefs.Optimize, true, true},
		{"Fewest jumps", OptimizeJumps, true, true},
		{"Least fatigue", OptimizeFatigue, true, true},
		{"Ignoring preferred midpoints", req.Prefs.Optimize, false, false},
	}
	var out []Route
	seen := map[string]bool{}
	for _, v := range variants {
		r := req
		r.Prefs.Optimize = v.optimize
		if !v.hard {
			r.Prefs.MidpointsRequired = false
		}
		path, kinds := g.shortest(r, v.soft)
		if path == nil {
			continue
		}
		key := fmt.Sprint(path)
		if seen[key] {
			continue
		}
		seen[key] = true
		rt := g.describe(req, path, kinds)
		rt.Label = v.label
		out = append(out, rt)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("plan route: no route from %s to %s with these preferences and a %.1f LY range", g.Systems[req.From].Name, g.Systems[req.To].Name, JumpRange(req.Drive, req.Skills))
	}
	return out, nil
}

// ValidateRoute checks and costs a pinned route given as waypoints. Between
// consecutive waypoints it jumps when the hop is in range and the landing is
// legal, and otherwise fills in the shortest gate path.
func (g *RouteGraph) ValidateRoute(req RouteRequest, waypoints []int64) (Route, error) {
	if len(waypoints) < 2 {
		return Route{}, fmt.Errorf("validate route: need at least two waypoints")
	}
	rng := JumpRange(req.Drive, req.Skills)
	path := []int64{waypoints[0]}
	var kinds []string
	for i := 1; i < len(waypoints); i++ {
		a, b := waypoints[i-1], waypoints[i]
		sa, okA := g.Systems[a]
		sb, okB := g.Systems[b]
		if !okA || !okB {
			return Route{}, fmt.Errorf("validate route: unknown system in %d → %d", a, b)
		}
		ly := LightYears(sa.Pos, sb.Pos)
		if CynoLegal(sb.Security, sb.RegionID) && ly <= rng && (req.CanUseHighsec || BandOf(sa.Security, sa.RegionID) != Highsec) {
			path = append(path, b)
			kinds = append(kinds, "jump")
			continue
		}
		{
			gp := g.gatePath(a, b, req.CanUseHighsec)
			if gp == nil {
				if CynoLegal(sb.Security, sb.RegionID) {
					return Route{}, fmt.Errorf("validate route: %s → %s is %.2f LY, beyond the %.1f LY range, and no gate path connects them", sa.Name, sb.Name, ly, rng)
				}
				return Route{}, fmt.Errorf("validate route: can't jump into %s (%s) and no gate path reaches it", sb.Name, BandOf(sb.Security, sb.RegionID))
			}
			path = append(path, gp[1:]...)
			for range gp[1:] {
				kinds = append(kinds, "gate")
			}
		}
	}
	rt := g.describe(req, path, kinds)
	rt.Label = "Your route"
	for _, id := range path[1 : len(path)-1] {
		if g.avoided(id, req.Prefs, req) {
			rt.Broken = append(rt.Broken, fmt.Sprintf("passes %s, which your preferences avoid", g.Systems[id].Name))
		}
	}
	// A negative cap counts as no gates, as in shortest.
	if mg := req.Prefs.MaxGates; mg != nil && rt.Gates > max(*mg, 0) {
		rt.Broken = append(rt.Broken, fmt.Sprintf("takes %d gates, more than your limit of %d", rt.Gates, max(*mg, 0)))
	}
	return rt, nil
}

// gatePath is the fewest-gates path, or nil.
func (g *RouteGraph) gatePath(a, b int64, highsecOK bool) []int64 {
	prev := map[int64]int64{a: a}
	queue := []int64{a}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == b {
			var p []int64
			for x := b; ; x = prev[x] {
				p = append(p, x)
				if x == a {
					break
				}
			}
			slices.Reverse(p)
			return p
		}
		for _, n := range g.gates[cur] {
			if _, seen := prev[n]; seen {
				continue
			}
			s := g.Systems[n]
			if !highsecOK && BandOf(s.Security, s.RegionID) == Highsec && n != b {
				continue
			}
			prev[n] = cur
			queue = append(queue, n)
		}
	}
	return nil
}

// describe turns a system path and its step kinds into legs, totals and
// dotlan links.
func (g *RouteGraph) describe(req RouteRequest, path []int64, kinds []string) Route {
	rt := Route{Systems: path}
	fatigue := 0.0
	for i := 1; i < len(path); i++ {
		a, b := g.Systems[path[i-1]], g.Systems[path[i]]
		leg := Leg{From: a.ID, To: b.ID, FromName: a.Name, ToName: b.Name,
			FromSecurity: SecurityOf(a.Security, a.RegionID), ToSecurity: SecurityOf(b.Security, b.RegionID)}
		if kinds[i-1] == "gate" {
			leg.Kind = "gate"
			rt.Gates++
			if BandOf(b.Security, b.RegionID) == Highsec {
				rt.HighsecGates++
			}
		} else {
			leg.Kind = "jump"
			leg.LY = LightYears(a.Pos, b.Pos)
			leg.Fuel = JumpFuel(req.Drive, req.Skills, leg.LY)
			var cd float64
			fatigue, cd = Fatigue(fatigue, leg.LY, req.Drive.FatigueMultiplier)
			leg.FatigueAfter, leg.CooldownMinutes = fatigue, cd
			fatigue = math.Max(fatigue-cd, 0) // decays while waiting out the cooldown
			rt.Jumps++
			rt.LY += leg.LY
			rt.Fuel += leg.Fuel
		}
		rt.Legs = append(rt.Legs, leg)
	}
	for _, l := range rt.Legs {
		if l.Kind == "jump" {
			rt.FatigueMinutes = l.FatigueAfter
		}
	}
	p := req.Prefs
	if p.MidpointsRequired {
		for _, l := range rt.Legs {
			if l.Kind == "jump" && l.To != req.To && !p.Midpoints[l.To] {
				rt.Broken = append(rt.Broken, fmt.Sprintf("lands in %s, which isn't one of your midpoints", l.ToName))
			}
		}
	}
	if p.PreferStationSystems {
		for _, l := range rt.Legs {
			if l.Kind == "jump" && l.To != req.To && !g.Systems[l.To].HasNPCStation {
				rt.Broken = append(rt.Broken, fmt.Sprintf("lands in %s, which has no NPC station", l.ToName))
			}
		}
	}
	rt.Links = g.dotlanLinks(req, rt.Legs)
	return rt
}

func (g *RouteGraph) dotlanLinks(req RouteRequest, legs []Leg) []DotlanLink {
	var links []DotlanLink
	for i := 0; i < len(legs); {
		j := i
		for j < len(legs) && legs[j].Kind == legs[i].Kind {
			j++
		}
		names := []string{legs[i].FromName}
		for _, l := range legs[i:j] {
			names = append(names, l.ToName)
		}
		label := fmt.Sprintf("%s: %s → %s", map[string]string{"jump": "Jumps", "gate": "Gates"}[legs[i].Kind], names[0], names[len(names)-1])
		if legs[i].Kind == "jump" {
			links = append(links, DotlanLink{Label: label, URL: DotlanJumpURL(req.Drive.ShipName, req.Skills, false, false, nil, names)})
		} else {
			links = append(links, DotlanLink{Label: label, URL: DotlanGateURL(names)})
		}
		i = j
	}
	return links
}

// Suggestion builds dotlan's own route for comparison: endpoints (and
// required midpoints) plus the pilot's preferences in dotlan's terms.
// Avoided constellations expand to their systems; sovereignty and standing
// preferences have no dotlan equivalent and are reported in the label.
func (g *RouteGraph) Suggestion(req RouteRequest, waypoints []int64) DotlanLink {
	p := req.Prefs
	var avoid []string
	for id := range p.AvoidRegions {
		for _, s := range g.Systems {
			if s.RegionID == id {
				avoid = append(avoid, s.RegionName)
				break
			}
		}
	}
	for id := range p.AvoidSystems {
		if s, ok := g.Systems[id]; ok {
			avoid = append(avoid, s.Name)
		}
	}
	for id := range p.AvoidConstellations {
		for _, s := range g.Systems {
			if s.ConstellationID == id {
				avoid = append(avoid, s.Name)
			}
		}
	}
	slices.Sort(avoid)
	avoid = slices.Compact(avoid)
	var names []string
	for _, w := range waypoints {
		names = append(names, g.Systems[w].Name)
	}
	label := "dotlan's own route with your avoid list"
	if len(p.AvoidSovSystems) > 0 || len(p.PreferSovSystems) > 0 {
		label += " (dotlan can't apply your sovereignty or standings preferences)"
	}
	return DotlanLink{Label: label, URL: DotlanJumpURL(req.Drive.ShipName, req.Skills, p.PreferStationSystems, p.AvoidIncursions, avoid, names)}
}

// dotlanName writes a system, region or ship name the way dotlan's URLs do:
// spaces become underscores, everything else is kept.
func dotlanName(s string) string { return strings.ReplaceAll(s, " ", "_") }

// DotlanJumpURL builds a dotlan jump planner URL:
// /jump/<Ship>,<JDC><JFC><JF>[,S][,I][,-A:-B]/<W1>:<W2>:...
// Format confirmed with the user 2026-11-18.
func DotlanJumpURL(ship string, s JumpSkills, preferStations, avoidIncursions bool, avoid []string, waypoints []string) string {
	var b strings.Builder
	b.WriteString("https://evemaps.dotlan.net/jump/")
	b.WriteString(dotlanName(ship))
	fmt.Fprintf(&b, ",%d%d%d", s.JumpDriveCalibration, s.JumpFuelConservation, s.JumpFreighters)
	if preferStations {
		b.WriteString(",S")
	}
	if avoidIncursions {
		b.WriteString(",I")
	}
	if len(avoid) > 0 {
		parts := make([]string, len(avoid))
		for i, a := range avoid {
			parts[i] = "-" + dotlanName(a)
		}
		b.WriteString(",")
		b.WriteString(strings.Join(parts, ":"))
	}
	b.WriteString("/")
	parts := make([]string, len(waypoints))
	for i, w := range waypoints {
		parts[i] = dotlanName(w)
	}
	b.WriteString(strings.Join(parts, ":"))
	return b.String()
}

// DotlanGateURL builds a dotlan route planner URL through every given system.
func DotlanGateURL(systems []string) string {
	parts := make([]string, len(systems))
	for i, s := range systems {
		parts[i] = dotlanName(s)
	}
	return "https://evemaps.dotlan.net/route/" + strings.Join(parts, ":")
}

// GateJumps is the fewest stargate jumps from one system to another, or -1
// when no gate path joins them (wormhole space, Pochven and the like).
func (g *RouteGraph) GateJumps(from, to int64) int {
	p := g.gatePath(from, to, true)
	if p == nil {
		return -1
	}
	return len(p) - 1
}
