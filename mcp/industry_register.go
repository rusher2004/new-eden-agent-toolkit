package mcp

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// registerIndustryPlanningTools adds the industry planning tools. See
// docs/superpowers/specs/2026-11-18-industry-planning-design.md.
func registerIndustryPlanningTools(server *mcpsdk.Server, version, clientID string, store *config.Store, sdeStore *sde.Store) {
	add := func(name, desc string) *mcpsdk.Tool { return &mcpsdk.Tool{Name: name, Description: desc} }

	// Facility profiles (the only tools here that write anything, and only to ~/.neat/facilities.yaml).
	mcpsdk.AddTool(server, add("save_facility_profile", "Save (or replace) a named industry or reprocessing facility profile in neat's local config: NPC station, Upwell structure or starbase, its system, fitted rigs and services, and owner-set taxes. Rig and structure bonuses are read from the SDE; the response shows every resolved multiplier. Writes to ~/.neat/facilities.yaml only; omit notes to keep the notes it has."), saveFacilityProfileHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("list_facility_profiles", "List saved facility profiles."), listFacilityProfilesHandler(store))
	mcpsdk.AddTool(server, add("delete_facility_profile", "Delete a saved facility profile from neat's local config."), deleteProfileHandler(store, (*config.Config).DeleteFacilityProfile))
	mcpsdk.AddTool(server, add("describe_facility", "Show a facility profile's security band and every material, time and cost multiplier with its source, and optionally, for a product, the combined multipliers and whether each activity can run there."), describeFacilityHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("calibrate_facility_profile", "Read a real industry job's charged fee, back out the effective fee rate, and save it on a facility profile so later estimates use it. Writes to ~/.neat/facilities.yaml only."), calibrateFacilityProfileHandler(version, clientID, store, sdeStore))

	// Calculation primitives (read-only).
	mcpsdk.AddTool(server, add("find_owned_blueprints", "Find a character's or corporation's blueprints by product or name, with ME, TE, runs, whether each is tied up in an active job, the station or structure it's in and any named container, plus per-blueprint totals."), findOwnedBlueprintsHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("check_build_skills", "For each character, show manufacturing, reaction and science slots, skill time multipliers, and which items in a product's build tree they can build or which skills they lack."), checkBuildSkillsHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("check_material_stock", "Net a list of materials against what a character or corporation holds at a station or structure (optionally one hangar division), with the volume of stock used."), checkMaterialStockHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("estimate_job_cost", "Estimate a manufacturing or reaction job's installation fee at a facility profile: estimated item value, cost index, structure bonus, taxes, or the profile's calibrated rate."), estimateJobCostHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("estimate_research", "Estimate ME research, TE research or copying time and fee for a blueprint at a facility profile with a character's skills."), estimateResearchHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("estimate_invention", "For a T2 item, estimate invention chance, output copy (runs, ME, TE) and expected cost per success and per run for no decryptor and each decryptor, with a character's skills and hub prices."), estimateInventionHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("estimate_reprocessing_yield", "Reprocessing yield per ore category for a character at a facility profile (NPC station or Upwell), counting skills, structure, rigs, security and the character's implant, plus the tax."), estimateReprocessingYieldHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("solve_ore_mix", "Find the cheapest mix of compressed ore and refined minerals at a hub to cover a mineral need, counting the refiner's yield, freight per m³ and live order-book depth; returns purchases, refined output, surplus, byproducts and a Multibuy list."), solveOreMixHandler(version, clientID, store, sdeStore))

	// Orchestrator.
	mcpsdk.AddTool(server, add("plan_build", "Plan building owned blueprints end to end: expand the build tree with your copies' ME and the facility's bonuses, net stock, price purchases on live order books, optionally buy compressed ore and refine it, compare building against buying every intermediate, price job fees, pack haul loads with fuel and dotlan route links, and estimate job time across builders. Returns cost, decisions, shopping and Multibuy lists, loads, jobs and assumptions, and saves the full plan under ~/.neat/plans/."), planBuildHandler(version, clientID, store, sdeStore))
}
