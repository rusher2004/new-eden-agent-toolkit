package mcp

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// ItemInput names an item by name or type ID, with a quantity.
type ItemInput struct {
	Name     string `json:"name,omitempty" jsonschema:"item name, e.g. Tritanium"`
	TypeID   int64  `json:"typeId,omitempty" jsonschema:"item type ID; used when name is empty"`
	Quantity int64  `json:"quantity"`
}

func (p *planner) resolveItems(ctx context.Context, items []ItemInput) ([]industry.TypeQty, error) {
	var names []string
	for _, it := range items {
		if it.TypeID == 0 {
			names = append(names, it.Name)
		}
	}
	ids, err := p.sde.TypesByName(ctx, p.ua(), names)
	if err != nil {
		return nil, err
	}
	out := make([]industry.TypeQty, 0, len(items))
	for _, it := range items {
		id := it.TypeID
		if id == 0 {
			if id = ids[it.Name]; id == 0 {
				return nil, fmt.Errorf("unknown item %q", it.Name)
			}
		}
		out = append(out, industry.TypeQty{TypeID: id, Quantity: it.Quantity})
	}
	return out, nil
}

func (p *planner) resolveType(ctx context.Context, name string) (int64, error) {
	ids, err := p.sde.TypesByName(ctx, p.ua(), []string{name})
	if err != nil {
		return 0, err
	}
	if ids[name] == 0 {
		return 0, fmt.Errorf("unknown item %q", name)
	}
	return ids[name], nil
}

func (p *planner) hub(name string) (config.HubProfile, error) {
	cfg, err := p.store.Load()
	if err != nil {
		return config.HubProfile{}, err
	}
	h, ok := cfg.HubProfile(name)
	if !ok {
		return config.HubProfile{}, fmt.Errorf("no hub profile named %q", name)
	}
	return h, nil
}

// --- quotes ---

// QuoteLine is one item priced against the hub's order book.
type QuoteLine struct {
	TypeID    int64   `json:"typeId"`
	Name      string  `json:"name"`
	Quantity  int64   `json:"quantity"`
	Average   float64 `json:"averagePrice"`
	Lowest    float64 `json:"lowestPrice"`
	Total     float64 `json:"total"`
	Shortfall int64   `json:"shortfall,omitempty"`
	Depth     int64   `json:"depthAtHub"`
	M3        float64 `json:"m3"`
}

func (p *planner) quote(ctx context.Context, hub config.HubProfile, items []industry.TypeQty, books map[int64][]industry.Order) ([]QuoteLine, error) {
	var ids []int64
	for _, it := range items {
		ids = append(ids, it.TypeID)
	}
	if books == nil {
		var err error
		if books, err = p.sellBooks(ctx, hub.RegionID, ids); err != nil {
			return nil, err
		}
	}
	basics, err := p.typeBasics(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]QuoteLine, 0, len(items))
	for _, it := range items {
		q := industry.WalkBook(it.TypeID, books[it.TypeID], it.Quantity, hub.StationID)
		b := basics[it.TypeID]
		out = append(out, QuoteLine{TypeID: it.TypeID, Name: b.Name, Quantity: it.Quantity, Average: q.Average, Lowest: q.Lowest, Total: q.Total,
			Shortfall: q.Shortfall, Depth: q.Depth, M3: b.Volume * float64(it.Quantity)})
	}
	return out, nil
}

func multibuyOf(lines []QuoteLine) string {
	ml := make([]industry.MultibuyLine, 0, len(lines))
	for _, l := range lines {
		ml = append(ml, industry.MultibuyLine{Name: l.Name, Quantity: l.Quantity})
	}
	return industry.Multibuy(ml)
}

type QuoteMarketPurchaseInput struct {
	Items []ItemInput `json:"items"`
	Hub   string      `json:"hub,omitempty" jsonschema:"hub profile name; default Jita 4-4"`
}

type QuoteMarketPurchaseOutput struct {
	Hub      string      `json:"hub"`
	Lines    []QuoteLine `json:"lines"`
	Total    float64     `json:"total"`
	M3       float64     `json:"m3"`
	Multibuy string      `json:"multibuy" jsonschema:"paste into the in-game Multibuy window's import"`
}

func quoteMarketPurchaseHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[QuoteMarketPurchaseInput, QuoteMarketPurchaseOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in QuoteMarketPurchaseInput) (*mcpsdk.CallToolResult, QuoteMarketPurchaseOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		hub, err := p.hub(in.Hub)
		if err != nil {
			return nil, QuoteMarketPurchaseOutput{}, err
		}
		items, err := p.resolveItems(ctx, in.Items)
		if err != nil {
			return nil, QuoteMarketPurchaseOutput{}, err
		}
		lines, err := p.quote(ctx, hub, items, nil)
		if err != nil {
			return nil, QuoteMarketPurchaseOutput{}, err
		}
		out := QuoteMarketPurchaseOutput{Hub: hub.Name, Lines: lines, Multibuy: multibuyOf(lines)}
		for _, l := range lines {
			out.Total += l.Total
			out.M3 += l.M3
		}
		return nil, out, nil
	}
}

// --- owned blueprints ---

// OwnedBlueprint is a blueprint a character or corporation owns.
type OwnedBlueprint struct {
	ItemID          int64  `json:"itemId"`
	BlueprintTypeID int64  `json:"blueprintTypeId"`
	BlueprintName   string `json:"blueprintName"`
	ME              int    `json:"me"`
	TE              int    `json:"te"`
	Runs            int64  `json:"runs"` // -1 for originals
	Original        bool   `json:"original"`
	InActiveJob     bool   `json:"inActiveJob"`
	LocationID      int64  `json:"locationId"`
	LocationFlag    string `json:"locationFlag"`
	RootLocationID  int64  `json:"rootLocationId" jsonschema:"the station or structure the blueprint is ultimately in"`
	Container       string `json:"container,omitempty" jsonschema:"named container holding it, if any"`
}

// ownership is a character's or corporation's blueprints and assets.
type ownership struct {
	blueprints []OwnedBlueprint
	assets     map[int64]AssetItem
	names      map[int64]string
}

