package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addWalletTools registers the character and corporation wallet tools.
func addWalletTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetWalletBalance(s, d.openESI, d.store)
	mcp.AddListCharacterWalletJournal(s, d.openESI, d.store)
	mcp.AddListCharacterWalletTransactions(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddListCorporationWallets(s, d.openESI, d.store)
	mcp.AddListCorporationWalletJournal(s, d.openESI, d.store)
	mcp.AddListCorporationWalletTransactions(s, d.userAgent, d.openESI, d.store, d.sde)
}

// addFreelanceJobTools registers the freelance job tools.
func addFreelanceJobTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterFreelanceJobs(s, d.openESI, d.store)
	mcp.AddGetCharacterFreelanceJobParticipation(s, d.openESI, d.store)
	mcp.AddListCorporationFreelanceJobs(s, d.openESI, d.store)
	mcp.AddListCorporationFreelanceJobParticipants(s, d.openESI, d.store)
	mcp.AddGetFreelanceJob(s, d.openPublicESI)
	mcp.AddListFreelanceJobs(s, d.openPublicESI)
}

// addWarTools registers the war tools.
func addWarTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListWars(s, d.openPublicESI)
	mcp.AddGetWar(s, d.openPublicESI)
	mcp.AddListWarKillmails(s, d.openPublicESI)
}

// addMilitaryCampaignTools registers the military campaign tools.
func addMilitaryCampaignTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListMilitaryCampaigns(s, d.openPublicESI)
	mcp.AddGetMilitaryCampaign(s, d.openPublicESI)
	mcp.AddListMilitaryCampaignObjectives(s, d.openPublicESI)
	mcp.AddGetMilitaryCampaignObjective(s, d.openPublicESI)
	mcp.AddListCharacterMilitaryCampaignObjectives(s, d.openESI, d.store)
	mcp.AddGetCharacterMilitaryCampaignObjectiveParticipation(s, d.openESI, d.store)
}

// addCosmeticsTools registers the SKINR tools.
func addCosmeticsTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterSkinrLicenses(s, d.openESI, d.store)
	mcp.AddListCharacterSkinrComponentLicenses(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddGetSkinr(s, d.userAgent, d.openPublicESI, d.sde)
}

// addParagonHubTools registers the Paragon Hub tools.
func addParagonHubTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListParagonHubSkinrListings(s, d.openPublicESI)
	mcp.AddListCharacterParagonHubSkinrListings(s, d.openESI, d.store)
	mcp.AddListParagonHubSkinrListingsForAlliance(s, d.openESI, d.store)
	mcp.AddListParagonHubSkinrListingsForCorporation(s, d.openESI, d.store)
	mcp.AddListParagonHubSkinrListingsForCharacter(s, d.openESI, d.store)
}
