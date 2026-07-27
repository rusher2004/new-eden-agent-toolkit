# ESI User Interface + Wars Implementation Plan (final batch of the sweep)

> **For agentic workers:** This plan is executed directly by the orchestrator (no subagent
> dispatch), per the user's explicit instruction after this session's subagent spawn cap was hit
> during the Sovereignty batch. Steps use checkbox (`- [ ]`) syntax for tracking only.

**Goal:** Add ESI coverage for the last two remaining tags -- "User Interface" (5 write endpoints
controlling the querying character's own live EVE client) and "Wars" (3 public read endpoints).
This is the final batch of the ESI coverage sweep.

**Architecture:** `esi/user_interface.go` (5 write methods, all returning only `error`) +
`esi/wars.go` (3 public read methods) + `mcp/user_interface.go` (5 write tools) + `mcp/wars.go` (3
public tools).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi@v0.0.37`,
`github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Current tool count baseline: **209** (verify with `grep -c "mcpsdk.AddTool" mcp/server.go`
  before any doc-count edit).
- Branch: `worktree-esi-access-list-activities` (shared PR #6). Base commit before Task 1: `154a97c`.
- OAuth2 scopes `esi-ui.write_waypoint.v1` and `esi-ui.open_window.v1` are ALREADY present in
  `mcp/add_character.go`'s `allScopes` -- no scope changes needed.
- All 5 User Interface endpoints return only `(*http.Response, error)` from the generated client
  -- no response body model exists for any of them (verified individually). Facade methods return
  only `error`, same as `esi/calendar.go`'s `RespondToCalendarEvent`.
- `GetWars` uses a `max_war_id` cursor parameter, NOT this project's usual `page`/`X-Pages`
  pattern -- verified: no `.Page()` setter on its request builder. `GetWarsWarIdKillmails` DOES
  use the usual `page`/`X-Pages` pattern -- verified: it has a `.Page()` setter. Do not confuse
  the two within this same tag.
- `GetWarsWarIdKillmails` reuses the exact same generated type as the existing Killmails batch
  (`CharactersCharacterIdKillmailsRecentGetInner`) -- reuse `mcp/killmails.go`'s existing
  `KillmailRef` type and `toKillmailRefs` helper, do not redeclare them.
- The 5 User Interface tools are write/action tools by this project's established convention:
  verb-first tool name, description states it changes state on the character's live client, no
  confirmation parameter (mirrors `respond_calendar_event` exactly).
- The full design spec, including every verified field name and request/response shape, is at
  `docs/superpowers/specs/2026-07-27-esi-user-interface-wars-design.md` -- read it before starting
  either task.
- This is the LAST batch of the sweep. After Task 2, every row in
  `docs/reference/esi-coverage-roadmap.md` reads either "This plan" or "Done (PR #N)" -- no row
  should remain "Planned".

---

## Task 1: `esi/user_interface.go` + `esi/wars.go`

**Files:**
- Create: `esi/user_interface.go`
- Create: `esi/user_interface_test.go`
- Create: `esi/wars.go`
- Create: `esi/wars_test.go`

**Interfaces:**
- Consumes: `Client.api.UserInterfaceAPI`, `Client.api.WarsAPI` (confirmed present on the
  generated client struct), `describeError`, `totalPagesFromHeader`.
- Produces:
  ```go
  func (c *Client) SetAutopilotWaypoint(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error
  func (c *Client) OpenContractWindow(ctx context.Context, contractID int64) error
  func (c *Client) OpenInformationWindow(ctx context.Context, targetID int64) error
  func (c *Client) OpenMarketDetailsWindow(ctx context.Context, typeID int64) error
  func (c *Client) OpenNewMailWindow(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error
  func (c *Client) Wars(ctx context.Context, maxWarID int64) ([]int64, error)
  func (c *Client) WarDetail(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error)
  func (c *Client) WarKillmails(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error)
  ```
  These exact signatures are what Task 2 will call.

- [ ] **Step 1: Write `esi/user_interface.go`** using the exact code in the design spec's
  "esi/user_interface.go -- 5 methods" section.

- [ ] **Step 2: Write `esi/user_interface_test.go`**

  Follow the `httptest.Server` + `esi.ExportedNewClientWithOptions(srv.Client(), 12345,
  goesi.ClientOptions{BaseURL, UserAgent})` pattern (all 5 are private/"my own character"). Write
  10 tests (5 methods x success + HTTP-error). For success cases, assert the request reached the
  server with the right path/query params (waypoint: `destination_id`, `add_to_beginning`,
  `clear_other_waypoints`; contract: `contract_id`; information: `target_id`; marketdetails:
  `type_id`), and for `OpenNewMailWindow`, decode the JSON request body and assert `body`,
  `recipients`, `subject` round-trip (mirroring `esi/assets.go`'s POST-body verification
  precedent). The test server should respond with `w.WriteHeader(http.StatusNoContent)` (204, no
  body) for the success cases, since that's ESI's real response shape here.

- [ ] **Step 3: Write `esi/wars.go`** using the exact code in the design spec's "esi/wars.go -- 3
  methods" section.

- [ ] **Step 4: Write `esi/wars_test.go`**

  Follow the `httptest.Server` + `esi.ExportedNewPublicClientWithOptions` pattern (all 3 are
  public). Write 6 tests (3 methods x success + HTTP-error), plus one extra test confirming
  `Wars` omits the `max_war_id` query param when called with `0` and includes it when called with
  a nonzero value (mirroring `esi/planetary_interaction.go`'s `page=0` omission precedent), and one
  extra test confirming `WarDetail`'s response round-trips `Aggressor`/`Defender`/`Allies` with at
  least one alliance-shaped and one corporation-shaped entry, asserting no cross-field bleed
  (e.g. a corporation-shaped ally has `AllianceId == nil`).

- [ ] **Step 5: Run all new tests**

  Run: `go test ./esi/... -run 'TestSetAutopilotWaypoint|TestOpenContractWindow|TestOpenInformationWindow|TestOpenMarketDetailsWindow|TestOpenNewMailWindow|TestWars|TestWarDetail|TestWarKillmails' -v`
  Expected: all pass.

- [ ] **Step 6: `go build ./... && go vet ./... && gofmt -l .`, then full suite + race**

  Run: `go build ./... && go vet ./... && gofmt -l .` then `go test ./... -race`
  Expected: clean, all packages `ok`.

- [ ] **Step 7: Commit**

```bash
git add esi/user_interface.go esi/user_interface_test.go esi/wars.go esi/wars_test.go
git commit -m "feat: add user interface and wars esi facades (autopilot/window writes, war lookups)"
```

---

## Task 2: `mcp/user_interface.go` + `mcp/wars.go` + registration + final docs sync

**Files:**
- Create: `mcp/user_interface.go`
- Create: `mcp/user_interface_internal_test.go`
- Create: `mcp/wars.go`
- Create: `mcp/wars_internal_test.go`
- Modify: `mcp/server.go` (register 8 tools, update package doc comment tail)
- Modify: `mcp/server_test.go` (append 8 tool names)
- Modify: `CLAUDE.md` (tool count + tool list)
- Modify: `README.md` (tool count + tool list)
- Modify: `docs/reference/esi-coverage-roadmap.md` (rows 34 "User Interface" and 36 "Wars":
  `Planned` -> `This plan`)

- [ ] **Step 1: Write `mcp/user_interface.go`**

  5 write tools, each mirroring `respond_calendar_event`'s exact structure (`characterExists`
  gate, verb-first name, output echoes input params back plus `characterId` since there's no ESI
  response body):

  - `set_autopilot_waypoint` -- input `{CharacterID int32, DestinationID int64, AddToBeginning
    bool, ClearOtherWaypoints bool}`.
  - `open_contract_window` -- input `{CharacterID int32, ContractID int64}`.
  - `open_information_window` -- input `{CharacterID int32, TargetID int64}`.
  - `open_market_details_window` -- input `{CharacterID int32, TypeID int64}`.
  - `open_new_mail_window` -- input `{CharacterID int32, Recipients []int64, Subject, Body string,
    ToCorpOrAllianceID, ToMailingListID *int64}`.

  Each `xHandler` follows the exact pattern: `characterExists` gate -> call the swappable
  package-level var -> wrap and return any error -> on success return an output struct echoing
  the input.

- [ ] **Step 2: Write `mcp/user_interface_internal_test.go`**

  Per tool: MapsFields (echoes input correctly) + PropagatesError + UnknownCharacter (15 tests
  total), following `mcp/skills_internal_test.go`'s exact private-tool test pattern (keyring mock +
  `config.NewStoreAt(t.TempDir()...)`).

- [ ] **Step 3: Write `mcp/wars.go`**

  Using the exact types from the design spec's "mcp/wars.go -- 3 tools" section (`WarParty`,
  `WarAlly`, `War`). 3 public tools:

  - `list_wars` -- input `{MaxWarID int64,omitempty}`, output `{WarIDs []int64}`.
  - `get_war` -- input `{WarID int64}`, output `War`.
  - `list_war_killmails` -- input `{WarID int64, Page int32,omitempty}`, output `{Killmails
    []KillmailRef, Page, TotalPages int32}` -- reuse `mcp/killmails.go`'s `KillmailRef`/
    `toKillmailRefs`, do not redeclare.

