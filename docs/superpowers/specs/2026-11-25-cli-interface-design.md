# A command line interface for all of neat — design

Status: approved by the owner, 2026-11-25, including dropping the MCP server from the plugin's default registration (see "Decisions for the owner").

## Intent and why

neat's functionality is reachable only through its MCP server, which registers 277 tools and will register more. Every client that loads the server pays for those tool definitions in context, whether or not a session uses them. The goal is to make a command line interface the primary way agents use neat: an agent runs `neat <group> <verb>` through its shell, reads `--help` for only the group it needs, and gets JSON back. Token cost and context size drop because nothing is loaded until it is asked for.

The CLI is purely another interface. It must offer exactly the functionality of the MCP tools, with no differing behavior, and with no business logic written twice.

The wider aim, stated by the project owner as the test of the design: when this is done, neat could also be served as a REST API without modifying or calling into the CLI or MCP packages.

## Goals

- Every MCP tool is reachable as a CLI command, with the same inputs, the same JSON output and the same errors in substance.
- One implementation of each operation. The MCP adapter and the CLI adapter both call it.
- Business logic lives in packages that each have one stated responsibility, know nothing about how they are called, and know nothing about where data comes from (they take small interfaces they declare themselves, per the existing consumer-interface standard).
- A consistent, semantically named command tree, named by game concept, not by data source.
- Adding an operation stays a small, explicit step: write the operation, add one line to the composition root, add one command-path entry. Tests fail if a step is missed.
- All docs, plugin skills, server instructions and generated references are updated so the CLI is the strongly preferred way for agents to use neat.

## Non-goals

- Building the REST interface. This design only has to leave it possible.
- Changing what any operation does, its output shapes or its error semantics.
- Renaming or removing MCP tools. Their names and schemas stay as they are.
- Table or colored output, interactive prompts, shell completion. Output is JSON only.
- Changing the ESI, SDE or config data sources.

## Anti-goals (would make this a failure even if it works)

- A catch-all package holding "the logic that used to be in `mcp/`". Every package must pass the one-sentence responsibility test.
- Any domain package importing `mcp/`, `cli/`, the MCP SDK, urfave/cli or `net/http`.
- The CLI reaching into `mcp/`, or calling an operation through the MCP protocol.
- Any interface, adapter or domain package that exposes where data comes from (ESI tag names, ESI operation IDs) other than the `esi/` package itself.
- A second copy of any behavior: character checks, enrichment, output mapping, planning.

## Constraints

From `CLAUDE.md`, all still binding:

- Flat package layout, no `internal/`, one package per responsibility. Producers (`sde`, `esi`, `sso`, `creds`, `config`, `analytics`) export concrete types and no interfaces for callers; consumers declare the small interfaces they need; fakes live in the consumer's `_test.go`. `ifaceaudit` gates this in CI.
- The pure calculation packages (`nav`, `cargo`, `market`, `dogma`, `industry`) stay pure and keep their names.
- No fallbacks in new code, errors wrapped with context, external test packages by default, tests required for new behavior, no environment variables for config or credentials.
- Analytics send nothing identifying; any change to what is sent changes `PRIVACY.md` in the same commit.
- Every PR touching Go source bumps `VERSION`, regenerates (`go generate ./...`) and adds a `CHANGELOG.md` entry.
- ESI errors go through `describeError`; the SDE is read from `sde/` before ESI.

New constraints this work meets:

- Each CLI invocation is a separate process, where an MCP server is one long-lived process. See "Risks that come with one process per command".

## Approaches considered

