package nav

import (
	"slices"
	"time"
)

// SkillReq is one skill a type requires, at a level.
type SkillReq struct {
	SkillID int64 `json:"skillId"`
	Level   int   `json:"level"`
}

// Trained reports whether skills (skill ID to active level) meet every
// requirement.
func Trained(skills map[int64]int, reqs []SkillReq) bool {
	for _, r := range reqs {
		if skills[r.SkillID] < r.Level {
			return false
		}
	}
	return true
}

// Missing lists the requirements skills don't meet.
func Missing(skills map[int64]int, reqs []SkillReq) []SkillReq {
	var out []SkillReq
	for _, r := range reqs {
		if skills[r.SkillID] < r.Level {
			out = append(out, r)
		}
	}
	return out
}

// Hull is a ship and the skills it takes to fly it, from the SDE.
type Hull struct {
	TypeID   int64
	Name     string
	GroupID  int64
	Requires []SkillReq
}

// Ability is something a character can do with a module (or a skill) on a
// hull: light a cyno, open a bridge, make a conduit jump. Requires is what
// the module takes; Hulls are the ships it can be used on.
type Ability struct {
	Kind     string // cyno, bridge or conduit
	Variant  string // standard, industrial, covert, titan or carrier
	Module   string // what it's done with, for showing the user
	Requires []SkillReq
	Hulls    []Hull
}

// AbilityResult is whether a character has an ability, and with which hulls.
type AbilityResult struct {
	Kind    string     `json:"kind"`
	Variant string     `json:"variant"`
	Module  string     `json:"module"`
	Can     bool       `json:"can" jsonschema:"trained for the module and for at least one hull it fits"`
	Hulls   []string   `json:"hulls,omitempty" jsonschema:"hulls the character can fly that this works on"`
	Missing []SkillReq `json:"missingModuleSkills,omitempty" jsonschema:"skills still needed for the module itself"`
}

// Abilities checks each ability against a character's skills: they have it
// when they're trained for the module and can fly at least one hull it
// works on.
func Abilities(skills map[int64]int, abilities []Ability) []AbilityResult {
	out := make([]AbilityResult, 0, len(abilities))
	for _, a := range abilities {
		r := AbilityResult{Kind: a.Kind, Variant: a.Variant, Module: a.Module, Missing: Missing(skills, a.Requires)}
		for _, h := range FlyableHulls(skills, a.Hulls) {
			r.Hulls = append(r.Hulls, h.Name)
		}
		r.Can = len(r.Missing) == 0 && len(r.Hulls) > 0
		out = append(out, r)
	}
	return out
}

// FlyableHulls lists the hulls a character is trained to fly, by name.
func FlyableHulls(skills map[int64]int, hulls []Hull) []Hull {
	var out []Hull
	for _, h := range hulls {
		if Trained(skills, h.Requires) {
			out = append(out, h)
		}
	}
	slices.SortFunc(out, func(a, b Hull) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return out
}

// CloneJumpCooldown is the wait between clone jumps for a level of
// Infomorph Synchronizing.
func CloneJumpCooldown(infomorphSynchronizing int) time.Duration {
	h := cloneJumpBaseHours - cloneJumpHoursPerLevel*min(max(infomorphSynchronizing, 0), 5)
	return time.Duration(h) * time.Hour
}

// NextCloneJump is when a character can next clone jump: the last jump plus
// their cooldown, or the zero time if they've never jumped.
func NextCloneJump(last time.Time, infomorphSynchronizing int) time.Time {
	if last.IsZero() {
		return time.Time{}
	}
	return last.Add(CloneJumpCooldown(infomorphSynchronizing))
}

// JumpCloneCapacity is how many jump clones a character can hold: one per
// level of Infomorph Psychology, Advanced Infomorph Psychology and Elite
// Infomorph Psychology.
func JumpCloneCapacity(psychology, advanced, elite int) int {
	return psychology + advanced + elite
}
