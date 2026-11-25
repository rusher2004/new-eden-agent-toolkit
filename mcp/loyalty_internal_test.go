package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// fakeLoyaltyESI fakes the one ESI call list_character_loyalty_points makes.
type fakeLoyaltyESI struct {
	points []esiapi.CharactersCharacterIdLoyaltyPointsGetInner
	err    error
}

func (f fakeLoyaltyESI) CharacterLoyaltyPoints(context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error) {
	return f.points, f.err
}

// fakeLoyaltyStoreESI fakes list_loyalty_store_offers's one ESI call.
type fakeLoyaltyStoreESI struct {
	offers []esiapi.LoyaltyStoresCorporationIdOffersGetInner
	err    error
}

func (f fakeLoyaltyStoreESI) LoyaltyStoreOffers(context.Context, int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error) {
	return f.offers, f.err
}

func openLoyaltyFake(c loyaltyESI) func(context.Context, int32) (loyaltyESI, error) {
	return func(context.Context, int32) (loyaltyESI, error) { return c, nil }
}

func TestListCharacterLoyaltyPointsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeLoyaltyESI{points: []esiapi.CharactersCharacterIdLoyaltyPointsGetInner{{CorporationId: 1000167, LoyaltyPoints: 15000}}}
	handler := listCharacterLoyaltyPointsHandler(openLoyaltyFake(fake), store)
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

	open := func(context.Context, int32) (loyaltyESI, error) {
		t.Fatal("no ESI client should be opened for an unknown character")
		return nil, nil
	}
	handler := listCharacterLoyaltyPointsHandler(open, store)
	_, _, err := handler(context.Background(), nil, ListCharacterLoyaltyPointsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterLoyaltyPointsHandler_PropagatesESIError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	handler := listCharacterLoyaltyPointsHandler(openLoyaltyFake(fakeLoyaltyESI{err: errBoom}), store)
	_, _, err := handler(context.Background(), nil, ListCharacterLoyaltyPointsInput{CharacterID: 7})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}

func TestListCharacterLoyaltyPointsHandler_PropagatesOpenError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	open := func(context.Context, int32) (loyaltyESI, error) { return nil, errBoom }
	handler := listCharacterLoyaltyPointsHandler(open, store)
	_, _, err := handler(context.Background(), nil, ListCharacterLoyaltyPointsInput{CharacterID: 7})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}

func TestListLoyaltyStoreOffersHandler_MapsFields(t *testing.T) {
	akCost := int64(50)
	fake := fakeLoyaltyStoreESI{offers: []esiapi.LoyaltyStoresCorporationIdOffersGetInner{{
		OfferId: 1, TypeId: 588, Quantity: 1, IskCost: 10000, LpCost: 100, AkCost: &akCost,
		RequiredItems: []esiapi.LoyaltyStoresCorporationIdOffersGetInnerRequiredItemsInner{{TypeId: 34, Quantity: 2}},
	}}}

	handler := listLoyaltyStoreOffersHandler("v1", func() (loyaltyStoreESI, error) { return fake, nil }, noTypesSDE{})
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
	handler := listLoyaltyStoreOffersHandler("v1", func() (loyaltyStoreESI, error) { return fakeLoyaltyStoreESI{err: errBoom}, nil }, noTypesSDE{})
	_, _, err := handler(context.Background(), nil, ListLoyaltyStoreOffersInput{CorporationID: 1000167})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}

func TestListLoyaltyStoreOffersHandler_ResolvesTypeNames(t *testing.T) {
	fake := fakeLoyaltyStoreESI{offers: []esiapi.LoyaltyStoresCorporationIdOffersGetInner{{
		OfferId: 1, TypeId: 587,
		RequiredItems: []esiapi.LoyaltyStoresCorporationIdOffersGetInnerRequiredItemsInner{{TypeId: 588, Quantity: 1}},
	}}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Tritanium"}
	}

	handler := listLoyaltyStoreOffersHandler("v1", func() (loyaltyStoreESI, error) { return fake, nil }, noTypesSDE{})
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

// noTypesSDE is an SDEStore that knows no types: every lookup misses, so name
// enrichment adds nothing. It embeds the interface so only Type needs a body;
// any other call panics, which would flag a handler reaching past what it needs.
type noTypesSDE struct{ SDEStore }

func (noTypesSDE) Type(context.Context, string, int64) (sde.Type, bool, error) {
	return sde.Type{}, false, nil
}
