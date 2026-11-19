# ESI Fleets (Fifteenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 14 endpoints of the "Fleets" ESI tag as a new `esi/fleets.go` facade and a new `mcp/fleets.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `FleetsAPI`, reusing the existing `describeError` helper. One new `mcp/` tool file with 14 private tools (`esi.NewClient` + existing `characterExists`), 10 of which are writes — the project's largest write-heavy batch to date. Handles a genuine generated-client quirk (wing rename reuses squad rename's model/method name) transparently at the facade layer.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `e6a3032`) — no new branch, no new PR.
- Current tool count baseline: **99**. After this plan: **113**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`) — do not redeclare it.
- Only `CharacterFleet` uses the client's own baked-in `characterID` (`int64(c.characterID)`) — every other `esi/` method takes `fleetID` (and `memberID`/`wingID`/`squadID` where relevant) as an explicit caller-supplied parameter, since a fleet is not "my own" anything.
- **Verified quirk — do not "fix" this**: `RenameFleetWing` must call `.PutFleetsFleetIdSquadsSquadIdRequest(esiapi.PutFleetsFleetIdSquadsSquadIdRequest{Name: name})` — the generated client reuses the squad-rename model/method name for wing rename (there is no separate wing-rename model in the installed module). This has been verified live via `httptest.Server` — it round-trips correctly.
- `CreateFleetWing`/`CreateFleetSquad` take **no request body** (the generated client has no body-setter method for either) and **do** return a response body (`{wing_id}`/`{squad_id}`) — the only 2 of the 10 writes in this tag that return a body.
- All other 8 writes (`UpdateFleet`, `InviteFleetMember`, `MoveFleetMember`, `KickFleetMember`, `RenameFleetWing`, `DeleteFleetWing`, `RenameFleetSquad`, `DeleteFleetSquad`) return only `error`, mirroring `RespondToCalendarEvent`'s exact shape.
- `invite_fleet_member`/`move_fleet_member` validate their `role` input against a shared `validFleetRoles` map (defined once) before any IO, per the `respond_calendar_event`/`validCalendarResponses` write-tool-validation precedent.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean), including 3 live `httptest.Server` tests proving the wing-creation response body, a representative body-carrying write's request/204 handling, and the wing-rename model-reuse quirk. Implementers should still run these themselves as part of each task.

---

### Task 1: `esi/fleets.go` facade (14 methods)

**Files:**
- Create: `esi/fleets.go`
- Test: `esi/fleets_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `Client.api`/`Client.characterID` (`esi/client.go`).
- Produces: 14 methods on `*Client` for Task 2's `mcp/fleets.go` to call.

- [ ] **Step 1: Write `esi/fleets.go`**

