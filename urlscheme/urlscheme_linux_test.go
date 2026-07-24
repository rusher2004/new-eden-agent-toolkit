//go:build linux

package urlscheme_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/urlscheme"
)

func TestRegister_Linux_WritesDesktopFileAndInvokesXdgMime(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	var gotName string
	var gotArgs []string
	urlscheme.SetCommandRunnerForTest(func(name string, args ...string) error {
		gotName = name
		gotArgs = args
		return nil
	})
	t.Cleanup(urlscheme.ResetCommandRunnerForTest)

	const launcherPath = "/home/pilot/.neat/bin/neat"
	if err := urlscheme.Register(launcherPath); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	desktopFilePath := filepath.Join(tmpHome, ".local", "share", "applications", "neat-url-handler.desktop")
	data, err := os.ReadFile(desktopFilePath)
	if err != nil {
		t.Fatalf("expected .desktop file at %s, read error: %v", desktopFilePath, err)
	}
	want := urlscheme.ExportedBuildDesktopFileContent(launcherPath)
	if string(data) != want {
		t.Errorf(".desktop file content = %q, want %q", string(data), want)
	}

	if gotName != "xdg-mime" {
		t.Errorf("commandRunner name = %q, want %q", gotName, "xdg-mime")
	}
	wantArgs := []string{"default", "neat-url-handler.desktop", "x-scheme-handler/eveauth-neat"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("commandRunner args = %v, want %v", gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Errorf("commandRunner args[%d] = %q, want %q", i, gotArgs[i], wantArgs[i])
		}
	}
}
