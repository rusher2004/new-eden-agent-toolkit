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

var listCharacterFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.FreelanceJobsDetailFreelancejob, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterFreelanceJobs(ctx)
}

type ListCharacterFreelanceJobsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterFreelanceJobsOutput struct {
	CharacterID int32                 `json:"characterId"`
	Jobs        []FreelanceJobSummary `json:"jobs"`
}

func listCharacterFreelanceJobsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCharacterFreelanceJobsInput, ListCharacterFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterFreelanceJobsInput) (*mcpsdk.CallToolResult, ListCharacterFreelanceJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFreelanceJobsOutput{}, err
		}
		if !found {
			return nil, ListCharacterFreelanceJobsOutput{}, fmt.Errorf("list character freelance jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		jobs, err := listCharacterFreelanceJobs(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterFreelanceJobsOutput{}, fmt.Errorf("list character freelance jobs: %w", err)
		}
		return nil, ListCharacterFreelanceJobsOutput{CharacterID: in.CharacterID, Jobs: toFreelanceJobSummaries(jobs)}, nil
	}
}

// --- get_character_freelance_job_participation (private) ---

var getCharacterFreelanceJobParticipation = func(ctx context.Context, clientID string, characterID int32, userAgent string, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterFreelanceJobParticipation(ctx, jobID)
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

func getCharacterFreelanceJobParticipationHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[GetCharacterFreelanceJobParticipationInput, GetCharacterFreelanceJobParticipationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFreelanceJobParticipationInput) (*mcpsdk.CallToolResult, GetCharacterFreelanceJobParticipationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, err
		}
		if !found {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, fmt.Errorf("get character freelance job participation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		p, err := getCharacterFreelanceJobParticipation(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.JobID)
		if err != nil {
			return nil, GetCharacterFreelanceJobParticipationOutput{}, fmt.Errorf("get character freelance job participation: %w", err)
		}
		return nil, GetCharacterFreelanceJobParticipationOutput{
			CharacterID: in.CharacterID, JobID: in.JobID, State: p.State, Contributed: p.Contributed, LastModified: p.LastModified,
		}, nil
	}
}

// --- list_corporation_freelance_jobs (private, cursor-paginated) ---

var listCorporationFreelanceJobs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationFreelanceJobs(ctx, corporationID, after, before, limit)
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

func listCorporationFreelanceJobsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCorporationFreelanceJobsInput, ListCorporationFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFreelanceJobsInput) (*mcpsdk.CallToolResult, ListCorporationFreelanceJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobsOutput{}, err
		}
		if !found {
			return nil, ListCorporationFreelanceJobsOutput{}, fmt.Errorf("list corporation freelance jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		jobs, cursor, err := listCorporationFreelanceJobs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.After, in.Before, in.Limit)
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

var listCorporationFreelanceJobParticipants = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationFreelanceJobParticipants(ctx, corporationID, jobID, after, before, limit)
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

func listCorporationFreelanceJobParticipantsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCorporationFreelanceJobParticipantsInput, ListCorporationFreelanceJobParticipantsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFreelanceJobParticipantsInput) (*mcpsdk.CallToolResult, ListCorporationFreelanceJobParticipantsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, err
		}
		if !found {
			return nil, ListCorporationFreelanceJobParticipantsOutput{}, fmt.Errorf("list corporation freelance job participants for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		participants, cursor, err := listCorporationFreelanceJobParticipants(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.JobID, in.After, in.Before, in.Limit)
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

var getFreelanceJob = func(ctx context.Context, userAgent string, jobID string) (*esiapi.FreelanceJobsDetail, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FreelanceJobDetail(ctx, jobID)
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

func getFreelanceJobHandler(version string) mcpsdk.ToolHandlerFor[GetFreelanceJobInput, GetFreelanceJobOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFreelanceJobInput) (*mcpsdk.CallToolResult, GetFreelanceJobOutput, error) {
		detail, err := getFreelanceJob(ctx, sso.UserAgent(version), in.JobID)
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

var listFreelanceJobs = func(ctx context.Context, userAgent string, after, before string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FreelanceJobsListing(ctx, after, before, limit, corporationID)
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

func listFreelanceJobsHandler(version string) mcpsdk.ToolHandlerFor[ListFreelanceJobsInput, ListFreelanceJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFreelanceJobsInput) (*mcpsdk.CallToolResult, ListFreelanceJobsOutput, error) {
		jobs, cursor, err := listFreelanceJobs(ctx, sso.UserAgent(version), in.After, in.Before, in.Limit, in.CorporationID)
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
