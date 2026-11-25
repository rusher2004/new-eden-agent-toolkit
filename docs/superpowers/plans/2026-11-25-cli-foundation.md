# CLI foundation and loyalty pilot — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the shared foundation for a neat CLI (operation contract, MCP and CLI adapters, import-boundary check, golden tool snapshot) and prove it end to end by moving the two loyalty tools out of `mcp/` so they run from both interfaces.

**Architecture:** Tool logic moves from `mcp/` into domain packages named for what they do. A small `operation` package defines what an operation is. `mcp/` and a new `cli/` are thin adapters over the same `[]operation.Operation`, built by the composition root in `commands/`. Unmigrated tools keep their old registration and keep working.

**Tech Stack:** Go 1.26, `urfave/cli/v3`, `modelcontextprotocol/go-sdk` v1.8.0, `google/jsonschema-go` (already an indirect dependency, promote it to direct).

**Spec:** `docs/superpowers/specs/2026-11-25-cli-interface-design.md`

## This plan's place in the whole

The spec is a migration of all 277 tools, too large for one plan. This is plan 1; each later plan is written when the previous one lands, because the first area teaches what the rest need.

1. **This plan:** foundation + loyalty pilot + the tool-to-package assignment table for review.
2. Owner-flag command groups, `{{op name}}` description placeholders, then the first large areas (character, corporation, contract).
3. Remaining areas, a few per plan, following the assignment table.
4. Cross-process safety: config file lock (works on Windows), token-refresh safety, SDE "last checked" window.
5. Switch: docs, `Instructions`, the four plugin skills and the plugin manifest go CLI-first and the plugin drops its MCP registration; generated `docs/reference/cli.md`; remove `mcp/` leftovers; enforce the adapter-imports rule for `mcp/`. Not before the CLI covers every tool and plan 4 has landed.

## Global Constraints

- Flat package layout, no `internal/`, one package per responsibility. Producers (`sde`, `esi`, `sso`, `creds`, `config`, `analytics`) export concrete types and no interfaces for callers; consumers declare the small interfaces they need; fakes live in the consumer's `_test.go`; `ifaceaudit` gates this in CI. Factories return an untyped `nil, err`, never a typed nil in an interface.
- Domain packages import no `mcp/`, `cli/`, `commands/`, MCP SDK, urfave/cli or `net/http`.
- No operation input or output type contains a type from the generated ESI module (`github.com/fnt-eve/goesi-openapi`).
- MCP tool names, descriptions and schemas must not change: the golden snapshot stays green through every task.
- External test packages (`package foo_test`); standard library only for assertions; no real keychain, `~/.neat/` or network in tests (`t.Setenv("HOME", t.TempDir())`, `keyring.MockInit()`).
- Errors wrapped with context (`fmt.Errorf("<what>: %w", err)`); no `_ = err`; no fallbacks in new code.
- CLI output is compact JSON on stdout only; notices and progress on stderr; errors as `{"error": "..."}` on stderr, exit 1; usage errors exit 2.
- Every PR that touches Go source bumps `VERSION`, runs `go generate ./...` and adds a `claude_resources/neat/CHANGELOG.md` entry.
- Commits end with the attribution lines given in the session's system reminder. Work on a branch, never on `main`.

## Review Focus

Failure modes the spec implies but no task's own happy path covers; each has a test in the named task.

- `--json -` with empty or malformed stdin: usage error, exit 2, JSON error on stderr (Task 8).
- A non-numeric or out-of-range ID flag (`--character abc`, `--character 99999999999` for an int32): usage error, never a panic (Task 8).
- `--json` combined with other input flags: usage error, not a silent merge (Task 8).
- `neat --help`, `neat loyalty --help` and `neat version` never open the config store or the SDE, so they work with an unwritable HOME (Task 8).
- An unknown character through the CLI returns the same `character not found` error the MCP does, with no ESI client opened (Task 8).
- The CLI prints exactly the JSON the MCP returns as structured content for the same operation and input (Task 8).

## File Structure

