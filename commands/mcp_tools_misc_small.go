package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addAccessListTools registers the access list tools.
func addAccessListTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListAccessLists(s, d.openESI, d.store)
	mcp.AddGetAccessList(s, d.openESI, d.store)
}

// addActivityTools registers the mercenary tactical operation and skyhook tools.
func addActivityTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListMercenaryTacticalOperations(s, d.openESI, d.store)
	mcp.AddGetMercenaryTacticalOperation(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListRaidableSkyhooks(s, d.userAgent, d.openPublicESI, d.store, d.sde)
}

// addAllianceTools registers the alliance tools.
func addAllianceTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListAlliances(s, d.openPublicESI)
	mcp.AddGetAlliance(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListAllianceCorporations(s, d.openPublicESI)
	mcp.AddGetAllianceIcons(s, d.openPublicESI)
}

// addCalendarTools registers the calendar tools.
func addCalendarTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCalendarEvents(s, d.openESI, d.store)
	mcp.AddGetCalendarEvent(s, d.openESI, d.store)
	mcp.AddListCalendarEventAttendees(s, d.openESI, d.store)
	mcp.AddRespondCalendarEvent(s, d.openESI, d.store)
}

// addDogmaTools registers the dogma tools.
func addDogmaTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListDogmaAttributes(s, d.openPublicESI)
	mcp.AddGetDogmaAttribute(s, d.openPublicESI)
	mcp.AddGetDogmaDynamicItem(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListDogmaEffects(s, d.openPublicESI)
	mcp.AddGetDogmaEffect(s, d.openPublicESI)
}

// addInsuranceTools registers the insurance tool.
func addInsuranceTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListInsurancePrices(s, d.userAgent, d.openPublicESI, d.sde)
}

// addKillmailTools registers the killmail tools.
func addKillmailTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterKillmailsRecent(s, d.openESI, d.store)
	mcp.AddListCorporationKillmailsRecent(s, d.openESI, d.store)
	mcp.AddGetKillmail(s, d.userAgent, d.openPublicESI, d.sde)
}

// addMetaTools registers the ESI meta tools.
func addMetaTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetEsiChangelog(s, d.openPublicESI)
	mcp.AddListEsiCompatibilityDates(s, d.openPublicESI)
	mcp.AddGetEsiName(s, d.openPublicESI)
	mcp.AddGetEsiStatus(s, d.openPublicESI)
}

// addPlanetaryInteractionTools registers the planetary interaction tools.
func addPlanetaryInteractionTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterPlanets(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetCharacterPlanet(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationCustomsOffices(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetPlanetarySchematic(s, d.openPublicESI)
}

// addRouteTools registers the route tool.
func addRouteTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetRoute(s, d.userAgent, d.openPublicESI, d.sde)
}

// addSearchTools registers the character search tool.
func addSearchTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddSearchCharacter(s, d.userAgent, d.openESI, d.store, d.sde)
}

// addStatusTools registers the server status tool.
func addStatusTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetStatus(s, d.openPublicESI)
}
