package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_incursions (public, no input) ---

type Incursion struct {
	ConstellationID          int64    `json:"constellationId"`
	ConstellationName        string   `json:"constellationName,omitempty"`
	FactionID                int64    `json:"factionId"`
	FactionName              string   `json:"factionName,omitempty"`
	HasBoss                  bool     `json:"hasBoss"`
	InfestedSolarSystemIDs   []int64  `json:"infestedSolarSystemIds"`
	InfestedSolarSystemNames []string `json:"infestedSolarSystemNames,omitempty"`
	Influence                float64  `json:"influence"`
	StagingSolarSystemID     int64    `json:"stagingSolarSystemId"`
	StagingSolarSystemName   string   `json:"stagingSolarSystemName,omitempty"`
	State                    string   `json:"state"`
	Type                     string   `json:"type"`
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

func listIncursionsHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListIncursionsInput, ListIncursionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIncursionsInput) (*mcpsdk.CallToolResult, ListIncursionsOutput, error) {
		incursions, err := listIncursions(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListIncursionsOutput{}, fmt.Errorf("list incursions: %w", err)
		}

		var constellationIDs, systemIDs, factionIDs []int64
		for _, inc := range incursions {
			constellationIDs = append(constellationIDs, inc.ConstellationID)
			factionIDs = append(factionIDs, inc.FactionID)
			systemIDs = append(systemIDs, inc.StagingSolarSystemID)
			systemIDs = append(systemIDs, inc.InfestedSolarSystemIDs...)
		}
		constellationNames := resolveConstellationNames(ctx, sdeStore, sso.UserAgent(version), constellationIDs)
		factionNames := resolveFactionNames(ctx, sdeStore, sso.UserAgent(version), factionIDs)
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		for i := range incursions {
			incursions[i].ConstellationName = constellationNames[incursions[i].ConstellationID]
			incursions[i].FactionName = factionNames[incursions[i].FactionID]
			incursions[i].StagingSolarSystemName = systemNames[incursions[i].StagingSolarSystemID]
			infestedNames := make([]string, len(incursions[i].InfestedSolarSystemIDs))
			var resolved bool
			for j, id := range incursions[i].InfestedSolarSystemIDs {
				if name, ok := systemNames[id]; ok {
					infestedNames[j] = name
					resolved = true
				}
			}
			if resolved {
				incursions[i].InfestedSolarSystemNames = infestedNames
			}
		}

		return nil, ListIncursionsOutput{Incursions: incursions}, nil
	}
}
