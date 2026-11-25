package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/market"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

// staleValidationDays is how old a hub's validation can be before tools say
// to measure it again.
const staleValidationDays = 30

// priciestContractsToOpen caps how many of a hub's priciest contracts
// validate_trade_hub opens to see which ships are on offer (one ESI request
// each).
const priciestContractsToOpen = 15

// hubCharacter picks the character that reads a structure hub's market: the
// one the caller named for the market, else the one that last read it, else
// a character the call named for another job. It never guesses,
// since each refused lookup counts against ESI's error limit.
func (p *planner) hubCharacter(hub config.HubProfile) (int32, error) {
	if p.marketCharacter != 0 {
		return p.marketCharacter, nil
	}
	if hub.ReadVia != nil && hub.ReadVia.ID != 0 {
		return int32(hub.ReadVia.ID), nil
	}
	if p.fallbackMarketCharacter != 0 {
		return p.fallbackMarketCharacter, nil
	}
	return 0, fmt.Errorf("%s is a player structure, so its market can only be read through a character with docking access: pass characterId, or run validate_trade_hub with one so neat remembers it", hub.Name)
}

// structureOrders reads every order in a structure's market.
func (p *planner) structureOrders(ctx context.Context, characterID int32, structureID int64) ([]esiapi.MarketsStructuresStructureIdGetInner, error) {
	orders, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return listStructureMarketOrders(ctx, p.clientID, characterID, p.ua(), structureID, page)
	})
	if err != nil {
		return nil, fmt.Errorf("read structure %d's market through character %d (it needs docking access and the market scope group): %w", structureID, characterID, err)
	}
	return orders, nil
}

// hubBooks reads the sell orders for typeIDs at a hub: from its region's
// public order book for an NPC station, or from the structure's own market
// for a player structure.
func (p *planner) hubBooks(ctx context.Context, hub config.HubProfile, typeIDs []int64) (map[int64][]market.Order, error) {
	if !hub.IsStructure() {
		return p.sellBooks(ctx, hub.RegionID(), typeIDs)
	}
	char, err := p.hubCharacter(hub)
	if err != nil {
		return nil, err
	}
	orders, err := p.structureOrders(ctx, char, hub.StationID)
	if err != nil {
		return nil, err
	}
	want := map[int64]bool{}
	for _, id := range typeIDs {
		want[id] = true
	}
	out := make(map[int64][]market.Order, len(typeIDs))
	for _, o := range orders {
		if !o.IsBuyOrder && want[o.TypeId] {
			out[o.TypeId] = append(out[o.TypeId], market.Order{Price: o.Price, Volume: o.VolumeRemain, LocationID: o.LocationId})
		}
	}
	return out, nil
}

// --- hub profiles ---

// HubView is a saved hub with whether its validation needs measuring again.
type HubView struct {
	config.HubProfile
	ValidationStale bool `json:"validationStale,omitempty" jsonschema:"true when the hub has never been validated, or not in the last 30 days; run validate_trade_hub before relying on it"`
}

func hubView(h config.HubProfile, now time.Time) HubView {
	v := HubView{HubProfile: h, ValidationStale: true}
	if h.Validation != nil {
		if d, err := time.Parse("2006-01-02", h.Validation.Date); err == nil && now.Sub(d) <= staleValidationDays*24*time.Hour {
			v.ValidationStale = false
		}
	}
	return v
}

// now is overridable in tests.
var now = time.Now

// ListHubProfilesOutput is list_hub_profiles' result.
type ListHubProfilesOutput struct {
	Hubs []HubView `json:"hubs" jsonschema:"saved hubs; the first is the default"`
}

func listHubProfilesHandler(store ConfigStore) mcpsdk.ToolHandlerFor[struct{}, ListHubProfilesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, ListHubProfilesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListHubProfilesOutput{}, err
		}
		out := ListHubProfilesOutput{Hubs: []HubView{}}
		for _, h := range cfg.HubProfiles {
			out.Hubs = append(out.Hubs, hubView(h, now()))
		}
		return nil, out, nil
	}
}

// SaveHubProfileInput is save_hub_profile's input.
type SaveHubProfileInput struct {
	ID          int64    `json:"id" jsonschema:"the NPC station or player structure ID"`
	Name        string   `json:"name,omitempty" jsonschema:"a name to call it by; defaults to the station's or structure's name"`
	CharacterID int32    `json:"characterId,omitempty" jsonschema:"for a player structure: a logged-in character with docking access, to read its name, system and market"`
	Tier        string   `json:"tier,omitempty" jsonschema:"major, secondary or lowsec"`
	Traits      []string `json:"traits,omitempty" jsonschema:"what the hub is good for, in the user's words; replaces the saved list, and omitting it keeps the list"`
	Notes       *string  `json:"notes,omitempty" jsonschema:"what the user says about the hub; omit to keep the notes it has"`
}

