//go:build darwin

package urlscheme

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed macos/build/NeatURLHandler.app
var bundleFS embed.FS

const bundleFSRoot = "macos/build/NeatURLHandler.app"

// lsregisterPath is lsregister's fixed, well-known location. It is not on
// $PATH on a stock macOS install, so it must be invoked by full path.
const lsregisterPath = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

// register ignores launcherPath: the embedded Swift wrapper hardcodes
// ~/.neat/bin/neat internally (see urlscheme/macos/main.swift) since it's
// compiled once and never varies per-install. The parameter still exists
// so this function's signature matches Windows/Linux's register(), which
// do need it.
func register(_ string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	bundlePath := filepath.Join(home, ".neat", "NeatURLHandler.app")

	if err := extractBundle(bundlePath); err != nil {
		return fmt.Errorf("extract bundle: %w", err)
	}

	if err := commandRunner(lsregisterPath, "-f", bundlePath); err != nil {
		return fmt.Errorf("run lsregister: %w", err)
	}
	return nil
}

// extractBundle copies the embedded .app bundle to destPath, overwriting
// anything already there (idempotent — safe to call on every neat mcp
// startup, e.g. after a binary update changes the embedded wrapper).
func extractBundle(destPath string) error {
	return fs.WalkDir(bundleFS, bundleFSRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(bundleFSRoot, path)
		if err != nil {
			return fmt.Errorf("compute relative path for %s: %w", path, err)
		}
		target := filepath.Join(destPath, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := bundleFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded file %s: %w", path, err)
		}
		mode := os.FileMode(0o644)
		if filepath.Base(path) == "NeatURLHandler" {
			mode = 0o755
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", target, err)
		}
		return os.WriteFile(target, data, mode)
	})
}
