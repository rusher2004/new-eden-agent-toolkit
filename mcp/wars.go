package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_wars (public) ---

var listWars = func(ctx context.Context, userAgent string, maxWarID int64) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.Wars(ctx, maxWarID)
}

type ListWarsInput struct {
	MaxWarID int64 `json:"maxWarId,omitempty" jsonschema:"only return wars with an ID less than this; omit to get the most recent wars"`
}
type ListWarsOutput struct {
	WarIDs []int64 `json:"warIds"`
}

func listWarsHandler(version string) mcpsdk.ToolHandlerFor[ListWarsInput, ListWarsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListWarsInput) (*mcpsdk.CallToolResult, ListWarsOutput, error) {
		ids, err := listWars(ctx, sso.UserAgent(version), in.MaxWarID)
		if err != nil {
			return nil, ListWarsOutput{}, fmt.Errorf("list wars: %w", err)
		}
		return nil, ListWarsOutput{WarIDs: ids}, nil
	}
}

// --- get_war (public) ---

// WarParty flattens ESI's Aggressor/Defender shapes -- both are the identical
// {allianceId, corporationId, iskDestroyed, shipsKilled} shape.
type WarParty struct {
	AllianceID    *int64  `json:"allianceId,omitempty"`
	CorporationID *int64  `json:"corporationId,omitempty"`
	IskDestroyed  float64 `json:"iskDestroyed"`
	ShipsKilled   int64   `json:"shipsKilled"`
}

type WarAlly struct {
	AllianceID    *int64 `json:"allianceId,omitempty"`
	CorporationID *int64 `json:"corporationId,omitempty"`
}

type War struct {
	WarID         int64      `json:"warId"`
	Aggressor     WarParty   `json:"aggressor"`
	Defender      WarParty   `json:"defender"`
	Allies        []WarAlly  `json:"allies,omitempty"`
	Declared      time.Time  `json:"declared"`
	Finished      *time.Time `json:"finished,omitempty"`
	Mutual        bool       `json:"mutual"`
	OpenForAllies bool       `json:"openForAllies"`
	Retracted     *time.Time `json:"retracted,omitempty"`
	Started       *time.Time `json:"started,omitempty"`
}

var getWar = func(ctx context.Context, userAgent string, warID int64) (War, error) {
	client := esi.NewPublicClient(userAgent)
	w, err := client.WarDetail(ctx, warID)
	if err != nil {
		return War{}, err
	}
	out := War{
		WarID: w.Id,
		Aggressor: WarParty{
			AllianceID: w.Aggressor.AllianceId, CorporationID: w.Aggressor.CorporationId,
			IskDestroyed: w.Aggressor.IskDestroyed, ShipsKilled: w.Aggressor.ShipsKilled,
		},
		Defender: WarParty{
			AllianceID: w.Defender.AllianceId, CorporationID: w.Defender.CorporationId,
			IskDestroyed: w.Defender.IskDestroyed, ShipsKilled: w.Defender.ShipsKilled,
		},
		Declared: w.Declared, Finished: w.Finished, Mutual: w.Mutual, OpenForAllies: w.OpenForAllies,
		Retracted: w.Retracted, Started: w.Started,
	}
	if len(w.Allies) > 0 {
		out.Allies = make([]WarAlly, len(w.Allies))
		for i, a := range w.Allies {
			out.Allies[i] = WarAlly{AllianceID: a.AllianceId, CorporationID: a.CorporationId}
		}
	}
	return out, nil
}

type GetWarInput struct {
	WarID int64 `json:"warId" jsonschema:"the war ID to look up"`
}

func getWarHandler(version string) mcpsdk.ToolHandlerFor[GetWarInput, War] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetWarInput) (*mcpsdk.CallToolResult, War, error) {
		out, err := getWar(ctx, sso.UserAgent(version), in.WarID)
		if err != nil {
			return nil, War{}, fmt.Errorf("get war: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_war_killmails (public, paginated) ---

var listWarKillmails = func(ctx context.Context, userAgent string, warID int64, page int32) ([]KillmailRef, int32, error) {
	client := esi.NewPublicClient(userAgent)
	refs, totalPages, err := client.WarKillmails(ctx, warID, page)
	if err != nil {
		return nil, 0, err
	}
	return toKillmailRefs(refs), totalPages, nil
}

type ListWarKillmailsInput struct {
	WarID int64 `json:"warId" jsonschema:"the war ID to look up"`
	Page  int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListWarKillmailsOutput struct {
	Killmails  []KillmailRef `json:"killmails"`
	Page       int32         `json:"page"`
	TotalPages int32         `json:"totalPages"`
}

func listWarKillmailsHandler(version string) mcpsdk.ToolHandlerFor[ListWarKillmailsInput, ListWarKillmailsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListWarKillmailsInput) (*mcpsdk.CallToolResult, ListWarKillmailsOutput, error) {
		killmails, totalPages, err := listWarKillmails(ctx, sso.UserAgent(version), in.WarID, in.Page)
		if err != nil {
			return nil, ListWarKillmailsOutput{}, fmt.Errorf("list war killmails: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListWarKillmailsOutput{Killmails: killmails, Page: page, TotalPages: totalPages}, nil
	}
}
