package industry

import "github.com/rusher2004/new-eden-agent-toolkit/nav"

// Constants not available in the SDE. Each cites its source. If one of these
// is later found in the SDE, read it from there instead.

const (
	// SCCSurcharge is added to every industry job fee as a fraction of the
	// estimated item value. Source: EVE University, Manufacturing (job
	// installation cost).
	SCCSurcharge = 0.04

	// AlphaCloneTax is added for Alpha clones. Source: EVE University,
	// Manufacturing.
	AlphaCloneTax = 0.0025

	// IndustryTaxCap caps the facility tax an owner can charge. Source: EVE
	// University, Manufacturing ("Industry service taxes are capped at 10%").
	IndustryTaxCap = 0.10

	// NPCFacilityTax is the facility tax at NPC stations. Source: EVE
	// University, Manufacturing.
	NPCFacilityTax = 0.0025

	// UpwellReprocessingBase is the Upwell reprocessing base yield in percent
	// before rig, security, structure, skill and implant terms. Source: EVE
	// University, Reprocessing (Upwell reprocessing formula).
	UpwellReprocessingBase = 50.0

	// CopyCostFactor and ResearchCostFactor scale estimated item value in the
	// copying and research job cost formulas. Source: EVE University,
	// Research (job cost formulas).
	CopyCostFactor     = 0.02
	ResearchCostFactor = 0.02105

	// CopyTimeFactor is the base copy time as a fraction of build time per
	// run. Source: EVE University, Research ("80% of the build time").
	CopyTimeFactor = 0.8

	// NPCReprocessingTaxAtZero is the NPC station reprocessing tax at 0
	// standing, falling linearly to zero at NPCReprocessingTaxFreeStanding.
	// Source: EVE University, Reprocessing (NPC stations: "The tax starts at
	// 5% with 0 standings and falls to 0% at 6.67 standings"). The linear
	// shape between the endpoints is an assumption; the page gives only the
	// endpoints.
	NPCReprocessingTaxAtZero       = 0.05
	NPCReprocessingTaxFreeStanding = 6.67
)

// UpwellReprocessingRigTerm is Rm in the Upwell reprocessing formula: added
// to the 50% base. Source: EVE University, Reprocessing.
var UpwellReprocessingRigTerm = map[int]float64{0: 0, 1: 1, 2: 3}

// ReprocessingSecurityTerm is Sec in the Upwell reprocessing formula. It
// applies only when a reprocessing rig is fitted. Source: EVE University,
// Reprocessing.
var ReprocessingSecurityTerm = map[nav.Band]float64{nav.Highsec: 0, nav.Lowsec: 0.06, nav.Nullsec: 0.12, nav.Wormhole: 0.12}

// ResearchLevelBaseSeconds is the base duration for a rank 1 blueprint to
// reach each ME/TE research level (index 1..10). Source: EVE University,
// Research (level duration table).
var ResearchLevelBaseSeconds = [11]float64{0, 105, 250, 595, 1414, 3360, 8000, 19000, 45255, 107700, 256000}

// InventionBaseChance by product kind. Source: EVE University, Invention.
var InventionBaseChance = map[string]float64{
	"module":         0.34, // modules, rigs, ammo
	"frigate":        0.30, // frigates and destroyers
	"cruiser":        0.26, // cruisers, battlecruisers, mining barges, haulers, intact relics
	"battleship":     0.22,
	"malfunctioning": 0.21,
	"freighter":      0.18,
	"wrecked":        0.14,
}

// Decryptor modifies invention probability (percent), runs, ME and TE.
type Decryptor struct {
	Name        string
	Probability float64 // percent
	Runs        int
	ME          int
	TE          int
}

// Decryptors lists the T2 invention decryptors. Source: EVE University,
// Invention (decryptor table).
var Decryptors = []Decryptor{
	{"Accelerant Decryptor", 20, 1, 2, 10},
	{"Attainment Decryptor", 80, 4, -1, 4},
	{"Augmentation Decryptor", -40, 9, -2, 2},
	{"Optimized Attainment Decryptor", 90, 2, 1, -2},
	{"Optimized Augmentation Decryptor", -10, 7, 2, 0},
	{"Parity Decryptor", 50, 3, 1, -2},
	{"Process Decryptor", 10, 0, 3, 6},
	{"Symmetry Decryptor", 0, 2, 1, 8},
}

// Invented copies start at ME 2 and TE 4 before decryptor modifiers.
// Source: EVE University, Invention.
const (
	InventionBaseME = 2
	InventionBaseTE = 4
)
