# ESI Faction Warfare (Thirteenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 8 endpoints of the "Faction Warfare" ESI tag as a new `esi/faction_warfare.go` facade and a new `mcp/faction_warfare.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `FactionWarfareAPI`, reusing the existing `describeError` helper. One new `mcp/` tool file with 2 private tools (existence-checked via the existing `characterExists`) and 6 public tools (`esi.NewPublicClient`). Flattens 18 distinct-but-identically-shaped generated "leaderboard entry" types into 3 reusable MCP types, and 6 distinct-but-identically-shaped generated "Kills"/"VictoryPoints" stats types into 1 reusable MCP type — mirroring the Corporation Projects batch's precedent for flattening generated-package type duplication.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `5e8b125`) — no new branch, no new PR.
- Current tool count baseline: **88**. After this plan: **96**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`) — do not redeclare it.
- `CharacterFwStats` takes **no** `characterID` parameter — it uses the `Client`'s own baked-in `characterID` field (via `int64(c.characterID)`), matching the existing "my own character" convention already used throughout `esi/` (e.g. `esi/access_list.go`, `esi/calendar.go`), NOT the Corporation Projects "explicit target character" pattern.
- `CorporationFwStats` is corp-scoped: takes `corporationID int64`, and the `mcp/` tool takes both `characterId` (whose stored auth token to use) and `corporationId` (the target), per the established corp-scoped convention.
- No pagination on any of these 8 endpoints — verified (no `.Page()` builder method on any of the 8 generated request builders).
- The 3 leaderboard entry types (`LeaderboardFactionEntry`, `LeaderboardCharacterEntry`, `LeaderboardCorporationEntry`) and 3 metric wrapper types (`FactionLeaderboardMetric`, `CharacterLeaderboardMetric`, `CorporationLeaderboardMetric`) and the shared `KillsOrPointsStats` type are each defined exactly once in `mcp/faction_warfare.go` — do not redeclare per-endpoint.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean). Implementers should still run these themselves as part of each task.

---

### Task 1: `esi/faction_warfare.go` facade (8 methods)

**Files:**
- Create: `esi/faction_warfare.go`
- Test: `esi/faction_warfare_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `Client.api`/`Client.characterID` (`esi/client.go`).
- Produces: 8 methods on `*Client` for Task 2's `mcp/faction_warfare.go` to call.

- [ ] **Step 1: Write `esi/faction_warfare.go`**

