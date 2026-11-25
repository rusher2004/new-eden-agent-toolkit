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

func getCharacterFleetHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[GetCharacterFleetInput, GetCharacterFleetOutput] {
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

func getFleetHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[GetFleetInput, GetFleetOutput] {
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

func updateFleetHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[UpdateFleetInput, UpdateFleetOutput] {
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
	CharacterID     int64     `json:"characterId"`
	JoinTime        time.Time `json:"joinTime"`
	Role            string    `json:"role"`
	RoleName        string    `json:"roleName"`
	ShipTypeID      int64     `json:"shipTypeId"`
	ShipTypeName    string    `json:"shipTypeName,omitempty"`
	SolarSystemID   int64     `json:"solarSystemId"`
	SolarSystemName string    `json:"solarSystemName,omitempty"`
	SquadID         int64     `json:"squadId"`
	StationID       *int64    `json:"stationId,omitempty"`
	StationName     string    `json:"stationName,omitempty"`
	TakesFleetWarp  bool      `json:"takesFleetWarp"`
	WingID          int64     `json:"wingId"`
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

func listFleetMembersHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListFleetMembersInput, ListFleetMembersOutput] {
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
		ids := make([]int64, len(members))
		for i, m := range members {
			ids[i] = m.ShipTypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range members {
			members[i].ShipTypeName = names[members[i].ShipTypeID]
		}
		systemIDs := make([]int64, len(members))
		var stationIDs []int64
		for i, m := range members {
			systemIDs[i] = m.SolarSystemID
			if m.StationID != nil {
				stationIDs = append(stationIDs, *m.StationID)
			}
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		stationNames := resolveStationNames(ctx, sdeStore, sso.UserAgent(version), stationIDs)
		for i := range members {
			members[i].SolarSystemName = systemNames[members[i].SolarSystemID]
			if members[i].StationID != nil {
				members[i].StationName = stationNames[*members[i].StationID]
			}
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
func inviteFleetMemberHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[InviteFleetMemberInput, InviteFleetMemberOutput] {
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

func moveFleetMemberHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[MoveFleetMemberInput, MoveFleetMemberOutput] {
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

func kickFleetMemberHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[KickFleetMemberInput, KickFleetMemberOutput] {
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

func listFleetWingsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListFleetWingsInput, ListFleetWingsOutput] {
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

func createFleetWingHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[CreateFleetWingInput, CreateFleetWingOutput] {
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

func renameFleetWingHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[RenameFleetWingInput, RenameFleetWingOutput] {
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

func deleteFleetWingHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[DeleteFleetWingInput, DeleteFleetWingOutput] {
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

func createFleetSquadHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[CreateFleetSquadInput, CreateFleetSquadOutput] {
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

func renameFleetSquadHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[RenameFleetSquadInput, RenameFleetSquadOutput] {
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

func deleteFleetSquadHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[DeleteFleetSquadInput, DeleteFleetSquadOutput] {
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
