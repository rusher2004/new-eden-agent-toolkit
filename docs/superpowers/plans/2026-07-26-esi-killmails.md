# ESI Killmails (Twentieth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 3 endpoints of the "Killmails" ESI tag as a new `esi/killmails.go` facade and a new `mcp/killmails.go` tool file, registered on the shared MCP server.

**Architecture:** One new `esi/` facade file wrapping the generated `KillmailsAPI`, reusing the existing `describeError` and `totalPagesFromHeader` helpers. One new `mcp/` tool file with 2 private page-paginated tools and 1 public tool. The public killmail-detail tool introduces a self-referential `KillmailItem` type to mirror ESI's own one-level-deep container-item nesting, and absorbs a reversed-parameter-order quirk in the generated client (mirroring the precedent set by `CorporationContractBids` in the Contracts batch).

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `5413350`) — no new branch, no new PR.
- Current tool count baseline: **129**. After this plan: **132**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`). The 2 paginated methods use the existing `totalPagesFromHeader` (`esi/contracts.go`) — do not redeclare either.
- `CharacterKillmailsRecent`/`CorporationKillmailsRecent` both return `[]esiapi.CharactersCharacterIdKillmailsRecentGetInner` — the SAME generated model — confirmed from the generated `Execute()` signatures. Do not invent a separate corp-specific type.
- **Verified reversed-parameter-order quirk**: `GetKillmailsKillmailIdKillmailHash`'s generated signature is `(ctx, killmailHash string, killmailId int64)` — hash first, ID second — reversed from the URL path and from `KillmailDetail`'s own `(killmailID, killmailHash)` signature. `KillmailDetail` absorbs this by swapping internally when calling the generated function, matching the `CorporationContractBids` precedent (`esi/contracts.go`).
- `KillmailItem` (mcp layer) is deliberately self-referential (`Items []KillmailItem`) to mirror ESI's own one-level-deep container nesting — the innermost generated type has no further `Items` field, so the mapper leaves the deepest level's `Items` empty rather than inventing a second, near-duplicate type.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean). Implementers should still run these themselves as part of each task.

---

### Task 1: `esi/killmails.go` facade (3 methods)

**Files:**
- Create: `esi/killmails.go`
- Test: `esi/killmails_test.go`

**Interfaces:**
- Consumes: `describeError(err error, resp *http.Response) error` (`esi/error.go`), `totalPagesFromHeader(resp *http.Response) int32` (`esi/contracts.go`), `Client.api`/`Client.characterID` (`esi/client.go`).
- Produces: 3 methods on `*Client` for Task 2's `mcp/killmails.go` to call.

- [ ] **Step 1: Write `esi/killmails.go`**

```go
// Package esi's killmails.go covers all 3 of "Killmails"'s endpoints: character/corp "recent
// killmails" listings (both page-paginated, both returning the SAME generated model -- verified
// from source, not a project typo) and a public killmail-detail lookup.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterKillmailsRecent returns the client's own character's recent killmail references
// (ID+hash pairs -- fetch full detail via KillmailDetail). page=0 omits the page query parameter.
func (c *Client) CharacterKillmailsRecent(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	req := c.api.KillmailsAPI.GetCharactersCharacterIdKillmailsRecent(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character killmails recent: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationKillmailsRecent returns corporationID's recent killmail references. The
// authenticated character must hold sufficient corp roles -- ESI itself returns 403 otherwise.
// Returns the exact same generated model as CharacterKillmailsRecent -- verified from the
// generated Execute() signature, not assumed. page=0 omits the page query parameter.
func (c *Client) CorporationKillmailsRecent(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	req := c.api.KillmailsAPI.GetCorporationsCorporationIdKillmailsRecent(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation killmails recent: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// KillmailDetail returns the full detail for one killmail. Public -- the hash itself functions as
// the access token ESI requires, no character auth needed.
//
// Quirk: the generated function's own argument order is (killmailHash, killmailId) -- reversed
// from the URL path (/killmails/{killmail_id}/{killmail_hash}/) and from this method's own
// (killmailID, killmailHash) signature. This facade normalizes to (killmailID, killmailHash) to
// match the URL and this project's tool input order, swapping internally -- same pattern as
// CorporationContractBids (esi/contracts.go).
func (c *Client) KillmailDetail(ctx context.Context, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
	result, resp, err := c.api.KillmailsAPI.GetKillmailsKillmailIdKillmailHash(ctx, killmailHash, killmailID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get killmail detail: %w", describeError(err, resp))
	}
	return result, nil
}
```

- [ ] **Step 2: Write `esi/killmails_test.go`**

Read `esi/contracts_test.go` first to copy its exact pagination-omit/set test pattern and its parameter-order-verification test style (used for `CorporationContractBids`) — this batch's `KillmailDetail` test should follow that same approach. Cover:

- `TestCharacterKillmailsRecent_ParsesResponseAndPageOmittedWhenZero` — serve a 1-element array `[{killmail_hash, killmail_id}]`, assert both fields map; assert no `page` param when `page=0`.
- `TestCharacterKillmailsRecent_PageSentWhenSetAndTotalPages` — call with `page=2`, assert query string contains `page=2`, plus `X-Pages: 3` response header round-trips to `totalPages == 3`.
- `TestCharacterKillmailsRecent_HTTPError` — serve a 404, assert error non-nil.
- `TestCorporationKillmailsRecent_ParsesResponseAndTotalPages` — serve a 1-element array plus `X-Pages: 1`, assert mapping and `totalPages == 1`.
- `TestKillmailDetail_ParsesResponseAndUsesCorrectArgumentOrder` — serve a full fixture: `{attackers: [{alliance_id, character_id, corporation_id, damage_done, faction_id, final_blow, security_status, ship_type_id, weapon_type_id}]}` (all fields populated on at least one attacker), `killmail_id`, `killmail_time`, `moon_id`, `solar_system_id`, `victim: {alliance_id, character_id, corporation_id, damage_taken, faction_id, items: [{flag, item_type_id, items: [{flag, item_type_id, quantity_destroyed, quantity_dropped, singleton}], quantity_destroyed, quantity_dropped, singleton}], position: {x, y, z}, ship_type_id}`, `war_id` — assert EVERY field maps correctly including the nested `Items[0].Items[0]` container entry, AND assert the request path the test server received resolves to `/killmails/{killmailID}/{killmailHash}/` with the correct ID/hash in the correct positions (proving the argument-order swap is correct, not inverted).
- `TestKillmailDetail_HTTPError` — serve a 404, assert error non-nil.

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run "CharacterKillmailsRecent|CorporationKillmailsRecent|KillmailDetail" -v`
Expected: all tests PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add esi/killmails.go esi/killmails_test.go
git commit -m "feat: add esi/killmails.go facade (reversed argument order, one-level item nesting)"
```

---

### Task 2: `mcp/killmails.go` tools (3 tools), server registration, and docs sync

**Files:**
- Create: `mcp/killmails.go`
- Test: `mcp/killmails_internal_test.go`
- Modify: `mcp/server.go` (register 3 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 129 to 132)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Killmails row: `Planned` → `This plan`)
- Modify: `CLAUDE.md` (tool count "one hundred twenty-nine" → "one hundred thirty-two")
- Modify: `README.md` (same edit, mirrored)

**Interfaces:**
- Consumes: all 3 `esi/killmails.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`); `sso.UserAgent`.
- Produces: 3 `mcpsdk.ToolHandlerFor[...]`-returning functions: `listCharacterKillmailsRecentHandler`, `listCorporationKillmailsRecentHandler` (both `(version, clientID string, store *config.Store)`), `getKillmailHandler` (`(version string)` only).

- [ ] **Step 1: Write `mcp/killmails.go`**

```go
package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// KillmailRef is shared by list_character_killmails_recent and list_corporation_killmails_recent
// since both endpoints return the exact same generated model.
type KillmailRef struct {
	KillmailID   int64  `json:"killmailId"`
	KillmailHash string `json:"killmailHash"`
}

func toKillmailRefs(refs []esiapi.CharactersCharacterIdKillmailsRecentGetInner) []KillmailRef {
	out := make([]KillmailRef, len(refs))
	for i, r := range refs {
		out[i] = KillmailRef{KillmailID: r.KillmailId, KillmailHash: r.KillmailHash}
	}
	return out
}

// --- list_character_killmails_recent (private, paginated) ---

var listCharacterKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterKillmailsRecent(ctx, page)
}

type ListCharacterKillmailsRecentInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterKillmailsRecentOutput struct {
	CharacterID int32         `json:"characterId"`
	Page        int32         `json:"page"`
	TotalPages  int32         `json:"totalPages"`
	Killmails   []KillmailRef `json:"killmails"`
}

func listCharacterKillmailsRecentHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterKillmailsRecentInput, ListCharacterKillmailsRecentOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterKillmailsRecentInput) (*mcpsdk.CallToolResult, ListCharacterKillmailsRecentOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterKillmailsRecentOutput{}, err
		}
		if !found {
			return nil, ListCharacterKillmailsRecentOutput{}, fmt.Errorf("list character killmails recent for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		refs, totalPages, err := listCharacterKillmailsRecent(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListCharacterKillmailsRecentOutput{}, fmt.Errorf("list character killmails recent: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCharacterKillmailsRecentOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Killmails: toKillmailRefs(refs)}, nil
	}
}

// --- list_corporation_killmails_recent (private, corp-scoped, paginated) ---

var listCorporationKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationKillmailsRecent(ctx, corporationID, page)
}

type ListCorporationKillmailsRecentInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationKillmailsRecentOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Killmails     []KillmailRef `json:"killmails"`
}

func listCorporationKillmailsRecentHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationKillmailsRecentInput, ListCorporationKillmailsRecentOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationKillmailsRecentInput) (*mcpsdk.CallToolResult, ListCorporationKillmailsRecentOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationKillmailsRecentOutput{}, err
		}
		if !found {
			return nil, ListCorporationKillmailsRecentOutput{}, fmt.Errorf("list corporation killmails recent for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		refs, totalPages, err := listCorporationKillmailsRecent(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationKillmailsRecentOutput{}, fmt.Errorf("list corporation killmails recent: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationKillmailsRecentOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Killmails: toKillmailRefs(refs)}, nil
	}
}

// --- get_killmail (public) ---

type KillmailAttacker struct {
	AllianceID     *int64  `json:"allianceId,omitempty"`
	CharacterID    *int64  `json:"characterId,omitempty"`
	CorporationID  *int64  `json:"corporationId,omitempty"`
	DamageDone     int64   `json:"damageDone"`
	FactionID      *int64  `json:"factionId,omitempty"`
	FinalBlow      bool    `json:"finalBlow"`
	SecurityStatus float64 `json:"securityStatus"`
	ShipTypeID     *int64  `json:"shipTypeId,omitempty"`
	WeaponTypeID   *int64  `json:"weaponTypeId,omitempty"`
}

