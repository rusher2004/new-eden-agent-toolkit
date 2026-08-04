package sde

import (
	"context"
	"testing"
)

func TestBloodlines_ReturnsAll(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "bloodlines", 1,
		`{"_key": 1, "charisma": 6, "corporationID": 1000006, "description": {"en": "The Deteis are..."}, "iconID": 1633, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "shipTypeID": 601, "willpower": 5}`)

	got, err := store.Bloodlines(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Bloodlines returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Bloodlines = %+v, want one entry", got)
	}
	b := got[0]
	if b.ID != 1 || b.Charisma != 6 || b.CorporationID != 1000006 {
		t.Errorf("got = %+v, want ID=1 Charisma=6 CorporationID=1000006", b)
	}
	if b.Name["en"] != "Deteis" {
		t.Errorf("Name[en] = %q, want Deteis", b.Name["en"])
	}
	if b.Intelligence != 7 || b.Memory != 7 || b.Perception != 5 || b.Willpower != 5 || b.RaceID != 1 {
		t.Errorf("got = %+v, want Intelligence=7 Memory=7 Perception=5 Willpower=5 RaceID=1", b)
	}
	if b.ShipTypeID == nil || *b.ShipTypeID != 601 {
		t.Errorf("ShipTypeID = %v, want 601", b.ShipTypeID)
	}
}

func TestBloodlines_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Bloodlines(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Bloodlines returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Bloodlines = %+v, want empty", got)
	}
}

func TestBloodline_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "bloodlines", 1, `{"_key": 1, "charisma": 6, "corporationID": 1000006, "description": {"en": "..."}, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "shipTypeID": 601, "willpower": 5}`)

	got, ok, err := store.Bloodline(context.Background(), "test-agent", 1)
	if err != nil {
		t.Fatalf("Bloodline returned error: %v", err)
	}
	if !ok {
		t.Fatal("Bloodline ok = false, want true")
	}
	if got.ID != 1 {
		t.Errorf("got.ID = %d, want 1", got.ID)
	}
	if got.Name["en"] != "Deteis" {
		t.Errorf("Name[en] = %q, want Deteis", got.Name["en"])
	}
	if got.RaceID != 1 {
		t.Errorf("RaceID = %d, want 1", got.RaceID)
	}
}

func TestBloodline_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Bloodline(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Bloodline returned error: %v", err)
	}
	if ok {
		t.Error("Bloodline ok = true, want false for an unknown ID")
	}
}

func TestBloodline_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "bloodlines", 1, `not valid json`)

	_, _, err := store.Bloodline(context.Background(), "test-agent", 1)
	if err == nil {
		t.Fatal("Bloodline returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestBloodline_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "bloodlines", 1, `{"_key": 999, "charisma": 6, "corporationID": 1000006, "description": {"en": "..."}, "intelligence": 7, "memory": 7, "name": {"en": "Deteis"}, "perception": 5, "raceID": 1, "shipTypeID": 601, "willpower": 5}`)

	_, _, err := store.Bloodline(context.Background(), "test-agent", 1)
	if err == nil {
		t.Fatal("Bloodline returned nil error, want an error for a mismatched _key")
	}
}
