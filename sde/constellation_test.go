package sde

import (
	"context"
	"testing"
)

func TestConstellation_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "constellations", 20000001,
		`{"_key": 20000001, "name": {"en": "San Matar", "de": "San Matar"}, "position": {"x": 1.5, "y": 2.5, "z": 3.5}, "regionID": 10000001, "solarSystemIDs": [30000001, 30000002]}`)

	got, ok, err := store.Constellation(context.Background(), "test-agent", 20000001)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false, want true")
	}
	if got.ID != 20000001 || got.RegionID != 10000001 {
		t.Errorf("got = %+v, want ID=20000001 RegionID=10000001", got)
	}
	if got.Name["en"] != "San Matar" {
		t.Errorf("Name[en] = %q, want San Matar", got.Name["en"])
	}
	if got.Position.X != 1.5 || got.Position.Y != 2.5 || got.Position.Z != 3.5 {
		t.Errorf("Position = %+v, want {1.5 2.5 3.5}", got.Position)
	}
	if len(got.SystemIDs) != 2 || got.SystemIDs[0] != 30000001 || got.SystemIDs[1] != 30000002 {
		t.Errorf("SystemIDs = %v, want [30000001 30000002]", got.SystemIDs)
	}
}

func TestConstellation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Constellation(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if ok {
		t.Error("Constellation ok = true, want false for an unknown ID")
	}
}

func TestConstellation_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "constellations", 20000001,
		`{"_key": 999, "name": {"en": "San Matar"}, "position": {"x": 0, "y": 0, "z": 0}, "regionID": 10000001, "solarSystemIDs": []}`)

	_, _, err := store.Constellation(context.Background(), "test-agent", 20000001)
	if err == nil {
		t.Fatal("Constellation returned nil error, want an error for a mismatched _key")
	}
}

func TestConstellationIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "constellations", 20000001, `{"_key": 20000001, "name": {"en": "San Matar"}, "position": {"x": 0, "y": 0, "z": 0}, "regionID": 10000001, "solarSystemIDs": []}`)

	ids, err := store.ConstellationIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("ConstellationIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 20000001 {
		t.Errorf("ConstellationIDs = %v, want [20000001]", ids)
	}
}

func TestConstellationIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.ConstellationIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("ConstellationIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("ConstellationIDs = %v, want empty", ids)
	}
}

func TestPutConstellation_ThenConstellation_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Constellation{ID: 20000099, RegionID: 10000001, Name: map[string]string{"en": "New Constellation"}, Position: Position{X: 1, Y: 2, Z: 3}, SystemIDs: []int64{30000099}}

	if err := store.PutConstellation(want); err != nil {
		t.Fatalf("PutConstellation returned error: %v", err)
	}

	got, ok, err := store.Constellation(context.Background(), "test-agent", 20000099)
	if err != nil {
		t.Fatalf("Constellation returned error: %v", err)
	}
	if !ok {
		t.Fatal("Constellation ok = false after PutConstellation, want true")
	}
	if got.ID != want.ID || got.RegionID != want.RegionID || got.Position != want.Position {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
