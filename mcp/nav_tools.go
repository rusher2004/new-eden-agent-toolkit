package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/market"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// Jump skill type IDs (verified against the SDE 2026-11-18).
const (
	skillJumpDriveCalibration int64 = 21611
	skillJumpFuelConservation int64 = 21610
	skillJumpFreighters       int64 = 29029
)

// jumpDrive reads a hull's jump drive from SDE dogma.
func (p *planner) jumpDrive(ctx context.Context, shipTypeID int64) (nav.JumpDrive, error) {
	attrs, err := p.sde.TypeAttributes(ctx, p.ua(), shipTypeID)
	if err != nil {
		return nav.JumpDrive{}, err
	}
	return nav.JumpDrive{
		ShipTypeID: shipTypeID, ShipName: p.typeName(ctx, shipTypeID),
		BaseRangeLY: attrs[sde.AttrJumpDriveRange], FuelPerLY: attrs[sde.AttrJumpDriveConsumption],
		FuelTypeID: int64(attrs[sde.AttrJumpDriveConsumptionType]), SkillBonusPerLevel: attrs[sde.AttrJumpFreighterFuelBonus],
		FatigueMultiplier: attrs[sde.AttrJumpFatigueMultiplier],
	}, nil
}

// canUseHighsec reports whether a hull may enter highsec: everything except
// the capital groups that are barred from it.
func canUseHighsec(groupID int64) bool {
	return !slices.Contains(industry.CapitalShipyardGroups, groupID) && !slices.Contains(industry.SupercapitalGroups, groupID)
}

func (p *planner) routeGraph(ctx context.Context) (*nav.RouteGraph, error) {
	if p.graph != nil {
		return p.graph, nil
	}
	sys, err := p.sde.RoutingSystems(ctx, p.ua())
	if err != nil {
		return nil, err
	}
	gates, err := p.sde.StargateConnections(ctx, p.ua())
	if err != nil {
		return nil, err
	}
	rs := make([]nav.RouteSystem, len(sys))
	for i, s := range sys {
		rs[i] = nav.RouteSystem{ID: s.ID, Name: s.Name, ConstellationID: s.ConstellationID, RegionID: s.RegionID, RegionName: s.RegionName,
			Security: s.Security, Pos: nav.Position{X: s.X, Y: s.Y, Z: s.Z}, HasNPCStation: s.HasNPCStation}
	}
	p.graph = nav.NewRouteGraph(rs, gates)
	return p.graph, nil
}

// routePrefs resolves saved preferences into system sets, fetching
// sovereignty, contacts, incursions and recent kills only when needed.
func (p *planner) routePrefs(ctx context.Context, pilotID int32, prefs config.RoutePrefs) (nav.RoutePrefs, []string, error) {
	set := func(ids []int64) map[int64]bool {
		m := map[int64]bool{}
		for _, id := range ids {
			m[id] = true
		}
		return m
	}
	out := nav.RoutePrefs{
		AvoidSystems: set(prefs.AvoidSystems), AvoidConstellations: set(prefs.AvoidConstellations), AvoidRegions: set(prefs.AvoidRegions),
		Midpoints: set(prefs.Midpoints), MidpointsRequired: prefs.MidpointsRequired, PreferStationSystems: prefs.PreferStationSystems,
		AvoidIncursions: prefs.AvoidIncursions, Optimize: prefs.Optimize, MaxGates: prefs.MaxGates, MaxRecentKills: prefs.MaxRecentKills,
		AvoidSovSystems: map[int64]bool{}, PreferSovSystems: map[int64]bool{},
	}
	var notes []string
	avoidAlliances := set(prefs.AvoidAlliances)
	preferAlliances := set(prefs.PreferAlliances)
	if prefs.AvoidBadStanding {
		hostile, err := p.badStandingAlliances(ctx, pilotID)
		if err != nil {
			notes = append(notes, "couldn't read contacts, so bad-standing sovereignty isn't avoided: "+err.Error())
		}
		for id := range hostile {
			avoidAlliances[id] = true
		}
	}
	if len(avoidAlliances) > 0 || len(preferAlliances) > 0 {
		sov, err := listSovereigntySystems(ctx, p.ua())
		if err != nil {
			return out, notes, fmt.Errorf("sovereignty map: %w", err)
		}
		for _, s := range sov {
			if s.Claim.Alliance == nil || s.Claim.Alliance.Alliance == nil {
				continue
			}
			a := s.Claim.Alliance.Alliance.AllianceId
			if avoidAlliances[a] {
				out.AvoidSovSystems[s.SolarSystemId] = true
			}
			if preferAlliances[a] {
				out.PreferSovSystems[s.SolarSystemId] = true
			}
		}
	}
	if prefs.AvoidIncursions {
		inc, err := listIncursions(ctx, p.ua())
		if err != nil {
			return out, notes, fmt.Errorf("incursions: %w", err)
		}
		out.IncursionSystems = map[int64]bool{}
		for _, i := range inc {
			for _, s := range i.InfestedSolarSystemIDs {
				out.IncursionSystems[s] = true
			}
		}
	}
	kills, err := listUniverseSystemKills(ctx, p.ua())
	if err != nil {
		notes = append(notes, "recent kills unavailable: "+err.Error())
	} else {
		out.RecentKills = map[int64]int{}
		for _, k := range kills {
			out.RecentKills[k.SystemID] = int(k.ShipKills + k.PodKills)
		}
	}
	return out, notes, nil
}

