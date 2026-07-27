# ESI Sovereignty — Design Spec (thirtieth batch)

**Batch:** Sovereignty tag, 2 endpoints, both public, no pagination.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

- `GetSovereigntyCampaigns` → `[]SovereigntyCampaignsGetInner{CampaignId, ConstellationId,
  SolarSystemId, StructureId int64, EventType string, StartTime time.Time (all required),
  AttackersScore *float64, DefenderId *int64, DefenderScore *float64 (only present in Defense
  Events), Participants []{AllianceId int64, Score float64} (only present in Freeport Events)}`.
  Public — `/sovereignty/campaigns` has no security block. Not paginated.
- `GetSovereigntySystems` → `*SovereigntySystems{SolarSystems []SovereigntySystemsSolarsystem{
  SolarSystemId int64, Claim SovereigntySystemsSolarsystemClaim}}`. Public — `/sovereignty/systems`
  has no security block. Not paginated. `esi/sovereignty.go` unwraps `SolarSystems` directly since
  the top-level struct wraps exactly one field.

**The nested oneOf (this batch's one genuine complexity)**: each solar system's `Claim` is ESI's
own discriminated union — `SovereigntySystemsSolarsystemClaim{Alliance *Alliance, Faction
*Faction, Unclaimed *Unclaimed}`, exactly one non-nil. `Alliance` wraps
`*SovereigntySystemsAlliance{AllianceId, CorporationId int64, ClaimedSince time.Time,
IsCapitalSystem bool, Development SovereigntySystemsDevelopment{ActivityDefenseMultiplier float64,
IndustrialLevel/MilitaryLevel/StrategicLevel int64}, SovereigntyHub
SovereigntySystemsSovereigntyhub{Id int64, VulnerabilityWindow
*SovereigntySystemsVulnerabilitywindow{Start, End time.Time}}}`. `Faction` wraps
`*SovereigntySystemsFaction{FactionId int64}`. `Unclaimed` wraps `*bool`. **No recursion anywhere**
— confirmed field-by-field across all 7 generated model files; nothing in the nested detail loops
back to `Claim`, `SolarSystem`, or itself.

## `esi/sovereignty.go`

```go
// Package esi's sovereignty.go covers both of "Sovereignty"'s endpoints -- active sovereignty
// campaigns and the current sovereignty claim (if any) for every solar system. Both are public
// (no security block in esi/api/openapi.yaml) and not paginated.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// SovereigntyCampaigns returns every active sovereignty campaign in the game.
func (c *Client) SovereigntyCampaigns(ctx context.Context) ([]esiapi.SovereigntyCampaignsGetInner, error) {
	result, resp, err := c.api.SovereigntyAPI.GetSovereigntyCampaigns(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get sovereignty campaigns: %w", describeError(err, resp))
	}
	return result, nil
}

// SovereigntySystems returns every solar system and its current sovereignty claim (if any).
func (c *Client) SovereigntySystems(ctx context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error) {
	result, resp, err := c.api.SovereigntyAPI.GetSovereigntySystems(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get sovereignty systems: %w", describeError(err, resp))
	}
	return result.SolarSystems, nil
}
```

`esi/sovereignty_test.go`: 4 tests (success + HTTP error per method), the systems test covering
all 3 claim variants (alliance with full nested detail, faction, unclaimed).

## `mcp/sovereignty.go`

Two public tools: `list_sovereignty_campaigns`, `list_sovereignty_systems`. The claim union is
flattened into a `Kind`-discriminated `SovereigntyClaim` type, following the same pattern
established for Faction Warfare's leaderboards and Corporation Projects' `Configuration`.

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_sovereignty_campaigns (public) ---

type SovereigntyParticipant struct {
	AllianceID int64   `json:"allianceId"`
	Score      float64 `json:"score"`
}

type SovereigntyCampaign struct {
	CampaignID      int64                    `json:"campaignId"`
	ConstellationID int64                    `json:"constellationId"`
	SolarSystemID   int64                    `json:"solarSystemId"`
	StructureID     int64                    `json:"structureId"`
	EventType       string                   `json:"eventType"`
	StartTime       time.Time                `json:"startTime"`
	AttackersScore  *float64                 `json:"attackersScore,omitempty"`
	DefenderID      *int64                   `json:"defenderId,omitempty"`
	DefenderScore   *float64                 `json:"defenderScore,omitempty"`
	Participants    []SovereigntyParticipant `json:"participants,omitempty"`
}

var listSovereigntyCampaigns = func(ctx context.Context, userAgent string) ([]esiapi.SovereigntyCampaignsGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.SovereigntyCampaigns(ctx)
}

type ListSovereigntyCampaignsInput struct{}
type ListSovereigntyCampaignsOutput struct {
	Campaigns []SovereigntyCampaign `json:"campaigns"`
}

func listSovereigntyCampaignsHandler(version string) mcpsdk.ToolHandlerFor[ListSovereigntyCampaignsInput, ListSovereigntyCampaignsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListSovereigntyCampaignsInput) (*mcpsdk.CallToolResult, ListSovereigntyCampaignsOutput, error) {
		campaigns, err := listSovereigntyCampaigns(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListSovereigntyCampaignsOutput{}, fmt.Errorf("list sovereignty campaigns: %w", err)
		}
		out := make([]SovereigntyCampaign, len(campaigns))
		for i, c := range campaigns {
			participants := make([]SovereigntyParticipant, len(c.Participants))
			for j, p := range c.Participants {
				participants[j] = SovereigntyParticipant{AllianceID: p.AllianceId, Score: p.Score}
			}
			out[i] = SovereigntyCampaign{
				CampaignID: c.CampaignId, ConstellationID: c.ConstellationId, SolarSystemID: c.SolarSystemId,
				StructureID: c.StructureId, EventType: c.EventType, StartTime: c.StartTime,
				AttackersScore: c.AttackersScore, DefenderID: c.DefenderId, DefenderScore: c.DefenderScore,
				Participants: participants,
			}
		}
		return nil, ListSovereigntyCampaignsOutput{Campaigns: out}, nil
	}
}