// KillmailItem is self-referential to mirror ESI's own one-level-deep container nesting (items
// destroyed inside a destroyed container). The innermost generated type has no further nested
// items field, so the mapper simply leaves Items empty when mapping that level.
type KillmailItem struct {
	Flag              int64          `json:"flag"`
	ItemTypeID        int64          `json:"itemTypeId"`
	Singleton         int64          `json:"singleton"`
	QuantityDestroyed *int64         `json:"quantityDestroyed,omitempty"`
	QuantityDropped   *int64         `json:"quantityDropped,omitempty"`
	Items             []KillmailItem `json:"items,omitempty"`
}

type KillmailPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type KillmailVictim struct {
	AllianceID    *int64            `json:"allianceId,omitempty"`
	CharacterID   *int64            `json:"characterId,omitempty"`
	CorporationID *int64            `json:"corporationId,omitempty"`
	DamageTaken   int64             `json:"damageTaken"`
	FactionID     *int64            `json:"factionId,omitempty"`
	ShipTypeID    int64             `json:"shipTypeId"`
	Items         []KillmailItem    `json:"items,omitempty"`
	Position      *KillmailPosition `json:"position,omitempty"`
}

func toKillmailInnerItems(items []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInnerItemsInner) []KillmailItem {
	out := make([]KillmailItem, len(items))
	for i, it := range items {
		out[i] = KillmailItem{
			Flag: it.Flag, ItemTypeID: it.ItemTypeId, Singleton: it.Singleton,
			QuantityDestroyed: it.QuantityDestroyed, QuantityDropped: it.QuantityDropped,
		}
	}
	return out
}

func toKillmailItems(items []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInner) []KillmailItem {
	out := make([]KillmailItem, len(items))
	for i, it := range items {
		out[i] = KillmailItem{
			Flag: it.Flag, ItemTypeID: it.ItemTypeId, Singleton: it.Singleton,
			QuantityDestroyed: it.QuantityDestroyed, QuantityDropped: it.QuantityDropped,
			Items: toKillmailInnerItems(it.Items),
		}
	}
	return out
}

var getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.KillmailDetail(ctx, killmailID, killmailHash)
}

type GetKillmailInput struct {
	KillmailID   int64  `json:"killmailId" jsonschema:"the killmail ID"`
	KillmailHash string `json:"killmailHash" jsonschema:"the killmail hash, from a killmail listing"`
}
type GetKillmailOutput struct {
	KillmailID    int64              `json:"killmailId"`
	KillmailTime  time.Time          `json:"killmailTime"`
	SolarSystemID int64              `json:"solarSystemId"`
	MoonID        *int64             `json:"moonId,omitempty"`
	WarID         *int64             `json:"warId,omitempty"`
	Attackers     []KillmailAttacker `json:"attackers"`
	Victim        KillmailVictim     `json:"victim"`
}

