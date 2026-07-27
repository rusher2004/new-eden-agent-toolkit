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
