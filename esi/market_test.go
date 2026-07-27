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

func TestCharacterMarketOrders_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{
			"duration": 90,
			"escrow": 100.5,
			"is_buy_order": true,
			"is_corporation": false,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 123456,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"type_id": 34,
			"volume_remain": 500,
			"volume_total": 1000
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	orders, err := client.CharacterMarketOrders(context.Background())
	if err != nil {
		t.Fatalf("CharacterMarketOrders returned error: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %+v, want one entry", orders)
	}
	o := orders[0]
	if o.Duration != 90 {
		t.Errorf("Duration = %d, want 90", o.Duration)
	}
	if o.Escrow == nil || *o.Escrow != 100.5 {
		t.Errorf("Escrow = %v, want 100.5", o.Escrow)
	}
	if o.IsBuyOrder == nil || !*o.IsBuyOrder {
		t.Errorf("IsBuyOrder = %v, want true", o.IsBuyOrder)
	}
	if o.IsCorporation {
		t.Errorf("IsCorporation = %v, want false", o.IsCorporation)
	}
	wantIssued := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if !o.Issued.Equal(wantIssued) {
		t.Errorf("Issued = %v, want %v", o.Issued, wantIssued)
	}
	if o.LocationId != 60003760 {
		t.Errorf("LocationId = %d, want 60003760", o.LocationId)
	}
	if o.MinVolume == nil || *o.MinVolume != 1 {
		t.Errorf("MinVolume = %v, want 1", o.MinVolume)
	}
	if o.OrderId != 123456 {
		t.Errorf("OrderId = %d, want 123456", o.OrderId)
	}
	if o.Price != 5.55 {
		t.Errorf("Price = %v, want 5.55", o.Price)
	}
	if o.Range != "region" {
		t.Errorf("Range = %q, want \"region\"", o.Range)
	}
	if o.RegionId != 10000002 {
		t.Errorf("RegionId = %d, want 10000002", o.RegionId)
	}
	if o.TypeId != 34 {
		t.Errorf("TypeId = %d, want 34", o.TypeId)
	}
	if o.VolumeRemain != 500 {
		t.Errorf("VolumeRemain = %d, want 500", o.VolumeRemain)
	}
	if o.VolumeTotal != 1000 {
		t.Errorf("VolumeTotal = %d, want 1000", o.VolumeTotal)
	}
}

func TestCharacterMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterMarketOrders(context.Background()); err == nil {
		t.Fatal("CharacterMarketOrders returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterMarketOrderHistory_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{
			"duration": 90,
			"is_corporation": true,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 654321,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"state": "expired",
			"type_id": 34,
			"volume_remain": 0,
			"volume_total": 1000
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	history, totalPages, err := client.CharacterMarketOrderHistory(context.Background(), 2)
	if err != nil {
		t.Fatalf("CharacterMarketOrderHistory returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want \"page=2\"", gotQuery)
	}
	if totalPages != 3 {
		t.Errorf("totalPages = %d, want 3", totalPages)
	}
	if len(history) != 1 {
		t.Fatalf("history = %+v, want one entry", history)
	}
	h := history[0]
	wantIssued := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if h.Duration != 90 || !h.IsCorporation || !h.Issued.Equal(wantIssued) || h.LocationId != 60003760 ||
		h.MinVolume == nil || *h.MinVolume != 1 || h.OrderId != 654321 || h.Price != 5.55 ||
		h.Range != "region" || h.RegionId != 10000002 || h.State != "expired" || h.TypeId != 34 ||
		h.VolumeRemain != 0 || h.VolumeTotal != 1000 {
		t.Errorf("history[0] = %+v, want Duration=90 IsCorporation=true Issued=%v LocationId=60003760 MinVolume=1 OrderId=654321 Price=5.55 Range=region RegionId=10000002 State=expired TypeId=34 VolumeRemain=0 VolumeTotal=1000",
			h, wantIssued)
	}
}

func TestCharacterMarketOrderHistory_PageOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterMarketOrderHistory(context.Background(), 0); err != nil {
		t.Fatalf("CharacterMarketOrderHistory returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
}

func TestCharacterMarketOrderHistory_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterMarketOrderHistory(context.Background(), 0); err == nil {
		t.Fatal("CharacterMarketOrderHistory returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationMarketOrders_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{
			"duration": 90,
			"issued": "2026-07-20T00:00:00Z",
			"issued_by": 998877,
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 111222,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"type_id": 34,
			"volume_remain": 500,
			"volume_total": 1000,
			"wallet_division": 1
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	orders, totalPages, err := client.CorporationMarketOrders(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationMarketOrders returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want \"page=1\"", gotQuery)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %+v, want one entry", orders)
	}
	o := orders[0]
	wantIssued := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if o.Duration != 90 || !o.Issued.Equal(wantIssued) || o.IssuedBy != 998877 || o.LocationId != 60003760 ||
		o.MinVolume == nil || *o.MinVolume != 1 || o.OrderId != 111222 || o.Price != 5.55 ||
		o.Range != "region" || o.RegionId != 10000002 || o.TypeId != 34 || o.VolumeRemain != 500 ||
		o.VolumeTotal != 1000 || o.WalletDivision != 1 {
		t.Errorf("orders[0] = %+v, want Duration=90 Issued=%v IssuedBy=998877 LocationId=60003760 MinVolume=1 OrderId=111222 Price=5.55 Range=region RegionId=10000002 TypeId=34 VolumeRemain=500 VolumeTotal=1000 WalletDivision=1",
			o, wantIssued)
	}
}

func TestCorporationMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationMarketOrders(context.Background(), 98000001, 0); err == nil {
		t.Fatal("CorporationMarketOrders returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationMarketOrderHistory_ParsesResponseAndPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Write([]byte(`[{
			"duration": 90,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 60003760,
			"min_volume": 1,
			"order_id": 333444,
			"price": 5.55,
			"range": "region",
			"region_id": 10000002,
			"state": "cancelled",
			"type_id": 34,
			"volume_remain": 0,
			"volume_total": 1000,
			"wallet_division": 2
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	history, totalPages, err := client.CorporationMarketOrderHistory(context.Background(), 98000001, 0)
	if err != nil {
		t.Fatalf("CorporationMarketOrderHistory returned error: %v", err)
	}
	if totalPages != 1 {
		t.Errorf("totalPages = %d, want 1", totalPages)
	}
	if len(history) != 1 {
		t.Fatalf("history = %+v, want one entry", history)
	}
	h := history[0]
	// IssuedBy is optional (*int64) for corp order history, unlike the required int64 on
	// CorporationMarketOrders -- this fixture omits it entirely to confirm it decodes as nil.
	if h.IssuedBy != nil {
		t.Errorf("IssuedBy = %v, want nil (omitted from fixture)", h.IssuedBy)
	}
	wantIssued := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if h.Duration != 90 || !h.Issued.Equal(wantIssued) || h.LocationId != 60003760 ||
		h.MinVolume == nil || *h.MinVolume != 1 || h.OrderId != 333444 || h.Price != 5.55 ||
		h.Range != "region" || h.RegionId != 10000002 || h.State != "cancelled" || h.TypeId != 34 ||
		h.VolumeRemain != 0 || h.VolumeTotal != 1000 || h.WalletDivision != 2 {
		t.Errorf("history[0] = %+v, want Duration=90 Issued=%v LocationId=60003760 MinVolume=1 OrderId=333444 Price=5.55 Range=region RegionId=10000002 State=cancelled TypeId=34 VolumeRemain=0 VolumeTotal=1000 WalletDivision=2",
			h, wantIssued)
	}
}

func TestCorporationMarketOrderHistory_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationMarketOrderHistory(context.Background(), 98000001, 0); err == nil {
		t.Fatal("CorporationMarketOrderHistory returned nil error, want an error for HTTP 403")
	}
}

func TestMarketGroups_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[1, 2, 3]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	groups, err := client.MarketGroups(context.Background())
	if err != nil {
		t.Fatalf("MarketGroups returned error: %v", err)
	}
	if len(groups) != 3 || groups[0] != 1 || groups[1] != 2 || groups[2] != 3 {
		t.Errorf("groups = %+v, want [1 2 3]", groups)
	}
}

func TestMarketGroups_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketGroups(context.Background()); err == nil {
		t.Fatal("MarketGroups returned nil error, want an error for HTTP 500")
	}
}

