package mcp

import (
	"context"
	"fmt"
	"maps"
	"slices"
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
)

// plannerClient is the authenticated ESI calls the planner's reads make:
// skills, contacts for standings, structure markets and details, implants,
// industry jobs, blueprints and assets, each interface declared beside the
// function that uses it. A handler whose own reads need more declares an
// interface that embeds it.
type plannerClient interface {
	characterSkillsESI
	characterContactsESI
	corporationContactsESI
	allianceContactsESI
	structureMarketOrdersESI
	universeStructureESI
	characterImplantsESI
	characterIndustryJobsESI
	corporationIndustryJobsESI
	characterBlueprintsESI
	corporationBlueprintsESI
	assetsESI
	corporationAssetsESI
	assetNamesESI
	corporationAssetNamesESI
}

// plannerPublicClient is the public ESI calls the planner's reads make:
// sovereignty, incursions, kills, affiliations, prices, indices, order books,
// public contracts and name lookups.
type plannerPublicClient interface {
	sovereigntySystemsESI
	incursionsESI
	universeSystemKillsESI
	characterAffiliationsESI
	marketPricesESI
	industrySystemsESI
	marketOrdersESI
	publicContractsESI
	publicContractItemsESI
	universeNamesESI
}

// plannerConfig is the config calls the planner family makes.
type plannerConfig interface {
	Load() (config.Config, error)
	Path() string
	Update(change func(*config.Config) error) ([]config.Change, error)
}

// plannerSDE is the SDE calls the planner family makes.
type plannerSDE interface {
	AttributeInfos(ctx context.Context, userAgent string) (map[int64]sde.AttributeInfo, error)
	Blueprint(ctx context.Context, userAgent string, blueprintTypeID int64) (sde.Blueprint, bool, error)
	BlueprintSourcesForActivity(ctx context.Context, userAgent string, productTypeID, activityID int64) ([]sde.BlueprintSource, error)
	BlueprintsForProduct(ctx context.Context, userAgent string, productTypeID int64) ([]sde.BlueprintSource, error)
	CompressedOres(ctx context.Context, userAgent string) ([]sde.Ore, error)
	DogmaTypes(ctx context.Context, userAgent string, ids []int64) (map[int64]sde.DogmaType, error)
	IndustryModifiers(ctx context.Context, userAgent string, typeID int64) ([]sde.IndustryModifierSource, error)
	NPCStationIndustryData(ctx context.Context, userAgent string, stationID int64) (sde.NPCStationIndustry, bool, error)
	PublishedTypesInGroups(ctx context.Context, userAgent string, groupIDs []int64) ([]sde.TypeBasic, error)
	PublishedTypesWithEffect(ctx context.Context, userAgent, effectName string) ([]sde.TypeBasic, error)
	RigInfo(ctx context.Context, userAgent string, typeID int64) (sde.Rig, bool, error)
	RoutingSystems(ctx context.Context, userAgent string) ([]sde.RoutingSystem, error)
	StargateConnections(ctx context.Context, userAgent string) ([][2]int64, error)
	Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error)
	TargetFilters(ctx context.Context, userAgent string) (map[int64]sde.TargetFilter, error)
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
	TypeAttributes(ctx context.Context, userAgent string, typeID int64) (map[int64]float64, error)
	TypeBasics(ctx context.Context, userAgent string, ids []int64) (map[int64]sde.TypeBasic, error)
	TypesByName(ctx context.Context, userAgent string, names []string) (map[string]int64, error)
}

// plannerOpener makes the planner's non-generic opener from a handler's
// generic one: C is constrained to satisfy plannerClient, so the compiler
// checks that the client a caller supplies has the calls the planner makes.
func plannerOpener[C plannerClient](open func(ctx context.Context, characterID int32) (C, error)) func(ctx context.Context, characterID int32) (plannerClient, error) {
	return func(ctx context.Context, characterID int32) (plannerClient, error) {
		c, err := open(ctx, characterID)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
}

// plannerPublicOpener is plannerOpener for public clients.
func plannerPublicOpener[C plannerPublicClient](open func() C) func() plannerPublicClient {
	return func() plannerPublicClient { return open() }
}

// fetchSellOrders returns every sell order for a type in a region.
func fetchSellOrders(ctx context.Context, client marketOrdersESI, regionID, typeID int64) ([]market.Order, error) {
	raw, err := esi.AllPages(ctx, func(ctx context.Context, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return listMarketOrders(ctx, client, regionID, "sell", typeID, page)
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
	userAgent     string
	openESI       func(ctx context.Context, characterID int32) (plannerClient, error)
	openPublicESI func() plannerPublicClient
	store         plannerConfig
	sde           plannerSDE

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
	// fallbackMarketCharacter reads a structure hub's market only when
	// neither marketCharacter nor the hub's read_via is set: a character
	// the call names for another job (builder, refiner), who may lack
	// docking access where read_via has it.
	fallbackMarketCharacter int32
}

func newPlanner(userAgent string, openESI func(ctx context.Context, characterID int32) (plannerClient, error), openPublicESI func() plannerPublicClient, store plannerConfig, sdeStore plannerSDE) *planner {
	return &planner{userAgent: userAgent, openESI: openESI, openPublicESI: openPublicESI, store: store, sde: sdeStore,
		recipes: map[int64]*industry.Recipe{}, basics: map[int64]sde.TypeBasic{}, skillsBy: map[int32]map[int64]int{}}
}

func (p *planner) ua() string { return p.userAgent }

// esi opens a character's authenticated ESI client. The opener's error
// already says which character, so it is returned as is.
func (p *planner) esi(ctx context.Context, characterID int32) (plannerClient, error) {
	return p.openESI(ctx, characterID)
}

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
	// A set, not slices.Contains: callers such as list_haul_ships pass one
	// ID per asset, tens of thousands with many repeats.
	var missing []int64
	seen := make(map[int64]bool)
	for _, id := range ids {
		if _, ok := p.basics[id]; !ok && !seen[id] {
			seen[id] = true
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
		maps.Copy(p.basics, got)
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
	public := p.openPublicESI() // one client for every request this call makes
	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(id int64) {
			defer wg.Done()
			defer func() { <-sem }()
			orders, err := fetchSellOrders(ctx, public, regionID, id)
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
	client, err := p.esi(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("skills of %d: %w", characterID, err)
	}
	raw, _, _, err := listCharacterSkills(ctx, client)
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
	slices.Sort(out)
	return out
}

// configUpdater is the config call the delete tools make: change saved
// state under the store's lock.
type configUpdater interface {
	Update(change func(*config.Config) error) ([]config.Change, error)
}

type ProfileNameInput struct {
	Name string `json:"name" jsonschema:"the profile's name"`
}

type DeletedOutput struct {
	Deleted string `json:"deleted"`
}

func deleteProfileHandler(store configUpdater, del func(*config.Config, string) error) mcpsdk.ToolHandlerFor[ProfileNameInput, DeletedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ProfileNameInput) (*mcpsdk.CallToolResult, DeletedOutput, error) {
		if _, err := store.Update(func(cfg *config.Config) error { return del(cfg, in.Name) }); err != nil {
			return nil, DeletedOutput{}, err
		}
		return nil, DeletedOutput{Deleted: in.Name}, nil
	}
}
