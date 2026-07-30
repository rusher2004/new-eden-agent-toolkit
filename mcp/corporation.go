package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_corporation (public) ---

type GetCorporationOutput struct {
	CorporationID       int64      `json:"corporationId"`
	Name                string     `json:"name"`
	Ticker              string     `json:"ticker"`
	Description         string     `json:"description"`
	CeoID               *int64     `json:"ceoId,omitempty"`
	AllianceID          *int64     `json:"allianceId,omitempty"`
	CreatorID           *int64     `json:"creatorId,omitempty"`
	DateFounded         *time.Time `json:"dateFounded,omitempty"`
	FactionID           *int64     `json:"factionId,omitempty"`
	HomeStationID       int64      `json:"homeStationId"`
	MemberCount         int64      `json:"memberCount"`
	Shares              int64      `json:"shares"`
	TaxRateIsk          float64    `json:"taxRateIsk"`
	TaxRateLoyaltyPoint float64    `json:"taxRateLoyaltyPoint"`
	MainColor           string     `json:"mainColor,omitempty"`
	SecondaryColor      *string    `json:"secondaryColor,omitempty"`
	TertiaryColor       *string    `json:"tertiaryColor,omitempty"`
	URL                 *string    `json:"url,omitempty"`
	FriendlyFire        string     `json:"friendlyFire"`
	State               string     `json:"state"`
	Type                string     `json:"type"`
}

var getCorporationDetail = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationOutput, error) {
	client := esi.NewPublicClient(userAgent)
	detail, err := client.CorporationDetail(ctx, corporationID)
	if err != nil {
		return GetCorporationOutput{}, err
	}
	out := GetCorporationOutput{
		CorporationID: corporationID, Name: detail.Name, Ticker: detail.Ticker, Description: detail.Description,
		CeoID: detail.CeoId, AllianceID: detail.AllianceId, CreatorID: detail.CreatorId, DateFounded: detail.DateFounded,
		FactionID: detail.EnlistedFactionId, HomeStationID: detail.HomeStationId, MemberCount: detail.MemberCount,
		Shares: detail.Shares, TaxRateIsk: detail.TaxRates.Isk, TaxRateLoyaltyPoint: detail.TaxRates.LoyaltyPoint,
		URL: detail.Url, FriendlyFire: detail.FriendlyFire, State: detail.State, Type: detail.Type,
	}
	if detail.Palette != nil {
		out.MainColor = detail.Palette.MainColor
		out.SecondaryColor = detail.Palette.SecondaryColor
		out.TertiaryColor = detail.Palette.TertiaryColor
	}
	return out, nil
}

type GetCorporationInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}

func getCorporationHandler(version string) mcpsdk.ToolHandlerFor[GetCorporationInput, GetCorporationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationInput) (*mcpsdk.CallToolResult, GetCorporationOutput, error) {
		out, err := getCorporationDetail(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationOutput{}, fmt.Errorf("get corporation: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_corporation_alliance_history (public) ---

type AllianceHistoryEntry struct {
	AllianceID *int64    `json:"allianceId,omitempty"`
	RecordID   int64     `json:"recordId"`
	StartDate  time.Time `json:"startDate"`
	IsDeleted  *bool     `json:"isDeleted,omitempty"`
}

var listCorporationAllianceHistory = func(ctx context.Context, userAgent string, corporationID int64) ([]AllianceHistoryEntry, error) {
	client := esi.NewPublicClient(userAgent)
	hist, err := client.CorporationAllianceHistory(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]AllianceHistoryEntry, len(hist))
	for i, h := range hist {
		out[i] = AllianceHistoryEntry{AllianceID: h.AllianceId, RecordID: h.RecordId, StartDate: h.StartDate, IsDeleted: h.IsDeleted}
	}
	return out, nil
}

type ListCorporationAllianceHistoryInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationAllianceHistoryOutput struct {
	CorporationID int64                  `json:"corporationId"`
	History       []AllianceHistoryEntry `json:"history"`
}

func listCorporationAllianceHistoryHandler(version string) mcpsdk.ToolHandlerFor[ListCorporationAllianceHistoryInput, ListCorporationAllianceHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationAllianceHistoryInput) (*mcpsdk.CallToolResult, ListCorporationAllianceHistoryOutput, error) {
		hist, err := listCorporationAllianceHistory(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationAllianceHistoryOutput{}, fmt.Errorf("list corporation alliance history: %w", err)
		}
		return nil, ListCorporationAllianceHistoryOutput{CorporationID: in.CorporationID, History: hist}, nil
	}
}

// --- get_corporation_icons (public) ---

type GetCorporationIconsOutput struct {
	CorporationID int64   `json:"corporationId"`
	Px64x64       *string `json:"px64x64,omitempty"`
	Px128x128     *string `json:"px128x128,omitempty"`
	Px256x256     *string `json:"px256x256,omitempty"`
}

var getCorporationIcons = func(ctx context.Context, userAgent string, corporationID int64) (GetCorporationIconsOutput, error) {
	client := esi.NewPublicClient(userAgent)
	icons, err := client.CorporationIcons(ctx, corporationID)
	if err != nil {
		return GetCorporationIconsOutput{}, err
	}
	return GetCorporationIconsOutput{CorporationID: corporationID, Px64x64: icons.Px64x64, Px128x128: icons.Px128x128, Px256x256: icons.Px256x256}, nil
}

type GetCorporationIconsInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}

