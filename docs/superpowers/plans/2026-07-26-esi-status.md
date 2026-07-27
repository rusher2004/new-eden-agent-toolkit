# ESI Status Implementation Plan (thirty-first batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Status" tag (1 endpoint: server status) as 1 new MCP tool,
folding into the same shared PR #6 branch as every prior batch.

**Architecture:** This is a trivial, fully-public, single-endpoint batch — combined into a single
implementation task, following the precedent set by Incursions/Insurance/Meta/Routes/Search.
`esi/status.go` wraps the generated `StatusAPIService`'s one method. `mcp/status.go` exposes one
public tool, no `characterExists` gate.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: whatever the actual tool count is once Sovereignty lands (this plan
  assumes Sovereignty's 30th-batch commit is already merged). **If Sovereignty has not yet landed
  when this plan is executed, adjust the baseline count and registration anchor accordingly** —
  use `grep -c 'mcpsdk.AddTool' mcp/server.go` for the real current count before writing any
  doc-count edits, and register after whatever the actual last tool is.
- `GetStatus` has no security block in `esi/api/openapi.yaml` (genuinely public) and no `.Page()`
  setter (not paginated).
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- `GetStatusOutput` is flat, non-recursive.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-status-design.md` for the verification detail.

---

### Task 1: `esi/status.go` + `mcp/status.go` — full batch in one task

**Files:**
- Create: `esi/status.go`
- Test: `esi/status_test.go`
- Create: `mcp/status.go`
- Test: `mcp/status_internal_test.go`
- Modify: `mcp/server.go` (register 1 tool after whatever the actual last-registered tool is at
  execution time; update package doc comment)
- Modify: `mcp/server_test.go` (append `"get_status"` to expected-tools list)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Status row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (increment tool count by 1 from whatever it currently reads, append `get_status`)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Produces: 1 tool name — `get_status` — registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `esi/status_test.go`:

```go
package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestStatus_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"players":23456,"server_version":"1234567","start_time":"2026-07-25T11:00:00Z","vip":false}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Players != 23456 || status.ServerVersion != "1234567" {
		t.Errorf("status = %+v, want Players=23456 ServerVersion=\"1234567\"", status)
	}
	wantStart := time.Date(2026, 7, 25, 11, 0, 0, 0, time.UTC)
	if !status.StartTime.Equal(wantStart) {
		t.Errorf("StartTime = %v, want %v", status.StartTime, wantStart)
	}
	if status.Vip == nil || *status.Vip {
		t.Errorf("Vip = %v, want *false", status.Vip)
	}
}

func TestStatus_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.Status(context.Background()); err == nil {
		t.Fatal("Status returned nil error, want an error for HTTP 500")
	}
}
```

Create `mcp/status_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGetStatusHandler_MapsFields(t *testing.T) {
	orig := getStatus
	t.Cleanup(func() { getStatus = orig })
	startTime := time.Date(2026, 7, 25, 11, 0, 0, 0, time.UTC)
	vip := false
	getStatus = func(ctx context.Context, userAgent string) (int32, string, time.Time, *bool, error) {
		return 23456, "1234567", startTime, &vip, nil
	}

	handler := getStatusHandler("v1")
	_, out, err := handler(context.Background(), nil, GetStatusInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Players != 23456 || out.ServerVersion != "1234567" {
		t.Errorf("out = %+v, want Players=23456 ServerVersion=\"1234567\"", out)
	}
	if !out.StartTime.Equal(startTime) {
		t.Errorf("StartTime = %v, want %v", out.StartTime, startTime)
	}
	if out.Vip == nil || *out.Vip {
		t.Errorf("Vip = %v, want *false", out.Vip)
	}
}

func TestGetStatusHandler_PropagatesError(t *testing.T) {
	orig := getStatus
	t.Cleanup(func() { getStatus = orig })
	getStatus = func(ctx context.Context, userAgent string) (int32, string, time.Time, *bool, error) {
		return 0, "", time.Time{}, nil, errors.New("boom")
	}

	handler := getStatusHandler("v1")
	_, _, err := handler(context.Background(), nil, GetStatusInput{})
	if err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestStatus' -v` and `go test ./mcp/... -run 'TestGetStatusHandler' -v`
Expected: FAIL — `esi.Client` has no `Status` method, and none of the mcp symbols exist yet.

- [ ] **Step 3: Write the implementation**

Create `esi/status.go`:

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

Create `mcp/status.go`:

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

Modify `mcp/server.go`: append after the actual last-registered tool block at execution time
(immediately before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_status",
		Description: "Get the current EVE Online server status: player count, version, start time, and VIP mode (public data, no character required).",
	}, getStatusHandler(version))
```

Update `mcp/server.go`'s package doc comment: append `get_status` to the trailing tool list,
preserving the "and X registered" grammar.

Modify `mcp/server_test.go`: append `"get_status"` to the expected-tools slice, matching the
file's existing membership-check style.

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Status row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: increment the tool count by 1 from whatever it currently reads (spell out the
new number in words, matching the file's existing style), and append `get_status` to the `mcp/`
row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestStatus' -v` and
`go test ./mcp/... -run 'TestGetStatusHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (2 esi tests + 2 mcp tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, no panics on tool registration.

```bash
git add esi/status.go esi/status_test.go mcp/status.go mcp/status_internal_test.go \
  mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add status MCP tool (get_status: server player count, version, start time)"
```
