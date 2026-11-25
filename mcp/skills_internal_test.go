package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
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

	unallocatedSp := int64(1000)
	fake := fakeCharacterSkillsESI{characterSkills: func(context.Context) (*esiapi.CharactersSkills, error) {
		return &esiapi.CharactersSkills{Skills: []esiapi.CharactersSkillsSkill{{
			SkillId: 3300, ActiveSkillLevel: 5, TrainedSkillLevel: 5, SkillpointsInSkill: 256000,
		}}, TotalSp: 5000000, UnallocatedSp: &unallocatedSp}, nil
	}}

	handler := listCharacterSkillsHandler(openNarrow[characterSkillsESI](fake), store)
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

	open := func(context.Context, int32) (characterSkillsESI, error) {
		t.Fatal("no ESI client should be opened for an unknown character")
		return nil, nil
	}

	handler := listCharacterSkillsHandler(open, store)
	_, _, err := handler(context.Background(), nil, ListCharacterSkillsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterAttributesHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterAttributes
	t.Cleanup(func() { getCharacterAttributes = orig })
	wantErr := errors.New("upstream failure")
	getCharacterAttributes = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, int64, int64, int64, int64, *time.Time, *int64, *time.Time, error) {
		return 0, 0, 0, 0, 0, nil, nil, nil, wantErr
	}

	handler := getCharacterAttributesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterAttributesInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterSkillQueueHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterSkillQueue
	t.Cleanup(func() { listCharacterSkillQueue = orig })
	wantErr := errors.New("upstream failure")
	listCharacterSkillQueue = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]SkillQueueEntry, error) {
		return nil, wantErr
	}

	handler := listCharacterSkillQueueHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterSkillQueueInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterSkillsHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterSkillsESI{characterSkills: func(context.Context) (*esiapi.CharactersSkills, error) { return nil, wantErr }}

	handler := listCharacterSkillsHandler(openNarrow[characterSkillsESI](fake), store)
	_, _, err := handler(context.Background(), nil, ListCharacterSkillsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterSkillsHandler_PropagatesOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("no token")
	open := func(context.Context, int32) (characterSkillsESI, error) { return nil, wantErr }
	handler := listCharacterSkillsHandler(open, store)
	_, _, err := handler(context.Background(), nil, ListCharacterSkillsInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// fakeCharacterSkillsESI fakes characterSkillsESI; a nil characterSkills fails as an unfaked call.
type fakeCharacterSkillsESI struct {
	characterSkills func(ctx context.Context) (*esiapi.CharactersSkills, error)
}

func (f fakeCharacterSkillsESI) CharacterSkills(ctx context.Context) (*esiapi.CharactersSkills, error) {
	if f.characterSkills == nil {
		return nil, errNotFaked
	}
	return f.characterSkills(ctx)
}
