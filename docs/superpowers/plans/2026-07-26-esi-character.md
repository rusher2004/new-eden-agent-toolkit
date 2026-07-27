# ESI Character Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cover the Character ESI resource group (14 endpoints: 4 public arbitrary-character lookups + 10 private own-character endpoints).

**Architecture:** `esi/character.go` is a thin, `describeError`-wrapped facade. The 4 public methods use `esi.NewPublicClient` (arbitrary-ID lookups, no auth); the 10 private methods use the client's own bound `characterID` via the authenticated client, matching every prior private facade. `mcp/character.go` adds 14 tools; the 10 private ones introduce a small shared `characterExists(store, characterID) (bool, error)` helper (new — justified here since 14 tools would otherwise duplicate the existence-check loop 10 times, well past the DRY threshold that made sense for smaller batches).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context: `fmt.Errorf("operation: %w", err)`. Never discard an error.
- Every `esi/` method that calls `.Execute()` must route its error and `*http.Response` through `describeError` before wrapping.
- External test packages by default; `*_internal_test.go` only when needed for unexported symbols.
- Standard library only in tests.
- Raw-ID passthrough: no code resolves bloodline/race/faction/skill-type IDs to names.
- The 4 public tools (`get_character`, `get_character_affiliations`, `list_character_corporation_history`, `get_character_portrait`) take **no existence-check** — they look up arbitrary character IDs, not the caller's own data, matching Alliance's precedent.
- The 10 private tools use the new `characterExists` helper (defined once in `mcp/character.go`, used only by this file's handlers — not a retrofit of prior files' duplicated loops, which stays out of scope for this batch).
- This branch is the shared PR branch (`worktree-esi-access-list-activities`) — current baseline is 23 tools (verified by reading `mcp/server.go` before writing this plan).
- Design spec: `docs/superpowers/specs/2026-07-26-esi-character-design.md`.
- All code in this plan has been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and `github.com/modelcontextprotocol/go-sdk` v1.6.1 modules — a full compile of every `esi/character.go` method and every `mcp/character.go` handler against real `esiapi`/`mcpsdk` types.

---

### Task 1: `esi/character.go` — Character facade (public + private methods)

**Files:**
- Create: `esi/character.go`
- Create: `esi/character_test.go`

**Interfaces:**
- Consumes: `esi.Client`, `describeError`, `esi.NewPublicClient` (existing, from Alliance batch); `esiapi.APIClient.CharacterAPI.GetCharactersDetail(ctx, characterId int64) ...Execute() (*esiapi.CharactersDetail, *http.Response, error)`, `.PostCharactersAffiliation(ctx).RequestBody(characterIDs []int64) ...Execute() ([]esiapi.CharactersAffiliationPostInner, *http.Response, error)`, `.GetCharactersCharacterIdCorporationhistory(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, *http.Response, error)`, `.GetCharactersCharacterIdPortrait(ctx, characterId int64) ...Execute() (*esiapi.CharactersCharacterIdPortraitGet, *http.Response, error)`, `.GetCharactersCharacterIdAgentsResearch(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, *http.Response, error)`, `.GetCharactersCharacterIdBlueprints(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdBlueprintsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdFatigue(ctx, characterId int64) ...Execute() (*esiapi.CharactersCharacterIdFatigueGet, *http.Response, error)`, `.GetCharactersCharacterIdMedals(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdMedalsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdNotifications(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdNotificationsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdNotificationsContacts(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdRoles(ctx, characterId int64) ...Execute() (*esiapi.CharactersCharacterIdRolesGet, *http.Response, error)`, `.GetCharactersCharacterIdStandings(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdStandingsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdTitles(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdTitlesGetInner, *http.Response, error)`, `.PostCharactersCharacterIdCspa(ctx, characterId int64).RequestBody(targetIDs []int64) ...Execute() (float64, *http.Response, error)` — all verified against the real installed module.
- Produces: `(*esi.Client).CharacterDetail`, `.CharacterAffiliations`, `.CharacterCorporationHistory`, `.CharacterPortrait`, `.CharacterAgentsResearch`, `.CharacterBlueprints`, `.CharacterFatigue`, `.CharacterMedals`, `.CharacterNotifications`, `.CharacterNotificationContacts`, `.CharacterRoles`, `.CharacterStandings`, `.CharacterTitles`, `.CharacterCSPACost` — all consumed by Task 2.

- [ ] **Step 1: Write the failing tests**

Create `esi/character_test.go`:

