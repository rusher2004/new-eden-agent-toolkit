package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// --- shared types ---

type JobProgress struct {
	Current int64 `json:"current"`
	Desired int64 `json:"desired"`
}

func toJobProgress(p esiapi.FreelanceJobsDetailProgress) JobProgress {
	return JobProgress{Current: p.Current, Desired: p.Desired}
}

type JobReward struct {
	Initial   float64 `json:"initial"`
	Remaining float64 `json:"remaining"`
}

func toJobReward(r *esiapi.FreelanceJobsDetailReward) *JobReward {
	if r == nil {
		return nil
	}
	return &JobReward{Initial: r.Initial, Remaining: r.Remaining}
}

type FreelanceJobSummary struct {
	JobID        string      `json:"jobId"`
	Name         string      `json:"name"`
	State        string      `json:"state"`
	LastModified time.Time   `json:"lastModified"`
	Progress     JobProgress `json:"progress"`
	Reward       *JobReward  `json:"reward,omitempty"`
}

func toFreelanceJobSummary(j esiapi.FreelanceJobsDetailFreelancejob) FreelanceJobSummary {
	return FreelanceJobSummary{
		JobID: j.Id, Name: j.Name, State: j.State, LastModified: j.LastModified,
		Progress: toJobProgress(j.Progress), Reward: toJobReward(j.Reward),
	}
}

func toFreelanceJobSummaries(js []esiapi.FreelanceJobsDetailFreelancejob) []FreelanceJobSummary {
	out := make([]FreelanceJobSummary, len(js))
	for i, j := range js {
		out[i] = toFreelanceJobSummary(j)
	}
	return out
}

// --- list_character_freelance_jobs (private) ---

// characterFreelanceJobsESI is the ESI call list_character_freelance_jobs makes.
type characterFreelanceJobsESI interface {
	CharacterFreelanceJobs(ctx context.Context) ([]esiapi.FreelanceJobsDetailFreelancejob, error)
}

type ListCharacterFreelanceJobsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterFreelanceJobsOutput struct {
	CharacterID int32                 `json:"characterId"`
	Jobs        []FreelanceJobSummary `json:"jobs"`
}

// AddListCharacterFreelanceJobs registers list_character_freelance_jobs. open builds the ESI client; any client with the
// characterFreelanceJobsESI methods will do.
func AddListCharacterFreelanceJobs[C characterFreelanceJobsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_freelance_jobs",
		Description: "List a logged-in EVE Online character's freelance jobs.",
	}, listCharacterFreelanceJobsHandler(open, store))
}

func listCharacterFreelanceJobsHandler[C characterFreelanceJobsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterFreelanceJobsInput, ListCharacterFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterFreelanceJobsInput) (*mcpsdk.CallToolResult, ListCharacterFreelanceJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFreelanceJobsOutput{}, err
		}
		if !found {
			return nil, ListCharacterFreelanceJobsOutput{}, fmt.Errorf("list character freelance jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFreelanceJobsOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		jobs, err := client.CharacterFreelanceJobs(ctx)
		if err != nil {
			return nil, ListCharacterFreelanceJobsOutput{}, fmt.Errorf("list character freelance jobs: %w", err)
		}
		return nil, ListCharacterFreelanceJobsOutput{CharacterID: in.CharacterID, Jobs: toFreelanceJobSummaries(jobs)}, nil
	}
}

// --- get_character_freelance_job_participation (private) ---

// characterFreelanceJobParticipationESI is the ESI call get_character_freelance_job_participation makes.
type characterFreelanceJobParticipationESI interface {
	CharacterFreelanceJobParticipation(ctx context.Context, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error)
}

type GetCharacterFreelanceJobParticipationInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check"`
	JobID       string `json:"jobId" jsonschema:"the freelance job ID"`
}
type GetCharacterFreelanceJobParticipationOutput struct {
	CharacterID  int32     `json:"characterId"`
	JobID        string    `json:"jobId"`
	State        string    `json:"state"`
	Contributed  int64     `json:"contributed"`
	LastModified time.Time `json:"lastModified"`
}

