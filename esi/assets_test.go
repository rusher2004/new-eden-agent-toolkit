package esi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestAssets_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "3")
		w.Write([]byte(`[{"is_singleton":true,"item_id":1,"location_flag":"Hangar","location_id":60003760,"location_type":"station","quantity":1,"type_id":587}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	assets, totalPages, err := client.Assets(context.Background(), 2)
	if err != nil {
		t.Fatalf("Assets returned error: %v", err)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want page=2", gotQuery)
	}
	if len(assets) != 1 || assets[0].ItemId != 1 || assets[0].TypeId != 587 || totalPages != 3 {
		t.Errorf("assets = %+v, totalPages = %d, want one entry with ItemId=1 TypeId=587 totalPages=3", assets, totalPages)
	}
}

func TestAssets_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, _, err := client.Assets(context.Background(), 0); err != nil {
		t.Fatalf("Assets returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page omitted)", gotQuery)
	}
}

func TestAssetLocations_SendsItemIDsAsBareJSONArray(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":1,"position":{"x":1.5,"y":2.5,"z":3.5}}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	locs, err := client.AssetLocations(context.Background(), []int64{1, 2, 3})
	if err != nil {
		t.Fatalf("AssetLocations returned error: %v", err)
	}
	// The body may have a trailing newline from the JSON encoder
	if gotBody != "[1,2,3]" && gotBody != "[1,2,3]\n" {
		t.Errorf("request body = %q, want \"[1,2,3]\" or \"[1,2,3]\\n\" (bare array, not wrapped)", gotBody)
	}
	if len(locs) != 1 || locs[0].ItemId != 1 || locs[0].Position.X != 1.5 {
		t.Errorf("locs = %+v, want one entry with ItemId=1 Position.X=1.5", locs)
	}
}

func TestAssetLocations_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.AssetLocations(context.Background(), []int64{1}); err == nil {
		t.Fatal("AssetLocations returned nil error, want an error for HTTP 403")
	}
}

func TestAssetNames_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":1,"name":"My Renamed Ship"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	names, err := client.AssetNames(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("AssetNames returned error: %v", err)
	}
	if len(names) != 1 || names[0].ItemId != 1 || names[0].Name != "My Renamed Ship" {
		t.Errorf("names = %+v, want one entry {ItemId:1 Name:\"My Renamed Ship\"}", names)
	}
}

func TestCorporationAssets_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"is_singleton":false,"item_id":2,"location_flag":"CorpDeliveries","location_id":60003760,"location_type":"station","quantity":10,"type_id":34}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	assets, totalPages, err := client.CorporationAssets(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationAssets returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want page=1", gotQuery)
	}
	if len(assets) != 1 || assets[0].ItemId != 2 || assets[0].Quantity != 10 || totalPages != 2 {
		t.Errorf("assets = %+v, totalPages = %d, want one entry with ItemId=2 Quantity=10 totalPages=2", assets, totalPages)
	}
}

func TestCorporationAssets_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, _, err := client.CorporationAssets(context.Background(), 98000001, 0); err != nil {
		t.Fatalf("CorporationAssets returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page omitted)", gotQuery)
	}
}

func TestCorporationAssetLocations_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":2,"position":{"x":4,"y":5,"z":6}}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	locs, err := client.CorporationAssetLocations(context.Background(), 98000001, []int64{2})
	if err != nil {
		t.Fatalf("CorporationAssetLocations returned error: %v", err)
	}
	if len(locs) != 1 || locs[0].ItemId != 2 {
		t.Errorf("locs = %+v, want one entry with ItemId=2", locs)
	}
}

func TestCorporationAssetNames_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"item_id":2,"name":"Corp Hauler"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	names, err := client.CorporationAssetNames(context.Background(), 98000001, []int64{2})
	if err != nil {
		t.Fatalf("CorporationAssetNames returned error: %v", err)
	}
	if len(names) != 1 || names[0].Name != "Corp Hauler" {
		t.Errorf("names = %+v, want one entry with Name=\"Corp Hauler\"", names)
	}
}
