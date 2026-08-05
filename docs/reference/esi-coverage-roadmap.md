# ESI Coverage Roadmap

Tracks progress covering CCP's full ESI API surface, one batch of resource groups at a time. The
tag order and endpoint counts below come directly from CCP's live OpenAPI 3.0 spec
(`https://esi.evetech.net/meta/openapi-3.0.json`), fetched the same way
`github.com/fnt-eve/goesi-openapi` itself generates its client (see
`docs/superpowers/specs/2026-07-25-esi-coverage-roadmap-design.md` for how this was verified).
Each tag's own row is updated to "Done (PR #N)" once its batch merges — this table is the one
place to check what's left, instead of re-deriving the order from the spec each time.

| # | Tag | Endpoints | Status |
|---|-----|-----------|--------|
| 1 | Access List | 2 | Done (PR #6) |
| 2 | Activities | 3 | Done (PR #6) |
| 3 | Alliance | 4 | Done (PR #6) |
| 4 | Assets | 6 | Done (PR #6) |
| 5 | Calendar | 4 | Done (PR #6) |
| 6 | Character | 14 | Done (PR #6) |
| 7 | Clones | 2 | Done (PR #6) |
| 8 | Contacts | 9 | Done (PR #6) |
| 9 | Contracts | 9 | Done (PR #6) |
| 10 | Corporation | 22 | Done (PR #6) |
| 11 | Corporation Projects | 4 | Done (PR #6) |
| 12 | Dogma | 5 | Done (PR #6) |
| 13 | Faction Warfare | 8 | Done (PR #6) |
| 14 | Fittings | 3 | Done (PR #6) |
| 15 | Fleets | 14 | Done (PR #6) |
| 16 | Freelance Jobs | 6 | Done (PR #6) |
| 17 | Incursions | 1 | Done (PR #6) |
| 18 | Industry | 8 | Done (PR #6) |
| 19 | Insurance | 1 | Done (PR #6) |
| 20 | Killmails | 3 | Done (PR #6) |
| 21 | Location | 3 | Done (PR #6) |
| 22 | Loyalty | 2 | Done (PR #6) |
| 23 | Mail | 9 | Done (PR #6) |
| 24 | Market | 11 | Done (PR #6) |
| 25 | Meta | 4 | Done (PR #6) |
| 26 | Planetary Interaction | 4 | Done (PR #6) |
| 27 | Routes | 1 | Done (PR #6) |
| 28 | Search | 1 | Done (PR #6) |
| 29 | Skills | 3 | Done (PR #6) |
| 30 | Sovereignty | 2 | Done (PR #6) |
| 31 | Status | 1 | Done (PR #6) |
| 32 | Structures | 6 | Done (PR #6) |
| 33 | Universe | 30 | Done (PR #6) |
| 34 | User Interface | 5 | Done (PR #6) |
| 35 | Wallet | 6 | Done (PR #5, corrected in PR #15 -- see note) |
| 36 | Wars | 3 | Done (PR #6) |
| 37 | Military Campaigns | 6 | Blocked -- see note |

219 total endpoints across the original 36 tags (225 including Military Campaigns).

**Wallet correction:** PR #5 marked all 6 Wallet endpoints "Done," but only `WalletBalance`/
`get_wallet_balance` (1 of 6) was actually implemented — the character wallet journal/transactions
and all three corporation wallet endpoints (balance, journal, transactions) were never built. This
was discovered when a user hit a wall trying to read a corporation's wallet balance (the exact gap:
`get_corporation_divisions` only exposes division *names*, and ESI's
`GET /corporations/{corporation_id}/wallets/` was never wired up). The remaining 5 endpoints were
added in PR #15, using the same conventions as every other batch in this table.

**Military Campaigns (blocked):** PR #15's full-coverage audit re-fetched CCP's live OpenAPI spec
with the current compatibility date (`?compatibility_date=2026-08-04` — the bare, undated
`/meta/openapi-3.0.json` silently returns a stale/default spec that hides newer tags, which is why
this wasn't caught by re-running the original tag count) and found a 37th tag, "Military
Campaigns" (6 endpoints: `GET /military-campaigns`, `GET /military-campaigns/{campaign_id}`,
`GET /military-campaigns/{campaign_id}/objectives`,
`GET /military-campaigns/{campaign_id}/objectives/{objective_id}`,
`GET /characters/{character_id}/military-campaigns/objectives`,
`GET /characters/{character_id}/military-campaigns/objectives/{objective_id}`), added by CCP after
the original sweep. It cannot be implemented yet: `github.com/fnt-eve/goesi-openapi` v0.0.37 (the
latest published version) has no generated bindings for it at all, and this project's hard rule is
that every ESI call goes through that generated client -- never a hand-rolled HTTP request. Needs
an upstream `goesi-openapi` release that regenerates against a spec including this tag before a
batch can be planned. The rest of the audit (every one of the original 36 tags' operationIds
diffed against both the live dated spec and the installed `goesi-openapi` module, plus every
`allScopes` entry checked against real CCP scopes in both directions) found no other gaps.

## Conventions (apply to every batch)

- One `esi/<tag_snake_case>.go` + one `mcp/<tag_snake_case>.go` per tag, even when a plan batches
  several tags together — only the plan/PR boundary batches tags, file boundaries stay
  per-resource.
- A plan covers a cluster of adjacent tags (in the table's order above) sized so the batch isn't
  trivially small.
- Every `esi/` method returns exactly what ESI returns, including bare IDs, with zero
  name-resolution enrichment. `type_id`-like fields, the six location-entity ID kinds
  (system/region/constellation/station/planet/moon), and now race/bloodline/faction/group IDs
  across the non-Universe tags are all enriched at the `mcp/` layer via `mcp/enrich.go`'s resolver
  functions (see `docs/superpowers/specs/2026-07-28-sde-type-id-enrichment-design.md`,
  `docs/superpowers/specs/2026-07-30-sde-location-id-enrichment-design.md`, and
  `docs/superpowers/specs/2026-07-30-sde-race-bloodline-faction-group-enrichment-design.md`) — the
  generic ambiguous `location_id` fields remain the one open follow-up.
- Every new batch must verify its required OAuth scopes are already present in
  `mcp/add_character.go`'s `allScopes`, the same way this plan did, before assuming no re-auth is
  needed.
