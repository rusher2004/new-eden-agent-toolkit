package mcp

import (
	"context"
	"fmt"
	"math"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/dogma"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// ShipSpec names a ship either as one the user owns (its item ID and the
// character who owns it, so its real fit and cargo are read from assets) or
// as a hull type with a list of modules.
type ShipSpec struct {
	ShipItemID int64    `json:"shipItemId,omitempty" jsonschema:"an assembled ship the user owns, by item ID (from list_location_assets, list_assets or get_logistics_overview); its fit and cargo are read from the owner's assets"`
	OwnerID    int32    `json:"ownerId,omitempty" jsonschema:"the logged-in character who owns shipItemId; defaults to the pilot"`
	ShipType   string   `json:"shipType,omitempty" jsonschema:"a hull by name, for a fit the user describes rather than owns"`
	Modules    []string `json:"modules,omitempty" jsonschema:"with shipType: fitted modules and rigs by name, one entry per module (repeat a name for each copy)"`
}

// resolvedShip is a ship's hull, fitted modules and what's in its holds.
type resolvedShip struct {
	hull        int64
	hullName    string
	shipName    string
	modules     []int64
	moduleNames []string
	cargoM3     float64 // what already sits in the cargo hold
}

// resolveShip reads a ship from assets or builds it from names.
func (p *planner) resolveShip(ctx context.Context, s ShipSpec, pilot int32) (resolvedShip, error) {
	var out resolvedShip
	if s.ShipItemID != 0 {
		owner := s.OwnerID
		if owner == 0 {
			owner = pilot
		}
		if owner == 0 {
			return out, fmt.Errorf("shipItemId needs ownerId or a pilot: the logged-in character who owns it")
		}
		if err := p.requireCharacter(owner); err != nil {
			return out, err
		}
		assets, truncated, err := allAssetPages(func(page int32) ([]AssetItem, int32, error) { return listAssets(ctx, p.clientID, owner, p.ua(), page) })
		if err != nil {
			return out, fmt.Errorf("read the owner's assets: %w", err)
		}
		var found bool
		var ids []int64
		for _, a := range assets {
			ids = append(ids, a.TypeID)
			if a.ItemID == s.ShipItemID {
				found, out.hull = true, a.TypeID
			}
		}
		if !found && truncated {
			return out, fmt.Errorf("character %d has more than %d pages of assets and item %d isn't in the first %d; give the ship as shipType and modules instead", owner, maxAssetPages, s.ShipItemID, maxAssetPages)
		}
		if !found {
			return out, fmt.Errorf("character %d has no asset with item ID %d", owner, s.ShipItemID)
		}
		basics, err := p.typeBasics(ctx, ids)
		if err != nil {
			return out, err
		}
		if basics[out.hull].CategoryID != shipCategoryID {
			return out, fmt.Errorf("item %d is a %s, not a ship", s.ShipItemID, basics[out.hull].Name)
		}
		if names, err := getAssetNames(ctx, p.clientID, owner, p.ua(), []int64{s.ShipItemID}); err == nil && len(names) == 1 {
			out.shipName = names[0].Name
		}
		for _, a := range assets {
			if a.LocationID != s.ShipItemID {
				continue
			}
			switch {
			case isFittedFlag(a.LocationFlag) && basics[a.TypeID].CategoryID != chargeCategoryID:
				// A charge loaded in a module shares its slot flag; it isn't
				// a fitted module.
				out.modules = append(out.modules, a.TypeID)
				out.moduleNames = append(out.moduleNames, basics[a.TypeID].Name)
			case a.LocationFlag == "Cargo":
				out.cargoM3 += cargoItem(a, basics[a.TypeID], "").StackVolume()
			}
		}
		out.hullName = basics[out.hull].Name
		return out, nil
	}
	if s.ShipType == "" {
		return out, fmt.Errorf("give shipItemId, or shipType with its modules")
	}
	names := append([]string{s.ShipType}, s.Modules...)
	ids, err := p.sde.TypesByName(ctx, p.ua(), names)
	if err != nil {
		return out, err
	}
	for _, n := range names {
		if ids[n] == 0 {
			return out, fmt.Errorf("unknown type %q", n)
		}
	}
	out.hull, out.hullName = ids[s.ShipType], s.ShipType
	for _, m := range s.Modules {
		out.modules = append(out.modules, ids[m])
		out.moduleNames = append(out.moduleNames, m)
	}
	return out, nil
}

// pilotSkills reads a pilot's skills, or nil for "untrained" when none is
// given.
func (p *planner) pilotSkills(ctx context.Context, pilot int32) (map[int64]int, error) {
	if pilot == 0 {
		return nil, nil
	}
	return p.skills(ctx, pilot)
}

// --- estimate_ship_hold ---

// EstimateShipHoldInput is estimate_ship_hold's input.
type EstimateShipHoldInput struct {
	ShipSpec
	PilotID int32 `json:"pilotId,omitempty" jsonschema:"the character who'll fly it, whose skills apply; defaults to the owner, and omitting both gives untrained values"`
}

// HoldView is one hold, before and after skills, modules and rigs.
type HoldView struct {
	Hold     string  `json:"hold"`
	BaseM3   float64 `json:"baseM3"`
	FittedM3 float64 `json:"fittedM3"`
}

// EstimateShipHoldOutput is a ship's fitted holds and movement.
type EstimateShipHoldOutput struct {
	Ship          string     `json:"ship"`
	ShipName      string     `json:"shipName,omitempty"`
	Pilot         string     `json:"pilot,omitempty"`
	Modules       []string   `json:"modules"`
	Holds         []HoldView `json:"holds"`
	CargoUsedM3   float64    `json:"cargoUsedM3" jsonschema:"what's already in the cargo hold (owned ships only)"`
	CargoFreeM3   float64    `json:"cargoFreeM3"`
	WarpSpeedAU   float64    `json:"warpSpeedAuPerSecond"`
	MaxVelocity   float64    `json:"maxVelocityMPerSecond"`
	AlignSeconds  float64    `json:"alignSeconds"`
	JumpRangeLY   float64    `json:"jumpRangeLy,omitempty"`
	JumpFuelPerLY float64    `json:"jumpFuelPerLy,omitempty"`
}

// shipHold works out a resolved ship's holds and movement for a pilot.
func (p *planner) shipHold(ctx context.Context, ship resolvedShip, pilot int32) (EstimateShipHoldOutput, *dogma.Fit, error) {
	skills, err := p.pilotSkills(ctx, pilot)
	if err != nil {
		return EstimateShipHoldOutput{}, nil, err
	}
	f, err := p.fit(ctx, ship.hull, ship.modules, skills)
	if err != nil {
		return EstimateShipHoldOutput{}, nil, err
	}
	out := EstimateShipHoldOutput{Ship: ship.hullName, ShipName: ship.shipName, Modules: ship.moduleNames, CargoUsedM3: ship.cargoM3, Holds: []HoldView{}}
	if out.Modules == nil {
		out.Modules = []string{}
	}
	if pilot != 0 {
		out.Pilot = p.characterName(pilot)
	}
	base := f.Ship.Attrs
	for _, h := range append([]struct {
		AttributeID int64
		Hold        string
	}{{sde.AttrCapacity, "cargo"}}, cargo.HoldAttributes...) {
		if base[h.AttributeID] <= 0 {
			continue
		}
		out.Holds = append(out.Holds, HoldView{Hold: h.Hold, BaseM3: base[h.AttributeID], FittedM3: f.ShipAttr(h.AttributeID)})
	}
	out.CargoFreeM3 = math.Max(f.ShipAttr(sde.AttrCapacity)-ship.cargoM3, 0)
	out.WarpSpeedAU = f.ShipAttr(nav.AttrBaseWarpSpeed) * f.ShipAttr(nav.AttrWarpSpeedMultiplier)
	out.MaxVelocity = f.ShipAttr(nav.AttrMaxVelocity)
	out.AlignSeconds = nav.AlignSeconds(f.ShipAttr(sde.AttrMass), f.ShipAttr(nav.AttrInertiaModifier))
	if base[nav.AttrCanJump] != 0 && base[nav.AttrJumpDriveRange] > 0 {
		// The same jump math plan_jump_route uses, which matches dotlan to
		// the unit, so a hold estimate and a route agree.
		drive, err := p.jumpDrive(ctx, ship.hull)
		if err != nil {
			return EstimateShipHoldOutput{}, nil, err
		}
		js := nav.JumpSkills{JumpDriveCalibration: skills[skillJumpDriveCalibration], JumpFuelConservation: skills[skillJumpFuelConservation], JumpFreighters: skills[skillJumpFreighters]}
		out.JumpRangeLY, out.JumpFuelPerLY = nav.JumpRange(drive, js), nav.FuelPerLY(drive, js)
	}
	return out, f, nil
}

func estimateShipHoldHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[EstimateShipHoldInput, EstimateShipHoldOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in EstimateShipHoldInput) (*mcpsdk.CallToolResult, EstimateShipHoldOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		pilot := in.PilotID
		if pilot == 0 {
			pilot = in.OwnerID
		}
		ship, err := p.resolveShip(ctx, in.ShipSpec, pilot)
		if err != nil {
			return nil, EstimateShipHoldOutput{}, fmt.Errorf("estimate ship hold: %w", err)
		}
		out, _, err := p.shipHold(ctx, ship, pilot)
		if err != nil {
			return nil, EstimateShipHoldOutput{}, err
		}
		return nil, out, nil
	}
}

