---
name: add-character
description: Log EVE Online characters in to neat, or grant a logged-in character more ESI access, choosing scope groups from what the user wants to do. Use when the user wants to add, log in or re-authorize a character, when no character is logged in yet, when a neat tool fails because a character hasn't granted a scope, or when a corporation tool is refused because a character lacks in-game roles.
---

# Adding characters

neat asks EVE for access in scope groups: `basic` (skills, location, ship, clones) is always granted, and the user picks the rest. Requesting only what they need keeps EVE's login working across several logins in a row. A login asking for every scope at once can fail with "HTTP Error 400. The size of the request headers is too long".

## 1. Ask what they'll use neat for

- Call `list_characters` first. If the character they mean is already logged in and only needs more access, go to section 3 instead.
- Call `list_scope_groups` to see the groups and what each one covers.
- Ask the user what they want neat to help with, in plain terms ("your wallet and trading, what you own and where, industry, mail, fleets..."). Don't read them the group list unless they ask. If the client offers a multiple-choice question, use one, with corporation data as its own option.
- **Always ask about their corporation too**, even if they didn't mention it: whether they want neat to see their corporation's members, wallets and assets, industry jobs and structures. Most players are in a player corporation and want this, and asking now saves a second round of logins later.
- Map their answers to groups:

  | They want | Groups |
  |---|---|
  | Wallet, trading, contracts | `wallet`, `market`, `contracts` |
  | What they own and where | `assets`, `structures` |
  | Industry, or industry planning | `industry`, `assets`, `market`, `structures` |
  | Mail, calendar, contacts | `mail`, `calendar`, `contacts` |
  | Fleets, fittings | `fleet`, `fittings` |
  | Corporation members, roles, titles | `corporation` |
  | Corporation wallets, assets, contracts, orders | `corporation_finance` |
  | Corporation industry, structures, mining | `corporation_industry` |

  Other tools show the player structures where items, jobs and ships are as bare IDs; `structures` lets you look up their names with `get_universe_structure`.
- Corporation groups only return data for characters holding the matching in-game corporation roles (Director, Accountant, Factory Manager, Station Manager...). Say so when you pick them, and add that granting them to characters without roles does no harm, so the same login can cover all of the user's characters.
- If they don't know or don't care, use no extra groups. `basic` is enough to start, and more can be granted later.

## 2. Log in

- Call `add_character` with the chosen `scopeGroups`. Before the call, tell the user their browser will open at EVE's login page, and that if no tab opens they can open http://127.0.0.1:48211/ in their browser while neat waits (for example when the browser runs on another display or remote desktop session).
- After they log in, the browser shows neat's page, where they can add more characters with the same access, then click "I'm done" or close the tab. The tool returns once they do.
- Report who was added and which scope groups each one granted.
- If the login fails because port 48211 is in use, tell the user another program on their computer holds it and to close it and try again. If EVE's page says the request headers are too long, have them clear their browser's cookies for `login.eveonline.com`.
- If this was their first character, suggest two or three things to ask that fit the access they just granted, such as their wallet balance, their skill queue, or where their ships are.

## 3. Granting more access later

When a tool fails because a character "hasn't granted neat the ESI scope ...", the error names the scope group and character:

- Tell the user what that access is for, and ask whether to grant it. Don't re-authorize without asking.
- If they agree, call `add_character` with that `characterId` and the named group in `scopeGroups`. The character keeps everything it already granted. Tell them to log in as that same character on EVE's page.
- If the result has `warnings`, act on them before retrying: if the user logged in as a different character, tell them and offer to try again; if the character's earlier scope groups couldn't be read, tell them which groups it has now and ask whether it needs any others back.
- Then retry what failed.

If you can tell the user is about to need several new groups (say they're starting industry planning), ask about all of them at once rather than one failure at a time.

### Adding a group to several characters

`characterId` re-authorizes one character per call. To give a new group to several logged-in characters in one browser session, call `add_character` **without** `characterId` and with every group they should end up with: their current groups (from the last `add_character` result, or the scope errors) plus the new one. A login replaces a character's access with exactly what it requested, so a group left out of the list is lost. Tell the user to log in each character in turn from neat's page. If you don't know every character's current groups (`list_characters` doesn't show them, so in a new conversation you usually won't), don't guess: re-authorize each character with its own `characterId` call instead, which keeps what it already had.

## Corporation tools refused for roles

A corporation tool that fails with "Character does not have required role(s)" (HTTP 403) isn't a scope problem, and re-authorizing won't fix it. The character lacks the in-game corporation role. Try one other logged-in character in the same corporation (`get_character_affiliations` shows who's in which), or ask the user which character holds the role. Don't loop the same call through every character: each refusal counts against ESI's error limit, and running it out gets neat temporarily blocked.
