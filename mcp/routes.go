package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_route (public) ---

type RouteConnectionInput struct {
	From int64 `json:"from" jsonschema:"the origin system ID of this one-way connection"`
	To   int64 `json:"to" jsonschema:"the destination system ID of this one-way connection"`
}

var getRoute = func(ctx context.Context, userAgent string, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []RouteConnectionInput, preference *string, securityPenalty *int64) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
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

func getRouteHandler(version string, sdeStore SDEStore) mcpsdk.ToolHandlerFor[GetRouteInput, GetRouteOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetRouteInput) (*mcpsdk.CallToolResult, GetRouteOutput, error) {
		route, err := getRoute(ctx, sso.UserAgent(version), in.OriginSystemID, in.DestinationSystemID, in.AvoidSystems, in.Connections, in.Preference, in.SecurityPenalty)
		if err != nil {
			return nil, GetRouteOutput{}, fmt.Errorf("get route: %w", err)
		}

		// ESI's route response always includes origin/destination as its first/last elements, so
		// names already covers both -- no second lookup needed.
		names := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), route)
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
