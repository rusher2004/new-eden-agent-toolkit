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

func TestCorporationProjectContribution_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"contributed":1000,"last_modified":"2026-07-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contribution, err := client.CorporationProjectContribution(context.Background(), 98000001, "abc-123", 998878)
	if err != nil {
		t.Fatalf("CorporationProjectContribution returned error: %v", err)
	}
	if contribution.Contributed != 1000 {
		t.Errorf("Contributed = %d, want 1000", contribution.Contributed)
	}
	if contribution.LastModified == nil {
		t.Fatal("LastModified = nil, want non-nil")
	}
	if got, want := contribution.LastModified.Format("2006-01-02T15:04:05Z"), "2026-07-01T00:00:00Z"; got != want {
		t.Errorf("LastModified = %q, want %q", got, want)
	}
}

func TestCorporationProjectContribution_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	_, err := client.CorporationProjectContribution(context.Background(), 98000001, "abc-123", 998878)
	if err == nil {
		t.Fatal("CorporationProjectContribution returned nil error, want an error for HTTP 403")
	}
}

func TestCorporationProjectContributors_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"contributors":[{"contributed":500,"id":998879,"name":"Some Pilot"}],"cursor":{"after":"cursor-after","before":"cursor-before"}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contributors, cursor, err := client.CorporationProjectContributors(context.Background(), 98000001, "abc-123", "", "", 0)
	if err != nil {
		t.Fatalf("CorporationProjectContributors returned error: %v", err)
	}
	if len(contributors) != 1 {
		t.Fatalf("contributors = %+v, want one entry", contributors)
	}
	c := contributors[0]
	if c.Contributed != 500 || c.Id != 998879 || c.Name != "Some Pilot" {
		t.Errorf("contributor = %+v, want {Contributed:500 Id:998879 Name:\"Some Pilot\"}", c)
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

func TestCorporationProjectContributors_CursorOmittedWhenEmpty(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"contributors":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationProjectContributors(context.Background(), 98000001, "abc-123", "", "", 0); err != nil {
		t.Fatalf("CorporationProjectContributors returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (after/before/limit should all be omitted)", gotQuery)
	}
}

func TestCorporationProjectContributors_CursorSentWhenSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"contributors":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationProjectContributors(context.Background(), 98000001, "abc-123", "abc123", "", 0); err != nil {
		t.Fatalf("CorporationProjectContributors returned error: %v", err)
	}
	if gotQuery != "after=abc123" {
		t.Errorf("query = %q, want \"after=abc123\"", gotQuery)
	}
}

func TestCorporationProjectDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"configuration": {
				"capture_fw_complex": {
					"archetypes": [{"archetype_id": 1}],
					"factions": [{"faction_id": 500001}],
					"locations": [{"constellation_id": 20000001}]
				}
			},
			"contribution": {
				"participation_limit": 10,
				"reward_per_contribution": 1.5,
				"submission_limit": 100,
				"submission_multiplier": 2.0
			},
			"creator": {"id": 998877, "name": "Creator Pilot"},
			"details": {
				"career": "Exploration",
				"created": "2026-06-01T00:00:00Z",
				"description": "A test project"
			},
			"id": "abc-123",
			"last_modified": "2026-07-01T00:00:00Z",
			"name": "Test Project",
			"progress": {"current": 10, "desired": 100},
			"reward": {"initial": 1000000.0, "remaining": 500000.0},
			"state": "Active"
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	detail, err := client.CorporationProjectDetail(context.Background(), 98000001, "abc-123")
	if err != nil {
		t.Fatalf("CorporationProjectDetail returned error: %v", err)
	}

	if detail.Configuration.CaptureFwComplex == nil || detail.Configuration.CaptureFwComplex.CaptureFwComplex == nil {
		t.Fatal("Configuration.CaptureFwComplex.CaptureFwComplex = nil, want non-nil")
	}
	cfc := detail.Configuration.CaptureFwComplex.CaptureFwComplex
	if len(cfc.Archetypes) != 1 || cfc.Archetypes[0].ArchetypeId == nil || *cfc.Archetypes[0].ArchetypeId != 1 {
		t.Errorf("Archetypes = %+v, want one entry with ArchetypeId=1", cfc.Archetypes)
	}
	if len(cfc.Factions) != 1 || cfc.Factions[0].FactionId == nil || *cfc.Factions[0].FactionId != 500001 {
		t.Errorf("Factions = %+v, want one entry with FactionId=500001", cfc.Factions)
	}
	if len(cfc.Locations) != 1 || cfc.Locations[0].ConstellationId == nil || cfc.Locations[0].ConstellationId.ConstellationId == nil || *cfc.Locations[0].ConstellationId.ConstellationId != 20000001 {
		t.Errorf("Locations = %+v, want one entry with ConstellationId=20000001", cfc.Locations)
	}

	if detail.Contribution == nil {
		t.Fatal("Contribution = nil, want non-nil")
	}
	if detail.Contribution.ParticipationLimit == nil || *detail.Contribution.ParticipationLimit != 10 {
		t.Errorf("Contribution.ParticipationLimit = %v, want 10", detail.Contribution.ParticipationLimit)
	}
	if detail.Contribution.RewardPerContribution == nil || *detail.Contribution.RewardPerContribution != 1.5 {
		t.Errorf("Contribution.RewardPerContribution = %v, want 1.5", detail.Contribution.RewardPerContribution)
	}
	if detail.Contribution.SubmissionLimit == nil || *detail.Contribution.SubmissionLimit != 100 {
		t.Errorf("Contribution.SubmissionLimit = %v, want 100", detail.Contribution.SubmissionLimit)
	}
	if detail.Contribution.SubmissionMultiplier == nil || *detail.Contribution.SubmissionMultiplier != 2.0 {
		t.Errorf("Contribution.SubmissionMultiplier = %v, want 2.0", detail.Contribution.SubmissionMultiplier)
	}

	if detail.Creator.Id != 998877 || detail.Creator.Name != "Creator Pilot" {
		t.Errorf("Creator = %+v, want {Id:998877 Name:\"Creator Pilot\"}", detail.Creator)
	}

	if detail.Details.Career != "Exploration" || detail.Details.Description != "A test project" {
		t.Errorf("Details = %+v, want Career=\"Exploration\" Description=\"A test project\"", detail.Details)
	}

	if detail.Id != "abc-123" {
		t.Errorf("Id = %q, want \"abc-123\"", detail.Id)
	}
	if got, want := detail.LastModified.Format("2006-01-02T15:04:05Z"), "2026-07-01T00:00:00Z"; got != want {
		t.Errorf("LastModified = %q, want %q", got, want)
	}
	if detail.Name != "Test Project" {
		t.Errorf("Name = %q, want \"Test Project\"", detail.Name)
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

func TestCorporationProjectListing_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"projects": [{
				"id": "proj-1",
				"last_modified": "2026-07-01T00:00:00Z",
				"name": "Test Project",
				"progress": {"current": 5, "desired": 50},
				"reward": {"initial": 200000.0, "remaining": 150000.0},
				"state": "Active"
			}],
			"cursor": {"after": "cursor-after", "before": "cursor-before"}
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	projects, cursor, err := client.CorporationProjectListing(context.Background(), 98000001, "", "", 0, "")
	if err != nil {
		t.Fatalf("CorporationProjectListing returned error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("projects = %+v, want one entry", projects)
	}
	p := projects[0]
	if p.Id != "proj-1" {
		t.Errorf("Id = %q, want \"proj-1\"", p.Id)
	}
	if got, want := p.LastModified.Format("2006-01-02T15:04:05Z"), "2026-07-01T00:00:00Z"; got != want {
		t.Errorf("LastModified = %q, want %q", got, want)
	}
	if p.Name != "Test Project" {
		t.Errorf("Name = %q, want \"Test Project\"", p.Name)
	}
	if p.Progress.Current != 5 || p.Progress.Desired != 50 {
		t.Errorf("Progress = %+v, want {Current:5 Desired:50}", p.Progress)
	}
	if p.Reward == nil || p.Reward.Initial != 200000.0 || p.Reward.Remaining != 150000.0 {
		t.Errorf("Reward = %+v, want {Initial:200000 Remaining:150000}", p.Reward)
	}
	if p.State != "Active" {
		t.Errorf("State = %q, want \"Active\"", p.State)
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

func TestCorporationProjectListing_CursorAndStateOmittedWhenEmpty(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"projects":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationProjectListing(context.Background(), 98000001, "", "", 0, ""); err != nil {
		t.Fatalf("CorporationProjectListing returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (after/before/limit/state should all be omitted)", gotQuery)
	}
}

func TestCorporationProjectListing_CursorAndStateSentWhenSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"projects":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, _, err := client.CorporationProjectListing(context.Background(), 98000001, "", "xyz", 5, "Completed"); err != nil {
		t.Fatalf("CorporationProjectListing returned error: %v", err)
	}

	gotValues, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("failed to parse query %q: %v", gotQuery, err)
	}
	if len(gotValues) != 3 {
		t.Fatalf("query = %q, want exactly 3 params (before, limit, state)", gotQuery)
	}
	if gotValues.Get("before") != "xyz" {
		t.Errorf("before = %q, want \"xyz\"", gotValues.Get("before"))
	}
	if gotValues.Get("limit") != "5" {
		t.Errorf("limit = %q, want \"5\"", gotValues.Get("limit"))
	}
	if gotValues.Get("state") != "Completed" {
		t.Errorf("state = %q, want \"Completed\"", gotValues.Get("state"))
	}
	if gotValues.Get("after") != "" {
		t.Errorf("after = %q, want empty (after was not set)", gotValues.Get("after"))
	}
}
