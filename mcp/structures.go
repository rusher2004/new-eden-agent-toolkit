package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// mercenaryDensESI is the ESI call list_character_mercenary_dens makes.
type mercenaryDensESI interface {
	CharacterMercenaryDens(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error)
}

// mercenaryDenESI is the ESI call get_character_mercenary_den makes.
type mercenaryDenESI interface {
	CharacterMercenaryDenDetail(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error)
}

// mercenaryDenSDE is the SDE name lookups get_character_mercenary_den makes.
type mercenaryDenSDE interface {
	typeNameCache
	planetNameCache
}

// corporationSkyhooksESI is the ESI call list_corporation_skyhooks makes.
type corporationSkyhooksESI interface {
	CorporationSkyhooks(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error)
}

// skyhookESI is the ESI call get_corporation_skyhook makes.
type skyhookESI interface {
	CorporationSkyhookDetail(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error)
}

// skyhookSDE is the SDE name lookups get_corporation_skyhook makes.
type skyhookSDE interface {
	typeNameCache
	planetNameCache
}

// sovereigntyHubsESI is the ESI call list_corporation_sovereignty_hubs makes.
type sovereigntyHubsESI interface {
	CorporationSovereigntyHubs(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error)
}

// sovereigntyHubESI is the ESI call get_corporation_sovereignty_hub makes.
type sovereigntyHubESI interface {
	CorporationSovereigntyHubDetail(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error)
}

// sovereigntyHubSDE is the SDE name lookups get_corporation_sovereignty_hub makes.
type sovereigntyHubSDE interface {
	typeNameCache
	systemNameCache
}

// --- list_character_mercenary_dens (private, "my own character") ---

type MercenaryDen struct {
	ID         int64  `json:"id"`
	PlanetID   int64  `json:"planetId"`
	PlanetName string `json:"planetName,omitempty"`
}

func listCharacterMercenaryDens(ctx context.Context, client mercenaryDensESI) ([]MercenaryDen, error) {
	dens, err := client.CharacterMercenaryDens(ctx)
	if err != nil {
		return nil, fmt.Errorf("read mercenary dens: %w", err)
	}
	out := make([]MercenaryDen, len(dens))
	for i, d := range dens {
		out[i] = MercenaryDen{ID: d.Id, PlanetID: d.PlanetId}
	}
	return out, nil
}

type ListCharacterMercenaryDensInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterMercenaryDensOutput struct {
	CharacterID int32          `json:"characterId"`
	Dens        []MercenaryDen `json:"dens"`
}

// AddListCharacterMercenaryDens registers list_character_mercenary_dens. open builds the ESI client; any client with the
// mercenaryDensESI methods will do.
func AddListCharacterMercenaryDens[C mercenaryDensESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore planetNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_mercenary_dens",
		Description: "List the given character's mercenary dens (includes resolved planet name).",
	}, listCharacterMercenaryDensHandler(userAgent, open, store, sdeStore))
}

func listCharacterMercenaryDensHandler[C mercenaryDensESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore planetNameCache) mcpsdk.ToolHandlerFor[ListCharacterMercenaryDensInput, ListCharacterMercenaryDensOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMercenaryDensInput) (*mcpsdk.CallToolResult, ListCharacterMercenaryDensOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMercenaryDensOutput{}, err
		}
		if !found {
			return nil, ListCharacterMercenaryDensOutput{}, fmt.Errorf("list character mercenary dens for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMercenaryDensOutput{}, fmt.Errorf("list character mercenary dens: build ESI client: %w", err)
		}
		dens, err := listCharacterMercenaryDens(ctx, client)
		if err != nil {
			return nil, ListCharacterMercenaryDensOutput{}, fmt.Errorf("list character mercenary dens: %w", err)
		}
		ids := make([]int64, len(dens))
		for i, d := range dens {
			ids[i] = d.PlanetID
		}
		names := resolvePlanetNames(ctx, sdeStore, userAgent, ids)
		for i := range dens {
			dens[i].PlanetName = names[dens[i].PlanetID]
		}
		return nil, ListCharacterMercenaryDensOutput{CharacterID: in.CharacterID, Dens: dens}, nil
	}
}

