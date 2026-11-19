# ESI User Interface + Wars Design (34th/35th batches -- final batch of the sweep)

**Status:** approved (autonomous sweep). Written directly by the orchestrator, no subagent
dispatch, per the user's standing instruction after the session's subagent spawn cap was hit
during the Sovereignty batch.

## Scope

The last two remaining ESI tags, combined into one plan since both are small (8 endpoints total):

| Endpoint | operationId | Security | Paginated |
|---|---|---|---|
| `POST /ui/autopilot/waypoint` | `PostUiAutopilotWaypoint` | `esi-ui.write_waypoint.v1` | No |
| `POST /ui/openwindow/contract` | `PostUiOpenwindowContract` | `esi-ui.open_window.v1` | No |
| `POST /ui/openwindow/information` | `PostUiOpenwindowInformation` | `esi-ui.open_window.v1` | No |
| `POST /ui/openwindow/marketdetails` | `PostUiOpenwindowMarketdetails` | `esi-ui.open_window.v1` | No |
| `POST /ui/openwindow/newmail` | `PostUiOpenwindowNewmail` | `esi-ui.open_window.v1` | No |
| `GET /wars` | `GetWars` | public | No (cursor-based via `max_war_id`, not page-based) |
| `GET /wars/{war_id}` | `GetWarsWarId` | public | No |
| `GET /wars/{war_id}/killmails` | `GetWarsWarIdKillmails` | public | **Yes** |

Both `esi-ui.write_waypoint.v1` and `esi-ui.open_window.v1` are already present in
`mcp/add_character.go`'s `allScopes` (added in an earlier batch) -- no re-auth needed.

## User Interface: this project's second write-return-shape family

All 5 User Interface endpoints return only `(*http.Response, error)` from the generated
client -- verified individually from `api_user_interface.go`: none has a response body model.
This is the same write-return-shape already established by Calendar's `RespondToCalendarEvent`
(esi/calendar.go) -- ESI gives a 204 with no body, so the facade method returns only `error`.

