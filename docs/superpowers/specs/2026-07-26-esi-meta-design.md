# ESI Meta — Design Spec (twenty-fifth batch)

**Batch:** Meta tag, 4 endpoints, all public, no pagination. Meta-information about the ESI
service itself (its own changelog, name history, compatibility dates, and route health), not EVE
game data.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

- `GetMetaChangelog` → `*MetaChangelog{Changelog map[string][]MetaChangelogEntry}`, keyed by date
  string; each entry `{CompatibilityDate, Description, Method, Path, Type string}`.
- `GetMetaCompatibilityDates` → `*MetaCompatibilityDates{CompatibilityDates []string}`.
- `GetMetaName` → `*MetaName{Current string, History []MetaNameEntry{Date, Name string}}`.
- `GetMetaStatus` → `*MetaStatus{Routes []MetaStatusRoutestatus{Method, Path, Status string}}`.

All 4 have no security block in `openapi.yaml` (public) and no `.Page()` setter on their request
builders (not paginated) — verified directly from `esi/api_meta.go`.

**Novel type shape check**: `MetaChangelog.Changelog` is a `map[string][]MetaChangelogEntry` —
the first top-level **map**-typed field to appear as an MCP tool output anywhere in this project's
25 batches so far (every prior batch has used only structs/slices/pointers). Given this project's
hard-won lesson from the Killmails batch (an untested type shape — self-referential structs —
passed build/vet/gofmt but panicked at runtime in the MCP SDK's schema generator), this shape was
explicitly sandbox-verified with a live `mcpsdk.AddTool` registration test before finalizing the
design: a `map[string][]struct{...}` output field registers and serializes cleanly, zero
schema-generator errors. Confirmed safe to use as-is (a plain Go map, not a novel type).

**No recursion**: none of the 4 response models nests a slice of its own type. No Killmails-style
risk in this batch either.

## `esi/meta.go`

4 methods on `*Client`, one-file-per-tag as always. Given all 4 are public with identical
zero-argument signatures, this mirrors `esi/insurance.go`'s single-method public-tag shape,
extended to 4 methods.

```go
// Package esi's meta.go covers all 4 of "Meta"'s endpoints -- ESI's own changelog, compatibility
// dates, name history, and route health status. All 4 are public (no security block in
// esi/api/openapi.yaml) and not paginated (no .Page() setter on any of the 4 request builders).
// This is meta-information about the ESI service itself, not EVE game data.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// MetaChangelog returns ESI's own changelog, keyed by date.
func (c *Client) MetaChangelog(ctx context.Context) (*esiapi.MetaChangelog, error) {
	result, resp, err := c.api.MetaAPI.GetMetaChangelog(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta changelog: %w", describeError(err, resp))
	}
	return result, nil
}

// MetaCompatibilityDates returns every compatibility date ESI currently accepts.
func (c *Client) MetaCompatibilityDates(ctx context.Context) (*esiapi.MetaCompatibilityDates, error) {
	result, resp, err := c.api.MetaAPI.GetMetaCompatibilityDates(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta compatibility dates: %w", describeError(err, resp))
	}
	return result, nil
}

// MetaName returns ESI's current and historical names.
func (c *Client) MetaName(ctx context.Context) (*esiapi.MetaName, error) {
	result, resp, err := c.api.MetaAPI.GetMetaName(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta name: %w", describeError(err, resp))
	}
	return result, nil
}

// MetaStatus returns the health status of every ESI route.
func (c *Client) MetaStatus(ctx context.Context) (*esiapi.MetaStatus, error) {
	result, resp, err := c.api.MetaAPI.GetMetaStatus(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get meta status: %w", describeError(err, resp))
	}
	return result, nil
}
```

`esi/meta_test.go`: 8 tests (success + HTTP-error pair per method), following
`esi/insurance_test.go`'s exact style.

## `mcp/meta.go`

4 public tools, no `characterExists` gate at all (matching `mcp/insurance.go`'s
`list_insurance_prices` shape): `get_esi_changelog`, `list_esi_compatibility_dates`,
`get_esi_name`, `get_esi_status`.

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_esi_changelog (public) ---

type ChangelogEntry struct {
	CompatibilityDate string `json:"compatibilityDate"`
	Description       string `json:"description"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Type              string `json:"type"`
}

var getEsiChangelog = func(ctx context.Context, userAgent string) (map[string][]ChangelogEntry, error) {
	client := esi.NewPublicClient(userAgent)
	changelog, err := client.MetaChangelog(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]ChangelogEntry, len(changelog.Changelog))
	for date, entries := range changelog.Changelog {
		converted := make([]ChangelogEntry, len(entries))
		for i, e := range entries {
			converted[i] = ChangelogEntry{
				CompatibilityDate: e.CompatibilityDate, Description: e.Description,
				Method: e.Method, Path: e.Path, Type: e.Type,
			}
		}
		out[date] = converted
	}
	return out, nil
}

type GetEsiChangelogInput struct{}
type GetEsiChangelogOutput struct {
	Changelog map[string][]ChangelogEntry `json:"changelog"`
}

func getEsiChangelogHandler(version string) mcpsdk.ToolHandlerFor[GetEsiChangelogInput, GetEsiChangelogOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiChangelogInput) (*mcpsdk.CallToolResult, GetEsiChangelogOutput, error) {
		changelog, err := getEsiChangelog(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetEsiChangelogOutput{}, fmt.Errorf("get esi changelog: %w", err)
		}
		return nil, GetEsiChangelogOutput{Changelog: changelog}, nil
	}
}

// --- list_esi_compatibility_dates (public, no input) ---

var listEsiCompatibilityDates = func(ctx context.Context, userAgent string) ([]string, error) {
	client := esi.NewPublicClient(userAgent)
	dates, err := client.MetaCompatibilityDates(ctx)
	if err != nil {
		return nil, err
	}
	return dates.CompatibilityDates, nil
}

type ListEsiCompatibilityDatesInput struct{}
type ListEsiCompatibilityDatesOutput struct {
	CompatibilityDates []string `json:"compatibilityDates"`
}

func listEsiCompatibilityDatesHandler(version string) mcpsdk.ToolHandlerFor[ListEsiCompatibilityDatesInput, ListEsiCompatibilityDatesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListEsiCompatibilityDatesInput) (*mcpsdk.CallToolResult, ListEsiCompatibilityDatesOutput, error) {
		dates, err := listEsiCompatibilityDates(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListEsiCompatibilityDatesOutput{}, fmt.Errorf("list esi compatibility dates: %w", err)
		}
		return nil, ListEsiCompatibilityDatesOutput{CompatibilityDates: dates}, nil
	}
}

// --- get_esi_name (public, no input) ---

type NameEntry struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

var getEsiName = func(ctx context.Context, userAgent string) (string, []NameEntry, error) {
	client := esi.NewPublicClient(userAgent)
	name, err := client.MetaName(ctx)
	if err != nil {
		return "", nil, err
	}
	history := make([]NameEntry, len(name.History))
	for i, h := range name.History {
		history[i] = NameEntry{Date: h.Date, Name: h.Name}
	}
	return name.Current, history, nil
}

type GetEsiNameInput struct{}
type GetEsiNameOutput struct {
	Current string      `json:"current"`
	History []NameEntry `json:"history"`
}

func getEsiNameHandler(version string) mcpsdk.ToolHandlerFor[GetEsiNameInput, GetEsiNameOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiNameInput) (*mcpsdk.CallToolResult, GetEsiNameOutput, error) {
		current, history, err := getEsiName(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetEsiNameOutput{}, fmt.Errorf("get esi name: %w", err)
		}
		return nil, GetEsiNameOutput{Current: current, History: history}, nil
	}
}

// --- get_esi_status (public, no input) ---

type RouteStatus struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

var getEsiStatus = func(ctx context.Context, userAgent string) ([]RouteStatus, error) {
	client := esi.NewPublicClient(userAgent)
	status, err := client.MetaStatus(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RouteStatus, len(status.Routes))
	for i, r := range status.Routes {
		out[i] = RouteStatus{Method: r.Method, Path: r.Path, Status: r.Status}
	}
	return out, nil
}

type GetEsiStatusInput struct{}
type GetEsiStatusOutput struct {
	Routes []RouteStatus `json:"routes"`
}

func getEsiStatusHandler(version string) mcpsdk.ToolHandlerFor[GetEsiStatusInput, GetEsiStatusOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiStatusInput) (*mcpsdk.CallToolResult, GetEsiStatusOutput, error) {
		routes, err := getEsiStatus(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetEsiStatusOutput{}, fmt.Errorf("get esi status: %w", err)
		}
		return nil, GetEsiStatusOutput{Routes: routes}, nil
	}
}
```

`ChangelogEntry`, `GetEsiChangelogOutput`, `NameEntry`, `RouteStatus` are all flat, non-recursive
types. The one novel shape (`map[string][]ChangelogEntry`) was sandbox-verified with a live
`mcpsdk.AddTool` registration test before this spec was finalized — zero schema-generator errors.

`mcp/meta_internal_test.go`: 4 `MapsFields` tests (one per tool; no `UnknownCharacter` tests since
none of these tools gate on a character), following `mcp/insurance_internal_test.go`'s
swappable-var-override pattern, with full field-coverage assertions including the changelog map's
nested entries.

## `mcp/server.go` / `mcp/server_test.go`

Register 4 tools after the `list_structure_market_orders` block. Update package doc comment.
`server_test.go`'s expected-tools list: 157 → 161.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred fifty-seven" → "one hundred sixty-one",
  append the 4 new tool names.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Meta row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/meta_test.go`,
swappable-`var`-override tests for `mcp/meta_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.
