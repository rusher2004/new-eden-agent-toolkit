package sde

import (
	"context"
	"testing"
)

func TestMoon_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (40009078, 14, 8, 30000142, 1.0, 2.0, 3.0, 'Jita 1 - Moon 1')`)

	got, ok, err := store.Moon(context.Background(), testUserAgent, 40009078)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false, want true")
	}
	if got.ID != 40009078 || got.SystemID != 30000142 || got.Name != "Jita 1 - Moon 1" {
		t.Errorf("got = %+v, want ID=40009078 SystemID=30000142 Name=%q", got, "Jita 1 - Moon 1")
	}
}

func TestMoon_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Moon(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if ok {
		t.Error("Moon ok = true, want false for an unknown ID")
	}
}

func TestPutMoon_ThenMoon_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Moon{ID: 999, Name: "Read-Through Moon", SystemID: 30000142}

	if err := store.PutMoon(want); err != nil {
		t.Fatalf("PutMoon returned error: %v", err)
	}
	got, ok, err := store.Moon(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Moon returned error: %v", err)
	}
	if !ok {
		t.Fatal("Moon ok = false, want true after PutMoon")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
