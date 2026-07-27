# ESI Loyalty — Design Spec (twenty-second batch)

**Batch:** Loyalty tag, 2 endpoints — one private ("my own character"), one public. No pagination.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

`esi/api_loyalty.go`'s `LoyaltyAPIService` exposes exactly 2 methods:

- `GetCharactersCharacterIdLoyaltyPoints(ctx, characterId int64)` → `[]CharactersCharacterIdLoyaltyPointsGetInner{CorporationId int64, LoyaltyPoints int64}`.
  Security: `OAuth2: [esi-characters.read_loyalty.v1]` — private, "my own character" style like every
  other character-scoped endpoint in this codebase (no caller-supplied character ID parameter).
- `GetLoyaltyStoresCorporationIdOffers(ctx, corporationId int64)` → `[]LoyaltyStoresCorporationIdOffersGetInner{AkCost *int64, IskCost int64, LpCost int64, OfferId int64, Quantity int64, RequiredItems []LoyaltyStoresCorporationIdOffersGetInnerRequiredItemsInner{Quantity int64, TypeId int64}, TypeId int64}`.
  Security: none in the OpenAPI spec — public, no character/auth needed at all (same shape as
  `esi/insurance.go`'s `InsurancePrices`), and takes a caller-supplied `corporationId` (there's no
  "my own corporation" concept here since this endpoint needs no authenticated character at all).

No pagination, no write endpoints, no oneOf/polymorphic types, no argument-order quirks. This is
the smallest batch since Location — standard 2-task split (esi layer, then mcp layer) given the
mix of private+public, mirroring Killmails' precedent for that same mix (though far simpler: no
nested container types, no reversed argument order).

## `esi/loyalty.go`

```go
// Package esi's loyalty.go covers both of "Loyalty"'s endpoints -- a character's own loyalty
// point balances (private, "my own character") and a corporation's loyalty store offers (public,
// no character/auth needed, caller-supplied corporation ID).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterLoyaltyPoints returns the client's own character's loyalty point balance with every
// corporation they've worked for.
func (c *Client) CharacterLoyaltyPoints(ctx context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error) {
	result, resp, err := c.api.LoyaltyAPI.GetCharactersCharacterIdLoyaltyPoints(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character loyalty points: %w", describeError(err, resp))
	}
	return result, nil
}

// LoyaltyStoreOffers returns a corporation's loyalty store offers. Public -- no authenticated
// character needed, so this is safe to call from a public Client (see esi.NewPublicClient).
func (c *Client) LoyaltyStoreOffers(ctx context.Context, corporationID int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error) {
	result, resp, err := c.api.LoyaltyAPI.GetLoyaltyStoresCorporationIdOffers(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get loyalty store offers: %w", describeError(err, resp))
	}
	return result, nil
}
```

`esi/loyalty_test.go`: standard `httptest.Server` round-trip tests, 4 total (ParsesResponse +
HTTPError per method), following `esi/fittings_test.go`'s exact structure. `LoyaltyStoreOffers`'s
test fixture must include a `required_items` entry and an `ak_cost` value to exercise both the
nested slice and the nullable pointer field, and both must be asserted (not left unchecked).

## `mcp/loyalty.go`

Two tools:

- `list_character_loyalty_points` (private) — follows `mcp/fittings.go`'s `list_character_fittings`
  shape exactly (swappable `var` + `characterExists`/`ErrCharacterNotFound` reuse + `esi.NewClient`).
- `list_loyalty_store_offers` (public) — follows `mcp/killmails.go`'s `get_killmail` shape (no
  `characterExists` gate, `esi.NewPublicClient`, only `version` in the handler constructor's args).

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_character_loyalty_points (private) ---

type LoyaltyPointsEntry struct {
	CorporationID int64 `json:"corporationId"`
	LoyaltyPoints int64 `json:"loyaltyPoints"`
}

var listCharacterLoyaltyPoints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]LoyaltyPointsEntry, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	entries, err := client.CharacterLoyaltyPoints(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LoyaltyPointsEntry, len(entries))
	for i, e := range entries {
		out[i] = LoyaltyPointsEntry{CorporationID: e.CorporationId, LoyaltyPoints: e.LoyaltyPoints}
	}
	return out, nil
}

type ListCharacterLoyaltyPointsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterLoyaltyPointsOutput struct {
	CharacterID int32                `json:"characterId"`
	Points      []LoyaltyPointsEntry `json:"points"`
}

func listCharacterLoyaltyPointsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterLoyaltyPointsInput, ListCharacterLoyaltyPointsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterLoyaltyPointsInput) (*mcpsdk.CallToolResult, ListCharacterLoyaltyPointsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterLoyaltyPointsOutput{}, err
		}
		if !found {
			return nil, ListCharacterLoyaltyPointsOutput{}, fmt.Errorf("list character loyalty points for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		points, err := listCharacterLoyaltyPoints(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterLoyaltyPointsOutput{}, fmt.Errorf("list character loyalty points: %w", err)
		}
		return nil, ListCharacterLoyaltyPointsOutput{CharacterID: in.CharacterID, Points: points}, nil
	}
}

// --- list_loyalty_store_offers (public) ---

type LoyaltyRequiredItem struct {
	TypeID   int64 `json:"typeId"`
	Quantity int64 `json:"quantity"`
}

type LoyaltyStoreOffer struct {
	OfferID       int64                 `json:"offerId"`
	TypeID        int64                 `json:"typeId"`
	Quantity      int64                 `json:"quantity"`
	IskCost       int64                 `json:"iskCost"`
	LpCost        int64                 `json:"lpCost"`
	AkCost        *int64                `json:"akCost,omitempty"`
	RequiredItems []LoyaltyRequiredItem `json:"requiredItems"`
}

var listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
	client := esi.NewPublicClient(userAgent)
	offers, err := client.LoyaltyStoreOffers(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]LoyaltyStoreOffer, len(offers))
	for i, o := range offers {
		items := make([]LoyaltyRequiredItem, len(o.RequiredItems))
		for j, it := range o.RequiredItems {
			items[j] = LoyaltyRequiredItem{TypeID: it.TypeId, Quantity: it.Quantity}
		}
		out[i] = LoyaltyStoreOffer{
			OfferID: o.OfferId, TypeID: o.TypeId, Quantity: o.Quantity,
			IskCost: o.IskCost, LpCost: o.LpCost, AkCost: o.AkCost, RequiredItems: items,
		}
	}
	return out, nil
}

type ListLoyaltyStoreOffersInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID whose loyalty store to look up"`
}
type ListLoyaltyStoreOffersOutput struct {
	CorporationID int64               `json:"corporationId"`
	Offers        []LoyaltyStoreOffer `json:"offers"`
}

func listLoyaltyStoreOffersHandler(version string) mcpsdk.ToolHandlerFor[ListLoyaltyStoreOffersInput, ListLoyaltyStoreOffersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListLoyaltyStoreOffersInput) (*mcpsdk.CallToolResult, ListLoyaltyStoreOffersOutput, error) {
		offers, err := listLoyaltyStoreOffers(ctx, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListLoyaltyStoreOffersOutput{}, fmt.Errorf("list loyalty store offers: %w", err)
		}
		return nil, ListLoyaltyStoreOffersOutput{CorporationID: in.CorporationID, Offers: offers}, nil
	}
}
```

Both `LoyaltyStoreOffer` and `LoyaltyRequiredItem` are flat, non-recursive types (verified against
this project's standing convention against self-referential ESI output types, added after the
Killmails batch) — confirmed by sandbox-verifying a live `mcpsdk.AddTool` registration test against
both handlers with zero schema-generator errors.

`mcp/loyalty_internal_test.go`: 4 test functions — `TestListCharacterLoyaltyPointsHandler_MapsFields`,
`_UnknownCharacter` (following `mcp/fittings_internal_test.go`'s swappable-`var` + `t.Cleanup`
pattern, `keyring.MockInit()` + temp-file `config.Store`), and
`TestListLoyaltyStoreOffersHandler_MapsFields`, `_PropagatesError` (following
`mcp/killmails_internal_test.go`'s public-tool pattern — no character/store setup needed at all).
The store-offers happy-path fixture must include one offer with a populated `RequiredItems` entry
and a non-nil `AkCost`, and the test must assert every field of both, per this project's standing
rule against shallow assertions (a past finding in the Corporation Part 2 batch).

## `mcp/server.go` / `mcp/server_test.go`

Register 2 tools after the `get_character_ship` block: `list_character_loyalty_points`,
`list_loyalty_store_offers`. Update package doc comment. `server_test.go`'s expected-tools list:
135 → 137.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred thirty-five" → "one hundred thirty-seven",
  append the 2 new tool names.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Loyalty row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/loyalty_test.go`,
swappable-`var`-override tests for `mcp/loyalty_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.
