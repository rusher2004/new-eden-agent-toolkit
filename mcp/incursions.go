package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_incursions (public, no input) ---

type Incursion struct {
	ConstellationID        int64   `json:"constellationId"`
	FactionID              int64   `json:"factionId"`
	HasBoss                bool    `json:"hasBoss"`
	InfestedSolarSystemIDs []int64 `json:"infestedSolarSystemIds"`
	Influence              float64 `json:"influence"`
	StagingSolarSystemID   int64   `json:"stagingSolarSystemId"`
	State                  string  `json:"state"`
	Type                   string  `json:"type"`
}

var listIncursions = func(ctx context.Context, userAgent string) ([]Incursion, error) {
	client := esi.NewPublicClient(userAgent)
	incursions, err := client.Incursions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Incursion, len(incursions))
	for i, inc := range incursions {
		out[i] = Incursion{
			ConstellationID: inc.ConstellationId, FactionID: inc.FactionId, HasBoss: inc.HasBoss,
			InfestedSolarSystemIDs: inc.InfestedSolarSystems, Influence: inc.Influence,
			StagingSolarSystemID: inc.StagingSolarSystemId, State: inc.State, Type: inc.Type,
		}
	}
	return out, nil
}

type ListIncursionsInput struct{}
type ListIncursionsOutput struct {
	Incursions []Incursion `json:"incursions"`
}

func listIncursionsHandler(version string) mcpsdk.ToolHandlerFor[ListIncursionsInput, ListIncursionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIncursionsInput) (*mcpsdk.CallToolResult, ListIncursionsOutput, error) {
		incursions, err := listIncursions(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListIncursionsOutput{}, fmt.Errorf("list incursions: %w", err)
		}
		return nil, ListIncursionsOutput{Incursions: incursions}, nil
	}
}