```go
// Package esi's faction_warfare.go covers all 8 of "Faction Warfare"'s endpoints. 2 are private
// (CharacterFwStats -- "my own" character, corp-scoped CorporationFwStats), 6 are public
// (leaderboards x3, per-faction stats, systems, wars).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterFwStats returns the client's own character's faction warfare stats. Uses the Client's
// own baked-in characterID -- unlike Corporation Projects' Contribution endpoint, this ESI
// endpoint is genuinely "my own stats," not "look up an arbitrary character."
func (c *Client) CharacterFwStats(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetCharactersCharacterIdFwStats(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character fw stats: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationFwStats returns corporationID's faction warfare stats. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise.
func (c *Client) CorporationFwStats(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetCorporationsCorporationIdFwStats(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation fw stats: %w", describeError(err, resp))
	}
	return result, nil
}

// FwLeaderboards returns the top-4 faction rankings by kills and victory points.
func (c *Client) FwLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwLeaderboards(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw leaderboards: %w", describeError(err, resp))
	}
	return result, nil
}

// FwCharacterLeaderboards returns the top-100 pilot rankings by kills and victory points.
func (c *Client) FwCharacterLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwLeaderboardsCharacters(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw character leaderboards: %w", describeError(err, resp))
	}
	return result, nil
}

// FwCorporationLeaderboards returns the top-10 corporation rankings by kills and victory points.
func (c *Client) FwCorporationLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwLeaderboardsCorporations(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw corporation leaderboards: %w", describeError(err, resp))
	}
	return result, nil
}

// FwStats returns per-faction faction warfare stats for every faction.
func (c *Client) FwStats(ctx context.Context) ([]esiapi.FwStatsGetInner, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwStats(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw stats: %w", describeError(err, resp))
	}
	return result, nil
}

// FwSystems returns every solar system's faction warfare occupation/contest state.
func (c *Client) FwSystems(ctx context.Context) ([]esiapi.FwSystemsGetInner, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwSystems(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw systems: %w", describeError(err, resp))
	}
	return result, nil
}

// FwWars returns every faction warfare declaration (which factions are at war with each other).
func (c *Client) FwWars(ctx context.Context) ([]esiapi.FwWarsGetInner, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwWars(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw wars: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Write `esi/faction_warfare_test.go`**

Read `esi/contracts_test.go` (for the private/authenticated-client harness, `ExportedNewClientWithOptions`) and `esi/alliance_test.go` (for the public-client harness, `ExportedNewPublicClientWithOptions`) first to copy their exact patterns. Cover:

- `TestCharacterFwStats_ParsesResponse` — serve `{current_rank, enlisted_on, faction_id, highest_rank, kills: {last_week, total, yesterday}, victory_points: {last_week, total, yesterday}}` fully populated, assert every field maps, using the AUTHENTICATED client harness (this is a private/"my own character" endpoint) and assert the request path includes the test client's own characterID.
- `TestCharacterFwStats_HTTPError` — serve a 404, assert the error is non-nil and wraps a description.
- `TestCorporationFwStats_ParsesResponse` — serve `{enlisted_on, faction_id, kills, pilots, victory_points}` fully populated, assert every field maps.
- `TestFwLeaderboards_ParsesResponse` — serve `{kills: {active_total: [{amount, faction_id}], last_week: [...], yesterday: [...]}, victory_points: {...}}`, assert every field of at least one entry in each of the 6 slices maps.
- `TestFwCharacterLeaderboards_ParsesResponse` — same shape, `character_id` instead of `faction_id`, assert mapping.
- `TestFwCorporationLeaderboards_ParsesResponse` — same shape, `corporation_id` instead of `faction_id`, assert mapping.
- `TestFwStats_ParsesResponse` — serve a 1-element array `[{faction_id, kills, pilots, systems_controlled, victory_points}]`, assert every field maps.
- `TestFwSystems_ParsesResponse` — serve a 1-element array `[{contested, occupier_faction_id, owner_faction_id, solar_system_id, victory_points, victory_points_threshold}]`, assert every field maps.
- `TestFwWars_ParsesResponse` — serve a 1-element array `[{against_id, faction_id}]`, assert both fields map.

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run FactionWarfare -v` and also `go test ./esi/... -run "^TestCharacterFwStats|^TestCorporationFwStats|^TestFw" -v` (test names don't share a common `FactionWarfare` prefix — use the method-name-based pattern to run them all).
Expected: all tests PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/faction_warfare.go esi/faction_warfare_test.go
git commit -m "feat: add esi/faction_warfare.go facade"
```

---

### Task 2: `mcp/faction_warfare.go` tools (8 tools), server registration, and docs sync

**Files:**
- Create: `mcp/faction_warfare.go`
- Test: `mcp/faction_warfare_internal_test.go`
- Modify: `mcp/server.go` (register 8 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 88 to 96)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Faction Warfare row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "eighty-eight" → "ninety-six")
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: all 8 `esi/faction_warfare.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent`.
- Produces: 8 `mcpsdk.ToolHandlerFor[...]`-returning functions for `mcp/server.go`: `getCharacterFwStatsHandler`, `getCorporationFwStatsHandler` (both `(version, clientID string, store *config.Store)`), `getFwLeaderboardsHandler`, `getFwCharacterLeaderboardsHandler`, `getFwCorporationLeaderboardsHandler`, `listFwStatsHandler`, `listFwSystemsHandler`, `listFwWarsHandler` (all `(version string)` only).

- [ ] **Step 1: Write `mcp/faction_warfare.go`**

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// KillsOrPointsStats is the shared {last_week, total, yesterday} shape used by every FW
// endpoint's Kills/VictoryPoints sub-fields (character, corporation, and per-faction stats) --
// 6 distinct generated Go types collapse to this one identical shape, verified from source.
type KillsOrPointsStats struct {
	LastWeek  int64 `json:"lastWeek"`
	Total     int64 `json:"total"`
	Yesterday int64 `json:"yesterday"`
}

// --- get_character_fw_stats (private) ---

var getCharacterFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterFwStats(ctx)
}

type GetCharacterFwStatsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterFwStatsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CurrentRank   *int64             `json:"currentRank,omitempty"`
	HighestRank   *int64             `json:"highestRank,omitempty"`
	FactionID     *int64             `json:"factionId,omitempty"`
	EnlistedOn    *time.Time         `json:"enlistedOn,omitempty"`
	Kills         KillsOrPointsStats `json:"kills"`
	VictoryPoints KillsOrPointsStats `json:"victoryPoints"`
}

func getCharacterFwStatsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterFwStatsInput, GetCharacterFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFwStatsInput) (*mcpsdk.CallToolResult, GetCharacterFwStatsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFwStatsOutput{}, err
		}
		if !found {
			return nil, GetCharacterFwStatsOutput{}, fmt.Errorf("get character fw stats for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		stats, err := getCharacterFwStats(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterFwStatsOutput{}, fmt.Errorf("get character fw stats: %w", err)
		}
		return nil, GetCharacterFwStatsOutput{
			CharacterID: in.CharacterID, CurrentRank: stats.CurrentRank, HighestRank: stats.HighestRank,
			FactionID: stats.FactionId, EnlistedOn: stats.EnlistedOn,
			Kills:         KillsOrPointsStats{LastWeek: stats.Kills.LastWeek, Total: stats.Kills.Total, Yesterday: stats.Kills.Yesterday},
			VictoryPoints: KillsOrPointsStats{LastWeek: stats.VictoryPoints.LastWeek, Total: stats.VictoryPoints.Total, Yesterday: stats.VictoryPoints.Yesterday},
		}, nil
	}
}

// --- get_corporation_fw_stats (private) ---

var getCorporationFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationFwStats(ctx, corporationID)
}

type GetCorporationFwStatsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type GetCorporationFwStatsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	FactionID     *int64             `json:"factionId,omitempty"`
	Pilots        *int64             `json:"pilots,omitempty"`
	EnlistedOn    *time.Time         `json:"enlistedOn,omitempty"`
	Kills         KillsOrPointsStats `json:"kills"`
	VictoryPoints KillsOrPointsStats `json:"victoryPoints"`
}

func getCorporationFwStatsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationFwStatsInput, GetCorporationFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationFwStatsInput) (*mcpsdk.CallToolResult, GetCorporationFwStatsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationFwStatsOutput{}, err
		}
		if !found {
			return nil, GetCorporationFwStatsOutput{}, fmt.Errorf("get corporation fw stats for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		stats, err := getCorporationFwStats(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationFwStatsOutput{}, fmt.Errorf("get corporation fw stats: %w", err)
		}
		return nil, GetCorporationFwStatsOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, FactionID: stats.FactionId,
			Pilots: stats.Pilots, EnlistedOn: stats.EnlistedOn,
			Kills:         KillsOrPointsStats{LastWeek: stats.Kills.LastWeek, Total: stats.Kills.Total, Yesterday: stats.Kills.Yesterday},
			VictoryPoints: KillsOrPointsStats{LastWeek: stats.VictoryPoints.LastWeek, Total: stats.VictoryPoints.Total, Yesterday: stats.VictoryPoints.Yesterday},
		}, nil
	}
}

// --- get_fw_leaderboards (public, no input) ---

type LeaderboardFactionEntry struct {
	FactionID *int64 `json:"factionId,omitempty"`
	Amount    *int64 `json:"amount,omitempty"`
}
type FactionLeaderboardMetric struct {
	ActiveTotal []LeaderboardFactionEntry `json:"activeTotal"`
	LastWeek    []LeaderboardFactionEntry `json:"lastWeek"`
	Yesterday   []LeaderboardFactionEntry `json:"yesterday"`
}

var getFwLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwLeaderboards(ctx)
}

type GetFwLeaderboardsInput struct{}
type GetFwLeaderboardsOutput struct {
	Kills         FactionLeaderboardMetric `json:"kills"`
	VictoryPoints FactionLeaderboardMetric `json:"victoryPoints"`
}

func getFwLeaderboardsHandler(version string) mcpsdk.ToolHandlerFor[GetFwLeaderboardsInput, GetFwLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwLeaderboardsOutput, error) {
		lb, err := getFwLeaderboards(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetFwLeaderboardsOutput{}, fmt.Errorf("get fw leaderboards: %w", err)
		}
		killsActiveTotal := make([]LeaderboardFactionEntry, len(lb.Kills.ActiveTotal))
		for i, v := range lb.Kills.ActiveTotal {
			killsActiveTotal[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		killsLastWeek := make([]LeaderboardFactionEntry, len(lb.Kills.LastWeek))
		for i, v := range lb.Kills.LastWeek {
			killsLastWeek[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		killsYesterday := make([]LeaderboardFactionEntry, len(lb.Kills.Yesterday))
		for i, v := range lb.Kills.Yesterday {
			killsYesterday[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		vpActiveTotal := make([]LeaderboardFactionEntry, len(lb.VictoryPoints.ActiveTotal))
		for i, v := range lb.VictoryPoints.ActiveTotal {
			vpActiveTotal[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		vpLastWeek := make([]LeaderboardFactionEntry, len(lb.VictoryPoints.LastWeek))
		for i, v := range lb.VictoryPoints.LastWeek {
			vpLastWeek[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		vpYesterday := make([]LeaderboardFactionEntry, len(lb.VictoryPoints.Yesterday))
		for i, v := range lb.VictoryPoints.Yesterday {
			vpYesterday[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		return nil, GetFwLeaderboardsOutput{
			Kills:         FactionLeaderboardMetric{ActiveTotal: killsActiveTotal, LastWeek: killsLastWeek, Yesterday: killsYesterday},
			VictoryPoints: FactionLeaderboardMetric{ActiveTotal: vpActiveTotal, LastWeek: vpLastWeek, Yesterday: vpYesterday},
		}, nil
	}
}

// --- get_fw_character_leaderboards (public, no input) ---

type LeaderboardCharacterEntry struct {
	CharacterID *int64 `json:"characterId,omitempty"`
	Amount      *int64 `json:"amount,omitempty"`
}
type CharacterLeaderboardMetric struct {
	ActiveTotal []LeaderboardCharacterEntry `json:"activeTotal"`
	LastWeek    []LeaderboardCharacterEntry `json:"lastWeek"`
	Yesterday   []LeaderboardCharacterEntry `json:"yesterday"`
}

var getFwCharacterLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCharactersGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwCharacterLeaderboards(ctx)
}

type GetFwCharacterLeaderboardsInput struct{}
type GetFwCharacterLeaderboardsOutput struct {
	Kills         CharacterLeaderboardMetric `json:"kills"`
	VictoryPoints CharacterLeaderboardMetric `json:"victoryPoints"`
}

func getFwCharacterLeaderboardsHandler(version string) mcpsdk.ToolHandlerFor[GetFwCharacterLeaderboardsInput, GetFwCharacterLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwCharacterLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwCharacterLeaderboardsOutput, error) {
		lb, err := getFwCharacterLeaderboards(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetFwCharacterLeaderboardsOutput{}, fmt.Errorf("get fw character leaderboards: %w", err)
		}
		killsActiveTotal := make([]LeaderboardCharacterEntry, len(lb.Kills.ActiveTotal))
		for i, v := range lb.Kills.ActiveTotal {
			killsActiveTotal[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		killsLastWeek := make([]LeaderboardCharacterEntry, len(lb.Kills.LastWeek))
		for i, v := range lb.Kills.LastWeek {
			killsLastWeek[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		killsYesterday := make([]LeaderboardCharacterEntry, len(lb.Kills.Yesterday))
		for i, v := range lb.Kills.Yesterday {
			killsYesterday[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		vpActiveTotal := make([]LeaderboardCharacterEntry, len(lb.VictoryPoints.ActiveTotal))
		for i, v := range lb.VictoryPoints.ActiveTotal {
			vpActiveTotal[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		vpLastWeek := make([]LeaderboardCharacterEntry, len(lb.VictoryPoints.LastWeek))
		for i, v := range lb.VictoryPoints.LastWeek {
			vpLastWeek[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		vpYesterday := make([]LeaderboardCharacterEntry, len(lb.VictoryPoints.Yesterday))
		for i, v := range lb.VictoryPoints.Yesterday {
			vpYesterday[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		return nil, GetFwCharacterLeaderboardsOutput{
			Kills:         CharacterLeaderboardMetric{ActiveTotal: killsActiveTotal, LastWeek: killsLastWeek, Yesterday: killsYesterday},
			VictoryPoints: CharacterLeaderboardMetric{ActiveTotal: vpActiveTotal, LastWeek: vpLastWeek, Yesterday: vpYesterday},
		}, nil
	}
}

// --- get_fw_corporation_leaderboards (public, no input) ---

type LeaderboardCorporationEntry struct {
	CorporationID *int64 `json:"corporationId,omitempty"`
	Amount        *int64 `json:"amount,omitempty"`
}
type CorporationLeaderboardMetric struct {
	ActiveTotal []LeaderboardCorporationEntry `json:"activeTotal"`
	LastWeek    []LeaderboardCorporationEntry `json:"lastWeek"`
	Yesterday   []LeaderboardCorporationEntry `json:"yesterday"`
}

var getFwCorporationLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCorporationsGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwCorporationLeaderboards(ctx)
}

type GetFwCorporationLeaderboardsInput struct{}
type GetFwCorporationLeaderboardsOutput struct {
	Kills         CorporationLeaderboardMetric `json:"kills"`
	VictoryPoints CorporationLeaderboardMetric `json:"victoryPoints"`
}

func getFwCorporationLeaderboardsHandler(version string) mcpsdk.ToolHandlerFor[GetFwCorporationLeaderboardsInput, GetFwCorporationLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwCorporationLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwCorporationLeaderboardsOutput, error) {
		lb, err := getFwCorporationLeaderboards(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetFwCorporationLeaderboardsOutput{}, fmt.Errorf("get fw corporation leaderboards: %w", err)
		}
		killsActiveTotal := make([]LeaderboardCorporationEntry, len(lb.Kills.ActiveTotal))
		for i, v := range lb.Kills.ActiveTotal {
			killsActiveTotal[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		killsLastWeek := make([]LeaderboardCorporationEntry, len(lb.Kills.LastWeek))
		for i, v := range lb.Kills.LastWeek {
			killsLastWeek[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		killsYesterday := make([]LeaderboardCorporationEntry, len(lb.Kills.Yesterday))
		for i, v := range lb.Kills.Yesterday {
			killsYesterday[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		vpActiveTotal := make([]LeaderboardCorporationEntry, len(lb.VictoryPoints.ActiveTotal))
		for i, v := range lb.VictoryPoints.ActiveTotal {
			vpActiveTotal[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		vpLastWeek := make([]LeaderboardCorporationEntry, len(lb.VictoryPoints.LastWeek))
		for i, v := range lb.VictoryPoints.LastWeek {
			vpLastWeek[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		vpYesterday := make([]LeaderboardCorporationEntry, len(lb.VictoryPoints.Yesterday))
		for i, v := range lb.VictoryPoints.Yesterday {
			vpYesterday[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		return nil, GetFwCorporationLeaderboardsOutput{
			Kills:         CorporationLeaderboardMetric{ActiveTotal: killsActiveTotal, LastWeek: killsLastWeek, Yesterday: killsYesterday},
			VictoryPoints: CorporationLeaderboardMetric{ActiveTotal: vpActiveTotal, LastWeek: vpLastWeek, Yesterday: vpYesterday},
		}, nil
	}
}

// --- list_fw_stats (public, no input) ---

type FwFactionStats struct {
	FactionID         int64              `json:"factionId"`
	Pilots            int64              `json:"pilots"`
	SystemsControlled int64              `json:"systemsControlled"`
	Kills             KillsOrPointsStats `json:"kills"`
	VictoryPoints     KillsOrPointsStats `json:"victoryPoints"`
}

var listFwStats = func(ctx context.Context, userAgent string) ([]esiapi.FwStatsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwStats(ctx)
}

type ListFwStatsInput struct{}
type ListFwStatsOutput struct {
	Stats []FwFactionStats `json:"stats"`
}

func listFwStatsHandler(version string) mcpsdk.ToolHandlerFor[ListFwStatsInput, ListFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwStatsInput) (*mcpsdk.CallToolResult, ListFwStatsOutput, error) {
		stats, err := listFwStats(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListFwStatsOutput{}, fmt.Errorf("list fw stats: %w", err)
		}
		out := make([]FwFactionStats, len(stats))
		for i, s := range stats {
			out[i] = FwFactionStats{
				FactionID: s.FactionId, Pilots: s.Pilots, SystemsControlled: s.SystemsControlled,
				Kills:         KillsOrPointsStats{LastWeek: s.Kills.LastWeek, Total: s.Kills.Total, Yesterday: s.Kills.Yesterday},
				VictoryPoints: KillsOrPointsStats{LastWeek: s.VictoryPoints.LastWeek, Total: s.VictoryPoints.Total, Yesterday: s.VictoryPoints.Yesterday},
			}
		}
		return nil, ListFwStatsOutput{Stats: out}, nil
	}
}

// --- list_fw_systems (public, no input) ---

type FwSystem struct {
	Contested              string `json:"contested"`
	OccupierFactionID      int64  `json:"occupierFactionId"`
	OwnerFactionID         int64  `json:"ownerFactionId"`
	SolarSystemID          int64  `json:"solarSystemId"`
	VictoryPoints          int64  `json:"victoryPoints"`
	VictoryPointsThreshold int64  `json:"victoryPointsThreshold"`
}

var listFwSystems = func(ctx context.Context, userAgent string) ([]esiapi.FwSystemsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwSystems(ctx)
}

type ListFwSystemsInput struct{}
type ListFwSystemsOutput struct {
	Systems []FwSystem `json:"systems"`
}

func listFwSystemsHandler(version string) mcpsdk.ToolHandlerFor[ListFwSystemsInput, ListFwSystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwSystemsInput) (*mcpsdk.CallToolResult, ListFwSystemsOutput, error) {
		systems, err := listFwSystems(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListFwSystemsOutput{}, fmt.Errorf("list fw systems: %w", err)
		}
		out := make([]FwSystem, len(systems))
		for i, s := range systems {
			out[i] = FwSystem{
				Contested: s.Contested, OccupierFactionID: s.OccupierFactionId, OwnerFactionID: s.OwnerFactionId,
				SolarSystemID: s.SolarSystemId, VictoryPoints: s.VictoryPoints, VictoryPointsThreshold: s.VictoryPointsThreshold,
			}
		}
		return nil, ListFwSystemsOutput{Systems: out}, nil
	}
}

// --- list_fw_wars (public, no input) ---

type FwWar struct {
	AgainstID int64 `json:"againstId"`
	FactionID int64 `json:"factionId"`
}

var listFwWars = func(ctx context.Context, userAgent string) ([]esiapi.FwWarsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwWars(ctx)
}

type ListFwWarsInput struct{}
type ListFwWarsOutput struct {
	Wars []FwWar `json:"wars"`
}

func listFwWarsHandler(version string) mcpsdk.ToolHandlerFor[ListFwWarsInput, ListFwWarsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwWarsInput) (*mcpsdk.CallToolResult, ListFwWarsOutput, error) {
		wars, err := listFwWars(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListFwWarsOutput{}, fmt.Errorf("list fw wars: %w", err)
		}
		out := make([]FwWar, len(wars))
		for i, w := range wars {
			out[i] = FwWar{AgainstID: w.AgainstId, FactionID: w.FactionId}
		}
		return nil, ListFwWarsOutput{Wars: out}, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Write `mcp/faction_warfare_internal_test.go`**

Follow `mcp/corporation_internal_test.go`'s pattern (package-level `var` override + `t.Cleanup`; `keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). Cover:

