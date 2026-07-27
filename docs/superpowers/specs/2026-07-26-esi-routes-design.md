# ESI Routes — Design Spec (twenty-seventh batch)

**Batch:** Routes tag, 1 endpoint, public, no pagination. A route-computation "compute" endpoint —
POST-based despite being read-only, the same pattern already established in this codebase by
`esi/assets.go`'s `AssetLocations`/`AssetNames` (POST used for a stateless lookup, not a mutation).

## Endpoint (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

`PostRoute(ctx, originSystemId, destinationSystemId int64) ApiPostRouteRequest` →
`.RouteRequestBody(RouteRequestBody{AvoidSystems []int64, Connections []RouteConnection{From, To int64}, Preference *string, SecurityPenalty *int64})`
→ `.Execute() (*Route{Route []int64}, *http.Response, error)`.

Public — verified: `/route/{origin_system_id}/{destination_system_id}` has no `security:` block
in `openapi.yaml`. Not paginated — no `.Page()` setter on `ApiPostRouteRequest`. All 4 request-body
fields are genuinely optional (`omitempty` in the generated JSON tags), so an empty
`RouteRequestBody{}` is valid and means "no avoid-list, no extra connections, ESI's own default
preference/strictness."

No recursion anywhere — `Route`, `RouteRequestBody`, `RouteConnection` are all flat.

## `esi/routes.go`

```go
// Package esi's routes.go covers the single "Routes" endpoint -- a public, stateless route
// computation between two solar systems. POST-based (like esi/assets.go's AssetLocations/
// AssetNames) despite being read-only: ESI uses POST here because the request carries a body
// (avoid-list, extra connections, preference), not because this mutates any state.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Route computes a route between originSystemID and destinationSystemID, returning the ordered
// list of solar system IDs to traverse. avoidSystems and connections may be nil (no restriction/
// no extra connections); preference and securityPenalty may be nil to accept ESI's own defaults.
// Public -- no security block on this operation in esi/api/openapi.yaml.
func (c *Client) Route(ctx context.Context, originSystemID, destinationSystemID int64, avoidSystems []int64, connections []esiapi.RouteConnection, preference *string, securityPenalty *int64) ([]int64, error) {
	body := esiapi.RouteRequestBody{
		AvoidSystems: avoidSystems, Connections: connections, Preference: preference, SecurityPenalty: securityPenalty,
	}
	result, resp, err := c.api.RoutesAPI.PostRoute(ctx, originSystemID, destinationSystemID).RouteRequestBody(body).Execute()
	if err != nil {
		return nil, fmt.Errorf("get route: %w", describeError(err, resp))
	}
	return result.Route, nil
}
```

`esi/routes_test.go`: 2 tests (success + HTTP error), plus a request-body assertion confirming
`avoid_systems`/`connections`/`preference`/`security_penalty` all serialize when set, following
`esi/assets_test.go`'s style for asserting a POST request body.

## `mcp/routes.go`

One public tool, `get_route`, no `characterExists` gate (matching `mcp/insurance.go`/
`mcp/meta.go`'s fully-public shape). `RouteConnectionInput{From, To int64}` mirrors
`esiapi.RouteConnection` 1:1 — same input/output split as `FittingItem`/`FittingItemInput` in
`mcp/fittings.go`.

```go
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
	OriginSystemID      int64   `json:"originSystemId"`
	DestinationSystemID int64   `json:"destinationSystemId"`
	Route               []int64 `json:"route"`
}

func getRouteHandler(version string) mcpsdk.ToolHandlerFor[GetRouteInput, GetRouteOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetRouteInput) (*mcpsdk.CallToolResult, GetRouteOutput, error) {
		route, err := getRoute(ctx, sso.UserAgent(version), in.OriginSystemID, in.DestinationSystemID, in.AvoidSystems, in.Connections, in.Preference, in.SecurityPenalty)
		if err != nil {
			return nil, GetRouteOutput{}, fmt.Errorf("get route: %w", err)
		}
		return nil, GetRouteOutput{OriginSystemID: in.OriginSystemID, DestinationSystemID: in.DestinationSystemID, Route: route}, nil
	}
}
```

Both types are flat, non-recursive. Already sandbox-verified against a live `mcpsdk.AddTool`
registration test with zero schema-generator errors.

`mcp/routes_internal_test.go`: 2 tests (`MapsFields` covering all optional fields set, and
`PropagatesError`), following `mcp/insurance_internal_test.go`'s public-tool pattern.

## `mcp/server.go` / `mcp/server_test.go`

Register 1 tool after the `get_planetary_schematic` block. Update package doc comment.
`server_test.go`'s expected-tools list: 165 → 166.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred sixty-five" → "one hundred sixty-six",
  append `get_route`.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Routes row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/routes_test.go`,
swappable-`var`-override tests for `mcp/routes_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.

Given this batch's size (1 endpoint, no auth complexity), Task 1 and Task 2 are combined into a
single implementation task, following the precedent set by Incursions/Insurance.
