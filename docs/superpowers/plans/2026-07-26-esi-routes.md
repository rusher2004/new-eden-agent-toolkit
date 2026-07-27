# ESI Routes Implementation Plan (twenty-seventh batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Routes" tag (1 endpoint: compute a route between two solar
systems) as 1 new MCP tool, folding into the same shared PR #6 branch as every prior batch.

**Architecture:** This is a trivial, fully-public, single-endpoint batch — combined into a single
implementation task, following the precedent set by the Incursions/Insurance batches (esi/ facade
+ mcp/ tool + registration + docs all in one task, single implementer + single reviewer, no
separate final whole-batch review stage). `esi/routes.go` wraps the generated `RoutesAPIService`'s
one method — POST-based (like `esi/assets.go`'s `AssetLocations`/`AssetNames`) despite being
read-only, since ESI uses POST here only because the request carries a body, not because it
mutates state. `mcp/routes.go` exposes one public tool, no `characterExists` gate.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: 165 MCP tools, branch `worktree-esi-access-list-activities`,
  PR #6 (open). This plan brings the count to 166.
- `PostRoute` has no security block in `esi/api/openapi.yaml` (genuinely public) and no `.Page()`
  setter (not paginated) — verified directly from `esi/api_routes.go`.
- All 4 `RouteRequestBody` fields (`AvoidSystems`, `Connections`, `Preference`, `SecurityPenalty`)
  are genuinely optional (`omitempty` in the generated JSON tags) — an empty `RouteRequestBody{}`
  is a valid request meaning "no restrictions, ESI's own defaults."
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- `RouteConnectionInput{From, To int64}` mirrors `esiapi.RouteConnection` 1:1 — same
  input/output-type split as `FittingItem`/`FittingItemInput` in `mcp/fittings.go`. No recursion
  anywhere in this batch's types.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-routes-design.md` for the verification detail.

---

### Task 1: `esi/routes.go` + `mcp/routes.go` — full batch in one task

**Files:**
- Create: `esi/routes.go`
- Test: `esi/routes_test.go`
- Create: `mcp/routes.go`
- Test: `mcp/routes_internal_test.go`
- Modify: `mcp/server.go` (register 1 tool after the `get_planetary_schematic` block; update package doc comment)
- Modify: `mcp/server_test.go` (expected-tools list: 165 → 166)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Routes row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (tool count "one hundred sixty-five" → "one hundred sixty-six", append 1 tool name)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Produces: 1 tool name — `get_route` — registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `esi/routes_test.go`:

```go
package esi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestRoute_SendsRequestBodyAndParsesResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"route":[30000142,30000144,30000145]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	preference := "secure"
	securityPenalty := int64(1)
	connections := []esiapi.RouteConnection{{From: 30000142, To: 30000144}}
	route, err := client.Route(context.Background(), 30000142, 30000145, []int64{30000200}, connections, &preference, &securityPenalty)
	if err != nil {
		t.Fatalf("Route returned error: %v", err)
	}

	var got struct {
		AvoidSystems []int64 `json:"avoid_systems"`
		Connections  []struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"connections"`
		Preference      string `json:"preference"`
		SecurityPenalty int64  `json:"security_penalty"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if len(got.AvoidSystems) != 1 || got.AvoidSystems[0] != 30000200 {
		t.Errorf("request body AvoidSystems = %+v, want [30000200]", got.AvoidSystems)
	}
	if len(got.Connections) != 1 || got.Connections[0].From != 30000142 || got.Connections[0].To != 30000144 {
		t.Errorf("request body Connections = %+v, want one entry {From:30000142 To:30000144}", got.Connections)
	}
	if got.Preference != "secure" {
		t.Errorf("request body Preference = %q, want \"secure\"", got.Preference)
	}
	if got.SecurityPenalty != 1 {
		t.Errorf("request body SecurityPenalty = %d, want 1", got.SecurityPenalty)
	}

	if len(route) != 3 || route[0] != 30000142 || route[1] != 30000144 || route[2] != 30000145 {
		t.Errorf("route = %+v, want [30000142 30000144 30000145]", route)
	}
}

func TestRoute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.Route(context.Background(), 30000142, 30000145, nil, nil, nil, nil); err == nil {
		t.Fatal("Route returned nil error, want an error for HTTP 404")
	}
}
```

Create `mcp/routes_internal_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestRoute' -v` and `go test ./mcp/... -run 'TestGetRouteHandler' -v`
Expected: FAIL — `esi.Client` has no `Route` method, and none of the mcp symbols exist yet.

- [ ] **Step 3: Write the implementation**

Create `esi/routes.go`:

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

Create `mcp/routes.go`:

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

Modify `mcp/server.go`: append after the existing `get_planetary_schematic` block (immediately
before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_route",
		Description: "Compute a route between two solar systems, optionally avoiding systems or adding extra connections (public data, no character required).",
	}, getRouteHandler(version))
```

Update `mcp/server.go`'s package doc comment: change the trailing
`"... and get_planetary_schematic registered."`
to end with
`"... get_planetary_schematic, and get_route registered."`

Modify `mcp/server_test.go`: append `"get_route"` to the expected-tools slice (after
`"get_planetary_schematic"`), matching the file's existing membership-check style.

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Routes row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: change "one hundred sixty-five" to "one hundred sixty-six", and append
`get_route` to the `mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestRoute' -v` and
`go test ./mcp/... -run 'TestGetRouteHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (2 esi tests + 2 mcp tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, 166 tools registered without panic.

```bash
git add esi/routes.go esi/routes_test.go mcp/routes.go mcp/routes_internal_test.go \
  mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add routes MCP tool (get_route: compute a route between two solar systems)"
```
