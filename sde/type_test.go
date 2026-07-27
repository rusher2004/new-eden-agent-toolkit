package sde

import (
	"context"
	"testing"
)

func TestType_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 587, "groupID": 25, "name": {"en": "Rifter", "de": "Rifter"}, "published": true}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.ID != 587 || got.GroupID != 25 || !got.Published {
		t.Errorf("got = %+v, want ID=587 GroupID=25 Published=true", got)
	}
	if got.Name["en"] != "Rifter" || got.Name["de"] != "Rifter" {
		t.Errorf("Name = %+v, want en=Rifter de=Rifter", got.Name)
	}
}

func TestType_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Type(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if ok {
		t.Error("Type ok = true, want false for an unknown ID")
	}
}

func TestType_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587, `not valid json`)

	_, _, err := store.Type(context.Background(), "test-agent", 587)
	if err == nil {
		t.Fatal("Type returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestType_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 12345, "groupID": 25, "name": {"en": "Rifter"}, "published": true}`)

	_, _, err := store.Type(context.Background(), "test-agent", 587)
	if err == nil {
		t.Fatal("Type returned nil error, want an error for a mismatched _key")
	}
}