These 5 tools control the querying character's actual live EVE client (move the autopilot,
pop open a UI window) -- they are unambiguously write/action tools by this project's existing
convention (verb-first tool name, description states it changes state, no confirmation
parameter -- same as `respond_calendar_event`, `add_character_contacts`, etc.). Every one of the 5
is scoped to "my own character" (the querying character's own client), same as
`esi/skills.go`'s pattern -- no separate target-character parameter, because ESI only ever acts on
the authenticated character's own client.

Exact request parameters, verified individually from `api_user_interface.go`'s request builders
(none of the first 4 accepts a JSON body -- all are query/path parameters only; only Newmail has a
JSON body):

- **`PostUiAutopilotWaypoint`**: `addToBeginning bool` (required), `clearOtherWaypoints bool`
  (required), `destinationId int64` (required, the item/station/structure/system ID to set as
  waypoint).
- **`PostUiOpenwindowContract`**: `contractId int64` (required).
- **`PostUiOpenwindowInformation`**: `targetId int64` (required, any character/corporation/
  alliance/item/system ID ESI's client can show an info window for).
- **`PostUiOpenwindowMarketdetails`**: `typeId int64` (required).
- **`PostUiOpenwindowNewmail`**: body `PostUiOpenwindowNewmailRequest{Body string, Recipients
  []int64, Subject string, ToCorpOrAllianceId *int64, ToMailingListId *int64}` (mail recipients
  are the *displayed* recipients list; `ToCorpOrAllianceId`/`ToMailingListId` are mutually
  exclusive optional extra mailing-group targets, per ESI's own doc comment on the field).

## Wars

`GetWars` returns bare `[]int64` (war IDs), optionally filtered by `maxWarId` (a cursor: "return
wars with ID less than this", FC's own descending-pagination convention for this endpoint --
NOT the `page`/`X-Pages` pattern used elsewhere in this project. Verified: `ApiGetWarsRequest` has
no `.Page()` setter, only `.MaxWarId(id int64)`).

`GetWarsWarId` returns `WarsWarIdGet` -- flat except 3 verified-non-recursive nested types:
`Aggressor{AllianceId *int64, CorporationId *int64, IskDestroyed float64, ShipsKilled int64}`,
`Defender` (identical shape to Aggressor), and `Allies []{AllianceId *int64, CorporationId
*int64}`. `Aggressor` and `Defender` share an identical shape -- flattened to one shared MCP type
(`WarParty`), reusing the shared-type-for-near-identical-models pattern.

`GetWarsWarIdKillmails` reuses `CharactersCharacterIdKillmailsRecentGetInner` (the same generated
type used by the existing Killmails batch) -- reuse `mcp/killmails.go`'s existing `KillmailRef`
type and `toKillmailRefs` helper directly, do not redeclare. This endpoint IS paginated (verified:
`ApiGetWarsWarIdKillmailsRequest` has a `.Page()` setter) -- use `totalPagesFromHeader`.

## esi/user_interface.go -- 5 methods

```go
// Package esi's user_interface.go covers all 5 of "User Interface"'s endpoints -- these are all
// writes that control the querying character's own live EVE client (move the autopilot, open a
// UI window). All 5 return only an error -- ESI gives a 204 with no response body for every one
// of them (verified individually from api_user_interface.go), same write-return-shape as
// esi/calendar.go's RespondToCalendarEvent. All 5 are "my own character" -- no separate target-
// character parameter, since ESI only ever acts on the authenticated character's own client.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func (c *Client) SetAutopilotWaypoint(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
	resp, err := c.api.UserInterfaceAPI.PostUiAutopilotWaypoint(ctx).
		DestinationId(destinationID).AddToBeginning(addToBeginning).ClearOtherWaypoints(clearOtherWaypoints).
		Execute()
	if err != nil {
		return fmt.Errorf("set autopilot waypoint: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenContractWindow(ctx context.Context, contractID int64) error {
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowContract(ctx).ContractId(contractID).Execute()
	if err != nil {
		return fmt.Errorf("open contract window: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenInformationWindow(ctx context.Context, targetID int64) error {
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowInformation(ctx).TargetId(targetID).Execute()
	if err != nil {
		return fmt.Errorf("open information window: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenMarketDetailsWindow(ctx context.Context, typeID int64) error {
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowMarketdetails(ctx).TypeId(typeID).Execute()
	if err != nil {
		return fmt.Errorf("open market details window: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenNewMailWindow(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
	req := esiapi.PostUiOpenwindowNewmailRequest{Body: body, Recipients: recipients, Subject: subject}
	if toCorpOrAllianceID != nil {
		req.ToCorpOrAllianceId = toCorpOrAllianceID
	}
	if toMailingListID != nil {
		req.ToMailingListId = toMailingListID
	}
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowNewmail(ctx).PostUiOpenwindowNewmailRequest(req).Execute()
	if err != nil {
		return fmt.Errorf("open new mail window: %w", describeError(err, resp))
	}
	return nil
}
```

## esi/wars.go -- 3 methods

```go
// Package esi's wars.go covers all 3 of "Wars"'s endpoints. All 3 are public. GetWars uses
// FC's cursor-based pagination (max_war_id, "wars with ID less than this"), NOT this project's
// usual page/X-Pages pattern -- verified individually: no .Page() setter on GetWars's request
// builder. GetWarsWarIdKillmails, unlike the other 2, IS page/X-Pages paginated -- verified
// individually.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func (c *Client) Wars(ctx context.Context, maxWarID int64) ([]int64, error) {
	req := c.api.WarsAPI.GetWars(ctx)
	if maxWarID > 0 {
		req = req.MaxWarId(maxWarID)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get wars: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) WarDetail(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error) {
	result, resp, err := c.api.WarsAPI.GetWarsWarId(ctx, warID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get war detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) WarKillmails(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	req := c.api.WarsAPI.GetWarsWarIdKillmails(ctx, warID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get war killmails: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
```

## mcp/user_interface.go -- 5 tools

All 5 are private write tools, mirroring `respond_calendar_event`'s exact structure
(`characterExists` gate, verb-first name, description states it changes state):
`set_autopilot_waypoint`, `open_contract_window`, `open_information_window`,
`open_market_details_window`, `open_new_mail_window`. Every output echoes back the input
parameters plus `characterId` (there's no ESI response body to report) -- same as
`respond_calendar_event`'s output shape.

## mcp/wars.go -- 3 tools

```go
type WarParty struct {
	AllianceID    *int64  `json:"allianceId,omitempty"`
	CorporationID *int64  `json:"corporationId,omitempty"`
	IskDestroyed  float64 `json:"iskDestroyed"`
	ShipsKilled   int64   `json:"shipsKilled"`
}

type WarAlly struct {
	AllianceID    *int64 `json:"allianceId,omitempty"`
	CorporationID *int64 `json:"corporationId,omitempty"`
}

type War struct {
	WarID         int64      `json:"warId"`
	Aggressor     WarParty   `json:"aggressor"`
	Defender      WarParty   `json:"defender"`
	Allies        []WarAlly  `json:"allies,omitempty"`
	Declared      time.Time  `json:"declared"`
	Finished      *time.Time `json:"finished,omitempty"`
	Mutual        bool       `json:"mutual"`
	OpenForAllies bool       `json:"openForAllies"`
	Retracted     *time.Time `json:"retracted,omitempty"`
	Started       *time.Time `json:"started,omitempty"`
}
```

- `list_wars` (public) -- input `{MaxWarID int64,omitempty}`, output `{WarIDs []int64}`.
- `get_war` (public) -- input `{WarID int64}`, output `War`.
- `list_war_killmails` (public, paginated) -- input `{WarID int64, Page int32,omitempty}`, output
  `{Killmails []KillmailRef, Page, TotalPages int32}` -- reuses `mcp/killmails.go`'s existing
  `KillmailRef`/`toKillmailRefs`, does not redeclare them.

## Testing approach

Same as every prior batch: `httptest.Server` + raw JSON bodies, `esi.ExportedNewClientWithOptions`
for the 5 private UI writes and the paginated killmails endpoint (well, killmails is actually
public -- use `ExportedNewPublicClientWithOptions` for all 3 Wars methods), covering success +
HTTP-error paths, plus assertions that each UI write's query parameters/request body round-trip
correctly (mirroring `esi/calendar.go`'s `RespondToCalendarEvent` test and `esi/search.go`'s query-
param assertion fix from earlier in this sweep). `mcp/user_interface_internal_test.go` and
`mcp/wars_internal_test.go` swap package-level vars per the established pattern; the 5 write tools
each get a MapsFields (echoes input back) + PropagatesError + UnknownCharacter test (15 tests);
Wars gets MapsFields + PropagatesError per tool (6 tests), plus one variant test confirming
Aggressor/Defender/Allies alliance-vs-corporation shapes both round-trip without bleed.

## Task breakdown

Given the moderate size (8 endpoints, one genuinely novel write-return-shape pattern reused from
Calendar, one novel pagination cursor pattern for Wars), split into 2 tasks mirroring the
Structures/Sovereignty precedent:

- **Task 1**: `esi/user_interface.go` + `esi/user_interface_test.go` (5 methods) AND
  `esi/wars.go` + `esi/wars_test.go` (3 methods) -- both esi/ files in one task since together
  they're a similar size to a single-file task elsewhere in this sweep.
- **Task 2**: `mcp/user_interface.go` + `mcp/user_interface_internal_test.go` (5 tools) AND
  `mcp/wars.go` + `mcp/wars_internal_test.go` (3 tools) + `mcp/server.go` registration (8 tools) +
  `mcp/server_test.go`'s expected-tools-list update (209 -> 217 tools) + docs sync
  (`CLAUDE.md`/`README.md` tool count and list, roadmap doc rows 34 and 36 flipped to "This plan"
  -- this is the LAST batch of the sweep, so after this task every row in the roadmap table reads
  either "This plan" or "Done (PR #N)").

Both tasks done directly by the orchestrator (no subagent dispatch), verified independently with
`go build`/`go vet`/`gofmt -l`/targeted `go test -v`, full `go test ./... -race` before each
commit.
