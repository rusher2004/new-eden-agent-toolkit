package nav

// Constants not available in the SDE. Each cites its source. If one of these
// is later found in the SDE, read it from there instead.

const (
	// MetersPerLightYear converts SDE coordinates (meters) to light years.
	// EVE uses a rounded 9.46e15 m, not the astronomical 9.4607e15. Verified
	// 2026-11-18: with 9.46e15 and per-jump fuel rounded to the nearest unit,
	// dotlan's Nomad legs match exactly at 5/4/4 (e.g. Jita → Aunenen, 28,420
	// isotopes, and Jita → Jan → NL6V-7 → PF-QHK, 15,553, 25,148 and 28,818);
	// 9.4607e15 misses legs by 1–2.
	MetersPerLightYear = 9.46e15

	// Jump fatigue limits, in minutes: the fatigue cap, the jump activation
	// cooldown cap, and the minimum fatigue a jump multiplies. Source: EVE
	// University, Jump fatigue (rules from the March 2018 changes).
	fatigueCapMinutes  = 300
	cooldownCapMinutes = 30
	fatigueMinBase     = 10
)