// badStandingAlliances returns alliances the pilot, their corporation or
// their alliance have set below zero standing.
func (p *planner) badStandingAlliances(ctx context.Context, pilotID int32) (map[int64]bool, error) {
	out := map[int64]bool{}
	add := func(cs []Contact) {
		for _, c := range cs {
			if c.ContactType == "alliance" && c.Standing < 0 {
				out[c.ContactID] = true
			}
		}
	}
	mine, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Contact, int32, error) {
		return listCharacterContacts(ctx, p.clientID, pilotID, p.ua(), page)
	})
	if err != nil {
		return out, fmt.Errorf("character contacts: %w", err)
	}
	add(mine)
	aff, err := getCharacterAffiliations(ctx, p.ua(), []int64{int64(pilotID)})
	if err != nil || len(aff) == 0 {
		return out, nil
	}
	if corp, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Contact, int32, error) {
		return listCorporationContacts(ctx, p.clientID, pilotID, p.ua(), aff[0].CorporationID, page)
	}); err == nil {
		add(corp)
	}
	if aff[0].AllianceID != nil {
		if al, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Contact, int32, error) {
			return listAllianceContacts(ctx, p.clientID, pilotID, p.ua(), *aff[0].AllianceID, page)
		}); err == nil {
			add(al)
		}
	}
	return out, nil
}

type RoutePrefsInput struct {
	AvoidSystems         []string `json:"avoidSystems,omitempty" jsonschema:"system names to never stop in or pass through"`
	AvoidConstellations  []string `json:"avoidConstellations,omitempty"`
	AvoidRegions         []string `json:"avoidRegions,omitempty"`
	AvoidAlliances       []int64  `json:"avoidAlliances,omitempty" jsonschema:"alliance IDs whose sovereignty to avoid"`
	AvoidBadStanding     bool     `json:"avoidBadStanding,omitempty" jsonschema:"avoid sovereignty of alliances the pilot, corp or alliance set below zero"`
	PreferAlliances      []int64  `json:"preferAlliances,omitempty" jsonschema:"alliance IDs whose sovereignty to prefer"`
	Midpoints            []string `json:"midpoints,omitempty" jsonschema:"systems where the pilot has docking, a cyno alt or a friendly POS"`
	MidpointsRequired    bool     `json:"midpointsRequired,omitempty" jsonschema:"only land on these midpoints"`
	PreferStationSystems bool     `json:"preferStationSystems,omitempty"`
	AvoidIncursions      bool     `json:"avoidIncursions,omitempty"`
	Optimize             string   `json:"optimize,omitempty" jsonschema:"fuel (default), jumps or fatigue"`
	MaxGates             int      `json:"maxGates,omitempty"`
	MaxRecentKills       int      `json:"maxRecentKills,omitempty" jsonschema:"skip gate systems with more ship and pod kills than this in the last hour"`
}

