# ESI Sovereignty Implementation Plan (thirtieth batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Sovereignty" tag (2 endpoints: active sovereignty
campaigns, and every solar system's current sovereignty claim) as 2 new MCP tools, folding into
the same shared PR #6 branch as every prior batch.

**Architecture:** Both endpoints are public with no auth complexity — combined into a single
implementation task, following the precedent set by Incursions/Insurance/Routes/Search.
`esi/sovereignty.go` wraps the 2 generated `SovereigntyAPIService` methods. `mcp/sovereignty.go`
exposes 2 public tools, flattening the nested oneOf sovereignty-claim union (alliance/faction/
unclaimed) into a single `Kind`-discriminated type, following the pattern established for Faction
Warfare's leaderboards and Corporation Projects' `Configuration`.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: whatever the actual tool count is once Skills lands (this plan
  assumes Skills' 29th-batch commit is already merged, bringing the count to 170). **If Skills has
  not yet landed when this plan is executed, adjust the baseline count and registration anchor
  accordingly** — use `grep -c 'mcpsdk.AddTool' mcp/server.go` for the real current count before
  writing any doc-count edits, and register after whatever the actual last tool is.
- Both `GetSovereigntyCampaigns` and `GetSovereigntySystems` have no security block in
  `esi/api/openapi.yaml` (genuinely public) and no `.Page()` setter (not paginated).
- **The nested oneOf claim union is verified NOT recursive.** `SovereigntySystemsSolarsystemClaim`
  has exactly 3 pointer fields (`Alliance`, `Faction`, `Unclaimed`), exactly one non-nil at a time.
  `Alliance`'s nested `Development`/`SovereigntyHub`/`VulnerabilityWindow` detail goes one level
  deeper but never loops back to `Claim`, `SolarSystem`, or itself — confirmed field-by-field
  across all 7 generated model files. Flatten this into a single `Kind`-discriminated
  `SovereigntyClaim` MCP type (`"alliance"`/`"faction"`/`"unclaimed"`), not an artificial
  self-referential shape.
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-sovereignty-design.md` for the verification detail.

---

### Task 1: `esi/sovereignty.go` + `mcp/sovereignty.go` — full batch in one task

**Files:**
- Create: `esi/sovereignty.go`
- Test: `esi/sovereignty_test.go`
- Create: `mcp/sovereignty.go`
- Test: `mcp/sovereignty_internal_test.go`
- Modify: `mcp/server.go` (register 2 tools after whatever the actual last-registered tool is at
  execution time; update package doc comment)
- Modify: `mcp/server_test.go` (append 2 tool names to expected-tools list)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Sovereignty row: `Planned` → `This plan`, only
  that row)
- Modify: `CLAUDE.md` (increment tool count by 2 from whatever it currently reads, append 2 tool names)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Produces: 2 tool names — `list_sovereignty_campaigns`, `list_sovereignty_systems` — registered
  in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `esi/sovereignty_test.go`:

```go
package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestSovereigntyCampaigns_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{
			"attackers_score": 0.5,
			"campaign_id": 12345,
			"constellation_id": 20000001,
			"defender_id": 99000001,
			"defender_score": 0.75,
			"event_type": "tcu_defense",
			"participants": [{"alliance_id": 99000002, "score": 0.25}],
			"solar_system_id": 30000001,
			"start_time": "2026-07-25T20:00:00Z",
			"structure_id": 1000000000001
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	campaigns, err := client.SovereigntyCampaigns(context.Background())
	if err != nil {
		t.Fatalf("SovereigntyCampaigns returned error: %v", err)
	}
	if len(campaigns) != 1 {
		t.Fatalf("campaigns = %+v, want one entry", campaigns)
	}
	c := campaigns[0]
	if c.CampaignId != 12345 || c.ConstellationId != 20000001 || c.SolarSystemId != 30000001 || c.StructureId != 1000000000001 || c.EventType != "tcu_defense" {
		t.Errorf("campaigns[0] = %+v, want CampaignId=12345 ConstellationId=20000001 SolarSystemId=30000001 StructureId=1000000000001 EventType=tcu_defense", c)
	}
	wantStart := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	if !c.StartTime.Equal(wantStart) {
		t.Errorf("StartTime = %v, want %v", c.StartTime, wantStart)
	}
	if c.AttackersScore == nil || *c.AttackersScore != 0.5 {
		t.Errorf("AttackersScore = %v, want 0.5", c.AttackersScore)
	}
	if c.DefenderId == nil || *c.DefenderId != 99000001 {
		t.Errorf("DefenderId = %v, want 99000001", c.DefenderId)
	}
	if c.DefenderScore == nil || *c.DefenderScore != 0.75 {
		t.Errorf("DefenderScore = %v, want 0.75", c.DefenderScore)
	}
	if len(c.Participants) != 1 || c.Participants[0].AllianceId != 99000002 || c.Participants[0].Score != 0.25 {
		t.Errorf("Participants = %+v, want one entry {AllianceId:99000002 Score:0.25}", c.Participants)
	}
}

func TestSovereigntyCampaigns_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.SovereigntyCampaigns(context.Background()); err == nil {
		t.Fatal("SovereigntyCampaigns returned nil error, want an error for HTTP 500")
	}
}

