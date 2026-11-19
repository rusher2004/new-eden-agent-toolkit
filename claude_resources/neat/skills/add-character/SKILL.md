---
name: add-character
description: Log EVE Online characters in to neat, or grant a logged-in character more ESI access, choosing scope groups from what the user wants to do. Use when the user wants to add, log in or re-authorize a character, when no character is logged in yet, or when a neat tool fails because a character hasn't granted a scope.
---

# Adding characters

neat asks EVE for access in scope groups: `basic` (skills, location, ship, clones) is always granted, and the user picks the rest. Requesting only what they need keeps EVE's login working across several logins in a row. A login asking for every scope at once can fail with "HTTP Error 400. The size of the request headers is too long".

## 1. Ask what they'll use neat for

- Call `list_characters` first. If the character they mean is already logged in and only needs more access, go to section 3 instead.
- Call `list_scope_groups` to see the groups and what each one covers.
- Ask the user what they want neat to help with, in plain terms ("industry, the market, reading mail, fleets..."). Don't read them the group list unless they ask.
- Map their answer to groups. For example, industry planning wants `industry`, `assets` and `market`. Corporation groups only work for characters with the matching in-game corporation roles; mention that when you pick one.
- If they don't know or don't care, use no extra groups. `basic` is enough to start, and more can be granted later.

## 2. Log in

- Call `add_character` with the chosen `scopeGroups`. Tell the user their browser will open at EVE's login page.
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
