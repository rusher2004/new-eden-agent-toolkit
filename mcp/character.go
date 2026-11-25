package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Public: get_character ---

type GetCharacterOutput struct {
	CharacterID      int64     `json:"characterId"`
	Name             string    `json:"name"`
	CorporationID    int64     `json:"corporationId"`
	AllianceID       *int64    `json:"allianceId,omitempty"`
	FactionID        *int64    `json:"factionId,omitempty"`
	FactionName      string    `json:"factionName,omitempty"`
	Birthday         time.Time `json:"birthday"`
	BloodlineID      int64     `json:"bloodlineId"`
	BloodlineName    string    `json:"bloodlineName,omitempty"`
	RaceID           int64     `json:"raceId"`
	RaceName         string    `json:"raceName,omitempty"`
	Gender           string    `json:"gender"`
	SecurityStatus   *float64  `json:"securityStatus,omitempty"`
	Description      *string   `json:"description,omitempty"`
	Title            *string   `json:"title,omitempty"`
	AchievementScore int64     `json:"achievementScore"`
}

// characterDetailESI is the public ESI call get_character makes.
type characterDetailESI interface {
	CharacterDetail(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error)
}

// characterNameCache is the SDE lookups get_character's name enrichment makes.
type characterNameCache interface {
	raceNameCache
	bloodlineNameCache
	factionNameCache
}

func getCharacterDetail(ctx context.Context, client characterDetailESI, characterID int64) (GetCharacterOutput, error) {
	detail, err := client.CharacterDetail(ctx, characterID)
	if err != nil {
		return GetCharacterOutput{}, err
	}
	return GetCharacterOutput{
		CharacterID: characterID, Name: detail.Name, CorporationID: detail.CorporationId,
		AllianceID: detail.AllianceId, FactionID: detail.FactionId, Birthday: detail.Birthday,
		BloodlineID: detail.BloodlineId, RaceID: detail.RaceId, Gender: detail.Gender,
		SecurityStatus: detail.SecurityStatus, Description: detail.Description,
		Title: detail.CorporationTitle, AchievementScore: detail.AchievementScore,
	}, nil
}

type GetCharacterInput struct {
	CharacterID int64 `json:"characterId" jsonschema:"the character ID to look up"`
}

// AddGetCharacter registers get_character. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacter[C characterDetailESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore characterNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character",
		Description: "Get public info (name, corporation, bloodline, security status) about an arbitrary character (public data, no character required).",
	}, getCharacterHandler(userAgent, open, sdeStore))
}

func getCharacterHandler[C characterDetailESI](userAgent string, open func() C, sdeStore characterNameCache) mcpsdk.ToolHandlerFor[GetCharacterInput, GetCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterInput) (*mcpsdk.CallToolResult, GetCharacterOutput, error) {
		out, err := getCharacterDetail(ctx, open(), in.CharacterID)
		if err != nil {
			return nil, GetCharacterOutput{}, fmt.Errorf("get character: %w", err)
		}
		raceNames := resolveRaceNames(ctx, sdeStore, userAgent, []int64{out.RaceID})
		out.RaceName = raceNames[out.RaceID]
		bloodlineNames := resolveBloodlineNames(ctx, sdeStore, userAgent, []int64{out.BloodlineID})
		out.BloodlineName = bloodlineNames[out.BloodlineID]
		if out.FactionID != nil {
			factionNames := resolveFactionNames(ctx, sdeStore, userAgent, []int64{*out.FactionID})
			out.FactionName = factionNames[*out.FactionID]
		}
		return nil, out, nil
	}
}

// --- Public: get_character_affiliations ---

type CharacterAffiliation struct {
	CharacterID   int64  `json:"characterId"`
	CorporationID int64  `json:"corporationId"`
	AllianceID    *int64 `json:"allianceId,omitempty"`
	FactionID     *int64 `json:"factionId,omitempty"`
	FactionName   string `json:"factionName,omitempty"`
}

// characterAffiliationsESI is the public ESI call get_character_affiliations
// makes; the planner family reads affiliations through it too.
type characterAffiliationsESI interface {
	CharacterAffiliations(ctx context.Context, characterIDs []int64) ([]esiapi.CharactersAffiliationPostInner, error)
}