func (p *planner) loadOwnership(ctx context.Context, characterID int32, corporationID int64, withAssets bool) (ownership, error) {
	if err := p.requireCharacter(characterID); err != nil {
		return ownership{}, err
	}
	var own ownership
	active := map[int64]bool{}
	if corporationID != 0 {
		bps, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]CorporationBlueprint, int32, error) {
			return listCorporationBlueprints(ctx, p.clientID, characterID, p.ua(), corporationID, page)
		})
		if err != nil {
			return own, fmt.Errorf("corporation blueprints: %w", err)
		}
		for _, b := range bps {
			own.blueprints = append(own.blueprints, OwnedBlueprint{ItemID: b.ItemID, BlueprintTypeID: b.TypeID, ME: int(b.MaterialEfficiency), TE: int(b.TimeEfficiency), Runs: b.Runs, Original: b.Quantity != -2, LocationID: b.LocationID, LocationFlag: b.LocationFlag})
		}
		jobs, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]IndustryJob, int32, error) {
			raw, total, err := listCorporationIndustryJobs(ctx, p.clientID, characterID, p.ua(), corporationID, false, page)
			out := make([]IndustryJob, len(raw))
			for i, j := range raw {
				out[i] = toIndustryJobFromCorporation(j)
			}
			return out, total, err
		})
		if err != nil {
			return own, fmt.Errorf("corporation jobs: %w", err)
		}
		for _, j := range jobs {
			if j.Status == "active" || j.Status == "paused" || j.Status == "ready" {
				active[j.BlueprintID] = true
			}
		}
		if withAssets {
			assets, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]AssetItem, int32, error) {
				return listCorporationAssets(ctx, p.clientID, characterID, p.ua(), corporationID, page)
			})
			if err != nil {
				return own, fmt.Errorf("corporation assets: %w", err)
			}
			own.assets = indexAssets(assets)
		}
	} else {
		bps, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Blueprint, int32, error) {
			return listCharacterBlueprints(ctx, p.clientID, characterID, p.ua(), page)
		})
		if err != nil {
			return own, fmt.Errorf("character blueprints: %w", err)
		}
		for _, b := range bps {
			own.blueprints = append(own.blueprints, OwnedBlueprint{ItemID: b.ItemID, BlueprintTypeID: b.TypeID, ME: int(b.MaterialEfficiency), TE: int(b.TimeEfficiency), Runs: b.Runs, Original: b.Quantity != -2, LocationID: b.LocationID, LocationFlag: b.LocationFlag})
		}
		jobs, err := listCharacterIndustryJobs(ctx, p.clientID, characterID, p.ua(), false)
		if err != nil {
			return own, fmt.Errorf("character jobs: %w", err)
		}
		for _, j := range jobs {
			active[j.BlueprintId] = true
		}
		if withAssets {
			assets, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]AssetItem, int32, error) {
				return listAssets(ctx, p.clientID, characterID, p.ua(), page)
			})
			if err != nil {
				return own, fmt.Errorf("character assets: %w", err)
			}
			own.assets = indexAssets(assets)
		}
	}
	for i := range own.blueprints {
		own.blueprints[i].InActiveJob = active[own.blueprints[i].ItemID]
		if own.blueprints[i].Original {
			own.blueprints[i].Runs = -1
		}
	}
	return own, nil
}

func indexAssets(items []AssetItem) map[int64]AssetItem {
	m := make(map[int64]AssetItem, len(items))
	for _, a := range items {
		m[a.ItemID] = a
	}
	return m
}

// root follows an item's location chain to the station or structure it's
// ultimately in, returning that ID and the top-level hangar flag.
func (o ownership) root(locationID int64, flag string) (int64, string) {
	for range 20 {
		a, ok := o.assets[locationID]
		if !ok {
			return locationID, flag
		}
		locationID, flag = a.LocationID, a.LocationFlag
	}
	return locationID, flag
}

