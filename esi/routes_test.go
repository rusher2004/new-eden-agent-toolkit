package esi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestRoute_SendsRequestBodyAndParsesResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"route":[30000142,30000144,30000145]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	preference := "secure"
	securityPenalty := int64(1)
	connections := []esiapi.RouteConnection{{From: 30000142, To: 30000144}}
	route, err := client.Route(context.Background(), 30000142, 30000145, []int64{30000200}, connections, &preference, &securityPenalty)
	if err != nil {
		t.Fatalf("Route returned error: %v", err)
	}

	var got struct {
		AvoidSystems []int64 `json:"avoid_systems"`
		Connections  []struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"connections"`
		Preference      string `json:"preference"`
		SecurityPenalty int64  `json:"security_penalty"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if len(got.AvoidSystems) != 1 || got.AvoidSystems[0] != 30000200 {
		t.Errorf("request body AvoidSystems = %+v, want [30000200]", got.AvoidSystems)
	}
	if len(got.Connections) != 1 || got.Connections[0].From != 30000142 || got.Connections[0].To != 30000144 {
		t.Errorf("request body Connections = %+v, want one entry {From:30000142 To:30000144}", got.Connections)
	}
	if got.Preference != "secure" {
		t.Errorf("request body Preference = %q, want \"secure\"", got.Preference)
	}
	if got.SecurityPenalty != 1 {
		t.Errorf("request body SecurityPenalty = %d, want 1", got.SecurityPenalty)
	}

	if len(route) != 3 || route[0] != 30000142 || route[1] != 30000144 || route[2] != 30000145 {
		t.Errorf("route = %+v, want [30000142 30000144 30000145]", route)
	}
}

func TestRoute_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.Route(context.Background(), 30000142, 30000145, nil, nil, nil, nil); err == nil {
		t.Fatal("Route returned nil error, want an error for HTTP 404")
	}
}
