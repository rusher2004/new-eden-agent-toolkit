package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Every cursor-paginated tool in this file and in paragon_hub.go documents its limit as "10-100":
// verified against the live 2026-08-18 spec (minimum 10, maximum 100, default 10) and against
// ESI itself, which rejects a limit below 10 with a 422 rather than clamping it.

// --- shared types ---

type MilitaryCampaign struct {
	CampaignID string     `json:"campaignId"`
	State      string     `json:"state"`
	Progress   int64      `json:"progress"`
	Started    *time.Time `json:"started,omitempty"`
	Finished   *time.Time `json:"finished,omitempty"`
}

type CampaignObjectiveParticipants struct {
	Total        int64 `json:"total"`
	Committed    int64 `json:"committed"`
	Contributors int64 `json:"contributors"`
}

type CampaignObjective struct {
	ObjectiveID  string                        `json:"objectiveId"`
	State        string                        `json:"state"`
	Progress     int64                         `json:"progress"`
	Participants CampaignObjectiveParticipants `json:"participants"`
	LastModified time.Time                     `json:"lastModified"`
	Started      *time.Time                    `json:"started,omitempty"`
	Finished     *time.Time                    `json:"finished,omitempty"`
}

func toCampaignObjectiveParticipants(p esiapi.MilitaryCampaignsObjectivesDetailParticipants) CampaignObjectiveParticipants {
	return CampaignObjectiveParticipants{Total: p.Total, Committed: p.Committed, Contributors: p.Contributors}
}

type CharacterCampaignObjective struct {
	ObjectiveID  string    `json:"objectiveId"`
	CampaignID   string    `json:"campaignId"`
	Contributed  int64     `json:"contributed"`
	IsCommitted  bool      `json:"isCommitted"`
	LastModified time.Time `json:"lastModified"`
}

// --- list_military_campaigns (public) ---

// militaryCampaignsESI is the ESI call list_military_campaigns makes.
type militaryCampaignsESI interface {
	MilitaryCampaigns(ctx context.Context) ([]esiapi.MilitaryCampaignsDetailCampaign, error)
}

type ListMilitaryCampaignsInput struct{}
type ListMilitaryCampaignsOutput struct {
	Campaigns []MilitaryCampaign `json:"campaigns"`
}

// AddListMilitaryCampaigns registers list_military_campaigns. open builds the ESI client; any client with the
// militaryCampaignsESI methods will do.
func AddListMilitaryCampaigns[C militaryCampaignsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_military_campaigns",
		Description: "List EVE Online military campaigns with their state, progress, and start/finish times (public data, no character required).",
	}, listMilitaryCampaignsHandler(open))
}

func listMilitaryCampaignsHandler[C militaryCampaignsESI](open func() C) mcpsdk.ToolHandlerFor[ListMilitaryCampaignsInput, ListMilitaryCampaignsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ ListMilitaryCampaignsInput) (*mcpsdk.CallToolResult, ListMilitaryCampaignsOutput, error) {
		client := open()
		campaigns, err := client.MilitaryCampaigns(ctx)
		if err != nil {
			return nil, ListMilitaryCampaignsOutput{}, fmt.Errorf("list military campaigns: %w", err)
		}
		out := make([]MilitaryCampaign, len(campaigns))
		for i, c := range campaigns {
			out[i] = MilitaryCampaign{CampaignID: c.Id, State: c.State, Progress: c.Progress, Started: c.Started, Finished: c.Finished}
		}
		return nil, ListMilitaryCampaignsOutput{Campaigns: out}, nil
	}
}

// --- get_military_campaign (public) ---

// militaryCampaignDetailESI is the ESI call get_military_campaign makes.
type militaryCampaignDetailESI interface {
	MilitaryCampaignDetail(ctx context.Context, campaignID string) (*esiapi.MilitaryCampaignsDetail, error)
}

type GetMilitaryCampaignInput struct {
	CampaignID string `json:"campaignId" jsonschema:"the military campaign ID (a UUID, from list_military_campaigns)"`
}
type GetMilitaryCampaignOutput = MilitaryCampaign

