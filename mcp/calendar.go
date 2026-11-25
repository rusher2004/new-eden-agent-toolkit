package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// calendarESI is the ESI calls the calendar tools make.
type calendarESI interface {
	CalendarEvents(ctx context.Context) ([]esiapi.CharactersCharacterIdCalendarGetInner, error)
	CalendarEventDetail(ctx context.Context, eventID int64) (*esiapi.CharactersCharacterIdCalendarEventIdGet, error)
	CalendarEventAttendees(ctx context.Context, eventID int64) ([]esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner, error)
	RespondToCalendarEvent(ctx context.Context, eventID int64, response string) error
}

// CalendarEventSummary is one calendar event's bare-ID summary. All fields are pointers — a
// still-pending invite can have some fields unset (verified from ESI's response model).
type CalendarEventSummary struct {
	EventID       *int64     `json:"eventId,omitempty"`
	EventDate     *time.Time `json:"eventDate,omitempty"`
	EventResponse *string    `json:"eventResponse,omitempty"`
	Importance    *int64     `json:"importance,omitempty"`
	Title         *string    `json:"title,omitempty"`
}

func listCalendarEvents(ctx context.Context, client calendarESI) ([]CalendarEventSummary, error) {
	events, err := client.CalendarEvents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CalendarEventSummary, len(events))
	for i, e := range events {
		out[i] = CalendarEventSummary{
			EventID: e.EventId, EventDate: e.EventDate, EventResponse: e.EventResponse,
			Importance: e.Importance, Title: e.Title,
		}
	}
	return out, nil
}

// ListCalendarEventsInput is list_calendar_events's tool input.
type ListCalendarEventsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListCalendarEventsOutput is list_calendar_events's tool result.
type ListCalendarEventsOutput struct {
	CharacterID int32                  `json:"characterId"`
	Events      []CalendarEventSummary `json:"events"`
}

// AddListCalendarEvents registers list_calendar_events. open builds the ESI client; any client with the
// calendarESI methods will do.
func AddListCalendarEvents[C calendarESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_calendar_events",
		Description: "List a logged-in EVE Online character's calendar events.",
	}, listCalendarEventsHandler(open, store))
}

func listCalendarEventsHandler[C calendarESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCalendarEventsInput, ListCalendarEventsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCalendarEventsInput) (*mcpsdk.CallToolResult, ListCalendarEventsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCalendarEventsOutput{}, err
		}
		if !found {
			return nil, ListCalendarEventsOutput{}, fmt.Errorf("list calendar events for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCalendarEventsOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		events, err := listCalendarEvents(ctx, client)
		if err != nil {
			return nil, ListCalendarEventsOutput{}, fmt.Errorf("list calendar events: %w", err)
		}
		return nil, ListCalendarEventsOutput{CharacterID: in.CharacterID, Events: events}, nil
	}
}

// CalendarEventDetailResult is the raw mapped detail returned by getCalendarEventDetail, before
// the handler adds the characterID/eventID it already knows from the request.
type CalendarEventDetailResult struct {
	Date       time.Time
	Duration   int64
	Importance int64
	OwnerID    int64
	OwnerName  string
	OwnerType  string
	Response   string
	Text       string
	Title      string
}

func getCalendarEventDetail(ctx context.Context, client calendarESI, eventID int64) (CalendarEventDetailResult, error) {
	detail, err := client.CalendarEventDetail(ctx, eventID)
	if err != nil {
		return CalendarEventDetailResult{}, err
	}
	return CalendarEventDetailResult{
		Date: detail.Date, Duration: detail.Duration, Importance: detail.Importance,
		OwnerID: detail.OwnerId, OwnerName: detail.OwnerName, OwnerType: detail.OwnerType,
		Response: detail.Response, Text: detail.Text, Title: detail.Title,
	}, nil
}

// GetCalendarEventInput is get_calendar_event's tool input.
type GetCalendarEventInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	EventID     int64 `json:"eventId" jsonschema:"the calendar event ID, from list_calendar_events"`
}

// GetCalendarEventOutput is get_calendar_event's tool result.
type GetCalendarEventOutput struct {
	CharacterID int32     `json:"characterId"`
	EventID     int64     `json:"eventId"`
	Date        time.Time `json:"date"`
	Duration    int64     `json:"duration"`
	Importance  int64     `json:"importance"`
	OwnerID     int64     `json:"ownerId"`
	OwnerName   string    `json:"ownerName"`
	OwnerType   string    `json:"ownerType"`
	Response    string    `json:"response"`
	Text        string    `json:"text"`
	Title       string    `json:"title"`
}

// AddGetCalendarEvent registers get_calendar_event. open builds the ESI client; any client with the
// calendarESI methods will do.
func AddGetCalendarEvent[C calendarESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_calendar_event",
		Description: "Get detail (date, duration, owner, description, RSVP status) for one calendar event.",
	}, getCalendarEventHandler(open, store))
}

