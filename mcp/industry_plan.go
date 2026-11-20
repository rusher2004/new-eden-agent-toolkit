package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/market"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

type PlanBuildInput struct {
	CharacterID      int32             `json:"characterId" jsonschema:"logged-in character; with corporationId, one with roles to read corp blueprints, assets and jobs"`
	CorporationID    int64             `json:"corporationId,omitempty" jsonschema:"use this corporation's blueprints, stock and jobs"`
	Product          string            `json:"product" jsonschema:"what to build, e.g. Minokawa"`
	BlueprintItemIDs []int64           `json:"blueprintItemIds,omitempty" jsonschema:"specific owned blueprints to run for the product; default every owned copy not in an active job"`
	Quantity         int64             `json:"quantity,omitempty" jsonschema:"units to build when running an original (or when you own no copies)"`
	Facility         string            `json:"facility" jsonschema:"facility profile for manufacturing"`
	ReactionFacility string            `json:"reactionFacility,omitempty" jsonschema:"facility profile for reactions, if any are built"`
	RefineFacility   string            `json:"refineFacility,omitempty" jsonschema:"facility profile where compressed ore is reprocessed"`
	RefinerID        int32             `json:"refinerId,omitempty" jsonschema:"character who reprocesses; default characterId"`
	BuilderIDs       []int32           `json:"builderIds,omitempty" jsonschema:"characters who install jobs; default characterId"`
	HaulProfile      string            `json:"haulProfile,omitempty" jsonschema:"haul profile for the trip from the hub; omit to skip hauling"`
	Hub              string            `json:"hub,omitempty" jsonschema:"hub profile; default Jita 4-4"`
	UseCompressedOre bool              `json:"useCompressedOre,omitempty" jsonschema:"buy compressed ore for minerals where it's cheaper (needs refineFacility)"`
	UseImplant       *bool             `json:"useImplant,omitempty" jsonschema:"count the refiner's reprocessing implant (default true)"`
	Decisions        map[string]string `json:"decisions,omitempty" jsonschema:"item name → build or buy, overriding the defaults"`
	StockLocationID  int64             `json:"stockLocationId,omitempty" jsonschema:"station or structure whose stock to use; default the build facility's structure"`
	StockDivision    string            `json:"stockDivision,omitempty" jsonschema:"only stock in this hangar flag, e.g. CorpSAG3"`
	IgnoreStock      bool              `json:"ignoreStock,omitempty"`
	FreightPerM3     float64           `json:"freightPerM3,omitempty" jsonschema:"ISK per m³ when there's no haul profile"`
}

type CostLine struct {
	Item   string  `json:"item"`
	Detail string  `json:"detail,omitempty"`
	ISK    float64 `json:"isk"`
	M3     float64 `json:"m3,omitempty"`
}

type DecisionLine struct {
	Name      string `json:"name"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason"`
	Demand    int64  `json:"demand"`
	FromStock int64  `json:"fromStock,omitempty"`
	Built     int64  `json:"built,omitempty"`
	Bought    int64  `json:"bought,omitempty"`
	Surplus   int64  `json:"surplus,omitempty"`
}

type BuildBuyView struct {
	Name string `json:"name"`
	industry.BuildBuy
}

type JobGroup struct {
	Product     string  `json:"product"`
	Activity    string  `json:"activity"`
	Jobs        int     `json:"jobs"`
	Runs        int64   `json:"runs"`
	Output      int64   `json:"output"`
	CopiesUsed  int     `json:"copiesUsed"`
	Fees        float64 `json:"fees"`
	HoursPerJob float64 `json:"longestJobHours"`
	OutputM3    float64 `json:"outputM3"`
}

type StockUse struct {
	Name     string  `json:"name"`
	Quantity int64   `json:"quantity"`
	Value    float64 `json:"value"`
	M3       float64 `json:"m3"`
}

