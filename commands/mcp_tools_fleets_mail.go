package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addFittingsTools registers the ship fitting tools.
func addFittingsTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterFittings(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddCreateCharacterFitting(s, d.openESI, d.store)
	mcp.AddDeleteCharacterFitting(s, d.openESI, d.store)
}

// addFleetTools registers the fleet tools.
func addFleetTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetCharacterFleet(s, d.openESI, d.store)
	mcp.AddGetFleet(s, d.openESI, d.store)
	mcp.AddUpdateFleet(s, d.openESI, d.store)
	mcp.AddListFleetMembers(s, d.userAgent, d.openESI, d.store, d.sde)
	mcp.AddInviteFleetMember(s, d.openESI, d.store)
	mcp.AddMoveFleetMember(s, d.openESI, d.store)
	mcp.AddKickFleetMember(s, d.openESI, d.store)
	mcp.AddListFleetWings(s, d.openESI, d.store)
	mcp.AddCreateFleetWing(s, d.openESI, d.store)
	mcp.AddRenameFleetWing(s, d.openESI, d.store)
	mcp.AddDeleteFleetWing(s, d.openESI, d.store)
	mcp.AddCreateFleetSquad(s, d.openESI, d.store)
	mcp.AddRenameFleetSquad(s, d.openESI, d.store)
	mcp.AddDeleteFleetSquad(s, d.openESI, d.store)
}

// addMailTools registers the mail tools.
func addMailTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListCharacterMail(s, d.openESI, d.store)
	mcp.AddGetCharacterMailLabels(s, d.openESI, d.store)
	mcp.AddListCharacterMailLists(s, d.openESI, d.store)
	mcp.AddGetCharacterMail(s, d.openESI, d.store)
	mcp.AddSendCharacterMail(s, d.openESI, d.store)
	mcp.AddCreateCharacterMailLabel(s, d.openESI, d.store)
	mcp.AddUpdateCharacterMail(s, d.openESI, d.store)
	mcp.AddDeleteCharacterMail(s, d.openESI, d.store)
	mcp.AddDeleteCharacterMailLabel(s, d.openESI, d.store)
}

// addUserInterfaceTools registers the in-game user interface tools.
func addUserInterfaceTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddSetAutopilotWaypoint(s, d.openESI, d.store)
	mcp.AddOpenContractWindow(s, d.openESI, d.store)
	mcp.AddOpenInformationWindow(s, d.openESI, d.store)
	mcp.AddOpenMarketDetailsWindow(s, d.openESI, d.store)
	mcp.AddOpenNewMailWindow(s, d.openESI, d.store)
}