func getCorporationIconsHandler(version string) mcpsdk.ToolHandlerFor[GetCorporationIconsInput, GetCorporationIconsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationIconsInput) (*mcpsdk.CallToolResult, GetCorporationIconsOutput, error) {
		out, err := getCorporationIcons(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationIconsOutput{}, fmt.Errorf("get corporation icons: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_npc_corporations (public, no input) ---

var listNPCCorporations = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.NPCCorporations(ctx)
}

type ListNPCCorporationsInput struct{}
type ListNPCCorporationsOutput struct {
	CorporationIDs []int64 `json:"corporationIds"`
}

func listNPCCorporationsHandler(version string) mcpsdk.ToolHandlerFor[ListNPCCorporationsInput, ListNPCCorporationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListNPCCorporationsInput) (*mcpsdk.CallToolResult, ListNPCCorporationsOutput, error) {
		ids, err := listNPCCorporations(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListNPCCorporationsOutput{}, fmt.Errorf("list npc corporations: %w", err)
		}
		return nil, ListNPCCorporationsOutput{CorporationIDs: ids}, nil
	}
}

// --- list_corporation_members (private) ---

var listCorporationMembers = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMembers(ctx, corporationID)
}

type ListCorporationMembersInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationMembersOutput struct {
	CharacterID   int32   `json:"characterId"`
	CorporationID int64   `json:"corporationId"`
	MemberIDs     []int64 `json:"memberIds"`
}

func listCorporationMembersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMembersInput, ListCorporationMembersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMembersInput) (*mcpsdk.CallToolResult, ListCorporationMembersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMembersOutput{}, err
		}
		if !found {
			return nil, ListCorporationMembersOutput{}, fmt.Errorf("list corporation members for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		ids, err := listCorporationMembers(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationMembersOutput{}, fmt.Errorf("list corporation members: %w", err)
		}
		return nil, ListCorporationMembersOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, MemberIDs: ids}, nil
	}
}

// --- get_corporation_members_limit (private) ---

var getCorporationMembersLimit = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMembersLimit(ctx, corporationID)
}

type GetCorporationMembersLimitInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type GetCorporationMembersLimitOutput struct {
	CharacterID   int32 `json:"characterId"`
	CorporationID int64 `json:"corporationId"`
	Limit         int64 `json:"limit"`
}

func getCorporationMembersLimitHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationMembersLimitInput, GetCorporationMembersLimitOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationMembersLimitInput) (*mcpsdk.CallToolResult, GetCorporationMembersLimitOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationMembersLimitOutput{}, err
		}
		if !found {
			return nil, GetCorporationMembersLimitOutput{}, fmt.Errorf("get corporation members limit for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		limit, err := getCorporationMembersLimit(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationMembersLimitOutput{}, fmt.Errorf("get corporation members limit: %w", err)
		}
		return nil, GetCorporationMembersLimitOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Limit: limit}, nil
	}
}

// --- list_corporation_member_titles (private) ---

type MemberTitles struct {
	CharacterID int64   `json:"characterId"`
	TitleIDs    []int64 `json:"titleIds"`
}

var listCorporationMemberTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTitles, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	titles, err := client.CorporationMemberTitles(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberTitles, len(titles))
	for i, t := range titles {
		out[i] = MemberTitles{CharacterID: t.CharacterId, TitleIDs: t.Titles}
	}
	return out, nil
}

type ListCorporationMemberTitlesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationMemberTitlesOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	MemberTitles  []MemberTitles `json:"memberTitles"`
}

func listCorporationMemberTitlesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMemberTitlesInput, ListCorporationMemberTitlesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMemberTitlesInput) (*mcpsdk.CallToolResult, ListCorporationMemberTitlesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMemberTitlesOutput{}, err
		}
		if !found {
			return nil, ListCorporationMemberTitlesOutput{}, fmt.Errorf("list corporation member titles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		titles, err := listCorporationMemberTitles(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationMemberTitlesOutput{}, fmt.Errorf("list corporation member titles: %w", err)
		}
		return nil, ListCorporationMemberTitlesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, MemberTitles: titles}, nil
	}
}

// --- list_corporation_member_tracking (private) ---

