package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// KillsOrPointsStats is the shared {last_week, total, yesterday} shape used by every FW
// endpoint's Kills/VictoryPoints sub-fields (character, corporation, and per-faction stats) --
// 6 distinct generated Go types collapse to this one identical shape, verified from source.
type KillsOrPointsStats struct {
	LastWeek  int64 `json:"lastWeek"`
	Total     int64 `json:"total"`
	Yesterday int64 `json:"yesterday"`
}

// --- get_character_fw_stats (private) ---

var getCharacterFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterFwStats(ctx)
}

type GetCharacterFwStatsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterFwStatsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CurrentRank   *int64             `json:"currentRank,omitempty"`
	HighestRank   *int64             `json:"highestRank,omitempty"`
	FactionID     *int64             `json:"factionId,omitempty"`
	FactionName   string             `json:"factionName,omitempty"`
	EnlistedOn    *time.Time         `json:"enlistedOn,omitempty"`
	Kills         KillsOrPointsStats `json:"kills"`
	VictoryPoints KillsOrPointsStats `json:"victoryPoints"`
}

func getCharacterFwStatsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCharacterFwStatsInput, GetCharacterFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFwStatsInput) (*mcpsdk.CallToolResult, GetCharacterFwStatsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFwStatsOutput{}, err
		}
		if !found {
			return nil, GetCharacterFwStatsOutput{}, fmt.Errorf("get character fw stats for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		stats, err := getCharacterFwStats(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterFwStatsOutput{}, fmt.Errorf("get character fw stats: %w", err)
		}
		var factionName string
		if stats.FactionId != nil {
			names := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), []int64{*stats.FactionId})
			factionName = names[*stats.FactionId]
		}
		return nil, GetCharacterFwStatsOutput{
			CharacterID: in.CharacterID, CurrentRank: stats.CurrentRank, HighestRank: stats.HighestRank,
			FactionID: stats.FactionId, FactionName: factionName, EnlistedOn: stats.EnlistedOn,
			Kills:         KillsOrPointsStats{LastWeek: stats.Kills.LastWeek, Total: stats.Kills.Total, Yesterday: stats.Kills.Yesterday},
			VictoryPoints: KillsOrPointsStats{LastWeek: stats.VictoryPoints.LastWeek, Total: stats.VictoryPoints.Total, Yesterday: stats.VictoryPoints.Yesterday},
		}, nil
	}
}

// --- get_corporation_fw_stats (private) ---

var getCorporationFwStats = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationFwStats(ctx, corporationID)
}

type GetCorporationFwStatsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type GetCorporationFwStatsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	FactionID     *int64             `json:"factionId,omitempty"`
	FactionName   string             `json:"factionName,omitempty"`
	Pilots        *int64             `json:"pilots,omitempty"`
	EnlistedOn    *time.Time         `json:"enlistedOn,omitempty"`
	Kills         KillsOrPointsStats `json:"kills"`
	VictoryPoints KillsOrPointsStats `json:"victoryPoints"`
}

func getCorporationFwStatsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetCorporationFwStatsInput, GetCorporationFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationFwStatsInput) (*mcpsdk.CallToolResult, GetCorporationFwStatsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationFwStatsOutput{}, err
		}
		if !found {
			return nil, GetCorporationFwStatsOutput{}, fmt.Errorf("get corporation fw stats for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		stats, err := getCorporationFwStats(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, GetCorporationFwStatsOutput{}, fmt.Errorf("get corporation fw stats: %w", err)
		}
		var factionName string
		if stats.FactionId != nil {
			names := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), []int64{*stats.FactionId})
			factionName = names[*stats.FactionId]
		}
		return nil, GetCorporationFwStatsOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, FactionID: stats.FactionId, FactionName: factionName,
			Pilots: stats.Pilots, EnlistedOn: stats.EnlistedOn,
			Kills:         KillsOrPointsStats{LastWeek: stats.Kills.LastWeek, Total: stats.Kills.Total, Yesterday: stats.Kills.Yesterday},
			VictoryPoints: KillsOrPointsStats{LastWeek: stats.VictoryPoints.LastWeek, Total: stats.VictoryPoints.Total, Yesterday: stats.VictoryPoints.Yesterday},
		}, nil
	}
}

