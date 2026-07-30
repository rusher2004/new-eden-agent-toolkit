package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestGetRouteHandler_MapsFields(t *testing.T) {
	orig := getRoute
	t.Cleanup(func() { getRoute = orig })
	var gotOriginID, gotDestinationID int64
	var gotAvoidSystems []int64
	var gotConnections []RouteConnectionInput
	var gotPreference *string
	var gotSecurityPenalty *int64
	getRoute = func(ctx context.Context, userAgent string, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
		gotOriginID, gotDestinationID = originSystemID, destinationSystemID
		gotAvoidSystems, gotConnections, gotPreference, gotSecurityPenalty = avoidSystems, connections, preference, securityPenalty
		return []int64{30000142, 30000144, 30000145}, nil
	}

	preference := "secure"
	securityPenalty := int64(1)
	input := GetRouteInput{
		OriginSystemID: 30000142, DestinationSystemID: 30000145,
		AvoidSystems: []int64{30000200},
		Connections:  []RouteConnectionInput{{From: 30000142, To: 30000144}},
		Preference:   &preference, SecurityPenalty: &securityPenalty,
	}
	handler := getRouteHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotOriginID != 30000142 || gotDestinationID != 30000145 {
		t.Errorf("captured origin/destination = %d/%d, want 30000142/30000145", gotOriginID, gotDestinationID)
	}
	if len(gotAvoidSystems) != 1 || gotAvoidSystems[0] != 30000200 {
		t.Errorf("captured avoidSystems = %+v, want [30000200]", gotAvoidSystems)
	}
	if len(gotConnections) != 1 || gotConnections[0].From != 30000142 || gotConnections[0].To != 30000144 {
		t.Errorf("captured connections = %+v, want one entry {From:30000142 To:30000144}", gotConnections)
	}
	if gotPreference == nil || *gotPreference != "secure" {
		t.Errorf("captured preference = %v, want \"secure\"", gotPreference)
	}
	if gotSecurityPenalty == nil || *gotSecurityPenalty != 1 {
		t.Errorf("captured securityPenalty = %v, want 1", gotSecurityPenalty)
	}

	if out.OriginSystemID != 30000142 || out.DestinationSystemID != 30000145 {
		t.Errorf("out origin/destination = %d/%d, want 30000142/30000145", out.OriginSystemID, out.DestinationSystemID)
	}
	if len(out.Route) != 3 || out.Route[0] != 30000142 || out.Route[1] != 30000144 || out.Route[2] != 30000145 {
		t.Errorf("out.Route = %+v, want [30000142 30000144 30000145]", out.Route)
	}
}

func TestGetRouteHandler_PropagatesError(t *testing.T) {
	orig := getRoute
	t.Cleanup(func() { getRoute = orig })
	getRoute = func(ctx context.Context, userAgent string, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
		return nil, errors.New("boom")
	}

	handler := getRouteHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 30000142, DestinationSystemID: 30000145})
	if err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}

func TestGetRouteHandler_ResolvesLocationNames(t *testing.T) {
	orig := getRoute
	t.Cleanup(func() { getRoute = orig })
	getRoute = func(ctx context.Context, userAgent string, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
		return []int64{30000142, 30000144, 30002187}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter", 30002187: "Amarr"}
	}

	handler := getRouteHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 30000142, DestinationSystemID: 30002187})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.OriginSystemName != "Jita" {
		t.Errorf("OriginSystemName = %q, want %q", out.OriginSystemName, "Jita")
	}
	if out.DestinationSystemName != "Amarr" {
		t.Errorf("DestinationSystemName = %q, want %q", out.DestinationSystemName, "Amarr")
	}
	want := []string{"Jita", "Perimeter", "Amarr"}
	if len(out.RouteNames) != len(want) {
		t.Fatalf("RouteNames = %v, want %v", out.RouteNames, want)
	}
	for i, w := range want {
		if out.RouteNames[i] != w {
			t.Errorf("RouteNames[%d] = %q, want %q", i, out.RouteNames[i], w)
		}
	}
}

func TestGetRouteHandler_RouteNamesSurvivesPartialMiss(t *testing.T) {
	orig := getRoute
	t.Cleanup(func() { getRoute = orig })
	getRoute = func(ctx context.Context, userAgent string, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
		return []int64{30000142, 999999999, 30002187}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// 999999999 is deliberately omitted -- a miss must not shift neighbors.
		return map[int64]string{30000142: "Jita", 30002187: "Amarr"}
	}

	handler := getRouteHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 30000142, DestinationSystemID: 30002187})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []string{"Jita", "", "Amarr"}
	if len(out.RouteNames) != len(want) {
		t.Fatalf("RouteNames = %v, want %v", out.RouteNames, want)
	}
	for i, w := range want {
		if out.RouteNames[i] != w {
			t.Errorf("RouteNames[%d] = %q, want %q", i, out.RouteNames[i], w)
		}
	}
}

func TestGetRouteHandler_RouteNamesNilOnTotalMiss(t *testing.T) {
	orig := getRoute
	t.Cleanup(func() { getRoute = orig })
	getRoute = func(ctx context.Context, userAgent string, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
		return []int64{999999997, 999999998, 999999999}, nil
	}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// Total sync failure: nothing resolves. RouteNames must stay nil so
		// omitempty drops it, rather than serializing an all-empty-string slice
		// that an LLM consumer could misread as "confirmed unnamed."
		return map[int64]string{}
	}

	handler := getRouteHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 999999997, DestinationSystemID: 999999999})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RouteNames != nil {
		t.Errorf("RouteNames = %v, want nil on a total miss", out.RouteNames)
	}
}