type MemberTracking struct {
	CharacterID  int64      `json:"characterId"`
	BaseID       *int64     `json:"baseId,omitempty"`
	LocationID   *int64     `json:"locationId,omitempty"`
	ShipTypeID   *int64     `json:"shipTypeId,omitempty"`
	ShipTypeName string     `json:"shipTypeName,omitempty"`
	LogonDate    *time.Time `json:"logonDate,omitempty"`
	LogoffDate   *time.Time `json:"logoffDate,omitempty"`
	StartDate    *time.Time `json:"startDate,omitempty"`
}

var listCorporationMemberTracking = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberTracking, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	tracking, err := client.CorporationMemberTracking(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberTracking, len(tracking))
	for i, t := range tracking {
		out[i] = MemberTracking{CharacterID: t.CharacterId, BaseID: t.BaseId, LocationID: t.LocationId, ShipTypeID: t.ShipTypeId, LogonDate: t.LogonDate, LogoffDate: t.LogoffDate, StartDate: t.StartDate}
	}
	return out, nil
}

type ListCorporationMemberTrackingInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationMemberTrackingOutput struct {
	CharacterID   int32            `json:"characterId"`
	CorporationID int64            `json:"corporationId"`
	Members       []MemberTracking `json:"members"`
}

func listCorporationMemberTrackingHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationMemberTrackingInput, ListCorporationMemberTrackingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMemberTrackingInput) (*mcpsdk.CallToolResult, ListCorporationMemberTrackingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMemberTrackingOutput{}, err
		}
		if !found {
			return nil, ListCorporationMemberTrackingOutput{}, fmt.Errorf("list corporation member tracking for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		members, err := listCorporationMemberTracking(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationMemberTrackingOutput{}, fmt.Errorf("list corporation member tracking: %w", err)
		}
		var ids []int64
		for i := range members {
			if members[i].ShipTypeID != nil {
				ids = append(ids, *members[i].ShipTypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range members {
			if members[i].ShipTypeID != nil {
				members[i].ShipTypeName = names[*members[i].ShipTypeID]
			}
		}
		return nil, ListCorporationMemberTrackingOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Members: members}, nil
	}
}

// --- list_corporation_roles (private) ---

type MemberRoles struct {
	CharacterID           int64    `json:"characterId"`
	Roles                 []string `json:"roles,omitempty"`
	RolesAtBase           []string `json:"rolesAtBase,omitempty"`
	RolesAtHq             []string `json:"rolesAtHq,omitempty"`
	RolesAtOther          []string `json:"rolesAtOther,omitempty"`
	GrantableRoles        []string `json:"grantableRoles,omitempty"`
	GrantableRolesAtBase  []string `json:"grantableRolesAtBase,omitempty"`
	GrantableRolesAtHq    []string `json:"grantableRolesAtHq,omitempty"`
	GrantableRolesAtOther []string `json:"grantableRolesAtOther,omitempty"`
}

var listCorporationRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]MemberRoles, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	roles, err := client.CorporationRoles(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberRoles, len(roles))
	for i, r := range roles {
		out[i] = MemberRoles{
			CharacterID: r.CharacterId, Roles: r.Roles, RolesAtBase: r.RolesAtBase, RolesAtHq: r.RolesAtHq, RolesAtOther: r.RolesAtOther,
			GrantableRoles: r.GrantableRoles, GrantableRolesAtBase: r.GrantableRolesAtBase, GrantableRolesAtHq: r.GrantableRolesAtHq, GrantableRolesAtOther: r.GrantableRolesAtOther,
		}
	}
	return out, nil
}

type ListCorporationRolesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationRolesOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Roles         []MemberRoles `json:"roles"`
}

func listCorporationRolesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationRolesInput, ListCorporationRolesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationRolesInput) (*mcpsdk.CallToolResult, ListCorporationRolesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationRolesOutput{}, err
		}
		if !found {
			return nil, ListCorporationRolesOutput{}, fmt.Errorf("list corporation roles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		roles, err := listCorporationRoles(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationRolesOutput{}, fmt.Errorf("list corporation roles: %w", err)
		}
		return nil, ListCorporationRolesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Roles: roles}, nil
	}
}

// --- list_corporation_roles_history (private, paginated) ---

type RoleChange struct {
	CharacterID int64     `json:"characterId"`
	IssuerID    int64     `json:"issuerId"`
	ChangedAt   time.Time `json:"changedAt"`
	RoleType    string    `json:"roleType"`
	OldRoles    []string  `json:"oldRoles"`
	NewRoles    []string  `json:"newRoles"`
}

var listCorporationRolesHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]RoleChange, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	history, totalPages, err := client.CorporationRolesHistory(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]RoleChange, len(history))
	for i, h := range history {
		out[i] = RoleChange{CharacterID: h.CharacterId, IssuerID: h.IssuerId, ChangedAt: h.ChangedAt, RoleType: h.RoleType, OldRoles: h.OldRoles, NewRoles: h.NewRoles}
	}
	return out, totalPages, nil
}

type ListCorporationRolesHistoryInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationRolesHistoryOutput struct {
	CharacterID   int32        `json:"characterId"`
	CorporationID int64        `json:"corporationId"`
	Page          int32        `json:"page"`
	TotalPages    int32        `json:"totalPages"`
	History       []RoleChange `json:"history"`
}

func listCorporationRolesHistoryHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationRolesHistoryInput, ListCorporationRolesHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationRolesHistoryInput) (*mcpsdk.CallToolResult, ListCorporationRolesHistoryOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationRolesHistoryOutput{}, err
		}
		if !found {
			return nil, ListCorporationRolesHistoryOutput{}, fmt.Errorf("list corporation roles history for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		history, totalPages, err := listCorporationRolesHistory(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationRolesHistoryOutput{}, fmt.Errorf("list corporation roles history: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationRolesHistoryOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, History: history}, nil
	}
}

// --- list_corporation_titles (private) ---

type CorporationTitle struct {
	TitleID *int64  `json:"titleId,omitempty"`
	Name    *string `json:"name,omitempty"`
}

var listCorporationTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]CorporationTitle, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	titles, err := client.CorporationTitles(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]CorporationTitle, len(titles))
	for i, t := range titles {
		out[i] = CorporationTitle{TitleID: t.TitleId, Name: t.Name}
	}
	return out, nil
}

type ListCorporationTitlesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationTitlesOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	Titles        []CorporationTitle `json:"titles"`
}

func listCorporationTitlesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationTitlesInput, ListCorporationTitlesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationTitlesInput) (*mcpsdk.CallToolResult, ListCorporationTitlesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationTitlesOutput{}, err
		}
		if !found {
			return nil, ListCorporationTitlesOutput{}, fmt.Errorf("list corporation titles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		titles, err := listCorporationTitles(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationTitlesOutput{}, fmt.Errorf("list corporation titles: %w", err)
		}
		return nil, ListCorporationTitlesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Titles: titles}, nil
	}
}

// --- get_corporation_divisions (private) ---

type Division struct {
	DivisionNumber *int64  `json:"divisionNumber,omitempty"`
	Name           *string `json:"name,omitempty"`
}

var getCorporationDivisions = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Division, []Division, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	divisions, err := client.CorporationDivisions(ctx, corporationID)
	if err != nil {
		return nil, nil, err
	}
	hangars := make([]Division, len(divisions.Hangar))
	for i, h := range divisions.Hangar {
		hangars[i] = Division{DivisionNumber: h.Division, Name: h.Name}
	}
	wallets := make([]Division, len(divisions.Wallet))
	for i, w := range divisions.Wallet {
		wallets[i] = Division{DivisionNumber: w.Division, Name: w.Name}
	}
	return hangars, wallets, nil
}

type GetCorporationDivisionsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type GetCorporationDivisionsOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Hangars       []Division `json:"hangars"`
	Wallets       []Division `json:"wallets"`
}

func getCorporationDivisionsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationDivisionsInput, GetCorporationDivisionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationDivisionsInput) (*mcpsdk.CallToolResult, GetCorporationDivisionsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationDivisionsOutput{}, err
		}
		if !found {
			return nil, GetCorporationDivisionsOutput{}, fmt.Errorf("get corporation divisions for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		hangars, wallets, err := getCorporationDivisions(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationDivisionsOutput{}, fmt.Errorf("get corporation divisions: %w", err)
		}
		return nil, GetCorporationDivisionsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Hangars: hangars, Wallets: wallets}, nil
	}
}

// --- list_corporation_shareholders (private, paginated) ---

type Shareholder struct {
	ShareholderID   int64  `json:"shareholderId"`
	ShareholderType string `json:"shareholderType"`
	ShareCount      int64  `json:"shareCount"`
}

var listCorporationShareholders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Shareholder, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	shareholders, totalPages, err := client.CorporationShareholders(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Shareholder, len(shareholders))
	for i, s := range shareholders {
		out[i] = Shareholder{ShareholderID: s.ShareholderId, ShareholderType: s.ShareholderType, ShareCount: s.ShareCount}
	}
	return out, totalPages, nil
}

type ListCorporationShareholdersInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationShareholdersOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Shareholders  []Shareholder `json:"shareholders"`
}

func listCorporationShareholdersHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationShareholdersInput, ListCorporationShareholdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationShareholdersInput) (*mcpsdk.CallToolResult, ListCorporationShareholdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationShareholdersOutput{}, err
		}
		if !found {
			return nil, ListCorporationShareholdersOutput{}, fmt.Errorf("list corporation shareholders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		shareholders, totalPages, err := listCorporationShareholders(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationShareholdersOutput{}, fmt.Errorf("list corporation shareholders: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationShareholdersOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Shareholders: shareholders}, nil
	}
}

