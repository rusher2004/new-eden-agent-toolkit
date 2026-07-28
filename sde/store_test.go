package sde_test

import (
	"context"
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

// TestNilStore_EveryMethodIsSafe proves every exported *sde.Store method
// can be called on a nil receiver without panicking -- required so
// commands/mcp.go can pass a nil sdeStore through on a failed sde.Open
// (e.g. bbolt's exclusive file lock already held by another neat process)
// and have every Universe tool's existing cache-miss/ESI-fallback path
// handle it exactly like a never-synced cache, rather than crashing the
// whole MCP server over an sde/-specific failure.
func TestNilStore_EveryMethodIsSafe(t *testing.T) {
	var store *sde.Store
	ctx := context.Background()
	const userAgent = "neat-test/0.1"

	if got, ok, err := store.Category(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Category on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutCategory(sde.Category{ID: 1}); err != nil {
		t.Errorf("PutCategory on nil store returned error: %v, want nil (silent no-op)", err)
	}
	if ids, err := store.CategoryIDs(ctx, userAgent); ids != nil || err != nil {
		t.Errorf("CategoryIDs on nil store = %v, %v, want nil/nil", ids, err)
	}

	if got, ok, err := store.Group(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Group on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutGroup(sde.Group{ID: 1}); err != nil {
		t.Errorf("PutGroup on nil store returned error: %v, want nil", err)
	}
	if ids, err := store.GroupIDs(ctx, userAgent); ids != nil || err != nil {
		t.Errorf("GroupIDs on nil store = %v, %v, want nil/nil", ids, err)
	}

	if got, ok, err := store.Type(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Type on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutType(sde.Type{ID: 1}); err != nil {
		t.Errorf("PutType on nil store returned error: %v, want nil", err)
	}
	if ids, err := store.TypeIDs(ctx, userAgent); ids != nil || err != nil {
		t.Errorf("TypeIDs on nil store = %v, %v, want nil/nil", ids, err)
	}

	if got, ok, err := store.Region(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Region on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutRegion(sde.Region{ID: 1}); err != nil {
		t.Errorf("PutRegion on nil store returned error: %v, want nil", err)
	}
	if ids, err := store.RegionIDs(ctx, userAgent); ids != nil || err != nil {
		t.Errorf("RegionIDs on nil store = %v, %v, want nil/nil", ids, err)
	}

	if got, ok, err := store.Constellation(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Constellation on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutConstellation(sde.Constellation{ID: 1}); err != nil {
		t.Errorf("PutConstellation on nil store returned error: %v, want nil", err)
	}
	if ids, err := store.ConstellationIDs(ctx, userAgent); ids != nil || err != nil {
		t.Errorf("ConstellationIDs on nil store = %v, %v, want nil/nil", ids, err)
	}

	if got, ok, err := store.System(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("System on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutSystem(sde.System{ID: 1}); err != nil {
		t.Errorf("PutSystem on nil store returned error: %v, want nil", err)
	}
	if ids, err := store.SystemIDs(ctx, userAgent); ids != nil || err != nil {
		t.Errorf("SystemIDs on nil store = %v, %v, want nil/nil", ids, err)
	}

	if got, ok, err := store.Graphic(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Graphic on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutGraphic(sde.Graphic{ID: 1}); err != nil {
		t.Errorf("PutGraphic on nil store returned error: %v, want nil", err)
	}
	if ids, err := store.GraphicIDs(ctx, userAgent); ids != nil || err != nil {
		t.Errorf("GraphicIDs on nil store = %v, %v, want nil/nil", ids, err)
	}

	if got, ok, err := store.AsteroidBelt(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("AsteroidBelt on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutAsteroidBelt(sde.AsteroidBelt{ID: 1}); err != nil {
		t.Errorf("PutAsteroidBelt on nil store returned error: %v, want nil", err)
	}

	if got, ok, err := store.Moon(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Moon on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutMoon(sde.Moon{ID: 1}); err != nil {
		t.Errorf("PutMoon on nil store returned error: %v, want nil", err)
	}

	if got, ok, err := store.Planet(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Planet on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutPlanet(sde.Planet{ID: 1}); err != nil {
		t.Errorf("PutPlanet on nil store returned error: %v, want nil", err)
	}

	if got, ok, err := store.Stargate(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Stargate on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutStargate(sde.Stargate{ID: 1}); err != nil {
		t.Errorf("PutStargate on nil store returned error: %v, want nil", err)
	}

	if got, ok, err := store.Star(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Star on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutStar(sde.Star{ID: 1}); err != nil {
		t.Errorf("PutStar on nil store returned error: %v, want nil", err)
	}

	if got, ok, err := store.Station(ctx, userAgent, 1); ok || err != nil || got.ID != 0 {
		t.Errorf("Station on nil store = %+v, %v, %v, want zero value/false/nil", got, ok, err)
	}
	if err := store.PutStation(sde.Station{ID: 1}); err != nil {
		t.Errorf("PutStation on nil store returned error: %v, want nil", err)
	}

	if got, err := store.Ancestries(ctx, userAgent); got != nil || err != nil {
		t.Errorf("Ancestries on nil store = %v, %v, want nil/nil", got, err)
	}
	if got, err := store.Bloodlines(ctx, userAgent); got != nil || err != nil {
		t.Errorf("Bloodlines on nil store = %v, %v, want nil/nil", got, err)
	}
	if got, err := store.Races(ctx, userAgent); got != nil || err != nil {
		t.Errorf("Races on nil store = %v, %v, want nil/nil", got, err)
	}
	if got, err := store.Factions(ctx, userAgent); got != nil || err != nil {
		t.Errorf("Factions on nil store = %v, %v, want nil/nil", got, err)
	}

	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store returned error: %v, want nil", err)
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
