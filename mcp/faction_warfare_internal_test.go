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

func TestGetCharacterFwStatsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterFwStats
	t.Cleanup(func() { getCharacterFwStats = orig })
	currentRank := int64(5)
	highestRank := int64(3)
	factionID := int64(500001)
	enlistedOn := time.Unix(1000, 0)
	getCharacterFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		return &esiapi.CharactersCharacterIdFwStatsGet{
			CurrentRank: &currentRank, HighestRank: &highestRank, FactionId: &factionID, EnlistedOn: &enlistedOn,
			Kills:         esiapi.CharactersCharacterIdFwStatsGetKills{LastWeek: 1, Total: 2, Yesterday: 3},
			VictoryPoints: esiapi.CharactersCharacterIdFwStatsGetVictoryPoints{LastWeek: 4, Total: 5, Yesterday: 6},
		}, nil
	}

	handler := getCharacterFwStatsHandler("v1", "client-id", store)
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

	orig := getCharacterFwStats
	t.Cleanup(func() { getCharacterFwStats = orig })
	getCharacterFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		t.Fatal("getCharacterFwStats should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterFwStatsHandler("v1", "client-id", store)
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

	orig := getCorporationFwStats
	t.Cleanup(func() { getCorporationFwStats = orig })
	factionID := int64(500001)
	pilots := int64(42)
	enlistedOn := time.Unix(2000, 0)
	getCorporationFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		return &esiapi.CorporationsCorporationIdFwStatsGet{
			FactionId: &factionID, Pilots: &pilots, EnlistedOn: &enlistedOn,
			Kills:         esiapi.CorporationsCorporationIdFwStatsGetKills{LastWeek: 10, Total: 20, Yesterday: 30},
			VictoryPoints: esiapi.CorporationsCorporationIdFwStatsGetVictoryPoints{LastWeek: 40, Total: 50, Yesterday: 60},
		}, nil
	}

	handler := getCorporationFwStatsHandler("v1", "client-id", store)
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

	orig := getCorporationFwStats
	t.Cleanup(func() { getCorporationFwStats = orig })
	getCorporationFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		t.Fatal("getCorporationFwStats should not be called for an unknown character")
		return nil, nil
	}

	handler := getCorporationFwStatsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetFwLeaderboardsHandler_MapsFields(t *testing.T) {
	orig := getFwLeaderboards
	t.Cleanup(func() { getFwLeaderboards = orig })

	killsActiveTotalFaction, killsActiveTotalAmount := int64(1001), int64(101)
	killsLastWeekFaction, killsLastWeekAmount := int64(1002), int64(102)
	killsYesterdayFaction, killsYesterdayAmount := int64(1003), int64(103)
	vpActiveTotalFaction, vpActiveTotalAmount := int64(1004), int64(104)
	vpLastWeekFaction, vpLastWeekAmount := int64(1005), int64(105)
	vpYesterdayFaction, vpYesterdayAmount := int64(1006), int64(106)

	getFwLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsGet, error) {
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

	handler := getFwLeaderboardsHandler("v1")
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
	orig := getFwCharacterLeaderboards
	t.Cleanup(func() { getFwCharacterLeaderboards = orig })

	killsActiveTotalChar, killsActiveTotalAmount := int64(2001), int64(201)
	killsLastWeekChar, killsLastWeekAmount := int64(2002), int64(202)
	killsYesterdayChar, killsYesterdayAmount := int64(2003), int64(203)
	vpActiveTotalChar, vpActiveTotalAmount := int64(2004), int64(204)
	vpLastWeekChar, vpLastWeekAmount := int64(2005), int64(205)
	vpYesterdayChar, vpYesterdayAmount := int64(2006), int64(206)

	getFwCharacterLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCharactersGet, error) {
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

	handler := getFwCharacterLeaderboardsHandler("v1")
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
	orig := getFwCorporationLeaderboards
	t.Cleanup(func() { getFwCorporationLeaderboards = orig })

	killsActiveTotalCorp, killsActiveTotalAmount := int64(3001), int64(301)
	killsLastWeekCorp, killsLastWeekAmount := int64(3002), int64(302)
	killsYesterdayCorp, killsYesterdayAmount := int64(3003), int64(303)
	vpActiveTotalCorp, vpActiveTotalAmount := int64(3004), int64(304)
	vpLastWeekCorp, vpLastWeekAmount := int64(3005), int64(305)
	vpYesterdayCorp, vpYesterdayAmount := int64(3006), int64(306)

	getFwCorporationLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCorporationsGet, error) {
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

	handler := getFwCorporationLeaderboardsHandler("v1")
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
	orig := listFwStats
	t.Cleanup(func() { listFwStats = orig })
	listFwStats = func(ctx context.Context, userAgent string) ([]esiapi.FwStatsGetInner, error) {
		return []esiapi.FwStatsGetInner{{
			FactionId: 500001, Pilots: 1000, SystemsControlled: 25,
			Kills:         esiapi.FwStatsGetInnerKills{LastWeek: 1, Total: 2, Yesterday: 3},
			VictoryPoints: esiapi.FwStatsGetInnerVictoryPoints{LastWeek: 4, Total: 5, Yesterday: 6},
		}}, nil
	}

	handler := listFwStatsHandler("v1")
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
	orig := listFwSystems
	t.Cleanup(func() { listFwSystems = orig })
	listFwSystems = func(ctx context.Context, userAgent string) ([]esiapi.FwSystemsGetInner, error) {
		return []esiapi.FwSystemsGetInner{{
			Contested: "contested", OccupierFactionId: 500001, OwnerFactionId: 500002,
			SolarSystemId: 30000142, VictoryPoints: 100, VictoryPointsThreshold: 500,
		}}, nil
	}

	handler := listFwSystemsHandler("v1")
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
	orig := listFwWars
	t.Cleanup(func() { listFwWars = orig })
	listFwWars = func(ctx context.Context, userAgent string) ([]esiapi.FwWarsGetInner, error) {
		return []esiapi.FwWarsGetInner{{AgainstId: 500001, FactionId: 500002}}, nil
	}

	handler := listFwWarsHandler("v1")
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

	orig := getCharacterFwStats
	t.Cleanup(func() { getCharacterFwStats = orig })
	wantErr := errors.New("upstream failure")
	getCharacterFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
		return nil, wantErr
	}

	handler := getCharacterFwStatsHandler("v1", "client-id", store)
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

	orig := getCorporationFwStats
	t.Cleanup(func() { getCorporationFwStats = orig })
	wantErr := errors.New("upstream failure")
	getCorporationFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
		return nil, wantErr
	}

	handler := getCorporationFwStatsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCorporationFwStatsInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetFwLeaderboardsHandler_PropagatesError(t *testing.T) {
	orig := getFwLeaderboards
	t.Cleanup(func() { getFwLeaderboards = orig })
	wantErr := errors.New("upstream failure")
	getFwLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsGet, error) {
		return nil, wantErr
	}

	handler := getFwLeaderboardsHandler("v1")
	_, _, err := handler(context.Background(), nil, GetFwLeaderboardsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetFwCharacterLeaderboardsHandler_PropagatesError(t *testing.T) {
	orig := getFwCharacterLeaderboards
	t.Cleanup(func() { getFwCharacterLeaderboards = orig })
	wantErr := errors.New("upstream failure")
	getFwCharacterLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCharactersGet, error) {
		return nil, wantErr
	}

	handler := getFwCharacterLeaderboardsHandler("v1")
	_, _, err := handler(context.Background(), nil, GetFwCharacterLeaderboardsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetFwCorporationLeaderboardsHandler_PropagatesError(t *testing.T) {
	orig := getFwCorporationLeaderboards
	t.Cleanup(func() { getFwCorporationLeaderboards = orig })
	wantErr := errors.New("upstream failure")
	getFwCorporationLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCorporationsGet, error) {
		return nil, wantErr
	}

	handler := getFwCorporationLeaderboardsHandler("v1")
	_, _, err := handler(context.Background(), nil, GetFwCorporationLeaderboardsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFwStatsHandler_PropagatesError(t *testing.T) {
	orig := listFwStats
	t.Cleanup(func() { listFwStats = orig })
	wantErr := errors.New("upstream failure")
	listFwStats = func(ctx context.Context, userAgent string) ([]esiapi.FwStatsGetInner, error) {
		return nil, wantErr
	}

	handler := listFwStatsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListFwStatsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFwSystemsHandler_PropagatesError(t *testing.T) {
	orig := listFwSystems
	t.Cleanup(func() { listFwSystems = orig })
	wantErr := errors.New("upstream failure")
	listFwSystems = func(ctx context.Context, userAgent string) ([]esiapi.FwSystemsGetInner, error) {
		return nil, wantErr
	}

	handler := listFwSystemsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListFwSystemsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListFwWarsHandler_PropagatesError(t *testing.T) {
	orig := listFwWars
	t.Cleanup(func() { listFwWars = orig })
	wantErr := errors.New("upstream failure")
	listFwWars = func(ctx context.Context, userAgent string) ([]esiapi.FwWarsGetInner, error) {
		return nil, wantErr
	}

	handler := listFwWarsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListFwWarsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