// --- get_character_mercenary_den (private, "my own character") ---

type MercenaryDenEvolutionStat struct {
	Amount int64  `json:"amount"`
	Level  string `json:"level"`
}

type MercenaryDenEvolution struct {
	Anarchy     MercenaryDenEvolutionStat `json:"anarchy"`
	Development MercenaryDenEvolutionStat `json:"development"`
}

type MercenaryDenSkyhookRef struct {
	CorporationID int64  `json:"corporationId"`
	ID            int64  `json:"id"`
	PlanetID      int64  `json:"planetId"`
	PlanetName    string `json:"planetName,omitempty"`
}

type MercenaryDenDetail struct {
	ID               int64                  `json:"id"`
	TypeID           int64                  `json:"typeId"`
	TypeName         string                 `json:"typeName,omitempty"`
	State            string                 `json:"state"`
	Evolution        MercenaryDenEvolution  `json:"evolution"`
	InfomorphsAmount int64                  `json:"infomorphsAmount"`
	ReinforcementEnd *time.Time             `json:"reinforcementEnd,omitempty"`
	Skyhook          MercenaryDenSkyhookRef `json:"skyhook"`
}

func getCharacterMercenaryDen(ctx context.Context, client mercenaryDenESI, mercenaryDenID int64) (MercenaryDenDetail, error) {
	d, err := client.CharacterMercenaryDenDetail(ctx, mercenaryDenID)
	if err != nil {
		return MercenaryDenDetail{}, fmt.Errorf("read mercenary den %d: %w", mercenaryDenID, err)
	}
	out := MercenaryDenDetail{
		ID:     d.Id,
		TypeID: d.TypeId,
		State:  d.State,
		Evolution: MercenaryDenEvolution{
			Anarchy:     MercenaryDenEvolutionStat{Amount: d.Evolution.Anarchy.Amount, Level: d.Evolution.Anarchy.Level},
			Development: MercenaryDenEvolutionStat{Amount: d.Evolution.Development.Amount, Level: d.Evolution.Development.Level},
		},
		InfomorphsAmount: d.Infomorphs.Amount,
		Skyhook: MercenaryDenSkyhookRef{
			CorporationID: d.Skyhook.CorporationId,
			ID:            d.Skyhook.Id,
			PlanetID:      d.Skyhook.PlanetId,
		},
	}
	if d.ReinforcementTimer != nil {
		out.ReinforcementEnd = &d.ReinforcementTimer.End
	}
	return out, nil
}

type GetCharacterMercenaryDenInput struct {
	CharacterID    int32 `json:"characterId" jsonschema:"the character ID to check"`
	MercenaryDenID int64 `json:"mercenaryDenId" jsonschema:"the mercenary den ID to look up"`
}

// AddGetCharacterMercenaryDen registers get_character_mercenary_den. open builds the ESI client; any client with the
// mercenaryDenESI methods will do.
func AddGetCharacterMercenaryDen[C mercenaryDenESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore mercenaryDenSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_mercenary_den",
		Description: "Get full detail on one of the given character's mercenary dens: evolution stats, infomorph count, reinforcement timer, and attached skyhook.",
	}, getCharacterMercenaryDenHandler(userAgent, open, store, sdeStore))
}

