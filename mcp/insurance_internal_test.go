package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestListInsurancePricesHandler_MapsFields(t *testing.T) {
	orig := listInsurancePrices
	t.Cleanup(func() { listInsurancePrices = orig })
	listInsurancePrices = func(ctx context.Context, userAgent string) ([]InsurancePrice, error) {
		return []InsurancePrice{
			{
				TypeID: 587,
				Levels: []InsuranceLevel{
					{Cost: 900.0, Payout: 450.0, Name: "Standard"},
					{Cost: 1350.0, Payout: 675.0, Name: "Extended"},
				},
			},
		}, nil
	}

	handler := listInsurancePricesHandler("v1")
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
	orig := listInsurancePrices
	t.Cleanup(func() { listInsurancePrices = orig })
	wantErr := errors.New("upstream failure")
	listInsurancePrices = func(ctx context.Context, userAgent string) ([]InsurancePrice, error) {
		return nil, wantErr
	}

	handler := listInsurancePricesHandler("v1")
	_, _, err := handler(context.Background(), nil, ListInsurancePricesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
