# ESI Incursions (Seventeenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the single "Incursions" ESI endpoint as a new `esi/incursions.go` facade and a new `mcp/incursions.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `IncursionsAPI`, reusing the existing `describeError` helper. One new `mcp/` tool file with 1 public tool. The smallest batch in this sweep.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `c32dc23`) — no new branch, no new PR.
- Current tool count baseline: **119**. After this plan: **120**.
- The `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`) — do not redeclare it.
- The tool is public — use `esi.NewPublicClient(userAgent string) *Client`, no `characterExists` check, no `characterId` input field. No pagination.

---

### Task 1: `esi/incursions.go` facade + `mcp/incursions.go` tool (single-file-pair batch)

**Files:**
- Create: `esi/incursions.go`
- Test: `esi/incursions_test.go`
- Create: `mcp/incursions.go`
- Test: `mcp/incursions_internal_test.go`
- Modify: `mcp/server.go` (register 1 new tool; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 119 to 120)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Incursions row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "one hundred nineteen" → "one hundred twenty")
- Modify: `README.md` (same edit, mirrored)

Given the trivial size of this batch (1 endpoint, 1 tool), both the `esi/` facade and the `mcp/` tool are combined into a single task rather than split across two, unlike every other batch in this sweep.

- [ ] **Step 1: Write `esi/incursions.go`**

```go
// Package esi's incursions.go covers the single "Incursions" endpoint -- a public, game-wide
// listing of active incursions, no auth required, not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Incursions returns every currently active incursion.
func (c *Client) Incursions(ctx context.Context) ([]esiapi.IncursionsGetInner, error) {
	result, resp, err := c.api.IncursionsAPI.GetIncursions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get incursions: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Write `esi/incursions_test.go`**

Read `esi/alliance_test.go` first to copy its exact public-client `httptest.Server` pattern. Cover:

- `TestIncursions_ParsesResponse` — serve a 1-element array `[{constellation_id, faction_id, has_boss, infested_solar_systems: [...], influence, staging_solar_system_id, state, type}]` fully populated, assert every field maps.
- `TestIncursions_HTTPError` — serve a 404, assert the error is non-nil and wraps a description.

- [ ] **Step 3: Write `mcp/incursions.go`**

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_incursions (public, no input) ---

type Incursion struct {
	ConstellationID        int64   `json:"constellationId"`
	FactionID              int64   `json:"factionId"`
	HasBoss                bool    `json:"hasBoss"`
	InfestedSolarSystemIDs []int64 `json:"infestedSolarSystemIds"`
	Influence              float64 `json:"influence"`
	StagingSolarSystemID   int64   `json:"stagingSolarSystemId"`
	State                  string  `json:"state"`
	Type                   string  `json:"type"`
}

var listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
	client := esi.NewPublicClient(userAgent)
	incursions, err := client.Incursions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Incursion, len(incursions))
	for i, inc := range incursions {
		out[i] = Incursion{
			ConstellationID: inc.ConstellationId, FactionID: inc.FactionId, HasBoss: inc.HasBoss,
			InfestedSolarSystemIDs: inc.InfestedSolarSystems, Influence: inc.Influence,
			StagingSolarSystemID: inc.StagingSolarSystemId, State: inc.State, Type: inc.Type,
		}
	}
	return out, nil
}

type ListIncursionsInput struct{}
type ListIncursionsOutput struct {
	Incursions []Incursion `json:"incursions"`
}

func listIncursionsHandler(version string) mcpsdk.ToolHandlerFor[ListIncursionsInput, ListIncursionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIncursionsInput) (*mcpsdk.CallToolResult, ListIncursionsOutput, error) {
		incursions, err := listIncursions(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListIncursionsOutput{}, fmt.Errorf("list incursions: %w", err)
		}
		return nil, ListIncursionsOutput{Incursions: incursions}, nil
	}
}
```

- [ ] **Step 4: Write `mcp/incursions_internal_test.go`**

Follow `mcp/alliance_internal_test.go`'s pattern (fully public, no `config.Store`/`characterExists` involved). Write one test:

- `TestListIncursionsHandler_MapsFields` — stub `listIncursions` to return a fully-populated `[]Incursion` (including a non-empty `InfestedSolarSystemIDs`), assert every field maps.

- [ ] **Step 5: Run tests**

Run: `go test ./esi/... -run Incursions -v && go test ./mcp/... -run Incursions -v`
Expected: all tests PASS.

- [ ] **Step 6: Register the 1 tool in `mcp/server.go`**

Add this block immediately after the existing `list_freelance_jobs` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_incursions",
		Description: "List every currently active incursion (public data, no character required).",
	}, listIncursionsHandler(version))

	return server
}
```

Update the package doc comment above `NewServer` to append `list_incursions` to its running list
(ending in `..., list_freelance_jobs, and list_incursions registered.`).

- [ ] **Step 7: Update `mcp/server_test.go`**

Append this name to the `[]string{...}` literal, right after `"list_freelance_jobs",`:

```go
			"list_incursions",
```

- [ ] **Step 8: Run the full test suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 120 tool names present).

- [ ] **Step 9: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "one hundred nineteen" to "one hundred twenty".
- Append `list_incursions` to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Incursions row from `Planned` to `This plan`.

- [ ] **Step 10: Commit**

```bash
git add esi/incursions.go esi/incursions_test.go mcp/incursions.go mcp/incursions_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add incursions MCP tool (1 tool: public listing)"
```

---

## Final Steps (after the task)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff c32dc23..HEAD` — the design spec commit forward). Given the tiny size, this review can be lighter-weight than usual, but should still verify build/vet/test cleanliness and confirm the roadmap doc change was scoped correctly.
2. If it finds real issues, fix them and re-verify.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