- [ ] **Step 4: Write `mcp/wars_internal_test.go`**

  6 tests (MapsFields + PropagatesError per tool), plus one variant test for `get_war` confirming
  an alliance-shaped `Aggressor` and a corporation-shaped `Defender` both map correctly with no
  cross-field bleed (mirroring the Sovereignty/Structures batches' variant-test style).

- [ ] **Step 5: Register the 8 tools in `mcp/server.go`**

  First run `grep -c "mcpsdk.AddTool" mcp/server.go` for the real baseline (this plan assumes
  209). Append after the last Universe block. Update the package doc comment tail to end with the
  final tool name, `resolve_universe_names, set_autopilot_waypoint, open_contract_window,
  open_information_window, open_market_details_window, open_new_mail_window, list_wars, get_war,
  and list_war_killmails registered.`

- [ ] **Step 6: Update `mcp/server_test.go`**

  Append all 8 new tool names after the `resolve_universe_names` entry.

- [ ] **Step 7: Run all new tests, then full suite + race**

  Run: `go test ./mcp/... -run 'Autopilot|Window|Mail|Wars|War' -v`
  then `go build ./... && go vet ./... && gofmt -l .`
  then `go test ./... -race`
  Expected: all pass, clean, all packages `ok`.

- [ ] **Step 8: Update `CLAUDE.md` and `README.md`**

  Change "two hundred nine" to "two hundred seventeen" (verify the real current count first --
  should read 217 after this task). Append all 8 new tool names to the tool list.

- [ ] **Step 9: Flip the roadmap doc's final two rows**

  Change row 34 (`| 34 | User Interface | 5 | Planned |`) to
  `| 34 | User Interface | 5 | This plan |`, and row 36 (`| 36 | Wars | 3 | Planned |`) to
  `| 36 | Wars | 3 | This plan |`. Verify with `git diff` that ONLY these 2 rows changed. After
  this edit, confirm with a quick read of the whole table that no row still reads "Planned" --
  this is the last batch of the sweep.

- [ ] **Step 10: Commit**

```bash
git add mcp/user_interface.go mcp/user_interface_internal_test.go mcp/wars.go mcp/wars_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add user interface and wars MCP tools (final batch of the ESI coverage sweep)"
```

- [ ] **Step 11: Push**

```bash
git push origin worktree-esi-access-list-activities
```
