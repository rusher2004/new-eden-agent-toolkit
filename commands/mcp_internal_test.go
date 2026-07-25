package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureLauncher_CopiesWhenMissing(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	fakeBinary := filepath.Join(t.TempDir(), "neat-fake")
	if err := os.WriteFile(fakeBinary, []byte("fake binary contents v1"), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	orig := currentExecutable
	t.Cleanup(func() { currentExecutable = orig })
	currentExecutable = func() (string, error) { return fakeBinary, nil }

	target, err := ensureLauncher()
	if err != nil {
		t.Fatalf("ensureLauncher: %v", err)
	}
	wantName := "neat"
	if runtime.GOOS == "windows" {
		wantName = "neat.exe"
	}
	wantTarget := filepath.Join(tmpHome, ".neat", "bin", wantName)
	if target != wantTarget {
		t.Errorf("target = %q, want %q", target, wantTarget)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "fake binary contents v1" {
		t.Errorf("target contents = %q, want copied fake binary contents", got)
	}
}

func TestEnsureLauncher_NoOpWhenAlreadyCurrent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	fakeBinary := filepath.Join(t.TempDir(), "neat-fake")
	if err := os.WriteFile(fakeBinary, []byte("v1"), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	orig := currentExecutable
	t.Cleanup(func() { currentExecutable = orig })
	currentExecutable = func() (string, error) { return fakeBinary, nil }
	target, err := ensureLauncher()
	if err != nil {
		t.Fatalf("first ensureLauncher: %v", err)
	}

	// Simulate running from the installed location: current == target.
	currentExecutable = func() (string, error) { return target, nil }
	target2, err := ensureLauncher()
	if err != nil {
		t.Fatalf("second ensureLauncher: %v", err)
	}
	if target2 != target {
		t.Errorf("second target = %q, want %q", target2, target)
	}
}

func TestEnsureLauncher_OverwritesWhenStale(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	fakeBinary := filepath.Join(t.TempDir(), "neat-fake")
	if err := os.WriteFile(fakeBinary, []byte("v1"), 0o700); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	orig := currentExecutable
	t.Cleanup(func() { currentExecutable = orig })
	currentExecutable = func() (string, error) { return fakeBinary, nil }
	target, err := ensureLauncher()
	if err != nil {
		t.Fatalf("first ensureLauncher: %v", err)
	}

	if err := os.WriteFile(fakeBinary, []byte("v2"), 0o700); err != nil {
		t.Fatalf("rewrite fake binary: %v", err)
	}
	if _, err := ensureLauncher(); err != nil {
		t.Fatalf("second ensureLauncher: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("target contents = %q, want %q", got, "v2")
	}
}