// --- list_corporation_standings (private, paginated) ---
// Reuses the existing Standing type from mcp/character.go -- same shape (FromID, FromType,
// Standing), same underlying ESI model (CharactersCharacterIdStandingsGetInner). Not redeclared.

var listCorporationStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Standing, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	standings, totalPages, err := client.CorporationStandings(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Standing, len(standings))
	for i, s := range standings {
		out[i] = Standing{FromID: s.FromId, FromType: s.FromType, Standing: s.Standing}
	}
	return out, totalPages, nil
}

type ListCorporationStandingsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationStandingsOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Page          int32      `json:"page"`
	TotalPages    int32      `json:"totalPages"`
	Standings     []Standing `json:"standings"`
}

func listCorporationStandingsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationStandingsInput, ListCorporationStandingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationStandingsInput) (*mcpsdk.CallToolResult, ListCorporationStandingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationStandingsOutput{}, err
		}
		if !found {
			return nil, ListCorporationStandingsOutput{}, fmt.Errorf("list corporation standings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		standings, totalPages, err := listCorporationStandings(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationStandingsOutput{}, fmt.Errorf("list corporation standings: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationStandingsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Standings: standings}, nil
	}
}

// --- list_corporation_blueprints (private, paginated) ---

type CorporationBlueprint struct {
	ItemID             int64  `json:"itemId"`
	LocationFlag       string `json:"locationFlag"`
	LocationID         int64  `json:"locationId"`
	MaterialEfficiency int64  `json:"materialEfficiency"`
	Quantity           int64  `json:"quantity"`
	Runs               int64  `json:"runs"`
	TimeEfficiency     int64  `json:"timeEfficiency"`
	TypeID             int64  `json:"typeId"`
	TypeName           string `json:"typeName,omitempty"`
}

var listCorporationBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationBlueprint, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	bps, totalPages, err := client.CorporationBlueprints(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CorporationBlueprint, len(bps))
	for i, b := range bps {
		out[i] = CorporationBlueprint{
			ItemID: b.ItemId, LocationFlag: b.LocationFlag, LocationID: b.LocationId,
			MaterialEfficiency: b.MaterialEfficiency, Quantity: b.Quantity, Runs: b.Runs,
			TimeEfficiency: b.TimeEfficiency, TypeID: b.TypeId,
		}
	}
	return out, totalPages, nil
}

type ListCorporationBlueprintsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationBlueprintsOutput struct {
	CharacterID   int32                  `json:"characterId"`
	CorporationID int64                  `json:"corporationId"`
	Page          int32                  `json:"page"`
	TotalPages    int32                  `json:"totalPages"`
	Blueprints    []CorporationBlueprint `json:"blueprints"`
}

func listCorporationBlueprintsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationBlueprintsInput, ListCorporationBlueprintsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationBlueprintsInput) (*mcpsdk.CallToolResult, ListCorporationBlueprintsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationBlueprintsOutput{}, err
		}
		if !found {
			return nil, ListCorporationBlueprintsOutput{}, fmt.Errorf("list corporation blueprints for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		bps, totalPages, err := listCorporationBlueprints(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationBlueprintsOutput{}, fmt.Errorf("list corporation blueprints: %w", err)
		}
		ids := make([]int64, len(bps))
		for i := range bps {
			ids[i] = bps[i].TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range bps {
			bps[i].TypeName = names[bps[i].TypeID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationBlueprintsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Blueprints: bps}, nil
	}
}

// --- list_corporation_container_logs (private, paginated) ---

type ContainerLog struct {
	Action            string    `json:"action"`
	CharacterID       int64     `json:"characterId"`
	ContainerID       int64     `json:"containerId"`
	ContainerTypeID   int64     `json:"containerTypeId"`
	ContainerTypeName string    `json:"containerTypeName,omitempty"`
	LocationFlag      string    `json:"locationFlag"`
	LocationID        int64     `json:"locationId"`
	LoggedAt          time.Time `json:"loggedAt"`
	NewConfigBitmask  *int64    `json:"newConfigBitmask,omitempty"`
	OldConfigBitmask  *int64    `json:"oldConfigBitmask,omitempty"`
	PasswordType      *string   `json:"passwordType,omitempty"`
	Quantity          *int64    `json:"quantity,omitempty"`
	TypeID            *int64    `json:"typeId,omitempty"`
	TypeName          string    `json:"typeName,omitempty"`
}

var listCorporationContainerLogs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]ContainerLog, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	logs, totalPages, err := client.CorporationContainerLogs(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContainerLog, len(logs))
	for i, l := range logs {
		out[i] = ContainerLog{
			Action: l.Action, CharacterID: l.CharacterId, ContainerID: l.ContainerId, ContainerTypeID: l.ContainerTypeId,
			LocationFlag: l.LocationFlag, LocationID: l.LocationId, LoggedAt: l.LoggedAt,
			NewConfigBitmask: l.NewConfigBitmask, OldConfigBitmask: l.OldConfigBitmask,
			PasswordType: l.PasswordType, Quantity: l.Quantity, TypeID: l.TypeId,
		}
	}
	return out, totalPages, nil
}

type ListCorporationContainerLogsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationContainerLogsOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	Page          int32          `json:"page"`
	TotalPages    int32          `json:"totalPages"`
	Logs          []ContainerLog `json:"logs"`
}

func listCorporationContainerLogsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationContainerLogsInput, ListCorporationContainerLogsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContainerLogsInput) (*mcpsdk.CallToolResult, ListCorporationContainerLogsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContainerLogsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContainerLogsOutput{}, fmt.Errorf("list corporation container logs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		logs, totalPages, err := listCorporationContainerLogs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationContainerLogsOutput{}, fmt.Errorf("list corporation container logs: %w", err)
		}
		var ids []int64
		for i := range logs {
			ids = append(ids, logs[i].ContainerTypeID)
			if logs[i].TypeID != nil {
				ids = append(ids, *logs[i].TypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range logs {
			logs[i].ContainerTypeName = names[logs[i].ContainerTypeID]
			if logs[i].TypeID != nil {
				logs[i].TypeName = names[*logs[i].TypeID]
			}
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationContainerLogsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Logs: logs}, nil
	}
}

// --- list_corporation_facilities (private, not paginated) ---

type Facility struct {
	FacilityID int64  `json:"facilityId"`
	SystemID   int64  `json:"systemId"`
	TypeID     int64  `json:"typeId"`
	TypeName   string `json:"typeName,omitempty"`
}

var listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	facilities, err := client.CorporationFacilities(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]Facility, len(facilities))
	for i, f := range facilities {
		out[i] = Facility{FacilityID: f.FacilityId, SystemID: f.SystemId, TypeID: f.TypeId}
	}
	return out, nil
}

type ListCorporationFacilitiesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationFacilitiesOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Facilities    []Facility `json:"facilities"`
}

func listCorporationFacilitiesHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationFacilitiesInput, ListCorporationFacilitiesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFacilitiesInput) (*mcpsdk.CallToolResult, ListCorporationFacilitiesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFacilitiesOutput{}, err
		}
		if !found {
			return nil, ListCorporationFacilitiesOutput{}, fmt.Errorf("list corporation facilities for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		facilities, err := listCorporationFacilities(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationFacilitiesOutput{}, fmt.Errorf("list corporation facilities: %w", err)
		}
		ids := make([]int64, len(facilities))
		for i := range facilities {
			ids[i] = facilities[i].TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range facilities {
			facilities[i].TypeName = names[facilities[i].TypeID]
		}
		return nil, ListCorporationFacilitiesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Facilities: facilities}, nil
	}
}

// --- list_corporation_medals (private, paginated) ---

type CorporationMedal struct {
	CreatedAt   time.Time `json:"createdAt"`
	CreatorID   int64     `json:"creatorId"`
	Description string    `json:"description"`
	MedalID     int64     `json:"medalId"`
	Title       string    `json:"title"`
}

var listCorporationMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationMedal, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	medals, totalPages, err := client.CorporationMedals(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CorporationMedal, len(medals))
	for i, m := range medals {
		out[i] = CorporationMedal{CreatedAt: m.CreatedAt, CreatorID: m.CreatorId, Description: m.Description, MedalID: m.MedalId, Title: m.Title}
	}
	return out, totalPages, nil
}

type ListCorporationMedalsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMedalsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	Page          int32              `json:"page"`
	TotalPages    int32              `json:"totalPages"`
	Medals        []CorporationMedal `json:"medals"`
}

func listCorporationMedalsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMedalsInput, ListCorporationMedalsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMedalsInput) (*mcpsdk.CallToolResult, ListCorporationMedalsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMedalsOutput{}, err
		}
		if !found {
			return nil, ListCorporationMedalsOutput{}, fmt.Errorf("list corporation medals for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		medals, totalPages, err := listCorporationMedals(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMedalsOutput{}, fmt.Errorf("list corporation medals: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationMedalsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Medals: medals}, nil
	}
}

// --- list_corporation_medals_issued (private, paginated) ---

