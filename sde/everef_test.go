package sde

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// eveRefServer stands in for one EVE Ref dataset: /index.json lists
// /<archive> with the current etag, and GET /<archive> serves body and
// counts downloads. Setting etag to "" makes the index request fail.
type eveRefServer struct {
	srv       *httptest.Server
	archive   string
	etag      atomic.Value // string
	downloads atomic.Int32
	failGet   atomic.Bool
}

func newEveRefServer(t *testing.T, archive, etag string, body []byte) *eveRefServer {
	t.Helper()
	e := &eveRefServer{archive: archive}
	e.etag.Store(etag)
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			etag := e.etag.Load().(string)
			if etag == "" {
				http.Error(w, "index unavailable", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, `{"files":[{"name":%q,"etag":%q,"size":1}],"path":"x"}`, archive, etag)
		case "/" + archive:
			if e.failGet.Load() {
				http.Error(w, "archive unavailable", http.StatusInternalServerError)
				return
			}
			e.downloads.Add(1)
			w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(e.srv.Close)
	return e
}

func (e *eveRefServer) archiveURL() string { return e.srv.URL + "/" + e.archive }
func (e *eveRefServer) indexURL() string   { return e.srv.URL + "/index.json" }

// supplementsStore returns a func building Stores that share one cache
// directory, each pointed at the same fake dump and EVE Ref servers, the way
// successive neat processes share ~/.neat/sde.
func supplementsStore(t *testing.T) (newStore func() *Store, ref, scrape *eveRefServer) {
	t.Helper()
	dumpBody := buildDumpGzip(t)
	dumpSrv := serveBody(t, `"dump-etag-1"`, dumpBody)
	ref = newEveRefServer(t, "reference-data-latest.tar.xz", "ref-1", buildReferenceDataArchive(t))
	scrape = newEveRefServer(t, "eve-ref-esi-scrape-latest.tar.xz", "scrape-1", buildEsiScrapeArchive(t))
	dbPath := filepath.Join(t.TempDir(), "eve.db")
	newStore = func() *Store {
		s := &Store{
			dbPath: dbPath, metaPath: filepath.Join(filepath.Dir(dbPath), "meta.json"),
			dumpURL:          dumpSrv.URL,
			referenceDataURL: ref.archiveURL(), referenceDataIndexURL: ref.indexURL(),
			esiScrapeURL: scrape.archiveURL(), esiScrapeIndexURL: scrape.indexURL(),
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	return newStore, ref, scrape
}

// runSync syncs a fresh Store and closes it, as one neat process would.
func runSync(t *testing.T, newStore func() *Store) *Store {
	t.Helper()
	s := newStore()
	if err := s.sync(context.Background(), testUserAgent); err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	s.Close()
	return s
}

func assertSupplements(t *testing.T, s *Store) {
	t.Helper()
	if r, ok := s.typeRadii[587]; !ok || r != 31.0 {
		t.Errorf("typeRadii[587] = %v, %v, want 31, true", r, ok)
	}
	if a, ok := s.raceAlliances[1]; !ok || a != 500001 {
		t.Errorf("raceAlliances[1] = %v, %v, want 500001, true", a, ok)
	}
	if dna, ok := s.graphicSofDNA[2]; !ok || dna != "caldari:caldari:cal" {
		t.Errorf("graphicSofDNA[2] = %v, %v, want caldari:caldari:cal, true", dna, ok)
	}
}

func TestSupplements_UnchangedETag_UsesSavedCopy(t *testing.T) {
	newStore, ref, scrape := supplementsStore(t)
	runSync(t, newStore)
	second := runSync(t, newStore)

	if got := ref.downloads.Load(); got != 1 {
		t.Errorf("reference-data downloaded %d times, want 1", got)
	}
	if got := scrape.downloads.Load(); got != 1 {
		t.Errorf("esi-scrape downloaded %d times, want 1", got)
	}
	assertSupplements(t, second)
}

func TestSupplements_ChangedETag_DownloadsOnlyThatArchive(t *testing.T) {
	newStore, ref, scrape := supplementsStore(t)
	runSync(t, newStore)
	ref.etag.Store("ref-2")
	second := runSync(t, newStore)

	if got := ref.downloads.Load(); got != 2 {
		t.Errorf("reference-data downloaded %d times, want 2 after its etag changed", got)
	}
	if got := scrape.downloads.Load(); got != 1 {
		t.Errorf("esi-scrape downloaded %d times, want 1 (its etag didn't change)", got)
	}
	assertSupplements(t, second)

	// The new etag was saved, so a third run downloads nothing.
	runSync(t, newStore)
	if got := ref.downloads.Load(); got != 2 {
		t.Errorf("reference-data downloaded %d times after a third run, want 2", got)
	}
}

func TestSupplements_IndexUnavailable_DownloadsArchive(t *testing.T) {
	newStore, ref, scrape := supplementsStore(t)
	runSync(t, newStore)
	ref.etag.Store("")
	scrape.etag.Store("")
	second := runSync(t, newStore)

	if got := ref.downloads.Load(); got != 2 {
		t.Errorf("reference-data downloaded %d times, want 2 when the index can't be read", got)
	}
	if got := scrape.downloads.Load(); got != 2 {
		t.Errorf("esi-scrape downloaded %d times, want 2 when the index can't be read", got)
	}
	assertSupplements(t, second)
}

func TestSupplements_DownloadFails_KeepsSavedCopy(t *testing.T) {
	newStore, ref, scrape := supplementsStore(t)
	runSync(t, newStore)
	ref.etag.Store("ref-2")
	scrape.etag.Store("scrape-2")
	ref.failGet.Store(true)
	scrape.failGet.Store(true)
	second := runSync(t, newStore)

	assertSupplements(t, second)

	// The failed download didn't overwrite the saved etag, so once the
	// archives are reachable again the next run fetches the new versions.
	ref.failGet.Store(false)
	scrape.failGet.Store(false)
	runSync(t, newStore)
	if got := ref.downloads.Load(); got != 2 {
		t.Errorf("reference-data downloaded %d times, want 2", got)
	}
	if got := scrape.downloads.Load(); got != 2 {
		t.Errorf("esi-scrape downloaded %d times, want 2", got)
	}
}

func TestSupplements_CorruptSavedFile_DownloadsAgain(t *testing.T) {
	newStore, ref, _ := supplementsStore(t)
	first := runSync(t, newStore)
	if err := os.WriteFile(first.supplementsPath(), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("corrupt saved file: %v", err)
	}
	second := runSync(t, newStore)

	if got := ref.downloads.Load(); got != 2 {
		t.Errorf("reference-data downloaded %d times, want 2 after the saved file was corrupt", got)
	}
	assertSupplements(t, second)
}

func TestSupplements_OtherVersionSavedFile_DownloadsAgain(t *testing.T) {
	newStore, ref, _ := supplementsStore(t)
	first := runSync(t, newStore)
	if err := os.WriteFile(first.supplementsPath(), []byte(`{"version":"0","referenceData":{"etag":"ref-1","typeRadii":{"587":1}}}`), 0o600); err != nil {
		t.Fatalf("write old-version file: %v", err)
	}
	second := runSync(t, newStore)

	if got := ref.downloads.Load(); got != 2 {
		t.Errorf("reference-data downloaded %d times, want 2 for a saved file from another version", got)
	}
	assertSupplements(t, second)
}