func getCharacterMercenaryDenHandler[C mercenaryDenESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore mercenaryDenSDE) mcpsdk.ToolHandlerFor[GetCharacterMercenaryDenInput, MercenaryDenDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterMercenaryDenInput) (*mcpsdk.CallToolResult, MercenaryDenDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, MercenaryDenDetail{}, err
		}
		if !found {
			return nil, MercenaryDenDetail{}, fmt.Errorf("get character mercenary den for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, MercenaryDenDetail{}, fmt.Errorf("get character mercenary den: build ESI client: %w", err)
		}
		detail, err := getCharacterMercenaryDen(ctx, client, in.MercenaryDenID)
		if err != nil {
			return nil, MercenaryDenDetail{}, fmt.Errorf("get character mercenary den: %w", err)
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, []int64{detail.TypeID})
		detail.TypeName = names[detail.TypeID]
		planetNames := resolvePlanetNames(ctx, sdeStore, userAgent, []int64{detail.Skyhook.PlanetID})
		detail.Skyhook.PlanetName = planetNames[detail.Skyhook.PlanetID]
		return nil, detail, nil
	}
}

// --- list_corporation_skyhooks (private, corp-scoped) ---

type Skyhook struct {
	ID         int64  `json:"id"`
	PlanetID   int64  `json:"planetId"`
	PlanetName string `json:"planetName,omitempty"`
}

func listCorporationSkyhooks(ctx context.Context, client corporationSkyhooksESI, corporationID int64) ([]Skyhook, error) {
	hooks, err := client.CorporationSkyhooks(ctx, corporationID)
	if err != nil {
		return nil, fmt.Errorf("read skyhooks of corporation %d: %w", corporationID, err)
	}
	out := make([]Skyhook, len(hooks))
	for i, h := range hooks {
		out[i] = Skyhook{ID: h.Id, PlanetID: h.PlanetId}
	}
	return out, nil
}

type ListCorporationSkyhooksInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationSkyhooksOutput struct {
	CharacterID   int32     `json:"characterId"`
	CorporationID int64     `json:"corporationId"`
	Skyhooks      []Skyhook `json:"skyhooks"`
}

// AddListCorporationSkyhooks registers list_corporation_skyhooks. open builds the ESI client; any client with the
// corporationSkyhooksESI methods will do.
func AddListCorporationSkyhooks[C corporationSkyhooksESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore planetNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_skyhooks",
		Description: "List the given corporation's skyhooks (includes resolved planet name).",
	}, listCorporationSkyhooksHandler(userAgent, open, store, sdeStore))
}

func listCorporationSkyhooksHandler[C corporationSkyhooksESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore planetNameCache) mcpsdk.ToolHandlerFor[ListCorporationSkyhooksInput, ListCorporationSkyhooksOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationSkyhooksInput) (*mcpsdk.CallToolResult, ListCorporationSkyhooksOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSkyhooksOutput{}, err
		}
		if !found {
			return nil, ListCorporationSkyhooksOutput{}, fmt.Errorf("list corporation skyhooks for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSkyhooksOutput{}, fmt.Errorf("list corporation skyhooks: build ESI client: %w", err)
		}
		hooks, err := listCorporationSkyhooks(ctx, client, in.CorporationID)
		if err != nil {
			return nil, ListCorporationSkyhooksOutput{}, fmt.Errorf("list corporation skyhooks: %w", err)
		}
		ids := make([]int64, len(hooks))
		for i, h := range hooks {
			ids[i] = h.PlanetID
		}
		names := resolvePlanetNames(ctx, sdeStore, userAgent, ids)
		for i := range hooks {
			hooks[i].PlanetName = names[hooks[i].PlanetID]
		}
		return nil, ListCorporationSkyhooksOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Skyhooks: hooks}, nil
	}
}

// --- get_corporation_skyhook (private, corp-scoped) ---

type SkyhookReagent struct {
	LastCycle      time.Time `json:"lastCycle"`
	SecuredStock   int64     `json:"securedStock"`
	TypeID         int64     `json:"typeId"`
	TypeName       string    `json:"typeName,omitempty"`
	UnsecuredStock int64     `json:"unsecuredStock"`
}

