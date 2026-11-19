package industry

import (
	"container/heap"
	"fmt"
	"sort"
)

// Builder is a character who can run jobs.
type Builder struct {
	CharacterID int64
	Name        string
	Slots       map[string]int // by activity kind: "manufacturing", "reaction", "science"
	// TimeMult maps an activity to this builder's skill time multiplier.
	TimeMult map[string]float64
	// Skills are the builder's trained levels, to check job requirements.
	Skills map[int64]int
}

// ScheduledJob is a job assigned to a builder.
type ScheduledJob struct {
	Job         TreeJob `json:"job"`
	BuilderID   int64   `json:"builderId"`
	BuilderName string  `json:"builderName"`
	StartHours  float64 `json:"startHours"`
	Hours       float64 `json:"hours"`
}

// Schedule is the result of assigning jobs.
type Schedule struct {
	Jobs        []ScheduledJob `json:"jobs"`
	Unassigned  []TreeJob      `json:"unassigned,omitempty"` // no builder has the skills
	Reasons     []string       `json:"reasons,omitempty"`
	MakespanHrs float64        `json:"makespanHours"`
	SlotDays    float64        `json:"slotDays"`
}

func slotKind(activity string) string {
	switch activity {
	case ActManufacturing:
		return ActManufacturing
	case ActReaction:
		return ActReaction
	default:
		return "science"
	}
}

// CanRunJob reports whether a builder has every skill a recipe needs and a
// slot for its activity.
func CanRunJob(b Builder, r Recipe) bool {
	return canBuild(b, TreeJob{Activity: r.Activity, Skills: r.Skills})
}

func canBuild(b Builder, j TreeJob) bool {
	for _, s := range j.Skills {
		if int64(b.Skills[s.TypeID]) < s.Quantity {
			return false
		}
	}
	return b.Slots[slotKind(j.Activity)] > 0
}

type slot struct {
	free    float64
	builder int
}
type slotHeap []slot

func (h slotHeap) Len() int           { return len(h) }
func (h slotHeap) Less(i, j int) bool { return h[i].free < h[j].free }
func (h slotHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *slotHeap) Push(x any)        { *h = append(*h, x.(slot)) }
func (h *slotHeap) Pop() any          { o := *h; s := o[len(o)-1]; *h = o[:len(o)-1]; return s }

// ScheduleJobs assigns jobs, longest first, to the eligible slot that frees up
// earliest, assuming a new job is installed as soon as a slot frees. Jobs
// whose inputs come from other jobs are not sequenced; the makespan is a
// lower bound for one level of the tree at a time.
func ScheduleJobs(jobs []TreeJob, builders []Builder) Schedule {
	var s Schedule
	order := make([]int, len(jobs))
	hours := make([]float64, len(jobs))
	for i := range jobs {
		order[i] = i
	}
	maxMult := func(j TreeJob) float64 {
		m := 0.0
		for _, b := range builders {
			if canBuild(b, j) {
				if v := b.TimeMult[j.Activity]; v > m {
					m = v
				}
			}
		}
		return m
	}
	for i, j := range jobs {
		hours[i] = j.Seconds * maxMult(j) / 3600
	}
	sort.SliceStable(order, func(a, b int) bool { return hours[order[a]] > hours[order[b]] })

	pools := map[string]*slotHeap{}
	for bi, b := range builders {
		for kind, n := range b.Slots {
			if pools[kind] == nil {
				pools[kind] = &slotHeap{}
			}
			for range n {
				heap.Push(pools[kind], slot{0, bi})
			}
		}
	}
	for _, i := range order {
		j := jobs[i]
		pool := pools[slotKind(j.Activity)]
		var skipped []slot
		assigned := false
		for pool != nil && pool.Len() > 0 {
			sl := heap.Pop(pool).(slot)
			b := builders[sl.builder]
			if !canBuild(b, j) {
				skipped = append(skipped, sl)
				continue
			}
			h := j.Seconds * b.TimeMult[j.Activity] / 3600
			s.Jobs = append(s.Jobs, ScheduledJob{Job: j, BuilderID: b.CharacterID, BuilderName: b.Name, StartHours: sl.free, Hours: h})
			s.SlotDays += h / 24
			s.MakespanHrs = max(s.MakespanHrs, sl.free+h)
			heap.Push(pool, slot{sl.free + h, sl.builder})
			assigned = true
			break
		}
		for _, sl := range skipped {
			heap.Push(pool, sl)
		}
		if !assigned {
			s.Unassigned = append(s.Unassigned, j)
			s.Reasons = append(s.Reasons, fmt.Sprintf("no builder has the skills and a free %s slot for type %d", slotKind(j.Activity), j.ProductTypeID))
		}
	}
	return s
}
