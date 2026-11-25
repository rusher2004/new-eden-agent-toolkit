package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addNavigationTools registers the haul profile, saved route, jump route and
// hauling tools. They gather SDE, ESI and config inputs through the planner,
// so each takes both ESI openers.
func addNavigationTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddSaveHaulProfile(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddListHaulProfiles(s, d.store)
	mcp.AddDeleteHaulProfile(s, d.store)
	mcp.AddSaveRoute(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddListRoutes(s, d.store)
	mcp.AddDeleteRoute(s, d.store)
	mcp.AddPlanJumpRoute(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddPlanHaul(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
}

// addLogisticsReadinessTools registers the tools that read what the user's
// characters and ships can do.
func addLogisticsReadinessTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetLogisticsOverview(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddListCharacterCapabilities(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddListJumpClones(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddListHaulShips(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddCompareHaulHulls(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddEstimateShipHold(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddEstimateCyno(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
}

// addSavedStructureTools registers the tools for the stations and structures
// the user works from.
func addSavedStructureTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListStructures(s, d.openESI, d.store)
	mcp.AddSaveStructure(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddDeleteStructure(s, d.store)
}

// addTradeHubTools registers the trade hub and purchase pricing tools.
func addTradeHubTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListHubProfiles(s, d.store)
	mcp.AddSaveHubProfile(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddDeleteHubProfile(s, d.store)
	mcp.AddValidateTradeHub(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddCompareTradeHubs(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddQuoteMarketPurchase(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
}

// addIndustryPlanningTools registers the facility profile and industry
// calculation tools.
func addIndustryPlanningTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddSaveFacilityProfile(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddListFacilityProfiles(s, d.store)
	mcp.AddDeleteFacilityProfile(s, d.store)
	mcp.AddDescribeFacility(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddCalibrateFacilityProfile(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddFindOwnedBlueprints(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddCheckBuildSkills(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddCheckMaterialStock(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddEstimateJobCost(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddEstimateResearch(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddEstimateInvention(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddEstimateReprocessingYield(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddSolveOreMix(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
	mcp.AddPlanBuild(s, d.userAgent, d.openESI, d.openPublicESI, d.store, d.sde)
}
