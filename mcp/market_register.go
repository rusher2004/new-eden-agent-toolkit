package mcp

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// registerMarketPlanningTools adds the trade hub and purchase pricing tools,
// which serve any purchase, not only build materials. See
// docs/superpowers/specs/2026-11-20-logistics-planning-design.md.
func registerMarketPlanningTools(server *mcpsdk.Server, version, clientID string, store *config.Store, sdeStore *sde.Store) {
	add := func(name, desc string) *mcpsdk.Tool { return &mcpsdk.Tool{Name: name, Description: desc} }

	// Hub profiles (written to ~/.neat/config.json only).
	mcpsdk.AddTool(server, add("save_hub_profile", "Save (or replace) a named trade hub (region and station) to buy at. Writes to ~/.neat/config.json only."), saveHubProfileHandler(store))
	mcpsdk.AddTool(server, add("list_hub_profiles", "List saved trade hub profiles and the default (Jita 4-4)."), listHubProfilesHandler(store))

	// Pricing (read-only).
	mcpsdk.AddTool(server, add("quote_market_purchase", "Price a shopping list against a trade hub's live sell order book, walking depth for the full quantity; returns average and total per item, shortfalls, volume, and an in-game Multibuy list."), quoteMarketPurchaseHandler(version, clientID, store, sdeStore))
}
