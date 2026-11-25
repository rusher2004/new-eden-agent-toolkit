package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fwCharacterESI is the ESI call get_character_fw_stats makes.
type fwCharacterESI interface {
	CharacterFwStats(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error)
}

// fwCorporationESI is the ESI call get_corporation_fw_stats makes.
type fwCorporationESI interface {
	CorporationFwStats(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error)
}

// fwLeaderboardsESI is the ESI call get_fw_leaderboards makes.
type fwLeaderboardsESI interface {
	FwLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsGet, error)
}

// fwCharacterLeaderboardsESI is the ESI call get_fw_character_leaderboards makes.
type fwCharacterLeaderboardsESI interface {
	FwCharacterLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error)
}

// fwCorporationLeaderboardsESI is the ESI call get_fw_corporation_leaderboards makes.
type fwCorporationLeaderboardsESI interface {
	FwCorporationLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error)
}

// fwStatsESI is the ESI call list_fw_stats makes.
type fwStatsESI interface {
	FwStats(ctx context.Context) ([]esiapi.FwStatsGetInner, error)
}

// fwSystemsESI is the ESI call list_fw_systems makes.
type fwSystemsESI interface {
	FwSystems(ctx context.Context) ([]esiapi.FwSystemsGetInner, error)
}

// fwSystemsSDE is the SDE lookups list_fw_systems makes: system and faction names.
type fwSystemsSDE interface {
	systemNameCache
	factionNameCache
}

// fwWarsESI is the ESI call list_fw_wars makes.
type fwWarsESI interface {
	FwWars(ctx context.Context) ([]esiapi.FwWarsGetInner, error)
}

// KillsOrPointsStats is the shared {last_week, total, yesterday} shape used by every FW
// endpoint's Kills/VictoryPoints sub-fields (character, corporation, and per-faction stats) --
// 6 distinct generated Go types collapse to this one identical shape, verified from source.
type KillsOrPointsStats struct {
	LastWeek  int64 `json:"lastWeek"`
	Total     int64 `json:"total"`
	Yesterday int64 `json:"yesterday"`
}

// --- get_character_fw_stats (private) ---

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

// AddGetCharacterFwStats registers get_character_fw_stats. open builds the ESI client; any client with the
// fwCharacterESI methods will do.
func AddGetCharacterFwStats[C fwCharacterESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore factionNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_fw_stats",
		Description: "Get a logged-in EVE Online character's faction warfare stats.",
	}, getCharacterFwStatsHandler(userAgent, open, store, sdeStore))
}

func getCharacterFwStatsHandler[C fwCharacterESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore factionNameCache) mcpsdk.ToolHandlerFor[GetCharacterFwStatsInput, GetCharacterFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFwStatsInput) (*mcpsdk.CallToolResult, GetCharacterFwStatsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFwStatsOutput{}, err
		}
		if !found {
			return nil, GetCharacterFwStatsOutput{}, fmt.Errorf("get character fw stats for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFwStatsOutput{}, fmt.Errorf("get character fw stats: build ESI client: %w", err)
		}
		stats, err := client.CharacterFwStats(ctx)
		if err != nil {
			return nil, GetCharacterFwStatsOutput{}, fmt.Errorf("get character fw stats: %w", err)
		}
		var factionName string
		if stats.FactionId != nil {
			names := resolveFactionNames(ctx, sdeStore, userAgent, []int64{*stats.FactionId})
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

// AddGetCorporationFwStats registers get_corporation_fw_stats. open builds the ESI client; any client with the
// fwCorporationESI methods will do.
func AddGetCorporationFwStats[C fwCorporationESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore factionNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_corporation_fw_stats",
		Description: "Get a corporation's faction warfare stats, using a logged-in character's roles to authorize.",
	}, getCorporationFwStatsHandler(userAgent, open, store, sdeStore))
}

func getCorporationFwStatsHandler[C fwCorporationESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore factionNameCache) mcpsdk.ToolHandlerFor[GetCorporationFwStatsInput, GetCorporationFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationFwStatsInput) (*mcpsdk.CallToolResult, GetCorporationFwStatsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationFwStatsOutput{}, err
		}
		if !found {
			return nil, GetCorporationFwStatsOutput{}, fmt.Errorf("get corporation fw stats for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCorporationFwStatsOutput{}, fmt.Errorf("get corporation fw stats: build ESI client: %w", err)
		}
		stats, err := client.CorporationFwStats(ctx, in.CorporationID)
		if err != nil {
			return nil, GetCorporationFwStatsOutput{}, fmt.Errorf("get corporation fw stats: %w", err)
		}
		var factionName string
		if stats.FactionId != nil {
			names := resolveFactionNames(ctx, sdeStore, userAgent, []int64{*stats.FactionId})
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

type GetFwLeaderboardsInput struct{}
type GetFwLeaderboardsOutput struct {
	Kills         FactionLeaderboardMetric `json:"kills"`
	VictoryPoints FactionLeaderboardMetric `json:"victoryPoints"`
}

// AddGetFwLeaderboards registers get_fw_leaderboards. open builds the ESI client; any client with the
// fwLeaderboardsESI methods will do.
func AddGetFwLeaderboards[C fwLeaderboardsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore factionNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_fw_leaderboards",
		Description: "Get the top faction rankings by kills and victory points (public data, no character required).",
	}, getFwLeaderboardsHandler(userAgent, open, sdeStore))
}

