package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CalendarEvents returns the client's character's calendar events. Several fields are pointers
// (a still-pending invite can have some fields unset) — verified from the response model.
func (c *Client) CalendarEvents(ctx context.Context) ([]esiapi.CharactersCharacterIdCalendarGetInner, error) {
	result, resp, err := c.api.CalendarAPI.GetCharactersCharacterIdCalendar(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get calendar events: %w", describeError(err, resp))
	}
	return result, nil
}

// CalendarEventDetail returns full detail for one calendar event.
func (c *Client) CalendarEventDetail(ctx context.Context, eventID int64) (*esiapi.CharactersCharacterIdCalendarEventIdGet, error) {
	result, resp, err := c.api.CalendarAPI.GetCharactersCharacterIdCalendarEventId(ctx, int64(c.characterID), eventID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get calendar event detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CalendarEventAttendees returns the attendee list for one calendar event.
func (c *Client) CalendarEventAttendees(ctx context.Context, eventID int64) ([]esiapi.CharactersCharacterIdCalendarEventIdAttendeesGetInner, error) {
	result, resp, err := c.api.CalendarAPI.GetCharactersCharacterIdCalendarEventIdAttendees(ctx, int64(c.characterID), eventID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get calendar event attendees: %w", describeError(err, resp))
	}
	return result, nil
}

// RespondToCalendarEvent sets the client's character's RSVP status on a calendar event.
// response must be one of "accepted", "declined", or "tentative" — ESI's own accepted values;
// the MCP layer validates this before calling, so a bad value fails fast with a clear message.
// This is a write: it changes state on FC's server. ESI returns no response body (204).
func (c *Client) RespondToCalendarEvent(ctx context.Context, eventID int64, response string) error {
	resp, err := c.api.CalendarAPI.PutCharactersCharacterIdCalendarEventId(ctx, int64(c.characterID), eventID).
		PutCharactersCharacterIdCalendarEventIdRequest(esiapi.PutCharactersCharacterIdCalendarEventIdRequest{Response: response}).
		Execute()
	if err != nil {
		return fmt.Errorf("respond to calendar event: %w", describeError(err, resp))
	}
	return nil
}