func getCharacterAffiliations(ctx context.Context, client characterAffiliationsESI, characterIDs []int64) ([]CharacterAffiliation, error) {
	affs, err := client.CharacterAffiliations(ctx, characterIDs)
	if err != nil {
		return nil, err
	}
	out := make([]CharacterAffiliation, len(affs))
	for i, a := range affs {
		out[i] = CharacterAffiliation{CharacterID: a.CharacterId, CorporationID: a.CorporationId, AllianceID: a.AllianceId, FactionID: a.FactionId}
	}
	return out, nil
}

type GetCharacterAffiliationsInput struct {
	CharacterIDs []int64 `json:"characterIds" jsonschema:"character IDs to resolve"`
}
type GetCharacterAffiliationsOutput struct {
	Affiliations []CharacterAffiliation `json:"affiliations"`
}

// AddGetCharacterAffiliations registers get_character_affiliations. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterAffiliations[C characterAffiliationsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore factionNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_affiliations",
		Description: "Resolve a set of character IDs to their current corporation/alliance/faction (public data, no character required).",
	}, getCharacterAffiliationsHandler(userAgent, open, sdeStore))
}

func getCharacterAffiliationsHandler[C characterAffiliationsESI](userAgent string, open func() C, sdeStore factionNameCache) mcpsdk.ToolHandlerFor[GetCharacterAffiliationsInput, GetCharacterAffiliationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterAffiliationsInput) (*mcpsdk.CallToolResult, GetCharacterAffiliationsOutput, error) {
		affs, err := getCharacterAffiliations(ctx, open(), in.CharacterIDs)
		if err != nil {
			return nil, GetCharacterAffiliationsOutput{}, fmt.Errorf("get character affiliations: %w", err)
		}
		var factionIDs []int64
		for _, a := range affs {
			if a.FactionID != nil {
				factionIDs = append(factionIDs, *a.FactionID)
			}
		}
		factionNames := resolveFactionNames(ctx, sdeStore, userAgent, factionIDs)
		for i := range affs {
			if affs[i].FactionID != nil {
				affs[i].FactionName = factionNames[*affs[i].FactionID]
			}
		}
		return nil, GetCharacterAffiliationsOutput{Affiliations: affs}, nil
	}
}

// --- Public: list_character_corporation_history ---

type CorporationHistoryEntry struct {
	CorporationID int64     `json:"corporationId"`
	RecordID      int64     `json:"recordId"`
	StartDate     time.Time `json:"startDate"`
	IsDeleted     *bool     `json:"isDeleted,omitempty"`
}

// characterCorporationHistoryESI is the public ESI call
// list_character_corporation_history makes.
type characterCorporationHistoryESI interface {
	CharacterCorporationHistory(ctx context.Context, characterID int64) ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, error)
}

func listCharacterCorporationHistory(ctx context.Context, client characterCorporationHistoryESI, characterID int64) ([]CorporationHistoryEntry, error) {
	hist, err := client.CharacterCorporationHistory(ctx, characterID)
	if err != nil {
		return nil, err
	}
	out := make([]CorporationHistoryEntry, len(hist))
	for i, h := range hist {
		out[i] = CorporationHistoryEntry{CorporationID: h.CorporationId, RecordID: h.RecordId, StartDate: h.StartDate, IsDeleted: h.IsDeleted}
	}
	return out, nil
}

type ListCharacterCorporationHistoryInput struct {
	CharacterID int64 `json:"characterId" jsonschema:"the character ID to look up"`
}
type ListCharacterCorporationHistoryOutput struct {
	CharacterID int64                     `json:"characterId"`
	History     []CorporationHistoryEntry `json:"history"`
}

// AddListCharacterCorporationHistory registers list_character_corporation_history. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterCorporationHistory[C characterCorporationHistoryESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_corporation_history",
		Description: "List an arbitrary character's corporation history (public data, no character required).",
	}, listCharacterCorporationHistoryHandler(open))
}

