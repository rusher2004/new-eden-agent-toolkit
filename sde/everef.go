package sde

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

// EVE Ref publishes an index.json beside each dataset listing every file with
// its etag. The archives themselves can't be version-checked directly: as of
// 2026-11-19 data.everef.net (Cloudflare) sends no ETag header, and its
// Last-Modified is the edge cache's fill time, which differs between
// requests. The index's etag is the origin's own and only changes when the
// archive does.
const (
	referenceDataIndexURL = "https://data.everef.net/reference-data/index.json"
	esiScrapeIndexURL     = "https://data.everef.net/esi-scrape/index.json"
)

// supplementsVersion identifies the shape of supplementsFile. Bump it when
// supplementsCache's fields or their meaning change, so an existing file is
// ignored and rebuilt rather than misread.
const supplementsVersion = "1"

// supplementsFile is the name, within the sde cache directory, of the file
// holding the values extracted from EVE Ref's archives.
const supplementsFile = "everef.json"

// supplementsCache is what's persisted to supplementsFile: the few values
// neat uses from each EVE Ref archive, plus the archive's etag when they
// were extracted, so a later process can skip downloading an unchanged
// archive. Either entry is nil if that archive has never been fetched
// successfully.
type supplementsCache struct {
	Version       string              `json:"version"`
	ReferenceData *referenceDataCache `json:"referenceData,omitempty"`
	EsiScrape     *esiScrapeCache     `json:"esiScrape,omitempty"`
}

type referenceDataCache struct {
	// ETag is "" when the index couldn't be read at download time, which
	// never matches, so the next sync downloads again.
	ETag      string            `json:"etag"`
	TypeRadii map[int64]float64 `json:"typeRadii"`
}

type esiScrapeCache struct {
	ETag          string           `json:"etag"`
	RaceAlliances map[int64]int64  `json:"raceAlliances"`
	GraphicSofDNA map[int64]string `json:"graphicSofDNA"`
}

// syncSupplements sets typeRadii, raceAlliances and graphicSofDNA from EVE
// Ref, downloading an archive only when its etag in EVE Ref's index differs
// from the cached one (or either is unknown). Best-effort throughout: any
// failure logs a warning and falls back to the cached values, or leaves the
// field nil when nothing is cached.
func (s *Store) syncSupplements(ctx context.Context, userAgent string) {
	cache := s.readSupplements(ctx)

	ref, refFresh := s.syncReferenceData(ctx, userAgent, cache.ReferenceData)
	scrape, scrapeFresh := s.syncEsiScrape(ctx, userAgent, cache.EsiScrape)

	if ref != nil {
		s.typeRadii = ref.TypeRadii
	}
	if scrape != nil {
		s.raceAlliances = scrape.RaceAlliances
		s.graphicSofDNA = scrape.GraphicSofDNA
	}

	if refFresh || scrapeFresh {
		next := supplementsCache{Version: supplementsVersion, ReferenceData: ref, EsiScrape: scrape}
		if err := s.writeSupplements(next); err != nil {
			slog.Default().WarnContext(ctx, "sde: failed to save EVE Ref data, it will be downloaded again next time", "error", err)
		}
	}
}

// syncReferenceData returns the reference-data values to use and whether
// they were freshly downloaded (and so need saving).
func (s *Store) syncReferenceData(ctx context.Context, userAgent string, cached *referenceDataCache) (*referenceDataCache, bool) {
	cachedETag := ""
	if cached != nil {
		cachedETag = cached.ETag
	}
	remote, current := checkSupplement(ctx, "reference-data", s.referenceDataIndexURL, s.referenceDataURL, userAgent, cachedETag)
	if current {
		return cached, false
	}
	radii := fetchReferenceDataTypeRadii(ctx, s.referenceDataURL, userAgent)
	if radii == nil {
		if cached != nil {
			slog.Default().WarnContext(ctx, "sde: using previously saved reference-data")
		}
		return cached, false
	}
	return &referenceDataCache{ETag: remote, TypeRadii: radii}, true
}