```go
package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"achievement_score":100,"birthday":"2020-01-01T00:00:00Z","bloodline_id":1,"corporation_id":98000001,"gender":"male","name":"Test Character","race_id":1}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	detail, err := client.CharacterDetail(context.Background(), 998877)
	if err != nil {
		t.Fatalf("CharacterDetail returned error: %v", err)
	}
	if detail.Name != "Test Character" || detail.CorporationId != 98000001 {
		t.Errorf("detail = %+v, want Name=\"Test Character\" CorporationId=98000001", detail)
	}
	if detail.AllianceId != nil {
		t.Errorf("detail.AllianceId = %v, want nil", detail.AllianceId)
	}
}

func TestCharacterDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterDetail(context.Background(), 998877); err == nil {
		t.Fatal("CharacterDetail returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterAffiliations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"character_id":998877,"corporation_id":98000001}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	affs, err := client.CharacterAffiliations(context.Background(), []int64{998877})
	if err != nil {
		t.Fatalf("CharacterAffiliations returned error: %v", err)
	}
	if len(affs) != 1 || affs[0].CharacterId != 998877 || affs[0].CorporationId != 98000001 {
		t.Errorf("affs = %+v, want one entry {CharacterId:998877 CorporationId:98000001}", affs)
	}
}

func TestCharacterCorporationHistory_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"corporation_id":98000001,"record_id":1,"start_date":"2020-01-01T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	hist, err := client.CharacterCorporationHistory(context.Background(), 998877)
	if err != nil {
		t.Fatalf("CharacterCorporationHistory returned error: %v", err)
	}
	if len(hist) != 1 || hist[0].CorporationId != 98000001 {
		t.Errorf("hist = %+v, want one entry with CorporationId=98000001", hist)
	}
}

func TestCharacterPortrait_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"px64x64":"https://images.evetech.net/a.png"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	portrait, err := client.CharacterPortrait(context.Background(), 998877)
	if err != nil {
		t.Fatalf("CharacterPortrait returned error: %v", err)
	}
	if portrait.Px64x64 == nil || *portrait.Px64x64 != "https://images.evetech.net/a.png" {
		t.Errorf("portrait.Px64x64 = %v, want a pointer to the URL", portrait.Px64x64)
	}
}

func TestCharacterAgentsResearch_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"agent_id":1,"points_per_day":10.5,"remainder_points":2.5,"skill_type_id":11450,"started_at":"2020-01-01T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	agents, err := client.CharacterAgentsResearch(context.Background())
	if err != nil {
		t.Fatalf("CharacterAgentsResearch returned error: %v", err)
	}
	if len(agents) != 1 || agents[0].AgentId != 1 {
		t.Errorf("agents = %+v, want one entry with AgentId=1", agents)
	}
}

func TestCharacterBlueprints_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":1,"location_flag":"Hangar","location_id":60003760,"material_efficiency":10,"quantity":1,"runs":-1,"time_efficiency":20,"type_id":950}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	bps, err := client.CharacterBlueprints(context.Background())
	if err != nil {
		t.Fatalf("CharacterBlueprints returned error: %v", err)
	}
	if len(bps) != 1 || bps[0].TypeId != 950 {
		t.Errorf("bps = %+v, want one entry with TypeId=950", bps)
	}
}

func TestCharacterFatigue_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	fatigue, err := client.CharacterFatigue(context.Background())
	if err != nil {
		t.Fatalf("CharacterFatigue returned error: %v", err)
	}
	if fatigue.LastJumpDate != nil {
		t.Errorf("fatigue.LastJumpDate = %v, want nil", fatigue.LastJumpDate)
	}
}

func TestCharacterMedals_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"corporation_id":98000001,"date":"2020-01-01T00:00:00Z","description":"desc","graphics":[{"graphic":"g1","layer":0,"part":1}],"issuer_id":998877,"medal_id":1,"reason":"reason","status":"private","title":"Title"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	medals, err := client.CharacterMedals(context.Background())
	if err != nil {
		t.Fatalf("CharacterMedals returned error: %v", err)
	}
	if len(medals) != 1 || medals[0].Title != "Title" || len(medals[0].Graphics) != 1 {
		t.Errorf("medals = %+v, want one entry with Title=\"Title\" and one graphic", medals)
	}
}

func TestCharacterNotifications_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"notification_id":1,"sender_id":998877,"sender_type":"character","timestamp":"2020-01-01T00:00:00Z","type":"CorpAllBillMsg"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	notifs, err := client.CharacterNotifications(context.Background())
	if err != nil {
		t.Fatalf("CharacterNotifications returned error: %v", err)
	}
	if len(notifs) != 1 || notifs[0].NotificationId != 1 {
		t.Errorf("notifs = %+v, want one entry with NotificationId=1", notifs)
	}
}

func TestCharacterNotificationContacts_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"message":"msg","notification_id":1,"send_date":"2020-01-01T00:00:00Z","sender_character_id":998877,"standing_level":5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	notifs, err := client.CharacterNotificationContacts(context.Background())
	if err != nil {
		t.Fatalf("CharacterNotificationContacts returned error: %v", err)
	}
	if len(notifs) != 1 || notifs[0].StandingLevel != 5 {
		t.Errorf("notifs = %+v, want one entry with StandingLevel=5", notifs)
	}
}

func TestCharacterRoles_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"roles":["Director"]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	roles, err := client.CharacterRoles(context.Background())
	if err != nil {
		t.Fatalf("CharacterRoles returned error: %v", err)
	}
	if len(roles.Roles) != 1 || roles.Roles[0] != "Director" {
		t.Errorf("roles.Roles = %v, want [\"Director\"]", roles.Roles)
	}
}

func TestCharacterStandings_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"from_id":500001,"from_type":"npc_corp","standing":5.5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	standings, err := client.CharacterStandings(context.Background())
	if err != nil {
		t.Fatalf("CharacterStandings returned error: %v", err)
	}
	if len(standings) != 1 || standings[0].Standing != 5.5 {
		t.Errorf("standings = %+v, want one entry with Standing=5.5", standings)
	}
}

func TestCharacterTitles_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"CEO","title_id":1}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	titles, err := client.CharacterTitles(context.Background())
	if err != nil {
		t.Fatalf("CharacterTitles returned error: %v", err)
	}
	if len(titles) != 1 || titles[0].Name == nil || *titles[0].Name != "CEO" {
		t.Errorf("titles = %+v, want one entry with Name=*\"CEO\"", titles)
	}
}

func TestCharacterCSPACost_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`1000000.0`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	cost, err := client.CharacterCSPACost(context.Background(), []int64{998878})
	if err != nil {
		t.Fatalf("CharacterCSPACost returned error: %v", err)
	}
	if cost != 1000000.0 {
		t.Errorf("cost = %v, want 1000000.0", cost)
	}
}

func TestCharacterCSPACost_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterCSPACost(context.Background(), []int64{998878}); err == nil {
		t.Fatal("CharacterCSPACost returned nil error, want an error for HTTP 403")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacter' -v`

Expected: FAIL to compile — all `(*esi.Client).Character*` methods are undefined.

- [ ] **Step 3: Implement `esi/character.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterDetail returns public info about an arbitrary character.
func (c *Client) CharacterDetail(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersDetail(ctx, characterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterAffiliations resolves a set of character IDs to their corporation/alliance/faction.
func (c *Client) CharacterAffiliations(ctx context.Context, characterIDs []int64) ([]esiapi.CharactersAffiliationPostInner, error) {
	result, resp, err := c.api.CharacterAPI.PostCharactersAffiliation(ctx).RequestBody(characterIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character affiliations: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterCorporationHistory returns an arbitrary character's corporation history.
func (c *Client) CharacterCorporationHistory(ctx context.Context, characterID int64) ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdCorporationhistory(ctx, characterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character corporation history: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterPortrait returns an arbitrary character's portrait URLs.
func (c *Client) CharacterPortrait(ctx context.Context, characterID int64) (*esiapi.CharactersCharacterIdPortraitGet, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdPortrait(ctx, characterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character portrait: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterAgentsResearch returns the client's character's agent research progress.
func (c *Client) CharacterAgentsResearch(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdAgentsResearch(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character agents research: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterBlueprints returns the client's character's blueprints.
func (c *Client) CharacterBlueprints(ctx context.Context) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdBlueprints(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character blueprints: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterFatigue returns the client's character's jump fatigue state.
func (c *Client) CharacterFatigue(ctx context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdFatigue(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character fatigue: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMedals returns the client's character's medals.
func (c *Client) CharacterMedals(ctx context.Context) ([]esiapi.CharactersCharacterIdMedalsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdMedals(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character medals: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterNotifications returns the client's character's notifications.
func (c *Client) CharacterNotifications(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdNotifications(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character notifications: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterNotificationContacts returns the client's character's contact-standing notifications.
func (c *Client) CharacterNotificationContacts(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdNotificationsContacts(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character notification contacts: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterRoles returns the client's character's corporation roles.
func (c *Client) CharacterRoles(ctx context.Context) (*esiapi.CharactersCharacterIdRolesGet, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdRoles(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character roles: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterStandings returns the client's character's standings.
func (c *Client) CharacterStandings(ctx context.Context) ([]esiapi.CharactersCharacterIdStandingsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdStandings(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character standings: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterTitles returns the client's character's corporation titles.
func (c *Client) CharacterTitles(ctx context.Context) ([]esiapi.CharactersCharacterIdTitlesGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdTitles(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character titles: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterCSPACost calculates the CSPA charge cost, from the client's character, to contact
// targetCharacterIDs.
func (c *Client) CharacterCSPACost(ctx context.Context, targetCharacterIDs []int64) (float64, error) {
	result, resp, err := c.api.CharacterAPI.PostCharactersCharacterIdCspa(ctx, int64(c.characterID)).RequestBody(targetCharacterIDs).Execute()
	if err != nil {
		return 0, fmt.Errorf("get character cspa cost: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all 16 new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/character.go esi/character_test.go
git commit -m "feat: add esi/character.go facade"
```

