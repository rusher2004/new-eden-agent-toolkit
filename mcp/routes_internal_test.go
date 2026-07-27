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
	handler := getRouteHandler("v1")
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

	handler := getRouteHandler("v1")
	_, _, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 30000142, DestinationSystemID: 30000145})
	if err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
