package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

//go:fix inline
func ptrInt64(v int64) *int64 { return new(v) }

func TestGetCorporationHandler_MapsFields(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{CorporationID: corporationID, Name: "Test Corp", Ticker: "TEST"}, nil
	}

	handler := getCorporationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.Name != "Test Corp" || out.Ticker != "TEST" {
		t.Errorf("out = %+v, want CorporationID=98000001 Name=Test Corp Ticker=TEST", out)
	}
}

func TestGetCorporationHandler_ResolvesFactionName(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	factionID := int64(500001)
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{CorporationID: 98000001, FactionID: &factionID, HomeStationID: 60003760}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}
	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{}
	}

	handler := getCorporationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetCorporationHandler_FactionNameEmptyWhenFactionIDNil(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{CorporationID: 98000001, HomeStationID: 60003760}, nil
	}

	handler := getCorporationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "" {
		t.Errorf("FactionName = %q, want empty when FactionID is nil", out.FactionName)
	}
}

func TestListCorporationAllianceHistoryHandler_MapsFields(t *testing.T) {
	orig := listCorporationAllianceHistory
	t.Cleanup(func() { listCorporationAllianceHistory = orig })
	allianceID := int64(99000001)
	listCorporationAllianceHistory = func(ctx context.Context, userAgent string, corporationID int64) ([]AllianceHistoryEntry, error) {
		return []AllianceHistoryEntry{{AllianceID: &allianceID, RecordID: 1, StartDate: time.Unix(0, 0)}}, nil
	}

	handler := listCorporationAllianceHistoryHandler("v1")
	_, out, err := handler(context.Background(), nil, ListCorporationAllianceHistoryInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.History) != 1 || out.History[0].RecordID != 1 {
		t.Errorf("out = %+v, want CorporationID=98000001 with one history entry RecordID=1", out)
	}
}

func TestGetCorporationIconsHandler_MapsFields(t *testing.T) {
	orig := getCorporationIcons
	t.Cleanup(func() { getCorporationIcons = orig })
	url64 := "https://images.example.com/64.png"
	getCorporationIcons = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationIconsOutput, error) {
		return GetCorporationIconsOutput{CorporationID: corporationID, Px64x64: &url64}, nil
	}

	handler := getCorporationIconsHandler("v1")
	_, out, err := handler(context.Background(), nil, GetCorporationIconsInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.Px64x64 == nil || *out.Px64x64 != url64 {
		t.Errorf("out = %+v, want CorporationID=98000001 Px64x64=%q", out, url64)
	}
}

func TestListNPCCorporationsHandler_MapsFields(t *testing.T) {
	orig := listNPCCorporations
	t.Cleanup(func() { listNPCCorporations = orig })
	listNPCCorporations = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{1000001, 1000002}, nil
	}

	handler := listNPCCorporationsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListNPCCorporationsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.CorporationIDs) != 2 || out.CorporationIDs[0] != 1000001 {
		t.Errorf("out = %+v, want two corporation IDs starting with 1000001", out)
	}
}

func TestListCorporationMembersHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMembers
	t.Cleanup(func() { listCorporationMembers = orig })
	listCorporationMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]int64, error) {
		return []int64{100, 200}, nil
	}

	handler := listCorporationMembersHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMembersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.MemberIDs) != 2 || out.MemberIDs[0] != 100 {
		t.Errorf("out = %+v, want CorporationID=98000001 with two member IDs starting with 100", out)
	}
}

func TestListCorporationMembersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMembers
	t.Cleanup(func() { listCorporationMembers = orig })
	listCorporationMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]int64, error) {
		t.Fatal("listCorporationMembers should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationMembersHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMembersInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationMembersLimitHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationMembersLimit
	t.Cleanup(func() { getCorporationMembersLimit = orig })
	getCorporationMembersLimit = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (int64, error) {
		return 500, nil
	}

	handler := getCorporationMembersLimitHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCorporationMembersLimitInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.Limit != 500 {
		t.Errorf("out = %+v, want CorporationID=98000001 Limit=500", out)
	}
}

func TestGetCorporationMembersLimitHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationMembersLimit
	t.Cleanup(func() { getCorporationMembersLimit = orig })
	getCorporationMembersLimit = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (int64, error) {
		t.Fatal("getCorporationMembersLimit should not be called for an unknown character")
		return 0, nil
	}

	handler := getCorporationMembersLimitHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationMembersLimitInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMemberTitlesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMemberTitles
	t.Cleanup(func() { listCorporationMemberTitles = orig })
	listCorporationMemberTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTitles, error) {
		return []MemberTitles{{CharacterID: 100, TitleIDs: []int64{1, 2}}}, nil
	}

	handler := listCorporationMemberTitlesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMemberTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.MemberTitles) != 1 || out.MemberTitles[0].CharacterID != 100 {
		t.Errorf("out = %+v, want CorporationID=98000001 with one MemberTitles entry CharacterID=100", out)
	}
}

func TestListCorporationMemberTitlesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMemberTitles
	t.Cleanup(func() { listCorporationMemberTitles = orig })
	listCorporationMemberTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTitles, error) {
		t.Fatal("listCorporationMemberTitles should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationMemberTitlesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTitlesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMemberTrackingHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMemberTracking
	t.Cleanup(func() { listCorporationMemberTracking = orig })
	locationID := int64(60003760)
	listCorporationMemberTracking = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTracking, error) {
		return []MemberTracking{{CharacterID: 100, LocationID: &locationID}}, nil
	}

	handler := listCorporationMemberTrackingHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Members) != 1 || out.Members[0].LocationID == nil || *out.Members[0].LocationID != locationID {
		t.Errorf("out = %+v, want CorporationID=98000001 with one member LocationID=%d", out, locationID)
	}
}

func TestListCorporationMemberTrackingHandler_ResolvesShipTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMemberTracking
	t.Cleanup(func() { listCorporationMemberTracking = orig })
	listCorporationMemberTracking = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTracking, error) {
		return []MemberTracking{{CharacterID: 1, ShipTypeID: new(int64(587))}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationMemberTrackingHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Members) != 1 || out.Members[0].ShipTypeName != "Rifter" {
		t.Errorf("Members[0].ShipTypeName = %q, want %q", out.Members[0].ShipTypeName, "Rifter")
	}
}

func TestListCorporationMemberTrackingHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMemberTracking
	t.Cleanup(func() { listCorporationMemberTracking = orig })
	listCorporationMemberTracking = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTracking, error) {
		t.Fatal("listCorporationMemberTracking should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationMemberTrackingHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationRolesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationRoles
	t.Cleanup(func() { listCorporationRoles = orig })
	listCorporationRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberRoles, error) {
		return []MemberRoles{{CharacterID: 100, Roles: []string{"Director"}}}, nil
	}

	handler := listCorporationRolesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationRolesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Roles) != 1 || out.Roles[0].Roles[0] != "Director" {
		t.Errorf("out = %+v, want CorporationID=98000001 with one MemberRoles entry Roles=[Director]", out)
	}
}

func TestListCorporationRolesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationRoles
	t.Cleanup(func() { listCorporationRoles = orig })
	listCorporationRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberRoles, error) {
		t.Fatal("listCorporationRoles should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationRolesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationRolesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationRolesHistoryHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationRolesHistory
	t.Cleanup(func() { listCorporationRolesHistory = orig })
	listCorporationRolesHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]RoleChange, int32, error) {
		return []RoleChange{{CharacterID: 100, IssuerID: 200, RoleType: "grant"}}, 4, nil
	}

	handler := listCorporationRolesHistoryHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationRolesHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 || out.TotalPages != 4 || len(out.History) != 1 || out.History[0].RoleType != "grant" {
		t.Errorf("out = %+v, want Page=1 TotalPages=4 with one RoleChange RoleType=grant", out)
	}
}

func TestListCorporationRolesHistoryHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationRolesHistory
	t.Cleanup(func() { listCorporationRolesHistory = orig })
	listCorporationRolesHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]RoleChange, int32, error) {
		t.Fatal("listCorporationRolesHistory should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationRolesHistoryHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationRolesHistoryInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationTitlesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationTitles
	t.Cleanup(func() { listCorporationTitles = orig })
	name := "Officer"
	listCorporationTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]CorporationTitle, error) {
		return []CorporationTitle{{Name: &name}}, nil
	}

	handler := listCorporationTitlesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Titles) != 1 || out.Titles[0].Name == nil || *out.Titles[0].Name != name {
		t.Errorf("out = %+v, want CorporationID=98000001 with one title Name=%q", out, name)
	}
}

func TestListCorporationTitlesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationTitles
	t.Cleanup(func() { listCorporationTitles = orig })
	listCorporationTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]CorporationTitle, error) {
		t.Fatal("listCorporationTitles should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationTitlesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationTitlesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationDivisionsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationDivisions
	t.Cleanup(func() { getCorporationDivisions = orig })
	hangarName := "Hangar 1"
	getCorporationDivisions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Division, []Division, error) {
		return []Division{{Name: &hangarName}}, []Division{}, nil
	}

	handler := getCorporationDivisionsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCorporationDivisionsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Hangars) != 1 || out.Hangars[0].Name == nil || *out.Hangars[0].Name != hangarName || len(out.Wallets) != 0 {
		t.Errorf("out = %+v, want CorporationID=98000001 with one hangar Name=%q and no wallets", out, hangarName)
	}
}

func TestGetCorporationDivisionsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationDivisions
	t.Cleanup(func() { getCorporationDivisions = orig })
	getCorporationDivisions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Division, []Division, error) {
		t.Fatal("getCorporationDivisions should not be called for an unknown character")
		return nil, nil, nil
	}

	handler := getCorporationDivisionsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationDivisionsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationShareholdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationShareholders
	t.Cleanup(func() { listCorporationShareholders = orig })
	listCorporationShareholders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Shareholder, int32, error) {
		return []Shareholder{{ShareholderID: 100, ShareholderType: "character", ShareCount: 1000}}, 2, nil
	}

	handler := listCorporationShareholdersHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationShareholdersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 || out.TotalPages != 2 || len(out.Shareholders) != 1 || out.Shareholders[0].ShareCount != 1000 {
		t.Errorf("out = %+v, want Page=1 TotalPages=2 with one shareholder ShareCount=1000", out)
	}
}

func TestListCorporationShareholdersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationShareholders
	t.Cleanup(func() { listCorporationShareholders = orig })
	listCorporationShareholders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Shareholder, int32, error) {
		t.Fatal("listCorporationShareholders should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationShareholdersHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationShareholdersInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationStandingsHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStandings
	t.Cleanup(func() { listCorporationStandings = orig })
	listCorporationStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Standing, int32, error) {
		return []Standing{{FromID: 500001, FromType: "npcCorp", Standing: 5.0}}, 3, nil
	}

	handler := listCorporationStandingsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationStandingsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 || out.TotalPages != 3 || len(out.Standings) != 1 || out.Standings[0].Standing != 5.0 {
		t.Errorf("out = %+v, want Page=1 TotalPages=3 with one standing Standing=5.0", out)
	}
}

func TestListCorporationStandingsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationStandings
	t.Cleanup(func() { listCorporationStandings = orig })
	listCorporationStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Standing, int32, error) {
		t.Fatal("listCorporationStandings should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationStandingsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationStandingsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationBlueprintsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationBlueprints
	t.Cleanup(func() { listCorporationBlueprints = orig })
	listCorporationBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationBlueprint, int32, error) {
		return []CorporationBlueprint{{
			ItemID: 1001, LocationFlag: "Hangar", LocationID: 60003760,
			MaterialEfficiency: 10, Quantity: -1, Runs: -1, TimeEfficiency: 20, TypeID: 950,
		}}, 2, nil
	}

	handler := listCorporationBlueprintsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationBlueprintsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Blueprints) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one blueprint", out)
	}
	bp := out.Blueprints[0]
	if bp.ItemID != 1001 || bp.LocationFlag != "Hangar" || bp.LocationID != 60003760 || bp.MaterialEfficiency != 10 ||
		bp.Quantity != -1 || bp.Runs != -1 || bp.TimeEfficiency != 20 || bp.TypeID != 950 {
		t.Errorf("blueprint = %+v, want ItemID=1001 LocationFlag=Hangar LocationID=60003760 MaterialEfficiency=10 Quantity=-1 Runs=-1 TimeEfficiency=20 TypeID=950", bp)
	}
}

func TestListCorporationBlueprintsHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationBlueprints
	t.Cleanup(func() { listCorporationBlueprints = orig })
	listCorporationBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationBlueprint, int32, error) {
		return []CorporationBlueprint{{ItemID: 1, TypeID: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationBlueprintsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationBlueprintsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Blueprints) != 1 || out.Blueprints[0].TypeName != "Rifter" {
		t.Errorf("Blueprints[0].TypeName = %q, want %q", out.Blueprints[0].TypeName, "Rifter")
	}
}

func TestListCorporationBlueprintsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationBlueprints
	t.Cleanup(func() { listCorporationBlueprints = orig })
	listCorporationBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationBlueprint, int32, error) {
		t.Fatal("listCorporationBlueprints should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationBlueprintsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationBlueprintsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContainerLogsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContainerLogs
	t.Cleanup(func() { listCorporationContainerLogs = orig })
	loggedAt := time.Unix(0, 0)
	newBitmask := int64(1)
	oldBitmask := int64(2)
	passwordType := "Config"
	quantity := int64(5)
	typeID := int64(17366)
	listCorporationContainerLogs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]ContainerLog, int32, error) {
		return []ContainerLog{{
			Action: "unlock", CharacterID: 100, ContainerID: 200, ContainerTypeID: 300,
			LocationFlag: "Hangar", LocationID: 60003760, LoggedAt: loggedAt,
			NewConfigBitmask: &newBitmask, OldConfigBitmask: &oldBitmask,
			PasswordType: &passwordType, Quantity: &quantity, TypeID: &typeID,
		}}, 3, nil
	}

	handler := listCorporationContainerLogsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationContainerLogsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 3 || len(out.Logs) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=3 with one log", out)
	}
	l := out.Logs[0]
	if l.Action != "unlock" || l.CharacterID != 100 || l.ContainerID != 200 || l.ContainerTypeID != 300 ||
		l.LocationFlag != "Hangar" || l.LocationID != 60003760 || !l.LoggedAt.Equal(loggedAt) ||
		l.NewConfigBitmask == nil || *l.NewConfigBitmask != newBitmask ||
		l.OldConfigBitmask == nil || *l.OldConfigBitmask != oldBitmask ||
		l.PasswordType == nil || *l.PasswordType != passwordType ||
		l.Quantity == nil || *l.Quantity != quantity ||
		l.TypeID == nil || *l.TypeID != typeID {
		t.Errorf("log = %+v, want Action=unlock CharacterID=100 ContainerID=200 ContainerTypeID=300 LocationFlag=Hangar LocationID=60003760 LoggedAt=%v NewConfigBitmask=%d OldConfigBitmask=%d PasswordType=%q Quantity=%d TypeID=%d",
			l, loggedAt, newBitmask, oldBitmask, passwordType, quantity, typeID)
	}
}

func TestListCorporationContainerLogsHandler_ResolvesTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContainerLogs
	t.Cleanup(func() { listCorporationContainerLogs = orig })
	listCorporationContainerLogs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]ContainerLog, int32, error) {
		return []ContainerLog{{ContainerTypeID: 588, TypeID: new(int64(587))}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := listCorporationContainerLogsHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationContainerLogsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Logs) != 1 || out.Logs[0].ContainerTypeName != "Slasher" || out.Logs[0].TypeName != "Rifter" {
		t.Errorf("Logs[0].ContainerTypeName = %q, TypeName = %q, want ContainerTypeName=%q TypeName=%q",
			out.Logs[0].ContainerTypeName, out.Logs[0].TypeName, "Slasher", "Rifter")
	}
}

func TestListCorporationContainerLogsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContainerLogs
	t.Cleanup(func() { listCorporationContainerLogs = orig })
	listCorporationContainerLogs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]ContainerLog, int32, error) {
		t.Fatal("listCorporationContainerLogs should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationContainerLogsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContainerLogsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationFacilitiesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFacilities
	t.Cleanup(func() { listCorporationFacilities = orig })
	listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
		return []Facility{{FacilityID: 500, SystemID: 30000142, TypeID: 2502}}, nil
	}

	handler := listCorporationFacilitiesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || len(out.Facilities) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 with one facility", out)
	}
	f := out.Facilities[0]
	if f.FacilityID != 500 || f.SystemID != 30000142 || f.TypeID != 2502 {
		t.Errorf("facility = %+v, want FacilityID=500 SystemID=30000142 TypeID=2502", f)
	}
}

func TestListCorporationFacilitiesHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFacilities
	t.Cleanup(func() { listCorporationFacilities = orig })
	listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
		return []Facility{{FacilityID: 1, TypeID: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationFacilitiesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Facilities) != 1 || out.Facilities[0].TypeName != "Rifter" {
		t.Errorf("Facilities[0].TypeName = %q, want %q", out.Facilities[0].TypeName, "Rifter")
	}
}

func TestListCorporationFacilitiesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationFacilities
	t.Cleanup(func() { listCorporationFacilities = orig })
	listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
		t.Fatal("listCorporationFacilities should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationFacilitiesHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMedalsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMedals
	t.Cleanup(func() { listCorporationMedals = orig })
	createdAt := time.Unix(0, 0)
	listCorporationMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationMedal, int32, error) {
		return []CorporationMedal{{
			CreatedAt: createdAt, CreatorID: 100, Description: "For valor", MedalID: 1, Title: "Valor",
		}}, 2, nil
	}

	handler := listCorporationMedalsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMedalsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Medals) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one medal", out)
	}
	m := out.Medals[0]
	if !m.CreatedAt.Equal(createdAt) || m.CreatorID != 100 || m.Description != "For valor" || m.MedalID != 1 || m.Title != "Valor" {
		t.Errorf("medal = %+v, want CreatedAt=%v CreatorID=100 Description=%q MedalID=1 Title=%q", m, createdAt, "For valor", "Valor")
	}
}

func TestListCorporationMedalsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMedals
	t.Cleanup(func() { listCorporationMedals = orig })
	listCorporationMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationMedal, int32, error) {
		t.Fatal("listCorporationMedals should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMedalsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMedalsIssuedHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMedalsIssued
	t.Cleanup(func() { listCorporationMedalsIssued = orig })
	issuedAt := time.Unix(0, 0)
	listCorporationMedalsIssued = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]MedalIssued, int32, error) {
		return []MedalIssued{{
			CharacterID: 100, IssuedAt: issuedAt, IssuerID: 200, MedalID: 1, Reason: "Great work", Status: "private",
		}}, 2, nil
	}

	handler := listCorporationMedalsIssuedHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationMedalsIssuedInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Issued) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one issued medal", out)
	}
	m := out.Issued[0]
	if m.CharacterID != 100 || !m.IssuedAt.Equal(issuedAt) || m.IssuerID != 200 || m.MedalID != 1 || m.Reason != "Great work" || m.Status != "private" {
		t.Errorf("issued medal = %+v, want CharacterID=100 IssuedAt=%v IssuerID=200 MedalID=1 Reason=%q Status=%q", m, issuedAt, "Great work", "private")
	}
}

func TestListCorporationMedalsIssuedHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationMedalsIssued
	t.Cleanup(func() { listCorporationMedalsIssued = orig })
	listCorporationMedalsIssued = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]MedalIssued, int32, error) {
		t.Fatal("listCorporationMedalsIssued should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMedalsIssuedHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsIssuedInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationStarbasesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStarbases
	t.Cleanup(func() { listCorporationStarbases = orig })
	moonID := int64(40000001)
	onlinedSince := time.Unix(1000, 0)
	reinforcedUntil := time.Unix(2000, 0)
	state := "online"
	unanchorAt := time.Unix(3000, 0)
	listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
		return []Starbase{{
			MoonID: &moonID, OnlinedSince: &onlinedSince, ReinforcedUntil: &reinforcedUntil,
			StarbaseID: 900, State: &state, SystemID: 30000142, TypeID: 16213, UnanchorAt: &unanchorAt,
		}}, 2, nil
	}

	handler := listCorporationStarbasesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Starbases) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one starbase", out)
	}
	s := out.Starbases[0]
	if s.MoonID == nil || *s.MoonID != moonID ||
		s.OnlinedSince == nil || !s.OnlinedSince.Equal(onlinedSince) ||
		s.ReinforcedUntil == nil || !s.ReinforcedUntil.Equal(reinforcedUntil) ||
		s.StarbaseID != 900 || s.State == nil || *s.State != state ||
		s.SystemID != 30000142 || s.TypeID != 16213 ||
		s.UnanchorAt == nil || !s.UnanchorAt.Equal(unanchorAt) {
		t.Errorf("starbase = %+v, want MoonID=%d OnlinedSince=%v ReinforcedUntil=%v StarbaseID=900 State=%q SystemID=30000142 TypeID=16213 UnanchorAt=%v",
			s, moonID, onlinedSince, reinforcedUntil, state, unanchorAt)
	}
}

func TestListCorporationStarbasesHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStarbases
	t.Cleanup(func() { listCorporationStarbases = orig })
	listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
		return []Starbase{{StarbaseID: 1, TypeID: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationStarbasesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Starbases) != 1 || out.Starbases[0].TypeName != "Rifter" {
		t.Errorf("Starbases[0].TypeName = %q, want %q", out.Starbases[0].TypeName, "Rifter")
	}
}

func TestListCorporationStarbasesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationStarbases
	t.Cleanup(func() { listCorporationStarbases = orig })
	listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
		t.Fatal("listCorporationStarbases should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationStarbasesHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationStarbaseHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationStarbase
	t.Cleanup(func() { getCorporationStarbase = orig })
	attackSecThreshold := 0.5
	attackStandingThreshold := 0.0
	getCorporationStarbase = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, starbaseID int64, systemID int64) (StarbaseDetail, error) {
		return StarbaseDetail{
			AllowAllianceMembers: true, AllowCorporationMembers: true, Anchor: "starbaseFuelTechnician",
			AttackIfAtWar: true, AttackIfOtherSecurityStatusDropping: true,
			AttackSecurityStatusThreshold: &attackSecThreshold, AttackStandingThreshold: &attackStandingThreshold,
			FuelBayTake: "starbaseFuelTechnician", FuelBayView: "corporationMembers",
			Fuels:                []StarbaseFuel{{Quantity: 100, TypeID: 4312}},
			Offline:              "starbaseFuelTechnician",
			Online:               "starbaseFuelTechnician",
			Unanchor:             "starbaseFuelTechnician",
			UseAllianceStandings: true,
		}, nil
	}

	handler := getCorporationStarbaseHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 7, CorporationID: 98000001, StarbaseID: 900, SystemID: 30000142})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.StarbaseID != 900 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 StarbaseID=900", out)
	}
	d := out.Detail
	if !d.AllowAllianceMembers || !d.AllowCorporationMembers || d.Anchor != "starbaseFuelTechnician" ||
		!d.AttackIfAtWar || !d.AttackIfOtherSecurityStatusDropping ||
		d.AttackSecurityStatusThreshold == nil || *d.AttackSecurityStatusThreshold != attackSecThreshold ||
		d.AttackStandingThreshold == nil || *d.AttackStandingThreshold != attackStandingThreshold ||
		d.FuelBayTake != "starbaseFuelTechnician" || d.FuelBayView != "corporationMembers" ||
		len(d.Fuels) != 1 || d.Fuels[0].Quantity != 100 || d.Fuels[0].TypeID != 4312 ||
		d.Offline != "starbaseFuelTechnician" || d.Online != "starbaseFuelTechnician" || d.Unanchor != "starbaseFuelTechnician" ||
		!d.UseAllianceStandings {
		t.Errorf("detail = %+v, want all bools true, Anchor/FuelBayTake/Offline/Online/Unanchor=starbaseFuelTechnician, FuelBayView=corporationMembers, one fuel Quantity=100 TypeID=4312, thresholds %v/%v",
			d, attackSecThreshold, attackStandingThreshold)
	}
}

func TestGetCorporationStarbaseHandler_ResolvesFuelTypeNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationStarbase
	t.Cleanup(func() { getCorporationStarbase = orig })
	getCorporationStarbase = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, starbaseID int64, systemID int64) (StarbaseDetail, error) {
		return StarbaseDetail{Fuels: []StarbaseFuel{{TypeID: 587}}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := getCorporationStarbaseHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 7, CorporationID: 98000001, StarbaseID: 900, SystemID: 30000142})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Detail.Fuels) != 1 || out.Detail.Fuels[0].TypeName != "Rifter" {
		t.Errorf("Detail.Fuels[0].TypeName = %q, want %q", out.Detail.Fuels[0].TypeName, "Rifter")
	}
}

func TestGetCorporationStarbaseHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCorporationStarbase
	t.Cleanup(func() { getCorporationStarbase = orig })
	getCorporationStarbase = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, starbaseID int64, systemID int64) (StarbaseDetail, error) {
		t.Fatal("getCorporationStarbase should not be called for an unknown character")
		return StarbaseDetail{}, nil
	}

	handler := getCorporationStarbaseHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 999, CorporationID: 98000001, StarbaseID: 900, SystemID: 30000142})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationStructuresHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStructures
	t.Cleanup(func() { listCorporationStructures = orig })
	fuelExpires := time.Unix(1000, 0)
	name := "My Keepstar"
	nextReinforceApply := time.Unix(2000, 0)
	nextReinforceHour := int64(18)
	reinforceHour := int64(19)
	stateTimerEnd := time.Unix(3000, 0)
	stateTimerStart := time.Unix(4000, 0)
	unanchorsAt := time.Unix(5000, 0)
	listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
		return []Structure{{
			CorporationID: 98000001, FuelExpires: &fuelExpires, Name: &name,
			NextReinforceApply: &nextReinforceApply, NextReinforceHour: &nextReinforceHour, ProfileID: 101,
			ReinforceHour: &reinforceHour, Services: []StructureService{{Name: "Clone Bay", State: "online"}},
			State: "shield_vulnerable", StateTimerEnd: &stateTimerEnd, StateTimerStart: &stateTimerStart,
			StructureID: 1000000000001, SystemID: 30000142, TypeID: 35832, UnanchorsAt: &unanchorsAt,
		}}, 2, nil
	}

	handler := listCorporationStructuresHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Structures) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one structure", out)
	}
	s := out.Structures[0]
	if s.CorporationID != 98000001 ||
		s.FuelExpires == nil || !s.FuelExpires.Equal(fuelExpires) ||
		s.Name == nil || *s.Name != name ||
		s.NextReinforceApply == nil || !s.NextReinforceApply.Equal(nextReinforceApply) ||
		s.NextReinforceHour == nil || *s.NextReinforceHour != nextReinforceHour ||
		s.ProfileID != 101 ||
		s.ReinforceHour == nil || *s.ReinforceHour != reinforceHour ||
		len(s.Services) != 1 || s.Services[0].Name != "Clone Bay" || s.Services[0].State != "online" ||
		s.State != "shield_vulnerable" ||
		s.StateTimerEnd == nil || !s.StateTimerEnd.Equal(stateTimerEnd) ||
		s.StateTimerStart == nil || !s.StateTimerStart.Equal(stateTimerStart) ||
		s.StructureID != 1000000000001 || s.SystemID != 30000142 || s.TypeID != 35832 ||
		s.UnanchorsAt == nil || !s.UnanchorsAt.Equal(unanchorsAt) {
		t.Errorf("structure = %+v, want fields matching fixture (CorporationID=98000001, Name=%q, ProfileID=101, one service Name=Clone Bay State=online, State=shield_vulnerable, StructureID=1000000000001, SystemID=30000142, TypeID=35832)",
			s, name)
	}
}

func TestListCorporationStructuresHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStructures
	t.Cleanup(func() { listCorporationStructures = orig })
	listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
		return []Structure{{StructureID: 1, TypeID: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationStructuresHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Structures) != 1 || out.Structures[0].TypeName != "Rifter" {
		t.Errorf("Structures[0].TypeName = %q, want %q", out.Structures[0].TypeName, "Rifter")
	}
}

func TestListCorporationStructuresHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationStructures
	t.Cleanup(func() { listCorporationStructures = orig })
	listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
		t.Fatal("listCorporationStructures should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationStructuresHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationStarbasesHandler_ResolvesLocationNames(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStarbases
	t.Cleanup(func() { listCorporationStarbases = orig })
	moonID := int64(40009087)
	listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
		return []Starbase{{StarbaseID: 1, SystemID: 30000142, MoonID: &moonID}}, 1, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}
	origMoon := resolveMoonNames
	t.Cleanup(func() { resolveMoonNames = origMoon })
	resolveMoonNames = func(ctx context.Context, sdeStore moonNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{40009087: "Jita IV - Moon 4"}
	}

	handler := listCorporationStarbasesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Starbases[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Starbases[0].SystemName, "Jita")
	}
	if out.Starbases[0].MoonName != "Jita IV - Moon 4" {
		t.Errorf("MoonName = %q, want %q", out.Starbases[0].MoonName, "Jita IV - Moon 4")
	}
}

func TestListCorporationFacilitiesHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFacilities
	t.Cleanup(func() { listCorporationFacilities = orig })
	listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
		return []Facility{{FacilityID: 1, SystemID: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationFacilitiesHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Facilities[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Facilities[0].SystemName, "Jita")
	}
}

func TestListCorporationStructuresHandler_ResolvesSystemName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStructures
	t.Cleanup(func() { listCorporationStructures = orig })
	listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
		return []Structure{{StructureID: 1, SystemID: 30000142}}, 1, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationStructuresHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Structures[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Structures[0].SystemName, "Jita")
	}
}

func TestGetCorporationHandler_ResolvesHomeStationName(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{CorporationID: 98000001, HomeStationID: 60003760}, nil
	}

	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := getCorporationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.HomeStationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("HomeStationName = %q, want %q", out.HomeStationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}

