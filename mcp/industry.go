package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// IndustryJob is shared by list_character_industry_jobs and list_corporation_industry_jobs --
// the two underlying ESI models are 95% identical, differing only in StationID (character-scoped
// jobs) vs LocationID (corp-scoped jobs). Only the applicable field is populated; the other is
// left at its zero value.
type IndustryJob struct {
	ActivityID           int64      `json:"activityId"`
	BlueprintID          int64      `json:"blueprintId"`
	BlueprintLocationID  int64      `json:"blueprintLocationId"`
	BlueprintTypeID      int64      `json:"blueprintTypeId"`
	BlueprintTypeName    string     `json:"blueprintTypeName,omitempty"`
	Duration             int64      `json:"duration"`
	EndDate              time.Time  `json:"endDate"`
	FacilityID           int64      `json:"facilityId"`
	InstallerID          int64      `json:"installerId"`
	JobID                int64      `json:"jobId"`
	OutputLocationID     int64      `json:"outputLocationId"`
	Runs                 int64      `json:"runs"`
	StartDate            time.Time  `json:"startDate"`
	Status               string     `json:"status"`
	StationID            int64      `json:"stationId,omitempty"`
	StationName          string     `json:"stationName,omitempty"`
	LocationID           int64      `json:"locationId,omitempty"`
	CompletedCharacterID *int64     `json:"completedCharacterId,omitempty"`
	CompletedDate        *time.Time `json:"completedDate,omitempty"`
	Cost                 *float64   `json:"cost,omitempty"`
	LicensedRuns         *int64     `json:"licensedRuns,omitempty"`
	PauseDate            *time.Time `json:"pauseDate,omitempty"`
	Probability          *float64   `json:"probability,omitempty"`
	ProductTypeID        *int64     `json:"productTypeId,omitempty"`
	ProductTypeName      string     `json:"productTypeName,omitempty"`
	SuccessfulRuns       *int64     `json:"successfulRuns,omitempty"`
}

func toIndustryJobFromCharacter(j esiapi.CharactersCharacterIdIndustryJobsGetInner) IndustryJob {
	return IndustryJob{
		ActivityID: j.ActivityId, BlueprintID: j.BlueprintId, BlueprintLocationID: j.BlueprintLocationId,
		BlueprintTypeID: j.BlueprintTypeId, Duration: j.Duration, EndDate: j.EndDate, FacilityID: j.FacilityId,
		InstallerID: j.InstallerId, JobID: j.JobId, OutputLocationID: j.OutputLocationId, Runs: j.Runs,
		StartDate: j.StartDate, Status: j.Status, StationID: j.StationId,
		CompletedCharacterID: j.CompletedCharacterId, CompletedDate: j.CompletedDate, Cost: j.Cost,
		LicensedRuns: j.LicensedRuns, PauseDate: j.PauseDate, Probability: j.Probability,
		ProductTypeID: j.ProductTypeId, SuccessfulRuns: j.SuccessfulRuns,
	}
}

func toIndustryJobFromCorporation(j esiapi.CorporationsCorporationIdIndustryJobsGetInner) IndustryJob {
	return IndustryJob{
		ActivityID: j.ActivityId, BlueprintID: j.BlueprintId, BlueprintLocationID: j.BlueprintLocationId,
		BlueprintTypeID: j.BlueprintTypeId, Duration: j.Duration, EndDate: j.EndDate, FacilityID: j.FacilityId,
		InstallerID: j.InstallerId, JobID: j.JobId, OutputLocationID: j.OutputLocationId, Runs: j.Runs,
		StartDate: j.StartDate, Status: j.Status, LocationID: j.LocationId,
		CompletedCharacterID: j.CompletedCharacterId, CompletedDate: j.CompletedDate, Cost: j.Cost,
		LicensedRuns: j.LicensedRuns, PauseDate: j.PauseDate, Probability: j.Probability,
		ProductTypeID: j.ProductTypeId, SuccessfulRuns: j.SuccessfulRuns,
	}
}

