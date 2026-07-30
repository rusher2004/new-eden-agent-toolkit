package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_character_mercenary_dens (private, "my own character") ---

type MercenaryDen struct {
	ID       int64 `json:"id"`
	PlanetID int64 `json:"planetId"`
}

var listCharacterMercenaryDens = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MercenaryDen, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	dens, err := client.CharacterMercenaryDens(ctx)
	if err != nil {
		return nil, err
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

func listCharacterMercenaryDensHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterMercenaryDensInput, ListCharacterMercenaryDensOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMercenaryDensInput) (*mcpsdk.CallToolResult, ListCharacterMercenaryDensOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMercenaryDensOutput{}, err
		}
		if !found {
			return nil, ListCharacterMercenaryDensOutput{}, fmt.Errorf("list character mercenary dens for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		dens, err := listCharacterMercenaryDens(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterMercenaryDensOutput{}, fmt.Errorf("list character mercenary dens: %w", err)
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
	CorporationID int64 `json:"corporationId"`
	ID            int64 `json:"id"`
	PlanetID      int64 `json:"planetId"`
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

var getCharacterMercenaryDen = func(ctx context.Context, clientID string, characterID int32, userAgent string, mercenaryDenID int64) (MercenaryDenDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return MercenaryDenDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	d, err := client.CharacterMercenaryDenDetail(ctx, mercenaryDenID)
	if err != nil {
		return MercenaryDenDetail{}, err
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

func getCharacterMercenaryDenHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCharacterMercenaryDenInput, MercenaryDenDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterMercenaryDenInput) (*mcpsdk.CallToolResult, MercenaryDenDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, MercenaryDenDetail{}, err
		}
		if !found {
			return nil, MercenaryDenDetail{}, fmt.Errorf("get character mercenary den for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCharacterMercenaryDen(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.MercenaryDenID)
		if err != nil {
			return nil, MercenaryDenDetail{}, fmt.Errorf("get character mercenary den: %w", err)
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{detail.TypeID})
		detail.TypeName = names[detail.TypeID]
		return nil, detail, nil
	}
}

// --- list_corporation_skyhooks (private, corp-scoped) ---

type Skyhook struct {
	ID       int64 `json:"id"`
	PlanetID int64 `json:"planetId"`
}

var listCorporationSkyhooks = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Skyhook, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	hooks, err := client.CorporationSkyhooks(ctx, corporationID)
	if err != nil {
		return nil, err
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

func listCorporationSkyhooksHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationSkyhooksInput, ListCorporationSkyhooksOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationSkyhooksInput) (*mcpsdk.CallToolResult, ListCorporationSkyhooksOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSkyhooksOutput{}, err
		}
		if !found {
			return nil, ListCorporationSkyhooksOutput{}, fmt.Errorf("list corporation skyhooks for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		hooks, err := listCorporationSkyhooks(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationSkyhooksOutput{}, fmt.Errorf("list corporation skyhooks: %w", err)
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
	IsActive                bool             `json:"isActive"`
	Reagents                []SkyhookReagent `json:"reagents,omitempty"`
	ReinforcementEnd        *time.Time       `json:"reinforcementEnd,omitempty"`
	State                   string           `json:"state"`
	TheftVulnerabilityStart *time.Time       `json:"theftVulnerabilityStart,omitempty"`
	TheftVulnerabilityEnd   *time.Time       `json:"theftVulnerabilityEnd,omitempty"`
}

var getCorporationSkyhook = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, skyhookID int64) (SkyhookDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return SkyhookDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	d, err := client.CorporationSkyhookDetail(ctx, corporationID, skyhookID)
	if err != nil {
		return SkyhookDetail{}, err
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

func getCorporationSkyhookHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCorporationSkyhookInput, SkyhookDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationSkyhookInput) (*mcpsdk.CallToolResult, SkyhookDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SkyhookDetail{}, err
		}
		if !found {
			return nil, SkyhookDetail{}, fmt.Errorf("get corporation skyhook for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCorporationSkyhook(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.SkyhookID)
		if err != nil {
			return nil, SkyhookDetail{}, fmt.Errorf("get corporation skyhook: %w", err)
		}
		ids := make([]int64, len(detail.Reagents))
		for i, r := range detail.Reagents {
			ids[i] = r.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range detail.Reagents {
			detail.Reagents[i].TypeName = names[detail.Reagents[i].TypeID]
		}
		return nil, detail, nil
	}
}

// --- list_corporation_sovereignty_hubs (private, corp-scoped) ---

type StructureSovereigntyHub struct {
	ID            int64 `json:"id"`
	SolarSystemID int64 `json:"solarSystemId"`
}

var listCorporationSovereigntyHubs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]StructureSovereigntyHub, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	hubs, err := client.CorporationSovereigntyHubs(ctx, corporationID)
	if err != nil {
		return nil, err
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

func listCorporationSovereigntyHubsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationSovereigntyHubsInput, ListCorporationSovereigntyHubsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationSovereigntyHubsInput) (*mcpsdk.CallToolResult, ListCorporationSovereigntyHubsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationSovereigntyHubsOutput{}, err
		}
		if !found {
			return nil, ListCorporationSovereigntyHubsOutput{}, fmt.Errorf("list corporation sovereignty hubs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		hubs, err := listCorporationSovereigntyHubs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationSovereigntyHubsOutput{}, fmt.Errorf("list corporation sovereignty hubs: %w", err)
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
	Kind                 string  `json:"kind"`
	ExportAmount         *int64  `json:"exportAmount,omitempty"`
	ExportSolarSystemID  *int64  `json:"exportSolarSystemId,omitempty"`
	ImportSolarSystemIDs []int64 `json:"importSolarSystemIds,omitempty"`
	Transit              *bool   `json:"transit,omitempty"`
}

type StructureSovereigntyHubTransportImportSource struct {
	Amount        int64 `json:"amount"`
	SolarSystemID int64 `json:"solarSystemId"`
}

// StructureSovereigntyHubTransportState flattens ESI's 3-way oneOf (export/import/transit) for the actual
// in-progress workforce transport state. Kind is one of "export", "import", "transit". Note this
// is NOT the same shape as StructureSovereigntyHubTransportConfig's import case: state's import sources
// carry an amount, configuration's do not (verified from the generated model source).
type StructureSovereigntyHubTransportState struct {
	Kind                string                                         `json:"kind"`
	ExportAmount        *int64                                         `json:"exportAmount,omitempty"`
	ExportSolarSystemID *int64                                         `json:"exportSolarSystemId,omitempty"`
	ImportSources       []StructureSovereigntyHubTransportImportSource `json:"importSources,omitempty"`
	Transit             *bool                                          `json:"transit,omitempty"`
}

type StructureSovereigntyHubTransport struct {
	Configuration StructureSovereigntyHubTransportConfig `json:"configuration"`
	State         StructureSovereigntyHubTransportState  `json:"state"`
}

type StructureSovereigntyHubDetail struct {
	FuelAccessListID         *int64                            `json:"fuelAccessListId,omitempty"`
	ID                       int64                             `json:"id"`
	SolarSystemID            int64                             `json:"solarSystemId"`
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

var getCorporationSovereigntyHub = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID, sovereigntyHubID int64) (StructureSovereigntyHubDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return StructureSovereigntyHubDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	d, err := client.CorporationSovereigntyHubDetail(ctx, corporationID, sovereigntyHubID)
	if err != nil {
		return StructureSovereigntyHubDetail{}, err
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

func getCorporationSovereigntyHubHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCorporationSovereigntyHubInput, StructureSovereigntyHubDetail] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationSovereigntyHubInput) (*mcpsdk.CallToolResult, StructureSovereigntyHubDetail, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, StructureSovereigntyHubDetail{}, err
		}
		if !found {
			return nil, StructureSovereigntyHubDetail{}, fmt.Errorf("get corporation sovereignty hub for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCorporationSovereigntyHub(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.SovereigntyHubID)
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
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range detail.ReagentBay.Reagents {
			detail.ReagentBay.Reagents[i].TypeName = names[detail.ReagentBay.Reagents[i].TypeID]
		}
		for i := range detail.Upgrades {
			detail.Upgrades[i].TypeName = names[detail.Upgrades[i].TypeID]
		}
		return nil, detail, nil
	}
}
