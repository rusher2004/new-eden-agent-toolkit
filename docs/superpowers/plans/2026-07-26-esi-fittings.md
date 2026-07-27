# ESI Fittings (Fourteenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 3 endpoints of the "Fittings" ESI tag as a new `esi/fittings.go` facade and a new `mcp/fittings.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `FittingsAPI`, reusing the existing `describeError` helper. One new `mcp/` tool file with 3 private tools (`esi.NewClient` + existing `characterExists`) — a read (`list_character_fittings`), a create (`create_character_fitting`), and this project's first stand-alone DELETE write endpoint (`delete_character_fitting`), following the established verb-first write-tool naming convention.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `ab61983`) — no new branch, no new PR.
- Current tool count baseline: **96**. After this plan: **99**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`) — do not redeclare it.
- All 3 tools are private, character-scoped, "my own character" — no separate target-character param. `CharacterFittings`/`CreateCharacterFitting`/`DeleteCharacterFitting` take no explicit `characterID` parameter beyond what's baked into `Client` (`int64(c.characterID)`), matching the convention already used in `esi/access_list.go`, `esi/calendar.go`, `esi/faction_warfare.go`.
- `DeleteCharacterFitting` returns only `error` (no other return value) — mirrors `RespondToCalendarEvent`'s exact "write returns only error" shape (`esi/calendar.go`), since ESI's DELETE here returns no body.
- `create_character_fitting`/`delete_character_fitting` are verb-first-named write tools whose descriptions must state they change state on EVE's server, per the established write-tool convention (`respond_calendar_event`, `add_character_contacts`) — no confirmation parameter.
- No pagination on any of these 3 endpoints.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean). Implementers should still run these themselves as part of each task.

---

### Task 1: `esi/fittings.go` facade (3 methods)

**Files:**
- Create: `esi/fittings.go`
- Test: `esi/fittings_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `Client.api`/`Client.characterID` (`esi/client.go`).
- Produces: 3 methods on `*Client` for Task 2's `mcp/fittings.go` to call.

- [ ] **Step 1: Write `esi/fittings.go`**

```go
// Package esi's fittings.go covers all 3 of "Fittings"'s endpoints -- list, create, and delete a
// character's saved ship fittings. All 3 are private/"my own character" -- the generated request
// builders take characterId as a plain positional int64, which this facade passes as
// int64(c.characterID), never a caller-supplied parameter.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterFittings returns the client's own character's saved ship fittings. Not paginated.
func (c *Client) CharacterFittings(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error) {
	result, resp, err := c.api.FittingsAPI.GetCharactersCharacterIdFittings(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character fittings: %w", describeError(err, resp))
	}
	return result, nil
}

// CreateCharacterFitting saves a new fitting for the client's own character and returns its
// assigned fitting ID -- the only field ESI's response contains.
func (c *Client) CreateCharacterFitting(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error) {
	result, resp, err := c.api.FittingsAPI.PostCharactersCharacterIdFittings(ctx, int64(c.characterID)).
		PostCharactersCharacterIdFittingsRequest(esiapi.PostCharactersCharacterIdFittingsRequest{
			Description: description, Items: items, Name: name, ShipTypeId: shipTypeID,
		}).Execute()
	if err != nil {
		return 0, fmt.Errorf("create character fitting: %w", describeError(err, resp))
	}
	return result.FittingId, nil
}

// DeleteCharacterFitting deletes one of the client's own character's saved fittings. ESI's
// response has no body -- same shape as RespondToCalendarEvent (esi/calendar.go).
func (c *Client) DeleteCharacterFitting(ctx context.Context, fittingID int64) error {
	resp, err := c.api.FittingsAPI.DeleteCharactersCharacterIdFittingsFittingId(ctx, int64(c.characterID), fittingID).Execute()
	if err != nil {
		return fmt.Errorf("delete character fitting: %w", describeError(err, resp))
	}
	return nil
}
```

- [ ] **Step 2: Write `esi/fittings_test.go`**

