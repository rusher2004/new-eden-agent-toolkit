package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fakeWarsESI fakes the one ESI call list_wars makes.
type fakeWarsESI func(ctx context.Context, maxWarID int64) ([]int64, error)

func (f fakeWarsESI) Wars(ctx context.Context, maxWarID int64) ([]int64, error) {
	return f(ctx, maxWarID)
}

// fakeWarDetailESI fakes the one ESI call get_war makes.
type fakeWarDetailESI func(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error)

func (f fakeWarDetailESI) WarDetail(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error) {
	return f(ctx, warID)
}

// fakeWarKillmailsESI fakes the one ESI call list_war_killmails makes.
type fakeWarKillmailsESI func(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error)

func (f fakeWarKillmailsESI) WarKillmails(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	return f(ctx, warID, page)
}

// --- list_wars ---

func TestListWarsHandler_MapsFields(t *testing.T) {
	fake := fakeWarsESI(func(ctx context.Context, maxWarID int64) ([]int64, error) {
		return []int64{5000, 5001}, nil
	})

	handler := listWarsHandler(func() warsESI { return fake })
	_, out, err := handler(context.Background(), nil, ListWarsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.WarIDs) != 2 || out.WarIDs[0] != 5000 {
		t.Errorf("out = %+v, want WarIDs=[5000 5001]", out)
	}
}

func TestListWarsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeWarsESI(func(ctx context.Context, maxWarID int64) ([]int64, error) {
		return nil, wantErr
	})

	handler := listWarsHandler(func() warsESI { return fake })
	_, _, err := handler(context.Background(), nil, ListWarsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_war ---

func TestGetWarHandler_MapsFields_AllianceAndCorporationParties(t *testing.T) {
	aggressorAllianceID := int64(99000001)
	defenderCorpID := int64(98000001)
	allyAllianceID := int64(99000002)
	allyCorpID := int64(98000002)
	fake := fakeWarDetailESI(func(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error) {
		return &esiapi.WarsWarIdGet{
			Id:        5000,
			Aggressor: esiapi.WarsWarIdGetAggressor{AllianceId: &aggressorAllianceID, IskDestroyed: 1000000, ShipsKilled: 5},
			Defender:  esiapi.WarsWarIdGetDefender{CorporationId: &defenderCorpID, IskDestroyed: 500000, ShipsKilled: 2},
			Allies: []esiapi.WarsWarIdGetAlliesInner{
				{AllianceId: &allyAllianceID},
				{CorporationId: &allyCorpID},
			},
			Mutual: false, OpenForAllies: true,
		}, nil
	})

	handler := getWarHandler(func() warDetailESI { return fake })
	_, out, err := handler(context.Background(), nil, GetWarInput{WarID: 5000})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.WarID != 5000 || out.Mutual || !out.OpenForAllies {
		t.Errorf("out = %+v, want WarID=5000 Mutual=false OpenForAllies=true", out)
	}
	if out.Aggressor.AllianceID == nil || *out.Aggressor.AllianceID != 99000001 || out.Aggressor.CorporationID != nil {
		t.Errorf("Aggressor = %+v, want AllianceID=99000001 CorporationID=nil", out.Aggressor)
	}
	if out.Aggressor.IskDestroyed != 1000000 || out.Aggressor.ShipsKilled != 5 {
		t.Errorf("Aggressor = %+v, want IskDestroyed=1000000 ShipsKilled=5", out.Aggressor)
	}
	if out.Defender.CorporationID == nil || *out.Defender.CorporationID != 98000001 || out.Defender.AllianceID != nil {
		t.Errorf("Defender = %+v, want CorporationID=98000001 AllianceID=nil", out.Defender)
	}
	if len(out.Allies) != 2 {
		t.Fatalf("Allies = %+v, want 2 entries", out.Allies)
	}
	if out.Allies[0].AllianceID == nil || out.Allies[0].CorporationID != nil {
		t.Errorf("Allies[0] = %+v, want AllianceID set, CorporationID nil", out.Allies[0])
	}
	if out.Allies[1].CorporationID == nil || out.Allies[1].AllianceID != nil {
		t.Errorf("Allies[1] = %+v, want CorporationID set, AllianceID nil", out.Allies[1])
	}
}

func TestGetWarHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeWarDetailESI(func(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error) {
		return nil, wantErr
	})

	handler := getWarHandler(func() warDetailESI { return fake })
	_, _, err := handler(context.Background(), nil, GetWarInput{WarID: 5000})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_war_killmails ---

func oneWarKillmail() fakeWarKillmailsESI {
	return func(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdKillmailsRecentGetInner{{KillmailId: 90000001, KillmailHash: "abc123"}}, 2, nil
	}
}

func TestListWarKillmailsHandler_MapsFields(t *testing.T) {
	fake := oneWarKillmail()

	handler := listWarKillmailsHandler(func() warKillmailsESI { return fake })
	_, out, err := handler(context.Background(), nil, ListWarKillmailsInput{WarID: 5000, Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Killmails) != 1 || out.Killmails[0].KillmailID != 90000001 || out.Killmails[0].KillmailHash != "abc123" || out.Page != 1 || out.TotalPages != 2 {
		t.Errorf("out = %+v, want one killmail KillmailID=90000001 Page=1 TotalPages=2", out)
	}
}

func TestListWarKillmailsHandler_DefaultsPageToOne(t *testing.T) {
	fake := oneWarKillmail()

	handler := listWarKillmailsHandler(func() warKillmailsESI { return fake })
	_, out, err := handler(context.Background(), nil, ListWarKillmailsInput{WarID: 5000})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("Page = %d, want 1", out.Page)
	}
}

// TestListWarKillmailsHandler_NormalizesNegativePageToOne locks in the fix for every paginated
// tool in this package: the echoed page number used to only guard against page==0, so a negative
// input page was echoed back verbatim instead of being normalized.
func TestListWarKillmailsHandler_NormalizesNegativePageToOne(t *testing.T) {
	fake := oneWarKillmail()

	handler := listWarKillmailsHandler(func() warKillmailsESI { return fake })
	_, out, err := handler(context.Background(), nil, ListWarKillmailsInput{WarID: 5000, Page: -5})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("Page = %d, want 1 (negative page normalized)", out.Page)
	}
}

func TestListWarKillmailsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeWarKillmailsESI(func(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		return nil, 0, wantErr
	})

	handler := listWarKillmailsHandler(func() warKillmailsESI { return fake })
	_, _, err := handler(context.Background(), nil, ListWarKillmailsInput{WarID: 5000})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
