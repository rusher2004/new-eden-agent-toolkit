package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterLoyaltyPointsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterLoyaltyPoints
	t.Cleanup(func() { listCharacterLoyaltyPoints = orig })
	listCharacterLoyaltyPoints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]LoyaltyPointsEntry, error) {
		return []LoyaltyPointsEntry{{CorporationID: 1000167, LoyaltyPoints: 15000}}, nil
	}

	handler := listCharacterLoyaltyPointsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterLoyaltyPointsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Points) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one point entry", out)
	}
	if out.Points[0].CorporationID != 1000167 || out.Points[0].LoyaltyPoints != 15000 {
		t.Errorf("out.Points[0] = %+v, want CorporationID=1000167 LoyaltyPoints=15000", out.Points[0])
	}
}

func TestListCharacterLoyaltyPointsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterLoyaltyPoints
	t.Cleanup(func() { listCharacterLoyaltyPoints = orig })
	listCharacterLoyaltyPoints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]LoyaltyPointsEntry, error) {
		t.Fatal("listCharacterLoyaltyPoints should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterLoyaltyPointsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterLoyaltyPointsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListLoyaltyStoreOffersHandler_MapsFields(t *testing.T) {
	orig := listLoyaltyStoreOffers
	t.Cleanup(func() { listLoyaltyStoreOffers = orig })
	akCost := int64(50)
	listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
		return []LoyaltyStoreOffer{{
			OfferID: 1, TypeID: 588, Quantity: 1, IskCost: 10000, LpCost: 100, AkCost: &akCost,
			RequiredItems: []LoyaltyRequiredItem{{TypeID: 34, Quantity: 2}},
		}}, nil
	}

	handler := listLoyaltyStoreOffersHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListLoyaltyStoreOffersInput{CorporationID: 1000167})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 1000167 || len(out.Offers) != 1 {
		t.Fatalf("out = %+v, want CorporationID=1000167 with one offer", out)
	}
	o := out.Offers[0]
	if o.OfferID != 1 || o.TypeID != 588 || o.Quantity != 1 || o.IskCost != 10000 || o.LpCost != 100 {
		t.Errorf("offer = %+v, want OfferID=1 TypeID=588 Quantity=1 IskCost=10000 LpCost=100", o)
	}
	if o.AkCost == nil || *o.AkCost != 50 {
		t.Errorf("AkCost = %v, want pointer to 50", o.AkCost)
	}
	if len(o.RequiredItems) != 1 || o.RequiredItems[0].TypeID != 34 || o.RequiredItems[0].Quantity != 2 {
		t.Errorf("RequiredItems = %+v, want one entry {TypeID:34 Quantity:2}", o.RequiredItems)
	}
}

func TestListLoyaltyStoreOffersHandler_PropagatesError(t *testing.T) {
	orig := listLoyaltyStoreOffers
	t.Cleanup(func() { listLoyaltyStoreOffers = orig })
	listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
		return nil, errBoom
	}

	handler := listLoyaltyStoreOffersHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListLoyaltyStoreOffersInput{CorporationID: 1000167})
	if err == nil {
		t.Fatal("handler returned nil error, want the propagated error")
	}
}

func TestListLoyaltyStoreOffersHandler_ResolvesTypeNames(t *testing.T) {
	orig := listLoyaltyStoreOffers
	t.Cleanup(func() { listLoyaltyStoreOffers = orig })
	listLoyaltyStoreOffers = func(ctx context.Context, userAgent string, corporationID int64) ([]LoyaltyStoreOffer, error) {
		return []LoyaltyStoreOffer{{
			OfferID: 1, TypeID: 587,
			RequiredItems: []LoyaltyRequiredItem{{TypeID: 588, Quantity: 1}},
		}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Tritanium"}
	}

	handler := listLoyaltyStoreOffersHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListLoyaltyStoreOffersInput{CorporationID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Offers[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Offers[0].TypeName, "Rifter")
	}
	if out.Offers[0].RequiredItems[0].TypeName != "Tritanium" {
		t.Errorf("RequiredItems[0].TypeName = %q, want %q", out.Offers[0].RequiredItems[0].TypeName, "Tritanium")
	}
}

var errBoom = errors.New("boom")
