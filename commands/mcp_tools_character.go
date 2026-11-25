package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addCharacterInfoTools registers the character info tools: public lookups
// (character, corporation history, portrait) and a logged-in character's
// research agents, medals, notifications, roles, standings, titles and CSPA
// cost.
func addCharacterInfoTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetCharacter(s, d.userAgent, d.openPublicESI, d.sde)
	mcp.AddListCharacterCorporationHistory(s, d.openPublicESI)
	mcp.AddGetCharacterPortrait(s, d.openPublicESI)
	mcp.AddListCharacterAgentsResearch(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCharacterMedals(s, d.openESI, d.store)
	mcp.AddListCharacterNotifications(s, d.openESI, d.store)
	mcp.AddListCharacterNotificationContacts(s, d.openESI, d.store)
	mcp.AddGetCharacterRoles(s, d.openESI, d.store)
	mcp.AddListCharacterStandings(s, d.openESI, d.store)
	mcp.AddListCharacterTitles(s, d.openESI, d.store)
	mcp.AddGetCharacterCSPACost(s, d.openESI, d.store)
}

// addContactWriteTools registers the contact label tools and the tools that
// change a character's contact list.
func addContactWriteTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterContactLabels(s, d.openESI, d.store)
	mcp.AddListCorporationContactLabels(s, d.openESI, d.store)
	mcp.AddListAllianceContactLabels(s, d.openESI, d.store)
	mcp.AddAddCharacterContacts(s, d.openESI, d.store)
	mcp.AddUpdateCharacterContacts(s, d.openESI, d.store)
	mcp.AddRemoveCharacterContacts(s, d.openESI, d.store)
}

// addMiningAndFacilityTools registers the mining ledger tools and the public
// industry facility list.
func addMiningAndFacilityTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterMining(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationMiningExtractions(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationMiningObservers(s, d.openESI, d.store)
	mcp.AddGetCorporationMiningObserverDetail(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListIndustryFacilities(s, d.userAgent, d.openPublicESI, d.sde)
}
