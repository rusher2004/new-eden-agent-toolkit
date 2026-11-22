---
name: haul-plan
description: Plan moving cargo, ships or characters in EVE Online with neat — by jump freighter, freighter, transport, Black Ops bridge or capital suitcasing, for a PvP deployment, a staging move, a PvE relocation or a build — including which characters haul and light cynos, cyno ships and Liquid Ozone, jump clones to get pilots in place, assembled ships, routes and routing preferences, fuel, fatigue and the order of operations. Use when the user wants to move assets, haul to or from staging, JF something, suitcase ships, set up a haul profile or jump route, or mentions avoiding an area, a new cyno spot or a politics change.
---

# Haul plan

Plan a move of the user's cargo, ships and characters. neat works out every number and checks every game rule: holds, volumes, fuel, fatigue, ranges, cyno fuel, docking, highsec entry, clone jump timing and who can fly or light what. Your job is to find out what the user wants, put the right tools in order, and help them make the choices that are theirs. Never calculate a number yourself; if one looks wrong, find the input that's off (a profile, a saved structure, a character).

When another skill (such as **industry-build-plan**) sends you here only for a missing haul profile, do steps 5 and 6 for that ship and pilot, then go back to it.

## 1. Start from the overview

- Call `get_logistics_overview`. It gives every character's account, whether they're online, where they are, their ship, what they can light, bridge, conduit and fly, their next clone jump and their jump fatigue.
- If it warns about unknown accounts or reports a conflict, ask the user which characters share an account and save it with `update_character_details`. A pilot and the cyno lighter they jump to must be on different accounts. Don't pick characters until this is settled.
- Call `list_haul_ships` to see where every hauler, jump freighter, freighter, transport, Rorqual, Black Ops, carrier and other capital is parked, and every ship with a cyno generator fitted, with the station or system each is in and the ship carrying it if any. Never page through `list_assets` to find them.
- Call `list_structures` for the places the user works from: what each is for and whether it has a clone bay.

## 2. Pin down the job

- Confirm the origin, the destination station or structure, what's moving (everything there or a subset), and constraints such as not gating cargo through somewhere.
- Note every constraint the user has already stated ("JF it so nothing takes a gate", "stay out of Delve", "not through Goon space"). They become routing preferences in step 6 without asking again.
- Resolve names to IDs with neat. If a station or structure the plan uses isn't saved, offer to save it with `save_structure`, along with anything the user says about it.
- Quote every system with the displayed security the tools give, never one you'd guess from the name.

## 3. Inventory

- Call `list_location_assets` at the origin for every character, and the corporation's hangars through a character with the roles. It lists what's inside each container and ship, packaged and assembled volumes, each assembled ship's repackaged size and the rigs repackaging destroys, and totals.
- `sources` gives each owner's share of the totals. Move only what the user asked to move: mention useful extras that belong to other characters (boosters, fuel, modules), but don't add them to the cargo or the steps unless the user says yes.

## 4. Assembled ships

Cargo holds only take packaged ships. Point out every assembled ship and let the user choose:

- **Repackage:** name the rigs `list_location_assets` says each would lose.
- **Suitcase:** carry them assembled in a capital's ship maintenance bay. Use `compare_haul_hulls` for bay sizes and docking, and the overview for who can fly a suitable capital and owns one nearby.
- **Fly them** by gate.

This choice is the user's, so it never goes into the steps before they make it:

- Until they choose, leave the steps that depend on it out of the plan, or show each option as its own short branch (what it costs, what it takes, which steps it adds). Never write one option in as a numbered step and ask about it afterwards.
- Once the user takes something on themselves ("I'll fly the assembled ships"), remove it from the plan entirely: its steps, its purchases (shuttles, fuel), its trips and its place in the order. Don't plan it for them unless they ask. At most, say in one line what they're handling.

## 5. Ships and pilots