```go
// Package esi's fleets.go covers all 14 of "Fleets"'s endpoints. Every endpoint requires the
// calling character to be a member of the target fleet with sufficient fleet role -- ESI itself
// enforces this via 403, no corp-scoping needed on our side. Unlike Calendar/Fittings, most of
// these endpoints take a fleetId that is NOT "my own" anything -- it's simply a fleet the
// calling character happens to be in right now. Only CharacterFleet (GetCharactersCharacterIdFleet)
// is genuinely "my own character's current fleet"; every other method takes fleetId (and, for
// members/wings/squads, a further memberId/wingId/squadId) as a caller-supplied parameter, not
// int64(c.characterID).
//
// Real API quirks discovered while implementing this file against goesi-openapi@v0.0.37 (see
// verification tests in fleets_test.go):
//
//   - Of the 10 write endpoints here, only PostFleetsFleetIdWings and
//     PostFleetsFleetIdWingsWingIdSquads return a response body (the newly created wing_id /
//     squad_id respectively). All other 8 writes return only (*http.Response, error), same
//     shape as RespondToCalendarEvent.
//   - PostFleetsFleetIdWings and PostFleetsFleetIdWingsWingIdSquads take NO request body at all
//     -- there is no generated `...Request(...)` body-setter method for either of them. A new
//     wing/squad is created nameless; naming happens via a subsequent
//     PutFleetsFleetIdWingsWingId/PutFleetsFleetIdSquadsSquadId call.
//   - PutFleetsFleetIdWingsWingId's generated request-body setter is misnamed
//     `PutFleetsFleetIdSquadsSquadIdRequest` and takes a `PutFleetsFleetIdSquadsSquadIdRequest`
//     struct -- the code generator reused the squad-rename model/method name for wing rename
//     verbatim (there is no separate wing-rename model in the generated module). This is not a
//     copy-paste bug in this file -- it is what the installed v0.0.37 module actually exports;
//     RenameFleetWing below calls it exactly as generated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterFleet returns the client's own character's current fleet membership (fleet ID, boss,
// role, wing/squad). This is the one Fleets endpoint that is genuinely "my own character" --
// unlike every other method in this file, it takes no caller-supplied fleetId.
func (c *Client) CharacterFleet(ctx context.Context) (*esiapi.CharactersCharacterIdFleetGet, error) {
	result, resp, err := c.api.FleetsAPI.GetCharactersCharacterIdFleet(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character fleet: %w", describeError(err, resp))
	}
	return result, nil
}

// FleetInfo returns settings for the fleet identified by fleetID -- a fleet the client's
// character happens to be in, not necessarily one it owns or commands.
func (c *Client) FleetInfo(ctx context.Context, fleetID int64) (*esiapi.FleetsFleetIdGet, error) {
	result, resp, err := c.api.FleetsAPI.GetFleetsFleetId(ctx, fleetID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fleet info: %w", describeError(err, resp))
	}
	return result, nil
}

// UpdateFleet updates fleet-wide settings. Both fields are optional in ESI's request model --
// pass nil to leave a field unchanged. This is a write: it changes state on FC's server. ESI
// returns no response body (204), same shape as RespondToCalendarEvent.
func (c *Client) UpdateFleet(ctx context.Context, fleetID int64, isFreeMove *bool, motd *string) error {
	resp, err := c.api.FleetsAPI.PutFleetsFleetId(ctx, fleetID).
		PutFleetsFleetIdRequest(esiapi.PutFleetsFleetIdRequest{IsFreeMove: isFreeMove, Motd: motd}).
		Execute()
	if err != nil {
		return fmt.Errorf("update fleet: %w", describeError(err, resp))
	}
	return nil
}

// FleetMembers returns every member currently in the fleet, with their role, ship, location,
// and squad/wing assignment.
func (c *Client) FleetMembers(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error) {
	result, resp, err := c.api.FleetsAPI.GetFleetsFleetIdMembers(ctx, fleetID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fleet members: %w", describeError(err, resp))
	}
	return result, nil
}

// InviteFleetMember invites characterID into the fleet with the given role. wingID/squadID are
// required or forbidden depending on role -- see ESI's own rule (enforced server-side, not
// re-validated here): fleet_commander wants neither, wing_commander wants only wingID,
// squad_commander wants both, squad_member wants both-or-neither. Write, no response body.
func (c *Client) InviteFleetMember(ctx context.Context, fleetID, characterID int64, role string, wingID, squadID *int64) error {
	resp, err := c.api.FleetsAPI.PostFleetsFleetIdMembers(ctx, fleetID).
		PostFleetsFleetIdMembersRequest(esiapi.PostFleetsFleetIdMembersRequest{
			CharacterId: characterID, Role: role, SquadId: squadID, WingId: wingID,
		}).Execute()
	if err != nil {
		return fmt.Errorf("invite fleet member: %w", describeError(err, resp))
	}
	return nil
}

// MoveFleetMember moves memberID to a new role/squad/wing within the fleet. Same wingID/squadID
// role-dependent rule as InviteFleetMember, except squad_member always requires both. Write, no
// response body.
func (c *Client) MoveFleetMember(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error {
	resp, err := c.api.FleetsAPI.PutFleetsFleetIdMembersMemberId(ctx, fleetID, memberID).
		PutFleetsFleetIdMembersMemberIdRequest(esiapi.PutFleetsFleetIdMembersMemberIdRequest{
			Role: role, SquadId: squadID, WingId: wingID,
		}).Execute()
	if err != nil {
		return fmt.Errorf("move fleet member: %w", describeError(err, resp))
	}
	return nil
}

// KickFleetMember removes memberID from the fleet entirely. Write, no response body.
func (c *Client) KickFleetMember(ctx context.Context, fleetID, memberID int64) error {
	resp, err := c.api.FleetsAPI.DeleteFleetsFleetIdMembersMemberId(ctx, fleetID, memberID).Execute()
	if err != nil {
		return fmt.Errorf("kick fleet member: %w", describeError(err, resp))
	}
	return nil
}

// FleetWings returns every wing in the fleet, each with its nested squads.
func (c *Client) FleetWings(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdWingsGetInner, error) {
	result, resp, err := c.api.FleetsAPI.GetFleetsFleetIdWings(ctx, fleetID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fleet wings: %w", describeError(err, resp))
	}
	return result, nil
}

// CreateFleetWing creates a new, nameless wing in the fleet and returns its assigned wing ID --
// the only field ESI's response contains. Unlike most writes in this file, this one DOES return
// a response body. There is no request body at all -- rename the wing afterward via
// RenameFleetWing if a name is wanted.
func (c *Client) CreateFleetWing(ctx context.Context, fleetID int64) (int64, error) {
	result, resp, err := c.api.FleetsAPI.PostFleetsFleetIdWings(ctx, fleetID).Execute()
	if err != nil {
		return 0, fmt.Errorf("create fleet wing: %w", describeError(err, resp))
	}
	return result.WingId, nil
}

// RenameFleetWing renames wingID. Write, no response body.
//
// Quirk: goesi-openapi@v0.0.37's generated request-body setter for this endpoint is named
// PutFleetsFleetIdSquadsSquadIdRequest and takes a PutFleetsFleetIdSquadsSquadIdRequest struct
// -- the generator reused the squad-rename model/method verbatim for wing rename. This is not a
// mistake in this file; it is what the installed module actually exports (verified directly
// against api_fleets.go and confirmed to compile/round-trip correctly in fleets_test.go).
func (c *Client) RenameFleetWing(ctx context.Context, fleetID, wingID int64, name string) error {
	resp, err := c.api.FleetsAPI.PutFleetsFleetIdWingsWingId(ctx, fleetID, wingID).
		PutFleetsFleetIdSquadsSquadIdRequest(esiapi.PutFleetsFleetIdSquadsSquadIdRequest{Name: name}).
		Execute()
	if err != nil {
		return fmt.Errorf("rename fleet wing: %w", describeError(err, resp))
	}
	return nil
}

// DeleteFleetWing deletes wingID. ESI requires the wing (and any squads within it) be empty
// first. Write, no response body.
func (c *Client) DeleteFleetWing(ctx context.Context, fleetID, wingID int64) error {
	resp, err := c.api.FleetsAPI.DeleteFleetsFleetIdWingsWingId(ctx, fleetID, wingID).Execute()
	if err != nil {
		return fmt.Errorf("delete fleet wing: %w", describeError(err, resp))
	}
	return nil
}

// CreateFleetSquad creates a new, nameless squad within wingID and returns its assigned squad
// ID -- the only field ESI's response contains. Like CreateFleetWing, this DOES return a
// response body, and takes no request body.
func (c *Client) CreateFleetSquad(ctx context.Context, fleetID, wingID int64) (int64, error) {
	result, resp, err := c.api.FleetsAPI.PostFleetsFleetIdWingsWingIdSquads(ctx, fleetID, wingID).Execute()
	if err != nil {
		return 0, fmt.Errorf("create fleet squad: %w", describeError(err, resp))
	}
	return result.SquadId, nil
}

// RenameFleetSquad renames squadID. Write, no response body.
func (c *Client) RenameFleetSquad(ctx context.Context, fleetID, squadID int64, name string) error {
	resp, err := c.api.FleetsAPI.PutFleetsFleetIdSquadsSquadId(ctx, fleetID, squadID).
		PutFleetsFleetIdSquadsSquadIdRequest(esiapi.PutFleetsFleetIdSquadsSquadIdRequest{Name: name}).
		Execute()
	if err != nil {
		return fmt.Errorf("rename fleet squad: %w", describeError(err, resp))
	}
	return nil
}

// DeleteFleetSquad deletes squadID. ESI requires the squad be empty first. Write, no response
// body.
func (c *Client) DeleteFleetSquad(ctx context.Context, fleetID, squadID int64) error {
	resp, err := c.api.FleetsAPI.DeleteFleetsFleetIdSquadsSquadId(ctx, fleetID, squadID).Execute()
	if err != nil {
		return fmt.Errorf("delete fleet squad: %w", describeError(err, resp))
	}
	return nil
}
```

