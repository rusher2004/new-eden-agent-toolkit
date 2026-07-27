# ESI Freelance Jobs (Sixteenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 6 endpoints of the "Freelance Jobs" ESI tag as a new `esi/freelance_jobs.go` facade and a new `mcp/freelance_jobs.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `FreelanceJobsAPI`, reusing the existing `describeError` helper. One new `mcp/` tool file with 2 private character-scoped tools, 2 private corp-scoped cursor-paginated tools, and 2 public cursor-paginated tools. Flattens a nested 4-variant polymorphic `Configuration.Parameters` map value into a discriminated `ConfigParameter` type, mirroring the Corporation Projects batch's precedent (a smaller instance of the same pattern: 4 variants nested one level deep, vs. 17 variants at the top level).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `a04d6ae`) — no new branch, no new PR.
- Current tool count baseline: **113**. After this plan: **119**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`) — do not redeclare it.
- `jobId` is a **string** (like Corporation Projects' `projectId`) — do not "fix" this to int64.
- `CharacterFreelanceJobs`/`CharacterFreelanceJobParticipation` use the client's own baked-in `characterID` (`int64(c.characterID)`) — genuinely "my own character," same convention as `CharacterFleet`.
- `CorporationFreelanceJobs`/`CorporationFreelanceJobParticipants`/`FreelanceJobsListing` are **cursor-paginated** (`after, before string, limit int64` — empty/`0` omits the query param; return `(items, *esiapi.Cursor, error)`), matching the Corporation Projects convention exactly — do NOT use `totalPagesFromHeader`.
- `FreelanceJobsListing` additionally takes an optional `corporationID int64` filter (`0` omits it) — confirmed from the generated request builder's `.CorporationId(int64)` method.
- **Verified model layout quirk**: unlike Corporation Projects' `CorporationsProjectsDetail`, `FreelanceJobsDetail` does **not** have a top-level `Creator` field — `Creator` is nested inside `Details` (`detail.Details.Creator.Character`/`.Corporation`). Do not assume a top-level `detail.Creator` — it does not exist and will not compile.
- `Configuration.Parameters` is `map[string]FreelanceJobsDetailConfigurationParametersValue`, a 4-variant oneOf (`Boolean`, `CorporationItemDelivery`, `Matcher`, `Options`), each double-wrapped exactly like Corporation Projects' variants. Flattened via `toConfigParameter`'s 4-way switch into `ConfigParameter{Kind string, ...}` — defined once, not per-map-entry.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean) — including catching and fixing the top-level-`Creator` mistake during verification, which is why the code below is correct as written. Implementers should still run these themselves as part of each task.

---

### Task 1: `esi/freelance_jobs.go` facade (6 methods)

**Files:**
- Create: `esi/freelance_jobs.go`
- Test: `esi/freelance_jobs_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `Client.api`/`Client.characterID` (`esi/client.go`).
- Produces: 6 methods on `*Client` for Task 2's `mcp/freelance_jobs.go` to call.

- [ ] **Step 1: Write `esi/freelance_jobs.go`**