type PlanBuildOutput struct {
	Product     string             `json:"product"`
	Units       int64              `json:"units"`
	TotalCost   float64            `json:"totalCost"`
	PerUnit     float64            `json:"perUnit"`
	CargoM3     float64            `json:"cargoM3"`
	Cost        []CostLine         `json:"cost"`
	Decisions   []DecisionLine     `json:"decisions"`
	BuildVsBuy  []BuildBuyView     `json:"buildVsBuy,omitempty"`
	Shopping    []QuoteLine        `json:"shopping"`
	Multibuy    string             `json:"multibuy" jsonschema:"every purchase, for the in-game Multibuy; per-load lists are under haul"`
	OreMix      *SolveOreMixOutput `json:"oreMix,omitempty"`
	ShuttleM3   float64            `json:"refinedMineralsToShuttleM3,omitempty"`
	StockUsed   []StockUse         `json:"stockUsed,omitempty"`
	Haul        *PlanHaulOutput    `json:"haul,omitempty"`
	Jobs        []JobGroup         `json:"jobs"`
	Schedule    ScheduleSummary    `json:"schedule"`
	Warnings    []string           `json:"warnings,omitempty"`
	Assumptions []string           `json:"assumptions"`
	PlanPath    string             `json:"planPath"`
}

type ScheduleSummary struct {
	ComponentHours float64  `json:"intermediateHours" jsonschema:"wall-clock hours for all non-final jobs if slots are refilled as they free"`
	FinalHours     float64  `json:"finalHours" jsonschema:"wall-clock hours for the final product's jobs"`
	SlotDays       float64  `json:"slotDays"`
	Unassigned     []string `json:"unassigned,omitempty"`
}

func planBuildHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[PlanBuildInput, PlanBuildOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in PlanBuildInput) (*mcpsdk.CallToolResult, PlanBuildOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		out, err := p.planBuild(ctx, in)
		if err != nil {
			return nil, PlanBuildOutput{}, fmt.Errorf("plan build: %w", err)
		}
		return nil, out, nil
	}
}