// containerNames fetches custom names for container items holding the given
// blueprints.
func (p *planner) containerNames(ctx context.Context, own *ownership, characterID int32, corporationID int64) {
	var ids []int64
	for _, b := range own.blueprints {
		if a, ok := own.assets[b.LocationID]; ok && a.TypeID != 27 { // 27 = Office
			ids = append(ids, b.LocationID)
		}
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	own.names = map[int64]string{}
	for start := 0; start < len(ids); start += 1000 {
		chunk := ids[start:min(start+1000, len(ids))]
		var names []AssetName
		var err error
		if corporationID != 0 {
			names, err = getCorporationAssetNames(ctx, p.clientID, characterID, p.ua(), corporationID, chunk)
		} else {
			names, err = getAssetNames(ctx, p.clientID, characterID, p.ua(), chunk)
		}
		if err != nil {
			return // names are a convenience; missing ones stay blank
		}
		for _, n := range names {
			own.names[n.ItemID] = n.Name
		}
	}
}

type FindOwnedBlueprintsInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"a logged-in character (for corporation blueprints, one with the roles to read them)"`
	CorporationID int64  `json:"corporationId,omitempty" jsonschema:"read this corporation's blueprints instead of the character's"`
	Product       string `json:"product,omitempty" jsonschema:"only blueprints that build this item, e.g. Minokawa"`
	Blueprint     string `json:"blueprint,omitempty" jsonschema:"only blueprints whose name contains this text"`
}

type BlueprintGroup struct {
	BlueprintTypeID int64  `json:"blueprintTypeId"`
	BlueprintName   string `json:"blueprintName"`
	Copies          int    `json:"copies"`
	Originals       int    `json:"originals"`
	FreeRuns        int64  `json:"freeRuns" jsonschema:"runs on copies not tied up in active jobs"`
	InActiveJobs    int    `json:"inActiveJobs"`
}

type FindOwnedBlueprintsOutput struct {
	Groups     []BlueprintGroup `json:"groups"`
	Blueprints []OwnedBlueprint `json:"blueprints"`
}

func findOwnedBlueprintsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[FindOwnedBlueprintsInput, FindOwnedBlueprintsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in FindOwnedBlueprintsInput) (*mcpsdk.CallToolResult, FindOwnedBlueprintsOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		own, err := p.loadOwnership(ctx, in.CharacterID, in.CorporationID, true)
		if err != nil {
			return nil, FindOwnedBlueprintsOutput{}, err
		}
		var allow map[int64]bool
		if in.Product != "" {
			pid, err := p.resolveType(ctx, in.Product)
			if err != nil {
				return nil, FindOwnedBlueprintsOutput{}, err
			}
			srcs, err := sdeStore.BlueprintsForProduct(ctx, p.ua(), pid)
			if err != nil {
				return nil, FindOwnedBlueprintsOutput{}, err
			}
			allow = map[int64]bool{}
			for _, s := range srcs {
				allow[s.BlueprintTypeID] = true
			}
		}
		var ids []int64
		for _, b := range own.blueprints {
			ids = append(ids, b.BlueprintTypeID)
		}
		basics, err := p.typeBasics(ctx, ids)
		if err != nil {
			return nil, FindOwnedBlueprintsOutput{}, err
		}
		var kept []OwnedBlueprint
		for _, b := range own.blueprints {
			b.BlueprintName = basics[b.BlueprintTypeID].Name
			if allow != nil && !allow[b.BlueprintTypeID] {
				continue
			}
			if in.Blueprint != "" && !strings.Contains(strings.ToLower(b.BlueprintName), strings.ToLower(in.Blueprint)) {
				continue
			}
			kept = append(kept, b)
		}
		own.blueprints = kept
		p.containerNames(ctx, &own, in.CharacterID, in.CorporationID)
		groups := map[int64]*BlueprintGroup{}
		for i := range own.blueprints {
			b := &own.blueprints[i]
			b.RootLocationID, _ = own.root(b.LocationID, b.LocationFlag)
			b.Container = own.names[b.LocationID]
			g := groups[b.BlueprintTypeID]
			if g == nil {
				g = &BlueprintGroup{BlueprintTypeID: b.BlueprintTypeID, BlueprintName: b.BlueprintName}
				groups[b.BlueprintTypeID] = g
			}
			if b.Original {
				g.Originals++
			} else {
				g.Copies++
			}
			if b.InActiveJob {
				g.InActiveJobs++
			} else if b.Runs > 0 {
				g.FreeRuns += b.Runs
			}
		}
		out := FindOwnedBlueprintsOutput{Blueprints: own.blueprints}
		for _, id := range sortedKeys(groups) {
			out.Groups = append(out.Groups, *groups[id])
		}
		sort.Slice(out.Groups, func(i, j int) bool { return out.Groups[i].BlueprintName < out.Groups[j].BlueprintName })
		return nil, out, nil
	}
}

// --- stock ---

// stockAt sums owned items whose root location is locationID, optionally only
// in one top-level hangar flag (e.g. CorpSAG3). Blueprints are excluded.
func (o ownership) stockAt(locationID int64, flag string) map[int64]int64 {
	out := map[int64]int64{}
	if locationID == 0 {
		return out
	}
	for _, a := range o.assets {
		if a.IsBlueprintCopy != nil {
			continue
		}
		r, f := o.root(a.LocationID, a.LocationFlag)
		if r != locationID {
			continue
		}
		if flag != "" && f != flag && a.LocationFlag != flag {
			// f is the flag of the item directly in the station or office;
			// for corp offices the division flag sits one level down.
			if !o.inFlag(a, flag) {
				continue
			}
		}
		out[a.TypeID] += a.Quantity
	}
	return out
}

func (o ownership) inFlag(a AssetItem, flag string) bool {
	for range 20 {
		if a.LocationFlag == flag {
			return true
		}
		parent, ok := o.assets[a.LocationID]
		if !ok {
			return false
		}
		a = parent
	}
	return false
}

type CheckMaterialStockInput struct {
	CharacterID   int32       `json:"characterId"`
	CorporationID int64       `json:"corporationId,omitempty"`
	LocationID    int64       `json:"locationId" jsonschema:"station or structure ID the stock must be in"`
	Division      string      `json:"division,omitempty" jsonschema:"only this hangar flag, e.g. CorpSAG3 for corp hangar division 3"`
	Materials     []ItemInput `json:"materials"`
}

type StockLine struct {
	TypeID    int64   `json:"typeId"`
	Name      string  `json:"name"`
	Needed    int64   `json:"needed"`
	InStock   int64   `json:"inStock"`
	Remaining int64   `json:"remaining"`
	StockM3   float64 `json:"stockM3"`
}

type CheckMaterialStockOutput struct {
	Lines []StockLine `json:"lines"`
}

func checkMaterialStockHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[CheckMaterialStockInput, CheckMaterialStockOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CheckMaterialStockInput) (*mcpsdk.CallToolResult, CheckMaterialStockOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		items, err := p.resolveItems(ctx, in.Materials)
		if err != nil {
			return nil, CheckMaterialStockOutput{}, err
		}
		own, err := p.loadOwnership(ctx, in.CharacterID, in.CorporationID, true)
		if err != nil {
			return nil, CheckMaterialStockOutput{}, err
		}
		stock := own.stockAt(in.LocationID, in.Division)
		var ids []int64
		for _, it := range items {
			ids = append(ids, it.TypeID)
		}
		basics, err := p.typeBasics(ctx, ids)
		if err != nil {
			return nil, CheckMaterialStockOutput{}, err
		}
		var out CheckMaterialStockOutput
		for _, it := range items {
			have := min(stock[it.TypeID], it.Quantity)
			out.Lines = append(out.Lines, StockLine{TypeID: it.TypeID, Name: basics[it.TypeID].Name, Needed: it.Quantity, InStock: stock[it.TypeID], Remaining: it.Quantity - have, StockM3: float64(have) * basics[it.TypeID].Volume})
		}
		return nil, out, nil
	}
}

// --- skills ---

type CheckBuildSkillsInput struct {
	CharacterIDs []int32 `json:"characterIds"`
	Product      string  `json:"product" jsonschema:"item to build, e.g. Minokawa"`
}

type BuilderView struct {
	CharacterID int32              `json:"characterId"`
	Name        string             `json:"name"`
	Slots       map[string]int     `json:"slots"`
	TimeMult    map[string]float64 `json:"timeMultipliers"`
	CanBuild    map[string]string  `json:"canBuild" jsonschema:"item name → yes, or the missing skills"`
}

type CheckBuildSkillsOutput struct {
	Builders []BuilderView `json:"builders"`
}

func checkBuildSkillsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[CheckBuildSkillsInput, CheckBuildSkillsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CheckBuildSkillsInput) (*mcpsdk.CallToolResult, CheckBuildSkillsOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		pid, err := p.resolveType(ctx, in.Product)
		if err != nil {
			return nil, CheckBuildSkillsOutput{}, err
		}
		// Every buildable type in the product's tree, one level of recipes each.
		recipes := map[int64]industry.Recipe{}
		var walk func(t int64, depth int) error
		walk = func(t int64, depth int) error {
			if _, seen := recipes[t]; seen || depth > 8 {
				return nil
			}
			r, ok, err := p.recipe(ctx, t)
			if err != nil || !ok {
				return err
			}
			recipes[t] = r
			for _, m := range r.Materials {
				if err := walk(m.TypeID, depth+1); err != nil {
					return err
				}
			}
			return nil
		}
		if err := walk(pid, 0); err != nil {
			return nil, CheckBuildSkillsOutput{}, err
		}
		var allIDs []int64
		for t, r := range recipes {
			allIDs = append(allIDs, t)
			for _, s := range r.Skills {
				allIDs = append(allIDs, s.TypeID)
			}
		}
		basics, err := p.typeBasics(ctx, allIDs)
		if err != nil {
			return nil, CheckBuildSkillsOutput{}, err
		}
		var out CheckBuildSkillsOutput
		for _, cid := range in.CharacterIDs {
			b, err := p.builder(ctx, cid, p.characterName(cid))
			if err != nil {
				return nil, CheckBuildSkillsOutput{}, err
			}
			v := BuilderView{CharacterID: cid, Name: b.Name, Slots: b.Slots, TimeMult: b.TimeMult, CanBuild: map[string]string{}}
			for t, r := range recipes {
				var missing []string
				for _, s := range r.Skills {
					if int64(b.Skills[s.TypeID]) < s.Quantity {
						missing = append(missing, fmt.Sprintf("%s %d (has %d)", basics[s.TypeID].Name, s.Quantity, b.Skills[s.TypeID]))
					}
				}
				if len(missing) == 0 {
					v.CanBuild[basics[t].Name] = "yes"
				} else {
					v.CanBuild[basics[t].Name] = "needs " + strings.Join(missing, ", ")
				}
			}
			out.Builders = append(out.Builders, v)
		}
		return nil, out, nil
	}
}

// --- job cost ---

type EstimateJobCostInput struct {
	Product  string `json:"product" jsonschema:"item the job makes"`
	Runs     int64  `json:"runs"`
	Facility string `json:"facility" jsonschema:"facility profile name"`
	Alpha    bool   `json:"alpha,omitempty"`
}

type EstimateJobCostOutput struct {
	EstimatedItemValue float64 `json:"estimatedItemValue"`
	CostIndex          float64 `json:"costIndex"`
	StructureCostBonus float64 `json:"structureCostBonus"`
	FacilityTax        float64 `json:"facilityTax"`
	Fee                float64 `json:"fee"`
	FeeRate            float64 `json:"feeRate"`
	Calibrated         bool    `json:"calibrated" jsonschema:"true when the profile's observed fee rate was used"`
}

func (p *planner) jobFee(ctx context.Context, f industry.Facility, prof config.FacilityProfile, activity string, product int64, eiv float64, alpha bool) (EstimateJobCostOutput, error) {
	out := EstimateJobCostOutput{EstimatedItemValue: eiv, FacilityTax: f.FacilityTax}
	if prof.Calibration != nil && prof.Calibration.FeeRate > 0 {
		out.FeeRate, out.Calibrated = prof.Calibration.FeeRate, true
		out.Fee = eiv * out.FeeRate
		return out, nil
	}
	ci, err := p.costIndex(ctx, prof.SystemID, activity)
	if err != nil {
		return out, err
	}
	b, err := p.typeBasics(ctx, []int64{product})
	if err != nil {
		return out, err
	}
	m := f.Multipliers(activity, b[product].CategoryID, b[product].GroupID)
	out.CostIndex, out.StructureCostBonus = ci, m.Cost
	out.Fee = industry.JobFee(industry.FeeInput{BaseValue: eiv, CostIndex: ci, StructureCostBonus: m.Cost, FacilityTax: f.FacilityTax, Alpha: alpha})
	if eiv > 0 {
		out.FeeRate = out.Fee / eiv
	}
	return out, nil
}

func estimateJobCostHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[EstimateJobCostInput, EstimateJobCostOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in EstimateJobCostInput) (*mcpsdk.CallToolResult, EstimateJobCostOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		pid, err := p.resolveType(ctx, in.Product)
		if err != nil {
			return nil, EstimateJobCostOutput{}, err
		}
		r, ok, err := p.recipe(ctx, pid)
		if err != nil {
			return nil, EstimateJobCostOutput{}, err
		}
		if !ok {
			return nil, EstimateJobCostOutput{}, fmt.Errorf("%s isn't manufactured or reacted", in.Product)
		}
		prof, err := p.facilityProfile(in.Facility)
		if err != nil {
			return nil, EstimateJobCostOutput{}, err
		}
		f, err := p.facility(ctx, prof)
		if err != nil {
			return nil, EstimateJobCostOutput{}, err
		}
		prices, err := p.adjustedPrices(ctx)
		if err != nil {
			return nil, EstimateJobCostOutput{}, err
		}
		mats := map[int64]int64{}
		for _, m := range r.Materials {
			mats[m.TypeID] = m.Quantity
		}
		out, err := p.jobFee(ctx, f, prof, r.Activity, pid, industry.EstimatedItemValue(mats, prices, in.Runs), in.Alpha)
		return nil, out, err
	}
}

// --- research and invention ---

type EstimateResearchInput struct {
	Blueprint   string `json:"blueprint" jsonschema:"blueprint name"`
	Kind        string `json:"kind" jsonschema:"me, te or copy"`
	FromLevel   int    `json:"fromLevel,omitempty" jsonschema:"research level 0-10 to start from; for TE a level is the blueprint's TE percent divided by 2 (TE 14 is level 7)"`
	ToLevel     int    `json:"toLevel,omitempty" jsonschema:"research level 1-10 to reach; for TE a level is the TE percent divided by 2 (TE 20 is level 10)"`
	Copies      int64  `json:"copies,omitempty"`
	RunsPerCopy int64  `json:"runsPerCopy,omitempty"`
	CharacterID int32  `json:"characterId"`
	Facility    string `json:"facility"`
}

type EstimateResearchOutput struct {
	Seconds float64 `json:"seconds"`
	Hours   float64 `json:"hours"`
	Fee     float64 `json:"fee"`
	FeeNote string  `json:"feeNote,omitempty"`
}

func estimateResearchHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[EstimateResearchInput, EstimateResearchOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in EstimateResearchInput) (*mcpsdk.CallToolResult, EstimateResearchOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		if (in.Kind == "me" || in.Kind == "te") && (in.FromLevel < 0 || in.ToLevel > 10 || in.ToLevel <= in.FromLevel) {
			return nil, EstimateResearchOutput{}, fmt.Errorf("fromLevel and toLevel must be research levels 0-10 with fromLevel below toLevel, got %d and %d; for TE a level is the TE percent divided by 2 (TE 14 is level 7)", in.FromLevel, in.ToLevel)
		}
		bid, err := p.resolveType(ctx, in.Blueprint)
		if err != nil {
			return nil, EstimateResearchOutput{}, err
		}
		bp, ok, err := sdeStore.Blueprint(ctx, p.ua(), bid)
		if err != nil || !ok {
			return nil, EstimateResearchOutput{}, fmt.Errorf("%s has no industry data: %v", in.Blueprint, err)
		}
		prof, err := p.facilityProfile(in.Facility)
		if err != nil {
			return nil, EstimateResearchOutput{}, err
		}
		f, err := p.facility(ctx, prof)
		if err != nil {
			return nil, EstimateResearchOutput{}, err
		}
		skills, err := p.skills(ctx, in.CharacterID)
		if err != nil {
			return nil, EstimateResearchOutput{}, err
		}
		prices, err := p.adjustedPrices(ctx)
		if err != nil {
			return nil, EstimateResearchOutput{}, err
		}
		mfg, _ := bp.Activity(sde.ActivityManufacturing)
		if len(mfg.Materials) == 0 {
			mfg, _ = bp.Activity(sde.ActivityReactions)
		}
		mats := map[int64]int64{}
		for _, m := range mfg.Materials {
			mats[m.TypeID] = m.Quantity
		}
		eiv := industry.EstimatedItemValue(mats, prices, 1)
		var activity string
		var out EstimateResearchOutput
		var base float64
		switch in.Kind {
		case "me":
			activity = industry.ActResearchMaterial
			a, _ := bp.Activity(sde.ActivityResearchME)
			m := f.Multipliers(activity, 0, 0)
			out.Seconds = industry.ResearchSeconds(float64(a.TimeSeconds)/105, in.FromLevel, in.ToLevel, industry.ResearchSkillMultiplier(skills[3409], skills[skillAdvancedIndustry]), m.Time)
			base = industry.ResearchBaseValue(eiv, in.FromLevel, in.ToLevel)
			out.FeeNote = "research fees use EVE University's formula, which is unverified against the game; calibrate against a real research job before relying on it"
		case "te":
			activity = industry.ActResearchTime
			a, _ := bp.Activity(sde.ActivityResearchTE)
			m := f.Multipliers(activity, 0, 0)
			out.Seconds = industry.ResearchSeconds(float64(a.TimeSeconds)/105, in.FromLevel, in.ToLevel, industry.ResearchSkillMultiplier(skills[3403], skills[skillAdvancedIndustry]), m.Time)
			base = industry.ResearchBaseValue(eiv, in.FromLevel, in.ToLevel)
			out.FeeNote = "research fees use EVE University's formula, which is unverified against the game; calibrate against a real research job before relying on it"
		case "copy":
			activity = industry.ActCopying
			a, _ := bp.Activity(sde.ActivityCopying)
			m := f.Multipliers(activity, 0, 0)
			out.Seconds = industry.CopySeconds(float64(a.TimeSeconds), max(in.Copies, 1), max(in.RunsPerCopy, 1), industry.ResearchSkillMultiplier(skills[3402], skills[skillAdvancedIndustry]), m.Time)
			base = industry.CopyBaseValue(eiv, max(in.Copies, 1), max(in.RunsPerCopy, 1))
		default:
			return nil, EstimateResearchOutput{}, fmt.Errorf("kind must be me, te or copy")
		}
		fee, err := p.jobFee(ctx, f, config.FacilityProfile{SystemID: prof.SystemID}, activity, bid, base, false)
		if err != nil {
			return nil, EstimateResearchOutput{}, err
		}
		out.Fee, out.Hours = fee.Fee, out.Seconds/3600
		return nil, out, nil
	}
}

type EstimateInventionInput struct {
	Product     string `json:"product" jsonschema:"the T2 item to invent a blueprint for, e.g. Sabre"`
	CharacterID int32  `json:"characterId"`
	Facility    string `json:"facility"`
	Hub         string `json:"hub,omitempty"`
}

type InventionOption struct {
	Decryptor      string  `json:"decryptor"`
	Chance         float64 `json:"chance"`
	Runs           int     `json:"runs"`
	ME             int     `json:"me"`
	TE             int     `json:"te"`
	AttemptCost    float64 `json:"attemptCost" jsonschema:"datacores, decryptor and job fee for one attempt"`
	CostPerSuccess float64 `json:"costPerSuccess"`
	CostPerRun     float64 `json:"costPerRun" jsonschema:"expected cost per run of the invented copy"`
}

type EstimateInventionOutput struct {
	T1Blueprint string            `json:"t1Blueprint"`
	Options     []InventionOption `json:"options"`
	Notes       []string          `json:"notes,omitempty"`
}

func estimateInventionHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[EstimateInventionInput, EstimateInventionOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in EstimateInventionInput) (*mcpsdk.CallToolResult, EstimateInventionOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		pid, err := p.resolveType(ctx, in.Product)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		t2, ok, err := p.recipe(ctx, pid)
		if err != nil || !ok {
			return nil, EstimateInventionOutput{}, fmt.Errorf("%s isn't manufactured: %v", in.Product, err)
		}
		srcs, err := sdeStore.BlueprintSourcesForActivity(ctx, p.ua(), t2.BlueprintTypeID, sde.ActivityInvention)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		if len(srcs) == 0 {
			return nil, EstimateInventionOutput{}, fmt.Errorf("nothing invents the %s blueprint", in.Product)
		}
		t1, _, err := sdeStore.Blueprint(ctx, p.ua(), srcs[0].BlueprintTypeID)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		inv, _ := t1.Activity(sde.ActivityInvention)
		var prob float64
		for _, pr := range inv.Products {
			if pr.TypeID == t2.BlueprintTypeID {
				prob = pr.Probability
			}
		}
		if prob <= 0 {
			return nil, EstimateInventionOutput{}, fmt.Errorf("the SDE has no invention chance for the %s blueprint from blueprint %d", in.Product, srcs[0].BlueprintTypeID)
		}
		skills, err := p.skills(ctx, in.CharacterID)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		var skillIDs []int64
		for _, s := range inv.Skills {
			skillIDs = append(skillIDs, s.SkillID)
		}
		basics, err := p.typeBasics(ctx, append(skillIDs, srcs[0].BlueprintTypeID))
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		ii := industry.InventionInput{BaseChance: prob, BaseRuns: int(srcs[0].Quantity)}
		var sciences []int
		for _, s := range inv.Skills {
			if strings.Contains(basics[s.SkillID].Name, "Encryption Methods") {
				ii.Encryption = skills[s.SkillID]
			} else {
				sciences = append(sciences, skills[s.SkillID])
			}
		}
		if len(sciences) > 0 {
			ii.Science1 = sciences[0]
		}
		if len(sciences) > 1 {
			ii.Science2 = sciences[1]
		}
		hub, err := p.hub(in.Hub)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		var buy []industry.TypeQty
		for _, m := range inv.Materials {
			buy = append(buy, industry.TypeQty{TypeID: m.TypeID, Quantity: m.Quantity})
		}
		decIDs, err := sdeStore.TypesByName(ctx, p.ua(), decryptorNames())
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		for _, id := range decIDs {
			buy = append(buy, industry.TypeQty{TypeID: id, Quantity: 1})
		}
		lines, err := p.quote(ctx, hub, buy, nil)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		// A zero average means nothing is for sale at the hub; such items are
		// left unpriced rather than counted as free.
		price := map[int64]float64{}
		for _, l := range lines {
			if l.Quantity > 0 && l.Average > 0 {
				price[l.TypeID] = l.Average
			}
		}
		var notes []string
		datacores := 0.0
		for _, m := range inv.Materials {
			pr, ok := price[m.TypeID]
			if !ok {
				notes = append(notes, fmt.Sprintf("%s has no sell orders at %s; attempt costs leave it out", p.typeName(ctx, m.TypeID), hub.Name))
			}
			datacores += pr * float64(m.Quantity)
		}
		prof, err := p.facilityProfile(in.Facility)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		f, err := p.facility(ctx, prof)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		prices, err := p.adjustedPrices(ctx)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		mats := map[int64]int64{}
		for _, m := range t2.Materials {
			mats[m.TypeID] = m.Quantity
		}
		fee, err := p.jobFee(ctx, f, config.FacilityProfile{SystemID: prof.SystemID}, industry.ActInvention, pid, industry.CopyBaseValue(industry.EstimatedItemValue(mats, prices, 1), 1, 1), false)
		if err != nil {
			return nil, EstimateInventionOutput{}, err
		}
		out := EstimateInventionOutput{T1Blueprint: basics[srcs[0].BlueprintTypeID].Name,
			Notes: append([]string{"the invention job fee is charged on 2% of the T2 item's estimated value per attempt, by analogy with copying; calibrate against a real invention job"}, notes...)}
		addOption := func(name string, d *industry.Decryptor, decPrice float64) {
			r := industry.Invent(industry.InventionInput{BaseChance: ii.BaseChance, Science1: ii.Science1, Science2: ii.Science2, Encryption: ii.Encryption, Decryptor: d, BaseRuns: ii.BaseRuns})
			attempt := datacores + decPrice + fee.Fee
			per := industry.ExpectedCostPerSuccess(attempt, r.Chance)
			out.Options = append(out.Options, InventionOption{Decryptor: name, Chance: r.Chance, Runs: r.Runs, ME: r.ME, TE: r.TE, AttemptCost: attempt, CostPerSuccess: per, CostPerRun: per / float64(max(r.Runs, 1))})
		}
		addOption("none", nil, 0)
		for i := range industry.Decryptors {
			d := industry.Decryptors[i]
			pr, ok := price[decIDs[d.Name]]
			if !ok {
				out.Notes = append(out.Notes, fmt.Sprintf("%s skipped: no sell orders at %s", d.Name, hub.Name))
				continue
			}
			addOption(d.Name, &d, pr)
		}
		sort.Slice(out.Options, func(i, j int) bool { return out.Options[i].CostPerRun < out.Options[j].CostPerRun })
		return nil, out, nil
	}
}

func decryptorNames() []string {
	out := make([]string, len(industry.Decryptors))
	for i, d := range industry.Decryptors {
		out[i] = d.Name
	}
	return out
}

// --- reprocessing ---

type EstimateReprocessingYieldInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character who will reprocess"`
	Facility    string `json:"facility" jsonschema:"facility profile name of the refinery or station"`
	UseImplant  *bool  `json:"useImplant,omitempty" jsonschema:"count the character's reprocessing implant (default true); set false if it isn't plugged in"`
}

