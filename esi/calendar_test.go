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

func TestCalendarEvents_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"event_date":"2026-08-01T00:00:00Z","event_id":456,"event_response":"needs_response","importance":1,"title":"Fleet Op"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	events, err := client.CalendarEvents(context.Background())
	if err != nil {
		t.Fatalf("CalendarEvents returned error: %v", err)
	}
	if len(events) != 1 || events[0].EventId == nil || *events[0].EventId != 456 {
		t.Errorf("events = %+v, want one entry with EventId=*456", events)
	}
	if events[0].Title == nil || *events[0].Title != "Fleet Op" {
		t.Errorf("events[0].Title = %v, want *\"Fleet Op\"", events[0].Title)
	}
}

func TestCalendarEventDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"date":"2026-08-01T00:00:00Z","duration":60,"event_id":456,"importance":1,"owner_id":98000001,"owner_name":"Test Corp","owner_type":"corporation","response":"needs_response","text":"Bring your ship","title":"Fleet Op"}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.CalendarEventDetail(context.Background(), 456)
	if err != nil {
		t.Fatalf("CalendarEventDetail returned error: %v", err)
	}
	if detail.Title != "Fleet Op" || detail.Duration != 60 || detail.OwnerName != "Test Corp" {
		t.Errorf("detail = %+v, want Title=\"Fleet Op\" Duration=60 OwnerName=\"Test Corp\"", detail)
	}
}

func TestCalendarEventDetail_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CalendarEventDetail(context.Background(), 456); err == nil {
		t.Fatal("CalendarEventDetail returned nil error, want an error for HTTP 404")
	}
}

func TestCalendarEventAttendees_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"character_id":998877,"event_response":"accepted"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	attendees, err := client.CalendarEventAttendees(context.Background(), 456)
	if err != nil {
		t.Fatalf("CalendarEventAttendees returned error: %v", err)
	}
	if len(attendees) != 1 || attendees[0].CharacterId == nil || *attendees[0].CharacterId != 998877 {
		t.Errorf("attendees = %+v, want one entry with CharacterId=*998877", attendees)
	}
}

func TestRespondToCalendarEvent_SendsCorrectBodyAndHandlesNoContent(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.RespondToCalendarEvent(context.Background(), 456, "accepted"); err != nil {
		t.Fatalf("RespondToCalendarEvent returned error: %v", err)
	}
	// The body may have a trailing newline from the JSON encoder
	if gotBody != `{"response":"accepted"}` && gotBody != `{"response":"accepted"}`+"\n" {
		t.Errorf("request body = %q, want %q or %q", gotBody, `{"response":"accepted"}`, `{"response":"accepted"}`+"\n")
	}
}

func TestRespondToCalendarEvent_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.RespondToCalendarEvent(context.Background(), 456, "accepted"); err == nil {
		t.Fatal("RespondToCalendarEvent returned nil error, want an error for HTTP 403")
	}
}
