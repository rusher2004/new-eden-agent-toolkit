package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// listAlliances is overridable in tests so no real network call runs.
var listAlliances = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.Alliances(ctx)
}

// ListAlliancesInput is list_alliances's tool input. It takes no fields at all — this
// endpoint is fully public and returns every alliance ID in the game.
type ListAlliancesInput struct{}

// ListAlliancesOutput is list_alliances's tool result.
type ListAlliancesOutput struct {
	AllianceIDs []int64 `json:"allianceIds"`
}

func listAlliancesHandler(version string) mcpsdk.ToolHandlerFor[ListAlliancesInput, ListAlliancesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAlliancesInput) (*mcpsdk.CallToolResult, ListAlliancesOutput, error) {
		ids, err := listAlliances(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListAlliancesOutput{}, fmt.Errorf("list alliances: %w", err)
		}
		return nil, ListAlliancesOutput{AllianceIDs: ids}, nil
	}
}

// AllianceDetailResult is the raw mapped detail returned by getAllianceDetail, before the
// handler adds the allianceID it already knows from the request.
type AllianceDetailResult struct {
	Name                  string
	Ticker                string
	CreatorID             int64
	CreatorCorporationID  int64
	DateFounded           time.Time
	ExecutorCorporationID *int64
	FactionID             *int64
	FactionName           string
}

// getAllianceDetail is overridable in tests.
var getAllianceDetail = func(ctx context.Context, userAgent string, allianceID int64) (AllianceDetailResult, error) {
	client := esi.NewPublicClient(userAgent)
	detail, err := client.AllianceDetail(ctx, allianceID)
	if err != nil {
		return AllianceDetailResult{}, err
	}
	return AllianceDetailResult{
		Name:                  detail.Name,
		Ticker:                detail.Ticker,
		CreatorID:             detail.CreatorId,
		CreatorCorporationID:  detail.CreatorCorporationId,
		DateFounded:           detail.DateFounded,
		ExecutorCorporationID: detail.ExecutorCorporationId,
		FactionID:             detail.FactionId,
	}, nil
}

// GetAllianceInput is get_alliance's tool input.
type GetAllianceInput struct {
	AllianceID int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}

// GetAllianceOutput is get_alliance's tool result.
type GetAllianceOutput struct {
	AllianceID            int64     `json:"allianceId"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
	CreatorID             int64     `json:"creatorId"`
	CreatorCorporationID  int64     `json:"creatorCorporationId"`
	DateFounded           time.Time `json:"dateFounded"`
	ExecutorCorporationID *int64    `json:"executorCorporationId,omitempty"`
	FactionID             *int64    `json:"factionId,omitempty"`
	FactionName           string    `json:"factionName,omitempty"`
}

func getAllianceHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetAllianceInput, GetAllianceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAllianceInput) (*mcpsdk.CallToolResult, GetAllianceOutput, error) {
		result, err := getAllianceDetail(ctx, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, GetAllianceOutput{}, fmt.Errorf("get alliance detail: %w", err)
		}
		if result.FactionID != nil {
			names := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), []int64{*result.FactionID})
			result.FactionName = names[*result.FactionID]
		}
		return nil, GetAllianceOutput{
			AllianceID:            in.AllianceID,
			Name:                  result.Name,
			Ticker:                result.Ticker,
			CreatorID:             result.CreatorID,
			CreatorCorporationID:  result.CreatorCorporationID,
			DateFounded:           result.DateFounded,
			ExecutorCorporationID: result.ExecutorCorporationID,
			FactionID:             result.FactionID,
			FactionName:           result.FactionName,
		}, nil
	}
}

// listAllianceCorporations is overridable in tests.
var listAllianceCorporations = func(ctx context.Context, userAgent string, allianceID int64) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.AllianceCorporations(ctx, allianceID)
}

// ListAllianceCorporationsInput is list_alliance_corporations's tool input.
type ListAllianceCorporationsInput struct {
	AllianceID int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}

// ListAllianceCorporationsOutput is list_alliance_corporations's tool result.
type ListAllianceCorporationsOutput struct {
	AllianceID     int64   `json:"allianceId"`
	CorporationIDs []int64 `json:"corporationIds"`
}

func listAllianceCorporationsHandler(version string) mcpsdk.ToolHandlerFor[ListAllianceCorporationsInput, ListAllianceCorporationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAllianceCorporationsInput) (*mcpsdk.CallToolResult, ListAllianceCorporationsOutput, error) {
		ids, err := listAllianceCorporations(ctx, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, ListAllianceCorporationsOutput{}, fmt.Errorf("list alliance corporations: %w", err)
		}
		return nil, ListAllianceCorporationsOutput{AllianceID: in.AllianceID, CorporationIDs: ids}, nil
	}
}

// AllianceIconsResult is the raw mapped result returned by getAllianceIcons, before the
// handler adds the allianceID it already knows from the request.
type AllianceIconsResult struct {
	Px128x128 *string
	Px64x64   *string
}

// getAllianceIcons is overridable in tests.
var getAllianceIcons = func(ctx context.Context, userAgent string, allianceID int64) (AllianceIconsResult, error) {
	client := esi.NewPublicClient(userAgent)
	icons, err := client.AllianceIcons(ctx, allianceID)
	if err != nil {
		return AllianceIconsResult{}, err
	}
	return AllianceIconsResult{Px128x128: icons.Px128x128, Px64x64: icons.Px64x64}, nil
}

// GetAllianceIconsInput is get_alliance_icons's tool input.
type GetAllianceIconsInput struct {
	AllianceID int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}

// GetAllianceIconsOutput is get_alliance_icons's tool result.
type GetAllianceIconsOutput struct {
	AllianceID int64   `json:"allianceId"`
	Px128x128  *string `json:"px128x128,omitempty"`
	Px64x64    *string `json:"px64x64,omitempty"`
}

func getAllianceIconsHandler(version string) mcpsdk.ToolHandlerFor[GetAllianceIconsInput, GetAllianceIconsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAllianceIconsInput) (*mcpsdk.CallToolResult, GetAllianceIconsOutput, error) {
		result, err := getAllianceIcons(ctx, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, GetAllianceIconsOutput{}, fmt.Errorf("get alliance icons: %w", err)
		}
		return nil, GetAllianceIconsOutput{
			AllianceID: in.AllianceID,
			Px128x128:  result.Px128x128,
			Px64x64:    result.Px64x64,
		}, nil
	}
}
