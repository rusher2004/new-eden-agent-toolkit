package commands

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// toolDeps are the concrete producers this composition root builds once and
// hands to each converted tool. The mcp package never names these types: each
// Add function takes only the small interface its tool needs, and the value
// here satisfies it. A value with far more methods than the interface lists is
// the point, as with an *http.Client passed as a one-method doer.
type toolDeps struct {
	userAgent string
	store     *config.Store
	sde       *sde.Store
	clientID  string
}

// openESI builds an authenticated ESI client for a logged-in character.
func (d toolDeps) openESI(ctx context.Context, characterID int32) (*esi.Client, error) {
	c, err := esi.NewClient(ctx, d.clientID, characterID, d.userAgent)
	if err != nil {
		return nil, fmt.Errorf("character %d: %w", characterID, err)
	}
	return c, nil
}

// openPublicESI builds an ESI client for endpoints that need no character.
func (d toolDeps) openPublicESI() *esi.Client { return esi.NewPublicClient(d.userAgent) }

// NewMCPServer builds neat's MCP server with every tool registered, one
// explicit Add call each (addConvertedTools). sdeStore may be nil: every
// sde.Store method is nil-safe, so name enrichment and the Universe tools fall
// through to ESI. A new tool is its own file in mcp/ plus one line below.
func NewMCPServer(version, clientID string, store *config.Store, sdeStore *sde.Store) *mcpsdk.Server {
	server := mcp.NewServer(version)
	addConvertedTools(server, toolDeps{userAgent: sso.UserAgent(version), store: store, sde: sdeStore, clientID: clientID})
	return server
}

// addConvertedTools registers every tool, grouped by area: the operations
// (Operations) through mcp.Register, and the tools not yet moved to a domain
// package in the pattern of mcp/access_list.go. Each call passes the concrete values the tool's interfaces
// ask for; the compiler checks that they satisfy them.
func addConvertedTools(s *mcpsdk.Server, d toolDeps) {
	mcp.Register(s, Operations(d))
	addFittingsTools(s, d)
	addFleetTools(s, d)
	addMailTools(s, d)
	addUserInterfaceTools(s, d)
	addWalletTools(s, d)
	addFreelanceJobTools(s, d)
	addWarTools(s, d)
	addMilitaryCampaignTools(s, d)
	addCosmeticsTools(s, d)
	addParagonHubTools(s, d)
	addAccessListTools(s, d)
	addActivityTools(s, d)
	addAllianceTools(s, d)
	addCalendarTools(s, d)
	addDogmaTools(s, d)
	addInsuranceTools(s, d)
	addKillmailTools(s, d)
	addMetaTools(s, d)
	addPlanetaryInteractionTools(s, d)
	addRouteTools(s, d)
	addSearchTools(s, d)
	addStatusTools(s, d)
	addFactionWarfareTools(s, d)
	addCorporationProjectTools(s, d)
	addStructureTools(s, d)
	addAssetTools(s, d)
	addCharacterStatusTools(s, d)
	addContactTools(s, d)
	addBlueprintAndJobTools(s, d)
	addMarketDataTools(s, d)
	addNavigationTools(s, d)
	addLogisticsReadinessTools(s, d)
	addSavedStructureTools(s, d)
	addTradeHubTools(s, d)
	addIndustryPlanningTools(s, d)
	addCorporationTools(s, d)
	addSovereigntyTools(s, d)
	addSkillTools(s, d)
	addUniverseTools(s, d)
	addCharacterInfoTools(s, d)
	addContactWriteTools(s, d)
	addMiningAndFacilityTools(s, d)
	addMarketContractAssetTools(s, d)
	addAdminTools(s, d)
}