- [ ] **Step 2: Write `esi/fleets_test.go`**

Read `esi/calendar_test.go` and `esi/fittings_test.go` first to copy their exact patterns for testing writes with request bodies, writes returning only `error`/204, and writes returning a response body. Cover:

- `TestCharacterFleet_ParsesResponse` — serve `{fleet_boss_id, fleet_id, role, squad_id, wing_id}` fully populated, assert every field maps; use the authenticated-client harness and assert the request path reflects the client's own baked-in characterID.
- `TestCharacterFleet_HTTPError` — serve a 404, assert error non-nil, wraps a description.
- `TestFleetInfo_ParsesResponse` — serve `{is_free_move, is_registered, is_voice_enabled, motd}`, assert every field maps.
- `TestUpdateFleet_SendsCorrectRequestBody` — live server, assert the request body deserializes to exactly the `is_free_move`/`motd` values passed, and a 204 response yields nil error.
- `TestFleetMembers_ParsesResponse` — serve a 1-element array with every field populated including `station_id`, assert every field maps.
- `TestInviteFleetMember_SendsCorrectRequestBody` — live server, assert body has `character_id`/`role`/`squad_id`/`wing_id` exactly as passed.
- `TestMoveFleetMember_SendsBodyAndHandlesNoContent` — live server, assert request body (`role`/`squad_id`/`wing_id`) and that a 204 response yields nil error.
- `TestKickFleetMember_HandlesNoContentAndHTTPError` — one subtest for a clean 204, one for a 404 error path.
- `TestFleetWings_ParsesResponse` — serve a 1-element array with a nested non-empty `squads` array, assert every field maps including squad entries.
- `TestCreateFleetWing_ReturnsWingIDAndSendsNoBody` — live server, assert the request body is empty and the returned `wingID` matches the served `{wing_id}`.
- `TestRenameFleetWing_UsesSquadRenameBodyShape` — live server, assert the request body is exactly `{"name":"..."}` and a 204 response yields nil error — this is the one genuinely easy-to-get-wrong spot in this batch (the misnamed generated method), so this test must exist and must actually exercise the real generated call, not a mock.
- `TestDeleteFleetWing_HandlesNoContentAndHTTPError` — one subtest for 204, one for error.
- `TestCreateFleetSquad_ReturnsSquadIDAndSendsNoBody` — live server, assert empty request body and returned `squadID` matches served `{squad_id}`.
- `TestRenameFleetSquad_SendsCorrectRequestBody` — live server, assert `{"name":"..."}` body and 204 handling.
- `TestDeleteFleetSquad_HandlesNoContentAndHTTPError` — one subtest for 204, one for error.

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run "CharacterFleet|FleetInfo|UpdateFleet|FleetMembers|InviteFleetMember|MoveFleetMember|KickFleetMember|FleetWings|CreateFleetWing|RenameFleetWing|DeleteFleetWing|CreateFleetSquad|RenameFleetSquad|DeleteFleetSquad" -v`
Expected: all tests PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/fleets.go esi/fleets_test.go
git commit -m "feat: add esi/fleets.go facade (14 methods, generated-client rename quirk verified)"
```

---

### Task 2: `mcp/fleets.go` tools (14 tools), server registration, and docs sync

