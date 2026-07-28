package sde

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/ulikunitz/xz"
	"go.etcd.io/bbolt"
)

const sdeZipName = "eve-online-static-data-latest-jsonl.zip"
const referenceDataURL = "https://data.everef.net/reference-data/reference-data-latest.tar.xz"
const esiScrapeURL = "https://data.everef.net/esi-scrape/eve-ref-esi-scrape-latest.tar.xz"
const esiScrapePathPrefix = "eve-ref-esi-scrape/data/tranquility/universe/"

// schemaVersion identifies the shape of what sync() writes into bbolt --
// independent of the SDE zip's own etag. Bump this whenever a *Record
// struct, a loader's augmentation logic, or a new bucket is added or
// changed, so an install that already synced under older code detects the
// mismatch and does a full resync instead of silently keeping a
// stale-shaped cache (an old-format record still decodes cleanly into a
// newer Go struct -- every new field is a pointer/omitempty -- so nothing
// about a stale schema is otherwise observable).
const schemaVersion = "2"

var httpClient = &http.Client{Timeout: 30 * time.Second}

type sdeIndex struct {
	Files []sdeIndexFile `json:"files"`
}

type sdeIndexFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Etag string `json:"etag"`
}

// ensureSynced checks the store's cached SDE version against EVE Ref's live
// index exactly once per Store lifetime (via s.syncOnce), downloading and
// reloading every resource if it's stale or missing. A failed sync is
// cached too -- lookups fall back to whatever's already in bbolt (empty,
// on a never-synced install) rather than retrying every call. SDE data is
// enrichment, not a hard requirement for any ESI tool to function -- callers
// are expected to discard this method's return value, but the failure is
// still logged here (once, at the point it happens) so it isn't completely
// invisible when it does occur.
func (s *Store) ensureSynced(ctx context.Context, userAgent string) error {
	s.syncOnce.Do(func() {
		s.syncErr = s.sync(ctx, userAgent)
		if s.syncErr != nil {
			slog.Default().ErrorContext(ctx, "sde: sync failed, falling back to cached data", "error", s.syncErr)
		}
	})
	return s.syncErr
}

