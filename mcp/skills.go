package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
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

func getCharacterAttributesHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[GetCharacterAttributesInput, GetCharacterAttributesOutput] {
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

func listCharacterSkillQueueHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCharacterSkillQueueInput, ListCharacterSkillQueueOutput] {
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

// characterSkillsESI is the ESI call list_character_skills makes; the
// planner family reads skills through it too.
type characterSkillsESI interface {
	CharacterSkills(ctx context.Context) (*esiapi.CharactersSkills, error)
}

func listCharacterSkills(ctx context.Context, client characterSkillsESI) ([]CharacterSkill, int64, *int64, error) {
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

// AddListCharacterSkills registers list_character_skills. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterSkills[C characterSkillsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_skills",
		Description: "List a logged-in EVE Online character's trained skills, total skill points, and unallocated skill points.",
	}, listCharacterSkillsHandler(open, store))
}

func listCharacterSkillsHandler[C characterSkillsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterSkillsInput, ListCharacterSkillsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterSkillsInput) (*mcpsdk.CallToolResult, ListCharacterSkillsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkillsOutput{}, err
		}
		if !found {
			return nil, ListCharacterSkillsOutput{}, fmt.Errorf("list character skills for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkillsOutput{}, fmt.Errorf("list character skills: %w", err)
		}
		skills, totalSp, unallocatedSp, err := listCharacterSkills(ctx, client)
		if err != nil {
			return nil, ListCharacterSkillsOutput{}, fmt.Errorf("list character skills: %w", err)
		}
		return nil, ListCharacterSkillsOutput{CharacterID: in.CharacterID, Skills: skills, TotalSp: totalSp, UnallocatedSp: unallocatedSp}, nil
	}
}
