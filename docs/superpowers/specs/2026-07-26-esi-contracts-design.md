# ESI Contracts (Eighth Batch) Design

## Status

Approved (autonomous execution). Eighth batch: Contracts tag alone (9 endpoints — mixed
public/private, and this sweep's first encounter with ESI pagination).

## Scope and new patterns

**Mixed public/private** (3 public + 3 character-scoped + 3 corp-scoped), same split convention as
Character/Assets: public endpoints use `esi.NewPublicClient` with no existence check; private
endpoints use the authenticated client + `characterExists`.

**Pagination (new — first paginated endpoints in this sweep), verified live via `httptest.Server`**:
6 of the 9 endpoints paginate (`.Page(page int32)` builder method + an `X-Pages` response header):
`GetCharactersCharacterIdContracts`, `GetCorporationsCorporationIdContracts`,
`GetCorporationsCorporationIdContractsContractIdBids`, `GetContractsPublicRegionId`,
`GetContractsPublicBidsContractId`, `GetContractsPublicItemsContractId`. The other 3 (both
character/corp "contract items" endpoints and character "contract bids") do not paginate — verified
from source, not assumed, since ESI is inconsistent about which sub-resources paginate.

Convention: every paginated `esi/` method takes a `page int32` parameter — `0` means "don't call
`.Page()` at all," letting ESI apply its own default (page 1), verified live that omitting `.Page()`
sends no query param at all. Every paginated method returns `(items []T, totalPages int32, error)`
— `totalPages` parsed from the `X-Pages` response header (defaulting to `1` if missing or
unparseable, matching ESI's convention that no header means a single page). The MCP tool layer
surfaces `page` as an optional input and `totalPages` in the output, so an agent can page through
explicitly — no silent truncation, no automatic under-the-hood aggregation of all pages (which
would risk large numbers of ESI calls for a busy region without the caller asking for it).

**Verified parameter-order quirk**: `GetCorporationsCorporationIdContractsContractIdBids` and
`GetCorporationsCorporationIdContractsContractIdItems` take `(ctx, contractId int64, corporationId int64)`
— **contract ID first, corporation ID second** — the reverse of the URL path's own ordering
(`/corporations/{corporation_id}/contracts/{contract_id}/...`) and the reverse of every other
corp-scoped method in this sweep. Confirmed directly from the generated function signature, not
assumed from the URL shape. Both `esi/` methods and their brief-writer notes must get this right.

Scopes verified present in `allScopes`: `esi-contracts.read_character_contracts.v1`,
`esi-contracts.read_corporation_contracts.v1`.

## `esi/contracts.go`

```go
func (c *Client) CharacterContracts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error)
func (c *Client) CharacterContractBids(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error)
func (c *Client) CharacterContractItems(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error)
func (c *Client) CorporationContracts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error)
func (c *Client) CorporationContractBids(ctx context.Context, corporationID int64, contractID int64, page int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error)
func (c *Client) CorporationContractItems(ctx context.Context, corporationID int64, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error)
func (c *Client) PublicContracts(ctx context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error)
func (c *Client) PublicContractBids(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error)
func (c *Client) PublicContractItems(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error)
```

Note the corp-scoped facade methods keep `(corporationID, contractID)` parameter order in `neat`'s
own signature (matching every other corp-scoped facade method, e.g. Assets') even though the
underlying generated call must be made with `(contractID, corporationID)` — the facade absorbs
the quirk so callers of `esi.Client` see a consistent convention.

Response models: `CharactersCharacterIdContractsGetInner`/`CorporationsCorporationIdContractsGetInner`
share the same shape (`AcceptorId, AssigneeId, Availability, ContractId, DateExpired, DateIssued,
ForCorporation, IssuerCorporationId, IssuerId, Status, Type`, plus optional `Buyout, Collateral,
DateAccepted, DateCompleted, DaysToComplete, EndLocationId, Price, Reward, StartLocationId, Title,
Volume`). `ContractsPublicRegionIdGetInner` has the same fields **except no `Status`** (public
listings are implicitly outstanding) and `ForCorporation` is optional there, not required. Bids
share one model across all three scopes (`Amount, BidId, DateBid`, `BidderId` only on the
character-scoped one). Items differ slightly per scope (`RawQuantity` only on character; item-level
`ItemId`/`MaterialEfficiency`/`Runs`/`TimeEfficiency` only on the public variant).

## `mcp/contracts.go` (9 tools)

Public (no existence check):
- `list_public_contracts` — input `{regionId int64, page int32}`, output `{regionId, page, totalPages, contracts []Contract}`
- `list_public_contract_bids` — input `{contractId int64, page int32}`, output `{contractId, page, totalPages, bids []ContractBid}`
- `list_public_contract_items` — input `{contractId int64, page int32}`, output `{contractId, page, totalPages, items []ContractItem}`

Private (existence-checked via existing `characterExists`):
- `list_character_contracts` — input `{characterId int32, page int32}`, output `{characterId, page, totalPages, contracts []Contract}`
- `list_character_contract_bids` — input `{characterId int32, contractId int64}`, output `{characterId, contractId, bids []ContractBid}` (no page — unpaginated endpoint)
- `list_character_contract_items` — input `{characterId int32, contractId int64}`, output `{characterId, contractId, items []ContractItem}` (unpaginated)
- `list_corporation_contracts` — input `{characterId int32, corporationId int64, page int32}`, output `{characterId, corporationId, page, totalPages, contracts []Contract}`
- `list_corporation_contract_bids` — input `{characterId int32, corporationId int64, contractId int64, page int32}`, output `{characterId, corporationId, contractId, page, totalPages, bids []ContractBid}`
- `list_corporation_contract_items` — input `{characterId int32, corporationId int64, contractId int64}`, output `{characterId, corporationId, contractId, items []ContractItem}` (unpaginated)

Shared mapped types: `Contract{ContractID, AcceptorID, AssigneeID, IssuerID, IssuerCorporationID int64, Availability, Status, Type string, ForCorporation bool, DateIssued, DateExpired time.Time, DateAccepted, DateCompleted *time.Time, DaysToComplete, EndLocationID, StartLocationID *int64, Buyout, Collateral, Price, Reward, Volume *float64, Title *string}`
(`Status` omitted on public-region-sourced contracts — the mapping code leaves it as the Go zero
value `""` when populating from `ContractsPublicRegionIdGetInner`, which has no such field);
`ContractBid{BidID int64, BidderID *int64, Amount float64, DateBid time.Time}` (`BidderID`
populated for character-scoped and corporation-scoped bids — both use the same generated model
with a required `bidder_id` field — but genuinely absent for public bids, whose model has no such
field); `ContractItem{RecordID, TypeID int64, Quantity int64,
IsIncluded, IsSingleton bool, RawQuantity, ItemID, MaterialEfficiency, Runs, TimeEfficiency *int64,
IsBlueprintCopy *bool}`.

## Testing

`esi/contracts_test.go` verifies parsing for all 9 methods, the `page`/`totalPages` round-trip
(including confirming no `page` query param when `page=0`), and at least one HTTP-error path.
`mcp/contracts_internal_test.go` covers happy-path + unknown-character (private tools only) for
all 9. `mcp/server_test.go` grows from 48 to 57.

## Out of scope

- No automatic multi-page aggregation — `totalPages` is surfaced, paging through is the caller's
  explicit choice.
- No SDE-backed enrichment of `TypeId`/location IDs.
