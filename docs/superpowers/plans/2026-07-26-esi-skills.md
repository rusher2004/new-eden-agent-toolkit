# ESI Skills Implementation Plan (twenty-ninth batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Skills" tag (3 endpoints: character attributes, skill
queue, trained skills) as 3 new MCP tools, folding into the same shared PR #6 branch as every
prior batch.

**Architecture:** All 3 endpoints are private/"my own character" with no auth complexity —
combined into a single implementation task, following the precedent set by
Incursions/Insurance/Routes/Search. `esi/skills.go` wraps the 3 generated `SkillsAPIService`
methods. `mcp/skills.go` exposes 3 tools on top of that, reusing
`characterExists`/`ErrCharacterNotFound`.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: 167 MCP tools (once Search lands — this plan assumes Search's
  28th-batch commit is already merged). This plan brings the count to 170. **If Search has not yet
  landed when this plan is executed, adjust the baseline count and the registration anchor point
  accordingly** — register after whatever the actual last tool in `mcp/server.go` is at execution
  time, and use `grep -c 'mcpsdk.AddTool' mcp/server.go` for the real current count before writing
  any doc-count edits.
- All 3 endpoints require an OAuth2 scope (`esi-skills.read_skills.v1` for attributes and skills,
  `esi-skills.read_skillqueue.v1` for the queue) — private, "my own character", no caller-supplied
  character ID. Verified in `esi/api/openapi.yaml`. None are paginated.