func getFwLeaderboardsHandler[C fwLeaderboardsESI](userAgent string, open func() C, sdeStore factionNameCache) mcpsdk.ToolHandlerFor[GetFwLeaderboardsInput, GetFwLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwLeaderboardsOutput, error) {
		lb, err := open().FwLeaderboards(ctx)
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
		factionNames := resolveFactionNames(ctx, sdeStore, userAgent, factionIDs)
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

type GetFwCharacterLeaderboardsInput struct{}
type GetFwCharacterLeaderboardsOutput struct {
	Kills         CharacterLeaderboardMetric `json:"kills"`
	VictoryPoints CharacterLeaderboardMetric `json:"victoryPoints"`
}

// AddGetFwCharacterLeaderboards registers get_fw_character_leaderboards. open builds the ESI client; any client with the
// fwCharacterLeaderboardsESI methods will do.
func AddGetFwCharacterLeaderboards[C fwCharacterLeaderboardsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_fw_character_leaderboards",
		Description: "Get the top pilot rankings by kills and victory points (public data, no character required).",
	}, getFwCharacterLeaderboardsHandler(open))
}

func getFwCharacterLeaderboardsHandler[C fwCharacterLeaderboardsESI](open func() C) mcpsdk.ToolHandlerFor[GetFwCharacterLeaderboardsInput, GetFwCharacterLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwCharacterLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwCharacterLeaderboardsOutput, error) {
		lb, err := open().FwCharacterLeaderboards(ctx)
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

type GetFwCorporationLeaderboardsInput struct{}
type GetFwCorporationLeaderboardsOutput struct {
	Kills         CorporationLeaderboardMetric `json:"kills"`
	VictoryPoints CorporationLeaderboardMetric `json:"victoryPoints"`
}

// AddGetFwCorporationLeaderboards registers get_fw_corporation_leaderboards. open builds the ESI client; any client with the
// fwCorporationLeaderboardsESI methods will do.
func AddGetFwCorporationLeaderboards[C fwCorporationLeaderboardsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_fw_corporation_leaderboards",
		Description: "Get the top corporation rankings by kills and victory points (public data, no character required).",
	}, getFwCorporationLeaderboardsHandler(open))
}

func getFwCorporationLeaderboardsHandler[C fwCorporationLeaderboardsESI](open func() C) mcpsdk.ToolHandlerFor[GetFwCorporationLeaderboardsInput, GetFwCorporationLeaderboardsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetFwCorporationLeaderboardsInput) (*mcpsdk.CallToolResult, GetFwCorporationLeaderboardsOutput, error) {
		lb, err := open().FwCorporationLeaderboards(ctx)
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

type ListFwStatsInput struct{}
type ListFwStatsOutput struct {
	Stats []FwFactionStats `json:"stats"`
}

// AddListFwStats registers list_fw_stats. open builds the ESI client; any client with the
// fwStatsESI methods will do.
func AddListFwStats[C fwStatsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore factionNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_fw_stats",
		Description: "List every faction's faction warfare stats (public data, no character required).",
	}, listFwStatsHandler(userAgent, open, sdeStore))
}

func listFwStatsHandler[C fwStatsESI](userAgent string, open func() C, sdeStore factionNameCache) mcpsdk.ToolHandlerFor[ListFwStatsInput, ListFwStatsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwStatsInput) (*mcpsdk.CallToolResult, ListFwStatsOutput, error) {
		stats, err := open().FwStats(ctx)
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
		factionNames := resolveFactionNames(ctx, sdeStore, userAgent, factionIDs)
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

type ListFwSystemsInput struct{}
type ListFwSystemsOutput struct {
	Systems []FwSystem `json:"systems"`
}

// AddListFwSystems registers list_fw_systems. open builds the ESI client; any client with the
// fwSystemsESI methods will do.
func AddListFwSystems[C fwSystemsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore fwSystemsSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_fw_systems",
		Description: "List every solar system's faction warfare occupation/contest state (public data, no character required).",
	}, listFwSystemsHandler(userAgent, open, sdeStore))
}

func listFwSystemsHandler[C fwSystemsESI](userAgent string, open func() C, sdeStore fwSystemsSDE) mcpsdk.ToolHandlerFor[ListFwSystemsInput, ListFwSystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwSystemsInput) (*mcpsdk.CallToolResult, ListFwSystemsOutput, error) {
		systems, err := open().FwSystems(ctx)
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
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
		for i := range out {
			out[i].SolarSystemName = systemNames[out[i].SolarSystemID]
		}
		var factionIDs []int64
		for _, s := range out {
			factionIDs = append(factionIDs, s.OccupierFactionID, s.OwnerFactionID)
		}
		factionNames := resolveFactionNames(ctx, sdeStore, userAgent, factionIDs)
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

type ListFwWarsInput struct{}
type ListFwWarsOutput struct {
	Wars []FwWar `json:"wars"`
}

// AddListFwWars registers list_fw_wars. open builds the ESI client; any client with the
// fwWarsESI methods will do.
func AddListFwWars[C fwWarsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore factionNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_fw_wars",
		Description: "List every faction warfare declaration (public data, no character required).",
	}, listFwWarsHandler(userAgent, open, sdeStore))
}

func listFwWarsHandler[C fwWarsESI](userAgent string, open func() C, sdeStore factionNameCache) mcpsdk.ToolHandlerFor[ListFwWarsInput, ListFwWarsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListFwWarsInput) (*mcpsdk.CallToolResult, ListFwWarsOutput, error) {
		wars, err := open().FwWars(ctx)
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
		factionNames := resolveFactionNames(ctx, sdeStore, userAgent, factionIDs)
		for i := range out {
			out[i].AgainstFactionName = factionNames[out[i].AgainstID]
			out[i].FactionName = factionNames[out[i].FactionID]
		}
		return nil, ListFwWarsOutput{Wars: out}, nil
	}
}
