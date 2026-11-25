package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetCharacterFleetHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fleetsFake{}
	fake.characterFleet = func(ctx context.Context) (*esiapi.CharactersCharacterIdFleetGet, error) {
		return &esiapi.CharactersCharacterIdFleetGet{
			FleetId: 1000, FleetBossId: 100, Role: "fleet_commander", SquadId: 10, WingId: 20,
		}, nil
	}

	handler := getCharacterFleetHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.characterFleet = func(ctx context.Context) (*esiapi.CharactersCharacterIdFleetGet, error) {
		t.Fatal("getCharacterFleet should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterFleetHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.fleetInfo = func(ctx context.Context, fleetID int64) (*esiapi.FleetsFleetIdGet, error) {
		return &esiapi.FleetsFleetIdGet{
			IsFreeMove: true, IsRegistered: true, IsVoiceEnabled: true, Motd: "Form up on grid",
		}, nil
	}

	handler := getFleetHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.fleetInfo = func(ctx context.Context, fleetID int64) (*esiapi.FleetsFleetIdGet, error) {
		t.Fatal("getFleet should not be called for an unknown character")
		return nil, nil
	}

	handler := getFleetHandler(fake.open, store)
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

	fake := &fleetsFake{}

	var gotFleetID int64
	var gotIsFreeMove *bool
	var gotMotd *string
	fake.updateFleet = func(ctx context.Context, fleetID int64, isFreeMove *bool, motd *string) error {
		gotFleetID = fleetID
		gotIsFreeMove = isFreeMove
		gotMotd = motd
		return nil
	}

	isFreeMove := true
	motd := "New MOTD"
	input := UpdateFleetInput{CharacterID: 7, FleetID: 1000, IsFreeMove: &isFreeMove, Motd: &motd}

	handler := updateFleetHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.updateFleet = func(ctx context.Context, fleetID int64, isFreeMove *bool, motd *string) error {
		t.Fatal("updateFleet should not be called for an unknown character")
		return nil
	}

	handler := updateFleetHandler(fake.open, store)
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

	fake := &fleetsFake{}
	joinTime := time.Unix(1000, 0)
	stationID := int64(60003760)
	fake.fleetMembers = func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error) {
		return []esiapi.FleetsFleetIdMembersGetInner{{
			CharacterId: 100, JoinTime: joinTime, Role: "squad_member", RoleName: "Squad Member",
			ShipTypeId: 600, SolarSystemId: 30000142, SquadId: 10, StationId: &stationID,
			TakesFleetWarp: true, WingId: 20,
		}}, nil
	}

	handler := listFleetMembersHandler("v1", fake.open, store, nil)
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

	fake := &fleetsFake{}
	fake.fleetMembers = func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error) {
		t.Fatal("listFleetMembers should not be called for an unknown character")
		return nil, nil
	}

	handler := listFleetMembersHandler("v1", fake.open, store, nil)
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

	fake := &fleetsFake{}
	fake.fleetMembers = func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error) {
		return []esiapi.FleetsFleetIdMembersGetInner{{CharacterId: 2001, ShipTypeId: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listFleetMembersHandler("v1", fake.open, store, nil)
	_, out, err := handler(context.Background(), nil, ListFleetMembersInput{CharacterID: 1001, FleetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Members[0].ShipTypeName != "Rifter" {
		t.Errorf("ShipTypeName = %q, want %q", out.Members[0].ShipTypeName, "Rifter")
	}
}

func TestListFleetMembersHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fleetsFake{}
	stationID := int64(60003760)
	fake.fleetMembers = func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error) {
		return []esiapi.FleetsFleetIdMembersGetInner{
			{CharacterId: 2001, SolarSystemId: 30000142, StationId: &stationID},
			{CharacterId: 2002, SolarSystemId: 30000142, StationId: nil},
		}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := listFleetMembersHandler("v1", fake.open, store, nil)
	_, out, err := handler(context.Background(), nil, ListFleetMembersInput{CharacterID: 7, FleetID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Members[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Members[0].SolarSystemName, "Jita")
	}
	if out.Members[0].StationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("StationName = %q, want %q", out.Members[0].StationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
	if out.Members[1].StationName != "" {
		t.Errorf("StationName (nil StationID) = %q, want empty", out.Members[1].StationName)
	}
}

func TestInviteFleetMemberHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fleetsFake{}

	var gotFleetID, gotInviteCharacterID int64
	var gotRole string
	var gotWingID, gotSquadID *int64
	fake.inviteFleetMember = func(ctx context.Context, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
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

	handler := inviteFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.inviteFleetMember = func(ctx context.Context, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
		t.Fatal("inviteFleetMember should not be called for an unknown character")
		return nil
	}

	handler := inviteFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.inviteFleetMember = func(ctx context.Context, fleetID, inviteCharacterID int64, role string, wingID, squadID *int64) error {
		t.Fatal("inviteFleetMember should not be called for an invalid role")
		return nil
	}

	handler := inviteFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}

	var gotFleetID, gotMemberID int64
	var gotRole string
	var gotWingID, gotSquadID *int64
	fake.moveFleetMember = func(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error {
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

	handler := moveFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.moveFleetMember = func(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error {
		t.Fatal("moveFleetMember should not be called for an unknown character")
		return nil
	}

	handler := moveFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.moveFleetMember = func(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error {
		t.Fatal("moveFleetMember should not be called for an invalid role")
		return nil
	}

	handler := moveFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.kickFleetMember = func(ctx context.Context, fleetID, memberID int64) error {
		return nil
	}

	handler := kickFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.kickFleetMember = func(ctx context.Context, fleetID, memberID int64) error {
		t.Fatal("kickFleetMember should not be called for an unknown character")
		return nil
	}

	handler := kickFleetMemberHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.fleetWings = func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdWingsGetInner, error) {
		return []esiapi.FleetsFleetIdWingsGetInner{{
			Id: 20, Name: "Wing One",
			Squads: []esiapi.FleetsFleetIdWingsGetInnerSquadsInner{{Id: 10, Name: "Squad One"}, {Id: 11, Name: "Squad Two"}},
		}}, nil
	}

	handler := listFleetWingsHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.fleetWings = func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdWingsGetInner, error) {
		t.Fatal("listFleetWings should not be called for an unknown character")
		return nil, nil
	}

	handler := listFleetWingsHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.createFleetWing = func(ctx context.Context, fleetID int64) (int64, error) {
		return 20, nil
	}

	handler := createFleetWingHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.createFleetWing = func(ctx context.Context, fleetID int64) (int64, error) {
		t.Fatal("createFleetWing should not be called for an unknown character")
		return 0, nil
	}

	handler := createFleetWingHandler(fake.open, store)
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

	fake := &fleetsFake{}

	var gotFleetID, gotWingID int64
	var gotName string
	fake.renameFleetWing = func(ctx context.Context, fleetID, wingID int64, name string) error {
		gotFleetID = fleetID
		gotWingID = wingID
		gotName = name
		return nil
	}

	handler := renameFleetWingHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.renameFleetWing = func(ctx context.Context, fleetID, wingID int64, name string) error {
		t.Fatal("renameFleetWing should not be called for an unknown character")
		return nil
	}

	handler := renameFleetWingHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.deleteFleetWing = func(ctx context.Context, fleetID, wingID int64) error {
		return nil
	}

	handler := deleteFleetWingHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.deleteFleetWing = func(ctx context.Context, fleetID, wingID int64) error {
		t.Fatal("deleteFleetWing should not be called for an unknown character")
		return nil
	}

	handler := deleteFleetWingHandler(fake.open, store)
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

	fake := &fleetsFake{}

	var gotFleetID, gotWingID int64
	fake.createFleetSquad = func(ctx context.Context, fleetID, wingID int64) (int64, error) {
		gotFleetID = fleetID
		gotWingID = wingID
		return 10, nil
	}

	handler := createFleetSquadHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.createFleetSquad = func(ctx context.Context, fleetID, wingID int64) (int64, error) {
		t.Fatal("createFleetSquad should not be called for an unknown character")
		return 0, nil
	}

	handler := createFleetSquadHandler(fake.open, store)
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

	fake := &fleetsFake{}

	var gotFleetID, gotSquadID int64
	var gotName string
	fake.renameFleetSquad = func(ctx context.Context, fleetID, squadID int64, name string) error {
		gotFleetID = fleetID
		gotSquadID = squadID
		gotName = name
		return nil
	}

	handler := renameFleetSquadHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.renameFleetSquad = func(ctx context.Context, fleetID, squadID int64, name string) error {
		t.Fatal("renameFleetSquad should not be called for an unknown character")
		return nil
	}

	handler := renameFleetSquadHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.deleteFleetSquad = func(ctx context.Context, fleetID, squadID int64) error {
		return nil
	}

	handler := deleteFleetSquadHandler(fake.open, store)
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

	fake := &fleetsFake{}
	fake.deleteFleetSquad = func(ctx context.Context, fleetID, squadID int64) error {
		t.Fatal("deleteFleetSquad should not be called for an unknown character")
		return nil
	}

	handler := deleteFleetSquadHandler(fake.open, store)
	_, _, err := handler(context.Background(), nil, DeleteFleetSquadInput{CharacterID: 999, FleetID: 1000, SquadID: 10})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// TestFleetsHandlers_OpenFailure: every handler that opens an ESI client wraps and returns
// the opener's error, and never reaches an ESI call.
func TestFleetsHandlers_OpenFailure(t *testing.T) {
	store := newTestStoreWithCharacter(t)
	openErr := errors.New("keychain locked")
	fake := &fleetsFake{openErr: openErr}
	tests := []struct {
		name string
		call func() error
	}{
		{"get_character_fleet", func() error {
			return callHandlerForOpenFailure(getCharacterFleetHandler(fake.open, store), GetCharacterFleetInput{CharacterID: 7})
		}},
		{"get_fleet", func() error {
			return callHandlerForOpenFailure(getFleetHandler(fake.open, store), GetFleetInput{CharacterID: 7})
		}},
		{"update_fleet", func() error {
			return callHandlerForOpenFailure(updateFleetHandler(fake.open, store), UpdateFleetInput{CharacterID: 7})
		}},
		{"list_fleet_members", func() error {
			return callHandlerForOpenFailure(listFleetMembersHandler("v1", fake.open, store, nil), ListFleetMembersInput{CharacterID: 7})
		}},
		{"invite_fleet_member", func() error {
			return callHandlerForOpenFailure(inviteFleetMemberHandler(fake.open, store), InviteFleetMemberInput{CharacterID: 7, Role: "squad_member"})
		}},
		{"move_fleet_member", func() error {
			return callHandlerForOpenFailure(moveFleetMemberHandler(fake.open, store), MoveFleetMemberInput{CharacterID: 7, Role: "squad_member"})
		}},
		{"kick_fleet_member", func() error {
			return callHandlerForOpenFailure(kickFleetMemberHandler(fake.open, store), KickFleetMemberInput{CharacterID: 7})
		}},
		{"list_fleet_wings", func() error {
			return callHandlerForOpenFailure(listFleetWingsHandler(fake.open, store), ListFleetWingsInput{CharacterID: 7})
		}},
		{"create_fleet_wing", func() error {
			return callHandlerForOpenFailure(createFleetWingHandler(fake.open, store), CreateFleetWingInput{CharacterID: 7})
		}},
		{"rename_fleet_wing", func() error {
			return callHandlerForOpenFailure(renameFleetWingHandler(fake.open, store), RenameFleetWingInput{CharacterID: 7})
		}},
		{"delete_fleet_wing", func() error {
			return callHandlerForOpenFailure(deleteFleetWingHandler(fake.open, store), DeleteFleetWingInput{CharacterID: 7})
		}},
		{"create_fleet_squad", func() error {
			return callHandlerForOpenFailure(createFleetSquadHandler(fake.open, store), CreateFleetSquadInput{CharacterID: 7})
		}},
		{"rename_fleet_squad", func() error {
			return callHandlerForOpenFailure(renameFleetSquadHandler(fake.open, store), RenameFleetSquadInput{CharacterID: 7})
		}},
		{"delete_fleet_squad", func() error {
			return callHandlerForOpenFailure(deleteFleetSquadHandler(fake.open, store), DeleteFleetSquadInput{CharacterID: 7})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if !errors.Is(err, openErr) {
				t.Fatalf("err = %v, want wrapping %v", err, openErr)
			}
			if !strings.Contains(err.Error(), strings.ReplaceAll(tt.name, "_", " ")) {
				t.Errorf("err = %q, want it to start with the tool name %q", err, tt.name)
			}
		})
	}
}

// fleetsFake fakes fleetsESI: set a field to answer that call, and open records the character IDs it was asked for.
type fleetsFake struct {
	characterFleet    func(ctx context.Context) (*esiapi.CharactersCharacterIdFleetGet, error)
	fleetInfo         func(ctx context.Context, fleetID int64) (*esiapi.FleetsFleetIdGet, error)
	updateFleet       func(ctx context.Context, fleetID int64, isFreeMove *bool, motd *string) error
	fleetMembers      func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error)
	inviteFleetMember func(ctx context.Context, fleetID, characterID int64, role string, wingID, squadID *int64) error
	moveFleetMember   func(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error
	kickFleetMember   func(ctx context.Context, fleetID, memberID int64) error
	fleetWings        func(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdWingsGetInner, error)
	createFleetWing   func(ctx context.Context, fleetID int64) (int64, error)
	renameFleetWing   func(ctx context.Context, fleetID, wingID int64, name string) error
	deleteFleetWing   func(ctx context.Context, fleetID, wingID int64) error
	createFleetSquad  func(ctx context.Context, fleetID, wingID int64) (int64, error)
	renameFleetSquad  func(ctx context.Context, fleetID, squadID int64, name string) error
	deleteFleetSquad  func(ctx context.Context, fleetID, squadID int64) error
	openErr           error
	openedFor         []int32
}

func (f *fleetsFake) open(ctx context.Context, characterID int32) (fleetsESI, error) {
	f.openedFor = append(f.openedFor, characterID)
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f, nil
}

func (f *fleetsFake) CharacterFleet(ctx context.Context) (*esiapi.CharactersCharacterIdFleetGet, error) {
	if f.characterFleet == nil {
		return nil, errors.New("fleetsFake: unexpected CharacterFleet call")
	}
	return f.characterFleet(ctx)
}

func (f *fleetsFake) FleetInfo(ctx context.Context, fleetID int64) (*esiapi.FleetsFleetIdGet, error) {
	if f.fleetInfo == nil {
		return nil, errors.New("fleetsFake: unexpected FleetInfo call")
	}
	return f.fleetInfo(ctx, fleetID)
}

func (f *fleetsFake) UpdateFleet(ctx context.Context, fleetID int64, isFreeMove *bool, motd *string) error {
	if f.updateFleet == nil {
		return errors.New("fleetsFake: unexpected UpdateFleet call")
	}
	return f.updateFleet(ctx, fleetID, isFreeMove, motd)
}

func (f *fleetsFake) FleetMembers(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdMembersGetInner, error) {
	if f.fleetMembers == nil {
		return nil, errors.New("fleetsFake: unexpected FleetMembers call")
	}
	return f.fleetMembers(ctx, fleetID)
}

func (f *fleetsFake) InviteFleetMember(ctx context.Context, fleetID, characterID int64, role string, wingID, squadID *int64) error {
	if f.inviteFleetMember == nil {
		return errors.New("fleetsFake: unexpected InviteFleetMember call")
	}
	return f.inviteFleetMember(ctx, fleetID, characterID, role, wingID, squadID)
}

func (f *fleetsFake) MoveFleetMember(ctx context.Context, fleetID, memberID int64, role string, wingID, squadID *int64) error {
	if f.moveFleetMember == nil {
		return errors.New("fleetsFake: unexpected MoveFleetMember call")
	}
	return f.moveFleetMember(ctx, fleetID, memberID, role, wingID, squadID)
}

func (f *fleetsFake) KickFleetMember(ctx context.Context, fleetID, memberID int64) error {
	if f.kickFleetMember == nil {
		return errors.New("fleetsFake: unexpected KickFleetMember call")
	}
	return f.kickFleetMember(ctx, fleetID, memberID)
}

func (f *fleetsFake) FleetWings(ctx context.Context, fleetID int64) ([]esiapi.FleetsFleetIdWingsGetInner, error) {
	if f.fleetWings == nil {
		return nil, errors.New("fleetsFake: unexpected FleetWings call")
	}
	return f.fleetWings(ctx, fleetID)
}

func (f *fleetsFake) CreateFleetWing(ctx context.Context, fleetID int64) (int64, error) {
	if f.createFleetWing == nil {
		return 0, errors.New("fleetsFake: unexpected CreateFleetWing call")
	}
	return f.createFleetWing(ctx, fleetID)
}

func (f *fleetsFake) RenameFleetWing(ctx context.Context, fleetID, wingID int64, name string) error {
	if f.renameFleetWing == nil {
		return errors.New("fleetsFake: unexpected RenameFleetWing call")
	}
	return f.renameFleetWing(ctx, fleetID, wingID, name)
}

func (f *fleetsFake) DeleteFleetWing(ctx context.Context, fleetID, wingID int64) error {
	if f.deleteFleetWing == nil {
		return errors.New("fleetsFake: unexpected DeleteFleetWing call")
	}
	return f.deleteFleetWing(ctx, fleetID, wingID)
}

func (f *fleetsFake) CreateFleetSquad(ctx context.Context, fleetID, wingID int64) (int64, error) {
	if f.createFleetSquad == nil {
		return 0, errors.New("fleetsFake: unexpected CreateFleetSquad call")
	}
	return f.createFleetSquad(ctx, fleetID, wingID)
}

func (f *fleetsFake) RenameFleetSquad(ctx context.Context, fleetID, squadID int64, name string) error {
	if f.renameFleetSquad == nil {
		return errors.New("fleetsFake: unexpected RenameFleetSquad call")
	}
	return f.renameFleetSquad(ctx, fleetID, squadID, name)
}

func (f *fleetsFake) DeleteFleetSquad(ctx context.Context, fleetID, squadID int64) error {
	if f.deleteFleetSquad == nil {
		return errors.New("fleetsFake: unexpected DeleteFleetSquad call")
	}
	return f.deleteFleetSquad(ctx, fleetID, squadID)
}

// callHandlerForOpenFailure runs a tool handler for a known character and returns its error.
func callHandlerForOpenFailure[In, Out any](h mcpsdk.ToolHandlerFor[In, Out], in In) error {
	_, _, err := h(context.Background(), nil, in)
	return err
}
