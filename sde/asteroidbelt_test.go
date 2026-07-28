package sde

import (
	"context"
	"testing"
)

func TestAsteroidBelt_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "asteroidBelts", 40000003,
		`{"_key": 40000003, "name": "Tanoo I - Asteroid Belt 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	got, ok, err := store.AsteroidBelt(context.Background(), "test-agent", 40000003)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false, want true")
	}
	if got.ID != 40000003 || got.Name != "Tanoo I - Asteroid Belt 1" || got.SystemID != 30000001 {
		t.Errorf("got = %+v, want ID=40000003 Name=\"Tanoo I - Asteroid Belt 1\" SystemID=30000001", got)
	}
	if got.Position != (Position{X: 1, Y: 2, Z: 3}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
}

func TestAsteroidBelt_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.AsteroidBelt(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if ok {
		t.Error("AsteroidBelt ok = true, want false for an unknown ID")
	}
}

func TestAsteroidBelt_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "asteroidBelts", 40000003,
		`{"_key": 999, "name": "Tanoo I - Asteroid Belt 1", "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	_, _, err := store.AsteroidBelt(context.Background(), "test-agent", 40000003)
	if err == nil {
		t.Fatal("AsteroidBelt returned nil error, want an error for a mismatched _key")
	}
}

func TestPutAsteroidBelt_ThenAsteroidBelt_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := AsteroidBelt{ID: 40000099, Name: "New Belt", Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099}

	if err := store.PutAsteroidBelt(want); err != nil {
		t.Fatalf("PutAsteroidBelt returned error: %v", err)
	}

	got, ok, err := store.AsteroidBelt(context.Background(), "test-agent", 40000099)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false after PutAsteroidBelt, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
