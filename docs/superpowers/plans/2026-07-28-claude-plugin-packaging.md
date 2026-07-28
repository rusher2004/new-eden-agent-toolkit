# Claude Code Plugin Packaging + Versioning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Package `neat` as a Claude Code plugin that registers its MCP server on install, backed by a versioning system (`VERSION` file + generated plugin manifest + a deterministic CI gate + tag-and-release automation) adapted from the sibling `ci-cd-assistant` project.

**Architecture:** A single top-level `VERSION` file is the source of truth. `tools/genversion` propagates it into the plugin manifest (`go generate`-wired, CI-enforced no-diff). `tools/versiongate` is a required PR check that fails if release-significant files changed without a `VERSION` bump — no escape hatch, no external API dependency. `release.yml` tags `v<VERSION>` and runs GoReleaser on every push to `main` where that tag doesn't yet exist, building the `neat` binary for linux/darwin/windows from a single `macos-latest` runner (darwin needs `swiftc` for its embedded URL-scheme handler).

**Tech Stack:** Go 1.x, GoReleaser v2, GitHub Actions, `golang.org/x/mod/semver`.

## Global Constraints

- Plugin scope is MCP-registration only — no skills, no commands, no auto-download of the binary on install (see spec's Non-goals).
- No `ANTHROPIC_API_KEY` / AI-validated escape hatch anywhere in this system — the version gate is fully deterministic, two states only (pass/fail).
- `.claude-plugin/marketplace.json` must live at repo root (required for the `github` marketplace source type).
- The release job must run on `macos-latest` (not `ubuntu-latest`) because the darwin build needs `swiftc` (via `make urlscheme-macos-bundle`) before compiling — see `urlscheme/urlscheme_darwin.go`.
- Initial `VERSION` value is `0.1.0`.
- `ESI_CLIENT_ID` is injected into the release build via a GitHub Actions repo **variable** (not secret) — it is a PKCE public-app client ID, not sensitive (CLAUDE.md: "No client secret").
- Full spec: `docs/superpowers/specs/2026-07-28-claude-plugin-packaging-design.md`.

---

### Task 1: Plugin content skeleton + VERSION file

**Files:**
- Create: `VERSION`
- Create: `.claude-plugin/marketplace.json`
- Create: `claude_resources/neat/.claude-plugin/plugin.json`
- Create: `claude_resources/neat/.mcp.json`
- Create: `claude_resources/neat/CHANGELOG.md`
- Create: `claude_resources/README.md`

**Interfaces:**
- Produces: `VERSION` (plain-text semver, no `v` prefix, trailing newline) — read by Task 2's `genversion`, Task 3's `versiongate`, and Task 4's `release.yml`/`.goreleaser.yaml`.
- Produces: `claude_resources/neat/.claude-plugin/plugin.json` with exactly one top-level `"version"` field — the file Task 2's `genversion` regenerates.

- [ ] **Step 1: Create the VERSION file**

```bash
printf '0.1.0\n' > VERSION
```

- [ ] **Step 2: Create the marketplace manifest**

```bash
mkdir -p .claude-plugin
cat > .claude-plugin/marketplace.json << 'EOF'
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
EOF
```

- [ ] **Step 3: Create the plugin manifest**

```bash
mkdir -p claude_resources/neat/.claude-plugin
cat > claude_resources/neat/.claude-plugin/plugin.json << 'EOF'
{
  "name": "neat",
  "version": "0.1.0",
  "description": "MCP server for EVE Online via CCP's ESI API and the EVE Ref Static Data Export -- 217 tools covering wallet, assets, corporations, industry, universe data, and more.",
  "author": {
    "name": "rusher2004"
  },
  "homepage": "https://github.com/rusher2004/new-eden-agent-toolkit"
}
EOF
```

- [ ] **Step 4: Create the MCP server registration**

```bash
cat > claude_resources/neat/.mcp.json << 'EOF'
{
  "neat": {
    "command": "neat",
    "args": ["mcp"]
  }
}
EOF
```

- [ ] **Step 5: Create the plugin changelog**

```bash
cat > claude_resources/neat/CHANGELOG.md << 'EOF'
# neat plugin — changelog

## 0.1.0
- Initial release: registers the `neat` MCP server (217 tools covering ESI + a local SDE cache) with Claude Code.
EOF
```

- [ ] **Step 6: Create the claude_resources README**

```bash
cat > claude_resources/README.md << 'EOF'
# Claude resources — `neat`

Claude Code resources for `neat`. Currently a single plugin that registers the `neat` MCP server —
no skills or commands yet (see the project's design spec for what's deferred).

Contents:

```
new-eden-agent-toolkit/
├── .claude-plugin/
│   └── marketplace.json   # marketplace "neat" -- must live at repo root for the `github`
│                           # source type below to find it
└── claude_resources/
    └── neat/               # the plugin
        ├── .claude-plugin/plugin.json
        ├── .mcp.json       # registers the neat MCP server
        └── CHANGELOG.md
```

## Prerequisite: install the `neat` CLI

The plugin's `.mcp.json` expects `neat` to be resolvable on `PATH`. Download a release from
[GitHub Releases](https://github.com/rusher2004/new-eden-agent-toolkit/releases), put the `neat`
binary on your `PATH`, then run `neat setup` once to store ESI credentials in your OS keychain.

## Install the plugin (user scope)

**Two ordered steps** — step 1 registers the marketplace, step 2 installs the plugin from it:

```text
1) /plugin marketplace add rusher2004/new-eden-agent-toolkit
2) /plugin install neat@neat
```

This repo is **private**. Manual install/update reuses your existing git credentials (`gh auth
login`, SSH agent — whatever `git`/`gh` already use on this machine). Claude Code also
auto-refreshes git-backed marketplaces in the background; for that to work against a private repo
without an interactive prompt, set a `GITHUB_TOKEN` or `GH_TOKEN` environment variable. Otherwise,
pull the latest plugin content on demand with `/plugin marketplace update neat`.

## How to test

1. Complete the prerequisite above (`neat` on `PATH`, `neat setup` run once).
2. Install the plugin per the steps above.
3. In a Claude Code session, confirm the `neat` MCP server is connected (`claude mcp list` or ask
   Claude to call any `neat` tool, e.g. `get_status`).
EOF
```

