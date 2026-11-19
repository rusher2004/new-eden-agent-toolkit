package mcp

// Instructions is sent to every MCP client in the initialize result. Clients
// such as Claude Desktop don't load the Claude Code plugin's skills, so this
// is the one place every client learns how to onboard a new user and how to
// treat neat's tools. Keep it short and in step with the add-character and
// industry skills in claude_resources/neat/skills/.
const Instructions = `neat connects you to the user's EVE Online account: their characters' data through Fenris Creations' ESI API, static game data from a local copy of the Static Data Export, and industry, hauling and route planning.

Getting started: call list_characters. If no character is logged in, help the user add one: call list_scope_groups, ask in plain terms what they want neat to help with (industry, trading, mail, fleets...), and call add_character with the matching scopeGroups, or with none if they aren't sure, since basic is always granted and more can be added later. Tell them their browser will open at EVE's login page, where they can log in one or more characters.

Most character tools take a characterId: get it from list_characters, and ask which character the user means when several are logged in and it's unclear. Corporation tools also need the character to hold the matching in-game corporation roles.

When a tool fails because a character hasn't granted a scope, the error names the scope group. Ask the user before granting it, then call add_character with that characterId and group; the character keeps the access it already had.

Industry, hauling and routing numbers (quantities, prices, fees, yields, fuel, volumes, times, routes) come from neat's tools. Never calculate or adjust them yourself; if one looks wrong, find and fix the input (a facility or haul profile, a character, a decision).

Tools that change things in EVE or the game client say so in their descriptions (sending mail, editing contacts, fittings and fleets, responding to calendar events, setting waypoints, opening in-game windows). Confirm with the user before calling them.`
