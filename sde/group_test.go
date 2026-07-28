package sde

import (
	"context"
	"testing"
)

func TestGroup_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "groups", 25,
		`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate", "de": "Fregatte"}, "published": true}`)

	got, ok, err := store.Group(context.Background(), "test-agent", 25)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false, want true")
	}
	if got.ID != 25 || got.CategoryID != 6 || !got.Published {
		t.Errorf("got = %+v, want ID=25 CategoryID=6 Published=true", got)
	}
	if got.Name["en"] != "Frigate" || got.Name["de"] != "Fregatte" {
		t.Errorf("Name = %+v, want en=Frigate de=Fregatte", got.Name)
	}
}

func TestGroup_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Group(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if ok {
		t.Error("Group ok = true, want false for an unknown ID")
	}
}

func TestGroup_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "groups", 25, `not valid json`)

	_, _, err := store.Group(context.Background(), "test-agent", 25)
	if err == nil {
		t.Fatal("Group returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestGroup_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "groups", 25,
		`{"_key": 999, "categoryID": 6, "name": {"en": "Frigate"}, "published": true}`)

	_, _, err := store.Group(context.Background(), "test-agent", 25)
	if err == nil {
		t.Fatal("Group returned nil error, want an error for a mismatched _key")
	}
}

func TestGroup_KnownID_ReturnsTypeIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "groups", 25,
		`{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true, "typeIds": [587, 588]}`)

	got, ok, err := store.Group(context.Background(), "test-agent", 25)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false, want true")
	}
	if len(got.TypeIDs) != 2 || got.TypeIDs[0] != 587 || got.TypeIDs[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588]", got.TypeIDs)
	}
}

func TestGroupIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "groups", 25, `{"_key": 25, "categoryID": 6, "name": {"en": "Frigate"}, "published": true}`)
	putRecord(t, store, "groups", 26, `{"_key": 26, "categoryID": 6, "name": {"en": "Cruiser"}, "published": true}`)

	ids, err := store.GroupIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GroupIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 25 || ids[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26] (ascending)", ids)
	}
}

func TestGroupIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.GroupIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GroupIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("GroupIDs = %v, want empty", ids)
	}
}

func TestPutGroup_ThenGroup_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Group{ID: 999, CategoryID: 6, Name: map[string]string{"en": "New Group"}, Published: true, TypeIDs: []int64{111, 222}}

	if err := store.PutGroup(want); err != nil {
		t.Fatalf("PutGroup returned error: %v", err)
	}

	got, ok, err := store.Group(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false after PutGroup, want true")
	}
	if got.ID != want.ID || got.CategoryID != want.CategoryID || got.Name["en"] != want.Name["en"] || got.Published != want.Published {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.TypeIDs) != 2 || got.TypeIDs[0] != 111 || got.TypeIDs[1] != 222 {
		t.Errorf("TypeIDs = %v, want [111 222]", got.TypeIDs)
	}
}
