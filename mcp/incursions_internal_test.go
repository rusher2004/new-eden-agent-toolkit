package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestListIncursionsHandler_MapsFields(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return []Incursion{
			{
				ConstellationID:        30000001,
				FactionID:              500001,
				HasBoss:                true,
				InfestedSolarSystemIDs: []int64{30000002, 30000003},
				Influence:              0.8,
				StagingSolarSystemID:   30000004,
				State:                  "mobilizing",
				Type:                   "minor",
			},
		}, nil
	}

	handler := listIncursionsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListIncursionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Incursions) != 1 {
		t.Errorf("len(out.Incursions) = %d, want 1", len(out.Incursions))
	}
	inc := out.Incursions[0]
	if inc.ConstellationID != 30000001 {
		t.Errorf("ConstellationID = %d, want 30000001", inc.ConstellationID)
	}
	if inc.FactionID != 500001 {
		t.Errorf("FactionID = %d, want 500001", inc.FactionID)
	}
	if inc.HasBoss != true {
		t.Errorf("HasBoss = %v, want true", inc.HasBoss)
	}
	if len(inc.InfestedSolarSystemIDs) != 2 || inc.InfestedSolarSystemIDs[0] != 30000002 {
		t.Errorf("InfestedSolarSystemIDs = %v, want [30000002 30000003]", inc.InfestedSolarSystemIDs)
	}
	if inc.Influence != 0.8 {
		t.Errorf("Influence = %f, want 0.8", inc.Influence)
	}
	if inc.StagingSolarSystemID != 30000004 {
		t.Errorf("StagingSolarSystemID = %d, want 30000004", inc.StagingSolarSystemID)
	}
	if inc.State != "mobilizing" {
		t.Errorf("State = %s, want mobilizing", inc.State)
	}
	if inc.Type != "minor" {
		t.Errorf("Type = %s, want minor", inc.Type)
	}
}

func TestListIncursionsHandler_PropagatesError(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	wantErr := errors.New("upstream failure")
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return nil, wantErr
	}

	handler := listIncursionsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListIncursionsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
