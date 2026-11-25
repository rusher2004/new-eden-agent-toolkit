package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// insuranceESI is the ESI call list_insurance_prices makes; it is public.
type insuranceESI interface {
	InsurancePrices(ctx context.Context) ([]esiapi.InsurancePricesGetInner, error)
}

// --- list_insurance_prices (public, no input) ---

type InsuranceLevel struct {
	Cost   float64 `json:"cost"`
	Payout float64 `json:"payout"`
	Name   string  `json:"name"`
}

type InsurancePrice struct {
	TypeID   int64            `json:"typeId"`
	TypeName string           `json:"typeName,omitempty"`
	Levels   []InsuranceLevel `json:"levels"`
}

func listInsurancePrices(ctx context.Context, client insuranceESI) ([]InsurancePrice, error) {
	prices, err := client.InsurancePrices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]InsurancePrice, len(prices))
	for i, p := range prices {
		levels := make([]InsuranceLevel, len(p.Levels))
		for j, l := range p.Levels {
			levels[j] = InsuranceLevel{Cost: l.Cost, Payout: l.Payout, Name: l.Name}
		}
		out[i] = InsurancePrice{TypeID: p.TypeId, Levels: levels}
	}
	return out, nil
}

type ListInsurancePricesInput struct{}
type ListInsurancePricesOutput struct {
	Prices []InsurancePrice `json:"prices"`
}

// AddListInsurancePrices registers list_insurance_prices. open builds the ESI client; any client with the
// insuranceESI methods will do.
func AddListInsurancePrices[C insuranceESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_insurance_prices",
		Description: "List insurance price levels for every insurable ship type (public data, no character required).",
	}, listInsurancePricesHandler(userAgent, open, sdeStore))
}

func listInsurancePricesHandler[C insuranceESI](userAgent string, open func() C, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListInsurancePricesInput, ListInsurancePricesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListInsurancePricesInput) (*mcpsdk.CallToolResult, ListInsurancePricesOutput, error) {
		prices, err := listInsurancePrices(ctx, open())
		if err != nil {
			return nil, ListInsurancePricesOutput{}, fmt.Errorf("list insurance prices: %w", err)
		}
		ids := make([]int64, len(prices))
		for i, p := range prices {
			ids[i] = p.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range prices {
			prices[i].TypeName = names[prices[i].TypeID]
		}
		return nil, ListInsurancePricesOutput{Prices: prices}, nil
	}
}
