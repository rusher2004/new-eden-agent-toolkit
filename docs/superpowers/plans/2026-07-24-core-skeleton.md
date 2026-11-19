# Core Skeleton, Config & Creds Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up a buildable `neat` CLI (`neat version`, `neat setup`) backed by fully-tested `config/` and `creds/` packages — the foundation every later plan (auth/SSO, SDE, distribution) builds on.

**Architecture:** Flat package layout, no `internal/`. `cmd/main.go` is a thin entrypoint that hands off to `commands.Root(version)`, an `urfave/cli/v3` command tree. `config/` owns non-secret state at `~/.neat/config.json`. `creds/` wraps the OS keychain (`zalando/go-keyring`) for per-character refresh tokens. `neat setup` is the first command to use both.

**Tech Stack:** Go 1.26.1, `github.com/urfave/cli/v3`, `github.com/zalando/go-keyring`.

## Global Constraints

(Copied verbatim from `docs/superpowers/specs/2026-07-23-foundation-design.md`.)

- Module: `github.com/rusher2004/new-eden-agent-toolkit`. Go 1.26.1. Binary name: `neat`.
- `CGO_ENABLED=0` throughout — every dependency must be pure Go.
- Flat package structure, no `internal/`.
- The end user is not assumed to be technically knowledgeable. No terminal usage should be required for the golden path (this plan's `setup` command is a manual/troubleshooting entrypoint, not a golden-path requirement — see the distribution plan for how registration becomes automatic).
- Support adding zero or more authenticated EVE characters (not built in this plan, but `creds/`'s key shape must already support N characters, not a fixed set).
- Credentials go in the OS keychain; other config goes under `~/.neat/`. No environment variables (config/credential *storage* only — env vars used for *test* isolation, e.g. `HOME` overrides, are fine).
- `fmt.Errorf("operation: %w", err)` wrapping everywhere; never discard an error.
- External test packages by default (`package foo_test`); `*_internal_test.go` only for unexported-internals tests; table-driven, stdlib-only (no testify).
- Docs updated in the same commit/PR as the behavior they describe.

---

## File Structure

```
go.mod
go.sum
cmd/
  main.go              # entrypoint only
commands/
  commands.go          # Root(version) *cli.Command
  version.go            # `neat version`
  version_test.go
  setup.go              # `neat setup`
  setup_test.go
config/
  config.go             # Config, Character, Store
  config_test.go
creds/
  creds.go               # keychain wrapper, character refresh-token keys
  creds_test.go
CLAUDE.md
```

---

### Task 1: `commands` package skeleton + `version` subcommand

**Files:**
- Create: `go.mod`, `go.sum` (via `go mod init` / `go get`)
- Create: `commands/commands.go`
- Create: `commands/version.go`
- Test: `commands/version_test.go`

**Interfaces:**
- Produces: `commands.Root(version string) *cli.Command` — the root command every later task (main.go, and every future subcommand) adds to or invokes.

- [ ] **Step 1: Initialize the module and fetch the CLI dependency**

Run:
```bash
cd new-eden-agent-toolkit
go mod init github.com/rusher2004/new-eden-agent-toolkit
go get github.com/urfave/cli/v3
```
Expected: `go.mod` created with `module github.com/rusher2004/new-eden-agent-toolkit` and a `go 1.26.1` (or the installed toolchain's version) directive; `go.sum` created; no errors.

- [ ] **Step 2: Write the failing test**

Create `commands/version_test.go`:
```go
package commands_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

func TestVersionCommand_PrintsVersion(t *testing.T) {
	var out bytes.Buffer
	root := commands.Root("1.2.3")
	root.Writer = &out

	if err := root.Run(context.Background(), []string{"neat", "version"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "1.2.3" {
		t.Errorf("output = %q, want %q", got, "1.2.3")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./commands/... -run TestVersionCommand_PrintsVersion -v`
Expected: FAIL to compile — `package commands is not in std` / `no Go files in .../commands` or `undefined: commands.Root` (the `commands` package doesn't exist yet).

- [ ] **Step 4: Write minimal implementation**

Create `commands/commands.go`:
```go
// Package commands defines neat's CLI command tree.
//
// Named commands, not cli, to avoid shadowing the github.com/urfave/cli/v3 import.
package commands

import "github.com/urfave/cli/v3"

// Root builds the top-level neat command, wiring in every subcommand.
func Root(version string) *cli.Command {
	return &cli.Command{
		Name:  "neat",
		Usage: "New Eden Agent Toolkit",
		Commands: []*cli.Command{
			versionCommand(version),
		},
	}
}
```

Create `commands/version.go`:
```go
package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func versionCommand(version string) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print the neat version",
		Action: func(_ context.Context, cmd *cli.Command) error {
			fmt.Fprintln(cmd.Writer, version)
			return nil
		},
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./commands/... -run TestVersionCommand_PrintsVersion -v`
Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum commands/commands.go commands/version.go commands/version_test.go
git commit -m "feat: add neat command tree with version subcommand"
```

---

### Task 2: `cmd/main.go` entrypoint

**Files:**
- Create: `cmd/main.go`

**Interfaces:**
- Consumes: `commands.Root(version string) *cli.Command` (Task 1).
- Produces: the `neat` binary (`go build ./cmd`), consumed by every later manual verification step and eventually by release tooling.

- [ ] **Step 1: Write the entrypoint**

Create `cmd/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	root := commands.Root(version)
	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "neat: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build the binary**

Run: `go build -o /tmp/neat-test ./cmd`
Expected: exits 0, no output.

- [ ] **Step 3: Verify the default version**

Run: `/tmp/neat-test version`
Expected: prints `dev`

- [ ] **Step 4: Verify ldflags injection**

Run:
```bash
go build -ldflags "-X main.version=9.9.9" -o /tmp/neat-test-v ./cmd
/tmp/neat-test-v version
```
Expected: prints `9.9.9`

- [ ] **Step 5: Commit**

```bash
git add cmd/main.go
git commit -m "feat: add neat CLI entrypoint"
```

---

### Task 3: `config` package

**Files:**
- Create: `config/config.go`
- Test: `config/config_test.go`

**Interfaces:**
- Produces:
  - `config.Character{ID int32, Name string}`
  - `config.Config{Characters []config.Character, SDEVersion string}`
  - `config.NewStore() (*config.Store, error)` — resolves `~/.neat/config.json`
  - `config.NewStoreAt(path string) *config.Store` — for test/CLI injection
  - `(*config.Store) Load() (config.Config, error)` — missing file returns zero-value `Config`, not an error
  - `(*config.Store) Save(cfg config.Config) error` — atomic write, creates parent directories
  - `(*config.Store) Path() string`
- Consumed by: Task 5 (`setup` command), and by every future plan that needs to read/write known characters or the SDE version.

- [ ] **Step 1: Write the failing tests**

Create `config/config_test.go`:
```go
package config_test

import (
	"path/filepath"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestStore_Load_MissingFileReturnsZeroValue(t *testing.T) {
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.Characters) != 0 {
		t.Errorf("Characters = %v, want empty", cfg.Characters)
	}
	if cfg.SDEVersion != "" {
		t.Errorf("SDEVersion = %q, want empty", cfg.SDEVersion)
	}
}

func TestStore_SaveThenLoad_RoundTrips(t *testing.T) {
	// Nested, not-yet-created directory: Save must create it.
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "nested", "config.json"))
	want := config.Config{
		Characters: []config.Character{{ID: 12345, Name: "Test Pilot"}},
		SDEVersion: "sde-20250303",
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got.Characters) != 1 || got.Characters[0] != want.Characters[0] {
		t.Errorf("Characters = %v, want %v", got.Characters, want.Characters)
	}
	if got.SDEVersion != want.SDEVersion {
		t.Errorf("SDEVersion = %q, want %q", got.SDEVersion, want.SDEVersion)
	}
}

func TestStore_Path_ReturnsConfiguredPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := config.NewStoreAt(path)

	if got := store.Path(); got != path {
		t.Errorf("Path() = %q, want %q", got, path)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./config/... -v`
Expected: FAIL to compile — `config` package doesn't exist yet.

- [ ] **Step 3: Write minimal implementation**

Create `config/config.go`:
```go
// Package config manages neat's non-secret user state, stored at
// ~/.neat/config.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Character identifies a known EVE character by ID and display name.
type Character struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Config is neat's persisted, non-secret state.
type Config struct {
	Characters []Character `json:"characters"`
	SDEVersion string      `json:"sdeVersion"`
}

// Store reads and writes a Config to a fixed path on disk.
type Store struct {
	path string
}

// NewStore resolves the default config path, ~/.neat/config.json.
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	return NewStoreAt(filepath.Join(home, ".neat", "config.json")), nil
}

// NewStoreAt returns a Store backed by an explicit path, for tests and
// callers that need to override the location.
func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

// Path returns the file path this Store reads from and writes to.
func (s *Store) Path() string {
	return s.path
}

// Load reads the Config from disk. A missing file is not an error — it
// returns a zero-value Config, so first-run has nothing special to handle.
func (s *Store) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", s.path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", s.path, err)
	}
	return cfg, nil
}

// Save writes cfg to disk atomically (temp file + rename), creating parent
// directories as needed.
func (s *Store) Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename temp config into place: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./config/... -v`
Expected: `PASS` (all 3 tests)

- [ ] **Step 5: Commit**

```bash
git add config/config.go config/config_test.go
git commit -m "feat: add config package for ~/.neat/config.json"
```

---

### Task 4: `creds` package

**Files:**
- Create: `creds/creds.go`
- Test: `creds/creds_test.go`

**Interfaces:**
- Produces:
  - `const creds.Service = "neat"`
  - `var creds.ErrNotFound error`
  - `creds.SaveCharacterRefreshToken(characterID int32, token string) error`
  - `creds.CharacterRefreshToken(characterID int32) (string, error)` — returns an error wrapping `creds.ErrNotFound` when absent
  - `creds.DeleteCharacterRefreshToken(characterID int32) error` — deleting an absent token is success
- Consumed by: Task 5 (`setup`'s keychain verification, via `creds.Service`), and by the `sso/` package in the next plan (auth/SSO) for persisting refresh tokens.

- [ ] **Step 1: Fetch the keychain dependency**

Run: `go get github.com/zalando/go-keyring`

- [ ] **Step 2: Write the failing tests**

Create `creds/creds_test.go`:
```go
package creds_test

import (
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

func TestMain(m *testing.M) {
	// Never touch the real OS keychain in tests.
	keyring.MockInit()
	os.Exit(m.Run())
}

func TestCharacterRefreshToken_MissingReturnsErrNotFound(t *testing.T) {
	_, err := creds.CharacterRefreshToken(999999)
	if !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound", err)
	}
}

func TestSaveThenGetCharacterRefreshToken_RoundTrips(t *testing.T) {
	const characterID = 123456
	const token = "refresh-token-value"

	if err := creds.SaveCharacterRefreshToken(characterID, token); err != nil {
		t.Fatalf("SaveCharacterRefreshToken returned error: %v", err)
	}

	got, err := creds.CharacterRefreshToken(characterID)
	if err != nil {
		t.Fatalf("CharacterRefreshToken returned error: %v", err)
	}
	if got != token {
		t.Errorf("token = %q, want %q", got, token)
	}
}

func TestDeleteCharacterRefreshToken_RemovesIt(t *testing.T) {
	const characterID = 654321
	if err := creds.SaveCharacterRefreshToken(characterID, "temp"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken returned error: %v", err)
	}

	if err := creds.DeleteCharacterRefreshToken(characterID); err != nil {
		t.Fatalf("DeleteCharacterRefreshToken returned error: %v", err)
	}

	if _, err := creds.CharacterRefreshToken(characterID); !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound after delete", err)
	}
}

func TestDeleteCharacterRefreshToken_AlreadyAbsentIsSuccess(t *testing.T) {
	if err := creds.DeleteCharacterRefreshToken(999998); err != nil {
		t.Errorf("DeleteCharacterRefreshToken returned error for absent key: %v", err)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./creds/... -v`
Expected: FAIL to compile — `creds` package doesn't exist yet.

- [ ] **Step 4: Write minimal implementation**

Create `creds/creds.go`:
```go
// Package creds wraps OS keychain access for neat's stored credentials.
package creds

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// Service is the OS keychain service name under which every neat
// credential is stored, regardless of which character it belongs to.
const Service = "neat"

// ErrNotFound is returned (wrapped) when a requested credential does not
// exist in the keychain.
var ErrNotFound = errors.New("credential not found")

// refreshTokenKey builds the per-character keychain key. neat supports zero
// or more characters, so keys are generated per character rather than drawn
// from a fixed set of named constants.
func refreshTokenKey(characterID int32) string {
	return fmt.Sprintf("character_refresh_token:%d", characterID)
}

// SaveCharacterRefreshToken persists the ESI refresh token for a character.
func SaveCharacterRefreshToken(characterID int32, token string) error {
	if err := keyring.Set(Service, refreshTokenKey(characterID), token); err != nil {
		return fmt.Errorf("save refresh token for character %d: %w", characterID, err)
	}
	return nil
}

// CharacterRefreshToken retrieves the ESI refresh token for a character.
func CharacterRefreshToken(characterID int32) (string, error) {
	token, err := keyring.Get(Service, refreshTokenKey(characterID))
	if errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("refresh token for character %d: %w", characterID, ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("get refresh token for character %d: %w", characterID, err)
	}
	return token, nil
}

// DeleteCharacterRefreshToken removes the ESI refresh token for a character.
// Deleting an already-absent token is treated as success.
func DeleteCharacterRefreshToken(characterID int32) error {
	err := keyring.Delete(Service, refreshTokenKey(characterID))
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete refresh token for character %d: %w", characterID, err)
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./creds/... -v`
Expected: `PASS` (all 4 tests)

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum creds/creds.go creds/creds_test.go
git commit -m "feat: add creds package for keychain-backed refresh tokens"
```

---

### Task 5: `setup` command

**Files:**
- Create: `commands/setup.go`
- Test: `commands/setup_test.go`

**Interfaces:**
- Consumes: `config.NewStore`, `config.NewStoreAt`, `config.Config` (Task 3); `creds.Service` (Task 4).
- Produces: `neat setup` wired into `commands.Root` — the manual/troubleshooting entrypoint later plans' `neat mcp` startup will also call into automatically (see the auth/SSO plan).

- [ ] **Step 1: Write the failing tests**

Create `commands/setup_test.go`:
```go
package commands_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestSetupCommand_CreatesConfigAndVerifiesKeychain(t *testing.T) {
	keyring.MockInit()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	var out bytes.Buffer
	root := commands.Root("dev")
	root.Writer = &out

	if err := root.Run(context.Background(), []string{"neat", "setup"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	wantConfigPath := filepath.Join(tmpHome, ".neat", "config.json")
	if _, err := os.Stat(wantConfigPath); err != nil {
		t.Errorf("expected config file at %s, stat error: %v", wantConfigPath, err)
	}

	got := out.String()
	if !strings.Contains(got, "Config directory ready") {
		t.Errorf("output = %q, want it to mention config directory readiness", got)
	}
	if !strings.Contains(got, "Keychain access verified") {
		t.Errorf("output = %q, want it to mention keychain verification", got)
	}
}

func TestSetupCommand_RunTwice_DoesNotClobberExistingConfig(t *testing.T) {
	keyring.MockInit()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	if err := commands.Root("dev").Run(context.Background(), []string{"neat", "setup"}); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}

	// Simulate a character already having been added between setup runs.
	store := config.NewStoreAt(filepath.Join(tmpHome, ".neat", "config.json"))
	want := config.Config{Characters: []config.Character{{ID: 1, Name: "Existing Pilot"}}}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := commands.Root("dev").Run(context.Background(), []string{"neat", "setup"}); err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got.Characters) != 1 || got.Characters[0] != want.Characters[0] {
		t.Errorf("Characters = %v, want existing config preserved: %v", got.Characters, want.Characters)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./commands/... -run TestSetupCommand -v`
Expected: FAIL to compile — `undefined: setup` wiring / `neat setup` not a registered command (no matching test output yet since `setup.go` doesn't exist).

- [ ] **Step 3: Write minimal implementation**

Create `commands/setup.go`:
```go
package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

const keychainCanaryKey = "setup_canary"

func setupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "prepare neat's local config directory and verify keychain access",
		Action: func(_ context.Context, cmd *cli.Command) error {
			store, err := config.NewStore()
			if err != nil {
				return fmt.Errorf("resolve config store: %w", err)
			}
			return runSetup(cmd.Writer, store)
		},
	}
}

func runSetup(w io.Writer, store *config.Store) error {
	if _, err := os.Stat(store.Path()); errors.Is(err, os.ErrNotExist) {
		if err := store.Save(config.Config{}); err != nil {
			return fmt.Errorf("initialize config file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check config file: %w", err)
	}
	fmt.Fprintf(w, "Config directory ready at %s\n", store.Path())

	if err := verifyKeychain(); err != nil {
		return fmt.Errorf("keychain check: %w", err)
	}
	fmt.Fprintln(w, "Keychain access verified.")

	return nil
}

// verifyKeychain round-trips a throwaway value through the OS keychain so
// setup fails loudly (rather than treating a locked/unavailable keychain as
// "unconfigured").
func verifyKeychain() error {
	if err := keyring.Set(creds.Service, keychainCanaryKey, "ok"); err != nil {
		return fmt.Errorf("write canary credential: %w", err)
	}
	if _, err := keyring.Get(creds.Service, keychainCanaryKey); err != nil {
		return fmt.Errorf("read canary credential: %w", err)
	}
	if err := keyring.Delete(creds.Service, keychainCanaryKey); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete canary credential: %w", err)
	}
	return nil
}
```

Update `commands/commands.go`'s `Commands` slice to add `setupCommand()`:
```go
		Commands: []*cli.Command{
			versionCommand(version),
			setupCommand(),
		},
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./commands/... -v`
Expected: `PASS` (all tests, including Task 1's version test)

- [ ] **Step 5: Commit**

```bash
git add commands/commands.go commands/setup.go commands/setup_test.go
git commit -m "feat: add neat setup command"
```

---

### Task 6: `CLAUDE.md` + final verification

**Files:**
- Create: `CLAUDE.md`

**Interfaces:**
- None — documentation only, describing the packages Tasks 1-5 produced.

- [ ] **Step 1: Write CLAUDE.md**

Create `CLAUDE.md`:
```markdown
# neat — New Eden Agent Toolkit

Go MCP server + Claude Code plugin + Claude Desktop extension for interacting with EVE Online via
FC's ESI API and the EVE Ref Static Data Export (SDE). Design spec:
`docs/superpowers/specs/2026-07-23-foundation-design.md`.

## Architecture

Flat package structure at repo root. No `internal/`. One package per responsibility:

| Package | Responsibility |
|---|---|
| `cmd/` | `main.go` entrypoint only |
| `commands/` | `urfave/cli/v3` command tree (`version`, `setup`, and more to come) |
| `config/` | Non-secret state in `~/.neat/config.json` |
| `creds/` | OS keychain wrapper (`zalando/go-keyring`), service name `"neat"` |

More packages (`sso/`, `urlscheme/`, `esi/`, `sde/`, `mcp/`) land in follow-up plans — see
`docs/superpowers/plans/`.

## Hard rules

- **SDE over ESI**: any tool needing a resource that exists in the SDE must read it from `sde/`,
  never call ESI for it. Performance and API-citizenship requirement, not a suggestion.
- **No environment variables for config/credentials.** Config lives under `~/.neat/`; credentials
  live in the OS keychain only.
- **No client secret.** The ESI app is a PKCE-only public app — nothing secret ships in the binary.

## Conventions

- Wrap errors with context: `fmt.Errorf("operation: %w", err)`. Never discard an error (`_ = err`).
- External test packages by default (`package foo_test`); use `*_internal_test.go` only when a
  test genuinely needs an unexported symbol.
- Table-driven tests, standard library only — no `testify` or other assertion libraries.
- `creds/` tests must call `keyring.MockInit()` before touching keychain functions — never let
  tests hit the real OS keychain.
- Docs (this file, the design spec, package docs) get updated in the same commit as the behavior
  they describe — no separate "update docs" commits.

## Build

```bash
go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev)" -o neat ./cmd
```

## Test

```bash
go test ./...
```
```

- [ ] **Step 2: Run the full test suite**

Run: `go test ./... -v`
Expected: `PASS` for every package (`commands`, `config`, `creds`)

- [ ] **Step 3: Vet and build check**

Run:
```bash
go vet ./...
go build ./...
```
Expected: both exit 0 with no output.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add CLAUDE.md documenting core skeleton conventions"
```

---

## Plan Self-Review Notes

- **Spec coverage** (for this plan's declared scope only — auth/SSO, SDE, and distribution are
  separate plans): repo/module setup ✓ (Task 1), CLI entrypoint ✓ (Task 2), `config/` per the
  spec's package table ✓ (Task 3), `creds/` per the spec's package table including the
  per-character key shape ✓ (Task 4), a real `setup` command exercising both ✓ (Task 5),
  conventions documented in `CLAUDE.md` ✓ (Task 6).
- **Placeholder scan**: none — every step has complete, runnable code and exact expected output.
- **Type consistency**: `config.Character{ID int32, Name string}` and
  `config.Config{Characters []Character, SDEVersion string}` are used identically in Task 3's
  implementation and Task 5's tests. `creds.Service`, `creds.ErrNotFound`, and the three
  `creds.*CharacterRefreshToken` functions are used identically in Task 4's implementation and
  Task 5's `verifyKeychain`.
