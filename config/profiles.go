package config

import (
	"fmt"
	"slices"
	"strings"
)

// FacilityProfile records what ESI can't tell neat about an industry or
// reprocessing facility: its rigs, fitted services and owner-set taxes.
// Bonuses themselves are read from the SDE at use time.
type FacilityProfile struct {
	Name            string   `json:"name" yaml:"name"`
	Kind            string   `json:"kind" yaml:"kind"` // npc_station, upwell or starbase
	StationID       int64    `json:"stationId,omitempty" yaml:"station_id,omitempty"`
	StructureType   int64    `json:"structureType,omitempty" yaml:"structure_type,omitempty"`
	SystemID        int64    `json:"systemId" yaml:"system_id"`
	SystemName      string   `json:"systemName,omitempty" yaml:"system_name,omitempty"`
	RigTypeIDs      []int64  `json:"rigTypeIds,omitempty" yaml:"rig_type_ids,omitempty"`
	Services        []string `json:"services,omitempty" yaml:"services,omitempty"`
	SupercapUpgrade bool     `json:"supercapUpgrade,omitempty" yaml:"supercap_upgrade,omitempty"`
	FacilityTax     float64  `json:"facilityTax,omitempty" yaml:"facility_tax,omitempty"`
	ReprocessingTax float64  `json:"reprocessingTax,omitempty" yaml:"reprocessing_tax,omitempty"`
	// ReprocessingRigTier is 0 (none), 1 (T1) or 2 (T2) for the structure's
	// reprocessing rig, if any.
	ReprocessingRigTier int `json:"reprocessingRigTier,omitempty" yaml:"reprocessing_rig_tier,omitempty"`
	// Standing is the higher of personal and corporation standing with an NPC
	// station's owner, for its reprocessing tax.
	Standing    float64      `json:"standing,omitempty" yaml:"standing,omitempty"`
	Calibration *Calibration `json:"calibration,omitempty" yaml:"calibration,omitempty"`
	Notes       string       `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// Calibration holds rates observed from real jobs at a facility, which take
// precedence over estimates.
type Calibration struct {
	FeeRate        float64 `json:"feeRate,omitempty" yaml:"fee_rate,omitempty"` // fee ÷ estimated item value
	FromJobID      int64   `json:"fromJobId,omitempty" yaml:"from_job_id,omitempty"`
	ObservedAt     string  `json:"observedAt,omitempty" yaml:"observed_at,omitempty"`         // RFC 3339
	TimeMultiplier float64 `json:"timeMultiplier,omitempty" yaml:"time_multiplier,omitempty"` // observed duration ÷ base duration after TE and skills
}

// HaulProfile is a hauling ship, its pilot, and their routing preferences.
type HaulProfile struct {
	Name       string  `json:"name" yaml:"name"`
	ShipTypeID int64   `json:"shipTypeId" yaml:"ship_type_id"`
	ShipName   string  `json:"shipName,omitempty" yaml:"ship_name,omitempty"`
	PilotID    int64   `json:"pilotId" yaml:"pilot_id"`
	PilotName  string  `json:"pilotName,omitempty" yaml:"pilot_name,omitempty"`
	HoldM3     float64 `json:"holdM3"`
	// OutboundRoute and ReturnRoute name saved routes (routes.yaml) pinned
	// for this profile's regular run, hub → destination and back.
	OutboundRoute string     `json:"outboundRoute,omitempty" yaml:"outbound_route,omitempty"`
	ReturnRoute   string     `json:"returnRoute,omitempty" yaml:"return_route,omitempty"`
	Prefs         RoutePrefs `json:"prefs" yaml:"prefs"`
	Notes         string     `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// RoutePrefs are saved routing preferences. IDs are systems, constellations,
// regions or alliances as named.
type RoutePrefs struct {
	AvoidSystems         []int64 `json:"avoidSystems,omitempty" yaml:"avoid_systems,omitempty"`
	AvoidConstellations  []int64 `json:"avoidConstellations,omitempty" yaml:"avoid_constellations,omitempty"`
	AvoidRegions         []int64 `json:"avoidRegions,omitempty" yaml:"avoid_regions,omitempty"`
	AvoidAlliances       []int64 `json:"avoidAlliances,omitempty" yaml:"avoid_alliances,omitempty"`
	AvoidBadStanding     bool    `json:"avoidBadStanding,omitempty" yaml:"avoid_bad_standing,omitempty"`
	PreferAlliances      []int64 `json:"preferAlliances,omitempty" yaml:"prefer_alliances,omitempty"`
	Midpoints            []int64 `json:"midpoints,omitempty" yaml:"midpoints,omitempty"`
	MidpointsRequired    bool    `json:"midpointsRequired,omitempty" yaml:"midpoints_required,omitempty"`
	PreferStationSystems bool    `json:"preferStationSystems,omitempty" yaml:"prefer_station_systems,omitempty"`
	AvoidIncursions      bool    `json:"avoidIncursions,omitempty" yaml:"avoid_incursions,omitempty"`
	Optimize             string  `json:"optimize,omitempty" yaml:"optimize,omitempty"` // fuel, jumps or fatigue
	MaxGates             int     `json:"maxGates,omitempty" yaml:"max_gates,omitempty"`
	MaxRecentKills       int     `json:"maxRecentKills,omitempty" yaml:"max_recent_kills,omitempty"`
}

func sameName(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }

func upsert[T any](list []T, item T, name func(T) string) []T {
	for i, x := range list {
		if sameName(name(x), name(item)) {
			list[i] = item
			return list
		}
	}
	return append(list, item)
}

func remove[T any](list []T, n string, name func(T) string) ([]T, error) {
	i := slices.IndexFunc(list, func(x T) bool { return sameName(name(x), n) })
	if i < 0 {
		return list, fmt.Errorf("no profile named %q", n)
	}
	return slices.Delete(list, i, i+1), nil
}

func find[T any](list []T, n string, name func(T) string) (T, bool) {
	for _, x := range list {
		if sameName(name(x), n) {
			return x, true
		}
	}
	var zero T
	return zero, false
}

func facName(p FacilityProfile) string { return p.Name }
func haulName(p HaulProfile) string    { return p.Name }

// UpsertFacilityProfile adds or replaces (by case-insensitive name) a profile.
func (c *Config) UpsertFacilityProfile(p FacilityProfile) {
	c.FacilityProfiles = upsert(c.FacilityProfiles, p, facName)
}

// DeleteFacilityProfile removes a profile by name.
func (c *Config) DeleteFacilityProfile(name string) error {
	var err error
	c.FacilityProfiles, err = remove(c.FacilityProfiles, name, facName)
	return err
}

// FacilityProfile looks up a profile by name.
func (c Config) FacilityProfile(name string) (FacilityProfile, bool) {
	return find(c.FacilityProfiles, name, facName)
}

// UpsertHaulProfile adds or replaces a haul profile.
func (c *Config) UpsertHaulProfile(p HaulProfile) {
	c.HaulProfiles = upsert(c.HaulProfiles, p, haulName)
}

// DeleteHaulProfile removes a haul profile by name.
func (c *Config) DeleteHaulProfile(name string) error {
	var err error
	c.HaulProfiles, err = remove(c.HaulProfiles, name, haulName)
	return err
}

// HaulProfile looks up a haul profile by name.
func (c Config) HaulProfile(name string) (HaulProfile, bool) {
	return find(c.HaulProfiles, name, haulName)
}