type YieldLine struct {
	Skill string  `json:"oreProcessingSkill"`
	Level int     `json:"level"`
	Yield float64 `json:"yield"`
}

type EstimateReprocessingYieldOutput struct {
	Implant string      `json:"implant,omitempty"`
	Tax     float64     `json:"tax"`
	Yields  []YieldLine `json:"yields"`
	Basis   string      `json:"basis"`
}

// reprocessYield returns a function giving a character's yield for an ore
// processing skill at a facility, plus the implant used and the tax.
func (p *planner) reprocessYield(ctx context.Context, characterID int32, prof config.FacilityProfile, useImplant bool) (func(skillID int64) float64, string, float64, error) {
	skills, err := p.skills(ctx, characterID)
	if err != nil {
		return nil, "", 0, err
	}
	rs := industry.ReprocessSkills{Reprocessing: skills[skillReprocessing], ReprocessingEfficiency: skills[skillReprocessingEfficiency]}
	implant := ""
	if useImplant {
		if rs.Implant, implant, err = p.implantRefiningBonus(ctx, characterID); err != nil {
			return nil, "", 0, err
		}
	}
	switch prof.Kind {
	case industry.KindNPCStation:
		st, ok, err := p.sde.NPCStationIndustryData(ctx, p.ua(), prof.StationID)
		if err != nil || !ok {
			return nil, "", 0, fmt.Errorf("reprocessing at %q: station %d not found: %v", prof.Name, prof.StationID, err)
		}
		return func(skill int64) float64 { return industry.NPCYield(st.ReprocessingEfficiency, rs, skills[skill]) }, implant, industry.NPCReprocessingTax(prof.Standing), nil
	default:
		sys, err := p.system(ctx, prof.SystemID)
		if err != nil {
			return nil, "", 0, err
		}
		attrs, err := p.sde.TypeAttributes(ctx, p.ua(), prof.StructureType)
		if err != nil {
			return nil, "", 0, err
		}
		in := industry.UpwellYieldInput{RigTier: prof.ReprocessingRigTier, Band: industry.BandOf(sys.Security, sys.RegionID), StructureBonusPct: attrs[sde.AttrStrRefiningYieldBonus]}
		return func(skill int64) float64 { return industry.UpwellYield(in, rs, skills[skill]) }, implant, prof.ReprocessingTax, nil
	}
}

func estimateReprocessingYieldHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[EstimateReprocessingYieldInput, EstimateReprocessingYieldOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in EstimateReprocessingYieldInput) (*mcpsdk.CallToolResult, EstimateReprocessingYieldOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		prof, err := p.facilityProfile(in.Facility)
		if err != nil {
			return nil, EstimateReprocessingYieldOutput{}, err
		}
		use := in.UseImplant == nil || *in.UseImplant
		yield, implant, tax, err := p.reprocessYield(ctx, in.CharacterID, prof, use)
		if err != nil {
			return nil, EstimateReprocessingYieldOutput{}, err
		}
		ores, err := sdeStore.CompressedOres(ctx, p.ua())
		if err != nil {
			return nil, EstimateReprocessingYieldOutput{}, err
		}
		skillSet := map[int64]bool{}
		for _, o := range ores {
			if o.SkillID != 0 {
				skillSet[o.SkillID] = true
			}
		}
		ids := sortedKeys(skillSet)
		basics, err := p.typeBasics(ctx, ids)
		if err != nil {
			return nil, EstimateReprocessingYieldOutput{}, err
		}
		skills, err := p.skills(ctx, in.CharacterID)
		if err != nil {
			return nil, EstimateReprocessingYieldOutput{}, err
		}
		out := EstimateReprocessingYieldOutput{Implant: implant, Tax: tax, Basis: "EVE University reprocessing formula; yields are before tax"}
		for _, id := range ids {
			out.Yields = append(out.Yields, YieldLine{Skill: basics[id].Name, Level: skills[id], Yield: yield(id)})
		}
		return nil, out, nil
	}
}

