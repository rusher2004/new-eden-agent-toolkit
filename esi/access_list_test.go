package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestAccessLists_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_lists":[{"id":1},{"id":2}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	lists, err := client.AccessLists(context.Background())
	if err != nil {
		t.Fatalf("AccessLists returned error: %v", err)
	}
	if len(lists) != 2 || lists[0].Id != 1 || lists[1].Id != 2 {
		t.Errorf("lists = %+v, want [{Id:1} {Id:2}]", lists)
	}
}

func TestAccessListDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"description":"desc","id":1,"membership":{"alliances":[{"access":"read","alliance_id":99000001}],"allow_everyone":false,"characters":[{"access":"read","character_id":123}],"corporations":[{"access":"read","corporation_id":456}]},"name":"My List"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.AccessListDetail(context.Background(), 1)
	if err != nil {
		t.Fatalf("AccessListDetail returned error: %v", err)
	}
	if detail.Name != "My List" || detail.Description != "desc" {
		t.Errorf("detail = %+v, want Name=\"My List\" Description=\"desc\"", detail)
	}
	if len(detail.Membership.Alliances) != 1 || detail.Membership.Alliances[0].AllianceId != 99000001 {
		t.Errorf("detail.Membership.Alliances = %+v, want one entry with AllianceId 99000001", detail.Membership.Alliances)
	}
	if len(detail.Membership.Characters) != 1 || detail.Membership.Characters[0].CharacterId != 123 {
		t.Errorf("detail.Membership.Characters = %+v, want one entry with CharacterId 123", detail.Membership.Characters)
	}
	if len(detail.Membership.Corporations) != 1 || detail.Membership.Corporations[0].CorporationId != 456 {
		t.Errorf("detail.Membership.Corporations = %+v, want one entry with CorporationId 456", detail.Membership.Corporations)
	}
}

func TestAccessListDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.AccessListDetail(context.Background(), 1); err == nil {
		t.Fatal("AccessListDetail returned nil error, want an error for HTTP 403")
	}
}
