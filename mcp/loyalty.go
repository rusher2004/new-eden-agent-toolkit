package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// loyaltyESI is the ESI calls list_character_loyalty_points makes.
type loyaltyESI interface {
	CharacterLoyaltyPoints(ctx context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error)
}

// loyaltyStoreESI is the ESI call list_loyalty_store_offers makes.
type loyaltyStoreESI interface {
	LoyaltyStoreOffers(ctx context.Context, corporationID int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error)
}

// --- list_character_loyalty_points (private) ---

type LoyaltyPointsEntry struct {
	CorporationID int64 `json:"corporationId"`
	LoyaltyPoints int64 `json:"loyaltyPoints"`
}

func listCharacterLoyaltyPoints(ctx context.Context, client loyaltyESI) ([]LoyaltyPointsEntry, error) {
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

// AddListCharacterLoyaltyPoints registers list_character_loyalty_points. open
// builds the character's ESI client; any client with the loyaltyESI methods
// will do.
func AddListCharacterLoyaltyPoints[C loyaltyESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_loyalty_points",
		Description: "List a logged-in EVE Online character's loyalty point balances with every corporation they've worked for.",
	}, listCharacterLoyaltyPointsHandler(open, store))
}

func listCharacterLoyaltyPointsHandler[C loyaltyESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterLoyaltyPointsInput, ListCharacterLoyaltyPointsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterLoyaltyPointsInput) (*mcpsdk.CallToolResult, ListCharacterLoyaltyPointsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterLoyaltyPointsOutput{}, err
		}
		if !found {
			return nil, ListCharacterLoyaltyPointsOutput{}, fmt.Errorf("list character loyalty points for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterLoyaltyPointsOutput{}, fmt.Errorf("list character loyalty points: %w", err)
		}
		points, err := listCharacterLoyaltyPoints(ctx, client)
		if err != nil {
			return nil, ListCharacterLoyaltyPointsOutput{}, fmt.Errorf("list character loyalty points: %w", err)
		}
		return nil, ListCharacterLoyaltyPointsOutput{CharacterID: in.CharacterID, Points: points}, nil
	}
}

// --- list_loyalty_store_offers (public) ---

type LoyaltyRequiredItem struct {
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
	Quantity int64  `json:"quantity"`
}

type LoyaltyStoreOffer struct {
	OfferID       int64                 `json:"offerId"`
	TypeID        int64                 `json:"typeId"`
	TypeName      string                `json:"typeName,omitempty"`
	Quantity      int64                 `json:"quantity"`
	IskCost       int64                 `json:"iskCost"`
	LpCost        int64                 `json:"lpCost"`
	AkCost        *int64                `json:"akCost,omitempty"`
	RequiredItems []LoyaltyRequiredItem `json:"requiredItems"`
}

func listLoyaltyStoreOffers(ctx context.Context, client loyaltyStoreESI, corporationID int64) ([]LoyaltyStoreOffer, error) {
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

// AddListLoyaltyStoreOffers registers list_loyalty_store_offers. open builds a
// public ESI client; names resolves the type names added to each offer.
func AddListLoyaltyStoreOffers[C loyaltyStoreESI](s *mcpsdk.Server, userAgent string, open func() C, names typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_loyalty_store_offers",
		Description: "List a corporation's loyalty store offers (public data, no character required).",
	}, listLoyaltyStoreOffersHandler(userAgent, open, names))
}

func listLoyaltyStoreOffersHandler[C loyaltyStoreESI](userAgent string, open func() C, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListLoyaltyStoreOffersInput, ListLoyaltyStoreOffersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListLoyaltyStoreOffersInput) (*mcpsdk.CallToolResult, ListLoyaltyStoreOffersOutput, error) {
		offers, err := listLoyaltyStoreOffers(ctx, open(), in.CorporationID)
		if err != nil {
			return nil, ListLoyaltyStoreOffersOutput{}, fmt.Errorf("list loyalty store offers: %w", err)
		}
		// Collect type IDs from both levels for enrichment
		ids := make([]int64, 0, len(offers))
		for _, o := range offers {
			ids = append(ids, o.TypeID)
			for _, it := range o.RequiredItems {
				ids = append(ids, it.TypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		// Enrich both outer offer type and nested required item types
		for i := range offers {
			offers[i].TypeName = names[offers[i].TypeID]
			for j := range offers[i].RequiredItems {
				offers[i].RequiredItems[j].TypeName = names[offers[i].RequiredItems[j].TypeID]
			}
		}
		return nil, ListLoyaltyStoreOffersOutput{CorporationID: in.CorporationID, Offers: offers}, nil
	}
}