func TestGetCorporationHandler_PropagatesError(t *testing.T) {
	orig := getCorporationDetail
	t.Cleanup(func() { getCorporationDetail = orig })
	wantErr := errors.New("upstream failure")
	getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
		return GetCorporationOutput{}, wantErr
	}

	handler := getCorporationHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationAllianceHistoryHandler_PropagatesError(t *testing.T) {
	orig := listCorporationAllianceHistory
	t.Cleanup(func() { listCorporationAllianceHistory = orig })
	wantErr := errors.New("upstream failure")
	listCorporationAllianceHistory = func(ctx context.Context, userAgent string, corporationID int64) ([]AllianceHistoryEntry, error) {
		return nil, wantErr
	}

	handler := listCorporationAllianceHistoryHandler("v1")
	_, _, err := handler(context.Background(), nil, ListCorporationAllianceHistoryInput{CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationIconsHandler_PropagatesError(t *testing.T) {
	orig := getCorporationIcons
	t.Cleanup(func() { getCorporationIcons = orig })
	wantErr := errors.New("upstream failure")
	getCorporationIcons = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationIconsOutput, error) {
		return GetCorporationIconsOutput{}, wantErr
	}

	handler := getCorporationIconsHandler("v1")
	_, _, err := handler(context.Background(), nil, GetCorporationIconsInput{CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListNPCCorporationsHandler_PropagatesError(t *testing.T) {
	orig := listNPCCorporations
	t.Cleanup(func() { listNPCCorporations = orig })
	wantErr := errors.New("upstream failure")
	listNPCCorporations = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listNPCCorporationsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListNPCCorporationsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMembersHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMembers
	t.Cleanup(func() { listCorporationMembers = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]int64, error) {
		return nil, wantErr
	}

	handler := listCorporationMembersHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMembersInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationMembersLimitHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationMembersLimit
	t.Cleanup(func() { getCorporationMembersLimit = orig })
	wantErr := errors.New("upstream failure")
	getCorporationMembersLimit = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (int64, error) {
		return 0, wantErr
	}

	handler := getCorporationMembersLimitHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationMembersLimitInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMemberTitlesHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMemberTitles
	t.Cleanup(func() { listCorporationMemberTitles = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMemberTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTitles, error) {
		return nil, wantErr
	}

	handler := listCorporationMemberTitlesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMemberTrackingHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMemberTracking
	t.Cleanup(func() { listCorporationMemberTracking = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMemberTracking = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTracking, error) {
		return nil, wantErr
	}

	handler := listCorporationMemberTrackingHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationRolesHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationRoles
	t.Cleanup(func() { listCorporationRoles = orig })
	wantErr := errors.New("upstream failure")
	listCorporationRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberRoles, error) {
		return nil, wantErr
	}

	handler := listCorporationRolesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationRolesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationRolesHistoryHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationRolesHistory
	t.Cleanup(func() { listCorporationRolesHistory = orig })
	wantErr := errors.New("upstream failure")
	listCorporationRolesHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]RoleChange, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationRolesHistoryHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationRolesHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationTitlesHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationTitles
	t.Cleanup(func() { listCorporationTitles = orig })
	wantErr := errors.New("upstream failure")
	listCorporationTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]CorporationTitle, error) {
		return nil, wantErr
	}

	handler := listCorporationTitlesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationDivisionsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationDivisions
	t.Cleanup(func() { getCorporationDivisions = orig })
	wantErr := errors.New("upstream failure")
	getCorporationDivisions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Division, []Division, error) {
		return nil, nil, wantErr
	}

	handler := getCorporationDivisionsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationDivisionsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationShareholdersHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationShareholders
	t.Cleanup(func() { listCorporationShareholders = orig })
	wantErr := errors.New("upstream failure")
	listCorporationShareholders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Shareholder, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationShareholdersHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationShareholdersInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationStandingsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStandings
	t.Cleanup(func() { listCorporationStandings = orig })
	wantErr := errors.New("upstream failure")
	listCorporationStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Standing, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationStandingsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationStandingsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationBlueprintsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationBlueprints
	t.Cleanup(func() { listCorporationBlueprints = orig })
	wantErr := errors.New("upstream failure")
	listCorporationBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationBlueprint, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationBlueprintsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationBlueprintsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationContainerLogsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContainerLogs
	t.Cleanup(func() { listCorporationContainerLogs = orig })
	wantErr := errors.New("upstream failure")
	listCorporationContainerLogs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]ContainerLog, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationContainerLogsHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContainerLogsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationFacilitiesHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationFacilities
	t.Cleanup(func() { listCorporationFacilities = orig })
	wantErr := errors.New("upstream failure")
	listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
		return nil, wantErr
	}

	handler := listCorporationFacilitiesHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMedalsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMedals
	t.Cleanup(func() { listCorporationMedals = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationMedal, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationMedalsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMedalsIssuedHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationMedalsIssued
	t.Cleanup(func() { listCorporationMedalsIssued = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMedalsIssued = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]MedalIssued, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationMedalsIssuedHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsIssuedInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationStarbasesHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStarbases
	t.Cleanup(func() { listCorporationStarbases = orig })
	wantErr := errors.New("upstream failure")
	listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationStarbasesHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationStarbaseHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCorporationStarbase
	t.Cleanup(func() { getCorporationStarbase = orig })
	wantErr := errors.New("upstream failure")
	getCorporationStarbase = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, starbaseID int64, systemID int64) (StarbaseDetail, error) {
		return StarbaseDetail{}, wantErr
	}

	handler := getCorporationStarbaseHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 7, CorporationID: 98000001, StarbaseID: 1000000000001, SystemID: 30000142})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationStructuresHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationStructures
	t.Cleanup(func() { listCorporationStructures = orig })
	wantErr := errors.New("upstream failure")
	listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationStructuresHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