- [ ] **Step 7: Verify all JSON files are valid**

Run: `jq . .claude-plugin/marketplace.json claude_resources/neat/.claude-plugin/plugin.json claude_resources/neat/.mcp.json`
Expected: all three print their parsed contents with no error.

- [ ] **Step 8: Commit**

```bash
git add VERSION .claude-plugin claude_resources
git commit -m "feat: add Claude Code plugin content skeleton + VERSION file"
```

---

### Task 2: genversion tool + go:generate wiring + Makefile update

**Files:**
- Create: `tools/genversion/main.go`
- Create: `tools/genversion/main_test.go`
- Modify: `cmd/main.go`
- Modify: `Makefile`

**Interfaces:**
- Consumes: `VERSION` (Task 1), `claude_resources/neat/.claude-plugin/plugin.json` (Task 1).
- Produces: `go generate ./...` (run from repo root or `cmd/`) regenerates `plugin.json`'s `"version"` field to match `VERSION`.

- [ ] **Step 1: Write the failing test**

Create `tools/genversion/main_test.go`:

```go
package main

import "testing"

func TestReplaceVersion(t *testing.T) {
	manifest := []byte(`{
  "name": "neat",
  "version": "0.1.0",
  "description": "x"
}`)

	out, changed, err := replaceVersion(manifest, "0.2.0")
	if err != nil {
		t.Fatalf("replaceVersion: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	want := `{
  "name": "neat",
  "version": "0.2.0",
  "description": "x"
}`
	if string(out) != want {
		t.Errorf("output mismatch:\n got: %s\nwant: %s", out, want)
	}
}

func TestReplaceVersionIdempotent(t *testing.T) {
	manifest := []byte(`{"version": "0.2.0"}`)
	out, changed, err := replaceVersion(manifest, "0.2.0")
	if err != nil {
		t.Fatalf("replaceVersion: %v", err)
	}
	if changed {
		t.Error("changed = true, want false when already in sync")
	}
	if string(out) != string(manifest) {
		t.Errorf("output altered when in sync: %s", out)
	}
}

func TestReplaceVersionRejectsZeroOrMultiple(t *testing.T) {
	if _, _, err := replaceVersion([]byte(`{"name": "x"}`), "0.2.0"); err == nil {
		t.Error("expected error when no version field present")
	}
	multi := []byte(`{"version": "0.1.0", "nested": {"version": "9.9.9"}}`)
	if _, _, err := replaceVersion(multi, "0.2.0"); err == nil {
		t.Error("expected error when more than one version field present (would clobber nested key)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tools/genversion/... -run TestReplaceVersion -v`