- Use `compare_haul_hulls` (with `pilotId` for each candidate) to compare holds, align, cloak and nullification, highsec entry, docking and jump drives.
- For the actual ships, use `estimate_ship_hold` with the `itemId` and `ownerId` from `list_haul_ships` as `shipItemId` and `ownerId`: it reads the real fit and the cargo already aboard. A corporation's ship has no `ownerId`; give its `type` and `modules` as `shipType` and `modules`. For a ship the user doesn't own yet, or one neat can't find, give `shipType` and `modules` instead. Never ask the user for their hold.
- Consider more than one ship. Holds add up, so two jump freighters, or a JF and a carrier, can make one trip instead of several. Weigh the extra pilots, fuel and fatigue against the trips saved, and say which you chose and why.
- Check each pilot's jump fatigue in the overview.

## 6. Route and preferences

- Check `list_routes` for a saved route between the same places, and read its notes: it usually reflects choices the user made on purpose. Cost it with `plan_jump_route` and `route` set to its name. To check a route the pilot already flies, pass it as `waypoints` instead.
- Start from the constraints the user already stated (step 2) and turn each into a preference before calling `plan_jump_route` or `plan_haul`: "no gates" is `maxGates: 0` (leaving `maxGates` out means no limit), an area to stay out of is an avoid list, an alliance to stay away from is `avoidAlliances`. Never save a profile with empty preferences when the user has stated a constraint, and never leave it to a route's result to catch one.
- Ask in plain terms about anything else, and save what they say on a haul profile with `save_haul_profile`, giving `shipItemId` so neat works out the hold. Save one profile per ship: `plan_haul`'s `with` takes the other ships' haul profile names. Saved routes and profiles live in `~/.neat/routes.yaml` and `haul_profiles.yaml`, which the user can also edit by hand.
  - **Areas to stay out of:** systems, constellations or regions (`avoidSystems`, `avoidConstellations`, `avoidRegions`).
  - **Whose space:** alliances to avoid (`avoidAlliances`), avoiding sovereignty held by anyone at negative standing (`avoidBadStanding`), and blue alliances to prefer (`preferAlliances`).
  - **Midpoints:** systems where they can dock, have a cyno alt or a friendly POS (`midpoints`), and whether jumps must land only there (`midpointsRequired`).
  - **dotlan-style options:** prefer station systems (`preferStationSystems`), avoid incursions (`avoidIncursions`).
  - **What to optimise:** fuel (the default), number of jumps, or fatigue (`optimize`).
  - **Gates:** a cap on gate count (`maxGates`; 0 means no gates at all, and leaving it out means no limit), and whether to skip gate systems with recent kills (`maxRecentKills`).
