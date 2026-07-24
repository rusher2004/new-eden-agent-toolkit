// Package urlscheme registers eveauth-neat:// as this machine's handler for
// the ESI OAuth2 redirect, one implementation per OS (see the
// urlscheme_{darwin,windows,linux}.go build-tag-separated files).
package urlscheme

import (
	"fmt"
	"os/exec"
)

// Scheme is the custom URI scheme registered as the ESI OAuth redirect
// target. ESI's app settings allow schemes starting with "eveauth".
const Scheme = "eveauth-neat"

// realCommandRunner is the actual OS-command invocation; commandRunner
// starts out pointing at it and tests substitute a fake, restoring this via
// ResetCommandRunnerForTest.
func realCommandRunner(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// commandRunner invokes an external OS command. Overridable in tests so
// registration logic can be verified without actually invoking lsregister,
// reg, or xdg-mime.
var commandRunner = realCommandRunner

// Register installs eveauth-neat:// as this machine's OS-level handler for
// the scheme, pointing at launcherPath. Idempotent: safe to call on every
// `neat mcp` startup.
func Register(launcherPath string) error {
	if err := register(launcherPath); err != nil {
		return fmt.Errorf("register %s handler: %w", Scheme, err)
	}
	return nil
}

// buildWindowsCommand returns the shell\open\command registry value for the
// given launcher path. Pure string formatting — no Windows-specific types
// — so it's testable on any OS, even though only urlscheme_windows.go
// actually writes it to the registry.
func buildWindowsCommand(launcherPath string) string {
	return fmt.Sprintf(`"%s" callback-handler "%%1"`, launcherPath)
}

// buildDesktopFileContent returns the contents of the .desktop file Linux
// uses to register a URI scheme handler. Pure string formatting — testable
// on any OS, even though only urlscheme_linux.go actually writes it.
func buildDesktopFileContent(launcherPath string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Neat URL Handler
Exec=%s callback-handler %%u
NoDisplay=true
MimeType=x-scheme-handler/%s;
`, launcherPath, Scheme)
}

// ExportedBuildWindowsCommand exposes buildWindowsCommand for external tests.
func ExportedBuildWindowsCommand(launcherPath string) string {
	return buildWindowsCommand(launcherPath)
}

// ExportedBuildDesktopFileContent exposes buildDesktopFileContent for external tests.
func ExportedBuildDesktopFileContent(launcherPath string) string {
	return buildDesktopFileContent(launcherPath)
}

// SetCommandRunnerForTest overrides the OS-command runner for tests.
func SetCommandRunnerForTest(fn func(name string, args ...string) error) {
	commandRunner = fn
}

// ResetCommandRunnerForTest restores the real OS-command runner.
func ResetCommandRunnerForTest() {
	commandRunner = realCommandRunner
}
