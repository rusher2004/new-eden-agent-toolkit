package industry

import "math"

// ReprocessSkills are a character's reprocessing skill levels.
type ReprocessSkills struct {
	Reprocessing           int
	ReprocessingEfficiency int
	// OreSkills maps a processing skill ID (e.g. 60377 Simple Ore
	// Processing) to its level.
	OreSkills map[int64]int
	// Implant is the reprocessing implant bonus: 0.01 (RX-801), 0.02
	// (RX-802) or 0.04 (RX-804).
	Implant float64
}

// UpwellYieldInput describes an Upwell structure's reprocessing setup.
type UpwellYieldInput struct {
	RigTier           int     // 0 none, 1 T1, 2 T2 reprocessing rig
	Band              Band    // security band of the structure
	StructureBonusPct float64 // strRefiningYieldBonus (2722): 2 Athanor, 5.5 Tatara, 0 others
}

// UpwellYield is the Upwell reprocessing yield (0–1) for an ore whose
// processing skill is at oreSkillLevel. Source: EVE University, Reprocessing:
// (50 + Rm) × (1 + Sec) × (1 + Sm) × (1 + 0.03 R) × (1 + 0.02 Re) × (1 +
// 0.02 Op) × (1 + Im), where Sec applies only with a rig.
func UpwellYield(in UpwellYieldInput, s ReprocessSkills, oreSkillLevel int) float64 {
	sec := 0.0
	if in.RigTier > 0 {
		sec = ReprocessingSecurityTerm[in.Band]
	}
	y := (UpwellReprocessingBase + UpwellReprocessingRigTerm[in.RigTier]) * (1 + sec) * (1 + in.StructureBonusPct/100)
	return y * skillTerms(s, oreSkillLevel) / 100
}

// NPCYield is an NPC station's reprocessing yield (0–1) from its own
// efficiency (staStations.reprocessingEfficiency).
func NPCYield(stationEfficiency float64, s ReprocessSkills, oreSkillLevel int) float64 {
	return stationEfficiency * skillTerms(s, oreSkillLevel)
}

func skillTerms(s ReprocessSkills, op int) float64 {
	return (1 + 0.03*float64(s.Reprocessing)) * (1 + 0.02*float64(s.ReprocessingEfficiency)) * (1 + 0.02*float64(op)) * (1 + s.Implant)
}

// NPCReprocessingTax is the NPC station reprocessing tax for the higher of
// personal and corporation standing with the station owner. Source: EVE
// University, Reprocessing (5% at 0 standing, 0% at 6.67; linear between is
// an assumption).
func NPCReprocessingTax(standing float64) float64 {
	if standing >= NPCReprocessingTaxFreeStanding {
		return 0
	}
	return max(0, NPCReprocessingTaxAtZero*(1-max(standing, 0)/NPCReprocessingTaxFreeStanding))
}

// ScrapYield is scrapmetal reprocessing yield: base × (1 + 0.02 × Scrapmetal
// Processing); structures use a 50% base with no other bonus. Source: EVE
// University, Reprocessing.
func ScrapYield(base float64, scrapmetalLvl int) float64 {
	return base * (1 + 0.02*float64(scrapmetalLvl))
}

// Reprocess returns what qty units of an item yield. Only whole portions
// reprocess; output per material is rounded down.
func Reprocess(perPortion []TypeQty, portion, qty int64, yield float64) []TypeQty {
	if portion <= 0 {
		portion = 1
	}
	batches := qty / portion
	var out []TypeQty
	for _, m := range perPortion {
		if q := int64(math.Floor(float64(batches) * float64(m.Quantity) * yield)); q > 0 {
			out = append(out, TypeQty{m.TypeID, q})
		}
	}
	return out
}
