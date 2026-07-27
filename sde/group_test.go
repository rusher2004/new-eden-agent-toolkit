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