Read `esi/calendar_test.go` first to copy its exact pattern for testing a write method with a request body and a write method returning only `error` (the PUT-based `RespondToCalendarEvent` test is the closest precedent for this file's POST/DELETE tests). Cover:

- `TestCharacterFittings_ParsesResponse` — serve a 1-element array `[{description, fitting_id, items: [{flag, quantity, type_id}], name, ship_type_id}]`, assert every field maps, including the nested item.
- `TestCharacterFittings_HTTPError` — serve a 404, assert the error is non-nil and wraps a description.
- `TestCreateCharacterFitting_SendsCorrectRequestBodyAndParsesResponse` — run a live `httptest.Server`, assert the request body the server received deserializes to the exact `{description, items, name, ship_type_id}` passed in (mirroring the precedent set by Calendar's PUT-body test and Assets' POST-array-body test), and assert the returned `fittingID` matches the served `{fitting_id}` response.
- `TestDeleteCharacterFitting_SendsNoBodyAndHandlesNoContentResponse` — run a live `httptest.Server` that returns 204 with no body, assert `DeleteCharacterFitting` returns a nil error and that the request the server received had no request body.
- `TestDeleteCharacterFitting_HTTPError` — serve a 404, assert the error is non-nil and wraps a description.

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run "CharacterFittings|CreateCharacterFitting|DeleteCharacterFitting" -v`
Expected: all tests PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/fittings.go esi/fittings_test.go
git commit -m "feat: add esi/fittings.go facade (project's first standalone DELETE write)"
```

---

### Task 2: `mcp/fittings.go` tools (3 tools), server registration, and docs sync

**Files:**
- Create: `mcp/fittings.go`
- Test: `mcp/fittings_internal_test.go`
- Modify: `mcp/server.go` (register 3 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 96 to 99)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Fittings row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "ninety-six" → "ninety-nine")
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: all 3 `esi/fittings.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent`.
- Produces: 3 `mcpsdk.ToolHandlerFor[...]`-returning functions for `mcp/server.go`: `listCharacterFittingsHandler`, `createCharacterFittingHandler`, `deleteCharacterFittingHandler` — all `(version, clientID string, store *config.Store)`.

- [ ] **Step 1: Write `mcp/fittings.go`**

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

// --- list_character_fittings (private) ---

type FittingItem struct {
	Flag     string `json:"flag"`
	Quantity int64  `json:"quantity"`
	TypeID   int64  `json:"typeId"`
}

type Fitting struct {
	FittingID   int64         `json:"fittingId"`
	ShipTypeID  int64         `json:"shipTypeId"`
	Description string        `json:"description"`
	Name        string        `json:"name"`
	Items       []FittingItem `json:"items"`
}

var listCharacterFittings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Fitting, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	fittings, err := client.CharacterFittings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Fitting, len(fittings))
	for i, f := range fittings {
		items := make([]FittingItem, len(f.Items))
		for j, it := range f.Items {
			items[j] = FittingItem{Flag: it.Flag, Quantity: it.Quantity, TypeID: it.TypeId}
		}
		out[i] = Fitting{FittingID: f.FittingId, ShipTypeID: f.ShipTypeId, Description: f.Description, Name: f.Name, Items: items}
	}
	return out, nil
}

type ListCharacterFittingsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterFittingsOutput struct {
	CharacterID int32     `json:"characterId"`
	Fittings    []Fitting `json:"fittings"`
}

func listCharacterFittingsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterFittingsInput, ListCharacterFittingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterFittingsInput) (*mcpsdk.CallToolResult, ListCharacterFittingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFittingsOutput{}, err
		}
		if !found {
			return nil, ListCharacterFittingsOutput{}, fmt.Errorf("list character fittings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		fittings, err := listCharacterFittings(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterFittingsOutput{}, fmt.Errorf("list character fittings: %w", err)
		}
		return nil, ListCharacterFittingsOutput{CharacterID: in.CharacterID, Fittings: fittings}, nil
	}
}

// --- create_character_fitting (private, write) ---

type FittingItemInput struct {
	Flag     string `json:"flag" jsonschema:"fitting location for the item, e.g. HiSlot0, LoSlot0, Cargo"`
	Quantity int64  `json:"quantity" jsonschema:"how many of this item to fit"`
	TypeID   int64  `json:"typeId" jsonschema:"the item's type ID"`
}

var createCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, description, name string, shipTypeID int64, items []FittingItemInput) (int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	esiItems := make([]esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, len(items))
	for i, it := range items {
		esiItems[i] = esiapi.PostCharactersCharacterIdFittingsRequestItemsInner{Flag: it.Flag, Quantity: it.Quantity, TypeId: it.TypeID}
	}
	return client.CreateCharacterFitting(ctx, description, esiItems, name, shipTypeID)
}

type CreateCharacterFittingInput struct {
	CharacterID int32              `json:"characterId" jsonschema:"the character ID to check"`
	Description string             `json:"description" jsonschema:"the fitting's description"`
	Name        string             `json:"name" jsonschema:"the fitting's name"`
	ShipTypeID  int64              `json:"shipTypeId" jsonschema:"the ship type ID this fitting is for"`
	Items       []FittingItemInput `json:"items" jsonschema:"the modules/items in this fitting"`
}
type CreateCharacterFittingOutput struct {
	CharacterID int32 `json:"characterId"`
	FittingID   int64 `json:"fittingId"`
}

func createCharacterFittingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[CreateCharacterFittingInput, CreateCharacterFittingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateCharacterFittingInput) (*mcpsdk.CallToolResult, CreateCharacterFittingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateCharacterFittingOutput{}, err
		}
		if !found {
			return nil, CreateCharacterFittingOutput{}, fmt.Errorf("create character fitting for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		fittingID, err := createCharacterFitting(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Description, in.Name, in.ShipTypeID, in.Items)
		if err != nil {
			return nil, CreateCharacterFittingOutput{}, fmt.Errorf("create character fitting: %w", err)
		}
		return nil, CreateCharacterFittingOutput{CharacterID: in.CharacterID, FittingID: fittingID}, nil
	}
}

// --- delete_character_fitting (private, write) ---

var deleteCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, fittingID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.DeleteCharacterFitting(ctx, fittingID)
}

type DeleteCharacterFittingInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	FittingID   int64 `json:"fittingId" jsonschema:"the fitting ID to delete"`
}
type DeleteCharacterFittingOutput struct {
	CharacterID int32 `json:"characterId"`
	FittingID   int64 `json:"fittingId"`
}

func deleteCharacterFittingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[DeleteCharacterFittingInput, DeleteCharacterFittingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteCharacterFittingInput) (*mcpsdk.CallToolResult, DeleteCharacterFittingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterFittingOutput{}, err
		}
		if !found {
			return nil, DeleteCharacterFittingOutput{}, fmt.Errorf("delete character fitting for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := deleteCharacterFitting(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FittingID); err != nil {
			return nil, DeleteCharacterFittingOutput{}, fmt.Errorf("delete character fitting: %w", err)
		}
		return nil, DeleteCharacterFittingOutput{CharacterID: in.CharacterID, FittingID: in.FittingID}, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Write `mcp/fittings_internal_test.go`**

Follow `mcp/corporation_internal_test.go`'s pattern (package-level `var` override + `t.Cleanup`; `keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). Cover:

- `TestListCharacterFittingsHandler_MapsFields` — stub returns a populated `[]Fitting` including at least one item, assert every field maps.
- `TestListCharacterFittingsHandler_UnknownCharacter`
- `TestCreateCharacterFittingHandler_MapsFields` — override the package-level `createCharacterFitting` var with a closure that captures its received `description`/`name`/`shipTypeID`/`items` arguments into local variables and returns a fixed `fittingID`. Call the handler with a specific `CreateCharacterFittingInput` (including at least one `FittingItemInput`), then assert: (1) the captured arguments match the input's `Description`/`Name`/`ShipTypeID`/`Items` exactly, proving the handler passes them through unchanged; (2) the output's `CharacterID`/`FittingID` match the input's `CharacterID` and the stub's returned `fittingID`.
- `TestCreateCharacterFittingHandler_UnknownCharacter`
- `TestDeleteCharacterFittingHandler_Succeeds` — stub `deleteCharacterFitting` to return nil, assert the output echoes `characterId`/`fittingId`.
- `TestDeleteCharacterFittingHandler_UnknownCharacter`

That's 6 test functions total.

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run "CharacterFitting" -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 3 tools in `mcp/server.go`**

Add these 3 `mcpsdk.AddTool` blocks immediately after the existing `list_fw_wars` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_fittings",
		Description: "List a logged-in EVE Online character's saved ship fittings.",
	}, listCharacterFittingsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "create_character_fitting",
		Description: "Save a new ship fitting for a logged-in character — this creates a new fitting on EVE's server.",
	}, createCharacterFittingHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "delete_character_fitting",
		Description: "Delete one of a logged-in character's saved ship fittings — this deletes the fitting on EVE's server.",
	}, deleteCharacterFittingHandler(version, clientID, store))

	return server
}
```

Update the package doc comment above `NewServer` to append the 3 new tool names to its running
list (ending in `..., list_fw_wars, list_character_fittings, create_character_fitting, and
delete_character_fitting registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 3 names to the `[]string{...}` literal, right after `"list_fw_wars",`:

```go
			"list_character_fittings", "create_character_fitting", "delete_character_fitting",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 99 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "ninety-six" to "ninety-nine".
- Append the 3 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Fittings row from `Planned` to `This plan`.

- [ ] **Step 8: Commit**

```bash
git add mcp/fittings.go mcp/fittings_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add fittings MCP tools (3 tools: list/create/delete)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff ab61983..HEAD` — the design spec commit forward).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