func (s *Store) sync(ctx context.Context, userAgent string) error {
	remoteEtag, zipURL, err := latestZipInfo(ctx, s.indexURL, userAgent)
	if err != nil {
		return fmt.Errorf("fetch sde index: %w", err)
	}
	if remoteEtag == "" {
		// A never-synced Store also has currentVersion == "" -- without this
		// check, an index response missing its etag field would silently
		// look "already up to date" and permanently skip downloading.
		return fmt.Errorf("sde index returned an empty etag for %s", sdeZipName)
	}
	remoteSDEVersion := schemaVersion + "|" + remoteEtag

	// reference-data/esi-scrape are tracked independently of the SDE zip's
	// own release cadence -- both expose their own ETag/Last-Modified, so a
	// transient EVE Ref outage that happens to coincide with an SDE release
	// doesn't leave those two sources permanently stale until CCP's next
	// zip. A HEAD failure here is treated as "assume changed" (attempt a
	// fetch this cycle) rather than fatal -- the same best-effort principle
	// as the fetches themselves.
	remoteReferenceDataVersion, refHeadErr := fetchRemoteVersion(ctx, s.referenceDataURL, userAgent)
	remoteEsiScrapeVersion, scrapeHeadErr := fetchRemoteVersion(ctx, s.esiScrapeURL, userAgent)

	currentSDEVersion, currentReferenceDataVersion, currentEsiScrapeVersion, err := s.versions()
	if err != nil {
		return fmt.Errorf("read cached sde versions: %w", err)
	}

	sdeChanged := currentSDEVersion != remoteSDEVersion
	referenceDataChanged := refHeadErr != nil || currentReferenceDataVersion != remoteReferenceDataVersion
	esiScrapeChanged := scrapeHeadErr != nil || currentEsiScrapeVersion != remoteEsiScrapeVersion
	if !sdeChanged && !referenceDataChanged && !esiScrapeChanged {
		return nil
	}

	// Every bucket below is recomputed from the zip's data (EVE Ref sources
	// only ever augment fields on top of it) -- so any of the three sources
	// needing a refresh means the zip must be re-fetched and re-parsed too,
	// even if the zip's own etag didn't change. This costs bandwidth on the
	// (rare) case that only an EVE Ref source changed, but keeps the sync
	// pipeline's existing single-atomic-transaction design rather than
	// requiring a new partial-bucket-merge capability.
	body, err := fetchBytes(ctx, zipURL, userAgent)
	if err != nil {
		return fmt.Errorf("download sde archive: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("open sde archive: %w", err)
	}

	// reference-data and esi-scrape are best-effort: a failure here logs a
	// warning and degrades to an empty result, never fails the sync the
	// core SDE zip is about to perform below. referenceDataOK/esiScrapeOK
	// gate whether that source's version marker gets persisted below -- a
	// failed fetch must leave the cached version stale so the next sync
	// attempt (not necessarily the next process, since this one already
	// detected the remote change) keeps retrying instead of appearing
	// permanently up to date.
	packagedVolumes := fetchReferenceDataTypeVolumes(ctx, s.referenceDataURL, userAgent)
	referenceDataOK := packagedVolumes != nil
	scrapeData, esiScrapeOK := fetchEsiScrapeSupplements(ctx, s.esiScrapeURL, userAgent)

	return s.db.Update(func(tx *bbolt.Tx) error {
		// Types/groups/categories: each level needs the level below it's
		// reverse index before it can be written.
		dogma, err := loadTypeDogma(zr)
		if err != nil {
			return fmt.Errorf("load typeDogma.jsonl: %w", err)
		}
		typeIDsByGroup, err := loadTypes(tx, zr, dogma, packagedVolumes)
		if err != nil {
			return fmt.Errorf("load types.jsonl: %w", err)
		}
		categoryToGroups, err := loadGroups(tx, zr, typeIDsByGroup)
		if err != nil {
			return fmt.Errorf("load groups.jsonl: %w", err)
		}
		if err := loadCategories(tx, zr, categoryToGroups); err != nil {
			return fmt.Errorf("load categories.jsonl: %w", err)
		}

		// Systems/factions: both need a slim pass over npcStations.jsonl;
		// factions additionally need npcCorporations.jsonl to resolve each
		// station's owning corp to a faction.
		corpToFaction, err := loadCorpFactionMap(zr)
		if err != nil {
			return fmt.Errorf("load npcCorporations.jsonl: %w", err)
		}
		stationsBySystem, factionAgg, err := loadStationIndices(zr, corpToFaction)
		if err != nil {
			return fmt.Errorf("load npcStations.jsonl: %w", err)
		}
		if err := loadSystems(tx, zr, stationsBySystem, scrapeData.systemPlanets); err != nil {
			return fmt.Errorf("load mapSolarSystems.jsonl: %w", err)
		}
		if err := loadFactions(tx, zr, factionAgg); err != nil {
			return fmt.Errorf("load factions.jsonl: %w", err)
		}

		// races/graphics need one esi-scrape-sourced merge each; every
		// other passthrough resource needs no augmentation at all.
		if err := loadRaces(tx, zr, scrapeData.raceAlliances); err != nil {
			return fmt.Errorf("load races.jsonl: %w", err)
		}
		if err := loadGraphics(tx, zr, scrapeData.graphicSofDNA); err != nil {
			return fmt.Errorf("load graphics.jsonl: %w", err)
		}
		passthrough := []struct{ file, bucket string }{
			{"mapRegions.jsonl", "regions"},
			{"mapConstellations.jsonl", "constellations"},
			{"ancestries.jsonl", "ancestries"},
			{"bloodlines.jsonl", "bloodlines"},
		}
		for _, p := range passthrough {
			if err := loadResource(tx, zr, p.file, p.bucket); err != nil {
				return fmt.Errorf("load %s: %w", p.file, err)
			}
		}

		// The 6 celestial-detail resources have no official SDE source at
		// all -- entirely from esi-scrape (or empty, if that fetch failed
		// this cycle; every Store accessor already falls through to a live
		// ESI call on a cache miss regardless of why the cache is empty).
		if err := loadAsteroidBelts(tx, scrapeData.asteroidBelts); err != nil {
			return fmt.Errorf("load esi-scrape asteroid belts: %w", err)
		}
		if err := loadMoons(tx, scrapeData.moons); err != nil {
			return fmt.Errorf("load esi-scrape moons: %w", err)
		}
		if err := loadPlanets(tx, scrapeData.planets); err != nil {
			return fmt.Errorf("load esi-scrape planets: %w", err)
		}
		if err := loadStargates(tx, scrapeData.stargates); err != nil {
			return fmt.Errorf("load esi-scrape stargates: %w", err)
		}
		if err := loadStars(tx, scrapeData.stars); err != nil {
			return fmt.Errorf("load esi-scrape stars: %w", err)
		}
		if err := loadStations(tx, scrapeData.stations); err != nil {
			return fmt.Errorf("load esi-scrape stations: %w", err)
		}

		meta, err := tx.CreateBucketIfNotExists([]byte("_meta"))
		if err != nil {
			return fmt.Errorf("create _meta bucket: %w", err)
		}
		// The zip itself is guaranteed fetched and parsed successfully by
		// this point (errors above already returned), so its version is
		// always persisted. reference-data/esi-scrape's versions are only
		// persisted if that specific source's fetch succeeded this cycle --
		// leaving a failed source's cached version stale (not advanced to
		// the version we just observed remotely) is exactly what makes the
		// next sync attempt notice the mismatch and retry it, rather than
		// this cycle's failure looking identical to "nothing ever changed".
		if err := meta.Put([]byte("version"), []byte(remoteSDEVersion)); err != nil {
			return fmt.Errorf("write sde version: %w", err)
		}
		if referenceDataOK {
			if err := meta.Put([]byte("referenceDataVersion"), []byte(remoteReferenceDataVersion)); err != nil {
				return fmt.Errorf("write reference-data version: %w", err)
			}
		}
		if esiScrapeOK {
			if err := meta.Put([]byte("esiScrapeVersion"), []byte(remoteEsiScrapeVersion)); err != nil {
				return fmt.Errorf("write esi-scrape version: %w", err)
			}
		}
		return nil
	})
}

// version returns the SDE schema+zip version currently cached in bbolt's
// _meta bucket, or "" if the store has never been synced.
func (s *Store) version() (string, error) {
	return s.metaValue("version")
}

// versions returns all three sources' cached versions in one pass: the SDE
// zip's (schema-composed), reference-data's, and esi-scrape's. Each is ""
// if that source has never been successfully synced.
func (s *Store) versions() (sdeVersion, referenceDataVersion, esiScrapeVersion string, err error) {
	err = s.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte("_meta"))
		if meta == nil {
			return nil
		}
		sdeVersion = string(meta.Get([]byte("version")))
		referenceDataVersion = string(meta.Get([]byte("referenceDataVersion")))
		esiScrapeVersion = string(meta.Get([]byte("esiScrapeVersion")))
		return nil
	})
	return sdeVersion, referenceDataVersion, esiScrapeVersion, err
}

