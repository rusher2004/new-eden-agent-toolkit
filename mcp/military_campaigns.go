package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
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

var listMilitaryCampaigns = func(ctx context.Context, userAgent string) ([]esiapi.MilitaryCampaignsDetailCampaign, error) {
	return esi.NewPublicClient(userAgent).MilitaryCampaigns(ctx)
}

type ListMilitaryCampaignsInput struct{}
type ListMilitaryCampaignsOutput struct {
	Campaigns []MilitaryCampaign `json:"campaigns"`
}

func listMilitaryCampaignsHandler(version string) mcpsdk.ToolHandlerFor[ListMilitaryCampaignsInput, ListMilitaryCampaignsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ ListMilitaryCampaignsInput) (*mcpsdk.CallToolResult, ListMilitaryCampaignsOutput, error) {
		campaigns, err := listMilitaryCampaigns(ctx, sso.UserAgent(version))
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

var getMilitaryCampaign = func(ctx context.Context, userAgent, campaignID string) (*esiapi.MilitaryCampaignsDetail, error) {
	return esi.NewPublicClient(userAgent).MilitaryCampaignDetail(ctx, campaignID)
}

type GetMilitaryCampaignInput struct {
	CampaignID string `json:"campaignId" jsonschema:"the military campaign ID (a UUID, from list_military_campaigns)"`
}
type GetMilitaryCampaignOutput = MilitaryCampaign

func getMilitaryCampaignHandler(version string) mcpsdk.ToolHandlerFor[GetMilitaryCampaignInput, GetMilitaryCampaignOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMilitaryCampaignInput) (*mcpsdk.CallToolResult, GetMilitaryCampaignOutput, error) {
		c, err := getMilitaryCampaign(ctx, sso.UserAgent(version), in.CampaignID)
		if err != nil {
			return nil, GetMilitaryCampaignOutput{}, fmt.Errorf("get military campaign: %w", err)
		}
		return nil, GetMilitaryCampaignOutput{CampaignID: c.Id, State: c.State, Progress: c.Progress, Started: c.Started, Finished: c.Finished}, nil
	}
}

// --- list_military_campaign_objectives (public, cursor-paginated) ---

var listMilitaryCampaignObjectives = func(ctx context.Context, userAgent, campaignID, after, before string, limit int64) ([]esiapi.MilitaryCampaignsObjectivesDetailObjective, *esiapi.Cursor, error) {
	return esi.NewPublicClient(userAgent).MilitaryCampaignObjectives(ctx, campaignID, after, before, limit)
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

func listMilitaryCampaignObjectivesHandler(version string) mcpsdk.ToolHandlerFor[ListMilitaryCampaignObjectivesInput, ListMilitaryCampaignObjectivesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMilitaryCampaignObjectivesInput) (*mcpsdk.CallToolResult, ListMilitaryCampaignObjectivesOutput, error) {
		objectives, cursor, err := listMilitaryCampaignObjectives(ctx, sso.UserAgent(version), in.CampaignID, in.After, in.Before, in.Limit)
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

var getMilitaryCampaignObjective = func(ctx context.Context, userAgent, campaignID, objectiveID string) (*esiapi.MilitaryCampaignsObjectivesDetail, error) {
	return esi.NewPublicClient(userAgent).MilitaryCampaignObjectiveDetail(ctx, campaignID, objectiveID)
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

func getMilitaryCampaignObjectiveHandler(version string) mcpsdk.ToolHandlerFor[GetMilitaryCampaignObjectiveInput, GetMilitaryCampaignObjectiveOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMilitaryCampaignObjectiveInput) (*mcpsdk.CallToolResult, GetMilitaryCampaignObjectiveOutput, error) {
		o, err := getMilitaryCampaignObjective(ctx, sso.UserAgent(version), in.CampaignID, in.ObjectiveID)
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

var listCharacterMilitaryCampaignObjectives = func(ctx context.Context, clientID string, characterID int32, userAgent, after, before string, limit int64) ([]esiapi.CharactersMilitaryCampaignsObjectivesParticipationCharacterobjective, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterMilitaryCampaignObjectives(ctx, after, before, limit)
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

func listCharacterMilitaryCampaignObjectivesHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCharacterMilitaryCampaignObjectivesInput, ListCharacterMilitaryCampaignObjectivesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMilitaryCampaignObjectivesInput) (*mcpsdk.CallToolResult, ListCharacterMilitaryCampaignObjectivesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMilitaryCampaignObjectivesOutput{}, err
		}
		if !found {
			return nil, ListCharacterMilitaryCampaignObjectivesOutput{}, fmt.Errorf("list character military campaign objectives for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		objectives, cursor, err := listCharacterMilitaryCampaignObjectives(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.After, in.Before, in.Limit)
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

var getCharacterMilitaryCampaignObjectiveParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent, objectiveID string) (*esiapi.CharactersMilitaryCampaignsObjectivesParticipation, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterMilitaryCampaignObjectiveParticipation(ctx, objectiveID)
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

func getCharacterMilitaryCampaignObjectiveParticipationHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[GetCharacterMilitaryCampaignObjectiveParticipationInput, GetCharacterMilitaryCampaignObjectiveParticipationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterMilitaryCampaignObjectiveParticipationInput) (*mcpsdk.CallToolResult, GetCharacterMilitaryCampaignObjectiveParticipationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{}, err
		}
		if !found {
			return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{}, fmt.Errorf("get character military campaign objective participation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		p, err := getCharacterMilitaryCampaignObjectiveParticipation(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ObjectiveID)
		if err != nil {
			return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{}, fmt.Errorf("get character military campaign objective participation: %w", err)
		}
		return nil, GetCharacterMilitaryCampaignObjectiveParticipationOutput{
			CharacterID: in.CharacterID, ObjectiveID: p.Id, CampaignID: p.CampaignId, Contributed: p.Contributed,
			IsCommitted: p.IsCommitted, LastModified: p.LastModified,
		}, nil
	}
}
