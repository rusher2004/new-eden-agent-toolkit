# ESI Fittings (Fourteenth Batch) Design

## Status

Approved (autonomous execution). Fourteenth batch: the "Fittings" tag alone (3 endpoints, all
private/"my own character"). This tag's first 2 endpoints (list, create) are read/write; the third
(delete) is the project's first **DELETE** write endpoint (Calendar's `respond_calendar_event` was
the first PUT write; Contacts added POST/PUT/DELETE for contacts specifically — Fittings is the
first DELETE on a resource with no corresponding "update" verb, i.e. create-or-destroy, not
create/modify/destroy).

## Scope

All 3 endpoints are **private**, character-scoped, "my own character" (no separate target-character
param — same convention as Calendar/Access List, verified from the generated request builders which
take `characterId` as a plain positional int64 that this project's `esi/` facade passes as
`int64(c.characterID)`, not as a caller-supplied parameter):

- `GetCharactersCharacterIdFittings` — list fittings (read-only, not paginated).
- `PostCharactersCharacterIdFittings` — create a fitting (write; request body `{description, items[], name, ship_type_id}`; response `{fitting_id}`).
- `DeleteCharactersCharacterIdFittingsFittingId` — delete a fitting (write; takes `fittingId`; returns only `(*http.Response, error)`, no body — same "write returns only error" shape as `RespondToCalendarEvent`).

## `esi/fittings.go` (3 methods)

```go
func (c *Client) CharacterFittings(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error)
func (c *Client) CreateCharacterFitting(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error)
func (c *Client) DeleteCharacterFitting(ctx context.Context, fittingID int64) error
```

`CreateCharacterFitting` returns the bare `fittingID int64` (from `CharactersCharacterIdFittingsPost.FittingId`), not the whole response struct, since that's the only field ESI returns. `DeleteCharacterFitting` follows the exact `RespondToCalendarEvent` shape (`esi/calendar.go`) — routes error+response through `describeError`, returns only `error` on success.

Verified model shapes: `CharactersCharacterIdFittingsGetInner{Description, Name string, FittingId,
ShipTypeId int64, Items []...ItemsInner}`; `...ItemsInner{Flag string, Quantity, TypeId int64}` (all
required, no pointers, on both the GET response item shape and the POST request item shape — they
happen to have the identical `{Flag, Quantity, TypeId}` shape but are distinct generated Go types
for GET vs POST, per ESI's usual codegen pattern).

## `mcp/fittings.go` (3 tools)

All private (existence-checked via existing `characterExists`):

- `list_character_fittings` — input `{characterId int32}`, output `{characterId int32, fittings []Fitting}` where `Fitting{FittingID, ShipTypeID int64, Description, Name string, Items []FittingItem}` and `FittingItem{Flag string, Quantity, TypeID int64}`
- `create_character_fitting` — **write tool**, verb-first name per established convention (`respond_calendar_event`, `add_character_contacts`), description states it creates a new fitting on EVE's server. Input `{characterId int32, description, name string, shipTypeId int64, items []FittingItemInput}` where `FittingItemInput{Flag string, Quantity, TypeID int64}` (caller-supplied, no server-assigned fields), output `{characterId int32, fittingId int64}`
- `delete_character_fitting` — **write tool**, verb-first name, description states it deletes a fitting from EVE's server. Input `{characterId int32, fittingId int64}`, output `{characterId, fittingId int64}` (no other fields — ESI's delete returns no body)

## Testing

Same three-layer isolation as every prior batch. `esi/fittings_test.go` covers all 3 methods
(parsing + at least one error path for the read method; a live `httptest.Server` check that
`CreateCharacterFitting`'s request body serializes correctly, mirroring the precedent set for
Calendar's PUT body and Assets' POST array body; a check that `DeleteCharacterFitting` sends no
body and handles a 204 no-content response cleanly). `mcp/fittings_internal_test.go` covers
happy-path + unknown-character for all 3 tools. `mcp/server_test.go` grows from 96 to 99.

## Out of scope

- No client-side validation of `items[].flag` against ESI's known flag enum (invalid flags are
  ESI's own 4xx to return, not this project's to pre-validate) — consistent with this project's
  existing philosophy of not duplicating API-level validation it doesn't own.
