package sde

import (
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
	"time"

	"go.etcd.io/bbolt"
)

const sdeZipName = "eve-online-static-data-latest-jsonl.zip"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// syncedFiles maps each JSONL file inside the SDE archive to the bbolt
// bucket its records are stored under.
var syncedFiles = map[string]string{
	"categories.jsonl": "categories",
	"groups.jsonl":     "groups",
	"types.jsonl":      "types",
}

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
// reloading categories/groups/types if it's stale or missing. A failed sync
// is cached too -- lookups fall back to whatever's already in bbolt (empty,
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

	currentVersion, err := s.version()
	if err != nil {
		return fmt.Errorf("read cached sde version: %w", err)
	}
	if currentVersion == remoteEtag {
		return nil
	}

	body, err := fetchBytes(ctx, zipURL, userAgent)
	if err != nil {
		return fmt.Errorf("download sde archive: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("open sde archive: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		for file, bucket := range syncedFiles {
			if err := loadResource(tx, zr, file, bucket); err != nil {
				return fmt.Errorf("load %s: %w", file, err)
			}
		}
		meta, err := tx.CreateBucketIfNotExists([]byte("_meta"))
		if err != nil {
			return fmt.Errorf("create _meta bucket: %w", err)
		}
		return meta.Put([]byte("version"), []byte(remoteEtag))
	})
}

// version returns the SDE version currently cached in bbolt's _meta bucket,
// or "" if the store has never been synced.
func (s *Store) version() (string, error) {
	var version string
	err := s.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte("_meta"))
		if meta == nil {
			return nil
		}
		version = string(meta.Get([]byte("version")))
		return nil
	})
	return version, err
}

// loadResource clears and rewrites bucket from file's JSONL lines within
// the zip, keyed by each record's _key field.
func loadResource(tx *bbolt.Tx, zr *zip.Reader, file, bucket string) error {
	f, err := zr.Open(file)
	if err != nil {
		return fmt.Errorf("open zip entry: %w", err)
	}
	defer f.Close()

	if err := tx.DeleteBucket([]byte(bucket)); err != nil && !errors.Is(err, bbolt.ErrBucketNotFound) {
		return fmt.Errorf("clear bucket: %w", err)
	}
	b, err := tx.CreateBucket([]byte(bucket))
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}

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
		if err := b.Put(int64Key(keyed.Key), append([]byte(nil), line...)); err != nil {
			return fmt.Errorf("write record %d: %w", keyed.Key, err)
		}
	}
	return scanner.Err()
}

// int64Key encodes id as a big-endian byte slice suitable for use as a
// bbolt key.
func int64Key(id int64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(id))
	return key
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