type SkyhookDetail struct {
	EffectiveWorkforce      *int64           `json:"effectiveWorkforce,omitempty"`
	ID                      int64            `json:"id"`
	PlanetID                int64            `json:"planetId"`
	PlanetName              string           `json:"planetName,omitempty"`
	IsActive                bool             `json:"isActive"`
	Reagents                []SkyhookReagent `json:"reagents,omitempty"`
	ReinforcementEnd        *time.Time       `json:"reinforcementEnd,omitempty"`
	State                   string           `json:"state"`
	TheftVulnerabilityStart *time.Time       `json:"theftVulnerabilityStart,omitempty"`
	TheftVulnerabilityEnd   *time.Time       `json:"theftVulnerabilityEnd,omitempty"`
}

func getCorporationSkyhook(ctx context.Context, client skyhookESI, corporationID, skyhookID int64) (SkyhookDetail, error) {
	d, err := client.CorporationSkyhookDetail(ctx, corporationID, skyhookID)
	if err != nil {
		return SkyhookDetail{}, fmt.Errorf("read skyhook %d of corporation %d: %w", skyhookID, corporationID, err)
	}
	out := SkyhookDetail{
		EffectiveWorkforce: d.EffectiveWorkforce,
		ID:                 d.Id,
		PlanetID:           d.PlanetId,
		IsActive:           d.IsActive,
		State:              d.State,
	}
	if len(d.Reagents) > 0 {
		out.Reagents = make([]SkyhookReagent, len(d.Reagents))
		for i, r := range d.Reagents {
			out.Reagents[i] = SkyhookReagent{
				LastCycle: r.LastCycle, SecuredStock: r.SecuredStock, TypeID: r.TypeId, UnsecuredStock: r.UnsecuredStock,
			}
		}
	}
	if d.ReinforcementTimer != nil {
		out.ReinforcementEnd = &d.ReinforcementTimer.End
	}
	if d.TheftVulnerability != nil {
		out.TheftVulnerabilityStart = &d.TheftVulnerability.Start
		out.TheftVulnerabilityEnd = &d.TheftVulnerability.End
	}
	return out, nil
}

type GetCorporationSkyhookInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	SkyhookID     int64 `json:"skyhookId" jsonschema:"the skyhook ID to look up"`
}

// AddGetCorporationSkyhook registers get_corporation_skyhook. open builds the ESI client; any client with the
// skyhookESI methods will do.
func AddGetCorporationSkyhook[C skyhookESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore skyhookSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_corporation_skyhook",
		Description: "Get full detail on one of the given corporation's skyhooks: workforce, reagent stock, active/reinforcement state, and theft vulnerability window.",
	}, getCorporationSkyhookHandler(userAgent, open, store, sdeStore))
}

func getCorporationSkyhookHandler[C skyhookESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore skyhookSDE) mcpsdk.ToolHandlerFor[GetCorporationSkyhookInput, SkyhookDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationSkyhookInput) (*mcpsdk.CallToolResult, SkyhookDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SkyhookDetail{}, err
		}
		if !found {
			return nil, SkyhookDetail{}, fmt.Errorf("get corporation skyhook for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, SkyhookDetail{}, fmt.Errorf("get corporation skyhook: build ESI client: %w", err)
		}
		detail, err := getCorporationSkyhook(ctx, client, in.CorporationID, in.SkyhookID)
		if err != nil {
			return nil, SkyhookDetail{}, fmt.Errorf("get corporation skyhook: %w", err)
		}
		ids := make([]int64, len(detail.Reagents))
		for i, r := range detail.Reagents {
			ids[i] = r.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range detail.Reagents {
			detail.Reagents[i].TypeName = names[detail.Reagents[i].TypeID]
		}
		planetNames := resolvePlanetNames(ctx, sdeStore, userAgent, []int64{detail.PlanetID})
		detail.PlanetName = planetNames[detail.PlanetID]
		return nil, detail, nil
	}
}

// --- list_corporation_sovereignty_hubs (private, corp-scoped) ---