func (p *planner) planBuild(ctx context.Context, in PlanBuildInput) (PlanBuildOutput, error) {
	var out PlanBuildOutput
	pid, err := p.resolveType(ctx, in.Product)
	if err != nil {
		return out, err
	}
	rootRecipe, ok, err := p.recipe(ctx, pid)
	if err != nil {
		return out, err
	}
	if !ok {
		return out, fmt.Errorf("%s isn't manufactured or reacted", in.Product)
	}
	out.Product = p.typeName(ctx, pid)
	hub, err := p.hub(in.Hub)
	if err != nil {
		return out, err
	}

	// Facilities.
	buildProf, err := p.facilityProfile(in.Facility)
	if err != nil {
		return out, err
	}
	buildFac, err := p.facility(ctx, buildProf)
	if err != nil {
		return out, err
	}
	out.Warnings = append(out.Warnings, buildFac.Warnings...)
	var rxFac *industry.Facility
	var rxProf config.FacilityProfile
	if in.ReactionFacility != "" {
		if rxProf, err = p.facilityProfile(in.ReactionFacility); err != nil {
			return out, err
		}
		f, err := p.facility(ctx, rxProf)
		if err != nil {
			return out, err
		}
		rxFac = &f
		out.Warnings = append(out.Warnings, f.Warnings...)
	}
	mult := p.multipliersFor(ctx, &buildFac, rxFac)
	if b, err := p.typeBasics(ctx, []int64{pid}); err == nil {
		if err := buildFac.CanRun(rootRecipe.Activity, industry.Product{Name: out.Product, CategoryID: b[pid].CategoryID, GroupID: b[pid].GroupID}); err != nil {
			out.Warnings = append(out.Warnings, err.Error())
		}
	}

	// Ownership: copies, active jobs, stock.
	stockLoc := in.StockLocationID
	if stockLoc == 0 {
		stockLoc = buildProf.StationID
	}
	needAssets := !in.IgnoreStock && stockLoc != 0
	own, err := p.loadOwnership(ctx, in.CharacterID, in.CorporationID, needAssets)
	if err != nil {
		return out, err
	}
	var rootCopies []industry.Copy
	copies := map[int64][]industry.Copy{}
	for _, b := range own.blueprints {
		if b.InActiveJob {
			continue
		}
		c := industry.Copy{ItemID: b.ItemID, ME: b.ME, TE: b.TE, Runs: b.Runs}
		if b.BlueprintTypeID == rootRecipe.BlueprintTypeID && !b.Original && (len(in.BlueprintItemIDs) == 0 || slices.Contains(in.BlueprintItemIDs, b.ItemID)) {
			rootCopies = append(rootCopies, c)
			continue
		}
		copies[b.BlueprintTypeID] = append(copies[b.BlueprintTypeID], c)
	}
	if in.Quantity > 0 {
		// An explicit quantity runs from originals or copies via the normal
		// path, so the product's own copies go back into the pool it draws on.
		copies[rootRecipe.BlueprintTypeID] = append(copies[rootRecipe.BlueprintTypeID], rootCopies...)
		rootCopies = nil
	}
	if len(rootCopies) == 0 && in.Quantity <= 0 {
		return out, fmt.Errorf("no free copies of the %s blueprint found; pass quantity to plan from an original or without owned blueprints", out.Product)
	}
	decisions := map[int64]string{}
	if len(in.Decisions) > 0 {
		names := make([]string, 0, len(in.Decisions))
		for n := range in.Decisions {
			names = append(names, n)
		}
		ids, err := p.sde.TypesByName(ctx, p.ua(), names)
		if err != nil {
			return out, err
		}
		for n, d := range in.Decisions {
			id, ok := ids[n]
			if !ok {
				return out, fmt.Errorf("decisions: unknown item %q", n)
			}
			if d != industry.DecideBuild && d != industry.DecideBuy {
				return out, fmt.Errorf("decisions: %q must be build or buy", n)
			}
			decisions[id] = d
		}
	}
	stock := map[int64]int64{}
	if needAssets {
		stock = own.stockAt(stockLoc, in.StockDivision)
	}

	// Builders, so the tree only chooses to build what someone can run.
	builderIDs := in.BuilderIDs
	if len(builderIDs) == 0 {
		builderIDs = []int32{in.CharacterID}
	}
	var builders []industry.Builder
	for _, id := range builderIDs {
		b, err := p.builder(ctx, id, p.characterName(id))
		if err != nil {
			return out, err
		}
		builders = append(builders, b)
	}
	canBuild := func(r industry.Recipe) (bool, string) {
		for _, b := range builders {
			if industry.CanRunJob(b, r) {
				return true, ""
			}
		}
		return false, "none of the builders has the skills and a slot for it"
	}

	// Recipes are loaded on demand; ExpandTree needs a non-failing lookup.
	var recipeErr error
	recipeFn := func(t int64) (industry.Recipe, bool) {
		r, ok, err := p.recipe(ctx, t)
		if err != nil && recipeErr == nil {
			recipeErr = err
		}
		return r, ok
	}
	tree, err := industry.ExpandTree(industry.TreeRequest{ProductTypeID: pid, Quantity: in.Quantity, RootCopies: rootCopies,
		Recipe: recipeFn, Multipliers: mult, Copies: copies, Decisions: decisions, Stock: stock, CanBuild: canBuild})
	if err != nil {
		return out, err
	}
	if recipeErr != nil {
		return out, recipeErr
	}
	for _, j := range tree.Jobs {
		if j.ProductTypeID == pid {
			out.Units += j.Output
		}
	}

	// Names for everything in the tree.
	var ids []int64
	for _, n := range tree.Nodes {
		ids = append(ids, n.TypeID)
	}
	basics, err := p.typeBasics(ctx, ids)
	if err != nil {
		return out, err
	}
	for _, n := range tree.Nodes {
		out.Decisions = append(out.Decisions, DecisionLine{Name: basics[n.TypeID].Name, Decision: n.Decision, Reason: n.Reason,
			Demand: n.Demand, FromStock: n.FromStock, Built: n.Built, Bought: n.Bought, Surplus: n.Surplus})
	}

	// Freight rate: real fuel per m³ from the haul profile, or the flat rate.
	freight := in.FreightPerM3
	var haulProf config.HaulProfile
	var hubSystem int64
	if in.HaulProfile != "" {
		cfg, err := p.store.Load()
		if err != nil {
			return out, err
		}
		var ok bool
		if haulProf, ok = cfg.HaulProfile(in.HaulProfile); !ok {
			return out, fmt.Errorf("no haul profile named %q", in.HaulProfile)
		}
		st, ok, err := p.sde.NPCStationIndustryData(ctx, p.ua(), hub.StationID)
		if err != nil || !ok {
			return out, fmt.Errorf("hub station %d isn't an NPC station in the SDE: %v", hub.StationID, err)
		}
		hubSystem = st.SystemID
		pre, err := p.haulPlan(ctx, haulProf, hub, hubSystem, buildProf.SystemID, nil, nil)
		if err != nil {
			return out, err
		}
		freight = pre.FreightPerM3
	}

	// Purchases: minerals through the ore solver when asked, the rest quoted.
	var mineralNeeds = map[int64]int64{}
	var direct []industry.TypeQty
	buyBasics, err := p.typeBasics(ctx, typeIDs(tree.Buy))
	if err != nil {
		return out, err
	}
	for _, b := range tree.Buy {
		if in.UseCompressedOre && buyBasics[b.TypeID].GroupID == mineralGroupID {
			mineralNeeds[b.TypeID] += b.Quantity
		} else {
			direct = append(direct, b)
		}
	}
	if len(mineralNeeds) > 0 {
		if in.RefineFacility == "" {
			return out, fmt.Errorf("useCompressedOre needs refineFacility")
		}
		refProf, err := p.facilityProfile(in.RefineFacility)
		if err != nil {
			return out, err
		}
		refiner := in.RefinerID
		if refiner == 0 {
			refiner = in.CharacterID
		}
		mix, err := p.solveOreMix(ctx, mineralNeeds, refiner, refProf, hub, freight, in.UseImplant == nil || *in.UseImplant, false)
		if err != nil {
			return out, err
		}
		out.OreMix = &mix
		out.ShuttleM3 = mix.RefinedM3
		if refProf.SystemID != buildProf.SystemID || refProf.StationID != buildProf.StationID {
			out.Assumptions = append(out.Assumptions, fmt.Sprintf("refined minerals (%.0f m³) move from %s to %s", mix.RefinedM3, refProf.Name, buildProf.Name))
		}
	}
	allPriced := slices.Clone(direct)
	// Books for every input of every built job, for build-vs-buy.
	bookIDs := typeIDs(direct)
	for _, j := range tree.Jobs {
		bookIDs = append(bookIDs, j.ProductTypeID)
		for _, m := range j.Materials {
			bookIDs = append(bookIDs, m.TypeID)
		}
	}
	books, err := p.sellBooks(ctx, hub.RegionID, bookIDs)
	if err != nil {
		return out, err
	}
	shopping, err := p.quote(ctx, hub, allPriced, books)
	if err != nil {
		return out, err
	}
	out.Shopping = shopping
	prices := map[int64]float64{}
	for _, l := range shopping {
		prices[l.TypeID] = l.Average
		if l.Shortfall > 0 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %s sells only %d of the %d needed; the total cost leaves out the %d it can't supply", l.Name, hub.Name, l.Quantity-l.Shortfall, l.Quantity, l.Shortfall))
		}
	}

	// Fees and job groups.
	adj, err := p.adjustedPrices(ctx)
	if err != nil {
		return out, err
	}
	groups := map[int64]*JobGroup{}
	feesByProduct := map[int64]float64{}
	var totalFees float64
	for _, j := range tree.Jobs {
		r, _, err := p.recipe(ctx, j.ProductTypeID)
		if err != nil {
			return out, err
		}
		mats := map[int64]int64{}
		for _, m := range r.Materials {
			mats[m.TypeID] = m.Quantity
		}
		f, prof := buildFac, buildProf
		if j.Activity == industry.ActReaction && rxFac != nil {
			f, prof = *rxFac, rxProf
		}
		fee, err := p.jobFee(ctx, f, prof, j.Activity, j.ProductTypeID, industry.EstimatedItemValue(mats, adj, j.Runs), false)
		if err != nil {
			return out, err
		}
		totalFees += fee.Fee
		feesByProduct[j.ProductTypeID] += fee.Fee
		g := groups[j.ProductTypeID]
		if g == nil {
			pb, err := p.typeBasics(ctx, []int64{j.ProductTypeID})
			if err != nil {
				return out, err
			}
			g = &JobGroup{Product: pb[j.ProductTypeID].Name, Activity: j.Activity}
			groups[j.ProductTypeID] = g
		}
		g.Jobs++
		g.Runs += j.Runs
		g.Output += j.Output
		g.Fees += fee.Fee
		if j.CopyItemID != 0 {
			g.CopiesUsed++
		}
	}
	for _, id := range sortedKeys(groups) {
		b, err := p.typeBasics(ctx, []int64{id})
		if err != nil {
			return out, err
		}
		groups[id].OutputM3 = float64(groups[id].Output) * b[id].Volume
		out.Jobs = append(out.Jobs, *groups[id])
	}

	// Build vs buy for every intermediate that's built.
	for _, n := range tree.Nodes {
		if n.Level == 0 || n.Built == 0 {
			continue
		}
		var inputCost, inputM3 float64
		for _, j := range tree.Jobs {
			if j.ProductTypeID != n.TypeID {
				continue
			}
			for _, m := range j.Materials {
				q := market.WalkBook(m.TypeID, books[m.TypeID], m.Quantity, hub.StationID)
				mb, err := p.typeBasics(ctx, []int64{m.TypeID})
				if err != nil {
					return out, err
				}
				inputCost += q.Average * float64(m.Quantity)
				inputM3 += mb[m.TypeID].Volume * float64(m.Quantity)
			}
		}
		units := n.Built
		bq := market.WalkBook(n.TypeID, books[n.TypeID], units, hub.StationID)
		bb := industry.CompareBuildVsBuy(industry.BuildBuyInput{TypeID: n.TypeID, Units: units, InputCost: inputCost, InputM3: inputM3,
			JobFees: feesByProduct[n.TypeID], Buy: bq, UnitM3: basics[n.TypeID].Volume, FreightPerM3: freight, CanBuild: true})
		out.BuildVsBuy = append(out.BuildVsBuy, BuildBuyView{Name: basics[n.TypeID].Name, BuildBuy: bb})
	}
	sort.Slice(out.BuildVsBuy, func(i, j int) bool { return out.BuildVsBuy[i].SavedByBuilding > out.BuildVsBuy[j].SavedByBuilding })

	// Stock used, valued at hub prices.
	if len(tree.StockUsed) > 0 {
		sb, err := p.typeBasics(ctx, typeIDs(tree.StockUsed))
		if err != nil {
			return out, err
		}
		sq, err := p.quote(ctx, hub, tree.StockUsed, books)
		if err != nil {
			return out, err
		}
		for i, s := range tree.StockUsed {
			out.StockUsed = append(out.StockUsed, StockUse{Name: sb[s.TypeID].Name, Quantity: s.Quantity, Value: sq[i].Total, M3: sb[s.TypeID].Volume * float64(s.Quantity)})
		}
	}

	// Cost lines.
	var shopTotal, shopM3 float64
	for _, l := range shopping {
		shopTotal += l.Total
		shopM3 += l.M3
	}
	out.Cost = append(out.Cost, CostLine{Item: "Purchases at " + hub.Name, Detail: fmt.Sprintf("%d items", len(shopping)), ISK: shopTotal, M3: shopM3})
	if out.OreMix != nil {
		var oreISK, oreM3, minISK, minM3 float64
		for _, o := range out.OreMix.Ores {
			oreISK += o.Cost
			oreM3 += o.M3
		}
		for _, m := range out.OreMix.Minerals {
			minISK += m.Cost
			minM3 += m.M3
		}
		out.Cost = append(out.Cost, CostLine{Item: "Compressed ore", Detail: "reprocessed at " + in.RefineFacility, ISK: oreISK, M3: oreM3})
		if minISK > 0 {
			out.Cost = append(out.Cost, CostLine{Item: "Minerals bought refined", ISK: minISK, M3: minM3})
		}
	}
	out.Cost = append(out.Cost, CostLine{Item: "Job fees", Detail: fmt.Sprintf("%d jobs", len(tree.Jobs)), ISK: totalFees})

	// Haul.
	if in.HaulProfile != "" {
		var items []cargo.Cargo
		destBuild := buildProf.Name
		for _, l := range shopping {
			b := buyBasics[l.TypeID]
			items = append(items, cargo.Cargo{TypeID: l.TypeID, Name: l.Name, Quantity: l.Quantity, UnitM3: b.Volume, Dest: destBuild})
		}
		if out.OreMix != nil {
			ob, err := p.typeBasics(ctx, oreIDs(out.OreMix.OreMix))
			if err != nil {
				return out, err
			}
			for _, o := range out.OreMix.Ores {
				items = append(items, cargo.Cargo{TypeID: o.TypeID, Name: o.Name, Quantity: o.Quantity, UnitM3: ob[o.TypeID].Volume, Portion: portionFor(ob[o.TypeID]), Dest: in.RefineFacility})
				prices[o.TypeID] = o.Cost / float64(o.Quantity)
			}
			for _, m := range out.OreMix.Minerals {
				items = append(items, cargo.Cargo{TypeID: m.TypeID, Name: m.Name, Quantity: m.Quantity, UnitM3: 0.01, Dest: destBuild})
				prices[m.TypeID] = m.Cost / float64(m.Quantity)
			}
		}
		h, err := p.haulPlan(ctx, haulProf, hub, hubSystem, buildProf.SystemID, items, prices)
		if err != nil {
			return out, err
		}
		out.Haul = &h
		out.CargoM3 = h.TotalM3
		out.Cost = append(out.Cost, CostLine{Item: "Fuel", Detail: fmt.Sprintf("%d round trips × %d %s", len(h.Loads), h.FuelPerTrip, h.FuelName), ISK: h.FuelCostTrip * float64(len(h.Loads)), M3: h.FuelM3PerTrip * float64(len(h.Loads))})
	} else if freight > 0 {
		m3 := shopM3
		if out.OreMix != nil {
			m3 += out.OreMix.TotalM3
		}
		out.CargoM3 = m3
		out.Cost = append(out.Cost, CostLine{Item: "Freight", Detail: fmt.Sprintf("%.0f ISK/m³", freight), ISK: m3 * freight, M3: m3})
	}
	for _, c := range out.Cost {
		out.TotalCost += c.ISK
	}
	if out.Units > 0 {
		out.PerUnit = out.TotalCost / float64(out.Units)
	}

	// Multibuy for everything bought.
	var ml []market.MultibuyLine
	for _, l := range shopping {
		ml = append(ml, market.MultibuyLine{Name: l.Name, Quantity: l.Quantity})
	}
	if out.OreMix != nil {
		for _, o := range out.OreMix.Ores {
			ml = append(ml, market.MultibuyLine{Name: o.Name, Quantity: o.Quantity})
		}
		for _, m := range out.OreMix.Minerals {
			ml = append(ml, market.MultibuyLine{Name: m.Name, Quantity: m.Quantity})
		}
	}
	out.Multibuy = market.Multibuy(ml)

	// Schedule: intermediates first, then the final product.
	var inter, final []industry.TreeJob
	for _, j := range tree.Jobs {
		if j.ProductTypeID == pid {
			final = append(final, j)
		} else {
			inter = append(inter, j)
		}
	}
	si := industry.ScheduleJobs(inter, builders)
	sf := industry.ScheduleJobs(final, builders)
	out.Schedule = ScheduleSummary{ComponentHours: si.MakespanHrs, FinalHours: sf.MakespanHrs, SlotDays: si.SlotDays + sf.SlotDays}
	out.Schedule.Unassigned = append(si.Reasons, sf.Reasons...)
	for _, s := range append(si.Jobs, sf.Jobs...) {
		for i := range out.Jobs {
			if out.Jobs[i].Product == p.typeName(ctx, s.Job.ProductTypeID) && s.Hours > out.Jobs[i].HoursPerJob {
				out.Jobs[i].HoursPerJob = s.Hours
			}
		}
	}

	out.Assumptions = append(out.Assumptions,
		fmt.Sprintf("prices walk the live %s sell order books for the full quantity", hub.Name),
		"material quantities use the game's per-job rounding with your copies' ME and the facility's SDE bonuses",
		"job fees use the system cost index, structure bonus, facility tax and 4% SCC surcharge, or the profile's calibrated rate",
	)
	if freight > 0 {
		out.Assumptions = append(out.Assumptions, fmt.Sprintf("freight is %.0f ISK/m³ (one round trip's fuel ÷ the hold)", freight))
	}
	path, err := p.savePlan(out)
	if err != nil {
		out.Warnings = append(out.Warnings, "plan not saved: "+err.Error())
	}
	out.PlanPath = path
	return out, nil
}

func typeIDs(list []industry.TypeQty) []int64 {
	out := make([]int64, len(list))
	for i, t := range list {
		out[i] = t.TypeID
	}
	return out
}

func oreIDs(m industry.OreMix) []int64 {
	var out []int64
	for _, o := range m.Ores {
		out = append(out, o.TypeID)
	}
	return out
}

// savePlan writes the full plan next to the config file, under plans/.
func (p *planner) savePlan(out PlanBuildOutput) (string, error) {
	dir := filepath.Join(filepath.Dir(p.store.Path()), "plans")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create plans directory: %w", err)
	}
	name := strings.ToLower(strings.Map(func(r rune) rune {
		if r == ' ' || r == '/' {
			return '-'
		}
		return r
	}, out.Product))
	path := filepath.Join(dir, time.Now().UTC().Format("20060102T150405Z")+"-"+name+".json")
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal plan: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write plan: %w", err)
	}
	return path, nil
}
