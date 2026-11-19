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
	Name            string   `json:"name"`
	Kind            string   `json:"kind"` // npc_station, upwell or starbase
	StationID       int64    `json:"stationId,omitempty"`
	StructureType   int64    `json:"structureType,omitempty"`
	SystemID        int64    `json:"systemId"`
	RigTypeIDs      []int64  `json:"rigTypeIds,omitempty"`
	Services        []string `json:"services,omitempty"`
	SupercapUpgrade bool     `json:"supercapUpgrade,omitempty"`
	FacilityTax     float64  `json:"facilityTax,omitempty"`
	ReprocessingTax float64  `json:"reprocessingTax,omitempty"`
	// ReprocessingRigTier is 0 (none), 1 (T1) or 2 (T2) for the structure's
	// reprocessing rig, if any.
	ReprocessingRigTier int `json:"reprocessingRigTier,omitempty"`
	// Standing is the higher of personal and corporation standing with an NPC
	// station's owner, for its reprocessing tax.
	Standing    float64      `json:"standing,omitempty"`
	Calibration *Calibration `json:"calibration,omitempty"`
	Notes       string       `json:"notes,omitempty"`
}

// Calibration holds rates observed from real jobs at a facility, which take
// precedence over estimates.
type Calibration struct {
	FeeRate        float64 `json:"feeRate,omitempty"` // fee ÷ estimated item value
	FromJobID      int64   `json:"fromJobId,omitempty"`
	ObservedAt     string  `json:"observedAt,omitempty"`     // RFC 3339
	TimeMultiplier float64 `json:"timeMultiplier,omitempty"` // observed duration ÷ base duration after TE and skills
}

// HaulProfile is a hauling ship, its pilot, and their routing preferences.
type HaulProfile struct {
	Name       string     `json:"name"`
	ShipTypeID int64      `json:"shipTypeId"`
	PilotID    int64      `json:"pilotId"`
	HoldM3     float64    `json:"holdM3"`
	Outbound   []int64    `json:"outbound,omitempty"` // pinned route waypoints, hub → destination
	Return     []int64    `json:"return,omitempty"`
	Prefs      RoutePrefs `json:"prefs"`
}

// RoutePrefs are saved routing preferences. IDs are systems, constellations,
// regions or alliances as named.
type RoutePrefs struct {
	AvoidSystems         []int64 `json:"avoidSystems,omitempty"`
	AvoidConstellations  []int64 `json:"avoidConstellations,omitempty"`
	AvoidRegions         []int64 `json:"avoidRegions,omitempty"`
	AvoidAlliances       []int64 `json:"avoidAlliances,omitempty"`
	AvoidBadStanding     bool    `json:"avoidBadStanding,omitempty"`
	PreferAlliances      []int64 `json:"preferAlliances,omitempty"`
	Midpoints            []int64 `json:"midpoints,omitempty"`
	MidpointsRequired    bool    `json:"midpointsRequired,omitempty"`
	PreferStationSystems bool    `json:"preferStationSystems,omitempty"`
	AvoidIncursions      bool    `json:"avoidIncursions,omitempty"`
	Optimize             string  `json:"optimize,omitempty"` // fuel, jumps or fatigue
	MaxGates             int     `json:"maxGates,omitempty"`
	MaxRecentKills       int     `json:"maxRecentKills,omitempty"`
}

// HubProfile is a trade hub to buy at.
type HubProfile struct {
	Name      string `json:"name"`
	RegionID  int64  `json:"regionId"`
	StationID int64  `json:"stationId"`
}

// DefaultHub is Jita 4-4, used when no hub profile exists.
var DefaultHub = HubProfile{Name: "Jita 4-4", RegionID: 10000002, StationID: 60003760}

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
func hubName(p HubProfile) string      { return p.Name }

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

// UpsertHubProfile adds or replaces a hub profile.
func (c *Config) UpsertHubProfile(p HubProfile) { c.HubProfiles = upsert(c.HubProfiles, p, hubName) }

// DeleteHubProfile removes a hub profile by name.
func (c *Config) DeleteHubProfile(name string) error {
	var err error
	c.HubProfiles, err = remove(c.HubProfiles, name, hubName)
	return err
}

// HubProfile looks up a hub profile by name; an empty name returns the first
// saved hub, or DefaultHub when none is saved.
func (c Config) HubProfile(name string) (HubProfile, bool) {
	if strings.TrimSpace(name) == "" {
		if len(c.HubProfiles) > 0 {
			return c.HubProfiles[0], true
		}
		return DefaultHub, true
	}
	if sameName(name, DefaultHub.Name) {
		if h, ok := find(c.HubProfiles, name, hubName); ok {
			return h, true
		}
		return DefaultHub, true
	}
	return find(c.HubProfiles, name, hubName)
}