Expected: FAIL — `replaceVersion` undefined (package doesn't exist yet).

- [ ] **Step 3: Write the implementation**

Create `tools/genversion/main.go`:

```go
// Command genversion propagates the repo's single source-of-truth version
// (the top-level VERSION file) into files that must carry it literally.
//
// Right now that is the Claude Code plugin manifest
// (claude_resources/neat/.claude-plugin/plugin.json), whose "version" the
// plugin marketplace reads straight from the checked-out repo tree -- it
// cannot be injected at build time the way the neat binary's version is (via
// GoReleaser ldflags from the git tag). Keeping VERSION as the source of
// truth and regenerating the manifest from it means the binary, the plugin,
// and the release tag all derive from one number.
//
// It is wired via `go:generate` (see cmd/main.go) and run as a no-diff guard
// in CI: `go generate ./...` must produce no changes, mirroring the
// gofmt/go-fix checks. The replacement is a line-targeted regexp so the
// manifest's existing formatting and key order are preserved.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// repoRoot walks up from the working directory to the module root (the dir
// holding go.mod), so the tool works from `go generate` (run in cmd/) or the
// repo root alike.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

// pluginManifest is the file whose "version" must mirror VERSION.
var pluginManifest = filepath.Join("claude_resources", "neat", ".claude-plugin", "plugin.json")

// versionLine matches the manifest's `"version": "x.y.z"` line, capturing the
// surrounding punctuation so only the value is rewritten.
var versionLine = regexp.MustCompile(`("version":\s*")[^"]*(")`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genversion:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return fmt.Errorf("read VERSION: %w", err)
	}
	version := strings.TrimSpace(string(raw))
	if version == "" {
		return fmt.Errorf("VERSION is empty")
	}

	path := filepath.Join(root, pluginManifest)
	manifest, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", pluginManifest, err)
	}

	updated, changed, err := replaceVersion(manifest, version)
	if err != nil {
		return fmt.Errorf("%s: %w", pluginManifest, err)
	}
	if !changed {
		return nil // already in sync -- no write, no spurious mtime change
	}
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", pluginManifest, err)
	}
	fmt.Printf("genversion: set %s version to %s\n", pluginManifest, version)
	return nil
}