// --- list_sovereignty_systems (public) ---
//
// Each solar system's Claim is ESI's own oneOf discriminated union of exactly one of Alliance,
// Faction, or Unclaimed (esiapi.SovereigntySystemsSolarsystemClaim{Alliance *Alliance, Faction
// *Faction, Unclaimed *Unclaimed}) -- flattened here into a single Kind-discriminated
// SovereigntyClaim type, following the same pattern established for Faction Warfare's
// leaderboards and Corporation Projects' Configuration. Not a self-referential type: Alliance's
// nested Development/SovereigntyHub/VulnerabilityWindow details go one level deeper but never
// loop back to SovereigntyClaim, SovereigntySystem, or each other.

type SovereigntyDevelopment struct {
	ActivityDefenseMultiplier float64 `json:"activityDefenseMultiplier"`
	IndustrialLevel           int64   `json:"industrialLevel"`
	MilitaryLevel             int64   `json:"militaryLevel"`
	StrategicLevel            int64   `json:"strategicLevel"`
}

type SovereigntyVulnerabilityWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type SovereigntyHub struct {
	ID                  int64                           `json:"id"`
	VulnerabilityWindow *SovereigntyVulnerabilityWindow `json:"vulnerabilityWindow,omitempty"`
}

type SovereigntyClaim struct {
	Kind            string                  `json:"kind"` // alliance, faction, or unclaimed
	AllianceID      *int64                  `json:"allianceId,omitempty"`
	CorporationID   *int64                  `json:"corporationId,omitempty"`
	ClaimedSince    *time.Time              `json:"claimedSince,omitempty"`
	IsCapitalSystem *bool                   `json:"isCapitalSystem,omitempty"`
	Development     *SovereigntyDevelopment `json:"development,omitempty"`
	SovereigntyHub  *SovereigntyHub         `json:"sovereigntyHub,omitempty"`
	FactionID       *int64                  `json:"factionId,omitempty"`
}