type SaveHaulProfileInput struct {
	Name     string   `json:"name"`
	Ship     string   `json:"ship" jsonschema:"hull type name, e.g. Nomad"`
	PilotID  int64    `json:"pilotId" jsonschema:"the pilot's character ID (skills and contacts are read from ESI)"`
	HoldM3   float64  `json:"holdM3" jsonschema:"the fitted cargo hold in m³ as shown in game"`
	Outbound []string `json:"outbound,omitempty" jsonschema:"optional pinned route, hub to destination, as system names; saved in routes.yaml as '<name> outbound'"`
	Return   []string `json:"return,omitempty" jsonschema:"optional pinned return route as system names; saved in routes.yaml as '<name> return'"`
	// OutboundRoute and ReturnRoute pin routes already saved by name.
	OutboundRoute string          `json:"outboundRoute,omitempty" jsonschema:"pin a route already saved with save_route, by name, instead of giving outbound"`
	ReturnRoute   string          `json:"returnRoute,omitempty" jsonschema:"pin a saved route by name for the return trip, instead of giving return"`
	Prefs         RoutePrefsInput `json:"prefs"`
	Notes         *string         `json:"notes,omitempty" jsonschema:"what the user says about this profile; omit to keep the notes it has"`
}

func (p *planner) resolvePrefs(ctx context.Context, in RoutePrefsInput) (config.RoutePrefs, error) {
	out := config.RoutePrefs{AvoidAlliances: in.AvoidAlliances, AvoidBadStanding: in.AvoidBadStanding, PreferAlliances: in.PreferAlliances,
		MidpointsRequired: in.MidpointsRequired, PreferStationSystems: in.PreferStationSystems, AvoidIncursions: in.AvoidIncursions,
		Optimize: in.Optimize, MaxGates: in.MaxGates, MaxRecentKills: in.MaxRecentKills}
	switch in.Optimize {
	case "", nav.OptimizeFuel, nav.OptimizeJumps, nav.OptimizeFatigue:
	default:
		return out, fmt.Errorf("optimize must be fuel, jumps or fatigue, not %q", in.Optimize)
	}
	var err error
	if out.AvoidSystems, err = p.systemIDs(ctx, in.AvoidSystems); err != nil {
		return out, err
	}
	if out.Midpoints, err = p.systemIDs(ctx, in.Midpoints); err != nil {
		return out, err
	}
	cons, regs := map[string]int64{}, map[string]int64{}
	for _, s := range p.systems {
		cons[strings.ToLower(s.ConstellationName)] = s.ConstellationID
		regs[strings.ToLower(s.RegionName)] = s.RegionID
	}
	for _, n := range in.AvoidConstellations {
		id, ok := cons[strings.ToLower(n)]
		if !ok {
			return out, fmt.Errorf("unknown constellation %q", n)
		}
		out.AvoidConstellations = append(out.AvoidConstellations, id)
	}
	for _, n := range in.AvoidRegions {
		id, ok := regs[strings.ToLower(n)]
		if !ok {
			return out, fmt.Errorf("unknown region %q", n)
		}
		out.AvoidRegions = append(out.AvoidRegions, id)
	}
	return out, nil
}

func saveHaulProfileHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[SaveHaulProfileInput, config.HaulProfile] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveHaulProfileInput) (*mcpsdk.CallToolResult, config.HaulProfile, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		if strings.TrimSpace(in.Name) == "" || in.HoldM3 <= 0 {
			return nil, config.HaulProfile{}, fmt.Errorf("save haul profile: name and a positive holdM3 are required")
		}
		ids, err := sdeStore.TypesByName(ctx, p.ua(), []string{in.Ship})
		if err != nil {
			return nil, config.HaulProfile{}, err
		}
		ship, ok := ids[in.Ship]
		if !ok {
			return nil, config.HaulProfile{}, fmt.Errorf("save haul profile: unknown ship %q", in.Ship)
		}
		prefs, err := p.resolvePrefs(ctx, in.Prefs)
		if err != nil {
			return nil, config.HaulProfile{}, fmt.Errorf("save haul profile: %w", err)
		}
		prof := config.HaulProfile{Name: strings.TrimSpace(in.Name), ShipTypeID: ship, ShipName: p.typeName(ctx, ship), PilotID: in.PilotID, HoldM3: in.HoldM3, Prefs: prefs}
		if _, err := store.Update(func(cfg *config.Config) error {
			if c, ok := cfg.Character(int32(in.PilotID)); ok {
				prof.PilotName = c.Name
			}
			if old, ok := cfg.HaulProfile(prof.Name); ok {
				prof.Name, prof.Notes = old.Name, old.Notes
			}
			if in.Notes != nil {
				prof.Notes = strings.TrimSpace(*in.Notes)
			}
			pin := func(systems []string, named, suffix string) (string, error) {
				if len(systems) > 0 && named != "" {
					return "", fmt.Errorf("give %s systems or a saved %s route by name, not both", suffix, suffix)
				}
				if len(systems) > 0 {
					r, err := p.namedRoute(ctx, prof.Name+" "+suffix, systems)
					if err != nil {
						return "", err
					}
					if old, ok := cfg.Route(r.Name); ok {
						r.ShipClass, r.Notes = old.ShipClass, old.Notes
					}
					cfg.UpsertRoute(r)
					return r.Name, nil
				}
				if named == "" {
					return "", nil
				}
				r, ok := cfg.Route(named)
				if !ok {
					return "", fmt.Errorf("no saved route named %q; save it with save_route first", named)
				}
				return r.Name, nil
			}
			var err error
			if prof.OutboundRoute, err = pin(in.Outbound, in.OutboundRoute, "outbound"); err != nil {
				return fmt.Errorf("save haul profile: %w", err)
			}
			if prof.ReturnRoute, err = pin(in.Return, in.ReturnRoute, "return"); err != nil {
				return fmt.Errorf("save haul profile: %w", err)
			}
			cfg.UpsertHaulProfile(prof)
			return nil
		}); err != nil {
			return nil, config.HaulProfile{}, err
		}
		return nil, prof, nil
	}
}

type ListHaulProfilesOutput struct {
	Profiles []config.HaulProfile `json:"profiles"`
}

func listHaulProfilesHandler(store *config.Store) mcpsdk.ToolHandlerFor[struct{}, ListHaulProfilesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, ListHaulProfilesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListHaulProfilesOutput{}, err
		}
		return nil, ListHaulProfilesOutput{Profiles: cfg.HaulProfiles}, nil
	}
}

// --- hub profiles ---

type PlanJumpRouteInput struct {
	HaulProfile string           `json:"haulProfile" jsonschema:"haul profile name (ship, pilot and saved preferences)"`
	From        string           `json:"from" jsonschema:"origin system name"`
	To          string           `json:"to" jsonschema:"destination system name"`
	Waypoints   []string         `json:"waypoints,omitempty" jsonschema:"validate and cost this exact route instead of searching (system names, origin to destination)"`
	Route       string           `json:"route,omitempty" jsonschema:"validate and cost a route saved with save_route, by name, instead of giving waypoints"`
	Prefs       *RoutePrefsInput `json:"prefs,omitempty" jsonschema:"override the profile's saved preferences for this call"`
}

type PlanJumpRouteOutput struct {
	Ship       string         `json:"ship"`
	RangeLY    float64        `json:"rangeLy"`
	FuelPerLY  float64        `json:"fuelPerLy"`
	FuelName   string         `json:"fuelName"`
	FuelPrice  float64        `json:"fuelPrice"`
	Routes     []nav.Route    `json:"routes"`
	Suggestion nav.DotlanLink `json:"dotlanSuggestion"`
	Notes      []string       `json:"notes,omitempty"`
}