// --- list_character_industry_jobs (private) ---

// characterIndustryJobNameCache is the SDE lookups list_character_industry_jobs's
// name enrichment makes.
type characterIndustryJobNameCache interface {
	typeNameCache
	stationNameCache
}

// characterIndustryJobsESI is the ESI call list_character_industry_jobs
// makes; the industry planning tools read jobs through it too.
type characterIndustryJobsESI interface {
	CharacterIndustryJobs(ctx context.Context, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error)
}

func listCharacterIndustryJobs(ctx context.Context, client characterIndustryJobsESI, includeCompleted bool) ([]esiapi.CharactersCharacterIdIndustryJobsGetInner, error) {
	return client.CharacterIndustryJobs(ctx, includeCompleted)
}

type ListCharacterIndustryJobsInput struct {
	CharacterID      int32 `json:"characterId" jsonschema:"the character ID to check"`
	IncludeCompleted bool  `json:"includeCompleted,omitempty" jsonschema:"whether to include completed jobs"`
}
type ListCharacterIndustryJobsOutput struct {
	CharacterID      int32         `json:"characterId"`
	IncludeCompleted bool          `json:"includeCompleted"`
	Jobs             []IndustryJob `json:"jobs"`
}

// industryJobNameCache is the SDE lookups the industry job tools make: type
// and station names.
type industryJobNameCache interface {
	typeNameCache
	stationNameCache
}

// AddListCharacterIndustryJobs registers list_character_industry_jobs. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterIndustryJobs[C characterIndustryJobsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore industryJobNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_industry_jobs",
		Description: "List a logged-in EVE Online character's industry jobs.",
	}, listCharacterIndustryJobsHandler(userAgent, open, store, sdeStore))
}

func listCharacterIndustryJobsHandler[C characterIndustryJobsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore industryJobNameCache) mcpsdk.ToolHandlerFor[ListCharacterIndustryJobsInput, ListCharacterIndustryJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterIndustryJobsInput) (*mcpsdk.CallToolResult, ListCharacterIndustryJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterIndustryJobsOutput{}, err
		}
		if !found {
			return nil, ListCharacterIndustryJobsOutput{}, fmt.Errorf("list character industry jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterIndustryJobsOutput{}, fmt.Errorf("list character industry jobs: %w", err)
		}
		jobs, err := listCharacterIndustryJobs(ctx, client, in.IncludeCompleted)
		if err != nil {
			return nil, ListCharacterIndustryJobsOutput{}, fmt.Errorf("list character industry jobs: %w", err)
		}
		out := make([]IndustryJob, len(jobs))
		for i, j := range jobs {
			out[i] = toIndustryJobFromCharacter(j)
		}
		enrichIndustryJobs(ctx, out, sdeStore, userAgent)
		stationIDs := make([]int64, 0, len(out))
		for _, j := range out {
			if j.StationID != 0 {
				stationIDs = append(stationIDs, j.StationID)
			}
		}
		stationNames := resolveStationNames(ctx, sdeStore, userAgent, stationIDs)
		for i := range out {
			if out[i].StationID != 0 {
				out[i].StationName = stationNames[out[i].StationID]
			}
		}
		return nil, ListCharacterIndustryJobsOutput{CharacterID: in.CharacterID, IncludeCompleted: in.IncludeCompleted, Jobs: out}, nil
	}
}

// enrichIndustryJobs resolves BlueprintTypeName (always populated) and ProductTypeName (only when
// ProductTypeID is non-nil) in place, sharing one resolveTypeNames batch across both fields.
func enrichIndustryJobs(ctx context.Context, jobs []IndustryJob, sdeStore typeNameCache, userAgent string) {
	ids := make([]int64, 0, len(jobs)*2)
	for _, j := range jobs {
		ids = append(ids, j.BlueprintTypeID)
		if j.ProductTypeID != nil {
			ids = append(ids, *j.ProductTypeID)
		}
	}
	names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
	for i := range jobs {
		jobs[i].BlueprintTypeName = names[jobs[i].BlueprintTypeID]
		if jobs[i].ProductTypeID != nil {
			jobs[i].ProductTypeName = names[*jobs[i].ProductTypeID]
		}
	}
}

