package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/market"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

type SaveHubProfileInput struct {
	Name      string `json:"name"`
	RegionID  int64  `json:"regionId" jsonschema:"region whose market to read, e.g. 10000002 for The Forge"`
	StationID int64  `json:"stationId" jsonschema:"station or structure orders must be at, e.g. 60003760 for Jita 4-4"`
}

type ListHubProfilesOutput struct {
	Profiles []config.HubProfile `json:"profiles"`
	Default  config.HubProfile   `json:"default"`
}

func saveHubProfileHandler(store *config.Store) mcpsdk.ToolHandlerFor[SaveHubProfileInput, config.HubProfile] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SaveHubProfileInput) (*mcpsdk.CallToolResult, config.HubProfile, error) {
		if strings.TrimSpace(in.Name) == "" || in.RegionID == 0 || in.StationID == 0 {
			return nil, config.HubProfile{}, fmt.Errorf("save hub profile: name, regionId and stationId are required")
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, config.HubProfile{}, err
		}
		h := config.HubProfile{Name: strings.TrimSpace(in.Name), RegionID: in.RegionID, StationID: in.StationID}
		cfg.UpsertHubProfile(h)
		if err := store.Save(cfg); err != nil {
			return nil, config.HubProfile{}, err
		}
		return nil, h, nil
	}
}

func listHubProfilesHandler(store *config.Store) mcpsdk.ToolHandlerFor[struct{}, ListHubProfilesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, ListHubProfilesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListHubProfilesOutput{}, err
		}
		return nil, ListHubProfilesOutput{Profiles: cfg.HubProfiles, Default: config.DefaultHub}, nil
	}
}

// ItemInput names an item by name or type ID, with a quantity.
type ItemInput struct {
	Name     string `json:"name,omitempty" jsonschema:"item name, e.g. Tritanium"`
	TypeID   int64  `json:"typeId,omitempty" jsonschema:"item type ID; used when name is empty"`
	Quantity int64  `json:"quantity"`
}

func (p *planner) resolveItems(ctx context.Context, items []ItemInput) ([]industry.TypeQty, error) {
	var names []string
	for _, it := range items {
		if it.TypeID == 0 {
			names = append(names, it.Name)
		}
	}
	ids, err := p.sde.TypesByName(ctx, p.ua(), names)
	if err != nil {
		return nil, err
	}
	out := make([]industry.TypeQty, 0, len(items))
	for _, it := range items {
		id := it.TypeID
		if id == 0 {
			if id = ids[it.Name]; id == 0 {
				return nil, fmt.Errorf("unknown item %q", it.Name)
			}
		}
		out = append(out, industry.TypeQty{TypeID: id, Quantity: it.Quantity})
	}
	return out, nil
}

func (p *planner) resolveType(ctx context.Context, name string) (int64, error) {
	ids, err := p.sde.TypesByName(ctx, p.ua(), []string{name})
	if err != nil {
		return 0, err
	}
	if ids[name] == 0 {
		return 0, fmt.Errorf("unknown item %q", name)
	}
	return ids[name], nil
}

func (p *planner) hub(name string) (config.HubProfile, error) {
	cfg, err := p.store.Load()
	if err != nil {
		return config.HubProfile{}, err
	}
	h, ok := cfg.HubProfile(name)
	if !ok {
		return config.HubProfile{}, fmt.Errorf("no hub profile named %q", name)
	}
	return h, nil
}

// --- quotes ---

// QuoteLine is one item priced against the hub's order book.
type QuoteLine struct {
	TypeID    int64   `json:"typeId"`
	Name      string  `json:"name"`
	Quantity  int64   `json:"quantity"`
	Average   float64 `json:"averagePrice"`
	Lowest    float64 `json:"lowestPrice"`
	Total     float64 `json:"total"`
	Shortfall int64   `json:"shortfall,omitempty"`
	Depth     int64   `json:"depthAtHub"`
	M3        float64 `json:"m3"`
}

func (p *planner) quote(ctx context.Context, hub config.HubProfile, items []industry.TypeQty, books map[int64][]market.Order) ([]QuoteLine, error) {
	var ids []int64
	for _, it := range items {
		ids = append(ids, it.TypeID)
	}
	if books == nil {
		var err error
		if books, err = p.sellBooks(ctx, hub.RegionID, ids); err != nil {
			return nil, err
		}
	}
	basics, err := p.typeBasics(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]QuoteLine, 0, len(items))
	for _, it := range items {
		q := market.WalkBook(it.TypeID, books[it.TypeID], it.Quantity, hub.StationID)
		b := basics[it.TypeID]
		out = append(out, QuoteLine{TypeID: it.TypeID, Name: b.Name, Quantity: it.Quantity, Average: q.Average, Lowest: q.Lowest, Total: q.Total,
			Shortfall: q.Shortfall, Depth: q.Depth, M3: b.Volume * float64(it.Quantity)})
	}
	return out, nil
}

func multibuyOf(lines []QuoteLine) string {
	ml := make([]market.MultibuyLine, 0, len(lines))
	for _, l := range lines {
		ml = append(ml, market.MultibuyLine{Name: l.Name, Quantity: l.Quantity})
	}
	return market.Multibuy(ml)
}

type QuoteMarketPurchaseInput struct {
	Items []ItemInput `json:"items"`
	Hub   string      `json:"hub,omitempty" jsonschema:"hub profile name; default Jita 4-4"`
}

type QuoteMarketPurchaseOutput struct {
	Hub      string      `json:"hub"`
	Lines    []QuoteLine `json:"lines"`
	Total    float64     `json:"total"`
	M3       float64     `json:"m3"`
	Multibuy string      `json:"multibuy" jsonschema:"paste into the in-game Multibuy window's import"`
}

func quoteMarketPurchaseHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[QuoteMarketPurchaseInput, QuoteMarketPurchaseOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in QuoteMarketPurchaseInput) (*mcpsdk.CallToolResult, QuoteMarketPurchaseOutput, error) {
		p := newPlanner(version, clientID, store, sdeStore)
		hub, err := p.hub(in.Hub)
		if err != nil {
			return nil, QuoteMarketPurchaseOutput{}, err
		}
		items, err := p.resolveItems(ctx, in.Items)
		if err != nil {
			return nil, QuoteMarketPurchaseOutput{}, err
		}
		lines, err := p.quote(ctx, hub, items, nil)
		if err != nil {
			return nil, QuoteMarketPurchaseOutput{}, err
		}
		out := QuoteMarketPurchaseOutput{Hub: hub.Name, Lines: lines, Multibuy: multibuyOf(lines)}
		for _, l := range lines {
			out.Total += l.Total
			out.M3 += l.M3
		}
		return nil, out, nil
	}
}
