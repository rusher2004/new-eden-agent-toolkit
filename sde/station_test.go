package sde

import (
	"context"
	"testing"
)

func TestStation_KnownID_ReturnsRowWithJoinedRaceID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published, raceID) VALUES (1531, 15, 'Caldari Trading Post', 1, 1)`)
	seedExec(t, store, `INSERT INTO staStations (stationID, stationName, solarSystemID, stationTypeID, x, y, z, maxShipVolumeDockable, officeRentalCost, corporationID, reprocessingEfficiency, reprocessingStationsTake)
		VALUES (60003760, 'Jita IV - Moon 4', 30000142, 1531, 1.0, 2.0, 3.0, 500000000.0, 800000.0, 1000035, 0.5, 0.05)`)

	got, ok, err := store.Station(context.Background(), testUserAgent, 60003760)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false, want true")
	}
	if got.ID != 60003760 || got.Name != "Jita IV - Moon 4" || got.SystemID != 30000142 || got.TypeID != 1531 {
		t.Errorf("got = %+v, want ID=60003760 Name=%q SystemID=30000142 TypeID=1531", got, "Jita IV - Moon 4")
	}
	if got.Owner == nil || *got.Owner != 1000035 {
		t.Errorf("Owner = %v, want 1000035", got.Owner)
	}
	if got.RaceID == nil || *got.RaceID != 1 {
		t.Errorf("RaceID = %v, want 1 (joined from invTypes.raceID)", got.RaceID)
	}
	if got.Services != nil {
		t.Errorf("Services = %v, want nil (staOperationServices has no rows in the fuzzwork dump)", *got.Services)
	}
}

func TestStation_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Station(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if ok {
		t.Error("Station ok = true, want false for an unknown ID")
	}
}

func TestPutStation_ThenStation_ReturnsOverlaidServices(t *testing.T) {
	store := seedStore(t)
	services := []string{"market", "reprocessing-plant"}
	want := Station{ID: 999, Name: "Read-Through Station", Services: &services}

	if err := store.PutStation(want); err != nil {
		t.Fatalf("PutStation returned error: %v", err)
	}
	got, ok, err := store.Station(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Station returned error: %v", err)
	}
	if !ok {
		t.Fatal("Station ok = false, want true after PutStation")
	}
	if got.Services == nil || len(*got.Services) != 2 {
		t.Errorf("Services = %v, want [market reprocessing-plant] (overlay values keep whatever PutStation was given)", got.Services)
	}
}
