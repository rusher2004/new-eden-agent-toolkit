# ESI Insurance (Nineteenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the single "Insurance" ESI endpoint as a new `esi/insurance.go` facade and a new `mcp/insurance.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `InsuranceAPI`, reusing the existing `describeError` helper. One new `mcp/` tool file with 1 public tool. Same tiny size class as the Incursions batch.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `113b964`) — no new branch, no new PR.
- Current tool count baseline: **128**. After this plan: **129**.
- The `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`) — do not redeclare it.
- The tool is public — use `esi.NewPublicClient(userAgent string) *Client`, no `characterExists` check, no `characterId` input field. No pagination.

---

### Task 1: `esi/insurance.go` facade + `mcp/insurance.go` tool (single-file-pair batch)

**Files:**
- Create: `esi/insurance.go`
- Test: `esi/insurance_test.go`
- Create: `mcp/insurance.go`
- Test: `mcp/insurance_internal_test.go`
- Modify: `mcp/server.go` (register 1 new tool; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 128 to 129)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Insurance row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "one hundred twenty-eight" → "one hundred twenty-nine")
- Modify: `README.md` (same edit, mirrored)

Given the trivial size of this batch (1 endpoint, 1 tool), both the `esi/` facade and the `mcp/` tool are combined into a single task, mirroring the Incursions batch precedent.

- [ ] **Step 1: Write `esi/insurance.go`**

```go
// Package esi's insurance.go covers the single "Insurance" endpoint -- a public, game-wide
// listing of insurance prices per ship type, no auth required, not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// InsurancePrices returns insurance price levels for every insurable ship type.
func (c *Client) InsurancePrices(ctx context.Context) ([]esiapi.InsurancePricesGetInner, error) {
	result, resp, err := c.api.InsuranceAPI.GetInsurancePrices(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get insurance prices: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Write `esi/insurance_test.go`**

Read `esi/incursions_test.go` first to copy its exact public-client `httptest.Server` pattern. Cover:

- `TestInsurancePrices_ParsesResponse` — serve a 1-element array `[{type_id, levels: [{cost, name, payout}, {cost, name, payout}]}]` (at least 2 levels to exercise the nested slice), assert every field maps including both level entries.
- `TestInsurancePrices_HTTPError` — serve a 404, assert the error is non-nil and wraps a description.

- [ ] **Step 3: Write `mcp/insurance.go`**

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_insurance_prices (public, no input) ---

type InsuranceLevel struct {
	Cost   float64 `json:"cost"`
	Payout float64 `json:"payout"`
	Name   string  `json:"name"`
}

type InsurancePrice struct {
	TypeID int64            `json:"typeId"`
	Levels []InsuranceLevel `json:"levels"`
}

var listInsurancePrices = func(ctx context.Context, userAgent string) ([]InsurancePrice, error) {
	client := esi.NewPublicClient(userAgent)
	prices, err := client.InsurancePrices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]InsurancePrice, len(prices))
	for i, p := range prices {
		levels := make([]InsuranceLevel, len(p.Levels))
		for j, l := range p.Levels {
			levels[j] = InsuranceLevel{Cost: l.Cost, Payout: l.Payout, Name: l.Name}
		}
		out[i] = InsurancePrice{TypeID: p.TypeId, Levels: levels}
	}
	return out, nil
}

type ListInsurancePricesInput struct{}
type ListInsurancePricesOutput struct {
	Prices []InsurancePrice `json:"prices"`
}

func listInsurancePricesHandler(version string) mcpsdk.ToolHandlerFor[ListInsurancePricesInput, ListInsurancePricesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListInsurancePricesInput) (*mcpsdk.CallToolResult, ListInsurancePricesOutput, error) {
		prices, err := listInsurancePrices(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListInsurancePricesOutput{}, fmt.Errorf("list insurance prices: %w", err)
		}
		return nil, ListInsurancePricesOutput{Prices: prices}, nil
	}
}
```

Note: no `esiapi` import is needed in this file — `client.InsurancePrices(ctx)`'s return type is inferred at the call site; it's never named explicitly here.

- [ ] **Step 4: Write `mcp/insurance_internal_test.go`**

Follow `mcp/incursions_internal_test.go`'s pattern (fully public, no `config.Store`/`characterExists` involved). Write one test:

- `TestListInsurancePricesHandler_MapsFields` — stub `listInsurancePrices` to return a fully-populated `[]InsurancePrice` (including at least one entry with 2 `Levels`), assert every field maps.

- [ ] **Step 5: Run tests**

Run: `go test ./esi/... -run Insurance -v && go test ./mcp/... -run Insurance -v`
Expected: all tests PASS.

- [ ] **Step 6: Register the 1 tool in `mcp/server.go`**

Add this block immediately after the existing `list_industry_systems` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_insurance_prices",
		Description: "List insurance price levels for every insurable ship type (public data, no character required).",
	}, listInsurancePricesHandler(version))

	return server
}
```

Update the package doc comment above `NewServer` to append `list_insurance_prices` to its running
list (ending in `..., list_industry_systems, and list_insurance_prices registered.`).

- [ ] **Step 7: Update `mcp/server_test.go`**

Append this name to the `[]string{...}` literal, right after `"list_industry_systems",`:

```go
			"list_insurance_prices",
```

- [ ] **Step 8: Run the full test suite**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 129 tool names present).

- [ ] **Step 9: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "one hundred twenty-eight" to "one hundred twenty-nine".
- Append `list_insurance_prices` to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Insurance row from `Planned` to `This plan`.

- [ ] **Step 10: Commit**

```bash
git add esi/insurance.go esi/insurance_test.go mcp/insurance.go mcp/insurance_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add insurance MCP tool (1 tool: public listing)"
```

---

## Final Steps (after the task)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff 113b964..HEAD` — the design spec commit forward). Given the tiny size, this review can be lighter-weight than usual, but should still verify build/vet/test cleanliness and confirm the roadmap doc change was scoped correctly.
2. If it finds real issues, fix them and re-verify.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
