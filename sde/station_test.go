package sde

import (
	"context"
	"testing"
)

func TestStation_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	owner := int64(1000002)
	raceID := int64(1)
	putRecord(t, store, "stations", 60000004,
		`{"_key": 60000004, "name": "Muvolailen X - Moon 3 - CBD Corporation Storage", "systemId": 30002780, "typeId": 1531, "position": {"x": 1, "y": 2, "z": 3}, "maxDockableShipVolume": 50000000, "officeRentalCost": 3627229, "owner": 1000002, "raceId": 1, "reprocessingEfficiency": 0.5, "reprocessingStationsTake": 0.05, "services": ["docking", "market"]}`)

	got, ok, err := store.Station(context.Background(), "test-agent", 60000004)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false, want true")
	}
	if got.ID != 60000004 || got.Name != "Muvolailen X - Moon 3 - CBD Corporation Storage" || got.SystemID != 30002780 || got.TypeID != 1531 {
		t.Errorf("got = %+v, want ID=60000004 SystemID=30002780 TypeID=1531", got)
	}
	if got.Owner == nil || *got.Owner != owner {
		t.Errorf("Owner = %v, want %d", got.Owner, owner)
	}
	if got.RaceID == nil || *got.RaceID != raceID {
		t.Errorf("RaceID = %v, want %d", got.RaceID, raceID)
	}
	if got.MaxDockableShipVolume != 50000000 || got.OfficeRentalCost != 3627229 || got.ReprocessingEfficiency != 0.5 || got.ReprocessingStationsTake != 0.05 {
		t.Errorf("got = %+v, want MaxDockableShipVolume=50000000 OfficeRentalCost=3627229 ReprocessingEfficiency=0.5 ReprocessingStationsTake=0.05", got)
	}
	if len(got.Services) != 2 || got.Services[0] != "docking" || got.Services[1] != "market" {
		t.Errorf("Services = %v, want [docking market]", got.Services)
	}
}

func TestStation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Station(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if ok {
		t.Error("Station ok = true, want false for an unknown ID")
	}
}

func TestStation_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stations", 60000004,
		`{"_key": 999, "name": "X", "systemId": 1, "typeId": 1, "position": {"x": 1, "y": 2, "z": 3}}`)

	_, _, err := store.Station(context.Background(), "test-agent", 60000004)
	if err == nil {
		t.Fatal("Station returned nil error, want an error for a mismatched _key")
	}
}

func TestPutStation_ThenStation_RoundTrips(t *testing.T) {
	store := seedStore(t)
	owner := int64(1000002)
	want := Station{
		ID: 60000099, Name: "New Station", SystemID: 30000099, TypeID: 1531, Position: Position{X: 9, Y: 8, Z: 7},
		MaxDockableShipVolume: 1000, OfficeRentalCost: 10000, Owner: &owner,
		ReprocessingEfficiency: 0.5, ReprocessingStationsTake: 0.05, Services: []string{"docking"},
	}

	if err := store.PutStation(want); err != nil {
		t.Fatalf("PutStation returned error: %v", err)
	}

	got, ok, err := store.Station(context.Background(), "test-agent", 60000099)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false after PutStation, want true")
	}
	if got.ID != want.ID || got.Name != want.Name || *got.Owner != *want.Owner || len(got.Services) != 1 {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