func (p *planner) planRoute(ctx context.Context, prof config.HaulProfile, from, to int64, pinned []int64, prefOverride *config.RoutePrefs) (PlanJumpRouteOutput, *nav.RouteGraph, nav.RouteRequest, error) {
	drive, err := p.jumpDrive(ctx, prof.ShipTypeID)
	if err != nil {
		return PlanJumpRouteOutput{}, nil, nav.RouteRequest{}, err
	}
	s, err := p.skills(ctx, int32(prof.PilotID))
	if err != nil {
		return PlanJumpRouteOutput{}, nil, nav.RouteRequest{}, err
	}
	saved := prof.Prefs
	if prefOverride != nil {
		saved = *prefOverride
	}
	prefs, notes, err := p.routePrefs(ctx, int32(prof.PilotID), saved)
	if err != nil {
		return PlanJumpRouteOutput{}, nil, nav.RouteRequest{}, err
	}
	g, err := p.routeGraph(ctx)
	if err != nil {
		return PlanJumpRouteOutput{}, nil, nav.RouteRequest{}, err
	}
	b, err := p.typeBasics(ctx, []int64{prof.ShipTypeID, drive.FuelTypeID})
	if err != nil {
		return PlanJumpRouteOutput{}, nil, nav.RouteRequest{}, err
	}
	req := nav.RouteRequest{From: from, To: to, Drive: drive, Prefs: prefs, CanUseHighsec: canUseHighsec(b[prof.ShipTypeID].GroupID),
		Skills: nav.JumpSkills{JumpDriveCalibration: s[skillJumpDriveCalibration], JumpFuelConservation: s[skillJumpFuelConservation], JumpFreighters: s[skillJumpFreighters]}}
	out := PlanJumpRouteOutput{Ship: drive.ShipName, RangeLY: nav.JumpRange(drive, req.Skills), FuelPerLY: nav.FuelPerLY(drive, req.Skills), FuelName: b[drive.FuelTypeID].Name, Notes: notes}
	if len(pinned) > 0 {
		rt, err := g.ValidateRoute(req, pinned)
		if err != nil {
			return out, g, req, err
		}
		out.Routes = []nav.Route{rt}
	} else {
		if out.Routes, err = g.PlanRoutes(req); err != nil {
			return out, g, req, err
		}
	}
	// With required midpoints, dotlan gets the midpoints the best route
	// actually lands on, in order; it has no "only land here" option.
	wps := []int64{from}
	if prefs.MidpointsRequired && len(out.Routes) > 0 {
		for _, l := range out.Routes[0].Legs {
			if l.Kind == "jump" && l.To != to && prefs.Midpoints[l.To] {
				wps = append(wps, l.To)
			}
		}
	}
	out.Suggestion = g.Suggestion(req, append(wps, to))
	return out, g, req, nil
}

func planJumpRouteHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[PlanJumpRouteInput, PlanJumpRouteOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in PlanJumpRouteInput) (*mcpsdk.CallToolResult, PlanJumpRouteOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		cfg, err := store.Load()
		if err != nil {
			return nil, PlanJumpRouteOutput{}, err
		}
		prof, ok := cfg.HaulProfile(in.HaulProfile)
		if !ok {
			return nil, PlanJumpRouteOutput{}, fmt.Errorf("no haul profile named %q", in.HaulProfile)
		}
		ends, err := p.systemIDs(ctx, []string{in.From, in.To})
		if err != nil {
			return nil, PlanJumpRouteOutput{}, err
		}
		pinned, err := p.systemIDs(ctx, in.Waypoints)
		if err != nil {
			return nil, PlanJumpRouteOutput{}, err
		}
		if in.Route != "" {
			if len(in.Waypoints) > 0 {
				return nil, PlanJumpRouteOutput{}, fmt.Errorf("plan jump route: give waypoints or route, not both")
			}
			r, ok := cfg.Route(in.Route)
			if !ok {
				return nil, PlanJumpRouteOutput{}, fmt.Errorf("no saved route named %q", in.Route)
			}
			pinned = r.SystemIDs()
		}
		var override *config.RoutePrefs
		if in.Prefs != nil {
			rp, err := p.resolvePrefs(ctx, *in.Prefs)
			if err != nil {
				return nil, PlanJumpRouteOutput{}, err
			}
			override = &rp
		}
		out, _, req, err := p.planRoute(ctx, prof, ends[0], ends[1], pinned, override)
		if err != nil {
			return nil, out, err
		}
		hub, err := p.hub("")
		if err != nil {
			return nil, out, err
		}
		lines, err := p.quote(ctx, hub, []industry.TypeQty{{TypeID: req.Drive.FuelTypeID, Quantity: max(out.Routes[0].Fuel, 1)}}, nil)
		if err != nil {
			out.Notes = append(out.Notes, "fuel price unavailable: "+err.Error())
		} else if len(lines) == 1 {
			out.FuelPrice = lines[0].Average
		}
		return nil, out, nil
	}
}

