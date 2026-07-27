package sde

import (
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

	"go.etcd.io/bbolt"
)

const testUserAgent = "neat-test/0.1"

func buildFixtureZip(t *testing.T) []byte {
	t.Helper()
	return buildZip(t, map[string][]string{
		"categories.jsonl": {
			`{"_key": 6, "name": {"en": "Ship", "de": "Schiff"}, "published": true}`,
		},
		"groups.jsonl": {
			`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate", "de": "Fregatte"}, "published": true}`,
		},
		"types.jsonl": {
			`{"_key": 587, "groupID": 25, "name": {"en": "Rifter", "de": "Rifter"}, "published": true}`,
		},
	})
}

func buildFixtureZipWithRenamedShipCategory(t *testing.T) []byte {
	t.Helper()
	return buildZip(t, map[string][]string{
		"categories.jsonl": {
			`{"_key": 6, "name": {"en": "Ship Hull", "de": "Schiffsrumpf"}, "published": true}`,
		},
		"groups.jsonl": {
			`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate", "de": "Fregatte"}, "published": true}`,
		},
		"types.jsonl": {
			`{"_key": 587, "groupID": 25, "name": {"en": "Rifter", "de": "Rifter"}, "published": true}`,
		},
	})
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

// fixtureServer serves an index.json (pointing back at its own /sde.zip)
// and the zip body itself, counting hits to each so tests can assert
// exactly how many times each endpoint was actually called.
type fixtureServer struct {
	*httptest.Server
	indexHits atomic.Int32
	zipHits   atomic.Int32
	etag      string
	zipBody   []byte
}

func newFixtureServer(t *testing.T, etag string, zipBody []byte) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{etag: etag, zipBody: zipBody}
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
	fs.Server = httptest.NewServer(mux)
	t.Cleanup(fs.Close)
	return fs
}

// openWithIndexURL opens a fresh Store at a temp path with its indexURL
// overridden to point at a test fixture server instead of the real EVE Ref
// endpoint. Direct field access, now that this test file is in-package --
// no exported test-only seam needed.
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

// openAtPathWithIndexURL is like openWithIndexURL but reuses an explicit
// db path, for tests that reopen the same file as a second Store.
func openAtPathWithIndexURL(t *testing.T, path, indexURL string) *Store {
	t.Helper()
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	store.indexURL = indexURL
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

func TestSync_FirstSyncLoadsAllThreeResources(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("ensureSynced returned error: %v", err)
	}

	assertRecord(t, store, "categories", 6, "Ship", true)
	assertRecord(t, store, "groups", 25, "Frigate", true)
	assertRecord(t, store, "types", 587, "Rifter", true)

	if srv.zipHits.Load() != 1 {
		t.Errorf("zip requested %d times, want exactly 1", srv.zipHits.Load())
	}
}

func TestSync_OnceOnlyPerStoreLifetime(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

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
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

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

	first := openAtPathWithIndexURL(t, dbPath, srv.Server.URL+"/index.json")
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}
	if srv.zipHits.Load() != 1 {
		t.Fatalf("zip requested %d times on first sync, want exactly 1", srv.zipHits.Load())
	}

	// A second Store opened against the same, already-synced db file, with
	// the index still reporting the same etag, must not re-download.
	second := openAtPathWithIndexURL(t, dbPath, srv.Server.URL+"/index.json")
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
	first := openAtPathWithIndexURL(t, dbPath, srv1.Server.URL+"/index.json")
	if err := first.ensureSynced(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close returned error: %v", err)
	}

	srv2 := newFixtureServer(t, "etag-v2", buildFixtureZipWithRenamedShipCategory(t))
	second := openAtPathWithIndexURL(t, dbPath, srv2.Server.URL+"/index.json")
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
	first := openAtPathWithIndexURL(t, dbPath, srv.Server.URL+"/index.json")
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

	// The failed sync must not have touched the data a prior, successful
	// sync already wrote.
	assertRecord(t, second, "categories", 6, "Ship", true)
}

func TestSync_EmptyRemoteEtagReturnsError(t *testing.T) {
	// A remote etag of "" must not be treated as "matches the never-synced
	// currentVersion of \"\"" -- that would silently disable syncing forever.
	srv := newFixtureServer(t, "", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error for an empty remote etag")
	}
	if srv.zipHits.Load() != 0 {
		t.Errorf("zip requested %d times, want 0 (should fail before attempting a download)", srv.zipHits.Load())
	}
}

func TestSync_FailedResourceLoadRollsBackWholeTransaction(t *testing.T) {
	// categories/groups are valid so they'd be staged in the same bbolt
	// transaction if processed before types (map iteration order is
	// randomized) -- types.jsonl's bad line must cause the whole
	// transaction, including any already-staged writes, to roll back.
	badZip := buildZip(t, map[string][]string{
		"categories.jsonl": {
			`{"_key": 6, "name": {"en": "Ship"}, "published": true}`,
		},
		"groups.jsonl": {
			`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true}`,
		},
		"types.jsonl": {
			`not valid json`,
		},
	})
	srv := newFixtureServer(t, "etag-v1", badZip)
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

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
	for _, bucket := range []string{"categories", "groups", "types"} {
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
	incompleteZip := buildZip(t, map[string][]string{
		"categories.jsonl": {
			`{"_key": 6, "name": {"en": "Ship"}, "published": true}`,
		},
		"groups.jsonl": {
			`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true}`,
		},
		// types.jsonl intentionally omitted.
	})
	srv := newFixtureServer(t, "etag-v1", incompleteZip)
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

	if err := store.ensureSynced(context.Background(), testUserAgent); err == nil {
		t.Fatal("ensureSynced returned nil error, want an error when the archive is missing an expected JSONL file")
	}
}

func TestCategory_TriggersRealSyncOnNeverSyncedStore(t *testing.T) {
	srv := newFixtureServer(t, "etag-v1", buildFixtureZip(t))
	store := openWithIndexURL(t, srv.Server.URL+"/index.json")

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
