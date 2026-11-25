// Package sdetest builds small SDE databases for tests in other packages,
// the way net/http/httptest builds servers. The schema covers the tables
// sde/'s industry, dogma and routing accessors read, with the fuzzwork dump's
// own column names.
package sdetest

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// Schema is the subset of the fuzzwork dump's schema that sde's industry,
// dogma and routing accessors need.
const Schema = `
CREATE TABLE invCategories (categoryID INTEGER PRIMARY KEY, categoryName VARCHAR(100), published BOOLEAN);
CREATE TABLE invGroups (groupID INTEGER PRIMARY KEY, categoryID INTEGER, groupName VARCHAR(100), published BOOLEAN);
CREATE TABLE invTypes (typeID INTEGER PRIMARY KEY, groupID INTEGER, typeName VARCHAR(100), description TEXT, published BOOLEAN, capacity FLOAT, graphicID INTEGER, iconID INTEGER, marketGroupID INTEGER, mass FLOAT, portionSize INTEGER, radius FLOAT, volume FLOAT, raceID INTEGER, packagedVolume FLOAT);
CREATE TABLE dgmTypeAttributes (typeID INTEGER, attributeID INTEGER, valueInt INTEGER, valueFloat FLOAT, PRIMARY KEY (typeID, attributeID));
CREATE TABLE dgmTypeEffects (typeID INTEGER, effectID INTEGER, isDefault BOOLEAN, PRIMARY KEY (typeID, effectID));
CREATE TABLE dgmEffects (effectID INTEGER PRIMARY KEY, effectName VARCHAR(400), effectCategory INTEGER, modifierInfo TEXT);
CREATE TABLE dgmAttributeTypes (attributeID INTEGER PRIMARY KEY, attributeName VARCHAR(100), defaultValue FLOAT, stackable BOOLEAN, highIsGood BOOLEAN);
CREATE TABLE mapRegions (regionID INTEGER PRIMARY KEY, regionName VARCHAR(100));
CREATE TABLE mapConstellations (constellationID INTEGER PRIMARY KEY, regionID INTEGER, constellationName VARCHAR(100), x FLOAT, y FLOAT, z FLOAT);
CREATE TABLE mapSolarSystems (solarSystemID INTEGER PRIMARY KEY, constellationID INTEGER, solarSystemName VARCHAR(100), x FLOAT, y FLOAT, z FLOAT, security FLOAT, securityClass VARCHAR(2));
CREATE TABLE mapSolarSystemJumps (fromRegionID INTEGER, fromConstellationID INTEGER, fromSolarSystemID INTEGER, toSolarSystemID INTEGER, toConstellationID INTEGER, toRegionID INTEGER);
CREATE TABLE staStations (stationID INTEGER PRIMARY KEY, stationName VARCHAR(100), solarSystemID INTEGER, stationTypeID INTEGER, x FLOAT, y FLOAT, z FLOAT, maxShipVolumeDockable FLOAT, officeRentalCost FLOAT, corporationID INTEGER, operationID INTEGER, reprocessingEfficiency FLOAT, reprocessingStationsTake FLOAT);
CREATE TABLE industryActivity (typeID INTEGER, activityID INTEGER, time INTEGER);
CREATE TABLE industryActivityMaterials (typeID INTEGER, activityID INTEGER, materialTypeID INTEGER, quantity INTEGER);
CREATE TABLE industryActivityProducts (typeID INTEGER, activityID INTEGER, productTypeID INTEGER, quantity INTEGER);
CREATE TABLE industryActivitySkills (typeID INTEGER, activityID INTEGER, skillID INTEGER, level INTEGER);
CREATE TABLE industryActivityProbabilities (typeID INTEGER, activityID INTEGER, productTypeID INTEGER, probability DECIMAL(3, 2));
CREATE TABLE invTypeMaterials (typeID INTEGER, materialTypeID INTEGER, quantity INTEGER);
CREATE TABLE indModifierSources (typeID INTEGER, activityName VARCHAR(100), modifierType VARCHAR(100), dogmaAttributeID INTEGER, filterID INTEGER);
CREATE TABLE indTargetFilters (targetFilterID INTEGER PRIMARY KEY, name VARCHAR(100));
CREATE TABLE indTargetFilterCategories (targetFilterID INTEGER, categoryID INTEGER);
CREATE TABLE indTargetFilterGroups (targetFilterID INTEGER, groupID INTEGER);
`

// NewStore creates a temp-dir SDE database with Schema, runs each fixture
// SQL file (one statement per line; lines starting with -- are skipped), and
// opens it with sde.OpenStatic so nothing ever touches the network.
func NewStore(t testing.TB, fixtureFiles ...string) *sde.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "eve.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sdetest: open: %v", err)
	}
	// A throwaway file: no journal, no fsync, and every statement in one
	// transaction. Committing each line on its own flushed the file to disk
	// hundreds of times per fixture, which on Windows runners (FlushFileBuffers)
	// took the mcp package's tests past go test's 10-minute timeout.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode = OFF; PRAGMA synchronous = OFF;"); err != nil {
		t.Fatalf("sdetest: pragmas: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("sdetest: begin: %v", err)
	}
	if _, err := tx.Exec(Schema); err != nil {
		t.Fatalf("sdetest: schema: %v", err)
	}
	for _, f := range fixtureFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("sdetest: read %s: %v", f, err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "--") {
				continue
			}
			if _, err := tx.Exec(line); err != nil {
				t.Fatalf("sdetest: %s: %v\n%s", f, err, line)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("sdetest: commit: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("sdetest: close: %v", err)
	}
	store, err := sde.OpenStatic(path)
	if err != nil {
		t.Fatalf("sdetest: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}