// AddGetMilitaryCampaign registers get_military_campaign. open builds the ESI client; any client with the
// militaryCampaignDetailESI methods will do.
func AddGetMilitaryCampaign[C militaryCampaignDetailESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_military_campaign",
		Description: "Get one military campaign's state, progress, and start/finish times (public data, no character required).",
	}, getMilitaryCampaignHandler(open))
}

func getMilitaryCampaignHandler[C militaryCampaignDetailESI](open func() C) mcpsdk.ToolHandlerFor[GetMilitaryCampaignInput, GetMilitaryCampaignOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMilitaryCampaignInput) (*mcpsdk.CallToolResult, GetMilitaryCampaignOutput, error) {
		client := open()
		c, err := client.MilitaryCampaignDetail(ctx, in.CampaignID)
		if err != nil {
			return nil, GetMilitaryCampaignOutput{}, fmt.Errorf("get military campaign: %w", err)
		}
		return nil, GetMilitaryCampaignOutput{CampaignID: c.Id, State: c.State, Progress: c.Progress, Started: c.Started, Finished: c.Finished}, nil
	}
}

// --- list_military_campaign_objectives (public, cursor-paginated) ---

// militaryCampaignObjectivesESI is the ESI call list_military_campaign_objectives makes.
type militaryCampaignObjectivesESI interface {
	MilitaryCampaignObjectives(ctx context.Context, campaignID, after, before string, limit int64) ([]esiapi.MilitaryCampaignsObjectivesDetailObjective, *esiapi.Cursor, error)
}

type ListMilitaryCampaignObjectivesInput struct {
	CampaignID string `json:"campaignId" jsonschema:"the military campaign ID (a UUID, from list_military_campaigns)"`
	After      string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before     string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit      int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListMilitaryCampaignObjectivesOutput struct {
	CampaignID string              `json:"campaignId"`
	Objectives []CampaignObjective `json:"objectives"`
	NextAfter  *string             `json:"nextAfter,omitempty"`
	NextBefore *string             `json:"nextBefore,omitempty"`
}

// AddListMilitaryCampaignObjectives registers list_military_campaign_objectives. open builds the ESI client; any client with the
// militaryCampaignObjectivesESI methods will do.
func AddListMilitaryCampaignObjectives[C militaryCampaignObjectivesESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_military_campaign_objectives",
		Description: "List a military campaign's objectives with progress and participant counts, cursor-paginated (public data, no character required).",
	}, listMilitaryCampaignObjectivesHandler(open))
}

func listMilitaryCampaignObjectivesHandler[C militaryCampaignObjectivesESI](open func() C) mcpsdk.ToolHandlerFor[ListMilitaryCampaignObjectivesInput, ListMilitaryCampaignObjectivesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMilitaryCampaignObjectivesInput) (*mcpsdk.CallToolResult, ListMilitaryCampaignObjectivesOutput, error) {
		client := open()
		objectives, cursor, err := client.MilitaryCampaignObjectives(ctx, in.CampaignID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListMilitaryCampaignObjectivesOutput{}, fmt.Errorf("list military campaign objectives: %w", err)
		}
		out := ListMilitaryCampaignObjectivesOutput{CampaignID: in.CampaignID, Objectives: make([]CampaignObjective, len(objectives))}
		for i, o := range objectives {
			out.Objectives[i] = CampaignObjective{
				ObjectiveID: o.Id, State: o.State, Progress: o.Progress, Participants: toCampaignObjectiveParticipants(o.Participants),
				LastModified: o.LastModified, Started: o.Started, Finished: o.Finished,
			}
		}
		if cursor != nil {
			out.NextAfter, out.NextBefore = cursor.After, cursor.Before
		}
		return nil, out, nil
	}
}

// --- get_military_campaign_objective (public) ---