// replaceVersion sets the manifest's single top-level "version" value,
// returning the new bytes and whether anything changed. It requires exactly
// one "version" field so a future nested key can't be silently clobbered by a
// blanket replace.
func replaceVersion(manifest []byte, version string) (out []byte, changed bool, err error) {
	switch n := len(versionLine.FindAll(manifest, -1)); n {
	case 1:
		// expected
	case 0:
		return nil, false, fmt.Errorf(`no "version" field found`)
	default:
		return nil, false, fmt.Errorf(`expected exactly one "version" field, found %d`, n)
	}
	updated := versionLine.ReplaceAll(manifest, []byte("${1}"+version+"${2}"))
	return updated, string(updated) != string(manifest), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./tools/genversion/... -v`
Expected: PASS (all three tests).

- [ ] **Step 5: Wire go:generate into cmd/main.go**

In `cmd/main.go`, add the generate directive directly above the `package main` line's imports (as
the first non-package line):

```go
package main

//go:generate go run ../tools/genversion

import (
	"context"
	"fmt"
	"os"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)
```

- [ ] **Step 6: Update the Makefile's build target to read VERSION**

In `Makefile`, replace the hardcoded `main.version=dev` with a `VERSION`-file read (falling back to
`dev` when absent, matching the doc line CLAUDE.md already carries):

```makefile
build:
	go build -ldflags "-X main.version=$$(cat VERSION 2>/dev/null || echo dev) -X main.clientID=$$(jq -r .clientId app_config.json)" -o bin/neat ./cmd
```

- [ ] **Step 7: Verify go generate is a no-op when already in sync**

Run: `go generate ./...`
Expected: no output (plugin.json's version already matches VERSION from Task 1), exit code 0.

- [ ] **Step 8: Verify go generate updates a stale version**

Run:
```bash
sed -i '' 's/"version": "0.1.0"/"version": "0.0.9"/' claude_resources/neat/.claude-plugin/plugin.json
go generate ./...
git diff claude_resources/neat/.claude-plugin/plugin.json
```
Expected: `go generate` prints `genversion: set claude_resources/neat/.claude-plugin/plugin.json version to 0.1.0`, and the `git diff` shows the version field restored to `0.1.0` with no other changes.

- [ ] **Step 9: Confirm the full build still works**

Run: `go build ./...`
Expected: PASS, no errors.

- [ ] **Step 10: Commit**

```bash
git add tools/genversion cmd/main.go Makefile
git commit -m "feat: add genversion tool, wire go:generate, read VERSION in Makefile build"
```

---

### Task 3: versiongate tool

**Files:**
- Create: `tools/versiongate/main.go`
- Create: `tools/versiongate/main_test.go`
- Modify: `go.mod`, `go.sum` (adds `golang.org/x/mod`)

**Interfaces:**
- Consumes: `VERSION` (Task 1) at runtime (reads the file directly, not a Go import).
- Produces: `go run ./tools/versiongate`, invoked with `GITHUB_BASE_REF` and `GITHUB_OUTPUT` env vars set (by Task 4's `version-gate.yml`), writes `decision=pass|fail` and `reason=...` to the file at `$GITHUB_OUTPUT`.

- [ ] **Step 1: Add the semver dependency**

Run: `go get golang.org/x/mod@latest`
Expected: `go.mod` gains a `golang.org/x/mod` line; `go.sum` gains matching entries.

- [ ] **Step 2: Write the failing test**

Create `tools/versiongate/main_test.go`:

```go
package main

import "testing"

func TestIsExempt(t *testing.T) {
	exempt := []string{
		"README.md",
		"CLAUDE.md",
		"reference/cli-interface.md",
		"docs/reference/esi-coverage-roadmap.md",
		"docs/superpowers/plans/2026-07-28-x.md",
		".github/workflows/test.yml",
		".claude/settings.local.json",
		".gitignore",
		"VERSION",
		"sde/sync_test.go",
	}
	for _, p := range exempt {
		if !isExempt(p) {
			t.Errorf("isExempt(%q) = false, want true (should not require a bump)", p)
		}
	}

	significant := []string{
		"cmd/main.go",
		"sde/sync.go",
		"go.mod",
		"go.sum",
		".goreleaser.yaml",
		"claude_resources/neat/.mcp.json",
		"claude_resources/neat/.claude-plugin/plugin.json",
		"claude_resources/neat/CHANGELOG.md",
	}
	for _, p := range significant {
		if isExempt(p) {
			t.Errorf("isExempt(%q) = true, want false (should require a bump)", p)
		}
	}
}

func TestBumped(t *testing.T) {
	cases := []struct {
		base, head string
		want       bool
	}{
		{"0.1.0", "0.2.0", true},
		{"0.1.0", "0.1.1", true},
		{"v0.1.0", "v0.2.0", true}, // tolerate a leading v
		{"0.1.0", "0.1.0", false},  // unchanged
		{"0.2.0", "0.1.0", false},  // downgrade is not a bump
		{"", "0.1.0", true},        // base had no VERSION yet (treated as 0.0.0)
		{"0.1.0", "", false},       // missing/invalid head is not a bump
		{"0.9.0", "0.10.0", true},  // semver, not lexical (10 > 9)
	}
	for _, c := range cases {
		if got := bumped(c.base, c.head); got != c.want {
			t.Errorf("bumped(%q, %q) = %v, want %v", c.base, c.head, got, c.want)
		}
	}
}

func TestDecide(t *testing.T) {
	goChange := []string{"sde/sync.go"}
	docsOnly := []string{"README.md", "docs/x.md", "reference/cli-interface.md"}

	cases := []struct {
		name    string
		in      input
		want    decision
		wantSub string
	}{
		{"docs only, no bump", input{ChangedFiles: docsOnly, Base: "0.1.0", Head: "0.1.0"}, pass, "no release-significant"},
		{"code, no bump", input{ChangedFiles: goChange, Base: "0.1.0", Head: "0.1.0"}, fail, "bump VERSION"},
		{"code, bumped", input{ChangedFiles: goChange, Base: "0.1.0", Head: "0.2.0"}, pass, "bumped"},
		{"test-only change", input{ChangedFiles: []string{"sde/sync_test.go"}, Base: "0.1.0", Head: "0.1.0"}, pass, "no release-significant"},
		{"mixed docs+code, no bump", input{ChangedFiles: []string{"README.md", "sde/sync.go"}, Base: "0.1.0", Head: "0.1.0"}, fail, "bump VERSION"},
		{"plugin content, no bump", input{ChangedFiles: []string{"claude_resources/neat/.mcp.json"}, Base: "0.1.0", Head: "0.1.0"}, fail, "bump VERSION"},
		{"equal version with v prefix", input{ChangedFiles: goChange, Base: "0.1.0", Head: "v0.1.0"}, fail, "bump VERSION"},
	}
	for _, c := range cases {
		got, reason := decide(c.in)
		if got != c.want {
			t.Errorf("%s: decide = %v, want %v (reason: %s)", c.name, got, c.want, reason)
		}
		if c.wantSub != "" && !contains(reason, c.wantSub) {
			t.Errorf("%s: reason %q missing %q", c.name, reason, c.wantSub)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./tools/versiongate/... -v`
Expected: FAIL — `isExempt`, `bumped`, `decide`, `input`, `decision`, `pass`, `fail` undefined (package doesn't exist yet).

- [ ] **Step 4: Write the implementation**

Create `tools/versiongate/main.go`:

```go
// Command versiongate is the CI gate that keeps the VERSION file honest: a PR
// that changes release-significant files (the shipped binary or the plugin)
// must bump VERSION, while docs/CI/test-only PRs may merge without a bump
// (and therefore without cutting a release -- see release.yml, which only
// tags when v<VERSION> doesn't yet exist).
//
// It classifies the PR's changed files against an exempt allowlist; anything
// NOT on the allowlist is release-significant (unknown paths default to
// significant -- fail-safe). The decision is deterministic, with no escape
// hatch:
//
//	not significant  OR  VERSION strictly-greater than base  -> pass
//	significant, not bumped                                  -> fail (bump it)
//
// The decision is written to GITHUB_OUTPUT (decision=, reason=) for the
// workflow to act on; main exits 0 unless it hits an internal error (exit 2),
// so the workflow owns how a failing decision is surfaced.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"
)

type decision string

const (
	pass decision = "pass"
	fail decision = "fail"
)

type input struct {
	ChangedFiles []string
	Base         string // base branch VERSION value (may be empty if absent)
	Head         string // this PR's VERSION value
}

// isExempt reports whether a changed path is NOT release-significant: docs,
// CI/workflow config, editor/repo metadata, and test files don't ship in the
// binary or the plugin. Everything else (Go source, go.mod/sum, .goreleaser,
// and anything under claude_resources/ -- the plugin content the marketplace
// reads from the tree) requires a VERSION bump. Unlisted paths are significant.
func isExempt(path string) bool {
	switch {
	case strings.HasSuffix(path, "_test.go"):
		return true
	case strings.HasPrefix(path, "docs/"),
		strings.HasPrefix(path, "reference/"),
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

func anySignificant(files []string) bool {
	for _, f := range files {
		if f = strings.TrimSpace(f); f != "" && !isExempt(f) {
			return true
		}
	}
	return false
}

// ensureV canonicalizes a version to a valid semver "vX.Y.Z" string, or
// returns "" if it isn't valid semver.
func ensureV(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "v") {
		s = "v" + s
	}
	if !semver.IsValid(s) {
		return ""
	}
	return s
}

// bumped reports whether head is a strictly-greater semver than base. An
// empty base is treated as 0.0.0 (the PR that first introduces VERSION); an
// invalid head can never be a bump.
func bumped(base, head string) bool {
	h := ensureV(head)
	if h == "" {
		return false
	}
	b := ensureV(base)
	if b == "" {
		b = "v0.0.0"
	}
	return semver.Compare(h, b) > 0
}

func decide(in input) (decision, string) {
	if !anySignificant(in.ChangedFiles) {
		return pass, "no release-significant files changed; a VERSION bump is not required"
	}
	if bumped(in.Base, in.Head) {
		return pass, fmt.Sprintf("VERSION bumped %s -> %s", emptyAsNone(in.Base), in.Head)
	}
	return fail, fmt.Sprintf("release-significant files changed but VERSION was not bumped (still %s): bump VERSION", emptyAsNone(in.Base))
}

func emptyAsNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "versiongate:", err)
		os.Exit(2)
	}
}

func run() error {
	base := os.Getenv("GITHUB_BASE_REF")
	if base == "" {
		base = "main"
	}

	headRaw, err := os.ReadFile("VERSION")
	if err != nil {
		return fmt.Errorf("read VERSION: %w", err)
	}
	head := strings.TrimSpace(string(headRaw))

	in := input{
		ChangedFiles: changedFiles(base),
		Base:         baseVersion(base),
		Head:         head,
	}

	d, reason := decide(in)
	fmt.Printf("versiongate: %s -- %s\n", d, reason)
	return writeOutputs(d, reason)
}

// changedFiles lists files changed on the PR branch since its merge base with
// the target branch. An error (e.g. running outside a checkout) yields nil,
// which decide() treats as "nothing significant".
func changedFiles(base string) []string {
	out, err := exec.Command("git", "diff", "--name-only", "origin/"+base+"...HEAD").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "versiongate: warning: git diff failed (%v); treating as no changes\n", err)
		return nil
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

// baseVersion reads the VERSION file from the tip of the target branch, or ""
// if it doesn't exist there yet.
func baseVersion(base string) string {
	out, err := exec.Command("git", "show", "origin/"+base+":VERSION").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// writeOutputs emits the decision for the workflow via GITHUB_OUTPUT (when set).
func writeOutputs(d decision, reason string) (err error) {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	// Surface a Close error (e.g. a failed flush, which would drop the write)
	// when nothing earlier already failed.
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	// Reason is single-line by construction, so a plain key=value is safe.
	_, err = fmt.Fprintf(f, "decision=%s\nreason=%s\n", d, reason)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./tools/versiongate/... -v`
Expected: PASS (`TestIsExempt`, `TestBumped`, `TestDecide` all pass).

- [ ] **Step 6: Confirm the full build still works**

Run: `go build ./... && go vet ./...`
Expected: PASS, no errors.

- [ ] **Step 7: Commit**

```bash
git add tools/versiongate go.mod go.sum
git commit -m "feat: add versiongate tool (deterministic, no escape hatch)"
```

---

### Task 4: GitHub Actions workflows + GoReleaser config

**Files:**
- Create: `.github/scripts/gh-sticky-comment.sh`
- Create: `.github/workflows/version-gate.yml`
- Create: `.github/workflows/release.yml`
- Create: `.goreleaser.yaml`

**Interfaces:**
- Consumes: `tools/versiongate` (Task 3, invoked via `go run ./tools/versiongate`), `VERSION` (Task 1), `make urlscheme-macos-bundle` (existing Makefile target).
- Produces: a required "Version Gate / gate" PR check; a `release` GitHub Actions job that tags and publishes on merge to `main`.

- [ ] **Step 1: Create the sticky-comment helper script**

```bash
mkdir -p .github/scripts
cat > .github/scripts/gh-sticky-comment.sh << 'EOF'
#!/usr/bin/env bash
# Sticky PR-comment helpers for the Version Gate workflow. Source this file;
# it requires GH_TOKEN, GH_REPO, and PR in the environment.
#
# A single comment is reused (found and edited by a hidden marker) so re-runs
# on synchronize don't accumulate duplicate Version Gate comments -- the
# ::error:: annotation and the required check carry the blocking signal.

MARKER="<!-- version-gate -->"

# find_sticky prints the id of the existing Version Gate comment, or nothing.
# Tolerant of SIGPIPE from `head` closing the pipe under `set -o pipefail`.
find_sticky() {
  gh api "repos/$GH_REPO/issues/$PR/comments" --paginate \
    --jq ".[] | select(.body | startswith(\"$MARKER\")) | .id" 2>/dev/null | head -n1 || true
}

# upsert_comment <body>: edit the sticky comment if present, else create it.
upsert_comment() {
  local id body
  id="$(find_sticky)"
  body="$(printf '%s\n%s' "$MARKER" "$1")"
  if [ -n "$id" ]; then
    gh api -X PATCH "repos/$GH_REPO/issues/comments/$id" -f body="$body" >/dev/null
  else
    gh api -X POST "repos/$GH_REPO/issues/$PR/comments" -f body="$body" >/dev/null
  fi
}

# resolve_comment <body>: edit an existing sticky only (no-op when none
# exists), so PRs that never failed the gate don't get a comment at all.
resolve_comment() {
  local id body
  id="$(find_sticky)"
  [ -z "$id" ] && return 0
  body="$(printf '%s\n%s' "$MARKER" "$1")"
  gh api -X PATCH "repos/$GH_REPO/issues/comments/$id" -f body="$body" >/dev/null
}
EOF
chmod +x .github/scripts/gh-sticky-comment.sh
```

- [ ] **Step 2: Create the version-gate workflow**

```bash
cat > .github/workflows/version-gate.yml << 'EOF'
name: Version Gate

# Blocks a PR that changes release-significant files (the shipped binary or
# the plugin) without bumping the top-level VERSION file. Docs/CI/test-only
# PRs pass without a bump. There is no escape hatch -- every release-
# significant change must bump VERSION.
#
# Mark the "Version Gate / gate" check Required in branch protection to
# enforce.
on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: read
  pull-requests: write

env:
  GH_TOKEN: ${{ github.token }}
  GH_REPO: ${{ github.repository }}
  PR: ${{ github.event.pull_request.number }}
  BASE: ${{ github.base_ref }}

jobs:
  gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod

      - name: Evaluate version gate
        id: gate
        env:
          GITHUB_BASE_REF: ${{ github.base_ref }}
        run: go run ./tools/versiongate

      - name: Block — significant changes without a VERSION bump
        if: steps.gate.outputs.decision == 'fail'
        env:
          REASON: ${{ steps.gate.outputs.reason }}
        run: |
          set -euo pipefail
          source .github/scripts/gh-sticky-comment.sh
          upsert_comment "🚦 **Version Gate failed.** $REASON

          Bump the top-level \`VERSION\` file (then run \`go generate ./...\` so \`plugin.json\` matches)."
          echo "::error::$REASON"
          exit 1

      - name: Resolve sticky comment on pass
        if: steps.gate.outputs.decision == 'pass'
        env:
          REASON: ${{ steps.gate.outputs.reason }}
        run: |
          set -euo pipefail
          source .github/scripts/gh-sticky-comment.sh
          resolve_comment "✅ **Version Gate passed.** $REASON"
EOF
```

- [ ] **Step 3: Create the GoReleaser config**

```bash
cat > .goreleaser.yaml << 'EOF'
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
EOF
```

- [ ] **Step 4: Validate the GoReleaser config**

Run: `goreleaser check`
Expected: `1 configuration file(s) validated` with no errors. (Requires `goreleaser` installed locally — `brew install goreleaser` if missing.)

- [ ] **Step 5: Dry-run the GoReleaser build (snapshot mode, no publish, no tag needed)**

Run: `ESI_CLIENT_ID=dev-test goreleaser release --snapshot --clean --skip=publish`
Expected: builds complete for all 6 GOOS/GOARCH combinations (linux/darwin/windows × amd64/arm64), archives + checksums land in `dist/`. This is macOS-only (needs `swiftc` for the `before.hooks` step) — run it on a Mac.

- [ ] **Step 6: Create the release workflow**

```bash
cat > .github/workflows/release.yml << 'EOF'
name: Release

# Releases are driven by the top-level VERSION file, the single source of
# truth for the neat CLI version, the plugin manifest version, and the git
# tag. On a push to main, if no `v<VERSION>` tag exists yet, this workflow
# creates+pushes the tag and runs GoReleaser in the same job. Doing both here
# (rather than tagging in one workflow and releasing in another) sidesteps
# GitHub's rule that a tag pushed with GITHUB_TOKEN does not trigger another
# workflow. To cut a release: bump VERSION in a PR and merge.
# workflow_dispatch allows a manual re-run against the current VERSION.
#
# Runs on macos-latest (not ubuntu-latest): the darwin build embeds a
# swiftc-compiled .app bundle via go:embed (urlscheme/urlscheme_darwin.go),
# which needs `make urlscheme-macos-bundle` -- swiftc only exists on macOS.
# Go itself cross-compiles the linux/windows targets fine from this runner.
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
            echo "tag v$version already exists — nothing to release"
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
EOF
```

- [ ] **Step 7: Lint all new workflow YAML**

Run: `actionlint .github/workflows/version-gate.yml .github/workflows/release.yml`
Expected: no output (no errors).

- [ ] **Step 8: Commit**

```bash
git add .github/scripts .github/workflows .goreleaser.yaml
git commit -m "feat: add version-gate and release GitHub Actions workflows + GoReleaser config"
```

---

### Task 5: Docs updates + manual setup checklist

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: nothing new — pure documentation, describing Tasks 1-4's output.

- [ ] **Step 1: Update README.md's Status section**

Find the line in `README.md`'s `## Status` section reading:

```
Plugin/extension packaging doesn't exist yet. See the plans directory for what's next.
```

Replace it with:

```
The Claude Code plugin is available: `/plugin marketplace add rusher2004/new-eden-agent-toolkit`
then `/plugin install neat@neat` registers the `neat` MCP server (requires the `neat` binary on
`PATH` and `neat setup` run once — see `claude_resources/README.md`). The Claude Desktop extension
doesn't exist yet.
```

- [ ] **Step 2: Update CLAUDE.md's "doesn't exist yet" list**

Find the bullet in `CLAUDE.md` reading (approximately):

```
- **Agent-facing surfaces that don't exist yet** — once built in a later plan, add them to this
  list: the Claude Code plugin (`.claude-plugin/plugin.json` + `skills/`), the Claude Desktop
  extension (`extension/manifest.json`), and any `reference/*.md` API-contract docs. Don't leave a
  plan's own doc updates (e.g. its plan file, or a spec status line) as a "follow-up" task.
```

Replace it with (removing only the plugin item — Desktop extension and `reference/*.md` docs remain
future work):

```
- **Agent-facing surfaces that don't exist yet** — once built in a later plan, add them to this
  list: the Claude Desktop extension (`extension/manifest.json`), and any `reference/*.md`
  API-contract docs. Don't leave a plan's own doc updates (e.g. its plan file, or a spec status
  line) as a "follow-up" task.
```

- [ ] **Step 3: Add a Versioning/Release section to CLAUDE.md**

Immediately after CLAUDE.md's existing `## Build` section (after the `go test ./...` code block that
ends the `## Test` section, or immediately before it — place it as its own `## Versioning & Release`
section following `## Build`), add:

```markdown
## Versioning & Release

The top-level `VERSION` file is the single source of truth for the `neat` binary version, the
Claude Code plugin manifest version, and the release git tag. `tools/genversion` (wired via
`go:generate` in `cmd/main.go`) propagates it into `claude_resources/neat/.claude-plugin/plugin.json`
— run `go generate ./...` after bumping `VERSION` and commit the result.

`tools/versiongate` is a required PR check: any PR touching release-significant files (Go source,
`go.mod`/`go.sum`, `.goreleaser.yaml`, anything under `claude_resources/`) must bump `VERSION`. Docs,
tests, and CI config are exempt. There is no override — bump it.

Merging to `main` triggers `.github/workflows/release.yml`, which tags `v<VERSION>` (if that tag
doesn't already exist) and runs GoReleaser to build/publish `neat` for linux/darwin/windows.
```

- [ ] **Step 4: Verify the doc-sync rule is satisfied**

Run: `grep -n "doesn't exist yet" README.md CLAUDE.md`
Expected: README.md's line no longer mentions plugin packaging as missing; CLAUDE.md's line lists
only the Claude Desktop extension and `reference/*.md` docs.

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: document the Claude Code plugin install flow and versioning/release process"
```

---

## Final manual checklist (human only — not automatable by an implementer subagent)

After all 5 tasks are merged:

1. **Add the `ESI_CLIENT_ID` repo variable.** GitHub repo → Settings → Secrets and variables →
   Actions → Variables tab → New repository variable → name `ESI_CLIENT_ID`, value from the local
   git-ignored `app_config.json`'s `clientId` field (run `jq -r .clientId app_config.json` to get
   it). This must be a **variable**, not a secret — it's a PKCE public-app client ID, not sensitive.
2. **Mark "Version Gate / gate" as a required check.** GitHub repo → Settings → Branches → branch
   protection rule for `main` → add `Version Gate / gate` to required status checks.
3. **Confirm the first release.** After merging this branch's PR (which itself bumps nothing beyond
   what's already in `VERSION` at `0.1.0`), push to `main` should trigger `release.yml`, create tag
   `v0.1.0`, and publish a GitHub Release with linux/darwin/windows archives + checksums. Verify at
   `https://github.com/rusher2004/new-eden-agent-toolkit/releases`.
4. **Install the plugin end-to-end.** Follow `claude_resources/README.md`'s install steps in a real
   Claude Code session; confirm the `neat` MCP server connects and a tool call (e.g. `get_status`)
   succeeds.