// SaveHubProfileOutput is the saved hub and every value the save changed.
type SaveHubProfileOutput struct {
	Hub      config.HubProfile `json:"hub"`
	Changes  []config.Change   `json:"changes"`
	Warnings []string          `json:"warnings,omitempty"`
}

// hubFacts reads a hub's name, kind, system and region from the game.
func (p *planner) hubFacts(ctx context.Context, id int64, characterID int32) (config.HubProfile, error) {
	h := config.HubProfile{StationID: id}
	var systemID int64
	if st, ok, err := p.sde.Station(ctx, p.ua(), id); err != nil {
		return h, fmt.Errorf("look up station %d: %w", id, err)
	} else if ok {
		h.Name, h.Kind, systemID = st.Name, config.HubNPCStation, st.SystemID
	} else {
		if characterID == 0 {
			return h, fmt.Errorf("%d isn't an NPC station, so it's read as a player structure, which needs characterId: a logged-in character with docking access", id)
		}
		us, err := getUniverseStructure(ctx, p.clientID, characterID, p.ua(), id)
		if err != nil {
			return h, fmt.Errorf("read structure %d through character %d (it needs docking access and the structures scope group): %w", id, characterID, err)
		}
		h.Name, h.Kind, systemID = us.Name, config.HubStructure, us.SolarSystemID
	}
	sys, err := p.system(ctx, systemID)
	if err != nil {
		return h, err
	}
	h.System = &config.NamedID{ID: sys.ID, Name: sys.Name}
	h.Region = config.NamedID{ID: sys.RegionID, Name: sys.RegionName}
	return h, nil
}

func saveHubProfileHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[SaveHubProfileInput, SaveHubProfileOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveHubProfileInput) (*mcpsdk.CallToolResult, SaveHubProfileOutput, error) {
		if in.ID == 0 {
			return nil, SaveHubProfileOutput{}, fmt.Errorf("save hub profile: id is required")
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, SaveHubProfileOutput{}, err
		}
		if in.CharacterID != 0 {
			if _, ok := cfg.Character(in.CharacterID); !ok {
				return nil, SaveHubProfileOutput{}, fmt.Errorf("save hub profile through character %d: %w", in.CharacterID, ErrCharacterNotFound)
			}
		}
		p := newPlanner(version, clientID, store, sdeStore)
		facts, factsErr := p.hubFacts(ctx, in.ID, in.CharacterID)
		// The lookup above goes over the network, so the hub is merged into
		// whatever is saved now, under Update's lock, rather than saved over
		// anything written meanwhile.
		var h config.HubProfile
		var warnings []string
		changes, err := store.Update(func(cfg *config.Config) error {
			var saved bool
			h, saved = cfg.HubProfile(fmt.Sprint(in.ID))
			switch {
			case factsErr == nil:
				name := h.Name
				h.StationID, h.Kind, h.System, h.Region = facts.StationID, facts.Kind, facts.System, facts.Region
				h.Name = name
				if h.Name == "" {
					h.Name = facts.Name
				}
				if facts.Kind == config.HubStructure {
					c, _ := cfg.Character(in.CharacterID)
					h.ReadVia = &config.NamedID{ID: int64(c.ID), Name: c.Name}
				}
			case !saved:
				return fmt.Errorf("save hub profile: %w", factsErr)
			default:
				warnings = append(warnings, "kept the saved name, system and region, since they couldn't be read from the game: "+factsErr.Error())
			}
			if n := strings.TrimSpace(in.Name); n != "" {
				h.Name = n
			}
			if in.Tier != "" {
				h.Tier = in.Tier
			}
			if in.Traits != nil {
				h.Traits = in.Traits
			}
			if in.Notes != nil {
				h.Notes = strings.TrimSpace(*in.Notes)
			}
			cfg.UpsertHubProfile(h)
			return nil
		})
		if err != nil {
			return nil, SaveHubProfileOutput{}, err
		}
		if changes == nil {
			changes = []config.Change{}
		}
		return nil, SaveHubProfileOutput{Hub: h, Changes: changes, Warnings: warnings}, nil
	}
}

