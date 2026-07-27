# ESI Freelance Jobs (Sixteenth Batch) Design

## Status

Approved (autonomous execution). Sixteenth batch: the "Freelance Jobs" tag alone (6 endpoints, all
read-only, mixed public/private). Conceptually close to Corporation Projects (contribution-based
jobs with progress/reward tracking, cursor pagination) but with a much smaller polymorphic
surface: one nested `Configuration.Parameters` map whose values are a 4-variant oneOf (vs.
Corporation Projects' 17-variant top-level `Configuration`).

## Scope

**2 private, character-scoped** (`characterId` for auth, no separate target — genuinely "my own
character"): `GetCharactersFreelanceJobsListing` (not paginated), `GetCharactersFreelanceJobsParticipation`
(takes a `jobId string`, not paginated).

**2 private, corp-scoped** (`characterId` for auth + `corporationId` for target, cursor-paginated
via `after`/`before`/`limit`): `GetCorporationsFreelanceJobsListing`, `GetCorporationsFreelanceJobsParticipants`
(takes `corporationId` + `jobId string`).

**2 public** (`esi.NewPublicClient`, no existence check — neither takes a `characterId` or
`corporationId` at all): `GetFreelanceJobsDetail` (takes `jobId string`, not paginated),
`GetFreelanceJobsListing` (cursor-paginated, plus an optional `corporationId` filter — confirmed
from the generated request builder's `.CorporationId(int64)` method).

`jobId` is a **string** (like Corporation Projects' `projectId`) — consistent, not a one-off.

## New pattern: nested 4-variant `Configuration.Parameters` oneOf

`FreelanceJobsDetail.Configuration` is `{Method string, Parameters map[string]ParametersValue,
Version int64}` — NOT itself polymorphic (unlike Corporation Projects' top-level `Configuration`).
The polymorphism lives one level deeper, in each map *value*: `FreelanceJobsDetailConfigurationParametersValue`
is a 4-variant oneOf (`Boolean`, `CorporationItemDelivery`, `Matcher`, `Options`), each
double-wrapped exactly like Corporation Projects' variants (e.g. `Boolean{Boolean
*FreelanceJobsDetailParameterboolean{Value bool}}`). Flattened here the same way Corporation
Projects flattened its 17-variant `Configuration`: a discriminated `ConfigParameter{Kind string,
...}` MCP type with only the fields relevant to `Kind` populated, mapped by a `toConfigParameter`
function.

The 4 variants: `Boolean{Value bool}`; `CorporationItemDelivery{CorporationOfficeLocation, ItemType
Matcher}` (itself nests 2 `Matcher`s); `Matcher{Values []MatcherValue}` where `MatcherValue{ValueType
string, Values []string}`; `Options{Selected []string}`.

## `esi/freelance_jobs.go` (6 methods)

```go
func (c *Client) CharacterFreelanceJobs(ctx context.Context) ([]esiapi.FreelanceJobsDetailFreelancejob, error)
func (c *Client) CharacterFreelanceJobParticipation(ctx context.Context, jobID string) (*esiapi.CharactersFreelanceJobsParticipation, error)
func (c *Client) CorporationFreelanceJobs(ctx context.Context, corporationID int64, after, before string, limit int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error)
func (c *Client) CorporationFreelanceJobParticipants(ctx context.Context, corporationID int64, jobID string, after, before string, limit int64) ([]esiapi.CorporationsFreelanceJobsParticipantsParticipant, *esiapi.Cursor, error)
func (c *Client) FreelanceJobDetail(ctx context.Context, jobID string) (*esiapi.FreelanceJobsDetail, error)
func (c *Client) FreelanceJobsListing(ctx context.Context, after, before string, limit, corporationID int64) ([]esiapi.FreelanceJobsDetailFreelancejob, *esiapi.Cursor, error)
```

`CharacterFreelanceJobs` uses `int64(c.characterID)` internally (genuinely "my own character"),
same as `CharacterFleet`. Cursor pagination follows the exact convention established by
Corporation Projects: `after`/`before` empty string omits the query param, `limit`/`corporationID`
`0` omits theirs; every cursor-paginated method returns `(items, *esiapi.Cursor, error)` — not the
old page/`totalPages` style.

Verified model shapes: `FreelanceJobsDetailFreelancejob{Id, Name, State string, LastModified
time.Time, Progress FreelanceJobsDetailProgress{Current, Desired int64}, Reward
*FreelanceJobsDetailReward{Initial, Remaining float64}}` — the shared summary type reused across
character listing, corp listing, and public listing. `CharactersFreelanceJobsParticipation{Contributed
int64, LastModified time.Time, State string}`. `CorporationsFreelanceJobsParticipantsParticipant{Contributed,
Id int64, Name, State string}`.

## `mcp/freelance_jobs.go` (6 tools)

- `list_character_freelance_jobs` — private, input `{characterId int32}`, output `{characterId
  int32, jobs []FreelanceJobSummary}` where `FreelanceJobSummary{JobID, Name, State string,
  LastModified time.Time, Progress JobProgress, Reward *JobReward}` (reusing `JobProgress{Current,
  Desired int64}`/`JobReward{Initial, Remaining float64}` across all 3 listing tools)
- `get_character_freelance_job_participation` — private, input `{characterId int32, jobId
  string}`, output `{characterId int32, jobId, state string, contributed int64, lastModified
  time.Time}`
- `list_corporation_freelance_jobs` — private/corp-scoped, cursor-paginated, input `{characterId
  int32, corporationId int64, after, before string, limit int64}`, output `{characterId,
  corporationId, jobs []FreelanceJobSummary, nextAfter, nextBefore *string}`
- `list_corporation_freelance_job_participants` — private/corp-scoped, cursor-paginated, input
  `{characterId int32, corporationId int64, jobId string, after, before string, limit int64}`,
  output `{characterId, corporationId, jobId, participants []JobParticipant, nextAfter,
  nextBefore *string}` where `JobParticipant{ParticipantID int64, Name, State string, Contributed
  int64}`
- `get_freelance_job` — public, input `{jobId string}`, output the full detail: `{jobId, name,
  state string, lastModified time.Time, progress JobProgress, reward *JobReward, configuration
  JobConfiguration, contribution *JobContributionLimits, creator JobCreator, details JobDetails,
  accessAndVisibility JobAccessAndVisibility}` where `JobConfiguration{Method string, Parameters
  map[string]ConfigParameter, Version int64}` (the 4-variant flattened oneOf), `JobCreator{Character,
  Corporation JobCreatorEntity}` with `JobCreatorEntity{ID int64, Name string}`, `JobDetails{Career,
  Description string, Created time.Time, Expires, Finished *time.Time}`, `JobAccessAndVisibility{AclProtected
  bool, BroadcastLocations []JobBroadcastLocation, Restrictions *JobRestrictions}` where
  `JobBroadcastLocation{SystemID int64, Name string}` and `JobRestrictions{MinimumAge, MaximumAge
  *int64}`, `JobContributionLimits{ContributionPerParticipantLimit, SubmissionLimit *int64,
  MaxCommittedParticipants int64, RewardPerContribution, SubmissionMultiplier *float64}`
- `list_freelance_jobs` — public, cursor-paginated, input `{after, before string, limit,
  corporationId int64}` (all optional; `corporationId` filters to one corp's jobs), output `{jobs
  []FreelanceJobSummary, nextAfter, nextBefore *string}`

Shared flattened Configuration types: `ConfigParameter{Kind string, BooleanValue *bool,
CorporationItemDelivery *ParameterCorporationItemDelivery, Matcher *ParameterMatcher, Options
*ParameterOptions}`, `ParameterCorporationItemDelivery{CorporationOfficeLocation, ItemType
ParameterMatcher}`, `ParameterMatcher{Values []ParameterMatcherValue}`,
`ParameterMatcherValue{ValueType string, Values []string}`, `ParameterOptions{Selected []string}`
— each defined once, reused by `toConfigParameter`'s 4-way switch.

## Testing

Same three-layer isolation as every prior batch. `esi/freelance_jobs_test.go` covers all 6
methods (parsing + at least one error path), plus cursor-pagination omit/set tests for the 3
paginated methods (mirroring Corporation Projects' precedent) and a check that
`FreelanceJobsListing`'s optional `corporationId` filter is genuinely omitted when `0`.
`mcp/freelance_jobs_internal_test.go` covers happy-path for all 6 tools + unknown-character for
the 3 private tools, plus a dedicated test asserting `get_freelance_job`'s `toConfigParameter`
mapper correctly discriminates all 4 `Configuration.Parameters` variants (`Boolean`,
`CorporationItemDelivery`, `Matcher`, `Options`) including the nested double-`Matcher` inside
`CorporationItemDelivery`. `mcp/server_test.go` grows from 113 to 119.

## Out of scope

- No SDE cross-referencing of `configuration.method`/`parameters` keys (ESI's own doc comment
  says "see SDE for valid values/parameter definition" — this project doesn't resolve those here).
- No automatic multi-page/cursor aggregation, consistent with every other cursor-paginated tool.
