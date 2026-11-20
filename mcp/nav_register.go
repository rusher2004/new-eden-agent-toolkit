package mcp

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// registerNavigationTools adds the jump route and hauling tools, which serve
// any move through New Eden (PvP, PvE or industry), not only builds. See
// docs/superpowers/specs/2026-11-20-logistics-planning-design.md.
func registerNavigationTools(server *mcpsdk.Server, version, clientID string, store *config.Store, sdeStore *sde.Store) {
	add := func(name, desc string) *mcpsdk.Tool { return &mcpsdk.Tool{Name: name, Description: desc} }

	// Haul profiles (written to ~/.neat/config.json only).
	mcpsdk.AddTool(server, add("save_haul_profile", "Save (or replace) a named hauling profile: hull, pilot, fitted hold, optional pinned routes, and routing preferences (avoid lists, sovereignty and standings, midpoints, optimise for fuel, jumps or fatigue). Writes to ~/.neat/config.json only."), saveHaulProfileHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("list_haul_profiles", "List saved haul profiles."), listHaulProfilesHandler(store))
	mcpsdk.AddTool(server, add("delete_haul_profile", "Delete a saved haul profile from neat's local config."), deleteProfileHandler(store, (*config.Config).DeleteHaulProfile))

	// Routing and hauling (read-only).
	mcpsdk.AddTool(server, add("plan_jump_route", "Plan jump and gate routes between two systems for a haul profile's ship and pilot, applying their routing preferences, or validate and cost a given route; per leg light years, fuel and fatigue, and dotlan links that display each route."), planJumpRouteHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("plan_haul", "Pack a cargo list into loads for a haul profile's hold, each an even slice with volume by drop-off, plus round-trip route, fuel per trip, and a Multibuy list per load including fuel."), planHaulHandler(version, clientID, store, sdeStore))
}
