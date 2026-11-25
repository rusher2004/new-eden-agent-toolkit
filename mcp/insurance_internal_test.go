package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fakeInsuranceESI fakes list_insurance_prices's one ESI call.
type fakeInsuranceESI struct {
	prices []esiapi.InsurancePricesGetInner
	err    error
}

func (f fakeInsuranceESI) InsurancePrices(context.Context) ([]esiapi.InsurancePricesGetInner, error) {
	return f.prices, f.err
}

func openInsuranceFake(c insuranceESI) func() insuranceESI { return func() insuranceESI { return c } }

func TestListInsurancePricesHandler_MapsFields(t *testing.T) {
	fake := fakeInsuranceESI{prices: []esiapi.InsurancePricesGetInner{
		{
			TypeId: 587,
			Levels: []esiapi.InsurancePricesGetInnerLevelsInner{
				{Cost: 900.0, Payout: 450.0, Name: "Standard"},
				{Cost: 1350.0, Payout: 675.0, Name: "Extended"},
			},
		},
	}}

	handler := listInsurancePricesHandler("v1", openInsuranceFake(fake), nil)
	_, out, err := handler(context.Background(), nil, ListInsurancePricesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Prices) != 1 {
		t.Errorf("len(out.Prices) = %d, want 1", len(out.Prices))
	}
	price := out.Prices[0]
	if price.TypeID != 587 {
		t.Errorf("TypeID = %d, want 587", price.TypeID)
	}
	if len(price.Levels) != 2 {
		t.Errorf("len(Levels) = %d, want 2", len(price.Levels))
	}
	if price.Levels[0].Cost != 900.0 {
		t.Errorf("Levels[0].Cost = %f, want 900.0", price.Levels[0].Cost)
	}
	if price.Levels[0].Payout != 450.0 {
		t.Errorf("Levels[0].Payout = %f, want 450.0", price.Levels[0].Payout)
	}
	if price.Levels[0].Name != "Standard" {
		t.Errorf("Levels[0].Name = %s, want Standard", price.Levels[0].Name)
	}
	if price.Levels[1].Cost != 1350.0 {
		t.Errorf("Levels[1].Cost = %f, want 1350.0", price.Levels[1].Cost)
	}
	if price.Levels[1].Payout != 675.0 {
		t.Errorf("Levels[1].Payout = %f, want 675.0", price.Levels[1].Payout)
	}
	if price.Levels[1].Name != "Extended" {
		t.Errorf("Levels[1].Name = %s, want Extended", price.Levels[1].Name)
	}
}

func TestListInsurancePricesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := listInsurancePricesHandler("v1", openInsuranceFake(fakeInsuranceESI{err: wantErr}), nil)
	_, _, err := handler(context.Background(), nil, ListInsurancePricesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListInsurancePricesHandler_ResolvesTypeName(t *testing.T) {
	fake := fakeInsuranceESI{prices: []esiapi.InsurancePricesGetInner{{TypeId: 587, Levels: []esiapi.InsurancePricesGetInnerLevelsInner{{Cost: 900.0, Payout: 450.0, Name: "Standard"}}}}}

	orig2 := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = orig2 })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listInsurancePricesHandler("v1", openInsuranceFake(fake), nil)
	_, out, err := handler(context.Background(), nil, ListInsurancePricesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Prices[0].TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.Prices[0].TypeName, "Rifter")
	}
}
