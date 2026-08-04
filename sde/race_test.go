package sde

import (
	"context"
	"testing"
)

func TestRaces_MergesAllianceIDFromInMemoryAugmentation(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrRaces (raceID, raceName, description) VALUES (1, 'Caldari', 'The Caldari.')`)
	store.raceAlliances = map[int64]int64{1: 500001}

	got, err := store.Races(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Races) = %d, want 1", len(got))
	}
	r := got[0]
	if r.ID != 1 || r.Name != "Caldari" || r.Description != "The Caldari." {
		t.Errorf("got = %+v, want ID=1 Name=Caldari Description=%q", r, "The Caldari.")
	}
	if r.AllianceID == nil || *r.AllianceID != 500001 {
		t.Errorf("AllianceID = %v, want 500001", r.AllianceID)
	}
}

func TestRaces_NoAugmentation_ReturnsNilAllianceID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrRaces (raceID, raceName) VALUES (1, 'Caldari')`)

	got, err := store.Races(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Races returned error: %v", err)
	}
	if len(got) != 1 || got[0].AllianceID != nil {
		t.Errorf("Races = %+v, want one Race with nil AllianceID", got)
	}
}

func TestRace_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrRaces (raceID, raceName) VALUES (1, 'Caldari')`)

	got, ok, err := store.Race(context.Background(), testUserAgent, 1)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if !ok {
		t.Fatal("Race ok = false, want true")
	}
	if got.Name != "Caldari" {
		t.Errorf("Name = %q, want Caldari", got.Name)
	}
}

func TestRace_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Race(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Race returned error: %v", err)
	}
	if ok {
		t.Error("Race ok = true, want false for an unknown ID")
	}
}
