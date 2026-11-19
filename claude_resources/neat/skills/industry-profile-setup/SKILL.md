---
name: industry-profile-setup
description: Set up or fix neat facility profiles for EVE Online industry and refining — NPC stations, Upwell structures (Raitaru, Azbel, Sotiyo, Athanor, Tatara, citadels) or starbases, with their rigs, services and taxes — and calibrate them against real jobs. Use when a build, reaction, research or refining plan needs a facility that isn't saved yet, when the user mentions a new structure or station to work from, or when estimates don't match what the game shows.
---

# Facility profile setup

ESI can't see a structure's rigs, fitted services or owner-set taxes. neat stores them in a facility profile; every bonus is then read from the SDE. Your job is to collect the facts, save the profile, and check it against the game. Don't work out bonuses yourself.

## 1. Which facility

- Ask where they build or refine. If unsure, look at where their blueprints and jobs are: `find_owned_blueprints` shows each blueprint's station or structure, and `list_corporation_industry_jobs` or `list_character_industry_jobs` show `facilityId`. For Upwell structures, `get_universe_structure` gives the name, type and system when the character has docking access.
- Decide the kind: `npc_station`, `upwell` or `starbase`.

## 2. What to ask the user for

Ask only for what applies, and say where to find it in game:

- **Upwell structures:**
  - the structure type, e.g. Sotiyo or Tatara;
  - every fitted rig, by exact name, from the structure's fitting window;
  - the fitted services (manufacturing, capital or supercapital shipyard, research, invention, reprocessing, and which reactors);
  - the owner's industry and reprocessing taxes, from the structure's info or the industry window;
  - for a Sotiyo building supercapitals, whether the system has the Supercapital Construction Facilities upgrade.
- **Refineries:** the reprocessing rig tier (none, T1 or T2).
- **NPC stations:** the station ID, and for reprocessing, the higher of the character's and their corporation's standing with the station's owner.

## 3. Save and check

- Call `save_facility_profile`. The response shows the security band and every resolved multiplier with its source. Read the warnings out to the user, for example a rig that can't be fitted in highsec, or a tax above the 10% cap.
- Ask the user to compare one number with the game:
  - for manufacturing, the industry window's material count for one blueprint (pass that product to `describe_facility`);
  - for refining, the reprocessing window's yield breakdown and fee. `estimate_reprocessing_yield` shows the same breakdown.
- If they don't match, the profile is missing a rig or has the wrong one. Fix the input and save again.

## 4. Calibrate

- If the user has a completed or running job at this facility, offer `calibrate_facility_profile` with its job ID. It saves the fee rate the game actually charged, which later plans use in place of estimates.
