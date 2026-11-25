# Review instructions

The checklist for reviewing a PR (human or Claude) on top of the generic review (security, correctness, breaking changes, test coverage). The rules themselves live in `CLAUDE.md` ("Go standards" and "Hard rules"); this file says what to flag and at what bar. Keep it short: only rules that change review behavior belong here. The standards bind new and changed code; do not flag a violation in code the PR did not touch.

## Always check

1. **Consumer-defined interfaces** (`CLAUDE.md` "Go standards", rules 1-7). Flag:
   - an interface declared in a producer package (`sde/`, `esi/`, `sso/`, `creds/`, `config/`, `analytics/`)
   - new or changed code that takes a concrete producer type (`*sde.Store`, `*esi.Client`, ...) instead of a small interface of exactly the methods it calls
   - an interface with methods none of its callers use, or one declared away from its consumer. A concrete value with more methods than the interface lists is fine and expected (an `*http.Client` passed as a one-method `doer`), so do not ask for a narrower wrapper
   - a file that only declares types, or any `*_iface.go` file
   - a factory that returns a typed nil inside an interface; it must return an untyped `nil, err`

   `go run ./tools/ifaceaudit` lists the findings; CI enforces every kind but `INTERFACE` (`TestRepoConformsToConsumerInterfaceRules`), so a PR cannot add any.
2. **Error handling** (`CLAUDE.md` "Go standards"). Flag:
   - a dropped error: `_ = f()`, an ignored return, an empty `if err != nil {}`, a deferred `Close`/`Remove` whose error is discarded on a path that matters
   - an error returned without context across a layer or package boundary (`return err`), or wrapped with `%v`/`%s` where `%w` would keep the cause inspectable
   - a lower layer that logs or prints an error and carries on instead of returning it
   - a loop that swallows a per-item failure instead of collecting it with `errors.Join`
   - a failure a caller needs to branch on that has no exported sentinel or typed error
3. **No fallbacks in new code.** Flag best-effort matching, silent degradation or a fail-open path in new code. The documented best-effort paths (analytics, EVE Ref augmentation, the read-through ESI fallback, additive `*Name` enrichment, config seeding) are not findings; a new one without a written justification and a `CLAUDE.md` entry is.
4. **Tests.** New behavior has tests; security-sensitive paths (scope checks, SDE-vs-ESI routing, config writes) have no untested branch. Flag a test in the package under test that reaches an unexported identifier when a better exported API would do (say so), a change that exports something only to make a test pass, a test that touches the real keychain, `~/.neat/` or the network, and a library assertion package (`testify`).
5. **Hard rules** (`CLAUDE.md`). Flag any new code that breaks one: an ESI call for data the SDE has, a request without `sso.UserAgent`, an `.Execute()` error not routed through `describeError`, a login asking for every scope, an env var for config or credentials, a client secret, an analytics change that adds identifying data (or doesn't update `PRIVACY.md`), a self-referential output type.
6. **Surfaces in sync.** Changed tools, scope groups, CLI, credentials or architecture need the docs, skills, `mcp/instructions.go`, `CLAUDE.md` tables and `docs/reference/tools.md` updated in the same PR, and a `VERSION` bump with a `CHANGELOG.md` entry for release-significant changes (`tools/versiongate` is the backstop).
7. **GitHub Actions.** Third-party actions pinned to a full commit SHA with a `# vX.Y.Z` comment; no untrusted input (PR title, body, branch names) interpolated into `run:`.

## Severity

- **Important:** violations of checks 1, 2, 4 (security-sensitive paths) and 5; anything that changes behavior, leaks a credential or weakens a guard.
- **Nit at most:** naming, comment wording. Report at most five nits per review.

## Do not report

- Anything CI already enforces: gofmt, `go vet`, `go mod tidy`, `go generate`, `go test`, the version gate.
- `docs/superpowers/` (historical specs and plans).
- The contents of any `testdata/` directory.
- Standards violations in code the PR did not change.
