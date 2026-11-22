# neat — New Eden Agent Toolkit

neat is a tool that enables you to work with Claude for all things Eve Online. Using static data exports and the ESI API, you'll have anything you need related to your characters, corporations, and the wider Eve universe.

Once it's set up, you can ask Claude things like:

- "How much ISK do my characters have, and what did I spend it on this week?"
- "Where are all my Hulks, and what's in the hangar in Jita?"
- "What's in my skill queue and when does it finish?"
- "Which of my market orders have been undercut?"
- "Help me setup a PI network with my characters."
- "What would it cost to build 10 Ravens with my blueprints at my Raitaru? Should I build the components or buy them?"
- "Plan a jump freighter route from Jita to my home system that stays out of Delve."
- "Find me a quiet low-sec system bordering a high-sec system near Amarr."
- "Summarize my unread mail and the corp calendar for this week."

neat runs entirely on your own computer, nothing is stored on a server. Your login tokens live in your operating system's keychain, never in a file. neat talks only to Fenris Creations (FC), the makers of EVE, and downloads public game data from fuzzwork.co.uk and EVE Ref. See [Privacy](https://github.com/rusher2004/new-eden-agent-toolkit/blob/main/PRIVACY.md) for everything neat connects to and stores.

It works in [Claude Code](https://claude.com/claude-code) and the [Claude desktop app](https://claude.ai/download), and in any other app that supports MCP servers.

> heads up, the first time neat needs to access info about in-game items or universe stuff, there will be a big download of the static data export, so it'll take a few minutes or more.

## Install

### 1. Download neat

Get the archive for your computer from the [latest release](https://github.com/rusher2004/new-eden-agent-toolkit/releases/latest):

- **Mac with Apple silicon (M1 or newer):** `neat_<version>_mac_arm64.tar.gz`
- **Intel Mac:** `neat_<version>_mac_amd64.tar.gz`
- **Windows:** `neat_<version>_windows_amd64.zip`
- **Windows on ARM:** `neat_<version>_windows_arm64.zip`
- **Linux:** `neat_<version>_linux_amd64.tar.gz` (or `arm64`)

**On a Mac or Linux**, put `neat` in `~/.local/bin` from a terminal:

```bash
mkdir -p ~/.local/bin
tar -xzf ~/Downloads/neat_<version>_<platform>.tar.gz -C ~/.local/bin neat  # the exact file you downloaded
xattr -d com.apple.quarantine ~/.local/bin/neat 2>/dev/null  # Mac only: lets macOS run a downloaded app that isn't from the App Store
```

If `neat version` then says "command not found", add `~/.local/bin` to your PATH: run `echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc` (or `~/.bashrc` on Linux), then open a new terminal.

**On Windows**, unzip the archive and move `neat.exe` into a folder of its own, such as `%LOCALAPPDATA%\Programs\neat`. Then add that folder to your PATH: search the Start menu for "Edit environment variables for your account", select **Path**, click **Edit**, then **New**, and paste the folder. Open a new terminal so it picks up the change.

### 2. Check it works

```bash
neat setup
```

This creates neat's settings folder (`~/.neat`) and checks that neat can use your keychain. On a Mac you may be asked to allow keychain access; on Linux you need a keychain service such as GNOME Keyring or KWallet running.

### 3. Connect it to Claude

**Claude Code:** run these two commands inside Claude Code, then restart it:

```text
/plugin marketplace add rusher2004/new-eden-agent-toolkit
/plugin install neat@neat
```

The plugin also teaches Claude how to log characters in and how to run industry and hauling plans step by step. To get a newer version of the plugin, run `/plugin marketplace update neat` (or turn on auto-update for the `neat` marketplace under `/plugin`), and install the matching `neat` program from the release page.

**Claude desktop app:** open **Settings → Developer → Edit Config**, and add neat to the `claude_desktop_config.json` file that opens, using the full path to the `neat` you installed:

```json
{
  "mcpServers": {
    "neat": {
      "command": "/Users/YOUR_NAME/.local/bin/neat",
      "args": ["mcp"]
    }
  }
}
```

On Windows the command looks like `"C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\neat\\neat.exe"`. Keep anything else already in the file. Quit and reopen the app, and neat appears in its list of connected tools.

**Other MCP apps:** run `neat mcp` as a stdio MCP server.

### 4. Log in a character

Ask Claude to "add my EVE character". It asks what you want neat to help with, and whether you want your corporation's data too, then opens EVE's login page in your browser. After you log in, the browser shows a neat page where you can log in more characters with the same access, then click **I'm done** or close the tab. If no browser tab opens, go to http://127.0.0.1:48211/ yourself while Claude waits.

The same page groups your characters by EVE account. Pick an account for each character, or start a new one, and rename or delete groups as you like. When you log in several characters one after another, neat suggests putting each in the same account as the one before it, and saves that only when you confirm. It matters because characters on one account can't be logged in at the same time, so plans that need two of them online together (a jump freighter pilot and their cyno alt, for example) need to know. EVE doesn't tell neat which characters share an account, so Claude may also ask you, and it saves anything you tell it about a character as notes.

## Choosing what neat can see

EVE asks you to approve the access an app requests. neat groups that access by what you'll use it for, and every character gets the `basic` group: skills, location, current ship, clones and implants. You add the rest as you need them, for example:

- `wallet`, `assets`, `market`, `contracts`: your ISK, your stuff and your trading
- `structures`: lets Claude look up the names of player structures (Upwell citadels and engineering complexes) your items, ships and jobs are in; without it they show only as a number
- `industry`: blueprints, industry jobs and mining; industry planning also wants `assets`, `market` and `structures`
- `mail`, `calendar`, `contacts`, `fleet`, `fittings`
- `corporation`, `corporation_finance`, `corporation_industry`: your corporation's members, wallets, assets, industry jobs and structures

If you're in a player corporation, grant the corporation groups at your first login: Claude asks about them every time. They only show data for characters holding the matching in-game roles (Director, Accountant, Factory Manager and so on), but it does no harm to grant them to your other characters, so one login can cover them all.

You don't have to get it right up front. If you ask for something a character hasn't allowed yet, Claude tells you which access it needs and asks before sending you back to EVE's login page to grant it. The character keeps everything it already allowed. Ask Claude to "list neat's scope groups" for the full list.

To log a character out of neat, ask Claude to remove it. That deletes its saved login from your keychain. You can also revoke neat's access entirely from EVE's [third-party applications page](https://community.eveonline.com/support/third-party-applications/).

## What neat can do

neat covers every ESI endpoint, plus planning tools of its own:

- **Your characters:** wallets, assets, skills and skill queues, clones and implants, location and ship, standings, loyalty points, mail, calendar, contacts, notifications, fittings, killmails and planetary colonies
- **Your corporation and alliance:** members, roles, wallets, assets, contracts, industry jobs, structures, starbases, mining and more, for characters with the right roles
- **The market:** orders, price history, prices in player structures, and public contracts
- **The universe:** every item, ship, system and station, routes, sovereignty, faction warfare, incursions, wars and server status
- **Fleets and the game client:** manage your fleet, set autopilot waypoints, and open in-game windows
- **Industry planning:** full build plans with build-or-buy decisions, job costs, research and invention estimates, refining and the cheapest compressed-ore mix, jump freighter routes with fuel and fatigue, and in-game Multibuy lists

Some tools change things in EVE: sending mail, editing contacts, fittings and fleets, answering calendar invites, setting waypoints. Claude is told to check with you before using them.

The [tool reference](docs/reference/tools.md) lists every tool.

### Industry planning

For build plans, neat needs to know things ESI can't tell it: which rigs your structure has and what taxes it charges, which ship you haul with, which areas you avoid. Claude asks you for these the first time and saves them as named profiles, so you only describe each structure or freighter once. Every number in a plan comes from neat's calculations against EVE's own data, not from Claude's guesswork, and you can check a facility against a real job in game to calibrate its fees.

## Anonymous usage statistics

neat sends Aptabase anonymous usage statistics so the project can count its monthly users, which FC's partner program asks for, and see which features people use. Each time neat starts, it sends neat's version, which download you're running (for example macOS on `arm64`) and a random ID that's replaced every month. Each time Claude uses one of neat's tools, it sends the tool's name and whether it worked, with no ID that ties it to you or your install. It never includes your characters, your EVE account, what you ask, what a tool was given or returned, or anything about your computer. To turn it off, run `neat analytics off`. [PRIVACY.md](PRIVACY.md) lists exactly what's sent.

## Where neat keeps things

- `~/.neat/characters.yaml`, `facilities.yaml`, `haul_profiles.yaml` and `routes.yaml`: your logged-in characters (with the account labels and notes you gave them), saved facilities, haul profiles and jump routes. neat keeps these up to date, and you can open and edit them too: change a value, add notes, or add comments with `#`. neat changes only the fields it has a reason to and leaves the rest of the file, including your comments, alone. If you break a file's YAML, neat says which line and won't touch it until it's fixed.
- `~/.neat/config.json`: neat's own settings, such as whether you turned the usage statistics off, and your trade hub profiles
- `~/.neat/sde/`: a local copy of EVE's static game data (about 500 MB), downloaded the first time it's needed and refreshed when FC publishes a new one
- `~/.neat/plans/`: build plans you've asked for
- `~/.neat/analytics.json`: the random monthly ID for the anonymous usage statistics, unless you turned them off
- Your OS keychain (macOS Keychain, Windows Credential Manager, or the Linux Secret Service), under the name `neat`: each character's login token

To uninstall, delete the `neat` program and the `~/.neat` folder, remove the `neat` entries from your keychain, and uninstall the plugin (`/plugin uninstall neat@neat`) or remove neat from your desktop app config.

## Troubleshooting

- **`neat setup` seems stuck after "Config directory ready".** It's waiting for your keychain, which is locked and showing an unlock prompt. After 5 seconds neat says so. Look for the prompt and enter your password. On Linux, if you use a remote desktop such as VNC alongside your monitor, the prompt may have opened on the other screen.
- **On Linux, `neat setup` fails with "Object does not exist at path /org/freedesktop/secrets/collection/login".** Your keyring service has no usable default keyring. Open **Passwords and Keys** (`seahorse`; install it with your package manager if needed), click **+**, choose **Password keyring**, name it `Login` and use your login password, then run `neat setup` again. If the new keyring doesn't appear, or the password prompt never shows, log out and back in (or run `systemctl --user restart gnome-keyring-daemon`) and try again.
- **`neat` is "command not found" in some terminals but not others.** `~/.local/bin` was added to your PATH by a login script after that desktop session started. Log out and back in, or add `export PATH="$HOME/.local/bin:$PATH"` to `~/.bashrc`.
- **Claude doesn't see neat's tools.** In Claude Code, check that `neat version` works in a terminal; the plugin runs `neat` from your PATH. In the desktop app, use the full path to `neat` in the config file, because the app doesn't see your terminal's PATH, then quit and reopen the app.
- **macOS says neat can't be opened, or it quits immediately.** Run the `xattr` command from step 1. If you copied a new version over an old one, delete the old file first and copy it again.
- **No browser tab opens when adding a character.** Go to http://127.0.0.1:48211/ in your browser while Claude is waiting, and click **Add another character**. This happens when neat opens the browser on a different screen, such as a remote desktop session, or can't start one at all.
- **Logging in fails because port 48211 is in use.** neat briefly runs a small web page on your own computer at that port to receive EVE's login. Close whatever else is using it and try again.
- **EVE's login page shows "HTTP Error 400. The size of the request headers is too long".** Clear your browser's cookies for `login.eveonline.com` and try again.
- **The first question about items or systems is slow.** neat is downloading EVE's static data. It only does this once, and again when FC publishes an update.
- **A tool says a character "hasn't granted" a scope.** Say yes when Claude offers to grant it, and log in as that same character.
- **A corporation tool says the character "does not have required role(s)".** That character doesn't hold the in-game corporation role the data needs. Ask Claude to use a character that does, or have a director grant the role in game.

## Development

To build neat from source you need Go, `jq`, and your own ESI application from [developers.eveonline.com](https://developers.eveonline.com/), with the callback URL `http://127.0.0.1:48211/callback` and every scope enabled. Put its client ID in a git-ignored `app_config.json`:

```json
{ "esiClientId": "your-esi-client-id" }
```

Builds without an `aptabaseAppKey` in that file send no usage statistics.

Then:

```bash
make install   # builds neat and installs it to ~/.local/bin
go test ./...
```

See [docs/README.md](docs/README.md) for how the code is organized, how releases work, and the design docs behind it.

---

© 2014 CCP hf. All rights reserved. "EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.

neat is a fan project and isn't affiliated with or endorsed by Fenris Creations.