- Call `plan_jump_route`, or `plan_haul` for cargo (with the other ships' haul profiles in `with` when several fly together). Read each route's `checks`:
  - a landing with `noDock` has nowhere known to dock: ask whether the user has a structure there, and save it if so;
  - `fuel` says whether the fuel fits the fuel bay or needs cargo space;
  - `cynoTypes` (or `sharedCynoTypes` for several ships) is what the lighters must light.
- Show the routes side by side with their dotlan links and the `dotlanSuggestion`. Sovereignty and standings preferences can't go in a dotlan link; the tool says so in the link's label.
- When they pick a route, save it with `save_route` and their reasons as `notes`, and pin it on the haul profile with `save_haul_profile`'s `outboundRoute` (and `returnRoute`). A pinned route is used only for trips with the same start and end systems; for any other trip the planner searches a route and says so in its notes, so save a separate haul profile for each regular run.

## 7. Cynos

- **Lighters:** characters whose `list_character_capabilities` include the cyno type needed, on a different account from the pilots jumping to them.
- **Fuel:** `estimate_cyno` with the lighter, their ship (by `shipItemId` when they own it), the cyno type the route needs and the number of lights (one per jump leg) gives Liquid Ozone per light, lights per load, the fuel already aboard and the lights it covers, how many loads of fuel the lights take, whether the generator fits, and any skills they're missing. Its `loads` list each top-up in order: before which light, how many units to add to what's aboard then, and the lights it covers. Put those numbers in the steps as they are ("trade 158 Liquid Ozone to the lighter before the second light"); never subtract fuel aboard from fuel needed yourself, and don't read the hold contents yourself.
- **Ship:** compare sources before choosing: a cyno ship already near the cyno point or the lighter's location or jump clones (`list_haul_ships` lists every ship with a cyno generator fitted), one fitted from assets, or one bought (`compare_trade_hubs` for the nearest hub that stocks the fit, `quote_market_purchase` for the list).
- **Implants:** a lit cyno is an easy target, and the lighter can lose their pod. `list_jump_clones` gives the implants in the clone each character is in now (`activeImplants`) and in each jump clone. An empty `activeImplants` means none; a missing one means neat couldn't read them, so say so rather than calling the clone clean. Price the ones the lighter will wear with `quote_market_purchase` and give the user the pod's value, since that's what they weigh the risk against; a list of implant names doesn't show it. Prefer a clone with none or cheap ones. Never route a character in a valuable clone through lowsec or nullsec gates in a pod: suggest a shuttle or another cheap, fast ship they own or can buy there, and say why.
- **One lighter or several:** with one lighter, the hauler waits while the lighter gates to each next cyno point. The route's `lighterGatesFromPreviousLanding` says how far that is. Weigh it against getting another lighter in place, and say which you chose and why. Don't bring in a character only so two parts of the job run at once.

## 8. Get characters in place

- Check `list_jump_clones` before planning gate travel: where each clone is, its implants, whether the character can clone jump now and when they next can, including the trip back.
- A clone jump has to start where there's a clone bay, and that applies to where the character is docked now, not only where they're going. `readyNow` is true only when the cooldown has passed (`cooldownReady`) and `jumpFromHere` is `yes`. When `jumpFromHere` is `no`, the character has to move to a station or structure with a clone bay first, so put that move in the steps or pick another character. When it's `unknown`, ask the user to check in game and save the answer with `save_structure`. Don't assume either way.
- A clone jump arrives in a capsule. Any step there beyond flying out in a pod needs a ship already at that station, owned by the character or contracted to them.

## 9. Order of operations

- Contracts can be made and accepted from anywhere. Only put a contract in the order when something in it is used by a later step.
- A pilot can load only what they own or can take from a corporation hangar they have access to. Anything in another character's personal hangar needs an item exchange contract from its owner to the pilot before the loading step, so each of those contracts is a step: who makes it, to whom, at which station, and what's in it (`list_location_assets`' `sources` give each owner's share). The same goes for fuel or a cyno ship one character hands to another.
- Write numbered steps saying who does what, in which ship, at which station: cyno cycles, fuel loads, docking and unloading included.
- Only decided work goes in the steps. Leave out choices still open (step 4) and anything the user said they'll handle themselves.
- Totals (fuel, fatigue, gates, volumes, cost) come from the tools' outputs.

## 10. Present the plan

- Lead with the outcome: how many trips, who hauls, who lights.
- Write systems with their displayed security, colored with the colors the tools give in formatted output.
- Say which choices you made and why: one lighter or several, one ship or several, where the cyno ship comes from.
- End with the decisions that are the user's and anything neat couldn't confirm, such as an unknown clone bay. A decision listed here must not already be made for them in the steps above.
- Before finishing, check every station and structure the plan uses (origin, destination, staging, where each character starts and every landing) against `list_structures`, and offer to save each one that isn't saved with `save_structure`, in one list.
- When you learn something durable (a structure's clone bay, why a route goes a certain way, which account a character is on), record it with `save_structure`, `save_route` or `update_character_details`, and tell the user what you saved.

## Keep it current

Offer to re-run route planning whenever politics change, a midpoint or structure is lost, or a pilot trains a jump or cyno skill.
