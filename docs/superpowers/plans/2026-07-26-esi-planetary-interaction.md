# ESI Planetary Interaction Implementation Plan (twenty-sixth batch)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full coverage for ESI's "Planetary Interaction" tag (4 endpoints: a character's
colonized planets + one colony's full layout, a corporation's customs offices, and a public PI
schematic lookup) as 4 new MCP tools, folding into the same shared PR #6 branch as every prior
batch.

**Architecture:** Same two-layer split as every prior batch. `esi/planetary_interaction.go` wraps
the 4 generated `PlanetaryInteractionAPIService` methods: 2 private "my own character", 1
corp-scoped (character authenticates, corporation ID is the resource, ESI's own 403 is the sole
gate), 1 public. `mcp/planetary_interaction.go` exposes 4 tools on top of that, modeling the
colony-layout endpoint's Links/Pins/Routes as 3 flat sibling slices with one further optional
nesting level inside Pin — mirroring ESI's own non-recursive shape exactly.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Baseline before this plan: 161 MCP tools, branch `worktree-esi-access-list-activities`,
  PR #6 (open). This plan brings the count to 165.
- `GetCharactersCharacterIdPlanets` and `GetCharactersCharacterIdPlanetsPlanetId` both require OAuth2
  scope `esi-planets.manage_planets.v1` (private, "my own character") — verified in
  `esi/api/openapi.yaml`. `GetCorporationsCorporationIdCustomsOffices` requires
  `esi-planets.read_customs_offices.v1` plus the Director role (corp-scoped, ESI's own 403 gate, no
  additional local check). `GetUniverseSchematicsSchematicId` has no security block at all
  (genuinely public).
- **The colony-layout model is verified NOT recursive.** `CharactersCharacterIdPlanetsPlanetIdGet`
  has exactly 3 flat sibling slice fields (`Links`, `Pins`, `Routes`); `Links`/`Routes` reference
  pins purely by `int64` ID, never by embedding another `Link`/`Route`/`Pin` value; `Pin`'s one
  level of optional nested detail (`Contents`/`ExtractorDetails`/`FactoryDetails`) and
  `ExtractorDetails`'s one further level (`Heads`) do not loop back to `Pin`, `Link`, or `Route`.
  Model this as 3 flat sibling MCP types (`PlanetLink`, `PlanetPin`, `PlanetRoute`) plus the two
  further non-recursive nesting levels inside `PlanetPin` — do NOT introduce an artificial
  self-referential type here; none is needed.
- Pagination is verified **per-endpoint**: only `CorporationCustomsOffices` is paginated (has
  `.Page()`); the other 3 methods have no `.Page()` setter at all.
- Every `esi/` method's `.Execute()` call MUST route through the existing `describeError(err, resp)`
  helper (`esi/error.go`) before wrapping with `fmt.Errorf` — reused, never redeclared.
- Reuse `characterExists(store *config.Store, characterID int32) (bool, error)`,
  `ErrCharacterNotFound` (`mcp/character.go`), and `totalPagesFromHeader(resp *http.Response) int32`
  (`esi/contracts.go`) — do not redeclare.
- This plan's code has already been verified to compile clean (`go build`/`go vet`/`gofmt -l`)
  against the real installed dependencies in an isolated sandbox, AND (for the `mcp/` layer) to
  pass a live `mcpsdk.AddTool`+`Connect` registration test with zero output-schema errors — see
  `docs/superpowers/specs/2026-07-26-esi-planetary-interaction-design.md` for the verification
  detail.

---

### Task 1: `esi/planetary_interaction.go` — ESI facade layer

**Files:**
- Create: `esi/planetary_interaction.go`
- Test: `esi/planetary_interaction_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`);
  `totalPagesFromHeader(resp *http.Response) int32` (`esi/contracts.go`);
  `c.api.PlanetaryInteractionAPI` and `c.characterID` (`esi/client.go`'s `Client`, unchanged).
- Produces (for Task 2):
  `func (c *Client) CharacterPlanets(ctx context.Context) ([]esiapi.CharactersCharacterIdPlanetsGetInner, error)`;
  `func (c *Client) CharacterPlanetDetail(ctx context.Context, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error)`;
  `func (c *Client) CorporationCustomsOffices(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error)`;
  `func (c *Client) PlanetarySchematic(ctx context.Context, schematicID int64) (*esiapi.UniverseSchematicsSchematicIdGet, error)`.

- [ ] **Step 1: Write the failing tests**

Create `esi/planetary_interaction_test.go`:

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

func TestCharacterPlanets_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"last_update":"2026-07-01T00:00:00Z","num_pins":5,"owner_id":90000001,"planet_id":40000001,"planet_type":"plasma","solar_system_id":30000001,"upgrade_level":3}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	planets, err := client.CharacterPlanets(context.Background())
	if err != nil {
		t.Fatalf("CharacterPlanets returned error: %v", err)
	}
	if len(planets) != 1 {
		t.Fatalf("planets = %+v, want one entry", planets)
	}
	p := planets[0]
	if p.NumPins != 5 || p.OwnerId != 90000001 || p.PlanetId != 40000001 || p.PlanetType != "plasma" || p.SolarSystemId != 30000001 || p.UpgradeLevel != 3 {
		t.Errorf("planets[0] = %+v, want NumPins=5 OwnerId=90000001 PlanetId=40000001 PlanetType=plasma SolarSystemId=30000001 UpgradeLevel=3", p)
	}
}

func TestCharacterPlanets_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterPlanets(context.Background()); err == nil {
		t.Fatal("CharacterPlanets returned nil error, want an error for HTTP 404")
	}
}

