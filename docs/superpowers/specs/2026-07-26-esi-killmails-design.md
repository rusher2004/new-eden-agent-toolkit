# ESI Killmails (Twentieth Batch) Design

## Status

Approved (autonomous execution). Twentieth batch: the "Killmails" tag (3 endpoints — the roadmap
doc's earlier estimate of 2 was off by one; the real generated client has 3: character-recent,
corporation-recent, and killmail-detail).

## Scope

**2 private, page-paginated** (existing `characterExists`): `GetCharactersCharacterIdKillmailsRecent`
("my own character," no separate target), `GetCorporationsCorporationIdKillmailsRecent`
(corp-scoped, `characterId` for auth + `corporationId` for target).

**1 public** (`esi.NewPublicClient`, no existence check, not paginated):
`GetKillmailsKillmailIdKillmailHash` — takes `killmailID int64` + `killmailHash string`. This is
public by design: the hash itself functions as the access token ESI requires — no character auth
needed once a caller has both the ID and hash (typically obtained from one of the two "recent"
listings above, or from a third party).

## Verified model shapes

`CharactersCharacterIdKillmailsRecentGetInner{KillmailHash string, KillmailId int64}` — both
required; the corp-scoped listing reuses this exact same generated type (confirmed from the
generated `Execute()` return type, which is `[]CharactersCharacterIdKillmailsRecentGetInner` for
both the character and corporation "recent" endpoints, not a separate corp-specific model).

`KillmailsKillmailIdKillmailHashGet{KillmailId int64, KillmailTime time.Time, SolarSystemId int64,
Attackers []...AttackersInner, Victim ...Victim, MoonId, WarId *int64}` (both optional).
`AttackersInner{DamageDone int64, FinalBlow bool, SecurityStatus float64, AllianceId,
CharacterId, CorporationId, FactionId, ShipTypeId, WeaponTypeId *int64}` (only `DamageDone`,
`FinalBlow`, `SecurityStatus` required — an NPC/rogue-drone attacker can have every ID field nil).
`Victim{DamageTaken, ShipTypeId int64 (both required), AllianceId, CharacterId, CorporationId,
FactionId *int64, Items []...ItemsInner (optional), Position *...Position (optional)}`.
`ItemsInner{Flag, ItemTypeId, Singleton int64 (required), QuantityDestroyed, QuantityDropped
*int64 (optional), Items []...ItemsInnerItemsInner (optional — one level of container-nesting
only, e.g. items inside a destroyed container; the innermost items type has no further nested
`Items` field, confirmed from source, matching ESI's real one-level-deep container model)}`.
`Position{X, Y, Z float64}`.

## `esi/killmails.go` (3 methods)

```go
func (c *Client) CharacterKillmailsRecent(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error)
func (c *Client) CorporationKillmailsRecent(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error)
func (c *Client) KillmailDetail(ctx context.Context, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error)
```

Both "recent" methods use `page` (`0` omits `.Page()`) and the existing `totalPagesFromHeader` —
reused, not redeclared. `CharacterKillmailsRecent` uses `int64(c.characterID)` internally
("genuinely my own character"), matching the established convention.

**Verified reversed-parameter-order quirk**: the generated function is
`GetKillmailsKillmailIdKillmailHash(ctx, killmailHash string, killmailId int64)` —
`killmailHash` first, `killmailId` second — reversed from the URL path's own ordering
(`/killmails/{killmail_id}/{killmail_hash}/`) and from every other similarly-shaped method in
this project. `KillmailDetail`'s own signature keeps `(killmailID, killmailHash)` to match the
URL/tool-input order, swapping internally when calling the generated function — the same
facade-absorbs-the-quirk pattern already used for `CorporationContractBids`
(`esi/contracts.go`).

## `mcp/killmails.go` (3 tools)

- `list_character_killmails_recent` — private, paginated, input `{characterId int32, page int32}`, output `{characterId, page, totalPages int32, killmails []KillmailRef}` where `KillmailRef{KillmailID int64, KillmailHash string}` (shared with the corp variant since both use the same generated model)
- `list_corporation_killmails_recent` — private/corp-scoped, paginated, input `{characterId int32, corporationId int64, page int32}`, output `{characterId, corporationId, page, totalPages int32, killmails []KillmailRef}`
- `get_killmail` — public, input `{killmailId int64, killmailHash string}`, output `{killmailId int64, killmailTime time.Time, solarSystemId int64, moonId, warId *int64, attackers []KillmailAttacker, victim KillmailVictim}` where `KillmailAttacker{DamageDone int64, FinalBlow bool, SecurityStatus float64, AllianceID, CharacterID, CorporationID, FactionID, ShipTypeID, WeaponTypeID *int64}`, `KillmailVictim{DamageTaken, ShipTypeID int64, AllianceID, CharacterID, CorporationID, FactionID *int64, Items []KillmailItem, Position *KillmailPosition}`, `KillmailItem{Flag, ItemTypeID, Singleton int64, QuantityDestroyed, QuantityDropped *int64, Items []KillmailItem}` (self-referential Go type for the one-level container nesting — matches the generated client's own one-level-deep shape; the inner `Items` field is simply left empty when mapping the innermost generated type, since it has no further nested items), `KillmailPosition{X, Y, Z float64}`

## Testing

Same three-layer isolation as every prior batch. `esi/killmails_test.go` covers all 3 methods
(parsing + at least one error path + pagination omit/set for the 2 "recent" methods; the detail
test must cover a full fixture including a non-empty `Attackers` array, a `Victim.Items` entry
that itself has a non-empty nested `Items` array, and both `MoonId`/`WarId` populated).
`mcp/killmails_internal_test.go` covers happy-path for all 3 tools + unknown-character for the 2
private tools, plus the detail test's nested-container-items mapping. `mcp/server_test.go` grows
from 129 to 132.

## Out of scope

- No automatic multi-page aggregation, consistent with every other paginated tool.
- No SDE cross-referencing of type/system/flag IDs to human-readable names.
