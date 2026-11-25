package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// operationsESI is the ESI calls the mercenary tactical operation tools make.
type operationsESI interface {
	MercenaryTacticalOperations(ctx context.Context) ([]esiapi.CharactersMercenaryTacticalOperationsListingOperation, error)
	MercenaryTacticalOperationDetail(ctx context.Context, operationID string) (*esiapi.CharactersMercenaryTacticalOperationsDetail, error)
}

// skyhooksESI is the ESI call list_raidable_skyhooks makes; it is public.
type skyhooksESI interface {
	RaidableSkyhooks(ctx context.Context) ([]esiapi.SkyhooksRaidableVulnerableskyhook, error)
}

// skyhookNameCache is the SDE lookups list_raidable_skyhooks makes: system and planet names.
type skyhookNameCache interface {
	systemNameCache
	planetNameCache
}

// OperationSummary is one mercenary tactical operation's bare-ID summary.
type OperationSummary struct {
	ID             string `json:"id"`
	MercenaryDenID int64  `json:"mercenaryDenId"`
}

func listMercenaryTacticalOperations(ctx context.Context, client operationsESI) ([]OperationSummary, error) {
	ops, err := client.MercenaryTacticalOperations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OperationSummary, len(ops))
	for i, o := range ops {
		out[i] = OperationSummary{ID: o.Id, MercenaryDenID: o.MercenaryDenId}
	}
	return out, nil
}

// ListMercenaryTacticalOperationsInput is list_mercenary_tactical_operations's tool input.
type ListMercenaryTacticalOperationsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListMercenaryTacticalOperationsOutput is list_mercenary_tactical_operations's tool result.
type ListMercenaryTacticalOperationsOutput struct {
	CharacterID int32              `json:"characterId"`
	Operations  []OperationSummary `json:"operations"`
}

// AddListMercenaryTacticalOperations registers list_mercenary_tactical_operations. open builds the ESI client; any client with the
// operationsESI methods will do.
func AddListMercenaryTacticalOperations[C operationsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_mercenary_tactical_operations",
		Description: "List a logged-in EVE Online character's mercenary tactical operations.",
	}, listMercenaryTacticalOperationsHandler(open, store))
}

func listMercenaryTacticalOperationsHandler[C operationsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListMercenaryTacticalOperationsInput, ListMercenaryTacticalOperationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMercenaryTacticalOperationsInput) (*mcpsdk.CallToolResult, ListMercenaryTacticalOperationsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListMercenaryTacticalOperationsOutput{}, err
		}
		if !found {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("list mercenary tactical operations for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		ops, err := listMercenaryTacticalOperations(ctx, client)
		if err != nil {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("list mercenary tactical operations: %w", err)
		}
		return nil, ListMercenaryTacticalOperationsOutput{CharacterID: in.CharacterID, Operations: ops}, nil
	}
}

// OperationDetailResult is the raw mapped detail returned by
// getMercenaryTacticalOperationDetail, before the handler adds the
// characterID/operationID it already knows from the request.
type OperationDetailResult struct {
	DungeonTypeID  int64
	MercenaryDenID int64
	State          string
	Expires        time.Time
}

func getMercenaryTacticalOperationDetail(ctx context.Context, client operationsESI, operationID string) (OperationDetailResult, error) {
	detail, err := client.MercenaryTacticalOperationDetail(ctx, operationID)
	if err != nil {
		return OperationDetailResult{}, err
	}
	return OperationDetailResult{
		DungeonTypeID:  detail.DungeonTypeId,
		MercenaryDenID: detail.MercenaryDenId,
		State:          detail.State,
		Expires:        detail.Expires,
	}, nil
}

// GetMercenaryTacticalOperationInput is get_mercenary_tactical_operation's tool input.
type GetMercenaryTacticalOperationInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check"`
	OperationID string `json:"operationId" jsonschema:"the operation ID, from list_mercenary_tactical_operations"`
}

// GetMercenaryTacticalOperationOutput is get_mercenary_tactical_operation's tool result.
type GetMercenaryTacticalOperationOutput struct {
	CharacterID     int32     `json:"characterId"`
	OperationID     string    `json:"operationId"`
	DungeonTypeID   int64     `json:"dungeonTypeId"`
	DungeonTypeName string    `json:"dungeonTypeName,omitempty"`
	MercenaryDenID  int64     `json:"mercenaryDenId"`
	State           string    `json:"state"`
	Expires         time.Time `json:"expires"`
}

// AddGetMercenaryTacticalOperation registers get_mercenary_tactical_operation. open builds the ESI client; any client with the
// operationsESI methods will do.
func AddGetMercenaryTacticalOperation[C operationsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_mercenary_tactical_operation",
		Description: "Get detail (dungeon type, mercenary den, state, expiry) for one mercenary tactical operation.",
	}, getMercenaryTacticalOperationHandler(userAgent, open, store, sdeStore))
}

