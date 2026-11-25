package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addSovereigntyTools registers the sovereignty campaign tool.
func addSovereigntyTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddListSovereigntyCampaigns(s, d.userAgent, d.openPublicESI, d.sde)
}