func getKillmailHandler(version string) mcpsdk.ToolHandlerFor[GetKillmailInput, GetKillmailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetKillmailInput) (*mcpsdk.CallToolResult, GetKillmailOutput, error) {
		detail, err := getKillmail(ctx, sso.UserAgent(version), in.KillmailID, in.KillmailHash)
		if err != nil {
			return nil, GetKillmailOutput{}, fmt.Errorf("get killmail: %w", err)
		}
		attackers := make([]KillmailAttacker, len(detail.Attackers))
		for i, a := range detail.Attackers {
			attackers[i] = KillmailAttacker{
				AllianceID: a.AllianceId, CharacterID: a.CharacterId, CorporationID: a.CorporationId,
				DamageDone: a.DamageDone, FactionID: a.FactionId, FinalBlow: a.FinalBlow,
				SecurityStatus: a.SecurityStatus, ShipTypeID: a.ShipTypeId, WeaponTypeID: a.WeaponTypeId,
			}
		}
		var position *KillmailPosition
		if detail.Victim.Position != nil {
			position = &KillmailPosition{X: detail.Victim.Position.X, Y: detail.Victim.Position.Y, Z: detail.Victim.Position.Z}
		}
		return nil, GetKillmailOutput{
			KillmailID: detail.KillmailId, KillmailTime: detail.KillmailTime, SolarSystemID: detail.SolarSystemId,
			MoonID: detail.MoonId, WarID: detail.WarId, Attackers: attackers,
			Victim: KillmailVictim{
				AllianceID: detail.Victim.AllianceId, CharacterID: detail.Victim.CharacterId,
				CorporationID: detail.Victim.CorporationId, DamageTaken: detail.Victim.DamageTaken,
				FactionID: detail.Victim.FactionId, ShipTypeID: detail.Victim.ShipTypeId,
				Items: toKillmailItems(detail.Victim.Items), Position: position,
			},
		}, nil
	}
}
```

This exact file has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Write `mcp/killmails_internal_test.go`**

Follow `mcp/industry_internal_test.go`'s pattern (package-level `var` override + `t.Cleanup`; `keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). Cover:

- `TestListCharacterKillmailsRecentHandler_MapsFieldsAndDefaultsPageTo1` — stub returns a populated `[]esiapi.CharactersCharacterIdKillmailsRecentGetInner` plus a `totalPages`, assert every field maps and `Page` defaults to 1.
- `TestListCharacterKillmailsRecentHandler_UnknownCharacter`
- `TestListCorporationKillmailsRecentHandler_MapsFieldsAndDefaultsPageTo1`
- `TestListCorporationKillmailsRecentHandler_UnknownCharacter`
- `TestGetKillmailHandler_MapsFields` — stub `getKillmail` to return a fully-populated `*esiapi.KillmailsKillmailIdKillmailHashGet` including at least one attacker with every optional field set, a victim with a non-empty `Items` array whose first entry itself has a non-empty nested `Items` array, and a non-nil `Position` — assert every field maps at every nesting level, including the innermost item's fields.
- `TestGetKillmailHandler_HandlesNilOptionalFields` — stub returns a detail with `MoonId`, `WarId`, `Victim.Position` all nil and `Victim.Items` empty, assert the output correctly reflects all of these as nil/empty (no panics on nil pointer dereference).

That's 6 test functions total (2 tool-pairs × happy-path + unknown-character = 4, plus 2 tests for `get_killmail`, which is public and has no unknown-character test but needs both a fully-populated and a nil-fields test given its structural complexity).

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run "CharacterKillmailsRecent|CorporationKillmailsRecent|Killmail" -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 3 tools in `mcp/server.go`**

Add these 3 `mcpsdk.AddTool` blocks immediately after the existing `list_insurance_prices` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_killmails_recent",
		Description: "List a logged-in EVE Online character's recent killmail references.",
	}, listCharacterKillmailsRecentHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_killmails_recent",
		Description: "List a corporation's recent killmail references, using a logged-in character's roles to authorize.",
	}, listCorporationKillmailsRecentHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_killmail",
		Description: "Get the full details of a killmail, given its ID and hash (public data, no character required).",
	}, getKillmailHandler(version))

	return server
}
```

Update the package doc comment above `NewServer` to append the 3 new tool names to its running
list (ending in `..., list_insurance_prices, list_character_killmails_recent,
list_corporation_killmails_recent, and get_killmail registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 3 names to the `[]string{...}` literal, right after `"list_insurance_prices",`:

```go
			"list_character_killmails_recent", "list_corporation_killmails_recent", "get_killmail",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 132 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "one hundred twenty-nine" to "one hundred thirty-two".
- Append the 3 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Killmails row from `Planned` to `This plan`.

- [ ] **Step 8: Commit**

```bash
git add mcp/killmails.go mcp/killmails_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add killmails MCP tools (3 tools: recent listings + public detail)"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff 5413350..HEAD` — the design spec commit forward).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