---

### Task 2: `mcp/character.go` tools + registration + `server_test.go` update

**Files:**
- Create: `mcp/character.go`
- Create: `mcp/character_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/server_test.go`

**Interfaces:**
- Consumes: `esi.NewClient`, `esi.NewPublicClient` (existing), all 14 `(*esi.Client).Character*` methods (Task 1); `sso.UserAgent`, `config.Store.Load`, `ErrCharacterNotFound` (existing).
- Produces: 14 new tool Input/Output types (see below), plus a new unexported helper `characterExists(store *config.Store, characterID int32) (bool, error)`. Nothing later in this plan depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/character_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetCharacterHandler_ReturnsDetail(t *testing.T) {
	orig := getCharacterDetail
	t.Cleanup(func() { getCharacterDetail = orig })
	getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
		return GetCharacterOutput{CharacterID: characterID, Name: "Test Character", CorporationID: 98000001}, nil
	}

	handler := getCharacterHandler("v1")
	_, out, err := handler(context.Background(), nil, GetCharacterInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 998877 || out.Name != "Test Character" {
		t.Errorf("out = %+v, want CharacterID=998877 Name=\"Test Character\"", out)
	}
}

func TestGetCharacterAffiliationsHandler_ReturnsAffiliations(t *testing.T) {
	orig := getCharacterAffiliations
	t.Cleanup(func() { getCharacterAffiliations = orig })
	getCharacterAffiliations = func(ctx context.Context, userAgent string, characterIDs []int64) ([]CharacterAffiliation, error) {
		return []CharacterAffiliation{{CharacterID: 998877, CorporationID: 98000001}}, nil
	}

	handler := getCharacterAffiliationsHandler("v1")
	_, out, err := handler(context.Background(), nil, GetCharacterAffiliationsInput{CharacterIDs: []int64{998877}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Affiliations) != 1 || out.Affiliations[0].CharacterID != 998877 {
		t.Errorf("out = %+v, want one affiliation with CharacterID=998877", out)
	}
}

func TestListCharacterCorporationHistoryHandler_ReturnsHistory(t *testing.T) {
	orig := listCharacterCorporationHistory
	t.Cleanup(func() { listCharacterCorporationHistory = orig })
	listCharacterCorporationHistory = func(ctx context.Context, userAgent string, characterID int64) ([]CorporationHistoryEntry, error) {
		return []CorporationHistoryEntry{{CorporationID: 98000001, RecordID: 1}}, nil
	}

	handler := listCharacterCorporationHistoryHandler("v1")
	_, out, err := handler(context.Background(), nil, ListCharacterCorporationHistoryInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.History) != 1 || out.History[0].CorporationID != 98000001 {
		t.Errorf("out = %+v, want one history entry with CorporationID=98000001", out)
	}
}

func TestGetCharacterPortraitHandler_ReturnsPortrait(t *testing.T) {
	orig := getCharacterPortrait
	t.Cleanup(func() { getCharacterPortrait = orig })
	url := "https://images.evetech.net/a.png"
	getCharacterPortrait = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterPortraitOutput, error) {
		return GetCharacterPortraitOutput{CharacterID: characterID, Px64x64: &url}, nil
	}

	handler := getCharacterPortraitHandler("v1")
	_, out, err := handler(context.Background(), nil, GetCharacterPortraitInput{CharacterID: 998877})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Px64x64 == nil || *out.Px64x64 != url {
		t.Errorf("out = %+v, want Px64x64=*%q", out, url)
	}
}

func TestListCharacterAgentsResearchHandler_ReturnsAgents(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterAgentsResearch
	t.Cleanup(func() { listCharacterAgentsResearch = orig })
	listCharacterAgentsResearch = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AgentResearch, error) {
		return []AgentResearch{{AgentID: 1}}, nil
	}

	handler := listCharacterAgentsResearchHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterAgentsResearchInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Agents) != 1 || out.Agents[0].AgentID != 1 {
		t.Errorf("out = %+v, want one agent with AgentID=1", out)
	}
}

func TestListCharacterAgentsResearchHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterAgentsResearch
	t.Cleanup(func() { listCharacterAgentsResearch = orig })
	listCharacterAgentsResearch = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AgentResearch, error) {
		t.Fatal("listCharacterAgentsResearch should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterAgentsResearchHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterAgentsResearchInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterBlueprintsHandler_ReturnsBlueprints(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterBlueprints
	t.Cleanup(func() { listCharacterBlueprints = orig })
	listCharacterBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Blueprint, error) {
		return []Blueprint{{ItemID: 1, TypeID: 950}}, nil
	}

	handler := listCharacterBlueprintsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Blueprints) != 1 || out.Blueprints[0].TypeID != 950 {
		t.Errorf("out = %+v, want one blueprint with TypeID=950", out)
	}
}

func TestListCharacterBlueprintsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterBlueprints
	t.Cleanup(func() { listCharacterBlueprints = orig })
	listCharacterBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Blueprint, error) {
		t.Fatal("listCharacterBlueprints should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterBlueprintsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterBlueprintsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterFatigueHandler_ReturnsFatigue(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterFatigue
	t.Cleanup(func() { getCharacterFatigue = orig })
	getCharacterFatigue = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFatigueOutput, error) {
		return GetCharacterFatigueOutput{CharacterID: characterID}, nil
	}

	handler := getCharacterFatigueHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterFatigueInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 {
		t.Errorf("out = %+v, want CharacterID=7", out)
	}
}

func TestGetCharacterFatigueHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterFatigue
	t.Cleanup(func() { getCharacterFatigue = orig })
	getCharacterFatigue = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFatigueOutput, error) {
		t.Fatal("getCharacterFatigue should not be called for an unknown character")
		return GetCharacterFatigueOutput{}, nil
	}

	handler := getCharacterFatigueHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterFatigueInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterMedalsHandler_ReturnsMedals(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterMedals
	t.Cleanup(func() { listCharacterMedals = orig })
	listCharacterMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Medal, error) {
		return []Medal{{MedalID: 1, Title: "Title"}}, nil
	}

	handler := listCharacterMedalsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterMedalsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Medals) != 1 || out.Medals[0].Title != "Title" {
		t.Errorf("out = %+v, want one medal with Title=\"Title\"", out)
	}
}

func TestListCharacterMedalsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterMedals
	t.Cleanup(func() { listCharacterMedals = orig })
	listCharacterMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Medal, error) {
		t.Fatal("listCharacterMedals should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMedalsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterMedalsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterNotificationsHandler_ReturnsNotifications(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterNotifications
	t.Cleanup(func() { listCharacterNotifications = orig })
	listCharacterNotifications = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Notification, error) {
		return []Notification{{NotificationID: 1}}, nil
	}

	handler := listCharacterNotificationsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterNotificationsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Notifications) != 1 || out.Notifications[0].NotificationID != 1 {
		t.Errorf("out = %+v, want one notification with NotificationID=1", out)
	}
}

func TestListCharacterNotificationsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterNotifications
	t.Cleanup(func() { listCharacterNotifications = orig })
	listCharacterNotifications = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Notification, error) {
		t.Fatal("listCharacterNotifications should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterNotificationsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterNotificationsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterNotificationContactsHandler_ReturnsNotifications(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterNotificationContacts
	t.Cleanup(func() { listCharacterNotificationContacts = orig })
	listCharacterNotificationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactNotification, error) {
		return []ContactNotification{{NotificationID: 1, StandingLevel: 5}}, nil
	}

	handler := listCharacterNotificationContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterNotificationContactsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Notifications) != 1 || out.Notifications[0].StandingLevel != 5 {
		t.Errorf("out = %+v, want one notification with StandingLevel=5", out)
	}
}

func TestListCharacterNotificationContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterNotificationContacts
	t.Cleanup(func() { listCharacterNotificationContacts = orig })
	listCharacterNotificationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactNotification, error) {
		t.Fatal("listCharacterNotificationContacts should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterNotificationContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterNotificationContactsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterRolesHandler_ReturnsRoles(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterRoles
	t.Cleanup(func() { getCharacterRoles = orig })
	getCharacterRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterRolesOutput, error) {
		return GetCharacterRolesOutput{CharacterID: characterID, Roles: []string{"Director"}}, nil
	}

	handler := getCharacterRolesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterRolesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Roles) != 1 || out.Roles[0] != "Director" {
		t.Errorf("out = %+v, want Roles=[\"Director\"]", out)
	}
}

func TestGetCharacterRolesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterRoles
	t.Cleanup(func() { getCharacterRoles = orig })
	getCharacterRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterRolesOutput, error) {
		t.Fatal("getCharacterRoles should not be called for an unknown character")
		return GetCharacterRolesOutput{}, nil
	}

	handler := getCharacterRolesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterRolesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterStandingsHandler_ReturnsStandings(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterStandings
	t.Cleanup(func() { listCharacterStandings = orig })
	listCharacterStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Standing, error) {
		return []Standing{{FromID: 500001, Standing: 5.5}}, nil
	}

	handler := listCharacterStandingsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterStandingsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Standings) != 1 || out.Standings[0].Standing != 5.5 {
		t.Errorf("out = %+v, want one standing with Standing=5.5", out)
	}
}

func TestListCharacterStandingsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterStandings
	t.Cleanup(func() { listCharacterStandings = orig })
	listCharacterStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Standing, error) {
		t.Fatal("listCharacterStandings should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterStandingsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterStandingsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterTitlesHandler_ReturnsTitles(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterTitles
	t.Cleanup(func() { listCharacterTitles = orig })
	name := "CEO"
	listCharacterTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Title, error) {
		return []Title{{Name: &name}}, nil
	}

	handler := listCharacterTitlesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterTitlesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Titles) != 1 || out.Titles[0].Name == nil || *out.Titles[0].Name != "CEO" {
		t.Errorf("out = %+v, want one title with Name=*\"CEO\"", out)
	}
}

func TestListCharacterTitlesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterTitles
	t.Cleanup(func() { listCharacterTitles = orig })
	listCharacterTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Title, error) {
		t.Fatal("listCharacterTitles should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterTitlesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterTitlesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCharacterCSPACostHandler_ReturnsCost(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterCSPACost
	t.Cleanup(func() { getCharacterCSPACost = orig })
	getCharacterCSPACost = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetCharacterIDs []int64) (float64, error) {
		return 1000000.0, nil
	}

	handler := getCharacterCSPACostHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterCSPACostInput{CharacterID: 7, TargetCharacterIDs: []int64{998878}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Cost != 1000000.0 {
		t.Errorf("out = %+v, want Cost=1000000.0", out)
	}
}

func TestGetCharacterCSPACostHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterCSPACost
	t.Cleanup(func() { getCharacterCSPACost = orig })
	getCharacterCSPACost = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetCharacterIDs []int64) (float64, error) {
		t.Fatal("getCharacterCSPACost should not be called for an unknown character")
		return 0, nil
	}

	handler := getCharacterCSPACostHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterCSPACostInput{CharacterID: 999, TargetCharacterIDs: []int64{998878}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestGetCharacterHandler|TestGetCharacterAffiliationsHandler|TestListCharacterCorporationHistoryHandler|TestGetCharacterPortraitHandler|TestListCharacterAgentsResearchHandler|TestListCharacterBlueprintsHandler|TestGetCharacterFatigueHandler|TestListCharacterMedalsHandler|TestListCharacterNotificationsHandler|TestListCharacterNotificationContactsHandler|TestGetCharacterRolesHandler|TestListCharacterStandingsHandler|TestListCharacterTitlesHandler|TestGetCharacterCSPACostHandler' -v`

Expected: FAIL to compile — all the new `mcp` symbols are undefined.

- [ ] **Step 3: Implement `mcp/character.go`**

The complete file (14 tools, the shared `characterExists` helper, all Input/Output types) is exactly
the code verified to compile against real `esiapi`/`mcpsdk` types during plan-writing:

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- Public: get_character ---

type GetCharacterOutput struct {
	CharacterID      int64     `json:"characterId"`
	Name             string    `json:"name"`
	CorporationID    int64     `json:"corporationId"`
	AllianceID       *int64    `json:"allianceId,omitempty"`
	FactionID        *int64    `json:"factionId,omitempty"`
	Birthday         time.Time `json:"birthday"`
	BloodlineID      int64     `json:"bloodlineId"`
	RaceID           int64     `json:"raceId"`
	Gender           string    `json:"gender"`
	SecurityStatus   *float64  `json:"securityStatus,omitempty"`
	Description      *string   `json:"description,omitempty"`
	Title            *string   `json:"title,omitempty"`
	AchievementScore int64     `json:"achievementScore"`
}

var getCharacterDetail = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterOutput, error) {
	client := esi.NewPublicClient(userAgent)
	detail, err := client.CharacterDetail(ctx, characterID)
	if err != nil {
		return GetCharacterOutput{}, err
	}
	return GetCharacterOutput{
		CharacterID: characterID, Name: detail.Name, CorporationID: detail.CorporationId,
		AllianceID: detail.AllianceId, FactionID: detail.FactionId, Birthday: detail.Birthday,
		BloodlineID: detail.BloodlineId, RaceID: detail.RaceId, Gender: detail.Gender,
		SecurityStatus: detail.SecurityStatus, Description: detail.Description,
		Title: detail.CorporationTitle, AchievementScore: detail.AchievementScore,
	}, nil
}