// TestCharacterPlanetDetail_ParsesResponse covers every nested level of the colony-layout model:
// top-level Links/Pins/Routes, plus a Pin carrying Contents, ExtractorDetails (with a nested
// Heads entry), FactoryDetails, and a Route with Waypoints -- the full depth of what ESI actually
// returns, per the real generated model files.
func TestCharacterPlanetDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"links": [{"destination_pin_id":1002,"link_level":1,"source_pin_id":1001}],
			"pins": [
				{
					"pin_id":1001,
					"type_id":2254,
					"latitude":0.1,
					"longitude":0.2,
					"schematic_id":100,
					"expiry_time":"2026-07-02T00:00:00Z",
					"install_time":"2026-07-01T00:00:00Z",
					"last_cycle_start":"2026-07-01T12:00:00Z",
					"contents":[{"amount":500,"type_id":2073}],
					"extractor_details":{
						"cycle_time":1800,
						"head_radius":200.5,
						"product_type_id":2073,
						"qty_per_cycle":100,
						"heads":[{"head_id":0,"latitude":0.3,"longitude":0.4}]
					},
					"factory_details":{"schematic_id":100}
				},
				{"pin_id":1002,"type_id":2255,"latitude":0.5,"longitude":0.6}
			],
			"routes": [{"content_type_id":2073,"destination_pin_id":1002,"quantity":20,"route_id":5001,"source_pin_id":1001,"waypoints":[1003,1004]}]
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CharacterPlanetDetail(context.Background(), 40000001)
	if err != nil {
		t.Fatalf("CharacterPlanetDetail returned error: %v", err)
	}

	if len(detail.Links) != 1 {
		t.Fatalf("Links = %+v, want one entry", detail.Links)
	}
	l := detail.Links[0]
	if l.DestinationPinId != 1002 || l.LinkLevel != 1 || l.SourcePinId != 1001 {
		t.Errorf("Links[0] = %+v, want DestinationPinId=1002 LinkLevel=1 SourcePinId=1001", l)
	}

	if len(detail.Pins) != 2 {
		t.Fatalf("Pins = %+v, want two entries", detail.Pins)
	}
	p0 := detail.Pins[0]
	if p0.PinId != 1001 || p0.TypeId != 2254 || p0.Latitude != 0.1 || p0.Longitude != 0.2 {
		t.Errorf("Pins[0] = %+v, want PinId=1001 TypeId=2254 Latitude=0.1 Longitude=0.2", p0)
	}
	if p0.SchematicId == nil || *p0.SchematicId != 100 {
		t.Errorf("Pins[0].SchematicId = %v, want *100", p0.SchematicId)
	}
	if p0.ExpiryTime == nil || p0.InstallTime == nil || p0.LastCycleStart == nil {
		t.Errorf("Pins[0] timestamps = %+v, want all three non-nil", p0)
	}
	if len(p0.Contents) != 1 || p0.Contents[0].Amount != 500 || p0.Contents[0].TypeId != 2073 {
		t.Errorf("Pins[0].Contents = %+v, want one entry {Amount:500 TypeId:2073}", p0.Contents)
	}
	if p0.ExtractorDetails == nil {
		t.Fatal("Pins[0].ExtractorDetails = nil, want non-nil")
	}
	ed := p0.ExtractorDetails
	if ed.CycleTime == nil || *ed.CycleTime != 1800 {
		t.Errorf("ExtractorDetails.CycleTime = %v, want *1800", ed.CycleTime)
	}
	if ed.HeadRadius == nil || *ed.HeadRadius != 200.5 {
		t.Errorf("ExtractorDetails.HeadRadius = %v, want *200.5", ed.HeadRadius)
	}
	if ed.ProductTypeId == nil || *ed.ProductTypeId != 2073 {
		t.Errorf("ExtractorDetails.ProductTypeId = %v, want *2073", ed.ProductTypeId)
	}
	if ed.QtyPerCycle == nil || *ed.QtyPerCycle != 100 {
		t.Errorf("ExtractorDetails.QtyPerCycle = %v, want *100", ed.QtyPerCycle)
	}
	if len(ed.Heads) != 1 || ed.Heads[0].HeadId != 0 || ed.Heads[0].Latitude != 0.3 || ed.Heads[0].Longitude != 0.4 {
		t.Errorf("ExtractorDetails.Heads = %+v, want one entry {HeadId:0 Latitude:0.3 Longitude:0.4}", ed.Heads)
	}
	if p0.FactoryDetails == nil || p0.FactoryDetails.SchematicId != 100 {
		t.Errorf("Pins[0].FactoryDetails = %+v, want non-nil with SchematicId=100", p0.FactoryDetails)
	}
	p1 := detail.Pins[1]
	if p1.ExtractorDetails != nil || p1.FactoryDetails != nil || len(p1.Contents) != 0 {
		t.Errorf("Pins[1] = %+v, want no extractor/factory details and no contents", p1)
	}

	if len(detail.Routes) != 1 {
		t.Fatalf("Routes = %+v, want one entry", detail.Routes)
	}
	r := detail.Routes[0]
	if r.ContentTypeId != 2073 || r.DestinationPinId != 1002 || r.Quantity != 20 || r.RouteId != 5001 || r.SourcePinId != 1001 {
		t.Errorf("Routes[0] = %+v, want ContentTypeId=2073 DestinationPinId=1002 Quantity=20 RouteId=5001 SourcePinId=1001", r)
	}
	if len(r.Waypoints) != 2 || r.Waypoints[0] != 1003 || r.Waypoints[1] != 1004 {
		t.Errorf("Routes[0].Waypoints = %+v, want [1003 1004]", r.Waypoints)
	}
}

func TestCharacterPlanetDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterPlanetDetail(context.Background(), 40000001); err == nil {
		t.Fatal("CharacterPlanetDetail returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationCustomsOffices_ParsesResponseAndPagination(t *testing.T) {
	var gotPage string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPage = r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{
			"alliance_tax_rate":0.1,
			"allow_access_with_standings":true,
			"allow_alliance_access":true,
			"bad_standing_tax_rate":0.5,
			"corporation_tax_rate":0.05,
			"excellent_standing_tax_rate":0.02,
			"good_standing_tax_rate":0.03,
			"neutral_standing_tax_rate":0.08,
			"office_id":1234567890,
			"reinforce_exit_end":18,
			"reinforce_exit_start":16,
			"standing_level":"good",
			"system_id":30000001,
			"terrible_standing_tax_rate":0.75,
			"type_id":2233
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	offices, totalPages, err := client.CorporationCustomsOffices(context.Background(), 98000001, 2)
	if err != nil {
		t.Fatalf("CorporationCustomsOffices returned error: %v", err)
	}
	if gotPage != "2" {
		t.Errorf("page query param = %q, want \"2\"", gotPage)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(offices) != 1 {
		t.Fatalf("offices = %+v, want one entry", offices)
	}
	o := offices[0]
	if o.OfficeId != 1234567890 || o.SystemId != 30000001 {
		t.Errorf("offices[0] = %+v, want OfficeId=1234567890 SystemId=30000001", o)
	}
	if !o.AllowAccessWithStandings || !o.AllowAllianceAccess {
		t.Errorf("offices[0] = %+v, want AllowAccessWithStandings and AllowAllianceAccess both true", o)
	}
	if o.AllianceTaxRate == nil || *o.AllianceTaxRate != 0.1 {
		t.Errorf("offices[0].AllianceTaxRate = %v, want *0.1", o.AllianceTaxRate)
	}
	if o.BadStandingTaxRate == nil || *o.BadStandingTaxRate != 0.5 {
		t.Errorf("offices[0].BadStandingTaxRate = %v, want *0.5", o.BadStandingTaxRate)
	}
	if o.CorporationTaxRate == nil || *o.CorporationTaxRate != 0.05 {
		t.Errorf("offices[0].CorporationTaxRate = %v, want *0.05", o.CorporationTaxRate)
	}
	if o.ExcellentStandingTaxRate == nil || *o.ExcellentStandingTaxRate != 0.02 {
		t.Errorf("offices[0].ExcellentStandingTaxRate = %v, want *0.02", o.ExcellentStandingTaxRate)
	}
	if o.GoodStandingTaxRate == nil || *o.GoodStandingTaxRate != 0.03 {
		t.Errorf("offices[0].GoodStandingTaxRate = %v, want *0.03", o.GoodStandingTaxRate)
	}
	if o.NeutralStandingTaxRate == nil || *o.NeutralStandingTaxRate != 0.08 {
		t.Errorf("offices[0].NeutralStandingTaxRate = %v, want *0.08", o.NeutralStandingTaxRate)
	}
	if o.TerribleStandingTaxRate == nil || *o.TerribleStandingTaxRate != 0.75 {
		t.Errorf("offices[0].TerribleStandingTaxRate = %v, want *0.75", o.TerribleStandingTaxRate)
	}
	if o.StandingLevel == nil || *o.StandingLevel != "good" {
		t.Errorf("offices[0].StandingLevel = %v, want *\"good\"", o.StandingLevel)
	}
	if o.TypeId == nil || *o.TypeId != 2233 {
		t.Errorf("offices[0].TypeId = %v, want *2233", o.TypeId)
	}
	if o.ReinforceExitEnd != 18 || o.ReinforceExitStart != 16 {
		t.Errorf("offices[0] reinforce window = start=%d end=%d, want start=16 end=18", o.ReinforceExitStart, o.ReinforceExitEnd)
	}
}

func TestCorporationCustomsOffices_OmitsPageWhenZero(t *testing.T) {
	var sawPageParam bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, sawPageParam = r.URL.Query()["page"]
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, _, err := client.CorporationCustomsOffices(context.Background(), 98000001, 0); err != nil {
		t.Fatalf("CorporationCustomsOffices returned error: %v", err)
	}
	if sawPageParam {
		t.Error("page query param present, want it omitted when page=0")
	}
}

func TestCorporationCustomsOffices_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, _, err := client.CorporationCustomsOffices(context.Background(), 98000001, 0); err == nil {
		t.Fatal("CorporationCustomsOffices returned nil error, want an error for HTTP 403")
	}
}

func TestPlanetarySchematic_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"cycle_time":1800,"schematic_name":"Water"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	schematic, err := client.PlanetarySchematic(context.Background(), 90)
	if err != nil {
		t.Fatalf("PlanetarySchematic returned error: %v", err)
	}
	if schematic.CycleTime != 1800 || schematic.SchematicName != "Water" {
		t.Errorf("schematic = %+v, want CycleTime=1800 SchematicName=\"Water\"", schematic)
	}
}

func TestPlanetarySchematic_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.PlanetarySchematic(context.Background(), 90); err == nil {
		t.Fatal("PlanetarySchematic returned nil error, want an error for HTTP 404")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterPlanets|TestCharacterPlanetDetail|TestCorporationCustomsOffices|TestPlanetarySchematic' -v`
Expected: FAIL — `esi.Client` has none of these methods yet.

- [ ] **Step 3: Write the implementation**

Create `esi/planetary_interaction.go`:

```go
// Package esi's planetary_interaction.go covers all 4 of "Planetary Interaction"'s endpoints:
// list a character's colonized planets, get one colony's full layout (pins/links/routes), list a
// corporation's customs offices (POCOs), and look up a public PI schematic.
//
// The colony-layout endpoint (CharacterPlanetDetail) returns this batch's most structurally
// complex model -- but it is NOT recursive. Verified field-by-field against goesi-openapi
// v0.0.37's generated models: esiapi.CharactersCharacterIdPlanetsPlanetIdGet has exactly 3 flat
// sibling slice fields -- Links, Pins, Routes. Pins may carry one level of nested detail
// (ExtractorDetails, itself containing a Heads slice; FactoryDetails; Contents), but nothing in
// that nesting is a Pin, a Link, or a Route. Links and Routes reference pins purely by int64 ID
// (SourcePinId/DestinationPinId/Waypoints []int64), never by embedding another Link/Route/Pin
// value. There is no self-referential or mutually-recursive Go type here, unlike the shape
// Killmails had to split apart (see mcp/killmails.go's KillmailItem/KillmailInnerItem note) --
// this batch needs no equivalent split at the esi/ layer since goesi-openapi already generated
// flat, non-recursive types.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterPlanets returns the client's own character's colonized planets. Private
// ("my own character" -- OAuth2 scope esi-planets.manage_planets.v1, verified against
// esi/api/openapi.yaml). Not paginated -- the generated request builder has no Page() setter.
func (c *Client) CharacterPlanets(ctx context.Context) ([]esiapi.CharactersCharacterIdPlanetsGetInner, error) {
	result, resp, err := c.api.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanets(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character planets: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterPlanetDetail returns the full colony layout (links, pins, routes) of one of the
// client's own character's planets. Private ("my own character" -- OAuth2 scope
// esi-planets.manage_planets.v1, verified against esi/api/openapi.yaml). Not paginated -- the
// generated request builder has no Page() setter.
func (c *Client) CharacterPlanetDetail(ctx context.Context, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
	result, resp, err := c.api.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanetsPlanetId(ctx, int64(c.characterID), planetID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character planet detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationCustomsOffices returns corporationID's customs offices (POCOs), including their tax
// rates and reinforcement schedule. Private, corp-scoped like esi/industry.go's pattern: the
// authenticated character must hold sufficient corp roles (Director, per openapi.yaml's
// x-required-roles) -- ESI itself returns 403 otherwise; this facade adds no additional gate.
// OAuth2 scope esi-planets.read_customs_offices.v1, verified against esi/api/openapi.yaml.
// page=0 omits the page query parameter -- the generated request builder DOES have a Page()
// setter here (verified individually; the other 3 methods in this file do not).
func (c *Client) CorporationCustomsOffices(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
	req := c.api.PlanetaryInteractionAPI.GetCorporationsCorporationIdCustomsOffices(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation customs offices: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// PlanetarySchematic returns one PI schematic's detail (cycle time and name). Public -- verified
// against esi/api/openapi.yaml: this operation has no `security:` block at all, unlike the other
// 3 methods in this file. Not paginated.
func (c *Client) PlanetarySchematic(ctx context.Context, schematicID int64) (*esiapi.UniverseSchematicsSchematicIdGet, error) {
	result, resp, err := c.api.PlanetaryInteractionAPI.GetUniverseSchematicsSchematicId(ctx, schematicID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get planetary schematic: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./esi/... -run 'TestCharacterPlanets|TestCharacterPlanetDetail|TestCorporationCustomsOffices|TestPlanetarySchematic' -v`
Expected: PASS (9 tests).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./...`
Expected: all clean, no output from `gofmt -l`.

```bash
git add esi/planetary_interaction.go esi/planetary_interaction_test.go
git commit -m "feat: add esi/planetary_interaction.go facade (character planets/colony detail, corp customs offices, schematics)"
```

---

### Task 2: `mcp/planetary_interaction.go` — MCP tool layer + registration + docs

**Files:**
- Create: `mcp/planetary_interaction.go`
- Test: `mcp/planetary_interaction_internal_test.go`
- Modify: `mcp/server.go` (register 4 tools after the `get_esi_status` block; update package doc comment)
- Modify: `mcp/server_test.go` (expected-tools list: 161 → 165)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Planetary Interaction row: `Planned` → `This plan`, only that row)
- Modify: `CLAUDE.md` (tool count "one hundred sixty-one" → "one hundred sixty-five", append 4 tool names)
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: `esi.NewClient(ctx, clientID, characterID, userAgent) (*esi.Client, error)`;
  `esi.NewPublicClient(userAgent) *esi.Client`; all 4 `(*esi.Client)` methods from Task 1;
  `characterExists`/`ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent(version string) string`
  (`mcp/sso`); `config.Store` (`mcp/config`).
- Produces: 4 tool names — `list_character_planets`, `get_character_planet`,
  `list_corporation_customs_offices`, `get_planetary_schematic` — registered in `mcp/server.go`.

- [ ] **Step 1: Write the failing tests**

Create `mcp/planetary_interaction_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// --- list_character_planets ---

func TestListCharacterPlanetsHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterPlanets
	t.Cleanup(func() { listCharacterPlanets = orig })
	lastUpdate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	listCharacterPlanets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Planet, error) {
		return []Planet{{
			PlanetID: 40000001, SolarSystemID: 30000001, PlanetType: "plasma",
			OwnerID: 90000001, UpgradeLevel: 3, NumPins: 5, LastUpdate: lastUpdate,
		}}, nil
	}

	handler := listCharacterPlanetsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterPlanetsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Planets) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one planet", out)
	}
	p := out.Planets[0]
	if p.PlanetID != 40000001 || p.SolarSystemID != 30000001 || p.PlanetType != "plasma" ||
		p.OwnerID != 90000001 || p.UpgradeLevel != 3 || p.NumPins != 5 || !p.LastUpdate.Equal(lastUpdate) {
		t.Errorf("planet = %+v, want PlanetID=40000001 SolarSystemID=30000001 PlanetType=plasma OwnerID=90000001 UpgradeLevel=3 NumPins=5 LastUpdate=%v", p, lastUpdate)
	}
}

func TestListCharacterPlanetsHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterPlanets
	t.Cleanup(func() { listCharacterPlanets = orig })
	listCharacterPlanets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Planet, error) {
		t.Fatal("listCharacterPlanets should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterPlanetsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterPlanetsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- get_character_planet ---
//
// This test exercises every nested level of the colony-layout mapping: top-level Links/Pins/
// Routes, plus a Pin carrying Contents, ExtractorDetails (with a nested Heads entry), and
// FactoryDetails, and a second Pin with none of those optional nested fields set (proving the
// pointer/slice fields are correctly left nil/empty rather than defaulted to zero-value structs).

func TestGetCharacterPlanetHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getCharacterPlanet
	t.Cleanup(func() { getCharacterPlanet = orig })

	expiryTime := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	installTime := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	lastCycleStart := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	schematicID := int64(100)
	cycleTime := int64(1800)
	headRadius := 200.5
	productTypeID := int64(2073)
	qtyPerCycle := int64(100)

	getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
		return &esiapi.CharactersCharacterIdPlanetsPlanetIdGet{
			Links: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetLinksInner{
				{SourcePinId: 1001, DestinationPinId: 1002, LinkLevel: 1},
			},
			Pins: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInner{
				{
					PinId: 1001, TypeId: 2254, Latitude: 0.1, Longitude: 0.2,
					SchematicId: &schematicID, ExpiryTime: &expiryTime, InstallTime: &installTime,
					LastCycleStart: &lastCycleStart,
					Contents: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerContentsInner{
						{TypeId: 2073, Amount: 500},
					},
					ExtractorDetails: &esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerExtractorDetails{
						CycleTime: &cycleTime, HeadRadius: &headRadius, ProductTypeId: &productTypeID, QtyPerCycle: &qtyPerCycle,
						Heads: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerExtractorDetailsHeadsInner{
							{HeadId: 0, Latitude: 0.3, Longitude: 0.4},
						},
					},
					FactoryDetails: &esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInnerFactoryDetails{SchematicId: 100},
				},
				{PinId: 1002, TypeId: 2255, Latitude: 0.5, Longitude: 0.6},
			},
			Routes: []esiapi.CharactersCharacterIdPlanetsPlanetIdGetRoutesInner{
				{
					RouteId: 5001, SourcePinId: 1001, DestinationPinId: 1002, ContentTypeId: 2073,
					Quantity: 20, Waypoints: []int64{1003, 1004},
				},
			},
		}, nil
	}

	handler := getCharacterPlanetHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterPlanetInput{CharacterID: 7, PlanetID: 40000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.PlanetID != 40000001 {
		t.Fatalf("out = %+v, want CharacterID=7 PlanetID=40000001", out)
	}

	if len(out.Links) != 1 {
		t.Fatalf("Links = %+v, want one entry", out.Links)
	}
	l := out.Links[0]
	if l.SourcePinID != 1001 || l.DestinationPinID != 1002 || l.LinkLevel != 1 {
		t.Errorf("Links[0] = %+v, want SourcePinID=1001 DestinationPinID=1002 LinkLevel=1", l)
	}

	if len(out.Pins) != 2 {
		t.Fatalf("Pins = %+v, want two entries", out.Pins)
	}
	p0 := out.Pins[0]
	if p0.PinID != 1001 || p0.TypeID != 2254 || p0.Latitude != 0.1 || p0.Longitude != 0.2 {
		t.Errorf("Pins[0] = %+v, want PinID=1001 TypeID=2254 Latitude=0.1 Longitude=0.2", p0)
	}
	if p0.SchematicID == nil || *p0.SchematicID != 100 {
		t.Errorf("Pins[0].SchematicID = %v, want *100", p0.SchematicID)
	}
	if p0.ExpiryTime == nil || !p0.ExpiryTime.Equal(expiryTime) {
		t.Errorf("Pins[0].ExpiryTime = %v, want %v", p0.ExpiryTime, expiryTime)
	}
	if p0.InstallTime == nil || !p0.InstallTime.Equal(installTime) {
		t.Errorf("Pins[0].InstallTime = %v, want %v", p0.InstallTime, installTime)
	}
	if p0.LastCycleStart == nil || !p0.LastCycleStart.Equal(lastCycleStart) {
		t.Errorf("Pins[0].LastCycleStart = %v, want %v", p0.LastCycleStart, lastCycleStart)
	}
	if len(p0.Contents) != 1 || p0.Contents[0].TypeID != 2073 || p0.Contents[0].Amount != 500 {
		t.Errorf("Pins[0].Contents = %+v, want one entry {TypeID:2073 Amount:500}", p0.Contents)
	}
	if p0.ExtractorDetails == nil {
		t.Fatal("Pins[0].ExtractorDetails = nil, want non-nil")
	}
	ed := p0.ExtractorDetails
	if ed.CycleTime == nil || *ed.CycleTime != 1800 {
		t.Errorf("ExtractorDetails.CycleTime = %v, want *1800", ed.CycleTime)
	}
	if ed.HeadRadius == nil || *ed.HeadRadius != 200.5 {
		t.Errorf("ExtractorDetails.HeadRadius = %v, want *200.5", ed.HeadRadius)
	}
	if ed.ProductTypeID == nil || *ed.ProductTypeID != 2073 {
		t.Errorf("ExtractorDetails.ProductTypeID = %v, want *2073", ed.ProductTypeID)
	}
	if ed.QtyPerCycle == nil || *ed.QtyPerCycle != 100 {
		t.Errorf("ExtractorDetails.QtyPerCycle = %v, want *100", ed.QtyPerCycle)
	}
	if len(ed.Heads) != 1 || ed.Heads[0].HeadID != 0 || ed.Heads[0].Latitude != 0.3 || ed.Heads[0].Longitude != 0.4 {
		t.Errorf("ExtractorDetails.Heads = %+v, want one entry {HeadID:0 Latitude:0.3 Longitude:0.4}", ed.Heads)
	}
	if p0.FactoryDetails == nil || p0.FactoryDetails.SchematicID != 100 {
		t.Errorf("Pins[0].FactoryDetails = %+v, want non-nil with SchematicID=100", p0.FactoryDetails)
	}
	p1 := out.Pins[1]
	if p1.PinID != 1002 || p1.TypeID != 2255 {
		t.Errorf("Pins[1] = %+v, want PinID=1002 TypeID=2255", p1)
	}
	if p1.ExtractorDetails != nil || p1.FactoryDetails != nil || len(p1.Contents) != 0 || p1.SchematicID != nil {
		t.Errorf("Pins[1] = %+v, want no extractor/factory details, no contents, no schematicId", p1)
	}

	if len(out.Routes) != 1 {
		t.Fatalf("Routes = %+v, want one entry", out.Routes)
	}
	r := out.Routes[0]
	if r.RouteID != 5001 || r.SourcePinID != 1001 || r.DestinationPinID != 1002 || r.ContentTypeID != 2073 || r.Quantity != 20 {
		t.Errorf("Routes[0] = %+v, want RouteID=5001 SourcePinID=1001 DestinationPinID=1002 ContentTypeID=2073 Quantity=20", r)
	}
	if len(r.Waypoints) != 2 || r.Waypoints[0] != 1003 || r.Waypoints[1] != 1004 {
		t.Errorf("Routes[0].Waypoints = %+v, want [1003 1004]", r.Waypoints)
	}
}

func TestGetCharacterPlanetHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCharacterPlanet
	t.Cleanup(func() { getCharacterPlanet = orig })
	getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
		t.Fatal("getCharacterPlanet should not be called for an unknown character")
		return nil, nil
	}

	handler := getCharacterPlanetHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterPlanetInput{CharacterID: 999, PlanetID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_corporation_customs_offices ---

func TestListCorporationCustomsOfficesHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })

	allianceTax := 0.1
	badTax := 0.5
	corpTax := 0.05
	excellentTax := 0.02
	goodTax := 0.03
	neutralTax := 0.08
	terribleTax := 0.75
	standingLevel := "good"
	typeID := int64(2233)

	var gotCorporationID int64
	var gotPage int32
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		gotCorporationID = corporationID
		gotPage = page
		return []esiapi.CorporationsCorporationIdCustomsOfficesGetInner{
			{
				OfficeId: 1234567890, SystemId: 30000001, TypeId: &typeID,
				ReinforceExitStart: 16, ReinforceExitEnd: 18,
				AllowAllianceAccess: true, AllowAccessWithStandings: true,
				StandingLevel: &standingLevel, AllianceTaxRate: &allianceTax, CorporationTaxRate: &corpTax,
				ExcellentStandingTaxRate: &excellentTax, GoodStandingTaxRate: &goodTax, NeutralStandingTaxRate: &neutralTax,
				BadStandingTaxRate: &badTax, TerribleStandingTaxRate: &terribleTax,
			},
		}, 3, nil
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 7, CorporationID: 98000001, Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCorporationID != 98000001 || gotPage != 2 {
		t.Errorf("captured corporationID/page = %d/%d, want 98000001/2", gotCorporationID, gotPage)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 2 || out.TotalPages != 3 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=2 TotalPages=3", out)
	}
	if len(out.Offices) != 1 {
		t.Fatalf("Offices = %+v, want one entry", out.Offices)
	}
	o := out.Offices[0]
	if o.OfficeID != 1234567890 || o.SystemID != 30000001 {
		t.Errorf("office = %+v, want OfficeID=1234567890 SystemID=30000001", o)
	}
	if o.TypeID == nil || *o.TypeID != 2233 {
		t.Errorf("office.TypeID = %v, want *2233", o.TypeID)
	}
	if o.ReinforceExitStart != 16 || o.ReinforceExitEnd != 18 {
		t.Errorf("office reinforce window = start=%d end=%d, want start=16 end=18", o.ReinforceExitStart, o.ReinforceExitEnd)
	}
	if !o.AllowAllianceAccess || !o.AllowAccessWithStandings {
		t.Errorf("office = %+v, want AllowAllianceAccess and AllowAccessWithStandings both true", o)
	}
	if o.StandingLevel == nil || *o.StandingLevel != "good" {
		t.Errorf("office.StandingLevel = %v, want *\"good\"", o.StandingLevel)
	}
	if o.AllianceTaxRate == nil || *o.AllianceTaxRate != 0.1 {
		t.Errorf("office.AllianceTaxRate = %v, want *0.1", o.AllianceTaxRate)
	}
	if o.CorporationTaxRate == nil || *o.CorporationTaxRate != 0.05 {
		t.Errorf("office.CorporationTaxRate = %v, want *0.05", o.CorporationTaxRate)
	}
	if o.ExcellentStandingTaxRate == nil || *o.ExcellentStandingTaxRate != 0.02 {
		t.Errorf("office.ExcellentStandingTaxRate = %v, want *0.02", o.ExcellentStandingTaxRate)
	}
	if o.GoodStandingTaxRate == nil || *o.GoodStandingTaxRate != 0.03 {
		t.Errorf("office.GoodStandingTaxRate = %v, want *0.03", o.GoodStandingTaxRate)
	}
	if o.NeutralStandingTaxRate == nil || *o.NeutralStandingTaxRate != 0.08 {
		t.Errorf("office.NeutralStandingTaxRate = %v, want *0.08", o.NeutralStandingTaxRate)
	}
	if o.BadStandingTaxRate == nil || *o.BadStandingTaxRate != 0.5 {
		t.Errorf("office.BadStandingTaxRate = %v, want *0.5", o.BadStandingTaxRate)
	}
	if o.TerribleStandingTaxRate == nil || *o.TerribleStandingTaxRate != 0.75 {
		t.Errorf("office.TerribleStandingTaxRate = %v, want *0.75", o.TerribleStandingTaxRate)
	}
}

