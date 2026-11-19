# ESI Contacts (Seventh Batch) Design

## Status

Approved (autonomous execution). Seventh batch: Contacts tag alone (9 endpoints — 6 reads across
three scope levels + 3 writes, the project's 2nd/3rd/4th write endpoints after `respond_calendar_event`).

## Scope

**6 read endpoints**, all verified read-only, character-authenticated (corp/alliance reads need
the character's own roles, same pattern as Assets' corp-scoped tools — `characterId` for auth,
`corporationId`/`allianceId` for the resource):
- `list_character_contacts`, `list_character_contact_labels` (own data, existence-checked only)
- `list_corporation_contacts`, `list_corporation_contact_labels` (characterId + corporationId)
- `list_alliance_contacts`, `list_alliance_contact_labels` (characterId + allianceId)

**3 write endpoints**, verified live (query params vs. body, exact wire shapes):
- `add_character_contacts` (POST) — body: bare `[]int64` contact IDs; query params `standing float64`,
  `label_ids []int64` (optional), `watched bool` (optional). Returns the created contact IDs
  (`[]int64`) — the first write endpoint in this sweep with a real return value.
- `update_character_contacts` (PUT) — same params as POST, returns only `error` (204, matching
  `respond_calendar_event`'s no-body-response shape).
- `remove_character_contacts` (DELETE) — `contact_ids []int64` as a **query param**, not a body
  (verified live — DELETE has no request body at all). Returns only `error` (204).

All three follow the write-tool convention: verb-first name, description states the effect, no
confirmation parameter, input validated before IO where applicable.

Scopes verified present in `allScopes`: `esi-characters.read_contacts.v1`, `write_contacts.v1`,
`esi-corporations.read_contacts.v1`, `esi-alliances.read_contacts.v1`.

## `esi/contacts.go`

```go
func (c *Client) CharacterContacts(ctx context.Context) ([]esiapi.CharactersCharacterIdContactsGetInner, error)
func (c *Client) CharacterContactLabels(ctx context.Context) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
func (c *Client) CorporationContacts(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdContactsGetInner, error)
func (c *Client) CorporationContactLabels(ctx context.Context, corporationID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
func (c *Client) AllianceContacts(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsGetInner, error)
func (c *Client) AllianceContactLabels(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
func (c *Client) AddCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error)
func (c *Client) UpdateCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error
func (c *Client) RemoveCharacterContacts(ctx context.Context, contactIDs []int64) error
```

All `describeError`-wrapped. Note the character/corp/alliance contact-labels endpoints all reuse
the same generated model (`AlliancesAllianceIdContactsLabelsGetInner{LabelId, LabelName}`) —
verified from source, not assumed.

Verified live via `httptest.Server` round-trip: POST sends `?label_ids=1&standing=5&watched=true`
with body `[998878]` and returns `[998878]` (200); PUT sends `?standing=3` with the same body shape
and returns 204 no-content; DELETE sends `?contact_ids=998878` with **no body** and returns 204.

Response models (verified):
```go
type CharactersCharacterIdContactsGetInner struct {
	ContactId   int64    `json:"contact_id"`
	ContactType string   `json:"contact_type"`
	IsBlocked   *bool    `json:"is_blocked,omitempty"`
	IsWatched   *bool    `json:"is_watched,omitempty"`
	LabelIds    []int64  `json:"label_ids,omitempty"`
	Standing    float64  `json:"standing"`
}
// AlliancesAllianceIdContactsGetInner / CorporationsCorporationIdContactsGetInner: same shape,
// minus IsBlocked (alliance) or IsBlocked+IsWatched-name-difference (corp has IsWatched only).
type AlliancesAllianceIdContactsLabelsGetInner struct {
	LabelId   int64  `json:"label_id"`
	LabelName string `json:"label_name"`
}
```

## `mcp/contacts.go` (9 tools)

All 9 handlers (6 reads + 3 writes) reuse the existing `characterExists(store, characterID) (bool, error)`
helper already defined in `mcp/character.go` — it's an unexported package-level function, visible
to every file in package `mcp` without redefinition. `contacts.go` does not declare its own copy.


Reads (existence-checked on `characterId`, standard inline pattern):
- `list_character_contacts` → `{characterId, contacts []Contact}` where `Contact{ContactID int64, ContactType string, Standing float64, IsBlocked, IsWatched *bool, LabelIDs []int64}`
- `list_character_contact_labels` → `{characterId, labels []ContactLabel}` where `ContactLabel{LabelID int64, LabelName string}`
- `list_corporation_contacts` → input adds `corporationId int64`, output `{characterId, corporationId, contacts []Contact}`
- `list_corporation_contact_labels` → `{characterId, corporationId, labels []ContactLabel}`
- `list_alliance_contacts` → input adds `allianceId int64`, output `{characterId, allianceId, contacts []Contact}`
- `list_alliance_contact_labels` → `{characterId, allianceId, labels []ContactLabel}`

Writes (existence-checked, verb-first, effect-stating description):
- `add_character_contacts` — input `{characterId int32, contactIds []int64, standing float64, labelIds []int64, watched bool}`, output `{characterId, addedContactIds []int64}`. Description: "Add contacts to a character's contact list with a given standing — this creates new contacts on EVE's server."
- `update_character_contacts` — same input shape, output `{characterId, contactIds []int64}` (echoes input, no ESI response body). Description: "Update the standing/label/watched status of existing contacts — this changes state on EVE's server."
- `remove_character_contacts` — input `{characterId int32, contactIds []int64}`, output `{characterId, removedContactIds []int64}` (echoes input). Description: "Remove contacts from a character's contact list — this deletes contacts on EVE's server."

## Testing

Same three-layer isolation. `esi/contacts_test.go` verifies both read parsing and the write
endpoints' query-param/body shapes (mirroring the live verification above) plus HTTP-error paths.
`mcp/contacts_internal_test.go` covers happy-path + unknown-character for all 9 tools.
`mcp/server_test.go` grows from 39 to 48.

## Out of scope

- No client-side validation of `standing` range (-10.0 to 10.0 per ESI's own rules) — ESI enforces
  this server-side and returns a clear 4xx; adding client-side range validation would duplicate
  logic that already exists correctly on FC's side, unlike `respond_calendar_event`'s validation
  (which exists because FC's own error for an invalid `response` string is comparatively opaque).