```go
// Package esi's freelance_jobs.go covers all 6 of "Freelance Jobs"'s endpoints. 2 are private
// character-scoped ("my own character" -- CharacterFreelanceJobs/CharacterFreelanceJobParticipation
// take no explicit characterID, using int64(c.characterID)); 2 are private corp-scoped,
// cursor-paginated; 2 are public, cursor-paginated. jobId is a string throughout, like Corporation
// Projects' projectId.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterFreelanceJobs returns the client's own character's freelance jobs. Not paginated.
func (c *Client) CharacterFreelanceJobs(ctx context.Context) ([]esiapi.FreelanceJobsDetailFreelancejob, error) {
	result, resp, err := c.api.FreelanceJobsAPI.GetCharactersFreelanceJobsListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character freelance jobs: %w", describeError(err, resp))
	}
	return result.FreelanceJobs, nil
}

// CharacterFreelanceJobParticipation returns the client's own character's participation in
// jobID. Not paginated.
func (c *Client) CharacterFreelanceJobParticipation(ctx context.Context, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error) {
	result, resp, err := c.api.FreelanceJobsAPI.GetCharactersFreelanceJobsParticipation(ctx, int64(c.characterID), jobID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character freelance job participation: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationFreelanceJobs returns corporationID's freelance jobs, cursor-paginated. after=""
// and before="" each omit their own query param (mutually exclusive); limit=0 omits the limit
// param.
func (c *Client) CorporationFreelanceJobs(ctx context.Context, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	req := c.api.FreelanceJobsAPI.GetCorporationsFreelanceJobsListing(ctx, corporationID)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get corporation freelance jobs: %w", describeError(err, resp))
	}
	return result.FreelanceJobs, result.Cursor, nil
}

// CorporationFreelanceJobParticipants returns the participants in corporationID's job jobID,
// cursor-paginated.
func (c *Client) CorporationFreelanceJobParticipants(ctx context.Context, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error) {
	req := c.api.FreelanceJobsAPI.GetCorporationsFreelanceJobsParticipants(ctx, corporationID, jobID)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get corporation freelance job participants: %w", describeError(err, resp))
	}
	return result.Participants, result.Cursor, nil
}

// FreelanceJobDetail returns the full details of jobID, including its nested Configuration
// (whose Parameters map values are a 4-variant oneOf -- see mcp/freelance_jobs.go's
// ConfigParameter and toConfigParameter for the flattened shape). Public, no auth required.
func (c *Client) FreelanceJobDetail(ctx context.Context, jobID string) (*esiapi.FreelanceJobsDetail, error) {
	result, resp, err := c.api.FreelanceJobsAPI.GetFreelanceJobsDetail(ctx, jobID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get freelance job detail: %w", describeError(err, resp))
	}
	return result, nil
}

// FreelanceJobsListing returns freelance jobs game-wide, cursor-paginated and optionally
// filtered to one corporation's jobs (corporationID=0 omits the filter). Public, no auth
// required.
func (c *Client) FreelanceJobsListing(ctx context.Context, after, before string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	req := c.api.FreelanceJobsAPI.GetFreelanceJobsListing(ctx)
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	if corporationID > 0 {
		req = req.CorporationId(corporationID)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, nil, fmt.Errorf("get freelance jobs listing: %w", describeError(err, resp))
	}
	return result.FreelanceJobs, result.Cursor, nil
}
```

- [ ] **Step 2: Write `esi/freelance_jobs_test.go`**

Read `esi/corporation_projects_test.go` first to copy its exact patterns for cursor pagination (omit-when-empty/set-when-present via raw query string checks) and the general private/public client harness split. Cover:

- `TestCharacterFreelanceJobs_ParsesResponse` — serve `{freelance_jobs: [{id, last_modified, name, progress: {current, desired}, reward: {initial, remaining}, state}]}`, assert every field maps; use the authenticated-client harness and assert the request path reflects the client's own baked-in characterID.
- `TestCharacterFreelanceJobs_HTTPError` — serve a 404, assert error non-nil.
- `TestCharacterFreelanceJobParticipation_ParsesResponse` — serve `{contributed, last_modified, state}`, assert every field maps.
- `TestCorporationFreelanceJobs_ParsesResponseAndCursorOmittedWhenEmpty` — one subtest asserting field mapping (including `cursor`), one subtest asserting no `after`/`before`/`limit` query params when all are empty/zero.
- `TestCorporationFreelanceJobs_CursorSentWhenSet` — call with `after="abc"`, assert the query string contains exactly that.
- `TestCorporationFreelanceJobParticipants_ParsesResponseAndCursor` — serve `{cursor: {after, before}, participants: [{contributed, id, name, state}]}`, assert every field maps and cursor round-trips.
- `TestFreelanceJobDetail_ParsesResponse` — serve a full `FreelanceJobsDetail` with `configuration.parameters` containing at least one entry for EACH of the 4 variants (`boolean`, `corporation_item_delivery`, `matcher`, `options`), and `details.creator` populated — assert every top-level field maps AND that `Details.Creator.Character`/`.Corporation` map correctly (this is the field that must NOT be assumed top-level).
- `TestFreelanceJobsListing_CursorAndCorporationFilterOmittedWhenEmpty` — call with all filter params empty/zero, assert no `after`/`before`/`limit`/`corporation_id` in the query string.
- `TestFreelanceJobsListing_CursorAndCorporationFilterSentWhenSet` — call with `before="xyz"`, `limit=5`, `corporationID=98000001`, assert exactly those 3 query params are sent and `after` is absent.

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run "FreelanceJob" -v`
Expected: all tests PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/freelance_jobs.go esi/freelance_jobs_test.go
git commit -m "feat: add esi/freelance_jobs.go facade (cursor pagination, nested 4-variant Configuration)"
```

