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
