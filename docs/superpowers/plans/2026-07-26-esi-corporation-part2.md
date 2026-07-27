# ESI Corporation Part 2 (Tenth Batch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the Corporation tag by adding its remaining 8 endpoints (assets/infrastructure) to the existing `esi/corporation.go` and `mcp/corporation.go` files, registered on the shared MCP server.

**Architecture:** Append 8 methods to the existing `esi/corporation.go` facade (same file — same ESI tag as Part 1, per the project's one-file-per-tag convention) reusing the existing `describeError`/`totalPagesFromHeader` helpers. Append 8 tool handlers to the existing `mcp/corporation.go` (all private — no public endpoints in this half), reusing the existing `characterExists`/`ErrCharacterNotFound` helpers from `mcp/character.go`.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk`.

## Global Constraints

- This plan builds directly on the shared PR branch `worktree-esi-access-list-activities` (current HEAD `28931e6`) — no new branch, no new PR.
- Current tool count baseline: **71**. After this plan: **79**.
- Every `esi/` method's `.Execute()` call must route its error and `*http.Response` through the existing `describeError` (`esi/error.go`). Every paginated method uses the existing `totalPagesFromHeader` (`esi/contracts.go`). Neither is redeclared.
- All 8 endpoints in this batch are **private** — every tool takes `characterId` and is existence-checked via the existing `characterExists(store *config.Store, characterID int32) (bool, error)` helper (`mcp/character.go`), returning `fmt.Errorf("<action> for character %d: %w", characterID, ErrCharacterNotFound)` on not-found.
- 6 of the 8 are paginated (`page int32`, `0` omits `.Page()`, output defaults `page` to 1 when input is 0): blueprints, container logs, medals, medals issued, starbases, structures. 2 are not paginated: facilities, and the single-starbase detail lookup.
- `get_corporation_starbase`'s `systemId` is a **required** input (not `page`-style optional) — the facade method always calls `.SystemId(systemID)` unconditionally.
- All code below has already been verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` module in an isolated sandbox (`go build`/`go vet`/`gofmt -l` all clean). Implementers should still run these themselves as part of each task.
- Per the design spec, this batch completes the Corporation tag — the docs task must flip `docs/reference/esi-coverage-roadmap.md`'s Corporation row from `Planned` to `This plan`.

---

### Task 1: Append 8 methods to `esi/corporation.go`

**Files:**
- Modify: `esi/corporation.go` (append methods; update the package doc comment listing the file's covered endpoints if one exists)
- Modify: `esi/corporation_test.go` (append tests)

**Interfaces:**
- Consumes: `describeError`, `totalPagesFromHeader` (existing, same file/package as Part 1).
- Produces: 8 new methods on `*Client` for Task 2 to call.

- [ ] **Step 1: Append these 8 methods to the end of `esi/corporation.go`** (no new imports needed — `context`, `fmt`, `esiapi` are already imported)

```go
// CorporationBlueprints returns corporationID's blueprints. page=0 omits the page query parameter.
func (c *Client) CorporationBlueprints(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdBlueprints(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation blueprints: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationContainerLogs returns corporationID's container access logs. page=0 omits the page
// query parameter.
func (c *Client) CorporationContainerLogs(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdContainersLogs(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation container logs: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationFacilities returns corporationID's facilities. Not paginated.
func (c *Client) CorporationFacilities(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdFacilities(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation facilities: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMedals returns corporationID's defined medals. page=0 omits the page query parameter.
func (c *Client) CorporationMedals(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdMedals(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation medals: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMedalsIssued returns medals corporationID has issued to members. page=0 omits the
// page query parameter.
func (c *Client) CorporationMedalsIssued(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsIssuedGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdMedalsIssued(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation medals issued: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationStarbases returns corporationID's starbases (POSes). page=0 omits the page query
// parameter.
func (c *Client) CorporationStarbases(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdStarbases(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation starbases: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationStarbaseDetail returns detail for one starbase. systemID is required by ESI to
// resolve the request (verified from the generated request builder's SystemId method) -- always
// sent, unlike page which is optionally omitted.
func (c *Client) CorporationStarbaseDetail(ctx context.Context, corporationID int64, starbaseID int64, systemID int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdStarbasesStarbaseId(ctx, corporationID, starbaseID).SystemId(systemID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation starbase detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationStructures returns corporationID's structures. page=0 omits the page query parameter.
func (c *Client) CorporationStructures(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdStructures(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation structures: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
```

- [ ] **Step 2: Append tests to `esi/corporation_test.go`**

Follow the file's existing pattern (httptest.Server + `ExportedNewPublicClientWithOptions`). Cover:

- `TestCorporationBlueprints_ParsesResponse` — serve one entry with all 8 required fields populated, assert every field maps.
- `TestCorporationBlueprints_PageZeroOmitsPageParam` — call with `page=0`, assert no `page` query param sent.
- `TestCorporationContainerLogs_ParsesResponse` — serve one entry with all optional fields populated (`new_config_bitmask`, `old_config_bitmask`, `password_type`, `quantity`, `type_id`), assert every pointer field is non-nil and correct.
- `TestCorporationContainerLogs_ParsesResponseAndTotalPages` — serve plus `X-Pages: 2`, assert `totalPages == 2`.
- `TestCorporationFacilities_ParsesResponse` — serve `[{facility_id, system_id, type_id}]`, assert mapping.
- `TestCorporationMedals_ParsesResponseAndTotalPages` — serve `[{created_at, creator_id, description, medal_id, title}]` plus `X-Pages: 1`, assert mapping and `totalPages == 1`.
- `TestCorporationMedalsIssued_ParsesResponseAndTotalPages` — serve `[{character_id, issued_at, issuer_id, medal_id, reason, status}]` plus `X-Pages: 1`, assert mapping.
- `TestCorporationStarbases_ParsesResponseAndTotalPages` — serve one entry with all optional fields populated (`moon_id`, `onlined_since`, `reinforced_until`, `state`, `unanchor_at`), assert all pointer fields non-nil, plus `X-Pages: 3`, assert `totalPages == 3`.
- `TestCorporationStarbaseDetail_ParsesResponseAndSendsSystemId` — serve a full `CorporationsCorporationIdStarbasesStarbaseIdGet` body (including a non-empty `fuels` array), assert every field maps (including `Fuels[0].Quantity`/`.TypeId`), AND assert the request the test server received has `system_id` in its query string equal to the value passed in — this is the one endpoint in the whole file where a "required-looking" parameter must ALWAYS appear, unlike `page`.
- `TestCorporationStructures_ParsesResponseAndTotalPages` — serve one entry with all optional fields populated (`fuel_expires`, `name`, `next_reinforce_apply`, `next_reinforce_hour`, `reinforce_hour`, `state_timer_end`, `state_timer_start`, `unanchors_at`) and a non-empty `services` array, assert every field including `Services[0].Name`/`.State`, plus `X-Pages: 2`, assert `totalPages == 2`.
- At least one HTTP-error test among the 8 (following the file's existing convention of not needing one per method).

- [ ] **Step 3: Run tests**

Run: `go test ./esi/... -run Corporation -v`
Expected: all tests (Part 1's 16 plus these ~10 new ones) PASS.

- [ ] **Step 4: Build, vet, format**

Run: `go build ./... && go vet ./... && gofmt -l .`
Expected: clean, `gofmt -l .` prints nothing.

- [ ] **Step 5: Commit**

```bash
git add esi/corporation.go esi/corporation_test.go
git commit -m "feat: extend esi/corporation.go facade (Part 2, 8 more read methods)"
```

---

### Task 2: Append 8 tools to `mcp/corporation.go`, server registration, and docs sync

**Files:**
- Modify: `mcp/corporation.go` (append 8 tool handlers)
- Modify: `mcp/corporation_internal_test.go` (append tests)
- Modify: `mcp/server.go` (register 8 new tools; doc comment tool list grows)
- Modify: `mcp/server_test.go` (`TestServer_AllToolsRegistered`'s expected-tools list grows from 71 to 79)
- Modify: `CLAUDE.md` (tool count "seventy-one" → "seventy-nine", tool list gains 8 names)
- Modify: `README.md` (same two edits, mirrored)
- Modify: `docs/reference/esi-coverage-roadmap.md` (Corporation row: `Planned` → `This plan`)

**Interfaces:**
- Consumes: all 8 new `esi/corporation.go` methods from Task 1; `characterExists`, `ErrCharacterNotFound` (`mcp/character.go`, already imported/used in `mcp/corporation.go`); `sso.UserAgent`.
- Produces: 8 more `mcpsdk.ToolHandlerFor[...]`-returning functions for `mcp/server.go`: `listCorporationBlueprintsHandler`, `listCorporationContainerLogsHandler`, `listCorporationFacilitiesHandler`, `listCorporationMedalsHandler`, `listCorporationMedalsIssuedHandler`, `listCorporationStarbasesHandler`, `getCorporationStarbaseHandler`, `listCorporationStructuresHandler` — all with signature `(version, clientID string, store *config.Store)`.

- [ ] **Step 1: Append these 8 tool blocks to the end of `mcp/corporation.go`** (no new imports needed)

```go
// --- list_corporation_blueprints (private, paginated) ---

type CorporationBlueprint struct {
	ItemID             int64  `json:"itemId"`
	LocationFlag       string `json:"locationFlag"`
	LocationID         int64  `json:"locationId"`
	MaterialEfficiency int64  `json:"materialEfficiency"`
	Quantity           int64  `json:"quantity"`
	Runs               int64  `json:"runs"`
	TimeEfficiency     int64  `json:"timeEfficiency"`
	TypeID             int64  `json:"typeId"`
}

var listCorporationBlueprints = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationBlueprint, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	bps, totalPages, err := client.CorporationBlueprints(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CorporationBlueprint, len(bps))
	for i, b := range bps {
		out[i] = CorporationBlueprint{
			ItemID: b.ItemId, LocationFlag: b.LocationFlag, LocationID: b.LocationId,
			MaterialEfficiency: b.MaterialEfficiency, Quantity: b.Quantity, Runs: b.Runs,
			TimeEfficiency: b.TimeEfficiency, TypeID: b.TypeId,
		}
	}
	return out, totalPages, nil
}

type ListCorporationBlueprintsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationBlueprintsOutput struct {
	CharacterID   int32                  `json:"characterId"`
	CorporationID int64                  `json:"corporationId"`
	Page          int32                  `json:"page"`
	TotalPages    int32                  `json:"totalPages"`
	Blueprints    []CorporationBlueprint `json:"blueprints"`
}

func listCorporationBlueprintsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationBlueprintsInput, ListCorporationBlueprintsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationBlueprintsInput) (*mcpsdk.CallToolResult, ListCorporationBlueprintsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationBlueprintsOutput{}, err
		}
		if !found {
			return nil, ListCorporationBlueprintsOutput{}, fmt.Errorf("list corporation blueprints for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		bps, totalPages, err := listCorporationBlueprints(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationBlueprintsOutput{}, fmt.Errorf("list corporation blueprints: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationBlueprintsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Blueprints: bps}, nil
	}
}

// --- list_corporation_container_logs (private, paginated) ---

type ContainerLog struct {
	Action           string    `json:"action"`
	CharacterID      int64     `json:"characterId"`
	ContainerID      int64     `json:"containerId"`
	ContainerTypeID  int64     `json:"containerTypeId"`
	LocationFlag     string    `json:"locationFlag"`
	LocationID       int64     `json:"locationId"`
	LoggedAt         time.Time `json:"loggedAt"`
	NewConfigBitmask *int64    `json:"newConfigBitmask,omitempty"`
	OldConfigBitmask *int64    `json:"oldConfigBitmask,omitempty"`
	PasswordType     *string   `json:"passwordType,omitempty"`
	Quantity         *int64    `json:"quantity,omitempty"`
	TypeID           *int64    `json:"typeId,omitempty"`
}

var listCorporationContainerLogs = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]ContainerLog, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	logs, totalPages, err := client.CorporationContainerLogs(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContainerLog, len(logs))
	for i, l := range logs {
		out[i] = ContainerLog{
			Action: l.Action, CharacterID: l.CharacterId, ContainerID: l.ContainerId, ContainerTypeID: l.ContainerTypeId,
			LocationFlag: l.LocationFlag, LocationID: l.LocationId, LoggedAt: l.LoggedAt,
			NewConfigBitmask: l.NewConfigBitmask, OldConfigBitmask: l.OldConfigBitmask,
			PasswordType: l.PasswordType, Quantity: l.Quantity, TypeID: l.TypeId,
		}
	}
	return out, totalPages, nil
}

type ListCorporationContainerLogsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationContainerLogsOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	Page          int32          `json:"page"`
	TotalPages    int32          `json:"totalPages"`
	Logs          []ContainerLog `json:"logs"`
}

func listCorporationContainerLogsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationContainerLogsInput, ListCorporationContainerLogsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContainerLogsInput) (*mcpsdk.CallToolResult, ListCorporationContainerLogsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContainerLogsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContainerLogsOutput{}, fmt.Errorf("list corporation container logs for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		logs, totalPages, err := listCorporationContainerLogs(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationContainerLogsOutput{}, fmt.Errorf("list corporation container logs: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationContainerLogsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Logs: logs}, nil
	}
}

// --- list_corporation_facilities (private, not paginated) ---

type Facility struct {
	FacilityID int64 `json:"facilityId"`
	SystemID   int64 `json:"systemId"`
	TypeID     int64 `json:"typeId"`
}

var listCorporationFacilities = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Facility, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	facilities, err := client.CorporationFacilities(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]Facility, len(facilities))
	for i, f := range facilities {
		out[i] = Facility{FacilityID: f.FacilityId, SystemID: f.SystemId, TypeID: f.TypeId}
	}
	return out, nil
}

type ListCorporationFacilitiesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationFacilitiesOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Facilities    []Facility `json:"facilities"`
}

func listCorporationFacilitiesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationFacilitiesInput, ListCorporationFacilitiesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationFacilitiesInput) (*mcpsdk.CallToolResult, ListCorporationFacilitiesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationFacilitiesOutput{}, err
		}
		if !found {
			return nil, ListCorporationFacilitiesOutput{}, fmt.Errorf("list corporation facilities for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		facilities, err := listCorporationFacilities(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationFacilitiesOutput{}, fmt.Errorf("list corporation facilities: %w", err)
		}
		return nil, ListCorporationFacilitiesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Facilities: facilities}, nil
	}
}

// --- list_corporation_medals (private, paginated) ---

type CorporationMedal struct {
	CreatedAt   time.Time `json:"createdAt"`
	CreatorID   int64     `json:"creatorId"`
	Description string    `json:"description"`
	MedalID     int64     `json:"medalId"`
	Title       string    `json:"title"`
}

var listCorporationMedals = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]CorporationMedal, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	medals, totalPages, err := client.CorporationMedals(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CorporationMedal, len(medals))
	for i, m := range medals {
		out[i] = CorporationMedal{CreatedAt: m.CreatedAt, CreatorID: m.CreatorId, Description: m.Description, MedalID: m.MedalId, Title: m.Title}
	}
	return out, totalPages, nil
}

type ListCorporationMedalsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMedalsOutput struct {
	CharacterID   int32              `json:"characterId"`
	CorporationID int64              `json:"corporationId"`
	Page          int32              `json:"page"`
	TotalPages    int32              `json:"totalPages"`
	Medals        []CorporationMedal `json:"medals"`
}

func listCorporationMedalsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMedalsInput, ListCorporationMedalsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMedalsInput) (*mcpsdk.CallToolResult, ListCorporationMedalsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMedalsOutput{}, err
		}
		if !found {
			return nil, ListCorporationMedalsOutput{}, fmt.Errorf("list corporation medals for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		medals, totalPages, err := listCorporationMedals(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMedalsOutput{}, fmt.Errorf("list corporation medals: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationMedalsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Medals: medals}, nil
	}
}