---

### Task 2: `mcp/freelance_jobs.go` tools (6 tools), server registration, and docs sync

**Files:**
- Create: `mcp/freelance_jobs.go`
- Test: `mcp/freelance_jobs_internal_test.go`
- Modify: `mcp/server.go` (register 6 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 113 to 119)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Freelance Jobs row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "one hundred thirteen" → "one hundred nineteen")
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: all 6 `esi/freelance_jobs.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent`.
- Produces: 6 `mcpsdk.ToolHandlerFor[...]`-returning functions: `listCharacterFreelanceJobsHandler`, `getCharacterFreelanceJobParticipationHandler`, `listCorporationFreelanceJobsHandler`, `listCorporationFreelanceJobParticipantsHandler` (all `(version, clientID string, store *config.Store)`), `getFreelanceJobHandler`, `listFreelanceJobsHandler` (both `(version string)` only).

- [ ] **Step 1: Write `mcp/freelance_jobs.go`**

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

// --- shared types ---

type JobProgress struct {
	Current int64 `json:"current"`
	Desired int64 `json:"desired"`
}

func toJobProgress(p esiapi.FreelanceJobsDetailProgress) JobProgress {
	return JobProgress{Current: p.Current, Desired: p.Desired}
}

type JobReward struct {
	Initial   float64 `json:"initial"`
	Remaining float64 `json:"remaining"`
}

func toJobReward(r *esiapi.FreelanceJobsDetailReward) *JobReward {
	if r == nil {
		return nil
	}
	return &JobReward{Initial: r.Initial, Remaining: r.Remaining}
}

type FreelanceJobSummary struct {
	JobID        string      `json:"jobId"`
	Name         string      `json:"name"`
	State        string      `json:"state"`
	LastModified time.Time   `json:"lastModified"`
	Progress     JobProgress `json:"progress"`
	Reward       *JobReward  `json:"reward,omitempty"`
}

func toFreelanceJobSummary(j esiapi.FreelanceJobsDetailFreelancejob) FreelanceJobSummary {
	return FreelanceJobSummary{
		JobID: j.Id, Name: j.Name, State: j.State, LastModified: j.LastModified,
		Progress: toJobProgress(j.Progress), Reward: toJobReward(j.Reward),
	}
}

func toFreelanceJobSummaries(js []esiapi.FreelanceJobsDetailFreelancejob) []FreelanceJobSummary {
	out := make([]FreelanceJobSummary, len(js))
	for i, j := range js {
		out[i] = toFreelanceJobSummary(j)
	}
	return out
}

// --- list_character_freelance_jobs (private) ---

var listCharacterFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.FreelanceJobsDetailFreelancejob, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterFreelanceJobs(ctx)
}

type ListCharacterFreelanceJobsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterFreelanceJobsOutput struct {
	CharacterID int32                 `json:"characterId"`
	Jobs        []FreelanceJobSummary `json:"jobs"`
}

func listCharacterFreelanceJobsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterFreelanceJobsInput, ListCharacterFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterFreelanceJobsInput) (*mcpsdk.CallToolResult, ListCharacterFreelanceJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFreelanceJobsOutput{}, err
		}
		if !found {
			return nil, ListCharacterFreelanceJobsOutput{}, fmt.Errorf("list character freelance jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		jobs, err := listCharacterFreelanceJobs(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterFreelanceJobsOutput{}, fmt.Errorf("list character freelance jobs: %w", err)
		}
		return nil, ListCharacterFreelanceJobsOutput{CharacterID: in.CharacterID, Jobs: toFreelanceJobSummaries(jobs)}, nil
	}
}

// --- get_character_freelance_job_participation (private) ---

var getCharacterFreelanceJobParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent string, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterFreelanceJobParticipation(ctx, jobID)
}

type GetCharacterFreelanceJobParticipationInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check"`
	JobID       string `json:"jobId" jsonschema:"the freelance job ID"`
}
type GetCharacterFreelanceJobParticipationOutput struct {
	CharacterID  int32     `json:"characterId"`
	JobID        string    `json:"jobId"`
	State        string    `json:"state"`
	Contributed  int64     `json:"contributed"`
	LastModified time.Time `json:"lastModified"`
}

func getCharacterFreelanceJobParticipationHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterFreelanceJobParticipationInput, GetCharacterFreelanceJobParticipationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFreelanceJobParticipationInput) (*mcpsdk.CallToolResult, GetCharacterFreelanceJobParticipationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, err
		}
		if !found {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, fmt.Errorf("get character freelance job participation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		p, err := getCharacterFreelanceJobParticipation(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.JobID)
		if err != nil {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, fmt.Errorf("get character freelance job participation: %w", err)
		}
		return nil, GetCharacterFreelanceJobParticipationOutput{
			CharacterID: in.CharacterID, JobID: in.JobID, State: p.State, Contributed: p.Contributed, LastModified: p.LastModified,
		}, nil
	}
}

// --- list_corporation_freelance_jobs (private, cursor-paginated) ---

var listCorporationFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationFreelanceJobs(ctx, corporationID, after, before, limit)
}

type ListCorporationFreelanceJobsInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation ID to look up"`
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
}
type ListCorporationFreelanceJobsOutput struct {
	CharacterID   int32                 `json:"characterId"`
	CorporationID int64                 `json:"corporationId"`
	Jobs          []FreelanceJobSummary `json:"jobs"`
	NextAfter     *string               `json:"nextAfter,omitempty"`
	NextBefore    *string               `json:"nextBefore,omitempty"`
}

func listCorporationFreelanceJobsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationFreelanceJobsInput, ListCorporationFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFreelanceJobsInput) (*mcpsdk.CallToolResult, ListCorporationFreelanceJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobsOutput{}, err
		}
		if !found {
			return nil, ListCorporationFreelanceJobsOutput{}, fmt.Errorf("list corporation freelance jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		jobs, cursor, err := listCorporationFreelanceJobs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListCorporationFreelanceJobsOutput{}, fmt.Errorf("list corporation freelance jobs: %w", err)
		}
		out := ListCorporationFreelanceJobsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Jobs: toFreelanceJobSummaries(jobs)}
		if cursor != nil {
			out.NextAfter = cursor.After
			out.NextBefore = cursor.Before
		}
		return nil, out, nil
	}
}

// --- list_corporation_freelance_job_participants (private, cursor-paginated) ---

type JobParticipant struct {
	ParticipantID int64  `json:"participantId"`
	Name          string `json:"name"`
	State         string `json:"state"`
	Contributed   int64  `json:"contributed"`
}

var listCorporationFreelanceJobParticipants = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationFreelanceJobParticipants(ctx, corporationID, jobID, after, before, limit)
}

type ListCorporationFreelanceJobParticipantsInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation ID the job belongs to"`
	JobID         string `json:"jobId" jsonschema:"the freelance job ID"`
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
}
type ListCorporationFreelanceJobParticipantsOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	JobID         string           `json:"jobId"`
	Participants  []JobParticipant `json:"participants"`
	NextAfter     *string          `json:"nextAfter,omitempty"`
	NextBefore    *string          `json:"nextBefore,omitempty"`
}

func listCorporationFreelanceJobParticipantsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationFreelanceJobParticipantsInput, ListCorporationFreelanceJobParticipantsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFreelanceJobParticipantsInput) (*mcpsdk.CallToolResult, ListCorporationFreelanceJobParticipantsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, err
		}
		if !found {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, fmt.Errorf("list corporation freelance job participants for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		participants, cursor, err := listCorporationFreelanceJobParticipants(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.JobID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, fmt.Errorf("list corporation freelance job participants: %w", err)
		}
		out := make([]JobParticipant, len(participants))
		for i, p := range participants {
			out[i] = JobParticipant{ParticipantID: p.Id, Name: p.Name, State: p.State, Contributed: p.Contributed}
		}
		result := ListCorporationFreelanceJobParticipantsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, JobID: in.JobID, Participants: out}
		if cursor != nil {
			result.NextAfter = cursor.After
			result.NextBefore = cursor.Before
		}
		return nil, result, nil
	}
}

// --- get_freelance_job (public) ---

type ParameterMatcherValue struct {
	ValueType string   `json:"valueType"`
	Values    []string `json:"values"`
}

type ParameterMatcher struct {
	Values []ParameterMatcherValue `json:"values"`
}

func toParameterMatcher(m esiapi.FreelanceJobsDetailParametermatcher) ParameterMatcher {
	values := make([]ParameterMatcherValue, len(m.Values))
	for i, v := range m.Values {
		values[i] = ParameterMatcherValue{ValueType: v.ValueType, Values: v.Values}
	}
	return ParameterMatcher{Values: values}
}

type ParameterCorporationItemDelivery struct {
	CorporationOfficeLocation ParameterMatcher `json:"corporationOfficeLocation"`
	ItemType                  ParameterMatcher `json:"itemType"`
}

type ParameterOptions struct {
	Selected []string `json:"selected"`
}

// ConfigParameter is the flattened form of FreelanceJobsDetailConfigurationParametersValue, a
// 4-variant oneOf. Kind names the active variant ("boolean", "corporation_item_delivery",
// "matcher", "options"); only the fields relevant to that Kind are populated.
type ConfigParameter struct {
	Kind                    string                            `json:"kind"`
	BooleanValue            *bool                             `json:"booleanValue,omitempty"`
	CorporationItemDelivery *ParameterCorporationItemDelivery `json:"corporationItemDelivery,omitempty"`
	Matcher                 *ParameterMatcher                 `json:"matcher,omitempty"`
	Options                 *ParameterOptions                 `json:"options,omitempty"`
}

func toConfigParameter(v esiapi.FreelanceJobsDetailConfigurationParametersValue) ConfigParameter {
	switch {
	case v.Boolean != nil && v.Boolean.Boolean != nil:
		val := v.Boolean.Boolean.Value
		return ConfigParameter{Kind: "boolean", BooleanValue: &val}
	case v.CorporationItemDelivery != nil && v.CorporationItemDelivery.CorporationItemDelivery != nil:
		d := v.CorporationItemDelivery.CorporationItemDelivery
		return ConfigParameter{Kind: "corporation_item_delivery", CorporationItemDelivery: &ParameterCorporationItemDelivery{
			CorporationOfficeLocation: toParameterMatcher(d.CorporationOfficeLocation),
			ItemType:                  toParameterMatcher(d.ItemType),
		}}
	case v.Matcher != nil && v.Matcher.Matcher != nil:
		m := toParameterMatcher(*v.Matcher.Matcher)
		return ConfigParameter{Kind: "matcher", Matcher: &m}
	case v.Options != nil && v.Options.Options != nil:
		return ConfigParameter{Kind: "options", Options: &ParameterOptions{Selected: v.Options.Options.Selected}}
	default:
		return ConfigParameter{}
	}
}

type JobConfiguration struct {
	Method     string                     `json:"method"`
	Parameters map[string]ConfigParameter `json:"parameters"`
	Version    int64                      `json:"version"`
}

