# ESI Market — Design Spec (twenty-fourth batch)

**Batch:** Market tag, 11 endpoints — 2 private ("my own character"), 2 corp-scoped, 6 public,
1 that looks public but is actually private (structure market orders). Full verification detail,
including drafted and sandbox-compiled code for both layers with a live `mcpsdk.AddTool`+`Connect`
registration test.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

1. `GetCharactersCharacterIdOrders` — private, "my own character". **Not paginated** (no `.Page()`
   setter — EVE caps concurrent open orders, so ESI evidently doesn't expect this list to grow
   large). Response: 14 fields (`Duration`, `Escrow *float64`, `IsBuyOrder *bool`, `IsCorporation
   bool`, `Issued`, `LocationId`, `MinVolume *int64`, `OrderId`, `Price`, `Range`, `RegionId`,
   `TypeId`, `VolumeRemain`, `VolumeTotal`).
2. `GetCharactersCharacterIdOrdersHistory` — private, "my own character". **IS paginated**.
   Same 13 fields as #1 plus `State string` (required); `IsCorporation` is required on both
   character-scoped variants (verified from each model's own required-fields list, not assumed
   from the open-orders shape alone).
3. `GetCorporationsCorporationIdOrders` — corp-scoped (`characterId` authenticates + `corporationId`
   is the resource, ESI's own 403 is the sole gate, same pattern as `esi/industry.go`). **IS
   paginated**. Same 13 core fields minus `IsCorporation`, plus `IssuedBy int64` (required) and
   `WalletDivision int64` (required).
4. `GetCorporationsCorporationIdOrdersHistory` — corp-scoped. **IS paginated**. Adds `State string`
   (required); `IssuedBy` becomes `*int64` (**optional** — differs from #3's required `IssuedBy`,
   the one field whose optionality flips between a tag's own open/history pair). `WalletDivision`
   stays required.
5. `GetMarketsGroups` — public, not paginated. Bare `[]int64` of every market group ID.
6. `GetMarketsGroupsMarketGroupId` — public, not paginated. `{Description, MarketGroupId, Name,
   ParentGroupId *int64, Types []int64}`.
7. `GetMarketsPrices` — public, not paginated. `{AdjustedPrice *float64, AveragePrice *float64,
   TypeId}`.
8. `GetMarketsRegionIdHistory` — public, **not paginated**. `TypeId` is a **required** query param
   (verified: the generated `Execute` body itself errors if unset, and `openapi.yaml` marks it
   `required: true`). Response: `{Average, Date string, Highest, Lowest, OrderCount, Volume}`, all
   required.
9. `GetMarketsRegionIdOrders` — public. **IS paginated**. `OrderType(string)` is **required**
   (verified: `Execute` errors if unset, even though the spec lists a `default: all` the generated
   code doesn't apply) — callers must always pass `"buy"`/`"sell"`/`"all"`. `TypeId(int64)` is
   genuinely optional. Response adds `SystemId` (which solar system the order sits in) vs. the
   character/corp order models.
10. `GetMarketsRegionIdTypes` — public. **IS paginated**. Bare `[]int64` of type IDs with an active
    order in the region.
11. `GetMarketsStructuresStructureId` — **NOT public**, despite resembling #9. Verified in
    `esi/api/openapi.yaml`: requires OAuth2 scope `esi-markets.structure_markets.v1` (the querying
    character must have docking access to the structure) — modeled as "my own character" auth
    exactly like `CharacterFittings`, not `NewPublicClient`. **IS paginated**. Response identical
    to region orders minus `SystemId` (a structure pins its own system).

**Self-referential-type check**: none of the 9 response models nests a slice of its own type or of
each other. No Killmails-style recursion risk in this batch.

## `esi/market.go`

11 methods on `*Client`. Full code, already sandbox-verified clean (`go build`/`go vet`/`gofmt -l`/
`go test`), is in the implementation plan's Task 1.

`esi/market_test.go`: 24 tests (success + HTTP-error pair per method, plus dedicated
pagination-behavior tests — `page=0` omits the query param, a set `page` sends `page=N` and reads
back `X-Pages` — for every paginated endpoint), following `esi/fittings_test.go`'s and
`esi/industry_test.go`'s established styles.

## `mcp/market.go`

11 tools. Two genuine shared MCP types emerged from the verified field overlap (the "shared model
when it genuinely fits" precedent from `esi/industry.go`'s `IndustryJob`):

- **`MarketOrder`** — shared by all 4 character/corp order-listing tools. 13 fields identical
  across all four; each variant populates only its own applicable extra field(s)
  (`IsCorporation`/`State`/`IssuedBy`/`WalletDivision`), leaving the rest nil/zero.
- **`MarketOrderBookEntry`** — shared by `list_market_orders` (public region order book) and
  `list_structure_market_orders` (private structure order book) — identical shape except
  `SystemID`. Named for the anonymous order-book data shape, not for auth model, since one
  producer is public and the other private despite sharing this type.

Everything else (`GetMarketGroupOutput`, `MarketPrice`, `MarketHistoryEntry`) is a plain
per-endpoint type with nothing to share.

**The structure-market-orders privacy finding** is this batch's one genuine surprise:
`GetMarketsStructuresStructureId` reads and paginates exactly like `GetMarketsRegionIdOrders`, and
region orders are unambiguously public — it would have been easy to assume the structure endpoint
is too. `esi/api/openapi.yaml`'s security block settles it: `esi-markets.structure_markets.v1` is
required, so this tool takes `characterId` + the standard `characterExists` gate, modeled after
`CharacterFittings` rather than any public tool in this batch.

Full code for all 11 handlers, already sandbox-verified against a live `mcpsdk.AddTool`+`Connect`
registration test (zero schema-generator panics), is in the implementation plan's Task 2.

`mcp/market_internal_test.go`: 16 tests (a `MapsFields` + `UnknownCharacter` pair for the 5
private/corp tools, a `MapsFields` test for the 6 public tools), following
`mcp/fittings_internal_test.go`'s/`mcp/industry_internal_test.go`'s swappable-var-override pattern,
with full field-coverage assertions on every populated field AND every field expected to stay
nil/zero-value (every `Issued time.Time`/`*time.Time` field is asserted with `.Equal(...)`, per
this project's Corporation-Part-2 lesson and the Mail batch's own Minor finding about a missed
Timestamp assertion).

## `mcp/server.go` / `mcp/server_test.go`

Register 11 tools after the `delete_character_mail_label` block. Update package doc comment.
`server_test.go`'s expected-tools list: 146 → 157.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred forty-six" → "one hundred fifty-seven",
  append the 11 new tool names.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Market row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/market_test.go`,
swappable-`var`-override tests for `mcp/market_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.
