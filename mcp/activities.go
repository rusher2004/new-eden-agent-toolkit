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

// OperationSummary is one mercenary tactical operation's bare-ID summary.
type OperationSummary struct {
	ID             string `json:"id"`
	MercenaryDenID int64  `json:"mercenaryDenId"`
}

// listMercenaryTacticalOperations is overridable in tests.
var listMercenaryTacticalOperations = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]OperationSummary, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func listMercenaryTacticalOperationsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListMercenaryTacticalOperationsInput, ListMercenaryTacticalOperationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMercenaryTacticalOperationsInput) (*mcpsdk.CallToolResult, ListMercenaryTacticalOperationsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListMercenaryTacticalOperationsOutput{}, fmt.Errorf("list mercenary tactical operations for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		ops, err := listMercenaryTacticalOperations(ctx, clientID, in.CharacterID, sso.UserAgent(version))
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

// getMercenaryTacticalOperationDetail is overridable in tests.
var getMercenaryTacticalOperationDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, operationID string) (OperationDetailResult, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return OperationDetailResult{}, fmt.Errorf("build ESI client: %w", err)
	}
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

func getMercenaryTacticalOperationHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetMercenaryTacticalOperationInput, GetMercenaryTacticalOperationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMercenaryTacticalOperationInput) (*mcpsdk.CallToolResult, GetMercenaryTacticalOperationOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("get mercenary tactical operation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		result, err := getMercenaryTacticalOperationDetail(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.OperationID)
		if err != nil {
			return nil, GetMercenaryTacticalOperationOutput{}, fmt.Errorf("get mercenary tactical operation detail: %w", err)
		}

		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{result.DungeonTypeID})

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

// listRaidableSkyhooks is overridable in tests. Uses the public client -- RaidableSkyhooks needs
// no character scope at all, so there's no reason to require a live authenticated token just to
// call it (unlike ListRaidableSkyhooksInput's characterId, which is kept purely for the
// existence-check gate below, not for the ESI call itself).
var listRaidableSkyhooks = func(ctx context.Context, userAgent string) ([]SkyhookSummary, error) {
	client := esi.NewPublicClient(userAgent)
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

func listRaidableSkyhooksHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListRaidableSkyhooksInput, ListRaidableSkyhooksOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListRaidableSkyhooksInput) (*mcpsdk.CallToolResult, ListRaidableSkyhooksOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("list raidable skyhooks for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		skyhooks, err := listRaidableSkyhooks(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListRaidableSkyhooksOutput{}, fmt.Errorf("list raidable skyhooks: %w", err)
		}

		systemIDs := make([]int64, len(skyhooks))
		planetIDs := make([]int64, len(skyhooks))
		for i, s := range skyhooks {
			systemIDs[i] = s.SolarSystemID
			planetIDs[i] = s.PlanetID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		planetNames := resolvePlanetNames(ctx, sdeStore, sso.UserAgent(version), planetIDs)
		for i := range skyhooks {
			skyhooks[i].SolarSystemName = systemNames[skyhooks[i].SolarSystemID]
			skyhooks[i].PlanetName = planetNames[skyhooks[i].PlanetID]
		}

		return nil, ListRaidableSkyhooksOutput{Skyhooks: skyhooks}, nil
	}
}
