# neat Claude Code plugin

This folder holds the Claude Code plugin for neat. It registers the `neat` MCP server and adds skills that guide Claude through the multi-step parts of using it. Installing neat and connecting it to Claude is covered in the [main README](../README.md#install); this page is for people working on the plugin.

## What's in it

```
new-eden-agent-toolkit/
├── .claude-plugin/
│   └── marketplace.json   # the "neat" marketplace; must sit at the repo root so
│                          # `/plugin marketplace add rusher2004/new-eden-agent-toolkit` finds it
└── claude_resources/
    └── neat/               # the plugin
        ├── .claude-plugin/plugin.json   # version is generated from VERSION
        ├── .mcp.json                    # runs `neat mcp` from the user's PATH
        ├── skills/
        └── CHANGELOG.md                 # user-facing release notes
```

The skills:

- `add-character`: log characters in, picking scope groups from what the user wants to do (always offering the corporation groups), grant more access when a tool reports a missing scope, add a group to several characters in one login, and handle corporation tools refused for in-game roles.
- `industry-build-plan`: plan a build end to end, from blueprints and builders through build-or-buy, refining and hauling to a shareable plan page.
- `industry-profile-setup`: save and calibrate facility profiles (NPC stations, Upwell structures, starbases) with their rigs, services and taxes.
- `haul-plan`: plan any move of cargo, ships or characters: haulers and cyno lighters, assembled ships, routes and routing preferences, jump clones, and the order of operations.

Skills hold discovery and decision steps only. Every number comes from a neat tool, so skills never contain formulas or arithmetic. When a tool a skill drives changes its inputs, outputs or behavior, update the skill in the same change.

Clients that don't load plugin skills, such as the Claude desktop app, still get the essentials (how to onboard a new user, scope errors, confirming before write tools, never computing industry numbers, the order for planning a move) from the MCP server's instructions in `mcp/instructions.go`. Keep the two consistent.

## Install

```text
/plugin marketplace add rusher2004/new-eden-agent-toolkit
/plugin install neat@neat
```

Claude Code doesn't auto-update third-party marketplaces unless the user turns it on under `/plugin`; `/plugin marketplace update neat` pulls the latest. The plugin needs the `neat` binary on PATH; see the main README.

## Testing changes

1. Install your local build with `make install`, so the plugin's `neat mcp` runs it.
2. Point Claude Code at your working copy instead of GitHub: `/plugin marketplace add ./` from the repo root, then `/plugin install neat@neat`. After editing a skill, run `/plugin marketplace update neat` and restart Claude Code.
3. Check the server is connected with `/mcp`, then ask Claude something that drives the skill you changed, such as "add my EVE character" for `add-character`.
