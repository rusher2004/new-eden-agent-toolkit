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
	if result == nil {
		return 0, fmt.Errorf("create fleet wing: empty response")
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
	if result == nil {
		return 0, fmt.Errorf("create fleet squad: empty response")
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
