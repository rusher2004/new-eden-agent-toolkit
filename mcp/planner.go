package mcp

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/industry"
	"github.com/rusher2004/new-eden-agent-toolkit/market"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// fetchSellOrders returns every sell order for a type in a region. A package
// variable so tests can fake it.
var fetchSellOrders = func(ctx context.Context, userAgent string, regionID, typeID int64) ([]market.Order, error) {
	raw, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return listMarketOrders(ctx, userAgent, regionID, "sell", typeID, page)
	})
	if err != nil {
		return nil, fmt.Errorf("sell orders for type %d: %w", typeID, err)
	}
	out := make([]market.Order, 0, len(raw))
	for _, o := range raw {
		out = append(out, market.Order{Price: o.Price, Volume: o.VolumeRemain, LocationID: o.LocationId})
	}
	return out, nil
}

// planner gathers SDE, ESI and config inputs for the industry tools and
// caches them for the life of one tool call.
type planner struct {
	version, clientID string
	store             *config.Store
	sde               *sde.Store

	mu       sync.Mutex
	recipes  map[int64]*industry.Recipe // nil value = not buildable
	basics   map[int64]sde.TypeBasic
	filters  map[int64]industry.TargetFilter
	systems  map[int64]sde.RoutingSystem
	graph    *nav.RouteGraph
	adjusted map[int64]float64
	indices  map[int64]map[string]float64
	skillsBy map[int32]map[int64]int
	// marketCharacter reads a structure hub's market when set; otherwise
	// the hub's saved read_via character does.
	marketCharacter int32
}

func newPlanner(version, clientID string, store *config.Store, sdeStore *sde.Store) *planner {
	return &planner{version: version, clientID: clientID, store: store, sde: sdeStore,
		recipes: map[int64]*industry.Recipe{}, basics: map[int64]sde.TypeBasic{}, skillsBy: map[int32]map[int64]int{}}
}

func (p *planner) ua() string { return sso.UserAgent(p.version) }

// requireCharacter fails unless the character is logged in to neat.
func (p *planner) requireCharacter(id int32) error {
	found, err := characterExists(p.store, id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("character %d: %w", id, ErrCharacterNotFound)
	}
	return nil
}

// typeBasics loads and caches basic type data.
func (p *planner) typeBasics(ctx context.Context, ids []int64) (map[int64]sde.TypeBasic, error) {
	p.mu.Lock()
	var missing []int64
	for _, id := range ids {
		if _, ok := p.basics[id]; !ok && !slices.Contains(missing, id) {
			missing = append(missing, id)
		}
	}
	p.mu.Unlock()
	if len(missing) > 0 {
		got, err := p.sde.TypeBasics(ctx, p.ua(), missing)
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		for k, v := range got {
			p.basics[k] = v
		}
		p.mu.Unlock()
	}
	out := make(map[int64]sde.TypeBasic, len(ids))
	p.mu.Lock()
	for _, id := range ids {
		if b, ok := p.basics[id]; ok {
			out[id] = b
		}
	}
	p.mu.Unlock()
	return out, nil
}

func (p *planner) typeName(ctx context.Context, id int64) string {
	b, err := p.typeBasics(ctx, []int64{id})
	if err != nil || b[id].Name == "" {
		return fmt.Sprintf("type %d", id)
	}
	return b[id].Name
}

func (p *planner) system(ctx context.Context, id int64) (sde.RoutingSystem, error) {
	if p.systems == nil {
		all, err := p.sde.RoutingSystems(ctx, p.ua())
		if err != nil {
			return sde.RoutingSystem{}, err
		}
		p.systems = make(map[int64]sde.RoutingSystem, len(all))
		for _, s := range all {
			p.systems[s.ID] = s
		}
	}
	s, ok := p.systems[id]
	if !ok {
		return sde.RoutingSystem{}, fmt.Errorf("unknown solar system %d", id)
	}
	return s, nil
}

// sellBooks fetches sell order books for many types with bounded concurrency.
func (p *planner) sellBooks(ctx context.Context, regionID int64, typeIDs []int64) (map[int64][]market.Order, error) {
	ids := slices.Clone(typeIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	out := make(map[int64][]market.Order, len(ids))
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(id int64) {
			defer wg.Done()
			defer func() { <-sem }()
			orders, err := fetchSellOrders(ctx, p.ua(), regionID, id)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			out[id] = orders
		}(id)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

// skills returns a character's active skill levels.
func (p *planner) skills(ctx context.Context, characterID int32) (map[int64]int, error) {
	if s, ok := p.skillsBy[characterID]; ok {
		return s, nil
	}
	if err := p.requireCharacter(characterID); err != nil {
		return nil, err
	}
	raw, _, _, err := listCharacterSkills(ctx, p.clientID, characterID, p.ua())
	if err != nil {
		return nil, fmt.Errorf("skills of %d: %w", characterID, err)
	}
	m := make(map[int64]int, len(raw))
	for _, s := range raw {
		m[s.SkillID] = int(s.ActiveSkillLevel)
	}
	p.skillsBy[characterID] = m
	return m, nil
}

func (p *planner) characterName(id int32) string {
	cfg, err := p.store.Load()
	if err == nil {
		for _, c := range cfg.Characters {
			if c.ID == id {
				return c.Name
			}
		}
	}
	return fmt.Sprintf("character %d", id)
}

// systemIDs resolves system names to IDs via the routing system list.
func (p *planner) systemIDs(ctx context.Context, names []string) ([]int64, error) {
	if _, err := p.system(ctx, 30000142); err != nil && p.systems == nil {
		return nil, err
	}
	byName := map[string]int64{}
	for id, s := range p.systems {
		byName[strings.ToLower(s.Name)] = id
	}
	out := make([]int64, 0, len(names))
	for _, n := range names {
		id, ok := byName[strings.ToLower(strings.TrimSpace(n))]
		if !ok {
			return nil, fmt.Errorf("unknown system %q", n)
		}
		out = append(out, id)
	}
	return out, nil
}

func sortedKeys[V any](m map[int64]V) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

type ProfileNameInput struct {
	Name string `json:"name" jsonschema:"the profile's name"`
}

type DeletedOutput struct {
	Deleted string `json:"deleted"`
}

func deleteProfileHandler(store *config.Store, del func(*config.Config, string) error) mcpsdk.ToolHandlerFor[ProfileNameInput, DeletedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ProfileNameInput) (*mcpsdk.CallToolResult, DeletedOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, DeletedOutput{}, err
		}
		if err := del(&cfg, in.Name); err != nil {
			return nil, DeletedOutput{}, err
		}
		if err := store.Save(cfg); err != nil {
			return nil, DeletedOutput{}, err
		}
		return nil, DeletedOutput{Deleted: in.Name}, nil
	}
}

// --- calibration ---