func listCharacterCorporationHistoryHandler[C characterCorporationHistoryESI](open func() C) mcpsdk.ToolHandlerFor[ListCharacterCorporationHistoryInput, ListCharacterCorporationHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterCorporationHistoryInput) (*mcpsdk.CallToolResult, ListCharacterCorporationHistoryOutput, error) {
		hist, err := listCharacterCorporationHistory(ctx, open(), in.CharacterID)
		if err != nil {
			return nil, ListCharacterCorporationHistoryOutput{}, fmt.Errorf("list character corporation history: %w", err)
		}
		return nil, ListCharacterCorporationHistoryOutput{CharacterID: in.CharacterID, History: hist}, nil
	}
}

// --- Public: get_character_portrait ---

type GetCharacterPortraitOutput struct {
	CharacterID int64   `json:"characterId"`
	Px64x64     *string `json:"px64x64,omitempty"`
	Px128x128   *string `json:"px128x128,omitempty"`
	Px256x256   *string `json:"px256x256,omitempty"`
	Px512x512   *string `json:"px512x512,omitempty"`
}

// characterPortraitESI is the public ESI call get_character_portrait makes.
type characterPortraitESI interface {
	CharacterPortrait(ctx context.Context, characterID int64) (*esiapi.CharactersCharacterIdPortraitGet, error)
}

func getCharacterPortrait(ctx context.Context, client characterPortraitESI, characterID int64) (GetCharacterPortraitOutput, error) {
	portrait, err := client.CharacterPortrait(ctx, characterID)
	if err != nil {
		return GetCharacterPortraitOutput{}, err
	}
	return GetCharacterPortraitOutput{
		CharacterID: characterID, Px64x64: portrait.Px64x64, Px128x128: portrait.Px128x128,
		Px256x256: portrait.Px256x256, Px512x512: portrait.Px512x512,
	}, nil
}

type GetCharacterPortraitInput struct {
	CharacterID int64 `json:"characterId" jsonschema:"the character ID to look up"`
}

// AddGetCharacterPortrait registers get_character_portrait. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterPortrait[C characterPortraitESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_portrait",
		Description: "Get an arbitrary character's portrait URLs (public data, no character required).",
	}, getCharacterPortraitHandler(open))
}

func getCharacterPortraitHandler[C characterPortraitESI](open func() C) mcpsdk.ToolHandlerFor[GetCharacterPortraitInput, GetCharacterPortraitOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterPortraitInput) (*mcpsdk.CallToolResult, GetCharacterPortraitOutput, error) {
		out, err := getCharacterPortrait(ctx, open(), in.CharacterID)
		if err != nil {
			return nil, GetCharacterPortraitOutput{}, fmt.Errorf("get character portrait: %w", err)
		}
		return nil, out, nil
	}
}

// --- shared existence-check helper for the private tools below ---

// characterExists reports whether characterID is a logged-in character in store. New in this
// file: with 10 private tools sharing the same check, duplicating the loop 10 times crosses the
// DRY threshold that made sense for smaller batches.
func characterExists(store configLoader, characterID int32) (bool, error) {
	cfg, err := store.Load()
	if err != nil {
		return false, fmt.Errorf("load config: %w", err)
	}
	for _, c := range cfg.Characters {
		if c.ID == characterID {
			return true, nil
		}
	}
	return false, nil
}

// --- Private: list_character_agents_research ---

type AgentResearch struct {
	AgentID         int64     `json:"agentId"`
	SkillTypeID     int64     `json:"skillTypeId"`
	SkillTypeName   string    `json:"skillTypeName,omitempty"`
	PointsPerDay    float64   `json:"pointsPerDay"`
	RemainderPoints float64   `json:"remainderPoints"`
	StartedAt       time.Time `json:"startedAt"`
}

// characterAgentsResearchESI is the ESI call list_character_agents_research makes.
type characterAgentsResearchESI interface {
	CharacterAgentsResearch(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error)
}

func listCharacterAgentsResearch(ctx context.Context, client characterAgentsResearchESI) ([]AgentResearch, error) {
	agents, err := client.CharacterAgentsResearch(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AgentResearch, len(agents))
	for i, a := range agents {
		out[i] = AgentResearch{AgentID: a.AgentId, SkillTypeID: a.SkillTypeId, PointsPerDay: a.PointsPerDay, RemainderPoints: a.RemainderPoints, StartedAt: a.StartedAt}
	}
	return out, nil
}

type ListCharacterAgentsResearchInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterAgentsResearchOutput struct {
	CharacterID int32           `json:"characterId"`
	Agents      []AgentResearch `json:"agents"`
}

// AddListCharacterAgentsResearch registers list_character_agents_research. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterAgentsResearch[C characterAgentsResearchESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_agents_research",
		Description: "List a logged-in EVE Online character's agent research progress.",
	}, listCharacterAgentsResearchHandler(userAgent, open, store, sdeStore))
}

func listCharacterAgentsResearchHandler[C characterAgentsResearchESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListCharacterAgentsResearchInput, ListCharacterAgentsResearchOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterAgentsResearchInput) (*mcpsdk.CallToolResult, ListCharacterAgentsResearchOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterAgentsResearchOutput{}, err
		}
		if !found {
			return nil, ListCharacterAgentsResearchOutput{}, fmt.Errorf("list character agents research for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterAgentsResearchOutput{}, fmt.Errorf("list character agents research: %w", err)
		}
		agents, err := listCharacterAgentsResearch(ctx, client)
		if err != nil {
			return nil, ListCharacterAgentsResearchOutput{}, fmt.Errorf("list character agents research: %w", err)
		}
		ids := make([]int64, len(agents))
		for i, a := range agents {
			ids[i] = a.SkillTypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range agents {
			agents[i].SkillTypeName = names[agents[i].SkillTypeID]
		}
		return nil, ListCharacterAgentsResearchOutput{CharacterID: in.CharacterID, Agents: agents}, nil
	}
}

// --- Private: list_character_blueprints ---

type Blueprint struct {
	ItemID             int64  `json:"itemId"`
	TypeID             int64  `json:"typeId"`
	TypeName           string `json:"typeName,omitempty"`
	LocationID         int64  `json:"locationId"`
	LocationFlag       string `json:"locationFlag"`
	Quantity           int64  `json:"quantity"`
	Runs               int64  `json:"runs"`
	MaterialEfficiency int64  `json:"materialEfficiency"`
	TimeEfficiency     int64  `json:"timeEfficiency"`
}

// characterBlueprintsESI is the ESI call list_character_blueprints makes;
// the planner family reads blueprints through it too.
type characterBlueprintsESI interface {
	CharacterBlueprints(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, int32, error)
}

func listCharacterBlueprints(ctx context.Context, client characterBlueprintsESI, page int32) ([]Blueprint, int32, error) {
	bps, totalPages, err := client.CharacterBlueprints(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Blueprint, len(bps))
	for i, b := range bps {
		out[i] = Blueprint{ItemID: b.ItemId, TypeID: b.TypeId, LocationID: b.LocationId, LocationFlag: b.LocationFlag, Quantity: b.Quantity, Runs: b.Runs, MaterialEfficiency: b.MaterialEfficiency, TimeEfficiency: b.TimeEfficiency}
	}
	return out, totalPages, nil
}

type ListCharacterBlueprintsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterBlueprintsOutput struct {
	CharacterID int32       `json:"characterId"`
	Page        int32       `json:"page"`
	TotalPages  int32       `json:"totalPages"`
	Blueprints  []Blueprint `json:"blueprints"`
}

// AddListCharacterBlueprints registers list_character_blueprints. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterBlueprints[C characterBlueprintsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_blueprints",
		Description: "List a logged-in EVE Online character's blueprints.",
	}, listCharacterBlueprintsHandler(userAgent, open, store, sdeStore))
}

