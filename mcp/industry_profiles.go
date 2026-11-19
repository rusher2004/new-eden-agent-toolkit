package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// --- facility profiles ---

type SaveFacilityProfileInput struct {
	Name                string   `json:"name" jsonschema:"a short name to refer to this facility by, e.g. Home Sotiyo"`
	Kind                string   `json:"kind" jsonschema:"npc_station, upwell or starbase"`
	StationID           int64    `json:"stationId,omitempty" jsonschema:"NPC station ID, or the Upwell structure's ID (optional for Upwell)"`
	StructureType       string   `json:"structureType,omitempty" jsonschema:"Upwell structure type name (e.g. Sotiyo, Tatara) or starbase array type name"`
	System              string   `json:"system,omitempty" jsonschema:"solar system name; not needed for NPC stations"`
	Rigs                []string `json:"rigs,omitempty" jsonschema:"fitted rig type names, e.g. Standup XL-Set Ship Manufacturing Efficiency I"`
	Services            []string `json:"services,omitempty" jsonschema:"fitted services: manufacturing, capital_shipyard, supercapital_shipyard, research, invention, reprocessing, biochemical_reactor, composite_reactor, hybrid_reactor"`
	SupercapUpgrade     bool     `json:"supercapUpgrade,omitempty" jsonschema:"the system has the Supercapital Construction Facilities sovereignty upgrade"`
	FacilityTax         float64  `json:"facilityTax,omitempty" jsonschema:"owner-set industry tax as a fraction, e.g. 0.01 for 1%"`
	ReprocessingTax     float64  `json:"reprocessingTax,omitempty" jsonschema:"owner-set reprocessing tax as a fraction"`
	ReprocessingRigTier int      `json:"reprocessingRigTier,omitempty" jsonschema:"reprocessing rig: 0 none, 1 T1 (e.g. Reprocessing Monitor I), 2 T2"`
	Standing            float64  `json:"standing,omitempty" jsonschema:"NPC stations only: the higher of personal and corporation standing with the station owner"`
	Notes               string   `json:"notes,omitempty"`
}

type FacilityModifierView struct {
	Activity string  `json:"activity"`
	Kind     string  `json:"kind"`
	Value    float64 `json:"multiplier"`
	Applies  string  `json:"appliesTo"`
	Source   string  `json:"source"`
}

type FacilityView struct {
	Profile    config.FacilityProfile `json:"profile"`
	SystemName string                 `json:"systemName"`
	Band       string                 `json:"securityBand"`
	Security   float64                `json:"displaySecurity"`
	Modifiers  []FacilityModifierView `json:"modifiers"`
	Warnings   []string               `json:"warnings,omitempty"`
	// Product-specific view, when a product was given.
	Product     string                `json:"product,omitempty"`
	Multipliers *industry.Multipliers `json:"multipliers,omitempty"`
	CanRun      map[string]string     `json:"canRun,omitempty"` // activity → "yes" or the reason it can't
}

