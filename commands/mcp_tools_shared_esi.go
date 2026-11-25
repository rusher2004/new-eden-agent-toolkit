package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addAssetTools registers the asset tools.
func addAssetTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListAssets(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetAssetNames(s, d.openESI, d.store)
	mcp.AddListCorporationAssets(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCorporationAssetNames(s, d.openESI, d.store)
	mcp.AddListLocationAssets(s, d.userAgent, d.openESI, d.store, d.sde)
}

// addCharacterStatusTools registers the character location, ship, online,
// clone, implant, fatigue, skill and affiliation tools.
func addCharacterStatusTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetCharacterAffiliations(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetCharacterLocation(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCharacterOnline(s, d.openESI, d.store)
	mcp.AddGetCharacterShip(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCharacterClones(s, d.openESI, d.store)
	mcp.AddListCharacterImplants(s, d.openESI, d.store)
	mcp.AddGetCharacterFatigue(s, d.openESI, d.store)
	mcp.AddListCharacterSkills(s, d.openESI, d.store)
}

// addContactTools registers the character, corporation and alliance contact
// list tools.
func addContactTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterContacts(s, d.openESI, d.store)
	mcp.AddListCorporationContacts(s, d.openESI, d.store)
	mcp.AddListAllianceContacts(s, d.openESI, d.store)
}

// addBlueprintAndJobTools registers the blueprint and industry job tools.
func addBlueprintAndJobTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterBlueprints(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationBlueprints(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCharacterIndustryJobs(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationIndustryJobs(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListIndustrySystems(s, d.userAgent, d.openPublicESI, d.sde)
}

// addMarketDataTools registers the contract, market, sovereignty, incursion
// and universe tools that other tools also read through shared functions.
func addMarketDataTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListPublicContracts(s, d.openPublicESI)
	mcp.AddListPublicContractItems(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListMarketPrices(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListMarketOrders(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListStructureMarketOrders(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListSovereigntySystems(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListIncursions(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseStructure(s, d.openESI, d.store)
	mcp.AddListUniverseSystemKills(s, d.openPublicESI)
	mcp.AddResolveUniverseNames(s, d.openPublicESI)
}