type StructureSovereigntyHub struct {
	ID              int64  `json:"id"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
}

func listCorporationSovereigntyHubs(ctx context.Context, client sovereigntyHubsESI, corporationID int64) ([]StructureSovereigntyHub, error) {
	hubs, err := client.CorporationSovereigntyHubs(ctx, corporationID)
	if err != nil {
		return nil, fmt.Errorf("read sovereignty hubs of corporation %d: %w", corporationID, err)
	}
	out := make([]StructureSovereigntyHub, len(hubs))
	for i, h := range hubs {
		out[i] = StructureSovereigntyHub{ID: h.Id, SolarSystemID: h.SolarSystemId}
	}
	return out, nil
}

type ListCorporationSovereigntyHubsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationSovereigntyHubsOutput struct {
	CharacterID   int32                     `json:"characterId"`
	CorporationID int64                     `json:"corporationId"`
	Hubs          []StructureSovereigntyHub `json:"hubs"`
}

// AddListCorporationSovereigntyHubs registers list_corporation_sovereignty_hubs. open builds the ESI client; any client with the
// sovereigntyHubsESI methods will do.
func AddListCorporationSovereigntyHubs[C sovereigntyHubsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore systemNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_sovereignty_hubs",
		Description: "List the given corporation's sovereignty hubs (includes resolved solar system name).",
	}, listCorporationSovereigntyHubsHandler(userAgent, open, store, sdeStore))
}

func listCorporationSovereigntyHubsHandler[C sovereigntyHubsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore systemNameCache) mcpsdk.ToolHandlerFor[ListCorporationSovereigntyHubsInput, ListCorporationSovereigntyHubsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationSovereigntyHubsInput) (*mcpsdk.CallToolResult, ListCorporationSovereigntyHubsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSovereigntyHubsOutput{}, err
		}
		if !found {
			return nil, ListCorporationSovereigntyHubsOutput{}, fmt.Errorf("list corporation sovereignty hubs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSovereigntyHubsOutput{}, fmt.Errorf("list corporation sovereignty hubs: build ESI client: %w", err)
		}
		hubs, err := listCorporationSovereigntyHubs(ctx, client, in.CorporationID)
		if err != nil {
			return nil, ListCorporationSovereigntyHubsOutput{}, fmt.Errorf("list corporation sovereignty hubs: %w", err)
		}
		ids := make([]int64, len(hubs))
		for i, h := range hubs {
			ids[i] = h.SolarSystemID
		}
		names := resolveSystemNames(ctx, sdeStore, userAgent, ids)
		for i := range hubs {
			hubs[i].SolarSystemName = names[hubs[i].SolarSystemID]
		}
		return nil, ListCorporationSovereigntyHubsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Hubs: hubs}, nil
	}
}

// --- get_corporation_sovereignty_hub (private, corp-scoped) ---

type StructureSovereigntyHubReagent struct {
	Amount         int64  `json:"amount"`
	BurningPerHour int64  `json:"burningPerHour"`
	TypeID         int64  `json:"typeId"`
	TypeName       string `json:"typeName,omitempty"`
}

type StructureSovereigntyHubReagentBay struct {
	LastUpdated time.Time                        `json:"lastUpdated"`
	Reagents    []StructureSovereigntyHubReagent `json:"reagents"`
}

type StructureSovereigntyHubResourceLevel struct {
	Allocated int64 `json:"allocated"`
	Available int64 `json:"available"`
}

type StructureSovereigntyHubResources struct {
	Power     StructureSovereigntyHubResourceLevel `json:"power"`
	Workforce StructureSovereigntyHubResourceLevel `json:"workforce"`
}

type StructureSovereigntyHubUpgrade struct {
	PowerState string `json:"powerState"`
	TypeID     int64  `json:"typeId"`
	TypeName   string `json:"typeName,omitempty"`
}

// StructureSovereigntyHubTransportConfig flattens ESI's 3-way oneOf (export/import/transit) for the
// requested workforce transport configuration. Kind is one of "export", "import", "transit".
type StructureSovereigntyHubTransportConfig struct {
	Kind                   string   `json:"kind"`
	ExportAmount           *int64   `json:"exportAmount,omitempty"`
	ExportSolarSystemID    *int64   `json:"exportSolarSystemId,omitempty"`
	ExportSolarSystemName  string   `json:"exportSolarSystemName,omitempty"`
	ImportSolarSystemIDs   []int64  `json:"importSolarSystemIds,omitempty"`
	ImportSolarSystemNames []string `json:"importSolarSystemNames,omitempty"`
	Transit                *bool    `json:"transit,omitempty"`
}

type StructureSovereigntyHubTransportImportSource struct {
	Amount          int64  `json:"amount"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
}

