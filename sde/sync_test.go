package sde

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ulikunitz/xz"
	"go.etcd.io/bbolt"
)

const testUserAgent = "neat-test/0.1"

// fullFixtureFiles returns the complete 13-file source set sync() now
// requires, with enough cross-linked data (group 25 belongs to category 6
// and contains type 587; station 60000001 is in system 30000001, owned by
// corp 1000035, which belongs to faction 500001) to exercise every
// augmentation this task adds: Group.TypeIDs, Category.GroupIDs,
// System.StationIDs, Faction.StationCount/StationSystemCount.
func fullFixtureFiles(shipCategoryName string) map[string][]string {
	return map[string][]string{
		"categories.jsonl": {
			`{"_key": 6, "name": {"en": "` + shipCategoryName + `"}, "published": true}`,
		},
		"groups.jsonl": {
			`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true}`,
		},
		"types.jsonl": {
			`{"_key": 587, "groupID": 25, "name": {"en": "Rifter"}, "published": true}`,
		},
		"typeDogma.jsonl": {
			`{"_key": 587, "dogmaAttributes": [{"attributeID": 182, "value": 3386.0}], "dogmaEffects": [{"effectID": 4, "isDefault": true}]}`,
		},
		"npcCorporations.jsonl": {
			`{"_key": 1000035, "factionID": 500001}`,
		},
		"npcStations.jsonl": {
			`{"_key": 60000001, "solarSystemID": 30000001, "ownerID": 1000035}`,
		},
		"mapSolarSystems.jsonl": {
			`{"_key": 30000001, "constellationID": 20000001, "name": {"en": "Tanoo"}, "position": {"x": 1, "y": 2, "z": 3}, "securityStatus": 0.85}`,
		},
		"factions.jsonl": {
			`{"_key": 500001, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "uniqueName": true}`,
		},
		"mapRegions.jsonl": {
			`{"_key": 10000001, "constellationIDs": [20000001], "name": {"en": "Derelik"}}`,
		},
		"mapConstellations.jsonl": {
			`{"_key": 20000001, "name": {"en": "San Matar"}, "position": {"x": 0, "y": 0, "z": 0}, "regionID": 10000001, "solarSystemIDs": [30000001]}`,
		},
		"ancestries.jsonl": {
			`{"_key": 1, "bloodlineID": 5, "description": {"en": "..."}, "name": {"en": "Liberal Holders"}}`,
		},
		"bloodlines.jsonl": {
			`{"_key": 1, "charisma": 6, "corporationID": 1000006, "description": {"en": "..."}, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "willpower": 5}`,
		},
		"races.jsonl": {
			`{"_key": 1, "description": {"en": "..."}, "name": {"en": "Caldari"}}`,
		},
		"graphics.jsonl": {
			`{"_key": 10, "graphicFile": "res:/x"}`,
		},
	}
}

func buildFixtureZip(t *testing.T) []byte {
	t.Helper()
	return buildZip(t, fullFixtureFiles("Ship"))
}

func buildFixtureZipWithRenamedShipCategory(t *testing.T) []byte {
	t.Helper()
	return buildZip(t, fullFixtureFiles("Ship Hull"))
}

