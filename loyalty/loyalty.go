// Package loyalty answers questions about loyalty points: a character's
// balances with each corporation, and what a corporation's loyalty store
// offers. Its operations are exposed by the MCP server and the CLI through
// operation.Operation.
package loyalty

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/character"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

// characterESI is the ESI call Points makes for a character.
type characterESI interface {
	CharacterLoyaltyPoints(ctx context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error)
}

// storeESI is the public ESI call StoreOffers makes.
type storeESI interface {
	LoyaltyStoreOffers(ctx context.Context, corporationID int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error)
}

// loader is the config call the logged-in check makes. *config.Store
// satisfies it.
type loader interface {
	Load() (config.Config, error)
}

// typeNames resolves type IDs to names for the additive TypeName fields.
type typeNames interface {
	TypeNames(ctx context.Context, ids []int64) map[int64]string
}

// Service holds the loyalty operations.
type Service struct {
	openCharacter func(ctx context.Context, characterID int32) (characterESI, error)
	openPublic    func() storeESI
	characters    loader
	names         typeNames
}

// New returns a Service. openCharacter builds a character's ESI client,
// openPublic a public one, characters is the logged-in character registry and
// names resolves type names. It is generic so callers outside the package can
// pass concrete values without naming the unexported interfaces.
func New[C characterESI, P storeESI, L loader, N typeNames](
	openCharacter func(ctx context.Context, characterID int32) (C, error),
	openPublic func() P, characters L, names N,
) *Service {
	return &Service{
		openCharacter: func(ctx context.Context, characterID int32) (characterESI, error) {
			c, err := openCharacter(ctx, characterID)
			if err != nil {
				// Untyped nil: a typed nil C inside characterESI would compare non-nil.
				return nil, err
			}
			return c, nil
		},
		openPublic: func() storeESI { return openPublic() },
		characters: characters,
		names:      names,
	}
}

// Operations wraps each method as an operation.Operation.
func (s *Service) Operations() []operation.Operation {
	return []operation.Operation{
		operation.New("list_character_loyalty_points",
			"List a logged-in EVE Online character's loyalty point balances with every corporation they've worked for.",
			operation.Read, s.Points),
		operation.New("list_loyalty_store_offers",
			"List a corporation's loyalty store offers (public data, no character required).",
			operation.Read, s.StoreOffers),
	}
}

// --- list_character_loyalty_points (private) ---

type LoyaltyPointsEntry struct {
	CorporationID int64 `json:"corporationId"`
	LoyaltyPoints int64 `json:"loyaltyPoints"`
}

type PointsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type PointsOutput struct {
	CharacterID int32                `json:"characterId"`
	Points      []LoyaltyPointsEntry `json:"points"`
}

// Points lists a logged-in character's loyalty point balances. An unknown
// character fails with character.ErrNotFound before any ESI client is opened.
func (s *Service) Points(ctx context.Context, in PointsInput) (PointsOutput, error) {
	found, err := character.Known(s.characters, in.CharacterID)
	if err != nil {
		return PointsOutput{}, fmt.Errorf("list character loyalty points for character %d: %w", in.CharacterID, err)
	}
	if !found {
		return PointsOutput{}, fmt.Errorf("list character loyalty points for character %d: %w", in.CharacterID, character.ErrNotFound)
	}
	client, err := s.openCharacter(ctx, in.CharacterID)
	if err != nil {
		return PointsOutput{}, fmt.Errorf("list character loyalty points: %w", err)
	}
	entries, err := client.CharacterLoyaltyPoints(ctx)
	if err != nil {
		return PointsOutput{}, fmt.Errorf("list character loyalty points: %w", err)
	}
	points := make([]LoyaltyPointsEntry, len(entries))
	for i, e := range entries {
		points[i] = LoyaltyPointsEntry{CorporationID: e.CorporationId, LoyaltyPoints: e.LoyaltyPoints}
	}
	return PointsOutput{CharacterID: in.CharacterID, Points: points}, nil
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

type StoreOffersInput struct {
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID whose loyalty store to look up"`
}
type StoreOffersOutput struct {
	CorporationID int64               `json:"corporationId"`
	Offers        []LoyaltyStoreOffer `json:"offers"`
}

// StoreOffers lists a corporation's loyalty store offers, naming each offer's
// type and its required items' types.
func (s *Service) StoreOffers(ctx context.Context, in StoreOffersInput) (StoreOffersOutput, error) {
	raw, err := s.openPublic().LoyaltyStoreOffers(ctx, in.CorporationID)
	if err != nil {
		return StoreOffersOutput{}, fmt.Errorf("list loyalty store offers: %w", err)
	}
	offers := make([]LoyaltyStoreOffer, len(raw))
	ids := make([]int64, 0, len(raw))
	for i, o := range raw {
		items := make([]LoyaltyRequiredItem, len(o.RequiredItems))
		ids = append(ids, o.TypeId)
		for j, it := range o.RequiredItems {
			items[j] = LoyaltyRequiredItem{TypeID: it.TypeId, Quantity: it.Quantity}
			ids = append(ids, it.TypeId)
		}
		offers[i] = LoyaltyStoreOffer{
			OfferID: o.OfferId, TypeID: o.TypeId, Quantity: o.Quantity,
			IskCost: o.IskCost, LpCost: o.LpCost, AkCost: o.AkCost, RequiredItems: items,
		}
	}
	// Additive enrichment: a type the lookup can't name keeps an empty
	// TypeName rather than failing the call.
	names := s.names.TypeNames(ctx, ids)
	for i := range offers {
		offers[i].TypeName = names[offers[i].TypeID]
		for j := range offers[i].RequiredItems {
			offers[i].RequiredItems[j].TypeName = names[offers[i].RequiredItems[j].TypeID]
		}
	}
	return StoreOffersOutput{CorporationID: in.CorporationID, Offers: offers}, nil
}
