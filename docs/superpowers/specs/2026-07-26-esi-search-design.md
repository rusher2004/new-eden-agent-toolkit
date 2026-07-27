# ESI Search — Design Spec (twenty-eighth batch)

**Batch:** Search tag, 1 endpoint, private ("my own character"), no pagination.

## Endpoint (verified against installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and its
`esi/api/openapi.yaml` security requirements)

`GetCharactersCharacterIdSearch(ctx, characterId int64) ApiGetCharactersCharacterIdSearchRequest` →
`.Categories([]string).Search(string)` (both **required** at the ESI-request level — verified: the
generated `Execute` body errors if either is unset, and `openapi.yaml` marks `categories`
`required: true` with `minItems: 1, maxItems: 11` and an enum of 11 valid values — `agent`,
`alliance`, `character`, `constellation`, `corporation`, `faction`, `inventory_type`, `region`,
`solar_system`, `station`, `structure`; `search` requires at least 3 characters) plus an optional
`.Strict(bool)` → `.Execute() (*CharactersCharacterIdSearchGet{Agent, Alliance, Character,
Constellation, Corporation, Faction, InventoryType, Region, SolarSystem, Station, Structure
[]int64, all omitempty}, *http.Response, error)`.

Private — verified: requires OAuth2 scope `esi-search.search_structures.v1` — "my own character"
style, `characterId` passed as `int64(c.characterID)`, never caller-supplied. Not paginated — no
`.Page()` setter. Response is a flat struct of 11 optional `[]int64` fields, one per searchable
category; no recursion.

## `esi/search.go`

```go
// Package esi's search.go covers the single "Search" endpoint -- search for entities matching a
// sub-string, scoped to the client's own character (private, "my own character").
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterSearch searches for entities matching search, restricted to categories (1-11 of:
// agent, alliance, character, constellation, corporation, faction, inventory_type, region,
// solar_system, station, structure -- verified against esi/api/openapi.yaml's enum). Both
// categories and search are required at the ESI-request level -- verified from source: the
// generated Execute body errors if either is unset. strict=false requests a substring match;
// strict=true requires an exact match.
func (c *Client) CharacterSearch(ctx context.Context, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
	req := c.api.SearchAPI.GetCharactersCharacterIdSearch(ctx, int64(c.characterID)).Categories(categories).Search(search)
	if strict {
		req = req.Strict(strict)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get character search: %w", describeError(err, resp))
	}
	return result, nil
}
```

`esi/search_test.go`: 2 tests (success + HTTP error), asserting the query string carries
`categories`/`search` and that omitting `strict` (false) omits the query param.

## `mcp/search.go`

One private tool, `search_character`, following `mcp/fittings.go`'s shape. `SearchResults` mirrors
the flat 11-field response 1:1; no shared/reusable sub-types needed (nothing nests).

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- search_character (private) ---

type SearchResults struct {
	Agent         []int64 `json:"agent,omitempty"`
	Alliance      []int64 `json:"alliance,omitempty"`
	Character     []int64 `json:"character,omitempty"`
	Constellation []int64 `json:"constellation,omitempty"`
	Corporation   []int64 `json:"corporation,omitempty"`
	Faction       []int64 `json:"faction,omitempty"`
	InventoryType []int64 `json:"inventoryType,omitempty"`
	Region        []int64 `json:"region,omitempty"`
	SolarSystem   []int64 `json:"solarSystem,omitempty"`
	Station       []int64 `json:"station,omitempty"`
	Structure     []int64 `json:"structure,omitempty"`
}

var searchCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent string, categories []string, search string, strict bool) (*esiapi.CharactersCharacterIdSearchGet, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterSearch(ctx, categories, search, strict)
}

type SearchCharacterInput struct {
	CharacterID int32    `json:"characterId" jsonschema:"the character ID to check"`
	Categories  []string `json:"categories" jsonschema:"1-11 of: agent, alliance, character, constellation, corporation, faction, inventory_type, region, solar_system, station, structure"`
	Search      string   `json:"search" jsonschema:"the search string, at least 3 characters"`
	Strict      bool     `json:"strict,omitempty" jsonschema:"require an exact match instead of a substring match; omit for false"`
}
type SearchCharacterOutput struct {
	CharacterID int32         `json:"characterId"`
	Results     SearchResults `json:"results"`
}

func searchCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[SearchCharacterInput, SearchCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SearchCharacterInput) (*mcpsdk.CallToolResult, SearchCharacterOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SearchCharacterOutput{}, err
		}
		if !found {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		results, err := searchCharacter(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Categories, in.Search, in.Strict)
		if err != nil {
			return nil, SearchCharacterOutput{}, fmt.Errorf("search character: %w", err)
		}
		return nil, SearchCharacterOutput{CharacterID: in.CharacterID, Results: SearchResults{
			Agent: results.Agent, Alliance: results.Alliance, Character: results.Character,
			Constellation: results.Constellation, Corporation: results.Corporation, Faction: results.Faction,
			InventoryType: results.InventoryType, Region: results.Region, SolarSystem: results.SolarSystem,
			Station: results.Station, Structure: results.Structure,
		}}, nil
	}
}
```

`SearchResults` and its Input/Output types are flat, non-recursive. Already sandbox-verified
against a live `mcpsdk.AddTool` registration test with zero schema-generator errors.

`mcp/search_internal_test.go`: 2 tests (`MapsFields` + `UnknownCharacter`), following
`mcp/fittings_internal_test.go`'s swappable-var-override pattern.

## `mcp/server.go` / `mcp/server_test.go`

Register 1 tool after the `get_route` block (once Routes lands). Update package doc comment.
`server_test.go`'s expected-tools list: 166 → 167.

## Docs sync

- `CLAUDE.md` / `README.md`: tool count "one hundred sixty-six" → "one hundred sixty-seven",
  append `search_character`.
- `docs/reference/esi-coverage-roadmap.md`: flip only the Search row from "Planned" to "This plan".

## Testing approach

Same as every prior batch: `httptest.Server` round-trip tests for `esi/search_test.go`,
swappable-`var`-override tests for `mcp/search_internal_test.go`, full
`go build ./... && go vet ./... && gofmt -l . && go test ./... -race` before commit.

Given this batch's size (1 endpoint), Task 1 and Task 2 are combined into a single implementation
task, following the precedent set by Incursions/Insurance/Routes.
