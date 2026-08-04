package sde

import (
	"context"
	"testing"
)

func TestPlanet_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (40009077, 2016, 7, 30000142, 1.0, 2.0, 3.0, 'Jita 1')`)

	got, ok, err := store.Planet(context.Background(), testUserAgent, 40009077)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false, want true")
	}
	if got.ID != 40009077 || got.TypeID != 2016 || got.SystemID != 30000142 || got.Name != "Jita 1" {
		t.Errorf("got = %+v, want ID=40009077 TypeID=2016 SystemID=30000142 Name=%q", got, "Jita 1")
	}
	if got.Position != (Position{X: 1.0, Y: 2.0, Z: 3.0}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
}

func TestPlanet_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Planet(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if ok {
		t.Error("Planet ok = true, want false for an unknown ID")
	}
}

func TestPlanet_NullItemName_DegradesToEmptyStringInsteadOfErroring(t *testing.T) {
	// itemName is nullable in mapDenormalize's fuzzwork schema, and this
	// query is shared (mechanically, if not literally) by every celestial
	// resource (moon/asteroidBelt/star/stargate all scan the same column).
	// See PR #14 review finding #2.
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009077, 2016, 7, 30000142, NULL)`)

	got, ok, err := store.Planet(context.Background(), testUserAgent, 40009077)
	if err != nil {
		t.Fatalf("Planet returned error: %v, want nil (NULL itemName should degrade gracefully)", err)
	}
	if !ok {
		t.Fatal("Planet ok = false, want true")
	}
	if got.Name != "" {
		t.Errorf("Name = %q, want empty string", got.Name)
	}
}

func TestPlanet_WrongGroupID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009078, 14, 8, 30000142, 'Jita 1 - Moon 1')`)

	_, ok, err := store.Planet(context.Background(), testUserAgent, 40009078)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if ok {
		t.Error("Planet ok = true, want false for a moon's itemID (groupID 8, not 7)")
	}
}

func TestPutPlanet_ThenPlanet_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Planet{ID: 999, Name: "Read-Through Planet", TypeID: 2016, SystemID: 30000142}

	if err := store.PutPlanet(want); err != nil {
		t.Fatalf("PutPlanet returned error: %v", err)
	}
	got, ok, err := store.Planet(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Planet returned error: %v", err)
	}
	if !ok {
		t.Fatal("Planet ok = false, want true after PutPlanet")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