func getCalendarEventHandler[C calendarESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCalendarEventInput, GetCalendarEventOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCalendarEventInput) (*mcpsdk.CallToolResult, GetCalendarEventOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCalendarEventOutput{}, err
		}
		if !found {
			return nil, GetCalendarEventOutput{}, fmt.Errorf("get calendar event for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCalendarEventOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		result, err := getCalendarEventDetail(ctx, client, in.EventID)
		if err != nil {
			return nil, GetCalendarEventOutput{}, fmt.Errorf("get calendar event: %w", err)
		}
		return nil, GetCalendarEventOutput{
			CharacterID: in.CharacterID, EventID: in.EventID, Date: result.Date, Duration: result.Duration,
			Importance: result.Importance, OwnerID: result.OwnerID, OwnerName: result.OwnerName,
			OwnerType: result.OwnerType, Response: result.Response, Text: result.Text, Title: result.Title,
		}, nil
	}
}

// CalendarAttendee is one calendar event attendee.
type CalendarAttendee struct {
	CharacterID   *int64  `json:"characterId,omitempty"`
	EventResponse *string `json:"eventResponse,omitempty"`
}

func listCalendarEventAttendees(ctx context.Context, client calendarESI, eventID int64) ([]CalendarAttendee, error) {
	attendees, err := client.CalendarEventAttendees(ctx, eventID)
	if err != nil {
		return nil, err
	}
	out := make([]CalendarAttendee, len(attendees))
	for i, a := range attendees {
		out[i] = CalendarAttendee{CharacterID: a.CharacterId, EventResponse: a.EventResponse}
	}
	return out, nil
}

// ListCalendarEventAttendeesInput is list_calendar_event_attendees's tool input.
type ListCalendarEventAttendeesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	EventID     int64 `json:"eventId" jsonschema:"the calendar event ID, from list_calendar_events"`
}

// ListCalendarEventAttendeesOutput is list_calendar_event_attendees's tool result.
type ListCalendarEventAttendeesOutput struct {
	CharacterID int32              `json:"characterId"`
	EventID     int64              `json:"eventId"`
	Attendees   []CalendarAttendee `json:"attendees"`
}

// AddListCalendarEventAttendees registers list_calendar_event_attendees. open builds the ESI client; any client with the
// calendarESI methods will do.
func AddListCalendarEventAttendees[C calendarESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_calendar_event_attendees",
		Description: "List the attendees and their RSVP status for one calendar event.",
	}, listCalendarEventAttendeesHandler(open, store))
}

func listCalendarEventAttendeesHandler[C calendarESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCalendarEventAttendeesInput, ListCalendarEventAttendeesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCalendarEventAttendeesInput) (*mcpsdk.CallToolResult, ListCalendarEventAttendeesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCalendarEventAttendeesOutput{}, err
		}
		if !found {
			return nil, ListCalendarEventAttendeesOutput{}, fmt.Errorf("list calendar event attendees for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCalendarEventAttendeesOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		attendees, err := listCalendarEventAttendees(ctx, client, in.EventID)
		if err != nil {
			return nil, ListCalendarEventAttendeesOutput{}, fmt.Errorf("list calendar event attendees: %w", err)
		}
		return nil, ListCalendarEventAttendeesOutput{CharacterID: in.CharacterID, EventID: in.EventID, Attendees: attendees}, nil
	}
}

// validCalendarResponses are ESI's own accepted values for RespondToCalendarEvent.
var validCalendarResponses = map[string]bool{"accepted": true, "declined": true, "tentative": true}

func respondToCalendarEvent(ctx context.Context, client calendarESI, eventID int64, response string) error {
	return client.RespondToCalendarEvent(ctx, eventID, response)
}

// RespondCalendarEventInput is respond_calendar_event's tool input.
type RespondCalendarEventInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to respond as"`
	EventID     int64  `json:"eventId" jsonschema:"the calendar event ID to respond to"`
	Response    string `json:"response" jsonschema:"one of accepted, declined, or tentative"`
}

// RespondCalendarEventOutput is respond_calendar_event's tool result.
type RespondCalendarEventOutput struct {
	CharacterID int32  `json:"characterId"`
	EventID     int64  `json:"eventId"`
	Response    string `json:"response"`
}

// respondCalendarEventHandler is neat's first write tool. It changes state on FC's server, so
// unlike every read-only handler in this project it validates its input BEFORE loading
// the config or calling ESI — failing fast on bad input without an unnecessary IO round-trip.
// AddRespondCalendarEvent registers respond_calendar_event. open builds the ESI client; any client with the
// calendarESI methods will do.
func AddRespondCalendarEvent[C calendarESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "respond_calendar_event",
		Description: "Respond to a calendar event invite (accepted/declined/tentative) — this changes your RSVP status on EVE's server.",
	}, respondCalendarEventHandler(open, store))
}

func respondCalendarEventHandler[C calendarESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[RespondCalendarEventInput, RespondCalendarEventOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RespondCalendarEventInput) (*mcpsdk.CallToolResult, RespondCalendarEventOutput, error) {
		if !validCalendarResponses[in.Response] {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("respond to calendar event: invalid response %q, must be one of accepted, declined, tentative", in.Response)
		}
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, RespondCalendarEventOutput{}, err
		}
		if !found {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("respond to calendar event for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		if err := respondToCalendarEvent(ctx, client, in.EventID, in.Response); err != nil {
			return nil, RespondCalendarEventOutput{}, fmt.Errorf("respond to calendar event: %w", err)
		}
		return nil, RespondCalendarEventOutput{CharacterID: in.CharacterID, EventID: in.EventID, Response: in.Response}, nil
	}
}