// DeleteHubProfileInput is delete_hub_profile's input.
type DeleteHubProfileInput struct {
	Hub string `json:"hub" jsonschema:"the hub's name or ID"`
}

func deleteHubProfileHandler(store ConfigStore) mcpsdk.ToolHandlerFor[DeleteHubProfileInput, DeletedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteHubProfileInput) (*mcpsdk.CallToolResult, DeletedOutput, error) {
		if strings.TrimSpace(in.Hub) == "" {
			return nil, DeletedOutput{}, fmt.Errorf("no hub named %q", in.Hub)
		}
		var name string
		if _, err := store.Update(func(cfg *config.Config) error {
			h, ok := cfg.HubProfile(in.Hub)
			if !ok {
				// Says whether the name matched nothing or several hubs.
				return cfg.DeleteHubProfile(in.Hub)
			}
			name = h.Name
			return cfg.DeleteHubProfile(fmt.Sprint(h.StationID))
		}); err != nil {
			return nil, DeletedOutput{}, err
		}
		return nil, DeletedOutput{Deleted: name}, nil
	}
}

// --- validate_trade_hub ---

// ValidateTradeHubInput is validate_trade_hub's input.
type ValidateTradeHubInput struct {
	Hub         string      `json:"hub" jsonschema:"a saved hub's name or ID, or a new station or structure ID to measure and save"`
	CharacterID int32       `json:"characterId,omitempty" jsonschema:"for a player structure: a logged-in character with docking access; defaults to the one that last read it"`
	Basket      []ItemInput `json:"basket,omitempty" jsonschema:"items to check for sale instead of the default basket of 24 everyday goods; quantities are ignored"`
	Contracts   bool        `json:"contracts,omitempty" jsonschema:"also count public contracts starting there by value, and list the capital ship groups on offer; the check for a capital market"`
}

// ValidateTradeHubOutput is the measured hub and what changed.
type ValidateTradeHubOutput struct {
	Hub     config.HubProfile `json:"hub"`
	Changes []config.Change   `json:"changes"`
}

func validateTradeHubHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ValidateTradeHubInput, ValidateTradeHubOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ValidateTradeHubInput) (*mcpsdk.CallToolResult, ValidateTradeHubOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ValidateTradeHubOutput{}, err
		}
		if in.CharacterID != 0 {
			if _, ok := cfg.Character(in.CharacterID); !ok {
				return nil, ValidateTradeHubOutput{}, fmt.Errorf("validate trade hub through character %d: %w", in.CharacterID, ErrCharacterNotFound)
			}
		}
		p := newPlanner(version, clientID, store, sdeStore)
		p.marketCharacter = in.CharacterID
		hub, ok := cfg.HubProfile(in.Hub)
		if !ok || strings.TrimSpace(in.Hub) == "" {
			var id int64
			if _, err := fmt.Sscan(strings.TrimSpace(in.Hub), &id); err != nil || id == 0 {
				return nil, ValidateTradeHubOutput{}, fmt.Errorf("no hub named %q; give a saved hub's name or a station or structure ID", in.Hub)
			}
			if hub, err = p.hubFacts(ctx, id, in.CharacterID); err != nil {
				return nil, ValidateTradeHubOutput{}, fmt.Errorf("validate trade hub: %w", err)
			}
		}

		basket := market.DefaultBasket
		if len(in.Basket) > 0 {
			tq, err := p.resolveItems(ctx, in.Basket)
			if err != nil {
				return nil, ValidateTradeHubOutput{}, err
			}
			basket = nil
			for _, t := range tq {
				basket = append(basket, t.TypeID)
			}
		}
		orders, err := p.allOrders(ctx, hub, basket)
		if err != nil {
			return nil, ValidateTradeHubOutput{}, err
		}
		score := market.ScoreBasket(orders, hub.StationID, basket)
		v := &config.HubValidation{Date: today(), Basket: &config.BasketCheck{Stocked: score.Stocked, Of: score.Of, SellOrders: score.SellOrders, BuyOrders: score.BuyOrders}}
		if len(score.Missing) > 0 {
			basics, err := p.typeBasics(ctx, score.Missing)
			if err != nil {
				return nil, ValidateTradeHubOutput{}, fmt.Errorf("name the missing basket items: %w", err)
			}
			for _, id := range score.Missing {
				name := basics[id].Name
				if name == "" {
					name = fmt.Sprintf("type %d", id)
				}
				v.Basket.Missing = append(v.Basket.Missing, name)
			}
		}
		if in.Contracts {
			if v.Contracts, err = p.contractCheck(ctx, hub); err != nil {
				return nil, ValidateTradeHubOutput{}, err
			}
		}
		var readVia *config.NamedID
		if hub.IsStructure() {
			char, err := p.hubCharacter(hub)
			if err != nil {
				return nil, ValidateTradeHubOutput{}, err
			}
			rv := config.NamedID{ID: int64(char)}
			if c, ok := cfg.Character(char); ok {
				rv.Name = c.Name
			} else if hub.ReadVia != nil && hub.ReadVia.ID == int64(char) {
				rv.Name = hub.ReadVia.Name
			}
			readVia = &rv
		}
		// The measuring above goes over the network, so the result is
		// recorded on the hub as saved now, under Update's lock, rather than
		// saved over anything written meanwhile.
		changes, err := store.Update(func(c *config.Config) error {
			if latest, ok := c.HubProfile(fmt.Sprint(hub.StationID)); ok {
				hub = latest
			}
			hub.Validation = v
			if readVia != nil {
				hub.ReadVia = readVia
			}
			c.UpsertHubProfile(hub)
			return nil
		})
		if err != nil {
			return nil, ValidateTradeHubOutput{}, err
		}
		if changes == nil {
			changes = []config.Change{}
		}
		return nil, ValidateTradeHubOutput{Hub: hub, Changes: changes}, nil
	}
}