type GetCharacterInput struct {
	CharacterID int64 `json:"characterId" jsonschema:"the character ID to look up"`
}

func getCharacterHandler(version string) mcpsdk.ToolHandlerFor[GetCharacterInput, GetCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterInput) (*mcpsdk.CallToolResult, GetCharacterOutput, error) {
		out, err := getCharacterDetail(ctx, sso.UserAgent(version), in.CharacterID)
		if err != nil {
			return nil, GetCharacterOutput{}, fmt.Errorf("get character: %w", err)
		}
		return nil, out, nil
	}
}

// --- Public: get_character_affiliations ---

type CharacterAffiliation struct {
	CharacterID   int64  `json:"characterId"`
	CorporationID int64  `json:"corporationId"`
	AllianceID    *int64 `json:"allianceId,omitempty"`
	FactionID     *int64 `json:"factionId,omitempty"`
}

var getCharacterAffiliations = func(ctx context.Context, userAgent string, characterIDs []int64) ([]CharacterAffiliation, error) {
	client := esi.NewPublicClient(userAgent)
	affs, err := client.CharacterAffiliations(ctx, characterIDs)
	if err != nil {
		return nil, err
	}
	out := make([]CharacterAffiliation, len(affs))
	for i, a := range affs {
		out[i] = CharacterAffiliation{CharacterID: a.CharacterId, CorporationID: a.CorporationId, AllianceID: a.AllianceId, FactionID: a.FactionId}
	}
	return out, nil
}

type GetCharacterAffiliationsInput struct {
	CharacterIDs []int64 `json:"characterIds" jsonschema:"character IDs to resolve"`
}
type GetCharacterAffiliationsOutput struct {
	Affiliations []CharacterAffiliation `json:"affiliations"`
}

func getCharacterAffiliationsHandler(version string) mcpsdk.ToolHandlerFor[GetCharacterAffiliationsInput, GetCharacterAffiliationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterAffiliationsInput) (*mcpsdk.CallToolResult, GetCharacterAffiliationsOutput, error) {
		affs, err := getCharacterAffiliations(ctx, sso.UserAgent(version), in.CharacterIDs)
		if err != nil {
			return nil, GetCharacterAffiliationsOutput{}, fmt.Errorf("get character affiliations: %w", err)
		}
		return nil, GetCharacterAffiliationsOutput{Affiliations: affs}, nil
	}
}

// --- Public: list_character_corporation_history ---

type CorporationHistoryEntry struct {
	CorporationID int64     `json:"corporationId"`
	RecordID      int64     `json:"recordId"`
	StartDate     time.Time `json:"startDate"`
	IsDeleted     *bool     `json:"isDeleted,omitempty"`
}

var listCharacterCorporationHistory = func(ctx context.Context, userAgent string, characterID int64) ([]CorporationHistoryEntry, error) {
	client := esi.NewPublicClient(userAgent)
	hist, err := client.CharacterCorporationHistory(ctx, characterID)
	if err != nil {
		return nil, err
	}
	out := make([]CorporationHistoryEntry, len(hist))
	for i, h := range hist {
		out[i] = CorporationHistoryEntry{CorporationID: h.CorporationId, RecordID: h.RecordId, StartDate: h.StartDate, IsDeleted: h.IsDeleted}
	}
	return out, nil
}

type ListCharacterCorporationHistoryInput struct {
	CharacterID int64 `json:"characterId" jsonschema:"the character ID to look up"`
}
type ListCharacterCorporationHistoryOutput struct {
	CharacterID int64                     `json:"characterId"`
	History     []CorporationHistoryEntry `json:"history"`
}

func listCharacterCorporationHistoryHandler(version string) mcpsdk.ToolHandlerFor[ListCharacterCorporationHistoryInput, ListCharacterCorporationHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterCorporationHistoryInput) (*mcpsdk.CallToolResult, ListCharacterCorporationHistoryOutput, error) {
		hist, err := listCharacterCorporationHistory(ctx, sso.UserAgent(version), in.CharacterID)
		if err != nil {
			return nil, ListCharacterCorporationHistoryOutput{}, fmt.Errorf("list character corporation history: %w", err)
		}
		return nil, ListCharacterCorporationHistoryOutput{CharacterID: in.CharacterID, History: hist}, nil
	}
}

// --- Public: get_character_portrait ---

type GetCharacterPortraitOutput struct {
	CharacterID int64   `json:"characterId"`
	Px64x64     *string `json:"px64x64,omitempty"`
	Px128x128   *string `json:"px128x128,omitempty"`
	Px256x256   *string `json:"px256x256,omitempty"`
	Px512x512   *string `json:"px512x512,omitempty"`
}

var getCharacterPortrait = func(ctx context.Context, userAgent string, characterID int64) (GetCharacterPortraitOutput, error) {
	client := esi.NewPublicClient(userAgent)
	portrait, err := client.CharacterPortrait(ctx, characterID)
	if err != nil {
		return GetCharacterPortraitOutput{}, err
	}
	return GetCharacterPortraitOutput{
		CharacterID: characterID, Px64x64: portrait.Px64x64, Px128x128: portrait.Px128x128,
		Px256x256: portrait.Px256x256, Px512x512: portrait.Px512x512,
	}, nil
}

type GetCharacterPortraitInput struct {
	CharacterID int64 `json:"characterId" jsonschema:"the character ID to look up"`
}

