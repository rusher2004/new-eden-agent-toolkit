# ESI Industry (Eighteenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 8 endpoints of the "Industry" ESI tag as a new `esi/industry.go` facade and a new `mcp/industry.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `IndustryAPI`, reusing the existing `describeError` and `totalPagesFromHeader` helpers — this is the first batch since Corporation to use the classic page/`X-Pages` pagination style rather than cursors. One new `mcp/` tool file with 6 private tools (2 character-scoped, 4 corp-scoped) and 2 public tools. Introduces a shared `IndustryJob` type reused across the character- and corp-scoped job-listing tools, since the two underlying ESI models are 95% identical (differing only in `StationID` vs `LocationID`).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `eb5985d`) — no new branch, no new PR.
- Current tool count baseline: **120**. After this plan: **128**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`). Every paginated method uses the existing `totalPagesFromHeader` (`esi/contracts.go`) — do not redeclare either.
- `CharacterIndustryJobs`/`CorporationIndustryJobs` take an `includeCompleted bool` and call `.IncludeCompleted(includeCompleted)` **unconditionally** — this is a genuine boolean filter always sent, not an ID/cursor with "omit when zero" semantics.
- 4 of 8 methods are paginated (`CharacterMining`, `CorporationMiningExtractions`, `CorporationMiningObservers`, `CorporationMiningObserverDetail`, `CorporationIndustryJobs` — that's actually 5; `CharacterIndustryJobs` is NOT paginated despite also taking `includeCompleted`). Paginated methods: `page=0` omits `.Page()`, return `(items, totalPages int32, error)`.
- `IndustryJob` is shared between `list_character_industry_jobs` and `list_corporation_industry_jobs` — it carries both `StationID` and `LocationID` fields; only the applicable one is populated per source, the other left zero. Do not create two near-duplicate types for a one-field difference.
- ESI itself is inconsistent in URL naming (`/corporation/{id}/mining/...` singular vs. `/corporations/{id}/industry/jobs` plural) — this is verified from the generated method names themselves and does not affect neat's own facade method names or behavior; no action needed beyond awareness.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean). Implementers should still run these themselves as part of each task.

---

### Task 1: `esi/industry.go` facade (8 methods)

**Files:**
- Create: `esi/industry.go`
- Test: `esi/industry_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `totalPagesFromHeader(resp *http.Response) int32` (`esi/contracts.go`), `Client.api`/`Client.characterID` (`esi/client.go`).
- Produces: 8 methods on `*Client` for Task 2's `mcp/industry.go` to call.

- [ ] **Step 1: Write `esi/industry.go`**

