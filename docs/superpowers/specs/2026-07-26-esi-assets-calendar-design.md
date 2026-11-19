# ESI Assets + Calendar (Fourth Batch) Design

## Status

Approved. Fourth batch of the ESI coverage sweep (`docs/reference/esi-coverage-roadmap.md`),
covering the Assets and Calendar tags together (10 endpoints). Per explicit standing instruction,
this batch lands in the same shared PR as every prior batch since Wallet (currently PR #6), not a
new PR — see the "single shared PR" convention in project memory.

## Goal

Cover Assets (6 endpoints, including corporation-scoped variants) and Calendar (4 endpoints,
including ESI's first write endpoint in this sweep — responding to a calendar invite). Both were
explicitly deferred from the Alliance batch pending their own design.

## Scope decisions

**Assets: corp-scoped variants included.** Character-scoped Assets (`list`, `resolve-locations`,
`resolve-names`) and corporation-scoped Assets (same three operations, for a corp) are both in
this batch. The corp variants take both `characterId` (whose stored auth token to use) and
`corporationId` (which corp to query) as inputs — the same pattern already established for
Alliance's `allianceId`. ESI itself enforces that the authenticated character actually holds
sufficient corp roles (403 otherwise); no new client-side authorization logic is needed. This
project still has no other corp-level concept (no stored `corporationId` per character, no corp
auth flow) — these tools work by the caller supplying the corp ID explicitly, same as Alliance.

**Assets' POST endpoints are read-only, verified from source, not assumed.** Both
`PostCharactersCharacterIdAssetsLocations` and `PostCharactersCharacterIdAssetsNames`'s own doc
comments read "Get character asset locations / names... Return locations/names for a set of item
ids, which you can get from character assets endpoint." They're POST purely because the request
needs a body (`RequestBody(itemIDs []int64)` — verified via a live `httptest.Server` round-trip
that the ID list serializes as a bare JSON array, `[1,2,3]`, not a wrapped object), not because
they mutate anything. No special write-tool caution applies to them.

**Calendar's PUT is a genuine write**, verified from source: `PutCharactersCharacterIdCalendarEventId`'s
doc comment reads "Respond to an event / Set your response status to an event." Verified live: it
returns only `(*http.Response, error)` — a 204 No Content, no response body — the first method in
this project with that shape.

**Write-tool convention (new, applies going forward, not just this endpoint).** No new formal
CLAUDE.md hard-rule section yet for a single write tool, but the pattern to follow for this and
every future write endpoint in the sweep (Fleets, Contacts, Mail, Fittings, Corporation Projects,
etc. all have writes coming later):
- Tool name is verb-first and distinct from every `list_`/`get_` read tool (e.g.
  `respond_calendar_event`, not `set_calendar_event_response` or similar that could be mistaken for
  a getter).
- Tool description explicitly states the tool changes state (e.g. "this changes your RSVP status
  on EVE's server").
- No extra confirmation parameter — the naming and description are the safety mechanism.
- Once a second write tool exists, promote this into a proper CLAUDE.md Hard Rule with both
  examples cited, the same way "SDE over ESI" or "ESI error detail must not be discarded" are
  codified now.

## `esi/assets.go`

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// Assets returns the client's character's assets (bare IDs — call AssetLocations/AssetNames
// to resolve a subset of item IDs into positions or player-assigned names).
func (c *Client) Assets(ctx context.Context) ([]esiapi.CharactersCharacterIdAssetsGetInner, error) {
	result, resp, err := c.api.AssetsAPI.GetCharactersCharacterIdAssets(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get assets: %w", describeError(err, resp))
	}
	return result, nil
}

// AssetLocations resolves a set of item IDs (from Assets) into their positions.
func (c *Client) AssetLocations(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCharactersCharacterIdAssetsLocations(ctx, int64(c.characterID)).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get asset locations: %w", describeError(err, resp))
	}
	return result, nil
}

// AssetNames resolves a set of item IDs (from Assets) into their player-assigned custom names
// (e.g. a renamed ship or container) — not an SDE-backed item-type name lookup.
func (c *Client) AssetNames(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCharactersCharacterIdAssetsNames(ctx, int64(c.characterID)).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get asset names: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssets returns corporationID's assets. The authenticated character must hold
// sufficient corp roles — ESI itself returns 403 otherwise; no client-side check needed.
func (c *Client) CorporationAssets(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdAssetsGetInner, error) {
	result, resp, err := c.api.AssetsAPI.GetCorporationsCorporationIdAssets(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation assets: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssetLocations resolves a set of corporation item IDs into their positions.
func (c *Client) CorporationAssetLocations(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCorporationsCorporationIdAssetsLocations(ctx, corporationID).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation asset locations: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAssetNames resolves a set of corporation item IDs into their player-assigned names.
func (c *Client) CorporationAssetNames(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
	result, resp, err := c.api.AssetsAPI.PostCorporationsCorporationIdAssetsNames(ctx, corporationID).RequestBody(itemIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation asset names: %w", describeError(err, resp))
	}
	return result, nil
}
```

Verified generated signatures (from `github.com/fnt-eve/goesi-openapi@v0.0.37/esi/api_assets.go`):
- `GetCharactersCharacterIdAssets(ctx, characterId int64) ...Execute() ([]CharactersCharacterIdAssetsGetInner, *http.Response, error)`
- `PostCharactersCharacterIdAssetsLocations(ctx, characterId int64) ...RequestBody([]int64) ...Execute() ([]CharactersCharacterIdAssetsLocationsPostInner, *http.Response, error)`
- `PostCharactersCharacterIdAssetsNames(ctx, characterId int64) ...RequestBody([]int64) ...Execute() ([]CharactersCharacterIdAssetsNamesPostInner, *http.Response, error)`
- `GetCorporationsCorporationIdAssets(ctx, corporationId int64) ...Execute() ([]CorporationsCorporationIdAssetsGetInner, *http.Response, error)`
- `PostCorporationsCorporationIdAssetsLocations(ctx, corporationId int64) ...RequestBody([]int64) ...Execute() ([]CharactersCharacterIdAssetsLocationsPostInner, *http.Response, error)` — **note**: the corp variant's response reuses the character variant's `CharactersCharacterIdAssetsLocationsPostInner`/`CharactersCharacterIdAssetsNamesPostInner` types (confirmed from source — FC's generator produced identical shapes and didn't duplicate the model)
- `PostCorporationsCorporationIdAssetsNames(ctx, corporationId int64) ...RequestBody([]int64) ...Execute() ([]CharactersCharacterIdAssetsNamesPostInner, *http.Response, error)`

Response models (verified):
```go
type CharactersCharacterIdAssetsGetInner struct { // same shape for CorporationsCorporationIdAssetsGetInner
	IsBlueprintCopy *bool  `json:"is_blueprint_copy,omitempty"`
	IsSingleton     bool   `json:"is_singleton"`
	ItemId          int64  `json:"item_id"`
	LocationFlag    string `json:"location_flag"`
	LocationId      int64  `json:"location_id"`
	LocationType    string `json:"location_type"`
	Quantity        int64  `json:"quantity"`
	TypeId          int64  `json:"type_id"`
}
type CharactersCharacterIdAssetsLocationsPostInner struct {
	ItemId   int64 `json:"item_id"`
	Position CharactersCharacterIdAssetsLocationsPostInnerPosition `json:"position"`
}
type CharactersCharacterIdAssetsLocationsPostInnerPosition struct {
	X, Y, Z float64 `json:"x"/"y"/"z"`
}
type CharactersCharacterIdAssetsNamesPostInner struct {
	ItemId int64  `json:"item_id"`
	Name   string `json:"name"`
}
```

## `esi/calendar.go`

```go
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
```

Verified generated signatures (from `.../esi/api_calendar.go`):
- `GetCharactersCharacterIdCalendar(ctx, characterId int64) ...Execute() ([]CharactersCharacterIdCalendarGetInner, *http.Response, error)`
- `GetCharactersCharacterIdCalendarEventId(ctx, characterId, eventId int64) ...Execute() (*CharactersCharacterIdCalendarEventIdGet, *http.Response, error)`
- `GetCharactersCharacterIdCalendarEventIdAttendees(ctx, characterId, eventId int64) ...Execute() ([]CharactersCharacterIdCalendarEventIdAttendeesGetInner, *http.Response, error)`
- `PutCharactersCharacterIdCalendarEventId(ctx, characterId, eventId int64) ...PutCharactersCharacterIdCalendarEventIdRequest(esiapi.PutCharactersCharacterIdCalendarEventIdRequest{Response string}) ...Execute() (*http.Response, error)` — verified live via `httptest.Server`: request body serializes as `{"response":"accepted"}`, response is 204 with no body.

Response models (verified, note the pointer fields on the listing item):
```go
type CharactersCharacterIdCalendarGetInner struct {
	EventDate     *time.Time `json:"event_date,omitempty"`
	EventId       *int64     `json:"event_id,omitempty"`
	EventResponse *string    `json:"event_response,omitempty"`
	Importance    *int64     `json:"importance,omitempty"`
	Title         *string    `json:"title,omitempty"`
}
type CharactersCharacterIdCalendarEventIdGet struct {
	Date      time.Time `json:"date"`
	Duration  int64     `json:"duration"`
	EventId   int64     `json:"event_id"`
	Importance int64    `json:"importance"`
	OwnerId   int64     `json:"owner_id"`
	OwnerName string    `json:"owner_name"`
	OwnerType string    `json:"owner_type"`
	Response  string    `json:"response"`
	Text      string    `json:"text"`
	Title     string    `json:"title"`
}
type CharactersCharacterIdCalendarEventIdAttendeesGetInner struct {
	CharacterId   *int64  `json:"character_id,omitempty"`
	EventResponse *string `json:"event_response,omitempty"`
}
```

## `mcp/assets.go` (6 tools, all read-only, all character-existence-checked)

- `list_assets` — input `{characterId int32}`, output `{characterId int32, assets []AssetItem}` where `AssetItem{ItemID, TypeID, LocationID int64, LocationType, LocationFlag string, Quantity int64, IsSingleton bool, IsBlueprintCopy *bool}`
- `get_asset_locations` — input `{characterId int32, itemIds []int64}`, output `{characterId int32, locations []AssetLocation}` where `AssetLocation{ItemID int64, X, Y, Z float64}`
- `get_asset_names` — input `{characterId int32, itemIds []int64}`, output `{characterId int32, names []AssetName}` where `AssetName{ItemID int64, Name string}`
- `list_corporation_assets` — input `{characterId int32, corporationId int64}`, output `{characterId int32, corporationId int64, assets []AssetItem}`
- `get_corporation_asset_locations` — input `{characterId int32, corporationId int64, itemIds []int64}`, output `{characterId int32, corporationId int64, locations []AssetLocation}`
- `get_corporation_asset_names` — input `{characterId int32, corporationId int64, itemIds []int64}`, output `{characterId int32, corporationId int64, names []AssetName}`

Every handler follows the existing `config.Store` existence-check on `characterId`, identical to `mcp/wallet.go`/`mcp/access_list.go`/`mcp/activities.go`.

## `mcp/calendar.go` (4 tools, including the first write tool)

- `list_calendar_events` — input `{characterId int32}`, output `{characterId int32, events []CalendarEventSummary}` where `CalendarEventSummary{EventID *int64, EventDate *time.Time, EventResponse *string, Importance *int64, Title *string}` (all pointers — verified optional shape)
- `get_calendar_event` — input `{characterId int32, eventId int64}`, output `{characterId int32, eventId int64, date time.Time, duration int64, importance int64, ownerId int64, ownerName string, ownerType string, response string, text string, title string}`
- `list_calendar_event_attendees` — input `{characterId int32, eventId int64}`, output `{characterId int32, eventId int64, attendees []CalendarAttendee}` where `CalendarAttendee{CharacterID *int64, EventResponse *string}`
- `respond_calendar_event` — input `{characterId int32, eventId int64, response string}`, output `{characterId int32, eventId int64, response string}` (echoes back the sent value, since ESI returns no body). Tool description: *"Respond to a calendar event invite (accepted/declined/tentative) — this changes your RSVP status on EVE's server."* Handler validates `response ∈ {"accepted", "declined", "tentative"}` before calling ESI, returning a clear `fmt.Errorf` otherwise — ESI is never called with an invalid value.

Every handler follows the existing `config.Store` existence-check on `characterId`.

## Testing

Same three-layer isolation as every prior batch:
- `esi/assets_test.go`, `esi/calendar_test.go`: real `httptest.Server` + `ExportedNewClientWithOptions`, happy path + HTTP-error path per method. `RespondToCalendarEvent`'s test additionally asserts the outgoing request body (`{"response":"accepted"}`) and that a 204-with-no-body response doesn't error.
- `mcp/assets_internal_test.go`, `mcp/calendar_internal_test.go`: func-var override pattern, happy-path + unknown-character existence-check test for all 10 tools. `respond_calendar_event` additionally gets a test asserting an invalid `response` value (e.g. `"maybe"`) returns an error without ever calling the underlying func-var (proving ESI is never hit with bad input).
- `mcp/server_test.go`: expected-tools list grows from 13 to 23 names.

No changes to `sso/`. No new OAuth scopes — `esi-assets.read_assets.v1`, `esi-assets.read_corporation_assets.v1`, `esi-calendar.read_calendar_events.v1`, and `esi-calendar.respond_calendar_events.v1` are all already present in `mcp/add_character.go`'s `allScopes` (verified).

## Out of scope

- Any further corp-level scaffolding beyond passing a bare `corporationId` parameter (e.g. storing
  "my corporation" per character, a corp-level auth flow) — deferred to whenever the much larger
  Corporation tag (22 endpoints) is designed.
- SDE-backed enrichment of `TypeId`/`LocationId` into human-readable names — same raw-ID-passthrough
  convention as every prior batch.
- Promoting the write-tool naming convention into a formal CLAUDE.md Hard Rule — deferred until a
  second write tool exists to cite as a second example.