- `TestGetCharacterFwStatsHandler_MapsFields` — stub returns a fully-populated `*esiapi.CharactersCharacterIdFwStatsGet`, assert every output field maps.
- `TestGetCharacterFwStatsHandler_UnknownCharacter`
- `TestGetCorporationFwStatsHandler_MapsFields` — stub returns a fully-populated `*esiapi.CorporationsCorporationIdFwStatsGet`, assert every field maps.
- `TestGetCorporationFwStatsHandler_UnknownCharacter`
- `TestGetFwLeaderboardsHandler_MapsFields` — stub returns a `*esiapi.FwLeaderboardsGet` with at least one entry in each of the 6 slices, assert every entry's `FactionId`/`Amount` maps into the correct `ActiveTotal`/`LastWeek`/`Yesterday` bucket of the correct `Kills`/`VictoryPoints` metric.
- `TestGetFwCharacterLeaderboardsHandler_MapsFields` — same shape check with `CharacterId`.
- `TestGetFwCorporationLeaderboardsHandler_MapsFields` — same shape check with `CorporationId`.
- `TestListFwStatsHandler_MapsFields` — stub returns a 1-element `[]esiapi.FwStatsGetInner`, assert every field maps.
- `TestListFwSystemsHandler_MapsFields` — stub returns a 1-element `[]esiapi.FwSystemsGetInner`, assert every field maps.
- `TestListFwWarsHandler_MapsFields` — stub returns a 1-element `[]esiapi.FwWarsGetInner`, assert both fields map.