// --- list_character_mining (private, paginated) ---

// miningNameCache is the SDE lookups list_character_mining's name enrichment makes.
type miningNameCache interface {
	typeNameCache
	systemNameCache
}

type MiningLedgerEntry struct {
	Date            string `json:"date"`
	Quantity        int64  `json:"quantity"`
	SolarSystemID   int64  `json:"solarSystemId"`
	SolarSystemName string `json:"solarSystemName,omitempty"`
	TypeID          int64  `json:"typeId"`
	TypeName        string `json:"typeName,omitempty"`
}

// characterMiningESI is the ESI call list_character_mining makes.
type characterMiningESI interface {
	CharacterMining(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error)
}

func listCharacterMining(ctx context.Context, client characterMiningESI, page int32) ([]esiapi.CharactersCharacterIdMiningGetInner, int32, error) {
	return client.CharacterMining(ctx, page)
}

type ListCharacterMiningInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterMiningOutput struct {
	CharacterID int32               `json:"characterId"`
	Page        int32               `json:"page"`
	TotalPages  int32               `json:"totalPages"`
	Ledger      []MiningLedgerEntry `json:"ledger"`
}

// AddListCharacterMining registers list_character_mining. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterMining[C characterMiningESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore miningNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_mining",
		Description: "List a logged-in EVE Online character's mining ledger.",
	}, listCharacterMiningHandler(userAgent, open, store, sdeStore))
}

func listCharacterMiningHandler[C characterMiningESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore miningNameCache) mcpsdk.ToolHandlerFor[ListCharacterMiningInput, ListCharacterMiningOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMiningInput) (*mcpsdk.CallToolResult, ListCharacterMiningOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMiningOutput{}, err
		}
		if !found {
			return nil, ListCharacterMiningOutput{}, fmt.Errorf("list character mining for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMiningOutput{}, fmt.Errorf("list character mining: %w", err)
		}
		ledger, totalPages, err := listCharacterMining(ctx, client, in.Page)
		if err != nil {
			return nil, ListCharacterMiningOutput{}, fmt.Errorf("list character mining: %w", err)
		}
		out := make([]MiningLedgerEntry, len(ledger))
		for i, e := range ledger {
			out[i] = MiningLedgerEntry{Date: e.Date, Quantity: e.Quantity, SolarSystemID: e.SolarSystemId, TypeID: e.TypeId}
		}
		ids := make([]int64, len(out))
		for i, e := range out {
			ids[i] = e.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		systemIDs := make([]int64, len(out))
		for i, e := range out {
			systemIDs[i] = e.SolarSystemID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		for i := range out {
			out[i].SolarSystemName = systemNames[out[i].SolarSystemID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCharacterMiningOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Ledger: out}, nil
	}
}

// --- list_corporation_mining_extractions (private, paginated) ---

type MiningExtraction struct {
	MoonID              int64     `json:"moonId"`
	MoonName            string    `json:"moonName,omitempty"`
	StructureID         int64     `json:"structureId"`
	ChunkArrivalTime    time.Time `json:"chunkArrivalTime"`
	ExtractionStartTime time.Time `json:"extractionStartTime"`
	NaturalDecayTime    time.Time `json:"naturalDecayTime"`
}

// corporationMiningExtractionsESI is the ESI call list_corporation_mining_extractions makes.
type corporationMiningExtractionsESI interface {
	CorporationMiningExtractions(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error)
}

func listCorporationMiningExtractions(ctx context.Context, client corporationMiningExtractionsESI, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningExtractionsGetInner, int32, error) {
	return client.CorporationMiningExtractions(ctx, corporationID, page)
}

type ListCorporationMiningExtractionsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMiningExtractionsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	Page          int32              `json:"page"`
	TotalPages    int32              `json:"totalPages"`
	Extractions   []MiningExtraction `json:"extractions"`
}

// AddListCorporationMiningExtractions registers list_corporation_mining_extractions. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationMiningExtractions[C corporationMiningExtractionsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore moonNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_mining_extractions",
		Description: "List a corporation's moon mining extractions, using a logged-in character's roles to authorize.",
	}, listCorporationMiningExtractionsHandler(userAgent, open, store, sdeStore))
}

