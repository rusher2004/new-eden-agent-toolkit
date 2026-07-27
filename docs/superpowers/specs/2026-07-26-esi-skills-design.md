# ESI Skills — Design Spec (twenty-ninth batch)

**Batch:** Skills tag, 3 endpoints, all private ("my own character"), no pagination.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

- `GetCharactersCharacterIdAttributes` → `*CharactersCharacterIdAttributesGet{Charisma, Intelligence,
  Memory, Perception, Willpower int64 (all required), AccruedRemapCooldownDate *time.Time,
  BonusRemaps *int64, LastRemapDate *time.Time (all optional)}`. Requires OAuth2 scope
  `esi-skills.read_skills.v1`.
- `GetCharactersCharacterIdSkillqueue` → `[]CharactersSkillqueueSkill{FinishedLevel, QueuePosition,
  SkillId int64 (required — note: also `SkillId`/`StartDate` on the actual per-inner-item type,
  verified from source, not the similarly-but-differently-named `CharactersCharacterIdSkillqueueGetInner`
  which is a distinct generated type used elsewhere), FinishDate, StartDate *time.Time (optional),
  LevelStartSp, LevelEndSp, TrainingStartSp *int64 (optional)}`. Requires OAuth2 scope
  `esi-skills.read_skillqueue.v1`. **Naming trap avoided**: the generated client has two
  similarly-named skillqueue-entry types (`CharactersSkillqueueSkill` and
  `CharactersCharacterIdSkillqueueGetInner`) — `GetCharactersCharacterIdSkillqueueExecute`'s actual
  return type is `[]CharactersSkillqueueSkill`, confirmed by reading the generated method signature
  directly rather than assuming from the endpoint's own name.
- `GetCharactersCharacterIdSkills` → `*CharactersSkills{Skills []CharactersSkillsSkill{ActiveSkillLevel,
  SkillId, SkillpointsInSkill, TrainedSkillLevel int64, all required}, TotalSp int64 (required),
  UnallocatedSp *int64 (optional)}`. Requires OAuth2 scope `esi-skills.read_skills.v1` (same scope
  as attributes).

All 3 not paginated — no `.Page()` setter on any of the 3 request builders. No recursion anywhere.

## `esi/skills.go`

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

`esi/skills_test.go`: 6 tests (success + HTTP error per method).

## `mcp/skills.go`

3 private tools: `get_character_attributes`, `list_character_skill_queue`,
`list_character_skills`. All follow `mcp/fittings.go`'s shape.

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

All new types are flat, non-recursive. Already sandbox-verified against a live `mcpsdk.AddTool`
registration test with zero schema-generator errors — including catching and fixing a real
naming-collision trap during sandbox verification (`CharacterSkillQueue`'s actual generated return
type is `CharactersSkillqueueSkill`, not the similarly-named `CharactersCharacterIdSkillqueueGetInner`).

`mcp/skills_internal_test.go`: 6 tests (`MapsFields` + `UnknownCharacter` pair per tool), following
`mcp/fittings_internal_test.go`'s swappable-var-override pattern.

## `mcp/server.go` / `mcp/server_test.go`

Register 3 tools after whatever the last-registered tool is at execution time (expected to be
`search_character` from the Search batch). Update package doc comment. `server_test.go`'s
expected-tools list: append 3 names.

## Docs sync

- `CLAUDE.md` / `README.md`: increment tool count by 3 from whatever it currently reads, append
  the 3 new tool names.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Skills row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/skills_test.go`,
swappable-`var`-override tests for `mcp/skills_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.

Given this batch's size (3 endpoints, all private, no auth complexity), Task 1 and Task 2 are
combined into a single implementation task, following the precedent set by Location/Loyalty
(2-task split) but scaled down given only 3 simple flat endpoints — actually combined into one
task here given the low complexity, mirroring Search/Routes' precedent.