func TestListCorporationCustomsOfficesHandler_DefaultsPageToOne(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		return nil, 1, nil
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("Page = %d, want 1 when input page is omitted", out.Page)
	}
}

func TestListCorporationCustomsOfficesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationCustomsOffices
	t.Cleanup(func() { listCorporationCustomsOffices = orig })
	listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
		t.Fatal("listCorporationCustomsOffices should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationCustomsOfficesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationCustomsOfficesInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- get_planetary_schematic (public, no character check) ---

func TestGetPlanetarySchematicHandler_MapsFields(t *testing.T) {
	orig := getPlanetarySchematic
	t.Cleanup(func() { getPlanetarySchematic = orig })
	getPlanetarySchematic = func(ctx context.Context, userAgent string, schematicID int64) (GetPlanetarySchematicOutput, error) {
		return GetPlanetarySchematicOutput{SchematicID: schematicID, SchematicName: "Water", CycleTime: 1800}, nil
	}

	handler := getPlanetarySchematicHandler("v1")
	_, out, err := handler(context.Background(), nil, GetPlanetarySchematicInput{SchematicID: 90})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SchematicID != 90 || out.SchematicName != "Water" || out.CycleTime != 1800 {
		t.Errorf("out = %+v, want SchematicID=90 SchematicName=\"Water\" CycleTime=1800", out)
	}
}

func TestGetPlanetarySchematicHandler_PropagatesError(t *testing.T) {
	orig := getPlanetarySchematic
	t.Cleanup(func() { getPlanetarySchematic = orig })
	getPlanetarySchematic = func(ctx context.Context, userAgent string, schematicID int64) (GetPlanetarySchematicOutput, error) {
		return GetPlanetarySchematicOutput{}, errors.New("boom")
	}

	handler := getPlanetarySchematicHandler("v1")
	_, _, err := handler(context.Background(), nil, GetPlanetarySchematicInput{SchematicID: 90})
	if err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListCharacterPlanetsHandler|TestGetCharacterPlanetHandler|TestListCorporationCustomsOfficesHandler|TestGetPlanetarySchematicHandler' -v`
Expected: FAIL — undefined symbols (none of these handlers/types exist yet).

- [ ] **Step 3: Write the implementation**

Create `mcp/planetary_interaction.go`:

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_character_planets (private) ---

type Planet struct {
	PlanetID      int64     `json:"planetId"`
	SolarSystemID int64     `json:"solarSystemId"`
	PlanetType    string    `json:"planetType"`
	OwnerID       int64     `json:"ownerId"`
	UpgradeLevel  int64     `json:"upgradeLevel"`
	NumPins       int64     `json:"numPins"`
	LastUpdate    time.Time `json:"lastUpdate"`
}

var listCharacterPlanets = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Planet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	planets, err := client.CharacterPlanets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Planet, len(planets))
	for i, p := range planets {
		out[i] = Planet{
			PlanetID: p.PlanetId, SolarSystemID: p.SolarSystemId, PlanetType: p.PlanetType,
			OwnerID: p.OwnerId, UpgradeLevel: p.UpgradeLevel, NumPins: p.NumPins, LastUpdate: p.LastUpdate,
		}
	}
	return out, nil
}

type ListCharacterPlanetsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterPlanetsOutput struct {
	CharacterID int32    `json:"characterId"`
	Planets     []Planet `json:"planets"`
}

func listCharacterPlanetsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterPlanetsInput, ListCharacterPlanetsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterPlanetsInput) (*mcpsdk.CallToolResult, ListCharacterPlanetsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterPlanetsOutput{}, err
		}
		if !found {
			return nil, ListCharacterPlanetsOutput{}, fmt.Errorf("list character planets for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		planets, err := listCharacterPlanets(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterPlanetsOutput{}, fmt.Errorf("list character planets: %w", err)
		}
		return nil, ListCharacterPlanetsOutput{CharacterID: in.CharacterID, Planets: planets}, nil
	}
}

// --- get_character_planet (private) ---
//
// The colony-layout response is this batch's most structurally complex model, but it is not
// recursive: ESI's own CharactersCharacterIdPlanetsPlanetIdGet has exactly 3 flat sibling
// slices -- Links, Pins, Routes -- and PlanetLink/PlanetPin/PlanetRoute below mirror that shape
// exactly, one distinct type per real nesting level (PlanetPin's ExtractorDetails/FactoryDetails/
// Contents go one level deeper; PlanetExtractorDetails' Heads go one level deeper than that).
// Links and Routes reference pins purely by int64 ID (SourcePinID/DestinationPinID/Waypoints),
// never by embedding another PlanetLink/PlanetPin/PlanetRoute value, so none of these types is
// self-referential -- no Killmails-style split into "outer"/"inner" variants of the same
// conceptual type is needed here, because goesi-openapi never generated a recursive shape for PI
// in the first place.

