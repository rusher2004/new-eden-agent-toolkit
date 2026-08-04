package sde

import (
	"context"
	"testing"
)

func TestAsteroidBelt_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (40009079, 16, 9, 30000142, 1.0, 2.0, 3.0, 'Jita 1 - Asteroid Belt 1')`)

	got, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40009079)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false, want true")
	}
	if got.ID != 40009079 || got.SystemID != 30000142 || got.Name != "Jita 1 - Asteroid Belt 1" {
		t.Errorf("got = %+v, want ID=40009079 SystemID=30000142 Name=%q", got, "Jita 1 - Asteroid Belt 1")
	}
}

func TestAsteroidBelt_GroupID4430_AlsoMatches(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009999, 16, 4430, 30000142, 'Wormhole Belt')`)

	_, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40009999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Error("AsteroidBelt ok = false, want true for groupID 4430 (the wormhole-system asteroid belt group)")
	}
}

func TestAsteroidBelt_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if ok {
		t.Error("AsteroidBelt ok = true, want false for an unknown ID")
	}
}

func TestPutAsteroidBelt_ThenAsteroidBelt_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := AsteroidBelt{ID: 999, Name: "Read-Through Belt", SystemID: 30000142}

	if err := store.PutAsteroidBelt(want); err != nil {
		t.Fatalf("PutAsteroidBelt returned error: %v", err)
	}
	got, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if !ok {
		t.Fatal("AsteroidBelt ok = false, want true after PutAsteroidBelt")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
