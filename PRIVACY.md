# Privacy

This page lists everything neat connects to, everything it stores on your computer, and where your EVE data goes. neat has no server of its own, no accounts, no analytics and no telemetry. The neat developer never receives any of your data.

## The short version

- neat runs on your computer and only talks to CCP, to two public game-data sites (fuzzwork.co.uk and EVE Ref), and to the AI app you connect it to.
- Your login tokens are stored in your operating system's keychain. Everything else neat saves is in the `~/.neat` folder.
- Whatever neat looks up for you goes to your AI app, the same as anything else you share in a conversation. That app's own privacy policy covers what happens to it next.

## Who neat talks to

### CCP (EVE Online)

| Address | What for | What neat sends |
|---|---|---|
| `login.eveonline.com` | Logging characters in, and renewing their access while neat runs | During a login your browser goes to EVE's login page, where you sign in with CCP directly; neat never sees your EVE password. After that, neat sends CCP the one-time login code and then, every so often, the character's refresh token to get a fresh short-lived access token. neat also downloads CCP's public signing keys from here to check that tokens are genuine. |
| `esi.evetech.net` | Everything you ask about your characters, corporations, the market and the universe | The details of each request (character, corporation, item, system or other IDs and names you asked about), plus the character's access token when the data is private. For tools that change things in EVE, neat also sends what you asked it to send: the text of a mail, the contacts to add, a fitting, a fleet change, a calendar reply or an autopilot waypoint. |
| `images.evetech.net` | Character portraits on neat's login page | neat doesn't contact this itself. Your browser loads each logged-in character's portrait from CCP's image server while neat's login page is open. |

ESI returns your character's data only for the access you granted (see "Choosing what neat can see" in the [README](README.md)). You can review or revoke that access at any time on EVE's [third-party applications page](https://community.eveonline.com/support/third-party-applications/).

### Public game data

neat keeps a local copy of EVE's static game data (items, ships, blueprints, systems, stations and so on) so it doesn't have to ask CCP for it again and again. These downloads contain no personal data, and neat sends nothing about you to get them.

| Address | What for | When |
|---|---|---|
| `www.fuzzwork.co.uk` | A SQLite copy of CCP's Static Data Export, about 176 MB to download and about 500 MB on disk | The first time neat needs game data in each session, it checks whether a newer copy exists, and downloads it if there is one |
| `data.everef.net` | Three details missing from the fuzzwork copy: item sizes (radius), which alliance a race belongs to, and graphic details | The first time neat needs game data in each session, it checks a small index file to see whether either archive has changed, and downloads only one that has. What it uses from them is saved in `~/.neat/sde/everef.json` |

### Your AI app

neat is an MCP server: your AI app (Claude Code, the Claude desktop app, or another MCP app) starts it and asks it for things. Everything neat returns, such as your wallet balance, your assets, your mail or a build plan, goes to that app and is handled under the app's own privacy policy and settings. If you use Claude, see [Anthropic's privacy policy](https://www.anthropic.com/legal/privacy). neat itself doesn't send your data anywhere else.

### Things neat doesn't contact

- **dotlan:** route plans include evemaps.dotlan.net links so you can see a route on a map. neat only writes the links; nothing goes to dotlan unless you open one.
- **The neat developer, GitHub, or any analytics service:** neat never contacts them. To update neat, you download new versions yourself, and the Claude Code plugin updates through Claude Code.

### What every request carries

Every request to CCP, fuzzwork and EVE Ref carries a User-Agent header like `neat/0.8.2 (project contact address; +https://github.com/rusher2004/new-eden-agent-toolkit)`. It names neat's version and gives the neat project's contact address, which CCP asks every third-party app to include. It contains nothing about you or your computer. As with any internet connection, these services can see your IP address.

## What neat stores on your computer

| Where | What | Notes |
|---|---|---|
| Your OS keychain (macOS Keychain, Windows Credential Manager, or the Linux Secret Service), under the service name `neat` | One entry per logged-in character, `character_refresh_token:<character ID>`, holding that character's EVE refresh token | This is the only secret neat keeps. Removing a character in neat deletes its entry. `neat setup` briefly writes and deletes a test entry (`internal:setup_canary`) to check that the keychain works. |
| `~/.neat/config.json` | Each logged-in character's ID and name, plus any facility, haul and trade hub profiles you saved: structure names and their rigs, services and taxes, your hauling ships and pilots, and routing preferences such as areas to avoid | Contains no tokens. Only your user account can read it. |
| `~/.neat/plans/` | One file per build plan you asked for, named by date and product: costs, build-or-buy decisions, shopping lists, haul loads and job schedules, including the stock you already own that the plan uses | Only your user account can read these. Delete them whenever you like. |
| `~/.neat/sde/eve.db`, `~/.neat/sde/meta.json` and `~/.neat/sde/everef.json` | The downloaded copy of EVE's static game data, a note of which version it is, and the details neat uses from EVE Ref's archives along with their versions | Public game data only. Temporary `eve.db.*.tmp` and `everef.json.*.tmp` files sit next to them while a new copy is saved. |
| Memory only (gone when neat stops) | Each character's short-lived access token, and any game data neat fetched from ESI that the local copy was missing | Nothing in memory is written to disk. |

neat doesn't write log files of its own. It prints warnings and errors, such as a failed download, to your AI app, which may save them in its own logs.

### Files from older versions

If you ran neat 0.7.x or earlier, these may still be on your computer. They're no longer used, and you can delete them:

- `~/.neat/sde/sde.db`: an older game-data cache
- `~/.neat/bin/neat` and `~/.neat/NeatURLHandler.app` (macOS), the `HKCU\Software\Classes\eveauth-neat` registry key (Windows), or `~/.local/share/applications/neat-url-handler.desktop` (Linux): the old login link handler

## Logging in

While you're adding a character, neat runs a small web page at `http://127.0.0.1:48211` that only your own computer can reach. EVE's login page sends you back to it when you finish, and it lists the characters you've logged in. It stops as soon as you click **I'm done** or close the tab, or after 5 minutes without a login. It accepts requests only from its own page, so other websites you have open can't use it.

## Removing your data

- **Log out one character:** ask Claude to remove it. neat deletes its keychain entry and its line in `config.json`. To also cancel neat's access on CCP's side, revoke it on EVE's [third-party applications page](https://community.eveonline.com/support/third-party-applications/).
- **Remove everything:** delete the `neat` program and the `~/.neat` folder, delete any `neat` entries left in your keychain, and remove neat from your AI app (`/plugin uninstall neat@neat` in Claude Code, or its entry in the desktop app's config).
- **Conversations:** what neat returned during a conversation stays in your AI app's history. Delete it there.