- **Naming trap, already caught and avoided during sandbox verification**: the generated client has
  two similarly-named skillqueue-entry types — `CharactersSkillqueueSkill` and
  `CharactersCharacterIdSkillqueueGetInner`. `GetCharactersCharacterIdSkillqueueExecute`'s actual
  return type is `[]CharactersSkillqueueSkill` — confirmed by reading the generated method
  signature directly. Do not assume the other, similarly-named type is correct just because it
  matches the endpoint's own name more closely.
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- Reuse `characterExists(store *config.Store, characterID int32) (bool, error)` and
  `ErrCharacterNotFound` from `mcp/character.go` — do not redeclare.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-skills-design.md` for the verification detail.

---

### Task 1: `esi/skills.go` + `mcp/skills.go` — full batch in one task

**Files:**
- Create: `esi/skills.go`
- Test: `esi/skills_test.go`
- Create: `mcp/skills.go`
- Test: `mcp/skills_internal_test.go`
- Modify: `mcp/server.go` (register 3 tools after whatever the actual last-registered tool is at
  execution time; update package doc comment)
- Modify: `mcp/server_test.go` (append 3 tool names to expected-tools list)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Skills row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (increment tool count by 3 from whatever it currently reads, append 3 tool names)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Produces: 3 tool names — `get_character_attributes`, `list_character_skill_queue`,
  `list_character_skills` — registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `esi/skills_test.go`:

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

func TestCharacterAttributes_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"accrued_remap_cooldown_date":"2026-08-01T00:00:00Z","bonus_remaps":1,"charisma":20,"intelligence":21,"last_remap_date":"2026-01-01T00:00:00Z","memory":22,"perception":23,"willpower":24}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	attrs, err := client.CharacterAttributes(context.Background())
	if err != nil {
		t.Fatalf("CharacterAttributes returned error: %v", err)
	}
	if attrs.Charisma != 20 || attrs.Intelligence != 21 || attrs.Memory != 22 || attrs.Perception != 23 || attrs.Willpower != 24 {
		t.Errorf("attrs = %+v, want Charisma=20 Intelligence=21 Memory=22 Perception=23 Willpower=24", attrs)
	}
	wantCooldown := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if attrs.AccruedRemapCooldownDate == nil || !attrs.AccruedRemapCooldownDate.Equal(wantCooldown) {
		t.Errorf("AccruedRemapCooldownDate = %v, want %v", attrs.AccruedRemapCooldownDate, wantCooldown)
	}
	if attrs.BonusRemaps == nil || *attrs.BonusRemaps != 1 {
		t.Errorf("BonusRemaps = %v, want 1", attrs.BonusRemaps)
	}
	wantLastRemap := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if attrs.LastRemapDate == nil || !attrs.LastRemapDate.Equal(wantLastRemap) {
		t.Errorf("LastRemapDate = %v, want %v", attrs.LastRemapDate, wantLastRemap)
	}
}

func TestCharacterAttributes_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterAttributes(context.Background()); err == nil {
		t.Fatal("CharacterAttributes returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterSkillQueue_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"finish_date":"2026-08-01T00:00:00Z","finished_level":5,"level_end_sp":256000,"level_start_sp":90510,"queue_position":0,"skill_id":3300,"start_date":"2026-07-20T00:00:00Z","training_start_sp":90510}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	queue, err := client.CharacterSkillQueue(context.Background())
	if err != nil {
		t.Fatalf("CharacterSkillQueue returned error: %v", err)
	}
	if len(queue) != 1 {
		t.Fatalf("queue = %+v, want one entry", queue)
	}
	q := queue[0]
	if q.SkillId != 3300 || q.FinishedLevel != 5 || q.QueuePosition != 0 {
		t.Errorf("queue[0] = %+v, want SkillId=3300 FinishedLevel=5 QueuePosition=0", q)
	}
	wantFinish := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if q.FinishDate == nil || !q.FinishDate.Equal(wantFinish) {
		t.Errorf("FinishDate = %v, want %v", q.FinishDate, wantFinish)
	}
	wantStart := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if q.StartDate == nil || !q.StartDate.Equal(wantStart) {
		t.Errorf("StartDate = %v, want %v", q.StartDate, wantStart)
	}
	if q.LevelStartSp == nil || *q.LevelStartSp != 90510 {
		t.Errorf("LevelStartSp = %v, want 90510", q.LevelStartSp)
	}
	if q.LevelEndSp == nil || *q.LevelEndSp != 256000 {
		t.Errorf("LevelEndSp = %v, want 256000", q.LevelEndSp)
	}
	if q.TrainingStartSp == nil || *q.TrainingStartSp != 90510 {
		t.Errorf("TrainingStartSp = %v, want 90510", q.TrainingStartSp)
	}
}

func TestCharacterSkillQueue_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterSkillQueue(context.Background()); err == nil {
		t.Fatal("CharacterSkillQueue returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterSkills_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"skills":[{"active_skill_level":5,"skill_id":3300,"skillpoints_in_skill":256000,"trained_skill_level":5}],"total_sp":5000000,"unallocated_sp":1000}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	skills, err := client.CharacterSkills(context.Background())
	if err != nil {
		t.Fatalf("CharacterSkills returned error: %v", err)
	}
	if skills.TotalSp != 5000000 {
		t.Errorf("TotalSp = %d, want 5000000", skills.TotalSp)
	}
	if skills.UnallocatedSp == nil || *skills.UnallocatedSp != 1000 {
		t.Errorf("UnallocatedSp = %v, want 1000", skills.UnallocatedSp)
	}
	if len(skills.Skills) != 1 {
		t.Fatalf("Skills = %+v, want one entry", skills.Skills)
	}
	s := skills.Skills[0]
	if s.SkillId != 3300 || s.ActiveSkillLevel != 5 || s.TrainedSkillLevel != 5 || s.SkillpointsInSkill != 256000 {
		t.Errorf("Skills[0] = %+v, want SkillId=3300 ActiveSkillLevel=5 TrainedSkillLevel=5 SkillpointsInSkill=256000", s)
	}
}

func TestCharacterSkills_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterSkills(context.Background()); err == nil {
		t.Fatal("CharacterSkills returned nil error, want an error for HTTP 404")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterAttributes|TestCharacterSkillQueue|TestCharacterSkills' -v`
