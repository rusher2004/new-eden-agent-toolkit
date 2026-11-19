---
name: haul-route-setup
description: Set up neat haul profiles and jump freighter or capital routes in EVE Online — hull, pilot, fitted hold, and routing preferences such as areas or alliances to avoid, friendly midpoints, cyno alts, and fuel versus jumps versus fatigue — and compare routes with dotlan links. Use when a plan needs hauling and no haul profile exists, when the user wants a jump route, or when they mention avoiding an area, a new cyno spot, or a politics change.
---

# Haul and route setup

neat plans jump and gate routes itself and returns dotlan links for each one. Your job is to learn the pilot's preferences, save them, and help them pick a route. Never work out distances, fuel or fatigue yourself.

## 1. Ship and pilot

- Ask which hull and which pilot. The pilot must be logged in to neat. Skills and contacts come from ESI.
- Ask for the fitted cargo hold in m³ as the game shows it. Don't assume an unfitted value.

## 2. Preferences

Ask in plain terms and save what they say:
- **Areas to stay out of:** systems, constellations or regions (`avoidSystems`, `avoidConstellations`, `avoidRegions`).
- **Whose space:** alliances to avoid (`avoidAlliances`). Offer `avoidBadStanding` to avoid sovereignty held by anyone at negative standing in their character, corp or alliance contacts. Ask about blue alliances to prefer (`preferAlliances`).
- **Midpoints:** systems where they can dock, have a cyno alt, or have a friendly POS. Ask whether jumps must land only there (`midpointsRequired`) or should just prefer them.
- **dotlan-style options:** prefer station systems, and avoid incursions.
- **What to optimise:** fuel (the default), number of jumps, or fatigue.
- **Gates:** a cap on gate count, and whether to skip gate systems with recent kills (`maxRecentKills`).

Call `save_haul_profile` with the answers.

## 3. Pick a route

- Call `plan_jump_route` from the hub to the destination. It returns several ranked routes: the best for their preferences, the fewest jumps, the least fatigue, and one ignoring preferred midpoints. Each has per-leg distance, fuel and fatigue, the security, owner and recent kills at each midpoint, any preference it breaks, and dotlan links.
- Show the routes side by side with their dotlan links, plus the `dotlanSuggestion` link so the pilot can compare dotlan's own pick. Sovereignty and standings preferences can't be expressed in a dotlan link; the tool says so in the link's label.
- When they pick one, save it as the pinned `outbound` (and `return`) route on the haul profile with `save_haul_profile`, as system names in order. A pinned route is only used for trips with the same start and end systems; for any other trip the planner searches a route and says so in its notes, so save a separate haul profile for each regular run.
- To check a route the pilot already flies, pass it as `waypoints` to `plan_jump_route`. It validates range and cyno-legal landings and costs it.

## 4. Keep it current

Offer to re-run this whenever politics change, a midpoint is lost, or the pilot trains a jump skill.
