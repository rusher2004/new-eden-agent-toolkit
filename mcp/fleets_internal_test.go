package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetCharacterFleetHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterFleet
	t.Cleanup(func() { getCharacterFleet = orig })
	getCharacterFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFleetOutput, error) {
		return GetCharacterFleetOutput{
			CharacterID: characterID, FleetID: 1000, FleetBossID: 100, Role: "fleet_commander", SquadID: 10, WingID: 20,
		}, nil
	}

	handler := getCharacterFleetHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterFleetInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.FleetBossID != 100 || out.Role != "fleet_commander" || out.SquadID != 10 || out.WingID != 20 {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 FleetBossID=100 Role=fleet_commander SquadID=10 WingID=20", out)
	}
}

func TestGetCharacterFleetHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterFleet
	t.Cleanup(func() { getCharacterFleet = orig })
	getCharacterFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFleetOutput, error) {
		t.Fatal("getCharacterFleet should not be called for an unknown character")
		return GetCharacterFleetOutput{}, nil
	}

	handler := getCharacterFleetHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterFleetInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetFleetHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getFleet
	t.Cleanup(func() { getFleet = orig })
	getFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) (GetFleetOutput, error) {
		return GetFleetOutput{
			CharacterID: characterID, FleetID: fleetID, IsFreeMove: true, IsRegistered: true, IsVoiceEnabled: true, Motd: "Form up on grid",
		}, nil
	}

	handler := getFleetHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetFleetInput{CharacterID: 7, FleetID: 1000})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || !out.IsFreeMove || !out.IsRegistered || !out.IsVoiceEnabled || out.Motd != "Form up on grid" {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 IsFreeMove=true IsRegistered=true IsVoiceEnabled=true Motd=%q", out, "Form up on grid")
	}
}

func TestGetFleetHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getFleet
	t.Cleanup(func() { getFleet = orig })
	getFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) (GetFleetOutput, error) {
		t.Fatal("getFleet should not be called for an unknown character")
		return GetFleetOutput{}, nil
	}

	handler := getFleetHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetFleetInput{CharacterID: 999, FleetID: 1000})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestUpdateFleetHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := updateFleet
	t.Cleanup(func() { updateFleet = orig })

	var gotFleetID int64
	var gotIsFreeMove *bool
	var gotMotd *string
	updateFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64, isFreeMove *bool, motd *string) error {
		gotFleetID = fleetID
		gotIsFreeMove = isFreeMove
		gotMotd = motd
		return nil
	}

	isFreeMove := true
	motd := "New MOTD"
	input := UpdateFleetInput{CharacterID: 7, FleetID: 1000, IsFreeMove: &isFreeMove, Motd: &motd}

	handler := updateFleetHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotFleetID != 1000 || gotIsFreeMove == nil || *gotIsFreeMove != true || gotMotd == nil || *gotMotd != "New MOTD" {
		t.Errorf("captured fleetID/isFreeMove/motd = %d/%v/%v, want 1000/true/%q", gotFleetID, gotIsFreeMove, gotMotd, "New MOTD")
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.IsFreeMove == nil || *out.IsFreeMove != true || out.Motd == nil || *out.Motd != "New MOTD" {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 IsFreeMove=true Motd=%q", out, "New MOTD")
	}
}

func TestUpdateFleetHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := updateFleet
	t.Cleanup(func() { updateFleet = orig })
	updateFleet = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64, isFreeMove *bool, motd *string) error {
		t.Fatal("updateFleet should not be called for an unknown character")
		return nil
	}

	handler := updateFleetHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, UpdateFleetInput{CharacterID: 999, FleetID: 1000})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListFleetMembersHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listFleetMembers
	t.Cleanup(func() { listFleetMembers = orig })
	joinTime := time.Unix(1000, 0)
	stationID := int64(60003760)
	listFleetMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetMember, error) {
		return []FleetMember{{
			CharacterID: 100, JoinTime: joinTime, Role: "squad_member", RoleName: "Squad Member",
			ShipTypeID: 600, SolarSystemID: 30000142, SquadID: 10, StationID: &stationID,
			TakesFleetWarp: true, WingID: 20,
		}}, nil
	}

	handler := listFleetMembersHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListFleetMembersInput{CharacterID: 7, FleetID: 1000})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || len(out.Members) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 FleetID=1000 with one member", out)
	}
	m := out.Members[0]
	if m.CharacterID != 100 || !m.JoinTime.Equal(joinTime) || m.Role != "squad_member" || m.RoleName != "Squad Member" ||
		m.ShipTypeID != 600 || m.SolarSystemID != 30000142 || m.SquadID != 10 ||
		m.StationID == nil || *m.StationID != stationID || !m.TakesFleetWarp || m.WingID != 20 {
		t.Errorf("member = %+v, want CharacterID=100 JoinTime=%v Role=squad_member RoleName=%q ShipTypeID=600 SolarSystemID=30000142 SquadID=10 StationID=%d TakesFleetWarp=true WingID=20",
			m, joinTime, "Squad Member", stationID)
	}
}

func TestListFleetMembersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listFleetMembers
	t.Cleanup(func() { listFleetMembers = orig })
	listFleetMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetMember, error) {
		t.Fatal("listFleetMembers should not be called for an unknown character")
		return nil, nil
	}

	handler := listFleetMembersHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListFleetMembersInput{CharacterID: 999, FleetID: 1000})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListFleetMembersHandler_ResolvesShipTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 1001, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listFleetMembers
	t.Cleanup(func() { listFleetMembers = orig })
	listFleetMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetMember, error) {
		return []FleetMember{{CharacterID: 2001, ShipTypeID: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listFleetMembersHandler("v1", "client", store, nil)
	_, out, err := handler(context.Background(), nil, ListFleetMembersInput{CharacterID: 1001, FleetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Members[0].ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.Members[0].ShipTypeName, "Rifter")
	}
}

func TestInviteFleetMemberHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := inviteFleetMember
	t.Cleanup(func() { inviteFleetMember = orig })

	var gotFleetID, gotInviteCharacterID int64
	var gotRole string
	var gotWingID, gotSquadID *int64
	inviteFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
		gotFleetID = fleetID
		gotInviteCharacterID = inviteCharacterID
		gotRole = role
		gotWingID = wingID
		gotSquadID = squadID
		return nil
	}

	wingID := int64(20)
	squadID := int64(10)
	input := InviteFleetMemberInput{
		CharacterID: 7, FleetID: 1000, InviteCharacterID: 200, Role: "squad_member", WingID: &wingID, SquadID: &squadID,
	}

	handler := inviteFleetMemberHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotFleetID != 1000 || gotInviteCharacterID != 200 || gotRole != "squad_member" ||
		gotWingID == nil || *gotWingID != 20 || gotSquadID == nil || *gotSquadID != 10 {
		t.Errorf("captured fleetID/inviteCharacterID/role/wingID/squadID = %d/%d/%s/%v/%v, want 1000/200/squad_member/20/10",
			gotFleetID, gotInviteCharacterID, gotRole, gotWingID, gotSquadID)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.InviteCharacterID != 200 || out.Role != "squad_member" {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 InviteCharacterID=200 Role=squad_member", out)
	}
}

func TestInviteFleetMemberHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := inviteFleetMember
	t.Cleanup(func() { inviteFleetMember = orig })
	inviteFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
		t.Fatal("inviteFleetMember should not be called for an unknown character")
		return nil
	}

	handler := inviteFleetMemberHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, InviteFleetMemberInput{CharacterID: 999, FleetID: 1000, InviteCharacterID: 200, Role: "squad_member"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestInviteFleetMemberHandler_InvalidRole(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := inviteFleetMember
	t.Cleanup(func() { inviteFleetMember = orig })
	inviteFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
		t.Fatal("inviteFleetMember should not be called for an invalid role")
		return nil
	}

	handler := inviteFleetMemberHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, InviteFleetMemberInput{CharacterID: 7, FleetID: 1000, InviteCharacterID: 200, Role: "bogus_role"})
	if err == nil {
		t.Fatal("expected an error for invalid role, got nil")
	}
}

func TestMoveFleetMemberHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := moveFleetMember
	t.Cleanup(func() { moveFleetMember = orig })

	var gotFleetID, gotMemberID int64
	var gotRole string
	var gotWingID, gotSquadID *int64
	moveFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, memberID int64, role string, wingID, squadID *int64) error {
		gotFleetID = fleetID
		gotMemberID = memberID
		gotRole = role
		gotWingID = wingID
		gotSquadID = squadID
		return nil
	}

	wingID := int64(20)
	squadID := int64(10)
	input := MoveFleetMemberInput{
		CharacterID: 7, FleetID: 1000, MemberID: 200, Role: "squad_commander", WingID: &wingID, SquadID: &squadID,
	}

	handler := moveFleetMemberHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotFleetID != 1000 || gotMemberID != 200 || gotRole != "squad_commander" ||
		gotWingID == nil || *gotWingID != 20 || gotSquadID == nil || *gotSquadID != 10 {
		t.Errorf("captured fleetID/memberID/role/wingID/squadID = %d/%d/%s/%v/%v, want 1000/200/squad_commander/20/10",
			gotFleetID, gotMemberID, gotRole, gotWingID, gotSquadID)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.MemberID != 200 || out.Role != "squad_commander" {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 MemberID=200 Role=squad_commander", out)
	}
}

func TestMoveFleetMemberHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := moveFleetMember
	t.Cleanup(func() { moveFleetMember = orig })
	moveFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, memberID int64, role string, wingID, squadID *int64) error {
		t.Fatal("moveFleetMember should not be called for an unknown character")
		return nil
	}

	handler := moveFleetMemberHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, MoveFleetMemberInput{CharacterID: 999, FleetID: 1000, MemberID: 200, Role: "squad_member"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestMoveFleetMemberHandler_InvalidRole(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := moveFleetMember
	t.Cleanup(func() { moveFleetMember = orig })
	moveFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, memberID int64, role string, wingID, squadID *int64) error {
		t.Fatal("moveFleetMember should not be called for an invalid role")
		return nil
	}

	handler := moveFleetMemberHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, MoveFleetMemberInput{CharacterID: 7, FleetID: 1000, MemberID: 200, Role: "bogus_role"})
	if err == nil {
		t.Fatal("expected an error for invalid role, got nil")
	}
}

func TestKickFleetMemberHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := kickFleetMember
	t.Cleanup(func() { kickFleetMember = orig })
	kickFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, memberID int64) error {
		return nil
	}

	handler := kickFleetMemberHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, KickFleetMemberInput{CharacterID: 7, FleetID: 1000, MemberID: 200})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.MemberID != 200 {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 MemberID=200", out)
	}
}

func TestKickFleetMemberHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := kickFleetMember
	t.Cleanup(func() { kickFleetMember = orig })
	kickFleetMember = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, memberID int64) error {
		t.Fatal("kickFleetMember should not be called for an unknown character")
		return nil
	}

	handler := kickFleetMemberHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, KickFleetMemberInput{CharacterID: 999, FleetID: 1000, MemberID: 200})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListFleetWingsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listFleetWings
	t.Cleanup(func() { listFleetWings = orig })
	listFleetWings = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetWing, error) {
		return []FleetWing{{
			WingID: 20, Name: "Wing One",
			Squads: []FleetSquad{{SquadID: 10, Name: "Squad One"}, {SquadID: 11, Name: "Squad Two"}},
		}}, nil
	}

	handler := listFleetWingsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListFleetWingsInput{CharacterID: 7, FleetID: 1000})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || len(out.Wings) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 FleetID=1000 with one wing", out)
	}
	w := out.Wings[0]
	if w.WingID != 20 || w.Name != "Wing One" || len(w.Squads) != 2 {
		t.Fatalf("wing = %+v, want WingID=20 Name=%q with two squads", w, "Wing One")
	}
	if w.Squads[0].SquadID != 10 || w.Squads[0].Name != "Squad One" {
		t.Errorf("squad[0] = %+v, want SquadID=10 Name=%q", w.Squads[0], "Squad One")
	}
	if w.Squads[1].SquadID != 11 || w.Squads[1].Name != "Squad Two" {
		t.Errorf("squad[1] = %+v, want SquadID=11 Name=%q", w.Squads[1], "Squad Two")
	}
}

func TestListFleetWingsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listFleetWings
	t.Cleanup(func() { listFleetWings = orig })
	listFleetWings = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) ([]FleetWing, error) {
		t.Fatal("listFleetWings should not be called for an unknown character")
		return nil, nil
	}

	handler := listFleetWingsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListFleetWingsInput{CharacterID: 999, FleetID: 1000})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestCreateFleetWingHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := createFleetWing
	t.Cleanup(func() { createFleetWing = orig })
	createFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) (int64, error) {
		return 20, nil
	}

	handler := createFleetWingHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, CreateFleetWingInput{CharacterID: 7, FleetID: 1000})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.WingID != 20 {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 WingID=20", out)
	}
}

func TestCreateFleetWingHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := createFleetWing
	t.Cleanup(func() { createFleetWing = orig })
	createFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID int64) (int64, error) {
		t.Fatal("createFleetWing should not be called for an unknown character")
		return 0, nil
	}

	handler := createFleetWingHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, CreateFleetWingInput{CharacterID: 999, FleetID: 1000})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRenameFleetWingHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := renameFleetWing
	t.Cleanup(func() { renameFleetWing = orig })

	var gotFleetID, gotWingID int64
	var gotName string
	renameFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64, name string) error {
		gotFleetID = fleetID
		gotWingID = wingID
		gotName = name
		return nil
	}

	handler := renameFleetWingHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, RenameFleetWingInput{CharacterID: 7, FleetID: 1000, WingID: 20, Name: "Interceptor Wing"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotFleetID != 1000 || gotWingID != 20 || gotName != "Interceptor Wing" {
		t.Errorf("captured fleetID/wingID/name = %d/%d/%q, want 1000/20/%q", gotFleetID, gotWingID, gotName, "Interceptor Wing")
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.WingID != 20 || out.Name != "Interceptor Wing" {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 WingID=20 Name=%q", out, "Interceptor Wing")
	}
}

func TestRenameFleetWingHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := renameFleetWing
	t.Cleanup(func() { renameFleetWing = orig })
	renameFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64, name string) error {
		t.Fatal("renameFleetWing should not be called for an unknown character")
		return nil
	}

	handler := renameFleetWingHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, RenameFleetWingInput{CharacterID: 999, FleetID: 1000, WingID: 20, Name: "Interceptor Wing"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestDeleteFleetWingHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := deleteFleetWing
	t.Cleanup(func() { deleteFleetWing = orig })
	deleteFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64) error {
		return nil
	}

	handler := deleteFleetWingHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, DeleteFleetWingInput{CharacterID: 7, FleetID: 1000, WingID: 20})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.WingID != 20 {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 WingID=20", out)
	}
}

func TestDeleteFleetWingHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := deleteFleetWing
	t.Cleanup(func() { deleteFleetWing = orig })
	deleteFleetWing = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64) error {
		t.Fatal("deleteFleetWing should not be called for an unknown character")
		return nil
	}

	handler := deleteFleetWingHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, DeleteFleetWingInput{CharacterID: 999, FleetID: 1000, WingID: 20})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestCreateFleetSquadHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := createFleetSquad
	t.Cleanup(func() { createFleetSquad = orig })

	var gotFleetID, gotWingID int64
	createFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64) (int64, error) {
		gotFleetID = fleetID
		gotWingID = wingID
		return 10, nil
	}

	handler := createFleetSquadHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, CreateFleetSquadInput{CharacterID: 7, FleetID: 1000, WingID: 20})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotFleetID != 1000 || gotWingID != 20 {
		t.Errorf("captured fleetID/wingID = %d/%d, want 1000/20", gotFleetID, gotWingID)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.WingID != 20 || out.SquadID != 10 {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 WingID=20 SquadID=10", out)
	}
}

func TestCreateFleetSquadHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := createFleetSquad
	t.Cleanup(func() { createFleetSquad = orig })
	createFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, wingID int64) (int64, error) {
		t.Fatal("createFleetSquad should not be called for an unknown character")
		return 0, nil
	}

	handler := createFleetSquadHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, CreateFleetSquadInput{CharacterID: 999, FleetID: 1000, WingID: 20})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRenameFleetSquadHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := renameFleetSquad
	t.Cleanup(func() { renameFleetSquad = orig })

	var gotFleetID, gotSquadID int64
	var gotName string
	renameFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, squadID int64, name string) error {
		gotFleetID = fleetID
		gotSquadID = squadID
		gotName = name
		return nil
	}

	handler := renameFleetSquadHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, RenameFleetSquadInput{CharacterID: 7, FleetID: 1000, SquadID: 10, Name: "Alpha Squad"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotFleetID != 1000 || gotSquadID != 10 || gotName != "Alpha Squad" {
		t.Errorf("captured fleetID/squadID/name = %d/%d/%q, want 1000/10/%q", gotFleetID, gotSquadID, gotName, "Alpha Squad")
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.SquadID != 10 || out.Name != "Alpha Squad" {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 SquadID=10 Name=%q", out, "Alpha Squad")
	}
}

func TestRenameFleetSquadHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := renameFleetSquad
	t.Cleanup(func() { renameFleetSquad = orig })
	renameFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, squadID int64, name string) error {
		t.Fatal("renameFleetSquad should not be called for an unknown character")
		return nil
	}

	handler := renameFleetSquadHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, RenameFleetSquadInput{CharacterID: 999, FleetID: 1000, SquadID: 10, Name: "Alpha Squad"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestDeleteFleetSquadHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := deleteFleetSquad
	t.Cleanup(func() { deleteFleetSquad = orig })
	deleteFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, squadID int64) error {
		return nil
	}

	handler := deleteFleetSquadHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, DeleteFleetSquadInput{CharacterID: 7, FleetID: 1000, SquadID: 10})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.FleetID != 1000 || out.SquadID != 10 {
		t.Errorf("out = %+v, want CharacterID=7 FleetID=1000 SquadID=10", out)
	}
}

func TestDeleteFleetSquadHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := deleteFleetSquad
	t.Cleanup(func() { deleteFleetSquad = orig })
	deleteFleetSquad = func(ctx context.Context, clientID string, characterID int32, userAgent string, fleetID, squadID int64) error {
		t.Fatal("deleteFleetSquad should not be called for an unknown character")
		return nil
	}

	handler := deleteFleetSquadHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, DeleteFleetSquadInput{CharacterID: 999, FleetID: 1000, SquadID: 10})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
