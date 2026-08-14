package sde

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz"
)

const testUserAgent = "neat-test/0.1"

// buildDumpGzip builds a minimal valid sqlite file (testSchema, no rows) and
// gzips it, standing in for fuzzwork's latest-sqlite.db.gz.
func buildDumpGzip(t *testing.T) []byte {
	t.Helper()
	srcPath := filepath.Join(t.TempDir(), "src.db")
	db, err := sql.Open("sqlite", srcPath)
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("apply schema to source db: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`); err != nil {
		t.Fatalf("seed source db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close source db: %v", err)
	}
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read source db: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		t.Fatalf("gzip source db: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// buildEsiScrapeArchive builds a minimal esi-scrape tar.xz containing only
// the two members sync() still reads (races.en-us.yaml, graphics.yaml).
func buildEsiScrapeArchive(t *testing.T) []byte {
	t.Helper()
	files := map[string]string{
		esiScrapePathPrefix + "races.en-us.yaml": "- race_id: 1\n  alliance_id: 500001\n",
		esiScrapePathPrefix + "graphics.yaml":    "\"2\":\n  sof_dna: \"caldari:caldari:cal\"\n",
	}

	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0o600}); err != nil {
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

// testServers wires up two httptest servers standing in for fuzzwork's dump
// and EVE Ref's esi-scrape export, and returns a Store pointed at them plus
// the dump's ETag.
func testServers(t *testing.T) (store *Store, dumpEtag string) {
	t.Helper()

	dumpBody := buildDumpGzip(t)
	dumpEtag = `"dump-etag-1"`
	dumpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", dumpEtag)
		if r.Method == http.MethodHead {
			return
		}
		w.Write(dumpBody)
	}))
	t.Cleanup(dumpSrv.Close)

	scrapeBody := buildEsiScrapeArchive(t)
	scrapeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"scrape-etag-1"`)
		if r.Method == http.MethodHead {
			return
		}
		w.Write(scrapeBody)
	}))
	t.Cleanup(scrapeSrv.Close)

	dbPath := filepath.Join(t.TempDir(), "eve.db")
	store = &Store{
		dbPath: dbPath, metaPath: filepath.Join(filepath.Dir(dbPath), "meta.json"),
		dumpURL: dumpSrv.URL, esiScrapeURL: scrapeSrv.URL,
	}
	// A leftover open *sql.DB handle blocks a later rename/delete of the same
	// file on Windows (unlike POSIX, which allows unlinking/renaming an
	// open file) -- every Store this helper hands out must be closed by the
	// test that uses it, including before constructing any "second process"
	// Store pointed at the same dbPath.
	t.Cleanup(func() { store.Close() })
	return store, dumpEtag
}

func TestSync_FirstRun_DownloadsDumpAndAugmentation(t *testing.T) {
	store, dumpEtag := testServers(t)

	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}

	if store.db == nil {
		t.Fatal("sync did not open a db connection")
	}
	var name string
	if err := store.db.QueryRow(`SELECT typeName FROM invTypes WHERE typeID = 587`).Scan(&name); err != nil {
		t.Fatalf("query downloaded db: %v", err)
	}
	if name != "Rifter" {
		t.Errorf("typeName = %q, want Rifter", name)
	}

	if a, ok := store.raceAlliances[1]; !ok || a != 500001 {
		t.Errorf("raceAlliances[1] = %v, %v, want 500001, true", a, ok)
	}
	if dna, ok := store.graphicSofDNA[2]; !ok || dna != "caldari:caldari:cal" {
		t.Errorf("graphicSofDNA[2] = %v, %v, want caldari:caldari:cal, true", dna, ok)
	}

	meta, err := store.readMeta()
	if err != nil {
		t.Fatalf("readMeta returned error: %v", err)
	}
	if meta.DumpVersion != dumpEtag || meta.SchemaVersion != schemaVersion {
		t.Errorf("meta = %+v, want DumpVersion=%s SchemaVersion=%s", meta, dumpEtag, schemaVersion)
	}
}

func TestSync_UnchangedDump_SkipsRedownload(t *testing.T) {
	store, dumpEtag := testServers(t)
	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	firstInfo, err := os.Stat(store.dbPath)
	if err != nil {
		t.Fatalf("stat db after first sync: %v", err)
	}

	// A second Store pointed at the same dbPath/metaPath, simulating a new
	// process starting up with an already-synced cache on disk -- close the
	// first store first, exactly as a real prior process would have exited
	// (and released its file handle) before this one starts.
	store.Close()
	second := &Store{
		dbPath: store.dbPath, metaPath: store.metaPath,
		dumpURL: store.dumpURL, esiScrapeURL: store.esiScrapeURL,
	}
	t.Cleanup(func() { second.Close() })
	if err := second.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}
	secondInfo, err := os.Stat(second.dbPath)
	if err != nil {
		t.Fatalf("stat db after second sync: %v", err)
	}
	if !firstInfo.ModTime().Equal(secondInfo.ModTime()) {
		t.Error("db file was rewritten on an unchanged-ETag sync, want it left untouched")
	}
	if second.db == nil {
		t.Error("second sync did not open a db connection to the existing file")
	}

	_ = dumpEtag
}

func TestSync_MissingDBFile_ReDownloadsDespiteUnchangedMeta(t *testing.T) {
	store, _ := testServers(t)
	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}

	// Simulate a user deleting eve.db to reclaim disk space while meta.json
	// survives untouched -- the desync case that used to leave sync()
	// thinking nothing changed, skipping the download, and opening a fresh,
	// table-less sqlite file at s.dbPath instead. Close store first -- an
	// open handle would block the delete on Windows the same way it'd block
	// a rename, and a real user reclaiming disk space implies the process
	// that had the file open already exited.
	store.Close()
	if err := os.Remove(store.dbPath); err != nil {
		t.Fatalf("remove db file: %v", err)
	}

	second := &Store{
		dbPath: store.dbPath, metaPath: store.metaPath,
		dumpURL: store.dumpURL, esiScrapeURL: store.esiScrapeURL,
	}
	t.Cleanup(func() { second.Close() })
	if err := second.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}
	if second.db == nil {
		t.Fatal("second sync did not open a db connection")
	}

	var name string
	if err := second.db.QueryRow(`SELECT typeName FROM invTypes WHERE typeID = 587`).Scan(&name); err != nil {
		t.Fatalf("query re-downloaded db: %v", err)
	}
	if name != "Rifter" {
		t.Errorf("typeName = %q, want Rifter", name)
	}
}

func TestSync_ChangedETag_Redownloads(t *testing.T) {
	// TestSync_UnchangedDump_SkipsRedownload covers "ETag unchanged, skip";
	// this covers the other half of the freshness mechanism -- a genuinely
	// new remote ETag must trigger a real re-download, not just avoid
	// re-downloading forever. See PR #14 review suggestions.
	firstBody := buildDumpGzip(t) // seeds typeID 587 "Rifter"
	secondSrcPath := filepath.Join(t.TempDir(), "second.db")
	secondDB, err := sql.Open("sqlite", secondSrcPath)
	if err != nil {
		t.Fatalf("open second source db: %v", err)
	}
	if _, err := secondDB.Exec(testSchema); err != nil {
		t.Fatalf("apply schema to second source db: %v", err)
	}
	if _, err := secondDB.Exec(`INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (588, 25, 'Slasher', 1)`); err != nil {
		t.Fatalf("seed second source db: %v", err)
	}
	if err := secondDB.Close(); err != nil {
		t.Fatalf("close second source db: %v", err)
	}
	secondRaw, err := os.ReadFile(secondSrcPath)
	if err != nil {
		t.Fatalf("read second source db: %v", err)
	}
	var secondBuf bytes.Buffer
	gz := gzip.NewWriter(&secondBuf)
	if _, err := gz.Write(secondRaw); err != nil {
		t.Fatalf("gzip second source db: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	secondBody := secondBuf.Bytes()

	etag := `"dump-etag-1"`
	dumpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", etag)
		if r.Method == http.MethodHead {
			return
		}
		if etag == `"dump-etag-1"` {
			w.Write(firstBody)
		} else {
			w.Write(secondBody)
		}
	}))
	t.Cleanup(dumpSrv.Close)
	scrapeBody := buildEsiScrapeArchive(t)
	scrapeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"scrape-etag-1"`)
		if r.Method != http.MethodHead {
			w.Write(scrapeBody)
		}
	}))
	t.Cleanup(scrapeSrv.Close)

	dbPath := filepath.Join(t.TempDir(), "eve.db")
	store := &Store{
		dbPath: dbPath, metaPath: filepath.Join(filepath.Dir(dbPath), "meta.json"),
		dumpURL: dumpSrv.URL, esiScrapeURL: scrapeSrv.URL,
	}
	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	var name string
	if err := store.db.QueryRow(`SELECT typeName FROM invTypes WHERE typeID = 587`).Scan(&name); err != nil {
		t.Fatalf("query db after first sync: %v", err)
	}
	if name != "Rifter" {
		t.Fatalf("typeName = %q, want Rifter after first sync", name)
	}

	// The remote ETag genuinely changes -- a new dump was published. Close
	// store first: a real prior process would have exited (releasing its
	// file handle) before a new one starts and finds the fresh ETag, and an
	// open handle would block the rename below on Windows regardless.
	store.Close()
	etag = `"dump-etag-2"`
	second := &Store{
		dbPath: store.dbPath, metaPath: store.metaPath,
		dumpURL: store.dumpURL, esiScrapeURL: store.esiScrapeURL,
	}
	t.Cleanup(func() { second.Close() })
	if err := second.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	var count int
	if err := second.db.QueryRow(`SELECT COUNT(*) FROM invTypes WHERE typeID = 587`).Scan(&count); err != nil {
		t.Fatalf("query old row after second sync: %v", err)
	}
	if count != 0 {
		t.Error("typeID 587 still present after a changed-ETag re-download, want the old dump's content replaced")
	}
	if err := second.db.QueryRow(`SELECT typeName FROM invTypes WHERE typeID = 588`).Scan(&name); err != nil {
		t.Fatalf("query new row after second sync: %v", err)
	}
	if name != "Slasher" {
		t.Errorf("typeName = %q, want Slasher (the new dump's content)", name)
	}

	meta, err := second.readMeta()
	if err != nil {
		t.Fatalf("readMeta returned error: %v", err)
	}
	if meta.DumpVersion != etag {
		t.Errorf("meta.DumpVersion = %q, want %q", meta.DumpVersion, etag)
	}
}

