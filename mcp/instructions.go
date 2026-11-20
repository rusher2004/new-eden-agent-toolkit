package mcp

// Instructions is sent to every MCP client in the initialize result. Clients
// such as Claude Desktop don't load the Claude Code plugin's skills, so this
// is the one place every client learns how to onboard a new user and how to
// treat neat's tools. Keep it short and in step with the add-character and
// industry skills in claude_resources/neat/skills/.
const Instructions = `neat connects you to the user's EVE Online account: their characters' data through Fenris Creations' ESI API, static game data from a local copy of the Static Data Export, and industry, hauling and route planning.

Getting started: call list_characters. If no character is logged in, help the user add one: call list_scope_groups, ask in plain terms what they want neat to help with (their wallet and trading, what they own, industry, mail, fleets...), and always also ask whether they want their corporation's data (members, corp wallets and assets, corp industry jobs, structures). Most players are in a player corporation and want it, and it's easiest to grant at the first login. The corporation groups return data only for characters holding the matching in-game roles, but granting them to other characters does no harm. Add structures whenever they'll look at assets or industry, since items in player structures otherwise show only a structure ID. Call add_character with the matching scopeGroups, or with none if they aren't sure, since basic is always granted and more can be added later. Before calling it, tell them their browser will open at EVE's login page, where they can log in one or more characters, and that if no tab opens they can open http://127.0.0.1:48211/ in their browser while it waits.

Most character tools take a characterId: get it from list_characters, and ask which character the user means when several are logged in and it's unclear. Corporation tools also need the character to hold the matching in-game corporation roles. A corporation tool that fails with "does not have required role(s)" needs a different character, not a new scope: try another logged-in character in the same corporation (get_character_affiliations), or ask the user which one holds the role. Don't retry a call ESI refused across every character, since each refusal counts against ESI's error limit.

When a tool fails because a character hasn't granted a scope, the error names the scope group. Ask the user before granting it, then call add_character with that characterId and group; the character keeps the access it already had. To add a group to several characters in one login, call add_character without characterId and with every group they should end up with, since that login replaces each character's access with exactly what it requested.

Industry, hauling and routing numbers (quantities, prices, fees, yields, fuel, volumes, times, routes) come from neat's tools. Never calculate or adjust them yourself; if one looks wrong, find and fix the input (a facility or haul profile, a character, a decision).

Tools that change things in EVE or the game client say so in their descriptions (sending mail, editing contacts, fittings and fleets, responding to calendar events, setting waypoints, opening in-game windows). Confirm with the user before calling them.`