func toSovereigntyClaim(claim esiapi.SovereigntySystemsSolarsystemClaim) SovereigntyClaim {
	if claim.Alliance != nil && claim.Alliance.Alliance != nil {
		a := claim.Alliance.Alliance
		isCapital := a.IsCapitalSystem
		var hub *SovereigntyHub
		var vw *SovereigntyVulnerabilityWindow
		if a.SovereigntyHub.VulnerabilityWindow != nil {
			vw = &SovereigntyVulnerabilityWindow{Start: a.SovereigntyHub.VulnerabilityWindow.Start, End: a.SovereigntyHub.VulnerabilityWindow.End}
		}
		hub = &SovereigntyHub{ID: a.SovereigntyHub.Id, VulnerabilityWindow: vw}
		return SovereigntyClaim{
			Kind: "alliance", AllianceID: &a.AllianceId, CorporationID: &a.CorporationId,
			ClaimedSince: &a.ClaimedSince, IsCapitalSystem: &isCapital,
			Development: &SovereigntyDevelopment{
				ActivityDefenseMultiplier: a.Development.ActivityDefenseMultiplier,
				IndustrialLevel:           a.Development.IndustrialLevel,
				MilitaryLevel:             a.Development.MilitaryLevel,
				StrategicLevel:            a.Development.StrategicLevel,
			},
			SovereigntyHub: hub,
		}
	}
	if claim.Faction != nil && claim.Faction.Faction != nil {
		return SovereigntyClaim{Kind: "faction", FactionID: &claim.Faction.Faction.FactionId}
	}
	return SovereigntyClaim{Kind: "unclaimed"}
}

type SovereigntySystem struct {
	SolarSystemID int64            `json:"solarSystemId"`
	Claim         SovereigntyClaim `json:"claim"`
}

var listSovereigntySystems = func(ctx context.Context, userAgent string) ([]esiapi.SovereigntySystemsSolarsystem, error) {
	client := esi.NewPublicClient(userAgent)
	return client.SovereigntySystems(ctx)
}

type ListSovereigntySystemsInput struct{}
type ListSovereigntySystemsOutput struct {
	Systems []SovereigntySystem `json:"systems"`
}

func listSovereigntySystemsHandler(version string) mcpsdk.ToolHandlerFor[ListSovereigntySystemsInput, ListSovereigntySystemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListSovereigntySystemsInput) (*mcpsdk.CallToolResult, ListSovereigntySystemsOutput, error) {
		systems, err := listSovereigntySystems(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListSovereigntySystemsOutput{}, fmt.Errorf("list sovereignty systems: %w", err)
		}
		out := make([]SovereigntySystem, len(systems))
		for i, s := range systems {
			out[i] = SovereigntySystem{SolarSystemID: s.SolarSystemId, Claim: toSovereigntyClaim(s.Claim)}
		}
		return nil, ListSovereigntySystemsOutput{Systems: out}, nil
	}
}
```

All new types are flat, non-recursive — already sandbox-verified against a live `mcpsdk.AddTool`
registration test with zero schema-generator errors.

`mcp/sovereignty_internal_test.go`: 3 tests — `MapsFields` for campaigns, and `MapsFields` for
systems covering all 3 claim variants (alliance with full nested detail including a populated
vulnerability window, faction, unclaimed) in one fixture, following `mcp/insurance_internal_test.go`'s
public-tool pattern.

## `mcp/server.go` / `mcp/server_test.go`

Register 2 tools after whatever the last-registered tool is at execution time (expected to be one
of the Skills tools). Update package doc comment. `server_test.go`'s expected-tools list: append 2
names.

## Docs sync

- `CLAUDE.md` / `README.md`: increment tool count by 2 from whatever it currently reads, append
  `list_sovereignty_campaigns`, `list_sovereignty_systems`.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Sovereignty row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/sovereignty_test.go`,
swappable-`var`-override tests for `mcp/sovereignty_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.

Given this batch's size (2 public endpoints, no auth complexity), Task 1 and Task 2 are combined
into a single implementation task, following the precedent set by Incursions/Insurance/Routes/Search.
