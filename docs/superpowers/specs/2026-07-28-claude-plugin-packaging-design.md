# Claude Code Plugin Packaging + Versioning — Design Spec

## Goal

`neat`'s own README and CLAUDE.md both call out "Claude Code plugin" as an agent-facing surface
that doesn't exist yet, alongside a Claude Desktop extension and `reference/*.md` docs. This spec
covers the first of those: package `neat` as a Claude Code plugin that registers its MCP server on
install, plus a versioning system solid enough to keep the plugin manifest, the release tag, and
the shipped binary all derived from one number.

Modeled directly on the sibling project `ci-cd-assistant`,
which already versions and ships a Claude Code plugin (`recon-deploy`) with an equivalent
architecture. Every file/mechanism below is a direct adaptation of that project's proven pattern,
not a from-scratch design — deviations are called out explicitly with the reason.

## Non-goals

- **No skills or slash commands in this plugin.** Scope is MCP server registration only. Unlike
  `ci-cd-assistant`'s `recon-deploy` plugin (which bundles skills that orchestrate the `recon` CLI),
  `neat` is fundamentally an MCP server — its 217 tools are the interface. Workflow skills/commands
  are an explicit future expansion, not part of this spec.
- **No automated binary download on plugin install.** Claude Code plugins have no first-class
  "run a script on install" hook. Binary distribution stays a separate manual step: download a
  GitHub Release asset (built by the release pipeline this spec adds), put it on `PATH`, run
  `neat setup` for keychain access — identical to `recon`'s documented install flow today.
- **No Claude Desktop extension.** Named in CLAUDE.md as a separate future surface; out of scope here.
- **No AI-validated version-gate escape hatch.** `ci-cd-assistant`'s `no-release` label is validated
  by a live call to the Claude API before being trusted. This project explicitly drops that
  dependency (see "Version gate" below) in favor of a fully deterministic rule with no escape hatch
  at all.

## Architecture

New top-level structure:

```
new-eden-agent-toolkit/
├── VERSION                                  # single source of truth, e.g. "0.1.0"
├── .goreleaser.yaml                         # cross-platform build/release config
├── .claude-plugin/
│   └── marketplace.json                     # marketplace "neat" -- github source, repo root
├── claude_resources/
│   ├── README.md                            # how to install/test the plugin
│   └── neat/                                # the plugin itself
│       ├── .claude-plugin/plugin.json       # name, version (generated), description, author
│       ├── .mcp.json                        # registers the neat MCP server on install
│       └── CHANGELOG.md                     # plugin-facing changelog, hand-maintained
├── tools/
│   ├── genversion/                          # go:generate: VERSION -> plugin.json
│   │   ├── main.go
│   │   └── main_test.go
│   └── versiongate/                         # CI gate logic
│       ├── main.go
│       └── main_test.go
└── .github/
    ├── scripts/gh-sticky-comment.sh         # sticky PR-comment helper (verbatim reuse)
    └── workflows/
        ├── version-gate.yml
        └── release.yml
```

`cmd/main.go` gains one line: `//go:generate go run ../tools/genversion`.

## Plugin content

**`.claude-plugin/marketplace.json`** (repo root — must live here for the `github` source type to
resolve without a local clone, matching `ci-cd-assistant`; this repo is also private, so the same
"set `GITHUB_TOKEN`/`GH_TOKEN` for background marketplace refresh" caveat applies):

```json
{
  "name": "neat",
  "owner": {
    "name": "rusher2004"
  },
  "metadata": {
    "description": "neat -- MCP server for interacting with EVE Online via CCP's ESI API and the EVE Ref Static Data Export."
  },
  "plugins": [
    {
      "name": "neat",
      "source": "./claude_resources/neat",
      "description": "Registers the neat MCP server (217 tools covering ESI + a local SDE cache) with Claude Code."
    }
  ]
}
```

**`claude_resources/neat/.claude-plugin/plugin.json`** (version field is generated, see below —
initial value `0.1.0`):

```json
{
  "name": "neat",
  "version": "0.1.0",
  "description": "MCP server for EVE Online via CCP's ESI API and the EVE Ref Static Data Export -- 217 tools covering wallet, assets, corporations, industry, universe data, and more.",
  "author": {
    "name": "rusher2004"
  },
  "homepage": "https://github.com/rusher2004/new-eden-agent-toolkit"
}
```

**`claude_resources/neat/.mcp.json`** — schema confirmed against several already-installed plugins
(`playwright`, `laravel-boost`, `firebase`: a flat map of server name → `{command, args}`, not
wrapped in an `mcpServers` key at the plugin-bundle level):

