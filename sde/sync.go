package sde

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/ulikunitz/xz"
)

const dumpURL = "https://www.fuzzwork.co.uk/dump/latest-sqlite.db.gz"
const referenceDataURL = "https://data.everef.net/reference-data/reference-data-latest.tar.xz"
const esiScrapeURL = "https://data.everef.net/esi-scrape/eve-ref-esi-scrape-latest.tar.xz"
const esiScrapePathPrefix = "eve-ref-esi-scrape/data/tranquility/universe/"

// schemaVersion identifies the shape of what sync() writes to dbPath/metaPath
// -- independent of the dump's own ETag. Bumped here from the old bbolt
// design's "2" to "3" to mark the move to a directly-queried sqlite file, so
// an install that synced under the old bbolt schema (a different dbPath,
// eve.db vs the old sde.db, in fact -- but bumping this too is cheap
// insurance and documents the shape change explicitly) forces a fresh
// download rather than trying to open a file that was never written by this
// code.
const schemaVersion = "3"

// Timeout covers the entire request including body read, so it must be sized
// for the largest download this client makes, not just a typical one -- the
// fuzzwork dump is ~176MB compressed, which a 30s timeout would require
// ~47 Mbit/s sustained to finish within. 5 minutes is generous enough for a
// slow connection on this one-time-per-process download without leaving a
// truly hung request unbounded forever.
var httpClient = &http.Client{Timeout: 5 * time.Minute}

// ensureSynced checks the store's cached dump version against fuzzwork's
// live ETag exactly once per Store lifetime (via s.syncOnce), downloading and
// reopening the database if it's stale or missing. A failed sync is cached
// too -- lookups fall back to whatever's already on disk (nothing, on a
// never-synced install) rather than retrying every call. SDE data is
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
	meta, err := s.readMeta()
	if err != nil {
		return fmt.Errorf("read cached sde meta: %w", err)
	}

	// Open whatever's already on disk before any network call. A later
	// failure below (a HEAD timeout, a broken download) then degrades to
	// "keep serving what's already there" instead of leaving s.db nil for
	// the rest of this process's life -- every accessor's s.db == nil guard
	// otherwise treats a transient network hiccup on an already-synced
	// install identically to "no cache has ever existed," silently routing
	// every SDE-backed tool to live ESI until the next restart.
	if db, ok := s.openAndVerify(s.dbPath); ok {
		s.db = db
	}

	remoteDumpVersion, err := fetchRemoteVersion(ctx, s.dumpURL, userAgent)
	if err != nil {
		if s.db != nil {
			slog.Default().WarnContext(ctx, "sde: dump version check failed, continuing with existing cache", "error", err)
			return nil
		}
		return fmt.Errorf("check sde dump version: %w", err)
	}

	// s.db == nil here covers both "the file is simply missing" (the
	// meta.json/eve.db desync case: a user deletes eve.db while meta.json
	// survives) and "the file exists but failed openAndVerify's sentinel
	// query" (a corrupt or truncated file) -- either way, forcing a
	// download regardless of whether meta already claims to be current is
	// what makes both cases self-healing rather than a permanent,
	// human-intervention-only failure.
	if meta.SchemaVersion != schemaVersion || meta.DumpVersion != remoteDumpVersion || s.db == nil {
		hadExisting := s.db != nil
		if s.db != nil {
			// Close before downloadDump's rename -- an open handle on the
			// destination can make the rename fail on Windows, and there's
			// no reason to hold a connection open to a file about to be
			// replaced on any platform.
			s.db.Close()
			s.db = nil
		}
		if err := s.downloadDump(ctx, userAgent); err != nil {
			if hadExisting {
				// downloadDump only renames over dbPath on full success, so
				// a failure here leaves the previous file untouched --
				// reopen it rather than leaving the cache unusable for the
				// rest of this process over what may be a transient issue.
				if db, ok := s.openAndVerify(s.dbPath); ok {
					s.db = db
					slog.Default().WarnContext(ctx, "sde: dump refresh failed, continuing with previous cache", "error", err)
					return nil
				}
			}
			return fmt.Errorf("download sde dump: %w", err)
		}

		db, ok := s.openAndVerify(s.dbPath)
		if !ok {
			// The freshly downloaded file itself is unusable (a corrupt or
			// truncated transfer) -- remove it so the next sync sees a
			// missing file and retries, rather than inheriting a broken
			// database that meta.json would otherwise mark as current
			// forever.
			os.Remove(s.dbPath)
			return fmt.Errorf("verify downloaded sde dump: database failed sentinel query")
		}
		s.db = db
		meta.SchemaVersion = schemaVersion
		meta.DumpVersion = remoteDumpVersion
		if err := s.writeMeta(meta); err != nil {
			return fmt.Errorf("write sde meta: %w", err)
		}
	}

	// reference-data/esi-scrape augmentation is never persisted to disk (see
	// Store's typeVolumes/raceAlliances/graphicSofDNA doc comment) -- every
	// process's single sync() fetches both fresh, best-effort, regardless of
	// whether either has changed remotely, since there is no local copy to
	// otherwise fall back to skipping.
	s.typeVolumes = fetchReferenceDataTypeVolumes(ctx, s.referenceDataURL, userAgent)
	if data, ok := fetchEsiScrapeSupplements(ctx, s.esiScrapeURL, userAgent); ok {
		s.raceAlliances = data.raceAlliances
		s.graphicSofDNA = data.graphicSofDNA
	}
	return nil
}

