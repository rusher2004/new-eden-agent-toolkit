package sde

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// testSchema is the subset of the fuzzwork dump's schema every sde/ accessor
// test needs -- far smaller than the real ~150-table dump, but identical
// column names/types (verified directly against the live dump) for every
// table sde/ actually queries.
const testSchema = `
CREATE TABLE invCategories (categoryID INTEGER PRIMARY KEY, categoryName VARCHAR(100), published BOOLEAN);
CREATE TABLE invGroups (groupID INTEGER PRIMARY KEY, categoryID INTEGER, groupName VARCHAR(100), published BOOLEAN);
CREATE TABLE invTypes (typeID INTEGER PRIMARY KEY, groupID INTEGER, typeName VARCHAR(100), description TEXT, published BOOLEAN, capacity FLOAT, graphicID INTEGER, iconID INTEGER, marketGroupID INTEGER, mass FLOAT, portionSize INTEGER, radius FLOAT, volume FLOAT, raceID INTEGER, packagedVolume FLOAT);
CREATE TABLE dgmTypeAttributes (typeID INTEGER, attributeID INTEGER, valueInt INTEGER, valueFloat FLOAT, PRIMARY KEY (typeID, attributeID));
CREATE TABLE dgmTypeEffects (typeID INTEGER, effectID INTEGER, isDefault BOOLEAN, PRIMARY KEY (typeID, effectID));
CREATE TABLE mapRegions (regionID INTEGER PRIMARY KEY, regionName VARCHAR(100));
CREATE TABLE mapConstellations (constellationID INTEGER PRIMARY KEY, regionID INTEGER, constellationName VARCHAR(100), x FLOAT, y FLOAT, z FLOAT);
CREATE TABLE mapSolarSystems (solarSystemID INTEGER PRIMARY KEY, constellationID INTEGER, solarSystemName VARCHAR(100), x FLOAT, y FLOAT, z FLOAT, security FLOAT, securityClass VARCHAR(2));
CREATE TABLE staStations (stationID INTEGER PRIMARY KEY, stationName VARCHAR(100), solarSystemID INTEGER, stationTypeID INTEGER, x FLOAT, y FLOAT, z FLOAT, maxShipVolumeDockable FLOAT, officeRentalCost FLOAT, corporationID INTEGER, reprocessingEfficiency FLOAT, reprocessingStationsTake FLOAT);
CREATE TABLE mapDenormalize (itemID INTEGER PRIMARY KEY, typeID INTEGER, groupID INTEGER, solarSystemID INTEGER, orbitID INTEGER, x FLOAT, y FLOAT, z FLOAT, itemName VARCHAR(100), celestialIndex INTEGER);
CREATE TABLE mapCelestialStatistics (celestialID INTEGER PRIMARY KEY, temperature FLOAT, spectralClass VARCHAR(10), luminosity FLOAT, age FLOAT, radius FLOAT);
CREATE TABLE mapJumps (stargateID INTEGER PRIMARY KEY, destinationID INTEGER);
CREATE TABLE chrFactions (factionID INTEGER PRIMARY KEY, factionName VARCHAR(100), description VARCHAR(2000), solarSystemID INTEGER, corporationID INTEGER, sizeFactor FLOAT, stationCount INTEGER, stationSystemCount INTEGER, militiaCorporationID INTEGER);
CREATE TABLE chrRaces (raceID INTEGER PRIMARY KEY, raceName VARCHAR(100), description VARCHAR(1000));
CREATE TABLE chrBloodlines (bloodlineID INTEGER PRIMARY KEY, bloodlineName VARCHAR(100), raceID INTEGER, description VARCHAR(1000), shipTypeID INTEGER, corporationID INTEGER, perception INTEGER, willpower INTEGER, charisma INTEGER, memory INTEGER, intelligence INTEGER);
CREATE TABLE chrAncestries (ancestryID INTEGER PRIMARY KEY, ancestryName VARCHAR(100), bloodlineID INTEGER, description VARCHAR(1000), iconID INTEGER, shortDescription VARCHAR(500));
CREATE TABLE eveGraphics (graphicID INTEGER PRIMARY KEY, sofFactionName VARCHAR(100), graphicFile VARCHAR(256), sofHullName VARCHAR(100), sofRaceName VARCHAR(100), iconFolder VARCHAR(256));
`