Expected: FAIL — `esi.Client` has none of these methods yet.

- [ ] **Step 3: Write the esi/ implementation**

Create `esi/skills.go`:

```go
// Package esi's skills.go covers all 3 of "Skills"'s endpoints -- character attributes, skill
// queue, and trained skills. All 3 are private/"my own character" -- verified against
// esi/api/openapi.yaml's security blocks (esi-skills.read_skills.v1 for attributes and skills,
// esi-skills.read_skillqueue.v1 for the queue), never a caller-supplied character ID.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterAttributes returns the client's own character's neural attributes (Charisma,
// Intelligence, Memory, Perception, Willpower) and remap history.
func (c *Client) CharacterAttributes(ctx context.Context) (*esiapi.CharactersCharacterIdAttributesGet, error) {
	result, resp, err := c.api.SkillsAPI.GetCharactersCharacterIdAttributes(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character attributes: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterSkillQueue returns the client's own character's current skill training queue. The
// generated return type is CharactersSkillqueueSkill -- verified directly from source, NOT the
// similarly-named CharactersCharacterIdSkillqueueGetInner (a distinct type used elsewhere in the
// generated client).
func (c *Client) CharacterSkillQueue(ctx context.Context) ([]esiapi.CharactersSkillqueueSkill, error) {
	result, resp, err := c.api.SkillsAPI.GetCharactersCharacterIdSkillqueue(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character skill queue: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterSkills returns the client's own character's trained skills, total skill points, and
// any unallocated skill points.
func (c *Client) CharacterSkills(ctx context.Context) (*esiapi.CharactersSkills, error) {
	result, resp, err := c.api.SkillsAPI.GetCharactersCharacterIdSkills(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character skills: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestCharacterAttributes|TestCharacterSkillQueue|TestCharacterSkills' -v`
Expected: PASS (6 tests).

- [ ] **Step 5: Write the failing mcp/ tests**

