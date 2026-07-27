# ESI Fleets (Fifteenth Batch) Design

## Status

Approved (autonomous execution). Fifteenth batch: the "Fleets" tag alone (14 endpoints — the
project's largest and most write-heavy batch to date: 10 of 14 endpoints are writes). Every
endpoint requires the calling character to be a member of the target fleet with sufficient fleet
role — ESI itself enforces this via 403, no corp-scoping needed on this project's side.

## Scope

All 14 are **private**, but with an important distinction from every prior "my own character"
batch (Calendar, Fittings, Faction Warfare's `CharacterFwStats`): only `GetCharactersCharacterIdFleet`
is genuinely "my own character's current fleet" (no caller-supplied `fleetId`). Every other
endpoint takes a `fleetId` that is **not** owned by the calling character — it's simply a fleet
they happen to be in right now — plus, for members/wings/squads, a further `memberId`/`wingId`/
`squadId`. Every tool still authenticates via `characterId` (validated via existing `characterExists`)
to select whose stored OAuth token calls ESI; only `get_character_fleet` omits `fleetId` from its
input.

Endpoints: `GetCharactersCharacterIdFleet`, `GetFleetsFleetId`, `PutFleetsFleetId` (write),
`GetFleetsFleetIdMembers`, `PostFleetsFleetIdMembers` (write), `PutFleetsFleetIdMembersMemberId`
(write), `DeleteFleetsFleetIdMembersMemberId` (write), `GetFleetsFleetIdWings`,
`PostFleetsFleetIdWings` (write), `PutFleetsFleetIdWingsWingId` (write),
`DeleteFleetsFleetIdWingsWingId` (write), `PostFleetsFleetIdWingsWingIdSquads` (write),
`PutFleetsFleetIdSquadsSquadId` (write), `DeleteFleetsFleetIdSquadsSquadId` (write).

## Verified API quirks (from source + live httptest.Server verification)

- **Only 2 of 10 writes return a response body**: `PostFleetsFleetIdWings` → `{wing_id}` and
  `PostFleetsFleetIdWingsWingIdSquads` → `{squad_id}`. Every other write returns only
  `(*http.Response, error)`, same shape as `RespondToCalendarEvent`.
- **Those same 2 POSTs take no request body at all** — no generated request-body setter method
  exists for either. A new wing/squad is created nameless; naming happens via a subsequent
  rename call.
- **Generator quirk, verified live**: `PutFleetsFleetIdWingsWingId`'s generated request-body
  setter method is literally named `PutFleetsFleetIdSquadsSquadIdRequest` and takes a
  `PutFleetsFleetIdSquadsSquadIdRequest{Name string}` struct — the code generator reused the
  squad-rename model/method name verbatim for wing rename; there is no separate wing-rename model
  in the installed `v0.0.37` module. This is not a bug to work around in neat's own code — it's
  what the generated client actually exports, confirmed to round-trip correctly (`{"name":"..."}`
  body, 204 response) via a live `httptest.Server` test.
- `PutFleetsFleetIdMembersMemberIdRequest`'s body is itself required by the generated client (a
  nil body triggers a client-side `reportError`) — always constructed, never omitted.
- Response shapes verified: `CharactersCharacterIdFleetGet{FleetBossId, FleetId, Role, SquadId,
  WingId int64/string}` (all required, no pointers; `SquadId`/`WingId` are `-1` when not
  applicable); `FleetsFleetIdGet{IsFreeMove, IsRegistered, IsVoiceEnabled bool, Motd string}`;
  `FleetsFleetIdMembersGetInner{CharacterId, JoinTime, Role, RoleName, ShipTypeId, SolarSystemId,
  SquadId, TakesFleetWarp, WingId}` (`StationId *int64` is the only optional field);
  `FleetsFleetIdWingsGetInner{Id, Name, Squads []{Id, Name}}` (nested squads, no pointers).
- Write request-body shapes: `PutFleetsFleetIdRequest{IsFreeMove *bool, Motd *string}` (both
  optional); `PostFleetsFleetIdMembersRequest{CharacterId int64, Role string, SquadId, WingId
  *int64}`; `PutFleetsFleetIdMembersMemberIdRequest{Role string, SquadId, WingId *int64}`.

## `esi/fleets.go` (14 methods)

```go
func (c *Client) CharacterFleet(ctx context.Context) (*esiapi.CharactersCharacterIdFleetGet, error)
func (c *Client) FleetInfo(ctx context.Context, fleetID int64) (*esiapi.FleetsFleetIdGet, error)
func (c *Client) UpdateFleet(ctx context.Context, fleetID int64, isFreeMove *bool, motd *string) error
func (c *Client) FleetMembers(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error)
func (c *Client) InviteFleetMember(ctx context.Context, fleetID, characterID int64, role string, wingID, squadID *int64) error
func (c *Client) MoveFleetMember(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error
func (c *Client) KickFleetMember(ctx context.Context, fleetID, memberID int64) error
func (c *Client) FleetWings(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdWingsGetInner, error)
func (c *Client) CreateFleetWing(ctx context.Context, fleetID int64) (int64, error)
func (c *Client) RenameFleetWing(ctx context.Context, fleetID, wingID int64, name string) error
func (c *Client) DeleteFleetWing(ctx context.Context, fleetID, wingID int64) error
func (c *Client) CreateFleetSquad(ctx context.Context, fleetID, wingID int64) (int64, error)
func (c *Client) RenameFleetSquad(ctx context.Context, fleetID, squadID int64, name string) error
func (c *Client) DeleteFleetSquad(ctx context.Context, fleetID, squadID int64) error
```

Only `CharacterFleet` uses `int64(c.characterID)` internally — every other method takes `fleetID`
(and `memberId`/`wingId`/`squadId`) as a caller-supplied parameter, since fleet membership is not
"my own" anything the way a calendar or fitting is.

## `mcp/fleets.go` (14 tools)

All private (existence-checked via existing `characterExists`), all taking `characterId` to
select whose stored auth token calls ESI:

- `get_character_fleet` — input `{characterId int32}` (no fleetId), output `{characterId, fleetId, fleetBossId, squadId, wingId int64, role string}`
- `get_fleet` — input `{characterId int32, fleetId int64}`, output `{characterId, fleetId int64, isFreeMove, isRegistered, isVoiceEnabled bool, motd string}`
- `update_fleet` — **write**, input `{characterId int32, fleetId int64, isFreeMove *bool, motd *string}` (both optional, omit to leave unchanged), output echoes input
- `list_fleet_members` — input `{characterId int32, fleetId int64}`, output `{characterId, fleetId int64, members []FleetMember}` where `FleetMember{CharacterID, ShipTypeID, SolarSystemID, SquadID, WingID int64, JoinTime time.Time, Role, RoleName string, StationID *int64, TakesFleetWarp bool}`
- `invite_fleet_member` — **write**, validates `role` against `validFleetRoles` (`fleet_commander`, `wing_commander`, `squad_commander`, `squad_member`) before any IO, input `{characterId int32, fleetId, inviteCharacterId int64, role string, wingId, squadId *int64}`, output echoes input
- `move_fleet_member` — **write**, same role validation, input `{characterId int32, fleetId, memberId int64, role string, wingId, squadId *int64}`, output echoes input
- `kick_fleet_member` — **write**, input `{characterId int32, fleetId, memberId int64}`, output echoes input
- `list_fleet_wings` — input `{characterId int32, fleetId int64}`, output `{characterId, fleetId int64, wings []FleetWing}` where `FleetWing{WingID int64, Name string, Squads []FleetSquad}` and `FleetSquad{SquadID int64, Name string}`
- `create_fleet_wing` — **write**, input `{characterId int32, fleetId int64}`, output `{characterId, fleetId, wingId int64}`
- `rename_fleet_wing` — **write**, input `{characterId int32, fleetId, wingId int64, name string}`, output echoes input
- `delete_fleet_wing` — **write**, input `{characterId int32, fleetId, wingId int64}`, output echoes input
- `create_fleet_squad` — **write**, input `{characterId int32, fleetId, wingId int64}`, output `{characterId, fleetId, wingId, squadId int64}`
- `rename_fleet_squad` — **write**, input `{characterId int32, fleetId, squadId int64, name string}`, output echoes input
- `delete_fleet_squad` — **write**, input `{characterId int32, fleetId, squadId int64}`, output echoes input

`validFleetRoles` is a shared package-level `map[string]bool` used by both `invite_fleet_member`
and `move_fleet_member` for their role-string validation (mirrors `respond_calendar_event`'s
`validCalendarResponses` precedent) — defined once, not duplicated.

## Testing

Same three-layer isolation as every prior batch. `esi/fleets_test.go` covers all 14 methods
(parsing + at least one error path), plus at minimum: a live test proving `CreateFleetWing`'s
response body genuinely contains the new `wing_id` with no request body sent; a live test proving
a representative body-carrying write (`MoveFleetMember`) sends the correct request body and
handles a 204 cleanly; a live test proving `RenameFleetWing`'s "reused squad-rename model" quirk
round-trips correctly (`{"name":...}` body, 204 response) — this is the one genuinely
easy-to-get-wrong spot in this batch and deserves its own explicit proof, not just inline trust in
the doc comment. `mcp/fleets_internal_test.go` covers happy-path + unknown-character for all 14
tools, plus invalid-role-rejection tests for `invite_fleet_member`/`move_fleet_member`.
`mcp/server_test.go` grows from 99 to 113.

## Out of scope

- No client-side enforcement of ESI's own role/wing/squad combination rules for
  `invite_fleet_member`/`move_fleet_member` (e.g. "squad_commander requires both wingId and
  squadId") — ESI is the sole authority on these, consistent with this project's established
  philosophy of not duplicating API-level business rules it doesn't own; the `jsonschema`
  description on each optional field documents the rule for the calling agent's benefit only.
