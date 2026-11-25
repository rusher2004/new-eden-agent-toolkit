package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fakeRouteESI fakes get_route's one ESI call and records its arguments.
type fakeRouteESI struct {
	route []int64
	err   error

	gotOrigin, gotDestination int64
	gotAvoid                  []int64
	gotConnections            []esiapi.RouteConnection
	gotPreference             *string
	gotSecurityPenalty        *int64
}

func (f *fakeRouteESI) Route(_ context.Context, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []esiapi.RouteConnection, preference *string, securityPenalty *int64) ([]int64, error) {
	f.gotOrigin, f.gotDestination = originSystemID, destinationSystemID
	f.gotAvoid, f.gotConnections, f.gotPreference, f.gotSecurityPenalty = avoidSystems, connections, preference, securityPenalty
	return f.route, f.err
}

func openRouteFake(c routeESI) func() routeESI { return func() routeESI { return c } }

func TestGetRouteHandler_MapsFields(t *testing.T) {
	fake := &fakeRouteESI{route: []int64{30000142, 30000144, 30000145}}

	preference := "secure"
	securityPenalty := int64(1)
	input := GetRouteInput{
		OriginSystemID: 30000142, DestinationSystemID: 30000145,
		AvoidSystems: []int64{30000200},
		Connections:  []RouteConnectionInput{{From: 30000142, To: 30000144}},
		Preference:   &preference, SecurityPenalty: &securityPenalty,
	}
	handler := getRouteHandler("v1", openRouteFake(fake), nil)
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if fake.gotOrigin != 30000142 || fake.gotDestination != 30000145 {
		t.Errorf("captured origin/destination = %d/%d, want 30000142/30000145", fake.gotOrigin, fake.gotDestination)
	}
	if len(fake.gotAvoid) != 1 || fake.gotAvoid[0] != 30000200 {
		t.Errorf("captured avoidSystems = %+v, want [30000200]", fake.gotAvoid)
	}
	if len(fake.gotConnections) != 1 || fake.gotConnections[0].From != 30000142 || fake.gotConnections[0].To != 30000144 {
		t.Errorf("captured connections = %+v, want one entry {From:30000142 To:30000144}", fake.gotConnections)
	}
	if fake.gotPreference == nil || *fake.gotPreference != "secure" {
		t.Errorf("captured preference = %v, want \"secure\"", fake.gotPreference)
	}
	if fake.gotSecurityPenalty == nil || *fake.gotSecurityPenalty != 1 {
		t.Errorf("captured securityPenalty = %v, want 1", fake.gotSecurityPenalty)
	}

	if out.OriginSystemID != 30000142 || out.DestinationSystemID != 30000145 {
		t.Errorf("out origin/destination = %d/%d, want 30000142/30000145", out.OriginSystemID, out.DestinationSystemID)
	}
	if len(out.Route) != 3 || out.Route[0] != 30000142 || out.Route[1] != 30000144 || out.Route[2] != 30000145 {
		t.Errorf("out.Route = %+v, want [30000142 30000144 30000145]", out.Route)
	}
}

func TestGetRouteHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")

	handler := getRouteHandler("v1", openRouteFake(&fakeRouteESI{err: wantErr}), nil)
	_, _, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 30000142, DestinationSystemID: 30000145})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetRouteHandler_ResolvesLocationNames(t *testing.T) {
	fake := &fakeRouteESI{route: []int64{30000142, 30000144, 30002187}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita", 30000144: "Perimeter", 30002187: "Amarr"}
	}

	handler := getRouteHandler("v1", openRouteFake(fake), nil)
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
	fake := &fakeRouteESI{route: []int64{30000142, 999999999, 30002187}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// 999999999 is deliberately omitted -- a miss must not shift neighbors.
		return map[int64]string{30000142: "Jita", 30002187: "Amarr"}
	}

	handler := getRouteHandler("v1", openRouteFake(fake), nil)
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
	fake := &fakeRouteESI{route: []int64{999999997, 999999998, 999999999}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		// Total sync failure: nothing resolves. RouteNames must stay nil so
		// omitempty drops it, rather than serializing an all-empty-string slice
		// that an LLM consumer could misread as "confirmed unnamed."
		return map[int64]string{}
	}

	handler := getRouteHandler("v1", openRouteFake(fake), nil)
	_, out, err := handler(context.Background(), nil, GetRouteInput{OriginSystemID: 999999997, DestinationSystemID: 999999999})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RouteNames != nil {
		t.Errorf("RouteNames = %v, want nil on a total miss", out.RouteNames)
	}
}
