package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCalendarEventsHandler_ReturnsEvents(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCalendarEvents
	t.Cleanup(func() { listCalendarEvents = orig })
	eventID := int64(456)
	title := "Fleet Op"
	listCalendarEvents = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CalendarEventSummary, error) {
		return []CalendarEventSummary{{EventID: &eventID, Title: &title}}, nil
	}

	handler := listCalendarEventsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCalendarEventsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Events) != 1 || out.Events[0].EventID == nil || *out.Events[0].EventID != 456 {
		t.Errorf("out = %+v, want one event with EventID=*456", out)
	}
}

func TestListCalendarEventsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCalendarEvents
	t.Cleanup(func() { listCalendarEvents = orig })
	listCalendarEvents = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]CalendarEventSummary, error) {
		t.Fatal("listCalendarEvents should not be called for an unknown character")
		return nil, nil
	}

	handler := listCalendarEventsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCalendarEventsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetCalendarEventHandler_ReturnsDetail(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	orig := getCalendarEventDetail
	t.Cleanup(func() { getCalendarEventDetail = orig })
	getCalendarEventDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) (CalendarEventDetailResult, error) {
		return CalendarEventDetailResult{
			Date: wantDate, Duration: 60, Importance: 1, OwnerID: 98000001,
			OwnerName: "Test Corp", OwnerType: "corporation", Response: "needs_response",
			Text: "Bring your ship", Title: "Fleet Op",
		}, nil
	}

	handler := getCalendarEventHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCalendarEventInput{CharacterID: 7, EventID: 456})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.EventID != 456 || out.Title != "Fleet Op" || out.Duration != 60 || !out.Date.Equal(wantDate) {
		t.Errorf("out = %+v, want EventID=456 Title=\"Fleet Op\" Duration=60 Date=%v", out, wantDate)
	}
}

func TestGetCalendarEventHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getCalendarEventDetail
	t.Cleanup(func() { getCalendarEventDetail = orig })
	getCalendarEventDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) (CalendarEventDetailResult, error) {
		t.Fatal("getCalendarEventDetail should not be called for an unknown character")
		return CalendarEventDetailResult{}, nil
	}

	handler := getCalendarEventHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCalendarEventInput{CharacterID: 999, EventID: 456})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCalendarEventAttendeesHandler_ReturnsAttendees(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCalendarEventAttendees
	t.Cleanup(func() { listCalendarEventAttendees = orig })
	charID := int64(998877)
	response := "accepted"
	listCalendarEventAttendees = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) ([]CalendarAttendee, error) {
		return []CalendarAttendee{{CharacterID: &charID, EventResponse: &response}}, nil
	}

	handler := listCalendarEventAttendeesHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCalendarEventAttendeesInput{CharacterID: 7, EventID: 456})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Attendees) != 1 || out.Attendees[0].CharacterID == nil || *out.Attendees[0].CharacterID != 998877 {
		t.Errorf("out = %+v, want one attendee with CharacterID=*998877", out)
	}
}

func TestListCalendarEventAttendeesHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCalendarEventAttendees
	t.Cleanup(func() { listCalendarEventAttendees = orig })
	listCalendarEventAttendees = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64) ([]CalendarAttendee, error) {
		t.Fatal("listCalendarEventAttendees should not be called for an unknown character")
		return nil, nil
	}

	handler := listCalendarEventAttendeesHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCalendarEventAttendeesInput{CharacterID: 999, EventID: 456})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRespondCalendarEventHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := respondToCalendarEvent
	t.Cleanup(func() { respondToCalendarEvent = orig })
	var gotResponse string
	respondToCalendarEvent = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64, response string) error {
		gotResponse = response
		return nil
	}

	handler := respondCalendarEventHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 7, EventID: 456, Response: "accepted"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotResponse != "accepted" || out.Response != "accepted" || out.EventID != 456 {
		t.Errorf("out = %+v, gotResponse = %q, want response \"accepted\" passed through and echoed", out, gotResponse)
	}
}

func TestRespondCalendarEventHandler_InvalidResponse_NeverCallsESI(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := respondToCalendarEvent
	t.Cleanup(func() { respondToCalendarEvent = orig })
	respondToCalendarEvent = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64, response string) error {
		t.Fatal("respondToCalendarEvent should not be called for an invalid response value")
		return nil
	}

	handler := respondCalendarEventHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 7, EventID: 456, Response: "maybe"})
	if err == nil {
		t.Fatal("handler returned nil error, want a validation error for an invalid response value")
	}
}

func TestRespondCalendarEventHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := respondToCalendarEvent
	t.Cleanup(func() { respondToCalendarEvent = orig })
	respondToCalendarEvent = func(ctx context.Context, clientID string, characterID int32, userAgent string, eventID int64, response string) error {
		t.Fatal("respondToCalendarEvent should not be called for an unknown character")
		return nil
	}

	handler := respondCalendarEventHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 999, EventID: 456, Response: "accepted"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
