package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

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

var listInsurancePrices = func(ctx context.Context, userAgent string) ([]InsurancePrice, error) {
	client := esi.NewPublicClient(userAgent)
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

func listInsurancePricesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListInsurancePricesInput, ListInsurancePricesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListInsurancePricesInput) (*mcpsdk.CallToolResult, ListInsurancePricesOutput, error) {
		prices, err := listInsurancePrices(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListInsurancePricesOutput{}, fmt.Errorf("list insurance prices: %w", err)
		}
		ids := make([]int64, len(prices))
		for i, p := range prices {
			ids[i] = p.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range prices {
			prices[i].TypeName = names[prices[i].TypeID]
		}
		return nil, ListInsurancePricesOutput{Prices: prices}, nil
	}
}