// militaryCampaignObjectiveDetailESI is the ESI call get_military_campaign_objective makes.
type militaryCampaignObjectiveDetailESI interface {
	MilitaryCampaignObjectiveDetail(ctx context.Context, campaignID, objectiveID string) (*esiapi.MilitaryCampaignsObjectivesDetail, error)
}

type GetMilitaryCampaignObjectiveInput struct {
	CampaignID  string `json:"campaignId" jsonschema:"the military campaign ID (a UUID, from list_military_campaigns)"`
	ObjectiveID string `json:"objectiveId" jsonschema:"the objective ID (a UUID, from list_military_campaign_objectives)"`
}
type GetMilitaryCampaignObjectiveOutput struct {
	CampaignID   string                        `json:"campaignId"`
	ObjectiveID  string                        `json:"objectiveId"`
	State        string                        `json:"state"`
	Progress     int64                         `json:"progress"`
	Participants CampaignObjectiveParticipants `json:"participants"`
	LastModified time.Time                     `json:"lastModified"`
	Started      *time.Time                    `json:"started,omitempty"`
	Finished     *time.Time                    `json:"finished,omitempty"`
}

// AddGetMilitaryCampaignObjective registers get_military_campaign_objective. open builds the ESI client; any client with the
// militaryCampaignObjectiveDetailESI methods will do.
func AddGetMilitaryCampaignObjective[C militaryCampaignObjectiveDetailESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_military_campaign_objective",
		Description: "Get one military campaign objective's state, progress, and participant counts (public data, no character required).",
	}, getMilitaryCampaignObjectiveHandler(open))
}

func getMilitaryCampaignObjectiveHandler[C militaryCampaignObjectiveDetailESI](open func() C) mcpsdk.ToolHandlerFor[GetMilitaryCampaignObjectiveInput, GetMilitaryCampaignObjectiveOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMilitaryCampaignObjectiveInput) (*mcpsdk.CallToolResult, GetMilitaryCampaignObjectiveOutput, error) {
		client := open()
		o, err := client.MilitaryCampaignObjectiveDetail(ctx, in.CampaignID, in.ObjectiveID)
		if err != nil {
			return nil, GetMilitaryCampaignObjectiveOutput{}, fmt.Errorf("get military campaign objective: %w", err)
		}
		return nil, GetMilitaryCampaignObjectiveOutput{
			CampaignID: in.CampaignID, ObjectiveID: o.Id, State: o.State, Progress: o.Progress,
			Participants: toCampaignObjectiveParticipants(o.Participants), LastModified: o.LastModified,
			Started: o.Started, Finished: o.Finished,
		}, nil
	}
}

// --- list_character_military_campaign_objectives (private, cursor-paginated) ---

// characterMilitaryCampaignObjectivesESI is the ESI call list_character_military_campaign_objectives makes.
type characterMilitaryCampaignObjectivesESI interface {
	CharacterMilitaryCampaignObjectives(ctx context.Context, after, before string, limit int64) ([]esiapi.CharactersMilitaryCampaignsObjectivesParticipationCharacterobjective, *esiapi.Cursor, error)
}

type ListCharacterMilitaryCampaignObjectivesInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check"`
	After       string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before      string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit       int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListCharacterMilitaryCampaignObjectivesOutput struct {
	CharacterID int32                        `json:"characterId"`
	Objectives  []CharacterCampaignObjective `json:"objectives"`
	NextAfter   *string                      `json:"nextAfter,omitempty"`
	NextBefore  *string                      `json:"nextBefore,omitempty"`
}

// AddListCharacterMilitaryCampaignObjectives registers list_character_military_campaign_objectives. open builds the ESI client; any client with the
// characterMilitaryCampaignObjectivesESI methods will do.
func AddListCharacterMilitaryCampaignObjectives[C characterMilitaryCampaignObjectivesESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_military_campaign_objectives",
		Description: "List the military campaign objectives a logged-in character has participated in, with contribution and commitment, cursor-paginated.",
	}, listCharacterMilitaryCampaignObjectivesHandler(open, store))
}

