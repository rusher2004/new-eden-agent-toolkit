# ESI Corporation Part 1 (Ninth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the first 14 of Corporation's 22 endpoints (basics + membership/roles) as a new `esi/corporation.go` facade and a new `mcp/corporation.go` tool file, registered on the shared MCP server.

**Architecture:** One `esi/` facade file (`esi/corporation.go`) wrapping the generated `CorporationAPI`, reusing the package's existing `describeError` (`esi/error.go`) and `totalPagesFromHeader` (`esi/contracts.go`) helpers — do not redeclare either. One `mcp/` tool file (`mcp/corporation.go`) with 4 public tools (`esi.NewPublicClient`, no existence check) and 10 private tools (authenticated client + the existing `characterExists` helper from `mcp/character.go`, reused not redeclared). `list_corporation_standings` reuses the existing `Standing` type from `mcp/character.go` directly.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `8315a56`) — no new branch, no new PR. Every commit lands on this branch.
- Current tool count baseline: **57** (registered in `mcp/server.go`, asserted in `mcp/server_test.go`). After this plan: **71**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError(err error, resp *http.Response) error` (`esi/error.go`) before wrapping with `fmt.Errorf`.
- Every paginated `esi/` method takes `page int32` where `0` means "don't call `.Page()`"; returns `(items, totalPages int32, error)` using the existing `totalPagesFromHeader(resp *http.Response) int32` from `esi/contracts.go` — do not redeclare it.
- Public tools (no character/auth needed) use `esi.NewPublicClient(userAgent string) *Client` and take no `characterId` input. Private tools use `esi.NewClient(ctx, clientID, characterID, userAgent) (*Client, error)` plus the existing `characterExists(store *config.Store, characterID int32) (bool, error)` helper from `mcp/character.go` — reuse it, do not redeclare it. On not-found, return `fmt.Errorf("<action> for character %d: %w", characterID, ErrCharacterNotFound)` where `ErrCharacterNotFound` is the existing sentinel already defined in `mcp/character.go`.
- `list_corporation_standings` reuses the existing `Standing{FromID int64, FromType string, Standing float64}` type from `mcp/character.go` — do not redeclare it.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet` both clean). Implementers should still run `go build ./...` and `go vet ./...` themselves as part of each task — this is not a substitute for that, just a guarantee the code is correct as written.
- Per standing project convention (see `docs/superpowers/specs/2026-07-26-esi-corporation-part1-design.md`), the roadmap doc's Corporation row stays `Planned` in this plan — it only flips to "This plan" once Part 2 (assets/infrastructure, a later batch) also lands. Do not edit that row in this plan's docs task.

---

### Task 1: `esi/corporation.go` facade (14 read methods)

