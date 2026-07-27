package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterSearch_SendsQueryParamsAndParsesResponse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"character":[90000001,90000002],"corporation":[98000001]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	results, err := client.CharacterSearch(context.Background(), []string{"character", "corporation"}, "wanted", true)
	if err != nil {
		t.Fatalf("CharacterSearch returned error: %v", err)
	}
	values, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("failed to parse query %q: %v", gotQuery, err)
	}
	if got := values["categories"]; len(got) != 2 || got[0] != "character" || got[1] != "corporation" {
		t.Errorf("query categories = %v, want [character corporation]", got)
	}
	if got := values.Get("search"); got != "wanted" {
		t.Errorf("query search = %q, want \"wanted\"", got)
	}
	if got := values.Get("strict"); got != "true" {
		t.Errorf("query strict = %q, want \"true\"", got)
	}
	if len(results.Character) != 2 || results.Character[0] != 90000001 || results.Character[1] != 90000002 {
		t.Errorf("results.Character = %+v, want [90000001 90000002]", results.Character)
	}
	if len(results.Corporation) != 1 || results.Corporation[0] != 98000001 {
		t.Errorf("results.Corporation = %+v, want [98000001]", results.Corporation)
	}
}

func TestCharacterSearch_OmitsStrictWhenFalse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterSearch(context.Background(), []string{"character"}, "wanted", false); err != nil {
		t.Fatalf("CharacterSearch returned error: %v", err)
	}
	if strictPresent := queryHasKey(gotQuery, "strict"); strictPresent {
		t.Error("strict query param present, want it omitted when strict=false")
	}
}

func queryHasKey(rawQuery, key string) bool {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false
	}
	_, ok := values[key]
	return ok
}

func TestCharacterSearch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterSearch(context.Background(), []string{"character"}, "wanted", false); err == nil {
		t.Fatal("CharacterSearch returned nil error, want an error for HTTP 404")
	}
}