// openAndVerify opens dbPath and confirms it's a genuinely usable SDE
// database, not merely a valid-but-empty SQLite file -- sql.Open+Ping alone
// can't tell the difference, since SQLite happily creates a fresh, valid,
// empty database for any missing or zero-byte path (verified directly
// against modernc.org/sqlite). ok is false if the path doesn't exist, can't
// be opened, or fails the sentinel query; any *sql.DB opened along the way
// is closed before returning false, so a caller never leaks a handle to an
// unusable file.
func (s *Store) openAndVerify(path string) (*sql.DB, bool) {
	if _, err := os.Stat(path); err != nil {
		return nil, false
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, false
	}
	var name string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'invTypes'`).Scan(&name); err != nil {
		db.Close()
		return nil, false
	}
	return db, true
}

// downloadDump fetches and gunzips the fuzzwork dump, writing it to a
// process-unique temp path and atomically renaming it over s.dbPath --
// never leaves a partially-written db in place if the process dies
// mid-download. The temp path is unique per call (via os.CreateTemp), not a
// fixed name, so two neat processes racing to refresh a shared
// ~/.neat/sde/eve.db -- e.g. the Claude Code plugin and Claude Desktop
// extension both running "neat mcp" concurrently -- can't truncate or
// interleave-write into each other's in-progress download; each still
// renames independently over dbPath, so whichever finishes last "wins",
// same as before, but neither corrupts the other's data en route.
func (s *Store) downloadDump(ctx context.Context, userAgent string) error {
	body, err := fetchBytes(ctx, s.dumpURL, userAgent)
	if err != nil {
		return fmt.Errorf("download dump: %w", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	tmp, err := os.CreateTemp(filepath.Dir(s.dbPath), "eve.db.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp dump file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, gz); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("decompress dump: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp dump file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp dump file: %w", err)
	}
	if err := os.Rename(tmpPath, s.dbPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace dump file: %w", err)
	}
	return nil
}

// fetchRemoteVersion does a lightweight HEAD request to check a URL's
// current version (ETag, falling back to Last-Modified) without downloading
// its body.
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

// fetchReferenceDataTypeVolumes fetches EVE Ref's reference-data dataset and
// returns typeID -> packagedVolume for every type that has one. Returns nil
// (not an error) on any failure -- reference-data is a best-effort
// augmentation, never fatal to the rest of sync().
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

// esiScrapeData bundles the two values fetchEsiScrapeSupplements extracts
// from EVE Ref's esi-scrape dataset -- races.en-us.yaml's AllianceID and
// graphics.yaml's SofDNA. Every other resource this dataset used to supply
// (celestial bodies, systemPlanets) is now sourced directly from the fuzzwork
// dump's mapDenormalize/mapJumps/mapCelestialStatistics tables instead (see
// Tasks 6-7).
type esiScrapeData struct {
	raceAlliances map[int64]int64
	graphicSofDNA map[int64]string
}

type esiScrapeRace struct {
	AllianceID int64 `yaml:"alliance_id"`
	RaceID     int64 `yaml:"race_id"`
}

type esiScrapeGraphic struct {
	SofDNA *string `yaml:"sof_dna"`
}

// fetchEsiScrapeSupplements fetches EVE Ref's esi-scrape dataset. ok is true
// if the fetch/extract/parse succeeded overall (an individual missing member
// still leaves its own field nil within data) -- callers use ok to decide
// whether to keep the previous cycle's values or overwrite with (possibly
// nil) new ones.
func fetchEsiScrapeSupplements(ctx context.Context, url, userAgent string) (data esiScrapeData, ok bool) {
	body, err := fetchBytes(ctx, url, userAgent)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to fetch esi-scrape, skipping AllianceID/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}

	want := map[string]bool{
		esiScrapePathPrefix + "races.en-us.yaml": true,
		esiScrapePathPrefix + "graphics.yaml":    true,
	}
	members, err := extractTarXZMembers(body, want)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to extract esi-scrape, skipping AllianceID/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}

	data, err = parseEsiScrapeMembers(ctx, members)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to parse esi-scrape, skipping AllianceID/SofDNA augmentation", "error", err)
		return esiScrapeData{}, false
	}
	return data, true
}

// esiScrapeMemberPresent reports whether key was actually extracted from the
// archive, warning if not.
func esiScrapeMemberPresent(ctx context.Context, members map[string][]byte, key string) bool {
	if _, ok := members[key]; ok {
		return true
	}
	slog.Default().WarnContext(ctx, "sde: esi-scrape archive missing expected member, skipping its augmentation", "member", key)
	return false
}

func parseEsiScrapeMembers(ctx context.Context, members map[string][]byte) (esiScrapeData, error) {
	var data esiScrapeData

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

// extractTarXZMembers decompresses an XZ-compressed tar archive and returns
// the raw bytes of every member whose path is in want, keyed by that same
// path. Members not in want are skipped without being read into memory.
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
