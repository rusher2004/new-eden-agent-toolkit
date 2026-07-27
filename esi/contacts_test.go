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

func TestCharacterContacts_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"contact_id":998878,"contact_type":"character","standing":5.5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contacts, totalPages, err := client.CharacterContacts(context.Background(), 1)
	if err != nil {
		t.Fatalf("CharacterContacts returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want page=1", gotQuery)
	}
	if len(contacts) != 1 || contacts[0].Standing != 5.5 || totalPages != 2 {
		t.Errorf("contacts = %+v, totalPages = %d, want one entry with Standing=5.5 totalPages=2", contacts, totalPages)
	}
}

func TestCharacterContacts_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CharacterContacts(context.Background(), 0); err != nil {
		t.Fatalf("CharacterContacts returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page omitted)", gotQuery)
	}
}

func TestCharacterContactLabels_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"label_id":1,"label_name":"Friends"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	labels, err := client.CharacterContactLabels(context.Background())
	if err != nil {
		t.Fatalf("CharacterContactLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].LabelName != "Friends" {
		t.Errorf("labels = %+v, want one entry with LabelName=\"Friends\"", labels)
	}
}

func TestCorporationContacts_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"contact_id":998879,"contact_type":"character","standing":1.0}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contacts, totalPages, err := client.CorporationContacts(context.Background(), 98000001, 1)
	if err != nil {
		t.Fatalf("CorporationContacts returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want page=1", gotQuery)
	}
	if len(contacts) != 1 || contacts[0].ContactId != 998879 || totalPages != 2 {
		t.Errorf("contacts = %+v, totalPages = %d, want one entry with ContactId=998879 totalPages=2", contacts, totalPages)
	}
}

func TestCorporationContacts_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationContacts(context.Background(), 98000001, 0); err != nil {
		t.Fatalf("CorporationContacts returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page omitted)", gotQuery)
	}
}

func TestCorporationContactLabels_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"label_id":2,"label_name":"Corp VIP"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	labels, err := client.CorporationContactLabels(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationContactLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].LabelName != "Corp VIP" {
		t.Errorf("labels = %+v, want one entry with LabelName=\"Corp VIP\"", labels)
	}
}

func TestAllianceContacts_ParsesResponseAndTotalPages(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "2")
		w.Write([]byte(`[{"contact_id":998880,"contact_type":"corporation","standing":-5.0}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contacts, totalPages, err := client.AllianceContacts(context.Background(), 99000001, 1)
	if err != nil {
		t.Fatalf("AllianceContacts returned error: %v", err)
	}
	if gotQuery != "page=1" {
		t.Errorf("query = %q, want page=1", gotQuery)
	}
	if len(contacts) != 1 || contacts[0].Standing != -5.0 || totalPages != 2 {
		t.Errorf("contacts = %+v, totalPages = %d, want one entry with Standing=-5.0 totalPages=2", contacts, totalPages)
	}
}

func TestAllianceContacts_PageZeroOmitsPageParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.AllianceContacts(context.Background(), 99000001, 0); err != nil {
		t.Fatalf("AllianceContacts returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (page omitted)", gotQuery)
	}
}

func TestAllianceContactLabels_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"label_id":3,"label_name":"Alliance Blues"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	labels, err := client.AllianceContactLabels(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceContactLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].LabelName != "Alliance Blues" {
		t.Errorf("labels = %+v, want one entry with LabelName=\"Alliance Blues\"", labels)
	}
}

func TestAddCharacterContacts_SendsCorrectQueryAndBody(t *testing.T) {
	var gotQuery, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[998878]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	added, err := client.AddCharacterContacts(context.Background(), []int64{998878}, 5.0, []int64{1}, true)
	if err != nil {
		t.Fatalf("AddCharacterContacts returned error: %v", err)
	}
	if gotQuery != "label_ids=1&standing=5&watched=true" {
		t.Errorf("query = %q, want \"label_ids=1&standing=5&watched=true\"", gotQuery)
	}
	if gotBody != "[998878]" && gotBody != "[998878]\n" {
		t.Errorf("body = %q, want \"[998878]\" (bare array, not wrapped)", gotBody)
	}
	if len(added) != 1 || added[0] != 998878 {
		t.Errorf("added = %v, want [998878]", added)
	}
}

func TestUpdateCharacterContacts_SendsCorrectQueryAndHandlesNoContent(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if err := client.UpdateCharacterContacts(context.Background(), []int64{998878}, 3.0, nil, false); err != nil {
		t.Fatalf("UpdateCharacterContacts returned error: %v", err)
	}
	if gotQuery != "standing=3" && gotQuery != "standing=3&watched=false" {
		t.Errorf("query = %q, want \"standing=3\" or \"standing=3&watched=false\" (library artifact)", gotQuery)
	}
}

func TestRemoveCharacterContacts_SendsContactIDsAsQueryParamNoBody(t *testing.T) {
	var gotQuery, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if err := client.RemoveCharacterContacts(context.Background(), []int64{998878}); err != nil {
		t.Fatalf("RemoveCharacterContacts returned error: %v", err)
	}
	if gotQuery != "contact_ids=998878" {
		t.Errorf("query = %q, want \"contact_ids=998878\"", gotQuery)
	}
	if gotBody != "" {
		t.Errorf("body = %q, want empty (DELETE has no body)", gotBody)
	}
}

func TestRemoveCharacterContacts_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if err := client.RemoveCharacterContacts(context.Background(), []int64{998878}); err == nil {
		t.Fatal("RemoveCharacterContacts returned nil error, want an error for HTTP 403")
	}
}