Create `mcp/skills_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// --- get_character_attributes ---

func TestGetCharacterAttributesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterAttributes
	t.Cleanup(func() { getCharacterAttributes = orig })
	cooldown := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	lastRemap := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bonusRemaps := int64(1)
	getCharacterAttributes = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, int64, int64, int64, int64, *time.Time, *int64, *time.Time, error) {
		return 20, 21, 22, 23, 24, &cooldown, &bonusRemaps, &lastRemap, nil
	}

	handler := getCharacterAttributesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterAttributesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.Charisma != 20 || out.Intelligence != 21 || out.Memory != 22 || out.Perception != 23 || out.Willpower != 24 {
		t.Errorf("out = %+v, want CharacterID=7 Charisma=20 Intelligence=21 Memory=22 Perception=23 Willpower=24", out)
	}
	if out.AccruedRemapCooldownDate == nil || !out.AccruedRemapCooldownDate.Equal(cooldown) {
		t.Errorf("AccruedRemapCooldownDate = %v, want %v", out.AccruedRemapCooldownDate, cooldown)
	}
	if out.BonusRemaps == nil || *out.BonusRemaps != 1 {
		t.Errorf("BonusRemaps = %v, want 1", out.BonusRemaps)
	}
	if out.LastRemapDate == nil || !out.LastRemapDate.Equal(lastRemap) {
		t.Errorf("LastRemapDate = %v, want %v", out.LastRemapDate, lastRemap)
	}
}

func TestGetCharacterAttributesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterAttributes
	t.Cleanup(func() { getCharacterAttributes = orig })
	getCharacterAttributes = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, int64, int64, int64, int64, *time.Time, *int64, *time.Time, error) {
		t.Fatal("getCharacterAttributes should not be called for an unknown character")
		return 0, 0, 0, 0, 0, nil, nil, nil, nil
	}

	handler := getCharacterAttributesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterAttributesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_character_skill_queue ---

func TestListCharacterSkillQueueHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterSkillQueue
	t.Cleanup(func() { listCharacterSkillQueue = orig })
	finish := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	levelStartSp := int64(90510)
	levelEndSp := int64(256000)
	trainingStartSp := int64(90510)
	listCharacterSkillQueue = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkillQueueEntry, error) {
		return []SkillQueueEntry{{
			SkillID: 3300, FinishedLevel: 5, QueuePosition: 0,
			FinishDate: &finish, StartDate: &start,
			LevelStartSp: &levelStartSp, LevelEndSp: &levelEndSp, TrainingStartSp: &trainingStartSp,
		}}, nil
	}

	handler := listCharacterSkillQueueHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterSkillQueueInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Queue) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one queue entry", out)
	}
	q := out.Queue[0]
	if q.SkillID != 3300 || q.FinishedLevel != 5 || q.QueuePosition != 0 {
		t.Errorf("queue[0] = %+v, want SkillID=3300 FinishedLevel=5 QueuePosition=0", q)
	}
	if q.FinishDate == nil || !q.FinishDate.Equal(finish) {
		t.Errorf("FinishDate = %v, want %v", q.FinishDate, finish)
	}
	if q.StartDate == nil || !q.StartDate.Equal(start) {
		t.Errorf("StartDate = %v, want %v", q.StartDate, start)
	}
	if q.LevelStartSp == nil || *q.LevelStartSp != 90510 {
		t.Errorf("LevelStartSp = %v, want 90510", q.LevelStartSp)
	}
	if q.LevelEndSp == nil || *q.LevelEndSp != 256000 {
		t.Errorf("LevelEndSp = %v, want 256000", q.LevelEndSp)
	}
	if q.TrainingStartSp == nil || *q.TrainingStartSp != 90510 {
		t.Errorf("TrainingStartSp = %v, want 90510", q.TrainingStartSp)
	}
}

func TestListCharacterSkillQueueHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterSkillQueue
	t.Cleanup(func() { listCharacterSkillQueue = orig })
	listCharacterSkillQueue = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkillQueueEntry, error) {
		t.Fatal("listCharacterSkillQueue should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterSkillQueueHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterSkillQueueInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_character_skills ---

func TestListCharacterSkillsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterSkills
	t.Cleanup(func() { listCharacterSkills = orig })
	unallocatedSp := int64(1000)
	listCharacterSkills = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CharacterSkill, int64, *int64, error) {
		return []CharacterSkill{{
			SkillID: 3300, ActiveSkillLevel: 5, TrainedSkillLevel: 5, SkillpointsInSkill: 256000,
		}}, 5000000, &unallocatedSp, nil
	}

	handler := listCharacterSkillsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterSkillsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.TotalSp != 5000000 {
		t.Fatalf("out = %+v, want CharacterID=7 TotalSp=5000000", out)
	}
	if out.UnallocatedSp == nil || *out.UnallocatedSp != 1000 {
		t.Errorf("UnallocatedSp = %v, want 1000", out.UnallocatedSp)
	}
	if len(out.Skills) != 1 {
		t.Fatalf("Skills = %+v, want one entry", out.Skills)
	}
	s := out.Skills[0]
	if s.SkillID != 3300 || s.ActiveSkillLevel != 5 || s.TrainedSkillLevel != 5 || s.SkillpointsInSkill != 256000 {
		t.Errorf("Skills[0] = %+v, want SkillID=3300 ActiveSkillLevel=5 TrainedSkillLevel=5 SkillpointsInSkill=256000", s)
	}
}

func TestListCharacterSkillsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterSkills
	t.Cleanup(func() { listCharacterSkills = orig })
	listCharacterSkills = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CharacterSkill, int64, *int64, error) {
		t.Fatal("listCharacterSkills should not be called for an unknown character")
		return nil, 0, nil, nil
	}

	handler := listCharacterSkillsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterSkillsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestGetCharacterAttributesHandler|TestListCharacterSkillQueueHandler|TestListCharacterSkillsHandler' -v`
