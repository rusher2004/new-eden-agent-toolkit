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

func TestCharacterFreelanceJobs_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"freelance_jobs":[{"id":"job-1","last_modified":"2026-07-01T00:00:00Z","name":"Test Job","progress":{"current":5,"desired":10},"reward":{"initial":100000.0,"remaining":50000.0},"state":"Active"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	jobs, err := client.CharacterFreelanceJobs(context.Background())
	if err != nil {
		t.Fatalf("CharacterFreelanceJobs returned error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs = %+v, want one entry", jobs)
	}
	j := jobs[0]
	if j.Id != "job-1" {
		t.Errorf("Id = %q, want \"job-1\"", j.Id)
	}
	if got, want := j.LastModified.Format("2006-01-02T15:04:05Z"), "2026-07-01T00:00:00Z"; got != want {
		t.Errorf("LastModified = %q, want %q", got, want)
	}
	if j.Name != "Test Job" {
		t.Errorf("Name = %q, want \"Test Job\"", j.Name)
	}
	if j.Progress.Current != 5 || j.Progress.Desired != 10 {
		t.Errorf("Progress = %+v, want {Current:5 Desired:10}", j.Progress)
	}
	if j.Reward == nil || j.Reward.Initial != 100000.0 || j.Reward.Remaining != 50000.0 {
		t.Errorf("Reward = %+v, want {Initial:100000 Remaining:50000}", j.Reward)
	}
	if j.State != "Active" {
		t.Errorf("State = %q, want \"Active\"", j.State)
	}
	if gotPath != "/characters/998877/freelance-jobs" {
		t.Errorf("path = %q, want it to reflect the client's own characterID 998877", gotPath)
	}
}

func TestCharacterFreelanceJobs_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	_, err := client.CharacterFreelanceJobs(context.Background())
	if err == nil {
		t.Fatal("CharacterFreelanceJobs returned nil error, want an error for HTTP 404")
	}
}

func TestCharacterFreelanceJobParticipation_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"contributed":250,"last_modified":"2026-07-01T00:00:00Z","state":"Contributing"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	participation, err := client.CharacterFreelanceJobParticipation(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("CharacterFreelanceJobParticipation returned error: %v", err)
	}
	if participation.Contributed != 250 {
		t.Errorf("Contributed = %d, want 250", participation.Contributed)
	}
	if got, want := participation.LastModified.Format("2006-01-02T15:04:05Z"), "2026-07-01T00:00:00Z"; got != want {
		t.Errorf("LastModified = %q, want %q", got, want)
	}
	if participation.State != "Contributing" {
		t.Errorf("State = %q, want \"Contributing\"", participation.State)
	}
}

func TestCorporationFreelanceJobs_ParsesResponseAndCursorOmittedWhenEmpty(t *testing.T) {
	t.Run("ParsesResponse", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"freelance_jobs":[{"id":"job-2","last_modified":"2026-07-01T00:00:00Z","name":"Corp Job","progress":{"current":1,"desired":2},"reward":{"initial":1.0,"remaining":2.0},"state":"Active"}],"cursor":{"after":"cursor-after","before":"cursor-before"}}`))
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

		jobs, cursor, err := client.CorporationFreelanceJobs(context.Background(), 98000001, "", "", 0)
		if err != nil {
			t.Fatalf("CorporationFreelanceJobs returned error: %v", err)
		}
		if len(jobs) != 1 {
			t.Fatalf("jobs = %+v, want one entry", jobs)
		}
		j := jobs[0]
		if j.Id != "job-2" || j.Name != "Corp Job" || j.State != "Active" {
			t.Errorf("job = %+v, want {Id:\"job-2\" Name:\"Corp Job\" State:\"Active\"}", j)
		}
		if j.Progress.Current != 1 || j.Progress.Desired != 2 {
			t.Errorf("Progress = %+v, want {Current:1 Desired:2}", j.Progress)
		}
		if j.Reward == nil || j.Reward.Initial != 1.0 || j.Reward.Remaining != 2.0 {
			t.Errorf("Reward = %+v, want {Initial:1 Remaining:2}", j.Reward)
		}
		if cursor == nil {
			t.Fatal("cursor = nil, want non-nil")
		}
		if cursor.After == nil || *cursor.After != "cursor-after" {
			t.Errorf("cursor.After = %v, want \"cursor-after\"", cursor.After)
		}
		if cursor.Before == nil || *cursor.Before != "cursor-before" {
			t.Errorf("cursor.Before = %v, want \"cursor-before\"", cursor.Before)
		}
	})

	t.Run("CursorOmittedWhenEmpty", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"freelance_jobs":[]}`))
		}))
		defer srv.Close()

		client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

		if _, _, err := client.CorporationFreelanceJobs(context.Background(), 98000001, "", "", 0); err != nil {
			t.Fatalf("CorporationFreelanceJobs returned error: %v", err)
		}
		if gotQuery != "" {
			t.Errorf("query = %q, want empty (after/before/limit should all be omitted)", gotQuery)
		}
	})
}