// StructureSovereigntyHubTransportState flattens ESI's 3-way oneOf (export/import/transit) for the actual
// in-progress workforce transport state. Kind is one of "export", "import", "transit". Note this
// is NOT the same shape as StructureSovereigntyHubTransportConfig's import case: state's import sources
// carry an amount, configuration's do not (verified from the generated model source).
type StructureSovereigntyHubTransportState struct {
	Kind                  string                                         `json:"kind"`
	ExportAmount          *int64                                         `json:"exportAmount,omitempty"`
	ExportSolarSystemID   *int64                                         `json:"exportSolarSystemId,omitempty"`
	ExportSolarSystemName string                                         `json:"exportSolarSystemName,omitempty"`
	ImportSources         []StructureSovereigntyHubTransportImportSource `json:"importSources,omitempty"`
	Transit               *bool                                          `json:"transit,omitempty"`
}

type StructureSovereigntyHubTransport struct {
	Configuration StructureSovereigntyHubTransportConfig `json:"configuration"`
	State         StructureSovereigntyHubTransportState  `json:"state"`
}

type StructureSovereigntyHubDetail struct {
	FuelAccessListID         *int64                            `json:"fuelAccessListId,omitempty"`
	ID                       int64                             `json:"id"`
	SolarSystemID            int64                             `json:"solarSystemId"`
	SolarSystemName          string                            `json:"solarSystemName,omitempty"`
	ReagentBay               StructureSovereigntyHubReagentBay `json:"reagentBay"`
	Resources                StructureSovereigntyHubResources  `json:"resources"`
	Upgrades                 []StructureSovereigntyHubUpgrade  `json:"upgrades"`
	VulnerabilityWindowStart *time.Time                        `json:"vulnerabilityWindowStart,omitempty"`
	VulnerabilityWindowEnd   *time.Time                        `json:"vulnerabilityWindowEnd,omitempty"`
	WorkforceTransport       StructureSovereigntyHubTransport  `json:"workforceTransport"`
}

type GetCorporationSovereigntyHubInput struct {
	CharacterID      int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID    int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	SovereigntyHubID int64 `json:"sovereigntyHubId" jsonschema:"the sovereignty hub ID to look up"`
}