func (p *planner) describeFacility(ctx context.Context, prof config.FacilityProfile, productTypeID int64) (FacilityView, error) {
	f, err := p.facility(ctx, prof)
	if err != nil {
		return FacilityView{}, err
	}
	sys, err := p.system(ctx, prof.SystemID)
	if err != nil {
		return FacilityView{}, err
	}
	v := FacilityView{Profile: prof, SystemName: sys.Name, Band: f.Band.String(), Security: industry.DisplaySecurity(sys.Security), Warnings: f.Warnings}
	for _, m := range f.Modifiers {
		applies := "everything"
		if m.FilterID != 0 {
			applies = f.Filters[m.FilterID].Name
		}
		v.Modifiers = append(v.Modifiers, FacilityModifierView{Activity: m.Activity, Kind: m.Kind, Value: m.Value, Applies: applies, Source: m.Source})
	}
	if productTypeID != 0 {
		b, err := p.typeBasics(ctx, []int64{productTypeID})
		if err != nil {
			return FacilityView{}, err
		}
		pb := b[productTypeID]
		v.Product = pb.Name
		r, ok, err := p.recipe(ctx, productTypeID)
		if err != nil {
			return FacilityView{}, err
		}
		act := industry.ActManufacturing
		if ok {
			act = r.Activity
		}
		m := f.Multipliers(act, pb.CategoryID, pb.GroupID)
		v.Multipliers = &m
		v.CanRun = map[string]string{}
		for _, a := range []string{industry.ActManufacturing, industry.ActReaction, industry.ActCopying, industry.ActInvention, industry.ActResearchMaterial} {
			if err := f.CanRun(a, industry.Product{Name: pb.Name, CategoryID: pb.CategoryID, GroupID: pb.GroupID}); err != nil {
				v.CanRun[a] = err.Error()
			} else {
				v.CanRun[a] = "yes"
			}
		}
	}
	return v, nil
}

func saveFacilityProfileHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[SaveFacilityProfileInput, FacilityView] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveFacilityProfileInput) (*mcpsdk.CallToolResult, FacilityView, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		prof := config.FacilityProfile{Name: strings.TrimSpace(in.Name), Kind: in.Kind, StationID: in.StationID, Services: in.Services,
			SupercapUpgrade: in.SupercapUpgrade, FacilityTax: in.FacilityTax, ReprocessingTax: in.ReprocessingTax,
			ReprocessingRigTier: in.ReprocessingRigTier, Standing: in.Standing, Notes: in.Notes}
		if prof.Name == "" {
			return nil, FacilityView{}, fmt.Errorf("save facility profile: name is required")
		}
		switch in.Kind {
		case industry.KindNPCStation, industry.KindUpwell, industry.KindStarbase:
		default:
			return nil, FacilityView{}, fmt.Errorf("save facility profile: kind must be npc_station, upwell or starbase, not %q", in.Kind)
		}
		for _, s := range in.Services {
			if !slices.Contains(industry.KnownServices, s) {
				return nil, FacilityView{}, fmt.Errorf("save facility profile: unknown service %q; known: %s", s, strings.Join(industry.KnownServices, ", "))
			}
		}
		if in.ReprocessingRigTier < 0 || in.ReprocessingRigTier > 2 {
			return nil, FacilityView{}, fmt.Errorf("save facility profile: reprocessingRigTier must be 0, 1 or 2")
		}
		if in.Kind == industry.KindNPCStation {
			st, ok, err := sdeStore.NPCStationIndustryData(ctx, p.ua(), in.StationID)
			if err != nil {
				return nil, FacilityView{}, err
			}
			if !ok {
				return nil, FacilityView{}, fmt.Errorf("save facility profile: %d isn't an NPC station", in.StationID)
			}
			prof.SystemID = st.SystemID
		} else {
			ids, err := p.systemIDs(ctx, []string{in.System})
			if err != nil {
				return nil, FacilityView{}, fmt.Errorf("save facility profile: %w", err)
			}
			prof.SystemID = ids[0]
		}
		var names []string
		if in.StructureType != "" {
			names = append(names, in.StructureType)
		}
		names = append(names, in.Rigs...)
		resolved, err := sdeStore.TypesByName(ctx, p.ua(), names)
		if err != nil {
			return nil, FacilityView{}, err
		}
		if in.StructureType != "" {
			id, ok := resolved[in.StructureType]
			if !ok {
				return nil, FacilityView{}, fmt.Errorf("save facility profile: unknown structure type %q", in.StructureType)
			}
			prof.StructureType = id
		} else if in.Kind == industry.KindUpwell {
			return nil, FacilityView{}, fmt.Errorf("save facility profile: structureType is required for Upwell structures")
		}
		for _, r := range in.Rigs {
			id, ok := resolved[r]
			if !ok {
				return nil, FacilityView{}, fmt.Errorf("save facility profile: unknown rig %q", r)
			}
			prof.RigTypeIDs = append(prof.RigTypeIDs, id)
		}
		view, err := p.describeFacility(ctx, prof, 0)
		if err != nil {
			return nil, FacilityView{}, err
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, FacilityView{}, err
		}
		cfg.UpsertFacilityProfile(prof)
		if err := store.Save(cfg); err != nil {
			return nil, FacilityView{}, err
		}
		return nil, view, nil
	}
}

