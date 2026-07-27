# ESI Industry (Eighteenth Batch) Design

## Status

Approved (autonomous execution). Eighteenth batch: the "Industry" tag alone (8 endpoints, mixed
public/private). Back to the classic page/`X-Pages` pagination style (not cursor) — the first
batch to use it since Corporation.

## Scope

**2 private, character-scoped** ("my own character," no separate target):
`GetCharactersCharacterIdIndustryJobs` (not paginated, takes an `includeCompleted bool` query
filter), `GetCharactersCharacterIdMining` (paginated).

**4 private, corp-scoped** (`characterId` for auth + `corporationId` for target, all
page-paginated): `GetCorporationCorporationIdMiningExtractions`,
`GetCorporationCorporationIdMiningObservers`, `GetCorporationCorporationIdMiningObserversObserverId`
(takes an additional `observerId`), `GetCorporationsCorporationIdIndustryJobs` (also takes
`includeCompleted bool`).

**2 public**: `GetIndustryFacilities`, `GetIndustrySystems` (neither paginated).

**Note on ESI's own inconsistent naming**: 3 of the 4 corp-scoped endpoints use the URL segment
`/corporation/{corporation_id}/...` (singular "corporation") while `GetCorporationsCorporationIdIndustryJobs`
uses `/corporations/{corporation_id}/...` (plural) — verified from the generated method names
themselves (`GetCorporationCorporationIdMiningExtractions` vs.
`GetCorporationsCorporationIdIndustryJobs`). This is ESI's own inconsistency, not a project typo —
neat's own facade method names are unaffected either way (`CorporationMiningExtractions`,
`CorporationIndustryJobs`, etc.), so this quirk is invisible to callers.

## Verified model shapes

`CharactersCharacterIdIndustryJobsGetInner`/`CorporationsCorporationIdIndustryJobsGetInner` are
nearly identical (activity/blueprint/facility/job/timing fields), with one asymmetry: the
character variant has `StationId int64` (required) but no `LocationId`; the corporation variant
has `LocationId int64` (required) but no `StationId`. Both models share the same optional fields
(`CompletedCharacterId`, `CompletedDate`, `Cost`, `LicensedRuns`, `PauseDate`, `Probability`,
`ProductTypeId`, `SuccessfulRuns` — all pointers).