func TestSovereigntySystems_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"solar_systems":[
			{"solar_system_id":30000001,"claim":{"alliance":{"alliance_id":99000001,"claimed_since":"2026-01-01T00:00:00Z","corporation_id":98000001,"development":{"activity_defense_multiplier":1.0,"industrial_level":2,"military_level":3,"strategic_level":1},"is_capital_system":true,"sovereignty_hub":{"id":1234567890,"vulnerability_window":{"end":"2026-07-25T22:00:00Z","start":"2026-07-25T20:00:00Z"}}}}},
			{"solar_system_id":30000002,"claim":{"faction":{"faction_id":500001}}},
			{"solar_system_id":30000003,"claim":{"unclaimed":true}}
		]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	systems, err := client.SovereigntySystems(context.Background())
	if err != nil {
		t.Fatalf("SovereigntySystems returned error: %v", err)
	}
	if len(systems) != 3 {
		t.Fatalf("systems = %+v, want three entries", systems)
	}

	s0 := systems[0]
	if s0.SolarSystemId != 30000001 {
		t.Errorf("systems[0].SolarSystemId = %d, want 30000001", s0.SolarSystemId)
	}
	if s0.Claim.Alliance == nil || s0.Claim.Alliance.Alliance == nil {
		t.Fatal("systems[0].Claim.Alliance = nil, want non-nil")
	}
	a := s0.Claim.Alliance.Alliance
	if a.AllianceId != 99000001 || a.CorporationId != 98000001 || !a.IsCapitalSystem {
		t.Errorf("alliance claim = %+v, want AllianceId=99000001 CorporationId=98000001 IsCapitalSystem=true", a)
	}
	wantClaimedSince := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !a.ClaimedSince.Equal(wantClaimedSince) {
		t.Errorf("ClaimedSince = %v, want %v", a.ClaimedSince, wantClaimedSince)
	}
	if a.Development.ActivityDefenseMultiplier != 1.0 || a.Development.IndustrialLevel != 2 || a.Development.MilitaryLevel != 3 || a.Development.StrategicLevel != 1 {
		t.Errorf("Development = %+v, want ActivityDefenseMultiplier=1.0 IndustrialLevel=2 MilitaryLevel=3 StrategicLevel=1", a.Development)
	}
	if a.SovereigntyHub.Id != 1234567890 {
		t.Errorf("SovereigntyHub.Id = %d, want 1234567890", a.SovereigntyHub.Id)
	}
	if a.SovereigntyHub.VulnerabilityWindow == nil {
		t.Fatal("SovereigntyHub.VulnerabilityWindow = nil, want non-nil")
	}
	wantVwStart := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	wantVwEnd := time.Date(2026, 7, 25, 22, 0, 0, 0, time.UTC)
	if !a.SovereigntyHub.VulnerabilityWindow.Start.Equal(wantVwStart) || !a.SovereigntyHub.VulnerabilityWindow.End.Equal(wantVwEnd) {
		t.Errorf("VulnerabilityWindow = %+v, want Start=%v End=%v", a.SovereigntyHub.VulnerabilityWindow, wantVwStart, wantVwEnd)
	}

	s1 := systems[1]
	if s1.SolarSystemId != 30000002 || s1.Claim.Faction == nil || s1.Claim.Faction.Faction == nil || s1.Claim.Faction.Faction.FactionId != 500001 {
		t.Errorf("systems[1] = %+v, want SolarSystemId=30000002 Claim.Faction.Faction.FactionId=500001", s1)
	}

	s2 := systems[2]
	if s2.SolarSystemId != 30000003 || s2.Claim.Unclaimed == nil || s2.Claim.Unclaimed.Unclaimed == nil || !*s2.Claim.Unclaimed.Unclaimed {
		t.Errorf("systems[2] = %+v, want SolarSystemId=30000003 Claim.Unclaimed.Unclaimed=true", s2)
	}
}

