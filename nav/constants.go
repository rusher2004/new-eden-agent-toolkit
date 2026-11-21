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

// securityColors are the in-game colors for each displayed security value
// from 0.1 to 1.0 (index 1 to 10), and securityColorNull is the color for
// 0.0 and below. Source: FC developer guide, System security
// (https://developers.eveonline.com/docs/guides/system-security/).
var securityColors = [11]string{
	"", "#731F1F", "#BB1116", "#CE440F", "#DC6C06", "#F5FF83", "#71E754", "#60DBA3", "#4ECEF8", "#399AEB", "#2C75E1",
}

const securityColorNull = "#8D3163"

// Clone jump cooldown: 24 hours, less one hour per level of Infomorph
// Synchronizing. Source: EVE University, Jump clones; the skill's own
// description gives the hour per level ("Reduced time between clone jumps
// by 1 hour per level").
const (
	cloneJumpBaseHours     = 24
	cloneJumpHoursPerLevel = 1
)

// Skills behind clone jumps, cynos and conduits. Type IDs checked against
// the SDE 2026-11-20.
const (
	SkillInfomorphSynchronizing      int64 = 33399
	SkillInfomorphPsychology         int64 = 24242
	SkillAdvancedInfomorphPsychology int64 = 33407
	SkillEliteInfomorphPsychology    int64 = 73910
	// SkillCapitalJumpPortalGeneration lets a carrier, supercarrier or
	// command carrier make a conduit jump. The SDE ties it to those hulls
	// only through their jumpPortalPassengerBonusModAddSkill effect, not as
	// a required skill. Source: the skill's description ("the number of ships
	// that can be moved with a capital ship conduit jump").
	SkillCapitalJumpPortalGeneration int64 = 83094
)

// CapitalConduitEffect is the dogma effect that marks hulls with a built-in
// conduit jump.
const CapitalConduitEffect = "jumpPortalPassengerBonusModAddSkill"

// Generators behind cynos, bridges and conduits. The covert and industrial
// jump portal generators do both bridges and conduit jumps; the titan's
// does bridges only. Source: EVE University, Jump portal generators and
// Conduit jumps. Type IDs checked against the SDE 2026-11-20.
const (
	StandardCynoGenerator   int64 = 21096 // Cynosural Field Generator I
	IndustrialCynoGenerator int64 = 52694 // Industrial Cynosural Field Generator
	CovertCynoGenerator     int64 = 28646 // Covert Cynosural Field Generator I
	CovertPortalGenerator   int64 = 28652 // Covert Jump Portal Generator I
	IndustrialPortalGen     int64 = 63140 // Industrial Jump Portal Generator I
	TitanPortalGenerator    int64 = 23953 // Jump Portal Generator I
)

// allUpwell is every Upwell structure base type a ship can dock in.
var allUpwell = []string{"Astrahus", "Fortizar", "Keepstar", "Raitaru", "Azbel", "Sotiyo", "Athanor", "Tatara"}

// capitalDocking is where each capital class can dock, by ship group ID.
// Every other ship can dock anywhere. Source: EVE University, Capital
// ships (Docking table). Black Ops and command carriers aren't in that
// table: Black Ops are taken to dock anywhere a battleship can, and command
// carriers like carriers, and both are marked assumed.
var capitalDocking = map[int64]Docking{
	30:   {Upwell: []string{"Keepstar"}, BuildOnly: []string{"Sotiyo"}},                                                        // Titan
	659:  {Upwell: []string{"Keepstar"}, BuildOnly: []string{"Sotiyo"}},                                                        // Supercarrier
	485:  {NPCStations: true, Upwell: []string{"Fortizar", "Keepstar", "Sotiyo"}, BuildOnly: []string{"Azbel"}},                // Dreadnought
	4594: {NPCStations: true, Upwell: []string{"Fortizar", "Keepstar", "Sotiyo"}, BuildOnly: []string{"Azbel"}},                // Lancer Dreadnought
	547:  {NPCStations: true, Upwell: []string{"Fortizar", "Keepstar", "Sotiyo"}, BuildOnly: []string{"Azbel"}},                // Carrier
	1538: {NPCStations: true, Upwell: []string{"Fortizar", "Keepstar", "Sotiyo"}, BuildOnly: []string{"Azbel"}},                // Force Auxiliary
	883:  {NPCStations: true, Upwell: []string{"Fortizar", "Keepstar", "Sotiyo", "Tatara"}, BuildOnly: []string{"Azbel"}},      // Capital Industrial Ship
	5120: {NPCStations: true, Upwell: []string{"Fortizar", "Keepstar", "Sotiyo"}, BuildOnly: []string{"Azbel"}, Assumed: true}, // Command Carrier
	898:  {NPCStations: true, Upwell: allUpwell, Assumed: true},                                                                // Black Ops
}

// Which jump drives can use industrial and covert cynos, by ship group;
// every jump drive can use a standard cyno. Source: the cyno generators'
// own SDE descriptions ("Industrial Cynosural Fields are specifically
// attuned to support the jump drives of Jump Freighters and the Rorqual";
// a covert field "only Black Ops jump drives can lock on to"), and EVE
// University, Cynosural field, which adds Black Ops for industrial cynos.
var (
	industrialCynoGroups = []int64{902, 883, 898} // Jump Freighter, Capital Industrial Ship, Black Ops
	covertCynoGroups     = []int64{898}           // Black Ops
)
