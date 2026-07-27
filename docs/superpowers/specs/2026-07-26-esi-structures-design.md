# ESI Structures Design (32nd batch)

**Status:** approved (autonomous sweep — see `docs/superpowers/specs/2026-07-23-foundation-design.md`
and the sweep's standing conventions in `docs/reference/esi-coverage-roadmap.md`). Written directly
by the orchestrator, no subagent dispatch, per the user's explicit instruction after the session's
subagent spawn cap was hit during the Sovereignty batch.

## Scope

CCP's "Structures" tag, 6 endpoints, all verified directly against the installed
`github.com/fnt-eve/goesi-openapi@v0.0.37` module's `esi/api_structures.go` and
`esi/api/openapi.yaml`:

| Endpoint | operationId | Security | Paginated |
|---|---|---|---|
| `GET /characters/{character_id}/structures/mercenary-dens` | `GetCharactersStructuresMercenaryDensListing` | `esi-structures.read_character.v1` | No |
| `GET /characters/{character_id}/structures/mercenary-dens/{mercenary_den_id}` | `GetCharactersStructuresMercenaryDensDetail` | `esi-structures.read_character.v1` | No |
| `GET /corporations/{corporation_id}/structures/skyhooks` | `GetCorporationsStructuresSkyhooksListing` | `esi-structures.read_corporation.v1` | No |
| `GET /corporations/{corporation_id}/structures/skyhooks/{skyhook_id}` | `GetCorporationsStructuresSkyhooksDetail` | `esi-structures.read_corporation.v1` | No |
| `GET /corporations/{corporation_id}/structures/sovereignty-hubs` | `GetCorporationsStructuresSovereigntyHubsListing` | `esi-structures.read_corporation.v1` | No |
| `GET /corporations/{corporation_id}/structures/sovereignty-hubs/{sovereignty_hub_id}` | `GetCorporationsStructuresSovereigntyHubsDetail` | `esi-structures.read_corporation.v1` | No |

None of the 6 request builders has a `.Page()` setter — verified individually per endpoint, not
assumed from tag similarity (per the project's standing rule).

Both OAuth2 scopes (`esi-structures.read_character.v1`, `esi-structures.read_corporation.v1`) are
already present in `mcp/add_character.go`'s `allScopes` (added in an earlier batch) — no re-auth
flow needed.

Note: `GET /skyhooks/raidable` (`GetSkyhooksRaidable`) is also declared on `StructuresAPIService`
in the generated client, but it is already covered by the existing `list_raidable_skyhooks` tool
from an earlier batch (Access List/Activities) — out of scope here, not re-implemented.

## Conventions applied

- Mercenary Dens are "my own character" (private, no caller-supplied character ID beyond the one
  baked into `esi.Client` via `esi.NewClient`) — same pattern as `esi/skills.go`.
- Skyhooks and Sovereignty Hubs are corp-scoped (characterId authenticates via the cached HTTP
  client, corporationId is the resource; ESI's own 403 is the sole authorization gate, no
  additional role check in this facade) — same pattern as
  `esi/planetary_interaction.go`'s `CorporationCustomsOffices`.
- Every method returns exactly what ESI returns, zero name resolution.

## Nested model verification (non-recursion check)

Per the project's standing rule (discovered during the Killmails batch: a self-referential Go
struct compiles/vets/gofmts clean but panics at runtime inside the MCP SDK's schema generator),
every nested model below was read directly from its generated source file and confirmed to bottom
out in scalar/slice-of-scalar leaves — no type embeds itself, directly or transitively.

**Mercenary Den detail** (`CharactersStructuresMercenaryDensDetail`):
```
Evolution: { Anarchy: {Amount int64, Level string}, Development: {Amount int64, Level string} }
Id, TypeId int64
Infomorphs: { Amount int64 }
ReinforcementTimer *{ End time.Time }   // optional
Skyhook: { CorporationId, Id, PlanetId int64 }
State string
```
Anarchy and Development share an identical two-field shape (Amount/Level) — flattened to one
shared MCP type (`MercenaryDenEvolutionStat`), reusing the project's established
shared-type-for-near-identical-models pattern (`IndustryJob`, `MarketOrder`).

**Mercenary Den listing item** (`CharactersStructuresMercenaryDensListingMercenaryden`): `{Id,
PlanetId int64}` — flat, 2 fields.

**Skyhook detail** (`CorporationsStructuresSkyhooksDetail`):
```
EffectiveWorkforce *int64        // optional
Id, PlanetId int64
IsActive bool
Reagents []{ Amount, TypeId int64 removed -- see below }  -- actually LastCycle time.Time, SecuredStock, TypeId, UnsecuredStock int64
ReinforcementTimer *{ End time.Time }              // optional
State string
TheftVulnerability *{ End, Start time.Time }       // optional
```
(Reagent's real fields, re-verified from source: `LastCycle time.Time`, `SecuredStock int64`,
`TypeId int64`, `UnsecuredStock int64` — no `Amount`/`BurningPerHour` on this one; those names
belong to the *Sovereignty Hub* reagent, a separate, differently-shaped type. Confirmed by reading
`model_corporations_structures_skyhooks_detail_reagent.go` directly — do not conflate the two
"reagent" types across Skyhooks and Sovereignty Hubs.)

**Skyhook listing item** (`CorporationsStructuresSkyhooksListingSkyhook`): `{Id, PlanetId int64}`
— flat, 2 fields.

**Sovereignty Hub detail** (`CorporationsStructuresSovereigntyHubsDetail`):
```
FuelAccessListId *int64                    // optional
Id, SolarSystemId int64
ReagentBay: { LastUpdated time.Time, Reagents []{ Amount, BurningPerHour, TypeId int64 } }
Resources: { Power: {Allocated, Available int64}, Workforce: {Allocated, Available int64} }
Upgrades []{ PowerState string, TypeId int64 }
VulnerabilityWindow *{ End, Start time.Time }        // optional
WorkforceTransport: {
  Configuration: oneOf{ Export{Export *{Amount int64, SolarSystemId *int64}},
                        ModelImport{Import *{Sources []{SolarSystemId int64}}},
                        Transit{Transit NullableBool} }
  State:         oneOf{ Export1{Export *{Amount *int64, SolarSystemId *int64}},
                        Import1{Import *{Sources []{Amount, SolarSystemId int64}}},
                        Transit1{Transit NullableBool} }
}
```
`WorkforceTransport.Configuration` and `.State` are each a 3-way oneOf union (verified from
`model_corporations_structures_sovereignty_hubs_detail_transport_configuration.go` and
`..._transport_state.go` — each generated as a struct with 3 optional pointer fields, one of which
is set). Every member type in both unions is itself a flat 0-2-field leaf (confirmed from source:
`Export`, `ModelImport`→`Import`, `Transit`, `Export1`, `Import1`→`Import`, `Transit1` all bottom
out immediately). This mirrors the Sovereignty batch's `SovereigntyClaim` flattening: each union
collapses into one `Kind`-discriminated MCP struct.

**Sovereignty Hub listing item** (`CorporationsStructuresSovereigntyHubsListingSovereigntyhub`):
`{Id, SolarSystemId int64}` — flat, 2 fields.

No recursion anywhere in this batch — every path terminates within 2-3 levels in scalars, slices
of scalars, or slices of flat structs.

## esi/structures.go — 6 methods

```go
// Package esi's structures.go covers all 6 of "Structures"'s endpoints -- a character's own
// mercenary dens, and a corporation's skyhooks and sovereignty hubs. Mercenary dens are
// private/"my own character" (esi-structures.read_character.v1); skyhooks and sovereignty hubs
// are corp-scoped (esi-structures.read_corporation.v1), same convention as
// esi/planetary_interaction.go's CorporationCustomsOffices -- ESI's own 403 is the sole
// authorization gate, no additional role check in this facade. None of the 6 endpoints is
// paginated (verified individually: no .Page() setter on any of the 6 request builders).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterMercenaryDens returns the client's own character's mercenary dens (bare ID + planet ID
// per den; call CharacterMercenaryDenDetail for full detail on one).
func (c *Client) CharacterMercenaryDens(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
	result, resp, err := c.api.StructuresAPI.GetCharactersStructuresMercenaryDensListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mercenary dens: %w", describeError(err, resp))
	}
	return result.MercenaryDens, nil
}

// CharacterMercenaryDenDetail returns one of the client's own character's mercenary dens in full
// detail: evolution stats, infomorph count, reinforcement timer (if any), and the skyhook it's
// attached to.
func (c *Client) CharacterMercenaryDenDetail(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCharactersStructuresMercenaryDensDetail(ctx, mercenaryDenID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mercenary den detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationSkyhooks returns corporationID's skyhooks (bare ID + planet ID per skyhook; call
// CorporationSkyhookDetail for full detail on one). Corp-scoped: the authenticated character must
// hold sufficient corp roles per ESI's own x-required-roles -- this facade adds no additional
// gate, ESI's 403 is authoritative.
func (c *Client) CorporationSkyhooks(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSkyhooksListing(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation skyhooks: %w", describeError(err, resp))
	}
	return result.Skyhooks, nil
}

// CorporationSkyhookDetail returns one of corporationID's skyhooks in full detail: workforce,
// reagent stock, active/reinforcement state, and theft vulnerability window (if any).
func (c *Client) CorporationSkyhookDetail(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSkyhooksDetail(ctx, skyhookID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation skyhook detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationSovereigntyHubs returns corporationID's sovereignty hubs (bare ID + solar system ID
// per hub; call CorporationSovereigntyHubDetail for full detail on one).
func (c *Client) CorporationSovereigntyHubs(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSovereigntyHubsListing(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation sovereignty hubs: %w", describeError(err, resp))
	}
	return result.SovereigntyHubs, nil
}

// CorporationSovereigntyHubDetail returns one of corporationID's sovereignty hubs in full detail:
// reagent bay, power/workforce resources, installed upgrades, vulnerability window (if any), and
// workforce transport configuration/state.
func (c *Client) CorporationSovereigntyHubDetail(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSovereigntyHubsDetail(ctx, sovereigntyHubID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation sovereignty hub detail: %w", describeError(err, resp))
	}
	return result, nil
}
```

Note the generated request builders take `(ctx, mercenaryDenId, characterId)` and `(ctx,
skyhookId/sovereigntyHubId, corporationId)` — detail ID first, scope ID second — verified directly
from `api_structures.go`'s signatures; the facade's own parameter order
(`corporationID, skyhookID`/`corporationID, sovereigntyHubID`) matches this project's existing
convention of scope-ID-first, matching `CorporationCustomsOffices(ctx, corporationID, page)`.

## mcp/structures.go — 6 tools

Flattened output types (all non-recursive, verified above):

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_character_mercenary_dens (private, "my own character") ---

type MercenaryDen struct {
	ID       int64 `json:"id"`
	PlanetID int64 `json:"planetId"`
}

type MercenaryDenEvolutionStat struct {
	Amount int64  `json:"amount"`
	Level  string `json:"level"`
}

type MercenaryDenEvolution struct {
	Anarchy     MercenaryDenEvolutionStat `json:"anarchy"`
	Development MercenaryDenEvolutionStat `json:"development"`
}

type MercenaryDenSkyhookRef struct {
	CorporationID int64 `json:"corporationId"`
	ID            int64 `json:"id"`
	PlanetID      int64 `json:"planetId"`
}

type MercenaryDenDetail struct {
	ID               int64                 `json:"id"`
	TypeID           int64                 `json:"typeId"`
	State            string                `json:"state"`
	Evolution        MercenaryDenEvolution `json:"evolution"`
	InfomorphsAmount int64                 `json:"infomorphsAmount"`
	ReinforcementEnd *time.Time            `json:"reinforcementEnd,omitempty"`
	Skyhook          MercenaryDenSkyhookRef `json:"skyhook"`
}

// --- Skyhook types ---

type Skyhook struct {
	ID       int64 `json:"id"`
	PlanetID int64 `json:"planetId"`
}

type SkyhookReagent struct {
	LastCycle      time.Time `json:"lastCycle"`
	SecuredStock   int64     `json:"securedStock"`
	TypeID         int64     `json:"typeId"`
	UnsecuredStock int64     `json:"unsecuredStock"`
}

type SkyhookDetail struct {
	EffectiveWorkforce *int64           `json:"effectiveWorkforce,omitempty"`
	ID                 int64            `json:"id"`
	PlanetID           int64            `json:"planetId"`
	IsActive           bool             `json:"isActive"`
	Reagents           []SkyhookReagent `json:"reagents,omitempty"`
	ReinforcementEnd   *time.Time       `json:"reinforcementEnd,omitempty"`
	State              string           `json:"state"`
	TheftVulnerabilityStart *time.Time  `json:"theftVulnerabilityStart,omitempty"`
	TheftVulnerabilityEnd   *time.Time  `json:"theftVulnerabilityEnd,omitempty"`
}

// --- Sovereignty Hub types ---

type SovereigntyHub struct {
	ID           int64 `json:"id"`
	SolarSystemID int64 `json:"solarSystemId"`
}

type SovereigntyHubReagent struct {
	Amount         int64 `json:"amount"`
	BurningPerHour int64 `json:"burningPerHour"`
	TypeID         int64 `json:"typeId"`
}

type SovereigntyHubReagentBay struct {
	LastUpdated time.Time               `json:"lastUpdated"`
	Reagents    []SovereigntyHubReagent `json:"reagents"`
}

type SovereigntyHubResourceLevel struct {
	Allocated int64 `json:"allocated"`
	Available int64 `json:"available"`
}

type SovereigntyHubResources struct {
	Power     SovereigntyHubResourceLevel `json:"power"`
	Workforce SovereigntyHubResourceLevel `json:"workforce"`
}

type SovereigntyHubUpgrade struct {
	PowerState string `json:"powerState"`
	TypeID     int64  `json:"typeId"`
}

// SovereigntyHubTransportConfig flattens ESI's 3-way oneOf (export/import/transit) for the
// requested workforce transport configuration. Kind is one of "export", "import", "transit".
type SovereigntyHubTransportConfig struct {
	Kind                string  `json:"kind"`
	ExportAmount        *int64  `json:"exportAmount,omitempty"`
	ExportSolarSystemID *int64  `json:"exportSolarSystemId,omitempty"`
	ImportSolarSystemIDs []int64 `json:"importSolarSystemIds,omitempty"`
	Transit             *bool   `json:"transit,omitempty"`
}

// SovereigntyHubTransportState flattens ESI's 3-way oneOf (export/import/transit) for the actual
// in-progress workforce transport state. Kind is one of "export", "import", "transit".
type SovereigntyHubTransportState struct {
	Kind          string                          `json:"kind"`
	ExportAmount  *int64                          `json:"exportAmount,omitempty"`
	ExportSolarSystemID *int64                    `json:"exportSolarSystemId,omitempty"`
	ImportSources []SovereigntyHubTransportImportSource `json:"importSources,omitempty"`
	Transit       *bool                           `json:"transit,omitempty"`
}

type SovereigntyHubTransportImportSource struct {
	Amount        int64 `json:"amount"`
	SolarSystemID int64 `json:"solarSystemId"`
}

type SovereigntyHubTransport struct {
	Configuration SovereigntyHubTransportConfig `json:"configuration"`
	State         SovereigntyHubTransportState  `json:"state"`
}

type SovereigntyHubDetail struct {
	FuelAccessListID    *int64                    `json:"fuelAccessListId,omitempty"`
	ID                   int64                    `json:"id"`
	SolarSystemID        int64                    `json:"solarSystemId"`
	ReagentBay           SovereigntyHubReagentBay `json:"reagentBay"`
	Resources            SovereigntyHubResources  `json:"resources"`
	Upgrades             []SovereigntyHubUpgrade  `json:"upgrades"`
	VulnerabilityWindowStart *time.Time           `json:"vulnerabilityWindowStart,omitempty"`
	VulnerabilityWindowEnd   *time.Time           `json:"vulnerabilityWindowEnd,omitempty"`
	WorkforceTransport   SovereigntyHubTransport  `json:"workforceTransport"`
}
```

Note: `ModelImport`'s `Sources []{SolarSystemId int64}` (configuration side) has no `Amount` field
— only the *state* side's import sources carry an amount. So
`SovereigntyHubTransportConfig.ImportSolarSystemIDs` is `[]int64` (bare IDs), while
`SovereigntyHubTransportState.ImportSources` is `[]SovereigntyHubTransportImportSource`
(amount+ID pairs) — these two "import" shapes are NOT identical, verified from source
(`transportconfigurationsource.go` has only `SolarSystemId`; `transportstateimportsource.go` has
both `Amount` and `SolarSystemId`). Do not conflate them into one shared type.

The 6 tools:

1. `list_character_mercenary_dens` — private, "my own character", returns `[]MercenaryDen`.
2. `get_character_mercenary_den` — private, "my own character", input
   `{CharacterID int32, MercenaryDenID int64}`, returns `MercenaryDenDetail`.
3. `list_corporation_skyhooks` — private, corp-scoped, input
   `{CharacterID int32, CorporationID int64}`, returns `[]Skyhook`.
4. `get_corporation_skyhook` — private, corp-scoped, input
   `{CharacterID int32, CorporationID int64, SkyhookID int64}`, returns `SkyhookDetail`.
5. `list_corporation_sovereignty_hubs` — private, corp-scoped, input
   `{CharacterID int32, CorporationID int64}`, returns `[]SovereigntyHub`.
6. `get_corporation_sovereignty_hub` — private, corp-scoped, input
   `{CharacterID int32, CorporationID int64, SovereigntyHubID int64}`, returns
   `SovereigntyHubDetail`.

All 6 follow the existing private-tool pattern: `characterExists(store, in.CharacterID)` gate,
then `esi.NewClient(ctx, clientID, in.CharacterID, sso.UserAgent(version))`, mirroring
`mcp/skills.go` (my-own-character) and `mcp/planetary_interaction.go`'s
`listCorporationCustomsOfficesHandler` (corp-scoped).

## Testing approach

Same as every prior batch: `esi/structures_test.go` uses `httptest.Server` + a fake ESI response
JSON body per endpoint (including one full Sovereignty Hub detail JSON fixture exercising all 3
transport-config/state variants across up to 3 test cases, mirroring how Sovereignty's
`toSovereigntyClaim` got 3 variant tests), covering both the 200 success path and the
HTTP-error-wrapped-through-`describeError` path per method (12 tests: 6 endpoints × 2). Plus one
extra test asserting the corp-scoped detail param order (`skyhookId`/`sovereigntyHubId` first,
`corporationId` second) round-trips correctly in the request path, since this is the one place in
this batch where get-mixed-up-argument-order is a real risk (per the Killmails precedent).

`mcp/structures_internal_test.go` swaps each `list*`/`get*` var for a fake per the established
internal-test pattern, asserting: (a) field mapping for a "full" fixture (all optional fields
populated) and a "sparse" fixture (all optional fields nil) per detail-shaped tool, (b) all 3 Kind
variants for both `SovereigntyHubTransportConfig` and `SovereigntyHubTransportState` map correctly
with no cross-variant field bleed (mirroring Sovereignty's claim-variant tests), (c)
`ErrCharacterNotFound` propagation for all 6 handlers, (d) upstream error propagation for all 6
handlers.

## Task breakdown (mirrors Planetary Interaction's precedent: esi/ layer, then mcp/ layer)

- **Task 1**: `esi/structures.go` + `esi/structures_test.go` (6 methods, all read-only).
- **Task 2**: `mcp/structures.go` + `mcp/structures_internal_test.go` + `mcp/server.go`
  registration (6 tools) + `mcp/server_test.go`'s expected-tools list update (173 → 179 tools) +
  docs sync (`CLAUDE.md`/`README.md` tool count and list, `docs/reference/esi-coverage-roadmap.md`
  row 32 flipped to "This plan").

Both tasks done directly by the orchestrator (no subagent dispatch), per the user's standing
instruction for the remainder of this sweep. Each task is verified independently: `go build
./...`, `go vet ./...`, `gofmt -l .`, targeted `go test -v` for the new tests, then the full `go
test ./... -race` before moving to Task 2, and again before committing/pushing Task 2.