// allOrders reads both sides of the market at a hub for typeIDs.
func (p *planner) allOrders(ctx context.Context, hub config.HubProfile, typeIDs []int64) ([]market.BookOrder, error) {
	var out []market.BookOrder
	if hub.IsStructure() {
		char, err := p.hubCharacter(hub)
		if err != nil {
			return nil, err
		}
		orders, err := p.structureOrders(ctx, char, hub.StationID)
		if err != nil {
			return nil, err
		}
		for _, o := range orders {
			out = append(out, market.BookOrder{TypeID: o.TypeId, Price: o.Price, Volume: o.VolumeRemain, LocationID: o.LocationId, IsBuy: o.IsBuyOrder})
		}
		return out, nil
	}
	for _, id := range typeIDs {
		orders, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
			return listMarketOrders(ctx, p.ua(), hub.RegionID(), "all", id, page)
		})
		if err != nil {
			return nil, fmt.Errorf("orders for type %d: %w", id, err)
		}
		for _, o := range orders {
			out = append(out, market.BookOrder{TypeID: id, Price: o.Price, Volume: o.VolumeRemain, LocationID: o.LocationId, IsBuy: o.IsBuyOrder})
		}
	}
	return out, nil
}

// contractCheck counts a hub's public contracts by value and lists the
// capital ship groups in the priciest ones.
func (p *planner) contractCheck(ctx context.Context, hub config.HubProfile) (*config.ContractCheck, error) {
	all, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]Contract, int32, error) {
		return listPublicContracts(ctx, p.ua(), hub.RegionID(), page)
	})
	if err != nil {
		return nil, fmt.Errorf("public contracts in %s: %w", hub.Region.Name, err)
	}
	var cs []market.PublicContract
	for _, c := range all {
		pc := market.PublicContract{ID: c.ContractID, Type: c.Type}
		if c.Price != nil {
			pc.Price = *c.Price
		}
		if c.StartLocationID != nil {
			pc.StartLocationID = *c.StartLocationID
		}
		cs = append(cs, pc)
	}
	bands := market.BandContracts(cs, hub.StationID, priciestContractsToOpen)
	check := &config.ContractCheck{Public: bands.Public, Over1B: bands.Over1B, Over10B: bands.Over10B}
	var typeIDs []int64
	for _, id := range bands.Priciest {
		items, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]ContractItem, int32, error) {
			return listPublicContractItems(ctx, p.ua(), id, page)
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("items of contract %d: %w", id, ctx.Err())
			}
			continue // a contract can expire between the listing and this read
		}
		for _, it := range items {
			if it.IsIncluded {
				typeIDs = append(typeIDs, it.TypeID)
			}
		}
	}
	basics, err := p.typeBasics(ctx, typeIDs)
	if err != nil {
		return nil, err
	}
	for _, b := range basics {
		if b.CategoryID == shipCategoryID && !slices.Contains(check.ShipGroups, b.GroupName) {
			check.ShipGroups = append(check.ShipGroups, b.GroupName)
		}
	}
	slices.Sort(check.ShipGroups)
	return check, nil
}

