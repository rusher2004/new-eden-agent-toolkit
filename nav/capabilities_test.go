package nav_test

import (
	"testing"
	"time"

	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

func TestAbilities(t *testing.T) {
	const cft, recon, blops, hauler = 21603, 22761, 28656, 3340
	hulls := []nav.Hull{
		{Name: "Falcon", Requires: []nav.SkillReq{{recon, 1}}},
		{Name: "Sin", Requires: []nav.SkillReq{{blops, 1}}},
	}
	abilities := []nav.Ability{
		{Kind: "cyno", Variant: "standard", Module: "Cynosural Field Generator I", Requires: []nav.SkillReq{{cft, 1}}, Hulls: hulls},
		{Kind: "cyno", Variant: "covert", Module: "Covert Cynosural Field Generator I", Requires: []nav.SkillReq{{cft, 5}}, Hulls: hulls},
		{Kind: "cyno", Variant: "industrial", Module: "Industrial Cynosural Field Generator", Requires: []nav.SkillReq{{cft, 1}}, Hulls: []nav.Hull{{Name: "Badger", Requires: []nav.SkillReq{{hauler, 1}}}}},
	}
	got := nav.Abilities(map[int64]int{cft: 4, recon: 4, blops: 0}, abilities)
	if !got[0].Can || len(got[0].Hulls) != 1 || got[0].Hulls[0] != "Falcon" {
		t.Errorf("standard = %+v", got[0])
	}
	if got[1].Can || len(got[1].Missing) != 1 || got[1].Missing[0] != (nav.SkillReq{cft, 5}) {
		t.Errorf("covert = %+v, want missing CFT V", got[1])
	}
	if got[2].Can || len(got[2].Hulls) != 0 || len(got[2].Missing) != 0 {
		t.Errorf("industrial = %+v, want no hull it can fly", got[2])
	}
}

func TestCloneJumps(t *testing.T) {
	if got := nav.CloneJumpCooldown(5); got != 19*time.Hour {
		t.Errorf("cooldown at V = %v, want 19h", got)
	}
	if got := nav.CloneJumpCooldown(0); got != 24*time.Hour {
		t.Errorf("cooldown untrained = %v, want 24h", got)
	}
	last := time.Date(2026, 11, 19, 20, 0, 0, 0, time.UTC)
	if got := nav.NextCloneJump(last, 4); !got.Equal(time.Date(2026, 11, 20, 16, 0, 0, 0, time.UTC)) {
		t.Errorf("next jump = %v", got)
	}
	if got := nav.NextCloneJump(time.Time{}, 5); !got.IsZero() {
		t.Errorf("never jumped = %v, want zero", got)
	}
	if got := nav.JumpCloneCapacity(5, 5, 3); got != 13 {
		t.Errorf("capacity = %d, want 13", got)
	}
}
