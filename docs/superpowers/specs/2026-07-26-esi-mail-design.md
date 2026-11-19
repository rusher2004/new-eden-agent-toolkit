# ESI Mail — Design Spec (twenty-third batch)

**Batch:** Mail tag, 9 endpoints, all private ("my own character") — 4 read, 5 write. This is the
largest write-heavy batch since Fleets.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements — full verification detail, including drafted and
sandbox-compiled code for both layers, delegated to a research subagent)

All 9 `/characters/{character_id}/mail...` paths require an
`esi-mail.{read_mail,send_mail,organize_mail}.v1` character-scoped OAuth2 scope — every method is
private/"my own character" style (`int64(c.characterID)`, never a caller-supplied character ID).

1. `GetCharactersCharacterIdMail` → `[]CharactersCharacterIdMailGetInner` (all fields pointer-optional:
   `From`, `IsRead`, `Labels []int64`, `MailId`, `Recipients []PostCharactersCharacterIdMailRequestRecipientsInner`,
   `Subject`, `Timestamp`). **Pagination is NOT this project's usual page/`X-Pages` style** — the
   request builder's only optional params are `.Labels([]int64)` and `.LastMailId(int64)`; ESI's own
   doc comment: "Return the 50 most recent mail headers... last_mail_id can be used to paginate
   backwards." `totalPagesFromHeader` (`esi/contracts.go`) does not apply here.
2. `GetCharactersCharacterIdMailLabels` → `CharactersCharacterIdMailLabelsGet{Labels []...Inner, TotalUnreadCount *int64}`,
   each label `{Color *string, LabelId *int64, Name *string, UnreadCount *int64}`.
3. `GetCharactersCharacterIdMailLists` → `[]CharactersCharacterIdMailListsGetInner{MailingListId int64, Name string}`
   (both required/non-pointer, unlike every other mail model).
4. `GetCharactersCharacterIdMailMailId` → `CharactersCharacterIdMailMailIdGet{Body *string, From *int64,
   Labels []int64, Read *bool, Recipients []PostCharactersCharacterIdMailRequestRecipientsInner, Subject *string, Timestamp *time.Time}`
   — same recipient type as #1 (confirmed identical generated type, not just structurally similar).
5. `PostCharactersCharacterIdMail` (write) → `int64` (new mail ID). Body:
   `PostCharactersCharacterIdMailRequest{ApprovedCost *int64, Body string, Recipients []...Inner, Subject string}`.
   Recipient type `PostCharactersCharacterIdMailRequestRecipientsInner{RecipientId int64, RecipientType string}`
   is a **flat struct with a string discriminator**, not a oneOf/polymorphic type — `RecipientType`
   is documented in the OpenAPI spec as an enum (`alliance`/`character`/`corporation`/`mailing_list`)
   but `goesi-openapi` generates a plain `string` field, not a Go enum. This is emphatically not the
   Killmails self-referential-type trap.
6. `PostCharactersCharacterIdMailLabels` (write) → `int64` (new label ID). Body:
   `PostCharactersCharacterIdMailLabelsRequest{Color *string, Name string}`.
7. `PutCharactersCharacterIdMailMailId` (write) → no response body, same shape as
   `DeleteCharacterFitting` (`esi/fittings.go`). Body: `PutCharactersCharacterIdMailMailIdRequest{Labels []int64, Read *bool}`.
8. `DeleteCharactersCharacterIdMailMailId` (write) → no response body.
9. `DeleteCharactersCharacterIdMailLabelsLabelId` (write) → no response body.

**No recursion anywhere** — every model is flat scalars/pointers, a `[]int64`/`[]string`, or the
flat recipient struct reused 3 times. Confirmed both by source inspection and by a sandboxed live
`mcpsdk.AddTool` + `Connect` registration test with zero schema-generator panics (this project's
standing convention since the Killmails batch).

## `esi/mail.go`

9 methods on `*Client`, verb + `CharacterMail*` naming (`CharacterMailHeaders`, `CharacterMailLabels`,
`CharacterMailLists`, `CharacterMail`, `SendCharacterMail`, `CreateCharacterMailLabel`,
`UpdateCharacterMail`, `DeleteCharacterMail`, `DeleteCharacterMailLabel`) — same convention as
`CharacterFittings`/`DeleteCharacterFitting` in `esi/fittings.go`. Full code, already sandbox-verified
clean (`go build`/`go vet`/`gofmt -l`/`go test`), is in the implementation plan's Task 1.

`esi/mail_test.go`: 20 tests (success + HTTP-error pair per method, plus an extra
`OmitsQueryParamsWhenUnset` case for `CharacterMailHeaders`' optional query params), following
`esi/fittings_test.go`'s exact `httptest.Server` round-trip style.

## `mcp/mail.go`

9 tools (4 read: `list_character_mail`, `get_character_mail_labels`, `list_character_mail_lists`,
`get_character_mail`; 5 write, verb-first: `send_character_mail`, `create_character_mail_label`,
`update_character_mail`, `delete_character_mail`, `delete_character_mail_label`).

**Recipient enum modeling:** `MailRecipient{RecipientID int64, RecipientType string}` (output) and
`MailRecipientInput` (input) mirror the flat generated struct 1:1 — same split as
`FittingItem`/`FittingItemInput` in `mcp/fittings.go`. `RecipientType` stays a plain string with a
`jsonschema` hint documenting the valid values, since `goesi-openapi` itself doesn't generate an
enum type and inventing one in this facade would risk drifting from ESI's actual model.

**`last_mail_id` cursor in the MCP output shape:** `ListCharacterMailOutput` carries an `omitempty`
`NextLastMailID *int64`, computed as the minimum `MailID` across the returned batch, so a calling
agent can copy it straight into the next call's `lastMailId` to page further back — no `page`/
`totalPages` pair, since ESI's mail-listing endpoint has no such concept (see the pagination note
above).

Full code for all 9 handlers, already sandbox-verified against a live `mcpsdk.AddTool`+`Connect`
registration test (zero schema-generator panics), is in the implementation plan's Task 2.

`mcp/mail_internal_test.go`: 18 tests (a `MapsFields`/`Succeeds` + `UnknownCharacter` pair per tool),
following `mcp/fittings_internal_test.go`'s swappable-var-override pattern, with full field-coverage
assertions (no shallow checks, per this project's Corporation-Part-2 lesson).

## `mcp/server.go` / `mcp/server_test.go`

Register 9 tools after the `list_loyalty_store_offers` block. Update package doc comment.
`server_test.go`'s expected-tools list: 137 → 146.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred thirty-seven" → "one hundred forty-six",
  append the 9 new tool names.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Mail row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/mail_test.go`,
swappable-`var`-override tests for `mcp/mail_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.