**Files:**
- Create: `mcp/fleets.go`
- Test: `mcp/fleets_internal_test.go`
- Modify: `mcp/server.go` (register 14 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 99 to 113)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Fleets row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "ninety-nine" → "one hundred thirteen")
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: all 14 `esi/fleets.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent`.
- Produces: 14 `mcpsdk.ToolHandlerFor[...]`-returning functions for `mcp/server.go`, all `(version, clientID string, store *config.Store)`: `getCharacterFleetHandler`, `getFleetHandler`, `updateFleetHandler`, `listFleetMembersHandler`, `inviteFleetMemberHandler`, `moveFleetMemberHandler`, `kickFleetMemberHandler`, `listFleetWingsHandler`, `createFleetWingHandler`, `renameFleetWingHandler`, `deleteFleetWingHandler`, `createFleetSquadHandler`, `renameFleetSquadHandler`, `deleteFleetSquadHandler`.

- [ ] **Step 1: Write `mcp/fleets.go`**

```go
// fleets.go covers all 14 "Fleets" tools. Every tool is private -- it authenticates as a
// specific characterId (like Calendar/Fittings) -- but unlike those resource groups, most of
// these tools also take a fleetId (and, for members/wings/squads, a further
// memberId/wingId/squadId) that is NOT "my own" anything: it's simply a fleet the calling
// character happens to be in right now, with sufficient fleet role. ESI itself enforces role
// requirements via 403; this file does not re-derive or check them client-side. Only
// get_character_fleet omits fleetId entirely -- it discovers the calling character's current
// fleet from ESI rather than taking one as input.
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// validFleetRoles are ESI's own accepted values for a fleet member's role, used by
// invite_fleet_member and move_fleet_member.
var validFleetRoles = map[string]bool{
	"fleet_commander": true,
	"wing_commander":  true,
	"squad_commander": true,
	"squad_member":    true,
}

// --- get_character_fleet (private) ---

// GetCharacterFleetOutput is get_character_fleet's tool result -- the calling character's
// current fleet membership.
type GetCharacterFleetOutput struct {
	CharacterID int32  `json:"characterId"`
	FleetID     int64  `json:"fleetId"`
	FleetBossID int64  `json:"fleetBossId"`
	Role        string `json:"role"`
	SquadID     int64  `json:"squadId"`
	WingID      int64  `json:"wingId"`
}

var getCharacterFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFleetOutput, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return GetCharacterFleetOutput{}, fmt.Errorf("build ESI client: %w", err)
	}
	fleet, err := client.CharacterFleet(ctx)
	if err != nil {
		return GetCharacterFleetOutput{}, err
	}
	return GetCharacterFleetOutput{
		CharacterID: characterID, FleetID: fleet.FleetId, FleetBossID: fleet.FleetBossId,
		Role: fleet.Role, SquadID: fleet.SquadId, WingID: fleet.WingId,
	}, nil
}

type GetCharacterFleetInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

func getCharacterFleetHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterFleetInput, GetCharacterFleetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFleetInput) (*mcpsdk.CallToolResult, GetCharacterFleetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFleetOutput{}, err
		}
		if !found {
			return nil, GetCharacterFleetOutput{}, fmt.Errorf("get character fleet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		out, err := getCharacterFleet(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterFleetOutput{}, fmt.Errorf("get character fleet: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_fleet (private) ---

// GetFleetOutput is get_fleet's tool result -- fleet-wide settings.
type GetFleetOutput struct {
	CharacterID    int32  `json:"characterId"`
	FleetID        int64  `json:"fleetId"`
	IsFreeMove     bool   `json:"isFreeMove"`
	IsRegistered   bool   `json:"isRegistered"`
	IsVoiceEnabled bool   `json:"isVoiceEnabled"`
	Motd           string `json:"motd"`
}

var getFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) (GetFleetOutput, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return GetFleetOutput{}, fmt.Errorf("build ESI client: %w", err)
	}
	fleet, err := client.FleetInfo(ctx, fleetID)
	if err != nil {
		return GetFleetOutput{}, err
	}
	return GetFleetOutput{
		CharacterID: characterID, FleetID: fleetID, IsFreeMove: fleet.IsFreeMove,
		IsRegistered: fleet.IsRegistered, IsVoiceEnabled: fleet.IsVoiceEnabled, Motd: fleet.Motd,
	}, nil
}

type GetFleetInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must be a member of the fleet"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
}

func getFleetHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetFleetInput, GetFleetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFleetInput) (*mcpsdk.CallToolResult, GetFleetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetFleetOutput{}, err
		}
		if !found {
			return nil, GetFleetOutput{}, fmt.Errorf("get fleet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		out, err := getFleet(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID)
		if err != nil {
			return nil, GetFleetOutput{}, fmt.Errorf("get fleet: %w", err)
		}
		return nil, out, nil
	}
}

// --- update_fleet (private, write) ---

var updateFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64, isFreeMove *bool, motd *string) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.UpdateFleet(ctx, fleetID, isFreeMove, motd)
}

type UpdateFleetInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check as -- must have fleet_commander role"`
	FleetID     int64   `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	IsFreeMove  *bool   `json:"isFreeMove,omitempty" jsonschema:"whether free-move is enabled; omit to leave unchanged"`
	Motd        *string `json:"motd,omitempty" jsonschema:"new fleet MOTD in FC-flavoured HTML; omit to leave unchanged"`
}
type UpdateFleetOutput struct {
	CharacterID int32   `json:"characterId"`
	FleetID     int64   `json:"fleetId"`
	IsFreeMove  *bool   `json:"isFreeMove,omitempty"`
	Motd        *string `json:"motd,omitempty"`
}

func updateFleetHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[UpdateFleetInput, UpdateFleetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateFleetInput) (*mcpsdk.CallToolResult, UpdateFleetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, UpdateFleetOutput{}, err
		}
		if !found {
			return nil, UpdateFleetOutput{}, fmt.Errorf("update fleet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := updateFleet(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.IsFreeMove, in.Motd); err != nil {
			return nil, UpdateFleetOutput{}, fmt.Errorf("update fleet: %w", err)
		}
		return nil, UpdateFleetOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, IsFreeMove: in.IsFreeMove, Motd: in.Motd}, nil
	}
}

// --- list_fleet_members (private) ---

// FleetMember is one fleet member's role, ship, and location.
type FleetMember struct {
	CharacterID    int64     `json:"characterId"`
	JoinTime       time.Time `json:"joinTime"`
	Role           string    `json:"role"`
	RoleName       string    `json:"roleName"`
	ShipTypeID     int64     `json:"shipTypeId"`
	SolarSystemID  int64     `json:"solarSystemId"`
	SquadID        int64     `json:"squadId"`
	StationID      *int64    `json:"stationId,omitempty"`
	TakesFleetWarp bool      `json:"takesFleetWarp"`
	WingID         int64     `json:"wingId"`
}

var listFleetMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetMember, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	members, err := client.FleetMembers(ctx, fleetID)
	if err != nil {
		return nil, err
	}
	out := make([]FleetMember, len(members))
	for i, m := range members {
		out[i] = FleetMember{
			CharacterID: m.CharacterId, JoinTime: m.JoinTime, Role: m.Role, RoleName: m.RoleName,
			ShipTypeID: m.ShipTypeId, SolarSystemID: m.SolarSystemId, SquadID: m.SquadId,
			StationID: m.StationId, TakesFleetWarp: m.TakesFleetWarp, WingID: m.WingId,
		}
	}
	return out, nil
}

type ListFleetMembersInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must be a member of the fleet"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
}
type ListFleetMembersOutput struct {
	CharacterID int32         `json:"characterId"`
	FleetID     int64         `json:"fleetId"`
	Members     []FleetMember `json:"members"`
}

func listFleetMembersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListFleetMembersInput, ListFleetMembersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFleetMembersInput) (*mcpsdk.CallToolResult, ListFleetMembersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListFleetMembersOutput{}, err
		}
		if !found {
			return nil, ListFleetMembersOutput{}, fmt.Errorf("list fleet members for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		members, err := listFleetMembers(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID)
		if err != nil {
			return nil, ListFleetMembersOutput{}, fmt.Errorf("list fleet members: %w", err)
		}
		return nil, ListFleetMembersOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, Members: members}, nil
	}
}

// --- invite_fleet_member (private, write) ---

var inviteFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.InviteFleetMember(ctx, fleetID, inviteCharacterID, role, wingID, squadID)
}

type InviteFleetMemberInput struct {
	CharacterID       int32  `json:"characterId" jsonschema:"the character ID to check as -- must have appropriate fleet role"`
	FleetID           int64  `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	InviteCharacterID int64  `json:"inviteCharacterId" jsonschema:"the character ID to invite into the fleet"`
	Role              string `json:"role" jsonschema:"one of fleet_commander, wing_commander, squad_commander, squad_member"`
	WingID            *int64 `json:"wingId,omitempty" jsonschema:"required for wing_commander and squad_commander roles"`
	SquadID           *int64 `json:"squadId,omitempty" jsonschema:"required for squad_commander role"`
}
type InviteFleetMemberOutput struct {
	CharacterID       int32  `json:"characterId"`
	FleetID           int64  `json:"fleetId"`
	InviteCharacterID int64  `json:"inviteCharacterId"`
	Role              string `json:"role"`
}

