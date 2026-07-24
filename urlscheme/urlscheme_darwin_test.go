//go:build darwin

package urlscheme_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/urlscheme"
)

func TestRegister_Darwin_WritesBundleAndInvokesLsregister(t *testing.T) {
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

	if err := urlscheme.Register(filepath.Join(tmpHome, ".neat", "bin", "neat")); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	bundlePath := filepath.Join(tmpHome, ".neat", "NeatURLHandler.app")
	execPath := filepath.Join(bundlePath, "Contents", "MacOS", "NeatURLHandler")
	plistPath := filepath.Join(bundlePath, "Contents", "Info.plist")

	if _, err := os.Stat(execPath); err != nil {
		t.Errorf("expected bundle executable at %s, stat error: %v", execPath, err)
	}
	if _, err := os.Stat(plistPath); err != nil {
		t.Errorf("expected Info.plist at %s, stat error: %v", plistPath, err)
	}

	const wantLsregisterPath = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	if gotName != wantLsregisterPath {
		t.Errorf("commandRunner name = %q, want %q (lsregister is not on $PATH by default on macOS)", gotName, wantLsregisterPath)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-f" || gotArgs[1] != bundlePath {
		t.Errorf("commandRunner args = %v, want [-f %s]", gotArgs, bundlePath)
	}
}