func buildZip(t *testing.T, files map[string][]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, lines := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		for _, line := range lines {
			if _, err := w.Write([]byte(line + "\n")); err != nil {
				t.Fatalf("write zip entry %s: %v", name, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

// buildTarXZ builds an in-memory .tar.xz archive from name -> content,
// used to fixture both reference-data and esi-scrape in tests without any
// network dependency.
func buildTarXZ(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0o644}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content %s: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	var xzBuf bytes.Buffer
	xw, err := xz.NewWriter(&xzBuf)
	if err != nil {
		t.Fatalf("create xz writer: %v", err)
	}
	if _, err := xw.Write(tarBuf.Bytes()); err != nil {
		t.Fatalf("write xz content: %v", err)
	}
	if err := xw.Close(); err != nil {
		t.Fatalf("close xz writer: %v", err)
	}
	return xzBuf.Bytes()
}

// fixtureReferenceDataTarXZ builds a minimal reference-data archive with
// just enough of types.json to exercise the PackagedVolume join for type
// 587 (Rifter, matching fullFixtureFiles's types.jsonl entry).
func fixtureReferenceDataTarXZ(t *testing.T) []byte {
	t.Helper()
	return buildTarXZ(t, map[string]string{
		"types.json": `{"587": {"packaged_volume": 2500}}`,
	})
}

// fixtureEsiScrapeTarXZ builds a minimal esi-scrape archive with just
// enough of each of the 9 needed members to exercise every augmentation
// this task adds, cross-linked with fullFixtureFiles's IDs where it
// matters (system 30000001, race 1).
func fixtureEsiScrapeTarXZ(t *testing.T) []byte {
	t.Helper()
	const prefix = "eve-ref-esi-scrape/data/tranquility/universe/"
	return buildTarXZ(t, map[string]string{
		prefix + "asteroid_belts.en-us.yaml": `"40000003":
  name: "Tanoo I - Asteroid Belt 1"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
`,
		prefix + "moons.en-us.yaml": `"40000004":
  name: "Tanoo I - Moon 1"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
`,
		prefix + "planets.en-us.yaml": `"40000002":
  name: "Tanoo I"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
  type_id: 11
`,
		prefix + "stargates.en-us.yaml": `"50000001":
  destination: {stargate_id: 50000482, system_id: 30000778}
  name: "Stargate (G-EURJ)"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
  type_id: 29633
`,
		prefix + "stars.en-us.yaml": `"40000001":
  age: 14262808228
  luminosity: 0.01575
  name: "Tanoo - Star"
  radius: 63350000
  solar_system_id: 30000001
  spectral_class: "K2 V"
  temperature: 4567
  type_id: 45041
`,
		prefix + "stations.en-us.yaml": `"60000001":
  max_dockable_ship_volume: 50000000
  name: "Test Station"
  office_rental_cost: 3627229
  owner: 1000035
  position: {x: 1, "y": 2, z: 3}
  race_id: 1
  reprocessing_efficiency: 0.5
  reprocessing_stations_take: 0.05
  services: ["docking"]
  system_id: 30000001
  type_id: 1531
`,
		prefix + "races.en-us.yaml": `- alliance_id: 500001
  race_id: 1
`,
		prefix + "systems.en-us.yaml": `"30000001":
  planets:
    - planet_id: 40000002
      moons: [40000004]
      asteroid_belts: [40000003]
`,
		prefix + "graphics.yaml": `"10":
  sof_dna: "cf1_t1:caldaribase:caldari"
`,
	})
}

// fixtureServer serves an index.json (pointing back at its own /sde.zip)
// and the zip body itself, counting hits to each so tests can assert
// exactly how many times each endpoint was actually called. It also serves
// fixture reference-data/esi-scrape archives at fixed paths, since those
// two sources are fetched from hardcoded URLs rather than discovered via
// an index.json.
type fixtureServer struct {
	*httptest.Server
	indexHits         atomic.Int32
	zipHits           atomic.Int32
	referenceDataHits atomic.Int32
	esiScrapeHits     atomic.Int32
	etag              string
	referenceDataEtag string
	esiScrapeEtag     string
	zipBody           []byte
	referenceDataBody []byte
	esiScrapeBody     []byte
}

func newFixtureServer(t *testing.T, etag string, zipBody []byte) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{
		etag: etag, zipBody: zipBody,
		referenceDataEtag: "reference-data-etag-v1",
		esiScrapeEtag:     "esi-scrape-etag-v1",
		referenceDataBody: fixtureReferenceDataTarXZ(t),
		esiScrapeBody:     fixtureEsiScrapeTarXZ(t),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		fs.indexHits.Add(1)
		idx := map[string]any{
			"files": []map[string]string{
				{"name": "eve-online-static-data-latest-jsonl.zip", "url": fs.Server.URL + "/sde.zip", "etag": fs.etag},
			},
		}
		_ = json.NewEncoder(w).Encode(idx)
	})
	mux.HandleFunc("/sde.zip", func(w http.ResponseWriter, r *http.Request) {
		fs.zipHits.Add(1)
		_, _ = w.Write(fs.zipBody)
	})
	mux.HandleFunc("/reference-data.tar.xz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", fs.referenceDataEtag)
		if r.Method == http.MethodHead {
			return
		}
		fs.referenceDataHits.Add(1)
		_, _ = w.Write(fs.referenceDataBody)
	})
	mux.HandleFunc("/esi-scrape.tar.xz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", fs.esiScrapeEtag)
		if r.Method == http.MethodHead {
			return
		}
		fs.esiScrapeHits.Add(1)
		_, _ = w.Write(fs.esiScrapeBody)
	})
	fs.Server = httptest.NewServer(mux)
	t.Cleanup(fs.Close)
	return fs
}