// AddGetCharacterFreelanceJobParticipation registers get_character_freelance_job_participation. open builds the ESI client; any client with the
// characterFreelanceJobParticipationESI methods will do.
func AddGetCharacterFreelanceJobParticipation[C characterFreelanceJobParticipationESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_freelance_job_participation",
		Description: "Get a logged-in character's contribution to a freelance job.",
	}, getCharacterFreelanceJobParticipationHandler(open, store))
}

func getCharacterFreelanceJobParticipationHandler[C characterFreelanceJobParticipationESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterFreelanceJobParticipationInput, GetCharacterFreelanceJobParticipationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFreelanceJobParticipationInput) (*mcpsdk.CallToolResult, GetCharacterFreelanceJobParticipationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, err
		}
		if !found {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, fmt.Errorf("get character freelance job participation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		p, err := client.CharacterFreelanceJobParticipation(ctx, in.JobID)
		if err != nil {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, fmt.Errorf("get character freelance job participation: %w", err)
		}
		return nil, GetCharacterFreelanceJobParticipationOutput{
			CharacterID: in.CharacterID, JobID: in.JobID, State: p.State, Contributed: p.Contributed, LastModified: p.LastModified,
		}, nil
	}
}

// --- list_corporation_freelance_jobs (private, cursor-paginated) ---

// corporationFreelanceJobsESI is the ESI call list_corporation_freelance_jobs makes.
type corporationFreelanceJobsESI interface {
	CorporationFreelanceJobs(ctx context.Context, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error)
}

type ListCorporationFreelanceJobsInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation ID to look up"`
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
}
type ListCorporationFreelanceJobsOutput struct {
	CharacterID   int32                 `json:"characterId"`
	CorporationID int64                 `json:"corporationId"`
	Jobs          []FreelanceJobSummary `json:"jobs"`
	NextAfter     *string               `json:"nextAfter,omitempty"`
	NextBefore    *string               `json:"nextBefore,omitempty"`
}

// AddListCorporationFreelanceJobs registers list_corporation_freelance_jobs. open builds the ESI client; any client with the
// corporationFreelanceJobsESI methods will do.
func AddListCorporationFreelanceJobs[C corporationFreelanceJobsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_freelance_jobs",
		Description: "List a corporation's freelance jobs, using a logged-in character's roles to authorize.",
	}, listCorporationFreelanceJobsHandler(open, store))
}

func listCorporationFreelanceJobsHandler[C corporationFreelanceJobsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationFreelanceJobsInput, ListCorporationFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFreelanceJobsInput) (*mcpsdk.CallToolResult, ListCorporationFreelanceJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobsOutput{}, err
		}
		if !found {
			return nil, ListCorporationFreelanceJobsOutput{}, fmt.Errorf("list corporation freelance jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobsOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		jobs, cursor, err := client.CorporationFreelanceJobs(ctx, in.CorporationID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListCorporationFreelanceJobsOutput{}, fmt.Errorf("list corporation freelance jobs: %w", err)
		}
		out := ListCorporationFreelanceJobsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Jobs: toFreelanceJobSummaries(jobs)}
		if cursor != nil {
			out.NextAfter = cursor.After
			out.NextBefore = cursor.Before
		}
		return nil, out, nil
	}
}

// --- list_corporation_freelance_job_participants (private, cursor-paginated) ---

type JobParticipant struct {
	ParticipantID int64  `json:"participantId"`
	Name          string `json:"name"`
	State         string `json:"state"`
	Contributed   int64  `json:"contributed"`
}

// corporationFreelanceJobParticipantsESI is the ESI call list_corporation_freelance_job_participants makes.
type corporationFreelanceJobParticipantsESI interface {
	CorporationFreelanceJobParticipants(ctx context.Context, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error)
}

type ListCorporationFreelanceJobParticipantsInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation ID the job belongs to"`
	JobID         string `json:"jobId" jsonschema:"the freelance job ID"`
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
}
type ListCorporationFreelanceJobParticipantsOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	JobID         string           `json:"jobId"`
	Participants  []JobParticipant `json:"participants"`
	NextAfter     *string          `json:"nextAfter,omitempty"`
	NextBefore    *string          `json:"nextBefore,omitempty"`
}