func listCorporationMiningExtractionsHandler[C corporationMiningExtractionsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore moonNameCache) mcpsdk.ToolHandlerFor[ListCorporationMiningExtractionsInput, ListCorporationMiningExtractionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMiningExtractionsInput) (*mcpsdk.CallToolResult, ListCorporationMiningExtractionsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMiningExtractionsOutput{}, err
		}
		if !found {
			return nil, ListCorporationMiningExtractionsOutput{}, fmt.Errorf("list corporation mining extractions for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMiningExtractionsOutput{}, fmt.Errorf("list corporation mining extractions: %w", err)
		}
		extractions, totalPages, err := listCorporationMiningExtractions(ctx, client, in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMiningExtractionsOutput{}, fmt.Errorf("list corporation mining extractions: %w", err)
		}
		out := make([]MiningExtraction, len(extractions))
		for i, e := range extractions {
			out[i] = MiningExtraction{
				MoonID: e.MoonId, StructureID: e.StructureId, ChunkArrivalTime: e.ChunkArrivalTime,
				ExtractionStartTime: e.ExtractionStartTime, NaturalDecayTime: e.NaturalDecayTime,
			}
		}
		moonIDs := make([]int64, len(out))
		for i, e := range out {
			moonIDs[i] = e.MoonID
		}
		moonNames := resolveMoonNames(ctx, sdeStore, userAgent, moonIDs)
		for i := range out {
			out[i].MoonName = moonNames[out[i].MoonID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationMiningExtractionsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Extractions: out}, nil
	}
}

// --- list_corporation_mining_observers (private, paginated) ---

type MiningObserver struct {
	ObserverID   int64  `json:"observerId"`
	LastUpdated  string `json:"lastUpdated"`
	ObserverType string `json:"observerType"`
}

// corporationMiningObserversESI is the ESI call list_corporation_mining_observers makes.
type corporationMiningObserversESI interface {
	CorporationMiningObservers(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error)
}

func listCorporationMiningObservers(ctx context.Context, client corporationMiningObserversESI, corporationID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversGetInner, int32, error) {
	return client.CorporationMiningObservers(ctx, corporationID, page)
}

type ListCorporationMiningObserversInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMiningObserversOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	Page          int32            `json:"page"`
	TotalPages    int32            `json:"totalPages"`
	Observers     []MiningObserver `json:"observers"`
}

// AddListCorporationMiningObservers registers list_corporation_mining_observers. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationMiningObservers[C corporationMiningObserversESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_mining_observers",
		Description: "List a corporation's mining observers, using a logged-in character's roles to authorize.",
	}, listCorporationMiningObserversHandler(open, store))
}

func listCorporationMiningObserversHandler[C corporationMiningObserversESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationMiningObserversInput, ListCorporationMiningObserversOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMiningObserversInput) (*mcpsdk.CallToolResult, ListCorporationMiningObserversOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMiningObserversOutput{}, err
		}
		if !found {
			return nil, ListCorporationMiningObserversOutput{}, fmt.Errorf("list corporation mining observers for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMiningObserversOutput{}, fmt.Errorf("list corporation mining observers: %w", err)
		}
		observers, totalPages, err := listCorporationMiningObservers(ctx, client, in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMiningObserversOutput{}, fmt.Errorf("list corporation mining observers: %w", err)
		}
		out := make([]MiningObserver, len(observers))
		for i, o := range observers {
			out[i] = MiningObserver{ObserverID: o.ObserverId, LastUpdated: o.LastUpdated, ObserverType: o.ObserverType}
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationMiningObserversOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Observers: out}, nil
	}
}

// --- get_corporation_mining_observer_detail (private, paginated) ---