// --- ore mix ---

// mineralGroupID is the SDE group of refined minerals.
const mineralGroupID int64 = 18

type SolveOreMixInput struct {
	Minerals     []ItemInput `json:"minerals" jsonschema:"minerals needed, by name and quantity"`
	RefinerID    int32       `json:"refinerId" jsonschema:"character who will reprocess"`
	Facility     string      `json:"facility" jsonschema:"facility profile where the ore will be reprocessed"`
	Hub          string      `json:"hub,omitempty"`
	FreightPerM3 float64     `json:"freightPerM3,omitempty" jsonschema:"ISK per m³ to haul; plan_build derives this from the haul profile"`
	UseImplant   *bool       `json:"useImplant,omitempty"`
	IncludeMoon  bool        `json:"includeMoonOre,omitempty"`
}

// TypeName pairs a type ID with its name.
type TypeName struct {
	TypeID int64  `json:"typeId"`
	Name   string `json:"name"`
}

type SolveOreMixOutput struct {
	industry.OreMix
	Names    []TypeName `json:"names" jsonschema:"names for the type IDs in fromOre, surplus and byproducts"`
	Multibuy string     `json:"multibuy"`
	Yield    string     `json:"yieldBasis"`
}

func (p *planner) solveOreMix(ctx context.Context, needs map[int64]int64, refiner int32, prof config.FacilityProfile, hub config.HubProfile, freight float64, useImplant, includeMoon bool) (SolveOreMixOutput, error) {
	yield, implant, tax, err := p.reprocessYield(ctx, refiner, prof, useImplant)
	if err != nil {
		return SolveOreMixOutput{}, err
	}
	ores, err := p.sde.CompressedOres(ctx, p.ua())
	if err != nil {
		return SolveOreMixOutput{}, err
	}
	var typeIDs []int64
	var opts []industry.OreOption
	for _, o := range ores {
		g := strings.ToLower(o.GroupName)
		if g == "ice" || (!includeMoon && strings.Contains(g, "moon")) {
			continue
		}
		useful := false
		for _, m := range o.Materials {
			if needs[m.TypeID] > 0 {
				useful = true
			}
		}
		if !useful {
			continue
		}
		pp := make([]industry.TypeQty, len(o.Materials))
		for i, m := range o.Materials {
			pp[i] = industry.TypeQty{TypeID: m.TypeID, Quantity: m.Quantity}
		}
		opts = append(opts, industry.OreOption{TypeID: o.TypeID, Name: o.Name, Portion: o.PortionSize, M3: o.Volume, PerPortion: pp, Yield: yield(o.SkillID) * (1 - tax)})
		typeIDs = append(typeIDs, o.TypeID)
	}
	var mins []industry.MineralOption
	for t := range needs {
		typeIDs = append(typeIDs, t)
	}
	books, err := p.sellBooks(ctx, hub.RegionID, typeIDs)
	if err != nil {
		return SolveOreMixOutput{}, err
	}
	for i := range opts {
		opts[i].Book = books[opts[i].TypeID]
	}
	basics, err := p.typeBasics(ctx, typeIDs)
	if err != nil {
		return SolveOreMixOutput{}, err
	}
	for t := range needs {
		mins = append(mins, industry.MineralOption{TypeID: t, Name: basics[t].Name, M3: basics[t].Volume, Book: books[t]})
	}
	sort.Slice(mins, func(i, j int) bool { return mins[i].TypeID < mins[j].TypeID })
	mix, err := industry.SolveOreMix(industry.OreMixRequest{Needs: needs, Ores: opts, Minerals: mins, FreightPerM3: freight, StationID: hub.StationID})
	if err != nil {
		return SolveOreMixOutput{}, err
	}
	out := SolveOreMixOutput{OreMix: mix}
	var ml []industry.MultibuyLine
	for _, o := range mix.Ores {
		ml = append(ml, industry.MultibuyLine{Name: o.Name, Quantity: o.Quantity})
	}
	for _, m := range mix.Minerals {
		ml = append(ml, industry.MultibuyLine{Name: m.Name, Quantity: m.Quantity})
	}
	out.Multibuy = industry.Multibuy(ml)
	var outIDs []int64
	for _, list := range [][]industry.TypeQty{mix.FromOre, mix.Surplus, mix.Byproducts} {
		for _, t := range list {
			outIDs = append(outIDs, t.TypeID)
		}
	}
	nb, err := p.typeBasics(ctx, outIDs)
	if err != nil {
		return SolveOreMixOutput{}, err
	}
	for _, id := range sortedKeys(nb) {
		out.Names = append(out.Names, TypeName{TypeID: id, Name: nb[id].Name})
	}
	out.Yield = fmt.Sprintf("yields per ore at %s by character %d", prof.Name, refiner)
	if implant != "" {
		out.Yield += " with " + implant
	}
	if tax > 0 {
		out.Yield += fmt.Sprintf(", after %.2f%% reprocessing tax", tax*100)
	}
	return out, nil
}

func solveOreMixHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[SolveOreMixInput, SolveOreMixOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SolveOreMixInput) (*mcpsdk.CallToolResult, SolveOreMixOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		items, err := p.resolveItems(ctx, in.Minerals)
		if err != nil {
			return nil, SolveOreMixOutput{}, err
		}
		needs := map[int64]int64{}
		for _, it := range items {
			needs[it.TypeID] += it.Quantity
		}
		prof, err := p.facilityProfile(in.Facility)
		if err != nil {
			return nil, SolveOreMixOutput{}, err
		}
		hub, err := p.hub(in.Hub)
		if err != nil {
			return nil, SolveOreMixOutput{}, err
		}
		out, err := p.solveOreMix(ctx, needs, in.RefinerID, prof, hub, in.FreightPerM3, in.UseImplant == nil || *in.UseImplant, in.IncludeMoon)
		return nil, out, err
	}
}

// --- routing ---

type PlanJumpRouteInput struct {
	HaulProfile string           `json:"haulProfile" jsonschema:"haul profile name (ship, pilot and saved preferences)"`
	From        string           `json:"from" jsonschema:"origin system name"`
	To          string           `json:"to" jsonschema:"destination system name"`
	Waypoints   []string         `json:"waypoints,omitempty" jsonschema:"validate and cost this exact route instead of searching (system names, origin to destination)"`
	Prefs       *RoutePrefsInput `json:"prefs,omitempty" jsonschema:"override the profile's saved preferences for this call"`
}

