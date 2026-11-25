package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addUniverseTools registers the universe tools.
func addUniverseTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListUniverseAncestries(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseBloodlines(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseRaces(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseFactions(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseCategories(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseCategory(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseGroups(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseGroup(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseTypes(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseType(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseRegions(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseRegion(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseConstellations(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseConstellation(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseSystems(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseSystem(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseAsteroidBelt(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseMoon(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniversePlanet(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseStargate(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseStar(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseStation(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseGraphics(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddGetUniverseGraphic(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListUniverseStructures(s, d.openPublicESI)
	mcp.AddListUniverseSystemJumps(s, d.openPublicESI)
	mcp.AddResolveUniverseIDs(s, d.openPublicESI)
}