type MedalIssued struct {
	CharacterID int64     `json:"characterId"`
	IssuedAt    time.Time `json:"issuedAt"`
	IssuerID    int64     `json:"issuerId"`
	MedalID     int64     `json:"medalId"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
}

var listCorporationMedalsIssued = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]MedalIssued, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	issued, totalPages, err := client.CorporationMedalsIssued(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]MedalIssued, len(issued))
	for i, m := range issued {
		out[i] = MedalIssued{CharacterID: m.CharacterId, IssuedAt: m.IssuedAt, IssuerID: m.IssuerId, MedalID: m.MedalId, Reason: m.Reason, Status: m.Status}
	}
	return out, totalPages, nil
}

type ListCorporationMedalsIssuedInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMedalsIssuedOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Issued        []MedalIssued `json:"issued"`
}

func listCorporationMedalsIssuedHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMedalsIssuedInput, ListCorporationMedalsIssuedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMedalsIssuedInput) (*mcpsdk.CallToolResult, ListCorporationMedalsIssuedOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMedalsIssuedOutput{}, err
		}
		if !found {
			return nil, ListCorporationMedalsIssuedOutput{}, fmt.Errorf("list corporation medals issued for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		issued, totalPages, err := listCorporationMedalsIssued(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMedalsIssuedOutput{}, fmt.Errorf("list corporation medals issued: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationMedalsIssuedOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Issued: issued}, nil
	}
}

// --- list_corporation_starbases (private, paginated) ---

type Starbase struct {
	MoonID          *int64     `json:"moonId,omitempty"`
	OnlinedSince    *time.Time `json:"onlinedSince,omitempty"`
	ReinforcedUntil *time.Time `json:"reinforcedUntil,omitempty"`
	StarbaseID      int64      `json:"starbaseId"`
	State           *string    `json:"state,omitempty"`
	SystemID        int64      `json:"systemId"`
	TypeID          int64      `json:"typeId"`
	TypeName        string     `json:"typeName,omitempty"`
	UnanchorAt      *time.Time `json:"unanchorAt,omitempty"`
}

var listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	starbases, totalPages, err := client.CorporationStarbases(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Starbase, len(starbases))
	for i, s := range starbases {
		out[i] = Starbase{
			MoonID: s.MoonId, OnlinedSince: s.OnlinedSince, ReinforcedUntil: s.ReinforcedUntil,
			StarbaseID: s.StarbaseId, State: s.State, SystemID: s.SystemId, TypeID: s.TypeId, UnanchorAt: s.UnanchorAt,
		}
	}
	return out, totalPages, nil
}

type ListCorporationStarbasesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationStarbasesOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Page          int32      `json:"page"`
	TotalPages    int32      `json:"totalPages"`
	Starbases     []Starbase `json:"starbases"`
}

func listCorporationStarbasesHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationStarbasesInput, ListCorporationStarbasesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationStarbasesInput) (*mcpsdk.CallToolResult, ListCorporationStarbasesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationStarbasesOutput{}, err
		}
		if !found {
			return nil, ListCorporationStarbasesOutput{}, fmt.Errorf("list corporation starbases for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		starbases, totalPages, err := listCorporationStarbases(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationStarbasesOutput{}, fmt.Errorf("list corporation starbases: %w", err)
		}
		ids := make([]int64, len(starbases))
		for i := range starbases {
			ids[i] = starbases[i].TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range starbases {
			starbases[i].TypeName = names[starbases[i].TypeID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationStarbasesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Starbases: starbases}, nil
	}
}

// --- get_corporation_starbase (private, not paginated; requires systemId) ---

type StarbaseFuel struct {
	Quantity int64  `json:"quantity"`
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
}

type StarbaseDetail struct {
	AllowAllianceMembers                bool           `json:"allowAllianceMembers"`
	AllowCorporationMembers             bool           `json:"allowCorporationMembers"`
	Anchor                              string         `json:"anchor"`
	AttackIfAtWar                       bool           `json:"attackIfAtWar"`
	AttackIfOtherSecurityStatusDropping bool           `json:"attackIfOtherSecurityStatusDropping"`
	AttackSecurityStatusThreshold       *float64       `json:"attackSecurityStatusThreshold,omitempty"`
	AttackStandingThreshold             *float64       `json:"attackStandingThreshold,omitempty"`
	FuelBayTake                         string         `json:"fuelBayTake"`
	FuelBayView                         string         `json:"fuelBayView"`
	Fuels                               []StarbaseFuel `json:"fuels,omitempty"`
	Offline                             string         `json:"offline"`
	Online                              string         `json:"online"`
	Unanchor                            string         `json:"unanchor"`
	UseAllianceStandings                bool           `json:"useAllianceStandings"`
}

var getCorporationStarbase = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, starbaseID int64, systemID int64) (StarbaseDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return StarbaseDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	detail, err := client.CorporationStarbaseDetail(ctx, corporationID, starbaseID, systemID)
	if err != nil {
		return StarbaseDetail{}, err
	}
	fuels := make([]StarbaseFuel, len(detail.Fuels))
	for i, f := range detail.Fuels {
		fuels[i] = StarbaseFuel{Quantity: f.Quantity, TypeID: f.TypeId}
	}
	return StarbaseDetail{
		AllowAllianceMembers: detail.AllowAllianceMembers, AllowCorporationMembers: detail.AllowCorporationMembers,
		Anchor: detail.Anchor, AttackIfAtWar: detail.AttackIfAtWar, AttackIfOtherSecurityStatusDropping: detail.AttackIfOtherSecurityStatusDropping,
		AttackSecurityStatusThreshold: detail.AttackSecurityStatusThreshold, AttackStandingThreshold: detail.AttackStandingThreshold,
		FuelBayTake: detail.FuelBayTake, FuelBayView: detail.FuelBayView, Fuels: fuels,
		Offline: detail.Offline, Online: detail.Online, Unanchor: detail.Unanchor, UseAllianceStandings: detail.UseAllianceStandings,
	}, nil
}

type GetCorporationStarbaseInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	StarbaseID    int64 `json:"starbaseId" jsonschema:"the starbase (POS) ID to look up"`
	SystemID      int64 `json:"systemId" jsonschema:"the solar system ID the starbase is in"`
}
type GetCorporationStarbaseOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	StarbaseID    int64          `json:"starbaseId"`
	Detail        StarbaseDetail `json:"detail"`
}

func getCorporationStarbaseHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCorporationStarbaseInput, GetCorporationStarbaseOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationStarbaseInput) (*mcpsdk.CallToolResult, GetCorporationStarbaseOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationStarbaseOutput{}, err
		}
		if !found {
			return nil, GetCorporationStarbaseOutput{}, fmt.Errorf("get corporation starbase for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCorporationStarbase(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.StarbaseID, in.SystemID)
		if err != nil {
			return nil, GetCorporationStarbaseOutput{}, fmt.Errorf("get corporation starbase: %w", err)
		}
		ids := make([]int64, len(detail.Fuels))
		for i := range detail.Fuels {
			ids[i] = detail.Fuels[i].TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range detail.Fuels {
			detail.Fuels[i].TypeName = names[detail.Fuels[i].TypeID]
		}
		return nil, GetCorporationStarbaseOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, StarbaseID: in.StarbaseID, Detail: detail}, nil
	}
}

// --- list_corporation_structures (private, paginated) ---

type StructureService struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type Structure struct {
	CorporationID      int64              `json:"corporationId"`
	FuelExpires        *time.Time         `json:"fuelExpires,omitempty"`
	Name               *string            `json:"name,omitempty"`
	NextReinforceApply *time.Time         `json:"nextReinforceApply,omitempty"`
	NextReinforceHour  *int64             `json:"nextReinforceHour,omitempty"`
	ProfileID          int64              `json:"profileId"`
	ReinforceHour      *int64             `json:"reinforceHour,omitempty"`
	Services           []StructureService `json:"services,omitempty"`
	State              string             `json:"state"`
	StateTimerEnd      *time.Time         `json:"stateTimerEnd,omitempty"`
	StateTimerStart    *time.Time         `json:"stateTimerStart,omitempty"`
	StructureID        int64              `json:"structureId"`
	SystemID           int64              `json:"systemId"`
	TypeID             int64              `json:"typeId"`
	TypeName           string             `json:"typeName,omitempty"`
	UnanchorsAt        *time.Time         `json:"unanchorsAt,omitempty"`
}

var listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	structures, totalPages, err := client.CorporationStructures(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Structure, len(structures))
	for i, s := range structures {
		services := make([]StructureService, len(s.Services))
		for j, svc := range s.Services {
			services[j] = StructureService{Name: svc.Name, State: svc.State}
		}
		out[i] = Structure{
			CorporationID: s.CorporationId, FuelExpires: s.FuelExpires, Name: s.Name,
			NextReinforceApply: s.NextReinforceApply, NextReinforceHour: s.NextReinforceHour, ProfileID: s.ProfileId,
			ReinforceHour: s.ReinforceHour, Services: services, State: s.State,
			StateTimerEnd: s.StateTimerEnd, StateTimerStart: s.StateTimerStart, StructureID: s.StructureId,
			SystemID: s.SystemId, TypeID: s.TypeId, UnanchorsAt: s.UnanchorsAt,
		}
	}
	return out, totalPages, nil
}

type ListCorporationStructuresInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationStructuresOutput struct {
	CharacterID   int32       `json:"characterId"`
	CorporationID int64       `json:"corporationId"`
	Page          int32       `json:"page"`
	TotalPages    int32       `json:"totalPages"`
	Structures    []Structure `json:"structures"`
}

func listCorporationStructuresHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationStructuresInput, ListCorporationStructuresOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationStructuresInput) (*mcpsdk.CallToolResult, ListCorporationStructuresOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationStructuresOutput{}, err
		}
		if !found {
			return nil, ListCorporationStructuresOutput{}, fmt.Errorf("list corporation structures for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		structures, totalPages, err := listCorporationStructures(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationStructuresOutput{}, fmt.Errorf("list corporation structures: %w", err)
		}
		ids := make([]int64, len(structures))
		for i := range structures {
			ids[i] = structures[i].TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range structures {
			structures[i].TypeName = names[structures[i].TypeID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationStructuresOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Structures: structures}, nil
	}
}
