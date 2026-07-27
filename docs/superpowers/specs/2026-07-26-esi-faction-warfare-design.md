# ESI Faction Warfare (Thirteenth Batch) Design

## Status

Approved (autonomous execution). Thirteenth batch: the "Faction Warfare" tag alone (8 endpoints).
Mixed public/private, no pagination anywhere in the tag (verified — no `.Page()` method on any of
the 8 generated request builders).

## Scope

**2 private** (character auth; existence-checked via existing `characterExists`):
`GetCharactersCharacterIdFwStats` ("my own" character's FW stats — `esi-characters.read_fw_stats.v1`,
already in `allScopes`), `GetCorporationsCorporationIdFwStats` (corp-scoped, needs `characterId` for
auth + `corporationId` for target — `esi-corporations.read_fw_stats.v1`, already in `allScopes`).

**6 public** (`esi.NewPublicClient`, no existence check): `GetFwLeaderboards` (top factions),
`GetFwLeaderboardsCharacters` (top pilots), `GetFwLeaderboardsCorporations` (top corporations),
`GetFwStats` (per-faction stats, bare list), `GetFwSystems` (system occupation/contest state),
`GetFwWars` (faction war declarations).

## New pattern: three-way "leaderboard entry" reuse

`GetFwLeaderboards`/`GetFwLeaderboardsCharacters`/`GetFwLeaderboardsCorporations` each return a
`{Kills, VictoryPoints}` pair, and each of those two fields is itself
`{ActiveTotal, LastWeek, Yesterday []Entry}` — 3 ranking windows. Verified from source: the
generated package produces **18 distinct Go types** for these 9 slices across the 3 endpoints
(kills/VP × 3 windows × 3 endpoints), but they collapse to exactly **3 reusable shapes** since the
only difference between them is which single identifying field each entry carries:

- `GetFwLeaderboards`'s 6 slice-entry types all share the shape `{Amount *int64, FactionId *int64}`.
- `GetFwLeaderboardsCharacters`'s 6 share `{Amount *int64, CharacterId *int64}`.
- `GetFwLeaderboardsCorporations`'s 6 share `{Amount *int64, CorporationId *int64}`.

This MCP layer defines 3 reusable entry types (`LeaderboardFactionEntry`, `LeaderboardCharacterEntry`,
`LeaderboardCorporationEntry`) and 3 reusable `{ActiveTotal, LastWeek, Yesterday}` wrapper types
(`FactionLeaderboardMetric`, `CharacterLeaderboardMetric`, `CorporationLeaderboardMetric`), each
defined once and reused for both the `Kills` and `VictoryPoints` fields of their respective
endpoint — mirroring the Corporation Projects batch's "flatten reused generated shapes into shared
MCP types" precedent.

## New pattern: shared "Kills"/"VictoryPoints" stats shape

`CharactersCharacterIdFwStatsGet`, `CorporationsCorporationIdFwStatsGet`, and each entry of
`GetFwStats`'s bare list (`FwStatsGetInner`) each carry a `Kills` and a `VictoryPoints` field —
6 more distinct generated Go types, all sharing the identical shape `{LastWeek, Total, Yesterday
int64}`. Flattened to one reusable MCP type `KillsOrPointsStats{LastWeek, Total, Yesterday int64}`,
reused for all 6 occurrences (Character's Kills/VictoryPoints, Corporation's Kills/VictoryPoints,
and the per-faction stats list's Kills/VictoryPoints).

## `esi/faction_warfare.go` (8 methods)

```go
func (c *Client) CharacterFwStats(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error)
func (c *Client) CorporationFwStats(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error)
func (c *Client) FwLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsGet, error)
func (c *Client) FwCharacterLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error)
func (c *Client) FwCorporationLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error)
func (c *Client) FwStats(ctx context.Context) ([]esiapi.FwStatsGetInner, error)
func (c *Client) FwSystems(ctx context.Context) ([]esiapi.FwSystemsGetInner, error)
func (c *Client) FwWars(ctx context.Context) ([]esiapi.FwWarsGetInner, error)
```

Note `CharacterFwStats` uses the client's own baked-in `characterID` (via `Client.characterID`,
same pattern as e.g. `esi/wallet.go`'s "my own character" endpoints) — it does NOT take a separate
`characterID` parameter the way Corporation Projects' Contribution endpoint did, since this ESI
endpoint is genuinely "my own stats," not "look up an arbitrary character's stats."

Verified model shapes: `FwSystemsGetInner{Contested string, OccupierFactionId, OwnerFactionId,
SolarSystemId, VictoryPoints, VictoryPointsThreshold int64}` (all required, no pointers);
`FwWarsGetInner{AgainstId, FactionId int64}` (both required).

## `mcp/faction_warfare.go` (8 tools)

Private (existence-checked via existing `characterExists`):
- `get_character_fw_stats` — input `{characterId int32}`, output `{characterId int32, currentRank, highestRank, factionId *int64, enlistedOn *time.Time, kills, victoryPoints KillsOrPointsStats}`
- `get_corporation_fw_stats` — input `{characterId int32, corporationId int64}`, output `{characterId, corporationId, factionId, pilots *int64, enlistedOn *time.Time, kills, victoryPoints KillsOrPointsStats}`

Public (no existence check):
- `get_fw_leaderboards` — input: none, output `{kills, victoryPoints FactionLeaderboardMetric}`
- `get_fw_character_leaderboards` — input: none, output `{kills, victoryPoints CharacterLeaderboardMetric}`
- `get_fw_corporation_leaderboards` — input: none, output `{kills, victoryPoints CorporationLeaderboardMetric}`
- `list_fw_stats` — input: none, output `{stats []FwFactionStats}` where `FwFactionStats{FactionID, Pilots, SystemsControlled int64, Kills, VictoryPoints KillsOrPointsStats}`
- `list_fw_systems` — input: none, output `{systems []FwSystem}` where `FwSystem{Contested string, OccupierFactionID, OwnerFactionID, SolarSystemID, VictoryPoints, VictoryPointsThreshold int64}`
- `list_fw_wars` — input: none, output `{wars []FwWar}` where `FwWar{AgainstID, FactionID int64}`

Shared types: `KillsOrPointsStats{LastWeek, Total, Yesterday int64}`, `LeaderboardFactionEntry{FactionID, Amount *int64}`,
`LeaderboardCharacterEntry{CharacterID, Amount *int64}`, `LeaderboardCorporationEntry{CorporationID, Amount *int64}`,
`FactionLeaderboardMetric{ActiveTotal, LastWeek, Yesterday []LeaderboardFactionEntry}`,
`CharacterLeaderboardMetric{ActiveTotal, LastWeek, Yesterday []LeaderboardCharacterEntry}`,
`CorporationLeaderboardMetric{ActiveTotal, LastWeek, Yesterday []LeaderboardCorporationEntry}`.

## Testing

Same three-layer isolation as every prior batch. `esi/faction_warfare_test.go` covers all 8
methods (parsing + at least one error path). `mcp/faction_warfare_internal_test.go` covers
happy-path for all 8 tools + unknown-character for the 2 private tools. `mcp/server_test.go` grows
from 88 to 96.

## Out of scope

- No SDE cross-referencing of `faction_id`/`system_id` to human-readable names.
