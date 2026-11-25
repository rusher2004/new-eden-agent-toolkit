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
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

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
	Notes               *string  `json:"notes,omitempty" jsonschema:"what the user says about this facility, such as how a tax was measured; omit to keep the notes it has"`
}

type FacilityModifierView struct {
	Activity string  `json:"activity"`
	Kind     string  `json:"kind"`
	Value    float64 `json:"multiplier"`
	Applies  string  `json:"appliesTo"`
	Source   string  `json:"source"`
}

type FacilityView struct {
	Profile       config.FacilityProfile `json:"profile"`
	SystemName    string                 `json:"systemName"`
	Band          string                 `json:"securityBand"`
	Security      float64                `json:"displaySecurity"`
	SecurityColor string                 `json:"securityColor"`
	Modifiers     []FacilityModifierView `json:"modifiers"`
	Warnings      []string               `json:"warnings,omitempty"`
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
	v := FacilityView{Profile: prof, SystemName: sys.Name, Band: f.Band.String(), Security: nav.DisplaySecurity(sys.Security), SecurityColor: nav.SecurityColor(sys.Security), Warnings: f.Warnings}
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

// AddSaveFacilityProfile registers save_facility_profile. openESI and openPublicESI build the ESI clients;
// any client with the methods the planner and this tool call will do.
func AddSaveFacilityProfile[C plannerClient, P plannerPublicClient](s *mcpsdk.Server, userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "save_facility_profile",
		Description: "Save (or replace) a named industry or reprocessing facility profile in neat's local config: NPC station, Upwell structure or starbase, its system, fitted rigs and services, and owner-set taxes. Rig and structure bonuses are read from the SDE; the response shows every resolved multiplier. Writes to ~/.neat/facilities.yaml only; omit notes to keep the notes it has.",
	}, saveFacilityProfileHandler(userAgent, openESI, openPublicESI, store, sdeStore))
}

func saveFacilityProfileHandler[C plannerClient, P plannerPublicClient](userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) mcpsdk.ToolHandlerFor[SaveFacilityProfileInput, FacilityView] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveFacilityProfileInput) (*mcpsdk.CallToolResult, FacilityView, error) {
		p := newPlanner(userAgent, plannerOpener(openESI), plannerPublicOpener(openPublicESI), store, sdeStore)
		prof := config.FacilityProfile{Name: strings.TrimSpace(in.Name), Kind: in.Kind, StationID: in.StationID, Services: in.Services,
			SupercapUpgrade: in.SupercapUpgrade, FacilityTax: in.FacilityTax, ReprocessingTax: in.ReprocessingTax,
			ReprocessingRigTier: in.ReprocessingRigTier, Standing: in.Standing}
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
		if sys, err := p.system(ctx, prof.SystemID); err == nil {
			prof.SystemName = sys.Name
		}
		if _, err := store.Update(func(cfg *config.Config) error {
			if old, ok := cfg.FacilityProfile(prof.Name); ok {
				prof.Name, prof.Notes = old.Name, old.Notes
			}
			if in.Notes != nil {
				prof.Notes = strings.TrimSpace(*in.Notes)
			}
			cfg.UpsertFacilityProfile(prof)
			return nil
		}); err != nil {
			return nil, FacilityView{}, err
		}
		return nil, view, nil
	}
}

type DescribeFacilityInput struct {
	Name        string `json:"name" jsonschema:"facility profile name"`
	ProductName string `json:"productName,omitempty" jsonschema:"optional product to show multipliers and whether each activity can run for it"`
}

// AddDescribeFacility registers describe_facility. openESI and openPublicESI build the ESI clients;
// any client with the methods the planner and this tool call will do.
func AddDescribeFacility[C plannerClient, P plannerPublicClient](s *mcpsdk.Server, userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "describe_facility",
		Description: "Show a facility profile's security band and every material, time and cost multiplier with its source, and optionally, for a product, the combined multipliers and whether each activity can run there.",
	}, describeFacilityHandler(userAgent, openESI, openPublicESI, store, sdeStore))
}

func describeFacilityHandler[C plannerClient, P plannerPublicClient](userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) mcpsdk.ToolHandlerFor[DescribeFacilityInput, FacilityView] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DescribeFacilityInput) (*mcpsdk.CallToolResult, FacilityView, error) {
		p := newPlanner(userAgent, plannerOpener(openESI), plannerPublicOpener(openPublicESI), store, sdeStore)
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

// AddListFacilityProfiles registers list_facility_profiles.
func AddListFacilityProfiles(s *mcpsdk.Server, store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_facility_profiles",
		Description: "List saved facility profiles.",
	}, listFacilityProfilesHandler(store))
}

// AddDeleteFacilityProfile registers delete_facility_profile.
func AddDeleteFacilityProfile(s *mcpsdk.Server, store configUpdater) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_facility_profile",
		Description: "Delete a saved facility profile from neat's local config.",
	}, deleteProfileHandler(store, (*config.Config).DeleteFacilityProfile))
}

func listFacilityProfilesHandler(store configLoader) mcpsdk.ToolHandlerFor[struct{}, ListFacilityProfilesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, ListFacilityProfilesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListFacilityProfilesOutput{}, err
		}
		return nil, ListFacilityProfilesOutput{Profiles: cfg.FacilityProfiles}, nil
	}
}

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

// AddCalibrateFacilityProfile registers calibrate_facility_profile. openESI and openPublicESI build the ESI clients;
// any client with the methods the planner and this tool call will do.
func AddCalibrateFacilityProfile[C plannerClient, P plannerPublicClient](s *mcpsdk.Server, userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "calibrate_facility_profile",
		Description: "Read a real industry job's charged fee, back out the effective fee rate, and save it on a facility profile so later estimates use it. Writes to ~/.neat/facilities.yaml only.",
	}, calibrateFacilityProfileHandler(userAgent, openESI, openPublicESI, store, sdeStore))
}

func calibrateFacilityProfileHandler[C plannerClient, P plannerPublicClient](userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) mcpsdk.ToolHandlerFor[CalibrateFacilityProfileInput, CalibrateFacilityProfileOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CalibrateFacilityProfileInput) (*mcpsdk.CallToolResult, CalibrateFacilityProfileOutput, error) {
		p := newPlanner(userAgent, plannerOpener(openESI), plannerPublicOpener(openPublicESI), store, sdeStore)
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
		// Set only the calibration on the profile as it is now, so an edit
		// saved while the job was being looked up isn't undone.
		if _, err := store.Update(func(cfg *config.Config) error {
			cur, ok := cfg.FacilityProfile(prof.Name)
			if !ok {
				return fmt.Errorf("no facility profile named %q", prof.Name)
			}
			cur.Calibration = prof.Calibration
			prof = cur
			cfg.UpsertFacilityProfile(cur)
			return nil
		}); err != nil {
			return nil, CalibrateFacilityProfileOutput{}, err
		}
		out.Profile = prof
		return nil, out, nil
	}
}

// findJob looks a job up in a character's or corporation's job list,
// including completed jobs.
func (p *planner) findJob(ctx context.Context, characterID int32, corporationID, jobID int64) (IndustryJob, error) {
	client, err := p.esi(ctx, characterID)
	if corporationID != 0 {
		if err != nil {
			return IndustryJob{}, fmt.Errorf("corporation jobs: %w", err)
		}
		raw, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]IndustryJob, int32, error) {
			jobs, total, err := listCorporationIndustryJobs(ctx, client, corporationID, true, page)
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
		if err != nil {
			return IndustryJob{}, fmt.Errorf("character jobs: %w", err)
		}
		jobs, err := listCharacterIndustryJobs(ctx, client, true)
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
