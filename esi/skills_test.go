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
