package sde

import (
	"context"
	"testing"
)

func TestSystem_KnownID_ReturnsRowWithReverseRelations(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName, x, y, z, security, securityClass) VALUES (30000142, 20000020, 'Jita', 1.0, 2.0, 3.0, 0.9, 'B')`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009076, 3796, 6, 30000142, NULL, 'Jita - Sun', 0)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009077, 2016, 7, 30000142, 40009076, 'Jita 1', 1)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009078, 14, 8, 30000142, 40009077, 'Jita 1 - Moon 1', NULL)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (40009079, 16, 9, 30000142, 40009077, 'Jita 1 - Asteroid Belt 1', NULL)`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, orbitID, itemName, celestialIndex) VALUES (50008077, 16, 10, 30000142, NULL, 'Jita - Perimeter', NULL)`)
	seedExec(t, store, `INSERT INTO staStations (stationID, stationName, solarSystemID, stationTypeID) VALUES (60003760, 'Jita IV - Moon 4', 30000142, 1531)`)

	got, ok, err := store.System(context.Background(), testUserAgent, 30000142)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.ID != 30000142 || got.ConstellationID != 20000020 || got.Name != "Jita" {
		t.Errorf("got = %+v, want ID=30000142 ConstellationID=20000020 Name=Jita", got)
	}
	if got.SecurityStatus != 0.9 || got.SecurityClass == nil || *got.SecurityClass != "B" {
		t.Errorf("SecurityStatus/SecurityClass = %v/%v, want 0.9/B", got.SecurityStatus, got.SecurityClass)
	}
	if got.StarID == nil || *got.StarID != 40009076 {
		t.Errorf("StarID = %v, want 40009076", got.StarID)
	}
	if len(got.StargateIDs) != 1 || got.StargateIDs[0] != 50008077 {
		t.Errorf("StargateIDs = %v, want [50008077]", got.StargateIDs)
	}
	if len(got.StationIDs) != 1 || got.StationIDs[0] != 60003760 {
		t.Errorf("StationIDs = %v, want [60003760]", got.StationIDs)
	}
	if len(got.Planets) != 1 {
		t.Fatalf("len(Planets) = %d, want 1", len(got.Planets))
	}
	p := got.Planets[0]
	if p.PlanetID != 40009077 {
		t.Errorf("Planets[0].PlanetID = %d, want 40009077", p.PlanetID)
	}
	if len(p.MoonIDs) != 1 || p.MoonIDs[0] != 40009078 {
		t.Errorf("Planets[0].MoonIDs = %v, want [40009078]", p.MoonIDs)
	}
	if len(p.AsteroidBeltIDs) != 1 || p.AsteroidBeltIDs[0] != 40009079 {
		t.Errorf("Planets[0].AsteroidBeltIDs = %v, want [40009079]", p.AsteroidBeltIDs)
	}
}

func TestSystem_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.System(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if ok {
		t.Error("System ok = true, want false for an unknown ID")
	}
}

func TestSystem_NoStar_ReturnsNilStarID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000001, 20000001, 'Tanoo')`)

	got, ok, err := store.System(context.Background(), testUserAgent, 30000001)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true")
	}
	if got.StarID != nil {
		t.Errorf("StarID = %v, want nil", *got.StarID)
	}
}

func TestPutSystem_ThenSystem_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := System{ID: 999, ConstellationID: 20000001, Name: "Read-Through System"}

	if err := store.PutSystem(want); err != nil {
		t.Fatalf("PutSystem returned error: %v", err)
	}
	got, ok, err := store.System(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("System returned error: %v", err)
	}
	if !ok {
		t.Fatal("System ok = false, want true after PutSystem")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestSystemIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000001, 20000001, 'Tanoo')`)
	seedExec(t, store, `INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName) VALUES (30000142, 20000020, 'Jita')`)

	got, err := store.SystemIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("SystemIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 30000001 || got[1] != 30000142 {
		t.Errorf("SystemIDs = %v, want [30000001 30000142]", got)
	}
}