// metaValue reads a single key from the _meta bucket, or "" if the store
// has never been synced or the key was never written.
func (s *Store) metaValue(key string) (string, error) {
	var v string
	err := s.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte("_meta"))
		if meta == nil {
			return nil
		}
		v = string(meta.Get([]byte(key)))
		return nil
	})
	return v, err
}

// fetchRemoteVersion does a lightweight HEAD request to check a URL's
// current version (ETag, falling back to Last-Modified) without
// downloading its body -- used to detect whether reference-data/esi-scrape
// have changed independently of the SDE zip's own release cadence.
func fetchRemoteVersion(ctx context.Context, url, userAgent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", fmt.Errorf("build HEAD request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute HEAD request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from HEAD %s", resp.StatusCode, url)
	}
	if etag := resp.Header.Get("ETag"); etag != "" {
		return etag, nil
	}
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		return lastModified, nil
	}
	return "", fmt.Errorf("HEAD %s returned neither ETag nor Last-Modified", url)
}

// scanRecords opens file inside zr and calls fn once per JSONL line with
// that record's _key and raw bytes, shared by every loader below.
func scanRecords(zr *zip.Reader, file string, fn func(key int64, raw []byte) error) error {
	f, err := zr.Open(file)
	if err != nil {
		return fmt.Errorf("open zip entry: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // types.jsonl has long lines
	for scanner.Scan() {
		line := scanner.Bytes()
		var keyed struct {
			Key int64 `json:"_key"`
		}
		if err := json.Unmarshal(line, &keyed); err != nil {
			return fmt.Errorf("parse record: %w", err)
		}
		if err := fn(keyed.Key, append([]byte(nil), line...)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// loadResource clears and rewrites bucket from file's JSONL lines within
// the zip, verbatim, keyed by each record's _key field. Used for resources
// needing no augmentation.
func loadResource(tx *bbolt.Tx, zr *zip.Reader, file, bucket string) error {
	if err := tx.DeleteBucket([]byte(bucket)); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte(bucket))
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	return scanRecords(zr, file, func(key int64, raw []byte) error {
		if err := b.Put(int64Key(key), raw); err != nil {
			return fmt.Errorf("write record %d: %w", key, err)
		}
		return nil
	})
}

type dogmaData struct {
	Attributes []typeDogmaAttributeRecord `json:"dogmaAttributes,omitempty"`
	Effects    []typeDogmaEffectRecord    `json:"dogmaEffects,omitempty"`
}

// loadTypeDogma reads typeDogma.jsonl into an in-memory typeID -> dogma map;
// it has no bucket of its own, only used to augment types.jsonl records.
func loadTypeDogma(zr *zip.Reader) (map[int64]dogmaData, error) {
	out := make(map[int64]dogmaData)
	err := scanRecords(zr, "typeDogma.jsonl", func(key int64, raw []byte) error {
		var d dogmaData
		if err := json.Unmarshal(raw, &d); err != nil {
			return fmt.Errorf("parse typeDogma record %d: %w", key, err)
		}
		out[key] = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// loadTypes writes the types bucket (augmented with dogma and, if
// packagedVolumes is non-nil, PackagedVolume from reference-data) and
// returns the groupID -> []typeID reverse index groups.jsonl needs next.
func loadTypes(tx *bbolt.Tx, zr *zip.Reader, dogma map[int64]dogmaData, packagedVolumes map[int64]float64) (map[int64][]int64, error) {
	if err := tx.DeleteBucket([]byte("types")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return nil, fmt.Errorf("clear types bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("types"))
	if err != nil {
		return nil, fmt.Errorf("create types bucket: %w", err)
	}

	groupToTypes := make(map[int64][]int64)
	err = scanRecords(zr, "types.jsonl", func(key int64, raw []byte) error {
		var rec typeRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse type record %d: %w", key, err)
		}
		if d, ok := dogma[key]; ok {
			rec.DogmaAttributes = d.Attributes
			rec.DogmaEffects = d.Effects
		}
		if v, ok := packagedVolumes[key]; ok {
			rec.PackagedVolume = &v
		}
		groupToTypes[rec.GroupID] = append(groupToTypes[rec.GroupID], key)
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented type %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
	if err != nil {
		return nil, err
	}
	return groupToTypes, nil
}

// loadGroups writes the groups bucket (augmented with TypeIDs from the map
// loadTypes produced) and returns the categoryID -> []groupID reverse index
// categories.jsonl needs next.
func loadGroups(tx *bbolt.Tx, zr *zip.Reader, typeIDsByGroup map[int64][]int64) (map[int64][]int64, error) {
	if err := tx.DeleteBucket([]byte("groups")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return nil, fmt.Errorf("clear groups bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("groups"))
	if err != nil {
		return nil, fmt.Errorf("create groups bucket: %w", err)
	}

	categoryToGroups := make(map[int64][]int64)
	err = scanRecords(zr, "groups.jsonl", func(key int64, raw []byte) error {
		var rec groupRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse group record %d: %w", key, err)
		}
		rec.TypeIDs = typeIDsByGroup[key]
		categoryToGroups[rec.CategoryID] = append(categoryToGroups[rec.CategoryID], key)
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented group %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
	if err != nil {
		return nil, err
	}
	return categoryToGroups, nil
}

// loadCategories writes the categories bucket, augmented with GroupIDs from
// the map loadGroups produced.
func loadCategories(tx *bbolt.Tx, zr *zip.Reader, groupIDsByCategory map[int64][]int64) error {
	if err := tx.DeleteBucket([]byte("categories")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear categories bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("categories"))
	if err != nil {
		return fmt.Errorf("create categories bucket: %w", err)
	}
	return scanRecords(zr, "categories.jsonl", func(key int64, raw []byte) error {
		var rec categoryRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse category record %d: %w", key, err)
		}
		rec.GroupIDs = groupIDsByCategory[key]
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented category %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

type corpFactionRecord struct {
	FactionID *int64 `json:"factionID,omitempty"`
}

// loadCorpFactionMap reads a slim pass over npcCorporations.jsonl (just the
// optional factionID field) into an in-memory corpID -> factionID map; it
// has no bucket of its own.
func loadCorpFactionMap(zr *zip.Reader) (map[int64]int64, error) {
	out := make(map[int64]int64)
	err := scanRecords(zr, "npcCorporations.jsonl", func(key int64, raw []byte) error {
		var rec corpFactionRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse npcCorporations record %d: %w", key, err)
		}
		if rec.FactionID != nil {
			out[key] = *rec.FactionID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type stationSlimRecord struct {
	SolarSystemID int64 `json:"solarSystemID"`
	OwnerID       int64 `json:"ownerID"`
}

type factionStationAgg struct {
	StationCount int64
	Systems      map[int64]struct{}
}

// loadStationIndices does one slim pass over npcStations.jsonl, producing
// both the solarSystemID -> []stationID reverse index mapSolarSystems.jsonl
// needs, and the factionID -> station/system-count aggregate factions.jsonl
// needs (via corpToFaction, from loadCorpFactionMap) -- reading the file
// once serves both, rather than scanning it twice.
func loadStationIndices(zr *zip.Reader, corpToFaction map[int64]int64) (stationsBySystem map[int64][]int64, factionAgg map[int64]*factionStationAgg, err error) {
	stationsBySystem = make(map[int64][]int64)
	factionAgg = make(map[int64]*factionStationAgg)

	err = scanRecords(zr, "npcStations.jsonl", func(key int64, raw []byte) error {
		var rec stationSlimRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse npcStations record %d: %w", key, err)
		}
		stationsBySystem[rec.SolarSystemID] = append(stationsBySystem[rec.SolarSystemID], key)

		if factionID, ok := corpToFaction[rec.OwnerID]; ok {
			agg, exists := factionAgg[factionID]
			if !exists {
				agg = &factionStationAgg{Systems: make(map[int64]struct{})}
				factionAgg[factionID] = agg
			}
			agg.StationCount++
			agg.Systems[rec.SolarSystemID] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return stationsBySystem, factionAgg, nil
}

// loadSystems writes the systems bucket, augmented with StationIDs from the
// map loadStationIndices produced (mapSolarSystems.jsonl has no stationIDs
// field of its own) and, if systemPlanets is non-nil, Planets from
// esi-scrape (mapSolarSystems.jsonl has no planets field either).
func loadSystems(tx *bbolt.Tx, zr *zip.Reader, stationsBySystem map[int64][]int64, systemPlanets map[int64][]systemPlanetRecord) error {
	if err := tx.DeleteBucket([]byte("systems")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear systems bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("systems"))
	if err != nil {
		return fmt.Errorf("create systems bucket: %w", err)
	}
	return scanRecords(zr, "mapSolarSystems.jsonl", func(key int64, raw []byte) error {
		var rec systemRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse system record %d: %w", key, err)
		}
		rec.StationIDs = stationsBySystem[key]
		rec.Planets = systemPlanets[key]
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented system %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// loadFactions writes the factions bucket, augmented with StationCount/
// StationSystemCount from the aggregate loadStationIndices produced
// (factions.jsonl has neither field itself).
func loadFactions(tx *bbolt.Tx, zr *zip.Reader, factionAgg map[int64]*factionStationAgg) error {
	if err := tx.DeleteBucket([]byte("factions")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear factions bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("factions"))
	if err != nil {
		return fmt.Errorf("create factions bucket: %w", err)
	}
	return scanRecords(zr, "factions.jsonl", func(key int64, raw []byte) error {
		var rec factionRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse faction record %d: %w", key, err)
		}
		if agg, ok := factionAgg[key]; ok {
			rec.StationCount = agg.StationCount
			rec.StationSystemCount = int64(len(agg.Systems))
		}
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented faction %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// loadRaces writes the races bucket, augmented with AllianceID from
// esi-scrape (races.jsonl has no such field, and there is no
// alliances.jsonl anywhere in the SDE archive at all).
func loadRaces(tx *bbolt.Tx, zr *zip.Reader, raceAlliances map[int64]int64) error {
	if err := tx.DeleteBucket([]byte("races")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear races bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("races"))
	if err != nil {
		return fmt.Errorf("create races bucket: %w", err)
	}
	return scanRecords(zr, "races.jsonl", func(key int64, raw []byte) error {
		var rec raceRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse race record %d: %w", key, err)
		}
		if allianceID, ok := raceAlliances[key]; ok {
			rec.AllianceID = &allianceID
		}
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented race %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// loadGraphics writes the graphics bucket, augmented with SofDNA from
// esi-scrape (graphics.jsonl has no such field).
func loadGraphics(tx *bbolt.Tx, zr *zip.Reader, graphicSofDNA map[int64]string) error {
	if err := tx.DeleteBucket([]byte("graphics")); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear graphics bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte("graphics"))
	if err != nil {
		return fmt.Errorf("create graphics bucket: %w", err)
	}
	return scanRecords(zr, "graphics.jsonl", func(key int64, raw []byte) error {
		var rec graphicRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return fmt.Errorf("parse graphic record %d: %w", key, err)
		}
		if dna, ok := graphicSofDNA[key]; ok {
			rec.SofDNA = &dna
		}
		merged, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal augmented graphic %d: %w", key, err)
		}
		return b.Put(int64Key(key), merged)
	})
}

// loadAsteroidBelts writes the asteroidBelts bucket directly from
// esi-scrape records -- no augmentation, no official SDE source at all.
// records is nil (a no-op) if esi-scrape couldn't be fetched this cycle.
func loadAsteroidBelts(tx *bbolt.Tx, records map[int64]asteroidBeltRecord) error {
	return writeEsiScrapeBucket(tx, "asteroidBelts", records)
}

// loadMoons is loadAsteroidBelts' analogue for moons.
func loadMoons(tx *bbolt.Tx, records map[int64]moonRecord) error {
	return writeEsiScrapeBucket(tx, "moons", records)
}

// loadPlanets is loadAsteroidBelts' analogue for planets.
func loadPlanets(tx *bbolt.Tx, records map[int64]planetRecord) error {
	return writeEsiScrapeBucket(tx, "planets", records)
}

// loadStargates is loadAsteroidBelts' analogue for stargates.
func loadStargates(tx *bbolt.Tx, records map[int64]stargateRecord) error {
	return writeEsiScrapeBucket(tx, "stargates", records)
}

// loadStars is loadAsteroidBelts' analogue for stars.
func loadStars(tx *bbolt.Tx, records map[int64]starRecord) error {
	return writeEsiScrapeBucket(tx, "stars", records)
}

// loadStations is loadAsteroidBelts' analogue for stations.
func loadStations(tx *bbolt.Tx, records map[int64]stationRecord) error {
	return writeEsiScrapeBucket(tx, "stations", records)
}

// writeEsiScrapeBucket clears and rewrites bucket from an already-decoded
// esi-scrape record map, keyed by map key. Shared by all 6 celestial
// loaders above -- unlike scanRecords-based loaders, there's no zip/JSONL
// file to stream, just an in-memory map to marshal and write. records is
// nil when esi-scrape couldn't be fetched or parsed this cycle -- in that
// case the bucket is left untouched entirely (not cleared), so a prior
// successful sync's data survives a later cycle's fetch failure instead of
// being wiped out for no benefit.
func writeEsiScrapeBucket[T any](tx *bbolt.Tx, bucket string, records map[int64]T) error {
	if records == nil {
		return nil
	}
	if err := tx.DeleteBucket([]byte(bucket)); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear %s bucket: %w", bucket, err)
	}
	b, err := tx.CreateBucket([]byte(bucket))
	if err != nil {
		return fmt.Errorf("create %s bucket: %w", bucket, err)
	}
	for id, rec := range records {
		data, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal %s record %d: %w", bucket, id, err)
		}
		if err := b.Put(int64Key(id), data); err != nil {
			return fmt.Errorf("write %s record %d: %w", bucket, id, err)
		}
	}
	return nil
}

// esiScrapePosition mirrors sde.Position's shape for YAML decoding --
// Position itself has json tags, not yaml tags, so a separate decode type
// is needed; Position(esiScrapePositionValue) is a valid Go conversion
// since both have identical field names/types/order (struct tags don't
// affect conversion compatibility).
type esiScrapePosition struct {
	X float64 `yaml:"x"`
	Y float64 `yaml:"y"`
	Z float64 `yaml:"z"`
}

type esiScrapeAsteroidBelt struct {
	Name     string            `yaml:"name"`
	Position esiScrapePosition `yaml:"position"`
	SystemID int64             `yaml:"system_id"`
}

type esiScrapeMoon struct {
	Name     string            `yaml:"name"`
	Position esiScrapePosition `yaml:"position"`
	SystemID int64             `yaml:"system_id"`
}

type esiScrapePlanet struct {
	Name     string            `yaml:"name"`
	Position esiScrapePosition `yaml:"position"`
	SystemID int64             `yaml:"system_id"`
	TypeID   int64             `yaml:"type_id"`
}

type esiScrapeStargateDestination struct {
	StargateID int64 `yaml:"stargate_id"`
	SystemID   int64 `yaml:"system_id"`
}

type esiScrapeStargate struct {
	Destination esiScrapeStargateDestination `yaml:"destination"`
	Name        string                       `yaml:"name"`
	Position    esiScrapePosition            `yaml:"position"`
	SystemID    int64                        `yaml:"system_id"`
	TypeID      int64                        `yaml:"type_id"`
}

type esiScrapeStar struct {
	Age           int64   `yaml:"age"`
	Luminosity    float64 `yaml:"luminosity"`
	Name          string  `yaml:"name"`
	Radius        int64   `yaml:"radius"`
	SolarSystemID int64   `yaml:"solar_system_id"`
	SpectralClass string  `yaml:"spectral_class"`
	Temperature   int64   `yaml:"temperature"`
	TypeID        int64   `yaml:"type_id"`
}

type esiScrapeStation struct {
	MaxDockableShipVolume    float64           `yaml:"max_dockable_ship_volume"`
	Name                     string            `yaml:"name"`
	OfficeRentalCost         float64           `yaml:"office_rental_cost"`
	Owner                    *int64            `yaml:"owner"`
	Position                 esiScrapePosition `yaml:"position"`
	RaceID                   *int64            `yaml:"race_id"`
	ReprocessingEfficiency   float64           `yaml:"reprocessing_efficiency"`
	ReprocessingStationsTake float64           `yaml:"reprocessing_stations_take"`
	Services                 []string          `yaml:"services"`
	SystemID                 int64             `yaml:"system_id"`
	TypeID                   int64             `yaml:"type_id"`
}

type esiScrapeRace struct {
	AllianceID int64 `yaml:"alliance_id"`
	RaceID     int64 `yaml:"race_id"`
}

type esiScrapeSystemPlanet struct {
	PlanetID      int64   `yaml:"planet_id"`
	AsteroidBelts []int64 `yaml:"asteroid_belts,omitempty"`
	Moons         []int64 `yaml:"moons,omitempty"`
}

type esiScrapeSystem struct {
	Planets []esiScrapeSystemPlanet `yaml:"planets"`
}

type esiScrapeGraphic struct {
	SofDNA *string `yaml:"sof_dna"`
}

// esiScrapeData bundles every value fetchEsiScrapeSupplements extracts from
// EVE Ref's esi-scrape dataset. A zero-value esiScrapeData (every field
// nil) means the fetch or parse failed -- every augmentation and
// bucket-write using it becomes a no-op, not a crash; the 6 celestial
// buckets simply don't get (re)populated this sync cycle, and every Store
// accessor already falls through to a live ESI call on a cache miss
// regardless of why the cache is empty.
type esiScrapeData struct {
	raceAlliances map[int64]int64
	systemPlanets map[int64][]systemPlanetRecord
	graphicSofDNA map[int64]string
	asteroidBelts map[int64]asteroidBeltRecord
	moons         map[int64]moonRecord
	planets       map[int64]planetRecord
	stargates     map[int64]stargateRecord
	stars         map[int64]starRecord
	stations      map[int64]stationRecord
}

// fetchReferenceDataTypeVolumes fetches EVE Ref's reference-data dataset
// and returns typeID -> packagedVolume for every type that has one.
// Returns nil (not an error) on any failure -- reference-data is a
// best-effort augmentation, never fatal to the rest of sync().
func fetchReferenceDataTypeVolumes(ctx context.Context, url, userAgent string) map[int64]float64 {
	body, err := fetchBytes(ctx, url, userAgent)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to fetch reference-data, skipping PackagedVolume augmentation", "error", err)
		return nil
	}
	members, err := extractTarXZMembers(body, map[string]bool{"types.json": true})
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to extract reference-data, skipping PackagedVolume augmentation", "error", err)
		return nil
	}
	typesJSON, ok := members["types.json"]
	if !ok {
		slog.Default().WarnContext(ctx, "sde: reference-data archive missing types.json, skipping PackagedVolume augmentation")
		return nil
	}
	var raw map[string]struct {
		PackagedVolume *float64 `json:"packaged_volume"`
	}
	if err := json.Unmarshal(typesJSON, &raw); err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to parse reference-data types.json, skipping PackagedVolume augmentation", "error", err)
		return nil
	}
	out := make(map[int64]float64, len(raw))
	for idStr, rec := range raw {
		if rec.PackagedVolume == nil {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		out[id] = *rec.PackagedVolume
	}
	return out
}

// fetchEsiScrapeSupplements fetches EVE Ref's esi-scrape dataset and
// returns every value this plan sources from it. Returns a zero-value
// esiScrapeData (not an error) on any failure -- esi-scrape is a
// best-effort augmentation, never fatal to the rest of sync().
// fetchEsiScrapeSupplements fetches EVE Ref's esi-scrape dataset. ok is
// true if the fetch/extract/parse succeeded overall (individual missing
// members still leave their own field nil within data -- see
// parseEsiScrapeMembers) -- callers use ok to decide whether to advance
// this source's cached staleness-tracking version.
func fetchEsiScrapeSupplements(ctx context.Context, url, userAgent string) (data esiScrapeData, ok bool) {
	body, err := fetchBytes(ctx, url, userAgent)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to fetch esi-scrape, skipping celestial/AllianceID/Planets/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}

	want := map[string]bool{
		esiScrapePathPrefix + "asteroid_belts.en-us.yaml": true,
		esiScrapePathPrefix + "moons.en-us.yaml":          true,
		esiScrapePathPrefix + "planets.en-us.yaml":        true,
		esiScrapePathPrefix + "stargates.en-us.yaml":      true,
		esiScrapePathPrefix + "stars.en-us.yaml":          true,
		esiScrapePathPrefix + "stations.en-us.yaml":       true,
		esiScrapePathPrefix + "races.en-us.yaml":          true,
		esiScrapePathPrefix + "systems.en-us.yaml":        true,
		esiScrapePathPrefix + "graphics.yaml":             true,
	}
	members, err := extractTarXZMembers(body, want)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to extract esi-scrape, skipping celestial/AllianceID/Planets/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}

	data, err = parseEsiScrapeMembers(ctx, members)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to parse esi-scrape, skipping celestial/AllianceID/Planets/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}
	return data, true
}

// decodeEsiScrapeMap decodes an esi-scrape YAML file whose top level is a
// map keyed by string-formatted IDs, converting the keys to int64.
func decodeEsiScrapeMap[T any](raw []byte) (map[int64]T, error) {
	if raw == nil {
		return nil, nil
	}
	var strKeyed map[string]T
	if err := yaml.Unmarshal(raw, &strKeyed); err != nil {
		return nil, err
	}
	out := make(map[int64]T, len(strKeyed))
	for idStr, v := range strKeyed {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse key %q as int64: %w", idStr, err)
		}
		out[id] = v
	}
	return out, nil
}

// esiScrapeMemberPresent reports whether key was actually extracted from the
// archive, warning if not. Callers use this to distinguish "member missing
// from the archive" (leave the corresponding esiScrapeData field nil, so
// writeEsiScrapeBucket's nil-guard skips it and any prior synced data for
// that bucket survives) from "member present but decodes to zero entries"
// (a non-nil empty map, which legitimately overwrites a bucket with no
// data) -- both would otherwise look identical to code downstream of
// decodeEsiScrapeMap, which returns (nil, nil) for a nil input either way.
func esiScrapeMemberPresent(ctx context.Context, members map[string][]byte, key string) bool {
	if _, ok := members[key]; ok {
		return true
	}
	slog.Default().WarnContext(ctx, "sde: esi-scrape archive missing expected member, skipping its augmentation", "member", key)
	return false
}

// parseEsiScrapeMembers decodes every member fetchEsiScrapeSupplements
// extracted. Any single parse failure fails the whole batch -- a format
// change significant enough to break one file is likely to affect the
// others too, and partial augmentation would be harder to reason about
// than none.
func parseEsiScrapeMembers(ctx context.Context, members map[string][]byte) (esiScrapeData, error) {
	var data esiScrapeData

	if key := esiScrapePathPrefix + "asteroid_belts.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		asteroidBelts, err := decodeEsiScrapeMap[esiScrapeAsteroidBelt](members[key])
		if err != nil {
			return esiScrapeData{}, fmt.Errorf("parse asteroid_belts.en-us.yaml: %w", err)
		}
		data.asteroidBelts = make(map[int64]asteroidBeltRecord, len(asteroidBelts))
		for id, r := range asteroidBelts {
			data.asteroidBelts[id] = asteroidBeltRecord{Key: id, Name: r.Name, Position: Position(r.Position), SystemID: r.SystemID}
		}
	}

	if key := esiScrapePathPrefix + "moons.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		moons, err := decodeEsiScrapeMap[esiScrapeMoon](members[key])
		if err != nil {
			return esiScrapeData{}, fmt.Errorf("parse moons.en-us.yaml: %w", err)
		}
		data.moons = make(map[int64]moonRecord, len(moons))
		for id, r := range moons {
			data.moons[id] = moonRecord{Key: id, Name: r.Name, Position: Position(r.Position), SystemID: r.SystemID}
		}
	}

	if key := esiScrapePathPrefix + "planets.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		planets, err := decodeEsiScrapeMap[esiScrapePlanet](members[key])
		if err != nil {
			return esiScrapeData{}, fmt.Errorf("parse planets.en-us.yaml: %w", err)
		}
		data.planets = make(map[int64]planetRecord, len(planets))
		for id, r := range planets {
			data.planets[id] = planetRecord{Key: id, Name: r.Name, TypeID: r.TypeID, Position: Position(r.Position), SystemID: r.SystemID}
		}
	}

	if key := esiScrapePathPrefix + "stargates.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		stargates, err := decodeEsiScrapeMap[esiScrapeStargate](members[key])
		if err != nil {
			return esiScrapeData{}, fmt.Errorf("parse stargates.en-us.yaml: %w", err)
		}
		data.stargates = make(map[int64]stargateRecord, len(stargates))
		for id, r := range stargates {
			data.stargates[id] = stargateRecord{
				Key: id, Name: r.Name, TypeID: r.TypeID, Position: Position(r.Position), SystemID: r.SystemID,
				DestinationStargateID: r.Destination.StargateID, DestinationSystemID: r.Destination.SystemID,
			}
		}
	}

	if key := esiScrapePathPrefix + "stars.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		stars, err := decodeEsiScrapeMap[esiScrapeStar](members[key])
		if err != nil {
			return esiScrapeData{}, fmt.Errorf("parse stars.en-us.yaml: %w", err)
		}
		data.stars = make(map[int64]starRecord, len(stars))
		for id, r := range stars {
			data.stars[id] = starRecord{
				Key: id, Age: r.Age, Luminosity: r.Luminosity, Name: r.Name, Radius: r.Radius,
				SolarSystemID: r.SolarSystemID, SpectralClass: r.SpectralClass, Temperature: r.Temperature, TypeID: r.TypeID,
			}
		}
	}

	if key := esiScrapePathPrefix + "stations.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		stations, err := decodeEsiScrapeMap[esiScrapeStation](members[key])
		if err != nil {
			return esiScrapeData{}, fmt.Errorf("parse stations.en-us.yaml: %w", err)
		}
		data.stations = make(map[int64]stationRecord, len(stations))
		for id, r := range stations {
			data.stations[id] = stationRecord{
				Key: id, Name: r.Name, SystemID: r.SystemID, TypeID: r.TypeID, Position: Position(r.Position),
				MaxDockableShipVolume: r.MaxDockableShipVolume, OfficeRentalCost: r.OfficeRentalCost,
				Owner: r.Owner, RaceID: r.RaceID, ReprocessingEfficiency: r.ReprocessingEfficiency,
				ReprocessingStationsTake: r.ReprocessingStationsTake, Services: r.Services,
			}
		}
	}

	if key := esiScrapePathPrefix + "races.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		var races []esiScrapeRace
		if err := yaml.Unmarshal(members[key], &races); err != nil {
			return esiScrapeData{}, fmt.Errorf("parse races.en-us.yaml: %w", err)
		}
		data.raceAlliances = make(map[int64]int64, len(races))
		for _, r := range races {
			data.raceAlliances[r.RaceID] = r.AllianceID
		}
	}

	if key := esiScrapePathPrefix + "systems.en-us.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		systems, err := decodeEsiScrapeMap[esiScrapeSystem](members[key])
		if err != nil {
			return esiScrapeData{}, fmt.Errorf("parse systems.en-us.yaml: %w", err)
		}
		data.systemPlanets = make(map[int64][]systemPlanetRecord, len(systems))
		for id, r := range systems {
			if len(r.Planets) == 0 {
				continue
			}
			planetRecs := make([]systemPlanetRecord, len(r.Planets))
			for i, p := range r.Planets {
				planetRecs[i] = systemPlanetRecord{PlanetID: p.PlanetID, AsteroidBeltIDs: p.AsteroidBelts, MoonIDs: p.Moons}
			}
			data.systemPlanets[id] = planetRecs
		}
	}

	if key := esiScrapePathPrefix + "graphics.yaml"; esiScrapeMemberPresent(ctx, members, key) {
		var graphics map[string]esiScrapeGraphic
		if err := yaml.Unmarshal(members[key], &graphics); err != nil {
			return esiScrapeData{}, fmt.Errorf("parse graphics.yaml: %w", err)
		}
		data.graphicSofDNA = make(map[int64]string, len(graphics))
		for idStr, g := range graphics {
			if g.SofDNA == nil {
				continue
			}
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				continue
			}
			data.graphicSofDNA[id] = *g.SofDNA
		}
	}

	return data, nil
}

// extractTarXZMembers decompresses an XZ-compressed tar archive and
// returns the raw bytes of every member whose path is in want, keyed by
// that same path. Members not in want are skipped without being read into
// memory. Used for both reference-data and esi-scrape -- neither format
// supports partial/range extraction (XZ isn't seekable), so the whole
// archive is always decompressed regardless of how many members are kept.
func extractTarXZMembers(body []byte, want map[string]bool) (map[string][]byte, error) {
	xr, err := xz.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("open xz stream: %w", err)
	}
	tr := tar.NewReader(xr)
	out := make(map[string][]byte, len(want))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar header: %w", err)
		}
		if !want[hdr.Name] {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read tar member %s: %w", hdr.Name, err)
		}
		out[hdr.Name] = data
	}
	return out, nil
}

// int64Key encodes id as a big-endian byte slice suitable for use as a
// bbolt key.
func int64Key(id int64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(id))
	return key
}

// int64FromKey decodes a key produced by int64Key back into its original
// int64 form.
func int64FromKey(key []byte) int64 {
	return int64(binary.BigEndian.Uint64(key))
}

func latestZipInfo(ctx context.Context, indexURL, userAgent string) (etag, url string, err error) {
	body, err := fetchBytes(ctx, indexURL, userAgent)
	if err != nil {
		return "", "", fmt.Errorf("fetch index: %w", err)
	}
	var idx sdeIndex
	if err := json.Unmarshal(body, &idx); err != nil {
		return "", "", fmt.Errorf("parse index: %w", err)
	}
	for _, f := range idx.Files {
		if f.Name == sdeZipName {
			return f.Etag, f.URL, nil
		}
	}
	return "", "", fmt.Errorf("%s not found in sde index", sdeZipName)
}

func fetchBytes(ctx context.Context, url, userAgent string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}
