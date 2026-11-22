package config

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Hub kinds.
const (
	HubNPCStation = "npc_station"
	HubStructure  = "structure"
)

// HubProfile is a trade hub to buy and sell at: an NPC station, whose orders
// anyone can read from its region's public order book, or a player
// structure, whose market needs a character with docking access.
type HubProfile struct {
	StationID int64    `json:"id" yaml:"id" jsonschema:"the NPC station or structure ID"`
	Name      string   `json:"name" yaml:"name"`
	Kind      string   `json:"kind,omitempty" yaml:"kind,omitempty" jsonschema:"npc_station or structure"`
	Tier      string   `json:"tier,omitempty" yaml:"tier,omitempty" jsonschema:"major (the five NPC hubs), secondary (player structure markets) or lowsec"`
	System    *NamedID `json:"system,omitempty" yaml:"system,omitempty"`
	Region    NamedID  `json:"region" yaml:"region"`
	// ReadVia is the character whose docking access last read this
	// structure hub's market, so later reads don't have to guess.
	ReadVia    *NamedID       `json:"readVia,omitempty" yaml:"read_via,omitempty" jsonschema:"the character that last read this structure hub's market"`
	Traits     []string       `json:"traits,omitempty" yaml:"traits,omitempty" jsonschema:"what the hub is good for, in a few words each"`
	Validation *HubValidation `json:"validation,omitempty" yaml:"validation,omitempty"`
	Notes      string         `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// RegionID is the hub's region, whose public order book holds an NPC
// station's orders.
func (h HubProfile) RegionID() int64 { return h.Region.ID }

// IsStructure reports whether the hub is a player structure, whose market
// is read through a character with docking access.
func (h HubProfile) IsStructure() bool {
	if h.Kind != "" {
		return h.Kind == HubStructure
	}
	return h.StationID > maxNPCStationID
}

// maxNPCStationID is the end of the NPC station ID range; anything above it
// is a player structure.
const maxNPCStationID = 64000000

// HubValidation is dated evidence of what a hub is like, measured by neat
// on ESI data: how well it stocks a basket of everyday goods, and for a
// capital market, its public contracts.
type HubValidation struct {
	Date      string         `json:"date" yaml:"date" jsonschema:"when it was measured, YYYY-MM-DD"`
	Basket    *BasketCheck   `json:"basket,omitempty" yaml:"basket,omitempty"`
	Contracts *ContractCheck `json:"contracts,omitempty" yaml:"contracts,omitempty"`
}

// BasketCheck is how many items of a basket a hub has for sale.
type BasketCheck struct {
	Stocked    int      `json:"stocked" yaml:"stocked" jsonschema:"items of the basket with at least one sell order at the hub"`
	Of         int      `json:"of" yaml:"of" jsonschema:"items in the basket"`
	SellOrders int      `json:"sellOrders" yaml:"sell_orders"`
	BuyOrders  int      `json:"buyOrders" yaml:"buy_orders"`
	Missing    []string `json:"missing,omitempty" yaml:"missing,omitempty" jsonschema:"basket items with no sell order there"`
}

// ContractCheck is the public contracts that start at a hub.
type ContractCheck struct {
	Public     int      `json:"public" yaml:"public" jsonschema:"public item exchange and auction contracts starting there"`
	Over1B     int      `json:"over1b" yaml:"over_1b" jsonschema:"of those, priced at 1B ISK or more"`
	Over10B    int      `json:"over10b" yaml:"over_10b" jsonschema:"of those, priced at 10B ISK or more"`
	ShipGroups []string `json:"shipGroups,omitempty" yaml:"ship_groups,omitempty" jsonschema:"capital and supercapital ship groups on offer in the priciest contracts"`
}

// BuiltinHubs are the hubs neat ships with, from live checks on
// 2026-11-20: the five NPC hubs, the two player markets near Jita that rank
// among the busiest, and the lowsec hub where capitals trade. Their
// validation is left for neat to measure on ESI.
var BuiltinHubs = []HubProfile{
	{StationID: 60003760, Name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant", Kind: HubNPCStation, Tier: "major", System: &NamedID{30000142, "Jita"}, Region: NamedID{10000002, "The Forge"},
		Traits: []string{"by far the largest hub, the default for price checks and for buying almost anything", "very competitive with low margins", "the main place for high-value contracts"}},
	{StationID: 60008494, Name: "Amarr VIII (Oris) - Emperor Family Academy", Kind: HubNPCStation, Tier: "major", System: &NamedID{30002187, "Amarr"}, Region: NamedID{10000043, "Domain"},
		Traits: []string{"the clear second hub"}},
	{StationID: 60011866, Name: "Dodixie IX - Moon 20 - Federation Navy Assembly Plant", Kind: HubNPCStation, Tier: "major", System: &NamedID{30002659, "Dodixie"}, Region: NamedID{10000032, "Sinq Laison"}},
	{StationID: 60005686, Name: "Hek VIII - Moon 12 - Boundless Creation Factory", Kind: HubNPCStation, Tier: "major", System: &NamedID{30002053, "Hek"}, Region: NamedID{10000042, "Metropolis"},
		Traits: []string{"the larger of the two Minmatar hubs"}},
	{StationID: 60004588, Name: "Rens VI - Moon 8 - Brutor Tribe Treasury", Kind: HubNPCStation, Tier: "major", System: &NamedID{30002510, "Rens"}, Region: NamedID{10000030, "Heimatar"}},
	{StationID: 1044752365771, Name: "Perimeter - 0.0% Neutral States Market HQ", Kind: HubStructure, Tier: "secondary", System: &NamedID{30000144, "Perimeter"}, Region: NamedID{10000002, "The Forge"},
		Traits: []string{"player market one jump from Jita, used almost entirely for buy orders: a place to sell into, not to shop"}},
	{StationID: 1042508032148, Name: "Perimeter - Tranquility Trading Tower", Kind: HubStructure, Tier: "secondary", System: &NamedID{30000144, "Perimeter"}, Region: NamedID{10000002, "The Forge"},
		Traits: []string{"player market one jump from Jita, small and buy-leaning"}},
	{StationID: 1022167642188, Name: "Amamake - 5 times nearly AT winners", Kind: HubStructure, Tier: "lowsec", System: &NamedID{30002537, "Amamake"}, Region: NamedID{10000030, "Heimatar"},
		Traits: []string{"the lowsec hub, and the place to buy and sell capitals and supercapitals, which trade here by contract", "a Keepstar, so supercapitals can dock", "stocks most everyday goods for sale, with few buy orders"}},
}

// DefaultHub is Jita 4-4, used when no hub profile exists.
var DefaultHub = BuiltinHubs[0]

func hubKey(h HubProfile) string { return strconv.FormatInt(h.StationID, 10) }

// UpsertHubProfile adds or replaces a hub, matched by ID, or by name when
// it has no ID yet.
func (c *Config) UpsertHubProfile(h HubProfile) {
	for i, e := range c.HubProfiles {
		if (h.StationID != 0 && e.StationID == h.StationID) || (h.StationID == 0 && sameName(e.Name, h.Name)) {
			c.HubProfiles[i] = h
			return
		}
	}
	c.HubProfiles = append(c.HubProfiles, h)
}

// DeleteHubProfile removes a hub by name, ID or system name. A system name
// shared by several hubs (Perimeter has two) deletes nothing.
func (c *Config) DeleteHubProfile(nameOrID string) error {
	i, err := c.hubIndex(nameOrID)
	if err != nil {
		return err
	}
	c.HubProfiles = slices.Delete(c.HubProfiles, i, i+1)
	return nil
}

// hubIndex finds a saved hub by name or ID, else by its system's name when
// exactly one saved hub is in that system ("Amarr" finds Amarr VIII).
func (c Config) hubIndex(nameOrID string) (int, error) {
	if i := slices.IndexFunc(c.HubProfiles, func(h HubProfile) bool {
		return sameName(h.Name, nameOrID) || strconv.FormatInt(h.StationID, 10) == strings.TrimSpace(nameOrID)
	}); i >= 0 {
		return i, nil
	}
	var in []int
	for i, h := range c.HubProfiles {
		if h.System != nil && sameName(h.System.Name, nameOrID) {
			in = append(in, i)
		}
	}
	switch len(in) {
	case 1:
		return in[0], nil
	case 0:
		return -1, fmt.Errorf("no hub named %q", nameOrID)
	default:
		var names []string
		for _, i := range in {
			names = append(names, c.HubProfiles[i].Name)
		}
		return -1, fmt.Errorf("%d hubs are in %s (%s): name one", len(in), strings.TrimSpace(nameOrID), strings.Join(names, "; "))
	}
}

// HubProfile looks up a hub by name, ID or system name (when only one saved
// hub is in that system); an empty name
// returns the first saved hub, or DefaultHub when none is saved.
func (c Config) HubProfile(nameOrID string) (HubProfile, bool) {
	if strings.TrimSpace(nameOrID) == "" {
		if len(c.HubProfiles) > 0 {
			return c.HubProfiles[0], true
		}
		return DefaultHub, true
	}
	if i, err := c.hubIndex(nameOrID); err == nil {
		return c.HubProfiles[i], true
	}
	if sameName(nameOrID, "Jita 4-4") || sameName(nameOrID, DefaultHub.Name) {
		return DefaultHub, true
	}
	return HubProfile{}, false
}

// legacyHub is a hub profile as config.json held it.
type legacyHub struct {
	Name      string `json:"name"`
	RegionID  int64  `json:"regionId"`
	StationID int64  `json:"stationId"`
}

// hub converts a legacy profile, keeping the user's name. One for a
// built-in hub's station takes that hub's system, region, tier and traits,
// since the legacy profile had none of them and it replaces the built-in.
func (l legacyHub) hub() HubProfile {
	for _, b := range BuiltinHubs {
		if b.StationID == l.StationID {
			if l.Name != "" {
				b.Name = l.Name
			}
			return b
		}
	}
	h := HubProfile{StationID: l.StationID, Name: l.Name, Region: NamedID{ID: l.RegionID}, Kind: HubNPCStation}
	if l.StationID > maxNPCStationID {
		h.Kind = HubStructure
	}
	return h
}
