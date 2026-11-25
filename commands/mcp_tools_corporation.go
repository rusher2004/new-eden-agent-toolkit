package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addCorporationTools registers the corporation tools.
func addCorporationTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetCorporation(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListCorporationAllianceHistory(s, d.openPublicESI)
	mcp.AddGetCorporationIcons(s, d.openPublicESI)
	mcp.AddListNPCCorporations(s, d.openPublicESI)
	mcp.AddListCorporationMembers(s, d.openESI, d.store)
	mcp.AddGetCorporationMembersLimit(s, d.openESI, d.store)
	mcp.AddListCorporationMemberTitles(s, d.openESI, d.store)
	mcp.AddListCorporationMemberTracking(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationRoles(s, d.openESI, d.store)
	mcp.AddListCorporationRolesHistory(s, d.openESI, d.store)
	mcp.AddListCorporationTitles(s, d.openESI, d.store)
	mcp.AddGetCorporationDivisions(s, d.openESI, d.store)
	mcp.AddListCorporationShareholders(s, d.openESI, d.store)
	mcp.AddListCorporationStandings(s, d.openESI, d.store)
	mcp.AddListCorporationContainerLogs(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationFacilities(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationMedals(s, d.openESI, d.store)
	mcp.AddListCorporationMedalsIssued(s, d.openESI, d.store)
	mcp.AddListCorporationStarbases(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCorporationStarbase(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationStructures(s, d.userAgent, d.openESI, d.store, d.sde)
}
