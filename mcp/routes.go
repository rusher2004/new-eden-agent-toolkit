package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// routeESI is the ESI call get_route makes; it is public.
type routeESI interface {
	Route(ctx context.Context, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []esiapi.RouteConnection, preference *string, securityPenalty *int64) ([]int64, error)
}

// --- get_route (public) ---

type RouteConnectionInput struct {
	From int64 `json:"from" jsonschema:"the origin system ID of this one-way connection"`
	To   int64 `json:"to" jsonschema:"the destination system ID of this one-way connection"`
}

func getRoute(ctx context.Context, client routeESI, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
	esiConnections := make([]esiapi.RouteConnection, len(connections))
	for i, c := range connections {
		esiConnections[i] = esiapi.RouteConnection{From: c.From, To: c.To}
	}
	return client.Route(ctx, originSystemID, destinationSystemID, avoidSystems, esiConnections, preference, securityPenalty)
}

type GetRouteInput struct {
	OriginSystemID      int64                  `json:"originSystemId" jsonschema:"the origin solar system ID"`
	DestinationSystemID int64                  `json:"destinationSystemId" jsonschema:"the destination solar system ID"`
	AvoidSystems        []int64                `json:"avoidSystems,omitempty" jsonschema:"solar system IDs to avoid; omit for no restriction"`
	Connections         []RouteConnectionInput `json:"connections,omitempty" jsonschema:"additional one-way connections (e.g. jump bridges) between systems; omit for none"`
	Preference          *string                `json:"preference,omitempty" jsonschema:"route preference: shortest, secure, or insecure; omit for ESI's default (shortest)"`
	SecurityPenalty     *int64                 `json:"securityPenalty,omitempty" jsonschema:"strictness of the security preference; omit for ESI's default"`
}
type GetRouteOutput struct {
	OriginSystemID        int64    `json:"originSystemId"`
	OriginSystemName      string   `json:"originSystemName,omitempty"`
	DestinationSystemID   int64    `json:"destinationSystemId"`
	DestinationSystemName string   `json:"destinationSystemName,omitempty"`
	Route                 []int64  `json:"route"`
	RouteNames            []string `json:"routeNames,omitempty"`
}

// AddGetRoute registers get_route. open builds the ESI client; any client with the
// routeESI methods will do.
func AddGetRoute[C routeESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore systemNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_route",
		Description: "Compute a route between two solar systems, optionally avoiding systems or adding extra connections (public data, no character required).",
	}, getRouteHandler(userAgent, open, sdeStore))
}

func getRouteHandler[C routeESI](userAgent string, open func() C, sdeStore systemNameCache) mcpsdk.ToolHandlerFor[GetRouteInput, GetRouteOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetRouteInput) (*mcpsdk.CallToolResult, GetRouteOutput, error) {
		route, err := getRoute(ctx, open(), in.OriginSystemID, in.DestinationSystemID, in.AvoidSystems, in.Connections, in.Preference, in.SecurityPenalty)
		if err != nil {
			return nil, GetRouteOutput{}, fmt.Errorf("get route: %w", err)
		}

		// ESI's route response always includes origin/destination as its first/last elements, so
		// names already covers both -- no second lookup needed.
		names := resolveSystemNames(ctx, sdeStore, userAgent, route)
		routeNames := make([]string, len(route))
		var resolved bool
		for i, id := range route {
			if name, ok := names[id]; ok {
				routeNames[i] = name
				resolved = true
			}
		}
		if !resolved {
			routeNames = nil
		}

		return nil, GetRouteOutput{
			OriginSystemID: in.OriginSystemID, OriginSystemName: names[in.OriginSystemID],
			DestinationSystemID: in.DestinationSystemID, DestinationSystemName: names[in.DestinationSystemID],
			Route: route, RouteNames: routeNames,
		}, nil
	}
}