func TestMarketGroup_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"description":"Ammo & Charges","market_group_id":9,"name":"Ammo & Charges","parent_group_id":1,"types":[10,11]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	group, err := client.MarketGroup(context.Background(), 9)
	if err != nil {
		t.Fatalf("MarketGroup returned error: %v", err)
	}
	if group.Description != "Ammo & Charges" || group.MarketGroupId != 9 || group.Name != "Ammo & Charges" {
		t.Errorf("group = %+v, want Description/Name=\"Ammo & Charges\" MarketGroupId=9", group)
	}
	if group.ParentGroupId == nil || *group.ParentGroupId != 1 {
		t.Errorf("ParentGroupId = %v, want 1", group.ParentGroupId)
	}
	if len(group.Types) != 2 || group.Types[0] != 10 || group.Types[1] != 11 {
		t.Errorf("Types = %+v, want [10 11]", group.Types)
	}
}

func TestMarketGroup_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketGroup(context.Background(), 9); err == nil {
		t.Fatal("MarketGroup returned nil error, want an error for HTTP 404")
	}
}

func TestMarketPrices_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"adjusted_price":306.11,"average_price":306.29,"type_id":32772}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	prices, err := client.MarketPrices(context.Background())
	if err != nil {
		t.Fatalf("MarketPrices returned error: %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("prices = %+v, want one entry", prices)
	}
	p := prices[0]
	if p.AdjustedPrice == nil || *p.AdjustedPrice != 306.11 {
		t.Errorf("AdjustedPrice = %v, want 306.11", p.AdjustedPrice)
	}
	if p.AveragePrice == nil || *p.AveragePrice != 306.29 {
		t.Errorf("AveragePrice = %v, want 306.29", p.AveragePrice)
	}
	if p.TypeId != 32772 {
		t.Errorf("TypeId = %d, want 32772", p.TypeId)
	}
}

func TestMarketPrices_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketPrices(context.Background()); err == nil {
		t.Fatal("MarketPrices returned nil error, want an error for HTTP 500")
	}
}

func TestMarketHistory_SendsRequiredTypeIdAndParsesResponse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"average":5.5,"date":"2026-07-20","highest":6.0,"lowest":5.0,"order_count":10,"volume":1000}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	history, err := client.MarketHistory(context.Background(), 10000002, 34)
	if err != nil {
		t.Fatalf("MarketHistory returned error: %v", err)
	}
	if gotQuery != "type_id=34" {
		t.Errorf("query = %q, want \"type_id=34\" (typeId is a required query param)", gotQuery)
	}
	if len(history) != 1 {
		t.Fatalf("history = %+v, want one entry", history)
	}
	h := history[0]
	if h.Average != 5.5 || h.Date != "2026-07-20" || h.Highest != 6.0 || h.Lowest != 5.0 || h.OrderCount != 10 || h.Volume != 1000 {
		t.Errorf("history[0] = %+v, want Average=5.5 Date=2026-07-20 Highest=6.0 Lowest=5.0 OrderCount=10 Volume=1000", h)
	}
}

func TestMarketHistory_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.MarketHistory(context.Background(), 10000002, 34); err == nil {
		t.Fatal("MarketHistory returned nil error, want an error for HTTP 404")
	}
}

