package cargo

// HoldAttributes are the SDE dogma attributes for a hull's special holds,
// by the name tools report them under. The cargo hold itself is the type's
// capacity. Attribute IDs checked against the SDE 2026-11-20.
var HoldAttributes = []struct {
	AttributeID int64
	Hold        string
}{
	{908, "ship_maintenance_bay"},
	{912, "fleet_hangar"},
	{1549, "fuel_bay"},
	{1556, "mining_hold"},
	{1557, "gas_hold"},
	{1558, "mineral_hold"},
	{3136, "ice_hold"},
	{1573, "ammo_hold"},
	{1646, "command_center_hold"},
	{1653, "planetary_commodities_hold"},
	{2675, "subsystem_hold"},
	{5325, "mobile_depot_hold"},
	{5646, "colony_resources_hold"},
	{5944, "expedition_hold"},
	{1804, "quafe_hold"},
	{2467, "corpse_hold"},
	{2657, "booster_hold"},
	{3020, "frigate_escape_bay"},
	{283, "drone_bay"},
	{2055, "fighter_hangar"},
}

// Holds lists a hull's holds by base capacity in m³: the cargo hold, then
// every special hold it has.
func Holds(cargoM3 float64, attrs map[int64]float64) map[string]float64 {
	out := map[string]float64{}
	if cargoM3 > 0 {
		out["cargo"] = cargoM3
	}
	for _, h := range HoldAttributes {
		if v := attrs[h.AttributeID]; v > 0 {
			out[h.Hold] = v
		}
	}
	return out
}