type PlanetLink struct {
	SourcePinID      int64 `json:"sourcePinId"`
	DestinationPinID int64 `json:"destinationPinId"`
	LinkLevel        int64 `json:"linkLevel"`
}

type PlanetPinContent struct {
	TypeID int64 `json:"typeId"`
	Amount int64 `json:"amount"`
}

type PlanetExtractorHead struct {
	HeadID    int64   `json:"headId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PlanetExtractorDetails struct {
	CycleTime     *int64                `json:"cycleTime,omitempty"`
	HeadRadius    *float64              `json:"headRadius,omitempty"`
	ProductTypeID *int64                `json:"productTypeId,omitempty"`
	QtyPerCycle   *int64                `json:"qtyPerCycle,omitempty"`
	Heads         []PlanetExtractorHead `json:"heads"`
}

type PlanetFactoryDetails struct {
	SchematicID int64 `json:"schematicId"`
}

type PlanetPin struct {
	PinID            int64                   `json:"pinId"`
	TypeID           int64                   `json:"typeId"`
	Latitude         float64                 `json:"latitude"`
	Longitude        float64                 `json:"longitude"`
	SchematicID      *int64                  `json:"schematicId,omitempty"`
	ExpiryTime       *time.Time              `json:"expiryTime,omitempty"`
	InstallTime      *time.Time              `json:"installTime,omitempty"`
	LastCycleStart   *time.Time              `json:"lastCycleStart,omitempty"`
	Contents         []PlanetPinContent      `json:"contents,omitempty"`
	ExtractorDetails *PlanetExtractorDetails `json:"extractorDetails,omitempty"`
	FactoryDetails   *PlanetFactoryDetails   `json:"factoryDetails,omitempty"`
}

type PlanetRoute struct {
	RouteID          int64   `json:"routeId"`
	SourcePinID      int64   `json:"sourcePinId"`
	DestinationPinID int64   `json:"destinationPinId"`
	ContentTypeID    int64   `json:"contentTypeId"`
	Quantity         float64 `json:"quantity"`
	Waypoints        []int64 `json:"waypoints,omitempty"`
}

func toPlanetPins(pins []esiapi.CharactersCharacterIdPlanetsPlanetIdGetPinsInner) []PlanetPin {
	out := make([]PlanetPin, len(pins))
	for i, p := range pins {
		pin := PlanetPin{
			PinID: p.PinId, TypeID: p.TypeId, Latitude: p.Latitude, Longitude: p.Longitude,
			SchematicID: p.SchematicId, ExpiryTime: p.ExpiryTime, InstallTime: p.InstallTime,
			LastCycleStart: p.LastCycleStart,
		}
		if len(p.Contents) > 0 {
			contents := make([]PlanetPinContent, len(p.Contents))
			for j, c := range p.Contents {
				contents[j] = PlanetPinContent{TypeID: c.TypeId, Amount: c.Amount}
			}
			pin.Contents = contents
		}
		if p.ExtractorDetails != nil {
			heads := make([]PlanetExtractorHead, len(p.ExtractorDetails.Heads))
			for j, h := range p.ExtractorDetails.Heads {
				heads[j] = PlanetExtractorHead{HeadID: h.HeadId, Latitude: h.Latitude, Longitude: h.Longitude}
			}
			pin.ExtractorDetails = &PlanetExtractorDetails{
				CycleTime: p.ExtractorDetails.CycleTime, HeadRadius: p.ExtractorDetails.HeadRadius,
				ProductTypeID: p.ExtractorDetails.ProductTypeId, QtyPerCycle: p.ExtractorDetails.QtyPerCycle,
				Heads: heads,
			}
		}
		if p.FactoryDetails != nil {
			pin.FactoryDetails = &PlanetFactoryDetails{SchematicID: p.FactoryDetails.SchematicId}
		}
		out[i] = pin
	}
	return out
}

func toPlanetLinks(links []esiapi.CharactersCharacterIdPlanetsPlanetIdGetLinksInner) []PlanetLink {
	out := make([]PlanetLink, len(links))
	for i, l := range links {
		out[i] = PlanetLink{SourcePinID: l.SourcePinId, DestinationPinID: l.DestinationPinId, LinkLevel: l.LinkLevel}
	}
	return out
}

func toPlanetRoutes(routes []esiapi.CharactersCharacterIdPlanetsPlanetIdGetRoutesInner) []PlanetRoute {
	out := make([]PlanetRoute, len(routes))
	for i, r := range routes {
		out[i] = PlanetRoute{
			RouteID: r.RouteId, SourcePinID: r.SourcePinId, DestinationPinID: r.DestinationPinId,
			ContentTypeID: r.ContentTypeId, Quantity: r.Quantity, Waypoints: r.Waypoints,
		}
	}
	return out
}

var getCharacterPlanet = func(ctx context.Context, clientID string, characterID int32, userAgent string, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterPlanetDetail(ctx, planetID)
}

type GetCharacterPlanetInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	PlanetID    int64 `json:"planetId" jsonschema:"the planet ID, from list_character_planets"`
}
type GetCharacterPlanetOutput struct {
	CharacterID int32         `json:"characterId"`
	PlanetID    int64         `json:"planetId"`
	Links       []PlanetLink  `json:"links"`
	Pins        []PlanetPin   `json:"pins"`
	Routes      []PlanetRoute `json:"routes"`
}

func getCharacterPlanetHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterPlanetInput, GetCharacterPlanetOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterPlanetInput) (*mcpsdk.CallToolResult, GetCharacterPlanetOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterPlanetOutput{}, err
		}
		if !found {
			return nil, GetCharacterPlanetOutput{}, fmt.Errorf("get character planet for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCharacterPlanet(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.PlanetID)
		if err != nil {
			return nil, GetCharacterPlanetOutput{}, fmt.Errorf("get character planet: %w", err)
		}
		return nil, GetCharacterPlanetOutput{
			CharacterID: in.CharacterID, PlanetID: in.PlanetID,
			Links: toPlanetLinks(detail.Links), Pins: toPlanetPins(detail.Pins), Routes: toPlanetRoutes(detail.Routes),
		}, nil
	}
}

// --- list_corporation_customs_offices (private, corp-scoped, paginated) ---

type CustomsOffice struct {
	OfficeID                 int64    `json:"officeId"`
	SystemID                 int64    `json:"systemId"`
	TypeID                   *int64   `json:"typeId,omitempty"`
	ReinforceExitStart       int64    `json:"reinforceExitStart"`
	ReinforceExitEnd         int64    `json:"reinforceExitEnd"`
	AllowAllianceAccess      bool     `json:"allowAllianceAccess"`
	AllowAccessWithStandings bool     `json:"allowAccessWithStandings"`
	StandingLevel            *string  `json:"standingLevel,omitempty"`
	AllianceTaxRate          *float64 `json:"allianceTaxRate,omitempty"`
	CorporationTaxRate       *float64 `json:"corporationTaxRate,omitempty"`
	ExcellentStandingTaxRate *float64 `json:"excellentStandingTaxRate,omitempty"`
	GoodStandingTaxRate      *float64 `json:"goodStandingTaxRate,omitempty"`
	NeutralStandingTaxRate   *float64 `json:"neutralStandingTaxRate,omitempty"`
	BadStandingTaxRate       *float64 `json:"badStandingTaxRate,omitempty"`
	TerribleStandingTaxRate  *float64 `json:"terribleStandingTaxRate,omitempty"`
}

var listCorporationCustomsOffices = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationCustomsOffices(ctx, corporationID, page)
}

type ListCorporationCustomsOfficesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationCustomsOfficesOutput struct {
	CharacterID   int32           `json:"characterId"`
	CorporationID int64           `json:"corporationId"`
	Page          int32           `json:"page"`
	TotalPages    int32           `json:"totalPages"`
	Offices       []CustomsOffice `json:"offices"`
}

func listCorporationCustomsOfficesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationCustomsOfficesInput, ListCorporationCustomsOfficesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationCustomsOfficesInput) (*mcpsdk.CallToolResult, ListCorporationCustomsOfficesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationCustomsOfficesOutput{}, err
		}
		if !found {
			return nil, ListCorporationCustomsOfficesOutput{}, fmt.Errorf("list corporation customs offices for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		offices, totalPages, err := listCorporationCustomsOffices(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationCustomsOfficesOutput{}, fmt.Errorf("list corporation customs offices: %w", err)
		}
		out := make([]CustomsOffice, len(offices))
		for i, o := range offices {
			out[i] = CustomsOffice{
				OfficeID: o.OfficeId, SystemID: o.SystemId, TypeID: o.TypeId,
				ReinforceExitStart: o.ReinforceExitStart, ReinforceExitEnd: o.ReinforceExitEnd,
				AllowAllianceAccess: o.AllowAllianceAccess, AllowAccessWithStandings: o.AllowAccessWithStandings,
				StandingLevel: o.StandingLevel, AllianceTaxRate: o.AllianceTaxRate,
				CorporationTaxRate: o.CorporationTaxRate, ExcellentStandingTaxRate: o.ExcellentStandingTaxRate,
				GoodStandingTaxRate: o.GoodStandingTaxRate, NeutralStandingTaxRate: o.NeutralStandingTaxRate,
				BadStandingTaxRate: o.BadStandingTaxRate, TerribleStandingTaxRate: o.TerribleStandingTaxRate,
			}
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationCustomsOfficesOutput{
			CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Offices: out,
		}, nil
	}
}

// --- get_planetary_schematic (public) ---

type GetPlanetarySchematicOutput struct {
	SchematicID   int64  `json:"schematicId"`
	SchematicName string `json:"schematicName"`
	CycleTime     int64  `json:"cycleTime"`
}

var getPlanetarySchematic = func(ctx context.Context, userAgent string, schematicID int64) (GetPlanetarySchematicOutput, error) {
	client := esi.NewPublicClient(userAgent)
	schematic, err := client.PlanetarySchematic(ctx, schematicID)
	if err != nil {
		return GetPlanetarySchematicOutput{}, err
	}
	return GetPlanetarySchematicOutput{SchematicID: schematicID, SchematicName: schematic.SchematicName, CycleTime: schematic.CycleTime}, nil
}

type GetPlanetarySchematicInput struct {
	SchematicID int64 `json:"schematicId" jsonschema:"the PI schematic ID"`
}

func getPlanetarySchematicHandler(version string) mcpsdk.ToolHandlerFor[GetPlanetarySchematicInput, GetPlanetarySchematicOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetPlanetarySchematicInput) (*mcpsdk.CallToolResult, GetPlanetarySchematicOutput, error) {
		out, err := getPlanetarySchematic(ctx, sso.UserAgent(version), in.SchematicID)
		if err != nil {
			return nil, GetPlanetarySchematicOutput{}, fmt.Errorf("get planetary schematic: %w", err)
		}
		return nil, out, nil
	}
}
```

Modify `mcp/server.go`: append after the existing `get_esi_status` block (immediately before
`return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_planets",
		Description: "List a logged-in EVE Online character's colonized planets (Planetary Interaction).",
	}, listCharacterPlanetsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_character_planet",
		Description: "Get the full colony layout (links, pins, routes) of one of a character's planets.",
	}, getCharacterPlanetHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_customs_offices",
		Description: "List a corporation's customs offices (POCOs), including tax rates and reinforcement schedule.",
	}, listCorporationCustomsOfficesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_planetary_schematic",
		Description: "Get a Planetary Interaction schematic's cycle time and name.",
	}, getPlanetarySchematicHandler(version))
```

Update `mcp/server.go`'s package doc comment: change the trailing
`"... and get_esi_status registered."`
to end with
`"... get_esi_status, list_character_planets, get_character_planet, list_corporation_customs_offices, and get_planetary_schematic registered."`

Modify `mcp/server_test.go`: append `"list_character_planets"`, `"get_character_planet"`,
`"list_corporation_customs_offices"`, `"get_planetary_schematic"` to the expected-tools slice
(after `"get_esi_status"`), matching the file's existing membership-check style.

Modify `docs/reference/esi-coverage-roadmap.md`: flip only the Planetary Interaction row from
`Planned` to `This plan`. Do not touch any other row.

Modify `CLAUDE.md`: change "one hundred sixty-one" to "one hundred sixty-five", and append the
4 new tool names to the `mcp/` row's tool list.

Modify `README.md`: mirror the same two edits.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./mcp/... -run 'TestListCharacterPlanetsHandler|TestGetCharacterPlanetHandler|TestListCorporationCustomsOfficesHandler|TestGetPlanetarySchematicHandler|TestServer_AllToolsRegistered' -v`
Expected: PASS (10 new handler tests + the updated `TestServer_AllToolsRegistered`).

- [ ] **Step 5: Full verification and commit**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race`
Expected: all clean, no output from `gofmt -l`, 165 tools registered without panic.

```bash
git add mcp/planetary_interaction.go mcp/planetary_interaction_internal_test.go mcp/server.go mcp/server_test.go \
  CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add planetary interaction MCP tools (4 tools: character planets/colony detail, corp customs offices, schematics)"
```