// seedStore opens a fresh temp-file sqlite Store with testSchema applied and
// marks its sync as already complete (successfully, doing nothing), so
// accessor tests can INSERT fixture rows directly without any network path
// ever running.
func seedStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "eve.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("apply test schema: %v", err)
	}
	store := &Store{dbPath: dbPath, metaPath: filepath.Join(dir, "meta.json"), db: db}
	store.syncOnce.Do(func() {})
	t.Cleanup(func() { store.Close() })
	return store
}

// seedExec runs a fixture INSERT (or other statement) against store's db,
// failing the test immediately on error.
func seedExec(t *testing.T, store *Store, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatalf("exec fixture %q: %v", query, err)
	}
}

func TestOpen_CreatesCacheDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "eve.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	if store.dbPath != path {
		t.Errorf("dbPath = %q, want %q", store.dbPath, path)
	}
	if store.metaPath != filepath.Join(dir, "nested", "meta.json") {
		t.Errorf("metaPath = %q, want sibling meta.json", store.metaPath)
	}
}

func TestStore_Close_NilStore_NoOp(t *testing.T) {
	var store *Store
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil Store returned error: %v", err)
	}
}

func TestStore_Close_NilDB_NoOp(t *testing.T) {
	store := &Store{}
	if err := store.Close(); err != nil {
		t.Errorf("Close with nil db returned error: %v", err)
	}
}

func TestMeta_RoundTrip(t *testing.T) {
	store := &Store{metaPath: filepath.Join(t.TempDir(), "meta.json")}

	want := storeMeta{SchemaVersion: "3", DumpVersion: `"abc123"`}
	if err := store.writeMeta(want); err != nil {
		t.Fatalf("writeMeta returned error: %v", err)
	}

	got, err := store.readMeta()
	if err != nil {
		t.Fatalf("readMeta returned error: %v", err)
	}
	if got != want {
		t.Errorf("readMeta = %+v, want %+v", got, want)
	}
}

func TestMeta_NeverWritten_ReturnsZeroValue(t *testing.T) {
	store := &Store{metaPath: filepath.Join(t.TempDir(), "meta.json")}

	got, err := store.readMeta()
	if err != nil {
		t.Fatalf("readMeta returned error: %v", err)
	}
	if got != (storeMeta{}) {
		t.Errorf("readMeta = %+v, want zero value", got)
	}
}

func TestOverlay_PutThenGet_ReturnsValue(t *testing.T) {
	store := &Store{}
	putOverlay(store, "type", 587, Type{ID: 587, Name: "Rifter"})

	got, ok := getOverlay[Type](store, "type", 587)
	if !ok {
		t.Fatal("getOverlay ok = false, want true")
	}
	if got.Name != "Rifter" {
		t.Errorf("got.Name = %q, want Rifter", got.Name)
	}
}

func TestOverlay_UnknownKey_ReturnsNotOK(t *testing.T) {
	store := &Store{}
	_, ok := getOverlay[Type](store, "type", 999)
	if ok {
		t.Error("getOverlay ok = true, want false for an unseeded key")
	}
}

func TestOverlay_TypeMismatch_ReturnsNotOKInsteadOfPanicking(t *testing.T) {
	// Every real (kind, T) pairing in sde/ is correct today, but the type
	// assertion inside getOverlay must degrade to a cache miss rather than
	// panic if that ever drifts -- see PR #14 review finding #4. Simulated
	// here by storing a Region under the "type" kind and reading it back
	// requesting a Type.
	store := &Store{}
	putOverlay(store, "type", 587, Region{ID: 587, Name: "Not A Type"})

	got, ok := getOverlay[Type](store, "type", 587)
	if ok {
		t.Errorf("getOverlay ok = true, want false for a type mismatch; got = %+v", got)
	}
}