// inviteFleetMemberHandler is a write tool: it changes state on FC's server, so -- like
// respond_calendar_event -- it validates its input BEFORE loading config.Store or calling ESI.
func inviteFleetMemberHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[InviteFleetMemberInput, InviteFleetMemberOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in InviteFleetMemberInput) (*mcpsdk.CallToolResult, InviteFleetMemberOutput, error) {
		if !validFleetRoles[in.Role] {
			return nil, InviteFleetMemberOutput{}, fmt.Errorf("invite fleet member: invalid role %q, must be one of fleet_commander, wing_commander, squad_commander, squad_member", in.Role)
		}
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, InviteFleetMemberOutput{}, err
		}
		if !found {
			return nil, InviteFleetMemberOutput{}, fmt.Errorf("invite fleet member for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := inviteFleetMember(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.InviteCharacterID, in.Role, in.WingID, in.SquadID); err != nil {
			return nil, InviteFleetMemberOutput{}, fmt.Errorf("invite fleet member: %w", err)
		}
		return nil, InviteFleetMemberOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, InviteCharacterID: in.InviteCharacterID, Role: in.Role}, nil
	}
}

// --- move_fleet_member (private, write) ---

var moveFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, memberID int64, role string, wingID, squadID *int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.MoveFleetMember(ctx, fleetID, memberID, role, wingID, squadID)
}

type MoveFleetMemberInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check as -- must have appropriate fleet role"`
	FleetID     int64  `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	MemberID    int64  `json:"memberId" jsonschema:"the fleet member's character ID, from list_fleet_members"`
	Role        string `json:"role" jsonschema:"one of fleet_commander, wing_commander, squad_commander, squad_member"`
	WingID      *int64 `json:"wingId,omitempty" jsonschema:"required for wing_commander, squad_commander, and squad_member roles"`
	SquadID     *int64 `json:"squadId,omitempty" jsonschema:"required for squad_commander and squad_member roles"`
}
type MoveFleetMemberOutput struct {
	CharacterID int32  `json:"characterId"`
	FleetID     int64  `json:"fleetId"`
	MemberID    int64  `json:"memberId"`
	Role        string `json:"role"`
}

func moveFleetMemberHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[MoveFleetMemberInput, MoveFleetMemberOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in MoveFleetMemberInput) (*mcpsdk.CallToolResult, MoveFleetMemberOutput, error) {
		if !validFleetRoles[in.Role] {
			return nil, MoveFleetMemberOutput{}, fmt.Errorf("move fleet member: invalid role %q, must be one of fleet_commander, wing_commander, squad_commander, squad_member", in.Role)
		}
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, MoveFleetMemberOutput{}, err
		}
		if !found {
			return nil, MoveFleetMemberOutput{}, fmt.Errorf("move fleet member for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := moveFleetMember(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.MemberID, in.Role, in.WingID, in.SquadID); err != nil {
			return nil, MoveFleetMemberOutput{}, fmt.Errorf("move fleet member: %w", err)
		}
		return nil, MoveFleetMemberOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, MemberID: in.MemberID, Role: in.Role}, nil
	}
}

// --- kick_fleet_member (private, write) ---

var kickFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, memberID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.KickFleetMember(ctx, fleetID, memberID)
}

type KickFleetMemberInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must have appropriate fleet role"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	MemberID    int64 `json:"memberId" jsonschema:"the fleet member's character ID to kick, from list_fleet_members"`
}
type KickFleetMemberOutput struct {
	CharacterID int32 `json:"characterId"`
	FleetID     int64 `json:"fleetId"`
	MemberID    int64 `json:"memberId"`
}

func kickFleetMemberHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[KickFleetMemberInput, KickFleetMemberOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in KickFleetMemberInput) (*mcpsdk.CallToolResult, KickFleetMemberOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, KickFleetMemberOutput{}, err
		}
		if !found {
			return nil, KickFleetMemberOutput{}, fmt.Errorf("kick fleet member for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := kickFleetMember(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.MemberID); err != nil {
			return nil, KickFleetMemberOutput{}, fmt.Errorf("kick fleet member: %w", err)
		}
		return nil, KickFleetMemberOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, MemberID: in.MemberID}, nil
	}
}

// --- list_fleet_wings (private) ---

// FleetSquad is one squad nested within a fleet wing.
type FleetSquad struct {
	SquadID int64  `json:"squadId"`
	Name    string `json:"name"`
}

// FleetWing is one fleet wing, with its nested squads.
type FleetWing struct {
	WingID int64        `json:"wingId"`
	Name   string       `json:"name"`
	Squads []FleetSquad `json:"squads"`
}

var listFleetWings = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetWing, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	wings, err := client.FleetWings(ctx, fleetID)
	if err != nil {
		return nil, err
	}
	out := make([]FleetWing, len(wings))
	for i, w := range wings {
		squads := make([]FleetSquad, len(w.Squads))
		for j, s := range w.Squads {
			squads[j] = FleetSquad{SquadID: s.Id, Name: s.Name}
		}
		out[i] = FleetWing{WingID: w.Id, Name: w.Name, Squads: squads}
	}
	return out, nil
}

type ListFleetWingsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must be a member of the fleet"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
}
type ListFleetWingsOutput struct {
	CharacterID int32       `json:"characterId"`
	FleetID     int64       `json:"fleetId"`
	Wings       []FleetWing `json:"wings"`
}

func listFleetWingsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListFleetWingsInput, ListFleetWingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFleetWingsInput) (*mcpsdk.CallToolResult, ListFleetWingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListFleetWingsOutput{}, err
		}
		if !found {
			return nil, ListFleetWingsOutput{}, fmt.Errorf("list fleet wings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		wings, err := listFleetWings(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID)
		if err != nil {
			return nil, ListFleetWingsOutput{}, fmt.Errorf("list fleet wings: %w", err)
		}
		return nil, ListFleetWingsOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, Wings: wings}, nil
	}
}

// --- create_fleet_wing (private, write) ---

var createFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) (int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CreateFleetWing(ctx, fleetID)
}

type CreateFleetWingInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must have fleet_commander role"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
}
type CreateFleetWingOutput struct {
	CharacterID int32 `json:"characterId"`
	FleetID     int64 `json:"fleetId"`
	WingID      int64 `json:"wingId"`
}

func createFleetWingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[CreateFleetWingInput, CreateFleetWingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateFleetWingInput) (*mcpsdk.CallToolResult, CreateFleetWingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateFleetWingOutput{}, err
		}
		if !found {
			return nil, CreateFleetWingOutput{}, fmt.Errorf("create fleet wing for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		wingID, err := createFleetWing(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID)
		if err != nil {
			return nil, CreateFleetWingOutput{}, fmt.Errorf("create fleet wing: %w", err)
		}
		return nil, CreateFleetWingOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: wingID}, nil
	}
}

// --- rename_fleet_wing (private, write) ---

var renameFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64, name string) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.RenameFleetWing(ctx, fleetID, wingID, name)
}

type RenameFleetWingInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check as -- must have fleet_commander role"`
	FleetID     int64  `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	WingID      int64  `json:"wingId" jsonschema:"the wing ID to rename, from list_fleet_wings"`
	Name        string `json:"name" jsonschema:"the wing's new name"`
}
type RenameFleetWingOutput struct {
	CharacterID int32  `json:"characterId"`
	FleetID     int64  `json:"fleetId"`
	WingID      int64  `json:"wingId"`
	Name        string `json:"name"`
}

func renameFleetWingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[RenameFleetWingInput, RenameFleetWingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RenameFleetWingInput) (*mcpsdk.CallToolResult, RenameFleetWingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, RenameFleetWingOutput{}, err
		}
		if !found {
			return nil, RenameFleetWingOutput{}, fmt.Errorf("rename fleet wing for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := renameFleetWing(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.WingID, in.Name); err != nil {
			return nil, RenameFleetWingOutput{}, fmt.Errorf("rename fleet wing: %w", err)
		}
		return nil, RenameFleetWingOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: in.WingID, Name: in.Name}, nil
	}
}

// --- delete_fleet_wing (private, write) ---

var deleteFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.DeleteFleetWing(ctx, fleetID, wingID)
}

type DeleteFleetWingInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must have fleet_commander role"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	WingID      int64 `json:"wingId" jsonschema:"the wing ID to delete, from list_fleet_wings -- the wing and any squads within it must be empty"`
}
type DeleteFleetWingOutput struct {
	CharacterID int32 `json:"characterId"`
	FleetID     int64 `json:"fleetId"`
	WingID      int64 `json:"wingId"`
}

func deleteFleetWingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[DeleteFleetWingInput, DeleteFleetWingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteFleetWingInput) (*mcpsdk.CallToolResult, DeleteFleetWingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteFleetWingOutput{}, err
		}
		if !found {
			return nil, DeleteFleetWingOutput{}, fmt.Errorf("delete fleet wing for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := deleteFleetWing(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.WingID); err != nil {
			return nil, DeleteFleetWingOutput{}, fmt.Errorf("delete fleet wing: %w", err)
		}
		return nil, DeleteFleetWingOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: in.WingID}, nil
	}
}

// --- create_fleet_squad (private, write) ---

var createFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64) (int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CreateFleetSquad(ctx, fleetID, wingID)
}

type CreateFleetSquadInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must have fleet_commander or wing_commander role"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	WingID      int64 `json:"wingId" jsonschema:"the wing ID to create the squad in, from list_fleet_wings"`
}
type CreateFleetSquadOutput struct {
	CharacterID int32 `json:"characterId"`
	FleetID     int64 `json:"fleetId"`
	WingID      int64 `json:"wingId"`
	SquadID     int64 `json:"squadId"`
}

func createFleetSquadHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[CreateFleetSquadInput, CreateFleetSquadOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateFleetSquadInput) (*mcpsdk.CallToolResult, CreateFleetSquadOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateFleetSquadOutput{}, err
		}
		if !found {
			return nil, CreateFleetSquadOutput{}, fmt.Errorf("create fleet squad for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		squadID, err := createFleetSquad(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.WingID)
		if err != nil {
			return nil, CreateFleetSquadOutput{}, fmt.Errorf("create fleet squad: %w", err)
		}
		return nil, CreateFleetSquadOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: in.WingID, SquadID: squadID}, nil
	}
}

// --- rename_fleet_squad (private, write) ---

var renameFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, squadID int64, name string) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.RenameFleetSquad(ctx, fleetID, squadID, name)
}

type RenameFleetSquadInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check as -- must have fleet_commander or wing_commander role"`
	FleetID     int64  `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	SquadID     int64  `json:"squadId" jsonschema:"the squad ID to rename, from list_fleet_wings"`
	Name        string `json:"name" jsonschema:"the squad's new name"`
}
type RenameFleetSquadOutput struct {
	CharacterID int32  `json:"characterId"`
	FleetID     int64  `json:"fleetId"`
	SquadID     int64  `json:"squadId"`
	Name        string `json:"name"`
}

func renameFleetSquadHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[RenameFleetSquadInput, RenameFleetSquadOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RenameFleetSquadInput) (*mcpsdk.CallToolResult, RenameFleetSquadOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, RenameFleetSquadOutput{}, err
		}
		if !found {
			return nil, RenameFleetSquadOutput{}, fmt.Errorf("rename fleet squad for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := renameFleetSquad(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.SquadID, in.Name); err != nil {
			return nil, RenameFleetSquadOutput{}, fmt.Errorf("rename fleet squad: %w", err)
		}
		return nil, RenameFleetSquadOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, SquadID: in.SquadID, Name: in.Name}, nil
	}
}

// --- delete_fleet_squad (private, write) ---

var deleteFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, squadID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.DeleteFleetSquad(ctx, fleetID, squadID)
}

type DeleteFleetSquadInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check as -- must have fleet_commander or wing_commander role"`
	FleetID     int64 `json:"fleetId" jsonschema:"the fleet ID, from get_character_fleet"`
	SquadID     int64 `json:"squadId" jsonschema:"the squad ID to delete, from list_fleet_wings -- the squad must be empty"`
}
type DeleteFleetSquadOutput struct {
	CharacterID int32 `json:"characterId"`
	FleetID     int64 `json:"fleetId"`
	SquadID     int64 `json:"squadId"`
}

func deleteFleetSquadHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[DeleteFleetSquadInput, DeleteFleetSquadOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteFleetSquadInput) (*mcpsdk.CallToolResult, DeleteFleetSquadOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteFleetSquadOutput{}, err
		}
		if !found {
			return nil, DeleteFleetSquadOutput{}, fmt.Errorf("delete fleet squad for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := deleteFleetSquad(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FleetID, in.SquadID); err != nil {
			return nil, DeleteFleetSquadOutput{}, fmt.Errorf("delete fleet squad: %w", err)
		}
		return nil, DeleteFleetSquadOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, SquadID: in.SquadID}, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Write `mcp/fleets_internal_test.go`**

Follow `mcp/fittings_internal_test.go`'s pattern (package-level `var` override + `t.Cleanup`; `keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). Write one happy-path test per tool (asserting every field the stub returns), one unknown-character test per tool (14 + 14 = 28), plus 2 extra tests for invalid-role rejection:

- `TestGetCharacterFleetHandler_MapsFields` / `_UnknownCharacter`
- `TestGetFleetHandler_MapsFields` / `_UnknownCharacter`
- `TestUpdateFleetHandler_MapsFields` / `_UnknownCharacter`
- `TestListFleetMembersHandler_MapsFields` / `_UnknownCharacter`
- `TestInviteFleetMemberHandler_MapsFields` / `_UnknownCharacter` / `TestInviteFleetMemberHandler_InvalidRole`
- `TestMoveFleetMemberHandler_MapsFields` / `_UnknownCharacter` / `TestMoveFleetMemberHandler_InvalidRole`
- `TestKickFleetMemberHandler_Succeeds` / `_UnknownCharacter`
- `TestListFleetWingsHandler_MapsFields` (including nested squads) / `_UnknownCharacter`
- `TestCreateFleetWingHandler_MapsFields` / `_UnknownCharacter`
- `TestRenameFleetWingHandler_MapsFields` / `_UnknownCharacter`
- `TestDeleteFleetWingHandler_Succeeds` / `_UnknownCharacter`
- `TestCreateFleetSquadHandler_MapsFields` / `_UnknownCharacter`
- `TestRenameFleetSquadHandler_MapsFields` / `_UnknownCharacter`
- `TestDeleteFleetSquadHandler_Succeeds` / `_UnknownCharacter`

