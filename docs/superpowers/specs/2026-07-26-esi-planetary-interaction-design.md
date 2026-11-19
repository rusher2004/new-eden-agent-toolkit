# ESI Planetary Interaction — Design Spec (twenty-sixth batch)

**Batch:** Planetary Interaction tag, 4 endpoints — 2 private ("my own character"), 1 corp-scoped,
1 public. Full verification detail, including drafted and sandbox-compiled code for both layers
with a live `mcpsdk.AddTool`+`Connect` registration test.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

1. `GetCharactersCharacterIdPlanets` — private, "my own character" (OAuth2 scope
   `esi-planets.manage_planets.v1`). Not paginated. Response: `[]CharactersCharacterIdPlanetsGetInner`
   — 7 required fields (`LastUpdate`, `NumPins`, `OwnerId`, `PlanetId`, `PlanetType`,
   `SolarSystemId`, `UpgradeLevel`).
2. `GetCharactersCharacterIdPlanetsPlanetId` — private, same scope. Not paginated. Response: one
   colony's full layout, `*CharactersCharacterIdPlanetsPlanetIdGet{Links, Pins, Routes}` — **this
   batch's most structurally complex model, and explicitly verified NOT recursive.** `Links`/`Pins`/
   `Routes` are 3 flat sibling slices; `Links`/`Routes` reference pins purely by `int64` ID
   (`SourcePinId`/`DestinationPinId`/`Waypoints []int64`), never by embedding another
   `Link`/`Route`/`Pin` value. `Pin` carries one further optional level
   (`Contents`/`ExtractorDetails`/`FactoryDetails`), and `ExtractorDetails` carries one further level
   (`Heads`) — neither loops back to `Pin`, `Link`, or `Route`. Confirmed field-by-field across all
   7 generated model files. This is the "genuinely no recursion" case this project's standing
   Killmails-derived rule anticipates: no split-into-inner/outer type is needed because
   `goesi-openapi` never generated a self-referential shape for PI in the first place.
3. `GetCorporationsCorporationIdCustomsOffices` — private, corp-scoped (OAuth2 scope
   `esi-planets.read_customs_offices.v1`, `x-required-roles: [Director]` — same
   characterId-authenticates/corporationId-is-the-resource/ESI's-own-403-is-the-gate pattern as
   `esi/industry.go`). **IS paginated** (verified: `.Page()` setter present, `X-Pages` response
   header). Response: 15 fields, most of the tax-rate/standing fields genuinely optional (`*float64`/
   `*string`) per ESI's own doc comments — present only when the corresponding access level is
   configured.
4. `GetUniverseSchematicsSchematicId` — genuinely public (no `security:` block at all, unlike the
   other 3 methods in this tag). Not paginated. Response: just 2 fields, `CycleTime int64` and
   `SchematicName string` — the schematic's material inputs/outputs are SDE static data, not
   something this ESI endpoint returns.

## `esi/planetary_interaction.go`

4 methods on `*Client`. Full code, already sandbox-verified clean (`go build`/`go vet`/`gofmt -l`/
`go test`), is in the implementation plan's Task 1. Returns raw `esiapi.*` types directly (no
intermediate esi-layer reshaping needed) — matches `esi/fittings.go`/`esi/industry.go`'s precedent.

`esi/planetary_interaction_test.go`: 9 tests, including one exhaustive colony-detail test covering
every nested level (Links, 2 Pins — one fully populated including `ExtractorDetails.Heads` and
`FactoryDetails`, one deliberately bare to prove optional fields decode as nil/empty — and Routes
with Waypoints), plus a dedicated page-omission test and a real `X-Pages` pagination assertion for
customs offices.

## `mcp/planetary_interaction.go`

4 tools: `list_character_planets`, `get_character_planet`, `list_corporation_customs_offices`,
`get_planetary_schematic`.

**How Pins/Links/Routes were modeled — flat siblings, not nesting**: `GetCharacterPlanetOutput` has
`Links []PlanetLink`, `Pins []PlanetPin`, `Routes []PlanetRoute` as 3 flat sibling slices, exactly
mirroring ESI's own shape. `PlanetLink`/`PlanetRoute` reference pins only by `int64` ID fields — they
do **not** embed a `PlanetPin`, and `PlanetPin` does **not** embed a `PlanetLink`, `PlanetRoute`, or
another `PlanetPin`. `PlanetPin`'s one level of optional nested detail
(`Contents`/`ExtractorDetails`/`FactoryDetails`) and `PlanetExtractorDetails`'s one further level
(`Heads`) mirror ESI's real nesting depth exactly — one distinct, non-recursive type per real level,
the pattern this project's Killmails rule prescribes, applied here to a case that never needed the
split in the first place since `goesi-openapi`'s generated types were already non-recursive.

Full code for all 4 handlers, already sandbox-verified against a live `mcpsdk.AddTool`+`Connect`
registration test (zero schema-generator panics — the load-bearing check per this project's
standing rule that a recursive Go type builds/vets/gofmts clean but panics only at this point), is
in the implementation plan's Task 2.

`mcp/planetary_interaction_internal_test.go`: 10 tests — a `MapsFields` + `UnknownCharacter` pair
per private/corp tool (3 tools), a `DefaultsPageToOne` test for the paginated corp tool, and
`MapsFields`/`PropagatesError` for the public schematic tool. The colony-detail test asserts every
field at every nesting level on both a fully-populated Pin and a deliberately bare one, per this
project's "no shallow assertions" lesson (Corporation Projects Part 2, Market Task 1).

## `mcp/server.go` / `mcp/server_test.go`

Register 4 tools after the `get_esi_status` block. Update package doc comment.
`server_test.go`'s expected-tools list: 161 → 165.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred sixty-one" → "one hundred sixty-five",
  append the 4 new tool names.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Planetary Interaction row from "Planned"
  to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for
`esi/planetary_interaction_test.go`, swappable-`var`-override tests for
`mcp/planetary_interaction_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.
