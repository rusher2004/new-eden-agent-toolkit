# Consumer-defined interfaces for the mcp package

Status: done. The foundation settled the pattern (explicit wiring from
`commands/`, the `loyalty.go` worked example, the audit and the standards), the
ported PRs moved every file in `mcp/` to it, and the cleanup deleted `Deps`,
`ConfigStore`, `SDEStore` and `NewServer`'s registrations. `ifaceaudit` now
fails CI on any `CONCRETE_*`, `PRODUCER_IFACE`, `IFACE_AWAY` or `TYPE_ONLY_FILE`
(`TestRepoConformsToConsumerInterfaceRules`); only `INTERFACE` stays informational.

`tools/ifaceaudit` narrowed in that first PR: a CONCRETE_PARAM now needs a
producer type that a pointer-returning exported `New*`/`Open*` constructor
builds, and `cmd/`, `commands/` and `tools/` (the composition roots, rule 4)
report no CONCRETE_* findings. That drops the count without changing any mcp
code, and it now reports `New*` and `Open*` calls alike and a same-package interface declared in a file that never uses it (`IFACE_AWAY`), so compare later PRs against the narrowed audit, not the 360/211 below.

## Why

CLAUDE.md's Go standards say the package that uses a capability declares the
small interface it needs, producer packages (`sde`, `esi`, `sso`, `creds`,
`config`, `analytics`) export concrete types, and a resolver wires the two.
`go run ./tools/ifaceaudit` listed 360 `CONCRETE_PARAM` and 211
`CONCRETE_CTOR` findings, almost all in `mcp/`: every tool took
`*config.Store` / `*sde.Store` and built its own `esi.Client` with
`esi.NewClient(...)` inside a package-level function variable that tests
overwrote.

## End state

- The outermost process builds the concrete producers and passes them down.
  `commands/mcp_tools.go` builds `*config.Store`, `*sde.Store` and the ESI
  clients and registers each tool with one explicit call:
  `mcp.AddListCharacterLoyaltyPoints(s, d.openESI, d.store)`. Adding a tool is
  its own file plus one line there.
- `mcp` never names `esi.Client`, `sde.Store` or `config.Store`, never calls
  `esi.New...`/`esi.Open...`, and has no union interface: no `Deps`,
  `ConfigStore`, `SDEStore` or `ESIClient`. Each tool file declares a small
  interface for exactly the calls it makes, a handler constructor that takes
  them, and an exported `Add...` function.
- A value passed in may have far more methods than the interface lists. A
  consumer that only calls `Do` declares a one-method `doer` and is handed an
  `*http.Client`; callers pass the concrete value as it is, with no mirror
  interface of the producer's API and no pared-down wrapper. Likewise an
  interface may be declared in the first file that uses it and reused by other
  files in the package.
- Tests fake the small interfaces and pass them to the handler constructor;
  they no longer assign package variables.
- `ifaceaudit` exempts the composition roots (`cmd/`, `commands/`, `tools/`)
  and fixture packages named `*test`, reports `CONCRETE_PARAM` only for
  producer types a `New...`/`Open...` constructor returns by pointer (clients
  and stores, not data types), reports `New...` and `Open...` calls alike, and
  reports `IFACE_AWAY` for an interface declared in a file that never uses it
  while another file in the package does. A CI test fails if a
  `CONCRETE_*` finding remains once the last file moves.

## The wiring

`commands.NewMCPServer(version, clientID, store, sdeStore)` calls
`mcp.NewServer(version)`, which returns a server carrying only the
instructions, then `addConvertedTools` (`commands/mcp_tools.go`), which
registers every tool, grouped by area in `commands/mcp_tools_*.go`. A new tool:

1. is its own file in `mcp/` with an exported `Add...` function carrying the
   tool's name and description;
2. gains one line in the matching `add<Area>Tools` function in `commands/`.

`toolDeps` in `commands/mcp_tools.go` holds the concrete producers and two
methods, `openESI(ctx, characterID) (*esi.Client, error)` and
`openPublicESI() *esi.Client`. They are passed as method values:
`d.openESI`, `d.openPublicESI`. The same `*esi.Client` serves every tool.

A nil `*sde.Store` (the cache failed to open) must not panic: every
`sde.Store` method is nil-safe, so `commands` passes it as it is, and the
enrichment resolvers in `enrich.go` treat a nil cache as "no names".

