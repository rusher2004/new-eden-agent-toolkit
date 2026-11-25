package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// fakeFwESI fakes every ESI call the faction warfare tools make; a test sets the
// function for the call it exercises.
type fakeFwESI struct {
	getCharacterFwStats          func(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error)
	getCorporationFwStats        func(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error)
	getFwLeaderboards            func(ctx context.Context) (*esiapi.FwLeaderboardsGet, error)
	getFwCharacterLeaderboards   func(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error)
	getFwCorporationLeaderboards func(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error)
	listFwStats                  func(ctx context.Context) ([]esiapi.FwStatsGetInner, error)
	listFwSystems                func(ctx context.Context) ([]esiapi.FwSystemsGetInner, error)
	listFwWars                   func(ctx context.Context) ([]esiapi.FwWarsGetInner, error)
}

func (f *fakeFwESI) CharacterFwStats(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
	return f.getCharacterFwStats(ctx)
}
func (f *fakeFwESI) CorporationFwStats(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
	return f.getCorporationFwStats(ctx, corporationID)
}
func (f *fakeFwESI) FwLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsGet, error) {
	return f.getFwLeaderboards(ctx)
}
func (f *fakeFwESI) FwCharacterLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error) {
	return f.getFwCharacterLeaderboards(ctx)
}
func (f *fakeFwESI) FwCorporationLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error) {
	return f.getFwCorporationLeaderboards(ctx)
}
func (f *fakeFwESI) FwStats(ctx context.Context) ([]esiapi.FwStatsGetInner, error) {
	return f.listFwStats(ctx)
}
func (f *fakeFwESI) FwSystems(ctx context.Context) ([]esiapi.FwSystemsGetInner, error) {
	return f.listFwSystems(ctx)
}
func (f *fakeFwESI) FwWars(ctx context.Context) ([]esiapi.FwWarsGetInner, error) {
	return f.listFwWars(ctx)
}

// openFwFake opens the fake as the small interface one authenticated tool needs.
func openFwFake[C any](c C) func(context.Context, int32) (C, error) {
	return func(context.Context, int32) (C, error) { return c, nil }
}

// fwPublic opens the fake as the small interface one public tool needs.
func fwPublic[C any](c C) func() C {
	return func() C { return c }
}