// --- hauling ---

type HaulItemInput struct {
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
	Dest     string `json:"dest,omitempty" jsonschema:"drop-off label, e.g. the facility profile name; default destination"`
}

type PlanHaulInput struct {
	HaulProfile string          `json:"haulProfile"`
	From        string          `json:"from" jsonschema:"pickup system, e.g. Jita"`
	To          string          `json:"to" jsonschema:"delivery system"`
	Items       []HaulItemInput `json:"items"`
	Hub         string          `json:"hub,omitempty" jsonschema:"hub profile for pricing the cargo and fuel"`
}

type LoadView struct {
	cargo.Load
	CargoValue float64 `json:"cargoValue"`
	Multibuy   string  `json:"multibuy" jsonschema:"this load's cargo plus round-trip fuel, for the in-game Multibuy"`
}

type PlanHaulOutput struct {
	Loads         []LoadView `json:"loads"`
	Route         nav.Route  `json:"outboundRoute"`
	ReturnRoute   *nav.Route `json:"returnRoute,omitempty"`
	FuelName      string     `json:"fuelName"`
	FuelPerTrip   int64      `json:"fuelPerTrip"`
	FuelM3PerTrip float64    `json:"fuelM3PerTrip"`
	FuelCostTrip  float64    `json:"fuelCostPerTrip"`
	TotalM3       float64    `json:"totalM3"`
	FreightPerM3  float64    `json:"freightPerM3" jsonschema:"fuel cost of one full round trip divided by the hold"`
	Notes         []string   `json:"notes,omitempty"`
}

