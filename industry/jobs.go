package industry

import (
	"math"
)

// MaterialQuantity is the quantity of one input a job of runs runs needs:
// max(runs, ceil(round(base × runs × (1 − ME/100) × facilityMult, 2))).
// Rounding is per job, not per run, and every run needs at least one unit.
// Source: EVE University, Manufacturing and Research (ME).
func MaterialQuantity(base, runs int64, me int, facilityMult float64) int64 {
	if base <= 0 || runs <= 0 {
		return 0
	}
	x := float64(base) * float64(runs) * (1 - float64(me)/100) * facilityMult
	x = math.Round(x*100) / 100
	q := int64(math.Ceil(x))
	return max(q, runs)
}

// ManufacturingSkillMultiplier is the time multiplier from Industry (−4% per
// level), Advanced Industry (−3%) and any science skills that give −1% per
// level for this product. Source: EVE University, Manufacturing.
func ManufacturingSkillMultiplier(industryLvl, advancedIndustryLvl int, scienceLvls ...int) float64 {
	m := (1 - 0.04*float64(industryLvl)) * (1 - 0.03*float64(advancedIndustryLvl))
	for _, s := range scienceLvls {
		m *= 1 - 0.01*float64(s)
	}
	return m
}

// ReactionSkillMultiplier is Reactions' −4% per level. Source: EVE
// University, Reactions.
func ReactionSkillMultiplier(reactionsLvl int) float64 {
	return 1 - 0.04*float64(reactionsLvl)
}

// ResearchSkillMultiplier is the time multiplier for ME research (pass
// Metallurgy), TE research (pass Research) or copying (pass Science): −5% per
// level, with Advanced Industry's −3%. Source: EVE University, Research.
func ResearchSkillMultiplier(skillLvl, advancedIndustryLvl int) float64 {
	return (1 - 0.05*float64(skillLvl)) * (1 - 0.03*float64(advancedIndustryLvl))
}

// JobSeconds is a manufacturing or reaction job's duration, rounded to the
// nearest second: base × runs × (1 − TE/100) × skills × facility.
func JobSeconds(baseSeconds float64, runs int64, te int, skillMult, facilityMult float64) float64 {
	return math.Round(baseSeconds * float64(runs) * (1 - float64(te)/100) * skillMult * facilityMult)
}

// EstimatedItemValue is Σ ME0 material quantity × ESI adjusted price × runs.
// Materials without a price contribute nothing.
func EstimatedItemValue(materials map[int64]int64, adjustedPrices map[int64]float64, runs int64) float64 {
	v := 0.0
	for t, q := range materials {
		v += float64(q) * adjustedPrices[t]
	}
	return v * float64(runs)
}

// FeeInput prices one job's installation fee.
type FeeInput struct {
	BaseValue          float64 // EIV for manufacturing/reactions; the activity's base value otherwise
	CostIndex          float64 // system cost index for the activity (ESI industry/systems)
	StructureCostBonus float64 // Facility.Multipliers(...).Cost; 1 for none
	FacilityTax        float64
	Alpha              bool
}

// JobFee is BaseValue × (costIndex × structureCostBonus + facilityTax + SCC +
// Alpha tax). Source: EVE University, Manufacturing (job installation cost).
func JobFee(in FeeInput) float64 {
	bonus := in.StructureCostBonus
	if bonus == 0 {
		bonus = 1
	}
	tax := min(in.FacilityTax, IndustryTaxCap)
	rate := in.CostIndex*bonus + tax + SCCSurcharge
	if in.Alpha {
		rate += AlphaCloneTax
	}
	return in.BaseValue * rate
}

// CalibrateFeeRate backs out the effective fee rate (fee ÷ base value) from a
// job the game actually charged, so later estimates can use it directly.
func CalibrateFeeRate(baseValue, actualFee float64) float64 {
	if baseValue == 0 {
		return 0
	}
	return actualFee / baseValue
}

// CopyBaseValue is the base value copy fees are charged on: copies × runs per
// copy × 0.02 × the product's EIV per run. Source: EVE University, Research.
func CopyBaseValue(productEIVPerRun float64, copies, runsPerCopy int64) float64 {
	return float64(copies) * float64(runsPerCopy) * CopyCostFactor * productEIVPerRun
}

// CopySeconds is copy duration from the SDE's copying activity time per run.
func CopySeconds(copySecondsPerRun float64, copies, runsPerCopy int64, skillMult, facilityMult float64) float64 {
	return math.Round(copySecondsPerRun * float64(copies) * float64(runsPerCopy) * skillMult * facilityMult)
}

// ResearchSeconds is the time to research a blueprint from level `from` to
// level `to` (ME or TE): Σ level base durations × rank × skills × facility.
// Source: EVE University, Research (level duration table).
func ResearchSeconds(rank float64, from, to int, skillMult, facilityMult float64) float64 {
	from, to = clampLevel(from), clampLevel(to)
	if to <= from {
		return 0
	}
	return math.Round((ResearchLevelBaseSeconds[to] - ResearchLevelBaseSeconds[from]) * rank * skillMult * facilityMult)
}

// ResearchBaseValue applies EVE University's research "process time value"
// formula literally: Σ (duration_n − duration_n−1) × 0.02105 × EIV. It is
// unverified against the game; tools label it as an estimate and recommend
// calibrating against a real research job.
func ResearchBaseValue(productEIVPerRun float64, from, to int) float64 {
	from, to = clampLevel(from), clampLevel(to)
	if to <= from {
		return 0
	}
	return (ResearchLevelBaseSeconds[to] - ResearchLevelBaseSeconds[from]) * ResearchCostFactor * productEIVPerRun
}

func clampLevel(l int) int { return min(max(l, 0), 10) }

// InventionInput describes one invention attempt.
type InventionInput struct {
	BaseChance         float64 // from the SDE blueprint probability, or InventionBaseChance
	Science1, Science2 int     // the two required science skill levels
	Encryption         int     // the racial encryption methods level
	Decryptor          *Decryptor
	BaseRuns           int // runs of the invented copy before decryptor (10, or 1 for ships and rigs)
}

// InventionResult is the chance and the copy a success produces.
type InventionResult struct {
	Chance float64
	Runs   int
	ME     int
	TE     int
}

// Invent computes invention chance and output. Source: EVE University,
// Invention: base × (1 + (sci1 + sci2)/30 + encryption/40) × (1 +
// decryptor/100); output ME 2 / TE 4 plus decryptor.
func Invent(in InventionInput) InventionResult {
	chance := in.BaseChance * (1 + float64(in.Science1+in.Science2)/30 + float64(in.Encryption)/40)
	r := InventionResult{Runs: in.BaseRuns, ME: InventionBaseME, TE: InventionBaseTE}
	if d := in.Decryptor; d != nil {
		chance *= 1 + d.Probability/100
		r.Runs += d.Runs
		r.ME += d.ME
		r.TE += d.TE
	}
	r.Chance = min(chance, 1)
	return r
}

// ExpectedCostPerSuccess divides the cost of one attempt by the chance of
// success.
func ExpectedCostPerSuccess(attemptCost, chance float64) float64 {
	if chance <= 0 {
		return math.Inf(1)
	}
	return attemptCost / chance
}
