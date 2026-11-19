package industry_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
)

func capBuilder(id int64, name string, csc int) industry.Builder {
	return industry.Builder{CharacterID: id, Name: name, Slots: map[string]int{industry.ActManufacturing: 10},
		TimeMult: map[string]float64{industry.ActManufacturing: industry.ManufacturingSkillMultiplier(5, 5)},
		Skills:   map[int64]int{22242: csc}}
}

func TestScheduleJobs_FortySlots(t *testing.T) {
	// 80 identical 5-run component jobs (5 × 16000 s × TE20 × facility) on
	// 40 slots: two waves.
	job := industry.TreeJob{ProductTypeID: 21039, Activity: industry.ActManufacturing, Runs: 5, Seconds: 16000 * 5 * 0.8 * 0.7 * 0.58, Skills: []industry.TypeQty{{22242, 1}}}
	var jobs []industry.TreeJob
	for range 80 {
		jobs = append(jobs, job)
	}
	builders := []industry.Builder{capBuilder(1, "Builder A", 4), capBuilder(2, "Builder B", 4), capBuilder(3, "Builder C", 4), capBuilder(4, "Builder D", 3)}
	s := industry.ScheduleJobs(jobs, builders)
	perJob := job.Seconds * industry.ManufacturingSkillMultiplier(5, 5) / 3600
	if len(s.Jobs) != 80 || math.Abs(s.MakespanHrs-2*perJob) > 1e-9 {
		t.Errorf("makespan = %v h, want %v", s.MakespanHrs, 2*perJob)
	}
}

func TestScheduleJobs_SkillGate(t *testing.T) {
	hullJob := industry.TreeJob{ProductTypeID: 37605, Activity: industry.ActManufacturing, Runs: 1, Seconds: 1000, Skills: []industry.TypeQty{{22242, 3}}}
	s := industry.ScheduleJobs([]industry.TreeJob{hullJob}, []industry.Builder{capBuilder(9, "Untrained Builder", 0)})
	if len(s.Unassigned) != 1 || len(s.Reasons) != 1 {
		t.Errorf("schedule = %+v", s)
	}
}
