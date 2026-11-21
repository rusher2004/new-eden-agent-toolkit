---
name: haul-plan
description: Plan moving cargo, ships or characters in EVE Online with neat — by jump freighter, freighter, transport, Black Ops bridge or capital suitcasing, for a PvP deployment, a staging move, a PvE relocation or a build — including which characters haul and light cynos, cyno ships and Liquid Ozone, jump clones to get pilots in place, assembled ships, routes and routing preferences, fuel, fatigue and the order of operations. Use when the user wants to move assets, haul to or from staging, JF something, suitcase ships, set up a haul profile or jump route, or mentions avoiding an area, a new cyno spot or a politics change.
---

# Haul plan

Plan a move of the user's cargo, ships and characters. neat works out every number and checks every game rule: holds, volumes, fuel, fatigue, ranges, cyno fuel, docking, highsec entry, clone jump timing and who can fly or light what. Your job is to find out what the user wants, put the right tools in order, and help them make the choices that are theirs. Never calculate a number yourself; if one looks wrong, find the input that's off (a profile, a saved structure, a character).

## 1. Start from the overview

- Call `get_logistics_overview`. It gives every character's account, whether they're online, where they are, their ship, what they can light, bridge, conduit and fly, their next clone jump and their jump fatigue.
- If it warns about unknown accounts or reports a conflict, ask the user which characters share an account and save it with `update_character_details`. A pilot and the cyno lighter they jump to must be on different accounts. Don't pick characters until this is settled.
- Call `list_structures` for the places the user works from: what each is for and whether it has a clone bay.

## 2. Pin down the job

- Confirm the origin, the destination station or structure, what's moving (everything there or a subset), and constraints such as not gating cargo through somewhere.
- Resolve names to IDs with neat. If a station or structure the plan uses isn't saved, offer to save it with `save_structure`, along with anything the user says about it.
- Quote every system with the displayed security the tools give, never one you'd guess from the name.

## 3. Inventory

- Call `list_location_assets` at the origin for every character, and the corporation's hangars through a character with the roles. It lists what's inside each container and ship, packaged and assembled volumes, each assembled ship's repackaged size and the rigs repackaging destroys, and totals.
- Mention useful extras that belong to other characters, but don't add them unasked.

## 4. Assembled ships

Cargo holds only take packaged ships. Point out every assembled ship and let the user choose:

- **Repackage:** name the rigs `list_location_assets` says each would lose.
- **Suitcase:** carry them assembled in a capital's ship maintenance bay. Use `compare_haul_hulls` for bay sizes and docking, and the overview for who can fly a suitable capital and owns one nearby.
- **Fly them** by gate.

Drop whatever the user will handle themselves from the rest of the plan.

## 5. Ships and pilots

- Use `compare_haul_hulls` (with `pilotId` for each candidate) to compare holds, align, cloak and nullification, highsec entry, docking and jump drives.
- For the actual ships, use `estimate_ship_hold` with `shipItemId`: it reads the real fit and the cargo already aboard. Never ask the user for their hold.
- Consider more than one ship. Holds add up, so two jump freighters, or a JF and a carrier, can make one trip instead of several. Weigh the extra pilots, fuel and fatigue against the trips saved, and say which you chose and why.
- Check each pilot's jump fatigue in the overview.

## 6. Route and preferences

- Check `list_routes` for a saved route between the same places, and read its notes: it usually reflects choices the user made on purpose.
- Ask in plain terms, and save what they say on a haul profile with `save_haul_profile`, giving `shipItemId` so neat works out the hold:
  - **Areas to stay out of:** systems, constellations or regions.
  - **Whose space:** alliances to avoid, avoiding sovereignty held by anyone at negative standing, and blue alliances to prefer.
  - **Midpoints:** systems where they can dock, have a cyno alt or a friendly POS, and whether jumps must land only there.
  - **dotlan-style options:** prefer station systems, avoid incursions.
  - **What to optimise:** fuel (the default), number of jumps, or fatigue.
  - **Gates:** a cap on gate count, and whether to skip gate systems with recent kills.
- Call `plan_jump_route`, or `plan_haul` for cargo (with `with` for more ships flying together). Read each route's `checks`:
  - a landing with `noDock` has nowhere known to dock: ask whether the user has a structure there, and save it if so;
  - `fuel` says whether the fuel fits the fuel bay or needs cargo space;
  - `cynoTypes` (or `sharedCynoTypes` for several ships) is what the lighters must light.
- Show the routes side by side with their dotlan links and the `dotlanSuggestion`. Sovereignty and standings preferences can't go in a dotlan link; the tool says so in the link's label.
- When they pick a route, save it with `save_route` and their reasons as `notes`, and pin it on the haul profile.

## 7. Cynos

- **Lighters:** characters whose `list_character_capabilities` include the cyno type needed, on a different account from the pilots jumping to them.
- **Fuel:** `estimate_cyno` with the lighter, their ship and the number of lights (one per jump leg) gives Liquid Ozone per light, lights per hold and refills, whether the generator fits, and any skills they're missing.
- **Ship:** compare sources before choosing: a cyno ship already near the cyno point or the lighter's location or jump clones (`list_location_assets`), one fitted from assets, or one bought (`compare_trade_hubs` for the nearest hub that stocks the fit, `quote_market_purchase` for the list).
- **Implants:** a lit cyno is an easy target. `list_jump_clones` shows each clone's implants; prefer one with none or cheap ones.
- **One lighter or several:** with one lighter, the hauler waits while the lighter gates to each next cyno point. The route's `lighterGatesFromPreviousLanding` says how far that is. Weigh it against getting another lighter in place, and say which you chose and why. Don't bring in a character only so two parts of the job run at once.

## 8. Get characters in place

- Check `list_jump_clones` before planning gate travel: where each clone is, its implants, whether the character can clone jump now and when they next can, including the trip back.
- A clone jump has to start where there's a clone bay. If `list_structures` says it's unknown, ask the user to check in game and save the answer with `save_structure`. Don't assume either way.
- A clone jump arrives in a capsule. Any step there beyond flying out in a pod needs a ship already at that station, owned by the character or contracted to them.

## 9. Order of operations

- Contracts can be made and accepted from anywhere. Only put a contract in the order when something in it is used by a later step.
- Write numbered steps saying who does what, in which ship, at which station: cyno cycles, refills, docking and unloading included.
- Totals (fuel, fatigue, gates, volumes, cost) come from the tools' outputs.

## 10. Present the plan

- Lead with the outcome: how many trips, who hauls, who lights.
- Write systems with their displayed security, colored with the colors the tools give in formatted output.
- Say which choices you made and why: one lighter or several, one ship or several, where the cyno ship comes from.
- End with the decisions that are the user's and anything neat couldn't confirm, such as an unknown clone bay.
- When you learn something durable (a structure's clone bay, why a route goes a certain way, which account a character is on), record it with `save_structure`, `save_route` or `update_character_details`, and tell the user what you saved.

## Keep it current

Offer to re-run route planning whenever politics change, a midpoint or structure is lost, or a pilot trains a jump or cyno skill.