func getCharacterPortraitHandler(version string) mcpsdk.ToolHandlerFor[GetCharacterPortraitInput, GetCharacterPortraitOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterPortraitInput) (*mcpsdk.CallToolResult, GetCharacterPortraitOutput, error) {
		out, err := getCharacterPortrait(ctx, sso.UserAgent(version), in.CharacterID)
		if err != nil {
			return nil, GetCharacterPortraitOutput{}, fmt.Errorf("get character portrait: %w", err)
		}
		return nil, out, nil
	}
}

// --- shared existence-check helper for the private tools below ---

// characterExists reports whether characterID is a logged-in character in store. New in this
// file: with 10 private tools sharing the same check, duplicating the loop 10 times crosses the
// DRY threshold that made sense for smaller batches.
func characterExists(store *config.Store, characterID int32) (bool, error) {
	cfg, err := store.Load()
	if err != nil {
		return false, fmt.Errorf("load config: %w", err)
	}
	for _, c := range cfg.Characters {
		if c.ID == characterID {
			return true, nil
		}
	}
	return false, nil
}

// --- Private: list_character_agents_research ---

type AgentResearch struct {
	AgentID         int64     `json:"agentId"`
	SkillTypeID     int64     `json:"skillTypeId"`
	PointsPerDay    float64   `json:"pointsPerDay"`
	RemainderPoints float64   `json:"remainderPoints"`
	StartedAt       time.Time `json:"startedAt"`
}

var listCharacterAgentsResearch = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]AgentResearch, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	agents, err := client.CharacterAgentsResearch(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AgentResearch, len(agents))
	for i, a := range agents {
		out[i] = AgentResearch{AgentID: a.AgentId, SkillTypeID: a.SkillTypeId, PointsPerDay: a.PointsPerDay, RemainderPoints: a.RemainderPoints, StartedAt: a.StartedAt}
	}
	return out, nil
}

type ListCharacterAgentsResearchInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterAgentsResearchOutput struct {
	CharacterID int32           `json:"characterId"`
	Agents      []AgentResearch `json:"agents"`
}

func listCharacterAgentsResearchHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterAgentsResearchInput, ListCharacterAgentsResearchOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterAgentsResearchInput) (*mcpsdk.CallToolResult, ListCharacterAgentsResearchOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterAgentsResearchOutput{}, err
		}
		if !found {
			return nil, ListCharacterAgentsResearchOutput{}, fmt.Errorf("list character agents research for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		agents, err := listCharacterAgentsResearch(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterAgentsResearchOutput{}, fmt.Errorf("list character agents research: %w", err)
		}
		return nil, ListCharacterAgentsResearchOutput{CharacterID: in.CharacterID, Agents: agents}, nil
	}
}

// --- Private: list_character_blueprints ---

type Blueprint struct {
	ItemID             int64  `json:"itemId"`
	TypeID             int64  `json:"typeId"`
	LocationID         int64  `json:"locationId"`
	LocationFlag       string `json:"locationFlag"`
	Quantity           int64  `json:"quantity"`
	Runs               int64  `json:"runs"`
	MaterialEfficiency int64  `json:"materialEfficiency"`
	TimeEfficiency     int64  `json:"timeEfficiency"`
}

var listCharacterBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Blueprint, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	bps, err := client.CharacterBlueprints(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Blueprint, len(bps))
	for i, b := range bps {
		out[i] = Blueprint{ItemID: b.ItemId, TypeID: b.TypeId, LocationID: b.LocationId, LocationFlag: b.LocationFlag, Quantity: b.Quantity, Runs: b.Runs, MaterialEfficiency: b.MaterialEfficiency, TimeEfficiency: b.TimeEfficiency}
	}
	return out, nil
}

type ListCharacterBlueprintsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterBlueprintsOutput struct {
	CharacterID int32       `json:"characterId"`
	Blueprints  []Blueprint `json:"blueprints"`
}

func listCharacterBlueprintsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterBlueprintsInput, ListCharacterBlueprintsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterBlueprintsInput) (*mcpsdk.CallToolResult, ListCharacterBlueprintsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterBlueprintsOutput{}, err
		}
		if !found {
			return nil, ListCharacterBlueprintsOutput{}, fmt.Errorf("list character blueprints for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		bps, err := listCharacterBlueprints(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterBlueprintsOutput{}, fmt.Errorf("list character blueprints: %w", err)
		}
		return nil, ListCharacterBlueprintsOutput{CharacterID: in.CharacterID, Blueprints: bps}, nil
	}
}

// --- Private: get_character_fatigue ---

type GetCharacterFatigueOutput struct {
	CharacterID           int32      `json:"characterId"`
	JumpFatigueExpireDate *time.Time `json:"jumpFatigueExpireDate,omitempty"`
	LastJumpDate          *time.Time `json:"lastJumpDate,omitempty"`
	LastUpdateDate        *time.Time `json:"lastUpdateDate,omitempty"`
}

var getCharacterFatigue = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterFatigueOutput, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return GetCharacterFatigueOutput{}, fmt.Errorf("build ESI client: %w", err)
	}
	fatigue, err := client.CharacterFatigue(ctx)
	if err != nil {
		return GetCharacterFatigueOutput{}, err
	}
	return GetCharacterFatigueOutput{CharacterID: characterID, JumpFatigueExpireDate: fatigue.JumpFatigueExpireDate, LastJumpDate: fatigue.LastJumpDate, LastUpdateDate: fatigue.LastUpdateDate}, nil
}

type GetCharacterFatigueInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

func getCharacterFatigueHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterFatigueInput, GetCharacterFatigueOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterFatigueInput) (*mcpsdk.CallToolResult, GetCharacterFatigueOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterFatigueOutput{}, err
		}
		if !found {
			return nil, GetCharacterFatigueOutput{}, fmt.Errorf("get character fatigue for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		out, err := getCharacterFatigue(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterFatigueOutput{}, fmt.Errorf("get character fatigue: %w", err)
		}
		return nil, out, nil
	}
}

// --- Private: list_character_medals ---

type MedalGraphic struct {
	Graphic string `json:"graphic"`
	Layer   int64  `json:"layer"`
	Part    int64  `json:"part"`
	Color   *int64 `json:"color,omitempty"`
}
type Medal struct {
	MedalID       int64          `json:"medalId"`
	CorporationID int64          `json:"corporationId"`
	IssuerID      int64          `json:"issuerId"`
	Date          time.Time      `json:"date"`
	Description   string         `json:"description"`
	Reason        string         `json:"reason"`
	Status        string         `json:"status"`
	Title         string         `json:"title"`
	Graphics      []MedalGraphic `json:"graphics"`
}

var listCharacterMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Medal, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	medals, err := client.CharacterMedals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Medal, len(medals))
	for i, m := range medals {
		graphics := make([]MedalGraphic, len(m.Graphics))
		for j, g := range m.Graphics {
			graphics[j] = MedalGraphic{Graphic: g.Graphic, Layer: g.Layer, Part: g.Part, Color: g.Color}
		}
		out[i] = Medal{MedalID: m.MedalId, CorporationID: m.CorporationId, IssuerID: m.IssuerId, Date: m.Date, Description: m.Description, Reason: m.Reason, Status: m.Status, Title: m.Title, Graphics: graphics}
	}
	return out, nil
}

type ListCharacterMedalsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterMedalsOutput struct {
	CharacterID int32   `json:"characterId"`
	Medals      []Medal `json:"medals"`
}

func listCharacterMedalsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterMedalsInput, ListCharacterMedalsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMedalsInput) (*mcpsdk.CallToolResult, ListCharacterMedalsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMedalsOutput{}, err
		}
		if !found {
			return nil, ListCharacterMedalsOutput{}, fmt.Errorf("list character medals for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		medals, err := listCharacterMedals(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterMedalsOutput{}, fmt.Errorf("list character medals: %w", err)
		}
		return nil, ListCharacterMedalsOutput{CharacterID: in.CharacterID, Medals: medals}, nil
	}
}

// --- Private: list_character_notifications ---

type Notification struct {
	NotificationID int64     `json:"notificationId"`
	SenderID       int64     `json:"senderId"`
	SenderType     string    `json:"senderType"`
	Type           string    `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	IsRead         *bool     `json:"isRead,omitempty"`
	Text           *string   `json:"text,omitempty"`
}

var listCharacterNotifications = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Notification, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	notifs, err := client.CharacterNotifications(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Notification, len(notifs))
	for i, n := range notifs {
		out[i] = Notification{NotificationID: n.NotificationId, SenderID: n.SenderId, SenderType: n.SenderType, Type: n.Type, Timestamp: n.Timestamp, IsRead: n.IsRead, Text: n.Text}
	}
	return out, nil
}

type ListCharacterNotificationsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterNotificationsOutput struct {
	CharacterID   int32          `json:"characterId"`
	Notifications []Notification `json:"notifications"`
}

func listCharacterNotificationsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterNotificationsInput, ListCharacterNotificationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterNotificationsInput) (*mcpsdk.CallToolResult, ListCharacterNotificationsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterNotificationsOutput{}, err
		}
		if !found {
			return nil, ListCharacterNotificationsOutput{}, fmt.Errorf("list character notifications for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		notifs, err := listCharacterNotifications(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterNotificationsOutput{}, fmt.Errorf("list character notifications: %w", err)
		}
		return nil, ListCharacterNotificationsOutput{CharacterID: in.CharacterID, Notifications: notifs}, nil
	}
}

// --- Private: list_character_notification_contacts ---

type ContactNotification struct {
	NotificationID    int64     `json:"notificationId"`
	SenderCharacterID int64     `json:"senderCharacterId"`
	SendDate          time.Time `json:"sendDate"`
	StandingLevel     float64   `json:"standingLevel"`
	Message           string    `json:"message"`
}

var listCharacterNotificationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactNotification, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	notifs, err := client.CharacterNotificationContacts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ContactNotification, len(notifs))
	for i, n := range notifs {
		out[i] = ContactNotification{NotificationID: n.NotificationId, SenderCharacterID: n.SenderCharacterId, SendDate: n.SendDate, StandingLevel: n.StandingLevel, Message: n.Message}
	}
	return out, nil
}

type ListCharacterNotificationContactsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterNotificationContactsOutput struct {
	CharacterID   int32                  `json:"characterId"`
	Notifications []ContactNotification `json:"notifications"`
}

func listCharacterNotificationContactsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterNotificationContactsInput, ListCharacterNotificationContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterNotificationContactsInput) (*mcpsdk.CallToolResult, ListCharacterNotificationContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterNotificationContactsOutput{}, err
		}
		if !found {
			return nil, ListCharacterNotificationContactsOutput{}, fmt.Errorf("list character notification contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		notifs, err := listCharacterNotificationContacts(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterNotificationContactsOutput{}, fmt.Errorf("list character notification contacts: %w", err)
		}
		return nil, ListCharacterNotificationContactsOutput{CharacterID: in.CharacterID, Notifications: notifs}, nil
	}
}

// --- Private: get_character_roles ---

type GetCharacterRolesOutput struct {
	CharacterID  int32    `json:"characterId"`
	Roles        []string `json:"roles,omitempty"`
	RolesAtBase  []string `json:"rolesAtBase,omitempty"`
	RolesAtHq    []string `json:"rolesAtHq,omitempty"`
	RolesAtOther []string `json:"rolesAtOther,omitempty"`
}

var getCharacterRoles = func(ctx context.Context, clientID string, characterID int32, userAgent string) (GetCharacterRolesOutput, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return GetCharacterRolesOutput{}, fmt.Errorf("build ESI client: %w", err)
	}
	roles, err := client.CharacterRoles(ctx)
	if err != nil {
		return GetCharacterRolesOutput{}, err
	}
	return GetCharacterRolesOutput{CharacterID: characterID, Roles: roles.Roles, RolesAtBase: roles.RolesAtBase, RolesAtHq: roles.RolesAtHq, RolesAtOther: roles.RolesAtOther}, nil
}

type GetCharacterRolesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

func getCharacterRolesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterRolesInput, GetCharacterRolesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterRolesInput) (*mcpsdk.CallToolResult, GetCharacterRolesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterRolesOutput{}, err
		}
		if !found {
			return nil, GetCharacterRolesOutput{}, fmt.Errorf("get character roles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		out, err := getCharacterRoles(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterRolesOutput{}, fmt.Errorf("get character roles: %w", err)
		}
		return nil, out, nil
	}
}

// --- Private: list_character_standings ---

type Standing struct {
	FromID   int64   `json:"fromId"`
	FromType string  `json:"fromType"`
	Standing float64 `json:"standing"`
}

var listCharacterStandings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Standing, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	standings, err := client.CharacterStandings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Standing, len(standings))
	for i, s := range standings {
		out[i] = Standing{FromID: s.FromId, FromType: s.FromType, Standing: s.Standing}
	}
	return out, nil
}

type ListCharacterStandingsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterStandingsOutput struct {
	CharacterID int32      `json:"characterId"`
	Standings   []Standing `json:"standings"`
}

func listCharacterStandingsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterStandingsInput, ListCharacterStandingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterStandingsInput) (*mcpsdk.CallToolResult, ListCharacterStandingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterStandingsOutput{}, err
		}
		if !found {
			return nil, ListCharacterStandingsOutput{}, fmt.Errorf("list character standings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		standings, err := listCharacterStandings(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterStandingsOutput{}, fmt.Errorf("list character standings: %w", err)
		}
		return nil, ListCharacterStandingsOutput{CharacterID: in.CharacterID, Standings: standings}, nil
	}
}

// --- Private: list_character_titles ---

type Title struct {
	TitleID *int64  `json:"titleId,omitempty"`
	Name    *string `json:"name,omitempty"`
}

var listCharacterTitles = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Title, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	titles, err := client.CharacterTitles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Title, len(titles))
	for i, t := range titles {
		out[i] = Title{TitleID: t.TitleId, Name: t.Name}
	}
	return out, nil
}

type ListCharacterTitlesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterTitlesOutput struct {
	CharacterID int32   `json:"characterId"`
	Titles      []Title `json:"titles"`
}

func listCharacterTitlesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterTitlesInput, ListCharacterTitlesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterTitlesInput) (*mcpsdk.CallToolResult, ListCharacterTitlesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterTitlesOutput{}, err
		}
		if !found {
			return nil, ListCharacterTitlesOutput{}, fmt.Errorf("list character titles for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		titles, err := listCharacterTitles(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterTitlesOutput{}, fmt.Errorf("list character titles: %w", err)
		}
		return nil, ListCharacterTitlesOutput{CharacterID: in.CharacterID, Titles: titles}, nil
	}
}

// --- Private: get_character_cspa_cost ---

var getCharacterCSPACost = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetCharacterIDs []int64) (float64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterCSPACost(ctx, targetCharacterIDs)
}

type GetCharacterCSPACostInput struct {
	CharacterID        int32   `json:"characterId" jsonschema:"the character ID to check"`
	TargetCharacterIDs []int64 `json:"targetCharacterIds" jsonschema:"target character IDs to calculate CSPA cost for"`
}
type GetCharacterCSPACostOutput struct {
	CharacterID int32   `json:"characterId"`
	Cost        float64 `json:"cost"`
}

func getCharacterCSPACostHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterCSPACostInput, GetCharacterCSPACostOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterCSPACostInput) (*mcpsdk.CallToolResult, GetCharacterCSPACostOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterCSPACostOutput{}, err
		}
		if !found {
			return nil, GetCharacterCSPACostOutput{}, fmt.Errorf("get character cspa cost for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		cost, err := getCharacterCSPACost(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.TargetCharacterIDs)
		if err != nil {
			return nil, GetCharacterCSPACostOutput{}, fmt.Errorf("get character cspa cost: %w", err)
		}
		return nil, GetCharacterCSPACostOutput{CharacterID: in.CharacterID, Cost: cost}, nil
	}
}
```

- [ ] **Step 4: Register the fourteen tools in `mcp/server.go`**

Add 14 new `mcpsdk.AddTool` calls right after `respond_calendar_event`'s (the last tool from the
prior batch), updating the `NewServer` doc comment to list all 37 tools. Use these exact `Name`/
`Description` values and handler constructors, in this order:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character",
		Description: "Get public info (name, corporation, bloodline, security status) about an arbitrary character (public data, no character required).",
	}, getCharacterHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_affiliations",
		Description: "Resolve a set of character IDs to their current corporation/alliance/faction (public data, no character required).",
	}, getCharacterAffiliationsHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_corporation_history",
		Description: "List an arbitrary character's corporation history (public data, no character required).",
	}, listCharacterCorporationHistoryHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_portrait",
		Description: "Get an arbitrary character's portrait URLs (public data, no character required).",
	}, getCharacterPortraitHandler(version))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_agents_research",
		Description: "List a logged-in EVE Online character's agent research progress.",
	}, listCharacterAgentsResearchHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_blueprints",
		Description: "List a logged-in EVE Online character's blueprints.",
	}, listCharacterBlueprintsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_fatigue",
		Description: "Get a logged-in EVE Online character's jump fatigue state.",
	}, getCharacterFatigueHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_medals",
		Description: "List a logged-in EVE Online character's medals.",
	}, listCharacterMedalsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_notifications",
		Description: "List a logged-in EVE Online character's notifications.",
	}, listCharacterNotificationsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_notification_contacts",
		Description: "List a logged-in EVE Online character's contact-standing notifications.",
	}, listCharacterNotificationContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_roles",
		Description: "Get a logged-in EVE Online character's corporation roles.",
	}, getCharacterRolesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_standings",
		Description: "List a logged-in EVE Online character's standings.",
	}, listCharacterStandingsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_titles",
		Description: "List a logged-in EVE Online character's corporation titles.",
	}, listCharacterTitlesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_cspa_cost",
		Description: "Calculate the CSPA charge cost for a logged-in character to contact a set of target characters.",
	}, getCharacterCSPACostHandler(version, clientID, store))
