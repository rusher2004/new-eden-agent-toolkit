# Privacy

This page lists everything neat connects to, everything it stores on your computer, and where your EVE data goes. neat has no server of its own and no accounts. The only thing the neat developer ever receives is anonymous usage statistics: how many people use neat each month, which versions and platforms they run, and how often each of neat's tools is used. They contain nothing about you, your characters, what you ask or your computer, and you can turn them off with one command (see [Anonymous usage statistics](#anonymous-usage-statistics)).

## The short version

- neat runs on your computer and only talks to Fenris Creations (FC), the makers of EVE, to two public game-data sites (fuzzwork.co.uk and EVE Ref), to the AI app you connect it to, and, unless you turn it off, to Aptabase for anonymous usage statistics.
- Your login tokens are stored in your operating system's keychain. Everything else neat saves is in the `~/.neat` folder.
- Whatever neat looks up for you goes to your AI app, the same as anything else you share in a conversation. That app's own privacy policy covers what happens to it next.

## Who neat talks to

### FC (EVE Online)

| Address | What for | What neat sends |
|---|---|---|
| `login.eveonline.com` | Logging characters in, and renewing their access while neat runs | During a login your browser goes to EVE's login page, where you sign in with FC directly; neat never sees your EVE password. After that, neat sends FC the one-time login code and then, every so often, the character's refresh token to get a fresh short-lived access token. neat also downloads FC's public signing keys from here to check that tokens are genuine. |
| `esi.evetech.net` | Everything you ask about your characters, corporations, the market and the universe | The details of each request (character, corporation, item, system or other IDs and names you asked about), plus the character's access token when the data is private. For tools that change things in EVE, neat also sends what you asked it to send: the text of a mail, the contacts to add, a fitting, a fleet change, a calendar reply or an autopilot waypoint. |
| `images.evetech.net` | Character portraits on neat's login page | neat doesn't contact this itself. Your browser loads each logged-in character's portrait from FC's image server while neat's login page is open. |

