package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
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

// incursionsESI is the public ESI call list_incursions makes; route
// planning reads incursions through it too.
type incursionsESI interface {
	Incursions(ctx context.Context) ([]esiapi.IncursionsGetInner, error)
}

func listIncursions(ctx context.Context, client incursionsESI) ([]Incursion, error) {
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

// incursionNameCache is the SDE lookups list_incursions makes.
type incursionNameCache interface {
	constellationNameCache
	factionNameCache
	systemNameCache
}

// AddListIncursions registers list_incursions. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListIncursions[C incursionsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore incursionNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_incursions",
		Description: "List every currently active incursion (public data, no character required).",
	}, listIncursionsHandler(userAgent, open, sdeStore))
}

func listIncursionsHandler[C incursionsESI](userAgent string, open func() C, sdeStore incursionNameCache) mcpsdk.ToolHandlerFor[ListIncursionsInput, ListIncursionsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListIncursionsInput) (*mcpsdk.CallToolResult, ListIncursionsOutput, error) {
		incursions, err := listIncursions(ctx, open())
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
		constellationNames := resolveConstellationNames(ctx, sdeStore, userAgent, constellationIDs)
		factionNames := resolveFactionNames(ctx, sdeStore, userAgent, factionIDs)
		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, systemIDs)
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