type MiningObserverEntry struct {
	CharacterID           int64  `json:"characterId"`
	LastUpdated           string `json:"lastUpdated"`
	Quantity              int64  `json:"quantity"`
	RecordedCorporationID int64  `json:"recordedCorporationId"`
	TypeID                int64  `json:"typeId"`
	TypeName              string `json:"typeName,omitempty"`
}

// corporationMiningObserverDetailESI is the ESI call get_corporation_mining_observer_detail makes.
type corporationMiningObserverDetailESI interface {
	CorporationMiningObserverDetail(ctx context.Context, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error)
}

func getCorporationMiningObserverDetail(ctx context.Context, client corporationMiningObserverDetailESI, corporationID, observerID int64, page int32) ([]esiapi.CorporationCorporationIdMiningObserversObserverIdGetInner, int32, error) {
	return client.CorporationMiningObserverDetail(ctx, corporationID, observerID, page)
}

type GetCorporationMiningObserverDetailInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ObserverID    int64 `json:"observerId" jsonschema:"the observer ID, from list_corporation_mining_observers"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type GetCorporationMiningObserverDetailOutput struct {
	CharacterID   int32                 `json:"characterId"`
	CorporationID int64                 `json:"corporationId"`
	ObserverID    int64                 `json:"observerId"`
	Page          int32                 `json:"page"`
	TotalPages    int32                 `json:"totalPages"`
	Entries       []MiningObserverEntry `json:"entries"`
}

// AddGetCorporationMiningObserverDetail registers get_corporation_mining_observer_detail. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCorporationMiningObserverDetail[C corporationMiningObserverDetailESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_corporation_mining_observer_detail",
		Description: "Get the mining ledger recorded by one of a corporation's mining observers, using a logged-in character's roles to authorize.",
	}, getCorporationMiningObserverDetailHandler(userAgent, open, store, sdeStore))
}

func getCorporationMiningObserverDetailHandler[C corporationMiningObserverDetailESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[GetCorporationMiningObserverDetailInput, GetCorporationMiningObserverDetailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationMiningObserverDetailInput) (*mcpsdk.CallToolResult, GetCorporationMiningObserverDetailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationMiningObserverDetailOutput{}, err
		}
		if !found {
			return nil, GetCorporationMiningObserverDetailOutput{}, fmt.Errorf("get corporation mining observer detail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCorporationMiningObserverDetailOutput{}, fmt.Errorf("get corporation mining observer detail: %w", err)
		}
		entries, totalPages, err := getCorporationMiningObserverDetail(ctx, client, in.CorporationID, in.ObserverID, in.Page)
		if err != nil {
			return nil, GetCorporationMiningObserverDetailOutput{}, fmt.Errorf("get corporation mining observer detail: %w", err)
		}
		out := make([]MiningObserverEntry, len(entries))
		for i, e := range entries {
			out[i] = MiningObserverEntry{
				CharacterID: e.CharacterId, LastUpdated: e.LastUpdated, Quantity: e.Quantity,
				RecordedCorporationID: e.RecordedCorporationId, TypeID: e.TypeId,
			}
		}
		ids := make([]int64, len(out))
		for i, e := range out {
			ids[i] = e.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, GetCorporationMiningObserverDetailOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, ObserverID: in.ObserverID,
			Page: page, TotalPages: totalPages, Entries: out,
		}, nil
	}
}

// --- list_corporation_industry_jobs (private, paginated) ---

// corporationIndustryJobsESI is the ESI call list_corporation_industry_jobs
// makes; the industry planning tools read jobs through it too.
type corporationIndustryJobsESI interface {
	CorporationIndustryJobs(ctx context.Context, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error)
}

func listCorporationIndustryJobs(ctx context.Context, client corporationIndustryJobsESI, corporationID int64, includeCompleted bool, page int32) ([]esiapi.CorporationsCorporationIdIndustryJobsGetInner, int32, error) {
	return client.CorporationIndustryJobs(ctx, corporationID, includeCompleted, page)
}

type ListCorporationIndustryJobsInput struct {
	CharacterID      int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID    int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	IncludeCompleted bool  `json:"includeCompleted,omitempty" jsonschema:"whether to include completed jobs"`
	Page             int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationIndustryJobsOutput struct {
	CharacterID      int32         `json:"characterId"`
	CorporationID    int64         `json:"corporationId"`
	IncludeCompleted bool          `json:"includeCompleted"`
	Page             int32         `json:"page"`
	TotalPages       int32         `json:"totalPages"`
	Jobs             []IndustryJob `json:"jobs"`
}

// AddListCorporationIndustryJobs registers list_corporation_industry_jobs. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationIndustryJobs[C corporationIndustryJobsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore industryJobNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_industry_jobs",
		Description: "List a corporation's industry jobs, using a logged-in character's roles to authorize.",
	}, listCorporationIndustryJobsHandler(userAgent, open, store, sdeStore))
}

func listCorporationIndustryJobsHandler[C corporationIndustryJobsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore industryJobNameCache) mcpsdk.ToolHandlerFor[ListCorporationIndustryJobsInput, ListCorporationIndustryJobsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationIndustryJobsInput) (*mcpsdk.CallToolResult, ListCorporationIndustryJobsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationIndustryJobsOutput{}, err
		}
		if !found {
			return nil, ListCorporationIndustryJobsOutput{}, fmt.Errorf("list corporation industry jobs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationIndustryJobsOutput{}, fmt.Errorf("list corporation industry jobs: %w", err)
		}
		jobs, totalPages, err := listCorporationIndustryJobs(ctx, client, in.CorporationID, in.IncludeCompleted, in.Page)
		if err != nil {
			return nil, ListCorporationIndustryJobsOutput{}, fmt.Errorf("list corporation industry jobs: %w", err)
		}
		out := make([]IndustryJob, len(jobs))
		for i, j := range jobs {
			out[i] = toIndustryJobFromCorporation(j)
		}
		enrichIndustryJobs(ctx, out, sdeStore, userAgent)
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationIndustryJobsOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, IncludeCompleted: in.IncludeCompleted,
			Page: page, TotalPages: totalPages, Jobs: out,
		}, nil
	}
}

// --- list_industry_facilities (public, no input) ---

type IndustryFacility struct {
	FacilityID      int64    `json:"facilityId"`
	OwnerID         int64    `json:"ownerId"`
	RegionID        int64    `json:"regionId"`
	RegionName      string   `json:"regionName,omitempty"`
	SolarSystemID   int64    `json:"solarSystemId"`
	SolarSystemName string   `json:"solarSystemName,omitempty"`
	TypeID          int64    `json:"typeId"`
	TypeName        string   `json:"typeName,omitempty"`
	Tax             *float64 `json:"tax,omitempty"`
}

// industryFacilitiesESI is the public ESI call list_industry_facilities makes.
type industryFacilitiesESI interface {
	IndustryFacilities(ctx context.Context) ([]esiapi.IndustryFacilitiesGetInner, error)
}

// industryFacilityNameCache is the SDE lookups list_industry_facilities's name
// enrichment makes.
type industryFacilityNameCache interface {
	typeNameCache
	regionNameCache
	systemNameCache
}

func listIndustryFacilities(ctx context.Context, client industryFacilitiesESI) ([]esiapi.IndustryFacilitiesGetInner, error) {
	return client.IndustryFacilities(ctx)
}

type ListIndustryFacilitiesInput struct{}
type ListIndustryFacilitiesOutput struct {
	Facilities []IndustryFacility `json:"facilities"`
}

// AddListIndustryFacilities registers list_industry_facilities. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListIndustryFacilities[C industryFacilitiesESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore industryFacilityNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_industry_facilities",
		Description: "List every industry facility in the game (public data, no character required).",
	}, listIndustryFacilitiesHandler(userAgent, open, sdeStore))
}

func listIndustryFacilitiesHandler[C industryFacilitiesESI](userAgent string, open func() C, sdeStore industryFacilityNameCache) mcpsdk.ToolHandlerFor[ListIndustryFacilitiesInput, ListIndustryFacilitiesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIndustryFacilitiesInput) (*mcpsdk.CallToolResult, ListIndustryFacilitiesOutput, error) {
		facilities, err := listIndustryFacilities(ctx, open())
		if err != nil {
			return nil, ListIndustryFacilitiesOutput{}, fmt.Errorf("list industry facilities: %w", err)
		}
		out := make([]IndustryFacility, len(facilities))
		for i, f := range facilities {
			out[i] = IndustryFacility{
				FacilityID: f.FacilityId, OwnerID: f.OwnerId, RegionID: f.RegionId,
				SolarSystemID: f.SolarSystemId, TypeID: f.TypeId, Tax: f.Tax,
			}
		}
		ids := make([]int64, len(out))
		for i, f := range out {
			ids[i] = f.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		regionIDs := make([]int64, len(out))
		systemIDs := make([]int64, len(out))
		for i, f := range out {
			regionIDs[i] = f.RegionID
			systemIDs[i] = f.SolarSystemID
		}
		regionNames := resolveRegionNames(ctx, sdeStore, userAgent, regionIDs)
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		for i := range out {
			out[i].RegionName = regionNames[out[i].RegionID]
			out[i].SolarSystemName = systemNames[out[i].SolarSystemID]
		}
		return nil, ListIndustryFacilitiesOutput{Facilities: out}, nil
	}
}

// --- list_industry_systems (public, no input) ---

type CostIndex struct {
	Activity  string  `json:"activity"`
	CostIndex float64 `json:"costIndex"`
}

type IndustrySystem struct {
	SolarSystemID   int64       `json:"solarSystemId"`
	SolarSystemName string      `json:"solarSystemName,omitempty"`
	CostIndices     []CostIndex `json:"costIndices"`
}

// industrySystemsESI is the public ESI call list_industry_systems makes;
// the industry planning tools read cost indices through it too.
type industrySystemsESI interface {
	IndustrySystems(ctx context.Context) ([]esiapi.IndustrySystemsGetInner, error)
}

func listIndustrySystems(ctx context.Context, client industrySystemsESI) ([]esiapi.IndustrySystemsGetInner, error) {
	return client.IndustrySystems(ctx)
}

type ListIndustrySystemsInput struct{}
type ListIndustrySystemsOutput struct {
	Systems []IndustrySystem `json:"systems"`
}

// AddListIndustrySystems registers list_industry_systems. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListIndustrySystems[C industrySystemsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore systemNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_industry_systems",
		Description: "List industry cost indices for every solar system (public data, no character required).",
	}, listIndustrySystemsHandler(userAgent, open, sdeStore))
}

func listIndustrySystemsHandler[C industrySystemsESI](userAgent string, open func() C, sdeStore systemNameCache) mcpsdk.ToolHandlerFor[ListIndustrySystemsInput, ListIndustrySystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIndustrySystemsInput) (*mcpsdk.CallToolResult, ListIndustrySystemsOutput, error) {
		systems, err := listIndustrySystems(ctx, open())
		if err != nil {
			return nil, ListIndustrySystemsOutput{}, fmt.Errorf("list industry systems: %w", err)
		}
		out := make([]IndustrySystem, len(systems))
		for i, s := range systems {
			costIndices := make([]CostIndex, len(s.CostIndices))
			for j, ci := range s.CostIndices {
				costIndices[j] = CostIndex{Activity: ci.Activity, CostIndex: ci.CostIndex}
			}
			out[i] = IndustrySystem{SolarSystemID: s.SolarSystemId, CostIndices: costIndices}
		}
		systemIDs := make([]int64, len(out))
		for i, s := range out {
			systemIDs[i] = s.SolarSystemID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		for i := range out {
			out[i].SolarSystemName = systemNames[out[i].SolarSystemID]
		}
		return nil, ListIndustrySystemsOutput{Systems: out}, nil
	}
}