func TestGetCharacterFwStatsHandler_PropagatesOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("keychain locked")
	open := func(context.Context, int32) (fwCharacterESI, error) { return nil, wantErr }
	handler := getCharacterFwStatsHandler("v1", open, store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterFwStatsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationFwStatsHandler_PropagatesOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("keychain locked")
	open := func(context.Context, int32) (fwCorporationESI, error) { return nil, wantErr }
	handler := getCorporationFwStatsHandler("v1", open, store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCharacterFwStatsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeFwESI{}
	currentRank := int64(5)
	highestRank := int64(3)
	factionID := int64(500001)
	enlistedOn := time.Unix(1000, 0)
	fake.getCharacterFwStats = func(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		return &esiapi.CharactersCharacterIdFwStatsGet{
			CurrentRank: &currentRank, HighestRank: &highestRank, FactionId: &factionID, EnlistedOn: &enlistedOn,
			Kills:         esiapi.CharactersCharacterIdFwStatsGetKills{LastWeek: 1, Total: 2, Yesterday: 3},
			VictoryPoints: esiapi.CharactersCharacterIdFwStatsGetVictoryPoints{LastWeek: 4, Total: 5, Yesterday: 6},
		}, nil
	}

	handler := getCharacterFwStatsHandler("v1", openFwFake[fwCharacterESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterFwStatsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 ||
		out.CurrentRank == nil || *out.CurrentRank != currentRank ||
		out.HighestRank == nil || *out.HighestRank != highestRank ||
		out.FactionID == nil || *out.FactionID != factionID ||
		out.EnlistedOn == nil || !out.EnlistedOn.Equal(enlistedOn) ||
		out.Kills.LastWeek != 1 || out.Kills.Total != 2 || out.Kills.Yesterday != 3 ||
		out.VictoryPoints.LastWeek != 4 || out.VictoryPoints.Total != 5 || out.VictoryPoints.Yesterday != 6 {
		t.Errorf("out = %+v, want CharacterID=7 CurrentRank=%d HighestRank=%d FactionID=%d EnlistedOn=%v Kills={1,2,3} VictoryPoints={4,5,6}",
			out, currentRank, highestRank, factionID, enlistedOn)
	}
}

func TestGetCharacterFwStatsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeFwESI{}
	fake.getCharacterFwStats = func(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		t.Fatal("getCharacterFwStats should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterFwStatsHandler("v1", openFwFake[fwCharacterESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterFwStatsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCorporationFwStatsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeFwESI{}
	factionID := int64(500001)
	pilots := int64(42)
	enlistedOn := time.Unix(2000, 0)
	fake.getCorporationFwStats = func(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		return &esiapi.CorporationsCorporationIdFwStatsGet{
			FactionId: &factionID, Pilots: &pilots, EnlistedOn: &enlistedOn,
			Kills:         esiapi.CorporationsCorporationIdFwStatsGetKills{LastWeek: 10, Total: 20, Yesterday: 30},
			VictoryPoints: esiapi.CorporationsCorporationIdFwStatsGetVictoryPoints{LastWeek: 40, Total: 50, Yesterday: 60},
		}, nil
	}

	handler := getCorporationFwStatsHandler("v1", openFwFake[fwCorporationESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 ||
		out.FactionID == nil || *out.FactionID != factionID ||
		out.Pilots == nil || *out.Pilots != pilots ||
		out.EnlistedOn == nil || !out.EnlistedOn.Equal(enlistedOn) ||
		out.Kills.LastWeek != 10 || out.Kills.Total != 20 || out.Kills.Yesterday != 30 ||
		out.VictoryPoints.LastWeek != 40 || out.VictoryPoints.Total != 50 || out.VictoryPoints.Yesterday != 60 {
		t.Errorf("out = %+v, want CharacterID=7 CorporationID=98000001 FactionID=%d Pilots=%d EnlistedOn=%v Kills={10,20,30} VictoryPoints={40,50,60}",
			out, factionID, pilots, enlistedOn)
	}
}

func TestGetCorporationFwStatsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &fakeFwESI{}
	fake.getCorporationFwStats = func(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		t.Fatal("getCorporationFwStats should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationFwStatsHandler("v1", openFwFake[fwCorporationESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetFwLeaderboardsHandler_MapsFields(t *testing.T) {
	fake := &fakeFwESI{}

	killsActiveTotalFaction, killsActiveTotalAmount := int64(1001), int64(101)
	killsLastWeekFaction, killsLastWeekAmount := int64(1002), int64(102)
	killsYesterdayFaction, killsYesterdayAmount := int64(1003), int64(103)
	vpActiveTotalFaction, vpActiveTotalAmount := int64(1004), int64(104)
	vpLastWeekFaction, vpLastWeekAmount := int64(1005), int64(105)
	vpYesterdayFaction, vpYesterdayAmount := int64(1006), int64(106)

	fake.getFwLeaderboards = func(ctx context.Context) (*esiapi.FwLeaderboardsGet, error) {
		return &esiapi.FwLeaderboardsGet{
			Kills: esiapi.FwLeaderboardsGetKills{
				ActiveTotal: []esiapi.FwLeaderboardsGetKillsActiveTotalInner{{FactionId: &killsActiveTotalFaction, Amount: &killsActiveTotalAmount}},
				LastWeek:    []esiapi.FwLeaderboardsGetKillsLastWeekInner{{FactionId: &killsLastWeekFaction, Amount: &killsLastWeekAmount}},
				Yesterday:   []esiapi.FwLeaderboardsGetKillsYesterdayInner{{FactionId: &killsYesterdayFaction, Amount: &killsYesterdayAmount}},
			},
			VictoryPoints: esiapi.FwLeaderboardsGetVictoryPoints{
				ActiveTotal: []esiapi.FwLeaderboardsGetVictoryPointsActiveTotalInner{{FactionId: &vpActiveTotalFaction, Amount: &vpActiveTotalAmount}},
				LastWeek:    []esiapi.FwLeaderboardsGetVictoryPointsLastWeekInner{{FactionId: &vpLastWeekFaction, Amount: &vpLastWeekAmount}},
				Yesterday:   []esiapi.FwLeaderboardsGetVictoryPointsYesterdayInner{{FactionId: &vpYesterdayFaction, Amount: &vpYesterdayAmount}},
			},
		}, nil
	}

	handler := getFwLeaderboardsHandler("v1", fwPublic[fwLeaderboardsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetFwLeaderboardsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	checkEntry := func(t *testing.T, label string, entries []LeaderboardFactionEntry, wantFaction, wantAmount int64) {
		t.Helper()
		if len(entries) != 1 || entries[0].FactionID == nil || *entries[0].FactionID != wantFaction || entries[0].Amount == nil || *entries[0].Amount != wantAmount {
			t.Errorf("%s = %+v, want one entry FactionID=%d Amount=%d", label, entries, wantFaction, wantAmount)
		}
	}
	checkEntry(t, "Kills.ActiveTotal", out.Kills.ActiveTotal, killsActiveTotalFaction, killsActiveTotalAmount)
	checkEntry(t, "Kills.LastWeek", out.Kills.LastWeek, killsLastWeekFaction, killsLastWeekAmount)
	checkEntry(t, "Kills.Yesterday", out.Kills.Yesterday, killsYesterdayFaction, killsYesterdayAmount)
	checkEntry(t, "VictoryPoints.ActiveTotal", out.VictoryPoints.ActiveTotal, vpActiveTotalFaction, vpActiveTotalAmount)
	checkEntry(t, "VictoryPoints.LastWeek", out.VictoryPoints.LastWeek, vpLastWeekFaction, vpLastWeekAmount)
	checkEntry(t, "VictoryPoints.Yesterday", out.VictoryPoints.Yesterday, vpYesterdayFaction, vpYesterdayAmount)
}

func TestGetFwCharacterLeaderboardsHandler_MapsFields(t *testing.T) {
	fake := &fakeFwESI{}

	killsActiveTotalChar, killsActiveTotalAmount := int64(2001), int64(201)
	killsLastWeekChar, killsLastWeekAmount := int64(2002), int64(202)
	killsYesterdayChar, killsYesterdayAmount := int64(2003), int64(203)
	vpActiveTotalChar, vpActiveTotalAmount := int64(2004), int64(204)
	vpLastWeekChar, vpLastWeekAmount := int64(2005), int64(205)
	vpYesterdayChar, vpYesterdayAmount := int64(2006), int64(206)

	fake.getFwCharacterLeaderboards = func(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error) {
		return &esiapi.FwLeaderboardsCharactersGet{
			Kills: esiapi.FwLeaderboardsCharactersGetKills{
				ActiveTotal: []esiapi.FwLeaderboardsCharactersGetKillsActiveTotalInner{{CharacterId: &killsActiveTotalChar, Amount: &killsActiveTotalAmount}},
				LastWeek:    []esiapi.FwLeaderboardsCharactersGetKillsLastWeekInner{{CharacterId: &killsLastWeekChar, Amount: &killsLastWeekAmount}},
				Yesterday:   []esiapi.FwLeaderboardsCharactersGetKillsYesterdayInner{{CharacterId: &killsYesterdayChar, Amount: &killsYesterdayAmount}},
			},
			VictoryPoints: esiapi.FwLeaderboardsCharactersGetVictoryPoints{
				ActiveTotal: []esiapi.FwLeaderboardsCharactersGetVictoryPointsActiveTotalInner{{CharacterId: &vpActiveTotalChar, Amount: &vpActiveTotalAmount}},
				LastWeek:    []esiapi.FwLeaderboardsCharactersGetVictoryPointsLastWeekInner{{CharacterId: &vpLastWeekChar, Amount: &vpLastWeekAmount}},
				Yesterday:   []esiapi.FwLeaderboardsCharactersGetVictoryPointsYesterdayInner{{CharacterId: &vpYesterdayChar, Amount: &vpYesterdayAmount}},
			},
		}, nil
	}

	handler := getFwCharacterLeaderboardsHandler(fwPublic[fwCharacterLeaderboardsESI](fake))
	_, out, err := handler(context.Background(), nil, GetFwCharacterLeaderboardsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	checkEntry := func(t *testing.T, label string, entries []LeaderboardCharacterEntry, wantChar, wantAmount int64) {
		t.Helper()
		if len(entries) != 1 || entries[0].CharacterID == nil || *entries[0].CharacterID != wantChar || entries[0].Amount == nil || *entries[0].Amount != wantAmount {
			t.Errorf("%s = %+v, want one entry CharacterID=%d Amount=%d", label, entries, wantChar, wantAmount)
		}
	}
	checkEntry(t, "Kills.ActiveTotal", out.Kills.ActiveTotal, killsActiveTotalChar, killsActiveTotalAmount)
	checkEntry(t, "Kills.LastWeek", out.Kills.LastWeek, killsLastWeekChar, killsLastWeekAmount)
	checkEntry(t, "Kills.Yesterday", out.Kills.Yesterday, killsYesterdayChar, killsYesterdayAmount)
	checkEntry(t, "VictoryPoints.ActiveTotal", out.VictoryPoints.ActiveTotal, vpActiveTotalChar, vpActiveTotalAmount)
	checkEntry(t, "VictoryPoints.LastWeek", out.VictoryPoints.LastWeek, vpLastWeekChar, vpLastWeekAmount)
	checkEntry(t, "VictoryPoints.Yesterday", out.VictoryPoints.Yesterday, vpYesterdayChar, vpYesterdayAmount)
}

func TestGetFwCorporationLeaderboardsHandler_MapsFields(t *testing.T) {
	fake := &fakeFwESI{}

	killsActiveTotalCorp, killsActiveTotalAmount := int64(3001), int64(301)
	killsLastWeekCorp, killsLastWeekAmount := int64(3002), int64(302)
	killsYesterdayCorp, killsYesterdayAmount := int64(3003), int64(303)
	vpActiveTotalCorp, vpActiveTotalAmount := int64(3004), int64(304)
	vpLastWeekCorp, vpLastWeekAmount := int64(3005), int64(305)
	vpYesterdayCorp, vpYesterdayAmount := int64(3006), int64(306)

	fake.getFwCorporationLeaderboards = func(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error) {
		return &esiapi.FwLeaderboardsCorporationsGet{
			Kills: esiapi.FwLeaderboardsCorporationsGetKills{
				ActiveTotal: []esiapi.FwLeaderboardsCorporationsGetKillsActiveTotalInner{{CorporationId: &killsActiveTotalCorp, Amount: &killsActiveTotalAmount}},
				LastWeek:    []esiapi.FwLeaderboardsCorporationsGetKillsLastWeekInner{{CorporationId: &killsLastWeekCorp, Amount: &killsLastWeekAmount}},
				Yesterday:   []esiapi.FwLeaderboardsCorporationsGetKillsYesterdayInner{{CorporationId: &killsYesterdayCorp, Amount: &killsYesterdayAmount}},
			},
			VictoryPoints: esiapi.FwLeaderboardsCorporationsGetVictoryPoints{
				ActiveTotal: []esiapi.FwLeaderboardsCorporationsGetVictoryPointsActiveTotalInner{{CorporationId: &vpActiveTotalCorp, Amount: &vpActiveTotalAmount}},
				LastWeek:    []esiapi.FwLeaderboardsCorporationsGetVictoryPointsLastWeekInner{{CorporationId: &vpLastWeekCorp, Amount: &vpLastWeekAmount}},
				Yesterday:   []esiapi.FwLeaderboardsCorporationsGetVictoryPointsYesterdayInner{{CorporationId: &vpYesterdayCorp, Amount: &vpYesterdayAmount}},
			},
		}, nil
	}

	handler := getFwCorporationLeaderboardsHandler(fwPublic[fwCorporationLeaderboardsESI](fake))
	_, out, err := handler(context.Background(), nil, GetFwCorporationLeaderboardsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	checkEntry := func(t *testing.T, label string, entries []LeaderboardCorporationEntry, wantCorp, wantAmount int64) {
		t.Helper()
		if len(entries) != 1 || entries[0].CorporationID == nil || *entries[0].CorporationID != wantCorp || entries[0].Amount == nil || *entries[0].Amount != wantAmount {
			t.Errorf("%s = %+v, want one entry CorporationID=%d Amount=%d", label, entries, wantCorp, wantAmount)
		}
	}
	checkEntry(t, "Kills.ActiveTotal", out.Kills.ActiveTotal, killsActiveTotalCorp, killsActiveTotalAmount)
	checkEntry(t, "Kills.LastWeek", out.Kills.LastWeek, killsLastWeekCorp, killsLastWeekAmount)
	checkEntry(t, "Kills.Yesterday", out.Kills.Yesterday, killsYesterdayCorp, killsYesterdayAmount)
	checkEntry(t, "VictoryPoints.ActiveTotal", out.VictoryPoints.ActiveTotal, vpActiveTotalCorp, vpActiveTotalAmount)
	checkEntry(t, "VictoryPoints.LastWeek", out.VictoryPoints.LastWeek, vpLastWeekCorp, vpLastWeekAmount)
	checkEntry(t, "VictoryPoints.Yesterday", out.VictoryPoints.Yesterday, vpYesterdayCorp, vpYesterdayAmount)
}

func TestListFwStatsHandler_MapsFields(t *testing.T) {
	fake := &fakeFwESI{}
	fake.listFwStats = func(ctx context.Context) ([]esiapi.FwStatsGetInner, error) {
		return []esiapi.FwStatsGetInner{{
			FactionId: 500001, Pilots: 1000, SystemsControlled: 25,
			Kills:         esiapi.FwStatsGetInnerKills{LastWeek: 1, Total: 2, Yesterday: 3},
			VictoryPoints: esiapi.FwStatsGetInnerVictoryPoints{LastWeek: 4, Total: 5, Yesterday: 6},
		}}, nil
	}

	handler := listFwStatsHandler("v1", fwPublic[fwStatsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListFwStatsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Stats) != 1 {
		t.Fatalf("out = %+v, want one stats entry", out)
	}
	s := out.Stats[0]
	if s.FactionID != 500001 || s.Pilots != 1000 || s.SystemsControlled != 25 ||
		s.Kills.LastWeek != 1 || s.Kills.Total != 2 || s.Kills.Yesterday != 3 ||
		s.VictoryPoints.LastWeek != 4 || s.VictoryPoints.Total != 5 || s.VictoryPoints.Yesterday != 6 {
		t.Errorf("stats = %+v, want FactionID=500001 Pilots=1000 SystemsControlled=25 Kills={1,2,3} VictoryPoints={4,5,6}", s)
	}
}

func TestListFwSystemsHandler_MapsFields(t *testing.T) {
	fake := &fakeFwESI{}
	fake.listFwSystems = func(ctx context.Context) ([]esiapi.FwSystemsGetInner, error) {
		return []esiapi.FwSystemsGetInner{{
			Contested: "contested", OccupierFactionId: 500001, OwnerFactionId: 500002,
			SolarSystemId: 30000142, VictoryPoints: 100, VictoryPointsThreshold: 500,
		}}, nil
	}

	handler := listFwSystemsHandler("v1", fwPublic[fwSystemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListFwSystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Systems) != 1 {
		t.Fatalf("out = %+v, want one system entry", out)
	}
	s := out.Systems[0]
	if s.Contested != "contested" || s.OccupierFactionID != 500001 || s.OwnerFactionID != 500002 ||
		s.SolarSystemID != 30000142 || s.VictoryPoints != 100 || s.VictoryPointsThreshold != 500 {
		t.Errorf("system = %+v, want Contested=contested OccupierFactionID=500001 OwnerFactionID=500002 SolarSystemID=30000142 VictoryPoints=100 VictoryPointsThreshold=500", s)
	}
}

func TestListFwWarsHandler_MapsFields(t *testing.T) {
	fake := &fakeFwESI{}
	fake.listFwWars = func(ctx context.Context) ([]esiapi.FwWarsGetInner, error) {
		return []esiapi.FwWarsGetInner{{AgainstId: 500001, FactionId: 500002}}, nil
	}

	handler := listFwWarsHandler("v1", fwPublic[fwWarsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListFwWarsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Wars) != 1 {
		t.Fatalf("out = %+v, want one war entry", out)
	}
	w := out.Wars[0]
	if w.AgainstID != 500001 || w.FactionID != 500002 {
		t.Errorf("war = %+v, want AgainstID=500001 FactionID=500002", w)
	}
}

func TestGetCharacterFwStatsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.getCharacterFwStats = func(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		return nil, wantErr
	}

	handler := getCharacterFwStatsHandler("v1", openFwFake[fwCharacterESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCharacterFwStatsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetCorporationFwStatsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.getCorporationFwStats = func(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		return nil, wantErr
	}

	handler := getCorporationFwStatsHandler("v1", openFwFake[fwCorporationESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetFwLeaderboardsHandler_PropagatesError(t *testing.T) {
	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.getFwLeaderboards = func(ctx context.Context) (*esiapi.FwLeaderboardsGet, error) {
		return nil, wantErr
	}

	handler := getFwLeaderboardsHandler("v1", fwPublic[fwLeaderboardsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, GetFwLeaderboardsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetFwCharacterLeaderboardsHandler_PropagatesError(t *testing.T) {
	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.getFwCharacterLeaderboards = func(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error) {
		return nil, wantErr
	}

	handler := getFwCharacterLeaderboardsHandler(fwPublic[fwCharacterLeaderboardsESI](fake))
	_, _, err := handler(context.Background(), nil, GetFwCharacterLeaderboardsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetFwCorporationLeaderboardsHandler_PropagatesError(t *testing.T) {
	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.getFwCorporationLeaderboards = func(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error) {
		return nil, wantErr
	}

	handler := getFwCorporationLeaderboardsHandler(fwPublic[fwCorporationLeaderboardsESI](fake))
	_, _, err := handler(context.Background(), nil, GetFwCorporationLeaderboardsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFwStatsHandler_PropagatesError(t *testing.T) {
	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.listFwStats = func(ctx context.Context) ([]esiapi.FwStatsGetInner, error) {
		return nil, wantErr
	}

	handler := listFwStatsHandler("v1", fwPublic[fwStatsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListFwStatsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFwSystemsHandler_PropagatesError(t *testing.T) {
	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.listFwSystems = func(ctx context.Context) ([]esiapi.FwSystemsGetInner, error) {
		return nil, wantErr
	}

	handler := listFwSystemsHandler("v1", fwPublic[fwSystemsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListFwSystemsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFwWarsHandler_PropagatesError(t *testing.T) {
	fake := &fakeFwESI{}
	wantErr := errors.New("upstream failure")
	fake.listFwWars = func(ctx context.Context) ([]esiapi.FwWarsGetInner, error) {
		return nil, wantErr
	}

	handler := listFwWarsHandler("v1", fwPublic[fwWarsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListFwWarsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFwSystemsHandler_ResolvesSystemName(t *testing.T) {
	fake := &fakeFwESI{}
	fake.listFwSystems = func(ctx context.Context) ([]esiapi.FwSystemsGetInner, error) {
		return []esiapi.FwSystemsGetInner{{SolarSystemId: 30000142}}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listFwSystemsHandler("v1", fwPublic[fwSystemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListFwSystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Systems[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Systems[0].SolarSystemName, "Jita")
	}
}

func TestGetCharacterFwStatsHandler_ResolvesFactionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeFwESI{}
	factionID := int64(500001)
	fake.getCharacterFwStats = func(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		return &esiapi.CharactersCharacterIdFwStatsGet{FactionId: &factionID}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getCharacterFwStatsHandler("v1", openFwFake[fwCharacterESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCharacterFwStatsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetCorporationFwStatsHandler_ResolvesFactionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := &fakeFwESI{}
	factionID := int64(500001)
	fake.getCorporationFwStats = func(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		return &esiapi.CorporationsCorporationIdFwStatsGet{FactionId: &factionID}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getCorporationFwStatsHandler("v1", openFwFake[fwCorporationESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetFwLeaderboardsHandler_ResolvesFactionNames(t *testing.T) {
	fake := &fakeFwESI{}
	killsFactionID := int64(500001)
	killsAmount := int64(42)
	vpFactionID := int64(500002)
	vpAmount := int64(7)
	fake.getFwLeaderboards = func(ctx context.Context) (*esiapi.FwLeaderboardsGet, error) {
		return &esiapi.FwLeaderboardsGet{
			Kills: esiapi.FwLeaderboardsGetKills{
				ActiveTotal: []esiapi.FwLeaderboardsGetKillsActiveTotalInner{{FactionId: &killsFactionID, Amount: &killsAmount}},
			},
			VictoryPoints: esiapi.FwLeaderboardsGetVictoryPoints{
				Yesterday: []esiapi.FwLeaderboardsGetVictoryPointsYesterdayInner{{FactionId: &vpFactionID, Amount: &vpAmount}},
			},
		}, nil
	}

	callCount := 0
	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		callCount++
		return map[int64]string{500001: "Caldari State", 500002: "Gallente Federation"}
	}

	handler := getFwLeaderboardsHandler("v1", fwPublic[fwLeaderboardsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, GetFwLeaderboardsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Kills.ActiveTotal) != 1 || out.Kills.ActiveTotal[0].FactionName != "Caldari State" {
		t.Errorf("Kills.ActiveTotal = %+v, want one entry FactionName=%q", out.Kills.ActiveTotal, "Caldari State")
	}
	if len(out.VictoryPoints.Yesterday) != 1 || out.VictoryPoints.Yesterday[0].FactionName != "Gallente Federation" {
		t.Errorf("VictoryPoints.Yesterday = %+v, want one entry FactionName=%q", out.VictoryPoints.Yesterday, "Gallente Federation")
	}
	if callCount != 1 {
		t.Errorf("resolveFactionNames called %d times, want 1 (all six leaderboard slices should share one combined call)", callCount)
	}
}

func TestListFwStatsHandler_ResolvesFactionName(t *testing.T) {
	fake := &fakeFwESI{}
	fake.listFwStats = func(ctx context.Context) ([]esiapi.FwStatsGetInner, error) {
		return []esiapi.FwStatsGetInner{{FactionId: 500001}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := listFwStatsHandler("v1", fwPublic[fwStatsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListFwStatsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Stats) != 1 || out.Stats[0].FactionName != "Caldari State" {
		t.Errorf("Stats = %+v, want one entry FactionName=%q", out.Stats, "Caldari State")
	}
}

func TestListFwSystemsHandler_ResolvesOccupierAndOwnerFactionNames(t *testing.T) {
	fake := &fakeFwESI{}
	fake.listFwSystems = func(ctx context.Context) ([]esiapi.FwSystemsGetInner, error) {
		return []esiapi.FwSystemsGetInner{{OccupierFactionId: 500001, OwnerFactionId: 500002, SolarSystemId: 30000142}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State", 500002: "Gallente Federation"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listFwSystemsHandler("v1", fwPublic[fwSystemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListFwSystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Systems) != 1 || out.Systems[0].OccupierFactionName != "Caldari State" || out.Systems[0].OwnerFactionName != "Gallente Federation" {
		t.Errorf("Systems = %+v, want one entry OccupierFactionName=%q OwnerFactionName=%q", out.Systems, "Caldari State", "Gallente Federation")
	}
}

func TestListFwWarsHandler_ResolvesFactionName(t *testing.T) {
	fake := &fakeFwESI{}
	fake.listFwWars = func(ctx context.Context) ([]esiapi.FwWarsGetInner, error) {
		return []esiapi.FwWarsGetInner{{AgainstId: 500002, FactionId: 500001}}, nil
	}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State", 500002: "Minmatar Republic"}
	}

	handler := listFwWarsHandler("v1", fwPublic[fwWarsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListFwWarsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Wars) != 1 || out.Wars[0].FactionName != "Caldari State" || out.Wars[0].AgainstFactionName != "Minmatar Republic" {
		t.Errorf("Wars = %+v, want one entry FactionName=%q AgainstFactionName=%q", out.Wars, "Caldari State", "Minmatar Republic")
	}
}