ESI returns your character's data only for the access you granted (see "Choosing what neat can see" in the [README](README.md)). You can review or revoke that access at any time on EVE's [third-party applications page](https://community.eveonline.com/support/third-party-applications/).

### Public game data

neat keeps a local copy of EVE's static game data (items, ships, blueprints, systems, stations and so on) so it doesn't have to ask FC for it again and again. These downloads contain no personal data, and neat sends nothing about you to get them.

| Address | What for | When |
|---|---|---|
| `www.fuzzwork.co.uk` | A SQLite copy of FC's Static Data Export, about 176 MB to download and about 500 MB on disk | The first time neat needs game data in each session, it checks whether a newer copy exists, and downloads it if there is one |
| `data.everef.net` | Three details missing from the fuzzwork copy: item sizes (radius), which alliance a race belongs to, and graphic details | The first time neat needs game data in each session, it checks a small index file to see whether either archive has changed, and downloads only one that has. What it uses from them is saved in `~/.neat/sde/everef.json` |

### Anonymous usage statistics

FC's partner program asks third-party apps to show how many monthly users they have. To be able to show that, and to see which parts of neat people actually use, neat sends small anonymous messages to [Aptabase](https://aptabase.com), a privacy-focused analytics service. They go to `eu.aptabase.com`, whose data is stored in the EU (Germany). There are two kinds, and nothing else is ever sent.

**1. "neat started"**, sent once each time your AI app starts neat, to count monthly users:

| What it holds | Why |
|---|---|
| neat's version, for example `0.9.0` | To see which versions are in use, and when an old one can stop being supported |
| The operating system and processor type neat was built for, for example macOS and `arm64` | To know which platforms to test and fix. This is fixed when neat is built, so it names the download you chose, not anything about your computer: no OS version, computer model or name |
| A random ID that changes every month | To count each install once per month, however often you start neat. neat makes it from random numbers, not from anything about you or your computer, saves it in `~/.neat/analytics.json`, and replaces it with a new one at the start of each month (UTC). IDs from different months can't be linked to each other |
| A random ID for that one launch, and the time it happened | Aptabase needs these for every message. A new one is made each time neat starts |

**2. "tool used"**, one for each time your AI app uses one of neat's tools, to see which features people use and which ones fail:

| What it holds | Why |
|---|---|
| The tool's name, for example `plan_build` or `list_character_wallet_journal` | To count how often each feature is used. Only the names of neat's own tools are ever sent; a name neat doesn't recognise isn't recorded |
| Whether it worked (`ok` or `error`) | To find tools that fail often. The error itself is not sent |
| neat's version and the operating system and processor type it was built for, as above | To tell whether a problem is tied to one version or platform |
| A second random ID for that launch, and the time of the call | Aptabase needs these for every message. This ID is made separately from the one in "neat started", and these messages carry no monthly ID, so tool use can't be tied to an install, a month or a person: they only add up to totals per tool |

neat collects "tool used" messages in memory and sends them in batches about once a minute, and once more when neat stops. If a batch can't be sent, it's thrown away rather than saved. Nothing about tool use is ever written to your disk.

**What neat never sends:** what you asked for or the details of any tool call (characters, character IDs or names, items, systems, amounts, mail or anything else you or Claude passed to a tool), anything ESI returns, error messages, how long calls took, your EVE account, your login tokens, file paths, your computer's name, your username, your language or hardware IDs.

**What Aptabase sees:** like any internet service, Aptabase sees your IP address when a message arrives. Aptabase says it uses it only to work out your country and region and a temporary hash that changes every 24 hours, and never stores the address itself. See [Aptabase's privacy information](https://aptabase.com/docs). The data is kept for up to 5 years.

**What the neat developer can see:** only totals and breakdowns in Aptabase's dashboard: the number of launches and of distinct monthly IDs; how often each tool was used and how often it failed; and counts by version, platform and country. Nothing in it can be traced back to you, your characters or your computer, nobody can see which tools a given person used, and the neat developer has no way to look anyone up. The dashboard may be shown to FC to apply for the partner program.

**Turning it off:** run `neat analytics off`. neat then sends nothing at all to Aptabase, neither kind of message, and deletes `~/.neat/analytics.json`. `neat analytics status` shows whether it's on, and `neat analytics on` turns it back on. The setting is saved as `"analyticsDisabled": true` in `~/.neat/config.json`, so it stays off through updates. A neat that's already running picks up the change the next time your AI app starts it. Builds you make yourself from source send nothing unless you give them an Aptabase key of your own.

### Your AI app

neat is an MCP server: your AI app (Claude Code, the Claude desktop app, or another MCP app) starts it and asks it for things. Everything neat returns, such as your wallet balance, your assets, your mail or a build plan, goes to that app and is handled under the app's own privacy policy and settings. If you use Claude, see [Anthropic's privacy policy](https://www.anthropic.com/legal/privacy). neat itself doesn't send your data anywhere else.

### Things neat doesn't contact

- **dotlan:** route plans include evemaps.dotlan.net links so you can see a route on a map. neat only writes the links; nothing goes to dotlan unless you open one.
- **The neat developer or GitHub:** neat never contacts them. Apart from the anonymous usage statistics above, which go to Aptabase, nothing reaches the neat developer. To update neat, you download new versions yourself, and the Claude Code plugin updates through Claude Code.

### What every request carries

Every request to FC, fuzzwork, EVE Ref and Aptabase carries a User-Agent header like `neat/0.8.2 (project contact address; +https://github.com/rusher2004/new-eden-agent-toolkit)`. It names neat's version and gives the neat project's contact address, which FC asks every third-party app to include. It contains nothing about you or your computer. As with any internet connection, these services can see your IP address.

## What neat stores on your computer

| Where | What | Notes |
|---|---|---|
| Your OS keychain (macOS Keychain, Windows Credential Manager, or the Linux Secret Service), under the service name `neat` | One entry per logged-in character, `character_refresh_token:<character ID>`, holding that character's EVE refresh token | This is the only secret neat keeps. Removing a character in neat deletes its entry. `neat setup` briefly writes and deletes a test entry (`internal:setup_canary`) to check that the keychain works. |
| `~/.neat/config.json` | Each logged-in character's ID and name, plus any facility, haul and trade hub profiles you saved: structure names and their rigs, services and taxes, your hauling ships and pilots, and routing preferences such as areas to avoid. Also whether you turned the anonymous usage statistics off | Contains no tokens. Only your user account can read it. |
| `~/.neat/analytics.json` | The random monthly ID for the anonymous usage statistics, and which month it belongs to | Replaced with a new random ID each month. Deleted by `neat analytics off`, and never created while it's off. |
| `~/.neat/plans/` | One file per build plan you asked for, named by date and product: costs, build-or-buy decisions, shopping lists, haul loads and job schedules, including the stock you already own that the plan uses | Only your user account can read these. Delete them whenever you like. |
| `~/.neat/sde/eve.db`, `~/.neat/sde/meta.json` and `~/.neat/sde/everef.json` | The downloaded copy of EVE's static game data, a note of which version it is, and the details neat uses from EVE Ref's archives along with their versions | Public game data only. Temporary `eve.db.*.tmp` and `everef.json.*.tmp` files sit next to them while a new copy is saved. |
| Memory only (gone when neat stops) | Each character's short-lived access token, any game data neat fetched from ESI that the local copy was missing, and "tool used" messages waiting to be sent to Aptabase | Nothing in memory is written to disk. |

neat doesn't write log files of its own. It prints warnings and errors, such as a failed download, to your AI app, which may save them in its own logs.

### Files from older versions

If you ran neat 0.7.x or earlier, these may still be on your computer. They're no longer used, and you can delete them:

- `~/.neat/sde/sde.db`: an older game-data cache
- `~/.neat/bin/neat` and `~/.neat/NeatURLHandler.app` (macOS), the `HKCU\Software\Classes\eveauth-neat` registry key (Windows), or `~/.local/share/applications/neat-url-handler.desktop` (Linux): the old login link handler

## Logging in

While you're adding a character, neat runs a small web page at `http://127.0.0.1:48211` that only your own computer can reach. EVE's login page sends you back to it when you finish, and it lists the characters you've logged in. It stops as soon as you click **I'm done** or close the tab, or after 5 minutes without a login. It accepts requests only from its own page, so other websites you have open can't use it.

## Removing your data

- **Log out one character:** ask Claude to remove it. neat deletes its keychain entry and its line in `config.json`. To also cancel neat's access on FC's side, revoke it on EVE's [third-party applications page](https://community.eveonline.com/support/third-party-applications/).
- **Stop the usage statistics:** run `neat analytics off`. Messages already sent stay in Aptabase's totals, but nothing in them links to you, so there's nothing that could be found and deleted as yours.
- **Remove everything:** delete the `neat` program and the `~/.neat` folder, delete any `neat` entries left in your keychain, and remove neat from your AI app (`/plugin uninstall neat@neat` in Claude Code, or its entry in the desktop app's config).
- **Conversations:** what neat returned during a conversation stays in your AI app's history. Delete it there.

---

© 2014 CCP hf. All rights reserved. "EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.

neat is a fan project and isn't affiliated with or endorsed by Fenris Creations.
