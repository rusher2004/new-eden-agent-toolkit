package mcp

import (
	"context"
	"errors"
	"testing"
)

// --- list_wars ---

func TestListWarsHandler_MapsFields(t *testing.T) {
	orig := listWars
	t.Cleanup(func() { listWars = orig })
	listWars = func(ctx context.Context, userAgent string, maxWarID int64) ([]int64, error) {
		return []int64{5000, 5001}, nil
	}

	handler := listWarsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListWarsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.WarIDs) != 2 || out.WarIDs[0] != 5000 {
		t.Errorf("out = %+v, want WarIDs=[5000 5001]", out)
	}
}

func TestListWarsHandler_PropagatesError(t *testing.T) {
	orig := listWars
	t.Cleanup(func() { listWars = orig })
	wantErr := errors.New("upstream failure")
	listWars = func(ctx context.Context, userAgent string, maxWarID int64) ([]int64, error) {
		return nil, wantErr
	}

	handler := listWarsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListWarsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_war ---

func TestGetWarHandler_MapsFields_AllianceAndCorporationParties(t *testing.T) {
	orig := getWar
	t.Cleanup(func() { getWar = orig })
	aggressorAllianceID := int64(99000001)
	defenderCorpID := int64(98000001)
	allyAllianceID := int64(99000002)
	allyCorpID := int64(98000002)
	getWar = func(ctx context.Context, userAgent string, warID int64) (War, error) {
		return War{
			WarID:     5000,
			Aggressor: WarParty{AllianceID: &aggressorAllianceID, IskDestroyed: 1000000, ShipsKilled: 5},
			Defender:  WarParty{CorporationID: &defenderCorpID, IskDestroyed: 500000, ShipsKilled: 2},
			Allies: []WarAlly{
				{AllianceID: &allyAllianceID},
				{CorporationID: &allyCorpID},
			},
			Mutual: false, OpenForAllies: true,
		}, nil
	}

	handler := getWarHandler("v1")
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
	orig := getWar
	t.Cleanup(func() { getWar = orig })
	wantErr := errors.New("upstream failure")
	getWar = func(ctx context.Context, userAgent string, warID int64) (War, error) {
		return War{}, wantErr
	}

	handler := getWarHandler("v1")
	_, _, err := handler(context.Background(), nil, GetWarInput{WarID: 5000})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_war_killmails ---

func TestListWarKillmailsHandler_MapsFields(t *testing.T) {
	orig := listWarKillmails
	t.Cleanup(func() { listWarKillmails = orig })
	listWarKillmails = func(ctx context.Context, userAgent string, warID int64, page int32) ([]KillmailRef, int32, error) {
		return []KillmailRef{{KillmailID: 90000001, KillmailHash: "abc123"}}, 2, nil
	}

	handler := listWarKillmailsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListWarKillmailsInput{WarID: 5000, Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Killmails) != 1 || out.Killmails[0].KillmailID != 90000001 || out.Page != 1 || out.TotalPages != 2 {
		t.Errorf("out = %+v, want one killmail KillmailID=90000001 Page=1 TotalPages=2", out)
	}
}

func TestListWarKillmailsHandler_DefaultsPageToOne(t *testing.T) {
	orig := listWarKillmails
	t.Cleanup(func() { listWarKillmails = orig })
	listWarKillmails = func(ctx context.Context, userAgent string, warID int64, page int32) ([]KillmailRef, int32, error) {
		return []KillmailRef{{KillmailID: 90000001, KillmailHash: "abc123"}}, 2, nil
	}

	handler := listWarKillmailsHandler("v1")
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
	orig := listWarKillmails
	t.Cleanup(func() { listWarKillmails = orig })
	listWarKillmails = func(ctx context.Context, userAgent string, warID int64, page int32) ([]KillmailRef, int32, error) {
		return []KillmailRef{{KillmailID: 90000001, KillmailHash: "abc123"}}, 2, nil
	}

	handler := listWarKillmailsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListWarKillmailsInput{WarID: 5000, Page: -5})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("Page = %d, want 1 (negative page normalized)", out.Page)
	}
}

func TestListWarKillmailsHandler_PropagatesError(t *testing.T) {
	orig := listWarKillmails
	t.Cleanup(func() { listWarKillmails = orig })
	wantErr := errors.New("upstream failure")
	listWarKillmails = func(ctx context.Context, userAgent string, warID int64, page int32) ([]KillmailRef, int32, error) {
		return nil, 0, wantErr
	}

	handler := listWarKillmailsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListWarKillmailsInput{WarID: 5000})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
