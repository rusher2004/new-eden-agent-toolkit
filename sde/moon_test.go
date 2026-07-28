package sde

import (
	"context"
	"testing"
)

func TestMoon_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "moons", 40000004,
		`{"_key": 40000004, "name": "Tanoo I - Moon 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	got, ok, err := store.Moon(context.Background(), "test-agent", 40000004)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false, want true")
	}
	if got.ID != 40000004 || got.Name != "Tanoo I - Moon 1" || got.SystemID != 30000001 {
		t.Errorf("got = %+v, want ID=40000004 Name=\"Tanoo I - Moon 1\" SystemID=30000001", got)
	}
}

func TestMoon_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Moon(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if ok {
		t.Error("Moon ok = true, want false for an unknown ID")
	}
}

func TestMoon_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "moons", 40000004,
		`{"_key": 999, "name": "Tanoo I - Moon 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	_, _, err := store.Moon(context.Background(), "test-agent", 40000004)
	if err == nil {
		t.Fatal("Moon returned nil error, want an error for a mismatched _key")
	}
}

func TestPutMoon_ThenMoon_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Moon{ID: 40000199, Name: "New Moon", Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099}

	if err := store.PutMoon(want); err != nil {
		t.Fatalf("PutMoon returned error: %v", err)
	}

	got, ok, err := store.Moon(context.Background(), "test-agent", 40000199)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false after PutMoon, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
