package commands

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// addSkillTools registers the character attribute and skill queue tools.
func addSkillTools(s *mcpsdk.Server, d toolDeps) {
	mcp.AddGetCharacterAttributes(s, d.openESI, d.store)
	mcp.AddListCharacterSkillQueue(s, d.openESI, d.store)
}
