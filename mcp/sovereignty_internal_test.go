package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func TestListSovereigntyCampaignsHandler_MapsFields(t *testing.T) {
	attackersScore := 0.5
	defenderID := int64(99000001)
	defenderScore := 0.75
	startTime := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	fake := fakeSovereigntyCampaignsESI{sovereigntyCampaigns: func(context.Context) ([]esiapi.SovereigntyCampaignsGetInner, error) {
		return []esiapi.SovereigntyCampaignsGetInner{{
			CampaignId: 12345, ConstellationId: 20000001, SolarSystemId: 30000001, StructureId: 1000000000001,
			EventType: "tcu_defense", StartTime: startTime,
			AttackersScore: &attackersScore, DefenderId: &defenderID, DefenderScore: &defenderScore,
			Participants: []esiapi.SovereigntyCampaignsGetInnerParticipantsInner{{AllianceId: 99000002, Score: 0.25}},
		}}, nil
	}}

	handler := listSovereigntyCampaignsHandler("v1", publicNarrow[sovereigntyCampaignsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListSovereigntyCampaignsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Campaigns) != 1 {
		t.Fatalf("out.Campaigns = %+v, want one entry", out.Campaigns)
	}
	c := out.Campaigns[0]
	if c.CampaignID != 12345 || c.ConstellationID != 20000001 || c.SolarSystemID != 30000001 || c.StructureID != 1000000000001 || c.EventType != "tcu_defense" {
		t.Errorf("campaign = %+v, want CampaignID=12345 ConstellationID=20000001 SolarSystemID=30000001 StructureID=1000000000001 EventType=tcu_defense", c)
	}
	if !c.StartTime.Equal(startTime) {
		t.Errorf("StartTime = %v, want %v", c.StartTime, startTime)
	}
	if c.AttackersScore == nil || *c.AttackersScore != 0.5 {
		t.Errorf("AttackersScore = %v, want 0.5", c.AttackersScore)
	}
	if c.DefenderID == nil || *c.DefenderID != 99000001 {
		t.Errorf("DefenderID = %v, want 99000001", c.DefenderID)
	}
	if c.DefenderScore == nil || *c.DefenderScore != 0.75 {
		t.Errorf("DefenderScore = %v, want 0.75", c.DefenderScore)
	}
	if len(c.Participants) != 1 || c.Participants[0].AllianceID != 99000002 || c.Participants[0].Score != 0.25 {
		t.Errorf("Participants = %+v, want one entry {AllianceID:99000002 Score:0.25}", c.Participants)
	}
}

func TestListSovereigntySystemsHandler_MapsFieldsForAllThreeClaimKinds(t *testing.T) {
	claimedSince := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	vwStart := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	vwEnd := time.Date(2026, 7, 25, 22, 0, 0, 0, time.UTC)
	unclaimed := true

	fake := fakeSovereigntySystemsESI{sovereigntySystems: func(context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error) {
		return []esiapi.SovereigntySystemsSolarsystem{
			{
				SolarSystemId: 30000001,
				Claim: esiapi.SovereigntySystemsSolarsystemClaim{
					Alliance: &esiapi.Alliance{
						Alliance: &esiapi.SovereigntySystemsAlliance{
							AllianceId: 99000001, CorporationId: 98000001, ClaimedSince: claimedSince, IsCapitalSystem: true,
							Development: esiapi.SovereigntySystemsDevelopment{
								ActivityDefenseMultiplier: 1.0, IndustrialLevel: 2, MilitaryLevel: 3, StrategicLevel: 1,
							},
							SovereigntyHub: esiapi.SovereigntySystemsSovereigntyhub{
								Id:                  1234567890,
								VulnerabilityWindow: &esiapi.SovereigntySystemsVulnerabilitywindow{Start: vwStart, End: vwEnd},
							},
						},
					},
				},
			},
			{
				SolarSystemId: 30000002,
				Claim: esiapi.SovereigntySystemsSolarsystemClaim{
					Faction: &esiapi.Faction{Faction: &esiapi.SovereigntySystemsFaction{FactionId: 500001}},
				},
			},
			{
				SolarSystemId: 30000003,
				Claim: esiapi.SovereigntySystemsSolarsystemClaim{
					Unclaimed: &esiapi.Unclaimed{Unclaimed: &unclaimed},
				},
			},
		}, nil
	}}

	handler := listSovereigntySystemsHandler("v1", publicNarrow[sovereigntySystemsESI](fake), nil)

	_, out, err := handler(context.Background(), nil, ListSovereigntySystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Systems) != 3 {
		t.Fatalf("out.Systems = %+v, want three entries", out.Systems)
	}

	s0 := out.Systems[0]
	if s0.SolarSystemID != 30000001 || s0.Claim.Kind != "alliance" {
		t.Fatalf("systems[0] = %+v, want SolarSystemID=30000001 Claim.Kind=alliance", s0)
	}
	if s0.Claim.AllianceID == nil || *s0.Claim.AllianceID != 99000001 {
		t.Errorf("AllianceID = %v, want 99000001", s0.Claim.AllianceID)
	}
	if s0.Claim.CorporationID == nil || *s0.Claim.CorporationID != 98000001 {
		t.Errorf("CorporationID = %v, want 98000001", s0.Claim.CorporationID)
	}
	if s0.Claim.ClaimedSince == nil || !s0.Claim.ClaimedSince.Equal(claimedSince) {
		t.Errorf("ClaimedSince = %v, want %v", s0.Claim.ClaimedSince, claimedSince)
	}
	if s0.Claim.IsCapitalSystem == nil || !*s0.Claim.IsCapitalSystem {
		t.Errorf("IsCapitalSystem = %v, want true", s0.Claim.IsCapitalSystem)
	}
	if s0.Claim.Development == nil || s0.Claim.Development.ActivityDefenseMultiplier != 1.0 || s0.Claim.Development.IndustrialLevel != 2 || s0.Claim.Development.MilitaryLevel != 3 || s0.Claim.Development.StrategicLevel != 1 {
		t.Errorf("Development = %+v, want ActivityDefenseMultiplier=1.0 IndustrialLevel=2 MilitaryLevel=3 StrategicLevel=1", s0.Claim.Development)
	}
	if s0.Claim.SovereigntyHub == nil || s0.Claim.SovereigntyHub.ID != 1234567890 {
		t.Fatalf("SovereigntyHub = %+v, want ID=1234567890", s0.Claim.SovereigntyHub)
	}
	if s0.Claim.SovereigntyHub.VulnerabilityWindow == nil || !s0.Claim.SovereigntyHub.VulnerabilityWindow.Start.Equal(vwStart) || !s0.Claim.SovereigntyHub.VulnerabilityWindow.End.Equal(vwEnd) {
		t.Errorf("VulnerabilityWindow = %+v, want Start=%v End=%v", s0.Claim.SovereigntyHub.VulnerabilityWindow, vwStart, vwEnd)
	}
	if s0.Claim.FactionID != nil {
		t.Errorf("FactionID = %v, want nil for an alliance claim", s0.Claim.FactionID)
	}

	s1 := out.Systems[1]
	if s1.SolarSystemID != 30000002 || s1.Claim.Kind != "faction" {
		t.Fatalf("systems[1] = %+v, want SolarSystemID=30000002 Claim.Kind=faction", s1)
	}
	if s1.Claim.FactionID == nil || *s1.Claim.FactionID != 500001 {
		t.Errorf("FactionID = %v, want 500001", s1.Claim.FactionID)
	}
	if s1.Claim.AllianceID != nil || s1.Claim.Development != nil || s1.Claim.SovereigntyHub != nil {
		t.Errorf("systems[1].Claim = %+v, want no alliance-only fields set", s1.Claim)
	}

	s2 := out.Systems[2]
	if s2.SolarSystemID != 30000003 || s2.Claim.Kind != "unclaimed" {
		t.Fatalf("systems[2] = %+v, want SolarSystemID=30000003 Claim.Kind=unclaimed", s2)
	}
	if s2.Claim.AllianceID != nil || s2.Claim.FactionID != nil {
		t.Errorf("systems[2].Claim = %+v, want no allianceId/factionId set", s2.Claim)
	}
}

// TestToSovereigntyClaim_UnrecognizedVariantIsNotReportedAsUnclaimed locks in the fix for a real
// bug: toSovereigntyClaim used to fall through to Kind: "unclaimed" whenever none of Alliance/
// Faction/Unclaimed matched, silently conflating a genuinely unrecognized future ESI variant with
// a real unclaimed system. It must now report a distinct "unknown" Kind instead.
func TestToSovereigntyClaim_UnrecognizedVariantIsNotReportedAsUnclaimed(t *testing.T) {
	got := toSovereigntyClaim(esiapi.SovereigntySystemsSolarsystemClaim{})
	if got.Kind != "unknown" {
		t.Errorf("Kind = %q, want unknown for a claim with no recognized oneOf variant set", got.Kind)
	}
}

func TestListSovereigntyCampaignsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeSovereigntyCampaignsESI{sovereigntyCampaigns: func(context.Context) ([]esiapi.SovereigntyCampaignsGetInner, error) {
		return nil, wantErr
	}}

	handler := listSovereigntyCampaignsHandler("v1", publicNarrow[sovereigntyCampaignsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListSovereigntyCampaignsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListSovereigntySystemsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeSovereigntySystemsESI{sovereigntySystems: func(_ context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error) {
		return nil, wantErr
	}}

	handler := listSovereigntySystemsHandler("v1", publicNarrow[sovereigntySystemsESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListSovereigntySystemsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListSovereigntyCampaignsHandler_ResolvesLocationNames(t *testing.T) {
	fake := fakeSovereigntyCampaignsESI{sovereigntyCampaigns: func(context.Context) ([]esiapi.SovereigntyCampaignsGetInner, error) {
		return []esiapi.SovereigntyCampaignsGetInner{{
			CampaignId: 1, ConstellationId: 20000020, SolarSystemId: 30000142,
		}}, nil
	}}

	origConstellation := resolveConstellationNames
	t.Cleanup(func() { resolveConstellationNames = origConstellation })
	resolveConstellationNames = func(ctx context.Context, sdeStore constellationNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{20000020: "Kimotoro"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listSovereigntyCampaignsHandler("v1", publicNarrow[sovereigntyCampaignsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListSovereigntyCampaignsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Campaigns[0].ConstellationName != "Kimotoro" {
		t.Errorf("ConstellationName = %q, want %q", out.Campaigns[0].ConstellationName, "Kimotoro")
	}
	if out.Campaigns[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Campaigns[0].SolarSystemName, "Jita")
	}
}

func TestListSovereigntySystemsHandler_ResolvesSystemName(t *testing.T) {
	fake := fakeSovereigntySystemsESI{sovereigntySystems: func(_ context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error) {
		return []esiapi.SovereigntySystemsSolarsystem{{SolarSystemId: 30000142}}, nil
	}}

	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listSovereigntySystemsHandler("v1", publicNarrow[sovereigntySystemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListSovereigntySystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Systems[0].SolarSystemName != "Jita" {
		t.Errorf("SolarSystemName = %q, want %q", out.Systems[0].SolarSystemName, "Jita")
	}
}

func TestListSovereigntySystemsHandler_ResolvesClaimFactionName(t *testing.T) {
	factionID := int64(500001)
	fake := fakeSovereigntySystemsESI{sovereigntySystems: func(_ context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error) {
		return []esiapi.SovereigntySystemsSolarsystem{
			{SolarSystemId: 30000142, Claim: esiapi.SovereigntySystemsSolarsystemClaim{
				Faction: &esiapi.Faction{Faction: &esiapi.SovereigntySystemsFaction{FactionId: factionID}},
			}},
		}, nil
	}}

	origFaction := resolveFactionNames
	t.Cleanup(func() { resolveFactionNames = origFaction })
	resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{500001: "Sansha's Nation"}
	}
	origSystem := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origSystem })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listSovereigntySystemsHandler("v1", publicNarrow[sovereigntySystemsESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListSovereigntySystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Systems) != 1 || out.Systems[0].Claim.FactionName != "Sansha's Nation" {
		t.Errorf("Systems = %+v, want one entry Claim.FactionName=%q", out.Systems, "Sansha's Nation")
	}
}

// fakeSovereigntySystemsESI fakes sovereigntySystemsESI; a nil sovereigntySystems fails as an unfaked call.
type fakeSovereigntySystemsESI struct {
	sovereigntySystems func(ctx context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error)
}

func (f fakeSovereigntySystemsESI) SovereigntySystems(ctx context.Context) ([]esiapi.SovereigntySystemsSolarsystem, error) {
	if f.sovereigntySystems == nil {
		return nil, errNotFaked
	}
	return f.sovereigntySystems(ctx)
}

// fakeSovereigntyCampaignsESI fakes sovereigntyCampaignsESI; a nil sovereigntyCampaigns fails as an unfaked call.
type fakeSovereigntyCampaignsESI struct {
	sovereigntyCampaigns func(ctx context.Context) ([]esiapi.SovereigntyCampaignsGetInner, error)
}

func (f fakeSovereigntyCampaignsESI) SovereigntyCampaigns(ctx context.Context) ([]esiapi.SovereigntyCampaignsGetInner, error) {
	if f.sovereigntyCampaigns == nil {
		return nil, errNotFaked
	}
	return f.sovereigntyCampaigns(ctx)
}
