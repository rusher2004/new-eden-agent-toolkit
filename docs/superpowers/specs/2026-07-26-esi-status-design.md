# ESI Status — Design Spec (thirty-first batch)

**Batch:** Status tag, 1 endpoint, public, no pagination.

## Endpoint (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

`GetStatus(ctx) ApiGetStatusRequest` → `.Execute() (*StatusGet{Players int32, ServerVersion string,
StartTime time.Time (all required), Vip *bool (optional)}, *http.Response, error)`. Public — `/status`
has no security block. Not paginated. Flat, non-recursive.

## `esi/status.go`

```go
// Package esi's status.go covers the single "Status" endpoint -- server player count, version,
// start time, and VIP-mode flag. Public (no security block in esi/api/openapi.yaml), not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Status returns the current server status: online player count, running version, start time,
// and whether VIP mode is active.
func (c *Client) Status(ctx context.Context) (*esiapi.StatusGet, error) {
	result, resp, err := c.api.StatusAPI.GetStatus(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get status: %w", describeError(err, resp))
	}
	return result, nil
}
```

`esi/status_test.go`: 2 tests (success + HTTP error).

## `mcp/status.go`

One public tool, `get_status`, no `characterExists` gate, matching `mcp/meta.go`'s shape.

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_status (public) ---

var getStatus = func(ctx context.Context, userAgent string) (int32, string, time.Time, *bool, error) {
	client := esi.NewPublicClient(userAgent)
	status, err := client.Status(ctx)
	if err != nil {
		return 0, "", time.Time{}, nil, err
	}
	return status.Players, status.ServerVersion, status.StartTime, status.Vip, nil
}

type GetStatusInput struct{}
type GetStatusOutput struct {
	Players       int32     `json:"players"`
	ServerVersion string    `json:"serverVersion"`
	StartTime     time.Time `json:"startTime"`
	Vip           *bool     `json:"vip,omitempty"`
}

func getStatusHandler(version string) mcpsdk.ToolHandlerFor[GetStatusInput, GetStatusOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetStatusInput) (*mcpsdk.CallToolResult, GetStatusOutput, error) {
		players, serverVersion, startTime, vip, err := getStatus(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetStatusOutput{}, fmt.Errorf("get status: %w", err)
		}
		return nil, GetStatusOutput{Players: players, ServerVersion: serverVersion, StartTime: startTime, Vip: vip}, nil
	}
}
```

`GetStatusOutput` is flat, non-recursive. Already sandbox-verified against a live `mcpsdk.AddTool`
registration test with zero schema-generator errors.

`mcp/status_internal_test.go`: 2 tests (`MapsFields` + `PropagatesError`), following
`mcp/meta_internal_test.go`'s public-tool pattern.

## `mcp/server.go` / `mcp/server_test.go`

Register 1 tool after whatever the last-registered tool is at execution time (expected to be one
of the Sovereignty tools). Update package doc comment. `server_test.go`'s expected-tools list:
append `get_status`.

## Docs sync

- `CLAUDE.md` / `README.md`: increment tool count by 1 from whatever it currently reads, append
  `get_status`.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Status row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/status_test.go`,
swappable-`var`-override tests for `mcp/status_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.

Given this batch's size (1 endpoint, fully public), Task 1 and Task 2 are combined into a single
implementation task, following the precedent set by Incursions/Insurance/Meta/Routes/Search.
