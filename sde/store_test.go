package sde_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

func TestOpen_CreatesDatabaseFileInNestedDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "sde.db")

	store, err := sde.Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	if _, err := os.Stat(path); err != nil {
		t.Errorf("database file not created at %s: %v", path, err)
	}
}

// TestOpen_UnwritableParentDirectoryReturnsError mirrors the chmod-based
// failure-injection pattern used elsewhere in this repo (e.g.
// mcp/remove_character_internal_test.go) -- skipped on Windows, where
// os.Chmod doesn't block directory writes the same way.
func TestOpen_UnwritableParentDirectoryReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory-permission-based failure isn't portable to Windows")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	_, err := sde.Open(filepath.Join(dir, "nested", "sde.db"))
	if err == nil {
		t.Fatal("Open returned nil error, want an error for an unwritable parent directory")
	}
}

func TestOpen_CorruptExistingFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sde.db")
	if err := os.WriteFile(path, []byte("not a real bbolt file"), 0o600); err != nil {
		t.Fatalf("seed corrupt file: %v", err)
	}

	_, err := sde.Open(path)
	if err == nil {
		t.Fatal("Open returned nil error, want an error for a corrupt existing bbolt file")
	}
}

func TestClose_ReturnsNilOnHealthyStore(t *testing.T) {
	store, err := sde.Open(filepath.Join(t.TempDir(), "sde.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestDefaultPath_EndsInNeatSdeSdeDb(t *testing.T) {
	path, err := sde.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}
	want := filepath.Join(".neat", "sde", "sde.db")
	if !strings.HasSuffix(path, want) {
		t.Errorf("DefaultPath() = %q, want suffix %q", path, want)
	}
}
