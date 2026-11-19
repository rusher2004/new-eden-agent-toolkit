# ESI Character (Fifth Batch) Design

## Status

Approved (autonomous execution authorized by explicit user instruction — see project memory on
autonomous ESI-sweep execution). Fifth batch of the ESI coverage sweep, covering the Character tag
alone (14 endpoints — large enough to be its own batch, no bundling).

## Scope

Character mixes public and private endpoints, verified against FC's live spec:

- **4 public** (arbitrary-character lookups, no auth): `GetCharactersDetail`, `PostCharactersAffiliation`,
  `GetCharactersCharacterIdCorporationhistory`, `GetCharactersCharacterIdPortrait`. These use
  `esi.NewPublicClient` and take a **target** `characterId`/`characterIds` as the resource being
  looked up — no `config.Store` existence check, matching the Alliance batch's precedent (an
  arbitrary-ID public lookup, not "my own" authenticated data).
- **10 private** (the logged-in character's own data): `GetCharactersCharacterIdAgentsResearch`,
  `Blueprints`, `Fatigue`, `Medals`, `Notifications`, `NotificationsContacts`, `Roles`, `Standings`,
  `Titles`, and `PostCharactersCharacterIdCspa` (a CSPA charge-cost calculation — POST because it
  takes a body of target character IDs, same read-only-POST pattern as Assets' resolve endpoints).
  These use `esi.NewClient` (per-character auth) + the standard `config.Store` existence check.

All 10 private scopes verified already present in `mcp/add_character.go`'s `allScopes`
(`esi-characters.read_agents_research.v1`, `read_blueprints.v1`, `read_contacts.v1` [for CSPA —
FC's own scope for this charge-cost lookup], `read_fatigue.v1`, `read_medals.v1`,
`read_notifications.v1` [covers both notifications and notifications/contacts], `read_corporation_roles.v1`,
`read_standings.v1`, `read_titles.v1`).

## `esi/character.go`

14 methods verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37`:

```go
func (c *Client) CharacterDetail(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error)
func (c *Client) CharacterAffiliations(ctx context.Context, characterIDs []int64) ([]esiapi.CharactersAffiliationPostInner, error)
func (c *Client) CharacterCorporationHistory(ctx context.Context, characterID int64) ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, error)
func (c *Client) CharacterPortrait(ctx context.Context, characterID int64) (*esiapi.CharactersCharacterIdPortraitGet, error)

func (c *Client) CharacterAgentsResearch(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error)
func (c *Client) CharacterBlueprints(ctx context.Context) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, error)
func (c *Client) CharacterFatigue(ctx context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error)
func (c *Client) CharacterMedals(ctx context.Context) ([]esiapi.CharactersCharacterIdMedalsGetInner, error)
func (c *Client) CharacterNotifications(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsGetInner, error)
func (c *Client) CharacterNotificationContacts(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, error)
func (c *Client) CharacterRoles(ctx context.Context) (*esiapi.CharactersCharacterIdRolesGet, error)
func (c *Client) CharacterStandings(ctx context.Context) ([]esiapi.CharactersCharacterIdStandingsGetInner, error)
func (c *Client) CharacterTitles(ctx context.Context) ([]esiapi.CharactersCharacterIdTitlesGetInner, error)
func (c *Client) CharacterCSPACost(ctx context.Context, targetCharacterIDs []int64) (float64, error)
```

All `describeError`-wrapped. First four use `int64(characterID)` param directly (arbitrary target);
the other ten use `int64(c.characterID)` (the client's own bound character), matching every prior
private facade. `CharacterCSPACost`'s source character is `c.characterID`; `targetCharacterIDs` is
the request body (bare `[]int64`, same shape already proven in Assets).

Verified model shapes worth noting (all raw-ID passthrough, no SDE resolution):
- `CharactersDetail` has several optional pointer fields (`AllianceId`, `CharacterTitleId`,
  `CorporationTitle`, `Description`, `FactionId`, `SecurityStatus` — all `*T`), consistent with
  every prior model with optional data.
- `CharactersCharacterIdMedalsGetInner.Graphics []CharactersCharacterIdMedalsGetInnerGraphicsInner`
  is a nested slice — passed through as-is in the `esi/` facade; the `mcp/` mapping flattens it into
  a JSON-friendly slice of a small struct.
- `CharactersCharacterIdRolesGet`'s four `[]string` fields are `omitempty` slices, not pointers —
  a nil/empty slice serializes as `[]` or is omitted, no special handling needed.
- `CharactersCharacterIdTitlesGetInner{Name, TitleId}` are both optional pointers.

## `mcp/character.go` (14 tools)

Public (no `characterId`-of-caller, no existence check — target ID is the input):
- `get_character` — input `{characterId int64}`, output `{characterId, name, corporationId, allianceId *int64, factionId *int64, birthday, bloodlineId, raceId, gender, securityStatus *float64, description *string, title *string, achievementScore}`
- `get_character_affiliations` — input `{characterIds []int64}`, output `{affiliations []CharacterAffiliation}` where `CharacterAffiliation{CharacterID, CorporationID int64, AllianceID, FactionID *int64}`
- `list_character_corporation_history` — input `{characterId int64}`, output `{characterId, history []CorporationHistoryEntry}` where `CorporationHistoryEntry{CorporationID, RecordID int64, StartDate time.Time, IsDeleted *bool}`
- `get_character_portrait` — input `{characterId int64}`, output `{characterId, px64x64, px128x128, px256x256, px512x512 *string}`

Private (character-existence-checked, matching wallet/access_list/etc.):
- `list_character_agents_research` — output `{characterId, agents []AgentResearch}` where `AgentResearch{AgentID, SkillTypeID int64, PointsPerDay, RemainderPoints float64, StartedAt time.Time}`
- `list_character_blueprints` — output `{characterId, blueprints []Blueprint}` where `Blueprint{ItemID, LocationID, TypeID, Quantity, Runs, MaterialEfficiency, TimeEfficiency int64, LocationFlag string}`
- `get_character_fatigue` — output `{characterId, jumpFatigueExpireDate, lastJumpDate, lastUpdateDate *time.Time}`
- `list_character_medals` — output `{characterId, medals []Medal}` where `Medal{MedalID, CorporationID, IssuerID int64, Date time.Time, Description, Reason, Status, Title string, Graphics []MedalGraphic}`, `MedalGraphic{Graphic string, Layer, Part int64, Color *int64}`
- `list_character_notifications` — output `{characterId, notifications []Notification}` where `Notification{NotificationID, SenderID int64, SenderType, Type string, Timestamp time.Time, IsRead *bool, Text *string}`
- `list_character_notification_contacts` — output `{characterId, notifications []ContactNotification}` where `ContactNotification{NotificationID, SenderCharacterID int64, SendDate time.Time, StandingLevel float64, Message string}`
- `get_character_roles` — output `{characterId, roles, rolesAtBase, rolesAtHq, rolesAtOther []string}`
- `list_character_standings` — output `{characterId, standings []Standing}` where `Standing{FromID int64, FromType string, Standing float64}`
- `list_character_titles` — output `{characterId, titles []Title}` where `Title{TitleID *int64, Name *string}`
- `get_character_cspa_cost` — input `{characterId int32, targetCharacterIds []int64}`, output `{characterId, cost float64}`

## Testing

Same three-layer isolation as every prior batch — `esi/character_test.go` (httptest.Server + both
public/authenticated client constructors), `mcp/character_internal_test.go` (func-var override,
existence-check tests only on the 10 private tools, no existence-check tests needed on the 4
public ones), `mcp/server_test.go` expected-tools list grows from 23 to 37.

## Out of scope

- No SDE-backed enrichment (bloodline/race/faction names, skill-type names) — raw IDs only.
- No corp-level scaffolding beyond what Assets already introduced.