func TestCorporationFreelanceJobs_CursorSentWhenSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"freelance_jobs":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationFreelanceJobs(context.Background(), 98000001, "abc", "", 0); err != nil {
		t.Fatalf("CorporationFreelanceJobs returned error: %v", err)
	}
	if gotQuery != "after=abc" {
		t.Errorf("query = %q, want \"after=abc\"", gotQuery)
	}
}

func TestCorporationFreelanceJobParticipants_ParsesResponseAndCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"cursor":{"after":"cursor-after","before":"cursor-before"},"participants":[{"contributed":75,"id":998880,"name":"Some Participant","state":"Contributing"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	participants, cursor, err := client.CorporationFreelanceJobParticipants(context.Background(), 98000001, "job-2", "", "", 0)
	if err != nil {
		t.Fatalf("CorporationFreelanceJobParticipants returned error: %v", err)
	}
	if len(participants) != 1 {
		t.Fatalf("participants = %+v, want one entry", participants)
	}
	p := participants[0]
	if p.Contributed != 75 || p.Id != 998880 || p.Name != "Some Participant" || p.State != "Contributing" {
		t.Errorf("participant = %+v, want {Contributed:75 Id:998880 Name:\"Some Participant\" State:\"Contributing\"}", p)
	}
	if cursor == nil {
		t.Fatal("cursor = nil, want non-nil")
	}
	if cursor.After == nil || *cursor.After != "cursor-after" {
		t.Errorf("cursor.After = %v, want \"cursor-after\"", cursor.After)
	}
	if cursor.Before == nil || *cursor.Before != "cursor-before" {
		t.Errorf("cursor.Before = %v, want \"cursor-before\"", cursor.Before)
	}
}

func TestFreelanceJobDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"access_and_visibility": {"acl_protected": true},
			"configuration": {
				"method": "some_method",
				"version": 1,
				"parameters": {
					"boolean_param": {"boolean": {"value": true}},
					"corp_item_param": {
						"corporation_item_delivery": {
							"corporation_office_location": {"values": [{"value_type": "station_id", "values": ["60003760"]}]},
							"item_type": {"values": [{"value_type": "type_id", "values": ["34"]}]}
						}
					},
					"matcher_param": {"matcher": {"values": [{"value_type": "region_id", "values": ["10000002"]}]}},
					"options_param": {"options": {"selected": ["option_a", "option_b"]}}
				}
			},
			"details": {
				"career": "Exploration",
				"created": "2026-07-01T00:00:00Z",
				"creator": {
					"character": {"id": 998877, "name": "Creator Pilot"},
					"corporation": {"id": 98000001, "name": "Creator Corp"}
				},
				"description": "A test freelance job"
			},
			"id": "job-123",
			"last_modified": "2026-07-02T00:00:00Z",
			"name": "Test Freelance Job",
			"progress": {"current": 10, "desired": 100},
			"reward": {"initial": 1000000.0, "remaining": 500000.0},
			"state": "Active"
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	detail, err := client.FreelanceJobDetail(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("FreelanceJobDetail returned error: %v", err)
	}

	if !detail.AccessAndVisibility.AclProtected {
		t.Errorf("AccessAndVisibility.AclProtected = false, want true")
	}

	if detail.Configuration.Method != "some_method" {
		t.Errorf("Configuration.Method = %q, want \"some_method\"", detail.Configuration.Method)
	}
	if detail.Configuration.Version != 1 {
		t.Errorf("Configuration.Version = %d, want 1", detail.Configuration.Version)
	}
	params := detail.Configuration.Parameters
	if len(params) != 4 {
		t.Fatalf("Configuration.Parameters = %+v, want 4 entries", params)
	}

	boolParam, ok := params["boolean_param"]
	if !ok || boolParam.Boolean == nil || boolParam.Boolean.Boolean == nil {
		t.Fatalf("params[\"boolean_param\"] = %+v, want a populated Boolean variant", boolParam)
	}
	if !boolParam.Boolean.Boolean.Value {
		t.Errorf("boolean_param.Boolean.Boolean.Value = false, want true")
	}

	corpItemParam, ok := params["corp_item_param"]
	if !ok || corpItemParam.CorporationItemDelivery == nil || corpItemParam.CorporationItemDelivery.CorporationItemDelivery == nil {
		t.Fatalf("params[\"corp_item_param\"] = %+v, want a populated CorporationItemDelivery variant", corpItemParam)
	}
	cid := corpItemParam.CorporationItemDelivery.CorporationItemDelivery
	if len(cid.CorporationOfficeLocation.Values) != 1 || cid.CorporationOfficeLocation.Values[0].ValueType != "station_id" || len(cid.CorporationOfficeLocation.Values[0].Values) != 1 || cid.CorporationOfficeLocation.Values[0].Values[0] != "60003760" {
		t.Errorf("CorporationOfficeLocation = %+v, want value_type=station_id values=[60003760]", cid.CorporationOfficeLocation)
	}
	if len(cid.ItemType.Values) != 1 || cid.ItemType.Values[0].ValueType != "type_id" || len(cid.ItemType.Values[0].Values) != 1 || cid.ItemType.Values[0].Values[0] != "34" {
		t.Errorf("ItemType = %+v, want value_type=type_id values=[34]", cid.ItemType)
	}

	matcherParam, ok := params["matcher_param"]
	if !ok || matcherParam.Matcher == nil || matcherParam.Matcher.Matcher == nil {
		t.Fatalf("params[\"matcher_param\"] = %+v, want a populated Matcher variant", matcherParam)
	}
	m := matcherParam.Matcher.Matcher
	if len(m.Values) != 1 || m.Values[0].ValueType != "region_id" || len(m.Values[0].Values) != 1 || m.Values[0].Values[0] != "10000002" {
		t.Errorf("Matcher.Values = %+v, want value_type=region_id values=[10000002]", m.Values)
	}

	optionsParam, ok := params["options_param"]
	if !ok || optionsParam.Options == nil || optionsParam.Options.Options == nil {
		t.Fatalf("params[\"options_param\"] = %+v, want a populated Options variant", optionsParam)
	}
	selected := optionsParam.Options.Options.Selected
	if len(selected) != 2 || selected[0] != "option_a" || selected[1] != "option_b" {
		t.Errorf("Options.Selected = %+v, want [option_a option_b]", selected)
	}

	if detail.Details.Career != "Exploration" {
		t.Errorf("Details.Career = %q, want \"Exploration\"", detail.Details.Career)
	}
	if got, want := detail.Details.Created.Format("2006-01-02T15:04:05Z"), "2026-07-01T00:00:00Z"; got != want {
		t.Errorf("Details.Created = %q, want %q", got, want)
	}
	if detail.Details.Description != "A test freelance job" {
		t.Errorf("Details.Description = %q, want \"A test freelance job\"", detail.Details.Description)
	}
	if detail.Details.Creator.Character.Id != 998877 || detail.Details.Creator.Character.Name != "Creator Pilot" {
		t.Errorf("Details.Creator.Character = %+v, want {Id:998877 Name:\"Creator Pilot\"}", detail.Details.Creator.Character)
	}
	if detail.Details.Creator.Corporation.Id != 98000001 || detail.Details.Creator.Corporation.Name != "Creator Corp" {
		t.Errorf("Details.Creator.Corporation = %+v, want {Id:98000001 Name:\"Creator Corp\"}", detail.Details.Creator.Corporation)
	}

	if detail.Id != "job-123" {
		t.Errorf("Id = %q, want \"job-123\"", detail.Id)
	}
	if got, want := detail.LastModified.Format("2006-01-02T15:04:05Z"), "2026-07-02T00:00:00Z"; got != want {
		t.Errorf("LastModified = %q, want %q", got, want)
	}
	if detail.Name != "Test Freelance Job" {
		t.Errorf("Name = %q, want \"Test Freelance Job\"", detail.Name)
	}
	if detail.Progress.Current != 10 || detail.Progress.Desired != 100 {
		t.Errorf("Progress = %+v, want {Current:10 Desired:100}", detail.Progress)
	}
	if detail.Reward == nil || detail.Reward.Initial != 1000000.0 || detail.Reward.Remaining != 500000.0 {
		t.Errorf("Reward = %+v, want {Initial:1000000 Remaining:500000}", detail.Reward)
	}
	if detail.State != "Active" {
		t.Errorf("State = %q, want \"Active\"", detail.State)
	}
}

