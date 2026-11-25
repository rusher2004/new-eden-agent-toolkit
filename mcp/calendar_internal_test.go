package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// fakeCalendarESI fakes the calendar tools' ESI calls; err is returned by every call.
type fakeCalendarESI struct {
	events    []esiapi.CharactersCharacterIdCalendarGetInner
	detail    *esiapi.CharactersCharacterIdCalendarEventIdGet
	attendees []esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner
	err       error

	gotResponse string
}

func (f *fakeCalendarESI) CalendarEvents(context.Context) ([]esiapi.CharactersCharacterIdCalendarGetInner, error) {
	return f.events, f.err
}

func (f *fakeCalendarESI) CalendarEventDetail(context.Context, int64) (*esiapi.CharactersCharacterIdCalendarEventIdGet, error) {
	return f.detail, f.err
}

func (f *fakeCalendarESI) CalendarEventAttendees(context.Context, int64) ([]esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner, error) {
	return f.attendees, f.err
}

func (f *fakeCalendarESI) RespondToCalendarEvent(_ context.Context, _ int64, response string) error {
	f.gotResponse = response
	return f.err
}

// openCalendarESI opens the fake client for one character.
type openCalendarESI = func(ctx context.Context, characterID int32) (calendarESI, error)

func openCalendarFake(c calendarESI) openCalendarESI {
	return func(context.Context, int32) (calendarESI, error) { return c, nil }
}

// openCalendarMustNotRun fails the test if a handler opens an ESI client.
func openCalendarMustNotRun(t *testing.T) openCalendarESI {
	t.Helper()
	return func(context.Context, int32) (calendarESI, error) {
		t.Fatal("no ESI client should be opened")
		return nil, nil
	}
}

func TestListCalendarEventsHandler_ReturnsEvents(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	eventID := int64(456)
	title := "Fleet Op"
	fake := &fakeCalendarESI{events: []esiapi.CharactersCharacterIdCalendarGetInner{{EventId: &eventID, Title: &title}}}

	handler := listCalendarEventsHandler(openCalendarFake(fake), store)
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

	handler := listCalendarEventsHandler(openCalendarMustNotRun(t), store)
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
	fake := &fakeCalendarESI{detail: &esiapi.CharactersCharacterIdCalendarEventIdGet{
		Date: wantDate, Duration: 60, Importance: 1, OwnerId: 98000001,
		OwnerName: "Test Corp", OwnerType: "corporation", Response: "needs_response",
		Text: "Bring your ship", Title: "Fleet Op",
	}}

	handler := getCalendarEventHandler(openCalendarFake(fake), store)
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

	handler := getCalendarEventHandler(openCalendarMustNotRun(t), store)
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

	charID := int64(998877)
	response := "accepted"
	fake := &fakeCalendarESI{attendees: []esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner{{CharacterId: &charID, EventResponse: &response}}}

	handler := listCalendarEventAttendeesHandler(openCalendarFake(fake), store)
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

	handler := listCalendarEventAttendeesHandler(openCalendarMustNotRun(t), store)
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

	fake := &fakeCalendarESI{}

	handler := respondCalendarEventHandler(openCalendarFake(fake), store)
	_, out, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 7, EventID: 456, Response: "accepted"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if fake.gotResponse != "accepted" || out.Response != "accepted" || out.EventID != 456 {
		t.Errorf("out = %+v, gotResponse = %q, want response \"accepted\" passed through and echoed", out, fake.gotResponse)
	}
}

func TestRespondCalendarEventHandler_InvalidResponse_NeverCallsESI(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	handler := respondCalendarEventHandler(openCalendarMustNotRun(t), store)
	_, _, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 7, EventID: 456, Response: "maybe"})
	if err == nil {
		t.Fatal("handler returned nil error, want a validation error for an invalid response value")
	}
}

func TestRespondCalendarEventHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := respondCalendarEventHandler(openCalendarMustNotRun(t), store)
	_, _, err := handler(context.Background(), nil, RespondCalendarEventInput{CharacterID: 999, EventID: 456, Response: "accepted"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestCalendarHandlers_PropagateESIAndOpenErrors(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	failingOpen := func(context.Context, int32) (calendarESI, error) { return nil, errBoom }
	failingESI := openCalendarFake(&fakeCalendarESI{err: errBoom})

	calls := map[string]func(open openCalendarESI) error{
		"list_calendar_events": func(open openCalendarESI) error {
			_, _, err := listCalendarEventsHandler(open, store)(context.Background(), nil, ListCalendarEventsInput{CharacterID: 7})
			return err
		},
		"get_calendar_event": func(open openCalendarESI) error {
			_, _, err := getCalendarEventHandler(open, store)(context.Background(), nil, GetCalendarEventInput{CharacterID: 7, EventID: 456})
			return err
		},
		"list_calendar_event_attendees": func(open openCalendarESI) error {
			_, _, err := listCalendarEventAttendeesHandler(open, store)(context.Background(), nil, ListCalendarEventAttendeesInput{CharacterID: 7, EventID: 456})
			return err
		},
		"respond_calendar_event": func(open openCalendarESI) error {
			_, _, err := respondCalendarEventHandler(open, store)(context.Background(), nil, RespondCalendarEventInput{CharacterID: 7, EventID: 456, Response: "accepted"})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name+"/esi error", func(t *testing.T) {
			if err := call(failingESI); !errors.Is(err, errBoom) {
				t.Errorf("err = %v, want wrapping errBoom", err)
			}
		})
		t.Run(name+"/open error", func(t *testing.T) {
			if err := call(failingOpen); !errors.Is(err, errBoom) {
				t.Errorf("err = %v, want wrapping errBoom", err)
			}
		})
	}
}