Expected: FAIL — undefined symbols (none of these handlers/types exist yet).

- [ ] **Step 7: Write the mcp/ implementation**

Create `mcp/skills.go`:

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

// --- get_character_attributes (private) ---

var getCharacterAttributes = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, int64, int64, int64, int64, *time.Time, *int64, *time.Time, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, 0, 0, 0, 0, nil, nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	attrs, err := client.CharacterAttributes(ctx)
	if err != nil {
		return 0, 0, 0, 0, 0, nil, nil, nil, err
	}
	return attrs.Charisma, attrs.Intelligence, attrs.Memory, attrs.Perception, attrs.Willpower,
		attrs.AccruedRemapCooldownDate, attrs.BonusRemaps, attrs.LastRemapDate, nil
}

type GetCharacterAttributesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterAttributesOutput struct {
	CharacterID              int32      `json:"characterId"`
	Charisma                 int64      `json:"charisma"`
	Intelligence             int64      `json:"intelligence"`
	Memory                   int64      `json:"memory"`
	Perception               int64      `json:"perception"`
	Willpower                int64      `json:"willpower"`
	AccruedRemapCooldownDate *time.Time `json:"accruedRemapCooldownDate,omitempty"`
	BonusRemaps              *int64     `json:"bonusRemaps,omitempty"`
	LastRemapDate            *time.Time `json:"lastRemapDate,omitempty"`
}

func getCharacterAttributesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterAttributesInput, GetCharacterAttributesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterAttributesInput) (*mcpsdk.CallToolResult, GetCharacterAttributesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterAttributesOutput{}, err
		}
		if !found {
			return nil, GetCharacterAttributesOutput{}, fmt.Errorf("get character attributes for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		charisma, intelligence, memory, perception, willpower, accruedRemapCooldownDate, bonusRemaps, lastRemapDate, err := getCharacterAttributes(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterAttributesOutput{}, fmt.Errorf("get character attributes: %w", err)
		}
		return nil, GetCharacterAttributesOutput{
			CharacterID: in.CharacterID, Charisma: charisma, Intelligence: intelligence, Memory: memory,
			Perception: perception, Willpower: willpower, AccruedRemapCooldownDate: accruedRemapCooldownDate,
			BonusRemaps: bonusRemaps, LastRemapDate: lastRemapDate,
		}, nil
	}
}

// --- list_character_skill_queue (private) ---

type SkillQueueEntry struct {
	SkillID         int64      `json:"skillId"`
	FinishedLevel   int64      `json:"finishedLevel"`
	QueuePosition   int64      `json:"queuePosition"`
	FinishDate      *time.Time `json:"finishDate,omitempty"`
	StartDate       *time.Time `json:"startDate,omitempty"`
	LevelStartSp    *int64     `json:"levelStartSp,omitempty"`
	LevelEndSp      *int64     `json:"levelEndSp,omitempty"`
	TrainingStartSp *int64     `json:"trainingStartSp,omitempty"`
}

var listCharacterSkillQueue = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkillQueueEntry, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	queue, err := client.CharacterSkillQueue(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SkillQueueEntry, len(queue))
	for i, q := range queue {
		out[i] = SkillQueueEntry{
			SkillID: q.SkillId, FinishedLevel: q.FinishedLevel, QueuePosition: q.QueuePosition,
			FinishDate: q.FinishDate, StartDate: q.StartDate, LevelStartSp: q.LevelStartSp,
			LevelEndSp: q.LevelEndSp, TrainingStartSp: q.TrainingStartSp,
		}
	}
	return out, nil
}

type ListCharacterSkillQueueInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterSkillQueueOutput struct {
	CharacterID int32             `json:"characterId"`
	Queue       []SkillQueueEntry `json:"queue"`
}

func listCharacterSkillQueueHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterSkillQueueInput, ListCharacterSkillQueueOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterSkillQueueInput) (*mcpsdk.CallToolResult, ListCharacterSkillQueueOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkillQueueOutput{}, err
		}
		if !found {
			return nil, ListCharacterSkillQueueOutput{}, fmt.Errorf("list character skill queue for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		queue, err := listCharacterSkillQueue(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterSkillQueueOutput{}, fmt.Errorf("list character skill queue: %w", err)
		}
		return nil, ListCharacterSkillQueueOutput{CharacterID: in.CharacterID, Queue: queue}, nil
	}
}

// --- list_character_skills (private) ---

type CharacterSkill struct {
	SkillID            int64 `json:"skillId"`
	ActiveSkillLevel   int64 `json:"activeSkillLevel"`
	TrainedSkillLevel  int64 `json:"trainedSkillLevel"`
	SkillpointsInSkill int64 `json:"skillpointsInSkill"`
}

var listCharacterSkills = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CharacterSkill, int64, *int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("build ESI client: %w", err)
	}
	skills, err := client.CharacterSkills(ctx)
	if err != nil {
		return nil, 0, nil, err
	}
	out := make([]CharacterSkill, len(skills.Skills))
	for i, s := range skills.Skills {
		out[i] = CharacterSkill{
			SkillID: s.SkillId, ActiveSkillLevel: s.ActiveSkillLevel,
			TrainedSkillLevel: s.TrainedSkillLevel, SkillpointsInSkill: s.SkillpointsInSkill,
		}
	}
	return out, skills.TotalSp, skills.UnallocatedSp, nil
}

type ListCharacterSkillsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterSkillsOutput struct {
	CharacterID   int32            `json:"characterId"`
	Skills        []CharacterSkill `json:"skills"`
	TotalSp       int64            `json:"totalSp"`
	UnallocatedSp *int64           `json:"unallocatedSp,omitempty"`
}

func listCharacterSkillsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterSkillsInput, ListCharacterSkillsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterSkillsInput) (*mcpsdk.CallToolResult, ListCharacterSkillsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkillsOutput{}, err
		}
		if !found {
			return nil, ListCharacterSkillsOutput{}, fmt.Errorf("list character skills for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		skills, totalSp, unallocatedSp, err := listCharacterSkills(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterSkillsOutput{}, fmt.Errorf("list character skills: %w", err)
		}
		return nil, ListCharacterSkillsOutput{CharacterID: in.CharacterID, Skills: skills, TotalSp: totalSp, UnallocatedSp: unallocatedSp}, nil
	}
}
```

Modify `mcp/server.go`: append after the actual last-registered tool block at execution time
(immediately before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_attributes",
		Description: "Get a logged-in EVE Online character's neural attributes (Charisma, Intelligence, Memory, Perception, Willpower) and remap history.",
	}, getCharacterAttributesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_skill_queue",
		Description: "List a logged-in EVE Online character's current skill training queue.",
	}, listCharacterSkillQueueHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_skills",
		Description: "List a logged-in EVE Online character's trained skills, total skill points, and unallocated skill points.",
	}, listCharacterSkillsHandler(version, clientID, store))
```

Update `mcp/server.go`'s package doc comment: append `get_character_attributes`,
`list_character_skill_queue`, and `list_character_skills` to the trailing tool list, preserving the
"and X registered" grammar.

Modify `mcp/server_test.go`: append `"get_character_attributes"`, `"list_character_skill_queue"`,
`"list_character_skills"` to the expected-tools slice, matching the file's existing
membership-check style.

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Skills row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: increment the tool count by 3 from whatever it currently reads (spell out the
new number in words, matching the file's existing style), and append the 3 new tool names to the
`mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestGetCharacterAttributesHandler|TestListCharacterSkillQueueHandler|TestListCharacterSkillsHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (6 new handler tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 9: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, no panics on tool registration.

```bash
git add esi/skills.go esi/skills_test.go mcp/skills.go mcp/skills_internal_test.go \
  mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add skills MCP tools (3 tools: character attributes, skill queue, skills)"
```