```

- [ ] **Step 5: Update `mcp/server_test.go`'s expected-tools list**

Append these 14 names to `TestServer_AllToolsRegistered`'s expected-tools slice:

```go
		"get_character", "get_character_affiliations", "list_character_corporation_history", "get_character_portrait",
		"list_character_agents_research", "list_character_blueprints", "get_character_fatigue",
		"list_character_medals", "list_character_notifications", "list_character_notification_contacts",
		"get_character_roles", "list_character_standings", "list_character_titles", "get_character_cspa_cost",
```

- [ ] **Step 6: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race -v`

Expected: every new test from Step 1 PASSes, `TestServer_AllToolsRegistered` PASSes with all 37 tool
names, `gofmt -l .` prints nothing, and every pre-existing test still passes.

- [ ] **Step 7: Commit**

```bash
git add mcp/character.go mcp/character_internal_test.go mcp/server.go mcp/server_test.go
git commit -m "feat: add character MCP tools (14 tools: 4 public lookups + 10 private)"
```

---

### Task 3: Docs sync (`CLAUDE.md`, `README.md`, roadmap doc)

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Modify: `docs/reference/esi-coverage-roadmap.md`

**Interfaces:** none — documentation only.

- [ ] **Step 1: Update `CLAUDE.md`'s package table**

Bump the `mcp/` row's tool count from "twenty-three" to "thirty-seven" and list all 37 tool names
(the 23 existing plus the 14 new ones from Task 2, in registration order).

- [ ] **Step 2: Update `README.md`**

Update the `## Status` section's ESI-tool-coverage sentence to add "and Character" to the list of
covered tags, and update the package table's `mcp/` row the same way as `CLAUDE.md`'s Step 1.

- [ ] **Step 3: Update the roadmap doc**

In `docs/reference/esi-coverage-roadmap.md`, change the Character row from "Planned" to "This plan".

- [ ] **Step 4: Sanity-check the docs against the code**

Run: `grep -n "get_character_cspa_cost\|thirty-seven" CLAUDE.md README.md`

Expected: both files mention the new tool count and at least one new tool name.

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "docs: document character tools, update roadmap and tool count"
```