func toJobConfiguration(cfg esiapi.FreelanceJobsDetailConfiguration) JobConfiguration {
	params := make(map[string]ConfigParameter, len(cfg.Parameters))
	for k, v := range cfg.Parameters {
		params[k] = toConfigParameter(v)
	}
	return JobConfiguration{Method: cfg.Method, Parameters: params, Version: cfg.Version}
}

type JobContributionLimits struct {
	ContributionPerParticipantLimit *int64   `json:"contributionPerParticipantLimit,omitempty"`
	MaxCommittedParticipants        int64    `json:"maxCommittedParticipants"`
	RewardPerContribution           *float64 `json:"rewardPerContribution,omitempty"`
	SubmissionLimit                 *int64   `json:"submissionLimit,omitempty"`
	SubmissionMultiplier            *float64 `json:"submissionMultiplier,omitempty"`
}

func toJobContributionLimits(c *esiapi.FreelanceJobsDetailContribution) *JobContributionLimits {
	if c == nil {
		return nil
	}
	return &JobContributionLimits{
		ContributionPerParticipantLimit: c.ContributionPerParticipantLimit, MaxCommittedParticipants: c.MaxCommittedParticipants,
		RewardPerContribution: c.RewardPerContribution, SubmissionLimit: c.SubmissionLimit, SubmissionMultiplier: c.SubmissionMultiplier,
	}
}

type JobCreatorEntity struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type JobCreator struct {
	Character   JobCreatorEntity `json:"character"`
	Corporation JobCreatorEntity `json:"corporation"`
}

type JobDetails struct {
	Career      string     `json:"career"`
	Created     time.Time  `json:"created"`
	Description string     `json:"description"`
	Expires     *time.Time `json:"expires,omitempty"`
	Finished    *time.Time `json:"finished,omitempty"`
}

type JobBroadcastLocation struct {
	SystemID int64  `json:"systemId"`
	Name     string `json:"name"`
}
type JobRestrictions struct {
	MinimumAge *int64 `json:"minimumAge,omitempty"`
	MaximumAge *int64 `json:"maximumAge,omitempty"`
}
type JobAccessAndVisibility struct {
	AclProtected       bool                   `json:"aclProtected"`
	BroadcastLocations []JobBroadcastLocation `json:"broadcastLocations,omitempty"`
	Restrictions       *JobRestrictions       `json:"restrictions,omitempty"`
}

var getFreelanceJob = func(ctx context.Context, userAgent string, jobID string) (*esiapi.FreelanceJobsDetail, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FreelanceJobDetail(ctx, jobID)
}

type GetFreelanceJobInput struct {
	JobID string `json:"jobId" jsonschema:"the freelance job ID"`
}
type GetFreelanceJobOutput struct {
	JobID               string                 `json:"jobId"`
	Name                string                 `json:"name"`
	State               string                 `json:"state"`
	LastModified        time.Time              `json:"lastModified"`
	Progress            JobProgress            `json:"progress"`
	Reward              *JobReward             `json:"reward,omitempty"`
	Configuration       JobConfiguration       `json:"configuration"`
	Contribution        *JobContributionLimits `json:"contribution,omitempty"`
	Creator             JobCreator             `json:"creator"`
	Details             JobDetails             `json:"details"`
	AccessAndVisibility JobAccessAndVisibility `json:"accessAndVisibility"`
}