// --- list_corporation_medals_issued (private, paginated) ---

type MedalIssued struct {
	CharacterID int64     `json:"characterId"`
	IssuedAt    time.Time `json:"issuedAt"`
	IssuerID    int64     `json:"issuerId"`
	MedalID     int64     `json:"medalId"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
}

var listCorporationMedalsIssued = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]MedalIssued, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	issued, totalPages, err := client.CorporationMedalsIssued(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]MedalIssued, len(issued))
	for i, m := range issued {
		out[i] = MedalIssued{CharacterID: m.CharacterId, IssuedAt: m.IssuedAt, IssuerID: m.IssuerId, MedalID: m.MedalId, Reason: m.Reason, Status: m.Status}
	}
	return out, totalPages, nil
}

type ListCorporationMedalsIssuedInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMedalsIssuedOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Issued        []MedalIssued `json:"issued"`
}

func listCorporationMedalsIssuedHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationMedalsIssuedInput, ListCorporationMedalsIssuedOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMedalsIssuedInput) (*mcpsdk.CallToolResult, ListCorporationMedalsIssuedOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMedalsIssuedOutput{}, err
		}
		if !found {
			return nil, ListCorporationMedalsIssuedOutput{}, fmt.Errorf("list corporation medals issued for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		issued, totalPages, err := listCorporationMedalsIssued(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMedalsIssuedOutput{}, fmt.Errorf("list corporation medals issued: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationMedalsIssuedOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Issued: issued}, nil
	}
}

// --- list_corporation_starbases (private, paginated) ---

type Starbase struct {
	MoonID          *int64     `json:"moonId,omitempty"`
	OnlinedSince    *time.Time `json:"onlinedSince,omitempty"`
	ReinforcedUntil *time.Time `json:"reinforcedUntil,omitempty"`
	StarbaseID      int64      `json:"starbaseId"`
	State           *string    `json:"state,omitempty"`
	SystemID        int64      `json:"systemId"`
	TypeID          int64      `json:"typeId"`
	UnanchorAt      *time.Time `json:"unanchorAt,omitempty"`
}

var listCorporationStarbases = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Starbase, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	starbases, totalPages, err := client.CorporationStarbases(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Starbase, len(starbases))
	for i, s := range starbases {
		out[i] = Starbase{
			MoonID: s.MoonId, OnlinedSince: s.OnlinedSince, ReinforcedUntil: s.ReinforcedUntil,
			StarbaseID: s.StarbaseId, State: s.State, SystemID: s.SystemId, TypeID: s.TypeId, UnanchorAt: s.UnanchorAt,
		}
	}
	return out, totalPages, nil
}

type ListCorporationStarbasesInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationStarbasesOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Page          int32      `json:"page"`
	TotalPages    int32      `json:"totalPages"`
	Starbases     []Starbase `json:"starbases"`
}

func listCorporationStarbasesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationStarbasesInput, ListCorporationStarbasesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationStarbasesInput) (*mcpsdk.CallToolResult, ListCorporationStarbasesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationStarbasesOutput{}, err
		}
		if !found {
			return nil, ListCorporationStarbasesOutput{}, fmt.Errorf("list corporation starbases for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		starbases, totalPages, err := listCorporationStarbases(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationStarbasesOutput{}, fmt.Errorf("list corporation starbases: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationStarbasesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Starbases: starbases}, nil
	}
}

// --- get_corporation_starbase (private, not paginated; requires systemId) ---

type StarbaseFuel struct {
	Quantity int64 `json:"quantity"`
	TypeID   int64 `json:"typeId"`
}

type StarbaseDetail struct {
	AllowAllianceMembers                bool           `json:"allowAllianceMembers"`
	AllowCorporationMembers             bool           `json:"allowCorporationMembers"`
	Anchor                              string         `json:"anchor"`
	AttackIfAtWar                       bool           `json:"attackIfAtWar"`
	AttackIfOtherSecurityStatusDropping bool           `json:"attackIfOtherSecurityStatusDropping"`
	AttackSecurityStatusThreshold       *float64       `json:"attackSecurityStatusThreshold,omitempty"`
	AttackStandingThreshold             *float64       `json:"attackStandingThreshold,omitempty"`
	FuelBayTake                         string         `json:"fuelBayTake"`
	FuelBayView                         string         `json:"fuelBayView"`
	Fuels                               []StarbaseFuel `json:"fuels,omitempty"`
	Offline                             string         `json:"offline"`
	Online                              string         `json:"online"`
	Unanchor                            string         `json:"unanchor"`
	UseAllianceStandings                bool           `json:"useAllianceStandings"`
}

var getCorporationStarbase = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, starbaseID int64, systemID int64) (StarbaseDetail, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return StarbaseDetail{}, fmt.Errorf("build ESI client: %w", err)
	}
	detail, err := client.CorporationStarbaseDetail(ctx, corporationID, starbaseID, systemID)
	if err != nil {
		return StarbaseDetail{}, err
	}
	fuels := make([]StarbaseFuel, len(detail.Fuels))
	for i, f := range detail.Fuels {
		fuels[i] = StarbaseFuel{Quantity: f.Quantity, TypeID: f.TypeId}
	}
	return StarbaseDetail{
		AllowAllianceMembers: detail.AllowAllianceMembers, AllowCorporationMembers: detail.AllowCorporationMembers,
		Anchor: detail.Anchor, AttackIfAtWar: detail.AttackIfAtWar, AttackIfOtherSecurityStatusDropping: detail.AttackIfOtherSecurityStatusDropping,
		AttackSecurityStatusThreshold: detail.AttackSecurityStatusThreshold, AttackStandingThreshold: detail.AttackStandingThreshold,
		FuelBayTake: detail.FuelBayTake, FuelBayView: detail.FuelBayView, Fuels: fuels,
		Offline: detail.Offline, Online: detail.Online, Unanchor: detail.Unanchor, UseAllianceStandings: detail.UseAllianceStandings,
	}, nil
}

type GetCorporationStarbaseInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	StarbaseID    int64 `json:"starbaseId" jsonschema:"the starbase (POS) ID to look up"`
	SystemID      int64 `json:"systemId" jsonschema:"the solar system ID the starbase is in"`
}
type GetCorporationStarbaseOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	StarbaseID    int64          `json:"starbaseId"`
	Detail        StarbaseDetail `json:"detail"`
}

func getCorporationStarbaseHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationStarbaseInput, GetCorporationStarbaseOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationStarbaseInput) (*mcpsdk.CallToolResult, GetCorporationStarbaseOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCorporationStarbaseOutput{}, err
		}
		if !found {
			return nil, GetCorporationStarbaseOutput{}, fmt.Errorf("get corporation starbase for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		detail, err := getCorporationStarbase(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.StarbaseID, in.SystemID)
		if err != nil {
			return nil, GetCorporationStarbaseOutput{}, fmt.Errorf("get corporation starbase: %w", err)
		}
		return nil, GetCorporationStarbaseOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, StarbaseID: in.StarbaseID, Detail: detail}, nil
	}
}

// --- list_corporation_structures (private, paginated) ---

type StructureService struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type Structure struct {
	CorporationID      int64              `json:"corporationId"`
	FuelExpires        *time.Time         `json:"fuelExpires,omitempty"`
	Name               *string            `json:"name,omitempty"`
	NextReinforceApply *time.Time         `json:"nextReinforceApply,omitempty"`
	NextReinforceHour  *int64             `json:"nextReinforceHour,omitempty"`
	ProfileID          int64              `json:"profileId"`
	ReinforceHour      *int64             `json:"reinforceHour,omitempty"`
	Services           []StructureService `json:"services,omitempty"`
	State              string             `json:"state"`
	StateTimerEnd      *time.Time         `json:"stateTimerEnd,omitempty"`
	StateTimerStart    *time.Time         `json:"stateTimerStart,omitempty"`
	StructureID        int64              `json:"structureId"`
	SystemID           int64              `json:"systemId"`
	TypeID             int64              `json:"typeId"`
	UnanchorsAt        *time.Time         `json:"unanchorsAt,omitempty"`
}

var listCorporationStructures = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Structure, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	structures, totalPages, err := client.CorporationStructures(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Structure, len(structures))
	for i, s := range structures {
		services := make([]StructureService, len(s.Services))
		for j, svc := range s.Services {
			services[j] = StructureService{Name: svc.Name, State: svc.State}
		}
		out[i] = Structure{
			CorporationID: s.CorporationId, FuelExpires: s.FuelExpires, Name: s.Name,
			NextReinforceApply: s.NextReinforceApply, NextReinforceHour: s.NextReinforceHour, ProfileID: s.ProfileId,
			ReinforceHour: s.ReinforceHour, Services: services, State: s.State,
			StateTimerEnd: s.StateTimerEnd, StateTimerStart: s.StateTimerStart, StructureID: s.StructureId,
			SystemID: s.SystemId, TypeID: s.TypeId, UnanchorsAt: s.UnanchorsAt,
		}
	}
	return out, totalPages, nil
}

type ListCorporationStructuresInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationStructuresOutput struct {
	CharacterID   int32       `json:"characterId"`
	CorporationID int64       `json:"corporationId"`
	Page          int32       `json:"page"`
	TotalPages    int32       `json:"totalPages"`
	Structures    []Structure `json:"structures"`
}

func listCorporationStructuresHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationStructuresInput, ListCorporationStructuresOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationStructuresInput) (*mcpsdk.CallToolResult, ListCorporationStructuresOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationStructuresOutput{}, err
		}
		if !found {
			return nil, ListCorporationStructuresOutput{}, fmt.Errorf("list corporation structures for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		structures, totalPages, err := listCorporationStructures(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationStructuresOutput{}, fmt.Errorf("list corporation structures: %w", err)
		}
		page := in.Page
		if page == 0 {
			page = 1
		}
		return nil, ListCorporationStructuresOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Structures: structures}, nil
	}
}
```

This exact code has already been compiled clean (`go build`/`go vet`/`gofmt -l`) in an isolated sandbox against a stub package providing `characterExists`/`ErrCharacterNotFound` with the identical shape they have in the real `mcp/character.go`.

- [ ] **Step 2: Append tests to `mcp/corporation_internal_test.go`**

Follow the file's existing pattern exactly (package-level `var` override + `t.Cleanup` restore; `keyring.MockInit()` + temp-file `config.Store` for unknown-character tests). For each of the 8 tools, write one happy-path test (`TestX_MapsFields`) asserting the mapped output, plus one `TestX_UnknownCharacter` test asserting `errors.Is(err, ErrCharacterNotFound)` (all 8 are private). That's 16 new test functions:
`TestListCorporationBlueprintsHandler_MapsFields` + `..._UnknownCharacter`,
`TestListCorporationContainerLogsHandler_MapsFields` + `..._UnknownCharacter`,
`TestListCorporationFacilitiesHandler_MapsFields` + `..._UnknownCharacter`,
`TestListCorporationMedalsHandler_MapsFields` + `..._UnknownCharacter`,
`TestListCorporationMedalsIssuedHandler_MapsFields` + `..._UnknownCharacter`,
`TestListCorporationStarbasesHandler_MapsFields` + `..._UnknownCharacter`,
`TestGetCorporationStarbaseHandler_MapsFields` + `..._UnknownCharacter`,
`TestListCorporationStructuresHandler_MapsFields` + `..._UnknownCharacter`.

- [ ] **Step 3: Run the new tests**

Run: `go test ./mcp/... -run Corporation -v`
Expected: all tests PASS.

- [ ] **Step 4: Register the 8 tools in `mcp/server.go`**

Add these 8 `mcpsdk.AddTool` blocks immediately after the existing `list_corporation_standings` block (before `return server`):

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_blueprints",
		Description: "List a corporation's blueprints, using a logged-in character's roles to authorize.",
	}, listCorporationBlueprintsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_container_logs",
		Description: "List a corporation's container access logs, using a logged-in character's roles to authorize.",
	}, listCorporationContainerLogsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_facilities",
		Description: "List a corporation's facilities, using a logged-in character's roles to authorize.",
	}, listCorporationFacilitiesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_medals",
		Description: "List a corporation's defined medals, using a logged-in character's roles to authorize.",
	}, listCorporationMedalsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_medals_issued",
		Description: "List medals a corporation has issued to its members, using a logged-in character's roles to authorize.",
	}, listCorporationMedalsIssuedHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_starbases",
		Description: "List a corporation's starbases (POSes), using a logged-in character's roles to authorize.",
	}, listCorporationStarbasesHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_corporation_starbase",
		Description: "Get detail for one of a corporation's starbases (POSes), using a logged-in character's roles to authorize.",
	}, getCorporationStarbaseHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_structures",
		Description: "List a corporation's structures, using a logged-in character's roles to authorize.",
	}, listCorporationStructuresHandler(version, clientID, store))

	return server
}
```

Update the package doc comment above `NewServer` to append the 8 new tool names to its running list (ending in `..., list_corporation_standings, list_corporation_blueprints, list_corporation_container_logs, list_corporation_facilities, list_corporation_medals, list_corporation_medals_issued, list_corporation_starbases, get_corporation_starbase, and list_corporation_structures registered.`).

- [ ] **Step 5: Update `mcp/server_test.go`**

Append these 8 names to the `[]string{...}` literal, right after `"list_corporation_standings",`:

```go
			"list_corporation_blueprints", "list_corporation_container_logs", "list_corporation_facilities",
			"list_corporation_medals", "list_corporation_medals_issued", "list_corporation_starbases",
			"get_corporation_starbase", "list_corporation_structures",
