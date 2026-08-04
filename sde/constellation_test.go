package sde

import (
	"context"
	"testing"
)

func TestConstellation_KnownID_ReturnsRowWithSystemIDs(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName, x, y, z) VALUES (20000001, 10000001, 'San Matar', 1.0, 2.0, 3.0)`)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000001, 20000001, 'Tanoo')`)

	got, ok, err := store.Constellation(context.Background(), testUserAgent, 20000001)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false, want true")
	}
	if got.ID != 20000001 || got.RegionID != 10000001 || got.Name != "San Matar" {
		t.Errorf("got = %+v, want ID=20000001 RegionID=10000001 Name=San Matar", got)
	}
	if got.Position != (Position{X: 1.0, Y: 2.0, Z: 3.0}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
	if len(got.SystemIDs) != 1 || got.SystemIDs[0] != 30000001 {
		t.Errorf("SystemIDs = %v, want [30000001]", got.SystemIDs)
	}
}

func TestConstellation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Constellation(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if ok {
		t.Error("Constellation ok = true, want false for an unknown ID")
	}
}

func TestPutConstellation_ThenConstellation_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Constellation{ID: 999, RegionID: 10000001, Name: "Read-Through Constellation", SystemIDs: []int64{1}}

	if err := store.PutConstellation(want); err != nil {
		t.Fatalf("PutConstellation returned error: %v", err)
	}
	got, ok, err := store.Constellation(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false, want true after PutConstellation")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestConstellationIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000001, 10000001, 'San Matar')`)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000002, 10000001, 'Second')`)

	got, err := store.ConstellationIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("ConstellationIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 20000001 || got[1] != 20000002 {
		t.Errorf("ConstellationIDs = %v, want [20000001 20000002]", got)
	}
}