func getFreelanceJobHandler(version string) mcpsdk.ToolHandlerFor[GetFreelanceJobInput, GetFreelanceJobOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFreelanceJobInput) (*mcpsdk.CallToolResult, GetFreelanceJobOutput, error) {
		detail, err := getFreelanceJob(ctx, sso.UserAgent(version), in.JobID)
		if err != nil {
			return nil, GetFreelanceJobOutput{}, fmt.Errorf("get freelance job: %w", err)
		}
		locations := make([]JobBroadcastLocation, len(detail.AccessAndVisibility.BroadcastLocations))
		for i, l := range detail.AccessAndVisibility.BroadcastLocations {
			locations[i] = JobBroadcastLocation{SystemID: l.Id, Name: l.Name}
		}
		var restrictions *JobRestrictions
		if detail.AccessAndVisibility.Restrictions != nil {
			restrictions = &JobRestrictions{
				MinimumAge: detail.AccessAndVisibility.Restrictions.MinimumAge,
				MaximumAge: detail.AccessAndVisibility.Restrictions.MaximumAge,
			}
		}
		return nil, GetFreelanceJobOutput{
			JobID: in.JobID, Name: detail.Name, State: detail.State, LastModified: detail.LastModified,
			Progress: toJobProgress(detail.Progress), Reward: toJobReward(detail.Reward),
			Configuration: toJobConfiguration(detail.Configuration), Contribution: toJobContributionLimits(detail.Contribution),
			Creator: JobCreator{
				Character:   JobCreatorEntity{ID: detail.Details.Creator.Character.Id, Name: detail.Details.Creator.Character.Name},
				Corporation: JobCreatorEntity{ID: detail.Details.Creator.Corporation.Id, Name: detail.Details.Creator.Corporation.Name},
			},
			Details: JobDetails{
				Career: detail.Details.Career, Created: detail.Details.Created, Description: detail.Details.Description,
				Expires: detail.Details.Expires, Finished: detail.Details.Finished,
			},
			AccessAndVisibility: JobAccessAndVisibility{
				AclProtected: detail.AccessAndVisibility.AclProtected, BroadcastLocations: locations, Restrictions: restrictions,
			},
		}, nil
	}
}

// --- list_freelance_jobs (public, cursor-paginated) ---

var listFreelanceJobs = func(ctx context.Context, userAgent string, after, before string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FreelanceJobsListing(ctx, after, before, limit, corporationID)
}

type ListFreelanceJobsInput struct {
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
	CorporationID int64  `json:"corporationId,omitempty" jsonschema:"filter to only this corporation's jobs; omit for all corporations"`
}
type ListFreelanceJobsOutput struct {
	Jobs       []FreelanceJobSummary `json:"jobs"`
	NextAfter  *string               `json:"nextAfter,omitempty"`
	NextBefore *string               `json:"nextBefore,omitempty"`
}

