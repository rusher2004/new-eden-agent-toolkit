# Loopback OAuth2 callback with a character session page

**Status:** Implemented
**Date:** 2026-11-19

## Problem

The foundation spec chose the `eveauth-neat://` custom scheme as the ESI OAuth2 redirect. The browser hands a custom-scheme URL to the OS instead of loading it, so after the user authorized, the tab stayed on CCP's login page with nothing telling them they were done. No code on neat's side could change that: the tab never sent neat a request it could answer.

## Decision

Redirect to `http://127.0.0.1:48211/callback` and have `add_character` serve it from a short-lived loopback HTTP server, the RFC 8252 §7.3 pattern used by `gh`, `gcloud` and similar CLIs. The browser tab lands on neat's own page.

This reverses the foundation spec's "custom scheme only" decision. Its stated reason was that the scheme doesn't depend on a free port. That still holds, but the cost of a fixed port (a clear error if something else holds it) is smaller than the tab never knowing it's finished, and dropping the scheme removes a lot: `urlscheme/` with its per-OS registration (macOS's embedded Swift wrapper, Windows' registry, Linux's `.desktop` file, the latter two never exercised outside CI), the `callback-handler` subcommand, the `~/.neat/bin/neat` launcher copy, the two-socket Unix IPC and its cross-process stale-socket race, and the Swift build steps in CI, CodeQL and GoReleaser. RFC 8252 §8.1 already noted that PKCE, not the redirect type, is what protects the authorization code, so this doesn't weaken the flow.

**Fixed port.** CCP matches the registered callback URL exactly. Its docs describe a single URL per application and don't mention §7.3's any-port loopback matching, so the port has to be known in advance. 48211 is outside common development ranges. The URI uses the `127.0.0.1` literal (§8.3), and the server binds only that address.

**Exclusivity.** `addCharacterMu.TryLock` rejects a second call in the same process with `ErrAddCharacterInProgress`. Another `neat` process is kept out by the TCP bind itself, which is atomic and leaves nothing behind if the process crashes.

## The session page

The user chose a page that lists their characters and offers to add more, over a bare "you can close this tab" page.

- **Routes.** `GET /callback` validates the state, exchanges the code, saves the token, calls `AddOptions.OnAdded` (the MCP layer upserts the character into config immediately), then redirects with 303 to `/` so a reload can't replay the code. `GET /` lists saved characters (from `AddOptions.SavedCharacters`) with portraits from `images.evetech.net`, marks the ones added this session, and shows the latest success or error. `POST /login` starts another login and redirects to CCP. `POST /done` renders "All set" and ends the session. `POST /ping` and `POST /closed` come from `web/page.js`.
- **When the session ends.** `add_character` blocks until one of these happens:
  - the user clicks Done;
  - the page's `pagehide` beacon reports the tab closed, which ends the session after a 5-second grace so a reload, which fires the same beacon, can pick it back up;
  - the page has stopped pinging for 30 seconds, because the beacon was lost;
  - a started login gets no callback within 5 minutes;
  - the MCP request is cancelled.

  A close beacon is ignored while a login is in flight, since "Add another" navigates the page to CCP and fires it too.
- **Result.** The tool returns every character added. It returns an error only when none was: the last login error if there was one, otherwise the timeout.

## Security

- **Host check.** Every request must carry `Host: 127.0.0.1:48211`. A DNS-rebinding page sends its own hostname, so it can't read the character list.
- **CSRF.** Every POST must carry a random per-session page token embedded in the page, so another site can't start or end the session through the user's browser.
- **State.** Each OAuth2 state is single-use and compared in constant time. A callback with a mismatched state is rejected without disturbing the login actually in flight.
- **Headers.** A strict CSP allows images only from `images.evetech.net`, scripts and fetches only from self, and forms to post only to self and CCP's login origin. Responses also send `Referrer-Policy: no-referrer` and `Cache-Control: no-store`.

## Migration

The ESI application's callback URL must be changed to `http://127.0.0.1:48211/callback`. Existing refresh tokens are unaffected. Old scheme registrations and the launcher copy are left in place, inert, and listed under Known gaps in `CLAUDE.md`.