// AddListCorporationFreelanceJobParticipants registers list_corporation_freelance_job_participants. open builds the ESI client; any client with the
// corporationFreelanceJobParticipantsESI methods will do.
func AddListCorporationFreelanceJobParticipants[C corporationFreelanceJobParticipantsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_freelance_job_participants",
		Description: "List the participants in a corporation's freelance job, using a logged-in character's roles to authorize.",
	}, listCorporationFreelanceJobParticipantsHandler(open, store))
}

func listCorporationFreelanceJobParticipantsHandler[C corporationFreelanceJobParticipantsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationFreelanceJobParticipantsInput, ListCorporationFreelanceJobParticipantsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFreelanceJobParticipantsInput) (*mcpsdk.CallToolResult, ListCorporationFreelanceJobParticipantsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, err
		}
		if !found {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, fmt.Errorf("list corporation freelance job participants for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		participants, cursor, err := client.CorporationFreelanceJobParticipants(ctx, in.CorporationID, in.JobID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, fmt.Errorf("list corporation freelance job participants: %w", err)
		}
		out := make([]JobParticipant, len(participants))
		for i, p := range participants {
			out[i] = JobParticipant{ParticipantID: p.Id, Name: p.Name, State: p.State, Contributed: p.Contributed}
		}
		result := ListCorporationFreelanceJobParticipantsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, JobID: in.JobID, Participants: out}
		if cursor != nil {
			result.NextAfter = cursor.After
			result.NextBefore = cursor.Before
		}
		return nil, result, nil
	}
}

// --- get_freelance_job (public) ---

type ParameterMatcherValue struct {
	ValueType string   `json:"valueType"`
	Values    []string `json:"values"`
}

type ParameterMatcher struct {
	Values []ParameterMatcherValue `json:"values"`
}

func toParameterMatcher(m esiapi.FreelanceJobsDetailParametermatcher) ParameterMatcher {
	values := make([]ParameterMatcherValue, len(m.Values))
	for i, v := range m.Values {
		values[i] = ParameterMatcherValue{ValueType: v.ValueType, Values: v.Values}
	}
	return ParameterMatcher{Values: values}
}

type ParameterCorporationItemDelivery struct {
	CorporationOfficeLocation ParameterMatcher `json:"corporationOfficeLocation"`
	ItemType                  ParameterMatcher `json:"itemType"`
}

type ParameterOptions struct {
	Selected []string `json:"selected"`
}

// ConfigParameter is the flattened form of FreelanceJobsDetailConfigurationParametersValue, a
// 4-variant oneOf. Kind names the active variant ("boolean", "corporation_item_delivery",
// "matcher", "options"); only the fields relevant to that Kind are populated.
type ConfigParameter struct {
	Kind                    string                            `json:"kind"`
	BooleanValue            *bool                             `json:"booleanValue,omitempty"`
	CorporationItemDelivery *ParameterCorporationItemDelivery `json:"corporationItemDelivery,omitempty"`
	Matcher                 *ParameterMatcher                 `json:"matcher,omitempty"`
	Options                 *ParameterOptions                 `json:"options,omitempty"`
}

func toConfigParameter(v esiapi.FreelanceJobsDetailConfigurationParametersValue) ConfigParameter {
	switch {
	case v.Boolean != nil && v.Boolean.Boolean != nil:
		val := v.Boolean.Boolean.Value
		return ConfigParameter{Kind: "boolean", BooleanValue: &val}
	case v.CorporationItemDelivery != nil && v.CorporationItemDelivery.CorporationItemDelivery != nil:
		d := v.CorporationItemDelivery.CorporationItemDelivery
		return ConfigParameter{Kind: "corporation_item_delivery", CorporationItemDelivery: &ParameterCorporationItemDelivery{
			CorporationOfficeLocation: toParameterMatcher(d.CorporationOfficeLocation),
			ItemType:                  toParameterMatcher(d.ItemType),
		}}
	case v.Matcher != nil && v.Matcher.Matcher != nil:
		m := toParameterMatcher(*v.Matcher.Matcher)
		return ConfigParameter{Kind: "matcher", Matcher: &m}
	case v.Options != nil && v.Options.Options != nil:
		return ConfigParameter{Kind: "options", Options: &ParameterOptions{Selected: v.Options.Options.Selected}}
	default:
		return ConfigParameter{}
	}
}

