package industry_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
)

const sotiyoNullMat = 0.99 * 0.958 // Sotiyo role bonus × T1 XL rig in nullsec

func TestMaterialQuantity(t *testing.T) {
	tests := []struct {
		name       string
		base, runs int64
		me         int
		mult       float64
		want       int64
	}{
		{"ME9 Minokawa conduits", 32, 1, 9, sotiyoNullMat, 28},
		{"ME9 Minokawa shield emitters", 20, 1, 9, sotiyoNullMat, 18},
		{"ME9 Minokawa capacitor batteries", 15, 1, 9, sotiyoNullMat, 13},
		{"ME6 Minokawa capacitor batteries", 15, 1, 6, sotiyoNullMat, 14},
		{"ME6 Minokawa conduits", 32, 1, 6, sotiyoNullMat, 29},
		{"ME9 Minokawa auto-integrity seals", 400, 1, 9, sotiyoNullMat, 346},
		{"one per run floor", 1, 100, 10, 0.9, 100},
		{"5 runs of siege array pyerite", 101250, 5, 10, sotiyoNullMat, 432124},
		{"zero base", 0, 5, 10, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := industry.MaterialQuantity(tt.base, tt.runs, tt.me, tt.mult); got != tt.want {
				t.Errorf("MaterialQuantity = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestJobSeconds_MatchesRealJobs(t *testing.T) {
	skills := industry.ManufacturingSkillMultiplier(5, 5)
	facility := 0.7 * 0.58
	// Capital Siege Array, TE20, 4 runs: 14,135 s, matching an in-game job.
	if got := industry.JobSeconds(16000, 4, 20, skills, facility); got != 14135 {
		t.Errorf("siege array 4 runs = %v, want 14135", got)
	}
	// Minokawa hull, TE14: 474,858 s, matching an in-game job.
	if got := industry.JobSeconds(2000000, 1, 14, skills, facility); got != 474858 {
		t.Errorf("minokawa = %v, want 474858", got)
	}
}

func TestJobFee_CalibratedRate(t *testing.T) {
	rate := industry.CalibrateFeeRate(2692208286, 179799038)
	if math.Abs(rate-0.06678) > 0.0001 {
		t.Errorf("rate = %v, want about 0.0668", rate)
	}
	fee := industry.JobFee(industry.FeeInput{BaseValue: 1e9, CostIndex: 0.03, StructureCostBonus: 0.95, FacilityTax: 0.01})
	if want := 1e9 * (0.03*0.95 + 0.01 + 0.04); math.Abs(fee-want) > 1e-3 {
		t.Errorf("fee = %v, want %v", fee, want)
	}
	capped := industry.JobFee(industry.FeeInput{BaseValue: 100, FacilityTax: 0.5, Alpha: true})
	if want := 100 * (0.10 + 0.04 + 0.0025); math.Abs(capped-want) > 1e-9 {
		t.Errorf("capped fee = %v, want %v", capped, want)
	}
}

func TestEstimatedItemValue(t *testing.T) {
	got := industry.EstimatedItemValue(map[int64]int64{34: 100, 35: 10}, map[int64]float64{34: 4, 35: 16}, 3)
	if got != (100*4+10*16)*3 {
		t.Errorf("EIV = %v", got)
	}
}

func TestResearchAndCopy(t *testing.T) {
	if got := industry.ResearchSeconds(1, 0, 10, 1, 1); got != 256000 {
		t.Errorf("ME0→10 rank 1 = %v", got)
	}
	if got := industry.ResearchSeconds(2, 4, 5, industry.ResearchSkillMultiplier(5, 5), 0.7); got != math.Round((3360-1414)*2*0.75*0.85*0.7) {
		t.Errorf("ME4→5 = %v", got)
	}
	if got := industry.CopyBaseValue(1e6, 2, 10); got != 2*10*0.02*1e6 {
		t.Errorf("copy base = %v", got)
	}
}

func TestInvent(t *testing.T) {
	// Cruiser base 26%, sciences 4 and 4, encryption 4, no decryptor:
	// 0.26 × (1 + 8/30 + 4/40) = 0.35533.
	r := industry.Invent(industry.InventionInput{BaseChance: 0.26, Science1: 4, Science2: 4, Encryption: 4, BaseRuns: 1})
	if math.Abs(r.Chance-0.355333) > 1e-5 || r.ME != 2 || r.TE != 4 || r.Runs != 1 {
		t.Errorf("invent = %+v", r)
	}
	att := industry.Decryptors[1] // Attainment: +80%, +4 runs, -1 ME, +4 TE
	r = industry.Invent(industry.InventionInput{BaseChance: 0.34, Science1: 5, Science2: 5, Encryption: 5, Decryptor: &att, BaseRuns: 10})
	if math.Abs(r.Chance-0.8925) > 1e-9 || r.Runs != 14 || r.ME != 1 || r.TE != 8 {
		t.Errorf("invent with attainment = %+v , want chance 0.8925", r)
	}
	if got := industry.ExpectedCostPerSuccess(1000, 0.25); got != 4000 {
		t.Errorf("expected cost = %v", got)
	}
}