// haulPlan routes a round trip and packs cargo.
func (p *planner) haulPlan(ctx context.Context, prof config.HaulProfile, hub config.HubProfile, from, to int64, items []cargo.Cargo, prices map[int64]float64) (PlanHaulOutput, error) {
	var out PlanHaulOutput
	cfg, err := p.store.Load()
	if err != nil {
		return out, err
	}
	var pinnedOut, pinnedBack []int64
	var missing []string
	savedRoute := func(name string) []int64 {
		if name == "" {
			return nil
		}
		r, ok := cfg.Route(name)
		if !ok {
			// Deleted with delete_route, or renamed in routes.yaml.
			missing = append(missing, fmt.Sprintf("haul profile %q pins the route %q, which isn't saved any more, so a route was searched instead", prof.Name, name))
			return nil
		}
		return r.SystemIDs()
	}
	pinnedOut = savedRoute(prof.OutboundRoute)
	pinnedBack = savedRoute(prof.ReturnRoute)
	outbound := pinnedRoute(pinnedOut, from, to)
	outR, g, req, err := p.planRoute(ctx, prof, from, to, outbound, nil)
	if err != nil {
		return out, fmt.Errorf("outbound route: %w", err)
	}
	out.Route, out.Notes = outR.Routes[0], append(outR.Notes, missing...)
	if len(pinnedOut) > 0 && outbound == nil {
		out.Notes = append(out.Notes, fmt.Sprintf("haul profile %q's pinned outbound route doesn't run from %s to %s, so a route was searched instead", prof.Name, g.Systems[from].Name, g.Systems[to].Name))
	}
	back := req
	back.From, back.To = to, from
	var ret nav.Route
	pinnedReturn := pinnedRoute(pinnedBack, to, from)
	if len(pinnedBack) > 0 && pinnedReturn == nil {
		out.Notes = append(out.Notes, fmt.Sprintf("haul profile %q's pinned return route doesn't run from %s to %s, so a route was searched instead", prof.Name, g.Systems[to].Name, g.Systems[from].Name))
	}
	if len(pinnedReturn) > 0 {
		ret, err = g.ValidateRoute(back, pinnedReturn)
	} else {
		var rs []nav.Route
		rs, err = g.PlanRoutes(back)
		if err == nil {
			ret = rs[0]
		}
	}
	if err != nil {
		out.Notes = append(out.Notes, "no return route found: "+err.Error())
	} else {
		out.ReturnRoute = &ret
	}
	out.FuelPerTrip = out.Route.Fuel
	if out.ReturnRoute != nil {
		out.FuelPerTrip += out.ReturnRoute.Fuel
	}
	fb, err := p.typeBasics(ctx, []int64{req.Drive.FuelTypeID})
	if err != nil {
		return out, err
	}
	out.FuelName = fb[req.Drive.FuelTypeID].Name
	out.FuelM3PerTrip = float64(out.FuelPerTrip) * fb[req.Drive.FuelTypeID].Volume
	lines, err := p.quote(ctx, hub, []industry.TypeQty{{TypeID: req.Drive.FuelTypeID, Quantity: max(out.FuelPerTrip, 1)}}, nil)
	if err != nil {
		out.Notes = append(out.Notes, "fuel price unavailable, so fuel cost and freight are 0: "+err.Error())
	} else if len(lines) == 1 {
		out.FuelCostTrip = lines[0].Average * float64(out.FuelPerTrip)
	}
	out.FreightPerM3 = out.FuelCostTrip / prof.HoldM3
	loads, err := cargo.PackLoads(items, prof.HoldM3)
	if err != nil {
		return out, err
	}
	for _, l := range loads {
		cargo.SortLoadItems(&l)
		lv := LoadView{Load: l}
		var ml []market.MultibuyLine
		for _, it := range l.Items {
			lv.CargoValue += prices[it.TypeID] * float64(it.Quantity)
			ml = append(ml, market.MultibuyLine{Name: it.Name, Quantity: it.Quantity})
		}
		ml = append(ml, market.MultibuyLine{Name: out.FuelName, Quantity: out.FuelPerTrip})
		lv.Multibuy = market.Multibuy(ml)
		out.TotalM3 += l.M3
		out.Loads = append(out.Loads, lv)
	}
	return out, nil
}

func planHaulHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[PlanHaulInput, PlanHaulOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in PlanHaulInput) (*mcpsdk.CallToolResult, PlanHaulOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		cfg, err := store.Load()
		if err != nil {
			return nil, PlanHaulOutput{}, err
		}
		prof, ok := cfg.HaulProfile(in.HaulProfile)
		if !ok {
			return nil, PlanHaulOutput{}, fmt.Errorf("no haul profile named %q", in.HaulProfile)
		}
		hub, err := p.hub(in.Hub)
		if err != nil {
			return nil, PlanHaulOutput{}, err
		}
		ends, err := p.systemIDs(ctx, []string{in.From, in.To})
		if err != nil {
			return nil, PlanHaulOutput{}, err
		}
		var items []ItemInput
		for _, it := range in.Items {
			items = append(items, ItemInput{Name: it.Name, Quantity: it.Quantity})
		}
		tq, err := p.resolveItems(ctx, items)
		if err != nil {
			return nil, PlanHaulOutput{}, err
		}
		lines, err := p.quote(ctx, hub, tq, nil)
		if err != nil {
			return nil, PlanHaulOutput{}, err
		}
		prices := map[int64]float64{}
		var loadItems []cargo.Cargo
		for i, l := range lines {
			prices[l.TypeID] = l.Average
			b, err := p.typeBasics(ctx, []int64{l.TypeID})
			if err != nil {
				return nil, PlanHaulOutput{}, err
			}
			dest := in.Items[i].Dest
			if dest == "" {
				dest = in.To
			}
			loadItems = append(loadItems, cargo.Cargo{TypeID: l.TypeID, Name: l.Name, Quantity: l.Quantity, UnitM3: b[l.TypeID].Volume, Portion: portionFor(b[l.TypeID]), Dest: dest})
		}
		out, err := p.haulPlan(ctx, prof, hub, ends[0], ends[1], loadItems, prices)
		return nil, out, err
	}
}

