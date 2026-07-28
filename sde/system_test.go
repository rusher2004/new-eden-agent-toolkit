package sde

import (
	"context"
	"testing"
)

func TestSystem_KnownID_ReturnsAugmentedStationIDs(t *testing.T) {
	store := seedStore(t)
	starID := int64(40000001)
	secClass := "B"
	// Simulates what sync.go's writer produces: the raw mapSolarSystems.jsonl
	// fields plus an augmented "stationIds" key the source file never has.
	putRecord(t, store, "systems", 30000001,
		`{"_key": 30000001, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.858324, "securityClass": "B", "starID": 40000001, "stargateIDs": [50000056, 50000057], "stationIds": [60000001, 60000002]}`)

	got, ok, err := store.System(context.Background(), "test-agent", 30000001)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.ID != 30000001 || got.ConstellationID != 20000001 {
		t.Errorf("got = %+v, want ID=30000001 ConstellationID=20000001", got)
	}
	if got.Name["en"] != "Tanoo" {
		t.Errorf("Name[en] = %q, want Tanoo", got.Name["en"])
	}
	if got.Position != (Position{X: 1, Y: 2, Z: 3}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
	if got.SecurityStatus != 0.858324 {
		t.Errorf("SecurityStatus = %v, want 0.858324", got.SecurityStatus)
	}
	if got.SecurityClass == nil || *got.SecurityClass != secClass {
		t.Errorf("SecurityClass = %v, want %q", got.SecurityClass, secClass)
	}
	if got.StarID == nil || *got.StarID != starID {
		t.Errorf("StarID = %v, want %d", got.StarID, starID)
	}
	if len(got.StargateIDs) != 2 || got.StargateIDs[0] != 50000056 {
		t.Errorf("StargateIDs = %v, want [50000056 50000057]", got.StargateIDs)
	}
	if len(got.StationIDs) != 2 || got.StationIDs[0] != 60000001 || got.StationIDs[1] != 60000002 {
		t.Errorf("StationIDs = %v, want [60000001 60000002]", got.StationIDs)
	}
}

func TestSystem_KnownID_ReturnsPlanets(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000142,
		`{"_key": 30000142, "constellationID": 20000020, "name": {"en": "Jita"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.9459, "planets": [{"planetId": 40009077}, {"planetId": 40009080, "moonIds": [40009081]}, {"planetId": 40009082, "moonIds": [40009083, 40009084], "asteroidBeltIds": [40009085]}]}`)

	got, ok, err := store.System(context.Background(), "test-agent", 30000142)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if len(got.Planets) != 3 {
		t.Fatalf("Planets = %+v, want 3 entries", got.Planets)
	}
	if got.Planets[0].PlanetID != 40009077 || len(got.Planets[0].MoonIDs) != 0 || len(got.Planets[0].AsteroidBeltIDs) != 0 {
		t.Errorf("Planets[0] = %+v, want PlanetID=40009077 with no moons/belts", got.Planets[0])
	}
	if got.Planets[1].PlanetID != 40009080 || len(got.Planets[1].MoonIDs) != 1 || got.Planets[1].MoonIDs[0] != 40009081 {
		t.Errorf("Planets[1] = %+v, want PlanetID=40009080 MoonIDs=[40009081]", got.Planets[1])
	}
	if got.Planets[2].PlanetID != 40009082 || len(got.Planets[2].MoonIDs) != 2 || len(got.Planets[2].AsteroidBeltIDs) != 1 {
		t.Errorf("Planets[2] = %+v, want PlanetID=40009082 with 2 moons and 1 belt", got.Planets[2])
	}
}

func TestSystem_NoPlanets_ReturnsEmpty(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000002,
		`{"_key": 30000002, "constellationID": 20000001, "name": {"en": "Empty System"}, "position": {"x": 0, "y": 0, "z": 0}, "securityStatus": 0.5}`)

	got, ok, err := store.System(context.Background(), "test-agent", 30000002)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if len(got.Planets) != 0 {
		t.Errorf("Planets = %+v, want empty", got.Planets)
	}
}

func TestSystem_NoStationsOrStargates_ReturnsEmpty(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000002,
		`{"_key": 30000002, "constellationID": 20000001, "name": {"en": "Empty System"}, "position": {"x": 0, "y": 0, "z": 0}, "securityStatus": 0.5}`)

	got, ok, err := store.System(context.Background(), "test-agent", 30000002)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.SecurityClass != nil {
		t.Errorf("SecurityClass = %v, want nil", got.SecurityClass)
	}
	if got.StarID != nil {
		t.Errorf("StarID = %v, want nil", got.StarID)
	}
	if len(got.StargateIDs) != 0 || len(got.StationIDs) != 0 {
		t.Errorf("StargateIDs/StationIDs = %v/%v, want both empty", got.StargateIDs, got.StationIDs)
	}
}

func TestSystem_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.System(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if ok {
		t.Error("System ok = true, want false for an unknown ID")
	}
}

func TestSystem_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000001,
		`{"_key": 999, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 0, "y": 0, "z": 0}, "securityStatus": 0.5}`)

	_, _, err := store.System(context.Background(), "test-agent", 30000001)
	if err == nil {
		t.Fatal("System returned nil error, want an error for a mismatched _key")
	}
}

func TestSystemIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "systems", 30000001, `{"_key": 30000001, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.85}`)

	ids, err := store.SystemIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("SystemIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 30000001 {
		t.Errorf("SystemIDs = %v, want [30000001]", ids)
	}
}

func TestSystemIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.SystemIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("SystemIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("SystemIDs = %v, want empty", ids)
	}
}

func TestPutSystem_ThenSystem_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := System{
		ID: 30000099, ConstellationID: 20000001, Name: map[string]string{"en": "New System"}, Position: Position{X: 9, Y: 8, Z: 7}, SecurityStatus: 0.9, StationIDs: []int64{60000099},
		Planets: []SystemPlanet{{PlanetID: 40000099, MoonIDs: []int64{40000199}}},
	}

	if err := store.PutSystem(want); err != nil {
		t.Fatalf("PutSystem returned error: %v", err)
	}

	got, ok, err := store.System(context.Background(), "test-agent", 30000099)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false after PutSystem, want true")
	}
	if got.ID != want.ID || got.SecurityStatus != want.SecurityStatus {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.StationIDs) != 1 || got.StationIDs[0] != 60000099 {
		t.Errorf("StationIDs = %v, want [60000099]", got.StationIDs)
	}
	if len(got.Planets) != 1 || got.Planets[0].PlanetID != 40000099 || len(got.Planets[0].MoonIDs) != 1 || got.Planets[0].MoonIDs[0] != 40000199 {
		t.Errorf("Planets = %+v, want one entry PlanetID=40000099 MoonIDs=[40000199]", got.Planets)
	}
}
