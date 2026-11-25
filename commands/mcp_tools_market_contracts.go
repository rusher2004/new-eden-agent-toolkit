package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addMarketContractAssetTools registers the asset location, contract, market
// order and market data tools.
func addMarketContractAssetTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetAssetLocations(s, d.openESI, d.store)
	mcp.AddGetCorporationAssetLocations(s, d.openESI, d.store)

	mcp.AddListCharacterContracts(s, d.openESI, d.store)
	mcp.AddListCharacterContractBids(s, d.openESI, d.store)
	mcp.AddListCharacterContractItems(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationContracts(s, d.openESI, d.store)
	mcp.AddListCorporationContractBids(s, d.openESI, d.store)
	mcp.AddListCorporationContractItems(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListPublicContractBids(s, d.openPublicESI)

	mcp.AddListCharacterMarketOrders(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCharacterMarketOrderHistory(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationMarketOrders(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationMarketOrderHistory(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListMarketGroups(s, d.openPublicESI)
	mcp.AddGetMarketGroup(s, d.openPublicESI)
	mcp.AddGetMarketHistory(s, d.openPublicESI)
	mcp.AddListMarketTypes(s, d.userAgent, d.openPublicESI, d.sde)
}