// pinnedRoute returns a saved pinned route when it runs from → to, or nil so
// the caller searches for one instead.
func pinnedRoute(route []int64, from, to int64) []int64 {
	if len(route) >= 2 && route[0] == from && route[len(route)-1] == to {
		return route
	}
	return nil
}

// portionFor keeps ore in whole reprocessing portions.
func portionFor(b sde.TypeBasic) int64 {
	if b.CategoryID == 25 && b.PortionSize > 1 {
		return b.PortionSize
	}
	return 1
}

// namedRoute builds a saved route from system names, in order.
func (p *planner) namedRoute(ctx context.Context, name string, systems []string) (config.Route, error) {
	ids, err := p.systemIDs(ctx, systems)
	if err != nil {
		return config.Route{}, err
	}
	if len(ids) < 2 {
		return config.Route{}, fmt.Errorf("route %q needs at least two systems", name)
	}
	r := config.Route{Name: strings.TrimSpace(name)}
	for _, id := range ids {
		r.Waypoints = append(r.Waypoints, config.RouteWaypoint{ID: id, Name: p.systems[id].Name})
	}
	return r, nil
}

// --- saved routes ---

// SaveRouteInput is save_route's input.
type SaveRouteInput struct {
	Name      string   `json:"name" jsonschema:"a name for the route, such as '6RCQ-V to Jita'"`
	Systems   []string `json:"systems" jsonschema:"the route's systems in order, origin to destination, as system names: every system a jump lands in, plus any gate path"`
	ShipClass string   `json:"shipClass,omitempty" jsonschema:"the kind of ship it's planned for, such as jump_freighter"`
	Notes     *string  `json:"notes,omitempty" jsonschema:"why the route goes this way, in the user's words; omit to keep the notes it has"`
}

// SaveRouteOutput is the saved route and every value the save changed.
type SaveRouteOutput struct {
	Route   config.Route    `json:"route"`
	Changes []config.Change `json:"changes"`
}

func saveRouteHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[SaveRouteInput, SaveRouteOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveRouteInput) (*mcpsdk.CallToolResult, SaveRouteOutput, error) {
		if strings.TrimSpace(in.Name) == "" {
			return nil, SaveRouteOutput{}, fmt.Errorf("save route: name is required")
		}
		p := newPlanner(version, clientID, store, sdeStore)
		r, err := p.namedRoute(ctx, in.Name, in.Systems)
		if err != nil {
			return nil, SaveRouteOutput{}, fmt.Errorf("save route: %w", err)
		}
		r.ShipClass = strings.TrimSpace(in.ShipClass)
		changes, err := store.Update(func(cfg *config.Config) error {
			if old, ok := cfg.Route(r.Name); ok {
				r.Name, r.Notes = old.Name, old.Notes
				if r.ShipClass == "" {
					r.ShipClass = old.ShipClass
				}
			}
			if in.Notes != nil {
				r.Notes = strings.TrimSpace(*in.Notes)
			}
			cfg.UpsertRoute(r)
			return nil
		})
		if err != nil {
			return nil, SaveRouteOutput{}, err
		}
		if changes == nil {
			changes = []config.Change{}
		}
		return nil, SaveRouteOutput{Route: r, Changes: changes}, nil
	}
}

// ListRoutesOutput is list_routes' result.
type ListRoutesOutput struct {
	Routes []config.Route `json:"routes"`
}

func listRoutesHandler(store *config.Store) mcpsdk.ToolHandlerFor[struct{}, ListRoutesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, ListRoutesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListRoutesOutput{}, err
		}
		out := ListRoutesOutput{Routes: cfg.Routes}
		if out.Routes == nil {
			out.Routes = []config.Route{}
		}
		return nil, out, nil
	}
}
