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

	handler := listIncursionsHandler("v1", nil)
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

	handler := listIncursionsHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListIncursionsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListIncursionsHandler_ResolvesLocationNames(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return []Incursion{{
			ConstellationID: 20000020, StagingSolarSystemID: 30000142,
			InfestedSolarSystemIDs: []int64{30000142, 30000144},
		}}, nil
	}

	origConstellation := resolveConstellationNames
	t.Cleanup(func() { resolveConstellationNames = origConstellation })
	resolveConstellationNames = func(ctx context.Context, sdeStore constellationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{20000020: "Kimotoro"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter"}
	}

	handler := listIncursionsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIncursionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Incursions[0].ConstellationName != "Kimotoro" {
		t.Errorf("ConstellationName = %q, want %q", out.Incursions[0].ConstellationName, "Kimotoro")
	}
	if out.Incursions[0].StagingSolarSystemName != "Jita" {
		t.Errorf("StagingSolarSystemName = %q, want %q", out.Incursions[0].StagingSolarSystemName, "Jita")
	}
	want := []string{"Jita", "Perimeter"}
	if len(out.Incursions[0].InfestedSolarSystemNames) != len(want) {
		t.Fatalf("InfestedSolarSystemNames = %v, want %v", out.Incursions[0].InfestedSolarSystemNames, want)
	}
	for i, w := range want {
		if out.Incursions[0].InfestedSolarSystemNames[i] != w {
			t.Errorf("InfestedSolarSystemNames[%d] = %q, want %q", i, out.Incursions[0].InfestedSolarSystemNames[i], w)
		}
	}
}

func TestListIncursionsHandler_InfestedSolarSystemNamesSurvivesPartialMiss(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return []Incursion{{InfestedSolarSystemIDs: []int64{30000142, 999999999, 30000144}}}, nil
	}

	origConstellation := resolveConstellationNames
	t.Cleanup(func() { resolveConstellationNames = origConstellation })
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// 999999999 is deliberately omitted -- a miss must not shift neighbors.
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter"}
	}

	handler := listIncursionsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIncursionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []string{"Jita", "", "Perimeter"}
	if len(out.Incursions[0].InfestedSolarSystemNames) != len(want) {
		t.Fatalf("InfestedSolarSystemNames = %v, want %v", out.Incursions[0].InfestedSolarSystemNames, want)
	}
	for i, w := range want {
		if out.Incursions[0].InfestedSolarSystemNames[i] != w {
			t.Errorf("InfestedSolarSystemNames[%d] = %q, want %q", i, out.Incursions[0].InfestedSolarSystemNames[i], w)
		}
	}
}

func TestListIncursionsHandler_InfestedSolarSystemNamesNilOnTotalMiss(t *testing.T) {
	orig := listIncursions
	t.Cleanup(func() { listIncursions = orig })
	listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
		return []Incursion{{InfestedSolarSystemIDs: []int64{999999997, 999999998, 999999999}}}, nil
	}

	origConstellation := resolveConstellationNames
	t.Cleanup(func() { resolveConstellationNames = origConstellation })
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// Total sync failure: nothing resolves. InfestedSolarSystemNames must stay
		// nil so omitempty drops it, rather than serializing an all-empty-string
		// slice that an LLM consumer could misread as "confirmed unnamed."
		return map[int64]string{}
	}

	handler := listIncursionsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListIncursionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Incursions[0].InfestedSolarSystemNames != nil {
		t.Errorf("InfestedSolarSystemNames = %v, want nil on a total miss", out.Incursions[0].InfestedSolarSystemNames)
	}
}
