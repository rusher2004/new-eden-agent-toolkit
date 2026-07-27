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
	if claim.Unclaimed != nil {
		return SovereigntyClaim{Kind: "unclaimed"}
	}
	// None of the 3 known oneOf variants matched -- an unrecognized ESI variant, not a genuine
	// unclaimed system. Surface this distinctly rather than silently reporting "unclaimed".
	return SovereigntyClaim{Kind: "unknown"}
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