func listFreelanceJobsHandler(version string) mcpsdk.ToolHandlerFor[ListFreelanceJobsInput, ListFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFreelanceJobsInput) (*mcpsdk.CallToolResult, ListFreelanceJobsOutput, error) {
		jobs, cursor, err := listFreelanceJobs(ctx, sso.UserAgent(version), in.After, in.Before, in.Limit, in.CorporationID)
		if err != nil {
			return nil, ListFreelanceJobsOutput{}, fmt.Errorf("list freelance jobs: %w", err)
		}
		out := ListFreelanceJobsOutput{Jobs: toFreelanceJobSummaries(jobs)}
		if cursor != nil {
			out.NextAfter = cursor.After
			out.NextBefore = cursor.Before
		}
		return nil, out, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Write `mcp/freelance_jobs_internal_test.go`**

Follow `mcp/corporation_projects_internal_test.go`'s pattern (package-level `var` override + `t.Cleanup`; `keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). Cover:

- `TestListCharacterFreelanceJobsHandler_MapsFields` — stub returns a populated `[]esiapi.FreelanceJobsDetailFreelancejob`, assert every field maps.
- `TestListCharacterFreelanceJobsHandler_UnknownCharacter`
- `TestGetCharacterFreelanceJobParticipationHandler_MapsFields` — stub returns a fully-populated `*esiapi.CharactersFreelanceJobsParticipation`, assert every field maps.
- `TestGetCharacterFreelanceJobParticipationHandler_UnknownCharacter`
- `TestListCorporationFreelanceJobsHandler_MapsFieldsAndCursor` — stub returns jobs plus a non-nil `*esiapi.Cursor`, assert `NextAfter`/`NextBefore` map.
- `TestListCorporationFreelanceJobsHandler_UnknownCharacter`
- `TestListCorporationFreelanceJobParticipantsHandler_MapsFieldsAndCursor` — same cursor check for participants.
- `TestListCorporationFreelanceJobParticipantsHandler_UnknownCharacter`
- `TestGetFreelanceJobHandler_ConfigurationBooleanVariant` — stub `getFreelanceJob` to return a `*esiapi.FreelanceJobsDetail` with one `configuration.parameters` entry set to the `Boolean` variant, plus `details.creator` populated; assert `Configuration.Parameters["<key>"].Kind == "boolean"` and `.BooleanValue` matches, AND assert `Creator.Character`/`.Corporation` map from `detail.Details.Creator` (not any top-level field).
- `TestGetFreelanceJobHandler_ConfigurationCorporationItemDeliveryVariant` — same detail stub with a `CorporationItemDelivery` variant entry (nesting 2 `Matcher`s), assert `Kind == "corporation_item_delivery"` and both nested `Matcher`s map correctly.
- `TestGetFreelanceJobHandler_ConfigurationMatcherVariant` — `Matcher` variant entry, assert `Kind == "matcher"` and `Values` map.
- `TestGetFreelanceJobHandler_ConfigurationOptionsVariant` — `Options` variant entry, assert `Kind == "options"` and `Selected` maps.
- `TestListFreelanceJobsHandler_MapsFieldsAndCursor` — stub returns jobs plus a cursor, assert mapping.

That's 13 test functions total: 4 happy-path/unknown-character pairs (`list_character_freelance_jobs`, `get_character_freelance_job_participation`, `list_corporation_freelance_jobs`, `list_corporation_freelance_job_participants` — 8 tests), 4 Configuration-variant tests for `get_freelance_job`, and 1 happy-path test for `list_freelance_jobs`.

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run "FreelanceJob" -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 6 tools in `mcp/server.go`**

Add these 6 `mcpsdk.AddTool` blocks immediately after the existing `delete_fleet_squad` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_freelance_jobs",
		Description: "List a logged-in EVE Online character's freelance jobs.",
	}, listCharacterFreelanceJobsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_freelance_job_participation",
		Description: "Get a logged-in character's contribution to a freelance job.",
	}, getCharacterFreelanceJobParticipationHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_freelance_jobs",
		Description: "List a corporation's freelance jobs, using a logged-in character's roles to authorize.",
	}, listCorporationFreelanceJobsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_freelance_job_participants",
		Description: "List the participants in a corporation's freelance job, using a logged-in character's roles to authorize.",
	}, listCorporationFreelanceJobParticipantsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_freelance_job",
		Description: "Get the full details of a freelance job (public data, no character required).",
	}, getFreelanceJobHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_freelance_jobs",
		Description: "List freelance jobs game-wide, optionally filtered by corporation (public data, no character required).",
	}, listFreelanceJobsHandler(version))

	return server
}
```

Update the package doc comment above `NewServer` to append the 6 new tool names to its running
list (ending in `..., delete_fleet_squad, list_character_freelance_jobs,
get_character_freelance_job_participation, list_corporation_freelance_jobs,
list_corporation_freelance_job_participants, get_freelance_job, and list_freelance_jobs
registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 6 names to the `[]string{...}` literal, right after `"delete_fleet_squad",`:

```go
			"list_character_freelance_jobs", "get_character_freelance_job_participation",
			"list_corporation_freelance_jobs", "list_corporation_freelance_job_participants",
			"get_freelance_job", "list_freelance_jobs",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 119 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "one hundred thirteen" to "one hundred nineteen".
- Append the 6 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Freelance Jobs row from `Planned` to `This plan`.

- [ ] **Step 8: Commit**

```bash
git add mcp/freelance_jobs.go mcp/freelance_jobs_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add freelance jobs MCP tools (6 tools: cursor pagination, 4-variant Configuration)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff a04d6ae..HEAD` — the design spec commit forward).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