```

- [ ] **Step 6: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: everything PASSes, including `TestServer_AllToolsRegistered` (now asserting 79 tool names present).

- [ ] **Step 7: Docs sync**

In `CLAUDE.md` and `README.md`:
- Change `mcp/` row's tool-count word from "seventy-one" to "seventy-nine".
- Append the 8 new tool names (in the exact order registered above) to the end of the `mcp/` row's tool list.

In `docs/reference/esi-coverage-roadmap.md`:
- Flip the Corporation row from `| 10 | Corporation | 22 | Planned |` to `| 10 | Corporation | 22 | This plan |` — both parts of Corporation now land in this PR.

- [ ] **Step 8: Commit**

```bash
git add mcp/corporation.go mcp/corporation_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add corporation MCP tools, Part 2 (8 tools: assets/infrastructure) -- Corporation tag complete"
```

---

## Final Steps (after both tasks)

1. Dispatch a final whole-batch reviewer (most capable available model) covering the full diff for this batch (`git diff 28931e6..HEAD` — the design spec commit forward, matching Part 1's review scope convention).
2. If it finds real issues, fix them (one fix subagent for the whole findings list, not one per finding), re-verify, and commit the fix.
3. Push directly to the shared branch: `git push origin worktree-esi-access-list-activities`.
4. Update the tracker item (#162) with this batch's completion and the running remaining-tag list.
5. Continue the sweep with the next batch per the standing autonomous-execution instruction — no pause for approval.
