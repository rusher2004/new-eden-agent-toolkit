# ESI Location — Design Spec (twenty-first batch)

**Batch:** Location tag, 3 endpoints, all private ("my own character"), read-only, no pagination.

## Endpoints (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37`)

`esi/api_location.go`'s `LocationAPIService` exposes exactly 3 methods, all taking only
`(ctx, characterId int64)` plus optional standard headers (compatibility date, accept-language,
etag, tenant, if-modified-since) that this project's facade layer never surfaces (same as every
other "my own character" endpoint in the codebase):

- `GetCharactersCharacterIdLocation` → `CharactersCharacterIdLocationGet{SolarSystemId int64, StationId *int64, StructureId *int64}`
- `GetCharactersCharacterIdOnline` → `CharactersCharacterIdOnlineGet{LastLogin *time.Time, LastLogout *time.Time, Logins *int64, Online bool}`
- `GetCharactersCharacterIdShip` → `CharactersCharacterIdShipGet{ShipItemId int64, ShipName string, ShipTypeId int64}`

No write endpoints, no pagination, no oneOf/polymorphic types, no corp-scoped variants. This is
the smallest non-trivial batch since Incursions/Insurance, but unlike those it's private (needs
character auth), so it gets the standard 2-task split (esi layer, then mcp layer) rather than the
single-combined-task treatment reserved for fully-public 1-endpoint batches.

## `esi/location.go`

Follows the exact "my own character" convention from `esi/fittings.go`: each method takes only
`ctx` and passes `int64(c.characterID)` internally, never a caller-supplied character ID.

```go
// Package esi's location.go covers all 3 of "Location"'s endpoints -- current solar system/
// station/structure, online status, and current ship. All 3 are private/"my own character" --
// the generated request builders take characterId as a plain positional int64, which this facade
// passes as int64(c.characterID), never a caller-supplied parameter.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterLocation returns the client's own character's current solar system, and station or
// structure ID if docked/boarded.
func (c *Client) CharacterLocation(ctx context.Context) (*esiapi.CharactersCharacterIdLocationGet, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdLocation(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character location: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterOnline returns the client's own character's online status and login history.
func (c *Client) CharacterOnline(ctx context.Context) (*esiapi.CharactersCharacterIdOnlineGet, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdOnline(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character online status: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterShip returns the client's own character's current ship.
func (c *Client) CharacterShip(ctx context.Context) (*esiapi.CharactersCharacterIdShipGet, error) {
	result, resp, err := c.api.LocationAPI.GetCharactersCharacterIdShip(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character ship: %w", describeError(err, resp))
	}
	return result, nil
}
```

`esi/location_test.go`: standard `httptest.Server` round-trip tests per-method (3 tests), each
confirming the resolved request path is `/characters/{id}/location` etc. and that the response
fields map through untouched (these are thin passthroughs, no field renaming/flattening needed).

## `mcp/location.go`

Three tools, all following the exact `list_character_fittings` shape from `mcp/fittings.go`
(swappable `var` + `characterExists`/`ErrCharacterNotFound` reuse + `esi.NewClient`):

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

// --- get_character_location (private) ---

var getCharacterLocation = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, *int64, *int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	loc, err := client.CharacterLocation(ctx)
	if err != nil {
		return 0, nil, nil, err
	}
	return loc.SolarSystemId, loc.StationId, loc.StructureId, nil
}

type GetCharacterLocationInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterLocationOutput struct {
	CharacterID   int32  `json:"characterId"`
	SolarSystemID int64  `json:"solarSystemId"`
	StationID     *int64 `json:"stationId,omitempty"`
	StructureID   *int64 `json:"structureId,omitempty"`
}

func getCharacterLocationHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterLocationInput, GetCharacterLocationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterLocationInput) (*mcpsdk.CallToolResult, GetCharacterLocationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterLocationOutput{}, err
		}
		if !found {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		solarSystemID, stationID, structureID, err := getCharacterLocation(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterLocationOutput{}, fmt.Errorf("get character location: %w", err)
		}
		return nil, GetCharacterLocationOutput{CharacterID: in.CharacterID, SolarSystemID: solarSystemID, StationID: stationID, StructureID: structureID}, nil
	}
}

// --- get_character_online (private) ---

var getCharacterOnline = func(ctx context.Context, clientID string, characterID int32, userAgent string) (*esi.CharacterOnlineStatus, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	status, err := client.CharacterOnline(ctx)
	if err != nil {
		return nil, err
	}
	return &esi.CharacterOnlineStatus{Online: status.Online, LastLogin: status.LastLogin, LastLogout: status.LastLogout, Logins: status.Logins}, nil
}

type GetCharacterOnlineInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterOnlineOutput struct {
	CharacterID int32      `json:"characterId"`
	Online      bool       `json:"online"`
	LastLogin   *time.Time `json:"lastLogin,omitempty"`
	LastLogout  *time.Time `json:"lastLogout,omitempty"`
	Logins      *int64     `json:"logins,omitempty"`
}

func getCharacterOnlineHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterOnlineInput, GetCharacterOnlineOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterOnlineInput) (*mcpsdk.CallToolResult, GetCharacterOnlineOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, err
		}
		if !found {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		status, err := getCharacterOnline(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterOnlineOutput{}, fmt.Errorf("get character online status: %w", err)
		}
		return nil, GetCharacterOnlineOutput{CharacterID: in.CharacterID, Online: status.Online, LastLogin: status.LastLogin, LastLogout: status.LastLogout, Logins: status.Logins}, nil
	}
}

// --- get_character_ship (private) ---

var getCharacterShip = func(ctx context.Context, clientID string, characterID int32, userAgent string) (int64, string, int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, "", 0, fmt.Errorf("build ESI client: %w", err)
	}
	ship, err := client.CharacterShip(ctx)
	if err != nil {
		return 0, "", 0, err
	}
	return ship.ShipItemId, ship.ShipName, ship.ShipTypeId, nil
}

type GetCharacterShipInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterShipOutput struct {
	CharacterID int32  `json:"characterId"`
	ShipItemID  int64  `json:"shipItemId"`
	ShipName    string `json:"shipName"`
	ShipTypeID  int64  `json:"shipTypeId"`
}

func getCharacterShipHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCharacterShipInput, GetCharacterShipOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterShipInput) (*mcpsdk.CallToolResult, GetCharacterShipOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterShipOutput{}, err
		}
		if !found {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		shipItemID, shipName, shipTypeID, err := getCharacterShip(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetCharacterShipOutput{}, fmt.Errorf("get character ship: %w", err)
		}
		return nil, GetCharacterShipOutput{CharacterID: in.CharacterID, ShipItemID: shipItemID, ShipName: shipName, ShipTypeID: shipTypeID}, nil
	}
}
```

**Design note on `get_character_online`:** rather than pass 3 loose pointer/bool values back and
forth like `get_character_location`'s 3-tuple, this one groups them into a small unexported-package
helper type `esi.CharacterOnlineStatus` defined in `esi/location.go` alongside the client methods,
since 4 return values (status pointer + 3 error-adjacent-looking pointers) is unwieldy as bare
multiple-return. This mirrors no existing precedent exactly, but is the same "small struct beats
long tuple" judgment call already implicit in every other multi-field ESI response in this
codebase. Task 2's implementer should define `CharacterOnlineStatus` in `esi/location.go` (not
`mcp/location.go`) since it's a shape derived from the ESI response, exported from the `esi`
package for `mcp` to consume — add it to the Task 1 code above:

```go
// CharacterOnlineStatus is CharacterOnline's return shape, grouped into a struct instead of a
// long multi-value return since 4 nullable/boolean fields is unwieldy as bare return values.
type CharacterOnlineStatus struct {
	Online     bool
	LastLogin  *time.Time
	LastLogout *time.Time
	Logins     *int64
}
```

(`CharacterOnline`'s signature becomes `func (c *Client) CharacterOnline(ctx context.Context) (*CharacterOnlineStatus, error)`, mapping fields from `esiapi.CharactersCharacterIdOnlineGet` inside the method body.)

`mcp/location_internal_test.go`: 6 test functions (happy path + unknown-character for each of the
3 tools), following the exact structure of `mcp/fittings_internal_test.go` /
`mcp/industry_internal_test.go` (swappable `var` override + `t.Cleanup` restore; `keyring.MockInit()`
+ temp-file `config.Store` for unknown-character cases).

## `mcp/server.go` / `mcp/server_test.go`

Register 3 tools after the `get_killmail` block: `get_character_location`, `get_character_online`,
`get_character_ship`. Update package doc comment. `server_test.go`'s expected-tools list: 132 → 135.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred thirty-two" → "one hundred thirty-five",
  append the 3 new tool names.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Location row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/location_test.go` (path/
query verification + response mapping), swappable-`var`-override tests for
`mcp/location_internal_test.go` (handler logic + characterExists gating), full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.