```go
// Package esi's industry.go covers all 8 of "Industry"'s endpoints: character/corp industry
// jobs, character/corp mining ledgers/extractions/observers, and public facility/system data.
// Back to the classic page/X-Pages pagination style (esi/contracts.go's totalPagesFromHeader) --
// not cursor-based, unlike the two most recent batches (Corporation Projects, Freelance Jobs).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterIndustryJobs returns the client's own character's industry jobs. Not paginated.
// includeCompleted is always sent explicitly -- a genuine boolean filter, unlike page which is
// conditionally omitted.
func (c *Client) CharacterIndustryJobs(ctx context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
	result, resp, err := c.api.IndustryAPI.GetCharactersCharacterIdIndustryJobs(ctx, int64(c.characterID)).IncludeCompleted(includeCompleted).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character industry jobs: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMining returns the client's own character's mining ledger. page=0 omits the page
// query parameter.
func (c *Client) CharacterMining(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCharactersCharacterIdMining(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character mining: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMiningExtractions returns corporationID's moon mining extractions. page=0 omits
// the page query parameter.
func (c *Client) CorporationMiningExtractions(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationCorporationIdMiningExtractions(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation mining extractions: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMiningObservers returns corporationID's mining observers (structures/entities that
// recorded mining activity). page=0 omits the page query parameter.
func (c *Client) CorporationMiningObservers(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationCorporationIdMiningObservers(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation mining observers: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMiningObserverDetail returns the mining ledger recorded by one observer. page=0
// omits the page query parameter.
func (c *Client) CorporationMiningObserverDetail(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationCorporationIdMiningObserversObserverId(ctx, corporationID, observerID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation mining observer detail: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationIndustryJobs returns corporationID's industry jobs. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise. page=0 omits the page
// query parameter; includeCompleted is always sent explicitly.
func (c *Client) CorporationIndustryJobs(ctx context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
	req := c.api.IndustryAPI.GetCorporationsCorporationIdIndustryJobs(ctx, corporationID).IncludeCompleted(includeCompleted)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation industry jobs: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// IndustryFacilities returns every industry facility in the game. Public, not paginated.
func (c *Client) IndustryFacilities(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error) {
	result, resp, err := c.api.IndustryAPI.GetIndustryFacilities(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get industry facilities: %w", describeError(err, resp))
	}
	return result, nil
}

// IndustrySystems returns industry cost indices for every solar system. Public, not paginated.
func (c *Client) IndustrySystems(ctx context.Context) ([]esiapi.IndustrySystemsGetInner, error) {
	result, resp, err := c.api.IndustryAPI.GetIndustrySystems(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get industry systems: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Write `esi/industry_test.go`**

Read `esi/contracts_test.go` first to copy its exact pagination-omit/set test pattern. Cover:

- `TestCharacterIndustryJobs_ParsesResponseAndSendsIncludeCompleted` — serve a 1-element array with every field populated, assert every field maps, AND assert the query string contains `include_completed=true`/`=false` for both a `true` and `false` call (2 subtests or 2 test functions).
- `TestCharacterIndustryJobs_HTTPError` — serve a 404, assert error non-nil.
- `TestCharacterMining_ParsesResponseAndPageOmittedWhenZero` — serve a 1-element array, assert every field maps (note `Date` is a plain `string`, not `time.Time`); assert no `page` param when `page=0`.
- `TestCharacterMining_PageSentWhenSet` — call with `page=3`, assert query string contains `page=3`, plus `X-Pages` response header round-trips to `totalPages`.
- `TestCorporationMiningExtractions_ParsesResponseAndTotalPages` — serve a 1-element array with all 3 `time.Time` fields populated plus `X-Pages: 2`, assert every field maps and `totalPages == 2`.
- `TestCorporationMiningObservers_ParsesResponseAndTotalPages` — serve a 1-element array, assert mapping and `totalPages`.
- `TestCorporationMiningObserverDetail_ParsesResponseAndTotalPages` — serve a 1-element array, assert every field maps (note `LastUpdated` is a plain `string` here too) and `totalPages`.
- `TestCorporationIndustryJobs_ParsesResponseAndSendsIncludeCompleted` — serve a 1-element array with `location_id` populated (not `station_id`, confirming the corp variant's distinct field), assert every field maps and `include_completed` is sent.
- `TestIndustryFacilities_ParsesResponse` — serve a 1-element array, assert every field maps including `tax`.
- `TestIndustrySystems_ParsesResponse` — serve a 1-element array with a non-empty `cost_indices`, assert every field maps.

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run "CharacterIndustryJobs|CharacterMining|CorporationMiningExtractions|CorporationMiningObservers|CorporationMiningObserverDetail|CorporationIndustryJobs|IndustryFacilities|IndustrySystems" -v`
Expected: all tests PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/industry.go esi/industry_test.go
git commit -m "feat: add esi/industry.go facade (page pagination, shared job model)"
```

---

### Task 2: `mcp/industry.go` tools (8 tools), server registration, and docs sync

**Files:**
- Create: `mcp/industry.go`
- Test: `mcp/industry_internal_test.go`
- Modify: `mcp/server.go` (register 8 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 120 to 128)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Industry row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "one hundred twenty" → "one hundred twenty-eight")
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: all 8 `esi/industry.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent`.
- Produces: 8 `mcpsdk.ToolHandlerFor[...]`-returning functions: `listCharacterIndustryJobsHandler`, `listCharacterMiningHandler`, `listCorporationMiningExtractionsHandler`, `listCorporationMiningObserversHandler`, `getCorporationMiningObserverDetailHandler`, `listCorporationIndustryJobsHandler` (all `(version, clientID string, store *config.Store)`), `listIndustryFacilitiesHandler`, `listIndustrySystemsHandler` (both `(version string)` only).

- [ ] **Step 1: Write `mcp/industry.go`**

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

// IndustryJob is shared by list_character_industry_jobs and list_corporation_industry_jobs --
// the two underlying ESI models are 95% identical, differing only in StationID (character-scoped
// jobs) vs LocationID (corp-scoped jobs). Only the applicable field is populated; the other is
// left at its zero value.
type IndustryJob struct {
	ActivityID           int64      `json:"activityId"`
	BlueprintID          int64      `json:"blueprintId"`
	BlueprintLocationID  int64      `json:"blueprintLocationId"`
	BlueprintTypeID      int64      `json:"blueprintTypeId"`
	Duration             int64      `json:"duration"`
	EndDate              time.Time  `json:"endDate"`
	FacilityID           int64      `json:"facilityId"`
	InstallerID          int64      `json:"installerId"`
	JobID                int64      `json:"jobId"`
	OutputLocationID     int64      `json:"outputLocationId"`
	Runs                 int64      `json:"runs"`
	StartDate            time.Time  `json:"startDate"`
	Status               string     `json:"status"`
	StationID            int64      `json:"stationId,omitempty"`
	LocationID           int64      `json:"locationId,omitempty"`
	CompletedCharacterID *int64     `json:"completedCharacterId,omitempty"`
	CompletedDate        *time.Time `json:"completedDate,omitempty"`
	Cost                 *float64   `json:"cost,omitempty"`
	LicensedRuns         *int64     `json:"licensedRuns,omitempty"`
	PauseDate            *time.Time `json:"pauseDate,omitempty"`
	Probability          *float64   `json:"probability,omitempty"`
	ProductTypeID        *int64     `json:"productTypeId,omitempty"`
	SuccessfulRuns       *int64     `json:"successfulRuns,omitempty"`
}

func toIndustryJobFromCharacter(j esiapi.CharactersCharacterIdIndustryJobsGetInner) IndustryJob {
	return IndustryJob{
		ActivityID: j.ActivityId, BlueprintID: j.BlueprintId, BlueprintLocationID: j.BlueprintLocationId,
		BlueprintTypeID: j.BlueprintTypeId, Duration: j.Duration, EndDate: j.EndDate, FacilityID: j.FacilityId,
		InstallerID: j.InstallerId, JobID: j.JobId, OutputLocationID: j.OutputLocationId, Runs: j.Runs,
		StartDate: j.StartDate, Status: j.Status, StationID: j.StationId,
		CompletedCharacterID: j.CompletedCharacterId, CompletedDate: j.CompletedDate, Cost: j.Cost,
		LicensedRuns: j.LicensedRuns, PauseDate: j.PauseDate, Probability: j.Probability,
		ProductTypeID: j.ProductTypeId, SuccessfulRuns: j.SuccessfulRuns,
	}
}

func toIndustryJobFromCorporation(j esiapi.CorporationsCorporationIdIndustryJobsGetInner) IndustryJob {
	return IndustryJob{
		ActivityID: j.ActivityId, BlueprintID: j.BlueprintId, BlueprintLocationID: j.BlueprintLocationId,
		BlueprintTypeID: j.BlueprintTypeId, Duration: j.Duration, EndDate: j.EndDate, FacilityID: j.FacilityId,
		InstallerID: j.InstallerId, JobID: j.JobId, OutputLocationID: j.OutputLocationId, Runs: j.Runs,
		StartDate: j.StartDate, Status: j.Status, LocationID: j.LocationId,
		CompletedCharacterID: j.CompletedCharacterId, CompletedDate: j.CompletedDate, Cost: j.Cost,
		LicensedRuns: j.LicensedRuns, PauseDate: j.PauseDate, Probability: j.Probability,
		ProductTypeID: j.ProductTypeId, SuccessfulRuns: j.SuccessfulRuns,
	}
}

// --- list_character_industry_jobs (private) ---

var listCharacterIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterIndustryJobs(ctx, includeCompleted)
}

type ListCharacterIndustryJobsInput struct {
	CharacterID      int32 `json:"characterId" jsonschema:"the character ID to check"`
	IncludeCompleted bool  `json:"includeCompleted,omitempty" jsonschema:"whether to include completed jobs"`
}
type ListCharacterIndustryJobsOutput struct {
	CharacterID      int32         `json:"characterId"`
	IncludeCompleted bool          `json:"includeCompleted"`
	Jobs             []IndustryJob `json:"jobs"`
}

func listCharacterIndustryJobsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterIndustryJobsInput, ListCharacterIndustryJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterIndustryJobsInput) (*mcpsdk.CallToolResult, ListCharacterIndustryJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterIndustryJobsOutput{}, err
		}
		if !found {
			return nil, ListCharacterIndustryJobsOutput{}, fmt.Errorf("list character industry jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		jobs, err := listCharacterIndustryJobs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.IncludeCompleted)
		if err != nil {
			return nil, ListCharacterIndustryJobsOutput{}, fmt.Errorf("list character industry jobs: %w", err)
		}
		out := make([]IndustryJob, len(jobs))
		for i, j := range jobs {
			out[i] = toIndustryJobFromCharacter(j)
		}
		return nil, ListCharacterIndustryJobsOutput{CharacterID: in.CharacterID, IncludeCompleted: in.IncludeCompleted, Jobs: out}, nil
	}
}

// --- list_character_mining (private, paginated) ---

type MiningLedgerEntry struct {
	Date          string `json:"date"`
	Quantity      int64  `json:"quantity"`
	SolarSystemID int64  `json:"solarSystemId"`
	TypeID        int64  `json:"typeId"`
}

var listCharacterMining = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterMining(ctx, page)
}

type ListCharacterMiningInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterMiningOutput struct {
	CharacterID int32               `json:"characterId"`
	Page        int32               `json:"page"`
	TotalPages  int32               `json:"totalPages"`
	Ledger      []MiningLedgerEntry `json:"ledger"`
}

func listCharacterMiningHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterMiningInput, ListCharacterMiningOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMiningInput) (*mcpsdk.CallToolResult, ListCharacterMiningOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMiningOutput{}, err
		}
		if !found {
			return nil, ListCharacterMiningOutput{}, fmt.Errorf("list character mining for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		ledger, totalPages, err := listCharacterMining(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListCharacterMiningOutput{}, fmt.Errorf("list character mining: %w", err)
		}
		out := make([]MiningLedgerEntry, len(ledger))
		for i, e := range ledger {
			out[i] = MiningLedgerEntry{Date: e.Date, Quantity: e.Quantity, SolarSystemID: e.SolarSystemId, TypeID: e.TypeId}
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCharacterMiningOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Ledger: out}, nil
	}
}

// --- list_corporation_mining_extractions (private, paginated) ---

type MiningExtraction struct {
	MoonID              int64     `json:"moonId"`
	StructureID         int64     `json:"structureId"`
	ChunkArrivalTime    time.Time `json:"chunkArrivalTime"`
	ExtractionStartTime time.Time `json:"extractionStartTime"`
	NaturalDecayTime    time.Time `json:"naturalDecayTime"`
}

var listCorporationMiningExtractions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMiningExtractions(ctx, corporationID, page)
}

type ListCorporationMiningExtractionsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMiningExtractionsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	Page          int32              `json:"page"`
	TotalPages    int32              `json:"totalPages"`
	Extractions   []MiningExtraction `json:"extractions"`
}

func listCorporationMiningExtractionsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMiningExtractionsInput, ListCorporationMiningExtractionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMiningExtractionsInput) (*mcpsdk.CallToolResult, ListCorporationMiningExtractionsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMiningExtractionsOutput{}, err
		}
		if !found {
			return nil, ListCorporationMiningExtractionsOutput{}, fmt.Errorf("list corporation mining extractions for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		extractions, totalPages, err := listCorporationMiningExtractions(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMiningExtractionsOutput{}, fmt.Errorf("list corporation mining extractions: %w", err)
		}
		out := make([]MiningExtraction, len(extractions))
		for i, e := range extractions {
			out[i] = MiningExtraction{
				MoonID: e.MoonId, StructureID: e.StructureId, ChunkArrivalTime: e.ChunkArrivalTime,
				ExtractionStartTime: e.ExtractionStartTime, NaturalDecayTime: e.NaturalDecayTime,
			}
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationMiningExtractionsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Extractions: out}, nil
	}
}

// --- list_corporation_mining_observers (private, paginated) ---

type MiningObserver struct {
	ObserverID   int64  `json:"observerId"`
	LastUpdated  string `json:"lastUpdated"`
	ObserverType string `json:"observerType"`
}

var listCorporationMiningObservers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMiningObservers(ctx, corporationID, page)
}

type ListCorporationMiningObserversInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMiningObserversOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	Page          int32            `json:"page"`
	TotalPages    int32            `json:"totalPages"`
	Observers     []MiningObserver `json:"observers"`
}

func listCorporationMiningObserversHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMiningObserversInput, ListCorporationMiningObserversOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMiningObserversInput) (*mcpsdk.CallToolResult, ListCorporationMiningObserversOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMiningObserversOutput{}, err
		}
		if !found {
			return nil, ListCorporationMiningObserversOutput{}, fmt.Errorf("list corporation mining observers for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		observers, totalPages, err := listCorporationMiningObservers(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMiningObserversOutput{}, fmt.Errorf("list corporation mining observers: %w", err)
		}
		out := make([]MiningObserver, len(observers))
		for i, o := range observers {
			out[i] = MiningObserver{ObserverID: o.ObserverId, LastUpdated: o.LastUpdated, ObserverType: o.ObserverType}
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationMiningObserversOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Observers: out}, nil
	}
}

// --- get_corporation_mining_observer_detail (private, paginated) ---

type MiningObserverEntry struct {
	CharacterID           int64  `json:"characterId"`
	LastUpdated           string `json:"lastUpdated"`
	Quantity              int64  `json:"quantity"`
	RecordedCorporationID int64  `json:"recordedCorporationId"`
	TypeID                int64  `json:"typeId"`
}

var getCorporationMiningObserverDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMiningObserverDetail(ctx, corporationID, observerID, page)
}

type GetCorporationMiningObserverDetailInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ObserverID    int64 `json:"observerId" jsonschema:"the observer ID, from list_corporation_mining_observers"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type GetCorporationMiningObserverDetailOutput struct {
	CharacterID   int32                 `json:"characterId"`
	CorporationID int64                 `json:"corporationId"`
	ObserverID    int64                 `json:"observerId"`
	Page          int32                 `json:"page"`
	TotalPages    int32                 `json:"totalPages"`
	Entries       []MiningObserverEntry `json:"entries"`
}

func getCorporationMiningObserverDetailHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationMiningObserverDetailInput, GetCorporationMiningObserverDetailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationMiningObserverDetailInput) (*mcpsdk.CallToolResult, GetCorporationMiningObserverDetailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationMiningObserverDetailOutput{}, err
		}
		if !found {
			return nil, GetCorporationMiningObserverDetailOutput{}, fmt.Errorf("get corporation mining observer detail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		entries, totalPages, err := getCorporationMiningObserverDetail(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ObserverID, in.Page)
		if err != nil {
			return nil, GetCorporationMiningObserverDetailOutput{}, fmt.Errorf("get corporation mining observer detail: %w", err)
		}
		out := make([]MiningObserverEntry, len(entries))
		for i, e := range entries {
			out[i] = MiningObserverEntry{
				CharacterID: e.CharacterId, LastUpdated: e.LastUpdated, Quantity: e.Quantity,
				RecordedCorporationID: e.RecordedCorporationId, TypeID: e.TypeId,
			}
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, GetCorporationMiningObserverDetailOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, ObserverID: in.ObserverID,
			Page: page, TotalPages: totalPages, Entries: out,
		}, nil
	}
}

// --- list_corporation_industry_jobs (private, paginated) ---

var listCorporationIndustryJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationIndustryJobs(ctx, corporationID, includeCompleted, page)
}