func getCorporationSovereigntyHub(ctx context.Context, client sovereigntyHubESI, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
	d, err := client.CorporationSovereigntyHubDetail(ctx, corporationID, sovereigntyHubID)
	if err != nil {
		return StructureSovereigntyHubDetail{}, fmt.Errorf("read sovereignty hub %d of corporation %d: %w", sovereigntyHubID, corporationID, err)
	}
	out := StructureSovereigntyHubDetail{
		FuelAccessListID: d.FuelAccessListId,
		ID:               d.Id,
		SolarSystemID:    d.SolarSystemId,
		ReagentBay: StructureSovereigntyHubReagentBay{
			LastUpdated: d.ReagentBay.LastUpdated,
		},
		Resources: StructureSovereigntyHubResources{
			Power:     StructureSovereigntyHubResourceLevel{Allocated: d.Resources.Power.Allocated, Available: d.Resources.Power.Available},
			Workforce: StructureSovereigntyHubResourceLevel{Allocated: d.Resources.Workforce.Allocated, Available: d.Resources.Workforce.Available},
		},
	}
	out.ReagentBay.Reagents = make([]StructureSovereigntyHubReagent, len(d.ReagentBay.Reagents))
	for i, r := range d.ReagentBay.Reagents {
		out.ReagentBay.Reagents[i] = StructureSovereigntyHubReagent{Amount: r.Amount, BurningPerHour: r.BurningPerHour, TypeID: r.TypeId}
	}
	out.Upgrades = make([]StructureSovereigntyHubUpgrade, len(d.Upgrades))
	for i, u := range d.Upgrades {
		out.Upgrades[i] = StructureSovereigntyHubUpgrade{PowerState: u.PowerState, TypeID: u.TypeId}
	}
	if d.VulnerabilityWindow != nil {
		out.VulnerabilityWindowStart = &d.VulnerabilityWindow.Start
		out.VulnerabilityWindowEnd = &d.VulnerabilityWindow.End
	}

	cfg := d.WorkforceTransport.Configuration
	switch {
	case cfg.Export != nil && cfg.Export.Export != nil:
		out.WorkforceTransport.Configuration = StructureSovereigntyHubTransportConfig{
			Kind: "export", ExportAmount: &cfg.Export.Export.Amount, ExportSolarSystemID: cfg.Export.Export.SolarSystemId,
		}
	case cfg.ModelImport != nil && cfg.ModelImport.Import != nil:
		ids := make([]int64, len(cfg.ModelImport.Import.Sources))
		for i, s := range cfg.ModelImport.Import.Sources {
			ids[i] = s.SolarSystemId
		}
		out.WorkforceTransport.Configuration = StructureSovereigntyHubTransportConfig{Kind: "import", ImportSolarSystemIDs: ids}
	case cfg.Transit != nil:
		out.WorkforceTransport.Configuration = StructureSovereigntyHubTransportConfig{Kind: "transit", Transit: cfg.Transit.Transit.Get()}
	default:
		// None of the 3 known oneOf variants matched -- an unrecognized ESI variant. Surface this
		// distinctly rather than silently leaving Kind as the zero value.
		out.WorkforceTransport.Configuration = StructureSovereigntyHubTransportConfig{Kind: "unknown"}
	}

	st := d.WorkforceTransport.State
	switch {
	case st.Export1 != nil && st.Export1.Export != nil:
		out.WorkforceTransport.State = StructureSovereigntyHubTransportState{
			Kind: "export", ExportAmount: st.Export1.Export.Amount, ExportSolarSystemID: st.Export1.Export.SolarSystemId,
		}
	case st.Import1 != nil && st.Import1.Import != nil:
		sources := make([]StructureSovereigntyHubTransportImportSource, len(st.Import1.Import.Sources))
		for i, s := range st.Import1.Import.Sources {
			sources[i] = StructureSovereigntyHubTransportImportSource{Amount: s.Amount, SolarSystemID: s.SolarSystemId}
		}
		out.WorkforceTransport.State = StructureSovereigntyHubTransportState{Kind: "import", ImportSources: sources}
	case st.Transit1 != nil:
		out.WorkforceTransport.State = StructureSovereigntyHubTransportState{Kind: "transit", Transit: st.Transit1.Transit.Get()}
	default:
		// None of the 3 known oneOf variants matched -- an unrecognized ESI variant. Surface this
		// distinctly rather than silently leaving Kind as the zero value.
		out.WorkforceTransport.State = StructureSovereigntyHubTransportState{Kind: "unknown"}
	}

	return out, nil
}

// AddGetCorporationSovereigntyHub registers get_corporation_sovereignty_hub. open builds the ESI client; any client with the
// sovereigntyHubESI methods will do.
func AddGetCorporationSovereigntyHub[C sovereigntyHubESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore sovereigntyHubSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_corporation_sovereignty_hub",
		Description: "Get full detail on one of the given corporation's sovereignty hubs: reagent bay, power/workforce resources, upgrades, vulnerability window, and workforce transport configuration/state.",
	}, getCorporationSovereigntyHubHandler(userAgent, open, store, sdeStore))
}