// syncEsiScrape returns the esi-scrape values to use and whether they were
// freshly downloaded (and so need saving).
func (s *Store) syncEsiScrape(ctx context.Context, userAgent string, cached *esiScrapeCache) (*esiScrapeCache, bool) {
	cachedETag := ""
	if cached != nil {
		cachedETag = cached.ETag
	}
	remote, current := checkSupplement(ctx, "esi-scrape", s.esiScrapeIndexURL, s.esiScrapeURL, userAgent, cachedETag)
	if current {
		return cached, false
	}
	data, ok := fetchEsiScrapeSupplements(ctx, s.esiScrapeURL, userAgent)
	if !ok {
		if cached != nil {
			slog.Default().WarnContext(ctx, "sde: using previously saved esi-scrape data")
		}
		return cached, false
	}
	return &esiScrapeCache{ETag: remote, RaceAlliances: data.raceAlliances, GraphicSofDNA: data.graphicSofDNA}, true
}

// checkSupplement looks up archiveURL's etag in EVE Ref's index at indexURL.
// current is true only when that etag is known and equals cachedETag. A
// failed lookup returns ("", false), so the caller downloads the archive as
// if no version check existed.
func checkSupplement(ctx context.Context, name, indexURL, archiveURL, userAgent, cachedETag string) (remote string, current bool) {
	remote, err := fetchEveRefETag(ctx, indexURL, archiveURL, userAgent)
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: couldn't check EVE Ref's index, downloading the archive", "dataset", name, "error", err)
		return "", false
	}
	return remote, cachedETag != "" && cachedETag == remote
}

// eveRefIndex is the part of an EVE Ref index.json that neat reads.
type eveRefIndex struct {
	Files []struct {
		Name string `json:"name"`
		ETag string `json:"etag"`
	} `json:"files"`
}

// fetchEveRefETag returns the etag EVE Ref's index at indexURL lists for the
// file archiveURL names, matched by its last path segment.
func fetchEveRefETag(ctx context.Context, indexURL, archiveURL, userAgent string) (string, error) {
	if indexURL == "" {
		return "", errors.New("no index URL configured")
	}
	u, err := url.Parse(archiveURL)
	if err != nil {
		return "", fmt.Errorf("parse archive URL: %w", err)
	}
	name := path.Base(u.Path)

	body, err := fetchBytes(ctx, indexURL, userAgent)
	if err != nil {
		return "", fmt.Errorf("fetch index: %w", err)
	}
	var idx eveRefIndex
	if err := json.Unmarshal(body, &idx); err != nil {
		return "", fmt.Errorf("parse index: %w", err)
	}
	for _, f := range idx.Files {
		if f.Name == name {
			if f.ETag == "" {
				return "", fmt.Errorf("index lists %s with no etag", name)
			}
			return f.ETag, nil
		}
	}
	return "", fmt.Errorf("index doesn't list %s", name)
}

func (s *Store) supplementsPath() string {
	return filepath.Join(filepath.Dir(s.dbPath), supplementsFile)
}

// readSupplements returns the saved EVE Ref values, or an empty cache when
// the file is missing, unreadable, or from a different supplementsVersion.
func (s *Store) readSupplements(ctx context.Context) supplementsCache {
	data, err := os.ReadFile(s.supplementsPath())
	if errors.Is(err, os.ErrNotExist) {
		return supplementsCache{}
	}
	if err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to read saved EVE Ref data, downloading it again", "error", err)
		return supplementsCache{}
	}
	var c supplementsCache
	if err := json.Unmarshal(data, &c); err != nil {
		slog.Default().WarnContext(ctx, "sde: failed to parse saved EVE Ref data, downloading it again", "error", err)
		return supplementsCache{}
	}
	if c.Version != supplementsVersion {
		return supplementsCache{}
	}
	return c
}

// writeSupplements saves c via a temp file and rename, so a process that
// dies mid-write, or two neat processes saving at once, never leave a
// half-written file behind.
func (s *Store) writeSupplements(c supplementsCache) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal EVE Ref data: %w", err)
	}
	dst := s.supplementsPath()
	tmp, err := os.CreateTemp(filepath.Dir(dst), supplementsFile+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace %s: %w", supplementsFile, err)
	}
	return nil
}