`CharactersCharacterIdMiningGetInner{Date string, Quantity, SolarSystemId, TypeId int64}` (all
required — note `Date` is a plain string per ESI's own spec, not `time.Time`).
`CorporationCorporationIdMiningExtractionsGetInner{ChunkArrivalTime, ExtractionStartTime,
NaturalDecayTime time.Time, MoonId, StructureId int64}` (all required).
`CorporationCorporationIdMiningObserversGetInner{LastUpdated string, ObserverId int64,
ObserverType string}`. `CorporationCorporationIdMiningObserversObserverIdGetInner{CharacterId,
Quantity, RecordedCorporationId, TypeId int64, LastUpdated string}` — note `LastUpdated` is a
string here too (matching Mining's convention), not `time.Time`. `IndustryFacilitiesGetInner{FacilityId,
OwnerId, RegionId, SolarSystemId, TypeId int64, Tax *float64}`. `IndustrySystemsGetInner{SolarSystemId
int64, CostIndices []{Activity string, CostIndex float64}}`.

## `esi/industry.go` (8 methods)

```go
func (c *Client) CharacterIndustryJobs(ctx context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error)
func (c *Client) CharacterMining(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error)
func (c *Client) CorporationMiningExtractions(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error)
func (c *Client) CorporationMiningObservers(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error)
func (c *Client) CorporationMiningObserverDetail(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error)
func (c *Client) CorporationIndustryJobs(ctx context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error)
func (c *Client) IndustryFacilities(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error)
func (c *Client) IndustrySystems(ctx context.Context) ([]esiapi.IndustrySystemsGetInner, error)
```

`CharacterIndustryJobs`/`CorporationIndustryJobs`'s `includeCompleted` maps directly to the
generated `.IncludeCompleted(bool)` builder (always sent explicitly, no "omit when false"
semantics — this is a genuine boolean filter, not an ID/cursor to conditionally omit). All 4
paginated methods use the existing `totalPagesFromHeader` (`esi/contracts.go`) — reused, not
redeclared — with the same `page=0` omits `.Page()` convention.

## `mcp/industry.go` (8 tools)

Private (existence-checked via existing `characterExists`):
- `list_character_industry_jobs` — input `{characterId int32, includeCompleted bool}`, output `{characterId int32, includeCompleted bool, jobs []IndustryJob}` where `IndustryJob{ActivityID, BlueprintID, BlueprintLocationID, BlueprintTypeID, Duration, FacilityID, InstallerID, JobID, OutputLocationID, Runs, StationID int64, EndDate, StartDate time.Time, Status string, CompletedCharacterID, LicensedRuns, ProductTypeID, SuccessfulRuns *int64, CompletedDate, PauseDate *time.Time, Cost, Probability *float64}` — `StationID` populated for character jobs, `0` for corp jobs (asymmetric field, see below)
- `list_character_mining` — input `{characterId int32, page int32}`, output `{characterId, page, totalPages int32, ledger []MiningLedgerEntry}` where `MiningLedgerEntry{Date string, Quantity, SolarSystemID, TypeID int64}`
- `list_corporation_mining_extractions` — input `{characterId int32, corporationId int64, page int32}`, output `{..., extractions []MiningExtraction}` where `MiningExtraction{MoonID, StructureID int64, ChunkArrivalTime, ExtractionStartTime, NaturalDecayTime time.Time}`
- `list_corporation_mining_observers` — input `{characterId int32, corporationId int64, page int32}`, output `{..., observers []MiningObserver}` where `MiningObserver{ObserverID int64, LastUpdated, ObserverType string}`
- `get_corporation_mining_observer_detail` — input `{characterId int32, corporationId, observerId int64, page int32}`, output `{..., entries []MiningObserverEntry}` where `MiningObserverEntry{CharacterID, Quantity, RecordedCorporationID, TypeID int64, LastUpdated string}`
- `list_corporation_industry_jobs` — input `{characterId int32, corporationId int64, includeCompleted bool, page int32}`, output `{..., jobs []IndustryJob}` — reuses the same `IndustryJob` type as the character variant (`LocationID` populated for corp jobs, `0`/unset for character jobs — the asymmetric field flows both ways since `IndustryJob` is shared)

Public (no existence check):
- `list_industry_facilities` — input: none, output `{facilities []IndustryFacility}` where `IndustryFacility{FacilityID, OwnerID, RegionID, SolarSystemID, TypeID int64, Tax *float64}`
- `list_industry_systems` — input: none, output `{systems []IndustrySystem}` where `IndustrySystem{SolarSystemID int64, CostIndices []CostIndex}` and `CostIndex{Activity string, CostIndex float64}`

`IndustryJob` deliberately carries both `StationID` and `LocationID` fields (one populated,
one zero, depending on which of the two source models mapped it) rather than defining two
near-duplicate types — this mirrors how the two ESI models are 95% identical and avoids type
duplication for a one-field difference, while being explicit in the doc comment about which field
applies to which source.

## Testing

Same three-layer isolation as every prior batch. `esi/industry_test.go` covers all 8 methods
(parsing + at least one error path + pagination omit/set checks for the 4 paginated methods, plus
a check that `includeCompleted` is always sent as an explicit query value on both jobs
endpoints, never omitted). `mcp/industry_internal_test.go` covers happy-path for all 8 tools +
unknown-character for the 6 private tools. `mcp/server_test.go` grows from 120 to 128.

## Out of scope

- No automatic multi-page aggregation, consistent with every other paginated tool in this project.
