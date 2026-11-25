package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addFactionWarfareTools registers the faction warfare tools.
func addFactionWarfareTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetCharacterFwStats(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCorporationFwStats(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetFwLeaderboards(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetFwCharacterLeaderboards(s, d.openPublicESI)
	mcp.AddGetFwCorporationLeaderboards(s, d.openPublicESI)
	mcp.AddListFwStats(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListFwSystems(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListFwWars(s, d.userAgent, d.openPublicESI, d.sde)
}

// addCorporationProjectTools registers the corporation project tools.
func addCorporationProjectTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetCorporationProjectContribution(s, d.openESI, d.store)
	mcp.AddListCorporationProjectContributors(s, d.openESI, d.store)
	mcp.AddGetCorporationProjectDetail(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationProjects(s, d.openESI, d.store)
}

// addStructureTools registers the mercenary den, skyhook and sovereignty hub tools.
func addStructureTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterMercenaryDens(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCharacterMercenaryDen(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationSkyhooks(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCorporationSkyhook(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationSovereigntyHubs(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCorporationSovereigntyHub(s, d.userAgent, d.openESI, d.store, d.sde)
}