- `commands/testdata/tools.golden.json`, `commands/tools_golden_test.go`: snapshot of `tools/list`.
- `operation/operation.go`: the contract, schema derivation, input decoding, progress sink.
- `character/character.go`: the logged-in-character check and its sentinel error.
- `universe/names.go`: SDE-backed name lookup for IDs.
- `loyalty/loyalty.go`: the two loyalty operations (moved from `mcp/loyalty.go`).
- `mcp/register.go`: `Register(server, ops)`.
- `cli/build.go`, `cli/flags.go`, `cli/paths.go`: command tree, flag derivation and binding, the command-path table.
- `commands/operations.go`: builds the operation list from the sources; `commands/cli.go` mounts the CLI groups; `commands/mcp_tools*.go` loses the loyalty lines.
- `tools/archaudit/`: import-boundary check.
- `docs/superpowers/plans/2026-11-25-tool-package-assignment.md`: the assignment table (Task 9).

---

### Task 1: Golden snapshot of the MCP tool list

**Files:**
- Create: `commands/tools_golden_test.go`, `commands/testdata/tools.golden.json`

**Interfaces:**
- Produces: `TestMCPToolListMatchesGolden`, a regression gate every later task runs; `go test ./commands -run TestMCPToolListMatchesGolden -update` regenerates the file.

- [ ] **Step 1: Create the branch.** `git switch -c cli-foundation`.
- [ ] **Step 2: Write `TestMCPToolListMatchesGolden` in `package commands_test`.** It builds `commands.NewMCPServer("test", "client-id", config.NewStoreAt(<tempdir>/config.json), nil)`, lists tools over an in-memory client session (same pattern as `analytics.RegisteredTools`), and marshals them sorted by name as a JSON array of `{name, description, annotations, inputSchema, outputSchema}` with indentation. It compares to `testdata/tools.golden.json`; an `-update` flag rewrites the file.
- [ ] **Step 3: Run it, expect FAIL** (no golden file): `go test ./commands -run TestMCPToolListMatchesGolden`.
- [ ] **Step 4: Generate the file** with `-update`, then check it holds 277 entries (`jq length commands/testdata/tools.golden.json`).
- [ ] **Step 5: Run it, expect PASS.**
- [ ] **Step 6: Commit** (`test(commands): snapshot the MCP tool list`).

### Task 2: The `operation` package

**Files:**
- Create: `operation/operation.go`, `operation/operation_test.go` (`package operation_test`)

**Interfaces:**
- Produces:
  ```go
  type Kind int
  const ( Read Kind = iota; Write )
  var ErrInvalidInput error // wrapped by Decode for any bad input
  type Operation struct {
      Name, Description string
      Kind              Kind
      InputType, OutputType reflect.Type
      Run    func(ctx context.Context, in any) (any, error) // in is what Decode returned
  }
  func New[In, Out any](name, description string, kind Kind, run func(context.Context, In) (Out, error)) Operation
  func (Operation) Decode(raw []byte) (any, error)          // validates against InputSchema, returns *In
  func (Operation) InputSchema() *jsonschema.Schema
  func (Operation) OutputSchema() *jsonschema.Schema
  func WithProgress(ctx context.Context, sink func(msg string)) context.Context
  func Progress(ctx context.Context, msg string)             // no-op without a sink
  ```

- [ ] **Step 1: Write failing tests**, each in its own `Test` function:
  - `TestNew_RunsTypedFunction`: an op over `struct{A int \`json:"a"\`}` → `struct{B int \`json:"b"\`}`; `Decode([]byte('{"a":2}'))` then `Run` returns an `Out` with `B == 2` (the function doubles `A`).
  - `TestDecode_RejectsInvalidInput`: unknown field, wrong type and a missing non-`omitempty` field each return an error satisfying `errors.Is(err, operation.ErrInvalidInput)`.
  - `TestSchemas_UseJSONAndJSONSchemaTags`: for `characterId int32` tagged `jsonschema:"the character"`, `InputSchema().Properties["characterId"].Description == "the character"`, the property is required, and `omitempty` fields are not.
  - `TestRun_WrongInputTypeIsError`: `Run` with a value that is not the `Decode` result returns an error, no panic.
  - `TestProgress_ReachesSink`: `Progress` calls the sink installed by `WithProgress`; without one it does nothing.
- [ ] **Step 2: Run, expect FAIL** (`go test ./operation`).
- [ ] **Step 3: Implement** in `operation/operation.go`. Derive schemas with `jsonschema.For[In]` exactly as go-sdk v1.8.0's typed `AddTool` does (read `toolForErr` in the module cache's `mcp/server.go` and mirror its options so Task 6's golden matches); validate in `Decode` with the resolved schema; wrap failures with `ErrInvalidInput`. `go get github.com/google/jsonschema-go` to make it a direct dependency.
- [ ] **Step 4: Run, expect PASS**; `go vet ./operation`.
- [ ] **Step 5: Commit** (`feat(operation): the contract adapters share`).

