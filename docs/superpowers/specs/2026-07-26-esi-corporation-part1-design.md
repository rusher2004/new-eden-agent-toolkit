# ESI Corporation, Part 1 (Ninth Batch) Design

## Status

Approved (autonomous execution). Corporation is 22 endpoints — the largest single tag in the
sweep — split into two batches for manageable file size, matching the precedent set by
Assets+Calendar and Access List+Activities being sized/split for the same reason. **Part 1**
covers basics + membership/roles (14 endpoints, all read-only); **Part 2** (a later batch) covers
assets/infrastructure (blueprints, container logs, facilities, medals, starbases, structures — 8
endpoints). The roadmap doc's single Corporation row is only flipped to "This plan" once both
parts land.

## Scope

**4 public** (no auth, no existence check — `esi.NewPublicClient`): `GetCorporationsCorporationId`
(basic corp info), `GetCorporationsCorporationIdAlliancehistory`, `GetCorporationsCorporationIdIcons`,
`GetCorporationsNpccorps` (note: this one takes no `corporationId` at all — "list every NPC
corporation ID in the game").

**10 private** (character auth + corp roles, existence-checked via existing `characterExists`):
`GetCorporationsCorporationIdMembers`, `...MembersLimit` (returns a bare `int64`), `...MembersTitles`,
`...Membertracking`, `...Roles`, `...RolesHistory` (paginated), `...Titles`, `...Divisions`,
`...Shareholders` (paginated), `...Standings` (paginated, reuses `CharactersCharacterIdStandingsGetInner`
— same model as Character's own standings, confirmed from source).

All scopes verified present in `allScopes`: `esi-corporations.read_corporation_membership.v1`,
`read_titles.v1`, `read_divisions.v1`, `read_standings.v1`, `track_members.v1`,
`esi-wallet.read_corporation_wallets.v1` (for shareholders).

## `esi/corporation.go` (Part 1 — 14 methods)

```go
func (c *Client) CorporationDetail(ctx context.Context, corporationID int64) (*esiapi.CorporationsDetail, error)
func (c *Client) CorporationAllianceHistory(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdAlliancehistoryGetInner, error)
func (c *Client) CorporationIcons(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdIconsGet, error)
func (c *Client) NPCCorporations(ctx context.Context) ([]int64, error)
func (c *Client) CorporationMembers(ctx context.Context, corporationID int64) ([]int64, error)
func (c *Client) CorporationMembersLimit(ctx context.Context, corporationID int64) (int64, error)
func (c *Client) CorporationMemberTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembersTitlesGetInner, error)
func (c *Client) CorporationMemberTracking(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error)
func (c *Client) CorporationRoles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdRolesGetInner, error)
func (c *Client) CorporationRolesHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdRolesHistoryGetInner, int32, error)
func (c *Client) CorporationTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdTitlesGetInner, error)
func (c *Client) CorporationDivisions(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdDivisionsGet, error)
func (c *Client) CorporationShareholders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdShareholdersGetInner, int32, error)
func (c *Client) CorporationStandings(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdStandingsGetInner, int32, error)
```

All 14 verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37`.
`CorporationRolesHistory`/`CorporationShareholders`/`CorporationStandings` follow the established
pagination convention (`page=0` omits `.Page()`, returns `totalPages` parsed from `X-Pages`).
`CorporationMembersLimit` is a bare-`int64` return (like `NPCCorporations`/`CorporationMembers`
are bare `[]int64`) — no wrapper struct.

Verified model shapes: `CorporationsDetail` has nested `Palette{MainColor string, SecondaryColor,
TertiaryColor *string}` and `TaxRates{Isk, LoyaltyPoint float64}` (both non-optional structs on the
parent, `Palette` itself is `*CorporationsDetailPalette`, optional); `CorporationsCorporationIdDivisionsGet`
has `Hangar []CorporationsCorporationIdDivisionsGetHangarInner` and `Wallet []...WalletInner`, each
entry `{Division *int64, Name *string}`; `CorporationsCorporationIdRolesGetInner` has 8 `[]string`
role-list fields (`Roles`, `RolesAtBase/Hq/Other`, `GrantableRoles`, `GrantableRolesAtBase/Hq/Other`).

## `mcp/corporation.go` (Part 1 — 14 tools)

Public (no existence check):
- `get_corporation` — input `{corporationId int64}`, output `{corporationId, name, ticker, description, ceoId *int64, allianceId *int64, creatorId *int64, dateFounded *time.Time, factionId *int64, homeStationId, memberCount int64, shares int64, taxRateIsk, taxRateLoyaltyPoint float64, mainColor string, secondaryColor, tertiaryColor *string, url *string, friendlyFire, state, type string}`
- `list_corporation_alliance_history` — input `{corporationId int64}`, output `{corporationId, history []AllianceHistoryEntry}` where `AllianceHistoryEntry{AllianceID *int64, RecordID int64, StartDate time.Time, IsDeleted *bool}`
- `get_corporation_icons` — input `{corporationId int64}`, output `{corporationId, px64x64, px128x128, px256x256 *string}`
- `list_npc_corporations` — input: none, output `{corporationIds []int64}`

Private (existence-checked via existing `characterExists`):
- `list_corporation_members` — input `{characterId int32, corporationId int64}`, output `{characterId, corporationId, memberIds []int64}`
- `get_corporation_members_limit` — output `{characterId, corporationId, limit int64}`
- `list_corporation_member_titles` — output `{characterId, corporationId, memberTitles []MemberTitles}` where `MemberTitles{CharacterID int64, TitleIDs []int64}`
- `list_corporation_member_tracking` — output `{characterId, corporationId, members []MemberTracking}` where `MemberTracking{CharacterID int64, BaseID, LocationID, ShipTypeID *int64, LogonDate, LogoffDate, StartDate *time.Time}`
- `list_corporation_roles` — output `{characterId, corporationId, roles []MemberRoles}` where `MemberRoles{CharacterID int64, Roles, RolesAtBase, RolesAtHq, RolesAtOther, GrantableRoles, GrantableRolesAtBase, GrantableRolesAtHq, GrantableRolesAtOther []string}`
- `list_corporation_roles_history` — input adds `page int32`, output adds `page, totalPages int32`, `history []RoleChange` where `RoleChange{CharacterID, IssuerID int64, ChangedAt time.Time, RoleType string, OldRoles, NewRoles []string}`
- `list_corporation_titles` — output `{characterId, corporationId, titles []CorporationTitle}` where `CorporationTitle{TitleID *int64, Name *string}` (same shared model shape as `CorporationsCorporationIdTitlesGetInner`, verified similarly-shaped to Character's own titles model)
- `get_corporation_divisions` — output `{characterId, corporationId, hangars []Division, wallets []Division}` where `Division{DivisionNumber *int64, Name *string}`
- `list_corporation_shareholders` — input adds `page int32`, output adds `page, totalPages int32`, `shareholders []Shareholder` where `Shareholder{ShareholderID, ShareCount int64, ShareholderType string}`
- `list_corporation_standings` — input adds `page int32`, output adds `page, totalPages int32`, `standings []Standing` (reusing the same `Standing{FromID int64, FromType string, Standing float64}` type already defined in `mcp/character.go` — do not redeclare, reuse it directly since it's package-visible and the shape is identical)

## Testing

Same three-layer isolation as every prior batch. `esi/corporation_test.go` covers all 14 methods
(parsing + at least one error path + pagination for the 3 paginated ones). `mcp/corporation_internal_test.go`
covers happy-path + unknown-character (private tools only) for all 14. `mcp/server_test.go` grows
from 57 to 71.

## Out of scope (deferred to Part 2)

Blueprints, container logs, facilities, medals, medals/issued, starbases (list + detail), structures
— 8 endpoints, their own future batch.