func TestMarketOrders_SendsRequiredOrderTypeAndOptionalFilters(t *testing.T) {
	tests := []struct {
		name      string
		orderType string
		typeID    int64
		page      int32
		wantQuery string
	}{
		{name: "all, no type filter, no page", orderType: "all", typeID: 0, page: 0, wantQuery: "order_type=all"},
		{name: "buy with type filter and page", orderType: "buy", typeID: 34, page: 2, wantQuery: "order_type=buy&page=2&type_id=34"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Pages", "5")
				w.Write([]byte(`[{
					"duration": 90,
					"is_buy_order": true,
					"issued": "2026-07-20T00:00:00Z",
					"location_id": 60003760,
					"min_volume": 1,
					"order_id": 555,
					"price": 5.55,
					"range": "region",
					"system_id": 30000142,
					"type_id": 34,
					"volume_remain": 500,
					"volume_total": 1000
				}]`))
			}))
			defer srv.Close()

			client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

			orders, totalPages, err := client.MarketOrders(context.Background(), 10000002, tt.orderType, tt.typeID, tt.page)
			if err != nil {
				t.Fatalf("MarketOrders returned error: %v", err)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tt.wantQuery)
			}
			if totalPages != 5 {
				t.Errorf("totalPages = %d, want 5", totalPages)
			}
			if len(orders) != 1 {
				t.Fatalf("orders = %+v, want one entry", orders)
			}
			o := orders[0]
			wantIssued := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
			if o.Duration != 90 || !o.IsBuyOrder || !o.Issued.Equal(wantIssued) || o.LocationId != 60003760 ||
				o.MinVolume != 1 || o.OrderId != 555 || o.Price != 5.55 || o.Range != "region" ||
				o.SystemId != 30000142 || o.TypeId != 34 || o.VolumeRemain != 500 || o.VolumeTotal != 1000 {
				t.Errorf("orders[0] = %+v, want Duration=90 IsBuyOrder=true Issued=%v LocationId=60003760 MinVolume=1 OrderId=555 Price=5.55 Range=region SystemId=30000142 TypeId=34 VolumeRemain=500 VolumeTotal=1000",
					o, wantIssued)
			}
		})
	}
}

func TestMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.MarketOrders(context.Background(), 10000002, "all", 0, 0); err == nil {
		t.Fatal("MarketOrders returned nil error, want an error for HTTP 404")
	}
}

func TestMarketTypes_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "7")
		w.Write([]byte(`[34, 35, 36]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	types, totalPages, err := client.MarketTypes(context.Background(), 10000002, 3)
	if err != nil {
		t.Fatalf("MarketTypes returned error: %v", err)
	}
	if gotQuery != "page=3" {
		t.Errorf("query = %q, want \"page=3\"", gotQuery)
	}
	if totalPages != 7 {
		t.Errorf("totalPages = %d, want 7", totalPages)
	}
	if len(types) != 3 || types[0] != 34 || types[1] != 35 || types[2] != 36 {
		t.Errorf("types = %+v, want [34 35 36]", types)
	}
}

func TestMarketTypes_PageOmittedWhenZero(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.MarketTypes(context.Background(), 10000002, 0); err != nil {
		t.Fatalf("MarketTypes returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page=0 should omit the query param)", gotQuery)
	}
}

func TestMarketTypes_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.MarketTypes(context.Background(), 10000002, 0); err == nil {
		t.Fatal("MarketTypes returned nil error, want an error for HTTP 404")
	}
}

func TestStructureMarketOrders_ParsesResponseAndPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{
			"duration": 90,
			"is_buy_order": false,
			"issued": "2026-07-20T00:00:00Z",
			"location_id": 1000000000001,
			"min_volume": 1,
			"order_id": 777,
			"price": 12.5,
			"range": "station",
			"type_id": 34,
			"volume_remain": 250,
			"volume_total": 1000
		}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	orders, totalPages, err := client.StructureMarketOrders(context.Background(), 1000000000001, 1)
	if err != nil {
		t.Fatalf("StructureMarketOrders returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want \"page=1\"", gotQuery)
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %+v, want one entry", orders)
	}
	o := orders[0]
	wantIssued := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if o.Duration != 90 || o.IsBuyOrder || !o.Issued.Equal(wantIssued) || o.LocationId != 1000000000001 ||
		o.MinVolume != 1 || o.OrderId != 777 || o.Price != 12.5 || o.Range != "station" ||
		o.TypeId != 34 || o.VolumeRemain != 250 || o.VolumeTotal != 1000 {
		t.Errorf("orders[0] = %+v, want Duration=90 IsBuyOrder=false Issued=%v LocationId=1000000000001 MinVolume=1 OrderId=777 Price=12.5 Range=station TypeId=34 VolumeRemain=250 VolumeTotal=1000",
			o, wantIssued)
	}
}

func TestStructureMarketOrders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.StructureMarketOrders(context.Background(), 1000000000001, 0); err == nil {
		t.Fatal("StructureMarketOrders returned nil error, want an error for HTTP 403 (no docking access)")
	}
}
