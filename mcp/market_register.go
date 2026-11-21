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

	// Trade hubs (written to ~/.neat/hubs.yaml only).
	mcpsdk.AddTool(server, add("list_hub_profiles", "List the trade hubs to buy and sell at (~/.neat/hubs.yaml): the five NPC hubs, the player markets near Jita and the Amamake Keepstar to start with, plus any the user saved. Each has its kind (NPC station or player structure), tier, system and region, what it's good for, its latest validation, and whether that validation is stale. The first hub is the default for pricing."), listHubProfilesHandler(store))
	mcpsdk.AddTool(server, add("save_hub_profile", "Save a trade hub, or update one: its name, system and region are read from the game (a player structure needs a character with docking access, which neat then uses to read its market). Records the user's tier, traits and notes. Returns every value it changed. Writes to ~/.neat/hubs.yaml only."), saveHubProfileHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("delete_hub_profile", "Delete a trade hub from ~/.neat/hubs.yaml. A built-in hub the user deletes stays deleted."), deleteHubProfileHandler(store))
	mcpsdk.AddTool(server, add("validate_trade_hub", "Measure how good a trade hub is on live ESI data and record the dated result on it: how many of a basket of 24 everyday goods it has for sale (or a basket you give), with its sell and buy order counts, and optionally its public contracts by value with the capital ship groups on offer (the check for a capital market). Works for a saved hub or a new station or structure ID, which it then saves. Run it before relying on a hub whose validation is stale. Writes to ~/.neat/hubs.yaml only."), validateTradeHubHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("compare_trade_hubs", "Rank trade hubs for a shopping list: how many items each supplies in full, the total walking each hub's live order book, and, with from, the gate jumps from a system such as the staging system. Use it to find the nearest hub that actually stocks what a plan needs."), compareTradeHubsHandler(version, clientID, store, sdeStore))

	// Pricing (read-only).
	mcpsdk.AddTool(server, add("quote_market_purchase", "Price a shopping list against a trade hub's live sell order book, walking depth for the full quantity; returns average and total per item, shortfalls, volume, and an in-game Multibuy list."), quoteMarketPurchaseHandler(version, clientID, store, sdeStore))
}
