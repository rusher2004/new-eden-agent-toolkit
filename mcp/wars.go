package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- list_wars (public) ---

// warsESI is the ESI call list_wars makes.
type warsESI interface {
	Wars(ctx context.Context, maxWarID int64) ([]int64, error)
}

type ListWarsInput struct {
	MaxWarID int64 `json:"maxWarId,omitempty" jsonschema:"only return wars with an ID less than this; omit to get the most recent wars"`
}
type ListWarsOutput struct {
	WarIDs []int64 `json:"warIds"`
}

// AddListWars registers list_wars. open builds the ESI client; any client with the
// warsESI methods will do.
func AddListWars[C warsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_wars",
		Description: "List war IDs, optionally only those older than a given war ID (public data, no character required).",
	}, listWarsHandler(open))
}

func listWarsHandler[C warsESI](open func() C) mcpsdk.ToolHandlerFor[ListWarsInput, ListWarsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListWarsInput) (*mcpsdk.CallToolResult, ListWarsOutput, error) {
		ids, err := open().Wars(ctx, in.MaxWarID)
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

// warDetailESI is the ESI call get_war makes.
type warDetailESI interface {
	WarDetail(ctx context.Context, warID int64) (*esiapi.WarsWarIdGet, error)
}

func getWar(ctx context.Context, client warDetailESI, warID int64) (War, error) {
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

// AddGetWar registers get_war. open builds the ESI client; any client with the
// warDetailESI methods will do.
func AddGetWar[C warDetailESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_war",
		Description: "Get a war's aggressor/defender/allies, declared/started/retracted/finished times, and mutual/open-for-allies flags (public data, no character required).",
	}, getWarHandler(open))
}

func getWarHandler[C warDetailESI](open func() C) mcpsdk.ToolHandlerFor[GetWarInput, War] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetWarInput) (*mcpsdk.CallToolResult, War, error) {
		out, err := getWar(ctx, open(), in.WarID)
		if err != nil {
			return nil, War{}, fmt.Errorf("get war: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_war_killmails (public, paginated) ---

// warKillmailsESI is the ESI call list_war_killmails makes.
type warKillmailsESI interface {
	WarKillmails(ctx context.Context, warID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error)
}

func listWarKillmails(ctx context.Context, client warKillmailsESI, warID int64, page int32) ([]KillmailRef, int32, error) {
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

// AddListWarKillmails registers list_war_killmails. open builds the ESI client; any client with the
// warKillmailsESI methods will do.
func AddListWarKillmails[C warKillmailsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_war_killmails",
		Description: "List killmail references for a war, paginated (public data, no character required).",
	}, listWarKillmailsHandler(open))
}

func listWarKillmailsHandler[C warKillmailsESI](open func() C) mcpsdk.ToolHandlerFor[ListWarKillmailsInput, ListWarKillmailsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListWarKillmailsInput) (*mcpsdk.CallToolResult, ListWarKillmailsOutput, error) {
		killmails, totalPages, err := listWarKillmails(ctx, open(), in.WarID, in.Page)
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