func TestSovereigntySystems_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.SovereigntySystems(context.Background()); err == nil {
		t.Fatal("SovereigntySystems returned nil error, want an error for HTTP 500")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestSovereigntyCampaigns|TestSovereigntySystems' -v`
Expected: FAIL — `esi.Client` has neither method yet.

- [ ] **Step 3: Write the esi/ implementation**

Create `esi/sovereignty.go`:

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

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestSovereigntyCampaigns|TestSovereigntySystems' -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Write the failing mcp/ tests**

Create `mcp/sovereignty_internal_test.go`:

```go
package mcp

import (
	"context"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func TestListSovereigntyCampaignsHandler_MapsFields(t *testing.T) {
	orig := listSovereigntyCampaigns
	t.Cleanup(func() { listSovereigntyCampaigns = orig })
	attackersScore := 0.5
	defenderID := int64(99000001)
	defenderScore := 0.75
	startTime := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	listSovereigntyCampaigns = func(ctx context.Context, userAgent string) ([]esiapi.SovereigntyCampaignsGetInner, error) {
		return []esiapi.SovereigntyCampaignsGetInner{{
			CampaignId: 12345, ConstellationId: 20000001, SolarSystemId: 30000001, StructureId: 1000000000001,
			EventType: "tcu_defense", StartTime: startTime,
			AttackersScore: &attackersScore, DefenderId: &defenderID, DefenderScore: &defenderScore,
			Participants: []esiapi.SovereigntyCampaignsGetInnerParticipantsInner{{AllianceId: 99000002, Score: 0.25}},
		}}, nil
	}

	handler := listSovereigntyCampaignsHandler("v1")
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
	orig := listSovereigntySystems
	t.Cleanup(func() { listSovereigntySystems = orig })
	claimedSince := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	vwStart := time.Date(2026, 7, 25, 20, 0, 0, 0, time.UTC)
	vwEnd := time.Date(2026, 7, 25, 22, 0, 0, 0, time.UTC)
	unclaimed := true

	listSovereigntySystems = func(ctx context.Context, userAgent string) ([]esiapi.SovereigntySystemsSolarsystem, error) {
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
								Id: 1234567890,
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
	}

	handler := listSovereigntySystemsHandler("v1")
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
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListSovereigntyCampaignsHandler|TestListSovereigntySystemsHandler' -v`
Expected: FAIL — undefined symbols (none of these handlers/types exist yet).

- [ ] **Step 7: Write the mcp/ implementation**

Create `mcp/sovereignty.go`:

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

Modify `mcp/server.go`: append after the actual last-registered tool block at execution time
(immediately before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_sovereignty_campaigns",
		Description: "List every active sovereignty campaign in the game (public data, no character required).",
	}, listSovereigntyCampaignsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_sovereignty_systems",
		Description: "List every solar system and its current sovereignty claim, if any (public data, no character required).",
	}, listSovereigntySystemsHandler(version))
```

Update `mcp/server.go`'s package doc comment: append `list_sovereignty_campaigns` and
`list_sovereignty_systems` to the trailing tool list, preserving the "and X registered" grammar.

Modify `mcp/server_test.go`: append `"list_sovereignty_campaigns"`, `"list_sovereignty_systems"`
to the expected-tools slice, matching the file's existing membership-check style.

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Sovereignty row from `Planned` to
`This plan`. Do not touch any other row.

Modify `CLAUDE.md`: increment the tool count by 2 from whatever it currently reads (spell out the
new number in words, matching the file's existing style), and append the 2 new tool names to the
`mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestListSovereigntyCampaignsHandler|TestListSovereigntySystemsHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (2 new handler tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 9: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, no panics on tool registration.

```bash
git add esi/sovereignty.go esi/sovereignty_test.go mcp/sovereignty.go mcp/sovereignty_internal_test.go \
  mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add sovereignty MCP tools (2 tools: campaigns, system claims)"
```
