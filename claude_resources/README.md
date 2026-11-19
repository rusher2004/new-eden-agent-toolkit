# Claude resources — `neat`

Claude Code resources for `neat`: a single plugin that registers the `neat` MCP server and ships three
skills — `industry-build-plan`, `industry-profile-setup` and `haul-route-setup` — that drive neat's
industry planning tools. The skills hold discovery and decision steps only; every number comes from a tool.

Contents:

```
new-eden-agent-toolkit/
├── .claude-plugin/
│   └── marketplace.json   # marketplace "neat" -- must live at repo root for the `github`
│                           # source type below to find it
└── claude_resources/
    └── neat/               # the plugin
        ├── .claude-plugin/plugin.json
        ├── .mcp.json       # registers the neat MCP server
        ├── skills/         # industry-build-plan, industry-profile-setup, haul-route-setup
        └── CHANGELOG.md
```

## Prerequisite: install the `neat` CLI

The plugin's `.mcp.json` expects `neat` to be resolvable on `PATH`. Download a release from
[GitHub Releases](https://github.com/rusher2004/new-eden-agent-toolkit/releases), put the `neat`
binary on your `PATH`, then run `neat setup` once to store ESI credentials in your OS keychain.

## Install the plugin (user scope)

**Two ordered steps** — step 1 registers the marketplace, step 2 installs the plugin from it:

```text
1) /plugin marketplace add rusher2004/new-eden-agent-toolkit
2) /plugin install neat@neat
```

This repo is **private**. Manual install/update reuses your existing git credentials (`gh auth
login`, SSH agent — whatever `git`/`gh` already use on this machine). Claude Code also
auto-refreshes git-backed marketplaces in the background; for that to work against a private repo
without an interactive prompt, set a `GITHUB_TOKEN` or `GH_TOKEN` environment variable. Otherwise,
pull the latest plugin content on demand with `/plugin marketplace update neat`.

## How to test

1. Complete the prerequisite above (`neat` on `PATH`, `neat setup` run once).
2. Install the plugin per the steps above.
3. In a Claude Code session, confirm the `neat` MCP server is connected (`claude mcp list` or ask
   Claude to call any `neat` tool, e.g. `get_status`).