// shipCategoryID is the SDE category of ships.
const shipCategoryID = 6

// --- compare_trade_hubs ---

// CompareTradeHubsInput is compare_trade_hubs' input.
type CompareTradeHubsInput struct {
	Items       []ItemInput `json:"items" jsonschema:"the shopping list"`
	From        string      `json:"from,omitempty" jsonschema:"a system to measure gate jumps from, such as the staging system"`
	Hubs        []string    `json:"hubs,omitempty" jsonschema:"hub names or IDs to compare; omit for every saved hub"`
	CharacterID int32       `json:"characterId,omitempty" jsonschema:"a character with docking access, for structure hubs that haven't been read before"`
}

// HubComparison is one hub's result.
type HubComparison struct {
	market.HubRank
	HubID           int64  `json:"hubId"`
	System          string `json:"system,omitempty"`
	SecurityDisplay string `json:"securityDisplay,omitempty"`
	SecurityColor   string `json:"securityColor,omitempty"`
	ValidationStale bool   `json:"validationStale,omitempty"`
}

// CompareTradeHubsOutput ranks the hubs.
type CompareTradeHubsOutput struct {
	Hubs     []HubComparison `json:"hubs" jsonschema:"best first: most items supplied in full, then cheapest among hubs covering the same items, then fewest jumps"`
	Warnings []string        `json:"warnings,omitempty"`
}

func compareTradeHubsHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[CompareTradeHubsInput, CompareTradeHubsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CompareTradeHubsInput) (*mcpsdk.CallToolResult, CompareTradeHubsOutput, error) {
		if len(in.Items) == 0 {
			return nil, CompareTradeHubsOutput{}, fmt.Errorf("compare trade hubs: give at least one item")
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, CompareTradeHubsOutput{}, err
		}
		p := newPlanner(version, clientID, store, sdeStore)
		// characterId is for structure hubs no character has read yet; a
		// hub's own read_via still wins.
		p.fallbackMarketCharacter = in.CharacterID
		hubs := cfg.HubProfiles
		if len(in.Hubs) > 0 {
			hubs = nil
			for _, n := range in.Hubs {
				h, ok := cfg.HubProfile(n)
				if !ok {
					return nil, CompareTradeHubsOutput{}, fmt.Errorf("no hub named %q", n)
				}
				hubs = append(hubs, h)
			}
		}
		items, err := p.resolveItems(ctx, in.Items)
		if err != nil {
			return nil, CompareTradeHubsOutput{}, err
		}
		var from int64
		var g interface{ GateJumps(int64, int64) int }
		if in.From != "" {
			ids, err := p.systemIDs(ctx, []string{in.From})
			if err != nil {
				return nil, CompareTradeHubsOutput{}, err
			}
			from = ids[0]
			if g, err = p.routeGraph(ctx); err != nil {
				return nil, CompareTradeHubsOutput{}, err
			}
		}
		var ids []int64
		for _, it := range items {
			ids = append(ids, it.TypeID)
		}
		var out CompareTradeHubsOutput
		var quotes []market.HubQuote
		byID := map[int64]config.HubProfile{}
		for _, h := range hubs {
			books, err := p.hubBooks(ctx, h, ids)
			if err != nil {
				out.Warnings = append(out.Warnings, "skipped "+h.Name+": "+err.Error())
				continue
			}
			hq := market.HubQuote{Hub: h.Name, ID: h.StationID, Jumps: -1}
			for _, it := range items {
				hq.Quotes = append(hq.Quotes, market.WalkBook(it.TypeID, books[it.TypeID], it.Quantity, h.StationID))
			}
			if g != nil && h.System != nil {
				hq.Jumps = g.GateJumps(from, h.System.ID)
			}
			quotes = append(quotes, hq)
			byID[h.StationID] = h
		}
		out.Hubs = []HubComparison{}
		for _, r := range market.RankHubs(quotes) {
			h := byID[r.ID]
			c := HubComparison{HubRank: r, HubID: h.StationID, ValidationStale: hubView(h, now()).ValidationStale}
			if h.System != nil {
				c.System = h.System.Name
				if sys, err := p.system(ctx, h.System.ID); err == nil {
					c.SecurityDisplay, c.SecurityColor = nav.SecurityLabel(sys.Security), nav.SecurityColor(sys.Security)
				}
			}
			out.Hubs = append(out.Hubs, c)
		}
		return nil, out, nil
	}
}