That's 30 test functions total (14 happy-path/succeeds + 14 unknown-character + 2 invalid-role).

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run "CharacterFleet|GetFleet|UpdateFleet|FleetMembers|FleetMember|FleetWing|FleetSquad" -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 14 tools in `mcp/server.go`**

Add these 14 `mcpsdk.AddTool` blocks immediately after the existing `delete_character_fitting` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_fleet",
		Description: "Get a logged-in EVE Online character's current fleet membership.",
	}, getCharacterFleetHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fleet",
		Description: "Get a fleet's settings, using a logged-in character's fleet membership to authorize.",
	}, getFleetHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "update_fleet",
		Description: "Update a fleet's settings — this changes state on EVE's server.",
	}, updateFleetHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fleet_members",
		Description: "List a fleet's members, using a logged-in character's fleet membership to authorize.",
	}, listFleetMembersHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "invite_fleet_member",
		Description: "Invite a character into a fleet — this changes state on EVE's server.",
	}, inviteFleetMemberHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "move_fleet_member",
		Description: "Move a fleet member to a new role, wing, or squad — this changes state on EVE's server.",
	}, moveFleetMemberHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "kick_fleet_member",
		Description: "Remove a member from a fleet — this changes state on EVE's server.",
	}, kickFleetMemberHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fleet_wings",
		Description: "List a fleet's wings and their nested squads, using a logged-in character's fleet membership to authorize.",
	}, listFleetWingsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "create_fleet_wing",
		Description: "Create a new wing in a fleet — this changes state on EVE's server.",
	}, createFleetWingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rename_fleet_wing",
		Description: "Rename a fleet wing — this changes state on EVE's server.",
	}, renameFleetWingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_fleet_wing",
		Description: "Delete a fleet wing — this changes state on EVE's server.",
	}, deleteFleetWingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "create_fleet_squad",
		Description: "Create a new squad within a fleet wing — this changes state on EVE's server.",
	}, createFleetSquadHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rename_fleet_squad",
		Description: "Rename a fleet squad — this changes state on EVE's server.",
	}, renameFleetSquadHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_fleet_squad",
		Description: "Delete a fleet squad — this changes state on EVE's server.",
	}, deleteFleetSquadHandler(version, clientID, store))

	return server
}
```

Update the package doc comment above `NewServer` to append the 14 new tool names to its running
list (ending in `..., delete_character_fitting, get_character_fleet, get_fleet, update_fleet,
list_fleet_members, invite_fleet_member, move_fleet_member, kick_fleet_member, list_fleet_wings,
create_fleet_wing, rename_fleet_wing, delete_fleet_wing, create_fleet_squad, rename_fleet_squad,
and delete_fleet_squad registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 14 names to the `[]string{...}` literal, right after `"delete_character_fitting",`:

```go
			"get_character_fleet", "get_fleet", "update_fleet", "list_fleet_members",
			"invite_fleet_member", "move_fleet_member", "kick_fleet_member", "list_fleet_wings",
			"create_fleet_wing", "rename_fleet_wing", "delete_fleet_wing", "create_fleet_squad",
			"rename_fleet_squad", "delete_fleet_squad",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 113 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "ninety-nine" to "one hundred thirteen".
- Append the 14 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Fleets row from `Planned` to `This plan`.

- [ ] **Step 8: Commit**

```bash
git add mcp/fleets.go mcp/fleets_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add fleets MCP tools (14 tools: fleet/members/wings/squads)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff e6a3032..HEAD` — the design spec commit forward).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
