package sde

import (
	"context"
	"testing"
)

func TestRaces_ReturnsAll(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 1, "description": {"en": "Founded on the tenets..."}, "name": {"en": "Caldari"}}`)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Races = %+v, want one entry", got)
	}
	r := got[0]
	if r.ID != 1 {
		t.Errorf("got.ID = %d, want 1", r.ID)
	}
	if r.Name["en"] != "Caldari" {
		t.Errorf("Name[en] = %q, want Caldari", r.Name["en"])
	}
}

func TestRaces_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Races = %+v, want empty", got)
	}
}

func TestRaces_ReturnsAllianceID(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 1, "allianceId": 500001, "description": {"en": "..."}, "name": {"en": "Caldari"}}`)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Races = %+v, want one entry", got)
	}
	if got[0].AllianceID == nil || *got[0].AllianceID != 500001 {
		t.Errorf("AllianceID = %v, want 500001", got[0].AllianceID)
	}
}

func TestRaces_NoAllianceID_ReturnsNil(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 2, `{"_key": 2, "description": {"en": "..."}, "name": {"en": "No Alliance"}}`)

	got, err := store.Races(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 || got[0].AllianceID != nil {
		t.Errorf("AllianceID = %v, want nil", got[0].AllianceID)
	}
}

func TestRace_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 1, "description": {"en": "..."}, "name": {"en": "Caldari", "de": "Kaldari"}}`)

	got, ok, err := store.Race(context.Background(), "test-agent", 1)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if !ok {
		t.Fatal("Race ok = false, want true")
	}
	if got.ID != 1 {
		t.Errorf("got.ID = %d, want 1", got.ID)
	}
	if got.Name["en"] != "Caldari" || got.Name["de"] != "Kaldari" {
		t.Errorf("Name = %+v, want en=Caldari de=Kaldari", got.Name)
	}
}

func TestRace_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Race(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if ok {
		t.Error("Race ok = true, want false for an unknown ID")
	}
}

func TestRace_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `not valid json`)

	_, _, err := store.Race(context.Background(), "test-agent", 1)
	if err == nil {
		t.Fatal("Race returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestRace_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "races", 1, `{"_key": 999, "description": {"en": "..."}, "name": {"en": "Caldari"}}`)

	_, _, err := store.Race(context.Background(), "test-agent", 1)
	if err == nil {
		t.Fatal("Race returned nil error, want an error for a mismatched _key")
	}
}