type ProfileNameInput struct {
	Name string `json:"name" jsonschema:"the profile's name"`
}

type DescribeFacilityInput struct {
	Name        string `json:"name" jsonschema:"facility profile name"`
	ProductName string `json:"productName,omitempty" jsonschema:"optional product to show multipliers and whether each activity can run for it"`
}

func describeFacilityHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[DescribeFacilityInput, FacilityView] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DescribeFacilityInput) (*mcpsdk.CallToolResult, FacilityView, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		prof, err := p.facilityProfile(in.Name)
		if err != nil {
			return nil, FacilityView{}, err
		}
		var pid int64
		if in.ProductName != "" {
			ids, err := sdeStore.TypesByName(ctx, p.ua(), []string{in.ProductName})
			if err != nil {
				return nil, FacilityView{}, err
			}
			if pid = ids[in.ProductName]; pid == 0 {
				return nil, FacilityView{}, fmt.Errorf("describe facility: unknown product %q", in.ProductName)
			}
		}
		v, err := p.describeFacility(ctx, prof, pid)
		return nil, v, err
	}
}

func (p *planner) facilityProfile(name string) (config.FacilityProfile, error) {
	cfg, err := p.store.Load()
	if err != nil {
		return config.FacilityProfile{}, err
	}
	prof, ok := cfg.FacilityProfile(name)
	if !ok {
		var names []string
		for _, f := range cfg.FacilityProfiles {
			names = append(names, f.Name)
		}
		return config.FacilityProfile{}, fmt.Errorf("no facility profile named %q (saved: %s)", name, strings.Join(names, ", "))
	}
	return prof, nil
}

type ListFacilityProfilesOutput struct {
	Profiles []config.FacilityProfile `json:"profiles"`
}

func listFacilityProfilesHandler(store *config.Store) mcpsdk.ToolHandlerFor[struct{}, ListFacilityProfilesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, ListFacilityProfilesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListFacilityProfilesOutput{}, err
		}
		return nil, ListFacilityProfilesOutput{Profiles: cfg.FacilityProfiles}, nil
	}
}

type DeletedOutput struct {
	Deleted string `json:"deleted"`
}

func deleteProfileHandler(store *config.Store, del func(*config.Config, string) error) mcpsdk.ToolHandlerFor[ProfileNameInput, DeletedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ProfileNameInput) (*mcpsdk.CallToolResult, DeletedOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, DeletedOutput{}, err
		}
		if err := del(&cfg, in.Name); err != nil {
			return nil, DeletedOutput{}, err
		}
		if err := store.Save(cfg); err != nil {
			return nil, DeletedOutput{}, err
		}
		return nil, DeletedOutput{Deleted: in.Name}, nil
	}
}

// --- calibration ---

type CalibrateFacilityProfileInput struct {
	Name          string `json:"name" jsonschema:"facility profile name"`
	CharacterID   int32  `json:"characterId" jsonschema:"a logged-in character that can read the job"`
	CorporationID int64  `json:"corporationId,omitempty" jsonschema:"set to read a corporation job; omit for the character's own jobs"`
	JobID         int64  `json:"jobId" jsonschema:"a manufacturing or reaction job installed at this facility"`
}

type CalibrateFacilityProfileOutput struct {
	Profile        config.FacilityProfile `json:"profile"`
	EstimatedValue float64                `json:"estimatedItemValue"`
	ChargedFee     float64                `json:"chargedFee"`
	FeeRate        float64                `json:"feeRate"`
	EstimatedRate  float64                `json:"estimatedRateFromCostIndex"`
}

func calibrateFacilityProfileHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[CalibrateFacilityProfileInput, CalibrateFacilityProfileOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CalibrateFacilityProfileInput) (*mcpsdk.CallToolResult, CalibrateFacilityProfileOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		if err := p.requireCharacter(in.CharacterID); err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		prof, err := p.facilityProfile(in.Name)
		if err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		job, err := p.findJob(ctx, in.CharacterID, in.CorporationID, in.JobID)
		if err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		if job.Cost == nil || *job.Cost == 0 {
			return nil, CalibrateFacilityProfileOutput{}, fmt.Errorf("calibrate: job %d has no recorded cost", in.JobID)
		}
		bp, ok, err := sdeStore.Blueprint(ctx, p.ua(), job.BlueprintTypeID)
		if err != nil || !ok {
			return nil, CalibrateFacilityProfileOutput{}, fmt.Errorf("calibrate: blueprint %d not in the SDE: %v", job.BlueprintTypeID, err)
		}
		act, ok := bp.Activity(job.ActivityID)
		if !ok {
			return nil, CalibrateFacilityProfileOutput{}, fmt.Errorf("calibrate: blueprint %d has no activity %d", job.BlueprintTypeID, job.ActivityID)
		}
		prices, err := p.adjustedPrices(ctx)
		if err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		mats := map[int64]int64{}
		for _, m := range act.Materials {
			mats[m.TypeID] = m.Quantity
		}
		eiv := industry.EstimatedItemValue(mats, prices, job.Runs)
		rate := industry.CalibrateFeeRate(eiv, *job.Cost)
		out := CalibrateFacilityProfileOutput{EstimatedValue: eiv, ChargedFee: *job.Cost, FeeRate: rate}
		f, err := p.facility(ctx, prof)
		if err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		activity := industry.ActManufacturing
		if job.ActivityID == sde.ActivityReactions {
			activity = industry.ActReaction
		}
		ci, err := p.costIndex(ctx, prof.SystemID, activity)
		if err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		m := f.Multipliers(activity, 0, 0)
		out.EstimatedRate = industry.JobFee(industry.FeeInput{BaseValue: 1, CostIndex: ci, StructureCostBonus: m.Cost, FacilityTax: f.FacilityTax})
		prof.Calibration = &config.Calibration{FeeRate: rate, FromJobID: in.JobID, ObservedAt: time.Now().UTC().Format(time.RFC3339)}
		cfg, err := store.Load()
		if err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		cfg.UpsertFacilityProfile(prof)
		if err := store.Save(cfg); err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		out.Profile = prof
		return nil, out, nil
	}
}

// findJob looks a job up in a character's or corporation's job list,
// including completed jobs.
func (p *planner) findJob(ctx context.Context, characterID int32, corporationID, jobID int64) (IndustryJob, error) {
	if corporationID != 0 {
		raw, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]IndustryJob, int32, error) {
			jobs, total, err := listCorporationIndustryJobs(ctx, p.clientID, characterID, p.ua(), corporationID, true, page)
			out := make([]IndustryJob, len(jobs))
			for i, j := range jobs {
				out[i] = toIndustryJobFromCorporation(j)
			}
			return out, total, err
		})
		if err != nil {
			return IndustryJob{}, fmt.Errorf("corporation jobs: %w", err)
		}
		for _, j := range raw {
			if j.JobID == jobID {
				return j, nil
			}
		}
	} else {
		jobs, err := listCharacterIndustryJobs(ctx, p.clientID, characterID, p.ua(), true)
		if err != nil {
			return IndustryJob{}, fmt.Errorf("character jobs: %w", err)
		}
		for _, j := range jobs {
			if j.JobId == jobID {
				return toIndustryJobFromCharacter(j), nil
			}
		}
	}
	return IndustryJob{}, fmt.Errorf("job %d not found", jobID)
}

