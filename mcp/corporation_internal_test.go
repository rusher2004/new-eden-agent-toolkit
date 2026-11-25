package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

//go:fix inline
func ptrInt64(v int64) *int64 { return new(v) }

// fakeCorporationESI fakes every corporation ESI interface in corporation.go
// except the blueprints one; a nil func field fails as an unfaked call.
type fakeCorporationESI struct {
	corporationDetail         func(ctx context.Context, corporationID int64) (*esiapi.CorporationsDetail, error)
	corporationAllianceHist   func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdAlliancehistoryGetInner, error)
	corporationIcons          func(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdIconsGet, error)
	npcCorporations           func(ctx context.Context) ([]int64, error)
	corporationMembers        func(ctx context.Context, corporationID int64) ([]int64, error)
	corporationMembersLimit   func(ctx context.Context, corporationID int64) (int64, error)
	corporationMemberTitles   func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembersTitlesGetInner, error)
	corporationMemberTracking func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error)
	corporationRoles          func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdRolesGetInner, error)
	corporationRolesHistory   func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdRolesHistoryGetInner, int32, error)
	corporationTitles         func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdTitlesGetInner, error)
	corporationDivisions      func(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdDivisionsGet, error)
	corporationShareholders   func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdShareholdersGetInner, int32, error)
	corporationStandings      func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdStandingsGetInner, int32, error)
	corporationContainerLogs  func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error)
	corporationFacilities     func(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error)
	corporationMedals         func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsGetInner, int32, error)
	corporationMedalsIssued   func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsIssuedGetInner, int32, error)
	corporationStarbases      func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error)
	corporationStarbaseDetail func(ctx context.Context, corporationID, starbaseID, systemID int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error)
	corporationStructures     func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error)
}

