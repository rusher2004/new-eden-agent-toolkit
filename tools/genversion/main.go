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