func listCharacterBlueprintsHandler[C characterBlueprintsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListCharacterBlueprintsInput, ListCharacterBlueprintsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterBlueprintsInput) (*mcpsdk.CallToolResult, ListCharacterBlueprintsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterBlueprintsOutput{}, err
		}
		if !found {
			return nil, ListCharacterBlueprintsOutput{}, fmt.Errorf("list character blueprints for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterBlueprintsOutput{}, fmt.Errorf("list character blueprints: %w", err)
		}
		bps, totalPages, err := listCharacterBlueprints(ctx, client, in.Page)
		if err != nil {
			return nil, ListCharacterBlueprintsOutput{}, fmt.Errorf("list character blueprints: %w", err)
		}
		ids := make([]int64, len(bps))
		for i, b := range bps {
			ids[i] = b.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range bps {
			bps[i].TypeName = names[bps[i].TypeID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCharacterBlueprintsOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Blueprints: bps}, nil
	}
}

// --- Private: get_character_fatigue ---

type GetCharacterFatigueOutput struct {
	CharacterID           int32      `json:"characterId"`
	JumpFatigueExpireDate *time.Time `json:"jumpFatigueExpireDate,omitempty"`
	LastJumpDate          *time.Time `json:"lastJumpDate,omitempty"`
	LastUpdateDate        *time.Time `json:"lastUpdateDate,omitempty"`
}

// characterFatigueESI is the ESI call get_character_fatigue makes; the
// planner family reads fatigue through it too.
type characterFatigueESI interface {
	CharacterFatigue(ctx context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error)
}

func getCharacterFatigue(ctx context.Context, client characterFatigueESI, characterID int32) (GetCharacterFatigueOutput, error) {
	fatigue, err := client.CharacterFatigue(ctx)
	if err != nil {
		return GetCharacterFatigueOutput{}, err
	}
	return GetCharacterFatigueOutput{CharacterID: characterID, JumpFatigueExpireDate: fatigue.JumpFatigueExpireDate, LastJumpDate: fatigue.LastJumpDate, LastUpdateDate: fatigue.LastUpdateDate}, nil
}

type GetCharacterFatigueInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// AddGetCharacterFatigue registers get_character_fatigue. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterFatigue[C characterFatigueESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_fatigue",
		Description: "Get a logged-in EVE Online character's jump fatigue state.",
	}, getCharacterFatigueHandler(open, store))
}

func getCharacterFatigueHandler[C characterFatigueESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterFatigueInput, GetCharacterFatigueOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFatigueInput) (*mcpsdk.CallToolResult, GetCharacterFatigueOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFatigueOutput{}, err
		}
		if !found {
			return nil, GetCharacterFatigueOutput{}, fmt.Errorf("get character fatigue for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFatigueOutput{}, fmt.Errorf("get character fatigue: %w", err)
		}
		out, err := getCharacterFatigue(ctx, client, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFatigueOutput{}, fmt.Errorf("get character fatigue: %w", err)
		}
		return nil, out, nil
	}
}

// --- Private: list_character_medals ---

type MedalGraphic struct {
	Graphic string `json:"graphic"`
	Layer   int64  `json:"layer"`
	Part    int64  `json:"part"`
	Color   *int64 `json:"color,omitempty"`
}
type Medal struct {
	MedalID       int64          `json:"medalId"`
	CorporationID int64          `json:"corporationId"`
	IssuerID      int64          `json:"issuerId"`
	Date          time.Time      `json:"date"`
	Description   string         `json:"description"`
	Reason        string         `json:"reason"`
	Status        string         `json:"status"`
	Title         string         `json:"title"`
	Graphics      []MedalGraphic `json:"graphics"`
}

// characterMedalsESI is the ESI call list_character_medals makes.
type characterMedalsESI interface {
	CharacterMedals(ctx context.Context) ([]esiapi.CharactersCharacterIdMedalsGetInner, error)
}

func listCharacterMedals(ctx context.Context, client characterMedalsESI) ([]Medal, error) {
	medals, err := client.CharacterMedals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Medal, len(medals))
	for i, m := range medals {
		graphics := make([]MedalGraphic, len(m.Graphics))
		for j, g := range m.Graphics {
			graphics[j] = MedalGraphic{Graphic: g.Graphic, Layer: g.Layer, Part: g.Part, Color: g.Color}
		}
		out[i] = Medal{MedalID: m.MedalId, CorporationID: m.CorporationId, IssuerID: m.IssuerId, Date: m.Date, Description: m.Description, Reason: m.Reason, Status: m.Status, Title: m.Title, Graphics: graphics}
	}
	return out, nil
}

type ListCharacterMedalsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterMedalsOutput struct {
	CharacterID int32   `json:"characterId"`
	Medals      []Medal `json:"medals"`
}

// AddListCharacterMedals registers list_character_medals. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterMedals[C characterMedalsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_medals",
		Description: "List a logged-in EVE Online character's medals.",
	}, listCharacterMedalsHandler(open, store))
}