func openWithIndexURL(t *testing.T, indexURL string) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "sde.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	store.indexURL = indexURL
	return store
}

// openWithFixtureServer is like openWithIndexURL, but also redirects
// referenceDataURL/esiScrapeURL to the same fixture server -- every sync
// test that needs the full three-source pipeline uses this instead.
func openWithFixtureServer(t *testing.T, srv *fixtureServer) *Store {
	t.Helper()
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")
	store.referenceDataURL = srv.Server.URL + "/reference-data.tar.xz"
	store.esiScrapeURL = srv.Server.URL + "/esi-scrape.tar.xz"
	return store
}

func openAtPathWithIndexURL(t *testing.T, path, indexURL string) *Store {
	t.Helper()
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	store.indexURL = indexURL
	return store
}

// openAtPathWithFixtureServer is the openAtPathWithIndexURL analogue of
// openWithFixtureServer, for tests that reopen the same db path as a
// second Store.
func openAtPathWithFixtureServer(t *testing.T, path string, srv *fixtureServer) *Store {
	t.Helper()
	store := openAtPathWithIndexURL(t, path, srv.Server.URL+"/index.json")
	store.referenceDataURL = srv.Server.URL + "/reference-data.tar.xz"
	store.esiScrapeURL = srv.Server.URL + "/esi-scrape.tar.xz"
	return store
}