type JobConfiguration struct {
	Method     string                     `json:"method"`
	Parameters map[string]ConfigParameter `json:"parameters"`
	Version    int64                      `json:"version"`
}

func toJobConfiguration(cfg esiapi.FreelanceJobsDetailConfiguration) JobConfiguration {
	params := make(map[string]ConfigParameter, len(cfg.Parameters))
	for k, v := range cfg.Parameters {
		params[k] = toConfigParameter(v)
	}
	return JobConfiguration{Method: cfg.Method, Parameters: params, Version: cfg.Version}
}

type JobContributionLimits struct {
	ContributionPerParticipantLimit *int64   `json:"contributionPerParticipantLimit,omitempty"`
	MaxCommittedParticipants        int64    `json:"maxCommittedParticipants"`
	RewardPerContribution           *float64 `json:"rewardPerContribution,omitempty"`
	SubmissionLimit                 *int64   `json:"submissionLimit,omitempty"`
	SubmissionMultiplier            *float64 `json:"submissionMultiplier,omitempty"`
}

func toJobContributionLimits(c *esiapi.FreelanceJobsDetailContribution) *JobContributionLimits {
	if c == nil {
		return nil
	}
	return &JobContributionLimits{
		ContributionPerParticipantLimit: c.ContributionPerParticipantLimit, MaxCommittedParticipants: c.MaxCommittedParticipants,
		RewardPerContribution: c.RewardPerContribution, SubmissionLimit: c.SubmissionLimit, SubmissionMultiplier: c.SubmissionMultiplier,
	}
}

type JobCreatorEntity struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type JobCreator struct {
	Character   JobCreatorEntity `json:"character"`
	Corporation JobCreatorEntity `json:"corporation"`
}

type JobDetails struct {
	Career      string     `json:"career"`
	Created     time.Time  `json:"created"`
	Description string     `json:"description"`
	Expires     *time.Time `json:"expires,omitempty"`
	Finished    *time.Time `json:"finished,omitempty"`
}

type JobBroadcastLocation struct {
	SystemID int64  `json:"systemId"`
	Name     string `json:"name"`
}
type JobRestrictions struct {
	MinimumAge *int64 `json:"minimumAge,omitempty"`
	MaximumAge *int64 `json:"maximumAge,omitempty"`
}
type JobAccessAndVisibility struct {
	AclProtected       bool                   `json:"aclProtected"`
	BroadcastLocations []JobBroadcastLocation `json:"broadcastLocations,omitempty"`
	Restrictions       *JobRestrictions       `json:"restrictions,omitempty"`
}

// freelanceJobESI is the ESI call get_freelance_job makes.
type freelanceJobESI interface {
	FreelanceJobDetail(ctx context.Context, jobID string) (*esiapi.FreelanceJobsDetail, error)
}

type GetFreelanceJobInput struct {
	JobID string `json:"jobId" jsonschema:"the freelance job ID"`
}
type GetFreelanceJobOutput struct {
	JobID               string                 `json:"jobId"`
	Name                string                 `json:"name"`
	State               string                 `json:"state"`
	LastModified        time.Time              `json:"lastModified"`
	Progress            JobProgress            `json:"progress"`
	Reward              *JobReward             `json:"reward,omitempty"`
	Configuration       JobConfiguration       `json:"configuration"`
	Contribution        *JobContributionLimits `json:"contribution,omitempty"`
	Creator             JobCreator             `json:"creator"`
	Details             JobDetails             `json:"details"`
	AccessAndVisibility JobAccessAndVisibility `json:"accessAndVisibility"`
}

// AddGetFreelanceJob registers get_freelance_job. open builds the ESI client; any client with the
// freelanceJobESI methods will do.
func AddGetFreelanceJob[C freelanceJobESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_freelance_job",
		Description: "Get the full details of a freelance job (public data, no character required).",
	}, getFreelanceJobHandler(open))
}