## The pattern for one tool file

Worked example: `mcp/loyalty.go`, `mcp/loyalty_internal_test.go` and the two
calls in `commands/mcp_tools.go`.

1. Declare the ESI calls the file makes, in that file, with the generated
   `esiapi` types (`esiapi "github.com/fnt-eve/goesi-openapi/esi"`):

   ```go
   // loyaltyESI is the ESI calls list_character_loyalty_points makes.
   type loyaltyESI interface {
       CharacterLoyaltyPoints(ctx context.Context) ([]esiapi.CharactersCharacterIdLoyaltyPointsGetInner, error)
   }
   ```

   One interface per set of calls that go together (public and authenticated
   tools usually differ). Copy the method signature from `esi/*.go`.
2. Replace the package variable that called `esi.NewClient` (or
   `NewPublicClient`) with a plain function that takes the client and does the
   mapping: `func listCharacterLoyaltyPoints(ctx context.Context, client loyaltyESI) (...)`.
   Delete the `esi` and, if now unused, `sso` imports.
3. The handler constructor is generic over the client:
   `func listCharacterLoyaltyPointsHandler[C loyaltyESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader)`.
   A public tool takes `open func() C` and a `userAgent string` for SDE calls.
   Because `C` is a type parameter, `commands` passes `d.openESI` returning
   `*esi.Client` directly, and the compiler checks it satisfies `loyaltyESI`.
   No adapter, no runtime type assertion. Tests instantiate `C` with the
   interface type itself.
4. The exported function registers the tool and carries its name and
   description: `func AddListCharacterLoyaltyPoints[C loyaltyESI](s *mcpsdk.Server, open ..., store configLoader)`.
   Argument order: server, `userAgent` (when needed), openers, config, SDE.
   `version` and `clientID` stay out of the signature; `commands` builds the
   user agent and the client ID lives in the opener. A tool with two client
   kinds takes two type parameters.
5. Config: a handler that only reads config takes `configLoader` (declared in
   `list_characters.go`, used by the other files). One that writes declares its
   own small interface in its file. There is no `ConfigStore`.
6. SDE: a function that needs a few SDE calls declares its own interface (see
   the `*NameCache` interfaces in `enrich.go`). Reuse an existing one when it
   matches. There is no `SDEStore`.
7. Tests: replace every `orig := x; x = func...; t.Cleanup` override of the ESI
   seam with a fake of the small interface declared in the `_test.go`, returning
   `esiapi` values, and call the handler constructor with it. Keep every
   scenario the file had, and add one for a failing `open`. Tests keep using
   `config.NewStoreAt(t.TempDir()+...)` for config.

## Shared functions and the planner family

A few ESI function variables are called from other files (`nav_tools.go`,
`hubs.go`, `overview.go`, `readiness.go`, `industry_*.go`, `haul_ships.go`,
`ship_tools.go`, `saved_structures.go`, `location_assets.go`, `planner.go`).
They become plain functions taking a small interface, declared next to the
function; a caller passes a client that satisfies it.

`planner` is a consumer too: it declares the interfaces for the SDE, config and
ESI calls it makes, in `planner.go`. The handlers that build a planner are
registered like any other tool, with generic `Add...` functions whose type
parameters are the planner's client interfaces, and the planner receives
non-generic openers made inside the `Add` function by a one-line conversion:
a `func(ctx, id) (C, error)` becomes a `func(ctx, id) (plannerClient, error)`
because `C` is constrained to satisfy `plannerClient`, which the compiler checks.

## Out of scope

Package-level function variables that are not ESI seams (for example
`resolveTypeNames` in `enrich.go`, which tests overwrite) stay for now.

## Stack

All three steps are done:

1. Foundation: the pattern, `commands.NewMCPServer`, `loyalty.go` as the worked
   example, the audit changes and the standards.
2. Every other file in `mcp/` moved to the pattern, each exporting `Add...`
   functions registered from `commands/`.
3. Cleanup: `Deps`, `ConfigStore`, `SDEStore` and `NewServer`'s registrations
   are deleted (`NewServer(version)` is the SDK server plus the instructions),
   `add_character` reaches the login flow through its own `characterLogin`
   interface that `commands/mcp_tools_admin.go` implements over `sso`, and
   the CI tests above keep the repo clean.