func getMercenaryTacticalOperationHandler[C operationsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[GetMercenaryTacticalOperationInput, GetMercenaryTacticalOperationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMercenaryTacticalOperationInput) (*mcpsdk.CallToolResult, GetMercenaryTacticalOperationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetMercenaryTacticalOperationOutput{}, err
		}
		if !found {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("get mercenary tactical operation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		result, err := getMercenaryTacticalOperationDetail(ctx, client, in.OperationID)
		if err != nil {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("get mercenary tactical operation detail: %w", err)
		}

		names := resolveTypeNames(ctx, sdeStore, userAgent, []int64{result.DungeonTypeID})

		return nil, GetMercenaryTacticalOperationOutput{
			CharacterID:     in.CharacterID,
			OperationID:     in.OperationID,
			DungeonTypeID:   result.DungeonTypeID,
			DungeonTypeName: names[result.DungeonTypeID],
			MercenaryDenID:  result.MercenaryDenID,
			State:           result.State,
			Expires:         result.Expires,
		}, nil
	}
}

// SkyhookSummary is one upcoming raidable skyhook.
type SkyhookSummary struct {
	PlanetID                int64     `json:"planetId"`
	PlanetName              string    `json:"planetName,omitempty"`
	SolarSystemID           int64     `json:"solarSystemId"`
	SolarSystemName         string    `json:"solarSystemName,omitempty"`
	TheftVulnerabilityStart time.Time `json:"theftVulnerabilityStart"`
	TheftVulnerabilityEnd   time.Time `json:"theftVulnerabilityEnd"`
}

// listRaidableSkyhooks uses the public client -- RaidableSkyhooks needs no character scope at
// all, so there's no reason to require a live authenticated token just to call it (unlike
// ListRaidableSkyhooksInput's characterId, which is kept purely for the existence-check gate in
// the handler, not for the ESI call itself).
func listRaidableSkyhooks(ctx context.Context, client skyhooksESI) ([]SkyhookSummary, error) {
	skyhooks, err := client.RaidableSkyhooks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SkyhookSummary, len(skyhooks))
	for i, s := range skyhooks {
		out[i] = SkyhookSummary{
			PlanetID:                s.PlanetId,
			SolarSystemID:           s.SolarSystemId,
			TheftVulnerabilityStart: s.TheftVulnerability.Start,
			TheftVulnerabilityEnd:   s.TheftVulnerability.End,
		}
	}
	return out, nil
}

// ListRaidableSkyhooksInput is list_raidable_skyhooks's tool input. It still takes a
// characterId, purely for the existence check — the underlying ESI call needs no character
// scope at all (RaidableSkyhooks is public data).
type ListRaidableSkyhooksInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListRaidableSkyhooksOutput is list_raidable_skyhooks's tool result.
type ListRaidableSkyhooksOutput struct {
	Skyhooks []SkyhookSummary `json:"skyhooks"`
}

// AddListRaidableSkyhooks registers list_raidable_skyhooks. open builds the ESI client; any client with the
// skyhooksESI methods will do.
func AddListRaidableSkyhooks[C skyhooksESI](s *mcpsdk.Server, userAgent string, open func() C, store configLoader, sdeStore skyhookNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_raidable_skyhooks",
		Description: "List upcoming raidable skyhooks (public data, no character scope required).",
	}, listRaidableSkyhooksHandler(userAgent, open, store, sdeStore))
}

func listRaidableSkyhooksHandler[C skyhooksESI](userAgent string, open func() C, store configLoader, sdeStore skyhookNameCache) mcpsdk.ToolHandlerFor[ListRaidableSkyhooksInput, ListRaidableSkyhooksOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListRaidableSkyhooksInput) (*mcpsdk.CallToolResult, ListRaidableSkyhooksOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListRaidableSkyhooksOutput{}, err
		}
		if !found {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("list raidable skyhooks for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		skyhooks, err := listRaidableSkyhooks(ctx, open())
		if err != nil {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("list raidable skyhooks: %w", err)
		}

		systemIDs := make([]int64, len(skyhooks))
		planetIDs := make([]int64, len(skyhooks))
		for i, s := range skyhooks {
			systemIDs[i] = s.SolarSystemID
			planetIDs[i] = s.PlanetID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		planetNames := resolvePlanetNames(ctx, sdeStore, userAgent, planetIDs)
		for i := range skyhooks {
			skyhooks[i].SolarSystemName = systemNames[skyhooks[i].SolarSystemID]
			skyhooks[i].PlanetName = planetNames[skyhooks[i].PlanetID]
		}

		return nil, ListRaidableSkyhooksOutput{Skyhooks: skyhooks}, nil
	}
}
