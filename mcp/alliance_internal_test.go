package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fakeAllianceESI fakes the alliance tools' ESI calls; err is returned by every call.
type fakeAllianceESI struct {
	ids          []int64
	detail       *esiapi.AlliancesDetail
	corporations []int64
	icons        *esiapi.AlliancesAllianceIdIconsGet
	err          error
}

func (f fakeAllianceESI) Alliances(context.Context) ([]int64, error) { return f.ids, f.err }

func (f fakeAllianceESI) AllianceDetail(context.Context, int64) (*esiapi.AlliancesDetail, error) {
	return f.detail, f.err
}

func (f fakeAllianceESI) AllianceCorporations(context.Context, int64) ([]int64, error) {
	return f.corporations, f.err
}

func (f fakeAllianceESI) AllianceIcons(context.Context, int64) (*esiapi.AlliancesAllianceIdIconsGet, error) {
	return f.icons, f.err
}

func openAllianceFake(c allianceESI) func() allianceESI { return func() allianceESI { return c } }

func TestListAlliancesHandler_ReturnsIDs(t *testing.T) {
	handler := listAlliancesHandler(openAllianceFake(fakeAllianceESI{ids: []int64{99000001, 99000002}}))
	_, out, err := handler(context.Background(), nil, ListAlliancesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.AllianceIDs) != 2 || out.AllianceIDs[0] != 99000001 {
		t.Errorf("out = %+v, want AllianceIDs=[99000001 99000002]", out)
	}
}

func TestListAlliancesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("simulated failure")

	handler := listAlliancesHandler(openAllianceFake(fakeAllianceESI{err: wantErr}))
	if _, _, err := handler(context.Background(), nil, ListAlliancesInput{}); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetAllianceHandler_ReturnsDetail(t *testing.T) {
	execCorp := int64(3)
	fake := fakeAllianceESI{detail: &esiapi.AlliancesDetail{
		Name:                  "Test Alliance",
		Ticker:                "TEST",
		CreatorId:             2,
		CreatorCorporationId:  1,
		ExecutorCorporationId: &execCorp,
	}}

	handler := getAllianceHandler("v1", openAllianceFake(fake), nil)
	_, out, err := handler(context.Background(), nil, GetAllianceInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || out.Name != "Test Alliance" || out.ExecutorCorporationID == nil || *out.ExecutorCorporationID != 3 {
		t.Errorf("out = %+v, want AllianceID=99000001 Name=\"Test Alliance\" ExecutorCorporationID=*3", out)
	}
	if out.FactionID != nil {
		t.Errorf("out.FactionID = %v, want nil", out.FactionID)
	}
}

func TestListAllianceCorporationsHandler_ReturnsIDs(t *testing.T) {
	handler := listAllianceCorporationsHandler(openAllianceFake(fakeAllianceESI{corporations: []int64{98000001}}))
	_, out, err := handler(context.Background(), nil, ListAllianceCorporationsInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || len(out.CorporationIDs) != 1 || out.CorporationIDs[0] != 98000001 {
		t.Errorf("out = %+v, want AllianceID=99000001 CorporationIDs=[98000001]", out)
	}
}

func TestGetAllianceIconsHandler_ReturnsIcons(t *testing.T) {
	url := "https://images.evetech.net/a.png"
	fake := fakeAllianceESI{icons: &esiapi.AlliancesAllianceIdIconsGet{Px128x128: &url}}

	handler := getAllianceIconsHandler(openAllianceFake(fake))
	_, out, err := handler(context.Background(), nil, GetAllianceIconsInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || out.Px128x128 == nil || *out.Px128x128 != "https://images.evetech.net/a.png" {
		t.Errorf("out = %+v, want AllianceID=99000001 Px128x128=*\"https://images.evetech.net/a.png\"", out)
	}
	if out.Px64x64 != nil {
		t.Errorf("out.Px64x64 = %v, want nil", out.Px64x64)
	}
}

func TestGetAllianceHandler_ResolvesFactionName(t *testing.T) {
	factionID := int64(500001)
	fake := fakeAllianceESI{detail: &esiapi.AlliancesDetail{Name: "Test Alliance", FactionId: &factionID}}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Caldari State"}
	}

	handler := getAllianceHandler("v1", openAllianceFake(fake), nil)
	_, out, err := handler(context.Background(), nil, GetAllianceInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "Caldari State" {
		t.Errorf("FactionName = %q, want %q", out.FactionName, "Caldari State")
	}
}

func TestGetAllianceHandler_FactionNameEmptyWhenFactionIDNil(t *testing.T) {
	fake := fakeAllianceESI{detail: &esiapi.AlliancesDetail{Name: "Test Alliance"}}

	handler := getAllianceHandler("v1", openAllianceFake(fake), nil)
	_, out, err := handler(context.Background(), nil, GetAllianceInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.FactionName != "" {
		t.Errorf("FactionName = %q, want empty when FactionID is nil", out.FactionName)
	}
}

func TestGetAllianceHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	handler := getAllianceHandler("v1", openAllianceFake(fakeAllianceESI{err: wantErr}), nil)
	if _, _, err := handler(context.Background(), nil, GetAllianceInput{AllianceID: 99000001}); !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