### Task 3: The `character` package

**Files:**
- Create: `character/character.go`, `character/character_test.go`
- Modify: `mcp/remove_character.go` (the `ErrCharacterNotFound` line), `mcp/character.go:264` (`characterExists`)

**Interfaces:**
- Produces:
  ```go
  var ErrNotFound = errors.New("character not found")
  type loader interface{ Load() (config.Config, error) } // unexported; *config.Store satisfies it
  func Known[L loader](store L, characterID int32) (bool, error)
  ```
  (`Known` is generic so callers outside the package can pass a value without naming the unexported interface.)
- Consumes: `config.Config`, `config.Character`.

- [ ] **Step 1: Write failing tests:** `TestKnown_ReportsLoggedInCharacters` (store with character 7: 7 true, 8 false), `TestKnown_WrapsLoadError` (`errors.Is` of the fake's error, message names "load config"), `TestErrNotFound_Message` (`ErrNotFound.Error() == "character not found"`).
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement.** Then in `mcp/`: `var ErrCharacterNotFound = character.ErrNotFound` and `characterExists` delegates to `character.Known`, so every unmigrated tool and its tests keep working and `errors.Is` still matches.
- [ ] **Step 4: Run `go test ./character ./mcp/...`, expect PASS.**
- [ ] **Step 5: Commit** (`refactor: character package owns the logged-in check`).

### Task 4: `universe.TypeNamer`

**Files:**
- Create: `universe/names.go`, `universe/names_test.go`
- Modify: `mcp/enrich.go` (`resolveTypeNames` body only; keep the variable, tests stub it)

**Interfaces:**
- Produces:
  ```go
  func NewTypeNamer[C typeCache](cache C, userAgent string) *TypeNamer
  func (*TypeNamer) TypeNames(ctx context.Context, ids []int64) map[int64]string
  // typeCache: Type(ctx, userAgent string, id int64) (sde.Type, bool, error)
  ```
  Contract (unchanged from `resolveTypeNames`): dedupe IDs, drop misses and errors, return nil for no IDs or a nil cache, never call ESI.

- [ ] **Step 1: Write failing tests:** `TestTypeNames_DedupesLookups` (a fake counting calls: IDs `[34,34,35]` → 2 lookups), `TestTypeNames_DropsMissesAndErrors`, `TestTypeNames_NoIDsReturnsNil`, `TestTypeNames_NilCacheReturnsNil`.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement**, moving the body of `resolveTypeNames`; `mcp/enrich.go`'s variable becomes `universe.NewTypeNamer(sdeStore, userAgent).TypeNames(ctx, ids)`.
- [ ] **Step 4: Run `go test ./universe ./mcp/...`, expect PASS** (mcp tests that stub `resolveTypeNames` prove the seam survives).
- [ ] **Step 5: Commit** (`refactor: universe owns SDE type-name lookup`).

### Task 5: The `loyalty` package

**Files:**
- Create: `loyalty/loyalty.go`, `loyalty/loyalty_test.go` (`package loyalty_test`)
- Delete: `mcp/loyalty.go`, `mcp/loyalty_internal_test.go`

**Interfaces:**
- Consumes: `character.Known`, `character.ErrNotFound` (Task 3); `operation.New`, `operation.Read` (Task 2); a name lookup shaped like `universe.TypeNamer` (Task 4).
- Produces:
  ```go
  func New[C characterESI, P storeESI, L loader, N typeNames](
      openCharacter func(ctx context.Context, characterID int32) (C, error),
      openPublic func() P, characters L, names N) *Service
  func (*Service) Points(ctx, PointsInput) (PointsOutput, error)
  func (*Service) StoreOffers(ctx, StoreOffersInput) (StoreOffersOutput, error)
  func (*Service) Operations() []operation.Operation // list_character_loyalty_points, list_loyalty_store_offers; both Read
  // characterESI: CharacterLoyaltyPoints(ctx) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error)
  // storeESI:     LoyaltyStoreOffers(ctx, corporationID int64) ([]esiapi.LoyaltyStoresCorporationIdOffersGetInner, error)
  // typeNames:    TypeNames(ctx, ids []int64) map[int64]string
  ```
  Input and output JSON shapes, field tags and descriptions are copied unchanged from `mcp/loyalty.go`, so the golden stays green. `New` is generic so `commands/` can pass concrete clients; it stores closures that convert the result to the interface and return an untyped `nil, err` on failure.

- [ ] **Step 1: Port the tests** from `mcp/loyalty_internal_test.go` to the new API as external tests with fakes in the file: `TestPoints_MapsFields`, `TestPoints_UnknownCharacter` (no client opened; `errors.Is(err, character.ErrNotFound)`), `TestPoints_PropagatesESIError`, `TestPoints_PropagatesOpenError`, `TestStoreOffers_MapsFields` (including `AkCost` pointer and required items), `TestStoreOffers_PropagatesError`, `TestStoreOffers_ResolvesTypeNames` (fake names fill `TypeName` on offers and required items), plus `TestOperations_NamesAndKinds` (the two names, both `operation.Read`).
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** the package by moving the logic from `mcp/loyalty.go`, replacing handler glue with the methods above.
- [ ] **Step 4: Run `go test ./loyalty`, expect PASS.** Do not delete the `mcp` files yet.
- [ ] **Step 5: Commit** (`feat(loyalty): loyalty operations as a domain package`).

### Task 6: `mcp.Register` and the operation list

**Files:**
- Create: `mcp/register.go`, `mcp/register_test.go` (`package mcp_test`), `commands/operations.go`, `commands/operations_test.go`
- Modify: `commands/mcp_tools.go` (drop the two `mcp.AddListCharacterLoyaltyPoints` / `AddListLoyaltyStoreOffers` lines, call `mcp.Register`)
- Delete: `mcp/loyalty.go`, `mcp/loyalty_internal_test.go`

**Interfaces:**
- Consumes: `operation.Operation` (Task 2), `loyalty.New` (Task 5).
- Produces:
  ```go
  func Register(s *mcpsdk.Server, ops []operation.Operation)        // mcp/register.go
  func Operations(d toolDeps) []operation.Operation                  // commands/operations.go; safe with a zero toolDeps
  ```

- [ ] **Step 1: Write failing tests** for `Register` over an in-memory session with a demo operation: `TestRegister_ListsToolWithSchemas` (name, description, input and output schema present), `TestRegister_CallReturnsStructuredAndTextContent` (structured content equals the output; text content is its JSON, as the SDK's typed handler produces), `TestRegister_RunErrorIsToolError` (`IsError` true, text holds the message), `TestRegister_InvalidArgumentsFail` (the call returns an error, as the typed handler does). In `commands/operations_test.go`: `TestOperationTypesHaveNoESIModels` (walk each operation's `InputType`/`OutputType` recursively and fail on any type whose `PkgPath` contains `goesi-openapi`).
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement `Register`** with the SDK's low-level `AddTool`: schemas from the operation, `Decode` for arguments, `StructuredContent` plus one `TextContent` holding the JSON, a failed `Run` as an `IsError` result. No annotations (adding them would change `tools/list` for every client; do it in its own change). Implement `Operations` with the loyalty service built from `d.openESI`, `d.openPublicESI`, `d.store` and `universe.NewTypeNamer(d.sde, d.userAgent)`; call `mcp.Register(server, Operations(d))` from `addConvertedTools` and delete the two old calls and files.
- [ ] **Step 4: Run `go test ./...`, expect PASS**, in particular `TestMCPToolListMatchesGolden` unchanged (this proves schemas and descriptions survived the move). If the golden differs, fix the derivation in Task 2's code, not the golden.
- [ ] **Step 5: Commit** (`refactor(mcp): register operations; loyalty moves out of mcp`).

### Task 7: Import-boundary check

**Files:**
- Create: `tools/archaudit/main.go`, `tools/archaudit/main_test.go`, `tools/archaudit/testdata/` (small violating trees)

**Interfaces:**
- Produces: `func Audit(root string) ([]Finding, error)`; `Finding{Kind, File, Import string}`; kinds `DOMAIN_IMPORTS_ADAPTER` (a domain, pure or source package imports `mcp`, `cli`, `commands`, the MCP SDK or urfave/cli; a domain or pure package imports `net/http`), `CLI_IMPORTS_DOMAIN` (`cli/` imports any package other than `operation` and the standard library and third-party libraries). Package roles: adapters `mcp`, `cli`, `rest`; composition `cmd`, `commands`; sources `esi`, `sde`, `config`, `creds`, `sso`, `analytics`; pure `nav`, `cargo`, `market`, `dogma`, `industry`; contract `operation`; every other top-level directory with Go files and not `tools`, `docs`, `claude_resources`, `bin` is a domain package. `_test.go` files are skipped. Rule for `mcp/` importing domain packages is deferred to plan 5 (it legitimately does while areas migrate).

- [ ] **Step 1: Write failing tests**, using fixture trees under `testdata`: `TestAuditFlagsDomainImportingMCP`, `TestAuditFlagsDomainImportingSDK`, `TestAuditFlagsDomainImportingHTTP`, `TestAuditFlagsCLIImportingDomain`, `TestAuditAllowsSourceUsingHTTP`, and `TestRepoRespectsBoundaries` (runs `Audit("../..")`, fails on any finding, like `TestRepoConformsToConsumerInterfaceRules`).
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** with `go/parser` import scanning, deterministic sorted output.
- [ ] **Step 4: Run `go test ./tools/archaudit`, expect PASS** (repo currently has `character`, `universe`, `loyalty` as domain packages; none may violate).
- [ ] **Step 5: Commit** (`test(archaudit): enforce package boundaries`).

### Task 8: The `cli` package and its wiring

**Files:**
- Create: `cli/build.go`, `cli/flags.go`, `cli/paths.go`, `cli/build_test.go` (`package cli_test`), `commands/cli.go`, `commands/cli_test.go`
- Modify: `commands/commands.go` (`Root` appends the groups from `cliCommands`)

**Interfaces:**
- Consumes: `operation.Operation`, `operation.ErrInvalidInput` (Task 2); `Operations` (Task 6).
- Produces:
  ```go
  // describe supplies names, help and flags; source builds working operations at run time.
  func Build(describe []operation.Operation, source func(ctx context.Context) (ops []operation.Operation, closeFn func() error, err error)) ([]*urfave.Command, error)
  var Paths = map[string][]string{ // operation name -> command path
      "list_character_loyalty_points": {"loyalty", "points"},
      "list_loyalty_store_offers":     {"loyalty", "store-offers"},
  }
  var Groups = map[string]string{"loyalty": "loyalty point balances and loyalty stores"}
  ```
  Flag rules: each scalar field of the input struct (`int32`, `int64`, `string`, `bool`, `float64`) becomes a flag named from its JSON name with a trailing `Id`/`ID` removed, kebab-cased (`characterId` → `--character`, `corporationId` → `--corporation`, `accessListId` → `--access-list`); a slice of scalars is a repeatable flag; anything else is reachable only through `--json '<object>'` or `--json -` (stdin). Combining `--json` with any input flag is a usage error. Flags build a JSON object that goes through `Operation.Decode`, so there is one validation path. Every leaf has `--pretty`. Success: compact JSON plus newline on stdout. Failure: `{"error": "<message>"}` on stderr (newline-terminated), exit 1; `errors.Is(err, operation.ErrInvalidInput)` or bad flag values: exit 2. The leaf writes the error JSON itself and returns `cli.Exit("", code)` so nothing prints twice (verify urfave prints nothing for an empty message). Help for a leaf: first sentence of the description as usage, the full description as the body, and "changes state" noted for `Write` operations.

- [ ] **Step 1: Write failing `cli` tests** around a demo operation (input `characterId int32`, `corporationId int64 omitempty`, `types []int64 omitempty`, `nested struct{...}`), invoking the built command with captured writers: `TestBuild_RunsOperationWithFlags`, `TestBuild_FlagNames` (`--character`, `--corporation`, `--type`; nested has no flag), `TestBuild_RepeatableSliceFlag`, `TestBuild_JSONFromStdin`, `TestBuild_EmptyOrMalformedStdinIsUsageError` (exit 2, JSON error), `TestBuild_NonNumericAndOutOfRangeIDIsUsageError` (`abc` and `99999999999` for int32), `TestBuild_JSONWithFlagsIsUsageError`, `TestBuild_MissingRequiredIsUsageError`, `TestBuild_OperationErrorIsExit1`, `TestBuild_PrettyFlag`, `TestBuild_StdoutHoldsOnlyResult` (progress sent through `operation.Progress` appears on stderr), `TestBuild_HelpDoesNotCallSource` (source panics if called; `--help` still works), `TestBuild_OperationWithoutPathIsError` (Build returns an error naming it).
- [ ] **Step 2: Write failing `commands` tests:** `TestCLIPathsCoverOperations` (every operation in `Operations(toolDeps{})` has a `Paths` entry; no two share a path; each group in a path has a `Groups` entry; each operation has a non-empty description), `TestCLIAndMCPReturnTheSameJSON` (a demo operation run through `cli.Build` and through `mcp.Register` over an in-memory session yields byte-identical JSON), `TestCLI_UnknownCharacterNoESI` (HOME is a temp dir, `keyring.MockInit()`, `neat loyalty points --character 999` exits 1 with `character not found` on stderr and never builds an ESI client), `TestCLI_HelpAndVersionTouchNothing` (HOME points at a regular file so `MkdirAll` would fail; `neat --help`, `neat loyalty --help`, `neat version` still succeed).
- [ ] **Step 3: Run, expect FAIL.**
- [ ] **Step 4: Implement `cli/`.** Derive flags with `reflect` over `Operation.InputType`; set urfave flag types per scalar kind and let urfave's parse errors map to exit 2.
- [ ] **Step 5: Implement `commands/cli.go`:** `cliCommands(version, clientID string)` calls `cli.Build(Operations(toolDeps{}), source)` where `source` resolves `config.NewStore`, `sde.DefaultPath`/`sde.Open` (a failed open leaves a nil store with a warning on stderr, as `neat mcp` does) and returns `Operations(...)` over them plus a close function that closes the SDE store. Add the groups to `Root`. A zero `toolDeps` must build metadata without touching disk or network.
- [ ] **Step 6: Run `go test ./...`, expect PASS** (golden still green).
- [ ] **Step 7: Commit** (`feat(cli): command line interface over operations; loyalty pilot`).

### Task 9: Docs, version, and the assignment table

**Files:**
- Modify: `CLAUDE.md` (package table: new `operation`, `character`, `universe`, `loyalty`, `cli`, `tools/archaudit` rows; note the migration is in progress and that `mcp/loyalty.go` is no longer the worked example, `loyalty/loyalty.go` is), `docs/README.md` (package list), `VERSION`, `claude_resources/neat/CHANGELOG.md`, `claude_resources/neat/.claude-plugin/plugin.json` (via `go generate ./...`)
- Create: `docs/superpowers/plans/2026-11-25-tool-package-assignment.md`

- [ ] **Step 1:** Update `CLAUDE.md` and `docs/README.md` as above. Do not touch `README.md`, `Instructions` or the skills: the CLI is a preview until plan 5.
- [ ] **Step 2:** Bump `VERSION` (minor), run `go generate ./...`, add a CHANGELOG entry describing `neat loyalty points` and `neat loyalty store-offers` as a CLI preview.
- [ ] **Step 3:** Write the assignment table: for every one of the 277 tools in `docs/reference/tools.md`, its target domain package, its CLI command path (from the spec's tree) and its owner flag if any. Flag every collision with `market/` and `industry/` and every tool whose package is not obvious. Group by target package.
- [ ] **Step 4:** Run `go test ./...`, `go vet ./...`, `gofmt -l .` (expect no output), `go mod tidy` and `go generate ./...` (expect no diff).
- [ ] **Step 5: Commit** (`docs: CLI foundation, version bump, tool assignment table`).
- [ ] **Step 6: Stop and ask the owner to review the assignment table** before plan 2 is written.

---

## Self-review notes

- **Spec coverage:** operation contract (Task 2), adapters (Tasks 6, 8), import check and the ESI-types-in-schemas check (Tasks 6, 7), golden (Task 1), progress sink (Task 2), JSON-only output and exit codes (Task 8), assignment table (Task 9). Deferred to later plans, stated above: owner-flag groups, `{{op}}` placeholders, cross-process safety, the docs, skills, instructions and plugin switch, `cli.md`, and read-only annotations.
- **Two spec conflicts, resolved here:** (1) the spec's tree lists bare resource names (`loyalty points`) while its rules say verbs come from a fixed set; this plan follows the approved tree and does not test a verb set. (2) The spec says `Register` sets a read-only hint; this plan omits annotations so `tools/list` stays identical, and revisits them in their own change.
- **Consistency:** `character.Known`/`ErrNotFound`, `universe.NewTypeNamer`, `loyalty.New`, `operation.New`/`Decode`/`ErrInvalidInput`, `mcp.Register`, `Operations`, `cli.Build`/`Paths`/`Groups` are named identically wherever used.