**A. A generic adapter that drives the existing MCP server in-process.** The CLI builds the same server, lists its tools over an in-memory session and calls them through the MCP protocol. Least code, and automatic coverage of new tools. Rejected: the business logic stays in `mcp/`, so the CLI depends on the MCP interface and the REST test fails. Calling `esi/` directly instead was also rejected: `esi/` returns raw generated types and holds none of the behavior (character check, mapping to neat's shapes, name enrichment, planners), so a CLI doing that would duplicate it.

**B. Hand-written CLI commands, one per tool.** Best flags and help per command. Rejected: about 277 commands to write and keep in sync, and each tool's logic would still need to be callable without MCP types.

**C. Move the logic into packages named for what they do, and make `mcp/` and the CLI thin adapters. (Chosen.)** Two variants were rejected within it:

- A single `ops/` package for all moved logic. It moves the "does too much" problem to a new name.
- Domain packages cut by ESI tag, with a `neat esi <tag>` command tree. It lets the data source's grouping shape the interfaces. ESI's grouping is a useful guide because it models the game the way players meet it, but it is not the boundary.

The owner chose C with domain packages cut by game concept, flag-based owner selection, and no `esi` command namespace.

## Architecture

### Layers

| Layer | Packages | Responsibility |
|---|---|---|
| Data sources | `esi`, `sde`, `config`, `creds`, `sso`, `analytics` | Each talks to its one external thing. `esi` stays raw: it returns the generated types and nothing neat-specific. |
| Pure calculation | `nav`, `cargo`, `market`, `dogma`, `industry` | Unchanged. |
| Domain | one package per game concept (below) | neat's behavior for that concept: validate, call sources through interfaces it declares, map to neat's shapes, enrich. No knowledge of MCP, CLI, REST or any source's package. |
| Contract | `operation` | What an operation is: name, description, input and output types, run function, read or write. No logic. |
| Adapters | `mcp`, `cli` (later `rest`) | Decode a request into an operation's input, run it, encode the result or error. Nothing else. |
| Composition | `cmd`, `commands` | Build the real sources and domain services, collect their operations into one list, hand it to the adapters. |

### Domain packages

A package is cut by a sentence about what a player does or knows ("what contracts exist and who holds them"). Where a concept lines up with an ESI tag the package shares its name, but that is coincidence, not a rule: tags can be split or merged when the game's concepts say so, and one concept may draw on several tags.

Starting candidates, matching the CLI tree: `character` (identity, login, accounts, roles, titles, medals, standings, affiliations, notifications), `skill`, `clone`, `location`, `corporation`, `alliance`, `wallet`, `asset`, `contract`, `loyalty`, `fleet`, `mail`, `calendar`, `contact`, `fitting`, `killmail`, `war`, `factionwarfare`, `sovereignty`, `militarycampaign`, `freelancejob`, `project`, `skinr`, `accesslist`, `dogma`, `planet`, `structure` (Upwell structures and saved stations and structures), `universe` (the SDE-first, ESI-fallback read-through, ids, names, gate routes), `server`, `game` (the running EVE client), `incursion`, `insurance`, `search`, plus the planning concepts `logistics`, `haul`, `route`, `ship`, `trade` (orders, prices, hubs, validation, quotes) and `build` (facility profiles, blueprints, build and estimate planning).

Two existing pure packages are named `market` and `industry`, so the domain packages for market orders and for industry jobs and planning need other names; `trade` and `build` are working names. The first plan step produces the full assignment table of every tool to its package, for review before any code moves. That table is the builder's to settle, within the one-sentence test.

### The operation contract

```go
package operation

type Kind int
const ( Read Kind = iota; Write ) // Write changes state in the game or in ~/.neat

type Operation struct {
	Name, Description string
	Kind              Kind
	NewInput          func() any
	Run               func(ctx context.Context, in any) (any, error)
	// InputSchema / OutputSchema derive from the types' json and jsonschema tags.
}

func New[In, Out any](name, description string, kind Kind, run func(context.Context, In) (Out, error)) Operation
```

Each domain service has an `Operations()` method returning its operations. `commands/` builds every service with real sources and concatenates the lists. Operation names are today's MCP tool names, so MCP clients see no change. The contract carries nothing about data sources.

Operation descriptions and errors currently name other MCP tools ("from list_access_lists", "re-authorize via add_character"). Shared text must work for both adapters, so descriptions refer to other operations by a placeholder (for example `{{op list_access_lists}}`) that each adapter renders as a tool name or a command path, and errors that give instructions (`esi.MissingScopeError`) carry typed data (group, character) that each adapter words in its own terms.

### Domain example

`contract.Service` declares `characterContractsAPI`, `characterRegistry` and `typeNamer` interfaces for exactly the calls it makes, takes them in its constructor, and exposes `ListCharacterContracts(ctx, ListCharacterContractsInput) (ListCharacterContractsOutput, error)`. The unknown-character check, the ESI call, the mapping and the name enrichment all live there. `Operations()` wraps each method in `operation.New`. `*esi.Client` satisfies the interface where `commands/` builds it; typed-nil results follow the existing factory rule (return an untyped `nil, err`).

### Adapters

`mcp.Register(server, ops)` registers each operation with the SDK's low-level `AddTool`, using the operation's schemas and a read-only hint for `Read` operations. Tool names, descriptions (after placeholder rendering) and schemas must match today's. Moving off the SDK's typed `AddTool` loses its automatic input validation and output-schema generation, which the adapter must replace; the SDK's public schema generator and the existing self-referential-type panic rule (one non-recursive type per nesting level) still apply.

`cli.Build(ops)` builds the urfave/cli command tree. Flags come from the input struct's `json` and `jsonschema` tags: scalars become flags (kebab-case), arrays are repeatable flags, anything nested is passed with `--json '{...}'` or `--json -` (stdin). The action decodes flags into the operation's input, runs it, and writes compact JSON to stdout (`--pretty` for indented). Errors go to stderr as `{"error": "..."}` with exit code 1; usage errors exit 2 (existing convention). Operations of kind `Write` are marked as changing state in `--help`.

### Import-boundary check

A new `tools/` check, run by `go test ./...` like `ifaceaudit`, fails when a domain package imports `mcp`, `cli`, `commands`, the MCP SDK, urfave/cli or `net/http`, or when an adapter imports another adapter or a domain package other than through `operation`. A second test walks the operation list by reflection and fails when any operation's input or output type contains a type from the generated ESI module, so no interface sees the data source's shapes. Domain packages may still import the generated models inside their private interfaces and mapping code; that is where the mapping to neat's shapes happens, as in `mcp/` today (a check found no exported tool input or output type using an ESI type). The REST test is the standing definition: a `rest/` package could be added by passing the same operation list, with no change elsewhere.

## The command tree

Named by game concept. One top-level group per domain concept, then an optional sub-resource, then a verb.

```
neat
├── character     list | add | remove | update | scope-groups | get | roles | titles | medals
│                 | standings | portrait | notifications | affiliations lookup
│                 | corporation-history | agents-research
├── skill         list | attributes | queue
├── clone         list | implants | jump-clones
├── location      get | online | ship
├── corporation   get | members | titles | roles | standings | divisions | shareholders
│                 | medals | icons | facilities | member-tracking | alliance-history | npc list
├── alliance      list | get | corporations | icons
├── wallet        balance | journal | transactions          (--character | --corporation [--division])
├── asset         list | locations lookup | names lookup     (--character | --corporation)
├── contract      list | items | bids | public              (--character | --corporation | --region)
├── market        orders | order-history | history | prices | groups | types | structure
│   market hub        list | save | delete | validate | compare
│   market quote
├── loyalty       points | store-offers
├── industry      jobs | mining | blueprints | facilities | systems | plan-build | find-blueprints
│                 | check-skills | check-stock | solve-ore-mix | estimate-job-cost
│                 | estimate-research | estimate-invention | estimate-reprocessing
│   industry profile    list | save | delete | describe | calibrate
├── planet        list | get | schematic | customs-offices
├── structure     list | get | skyhooks | sovereignty-hubs | starbases
│   structure saved     list | save | delete
├── universe      type|system|region|constellation|group|category|station|… get|list
│                 | ids resolve | names resolve | route
├── route         plan | plan-jump
│   route saved         list | save | delete
├── haul          plan | ships | compare-hulls
│   haul profile        list | save | delete
├── ship          estimate-hold | estimate-cyno
├── logistics     overview | capabilities
├── fleet  mail  calendar  contact  fitting  killmail  war  faction-warfare  sovereignty
│   military-campaign  freelance-job  project  skinr  access-list  dogma  incursion
│   insurance  search  game  server
└── setup  mcp  version  analytics                           (unchanged)
```

Rules:

- Verbs come from a fixed set: `list`, `get`, `save`, `delete`, `add`, `update`, `remove`, `create`, `plan`, `estimate`, `check`, `find`, `compare`, `validate`, `lookup`, `resolve`. A test rejects any other.
- Owner selection is by flag: `--character`, `--corporation`, `--alliance`, `--fleet`, `--region`. A command that serves several owners takes one of them, exactly one required, and runs the matching operation. Path IDs become kebab-case flags (`--contract`, `--market-group`). Operations that need an acting character in addition to a corporation take both flags.
- A hand-maintained table in `cli/` links each operation name to its command path and owner flag. A test fails when an operation has no entry, when two operations share a path and owner, when a path uses an unknown verb or when a command has no description. The table is the only place command naming lives. It is not derived from the ESI spec.
- The ESI spec still matters for completeness, not for naming. A test in `esi/` that each spec operation is called by some method there stays (see `docs/reference/esi-coverage-roadmap.md`).
- `neat --help` lists groups only; each group's own help lists its verbs, so an agent reads only what it uses.

## Risks that come with one process per command

An MCP server is one long-lived process. A CLI run is one short process, and agents may run several at once. Each is a design item, resolved in the plan:

- **SDE freshness.** `sde/` checks the dump's ETag and may re-download once per process. Per-command that is a network round trip on every call. The check needs a persisted "last checked" time with a window (proposal: skip the check if made within the last few hours), without changing what a sync does.
- **Concurrent config writes.** The `config` lock is per process; two processes writing at once can lose a change, which is a documented limitation that parallel CLI calls make likely. `config.Update` needs a cross-process file lock.
- **Token refresh.** Two processes refreshing one character's token may race on a rotated refresh token. The keychain write after a refresh needs the same cross-process care.
- **Keychain and startup cost.** Every invocation reads the keychain; the existing slow-keychain notice applies. Startup must do only what the command needs (no work for `--help`).
- **`add_character`.** It blocks while the user uses the browser and holds port 48211. As a CLI command it prints progress to stderr and JSON to stdout at the end.

## Analytics

`tool_called` is recorded by an MCP middleware today. A CLI invocation is a short process, and flushing events adds network latency to every command. This design sends nothing from CLI invocations; `PRIVACY.md` says so. Usage counts will therefore under-report as agents move to the CLI; revisit if that matters.

## Migration

Done area by area, with the full test suite green after each step and no change visible to MCP clients:

1. Before moving anything, snapshot the MCP `tools/list` output (names, descriptions, input and output schemas) as a golden test. It stays through the migration; description changes from the placeholder rendering are the only expected differences and are reviewed.
2. Add `operation/`, the import-boundary check and the assignment table.
3. Move each area's logic out of `mcp/` into its domain package: the plain functions, the input and output types, the handlers' checks and enrichment, and the `mcp/*_internal_test.go` tests that cover them (fixing the API where a test needs an unexported symbol, per the testing standard). `mcp/` loses that file and gains nothing.
4. Replace `mcp`'s per-tool registration with `Register(ops)`; `commands/mcp_tools*.go` becomes `commands/operations.go`.
5. Add `cli/` and the command-path table once the operation list is complete, or earlier per area as each lands.
6. Generated references: `docs/reference/tools.md` stays (MCP); a new generated `docs/reference/cli.md` lists the tree. Both fail their tests when stale.

PR slicing is left to the plan. Each PR bumps `VERSION` and adds a `CHANGELOG.md` entry, since Go source changes.

## Documentation, skills and instructions

All updated in the same change as the code they describe, with the CLI presented as the preferred interface:

- `README.md`: the CLI first for agents, with example commands and the install; MCP as the alternative for clients that want it.
- `docs/README.md`, this repo's `CLAUDE.md` (package table, hard rules, conventions, "keeping docs in sync"), `.claude/review.md`.
- `mcp/instructions.go` (`Instructions`): tells clients to use the CLI via the shell where one is available, keeping the onboarding order, and renders command paths.
- The four plugin skills in `claude_resources/neat/skills/` (`add-character`, `haul-plan`, `industry-build-plan`, `industry-profile-setup`): their steps become CLI commands, still discovery and decisions only, never formulas.
- `claude_resources/README.md`, `claude_resources/neat/CHANGELOG.md` and the plugin and marketplace descriptions.
- `PRIVACY.md`: CLI invocations send nothing.

## Calls made after the builder review

These were settled by the author, not agreed in conversation; the owner can overturn any.

- **Same output, defined precisely.** The CLI prints the same output value an operation returns, marshaled by `encoding/json`, which is the value the MCP adapter returns as structured content. A test runs sample operations through both adapters over the same fakes and compares the JSON. The golden snapshot covers tool names and schemas.
- **Errors.** `Run` returns `(Out, error)`. Today's handlers return plain Go errors (none builds an `IsError` result itself), and the adapter maps them: MCP as a tool error result, CLI as `{"error": ...}` on stderr with exit 1. Anything a caller may branch on stays a typed or sentinel error from the domain package. Warnings that today travel inside a result (for example `add_character`'s `warnings`) stay fields of the output, so both adapters carry them unchanged.
- **Progress and side effects.** `add_character` needs to report progress while it waits on the browser. The operation package provides a small progress sink carried in the context (`operation.Progress(ctx, msg)`); the MCP adapter sends it as MCP progress or log notifications and the CLI writes it to stderr. State an operation saves, such as `plan_build`'s files under `~/.neat/plans/`, stays inside the domain package and the operation is marked `Write`.
- **Owner flags select between separate operations.** `list_character_contracts`, `list_corporation_contracts` and `list_public_contracts` stay three operations with their own schemas, as they are today. The `cli/` table maps one command path to a set of (owner flag, operation) pairs; its help merges the flags and each variant's description.
- **Incremental delivery.** `mcp/` can hold unmigrated tools (old `Add...` registration) beside `Register(ops)` while areas move, with the golden test covering both. `cli/` ships as soon as the first area is migrated and covers migrated areas only. The documentation, server instructions and skills switch to "prefer the CLI" only when the CLI covers every tool and the cross-process work below has landed, so agents are never pointed at a partial CLI.
- **Cross-process minimum.** The cross-process config lock, the token-refresh safety and the SDE check window are required before the documentation switches to CLI-first, not before the first area ships. The lock must work on Windows, which CI runs.
- **Output size.** The CLI does not truncate or cap output, since that would differ from the MCP. Paginated ESI results keep today's behavior (`AllPages`). Reducing output (field selection, limits) is a later, separate change that would apply to every adapter.
- **Stdout and stderr.** Stdout carries only the result JSON. Notices (a slow keychain, progress) go to stderr.

## Decisions for the owner (flagged, recommendation given)

- **The plugin's default MCP registration. Decided: drop it.** The token saving only happens if agents are not also loading the MCP tools, so the plugin stops registering the MCP server and its skills use the CLI; `neat mcp` stays for clients that want it (README documents the manual config). This is a user-visible change for existing plugin users and gets a prominent `CHANGELOG.md` entry. It takes effect with the documentation switch, when the CLI covers every tool, not before.
- **CLI analytics** (above): none for now.

## Left to the builder

The full tool-to-package assignment table, package names that collide with existing ones, exact flag names and their kebab-casing, the command-path table, the SDE freshness window, the lock mechanism for config and tokens, how `jsonschema` tags are read for flags, how the placeholder rendering is implemented, the PR slicing and the migration order within an area.
