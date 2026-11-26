package loyalty_test

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/character"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/loyalty"
	"github.com/rusher2004/new-eden-agent-toolkit/operation"
)

var errBoom = errors.New("boom")

// fakePointsESI fakes the one ESI call Points makes.
type fakePointsESI struct {
	points []esiapi.CharactersCharacterIdLoyaltyPointsGetInner
	err    error
}

func (f fakePointsESI) CharacterLoyaltyPoints(context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error) {
	return f.points, f.err
}

// fakeStoreESI fakes the one ESI call StoreOffers makes.
type fakeStoreESI struct {
	offers []esiapi.LoyaltyStoresCorporationIdOffersGetInner
	err    error
}

func (f fakeStoreESI) LoyaltyStoreOffers(context.Context, int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error) {
	return f.offers, f.err
}

// fakeNames resolves type IDs from a fixed map.
type fakeNames map[int64]string

func (f fakeNames) TypeNames(_ context.Context, ids []int64) map[int64]string {
	out := map[int64]string{}
	for _, id := range ids {
		if n, ok := f[id]; ok {
			out[id] = n
		}
	}
	return out
}

func openPoints(c fakePointsESI) func(context.Context, int32) (fakePointsESI, error) {
	return func(context.Context, int32) (fakePointsESI, error) { return c, nil }
}

func openStore(c fakeStoreESI) func() fakeStoreESI {
	return func() fakeStoreESI { return c }
}

// storeWithPilot returns a config store holding one logged-in character, 7.
func storeWithPilot(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

func TestPoints_MapsFields(t *testing.T) {
	fake := fakePointsESI{points: []esiapi.CharactersCharacterIdLoyaltyPointsGetInner{{CorporationId: 1000167, LoyaltyPoints: 15000}}}
	svc := loyalty.New(openPoints(fake), openStore(fakeStoreESI{}), storeWithPilot(t), fakeNames{})

	out, err := svc.Points(context.Background(), loyalty.PointsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("Points returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Points) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one point entry", out)
	}
	if out.Points[0].CorporationID != 1000167 || out.Points[0].LoyaltyPoints != 15000 {
		t.Errorf("out.Points[0] = %+v, want CorporationID=1000167 LoyaltyPoints=15000", out.Points[0])
	}
}

func TestPoints_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	open := func(context.Context, int32) (fakePointsESI, error) {
		t.Fatal("no ESI client should be opened for an unknown character")
		return fakePointsESI{}, nil
	}
	svc := loyalty.New(open, openStore(fakeStoreESI{}), store, fakeNames{})

	_, err := svc.Points(context.Background(), loyalty.PointsInput{CharacterID: 999})
	if !errors.Is(err, character.ErrNotFound) {
		t.Errorf("err = %v, want wrapping character.ErrNotFound", err)
	}
}

func TestPoints_PropagatesESIError(t *testing.T) {
	svc := loyalty.New(openPoints(fakePointsESI{err: errBoom}), openStore(fakeStoreESI{}), storeWithPilot(t), fakeNames{})

	_, err := svc.Points(context.Background(), loyalty.PointsInput{CharacterID: 7})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}

func TestPoints_PropagatesOpenError(t *testing.T) {
	open := func(context.Context, int32) (*fakePointsESI, error) { return nil, errBoom }
	svc := loyalty.New(open, openStore(fakeStoreESI{}), storeWithPilot(t), fakeNames{})

	_, err := svc.Points(context.Background(), loyalty.PointsInput{CharacterID: 7})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}

func TestStoreOffers_MapsFields(t *testing.T) {
	akCost := int64(50)
	fake := fakeStoreESI{offers: []esiapi.LoyaltyStoresCorporationIdOffersGetInner{{
		OfferId: 1, TypeId: 588, Quantity: 1, IskCost: 10000, LpCost: 100, AkCost: &akCost,
		RequiredItems: []esiapi.LoyaltyStoresCorporationIdOffersGetInnerRequiredItemsInner{{TypeId: 34, Quantity: 2}},
	}}}
	svc := loyalty.New(openPoints(fakePointsESI{}), openStore(fake), storeWithPilot(t), fakeNames{})

	out, err := svc.StoreOffers(context.Background(), loyalty.StoreOffersInput{CorporationID: 1000167})
	if err != nil {
		t.Fatalf("StoreOffers returned error: %v", err)
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

func TestStoreOffers_PropagatesError(t *testing.T) {
	svc := loyalty.New(openPoints(fakePointsESI{}), openStore(fakeStoreESI{err: errBoom}), storeWithPilot(t), fakeNames{})

	_, err := svc.StoreOffers(context.Background(), loyalty.StoreOffersInput{CorporationID: 1000167})
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want wrapping errBoom", err)
	}
}

func TestStoreOffers_ResolvesTypeNames(t *testing.T) {
	fake := fakeStoreESI{offers: []esiapi.LoyaltyStoresCorporationIdOffersGetInner{{
		OfferId: 1, TypeId: 587,
		RequiredItems: []esiapi.LoyaltyStoresCorporationIdOffersGetInnerRequiredItemsInner{{TypeId: 588, Quantity: 1}},
	}}}
	names := fakeNames{587: "Rifter", 588: "Tritanium"}
	svc := loyalty.New(openPoints(fakePointsESI{}), openStore(fake), storeWithPilot(t), names)

	out, err := svc.StoreOffers(context.Background(), loyalty.StoreOffersInput{CorporationID: 1})
	if err != nil {
		t.Fatalf("StoreOffers returned error: %v", err)
	}
	if out.Offers[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Offers[0].TypeName, "Rifter")
	}
	if out.Offers[0].RequiredItems[0].TypeName != "Tritanium" {
		t.Errorf("RequiredItems[0].TypeName = %q, want %q", out.Offers[0].RequiredItems[0].TypeName, "Tritanium")
	}
}

func TestOperations_NamesAndKinds(t *testing.T) {
	svc := loyalty.New(openPoints(fakePointsESI{}), openStore(fakeStoreESI{}), storeWithPilot(t), fakeNames{})

	ops := svc.Operations()
	want := []string{"list_character_loyalty_points", "list_loyalty_store_offers"}
	if len(ops) != len(want) {
		t.Fatalf("len(Operations()) = %d, want %d", len(ops), len(want))
	}
	for i, op := range ops {
		if op.Name != want[i] {
			t.Errorf("ops[%d].Name = %q, want %q", i, op.Name, want[i])
		}
		if op.Kind != operation.Read {
			t.Errorf("ops[%d] (%s).Kind = %v, want operation.Read", i, op.Name, op.Kind)
		}
	}
}