type PlanJumpRouteOutput struct {
	Ship       string              `json:"ship"`
	RangeLY    float64             `json:"rangeLy"`
	FuelPerLY  float64             `json:"fuelPerLy"`
	FuelName   string              `json:"fuelName"`
	FuelPrice  float64             `json:"fuelPrice"`
	Routes     []industry.Route    `json:"routes"`
	Suggestion industry.DotlanLink `json:"dotlanSuggestion"`
	Notes      []string            `json:"notes,omitempty"`
}

func (p *planner) planRoute(ctx context.Context, prof config.HaulProfile, from, to int64, pinned []int64, prefOverride *config.RoutePrefs) (PlanJumpRouteOutput, *industry.RouteGraph, industry.RouteRequest, error) {
	drive, err := p.jumpDrive(ctx, prof.ShipTypeID)
	if err != nil {
		return PlanJumpRouteOutput{}, nil, industry.RouteRequest{}, err
	}
	s, err := p.skills(ctx, int32(prof.PilotID))
	if err != nil {
		return PlanJumpRouteOutput{}, nil, industry.RouteRequest{}, err
	}
	saved := prof.Prefs
	if prefOverride != nil {
		saved = *prefOverride
	}
	prefs, notes, err := p.routePrefs(ctx, int32(prof.PilotID), saved)
	if err != nil {
		return PlanJumpRouteOutput{}, nil, industry.RouteRequest{}, err
	}
	g, err := p.routeGraph(ctx)
	if err != nil {
		return PlanJumpRouteOutput{}, nil, industry.RouteRequest{}, err
	}
	b, err := p.typeBasics(ctx, []int64{prof.ShipTypeID, drive.FuelTypeID})
	if err != nil {
		return PlanJumpRouteOutput{}, nil, industry.RouteRequest{}, err
	}
	req := industry.RouteRequest{From: from, To: to, Drive: drive, Prefs: prefs, CanUseHighsec: canUseHighsec(b[prof.ShipTypeID].GroupID),
		Skills: industry.JumpSkills{JumpDriveCalibration: s[skillJumpDriveCalibration], JumpFuelConservation: s[skillJumpFuelConservation], JumpFreighters: s[skillJumpFreighters]}}
	out := PlanJumpRouteOutput{Ship: drive.ShipName, RangeLY: industry.JumpRange(drive, req.Skills), FuelPerLY: industry.FuelPerLY(drive, req.Skills), FuelName: b[drive.FuelTypeID].Name, Notes: notes}
	if len(pinned) > 0 {
		rt, err := g.ValidateRoute(req, pinned)
		if err != nil {
			return out, g, req, err
		}
		out.Routes = []industry.Route{rt}
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
	industry.Load
	CargoValue float64 `json:"cargoValue"`
	Multibuy   string  `json:"multibuy" jsonschema:"this load's cargo plus round-trip fuel, for the in-game Multibuy"`
}

type PlanHaulOutput struct {
	Loads         []LoadView      `json:"loads"`
	Route         industry.Route  `json:"outboundRoute"`
	ReturnRoute   *industry.Route `json:"returnRoute,omitempty"`
	FuelName      string          `json:"fuelName"`
	FuelPerTrip   int64           `json:"fuelPerTrip"`
	FuelM3PerTrip float64         `json:"fuelM3PerTrip"`
	FuelCostTrip  float64         `json:"fuelCostPerTrip"`
	TotalM3       float64         `json:"totalM3"`
	FreightPerM3  float64         `json:"freightPerM3" jsonschema:"fuel cost of one full round trip divided by the hold"`
	Notes         []string        `json:"notes,omitempty"`
}

// haulPlan routes a round trip and packs cargo.
func (p *planner) haulPlan(ctx context.Context, prof config.HaulProfile, hub config.HubProfile, from, to int64, cargo []industry.Cargo, prices map[int64]float64) (PlanHaulOutput, error) {
	var out PlanHaulOutput
	outbound := pinnedRoute(prof.Outbound, from, to)
	outR, g, req, err := p.planRoute(ctx, prof, from, to, outbound, nil)
	if err != nil {
		return out, fmt.Errorf("outbound route: %w", err)
	}
	out.Route, out.Notes = outR.Routes[0], outR.Notes
	if len(prof.Outbound) > 0 && outbound == nil {
		out.Notes = append(out.Notes, fmt.Sprintf("haul profile %q's pinned outbound route doesn't run from %s to %s, so a route was searched instead", prof.Name, g.Systems[from].Name, g.Systems[to].Name))
	}
	back := req
	back.From, back.To = to, from
	var ret industry.Route
	pinnedReturn := pinnedRoute(prof.Return, to, from)
	if len(prof.Return) > 0 && pinnedReturn == nil {
		out.Notes = append(out.Notes, fmt.Sprintf("haul profile %q's pinned return route doesn't run from %s to %s, so a route was searched instead", prof.Name, g.Systems[to].Name, g.Systems[from].Name))
	}
	if len(pinnedReturn) > 0 {
		ret, err = g.ValidateRoute(back, pinnedReturn)
	} else {
		var rs []industry.Route
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
	loads, err := industry.PackLoads(cargo, prof.HoldM3)
	if err != nil {
		return out, err
	}
	for _, l := range loads {
		industry.SortLoadItems(&l)
		lv := LoadView{Load: l}
		var ml []industry.MultibuyLine
		for _, it := range l.Items {
			lv.CargoValue += prices[it.TypeID] * float64(it.Quantity)
			ml = append(ml, industry.MultibuyLine{Name: it.Name, Quantity: it.Quantity})
		}
		ml = append(ml, industry.MultibuyLine{Name: out.FuelName, Quantity: out.FuelPerTrip})
		lv.Multibuy = industry.Multibuy(ml)
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
		var cargo []industry.Cargo
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
			cargo = append(cargo, industry.Cargo{TypeID: l.TypeID, Name: l.Name, Quantity: l.Quantity, UnitM3: b[l.TypeID].Volume, Portion: portionFor(b[l.TypeID]), Dest: dest})
		}
		out, err := p.haulPlan(ctx, prof, hub, ends[0], ends[1], cargo, prices)
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