func assertRecord(t *testing.T, store *Store, bucket string, id int64, wantEnglishName string, wantPublished bool) {
	t.Helper()
	raw := getRawRecord(t, store, bucket, id)
	if raw == nil {
		t.Fatalf("no cached record for %s[%d]", bucket, id)
	}
	var decoded struct {
		Name      map[string]string `json:"name"`
		Published bool              `json:"published"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode cached %s record %d: %v", bucket, id, err)
	}
	if decoded.Name["en"] != wantEnglishName {
		t.Errorf("%s[%d].Name[en] = %q, want %q", bucket, id, decoded.Name["en"], wantEnglishName)
	}
	if decoded.Published != wantPublished {
		t.Errorf("%s[%d].Published = %v, want %v", bucket, id, decoded.Published, wantPublished)
	}
}

func getRawRecord(t *testing.T, store *Store, bucket string, id int64) []byte {
	t.Helper()
	var raw []byte
	err := store.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		if v := b.Get(int64Key(id)); v != nil {
			raw = append([]byte(nil), v...)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("read %s[%d]: %v", bucket, id, err)
	}
	return raw
}

func bucketExists(t *testing.T, store *Store, bucket string) bool {
	t.Helper()
	var exists bool
	err := store.db.View(func(tx *bbolt.Tx) error {
		exists = tx.Bucket([]byte(bucket)) != nil
		return nil
	})
	if err != nil {
		t.Fatalf("check bucket %s: %v", bucket, err)
	}
	return exists
}

func TestSync_FirstSyncLoadsAllResourcesWithAugmentation(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v", err)
	}

	assertRecord(t, store, "categories", 6, "Ship", true)
	assertRecord(t, store, "groups", 25, "Frigate", true)
	assertRecord(t, store, "types", 587, "Rifter", true)

	group, ok, err := store.Group(context.Background(), testUserAgent, 25)
	if err != nil || !ok {
		t.Fatalf("Group(25) = %+v, %v, %v", group, ok, err)
	}
	if len(group.TypeIDs) != 1 || group.TypeIDs[0] != 587 {
		t.Errorf("Group(25).TypeIDs = %v, want [587] (reverse index from types.jsonl)", group.TypeIDs)
	}

	category, ok, err := store.Category(context.Background(), testUserAgent, 6)
	if err != nil || !ok {
		t.Fatalf("Category(6) = %+v, %v, %v", category, ok, err)
	}
	if len(category.GroupIDs) != 1 || category.GroupIDs[0] != 25 {
		t.Errorf("Category(6).GroupIDs = %v, want [25] (reverse index from groups.jsonl)", category.GroupIDs)
	}

	typ, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil || !ok {
		t.Fatalf("Type(587) = %+v, %v, %v", typ, ok, err)
	}
	if len(typ.DogmaAttributes) != 1 || typ.DogmaAttributes[0].AttributeID != 182 {
		t.Errorf("Type(587).DogmaAttributes = %+v, want one entry with AttributeID=182 (from typeDogma.jsonl)", typ.DogmaAttributes)
	}
	if len(typ.DogmaEffects) != 1 || typ.DogmaEffects[0].EffectID != 4 {
		t.Errorf("Type(587).DogmaEffects = %+v, want one entry with EffectID=4", typ.DogmaEffects)
	}
	if typ.PackagedVolume == nil || *typ.PackagedVolume != 2500 {
		t.Errorf("Type(587).PackagedVolume = %v, want 2500 (from reference-data)", typ.PackagedVolume)
	}

	sys, ok, err := store.System(context.Background(), testUserAgent, 30000001)
	if err != nil || !ok {
		t.Fatalf("System(30000001) = %+v, %v, %v", sys, ok, err)
	}
	if len(sys.StationIDs) != 1 || sys.StationIDs[0] != 60000001 {
		t.Errorf("System(30000001).StationIDs = %v, want [60000001] (reverse index from npcStations.jsonl)", sys.StationIDs)
	}
	if len(sys.Planets) != 1 || sys.Planets[0].PlanetID != 40000002 || len(sys.Planets[0].MoonIDs) != 1 || len(sys.Planets[0].AsteroidBeltIDs) != 1 {
		t.Errorf("System(30000001).Planets = %+v, want one entry PlanetID=40000002 with 1 moon and 1 belt (from esi-scrape)", sys.Planets)
	}

	factions, err := store.Factions(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(factions) != 1 {
		t.Fatalf("Factions = %+v, want one entry", factions)
	}
	if factions[0].StationCount != 1 || factions[0].StationSystemCount != 1 {
		t.Errorf("Faction(500001) StationCount/StationSystemCount = %d/%d, want 1/1 (3-hop join: station 60000001 -> corp 1000035 -> faction 500001)",
			factions[0].StationCount, factions[0].StationSystemCount)
	}

	races, err := store.Races(context.Background(), testUserAgent)
	if err != nil || len(races) != 1 {
		t.Errorf("Races = %+v, %v, want one entry", races, err)
	}
	if races[0].AllianceID == nil || *races[0].AllianceID != 500001 {
		t.Errorf("Races[0].AllianceID = %v, want 500001 (from esi-scrape)", races[0].AllianceID)
	}

	region, ok, err := store.Region(context.Background(), testUserAgent, 10000001)
	if err != nil || !ok || region.Name["en"] != "Derelik" {
		t.Errorf("Region(10000001) = %+v, %v, %v, want Name[en]=Derelik", region, ok, err)
	}
	constellation, ok, err := store.Constellation(context.Background(), testUserAgent, 20000001)
	if err != nil || !ok || constellation.Name["en"] != "San Matar" {
		t.Errorf("Constellation(20000001) = %+v, %v, %v, want Name[en]=San Matar", constellation, ok, err)
	}
	ancestries, err := store.Ancestries(context.Background(), testUserAgent)
	if err != nil || len(ancestries) != 1 {
		t.Errorf("Ancestries = %+v, %v, want one entry", ancestries, err)
	}
	bloodlines, err := store.Bloodlines(context.Background(), testUserAgent)
	if err != nil || len(bloodlines) != 1 {
		t.Errorf("Bloodlines = %+v, %v, want one entry", bloodlines, err)
	}
	graphic, ok, err := store.Graphic(context.Background(), testUserAgent, 10)
	if err != nil || !ok || graphic.SofDNA == nil || *graphic.SofDNA != "cf1_t1:caldaribase:caldari" {
		t.Errorf("Graphic(10) = %+v, %v, %v, want SofDNA=cf1_t1:caldaribase:caldari (from esi-scrape)", graphic, ok, err)
	}

	belt, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40000003)
	if err != nil || !ok || belt.Name != "Tanoo I - Asteroid Belt 1" {
		t.Errorf("AsteroidBelt(40000003) = %+v, %v, %v, want Name=\"Tanoo I - Asteroid Belt 1\"", belt, ok, err)
	}
	moon, ok, err := store.Moon(context.Background(), testUserAgent, 40000004)
	if err != nil || !ok || moon.Name != "Tanoo I - Moon 1" {
		t.Errorf("Moon(40000004) = %+v, %v, %v, want Name=\"Tanoo I - Moon 1\"", moon, ok, err)
	}
	planet, ok, err := store.Planet(context.Background(), testUserAgent, 40000002)
	if err != nil || !ok || planet.Name != "Tanoo I" {
		t.Errorf("Planet(40000002) = %+v, %v, %v, want Name=\"Tanoo I\"", planet, ok, err)
	}
	stargate, ok, err := store.Stargate(context.Background(), testUserAgent, 50000001)
	if err != nil || !ok || stargate.DestinationStargateID != 50000482 {
		t.Errorf("Stargate(50000001) = %+v, %v, %v, want DestinationStargateID=50000482", stargate, ok, err)
	}
	star, ok, err := store.Star(context.Background(), testUserAgent, 40000001)
	if err != nil || !ok || star.SpectralClass != "K2 V" {
		t.Errorf("Star(40000001) = %+v, %v, %v, want SpectralClass=\"K2 V\"", star, ok, err)
	}
	station, ok, err := store.Station(context.Background(), testUserAgent, 60000001)
	if err != nil || !ok || station.Name != "Test Station" {
		t.Errorf("Station(60000001) = %+v, %v, %v, want Name=\"Test Station\"", station, ok, err)
	}

	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times, want exactly 1", srv.zipHits.Load())
	}
	if srv.referenceDataHits.Load() != 1 {
		t.Errorf("reference-data requested %d times, want exactly 1", srv.referenceDataHits.Load())
	}
	if srv.esiScrapeHits.Load() != 1 {
		t.Errorf("esi-scrape requested %d times, want exactly 1", srv.esiScrapeHits.Load())
	}
}

func TestSync_ReferenceDataFetchFails_RestOfSyncStillSucceeds(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)
	store.referenceDataURL = "http://127.0.0.1:1/does-not-exist" // guaranteed connection failure

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v, want nil (reference-data failure must not be fatal)", err)
	}

	assertRecord(t, store, "categories", 6, "Ship", true)

	typ, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil || !ok {
		t.Fatalf("Type(587) = %+v, %v, %v", typ, ok, err)
	}
	if typ.PackagedVolume != nil {
		t.Errorf("Type(587).PackagedVolume = %v, want nil (reference-data fetch failed, no augmentation this cycle)", typ.PackagedVolume)
	}
}

func TestSync_EsiScrapeFetchFails_RestOfSyncStillSucceeds(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)
	store.esiScrapeURL = "http://127.0.0.1:1/does-not-exist" // guaranteed connection failure

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v, want nil (esi-scrape failure must not be fatal)", err)
	}

	assertRecord(t, store, "categories", 6, "Ship", true)

	races, err := store.Races(context.Background(), testUserAgent)
	if err != nil || len(races) != 1 {
		t.Fatalf("Races = %+v, %v, want one entry (races.jsonl itself still loads from the SDE)", races, err)
	}
	if races[0].AllianceID != nil {
		t.Errorf("Races[0].AllianceID = %v, want nil (esi-scrape fetch failed, no augmentation this cycle)", races[0].AllianceID)
	}
	if bucketExists(t, store, "asteroidBelts") {
		t.Error("asteroidBelts bucket exists after an esi-scrape fetch failure, want no bucket (nothing to write)")
	}

	belt, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40000003)
	if err != nil {
		t.Fatalf("AsteroidBelt returned error: %v", err)
	}
	if ok {
		t.Errorf("AsteroidBelt ok = true, want false: %+v", belt)
	}
}

func TestSync_OnceOnlyPerStoreLifetime(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first ensureSynced returned error: %v", err)
	}
	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second ensureSynced returned error: %v", err)
	}

	if srv.indexHits.Load() != 1 {
		t.Errorf("index requested %d times across two ensureSynced calls on the same Store, want exactly 1", srv.indexHits.Load())
	}
}

func TestSync_ConcurrentEnsureSyncedCallsOnlySyncOnce(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.ensureSynced(context.Background(), testUserAgent)
		}()
	}
	wg.Wait()

	if srv.indexHits.Load() != 1 {
		t.Errorf("index requested %d times across 20 concurrent ensureSynced calls, want exactly 1", srv.indexHits.Load())
	}
}

func TestSync_UnchangedEtagSkipsDownload(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))

	first := openAtPathWithFixtureServer(t, dbPath, srv)
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}
	if srv.zipHits.Load() != 1 {
		t.Fatalf("zip requested %d times on first sync, want exactly 1", srv.zipHits.Load())
	}

	second := openAtPathWithFixtureServer(t, dbPath, srv)
	t.Cleanup(func() { second.Close() })
	if err := second.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times after a second Store synced against an unchanged etag, want still exactly 1", srv.zipHits.Load())
	}
	if srv.indexHits.Load() != 2 {
		t.Errorf("index requested %d times total, want exactly 2 (one per Store)", srv.indexHits.Load())
	}
}

func TestSync_ChangedEtagReloads(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")

	srv1 := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	first := openAtPathWithFixtureServer(t, dbPath, srv1)
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}

	srv2 := newFixtureServer(t, "etag-v2", buildFixtureZipWithRenamedShipCategory(t))
	second := openAtPathWithFixtureServer(t, dbPath, srv2)
	t.Cleanup(func() { second.Close() })
	if err := second.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	assertRecord(t, second, "categories", 6, "Ship Hull", true)
	if srv2.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times on the changed-etag sync, want exactly 1", srv2.zipHits.Load())
	}
}

func TestSync_FailedSyncFallsBackToCachedData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")

	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	first := openAtPathWithFixtureServer(t, dbPath, srv)
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failingServer.Close)

	second := openAtPathWithIndexURL(t, dbPath, failingServer.URL+"/index.json")
	t.Cleanup(func() { second.Close() })

	if err := second.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error from the failing index server")
	}

	assertRecord(t, second, "categories", 6, "Ship", true)
}

func TestSync_EmptyRemoteEtagReturnsError(t *testing.T) {
	srv := newFixtureServer(t, "", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error for an empty remote etag")
	}
	if srv.zipHits.Load() != 0 {
		t.Errorf("zip requested %d times, want 0 (should fail before attempting a download)", srv.zipHits.Load())
	}
}

func TestSync_FailedResourceLoadRollsBackWholeTransaction(t *testing.T) {
	files := fullFixtureFiles("Ship")
	files["types.jsonl"] = []string{"not valid json"}
	badZip := buildZip(t, files)
	srv := newFixtureServer(t, "etag-v1", badZip)
	store := openWithFixtureServer(t, srv)

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want a parse error from the malformed types.jsonl line")
	}

	v, err := store.version()
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if v != "" {
		t.Errorf("version = %q, want empty (a failed sync must not persist a version)", v)
	}
	for _, bucket := range []string{
		"categories", "groups", "types", "regions", "constellations", "systems", "factions",
		"ancestries", "bloodlines", "races", "graphics",
		"asteroidBelts", "moons", "planets", "stargates", "stars", "stations",
	} {
		if bucketExists(t, store, bucket) {
			t.Errorf("bucket %q exists after a failed sync, want no bucket (the whole transaction should roll back)", bucket)
		}
	}
}

func TestSync_MalformedIndexJSON_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not valid json"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	store := openWithIndexURL(t, srv.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want a parse error for malformed index.json")
	}
}

func TestSync_IndexMissingSDEZipEntry_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		idx := map[string]any{
			"files": []map[string]string{
				{"name": "some-other-file.zip", "url": "http://example.invalid/other.zip", "etag": "etag-x"},
			},
		}
		_ = json.NewEncoder(w).Encode(idx)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	store := openWithIndexURL(t, srv.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the index has no eve-online-static-data-latest-jsonl.zip entry")
	}
}

func TestSync_ZipFetchFails_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		idx := map[string]any{
			"files": []map[string]string{
				{"name": "eve-online-static-data-latest-jsonl.zip", "url": srv.URL + "/sde.zip", "etag": "etag-v1"},
			},
		}
		_ = json.NewEncoder(w).Encode(idx)
	})
	mux.HandleFunc("/sde.zip", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	store := openWithIndexURL(t, srv.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the zip fetch itself fails")
	}
}

func TestSync_ZipMissingExpectedEntry_ReturnsError(t *testing.T) {
	files := fullFixtureFiles("Ship")
	delete(files, "types.jsonl")
	incompleteZip := buildZip(t, files)
	srv := newFixtureServer(t, "etag-v1", incompleteZip)
	store := openWithFixtureServer(t, srv)

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the archive is missing an expected JSONL file")
	}
}

func TestParseEsiScrapeMembers_MissingMember_LeavesFieldNilNotEmpty(t *testing.T) {
	members := map[string][]byte{
		esiScrapePathPrefix + "moons.en-us.yaml": []byte(`"40000004":
  name: "Tanoo I - Moon 1"
  position: {x: 1, "y": 2, z: 3}
  system_id: 30000001
`),
		// asteroid_belts.en-us.yaml deliberately absent -- simulates a
		// missing/renamed tar member, distinct from "present but empty".
	}

	data, err := parseEsiScrapeMembers(context.Background(), members)
	if err != nil {
		t.Fatalf("parseEsiScrapeMembers returned error: %v", err)
	}
	if data.asteroidBelts != nil {
		t.Errorf("asteroidBelts = %#v, want nil (missing member must not produce a non-nil empty map -- writeEsiScrapeBucket's nil-guard only skips a true nil, and a non-nil empty map would wipe any prior synced data)", data.asteroidBelts)
	}
	if data.moons == nil || len(data.moons) != 1 {
		t.Errorf("moons = %#v, want one entry (present member should still decode normally)", data.moons)
	}
}

func TestSync_OlderSchemaVersion_TriggersResyncEvenIfZipEtagUnchanged(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))

	// Simulate an install synced under pre-schema-versioning code: _meta's
	// "version" key holds the bare etag, with no schema prefix at all.
	store := openAtPathWithFixtureServer(t, dbPath, srv)
	if err := store.db.Update(func(tx *bbolt.Tx) error {
		meta, err := tx.CreateBucketIfNotExists([]byte("_meta"))
		if err != nil {
			return err
		}
		return meta.Put([]byte("version"), []byte("etag-v1"))
	}); err != nil {
		t.Fatalf("seed old-format version: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v", err)
	}

	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times, want exactly 1 (a bare-etag cached version with no schema prefix must not look up to date against the current schema)", srv.zipHits.Load())
	}
	belt, ok, err := store.AsteroidBelt(context.Background(), testUserAgent, 40000003)
	if err != nil || !ok || belt.Name == "" {
		t.Errorf("AsteroidBelt(40000003) = %+v, %v, %v, want a real name (proves the forced resync actually populated the new schema's buckets, not just that a fetch happened)", belt, ok, err)
	}
}

func TestSync_EsiScrapeFailureThenRecovery_RetriesOnNextSync(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))

	first := openAtPathWithFixtureServer(t, dbPath, srv)
	first.esiScrapeURL = "http://127.0.0.1:1/does-not-exist" // guaranteed connection failure
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if _, ok, err := first.AsteroidBelt(context.Background(), testUserAgent, 40000003); err != nil || ok {
		t.Fatalf("AsteroidBelt after first (esi-scrape-down) sync: ok=%v err=%v, want ok=false", ok, err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}

	// Second cycle: SDE zip etag unchanged, esi-scrape now healthy. Without
	// the fix, the first cycle's failure would never have been recorded as
	// "still needs a retry", so this second sync would see nothing to do.
	second := openAtPathWithFixtureServer(t, dbPath, srv)
	t.Cleanup(func() { second.Close() })
	if err := second.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	if srv.esiScrapeHits.Load() != 1 {
		t.Errorf("esi-scrape requested %d times, want exactly 1 (only the second, healthy cycle should ever reach the real fixture server)", srv.esiScrapeHits.Load())
	}
	belt, ok, err := second.AsteroidBelt(context.Background(), testUserAgent, 40000003)
	if err != nil || !ok || belt.Name != "Tanoo I - Asteroid Belt 1" {
		t.Errorf("AsteroidBelt(40000003) after recovery = %+v, %v, %v, want the real fixture name (proves the second cycle actually retried esi-scrape rather than treating the first cycle's failure as done)", belt, ok, err)
	}
}

func TestSync_OnlyReferenceDataChanged_TriggersIndependentResync(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sde.db")
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))

	first := openAtPathWithFixtureServer(t, dbPath, srv)
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}
	if srv.zipHits.Load() != 1 || srv.referenceDataHits.Load() != 1 {
		t.Fatalf("setup: zip=%d referenceData=%d, want 1/1", srv.zipHits.Load(), srv.referenceDataHits.Load())
	}

	// Only reference-data's own remote version changes -- the SDE zip's
	// etag and esi-scrape's version marker stay exactly the same.
	srv.referenceDataEtag = "reference-data-etag-v2"

	second := openAtPathWithFixtureServer(t, dbPath, srv)
	t.Cleanup(func() { second.Close() })
	if err := second.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	if srv.referenceDataHits.Load() != 2 {
		t.Errorf("reference-data requested %d times, want exactly 2 (independent staleness tracking must trigger a refetch on its own remote-version change alone, not only when the SDE zip's etag changes)", srv.referenceDataHits.Load())
	}
}

func TestCategory_TriggersRealSyncOnNeverSyncedStore(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithFixtureServer(t, srv)

	got, ok, err := store.Category(context.Background(), testUserAgent, 6)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok || got.Name["en"] != "Ship" {
		t.Errorf("Category(6) = %+v, ok=%v, want Name[en]=Ship (from a real triggered sync)", got, ok)
	}
	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times, want exactly 1 (Category should trigger the lazy sync itself)", srv.zipHits.Load())
	}
}

// TestSync_FactionStationAggregation_DistinguishesCountFromSystemCount uses
// its own fixture (rather than fullFixtureFiles, shared by ~15 other tests)
// with 2 stations in one system plus a 3rd in a different system, all owned
// by the same faction -- fullFixtureFiles' original 1-station fixture
// couldn't distinguish StationCount from StationSystemCount, since a bug
// conflating the two aggregates would still produce matching values (1/1)
// either way.
func TestSync_FactionStationAggregation_DistinguishesCountFromSystemCount(t *testing.T) {
	files := fullFixtureFiles("Ship")
	files["npcStations.jsonl"] = []string{
		`{"_key": 60000001, "solarSystemID": 30000001, "ownerID": 1000035}`,
		`{"_key": 60000002, "solarSystemID": 30000001, "ownerID": 1000035}`,
		`{"_key": 60000003, "solarSystemID": 30000002, "ownerID": 1000035}`,
	}
	srv := newFixtureServer(t, "etag-v1", buildZip(t, files))
	store := openWithFixtureServer(t, srv)

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v", err)
	}

	factions, err := store.Factions(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(factions) != 1 {
		t.Fatalf("Factions = %+v, want one entry", factions)
	}
	if factions[0].StationCount != 3 {
		t.Errorf("StationCount = %d, want 3 (one per station, regardless of system)", factions[0].StationCount)
	}
	if factions[0].StationSystemCount != 2 {
		t.Errorf("StationSystemCount = %d, want 2 (distinct systems 30000001 and 30000002, not 3 -- a bug conflating the two aggregates would show 3 here too)", factions[0].StationSystemCount)
	}
}
