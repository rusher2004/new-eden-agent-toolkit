package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestListAlliancesHandler_ReturnsIDs(t *testing.T) {
	orig := listAlliances
	t.Cleanup(func() { listAlliances = orig })
	listAlliances = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{99000001, 99000002}, nil
	}

	handler := listAlliancesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListAlliancesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.AllianceIDs) != 2 || out.AllianceIDs[0] != 99000001 {
		t.Errorf("out = %+v, want AllianceIDs=[99000001 99000002]", out)
	}
}

func TestListAlliancesHandler_PropagatesError(t *testing.T) {
	orig := listAlliances
	t.Cleanup(func() { listAlliances = orig })
	listAlliances = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, errors.New("simulated failure")
	}

	handler := listAlliancesHandler("v1")
	if _, _, err := handler(context.Background(), nil, ListAlliancesInput{}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

func TestGetAllianceHandler_ReturnsDetail(t *testing.T) {
	orig := getAllianceDetail
	t.Cleanup(func() { getAllianceDetail = orig })
	getAllianceDetail = func(ctx context.Context, userAgent string, allianceID int64) (AllianceDetailResult, error) {
		execCorp := int64(3)
		return AllianceDetailResult{
			Name:                  "Test Alliance",
			Ticker:                "TEST",
			CreatorID:             2,
			CreatorCorporationID:  1,
			ExecutorCorporationID: &execCorp,
		}, nil
	}

	handler := getAllianceHandler("v1")
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
	orig := listAllianceCorporations
	t.Cleanup(func() { listAllianceCorporations = orig })
	listAllianceCorporations = func(ctx context.Context, userAgent string, allianceID int64) ([]int64, error) {
		return []int64{98000001}, nil
	}

	handler := listAllianceCorporationsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListAllianceCorporationsInput{AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || len(out.CorporationIDs) != 1 || out.CorporationIDs[0] != 98000001 {
		t.Errorf("out = %+v, want AllianceID=99000001 CorporationIDs=[98000001]", out)
	}
}

func TestGetAllianceIconsHandler_ReturnsIcons(t *testing.T) {
	orig := getAllianceIcons
	t.Cleanup(func() { getAllianceIcons = orig })
	getAllianceIcons = func(ctx context.Context, userAgent string, allianceID int64) (AllianceIconsResult, error) {
		url := "https://images.evetech.net/a.png"
		return AllianceIconsResult{Px128x128: &url}, nil
	}

	handler := getAllianceIconsHandler("v1")
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