func getCorporationSovereigntyHubHandler[C sovereigntyHubESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore sovereigntyHubSDE) mcpsdk.ToolHandlerFor[GetCorporationSovereigntyHubInput, StructureSovereigntyHubDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationSovereigntyHubInput) (*mcpsdk.CallToolResult, StructureSovereigntyHubDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, StructureSovereigntyHubDetail{}, err
		}
		if !found {
			return nil, StructureSovereigntyHubDetail{}, fmt.Errorf("get corporation sovereignty hub for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, StructureSovereigntyHubDetail{}, fmt.Errorf("get corporation sovereignty hub: build ESI client: %w", err)
		}
		detail, err := getCorporationSovereigntyHub(ctx, client, in.CorporationID, in.SovereigntyHubID)
		if err != nil {
			return nil, StructureSovereigntyHubDetail{}, fmt.Errorf("get corporation sovereignty hub: %w", err)
		}
		ids := make([]int64, 0, len(detail.ReagentBay.Reagents)+len(detail.Upgrades))
		for _, r := range detail.ReagentBay.Reagents {
			ids = append(ids, r.TypeID)
		}
		for _, u := range detail.Upgrades {
			ids = append(ids, u.TypeID)
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range detail.ReagentBay.Reagents {
			detail.ReagentBay.Reagents[i].TypeName = names[detail.ReagentBay.Reagents[i].TypeID]
		}
		for i := range detail.Upgrades {
			detail.Upgrades[i].TypeName = names[detail.Upgrades[i].TypeID]
		}

		systemIDs := []int64{detail.SolarSystemID}
		if detail.WorkforceTransport.Configuration.ExportSolarSystemID != nil {
			systemIDs = append(systemIDs, *detail.WorkforceTransport.Configuration.ExportSolarSystemID)
		}
		systemIDs = append(systemIDs, detail.WorkforceTransport.Configuration.ImportSolarSystemIDs...)
		if detail.WorkforceTransport.State.ExportSolarSystemID != nil {
			systemIDs = append(systemIDs, *detail.WorkforceTransport.State.ExportSolarSystemID)
		}
		for _, src := range detail.WorkforceTransport.State.ImportSources {
			systemIDs = append(systemIDs, src.SolarSystemID)
		}
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		detail.SolarSystemName = systemNames[detail.SolarSystemID]
		if detail.WorkforceTransport.Configuration.ExportSolarSystemID != nil {
			detail.WorkforceTransport.Configuration.ExportSolarSystemName = systemNames[*detail.WorkforceTransport.Configuration.ExportSolarSystemID]
		}
		if len(detail.WorkforceTransport.Configuration.ImportSolarSystemIDs) > 0 {
			importNames := make([]string, len(detail.WorkforceTransport.Configuration.ImportSolarSystemIDs))
			var resolved bool
			for i, id := range detail.WorkforceTransport.Configuration.ImportSolarSystemIDs {
				if name, ok := systemNames[id]; ok {
					importNames[i] = name
					resolved = true
				}
			}
			if resolved {
				detail.WorkforceTransport.Configuration.ImportSolarSystemNames = importNames
			}
		}
		if detail.WorkforceTransport.State.ExportSolarSystemID != nil {
			detail.WorkforceTransport.State.ExportSolarSystemName = systemNames[*detail.WorkforceTransport.State.ExportSolarSystemID]
		}
		for i := range detail.WorkforceTransport.State.ImportSources {
			detail.WorkforceTransport.State.ImportSources[i].SolarSystemName = systemNames[detail.WorkforceTransport.State.ImportSources[i].SolarSystemID]
		}

		return nil, detail, nil
	}
}