// --- estimate_cyno ---

// Cyno generators by kind.
var cynoGenerators = map[string]int64{"standard": nav.StandardCynoGenerator, "industrial": nav.IndustrialCynoGenerator, "covert": nav.CovertCynoGenerator}

// attrCynoFuelType and attrCynoFuelPerCycle are a cyno generator's fuel type
// and fuel per cycle; attrDuration is its cycle in milliseconds.
const (
	attrDuration         int64 = 73
	attrCynoFuelType     int64 = 713
	attrCynoFuelPerCycle int64 = 714
)

// EstimateCynoInput is estimate_cyno's input.
type EstimateCynoInput struct {
	ShipSpec
	LighterID int32  `json:"lighterId" jsonschema:"the character who'll light it, whose skills apply"`
	Cyno      string `json:"cyno" jsonschema:"standard, industrial or covert"`
	Lights    int    `json:"lights,omitempty" jsonschema:"how many cynos the plan needs this ship to light, such as one per jump leg"`
}

// EstimateCynoOutput is what lighting cynos with a ship takes.
type EstimateCynoOutput struct {
	Ship             string   `json:"ship"`
	Generator        string   `json:"generator"`
	Fuel             string   `json:"fuel"`
	FuelPerLight     float64  `json:"fuelPerLight" jsonschema:"units of fuel one cyno uses, with the lighter's skills and the hull's bonuses"`
	FuelM3PerLight   float64  `json:"fuelM3PerLight"`
	CycleSeconds     float64  `json:"cycleSeconds" jsonschema:"how long the cyno stays up"`
	CargoM3          float64  `json:"cargoM3" jsonschema:"free space in the fitted cargo hold, after what's already in it (owned ships only)"`
	LightsPerHold    int      `json:"lightsPerHold" jsonschema:"cynos one full hold of fuel covers"`
	Lights           int      `json:"lights,omitempty"`
	FuelForLights    float64  `json:"fuelForLights,omitempty"`
	Refills          int      `json:"refills,omitempty" jsonschema:"times the lighter must reload fuel between lights"`
	MissingSkills    []string `json:"missingSkills,omitempty" jsonschema:"skills the lighter still needs for the generator"`
	GeneratorNotFits bool     `json:"generatorDoesntFit,omitempty" jsonschema:"the generator can't be fitted to this hull"`
}

func estimateCynoHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[EstimateCynoInput, EstimateCynoOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in EstimateCynoInput) (*mcpsdk.CallToolResult, EstimateCynoOutput, error) {
		gen, ok := cynoGenerators[strings.ToLower(strings.TrimSpace(in.Cyno))]
		if !ok {
			return nil, EstimateCynoOutput{}, fmt.Errorf("estimate cyno: cyno must be standard, industrial or covert, not %q", in.Cyno)
		}
		if in.LighterID == 0 {
			return nil, EstimateCynoOutput{}, fmt.Errorf("estimate cyno: lighterId is required")
		}
		p := newPlanner(version, clientID, store, sdeStore)
		ship, err := p.resolveShip(ctx, in.ShipSpec, in.LighterID)
		if err != nil {
			return nil, EstimateCynoOutput{}, fmt.Errorf("estimate cyno: %w", err)
		}
		// Evaluate with the generator fitted, whether or not the ship has one.
		modules := ship.modules
		has := false
		for _, m := range modules {
			has = has || m == gen
		}
		if !has {
			modules = append(append([]int64{}, modules...), gen)
		}
		skills, err := p.skills(ctx, in.LighterID)
		if err != nil {
			return nil, EstimateCynoOutput{}, err
		}
		f, err := p.fit(ctx, ship.hull, modules, skills)
		if err != nil {
			return nil, EstimateCynoOutput{}, err
		}
		gi := 0
		for i, m := range modules {
			if m == gen {
				gi = i
			}
		}
		genAttrs := f.Modules[gi].Attrs
		fuel := int64(genAttrs[attrCynoFuelType])
		basics, err := p.typeBasics(ctx, []int64{fuel, gen, ship.hull})
		if err != nil {
			return nil, EstimateCynoOutput{}, err
		}
		out := EstimateCynoOutput{Ship: ship.hullName, Generator: basics[gen].Name, Fuel: basics[fuel].Name,
			FuelPerLight: f.ModuleAttr(gi, attrCynoFuelPerCycle), CycleSeconds: f.ModuleAttr(gi, attrDuration) / 1000, Lights: in.Lights}
		out.FuelM3PerLight = out.FuelPerLight * basics[fuel].Volume
		out.CargoM3 = math.Max(f.ShipAttr(sde.AttrCapacity)-ship.cargoM3, 0)
		if out.FuelM3PerLight > 0 {
			out.LightsPerHold = int(math.Floor(out.CargoM3/out.FuelM3PerLight + 1e-9))
		}
		if in.Lights > 0 {
			out.FuelForLights = out.FuelPerLight * float64(in.Lights)
			if out.LightsPerHold > 0 {
				out.Refills = (in.Lights+out.LightsPerHold-1)/out.LightsPerHold - 1
			}
		}
		for _, r := range nav.Missing(skills, requiredSkills(genAttrs)) {
			out.MissingSkills = append(out.MissingSkills, p.skillLabel(ctx, r))
		}
		rule, err := p.fitRule(ctx, gen)
		if err != nil {
			return nil, EstimateCynoOutput{}, err
		}
		out.GeneratorNotFits = !rule.allows(ship.hull, basics[ship.hull].GroupID)
		return nil, out, nil
	}
}
