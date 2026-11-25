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

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// fleetsESI is the ESI calls the Fleets tools make.
type fleetsESI interface {
	CharacterFleet(ctx context.Context) (*esiapi.CharactersCharacterIdFleetGet, error)
	FleetInfo(ctx context.Context, fleetID int64) (*esiapi.FleetsFleetIdGet, error)
	UpdateFleet(ctx context.Context, fleetID int64, isFreeMove *bool, motd *string) error
	FleetMembers(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error)
	InviteFleetMember(ctx context.Context, fleetID, characterID int64, role string, wingID, squadID *int64) error
	MoveFleetMember(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error
	KickFleetMember(ctx context.Context, fleetID, memberID int64) error
	FleetWings(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdWingsGetInner, error)
	CreateFleetWing(ctx context.Context, fleetID int64) (int64, error)
	RenameFleetWing(ctx context.Context, fleetID, wingID int64, name string) error
	DeleteFleetWing(ctx context.Context, fleetID, wingID int64) error
	CreateFleetSquad(ctx context.Context, fleetID, wingID int64) (int64, error)
	RenameFleetSquad(ctx context.Context, fleetID, squadID int64, name string) error
	DeleteFleetSquad(ctx context.Context, fleetID, squadID int64) error
}

// fleetMemberNames is the SDE lookups list_fleet_members enriches with.
type fleetMemberNames interface {
	typeNameCache
	systemNameCache
	stationNameCache
}

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

func getCharacterFleet(ctx context.Context, client fleetsESI, characterID int32) (GetCharacterFleetOutput, error) {
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

// AddGetCharacterFleet registers get_character_fleet. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddGetCharacterFleet[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_fleet",
		Description: "Get a logged-in EVE Online character's current fleet membership.",
	}, getCharacterFleetHandler(open, store))
}

func getCharacterFleetHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterFleetInput, GetCharacterFleetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFleetInput) (*mcpsdk.CallToolResult, GetCharacterFleetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFleetOutput{}, err
		}
		if !found {
			return nil, GetCharacterFleetOutput{}, fmt.Errorf("get character fleet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFleetOutput{}, fmt.Errorf("get character fleet: %w", err)
		}
		out, err := getCharacterFleet(ctx, client, in.CharacterID)
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

func getFleet(ctx context.Context, client fleetsESI, characterID int32, fleetID int64) (GetFleetOutput, error) {
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

// AddGetFleet registers get_fleet. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddGetFleet[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_fleet",
		Description: "Get a fleet's settings, using a logged-in character's fleet membership to authorize.",
	}, getFleetHandler(open, store))
}

func getFleetHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetFleetInput, GetFleetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFleetInput) (*mcpsdk.CallToolResult, GetFleetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetFleetOutput{}, err
		}
		if !found {
			return nil, GetFleetOutput{}, fmt.Errorf("get fleet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetFleetOutput{}, fmt.Errorf("get fleet: %w", err)
		}
		out, err := getFleet(ctx, client, in.CharacterID, in.FleetID)
		if err != nil {
			return nil, GetFleetOutput{}, fmt.Errorf("get fleet: %w", err)
		}
		return nil, out, nil
	}
}

// --- update_fleet (private, write) ---

func updateFleet(ctx context.Context, client fleetsESI, fleetID int64, isFreeMove *bool, motd *string) error {
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

// AddUpdateFleet registers update_fleet. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddUpdateFleet[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "update_fleet",
		Description: "Update a fleet's settings — this changes state on EVE's server.",
	}, updateFleetHandler(open, store))
}

func updateFleetHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[UpdateFleetInput, UpdateFleetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateFleetInput) (*mcpsdk.CallToolResult, UpdateFleetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, UpdateFleetOutput{}, err
		}
		if !found {
			return nil, UpdateFleetOutput{}, fmt.Errorf("update fleet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, UpdateFleetOutput{}, fmt.Errorf("update fleet: %w", err)
		}
		if err := updateFleet(ctx, client, in.FleetID, in.IsFreeMove, in.Motd); err != nil {
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

func listFleetMembers(ctx context.Context, client fleetsESI, fleetID int64) ([]FleetMember, error) {
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

// AddListFleetMembers registers list_fleet_members. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddListFleetMembers[C fleetsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore fleetMemberNames) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_fleet_members",
		Description: "List a fleet's members, using a logged-in character's fleet membership to authorize.",
	}, listFleetMembersHandler(userAgent, open, store, sdeStore))
}