That's 10 test functions total (2 tools get an extra unknown-character test each; the other 6 are public and don't need one).

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run "^TestGetCharacterFwStats|^TestGetCorporationFwStats|^TestGetFw|^TestListFw" -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 8 tools in `mcp/server.go`**

Add these 8 `mcpsdk.AddTool` blocks immediately after the existing `get_dogma_effect` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_fw_stats",
		Description: "Get a logged-in EVE Online character's faction warfare stats.",
	}, getCharacterFwStatsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_fw_stats",
		Description: "Get a corporation's faction warfare stats, using a logged-in character's roles to authorize.",
	}, getCorporationFwStatsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fw_leaderboards",
		Description: "Get the top faction rankings by kills and victory points (public data, no character required).",
	}, getFwLeaderboardsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fw_character_leaderboards",
		Description: "Get the top pilot rankings by kills and victory points (public data, no character required).",
	}, getFwCharacterLeaderboardsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_fw_corporation_leaderboards",
		Description: "Get the top corporation rankings by kills and victory points (public data, no character required).",
	}, getFwCorporationLeaderboardsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fw_stats",
		Description: "List every faction's faction warfare stats (public data, no character required).",
	}, listFwStatsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fw_systems",
		Description: "List every solar system's faction warfare occupation/contest state (public data, no character required).",
	}, listFwSystemsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_fw_wars",
		Description: "List every faction warfare declaration (public data, no character required).",
	}, listFwWarsHandler(version))

	return server
}
```

Update the package doc comment above `NewServer` to append the 8 new tool names to its running
list (ending in `..., get_dogma_effect, get_character_fw_stats, get_corporation_fw_stats,
get_fw_leaderboards, get_fw_character_leaderboards, get_fw_corporation_leaderboards,
list_fw_stats, list_fw_systems, and list_fw_wars registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 8 names to the `[]string{...}` literal, right after `"get_dogma_effect",`:

```go
			"get_character_fw_stats", "get_corporation_fw_stats", "get_fw_leaderboards",
			"get_fw_character_leaderboards", "get_fw_corporation_leaderboards",
			"list_fw_stats", "list_fw_systems", "list_fw_wars",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 96 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "eighty-eight" to "ninety-six".
- Append the 8 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Faction Warfare row from `Planned` to `This plan`.

- [ ] **Step 8: Commit**

```bash
git add mcp/faction_warfare.go mcp/faction_warfare_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add faction warfare MCP tools (8 tools: mixed public/private)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff 5e8b125..HEAD` — the design spec commit forward).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