// --- get_fw_leaderboards (public, no input) ---

type LeaderboardFactionEntry struct {
	FactionID   *int64 `json:"factionId,omitempty"`
	FactionName string `json:"factionName,omitempty"`
	Amount      *int64 `json:"amount,omitempty"`
}
type FactionLeaderboardMetric struct {
	ActiveTotal []LeaderboardFactionEntry `json:"activeTotal"`
	LastWeek    []LeaderboardFactionEntry `json:"lastWeek"`
	Yesterday   []LeaderboardFactionEntry `json:"yesterday"`
}

var getFwLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwLeaderboards(ctx)
}

type GetFwLeaderboardsInput struct{}
type GetFwLeaderboardsOutput struct {
	Kills         FactionLeaderboardMetric `json:"kills"`
	VictoryPoints FactionLeaderboardMetric `json:"victoryPoints"`
}

func getFwLeaderboardsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetFwLeaderboardsInput, GetFwLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwLeaderboardsOutput, error) {
		lb, err := getFwLeaderboards(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetFwLeaderboardsOutput{}, fmt.Errorf("get fw leaderboards: %w", err)
		}
		killsActiveTotal := make([]LeaderboardFactionEntry, len(lb.Kills.ActiveTotal))
		for i, v := range lb.Kills.ActiveTotal {
			killsActiveTotal[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		killsLastWeek := make([]LeaderboardFactionEntry, len(lb.Kills.LastWeek))
		for i, v := range lb.Kills.LastWeek {
			killsLastWeek[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		killsYesterday := make([]LeaderboardFactionEntry, len(lb.Kills.Yesterday))
		for i, v := range lb.Kills.Yesterday {
			killsYesterday[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		vpActiveTotal := make([]LeaderboardFactionEntry, len(lb.VictoryPoints.ActiveTotal))
		for i, v := range lb.VictoryPoints.ActiveTotal {
			vpActiveTotal[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		vpLastWeek := make([]LeaderboardFactionEntry, len(lb.VictoryPoints.LastWeek))
		for i, v := range lb.VictoryPoints.LastWeek {
			vpLastWeek[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		vpYesterday := make([]LeaderboardFactionEntry, len(lb.VictoryPoints.Yesterday))
		for i, v := range lb.VictoryPoints.Yesterday {
			vpYesterday[i] = LeaderboardFactionEntry{FactionID: v.FactionId, Amount: v.Amount}
		}
		var factionIDs []int64
		for _, group := range [][]LeaderboardFactionEntry{killsActiveTotal, killsLastWeek, killsYesterday, vpActiveTotal, vpLastWeek, vpYesterday} {
			for _, e := range group {
				if e.FactionID != nil {
					factionIDs = append(factionIDs, *e.FactionID)
				}
			}
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for _, group := range [][]LeaderboardFactionEntry{killsActiveTotal, killsLastWeek, killsYesterday, vpActiveTotal, vpLastWeek, vpYesterday} {
			for i := range group {
				if group[i].FactionID != nil {
					group[i].FactionName = factionNames[*group[i].FactionID]
				}
			}
		}
		return nil, GetFwLeaderboardsOutput{
			Kills:         FactionLeaderboardMetric{ActiveTotal: killsActiveTotal, LastWeek: killsLastWeek, Yesterday: killsYesterday},
			VictoryPoints: FactionLeaderboardMetric{ActiveTotal: vpActiveTotal, LastWeek: vpLastWeek, Yesterday: vpYesterday},
		}, nil
	}
}

// --- get_fw_character_leaderboards (public, no input) ---

type LeaderboardCharacterEntry struct {
	CharacterID *int64 `json:"characterId,omitempty"`
	Amount      *int64 `json:"amount,omitempty"`
}
type CharacterLeaderboardMetric struct {
	ActiveTotal []LeaderboardCharacterEntry `json:"activeTotal"`
	LastWeek    []LeaderboardCharacterEntry `json:"lastWeek"`
	Yesterday   []LeaderboardCharacterEntry `json:"yesterday"`
}

var getFwCharacterLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCharactersGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwCharacterLeaderboards(ctx)
}

type GetFwCharacterLeaderboardsInput struct{}
type GetFwCharacterLeaderboardsOutput struct {
	Kills         CharacterLeaderboardMetric `json:"kills"`
	VictoryPoints CharacterLeaderboardMetric `json:"victoryPoints"`
}

func getFwCharacterLeaderboardsHandler(version string) mcpsdk.ToolHandlerFor[GetFwCharacterLeaderboardsInput, GetFwCharacterLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwCharacterLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwCharacterLeaderboardsOutput, error) {
		lb, err := getFwCharacterLeaderboards(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetFwCharacterLeaderboardsOutput{}, fmt.Errorf("get fw character leaderboards: %w", err)
		}
		killsActiveTotal := make([]LeaderboardCharacterEntry, len(lb.Kills.ActiveTotal))
		for i, v := range lb.Kills.ActiveTotal {
			killsActiveTotal[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		killsLastWeek := make([]LeaderboardCharacterEntry, len(lb.Kills.LastWeek))
		for i, v := range lb.Kills.LastWeek {
			killsLastWeek[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		killsYesterday := make([]LeaderboardCharacterEntry, len(lb.Kills.Yesterday))
		for i, v := range lb.Kills.Yesterday {
			killsYesterday[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		vpActiveTotal := make([]LeaderboardCharacterEntry, len(lb.VictoryPoints.ActiveTotal))
		for i, v := range lb.VictoryPoints.ActiveTotal {
			vpActiveTotal[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		vpLastWeek := make([]LeaderboardCharacterEntry, len(lb.VictoryPoints.LastWeek))
		for i, v := range lb.VictoryPoints.LastWeek {
			vpLastWeek[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		vpYesterday := make([]LeaderboardCharacterEntry, len(lb.VictoryPoints.Yesterday))
		for i, v := range lb.VictoryPoints.Yesterday {
			vpYesterday[i] = LeaderboardCharacterEntry{CharacterID: v.CharacterId, Amount: v.Amount}
		}
		return nil, GetFwCharacterLeaderboardsOutput{
			Kills:         CharacterLeaderboardMetric{ActiveTotal: killsActiveTotal, LastWeek: killsLastWeek, Yesterday: killsYesterday},
			VictoryPoints: CharacterLeaderboardMetric{ActiveTotal: vpActiveTotal, LastWeek: vpLastWeek, Yesterday: vpYesterday},
		}, nil
	}
}

// --- get_fw_corporation_leaderboards (public, no input) ---

type LeaderboardCorporationEntry struct {
	CorporationID *int64 `json:"corporationId,omitempty"`
	Amount        *int64 `json:"amount,omitempty"`
}
type CorporationLeaderboardMetric struct {
	ActiveTotal []LeaderboardCorporationEntry `json:"activeTotal"`
	LastWeek    []LeaderboardCorporationEntry `json:"lastWeek"`
	Yesterday   []LeaderboardCorporationEntry `json:"yesterday"`
}

var getFwCorporationLeaderboards = func(ctx context.Context, userAgent string) (*esiapi.FwLeaderboardsCorporationsGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwCorporationLeaderboards(ctx)
}

type GetFwCorporationLeaderboardsInput struct{}
type GetFwCorporationLeaderboardsOutput struct {
	Kills         CorporationLeaderboardMetric `json:"kills"`
	VictoryPoints CorporationLeaderboardMetric `json:"victoryPoints"`
}

func getFwCorporationLeaderboardsHandler(version string) mcpsdk.ToolHandlerFor[GetFwCorporationLeaderboardsInput, GetFwCorporationLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwCorporationLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwCorporationLeaderboardsOutput, error) {
		lb, err := getFwCorporationLeaderboards(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetFwCorporationLeaderboardsOutput{}, fmt.Errorf("get fw corporation leaderboards: %w", err)
		}
		killsActiveTotal := make([]LeaderboardCorporationEntry, len(lb.Kills.ActiveTotal))
		for i, v := range lb.Kills.ActiveTotal {
			killsActiveTotal[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		killsLastWeek := make([]LeaderboardCorporationEntry, len(lb.Kills.LastWeek))
		for i, v := range lb.Kills.LastWeek {
			killsLastWeek[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		killsYesterday := make([]LeaderboardCorporationEntry, len(lb.Kills.Yesterday))
		for i, v := range lb.Kills.Yesterday {
			killsYesterday[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		vpActiveTotal := make([]LeaderboardCorporationEntry, len(lb.VictoryPoints.ActiveTotal))
		for i, v := range lb.VictoryPoints.ActiveTotal {
			vpActiveTotal[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		vpLastWeek := make([]LeaderboardCorporationEntry, len(lb.VictoryPoints.LastWeek))
		for i, v := range lb.VictoryPoints.LastWeek {
			vpLastWeek[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		vpYesterday := make([]LeaderboardCorporationEntry, len(lb.VictoryPoints.Yesterday))
		for i, v := range lb.VictoryPoints.Yesterday {
			vpYesterday[i] = LeaderboardCorporationEntry{CorporationID: v.CorporationId, Amount: v.Amount}
		}
		return nil, GetFwCorporationLeaderboardsOutput{
			Kills:         CorporationLeaderboardMetric{ActiveTotal: killsActiveTotal, LastWeek: killsLastWeek, Yesterday: killsYesterday},
			VictoryPoints: CorporationLeaderboardMetric{ActiveTotal: vpActiveTotal, LastWeek: vpLastWeek, Yesterday: vpYesterday},
		}, nil
	}
}

// --- list_fw_stats (public, no input) ---

type FwFactionStats struct {
	FactionID         int64              `json:"factionId"`
	FactionName       string             `json:"factionName,omitempty"`
	Pilots            int64              `json:"pilots"`
	SystemsControlled int64              `json:"systemsControlled"`
	Kills             KillsOrPointsStats `json:"kills"`
	VictoryPoints     KillsOrPointsStats `json:"victoryPoints"`
}

var listFwStats = func(ctx context.Context, userAgent string) ([]esiapi.FwStatsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwStats(ctx)
}

type ListFwStatsInput struct{}
type ListFwStatsOutput struct {
	Stats []FwFactionStats `json:"stats"`
}

func listFwStatsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListFwStatsInput, ListFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwStatsInput) (*mcpsdk.CallToolResult, ListFwStatsOutput, error) {
		stats, err := listFwStats(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListFwStatsOutput{}, fmt.Errorf("list fw stats: %w", err)
		}
		out := make([]FwFactionStats, len(stats))
		for i, s := range stats {
			out[i] = FwFactionStats{
				FactionID: s.FactionId, Pilots: s.Pilots, SystemsControlled: s.SystemsControlled,
				Kills:         KillsOrPointsStats{LastWeek: s.Kills.LastWeek, Total: s.Kills.Total, Yesterday: s.Kills.Yesterday},
				VictoryPoints: KillsOrPointsStats{LastWeek: s.VictoryPoints.LastWeek, Total: s.VictoryPoints.Total, Yesterday: s.VictoryPoints.Yesterday},
			}
		}
		factionIDs := make([]int64, len(out))
		for i, s := range out {
			factionIDs[i] = s.FactionID
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for i := range out {
			out[i].FactionName = factionNames[out[i].FactionID]
		}
		return nil, ListFwStatsOutput{Stats: out}, nil
	}
}

// --- list_fw_systems (public, no input) ---

type FwSystem struct {
	Contested              string `json:"contested"`
	OccupierFactionID      int64  `json:"occupierFactionId"`
	OccupierFactionName    string `json:"occupierFactionName,omitempty"`
	OwnerFactionID         int64  `json:"ownerFactionId"`
	OwnerFactionName       string `json:"ownerFactionName,omitempty"`
	SolarSystemID          int64  `json:"solarSystemId"`
	SolarSystemName        string `json:"solarSystemName,omitempty"`
	VictoryPoints          int64  `json:"victoryPoints"`
	VictoryPointsThreshold int64  `json:"victoryPointsThreshold"`
}

var listFwSystems = func(ctx context.Context, userAgent string) ([]esiapi.FwSystemsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwSystems(ctx)
}

type ListFwSystemsInput struct{}
type ListFwSystemsOutput struct {
	Systems []FwSystem `json:"systems"`
}

func listFwSystemsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListFwSystemsInput, ListFwSystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwSystemsInput) (*mcpsdk.CallToolResult, ListFwSystemsOutput, error) {
		systems, err := listFwSystems(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListFwSystemsOutput{}, fmt.Errorf("list fw systems: %w", err)
		}
		out := make([]FwSystem, len(systems))
		for i, s := range systems {
			out[i] = FwSystem{
				Contested: s.Contested, OccupierFactionID: s.OccupierFactionId, OwnerFactionID: s.OwnerFactionId,
				SolarSystemID: s.SolarSystemId, VictoryPoints: s.VictoryPoints, VictoryPointsThreshold: s.VictoryPointsThreshold,
			}
		}
		systemIDs := make([]int64, len(out))
		for i, s := range out {
			systemIDs[i] = s.SolarSystemID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		for i := range out {
			out[i].SolarSystemName = systemNames[out[i].SolarSystemID]
		}
		var factionIDs []int64
		for _, s := range out {
			factionIDs = append(factionIDs, s.OccupierFactionID, s.OwnerFactionID)
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for i := range out {
			out[i].OccupierFactionName = factionNames[out[i].OccupierFactionID]
			out[i].OwnerFactionName = factionNames[out[i].OwnerFactionID]
		}
		return nil, ListFwSystemsOutput{Systems: out}, nil
	}
}

// --- list_fw_wars (public, no input) ---

type FwWar struct {
	AgainstID          int64  `json:"againstId"`
	AgainstFactionName string `json:"againstFactionName,omitempty"`
	FactionID          int64  `json:"factionId"`
	FactionName        string `json:"factionName,omitempty"`
}

var listFwWars = func(ctx context.Context, userAgent string) ([]esiapi.FwWarsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.FwWars(ctx)
}

type ListFwWarsInput struct{}
type ListFwWarsOutput struct {
	Wars []FwWar `json:"wars"`
}

func listFwWarsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListFwWarsInput, ListFwWarsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwWarsInput) (*mcpsdk.CallToolResult, ListFwWarsOutput, error) {
		wars, err := listFwWars(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListFwWarsOutput{}, fmt.Errorf("list fw wars: %w", err)
		}
		out := make([]FwWar, len(wars))
		for i, w := range wars {
			out[i] = FwWar{AgainstID: w.AgainstId, FactionID: w.FactionId}
		}
		var factionIDs []int64
		for _, w := range out {
			factionIDs = append(factionIDs, w.AgainstID, w.FactionID)
		}
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		for i := range out {
			out[i].AgainstFactionName = factionNames[out[i].AgainstID]
			out[i].FactionName = factionNames[out[i].FactionID]
		}
		return nil, ListFwWarsOutput{Wars: out}, nil
	}
}