func listCharacterMilitaryCampaignObjectivesHandler[C characterMilitaryCampaignObjectivesESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterMilitaryCampaignObjectivesInput, ListCharacterMilitaryCampaignObjectivesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMilitaryCampaignObjectivesInput) (*mcpsdk.CallToolResult, ListCharacterMilitaryCampaignObjectivesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMilitaryCampaignObjectivesOutput{}, err
		}
		if !found {
			return nil, ListCharacterMilitaryCampaignObjectivesOutput{}, fmt.Errorf("list character military campaign objectives for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMilitaryCampaignObjectivesOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		objectives, cursor, err := client.CharacterMilitaryCampaignObjectives(ctx, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListCharacterMilitaryCampaignObjectivesOutput{}, fmt.Errorf("list character military campaign objectives: %w", err)
		}
		out := ListCharacterMilitaryCampaignObjectivesOutput{CharacterID: in.CharacterID, Objectives: make([]CharacterCampaignObjective, len(objectives))}
		for i, o := range objectives {
			out.Objectives[i] = CharacterCampaignObjective{
				ObjectiveID: o.Id, CampaignID: o.CampaignId, Contributed: o.Contributed, IsCommitted: o.IsCommitted, LastModified: o.LastModified,
			}
		}
		if cursor != nil {
			out.NextAfter, out.NextBefore = cursor.After, cursor.Before
		}
		return nil, out, nil
	}
}

// --- get_character_military_campaign_objective_participation (private) ---

// characterMilitaryCampaignObjectiveParticipationESI is the ESI call get_character_military_campaign_objective_participation makes.
type characterMilitaryCampaignObjectiveParticipationESI interface {
	CharacterMilitaryCampaignObjectiveParticipation(ctx context.Context, objectiveID string) (*esiapi.CharactersMilitaryCampaignsObjectivesParticipation, error)
}

type GetCharacterMilitaryCampaignObjectiveParticipationInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check"`
	ObjectiveID string `json:"objectiveId" jsonschema:"the objective ID (a UUID, from list_military_campaign_objectives or list_character_military_campaign_objectives)"`
}
type GetCharacterMilitaryCampaignObjectiveParticipationOutput struct {
	CharacterID  int32     `json:"characterId"`
	ObjectiveID  string    `json:"objectiveId"`
	CampaignID   string    `json:"campaignId"`
	Contributed  int64     `json:"contributed"`
	IsCommitted  bool      `json:"isCommitted"`
	LastModified time.Time `json:"lastModified"`
}

// AddGetCharacterMilitaryCampaignObjectiveParticipation registers get_character_military_campaign_objective_participation. open builds the ESI client; any client with the
// characterMilitaryCampaignObjectiveParticipationESI methods will do.
func AddGetCharacterMilitaryCampaignObjectiveParticipation[C characterMilitaryCampaignObjectiveParticipationESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_military_campaign_objective_participation",
		Description: "Get a logged-in character's contribution to and commitment status in one military campaign objective.",
	}, getCharacterMilitaryCampaignObjectiveParticipationHandler(open, store))
}

func getCharacterMilitaryCampaignObjectiveParticipationHandler[C characterMilitaryCampaignObjectiveParticipationESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterMilitaryCampaignObjectiveParticipationInput, GetCharacterMilitaryCampaignObjectiveParticipationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterMilitaryCampaignObjectiveParticipationInput) (*mcpsdk.CallToolResult, GetCharacterMilitaryCampaignObjectiveParticipationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{}, err
		}
		if !found {
			return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{}, fmt.Errorf("get character military campaign objective participation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		p, err := client.CharacterMilitaryCampaignObjectiveParticipation(ctx, in.ObjectiveID)
		if err != nil {
			return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{}, fmt.Errorf("get character military campaign objective participation: %w", err)
		}
		return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{
			CharacterID: in.CharacterID, ObjectiveID: p.Id, CampaignID: p.CampaignId, Contributed: p.Contributed,
			IsCommitted: p.IsCommitted, LastModified: p.LastModified,
		}, nil
	}
}