func listCharacterMedalsHandler[C characterMedalsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterMedalsInput, ListCharacterMedalsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMedalsInput) (*mcpsdk.CallToolResult, ListCharacterMedalsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMedalsOutput{}, err
		}
		if !found {
			return nil, ListCharacterMedalsOutput{}, fmt.Errorf("list character medals for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMedalsOutput{}, fmt.Errorf("list character medals: %w", err)
		}
		medals, err := listCharacterMedals(ctx, client)
		if err != nil {
			return nil, ListCharacterMedalsOutput{}, fmt.Errorf("list character medals: %w", err)
		}
		return nil, ListCharacterMedalsOutput{CharacterID: in.CharacterID, Medals: medals}, nil
	}
}

// --- Private: list_character_notifications ---

type Notification struct {
	NotificationID int64     `json:"notificationId"`
	SenderID       int64     `json:"senderId"`
	SenderType     string    `json:"senderType"`
	Type           string    `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	IsRead         *bool     `json:"isRead,omitempty"`
	Text           *string   `json:"text,omitempty"`
}

// characterNotificationsESI is the ESI call list_character_notifications makes.
type characterNotificationsESI interface {
	CharacterNotifications(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsGetInner, error)
}

func listCharacterNotifications(ctx context.Context, client characterNotificationsESI) ([]Notification, error) {
	notifs, err := client.CharacterNotifications(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Notification, len(notifs))
	for i, n := range notifs {
		out[i] = Notification{NotificationID: n.NotificationId, SenderID: n.SenderId, SenderType: n.SenderType, Type: n.Type, Timestamp: n.Timestamp, IsRead: n.IsRead, Text: n.Text}
	}
	return out, nil
}

type ListCharacterNotificationsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterNotificationsOutput struct {
	CharacterID   int32          `json:"characterId"`
	Notifications []Notification `json:"notifications"`
}

// AddListCharacterNotifications registers list_character_notifications. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterNotifications[C characterNotificationsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_notifications",
		Description: "List a logged-in EVE Online character's notifications.",
	}, listCharacterNotificationsHandler(open, store))
}

func listCharacterNotificationsHandler[C characterNotificationsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterNotificationsInput, ListCharacterNotificationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterNotificationsInput) (*mcpsdk.CallToolResult, ListCharacterNotificationsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterNotificationsOutput{}, err
		}
		if !found {
			return nil, ListCharacterNotificationsOutput{}, fmt.Errorf("list character notifications for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterNotificationsOutput{}, fmt.Errorf("list character notifications: %w", err)
		}
		notifs, err := listCharacterNotifications(ctx, client)
		if err != nil {
			return nil, ListCharacterNotificationsOutput{}, fmt.Errorf("list character notifications: %w", err)
		}
		return nil, ListCharacterNotificationsOutput{CharacterID: in.CharacterID, Notifications: notifs}, nil
	}
}

// --- Private: list_character_notification_contacts ---

type ContactNotification struct {
	NotificationID    int64     `json:"notificationId"`
	SenderCharacterID int64     `json:"senderCharacterId"`
	SendDate          time.Time `json:"sendDate"`
	StandingLevel     float64   `json:"standingLevel"`
	Message           string    `json:"message"`
}

// characterNotificationContactsESI is the ESI call list_character_notification_contacts makes.
type characterNotificationContactsESI interface {
	CharacterNotificationContacts(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, error)
}

func listCharacterNotificationContacts(ctx context.Context, client characterNotificationContactsESI) ([]ContactNotification, error) {
	notifs, err := client.CharacterNotificationContacts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ContactNotification, len(notifs))
	for i, n := range notifs {
		out[i] = ContactNotification{NotificationID: n.NotificationId, SenderCharacterID: n.SenderCharacterId, SendDate: n.SendDate, StandingLevel: n.StandingLevel, Message: n.Message}
	}
	return out, nil
}

type ListCharacterNotificationContactsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterNotificationContactsOutput struct {
	CharacterID   int32                 `json:"characterId"`
	Notifications []ContactNotification `json:"notifications"`
}

// AddListCharacterNotificationContacts registers list_character_notification_contacts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterNotificationContacts[C characterNotificationContactsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_notification_contacts",
		Description: "List a logged-in EVE Online character's contact-standing notifications.",
	}, listCharacterNotificationContactsHandler(open, store))
}

func listCharacterNotificationContactsHandler[C characterNotificationContactsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterNotificationContactsInput, ListCharacterNotificationContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterNotificationContactsInput) (*mcpsdk.CallToolResult, ListCharacterNotificationContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterNotificationContactsOutput{}, err
		}
		if !found {
			return nil, ListCharacterNotificationContactsOutput{}, fmt.Errorf("list character notification contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterNotificationContactsOutput{}, fmt.Errorf("list character notification contacts: %w", err)
		}
		notifs, err := listCharacterNotificationContacts(ctx, client)
		if err != nil {
			return nil, ListCharacterNotificationContactsOutput{}, fmt.Errorf("list character notification contacts: %w", err)
		}
		return nil, ListCharacterNotificationContactsOutput{CharacterID: in.CharacterID, Notifications: notifs}, nil
	}
}

// --- Private: get_character_roles ---

type GetCharacterRolesOutput struct {
	CharacterID  int32    `json:"characterId"`
	Roles        []string `json:"roles,omitempty"`
	RolesAtBase  []string `json:"rolesAtBase,omitempty"`
	RolesAtHq    []string `json:"rolesAtHq,omitempty"`
	RolesAtOther []string `json:"rolesAtOther,omitempty"`
}

// characterRolesESI is the ESI call get_character_roles makes.
type characterRolesESI interface {
	CharacterRoles(ctx context.Context) (*esiapi.CharactersCharacterIdRolesGet, error)
}

func getCharacterRoles(ctx context.Context, client characterRolesESI, characterID int32) (GetCharacterRolesOutput, error) {
	roles, err := client.CharacterRoles(ctx)
	if err != nil {
		return GetCharacterRolesOutput{}, err
	}
	return GetCharacterRolesOutput{CharacterID: characterID, Roles: roles.Roles, RolesAtBase: roles.RolesAtBase, RolesAtHq: roles.RolesAtHq, RolesAtOther: roles.RolesAtOther}, nil
}

type GetCharacterRolesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// AddGetCharacterRoles registers get_character_roles. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterRoles[C characterRolesESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_roles",
		Description: "Get a logged-in EVE Online character's corporation roles.",
	}, getCharacterRolesHandler(open, store))
}

func getCharacterRolesHandler[C characterRolesESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterRolesInput, GetCharacterRolesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterRolesInput) (*mcpsdk.CallToolResult, GetCharacterRolesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterRolesOutput{}, err
		}
		if !found {
			return nil, GetCharacterRolesOutput{}, fmt.Errorf("get character roles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterRolesOutput{}, fmt.Errorf("get character roles: %w", err)
		}
		out, err := getCharacterRoles(ctx, client, in.CharacterID)
		if err != nil {
			return nil, GetCharacterRolesOutput{}, fmt.Errorf("get character roles: %w", err)
		}
		return nil, out, nil
	}
}

// --- Private: list_character_standings ---

type Standing struct {
	FromID   int64   `json:"fromId"`
	FromType string  `json:"fromType"`
	Standing float64 `json:"standing"`
}

// characterStandingsESI is the ESI call list_character_standings makes.
type characterStandingsESI interface {
	CharacterStandings(ctx context.Context) ([]esiapi.CharactersCharacterIdStandingsGetInner, error)
}

func listCharacterStandings(ctx context.Context, client characterStandingsESI) ([]Standing, error) {
	standings, err := client.CharacterStandings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Standing, len(standings))
	for i, s := range standings {
		out[i] = Standing{FromID: s.FromId, FromType: s.FromType, Standing: s.Standing}
	}
	return out, nil
}

type ListCharacterStandingsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterStandingsOutput struct {
	CharacterID int32      `json:"characterId"`
	Standings   []Standing `json:"standings"`
}

// AddListCharacterStandings registers list_character_standings. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterStandings[C characterStandingsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_standings",
		Description: "List a logged-in EVE Online character's standings.",
	}, listCharacterStandingsHandler(open, store))
}

func listCharacterStandingsHandler[C characterStandingsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterStandingsInput, ListCharacterStandingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterStandingsInput) (*mcpsdk.CallToolResult, ListCharacterStandingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterStandingsOutput{}, err
		}
		if !found {
			return nil, ListCharacterStandingsOutput{}, fmt.Errorf("list character standings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterStandingsOutput{}, fmt.Errorf("list character standings: %w", err)
		}
		standings, err := listCharacterStandings(ctx, client)
		if err != nil {
			return nil, ListCharacterStandingsOutput{}, fmt.Errorf("list character standings: %w", err)
		}
		return nil, ListCharacterStandingsOutput{CharacterID: in.CharacterID, Standings: standings}, nil
	}
}

// --- Private: list_character_titles ---

type Title struct {
	TitleID *int64  `json:"titleId,omitempty"`
	Name    *string `json:"name,omitempty"`
}

// characterTitlesESI is the ESI call list_character_titles makes.
type characterTitlesESI interface {
	CharacterTitles(ctx context.Context) ([]esiapi.CharactersCharacterIdTitlesGetInner, error)
}

func listCharacterTitles(ctx context.Context, client characterTitlesESI) ([]Title, error) {
	titles, err := client.CharacterTitles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Title, len(titles))
	for i, t := range titles {
		out[i] = Title{TitleID: t.TitleId, Name: t.Name}
	}
	return out, nil
}

type ListCharacterTitlesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterTitlesOutput struct {
	CharacterID int32   `json:"characterId"`
	Titles      []Title `json:"titles"`
}

// AddListCharacterTitles registers list_character_titles. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterTitles[C characterTitlesESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_titles",
		Description: "List a logged-in EVE Online character's corporation titles.",
	}, listCharacterTitlesHandler(open, store))
}

func listCharacterTitlesHandler[C characterTitlesESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterTitlesInput, ListCharacterTitlesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterTitlesInput) (*mcpsdk.CallToolResult, ListCharacterTitlesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterTitlesOutput{}, err
		}
		if !found {
			return nil, ListCharacterTitlesOutput{}, fmt.Errorf("list character titles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterTitlesOutput{}, fmt.Errorf("list character titles: %w", err)
		}
		titles, err := listCharacterTitles(ctx, client)
		if err != nil {
			return nil, ListCharacterTitlesOutput{}, fmt.Errorf("list character titles: %w", err)
		}
		return nil, ListCharacterTitlesOutput{CharacterID: in.CharacterID, Titles: titles}, nil
	}
}

// --- Private: get_character_cspa_cost ---

// characterCSPACostESI is the ESI call get_character_cspa_cost makes.
type characterCSPACostESI interface {
	CharacterCSPACost(ctx context.Context, targetCharacterIDs []int64) (float64, error)
}

func getCharacterCSPACost(ctx context.Context, client characterCSPACostESI, targetCharacterIDs []int64) (float64, error) {
	return client.CharacterCSPACost(ctx, targetCharacterIDs)
}

type GetCharacterCSPACostInput struct {
	CharacterID        int32   `json:"characterId" jsonschema:"the character ID to check"`
	TargetCharacterIDs []int64 `json:"targetCharacterIds" jsonschema:"target character IDs to calculate CSPA cost for"`
}
type GetCharacterCSPACostOutput struct {
	CharacterID int32   `json:"characterId"`
	Cost        float64 `json:"cost"`
}

// AddGetCharacterCSPACost registers get_character_cspa_cost. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCharacterCSPACost[C characterCSPACostESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_cspa_cost",
		Description: "Calculate the CSPA charge cost for a logged-in character to contact a set of target characters.",
	}, getCharacterCSPACostHandler(open, store))
}

func getCharacterCSPACostHandler[C characterCSPACostESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterCSPACostInput, GetCharacterCSPACostOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterCSPACostInput) (*mcpsdk.CallToolResult, GetCharacterCSPACostOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterCSPACostOutput{}, err
		}
		if !found {
			return nil, GetCharacterCSPACostOutput{}, fmt.Errorf("get character cspa cost for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterCSPACostOutput{}, fmt.Errorf("get character cspa cost: %w", err)
		}
		cost, err := getCharacterCSPACost(ctx, client, in.TargetCharacterIDs)
		if err != nil {
			return nil, GetCharacterCSPACostOutput{}, fmt.Errorf("get character cspa cost: %w", err)
		}
		return nil, GetCharacterCSPACostOutput{CharacterID: in.CharacterID, Cost: cost}, nil
	}
}
