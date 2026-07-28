package sde

import (
	"context"
	"testing"
)

func TestPlanet_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "planets", 40000002,
		`{"_key": 40000002, "name": "Tanoo I", "typeId": 11, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	got, ok, err := store.Planet(context.Background(), "test-agent", 40000002)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false, want true")
	}
	if got.ID != 40000002 || got.Name != "Tanoo I" || got.TypeID != 11 || got.SystemID != 30000001 {
		t.Errorf("got = %+v, want ID=40000002 Name=\"Tanoo I\" TypeID=11 SystemID=30000001", got)
	}
}

func TestPlanet_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Planet(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if ok {
		t.Error("Planet ok = true, want false for an unknown ID")
	}
}

func TestPlanet_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "planets", 40000002,
		`{"_key": 999, "name": "Tanoo I", "typeId": 11, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000001}`)

	_, _, err := store.Planet(context.Background(), "test-agent", 40000002)
	if err == nil {
		t.Fatal("Planet returned nil error, want an error for a mismatched _key")
	}
}

func TestPutPlanet_ThenPlanet_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Planet{ID: 40000299, Name: "New Planet", TypeID: 12, Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099}

	if err := store.PutPlanet(want); err != nil {
		t.Fatalf("PutPlanet returned error: %v", err)
	}

	got, ok, err := store.Planet(context.Background(), "test-agent", 40000299)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false after PutPlanet, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
