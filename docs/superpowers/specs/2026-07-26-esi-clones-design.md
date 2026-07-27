# ESI Clones (Sixth Batch) Design

## Status

Approved (autonomous execution). Sixth batch: Clones tag alone (2 endpoints — kept standalone
rather than padded with an adjacent tag, since Contacts (next in order) is a 9-endpoint tag with
3 write operations and a meaningfully different complexity profile; bundling them would produce a
mismatched batch).

## Scope

Both endpoints are standard character-scoped, read-only, no public split needed:
- `GetCharactersCharacterIdClones` — jump clones + home location. Scope `esi-clones.read_clones.v1`.
- `GetCharactersCharacterIdImplants` — active implant type IDs (bare `[]int64`). Scope
  `esi-clones.read_implants.v1`.

Both scopes verified present in `mcp/add_character.go`'s `allScopes`.

## `esi/clones.go`

```go
func (c *Client) CharacterClones(ctx context.Context) (*esiapi.CharactersCharacterIdClonesGet, error)
func (c *Client) CharacterImplants(ctx context.Context) ([]int64, error)
```

Both `describeError`-wrapped, using `int64(c.characterID)` per the standard private-facade pattern.
Verified signatures: `GetCharactersCharacterIdClones(ctx, characterId int64) ...Execute() (*esiapi.CharactersCharacterIdClonesGet, *http.Response, error)`, `GetCharactersCharacterIdImplants(ctx, characterId int64) ...Execute() ([]int64, *http.Response, error)`.

`CharactersCharacterIdClonesGet` has optional `HomeLocation *CharactersCharacterIdClonesGetHomeLocation`
(itself with optional `LocationId *int64`/`LocationType *string`), `LastCloneJumpDate`/`LastStationChangeDate *time.Time`,
and a non-optional `JumpClones []CharactersCharacterIdClonesGetJumpClonesInner` where each entry has
`Implants []int64, JumpCloneId, LocationId int64, LocationType string, Name *string`.

## `mcp/clones.go` (2 tools, standard existence-check pattern)

- `get_character_clones` — output `{characterId, homeLocationId *int64, homeLocationType *string, lastCloneJumpDate *time.Time, lastStationChangeDate *time.Time, jumpClones []JumpClone}` where `JumpClone{JumpCloneID, LocationID int64, LocationType string, Name *string, Implants []int64}`
- `list_character_implants` — output `{characterId, implantTypeIds []int64}`

Raw-ID passthrough: implant/location type IDs are not resolved to names.

## Testing

Same pattern as every private tool: `esi/clones_test.go` (httptest), `mcp/clones_internal_test.go`
(func-var override, happy-path + unknown-character), `mcp/server_test.go` grows from 37 to 39.