type ListCorporationIndustryJobsInput struct {
	CharacterID      int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID    int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	IncludeCompleted bool  `json:"includeCompleted,omitempty" jsonschema:"whether to include completed jobs"`
	Page             int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationIndustryJobsOutput struct {
	CharacterID      int32         `json:"characterId"`
	CorporationID    int64         `json:"corporationId"`
	IncludeCompleted bool          `json:"includeCompleted"`
	Page             int32         `json:"page"`
	TotalPages       int32         `json:"totalPages"`
	Jobs             []IndustryJob `json:"jobs"`
}

func listCorporationIndustryJobsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationIndustryJobsInput, ListCorporationIndustryJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationIndustryJobsInput) (*mcpsdk.CallToolResult, ListCorporationIndustryJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationIndustryJobsOutput{}, err
		}
		if !found {
			return nil, ListCorporationIndustryJobsOutput{}, fmt.Errorf("list corporation industry jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		jobs, totalPages, err := listCorporationIndustryJobs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.IncludeCompleted, in.Page)
		if err != nil {
			return nil, ListCorporationIndustryJobsOutput{}, fmt.Errorf("list corporation industry jobs: %w", err)
		}
		out := make([]IndustryJob, len(jobs))
		for i, j := range jobs {
			out[i] = toIndustryJobFromCorporation(j)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationIndustryJobsOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, IncludeCompleted: in.IncludeCompleted,
			Page: page, TotalPages: totalPages, Jobs: out,
		}, nil
	}
}