```json
{
  "neat": {
    "command": "neat",
    "args": ["mcp"]
  }
}
```

This assumes `neat` is on `PATH` — consistent with the "binary install is a separate manual step"
non-goal above.

**`claude_resources/neat/CHANGELOG.md`** — hand-maintained, one section per version, same format as
`ci-cd-assistant`'s (`## 0.1.0` heading, bullet list of user-visible changes). Editing this file
counts as a release-significant change under the version gate (see below), since it's the plugin
declaring what a release contains.

**`claude_resources/README.md`** — adapts `ci-cd-assistant`'s `claude_resources/README.md`: what the
plugin does, its file layout, the two-step install (`/plugin marketplace add
rusher2004/new-eden-agent-toolkit` then `/plugin install neat@neat`), and the prerequisite
(`neat setup` for keychain access) before the MCP server will work.

## Versioning mechanism

**`VERSION`** — a single line, no trailing content beyond a newline, e.g. `0.1.0`. Starts at
`0.1.0` since this is the first version-tagged release of the project.

**`tools/genversion`** — adapted from `ci-cd-assistant`'s tool verbatim except for one path
constant:

```go
var pluginManifest = filepath.Join("claude_resources", "neat", ".claude-plugin", "plugin.json")
```

Everything else (repo-root discovery via `go.mod`, the single-`"version"`-field regex guard, the
no-write-if-unchanged behavior, the CI no-diff enforcement via `go generate ./...`) is identical to
the reference implementation — it's a generic "propagate VERSION into one JSON file" tool with no
project-specific logic beyond that path.

## Version gate

**`tools/versiongate`** — same file-classification approach as `ci-cd-assistant`'s (`isExempt`
allowlist, fail-safe default of "significant" for anything not explicitly exempt), adapted
exempt-path list for `neat`'s actual tree:

```go
func isExempt(path string) bool {
    switch {
    case strings.HasSuffix(path, "_test.go"):
        return true
    case strings.HasPrefix(path, "docs/"),      // covers docs/reference/, docs/superpowers/
        strings.HasPrefix(path, "reference/"),  // future top-level reference/*.md (see CLAUDE.md)
        strings.HasPrefix(path, ".github/"),
        strings.HasPrefix(path, ".claude/"):
        return true
    case path == ".gitignore", path == "VERSION":
        return true
    case strings.HasSuffix(path, ".md") && !strings.HasPrefix(path, "claude_resources/"):
        return true
    }
    return false
}
```

Release-significant (everything not matched above): any non-test Go source file, `go.mod`/`go.sum`,
`.goreleaser.yaml`, and anything under `claude_resources/` (`plugin.json`, `.mcp.json`,
`CHANGELOG.md`) — the plugin content the marketplace reads straight from the repo tree.

**Deviation from `ci-cd-assistant`: no escape hatch.** The reference implementation's `no-release`
label (validated by a live Claude API call, with a `no-release-override` maintainer bypass) is
dropped entirely per explicit decision — this repo has one maintainer, and the judgment call the
label existed to gate is simpler to just decide by bumping `VERSION`, which is nearly free. The
decision collapses to two states instead of three:

```
not significant  OR  VERSION strictly-greater than base  -> pass
significant, not bumped                                  -> fail
```

No labels, no `needs_label_review`, no `ANTHROPIC_API_KEY` dependency anywhere in this system.

**`.github/workflows/version-gate.yml`** — same shape as the reference (checkout with
`fetch-depth: 0`, `setup-go`, run the gate, sticky-comment on fail via the reused
`gh-sticky-comment.sh`, resolve the sticky comment on pass), with the label-handling step and its
`pull_request` trigger types removed:

```yaml
on:
  pull_request:
    types: [opened, synchronize, reopened]
```

Mark "Version Gate / gate" Required in branch protection to enforce, same as the reference.

## Release workflow

**`.github/workflows/release.yml`** — same tag-if-missing-then-GoReleaser-in-one-job pattern as the
reference (sidesteps GitHub's rule that a tag pushed via `GITHUB_TOKEN` doesn't trigger another
workflow), with one deviation:

**Deviation: runs on `macos-latest`, not `ubuntu-latest`.** `neat`'s darwin build embeds a
`swiftc`-compiled `.app` bundle via `go:embed` (`urlscheme/urlscheme_darwin.go`), which requires
`make urlscheme-macos-bundle` to run before any Go build touching that file — and `swiftc` only
exists on macOS. Go itself cross-compiles the linux and windows targets fine from a macOS host, so
one `macos-latest` runner building all three targets is simpler than a multi-OS build matrix.

