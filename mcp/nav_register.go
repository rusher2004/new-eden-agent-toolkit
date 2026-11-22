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

	// Haul profiles and saved routes (written to ~/.neat/haul_profiles.yaml and routes.yaml only).
	mcpsdk.AddTool(server, add("save_haul_profile", "Save (or replace) a named hauling profile: hull, pilot, fitted hold, optional pinned routes (saved in routes.yaml, or saved routes by name), and routing preferences (avoid lists, sovereignty and standings, midpoints, optimise for fuel, jumps or fatigue). Omit notes to keep the notes it has. Writes to ~/.neat/haul_profiles.yaml and routes.yaml only."), saveHaulProfileHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("list_haul_profiles", "List saved haul profiles."), listHaulProfilesHandler(store))
	mcpsdk.AddTool(server, add("save_route", "Save (or replace) a named jump route: its systems in order, the kind of ship it's for, and the user's notes on why it goes that way. Check list_routes before planning a route between the same places, since a saved route usually reflects choices the user made on purpose. plan_jump_route costs a saved route by name, and haul profiles pin saved routes for their regular runs. Returns every value it changed. Writes to ~/.neat/routes.yaml only."), saveRouteHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("list_routes", "List saved jump routes with their systems, ship class and the user's notes. Check it before planning a route between the same places."), listRoutesHandler(store))
	mcpsdk.AddTool(server, add("delete_route", "Delete a saved jump route from ~/.neat/routes.yaml."), deleteProfileHandler(store, (*config.Config).DeleteRoute))
	mcpsdk.AddTool(server, add("delete_haul_profile", "Delete a saved haul profile from neat's local config."), deleteProfileHandler(store, (*config.Config).DeleteHaulProfile))

	// Routing and hauling (read-only).
	mcpsdk.AddTool(server, add("plan_jump_route", "Plan jump and gate routes between two systems for a haul profile's ship and pilot, applying their routing preferences, or validate and cost a given route; per leg light years, fuel and fatigue, and dotlan links that display each route."), planJumpRouteHandler(version, clientID, store, sdeStore))
	mcpsdk.AddTool(server, add("plan_haul", "Pack a cargo list into loads for a haul profile's hold, each an even slice with volume by drop-off, plus round-trip route, fuel per trip, and a Multibuy list per load including fuel."), planHaulHandler(version, clientID, store, sdeStore))
}