// --- list_industry_facilities (public, no input) ---

type IndustryFacility struct {
	FacilityID    int64    `json:"facilityId"`
	OwnerID       int64    `json:"ownerId"`
	RegionID      int64    `json:"regionId"`
	SolarSystemID int64    `json:"solarSystemId"`
	TypeID        int64    `json:"typeId"`
	Tax           *float64 `json:"tax,omitempty"`
}

var listIndustryFacilities = func(ctx context.Context, userAgent string) ([]esiapi.IndustryFacilitiesGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.IndustryFacilities(ctx)
}

type ListIndustryFacilitiesInput struct{}
type ListIndustryFacilitiesOutput struct {
	Facilities []IndustryFacility `json:"facilities"`
}

func listIndustryFacilitiesHandler(version string) mcpsdk.ToolHandlerFor[ListIndustryFacilitiesInput, ListIndustryFacilitiesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIndustryFacilitiesInput) (*mcpsdk.CallToolResult, ListIndustryFacilitiesOutput, error) {
		facilities, err := listIndustryFacilities(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListIndustryFacilitiesOutput{}, fmt.Errorf("list industry facilities: %w", err)
		}
		out := make([]IndustryFacility, len(facilities))
		for i, f := range facilities {
			out[i] = IndustryFacility{
				FacilityID: f.FacilityId, OwnerID: f.OwnerId, RegionID: f.RegionId,
				SolarSystemID: f.SolarSystemId, TypeID: f.TypeId, Tax: f.Tax,
			}
		}
		return nil, ListIndustryFacilitiesOutput{Facilities: out}, nil
	}
}

