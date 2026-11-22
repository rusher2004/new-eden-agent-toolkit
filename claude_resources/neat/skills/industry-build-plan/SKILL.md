---
name: industry-build-plan
description: Plan building something in EVE Online with neat — what it costs, build or buy for each component, which characters build it, compressed ore and refining, jump freighter hauling, and a shareable plan page with Buy All lists. Use when the user asks what it would cost to build an item or their blueprints, wants a build, manufacturing or reaction plan, asks whether to build components or buy them, or wants a shopping and haul plan for a build.
---

# Industry build plan

You run the discovery and the decisions. neat's tools do every calculation: quantities, prices, fees, yields, fuel, volumes and times. Never compute these yourself, and never adjust a number a tool returned. If a number looks wrong, find which input caused it (a profile, a decision, a character) and fix the input.

## 1. What to build

- Ask what they want to build, or which blueprints. If they name owned blueprints, call `find_owned_blueprints` with the product (and `corporationId` for corp hangars). It shows copies, runs, ME and TE, location and container, and which copies are already in active jobs.
- Confirm the count. By default `plan_build` runs every free copy of the product's blueprint. If some copies are already in production, say so and confirm they're excluded.
- If they own no copies, ask how many units to plan (`quantity`).

## 2. Who

- Ask which character owns the blueprints (or which corporation), who will install jobs (`builderIds`), and, if minerals will come from ore, who refines (`refinerId`).
- Call `check_build_skills` for the candidate builders. Report who can build each part and what's missing. Don't guess skills.
- Planning reads blueprints, jobs, assets and market orders, which need the `industry`, `assets`, `market` and `structures` scope groups (`structures` names the player structures blueprints and stock sit in; `corporation_industry` and `corporation_finance` for corp blueprints and hangars). If a tool reports a missing scope, use the **add-character** skill to grant all the groups the plan needs in one login.

## 3. Where, and the profiles

- `plan_build` needs a facility profile for manufacturing, and optional ones for reactions and refining. It needs a haul profile if anything moves by jump freighter. Call `list_facility_profiles`, `list_haul_profiles` and `list_hub_profiles`.
- If a profile is missing, use the **industry-profile-setup** skill for facilities and **haul-plan** for hauling, then come back.
- Run `describe_facility` with the product to show the user the multipliers and any rule that blocks the build there (for example, capitals in highsec). If it can't run there, stop and ask.

## 4. The real decisions

Ask only what changes the plan, and recommend a default:
- **Compressed ore or refined minerals?** Default: compressed ore, if they have a refining profile and a refiner. It's usually cheaper and much smaller to haul, at the cost of refining and moving the minerals to the build site.
- **Build or buy overrides.** Default: build anything they own blueprints for and buy the rest. Ask only if they want to force an item either way (`decisions`).
- **Use stock on hand?** Default: yes, from the build structure. Ask for the hangar division if they keep materials in one (`stockDivision`, e.g. `CorpSAG3`).
- **Reprocessing implant plugged in?** Ask if refining; pass `useImplant: false` if not.

## 5. Plan

- Call `plan_build` with everything above.
- Lead with the answer: total cost, cost per unit, and the build-or-buy verdict from `buildVsBuy`. Then the biggest cost lines. Pass on every warning and assumption the tool returns.
- If the user wants to compare options (ore versus minerals, another facility, another hull), call `plan_build` again with the one input changed. Don't estimate the difference yourself.

## 6. The plan page

Every build plan gets an artifact. It must include:
- the volume in m³ wherever materials move: each purchase, each haul load (split by drop-off), the refined-mineral shuttle, and stock moved between hangars;
- a button that copies a Multibuy list to the clipboard at every purchase point: the full order (`multibuy`) and each haul load (`haul.loads[].multibuy`, which already includes that trip's fuel). Use the tool's text exactly; it is `Name<TAB>Quantity` per line;
- the dotlan links from `haul.outboundRoute.dotlanLinks` and `haul.returnRoute.dotlanLinks`, labelled as the tool labels them.

Build the page from `plan_build`'s output, or from the JSON file at `planPath`. Don't re-derive any figure.