func TestSync_DumpFetchFails_LeavesDBNil(t *testing.T) {
	store := &Store{
		dbPath: filepath.Join(t.TempDir(), "eve.db"), metaPath: filepath.Join(t.TempDir(), "meta.json"),
		dumpURL: "http://127.0.0.1:0/does-not-exist", esiScrapeURL: "http://127.0.0.1:0/does-not-exist",
	}
	t.Cleanup(func() { store.Close() })

	err := store.sync(context.Background(), testUserAgent)
	if err == nil {
		t.Fatal("sync returned nil error, want an error when the dump can't be fetched")
	}
	if store.db != nil {
		t.Error("db was opened despite a failed dump fetch")
	}
}

func TestSync_CorruptDBFile_ReDownloadsAndSelfHeals(t *testing.T) {
	store, _ := testServers(t)
	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}

	// Simulate a corrupt/truncated eve.db that still exists on disk (unlike
	// TestSync_MissingDBFile_ReDownloadsDespiteUnchangedMeta, which deletes
	// the file outright) -- sql.Open+Ping alone can't tell this apart from a
	// real database, since SQLite creates a valid empty schema for any
	// zero-byte file. Close store first: an open handle would block the
	// rewrite below on Windows, and this simulates the file being corrupted
	// (e.g. by a crash) after the process that had it open is already gone.
	store.Close()
	if err := os.WriteFile(store.dbPath, []byte("not a real sqlite file"), 0o600); err != nil {
		t.Fatalf("corrupt db file: %v", err)
	}

	second := &Store{
		dbPath: store.dbPath, metaPath: store.metaPath,
		dumpURL: store.dumpURL, esiScrapeURL: store.esiScrapeURL,
	}
	t.Cleanup(func() { second.Close() })
	if err := second.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}
	if second.db == nil {
		t.Fatal("second sync did not open a db connection")
	}

	var name string
	if err := second.db.QueryRow(`SELECT typeName FROM invTypes WHERE typeID = 587`).Scan(&name); err != nil {
		t.Fatalf("query re-downloaded db: %v", err)
	}
	if name != "Rifter" {
		t.Errorf("typeName = %q, want Rifter", name)
	}
}