// --- haul profiles ---

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
	Name     string          `json:"name"`
	Ship     string          `json:"ship" jsonschema:"hull type name, e.g. Nomad"`
	PilotID  int64           `json:"pilotId" jsonschema:"the pilot's character ID (skills and contacts are read from ESI)"`
	HoldM3   float64         `json:"holdM3" jsonschema:"the fitted cargo hold in m³ as shown in game"`
	Outbound []string        `json:"outbound,omitempty" jsonschema:"optional pinned route, hub to destination, as system names"`
	Return   []string        `json:"return,omitempty" jsonschema:"optional pinned return route as system names"`
	Prefs    RoutePrefsInput `json:"prefs"`
}

func (p *planner) resolvePrefs(ctx context.Context, in RoutePrefsInput) (config.RoutePrefs, error) {
	out := config.RoutePrefs{AvoidAlliances: in.AvoidAlliances, AvoidBadStanding: in.AvoidBadStanding, PreferAlliances: in.PreferAlliances,
		MidpointsRequired: in.MidpointsRequired, PreferStationSystems: in.PreferStationSystems, AvoidIncursions: in.AvoidIncursions,
		Optimize: in.Optimize, MaxGates: in.MaxGates, MaxRecentKills: in.MaxRecentKills}
	switch in.Optimize {
	case "", industry.OptimizeFuel, industry.OptimizeJumps, industry.OptimizeFatigue:
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
		prof := config.HaulProfile{Name: strings.TrimSpace(in.Name), ShipTypeID: ship, PilotID: in.PilotID, HoldM3: in.HoldM3, Prefs: prefs}
		if prof.Outbound, err = p.systemIDs(ctx, in.Outbound); err != nil {
			return nil, config.HaulProfile{}, fmt.Errorf("save haul profile: %w", err)
		}
		if prof.Return, err = p.systemIDs(ctx, in.Return); err != nil {
			return nil, config.HaulProfile{}, fmt.Errorf("save haul profile: %w", err)
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, config.HaulProfile{}, err
		}
		cfg.UpsertHaulProfile(prof)
		if err := store.Save(cfg); err != nil {
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

type SaveHubProfileInput struct {
	Name      string `json:"name"`
	RegionID  int64  `json:"regionId" jsonschema:"region whose market to read, e.g. 10000002 for The Forge"`
	StationID int64  `json:"stationId" jsonschema:"station or structure orders must be at, e.g. 60003760 for Jita 4-4"`
}

type ListHubProfilesOutput struct {
	Profiles []config.HubProfile `json:"profiles"`
	Default  config.HubProfile   `json:"default"`
}

func saveHubProfileHandler(store *config.Store) mcpsdk.ToolHandlerFor[SaveHubProfileInput, config.HubProfile] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveHubProfileInput) (*mcpsdk.CallToolResult, config.HubProfile, error) {
		if strings.TrimSpace(in.Name) == "" || in.RegionID == 0 || in.StationID == 0 {
			return nil, config.HubProfile{}, fmt.Errorf("save hub profile: name, regionId and stationId are required")
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, config.HubProfile{}, err
		}
		h := config.HubProfile{Name: strings.TrimSpace(in.Name), RegionID: in.RegionID, StationID: in.StationID}
		cfg.UpsertHubProfile(h)
		if err := store.Save(cfg); err != nil {
			return nil, config.HubProfile{}, err
		}
		return nil, h, nil
	}
}

func listHubProfilesHandler(store *config.Store) mcpsdk.ToolHandlerFor[struct{}, ListHubProfilesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, ListHubProfilesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListHubProfilesOutput{}, err
		}
		return nil, ListHubProfilesOutput{Profiles: cfg.HubProfiles, Default: config.DefaultHub}, nil
	}
}
