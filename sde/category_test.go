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

func TestCategory_KnownID_ReturnsGroupIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "categories", 6,
		`{"_key": 6, "name": {"en": "Ship"}, "published": true, "groupIds": [25, 26]}`)

	got, ok, err := store.Category(context.Background(), "test-agent", 6)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false, want true")
	}
	if len(got.GroupIDs) != 2 || got.GroupIDs[0] != 25 || got.GroupIDs[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26]", got.GroupIDs)
	}
}

func TestCategoryIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "categories", 6, `{"_key": 6, "name": {"en": "Ship"}, "published": true}`)
	putRecord(t, store, "categories", 4, `{"_key": 4, "name": {"en": "Material"}, "published": true}`)

	ids, err := store.CategoryIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("CategoryIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 4 || ids[1] != 6 {
		t.Errorf("CategoryIDs = %v, want [4 6] (ascending)", ids)
	}
}

func TestCategoryIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.CategoryIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("CategoryIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("CategoryIDs = %v, want empty", ids)
	}
}

func TestPutCategory_ThenCategory_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Category{ID: 999, Name: map[string]string{"en": "New Category"}, Published: true, GroupIDs: []int64{111, 222}}

	if err := store.PutCategory(want); err != nil {
		t.Fatalf("PutCategory returned error: %v", err)
	}

	got, ok, err := store.Category(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false after PutCategory, want true")
	}
	if got.ID != want.ID || got.Name["en"] != want.Name["en"] || got.Published != want.Published {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.GroupIDs) != 2 || got.GroupIDs[0] != 111 || got.GroupIDs[1] != 222 {
		t.Errorf("GroupIDs = %v, want [111 222]", got.GroupIDs)
	}
}