func (f fakeCorporationESI) CorporationDetail(ctx context.Context, corporationID int64) (*esiapi.CorporationsDetail, error) {
	if f.corporationDetail == nil {
		return nil, errNotFaked
	}
	return f.corporationDetail(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationAllianceHistory(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdAlliancehistoryGetInner, error) {
	if f.corporationAllianceHist == nil {
		return nil, errNotFaked
	}
	return f.corporationAllianceHist(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationIcons(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdIconsGet, error) {
	if f.corporationIcons == nil {
		return nil, errNotFaked
	}
	return f.corporationIcons(ctx, corporationID)
}

func (f fakeCorporationESI) NPCCorporations(ctx context.Context) ([]int64, error) {
	if f.npcCorporations == nil {
		return nil, errNotFaked
	}
	return f.npcCorporations(ctx)
}

func (f fakeCorporationESI) CorporationMembers(ctx context.Context, corporationID int64) ([]int64, error) {
	if f.corporationMembers == nil {
		return nil, errNotFaked
	}
	return f.corporationMembers(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationMembersLimit(ctx context.Context, corporationID int64) (int64, error) {
	if f.corporationMembersLimit == nil {
		return 0, errNotFaked
	}
	return f.corporationMembersLimit(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationMemberTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembersTitlesGetInner, error) {
	if f.corporationMemberTitles == nil {
		return nil, errNotFaked
	}
	return f.corporationMemberTitles(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationMemberTracking(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error) {
	if f.corporationMemberTracking == nil {
		return nil, errNotFaked
	}
	return f.corporationMemberTracking(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationRoles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdRolesGetInner, error) {
	if f.corporationRoles == nil {
		return nil, errNotFaked
	}
	return f.corporationRoles(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationRolesHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdRolesHistoryGetInner, int32, error) {
	if f.corporationRolesHistory == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationRolesHistory(ctx, corporationID, page)
}

func (f fakeCorporationESI) CorporationTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdTitlesGetInner, error) {
	if f.corporationTitles == nil {
		return nil, errNotFaked
	}
	return f.corporationTitles(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationDivisions(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdDivisionsGet, error) {
	if f.corporationDivisions == nil {
		return nil, errNotFaked
	}
	return f.corporationDivisions(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationShareholders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdShareholdersGetInner, int32, error) {
	if f.corporationShareholders == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationShareholders(ctx, corporationID, page)
}

func (f fakeCorporationESI) CorporationStandings(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdStandingsGetInner, int32, error) {
	if f.corporationStandings == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationStandings(ctx, corporationID, page)
}

func (f fakeCorporationESI) CorporationContainerLogs(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error) {
	if f.corporationContainerLogs == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationContainerLogs(ctx, corporationID, page)
}

func (f fakeCorporationESI) CorporationFacilities(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error) {
	if f.corporationFacilities == nil {
		return nil, errNotFaked
	}
	return f.corporationFacilities(ctx, corporationID)
}

func (f fakeCorporationESI) CorporationMedals(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsGetInner, int32, error) {
	if f.corporationMedals == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationMedals(ctx, corporationID, page)
}

func (f fakeCorporationESI) CorporationMedalsIssued(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsIssuedGetInner, int32, error) {
	if f.corporationMedalsIssued == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationMedalsIssued(ctx, corporationID, page)
}

func (f fakeCorporationESI) CorporationStarbases(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error) {
	if f.corporationStarbases == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationStarbases(ctx, corporationID, page)
}

func (f fakeCorporationESI) CorporationStarbaseDetail(ctx context.Context, corporationID int64, starbaseID int64, systemID int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error) {
	if f.corporationStarbaseDetail == nil {
		return nil, errNotFaked
	}
	return f.corporationStarbaseDetail(ctx, corporationID, starbaseID, systemID)
}

func (f fakeCorporationESI) CorporationStructures(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error) {
	if f.corporationStructures == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationStructures(ctx, corporationID, page)
}

// corpPilotStore is a config with the one character (7) these tests act as.
func corpPilotStore(t *testing.T) *config.Store {
	t.Helper()
	return characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
}

// --- get_corporation ---

func TestGetCorporationHandler_MapsFields(t *testing.T) {
	factionID := int64(500001)
	secondary := "#222222"
	fake := fakeCorporationESI{corporationDetail: func(_ context.Context, corporationID int64) (*esiapi.CorporationsDetail, error) {
		return &esiapi.CorporationsDetail{
			Name: "Test Corp", Ticker: "TEST", EnlistedFactionId: &factionID, HomeStationId: 60003760,
			TaxRates: esiapi.CorporationsDetailTaxrates{Isk: 0.1, LoyaltyPoint: 0.2},
			Palette:  &esiapi.CorporationsDetailPalette{MainColor: "#111111", SecondaryColor: &secondary},
		}, nil
	}}

	handler := getCorporationHandler("v1", publicNarrow[corporationDetailESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.Name != "Test Corp" || out.Ticker != "TEST" {
		t.Errorf("out = %+v, want CorporationID=98000001 Name=Test Corp Ticker=TEST", out)
	}
	if out.FactionID == nil || *out.FactionID != factionID || out.HomeStationID != 60003760 {
		t.Errorf("out = %+v, want FactionID=500001 HomeStationID=60003760", out)
	}
	if out.TaxRateIsk != 0.1 || out.TaxRateLoyaltyPoint != 0.2 {
		t.Errorf("tax rates = %v/%v, want 0.1/0.2", out.TaxRateIsk, out.TaxRateLoyaltyPoint)
	}
	if out.MainColor != "#111111" || out.SecondaryColor == nil || *out.SecondaryColor != secondary || out.TertiaryColor != nil {
		t.Errorf("palette = %q/%v/%v, want #111111/#222222/nil", out.MainColor, out.SecondaryColor, out.TertiaryColor)
	}
}

func TestGetCorporationHandler_ResolvesFactionName(t *testing.T) {
	factionID := int64(500001)
	fake := fakeCorporationESI{corporationDetail: func(context.Context, int64) (*esiapi.CorporationsDetail, error) {
		return &esiapi.CorporationsDetail{EnlistedFactionId: &factionID, HomeStationId: 60003760}, nil
	}}

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

	handler := getCorporationHandler("v1", publicNarrow[corporationDetailESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetCorporationHandler_FactionNameEmptyWhenFactionIDNil(t *testing.T) {
	fake := fakeCorporationESI{corporationDetail: func(context.Context, int64) (*esiapi.CorporationsDetail, error) {
		return &esiapi.CorporationsDetail{HomeStationId: 60003760}, nil
	}}

	handler := getCorporationHandler("v1", publicNarrow[corporationDetailESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "" {
		t.Errorf("FactionName = %q, want empty when FactionID is nil", out.FactionName)
	}
}

func TestGetCorporationHandler_ResolvesHomeStationName(t *testing.T) {
	fake := fakeCorporationESI{corporationDetail: func(context.Context, int64) (*esiapi.CorporationsDetail, error) {
		return &esiapi.CorporationsDetail{HomeStationId: 60003760}, nil
	}}

	origStation := resolveStationNames
	t.Cleanup(func() { resolveStationNames = origStation })
	resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{60003760: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"}
	}

	handler := getCorporationHandler("v1", publicNarrow[corporationDetailESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.HomeStationName != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("HomeStationName = %q, want %q", out.HomeStationName, "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
}

func TestGetCorporationHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationDetail: func(context.Context, int64) (*esiapi.CorporationsDetail, error) { return nil, wantErr }}

	handler := getCorporationHandler("v1", publicNarrow[corporationDetailESI](fake), nil)
	_, _, err := handler(context.Background(), nil, GetCorporationInput{CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_alliance_history ---

func TestListCorporationAllianceHistoryHandler_MapsFields(t *testing.T) {
	allianceID := int64(99000001)
	fake := fakeCorporationESI{corporationAllianceHist: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdAlliancehistoryGetInner, error) {
		return []esiapi.CorporationsCorporationIdAlliancehistoryGetInner{{AllianceId: &allianceID, RecordId: 1, StartDate: time.Unix(0, 0)}}, nil
	}}

	handler := listCorporationAllianceHistoryHandler(publicNarrow[corporationAllianceHistoryESI](fake))
	_, out, err := handler(context.Background(), nil, ListCorporationAllianceHistoryInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.History) != 1 || out.History[0].RecordID != 1 {
		t.Errorf("out = %+v, want CorporationID=98000001 with one history entry RecordID=1", out)
	}
	if out.History[0].AllianceID == nil || *out.History[0].AllianceID != allianceID {
		t.Errorf("AllianceID = %v, want %d", out.History[0].AllianceID, allianceID)
	}
}

func TestListCorporationAllianceHistoryHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationAllianceHist: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdAlliancehistoryGetInner, error) {
		return nil, wantErr
	}}

	handler := listCorporationAllianceHistoryHandler(publicNarrow[corporationAllianceHistoryESI](fake))
	_, _, err := handler(context.Background(), nil, ListCorporationAllianceHistoryInput{CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_corporation_icons ---

func TestGetCorporationIconsHandler_MapsFields(t *testing.T) {
	url64 := "https://images.example.com/64.png"
	fake := fakeCorporationESI{corporationIcons: func(context.Context, int64) (*esiapi.CorporationsCorporationIdIconsGet, error) {
		return &esiapi.CorporationsCorporationIdIconsGet{Px64x64: &url64}, nil
	}}

	handler := getCorporationIconsHandler(publicNarrow[corporationIconsESI](fake))
	_, out, err := handler(context.Background(), nil, GetCorporationIconsInput{CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.Px64x64 == nil || *out.Px64x64 != url64 {
		t.Errorf("out = %+v, want CorporationID=98000001 Px64x64=%q", out, url64)
	}
}

func TestGetCorporationIconsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationIcons: func(context.Context, int64) (*esiapi.CorporationsCorporationIdIconsGet, error) {
		return nil, wantErr
	}}

	handler := getCorporationIconsHandler(publicNarrow[corporationIconsESI](fake))
	_, _, err := handler(context.Background(), nil, GetCorporationIconsInput{CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_npc_corporations ---

func TestListNPCCorporationsHandler_MapsFields(t *testing.T) {
	fake := fakeCorporationESI{npcCorporations: func(context.Context) ([]int64, error) { return []int64{1000001, 1000002}, nil }}

	handler := listNPCCorporationsHandler(publicNarrow[npcCorporationsESI](fake))
	_, out, err := handler(context.Background(), nil, ListNPCCorporationsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.CorporationIDs) != 2 || out.CorporationIDs[0] != 1000001 {
		t.Errorf("out = %+v, want two corporation IDs starting with 1000001", out)
	}
}

func TestListNPCCorporationsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{npcCorporations: func(context.Context) ([]int64, error) { return nil, wantErr }}

	handler := listNPCCorporationsHandler(publicNarrow[npcCorporationsESI](fake))
	_, _, err := handler(context.Background(), nil, ListNPCCorporationsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_members ---

func TestListCorporationMembersHandler_MapsFields(t *testing.T) {
	fake := fakeCorporationESI{corporationMembers: func(context.Context, int64) ([]int64, error) { return []int64{100, 200}, nil }}

	handler := listCorporationMembersHandler(openNarrow[corporationMembersESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, ListCorporationMembersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.MemberIDs) != 2 || out.MemberIDs[0] != 100 {
		t.Errorf("out = %+v, want CorporationID=98000001 with two member IDs starting with 100", out)
	}
}

func TestListCorporationMembersHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationMembersHandler(noNarrow[corporationMembersESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMembersInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMembersHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationMembers: func(context.Context, int64) ([]int64, error) { return nil, wantErr }}

	handler := listCorporationMembersHandler(openNarrow[corporationMembersESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMembersInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_corporation_members_limit ---

func TestGetCorporationMembersLimitHandler_MapsFields(t *testing.T) {
	fake := fakeCorporationESI{corporationMembersLimit: func(context.Context, int64) (int64, error) { return 500, nil }}

	handler := getCorporationMembersLimitHandler(openNarrow[corporationMembersLimitESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, GetCorporationMembersLimitInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || out.Limit != 500 {
		t.Errorf("out = %+v, want CorporationID=98000001 Limit=500", out)
	}
}

func TestGetCorporationMembersLimitHandler_UnknownCharacter(t *testing.T) {
	handler := getCorporationMembersLimitHandler(noNarrow[corporationMembersLimitESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, GetCorporationMembersLimitInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationMembersLimitHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationMembersLimit: func(context.Context, int64) (int64, error) { return 0, wantErr }}

	handler := getCorporationMembersLimitHandler(openNarrow[corporationMembersLimitESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, GetCorporationMembersLimitInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_member_titles ---

func TestListCorporationMemberTitlesHandler_MapsFields(t *testing.T) {
	fake := fakeCorporationESI{corporationMemberTitles: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdMembersTitlesGetInner, error) {
		return []esiapi.CorporationsCorporationIdMembersTitlesGetInner{{CharacterId: 100, Titles: []int64{1, 2}}}, nil
	}}

	handler := listCorporationMemberTitlesHandler(openNarrow[corporationMemberTitlesESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, ListCorporationMemberTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.MemberTitles) != 1 || out.MemberTitles[0].CharacterID != 100 || len(out.MemberTitles[0].TitleIDs) != 2 {
		t.Errorf("out = %+v, want CorporationID=98000001 with one MemberTitles entry CharacterID=100 and two titles", out)
	}
}

func TestListCorporationMemberTitlesHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationMemberTitlesHandler(noNarrow[corporationMemberTitlesESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTitlesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMemberTitlesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationMemberTitles: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdMembersTitlesGetInner, error) {
		return nil, wantErr
	}}

	handler := listCorporationMemberTitlesHandler(openNarrow[corporationMemberTitlesESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_member_tracking ---

func TestListCorporationMemberTrackingHandler_MapsFields(t *testing.T) {
	locationID := int64(60003760)
	fake := fakeCorporationESI{corporationMemberTracking: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error) {
		return []esiapi.CorporationsCorporationIdMembertrackingGetInner{{CharacterId: 100, LocationId: &locationID}}, nil
	}}

	handler := listCorporationMemberTrackingHandler("v1", openNarrow[corporationMemberTrackingESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Members) != 1 || out.Members[0].LocationID == nil || *out.Members[0].LocationID != locationID {
		t.Errorf("out = %+v, want CorporationID=98000001 with one member LocationID=%d", out, locationID)
	}
}

func TestListCorporationMemberTrackingHandler_ResolvesShipTypeName(t *testing.T) {
	fake := fakeCorporationESI{corporationMemberTracking: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error) {
		return []esiapi.CorporationsCorporationIdMembertrackingGetInner{{CharacterId: 1, ShipTypeId: new(int64(587))}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationMemberTrackingHandler("v1", openNarrow[corporationMemberTrackingESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Members) != 1 || out.Members[0].ShipTypeName != "Rifter" {
		t.Errorf("Members[0].ShipTypeName = %q, want %q", out.Members[0].ShipTypeName, "Rifter")
	}
}

func TestListCorporationMemberTrackingHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationMemberTrackingHandler("v1", noNarrow[corporationMemberTrackingESI](t), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMemberTrackingHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationMemberTracking: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error) {
		return nil, wantErr
	}}

	handler := listCorporationMemberTrackingHandler("v1", openNarrow[corporationMemberTrackingESI](fake), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMemberTrackingInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_roles ---

func TestListCorporationRolesHandler_MapsFields(t *testing.T) {
	fake := fakeCorporationESI{corporationRoles: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdRolesGetInner, error) {
		return []esiapi.CorporationsCorporationIdRolesGetInner{{CharacterId: 100, Roles: []string{"Director"}}}, nil
	}}

	handler := listCorporationRolesHandler(openNarrow[corporationRolesESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, ListCorporationRolesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Roles) != 1 || out.Roles[0].Roles[0] != "Director" {
		t.Errorf("out = %+v, want CorporationID=98000001 with one MemberRoles entry Roles=[Director]", out)
	}
}

func TestListCorporationRolesHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationRolesHandler(noNarrow[corporationRolesESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationRolesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationRolesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationRoles: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdRolesGetInner, error) {
		return nil, wantErr
	}}

	handler := listCorporationRolesHandler(openNarrow[corporationRolesESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationRolesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_roles_history ---

func TestListCorporationRolesHistoryHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	fake := fakeCorporationESI{corporationRolesHistory: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdRolesHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdRolesHistoryGetInner{{CharacterId: 100, IssuerId: 200, RoleType: "grant"}}, 4, nil
	}}

	handler := listCorporationRolesHistoryHandler(openNarrow[corporationRolesHistoryESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, ListCorporationRolesHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 || out.TotalPages != 4 || len(out.History) != 1 || out.History[0].RoleType != "grant" {
		t.Errorf("out = %+v, want Page=1 TotalPages=4 with one RoleChange RoleType=grant", out)
	}
}

func TestListCorporationRolesHistoryHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationRolesHistoryHandler(noNarrow[corporationRolesHistoryESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationRolesHistoryInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationRolesHistoryHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationRolesHistory: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdRolesHistoryGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationRolesHistoryHandler(openNarrow[corporationRolesHistoryESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationRolesHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_titles ---

func TestListCorporationTitlesHandler_MapsFields(t *testing.T) {
	name := "Officer"
	fake := fakeCorporationESI{corporationTitles: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdTitlesGetInner, error) {
		return []esiapi.CorporationsCorporationIdTitlesGetInner{{Name: &name}}, nil
	}}

	handler := listCorporationTitlesHandler(openNarrow[corporationTitlesESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, ListCorporationTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Titles) != 1 || out.Titles[0].Name == nil || *out.Titles[0].Name != name {
		t.Errorf("out = %+v, want CorporationID=98000001 with one title Name=%q", out, name)
	}
}

func TestListCorporationTitlesHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationTitlesHandler(noNarrow[corporationTitlesESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationTitlesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationTitlesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationTitles: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdTitlesGetInner, error) {
		return nil, wantErr
	}}

	handler := listCorporationTitlesHandler(openNarrow[corporationTitlesESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationTitlesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_corporation_divisions ---

func TestGetCorporationDivisionsHandler_MapsFields(t *testing.T) {
	hangarName := "Hangar 1"
	walletName := "Master Wallet"
	fake := fakeCorporationESI{corporationDivisions: func(context.Context, int64) (*esiapi.CorporationsCorporationIdDivisionsGet, error) {
		return &esiapi.CorporationsCorporationIdDivisionsGet{
			Hangar: []esiapi.CorporationsCorporationIdDivisionsGetHangarInner{{Name: &hangarName}},
			Wallet: []esiapi.CorporationsCorporationIdDivisionsGetWalletInner{{Name: &walletName}},
		}, nil
	}}

	handler := getCorporationDivisionsHandler(openNarrow[corporationDivisionsESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, GetCorporationDivisionsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Hangars) != 1 || out.Hangars[0].Name == nil || *out.Hangars[0].Name != hangarName ||
		len(out.Wallets) != 1 || out.Wallets[0].Name == nil || *out.Wallets[0].Name != walletName {
		t.Errorf("out = %+v, want CorporationID=98000001 with one hangar Name=%q and one wallet Name=%q", out, hangarName, walletName)
	}
}

func TestGetCorporationDivisionsHandler_UnknownCharacter(t *testing.T) {
	handler := getCorporationDivisionsHandler(noNarrow[corporationDivisionsESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, GetCorporationDivisionsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationDivisionsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationDivisions: func(context.Context, int64) (*esiapi.CorporationsCorporationIdDivisionsGet, error) {
		return nil, wantErr
	}}

	handler := getCorporationDivisionsHandler(openNarrow[corporationDivisionsESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, GetCorporationDivisionsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_shareholders ---

func TestListCorporationShareholdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	fake := fakeCorporationESI{corporationShareholders: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdShareholdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdShareholdersGetInner{{ShareholderId: 100, ShareholderType: "character", ShareCount: 1000}}, 2, nil
	}}

	handler := listCorporationShareholdersHandler(openNarrow[corporationShareholdersESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, ListCorporationShareholdersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 || out.TotalPages != 2 || len(out.Shareholders) != 1 || out.Shareholders[0].ShareCount != 1000 {
		t.Errorf("out = %+v, want Page=1 TotalPages=2 with one shareholder ShareCount=1000", out)
	}
}

func TestListCorporationShareholdersHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationShareholdersHandler(noNarrow[corporationShareholdersESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationShareholdersInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationShareholdersHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationShareholders: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdShareholdersGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationShareholdersHandler(openNarrow[corporationShareholdersESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationShareholdersInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_standings ---

func TestListCorporationStandingsHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	fake := fakeCorporationESI{corporationStandings: func(context.Context, int64, int32) ([]esiapi.CharactersCharacterIdStandingsGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdStandingsGetInner{{FromId: 500001, FromType: "npc_corp", Standing: 5.0}}, 3, nil
	}}

	handler := listCorporationStandingsHandler(openNarrow[corporationStandingsESI](fake), corpPilotStore(t))
	_, out, err := handler(context.Background(), nil, ListCorporationStandingsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 || out.TotalPages != 3 || len(out.Standings) != 1 || out.Standings[0].Standing != 5.0 || out.Standings[0].FromID != 500001 {
		t.Errorf("out = %+v, want Page=1 TotalPages=3 with one standing FromID=500001 Standing=5.0", out)
	}
}

func TestListCorporationStandingsHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationStandingsHandler(noNarrow[corporationStandingsESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationStandingsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationStandingsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationStandings: func(context.Context, int64, int32) ([]esiapi.CharactersCharacterIdStandingsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationStandingsHandler(openNarrow[corporationStandingsESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationStandingsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_blueprints ---

func TestListCorporationBlueprintsHandler_MapsFields(t *testing.T) {
	store := corpPilotStore(t)

	fake := fakeCorporationBlueprintsESI{corporationBlueprints: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdBlueprintsGetInner{{
			ItemId: 1001, LocationFlag: "Hangar", LocationId: 60003760,
			MaterialEfficiency: 10, Quantity: -1, Runs: -1, TimeEfficiency: 20, TypeId: 950,
		}}, 2, nil
	}}

	handler := listCorporationBlueprintsHandler("v1", openNarrow[corporationBlueprintsESI](fake), store, nil)
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
	store := corpPilotStore(t)

	fake := fakeCorporationBlueprintsESI{corporationBlueprints: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdBlueprintsGetInner{{ItemId: 1, TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationBlueprintsHandler("v1", openNarrow[corporationBlueprintsESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationBlueprintsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Blueprints) != 1 || out.Blueprints[0].TypeName != "Rifter" {
		t.Errorf("Blueprints[0].TypeName = %q, want %q", out.Blueprints[0].TypeName, "Rifter")
	}
}

func TestListCorporationBlueprintsHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationBlueprintsHandler("v1", noNarrow[corporationBlueprintsESI](t), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationBlueprintsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationBlueprintsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationBlueprintsESI{corporationBlueprints: func(_ context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationBlueprintsHandler("v1", openNarrow[corporationBlueprintsESI](fake), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationBlueprintsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_container_logs ---

func TestListCorporationContainerLogsHandler_MapsFields(t *testing.T) {
	loggedAt := time.Unix(0, 0)
	newBitmask := int64(1)
	oldBitmask := int64(2)
	passwordType := "Config"
	quantity := int64(5)
	typeID := int64(17366)
	fake := fakeCorporationESI{corporationContainerLogs: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdContainersLogsGetInner{{
			Action: "unlock", CharacterId: 100, ContainerId: 200, ContainerTypeId: 300,
			LocationFlag: "Hangar", LocationId: 60003760, LoggedAt: loggedAt,
			NewConfigBitmask: &newBitmask, OldConfigBitmask: &oldBitmask,
			PasswordType: &passwordType, Quantity: &quantity, TypeId: &typeID,
		}}, 3, nil
	}}

	handler := listCorporationContainerLogsHandler("v1", openNarrow[corporationContainerLogsESI](fake), corpPilotStore(t), nil)
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
	fake := fakeCorporationESI{corporationContainerLogs: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdContainersLogsGetInner{{ContainerTypeId: 588, TypeId: new(int64(587))}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := listCorporationContainerLogsHandler("v1", openNarrow[corporationContainerLogsESI](fake), corpPilotStore(t), nil)
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
	handler := listCorporationContainerLogsHandler("v1", noNarrow[corporationContainerLogsESI](t), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContainerLogsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContainerLogsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationContainerLogs: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationContainerLogsHandler("v1", openNarrow[corporationContainerLogsESI](fake), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationContainerLogsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_facilities ---

func TestListCorporationFacilitiesHandler_MapsFields(t *testing.T) {
	fake := fakeCorporationESI{corporationFacilities: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error) {
		return []esiapi.CorporationsCorporationIdFacilitiesGetInner{{FacilityId: 500, SystemId: 30000142, TypeId: 2502}}, nil
	}}

	handler := listCorporationFacilitiesHandler("v1", openNarrow[corporationFacilitiesESI](fake), corpPilotStore(t), nil)
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
	fake := fakeCorporationESI{corporationFacilities: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error) {
		return []esiapi.CorporationsCorporationIdFacilitiesGetInner{{FacilityId: 1, TypeId: 587}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationFacilitiesHandler("v1", openNarrow[corporationFacilitiesESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Facilities) != 1 || out.Facilities[0].TypeName != "Rifter" {
		t.Errorf("Facilities[0].TypeName = %q, want %q", out.Facilities[0].TypeName, "Rifter")
	}
}

func TestListCorporationFacilitiesHandler_ResolvesSystemName(t *testing.T) {
	fake := fakeCorporationESI{corporationFacilities: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error) {
		return []esiapi.CorporationsCorporationIdFacilitiesGetInner{{FacilityId: 1, SystemId: 30000142}}, nil
	}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationFacilitiesHandler("v1", openNarrow[corporationFacilitiesESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Facilities[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Facilities[0].SystemName, "Jita")
	}
}

func TestListCorporationFacilitiesHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationFacilitiesHandler("v1", noNarrow[corporationFacilitiesESI](t), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationFacilitiesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationFacilities: func(context.Context, int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error) {
		return nil, wantErr
	}}

	handler := listCorporationFacilitiesHandler("v1", openNarrow[corporationFacilitiesESI](fake), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_medals ---

func TestListCorporationMedalsHandler_MapsFields(t *testing.T) {
	createdAt := time.Unix(0, 0)
	fake := fakeCorporationESI{corporationMedals: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdMedalsGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdMedalsGetInner{{
			CreatedAt: createdAt, CreatorId: 100, Description: "For valor", MedalId: 1, Title: "Valor",
		}}, 2, nil
	}}

	handler := listCorporationMedalsHandler(openNarrow[corporationMedalsESI](fake), corpPilotStore(t))
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
	handler := listCorporationMedalsHandler(noNarrow[corporationMedalsESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMedalsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationMedals: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdMedalsGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationMedalsHandler(openNarrow[corporationMedalsESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_medals_issued ---

func TestListCorporationMedalsIssuedHandler_MapsFields(t *testing.T) {
	issuedAt := time.Unix(0, 0)
	fake := fakeCorporationESI{corporationMedalsIssued: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdMedalsIssuedGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdMedalsIssuedGetInner{{
			CharacterId: 100, IssuedAt: issuedAt, IssuerId: 200, MedalId: 1, Reason: "Great work", Status: "private",
		}}, 2, nil
	}}

	handler := listCorporationMedalsIssuedHandler(openNarrow[corporationMedalsIssuedESI](fake), corpPilotStore(t))
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
	handler := listCorporationMedalsIssuedHandler(noNarrow[corporationMedalsIssuedESI](t), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsIssuedInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationMedalsIssuedHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationMedalsIssued: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdMedalsIssuedGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationMedalsIssuedHandler(openNarrow[corporationMedalsIssuedESI](fake), corpPilotStore(t))
	_, _, err := handler(context.Background(), nil, ListCorporationMedalsIssuedInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_starbases ---

func TestListCorporationStarbasesHandler_MapsFields(t *testing.T) {
	moonID := int64(40000001)
	onlinedSince := time.Unix(1000, 0)
	reinforcedUntil := time.Unix(2000, 0)
	state := "online"
	unanchorAt := time.Unix(3000, 0)
	fake := fakeCorporationESI{corporationStarbases: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdStarbasesGetInner{{
			MoonId: &moonID, OnlinedSince: &onlinedSince, ReinforcedUntil: &reinforcedUntil,
			StarbaseId: 900, State: &state, SystemId: 30000142, TypeId: 16213, UnanchorAt: &unanchorAt,
		}}, 2, nil
	}}

	handler := listCorporationStarbasesHandler("v1", openNarrow[corporationStarbasesESI](fake), corpPilotStore(t), nil)
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
	fake := fakeCorporationESI{corporationStarbases: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdStarbasesGetInner{{StarbaseId: 1, TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationStarbasesHandler("v1", openNarrow[corporationStarbasesESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Starbases) != 1 || out.Starbases[0].TypeName != "Rifter" {
		t.Errorf("Starbases[0].TypeName = %q, want %q", out.Starbases[0].TypeName, "Rifter")
	}
}

func TestListCorporationStarbasesHandler_ResolvesLocationNames(t *testing.T) {
	moonID := int64(40009087)
	fake := fakeCorporationESI{corporationStarbases: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdStarbasesGetInner{{StarbaseId: 1, SystemId: 30000142, MoonId: &moonID}}, 1, nil
	}}

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

	handler := listCorporationStarbasesHandler("v1", openNarrow[corporationStarbasesESI](fake), corpPilotStore(t), nil)
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

func TestListCorporationStarbasesHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationStarbasesHandler("v1", noNarrow[corporationStarbasesESI](t), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationStarbasesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationStarbases: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationStarbasesHandler("v1", openNarrow[corporationStarbasesESI](fake), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_corporation_starbase ---

func TestGetCorporationStarbaseHandler_MapsFields(t *testing.T) {
	attackSecThreshold := 0.5
	attackStandingThreshold := 0.0
	var gotCorp, gotStarbase, gotSystem int64
	fake := fakeCorporationESI{corporationStarbaseDetail: func(_ context.Context, corporationID, starbaseID, systemID int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error) {
		gotCorp, gotStarbase, gotSystem = corporationID, starbaseID, systemID
		return &esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet{
			AllowAllianceMembers: true, AllowCorporationMembers: true, Anchor: "starbaseFuelTechnician",
			AttackIfAtWar: true, AttackIfOtherSecurityStatusDropping: true,
			AttackSecurityStatusThreshold: &attackSecThreshold, AttackStandingThreshold: &attackStandingThreshold,
			FuelBayTake: "starbaseFuelTechnician", FuelBayView: "corporationMembers",
			Fuels:                []esiapi.CorporationsCorporationIdStarbasesStarbaseIdGetFuelsInner{{Quantity: 100, TypeId: 4312}},
			Offline:              "starbaseFuelTechnician",
			Online:               "starbaseFuelTechnician",
			Unanchor:             "starbaseFuelTechnician",
			UseAllianceStandings: true,
		}, nil
	}}

	handler := getCorporationStarbaseHandler("v1", openNarrow[corporationStarbaseESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 7, CorporationID: 98000001, StarbaseID: 900, SystemID: 30000142})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCorp != 98000001 || gotStarbase != 900 || gotSystem != 30000142 {
		t.Errorf("ESI called with corporation=%d starbase=%d system=%d, want 98000001/900/30000142", gotCorp, gotStarbase, gotSystem)
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
	fake := fakeCorporationESI{corporationStarbaseDetail: func(context.Context, int64, int64, int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error) {
		return &esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet{
			Fuels: []esiapi.CorporationsCorporationIdStarbasesStarbaseIdGetFuelsInner{{TypeId: 587}},
		}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := getCorporationStarbaseHandler("v1", openNarrow[corporationStarbaseESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 7, CorporationID: 98000001, StarbaseID: 900, SystemID: 30000142})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Detail.Fuels) != 1 || out.Detail.Fuels[0].TypeName != "Rifter" {
		t.Errorf("Detail.Fuels[0].TypeName = %q, want %q", out.Detail.Fuels[0].TypeName, "Rifter")
	}
}

func TestGetCorporationStarbaseHandler_UnknownCharacter(t *testing.T) {
	handler := getCorporationStarbaseHandler("v1", noNarrow[corporationStarbaseESI](t), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 999, CorporationID: 98000001, StarbaseID: 900, SystemID: 30000142})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationStarbaseHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationStarbaseDetail: func(context.Context, int64, int64, int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error) {
		return nil, wantErr
	}}

	handler := getCorporationStarbaseHandler("v1", openNarrow[corporationStarbaseESI](fake), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, GetCorporationStarbaseInput{CharacterID: 7, CorporationID: 98000001, StarbaseID: 1000000000001, SystemID: 30000142})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_corporation_structures ---

func TestListCorporationStructuresHandler_MapsFields(t *testing.T) {
	fuelExpires := time.Unix(1000, 0)
	name := "My Keepstar"
	nextReinforceApply := time.Unix(2000, 0)
	nextReinforceHour := int64(18)
	reinforceHour := int64(19)
	stateTimerEnd := time.Unix(3000, 0)
	stateTimerStart := time.Unix(4000, 0)
	unanchorsAt := time.Unix(5000, 0)
	fake := fakeCorporationESI{corporationStructures: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdStructuresGetInner{{
			CorporationId: 98000001, FuelExpires: &fuelExpires, Name: &name,
			NextReinforceApply: &nextReinforceApply, NextReinforceHour: &nextReinforceHour, ProfileId: 101,
			ReinforceHour: &reinforceHour, Services: []esiapi.CorporationsCorporationIdStructuresGetInnerServicesInner{{Name: "Clone Bay", State: "online"}},
			State: "shield_vulnerable", StateTimerEnd: &stateTimerEnd, StateTimerStart: &stateTimerStart,
			StructureId: 1000000000001, SystemId: 30000142, TypeId: 35832, UnanchorsAt: &unanchorsAt,
		}}, 2, nil
	}}

	handler := listCorporationStructuresHandler("v1", openNarrow[corporationStructuresESI](fake), corpPilotStore(t), nil)
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
	fake := fakeCorporationESI{corporationStructures: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdStructuresGetInner{{StructureId: 1, TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationStructuresHandler("v1", openNarrow[corporationStructuresESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Structures) != 1 || out.Structures[0].TypeName != "Rifter" {
		t.Errorf("Structures[0].TypeName = %q, want %q", out.Structures[0].TypeName, "Rifter")
	}
}

func TestListCorporationStructuresHandler_ResolvesSystemName(t *testing.T) {
	fake := fakeCorporationESI{corporationStructures: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdStructuresGetInner{{StructureId: 1, SystemId: 30000142}}, 1, nil
	}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listCorporationStructuresHandler("v1", openNarrow[corporationStructuresESI](fake), corpPilotStore(t), nil)
	_, out, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Structures[0].SystemName != "Jita" {
		t.Errorf("SystemName = %q, want %q", out.Structures[0].SystemName, "Jita")
	}
}

func TestListCorporationStructuresHandler_UnknownCharacter(t *testing.T) {
	handler := listCorporationStructuresHandler("v1", noNarrow[corporationStructuresESI](t), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationStructuresHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeCorporationESI{corporationStructures: func(context.Context, int64, int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationStructuresHandler("v1", openNarrow[corporationStructuresESI](fake), corpPilotStore(t), nil)
	_, _, err := handler(context.Background(), nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// fakeCorporationBlueprintsESI fakes corporationBlueprintsESI; a nil corporationBlueprints fails as an unfaked call.
type fakeCorporationBlueprintsESI struct {
	corporationBlueprints func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error)
}

func (f fakeCorporationBlueprintsESI) CorporationBlueprints(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
	if f.corporationBlueprints == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationBlueprints(ctx, corporationID, page)
}

// Each handler reports a client that can't be opened, wrapping the cause.
func TestCorporationHandlers_PropagateOpenError(t *testing.T) {
	store := corpPilotStore(t)
	wantErr := errors.New("no refresh token")
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"list_corporation_members": func() error {
			_, _, err := listCorporationMembersHandler(failNarrow[corporationMembersESI](wantErr), store)(ctx, nil, ListCorporationMembersInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"get_corporation_members_limit": func() error {
			_, _, err := getCorporationMembersLimitHandler(failNarrow[corporationMembersLimitESI](wantErr), store)(ctx, nil, GetCorporationMembersLimitInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_member_titles": func() error {
			_, _, err := listCorporationMemberTitlesHandler(failNarrow[corporationMemberTitlesESI](wantErr), store)(ctx, nil, ListCorporationMemberTitlesInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_member_tracking": func() error {
			_, _, err := listCorporationMemberTrackingHandler("v1", failNarrow[corporationMemberTrackingESI](wantErr), store, nil)(ctx, nil, ListCorporationMemberTrackingInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_roles": func() error {
			_, _, err := listCorporationRolesHandler(failNarrow[corporationRolesESI](wantErr), store)(ctx, nil, ListCorporationRolesInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_roles_history": func() error {
			_, _, err := listCorporationRolesHistoryHandler(failNarrow[corporationRolesHistoryESI](wantErr), store)(ctx, nil, ListCorporationRolesHistoryInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_titles": func() error {
			_, _, err := listCorporationTitlesHandler(failNarrow[corporationTitlesESI](wantErr), store)(ctx, nil, ListCorporationTitlesInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"get_corporation_divisions": func() error {
			_, _, err := getCorporationDivisionsHandler(failNarrow[corporationDivisionsESI](wantErr), store)(ctx, nil, GetCorporationDivisionsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_shareholders": func() error {
			_, _, err := listCorporationShareholdersHandler(failNarrow[corporationShareholdersESI](wantErr), store)(ctx, nil, ListCorporationShareholdersInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_standings": func() error {
			_, _, err := listCorporationStandingsHandler(failNarrow[corporationStandingsESI](wantErr), store)(ctx, nil, ListCorporationStandingsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_blueprints": func() error {
			_, _, err := listCorporationBlueprintsHandler("v1", failNarrow[corporationBlueprintsESI](wantErr), store, nil)(ctx, nil, ListCorporationBlueprintsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_container_logs": func() error {
			_, _, err := listCorporationContainerLogsHandler("v1", failNarrow[corporationContainerLogsESI](wantErr), store, nil)(ctx, nil, ListCorporationContainerLogsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_facilities": func() error {
			_, _, err := listCorporationFacilitiesHandler("v1", failNarrow[corporationFacilitiesESI](wantErr), store, nil)(ctx, nil, ListCorporationFacilitiesInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_medals": func() error {
			_, _, err := listCorporationMedalsHandler(failNarrow[corporationMedalsESI](wantErr), store)(ctx, nil, ListCorporationMedalsInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_medals_issued": func() error {
			_, _, err := listCorporationMedalsIssuedHandler(failNarrow[corporationMedalsIssuedESI](wantErr), store)(ctx, nil, ListCorporationMedalsIssuedInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_starbases": func() error {
			_, _, err := listCorporationStarbasesHandler("v1", failNarrow[corporationStarbasesESI](wantErr), store, nil)(ctx, nil, ListCorporationStarbasesInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"get_corporation_starbase": func() error {
			_, _, err := getCorporationStarbaseHandler("v1", failNarrow[corporationStarbaseESI](wantErr), store, nil)(ctx, nil, GetCorporationStarbaseInput{CharacterID: 7, CorporationID: 98000001, StarbaseID: 900, SystemID: 30000142})
			return err
		},
		"list_corporation_structures": func() error {
			_, _, err := listCorporationStructuresHandler("v1", failNarrow[corporationStructuresESI](wantErr), store, nil)(ctx, nil, ListCorporationStructuresInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}