func TestSync_VersionCheckFailsWithExistingCache_ContinuesServingIt(t *testing.T) {
	store, _ := testServers(t)
	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}

	// A second Store pointed at the same already-synced files, but with a
	// dumpURL that can never be reached -- simulating a transient network
	// failure (a HEAD timeout, a DNS blip) on a process that already has a
	// perfectly good cache on disk. This must not leave db nil for the rest
	// of the process's life. Close store first (a real new process
	// wouldn't inherit the prior one's open handle), then let second open
	// its own connection to the same, untouched file.
	store.Close()
	second := &Store{
		dbPath: store.dbPath, metaPath: store.metaPath,
		dumpURL: "http://127.0.0.1:0/unreachable", esiScrapeURL: store.esiScrapeURL,
	}
	t.Cleanup(func() { second.Close() })
	if err := second.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("sync returned error: %v, want nil (should fall back to the existing cache)", err)
	}
	if second.db == nil {
		t.Fatal("sync left db nil despite a usable pre-existing cache on disk")
	}

	var name string
	if err := second.db.QueryRow(`SELECT typeName FROM invTypes WHERE typeID = 587`).Scan(&name); err != nil {
		t.Fatalf("query existing cache: %v", err)
	}
	if name != "Rifter" {
		t.Errorf("typeName = %q, want Rifter", name)
	}
}

func TestSync_EsiScrapeFetchFails_DumpStillSucceeds(t *testing.T) {
	store, _ := testServers(t)
	store.esiScrapeURL = "http://127.0.0.1:0/does-not-exist"

	if err := store.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	if store.db == nil {
		t.Error("dump sync did not proceed despite esi-scrape fetch failure")
	}
	if store.raceAlliances != nil {
		t.Errorf("raceAlliances = %v, want nil after a failed esi-scrape fetch", store.raceAlliances)
	}
	if store.graphicSofDNA != nil {
		t.Errorf("graphicSofDNA = %v, want nil after a failed esi-scrape fetch", store.graphicSofDNA)
	}
}