func TestFreelanceJobsListing_CursorAndCorporationFilterOmittedWhenEmpty(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"freelance_jobs":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.FreelanceJobsListing(context.Background(), "", "", 0, 0); err != nil {
		t.Fatalf("FreelanceJobsListing returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (after/before/limit/corporation_id should all be omitted)", gotQuery)
	}
}

func TestFreelanceJobsListing_CursorAndCorporationFilterSentWhenSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"freelance_jobs":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.FreelanceJobsListing(context.Background(), "", "xyz", 5, 98000001); err != nil {
		t.Fatalf("FreelanceJobsListing returned error: %v", err)
	}

	gotValues, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("failed to parse query %q: %v", gotQuery, err)
	}
	if len(gotValues) != 3 {
		t.Fatalf("query = %q, want exactly 3 params (before, limit, corporation_id)", gotQuery)
	}
	if gotValues.Get("before") != "xyz" {
		t.Errorf("before = %q, want \"xyz\"", gotValues.Get("before"))
	}
	if gotValues.Get("limit") != "5" {
		t.Errorf("limit = %q, want \"5\"", gotValues.Get("limit"))
	}
	if gotValues.Get("corporation_id") != "98000001" {
		t.Errorf("corporation_id = %q, want \"98000001\"", gotValues.Get("corporation_id"))
	}
	if gotValues.Get("after") != "" {
		t.Errorf("after = %q, want empty (after was not set)", gotValues.Get("after"))
	}
}