// --- list_industry_systems (public, no input) ---

type CostIndex struct {
	Activity  string  `json:"activity"`
	CostIndex float64 `json:"costIndex"`
}

type IndustrySystem struct {
	SolarSystemID int64       `json:"solarSystemId"`
	CostIndices   []CostIndex `json:"costIndices"`
}

var listIndustrySystems = func(ctx context.Context, userAgent string) ([]esiapi.IndustrySystemsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.IndustrySystems(ctx)
}

type ListIndustrySystemsInput struct{}
type ListIndustrySystemsOutput struct {
	Systems []IndustrySystem `json:"systems"`
}

func listIndustrySystemsHandler(version string) mcpsdk.ToolHandlerFor[ListIndustrySystemsInput, ListIndustrySystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIndustrySystemsInput) (*mcpsdk.CallToolResult, ListIndustrySystemsOutput, error) {
		systems, err := listIndustrySystems(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListIndustrySystemsOutput{}, fmt.Errorf("list industry systems: %w", err)
		}
		out := make([]IndustrySystem, len(systems))
		for i, s := range systems {
			costIndices := make([]CostIndex, len(s.CostIndices))
			for j, ci := range s.CostIndices {
				costIndices[j] = CostIndex{Activity: ci.Activity, CostIndex: ci.CostIndex}
			}
			out[i] = IndustrySystem{SolarSystemID: s.SolarSystemId, CostIndices: costIndices}
		}
		return nil, ListIndustrySystemsOutput{Systems: out}, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Write `mcp/industry_internal_test.go`**

Follow `mcp/faction_warfare_internal_test.go`'s pattern (package-level `var` override + `t.Cleanup`; `keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). Write one happy-path test per tool (asserting every field the stub returns) and one unknown-character test per private tool:

- `TestListCharacterIndustryJobsHandler_MapsFields` / `_UnknownCharacter`
- `TestListCharacterMiningHandler_MapsFieldsAndDefaultsPageTo1` / `_UnknownCharacter`
- `TestListCorporationMiningExtractionsHandler_MapsFieldsAndDefaultsPageTo1` / `_UnknownCharacter`
- `TestListCorporationMiningObserversHandler_MapsFieldsAndDefaultsPageTo1` / `_UnknownCharacter`
- `TestGetCorporationMiningObserverDetailHandler_MapsFieldsAndDefaultsPageTo1` / `_UnknownCharacter`
- `TestListCorporationIndustryJobsHandler_MapsFieldsAndDefaultsPageTo1` / `_UnknownCharacter`
- `TestListIndustryFacilitiesHandler_MapsFields` (no unknown-character test — public)
- `TestListIndustrySystemsHandler_MapsFields` (no unknown-character test — public, including a non-empty nested `CostIndices`)

That's 14 test functions total (6 private tools × 2 + 2 public tools × 1).

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run "CharacterIndustryJobs|CharacterMining|CorporationMiningExtractions|CorporationMiningObservers|CorporationMiningObserverDetail|CorporationIndustryJobs|IndustryFacilities|IndustrySystems" -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 8 tools in `mcp/server.go`**

Add these 8 `mcpsdk.AddTool` blocks immediately after the existing `list_incursions` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_industry_jobs",
		Description: "List a logged-in EVE Online character's industry jobs.",
	}, listCharacterIndustryJobsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_mining",
		Description: "List a logged-in EVE Online character's mining ledger.",
	}, listCharacterMiningHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_mining_extractions",
		Description: "List a corporation's moon mining extractions, using a logged-in character's roles to authorize.",
	}, listCorporationMiningExtractionsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_mining_observers",
		Description: "List a corporation's mining observers, using a logged-in character's roles to authorize.",
	}, listCorporationMiningObserversHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_mining_observer_detail",
		Description: "Get the mining ledger recorded by one of a corporation's mining observers, using a logged-in character's roles to authorize.",
	}, getCorporationMiningObserverDetailHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_industry_jobs",
		Description: "List a corporation's industry jobs, using a logged-in character's roles to authorize.",
	}, listCorporationIndustryJobsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_industry_facilities",
		Description: "List every industry facility in the game (public data, no character required).",
	}, listIndustryFacilitiesHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_industry_systems",
		Description: "List industry cost indices for every solar system (public data, no character required).",
	}, listIndustrySystemsHandler(version))

	return server
}
```

Update the package doc comment above `NewServer` to append the 8 new tool names to its running
list (ending in `..., list_incursions, list_character_industry_jobs, list_character_mining,
list_corporation_mining_extractions, list_corporation_mining_observers,
get_corporation_mining_observer_detail, list_corporation_industry_jobs, list_industry_facilities,
and list_industry_systems registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 8 names to the `[]string{...}` literal, right after `"list_incursions",`:

```go
			"list_character_industry_jobs", "list_character_mining", "list_corporation_mining_extractions",
			"list_corporation_mining_observers", "get_corporation_mining_observer_detail",
			"list_corporation_industry_jobs", "list_industry_facilities", "list_industry_systems",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 128 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "one hundred twenty" to "one hundred twenty-eight".
- Append the 8 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Industry row from `Planned` to `This plan`.

- [ ] **Step 8: Commit**

```bash
git add mcp/industry.go mcp/industry_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add industry MCP tools (8 tools: jobs, mining, facilities, systems)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff eb5985d..HEAD` — the design spec commit forward).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