```yaml
name: Release

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: write

jobs:
  release:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Resolve version and check for an existing tag
        id: ver
        run: |
          version="$(tr -d '[:space:]' < VERSION)"
          if [ -z "$version" ]; then
            echo "VERSION is empty" >&2
            exit 1
          fi
          echo "version=$version" >> "$GITHUB_OUTPUT"
          if git rev-parse "v$version" >/dev/null 2>&1; then
            echo "release=false" >> "$GITHUB_OUTPUT"
          else
            echo "release=true" >> "$GITHUB_OUTPUT"
          fi

      - name: Create and push tag
        if: steps.ver.outputs.release == 'true'
        run: |
          git tag "v${{ steps.ver.outputs.version }}"
          git push origin "v${{ steps.ver.outputs.version }}"

      - uses: actions/setup-go@v6
        if: steps.ver.outputs.release == 'true'
        with:
          go-version-file: go.mod

      - name: Run GoReleaser
        if: steps.ver.outputs.release == 'true'
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          ESI_CLIENT_ID: ${{ vars.ESI_CLIENT_ID }}
```

**`.goreleaser.yaml`**:

```yaml
version: 2
project_name: neat

before:
  hooks:
    - go mod tidy
    - make urlscheme-macos-bundle

builds:
  - id: neat
    main: ./cmd
    binary: neat
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.clientID={{.Env.ESI_CLIENT_ID}}

archives:
  - id: neat
    ids:
      - neat
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ if eq .Os "darwin" }}mac{{ else }}{{ .Os }}{{ end }}_{{ .Arch }}'
    format_overrides:
      - goos: windows
        formats: [zip]

checksum:
  name_template: "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
  algorithm: sha256

changelog:
  use: github
  abbrev: 7
  sort: asc
  groups:
    - title: Features
      regexp: "(?i)^feat"
      order: 0
    - title: Bug fixes
      regexp: "(?i)^fix"
      order: 1
    - title: Other
      order: 999
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
      - Merge pull request
      - Merge branch
```

**Manual setup required (not automatable, one-time, repo owner action):** add a GitHub Actions repo
**variable** (Settings → Actions → Variables, not Secrets — this is a PKCE public-app client ID, not
sensitive per CLAUDE.md's "no client secret" statement) named `ESI_CLIENT_ID`, with the value
currently in the local git-ignored `app_config.json`'s `clientId` field. Without it, release builds
would embed an empty client ID.

## Docs updates

- **`README.md`**: Status section notes the plugin is now installable; add the two-step install
  instructions (mirrors `ci-cd-assistant`'s README "Install the plugin" section).
- **`CLAUDE.md`**: the Build section's `go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null
  || echo dev) ..."` line already anticipates a `VERSION` file that doesn't exist yet — this spec
  makes that real. Remove "the Claude Code plugin" from the "doesn't exist yet" bullet list (Claude
  Desktop extension and `reference/*.md` docs remain there).
- **`Makefile`**: `build` target reads `VERSION` (fallback `dev` if absent, matching CLAUDE.md's
  existing doc line) instead of hardcoding `-X main.version=dev`.

## Testing

This is packaging/CI, not application logic — no new production Go code beyond the two `tools/`
programs, which get direct unit tests:

- `tools/genversion/main_test.go` — mirrors the reference: regex replacement correctness, no-op
  when already in sync, error on zero or multiple `"version"` matches.
- `tools/versiongate/main_test.go` — mirrors the reference: `isExempt` classification table,
  `anySignificant`, the pass/fail decision matrix (now two states instead of three).

Manual validation after implementation (not automatable as a Go test — these exercise the actual
GitHub Actions environment):

1. Run `go generate ./...`; confirm `plugin.json`'s version now matches `VERSION`.
2. Open a throwaway PR touching a `.go` file without bumping `VERSION`; confirm the gate blocks it
   with the sticky comment.
3. Bump `VERSION` on that PR; confirm the gate now passes and the sticky comment resolves.
4. Run `goreleaser release --snapshot --clean` locally before ever pushing to `main`, to catch
   build/archive issues (especially the darwin/`swiftc` path) without cutting a real tag.
5. After merging to `main`, confirm `release.yml` tags `v<VERSION>` and produces a GitHub Release
   with all three platform archives + checksums.
6. Install the plugin locally end-to-end: `/plugin marketplace add rusher2004/new-eden-agent-toolkit`
   → `/plugin install neat@neat`, confirm the `neat` MCP server registers and a tool call succeeds.
