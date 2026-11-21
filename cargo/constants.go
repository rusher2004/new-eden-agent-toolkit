package cargo

// Constants not available in the SDE. Each cites its source.

const (
	// ShipCategoryID is the SDE category of ships.
	ShipCategoryID int64 = 6

	// StrategicCruiserGroupID is the SDE group of strategic cruisers, which
	// keep their rigs when repackaged; every other ship's rigs are destroyed.
	// Source: EVE University, Strategic cruisers ("Unlike other ships,
	// Strategic Cruisers can be repackaged without destroying their rigs").
	StrategicCruiserGroupID int64 = 963
)

// officeFlag is the location flag ESI gives a corporation's office in a
// station or structure. Corporation hangar items sit inside the office.
const officeFlag = "OfficeFolder"

// fittingFlagPrefixes are the location flags of items fitted to a ship,
// rather than carried in one of its holds.
var fittingFlagPrefixes = []string{"HiSlot", "MedSlot", "LoSlot", "RigSlot", "SubSystemSlot"}

// structureFittingFlagPrefixes are the location flags of items fitted to an
// Upwell structure itself (its modules, service modules, quantum core and
// deed), which ESI lists with the structure as their location. They aren't
// hangar contents, so they're no part of moving out of it.
var structureFittingFlagPrefixes = []string{"HiSlot", "MedSlot", "LoSlot", "RigSlot", "ServiceSlot", "QuantumCoreRoom", "StructureDeedFlag"}