**Files:**
- Create: `esi/corporation.go`
- Test: `esi/corporation_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `totalPagesFromHeader(resp *http.Response) int32` (`esi/contracts.go`), `Client.api` (`esi/client.go`, `*esiapi.APIClient`), `Client.characterID` (unused here — none of these 14 endpoints are character-scoped, only corp-scoped/public).
- Produces: 14 methods on `*Client` — exact names and signatures below — for Task 2's `mcp/corporation.go` to call.

- [ ] **Step 1: Write `esi/corporation.go`**

```go
// Package esi's corporation.go covers the first 14 of Corporation's 22 endpoints (basics and
// membership/roles). The remaining 8 (blueprints, container logs, facilities, medals, starbases,
// structures) are assets/infrastructure-flavored and deferred to a later "Part 2" batch — see
// docs/superpowers/specs/2026-07-26-esi-corporation-part1-design.md.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CorporationDetail returns corporationID's basic public info.
func (c *Client) CorporationDetail(ctx context.Context, corporationID int64) (*esiapi.CorporationsDetail, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationId(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAllianceHistory returns corporationID's alliance membership history.
func (c *Client) CorporationAllianceHistory(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdAlliancehistoryGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdAlliancehistory(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation alliance history: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationIcons returns corporationID's icon URLs.
func (c *Client) CorporationIcons(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdIconsGet, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdIcons(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation icons: %w", describeError(err, resp))
	}
	return result, nil
}

// NPCCorporations returns the IDs of every NPC corporation in the game. Takes no corporationID —
// this is a whole-game listing, not scoped to any one corporation.
func (c *Client) NPCCorporations(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsNpccorps(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get npc corporations: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMembers returns corporationID's member character IDs. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise.
func (c *Client) CorporationMembers(ctx context.Context, corporationID int64) ([]int64, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembers(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation members: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMembersLimit returns corporationID's member limit.
func (c *Client) CorporationMembersLimit(ctx context.Context, corporationID int64) (int64, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembersLimit(ctx, corporationID).Execute()
	if err != nil {
		return 0, fmt.Errorf("get corporation members limit: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMemberTitles returns each member's assigned title IDs.
func (c *Client) CorporationMemberTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembersTitlesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembersTitles(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation member titles: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMemberTracking returns member login/location/ship tracking data.
func (c *Client) CorporationMemberTracking(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembertracking(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation member tracking: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationRoles returns each member's corp roles.
func (c *Client) CorporationRoles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdRolesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdRoles(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation roles: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationRolesHistory returns corporationID's role-change history. page=0 omits the page
// query parameter entirely, letting ESI default to page 1.
func (c *Client) CorporationRolesHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdRolesHistoryGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdRolesHistory(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation roles history: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationTitles returns corporationID's defined titles.
func (c *Client) CorporationTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdTitlesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdTitles(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation titles: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationDivisions returns corporationID's hangar and wallet division names.
func (c *Client) CorporationDivisions(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdDivisionsGet, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdDivisions(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation divisions: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationShareholders returns corporationID's shareholders. page=0 omits the page query
// parameter entirely, letting ESI default to page 1.
func (c *Client) CorporationShareholders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdShareholdersGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdShareholders(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation shareholders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationStandings returns corporationID's standings toward NPC entities. Returns the same
// generated model as Character's own standings (CharactersCharacterIdStandingsGetInner) -- verified
// from source, not assumed. page=0 omits the page query parameter entirely.
func (c *Client) CorporationStandings(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdStandingsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdStandings(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation standings: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
```

- [ ] **Step 2: Write `esi/corporation_test.go`**

Follow `esi/contracts_test.go`'s pattern exactly: an `httptest.Server` per test, built via `ExportedNewPublicClientWithOptions` (all 14 of these endpoints require no auth token to reach — corp-role authorization is enforced by ESI itself, not by which client constructor is used), asserting on the parsed result and at least one HTTP-error path routed through `describeError`. Cover:

- `TestCorporationDetail_ParsesResponse` — serve a JSON body with all `CorporationsDetail` fields populated (including nested `palette: {main_color, secondary_color, tertiary_color}` and `tax_rates: {isk, loyalty_point}`), assert every field maps through, including `Palette.MainColor` and `TaxRates.Isk`/`TaxRates.LoyaltyPoint`.
- `TestCorporationDetail_HTTPError` — serve a 404, assert the returned error is non-nil and wraps a description (not just a bare status).
- `TestCorporationAllianceHistory_ParsesResponse` — serve a 2-element array, assert length and field mapping (`AllianceId`, `RecordId`, `StartDate`, `IsDeleted`).
- `TestCorporationIcons_ParsesResponse` — serve `{px64x64, px128x128, px256x256}`, assert all three map through.
- `TestNPCCorporations_ParsesResponse` — serve a bare `[]int64` JSON array, assert it round-trips unchanged.
- `TestCorporationMembers_ParsesResponse` — serve a bare `[]int64` JSON array, assert round-trip.
- `TestCorporationMembersLimit_ParsesResponse` — serve a bare JSON integer (e.g. `500`), assert the returned `int64` equals it.
- `TestCorporationMemberTitles_ParsesResponse` — serve `[{character_id, titles: [...]}]`, assert mapping.
- `TestCorporationMemberTracking_ParsesResponse` — serve one entry with all optional fields populated (`base_id`, `location_id`, `ship_type_id`, `logon_date`, `logoff_date`, `start_date`), assert every pointer field is non-nil and correct.
- `TestCorporationRoles_ParsesResponse` — serve one entry with all 8 `[]string` role fields populated, assert each maps to its own field (`Roles`, `RolesAtBase`, `RolesAtHq`, `RolesAtOther`, `GrantableRoles`, `GrantableRolesAtBase`, `GrantableRolesAtHq`, `GrantableRolesAtOther`).
- `TestCorporationRolesHistory_PageZeroOmitsPageParam` — call with `page=0`, assert the request the test server received has no `page` query parameter at all.
- `TestCorporationRolesHistory_ParsesResponseAndTotalPages` — serve one entry plus an `X-Pages: 3` response header, assert the entry maps correctly and `totalPages == 3`.
- `TestCorporationTitles_ParsesResponse` — serve `[{title_id, name}]`, assert mapping.
- `TestCorporationDivisions_ParsesResponse` — serve `{hangar: [{division, name}], wallet: [{division, name}]}`, assert both slices map correctly.
- `TestCorporationShareholders_ParsesResponseAndTotalPages` — serve `[{shareholder_id, shareholder_type, share_count}]` plus `X-Pages: 2`, assert mapping and `totalPages == 2`.
- `TestCorporationStandings_ParsesResponseAndTotalPages` — serve `[{from_id, from_type, standing}]` plus `X-Pages: 1`, assert mapping and `totalPages == 1`.

Use `esi.ExportedNewPublicClientWithOptions` exactly as `esi/contracts_test.go` and `esi/alliance_test.go` already do (check those files for the exact `goesi.ClientOptions{BaseURL: server.URL, UserAgent: "test"}` wiring before writing this file, since the constructor signature must match verbatim).

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run Corporation -v`
Expected: all new tests PASS.

- [ ] **Step 4: Build and vet the whole module**

Run: `go build ./... && go vet ./...`
Expected: clean, no errors.

- [ ] **Step 5: Commit**

```bash
git add esi/corporation.go esi/corporation_test.go
git commit -m "feat: add esi/corporation.go facade (Part 1, 14 read methods)"
```

---

### Task 2: `mcp/corporation.go` tools (14 tools), server registration, and docs sync

**Files:**
- Create: `mcp/corporation.go`
- Test: `mcp/corporation_internal_test.go`
- Modify: `mcp/server.go` (register 14 new tools; doc comment tool list grows to include them)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 57 to 71)
- Modify: `CLAUDE.md` (tool count "fifty-seven" → "seventy-one", `mcp/` row's tool list gains the 14 new names)
- Modify: `README.md` (same two edits, mirrored)

**Interfaces:**
- Consumes: all 14 `esi/corporation.go` methods from Task 1 (exact names/signatures above); `esi.NewClient`, `esi.NewPublicClient` (`esi/client.go`); `characterExists(store *config.Store, characterID int32) (bool, error)` and `ErrCharacterNotFound` and `type Standing struct{FromID int64; FromType string; Standing float64}` — all three already defined in `mcp/character.go`, package-visible, reuse directly, do not redeclare; `sso.UserAgent(version string) string` (`sso/useragent.go`).
- Produces: 14 `mcpsdk.ToolHandlerFor[...]`-returning handler functions for `mcp/server.go` to register: `getCorporationHandler(version string)`, `listCorporationAllianceHistoryHandler(version string)`, `getCorporationIconsHandler(version string)`, `listNPCCorporationsHandler(version string)`, `listCorporationMembersHandler(version, clientID string, store *config.Store)`, `getCorporationMembersLimitHandler(version, clientID string, store *config.Store)`, `listCorporationMemberTitlesHandler(version, clientID string, store *config.Store)`, `listCorporationMemberTrackingHandler(version, clientID string, store *config.Store)`, `listCorporationRolesHandler(version, clientID string, store *config.Store)`, `listCorporationRolesHistoryHandler(version, clientID string, store *config.Store)`, `listCorporationTitlesHandler(version, clientID string, store *config.Store)`, `getCorporationDivisionsHandler(version, clientID string, store *config.Store)`, `listCorporationShareholdersHandler(version, clientID string, store *config.Store)`, `listCorporationStandingsHandler(version, clientID string, store *config.Store)`.

- [ ] **Step 1: Write `mcp/corporation.go`**

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_corporation (public) ---

type GetCorporationOutput struct {
	CorporationID       int64      `json:"corporationId"`
	Name                string     `json:"name"`
	Ticker              string     `json:"ticker"`
	Description         string     `json:"description"`
	CeoID               *int64     `json:"ceoId,omitempty"`
	AllianceID          *int64     `json:"allianceId,omitempty"`
	CreatorID           *int64     `json:"creatorId,omitempty"`
	DateFounded         *time.Time `json:"dateFounded,omitempty"`
	FactionID           *int64     `json:"factionId,omitempty"`
	HomeStationID       int64      `json:"homeStationId"`
	MemberCount         int64      `json:"memberCount"`
	Shares              int64      `json:"shares"`
	TaxRateIsk          float64    `json:"taxRateIsk"`
	TaxRateLoyaltyPoint float64    `json:"taxRateLoyaltyPoint"`
	MainColor           string     `json:"mainColor,omitempty"`
	SecondaryColor      *string    `json:"secondaryColor,omitempty"`
	TertiaryColor       *string    `json:"tertiaryColor,omitempty"`
	URL                 *string    `json:"url,omitempty"`
	FriendlyFire        string     `json:"friendlyFire"`
	State               string     `json:"state"`
	Type                string     `json:"type"`
}

var getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
	client := esi.NewPublicClient(userAgent)
	detail, err := client.CorporationDetail(ctx, corporationID)
	if err != nil {
		return GetCorporationOutput{}, err
	}
	out := GetCorporationOutput{
		CorporationID: corporationID, Name: detail.Name, Ticker: detail.Ticker, Description: detail.Description,
		CeoID: detail.CeoId, AllianceID: detail.AllianceId, CreatorID: detail.CreatorId, DateFounded: detail.DateFounded,
		FactionID: detail.EnlistedFactionId, HomeStationID: detail.HomeStationId, MemberCount: detail.MemberCount,
		Shares: detail.Shares, TaxRateIsk: detail.TaxRates.Isk, TaxRateLoyaltyPoint: detail.TaxRates.LoyaltyPoint,
		URL: detail.Url, FriendlyFire: detail.FriendlyFire, State: detail.State, Type: detail.Type,
	}
	if detail.Palette != nil {
		out.MainColor = detail.Palette.MainColor
		out.SecondaryColor = detail.Palette.SecondaryColor
		out.TertiaryColor = detail.Palette.TertiaryColor
	}
	return out, nil
}

type GetCorporationInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}

func getCorporationHandler(version string) mcpsdk.ToolHandlerFor[GetCorporationInput, GetCorporationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationInput) (*mcpsdk.CallToolResult, GetCorporationOutput, error) {
		out, err := getCorporationDetail(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationOutput{}, fmt.Errorf("get corporation: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_corporation_alliance_history (public) ---

type AllianceHistoryEntry struct {
	AllianceID *int64    `json:"allianceId,omitempty"`
	RecordID   int64     `json:"recordId"`
	StartDate  time.Time `json:"startDate"`
	IsDeleted  *bool     `json:"isDeleted,omitempty"`
}

var listCorporationAllianceHistory = func(ctx context.Context, userAgent string, corporationID int64) ([]AllianceHistoryEntry, error) {
	client := esi.NewPublicClient(userAgent)
	hist, err := client.CorporationAllianceHistory(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]AllianceHistoryEntry, len(hist))
	for i, h := range hist {
		out[i] = AllianceHistoryEntry{AllianceID: h.AllianceId, RecordID: h.RecordId, StartDate: h.StartDate, IsDeleted: h.IsDeleted}
	}
	return out, nil
}

type ListCorporationAllianceHistoryInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationAllianceHistoryOutput struct {
	CorporationID int64                  `json:"corporationId"`
	History       []AllianceHistoryEntry `json:"history"`
}

func listCorporationAllianceHistoryHandler(version string) mcpsdk.ToolHandlerFor[ListCorporationAllianceHistoryInput, ListCorporationAllianceHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationAllianceHistoryInput) (*mcpsdk.CallToolResult, ListCorporationAllianceHistoryOutput, error) {
		hist, err := listCorporationAllianceHistory(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationAllianceHistoryOutput{}, fmt.Errorf("list corporation alliance history: %w", err)
		}
		return nil, ListCorporationAllianceHistoryOutput{CorporationID: in.CorporationID, History: hist}, nil
	}
}

// --- get_corporation_icons (public) ---

type GetCorporationIconsOutput struct {
	CorporationID int64   `json:"corporationId"`
	Px64x64       *string `json:"px64x64,omitempty"`
	Px128x128     *string `json:"px128x128,omitempty"`
	Px256x256     *string `json:"px256x256,omitempty"`
}

var getCorporationIcons = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationIconsOutput, error) {
	client := esi.NewPublicClient(userAgent)
	icons, err := client.CorporationIcons(ctx, corporationID)
	if err != nil {
		return GetCorporationIconsOutput{}, err
	}
	return GetCorporationIconsOutput{CorporationID: corporationID, Px64x64: icons.Px64x64, Px128x128: icons.Px128x128, Px256x256: icons.Px256x256}, nil
}

type GetCorporationIconsInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}

func getCorporationIconsHandler(version string) mcpsdk.ToolHandlerFor[GetCorporationIconsInput, GetCorporationIconsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationIconsInput) (*mcpsdk.CallToolResult, GetCorporationIconsOutput, error) {
		out, err := getCorporationIcons(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationIconsOutput{}, fmt.Errorf("get corporation icons: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_npc_corporations (public, no input) ---

var listNPCCorporations = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.NPCCorporations(ctx)
}

type ListNPCCorporationsInput struct{}
type ListNPCCorporationsOutput struct {
	CorporationIDs []int64 `json:"corporationIds"`
}

func listNPCCorporationsHandler(version string) mcpsdk.ToolHandlerFor[ListNPCCorporationsInput, ListNPCCorporationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListNPCCorporationsInput) (*mcpsdk.CallToolResult, ListNPCCorporationsOutput, error) {
		ids, err := listNPCCorporations(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListNPCCorporationsOutput{}, fmt.Errorf("list npc corporations: %w", err)
		}
		return nil, ListNPCCorporationsOutput{CorporationIDs: ids}, nil
	}
}

// --- list_corporation_members (private) ---

var listCorporationMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMembers(ctx, corporationID)
}

type ListCorporationMembersInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationMembersOutput struct {
	CharacterID   int32   `json:"characterId"`
	CorporationID int64   `json:"corporationId"`
	MemberIDs     []int64 `json:"memberIds"`
}

func listCorporationMembersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMembersInput, ListCorporationMembersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMembersInput) (*mcpsdk.CallToolResult, ListCorporationMembersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMembersOutput{}, err
		}
		if !found {
			return nil, ListCorporationMembersOutput{}, fmt.Errorf("list corporation members for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		ids, err := listCorporationMembers(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationMembersOutput{}, fmt.Errorf("list corporation members: %w", err)
		}
		return nil, ListCorporationMembersOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, MemberIDs: ids}, nil
	}
}

// --- get_corporation_members_limit (private) ---

var getCorporationMembersLimit = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMembersLimit(ctx, corporationID)
}

type GetCorporationMembersLimitInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type GetCorporationMembersLimitOutput struct {
	CharacterID   int32 `json:"characterId"`
	CorporationID int64 `json:"corporationId"`
	Limit         int64 `json:"limit"`
}

func getCorporationMembersLimitHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationMembersLimitInput, GetCorporationMembersLimitOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationMembersLimitInput) (*mcpsdk.CallToolResult, GetCorporationMembersLimitOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationMembersLimitOutput{}, err
		}
		if !found {
			return nil, GetCorporationMembersLimitOutput{}, fmt.Errorf("get corporation members limit for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		limit, err := getCorporationMembersLimit(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationMembersLimitOutput{}, fmt.Errorf("get corporation members limit: %w", err)
		}
		return nil, GetCorporationMembersLimitOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Limit: limit}, nil
	}
}

// --- list_corporation_member_titles (private) ---

type MemberTitles struct {
	CharacterID int64   `json:"characterId"`
	TitleIDs    []int64 `json:"titleIds"`
}

var listCorporationMemberTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTitles, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	titles, err := client.CorporationMemberTitles(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberTitles, len(titles))
	for i, t := range titles {
		out[i] = MemberTitles{CharacterID: t.CharacterId, TitleIDs: t.Titles}
	}
	return out, nil
}

type ListCorporationMemberTitlesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationMemberTitlesOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	MemberTitles  []MemberTitles `json:"memberTitles"`
}

func listCorporationMemberTitlesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMemberTitlesInput, ListCorporationMemberTitlesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMemberTitlesInput) (*mcpsdk.CallToolResult, ListCorporationMemberTitlesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMemberTitlesOutput{}, err
		}
		if !found {
			return nil, ListCorporationMemberTitlesOutput{}, fmt.Errorf("list corporation member titles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		titles, err := listCorporationMemberTitles(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationMemberTitlesOutput{}, fmt.Errorf("list corporation member titles: %w", err)
		}
		return nil, ListCorporationMemberTitlesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, MemberTitles: titles}, nil
	}
}

// --- list_corporation_member_tracking (private) ---

type MemberTracking struct {
	CharacterID int64      `json:"characterId"`
	BaseID      *int64     `json:"baseId,omitempty"`
	LocationID  *int64     `json:"locationId,omitempty"`
	ShipTypeID  *int64     `json:"shipTypeId,omitempty"`
	LogonDate   *time.Time `json:"logonDate,omitempty"`
	LogoffDate  *time.Time `json:"logoffDate,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
}

var listCorporationMemberTracking = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTracking, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	tracking, err := client.CorporationMemberTracking(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberTracking, len(tracking))
	for i, t := range tracking {
		out[i] = MemberTracking{CharacterID: t.CharacterId, BaseID: t.BaseId, LocationID: t.LocationId, ShipTypeID: t.ShipTypeId, LogonDate: t.LogonDate, LogoffDate: t.LogoffDate, StartDate: t.StartDate}
	}
	return out, nil
}

type ListCorporationMemberTrackingInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationMemberTrackingOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	Members       []MemberTracking `json:"members"`
}

func listCorporationMemberTrackingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMemberTrackingInput, ListCorporationMemberTrackingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMemberTrackingInput) (*mcpsdk.CallToolResult, ListCorporationMemberTrackingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMemberTrackingOutput{}, err
		}
		if !found {
			return nil, ListCorporationMemberTrackingOutput{}, fmt.Errorf("list corporation member tracking for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		members, err := listCorporationMemberTracking(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationMemberTrackingOutput{}, fmt.Errorf("list corporation member tracking: %w", err)
		}
		return nil, ListCorporationMemberTrackingOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Members: members}, nil
	}
}

// --- list_corporation_roles (private) ---

type MemberRoles struct {
	CharacterID           int64    `json:"characterId"`
	Roles                 []string `json:"roles,omitempty"`
	RolesAtBase           []string `json:"rolesAtBase,omitempty"`
	RolesAtHq             []string `json:"rolesAtHq,omitempty"`
	RolesAtOther          []string `json:"rolesAtOther,omitempty"`
	GrantableRoles        []string `json:"grantableRoles,omitempty"`
	GrantableRolesAtBase  []string `json:"grantableRolesAtBase,omitempty"`
	GrantableRolesAtHq    []string `json:"grantableRolesAtHq,omitempty"`
	GrantableRolesAtOther []string `json:"grantableRolesAtOther,omitempty"`
}

var listCorporationRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberRoles, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	roles, err := client.CorporationRoles(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberRoles, len(roles))
	for i, r := range roles {
		out[i] = MemberRoles{
			CharacterID: r.CharacterId, Roles: r.Roles, RolesAtBase: r.RolesAtBase, RolesAtHq: r.RolesAtHq, RolesAtOther: r.RolesAtOther,
			GrantableRoles: r.GrantableRoles, GrantableRolesAtBase: r.GrantableRolesAtBase, GrantableRolesAtHq: r.GrantableRolesAtHq, GrantableRolesAtOther: r.GrantableRolesAtOther,
		}
	}
	return out, nil
}

type ListCorporationRolesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationRolesOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Roles         []MemberRoles `json:"roles"`
}

func listCorporationRolesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationRolesInput, ListCorporationRolesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationRolesInput) (*mcpsdk.CallToolResult, ListCorporationRolesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationRolesOutput{}, err
		}
		if !found {
			return nil, ListCorporationRolesOutput{}, fmt.Errorf("list corporation roles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		roles, err := listCorporationRoles(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationRolesOutput{}, fmt.Errorf("list corporation roles: %w", err)
		}
		return nil, ListCorporationRolesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Roles: roles}, nil
	}
}

// --- list_corporation_roles_history (private, paginated) ---

type RoleChange struct {
	CharacterID int64     `json:"characterId"`
	IssuerID    int64     `json:"issuerId"`
	ChangedAt   time.Time `json:"changedAt"`
	RoleType    string    `json:"roleType"`
	OldRoles    []string  `json:"oldRoles"`
	NewRoles    []string  `json:"newRoles"`
}

var listCorporationRolesHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]RoleChange, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	history, totalPages, err := client.CorporationRolesHistory(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]RoleChange, len(history))
	for i, h := range history {
		out[i] = RoleChange{CharacterID: h.CharacterId, IssuerID: h.IssuerId, ChangedAt: h.ChangedAt, RoleType: h.RoleType, OldRoles: h.OldRoles, NewRoles: h.NewRoles}
	}
	return out, totalPages, nil
}

type ListCorporationRolesHistoryInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationRolesHistoryOutput struct {
	CharacterID   int32        `json:"characterId"`
	CorporationID int64        `json:"corporationId"`
	Page          int32        `json:"page"`
	TotalPages    int32        `json:"totalPages"`
	History       []RoleChange `json:"history"`
}

func listCorporationRolesHistoryHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationRolesHistoryInput, ListCorporationRolesHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationRolesHistoryInput) (*mcpsdk.CallToolResult, ListCorporationRolesHistoryOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationRolesHistoryOutput{}, err
		}
		if !found {
			return nil, ListCorporationRolesHistoryOutput{}, fmt.Errorf("list corporation roles history for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		history, totalPages, err := listCorporationRolesHistory(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationRolesHistoryOutput{}, fmt.Errorf("list corporation roles history: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationRolesHistoryOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, History: history}, nil
	}
}

// --- list_corporation_titles (private) ---

type CorporationTitle struct {
	TitleID *int64  `json:"titleId,omitempty"`
	Name    *string `json:"name,omitempty"`
}

var listCorporationTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]CorporationTitle, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	titles, err := client.CorporationTitles(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]CorporationTitle, len(titles))
	for i, t := range titles {
		out[i] = CorporationTitle{TitleID: t.TitleId, Name: t.Name}
	}
	return out, nil
}

type ListCorporationTitlesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationTitlesOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	Titles        []CorporationTitle `json:"titles"`
}

func listCorporationTitlesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationTitlesInput, ListCorporationTitlesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationTitlesInput) (*mcpsdk.CallToolResult, ListCorporationTitlesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationTitlesOutput{}, err
		}
		if !found {
			return nil, ListCorporationTitlesOutput{}, fmt.Errorf("list corporation titles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		titles, err := listCorporationTitles(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationTitlesOutput{}, fmt.Errorf("list corporation titles: %w", err)
		}
		return nil, ListCorporationTitlesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Titles: titles}, nil
	}
}

// --- get_corporation_divisions (private) ---

type Division struct {
	DivisionNumber *int64  `json:"divisionNumber,omitempty"`
	Name           *string `json:"name,omitempty"`
}

var getCorporationDivisions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Division, []Division, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	divisions, err := client.CorporationDivisions(ctx, corporationID)
	if err != nil {
		return nil, nil, err
	}
	hangars := make([]Division, len(divisions.Hangar))
	for i, h := range divisions.Hangar {
		hangars[i] = Division{DivisionNumber: h.Division, Name: h.Name}
	}
	wallets := make([]Division, len(divisions.Wallet))
	for i, w := range divisions.Wallet {
		wallets[i] = Division{DivisionNumber: w.Division, Name: w.Name}
	}
	return hangars, wallets, nil
}

type GetCorporationDivisionsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type GetCorporationDivisionsOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Hangars       []Division `json:"hangars"`
	Wallets       []Division `json:"wallets"`
}

func getCorporationDivisionsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationDivisionsInput, GetCorporationDivisionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationDivisionsInput) (*mcpsdk.CallToolResult, GetCorporationDivisionsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationDivisionsOutput{}, err
		}
		if !found {
			return nil, GetCorporationDivisionsOutput{}, fmt.Errorf("get corporation divisions for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		hangars, wallets, err := getCorporationDivisions(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationDivisionsOutput{}, fmt.Errorf("get corporation divisions: %w", err)
		}
		return nil, GetCorporationDivisionsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Hangars: hangars, Wallets: wallets}, nil
	}
}

// --- list_corporation_shareholders (private, paginated) ---

type Shareholder struct {
	ShareholderID   int64  `json:"shareholderId"`
	ShareholderType string `json:"shareholderType"`
	ShareCount      int64  `json:"shareCount"`
}

var listCorporationShareholders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Shareholder, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	shareholders, totalPages, err := client.CorporationShareholders(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Shareholder, len(shareholders))
	for i, s := range shareholders {
		out[i] = Shareholder{ShareholderID: s.ShareholderId, ShareholderType: s.ShareholderType, ShareCount: s.ShareCount}
	}
	return out, totalPages, nil
}

type ListCorporationShareholdersInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationShareholdersOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Shareholders  []Shareholder `json:"shareholders"`
}

func listCorporationShareholdersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationShareholdersInput, ListCorporationShareholdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationShareholdersInput) (*mcpsdk.CallToolResult, ListCorporationShareholdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationShareholdersOutput{}, err
		}
		if !found {
			return nil, ListCorporationShareholdersOutput{}, fmt.Errorf("list corporation shareholders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		shareholders, totalPages, err := listCorporationShareholders(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationShareholdersOutput{}, fmt.Errorf("list corporation shareholders: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationShareholdersOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Shareholders: shareholders}, nil
	}
}

// --- list_corporation_standings (private, paginated) ---
// Reuses the existing Standing type from mcp/character.go -- same shape (FromID, FromType,
// Standing), same underlying ESI model (CharactersCharacterIdStandingsGetInner). Not redeclared.

var listCorporationStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Standing, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	standings, totalPages, err := client.CorporationStandings(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Standing, len(standings))
	for i, s := range standings {
		out[i] = Standing{FromID: s.FromId, FromType: s.FromType, Standing: s.Standing}
	}
	return out, totalPages, nil
}

type ListCorporationStandingsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationStandingsOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Page          int32      `json:"page"`
	TotalPages    int32      `json:"totalPages"`
	Standings     []Standing `json:"standings"`
}

func listCorporationStandingsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationStandingsInput, ListCorporationStandingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationStandingsInput) (*mcpsdk.CallToolResult, ListCorporationStandingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationStandingsOutput{}, err
		}
		if !found {
			return nil, ListCorporationStandingsOutput{}, fmt.Errorf("list corporation standings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		standings, totalPages, err := listCorporationStandings(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationStandingsOutput{}, fmt.Errorf("list corporation standings: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationStandingsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Standings: standings}, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound`/`Standing` with the identical shape they have in the real `mcp/character.go` — the reuse compiles as designed.

- [ ] **Step 2: Write `mcp/corporation_internal_test.go`**

Follow `mcp/contracts_internal_test.go`'s pattern exactly (same package `mcp`, testing the package-level `var` funcs and handlers directly, not through a live MCP session). For **each of the 14 tools**, write:
- One happy-path test that stubs the relevant `esi.Client` method behavior indirectly by overriding the package-level `var` (e.g. `listCorporationMembers = func(...) {...}` inside the test, restored via `t.Cleanup`) and asserts the handler's output mapping is correct.
- For the **10 private** tools only: one `TestX_UnknownCharacter_ReturnsError` test using a real (temp-file) `config.Store` with no characters added, asserting the returned error wraps `ErrCharacterNotFound` (via `errors.Is`).

Concretely, the 14 test functions are:
`TestGetCorporationHandler_MapsFields`, `TestListCorporationAllianceHistoryHandler_MapsFields`, `TestGetCorporationIconsHandler_MapsFields`, `TestListNPCCorporationsHandler_MapsFields`, `TestListCorporationMembersHandler_MapsFields` + `TestListCorporationMembersHandler_UnknownCharacter`, `TestGetCorporationMembersLimitHandler_MapsFields` + `TestGetCorporationMembersLimitHandler_UnknownCharacter`, `TestListCorporationMemberTitlesHandler_MapsFields` + `TestListCorporationMemberTitlesHandler_UnknownCharacter`, `TestListCorporationMemberTrackingHandler_MapsFields` + `TestListCorporationMemberTrackingHandler_UnknownCharacter`, `TestListCorporationRolesHandler_MapsFields` + `TestListCorporationRolesHandler_UnknownCharacter`, `TestListCorporationRolesHistoryHandler_MapsFieldsAndDefaultsPageTo1` + `TestListCorporationRolesHistoryHandler_UnknownCharacter`, `TestListCorporationTitlesHandler_MapsFields` + `TestListCorporationTitlesHandler_UnknownCharacter`, `TestGetCorporationDivisionsHandler_MapsFields` + `TestGetCorporationDivisionsHandler_UnknownCharacter`, `TestListCorporationShareholdersHandler_MapsFieldsAndDefaultsPageTo1` + `TestListCorporationShareholdersHandler_UnknownCharacter`, `TestListCorporationStandingsHandler_MapsFieldsAndDefaultsPageTo1` + `TestListCorporationStandingsHandler_UnknownCharacter` — reusing `mcp/character.go`'s existing `Standing` type in the assertion, not a new one.

Use `keyring.MockInit()` at the start of any test that touches `config.Store`, matching every other `*_internal_test.go` file in this package.

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run Corporation -v`
Expected: all new tests PASS.

- [ ] **Step 4: Register the 14 tools in `mcp/server.go`**

Add these 14 `mcpsdk.AddTool` blocks immediately after the existing `list_public_contract_items` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation",
		Description: "Get a corporation's public info (public data, no character required).",
	}, getCorporationHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_alliance_history",
		Description: "List a corporation's alliance membership history (public data, no character required).",
	}, listCorporationAllianceHistoryHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_icons",
		Description: "Get a corporation's icon URLs (public data, no character required).",
	}, getCorporationIconsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_npc_corporations",
		Description: "List every NPC corporation ID in the game (public data, no character required).",
	}, listNPCCorporationsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_members",
		Description: "List a corporation's member character IDs, using a logged-in character's roles to authorize.",
	}, listCorporationMembersHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_members_limit",
		Description: "Get a corporation's member limit, using a logged-in character's roles to authorize.",
	}, getCorporationMembersLimitHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_member_titles",
		Description: "List a corporation's members' assigned title IDs, using a logged-in character's roles to authorize.",
	}, listCorporationMemberTitlesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_member_tracking",
		Description: "List a corporation's member login/location/ship tracking data, using a logged-in character's roles to authorize.",
	}, listCorporationMemberTrackingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_roles",
		Description: "List a corporation's members' corp roles, using a logged-in character's roles to authorize.",
	}, listCorporationRolesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_roles_history",
		Description: "List a corporation's role-change history, using a logged-in character's roles to authorize.",
	}, listCorporationRolesHistoryHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_titles",
		Description: "List a corporation's defined titles, using a logged-in character's roles to authorize.",
	}, listCorporationTitlesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_divisions",
		Description: "Get a corporation's hangar and wallet division names, using a logged-in character's roles to authorize.",
	}, getCorporationDivisionsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_shareholders",
		Description: "List a corporation's shareholders, using a logged-in character's roles to authorize.",
	}, listCorporationShareholdersHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_standings",
		Description: "List a corporation's standings toward NPC entities, using a logged-in character's roles to authorize.",
	}, listCorporationStandingsHandler(version, clientID, store))

	return server
}
```

Update the package doc comment above `NewServer` to append the 14 new tool names to its running list (ending in `..., list_public_contract_items, get_corporation, list_corporation_alliance_history, get_corporation_icons, list_npc_corporations, list_corporation_members, get_corporation_members_limit, list_corporation_member_titles, list_corporation_member_tracking, list_corporation_roles, list_corporation_roles_history, list_corporation_titles, get_corporation_divisions, list_corporation_shareholders, and list_corporation_standings registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

In `TestServer_AllToolsRegistered`, append these 14 names to the `[]string{...}` literal, right after `"list_public_contracts", "list_public_contract_bids", "list_public_contract_items",`:

```go
			"get_corporation", "list_corporation_alliance_history", "get_corporation_icons", "list_npc_corporations",
			"list_corporation_members", "get_corporation_members_limit", "list_corporation_member_titles",
			"list_corporation_member_tracking", "list_corporation_roles", "list_corporation_roles_history",
			"list_corporation_titles", "get_corporation_divisions", "list_corporation_shareholders",
			"list_corporation_standings",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 71 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "fifty-seven" to "seventy-one".
- Append the 14 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list, before the closing `` ` ``.
- Do **not** touch `docs/reference/esi-coverage-roadmap.md`'s Corporation row — it stays `Planned` until Part 2 also lands, per the design spec's explicit split-batch note.

- [ ] **Step 8: Commit**

```bash
git add mcp/corporation.go mcp/corporation_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md
git commit -m "feat: add corporation MCP tools, Part 1 (14 tools: basics + membership/roles)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full branch diff for this batch (`git diff 8315a56..HEAD` — or the actual base commit recorded before Task 1 began), same as every prior batch in this sweep.
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
