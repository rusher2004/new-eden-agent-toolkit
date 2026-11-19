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

// Payloads below are trimmed from real ESI responses (compatibility date 2026-08-18).

func TestMilitaryCampaigns_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"campaigns":[{"id":"7519d7db-1e95-4d0e-bbbe-c47cd47de3c0","state":"Completed","progress":30,"started":"2026-06-09T11:00:09.069Z","finished":"2026-09-16T10:24:53.154Z"},{"id":"37448f11-cd11-404c-adb5-b1f7ae4aa728","state":"Active","progress":12}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	campaigns, err := client.MilitaryCampaigns(context.Background())
	if err != nil {
		t.Fatalf("MilitaryCampaigns returned error: %v", err)
	}
	if gotPath != "/military-campaigns" {
		t.Errorf("path = %q, want /military-campaigns", gotPath)
	}
	if len(campaigns) != 2 {
		t.Fatalf("campaigns = %+v, want 2 entries", campaigns)
	}
	if c := campaigns[0]; c.Id != "7519d7db-1e95-4d0e-bbbe-c47cd47de3c0" || c.State != "Completed" || c.Progress != 30 || c.Started == nil || c.Finished == nil {
		t.Errorf("campaigns[0] = %+v, want completed campaign with start and finish times", c)
	}
	if c := campaigns[1]; c.State != "Active" || c.Started != nil || c.Finished != nil {
		t.Errorf("campaigns[1] = %+v, want active campaign with no start/finish times", c)
	}
}

func TestMilitaryCampaigns_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.MilitaryCampaigns(context.Background()); err == nil {
		t.Fatal("MilitaryCampaigns returned nil error, want an error for HTTP 500")
	}
}

func TestMilitaryCampaignDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"camp-1","state":"Active","progress":12,"started":"2026-11-10T11:00:00Z"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	c, err := client.MilitaryCampaignDetail(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("MilitaryCampaignDetail returned error: %v", err)
	}
	if gotPath != "/military-campaigns/camp-1" {
		t.Errorf("path = %q, want /military-campaigns/camp-1", gotPath)
	}
	if c.Id != "camp-1" || c.State != "Active" || c.Progress != 12 || c.Started == nil {
		t.Errorf("campaign = %+v, want Id=camp-1 State=Active Progress=12 with a start time", c)
	}
}