func getFreelanceJobHandler[C freelanceJobESI](open func() C) mcpsdk.ToolHandlerFor[GetFreelanceJobInput, GetFreelanceJobOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFreelanceJobInput) (*mcpsdk.CallToolResult, GetFreelanceJobOutput, error) {
		client := open()
		detail, err := client.FreelanceJobDetail(ctx, in.JobID)
		if err != nil {
			return nil, GetFreelanceJobOutput{}, fmt.Errorf("get freelance job: %w", err)
		}
		locations := make([]JobBroadcastLocation, len(detail.AccessAndVisibility.BroadcastLocations))
		for i, l := range detail.AccessAndVisibility.BroadcastLocations {
			locations[i] = JobBroadcastLocation{SystemID: l.Id, Name: l.Name}
		}
		var restrictions *JobRestrictions
		if detail.AccessAndVisibility.Restrictions != nil {
			restrictions = &JobRestrictions{
				MinimumAge: detail.AccessAndVisibility.Restrictions.MinimumAge,
				MaximumAge: detail.AccessAndVisibility.Restrictions.MaximumAge,
			}
		}
		return nil, GetFreelanceJobOutput{
			JobID: in.JobID, Name: detail.Name, State: detail.State, LastModified: detail.LastModified,
			Progress: toJobProgress(detail.Progress), Reward: toJobReward(detail.Reward),
			Configuration: toJobConfiguration(detail.Configuration), Contribution: toJobContributionLimits(detail.Contribution),
			Creator: JobCreator{
				Character:   JobCreatorEntity{ID: detail.Details.Creator.Character.Id, Name: detail.Details.Creator.Character.Name},
				Corporation: JobCreatorEntity{ID: detail.Details.Creator.Corporation.Id, Name: detail.Details.Creator.Corporation.Name},
			},
			Details: JobDetails{
				Career: detail.Details.Career, Created: detail.Details.Created, Description: detail.Details.Description,
				Expires: detail.Details.Expires, Finished: detail.Details.Finished,
			},
			AccessAndVisibility: JobAccessAndVisibility{
				AclProtected: detail.AccessAndVisibility.AclProtected, BroadcastLocations: locations, Restrictions: restrictions,
			},
		}, nil
	}
}

// --- list_freelance_jobs (public, cursor-paginated) ---

// freelanceJobsESI is the ESI call list_freelance_jobs makes.
type freelanceJobsESI interface {
	FreelanceJobsListing(ctx context.Context, after, before string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error)
}

type ListFreelanceJobsInput struct {
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"max records to return; omit for ESI's default"`
	CorporationID int64  `json:"corporationId,omitempty" jsonschema:"filter to only this corporation's jobs; omit for all corporations"`
}
type ListFreelanceJobsOutput struct {
	Jobs       []FreelanceJobSummary `json:"jobs"`
	NextAfter  *string               `json:"nextAfter,omitempty"`
	NextBefore *string               `json:"nextBefore,omitempty"`
}

// AddListFreelanceJobs registers list_freelance_jobs. open builds the ESI client; any client with the
// freelanceJobsESI methods will do.
func AddListFreelanceJobs[C freelanceJobsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_freelance_jobs",
		Description: "List freelance jobs game-wide, optionally filtered by corporation (public data, no character required).",
	}, listFreelanceJobsHandler(open))
}

func listFreelanceJobsHandler[C freelanceJobsESI](open func() C) mcpsdk.ToolHandlerFor[ListFreelanceJobsInput, ListFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFreelanceJobsInput) (*mcpsdk.CallToolResult, ListFreelanceJobsOutput, error) {
		client := open()
		jobs, cursor, err := client.FreelanceJobsListing(ctx, in.After, in.Before, in.Limit, in.CorporationID)
		if err != nil {
			return nil, ListFreelanceJobsOutput{}, fmt.Errorf("list freelance jobs: %w", err)
		}
		out := ListFreelanceJobsOutput{Jobs: toFreelanceJobSummaries(jobs)}
		if cursor != nil {
			out.NextAfter = cursor.After
			out.NextBefore = cursor.Before
		}
		return nil, out, nil
	}
}
