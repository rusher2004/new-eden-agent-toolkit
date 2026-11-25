# Consumer-defined interfaces for the mcp package

Status: in progress. Foundation and the `loyalty.go` pilot landed in the first
PR of the stack; the remaining files follow in later PRs.

`tools/ifaceaudit` narrowed in that first PR: a CONCRETE_PARAM now needs a
producer type that a pointer-returning exported `New*`/`Open*` constructor
builds, and `cmd/`, `commands/` and `tools/` (the composition roots, rule 4)
report no CONCRETE_* findings. That drops the count without changing any mcp
code, so compare later PRs against the narrowed audit, not the 360/211 below.

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

- `mcp` never names `esi.Client`, `sde.Store` or `config.Store`, and never calls
  `esi.NewClient`, `esi.NewPublicClient` or `config.NewStore`.
- `commands/mcp.go` (the composition root) builds the concrete producers and
  passes them in as `mcp.Deps`.
- Each tool file declares a small interface for exactly the ESI, SDE and config
  calls it makes. Tests fake those interfaces; they no longer assign package
  variables.
- `ifaceaudit` exempts the composition roots (`cmd/`, `commands/`, `tools/`),
  reports `CONCRETE_PARAM` only for producer types a `New...`/`Open...`
  constructor returns by pointer (clients and stores, not data types), and a CI
  test (`TestRepoHasNoConcreteProducerUse`) fails if any finding remains.

## The wiring

`mcp.NewServer(version string, deps Deps)`:

```go
type Deps struct {
	ClientID      string
	Config        ConfigStore
	SDE           SDEStore
	OpenESI       func(ctx context.Context, characterID int32) (ESIClient, error)
	OpenPublicESI func() ESIClient
}
```

`ConfigStore`, `SDEStore` and `ESIClient` are the unions of what the tool files
need (`server.go`). `ESIClient` embeds each tool file's own interface, so the
real `*esi.Client` must satisfy all of them; `commands/mcp.go` returns it as an
`mcp.ESIClient`, so the compiler checks that. `ConfigStore` and `SDEStore` start
as every exported method of the concrete types and are pruned to the union of
the small interfaces once every file has its own.

`narrowESI[C]` and `narrowPublicESI[C]` in `server.go` turn `Deps.OpenESI` /
`Deps.OpenPublicESI` into an opener for one file's small interface `C`.

A nil `SDEStore` (the cache failed to open, or a test passes none) must not
panic: `commands` passes the typed-nil `*sde.Store`, whose methods are
nil-safe, and the enrichment resolvers in `enrich.go` also treat a nil
interface as "no store".

## The pattern for one tool file

Worked example: `mcp/loyalty.go` and `mcp/loyalty_internal_test.go`.

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
   mapping:
   `func listCharacterLoyaltyPoints(ctx context.Context, client loyaltyESI) (...)`.
   Delete the `esi` and, if now unused, `sso` imports.
3. Authenticated handlers take an opener, public ones a no-argument opener:
   `open func(ctx context.Context, characterID int32) (loyaltyESI, error)` /
   `open func() loyaltyStoreESI`. Open the client inside the handler after the
   character check, wrapping a failure as `fmt.Errorf("build ESI client: %w", err)`.
   The `version`/`clientID` parameters go away unless the handler still needs
   `version` for an SDE user agent (`sso.UserAgent(version)`).
4. Config: a handler that only reads config takes `configLoader` (declared in
   `list_characters.go`). One that writes declares its own small interface in its
   file. Do not use `ConfigStore` in a tool file.
5. SDE: a function that only needs a few SDE calls declares its own interface
   (see the `*NameCache` interfaces in `enrich.go`). Reuse an existing one when
   it matches. Do not use `SDEStore` in a tool file.
6. Wire it in `NewServer`:
   `listCharacterLoyaltyPointsHandler(narrowESI[loyaltyESI](deps.OpenESI), store)`
   and add the file's interface(s) to the `ESIClient` embed list.
7. Tests: replace every `orig := x; x = func...; t.Cleanup` override of the ESI
   seam with a fake of the small interface declared in the `_test.go`, returning
   `esiapi` values. Keep every scenario the file had (success, error, unknown
   character, ...), and add one for a failing `open`. Tests keep using
   `config.NewStoreAt(t.TempDir()+...)` for config.

## Shared function variables

A few ESI function variables are called from other files (the planner family:
`nav_tools.go`, `hubs.go`, `overview.go`, `readiness.go`, `industry_*.go`,
`haul_ships.go`, `ship_tools.go`, `saved_structures.go`, `location_assets.go`,
`planner.go`). They become the same plain functions taking a small interface;
the planner-family callers get a client from an opener (`planner` holds
`openESI func(ctx, characterID int32) (ESIClient, error)`, the wide interface,
since it spans many areas) and pass it in.

## Out of scope

Package-level function variables that are not ESI seams (for example
`resolveTypeNames` in `enrich.go`, which tests overwrite) stay for now.

## Stack

1. Foundation: `Deps`, `ConfigStore`/`SDEStore`, the resolver in
   `commands/mcp.go`, the audit change, the `loyalty.go` pilot.
2. Shared function variables and the planner family.
3. The remaining tool files, in parallel groups (corporation, character,
   fleets/mail/contacts, market/contracts/assets, industry/faction warfare,
   the rest).
4. Cleanup: prune `ConfigStore`/`SDEStore` to the union of the small interfaces,
   add the CI test, update CLAUDE.md and the docs.