func TestMilitaryCampaignObjectives_ParsesResponseAndSendsCursorParams(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"cursor":{"before":"1.GOlG3gztl7C6AQ==","after":"1.GOlJhWOmT1i8AQ=="},"objectives":[{"id":"823b4bef-aed3-43d4-85ce-a4cef7788caf","state":"Active","progress":48700,"participants":{"total":667,"committed":585,"contributors":59},"last_modified":"2026-11-18T23:17:25.997Z","started":"2026-11-10T11:00:02.186Z"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	objectives, cursor, err := client.MilitaryCampaignObjectives(context.Background(), "camp-1", "cur-a", "", 25)
	if err != nil {
		t.Fatalf("MilitaryCampaignObjectives returned error: %v", err)
	}
	if gotPath != "/military-campaigns/camp-1/objectives" {
		t.Errorf("path = %q, want /military-campaigns/camp-1/objectives", gotPath)
	}
	if gotQuery.Get("after") != "cur-a" || gotQuery.Has("before") || gotQuery.Get("limit") != "25" {
		t.Errorf("query = %v, want after=cur-a limit=25 and no before", gotQuery)
	}
	if len(objectives) != 1 {
		t.Fatalf("objectives = %+v, want one entry", objectives)
	}
	o := objectives[0]
	if o.Id != "823b4bef-aed3-43d4-85ce-a4cef7788caf" || o.Progress != 48700 ||
		o.Participants.Total != 667 || o.Participants.Committed != 585 || o.Participants.Contributors != 59 {
		t.Errorf("objective = %+v, want progress 48700 and participants {667 585 59}", o)
	}
	if cursor == nil || cursor.After == nil || *cursor.After != "1.GOlJhWOmT1i8AQ==" {
		t.Errorf("cursor = %+v, want After=1.GOlJhWOmT1i8AQ==", cursor)
	}
}

func TestMilitaryCampaignObjectives_ZeroValuesOmitParams(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"objectives":[]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, _, err := client.MilitaryCampaignObjectives(context.Background(), "camp-1", "", "", 0); err != nil {
		t.Fatalf("MilitaryCampaignObjectives returned error: %v", err)
	}
	if gotQuery.Has("after") || gotQuery.Has("before") || gotQuery.Has("limit") {
		t.Errorf("query = %v, want no after/before/limit params", gotQuery)
	}
}

func TestMilitaryCampaignObjectiveDetail_ParsesResponse(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"obj-1","state":"Completed","progress":100,"participants":{"total":10,"committed":0,"contributors":8},"last_modified":"2026-11-01T00:00:00Z","started":"2026-10-01T00:00:00Z","finished":"2026-11-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	o, err := client.MilitaryCampaignObjectiveDetail(context.Background(), "camp-1", "obj-1")
	if err != nil {
		t.Fatalf("MilitaryCampaignObjectiveDetail returned error: %v", err)
	}
	if gotPath != "/military-campaigns/camp-1/objectives/obj-1" {
		t.Errorf("path = %q, want /military-campaigns/camp-1/objectives/obj-1", gotPath)
	}
	if o.Id != "obj-1" || o.State != "Completed" || o.Participants.Contributors != 8 || o.Finished == nil {
		t.Errorf("objective = %+v, want completed obj-1 with 8 contributors and a finish time", o)
	}
}

func TestCharacterMilitaryCampaignObjectives_UsesOwnCharacterID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"objectives":[{"id":"obj-1","campaign_id":"camp-1","contributed":42,"is_committed":true,"last_modified":"2026-11-01T00:00:00Z"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	objectives, cursor, err := client.CharacterMilitaryCampaignObjectives(context.Background(), "", "", 0)
	if err != nil {
		t.Fatalf("CharacterMilitaryCampaignObjectives returned error: %v", err)
	}
	if gotPath != "/characters/998877/military-campaigns/objectives" {
		t.Errorf("path = %q, want it to reflect the client's own characterID 998877", gotPath)
	}
	if len(objectives) != 1 || objectives[0].CampaignId != "camp-1" || objectives[0].Contributed != 42 || !objectives[0].IsCommitted {
		t.Errorf("objectives = %+v, want one committed entry for camp-1 with 42 contributed", objectives)
	}
	if cursor != nil {
		t.Errorf("cursor = %+v, want nil when ESI omits it", cursor)
	}
}

func TestCharacterMilitaryCampaignObjectiveParticipation_UsesOwnCharacterID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"obj-1","campaign_id":"camp-1","contributed":42,"is_committed":false,"last_modified":"2026-11-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	p, err := client.CharacterMilitaryCampaignObjectiveParticipation(context.Background(), "obj-1")
	if err != nil {
		t.Fatalf("CharacterMilitaryCampaignObjectiveParticipation returned error: %v", err)
	}
	if gotPath != "/characters/998877/military-campaigns/objectives/obj-1" {
		t.Errorf("path = %q, want /characters/998877/military-campaigns/objectives/obj-1", gotPath)
	}
	if p.Id != "obj-1" || p.CampaignId != "camp-1" || p.Contributed != 42 || p.IsCommitted {
		t.Errorf("participation = %+v, want obj-1/camp-1, 42 contributed, not committed", p)
	}
}

func TestCharacterMilitaryCampaignObjectiveParticipation_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.CharacterMilitaryCampaignObjectiveParticipation(context.Background(), "obj-1"); err == nil {
		t.Fatal("CharacterMilitaryCampaignObjectiveParticipation returned nil error, want an error for HTTP 403")
	}
}