func listFleetMembersHandler[C fleetsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore fleetMemberNames) mcpsdk.ToolHandlerFor[ListFleetMembersInput, ListFleetMembersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFleetMembersInput) (*mcpsdk.CallToolResult, ListFleetMembersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListFleetMembersOutput{}, err
		}
		if !found {
			return nil, ListFleetMembersOutput{}, fmt.Errorf("list fleet members for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListFleetMembersOutput{}, fmt.Errorf("list fleet members: %w", err)
		}
		members, err := listFleetMembers(ctx, client, in.FleetID)
		if err != nil {
			return nil, ListFleetMembersOutput{}, fmt.Errorf("list fleet members: %w", err)
		}
		ids := make([]int64, len(members))
		for i, m := range members {
			ids[i] = m.ShipTypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
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
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		stationNames := resolveStationNames(ctx, sdeStore, userAgent, stationIDs)
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

func inviteFleetMember(ctx context.Context, client fleetsESI, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
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

// AddInviteFleetMember registers invite_fleet_member. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddInviteFleetMember[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "invite_fleet_member",
		Description: "Invite a character into a fleet — this changes state on EVE's server.",
	}, inviteFleetMemberHandler(open, store))
}

// inviteFleetMemberHandler is a write tool: it changes state on FC's server, so -- like
// respond_calendar_event -- it validates its input BEFORE loading config.Store or calling ESI.
func inviteFleetMemberHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[InviteFleetMemberInput, InviteFleetMemberOutput] {
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
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, InviteFleetMemberOutput{}, fmt.Errorf("invite fleet member: %w", err)
		}
		if err := inviteFleetMember(ctx, client, in.FleetID, in.InviteCharacterID, in.Role, in.WingID, in.SquadID); err != nil {
			return nil, InviteFleetMemberOutput{}, fmt.Errorf("invite fleet member: %w", err)
		}
		return nil, InviteFleetMemberOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, InviteCharacterID: in.InviteCharacterID, Role: in.Role}, nil
	}
}

// --- move_fleet_member (private, write) ---

func moveFleetMember(ctx context.Context, client fleetsESI, fleetID, memberID int64, role string, wingID, squadID *int64) error {
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

// AddMoveFleetMember registers move_fleet_member. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddMoveFleetMember[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "move_fleet_member",
		Description: "Move a fleet member to a new role, wing, or squad — this changes state on EVE's server.",
	}, moveFleetMemberHandler(open, store))
}

func moveFleetMemberHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[MoveFleetMemberInput, MoveFleetMemberOutput] {
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
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, MoveFleetMemberOutput{}, fmt.Errorf("move fleet member: %w", err)
		}
		if err := moveFleetMember(ctx, client, in.FleetID, in.MemberID, in.Role, in.WingID, in.SquadID); err != nil {
			return nil, MoveFleetMemberOutput{}, fmt.Errorf("move fleet member: %w", err)
		}
		return nil, MoveFleetMemberOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, MemberID: in.MemberID, Role: in.Role}, nil
	}
}

// --- kick_fleet_member (private, write) ---

func kickFleetMember(ctx context.Context, client fleetsESI, fleetID, memberID int64) error {
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

// AddKickFleetMember registers kick_fleet_member. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddKickFleetMember[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "kick_fleet_member",
		Description: "Remove a member from a fleet — this changes state on EVE's server.",
	}, kickFleetMemberHandler(open, store))
}

func kickFleetMemberHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[KickFleetMemberInput, KickFleetMemberOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in KickFleetMemberInput) (*mcpsdk.CallToolResult, KickFleetMemberOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, KickFleetMemberOutput{}, err
		}
		if !found {
			return nil, KickFleetMemberOutput{}, fmt.Errorf("kick fleet member for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, KickFleetMemberOutput{}, fmt.Errorf("kick fleet member: %w", err)
		}
		if err := kickFleetMember(ctx, client, in.FleetID, in.MemberID); err != nil {
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

func listFleetWings(ctx context.Context, client fleetsESI, fleetID int64) ([]FleetWing, error) {
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

// AddListFleetWings registers list_fleet_wings. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddListFleetWings[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_fleet_wings",
		Description: "List a fleet's wings and their nested squads, using a logged-in character's fleet membership to authorize.",
	}, listFleetWingsHandler(open, store))
}

func listFleetWingsHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListFleetWingsInput, ListFleetWingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFleetWingsInput) (*mcpsdk.CallToolResult, ListFleetWingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListFleetWingsOutput{}, err
		}
		if !found {
			return nil, ListFleetWingsOutput{}, fmt.Errorf("list fleet wings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListFleetWingsOutput{}, fmt.Errorf("list fleet wings: %w", err)
		}
		wings, err := listFleetWings(ctx, client, in.FleetID)
		if err != nil {
			return nil, ListFleetWingsOutput{}, fmt.Errorf("list fleet wings: %w", err)
		}
		return nil, ListFleetWingsOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, Wings: wings}, nil
	}
}

// --- create_fleet_wing (private, write) ---

func createFleetWing(ctx context.Context, client fleetsESI, fleetID int64) (int64, error) {
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

// AddCreateFleetWing registers create_fleet_wing. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddCreateFleetWing[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_fleet_wing",
		Description: "Create a new wing in a fleet — this changes state on EVE's server.",
	}, createFleetWingHandler(open, store))
}

func createFleetWingHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[CreateFleetWingInput, CreateFleetWingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateFleetWingInput) (*mcpsdk.CallToolResult, CreateFleetWingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateFleetWingOutput{}, err
		}
		if !found {
			return nil, CreateFleetWingOutput{}, fmt.Errorf("create fleet wing for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, CreateFleetWingOutput{}, fmt.Errorf("create fleet wing: %w", err)
		}
		wingID, err := createFleetWing(ctx, client, in.FleetID)
		if err != nil {
			return nil, CreateFleetWingOutput{}, fmt.Errorf("create fleet wing: %w", err)
		}
		return nil, CreateFleetWingOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: wingID}, nil
	}
}

// --- rename_fleet_wing (private, write) ---

func renameFleetWing(ctx context.Context, client fleetsESI, fleetID, wingID int64, name string) error {
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

// AddRenameFleetWing registers rename_fleet_wing. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddRenameFleetWing[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "rename_fleet_wing",
		Description: "Rename a fleet wing — this changes state on EVE's server.",
	}, renameFleetWingHandler(open, store))
}

func renameFleetWingHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[RenameFleetWingInput, RenameFleetWingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RenameFleetWingInput) (*mcpsdk.CallToolResult, RenameFleetWingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, RenameFleetWingOutput{}, err
		}
		if !found {
			return nil, RenameFleetWingOutput{}, fmt.Errorf("rename fleet wing for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, RenameFleetWingOutput{}, fmt.Errorf("rename fleet wing: %w", err)
		}
		if err := renameFleetWing(ctx, client, in.FleetID, in.WingID, in.Name); err != nil {
			return nil, RenameFleetWingOutput{}, fmt.Errorf("rename fleet wing: %w", err)
		}
		return nil, RenameFleetWingOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: in.WingID, Name: in.Name}, nil
	}
}

// --- delete_fleet_wing (private, write) ---

func deleteFleetWing(ctx context.Context, client fleetsESI, fleetID, wingID int64) error {
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

// AddDeleteFleetWing registers delete_fleet_wing. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddDeleteFleetWing[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_fleet_wing",
		Description: "Delete a fleet wing — this changes state on EVE's server.",
	}, deleteFleetWingHandler(open, store))
}

func deleteFleetWingHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[DeleteFleetWingInput, DeleteFleetWingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteFleetWingInput) (*mcpsdk.CallToolResult, DeleteFleetWingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteFleetWingOutput{}, err
		}
		if !found {
			return nil, DeleteFleetWingOutput{}, fmt.Errorf("delete fleet wing for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, DeleteFleetWingOutput{}, fmt.Errorf("delete fleet wing: %w", err)
		}
		if err := deleteFleetWing(ctx, client, in.FleetID, in.WingID); err != nil {
			return nil, DeleteFleetWingOutput{}, fmt.Errorf("delete fleet wing: %w", err)
		}
		return nil, DeleteFleetWingOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: in.WingID}, nil
	}
}

// --- create_fleet_squad (private, write) ---

func createFleetSquad(ctx context.Context, client fleetsESI, fleetID, wingID int64) (int64, error) {
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

// AddCreateFleetSquad registers create_fleet_squad. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddCreateFleetSquad[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_fleet_squad",
		Description: "Create a new squad within a fleet wing — this changes state on EVE's server.",
	}, createFleetSquadHandler(open, store))
}

func createFleetSquadHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[CreateFleetSquadInput, CreateFleetSquadOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateFleetSquadInput) (*mcpsdk.CallToolResult, CreateFleetSquadOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateFleetSquadOutput{}, err
		}
		if !found {
			return nil, CreateFleetSquadOutput{}, fmt.Errorf("create fleet squad for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, CreateFleetSquadOutput{}, fmt.Errorf("create fleet squad: %w", err)
		}
		squadID, err := createFleetSquad(ctx, client, in.FleetID, in.WingID)
		if err != nil {
			return nil, CreateFleetSquadOutput{}, fmt.Errorf("create fleet squad: %w", err)
		}
		return nil, CreateFleetSquadOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, WingID: in.WingID, SquadID: squadID}, nil
	}
}

// --- rename_fleet_squad (private, write) ---

func renameFleetSquad(ctx context.Context, client fleetsESI, fleetID, squadID int64, name string) error {
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

// AddRenameFleetSquad registers rename_fleet_squad. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddRenameFleetSquad[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "rename_fleet_squad",
		Description: "Rename a fleet squad — this changes state on EVE's server.",
	}, renameFleetSquadHandler(open, store))
}

func renameFleetSquadHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[RenameFleetSquadInput, RenameFleetSquadOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RenameFleetSquadInput) (*mcpsdk.CallToolResult, RenameFleetSquadOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, RenameFleetSquadOutput{}, err
		}
		if !found {
			return nil, RenameFleetSquadOutput{}, fmt.Errorf("rename fleet squad for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, RenameFleetSquadOutput{}, fmt.Errorf("rename fleet squad: %w", err)
		}
		if err := renameFleetSquad(ctx, client, in.FleetID, in.SquadID, in.Name); err != nil {
			return nil, RenameFleetSquadOutput{}, fmt.Errorf("rename fleet squad: %w", err)
		}
		return nil, RenameFleetSquadOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, SquadID: in.SquadID, Name: in.Name}, nil
	}
}

// --- delete_fleet_squad (private, write) ---

func deleteFleetSquad(ctx context.Context, client fleetsESI, fleetID, squadID int64) error {
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

// AddDeleteFleetSquad registers delete_fleet_squad. open builds the character's ESI
// client; any client with the fleetsESI methods will do.
func AddDeleteFleetSquad[C fleetsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_fleet_squad",
		Description: "Delete a fleet squad — this changes state on EVE's server.",
	}, deleteFleetSquadHandler(open, store))
}

func deleteFleetSquadHandler[C fleetsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[DeleteFleetSquadInput, DeleteFleetSquadOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteFleetSquadInput) (*mcpsdk.CallToolResult, DeleteFleetSquadOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteFleetSquadOutput{}, err
		}
		if !found {
			return nil, DeleteFleetSquadOutput{}, fmt.Errorf("delete fleet squad for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, DeleteFleetSquadOutput{}, fmt.Errorf("delete fleet squad: %w", err)
		}
		if err := deleteFleetSquad(ctx, client, in.FleetID, in.SquadID); err != nil {
			return nil, DeleteFleetSquadOutput{}, fmt.Errorf("delete fleet squad: %w", err)
		}
		return nil, DeleteFleetSquadOutput{CharacterID: in.CharacterID, FleetID: in.FleetID, SquadID: in.SquadID}, nil
	}
}
