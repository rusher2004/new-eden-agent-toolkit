package sde

import (
	"context"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
)

// seedStore opens a fresh Store at a temp path and marks its sync as
// already complete (successfully, doing nothing), so accessor tests can
// pre-seed buckets directly without any network path ever running.
func seedStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "sde.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	store.syncOnce.Do(func() {})
	return store
}

func putRecord(t *testing.T, store *Store, bucket string, id int64, jsonLine string) {
	t.Helper()
	err := store.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put(int64Key(id), []byte(jsonLine))
	})
	if err != nil {
		t.Fatalf("seed %s record %d: %v", bucket, id, err)
	}
}

func TestCategory_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "categories", 6,
		`{"_key": 6, "name": {"en": "Ship", "de": "Schiff", "ja": "船"}, "published": true}`)

	got, ok, err := store.Category(context.Background(), "test-agent", 6)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false, want true")
	}
	if got.ID != 6 || !got.Published {
		t.Errorf("got = %+v, want ID=6 Published=true", got)
	}
	if got.Name["en"] != "Ship" || got.Name["de"] != "Schiff" || got.Name["ja"] != "船" {
		t.Errorf("Name = %+v, want en=Ship de=Schiff ja=船", got.Name)
	}
}

func TestCategory_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Category(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if ok {
		t.Error("Category ok = true, want false for an unknown ID")
	}
}

func TestCategory_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "categories", 6, `not valid json`)

	_, _, err := store.Category(context.Background(), "test-agent", 6)
	if err == nil {
		t.Fatal("Category returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestCategory_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	// Stored under bbolt key 6, but the record's own _key field says 999 --
	// simulates a corrupted/mis-keyed cache entry.
	putRecord(t, store, "categories", 6,
		`{"_key": 999, "name": {"en": "Ship"}, "published": true}`)

	_, _, err := store.Category(context.Background(), "test-agent", 6)
	if err == nil {
		t.Fatal("Category returned nil error, want an error for a mismatched _key")
	}
}
